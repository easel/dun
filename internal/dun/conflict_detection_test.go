package dun

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func runConflictDetectionCheckFromSpec(root string, check Check) (CheckResult, error) {
	def := CheckDefinition{ID: check.ID}
	config := ConflictDetectionConfig{Tracking: check.Tracking, ConflictRules: check.ConflictRules}
	return runConflictDetectionCheck(root, def, config)
}

func TestConflictDetection_NoManifest(t *testing.T) {
	root := t.TempDir()

	check := Check{
		ID: "test-conflict",
		Tracking: TrackingConfig{
			Manifest: ".dun/work-in-progress.yaml",
		},
	}

	result, err := runConflictDetectionCheckFromSpec(root, check)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Status != "pass" {
		t.Errorf("expected pass for missing manifest, got %s", result.Status)
	}
	if !strings.Contains(result.Signal, "no WIP manifest") {
		t.Errorf("expected 'no WIP manifest' in signal, got %s", result.Signal)
	}
}

func TestConflictDetection_InvalidManifest(t *testing.T) {
	root := t.TempDir()

	// Create invalid YAML
	manifestDir := filepath.Join(root, ".dun")
	if err := os.MkdirAll(manifestDir, 0755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(manifestDir, "work-in-progress.yaml"), "invalid: [yaml: content")

	check := Check{
		ID: "test-conflict",
		Tracking: TrackingConfig{
			Manifest: ".dun/work-in-progress.yaml",
		},
	}

	result, err := runConflictDetectionCheckFromSpec(root, check)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Status != "fail" {
		t.Errorf("expected fail for invalid manifest, got %s", result.Status)
	}
	if !strings.Contains(result.Signal, "failed to load") {
		t.Errorf("expected 'failed to load' in signal, got %s", result.Signal)
	}
}

func TestConflictDetection_EmptyManifest(t *testing.T) {
	root := t.TempDir()

	// Create empty manifest
	manifestDir := filepath.Join(root, ".dun")
	if err := os.MkdirAll(manifestDir, 0755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(manifestDir, "work-in-progress.yaml"), "claims: []")

	check := Check{
		ID: "test-conflict",
		Tracking: TrackingConfig{
			Manifest: ".dun/work-in-progress.yaml",
		},
		ConflictRules: []ConflictRule{
			{Type: "no-overlap", Scope: "file", Required: true},
		},
	}

	result, err := runConflictDetectionCheckFromSpec(root, check)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Status != "pass" {
		t.Errorf("expected pass for empty manifest, got %s", result.Status)
	}
}

func TestConflictDetection_NoOverlap_FileScope_Pass(t *testing.T) {
	root := t.TempDir()

	manifest := `claims:
  - agent: agent-1
    files:
      - path: internal/auth/handler.go
        scope: file
    claimed_at: "2026-01-31T10:00:00Z"
  - agent: agent-2
    files:
      - path: internal/user/service.go
        scope: file
    claimed_at: "2026-01-31T10:05:00Z"
`
	setupManifest(t, root, manifest)

	check := Check{
		ID: "test-conflict",
		Tracking: TrackingConfig{
			Manifest: ".dun/work-in-progress.yaml",
		},
		ConflictRules: []ConflictRule{
			{Type: "no-overlap", Scope: "file", Required: true},
		},
	}

	result, err := runConflictDetectionCheckFromSpec(root, check)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Status != "pass" {
		t.Errorf("expected pass (no overlap), got %s", result.Status)
	}
}

func TestConflictDetection_NoOverlap_FileScope_Fail(t *testing.T) {
	root := t.TempDir()

	manifest := `claims:
  - agent: agent-1
    files:
      - path: internal/auth/handler.go
        scope: file
    claimed_at: "2026-01-31T10:00:00Z"
  - agent: agent-2
    files:
      - path: internal/auth/handler.go
        scope: file
    claimed_at: "2026-01-31T10:05:00Z"
`
	setupManifest(t, root, manifest)

	check := Check{
		ID: "test-conflict",
		Tracking: TrackingConfig{
			Manifest: ".dun/work-in-progress.yaml",
		},
		ConflictRules: []ConflictRule{
			{Type: "no-overlap", Scope: "file", Required: true},
		},
	}

	result, err := runConflictDetectionCheckFromSpec(root, check)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Status != "fail" {
		t.Errorf("expected fail (overlap), got %s", result.Status)
	}
	if len(result.Issues) != 1 {
		t.Errorf("expected 1 issue, got %d", len(result.Issues))
	}
	if !strings.Contains(result.Issues[0].Summary, "agent-1") || !strings.Contains(result.Issues[0].Summary, "agent-2") {
		t.Errorf("expected issue to mention both agents, got %s", result.Issues[0].Summary)
	}
}

