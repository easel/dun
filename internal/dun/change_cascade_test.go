package dun

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestChangeCascade_GitDiffTrigger_NoChanges(t *testing.T) {
	root := t.TempDir()

	// Mock git diff to return no changes
	origGitDiff := gitDiffFunc
	gitDiffFunc = func(root, baseline string) ([]string, error) {
		return nil, nil
	}
	defer func() { gitDiffFunc = origGitDiff }()

	check := Check{
		ID:      "test-cascade",
		Trigger: "git-diff",
	}

	result, err := runChangeCascadeCheck(root, check)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Status != "pass" {
		t.Errorf("expected pass, got %s", result.Status)
	}
	if !strings.Contains(result.Signal, "no upstream changes") {
		t.Errorf("expected 'no upstream changes' in signal, got %s", result.Signal)
	}
}

func TestChangeCascade_GitDiffTrigger_Error(t *testing.T) {
	root := t.TempDir()

	// Mock git diff to return an error
	origGitDiff := gitDiffFunc
	gitDiffFunc = func(root, baseline string) ([]string, error) {
		return nil, errors.New("git diff failed")
	}
	defer func() { gitDiffFunc = origGitDiff }()

	check := Check{
		ID:      "test-cascade",
		Trigger: "git-diff",
	}

	result, err := runChangeCascadeCheck(root, check)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Status != "skip" {
		t.Errorf("expected skip, got %s", result.Status)
	}
	if !strings.Contains(result.Signal, "cannot determine changes") {
		t.Errorf("expected 'cannot determine changes' in signal, got %s", result.Signal)
	}
}

func TestChangeCascade_GitDiffTrigger_WithStaleDownstreams(t *testing.T) {
	root := t.TempDir()

	// Create test files
	setupCascadeTestFiles(t, root)

	// Mock git diff to return upstream change
	origGitDiff := gitDiffFunc
	gitDiffFunc = func(r, baseline string) ([]string, error) {
		return []string{"docs/prd.md"}, nil
	}
	defer func() { gitDiffFunc = origGitDiff }()

	check := Check{
		ID:      "test-cascade",
		Trigger: "git-diff",
		CascadeRules: []struct {
			Upstream    string `yaml:"upstream"`
			Downstreams []struct {
				Path     string   `yaml:"path"`
				Sections []string `yaml:"sections"`
				Required bool     `yaml:"required"`
			} `yaml:"downstreams"`
		}{
			{
				Upstream: "docs/prd.md",
				Downstreams: []struct {
					Path     string   `yaml:"path"`
					Sections []string `yaml:"sections"`
					Required bool     `yaml:"required"`
				}{
					{Path: "docs/architecture.md", Required: true},
				},
			},
		},
	}

	result, err := runChangeCascadeCheck(root, check)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Status != "fail" {
		t.Errorf("expected fail, got %s", result.Status)
	}
	if len(result.Issues) != 1 {
		t.Errorf("expected 1 issue, got %d", len(result.Issues))
	}
	if !strings.Contains(result.Issues[0].Summary, "architecture.md") {
		t.Errorf("expected issue about architecture.md, got %s", result.Issues[0].Summary)
	}
}

func TestChangeCascade_GitDiffTrigger_AllDownstreamsUpdated(t *testing.T) {
	root := t.TempDir()

	// Create test files
	setupCascadeTestFiles(t, root)

	// Mock git diff to return both upstream and downstream changes
	origGitDiff := gitDiffFunc
	gitDiffFunc = func(r, baseline string) ([]string, error) {
		return []string{"docs/prd.md", "docs/architecture.md"}, nil
	}
	defer func() { gitDiffFunc = origGitDiff }()

	check := Check{
		ID:      "test-cascade",
		Trigger: "git-diff",
		CascadeRules: []struct {
			Upstream    string `yaml:"upstream"`
			Downstreams []struct {
				Path     string   `yaml:"path"`
				Sections []string `yaml:"sections"`
				Required bool     `yaml:"required"`
			} `yaml:"downstreams"`
		}{
			{
				Upstream: "docs/prd.md",
				Downstreams: []struct {
					Path     string   `yaml:"path"`
					Sections []string `yaml:"sections"`
					Required bool     `yaml:"required"`
				}{
					{Path: "docs/architecture.md", Required: true},
				},
			},
		},
	}

	result, err := runChangeCascadeCheck(root, check)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Status != "pass" {
		t.Errorf("expected pass, got %s", result.Status)
	}
}

