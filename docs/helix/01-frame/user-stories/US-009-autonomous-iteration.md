---
dun:
  id: US-009
  depends_on:
  - F-017
---
# US-009: Run Autonomous Iteration Loop

As a developer, I want Dun to present available work and let an agent iterate
autonomously so I can step away while quality issues are resolved.

## Acceptance Criteria

- `dun check --prompt` outputs a compact work list with task IDs and summaries.
- Task IDs include a repo-state hash to prevent reuse across repo changes.
- `dun task <task-id> --prompt` emits the full prompt for the selected task.
- Invalid or stale task IDs return a clear error and non-zero exit code.
- `dun loop` runs an embedded loop calling a configurable agent harness.
- The loop supports multiple harnesses: claude, gemini, codex.
- Each iteration spawns fresh context to prevent drift.
- The loop exits when all checks pass or max iterations is reached.
- Yolo mode passes appropriate flags to the harness for autonomous operation.
- `dun install` adds agent documentation to AGENTS.md explaining the pattern.
- `dun help` documents the check --prompt and loop commands.

## Design Notes

Based on the Ralph Wiggum technique:
- Fresh context each iteration (context is liability)
- Agent picks ONE task per iteration
- Deterministic outer loop, LLM does the work
- Dual exit gate: heuristic (all pass) + explicit signal

See: SPIKE-001-nested-agent-harness.md
