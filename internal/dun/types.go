package dun

import (
	"io/fs"
	"time"
)

type Options struct {
	AgentCmd          string
	AgentTimeout      time.Duration
	AgentMode         string
	AutomationMode    string
	CoverageThreshold int
}

type Result struct {
	Checks []CheckResult `json:"checks"`
}

type CheckResult struct {
	ID     string          `json:"id"`
	Status string          `json:"status"`
	Signal string          `json:"signal"`
	Detail string          `json:"detail,omitempty"`
	Next   string          `json:"next,omitempty"`
	Prompt *PromptEnvelope `json:"prompt,omitempty"`
	Issues []Issue         `json:"issues,omitempty"`
}

type Plugin struct {
	Manifest Manifest
	FS       fs.FS
	Base     string
}

type Manifest struct {
	ID          string    `yaml:"id"`
	Version     string    `yaml:"version"`
	Description string    `yaml:"description"`
	Priority    int       `yaml:"priority"`
	Triggers    []Trigger `yaml:"triggers"`
	Checks      []Check   `yaml:"checks"`
}

type Trigger struct {
	Type  string `yaml:"type"`
	Value string `yaml:"value"`
}

type Check struct {
	ID             string   `yaml:"id"`
	Description    string   `yaml:"description"`
	Type           string   `yaml:"type"`
	Phase          string   `yaml:"phase"`
	Priority       int      `yaml:"priority"`
	StateRules     string   `yaml:"state_rules"`
	GateFiles      []string `yaml:"gate_files"`
	Inputs         []string `yaml:"inputs"`
	Rules          []Rule   `yaml:"rules"`
	Conditions     []Rule   `yaml:"conditions"`
	Command        string   `yaml:"command"`
	Prompt         string   `yaml:"prompt"`
	ResponseSchema string   `yaml:"response_schema"`

	// Command check fields (US-012)
	Parser       string            `yaml:"parser"`        // text|lines|json|json-lines|regex
	SuccessExit  int               `yaml:"success_exit"`  // Exit code for pass (default 0)
	WarnExits    []int             `yaml:"warn_exits"`    // Exit codes for warn
	Timeout      string            `yaml:"timeout"`       // Duration string (default "5m")
	Shell        string            `yaml:"shell"`         // Shell command (default "sh -c")
	Env          map[string]string `yaml:"env"`           // Additional env vars
	IssuePath    string            `yaml:"issue_path"`    // JSONPath for issues
	IssuePattern string            `yaml:"issue_pattern"` // Regex pattern for issues
	IssueFields  IssueFieldMap     `yaml:"issue_fields"`  // Field mapping for JSON

	// Spec-binding fields (spec-enforcement checks)
	Bindings struct {
		Specs []struct {
			Pattern               string `yaml:"pattern"`
			ImplementationSection string `yaml:"implementation_section"`
			IDPattern             string `yaml:"id_pattern"`
		} `yaml:"specs"`
		Code []struct {
			Pattern     string `yaml:"pattern"`
			SpecComment string `yaml:"spec_comment"`
		} `yaml:"code"`
	} `yaml:"bindings"`
	BindingRules []BindingRule `yaml:"binding_rules"`

	// Change-cascade fields (spec-enforcement checks)
	CascadeRules []struct {
		Upstream    string `yaml:"upstream"`
		Downstreams []struct {
			Path     string   `yaml:"path"`
			Sections []string `yaml:"sections"`
			Required bool     `yaml:"required"`
		} `yaml:"downstreams"`
	} `yaml:"cascade_rules"`
	Trigger  string `yaml:"trigger"`  // git-diff|always
	Baseline string `yaml:"baseline"` // default: HEAD~1

	// Integration-contract fields (spec-enforcement checks)
	Contracts struct {
		Map         string `yaml:"map"`         // Path to integration-map.yaml
		Definitions string `yaml:"definitions"` // Glob for interface definitions
	} `yaml:"contracts"`
	ContractRules []struct {
		Type string `yaml:"type"` // all-providers-implemented, all-consumers-satisfied, no-circular-dependencies
	} `yaml:"contract_rules"`

	// Conflict-detection fields (spec-enforcement checks)
	Tracking struct {
		Manifest     string `yaml:"manifest"`      // Path to WIP manifest
		ClaimPattern string `yaml:"claim_pattern"` // Pattern in code marking claimed sections
	} `yaml:"tracking"`
	ConflictRules []struct {
		Type     string `yaml:"type"`     // no-overlap, claim-before-edit
		Scope    string `yaml:"scope"`    // file, function, line
		Required bool   `yaml:"required"` // If false, warn only
	} `yaml:"conflict_rules"`

	// Agent-rule-injection fields (spec-enforcement checks)
	BasePrompt  string `yaml:"base_prompt"` // Path to base prompt template
	InjectRules []struct {
		Source  string `yaml:"source"`  // File path or "from_registry"
		Section string `yaml:"section"` // Where to inject in prompt
	} `yaml:"inject_rules"`
	EnforceRules []struct {
		ID       string `yaml:"id"`
		Pattern  string `yaml:"pattern"`  // Regex to verify in output
		Required bool   `yaml:"required"` // Whether pattern is mandatory
	} `yaml:"enforce_rules"`
}

type Rule struct {
	Type     string `yaml:"type"`
	Path     string `yaml:"path"`
	Pattern  string `yaml:"pattern"`
	Expected int    `yaml:"expected"`
	Severity string `yaml:"severity"`
}

// IssueFieldMap maps JSON paths to issue fields for command check output parsing.
type IssueFieldMap struct {
	File     string `yaml:"file"`
	Line     string `yaml:"line"`
	Message  string `yaml:"message"`
	Severity string `yaml:"severity"`
}

// BindingRule defines a rule for spec-binding checks.
type BindingRule struct {
	Type        string  `yaml:"type"`         // bidirectional-coverage, no-orphan-code, no-orphan-specs
	MinCoverage float64 `yaml:"min_coverage"` // 0.0-1.0 for coverage rules
	WarnOnly    bool    `yaml:"warn_only"`    // If true, warn instead of fail
}

type PromptEnvelope struct {
	Kind           string         `json:"kind"`
	ID             string         `json:"id"`
	Title          string         `json:"title,omitempty"`
	Summary        string         `json:"summary,omitempty"`
	Prompt         string         `json:"prompt"`
	Inputs         []string       `json:"inputs,omitempty"`
	ResponseSchema string         `json:"response_schema,omitempty"`
	Callback       PromptCallback `json:"callback"`
}

type PromptCallback struct {
	Command string `json:"command"`
	Stdin   bool   `json:"stdin"`
}
