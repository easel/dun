# API Contract: Dun CLI (FEAT-001)

**Contract ID**: API-001  
**Feature**: FEAT-001 (Core CLI)  
**Type**: CLI  
**Status**: Draft  
**Version**: 1.0.0  

*Define all external interfaces before implementation*

## CLI Interface Contract

### Command Structure
```bash
$ dun [command] [options] [arguments]
```

### Commands

#### Command: check
**Purpose**: Discover and run applicable checks for the current repo.  
**Usage**: `$ dun check [options]`

**Options**:
- `--format` : Output format (`prompt`, `llm`, or `json`, default `prompt`)
- `--changed` : Limit checks to changed files (default `false`)
- `--timeout` : Global timeout in seconds (default `600`)
- `--check-timeout` : Per-check timeout in seconds (default `120`)
- `--workers` : Max concurrent checks (default `min(4, CPU)`)
- `--config` : Path to config file (default `.dun/config.yaml` if present)
- `--agent-cmd` : Command to run agent checks (optional, used with `--agent-mode=auto`)
- `--agent-timeout` : Agent check timeout in seconds (default `300`)
- `--agent-mode` : Agent mode (`prompt` or `auto`, default `prompt`)
- `--automation` : Automation mode (`manual`, `plan`, `auto`, `autonomous`, default `auto`)

**Input**:
- Format: File system + optional config file
- Schema: See Data Contracts (config schema)

**Output**:
- Format: `llm` text blocks or JSON
- Schema: See Data Contracts (output schema)

**Exit Codes**:
- `0`: All checks pass or are skipped/warn-only
- `1`: Internal error (discovery or execution failure)
- `2`: One or more checks failed
- `3`: One or more checks timed out
- `4`: Invalid arguments or config

**Examples**:
```bash
# Default prompt-as-data output
$ dun check
{"checks":[{"id":"helix-create-architecture","status":"prompt","signal":"agent prompt ready","prompt":{"kind":"dun.prompt.v1","id":"helix-create-architecture","prompt":"Check-ID: helix-create-architecture\n...","callback":{"command":"dun respond --id helix-create-architecture --response -","stdin":true}}}]}

# LLM output
$ dun check --format=llm
check:go-test status:pass duration_ms:421
signal: 14 packages passed

# JSON output, changed files only
$ dun check --format=json --changed
{"checks":[{"id":"go-test","status":"fail","signal":"1 package failed","detail":"pkg/foo TestFoo panicked at foo_test.go:42","next":"go test ./pkg/foo -run TestFoo"}]}
```

---

#### Command: list
**Purpose**: Show the checks that would run for the repo.  
**Usage**: `$ dun list [options]`

**Options**:
- `--format` : Output format (`text` or `json`, default `text`)
- `--changed` : Limit checks to changed files (default `false`)
- `--config` : Path to config file (default `.dun/config.yaml` if present)

**Input**:
- Format: File system + optional config file
- Schema: See Data Contracts (config schema)

**Output**:
- Format: Text list or JSON array
- Schema: See Data Contracts (list schema)

**Exit Codes**:
- `0`: Success
- `1`: Internal error
- `4`: Invalid arguments or config

**Examples**:
```bash
# Text list
$ dun list
go-test    Run Go tests for ./...
go-vet     Run go vet ./...

# JSON list
$ dun list --format=json
{"version":"1","checks":[{"id":"go-test","description":"Run Go tests for ./..."},{"id":"go-vet","description":"Run go vet ./..."}]}
```

---

#### Command: explain
**Purpose**: Explain a specific check and how it was discovered.  
**Usage**: `$ dun explain <check-id> [options]`

**Options**:
- `--format` : Output format (`text` or `json`, default `text`)
- `--config` : Path to config file (default `.dun/config.yaml` if present)

**Input**:
- Format: Check ID argument
- Schema: `<check-id>` is a stable identifier (e.g., `go-test`)

**Output**:
- Format: Text or JSON
- Schema: See Data Contracts (explain schema)

**Exit Codes**:
- `0`: Success
- `1`: Internal error
- `4`: Invalid arguments or unknown check

**Examples**:
```bash
# Text explanation
$ dun explain go-test
id: go-test
description: Run Go tests for ./...
discoverer: go.mod detected
command: go test ./...
timeout_s: 120

# JSON explanation
$ dun explain go-test --format=json
{"id":"go-test","description":"Run Go tests for ./...","discoverer":"go.mod detected","command":"go test ./...","timeout_s":120}
```

---

#### Command: respond
**Purpose**: Parse an agent response for a prompt and emit a check result.  
**Usage**: `$ dun respond --id <check-id> --response <path|-> [options]`

**Options**:
- `--id` : Check ID from the prompt envelope (required)
- `--response` : JSON response path or `-` for stdin (default `-`)
- `--format` : Output format (`json` or `llm`, default `json`)

**Input**:
- Format: JSON response from agent
- Schema: See Data Contracts (agent response schema)

**Output**:
- Format: JSON or LLM text block
- Schema: See Data Contracts (check schema)

**Exit Codes**:
- `0`: Success
- `1`: Internal error or invalid response
- `4`: Invalid arguments

**Examples**:
```bash
# Respond using stdin
$ dun respond --id helix-create-architecture --response -
{"id":"helix-create-architecture","status":"pass","signal":"architecture drafted"}
```

---

## REST API Contract (if applicable)

Not applicable for MVP.

---

