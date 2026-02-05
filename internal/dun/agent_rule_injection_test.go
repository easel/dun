package dun

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func runAgentRuleInjectionCheckFromSpec(root string, check Check) (CheckResult, error) {
	def := CheckDefinition{ID: check.ID, Description: check.Description}
	config := AgentRuleInjectionConfig{
		BasePrompt:   check.BasePrompt,
		InjectRules:  check.InjectRules,
		EnforceRules: check.EnforceRules,
	}
	return runAgentRuleInjectionCheck(root, Plugin{}, def, config)
}

func TestRunAgentRuleInjectionCheck_BasicPass(t *testing.T) {
	root := t.TempDir()

	// Create base prompt template
	promptDir := filepath.Join(root, "prompts")
	if err := os.MkdirAll(promptDir, 0755); err != nil {
		t.Fatalf("failed to create prompt dir: %v", err)
	}
	basePrompt := `# Feature Implementation

## Context
Implement the requested feature.

## Rules to Follow
{{RULES}}

## Instructions
Follow the coding standards.
`
	if err := os.WriteFile(filepath.Join(promptDir, "implement-feature.md"), []byte(basePrompt), 0644); err != nil {
		t.Fatalf("failed to write base prompt: %v", err)
	}

	// Create rule source file
	rulesDir := filepath.Join(root, ".dun", "rules")
	if err := os.MkdirAll(rulesDir, 0755); err != nil {
		t.Fatalf("failed to create rules dir: %v", err)
	}
	rulesContent := `- Use meaningful variable names
- Add error handling
- Write tests for all public functions
`
	if err := os.WriteFile(filepath.Join(rulesDir, "coding-standards.yaml"), []byte(rulesContent), 0644); err != nil {
		t.Fatalf("failed to write rules: %v", err)
	}

	check := Check{
		ID:          "test-rule-injection",
		Type:        "agent-rule-injection",
		Description: "Test rule injection",
		BasePrompt:  "prompts/implement-feature.md",
		InjectRules: []InjectRule{
			{Source: ".dun/rules/coding-standards.yaml", Section: "## Rules to Follow"},
		},
		EnforceRules: []EnforceRule{
			{ID: "must-reference-spec", Pattern: `Implements: FEAT-\d+`, Required: true},
		},
	}

	result, err := runAgentRuleInjectionCheckFromSpec(root, check)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Status != "pass" {
		t.Errorf("expected status 'pass', got %q", result.Status)
	}
	if result.Prompt == nil {
		t.Fatal("expected prompt envelope to be set")
	}
	if result.Prompt.Kind != "dun.agent-rule-injection.v1" {
		t.Errorf("expected kind 'dun.agent-rule-injection.v1', got %q", result.Prompt.Kind)
	}
	if !strings.Contains(result.Prompt.Prompt, "meaningful variable names") {
		t.Error("expected injected rules in prompt")
	}
}

func TestRunAgentRuleInjectionCheck_NoBasePrompt(t *testing.T) {
	root := t.TempDir()

	check := Check{
		ID:         "test-no-base-prompt",
		Type:       "agent-rule-injection",
		BasePrompt: "", // Missing base prompt
	}

	result, err := runAgentRuleInjectionCheckFromSpec(root, check)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Status != "fail" {
		t.Errorf("expected status 'fail', got %q", result.Status)
	}
	if !strings.Contains(result.Signal, "base_prompt is required") {
		t.Errorf("expected signal about missing base_prompt, got %q", result.Signal)
	}
}

func TestRunAgentRuleInjectionCheck_BasePromptNotFound(t *testing.T) {
	root := t.TempDir()

	check := Check{
		ID:         "test-missing-prompt",
		Type:       "agent-rule-injection",
		BasePrompt: "prompts/nonexistent.md",
	}

	result, err := runAgentRuleInjectionCheckFromSpec(root, check)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Status != "fail" {
		t.Errorf("expected status 'fail', got %q", result.Status)
	}
	if !strings.Contains(result.Signal, "failed to load base prompt") {
		t.Errorf("expected signal about loading failure, got %q", result.Signal)
	}
}

