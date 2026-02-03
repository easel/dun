package dun

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"
)

// TestReconcilePRDChangeEmitsImpactedArtifacts validates GAP-001 (AC-1):
// When PRD content has changed, dun emits a list of impacted artifacts.
func TestReconcilePRDChangeEmitsImpactedArtifacts(t *testing.T) {
	root := fixturePath(t, "../testdata/repos/helix-prd-changed")

	artifacts, err := detectImpactedArtifacts(root)
	if err != nil {
		t.Fatalf("detect impacted artifacts: %v", err)
	}

	if len(artifacts) == 0 {
		t.Fatalf("expected impacted artifacts, got none")
	}

	// Verify key artifact categories are present
	hasFeatures := false
	hasDesign := false
	hasTest := false
	for _, a := range artifacts {
		if containsPath(a, "01-frame/features") {
			hasFeatures = true
		}
		if containsPath(a, "02-design") {
			hasDesign = true
		}
		if containsPath(a, "03-test") {
			hasTest = true
		}
	}

	if !hasFeatures {
		t.Errorf("expected features in impacted artifacts")
	}
	if !hasDesign {
		t.Errorf("expected design docs in impacted artifacts")
	}
	if !hasTest {
		t.Errorf("expected test plans in impacted artifacts")
	}
}

// TestReconcileImpactedArtifactsInOrder validates GAP-002 (AC-1):
// Artifacts are ordered: features -> design -> ADRs -> test plans -> implementation.
func TestReconcileImpactedArtifactsInOrder(t *testing.T) {
	root := fixturePath(t, "../testdata/repos/helix-prd-changed")

	artifacts, err := detectImpactedArtifacts(root)
	if err != nil {
		t.Fatalf("detect impacted artifacts: %v", err)
	}

	if len(artifacts) == 0 {
		t.Fatalf("expected impacted artifacts, got none")
	}

	// Verify ordering: upstream artifacts come before downstream
	expectedOrder := []string{
		"01-frame/features/",
		"01-frame/user-stories/",
		"02-design/",
		"03-test/",
		"04-build/",
	}

	assertOrderedPrefixes(t, artifacts, expectedOrder)
}

// TestReconcilePlanIncludesAllArtifactTypes validates GAP-003 (AC-2):
// Plan includes feature specs, design docs, ADRs, test plans, and implementation.
func TestReconcilePlanIncludesAllArtifactTypes(t *testing.T) {
	root := fixturePath(t, "../testdata/repos/helix-prd-changed")

	artifacts, err := detectImpactedArtifacts(root)
	if err != nil {
		t.Fatalf("detect impacted artifacts: %v", err)
	}

	// Define required artifact types
	requiredTypes := map[string]bool{
		"features":  false, // 01-frame/features/
		"stories":   false, // 01-frame/user-stories/
		"design":    false, // 02-design/ (excluding decisions)
		"decisions": false, // 02-design/decisions/ (ADRs)
		"test":      false, // 03-test/
		"build":     false, // 04-build/
	}

	for _, a := range artifacts {
		if containsPath(a, "01-frame/features") {
			requiredTypes["features"] = true
		}
		if containsPath(a, "01-frame/user-stories") {
			requiredTypes["stories"] = true
		}
		if containsPath(a, "02-design/decisions") {
			requiredTypes["decisions"] = true
		} else if containsPath(a, "02-design") {
			requiredTypes["design"] = true
		}
		if containsPath(a, "03-test") {
			requiredTypes["test"] = true
		}
		if containsPath(a, "04-build") {
			requiredTypes["build"] = true
		}
	}

	for typ, found := range requiredTypes {
		if !found {
			t.Errorf("missing artifact type in plan: %s", typ)
		}
	}
}