func TestChangeCascade_GitDiffTrigger_OptionalDownstreamStale(t *testing.T) {
	root := t.TempDir()

	// Create test files
	setupCascadeTestFiles(t, root)

	// Mock git diff to return upstream change
	origGitDiff := gitDiffFunc
	gitDiffFunc = func(r, baseline string) ([]string, error) {
		return []string{"docs/prd.md"}, nil
	}
	defer func() { gitDiffFunc = origGitDiff }()

	check := Check{
		ID:      "test-cascade",
		Trigger: "git-diff",
		CascadeRules: []struct {
			Upstream    string `yaml:"upstream"`
			Downstreams []struct {
				Path     string   `yaml:"path"`
				Sections []string `yaml:"sections"`
				Required bool     `yaml:"required"`
			} `yaml:"downstreams"`
		}{
			{
				Upstream: "docs/prd.md",
				Downstreams: []struct {
					Path     string   `yaml:"path"`
					Sections []string `yaml:"sections"`
					Required bool     `yaml:"required"`
				}{
					{Path: "docs/architecture.md", Required: false},
				},
			},
		},
	}

	result, err := runChangeCascadeCheck(root, check)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should warn for optional downstream, not fail
	if result.Status != "warn" {
		t.Errorf("expected warn, got %s", result.Status)
	}
}

func TestChangeCascade_AlwaysTrigger_StaleByMtime(t *testing.T) {
	root := t.TempDir()

	// Create test files
	setupCascadeTestFiles(t, root)

	// Mock mtime to make upstream newer
	now := time.Now()
	origMtimeFunc := getFileMtimeFunc
	getFileMtimeFunc = func(path string) (time.Time, error) {
		if strings.Contains(path, "prd.md") {
			return now, nil
		}
		return now.Add(-time.Hour), nil // Downstream is older
	}
	defer func() { getFileMtimeFunc = origMtimeFunc }()

	check := Check{
		ID:      "test-cascade",
		Trigger: "always",
		CascadeRules: []struct {
			Upstream    string `yaml:"upstream"`
			Downstreams []struct {
				Path     string   `yaml:"path"`
				Sections []string `yaml:"sections"`
				Required bool     `yaml:"required"`
			} `yaml:"downstreams"`
		}{
			{
				Upstream: "docs/prd.md",
				Downstreams: []struct {
					Path     string   `yaml:"path"`
					Sections []string `yaml:"sections"`
					Required bool     `yaml:"required"`
				}{
					{Path: "docs/architecture.md", Required: true},
				},
			},
		},
	}

	result, err := runChangeCascadeCheck(root, check)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Status != "fail" {
		t.Errorf("expected fail, got %s", result.Status)
	}
	if len(result.Issues) != 1 {
		t.Errorf("expected 1 issue, got %d", len(result.Issues))
	}
}

func TestChangeCascade_AlwaysTrigger_NotStale(t *testing.T) {
	root := t.TempDir()

	// Create test files
	setupCascadeTestFiles(t, root)

	// Mock mtime to make downstream newer
	now := time.Now()
	origMtimeFunc := getFileMtimeFunc
	getFileMtimeFunc = func(path string) (time.Time, error) {
		if strings.Contains(path, "prd.md") {
			return now.Add(-time.Hour), nil // Upstream is older
		}
		return now, nil
	}
	defer func() { getFileMtimeFunc = origMtimeFunc }()

	check := Check{
		ID:      "test-cascade",
		Trigger: "always",
		CascadeRules: []struct {
			Upstream    string `yaml:"upstream"`
			Downstreams []struct {
				Path     string   `yaml:"path"`
				Sections []string `yaml:"sections"`
				Required bool     `yaml:"required"`
			} `yaml:"downstreams"`
		}{
			{
				Upstream: "docs/prd.md",
				Downstreams: []struct {
					Path     string   `yaml:"path"`
					Sections []string `yaml:"sections"`
					Required bool     `yaml:"required"`
				}{
					{Path: "docs/architecture.md", Required: true},
				},
			},
		},
	}

	result, err := runChangeCascadeCheck(root, check)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Status != "pass" {
		t.Errorf("expected pass, got %s", result.Status)
	}
}

