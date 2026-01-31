# TP-003: Plugin System Test Plan

**User Story**: [US-003: Extend Checks with Plugin Manifests](../../01-frame/user-stories/US-003-plugin-system.md)

**Status**: Draft
**Created**: 2026-01-30

---

## 1. Acceptance Criteria

From US-003:

| ID | Criterion | Description |
|----|-----------|-------------|
| AC-1 | Built-in plugin manifests are embedded in the binary | Plugins like Helix must be compiled into the binary via `embed.FS` |
| AC-2 | Plugins activate based on repo signals (paths/globs) | Triggers like `path-exists` and `glob-exists` control plugin activation |
| AC-3 | Supported check types include rule-set, gates, state rules, and agent prompts | All four check types must be runnable by the engine |
| AC-4 | Check ordering is deterministic | Sorting by plugin priority, check priority, phase, and ID |
| AC-5 | The Helix plugin activates when `docs/helix/` exists | Specific trigger validation for the Helix plugin |

---

## 2. Test Coverage Mapping

### AC-1: Built-in Plugin Manifests Embedded in Binary

| Test File | Test Name | Coverage |
|-----------|-----------|----------|
| `internal/dun/plugin_loader_test.go` | `TestLoadBuiltinsSuccess` | Verifies `LoadBuiltins()` returns plugins from embedded FS |
| `internal/dun/plugin_loader_test.go` | `TestLoadBuiltinsError` | Verifies error handling when builtin plugin load fails |
| `internal/dun/plugin_loader_test.go` | `TestLoadPluginFSSuccess` | Verifies parsing of plugin.yaml from filesystem |
| `internal/dun/plugin_loader_test.go` | `TestLoadPluginFSReadError` | Verifies error on missing manifest file |
| `internal/dun/plugin_loader_test.go` | `TestLoadPluginFSInvalidYAML` | Verifies error on malformed YAML |
| `internal/dun/plugin_loader_test.go` | `TestLoadPluginFSMissingFields` | Verifies validation of required fields (id, version) |
| `internal/dun/plugin_loader_test.go` | `TestLoadPluginFSNoChecks` | Verifies error when no checks defined |

**Status**: COVERED

---

### AC-2: Plugins Activate Based on Repo Signals (paths/globs)

| Test File | Test Name | Coverage |
|-----------|-----------|----------|
| `internal/dun/engine_extra_test.go` | `TestIsPluginActiveWithTrigger` | Verifies `path-exists` trigger activates plugin |
| `internal/dun/engine_extra_test.go` | `TestIsPluginActiveNoTriggers` | Verifies plugin without triggers is always active |
| `internal/dun/engine_extra_test.go` | `TestIsPluginActiveNoMatch` | Verifies plugin with unmet triggers is inactive |
| `internal/dun/engine_extra_test.go` | `TestEvalTriggerUnknownType` | Verifies unknown trigger type returns false |
| `internal/dun/engine_extra_test.go` | `TestEvalTriggerGlobExists` | Verifies `glob-exists` trigger works |
| `internal/dun/engine_extra_test.go` | `TestEvalTriggerPathExistsFalseWhenMissing` | Verifies `path-exists` returns false when path missing |
| `internal/dun/engine_extra_test.go` | `TestEvalTriggerGlobExistsFalseWhenMissing` | Verifies `glob-exists` returns false when no match |

**Status**: COVERED

---

### AC-3: Supported Check Types (rule-set, gates, state-rules, agent)

#### 3a. Rule-Set Check Type

| Test File | Test Name | Coverage |
|-----------|-----------|----------|
| `internal/dun/rules_test.go` | `TestRunRuleSetStatuses` | Verifies pass/warn/fail status based on rules |
| `internal/dun/rules_test.go` | `TestEvalRulePathExistsAndMissing` | Verifies `path-exists` and `path-missing` rules |
| `internal/dun/rules_test.go` | `TestEvalRuleGlobCounts` | Verifies `glob-min-count` and `glob-max-count` rules |
| `internal/dun/rules_test.go` | `TestEvalRulePatternCount` | Verifies `pattern-count` rule |
| `internal/dun/rules_test.go` | `TestEvalRuleUniqueIDs` | Verifies `unique-ids` rule |
| `internal/dun/rules_test.go` | `TestEvalRuleCrossReference` | Verifies `cross-reference` rule |
| `internal/dun/rules_test.go` | `TestEvalRuleUnknownType` | Verifies error for unknown rule type |
| `internal/dun/engine_extra_test.go` | `TestRunCheckRuleSet` | Verifies rule-set check runs through engine |

