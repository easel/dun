---
dun:
  id: TP-014
  depends_on:
  - TD-014
---
# TP-014: Beads Work Routing

**User Story**: US-014
**Version**: 1.0.0
**Date**: 2026-02-03
**Status**: Draft
**Author**: oscar

## 1. Acceptance Criteria

From US-014:

| ID | Acceptance Criterion |
|----|---------------------|
| AC-1 | Detect Beads presence and skip safely when CLI missing |
| AC-2 | Routing prompt lists top ready beads |
| AC-3 | Work detail prompt instructs `bd show <id>` |

## 2. Existing Test Coverage

### 2.1 Tests Mapped to Acceptance Criteria

| AC | Existing Test | File | Status |
|----|---------------|------|--------|
| AC-1 | Beads CLI skip tests | `internal/dun/beads_checks_test.go` | Covered |
| AC-2 | None | - | Missing |
| AC-3 | None | - | Missing |

## 3. Test Gaps

| Gap ID | Description | Priority | Acceptance Criteria |
|--------|-------------|----------|---------------------|
| GAP-001 | Routing prompt lacks beads candidate section test | P0 | AC-2 |
| GAP-002 | Beads prompt missing `bd show` assertion | P0 | AC-3 |

## 4. Proposed Test Cases

### 4.1 Unit Tests

#### TC-001: Routing Prompt Includes Beads Candidates
**File**: `cmd/dun/main_test.go`
**Priority**: P0
**Covers**: AC-2

```go
func TestPrintPromptIncludesBeadsCandidates(t *testing.T) {
    // Given: beads-suggest or beads-ready issues present
    // Then: prompt shows a Beads section with IDs and titles
}
```

#### TC-002: Beads Prompt Includes bd show
**File**: `internal/dun/beads_checks_test.go`
**Priority**: P0
**Covers**: AC-3

```go
func TestRunBeadsSuggestCheck_PromptIncludesBdShow(t *testing.T) {
    // Given: beads-suggest returns a candidate
    // Then: prompt contains `bd show <id>`
}
```