func TestChangeCascade_DefaultBaseline(t *testing.T) {
	root := t.TempDir()

	var capturedBaseline string
	origGitDiff := gitDiffFunc
	gitDiffFunc = func(r, baseline string) ([]string, error) {
		capturedBaseline = baseline
		return nil, nil
	}
	defer func() { gitDiffFunc = origGitDiff }()

	check := Check{
		ID:      "test-cascade",
		Trigger: "git-diff",
		// No baseline specified
	}

	_, err := runChangeCascadeCheck(root, check)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if capturedBaseline != "HEAD~1" {
		t.Errorf("expected default baseline HEAD~1, got %s", capturedBaseline)
	}
}

func TestChangeCascade_CustomBaseline(t *testing.T) {
	root := t.TempDir()

	var capturedBaseline string
	origGitDiff := gitDiffFunc
	gitDiffFunc = func(r, baseline string) ([]string, error) {
		capturedBaseline = baseline
		return nil, nil
	}
	defer func() { gitDiffFunc = origGitDiff }()

	check := Check{
		ID:       "test-cascade",
		Trigger:  "git-diff",
		Baseline: "main",
	}

	_, err := runChangeCascadeCheck(root, check)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if capturedBaseline != "main" {
		t.Errorf("expected baseline main, got %s", capturedBaseline)
	}
}

func TestChangeCascade_GlobPattern(t *testing.T) {
	root := t.TempDir()

	// Create test files with glob pattern
	if err := os.MkdirAll(filepath.Join(root, "docs", "features"), 0755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(root, "docs", "prd.md"), "# PRD")
	writeFile(t, filepath.Join(root, "docs", "features", "feat-1.md"), "# Feature 1")
	writeFile(t, filepath.Join(root, "docs", "features", "feat-2.md"), "# Feature 2")

	// Mock git diff to return upstream change
	origGitDiff := gitDiffFunc
	gitDiffFunc = func(r, baseline string) ([]string, error) {
		return []string{"docs/prd.md"}, nil
	}
	defer func() { gitDiffFunc = origGitDiff }()

	check := Check{
		ID:      "test-cascade",
		Trigger: "git-diff",
		CascadeRules: []struct {
			Upstream    string `yaml:"upstream"`
			Downstreams []struct {
				Path     string   `yaml:"path"`
				Sections []string `yaml:"sections"`
				Required bool     `yaml:"required"`
			} `yaml:"downstreams"`
		}{
			{
				Upstream: "docs/prd.md",
				Downstreams: []struct {
					Path     string   `yaml:"path"`
					Sections []string `yaml:"sections"`
					Required bool     `yaml:"required"`
				}{
					{Path: "docs/features/*.md", Required: true},
				},
			},
		},
	}

	result, err := runChangeCascadeCheck(root, check)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should find 2 stale downstream files
	if result.Status != "fail" {
		t.Errorf("expected fail, got %s", result.Status)
	}
	if len(result.Issues) != 2 {
		t.Errorf("expected 2 issues, got %d", len(result.Issues))
	}
}

