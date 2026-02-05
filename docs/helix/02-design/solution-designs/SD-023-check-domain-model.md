---
dun:
  id: SD-023
  depends_on:
    - F-023
  review:
    self_hash: 08154ce72da29c20073be694be803ec654134d755e8b877ac69b03d8edd59dd2
    deps:
      F-023: 9b5f839c24e68e270c4166652ec89c5334406b462deefe97ac09582d8ae6263e
---
# Solution Design: Check Domain Model

## Problem

Check behavior is implemented through a monolithic config structure and a large
switch statement, making new check types and consistent summarization difficult.

## Goals

- Introduce a clear check lifecycle: discover, plan, evaluate, summarize, prompt,
  score.
- Decouple check execution from the engine using a registry of check types.
- Standardize result metadata without breaking existing outputs.

## Approach

- Keep plugin YAMLs intact, but normalize check definitions into a common
  structure and decode type-specific configs via a registry.
- Add a summarizer that sets `summary`, `score`, and optional update signals on
  check results after evaluation.
- Keep CLI rendering as a thin adapter over structured results.

## Components

- Check Registry: maps check types to type handlers.
- Check Definition: normalized metadata shared across types.
- Type Configs: per-check-type config structs.
- Summarizer: derives summary/score/update signals from results.

## Open Questions

- None beyond technical design details.
