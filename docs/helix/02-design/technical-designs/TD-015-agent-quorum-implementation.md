---
dun:
  id: TD-015
  depends_on:
  - US-015
---
# Technical Design: TD-015 Agent Quorum + Synthesis Commands

## Story Reference

**User Story**: US-015 Implement Agent Quorum + Synthesis Commands
**Parent Feature**: F-018 Agent Quorum
**Solution Design**: SD-011 Agent Quorum

## Goals

- Provide `dun quorum` and `dun synth` as one-shot command surfaces.
- Reuse the existing quorum engine from `dun loop --quorum`.
- Support `name@persona` harness specs and a synthesis meta-harness.
- Emit deterministic quorum summary metadata for automation.

## Non-Goals

- Changing quorum strategies or similarity algorithms beyond the existing spec.
- Introducing a new policy system; rely on existing automation modes.

## Technical Approach

### Implementation Strategy

- Add a dedicated command entry point (`cmd/dun/quorum.go`) that parses flags
  and builds a `QuorumConfig` shared with the loop.
- Reuse the quorum execution path in `internal/dun/quorum.go` for both one-shot
  and loop modes to keep behavior consistent.
- Extend harness parsing to accept `name@persona` and pass persona names to the
  harness layer without interpretation.
- In synthesis mode, run the meta-harness with a deterministic prompt that
  merges drafts into a single response.

### Key Decisions

- One-shot commands share the same quorum engine as `dun loop --quorum`.
- Quorum summary output is structured and stable for machine parsing.
- Persona definitions remain owned by the harness/DDX layer.

## Component Changes

### Components to Modify

- `cmd/dun/main.go`: register `quorum` and `synth` commands.
- `cmd/dun/quorum.go`: flag parsing and command execution.
- `internal/dun/quorum.go`: shared execution path for vote and synthesis.
- `internal/dun/config.go`: default quorum settings and synthesizer config.
- `internal/dun/types.go`: harness spec and synthesizer types.

### New Components

- None (reuse existing quorum and harness code paths).

## Interfaces and Config

- CLI:
  - `dun quorum --task "..." --quorum any|majority|unanimous|N --harnesses a,b`
  - `dun synth --task "..." --harnesses a,b --synthesizer c@persona`
- Config:
  - `.dun/config.yaml` entries under `quorum.*` for defaults.

## Data and State

- No new persistent state; summary metadata is emitted per invocation.

## Testing Approach

- CLI parsing tests for `dun quorum` and `dun synth` flags.
- Unit tests for `name@persona` parsing and synthesizer selection.
- Integration tests for synthesis mode and deterministic summary output.

## Risks and Mitigations

- **Risk**: Inconsistent behavior between loop and one-shot commands.
  **Mitigation**: Single shared quorum execution path.
- **Risk**: Non-deterministic summary ordering.
  **Mitigation**: Stable sorting and explicit field ordering in outputs.

## Rollout / Compatibility

- Additive commands; no breaking changes to existing loop behavior.