func TestChangeCascade_UpstreamGlobPattern(t *testing.T) {
	root := t.TempDir()

	// Create test files
	if err := os.MkdirAll(filepath.Join(root, "docs", "features"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "internal"), 0755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(root, "docs", "features", "feat-1.md"), "# Feature 1")
	writeFile(t, filepath.Join(root, "internal", "handler.go"), "package internal")

	// Mock git diff to return feature file change
	origGitDiff := gitDiffFunc
	gitDiffFunc = func(r, baseline string) ([]string, error) {
		return []string{"docs/features/feat-1.md"}, nil
	}
	defer func() { gitDiffFunc = origGitDiff }()

	check := Check{
		ID:      "test-cascade",
		Trigger: "git-diff",
		CascadeRules: []struct {
			Upstream    string `yaml:"upstream"`
			Downstreams []struct {
				Path     string   `yaml:"path"`
				Sections []string `yaml:"sections"`
				Required bool     `yaml:"required"`
			} `yaml:"downstreams"`
		}{
			{
				Upstream: "docs/features/*.md",
				Downstreams: []struct {
					Path     string   `yaml:"path"`
					Sections []string `yaml:"sections"`
					Required bool     `yaml:"required"`
				}{
					{Path: "internal/*.go", Required: false},
				},
			},
		},
	}

	result, err := runChangeCascadeCheck(root, check)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should find 1 stale downstream
	if result.Status != "warn" {
		t.Errorf("expected warn, got %s", result.Status)
	}
	if len(result.Issues) != 1 {
		t.Errorf("expected 1 issue, got %d", len(result.Issues))
	}
}

func TestChangeCascade_MixedRequiredOptional(t *testing.T) {
	root := t.TempDir()

	// Create test files
	if err := os.MkdirAll(filepath.Join(root, "docs"), 0755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(root, "docs", "prd.md"), "# PRD")
	writeFile(t, filepath.Join(root, "docs", "architecture.md"), "# Arch")
	writeFile(t, filepath.Join(root, "docs", "optional.md"), "# Optional")

	// Mock git diff to return upstream change
	origGitDiff := gitDiffFunc
	gitDiffFunc = func(r, baseline string) ([]string, error) {
		return []string{"docs/prd.md"}, nil
	}
	defer func() { gitDiffFunc = origGitDiff }()

	check := Check{
		ID:      "test-cascade",
		Trigger: "git-diff",
		CascadeRules: []struct {
			Upstream    string `yaml:"upstream"`
			Downstreams []struct {
				Path     string   `yaml:"path"`
				Sections []string `yaml:"sections"`
				Required bool     `yaml:"required"`
			} `yaml:"downstreams"`
		}{
			{
				Upstream: "docs/prd.md",
				Downstreams: []struct {
					Path     string   `yaml:"path"`
					Sections []string `yaml:"sections"`
					Required bool     `yaml:"required"`
				}{
					{Path: "docs/architecture.md", Required: true},
					{Path: "docs/optional.md", Required: false},
				},
			},
		},
	}

	result, err := runChangeCascadeCheck(root, check)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should fail because of required downstream, but also count optional
	if result.Status != "fail" {
		t.Errorf("expected fail, got %s", result.Status)
	}
	if len(result.Issues) != 2 {
		t.Errorf("expected 2 issues, got %d", len(result.Issues))
	}
	if !strings.Contains(result.Detail, "1 required") || !strings.Contains(result.Detail, "1 optional") {
		t.Errorf("expected '1 required, 1 optional' in detail, got %s", result.Detail)
	}
}

