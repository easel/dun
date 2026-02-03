package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func checkDDXVersion(root string) string {
	versionPath := filepath.Join(root, ".ddx-version")
	raw, err := os.ReadFile(versionPath)
	if err != nil {
		if os.IsNotExist(err) {
			return ""
		}
		return fmt.Sprintf("warning: failed to read .ddx-version: %v", err)
	}

	expected := firstVersionLine(string(raw))
	if expected == "" {
		return "warning: .ddx-version is empty"
	}

	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return fmt.Sprintf("warning: unable to locate cache directory: %v", err)
	}

	libraryDir := filepath.Join(cacheDir, "ddx", "library")
	if _, err := os.Stat(libraryDir); err != nil {
		if os.IsNotExist(err) {
			return "warning: .ddx-version set but ddx library cache not found; run `ddx update`"
		}
		return fmt.Sprintf("warning: unable to access ddx library cache: %v", err)
	}

	head, err := gitRevParse(libraryDir, "HEAD")
	if err != nil {
		return fmt.Sprintf("warning: unable to determine cached ddx library version: %v", err)
	}

	want, err := gitRevParse(libraryDir, expected)
	if err != nil {
		return fmt.Sprintf("warning: .ddx-version %q not found in cached ddx library; run `ddx update`", expected)
	}

	if head != want {
		return fmt.Sprintf("warning: .ddx-version %q resolves to %s but cached library is %s; run `ddx update`",
			expected, shortSHA(want), shortSHA(head))
	}

	return ""
}

func firstVersionLine(raw string) string {
	for _, line := range strings.Split(raw, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		return trimmed
	}
	return ""
}

func gitRevParse(dir, ref string) (string, error) {
	cmd := exec.Command("git", "-C", dir, "rev-parse", ref)
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git rev-parse %s: %w", ref, err)
	}
	return strings.TrimSpace(string(out)), nil
}

func shortSHA(value string) string {
	if len(value) > 12 {
		return value[:12]
	}
	return value
}
