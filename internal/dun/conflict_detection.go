package dun

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// ConflictDetectionConfig holds the configuration for a conflict-detection check.
type ConflictDetectionConfig struct {
	Tracking TrackingConfig `yaml:"tracking"`
	Rules    []ConflictRule `yaml:"rules"`
}

// TrackingConfig specifies where to find claim information.
type TrackingConfig struct {
	Manifest     string `yaml:"manifest"`      // Path to WIP manifest
	ClaimPattern string `yaml:"claim_pattern"` // Pattern in code marking claimed sections
}

// ConflictRule defines how conflicts are detected.
type ConflictRule struct {
	Type     string `yaml:"type"`     // no-overlap, claim-before-edit
	Scope    string `yaml:"scope"`    // file, function, line
	Required bool   `yaml:"required"` // If false, warn only
}

// WIPManifest represents the work-in-progress manifest file.
type WIPManifest struct {
	Claims []Claim `yaml:"claims"`
}

// Claim represents a single agent's claim on files.
type Claim struct {
	Agent     string      `yaml:"agent"`
	Files     []FileClaim `yaml:"files"`
	ClaimedAt time.Time   `yaml:"claimed_at"`
}

// FileClaim represents a claim on a specific file or function.
type FileClaim struct {
	Path     string `yaml:"path"`
	Scope    string `yaml:"scope"`    // file, function
	Function string `yaml:"function"` // if scope=function
}

// ClaimInfo stores parsed claim information for lookup.
type ClaimInfo struct {
	Agent     string
	Scope     string
	Function  string
	ClaimedAt time.Time
}

// gitDiffFilesFunc allows mocking git diff in tests.
var gitDiffFilesFunc = gitDiffFiles

// readFileFunc allows mocking file reads in tests.
var readFileFunc = os.ReadFile

func runConflictDetectionCheck(root string, check Check) (CheckResult, error) {
	config := extractConflictDetectionConfig(check)

	// Load WIP manifest
	manifest, err := loadWIPManifest(root, config.Tracking.Manifest)
	if err != nil {
		// If manifest doesn't exist, no claims = pass
		if os.IsNotExist(err) {
			return CheckResult{
				ID:     check.ID,
				Status: "pass",
				Signal: "no WIP manifest found",
				Detail: "No claims to conflict",
			}, nil
		}
		return CheckResult{
			ID:     check.ID,
			Status: "fail",
			Signal: "failed to load WIP manifest",
			Detail: err.Error(),
		}, nil
	}

	// Build file->claims mapping
	fileClaims := buildFileClaimsMap(manifest)

	// Check for conflicts based on rules
	var issues []Issue
	status := "pass"

	for _, rule := range config.Rules {
		var ruleIssues []Issue
		var ruleStatus string

		switch rule.Type {
		case "no-overlap":
			ruleIssues, ruleStatus = checkNoOverlap(fileClaims, rule.Scope, rule.Required)
		case "claim-before-edit":
			ruleIssues, ruleStatus = checkClaimBeforeEdit(root, fileClaims, rule.Required)
		}

		issues = append(issues, ruleIssues...)
		// Update overall status (fail > warn > pass)
		if ruleStatus == "fail" {
			status = "fail"
		} else if ruleStatus == "warn" && status != "fail" {
			status = "warn"
		}
	}

	if len(issues) == 0 {
		return CheckResult{
			ID:     check.ID,
			Status: "pass",
			Signal: "no conflicts detected",
		}, nil
	}

	return CheckResult{
		ID:     check.ID,
		Status: status,
		Signal: fmt.Sprintf("%d conflict(s) detected", len(issues)),
		Issues: issues,
		Next:   "Resolve claim conflicts before proceeding",
	}, nil
}

// extractConflictDetectionConfig extracts conflict detection config from check fields.
func extractConflictDetectionConfig(check Check) ConflictDetectionConfig {
	var rules []ConflictRule
	for _, r := range check.ConflictRules {
		rules = append(rules, ConflictRule{
			Type:     r.Type,
			Scope:    r.Scope,
			Required: r.Required,
		})
	}

	return ConflictDetectionConfig{
		Tracking: TrackingConfig{
			Manifest:     check.Tracking.Manifest,
			ClaimPattern: check.Tracking.ClaimPattern,
		},
		Rules: rules,
	}
}

// loadWIPManifest loads the WIP manifest from the given path.
func loadWIPManifest(root, manifestPath string) (*WIPManifest, error) {
	if manifestPath == "" {
		manifestPath = ".dun/work-in-progress.yaml"
	}

	fullPath := filepath.Join(root, manifestPath)
	data, err := readFileFunc(fullPath)
	if err != nil {
		return nil, err
	}

	var manifest WIPManifest
	if err := yaml.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("parsing manifest: %w", err)
	}

	return &manifest, nil
}

// buildFileClaimsMap creates a mapping from file paths to their claims.
func buildFileClaimsMap(manifest *WIPManifest) map[string][]ClaimInfo {
	fileClaims := make(map[string][]ClaimInfo)

	for _, claim := range manifest.Claims {
		for _, fc := range claim.Files {
			info := ClaimInfo{
				Agent:     claim.Agent,
				Scope:     fc.Scope,
				Function:  fc.Function,
				ClaimedAt: claim.ClaimedAt,
			}
			fileClaims[fc.Path] = append(fileClaims[fc.Path], info)
		}
	}

	return fileClaims
}

