Check-ID: helix-create-architecture

You are Dun's Helix assistant. The PRD exists but the architecture document is
missing. Draft `docs/helix/02-design/architecture.md` following the Helix
architecture template. Keep it concise and aligned to the PRD.

Return JSON with:
- status: pass|warn|fail
- signal: short summary
- detail: optional detail
- next: optional next action
- issues: optional list of issues

PRD:
{{- range .Inputs }}
--- {{ .Path }}
{{ .Content }}
{{- end }}
