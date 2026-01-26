# US-007: Enforce Go Quality Checks

As an agent operator, I want Dun to run Go tests, coverage, and static analysis
so I can trust that Go changes meet baseline quality.

## Acceptance Criteria

- Go repos automatically run tests, coverage, and vet.
- Coverage failures report the current percentage and target.
- Staticcheck warns when missing, fails when issues are found.
