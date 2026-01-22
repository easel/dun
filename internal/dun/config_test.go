package dun

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadConfigDefaultPath(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "dun.yaml")
	content := "agent:\n  cmd: echo hi\n  timeout_ms: 120000\n  mode: auto\n  automation: plan\n"
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
	if opts.AgentTimeout != 120*time.Second {
		t.Fatalf("expected timeout 120s, got %s", opts.AgentTimeout)
	}
	if opts.AgentMode != "auto" {
		t.Fatalf("expected agent mode auto, got %q", opts.AgentMode)
	}
	if opts.AutomationMode != "plan" {
		t.Fatalf("expected automation plan, got %q", opts.AutomationMode)
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
