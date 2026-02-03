---
dun:
  id: helix.architecture
  depends_on:
    - helix.prd
  inputs:
    - node:helix.prd
    - refs:helix.prd
    - code_refs:helix.prd
  review:
    self_hash: ""
    deps:
      helix.prd: oldhash
---
# Architecture

Initial architecture.
