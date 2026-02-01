# TP-009: Autonomous Iteration Test Plan

**User Story:** US-009 - Run Autonomous Iteration Loop
**Status:** Draft
**Last Updated:** 2026-01-30

## Overview

This test plan covers the `dun iterate` and `dun loop` commands that enable autonomous agent-driven iteration for resolving quality issues.

## Acceptance Criteria Coverage

### AC-1: `dun iterate` outputs a work list prompt for an external agent

| Test Case | Existing Test | Status |
|-----------|---------------|--------|
| Outputs structured prompt with work items | `TestRunIterateWithActionable` | Covered |
| Includes priority labels (HIGH/MEDIUM/LOW) | `TestRunIterateWithActionable` | Covered |
| Shows ALL_PASS status when no work | `TestRunIterateAllPass` | Covered |
| Handles parse errors | `TestRunIterateParseError` | Covered |
| Handles config errors | `TestRunIterateConfigError` | Covered |
| Handles check errors | `TestRunIterateCheckError` | Covered |
| Includes instructions section | `TestPrintIteratePromptVariants` | Covered |
| Shows working directory | `TestPrintIteratePromptVariants` | Covered |
| Shows automation mode | `TestPrintIteratePromptVariants` | Covered |

### AC-2: `dun loop` runs an embedded loop calling a configurable agent harness

| Test Case | Existing Test | Status |
|-----------|---------------|--------|
| Executes loop with harness calls | `TestRunLoopMaxIterations` | Covered |
| Handles parse errors | `TestRunLoopParseError` | Covered |
| Handles config errors | `TestRunLoopConfigError` | Covered |
| Handles check errors | `TestRunLoopCheckError` | Covered |
| Works with valid config file | `TestRunLoopWithConfig` | Covered |

### AC-3: The loop supports multiple harnesses: claude, gemini, codex

| Test Case | Existing Test | Status |
|-----------|---------------|--------|
| Claude harness recognized | `TestCallHarnessClaude` | Covered |
| Gemini harness recognized | `TestCallHarnessGemini` | Covered |
| Codex harness recognized | `TestCallHarnessCodex` | Covered |
| Unknown harness rejected | `TestCallHarnessUnknown` | Covered |
| Claude yolo mode flags | `TestCallHarnessClaudeYolo` | Covered |
| Codex yolo mode flags | `TestCallHarnessCodexYolo` | Covered |
| Gemini yolo mode flags | - | **GAP** |

### AC-4: Each iteration spawns fresh context to prevent drift

| Test Case | Existing Test | Status |
|-----------|---------------|--------|
| Each iteration runs checkRepo fresh | `TestRunLoopMaxIterations` | Partial |
| Context isolation between iterations | - | **GAP** |
| State not carried between harness calls | - | **GAP** |

### AC-5: The loop exits when all checks pass or max iterations is reached

| Test Case | Existing Test | Status |
|-----------|---------------|--------|
| Exits on all pass | `TestRunLoopAllPass` | Covered |
| Exits on max iterations | `TestRunLoopMaxIterations` | Covered |
| Exits on EXIT_SIGNAL from agent | `TestRunLoopExitSignal` | Covered |
| Continues on harness error | `TestRunLoopHarnessError` | Covered |

### AC-6: Yolo mode passes appropriate flags to the harness for autonomous operation

| Test Case | Existing Test | Status |
|-----------|---------------|--------|
| Claude: --dangerously-skip-permissions | `TestCallHarnessClaudeYolo` | Covered |
| Codex: --full-auto | `TestCallHarnessCodexYolo` | Covered |
| Gemini: appropriate API config | - | **GAP** |
| Iterate respects automation flag | `TestRunIterateWithActionable` | Covered |

### AC-7: `dun install` adds agent documentation to AGENTS.md explaining the pattern

| Test Case | Existing Test | Status |
|-----------|---------------|--------|
| Install outputs installed files | `TestRunInstallOutputsInstalled` | Covered |
| Install dry-run mode | `TestRunInstallDryRunAndError` | Covered |
| AGENTS.md content includes iterate/loop | - | **GAP** |

### AC-8: `dun help` documents the iterate and loop commands

| Test Case | Existing Test | Status |
|-----------|---------------|--------|
| Help includes iterate command | - | **GAP** |
| Help includes loop command | - | **GAP** |
| Help includes loop options | - | **GAP** |
| Help includes examples | - | **GAP** |

## Identified Gaps

### Gap 1: Gemini yolo mode testing
**Priority:** Low
**Description:** No test verifies Gemini-specific yolo mode behavior.
**Proposed Test:**
```go
func TestCallHarnessGeminiYolo(t *testing.T) {
    // Verify Gemini API configuration in yolo mode
    // Currently Gemini doesn't have a yolo-specific config
    _, err := callHarness("gemini", "test prompt", "yolo")
    if err == nil {
        return
    }
    if strings.Contains(err.Error(), "unknown harness") {
        t.Fatalf("gemini should be a known harness")
    }
}
```

