---
dun:
  id: TD-016
  depends_on:
  - F-002
  - F-017
  - US-016
---
# TD-016: Task List + Task Prompt Workflow

## Goal

Keep decision prompts compact by listing bounded task summaries and deferring
full prompt payloads to a follow-up `dun task` command.

## Technical Approach

### 1) Decision Prompt Task List

**File**: `cmd/dun/main.go`

- Emit a task list per check instead of inline prompt payloads.
- Cap tasks per check to a fixed top-N (default 10).
- Include short summaries and short "why" reasons.
- Truncate summaries/reasons to fixed byte limits.
- Append a repo-state hash to task IDs: `<check-id>@<state>` or
  `<check-id>#<n>@<state>`.

### 2) Repo-State Hash

**File**: `cmd/dun/task.go`

- Compute a short hash from git HEAD + working tree status.
- Use this hash in task IDs for staleness detection.
- Reject task IDs with missing or mismatched state hashes.

### 3) Task Command

**File**: `cmd/dun/task.go`

- Add `dun task <task-id>` to emit task summary metadata.
- Ensure task summaries never include full prompt payloads.
- Add `--prompt` flag to print the full prompt.
- Re-run checks to resolve the selected task in the current state.

### 4) Guidance Updates

- Update help text and AGENTS template to reference `dun task`.
- Update specs to document the bounded task list and task command.

## Testing

- Unit tests in `cmd/dun/main_test.go` for prompt rendering and task IDs.
- Task command tests for prompt retrieval and invalid/stale IDs.
- Truncation behavior tests for summaries and reasons.

## Rollout

- Additive; no flag changes required.
- Decision prompt remains deterministic for a given repo state.
