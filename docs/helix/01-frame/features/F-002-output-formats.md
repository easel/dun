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
- Output is deterministic and stable for a given repo state.

## Inputs

- Check results emitted by `dun check`.
- Repository state that drives deterministic ordering.
- PRD goals for deterministic, agent-friendly output formats.

## Acceptance Criteria

- `dun check` emits prompt envelopes by default when agent checks are present.
- `dun check --format=llm` prints concise summaries.
- `dun check --format=json` emits structured JSON output.

## Gaps & Conflicts

- Missing formal schema for the JSON output format (field names, ordering,
  error model).
- Missing definition of the prompt envelope structure and how callbacks are
  encoded for agent checks.
- Missing length and content guidelines for `--format=llm` summaries.
- Missing rules for how multi-check results are ordered and grouped across
  formats.
- No conflicts identified in the provided inputs.

## Traceability

- Supports success metrics for time to first output and median run time by
  keeping output deterministic and compact.
- Primary persona: agent operators who need fast feedback loops.
