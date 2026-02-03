---
dun:
  id: US-014
  depends_on:
  - F-021
---
# US-014: Beads Work Routing

## User Story

**As an** agent operator,
**I want** Dun to surface ready Beads work in the routing prompt,
**So that** I can pick the highest-impact task with minimal context switching.

## Acceptance Criteria

### AC-1: Beads Detection
- [ ] When `.beads/` exists, Dun attempts to read ready beads.
- [ ] If Beads CLI is missing, Dun skips without failing.

### AC-2: Routing Prompt Candidates
- [ ] Routing prompt lists top ready beads (ID + title).
- [ ] Candidate list is limited (default 3) and deterministic.

### AC-3: Work Detail Instructions
- [ ] Beads work detail prompt tells the agent to run `bd show <id>`.
- [ ] Prompt suggests how to fetch additional context (e.g., comments).

## Technical Notes

- Reuse existing `beads-ready` / `beads-suggest` check results for candidate
  summaries when possible.
- Extend `beads-suggest` prompt text with `bd show` guidance.

## Dependencies

- F-017 Autonomous iteration loop (routing prompt).

## Priority

P1 - Improves agent routing efficiency in Beads-enabled repos.

## Estimation

Small to Medium.
