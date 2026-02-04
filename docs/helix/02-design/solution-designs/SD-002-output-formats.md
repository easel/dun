---
dun:
  id: SD-002
  depends_on:
    - F-002
  review:
    self_hash: 8c32ec1de236d13a6d5cbd49f766f542ae1222821a9de0fe451079e7ec351af0
    deps:
      F-002: 5d4226456b8fd1dca4daae652bbcd24fb50d14f2b1e01193db67cd5a5cf2da35
---
# Solution Design: Output Formats

## Problem

Agents and tools need different output shapes, but the same check results must
remain deterministic and easy to parse.

## Goals

- Emit prompt envelopes by default for agent checks, omitting full prompt
  payloads from `dun check` output.
- Provide `--format=llm` for concise human-readable summaries.
- Provide `--format=json` for structured results.
- Preserve deterministic ordering and stable results for a given repo state.

## Inputs

- Check results emitted by `dun check`.
- Repository state that drives deterministic ordering.
- PRD goals for deterministic, agent-friendly output formats.

## Gaps & Conflicts

- Missing formal schema for the JSON output format (field names, ordering,
  error model).
- Missing definition of the prompt envelope structure and how callbacks are
  encoded for agent checks.
- Missing length and content guidelines for `--format=llm` summaries.
- Missing rules for how multi-check results are ordered and grouped across
  formats.
- No conflicts identified in the provided inputs.

## Approach

1. Normalize check results into a single internal model.
2. Select the output renderer based on `--format`.
3. Render results with stable ordering and consistent fields.

## Components

- Result Model: canonical representation of check outcomes.
- Prompt Emitter: renders prompt envelopes with compact prompt placeholders.
- LLM Renderer: emits concise summaries.
- JSON Renderer: emits structured machine output.
- Output Selector: chooses renderer based on CLI flags.

## Data Flow

1. Check runner produces result objects.
2. Reporter builds the result model in deterministic order.
3. Output selector chooses renderer.
4. Renderer emits formatted output to stdout.

## Open Questions

- None beyond the gaps listed above.
