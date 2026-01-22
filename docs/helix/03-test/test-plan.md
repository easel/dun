# Test Plan

**Project**: Dun  
**Version**: 1.0.0  
**Date**: 2026-01-21  
**Status**: Draft  
**Author**: fionn  

## Executive Summary

This test plan verifies the MVP plugin system and Helix plugin behaviors. The
focus is on deterministic detection, rule evaluation, prompt envelope emission,
and the feedback loop that turns missing artifacts into actionable prompts.

## Testing Strategy

### Scope and Objectives

**Testing Goals**:
- Validate plugin discovery via repo signals (Helix docs detection).
- Ensure rule-based checks identify missing artifacts and gate order.
- Ensure prompt envelopes are emitted for agent checks.
- Ensure `dun respond` parses structured responses.
- Demonstrate the feedback loop for missing docs and alignment checks.
- Demonstrate prompt-default behavior with optional auto mode.
- Validate Helix gate files against required evidence paths.

**Out of Scope**:
- Remote plugin registries or sandboxing.
- Baseline/ratchet storage.
- Complex cross-reference rules beyond MVP.

### Test Levels

| Level | Purpose | Coverage Target | Priority |
|-------|---------|-----------------|----------|
| Contract Tests | Manifest schema + agent response shape | 100% | P0 |
| Integration Tests | Plugin discovery + check execution | 90% | P0 |
| Unit Tests | Rule evaluation and prompt rendering | 80% | P1 |
| E2E Tests | CLI end-to-end runs | Critical paths | P1 |

### Framework Selection

| Test Type | Framework | Justification |
|-----------|-----------|---------------|
| Contract | Go testing | Built-in, deterministic |
| Integration | Go testing | Fits repo-local fixtures |
| Unit | Go testing | Minimal dependencies |
| E2E | Go testing | Exec CLI in temp repos |

## Test Organization

### Directory Structure

```
internal/
  dun/
    engine_test.go
  testdata/
    agent/
    repos/
```

### Naming Conventions

**Test Files**:
- Integration: `engine_test.go`
- Unit: `rules_test.go`, `prompt_test.go`

**Test Cases**:
- Format: `should [expected behavior] when [condition]`

### Test Data Strategy

**Static Data** (Fixtures):
- `internal/testdata/repos/helix-missing-architecture/`
- `internal/testdata/repos/helix-missing-features/`
- `internal/testdata/repos/helix-alignment/`

**External Services** (Mocks):
- Stub agent script in `internal/testdata/agent/agent.sh`

## Coverage Requirements

### Coverage Targets

| Metric | Target | Minimum | Enforcement |
|--------|--------|---------|-------------|
| Line Coverage | 80% | 70% | CI blocks merge |
| Critical Path | 100% | 100% | Required |

### Critical Paths

**P0 - Must Have Coverage**:
1. Helix plugin auto-detection.
2. Missing architecture emits a prompt envelope.
3. Missing feature specs emit a prompt envelope.
4. Alignment check emits a prompt when prerequisites exist.
5. `dun respond` parses structured agent output.
6. State rules detect missing upstream artifacts.
7. Gate checks fail when required evidence is missing.

## Implementation Roadmap

### Phase 1: Test Infrastructure (Day 1)
- [ ] Set up Go module and test structure
- [ ] Create agent stub (auto mode)
- [ ] Create fixture repos

### Phase 2: Contract + Integration Tests (Day 2)
- [ ] Manifest parsing tests
- [ ] Agent response parsing tests
- [ ] End-to-end check runs on fixtures

### Phase 3: Unit Tests (Day 3)
- [ ] Rule evaluation tests
- [ ] Prompt rendering tests

## Test Infrastructure

### Environment Requirements

**Local Development**:
- Go 1.22+
- Bash (for agent stub)

**CI/CD Pipeline**:
- `go test ./...`

### Tools and Dependencies

| Tool | Version | Purpose |
|------|---------|---------|
| Go | 1.22+ | Test runner |
| bash | 4+ | Agent stub |

## Risk Assessment

### Technical Risks

| Risk | Impact | Probability | Mitigation |
|------|--------|-------------|------------|
| Flaky agent output | High | Medium | Deterministic stub |
| Fixture drift | Medium | Medium | Keep fixtures minimal |

## Success Metrics

### Quality Metrics

| Metric | Target | Measurement |
|--------|--------|-------------|
| Test execution time | <1s | `go test ./...` |
| Feedback loop coverage | 100% | Integration tests |

## Handoff to Build Phase

### Deliverables for Build Team

1. **Test Suite** (failing tests ready)
2. **Fixtures** for Helix scenarios
3. **Agent stub** for deterministic agent responses

### Build Phase Integration Points

**Test Execution**:
- Command: `go test ./...`

---

**Sign-off**: Pending
