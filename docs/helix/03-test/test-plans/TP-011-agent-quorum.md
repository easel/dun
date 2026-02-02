# TP-011: Agent Quorum Test Plan

**User Story**: US-011 - Use Agent Quorum for High-Confidence Decisions
**Status**: Planned (Not Yet Implemented)
**Date**: 2026-01-30

## Overview

This test plan defines the verification strategy for the Agent Quorum feature,
which enables running tasks through multiple agent harnesses and requiring
consensus before applying changes.

## Acceptance Criteria (from US-011)

1. `dun loop --quorum 2` runs each task through multiple harnesses
2. Quorum strategies supported: majority, unanimous, any
3. Results are compared for agreement before applying changes
4. Conflicts are logged and can escalate to human review
5. Quorum can mix harnesses: `--harness claude,gemini,codex`
6. Performance mode runs harnesses in parallel
7. Cost mode runs harnesses sequentially, stopping on first agreement

## Test Cases

### TC-011-01: Basic Quorum Flag Parsing

**Objective**: Verify `--quorum` flag is parsed correctly with numeric and named values.

| Test | Input | Expected |
|------|-------|----------|
| TC-011-01a | `--quorum 2` | Quorum threshold set to 2 |
| TC-011-01b | `--quorum majority` | Quorum strategy set to majority |
| TC-011-01c | `--quorum unanimous` | Quorum strategy set to unanimous |
| TC-011-01d | `--quorum any` | Quorum strategy set to any |
| TC-011-01e | `--quorum 0` | Error: invalid quorum value |
| TC-011-01f | `--quorum -1` | Error: invalid quorum value |
| TC-011-01g | `--quorum invalid` | Error: unknown quorum strategy |

### TC-011-02: Multiple Harness Execution

**Objective**: Verify tasks run through all specified harnesses.

| Test | Input | Expected |
|------|-------|----------|
| TC-011-02a | `--harness claude,gemini --quorum 2` | Both harnesses invoked |
| TC-011-02b | `--harness claude,gemini,codex --quorum 2` | All three harnesses invoked |
| TC-011-02c | `--harness claude --quorum 2` | Error: quorum requires >= 2 harnesses |

### TC-011-03: Quorum Strategy - Any

**Objective**: Verify "any" strategy accepts first response.

| Test | Condition | Expected |
|------|-----------|----------|
| TC-011-03a | First harness responds | Response accepted immediately |
| TC-011-03b | First harness fails, second succeeds | Second response accepted |
| TC-011-03c | All harnesses fail | Task fails with aggregated errors |

### TC-011-04: Quorum Strategy - Majority

**Objective**: Verify "majority" strategy requires >50% agreement.

| Test | Condition | Expected |
|------|-----------|----------|
| TC-011-04a | 2/3 harnesses agree | Agreed response applied |
| TC-011-04b | 3/3 harnesses agree | Agreed response applied |
| TC-011-04c | 1/3 harnesses agree | Conflict detected, no changes applied |
| TC-011-04d | 2/2 harnesses agree | Agreed response applied |
| TC-011-04e | 1/2 harnesses agree, 1/2 different | Conflict detected |

### TC-011-05: Quorum Strategy - Unanimous

**Objective**: Verify "unanimous" strategy requires all harnesses to agree.

| Test | Condition | Expected |
|------|-----------|----------|
| TC-011-05a | 3/3 harnesses agree | Agreed response applied |
| TC-011-05b | 2/3 harnesses agree | Conflict detected |
| TC-011-05c | 1/3 harnesses agree | Conflict detected |
| TC-011-05d | 2/2 harnesses agree | Agreed response applied |

### TC-011-06: Quorum Strategy - Numeric Threshold

**Objective**: Verify numeric quorum requires at least N agreements.

| Test | Condition | Expected |
|------|-----------|----------|
| TC-011-06a | `--quorum 2`, 2/3 agree | Agreed response applied |
| TC-011-06b | `--quorum 2`, 1/3 agree | Conflict detected |
| TC-011-06c | `--quorum 3`, 3/3 agree | Agreed response applied |
| TC-011-06d | `--quorum 3`, 2/3 agree | Conflict detected |

### TC-011-07: Semantic Comparison

**Objective**: Verify responses are compared semantically, not byte-for-byte.

| Test | Condition | Expected |
|------|-----------|----------|
| TC-011-07a | Same content, different whitespace | Considered agreement |
| TC-011-07b | Same content, different formatting | Considered agreement |
| TC-011-07c | Same meaning, minor wording differences | Configurable threshold |
| TC-011-07d | Substantively different content | Conflict detected |

### TC-011-08: Conflict Logging

**Objective**: Verify conflicts are logged with all responses and diffs.

