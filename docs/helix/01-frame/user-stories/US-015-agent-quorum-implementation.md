---
dun:
  id: US-015
  depends_on:
  - F-018
---
# US-015: Implement Agent Quorum + Synthesis Commands

As a maintainer, I want one-shot quorum and synthesis commands that reuse the
loop quorum engine so I can obtain consensus or a merged result without
running a full iteration loop.

## Acceptance Criteria

- `dun quorum` parses `--task`, `--quorum`, `--harnesses`, and conflict flags.
- `dun synth` is shorthand for `dun quorum --synthesize`.
- Harness specs support `name@persona` and pass persona names through to the
  harness layer.
- Synthesis mode runs a meta-harness and returns merged output.
- Quorum summary metadata is emitted deterministically.
- `dun loop --quorum` uses the same quorum engine as one-shot commands.
