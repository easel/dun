package dun

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// ChangeCascadeConfig holds the configuration for a change-cascade check.
type ChangeCascadeConfig struct {
	CascadeRules []CascadeRule `yaml:"cascade_rules"`
	Trigger      string        `yaml:"trigger"`  // git-diff|always
	Baseline     string        `yaml:"baseline"` // default: HEAD~1
}

// CascadeRule defines an upstream file pattern and its downstream dependencies.
type CascadeRule struct {
	Upstream    string       `yaml:"upstream"`
	Downstreams []Downstream `yaml:"downstreams"`
}

// Downstream defines a downstream file that should be updated when upstream changes.
type Downstream struct {
	Path     string   `yaml:"path"`
	Sections []string `yaml:"sections"`
	Required bool     `yaml:"required"`
}

// gitDiffFunc allows mocking in tests.
var gitDiffFunc = gitDiffFiles

// getFileMtimeFunc allows mocking in tests.
var getFileMtimeFunc = getFileMtime

func runChangeCascadeCheck(root string, check Check) (CheckResult, error) {
	config := extractCascadeConfig(check)

	// Determine if we should run the check
	if config.Trigger == "git-diff" || config.Trigger == "" {
		// Only run if there are changes
		baseline := config.Baseline
		if baseline == "" {
			baseline = "HEAD~1"
		}

		changedFiles, err := gitDiffFunc(root, baseline)
		if err != nil {
			// If git diff fails (e.g., no commits), skip the check
			return CheckResult{
				ID:     check.ID,
				Status: "skip",
				Signal: "cannot determine changes",
				Detail: err.Error(),
			}, nil
		}

		if len(changedFiles) == 0 {
			return CheckResult{
				ID:     check.ID,
				Status: "pass",
				Signal: "no upstream changes detected",
			}, nil
		}

		return checkCascades(root, check.ID, config.CascadeRules, changedFiles)
	}

	// trigger: always - check all files by mtime
	return checkCascadesByMtime(root, check.ID, config.CascadeRules)
}

// extractCascadeConfig extracts cascade config from check fields.
func extractCascadeConfig(check Check) ChangeCascadeConfig {
	// Convert from Check's inline struct to our CascadeRule type
	var rules []CascadeRule
	for _, r := range check.CascadeRules {
		var downstreams []Downstream
		for _, d := range r.Downstreams {
			downstreams = append(downstreams, Downstream{
				Path:     d.Path,
				Sections: d.Sections,
				Required: d.Required,
			})
		}
		rules = append(rules, CascadeRule{
			Upstream:    r.Upstream,
			Downstreams: downstreams,
		})
	}

	return ChangeCascadeConfig{
		CascadeRules: rules,
		Trigger:      check.Trigger,
		Baseline:     check.Baseline,
	}
}

// gitDiffFiles returns files changed between baseline and HEAD.
func gitDiffFiles(root string, baseline string) ([]string, error) {
	cmd := exec.Command("git", "diff", "--name-only", baseline, "HEAD")
	cmd.Dir = root
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git diff: %w", err)
	}

	text := strings.TrimSpace(string(output))
	if text == "" {
		return nil, nil
	}

	return strings.Split(text, "\n"), nil
}

// checkCascades checks if changed upstream files have corresponding downstream updates.
func checkCascades(root, checkID string, rules []CascadeRule, changedFiles []string) (CheckResult, error) {
	changedSet := make(map[string]bool)
	for _, f := range changedFiles {
		changedSet[f] = true
	}

	var issues []Issue
	var requiredCount, warnCount int

	for _, rule := range rules {
		// Find changed files matching the upstream pattern
		upstreamMatches := matchPattern(changedFiles, rule.Upstream)
		if len(upstreamMatches) == 0 {
			continue
		}

		// For each matching upstream, check downstreams
		for _, upstream := range upstreamMatches {
			for _, ds := range rule.Downstreams {
				staleDownstreams := findStaleDownstreams(root, ds, changedSet)
				for _, stale := range staleDownstreams {
					issue := Issue{
						ID:      fmt.Sprintf("stale:%s->%s", upstream, stale),
						Summary: fmt.Sprintf("Downstream %s needs update after %s changed", stale, upstream),
						Path:    stale,
					}
					issues = append(issues, issue)
					if ds.Required {
						requiredCount++
					} else {
						warnCount++
					}
				}
			}
		}
	}

	if len(issues) == 0 {
		return CheckResult{
			ID:     checkID,
			Status: "pass",
			Signal: "all downstreams up to date",
		}, nil
	}

	status := "warn"
	if requiredCount > 0 {
		status = "fail"
	}

	return CheckResult{
		ID:     checkID,
		Status: status,
		Signal: fmt.Sprintf("%d downstream files need updates", len(issues)),
		Detail: fmt.Sprintf("%d required, %d optional", requiredCount, warnCount),
		Next:   "Update downstream files to reflect upstream changes",
		Issues: issues,
	}, nil
}

