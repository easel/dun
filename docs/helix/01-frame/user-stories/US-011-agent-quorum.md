---
dun:
  id: US-011
  depends_on:
  - F-018
---
# US-011: Use Agent Quorum for High-Confidence Decisions

As a maintainer, I want to run tasks through multiple agents and require
consensus (or synthesize a merged result) so I can increase confidence in
autonomous changes and produce higher-quality specs.

## Acceptance Criteria

- `dun loop --quorum 2` runs each task through multiple harnesses.
- Quorum strategies: majority, unanimous, any.
- Results are compared for agreement before applying changes.
- Conflicts are logged and can escalate to human review.
- Quorum can mix harnesses: `--harnesses claude,gemini,codex`.
- Performance mode runs harnesses in parallel.
- Cost mode runs harnesses sequentially, stopping on first agreement.
- `dun quorum --task "Write a spec"` returns a selected response.
- `dun synth --task "Write a spec"` returns a merged response via a synthesis
  meta-harness.
- Personas can be specified as `harness@persona` and are passed to the harness
  system prompt layer.
- `dun loop --quorum` applies quorum to the iteration prompt (not per-check).

## Example Usage

```bash
# Require 2 of 3 agents to agree
dun loop --harnesses claude,gemini,codex --quorum 2

# Require unanimous agreement (all must match)
dun loop --harnesses claude,gemini --quorum unanimous

# Any agent can proceed (fastest, lowest confidence)
dun loop --harnesses claude,gemini --quorum any

# One-shot quorum selection for a task
dun quorum --task "Write the quorum spec" \
  --harnesses codex@architect,claude@critic --quorum majority

# One-shot synthesis (merged result)
dun synth --task "Write the quorum spec" \
  --harnesses codex@architect,claude@critic --synthesizer codex@editor
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

In synthesis mode, disagreements are resolved by the synthesis meta-harness
using the configured synthesis prompt.

## Design Notes

- Compare responses semantically, not byte-for-byte
- Track agent agreement rates over time
- Consider cost: 3-harness quorum = 3x API cost
- Useful for high-risk changes (security, data migrations)
- Personas are defined by the harness/DDX; Dun only references names.
