# Feature Spec: F-015 Exit Codes

## Summary

Dun uses standardized exit codes for CI/CD integration.

## Exit Codes

| Code | Name | Description |
|------|------|-------------|
| 0 | Success | All checks pass |
| 1 | Check Failed | One or more checks failed |
| 2 | Config Error | Configuration file error |
| 3 | Runtime Error | Command not found, permission denied |
| 4 | Usage Error | Bad flags, missing arguments |

## Acceptance Criteria

- All CLI commands return appropriate exit codes
- Exit codes are documented and consistent
- CI systems can rely on exit code semantics
