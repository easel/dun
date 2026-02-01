package dun

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// AgentRuleInjectionConfig holds the configuration for agent-rule-injection checks.
type AgentRuleInjectionConfig struct {
	BasePrompt   string        `yaml:"base_prompt"`
	InjectRules  []InjectRule  `yaml:"inject_rules"`
	EnforceRules []EnforceRule `yaml:"enforce_rules"`
}

// InjectRule defines a rule source to inject into the prompt.
type InjectRule struct {
	Source  string `yaml:"source"`  // File path or "from_registry"
	Section string `yaml:"section"` // Where to inject in prompt (section header)
}

// EnforceRule defines a validation pattern to apply after agent response.
type EnforceRule struct {
	ID       string `yaml:"id"`
	Pattern  string `yaml:"pattern"`  // Regex to verify in output
	Required bool   `yaml:"required"` // Whether this pattern is mandatory
}

// EnforceRulesMetadata holds enforce rules in JSON format for embedding in prompt envelope.
type EnforceRulesMetadata struct {
	EnforceRules []EnforceRule `json:"enforce_rules"`
}

// runAgentRuleInjectionCheck builds an enhanced prompt with injected rules.
func runAgentRuleInjectionCheck(root string, check Check) (CheckResult, error) {
	// Extract configuration from check fields
	config := extractRuleInjectionConfig(check)

	// Validate configuration
	if config.BasePrompt == "" {
		return CheckResult{
			ID:     check.ID,
			Status: "fail",
			Signal: "base_prompt is required",
			Detail: "agent-rule-injection check requires a base_prompt to be specified",
		}, nil
	}

	// Load base prompt template
	basePromptContent, err := loadBasePrompt(root, config.BasePrompt)
	if err != nil {
		return CheckResult{
			ID:     check.ID,
			Status: "fail",
			Signal: "failed to load base prompt",
			Detail: err.Error(),
		}, nil
	}

	// Build enhanced prompt with injected rules
	enhancedPrompt, injectionIssues, err := buildEnhancedPrompt(root, basePromptContent, config.InjectRules)
	if err != nil {
		return CheckResult{
			ID:     check.ID,
			Status: "fail",
			Signal: "failed to build enhanced prompt",
			Detail: err.Error(),
		}, nil
	}

	// Create metadata JSON with enforce rules for later validation
	metadata := EnforceRulesMetadata{
		EnforceRules: config.EnforceRules,
	}
	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return CheckResult{
			ID:     check.ID,
			Status: "fail",
			Signal: "failed to serialize enforce rules",
			Detail: err.Error(),
		}, nil
	}

	// Determine status based on injection issues
	status := "pass"
	signal := "prompt enhanced with injected rules"
	if len(injectionIssues) > 0 {
		status = "warn"
		signal = fmt.Sprintf("prompt enhanced with %d warnings", len(injectionIssues))
	}

	// Build the prompt envelope with enhanced prompt
	envelope := PromptEnvelope{
		Kind:    "dun.agent-rule-injection.v1",
		ID:      check.ID,
		Title:   check.Description,
		Summary: buildPromptSummary(config),
		Prompt:  enhancedPrompt,
		Callback: PromptCallback{
			Command: fmt.Sprintf("dun respond --id %s --response - --metadata '%s'", check.ID, string(metadataJSON)),
			Stdin:   true,
		},
	}

	return CheckResult{
		ID:     check.ID,
		Status: status,
		Signal: signal,
		Detail: fmt.Sprintf("injected %d rules, %d enforce patterns", len(config.InjectRules), len(config.EnforceRules)),
		Prompt: &envelope,
		Issues: injectionIssues,
	}, nil
}

// extractRuleInjectionConfig extracts rule injection config from Check fields.
func extractRuleInjectionConfig(check Check) AgentRuleInjectionConfig {
	var injectRules []InjectRule
	for _, ir := range check.InjectRules {
		injectRules = append(injectRules, InjectRule{
			Source:  ir.Source,
			Section: ir.Section,
		})
	}

	var enforceRules []EnforceRule
	for _, er := range check.EnforceRules {
		enforceRules = append(enforceRules, EnforceRule{
			ID:       er.ID,
			Pattern:  er.Pattern,
			Required: er.Required,
		})
	}

	return AgentRuleInjectionConfig{
		BasePrompt:   check.BasePrompt,
		InjectRules:  injectRules,
		EnforceRules: enforceRules,
	}
}

// loadBasePrompt loads the base prompt template from the specified path.
func loadBasePrompt(root, promptPath string) (string, error) {
	fullPath := filepath.Join(root, promptPath)
	content, err := os.ReadFile(fullPath)
	if err != nil {
		return "", fmt.Errorf("loading base prompt %q: %w", promptPath, err)
	}
	return string(content), nil
}

