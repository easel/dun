# ADR-001: CLI Parsing with Go Standard Library

**Date**: 2026-01-21  
**Status**: Accepted  
**Deciders**: fionn  
**Related Feature(s)**: FEAT-001 (Core CLI)  
**Confidence Level**: Medium  

## Context

Dun needs a CLI interface that is fast, portable, and predictable. The initial
command surface is small (`check`, `list`, `explain`) and must support stable
output for agent loops. We want to keep dependency surface minimal and startup
time low while preserving room to grow if the CLI becomes more complex.

### Problem Statement
Choose a Go CLI parsing approach that balances simplicity, portability, and
maintainability without over-engineering the MVP.

### Current State
No CLI implementation exists yet.

### Requirements Driving This Decision
- Minimal dependencies for a portable binary.
- Deterministic output and help text.
- Support for a small number of subcommands.

## Decision

We will implement the CLI using Go's standard library `flag` package with
explicit `flag.FlagSet` instances per subcommand.

### Key Points
- Keep the dependency tree small and stable.
- Implement subcommands manually to retain control over usage text.
- Revisit if CLI complexity or UX needs outgrow stdlib.

## Alternatives Considered

### Option 1: spf13/cobra
**Description**: Full-featured CLI framework with subcommands and completion.

**Pros**:
- Rich CLI UX (completion, help, nested commands).
- Commonly used in Go CLIs.

**Cons**:
- Extra dependency surface and slower startup.
- More complex initialization for small command sets.

**Evaluation**: Rejected for MVP due to weight and complexity.

### Option 2: kong
**Description**: Declarative CLI framework with struct tags.

**Pros**:
- Clean declarative CLI definition.
- Good help output.

**Cons**:
- Dependency overhead.
- Indirect control over formatting and determinism.

**Evaluation**: Rejected for MVP due to dependency overhead.

### Option 3: Go `flag` package (Selected)
**Description**: Standard library flag parsing with manual subcommand routing.

**Pros**:
- Zero external dependencies.
- Predictable behavior and output.

**Cons**:
- Manual subcommand plumbing.
- Less polished UX by default (no completion).

**Evaluation**: Chosen for MVP to maximize simplicity and portability.

## Consequences

### Positive Consequences
- Small dependency surface and fast startup.
- Full control over command behavior and output.
- Easy to audit and maintain.

### Negative Consequences
- More manual code to manage subcommands.
- Help output must be curated by hand.

### Neutral Consequences
- CLI complexity can be revisited later with minimal lock-in.

## Implementation Impact

### Development Impact
- **Effort**: Low
- **Time**: 1-2 days for basic command routing
- **Skills Required**: Standard Go CLI patterns

### Operational Impact
- **Performance**: Fast startup, minimal overhead
- **Scalability**: Adequate for small command sets
- **Maintenance**: Simple, but manual updates to usage text

### Security Impact
- Reduced third-party dependency risk.

## Risks and Mitigation

| Risk | Probability | Impact | Mitigation Strategy |
|------|------------|--------|-------------------|
| CLI grows too complex | Med | Med | Revisit and migrate to cobra if needed |
| Inconsistent help output | Low | Low | Centralize usage strings and examples |

## Dependencies

### Technical Dependencies
- Go standard library `flag` package.

### Decision Dependencies
- None.

## Validation

### How We'll Know This Was Right
- CLI commands are easy to add and maintain.
- Startup time remains fast with minimal dependencies.
- Users can execute `dun check` without confusion.

### Review Triggers
This decision should be reviewed if:
- CLI complexity expands beyond a few subcommands.
- Users request shell completion or advanced UX features.
- Maintenance cost of manual parsing becomes high.

## References

### Internal References
- `docs/design/contracts/API-001-dun-cli.md`
- `docs/PRD.md`

### External References
- Go `flag` package: https://pkg.go.dev/flag

## Notes

### Future Considerations
If the CLI surface grows, evaluate cobra or kong and provide a migration plan
that preserves output compatibility.

---

## Decision History

### 2026-01-21 - Initial Decision
- Status: Accepted
- Author: fionn
- Notes: MVP uses stdlib for portability and determinism.

---
*This ADR documents a significant architectural decision and its rationale for future reference.*
