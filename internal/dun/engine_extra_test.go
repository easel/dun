package dun

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestEvalTriggerUnknownType(t *testing.T) {
	if evalTrigger(t.TempDir(), Trigger{Type: "unknown"}) {
		t.Fatalf("expected unknown trigger to be false")
	}
}

func TestEvalTriggerGlobExists(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "a.txt"), "a")
	if !evalTrigger(root, Trigger{Type: "glob-exists", Value: "*.txt"}) {
		t.Fatalf("expected glob trigger true")
	}
}

func TestRunCheckUnknownType(t *testing.T) {
	pc := plannedCheck{Check: Check{Type: "nope"}}
	_, err := runCheck(".", pc, Options{})
	if err == nil {
		t.Fatalf("expected error for unknown check type")
	}
}

func TestRunCheckCommandNotImplemented(t *testing.T) {
	pc := plannedCheck{Check: Check{Type: "command"}}
	_, err := runCheck(".", pc, Options{})
	if err == nil {
		t.Fatalf("expected command not implemented error")
	}
}

func TestConditionsMetError(t *testing.T) {
	_, err := conditionsMet(t.TempDir(), []Rule{{Type: "pattern-count", Path: "missing.txt", Pattern: "("}})
	if err == nil {
		t.Fatalf("expected error for invalid rule")
	}
}

func TestBuildPlanForRootError(t *testing.T) {
	orig := loadBuiltins
	loadBuiltins = func() ([]Plugin, error) {
		return nil, errors.New("boom")
	}
	t.Cleanup(func() { loadBuiltins = orig })

	if _, err := buildPlanForRoot(t.TempDir()); err == nil {
		t.Fatalf("expected buildPlanForRoot error")
	}
}

func TestBuildPlanForRootConditionError(t *testing.T) {
	orig := loadBuiltins
	loadBuiltins = func() ([]Plugin, error) {
		return []Plugin{
			{
				Manifest: Manifest{
					ID:      "p",
					Version: "1",
					Checks: []Check{
						{
							ID: "bad",
							Conditions: []Rule{
								{Type: "pattern-count", Path: "missing.txt", Pattern: "("},
							},
						},
					},
				},
			},
		}, nil
	}
	t.Cleanup(func() { loadBuiltins = orig })

	if _, err := buildPlanForRoot(t.TempDir()); err == nil {
		t.Fatalf("expected buildPlan error")
	}
}

func TestCheckRepoReturnsError(t *testing.T) {
	orig := loadBuiltins
	loadBuiltins = func() ([]Plugin, error) {
		return nil, errors.New("boom")
	}
	t.Cleanup(func() { loadBuiltins = orig })

	if _, err := CheckRepo(t.TempDir(), Options{}); err == nil {
		t.Fatalf("expected error from CheckRepo")
	}
}

func TestCheckRepoRunCheckError(t *testing.T) {
	orig := loadBuiltins
	loadBuiltins = func() ([]Plugin, error) {
		return []Plugin{
			{
				Manifest: Manifest{
					ID:      "p",
					Version: "1",
					Checks: []Check{
						{ID: "bad", Type: "command"},
					},
				},
			},
		}, nil
	}
	t.Cleanup(func() { loadBuiltins = orig })

	if _, err := CheckRepo(t.TempDir(), Options{}); err == nil {
		t.Fatalf("expected runCheck error")
	}
}

func TestPlanRepoReturnsError(t *testing.T) {
	orig := loadBuiltins
	loadBuiltins = func() ([]Plugin, error) {
		return nil, errors.New("boom")
	}
	t.Cleanup(func() { loadBuiltins = orig })

	if _, err := PlanRepo(t.TempDir()); err == nil {
		t.Fatalf("expected error from PlanRepo")
	}
}

func TestIsPluginActiveWithTrigger(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "go.mod"), "module example.com/test")
	plugin := Plugin{Manifest: Manifest{Triggers: []Trigger{{Type: "path-exists", Value: "go.mod"}}}}
	if !isPluginActive(root, plugin) {
		t.Fatalf("expected plugin active")
	}
}

