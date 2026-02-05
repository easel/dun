package dun

import (
	"os"
	"path/filepath"
	"testing"
)

func runIntegrationContractCheckFromSpec(root string, check Check) (CheckResult, error) {
	def := CheckDefinition{ID: check.ID}
	config := IntegrationContractConfig{Contracts: check.Contracts, ContractRules: check.ContractRules}
	return runIntegrationContractCheck(root, def, config)
}

func TestRunIntegrationContractCheck_BasicPass(t *testing.T) {
	root := t.TempDir()

	// Create integration map
	contractsDir := filepath.Join(root, "contracts")
	if err := os.MkdirAll(contractsDir, 0755); err != nil {
		t.Fatalf("failed to create contracts dir: %v", err)
	}

	mapContent := `components:
  auth-service:
    provides:
      - name: AuthProvider
        definition: contracts/auth.ts
    consumes: []
  user-service:
    provides:
      - name: UserRepository
        definition: contracts/user.ts
    consumes:
      - name: AuthProvider
        from: auth-service
`
	if err := os.WriteFile(filepath.Join(root, "contracts", "integration-map.yaml"), []byte(mapContent), 0644); err != nil {
		t.Fatalf("failed to write integration map: %v", err)
	}

	// Create definition files
	if err := os.WriteFile(filepath.Join(contractsDir, "auth.ts"), []byte("export interface AuthProvider {}"), 0644); err != nil {
		t.Fatalf("failed to write auth.ts: %v", err)
	}
	if err := os.WriteFile(filepath.Join(contractsDir, "user.ts"), []byte("export interface UserRepository {}"), 0644); err != nil {
		t.Fatalf("failed to write user.ts: %v", err)
	}

	check := Check{
		ID:   "test-integration-contract",
		Type: "integration-contract",
		Contracts: ContractsConfig{
			Map:         "contracts/integration-map.yaml",
			Definitions: "contracts/*.ts",
		},
		ContractRules: []ContractRule{
			{Type: "all-providers-implemented"},
			{Type: "all-consumers-satisfied"},
			{Type: "no-circular-dependencies"},
		},
	}

	result, err := runIntegrationContractCheckFromSpec(root, check)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != "pass" {
		t.Errorf("expected status 'pass', got %q", result.Status)
	}
	if result.Signal != "all contracts satisfied" {
		t.Errorf("unexpected signal: %q", result.Signal)
	}
}

func TestRunIntegrationContractCheck_MissingProvider(t *testing.T) {
	root := t.TempDir()

	// Create integration map
	contractsDir := filepath.Join(root, "contracts")
	if err := os.MkdirAll(contractsDir, 0755); err != nil {
		t.Fatalf("failed to create contracts dir: %v", err)
	}

	mapContent := `components:
  auth-service:
    provides:
      - name: AuthProvider
        definition: contracts/auth.ts
    consumes: []
`
	if err := os.WriteFile(filepath.Join(root, "contracts", "integration-map.yaml"), []byte(mapContent), 0644); err != nil {
		t.Fatalf("failed to write integration map: %v", err)
	}

	// Don't create auth.ts - provider definition missing

	check := Check{
		ID:   "test-missing-provider",
		Type: "integration-contract",
		Contracts: ContractsConfig{
			Map:         "contracts/integration-map.yaml",
			Definitions: "contracts/*.ts",
		},
		ContractRules: []ContractRule{
			{Type: "all-providers-implemented"},
		},
	}

	result, err := runIntegrationContractCheckFromSpec(root, check)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != "fail" {
		t.Errorf("expected status 'fail', got %q", result.Status)
	}
	if len(result.Issues) != 1 {
		t.Fatalf("expected 1 issue, got %d", len(result.Issues))
	}
	if result.Issues[0].ID != "missing-provider" {
		t.Errorf("expected issue type 'missing-provider', got %q", result.Issues[0].ID)
	}
}

