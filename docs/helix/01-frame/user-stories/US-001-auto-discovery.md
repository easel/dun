# US-001: Auto-Discover Repo Checks

As an agent operator, I want Dun to detect the right checks from repo signals
so I can run `dun check` without configuration.

## Acceptance Criteria

- Detect Go repositories via `go.mod`.
- Detect Helix workflow via `docs/helix/`.
- Check IDs and ordering are deterministic for the same repo state.
- No user configuration is required for core discovery.
