// Package update provides self-update functionality for the dun binary.
package update

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// Updater handles checking for and applying updates.
type Updater struct {
	CurrentVersion string
	RepoOwner      string
	RepoName       string
	BinaryName     string

	// HTTPClient allows injection for testing.
	HTTPClient HTTPClient
}

// HTTPClient interface for HTTP operations.
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// Release represents a GitHub release.
type Release struct {
	TagName     string    `json:"tag_name"`
	PublishedAt time.Time `json:"published_at"`
	Assets      []Asset   `json:"assets"`
}

// Asset represents a release asset.
type Asset struct {
	Name        string `json:"name"`
	DownloadURL string `json:"browser_download_url"`
	Size        int64  `json:"size"`
}

// DefaultUpdater returns an Updater with default settings.
func DefaultUpdater(currentVersion string) *Updater {
	return &Updater{
		CurrentVersion: currentVersion,
		RepoOwner:      "easel",
		RepoName:       "dun",
		BinaryName:     "dun",
		HTTPClient:     http.DefaultClient,
	}
}

// CheckForUpdate queries GitHub for the latest release and returns it
// along with a boolean indicating if an update is available.
func (u *Updater) CheckForUpdate() (*Release, bool, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest",
		u.RepoOwner, u.RepoName)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, false, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", fmt.Sprintf("%s-updater", u.BinaryName))

	client := u.httpClient()
	resp, err := client.Do(req)
	if err != nil {
		return nil, false, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, false, nil
	}
	if resp.StatusCode != http.StatusOK {
		return nil, false, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	var release Release
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, false, fmt.Errorf("decode response: %w", err)
	}

	hasUpdate := isNewerVersion(u.CurrentVersion, release.TagName)
	return &release, hasUpdate, nil
}

// DownloadRelease downloads the appropriate binary for the current platform
// and returns the path to the temporary file.
func (u *Updater) DownloadRelease(release *Release) (string, error) {
	asset := u.findAsset(release)
	if asset == nil {
		return "", fmt.Errorf("no asset found for %s/%s", runtime.GOOS, runtime.GOARCH)
	}

	req, err := http.NewRequest(http.MethodGet, asset.DownloadURL, nil)
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("User-Agent", fmt.Sprintf("%s-updater", u.BinaryName))

	client := u.httpClient()
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("download failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download status: %d", resp.StatusCode)
	}

	tmpFile, err := os.CreateTemp("", fmt.Sprintf("%s-update-*", u.BinaryName))
	if err != nil {
		return "", fmt.Errorf("create temp file: %w", err)
	}
	defer tmpFile.Close()

	hasher := sha256.New()
	writer := io.MultiWriter(tmpFile, hasher)

	written, err := io.Copy(writer, resp.Body)
	if err != nil {
		os.Remove(tmpFile.Name())
		return "", fmt.Errorf("write temp file: %w", err)
	}

	if asset.Size > 0 && written != asset.Size {
		os.Remove(tmpFile.Name())
		return "", fmt.Errorf("size mismatch: expected %d, got %d", asset.Size, written)
	}

	return tmpFile.Name(), nil
}

// ApplyUpdate atomically replaces the current binary with the downloaded one.
func (u *Updater) ApplyUpdate(downloadPath string) error {
	currentPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("get executable path: %w", err)
	}
	currentPath, err = filepath.EvalSymlinks(currentPath)
	if err != nil {
		return fmt.Errorf("eval symlinks: %w", err)
	}

	return applyUpdateToPath(downloadPath, currentPath)
}

// ApplyUpdateToPath is like ApplyUpdate but allows specifying the target path.
// This is useful for testing.
func (u *Updater) ApplyUpdateToPath(downloadPath, currentPath string) error {
	return applyUpdateToPath(downloadPath, currentPath)
}

func applyUpdateToPath(downloadPath, currentPath string) error {
	// Verify the downloaded file exists and is readable
	info, err := os.Stat(downloadPath)
	if err != nil {
		return fmt.Errorf("stat download: %w", err)
	}
	if info.Size() == 0 {
		return errors.New("downloaded file is empty")
	}

	// Create backup
	backupPath := currentPath + ".old"
	if err := os.Rename(currentPath, backupPath); err != nil {
		return fmt.Errorf("backup current binary: %w", err)
	}

	// Move new binary into place
	if err := copyFile(downloadPath, currentPath); err != nil {
		// Attempt restore on failure
		_ = os.Rename(backupPath, currentPath)
		return fmt.Errorf("install new binary: %w", err)
	}

	// Make executable
	if err := os.Chmod(currentPath, 0755); err != nil {
		// Attempt restore on failure
		_ = os.Remove(currentPath)
		_ = os.Rename(backupPath, currentPath)
		return fmt.Errorf("chmod new binary: %w", err)
	}

	// Verify the new binary
	if err := verifyBinary(currentPath); err != nil {
		// Attempt restore on failure
		_ = os.Remove(currentPath)
		_ = os.Rename(backupPath, currentPath)
		return fmt.Errorf("verify new binary: %w", err)
	}

	// Clean up temp file
	os.Remove(downloadPath)

	return nil
}

