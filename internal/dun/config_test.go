package dun

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func setTempUserConfig(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, "xdg"))
	return home
}

func TestLoadConfigDefaultPath(t *testing.T) {
	_ = setTempUserConfig(t)
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, DefaultConfigPath)
	if err := os.MkdirAll(filepath.Dir(cfgPath), 0755); err != nil {
		t.Fatalf("mkdir config dir: %v", err)
	}
	content := "agent:\n  cmd: echo hi\n  harness: codex\n  model: o3\n  models:\n    claude: sonnet\n  timeout_ms: 120000\n  mode: auto\n  automation: plan\n" +
		"go:\n  coverage_threshold: 95\n"
	if err := os.WriteFile(cfgPath, []byte(content), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, loaded, err := LoadConfig(dir, "")
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if !loaded {
		t.Fatalf("expected config to load")
	}

	opts := ApplyConfig(DefaultOptions(), cfg)
	if opts.AgentCmd != "echo hi" {
		t.Fatalf("expected agent cmd, got %q", opts.AgentCmd)
	}
	if opts.AgentHarness != "codex" {
		t.Fatalf("expected agent harness codex, got %q", opts.AgentHarness)
	}
	if opts.AgentModel != "o3" {
		t.Fatalf("expected agent model o3, got %q", opts.AgentModel)
	}
	if opts.AgentModels["claude"] != "sonnet" {
		t.Fatalf("expected agent model override for claude, got %q", opts.AgentModels["claude"])
	}
	if opts.AgentTimeout != 120*time.Second {
		t.Fatalf("expected timeout 120s, got %s", opts.AgentTimeout)
	}
	if opts.AgentMode != "auto" {
		t.Fatalf("expected agent mode auto, got %q", opts.AgentMode)
	}
	if opts.AutomationMode != "plan" {
		t.Fatalf("expected automation plan, got %q", opts.AutomationMode)
	}
	if opts.CoverageThreshold != 95 {
		t.Fatalf("expected coverage threshold 95, got %d", opts.CoverageThreshold)
	}
}

func TestLoadConfigAbsent(t *testing.T) {
	_ = setTempUserConfig(t)
	dir := t.TempDir()
	_, loaded, err := LoadConfig(dir, "")
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if loaded {
		t.Fatalf("expected no config to load")
	}
}

func TestLoadConfigExplicitMissing(t *testing.T) {
	_ = setTempUserConfig(t)
	dir := t.TempDir()
	_, _, err := LoadConfig(dir, "missing.yaml")
	if err == nil {
		t.Fatalf("expected missing config error")
	}
}

func TestNormalizeAutomationModeDefault(t *testing.T) {
	_ = setTempUserConfig(t)
	mode, err := normalizeAutomationMode("")
	if err != nil {
		t.Fatalf("normalize automation: %v", err)
	}
	if mode != "auto" {
		t.Fatalf("expected auto, got %q", mode)
	}
}

func TestLoadConfigInvalidYAML(t *testing.T) {
	_ = setTempUserConfig(t)
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, ".dun", "config.yaml")
	if err := os.MkdirAll(filepath.Dir(cfgPath), 0755); err != nil {
		t.Fatalf("mkdir config dir: %v", err)
	}
	if err := os.WriteFile(cfgPath, []byte("agent:\n  cmd: ["), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	_, _, err := LoadConfig(dir, "")
	if err == nil {
		t.Fatalf("expected yaml parse error")
	}
}

func TestLoadConfigReadError(t *testing.T) {
	_ = setTempUserConfig(t)
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	if err := os.MkdirAll(cfgPath, 0755); err != nil {
		t.Fatalf("mkdir config dir: %v", err)
	}
	_, _, err := LoadConfig(dir, "config.yaml")
	if err == nil {
		t.Fatalf("expected read error")
	}
}

func TestLoadConfigRelativePath(t *testing.T) {
	_ = setTempUserConfig(t)
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "custom.yaml")
	if err := os.WriteFile(cfgPath, []byte("agent:\n  automation: manual\n"), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	cfg, loaded, err := LoadConfig(dir, "custom.yaml")
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if !loaded {
		t.Fatalf("expected loaded")
	}
	opts := ApplyConfig(DefaultOptions(), cfg)
	if opts.AutomationMode != "manual" {
		t.Fatalf("expected manual, got %q", opts.AutomationMode)
	}
}

func TestLoadConfigDefaultPathStatError(t *testing.T) {
	_ = setTempUserConfig(t)
	dir := t.TempDir()
	dunPath := filepath.Join(dir, ".dun")
	if err := os.WriteFile(dunPath, []byte("not a dir"), 0644); err != nil {
		t.Fatalf("write .dun: %v", err)
	}
	_, _, err := LoadConfig(dir, "")
	if err == nil {
		t.Fatalf("expected stat error")
	}
}