func TestRunIntegrationContractCheck_UnsatisfiedConsumer(t *testing.T) {
	root := t.TempDir()

	// Create integration map
	contractsDir := filepath.Join(root, "contracts")
	if err := os.MkdirAll(contractsDir, 0755); err != nil {
		t.Fatalf("failed to create contracts dir: %v", err)
	}

	mapContent := `components:
  user-service:
    provides: []
    consumes:
      - name: NonExistentProvider
        from: nonexistent-service
`
	if err := os.WriteFile(filepath.Join(root, "contracts", "integration-map.yaml"), []byte(mapContent), 0644); err != nil {
		t.Fatalf("failed to write integration map: %v", err)
	}

	check := Check{
		ID:   "test-unsatisfied-consumer",
		Type: "integration-contract",
		Contracts: ContractsConfig{
			Map:         "contracts/integration-map.yaml",
			Definitions: "contracts/*.ts",
		},
		ContractRules: []ContractRule{
			{Type: "all-consumers-satisfied"},
		},
	}

	result, err := runIntegrationContractCheckFromSpec(root, check)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != "fail" {
		t.Errorf("expected status 'fail', got %q", result.Status)
	}
	if len(result.Issues) != 1 {
		t.Fatalf("expected 1 issue, got %d", len(result.Issues))
	}
	if result.Issues[0].ID != "unsatisfied-consumer" {
		t.Errorf("expected issue type 'unsatisfied-consumer', got %q", result.Issues[0].ID)
	}
}

func TestRunIntegrationContractCheck_CircularDependency(t *testing.T) {
	root := t.TempDir()

	// Create integration map with circular dependency
	contractsDir := filepath.Join(root, "contracts")
	if err := os.MkdirAll(contractsDir, 0755); err != nil {
		t.Fatalf("failed to create contracts dir: %v", err)
	}

	mapContent := `components:
  service-a:
    provides:
      - name: ProviderA
    consumes:
      - name: ProviderB
        from: service-b
  service-b:
    provides:
      - name: ProviderB
    consumes:
      - name: ProviderA
        from: service-a
`
	if err := os.WriteFile(filepath.Join(root, "contracts", "integration-map.yaml"), []byte(mapContent), 0644); err != nil {
		t.Fatalf("failed to write integration map: %v", err)
	}

	check := Check{
		ID:   "test-circular-dependency",
		Type: "integration-contract",
		Contracts: ContractsConfig{
			Map:         "contracts/integration-map.yaml",
			Definitions: "contracts/*.ts",
		},
		ContractRules: []ContractRule{
			{Type: "no-circular-dependencies"},
		},
	}

	result, err := runIntegrationContractCheckFromSpec(root, check)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != "fail" {
		t.Errorf("expected status 'fail', got %q", result.Status)
	}
	if len(result.Issues) != 1 {
		t.Fatalf("expected 1 issue, got %d", len(result.Issues))
	}
	if result.Issues[0].ID != "circular-dependency" {
		t.Errorf("expected issue type 'circular-dependency', got %q", result.Issues[0].ID)
	}
}

func TestRunIntegrationContractCheck_MissingMapPath(t *testing.T) {
	root := t.TempDir()

	check := Check{
		ID:   "test-missing-map",
		Type: "integration-contract",
		// No map path specified
	}

	result, err := runIntegrationContractCheckFromSpec(root, check)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != "fail" {
		t.Errorf("expected status 'fail', got %q", result.Status)
	}
	if result.Signal != "failed to load integration map" {
		t.Errorf("unexpected signal: %q", result.Signal)
	}
}

func TestRunIntegrationContractCheck_MapFileNotFound(t *testing.T) {
	root := t.TempDir()

	check := Check{
		ID:   "test-map-not-found",
		Type: "integration-contract",
		Contracts: ContractsConfig{
			Map: "nonexistent/integration-map.yaml",
		},
	}

	result, err := runIntegrationContractCheckFromSpec(root, check)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != "fail" {
		t.Errorf("expected status 'fail', got %q", result.Status)
	}
}

func TestRunIntegrationContractCheck_InvalidYAML(t *testing.T) {
	root := t.TempDir()

	// Create invalid YAML file
	contractsDir := filepath.Join(root, "contracts")
	if err := os.MkdirAll(contractsDir, 0755); err != nil {
		t.Fatalf("failed to create contracts dir: %v", err)
	}

	invalidYAML := `components:
  service-a:
    provides:
      - this is not valid YAML
      name: test
    invalid_indent`
	if err := os.WriteFile(filepath.Join(root, "contracts", "integration-map.yaml"), []byte(invalidYAML), 0644); err != nil {
		t.Fatalf("failed to write invalid map: %v", err)
	}

	check := Check{
		ID:   "test-invalid-yaml",
		Type: "integration-contract",
		Contracts: ContractsConfig{
			Map: "contracts/integration-map.yaml",
		},
	}

	result, err := runIntegrationContractCheckFromSpec(root, check)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != "fail" {
		t.Errorf("expected status 'fail', got %q", result.Status)
	}
}

