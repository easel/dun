# US-013: Doc DAG With Review Stamps

As an agent operator, I want Dun to track documentation dependencies via
frontmatter and review stamps so I can quickly see which docs are missing or
stale when upstream requirements change.

## Acceptance Criteria

- A document with a changed parent is flagged as stale until re-stamped.
- Required documents missing from the DAG are reported as missing.
- Dun emits prompts for stale or missing documents with parent context,
  including related requirements, ADRs, and code references.
- If conflicts or gaps are detected, the prompt requires they are flagged
  before proceeding.
