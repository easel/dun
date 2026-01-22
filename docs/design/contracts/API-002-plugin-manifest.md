# API Contract: Dun Plugin Manifest (FEAT-002)

**Contract ID**: API-002  
**Feature**: FEAT-002 (Plugin System)  
**Type**: Library  
**Status**: Draft  
**Version**: 1.0.0  

*Define all external interfaces before implementation*

## CLI Interface Contract

Not applicable. Plugins are loaded by the Dun CLI, but the interface is defined
by the plugin manifest schema below.

---

## Library API Contract

### Plugin Manifest Interface

**Purpose**: Define plugin discovery rules and checks in a portable format.

**Location**:
- Built-in plugins: embedded in the binary
- External plugins: `<repo>/.dun/plugins/<plugin-id>/plugin.yaml` or
  user-configured plugin paths

---

## Data Contracts

### Plugin Manifest Schema
```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "properties": {
    "id": { "type": "string" },
    "version": { "type": "string" },
    "description": { "type": "string" },
    "triggers": {
      "type": "array",
      "items": {
        "type": "object",
        "properties": {
          "type": { "type": "string", "enum": ["path-exists", "glob-exists"] },
          "value": { "type": "string" }
        },
        "required": ["type", "value"]
      }
    },
    "checks": {
      "type": "array",
      "items": {
        "type": "object",
        "properties": {
          "id": { "type": "string" },
          "description": { "type": "string" },
          "type": { "type": "string", "enum": ["rule-set", "command", "agent", "state-rules", "gates"] },
          "phase": { "type": "string" },
          "state_rules": { "type": "string" },
          "gate_files": { "type": "array", "items": { "type": "string" } },
          "inputs": { "type": "array", "items": { "type": "string" } },
          "conditions": {
            "type": "array",
            "items": {
              "type": "object",
              "properties": {
                "type": { "type": "string", "enum": ["path-exists", "path-missing", "glob-min-count", "glob-max-count"] },
                "path": { "type": "string" },
                "expected": { "type": "integer" }
              },
              "required": ["type"]
            }
          },
          "rules": {
            "type": "array",
            "items": {
              "type": "object",
              "properties": {
                "type": { "type": "string", "enum": ["path-exists", "path-missing", "glob-min-count", "glob-max-count", "pattern-count", "unique-ids", "cross-reference"] },
                "path": { "type": "string" },
                "pattern": { "type": "string" },
                "expected": { "type": "integer" },
                "severity": { "type": "string", "enum": ["warn", "fail"] }
              },
              "required": ["type"]
            }
          },
          "command": { "type": "string" },
          "prompt": { "type": "string" },
          "response_schema": { "type": "string" }
        },
        "required": ["id", "type", "description"]
      }
    }
  },
  "required": ["id", "version", "triggers", "checks"]
}
```

### Example: Helix Plugin (YAML)
```yaml
id: helix
version: "1.0.0"
description: "Helix workflow validation and gates"
triggers:
  - type: path-exists
    value: docs/helix
checks:
  - id: helix-structure
    description: Required Helix artifacts exist
    type: rule-set
    phase: frame
    rules:
      - type: path-exists
        path: docs/helix/01-frame/prd.md
        severity: fail
      - type: glob-min-count
        path: docs/helix/01-frame/user-stories/*.md
        expected: 1
        severity: warn

  - id: helix-spec-to-design
    description: PRD and design alignment
    type: agent
    phase: design
    conditions:
      - type: path-exists
        path: docs/helix/01-frame/prd.md
      - type: path-exists
        path: docs/helix/02-design/architecture.md
    inputs:
      - docs/helix/01-frame/prd.md
      - docs/helix/02-design/architecture.md
    prompt: prompts/helix/spec-to-design.md
    response_schema: responses/agent-default.json

  - id: helix-gates
    description: Validate Helix phase gates
    type: gates
    phase: frame
    gate_files:
      - gates/01-frame/input-gates.yml
      - gates/01-frame/exit-gates.yml
```

### Agent Response Schema (Default)
Agent checks are expected to return JSON with the following shape:

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

### Output Schema (Plugin Check)
```json
{
  "id": "helix-gates",
  "status": "fail",
  "signal": "2 required gates missing",
  "detail": "docs/helix/01-frame/prd.md missing",
  "next": "Create docs/helix/01-frame/prd.md"
}
```

---

## Error Contracts

### Error Codes
| Code | Description | Exit Code | Recovery Action |
|------|-------------|-----------|-----------------|
| ERR_PLUGIN_INVALID | Manifest schema invalid | 4 | Fix plugin manifest |
| ERR_PLUGIN_TRIGGER | Trigger evaluation failed | 1 | Check repo structure |
| ERR_AGENT_UNCONFIGURED | Agent not configured | 0 | Set `DUN_AGENT_CMD` |
| ERR_AGENT_RESPONSE | Agent response invalid | 2 | Fix response schema |

### Error Response Format (JSON)
```json
{
  "error": {
    "code": "ERR_PLUGIN_INVALID",
    "message": "Plugin manifest missing checks",
    "details": {},
    "timestamp": "2026-01-21T12:00:00Z"
  }
}
```

---

## Contract Validation

### Test Scenarios
1. **Trigger Match**: Plugin loads when sentinel path exists.
2. **Rule Set**: Missing artifact fails with clear signal.
3. **Agent Check**: Prompt renders and response parses.
4. **Unknown Rule**: Invalid rule type is rejected.

### Backwards Compatibility
- [ ] Additive-only changes to manifest schema
- [ ] Schema version bumps on breaking changes

---

## Feature Traceability

### Parent Feature
- **Feature Specification**: Not yet created (see `docs/PRD.md`)
- **User Stories Implemented**: Not yet created

### Related Artifacts
- **ADRs**: `docs/design/adr/ADR-001-cli-stdlib.md`
- **Solution Design**: `docs/design/solution-designs/SD-002-plugin-system.md`
- **Implementation**: `internal/plugins/`

---
*Note: Create one manifest per plugin. Plugins should be deterministic and
local-only by default.*