func TestLoadConfigAbsolutePath(t *testing.T) {
	_ = setTempUserConfig(t)
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "abs.yaml")
	if err := os.WriteFile(cfgPath, []byte("agent:\n  automation: manual\n"), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	cfg, loaded, err := LoadConfig(dir, cfgPath)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if !loaded {
		t.Fatalf("expected loaded")
	}
	opts := ApplyConfig(DefaultOptions(), cfg)
	if opts.AutomationMode != "manual" {
		t.Fatalf("expected manual, got %q", opts.AutomationMode)
	}
}

func TestLoadConfigUserOnly(t *testing.T) {
	_ = setTempUserConfig(t)
	dir := t.TempDir()
	userCfgPath := filepath.Join(os.Getenv("XDG_CONFIG_HOME"), "dun", "config.yaml")
	if err := os.MkdirAll(filepath.Dir(userCfgPath), 0755); err != nil {
		t.Fatalf("mkdir user config dir: %v", err)
	}
	content := "agent:\n  harness: claude\n  model: user-model\n  models:\n    codex: user-codex\n"
	if err := os.WriteFile(userCfgPath, []byte(content), 0644); err != nil {
		t.Fatalf("write user config: %v", err)
	}

	cfg, loaded, err := LoadConfig(dir, "")
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if !loaded {
		t.Fatalf("expected user config to load")
	}
	if cfg.Agent.Harness != "claude" {
		t.Fatalf("expected harness claude, got %q", cfg.Agent.Harness)
	}
	if cfg.Agent.Model != "user-model" {
		t.Fatalf("expected model user-model, got %q", cfg.Agent.Model)
	}
	if cfg.Agent.Models["codex"] != "user-codex" {
		t.Fatalf("expected user codex model, got %q", cfg.Agent.Models["codex"])
	}
}

func TestLoadConfigUserAndProjectMerge(t *testing.T) {
	_ = setTempUserConfig(t)
	dir := t.TempDir()
	userCfgPath := filepath.Join(os.Getenv("XDG_CONFIG_HOME"), "dun", "config.yaml")
	if err := os.MkdirAll(filepath.Dir(userCfgPath), 0755); err != nil {
		t.Fatalf("mkdir user config dir: %v", err)
	}
	userContent := "agent:\n  harness: gemini\n  model: user-model\n  models:\n    codex: user-codex\n    claude: user-claude\n" +
		"go:\n  coverage_threshold: 70\n"
	if err := os.WriteFile(userCfgPath, []byte(userContent), 0644); err != nil {
		t.Fatalf("write user config: %v", err)
	}

	projectCfgPath := filepath.Join(dir, DefaultConfigPath)
	if err := os.MkdirAll(filepath.Dir(projectCfgPath), 0755); err != nil {
		t.Fatalf("mkdir project config dir: %v", err)
	}
	projectContent := "agent:\n  harness: codex\n  model: project-model\n  models:\n    codex: project-codex\n" +
		"go:\n  coverage_threshold: 90\n"
	if err := os.WriteFile(projectCfgPath, []byte(projectContent), 0644); err != nil {
		t.Fatalf("write project config: %v", err)
	}

	cfg, loaded, err := LoadConfig(dir, "")
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if !loaded {
		t.Fatalf("expected merged config to load")
	}
	if cfg.Agent.Harness != "codex" {
		t.Fatalf("expected harness codex, got %q", cfg.Agent.Harness)
	}
	if cfg.Agent.Model != "project-model" {
		t.Fatalf("expected model project-model, got %q", cfg.Agent.Model)
	}
	if cfg.Agent.Models["codex"] != "project-codex" {
		t.Fatalf("expected project codex model, got %q", cfg.Agent.Models["codex"])
	}
	if cfg.Agent.Models["claude"] != "user-claude" {
		t.Fatalf("expected user claude model, got %q", cfg.Agent.Models["claude"])
	}
	if cfg.Go.CoverageThreshold != 90 {
		t.Fatalf("expected coverage 90, got %d", cfg.Go.CoverageThreshold)
	}
}

func TestLoadConfigUserInvalid(t *testing.T) {
	_ = setTempUserConfig(t)
	dir := t.TempDir()
	userCfgPath := filepath.Join(os.Getenv("XDG_CONFIG_HOME"), "dun", "config.yaml")
	if err := os.MkdirAll(filepath.Dir(userCfgPath), 0755); err != nil {
		t.Fatalf("mkdir user config dir: %v", err)
	}
	if err := os.WriteFile(userCfgPath, []byte("agent:\n  cmd: ["), 0644); err != nil {
		t.Fatalf("write user config: %v", err)
	}

	_, _, err := LoadConfig(dir, "")
	if err == nil {
		t.Fatalf("expected user config parse error")
	}
}
