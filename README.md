# Dun

Dun is an agent-friendly quality check runner for codebases. It focuses on
fast, automatic discovery of the right checks and produces prompt-as-data
output by default so agents (and humans) can quickly assess whether code works.

## Why

Agents can generate lots of code quickly. The hard part is confidence. Dun
answers: "What checks should run here?" and "What is the short, actionable
result?" without requiring a long, custom configuration.

## Goals

- Zero-config entrypoint: `dun check` inspects the repo and runs the right
  checks.
- Prompt-as-data output: prompt envelopes are emitted for agent checks.
- LLM-friendly summaries: each check can emit a short, structured result.
- Fast and portable: a single Go binary with minimal dependencies.
- Extensible: easy to add new discoverers, checks, and reporters.
- Deterministic: stable outputs for the same repo state.

## Non-goals

- Replacing CI. Dun complements CI by giving fast local feedback.
- Becoming a general build system. It orchestrates checks, not builds.
- Providing a UI. The primary interface is CLI output.

## Design Constraints

- Deterministic plans: discovered checks are ordered and identified stably.
- Bounded runtime: per-check timeouts plus a global budget.
- Bounded concurrency: a fixed worker limit to avoid resource spikes.
- Partial results: surface failures and timeouts without hiding other signals.
- Portable defaults: avoid per-project config unless necessary.

## How It Works (Conceptually)

1. Discoverers scan the repo (files, config, language hints).
2. A plan is built from discovered checks.
3. Runners execute checks in parallel with timeouts.
4. Reporters summarize results in LLM-friendly formats.

## CLI

The minimal entrypoint is:

```bash
dun check
```

Planned options:

```bash
dun check --format=prompt
dun check --format=llm
dun check --format=json
dun check --automation=plan
dun check --config dun.yaml
dun check --changed
dun list
dun explain <check-id>
dun respond --id <check-id> --response -
```

## Configuration

Dun reads `dun.yaml` in the repo root when present. CLI flags always override
config values. The default automation mode is `auto`.

Example:

```yaml
version: "1"
agent:
  automation: auto
  mode: prompt
  timeout_ms: 300000
```

## Prompt-as-Data Output

Dun emits prompt envelopes for agent checks by default. Example:

```json
{
  "kind": "dun.prompt.v1",
  "id": "helix-create-architecture",
  "title": "Create architecture doc",
  "summary": "Missing docs/helix/02-design/architecture.md",
  "prompt": "Check-ID: helix-create-architecture\n...",
  "inputs": ["docs/helix/01-frame/prd.md"],
  "callback": {
    "command": "dun respond --id helix-create-architecture --response -",
    "stdin": true
  }
}
```

## LLM-Friendly Output

Each check can emit a short, structured summary when using `--format=llm`.
Example:

```text
check:go-test status:fail duration_ms:421
signal: 1 package failed
detail: pkg/foo TestFoo panicked at foo_test.go:42
next: go test ./pkg/foo -run TestFoo
```

Guidelines:

- One check per block.
- Short signal line (what happened).
- One detail line for context.
- Optional next step command.

## Agent Loop Patterns (Ralph Wiggum Inspired)

Dun is designed to work well inside iterative agent loops (for example, the
Ralph Wiggum technique). The patterns we borrow:

- One repeatable command per iteration (`dun check`).
- Deterministic, compact summaries so loops can detect "all green".
- Explicit `next:` hints to guide the next iteration.
- Encourage escape hatches via loop limits and timeouts.

Example loop usage:

```text
/ralph-loop "Implement feature X. Run `dun check --format=llm --changed` each
iteration. If all checks pass, output <promise>DONE</promise>."
  --completion-promise "DONE" --max-iterations 20
```

## Extensibility Model

Dun is designed to be easy to extend. The core types are:

- Discoverers: detect languages, frameworks, tooling, or repo conventions.
- Checks: declarative definitions (id, command, inputs, timeouts).
- Runners: execute checks and capture output.
- Processors: summarize and classify output (pass/fail/warn/skip).
- Reporters: format results for humans or LLMs.

Adding a new rule should be as simple as:

1. Register a discoverer or check.
2. Implement a small processor that produces a summary.
3. Optionally add a reporter or output format.

## Integration Ideas

Agent helper via `AGENTS.md`:

```text
## Tools
- dun: run `dun check` before summarizing results
```

Hook usage (lefthook-style):

```yaml
pre-push:
  commands:
    dun:
      run: dun check --changed
```

## Related Tools

Dun is a near relative to lefthook, but is designed for agents and dynamic
discovery rather than static hook configuration.

## Status

This repository is the starting point for the design and implementation.
Expect the API and CLI to evolve as the architecture solidifies.
