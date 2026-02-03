package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestCheckDDXVersionMissingFile(t *testing.T) {
	root := t.TempDir()
	if warn := checkDDXVersion(root); warn != "" {
		t.Fatalf("expected no warning, got %q", warn)
	}
}

func TestCheckDDXVersionCacheMissing(t *testing.T) {
	root := t.TempDir()
	ddxWriteFile(t, filepath.Join(root, ".ddx-version"), "deadbeef\n")
	t.Setenv("XDG_CACHE_HOME", t.TempDir())

	warn := checkDDXVersion(root)
	if warn == "" || !strings.Contains(warn, "ddx library cache not found") {
		t.Fatalf("expected cache warning, got %q", warn)
	}
}

func TestCheckDDXVersionMatch(t *testing.T) {
	ddxEnsureGit(t)
	root := t.TempDir()

	cacheRoot := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", cacheRoot)
	libraryDir := filepath.Join(cacheRoot, "ddx", "library")
	commit := ddxInitGitRepo(t, libraryDir, "init")

	ddxWriteFile(t, filepath.Join(root, ".ddx-version"), commit+"\n")

	if warn := checkDDXVersion(root); warn != "" {
		t.Fatalf("expected no warning, got %q", warn)
	}
}

func TestCheckDDXVersionMismatch(t *testing.T) {
	ddxEnsureGit(t)
	root := t.TempDir()

	cacheRoot := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", cacheRoot)
	libraryDir := filepath.Join(cacheRoot, "ddx", "library")
	first := ddxInitGitRepo(t, libraryDir, "first")
	_ = ddxCommitFile(t, libraryDir, "second.txt", "second")

	ddxWriteFile(t, filepath.Join(root, ".ddx-version"), first+"\n")

	warn := checkDDXVersion(root)
	if warn == "" || !strings.Contains(warn, "resolves to") {
		t.Fatalf("expected mismatch warning, got %q", warn)
	}
}

func ddxEnsureGit(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
}

func ddxInitGitRepo(t *testing.T, dir, message string) string {
	t.Helper()
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	ddxRunGit(t, dir, "init")
	ddxRunGit(t, dir, "config", "user.email", "test@example.com")
	ddxRunGit(t, dir, "config", "user.name", "Test")
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("init\n"), 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	ddxRunGit(t, dir, "add", ".")
	ddxRunGit(t, dir, "commit", "-m", message)
	return strings.TrimSpace(ddxRunGitOutput(t, dir, "rev-parse", "HEAD"))
}

func ddxCommitFile(t *testing.T, dir, name, message string) string {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte("change\n"), 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	ddxRunGit(t, dir, "add", name)
	ddxRunGit(t, dir, "commit", "-m", message)
	return strings.TrimSpace(ddxRunGitOutput(t, dir, "rev-parse", "HEAD"))
}

func ddxRunGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %s failed: %v (%s)", strings.Join(args, " "), err, string(out))
	}
}

func ddxRunGitOutput(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("git %s failed: %v", strings.Join(args, " "), err)
	}
	return string(out)
}

func ddxWriteFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}
}
