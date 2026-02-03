---
dun:
  id: TD-005
  depends_on:
  - US-005
---
# Technical Design: TD-005 Doc Reconciliation

## Story Reference

**User Story**: US-005 Doc Reconcile
**Parent Feature**: F-006 Doc Reconciliation
**Solution Design**: SD-005 Doc Reconcile

## Goals

- Detect drift across PRD, features, stories, designs, ADRs, and tests.
- Produce a deterministic reconciliation plan for downstream updates.
- Keep the output actionable and ordered by dependency.

## Non-Goals

- Automated editing of documents (agent can apply changes).
- Semantic validation of implementation correctness.

## Technical Approach

### Implementation Strategy

- Build an input set of Helix artifacts in dependency order.
- Generate a prompt that asks the agent to identify mismatches and propose
  concrete update steps.
- Return a structured response envelope with clear next actions.

### Key Decisions

- Reuse existing prompt/response pipeline rather than a new check type.
- Use deterministic ordering so prompts are stable for the same repo state.

## Component Changes

### Components to Modify

- `internal/dun/input_resolver.go`: gather ordered inputs for the agent.
- `internal/dun/reconcile_check.go` (new): build the reconcile prompt envelope.
- `internal/dun/engine.go`: add reconcile check to the plan for Helix repos.

### New Components

- Prompt template for reconciliation (Helix plugin prompt).

## Interfaces and Config

- No new CLI flags; surfaced as a Helix check.

## Data and State

- Inputs are read-only doc snapshots; no persistent state required.

## Testing Approach

- Unit tests to ensure reconcile check appears for Helix repos.
- Prompt generation tests to verify ordering and included files.

## Risks and Mitigations

- **Risk**: Large prompts. **Mitigation**: include only key sections and rely
  on summaries for long docs.

## Rollout / Compatibility

- Backwards compatible; check only activates when Helix docs are present.
