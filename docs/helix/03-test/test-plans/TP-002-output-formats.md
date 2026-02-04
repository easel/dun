---
dun:
  id: TP-002
  depends_on:
  - TD-002
---
# TP-002: Output Formats Test Plan

Test plan for US-002: Emit Output Formats for Agents and Tools.

## Acceptance Criteria

| ID | Criterion | Status |
|----|-----------|--------|
| AC-1 | `dun check` emits prompt envelopes by default when agent checks are present (prompt payloads omitted) | Partially Covered |
| AC-2 | `dun check --format=llm` prints concise summaries for humans | Covered |
| AC-3 | `dun check --format=json` emits structured JSON output | Covered |
| AC-4 | Output is deterministic for a given repo state | Gap |

## Coverage Mapping

### AC-1: Prompt Envelope Default Output

**Existing Tests:**

| Test | File | Coverage |
|------|------|----------|
| `TestHelixMissingArchitecturePromptsAgent` | `internal/dun/engine_test.go` | Verifies prompt status and envelope structure |
| `TestHelixMissingFeaturesEmitsPrompt` | `internal/dun/engine_test.go` | Verifies prompt emission |
| `TestHelixAlignmentEmitsPrompt` | `internal/dun/engine_test.go` | Verifies prompt emission |
| `TestRunCheckAgentPrompt` | `internal/dun/engine_extra_test.go` | Verifies agent check returns prompt status |
| `TestCheckUsesConfigAgentAuto` | `cmd/dun/main_test.go` | Verifies agent response overwrites prompt |

**Gaps:**

- No test verifies the default format is "prompt" when running `dun check` without flags
- No test verifies prompt envelope `kind` field is always `dun.prompt.v1`
- No test verifies callback command format is correct across all agent checks
- No test verifies prompt envelope contains required fields (id, prompt, callback)
- No test verifies prompt payloads are omitted from `dun check` output

### AC-2: LLM Format Output

**Existing Tests:**

| Test | File | Coverage |
|------|------|----------|
| `TestRunCheckLLMOutput` | `cmd/dun/main_test.go` | Verifies `--format=llm` produces `check:` prefixed output |
| `TestPrintLLM` | `cmd/dun/main_test.go` | Verifies LLM output format includes check id, issues |
| `TestRunRespondVariants` | `cmd/dun/main_test.go` | Verifies respond command supports LLM format |

**Gaps:**

- No test verifies all check statuses are correctly formatted (pass, fail, warn, skip, error, prompt)
- No test verifies signal, detail, and next fields are properly included
- No test verifies issue formatting with and without paths
- No test verifies multi-check output ordering

### AC-3: JSON Format Output

**Existing Tests:**

| Test | File | Coverage |
|------|------|----------|
| `TestCheckUsesConfigAgentAuto` | `cmd/dun/main_test.go` | Parses JSON output to `dun.Result` |
| `TestCheckResolvesRepoRootFromSubdir` | `cmd/dun/main_test.go` | Parses JSON output |
| `TestRunCheckJSONEncodeError` | `cmd/dun/main_test.go` | Verifies JSON encode error handling |
| `TestRunListTextAndJSON` | `cmd/dun/main_test.go` | Verifies list JSON format |
| `TestRunExplainJSON` | `cmd/dun/main_test.go` | Verifies explain JSON format |
| `TestRunRespondJSONEncodeError` | `cmd/dun/main_test.go` | Verifies respond JSON error handling |

**Gaps:**

- No test explicitly validates JSON schema structure
- No test verifies JSON output is valid/parseable for all status types
- No test verifies optional fields (detail, next, prompt, issues) are correctly omitted when empty
- No test verifies JSON output with multiple checks

### AC-4: Deterministic Output

**Existing Tests:**

None.

**Gaps:**

- No test runs same check twice and compares output
- No test verifies check ordering is consistent
- No test verifies issue ordering within checks is consistent
- No test verifies timestamps or other non-deterministic fields are absent

## Proposed Test Cases

### TC-001: Default Format is Prompt

**File:** `cmd/dun/main_test.go`

```go
func TestCheckDefaultFormatIsPrompt(t *testing.T) {
    root := setupRepoFromFixture(t, "helix-missing-architecture")
    var stdout bytes.Buffer
    var stderr bytes.Buffer

    // Run without --format flag
    code := runInDirWithWriters(t, root, []string{"check"}, &stdout, &stderr)
    if code != 0 {
        t.Fatalf("expected success, got %d", code)
    }

    // Default output should be parseable as JSON with prompt envelope placeholder
    var result dun.Result
    if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
        t.Fatalf("expected JSON output by default: %v", err)
    }

    // Find agent check and verify it has prompt
    check := findCheck(t, result, "helix-create-architecture")
    if check.Status != "prompt" {
        t.Fatalf("expected prompt status, got %s", check.Status)
    }
    if check.Prompt == nil {
        t.Fatalf("expected prompt envelope in default output")
    }
    if !strings.Contains(check.Prompt.Prompt, "Prompt omitted") {
        t.Fatalf("expected compact prompt placeholder, not full prompt payload")
    }
}
```

