package dun

import (
	"encoding/json"
	"regexp"
	"sort"
	"strings"
)

// ComparisonResult holds the outcome of comparing two responses.
type ComparisonResult struct {
	Match      bool    `json:"match"`
	Confidence float64 `json:"confidence"`
	Level      string  `json:"level"` // "exact", "structural", "semantic"
	Diff       string  `json:"diff,omitempty"`
}

// SimilarityScore holds a similarity score between 0 and 1.
type SimilarityScore struct {
	Score float64 `json:"score"`
}

// ResponseGroup represents harnesses that agree on a response.
type ResponseGroup struct {
	Canonical  string          `json:"canonical"`
	Members    []HarnessResult `json:"members"`
	Confidence float64         `json:"confidence"`
}

// ResponseNormalizer normalizes responses for comparison.
type ResponseNormalizer struct {
	StripWhitespace   bool
	NormalizeLineEnds bool
	SortJSONKeys      bool
	IgnoreComments    bool
}

// DefaultNormalizer returns a normalizer with sensible defaults.
func DefaultNormalizer() ResponseNormalizer {
	return ResponseNormalizer{
		StripWhitespace:   true,
		NormalizeLineEnds: true,
		SortJSONKeys:      true,
		IgnoreComments:    true,
	}
}

// Normalize applies all enabled normalizations to a string.
func (rn *ResponseNormalizer) Normalize(s string) string {
	result := s

	// Normalize line endings (\r\n -> \n)
	if rn.NormalizeLineEnds {
		result = strings.ReplaceAll(result, "\r\n", "\n")
		result = strings.ReplaceAll(result, "\r", "\n")
	}

	// Strip comments (// and /* */)
	if rn.IgnoreComments {
		result = stripComments(result)
	}

	// Collapse whitespace
	if rn.StripWhitespace {
		result = collapseWhitespace(result)
	}

	// Sort JSON keys for deterministic comparison
	if rn.SortJSONKeys && looksLikeJSON(result) {
		if sorted, err := sortJSONKeys(result); err == nil {
			result = sorted
		}
	}

	return strings.TrimSpace(result)
}

// stripComments removes // and /* */ style comments from text.
func stripComments(s string) string {
	// Remove multi-line comments first
	multiLine := regexp.MustCompile(`/\*[\s\S]*?\*/`)
	result := multiLine.ReplaceAllString(s, "")

	// Remove single-line comments
	singleLine := regexp.MustCompile(`//[^\n]*`)
	result = singleLine.ReplaceAllString(result, "")

	return result
}

// collapseWhitespace normalizes whitespace by collapsing multiple spaces/tabs
// to single spaces and trimming lines.
func collapseWhitespace(s string) string {
	lines := strings.Split(s, "\n")
	var result []string

	spaceRe := regexp.MustCompile(`[ \t]+`)

	for _, line := range lines {
		// Collapse multiple spaces/tabs to single space
		line = spaceRe.ReplaceAllString(line, " ")
		line = strings.TrimSpace(line)
		if line != "" {
			result = append(result, line)
		}
	}

	return strings.Join(result, "\n")
}

// looksLikeJSON returns true if the string appears to be JSON.
func looksLikeJSON(s string) bool {
	s = strings.TrimSpace(s)
	if len(s) < 2 {
		return false
	}
	return (s[0] == '{' && s[len(s)-1] == '}') ||
		(s[0] == '[' && s[len(s)-1] == ']')
}

// sortJSONKeys parses JSON and re-serializes with sorted keys.
func sortJSONKeys(s string) (string, error) {
	var data interface{}
	if err := json.Unmarshal([]byte(s), &data); err != nil {
		return s, err
	}

	sorted, err := json.Marshal(sortMapKeys(data))
	if err != nil {
		return s, err
	}

	return string(sorted), nil
}

