package dun

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestGitStatusCheckWarnsWhenDirty(t *testing.T) {
	root := tempGitRepo(t)
	writeFile(t, filepath.Join(root, "notes.txt"), "hello")

	res, err := runGitStatusCheck(root, CheckDefinition{ID: "git-status"})
	if err != nil {
		t.Fatalf("git status check: %v", err)
	}
	if res.Status != "warn" {
		t.Fatalf("expected warn, got %s", res.Status)
	}
	if len(res.Issues) == 0 {
		t.Fatalf("expected issues for dirty files")
	}
	if !strings.Contains(res.Next, "git commit") {
		t.Fatalf("expected commit instruction, got %q", res.Next)
	}
}

func TestGitStatusCheckPassesWhenClean(t *testing.T) {
	root := tempGitRepo(t)

	res, err := runGitStatusCheck(root, CheckDefinition{ID: "git-status"})
	if err != nil {
		t.Fatalf("git status check: %v", err)
	}
	if res.Status != "pass" {
		t.Fatalf("expected pass, got %s", res.Status)
	}
}

func TestHookCheckWarnsWhenToolMissing(t *testing.T) {
	root := tempGitRepo(t)
	writeFile(t, filepath.Join(root, "lefthook.yml"), "pre-commit: {}")
	t.Setenv("PATH", "")

	res, err := runHookCheck(root, CheckDefinition{ID: "git-hooks"})
	if err != nil {
		t.Fatalf("hook check: %v", err)
	}
	if res.Status != "warn" {
		t.Fatalf("expected warn, got %s", res.Status)
	}
	if !strings.Contains(res.Detail, "lefthook") {
		t.Fatalf("expected lefthook detail, got %q", res.Detail)
	}
}

func TestHookCheckRunsWhenToolPresent(t *testing.T) {
	root := tempGitRepo(t)
	writeFile(t, filepath.Join(root, "lefthook.yml"), "pre-commit: {}")

	binDir := t.TempDir()
	toolPath := filepath.Join(binDir, "lefthook")
	script := "#!/bin/sh\nexit 0\n"
	writeFile(t, toolPath, script)
	if err := os.Chmod(toolPath, 0755); err != nil {
		t.Fatalf("chmod lefthook: %v", err)
	}

	origPath := os.Getenv("PATH")
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+origPath)

	res, err := runHookCheck(root, CheckDefinition{ID: "git-hooks"})
	if err != nil {
		t.Fatalf("hook check: %v", err)
	}
	if res.Status != "pass" {
		t.Fatalf("expected pass, got %s", res.Status)
	}
}

func tempGitRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	cmd := exec.Command("git", "init")
	cmd.Dir = dir
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init: %v (%s)", err, string(output))
	}
	return dir
}

func writeFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
