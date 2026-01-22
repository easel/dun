# Threat Model

## Assets

- Local repo code and documentation
- Deterministic check outputs

## Threats

- Accidental leakage of sensitive content in prompts
- Malicious plugin manifests or responses
- Excessive resource usage during checks

## Mitigations

- Local-only execution by default
- Schema validation of plugin manifests and responses
- Timeouts and worker limits
