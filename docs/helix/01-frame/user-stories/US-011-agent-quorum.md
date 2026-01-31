# US-011: Use Agent Quorum for High-Confidence Decisions

As a maintainer, I want to run tasks through multiple agents and require
consensus so I can increase confidence in autonomous changes.

## Acceptance Criteria

- `dun loop --quorum 2` runs each task through multiple harnesses.
- Quorum strategies: majority, unanimous, any.
- Results are compared for agreement before applying changes.
- Conflicts are logged and can escalate to human review.
- Quorum can mix harnesses: `--harness claude,gemini,codex`.
- Performance mode runs harnesses in parallel.
- Cost mode runs harnesses sequentially, stopping on first agreement.

## Example Usage

```bash
# Require 2 of 3 agents to agree
dun loop --harness claude,gemini,codex --quorum 2

# Require unanimous agreement (all must match)
dun loop --harness claude,gemini --quorum unanimous

# Any agent can proceed (fastest, lowest confidence)
dun loop --harness claude,gemini --quorum any
```

## Quorum Strategies

| Strategy | Behavior |
|----------|----------|
| `any` | First response wins |
| `majority` | >50% must agree |
| `unanimous` | All must agree |
| `N` (number) | At least N must agree |

## Conflict Resolution

When agents disagree:
1. Log all responses with diffs
2. If `--escalate`, pause for human review
3. If `--prefer <harness>`, use that agent's response
4. Otherwise, skip task and continue

## Design Notes

- Compare responses semantically, not byte-for-byte
- Track agent agreement rates over time
- Consider cost: 3-harness quorum = 3x API cost
- Useful for high-risk changes (security, data migrations)
