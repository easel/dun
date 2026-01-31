# SPIKE-001: Nested Agent Harness

## Problem Statement

Dun currently has a complete `DETECT → PROMPT → RECEIVE` pipeline but lacks loop closure.
The goal is to enable autonomous operation for hours using:
- Fixed-fee subscriptions (Claude Max)
- Fresh context per iteration (Ralph Wiggum technique)
- LLM-driven prioritization
- Deterministic outer loop

## The Ralph Philosophy

> "Ralph is deterministically bad in an undeterministic world."
> — ghuntley.com/ralph

### Why "Ralph Wiggum"?

The name references the famously incompetent character from *The Simpsons*. The technique embraces deterministic failure—Ralph inherently makes mistakes, but these are **predictable and correctable**. This naming reflects intellectual honesty about AI limitations while celebrating the potential for iterative improvement.

### The Pure Form

```bash
while :; do cat PROMPT.md | claude-code ; done
```

That's it. An infinite loop that:
1. Feeds instructions to Claude
2. Lets Claude work autonomously
3. Spawns fresh context each iteration
4. Repeats

This elegantly simple pattern shipped six repositories overnight at a Y Combinator hackathon. One engineer completed a $50K contract for $297 using this approach.

### PROMPT.md as Tuning Mechanism

The prompt file functions like a guitar's tuning system. When Ralph produces suboptimal results, rather than blaming tools, you modify PROMPT.md to guide behavior more effectively.

**The Playground Analogy:** After Ralph "falls off the slide," you add signage:
- "SLIDE DOWN, DON'T JUMP"
- "LOOK AROUND"
- "ONE TASK AT A TIME"

Each failure informs prompt adjustments. This requires patience and **faith in eventual consistency**.

### Key Insight: Context is Liability

Fresh context each iteration is not a bug—it's the core feature. Context accumulation leads to:
- Drift from original objectives
- Hallucinated state
- Compounding errors

By forcing a fresh start, Ralph stays aligned with the source of truth: the files on disk.

## Research Summary

### Ralph-Claude-Code Patterns (github.com/frankbria/ralph-claude-code)

| Pattern | Description |
|---------|-------------|
| **Fresh Context** | Spawn new Claude each iteration - context is liability |
| **Full Task List** | Show ALL available work, let LLM pick ONE |
| **Dual Exit Gate** | Require BOTH heuristic detection AND explicit `EXIT_SIGNAL` |
| **Circuit Breaker** | Halt after 3+ loops with no progress |
| **Rate Limiting** | 100 calls/hour with countdown wait |
| **Status Block** | Machine-parseable output for loop control |
| **fix_plan.md** | Markdown checkboxes for deterministic progress tracking |
| **Session Continuity** | Optional `--continue` flag, but fresh context preferred |

**Real-world validation:**
- Shipped 6 repositories overnight at Y Combinator hackathon
- One engineer completed $50K contract for $297

### Claude-Flow Patterns (github.com/ruvnet/claude-flow)

| Pattern | Description |
|---------|-------------|
| **Separation** | CLI coordinates, Claude Code executes |
| **Prompt-as-Data** | Agent definitions are markdown + YAML frontmatter |
| **File State** | All state in JSON/SQLite files, not LLM memory |
| **Hooks** | Pre/post lifecycle events for learning |
| **Model Routing** | Tier 1/2/3 based on task complexity |

**Critical insight:** Claude-Flow does NOT run autonomous loops. It's a coordination database + routing advisor. Actual execution delegated to Claude Code's Task tool.

### Comparison

| Aspect | Ralph | Claude-Flow |
|--------|-------|-------------|
| Loop driver | Bash `while true` | None (external) |
| Work detection | git diff + fix_plan.md + LLM status | MCP tool calls |
| Prompt format | Static PROMPT.md | Markdown + YAML frontmatter |
| State | .ralph/ JSON files | .claude-flow/ JSON/SQLite |
| Exit detection | Dual-gate (heuristic + signal) | External orchestrator |
| Autonomy | Full (hours of unattended work) | Coordination only |

