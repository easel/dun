---
dun:
  id: IP-016-task-workflow
  depends_on:
    - F-002
    - F-017
    - US-009
    - US-016
    - TD-009
    - TD-016
    - TP-002
    - TP-009
    - TP-016
---
# IP-016: Task List + Task Prompt Workflow

## Goal

Keep decision prompts small by emitting bounded task summaries and deferring
full prompt payloads to a follow-up task command.

## Inputs

- F-002 Output Formats (`docs/helix/01-frame/features/F-002-output-formats.md`)
- F-017 Autonomous Iteration (`docs/helix/01-frame/features/F-017-autonomous-iteration.md`)
- US-009 Autonomous Iteration (`docs/helix/01-frame/user-stories/US-009-autonomous-iteration.md`)
- TD-009 Autonomous Iteration (`docs/helix/02-design/technical-designs/TD-009-autonomous-iteration.md`)
- TP-002 Output Formats (`docs/helix/03-test/test-plans/TP-002-output-formats.md`)
- TP-009 Autonomous Iteration (`docs/helix/03-test/test-plans/TP-009-autonomous-iteration.md`)

## Execution Plan

1. **Decision prompt task list**
   - Update `printPrompt` to emit task IDs, short summaries, and short "why"
     reasons instead of inlining prompt payloads.
   - Limit tasks per check to a fixed top-N (default 10).
   - Enforce summary/reason byte caps (summary <= 200, reason <= 160, `...`
     truncation).
   - Ensure deterministic ordering and stable task IDs.
   - Compute a repo-state hash (git HEAD + working tree status) and append it
     to task IDs as `@<state>`.

2. **Task ID format + selection**
   - Define task IDs as `<check-id>@<state>` or `<check-id>#<n>@<state>` for
     per-issue tasks.
   - Show task IDs in the decision prompt and include a hint to fetch the
     full prompt with `dun task <task-id> --prompt`.
   - Specify task ordering: follow check plan order; per-issue tasks are
     1-based in the order emitted by the check.

3. **Task prompt command**
   - Add a `dun task` command that:
     - Accepts a task ID.
     - Prints a short summary by default.
     - Prints the full prompt when `--prompt` is specified.
     - Re-runs checks to resolve the task in the current repo state (no cache).
     - Validates the repo-state hash and returns a clear error + non-zero exit
       code for invalid/stale task IDs.
   - Wire help text and AGENTS guidance to the new command.

4. **Tests**
   - Add tests that verify prompt payloads are not inlined.
   - Add tests for the `dun task --prompt` flow.
   - Add tests for invalid/stale task IDs and truncation behavior.

5. **Run tests**
   - `go test ./cmd/dun -run Task`
   - `go test ./...`

## Completion Criteria

- `dun check --prompt` emits bounded task lists without inlining full prompts.
- Each task has a stable ID and concise summary + reason.
- `dun task <task-id> --prompt` prints the full prompt for the selected task.
- Tests for prompt output and task command behavior pass.
