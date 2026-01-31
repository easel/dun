# TP-005: Reconcile PRD Changes Through the Stack

**User Story**: US-005
**Version**: 1.0.0
**Date**: 2026-01-30
**Status**: Draft
**Author**: oscar

## 1. Acceptance Criteria

From US-005:

| ID | Acceptance Criterion |
|----|---------------------|
| AC-1 | When PRD changes, Dun emits a list of impacted artifacts in order |
| AC-2 | The plan includes updates for feature specs, design docs, ADRs, test plans, and implementation |
| AC-3 | The plan is structured and deterministic |

## 2. Existing Test Coverage

### 2.1 Tests Mapped to Acceptance Criteria

| AC | Existing Test | File | Status |
|----|---------------|------|--------|
| AC-1 | `TestHelixStateRulesDetectsMissingStory` | `internal/dun/engine_test.go` | Partial |
| AC-1 | `TestRunStateRulesPassAndFail` | `internal/dun/state_rules_test.go` | Partial |
| AC-2 | `TestPlanRepoIncludesHelixChecks` | `internal/dun/plan_test.go` | Partial |
| AC-2 | `TestRunGateCheckPassesWhenSatisfied` | `internal/dun/gates_test.go` | Partial |
| AC-2 | `TestRunGateCheckFailsWhenRequiredMissing` | `internal/dun/gates_test.go` | Partial |
| AC-3 | None | - | Missing |

### 2.2 Coverage Analysis

**AC-1: PRD changes emit impacted artifacts in order**
- `TestHelixStateRulesDetectsMissingStory` verifies that missing upstream artifacts (US-001) are detected
- `TestRunStateRulesPassAndFail` validates the state rules engine finds inconsistencies
- **Gap**: No test verifies that changing the PRD specifically triggers downstream impact detection
- **Gap**: No test verifies ordering of impacted artifacts

**AC-2: Plan includes updates for all artifact types**
- `TestPlanRepoIncludesHelixChecks` verifies helix checks are included in the plan
- Gate tests verify evidence paths for required documentation
- **Gap**: No test verifies that feature specs, design docs, ADRs, test plans, and implementation are all included
- **Gap**: No test verifies the reconciliation plan structure

**AC-3: Plan is structured and deterministic**
- No existing tests validate plan structure or determinism
- **Gap**: No test verifies plan output format
- **Gap**: No test verifies deterministic ordering across runs

## 3. Test Gaps

### 3.1 Critical Gaps (P0)

| Gap ID | Description | Priority | Acceptance Criteria |
|--------|-------------|----------|---------------------|
| GAP-001 | No test for PRD change detection | P0 | AC-1 |
| GAP-002 | No test for downstream impact ordering | P0 | AC-1 |
| GAP-003 | No test for complete artifact type coverage | P0 | AC-2 |
| GAP-004 | No test for plan determinism | P0 | AC-3 |

### 3.2 Secondary Gaps (P1)

| Gap ID | Description | Priority | Acceptance Criteria |
|--------|-------------|----------|---------------------|
| GAP-005 | No test for ADR inclusion in reconciliation | P1 | AC-2 |
| GAP-006 | No test for implementation artifact tracking | P1 | AC-2 |
| GAP-007 | No test for plan structure validation | P1 | AC-3 |

## 4. Proposed Test Cases

### 4.1 Unit Tests

#### TC-001: PRD Change Detection
**File**: `internal/dun/reconcile_test.go`
**Priority**: P0
**Covers**: AC-1, GAP-001

```go
func TestReconcilePRDChangeEmitsImpactedArtifacts(t *testing.T) {
    // Given: A repo with PRD and downstream artifacts
    // When: PRD content has changed (mocked via fixture or state)
    // Then: Dun emits a list of impacted artifacts
    // Assert: List includes feature specs, design docs, test plans
}
```

**Test Data**: Create fixture `internal/testdata/repos/helix-prd-changed/` with:
- `docs/helix/01-frame/prd.md` (modified)
- `docs/helix/01-frame/features/FEAT-001.md`
- `docs/helix/02-design/architecture.md`
- `docs/helix/03-test/test-plan.md`

#### TC-002: Downstream Impact Ordering
**File**: `internal/dun/reconcile_test.go`
**Priority**: P0
**Covers**: AC-1, GAP-002

```go
func TestReconcileImpactedArtifactsInOrder(t *testing.T) {
    // Given: A repo where PRD changed
    // When: Reconciliation plan is generated
    // Then: Artifacts are ordered: features -> design -> ADRs -> test plans -> implementation
    // Assert: Order is deterministic across multiple runs
}
```

#### TC-003: Complete Artifact Type Coverage
**File**: `internal/dun/reconcile_test.go`
**Priority**: P0
**Covers**: AC-2, GAP-003

```go
func TestReconcilePlanIncludesAllArtifactTypes(t *testing.T) {
    // Given: A repo with all artifact types
    // When: PRD changes and reconciliation runs
    // Then: Plan includes:
    //   - Feature specs (docs/helix/01-frame/features/)
    //   - Design docs (docs/helix/02-design/)
    //   - ADRs (docs/helix/02-design/decisions/)
    //   - Test plans (docs/helix/03-test/)
    //   - Implementation markers (docs/helix/04-build/)
}
```

**Test Data**: Create fixture `internal/testdata/repos/helix-full-stack/` with complete artifact hierarchy

