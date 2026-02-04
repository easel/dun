---
dun:
  id: F-022
  depends_on:
    - helix.prd
---
# Feature Spec: F-022 Doctor Command

## Summary

Provide a `dun doctor` command that checks harness/tool availability, performs
basic harness liveness pings, and reports actionable fixes when dependencies are
missing. Persist a cache of available harnesses for quorum defaults.

## Requirements

- Provide a `dun doctor` CLI command with no flags.
- Detect installed harness CLIs (codex, claude, gemini, opencode).
- Perform a short liveness ping per available harness.
- Report model info when the harness provides it.
- Detect project helpers based on repo signals:
  - Go: `go`, `go tool cover`, `staticcheck`, `govulncheck`, `gosec`.
  - Git: `git`, and configured hook tools (lefthook or pre-commit).
  - Beads: `bd` when `.beads` exists.
- Provide actionable next steps when tools are missing.
- Write `~/.dun/harnesses.json` with liveness results for quorum defaults.
- Do not fail when tools are missing; only fail on runtime errors (for example,
  unable to write the cache file).

## Non-Goals

- Installing or configuring tools automatically.
- Deep validation of tool configuration beyond presence and liveness.

## Acceptance Criteria

- `dun doctor` prints harness availability and liveness status.
- `dun doctor` prints helper/tool availability with actionable hints for missing tools.
- `dun doctor` writes/updates `~/.dun/harnesses.json`.
- Missing tools produce warnings but do not return a non-zero exit code.