func TestChangeCascade_MultipleRules(t *testing.T) {
	root := t.TempDir()

	// Create test files
	if err := os.MkdirAll(filepath.Join(root, "docs"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "internal"), 0755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(root, "docs", "prd.md"), "# PRD")
	writeFile(t, filepath.Join(root, "docs", "architecture.md"), "# Arch")
	writeFile(t, filepath.Join(root, "docs", "api.md"), "# API")
	writeFile(t, filepath.Join(root, "internal", "handler.go"), "package internal")

	// Mock git diff to return both upstream files changed
	origGitDiff := gitDiffFunc
	gitDiffFunc = func(r, baseline string) ([]string, error) {
		return []string{"docs/prd.md", "docs/api.md"}, nil
	}
	defer func() { gitDiffFunc = origGitDiff }()

	check := Check{
		ID:      "test-cascade",
		Trigger: "git-diff",
		CascadeRules: []struct {
			Upstream    string `yaml:"upstream"`
			Downstreams []struct {
				Path     string   `yaml:"path"`
				Sections []string `yaml:"sections"`
				Required bool     `yaml:"required"`
			} `yaml:"downstreams"`
		}{
			{
				Upstream: "docs/prd.md",
				Downstreams: []struct {
					Path     string   `yaml:"path"`
					Sections []string `yaml:"sections"`
					Required bool     `yaml:"required"`
				}{
					{Path: "docs/architecture.md", Required: true},
				},
			},
			{
				Upstream: "docs/api.md",
				Downstreams: []struct {
					Path     string   `yaml:"path"`
					Sections []string `yaml:"sections"`
					Required bool     `yaml:"required"`
				}{
					{Path: "internal/*.go", Required: true},
				},
			},
		},
	}

	result, err := runChangeCascadeCheck(root, check)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should find 2 stale downstreams from 2 different rules
	if result.Status != "fail" {
		t.Errorf("expected fail, got %s", result.Status)
	}
	if len(result.Issues) != 2 {
		t.Errorf("expected 2 issues, got %d", len(result.Issues))
	}
}

func TestChangeCascade_NoMatchingUpstream(t *testing.T) {
	root := t.TempDir()

	// Create test files
	if err := os.MkdirAll(filepath.Join(root, "docs"), 0755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(root, "docs", "prd.md"), "# PRD")
	writeFile(t, filepath.Join(root, "docs", "architecture.md"), "# Arch")

	// Mock git diff to return a file that doesn't match any rule
	origGitDiff := gitDiffFunc
	gitDiffFunc = func(r, baseline string) ([]string, error) {
		return []string{"docs/other.md"}, nil
	}
	defer func() { gitDiffFunc = origGitDiff }()

	check := Check{
		ID:      "test-cascade",
		Trigger: "git-diff",
		CascadeRules: []struct {
			Upstream    string `yaml:"upstream"`
			Downstreams []struct {
				Path     string   `yaml:"path"`
				Sections []string `yaml:"sections"`
				Required bool     `yaml:"required"`
			} `yaml:"downstreams"`
		}{
			{
				Upstream: "docs/prd.md",
				Downstreams: []struct {
					Path     string   `yaml:"path"`
					Sections []string `yaml:"sections"`
					Required bool     `yaml:"required"`
				}{
					{Path: "docs/architecture.md", Required: true},
				},
			},
		},
	}

	result, err := runChangeCascadeCheck(root, check)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// No upstream matched, so all downstreams are up to date
	if result.Status != "pass" {
		t.Errorf("expected pass, got %s", result.Status)
	}
}

func TestMatchPattern(t *testing.T) {
	tests := []struct {
		name     string
		files    []string
		pattern  string
		expected []string
	}{
		{
			name:     "exact match",
			files:    []string{"docs/prd.md", "docs/other.md"},
			pattern:  "docs/prd.md",
			expected: []string{"docs/prd.md"},
		},
		{
			name:     "glob pattern",
			files:    []string{"docs/prd.md", "docs/arch.md", "internal/handler.go"},
			pattern:  "docs/*.md",
			expected: []string{"docs/prd.md", "docs/arch.md"},
		},
		{
			name:     "basename pattern",
			files:    []string{"docs/prd.md", "internal/handler.go"},
			pattern:  "*.md",
			expected: []string{"docs/prd.md"},
		},
		{
			name:     "no matches",
			files:    []string{"docs/prd.md"},
			pattern:  "internal/*.go",
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := matchPattern(tt.files, tt.pattern)
			if len(result) != len(tt.expected) {
				t.Errorf("expected %d matches, got %d", len(tt.expected), len(result))
				return
			}
			for i, exp := range tt.expected {
				if result[i] != exp {
					t.Errorf("expected match[%d]=%s, got %s", i, exp, result[i])
				}
			}
		})
	}
}