func TestConflictDetection_NoOverlap_FileScope_WarnOnly(t *testing.T) {
	root := t.TempDir()

	manifest := `claims:
  - agent: agent-1
    files:
      - path: internal/auth/handler.go
        scope: file
    claimed_at: "2026-01-31T10:00:00Z"
  - agent: agent-2
    files:
      - path: internal/auth/handler.go
        scope: file
    claimed_at: "2026-01-31T10:05:00Z"
`
	setupManifest(t, root, manifest)

	check := Check{
		ID: "test-conflict",
		Tracking: TrackingConfig{
			Manifest: ".dun/work-in-progress.yaml",
		},
		ConflictRules: []ConflictRule{
			{Type: "no-overlap", Scope: "file", Required: false},
		},
	}

	result, err := runConflictDetectionCheckFromSpec(root, check)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Status != "warn" {
		t.Errorf("expected warn (not required), got %s", result.Status)
	}
}

func TestConflictDetection_NoOverlap_FunctionScope_Pass(t *testing.T) {
	root := t.TempDir()

	manifest := `claims:
  - agent: agent-1
    files:
      - path: internal/auth/handler.go
        scope: function
        function: HandleLogin
    claimed_at: "2026-01-31T10:00:00Z"
  - agent: agent-2
    files:
      - path: internal/auth/handler.go
        scope: function
        function: HandleLogout
    claimed_at: "2026-01-31T10:05:00Z"
`
	setupManifest(t, root, manifest)

	check := Check{
		ID: "test-conflict",
		Tracking: TrackingConfig{
			Manifest: ".dun/work-in-progress.yaml",
		},
		ConflictRules: []ConflictRule{
			{Type: "no-overlap", Scope: "function", Required: true},
		},
	}

	result, err := runConflictDetectionCheckFromSpec(root, check)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Status != "pass" {
		t.Errorf("expected pass (different functions), got %s", result.Status)
	}
}

func TestConflictDetection_NoOverlap_FunctionScope_SameFunction(t *testing.T) {
	root := t.TempDir()

	manifest := `claims:
  - agent: agent-1
    files:
      - path: internal/auth/handler.go
        scope: function
        function: HandleLogin
    claimed_at: "2026-01-31T10:00:00Z"
  - agent: agent-2
    files:
      - path: internal/auth/handler.go
        scope: function
        function: HandleLogin
    claimed_at: "2026-01-31T10:05:00Z"
`
	setupManifest(t, root, manifest)

	check := Check{
		ID: "test-conflict",
		Tracking: TrackingConfig{
			Manifest: ".dun/work-in-progress.yaml",
		},
		ConflictRules: []ConflictRule{
			{Type: "no-overlap", Scope: "function", Required: true},
		},
	}

	result, err := runConflictDetectionCheckFromSpec(root, check)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Status != "fail" {
		t.Errorf("expected fail (same function), got %s", result.Status)
	}
	if len(result.Issues) < 1 {
		t.Errorf("expected at least 1 issue, got %d", len(result.Issues))
	}
	if !strings.Contains(result.Issues[0].Summary, "HandleLogin") {
		t.Errorf("expected issue to mention HandleLogin, got %s", result.Issues[0].Summary)
	}
}

func TestConflictDetection_NoOverlap_FileFunctionConflict(t *testing.T) {
	root := t.TempDir()

	manifest := `claims:
  - agent: agent-1
    files:
      - path: internal/auth/handler.go
        scope: file
    claimed_at: "2026-01-31T10:00:00Z"
  - agent: agent-2
    files:
      - path: internal/auth/handler.go
        scope: function
        function: HandleLogin
    claimed_at: "2026-01-31T10:05:00Z"
`
	setupManifest(t, root, manifest)

	check := Check{
		ID: "test-conflict",
		Tracking: TrackingConfig{
			Manifest: ".dun/work-in-progress.yaml",
		},
		ConflictRules: []ConflictRule{
			{Type: "no-overlap", Scope: "function", Required: true},
		},
	}

	result, err := runConflictDetectionCheckFromSpec(root, check)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Status != "fail" {
		t.Errorf("expected fail (file vs function conflict), got %s", result.Status)
	}
	if len(result.Issues) < 1 {
		t.Errorf("expected at least 1 issue, got %d", len(result.Issues))
	}
}

