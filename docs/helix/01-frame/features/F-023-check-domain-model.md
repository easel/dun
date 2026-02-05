---
dun:
  id: F-023
  depends_on:
    - helix.prd
  review:
    self_hash: 9b5f839c24e68e270c4166652ec89c5334406b462deefe97ac09582d8ae6263e
    deps:
      helix.prd: 07d49919dec51a33254b7630622ee086a5108ed5deecd456f7228f03712e699d
---
# Feature Spec: F-023 Check Domain Model

## Summary

Make checks a first-class domain model with a consistent lifecycle (discover,
plan, evaluate, summarize, prompt, score) so Dun can extend and reason about
checks without special-case code.

## Requirements

- Represent checks as explicit domain objects with shared metadata and
  type-specific configuration.
- Provide a registry for check types so execution is data-driven (no monolithic
  switch statements).
- Standardize check results with summary, score, prompt metadata, and optional
  update/freshness signals.
- Preserve existing CLI behavior and check outputs while improving internal
  structure.
- Ensure prompt generation and task summaries remain bounded and deterministic.

## Acceptance Criteria

- Check planning uses a typed registry to decode and run checks.
- Check results include a summary and score field (optional in output when
  unset).
- Check types that detect stale/missing artifacts surface update signals in a
  consistent shape.
- Existing checks and plugins continue to run without changes to plugin YAML.
- `go test ./...` passes.

## Gaps & Conflicts

- None identified.
