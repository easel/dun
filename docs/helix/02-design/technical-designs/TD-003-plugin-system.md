---
dun:
  id: TD-003
  depends_on:
  - US-003
---
# Technical Design: TD-003 Plugin System

## Story Reference

**User Story**: US-003 Plugin System
**Parent Feature**: F-003 Plugin System
**Solution Design**: SD-003 Plugin System

## Goals

- Define checks as declarative plugins with triggers, inputs, and commands.
- Load built-in plugins reliably and deterministically.
- Allow checks to be composed without code changes.

## Non-Goals

- External plugin loading (covered by F-020).
- Remote plugin registries.

## Technical Approach

### Implementation Strategy

- Use YAML manifests to define plugin metadata, triggers, and check definitions.
- Parse manifests at startup and build a normalized plugin registry.
- Evaluate triggers during plan construction to activate checks.

### Key Decisions

- Keep plugin schema minimal: id, checks, triggers, inputs.
- Use strict schema validation to fail fast on malformed plugins.

## Component Changes

### Components to Modify

- `internal/dun/plugin_loader.go`: load and validate plugin manifests.
- `internal/dun/types.go`: plugin and check schema types.
- `internal/dun/engine.go`: resolve plugins into runnable checks.

### New Components

- `internal/plugins/builtin/**/plugin.yaml`: built-in plugin definitions.

## Interfaces and Config

- Plugin manifests under `internal/plugins/builtin/`.
- Config can disable specific plugin IDs.

## Data and State

- Plugin registry is in-memory; no persistence required.

## Testing Approach

- Schema validation tests for plugin manifests.
- Plan activation tests for trigger evaluation.

## Risks and Mitigations

- **Risk**: Plugin drift breaks plans. **Mitigation**: validation + unit tests.

## Rollout / Compatibility

- Backwards compatible; built-in plugins ship with the binary.