func TestIsPluginActiveNoTriggers(t *testing.T) {
	plugin := Plugin{Manifest: Manifest{}}
	if !isPluginActive(t.TempDir(), plugin) {
		t.Fatalf("expected plugin active without triggers")
	}
}

func TestIsPluginActiveNoMatch(t *testing.T) {
	plugin := Plugin{Manifest: Manifest{Triggers: []Trigger{{Type: "path-exists", Value: "missing"}}}}
	if isPluginActive(t.TempDir(), plugin) {
		t.Fatalf("expected plugin inactive")
	}
}

func TestBuildPlanSkipsConditions(t *testing.T) {
	root := t.TempDir()
	plugin := Plugin{
		Manifest: Manifest{
			Checks: []Check{
				{ID: "skip", Conditions: []Rule{{Type: "path-exists", Path: "missing.txt"}}},
				{ID: "keep", Conditions: []Rule{{Type: "path-missing", Path: "missing.txt"}}},
			},
		},
	}
	plan, err := buildPlan(root, []Plugin{plugin})
	if err != nil {
		t.Fatalf("build plan: %v", err)
	}
	if len(plan) != 1 || plan[0].Check.ID != "keep" {
		t.Fatalf("expected only keep check, got %v", plan)
	}
}

func TestEvalTriggerPathExistsFalseWhenMissing(t *testing.T) {
	root := t.TempDir()
	if evalTrigger(root, Trigger{Type: "path-exists", Value: "missing"}) {
		t.Fatalf("expected false for missing path")
	}
}

func TestEvalTriggerGlobExistsFalseWhenMissing(t *testing.T) {
	root := t.TempDir()
	if evalTrigger(root, Trigger{Type: "glob-exists", Value: "*.md"}) {
		t.Fatalf("expected false for missing glob")
	}
}

func TestRunCheckGoTest(t *testing.T) {
	binDir := stubGoBinary(t)
	t.Setenv("PATH", binDir)
	pc := plannedCheck{Check: Check{Type: "go-test", ID: "go-test"}}
	res, err := runCheck(t.TempDir(), pc, Options{})
	if err != nil {
		t.Fatalf("run go-test: %v", err)
	}
	if res.Status != "pass" {
		t.Fatalf("expected pass")
	}
}

func TestRunCheckGoCoverage(t *testing.T) {
	binDir := stubGoBinary(t)
	t.Setenv("PATH", binDir)
	pc := plannedCheck{Check: Check{Type: "go-coverage", ID: "go-coverage"}}
	res, err := runCheck(t.TempDir(), pc, Options{})
	if err != nil {
		t.Fatalf("run go-coverage: %v", err)
	}
	if res.Status != "pass" {
		t.Fatalf("expected pass")
	}
}

func TestRunCheckGoVet(t *testing.T) {
	binDir := stubGoBinary(t)
	t.Setenv("PATH", binDir)
	pc := plannedCheck{Check: Check{Type: "go-vet", ID: "go-vet"}}
	res, err := runCheck(t.TempDir(), pc, Options{})
	if err != nil {
		t.Fatalf("run go-vet: %v", err)
	}
	if res.Status != "pass" {
		t.Fatalf("expected pass")
	}
}

func TestRunCheckGoStaticcheckWarnWhenMissing(t *testing.T) {
	binDir := stubGoBinary(t)
	t.Setenv("PATH", binDir)
	pc := plannedCheck{Check: Check{Type: "go-staticcheck", ID: "go-staticcheck"}}
	res, err := runCheck(t.TempDir(), pc, Options{})
	if err != nil {
		t.Fatalf("run go-staticcheck: %v", err)
	}
	if res.Status != "warn" {
		t.Fatalf("expected warn")
	}
}

