package dun

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

func runSpecBindingCheckFromSpec(root string, check Check) (CheckResult, error) {
	def := CheckDefinition{ID: check.ID}
	config := SpecBindingConfig{Bindings: check.Bindings, BindingRules: check.BindingRules}
	return runSpecBindingCheck(root, def, config)
}

func TestRunSpecBindingCheck_BasicPass(t *testing.T) {
	root := t.TempDir()

	// Create spec file
	specDir := filepath.Join(root, "docs", "specs")
	if err := os.MkdirAll(specDir, 0755); err != nil {
		t.Fatalf("failed to create spec dir: %v", err)
	}
	specContent := `# FEAT-001 User Authentication

## Overview
This feature handles user authentication.

## Implementation
See ` + "`internal/auth/handler.go`" + `
`
	if err := os.WriteFile(filepath.Join(specDir, "FEAT-001.md"), []byte(specContent), 0644); err != nil {
		t.Fatalf("failed to write spec: %v", err)
	}

	// Create code file
	codeDir := filepath.Join(root, "internal", "auth")
	if err := os.MkdirAll(codeDir, 0755); err != nil {
		t.Fatalf("failed to create code dir: %v", err)
	}
	codeContent := `package auth

// Implements: FEAT-001

func HandleAuth() {
}
`
	if err := os.WriteFile(filepath.Join(codeDir, "handler.go"), []byte(codeContent), 0644); err != nil {
		t.Fatalf("failed to write code: %v", err)
	}

	check := Check{
		ID:   "test-spec-binding",
		Type: "spec-binding",
		Bindings: SpecBindings{
			Specs: []SpecBinding{
				{
					Pattern:               "docs/specs/FEAT-*.md",
					ImplementationSection: "## Implementation",
					IDPattern:             `FEAT-\d+`,
				},
			},
			Code: []CodeBinding{
				{
					Pattern:     "internal/**/*.go",
					SpecComment: "// Implements: FEAT-",
				},
			},
		},
		BindingRules: []BindingRule{
			{Type: "bidirectional-coverage", MinCoverage: 1.0},
		},
	}

	result, err := runSpecBindingCheckFromSpec(root, check)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != "pass" {
		t.Errorf("expected status 'pass', got %q", result.Status)
	}
	if result.Signal != "Spec coverage: 100%" {
		t.Errorf("unexpected signal: %q", result.Signal)
	}
}

func TestRunSpecBindingCheck_OrphanSpec(t *testing.T) {
	root := t.TempDir()

	// Create spec file without matching code
	specDir := filepath.Join(root, "docs", "specs")
	if err := os.MkdirAll(specDir, 0755); err != nil {
		t.Fatalf("failed to create spec dir: %v", err)
	}
	specContent := `# FEAT-002 User Logout

## Overview
Logout feature.

## Implementation
TBD
`
	if err := os.WriteFile(filepath.Join(specDir, "FEAT-002.md"), []byte(specContent), 0644); err != nil {
		t.Fatalf("failed to write spec: %v", err)
	}

	// Create empty code directory
	codeDir := filepath.Join(root, "internal")
	if err := os.MkdirAll(codeDir, 0755); err != nil {
		t.Fatalf("failed to create code dir: %v", err)
	}

	check := Check{
		ID:   "test-orphan-spec",
		Type: "spec-binding",
		Bindings: SpecBindings{
			Specs: []SpecBinding{
				{
					Pattern:   "docs/specs/FEAT-*.md",
					IDPattern: `FEAT-\d+`,
				},
			},
			Code: []CodeBinding{
				{
					Pattern:     "internal/**/*.go",
					SpecComment: "// Implements: FEAT-",
				},
			},
		},
		BindingRules: []BindingRule{
			{Type: "no-orphan-specs", WarnOnly: false},
		},
	}

	result, err := runSpecBindingCheckFromSpec(root, check)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != "fail" {
		t.Errorf("expected status 'fail', got %q", result.Status)
	}
	if len(result.Issues) != 1 {
		t.Fatalf("expected 1 issue, got %d", len(result.Issues))
	}
	if result.Issues[0].ID != "orphan-spec" {
		t.Errorf("expected issue type 'orphan-spec', got %q", result.Issues[0].ID)
	}
}

