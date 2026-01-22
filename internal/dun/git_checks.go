package dun

import (
	"bytes"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

func runGitStatusCheck(root string, check Check) (CheckResult, error) {
	lines, err := gitStatusLines(root)
	if err != nil {
		return CheckResult{}, err
	}
	if len(lines) == 0 {
		return CheckResult{
			ID:     check.ID,
			Status: "pass",
			Signal: "working tree clean",
		}, nil
	}

	var issues []Issue
	var files []string
	seen := map[string]bool{}
	for _, line := range lines {
		path := parseGitStatusPath(line)
		if path == "" || seen[path] {
			continue
		}
		seen[path] = true
		files = append(files, path)
		issues = append(issues, Issue{
			ID:      "git:" + path,
			Summary: "Include in commit: " + path,
			Path:    path,
		})
	}

	return CheckResult{
		ID:     check.ID,
		Status: "warn",
		Signal: "working tree has uncommitted changes",
		Detail: fmt.Sprintf("%d paths pending commit", len(files)),
		Next:   commitNextInstruction(files),
		Issues: issues,
	}, nil
}

func runHookCheck(root string, check Check) (CheckResult, error) {
	hook, err := detectHookTool(root)
	if err != nil {
		return CheckResult{}, err
	}
	if hook.Name == "" {
		return CheckResult{
			ID:     check.ID,
			Status: "skip",
			Signal: "no hook configuration detected",
		}, nil
	}
	if !hook.Installed {
		return CheckResult{
			ID:     check.ID,
			Status: "warn",
			Signal: "hook tool missing",
			Detail: fmt.Sprintf("%s config detected but tool not installed", hook.Name),
			Next:   hook.InstallHint,
		}, nil
	}

	cmd := exec.Command(hook.Command[0], hook.Command[1:]...)
	cmd.Dir = root
	output, err := cmd.CombinedOutput()
	if err != nil {
		return CheckResult{
			ID:     check.ID,
			Status: "fail",
			Signal: "hook checks failed",
			Detail: trimOutput(output),
			Next:   strings.Join(hook.Command, " "),
		}, nil
	}

	return CheckResult{
		ID:     check.ID,
		Status: "pass",
		Signal: fmt.Sprintf("%s hooks passed", hook.Name),
	}, nil
}

type hookTool struct {
	Name        string
	Command     []string
	Installed   bool
	InstallHint string
}

func detectHookTool(root string) (hookTool, error) {
	lefthookConfigured := exists(filepath.Join(root, "lefthook.yml")) ||
		exists(filepath.Join(root, ".lefthook"))
	if lefthookConfigured {
		_, err := exec.LookPath("lefthook")
		return hookTool{
			Name:        "lefthook",
			Command:     []string{"lefthook", "run", "pre-commit"},
			Installed:   err == nil,
			InstallHint: "Install lefthook (https://github.com/evilmartians/lefthook)",
		}, nil
	}

	preCommitConfigured := exists(filepath.Join(root, ".pre-commit-config.yaml"))
	if preCommitConfigured {
		_, err := exec.LookPath("pre-commit")
		return hookTool{
			Name:        "pre-commit",
			Command:     []string{"pre-commit", "run", "--all-files"},
			Installed:   err == nil,
			InstallHint: "Install pre-commit (https://pre-commit.com)",
		}, nil
	}

	return hookTool{}, nil
}

func gitStatusLines(root string) ([]string, error) {
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = root
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git status: %w", err)
	}
	out := strings.TrimSpace(string(output))
	if out == "" {
		return nil, nil
	}
	return strings.Split(out, "\n"), nil
}

func parseGitStatusPath(line string) string {
	if len(line) < 3 {
		return strings.TrimSpace(line)
	}
	path := strings.TrimSpace(line[3:])
	if path == "" {
		return ""
	}
	if strings.Contains(path, " -> ") {
		parts := strings.Split(path, " -> ")
		return strings.TrimSpace(parts[len(parts)-1])
	}
	return path
}

func commitNextInstruction(files []string) string {
	if len(files) == 0 {
		return "Create a commit message describing the changes, then run `git add -A && git commit -m \"<message>\"`."
	}
	list := strings.Join(files, ", ")
	return fmt.Sprintf("Create a commit message describing changes in: %s. Then run `git add -A && git commit -m \"<message>\"`.", list)
}

func trimOutput(output []byte) string {
	text := strings.TrimSpace(string(bytes.TrimSpace(output)))
	if text == "" {
		return "hook command failed"
	}
	lines := strings.Split(text, "\n")
	if len(lines) > 12 {
		lines = lines[:12]
	}
	return strings.Join(lines, "\n")
}
