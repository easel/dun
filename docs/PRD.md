# Product Requirements Document

**Version**: 1.0.0  
**Date**: 2026-01-21  
**Status**: Draft  
**Author**: fionn  

## Executive Summary

Dun is a fast, agent-friendly quality check runner that discovers the right
checks for a repository and emits deterministic, LLM-friendly summaries. It
answers the core question in agent workflows: "Is this code working to our
quality standard?" without requiring heavy configuration or manual wiring.

The product targets developers and agent operators who need reliable feedback
in tight iteration loops. Dun focuses on one command (`dun check`) that runs
locally, integrates into hooks and CI, and supports a quality ratchet to
prevent regressions while encouraging continuous improvement.

Scope: a single portable Go binary with discovery, execution, and reporting.
Timeline: prototype in weeks, MVP in a few months, then ratchet and plugin
hardening.

## Problem Statement

### The Problem
Agent workflows can generate large volumes of code quickly, but verifying that
code meets quality standards is inconsistent, slow, and often manual. Existing
tools require per-repo configuration and produce noisy output that is hard for
LLM loops to interpret. As a result, teams either skip checks or run them too
late, increasing regressions and rework.

### Current State
Quality checks are spread across test runners, linters, static analyzers, and
CI pipelines. Tools like lefthook and pre-commit require explicit configuration.
CI produces raw logs that are not structured for rapid iteration. Teams often
maintain bespoke scripts that are brittle and not reusable.

### Opportunity
Agents make it possible to move faster, but only if feedback is immediate and
actionable. A zero-config, discovery-based tool that provides structured,
deterministic summaries can close the loop and help teams improve quality over
time without heavy setup.

## Goals and Objectives

### Business Goals
1. Provide fast, reliable confidence for agent-driven code changes.
2. Reduce regressions by blocking quality backslides.
3. Enable gradual improvement without disruptive rewrites of pipelines.

### Success Metrics
| Metric | Target | Measurement Method | Timeline |
|--------|--------|-------------------|----------|
| Time to first output | < 2s on typical repos | CLI timing logs | MVP |
| Median run time | < 30s on medium repos | CLI timing logs | MVP |
| Regression rate | 50% reduction in broken builds | CI metrics | 3 months |
| Adoption | 5 active repos using `dun check` | Repo audits | 2 months |

### Non-Goals
- Replacing CI/CD pipelines.
- Acting as a general build system.
- Providing a GUI.
- Enforcing organization policy beyond the local repo scope.

## Users and Personas

### Primary Persona: Agent Operator
**Role**: Staff engineer using coding agents  
**Background**: Works on fast-moving repos with frequent automated changes  
**Goals**:
- Get fast, reliable quality feedback
- Keep agent loops moving without manual triage

**Pain Points**:
- Tooling requires per-repo setup
- CI output is too verbose for loops

**Needs**:
- One command that "just works"
- Deterministic, structured output for automation

### Secondary Persona: Engineering Lead
**Role**: Team lead or release manager  
**Background**: Owns quality standards and merge gates  
**Goals**:
- Prevent regressions without slowing velocity
- Gradually raise standards over time

**Pain Points**:
- Inconsistent enforcement across repos
- Manual tracking of quality debt

**Needs**:
- Clear gating policy and ratchet
- Metrics to show improvement

## Requirements Overview

### Must Have (P0)
1. Auto-discovery of repo language and tooling.
2. Stable, ordered check plan with deterministic IDs.
3. Parallel execution with per-check timeouts and global budget.
4. LLM-friendly and JSON output formats.
5. Exit codes aligned with pass/fail outcomes.

### Should Have (P1)
1. `--changed` mode scoped to affected files.
2. Baseline ratchet to prevent regressions.
3. Minimal config file for overrides.
4. Caching of discovery results where safe.

### Nice to Have (P2)
1. Plugin system for external checks.
2. Remote baseline storage for teams.
3. IDE or editor integrations.