func TestConflictDetection_ClaimBeforeEdit_Pass(t *testing.T) {
	root := t.TempDir()

	// Mock git diff
	origGitDiff := gitDiffFilesFunc
	gitDiffFilesFunc = func(r, baseline string) ([]string, error) {
		return []string{"internal/auth/handler.go"}, nil
	}
	defer func() { gitDiffFilesFunc = origGitDiff }()

	manifest := `claims:
  - agent: agent-1
    files:
      - path: internal/auth/handler.go
        scope: file
    claimed_at: "2026-01-31T10:00:00Z"
`
	setupManifest(t, root, manifest)

	check := Check{
		ID: "test-conflict",
		Tracking: TrackingConfig{
			Manifest: ".dun/work-in-progress.yaml",
		},
		ConflictRules: []ConflictRule{
			{Type: "claim-before-edit", Required: true},
		},
	}

	result, err := runConflictDetectionCheckFromSpec(root, check)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Status != "pass" {
		t.Errorf("expected pass (file has claim), got %s", result.Status)
	}
}

func TestConflictDetection_ClaimBeforeEdit_Fail(t *testing.T) {
	root := t.TempDir()

	// Mock git diff to return a file without claim
	origGitDiff := gitDiffFilesFunc
	gitDiffFilesFunc = func(r, baseline string) ([]string, error) {
		return []string{"internal/user/service.go"}, nil
	}
	defer func() { gitDiffFilesFunc = origGitDiff }()

	manifest := `claims:
  - agent: agent-1
    files:
      - path: internal/auth/handler.go
        scope: file
    claimed_at: "2026-01-31T10:00:00Z"
`
	setupManifest(t, root, manifest)

	check := Check{
		ID: "test-conflict",
		Tracking: TrackingConfig{
			Manifest: ".dun/work-in-progress.yaml",
		},
		ConflictRules: []ConflictRule{
			{Type: "claim-before-edit", Required: true},
		},
	}

	result, err := runConflictDetectionCheckFromSpec(root, check)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Status != "fail" {
		t.Errorf("expected fail (modified without claim), got %s", result.Status)
	}
	if len(result.Issues) != 1 {
		t.Errorf("expected 1 issue, got %d", len(result.Issues))
	}
	if !strings.Contains(result.Issues[0].Summary, "without claim") {
		t.Errorf("expected 'without claim' in issue, got %s", result.Issues[0].Summary)
	}
}

func TestConflictDetection_ClaimBeforeEdit_WarnOnly(t *testing.T) {
	root := t.TempDir()

	// Mock git diff
	origGitDiff := gitDiffFilesFunc
	gitDiffFilesFunc = func(r, baseline string) ([]string, error) {
		return []string{"internal/user/service.go"}, nil
	}
	defer func() { gitDiffFilesFunc = origGitDiff }()

	manifest := `claims: []`
	setupManifest(t, root, manifest)

	check := Check{
		ID: "test-conflict",
		Tracking: TrackingConfig{
			Manifest: ".dun/work-in-progress.yaml",
		},
		ConflictRules: []ConflictRule{
			{Type: "claim-before-edit", Required: false},
		},
	}

	result, err := runConflictDetectionCheckFromSpec(root, check)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Status != "warn" {
		t.Errorf("expected warn (not required), got %s", result.Status)
	}
}

func TestConflictDetection_ClaimBeforeEdit_GitDiffError(t *testing.T) {
	root := t.TempDir()

	// Mock git diff to fail
	origGitDiff := gitDiffFilesFunc
	gitDiffFilesFunc = func(r, baseline string) ([]string, error) {
		return nil, os.ErrNotExist
	}
	defer func() { gitDiffFilesFunc = origGitDiff }()

	manifest := `claims: []`
	setupManifest(t, root, manifest)

	check := Check{
		ID: "test-conflict",
		Tracking: TrackingConfig{
			Manifest: ".dun/work-in-progress.yaml",
		},
		ConflictRules: []ConflictRule{
			{Type: "claim-before-edit", Required: true},
		},
	}

	result, err := runConflictDetectionCheckFromSpec(root, check)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should pass when git diff fails (skip the check)
	if result.Status != "pass" {
		t.Errorf("expected pass when git diff fails, got %s", result.Status)
	}
}