func TestRunAgentRuleInjectionCheck_InjectRuleSourceNotFound(t *testing.T) {
	root := t.TempDir()

	// Create base prompt
	promptDir := filepath.Join(root, "prompts")
	if err := os.MkdirAll(promptDir, 0755); err != nil {
		t.Fatalf("failed to create prompt dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(promptDir, "prompt.md"), []byte("# Prompt"), 0644); err != nil {
		t.Fatalf("failed to write base prompt: %v", err)
	}

	check := Check{
		ID:         "test-missing-source",
		Type:       "agent-rule-injection",
		BasePrompt: "prompts/prompt.md",
		InjectRules: []InjectRule{
			{Source: "nonexistent/rules.yaml", Section: "## Rules"},
		},
	}

	result, err := runAgentRuleInjectionCheckFromSpec(root, check)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should warn about missing source but still succeed
	if result.Status != "warn" {
		t.Errorf("expected status 'warn', got %q", result.Status)
	}
	if len(result.Issues) != 1 {
		t.Fatalf("expected 1 issue, got %d", len(result.Issues))
	}
	if !strings.Contains(result.Issues[0].ID, "inject-error") {
		t.Errorf("expected inject-error issue, got %q", result.Issues[0].ID)
	}
}

func TestRunAgentRuleInjectionCheck_SectionNotFound(t *testing.T) {
	root := t.TempDir()

	// Create base prompt without the expected section
	promptDir := filepath.Join(root, "prompts")
	if err := os.MkdirAll(promptDir, 0755); err != nil {
		t.Fatalf("failed to create prompt dir: %v", err)
	}
	basePrompt := `# Feature Implementation

## Context
Just context, no rules section.
`
	if err := os.WriteFile(filepath.Join(promptDir, "prompt.md"), []byte(basePrompt), 0644); err != nil {
		t.Fatalf("failed to write base prompt: %v", err)
	}

	// Create rule source
	rulesDir := filepath.Join(root, "rules")
	if err := os.MkdirAll(rulesDir, 0755); err != nil {
		t.Fatalf("failed to create rules dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(rulesDir, "coding.yaml"), []byte("- Rule 1\n- Rule 2"), 0644); err != nil {
		t.Fatalf("failed to write rules: %v", err)
	}

	check := Check{
		ID:         "test-section-not-found",
		Type:       "agent-rule-injection",
		BasePrompt: "prompts/prompt.md",
		InjectRules: []InjectRule{
			{Source: "rules/coding.yaml", Section: "## Rules to Follow"},
		},
	}

	result, err := runAgentRuleInjectionCheckFromSpec(root, check)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Status != "warn" {
		t.Errorf("expected status 'warn', got %q", result.Status)
	}
	if len(result.Issues) != 1 {
		t.Fatalf("expected 1 issue, got %d", len(result.Issues))
	}
	if !strings.Contains(result.Issues[0].ID, "section-not-found") {
		t.Errorf("expected section-not-found issue, got %q", result.Issues[0].ID)
	}
	// Content should still be appended
	if !strings.Contains(result.Prompt.Prompt, "Rule 1") {
		t.Error("expected appended rules in prompt")
	}
}

func TestRunAgentRuleInjectionCheck_NoSection(t *testing.T) {
	root := t.TempDir()

	// Create base prompt
	promptDir := filepath.Join(root, "prompts")
	if err := os.MkdirAll(promptDir, 0755); err != nil {
		t.Fatalf("failed to create prompt dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(promptDir, "prompt.md"), []byte("# Prompt\n\nContent here."), 0644); err != nil {
		t.Fatalf("failed to write base prompt: %v", err)
	}

	// Create rule source
	rulesDir := filepath.Join(root, "rules")
	if err := os.MkdirAll(rulesDir, 0755); err != nil {
		t.Fatalf("failed to create rules dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(rulesDir, "coding.yaml"), []byte("- Rule 1"), 0644); err != nil {
		t.Fatalf("failed to write rules: %v", err)
	}

	check := Check{
		ID:         "test-no-section",
		Type:       "agent-rule-injection",
		BasePrompt: "prompts/prompt.md",
		InjectRules: []InjectRule{
			{Source: "rules/coding.yaml", Section: ""}, // No section specified
		},
	}

	result, err := runAgentRuleInjectionCheckFromSpec(root, check)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Status != "pass" {
		t.Errorf("expected status 'pass', got %q", result.Status)
	}
	// Content should be appended at end
	if !strings.Contains(result.Prompt.Prompt, "Rule 1") {
		t.Error("expected appended rules in prompt")
	}
}

func TestRunAgentRuleInjectionCheck_FromRegistry(t *testing.T) {
	root := t.TempDir()

	// Create base prompt
	promptDir := filepath.Join(root, "prompts")
	if err := os.MkdirAll(promptDir, 0755); err != nil {
		t.Fatalf("failed to create prompt dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(promptDir, "prompt.md"), []byte("# Prompt\n\n## Governing Specs"), 0644); err != nil {
		t.Fatalf("failed to write base prompt: %v", err)
	}

	// Create spec registry
	dunDir := filepath.Join(root, ".dun")
	if err := os.MkdirAll(dunDir, 0755); err != nil {
		t.Fatalf("failed to create .dun dir: %v", err)
	}
	registryContent := `specs:
  - id: FEAT-001
    title: User Authentication
  - id: FEAT-002
    title: User Authorization
`
	if err := os.WriteFile(filepath.Join(dunDir, "spec-registry.yaml"), []byte(registryContent), 0644); err != nil {
		t.Fatalf("failed to write registry: %v", err)
	}

	check := Check{
		ID:         "test-from-registry",
		Type:       "agent-rule-injection",
		BasePrompt: "prompts/prompt.md",
		InjectRules: []InjectRule{
			{Source: "from_registry", Section: "## Governing Specs"},
		},
	}

	result, err := runAgentRuleInjectionCheckFromSpec(root, check)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Status != "pass" {
		t.Errorf("expected status 'pass', got %q", result.Status)
	}
	if !strings.Contains(result.Prompt.Prompt, "FEAT-001") {
		t.Error("expected registry content in prompt")
	}
}

func TestRunAgentRuleInjectionCheck_FromRegistryNotFound(t *testing.T) {
	root := t.TempDir()

	// Create base prompt but no registry
	promptDir := filepath.Join(root, "prompts")
	if err := os.MkdirAll(promptDir, 0755); err != nil {
		t.Fatalf("failed to create prompt dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(promptDir, "prompt.md"), []byte("# Prompt"), 0644); err != nil {
		t.Fatalf("failed to write base prompt: %v", err)
	}

	check := Check{
		ID:         "test-from-registry-not-found",
		Type:       "agent-rule-injection",
		BasePrompt: "prompts/prompt.md",
		InjectRules: []InjectRule{
			{Source: "from_registry", Section: "## Specs"},
		},
	}

	result, err := runAgentRuleInjectionCheckFromSpec(root, check)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Status != "warn" {
		t.Errorf("expected status 'warn', got %q", result.Status)
	}
	if len(result.Issues) != 1 {
		t.Fatalf("expected 1 issue, got %d", len(result.Issues))
	}
}

func TestRunAgentRuleInjectionCheck_MultipleRules(t *testing.T) {
	root := t.TempDir()

	// Create base prompt
	promptDir := filepath.Join(root, "prompts")
	if err := os.MkdirAll(promptDir, 0755); err != nil {
		t.Fatalf("failed to create prompt dir: %v", err)
	}
	basePrompt := `# Feature Implementation

## Coding Standards

## Security Guidelines

## Final Instructions
`
	if err := os.WriteFile(filepath.Join(promptDir, "prompt.md"), []byte(basePrompt), 0644); err != nil {
		t.Fatalf("failed to write base prompt: %v", err)
	}

	// Create rule sources
	rulesDir := filepath.Join(root, "rules")
	if err := os.MkdirAll(rulesDir, 0755); err != nil {
		t.Fatalf("failed to create rules dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(rulesDir, "coding.yaml"), []byte("CODING_RULE_1"), 0644); err != nil {
		t.Fatalf("failed to write coding rules: %v", err)
	}
	if err := os.WriteFile(filepath.Join(rulesDir, "security.yaml"), []byte("SECURITY_RULE_1"), 0644); err != nil {
		t.Fatalf("failed to write security rules: %v", err)
	}

	check := Check{
		ID:         "test-multiple-rules",
		Type:       "agent-rule-injection",
		BasePrompt: "prompts/prompt.md",
		InjectRules: []InjectRule{
			{Source: "rules/coding.yaml", Section: "## Coding Standards"},
			{Source: "rules/security.yaml", Section: "## Security Guidelines"},
		},
	}

	result, err := runAgentRuleInjectionCheckFromSpec(root, check)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Status != "pass" {
		t.Errorf("expected status 'pass', got %q", result.Status)
	}
	if !strings.Contains(result.Prompt.Prompt, "CODING_RULE_1") {
		t.Error("expected coding rules in prompt")
	}
	if !strings.Contains(result.Prompt.Prompt, "SECURITY_RULE_1") {
		t.Error("expected security rules in prompt")
	}
}

func TestRunAgentRuleInjectionCheck_EnforceRulesMetadata(t *testing.T) {
	root := t.TempDir()

	// Create base prompt
	promptDir := filepath.Join(root, "prompts")
	if err := os.MkdirAll(promptDir, 0755); err != nil {
		t.Fatalf("failed to create prompt dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(promptDir, "prompt.md"), []byte("# Prompt"), 0644); err != nil {
		t.Fatalf("failed to write base prompt: %v", err)
	}

	check := Check{
		ID:         "test-enforce-metadata",
		Type:       "agent-rule-injection",
		BasePrompt: "prompts/prompt.md",
		EnforceRules: []EnforceRule{
			{ID: "must-have-tests", Pattern: `_test\.go$`, Required: true},
			{ID: "spec-reference", Pattern: `FEAT-\d+`, Required: false},
		},
	}

	result, err := runAgentRuleInjectionCheckFromSpec(root, check)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify enforce rules are in callback metadata
	if result.Prompt == nil {
		t.Fatal("expected prompt envelope")
	}
	if !strings.Contains(result.Prompt.Callback.Command, "metadata") {
		t.Error("expected metadata in callback command")
	}
}

func TestValidateAgentResponse_AllPatternsMatch(t *testing.T) {
	response := `package main

// Implements: FEAT-001

func Handler() error {
	return nil
}
`
	enforceRules := []EnforceRule{
		{ID: "spec-reference", Pattern: `Implements: FEAT-\d+`, Required: true},
		{ID: "has-error-handling", Pattern: `return nil|return err`, Required: true},
	}

	issues, passed := ValidateAgentResponse(response, enforceRules)
	if !passed {
		t.Error("expected validation to pass")
	}
	if len(issues) != 0 {
		t.Errorf("expected no issues, got %d: %v", len(issues), issues)
	}
}

func TestValidateAgentResponse_RequiredPatternMissing(t *testing.T) {
	response := `package main

func Handler() {
	// No spec reference
}
`
	enforceRules := []EnforceRule{
		{ID: "spec-reference", Pattern: `Implements: FEAT-\d+`, Required: true},
	}

	issues, passed := ValidateAgentResponse(response, enforceRules)
	if passed {
		t.Error("expected validation to fail")
	}
	if len(issues) != 1 {
		t.Fatalf("expected 1 issue, got %d", len(issues))
	}
	if !strings.Contains(issues[0].ID, "missing-required") {
		t.Errorf("expected missing-required issue, got %q", issues[0].ID)
	}
}

func TestValidateAgentResponse_OptionalPatternMissing(t *testing.T) {
	response := `package main

func Handler() {
}
`
	enforceRules := []EnforceRule{
		{ID: "optional-docs", Pattern: `// Doc:`, Required: false},
	}

	issues, passed := ValidateAgentResponse(response, enforceRules)
	// Optional patterns don't cause failure
	if !passed {
		t.Error("expected validation to pass (optional pattern)")
	}
	if len(issues) != 0 {
		t.Errorf("expected no issues for optional pattern, got %d", len(issues))
	}
}

func TestValidateAgentResponse_InvalidPattern(t *testing.T) {
	response := "some response"
	enforceRules := []EnforceRule{
		{ID: "invalid", Pattern: `[invalid`, Required: true},
	}

	issues, passed := ValidateAgentResponse(response, enforceRules)
	if passed {
		t.Error("expected validation to fail for invalid pattern")
	}
	if len(issues) != 1 {
		t.Fatalf("expected 1 issue, got %d", len(issues))
	}
	if !strings.Contains(issues[0].ID, "invalid-pattern") {
		t.Errorf("expected invalid-pattern issue, got %q", issues[0].ID)
	}
}

func TestValidateAgentResponse_InvalidPatternOptional(t *testing.T) {
	response := "some response"
	enforceRules := []EnforceRule{
		{ID: "invalid-optional", Pattern: `[invalid`, Required: false},
	}

	issues, passed := ValidateAgentResponse(response, enforceRules)
	// Invalid optional pattern should not fail
	if !passed {
		t.Error("expected validation to pass for invalid optional pattern")
	}
	if len(issues) != 1 {
		t.Fatalf("expected 1 issue (error), got %d", len(issues))
	}
}

func TestValidateAgentResponse_EmptyRules(t *testing.T) {
	response := "some response"
	enforceRules := []EnforceRule{}

	issues, passed := ValidateAgentResponse(response, enforceRules)
	if !passed {
		t.Error("expected validation to pass with empty rules")
	}
	if len(issues) != 0 {
		t.Errorf("expected no issues, got %d", len(issues))
	}
}

func TestParseEnforceRulesMetadata_Valid(t *testing.T) {
	metadata := `{"enforce_rules":[{"id":"test","pattern":"test.*","required":true}]}`

	rules, err := ParseEnforceRulesMetadata(metadata)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(rules))
	}
	if rules[0].ID != "test" {
		t.Errorf("expected id 'test', got %q", rules[0].ID)
	}
	if rules[0].Pattern != "test.*" {
		t.Errorf("expected pattern 'test.*', got %q", rules[0].Pattern)
	}
	if !rules[0].Required {
		t.Error("expected required to be true")
	}
}

func TestParseEnforceRulesMetadata_Empty(t *testing.T) {
	rules, err := ParseEnforceRulesMetadata("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rules != nil {
		t.Errorf("expected nil rules for empty metadata, got %v", rules)
	}
}

func TestParseEnforceRulesMetadata_Invalid(t *testing.T) {
	_, err := ParseEnforceRulesMetadata("not valid json")
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestExtractRuleInjectionConfig(t *testing.T) {
	check := Check{
		BasePrompt: "prompts/test.md",
		InjectRules: []InjectRule{
			{Source: "rules/coding.yaml", Section: "## Rules"},
			{Source: "from_registry", Section: "## Specs"},
		},
		EnforceRules: []EnforceRule{
			{ID: "test-rule", Pattern: "test", Required: true},
		},
	}

	config := AgentRuleInjectionConfig{
		BasePrompt:   check.BasePrompt,
		InjectRules:  check.InjectRules,
		EnforceRules: check.EnforceRules,
	}

	if config.BasePrompt != "prompts/test.md" {
		t.Errorf("expected base_prompt 'prompts/test.md', got %q", config.BasePrompt)
	}
	if len(config.InjectRules) != 2 {
		t.Fatalf("expected 2 inject rules, got %d", len(config.InjectRules))
	}
	if config.InjectRules[0].Source != "rules/coding.yaml" {
		t.Errorf("expected source 'rules/coding.yaml', got %q", config.InjectRules[0].Source)
	}
	if len(config.EnforceRules) != 1 {
		t.Fatalf("expected 1 enforce rule, got %d", len(config.EnforceRules))
	}
	if config.EnforceRules[0].ID != "test-rule" {
		t.Errorf("expected enforce rule id 'test-rule', got %q", config.EnforceRules[0].ID)
	}
}

func TestInjectAtSection(t *testing.T) {
	tests := []struct {
		name           string
		prompt         string
		section        string
		content        string
		expectFound    bool
		expectContains string
	}{
		{
			name:           "section found",
			prompt:         "# Title\n\n## Rules\n\nExisting rules.\n\n## Next Section",
			section:        "## Rules",
			content:        "NEW RULE",
			expectFound:    true,
			expectContains: "## Rules\n\nNEW RULE",
		},
		{
			name:        "section not found",
			prompt:      "# Title\n\nNo rules section.",
			section:     "## Rules",
			content:     "NEW RULE",
			expectFound: false,
		},
		{
			name:           "section at end",
			prompt:         "# Title\n\n## Rules",
			section:        "## Rules",
			content:        "NEW RULE",
			expectFound:    true,
			expectContains: "NEW RULE",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, found := injectAtSection(tt.prompt, tt.section, tt.content)
			if found != tt.expectFound {
				t.Errorf("expected found=%v, got %v", tt.expectFound, found)
			}
			if tt.expectContains != "" && !strings.Contains(result, tt.expectContains) {
				t.Errorf("expected result to contain %q, got %q", tt.expectContains, result)
			}
		})
	}
}

func TestAppendSection(t *testing.T) {
	prompt := "# Existing Content"
	section := "## New Section"
	content := "New content here"

	result := appendSection(prompt, section, content)

	if !strings.Contains(result, "# Existing Content") {
		t.Error("expected original content preserved")
	}
	if !strings.Contains(result, "## New Section") {
		t.Error("expected new section added")
	}
	if !strings.Contains(result, "New content here") {
		t.Error("expected new content added")
	}
}

func TestBuildPromptSummary(t *testing.T) {
	tests := []struct {
		name           string
		config         AgentRuleInjectionConfig
		expectContains []string
	}{
		{
			name: "full config",
			config: AgentRuleInjectionConfig{
				InjectRules: []InjectRule{
					{Source: "a.yaml"},
					{Source: "b.yaml"},
				},
				EnforceRules: []EnforceRule{
					{ID: "a", Required: true},
					{ID: "b", Required: false},
				},
			},
			expectContains: []string{"2 rules injected", "2 enforce patterns", "1 required"},
		},
		{
			name: "inject only",
			config: AgentRuleInjectionConfig{
				InjectRules: []InjectRule{{Source: "a.yaml"}},
			},
			expectContains: []string{"1 rules injected"},
		},
		{
			name: "enforce only",
			config: AgentRuleInjectionConfig{
				EnforceRules: []EnforceRule{{ID: "a", Required: true}},
			},
			expectContains: []string{"1 enforce patterns"},
		},
		{
			name:           "empty config",
			config:         AgentRuleInjectionConfig{},
			expectContains: []string{"Enhanced prompt ready"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildPromptSummary(tt.config)
			for _, expected := range tt.expectContains {
				if !strings.Contains(result, expected) {
					t.Errorf("expected summary to contain %q, got %q", expected, result)
				}
			}
		})
	}
}

func TestLoadBasePrompt(t *testing.T) {
	root := t.TempDir()

	// Create test prompt
	promptDir := filepath.Join(root, "prompts")
	if err := os.MkdirAll(promptDir, 0755); err != nil {
		t.Fatalf("failed to create prompt dir: %v", err)
	}
	content := "# Test Prompt\n\nContent here."
	if err := os.WriteFile(filepath.Join(promptDir, "test.md"), []byte(content), 0644); err != nil {
		t.Fatalf("failed to write prompt: %v", err)
	}

	// Test loading existing file
	result, err := loadBasePrompt(root, "prompts/test.md")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != content {
		t.Errorf("expected %q, got %q", content, result)
	}

	// Test loading nonexistent file
	_, err = loadBasePrompt(root, "prompts/nonexistent.md")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestLoadRuleContent(t *testing.T) {
	root := t.TempDir()

	// Create rule file
	rulesDir := filepath.Join(root, "rules")
	if err := os.MkdirAll(rulesDir, 0755); err != nil {
		t.Fatalf("failed to create rules dir: %v", err)
	}
	content := "- Rule 1\n- Rule 2"
	if err := os.WriteFile(filepath.Join(rulesDir, "rules.yaml"), []byte(content), 0644); err != nil {
		t.Fatalf("failed to write rules: %v", err)
	}

	// Test loading file source
	result, err := loadRuleContent(root, "rules/rules.yaml")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != content {
		t.Errorf("expected %q, got %q", content, result)
	}

	// Test loading nonexistent file
	_, err = loadRuleContent(root, "rules/nonexistent.yaml")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestLoadFromRegistry(t *testing.T) {
	root := t.TempDir()

	// Test with no registry
	_, err := loadFromRegistry(root)
	if err == nil {
		t.Error("expected error when no registry exists")
	}

	// Create registry in .dun
	dunDir := filepath.Join(root, ".dun")
	if err := os.MkdirAll(dunDir, 0755); err != nil {
		t.Fatalf("failed to create .dun dir: %v", err)
	}
	content := "registry content"
	if err := os.WriteFile(filepath.Join(dunDir, "spec-registry.yaml"), []byte(content), 0644); err != nil {
		t.Fatalf("failed to write registry: %v", err)
	}

	result, err := loadFromRegistry(root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != content {
		t.Errorf("expected %q, got %q", content, result)
	}
}

func TestLoadFromRegistry_AlternateLocations(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{
			name:     ".dun/spec-registry.yaml",
			path:     ".dun/spec-registry.yaml",
			expected: "content1",
		},
		{
			name:     ".dun/rules/registry.yaml",
			path:     ".dun/rules/registry.yaml",
			expected: "content2",
		},
		{
			name:     "docs/specs/registry.yaml",
			path:     "docs/specs/registry.yaml",
			expected: "content3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()

			// Create directory structure and registry file
			fullPath := filepath.Join(root, tt.path)
			dir := filepath.Dir(fullPath)
			if err := os.MkdirAll(dir, 0755); err != nil {
				t.Fatalf("failed to create dir: %v", err)
			}
			if err := os.WriteFile(fullPath, []byte(tt.expected), 0644); err != nil {
				t.Fatalf("failed to write registry: %v", err)
			}

			result, err := loadFromRegistry(root)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestRunAgentRuleInjectionCheck_EnforceRulesJSONSerialization(t *testing.T) {
	root := t.TempDir()

	// Create base prompt
	promptDir := filepath.Join(root, "prompts")
	if err := os.MkdirAll(promptDir, 0755); err != nil {
		t.Fatalf("failed to create prompt dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(promptDir, "prompt.md"), []byte("# Prompt"), 0644); err != nil {
		t.Fatalf("failed to write base prompt: %v", err)
	}

	check := Check{
		ID:         "test-json",
		Type:       "agent-rule-injection",
		BasePrompt: "prompts/prompt.md",
		EnforceRules: []EnforceRule{
			{ID: "test-1", Pattern: `test\d+`, Required: true},
			{ID: "test-2", Pattern: `other`, Required: false},
		},
	}

	result, err := runAgentRuleInjectionCheckFromSpec(root, check)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Extract and parse metadata from callback
	cmd := result.Prompt.Callback.Command
	metadataStart := strings.Index(cmd, "--metadata '") + len("--metadata '")
	metadataEnd := strings.LastIndex(cmd, "'")
	if metadataStart == -1 || metadataEnd == -1 || metadataStart >= metadataEnd {
		t.Fatal("could not extract metadata from callback")
	}
	metadataJSON := cmd[metadataStart:metadataEnd]

	var metadata EnforceRulesMetadata
	if err := json.Unmarshal([]byte(metadataJSON), &metadata); err != nil {
		t.Fatalf("failed to parse metadata JSON: %v", err)
	}

	if len(metadata.EnforceRules) != 2 {
		t.Fatalf("expected 2 enforce rules in metadata, got %d", len(metadata.EnforceRules))
	}
	if metadata.EnforceRules[0].ID != "test-1" {
		t.Errorf("expected first rule id 'test-1', got %q", metadata.EnforceRules[0].ID)
	}
}

func TestRunAgentRuleInjectionCheck_NoInjectRulesNoEnforceRules(t *testing.T) {
	root := t.TempDir()

	// Create base prompt
	promptDir := filepath.Join(root, "prompts")
	if err := os.MkdirAll(promptDir, 0755); err != nil {
		t.Fatalf("failed to create prompt dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(promptDir, "prompt.md"), []byte("# Simple Prompt"), 0644); err != nil {
		t.Fatalf("failed to write base prompt: %v", err)
	}

	check := Check{
		ID:         "test-minimal",
		Type:       "agent-rule-injection",
		BasePrompt: "prompts/prompt.md",
		// No inject rules or enforce rules
	}

	result, err := runAgentRuleInjectionCheckFromSpec(root, check)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Status != "pass" {
		t.Errorf("expected status 'pass', got %q", result.Status)
	}
	if result.Prompt.Prompt != "# Simple Prompt" {
		t.Errorf("expected original prompt content, got %q", result.Prompt.Prompt)
	}
}

func TestBuildEnhancedPrompt_ErrorInLoadRuleContent(t *testing.T) {
	root := t.TempDir()

	basePrompt := "# Prompt"
	injectRules := []InjectRule{
		{Source: "nonexistent.yaml", Section: "## Rules"},
	}

	_, issues, err := buildEnhancedPrompt(root, basePrompt, injectRules)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(issues) != 1 {
		t.Fatalf("expected 1 issue, got %d", len(issues))
	}
	if !strings.Contains(issues[0].ID, "inject-error") {
		t.Errorf("expected inject-error issue, got %q", issues[0].ID)
	}
}

func TestValidateAgentResponse_MultipleRequiredMissing(t *testing.T) {
	response := "minimal response without any required patterns"
	enforceRules := []EnforceRule{
		{ID: "rule-1", Pattern: `PATTERN_1`, Required: true},
		{ID: "rule-2", Pattern: `PATTERN_2`, Required: true},
		{ID: "rule-3", Pattern: `PATTERN_3`, Required: true},
	}

	issues, passed := ValidateAgentResponse(response, enforceRules)
	if passed {
		t.Error("expected validation to fail")
	}
	if len(issues) != 3 {
		t.Errorf("expected 3 issues, got %d", len(issues))
	}
}

func TestRunAgentRuleInjectionCheck_PromptEnvelopeFields(t *testing.T) {
	root := t.TempDir()

	// Create base prompt
	promptDir := filepath.Join(root, "prompts")
	if err := os.MkdirAll(promptDir, 0755); err != nil {
		t.Fatalf("failed to create prompt dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(promptDir, "prompt.md"), []byte("# Prompt"), 0644); err != nil {
		t.Fatalf("failed to write base prompt: %v", err)
	}

	check := Check{
		ID:          "test-envelope",
		Type:        "agent-rule-injection",
		Description: "Test description",
		BasePrompt:  "prompts/prompt.md",
	}

	result, err := runAgentRuleInjectionCheckFromSpec(root, check)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Prompt.ID != "test-envelope" {
		t.Errorf("expected envelope ID 'test-envelope', got %q", result.Prompt.ID)
	}
	if result.Prompt.Title != "Test description" {
		t.Errorf("expected envelope Title 'Test description', got %q", result.Prompt.Title)
	}
	if result.Prompt.Callback.Stdin != true {
		t.Error("expected callback Stdin to be true")
	}
}
