---
dun:
  id: SD-002
  depends_on:
    - F-002
  review:
    self_hash: f543f44dd37dea76fd38919c5f15a737ba8fbb46e621a5a1c83c526892fb0757
    deps:
      F-002: 83b2a3c2ac4e9a760bd04598c3ff9c3ea3504c0f49e8d246cd9a61d540e87898
---
# Solution Design: Output Formats

## Problem

Agents and tools need different output shapes, but the same check results must
remain deterministic and easy to parse.

## Goals

- Emit prompt envelopes by default for agent checks.
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
- Prompt Emitter: renders prompt envelopes.
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