### Gap 2: Context isolation verification
**Priority:** Medium
**Description:** No test explicitly verifies that each iteration gets fresh context without state leakage.
**Proposed Test:**
```go
func TestRunLoopFreshContextPerIteration(t *testing.T) {
    root := setupEmptyRepo(t)
    var callOrder []int
    iterationCount := 0

    origCheck := checkRepo
    checkRepo = func(_ string, opts dun.Options) (dun.Result, error) {
        iterationCount++
        callOrder = append(callOrder, iterationCount)
        if iterationCount >= 2 {
            return dun.Result{
                Checks: []dun.CheckResult{{ID: "pass", Status: "pass"}},
            }, nil
        }
        return dun.Result{
            Checks: []dun.CheckResult{{ID: "fail", Status: "fail"}},
        }, nil
    }
    t.Cleanup(func() { checkRepo = origCheck })

    origHarness := callHarnessFn
    callHarnessFn = func(harness, prompt, automation string) (string, error) {
        // Verify prompt doesn't contain state from previous iterations
        return "done", nil
    }
    t.Cleanup(func() { callHarnessFn = origHarness })

    var stdout, stderr bytes.Buffer
    code := runInDirWithWriters(t, root, []string{"loop", "--max-iterations", "3"}, &stdout, &stderr)
    if code != dun.ExitSuccess {
        t.Fatalf("expected success, got %d", code)
    }
    if iterationCount < 2 {
        t.Fatalf("expected at least 2 iterations, got %d", iterationCount)
    }
}
```

### Gap 3: Help command coverage for iterate/loop
**Priority:** High
**Description:** No tests verify the help output includes iterate and loop documentation.
**Proposed Tests:**
```go
func TestRunHelpIncludesIterate(t *testing.T) {
    var stdout bytes.Buffer
    var stderr bytes.Buffer
    code := run([]string{"help"}, &stdout, &stderr)
    if code != dun.ExitSuccess {
        t.Fatalf("expected success, got %d", code)
    }
    output := stdout.String()
    if !strings.Contains(output, "iterate") {
        t.Fatalf("help should document iterate command")
    }
    if !strings.Contains(output, "dun iterate") {
        t.Fatalf("help should show iterate usage")
    }
}

func TestRunHelpIncludesLoop(t *testing.T) {
    var stdout bytes.Buffer
    var stderr bytes.Buffer
    code := run([]string{"help"}, &stdout, &stderr)
    if code != dun.ExitSuccess {
        t.Fatalf("expected success, got %d", code)
    }
    output := stdout.String()
    if !strings.Contains(output, "loop") {
        t.Fatalf("help should document loop command")
    }
    if !strings.Contains(output, "--harness") {
        t.Fatalf("help should document harness option")
    }
    if !strings.Contains(output, "--max-iterations") {
        t.Fatalf("help should document max-iterations option")
    }
    if !strings.Contains(output, "claude, gemini, codex") {
        t.Fatalf("help should list available harnesses")
    }
}

func TestRunHelpIncludesExamples(t *testing.T) {
    var stdout bytes.Buffer
    var stderr bytes.Buffer
    code := run([]string{"help"}, &stdout, &stderr)
    if code != dun.ExitSuccess {
        t.Fatalf("expected success, got %d", code)
    }
    output := stdout.String()
    if !strings.Contains(output, "dun loop") {
        t.Fatalf("help should include loop examples")
    }
    if !strings.Contains(output, "--dry-run") {
        t.Fatalf("help should document dry-run option")
    }
}
```

### Gap 4: AGENTS.md content verification
**Priority:** Medium
**Description:** No test verifies that `dun install` creates AGENTS.md with iterate/loop documentation.
**Proposed Test:**
```go
func TestInstallCreatesAgentsMDWithLoopDocs(t *testing.T) {
    root := setupEmptyRepo(t)
    var stdout, stderr bytes.Buffer
    code := runInDirWithWriters(t, root, []string{"install"}, &stdout, &stderr)
    if code != dun.ExitSuccess {
        t.Fatalf("expected success, got %d", code)
    }

    agentsMD := filepath.Join(root, "AGENTS.md")
    content, err := os.ReadFile(agentsMD)
    if err != nil {
        t.Fatalf("read AGENTS.md: %v", err)
    }

    if !strings.Contains(string(content), "dun iterate") {
        t.Fatalf("AGENTS.md should document iterate command")
    }
    if !strings.Contains(string(content), "dun loop") {
        t.Fatalf("AGENTS.md should document loop command")
    }
}
```

### Gap 5: Dry-run mode for iterate
**Priority:** Low
**Description:** No test for iterate with --dry-run or similar preview behavior.
**Note:** Currently iterate always outputs the prompt, so dry-run may not be needed. Consider if a --format flag would be useful.

### Gap 6: Integration test with real harness
**Priority:** Low (CI limitation)
**Description:** Current tests mock the harness. Real integration tests would require harness binaries.
**Note:** Consider adding skip conditions for CI or using lightweight test harness scripts.

## Test Matrix

| Feature | Unit | Integration | E2E |
|---------|------|-------------|-----|
| iterate output format | Yes | - | - |
| iterate all-pass detection | Yes | - | - |
| iterate priority sorting | Yes | - | - |
| loop harness selection | Yes | - | - |
| loop exit conditions | Yes | - | - |
| loop fresh context | Partial | **Needed** | - |
| harness command construction | Yes | - | - |
| help documentation | **Needed** | - | - |
| install AGENTS.md | Partial | **Needed** | - |

## Recommended Priority Order

1. **High Priority (Add immediately)**
   - `TestRunHelpIncludesIterate`
   - `TestRunHelpIncludesLoop`
   - `TestRunHelpIncludesExamples`

2. **Medium Priority (Add before release)**
   - `TestRunLoopFreshContextPerIteration`
   - `TestInstallCreatesAgentsMDWithLoopDocs`

3. **Low Priority (Nice to have)**
   - `TestCallHarnessGeminiYolo`
   - Integration tests with test harness scripts

## Related Documents

- [US-009: Autonomous Iteration](../../01-frame/user-stories/US-009-autonomous-iteration.md)
- [SPIKE-001: Nested Agent Harness](../../01-frame/spikes/SPIKE-001-nested-agent-harness.md)
