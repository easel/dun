package dun

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// SpecInfo holds extracted information about a specification.
type SpecInfo struct {
	ID       string
	Path     string
	CodeRefs []string // Code files referenced in implementation section
}

// CodeInfo holds extracted spec references from a code file.
type CodeInfo struct {
	Path    string
	SpecIDs []string
}

// runSpecBindingCheck verifies bidirectional references between specifications and code.
func runSpecBindingCheck(root string, def CheckDefinition, config SpecBindingConfig) (CheckResult, error) {
	// Extract spec information
	specMap, err := extractSpecs(root, config.Bindings.Specs)
	if err != nil {
		return CheckResult{
			ID:     def.ID,
			Status: "fail",
			Signal: "failed to extract specs",
			Detail: err.Error(),
		}, nil
	}

	// Extract code references
	codeMap, err := extractCodeRefs(root, config.Bindings.Code)
	if err != nil {
		return CheckResult{
			ID:     def.ID,
			Status: "fail",
			Signal: "failed to extract code refs",
			Detail: err.Error(),
		}, nil
	}

	// Build bidirectional mappings
	specsToCode := buildSpecsToCode(specMap, codeMap)
	codeToSpecs := buildCodeToSpecs(codeMap)

	// Calculate coverage
	specsWithCode := 0
	for _, codeFiles := range specsToCode {
		if len(codeFiles) > 0 {
			specsWithCode++
		}
	}
	totalSpecs := len(specMap)
	coverage := 0.0
	if totalSpecs > 0 {
		coverage = float64(specsWithCode) / float64(totalSpecs)
	}

	// Apply rules and collect issues
	var issues []Issue
	status := "pass"

	for _, rule := range config.BindingRules {
		ruleIssues, ruleStatus := applyBindingRule(rule, specMap, codeMap, specsToCode, codeToSpecs, coverage)
		issues = append(issues, ruleIssues...)

		// Update overall status (fail > warn > pass)
		if ruleStatus == "fail" {
			status = "fail"
		} else if ruleStatus == "warn" && status != "fail" {
			status = "warn"
		}
	}

	// Build signal and detail
	signal := fmt.Sprintf("Spec coverage: %.0f%%", coverage*100)
	detail := fmt.Sprintf("%d/%d specs have implementations", specsWithCode, totalSpecs)

	return CheckResult{
		ID:     def.ID,
		Status: status,
		Signal: signal,
		Detail: detail,
		Issues: issues,
	}, nil
}

// extractSpecs finds all spec files and extracts their IDs and implementation references.
func extractSpecs(root string, specPatterns []SpecBinding) (map[string]SpecInfo, error) {
	specMap := make(map[string]SpecInfo)

	for _, sp := range specPatterns {
		files, err := findFiles(root, sp.Pattern)
		if err != nil {
			return nil, fmt.Errorf("finding spec files for pattern %q: %w", sp.Pattern, err)
		}

		idRe, err := compileIDPattern(sp.IDPattern)
		if err != nil {
			return nil, fmt.Errorf("compiling id_pattern %q: %w", sp.IDPattern, err)
		}

		for _, file := range files {
			content, err := os.ReadFile(file)
			if err != nil {
				continue // Skip unreadable files
			}

			ids := extractSpecIDs(file, string(content), idRe)
			codeRefs := extractImplementationRefs(string(content), sp.ImplementationSection)

			for _, id := range ids {
				specMap[id] = SpecInfo{
					ID:       id,
					Path:     file,
					CodeRefs: codeRefs,
				}
			}
		}
	}

	return specMap, nil
}

// extractCodeRefs finds all code files and extracts spec references from them.
func extractCodeRefs(root string, codePatterns []CodeBinding) (map[string]CodeInfo, error) {
	codeMap := make(map[string]CodeInfo)

	for _, cp := range codePatterns {
		files, err := findFiles(root, cp.Pattern)
		if err != nil {
			return nil, fmt.Errorf("finding code files for pattern %q: %w", cp.Pattern, err)
		}

		for _, file := range files {
			content, err := os.ReadFile(file)
			if err != nil {
				continue // Skip unreadable files
			}

			specIDs := extractSpecRefsFromCode(string(content), cp.SpecComment)

			if existing, ok := codeMap[file]; ok {
				// Merge spec IDs if file matched multiple patterns
				existing.SpecIDs = mergeUnique(existing.SpecIDs, specIDs)
				codeMap[file] = existing
			} else {
				codeMap[file] = CodeInfo{
					Path:    file,
					SpecIDs: specIDs,
				}
			}
		}
	}

	return codeMap, nil
}

// findFiles finds all files matching a glob pattern relative to root.
func findFiles(root, pattern string) ([]string, error) {
	fullPattern := filepath.Join(root, pattern)
	matches, err := filepath.Glob(fullPattern)
	if err != nil {
		return nil, err
	}
	return matches, nil
}

// compileIDPattern compiles the ID pattern regex.
func compileIDPattern(pattern string) (*regexp.Regexp, error) {
	if pattern == "" {
		return nil, nil
	}
	return regexp.Compile(pattern)
}

// extractSpecIDs extracts spec IDs from content using the ID pattern.
// If no pattern is provided, tries to extract from filename.
func extractSpecIDs(filePath, content string, idRe *regexp.Regexp) []string {
	if idRe == nil {
		// Try to extract from filename
		base := filepath.Base(filePath)
		ext := filepath.Ext(base)
		name := strings.TrimSuffix(base, ext)
		if name != "" {
			return []string{name}
		}
		return nil
	}

	// First try filename
	base := filepath.Base(filePath)
	if matches := idRe.FindAllString(base, -1); len(matches) > 0 {
		return matches
	}

	// Then try content
	matches := idRe.FindAllString(content, -1)
	return uniqueStrings(matches)
}