func TestGlobFiles(t *testing.T) {
	root := t.TempDir()

	// Create test files
	if err := os.MkdirAll(filepath.Join(root, "docs"), 0755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(root, "docs", "one.md"), "# One")
	writeFile(t, filepath.Join(root, "docs", "two.md"), "# Two")
	writeFile(t, filepath.Join(root, "docs", "readme.txt"), "text")

	tests := []struct {
		name     string
		pattern  string
		expected int
	}{
		{
			name:     "match md files",
			pattern:  "docs/*.md",
			expected: 2,
		},
		{
			name:     "match txt files",
			pattern:  "docs/*.txt",
			expected: 1,
		},
		{
			name:     "match all files",
			pattern:  "docs/*",
			expected: 3,
		},
		{
			name:     "no matches",
			pattern:  "internal/*.go",
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := globFiles(root, tt.pattern)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(result) != tt.expected {
				t.Errorf("expected %d files, got %d: %v", tt.expected, len(result), result)
			}
		})
	}
}

func TestFindStaleDownstreams(t *testing.T) {
	root := t.TempDir()

	// Create test files
	if err := os.MkdirAll(filepath.Join(root, "docs"), 0755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(root, "docs", "one.md"), "# One")
	writeFile(t, filepath.Join(root, "docs", "two.md"), "# Two")

	tests := []struct {
		name       string
		ds         Downstream
		changedSet map[string]bool
		expected   int
	}{
		{
			name:       "all stale",
			ds:         Downstream{Path: "docs/*.md"},
			changedSet: map[string]bool{},
			expected:   2,
		},
		{
			name:       "one updated",
			ds:         Downstream{Path: "docs/*.md"},
			changedSet: map[string]bool{"docs/one.md": true},
			expected:   1,
		},
		{
			name:       "all updated",
			ds:         Downstream{Path: "docs/*.md"},
			changedSet: map[string]bool{"docs/one.md": true, "docs/two.md": true},
			expected:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := findStaleDownstreams(root, tt.ds, tt.changedSet)
			if len(result) != tt.expected {
				t.Errorf("expected %d stale, got %d: %v", tt.expected, len(result), result)
			}
		})
	}
}

func TestExtractCascadeConfig(t *testing.T) {
	check := Check{
		Trigger:  "always",
		Baseline: "main",
		CascadeRules: []struct {
			Upstream    string `yaml:"upstream"`
			Downstreams []struct {
				Path     string   `yaml:"path"`
				Sections []string `yaml:"sections"`
				Required bool     `yaml:"required"`
			} `yaml:"downstreams"`
		}{
			{
				Upstream: "docs/prd.md",
				Downstreams: []struct {
					Path     string   `yaml:"path"`
					Sections []string `yaml:"sections"`
					Required bool     `yaml:"required"`
				}{
					{Path: "docs/arch.md", Sections: []string{"## Overview"}, Required: true},
				},
			},
		},
	}

	config := extractCascadeConfig(check)

	if config.Trigger != "always" {
		t.Errorf("expected trigger 'always', got %s", config.Trigger)
	}
	if config.Baseline != "main" {
		t.Errorf("expected baseline 'main', got %s", config.Baseline)
	}
	if len(config.CascadeRules) != 1 {
		t.Errorf("expected 1 rule, got %d", len(config.CascadeRules))
	}
	if config.CascadeRules[0].Upstream != "docs/prd.md" {
		t.Errorf("expected upstream 'docs/prd.md', got %s", config.CascadeRules[0].Upstream)
	}
	if len(config.CascadeRules[0].Downstreams) != 1 {
		t.Errorf("expected 1 downstream, got %d", len(config.CascadeRules[0].Downstreams))
	}
	ds := config.CascadeRules[0].Downstreams[0]
	if ds.Path != "docs/arch.md" {
		t.Errorf("expected path 'docs/arch.md', got %s", ds.Path)
	}
	if !ds.Required {
		t.Error("expected required to be true")
	}
	if len(ds.Sections) != 1 || ds.Sections[0] != "## Overview" {
		t.Errorf("expected sections ['## Overview'], got %v", ds.Sections)
	}
}