### Current Dun State

**Has:**
- `dun check` - detects work via plugin conditions
- `PromptEnvelope` - structured prompt with callback
- `dun respond` - accepts agent responses
- Priority ordering (plugin + check priorities)

**Missing:**
- `dun iterate` - present all work, let LLM choose
- Status block parsing - detect completion/exit signals
- Circuit breaker - prevent runaway loops
- Loop driver - outer bash/script that spawns Claude

## Proposed Architecture

```
┌─────────────────────────────────────────────────────────┐
│                    Outer Loop (bash)                     │
│  while true; do                                          │
│    dun iterate > prompt.md                               │
│    claude -p "$(cat prompt.md)" --allowedTools "..."     │
│    # Claude exits after ONE task                         │
│    dun iterate  # re-evaluate, detect exit condition     │
│  done                                                    │
└─────────────────────────────────────────────────────────┘
         │
         ▼
┌─────────────────────────────────────────────────────────┐
│                    dun iterate                           │
│  1. Run all checks (buildPlan + runCheck)                │
│  2. Collect non-passing checks                           │
│  3. Format as work list with priorities                  │
│  4. Append instruction: "Pick ONE, complete it, exit"    │
│  5. Include status block template for response           │
│  6. Detect exit conditions from previous response        │
└─────────────────────────────────────────────────────────┘
         │
         ▼
┌─────────────────────────────────────────────────────────┐
│                    Claude (fresh context)                │
│  1. See full work list                                   │
│  2. Choose ONE task based on impact                      │
│  3. Complete the task (edit files, run tests)            │
│  4. Output status block:                                 │
│     ---DUN_STATUS---                                     │
│     TASK_COMPLETED: helix-gates                          │
│     STATUS: COMPLETE | IN_PROGRESS | BLOCKED             │
│     EXIT_SIGNAL: true | false                            │
│     NEXT_RECOMMENDATION: "Run tests" | "Continue"        │
│     ---END_DUN_STATUS---                                 │
│  5. Exit                                                 │
└─────────────────────────────────────────────────────────┘
```

## Implementation Plan

### Phase 1: `dun iterate` Command

New command that outputs a prompt for one iteration:

```go
// cmd/dun/main.go
case "iterate":
    return runIterate(args[1:], stdout, stderr)

func runIterate(args []string, stdout, stderr io.Writer) int {
    // 1. Run dun check internally
    result, _ := checkRepo(root, opts)

    // 2. Filter to actionable items (non-pass, or agent prompts)
    var actionable []CheckResult
    for _, check := range result.Checks {
        if check.Status != "pass" || check.Prompt != nil {
            actionable = append(actionable, check)
        }
    }

    // 3. Check for exit conditions
    if len(actionable) == 0 {
        fmt.Fprintln(stdout, "---DUN_ITERATE---")
        fmt.Fprintln(stdout, "STATUS: ALL_PASS")
        fmt.Fprintln(stdout, "EXIT_SIGNAL: true")
        fmt.Fprintln(stdout, "---END_DUN_ITERATE---")
        return 0
    }

    // 4. Format as work list
    printIteratePrompt(stdout, actionable, opts.AutomationMode)
    return 0
}
```

### Phase 2: Status Block Parsing

Parse Claude's response for loop control:

```go
// internal/dun/status.go
type IterationStatus struct {
    TaskCompleted      string `json:"task_completed"`
    Status             string `json:"status"`  // COMPLETE, IN_PROGRESS, BLOCKED
    ExitSignal         bool   `json:"exit_signal"`
    NextRecommendation string `json:"next_recommendation"`
}

func ParseStatusBlock(output string) (*IterationStatus, error) {
    // Look for ---DUN_STATUS--- ... ---END_DUN_STATUS---
    // Parse fields
}
```

