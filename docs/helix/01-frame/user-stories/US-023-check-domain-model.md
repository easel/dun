---
dun:
  id: US-023
  depends_on:
    - F-023
  review:
    self_hash: 68b0c2dbf86881914c5acbd5cbe3eefc20eee60ec8e0d3539b97a8ac11a53cfa
    deps:
      F-023: 9b5f839c24e68e270c4166652ec89c5334406b462deefe97ac09582d8ae6263e
---
# US-023: First-Class Check Domain Model

As a maintainer, I want checks to be modeled as first-class domain objects so
Dun can discover, evaluate, summarize, and score them consistently without
special-case logic.

## Acceptance Criteria

- Check definitions are normalized into a common domain structure.
- Check execution is routed through a check type registry.
- Check results include summary and score fields.
- Update/freshness signals are available for checks that detect staleness.
- Existing plugin manifests remain compatible.