func TestRunIntegrationContractCheck_NoRules(t *testing.T) {
	root := t.TempDir()

	// Create integration map
	contractsDir := filepath.Join(root, "contracts")
	if err := os.MkdirAll(contractsDir, 0755); err != nil {
		t.Fatalf("failed to create contracts dir: %v", err)
	}

	mapContent := `components:
  auth-service:
    provides: []
    consumes: []
`
	if err := os.WriteFile(filepath.Join(root, "contracts", "integration-map.yaml"), []byte(mapContent), 0644); err != nil {
		t.Fatalf("failed to write integration map: %v", err)
	}

	check := Check{
		ID:   "test-no-rules",
		Type: "integration-contract",
		Contracts: ContractsConfig{
			Map: "contracts/integration-map.yaml",
		},
		// No rules
	}

	result, err := runIntegrationContractCheckFromSpec(root, check)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != "pass" {
		t.Errorf("expected status 'pass' with no rules, got %q", result.Status)
	}
}

func TestRunIntegrationContractCheck_WrongProvider(t *testing.T) {
	root := t.TempDir()

	// Create integration map where consumer expects wrong provider
	contractsDir := filepath.Join(root, "contracts")
	if err := os.MkdirAll(contractsDir, 0755); err != nil {
		t.Fatalf("failed to create contracts dir: %v", err)
	}

	mapContent := `components:
  auth-service:
    provides:
      - name: AuthProvider
    consumes: []
  user-service:
    provides:
      - name: UserRepository
    consumes:
      - name: AuthProvider
        from: wrong-service
`
	if err := os.WriteFile(filepath.Join(root, "contracts", "integration-map.yaml"), []byte(mapContent), 0644); err != nil {
		t.Fatalf("failed to write integration map: %v", err)
	}

	check := Check{
		ID:   "test-wrong-provider",
		Type: "integration-contract",
		Contracts: ContractsConfig{
			Map: "contracts/integration-map.yaml",
		},
		ContractRules: []ContractRule{
			{Type: "all-consumers-satisfied"},
		},
	}

	result, err := runIntegrationContractCheckFromSpec(root, check)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != "fail" {
		t.Errorf("expected status 'fail', got %q", result.Status)
	}
	if len(result.Issues) != 1 {
		t.Fatalf("expected 1 issue, got %d", len(result.Issues))
	}
	if result.Issues[0].ID != "wrong-provider" {
		t.Errorf("expected issue type 'wrong-provider', got %q", result.Issues[0].ID)
	}
}

func TestRunIntegrationContractCheck_EmptyComponents(t *testing.T) {
	root := t.TempDir()

	// Create integration map with no components
	contractsDir := filepath.Join(root, "contracts")
	if err := os.MkdirAll(contractsDir, 0755); err != nil {
		t.Fatalf("failed to create contracts dir: %v", err)
	}

	mapContent := `components: {}`
	if err := os.WriteFile(filepath.Join(root, "contracts", "integration-map.yaml"), []byte(mapContent), 0644); err != nil {
		t.Fatalf("failed to write integration map: %v", err)
	}

	check := Check{
		ID:   "test-empty-components",
		Type: "integration-contract",
		Contracts: ContractsConfig{
			Map: "contracts/integration-map.yaml",
		},
		ContractRules: []ContractRule{
			{Type: "all-providers-implemented"},
			{Type: "all-consumers-satisfied"},
			{Type: "no-circular-dependencies"},
		},
	}

	result, err := runIntegrationContractCheckFromSpec(root, check)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != "pass" {
		t.Errorf("expected status 'pass', got %q", result.Status)
	}
}

func TestRunIntegrationContractCheck_ProviderNoDefinition(t *testing.T) {
	root := t.TempDir()

	// Create integration map with provider that has no definition
	contractsDir := filepath.Join(root, "contracts")
	if err := os.MkdirAll(contractsDir, 0755); err != nil {
		t.Fatalf("failed to create contracts dir: %v", err)
	}

	mapContent := `components:
  auth-service:
    provides:
      - name: AuthProvider
    consumes: []
`
	if err := os.WriteFile(filepath.Join(root, "contracts", "integration-map.yaml"), []byte(mapContent), 0644); err != nil {
		t.Fatalf("failed to write integration map: %v", err)
	}

	check := Check{
		ID:   "test-no-definition",
		Type: "integration-contract",
		Contracts: ContractsConfig{
			Map: "contracts/integration-map.yaml",
		},
		ContractRules: []ContractRule{
			{Type: "all-providers-implemented"},
		},
	}

	result, err := runIntegrationContractCheckFromSpec(root, check)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should pass since no definition means nothing to check
	if result.Status != "pass" {
		t.Errorf("expected status 'pass', got %q", result.Status)
	}
}

