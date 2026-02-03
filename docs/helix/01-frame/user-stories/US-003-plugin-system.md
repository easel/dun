---
dun:
  id: US-003
  depends_on:
  - F-003
---
# US-003: Extend Checks with Plugin Manifests

As an engineering lead, I want Dun to load built-in plugin manifests so
workflow-specific checks are discovered and run consistently.

## Acceptance Criteria

- Built-in plugin manifests are embedded in the binary.
- Plugins activate based on repo signals (paths/globs).
- Supported check types include rule-set, gates, state rules, and agent prompts.
- Check ordering is deterministic.
- The Helix plugin activates when `docs/helix/` exists.
