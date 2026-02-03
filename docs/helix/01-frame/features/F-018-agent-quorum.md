---
dun:
  id: F-018
  depends_on:
    - helix.prd
    - F-017
  review:
    self_hash: c24fc4a26a1069344ccfa3d143953abcecc3a6cb1eb8caff37793ac673070030
    deps:
      F-017: 7bee4b95105a9318d51a42ae77fa37aedbc5f28d82f3cb2a155a97eedf809f8f
      helix.prd: 58d3c4be8edb0a0be9d01a3325824c9b350f758a998d02f16208525949c4f1ad
---
# Feature Spec: F-018 Agent Quorum

## Summary

Add quorum-based decision making to Dun with two surfaces:
- `dun loop --quorum` for per-iteration consensus in repo iteration.
- `dun quorum` / `dun synth` for one-shot multi-agent consensus or synthesis.

## Requirements

- Support quorum flags such as `--quorum any|majority|unanimous|N`.
- Allow multiple harnesses per loop invocation.
- Provide conflict handling (log, escalate, or prefer a harness).
- Support cost mode (sequential) and performance mode (parallel).
- Record agreement metadata for each iteration decision.
- Keep quorum evaluation deterministic for the same repo state and harness set.
- Add a dedicated quorum command for one-shot tasks:
  - `dun quorum` selects a winning response via quorum.
  - `dun synth` is shorthand for `dun quorum --synthesize` and produces a merged result.
- Support per-harness personas via `harness@persona` (persona definitions live in harness/DDX).
- Support synthesis via a meta-harness with its own prompt/model/persona.
- Make loop semantics explicit: `dun loop --quorum` applies quorum to the
  iteration prompt (not per-check prompts).

## Inputs

- Loop output and prompt envelope from F-017 Autonomous Iteration.
- Harness responses per iteration.
- Automation mode selection and harness configuration (from loop config).
- One-shot task prompt (for `dun quorum` / `dun synth`).
- Persona registry (provided by harness/DDX; optionally overridden in config).

## Gaps & Conflicts

- Conflicts: none identified in the provided inputs.
- Missing definition of response equivalence (how two harness outputs are
  compared to determine agreement).
- Missing schema for agreement metadata (where it is recorded and in which
  output format), depends on output format rules (F-002).
- Missing precedence rules for quorum flags vs configuration defaults (depends
  on automation policy precedence in F-007).
- Missing timeout and failure handling rules for partial harness responses
  (e.g., a harness unavailable in parallel mode).
- Dependencies: F-017 loop flow, output formats (F-002), and automation policy
  (F-007).
- Missing boundary definition for persona registry between Dun and DDX/harnesses.

## Quorum Flow

1. Run the loop iteration and render the prompt.
2. Dispatch the prompt to each configured harness (sequential or parallel).
3. Normalize responses and evaluate quorum based on the selected rule.
4. If quorum is met, apply changes and record agreement metadata.
5. If quorum is not met, log the conflict and follow the selected conflict
   handling policy.
6. For one-shot `dun quorum`, return the selected response instead of applying
   repo changes.
7. For `dun synth`, collect drafts and run a synthesis meta-harness to merge
   results into a single output.

## Acceptance Criteria

- `dun loop --harnesses a,b --quorum unanimous` blocks on disagreement.
- `dun loop --harnesses a,b,c --quorum 2` succeeds on two matching responses.
- Conflicts are logged with enough detail for human review.
- Sequential and parallel modes produce the same quorum result for identical
  responses.
- Agreement metadata is emitted deterministically for each iteration.
- `dun quorum --task "Write spec"` returns a single selected response.
- `dun synth --task "Write spec"` returns a merged response created by a
  synthesis meta-harness.
- `dun quorum --harnesses codex@architect,claude@critic --quorum majority`
  passes personas to harnesses via `harness@persona`.

## Traceability

- Supports PRD goals for deterministic, agent-friendly output and fast local
  feedback loops.
- Extends the autonomous iteration loop (F-017) with consensus before applying
  changes.
- Adds a one-shot consensus/synthesis command for agent collaboration workflows.
