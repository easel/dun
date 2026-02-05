---
dun:
  id: IP-023
  depends_on:
    - TD-023
  review:
    self_hash: 4b2a6fc43b11b206e13e208ea61a36f373a9ef8a2c3b6967e676864333c03ee1
    deps:
      TD-023: 9522804907a7c19f06e7868543b17c4fb35218c841953b07a058ac5d8360f6c4
---
# IP-023: Implement Check Domain Model

## Steps

1. Introduce `CheckDefinition`, `CheckSpec`, and a `CheckType` registry.
2. Update engine planning/execution to use the registry.
3. Create typed config structs for each check type and decode from `CheckSpec`.
4. Add summary/score/update fields to `CheckResult` and implement summarizer.
5. Update CLI rendering to prefer summaries and preserve existing output.
6. Update tests and run `go test ./...`.

## Acceptance

- All existing checks execute with no behavior regressions.
- Summary/score fields are present and deterministic.
- Tests pass.
