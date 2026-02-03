---
dun:
  id: F-019
  depends_on:
    - helix.prd
  review:
    self_hash: 0a720495a89abe2b925c716863c720ffa1a677bdc83282659f920a68831f18db
    deps:
      helix.prd: 58d3c4be8edb0a0be9d01a3325824c9b350f758a998d02f16208525949c4f1ad
---
# Feature Spec: F-019 Installer and Self-Updater

## Summary

Define how developers acquire and keep Dun current, including the PRD-scoped
`dun install` repo setup flow and any optional binary installer or self-update
behavior.

## Requirements

- Provide `dun install` to seed AGENTS guidance in repositories (PRD scope).
- Keep install/update behavior deterministic and agent-friendly.
- Operate as a local CLI without hosted execution (PRD non-goal).

## Inputs

- Repository path targeted by `dun install`.
- PRD constraints for local, deterministic CLI behavior.

## Gaps & Conflicts

- Conflict: the PRD only specifies `dun install` for repo setup and does not
  define a binary installer or self-updater; confirm whether self-update is in
  scope for this feature.
- Missing distribution details for the CLI binary (release source, channels,
  supported platforms).
- Missing security and verification requirements (signing, checksums, trust
  roots) for any installer or updater.
- Missing versioning and compatibility policy for updates.
- Overlap with the `dun install` repo-seeding command is ambiguous; confirm
  boundaries between repo setup and binary installation.

## Acceptance Criteria

- `dun install` is available and seeds AGENTS guidance in a repository.
- The feature spec explicitly documents whether a self-updater is in scope and
  names the trusted distribution source if it is.

## Traceability

- Supports the PRD scope for `dun install` and local CLI workflows.
- Supports the PRD goal for deterministic, agent-friendly output.
