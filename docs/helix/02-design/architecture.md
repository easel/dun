# Architecture

## Overview

Dun is a local CLI that discovers applicable checks for a repo, runs them with
bounded resources, and emits deterministic results. Agent checks are emitted as
prompt envelopes by default, with optional auto execution when configured.

## Goals

- Deterministic plans and outputs for a given repo state.
- Fast local feedback loops with bounded runtime.
- Extensible plugin system for new workflows.
- Prompt-as-data output as the default interface for agents.

## Non-Goals

- Replacing CI/CD pipelines.
- Hosted or remote execution.
- Organization-wide policy enforcement.

## System Components

- **CLI Entry**: parses flags and routes commands.
- **Plugin Registry**: loads built-in plugin manifests.
- **Discovery Engine**: activates plugins via repo signals.
- **Planner**: builds deterministic check plan.
- **Rule Engine**: evaluates file-based rules and gates.
- **Check Library**: built-in check types (Go quality, git hygiene, Helix gates).
- **Prompt Renderer**: renders agent prompts.
- **Prompt Emitter**: emits prompt envelopes + callbacks.
- **Agent Runner (optional)**: executes prompts when configured.
- **Reporter**: renders JSON/LLM output.
- **Installer**: writes `AGENTS.md` and optional `.dun/config.yaml` scaffolding.
- **Drift Analyzer**: compares docs and code to identify misalignment.
- **Change Planner**: produces ordered updates across artifacts.
- **Automation Policy**: enforces manual/plan/auto/yolo behavior.

## Data Flow

1. CLI loads plugins and builds a plan.
2. Rule checks run and emit results.
3. Agent checks emit prompt envelopes (default).
4. Optional auto mode runs the agent and parses responses.
5. Drift analyzer and planner identify downstream changes when docs shift.

## Command Flows

### `dun check`

- Discover checks via repo signals.
- Run rule checks and emit prompt envelopes.
- Optionally execute agent prompts based on automation policy.
- Emit deterministic output in the selected format.

### `dun install`

- Detect or create `AGENTS.md`.
- Insert Dun guidance using marker blocks.
- Optionally scaffold `.dun/config.yaml` when requested.

## Deployment

- Single Go binary installed locally.
- Uses local toolchains only.
- No network access by default.

## Risks and Mitigations

- **Slow checks**: timeouts and worker limits.
- **Noisy outputs**: strict output schema and concise summaries.
- **Plugin drift**: versioned manifests and deterministic checks.
- **Over-automation**: automation slider gates agent behavior.

## Traceability

- **Problem Statement**: deterministic, fast local checks directly address inconsistent validation and slow feedback loops.
- **Goals**: auto-discovery (F-001), prompt/LLM/JSON output (F-002), doc
  reconciliation (F-006), Go quality checks (F-014).
- **Scope**: plugin system (F-003), install command (F-004), git hygiene
  (F-005), automation slider (F-007), exit codes (F-015).
- **Personas**: agent operators rely on prompt envelopes and fast feedback;
  engineering leads rely on deterministic gates and exit codes.
- **Success metrics**: time to first output <2s and median runtime <30s validated
  in the test plan; adoption target (5 active repos) tracked operationally;
  drift resolution supported by reconciliation checks (see
  `docs/helix/03-test/test-plan.md`).