func TestConflictDetection_ClaimBeforeEdit_NoChanges(t *testing.T) {
	root := t.TempDir()

	// Mock git diff to return no changes
	origGitDiff := gitDiffFilesFunc
	gitDiffFilesFunc = func(r, baseline string) ([]string, error) {
		return nil, nil
	}
	defer func() { gitDiffFilesFunc = origGitDiff }()

	manifest := `claims: []`
	setupManifest(t, root, manifest)

	check := Check{
		ID: "test-conflict",
		Tracking: TrackingConfig{
			Manifest: ".dun/work-in-progress.yaml",
		},
		ConflictRules: []ConflictRule{
			{Type: "claim-before-edit", Required: true},
		},
	}

	result, err := runConflictDetectionCheckFromSpec(root, check)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Status != "pass" {
		t.Errorf("expected pass (no changes), got %s", result.Status)
	}
}

func TestConflictDetection_MultipleRules(t *testing.T) {
	root := t.TempDir()

	// Mock git diff
	origGitDiff := gitDiffFilesFunc
	gitDiffFilesFunc = func(r, baseline string) ([]string, error) {
		return []string{"internal/unclaimed.go"}, nil
	}
	defer func() { gitDiffFilesFunc = origGitDiff }()

	manifest := `claims:
  - agent: agent-1
    files:
      - path: internal/auth/handler.go
        scope: file
    claimed_at: "2026-01-31T10:00:00Z"
  - agent: agent-2
    files:
      - path: internal/auth/handler.go
        scope: file
    claimed_at: "2026-01-31T10:05:00Z"
`
	setupManifest(t, root, manifest)

	check := Check{
		ID: "test-conflict",
		Tracking: TrackingConfig{
			Manifest: ".dun/work-in-progress.yaml",
		},
		ConflictRules: []ConflictRule{
			{Type: "no-overlap", Scope: "file", Required: true},
			{Type: "claim-before-edit", Required: true},
		},
	}

	result, err := runConflictDetectionCheckFromSpec(root, check)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Status != "fail" {
		t.Errorf("expected fail, got %s", result.Status)
	}
	// Should have issues from both rules
	if len(result.Issues) != 2 {
		t.Errorf("expected 2 issues, got %d", len(result.Issues))
	}
}

func TestConflictDetection_MixedRequiredOptional(t *testing.T) {
	root := t.TempDir()

	// Mock git diff
	origGitDiff := gitDiffFilesFunc
	gitDiffFilesFunc = func(r, baseline string) ([]string, error) {
		return []string{"internal/unclaimed.go"}, nil
	}
	defer func() { gitDiffFilesFunc = origGitDiff }()

	manifest := `claims:
  - agent: agent-1
    files:
      - path: internal/auth/handler.go
        scope: file
    claimed_at: "2026-01-31T10:00:00Z"
  - agent: agent-2
    files:
      - path: internal/auth/handler.go
        scope: file
    claimed_at: "2026-01-31T10:05:00Z"
`
	setupManifest(t, root, manifest)

	check := Check{
		ID: "test-conflict",
		Tracking: TrackingConfig{
			Manifest: ".dun/work-in-progress.yaml",
		},
		ConflictRules: []ConflictRule{
			{Type: "no-overlap", Scope: "file", Required: false}, // warn only
			{Type: "claim-before-edit", Required: true},          // required
		},
	}

	result, err := runConflictDetectionCheckFromSpec(root, check)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should be fail because claim-before-edit is required
	if result.Status != "fail" {
		t.Errorf("expected fail, got %s", result.Status)
	}
}

func TestConflictDetection_DefaultManifestPath(t *testing.T) {
	root := t.TempDir()

	manifest := `claims: []`
	setupManifest(t, root, manifest)

	check := Check{
		ID: "test-conflict",
		// No manifest path specified - should use default
		Tracking: TrackingConfig{
			Manifest: "", // empty = use default
		},
	}

	result, err := runConflictDetectionCheckFromSpec(root, check)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Status != "pass" {
		t.Errorf("expected pass with default path, got %s", result.Status)
	}
}

