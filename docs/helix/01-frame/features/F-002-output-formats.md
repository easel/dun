---
dun:
  id: F-002
  depends_on:
    - helix.prd
  review:
    self_hash: 5d4226456b8fd1dca4daae652bbcd24fb50d14f2b1e01193db67cd5a5cf2da35
    deps:
      helix.prd: 07d49919dec51a33254b7630622ee086a5108ed5deecd456f7228f03712e699d
---
# Feature Spec: F-002 Output Formats

## Summary

Emit prompt-as-data output by default and provide LLM/JSON formats for
consumption by humans and tools.

## Requirements

- Default output format is prompt envelopes for agent checks, but `dun check`
  omits full prompt payloads (prompt field contains a task hint).
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

- `dun check` emits prompt envelope metadata by default when agent checks are
  present; prompt payloads are omitted and replaced with a task hint.
- `dun check --format=llm` prints concise summaries.
- `dun check --format=json` emits structured JSON output.
- `dun check --prompt` emits a compact, bounded task list (no inline prompt
  payloads) with task IDs and short summaries.
- `dun task <task-id> --prompt` emits the full prompt for the selected task.
- Default limits: top 10 tasks per check; summary <= 200 bytes; reason <= 160
  bytes; truncation uses `...`.
- Task IDs include the repo-state hash and are rejected if stale.
- JSON output remains structured and deterministic but omits full prompt
  payloads; it is not size-bounded like the decision prompt.

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
