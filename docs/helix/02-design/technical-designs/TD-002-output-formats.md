---
dun:
  id: TD-002
  depends_on:
  - US-002
---
# Technical Design: TD-002 Output Formats

## Story Reference

**User Story**: US-002 Output Formats
**Parent Feature**: F-002 Output Formats
**Solution Design**: SD-002 Output Formats

## Goals

- Provide prompt, LLM, and JSON output formats for check results.
- Keep output deterministic and stable for automation.
- Allow format selection via CLI flags and config.
- Omit full prompt payloads from `dun check` output and expose them via
  `dun task <id> --prompt`.

## Non-Goals

- Rich terminal UI or interactive views.
- Streaming output.

## Technical Approach

### Implementation Strategy

- Introduce a renderer interface with concrete implementations for
  `prompt`, `llm`, and `json` formats.
- Normalize check results into a shared structure before rendering.
- Use stable ordering for checks, issues, and fields.

### Key Decisions

- Default to `prompt` format for agent loops.
- `dun check` output uses prompt placeholders; full prompts are retrieved via
  `dun task`.
- JSON output should be schema-stable to avoid breaking integrations.

## Component Changes

### Components to Modify

- `cmd/dun/main.go`: parse `--format` flag and pass to the engine.
- `internal/dun/engine.go`: route output through the selected renderer.
- `internal/dun/types.go`: ensure result types serialize cleanly.

### New Components

- `internal/dun/output_prompt.go`: human/agent prompt format.
- `internal/dun/output_llm.go`: structured LLM-friendly format.
- `internal/dun/output_json.go`: machine-readable JSON output.

## Interfaces and Config

- CLI: `dun check --format prompt|llm|json`.
- Config: `.dun/config.yaml` optional default format.

## Data and State

- No persistent state; renderers operate on in-memory results.

## Testing Approach

- Golden tests for each output format.
- Serialization tests ensuring stable ordering and field presence.

## Risks and Mitigations

- **Risk**: Output drift breaks integrations. **Mitigation**: golden tests and
  versioned schemas for JSON output.

## Rollout / Compatibility

- Default format remains `prompt` to preserve existing behavior.
