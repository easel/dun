package dun

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDetectHookToolNone(t *testing.T) {
	root := tempGitRepo(t)
	tool, err := detectHookTool(root)
	if err != nil {
		t.Fatalf("detect hook tool: %v", err)
	}
	if tool.Name != "" {
		t.Fatalf("expected no tool, got %q", tool.Name)
	}
}

func TestDetectHookToolPreCommit(t *testing.T) {
	root := tempGitRepo(t)
	writeFile(t, root+"/.pre-commit-config.yaml", "repos: []")
	t.Setenv("PATH", "")
	tool, err := detectHookTool(root)
	if err != nil {
		t.Fatalf("detect hook tool: %v", err)
	}
	if tool.Name != "pre-commit" {
		t.Fatalf("expected pre-commit, got %q", tool.Name)
	}
}

func TestHookCheckFailsWhenToolErrors(t *testing.T) {
	root := tempGitRepo(t)
	writeFile(t, root+"/lefthook.yml", "pre-commit: {}")

	binDir := t.TempDir()
	toolPath := filepath.Join(binDir, "lefthook")
	writeFile(t, toolPath, "#!/bin/sh\nexit 1\n")
	if err := os.Chmod(toolPath, 0755); err != nil {
		t.Fatalf("chmod lefthook: %v", err)
	}
	origPath := os.Getenv("PATH")
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+origPath)

	res, err := runHookCheck(root, Check{ID: "git-hooks"})
	if err != nil {
		t.Fatalf("hook check: %v", err)
	}
	if res.Status != "fail" {
		t.Fatalf("expected fail, got %s", res.Status)
	}
}

func TestParseGitStatusPath(t *testing.T) {
	if got := parseGitStatusPath("R  old -> new"); got != "new" {
		t.Fatalf("expected rename to new, got %q", got)
	}
	if got := parseGitStatusPath("??"); got != "??" {
		t.Fatalf("expected raw path for short line, got %q", got)
	}
}

func TestCommitNextInstructionNoFiles(t *testing.T) {
	msg := commitNextInstruction(nil)
	if !strings.Contains(msg, "git commit") {
		t.Fatalf("expected commit instruction, got %q", msg)
	}
}

func TestTrimOutput(t *testing.T) {
	if got := trimOutput([]byte("")); got == "" {
		t.Fatalf("expected default message")
	}
	lines := make([]string, 20)
	for i := range lines {
		lines[i] = "line"
	}
	out := trimOutput([]byte(strings.Join(lines, "\n")))
	if len(strings.Split(out, "\n")) != 12 {
		t.Fatalf("expected trimmed output")
	}
}

func TestGitStatusLinesError(t *testing.T) {
	_, err := gitStatusLines(t.TempDir())
	if err == nil {
		t.Fatalf("expected git status error")
	}
}

func TestRunGitStatusCheckSkipsEmptyAndDuplicate(t *testing.T) {
	orig := gitStatusFunc
	gitStatusFunc = func(_ string) ([]string, error) {
		return []string{"?? ", "?? file.txt", "?? file.txt"}, nil
	}
	t.Cleanup(func() { gitStatusFunc = orig })

	res, err := runGitStatusCheck(t.TempDir(), Check{ID: "git-status"})
	if err != nil {
		t.Fatalf("git status: %v", err)
	}
	if len(res.Issues) != 1 {
		t.Fatalf("expected single issue, got %v", res.Issues)
	}
}

func TestRunGitStatusCheckError(t *testing.T) {
	orig := gitStatusFunc
	gitStatusFunc = func(_ string) ([]string, error) {
		return nil, os.ErrInvalid
	}
	t.Cleanup(func() { gitStatusFunc = orig })

	if _, err := runGitStatusCheck(t.TempDir(), Check{ID: "git-status"}); err == nil {
		t.Fatalf("expected git status error")
	}
}

func TestRunHookCheckDetectError(t *testing.T) {
	orig := detectHookToolFunc
	detectHookToolFunc = func(string) (hookTool, error) {
		return hookTool{}, os.ErrInvalid
	}
	t.Cleanup(func() { detectHookToolFunc = orig })

	if _, err := runHookCheck(t.TempDir(), Check{ID: "git-hooks"}); err == nil {
		t.Fatalf("expected detect error")
	}
}

func TestRunHookCheckSkip(t *testing.T) {
	root := tempGitRepo(t)
	res, err := runHookCheck(root, Check{ID: "git-hooks"})
	if err != nil {
		t.Fatalf("hook check: %v", err)
	}
	if res.Status != "skip" {
		t.Fatalf("expected skip, got %s", res.Status)
	}
}

func TestRunHookCheckWarnWhenToolMissing(t *testing.T) {
	root := tempGitRepo(t)
	writeFile(t, root+"/lefthook.yml", "pre-commit: {}")
	t.Setenv("PATH", "")

	res, err := runHookCheck(root, Check{ID: "git-hooks"})
	if err != nil {
		t.Fatalf("hook check: %v", err)
	}
	if res.Status != "warn" {
		t.Fatalf("expected warn, got %s", res.Status)
	}
}

func TestRunHookCheckPasses(t *testing.T) {
	root := tempGitRepo(t)
	writeFile(t, root+"/lefthook.yml", "pre-commit: {}")

	binDir := t.TempDir()
	toolPath := filepath.Join(binDir, "lefthook")
	writeFile(t, toolPath, "#!/bin/sh\nexit 0\n")
	if err := os.Chmod(toolPath, 0755); err != nil {
		t.Fatalf("chmod lefthook: %v", err)
	}
	origPath := os.Getenv("PATH")
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+origPath)

	res, err := runHookCheck(root, Check{ID: "git-hooks"})
	if err != nil {
		t.Fatalf("hook check: %v", err)
	}
	if res.Status != "pass" {
		t.Fatalf("expected pass, got %s", res.Status)
	}
}
