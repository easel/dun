# Security Requirements

- No network access or code exfiltration by default.
- Prompt envelopes must not include secrets from the repo.
- Agent execution is opt-in and must be explicit.
- Validate all external inputs (plugin manifests, responses).
