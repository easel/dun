---
dun:
  id: IP-013-doc-dag
  depends_on:
  - SD-013
  - F-016
  - US-013
  - TP-013
  - helix.prd
  review:
    self_hash: ''
    deps: {}
---
# IP-013: Doc DAG + Review Stamps Implementation Plan

## Goal Summary

- Implement a frontmatter-driven doc DAG that detects missing/stale docs using
  review stamps (no git/mtime).
- Resolve dynamic inputs deterministically (node/refs/code_refs/paths).
- Provide prompt envelopes that require related requirements/ADRs/code context
  and a Gaps & Conflicts section.

## Related Requirements / ADRs / Code

### Requirements

- F-016 Doc DAG + Review Stamps (`docs/helix/01-frame/features/F-016-doc-dag.md`)
- US-013 Doc DAG With Review Stamps (`docs/helix/01-frame/user-stories/US-013-doc-dag.md`)
- TP-013 Doc DAG + Review Stamps (`docs/helix/03-test/test-plans/TP-013-doc-dag.md`)
- PRD (`docs/helix/01-frame/prd.md`) and architecture (`docs/helix/02-design/architecture.md`)

### ADRs

- ADR-007 Frontmatter YAML Canonicalization for Doc Hashing
  (`docs/helix/02-design/adr/ADR-007-frontmatter-serialization.md`)

### Code (current state)

- Check orchestration: `internal/dun/engine.go`
- Existing cascade logic (reference): `internal/dun/change_cascade.go`
- Prompt envelopes + input loading: `internal/dun/agent.go`
- Helix checks: `internal/plugins/builtin/helix/plugin.yaml`

## Gaps & Conflicts

- Helix lacks a standard implementation-plan prompt template; this plan will
  introduce one for Doc-DAG only.

## Implementation Steps

1. **Define frontmatter model + parser**
   - Files: `internal/dun/frontmatter.go`, `internal/dun/frontmatter_test.go`
   - Parse `dun` block: id, depends_on, prompt, inputs, review stamps.
   - Ensure stable serialization for hashing per ADR-007 (exclude
     `dun.review`).

2. **Implement doc hashing**
   - Files: `internal/dun/hash.go`, `internal/dun/hash_test.go`
   - Hash includes canonicalized frontmatter (minus `dun.review`) + body,
     normalized newlines.
   - Invalid frontmatter blocks hashing and reports an issue.

3. **Add deterministic input resolver**
   - Files: `internal/dun/input_resolver.go`, `internal/dun/input_resolver_test.go`
   - Support selectors: `node:`, `refs:`, `code_refs:`, `paths:`
   - `node:<id>` resolves to the doc path for `<id>`.
   - `refs:<id>` expands to the doc paths in `<id>`'s `dun.depends_on`.
   - `code_refs:<id>` expands to code paths referenced in `<id>` (backticks or
     Markdown links).
   - `paths:<glob>` expands glob patterns to file paths.
   - Resolve IDs using optional graph `id_map` and frontmatter registry.
   - Sort + dedupe inputs for stable prompts.
   - Unknown selectors, unresolved IDs, and unmatched globs produce issues.

4. **Load optional graph defaults (required roots + id_map)**
   - Files: `internal/dun/doc_dag.go`
   - Parse `.dun/graphs/*.yaml` into required roots, id_map, default prompts.

5. **Build Doc DAG + staleness detection**
   - Files: `internal/dun/doc_dag.go`, `internal/dun/doc_dag_test.go`
   - Determine missing required nodes and stale descendants from review stamps.
   - Docs without `dun.review` or `dun.review.deps` are treated as stale.
   - Emit issues for `missing:<id>` and `stale:<id>`.

6. **Emit prompt envelopes for missing/stale docs**
   - Files: `internal/dun/doc_dag.go`, `internal/dun/agent.go`
   - Use prompt from frontmatter or graph default.
   - Inputs default to parents unless overridden.
   - Ensure prompt requires “Gaps & Conflicts” section.

7. **Add `dun stamp` command**
   - Files: `internal/dun/stamp.go`, `internal/dun/stamp_test.go`,
     `cmd/dun/main.go`
   - Update `dun.review.self_hash` and `dun.review.deps`.

8. **Helix prompt + graph defaults**
   - Files: `internal/plugins/builtin/helix/prompts/implementation-plan.md`
   - Add `.dun/graphs/helix.yaml` with id_map + required roots for doc DAG.

9. **Wire check type**
   - Files: `internal/dun/engine.go`, `internal/dun/types.go`,
     `internal/plugins/builtin/helix/plugin.yaml`
   - Register new `doc-dag` check in Helix plugin.

10. **Tests (P0 first)**
    - Unit tests: frontmatter parse, hash excludes review, input resolver.
    - Unit tests: invalid selector/inputs, invalid frontmatter.
    - Integration test (day 1): cascade stale detection end-to-end with prompt
      inputs (TC-006 in TP-013).

## Testing Plan

- Follow `docs/helix/03-test/test-plans/TP-013-doc-dag.md`.
- Day 1 integration test: `TestDocDagCascadeStale` in
  `internal/dun/engine_test.go` using fixture `internal/testdata/repos/doc-dag-cascade/`.
- Add unit coverage for invalid selector prefixes, unresolved IDs, and
  unmatched globs.

## Rollout

- Implement and test in isolation.
- Add Helix prompt + graph defaults.
- Re-run `dun check --agent-mode prompt` to confirm new check output.
