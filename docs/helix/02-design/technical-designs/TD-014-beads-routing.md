---
dun:
  id: TD-014
  depends_on:
  - SD-014
---
# TD-014: Beads Work Routing

## Goal

Implement Beads-aware routing in Dun prompts and provide explicit instructions
for retrieving bead details.

## Implementation Plan

### 1) Routing Prompt Enhancements

**File**: `cmd/dun/main.go`

- Add helper functions:
  - `collectBeadsCandidates(checks []dun.CheckResult) []beadsCandidate`
  - `renderBeadsCandidates(w io.Writer, candidates []beadsCandidate)`
- Use `beads-suggest` issues first, fallback to `beads-ready` issues.
- Limit to 3 candidates.

Pseudo-structure:

```go
type beadsCandidate struct {
    ID      string
    Summary string
}

func collectBeadsCandidates(checks []dun.CheckResult) []beadsCandidate {
    // prefer beads-suggest, fallback to beads-ready
}
```

### 2) Beads Work Detail Prompt

**File**: `internal/dun/beads_checks.go`

Update `runBeadsSuggestCheck()` to embed instructions:

```
Work on this bead: <id> - <title>

To get details:
- bd show <id>
- bd comments <id>
```

### 3) Tests

- `cmd/dun/main_test.go`:
  - Ensure routing prompt includes Beads candidate section when beads results
    are present.
- `internal/dun/beads_checks_test.go`:
  - Ensure `beads-suggest` prompt includes `bd show <id>`.

## Rollout

- No new flags required; behavior activates only when beads checks return
  candidates.
- Output is additive and does not alter check execution semantics.