func TestRunSpecBindingCheck_OrphanCode(t *testing.T) {
	root := t.TempDir()

	// Create code file without spec reference
	codeDir := filepath.Join(root, "internal")
	if err := os.MkdirAll(codeDir, 0755); err != nil {
		t.Fatalf("failed to create code dir: %v", err)
	}
	codeContent := `package internal

func OrphanFunction() {
}
`
	if err := os.WriteFile(filepath.Join(codeDir, "orphan.go"), []byte(codeContent), 0644); err != nil {
		t.Fatalf("failed to write code: %v", err)
	}

	check := Check{
		ID:   "test-orphan-code",
		Type: "spec-binding",
		Bindings: SpecBindings{
			Specs: []SpecBinding{},
			Code: []CodeBinding{
				{
					Pattern:     "internal/*.go",
					SpecComment: "// Implements: FEAT-",
				},
			},
		},
		BindingRules: []BindingRule{
			{Type: "no-orphan-code", WarnOnly: false},
		},
	}

	result, err := runSpecBindingCheckFromSpec(root, check)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != "fail" {
		t.Errorf("expected status 'fail', got %q", result.Status)
	}
	if len(result.Issues) != 1 {
		t.Fatalf("expected 1 issue, got %d", len(result.Issues))
	}
	if result.Issues[0].ID != "orphan-code" {
		t.Errorf("expected issue type 'orphan-code', got %q", result.Issues[0].ID)
	}
}

func TestRunSpecBindingCheck_WarnOnly(t *testing.T) {
	root := t.TempDir()

	// Create orphan spec
	specDir := filepath.Join(root, "docs", "specs")
	if err := os.MkdirAll(specDir, 0755); err != nil {
		t.Fatalf("failed to create spec dir: %v", err)
	}
	specContent := `# FEAT-003 Feature`
	if err := os.WriteFile(filepath.Join(specDir, "FEAT-003.md"), []byte(specContent), 0644); err != nil {
		t.Fatalf("failed to write spec: %v", err)
	}

	check := Check{
		ID:   "test-warn-only",
		Type: "spec-binding",
		Bindings: SpecBindings{
			Specs: []SpecBinding{
				{
					Pattern:   "docs/specs/FEAT-*.md",
					IDPattern: `FEAT-\d+`,
				},
			},
			Code: []CodeBinding{},
		},
		BindingRules: []BindingRule{
			{Type: "no-orphan-specs", WarnOnly: true},
		},
	}

	result, err := runSpecBindingCheckFromSpec(root, check)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != "warn" {
		t.Errorf("expected status 'warn', got %q", result.Status)
	}
}