## Library API Contract

Not applicable for MVP (CLI only).

---

## Data Contracts

### Input Schema (Config File)
Config is optional and may be provided as YAML. The schema below represents the
equivalent JSON shape.

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "properties": {
    "version": { "type": "string" },
    "timeouts": {
      "type": "object",
      "properties": {
        "global_ms": { "type": "integer", "minimum": 1000 },
        "per_check_ms": { "type": "integer", "minimum": 1000 }
      }
    },
    "workers": { "type": "integer", "minimum": 1 },
    "checks": {
      "type": "object",
      "properties": {
        "enable": { "type": "array", "items": { "type": "string" } },
        "disable": { "type": "array", "items": { "type": "string" } }
      }
    },
    "agent": {
      "type": "object",
      "properties": {
        "cmd": { "type": "string" },
        "timeout_ms": { "type": "integer", "minimum": 1000 },
        "response_format": { "type": "string", "enum": ["json"] },
        "mode": { "type": "string", "enum": ["prompt", "auto"] },
        "automation": { "type": "string", "enum": ["manual", "plan", "auto", "autonomous"] }
      }
    },
    "ratchet": {
      "type": "object",
      "properties": {
        "mode": { "type": "string", "enum": ["off", "warn", "block"] },
        "baseline_path": { "type": "string" }
      }
    }
  }
}
```

### Output Schema (check)
```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "properties": {
    "checks": {
      "type": "array",
      "items": {
        "type": "object",
        "properties": {
          "id": { "type": "string" },
          "status": { "type": "string", "enum": ["pass", "warn", "fail", "skip", "timeout", "prompt"] },
          "signal": { "type": "string" },
          "detail": { "type": "string" },
          "next": { "type": "string" },
          "issues": {
            "type": "array",
            "items": {
              "type": "object",
              "properties": {
                "id": { "type": "string" },
                "summary": { "type": "string" },
                "path": { "type": "string" }
              }
            }
          },
          "prompt": {
            "type": "object",
            "properties": {
              "kind": { "type": "string" },
              "id": { "type": "string" },
              "title": { "type": "string" },
              "summary": { "type": "string" },
              "prompt": { "type": "string" },
              "inputs": { "type": "array", "items": { "type": "string" } },
              "response_schema": { "type": "string" },
              "callback": {
                "type": "object",
                "properties": {
                  "command": { "type": "string" },
                  "stdin": { "type": "boolean" }
                },
                "required": ["command"]
              }
            },
            "required": ["kind", "id", "prompt", "callback"]
          }
        },
        "required": ["id", "status", "signal"]
      }
    }
  },
  "required": ["checks"]
}
```

### Output Schema (list)
```json
{
  "version": "1",
  "checks": [
    { "id": "go-test", "description": "Run Go tests for ./..." }
  ]
}
```

### Output Schema (explain)
```json
{
  "id": "go-test",
  "description": "Run Go tests for ./...",
  "discoverer": "go.mod detected",
  "command": "go test ./...",
  "timeout_s": 120
}
```

### Input Schema (agent response)
```json
{
  "status": "pass|warn|fail",
  "signal": "short summary",
  "detail": "optional detail",
  "next": "optional next command",
  "issues": [
    { "id": "ISSUE-1", "summary": "short issue", "path": "docs/..." }
  ]
}
```

---

## Error Contracts

### Error Codes
| Code | Description | Exit Code | Recovery Action |
|------|-------------|-----------|-----------------|
| ERR_INVALID_CONFIG | Config file invalid or unreadable | 4 | Fix config or pass `--config` |
| ERR_DISCOVERY_FAILED | Repo discovery failed | 1 | Re-run with `--format=json` for detail |
| ERR_EXEC_FAILED | Check execution failed | 1 | Inspect tool output and environment |
| ERR_TIMEOUT | Check timed out | 3 | Re-run with higher timeout |
| ERR_UNKNOWN_CHECK | Unknown check ID | 4 | Run `dun list` to see valid IDs |

### Error Response Format (JSON)
```json
{
  "error": {
    "code": "ERR_INVALID_CONFIG",
    "message": "Config parse error: invalid timeout value",
    "details": {},
    "timestamp": "2026-01-21T12:00:00Z"
  }
}
```

---

## Contract Validation

### Test Scenarios
1. **Happy Path**: Repo with Go tests passes `dun check`.
2. **Invalid Input**: Bad config yields ERR_INVALID_CONFIG.
3. **Edge Cases**: Repo with no checks yields empty plan and exit 0.
4. **Error Cases**: Missing tool yields ERR_EXEC_FAILED with guidance.

### Backwards Compatibility
- [ ] All changes are additive only
- [ ] No breaking changes to existing contracts
- [ ] Output schema version bumps on breaking changes

---

## Feature Traceability

### Parent Feature
- **Feature Specification**: Not yet created (see `docs/PRD.md`)
- **User Stories Implemented**: Not yet created

### Related Artifacts
- **ADRs**: None yet
- **Test Suites**: `tests/contract/`
- **Implementation**: `cmd/dun/`
- **Solution Design**: `docs/design/solution-designs/SD-001-dun.md`

### Contract Naming Convention
- Format: `[feature]-[interface-type]-contract.md`
- Example: `core-cli-contract.md`

---
*Note: Create one contract document per major interface.*
*Some contracts may serve multiple features (mark as "Cross-cutting").*
*Contract ID (API-XXX) should be unique across the project.*
