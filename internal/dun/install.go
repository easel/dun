package dun

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type InstallOptions struct {
	DryRun bool
}

type InstallStep struct {
	Type   string `json:"type"`
	Path   string `json:"path"`
	Action string `json:"action"`
}

type InstallResult struct {
	Steps []InstallStep `json:"steps"`
}

const (
	agentsMarkerStart = "<!-- DUN:BEGIN -->"
	agentsMarkerEnd   = "<!-- DUN:END -->"
	agentsToolLine    = "- dun: run `dun check` before summarizing results"
)

func InstallRepo(start string, opts InstallOptions) (InstallResult, error) {
	root, err := findRepoRoot(start)
	if err != nil {
		return InstallResult{}, err
	}

	agentsPath := filepath.Join(root, "AGENTS.md")
	action, err := upsertAgentsFile(agentsPath, opts.DryRun)
	if err != nil {
		return InstallResult{}, err
	}

	return InstallResult{
		Steps: []InstallStep{
			{
				Type:   "agents",
				Path:   agentsPath,
				Action: action,
			},
		},
	}, nil
}

func findRepoRoot(start string) (string, error) {
	dir, err := filepath.Abs(start)
	if err != nil {
		return "", err
	}
	for {
		if exists(filepath.Join(dir, ".git")) {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", errors.New("repo root not found (missing .git)")
}

func upsertAgentsFile(path string, dryRun bool) (string, error) {
	existing, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return "", err
	}

	content := string(existing)
	updated, changed, action, err := upsertAgentsContent(content)
	if err != nil {
		return "", err
	}
	if !changed {
		return "noop", nil
	}
	if dryRun {
		return action, nil
	}

	if err := os.WriteFile(path, []byte(updated), 0644); err != nil {
		return "", err
	}
	return action, nil
}

func upsertAgentsContent(content string) (string, bool, string, error) {
	snippetLines := []string{agentsMarkerStart, agentsToolLine, agentsMarkerEnd}
	snippet := strings.Join(snippetLines, "\n")

	if strings.Contains(content, agentsMarkerStart) && strings.Contains(content, agentsMarkerEnd) {
		updated, err := replaceMarkerBlock(content, snippet)
		if err != nil {
			return "", false, "", err
		}
		if updated == content {
			return content, false, "noop", nil
		}
		return updated, true, "update", nil
	}

	if hasToolsHeader(content) {
		updated := insertAfterTools(content, snippetLines)
		if updated == content {
			return content, false, "noop", nil
		}
		return updated, true, "update", nil
	}

	updated := strings.TrimRight(content, "\n")
	if updated != "" {
		updated += "\n\n"
	}
	updated += "## Tools\n" + snippet + "\n"
	return updated, true, "create", nil
}

func replaceMarkerBlock(content, snippet string) (string, error) {
	start := strings.Index(content, agentsMarkerStart)
	end := strings.Index(content, agentsMarkerEnd)
	if start == -1 || end == -1 || end < start {
		return "", fmt.Errorf("agent markers malformed")
	}
	end += len(agentsMarkerEnd)
	return content[:start] + snippet + content[end:], nil
}

func hasToolsHeader(content string) bool {
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) == "## Tools" {
			return true
		}
	}
	return false
}

func insertAfterTools(content string, snippetLines []string) string {
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		if strings.TrimSpace(line) == "## Tools" {
			insert := append([]string{line, ""}, snippetLines...)
			out := append([]string{}, lines[:i]...)
			out = append(out, insert...)
			out = append(out, lines[i+1:]...)
			return strings.Join(out, "\n")
		}
	}
	return content
}
