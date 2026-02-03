---
dun:
  id: IP-014-beads-routing
  depends_on:
    - F-021
    - US-014
    - TP-014
---
# IP-014: Beads Work Routing

## Goal

Add Beads-aware routing to `dun check --prompt` and `dun loop`, and include
explicit instructions for fetching bead details in the work detail prompt.

## Inputs

- F-021 Beads Work Routing (`docs/helix/01-frame/features/F-021-beads-routing.md`)
- US-014 Beads Work Routing (`docs/helix/01-frame/user-stories/US-014-beads-routing.md`)
- SD-014 Beads Work Routing (`docs/helix/02-design/solution-designs/SD-014-beads-routing.md`)
- TD-014 Beads Work Routing (`docs/helix/02-design/technical-designs/TD-014-beads-routing.md`)
- TP-014 Beads Work Routing (`docs/helix/03-test/test-plans/TP-014-beads-routing.md`)

## Execution Plan

1. **Routing prompt enhancement**
   - Add helper to extract Beads candidates from `beads-suggest` or
     `beads-ready` results in `cmd/dun/main.go`.
   - Render a Beads section ahead of the standard check list.
   - Limit candidates to 3 and include ID + title.

2. **Beads detail instructions**
   - Update `internal/dun/beads_checks.go` to include `bd show <id>` in the
     prompt text for `beads-suggest`.

3. **Tests**
   - Add `printPrompt` test ensuring Beads section appears.
   - Add beads-suggest test ensuring prompt contains `bd show <id>`.

4. **Run tests**
   - `go test ./cmd/dun -run BeadsRouting`
   - `go test ./internal/dun -run BeadsSuggest`

## Completion Criteria

- Routing prompt lists Beads candidates when available.
- Beads work detail prompt provides `bd show` instructions.
- Tests covering new behavior pass.
