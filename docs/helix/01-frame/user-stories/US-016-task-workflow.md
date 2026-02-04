---
dun:
  id: US-016
  depends_on:
  - F-002
  - F-017
---
# US-016: Task List + Task Prompt Workflow

## User Story

**As an** agent operator,
**I want** `dun check --prompt` to show a bounded task list and a follow-up task
command to fetch full prompts,
**So that** I can pick work quickly without giant decision prompts.

## Acceptance Criteria

### AC-1: Bounded Decision Prompt
- [ ] `dun check --prompt` lists tasks with short summaries and reasons.
- [ ] Each check is capped to top N tasks (default 10).
- [ ] Summaries and reasons are truncated to size limits.

### AC-2: Stable Task IDs
- [ ] Task IDs include a repo-state hash derived from git HEAD and working tree
      status.
- [ ] Task IDs are stable within a repo state and rejected when stale.

### AC-3: Task Prompt Retrieval
- [ ] `dun task <task-id>` prints a concise task summary.
- [ ] Task summaries never include full prompt payloads.
- [ ] `dun task <task-id> --prompt` prints the full prompt payload.
- [ ] Decision prompt hints how to fetch the full prompt.

## Technical Notes

- Use a short hash suffix for repo state to prevent stale task reuse.
- Keep decision prompt deterministic for the same repo state.
- Re-run checks in `dun task` to resolve prompts for the current state.

## Dependencies

- F-002 Output Formats
- F-017 Autonomous Iteration

## Priority

P1 - Reduces prompt bloat and improves routing.

## Estimation

Medium.
