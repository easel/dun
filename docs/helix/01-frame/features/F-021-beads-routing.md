---
dun:
  id: F-021
  depends_on:
    - helix.prd
    - F-017
---
# Feature Spec: F-021 Beads Work Routing

## Summary

Surface Beads-ready work items directly in Dun's routing prompt so agents can
pick the highest-impact task, and include clear instructions for fetching bead
details before implementation.

## Requirements

- Detect Beads-enabled repos via `.beads/` presence.
- Surface top Beads candidates in `dun check --prompt` and `dun loop` routing.
- Degrade gracefully when Beads CLI is unavailable (no hard failure).
- Provide explicit instructions for retrieving bead details in the work
  details prompt (e.g., `bd show <id>`).
- Keep routing output deterministic and succinct (limit candidates).

## Inputs

- Beads CLI output (`bd --json ready`).
- Dun check results and prompt generation pipeline.

## Gaps & Conflicts

- Need to define the number of Beads candidates shown (default to top 3).
- Need to decide where bead detail instructions live in the prompt flow.
- Clarify whether Beads suggestions should override other Dun checks (do not
  override; list alongside).

## Detection

- Beads presence is detected via `.beads/` directory and successful `bd` call.

## Output

- Routing prompt includes a Beads section listing candidate IDs and titles.
- Beads work detail prompt includes commands for fetching full context.

## Acceptance Criteria

- When `.beads/` exists and `bd` is available, routing prompt lists top beads
  (IDs + titles).
- When no beads are ready or `bd` is unavailable, routing prompt remains
  unchanged and does not fail.
- Beads work detail prompt explicitly instructs `bd show <id>` to fetch
  details and context.

## Traceability

- Supports PRD goals for fast, deterministic work routing.
- Reinforces autonomous iteration by surfacing ready tasks directly in the
  prompt loop.