func TestRunIntegrationContractCheck_ConsumerNoFrom(t *testing.T) {
	root := t.TempDir()

	// Create integration map where consumer doesn't specify 'from'
	contractsDir := filepath.Join(root, "contracts")
	if err := os.MkdirAll(contractsDir, 0755); err != nil {
		t.Fatalf("failed to create contracts dir: %v", err)
	}

	mapContent := `components:
  auth-service:
    provides:
      - name: AuthProvider
    consumes: []
  user-service:
    provides: []
    consumes:
      - name: AuthProvider
`
	if err := os.WriteFile(filepath.Join(root, "contracts", "integration-map.yaml"), []byte(mapContent), 0644); err != nil {
		t.Fatalf("failed to write integration map: %v", err)
	}

	check := Check{
		ID:   "test-consumer-no-from",
		Type: "integration-contract",
		Contracts: ContractsConfig{
			Map: "contracts/integration-map.yaml",
		},
		ContractRules: []ContractRule{
			{Type: "all-consumers-satisfied"},
		},
	}

	result, err := runIntegrationContractCheckFromSpec(root, check)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should pass since AuthProvider is provided by auth-service
	if result.Status != "pass" {
		t.Errorf("expected status 'pass', got %q", result.Status)
	}
}

func TestRunIntegrationContractCheck_ThreeWayCircle(t *testing.T) {
	root := t.TempDir()

	// Create integration map with 3-way circular dependency
	contractsDir := filepath.Join(root, "contracts")
	if err := os.MkdirAll(contractsDir, 0755); err != nil {
		t.Fatalf("failed to create contracts dir: %v", err)
	}

	mapContent := `components:
  service-a:
    provides:
      - name: ProviderA
    consumes:
      - name: ProviderC
        from: service-c
  service-b:
    provides:
      - name: ProviderB
    consumes:
      - name: ProviderA
        from: service-a
  service-c:
    provides:
      - name: ProviderC
    consumes:
      - name: ProviderB
        from: service-b
`
	if err := os.WriteFile(filepath.Join(root, "contracts", "integration-map.yaml"), []byte(mapContent), 0644); err != nil {
		t.Fatalf("failed to write integration map: %v", err)
	}

	check := Check{
		ID:   "test-three-way-circle",
		Type: "integration-contract",
		Contracts: ContractsConfig{
			Map: "contracts/integration-map.yaml",
		},
		ContractRules: []ContractRule{
			{Type: "no-circular-dependencies"},
		},
	}

	result, err := runIntegrationContractCheckFromSpec(root, check)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != "fail" {
		t.Errorf("expected status 'fail', got %q", result.Status)
	}
	if len(result.Issues) != 1 {
		t.Fatalf("expected 1 issue, got %d", len(result.Issues))
	}
	if result.Issues[0].ID != "circular-dependency" {
		t.Errorf("expected issue type 'circular-dependency', got %q", result.Issues[0].ID)
	}
}

func TestRunIntegrationContractCheck_MultipleIssues(t *testing.T) {
	root := t.TempDir()

	// Create integration map with multiple issues
	contractsDir := filepath.Join(root, "contracts")
	if err := os.MkdirAll(contractsDir, 0755); err != nil {
		t.Fatalf("failed to create contracts dir: %v", err)
	}

	mapContent := `components:
  service-a:
    provides:
      - name: ProviderA
        definition: contracts/missing.ts
    consumes:
      - name: NonExistentProvider
`
	if err := os.WriteFile(filepath.Join(root, "contracts", "integration-map.yaml"), []byte(mapContent), 0644); err != nil {
		t.Fatalf("failed to write integration map: %v", err)
	}

	check := Check{
		ID:   "test-multiple-issues",
		Type: "integration-contract",
		Contracts: ContractsConfig{
			Map: "contracts/integration-map.yaml",
		},
		ContractRules: []ContractRule{
			{Type: "all-providers-implemented"},
			{Type: "all-consumers-satisfied"},
		},
	}

	result, err := runIntegrationContractCheckFromSpec(root, check)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != "fail" {
		t.Errorf("expected status 'fail', got %q", result.Status)
	}
	if len(result.Issues) != 2 {
		t.Errorf("expected 2 issues, got %d", len(result.Issues))
	}
}