// extractImplementationRefs extracts code file references from the implementation section.
func extractImplementationRefs(content, sectionHeader string) []string {
	if sectionHeader == "" {
		return nil
	}

	// Find the section
	idx := strings.Index(content, sectionHeader)
	if idx == -1 {
		return nil
	}

	// Get content after section header until next section or end
	sectionContent := content[idx+len(sectionHeader):]
	nextSection := strings.Index(sectionContent, "\n## ")
	if nextSection > 0 {
		sectionContent = sectionContent[:nextSection]
	}

	// Extract file paths (looking for common patterns)
	var refs []string

	// Match backtick-wrapped paths like `internal/foo/bar.go`
	pathRe := regexp.MustCompile("`([^`]+\\.[a-zA-Z]+)`")
	matches := pathRe.FindAllStringSubmatch(sectionContent, -1)
	for _, m := range matches {
		if len(m) > 1 {
			refs = append(refs, m[1])
		}
	}

	// Match paths in markdown links [text](path)
	linkRe := regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`)
	linkMatches := linkRe.FindAllStringSubmatch(sectionContent, -1)
	for _, m := range linkMatches {
		if len(m) > 2 && looksLikeCodePath(m[2]) {
			refs = append(refs, m[2])
		}
	}

	return uniqueStrings(refs)
}

// extractSpecRefsFromCode extracts spec IDs from code comments.
func extractSpecRefsFromCode(content, specComment string) []string {
	if specComment == "" {
		return nil
	}

	var specIDs []string

	// Build regex to find spec references after the comment pattern
	// e.g., "// Implements: FEAT-" should match "// Implements: FEAT-001"
	escapedComment := regexp.QuoteMeta(specComment)
	pattern := escapedComment + `([A-Za-z0-9_-]+)`
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil
	}

	matches := re.FindAllStringSubmatch(content, -1)
	for _, m := range matches {
		if len(m) > 1 {
			// Reconstruct the full ID (comment prefix + captured suffix)
			// Extract the ID part from the spec_comment
			parts := strings.Split(specComment, ":")
			var prefix string
			if len(parts) > 1 {
				prefix = strings.TrimSpace(parts[len(parts)-1])
			}
			fullID := prefix + m[1]
			// Clean up the prefix
			fullID = strings.TrimSpace(fullID)
			if fullID != "" {
				specIDs = append(specIDs, fullID)
			}
		}
	}

	return uniqueStrings(specIDs)
}

// buildSpecsToCode creates a mapping from spec IDs to code files that reference them.
func buildSpecsToCode(specMap map[string]SpecInfo, codeMap map[string]CodeInfo) map[string][]string {
	result := make(map[string][]string)

	// Initialize all specs with empty lists
	for specID := range specMap {
		result[specID] = nil
	}

	// Find code files that reference each spec
	for codePath, codeInfo := range codeMap {
		for _, specID := range codeInfo.SpecIDs {
			if _, exists := specMap[specID]; exists {
				result[specID] = append(result[specID], codePath)
			}
		}
	}

	return result
}

// buildCodeToSpecs creates a mapping from code files to spec IDs they reference.
func buildCodeToSpecs(codeMap map[string]CodeInfo) map[string][]string {
	result := make(map[string][]string)
	for codePath, codeInfo := range codeMap {
		result[codePath] = codeInfo.SpecIDs
	}
	return result
}

// applyBindingRule applies a single binding rule and returns issues and status.
func applyBindingRule(rule BindingRule, specMap map[string]SpecInfo, codeMap map[string]CodeInfo,
	specsToCode map[string][]string, codeToSpecs map[string][]string, coverage float64) ([]Issue, string) {

	var issues []Issue
	status := "pass"

	switch rule.Type {
	case "bidirectional-coverage":
		if coverage < rule.MinCoverage {
			issues = append(issues, Issue{
				ID:      "coverage-below-threshold",
				Summary: fmt.Sprintf("Coverage %.0f%% is below minimum %.0f%%", coverage*100, rule.MinCoverage*100),
			})
			if rule.WarnOnly {
				status = "warn"
			} else {
				status = "fail"
			}
		}

	case "no-orphan-specs":
		for specID, codeFiles := range specsToCode {
			if len(codeFiles) == 0 {
				spec := specMap[specID]
				issues = append(issues, Issue{
					ID:      "orphan-spec",
					Path:    spec.Path,
					Summary: fmt.Sprintf("No implementation found for spec %s", specID),
				})
			}
		}
		if len(issues) > 0 {
			if rule.WarnOnly {
				status = "warn"
			} else {
				status = "fail"
			}
		}

	case "no-orphan-code":
		for codePath, specIDs := range codeToSpecs {
			if len(specIDs) == 0 {
				issues = append(issues, Issue{
					ID:      "orphan-code",
					Path:    codePath,
					Summary: "No spec reference found",
				})
			}
		}
		if len(issues) > 0 {
			if rule.WarnOnly {
				status = "warn"
			} else {
				status = "fail"
			}
		}
	}

	return issues, status
}

// looksLikeCodePath checks if a string looks like a code file path.
func looksLikeCodePath(s string) bool {
	codeExts := []string{".go", ".ts", ".js", ".py", ".rs", ".java", ".c", ".cpp", ".h"}
	for _, ext := range codeExts {
		if strings.HasSuffix(s, ext) {
			return true
		}
	}
	return false
}

// uniqueStrings returns a slice with duplicates removed.
func uniqueStrings(strs []string) []string {
	seen := make(map[string]bool)
	var result []string
	for _, s := range strs {
		if s != "" && !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}
	return result
}

// mergeUnique merges two string slices and removes duplicates.
func mergeUnique(a, b []string) []string {
	return uniqueStrings(append(a, b...))
}