func setupCascadeTestFiles(t *testing.T, root string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(root, "docs"), 0755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(root, "docs", "prd.md"), "# PRD\n\nRequirements here.")
	writeFile(t, filepath.Join(root, "docs", "architecture.md"), "# Architecture\n\nDesign here.")
}

func TestChangeCascade_EmptyRules(t *testing.T) {
	root := t.TempDir()

	// Mock git diff to return changes
	origGitDiff := gitDiffFunc
	gitDiffFunc = func(r, baseline string) ([]string, error) {
		return []string{"docs/prd.md"}, nil
	}
	defer func() { gitDiffFunc = origGitDiff }()

	check := Check{
		ID:           "test-cascade",
		Trigger:      "git-diff",
		CascadeRules: nil, // No rules
	}

	result, err := runChangeCascadeCheck(root, check)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// No rules means nothing to check, should pass
	if result.Status != "pass" {
		t.Errorf("expected pass, got %s", result.Status)
	}
}

func TestChangeCascade_AlwaysTrigger_NoFilesExist(t *testing.T) {
	root := t.TempDir()
	// Don't create any files

	check := Check{
		ID:      "test-cascade",
		Trigger: "always",
		CascadeRules: []struct {
			Upstream    string `yaml:"upstream"`
			Downstreams []struct {
				Path     string   `yaml:"path"`
				Sections []string `yaml:"sections"`
				Required bool     `yaml:"required"`
			} `yaml:"downstreams"`
		}{
			{
				Upstream: "docs/prd.md",
				Downstreams: []struct {
					Path     string   `yaml:"path"`
					Sections []string `yaml:"sections"`
					Required bool     `yaml:"required"`
				}{
					{Path: "docs/architecture.md", Required: true},
				},
			},
		},
	}

	result, err := runChangeCascadeCheck(root, check)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// No files exist, so nothing to check
	if result.Status != "pass" {
		t.Errorf("expected pass (no files to check), got %s", result.Status)
	}
}

func TestChangeCascade_AlwaysTrigger_UpstreamExistsNoDownstream(t *testing.T) {
	root := t.TempDir()

	// Create only upstream file
	if err := os.MkdirAll(filepath.Join(root, "docs"), 0755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(root, "docs", "prd.md"), "# PRD")

	// Mock mtime (shouldn't matter since downstream doesn't exist)
	now := time.Now()
	origMtimeFunc := getFileMtimeFunc
	getFileMtimeFunc = func(path string) (time.Time, error) {
		return now, nil
	}
	defer func() { getFileMtimeFunc = origMtimeFunc }()

	check := Check{
		ID:      "test-cascade",
		Trigger: "always",
		CascadeRules: []struct {
			Upstream    string `yaml:"upstream"`
			Downstreams []struct {
				Path     string   `yaml:"path"`
				Sections []string `yaml:"sections"`
				Required bool     `yaml:"required"`
			} `yaml:"downstreams"`
		}{
			{
				Upstream: "docs/prd.md",
				Downstreams: []struct {
					Path     string   `yaml:"path"`
					Sections []string `yaml:"sections"`
					Required bool     `yaml:"required"`
				}{
					{Path: "docs/architecture.md", Required: true},
				},
			},
		},
	}

	result, err := runChangeCascadeCheck(root, check)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Downstream doesn't exist, so no staleness detected
	if result.Status != "pass" {
		t.Errorf("expected pass (no downstream files), got %s", result.Status)
	}
}

func TestMatchPattern_InvalidPattern(t *testing.T) {
	// An invalid pattern should be skipped
	files := []string{"docs/prd.md"}
	result := matchPattern(files, "[invalid")
	if len(result) != 0 {
		t.Errorf("expected 0 matches for invalid pattern, got %d", len(result))
	}
}

