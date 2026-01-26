# Feature Registry

This registry tracks the core Dun features and their current status.

| ID | Feature | Priority | Status | Notes |
| --- | --- | --- | --- | --- |
| F-001 | Auto-discovery of repo tooling | P0 | In progress | Detect checks from repo signals |
| F-002 | Deterministic check plan | P0 | In progress | Stable ordering and IDs |
| F-003 | Prompt-as-data output | P0 | In progress | Default output format for agent loops |
| F-004 | LLM/JSON outputs | P0 | In progress | Human-readable summaries and JSON |
| F-005 | Helix plugin (docs + gates) | P0 | In progress | Validate Helix artifacts and gates |
| F-006 | Plugin manifest system | P0 | In progress | Extensible check definitions |
| F-007 | Git hygiene + hooks | P0 | In progress | Ensure clean working tree + hook checks |
| F-008 | Doc/code reconciliation | P0 | Planned | Detect drift and propagate updates |
| F-009 | Automation slider | P0 | Planned | Manual ↔ yolo execution policy |
| F-010 | Install command | P1 | In progress | Insert AGENTS.md guidance |
| F-011 | Changed-only checks | P1 | Planned | Scope checks to git diff |
| F-012 | Quality ratchet | P1 | Planned | Baseline compare to prevent regressions |
| F-013 | External plugin loading | P2 | Planned | Load plugin dirs via config |
| F-014 | Go quality checks | P0 | In progress | Tests, coverage, vet, staticcheck |
