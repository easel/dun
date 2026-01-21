# Fionn

Fionn is an agent-friendly quality check runner for codebases. It focuses on
fast, automatic discovery of the right checks and produces summarized,
LLM-friendly output so agents (and humans) can quickly assess whether code works.

## Why

Agents can generate lots of code quickly. The hard part is confidence. Fionn
answers: "What checks should run here?" and "What is the short, actionable
result?" without requiring a long, custom configuration.

## Goals

- Zero-config entrypoint: `fionn check` inspects the repo and runs the right
  checks.
- LLM-friendly summaries: each check emits a short, structured result.
- Fast and portable: a single Go binary with minimal dependencies.
- Extensible: easy to add new discoverers, checks, and reporters.
- Deterministic: stable outputs for the same repo state.

## Non-goals

- Replacing CI. Fionn complements CI by giving fast local feedback.
- Becoming a general build system. It orchestrates checks, not builds.
- Providing a UI. The primary interface is CLI output.

## How It Works (Conceptually)

1. Discoverers scan the repo (files, config, language hints).
2. A plan is built from discovered checks.
3. Runners execute checks in parallel with timeouts.
4. Reporters summarize results in LLM-friendly formats.

## CLI

The minimal entrypoint is:

```bash
fionn check
```

Planned options:

```bash
fionn check --format=llm
fionn check --format=json
fionn check --changed
fionn list
fionn explain <check-id>
```

## LLM-Friendly Output

Each check should emit a short, structured summary. Example:

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

## Extensibility Model

Fionn is designed to be easy to extend. The core types are:

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
- fionn: run `fionn check --format=llm` before summarizing results
```

Hook usage (lefthook-style):

```yaml
pre-push:
  commands:
    fionn:
      run: fionn check --changed
```

## Related Tools

Fionn is a near relative to lefthook, but is designed for agents and dynamic
discovery rather than static hook configuration.

## Status

This repository is the starting point for the design and implementation.
Expect the API and CLI to evolve as the architecture solidifies.
