# Feature Registry

This registry tracks the core Dun features and their current status.

| ID | Feature | Priority | Status | Notes |
| --- | --- | --- | --- | --- |
| F-001 | Auto-discovery | P0 | In progress | Detect checks from repo signals |
| F-002 | Output Formats | P0 | In progress | Prompt-as-data, LLM/JSON outputs |
| F-003 | Plugin System | P0 | In progress | Extensible check definitions |
| F-004 | Install Command | P1 | In progress | Insert AGENTS.md guidance |
| F-005 | Git Hygiene | P0 | In progress | Ensure clean working tree + hook checks |
| F-006 | Doc Reconciliation | P0 | In progress | Detect drift and propagate updates |
| F-007 | Automation Slider | P0 | In progress | Manual <-> yolo execution policy |
| F-008 | Deterministic check plan | P0 | Planned | Stable ordering and IDs (no spec file yet) |
| F-009 | Helix plugin (docs + gates) | P0 | Planned | Validate Helix artifacts and gates (no spec file yet) |
| F-010 | Changed-only checks | P1 | Planned | Scope checks to git diff (no spec file yet) |
| F-011 | Quality ratchet | P1 | Planned | Baseline compare to prevent regressions (no spec file yet) |
| F-012 | External plugin loading | P2 | Planned | Load plugin dirs via config (no spec file yet) |
| F-013 | Reserved | - | Planned | (no spec file yet) |
| F-014 | Go Quality Checks | P0 | In progress | Tests, coverage, vet, staticcheck |
| F-016 | Doc DAG + Review Stamps | P0 | Planned | Frontmatter DAG, stale detection, stamp command |
| F-017 | Autonomous iteration loop | P0 | Planned | `dun check --prompt` + `dun loop` |
| F-018 | Agent quorum | P1 | Planned | Quorum-based loop decisions |
| F-019 | Installer + self-updater | P1 | Planned | One-line install + `dun update` |
| F-020 | Generic command checks | P1 | Planned | Command checks + external plugin loading |
| F-021 | Beads work routing | P1 | Planned | Surface ready Beads tasks in prompts |
| F-023 | Check domain model | P1 | In progress | Registry-based check lifecycle, summaries, scoring |
