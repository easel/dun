# Risk Register

| ID | Risk | Probability | Impact | Mitigation |
| --- | --- | --- | --- | --- |
| R-001 | Slow checks degrade UX | Med | High | Timeouts, `--changed`, concurrency limits |
| R-002 | Noisy output reduces trust | Med | High | Strict output contracts, concise summaries |
| R-003 | Discovery misses checks | Med | Med | Expand discoverers, allow overrides |
| R-004 | Agent responses inconsistent | Med | High | Schema validation, prompt envelopes |
| R-005 | Plugin sprawl | Low | Med | Versioned manifests, minimal core APIs |
