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
	Cmd        string            `yaml:"cmd"`
	Harness    string            `yaml:"harness"`
	Model      string            `yaml:"model"`
	Models     map[string]string `yaml:"models"`
	TimeoutMS  int               `yaml:"timeout_ms"`
	Mode       string            `yaml:"mode"`
	Automation string            `yaml:"automation"`
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
  model: ""
  models: {}
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
	if cfg.Agent.Model != "" {
		opts.AgentModel = cfg.Agent.Model
	}
	if len(cfg.Agent.Models) > 0 {
		opts.AgentModels = make(map[string]string, len(cfg.Agent.Models))
		for harness, model := range cfg.Agent.Models {
			if model == "" {
				continue
			}
			opts.AgentModels[harness] = model
		}
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
	var merged Config
	loaded := false

	userPath, err := resolveUserConfigPath()
	if err != nil {
		return Config{}, false, err
	}
	if userPath != "" {
		cfg, err := loadConfigFile(userPath)
		if err != nil {
			return Config{}, false, err
		}
		merged = mergeConfig(merged, cfg)
		loaded = true
	}

	path, err := resolveConfigPath(root, explicitPath)
	if err != nil {
		return Config{}, false, err
	}
	if path != "" {
		cfg, err := loadConfigFile(path)
		if err != nil {
			return Config{}, false, err
		}
		merged = mergeConfig(merged, cfg)
		loaded = true
	}

	if !loaded {
		return Config{}, false, nil
	}
	return merged, true, nil
}

func loadConfigFile(path string) (Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}
	var cfg Config
	if err := yaml.Unmarshal(raw, &cfg); err != nil {
		return Config{}, err
	}
	return cfg, nil
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

func resolveUserConfigPath() (string, error) {
	var candidates []string
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		candidates = append(candidates, filepath.Join(xdg, "dun", "config.yaml"))
	}
	home, err := os.UserHomeDir()
	if err == nil && home != "" {
		candidates = append(candidates, filepath.Join(home, ".config", "dun", "config.yaml"))
		candidates = append(candidates, filepath.Join(home, ".dun", "config.yaml"))
	}

	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		} else if err != nil && !errors.Is(err, os.ErrNotExist) {
			return "", err
		}
	}
	return "", nil
}

func mergeConfig(base Config, override Config) Config {
	merged := base
	if override.Version != "" {
		merged.Version = override.Version
	}

	if override.Agent.Cmd != "" {
		merged.Agent.Cmd = override.Agent.Cmd
	}
	if override.Agent.Harness != "" {
		merged.Agent.Harness = override.Agent.Harness
	}
	if override.Agent.Model != "" {
		merged.Agent.Model = override.Agent.Model
	}
	if override.Agent.TimeoutMS > 0 {
		merged.Agent.TimeoutMS = override.Agent.TimeoutMS
	}
	if override.Agent.Mode != "" {
		merged.Agent.Mode = override.Agent.Mode
	}
	if override.Agent.Automation != "" {
		merged.Agent.Automation = override.Agent.Automation
	}
	if len(override.Agent.Models) > 0 {
		if merged.Agent.Models == nil {
			merged.Agent.Models = make(map[string]string, len(override.Agent.Models))
		}
		for key, value := range override.Agent.Models {
			merged.Agent.Models[key] = value
		}
	}

	if override.Go.CoverageThreshold > 0 {
		merged.Go.CoverageThreshold = override.Go.CoverageThreshold
	}

	return merged
}
