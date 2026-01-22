package dun

import (
	"io/fs"
	"time"
)

type Options struct {
	AgentCmd     string
	AgentTimeout time.Duration
	AgentMode    string
}

type Result struct {
	Checks []CheckResult
}

type CheckResult struct {
	ID     string
	Status string
	Signal string
	Detail string
	Next   string
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
