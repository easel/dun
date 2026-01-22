# Assumptions and Constraints

## Assumptions

- Developers have local toolchains installed (Go, Node, etc.).
- Git is available for change detection.
- Repos follow recognizable patterns for discovery (go.mod, package.json, docs/).
- Agents can parse prompt-as-data output reliably.

## Constraints

- Single static Go binary with no runtime service dependencies.
- Local-only execution by default (no network calls).
- Deterministic outputs for a given repo state.
- Bounded runtime via timeouts and concurrency limits.
