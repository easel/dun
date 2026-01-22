# Principles

These principles guide Dun's design and implementation.

1. **Agent-first workflows**: Outputs must be deterministic and easy for agents
   to parse, with prompt-as-data as the default.
2. **Fast feedback**: Local runs should be quick, with short time-to-signal.
3. **Local-only by default**: No network calls or code exfiltration unless
   explicitly enabled.
4. **Extensible core**: New checks should be added via manifests and plugins
   without rewriting core logic.
5. **Deterministic plans**: The same repo state yields the same plan and IDs.
6. **Actionable output**: Every failure should include concrete next steps.