**Status**: COVERED

#### 3b. Gates Check Type

| Test File | Test Name | Coverage |
|-----------|-----------|----------|
| `internal/dun/gates_test.go` | `TestRunGateCheckPassesWhenSatisfied` | Verifies pass when evidence exists |
| `internal/dun/gates_test.go` | `TestRunGateCheckFailsWhenRequiredMissing` | Verifies fail for missing required evidence |
| `internal/dun/gates_test.go` | `TestRunGateCheckWarnsWhenOptionalMissing` | Verifies warn for missing optional evidence |
| `internal/dun/gates_test.go` | `TestRunGateCheckMissingGateFiles` | Verifies error when no gate files specified |
| `internal/dun/gates_test.go` | `TestRunGateCheckMissingGateFile` | Verifies error when gate file not found |
| `internal/dun/gates_test.go` | `TestRunGateCheckEvidenceError` | Verifies error on invalid evidence path |
| `internal/dun/gates_test.go` | `TestLoadGateFileErrors` | Verifies error handling for gate file loading |
| `internal/dun/gates_test.go` | `TestEvidenceMissingWithAnchor` | Verifies anchor detection in markdown |
| `internal/dun/gates_test.go` | `TestEvidenceMissingNoAnchor` | Verifies file existence check without anchor |
| `internal/dun/gates_test.go` | `TestSplitEvidence` | Verifies evidence path parsing |
| `internal/dun/gates_test.go` | `TestHasMarkdownAnchor` | Verifies markdown heading detection |
| `internal/dun/gates_test.go` | `TestSlugify` | Verifies anchor slug generation |
| `internal/dun/gates_test.go` | `TestBuildGateActionBranches` | Verifies action message generation |
| `internal/dun/engine_extra_test.go` | `TestRunCheckGateAndStateRules` | Verifies gates check runs through engine |
| `internal/dun/engine_test.go` | `TestHelixGatesDetectMissingEvidence` | Integration test with Helix plugin |

**Status**: COVERED

#### 3c. State-Rules Check Type

| Test File | Test Name | Coverage |
|-----------|-----------|----------|
| `internal/dun/state_rules_test.go` | `TestRunStateRulesPassAndFail` | Verifies pass/fail based on artifact progression |
| `internal/dun/state_rules_test.go` | `TestRunStateRulesMissingPath` | Verifies error when no state_rules path |
| `internal/dun/state_rules_test.go` | `TestRunStateRulesReadError` | Verifies error when rules file not found |
| `internal/dun/state_rules_test.go` | `TestRunStateRulesParseError` | Verifies error on malformed YAML |
| `internal/dun/state_rules_test.go` | `TestIdsForPatternEmpty` | Verifies empty pattern handling |
| `internal/dun/state_rules_test.go` | `TestPrefixAndParseID` | Verifies ID extraction from filenames |
| `internal/dun/state_rules_test.go` | `TestIdsForPatternGlobError` | Verifies error on invalid glob |
| `internal/dun/state_rules_test.go` | `TestRunStateRulesFrameGlobError` | Verifies error on frame pattern glob |
| `internal/dun/state_rules_test.go` | `TestRunStateRulesDesignGlobError` | Verifies error on design pattern glob |
| `internal/dun/state_rules_test.go` | `TestRunStateRulesTestGlobError` | Verifies error on test pattern glob |
| `internal/dun/state_rules_test.go` | `TestRunStateRulesBuildGlobError` | Verifies error on build pattern glob |
| `internal/dun/engine_extra_test.go` | `TestRunCheckGateAndStateRules` | Verifies state-rules check runs through engine |
| `internal/dun/engine_test.go` | `TestHelixStateRulesDetectsMissingStory` | Integration test with Helix plugin |