func TestExtractConflictDetectionConfig(t *testing.T) {
	check := Check{
		Tracking: TrackingConfig{
			Manifest:     ".dun/wip.yaml",
			ClaimPattern: "// WIP: {agent}",
		},
		ConflictRules: []ConflictRule{
			{Type: "no-overlap", Scope: "function", Required: true},
			{Type: "claim-before-edit", Required: false},
		},
	}

	config := ConflictDetectionConfig{Tracking: check.Tracking, ConflictRules: check.ConflictRules}

	if config.Tracking.Manifest != ".dun/wip.yaml" {
		t.Errorf("expected manifest path '.dun/wip.yaml', got %s", config.Tracking.Manifest)
	}
	if config.Tracking.ClaimPattern != "// WIP: {agent}" {
		t.Errorf("expected claim pattern, got %s", config.Tracking.ClaimPattern)
	}
	if len(config.ConflictRules) != 2 {
		t.Errorf("expected 2 rules, got %d", len(config.ConflictRules))
	}
	if config.ConflictRules[0].Type != "no-overlap" {
		t.Errorf("expected first rule type 'no-overlap', got %s", config.ConflictRules[0].Type)
	}
	if config.ConflictRules[0].Scope != "function" {
		t.Errorf("expected first rule scope 'function', got %s", config.ConflictRules[0].Scope)
	}
	if !config.ConflictRules[0].Required {
		t.Error("expected first rule to be required")
	}
}

func TestBuildFileClaimsMap(t *testing.T) {
	manifest := &WIPManifest{
		Claims: []Claim{
			{
				Agent: "agent-1",
				Files: []FileClaim{
					{Path: "file1.go", Scope: "file"},
					{Path: "file2.go", Scope: "function", Function: "Foo"},
				},
				ClaimedAt: time.Now(),
			},
			{
				Agent: "agent-2",
				Files: []FileClaim{
					{Path: "file1.go", Scope: "function", Function: "Bar"},
				},
				ClaimedAt: time.Now(),
			},
		},
	}

	result := buildFileClaimsMap(manifest)

	if len(result["file1.go"]) != 2 {
		t.Errorf("expected 2 claims on file1.go, got %d", len(result["file1.go"]))
	}
	if len(result["file2.go"]) != 1 {
		t.Errorf("expected 1 claim on file2.go, got %d", len(result["file2.go"]))
	}
}

func TestCheckNoOverlap_EmptyClaims(t *testing.T) {
	fileClaims := make(map[string][]ClaimInfo)
	issues, status := checkNoOverlap(fileClaims, "file", true)

	if status != "pass" {
		t.Errorf("expected pass for empty claims, got %s", status)
	}
	if len(issues) != 0 {
		t.Errorf("expected 0 issues, got %d", len(issues))
	}
}

func TestCheckNoOverlap_SingleClaim(t *testing.T) {
	fileClaims := map[string][]ClaimInfo{
		"file.go": {
			{Agent: "agent-1", Scope: "file"},
		},
	}
	issues, status := checkNoOverlap(fileClaims, "file", true)

	if status != "pass" {
		t.Errorf("expected pass for single claim, got %s", status)
	}
	if len(issues) != 0 {
		t.Errorf("expected 0 issues, got %d", len(issues))
	}
}

func TestCheckFileOverlap_SameAgent(t *testing.T) {
	claims := []ClaimInfo{
		{Agent: "agent-1", Scope: "file"},
		{Agent: "agent-1", Scope: "function", Function: "Foo"},
	}
	issues := checkFileOverlap("file.go", claims)

	// Same agent, no conflict
	if len(issues) != 0 {
		t.Errorf("expected 0 issues for same agent, got %d", len(issues))
	}
}

func TestCheckFunctionOverlaps_NoOverlap(t *testing.T) {
	claims := []ClaimInfo{
		{Agent: "agent-1", Scope: "function", Function: "Foo"},
		{Agent: "agent-2", Scope: "function", Function: "Bar"},
	}
	issues := checkFunctionOverlaps("file.go", claims)

	if len(issues) != 0 {
		t.Errorf("expected 0 issues for different functions, got %d", len(issues))
	}
}

