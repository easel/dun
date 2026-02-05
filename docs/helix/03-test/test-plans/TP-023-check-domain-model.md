---
dun:
  id: TP-023
  depends_on:
    - TD-023
  review:
    self_hash: cc509d37fd92febd897f7565e61a9da76ecba61c834f57c54c3e9e41fb8fbc26
    deps:
      TD-023: 9522804907a7c19f06e7868543b17c4fb35218c841953b07a058ac5d8360f6c4
---
# TP-023: Check Domain Model

## Scope

Verify registry-based execution, typed config decoding, summary/score fields,
update signals, and backward compatibility.

## Acceptance Criteria

| ID | Criteria |
|----|----------|
| AC-1 | Registry dispatch runs all existing check types |
| AC-2 | Typed config decode succeeds for representative check types |
| AC-3 | Summary/score fields are populated and deterministic |
| AC-4 | Update signals appear for stale/missing checks |
| AC-5 | `go test ./...` passes |

## Proposed Tests

- Registry dispatch unit tests per check type.
- Summarizer tests for each status value.
- Golden test for `dun check --format=json` to ensure additive fields only.
