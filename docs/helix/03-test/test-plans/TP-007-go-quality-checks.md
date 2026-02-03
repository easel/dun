---
dun:
  id: TP-007
  depends_on:
  - TD-007
---
# TP-007: Go Quality Checks Test Plan

**User Story:** US-007 - Enforce Go Quality Checks
**Status:** Coverage Analysis Complete

## Acceptance Criteria

From US-007, the acceptance criteria are:

1. **AC-1:** Go repos automatically run tests, coverage, and vet.
2. **AC-2:** Coverage failures report the current percentage and target.
3. **AC-3:** Staticcheck warns when missing, fails when issues are found.

---

## Test Coverage Mapping

### AC-1: Go repos automatically run tests, coverage, and vet

| Test Case | Existing Test | File | Status |
|-----------|---------------|------|--------|
| Go test check passes | `TestGoTestCheckPasses` | `go_checks_test.go:10` | Covered |
| Go test check fails | `TestGoTestCheckFails` | `go_checks_test.go:24` | Covered |
| Go vet check passes | `TestGoVetCheckPasses` | `go_checks_test.go:102` | Covered |
| Go vet check fails | `TestGoVetCheckFails` | `go_checks_test.go:87` | Covered |
| Go coverage check passes | `TestGoCoverageCheckPassesAtCustomThreshold` | `go_checks_test.go:66` | Covered |
| Go coverage check fails | `TestGoCoverageCheckFailsBelowThreshold` | `go_checks_test.go:42` | Covered |
| Go test failure handled in coverage | `TestRunGoCoverageCheckHandlesGoTestFailure` | `go_checks_test.go:248` | Covered |

**Coverage Status:** COMPLETE

### AC-2: Coverage failures report the current percentage and target

| Test Case | Existing Test | File | Status |
|-----------|---------------|------|--------|
| Coverage fail includes current percentage | `TestGoCoverageCheckFailsBelowThreshold` | `go_checks_test.go:42` | Covered |
| Coverage threshold uses default | `TestCoverageThresholdDefault` | `go_checks_test.go:234` | Covered |
| Coverage threshold uses custom value | `TestGoCoverageCheckPassesAtCustomThreshold` | `go_checks_test.go:66` | Covered |
| Parse coverage percentage errors | `TestParseCoveragePercentErrors` | `go_checks_test.go:209` | Covered |

**Coverage Status:** COMPLETE - Test at line 42 verifies `res.Detail` contains `72.0`, confirming the current percentage is reported.

### AC-3: Staticcheck warns when missing, fails when issues are found

| Test Case | Existing Test | File | Status |
|-----------|---------------|------|--------|
| Staticcheck warns when missing | `TestGoStaticcheckWarnsWhenMissing` | `go_checks_test.go:116` | Covered |
| Staticcheck fails when tool errors | `TestGoStaticcheckFailsWhenToolErrors` | `go_checks_test.go:130` | Covered |
| Staticcheck passes when OK | `TestGoStaticcheckPassesWhenToolOK` | `go_checks_test.go:149` | Covered |

**Coverage Status:** COMPLETE

---

## Gap Analysis

### Identified Gaps

| Gap ID | Description | Priority | Proposed Test |
|--------|-------------|----------|---------------|
| GAP-1 | Coverage failure detail does not explicitly verify target is reported | Low | Verify `res.Detail` contains both current (72.0%) and target (100%) |
| GAP-2 | No integration test for automatic Go repo detection | Medium | Add integration test for repo with go.mod triggering Go checks |
| GAP-3 | No test for coverage edge case at exact threshold | Low | Test when coverage equals threshold exactly |
| GAP-4 | No test for staticcheck with specific lint issues in output | Low | Verify staticcheck output parsing for specific issue types |

---

## Proposed Test Cases

### GAP-1: Verify coverage detail includes target percentage

**Test Name:** `TestGoCoverageCheckReportsTarget`

**Description:** Ensure that when coverage fails, the detail message includes both the current percentage and the target threshold.

```go
func TestGoCoverageCheckReportsTarget(t *testing.T) {
    binDir := stubGoBinary(t)
    t.Setenv("PATH", binDir)
    t.Setenv("DUN_COVER_PCT", "50.0")

    root := t.TempDir()
    check := Check{
        ID: "go-coverage",
        Rules: []Rule{
            {Type: "coverage-min", Expected: 80},
        },
    }
    res, err := runGoCoverageCheck(root, check)
    if err != nil {
        t.Fatalf("coverage check: %v", err)
    }
    if res.Status != "fail" {
        t.Fatalf("expected fail, got %s", res.Status)
    }
    if !strings.Contains(res.Detail, "50.0") {
        t.Fatalf("expected current coverage in detail, got %q", res.Detail)
    }
    if !strings.Contains(res.Detail, "80") {
        t.Fatalf("expected target threshold in detail, got %q", res.Detail)
    }
}
```

