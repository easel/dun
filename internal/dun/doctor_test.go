package dun

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRunDoctorWritesCache(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/test"), 0644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}

	origLiveness := harnessLivenessFn
	harnessLivenessFn = func(_ string) (bool, string, string) {
		return true, "test-model", ""
	}
	t.Cleanup(func() { harnessLivenessFn = origLiveness })

	report, err := RunDoctor(root)
	if err != nil {
		t.Fatalf("run doctor: %v", err)
	}
	if report.CachePath == "" {
		t.Fatalf("expected cache path in report")
	}
	if _, err := os.Stat(report.CachePath); err != nil {
		t.Fatalf("expected cache file to exist: %v", err)
	}
	if len(report.Harnesses) == 0 {
		t.Fatalf("expected harnesses in report")
	}

	foundGo := false
	for _, helper := range report.Helpers {
		if helper.Category == "go" && helper.Name == "go" {
			foundGo = true
			break
		}
	}
	if !foundGo {
		t.Fatalf("expected go helper in report")
	}
}
