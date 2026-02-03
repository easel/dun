---
dun:
  id: SD-003
  depends_on:
    - F-003
  review:
    self_hash: c7db1b0bba417ae98a52309d663467cb5eb703a2f52159ae986d63fb71bbf643
    deps:
      F-003: 8bbe08567869ffbb1fa3c56eb1af0d585f1918acb50240d7c541a86b4ace030e
---
# Solution Design: Plugin System

## Problem

Dun needs workflow-specific checks without hardcoding every rule in the core
binary.

## Goals

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

## Gaps & Conflicts

- The plugin manifest schema, validation rules, and the split between embedded
  versus on-disk manifests are undefined.
- The activation signal format (paths/globs), precedence rules, and failure
  behavior are unspecified.
- The listed check types (rule-set, gates, state-rules, agent prompts) lack
  definitions and lifecycle requirements.
- The Helix plugin scope for doc and gate validation is not specified (which
  checks, inputs, or outputs it owns).
- No conflicts identified in the provided inputs.

## Approach

1. Embed built-in plugin manifests into the binary at build time.
2. Load manifests into a registry on startup.
3. Match repo signals (paths/globs) to activate plugins.
4. Merge active plugin checks into a single plan.
5. Sort plugins and checks by a stable key to keep ordering deterministic.
6. Emit agent prompt envelopes for agent prompt checks.

## Components

- Manifest Loader: reads embedded manifests.
- Plugin Registry: stores manifests and activation rules.
- Signal Matcher: evaluates repo signals.
- Check Planner: assembles a deterministic check plan.
- Prompt Emitter: renders prompt envelopes for agent prompt checks.

## Data Flow

1. Loader reads embedded manifests.
2. Matcher evaluates repo signals (ex: `docs/helix/`).
3. Registry activates matching plugins (Helix included).
4. Planner merges checks into a single ordered plan.
5. Check runner executes rule-set, gate, and state-rule checks.
6. Prompt emitter produces prompt envelopes for agent checks.

## Open Questions

- Should external manifests be supported in later phases?
- How should manifest versioning and conflicts be handled?
- What is the canonical schema for plugin manifests and check types?
