---
dun:
  id: TP-015
  depends_on:
  - TD-011
---
# TP-015: Agent Quorum + Synthesis Implementation Plan

**User Story**: US-011
**Version**: 1.0.0
**Date**: 2026-02-03
**Status**: Draft
**Author**: codex

## 1. Scope

This test plan verifies the implementation-plan specific work in IP-015:
- One-shot quorum commands (`dun quorum`, `dun synth`).
- Persona-aware harness selection (`name@persona`).
- Synthesis meta-harness execution.
- Deterministic quorum summary metadata.
- Shared quorum engine between `dun loop --quorum` and one-shot commands.

## 2. Acceptance Criteria

| ID | Acceptance Criterion |
|----|---------------------|
| AC-1 | `dun quorum` parses `--task`, `--quorum`, `--harnesses`, and conflict flags |
| AC-2 | `dun synth` is a shorthand for `dun quorum --synthesize` |
| AC-3 | `name@persona` parses and is passed through to harness execution |
| AC-4 | Synthesis mode runs a meta-harness and returns merged output |
| AC-5 | Quorum summary metadata is emitted deterministically |
| AC-6 | `dun loop --quorum` uses the same quorum engine as one-shot commands |

## 3. Existing Test Coverage

| AC | Existing Test | File | Status |
|----|---------------|------|--------|
| AC-1 | Quorum flag parsing | `docs/helix/03-test/test-plans/TP-011-agent-quorum.md` | Partial |
| AC-2 | None | - | Missing |
| AC-3 | None | - | Missing |
| AC-4 | None | - | Missing |
| AC-5 | None | - | Missing |
| AC-6 | None | - | Missing |

## 4. Test Gaps

| Gap ID | Description | Priority | Acceptance Criteria |
|--------|-------------|----------|---------------------|
| GAP-015-01 | One-shot command parsing and output tests | P0 | AC-1, AC-2 |
| GAP-015-02 | Persona parsing and passthrough tests | P0 | AC-3 |
| GAP-015-03 | Synthesis mode integration test | P0 | AC-4 |
| GAP-015-04 | Quorum summary metadata determinism | P1 | AC-5 |
| GAP-015-05 | Loop uses shared quorum engine | P1 | AC-6 |

## 5. Proposed Test Cases

### 5.1 CLI Parsing

#### TC-015-01: Quorum command parsing
**File**: `cmd/dun/quorum_test.go`
**Priority**: P0
**Covers**: AC-1

```go
func TestQuorumCommandParsing(t *testing.T) {
    // Given: dun quorum --task "spec" --quorum 2 --harnesses a,b
    // Then: parsed config includes task, quorum, harnesses
}
```

#### TC-015-02: Synth shorthand parsing
**File**: `cmd/dun/quorum_test.go`
**Priority**: P0
**Covers**: AC-2

```go
func TestSynthCommandShorthand(t *testing.T) {
    // Given: dun synth --task "spec" --harnesses a,b --synthesizer a
    // Then: synthesize mode is enabled and synthesizer is set
}
```

### 5.2 Persona Parsing

#### TC-015-03: Harness persona parsing
**File**: `internal/dun/quorum_test.go`
**Priority**: P0
**Covers**: AC-3

```go
func TestParseHarnessSpecPersona(t *testing.T) {
    // Given: "codex@architect"
    // Then: Name=codex, Persona=architect
}
```

### 5.3 Synthesis Mode

#### TC-015-04: Synthesis meta-harness invoked
**File**: `internal/dun/quorum_test.go`
**Priority**: P0
**Covers**: AC-4

```go
func TestSynthesisModeInvokesMetaHarness(t *testing.T) {
    // Given: multiple harness drafts
    // Then: synthesizer runs and returns merged output
}
```

### 5.4 Summary Metadata

#### TC-015-05: Deterministic quorum summary
**File**: `internal/dun/quorum_test.go`
**Priority**: P1
**Covers**: AC-5

```go
func TestQuorumSummaryDeterministic(t *testing.T) {
    // Given: stable inputs
    // Then: summary fields are ordered and deterministic
}
```

### 5.5 Loop Integration

#### TC-015-06: Loop uses shared quorum engine
**File**: `cmd/dun/main_test.go`
**Priority**: P1
**Covers**: AC-6

```go
func TestLoopUsesSharedQuorumEngine(t *testing.T) {
    // Given: loop with --quorum
    // Then: shared quorum engine is invoked
}
```
