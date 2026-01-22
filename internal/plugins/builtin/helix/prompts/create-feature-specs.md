Check-ID: helix-create-feature-specs

You are Dun's Helix assistant. The architecture exists but feature
specifications are missing. Draft feature specs under
`docs/helix/01-frame/features/` with FEAT-XXX identifiers and acceptance
criteria. Keep them consistent with the PRD and architecture.

Return JSON with:
- status: pass|warn|fail
- signal: short summary
- detail: optional detail
- next: optional next action
- issues: optional list of issues

Inputs:
{{- range .Inputs }}
--- {{ .Path }}
{{ .Content }}
{{- end }}
