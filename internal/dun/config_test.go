package dun

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadConfigDefaultPath(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, DefaultConfigPath)
	if err := os.MkdirAll(filepath.Dir(cfgPath), 0755); err != nil {
		t.Fatalf("mkdir config dir: %v", err)
	}
	content := "agent:\n  cmd: echo hi\n  harness: codex\n  timeout_ms: 120000\n  mode: auto\n  automation: plan\n" +
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
	dir := t.TempDir()
	_, _, err := LoadConfig(dir, "missing.yaml")
	if err == nil {
		t.Fatalf("expected missing config error")
	}
}

func TestNormalizeAutomationModeDefault(t *testing.T) {
	mode, err := normalizeAutomationMode("")
	if err != nil {
		t.Fatalf("normalize automation: %v", err)
	}
	if mode != "auto" {
		t.Fatalf("expected auto, got %q", mode)
	}
}

func TestLoadConfigInvalidYAML(t *testing.T) {
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
