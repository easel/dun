// Package tests focus on end-to-end feedback loop behavior.
package dun

import (
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestHelixMissingArchitecturePromptsAgent(t *testing.T) {
	result := runFixture(t, "helix-missing-architecture", "")

	check := findCheck(t, result, "helix-create-architecture")
	if check.Status != "prompt" {
		t.Fatalf("expected prompt, got %s", check.Status)
	}
	if check.Prompt == nil {
		t.Fatalf("expected prompt envelope")
	}
	if check.Prompt.Kind != "dun.prompt.v1" {
		t.Fatalf("expected prompt kind, got %s", check.Prompt.Kind)
	}
	if !strings.Contains(check.Prompt.Callback.Command, "dun respond --id helix-create-architecture") {
		t.Fatalf("expected callback command, got %q", check.Prompt.Callback.Command)
	}
}

func TestHelixMissingFeaturesEmitsPrompt(t *testing.T) {
	result := runFixture(t, "helix-missing-features", "")

	check := findCheck(t, result, "helix-create-feature-specs")
	if check.Status != "prompt" {
		t.Fatalf("expected prompt, got %s", check.Status)
	}
	if check.Prompt == nil {
		t.Fatalf("expected prompt envelope")
	}
}

func TestHelixAlignmentEmitsPrompt(t *testing.T) {
	result := runFixture(t, "helix-alignment", "")

	check := findCheck(t, result, "helix-align-specs")
	if check.Status != "prompt" {
		t.Fatalf("expected prompt, got %s", check.Status)
	}
	if check.Prompt == nil {
		t.Fatalf("expected prompt envelope")
	}
}

func TestHelixStateRulesDetectsMissingStory(t *testing.T) {
	result := runFixture(t, "helix-inconsistent", "")

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
	result := runFixture(t, "helix-gates-missing", "")

	check := findCheck(t, result, "helix-gates")
	if check.Status != "fail" {
		t.Fatalf("expected fail, got %s", check.Status)
	}
	if len(check.Issues) == 0 {
		t.Fatalf("expected gate issues")
	}
	if !hasIssueSummaryContains(check.Issues, "Create docs/helix/01-frame/prd.md", "section 'scope'") {
		t.Fatalf("expected scope issue, got %+v", check.Issues)
	}
}

func TestHelixAlignmentAutoRunsAgent(t *testing.T) {
	result := runFixture(t, "helix-alignment", "auto")

	check := findCheck(t, result, "helix-align-specs")
	if check.Status != "warn" {
		t.Fatalf("expected warn, got %s", check.Status)
	}
	if !strings.Contains(check.Signal, "alignment") {
		t.Fatalf("expected alignment signal, got %q", check.Signal)
	}
}

func runFixture(t *testing.T, name string, mode string) Result {
	t.Helper()

	root := fixturePath(t, "../testdata/repos/"+name)

	opts := Options{
		AgentTimeout: 5 * time.Second,
		AgentMode:    mode,
	}
	if mode == "auto" {
		orig := execAgentOutput
		execAgentOutput = func(cmdStr, prompt string, timeout time.Duration) ([]byte, error) {
			switch {
			case strings.Contains(prompt, "Check-ID: helix-create-architecture"):
				return []byte(`{"status":"fail","signal":"architecture doc missing","detail":"docs/helix/02-design/architecture.md is missing","next":"Create docs/helix/02-design/architecture.md using the Helix template"}`), nil
			case strings.Contains(prompt, "Check-ID: helix-create-feature-specs"):
				return []byte(`{"status":"fail","signal":"feature specs missing","detail":"docs/helix/01-frame/features/ has no FEAT files","next":"Create FEAT-XXX specs in docs/helix/01-frame/features/"}`), nil
			case strings.Contains(prompt, "Check-ID: helix-align-specs"):
				return []byte(`{"status":"warn","signal":"alignment gaps found","detail":"PRD scope does not fully map to feature specs","next":"Review sections and update specs"}`), nil
			case strings.Contains(prompt, "Check-ID: helix-reconcile-stack"):
				return []byte(`{"status":"warn","signal":"drift plan ready","detail":"Reconciliation plan generated","next":"Apply plan updates"}`), nil
			default:
				return []byte(`{"status":"fail","signal":"unknown check","detail":"agent could not identify check id"}`), nil
			}
		}
		t.Cleanup(func() { execAgentOutput = orig })
		opts.AgentCmd = "stub-agent"
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

func hasIssueSummaryContains(issues []Issue, parts ...string) bool {
	for _, issue := range issues {
		match := true
		for _, part := range parts {
			if !strings.Contains(issue.Summary, part) {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

func fixturePath(t *testing.T, rel string) string {
	t.Helper()
	path, err := filepath.Abs(rel)
	if err != nil {
		t.Fatalf("abs path: %v", err)
	}
	return path
}
