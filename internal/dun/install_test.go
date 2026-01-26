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
	if len(result.Steps) != 2 {
		t.Fatalf("expected 2 steps, got %d", len(result.Steps))
	}
	if step := findStep(result, "config"); step == nil || step.Action == "noop" {
		t.Fatalf("expected config create action")
	}
	if step := findStep(result, "agents"); step == nil || step.Action == "noop" {
		t.Fatalf("expected agents create action")
	}

	content := readFile(t, filepath.Join(root, "AGENTS.md"))
	if !strings.Contains(content, agentsMarkerStart) {
		t.Fatalf("expected marker start")
	}
	if !strings.Contains(content, agentsToolLine) {
		t.Fatalf("expected tool line")
	}

	config := readFile(t, filepath.Join(root, DefaultConfigPath))
	if !strings.Contains(config, "automation: auto") {
		t.Fatalf("expected automation default in config")
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
	if _, err := os.Stat(filepath.Join(root, DefaultConfigPath)); err == nil {
		t.Fatalf("expected no config on dry run")
	}
}

func TestInstallRepoConfigError(t *testing.T) {
	root := tempRepo(t)
	dunPath := filepath.Join(root, ".dun")
	if err := os.WriteFile(dunPath, []byte("not a dir"), 0644); err != nil {
		t.Fatalf("write .dun: %v", err)
	}
	if _, err := InstallRepo(root, InstallOptions{}); err == nil {
		t.Fatalf("expected install error on config")
	}
}

func TestInstallRepoAgentsFileError(t *testing.T) {
	root := tempRepo(t)
	if err := os.MkdirAll(filepath.Join(root, ".dun"), 0755); err != nil {
		t.Fatalf("mkdir .dun: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, DefaultConfigPath), []byte("version: \"1\""), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	agentsPath := filepath.Join(root, "AGENTS.md")
	if err := os.MkdirAll(agentsPath, 0755); err != nil {
		t.Fatalf("mkdir agents: %v", err)
	}
	if _, err := InstallRepo(root, InstallOptions{}); err == nil {
		t.Fatalf("expected install error on agents")
	}
}

func TestFindRepoRootError(t *testing.T) {
	_, err := FindRepoRoot(t.TempDir())
	if err == nil {
		t.Fatalf("expected error for missing .git")
	}
}

func TestFindRepoRootAbsError(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	dir := t.TempDir()
	sub := filepath.Join(dir, "gone")
	if err := os.MkdirAll(sub, 0755); err != nil {
		t.Fatalf("mkdir sub: %v", err)
	}
	if err := os.Chdir(sub); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	if err := os.RemoveAll(sub); err != nil {
		t.Fatalf("remove sub: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(cwd)
	})

	if _, err := FindRepoRoot("."); err == nil {
		t.Fatalf("expected abs error")
	}
}

func TestEnsureConfigFileNoop(t *testing.T) {
	root := tempRepo(t)
	path := filepath.Join(root, DefaultConfigPath)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("mkdir config: %v", err)
	}
	if err := os.WriteFile(path, []byte("version: \"1\""), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	action, err := ensureConfigFile(path, false)
	if err != nil {
		t.Fatalf("ensure config: %v", err)
	}
	if action != "noop" {
		t.Fatalf("expected noop, got %s", action)
	}
}

func TestEnsureConfigFileStatError(t *testing.T) {
	root := tempRepo(t)
	dunPath := filepath.Join(root, ".dun")
	if err := os.WriteFile(dunPath, []byte("not a dir"), 0644); err != nil {
		t.Fatalf("write .dun: %v", err)
	}
	path := filepath.Join(dunPath, "config.yaml")
	if _, err := ensureConfigFile(path, false); err == nil {
		t.Fatalf("expected stat error")
	}
}

func TestEnsureConfigFileMkdirError(t *testing.T) {
	root := tempRepo(t)
	if err := os.Chmod(root, 0555); err != nil {
		t.Fatalf("chmod root: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(root, 0755) })
	path := filepath.Join(root, ".dun", "config.yaml")
	if _, err := ensureConfigFile(path, false); err == nil {
		t.Fatalf("expected mkdir error")
	}
}

func TestReplaceMarkerBlockError(t *testing.T) {
	_, err := replaceMarkerBlock("<!-- DUN:END -->", "x")
	if err == nil {
		t.Fatalf("expected marker error")
	}
}

func TestInstallRepoErrorWhenNoGit(t *testing.T) {
	if _, err := InstallRepo(t.TempDir(), InstallOptions{}); err == nil {
		t.Fatalf("expected install error without repo")
	}
}

func TestUpsertAgentsFileReadError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "AGENTS.md")
	if err := os.MkdirAll(path, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if _, err := upsertAgentsFile(path, false); err == nil {
		t.Fatalf("expected read error for directory path")
	}
}

func TestUpsertAgentsFileContentError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "AGENTS.md")
	if err := os.WriteFile(path, []byte("<!-- DUN:END -->\n<!-- DUN:BEGIN -->\n"), 0644); err != nil {
		t.Fatalf("write agents: %v", err)
	}
	if _, err := upsertAgentsFile(path, false); err == nil {
		t.Fatalf("expected content error")
	}
}

func TestUpsertAgentsFileWriteError(t *testing.T) {
	dir := t.TempDir()
	if err := os.Chmod(dir, 0555); err != nil {
		t.Fatalf("chmod dir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0755) })
	path := filepath.Join(dir, "AGENTS.md")
	if _, err := upsertAgentsFile(path, false); err == nil {
		t.Fatalf("expected write error")
	}
}

func TestUpsertAgentsContentWithMarkers(t *testing.T) {
	content := "## Tools\n<!-- DUN:BEGIN -->\nold\n<!-- DUN:END -->\n"
	updated, changed, action, err := upsertAgentsContent(content)
	if err != nil {
		t.Fatalf("upsert markers: %v", err)
	}
	if !changed || action != "update" {
		t.Fatalf("expected update action")
	}
	if !strings.Contains(updated, agentsToolLine) {
		t.Fatalf("expected tool line")
	}
}

func TestUpsertAgentsContentMalformedMarkers(t *testing.T) {
	_, _, _, err := upsertAgentsContent("<!-- DUN:END -->\n<!-- DUN:BEGIN -->")
	if err == nil {
		t.Fatalf("expected malformed markers error")
	}
}

func TestUpsertAgentsContentWithPreface(t *testing.T) {
	content := "Intro text\n"
	updated, changed, action, err := upsertAgentsContent(content)
	if err != nil {
		t.Fatalf("upsert preface: %v", err)
	}
	if !changed || action != "create" {
		t.Fatalf("expected create action")
	}
	if !strings.Contains(updated, "Intro text\n\n## Tools") {
		t.Fatalf("expected blank line before tools")
	}
}

func TestInsertAfterToolsNoHeader(t *testing.T) {
	content := "no tools here"
	out := insertAfterTools(content, []string{"line"})
	if out != content {
		t.Fatalf("expected unchanged content")
	}
}

func TestEnsureConfigFileWriteError(t *testing.T) {
	root := tempRepo(t)
	dir := filepath.Join(root, ".dun")
	if err := os.MkdirAll(dir, 0555); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	path := filepath.Join(dir, "config.yaml")
	action, err := ensureConfigFile(path, false)
	if err == nil {
		t.Fatalf("expected write error, got action %s", action)
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

func findStep(result InstallResult, stepType string) *InstallStep {
	for i := range result.Steps {
		if result.Steps[i].Type == stepType {
			return &result.Steps[i]
		}
	}
	return nil
}