### TC-002: Prompt Envelope Structure Validation

**File:** `internal/dun/engine_test.go`

```go
func TestPromptEnvelopeStructure(t *testing.T) {
    result := runFixture(t, "helix-missing-architecture", "")
    check := findCheck(t, result, "helix-create-architecture")

    if check.Prompt == nil {
        t.Fatalf("expected prompt envelope")
    }

    // Required fields
    if check.Prompt.Kind != "dun.prompt.v1" {
        t.Fatalf("expected kind dun.prompt.v1, got %s", check.Prompt.Kind)
    }
    if check.Prompt.ID == "" {
        t.Fatalf("expected prompt ID")
    }
    if check.Prompt.Prompt == "" {
        t.Fatalf("expected prompt text")
    }
    if check.Prompt.Callback.Command == "" {
        t.Fatalf("expected callback command")
    }

    // Callback format
    expectedPrefix := "dun respond --id " + check.ID
    if !strings.HasPrefix(check.Prompt.Callback.Command, expectedPrefix) {
        t.Fatalf("callback command should start with %q, got %q",
            expectedPrefix, check.Prompt.Callback.Command)
    }
}
```

### TC-003: LLM Format All Status Types

**File:** `cmd/dun/main_test.go`

```go
func TestLLMFormatAllStatusTypes(t *testing.T) {
    orig := checkRepo
    checkRepo = func(_ string, _ dun.Options) (dun.Result, error) {
        return dun.Result{
            Checks: []dun.CheckResult{
                {ID: "pass-check", Status: "pass", Signal: "ok"},
                {ID: "fail-check", Status: "fail", Signal: "failed", Detail: "detail", Next: "fix"},
                {ID: "warn-check", Status: "warn", Signal: "warning"},
                {ID: "skip-check", Status: "skip", Signal: "skipped"},
                {ID: "error-check", Status: "error", Signal: "error"},
                {ID: "prompt-check", Status: "prompt", Signal: "needs input"},
            },
        }, nil
    }
    t.Cleanup(func() { checkRepo = orig })

    root := setupEmptyRepo(t)
    var stdout bytes.Buffer
    var stderr bytes.Buffer
    code := runInDirWithWriters(t, root, []string{"check", "--format=llm"}, &stdout, &stderr)
    if code != 0 {
        t.Fatalf("expected success, got %d", code)
    }

    output := stdout.String()
    for _, id := range []string{"pass-check", "fail-check", "warn-check", "skip-check", "error-check", "prompt-check"} {
        if !strings.Contains(output, "check:"+id) {
            t.Fatalf("expected %s in LLM output", id)
        }
    }
}
```

### TC-004: LLM Format Issue Display

**File:** `cmd/dun/main_test.go`

```go
func TestLLMFormatIssueDisplay(t *testing.T) {
    orig := checkRepo
    checkRepo = func(_ string, _ dun.Options) (dun.Result, error) {
        return dun.Result{
            Checks: []dun.CheckResult{
                {
                    ID: "check-with-issues",
                    Status: "fail",
                    Signal: "failed",
                    Issues: []dun.Issue{
                        {Summary: "Issue with path", Path: "src/file.go"},
                        {Summary: "Issue without path"},
                    },
                },
            },
        }, nil
    }
    t.Cleanup(func() { checkRepo = orig })

    root := setupEmptyRepo(t)
    var stdout bytes.Buffer
    var stderr bytes.Buffer
    code := runInDirWithWriters(t, root, []string{"check", "--format=llm"}, &stdout, &stderr)
    if code != 0 {
        t.Fatalf("expected success, got %d", code)
    }

    output := stdout.String()
    if !strings.Contains(output, "Issue with path (src/file.go)") {
        t.Fatalf("expected issue with path formatted correctly")
    }
    if !strings.Contains(output, "issue: Issue without path") {
        t.Fatalf("expected issue without path")
    }
}
```

### TC-005: JSON Schema Validation

**File:** `cmd/dun/main_test.go`

```go
func TestJSONSchemaValidation(t *testing.T) {
    orig := checkRepo
    checkRepo = func(_ string, _ dun.Options) (dun.Result, error) {
        return dun.Result{
            Checks: []dun.CheckResult{
                {
                    ID: "full-check",
                    Status: "fail",
                    Signal: "failed",
                    Detail: "some detail",
                    Next: "next action",
                    Issues: []dun.Issue{{Summary: "issue", Path: "path"}},
                },
                {
                    ID: "minimal-check",
                    Status: "pass",
                    Signal: "ok",
                    // No optional fields
                },
            },
        }, nil
    }
    t.Cleanup(func() { checkRepo = orig })

    root := setupEmptyRepo(t)
    var stdout bytes.Buffer
    var stderr bytes.Buffer
    code := runInDirWithWriters(t, root, []string{"check", "--format=json"}, &stdout, &stderr)
    if code != 0 {
        t.Fatalf("expected success, got %d", code)
    }

    var result dun.Result
    if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
        t.Fatalf("JSON should be valid: %v", err)
    }

    if len(result.Checks) != 2 {
        t.Fatalf("expected 2 checks, got %d", len(result.Checks))
    }

    // Verify full check has all fields
    full := result.Checks[0]
    if full.Detail == "" || full.Next == "" || len(full.Issues) == 0 {
        t.Fatalf("expected full check to have all optional fields")
    }

    // Verify minimal check has empty optional fields
    minimal := result.Checks[1]
    if minimal.Detail != "" || minimal.Next != "" || len(minimal.Issues) != 0 {
        t.Fatalf("expected minimal check to have empty optional fields")
    }
}
```

