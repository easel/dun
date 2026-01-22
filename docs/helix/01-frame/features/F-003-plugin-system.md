# Feature Spec: F-003 Plugin System

## Summary

Support manifest-defined checks so Dun can detect and run workflow-specific
rules (including Helix).

## Requirements

- Load built-in plugin manifests embedded in the binary.
- Activate plugins via repo signals (paths/globs).
- Support check types: rule-set, gates, state-rules, agent prompts.
- Maintain deterministic check ordering.

## Acceptance Criteria

- Helix plugin activates when `docs/helix/` exists.
- Gate and state rules run without custom config.
- Agent prompts emit prompt envelopes with callbacks.

## Traceability

- Supports adoption success metric by enabling reuse across repos.
- Primary personas: agent operators and engineering leads.