func TestRunIntegrationContractCheck_UnknownRuleType(t *testing.T) {
	root := t.TempDir()

	// Create minimal integration map
	contractsDir := filepath.Join(root, "contracts")
	if err := os.MkdirAll(contractsDir, 0755); err != nil {
		t.Fatalf("failed to create contracts dir: %v", err)
	}

	mapContent := `components: {}`
	if err := os.WriteFile(filepath.Join(root, "contracts", "integration-map.yaml"), []byte(mapContent), 0644); err != nil {
		t.Fatalf("failed to write integration map: %v", err)
	}

	check := Check{
		ID:   "test-unknown-rule",
		Type: "integration-contract",
		Contracts: ContractsConfig{
			Map: "contracts/integration-map.yaml",
		},
		ContractRules: []ContractRule{
			{Type: "unknown-rule-type"},
		},
	}

	result, err := runIntegrationContractCheckFromSpec(root, check)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Unknown rules should be ignored, resulting in pass
	if result.Status != "pass" {
		t.Errorf("expected status 'pass' for unknown rule, got %q", result.Status)
	}
}

// Unit tests for helper functions

func TestLoadIntegrationMap(t *testing.T) {
	root := t.TempDir()

	contractsDir := filepath.Join(root, "contracts")
	if err := os.MkdirAll(contractsDir, 0755); err != nil {
		t.Fatalf("failed to create contracts dir: %v", err)
	}

	mapContent := `components:
  test-service:
    provides:
      - name: TestProvider
        definition: test.ts
    consumes:
      - name: OtherProvider
        from: other-service
`
	if err := os.WriteFile(filepath.Join(root, "contracts", "map.yaml"), []byte(mapContent), 0644); err != nil {
		t.Fatalf("failed to write map: %v", err)
	}

	integrationMap, err := loadIntegrationMap(root, "contracts/map.yaml")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if integrationMap == nil {
		t.Fatal("expected non-nil integration map")
	}
	if len(integrationMap.Components) != 1 {
		t.Errorf("expected 1 component, got %d", len(integrationMap.Components))
	}

	component, ok := integrationMap.Components["test-service"]
	if !ok {
		t.Fatal("expected test-service component")
	}
	if len(component.Provides) != 1 {
		t.Errorf("expected 1 provider, got %d", len(component.Provides))
	}
	if len(component.Consumes) != 1 {
		t.Errorf("expected 1 consumer, got %d", len(component.Consumes))
	}
	if component.Provides[0].Name != "TestProvider" {
		t.Errorf("expected provider name 'TestProvider', got %q", component.Provides[0].Name)
	}
	if component.Consumes[0].Name != "OtherProvider" {
		t.Errorf("expected consumer name 'OtherProvider', got %q", component.Consumes[0].Name)
	}
	if component.Consumes[0].From != "other-service" {
		t.Errorf("expected consumer from 'other-service', got %q", component.Consumes[0].From)
	}
}

func TestLoadIntegrationMap_EmptyPath(t *testing.T) {
	root := t.TempDir()

	_, err := loadIntegrationMap(root, "")
	if err == nil {
		t.Error("expected error for empty map path")
	}
}

func TestLoadIntegrationMap_FileNotFound(t *testing.T) {
	root := t.TempDir()

	_, err := loadIntegrationMap(root, "nonexistent.yaml")
	if err == nil {
		t.Error("expected error for file not found")
	}
}

func TestBuildComponentGraph(t *testing.T) {
	integrationMap := &IntegrationMap{
		Components: map[string]Component{
			"service-a": {
				Provides: []Provider{
					{Name: "ProviderA", Definition: "a.ts"},
				},
				Consumes: []Consumer{
					{Name: "ProviderB", From: "service-b"},
				},
			},
			"service-b": {
				Provides: []Provider{
					{Name: "ProviderB", Definition: "b.ts"},
				},
				Consumes: []Consumer{},
			},
		},
	}

	graph := buildComponentGraph(integrationMap)

	// Check providers
	if graph.providers["ProviderA"] != "service-a" {
		t.Errorf("expected ProviderA from service-a, got %q", graph.providers["ProviderA"])
	}
	if graph.providers["ProviderB"] != "service-b" {
		t.Errorf("expected ProviderB from service-b, got %q", graph.providers["ProviderB"])
	}

	// Check consumers
	if len(graph.consumers["service-a"]) != 1 {
		t.Errorf("expected service-a to consume 1 interface, got %d", len(graph.consumers["service-a"]))
	}
	if len(graph.consumers["service-b"]) != 0 {
		t.Errorf("expected service-b to consume 0 interfaces, got %d", len(graph.consumers["service-b"]))
	}

	// Check dependencies
	if len(graph.dependencies["service-a"]) != 1 {
		t.Errorf("expected service-a to have 1 dependency, got %d", len(graph.dependencies["service-a"]))
	}
	if graph.dependencies["service-a"][0] != "service-b" {
		t.Errorf("expected service-a to depend on service-b")
	}
}