### TC-006: Deterministic Output

**File:** `cmd/dun/main_test.go`

```go
func TestOutputDeterminism(t *testing.T) {
    root := setupRepoFromFixture(t, "helix-alignment")
    agentCmd := "bash " + fixturePath(t, "internal/testdata/agent/agent.sh")
    writeConfig(t, root, agentCmd)

    // Run check multiple times
    outputs := make([]string, 3)
    for i := 0; i < 3; i++ {
        var stdout bytes.Buffer
        var stderr bytes.Buffer
        code := runInDirWithWriters(t, root, []string{"check", "--format=json"}, &stdout, &stderr)
        if code != 0 {
            t.Fatalf("run %d: expected success, got %d", i, code)
        }
        outputs[i] = stdout.String()
    }

    // All outputs should be identical
    for i := 1; i < len(outputs); i++ {
        if outputs[i] != outputs[0] {
            t.Fatalf("output %d differs from output 0:\n%s\nvs\n%s", i, outputs[0], outputs[i])
        }
    }
}
```

### TC-007: Check Ordering Consistency

**File:** `cmd/dun/main_test.go`

```go
func TestCheckOrderingConsistency(t *testing.T) {
    root := setupEmptyRepo(t)

    // Run multiple times and verify order
    var prevOrder []string
    for i := 0; i < 3; i++ {
        output := runInDir(t, root, []string{"check"})
        var result dun.Result
        if err := json.Unmarshal(output, &result); err != nil {
            t.Fatalf("decode: %v", err)
        }

        var order []string
        for _, check := range result.Checks {
            order = append(order, check.ID)
        }

        if prevOrder != nil {
            for j, id := range order {
                if prevOrder[j] != id {
                    t.Fatalf("check order changed at position %d: %s vs %s", j, prevOrder[j], id)
                }
            }
        }
        prevOrder = order
    }
}
```

### TC-008: Unknown Format Error

**File:** `cmd/dun/main_test.go`

**Status:** Already covered by `TestRunCheckUnknownFormat`

### TC-009: JSON Omits Empty Optional Fields

**File:** `cmd/dun/main_test.go`

```go
func TestJSONOmitsEmptyOptionalFields(t *testing.T) {
    orig := checkRepo
    checkRepo = func(_ string, _ dun.Options) (dun.Result, error) {
        return dun.Result{
            Checks: []dun.CheckResult{
                {ID: "minimal", Status: "pass", Signal: "ok"},
            },
        }, nil
    }
    t.Cleanup(func() { checkRepo = orig })

    root := setupEmptyRepo(t)
    var stdout bytes.Buffer
    var stderr bytes.Buffer
    code := runInDirWithWriters(t, root, []string{"check", "--format=json"}, &stdout, &stderr)
    if code != 0 {
        t.Fatalf("expected success, got %d", code)
    }

    output := stdout.String()

    // These fields should not appear when empty (omitempty)
    if strings.Contains(output, "\"detail\"") {
        t.Fatalf("empty detail should be omitted from JSON")
    }
    if strings.Contains(output, "\"next\"") {
        t.Fatalf("empty next should be omitted from JSON")
    }
    if strings.Contains(output, "\"prompt\"") {
        t.Fatalf("nil prompt should be omitted from JSON")
    }
    if strings.Contains(output, "\"issues\"") {
        t.Fatalf("empty issues should be omitted from JSON")
    }
}
```

## Summary

| Criterion | Existing Tests | Proposed Tests | Priority |
|-----------|---------------|----------------|----------|
| AC-1 | 5 partial | TC-001, TC-002 | High |
| AC-2 | 3 partial | TC-003, TC-004 | Medium |
| AC-3 | 6 partial | TC-005, TC-009 | Medium |
| AC-4 | 0 | TC-006, TC-007 | High |

**Total Gaps:** 8 test cases proposed

## Implementation Priority

1. **High Priority (AC-4):** TC-006, TC-007 - Determinism is untested
2. **High Priority (AC-1):** TC-001, TC-002 - Default format behavior needs explicit verification
3. **Medium Priority (AC-2):** TC-003, TC-004 - LLM format edge cases
4. **Medium Priority (AC-3):** TC-005, TC-009 - JSON schema completeness
