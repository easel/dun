# Solution Design: Doc and Code Reconciliation

## Problem

Documentation and implementation drift when changes are made in one layer
without updating downstream artifacts.

## Goals

- Detect drift across PRD, specs, design, tests, and code.
- Produce ordered, actionable changes.
- Support automation modes without losing determinism.

## Approach

1. **Artifact inventory**: enumerate Helix docs and relevant code paths.
2. **Delta detection**: compare current docs vs expected downstream artifacts.
3. **Impact graph**: map PRD changes to feature specs, stories, design, tests.
4. **Plan output**: emit issues in dependency order with clear next steps.
5. **Automation policy**: apply manual/plan/auto/yolo rules to prompts.

## Components

- Drift Analyzer: detects missing/changed artifacts.
- Impact Mapper: builds dependency order.
- Planner: emits plan + issues.
- Prompt Emitter: includes automation mode in prompts.

## Data Flow

1. Helix plugin triggers reconciliation check.
2. Drift analyzer builds artifact inventory.
3. Impact mapper produces ordered issues.
4. Output is emitted as prompt envelope (plan or yolo).

## Open Questions

- What minimal code context should be included in prompts?
- How should we track approved manual confirmations?
