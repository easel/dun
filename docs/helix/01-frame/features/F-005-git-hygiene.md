---
dun:
  id: F-005
  depends_on:
    - helix.prd
  review:
    self_hash: b98c3a3624a6c461688e617b70892cd2bbb0cf1b990ebefdbf350ee1fa8c6cdc
    deps:
      helix.prd: 58d3c4be8edb0a0be9d01a3325824c9b350f758a998d02f16208525949c4f1ad
---
# Feature Spec: F-005 Git Hygiene and Hook Checks

## Summary

Provide built-in checks that ensure the working tree is clean and that any
configured pre-commit hooks run successfully, so Dun can safely recommend
committing after manual approval.

## Requirements

- Detect whether the repo has uncommitted changes.
- Emit actionable guidance when the working tree is dirty.
- Detect common hook configurations (lefthook, pre-commit) without requiring
  those tools as hard dependencies.
- If a hook tool is present, run it with a deterministic, non-interactive
  command.
- If a hook tool is configured but missing, emit a warning with a clear next
  step (install tool or skip).
- Keep the check fast, local-only, and deterministic.

## Gaps & Conflicts

- Exit code behavior for warning-only checks is not defined in the PRD or
  F-015; confirm whether warnings should still return exit code 0.
- Hook support beyond lefthook and pre-commit (for example, husky or
  lint-staged) is not specified.
- The PRD does not specify whether a dirty working tree should be a warning or
  failure; this spec defaults to **warn** so other checks can still run.

## Detection

- Plugin is active when a `.git/` directory exists.
- Hook runners are detected in this order:
  1. `lefthook.yml` or `.lefthook/` + `lefthook` binary present
  2. `.pre-commit-config.yaml` + `pre-commit` binary present

## Check Behavior

### Git Clean Check

- Use `git status --porcelain` to detect uncommitted changes.
- **Pass**: no changes.
- **Warn**: working tree dirty; include a list of changed paths in issues.

### Hook Check (Optional)

- If lefthook is configured and installed, run `lefthook run pre-commit`.
- Else if pre-commit is configured and installed, run
  `pre-commit run --all-files`.
- **Pass**: hook command exits 0.
- **Warn**: hook configuration exists but tool is missing.
- **Skip**: no hook configuration detected.

## Output

- If checks fail: Dun emits actionable guidance on what to fix.
- If only git hygiene remains: Dun instructs the agent to craft a commit
  message describing the changed files and to commit them.
- `git-clean` check includes:
  - `issues`: one per dirty path.
  - `next`: "Create a commit message describing changes in: <files>. Then run
    `git add -A && git commit -m \"<message>\"`."
- `hook-check` includes:
  - `next`: install or run the required hook tool when missing.

## Non-Goals

- Installing hook tools automatically.
- Managing git branches or commits.
- Replacing CI/CD checks.

## Acceptance Criteria

- `dun check` warns when the working tree is dirty and lists changed files.
- The `next` field instructs the agent to craft a commit message describing
  the changed files.
- `dun check` passes when the tree is clean.
- If `lefthook.yml` exists and `lefthook` is available, hooks are executed.
- If `.pre-commit-config.yaml` exists and `pre-commit` is available, hooks are executed.
- If a hook config exists but the tool is missing, Dun warns with a next step.
