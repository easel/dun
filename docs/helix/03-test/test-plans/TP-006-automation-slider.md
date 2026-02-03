---
dun:
  id: TP-006
  depends_on:
  - TD-006
---
# TP-006: Test Plan for Automation Slider

**User Story:** US-006 - Control Autonomy With an Automation Slider

**Summary:** As a maintainer, I want to choose how much autonomy Dun has so I can require review when needed or let it run freely.

---

## 1. Acceptance Criteria

| ID | Criterion | Description |
|----|-----------|-------------|
| AC-1 | CLI flag for automation mode | I can set automation mode via a CLI flag |
| AC-2 | Config file default | I can set a default automation mode via `.dun/config.yaml` |
| AC-3 | Manual mode behavior | Manual mode requires approval for each suggestion |
| AC-4 | Yolo mode behavior | Yolo mode allows Dun to fill in missing artifacts to declare completeness |

---

## 2. Existing Test Coverage

### 2.1 Tests in `/home/erik/gt/dun/crew/oscar/internal/dun/agent_test.go`

| Test | Description | Covers |
|------|-------------|--------|
| `TestRenderPromptIncludesAutomationMode` | Verifies that the automation mode is rendered into prompt templates using `{{ .AutomationMode }}` | AC-1 (partial) |

### 2.2 Tests in `/home/erik/gt/dun/crew/oscar/cmd/dun/main_test.go`

| Test | Description | Covers |
|------|-------------|--------|
| `TestCheckUsesConfigAgentAuto` | Uses config with `automation: auto` and `mode: auto`, verifies agent responds | AC-2 (partial) |
| `TestCallHarnessClaudeYolo` | Verifies claude harness accepts yolo mode | AC-4 (partial) |
| `TestCallHarnessCodexYolo` | Verifies codex harness accepts yolo mode | AC-4 (partial) |
| `TestPrintPromptVariants` | Includes automation mode in output verification (`Automation mode: yolo`) | AC-1 (partial) |
| `TestRunLoopWithConfig` | Uses config with `mode: auto` | AC-2 (partial) |

---

## 3. Coverage Gap Analysis

### AC-1: CLI flag for automation mode

**Current Coverage:**
- `TestPrintPromptVariants` verifies mode appears in output
 - `TestRunCheckPromptOutput` exercises `check --prompt` output path

**Gaps:**
- No test verifies CLI flag is accepted on `check` command
- No test verifies CLI flag is accepted on `loop` command (only via config)
- No test for `--automation manual` flag
- No test for `--automation auto` flag
- No test for invalid automation mode values (e.g., `--automation invalid`)
- No test verifying CLI flag overrides config file setting

### AC-2: Config file default

**Current Coverage:**
- `TestCheckUsesConfigAgentAuto` uses `automation: auto` in config
- `TestRunLoopWithConfig` uses `mode: auto` in config

**Gaps:**
- No test for `automation: manual` in config
- No test for `automation: yolo` in config
- No test verifying config default is used when no CLI flag provided
- No test for missing automation field (should use sensible default)
- No test for config value case sensitivity (e.g., `YOLO` vs `yolo`)

### AC-3: Manual mode behavior

**Current Coverage:**
- None

**Gaps:**
- No test verifying manual mode returns prompts for agent checks instead of executing
- No test verifying manual mode halts for user approval
- No test for manual mode interaction with different check types
- No test for manual mode output format (should include approval-required indicator)

### AC-4: Yolo mode behavior

**Current Coverage:**
- `TestCallHarnessClaudeYolo` and `TestCallHarnessCodexYolo` verify harness accepts yolo
- `TestRenderPromptIncludesAutomationMode` verifies mode is in prompt (uses `yolo`)

**Gaps:**
- No test verifying yolo mode executes without prompting
- No test verifying yolo mode allows artifact creation
- No test for yolo mode completing checks that would normally prompt
- No integration test showing end-to-end yolo behavior

---

## 4. Proposed Test Cases

### 4.1 CLI Flag Tests (AC-1)

| Test ID | Test Name | Location | Description |
|---------|-----------|----------|-------------|
| TC-001 | `TestCheckAcceptsAutomationFlag` | `cmd/dun/main_test.go` | Verify `dun check --automation manual` is accepted |
| TC-002 | `TestCheckAcceptsAutomationYolo` | `cmd/dun/main_test.go` | Verify `dun check --automation yolo` is accepted |
| TC-003 | `TestCheckAcceptsAutomationAuto` | `cmd/dun/main_test.go` | Verify `dun check --automation auto` is accepted |
| TC-004 | `TestCheckRejectsInvalidAutomation` | `cmd/dun/main_test.go` | Verify `dun check --automation invalid` returns error |
| TC-005 | `TestLoopAcceptsAutomationFlag` | `cmd/dun/main_test.go` | Verify `dun loop --automation manual` is accepted |
| TC-006 | `TestCLIFlagOverridesConfig` | `cmd/dun/main_test.go` | Verify CLI `--automation yolo` overrides config `automation: manual` |

