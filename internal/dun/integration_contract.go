package dun

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// IntegrationContractConfig holds the configuration for an integration-contract check.
type IntegrationContractConfig struct {
	Contracts ContractsConfig `yaml:"contracts"`
	Rules     []ContractRule  `yaml:"rules"`
}

// ContractsConfig specifies where to find integration contracts.
type ContractsConfig struct {
	Map         string `yaml:"map"`         // Path to integration-map.yaml
	Definitions string `yaml:"definitions"` // Glob for interface definitions
}

// ContractRule specifies a rule to apply to integration contracts.
type ContractRule struct {
	Type string `yaml:"type"` // all-providers-implemented, all-consumers-satisfied, no-circular-dependencies
}

// IntegrationMap represents the structure of an integration-map.yaml file.
type IntegrationMap struct {
	Components map[string]Component `yaml:"components"`
}

// Component represents a component in the integration map.
type Component struct {
	Provides []Provider `yaml:"provides"`
	Consumes []Consumer `yaml:"consumes"`
}

// Provider represents an interface that a component provides.
type Provider struct {
	Name       string `yaml:"name"`
	Definition string `yaml:"definition"`
}

// Consumer represents an interface that a component consumes.
type Consumer struct {
	Name string `yaml:"name"`
	From string `yaml:"from"`
}

// runIntegrationContractCheck verifies components define and satisfy integration interfaces.
func runIntegrationContractCheck(root string, check Check) (CheckResult, error) {
	config := extractIntegrationContractConfig(check)

	// Load integration map
	integrationMap, err := loadIntegrationMap(root, config.Contracts.Map)
	if err != nil {
		return CheckResult{
			ID:     check.ID,
			Status: "fail",
			Signal: "failed to load integration map",
			Detail: err.Error(),
		}, nil
	}

	// Build component graph for analysis
	graph := buildComponentGraph(integrationMap)

	// Apply rules and collect issues
	var issues []Issue
	status := "pass"

	for _, rule := range config.Rules {
		ruleIssues, ruleStatus := applyContractRule(root, rule, integrationMap, graph, config.Contracts.Definitions)
		issues = append(issues, ruleIssues...)

		// Update overall status (fail > warn > pass)
		if ruleStatus == "fail" {
			status = "fail"
		} else if ruleStatus == "warn" && status != "fail" {
			status = "warn"
		}
	}

	// Build signal
	signal := "all contracts satisfied"
	if len(issues) > 0 {
		signal = fmt.Sprintf("%d contract issues found", len(issues))
	}

	return CheckResult{
		ID:     check.ID,
		Status: status,
		Signal: signal,
		Issues: issues,
	}, nil
}

// extractIntegrationContractConfig extracts contract config from check fields.
func extractIntegrationContractConfig(check Check) IntegrationContractConfig {
	var rules []ContractRule
	for _, r := range check.ContractRules {
		rules = append(rules, ContractRule{Type: r.Type})
	}

	return IntegrationContractConfig{
		Contracts: ContractsConfig{
			Map:         check.Contracts.Map,
			Definitions: check.Contracts.Definitions,
		},
		Rules: rules,
	}
}

// loadIntegrationMap loads and parses an integration-map.yaml file.
func loadIntegrationMap(root, mapPath string) (*IntegrationMap, error) {
	if mapPath == "" {
		return nil, fmt.Errorf("integration map path not specified")
	}

	fullPath := filepath.Join(root, mapPath)
	content, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, fmt.Errorf("reading integration map: %w", err)
	}

	var integrationMap IntegrationMap
	if err := yaml.Unmarshal(content, &integrationMap); err != nil {
		return nil, fmt.Errorf("parsing integration map: %w", err)
	}

	return &integrationMap, nil
}

// ComponentGraph represents the dependency graph of components.
type ComponentGraph struct {
	// providers maps interface name to the component that provides it
	providers map[string]string
	// consumers maps component name to list of interfaces it consumes
	consumers map[string][]string
	// dependencies maps component name to components it depends on
	dependencies map[string][]string
}

// buildComponentGraph constructs a graph from the integration map.
func buildComponentGraph(integrationMap *IntegrationMap) *ComponentGraph {
	graph := &ComponentGraph{
		providers:    make(map[string]string),
		consumers:    make(map[string][]string),
		dependencies: make(map[string][]string),
	}

	if integrationMap == nil {
		return graph
	}

	// Build providers map
	for componentName, component := range integrationMap.Components {
		for _, provider := range component.Provides {
			graph.providers[provider.Name] = componentName
		}
	}

	// Build consumers and dependencies maps
	for componentName, component := range integrationMap.Components {
		var consumedInterfaces []string
		var deps []string

		for _, consumer := range component.Consumes {
			consumedInterfaces = append(consumedInterfaces, consumer.Name)
			if consumer.From != "" {
				deps = append(deps, consumer.From)
			}
		}

		graph.consumers[componentName] = consumedInterfaces
		graph.dependencies[componentName] = deps
	}

	return graph
}