func TestCheckClaimBeforeEdit_AllClaimed(t *testing.T) {
	// Mock git diff
	origGitDiff := gitDiffFilesFunc
	gitDiffFilesFunc = func(r, baseline string) ([]string, error) {
		return []string{"file1.go", "file2.go"}, nil
	}
	defer func() { gitDiffFilesFunc = origGitDiff }()

	fileClaims := map[string][]ClaimInfo{
		"file1.go": {{Agent: "agent-1", Scope: "file"}},
		"file2.go": {{Agent: "agent-2", Scope: "file"}},
	}

	issues, status := checkClaimBeforeEdit("/tmp", fileClaims, true)

	if status != "pass" {
		t.Errorf("expected pass, got %s", status)
	}
	if len(issues) != 0 {
		t.Errorf("expected 0 issues, got %d", len(issues))
	}
}

func TestConflictDetection_ThreeAgentsSameFile(t *testing.T) {
	root := t.TempDir()

	manifest := `claims:
  - agent: agent-1
    files:
      - path: shared.go
        scope: file
    claimed_at: "2026-01-31T10:00:00Z"
  - agent: agent-2
    files:
      - path: shared.go
        scope: file
    claimed_at: "2026-01-31T10:05:00Z"
  - agent: agent-3
    files:
      - path: shared.go
        scope: file
    claimed_at: "2026-01-31T10:10:00Z"
`
	setupManifest(t, root, manifest)

	check := Check{
		ID: "test-conflict",
		Tracking: TrackingConfig{
			Manifest: ".dun/work-in-progress.yaml",
		},
		ConflictRules: []ConflictRule{
			{Type: "no-overlap", Scope: "file", Required: true},
		},
	}

	result, err := runConflictDetectionCheckFromSpec(root, check)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Status != "fail" {
		t.Errorf("expected fail, got %s", result.Status)
	}
	// Should report all 3 agents
	if len(result.Issues) < 1 {
		t.Errorf("expected at least 1 issue, got %d", len(result.Issues))
	}
	issue := result.Issues[0]
	if !strings.Contains(issue.Summary, "agent-1") {
		t.Errorf("expected issue to mention agent-1, got %s", issue.Summary)
	}
}

func TestConflictDetection_SameAgentMultipleFunctions(t *testing.T) {
	root := t.TempDir()

	manifest := `claims:
  - agent: agent-1
    files:
      - path: handler.go
        scope: function
        function: Foo
      - path: handler.go
        scope: function
        function: Bar
    claimed_at: "2026-01-31T10:00:00Z"
`
	setupManifest(t, root, manifest)

	check := Check{
		ID: "test-conflict",
		Tracking: TrackingConfig{
			Manifest: ".dun/work-in-progress.yaml",
		},
		ConflictRules: []ConflictRule{
			{Type: "no-overlap", Scope: "function", Required: true},
		},
	}

	result, err := runConflictDetectionCheckFromSpec(root, check)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Same agent claiming multiple functions is not a conflict
	if result.Status != "pass" {
		t.Errorf("expected pass (same agent), got %s", result.Status)
	}
}

func TestConflictDetection_NoRules(t *testing.T) {
	root := t.TempDir()

	manifest := `claims:
  - agent: agent-1
    files:
      - path: handler.go
        scope: file
    claimed_at: "2026-01-31T10:00:00Z"
`
	setupManifest(t, root, manifest)

	check := Check{
		ID: "test-conflict",
		Tracking: TrackingConfig{
			Manifest: ".dun/work-in-progress.yaml",
		},
		ConflictRules: nil, // No rules
	}

	result, err := runConflictDetectionCheckFromSpec(root, check)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// No rules = pass
	if result.Status != "pass" {
		t.Errorf("expected pass (no rules), got %s", result.Status)
	}
}

func TestConflictDetection_EmptyScopeTreatedAsFile(t *testing.T) {
	root := t.TempDir()

	manifest := `claims:
  - agent: agent-1
    files:
      - path: handler.go
    claimed_at: "2026-01-31T10:00:00Z"
  - agent: agent-2
    files:
      - path: handler.go
    claimed_at: "2026-01-31T10:05:00Z"
`
	setupManifest(t, root, manifest)

	check := Check{
		ID: "test-conflict",
		Tracking: TrackingConfig{
			Manifest: ".dun/work-in-progress.yaml",
		},
		ConflictRules: []ConflictRule{
			{Type: "no-overlap", Scope: "file", Required: true},
		},
	}

	result, err := runConflictDetectionCheckFromSpec(root, check)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Empty scope treated as file scope, should conflict
	if result.Status != "fail" {
		t.Errorf("expected fail (empty scope = file), got %s", result.Status)
	}
}

