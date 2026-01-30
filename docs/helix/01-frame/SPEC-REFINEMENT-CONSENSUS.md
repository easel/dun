# Dun Feature Spec Refinement Consensus

Generated from multi-perspective swarm analysis (Codex, Gemini, Claude viewpoints + Consistency Review).

## Executive Summary

The current specs provide a solid foundation but have **critical gaps** in:
1. **Feature ID alignment** - Registry IDs don't match actual files
2. **Determinism claims** - No ordering algorithm specified
3. **Safety concerns** - `git add -A` and hook execution risks
4. **Missing limits** - Timeouts, truncation, exit codes unspecified
5. **Flag interactions** - `--format` and `--automation` behavior undefined

---

## Critical Issues (Must Fix)

### 1. Feature ID Registry Mismatch

**Problem:** Feature Registry IDs don't match actual feature file IDs.

| Registry ID | Registry Name | Actual File ID | Actual Name |
|-------------|--------------|----------------|-------------|
| F-003 | Prompt-as-data output | F-003 | Plugin System |
| F-005 | Helix plugin | F-005 | Git Hygiene |
| F-006 | Plugin manifest | F-006 | Doc Reconciliation |
| F-007 | Git hygiene | F-007 | Automation Slider |

**Resolution:** Renumber feature registry OR rename files to match.

### 2. Determinism Without Algorithm

**Problem:** 5 specs claim "deterministic ordering" but no algorithm is specified.

**Resolution:** Add to F-001:
```markdown
## Check Ordering Algorithm

Checks are ordered by:
1. Plugin load order (alphabetical by plugin name)
2. Phase: discovery → validation → agent
3. Check ID within phase (alphabetical)

This produces stable ordering for the same repo state.
```

### 3. Safety: `git add -A` Suggestion

**Problem:** F-005 suggests `git add -A && git commit` which stages ALL files including:
- Sensitive files (.env, credentials, API keys)
- Build artifacts
- Large binaries

**Resolution:** Change F-005 Output section:
```markdown
- `next`: "Review changed files, then stage with `git add <files>` and commit.
  Avoid `git add -A` which may stage sensitive files."
```

### 4. Safety: Hook Execution Without Consent

**Problem:** F-005 runs hooks automatically, which may execute arbitrary code.

**Resolution:** Add to F-005:
```markdown
## Safety

- First-time hook execution emits a warning: "Running pre-commit hooks. Use --no-hooks to skip."
- Add `--no-hooks` flag to skip hook execution.
- Hook timeout defaults to 30 seconds.
```

### 5. Coverage Default of 100%

**Problem:** 100% coverage default fails nearly all repos on first run.

**Resolution:** Change F-014:
```markdown
- Coverage threshold defaults to 80%.
- Override via `.dun/config.yaml`:
  ```yaml
  go:
    coverage_threshold: 90
  ```
- First run with no config emits suggestion to set threshold.
```

---

## High Priority Refinements

### 6. Flag Interaction: --format + --automation

**Problem:** No spec defines behavior when both flags used.

**Resolution:** Add new section to F-007:
```markdown
## Flag Interactions

| --automation | --format | Behavior |
|--------------|----------|----------|
| any | json | JSON output with automation mode in metadata |
| any | llm | Human text with mode noted at top |
| any | prompt | Prompt envelopes include mode for agent |

`--format` controls output structure. `--automation` controls agent behavior policy.
Both can be used together.
```

### 7. Exit Codes Specification

**Problem:** No exit codes defined anywhere.

**Resolution:** Add to F-002 or new global spec:
```markdown
## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | All checks pass |
| 1 | One or more checks failed |
| 2 | Configuration error |
| 3 | Runtime error (command not found, permission denied) |
| 4 | Internal error (bug in Dun) |
```

### 8. Status Values Standardization

**Problem:** Status uses "pass/fail/warn/skip" but:
- API schema only lists "pass|warn|fail" (no skip)
- F-005 uses Title Case, F-014 uses lowercase
- Semantic difference between warn/fail not documented

**Resolution:** Add to F-002:
```markdown
## Status Values

| Status | Meaning | Blocks CI? |
|--------|---------|------------|
| pass | Check succeeded | No |
| warn | Issue found, actionable | Configurable |
| fail | Check failed, must fix | Yes |
| skip | Check not applicable | No |

All status values are lowercase. Use `--warn-as-error` to fail on warnings.
```

### 9. Timeout Defaults

**Problem:** No timeouts specified.

**Resolution:** Add global limits section:
```markdown
## Default Limits

| Operation | Default | Override |
|-----------|---------|----------|
| Hook execution | 30s | `--hook-timeout` |
| Go test | 5m | `--test-timeout` |
| Discovery scan | 5s | N/A |
| Agent check | 60s | `--agent-timeout` |
```

### 10. "yolo" Mode Rename

