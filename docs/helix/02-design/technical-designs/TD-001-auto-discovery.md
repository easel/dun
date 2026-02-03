---
dun:
  id: TD-001
  depends_on:
  - US-001
---
# Technical Design: TD-001 Auto-Discovery

## Story Reference

**User Story**: US-001 Auto-Discover Repo Checks
**Parent Feature**: F-001 Auto-discovery
**Solution Design**: SD-001 Auto-Discovery

## Goals

- Detect repo signals (`go.mod`, `docs/helix/`) and enable the right checks.
- Keep discovery deterministic for the same repo state.
- Keep discovery fast and local (no network calls).

## Non-Goals

- User-configured custom plugin discovery (covered by external plugin loading).
- Remote registry lookups.

## Technical Approach

### Implementation Strategy

- Centralize signal detection in the plan builder so all checks share the same
  input set.
- Represent plugin triggers as explicit predicates (path exists, file glob,
  config flag).
- Sort enabled checks deterministically by plugin ID then check ID.

### Key Decisions

- Use file existence checks for signal detection instead of content parsing to
  keep discovery fast.
- Treat `docs/helix/` as the Helix signal to activate documentation checks.

## Component Changes

### Components to Modify

- `internal/dun/plan.go`: build the plan by evaluating plugin triggers.
- `internal/dun/plugin_loader.go`: load built-in plugin manifests.
- `internal/plugins/builtin/**/plugin.yaml`: declare triggers for Go and Helix.

### New Components

- None.

## Interfaces and Config

- No new CLI flags.
- Config can override plugin enable/disable (existing behavior).

## Data and State

- Plan is an in-memory structure; no persisted state required.

## Testing Approach

- Unit tests for signal detection when `go.mod` or `docs/helix/` exist.
- Plan ordering tests to ensure deterministic output.

## Risks and Mitigations

- **Risk**: False positives if a repo contains `docs/helix/` for other reasons.
  **Mitigation**: require at least one known Helix file before enabling checks.

## Rollout / Compatibility

- Backwards compatible; only adds automatic activation when signals are present.