// sortMapKeys recursively sorts map keys in an interface{} value.
func sortMapKeys(v interface{}) interface{} {
	switch val := v.(type) {
	case map[string]interface{}:
		result := make(map[string]interface{})
		for k, v := range val {
			result[k] = sortMapKeys(v)
		}
		return result
	case []interface{}:
		result := make([]interface{}, len(val))
		for i, item := range val {
			result[i] = sortMapKeys(item)
		}
		return result
	default:
		return v
	}
}

// SemanticComparator compares responses with multiple levels of similarity.
type SemanticComparator struct {
	Normalizer ResponseNormalizer
	Threshold  float64 // Default: 0.95
}

// DefaultComparator returns a comparator with sensible defaults.
func DefaultComparator() *SemanticComparator {
	return &SemanticComparator{
		Normalizer: DefaultNormalizer(),
		Threshold:  0.95,
	}
}

// NewSemanticComparator creates a new comparator with the given threshold.
func NewSemanticComparator(threshold float64) *SemanticComparator {
	return &SemanticComparator{
		Normalizer: DefaultNormalizer(),
		Threshold:  threshold,
	}
}

// Compare compares two responses and returns a ComparisonResult.
func (sc *SemanticComparator) Compare(a, b string) ComparisonResult {
	// Level 1: Exact match after normalization
	normA := sc.Normalizer.Normalize(a)
	normB := sc.Normalizer.Normalize(b)

	if normA == normB {
		return ComparisonResult{
			Match:      true,
			Confidence: 1.0,
			Level:      "exact",
		}
	}

	// Level 2: Structural similarity (line-based Levenshtein)
	structural := sc.structuralCompare(normA, normB)
	if structural.Score >= sc.Threshold {
		return ComparisonResult{
			Match:      true,
			Confidence: structural.Score,
			Level:      "structural",
		}
	}

	// Level 3: Semantic similarity threshold check
	semantic := sc.semanticCompare(normA, normB)
	if semantic.Score >= sc.Threshold {
		return ComparisonResult{
			Match:      true,
			Confidence: semantic.Score,
			Level:      "semantic",
		}
	}

	// No match - compute diff
	return ComparisonResult{
		Match:      false,
		Confidence: max(structural.Score, semantic.Score),
		Diff:       computeDiff(normA, normB),
	}
}

// Normalize normalizes a response using the comparator's normalizer.
func (sc *SemanticComparator) Normalize(s string) string {
	return sc.Normalizer.Normalize(s)
}

// structuralCompare computes structural similarity using line-based Levenshtein.
func (sc *SemanticComparator) structuralCompare(a, b string) SimilarityScore {
	linesA := significantLines(a)
	linesB := significantLines(b)

	if len(linesA) == 0 && len(linesB) == 0 {
		return SimilarityScore{Score: 1.0}
	}

	distance := lineLevenshtein(linesA, linesB)
	maxLen := max(len(linesA), len(linesB))

	if maxLen == 0 {
		return SimilarityScore{Score: 1.0}
	}

	score := 1.0 - float64(distance)/float64(maxLen)
	return SimilarityScore{Score: score}
}

// semanticCompare computes semantic similarity using character-level Levenshtein
// as a fallback. In the future, this could use LLM-based similarity.
func (sc *SemanticComparator) semanticCompare(a, b string) SimilarityScore {
	if len(a) == 0 && len(b) == 0 {
		return SimilarityScore{Score: 1.0}
	}

	// Use character-level Levenshtein for now
	distance := levenshtein(a, b)
	maxLen := max(len(a), len(b))

	if maxLen == 0 {
		return SimilarityScore{Score: 1.0}
	}

	score := 1.0 - float64(distance)/float64(maxLen)
	return SimilarityScore{Score: score}
}

// significantLines splits a string into non-empty lines.
func significantLines(s string) []string {
	lines := strings.Split(s, "\n")
	var result []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			result = append(result, line)
		}
	}
	return result
}

