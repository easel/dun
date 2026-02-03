package dun

import (
	"errors"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Version string      `yaml:"version"`
	Agent   AgentConfig `yaml:"agent"`
	Go      GoConfig    `yaml:"go"`
}

type AgentConfig struct {
	Cmd        string `yaml:"cmd"`
	Harness    string `yaml:"harness"`
	TimeoutMS  int    `yaml:"timeout_ms"`
	Mode       string `yaml:"mode"`
	Automation string `yaml:"automation"`
}

type GoConfig struct {
	CoverageThreshold int `yaml:"coverage_threshold"`
}

const DefaultConfigPath = ".dun/config.yaml"

const DefaultConfigYAML = `version: "1"
agent:
  harness: codex
  automation: auto
  mode: auto
  timeout_ms: 300000
go:
  coverage_threshold: 80
`

func DefaultOptions() Options {
	return Options{
		AgentTimeout:   300 * time.Second,
		AgentMode:      "prompt",
		AutomationMode: "auto",
	}
}

func ApplyConfig(opts Options, cfg Config) Options {
	if cfg.Agent.Cmd != "" {
		opts.AgentCmd = cfg.Agent.Cmd
	}
	if cfg.Agent.Harness != "" {
		opts.AgentHarness = cfg.Agent.Harness
	}
	if cfg.Agent.TimeoutMS > 0 {
		opts.AgentTimeout = time.Duration(cfg.Agent.TimeoutMS) * time.Millisecond
	}
	if cfg.Agent.Mode != "" {
		opts.AgentMode = cfg.Agent.Mode
	}
	if cfg.Agent.Automation != "" {
		opts.AutomationMode = cfg.Agent.Automation
	}
	if cfg.Go.CoverageThreshold > 0 {
		opts.CoverageThreshold = cfg.Go.CoverageThreshold
	}
	return opts
}

func LoadConfig(root string, explicitPath string) (Config, bool, error) {
	path, err := resolveConfigPath(root, explicitPath)
	if err != nil {
		return Config{}, false, err
	}
	if path == "" {
		return Config{}, false, nil
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return Config{}, false, err
	}
	var cfg Config
	if err := yaml.Unmarshal(raw, &cfg); err != nil {
		return Config{}, false, err
	}
	return cfg, true, nil
}

func resolveConfigPath(root string, explicitPath string) (string, error) {
	if explicitPath != "" {
		path := explicitPath
		if !filepath.IsAbs(path) {
			path = filepath.Join(root, path)
		}
		if _, err := os.Stat(path); err != nil {
			return "", err
		}
		return path, nil
	}
	primary := filepath.Join(root, DefaultConfigPath)
	if _, err := os.Stat(primary); err == nil {
		return primary, nil
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return "", err
	}
	return "", nil
}
