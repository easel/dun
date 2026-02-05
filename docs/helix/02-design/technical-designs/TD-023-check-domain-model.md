---
dun:
  id: TD-023
  depends_on:
    - US-023
  review:
    self_hash: 9522804907a7c19f06e7868543b17c4fb35218c841953b07a058ac5d8360f6c4
    deps:
      US-023: 68b0c2dbf86881914c5acbd5cbe3eefc20eee60ec8e0d3539b97a8ac11a53cfa
---
# TD-023: Check Domain Model + Registry

## Overview

Implement a registry-based check pipeline with typed configs and a consistent
result schema that includes summary, score, and update signals.

## Data Model

- `CheckSpec`: raw manifest definition (YAML) used for decoding and plan
  inspection.
- `CheckDefinition`: normalized metadata (id, description, phase, priority,
  conditions, plugin id).
- `CheckConfig`: type-specific config struct decoded from `CheckSpec`.
- `CheckResult`: status/signal/detail/issues plus optional `summary`, `score`,
  and `update`.

## Pipeline

1. Load plugins and build plan from `CheckSpec`.
2. Decode each `CheckSpec` using the registry into a typed config.
3. Execute the check handler with `CheckDefinition`, config, and options.
4. Post-process results with a summarizer (summary + score + update signals).

## Registry Interface

- `Type() string`
- `Decode(CheckSpec) (CheckConfig, error)`
- `Run(root, def, config, opts) (CheckResult, error)`

Optional: `Explain(config) CheckExplain` for `dun explain` details.

## Summarization

- Summary: short, deterministic string derived from status + signal + detail.
- Score: numeric score derived from status (pass > warn > fail/skip).
- Update: optional structured signal for checks that detect staleness.

## Backwards Compatibility

- Plugin YAML structure stays unchanged.
- CLI outputs remain compatible; new fields are additive.

## Testing

- Registry dispatch tests for each check type.
- Config decode tests for representative check types.
- Summary/score tests for status variants.
- Full `go test ./...` pass.
