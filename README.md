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
dun check --config .dun/config.yaml
dun check --changed
dun list
dun explain <check-id>
dun respond --id <check-id> --response -
```

## Configuration

Dun reads `.dun/config.yaml` in the repo root when present. CLI flags always override
config values. The default automation mode is `auto`.

Example (`.dun/config.yaml`):

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

## Check Types

Dun supports various check types, each with specific configuration options.

### Built-in Checks

#### Go Quality Checks

```yaml
checks:
  - id: go-test
    type: go-test
    description: Run Go tests

  - id: go-coverage
    type: go-coverage
    description: Check test coverage

  - id: go-vet
    type: go-vet
    description: Run go vet

  - id: go-staticcheck
    type: go-staticcheck
    description: Run staticcheck
```

#### Git Hygiene Checks

```yaml
checks:
  - id: git-status
    type: git-status
    description: Check git working tree status

  - id: git-no-changes
    type: git-status
    description: Ensure no uncommitted changes
```

#### Helix Workflow Checks

```yaml
checks:
  - id: helix-gates
    type: gates
    description: Verify Helix phase gates
    gate_files:
      - docs/helix/01-frame/prd.md
      - docs/helix/02-design/architecture.md

  - id: helix-state
    type: state-rules
    description: Check state transition rules
    state_rules: docs/helix/state-rules.yaml
```

#### Beads Integration

```yaml
checks:
  - id: beads-ready
    type: beads-ready
    description: Check if beads are ready

  - id: beads-suggest
    type: beads-suggest
    description: Suggest next bead to work on
```

### Generic Command Checks

Execute arbitrary shell commands with flexible output parsing.

```yaml
checks:
  - id: eslint
    type: command
    command: npx eslint src/ --format json
    parser: json              # text|lines|json|json-lines|regex
    success_exit: 0           # Exit code for pass
    warn_exits: [1]           # Exit codes for warn
    timeout: 5m               # Duration string
    shell: sh -c              # Default shell
    env:
      NODE_ENV: test
    issue_path: $.errors      # JSONPath for issues array
    issue_fields:
      file: filename
      line: line
      message: message
      severity: severity
```

**Parser Types:**

| Parser | Description | Issue Extraction |
|--------|-------------|------------------|
| `text` | Raw output as detail | None |
| `lines` | Each line becomes an issue | Line text as summary |
| `json` | Parse JSON output | Via `issue_path` and `issue_fields` |
| `json-lines` | Newline-delimited JSON | Same as json, per line |
| `regex` | Regex with named groups | Groups: `file`, `message`, `id` |

**Regex Example:**

```yaml
checks:
  - id: grep-todos
    type: command
    command: grep -rn TODO src/
    parser: regex
    issue_pattern: '(?P<file>[^:]+):(?P<line>\d+):(?P<message>.*)'
```

### Spec-Enforcement Checks

#### Spec-Binding

Verify bidirectional references between specifications and code.

```yaml
checks:
  - id: spec-binding
    type: spec-binding
    bindings:
      specs:
        - pattern: "docs/specs/*.md"
          implementation_section: "## Implementation"
          id_pattern: "FEAT-\\d+"
      code:
        - pattern: "internal/**/*.go"
          spec_comment: "// Implements: FEAT-"
    binding_rules:
      - type: bidirectional-coverage
        min_coverage: 0.8
      - type: no-orphan-specs
        warn_only: true
      - type: no-orphan-code
```

#### Change-Cascade

Detect when upstream changes require downstream updates.

```yaml
checks:
  - id: change-cascade
    type: change-cascade
    trigger: git-diff          # git-diff|always
    baseline: HEAD~1
    cascade_rules:
      - upstream: "docs/specs/*.md"
        downstreams:
          - path: "internal/**/*.go"
            sections: ["implementation"]
            required: true
          - path: "docs/design/*.md"
            required: false
```

#### Integration-Contract

Verify component contracts and dependencies.

```yaml
checks:
  - id: integration-contract
    type: integration-contract
    contracts:
      map: docs/integration-map.yaml
      definitions: "internal/interfaces/*.go"
    contract_rules:
      - type: all-providers-implemented
      - type: all-consumers-satisfied
      - type: no-circular-dependencies
```

**Integration Map Format:**

```yaml
# docs/integration-map.yaml
components:
  auth-service:
    provides:
      - name: Authenticator
        definition: internal/interfaces/auth.go
    consumes:
      - name: UserStore
        from: user-service
  user-service:
    provides:
      - name: UserStore
        definition: internal/interfaces/user.go
```

#### Conflict-Detection

Detect multi-agent work overlap via claim tracking.

```yaml
checks:
  - id: conflict-detection
    type: conflict-detection
    tracking:
      manifest: .dun/work-in-progress.yaml
      claim_pattern: "// CLAIMED:"
    conflict_rules:
      - type: no-overlap
        scope: function        # file|function|line
        required: true
      - type: claim-before-edit
        required: false
```

**WIP Manifest Format:**

```yaml
# .dun/work-in-progress.yaml
claims:
  - agent: agent-1
    claimed_at: 2024-01-15T10:00:00Z
    files:
      - path: internal/auth/handler.go
        scope: file
  - agent: agent-2
    claimed_at: 2024-01-15T10:05:00Z
    files:
      - path: internal/auth/handler.go
        scope: function
        function: ValidateToken
```

#### Agent-Rule-Injection

Dynamically inject rules into agent prompts.

```yaml
checks:
  - id: agent-rule-injection
    type: agent-rule-injection
    base_prompt: prompts/code-review.md
    inject_rules:
      - source: docs/coding-standards.md
        section: "## Guidelines"
      - source: from_registry
        section: "## Project Rules"
    enforce_rules:
      - id: require-tests
        pattern: "test.*added|coverage.*increased"
        required: true
      - id: security-review
        pattern: "security.*reviewed"
        required: false
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