**Status**: COVERED

#### 3d. Agent Prompt Check Type

| Test File | Test Name | Coverage |
|-----------|-----------|----------|
| `internal/dun/agent_test.go` | `TestRenderPromptIncludesAutomationMode` | Verifies template rendering with automation mode |
| `internal/dun/engine_extra_test.go` | `TestRunCheckAgentPrompt` | Verifies agent check returns prompt status |
| `internal/dun/engine_test.go` | `TestHelixMissingArchitecturePromptsAgent` | Integration: agent prompt with callback |
| `internal/dun/engine_test.go` | `TestHelixMissingFeaturesEmitsPrompt` | Integration: agent prompt with conditions |
| `internal/dun/engine_test.go` | `TestHelixAlignmentEmitsPrompt` | Integration: agent prompt with multiple inputs |
| `internal/dun/engine_test.go` | `TestHelixAlignmentAutoRunsAgent` | Integration: agent auto-execution mode |
| `cmd/dun/main_test.go` | `TestCheckUsesConfigAgentAuto` | CLI integration: agent auto mode |
| `cmd/dun/main_test.go` | `TestCheckResolvesRepoRootFromSubdir` | CLI integration: agent from subdirectory |

**Status**: COVERED

---

### AC-4: Check Ordering is Deterministic

| Test File | Test Name | Coverage |
|-----------|-----------|----------|
| `internal/dun/engine_sort_test.go` | `TestSortPlanByPluginPriority` | Verifies plugin priority sorting |
| `internal/dun/engine_sort_test.go` | `TestSortPlanByCheckPriority` | Verifies check priority sorting within plugin |
| `internal/dun/engine_sort_test.go` | `TestSortPlanByPhase` | Verifies phase ordering (frame, design, test, build, deploy, iterate) |
| `internal/dun/engine_sort_test.go` | `TestSortPlanByID` | Verifies alphabetical ordering as tiebreaker |
| `internal/dun/engine_sort_test.go` | `TestSortPlanCombined` | Verifies combined sorting criteria |
| `internal/dun/engine_sort_test.go` | `TestSortPlanEmptyPlan` | Edge case: empty plan |
| `internal/dun/engine_sort_test.go` | `TestSortPlanSingleElement` | Edge case: single element |

**Status**: COVERED

---

### AC-5: Helix Plugin Activates When `docs/helix/` Exists

| Test File | Test Name | Coverage |
|-----------|-----------|----------|
| `internal/dun/plan_test.go` | `TestPlanRepoIncludesHelixChecks` | Verifies Helix checks included in plan |
| `internal/dun/engine_test.go` | `TestHelixMissingArchitecturePromptsAgent` | Uses fixture with `docs/helix/` |
| `internal/dun/engine_test.go` | `TestHelixMissingFeaturesEmitsPrompt` | Uses fixture with `docs/helix/` |
| `internal/dun/engine_test.go` | `TestHelixAlignmentEmitsPrompt` | Uses fixture with `docs/helix/` |
| `internal/dun/engine_test.go` | `TestHelixStateRulesDetectsMissingStory` | Uses fixture with `docs/helix/` |
| `internal/dun/engine_test.go` | `TestHelixGatesDetectMissingEvidence` | Uses fixture with `docs/helix/` |

**Status**: PARTIALLY COVERED (see gaps below)

---

## 3. Coverage Gaps

### Gap 1: Negative Test for Helix Activation

**Missing**: Explicit test that Helix plugin is NOT active when `docs/helix/` does NOT exist.

**Current State**: `TestIsPluginActiveNoMatch` tests the generic mechanism but not specifically for Helix.

**Proposed Test**:
```go
func TestHelixPluginInactiveWithoutDocsHelix(t *testing.T) {
    root := tempGitRepo(t)
    // No docs/helix/ directory
    plan, err := PlanRepo(root)
    if err != nil {
        t.Fatalf("plan repo: %v", err)
    }
    for _, check := range plan.Checks {
        if strings.HasPrefix(check.ID, "helix-") {
            t.Fatalf("helix check %s should not be active", check.ID)
        }
    }
}
```

---

### Gap 2: Plugin Manifest Field Validation