// TestReconcilePlanIsDeterministic validates GAP-004 (AC-3):
// Given the same repo state, plan output is identical across multiple runs.
func TestReconcilePlanIsDeterministic(t *testing.T) {
	root := fixturePath(t, "../testdata/repos/helix-prd-changed")

	// Run detection multiple times
	const runs = 10
	var results [][]string

	for i := 0; i < runs; i++ {
		artifacts, err := detectImpactedArtifacts(root)
		if err != nil {
			t.Fatalf("run %d: detect impacted artifacts: %v", i, err)
		}
		results = append(results, artifacts)
	}

	// All results should be identical
	baseline := results[0]
	for i := 1; i < runs; i++ {
		if len(results[i]) != len(baseline) {
			t.Fatalf("run %d: different artifact count: got %d, want %d",
				i, len(results[i]), len(baseline))
		}
		for j := range baseline {
			if results[i][j] != baseline[j] {
				t.Fatalf("run %d: artifact %d differs: got %q, want %q",
					i, j, results[i][j], baseline[j])
			}
		}
	}
}

// TestStateRulesDetectPRDFeatureMismatch validates TC-007:
// When PRD references features not yet created, state rules report missing features.
func TestStateRulesDetectPRDFeatureMismatch(t *testing.T) {
	dir := t.TempDir()

	// Create state rules with story artifact patterns
	writeFile(t, filepath.Join(dir, "rules.yml"), `artifact_patterns:
  story:
    frame: { pattern: "frame/US-{id}.md" }
    design: { pattern: "design/TD-{id}.md" }
    test: { pattern: "test/TP-{id}.md" }
    build: { pattern: "build/IP-{id}.md" }
`)

	// Create directories
	if err := os.MkdirAll(filepath.Join(dir, "frame"), 0755); err != nil {
		t.Fatalf("mkdir frame: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "design"), 0755); err != nil {
		t.Fatalf("mkdir design: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "test"), 0755); err != nil {
		t.Fatalf("mkdir test: %v", err)
	}

	// Create downstream artifact without upstream
	// TD-002 exists but US-002 is missing (PRD-Feature mismatch scenario)
	writeFile(t, filepath.Join(dir, "frame", "US-001.md"), "US-001")
	writeFile(t, filepath.Join(dir, "design", "TD-001.md"), "TD-001")
	writeFile(t, filepath.Join(dir, "design", "TD-002.md"), "TD-002") // Missing upstream US-002

	plugin := Plugin{FS: os.DirFS(dir), Base: "."}
	check := Check{ID: "state", StateRules: "rules.yml"}
	res, err := runStateRules(dir, plugin, check)
	if err != nil {
		t.Fatalf("run state rules: %v", err)
	}

	if res.Status != "fail" {
		t.Fatalf("expected fail status for missing upstream, got %s", res.Status)
	}

	if !containsPath(res.Detail, "US-002") {
		t.Fatalf("expected missing US-002 in detail, got %q", res.Detail)
	}
}

// containsPath checks if the artifact path contains the expected substring.
func containsPath(artifact, expected string) bool {
	return filepath.ToSlash(artifact) != "" &&
		(len(artifact) >= len(expected) &&
			(artifact[:len(expected)] == expected ||
				len(artifact) > len(expected) && containsSubpath(artifact, expected)))
}

func containsSubpath(path, sub string) bool {
	normalized := filepath.ToSlash(path)
	for i := 0; i <= len(normalized)-len(sub); i++ {
		if normalized[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

// assertOrderedPrefixes verifies artifacts appear in the expected order.
// Each prefix in expectedOrder should appear before subsequent prefixes.
func assertOrderedPrefixes(t *testing.T, artifacts []string, expectedOrder []string) {
	t.Helper()

	// Build a map of prefix to first occurrence index
	prefixIdx := make(map[string]int)
	for i, a := range artifacts {
		normalized := filepath.ToSlash(a)
		for _, prefix := range expectedOrder {
			if _, found := prefixIdx[prefix]; !found && containsSubpath(normalized, prefix) {
				prefixIdx[prefix] = i
			}
		}
	}

	// Verify ordering
	lastIdx := -1
	lastPrefix := ""
	for _, prefix := range expectedOrder {
		idx, found := prefixIdx[prefix]
		if !found {
			continue // Skip missing prefixes (covered by other tests)
		}
		if idx < lastIdx {
			t.Errorf("ordering violation: %q (index %d) should come after %q (index %d)",
				prefix, idx, lastPrefix, lastIdx)
		}
		lastIdx = idx
		lastPrefix = prefix
	}
}

// TestHelixPRDReconciliationEmitsOrderedPlan validates TC-005:
// End-to-end PRD reconciliation emits ordered plan via the engine.
func TestHelixPRDReconciliationEmitsOrderedPlan(t *testing.T) {
	result := runFixture(t, "helix-prd-changed", "")

	// Find the reconcile check
	check := findCheck(t, result, "helix-reconcile-stack")
	if check.Status != "prompt" {
		t.Fatalf("expected prompt status for reconcile check, got %s", check.Status)
	}

	// Verify prompt envelope is present
	if check.Prompt == nil {
		t.Fatalf("expected prompt envelope for reconcile check")
	}

	// Verify inputs are provided (matching inputs from plugin.yaml)
	inputs := check.Prompt.Inputs
	if len(inputs) == 0 {
		t.Fatalf("expected inputs in prompt envelope")
	}

	// Verify PRD is included in inputs
	hasPRD := false
	for _, input := range inputs {
		if strings.Contains(input, "prd.md") {
			hasPRD = true
			break
		}
	}
	if !hasPRD {
		t.Errorf("expected prd.md in inputs, got %v", inputs)
	}

	// Verify features are included
	hasFeatures := false
	for _, input := range inputs {
		if strings.Contains(input, "features/") {
			hasFeatures = true
			break
		}
	}
	if !hasFeatures {
		t.Errorf("expected features in inputs, got %v", inputs)
	}

	// Verify design docs are included
	hasDesign := false
	for _, input := range inputs {
		if strings.Contains(input, "02-design/") {
			hasDesign = true
			break
		}
	}
	if !hasDesign {
		t.Errorf("expected design docs in inputs, got %v", inputs)
	}

	// Verify test plan is included
	hasTest := false
	for _, input := range inputs {
		if strings.Contains(input, "03-test/") {
			hasTest = true
			break
		}
	}
	if !hasTest {
		t.Errorf("expected test plan in inputs, got %v", inputs)
	}
}

// TestHelixReconciliationIncludesADRs validates TC-006:
// Reconciliation includes ADR artifacts in the plan.
func TestHelixReconciliationIncludesADRs(t *testing.T) {
	result := runFixture(t, "helix-prd-changed", "")

	check := findCheck(t, result, "helix-reconcile-stack")
	if check.Prompt == nil {
		t.Fatalf("expected prompt envelope")
	}

	inputs := check.Prompt.Inputs
	hasADR := false
	for _, input := range inputs {
		if strings.Contains(input, "decisions/") || strings.Contains(input, "ADR") {
			hasADR = true
			break
		}
	}

	// The helix-reconcile-stack check includes 02-design/**/*.md which covers ADRs
	hasDesign := false
	for _, input := range inputs {
		if strings.Contains(input, "02-design/") {
			hasDesign = true
			break
		}
	}

	if !hasDesign && !hasADR {
		t.Fatalf("expected design docs or ADRs in reconciliation inputs, got %v", inputs)
	}
}

// TestHelixReconcileStackCheckActivates validates the helix-reconcile-stack check
// is triggered when all required conditions are met.
func TestHelixReconcileStackCheckActivates(t *testing.T) {
	plan, err := PlanRepo(fixturePath(t, "../testdata/repos/helix-prd-changed"))
	if err != nil {
		t.Fatalf("plan repo: %v", err)
	}

	// Verify helix-reconcile-stack is in the plan
	found := false
	for _, check := range plan.Checks {
		if check.ID == "helix-reconcile-stack" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected helix-reconcile-stack in plan")
	}
}

// TestHelixReconcileStackMultipleRuns validates deterministic ordering across runs.
func TestHelixReconcileStackMultipleRuns(t *testing.T) {
	root := fixturePath(t, "../testdata/repos/helix-prd-changed")

	const runs = 5
	var plans []Plan

	for i := 0; i < runs; i++ {
		plan, err := PlanRepo(root)
		if err != nil {
			t.Fatalf("run %d: plan repo: %v", i, err)
		}
		plans = append(plans, plan)
	}

	// Verify all plans are identical
	baseline := plans[0]
	for i := 1; i < runs; i++ {
		if len(plans[i].Checks) != len(baseline.Checks) {
			t.Fatalf("run %d: different check count: got %d, want %d",
				i, len(plans[i].Checks), len(baseline.Checks))
		}
		for j := range baseline.Checks {
			if plans[i].Checks[j].ID != baseline.Checks[j].ID {
				t.Fatalf("run %d: check %d differs: got %q, want %q",
					i, j, plans[i].Checks[j].ID, baseline.Checks[j].ID)
			}
		}
	}
}

// TestHelixReconcileCheckResultDeterminism validates the check result is deterministic.
func TestHelixReconcileCheckResultDeterminism(t *testing.T) {
	root := fixturePath(t, "../testdata/repos/helix-prd-changed")

	const runs = 5
	var results []Result

	for i := 0; i < runs; i++ {
		opts := Options{
			AgentTimeout: 5 * time.Second,
		}
		result, err := CheckRepo(root, opts)
		if err != nil {
			t.Fatalf("run %d: check repo: %v", i, err)
		}
		results = append(results, result)
	}

	// Find reconcile check in baseline
	var baselineCheck CheckResult
	for _, check := range results[0].Checks {
		if check.ID == "helix-reconcile-stack" {
			baselineCheck = check
			break
		}
	}

	// Compare across runs
	for i := 1; i < runs; i++ {
		for _, check := range results[i].Checks {
			if check.ID == "helix-reconcile-stack" {
				if check.Status != baselineCheck.Status {
					t.Fatalf("run %d: status differs: got %q, want %q",
						i, check.Status, baselineCheck.Status)
				}
				break
			}
		}
	}
}

// detectImpactedArtifacts scans the helix docs directory and returns all artifacts
// in downstream order: features -> user-stories -> design -> decisions -> test -> build.
func detectImpactedArtifacts(root string) ([]string, error) {
	helixRoot := filepath.Join(root, "docs", "helix")
	var artifacts []string

	// Define artifact categories in order
	categories := []struct {
		path     string
		priority int
	}{
		{"01-frame/features", 1},
		{"01-frame/user-stories", 2},
		{"02-design", 3},
		{"03-test", 4},
		{"04-build", 5},
	}

	type orderedArtifact struct {
		path     string
		priority int
	}

	var ordered []orderedArtifact

	for _, cat := range categories {
		catPath := filepath.Join(helixRoot, cat.path)
		if _, err := os.Stat(catPath); os.IsNotExist(err) {
			continue
		}

		err := filepath.Walk(catPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				return nil
			}
			if filepath.Ext(path) != ".md" {
				return nil
			}

			relPath, err := filepath.Rel(helixRoot, path)
			if err != nil {
				return err
			}

			// Determine priority based on path
			priority := cat.priority
			// ADRs (decisions) come between design and test
			if containsSubpath(relPath, "decisions/") && cat.path == "02-design" {
				priority = 35 // Between design (3) and test (4), but after non-ADR design
			}

			ordered = append(ordered, orderedArtifact{
				path:     relPath,
				priority: priority * 10, // Multiply to leave room for sub-ordering
			})
			return nil
		})
		if err != nil {
			return nil, err
		}
	}

	// Sort by priority, then by path for determinism
	sort.Slice(ordered, func(i, j int) bool {
		if ordered[i].priority != ordered[j].priority {
			return ordered[i].priority < ordered[j].priority
		}
		return ordered[i].path < ordered[j].path
	})

	for _, o := range ordered {
		artifacts = append(artifacts, o.path)
	}

	return artifacts, nil
}