// lineLevenshtein computes the Levenshtein distance at the line level.
func lineLevenshtein(a, b []string) int {
	if len(a) == 0 {
		return len(b)
	}
	if len(b) == 0 {
		return len(a)
	}

	// Create two rows for the DP table
	prev := make([]int, len(b)+1)
	curr := make([]int, len(b)+1)

	// Initialize first row
	for j := range prev {
		prev[j] = j
	}

	for i := 1; i <= len(a); i++ {
		curr[0] = i
		for j := 1; j <= len(b); j++ {
			cost := 0
			if a[i-1] != b[j-1] {
				cost = 1
			}
			curr[j] = min(
				prev[j]+1,      // deletion
				curr[j-1]+1,    // insertion
				prev[j-1]+cost, // substitution
			)
		}
		prev, curr = curr, prev
	}

	return prev[len(b)]
}

// levenshtein computes the Levenshtein distance between two strings.
func levenshtein(a, b string) int {
	if len(a) == 0 {
		return len(b)
	}
	if len(b) == 0 {
		return len(a)
	}

	// Convert to runes for proper Unicode handling
	runesA := []rune(a)
	runesB := []rune(b)

	// Create two rows for the DP table
	prev := make([]int, len(runesB)+1)
	curr := make([]int, len(runesB)+1)

	// Initialize first row
	for j := range prev {
		prev[j] = j
	}

	for i := 1; i <= len(runesA); i++ {
		curr[0] = i
		for j := 1; j <= len(runesB); j++ {
			cost := 0
			if runesA[i-1] != runesB[j-1] {
				cost = 1
			}
			curr[j] = min(
				prev[j]+1,      // deletion
				curr[j-1]+1,    // insertion
				prev[j-1]+cost, // substitution
			)
		}
		prev, curr = curr, prev
	}

	return prev[len(runesB)]
}

// computeDiff computes a simple unified diff between two strings.
func computeDiff(a, b string) string {
	linesA := strings.Split(a, "\n")
	linesB := strings.Split(b, "\n")

	var diff strings.Builder
	diff.WriteString("--- a\n")
	diff.WriteString("+++ b\n")

	// Simple line-by-line diff
	maxLines := max(len(linesA), len(linesB))
	for i := 0; i < maxLines; i++ {
		var lineA, lineB string
		if i < len(linesA) {
			lineA = linesA[i]
		}
		if i < len(linesB) {
			lineB = linesB[i]
		}

		if lineA == lineB {
			diff.WriteString(" " + lineA + "\n")
		} else {
			if i < len(linesA) {
				diff.WriteString("-" + lineA + "\n")
			}
			if i < len(linesB) {
				diff.WriteString("+" + lineB + "\n")
			}
		}
	}

	return diff.String()
}

// GroupByAgreement groups harness results by semantic similarity.
// Returns groups sorted by size (largest first).
func GroupByAgreement(results []HarnessResult, comparator *SemanticComparator) []ResponseGroup {
	if comparator == nil {
		comparator = DefaultComparator()
	}

	var groups []*ResponseGroup

	for _, r := range results {
		if r.Error != nil {
			continue
		}

		var assigned bool
		for _, group := range groups {
			comparison := comparator.Compare(r.Response, group.Canonical)
			if comparison.Match {
				group.Members = append(group.Members, r)
				if comparison.Confidence < group.Confidence {
					group.Confidence = comparison.Confidence
				}
				assigned = true
				break
			}
		}

		if !assigned {
			groups = append(groups, &ResponseGroup{
				Canonical:  r.Response,
				Members:    []HarnessResult{r},
				Confidence: 1.0,
			})
		}
	}

	// Sort by size (largest first), then by confidence
	sort.Slice(groups, func(i, j int) bool {
		if len(groups[i].Members) != len(groups[j].Members) {
			return len(groups[i].Members) > len(groups[j].Members)
		}
		return groups[i].Confidence > groups[j].Confidence
	})

	// Convert to non-pointer slice
	result := make([]ResponseGroup, len(groups))
	for i, g := range groups {
		result[i] = *g
	}

	return result
}
