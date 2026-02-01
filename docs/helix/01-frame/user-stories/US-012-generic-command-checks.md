# US-012: Generic Command Checks

## User Story

**As a** developer using dun,
**I want** to define custom checks using shell commands in YAML plugins,
**So that** I can extend dun with new workflows without writing Go code.

## Acceptance Criteria

### AC-1: Command Check Type
- [ ] `type: command` checks execute arbitrary shell commands
- [ ] Exit code 0 = pass, non-zero = fail
- [ ] Command output captured for detail/issues
- [ ] Timeout support with configurable duration

### AC-2: Output Parsing
- [ ] `parser: text` - Raw output as detail (default)
- [ ] `parser: lines` - Each line becomes an issue
- [ ] `parser: json` - Parse JSON output
- [ ] `parser: json-lines` - Parse newline-delimited JSON
- [ ] `parser: regex` - Extract issues via regex pattern

### AC-3: Issue Extraction
- [ ] `issue_path` - JSONPath for extracting issues from JSON
- [ ] `issue_pattern` - Regex pattern with named groups (file, line, message)
- [ ] Issues appear in CheckResult.Issues array

### AC-4: External Plugin Loading
- [ ] Load plugins from `~/.dun/plugins/*/plugin.yaml`
- [ ] Load plugins from `.dun/plugins/*/plugin.yaml` (project-local)
- [ ] Project plugins override user plugins with same ID

### AC-5: Built-in Security Plugin
- [ ] `govulncheck` check for Go vulnerability scanning
- [ ] Triggers on `go.mod` presence
- [ ] Parses JSON output into structured issues

## Technical Notes

- Implement `runCommandCheck()` in `internal/dun/command_check.go`
- Add external plugin loading to `plugin_loader.go`
- Create `internal/plugins/builtin/security/` with YAML manifest
- Reuse existing Issue type from types.go

## Dependencies

- None (extends existing plugin system)

## Priority

High - Enables all quorum-identified workflows via plugins

## Estimation

- Command check implementation: Medium
- External plugin loading: Small
- Security plugin: Small