func TestLoadWIPManifest_CustomPath(t *testing.T) {
	root := t.TempDir()

	customDir := filepath.Join(root, "custom")
	if err := os.MkdirAll(customDir, 0755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(customDir, "claims.yaml"), `claims:
  - agent: agent-1
    files:
      - path: file.go
        scope: file
`)

	manifest, err := loadWIPManifest(root, "custom/claims.yaml")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(manifest.Claims) != 1 {
		t.Errorf("expected 1 claim, got %d", len(manifest.Claims))
	}
	if manifest.Claims[0].Agent != "agent-1" {
		t.Errorf("expected agent-1, got %s", manifest.Claims[0].Agent)
	}
}

func TestConflictDetection_MultipleFilesMultipleConflicts(t *testing.T) {
	root := t.TempDir()

	manifest := `claims:
  - agent: agent-1
    files:
      - path: file1.go
        scope: file
      - path: file2.go
        scope: file
    claimed_at: "2026-01-31T10:00:00Z"
  - agent: agent-2
    files:
      - path: file1.go
        scope: file
    claimed_at: "2026-01-31T10:05:00Z"
  - agent: agent-3
    files:
      - path: file2.go
        scope: file
    claimed_at: "2026-01-31T10:10:00Z"
`
	setupManifest(t, root, manifest)

	check := Check{
		ID: "test-conflict",
		Tracking: TrackingConfig{
			Manifest: ".dun/work-in-progress.yaml",
		},
		ConflictRules: []ConflictRule{
			{Type: "no-overlap", Scope: "file", Required: true},
		},
	}

	result, err := runConflictDetectionCheckFromSpec(root, check)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Status != "fail" {
		t.Errorf("expected fail, got %s", result.Status)
	}
	// Should have 2 conflicts (one per file)
	if len(result.Issues) != 2 {
		t.Errorf("expected 2 issues, got %d: %+v", len(result.Issues), result.Issues)
	}
}

func TestCheckFileOverlap_NoFileScopeClaims(t *testing.T) {
	// Only function-scope claims, no file-scope claims
	claims := []ClaimInfo{
		{Agent: "agent-1", Scope: "function", Function: "Foo"},
		{Agent: "agent-2", Scope: "function", Function: "Bar"},
	}
	issues := checkFileOverlap("file.go", claims)

	// No file-scope conflicts
	if len(issues) != 0 {
		t.Errorf("expected 0 issues for function-only claims, got %d", len(issues))
	}
}

func TestCheckFileOverlap_FileScopeConflictsWithFunction(t *testing.T) {
	claims := []ClaimInfo{
		{Agent: "agent-1", Scope: "file"},
		{Agent: "agent-2", Scope: "function", Function: "Foo"},
	}
	issues := checkFileOverlap("file.go", claims)

	if len(issues) != 1 {
		t.Errorf("expected 1 issue for file vs function conflict, got %d", len(issues))
	}
	if len(issues) > 0 && !strings.Contains(issues[0].Summary, "conflicting claims") {
		t.Errorf("expected 'conflicting claims' in summary, got %s", issues[0].Summary)
	}
}

func TestCheckFunctionOverlaps_MultipleFileScopeClaims(t *testing.T) {
	claims := []ClaimInfo{
		{Agent: "agent-1", Scope: "file"},
		{Agent: "agent-2", Scope: "file"},
	}
	issues := checkFunctionOverlaps("file.go", claims)

	if len(issues) != 1 {
		t.Errorf("expected 1 issue for multiple file-scope claims, got %d", len(issues))
	}
}

func TestCheckFunctionOverlaps_FileScopeConflictsWithFunction(t *testing.T) {
	claims := []ClaimInfo{
		{Agent: "agent-1", Scope: "file"},
		{Agent: "agent-2", Scope: "function", Function: "Foo"},
	}
	issues := checkFunctionOverlaps("file.go", claims)

	if len(issues) != 1 {
		t.Errorf("expected 1 issue for file vs function conflict, got %d", len(issues))
	}
	if len(issues) > 0 && !strings.Contains(issues[0].Summary, "claims file") {
		t.Errorf("expected 'claims file' in summary, got %s", issues[0].Summary)
	}
}