// checkCascadesByMtime checks all configured cascades by modification time.
func checkCascadesByMtime(root, checkID string, rules []CascadeRule) (CheckResult, error) {
	var issues []Issue
	var requiredCount, warnCount int

	for _, rule := range rules {
		// Find all files matching upstream pattern
		upstreamFiles, err := globFiles(root, rule.Upstream)
		if err != nil {
			continue
		}

		for _, upstream := range upstreamFiles {
			upstreamMtime, err := getFileMtimeFunc(filepath.Join(root, upstream))
			if err != nil {
				continue
			}

			for _, ds := range rule.Downstreams {
				downstreamFiles, err := globFiles(root, ds.Path)
				if err != nil {
					continue
				}

				for _, downstream := range downstreamFiles {
					downstreamMtime, err := getFileMtimeFunc(filepath.Join(root, downstream))
					if err != nil {
						continue
					}

					if upstreamMtime.After(downstreamMtime) {
						issue := Issue{
							ID:      fmt.Sprintf("stale:%s->%s", upstream, downstream),
							Summary: fmt.Sprintf("Downstream %s is stale (upstream %s is newer)", downstream, upstream),
							Path:    downstream,
						}
						issues = append(issues, issue)
						if ds.Required {
							requiredCount++
						} else {
							warnCount++
						}
					}
				}
			}
		}
	}

	if len(issues) == 0 {
		return CheckResult{
			ID:     checkID,
			Status: "pass",
			Signal: "all downstreams up to date",
		}, nil
	}

	status := "warn"
	if requiredCount > 0 {
		status = "fail"
	}

	return CheckResult{
		ID:     checkID,
		Status: status,
		Signal: fmt.Sprintf("%d downstream files need updates", len(issues)),
		Detail: fmt.Sprintf("%d required, %d optional", requiredCount, warnCount),
		Next:   "Update downstream files to reflect upstream changes",
		Issues: issues,
	}, nil
}

// matchPattern returns files matching the given glob pattern.
func matchPattern(files []string, pattern string) []string {
	var matches []string
	for _, f := range files {
		matched, err := filepath.Match(pattern, f)
		if err != nil {
			continue
		}
		if matched {
			matches = append(matches, f)
		}
		// Also try matching the basename for patterns like "*.md"
		if !matched && !strings.Contains(pattern, "/") {
			matched, _ = filepath.Match(pattern, filepath.Base(f))
			if matched {
				matches = append(matches, f)
			}
		}
	}
	return matches
}

// findStaleDownstreams finds downstream files that were not updated.
func findStaleDownstreams(root string, ds Downstream, changedSet map[string]bool) []string {
	// Get all files matching the downstream pattern
	downstreamFiles, err := globFiles(root, ds.Path)
	if err != nil {
		return nil
	}

	var stale []string
	for _, f := range downstreamFiles {
		if !changedSet[f] {
			stale = append(stale, f)
		}
	}
	return stale
}

// globFiles returns files matching a glob pattern relative to root.
func globFiles(root, pattern string) ([]string, error) {
	fullPattern := filepath.Join(root, pattern)
	matches, err := filepath.Glob(fullPattern)
	if err != nil {
		return nil, err
	}

	var relative []string
	for _, m := range matches {
		rel, err := filepath.Rel(root, m)
		if err != nil {
			rel = m
		}
		relative = append(relative, filepath.ToSlash(rel))
	}
	return relative, nil
}

// getFileMtime returns the modification time of a file.
func getFileMtime(path string) (time.Time, error) {
	info, err := os.Stat(path)
	if err != nil {
		return time.Time{}, err
	}
	return info.ModTime(), nil
}
