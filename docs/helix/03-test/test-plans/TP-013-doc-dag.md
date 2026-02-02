# TP-013: Doc DAG + Review Stamps

**User Story**: US-013
**Version**: 1.0.0
**Date**: 2026-02-02
**Status**: Draft
**Author**: oscar

## 1. Acceptance Criteria

From US-013:

| ID | Acceptance Criterion |
|----|---------------------|
| AC-1 | A document with a changed parent is flagged as stale until re-stamped |
| AC-2 | Required documents missing from the DAG are reported as missing |
| AC-3 | Dun emits prompts for stale/missing docs with parent context |

## 2. Existing Test Coverage

### 2.1 Tests Mapped to Acceptance Criteria

| AC | Existing Test | File | Status |
|----|---------------|------|--------|
| AC-1 | None | - | Missing |
| AC-2 | None | - | Missing |
| AC-3 | None | - | Missing |

### 2.2 Coverage Analysis

No existing tests cover doc-DAG behavior. All acceptance criteria are gaps.

## 3. Test Gaps

### 3.1 Critical Gaps (P0)

| Gap ID | Description | Priority | Acceptance Criteria |
|--------|-------------|----------|---------------------|
| GAP-001 | No test for cascade stale detection | P0 | AC-1 |
| GAP-002 | No test for missing required roots | P0 | AC-2 |
| GAP-003 | No test for prompt envelope content | P0 | AC-3 |

### 3.2 Secondary Gaps (P1)

| Gap ID | Description | Priority | Acceptance Criteria |
|--------|-------------|----------|---------------------|
| GAP-004 | No test for `dun stamp` updating review deps | P1 | AC-1 |
| GAP-005 | No test for deterministic ordering | P1 | AC-3 |

## 4. Proposed Test Cases

### 4.1 Unit Tests

#### TC-001: Frontmatter Parsing
**File**: `internal/dun/frontmatter_test.go`
**Priority**: P0
**Covers**: AC-1, AC-3

```go
func TestFrontmatterParseDunBlock(t *testing.T) {
    // Given: a markdown file with dun frontmatter
    // When: parsed
    // Then: id, depends_on, prompt, review fields are extracted
}
```

#### TC-002: Hash Excludes Review
**File**: `internal/dun/hash_test.go`
**Priority**: P0
**Covers**: AC-1

```go
func TestHashExcludesReviewSection(t *testing.T) {
    // Given: same doc with different dun.review
    // Then: hash is identical
}
```

#### TC-003: Missing Required Root
**File**: `internal/dun/doc_dag_test.go`
**Priority**: P0
**Covers**: AC-2

```go
func TestDocDagMissingRequiredRoot(t *testing.T) {
    // Given: graph file requiring prd.md, file missing
    // Then: missing issue is emitted
}
```

#### TC-004: Stamp Updates Review Deps
**File**: `internal/dun/stamp_test.go`
**Priority**: P1
**Covers**: AC-1

```go
func TestStampUpdatesReviewDeps(t *testing.T) {
    // Given: parent + child
    // When: dun stamp runs on child
    // Then: review.deps[parent] matches current parent hash
}
```

### 4.2 Integration Tests (Required for Day 1)

#### TC-005: Cascade Stale Detection (End-to-End)
**File**: `internal/dun/engine_test.go`
**Priority**: P0
**Covers**: AC-1, AC-3

```go
func TestDocDagCascadeStale(t *testing.T) {
    // Given: fixture repo with parent+child, stamped
    // When: parent changes
    // Then: child is stale and prompt is emitted
}
```

**Fixture**: `internal/testdata/repos/doc-dag-cascade/`
- `docs/helix/01-frame/prd.md` with dun.frontmatter
- `docs/helix/02-design/architecture.md` depending on PRD
- Both have review stamps reflecting initial hashes

**Expected**:
- `doc-dag` check status = warn/fail
- Issue `stale:helix.architecture`
- Prompt envelope includes PRD content as input

#### TC-006: Missing Required Doc Prompt
**File**: `internal/dun/engine_test.go`
**Priority**: P0
**Covers**: AC-2, AC-3

```go
func TestDocDagMissingRequiredPrompt(t *testing.T) {
    // Given: graph requires prd.md, file missing
    // Then: prompt envelope for prd creation is emitted
}
```

## 5. Test Data Plan

- New fixtures under `internal/testdata/repos/doc-dag-*`.
- Graph file under `.dun/graphs/helix.yaml` in fixtures.
- Minimal prompt templates under `internal/plugins/builtin/helix/prompts/` (or test-local prompt stubs).

## 6. Exit Criteria

- All P0 tests implemented and passing.
- Integration test TC-005 confirms cascade detection from day 1.