**Problem:** "yolo" naming undermines professional trust.

**Resolution:** Rename in F-007:
```markdown
- `manual`: prompt-only, human approval for each change
- `plan`: emit detailed plan without modifying artifacts
- `auto`: agent executes changes but asks when blocked
- `autonomous`: agent may create/modify artifacts without confirmation
  (formerly "yolo")
```

---

## Medium Priority Refinements

### 11. Node.js Detection (F-001)

**Problem:** Spec promises Node detection but no plugin exists.

**Resolution:** Either:
- Remove from F-001: "Detect Node repositories via `package.json` (PLANNED)"
- Or implement the plugin

### 12. Output Format Behavior (F-002)

**Problem:** Implementation treats `--format=prompt` and `--format=json` identically.

**Resolution:** Clarify in F-002:
```markdown
- `--format=json` (default): Structured JSON for all checks
- `--format=prompt`: Alias for json (prompt envelopes are JSON)
- `--format=llm`: Human-readable text summaries
```

### 13. Skip/Only Flags

**Problem:** No way to run subset of checks.

**Resolution:** Add to F-002:
```markdown
## Check Filtering

- `--skip=<check-id>`: Skip specific checks (comma-separated)
- `--only=<check-id>`: Run only specified checks
- `--skip-plugin=<name>`: Skip entire plugin
```

### 14. Error Message Format

**Problem:** Error messages lack context (file, line, suggestion).

**Resolution:** Add to F-002:
```markdown
## Error Format

All errors include:
- `code`: Structured identifier (E001, E002...)
- `message`: Human-readable description
- `file`: Path if applicable
- `suggestion`: Actionable next step
```

### 15. Git Status warn vs fail

**Problem:** F-005 says "Fail: working tree dirty" but implementation returns "warn".

**Resolution:** Update F-005:
```markdown
### Git Clean Check

- **Pass**: no changes
- **Warn**: working tree dirty (actionable by agent)

Note: Dirty tree is `warn` not `fail` because the agent can resolve it
by committing. Use `--dirty-as-error` to treat as failure.
```

---

## Lower Priority Refinements

### 16. Terminology Consistency

Standardize across all specs:
- "repo signals" → "triggers" (match API)
- "check" (singular) for individual, "checks" for collection
- Plugin version format: always semver "1.0.0"

### 17. Missing Edge Cases to Document

Add to relevant specs:
- Multiple go.mod files (monorepo behavior)
- Symlinked signal files (security note)
- Bare git repository (skip git checks)
- Concurrent dun invocations (file locking)

### 18. Automation Scope Clarification

Add to F-007:
```markdown
## Scope

Automation mode affects:
- Agent checks (prompt behavior)
- Doc reconciliation (apply vs plan)

Automation mode does NOT affect:
- Go quality checks (always run)
- Git hygiene (always checks, guidance varies)
- Plugin detection (always runs)
```

### 19. Install Command Improvements

Add to F-004:
```markdown
## Additional Options

- `--backup`: Create .bak before modifying AGENTS.md
- `dun uninstall`: Remove Dun marker blocks
- `dun install --verify`: Check installation is correct
```

### 20. Plugin Disable Mechanism

Add to F-003:
```markdown
## Configuration

Disable plugins via `.dun/config.yaml`:
```yaml
plugins:
  disabled:
    - helix  # Skip Helix checks
```
```

---

## Implementation Checklist

### Spec File Updates Required

- [ ] `feature-registry.md` - Renumber or align IDs
- [ ] `F-001-auto-discovery.md` - Add ordering algorithm, Node status
- [ ] `F-002-output-formats.md` - Exit codes, status values, error format
- [ ] `F-003-plugin-system.md` - Disable mechanism, check types list
- [ ] `F-004-install-command.md` - Error handling, backup option
- [ ] `F-005-git-hygiene.md` - Safety notes, warn not fail, no-hooks flag
- [ ] `F-006-doc-reconciliation.md` - Drift detection method
- [ ] `F-007-automation-slider.md` - Rename yolo, flag interactions, scope
- [ ] `F-014-go-quality-checks.md` - Lower threshold default, config override

### New Specs Needed

- [ ] `F-015-global-limits.md` - Timeouts, truncation, limits
- [ ] `F-016-error-handling.md` - Error codes, message format

---

## Consensus Confidence

| Category | Findings | Agreement |
|----------|----------|-----------|
| Critical (Must Fix) | 5 | All 4 perspectives agree |
| High Priority | 5 | 3-4 perspectives agree |
| Medium Priority | 5 | 2-3 perspectives agree |
| Lower Priority | 5 | 1-2 perspectives mentioned |

All 4 analysis perspectives (implementation-pragmatic, edge-case-focused, UX-coherence, consistency) converged on the critical issues. The high and medium priority items had strong overlap. Lower priority items were identified by 1-2 perspectives but are still valid improvements.