// applyContractRule applies a single contract rule and returns issues and status.
func applyContractRule(root string, rule ContractRule, integrationMap *IntegrationMap, graph *ComponentGraph, definitionsGlob string) ([]Issue, string) {
	switch rule.Type {
	case "all-providers-implemented":
		return checkProvidersImplemented(root, integrationMap, definitionsGlob)
	case "all-consumers-satisfied":
		return checkConsumersSatisfied(integrationMap, graph)
	case "no-circular-dependencies":
		return checkNoCircularDependencies(integrationMap, graph)
	default:
		return nil, "pass"
	}
}

// checkProvidersImplemented verifies all provider definitions exist.
func checkProvidersImplemented(root string, integrationMap *IntegrationMap, definitionsGlob string) ([]Issue, string) {
	var issues []Issue

	if integrationMap == nil {
		return issues, "pass"
	}

	for componentName, component := range integrationMap.Components {
		for _, provider := range component.Provides {
			if provider.Definition == "" {
				continue // No definition specified, skip
			}

			// Check if definition file exists
			defPath := filepath.Join(root, provider.Definition)
			if _, err := os.Stat(defPath); os.IsNotExist(err) {
				issues = append(issues, Issue{
					ID:      "missing-provider",
					Path:    provider.Definition,
					Summary: fmt.Sprintf("Provider %s in component %s: definition file not found", provider.Name, componentName),
				})
			}
		}
	}

	if len(issues) > 0 {
		return issues, "fail"
	}
	return issues, "pass"
}

// checkConsumersSatisfied verifies all consumed interfaces have providers.
func checkConsumersSatisfied(integrationMap *IntegrationMap, graph *ComponentGraph) ([]Issue, string) {
	var issues []Issue

	if integrationMap == nil {
		return issues, "pass"
	}

	for componentName, component := range integrationMap.Components {
		for _, consumer := range component.Consumes {
			// Check if there's a provider for this interface
			providerComponent, exists := graph.providers[consumer.Name]

			if !exists {
				issues = append(issues, Issue{
					ID:      "unsatisfied-consumer",
					Summary: fmt.Sprintf("Component %s consumes %s but no provider found", componentName, consumer.Name),
				})
				continue
			}

			// If 'from' is specified, verify it matches the actual provider
			if consumer.From != "" && consumer.From != providerComponent {
				issues = append(issues, Issue{
					ID:      "wrong-provider",
					Summary: fmt.Sprintf("Component %s expects %s from %s but %s provides it", componentName, consumer.Name, consumer.From, providerComponent),
				})
			}
		}
	}

	if len(issues) > 0 {
		return issues, "fail"
	}
	return issues, "pass"
}

// checkNoCircularDependencies detects cycles in the component dependency graph.
func checkNoCircularDependencies(integrationMap *IntegrationMap, graph *ComponentGraph) ([]Issue, string) {
	var issues []Issue

	if integrationMap == nil {
		return issues, "pass"
	}

	// Use DFS to detect cycles
	visited := make(map[string]bool)
	inStack := make(map[string]bool)

	for componentName := range integrationMap.Components {
		if cycle := detectCycle(componentName, graph, visited, inStack, nil); cycle != nil {
			// Format cycle path
			cyclePath := formatCyclePath(cycle)
			issues = append(issues, Issue{
				ID:      "circular-dependency",
				Summary: fmt.Sprintf("Circular dependency detected: %s", cyclePath),
			})
			// Only report first cycle found per component
			break
		}
	}

	if len(issues) > 0 {
		return issues, "fail"
	}
	return issues, "pass"
}

// detectCycle performs DFS to detect cycles, returning the cycle path if found.
func detectCycle(node string, graph *ComponentGraph, visited, inStack map[string]bool, path []string) []string {
	if inStack[node] {
		// Found cycle - return path from this node
		return append(path, node)
	}
	if visited[node] {
		return nil
	}

	visited[node] = true
	inStack[node] = true
	path = append(path, node)

	for _, dep := range graph.dependencies[node] {
		if cycle := detectCycle(dep, graph, visited, inStack, path); cycle != nil {
			return cycle
		}
	}

	inStack[node] = false
	return nil
}

// formatCyclePath formats a cycle path as a string.
func formatCyclePath(cycle []string) string {
	if len(cycle) == 0 {
		return ""
	}
	result := cycle[0]
	for i := 1; i < len(cycle); i++ {
		result += " -> " + cycle[i]
	}
	return result
}
