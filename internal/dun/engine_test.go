// Package tests focus on end-to-end feedback loop behavior.
package dun

import (
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestHelixMissingArchitecturePromptsAgent(t *testing.T) {
	result := runFixture(t, "helix-missing-architecture", "auto")

	check := findCheck(t, result, "helix-create-architecture")
	if check.Status != "fail" {
		t.Fatalf("expected fail, got %s", check.Status)
	}
	if !strings.Contains(check.Detail, "architecture") {
		t.Fatalf("expected architecture detail, got %q", check.Detail)
	}
}

func TestHelixMissingFeaturesPromptsAgent(t *testing.T) {
	result := runFixture(t, "helix-missing-features", "auto")

	check := findCheck(t, result, "helix-create-feature-specs")
	if check.Status != "fail" {
		t.Fatalf("expected fail, got %s", check.Status)
	}
	if !strings.Contains(check.Detail, "features") {
		t.Fatalf("expected feature detail, got %q", check.Detail)
	}
}

func TestHelixAlignmentRunsAgent(t *testing.T) {
	result := runFixture(t, "helix-alignment", "auto")

	check := findCheck(t, result, "helix-align-specs")
	if check.Status != "warn" {
		t.Fatalf("expected warn, got %s", check.Status)
	}
	if !strings.Contains(check.Signal, "alignment") {
		t.Fatalf("expected alignment signal, got %q", check.Signal)
	}
}

func TestHelixAskModeRequiresApproval(t *testing.T) {
	result := runFixture(t, "helix-missing-architecture", "ask")

	check := findCheck(t, result, "helix-create-architecture")
	if check.Status != "warn" {
		t.Fatalf("expected warn, got %s", check.Status)
	}
	if !strings.Contains(check.Signal, "approval") {
		t.Fatalf("expected approval signal, got %q", check.Signal)
	}
}

func TestHelixStateRulesDetectsMissingStory(t *testing.T) {
	result := runFixture(t, "helix-inconsistent", "auto")

	check := findCheck(t, result, "helix-state-rules")
	if check.Status != "fail" {
		t.Fatalf("expected fail, got %s", check.Status)
	}
	if !strings.Contains(check.Detail, "US-001") {
		t.Fatalf("expected missing US detail, got %q", check.Detail)
	}
	if check.Next == "" {
		t.Fatalf("expected next action")
	}
}

func TestHelixGatesDetectMissingEvidence(t *testing.T) {
	result := runFixture(t, "helix-gates-missing", "auto")

	check := findCheck(t, result, "helix-gates")
	if check.Status != "fail" {
		t.Fatalf("expected fail, got %s", check.Status)
	}
	if !strings.Contains(check.Detail, "docs/helix/01-frame/prd.md") {
		t.Fatalf("expected missing prd detail, got %q", check.Detail)
	}
}

func runFixture(t *testing.T, name string, mode string) Result {
	t.Helper()

	agentCmd := fixturePath(t, "../testdata/agent/agent.sh")
	root := fixturePath(t, "../testdata/repos/"+name)

	opts := Options{
		AgentCmd:     "bash " + agentCmd,
		AgentTimeout: 5 * time.Second,
		AgentMode:    mode,
	}
	result, err := CheckRepo(root, opts)
	if err != nil {
		t.Fatalf("check repo: %v", err)
	}
	return result
}

func findCheck(t *testing.T, result Result, id string) CheckResult {
	t.Helper()
	for _, check := range result.Checks {
		if check.ID == id {
			return check
		}
	}
	t.Fatalf("check %s not found", id)
	return CheckResult{}
}

func fixturePath(t *testing.T, rel string) string {
	t.Helper()
	path, err := filepath.Abs(rel)
	if err != nil {
		t.Fatalf("abs path: %v", err)
	}
	return path
}
