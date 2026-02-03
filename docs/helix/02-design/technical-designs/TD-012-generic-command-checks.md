---
dun:
  id: TD-012
  depends_on:
  - US-012
---
# Technical Design: TD-012 Generic Command Checks

## Story Reference

**User Story**: US-012 Generic Command Checks
**Parent Feature**: F-020 Generic Command Checks
**Solution Design**: SD-012 Generic Command Checks

## Goals

- Allow YAML plugins to run shell commands as checks.
- Support multiple output parsers (text, lines, json, json-lines, regex).
- Load plugins from user and project directories.

## Non-Goals

- Remote plugin discovery.

## Technical Approach

### Implementation Strategy

- Add a `type: command` check handler that executes shell commands with
  configurable timeouts.
- Parse output into issues using parser-specific logic.
- Merge external plugins with built-ins, with project plugins taking precedence.

### Key Decisions

- Treat non-zero exit as failure unless `allow_failure` is explicitly set.
- Parse JSON output into structured issues using JSONPath or regex groups.

## Component Changes

### Components to Modify

- `internal/dun/command_check.go`: execute commands and parse output.
- `internal/dun/plugin_loader.go`: load external plugin directories.
- `internal/plugins/builtin/security/plugin.yaml`: example command plugin.

### New Components

- Output parser helpers for lines/json/json-lines/regex.

## Interfaces and Config

- Plugin schema: `type: command`, `parser`, `timeout`, `issue_pattern`.
- Config: external plugin search paths.

## Data and State

- No persistent state; command output parsed in-memory.

## Testing Approach

- Unit tests for each parser type.
- Integration tests for external plugin loading.

## Risks and Mitigations

- **Risk**: Arbitrary command execution. **Mitigation**: explicit opt-in via
  plugin files and automation policy gates.

## Rollout / Compatibility

- Backwards compatible; command checks are opt-in.
