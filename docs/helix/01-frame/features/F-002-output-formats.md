# Feature Spec: F-002 Output Formats

## Summary

Emit prompt-as-data output by default and provide LLM/JSON formats for
consumption by humans and tools.

## Requirements

- Default output format is prompt envelopes for agent checks.
- Provide `--format=llm` for concise human-readable summaries.
- Provide `--format=json` for structured results.
- Output is deterministic and stable for a given repo state.

## Acceptance Criteria

- `dun check` emits prompt envelopes by default when agent checks are present.
- `dun check --format=llm` prints concise summaries.
- `dun check --format=json` emits structured JSON output.

## Traceability

- Supports success metrics for time to first output and median run time by
  keeping output deterministic and compact.
- Primary persona: agent operators who need fast feedback loops.
