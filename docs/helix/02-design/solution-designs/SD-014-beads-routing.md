---
dun:
  id: SD-014
  depends_on:
  - F-021
---
# SD-014: Beads Work Routing

## Overview

Surface ready Beads tasks in Dun's routing prompt and provide explicit
instructions for pulling bead details in the work detail prompt.

**User Story**: US-014
**Status**: Planned

## Architecture

### Component Overview

```
+---------------------+     +--------------------+
| Beads Checks        |---->| Routing Prompt     |
| (beads-ready/suggest)|     | (printPrompt)      |
+---------------------+     +--------------------+
          |                          |
          v                          v
+---------------------+     +--------------------+
| Beads CLI (bd)      |     | Beads Work Prompt  |
+---------------------+     +--------------------+
```

### Data Flow

1. `dun check` runs beads checks when `.beads/` exists.
2. `printPrompt` extracts top candidates from beads check results.
3. Routing prompt includes a Beads section (ID + title).
4. `beads-suggest` emits a work detail prompt with `bd show` instructions.

## Proposed Changes

### 1. Routing Prompt Integration

- Add helper in `cmd/dun/main.go` to extract bead candidates from
  `beads-suggest` or `beads-ready` results.
- Render a `## Beads Candidates` section when candidates are available.
- Limit output to 3 candidates for readability.

### 2. Work Detail Prompt Instructions

- Extend `runBeadsSuggestCheck()` to embed:
  - `bd show <id>` for full details
  - `bd comments <id>` for context (optional)

### 3. Graceful Degradation

- If beads checks are `skip`/`pass`, routing prompt remains unchanged.
- No failures when Beads CLI is missing.

## Risks & Mitigations

- **Prompt noise**: limit candidates and keep formatting compact.
- **Missing CLI**: rely on current beads check behavior (skip, no failure).

## Acceptance Criteria Mapping

| AC | Implementation | Tests |
|----|----------------|-------|
| AC-1 | Beads checks already handle missing CLI | Existing beads tests |
| AC-2 | printPrompt bead candidates section | New printPrompt test |
| AC-3 | beads-suggest prompt includes `bd show` | New beads prompt test |
