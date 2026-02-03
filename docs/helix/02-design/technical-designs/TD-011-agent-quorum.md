---
dun:
  id: TD-011
  depends_on:
  - US-011
---
# Technical Design: TD-011 Agent Quorum

## Story Reference

**User Story**: US-011 Agent Quorum
**Parent Feature**: F-018 Agent Quorum
**Solution Design**: SD-011 Agent Quorum

## Goals

- Allow multiple harnesses to review each task.
- Require agreement based on quorum strategy before applying changes.
- Provide conflict reporting and escalation options.
- Provide a one-shot quorum/synthesis command for non-loop tasks.
- Allow per-harness personas (`harness@persona`) supplied by the harness/DDX layer.

## Non-Goals

- Human-in-the-loop UI (log-based escalation is sufficient).

## Technical Approach

### Implementation Strategy

- Run the same prompt through multiple harnesses.
- Normalize responses into a comparable form.
- Apply a quorum strategy: any, majority, unanimous, or N.
- Add a synthesis mode that merges drafts via a meta-harness.
- Add a dedicated `dun quorum` command and a `dun synth` shorthand.

### Key Decisions

- Default to parallel execution for performance; allow cost mode (sequential).
- Keep similarity threshold configurable for conflict detection.
- `dun loop --quorum` applies quorum to the iteration prompt, not per-check prompts.
- Persona definitions live in harness/DDX; Dun only passes persona names.

## Component Changes

### Components to Modify

- `internal/dun/quorum.go`: compute agreement and select a response.
- `internal/dun/conflict_detection.go`: compare responses semantically.
- `cmd/dun/main.go`: parse `--quorum`, `--harnesses`, and conflict flags.
- `cmd/dun/quorum.go`: new one-shot quorum/synthesis command surface.

### New Components

- Quorum result reporter with conflict details.
- Synthesis meta-harness prompt and result merger.

## Interfaces and Config

- CLI:
  - `dun loop --quorum`, `--harnesses`, `--cost-mode`, `--escalate`.
  - `dun quorum [--synthesize] --task "<prompt>" --harnesses ...`
  - `dun synth` = `dun quorum --synthesize`
- Config: default quorum strategy, similarity threshold, harness personas,
  and synthesis meta-harness settings.
- Quorum vote and synthesis prompts are owned by Dun and passed as task prompts
  to the harnesses (persona/system prompts remain agent-owned).

## Data and State

- Per-iteration quorum summary stored in memory; optional log output.
- One-shot quorum/synthesis outputs return a single selected/merged response.

## Testing Approach

- Unit tests for quorum strategies and similarity thresholds.
- Integration tests for conflict and escalation paths.

## Risks and Mitigations

- **Risk**: Conflicts slow down loops. **Mitigation**: allow `--prefer` and
  `--cost-mode` options.

## Rollout / Compatibility

- Backwards compatible; quorum is opt-in.
