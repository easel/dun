package dun

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInstallCreatesAgentsFile(t *testing.T) {
	root := tempRepo(t)

	result, err := InstallRepo(root, InstallOptions{})
	if err != nil {
		t.Fatalf("install: %v", err)
	}
	if len(result.Steps) != 1 {
		t.Fatalf("expected 1 step, got %d", len(result.Steps))
	}
	if result.Steps[0].Action == "noop" {
		t.Fatalf("expected create action")
	}

	content := readFile(t, filepath.Join(root, "AGENTS.md"))
	if !strings.Contains(content, agentsMarkerStart) {
		t.Fatalf("expected marker start")
	}
	if !strings.Contains(content, agentsToolLine) {
		t.Fatalf("expected tool line")
	}
}

func TestInstallIsIdempotent(t *testing.T) {
	root := tempRepo(t)

	if _, err := InstallRepo(root, InstallOptions{}); err != nil {
		t.Fatalf("install: %v", err)
	}
	first := readFile(t, filepath.Join(root, "AGENTS.md"))

	if _, err := InstallRepo(root, InstallOptions{}); err != nil {
		t.Fatalf("install again: %v", err)
	}
	second := readFile(t, filepath.Join(root, "AGENTS.md"))

	if first != second {
		t.Fatalf("expected idempotent install")
	}
}

func TestInstallInsertsUnderToolsHeader(t *testing.T) {
	root := tempRepo(t)
	path := filepath.Join(root, "AGENTS.md")
	if err := os.WriteFile(path, []byte("## Tools\n- existing\n"), 0644); err != nil {
		t.Fatalf("write agents: %v", err)
	}

	if _, err := InstallRepo(root, InstallOptions{}); err != nil {
		t.Fatalf("install: %v", err)
	}
	content := readFile(t, path)
	if !strings.Contains(content, agentsMarkerStart) {
		t.Fatalf("expected marker start")
	}
	if !strings.Contains(content, "- existing") {
		t.Fatalf("expected existing tool line preserved")
	}
}

func TestInstallDryRunDoesNotWrite(t *testing.T) {
	root := tempRepo(t)

	if _, err := InstallRepo(root, InstallOptions{DryRun: true}); err != nil {
		t.Fatalf("install dry run: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "AGENTS.md")); err == nil {
		t.Fatalf("expected no AGENTS.md on dry run")
	}
}

func tempRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, ".git"), 0755); err != nil {
		t.Fatalf("create .git: %v", err)
	}
	return dir
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(content)
}
