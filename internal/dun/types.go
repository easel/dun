package dun

import (
	"io/fs"
	"time"
)

type Options struct {
	AgentCmd        string
	AgentTimeout    time.Duration
	AgentMode       string
	AutomationMode  string
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
}

type Rule struct {
	Type     string `yaml:"type"`
	Path     string `yaml:"path"`
	Pattern  string `yaml:"pattern"`
	Expected int    `yaml:"expected"`
	Severity string `yaml:"severity"`
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
