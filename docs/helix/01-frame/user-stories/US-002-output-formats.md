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

- `dun check` emits prompt envelope metadata by default when agent checks are
  present; prompt payloads are omitted and replaced with a task hint.
- `dun check --format=llm` prints concise summaries for humans.
- `dun check --format=json` emits structured JSON output.
- Output is deterministic for a given repo state.
