# Solution Design: Plugin System

## Problem

Dun needs workflow-specific checks without hardcoding every rule in the core
binary.

## Goals

- Load embedded plugin manifests at startup.
- Activate plugins via repo signals (paths/globs).
- Support rule-sets, gates, state rules, and agent prompts.
- Keep check ordering deterministic.

## Approach

1. Embed plugin manifests into the binary at build time.
2. Load manifests into a registry on startup.
3. Match repo signals to activate plugins.
4. Build a combined plan and sort by a stable key.

## Components

- Manifest Loader: reads embedded manifests.
- Plugin Registry: stores manifests and activation rules.
- Signal Matcher: evaluates repo signals.
- Check Planner: assembles the deterministic plan.

## Data Flow

1. Loader reads embedded manifests.
2. Matcher evaluates repo signals.
3. Registry activates matching plugins.
4. Planner produces ordered checks for execution.

## Open Questions

- Should external manifests be supported in later phases?
- How should manifest versioning and conflicts be handled?
