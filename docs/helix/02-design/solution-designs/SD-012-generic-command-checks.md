---
dun:
  id: SD-012
  depends_on:
  - F-020
---
# SD-012: Generic Command Checks

## Overview

This document describes the implementation of generic command checks that allow
defining custom dun checks via YAML plugins without writing Go code.

**User Story**: US-012
**Status**: Planned

## Architecture

### Component Overview

```
+------------------+     +-------------------+     +------------------+
|  Plugin Loader   |---->|  Command Check    |---->|  Output Parser   |
|  (YAML manifest) |     |  (shell exec)     |     |  (json/regex/...)|
+------------------+     +-------------------+     +------------------+
        ^                                                   |
        |                                                   v
+------------------+                              +------------------+
| External Plugins |                              |  Issue Extractor |
| ~/.dun/plugins/  |                              |  (structured)    |
+------------------+                              +------------------+
```

### Data Flow

1. Plugin loader finds `plugin.yaml` files (builtin + external)
2. Engine encounters `type: command` check
3. `runCommandCheck()` executes shell command
4. Output parser converts raw output to structured form
5. Issue extractor pulls issues via path/pattern
6. CheckResult returned with status, detail, issues

## Check Schema Extension

```yaml
checks:
  - id: string           # Check identifier
    type: command        # NEW: generic command type
    command: string      # Shell command to execute

    # Exit code handling
    success_exit: int    # Exit code for pass (default: 0)
    warn_exit: int[]     # Exit codes for warn (optional)

    # Output parsing
    parser: string       # text|lines|json|json-lines|regex

    # Issue extraction (for json parsers)
    issue_path: string   # JSONPath to issues array
    issue_fields:        # Field mapping
      file: string       # JSONPath to filename
      line: string       # JSONPath to line number
      message: string    # JSONPath to message
      severity: string   # JSONPath to severity

    # Issue extraction (for regex parser)
    issue_pattern: string  # Regex with named groups: (?P<file>...)

    # Execution options
    timeout: duration    # Command timeout (default: 5m)
    shell: string        # Shell to use (default: sh -c)
    env:                 # Additional environment variables
      KEY: value
```

## Implementation

### 1. command_check.go

```go
package dun

import (
    "context"
    "encoding/json"
    "os/exec"
    "regexp"
    "strings"
    "time"
)

const defaultCommandTimeout = 5 * time.Minute

func runCommandCheck(root string, check Check) (CheckResult, error) {
    ctx, cancel := context.WithTimeout(context.Background(), commandTimeout(check))
    defer cancel()

    cmd := exec.CommandContext(ctx, "sh", "-c", check.Command)
    cmd.Dir = root
    cmd.Env = buildEnv(check)

    output, err := cmd.CombinedOutput()
    exitCode := exitCodeFromError(err)

    // Determine status from exit code
    status := "pass"
    if exitCode != successExit(check) {
        if isWarnExit(check, exitCode) {
            status = "warn"
        } else {
            status = "fail"
        }
    }

    // Parse output and extract issues
    issues, detail := parseOutput(check, output)

    return CheckResult{
        ID:     check.ID,
        Status: status,
        Signal: signalFromStatus(status, check),
        Detail: detail,
        Issues: issues,
        Next:   check.Command,
    }, nil
}

func parseOutput(check Check, output []byte) ([]Issue, string) {
    switch check.Parser {
    case "json":
        return parseJSON(check, output)
    case "json-lines":
        return parseJSONLines(check, output)
    case "lines":
        return parseLines(output)
    case "regex":
        return parseRegex(check, output)
    default: // "text" or empty
        return nil, trimOutput(output)
    }
}
```

### 2. External Plugin Loading

Add to `plugin_loader.go`:

```go
func LoadExternalPlugins() ([]Plugin, error) {
    var plugins []Plugin

    // User plugins: ~/.dun/plugins/*/plugin.yaml
    userDir := filepath.Join(os.Getenv("HOME"), ".dun", "plugins")
    userPlugins, _ := loadPluginsFromDir(userDir)
    plugins = append(plugins, userPlugins...)

    // Project plugins: .dun/plugins/*/plugin.yaml
    projectDir := ".dun/plugins"
    projectPlugins, _ := loadPluginsFromDir(projectDir)
    plugins = append(plugins, projectPlugins...)

    return plugins, nil
}

func loadPluginsFromDir(dir string) ([]Plugin, error) {
    entries, err := os.ReadDir(dir)
    if err != nil {
        return nil, nil // Not an error if dir doesn't exist
    }

    var plugins []Plugin
    for _, entry := range entries {
        if !entry.IsDir() {
            continue
        }
        pluginDir := filepath.Join(dir, entry.Name())
        p, err := loadPluginFromPath(pluginDir)
        if err != nil {
            continue // Skip invalid plugins
        }
        plugins = append(plugins, p)
    }
    return plugins, nil
}
```

### 3. Security Plugin (Builtin)

`internal/plugins/builtin/security/plugin.yaml`:

```yaml
id: security
version: "1"
description: "Security vulnerability scanning"
priority: 20
triggers:
  - type: path-exists
    value: go.mod
checks:
  - id: govulncheck
    description: "Check for known Go vulnerabilities"
    type: command
    command: govulncheck -json ./...
    parser: json-lines
    issue_path: $.vulnerability
    issue_fields:
      file: $.package
      message: $.summary
      severity: $.severity
    phase: security
    timeout: 10m
```

## New Files

| File | Purpose |
|------|---------|
| `internal/dun/command_check.go` | Command execution and parsing |
| `internal/dun/command_check_test.go` | Unit tests |
| `internal/dun/parsers.go` | Output parsers (json, regex, lines) |
| `internal/dun/parsers_test.go` | Parser tests |
| `internal/plugins/builtin/security/plugin.yaml` | Security plugin manifest |

## Modified Files

| File | Changes |
|------|---------|
| `internal/dun/engine.go` | Add `case "command"` handler |
| `internal/dun/plugin_loader.go` | Add external plugin loading |
| `internal/dun/types.go` | Extend Check struct with command fields |
| `internal/plugins/builtin/builtin.go` | Add security plugin to embed |

## Example Plugins

### npm-audit Plugin

```yaml
id: npm-security
version: "1"
triggers:
  - type: path-exists
    value: package.json
checks:
  - id: npm-audit
    type: command
    command: npm audit --json
    parser: json
    issue_path: $.vulnerabilities.*
    issue_fields:
      file: $.name
      message: $.title
      severity: $.severity
```

### License Compliance Plugin

```yaml
id: license
version: "1"
triggers:
  - type: path-exists
    value: go.mod
checks:
  - id: go-licenses
    type: command
    command: go-licenses check ./...
    parser: lines
    issue_pattern: "^(?P<severity>\\w+):\\s+(?P<file>[^:]+):\\s+(?P<message>.+)$"
```

### API Contract Plugin

```yaml
id: api-contract
version: "1"
triggers:
  - type: path-exists
    value: openapi.yaml
checks:
  - id: oapi-diff
    type: command
    command: oapi-diff --fail-on-diff openapi.yaml generated.yaml
    parser: text
```

## Testing Strategy

1. Unit tests for each parser type
2. Unit tests for exit code handling
3. Integration tests with mock commands
4. Test external plugin loading from temp directories
5. Test plugin override (project over user)

## Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| Command injection | Low | High | Commands from trusted YAML only |
| Slow commands block | Medium | Medium | Configurable timeout, default 5m |
| Parser failures | Medium | Low | Fallback to raw text output |
| Missing tools | High | Low | Graceful degradation with warn status |