// checkNoOverlap detects overlapping claims on the same file or function.
func checkNoOverlap(fileClaims map[string][]ClaimInfo, scope string, required bool) ([]Issue, string) {
	var issues []Issue

	for path, claims := range fileClaims {
		if len(claims) <= 1 {
			continue
		}

		// Group by function if scope=function
		if scope == "function" {
			issues = append(issues, checkFunctionOverlaps(path, claims)...)
		} else {
			// scope=file or default: any multiple claims on same file is conflict
			issues = append(issues, checkFileOverlap(path, claims)...)
		}
	}

	if len(issues) == 0 {
		return nil, "pass"
	}

	if required {
		return issues, "fail"
	}
	return issues, "warn"
}

// checkFileOverlap checks for overlapping file-level claims.
func checkFileOverlap(path string, claims []ClaimInfo) []Issue {
	// Check for actual conflicts:
	// - Multiple file-scope claims conflict
	// - File-scope + function-scope on same file conflict
	var fileScopeClaims []ClaimInfo
	for _, c := range claims {
		if c.Scope == "file" || c.Scope == "" {
			fileScopeClaims = append(fileScopeClaims, c)
		}
	}

	// Multiple file-scope claims = conflict
	if len(fileScopeClaims) > 1 {
		agents := make([]string, 0, len(fileScopeClaims))
		for _, c := range fileScopeClaims {
			agents = append(agents, c.Agent)
		}
		return []Issue{{
			ID:      fmt.Sprintf("overlap:%s", path),
			Summary: fmt.Sprintf("File %s claimed by multiple agents: %s", path, strings.Join(agents, ", ")),
			Path:    path,
		}}
	}

	// If there's a file-scope claim and function-scope claims, that's a conflict
	if len(fileScopeClaims) == 1 {
		for _, c := range claims {
			if c.Scope == "function" && c.Agent != fileScopeClaims[0].Agent {
				return []Issue{{
					ID:      fmt.Sprintf("overlap:%s", path),
					Summary: fmt.Sprintf("File %s has conflicting claims: %s (file) vs %s (function %s)", path, fileScopeClaims[0].Agent, c.Agent, c.Function),
					Path:    path,
				}}
			}
		}
	}

	return nil
}

// checkFunctionOverlaps checks for overlapping function-level claims.
func checkFunctionOverlaps(path string, claims []ClaimInfo) []Issue {
	var issues []Issue

	// Group claims by function
	functionClaims := make(map[string][]ClaimInfo)
	var fileScopeClaims []ClaimInfo

	for _, c := range claims {
		if c.Scope == "function" && c.Function != "" {
			functionClaims[c.Function] = append(functionClaims[c.Function], c)
		} else if c.Scope == "file" || c.Scope == "" {
			fileScopeClaims = append(fileScopeClaims, c)
		}
	}

	// Check for same function claimed by multiple agents
	for fn, fnClaims := range functionClaims {
		if len(fnClaims) > 1 {
			agents := make([]string, 0, len(fnClaims))
			for _, c := range fnClaims {
				agents = append(agents, c.Agent)
			}
			issues = append(issues, Issue{
				ID:      fmt.Sprintf("overlap:%s:%s", path, fn),
				Summary: fmt.Sprintf("Function %s in %s claimed by multiple agents: %s", fn, path, strings.Join(agents, ", ")),
				Path:    path,
			})
		}
	}

	// Check if file-scope claim conflicts with function-scope claims
	if len(fileScopeClaims) > 0 {
		for fn, fnClaims := range functionClaims {
			for _, fnClaim := range fnClaims {
				for _, fileClaim := range fileScopeClaims {
					if fnClaim.Agent != fileClaim.Agent {
						issues = append(issues, Issue{
							ID:      fmt.Sprintf("overlap:%s:%s", path, fn),
							Summary: fmt.Sprintf("Conflict on %s: %s claims file, %s claims function %s", path, fileClaim.Agent, fnClaim.Agent, fn),
							Path:    path,
						})
					}
				}
			}
		}
	}

	// Multiple file-scope claims
	if len(fileScopeClaims) > 1 {
		agents := make([]string, 0, len(fileScopeClaims))
		for _, c := range fileScopeClaims {
			agents = append(agents, c.Agent)
		}
		issues = append(issues, Issue{
			ID:      fmt.Sprintf("overlap:%s", path),
			Summary: fmt.Sprintf("File %s claimed by multiple agents: %s", path, strings.Join(agents, ", ")),
			Path:    path,
		})
	}

	return issues
}

// checkClaimBeforeEdit verifies all modified files have claims.
func checkClaimBeforeEdit(root string, fileClaims map[string][]ClaimInfo, required bool) ([]Issue, string) {
	// Get files changed since HEAD~1
	changedFiles, err := gitDiffFilesFunc(root, "HEAD~1")
	if err != nil {
		// If we can't get git diff, skip this check
		return nil, "pass"
	}

	var issues []Issue
	for _, file := range changedFiles {
		if _, hasClaim := fileClaims[file]; !hasClaim {
			issues = append(issues, Issue{
				ID:      fmt.Sprintf("unclaimed:%s", file),
				Summary: fmt.Sprintf("File %s modified without claim", file),
				Path:    file,
			})
		}
	}

	if len(issues) == 0 {
		return nil, "pass"
	}

	if required {
		return issues, "fail"
	}
	return issues, "warn"
}
