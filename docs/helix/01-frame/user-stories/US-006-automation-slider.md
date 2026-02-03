---
dun:
  id: US-006
  depends_on:
  - F-007
---
# US-006: Control Autonomy With an Automation Slider

As a maintainer, I want to choose how much autonomy Dun has so I can require
review when needed or let it run freely.

## Acceptance Criteria

- I can set automation mode via a CLI flag.
- I can set a default automation mode via `.dun/config.yaml`.
- Manual mode requires approval for each suggestion.
- Yolo mode allows Dun to fill in missing artifacts to declare completeness.
