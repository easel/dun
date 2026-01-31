package version

import (
	"runtime"
	"strings"
	"testing"
)

func TestGet(t *testing.T) {
	info := Get()

	if info.Version == "" {
		t.Error("Version should not be empty")
	}
	if info.Commit == "" {
		t.Error("Commit should not be empty")
	}
	if info.BuildDate == "" {
		t.Error("BuildDate should not be empty")
	}
	if info.GoVersion != runtime.Version() {
		t.Errorf("GoVersion = %q, want %q", info.GoVersion, runtime.Version())
	}
	expectedPlatform := runtime.GOOS + "/" + runtime.GOARCH
	if info.Platform != expectedPlatform {
		t.Errorf("Platform = %q, want %q", info.Platform, expectedPlatform)
	}
}

func TestGetDefaultValues(t *testing.T) {
	info := Get()

	if info.Version != "dev" {
		t.Errorf("default Version = %q, want %q", info.Version, "dev")
	}
	if info.Commit != "unknown" {
		t.Errorf("default Commit = %q, want %q", info.Commit, "unknown")
	}
	if info.BuildDate != "unknown" {
		t.Errorf("default BuildDate = %q, want %q", info.BuildDate, "unknown")
	}
}

func TestInfoString(t *testing.T) {
	info := Info{
		Version:   "1.0.0",
		Commit:    "abc123",
		BuildDate: "2024-01-15",
		GoVersion: "go1.21.0",
		Platform:  "linux/amd64",
	}

	s := info.String()

	if !strings.Contains(s, "1.0.0") {
		t.Errorf("String() should contain version, got %q", s)
	}
	if !strings.Contains(s, "abc123") {
		t.Errorf("String() should contain commit, got %q", s)
	}
	if !strings.Contains(s, "2024-01-15") {
		t.Errorf("String() should contain build date, got %q", s)
	}
	if !strings.Contains(s, "go1.21.0") {
		t.Errorf("String() should contain go version, got %q", s)
	}
	if !strings.Contains(s, "linux/amd64") {
		t.Errorf("String() should contain platform, got %q", s)
	}

	expected := "dun 1.0.0 (abc123) built 2024-01-15 with go1.21.0 for linux/amd64"
	if s != expected {
		t.Errorf("String() = %q, want %q", s, expected)
	}
}

func TestInfoStringWithDefaults(t *testing.T) {
	info := Get()
	s := info.String()

	if !strings.HasPrefix(s, "dun ") {
		t.Errorf("String() should start with 'dun ', got %q", s)
	}
	if !strings.Contains(s, info.Version) {
		t.Errorf("String() should contain version %q, got %q", info.Version, s)
	}
	if !strings.Contains(s, info.GoVersion) {
		t.Errorf("String() should contain go version %q, got %q", info.GoVersion, s)
	}
	if !strings.Contains(s, info.Platform) {
		t.Errorf("String() should contain platform %q, got %q", info.Platform, s)
	}
}