### Phase 3: Circuit Breaker

Prevent runaway loops:

```go
// internal/dun/circuit.go
type CircuitBreaker struct {
    State              string  // CLOSED, HALF_OPEN, OPEN
    NoProgressCount    int
    SameErrorCount     int
    LastError          string
}

func (cb *CircuitBreaker) ShouldHalt() bool {
    return cb.State == "OPEN"
}

func (cb *CircuitBreaker) RecordIteration(progress bool, err string) {
    if !progress {
        cb.NoProgressCount++
    } else {
        cb.NoProgressCount = 0
    }

    if cb.NoProgressCount >= 3 {
        cb.State = "OPEN"
    }
}
```

### Phase 4: Loop Driver Script

```bash
#!/bin/bash
# dun-loop.sh

MAX_ITERATIONS=100
RATE_LIMIT=100  # per hour
CALL_COUNT=0
HOUR_START=$(date +%s)

while true; do
    # Rate limiting
    if [[ $CALL_COUNT -ge $RATE_LIMIT ]]; then
        wait_for_next_hour
    fi

    # Get iteration prompt
    PROMPT=$(dun iterate --automation=auto)

    # Check for exit
    if echo "$PROMPT" | grep -q "EXIT_SIGNAL: true"; then
        echo "All work complete. Exiting."
        break
    fi

    # Spawn fresh Claude
    claude -p "$PROMPT" \
        --allowedTools "Edit,Read,Write,Bash(go *),Bash(git add),Bash(git commit)" \
        --output-format json \
        > /tmp/claude-output.json

    CALL_COUNT=$((CALL_COUNT + 1))

    # Parse status block from output
    # Update circuit breaker
    # Brief pause
    sleep 5
done
```

## Iterate Prompt Format

```markdown
# Dun Iteration

## Available Work (pick ONE)

### 1. helix-gates [PRIORITY: HIGH]
Missing exit gates for design phase.
**Inputs:** docs/helix/02-design/**/*.md
**Action:** Create missing gate files

### 2. go-coverage [PRIORITY: MEDIUM]
Coverage at 85%, target is 100%.
**Detail:** Missing tests in internal/dun/agent.go
**Action:** Add tests to reach 100%

### 3. git-status [PRIORITY: LOW]
3 uncommitted files: agent.go, types.go, main.go
**Action:** Review and commit changes

---

## Instructions

1. Review the available work above
2. Pick ONE task that will have the biggest impact
3. Complete that task fully (edit files, run tests, verify)
4. Output your status in this format:

---DUN_STATUS---
TASK_COMPLETED: <check-id>
STATUS: COMPLETE | IN_PROGRESS | BLOCKED
FILES_MODIFIED: <count>
EXIT_SIGNAL: false
NEXT_RECOMMENDATION: <what to do next iteration>
---END_DUN_STATUS---

5. Exit when done with that ONE task

## Current Automation Mode: auto
You may edit files and run commands. Ask if blocked.
```

## Exit Conditions

Dual-gate (Ralph pattern):

1. **Heuristic:** `len(actionable) == 0` (all checks pass)
2. **Explicit:** Claude outputs `EXIT_SIGNAL: true`

Both must be true for clean exit. This prevents:
- Premature exit when work remains
- Infinite loops when Claude thinks it's done but checks fail

## State Persistence

```
.dun/
├── config.yaml           # User config
├── state.json            # Iteration state
│   ├── iteration_count
│   ├── last_task_completed
│   ├── circuit_breaker_state
│   └── exit_signals[]    # Rolling window
├── call_count            # Rate limiting
└── logs/
    └── iteration-N.json  # Per-iteration log
```

## Success Metrics

1. Can run 50+ iterations without human intervention
2. Circuit breaker triggers correctly on no-progress
3. Exit detection works (no premature exits, no infinite loops)
4. Rate limiting prevents API exhaustion
5. Fresh context prevents drift/confusion