**Missing**: Tests for edge cases in manifest validation:
- Empty checks array (different from no checks field)
- Check with missing required fields (id, type)
- Invalid check type in manifest

**Proposed Tests**:
```go
func TestLoadPluginFSEmptyChecksArray(t *testing.T) {
    fs := fstest.MapFS{
        "plugin.yaml": {Data: []byte("id: test\nversion: \"1\"\nchecks: []")},
    }
    _, err := loadPluginFS(fs, ".")
    if err == nil {
        t.Fatalf("expected error for empty checks array")
    }
}

func TestLoadPluginFSCheckMissingID(t *testing.T) {
    fs := fstest.MapFS{
        "plugin.yaml": {Data: []byte("id: test\nversion: \"1\"\nchecks:\n  - type: rule-set\n    description: \"x\"")},
    }
    _, err := loadPluginFS(fs, ".")
    if err == nil {
        t.Fatalf("expected error for check without id")
    }
}
```

---

### Gap 3: Multiple Plugin Priority Ordering

**Missing**: Test verifying correct ordering when multiple plugins with different priorities are active.

**Current State**: `TestSortPlanByPluginPriority` tests sorting logic but with mock data, not real plugins.

**Proposed Test**:
```go
func TestMultiplePluginsSortedByPriority(t *testing.T) {
    // Create test setup with core (priority 10) and helix (priority 50) plugins active
    root := tempGitRepo(t)
    // Create docs/helix/ to activate helix
    os.MkdirAll(filepath.Join(root, "docs", "helix"), 0755)
    // Create go.mod to activate go plugin
    writeFile(t, filepath.Join(root, "go.mod"), "module test")

    plan, err := PlanRepo(root)
    if err != nil {
        t.Fatalf("plan repo: %v", err)
    }

    // Verify core checks come before helix checks (if core priority < helix priority)
    // This tests real plugin ordering behavior
}
```

---

### Gap 4: Check Conditions with Multiple Rules

**Missing**: Test for check conditions where some rules pass and some fail (currently only single-rule conditions tested).

**Proposed Test**:
```go
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
                        {Type: "path-exists", Path: "exists.txt"},   // passes
                        {Type: "path-exists", Path: "missing.txt"},  // fails
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
        t.Fatalf("expected check to be skipped when any condition fails")
    }
}
```

---

### Gap 5: Glob Trigger Pattern Matching

**Missing**: Tests for glob triggers with complex patterns (e.g., `**/*.go`, `src/**/*.ts`).

**Proposed Test**:
```go
func TestEvalTriggerGlobExistsRecursive(t *testing.T) {
    root := t.TempDir()
    os.MkdirAll(filepath.Join(root, "src", "pkg"), 0755)
    writeFile(t, filepath.Join(root, "src", "pkg", "file.go"), "package pkg")

    if !evalTrigger(root, Trigger{Type: "glob-exists", Value: "**/*.go"}) {
        t.Fatalf("expected recursive glob trigger to match")
    }
}
```

---

### Gap 6: Agent Prompt with Missing Template File

**Missing**: Test for agent check when prompt template file is missing.

**Proposed Test**:
```go
func TestRunCheckAgentMissingPrompt(t *testing.T) {
    dir := t.TempDir()
    // No prompt.md file
    plugin := Plugin{FS: os.DirFS(dir), Base: "."}
    pc := plannedCheck{
        Plugin: plugin,
        Check:  Check{Type: "agent", ID: "agent", Prompt: "missing.md"},
    }
    _, err := runCheck(dir, pc, Options{AgentMode: "prompt"})
    if err == nil {
        t.Fatalf("expected error for missing prompt template")
    }
}
```

---

### Gap 7: State Rules with Partial Progression

**Missing**: Test where artifacts exist at some phases but not others (e.g., US-001 has frame and design but not test or build).

**Current State**: Tests check complete vs missing, but not partial progression scenarios.

**Proposed Test**:
```go
func TestRunStateRulesPartialProgression(t *testing.T) {
    dir := t.TempDir()
    writeFile(t, filepath.Join(dir, "rules.yml"), `artifact_patterns:
  story:
    frame: { pattern: "frame/US-{id}.md" }
    design: { pattern: "design/TD-{id}.md" }
    test: { pattern: "test/TP-{id}.md" }
    build: { pattern: "build/IP-{id}.md" }