#### TC-004: Plan Determinism
**File**: `internal/dun/reconcile_test.go`
**Priority**: P0
**Covers**: AC-3, GAP-004

```go
func TestReconcilePlanIsDeterministic(t *testing.T) {
    // Given: Same repo state
    // When: Plan is generated multiple times
    // Then: Output is byte-for-byte identical
    // Assert: Run 10 times and compare outputs
}
```

### 4.2 Integration Tests

#### TC-005: End-to-End PRD Reconciliation
**File**: `internal/dun/engine_test.go`
**Priority**: P0
**Covers**: AC-1, AC-2, AC-3

```go
func TestHelixPRDReconciliationEmitsOrderedPlan(t *testing.T) {
    result := runFixture(t, "helix-prd-changed", "")

    check := findCheck(t, result, "helix-reconcile-prd")
    if check.Status != "prompt" {
        t.Fatalf("expected prompt, got %s", check.Status)
    }

    // Verify plan contains ordered artifacts
    plan := check.Prompt.Context["plan"].([]string)
    expectedOrder := []string{
        "docs/helix/01-frame/features/",
        "docs/helix/02-design/",
        "docs/helix/02-design/decisions/",
        "docs/helix/03-test/",
        "docs/helix/04-build/",
    }
    assertOrderedPrefixes(t, plan, expectedOrder)
}
```

#### TC-006: ADR Inclusion in Reconciliation
**File**: `internal/dun/engine_test.go`
**Priority**: P1
**Covers**: AC-2, GAP-005

```go
func TestHelixReconciliationIncludesADRs(t *testing.T) {
    result := runFixture(t, "helix-with-adrs", "")

    check := findCheck(t, result, "helix-reconcile-prd")
    plan := check.Prompt.Context["plan"].([]string)

    hasADR := false
    for _, artifact := range plan {
        if strings.Contains(artifact, "decisions/") {
            hasADR = true
            break
        }
    }
    if !hasADR {
        t.Fatalf("expected ADRs in reconciliation plan")
    }
}
```

### 4.3 State Rules Tests

#### TC-007: State Rules Detect PRD-Feature Mismatch
**File**: `internal/dun/state_rules_test.go`
**Priority**: P0
**Covers**: AC-1

```go
func TestStateRulesDetectPRDFeatureMismatch(t *testing.T) {
    // Given: PRD references features not yet created
    // When: State rules run
    // Then: Missing features are reported in order
}
```

### 4.4 CLI Tests

#### TC-008: CLI Reconcile Command Output Format
**File**: `cmd/dun/main_test.go`
**Priority**: P1
**Covers**: AC-3, GAP-007

```go
func TestRunReconcileOutputFormat(t *testing.T) {
    root := setupRepoFromFixture(t, "helix-prd-changed")
    var stdout bytes.Buffer
    var stderr bytes.Buffer

    // Test JSON format
    code := runInDirWithWriters(t, root, []string{"reconcile", "--format=json"}, &stdout, &stderr)
    if code != 0 {
        t.Fatalf("expected success, got %d", code)
    }

    var plan ReconcilePlan
    if err := json.Unmarshal(stdout.Bytes(), &plan); err != nil {
        t.Fatalf("invalid JSON output: %v", err)
    }

    // Verify structure
    if len(plan.Artifacts) == 0 {
        t.Fatalf("expected artifacts in plan")
    }
}
```

## 5. Test Data Requirements

### 5.1 New Fixtures Required

| Fixture | Purpose | Contents |
|---------|---------|----------|
| `helix-prd-changed` | PRD change detection | PRD with modifications, existing downstream artifacts |
| `helix-full-stack` | Complete artifact coverage | All artifact types present |
| `helix-with-adrs` | ADR inclusion | PRD + ADRs + other artifacts |

### 5.2 Fixture Structure: helix-prd-changed

```
internal/testdata/repos/helix-prd-changed/
  docs/helix/
    01-frame/
      prd.md                    # Contains scope changes
      features/
        FEAT-001-core.md        # Needs update marker
    02-design/
      architecture.md           # Needs update marker
      decisions/
        ADR-001-tech-choice.md  # Needs review marker
    03-test/
      test-plan.md              # Needs update marker
    04-build/
      implementation-plan.md    # Needs update marker
```

## 6. Implementation Priority

| Priority | Test Cases | Effort | Dependencies |
|----------|------------|--------|--------------|
| P0 | TC-001, TC-002, TC-003, TC-004, TC-005, TC-007 | 2 days | Reconciliation logic implementation |
| P1 | TC-006, TC-008 | 1 day | P0 tests passing |

## 7. Success Criteria

- All P0 test cases passing
- Code coverage for reconciliation logic >= 80%
- Plan output is deterministic (verified by TC-004)
- All artifact types included in reconciliation (verified by TC-003)
- Ordering is correct and stable (verified by TC-002, TC-005)

## 8. Notes

### 8.1 Implementation Dependencies

The test cases assume a reconciliation feature that:
1. Detects PRD changes (possibly via git diff or content hash)
2. Traverses downstream artifact dependencies
3. Emits an ordered list of impacted files
4. Provides structured output (JSON/YAML)

### 8.2 Open Questions

1. How should PRD "change" be detected? Git diff? Content hash? Manual trigger?
2. Should the reconciliation plan include confidence scores for each artifact?
3. Should implementation files (Go/TypeScript) be included or just documentation?

---

**Sign-off**: Pending
