---
dun:
  id: F-004
  depends_on:
    - helix.prd
  review:
    self_hash: f0ee8193702c1900c29140bab9213036a72b8ba2c411b55bfb63c03a23329d3e
    deps:
      helix.prd: 58d3c4be8edb0a0be9d01a3325824c9b350f758a998d02f16208525949c4f1ad
---
# Feature Spec: F-004 Install Command

## Summary

Provide a `dun install` command that seeds AGENTS guidance and default Dun
configuration in a repository.

## Requirements

- Provide a local CLI command named `dun install`.
- Seed AGENTS guidance in the target repo using a marker-delimited template.
- Create `.dun/config.yaml` when missing.
- Be idempotent and safe to re-run.
- Support `--dry-run` to show planned changes without writing.
- Keep output deterministic and agent-friendly.

## Gaps & Conflicts

- The PRD does not mention creating `.dun/config.yaml`; confirm this behavior
  remains desired.
- There is no defined backup or undo behavior for AGENTS edits.
- The install command only targets the repo root; no guidance exists for
  multi-root or monorepo installs.

## Template

`dun install` inserts or updates the following marker-delimited block in
`AGENTS.md`:

```
<!-- DUN:BEGIN -->
- **dun**: Development quality checker with autonomous loop support

  Quick commands:
  - `dun check` - Run all quality checks
  - `dun check --prompt` - Get work list as a prompt (pick ONE task, complete it, exit)
  - `dun loop --harness claude` - Run autonomous loop with Claude
  - `dun loop --harness gemini` - Run autonomous loop with Gemini
  - `dun help` - Full documentation

  Autonomous iteration pattern:
  1. Run `dun check --prompt` to see available work
  2. Pick ONE task with highest impact
  3. Complete that task fully (edit files, run tests)
  4. Exit - the loop will call you again for the next task
<!-- DUN:END -->
```

## Insertion Rules

- If `AGENTS.md` already contains the marker block, replace the block content.
- If `AGENTS.md` contains a `## Tools` header, insert the block immediately
  after the header (preserve existing content).
- Otherwise, append a new `## Tools` section with the block at the end.

## Acceptance Criteria

- `dun install` is available in the CLI and documents its usage.
- Running `dun install` creates `.dun/config.yaml` when missing.
- Running `dun install` results in AGENTS guidance being seeded in the repo
  using the defined template and insertion rules.
- Re-running `dun install` does not change files when no updates are needed.
- `dun install --dry-run` emits the planned steps without writing.