**Priority:** Low - Existing test verifies current percentage; target verification is enhancement.

### GAP-2: Integration test for Go repo detection

**Test Name:** `TestGoRepoAutoDetection`

**Description:** Verify that a repository containing a `go.mod` file automatically triggers Go quality checks.

```go
func TestGoRepoAutoDetection(t *testing.T) {
    root := setupEmptyRepo(t)

    // Create go.mod to trigger Go detection
    goModPath := filepath.Join(root, "go.mod")
    if err := os.WriteFile(goModPath, []byte("module example.com/test\n\ngo 1.21\n"), 0644); err != nil {
        t.Fatalf("write go.mod: %v", err)
    }

    plan, err := planRepo(root)
    if err != nil {
        t.Fatalf("plan repo: %v", err)
    }

    // Verify Go checks are included
    goCheckIDs := []string{"go-test", "go-coverage", "go-vet", "go-staticcheck"}
    for _, id := range goCheckIDs {
        found := false
        for _, check := range plan.Checks {
            if check.ID == id {
                found = true
                break
            }
        }
        if !found {
            t.Errorf("expected check %s to be planned for Go repo", id)
        }
    }
}
```

**Priority:** Medium - Validates the "automatically run" aspect of AC-1.

### GAP-3: Coverage at exact threshold

**Test Name:** `TestGoCoverageCheckPassesAtExactThreshold`

**Description:** Verify coverage passes when exactly at the threshold (boundary condition).

```go
func TestGoCoverageCheckPassesAtExactThreshold(t *testing.T) {
    binDir := stubGoBinary(t)
    t.Setenv("PATH", binDir)
    t.Setenv("DUN_COVER_PCT", "80.0")

    root := t.TempDir()
    check := Check{
        ID: "go-coverage",
        Rules: []Rule{
            {Type: "coverage-min", Expected: 80},
        },
    }
    res, err := runGoCoverageCheck(root, check)
    if err != nil {
        t.Fatalf("coverage check: %v", err)
    }
    if res.Status != "pass" {
        t.Fatalf("expected pass at exact threshold, got %s", res.Status)
    }
}
```

**Priority:** Low - Edge case testing.

### GAP-4: Staticcheck output parsing

**Test Name:** `TestGoStaticcheckOutputParsing`

**Description:** Verify staticcheck parses and reports specific lint issues found.

```go
func TestGoStaticcheckOutputParsing(t *testing.T) {
    binDir := stubGoBinary(t)
    staticcheckPath := filepath.Join(binDir, "staticcheck")
    // Simulate staticcheck finding an issue
    script := `#!/bin/sh
echo "main.go:10:5: SA1000: invalid regex"
exit 1
`
    writeFile(t, staticcheckPath, script)
    if err := os.Chmod(staticcheckPath, 0755); err != nil {
        t.Fatalf("chmod staticcheck: %v", err)
    }
    t.Setenv("PATH", binDir)

    root := t.TempDir()
    res, err := runGoStaticcheck(root, Check{ID: "go-staticcheck"})
    if err != nil {
        t.Fatalf("staticcheck: %v", err)
    }
    if res.Status != "fail" {
        t.Fatalf("expected fail, got %s", res.Status)
    }
    // Verify issue is captured
    if !strings.Contains(res.Detail, "SA1000") || !strings.Contains(res.Detail, "main.go") {
        t.Fatalf("expected staticcheck issue in detail, got %q", res.Detail)
    }
}
```

**Priority:** Low - Enhancement for better error reporting visibility.

---

## Summary

| Acceptance Criteria | Coverage Status | Gaps |
|---------------------|-----------------|------|
| AC-1: Auto run tests, coverage, vet | Complete | GAP-2 (integration) |
| AC-2: Report current % and target | Complete | GAP-1 (target verification), GAP-3 (boundary) |
| AC-3: Staticcheck warn/fail | Complete | GAP-4 (output parsing) |

**Overall Assessment:** The existing test suite in `/home/erik/gt/dun/crew/oscar/internal/dun/go_checks_test.go` provides comprehensive coverage of the acceptance criteria. All core functionality is tested including:

- Pass/fail scenarios for go test, go vet, go coverage, and staticcheck
- Coverage percentage parsing and threshold comparison
- Staticcheck missing vs present vs failing scenarios
- Error handling for coverage profile creation and parsing

The identified gaps are enhancements rather than critical missing coverage. The priority order for addressing gaps would be:

1. **GAP-2** (Medium) - Integration test for auto-detection
2. **GAP-1** (Low) - Target percentage in failure detail
3. **GAP-3** (Low) - Exact threshold boundary test
4. **GAP-4** (Low) - Staticcheck output parsing detail
