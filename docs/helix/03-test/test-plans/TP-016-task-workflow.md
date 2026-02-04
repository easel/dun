---
dun:
  id: TP-016
  depends_on:
  - TD-009
  - F-002
  - F-017
  - US-009
---
# TP-016: Task List + Task Prompt Workflow

**User Story**: US-009
**Version**: 1.0.0
**Date**: 2026-02-04
**Status**: Draft
**Author**: codex

## 1. Scope

This test plan verifies the IP-016 task workflow changes:
- Decision prompt shows bounded task lists (no inline prompt payloads).
- Task IDs include repo-state hashes and are stable within a repo state.
- Task summaries and reasons are truncated to size limits.
- `dun task` resolves task IDs and emits either a summary or the full prompt.
- Stale or invalid task IDs are rejected with clear errors.

## 2. Acceptance Criteria

| ID | Acceptance Criterion |
|----|---------------------|
| AC-1 | `dun check --prompt` lists bounded tasks with summaries + reasons |
| AC-2 | Decision prompt omits full prompt payloads |
| AC-3 | Task IDs include repo-state hash and are rejected when stale |
| AC-4 | `dun task <task-id>` prints task summary metadata |
| AC-5 | `dun task <task-id> --prompt` prints the full prompt |
| AC-6 | Summary/reason text is truncated to configured byte limits |

## 3. Existing Test Coverage

| AC | Existing Test | File | Status |
|----|---------------|------|--------|
| AC-1 | Prompt variants include task section | `cmd/dun/main_test.go` | Partial |
| AC-2 | `TestPrintPromptOmitsPromptContent` | `cmd/dun/main_test.go` | Covered |
| AC-3 | None | - | Missing |
| AC-4 | None | - | Missing |
| AC-5 | `TestRunTaskPrompt` | `cmd/dun/main_test.go` | Covered |
| AC-6 | None | - | Missing |

## 4. Test Gaps

| Gap ID | Description | Priority | Acceptance Criteria |
|--------|-------------|----------|---------------------|
| GAP-016-01 | Enforce max tasks per check (top N) | P1 | AC-1 |
| GAP-016-02 | Stale task ID rejection (state mismatch) | P0 | AC-3 |
| GAP-016-03 | Invalid task ID formatting errors | P1 | AC-3 |
| GAP-016-04 | `dun task` summary output assertions | P1 | AC-4 |
| GAP-016-05 | Summary/reason truncation limits | P1 | AC-6 |

## 5. Proposed Test Cases

### 5.1 Decision Prompt Task List

#### TC-016-01: Task list bounded per check
**File**: `cmd/dun/main_test.go`
**Priority**: P1
**Covers**: AC-1

```go
func TestPrintPromptTaskListBounded(t *testing.T) {
    // Given: a check with > maxTasksPerCategory issues
    // Then: prompt shows only the first N tasks and indicates total
}
```

#### TC-016-02: Prompt omits prompt payloads
**File**: `cmd/dun/main_test.go`
**Priority**: P0
**Covers**: AC-2

```go
func TestPrintPromptOmitsPromptContent(t *testing.T) {
    // Given: a prompt check with large prompt text
    // Then: decision prompt does not inline that prompt text
}
```

### 5.2 Task Command

#### TC-016-03: Stale task ID rejected
**File**: `cmd/dun/main_test.go`
**Priority**: P0
**Covers**: AC-3

```go
func TestRunTaskRejectsStaleID(t *testing.T) {
    // Given: task ID with a different repo-state hash
    // Then: dun task fails with a clear stale-id message
}
```

#### TC-016-04: Invalid task ID formatting
**File**: `cmd/dun/main_test.go`
**Priority**: P1
**Covers**: AC-3

```go
func TestRunTaskRejectsInvalidID(t *testing.T) {
    // Given: missing @state or invalid issue index
    // Then: dun task returns usage error
}
```

#### TC-016-05: Task summary output
**File**: `cmd/dun/main_test.go`
**Priority**: P1
**Covers**: AC-4

```go
func TestRunTaskSummaryOutput(t *testing.T) {
    // Given: a task ID for an issue
    // Then: output includes summary, status, and check metadata
}
```

### 5.3 Truncation

#### TC-016-06: Summary/reason truncation
**File**: `cmd/dun/task_test.go`
**Priority**: P1
**Covers**: AC-6

```go
func TestTaskSummaryTruncation(t *testing.T) {
    // Given: summary/reason longer than max bytes
    // Then: output is truncated with "..."
}
```
