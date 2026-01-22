# US-005: Reconcile PRD Changes Through the Stack

As an agent operator, I want Dun to detect when the PRD changes and produce a
clear downstream plan so I can keep docs and code aligned quickly.

## Acceptance Criteria

- When PRD changes, Dun emits a list of impacted artifacts in order.
- The plan includes updates for feature specs, design docs, ADRs, test plans,
  and implementation.
- The plan is structured and deterministic.
