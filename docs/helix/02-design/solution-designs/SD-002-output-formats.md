# Solution Design: Output Formats

## Problem

Agents and tools need different output shapes, but the same check results must
remain deterministic and easy to parse.

## Goals

- Emit prompt envelopes by default for agent checks.
- Provide concise LLM summaries and structured JSON output.
- Preserve deterministic ordering and stable results.

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

- Should a single run emit multiple formats?
- How should large prompt payloads be summarized in LLM format?