func TestBuildComponentGraph_NilMap(t *testing.T) {
	graph := buildComponentGraph(nil)
	if graph == nil {
		t.Fatal("expected non-nil graph")
	}
	if len(graph.providers) != 0 {
		t.Error("expected empty providers")
	}
	if len(graph.consumers) != 0 {
		t.Error("expected empty consumers")
	}
	if len(graph.dependencies) != 0 {
		t.Error("expected empty dependencies")
	}
}

func TestDetectCycle_NoCycle(t *testing.T) {
	graph := &ComponentGraph{
		providers:    make(map[string]string),
		consumers:    make(map[string][]string),
		dependencies: map[string][]string{"a": {"b"}, "b": {"c"}, "c": {}},
	}

	visited := make(map[string]bool)
	inStack := make(map[string]bool)

	cycle := detectCycle("a", graph, visited, inStack, nil)
	if cycle != nil {
		t.Errorf("expected no cycle, got %v", cycle)
	}
}

func TestDetectCycle_WithCycle(t *testing.T) {
	graph := &ComponentGraph{
		providers:    make(map[string]string),
		consumers:    make(map[string][]string),
		dependencies: map[string][]string{"a": {"b"}, "b": {"c"}, "c": {"a"}},
	}

	visited := make(map[string]bool)
	inStack := make(map[string]bool)

	cycle := detectCycle("a", graph, visited, inStack, nil)
	if cycle == nil {
		t.Error("expected cycle to be detected")
	}
}

func TestDetectCycle_SelfLoop(t *testing.T) {
	graph := &ComponentGraph{
		providers:    make(map[string]string),
		consumers:    make(map[string][]string),
		dependencies: map[string][]string{"a": {"a"}},
	}

	visited := make(map[string]bool)
	inStack := make(map[string]bool)

	cycle := detectCycle("a", graph, visited, inStack, nil)
	if cycle == nil {
		t.Error("expected self-loop to be detected")
	}
}