func TestRunCheckRuleSet(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "file.txt"), "ok")
	pc := plannedCheck{Check: Check{Type: "rule-set", ID: "rules", Rules: []Rule{{Type: "path-exists", Path: "file.txt"}}}}
	res, err := runCheck(root, pc, Options{})
	if err != nil {
		t.Fatalf("run rule-set: %v", err)
	}
	if res.Status != "pass" {
		t.Fatalf("expected pass")
	}
}

func TestRunCheckGitStatusAndHook(t *testing.T) {
	root := tempGitRepo(t)
	pc := plannedCheck{Check: Check{Type: "git-status", ID: "git-status"}}
	res, err := runCheck(root, pc, Options{})
	if err != nil {
		t.Fatalf("run git-status: %v", err)
	}
	if res.Status == "" {
		t.Fatalf("expected status")
	}
	pc = plannedCheck{Check: Check{Type: "hook-check", ID: "git-hooks"}}
	res, err = runCheck(root, pc, Options{})
	if err != nil {
		t.Fatalf("run hook-check: %v", err)
	}
	if res.Status == "" {
		t.Fatalf("expected status")
	}
}

func TestRunCheckGateAndStateRules(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "gate.yml"), "input_gates:\n  - criteria: \"Gate\"\n    required: false\n    evidence: \"docs/missing.md\"\n")
	writeFile(t, filepath.Join(dir, "rules.yml"), `artifact_patterns:
  story:
    frame: { pattern: "frame/US-{id}.md" }
    design: { pattern: "design/TD-{id}.md" }
    test: { pattern: "test/TP-{id}.md" }
    build: { pattern: "build/IP-{id}.md" }
`)
	plugin := Plugin{FS: os.DirFS(dir), Base: "."}
	gateCheck := plannedCheck{Plugin: plugin, Check: Check{Type: "gates", ID: "gates", GateFiles: []string{"gate.yml"}}}
	res, err := runCheck(dir, gateCheck, Options{})
	if err != nil {
		t.Fatalf("run gates: %v", err)
	}
	if res.Status == "" {
		t.Fatalf("expected status")
	}

	stateCheck := plannedCheck{Plugin: plugin, Check: Check{Type: "state-rules", ID: "state", StateRules: "rules.yml"}}
	res, err = runCheck(dir, stateCheck, Options{})
	if err != nil {
		t.Fatalf("run state rules: %v", err)
	}
	if res.Status == "" {
		t.Fatalf("expected status")
	}
}

func TestRunCheckAgentPrompt(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "prompt.md"), "hello")
	plugin := Plugin{FS: os.DirFS(dir), Base: "."}
	pc := plannedCheck{Plugin: plugin, Check: Check{Type: "agent", ID: "agent", Prompt: "prompt.md", Description: "desc"}}
	res, err := runCheck(dir, pc, Options{AgentMode: "prompt", AutomationMode: "auto"})
	if err != nil {
		t.Fatalf("run agent: %v", err)
	}
	if res.Status != "prompt" {
		t.Fatalf("expected prompt")
	}
}

func TestCheckRepoEmptyRoot(t *testing.T) {
	root := tempGitRepo(t)
	_, err := CheckRepo(root, Options{})
	if err != nil {
		t.Fatalf("check repo: %v", err)
	}
}

// TC-001: Go plugin activates when go.mod present
func TestGoPluginActiveWhenGoModExists(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "go.mod"), "module example.com/test")

	plan, err := PlanRepo(root)
	if err != nil {
		t.Fatalf("plan repo: %v", err)
	}

	// Verify Go checks are included
	goCheckIDs := []string{"go-test", "go-coverage", "go-vet", "go-staticcheck"}
	for _, id := range goCheckIDs {
		if !planHasCheck(plan, id) {
			t.Errorf("expected check %s when go.mod exists", id)
		}
	}
}

// TC-002: Go plugin deactivates when go.mod absent
func TestGoPluginInactiveWhenGoModMissing(t *testing.T) {
	root := t.TempDir()
	// No go.mod file

	plan, err := PlanRepo(root)
	if err != nil {
		t.Fatalf("plan repo: %v", err)
	}

	goCheckIDs := []string{"go-test", "go-coverage", "go-vet", "go-staticcheck"}
	for _, id := range goCheckIDs {
		if planHasCheck(plan, id) {
			t.Errorf("unexpected check %s when go.mod missing", id)
		}
	}
}

