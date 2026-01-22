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
- **Prompt Renderer**: renders agent prompts.
- **Prompt Emitter**: emits prompt envelopes + callbacks.
- **Agent Runner (optional)**: executes prompts when configured.
- **Reporter**: renders JSON/LLM output.

## Data Flow

1. CLI loads plugins and builds a plan.
2. Rule checks run and emit results.
3. Agent checks emit prompt envelopes (default).
4. Optional auto mode runs the agent and parses responses.

## Deployment

- Single Go binary installed locally.
- Uses local toolchains only.
- No network access by default.

## Risks and Mitigations

- **Slow checks**: timeouts and worker limits.
- **Noisy outputs**: strict output schema and concise summaries.
- **Plugin drift**: versioned manifests and deterministic checks.
