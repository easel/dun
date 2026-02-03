---
dun:
  id: F-003
  depends_on:
    - helix.prd
  review:
    self_hash: 8bbe08567869ffbb1fa3c56eb1af0d585f1918acb50240d7c541a86b4ace030e
    deps:
      helix.prd: 58d3c4be8edb0a0be9d01a3325824c9b350f758a998d02f16208525949c4f1ad
---
# Feature Spec: F-003 Plugin System

## Summary

Provide an extensible plugin system so Dun can ship built-in workflows
(including Helix doc and gate validation) and add future workflow-specific
checks while keeping discovery deterministic.

## Requirements

- Provide an extensible plugin system for future workflows.
- Include a built-in Helix plugin for documentation and gate validation.
- Load built-in plugin manifests embedded in the binary.
- Activate plugins via repo signals (paths/globs).
- Support check types: rule-set, gates, state-rules, agent prompts.
- Maintain deterministic plugin and check ordering.

## Inputs

- PRD goals for an extensible plugin system and built-in Helix plugin.
- Repository signals used to activate plugins.
- Built-in plugin manifests shipped with the CLI.

## Acceptance Criteria

- Helix plugin activates when `docs/helix/` exists.
- Gate and state rules run without custom config.
- Agent prompts emit prompt envelopes with callbacks.
- Plugin discovery and ordering are deterministic for a given repo state.

## Gaps & Conflicts

- The PRD does not define the plugin manifest schema, validation rules, or
  where manifests live on disk versus embedded in the binary.
- The activation signal format (paths/globs), precedence rules, and failure
  behavior are unspecified.
- The supported check types (rule-set, gates, state-rules, agent prompts) are
  listed here but lack definitions and lifecycle requirements.
- The scope of the Helix plugin's doc and gate validation is not specified
  (which checks, inputs, or outputs it owns).
- No conflicts identified in the provided inputs.

## Traceability

- Supports adoption success metric by enabling reuse across repos.
- Supports PRD scope for a built-in Helix plugin for doc and gate validation.
- Supports PRD scope for an extensible plugin system for future workflows.
- Supports deterministic output goals by requiring stable plugin discovery.
- Primary personas: agent operators and engineering leads.
