---
dun:
  id: TP-012
  depends_on:
  - TD-012
---
# Test Plan: TP-012 Generic Command Checks

## Summary

Test plan for US-012: Generic Command Checks implementation.

## Test Categories

### Unit Tests

#### Command Execution
- [ ] TC-001: Command executes successfully, returns pass
- [ ] TC-002: Command fails (non-zero exit), returns fail
- [ ] TC-003: Command times out, returns fail with timeout message
- [ ] TC-004: Command not found, returns fail with helpful message
- [ ] TC-005: Custom success_exit code recognized
- [ ] TC-006: Warn exit codes return warn status
- [ ] TC-007: Environment variables passed to command
- [ ] TC-008: Working directory set to project root

#### Output Parsers
- [ ] TC-010: parser=text returns raw output as detail
- [ ] TC-011: parser=lines creates issue per line
- [ ] TC-012: parser=json parses valid JSON
- [ ] TC-013: parser=json with invalid JSON falls back to text
- [ ] TC-014: parser=json-lines parses newline-delimited JSON
- [ ] TC-015: parser=regex extracts named groups as issues
- [ ] TC-016: parser=regex with no matches returns empty issues

#### Issue Extraction
- [ ] TC-020: issue_path extracts array from JSON
- [ ] TC-021: issue_fields maps JSON fields to Issue struct
- [ ] TC-022: issue_pattern extracts file, line, message from regex
- [ ] TC-023: Missing optional fields (line, severity) handled gracefully

#### External Plugin Loading
- [ ] TC-030: Loads plugins from ~/.dun/plugins/
- [ ] TC-031: Loads plugins from .dun/plugins/
- [ ] TC-032: Project plugins override user plugins with same ID
- [ ] TC-033: Invalid plugin YAML skipped without error
- [ ] TC-034: Missing plugin directory not an error
- [ ] TC-035: Plugin triggers evaluated correctly

### Integration Tests

- [ ] TC-040: Full command check with real `echo` command
- [ ] TC-041: Command check with environment variable substitution
- [ ] TC-042: Security plugin detects (mocked) vulnerability
- [ ] TC-043: External plugin discovered and executed
- [ ] TC-044: Mixed builtin and external plugins in plan

### Edge Cases

- [ ] TC-050: Empty command output handled
- [ ] TC-051: Very large output truncated
- [ ] TC-052: Binary output handled (non-UTF8)
- [ ] TC-053: Command with quotes and special characters
- [ ] TC-054: Concurrent command checks don't interfere

## Coverage Targets

- command_check.go: 95%+
- parsers.go: 95%+
- plugin_loader.go (new code): 90%+

## Test Fixtures

Create in `internal/testdata/plugins/`:
- `valid-plugin/plugin.yaml` - Valid command plugin
- `invalid-yaml/plugin.yaml` - Malformed YAML
- `missing-id/plugin.yaml` - Missing required fields
- `echo-plugin/plugin.yaml` - Plugin using echo for testing