// buildEnhancedPrompt builds the enhanced prompt by injecting rules at specified sections.
func buildEnhancedPrompt(root, basePrompt string, injectRules []InjectRule) (string, []Issue, error) {
	enhancedPrompt := basePrompt
	var issues []Issue

	for _, rule := range injectRules {
		content, err := loadRuleContent(root, rule.Source)
		if err != nil {
			issues = append(issues, Issue{
				ID:      fmt.Sprintf("inject-error:%s", rule.Source),
				Summary: fmt.Sprintf("failed to load rule source: %v", err),
				Path:    rule.Source,
			})
			continue
		}

		if rule.Section != "" {
			// Inject at specific section
			injected, found := injectAtSection(enhancedPrompt, rule.Section, content)
			if found {
				enhancedPrompt = injected
			} else {
				// Section not found, append at end
				issues = append(issues, Issue{
					ID:      fmt.Sprintf("section-not-found:%s", rule.Section),
					Summary: fmt.Sprintf("section %q not found in base prompt, appending content", rule.Section),
					Path:    rule.Source,
				})
				enhancedPrompt = appendSection(enhancedPrompt, rule.Section, content)
			}
		} else {
			// No section specified, append at end
			enhancedPrompt = enhancedPrompt + "\n\n" + content
		}
	}

	return enhancedPrompt, issues, nil
}

// loadRuleContent loads content from a rule source.
func loadRuleContent(root, source string) (string, error) {
	if source == "from_registry" {
		// Load from spec registry (placeholder for future implementation)
		return loadFromRegistry(root)
	}

	// Load from file
	fullPath := filepath.Join(root, source)
	content, err := os.ReadFile(fullPath)
	if err != nil {
		return "", fmt.Errorf("reading rule source %q: %w", source, err)
	}
	return string(content), nil
}

// loadFromRegistry loads rules from the spec registry.
// This is a placeholder that can be extended to query a real registry.
func loadFromRegistry(root string) (string, error) {
	// Try common registry locations
	registryPaths := []string{
		".dun/spec-registry.yaml",
		".dun/rules/registry.yaml",
		"docs/specs/registry.yaml",
	}

	for _, path := range registryPaths {
		fullPath := filepath.Join(root, path)
		content, err := os.ReadFile(fullPath)
		if err == nil {
			return string(content), nil
		}
	}

	return "", fmt.Errorf("no spec registry found")
}

// injectAtSection injects content after the specified section header.
func injectAtSection(prompt, section, content string) (string, bool) {
	// Find the section header
	idx := strings.Index(prompt, section)
	if idx == -1 {
		return prompt, false
	}

	// Find the end of the section header line
	endOfLine := strings.Index(prompt[idx:], "\n")
	if endOfLine == -1 {
		// Section header is at the end
		endOfLine = len(prompt) - idx
	}
	insertPos := idx + endOfLine

	// Insert the content after the section header
	injected := prompt[:insertPos] + "\n\n" + content + prompt[insertPos:]
	return injected, true
}

// appendSection appends a new section with content at the end of the prompt.
func appendSection(prompt, section, content string) string {
	return prompt + "\n\n" + section + "\n\n" + content
}

// buildPromptSummary creates a summary for the prompt envelope.
func buildPromptSummary(config AgentRuleInjectionConfig) string {
	var parts []string
	if len(config.InjectRules) > 0 {
		parts = append(parts, fmt.Sprintf("%d rules injected", len(config.InjectRules)))
	}
	if len(config.EnforceRules) > 0 {
		requiredCount := 0
		for _, er := range config.EnforceRules {
			if er.Required {
				requiredCount++
			}
		}
		parts = append(parts, fmt.Sprintf("%d enforce patterns (%d required)", len(config.EnforceRules), requiredCount))
	}
	if len(parts) == 0 {
		return "Enhanced prompt ready"
	}
	return "Enhanced prompt: " + strings.Join(parts, ", ")
}

// ValidateAgentResponse validates an agent's response against enforce rules.
// This function can be called by the response handler to verify compliance.
func ValidateAgentResponse(response string, enforceRules []EnforceRule) ([]Issue, bool) {
	var issues []Issue
	allPassed := true

	for _, rule := range enforceRules {
		re, err := regexp.Compile(rule.Pattern)
		if err != nil {
			issues = append(issues, Issue{
				ID:      fmt.Sprintf("invalid-pattern:%s", rule.ID),
				Summary: fmt.Sprintf("invalid regex pattern: %v", err),
			})
			if rule.Required {
				allPassed = false
			}
			continue
		}

		matched := re.MatchString(response)
		if rule.Required && !matched {
			issues = append(issues, Issue{
				ID:      fmt.Sprintf("missing-required:%s", rule.ID),
				Summary: fmt.Sprintf("required pattern %q not found in response", rule.Pattern),
			})
			allPassed = false
		}
	}

	return issues, allPassed
}

// ParseEnforceRulesMetadata parses enforce rules from metadata JSON.
func ParseEnforceRulesMetadata(metadataJSON string) ([]EnforceRule, error) {
	if metadataJSON == "" {
		return nil, nil
	}

	var metadata EnforceRulesMetadata
	if err := json.Unmarshal([]byte(metadataJSON), &metadata); err != nil {
		return nil, fmt.Errorf("parsing enforce rules metadata: %w", err)
	}
	return metadata.EnforceRules, nil
}
