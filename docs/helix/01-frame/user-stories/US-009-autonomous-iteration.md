# US-009: Run Autonomous Iteration Loop

As a developer, I want Dun to present available work and let an agent iterate
autonomously so I can step away while quality issues are resolved.

## Acceptance Criteria

- `dun iterate` outputs a work list prompt for an external agent.
- `dun loop` runs an embedded loop calling a configurable agent harness.
- The loop supports multiple harnesses: claude, gemini, codex.
- Each iteration spawns fresh context to prevent drift.
- The loop exits when all checks pass or max iterations is reached.
- Yolo mode passes appropriate flags to the harness for autonomous operation.
- `dun install` adds agent documentation to AGENTS.md explaining the pattern.
- `dun help` documents the iterate and loop commands.

## Design Notes

Based on the Ralph Wiggum technique:
- Fresh context each iteration (context is liability)
- Agent picks ONE task per iteration
- Deterministic outer loop, LLM does the work
- Dual exit gate: heuristic (all pass) + explicit signal

See: SPIKE-001-nested-agent-harness.md
