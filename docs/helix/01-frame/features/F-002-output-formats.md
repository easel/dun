---
dun:
  id: F-002
  depends_on:
    - helix.prd
  review:
    self_hash: 83b2a3c2ac4e9a760bd04598c3ff9c3ea3504c0f49e8d246cd9a61d540e87898
    deps:
      helix.prd: 58d3c4be8edb0a0be9d01a3325824c9b350f758a998d02f16208525949c4f1ad
---
# Feature Spec: F-002 Output Formats

## Summary

Emit prompt-as-data output by default and provide LLM/JSON formats for
consumption by humans and tools.

## Requirements

- Default output format is prompt envelopes for agent checks.
- Provide `--format=llm` for concise human-readable summaries.
- Provide `--format=json` for structured results.
- The decision prompt (`dun check --prompt`) must list tasks without inlining
  full prompt payloads.
- The decision prompt must include stable task IDs (format:
  `<check-id>@<state>` or `<check-id>#<n>@<state>` for per-issue tasks), a
  short summary, and a short "why" reason for each task.
- `state` is a short repo-state hash derived from the current repo revision
  and working tree status (when git is available).
- Limit the decision prompt to a bounded number of tasks per check (top N),
  with explicit size caps for summaries and reasons.
- Provide a follow-up command (`dun task <task-id> --prompt`) to emit the full
  prompt for the selected task.
- Output is deterministic and stable for a given repo state.

## Inputs

- Check results emitted by `dun check`.
- Repository state that drives deterministic ordering.
- PRD goals for deterministic, agent-friendly output formats.

## Acceptance Criteria

- `dun check` emits prompt envelopes by default when agent checks are present.
- `dun check --format=llm` prints concise summaries.
- `dun check --format=json` emits structured JSON output.
- `dun check --prompt` emits a compact, bounded task list (no inline prompt
  payloads) with task IDs and short summaries.
- `dun task <task-id> --prompt` emits the full prompt for the selected task.
- Default limits: top 10 tasks per check; summary <= 200 bytes; reason <= 160
  bytes; truncation uses `...`.
- Task IDs include the repo-state hash and are rejected if stale.
- JSON output remains a full check result (including prompt envelopes where
  available); it is not size-bounded like the decision prompt.

## Gaps & Conflicts

- Missing formal schema for the JSON output format (field names, ordering,
  error model).
- Missing definition of the prompt envelope structure and how callbacks are
  encoded for agent checks.
- Missing length and content guidelines for `--format=llm` summaries.
- Missing rules for how `dun task` behaves when the repo has changed and task
  IDs are stale (should return a clear error).
- Missing rules for how multi-check results are ordered and grouped across
  formats.
- No conflicts identified in the provided inputs.

## Traceability

- Supports success metrics for time to first output and median run time by
  keeping output deterministic and compact.
- Primary persona: agent operators who need fast feedback loops.
