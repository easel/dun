#!/usr/bin/env bash
set -euo pipefail

prompt="$(cat)"

if printf '%s' "$prompt" | grep -q "Check-ID: helix-create-architecture"; then
  cat <<'JSON'
{"status":"fail","signal":"architecture doc missing","detail":"docs/helix/02-design/architecture.md is missing","next":"Create docs/helix/02-design/architecture.md using the Helix template"}
JSON
  exit 0
fi

if printf '%s' "$prompt" | grep -q "Check-ID: helix-create-feature-specs"; then
  cat <<'JSON'
{"status":"fail","signal":"feature specs missing","detail":"docs/helix/01-frame/features/ has no FEAT files","next":"Create FEAT-XXX specs in docs/helix/01-frame/features/"}
JSON
  exit 0
fi

if printf '%s' "$prompt" | grep -q "Check-ID: helix-align-specs"; then
  cat <<'JSON'
{"status":"warn","signal":"alignment gaps found","detail":"PRD scope does not fully map to feature specs","next":"Review sections and update specs"}
JSON
  exit 0
fi

if printf '%s' "$prompt" | grep -q "Check-ID: helix-reconcile-stack"; then
  cat <<'JSON'
{"status":"warn","signal":"drift plan ready","detail":"Reconciliation plan generated","next":"Apply plan updates"}
JSON
  exit 0
fi

cat <<'JSON'
{"status":"fail","signal":"unknown check","detail":"agent could not identify check id"}
JSON
