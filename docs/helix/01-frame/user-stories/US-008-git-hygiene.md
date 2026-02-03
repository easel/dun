---
dun:
  id: US-008
  depends_on:
  - F-005
---
# US-008: Keep Git Hygiene and Hook Checks

As an agent operator, I want Dun to flag dirty working trees and run configured
pre-commit hooks so I can commit with confidence.

## Acceptance Criteria

- Dirty working trees are detected via `git status --porcelain`.
- Each dirty path appears as an issue with an actionable next step.
- Lefthook or pre-commit hooks run when configured and installed.
- Missing hook tools produce a warning with a clear next step.
- If no hook configuration exists, hook checks are skipped.