### 4.2 Config File Tests (AC-2)

| Test ID | Test Name | Location | Description |
|---------|-----------|----------|-------------|
| TC-007 | `TestConfigAutomationManual` | `cmd/dun/main_test.go` | Verify config with `automation: manual` is read correctly |
| TC-008 | `TestConfigAutomationYolo` | `cmd/dun/main_test.go` | Verify config with `automation: yolo` is read correctly |
| TC-009 | `TestConfigMissingAutomation` | `cmd/dun/main_test.go` | Verify default automation mode when not specified in config |
| TC-010 | `TestConfigAutomationCaseInsensitive` | `internal/dun/config_test.go` | Verify `YOLO`, `Yolo`, `yolo` all work |

### 4.3 Manual Mode Behavior Tests (AC-3)

| Test ID | Test Name | Location | Description |
|---------|-----------|----------|-------------|
| TC-011 | `TestManualModeReturnsPrompt` | `cmd/dun/main_test.go` | Verify agent checks return `prompt` status in manual mode |
| TC-012 | `TestManualModeNoAutoExecution` | `internal/dun/agent_test.go` | Verify agent is not called automatically in manual mode |
| TC-013 | `TestManualModePromptHalts` | `cmd/dun/main_test.go` | Verify check --prompt with manual mode includes approval instructions |
| TC-014 | `TestManualModeLoopWaitsForApproval` | `cmd/dun/main_test.go` | Verify loop with manual mode does not auto-proceed |

### 4.4 Yolo Mode Behavior Tests (AC-4)

| Test ID | Test Name | Location | Description |
|---------|-----------|----------|-------------|
| TC-015 | `TestYoloModeExecutesWithoutPrompt` | `cmd/dun/main_test.go` | Verify agent checks execute immediately in yolo mode |
| TC-016 | `TestYoloModeAllowsArtifactCreation` | `internal/dun/agent_test.go` | Verify yolo mode passes through to agent for artifact creation |
| TC-017 | `TestYoloModeInPromptTemplate` | `internal/dun/agent_test.go` | Verify `{{ .AutomationMode }}` renders as `yolo` (already exists, can expand) |
| TC-018 | `TestYoloModeHarnessFlags` | `cmd/dun/main_test.go` | Verify harness receives correct flags for yolo mode |

### 4.5 Integration Tests

| Test ID | Test Name | Location | Description |
|---------|-----------|----------|-------------|
| TC-019 | `TestAutomationModeE2EManual` | `cmd/dun/main_test.go` | End-to-end: config manual -> check -> prompt returned |
| TC-020 | `TestAutomationModeE2EYolo` | `cmd/dun/main_test.go` | End-to-end: config yolo -> check -> agent executes |

---

## 5. Priority Matrix

| Priority | Test IDs | Rationale |
|----------|----------|-----------|
| High | TC-001, TC-003, TC-007, TC-011, TC-015 | Core functionality for each mode |
| Medium | TC-002, TC-004, TC-005, TC-006, TC-008, TC-012, TC-016 | Complete flag/config coverage |
| Low | TC-009, TC-010, TC-013, TC-014, TC-017, TC-018, TC-019, TC-020 | Edge cases and integration |

---

## 6. Test Implementation Notes

### 6.1 Existing Helper Functions

The following helpers in `main_test.go` can be reused:
- `setupEmptyRepo(t)` - Creates temp git repo
- `runInDirWithWriters(t, dir, args, stdout, stderr)` - Runs CLI with custom writers
- `writeConfig(t, root, agentCmd)` - Writes config file (needs extension for automation modes)

### 6.2 Suggested Helper Extension

```go
func writeConfigWithAutomation(t *testing.T, root, agentCmd, automation string) {
    t.Helper()
    path := filepath.Join(root, ".dun", "config.yaml")
    if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
        t.Fatalf("mkdir config dir: %v", err)
    }
    content := fmt.Sprintf(`version: "1"
agent:
  automation: %s
  mode: auto
  timeout_ms: 5000
  cmd: "%s"
`, automation, agentCmd)
    if err := os.WriteFile(path, []byte(content), 0644); err != nil {
        t.Fatalf("write config: %v", err)
    }
}
```

### 6.3 Mock Requirements

For TC-011 through TC-014 (manual mode), tests will need to verify that:
- The agent command is NOT invoked when mode is manual
- The check result status is `prompt` instead of `pass`/`fail`

This may require extending the existing `checkRepo` mock pattern.

---

## 7. Definition of Done

- [ ] All High priority tests implemented and passing
- [ ] All Medium priority tests implemented and passing
- [ ] Test coverage for automation slider functionality exceeds 80%
- [ ] No regressions in existing tests
- [ ] Tests documented with clear descriptions