## User Journey

### Primary Flow
1. **Entry Point**: User or agent runs `dun check`.
2. **First Action**: Dun discovers checks and prints a plan summary.
3. **Core Loop**: Checks run, summaries guide fixes, user re-runs.
4. **Success State**: All checks pass; Dun emits a clean summary and exit 0.
5. **Exit**: User commits, pushes, or signals completion in agent loop.

### Alternative Flows
- No checks found: Dun prints guidance on detected repo signals.
- Timeouts: Dun marks checks as timeout with suggested next command.
- Partial failure: Dun reports failures without hiding other results.

## Constraints and Assumptions

### Constraints
- **Technical**: Single static Go binary; no runtime service dependencies.
- **Business**: Small team, short delivery cycles.
- **Legal/Compliance**: Local-only execution; no code exfiltration by default.
- **User**: Repo already has underlying tools (go test, npm, etc).

### Assumptions
- Git is available for changed-file detection.
- Teams accept minimal config for edge cases.
- LLM loops require stable, machine-parseable output.

### Dependencies
- Language toolchains installed locally (Go, Node, Python, etc).
- Shell execution for invoking checks.

## Risks and Mitigation

| Risk | Probability | Impact | Mitigation Strategy |
|------|------------|--------|-------------------|
| Slow checks cause poor UX | Med | High | Timeouts, `--changed`, concurrency limits |
| Noisy output reduces trust | Med | High | Strict output contract, trimming, summaries |
| Discovery misses checks | Med | Med | Expand discoverers, allow config overrides |
| Baseline ratchet too strict | Low | Med | Start in warn mode, gradual promotion |
| Plugin sprawl complicates support | Low | Med | Clear interfaces and versioning |

## Timeline and Milestones

### Phase 1: Prototype (2-3 weeks)
- Basic discovery and `dun check` CLI.
- LLM output format with deterministic ordering.

### Phase 2: MVP (4-6 weeks)
- JSON output and exit-code policy.
- Timeouts, concurrency limits, and `--changed`.

### Phase 3: Ratchet and Extensions (6-8 weeks)
- Baseline ratchet support.
- Plugin and config scaffolding.

### Key Milestones
- 2026-02-04: Prototype CLI and discovery.
- 2026-03-04: MVP with JSON and policy gating.
- 2026-03-25: Ratchet and extension scaffolding.

## Success Criteria

### Definition of Done
- [ ] All P0 requirements implemented
- [ ] Output contract documented and stable
- [ ] Performance targets met on sample repos
- [ ] Initial user docs complete
- [ ] Stakeholder review complete

### Launch Criteria
- [ ] No known P0 defects
- [ ] Median runtime target met
- [ ] Clear onboarding and usage guide

## Appendices

### A. Competitive Analysis
- lefthook: strong hook runner, static config, limited discovery.
- pre-commit: broad ecosystem, but config heavy and verbose.
- CI platforms: comprehensive but slow and not loop-friendly.

### B. Technical Feasibility
Dun can be built as a Go CLI with discoverers and runners that invoke existing
toolchains. The architecture maps cleanly to interfaces and supports extension
without runtime dependencies.

### C. User Research
Initial feedback: teams want one command for agent loops, predictable summaries,
and a way to raise quality standards without blocking early adoption.

### D. Research References (Comparable Tooling)
- lefthook: git hook runner with YAML config. https://github.com/evilmartians/lefthook
- pre-commit: language-agnostic hook framework. https://pre-commit.com/
- lint-staged: run checks on staged files. https://github.com/okonet/lint-staged
- Nx affected: change-aware task execution. https://nx.dev/ci/features/affected
- Bazel test: scalable test execution and caching. https://bazel.build/
- GitHub Actions: CI runner for full pipelines. https://docs.github.com/actions

---
*This PRD is a living document and will be updated as we learn more.*
