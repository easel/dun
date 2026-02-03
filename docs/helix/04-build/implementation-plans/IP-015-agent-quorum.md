---
dun:
  id: IP-015-agent-quorum
  depends_on:
  - SD-011
  - TD-011
  - F-018
  - US-011
  - TP-011
  - helix.prd
  review:
    self_hash: ''
    deps: {}
---
# IP-015: Agent Quorum + Synthesis Implementation Plan

## Goal Summary

- Add one-shot quorum commands: `dun quorum` and `dun synth`.
- Clarify and keep `dun loop --quorum` behavior as per-iteration prompt quorum.
- Support per-harness personas (`name@persona`) and a synthesis meta-harness.
- Provide deterministic quorum selection and synthesis output.

## Related Requirements / ADRs / Code

### Requirements

- F-018 Agent Quorum (`docs/helix/01-frame/features/F-018-agent-quorum.md`)
- US-011 Agent Quorum (`docs/helix/01-frame/user-stories/US-011-agent-quorum.md`)
- TP-011 Agent Quorum Test Plan (`docs/helix/03-test/test-plans/TP-011-agent-quorum.md`)
- SD-011 Agent Quorum Solution Design (`docs/helix/02-design/solution-designs/SD-011-agent-quorum.md`)
- TD-011 Agent Quorum Technical Design (`docs/helix/02-design/technical-designs/TD-011-agent-quorum.md`)

### Code (current state)

- Loop command: `cmd/dun/main.go`
- Quorum config + logic: `internal/dun/quorum.go`
- Conflict resolution utilities: `internal/dun/conflict.go`
- Harness execution: `internal/dun/harness.go`

## Gaps & Conflicts

- No `dun quorum` / `dun synth` command surface yet.
- No persona-aware harness selection (`name@persona`) or synthesis meta-harness.
- Similarity threshold flag exists but is not wired to grouping.
- Quorum summary metadata is not emitted in a structured, agent-friendly format.
- Ownership boundary: Dun owns quorum vote and synthesis prompts; agents own
  persona/system prompts via DDX.

## Implementation Steps

1. **Add new command surface**
   - Files: `cmd/dun/main.go`, new `cmd/dun/quorum.go`
   - Implement `dun quorum` with `--task`, `--quorum`, `--harnesses`, `--cost-mode`,
     `--escalate`, `--prefer`, `--similarity`, `--synthesize`, `--synthesizer`.
   - Implement `dun synth` as shorthand for `dun quorum --synthesize`.

2. **Parse personas and synthesizer config**
   - Files: `internal/dun/quorum.go`, `internal/dun/types.go`, `internal/dun/config.go`
   - Add `HarnessSpec` and `SynthSpec` types, allow `name@persona` parsing.
   - Extend config schema: `quorum.harnesses[]`, `quorum.synthesizer`,
     `quorum.synthesize`, `quorum.similarity`.

3. **Wire similarity into response grouping**
   - Files: `internal/dun/conflict.go` (or new semantic comparator file)
   - Implement response normalization + similarity threshold grouping.
   - Ensure deterministic grouping for the same inputs.

4. **Implement one-shot quorum execution**
   - Files: `cmd/dun/quorum.go`, `internal/dun/quorum.go`
   - Run harnesses (parallel or sequential), group responses, resolve quorum.
   - In vote mode, return the selected response.

5. **Implement synthesis mode**
   - Files: `cmd/dun/quorum.go`, `internal/dun/quorum.go`
   - Run harnesses to produce drafts.
   - Invoke synthesis meta-harness with its own prompt/model/persona.
   - Return merged response; surface failures cleanly.

6. **Clarify loop behavior and share engine**
   - Files: `cmd/dun/main.go`
   - Ensure `dun loop --quorum` uses the same quorum engine as `dun quorum`.
   - Document and log that quorum applies to the iteration prompt only.

7. **Emit quorum summary metadata**
   - Files: `internal/dun/quorum.go`, `cmd/dun/quorum.go`, `cmd/dun/main.go`
   - Provide a structured summary (JSON block or `--format json`) including:
     harnesses, personas, strategy, agreements, selected response, rationale.

8. **Tests**
   - Add unit tests for persona parsing, synthesizer parsing, and similarity grouping.
   - Add CLI tests for `dun quorum` and `dun synth` parsing and outputs.
   - Update loop tests to assert per-iteration quorum behavior and summary output.

## Testing Plan

- Follow `docs/helix/03-test/test-plans/TP-011-agent-quorum.md`.
- Add new unit tests:
  - `ParseHarnessSpec` parses `codex@architect` correctly.
  - Synthesis mode selects meta-harness and returns merged output.
  - Similarity threshold affects grouping decisions deterministically.
- Add CLI tests:
  - `dun quorum --task ...` succeeds with quorum selection.
  - `dun synth --task ...` invokes synthesizer and returns merged output.

## Rollout

- Ship `dun quorum` and `dun synth` first (one-shot usage).
- Update `dun loop --quorum` to share the same engine and emit summary output.
- Add persona registry documentation (DDX boundary) in docs and help output.

## Follow-Ups

- Evaluate a `dun plan` command that uses quorum synthesis to draft
  implementation plans from specs.
