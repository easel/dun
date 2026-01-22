# API Contract: Dun Install (FEAT-003)

**Contract ID**: API-003  
**Feature**: FEAT-003 (Install)  
**Type**: CLI  
**Status**: Draft  
**Version**: 1.0.0  

*Define all external interfaces before implementation*

## CLI Interface Contract

### Command Structure
```bash
$ dun install [options]
```

### Command: install
**Purpose**: Set up Dun in the current repo (AGENTS.md snippet).  
**Usage**: `$ dun install [options]`

**Options**:
- `--dry-run` : Show planned changes without writing (default `false`)

**Input**:
- Format: File system + optional config

**Output**:
- Format: Prompt JSON or LLM text (same as `check`)
- Schema: Includes plan steps and results

**Exit Codes**:
- `0`: Success
- `1`: Internal error
- `2`: Conflict or blocked action
- `4`: Invalid arguments

**Examples**:
```bash
# Preview install
$ dun install --dry-run
plan: create AGENTS.md

# Apply install
$ dun install
```

---

## Data Contracts

### Install Plan Output (JSON)
```json
{
  "version": "1",
  "steps": [
    { "type": "agents", "path": "AGENTS.md", "action": "create" }
  ]
}
```

---

## Error Contracts

### Error Codes
| Code | Description | Exit Code | Recovery Action |
|------|-------------|-----------|-----------------|
| ERR_NOT_A_REPO | Repo root not found | 4 | Run inside a git repo |
| ERR_WRITE_FAILED | File write failed | 1 | Check permissions |

---

## Contract Validation

### Test Scenarios
1. **Dry run**: outputs plan without file changes.
2. **Install**: creates AGENTS snippet.

---

## Feature Traceability

### Parent Feature
- **Feature Specification**: Not yet created (see `docs/PRD.md`)

### Related Artifacts
- **Solution Design**: `docs/design/solution-designs/SD-003-install.md`

---
*Note: Install should be safe and reversible.*