// Rollback restores the previous binary from the .old backup.
func (u *Updater) Rollback() error {
	currentPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("get executable path: %w", err)
	}
	currentPath, err = filepath.EvalSymlinks(currentPath)
	if err != nil {
		return fmt.Errorf("eval symlinks: %w", err)
	}

	return rollbackPath(currentPath)
}

// RollbackPath is like Rollback but allows specifying the target path.
// This is useful for testing.
func (u *Updater) RollbackPath(currentPath string) error {
	return rollbackPath(currentPath)
}

func rollbackPath(currentPath string) error {
	backupPath := currentPath + ".old"
	if _, err := os.Stat(backupPath); err != nil {
		if os.IsNotExist(err) {
			return errors.New("no backup found")
		}
		return fmt.Errorf("stat backup: %w", err)
	}

	if err := os.Remove(currentPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove current binary: %w", err)
	}

	if err := os.Rename(backupPath, currentPath); err != nil {
		return fmt.Errorf("restore backup: %w", err)
	}

	return nil
}

// findAsset locates the appropriate asset for the current platform.
func (u *Updater) findAsset(release *Release) *Asset {
	osName := runtime.GOOS
	archName := runtime.GOARCH

	// Normalize arch names
	archAliases := map[string][]string{
		"amd64": {"amd64", "x86_64", "x64"},
		"arm64": {"arm64", "aarch64"},
		"386":   {"386", "i386", "i686", "x86"},
	}

	aliases := archAliases[archName]
	if aliases == nil {
		aliases = []string{archName}
	}

	for i := range release.Assets {
		asset := &release.Assets[i]
		name := strings.ToLower(asset.Name)

		// Skip checksums and signatures
		if strings.HasSuffix(name, ".sha256") ||
			strings.HasSuffix(name, ".sig") ||
			strings.HasSuffix(name, ".asc") {
			continue
		}

		if !strings.Contains(name, osName) {
			continue
		}

		for _, arch := range aliases {
			if strings.Contains(name, arch) {
				return asset
			}
		}
	}

	return nil
}

func (u *Updater) httpClient() HTTPClient {
	if u.HTTPClient != nil {
		return u.HTTPClient
	}
	return http.DefaultClient
}

// isNewerVersion compares version strings (without 'v' prefix).
func isNewerVersion(current, latest string) bool {
	current = strings.TrimPrefix(current, "v")
	latest = strings.TrimPrefix(latest, "v")

	// Handle dev version
	if current == "dev" || current == "" {
		return latest != "" && latest != "dev"
	}

	return latest != current && latest > current
}

// copyFile copies a file from src to dst.
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}

	return out.Close()
}

// verifyBinary performs basic verification of the new binary.
func verifyBinary(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if info.Size() == 0 {
		return errors.New("binary is empty")
	}

	// Read first bytes to verify it's an executable
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	header := make([]byte, 4)
	if _, err := io.ReadFull(f, header); err != nil {
		return fmt.Errorf("read header: %w", err)
	}

	// Check for ELF (Linux), Mach-O (macOS), or PE (Windows)
	switch {
	case header[0] == 0x7f && header[1] == 'E' && header[2] == 'L' && header[3] == 'F':
		// ELF
	case header[0] == 0xfe && header[1] == 0xed && header[2] == 0xfa && header[3] == 0xce:
		// Mach-O 32-bit
	case header[0] == 0xfe && header[1] == 0xed && header[2] == 0xfa && header[3] == 0xcf:
		// Mach-O 64-bit
	case header[0] == 0xce && header[1] == 0xfa && header[2] == 0xed && header[3] == 0xfe:
		// Mach-O 32-bit (reverse byte order)
	case header[0] == 0xcf && header[1] == 0xfa && header[2] == 0xed && header[3] == 0xfe:
		// Mach-O 64-bit (reverse byte order)
	case header[0] == 'M' && header[1] == 'Z':
		// PE (Windows)
	default:
		return fmt.Errorf("unrecognized executable format: %x", header)
	}

	return nil
}

// ComputeChecksum computes SHA256 checksum of a file.
func ComputeChecksum(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}
