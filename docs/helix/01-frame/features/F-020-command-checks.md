---
dun:
  id: F-020
  depends_on:
    - helix.prd
    - F-003
  review:
    self_hash: 4c66eaea58dbad9471e770aaeb00fbfb3725e958672cedae9b283c5efadddaa8
    deps:
      F-003: 8bbe08567869ffbb1fa3c56eb1af0d585f1918acb50240d7c541a86b4ace030e
      helix.prd: 58d3c4be8edb0a0be9d01a3325824c9b350f758a998d02f16208525949c4f1ad
---
# Feature Spec: F-020 Generic Command Checks

## Summary

Enable shell-command based checks defined in plugin manifests so teams can
extend Dun without writing Go code, while keeping discovery and execution
deterministic.

## Requirements

- Support a `command` check type in plugin manifests.
- Execute configured commands and capture exit status plus trimmed output.
- Emit structured check results compatible with Dun output formats.
- Keep command checks ordered deterministically for a given repo state.
- Activate command checks through the plugin system and repo signals.

## Inputs

- Built-in plugin manifests embedded in the CLI (F-003).
- Repository signals used for plugin activation (F-003).
- Command output from executed checks.

## Gaps & Conflicts

- Conflict: F-003 lists supported check types (rule-set, gates, state-rules,
  agent prompts) but does not include `command`; clarify whether command checks
  are part of that list or a new type.
- Plugin manifest schema for command checks is undefined (fields, validation,
  and defaults).
- Execution environment details are missing: shell selection, working
  directory, environment variables, timeouts, and cancellation behavior.
- Output parsing rules are unspecified (plain text vs structured parsing,
  trimming, size limits).
- External plugin loading (user/project directories) is not mentioned in the
  PRD or F-003; confirm whether this feature should only use embedded manifests.
- Dependency: requires F-003 Plugin System for discovery and execution.
- No conflicts identified in the provided inputs beyond the check-type mismatch.

## Detection

- Command checks activate via their plugin's activation signals, consistent
  with F-003.

## Output

- Each command check emits deterministic `signal` values; `detail` and `next`
  are included when failures or warnings are actionable.

## Acceptance Criteria

- A plugin manifest can define a `command` check that runs and returns a
  structured result.
- Non-zero exit codes create failed check results with captured output.
- Command check ordering is stable across runs for the same repo state.

## Traceability

- Supports PRD goals for one-command discovery/execution and deterministic,
  agent-friendly output.
- Supports PRD scope for an extensible plugin system by enabling command-based
  checks via plugin manifests.