## Dun's Ralph Implementation

The key insight: **dun iterate IS the PROMPT.md**

```bash
# Pure Ralph
while :; do cat PROMPT.md | claude-code ; done

# Dun Ralph (dynamic prompt generation)
while :; do dun iterate | claude -p "$(cat -)" ; done
```

Instead of a static PROMPT.md, `dun iterate` dynamically generates the prompt by:
1. Running all checks to detect available work
2. Formatting results as a prioritized task list
3. Including loop control instructions
4. Adding status block template

This gives us the best of both worlds:
- **Ralph's simplicity**: Deterministic outer loop, fresh context
- **Dun's intelligence**: Dynamic work detection, plugin-driven priorities
- **Machine-readable control**: Status blocks enable circuit breakers

### Tuning the Guitar

When dun produces suboptimal results, tune via:

| Problem | Tuning Location |
|---------|-----------------|
| Wrong task priority | Plugin `priority` field in manifests |
| Missing work detection | Add new check conditions |
| Claude makes wrong choices | Improve `dun iterate` prompt format |
| Loop runs too long | Circuit breaker thresholds |
| Rate limiting | `MAX_CALLS_PER_HOUR` in config |

## Open Questions

1. Should `dun iterate` call `dun check` internally or expect pre-computed results?
   - **Proposed**: Call internally for simplicity
2. How to handle BLOCKED status - retry? skip? escalate?
   - **Proposed**: Skip and continue, log for human review
3. Should there be a `--max-iterations` safety limit?
   - **Proposed**: Yes, default 100, configurable
4. How to surface iteration logs to human for review?
   - **Proposed**: `.dun/logs/iteration-N.json` with summary command

## Design Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Fresh context | Yes | Core Ralph principle—context is liability |
| LLM picks task | Yes | Better prioritization than static ordering |
| Dual exit gate | Yes | Prevents premature/infinite loops |
| Circuit breaker | Yes | Safety net for runaway behavior |
| Status blocks | Yes | Machine-parseable loop control |
| Rate limiting | Yes | Respect API limits |

## Why This Approach

### Lessons from Ralph

1. **Fresh context is a feature, not a bug.** Context accumulation causes drift. The Ralph pattern's power comes from forcing Claude to re-read the actual files on disk each iteration.

2. **Deterministic loops are reliable.** A bash `while true` loop will run for hours. No complex orchestration needed.

3. **LLM-driven prioritization beats static ordering.** Showing the full task list and letting Claude pick creates emergent intelligence about what matters most.

4. **Dual-gate exit prevents premature termination.** Requiring both heuristic detection AND explicit EXIT_SIGNAL catches both false positives and false negatives.

### Lessons from Claude-Flow

1. **CLI coordinates, Claude executes.** Separation of concerns makes the system more robust.

2. **State belongs in files, not LLM memory.** JSON/SQLite persistence survives crashes and enables debugging.

3. **Hooks enable learning.** Pre/post events let the system improve over time.

### Why `dun iterate` is Right

Dun already has:
- `dun check` - Work detection ✓
- `PromptEnvelope` - Structured prompts ✓
- `dun respond` - Agent response handling ✓
- Priority ordering - Plugin + check priorities ✓

What's missing is **loop closure**. `dun iterate` fills this gap by:
1. Calling `dun check` internally
2. Formatting results as a work list for Claude
3. Including exit detection in the prompt
4. Outputting machine-parseable status for loop control

The outer loop is literally:
```bash
while :; do dun iterate | claude -p "$(cat -)" ; done
```

This is Ralph's genius applied to dun's strengths.

## Next Steps

1. [ ] Implement `dun iterate` command
2. [ ] Add status block parsing
3. [ ] Add circuit breaker
4. [ ] Create loop driver script
5. [ ] Test with 10-iteration run
6. [ ] Add rate limiting
7. [ ] Test with 50-iteration run