func TestGlobFiles_InvalidPattern(t *testing.T) {
	root := t.TempDir()
	// Invalid patterns return an error
	_, err := globFiles(root, "[invalid")
	if err == nil {
		t.Error("expected error for invalid pattern")
	}
}

func TestGlobFiles_NoMatches(t *testing.T) {
	root := t.TempDir()
	// Valid pattern but no matches
	result, err := globFiles(root, "*.nonexistent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected 0 files, got %d", len(result))
	}
}

func TestChangeCascade_GitDiffTrigger_DownstreamGlobError(t *testing.T) {
	root := t.TempDir()

	// Create upstream but downstream glob fails (no directory)
	if err := os.MkdirAll(filepath.Join(root, "docs"), 0755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(root, "docs", "prd.md"), "# PRD")

	// Mock git diff
	origGitDiff := gitDiffFunc
	gitDiffFunc = func(r, baseline string) ([]string, error) {
		return []string{"docs/prd.md"}, nil
	}
	defer func() { gitDiffFunc = origGitDiff }()

	check := Check{
		ID:      "test-cascade",
		Trigger: "git-diff",
		CascadeRules: []struct {
			Upstream    string `yaml:"upstream"`
			Downstreams []struct {
				Path     string   `yaml:"path"`
				Sections []string `yaml:"sections"`
				Required bool     `yaml:"required"`
			} `yaml:"downstreams"`
		}{
			{
				Upstream: "docs/prd.md",
				Downstreams: []struct {
					Path     string   `yaml:"path"`
					Sections []string `yaml:"sections"`
					Required bool     `yaml:"required"`
				}{
					// This directory doesn't exist
					{Path: "nonexistent/*.md", Required: true},
				},
			},
		},
	}

	result, err := runChangeCascadeCheck(root, check)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// No downstream files found to be stale
	if result.Status != "pass" {
		t.Errorf("expected pass (no downstream files), got %s", result.Status)
	}
}

func TestGetFileMtime_FileNotExists(t *testing.T) {
	_, err := getFileMtime("/nonexistent/path/file.txt")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestGetFileMtime_ValidFile(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "test.txt")
	writeFile(&testing.T{}, path, "content")

	mtime, err := getFileMtime(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mtime.IsZero() {
		t.Error("expected non-zero mtime")
	}
}

func TestGitDiffFiles_Integration(t *testing.T) {
	// Skip if not in a git repo
	root := tempGitRepo(t)

	// Create and commit a file
	writeFile(t, filepath.Join(root, "initial.txt"), "initial content")
	gitAdd(t, root, "initial.txt")
	gitCommit(t, root, "initial commit")

	// Make a change
	writeFile(t, filepath.Join(root, "changed.txt"), "new content")
	gitAdd(t, root, "changed.txt")
	gitCommit(t, root, "second commit")

	files, err := gitDiffFiles(root, "HEAD~1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(files) != 1 || files[0] != "changed.txt" {
		t.Errorf("expected [changed.txt], got %v", files)
	}
}

func TestGitDiffFiles_NoChanges(t *testing.T) {
	root := tempGitRepo(t)

	// Create and commit a file
	writeFile(t, filepath.Join(root, "initial.txt"), "initial content")
	gitAdd(t, root, "initial.txt")
	gitCommit(t, root, "initial commit")

	// No changes since last commit
	files, err := gitDiffFiles(root, "HEAD")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(files) != 0 {
		t.Errorf("expected no changes, got %v", files)
	}
}

func gitAdd(t *testing.T, root, file string) {
	t.Helper()
	cmd := exec.Command("git", "add", file)
	cmd.Dir = root
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git add: %v (%s)", err, string(output))
	}
}

func gitCommit(t *testing.T, root, msg string) {
	t.Helper()
	cmd := exec.Command("git", "-c", "user.email=test@test.com", "-c", "user.name=Test", "commit", "-m", msg)
	cmd.Dir = root
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git commit: %v (%s)", err, string(output))
	}
}