| Test | Condition | Expected |
|------|-----------|----------|
| TC-011-08a | Conflict detected | All responses logged |
| TC-011-08b | Conflict detected | Diffs between responses shown |
| TC-011-08c | Conflict detected | Harness names identified in logs |

### TC-011-09: Conflict Resolution - Escalate

**Objective**: Verify `--escalate` pauses for human review.

| Test | Condition | Expected |
|------|-----------|----------|
| TC-011-09a | Conflict with `--escalate` | Execution pauses |
| TC-011-09b | Conflict with `--escalate` | User prompted for decision |
| TC-011-09c | User selects response | Selected response applied |
| TC-011-09d | User skips | Task skipped, loop continues |

### TC-011-10: Conflict Resolution - Prefer

**Objective**: Verify `--prefer <harness>` uses specified harness on conflict.

| Test | Condition | Expected |
|------|-----------|----------|
| TC-011-10a | Conflict with `--prefer claude` | Claude's response applied |
| TC-011-10b | Conflict with `--prefer gemini` | Gemini's response applied |
| TC-011-10c | `--prefer invalid` | Error: unknown harness |
| TC-011-10d | Preferred harness failed | Error logged, task skipped |

### TC-011-11: Conflict Resolution - Default Skip

**Objective**: Verify default behavior skips conflicted tasks.

| Test | Condition | Expected |
|------|-----------|----------|
| TC-011-11a | Conflict, no escalate/prefer | Task skipped |
| TC-011-11b | Conflict, no escalate/prefer | Loop continues to next task |
| TC-011-11c | Conflict, no escalate/prefer | Conflict logged |

### TC-011-12: Performance Mode - Parallel Execution

**Objective**: Verify harnesses run in parallel in performance mode.

| Test | Condition | Expected |
|------|-----------|----------|
| TC-011-12a | Default (performance mode) | Harnesses invoked concurrently |
| TC-011-12b | 3 harnesses, each takes 1s | Total time ~1s, not 3s |
| TC-011-12c | One harness slow, others fast | Fast responses available immediately |

### TC-011-13: Cost Mode - Sequential Execution

**Objective**: Verify cost mode runs harnesses sequentially with early exit.

| Test | Condition | Expected |
|------|-----------|----------|
| TC-011-13a | `--cost-mode`, quorum 2, first 2 agree | Third harness not invoked |
| TC-011-13b | `--cost-mode`, quorum 2, first 2 disagree | Third harness invoked |
| TC-011-13c | `--cost-mode`, unanimous, first disagrees | Remaining harnesses not invoked |

### TC-011-14: Agreement Tracking

**Objective**: Verify agent agreement rates are tracked over time.

| Test | Condition | Expected |
|------|-----------|----------|
| TC-011-14a | Multiple tasks with quorum | Agreement rates recorded |
| TC-011-14b | Query agreement stats | Per-harness agreement rate shown |
| TC-011-14c | High disagreement harness | Warning logged |

### TC-011-15: Edge Cases

**Objective**: Verify edge case handling.

| Test | Condition | Expected |
|------|-----------|----------|
| TC-011-15a | Harness times out | Timeout counted as non-agreement |
| TC-011-15b | Harness returns error | Error counted as non-agreement |
| TC-011-15c | Empty response from harness | Handled as valid response |
| TC-011-15d | Very large response from harness | Comparison handles efficiently |
| TC-011-15e | All harnesses timeout | Task fails with timeout errors |

### TC-011-16: Integration with Loop Command

**Objective**: Verify quorum integrates with existing loop functionality.

| Test | Condition | Expected |
|------|-----------|----------|
| TC-011-16a | `dun loop --quorum 2` with tasks | Each task goes through quorum |
| TC-011-16b | Quorum with `--dry-run` | Shows what would happen |
| TC-011-16c | Quorum with `--verbose` | Detailed quorum logging (prompts + responses) |
| TC-011-16d | Quorum with automation modes | Modes apply per-harness |

## Test Data Requirements

### Fixture Repositories

- `internal/testdata/repos/quorum-basic/` - Simple project for quorum testing
- `internal/testdata/repos/quorum-conflict/` - Project that triggers conflicts

### Mock Harnesses

- Deterministic mock harnesses that return configurable responses
- Mock harnesses with configurable delays for timing tests
- Mock harnesses that return errors or timeouts

## Implementation Notes

When implementing these tests:

1. **Semantic comparison** needs a defined similarity threshold
2. **Agreement tracking** needs persistent storage strategy
3. **Cost mode** needs harness ordering strategy
4. **Performance mode** needs proper concurrency handling
5. **Conflict diffs** need a diff algorithm selection

## Related Documents

- [US-011: Agent Quorum User Story](../../01-frame/user-stories/US-011-agent-quorum.md)
- [Test Plan (main)](../test-plan.md)
