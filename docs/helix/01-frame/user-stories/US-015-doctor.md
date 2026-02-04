---
dun:
  id: US-015
  depends_on:
  - F-022
---
# US-015: Diagnose Tooling Readiness

As a user, I want to know if all supporting tools I need to use Dun are
available, with help fixing them when they are missing.

## Acceptance Criteria

- `dun doctor` runs without flags and produces a human-readable report.
- The report lists harness availability and liveness results.
- The report lists project helpers (Go, Git hooks, Beads) when applicable.
- Missing tools include a clear next step (install hint or command).
- `~/.dun/harnesses.json` is updated after each run.
- Missing tools do not cause `dun doctor` to fail.