// TC-003: Helix plugin activates when docs/helix present
func TestHelixPluginActiveWhenDocsHelixExists(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "docs", "helix"), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	plan, err := PlanRepo(root)
	if err != nil {
		t.Fatalf("plan repo: %v", err)
	}

	// Verify Helix checks are included
	if !planHasCheck(plan, "helix-gates") {
		t.Error("expected helix-gates check when docs/helix exists")
	}
	if !planHasCheck(plan, "helix-state-rules") {
		t.Error("expected helix-state-rules check when docs/helix exists")
	}
}

// TC-003 (inverse): Helix plugin deactivates when docs/helix missing
func TestHelixPluginInactiveWhenDocsHelixMissing(t *testing.T) {
	root := t.TempDir()
	// No docs/helix directory

	plan, err := PlanRepo(root)
	if err != nil {
		t.Fatalf("plan repo: %v", err)
	}

	helixCheckIDs := []string{"helix-gates", "helix-state-rules", "helix-create-architecture"}
	for _, id := range helixCheckIDs {
		if planHasCheck(plan, id) {
			t.Errorf("unexpected check %s when docs/helix missing", id)
		}
	}
}

// TC-004: Deterministic ordering across runs
func TestDeterministicOrdering(t *testing.T) {
	root := tempGitRepo(t)
	writeFile(t, filepath.Join(root, "go.mod"), "module example.com/test")
	if err := os.MkdirAll(filepath.Join(root, "docs", "helix"), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	// Run PlanRepo multiple times
	var previousIDs []string
	for i := 0; i < 5; i++ {
		plan, err := PlanRepo(root)
		if err != nil {
			t.Fatalf("plan repo run %d: %v", i, err)
		}

		var currentIDs []string
		for _, check := range plan.Checks {
			currentIDs = append(currentIDs, check.ID)
		}

		if previousIDs != nil {
			if !slicesEqual(previousIDs, currentIDs) {
				t.Errorf("run %d: check order differs\nprevious: %v\ncurrent: %v",
					i, previousIDs, currentIDs)
			}
		}
		previousIDs = currentIDs
	}
}

// Helper: check if plan contains a check with the given ID
func planHasCheck(plan Plan, id string) bool {
	for _, check := range plan.Checks {
		if check.ID == id {
			return true
		}
	}
	return false
}

// Helper: compare two string slices for equality
func slicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// Gap-4: Test for checks with multiple conditions where some pass and some fail
func TestBuildPlanMultipleConditionsAllMustPass(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "exists.txt"), "ok")
	// missing.txt does not exist

	plugin := Plugin{
		Manifest: Manifest{
			Checks: []Check{
				{
					ID: "partial-match",
					Conditions: []Rule{
						{Type: "path-exists", Path: "exists.txt"},  // passes
						{Type: "path-exists", Path: "missing.txt"}, // fails
					},
				},
			},
		},
	}
	plan, err := buildPlan(root, []Plugin{plugin})
	if err != nil {
		t.Fatalf("build plan: %v", err)
	}
	if len(plan) != 0 {
		t.Fatalf("expected check to be skipped when any condition fails, got %d checks", len(plan))
	}
}

// Gap-6: Test for agent check with missing prompt template (empty prompt path)
func TestRunCheckAgentMissingPromptTemplate(t *testing.T) {
	dir := t.TempDir()
	// No prompt.md file - prompt template path is empty
	plugin := Plugin{FS: os.DirFS(dir), Base: "."}
	pc := plannedCheck{
		Plugin: plugin,
		Check:  Check{Type: "agent", ID: "agent-missing-prompt", Prompt: "", Description: "test"},
	}
	_, err := runCheck(dir, pc, Options{AgentMode: "prompt"})
	if err == nil {
		t.Fatalf("expected error for missing prompt template")
	}
}