`)
    for _, subdir := range []string{"frame", "design", "test", "build"} {
        os.MkdirAll(filepath.Join(dir, subdir), 0755)
    }
    // US-001 has frame and design, but not test
    writeFile(t, filepath.Join(dir, "frame", "US-001.md"), "US-001")
    writeFile(t, filepath.Join(dir, "design", "TD-001.md"), "TD-001")
    // No TP-001.md

    plugin := Plugin{FS: os.DirFS(dir), Base: "."}
    check := Check{ID: "state", StateRules: "rules.yml"}
    res, err := runStateRules(dir, plugin, check)
    if err != nil {
        t.Fatalf("run state rules: %v", err)
    }
    if res.Status != "fail" {
        t.Fatalf("expected fail for partial progression, got %s", res.Status)
    }
    if !strings.Contains(res.Detail, "001") {
        t.Fatalf("expected detail to mention missing artifact ID")
    }
}
```

---

## 4. Proposed Test Cases Summary

| Gap ID | Proposed Test | Priority | Effort |
|--------|--------------|----------|--------|
| Gap-1 | `TestHelixPluginInactiveWithoutDocsHelix` | High | Low |
| Gap-2a | `TestLoadPluginFSEmptyChecksArray` | Medium | Low |
| Gap-2b | `TestLoadPluginFSCheckMissingID` | Medium | Low |
| Gap-3 | `TestMultiplePluginsSortedByPriority` | Medium | Medium |
| Gap-4 | `TestBuildPlanMultipleConditionsAllMustPass` | High | Low |
| Gap-5 | `TestEvalTriggerGlobExistsRecursive` | Low | Low |
| Gap-6 | `TestRunCheckAgentMissingPrompt` | High | Low |
| Gap-7 | `TestRunStateRulesPartialProgression` | Medium | Medium |

---

## 5. Test Implementation Files

New tests should be added to existing test files:

| Gap | Target File |
|-----|-------------|
| Gap-1 | `internal/dun/plan_test.go` |
| Gap-2 | `internal/dun/plugin_loader_test.go` |
| Gap-3 | `internal/dun/plan_test.go` |
| Gap-4 | `internal/dun/engine_extra_test.go` |
| Gap-5 | `internal/dun/engine_extra_test.go` |
| Gap-6 | `internal/dun/agent_extra_test.go` (new file or `engine_extra_test.go`) |
| Gap-7 | `internal/dun/state_rules_test.go` |

---

## 6. Execution Commands

```bash
# Run all plugin system tests
go test -v ./internal/dun/... -run "Plugin|Trigger|Gate|State|Agent|Sort|Plan"

# Run with coverage
go test -coverprofile=coverage.out ./internal/dun/...
go tool cover -html=coverage.out -o coverage.html

# Run specific acceptance criteria tests
go test -v ./internal/dun/... -run "LoadBuiltin"        # AC-1
go test -v ./internal/dun/... -run "Trigger|Active"     # AC-2
go test -v ./internal/dun/... -run "RuleSet|Gate|State|Agent"  # AC-3
go test -v ./internal/dun/... -run "Sort"               # AC-4
go test -v ./internal/dun/... -run "Helix"              # AC-5
```

---

## 7. Traceability Matrix

| Acceptance Criterion | Test Coverage | Gaps Identified |
|---------------------|---------------|-----------------|
| AC-1: Embedded manifests | 7 tests | None |
| AC-2: Trigger activation | 7 tests | Gap-5 |
| AC-3a: Rule-set | 8 tests | None |
| AC-3b: Gates | 15 tests | None |
| AC-3c: State-rules | 12 tests | Gap-7 |
| AC-3d: Agent | 8 tests | Gap-6 |
| AC-4: Deterministic ordering | 7 tests | Gap-3 |
| AC-5: Helix activation | 6 tests | Gap-1 |

**Overall Coverage**: High, with 8 identified gaps that should be addressed for complete coverage.
