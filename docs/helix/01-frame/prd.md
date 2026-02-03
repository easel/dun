---
dun:
  id: helix.prd
  depends_on: []
---
# PRD: Dun

This PRD is the Helix framing artifact for Dun. It summarizes the product
intent captured in `docs/PRD.md` and anchors the Helix workflow.

## Problem Statement

Agent workflows produce code quickly, but validating quality is inconsistent
and often too slow for tight iteration loops.

## Goals

- One command (`dun check`) that discovers and runs the right checks.
- Deterministic, agent-friendly output that is easy to parse.
- Fast local feedback to keep loops moving.
- Detect documentation drift and propose downstream updates.
- Enforce baseline Go quality checks (tests, coverage, static analysis).

## Scope

In scope:
- Local CLI that discovers checks and emits prompt-as-data output.
- Multiple output formats (prompt envelopes, LLM summaries, JSON output).
- Built-in Helix plugin for doc and gate validation.
- Extensible plugin system for future workflows.
- Doc-to-code reconciliation with plan and yolo modes.
- `dun install` to seed AGENTS guidance in repos.
- Git hygiene and hook checks for clean commits.
- Beads-aware work routing when `.beads/` is present.

Out of scope:
- Replacing CI/CD.
- Remote services or hosted execution.
- Full policy enforcement across organizations.

## Success Metrics

- Time to first output under 2 seconds on typical repos.
- Median run time under 30 seconds on medium repos.
- At least 5 active repos using `dun check` during MVP.
- Doc drift issues resolved within 2 loop iterations on average.

## Users and Personas

- Agent operators who need fast, reliable feedback loops.
- Engineering leads who want consistent quality gates.

## Non-Goals

- Building a general-purpose build system.
- Providing a graphical UI.
