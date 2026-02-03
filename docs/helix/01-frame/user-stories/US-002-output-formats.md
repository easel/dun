---
dun:
  id: US-002
  depends_on:
  - F-002
---
# US-002: Emit Output Formats for Agents and Tools

As an agent operator, I want Dun to emit prompt, LLM, and JSON outputs so I
can consume results in the right format for my workflow.

## Acceptance Criteria

- `dun check` emits prompt envelopes by default when agent checks are present.
- `dun check --format=llm` prints concise summaries for humans.
- `dun check --format=json` emits structured JSON output.
- Output is deterministic for a given repo state.