func TestFormatCyclePath(t *testing.T) {
	tests := []struct {
		name     string
		cycle    []string
		expected string
	}{
		{"empty", nil, ""},
		{"single", []string{"a"}, "a"},
		{"two", []string{"a", "b"}, "a -> b"},
		{"three", []string{"a", "b", "c"}, "a -> b -> c"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatCyclePath(tt.cycle)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestExtractIntegrationContractConfig(t *testing.T) {
	check := Check{
		Contracts: ContractsConfig{
			Map:         "contracts/map.yaml",
			Definitions: "contracts/*.ts",
		},
		ContractRules: []ContractRule{
			{Type: "all-providers-implemented"},
			{Type: "all-consumers-satisfied"},
		},
	}

	config := IntegrationContractConfig{Contracts: check.Contracts, ContractRules: check.ContractRules}

	if config.Contracts.Map != "contracts/map.yaml" {
		t.Errorf("expected map 'contracts/map.yaml', got %q", config.Contracts.Map)
	}
	if config.Contracts.Definitions != "contracts/*.ts" {
		t.Errorf("expected definitions 'contracts/*.ts', got %q", config.Contracts.Definitions)
	}
	if len(config.ContractRules) != 2 {
		t.Errorf("expected 2 rules, got %d", len(config.ContractRules))
	}
}

func TestCheckProvidersImplemented_NilMap(t *testing.T) {
	issues, status := checkProvidersImplemented("/tmp", nil, "")
	if status != "pass" {
		t.Errorf("expected pass for nil map, got %q", status)
	}
	if len(issues) != 0 {
		t.Errorf("expected no issues, got %d", len(issues))
	}
}

func TestCheckConsumersSatisfied_NilMap(t *testing.T) {
	issues, status := checkConsumersSatisfied(nil, nil)
	if status != "pass" {
		t.Errorf("expected pass for nil map, got %q", status)
	}
	if len(issues) != 0 {
		t.Errorf("expected no issues, got %d", len(issues))
	}
}

func TestCheckNoCircularDependencies_NilMap(t *testing.T) {
	issues, status := checkNoCircularDependencies(nil, nil)
	if status != "pass" {
		t.Errorf("expected pass for nil map, got %q", status)
	}
	if len(issues) != 0 {
		t.Errorf("expected no issues, got %d", len(issues))
	}
}

func TestApplyContractRule_AllTypes(t *testing.T) {
	root := t.TempDir()
	integrationMap := &IntegrationMap{Components: map[string]Component{}}
	graph := buildComponentGraph(integrationMap)

	tests := []struct {
		ruleType string
	}{
		{"all-providers-implemented"},
		{"all-consumers-satisfied"},
		{"no-circular-dependencies"},
		{"unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.ruleType, func(t *testing.T) {
			rule := ContractRule{Type: tt.ruleType}
			_, status := applyContractRule(root, rule, integrationMap, graph, "")
			if status != "pass" {
				t.Errorf("expected pass for empty map, got %q", status)
			}
		})
	}
}

func TestRunIntegrationContractCheck_NullConsumes(t *testing.T) {
	root := t.TempDir()

	// Create integration map with null consumes
	contractsDir := filepath.Join(root, "contracts")
	if err := os.MkdirAll(contractsDir, 0755); err != nil {
		t.Fatalf("failed to create contracts dir: %v", err)
	}

	mapContent := `components:
  service-a:
    provides:
      - name: ProviderA
    consumes: null
`
	if err := os.WriteFile(filepath.Join(root, "contracts", "integration-map.yaml"), []byte(mapContent), 0644); err != nil {
		t.Fatalf("failed to write integration map: %v", err)
	}

	check := Check{
		ID:   "test-null-consumes",
		Type: "integration-contract",
		Contracts: ContractsConfig{
			Map: "contracts/integration-map.yaml",
		},
		ContractRules: []ContractRule{
			{Type: "all-consumers-satisfied"},
		},
	}

	result, err := runIntegrationContractCheckFromSpec(root, check)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != "pass" {
		t.Errorf("expected status 'pass', got %q", result.Status)
	}
}

func TestRunIntegrationContractCheck_MultipleProviders(t *testing.T) {
	root := t.TempDir()

	// Create integration map with component providing multiple interfaces
	contractsDir := filepath.Join(root, "contracts")
	if err := os.MkdirAll(contractsDir, 0755); err != nil {
		t.Fatalf("failed to create contracts dir: %v", err)
	}

	mapContent := `components:
  multi-provider:
    provides:
      - name: InterfaceA
        definition: contracts/a.ts
      - name: InterfaceB
        definition: contracts/b.ts
    consumes: []
`
	if err := os.WriteFile(filepath.Join(root, "contracts", "integration-map.yaml"), []byte(mapContent), 0644); err != nil {
		t.Fatalf("failed to write integration map: %v", err)
	}

	// Only create one definition file
	if err := os.WriteFile(filepath.Join(contractsDir, "a.ts"), []byte("interface A {}"), 0644); err != nil {
		t.Fatalf("failed to write a.ts: %v", err)
	}

	check := Check{
		ID:   "test-multiple-providers",
		Type: "integration-contract",
		Contracts: ContractsConfig{
			Map: "contracts/integration-map.yaml",
		},
		ContractRules: []ContractRule{
			{Type: "all-providers-implemented"},
		},
	}

	result, err := runIntegrationContractCheckFromSpec(root, check)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != "fail" {
		t.Errorf("expected status 'fail', got %q", result.Status)
	}
	if len(result.Issues) != 1 {
		t.Errorf("expected 1 issue (missing b.ts), got %d", len(result.Issues))
	}
}

func TestRunIntegrationContractCheck_DiamondDependency(t *testing.T) {
	root := t.TempDir()

	// Create diamond dependency (A depends on B and C, B and C both depend on D)
	// This is NOT circular
	contractsDir := filepath.Join(root, "contracts")
	if err := os.MkdirAll(contractsDir, 0755); err != nil {
		t.Fatalf("failed to create contracts dir: %v", err)
	}

	mapContent := `components:
  service-a:
    provides:
      - name: ProviderA
    consumes:
      - name: ProviderB
        from: service-b
      - name: ProviderC
        from: service-c
  service-b:
    provides:
      - name: ProviderB
    consumes:
      - name: ProviderD
        from: service-d
  service-c:
    provides:
      - name: ProviderC
    consumes:
      - name: ProviderD
        from: service-d
  service-d:
    provides:
      - name: ProviderD
    consumes: []
`
	if err := os.WriteFile(filepath.Join(root, "contracts", "integration-map.yaml"), []byte(mapContent), 0644); err != nil {
		t.Fatalf("failed to write integration map: %v", err)
	}

	check := Check{
		ID:   "test-diamond",
		Type: "integration-contract",
		Contracts: ContractsConfig{
			Map: "contracts/integration-map.yaml",
		},
		ContractRules: []ContractRule{
			{Type: "no-circular-dependencies"},
		},
	}

	result, err := runIntegrationContractCheckFromSpec(root, check)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Diamond is not a cycle
	if result.Status != "pass" {
		t.Errorf("expected status 'pass', got %q", result.Status)
	}
}

func TestCheckConsumersSatisfied_ProviderFromCorrectComponent(t *testing.T) {
	integrationMap := &IntegrationMap{
		Components: map[string]Component{
			"auth-service": {
				Provides: []Provider{{Name: "AuthProvider"}},
				Consumes: []Consumer{},
			},
			"user-service": {
				Provides: []Provider{},
				Consumes: []Consumer{{Name: "AuthProvider", From: "auth-service"}},
			},
		},
	}
	graph := buildComponentGraph(integrationMap)

	issues, status := checkConsumersSatisfied(integrationMap, graph)
	if status != "pass" {
		t.Errorf("expected pass, got %q", status)
	}
	if len(issues) != 0 {
		t.Errorf("expected no issues, got %d", len(issues))
	}
}

func TestCheckProvidersImplemented_DefinitionExists(t *testing.T) {
	root := t.TempDir()

	// Create definition file
	if err := os.WriteFile(filepath.Join(root, "auth.ts"), []byte("interface AuthProvider {}"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	integrationMap := &IntegrationMap{
		Components: map[string]Component{
			"auth-service": {
				Provides: []Provider{{Name: "AuthProvider", Definition: "auth.ts"}},
			},
		},
	}

	issues, status := checkProvidersImplemented(root, integrationMap, "")
	if status != "pass" {
		t.Errorf("expected pass, got %q", status)
	}
	if len(issues) != 0 {
		t.Errorf("expected no issues, got %d", len(issues))
	}
}

func TestBuildComponentGraph_ConsumerNoFrom(t *testing.T) {
	integrationMap := &IntegrationMap{
		Components: map[string]Component{
			"service-a": {
				Provides: []Provider{},
				Consumes: []Consumer{{Name: "Provider", From: ""}}, // No from specified
			},
		},
	}

	graph := buildComponentGraph(integrationMap)

	// Dependencies should be empty since no 'from' specified
	if len(graph.dependencies["service-a"]) != 0 {
		t.Errorf("expected no dependencies when 'from' is empty, got %d", len(graph.dependencies["service-a"]))
	}
}

func TestRunIntegrationContractCheck_MultipleConsumersOfSameInterface(t *testing.T) {
	root := t.TempDir()

	// Two consumers of the same interface
	contractsDir := filepath.Join(root, "contracts")
	if err := os.MkdirAll(contractsDir, 0755); err != nil {
		t.Fatalf("failed to create contracts dir: %v", err)
	}

	mapContent := `components:
  provider-service:
    provides:
      - name: SharedProvider
    consumes: []
  consumer-a:
    provides: []
    consumes:
      - name: SharedProvider
        from: provider-service
  consumer-b:
    provides: []
    consumes:
      - name: SharedProvider
        from: provider-service
`
	if err := os.WriteFile(filepath.Join(root, "contracts", "integration-map.yaml"), []byte(mapContent), 0644); err != nil {
		t.Fatalf("failed to write integration map: %v", err)
	}

	check := Check{
		ID:   "test-multiple-consumers",
		Type: "integration-contract",
		Contracts: ContractsConfig{
			Map: "contracts/integration-map.yaml",
		},
		ContractRules: []ContractRule{
			{Type: "all-consumers-satisfied"},
			{Type: "no-circular-dependencies"},
		},
	}

	result, err := runIntegrationContractCheckFromSpec(root, check)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != "pass" {
		t.Errorf("expected status 'pass', got %q", result.Status)
	}
}