func TestRunSpecBindingCheck_CoverageBelowThreshold(t *testing.T) {
	root := t.TempDir()

	// Create two spec files
	specDir := filepath.Join(root, "docs", "specs")
	if err := os.MkdirAll(specDir, 0755); err != nil {
		t.Fatalf("failed to create spec dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(specDir, "FEAT-001.md"), []byte("# FEAT-001"), 0644); err != nil {
		t.Fatalf("failed to write spec: %v", err)
	}
	if err := os.WriteFile(filepath.Join(specDir, "FEAT-002.md"), []byte("# FEAT-002"), 0644); err != nil {
		t.Fatalf("failed to write spec: %v", err)
	}

	// Create code referencing only one spec
	codeDir := filepath.Join(root, "internal")
	if err := os.MkdirAll(codeDir, 0755); err != nil {
		t.Fatalf("failed to create code dir: %v", err)
	}
	codeContent := `package internal
// Implements: FEAT-001
`
	if err := os.WriteFile(filepath.Join(codeDir, "handler.go"), []byte(codeContent), 0644); err != nil {
		t.Fatalf("failed to write code: %v", err)
	}

	check := Check{
		ID:   "test-coverage",
		Type: "spec-binding",
		Bindings: SpecBindings{
			Specs: []SpecBinding{
				{
					Pattern:   "docs/specs/FEAT-*.md",
					IDPattern: `FEAT-\d+`,
				},
			},
			Code: []CodeBinding{
				{
					Pattern:     "internal/*.go",
					SpecComment: "// Implements: FEAT-",
				},
			},
		},
		BindingRules: []BindingRule{
			{Type: "bidirectional-coverage", MinCoverage: 0.9},
		},
	}

	result, err := runSpecBindingCheckFromSpec(root, check)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != "fail" {
		t.Errorf("expected status 'fail', got %q", result.Status)
	}
	if result.Signal != "Spec coverage: 50%" {
		t.Errorf("unexpected signal: %q", result.Signal)
	}
}

func TestRunSpecBindingCheck_NoSpecs(t *testing.T) {
	root := t.TempDir()

	check := Check{
		ID:   "test-no-specs",
		Type: "spec-binding",
		Bindings: SpecBindings{
			Specs: []SpecBinding{
				{
					Pattern:   "docs/specs/FEAT-*.md",
					IDPattern: `FEAT-\d+`,
				},
			},
			Code: []CodeBinding{},
		},
		BindingRules: []BindingRule{
			{Type: "bidirectional-coverage", MinCoverage: 0.8},
		},
	}

	result, err := runSpecBindingCheckFromSpec(root, check)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// With no specs, coverage calculation would be 0/0
	// This should pass since there are no specs to cover
	if result.Signal != "Spec coverage: 0%" {
		t.Errorf("unexpected signal: %q", result.Signal)
	}
}

func TestRunSpecBindingCheck_MultipleRules(t *testing.T) {
	root := t.TempDir()

	// Create orphan spec and orphan code
	specDir := filepath.Join(root, "docs", "specs")
	if err := os.MkdirAll(specDir, 0755); err != nil {
		t.Fatalf("failed to create spec dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(specDir, "FEAT-001.md"), []byte("# FEAT-001"), 0644); err != nil {
		t.Fatalf("failed to write spec: %v", err)
	}

	codeDir := filepath.Join(root, "internal")
	if err := os.MkdirAll(codeDir, 0755); err != nil {
		t.Fatalf("failed to create code dir: %v", err)
	}
	codeContent := `package internal
func NoSpec() {}
`
	if err := os.WriteFile(filepath.Join(codeDir, "orphan.go"), []byte(codeContent), 0644); err != nil {
		t.Fatalf("failed to write code: %v", err)
	}

	check := Check{
		ID:   "test-multiple-rules",
		Type: "spec-binding",
		Bindings: SpecBindings{
			Specs: []SpecBinding{
				{
					Pattern:   "docs/specs/FEAT-*.md",
					IDPattern: `FEAT-\d+`,
				},
			},
			Code: []CodeBinding{
				{
					Pattern:     "internal/*.go",
					SpecComment: "// Implements: FEAT-",
				},
			},
		},
		BindingRules: []BindingRule{
			{Type: "no-orphan-specs", WarnOnly: true},
			{Type: "no-orphan-code", WarnOnly: false},
		},
	}

	result, err := runSpecBindingCheckFromSpec(root, check)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should be "fail" because no-orphan-code is not warn_only
	if result.Status != "fail" {
		t.Errorf("expected status 'fail', got %q", result.Status)
	}
	// Should have 2 issues total
	if len(result.Issues) != 2 {
		t.Errorf("expected 2 issues, got %d", len(result.Issues))
	}
}

// Unit tests for helper functions

func TestFindFiles(t *testing.T) {
	root := t.TempDir()

	// Create test files
	dir := filepath.Join(root, "src")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("failed to create dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte(""), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "test.go"), []byte(""), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	files, err := findFiles(root, "src/*.go")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(files) != 2 {
		t.Errorf("expected 2 files, got %d", len(files))
	}
}

func TestFindFiles_NoMatches(t *testing.T) {
	root := t.TempDir()

	files, err := findFiles(root, "nonexistent/*.go")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(files) != 0 {
		t.Errorf("expected 0 files, got %d", len(files))
	}
}

func TestCompileIDPattern(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		wantNil bool
		wantErr bool
	}{
		{"empty", "", true, false},
		{"valid", `FEAT-\d+`, false, false},
		{"invalid", `[invalid`, false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			re, err := compileIDPattern(tt.pattern)
			if tt.wantErr && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if tt.wantNil && re != nil {
				t.Error("expected nil regex")
			}
			if !tt.wantNil && !tt.wantErr && re == nil {
				t.Error("expected non-nil regex")
			}
		})
	}
}

func TestExtractSpecIDs(t *testing.T) {
	tests := []struct {
		name     string
		filePath string
		content  string
		pattern  string
		expected []string
	}{
		{
			name:     "from filename",
			filePath: "/docs/FEAT-001.md",
			content:  "# Feature",
			pattern:  `FEAT-\d+`,
			expected: []string{"FEAT-001"},
		},
		{
			name:     "from content",
			filePath: "/docs/feature.md",
			content:  "# FEAT-001 and FEAT-002",
			pattern:  `FEAT-\d+`,
			expected: []string{"FEAT-001", "FEAT-002"},
		},
		{
			name:     "no pattern uses filename",
			filePath: "/docs/my-feature.md",
			content:  "content",
			pattern:  "",
			expected: []string{"my-feature"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var re *regexp.Regexp
			if tt.pattern != "" {
				re, _ = regexp.Compile(tt.pattern)
			}
			ids := extractSpecIDs(tt.filePath, tt.content, re)
			if len(ids) != len(tt.expected) {
				t.Errorf("expected %d IDs, got %d: %v", len(tt.expected), len(ids), ids)
			}
		})
	}
}

func TestExtractImplementationRefs(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		section  string
		expected int
	}{
		{
			name: "backtick paths",
			content: `# Feature
## Implementation
See ` + "`internal/handler.go`" + ` and ` + "`internal/service.go`" + `
`,
			section:  "## Implementation",
			expected: 2,
		},
		{
			name: "markdown links",
			content: `# Feature
## Implementation
[Handler](internal/handler.go)
`,
			section:  "## Implementation",
			expected: 1,
		},
		{
			name:     "no section",
			content:  "# Feature\nSome content",
			section:  "## Implementation",
			expected: 0,
		},
		{
			name:     "empty section header",
			content:  "# Feature",
			section:  "",
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			refs := extractImplementationRefs(tt.content, tt.section)
			if len(refs) != tt.expected {
				t.Errorf("expected %d refs, got %d: %v", tt.expected, len(refs), refs)
			}
		})
	}
}

func TestExtractSpecRefsFromCode(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		specComment string
		expected    int
	}{
		{
			name: "single ref",
			content: `package main
// Implements: FEAT-001
func Handler() {}
`,
			specComment: "// Implements: FEAT-",
			expected:    1,
		},
		{
			name: "multiple refs",
			content: `package main
// Implements: FEAT-001
// Implements: FEAT-002
func Handler() {}
`,
			specComment: "// Implements: FEAT-",
			expected:    2,
		},
		{
			name:        "no refs",
			content:     "package main\nfunc Handler() {}",
			specComment: "// Implements: FEAT-",
			expected:    0,
		},
		{
			name:        "empty comment pattern",
			content:     "// Implements: FEAT-001",
			specComment: "",
			expected:    0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			refs := extractSpecRefsFromCode(tt.content, tt.specComment)
			if len(refs) != tt.expected {
				t.Errorf("expected %d refs, got %d: %v", tt.expected, len(refs), refs)
			}
		})
	}
}

func TestUniqueStrings(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected int
	}{
		{"no duplicates", []string{"a", "b", "c"}, 3},
		{"with duplicates", []string{"a", "b", "a", "c", "b"}, 3},
		{"empty strings", []string{"a", "", "b", ""}, 2},
		{"all empty", []string{"", "", ""}, 0},
		{"nil", nil, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := uniqueStrings(tt.input)
			if len(result) != tt.expected {
				t.Errorf("expected %d strings, got %d: %v", tt.expected, len(result), result)
			}
		})
	}
}

func TestMergeUnique(t *testing.T) {
	a := []string{"a", "b"}
	b := []string{"b", "c"}
	result := mergeUnique(a, b)
	if len(result) != 3 {
		t.Errorf("expected 3 strings, got %d: %v", len(result), result)
	}
}

func TestLooksLikeCodePath(t *testing.T) {
	tests := []struct {
		path     string
		expected bool
	}{
		{"internal/handler.go", true},
		{"src/app.ts", true},
		{"main.py", true},
		{"docs/readme.md", false},
		{"config.yaml", false},
		{"handler", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := looksLikeCodePath(tt.path)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestBuildSpecsToCode(t *testing.T) {
	specMap := map[string]SpecInfo{
		"FEAT-001": {ID: "FEAT-001", Path: "/docs/FEAT-001.md"},
		"FEAT-002": {ID: "FEAT-002", Path: "/docs/FEAT-002.md"},
	}
	codeMap := map[string]CodeInfo{
		"/internal/handler.go": {Path: "/internal/handler.go", SpecIDs: []string{"FEAT-001"}},
		"/internal/service.go": {Path: "/internal/service.go", SpecIDs: []string{"FEAT-001", "FEAT-002"}},
	}

	result := buildSpecsToCode(specMap, codeMap)

	if len(result["FEAT-001"]) != 2 {
		t.Errorf("expected 2 files for FEAT-001, got %d", len(result["FEAT-001"]))
	}
	if len(result["FEAT-002"]) != 1 {
		t.Errorf("expected 1 file for FEAT-002, got %d", len(result["FEAT-002"]))
	}
}

func TestBuildCodeToSpecs(t *testing.T) {
	codeMap := map[string]CodeInfo{
		"/handler.go": {Path: "/handler.go", SpecIDs: []string{"FEAT-001", "FEAT-002"}},
	}

	result := buildCodeToSpecs(codeMap)

	if len(result["/handler.go"]) != 2 {
		t.Errorf("expected 2 spec IDs, got %d", len(result["/handler.go"]))
	}
}

func TestApplyBindingRule_Coverage(t *testing.T) {
	tests := []struct {
		name        string
		minCoverage float64
		coverage    float64
		warnOnly    bool
		wantStatus  string
	}{
		{"pass", 0.5, 0.8, false, "pass"},
		{"fail", 0.9, 0.5, false, "fail"},
		{"warn", 0.9, 0.5, true, "warn"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rule := BindingRule{
				Type:        "bidirectional-coverage",
				MinCoverage: tt.minCoverage,
				WarnOnly:    tt.warnOnly,
			}
			_, status := applyBindingRule(rule, nil, nil, nil, nil, tt.coverage)
			if status != tt.wantStatus {
				t.Errorf("expected status %q, got %q", tt.wantStatus, status)
			}
		})
	}
}

func TestApplyBindingRule_OrphanSpecs(t *testing.T) {
	specMap := map[string]SpecInfo{
		"FEAT-001": {ID: "FEAT-001", Path: "/docs/FEAT-001.md"},
	}
	specsToCode := map[string][]string{
		"FEAT-001": {}, // No code
	}

	rule := BindingRule{Type: "no-orphan-specs", WarnOnly: false}
	issues, status := applyBindingRule(rule, specMap, nil, specsToCode, nil, 0)

	if status != "fail" {
		t.Errorf("expected status 'fail', got %q", status)
	}
	if len(issues) != 1 {
		t.Errorf("expected 1 issue, got %d", len(issues))
	}
}

func TestApplyBindingRule_OrphanCode(t *testing.T) {
	codeToSpecs := map[string][]string{
		"/handler.go": {}, // No spec refs
	}

	rule := BindingRule{Type: "no-orphan-code", WarnOnly: false}
	issues, status := applyBindingRule(rule, nil, nil, nil, codeToSpecs, 0)

	if status != "fail" {
		t.Errorf("expected status 'fail', got %q", status)
	}
	if len(issues) != 1 {
		t.Errorf("expected 1 issue, got %d", len(issues))
	}
}

func TestExtractSpecs_InvalidIDPattern(t *testing.T) {
	root := t.TempDir()

	specPatterns := []SpecBinding{
		{Pattern: "*.md", IDPattern: "[invalid"},
	}

	_, err := extractSpecs(root, specPatterns)
	if err == nil {
		t.Error("expected error for invalid ID pattern")
	}
}

func TestExtractCodeRefs_MultiplePatterns(t *testing.T) {
	root := t.TempDir()

	// Create code files
	dir := filepath.Join(root, "src")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("failed to create dir: %v", err)
	}
	content := `// Implements: FEAT-001
// Spec: US-001
`
	if err := os.WriteFile(filepath.Join(dir, "handler.go"), []byte(content), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	codePatterns := []CodeBinding{
		{Pattern: "src/*.go", SpecComment: "// Implements: FEAT-"},
		{Pattern: "src/*.go", SpecComment: "// Spec: US-"},
	}

	result, err := extractCodeRefs(root, codePatterns)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	filePath := filepath.Join(dir, "handler.go")
	if info, ok := result[filePath]; !ok {
		t.Error("expected file in result")
	} else if len(info.SpecIDs) != 2 {
		t.Errorf("expected 2 merged spec IDs, got %d: %v", len(info.SpecIDs), info.SpecIDs)
	}
}

func TestRunSpecBindingCheck_InvalidSpecPattern(t *testing.T) {
	root := t.TempDir()

	check := Check{
		ID:   "test-invalid-pattern",
		Type: "spec-binding",
		Bindings: SpecBindings{
			Specs: []SpecBinding{
				{Pattern: "*.md", IDPattern: "[invalid-regex"},
			},
		},
	}

	result, err := runSpecBindingCheckFromSpec(root, check)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != "fail" {
		t.Errorf("expected status 'fail', got %q", result.Status)
	}
	if result.Signal != "failed to extract specs" {
		t.Errorf("expected signal about failed extraction, got %q", result.Signal)
	}
}

func TestExtractImplementationRefs_SectionEnd(t *testing.T) {
	content := `# Feature

## Implementation
See ` + "`handler.go`" + `

## Next Section
Other content with ` + "`other.go`" + `
`
	refs := extractImplementationRefs(content, "## Implementation")
	if len(refs) != 1 {
		t.Errorf("expected 1 ref (only from Implementation section), got %d: %v", len(refs), refs)
	}
	if len(refs) > 0 && refs[0] != "handler.go" {
		t.Errorf("expected 'handler.go', got %q", refs[0])
	}
}

func TestBuildSpecsToCode_UnknownSpec(t *testing.T) {
	specMap := map[string]SpecInfo{
		"FEAT-001": {ID: "FEAT-001", Path: "/docs/FEAT-001.md"},
	}
	codeMap := map[string]CodeInfo{
		"/handler.go": {Path: "/handler.go", SpecIDs: []string{"FEAT-999"}}, // Unknown spec
	}

	result := buildSpecsToCode(specMap, codeMap)

	// FEAT-001 should exist but be empty
	if len(result["FEAT-001"]) != 0 {
		t.Errorf("expected 0 files for FEAT-001, got %d", len(result["FEAT-001"]))
	}
	// FEAT-999 should not be in result since it's not in specMap
	if _, exists := result["FEAT-999"]; exists {
		t.Error("FEAT-999 should not be in result")
	}
}

func TestRunSpecBindingCheck_NoRules(t *testing.T) {
	root := t.TempDir()

	check := Check{
		ID:           "test-no-rules",
		Type:         "spec-binding",
		BindingRules: []BindingRule{},
	}

	result, err := runSpecBindingCheckFromSpec(root, check)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != "pass" {
		t.Errorf("expected status 'pass' with no rules, got %q", result.Status)
	}
}

func TestApplyBindingRule_UnknownType(t *testing.T) {
	rule := BindingRule{Type: "unknown-rule-type"}
	issues, status := applyBindingRule(rule, nil, nil, nil, nil, 0)

	if status != "pass" {
		t.Errorf("expected status 'pass' for unknown rule type, got %q", status)
	}
	if len(issues) != 0 {
		t.Errorf("expected no issues for unknown rule type, got %d", len(issues))
	}
}

func TestExtractSpecIDs_DuplicatesRemoved(t *testing.T) {
	re, _ := regexp.Compile(`FEAT-\d+`)
	content := "FEAT-001 FEAT-001 FEAT-001"
	ids := extractSpecIDs("/test.md", content, re)

	if len(ids) != 1 {
		t.Errorf("expected 1 unique ID, got %d: %v", len(ids), ids)
	}
}

func TestApplyBindingRule_NoOrphanSpecsPass(t *testing.T) {
	specMap := map[string]SpecInfo{
		"FEAT-001": {ID: "FEAT-001", Path: "/docs/FEAT-001.md"},
	}
	specsToCode := map[string][]string{
		"FEAT-001": {"/handler.go"}, // Has code
	}

	rule := BindingRule{Type: "no-orphan-specs", WarnOnly: false}
	issues, status := applyBindingRule(rule, specMap, nil, specsToCode, nil, 0)

	if status != "pass" {
		t.Errorf("expected status 'pass', got %q", status)
	}
	if len(issues) != 0 {
		t.Errorf("expected 0 issues, got %d", len(issues))
	}
}

func TestApplyBindingRule_NoOrphanCodePass(t *testing.T) {
	codeToSpecs := map[string][]string{
		"/handler.go": {"FEAT-001"}, // Has spec ref
	}

	rule := BindingRule{Type: "no-orphan-code", WarnOnly: false}
	issues, status := applyBindingRule(rule, nil, nil, nil, codeToSpecs, 0)

	if status != "pass" {
		t.Errorf("expected status 'pass', got %q", status)
	}
	if len(issues) != 0 {
		t.Errorf("expected 0 issues, got %d", len(issues))
	}
}

func TestExtractSpecs_UnreadableFile(t *testing.T) {
	root := t.TempDir()

	// Create a spec file
	specDir := filepath.Join(root, "docs")
	if err := os.MkdirAll(specDir, 0755); err != nil {
		t.Fatalf("failed to create spec dir: %v", err)
	}
	specPath := filepath.Join(specDir, "FEAT-001.md")
	if err := os.WriteFile(specPath, []byte("# FEAT-001"), 0644); err != nil {
		t.Fatalf("failed to write spec: %v", err)
	}

	// Make the file unreadable
	if err := os.Chmod(specPath, 0000); err != nil {
		t.Skipf("cannot change file permissions: %v", err)
	}
	defer os.Chmod(specPath, 0644)

	specPatterns := []SpecBinding{
		{Pattern: "docs/*.md", IDPattern: `FEAT-\d+`},
	}

	// Should not error, just skip the unreadable file
	result, err := extractSpecs(root, specPatterns)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected 0 specs (file unreadable), got %d", len(result))
	}
}

func TestExtractCodeRefs_UnreadableFile(t *testing.T) {
	root := t.TempDir()

	// Create a code file
	codeDir := filepath.Join(root, "src")
	if err := os.MkdirAll(codeDir, 0755); err != nil {
		t.Fatalf("failed to create code dir: %v", err)
	}
	codePath := filepath.Join(codeDir, "handler.go")
	if err := os.WriteFile(codePath, []byte("// Implements: FEAT-001"), 0644); err != nil {
		t.Fatalf("failed to write code: %v", err)
	}

	// Make the file unreadable
	if err := os.Chmod(codePath, 0000); err != nil {
		t.Skipf("cannot change file permissions: %v", err)
	}
	defer os.Chmod(codePath, 0644)

	codePatterns := []CodeBinding{
		{Pattern: "src/*.go", SpecComment: "// Implements: FEAT-"},
	}

	// Should not error, just skip the unreadable file
	result, err := extractCodeRefs(root, codePatterns)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected 0 code refs (file unreadable), got %d", len(result))
	}
}

func TestFindFiles_InvalidPattern(t *testing.T) {
	root := t.TempDir()

	// Invalid glob pattern
	files, err := findFiles(root, "[invalid")
	if err == nil {
		t.Error("expected error for invalid glob pattern")
	}
	if len(files) != 0 {
		t.Errorf("expected 0 files, got %d", len(files))
	}
}

func TestExtractSpecRefsFromCode_InvalidPattern(t *testing.T) {
	// Test with a spec_comment that would create an invalid regex
	content := "// Implements: FEAT-001"
	// This will create an invalid regex because [ is not escaped
	refs := extractSpecRefsFromCode(content, "[invalid")
	if len(refs) != 0 {
		t.Errorf("expected 0 refs for invalid pattern, got %d", len(refs))
	}
}

func TestExtractSpecIDs_NoPatternEmptyFilename(t *testing.T) {
	// Edge case: nil regex and filename with only extension
	ids := extractSpecIDs("/.md", "content", nil)
	if len(ids) != 0 {
		t.Errorf("expected 0 IDs for empty name, got %d: %v", len(ids), ids)
	}
}

func TestRunSpecBindingCheck_InvalidCodePattern(t *testing.T) {
	root := t.TempDir()

	// Create spec to avoid early exit
	specDir := filepath.Join(root, "docs")
	if err := os.MkdirAll(specDir, 0755); err != nil {
		t.Fatalf("failed to create dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(specDir, "FEAT-001.md"), []byte("# FEAT-001"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	check := Check{
		ID:   "test-invalid-code-pattern",
		Type: "spec-binding",
		Bindings: SpecBindings{
			Specs: []SpecBinding{
				{Pattern: "docs/*.md", IDPattern: `FEAT-\d+`},
			},
			Code: []CodeBinding{
				{Pattern: "[invalid-glob", SpecComment: "// Implements: FEAT-"},
			},
		},
	}

	result, err := runSpecBindingCheckFromSpec(root, check)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != "fail" {
		t.Errorf("expected status 'fail', got %q", result.Status)
	}
	if result.Signal != "failed to extract code refs" {
		t.Errorf("expected signal about failed code extraction, got %q", result.Signal)
	}
}

func TestApplyBindingRule_NoOrphanSpecsWarnOnly(t *testing.T) {
	specMap := map[string]SpecInfo{
		"FEAT-001": {ID: "FEAT-001", Path: "/docs/FEAT-001.md"},
	}
	specsToCode := map[string][]string{
		"FEAT-001": {}, // No code - orphan
	}

	rule := BindingRule{Type: "no-orphan-specs", WarnOnly: true}
	issues, status := applyBindingRule(rule, specMap, nil, specsToCode, nil, 0)

	if status != "warn" {
		t.Errorf("expected status 'warn', got %q", status)
	}
	if len(issues) != 1 {
		t.Errorf("expected 1 issue, got %d", len(issues))
	}
}

func TestApplyBindingRule_NoOrphanCodeWarnOnly(t *testing.T) {
	codeToSpecs := map[string][]string{
		"/handler.go": {}, // No spec ref - orphan
	}

	rule := BindingRule{Type: "no-orphan-code", WarnOnly: true}
	issues, status := applyBindingRule(rule, nil, nil, nil, codeToSpecs, 0)

	if status != "warn" {
		t.Errorf("expected status 'warn', got %q", status)
	}
	if len(issues) != 1 {
		t.Errorf("expected 1 issue, got %d", len(issues))
	}
}

func TestExtractSpecIDs_EmptyFilePath(t *testing.T) {
	re, _ := regexp.Compile(`FEAT-\d+`)
	ids := extractSpecIDs("", "FEAT-001 content", re)
	// Should find from content
	if len(ids) != 1 {
		t.Errorf("expected 1 ID from content, got %d: %v", len(ids), ids)
	}
}

func TestExtractSpecs_InvalidGlobPattern(t *testing.T) {
	root := t.TempDir()

	specPatterns := []SpecBinding{
		{Pattern: "[invalid-glob"},
	}

	_, err := extractSpecs(root, specPatterns)
	if err == nil {
		t.Error("expected error for invalid glob pattern")
	}
}

func TestExtractSpecRefsFromCode_MatchButEmptyID(t *testing.T) {
	// Test edge case where pattern matches but result is empty
	content := "// Implements: "
	refs := extractSpecRefsFromCode(content, "// Implements: ")
	// Pattern would match but the captured part would be empty or not match
	// This tests the "clean up the prefix" path
	if len(refs) != 0 {
		t.Errorf("expected 0 refs for empty ID, got %d: %v", len(refs), refs)
	}
}

func TestExtractSpecRefsFromCode_NoColonInComment(t *testing.T) {
	// Test spec_comment pattern without colon
	content := `// FEAT-001 is implemented here
// FEAT-002 also implemented`
	refs := extractSpecRefsFromCode(content, "// FEAT-")
	// This should still match, but with no prefix extraction
	if len(refs) != 2 {
		t.Errorf("expected 2 refs, got %d: %v", len(refs), refs)
	}
}