func TestCheckFunctionOverlaps_SameAgentFilePlusFunction(t *testing.T) {
	claims := []ClaimInfo{
		{Agent: "agent-1", Scope: "file"},
		{Agent: "agent-1", Scope: "function", Function: "Foo"},
	}
	issues := checkFunctionOverlaps("file.go", claims)

	// Same agent, no conflict
	if len(issues) != 0 {
		t.Errorf("expected 0 issues for same agent file+function, got %d", len(issues))
	}
}

func TestCheckFunctionOverlaps_EmptyFunctionName(t *testing.T) {
	claims := []ClaimInfo{
		{Agent: "agent-1", Scope: "function", Function: ""},
		{Agent: "agent-2", Scope: "function", Function: ""},
	}
	issues := checkFunctionOverlaps("file.go", claims)

	// Empty function names are not grouped together
	if len(issues) != 0 {
		t.Errorf("expected 0 issues for empty function names, got %d", len(issues))
	}
}

func TestConflictDetection_FunctionScopeWithFileConflict(t *testing.T) {
	root := t.TempDir()

	manifest := `claims:
  - agent: agent-1
    files:
      - path: internal/auth/handler.go
        scope: file
    claimed_at: "2026-01-31T10:00:00Z"
  - agent: agent-2
    files:
      - path: internal/auth/handler.go
        scope: file
    claimed_at: "2026-01-31T10:05:00Z"
  - agent: agent-3
    files:
      - path: internal/auth/handler.go
        scope: function
        function: HandleLogin
    claimed_at: "2026-01-31T10:10:00Z"
`
	setupManifest(t, root, manifest)

	check := Check{
		ID: "test-conflict",
		Tracking: TrackingConfig{
			Manifest: ".dun/work-in-progress.yaml",
		},
		ConflictRules: []ConflictRule{
			{Type: "no-overlap", Scope: "function", Required: true},
		},
	}

	result, err := runConflictDetectionCheckFromSpec(root, check)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Status != "fail" {
		t.Errorf("expected fail, got %s", result.Status)
	}
	// Should have multiple issues: file-file conflict and file-function conflicts
	if len(result.Issues) < 2 {
		t.Errorf("expected at least 2 issues, got %d: %+v", len(result.Issues), result.Issues)
	}
}

func TestConflictDetection_UnknownRuleType(t *testing.T) {
	root := t.TempDir()

	manifest := `claims:
  - agent: agent-1
    files:
      - path: file.go
        scope: file
`
	setupManifest(t, root, manifest)

	check := Check{
		ID: "test-conflict",
		Tracking: TrackingConfig{
			Manifest: ".dun/work-in-progress.yaml",
		},
		ConflictRules: []ConflictRule{
			{Type: "unknown-rule-type", Required: true},
		},
	}

	result, err := runConflictDetectionCheckFromSpec(root, check)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Unknown rules are skipped, should pass
	if result.Status != "pass" {
		t.Errorf("expected pass for unknown rule type, got %s", result.Status)
	}
}

func TestConflictDetection_IssuesHaveNextField(t *testing.T) {
	root := t.TempDir()

	manifest := `claims:
  - agent: agent-1
    files:
      - path: file.go
        scope: file
    claimed_at: "2026-01-31T10:00:00Z"
  - agent: agent-2
    files:
      - path: file.go
        scope: file
    claimed_at: "2026-01-31T10:05:00Z"
`
	setupManifest(t, root, manifest)

	check := Check{
		ID: "test-conflict",
		Tracking: TrackingConfig{
			Manifest: ".dun/work-in-progress.yaml",
		},
		ConflictRules: []ConflictRule{
			{Type: "no-overlap", Scope: "file", Required: true},
		},
	}

	result, err := runConflictDetectionCheckFromSpec(root, check)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Next == "" {
		t.Error("expected Next field to be set for conflicts")
	}
	if !strings.Contains(result.Next, "Resolve") {
		t.Errorf("expected 'Resolve' in next field, got %s", result.Next)
	}
}

// Helper function to setup manifest file
func setupManifest(t *testing.T, root, content string) {
	t.Helper()
	manifestDir := filepath.Join(root, ".dun")
	if err := os.MkdirAll(manifestDir, 0755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(manifestDir, "work-in-progress.yaml"), content)
}
