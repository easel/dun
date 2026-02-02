package dun

import (
	"testing"
	"time"
)

func TestResponseNormalizer_NormalizeLineEndings(t *testing.T) {
	n := ResponseNormalizer{NormalizeLineEnds: true}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"windows to unix", "line1\r\nline2\r\n", "line1\nline2"},
		{"mac to unix", "line1\rline2\r", "line1\nline2"},
		{"mixed", "line1\r\nline2\rline3\n", "line1\nline2\nline3"},
		{"already unix", "line1\nline2\n", "line1\nline2"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := n.Normalize(tt.input)
			if result != tt.expected {
				t.Errorf("got %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestResponseNormalizer_CollapseWhitespace(t *testing.T) {
	n := ResponseNormalizer{StripWhitespace: true, NormalizeLineEnds: true}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"multiple spaces", "hello   world", "hello world"},
		{"tabs", "hello\t\tworld", "hello world"},
		{"mixed", "hello  \t world", "hello world"},
		{"leading trailing", "  hello world  ", "hello world"},
		{"empty lines", "line1\n\n\nline2", "line1\nline2"},
		{"whitespace lines", "line1\n   \n\t\nline2", "line1\nline2"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := n.Normalize(tt.input)
			if result != tt.expected {
				t.Errorf("got %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestResponseNormalizer_StripComments(t *testing.T) {
	n := ResponseNormalizer{IgnoreComments: true, StripWhitespace: true, NormalizeLineEnds: true}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			"single line comment",
			"code // comment\nmore code",
			"code\nmore code",
		},
		{
			"multi-line comment",
			"code /* comment\nspanning lines */ more",
			"code more",
		},
		{
			"nested style not supported",
			"code /* outer /* inner */ end */ more",
			"code end */ more",
		},
		{
			"multiple comments",
			"a // first\nb /* second */ c",
			"a\nb c",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := n.Normalize(tt.input)
			if result != tt.expected {
				t.Errorf("got %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestResponseNormalizer_SortJSONKeys(t *testing.T) {
	n := ResponseNormalizer{SortJSONKeys: true}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			"simple object",
			`{"z": 1, "a": 2}`,
			`{"a":2,"z":1}`,
		},
		{
			"nested object",
			`{"z": {"b": 1, "a": 2}, "a": 3}`,
			`{"a":3,"z":{"a":2,"b":1}}`,
		},
		{
			"array",
			`[{"b": 1, "a": 2}]`,
			`[{"a":2,"b":1}]`,
		},
		{
			"not json",
			`not json at all`,
			`not json at all`,
		},
		{
			"empty object",
			`{}`,
			`{}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := n.Normalize(tt.input)
			if result != tt.expected {
				t.Errorf("got %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestResponseNormalizer_CombinedNormalization(t *testing.T) {
	n := DefaultNormalizer()

	input := `{
		"z": 1,   // comment
		"a": 2
	}`

	result := n.Normalize(input)

	// After normalization: sorted keys, no comments, collapsed whitespace
	expected := `{"a":2,"z":1}`
	if result != expected {
		t.Errorf("got %q, want %q", result, expected)
	}
}

func TestLooksLikeJSON(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{`{"key": "value"}`, true},
		{`[1, 2, 3]`, true},
		{`  {"key": "value"}  `, true},
		{`not json`, false},
		{`{incomplete`, false},
		{``, false},
		{`a`, false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := looksLikeJSON(tt.input); got != tt.expected {
				t.Errorf("looksLikeJSON(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestSemanticComparator_ExactMatch(t *testing.T) {
	sc := DefaultComparator()

	tests := []struct {
		name string
		a    string
		b    string
		want bool
	}{
		{"identical", "hello world", "hello world", true},
		{"whitespace diff", "hello  world", "hello world", true},
		{"line ending diff", "hello\r\nworld", "hello\nworld", true},
		{"comment diff", "code // comment", "code", true},
		{"json key order", `{"b":1,"a":2}`, `{"a":2,"b":1}`, true},
		{"different content", "hello", "goodbye", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sc.Compare(tt.a, tt.b)
			if result.Match != tt.want {
				t.Errorf("Compare(%q, %q).Match = %v, want %v", tt.a, tt.b, result.Match, tt.want)
			}
			if tt.want && result.Level != "exact" {
				t.Errorf("expected level 'exact', got %q", result.Level)
			}
		})
	}
}

func TestSemanticComparator_StructuralMatch(t *testing.T) {
	sc := NewSemanticComparator(0.95) // High threshold for testing

	// These are structurally different - only 2 of 3 lines match
	a := `func main() {
		fmt.Println("hello")
		return
	}`

	b := `func main() {
		fmt.Println("world")
		return
	}`

	result := sc.Compare(a, b)

	// 3 lines, 1 different = 2/3 = 0.66 structural similarity (below 0.95)
	// But semantic (character-level) might be higher
	// The important thing is these don't match at the "exact" level
	if result.Level == "exact" {
		t.Errorf("expected non-exact match level for different content")
	}

	// Test that completely different content doesn't match
	e := `completely different content here
	nothing in common at all
	more different stuff`

	f := `some other unrelated text
	absolutely nothing alike
	totally distinct`

	result = sc.Compare(e, f)
	if result.Match {
		t.Errorf("expected no match for completely different content, got confidence %v", result.Confidence)
	}
}

func TestSemanticComparator_StructuralMatchHighSimilarity(t *testing.T) {
	sc := NewSemanticComparator(0.7)

	a := `line1
line2
line3
line4
line5`

	b := `line1
line2
line3
line4
lineFIVE` // 1 of 5 lines different = 80% similarity

	result := sc.Compare(a, b)

	if !result.Match {
		t.Errorf("expected structural match for 80%% similar content, got confidence %v", result.Confidence)
	}
	if result.Level != "structural" {
		t.Errorf("expected level 'structural', got %q", result.Level)
	}
}

func TestSemanticComparator_Threshold(t *testing.T) {
	tests := []struct {
		name      string
		threshold float64
		a         string
		b         string
		wantMatch bool
	}{
		{"high threshold exact", 0.99, "hello", "hello", true},
		{"high threshold differ", 0.99, "hello", "hallo", false},
		{"low threshold similar", 0.5, "hello world", "hello earth", true},
		{"low threshold differ", 0.5, "abc", "xyz", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sc := NewSemanticComparator(tt.threshold)
			result := sc.Compare(tt.a, tt.b)
			if result.Match != tt.wantMatch {
				t.Errorf("threshold=%v: Compare(%q, %q).Match = %v, want %v (confidence=%v)",
					tt.threshold, tt.a, tt.b, result.Match, tt.wantMatch, result.Confidence)
			}
		})
	}
}

func TestSemanticComparator_EmptyStrings(t *testing.T) {
	sc := DefaultComparator()

	tests := []struct {
		name string
		a    string
		b    string
		want bool
	}{
		{"both empty", "", "", true},
		{"a empty", "", "hello", false},
		{"b empty", "hello", "", false},
		{"both whitespace", "   ", "\t\n", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sc.Compare(tt.a, tt.b)
			if result.Match != tt.want {
				t.Errorf("Compare(%q, %q).Match = %v, want %v", tt.a, tt.b, result.Match, tt.want)
			}
		})
	}
}

func TestSemanticComparator_DiffOutput(t *testing.T) {
	sc := DefaultComparator()

	result := sc.Compare("line1\nline2", "line1\nline3")

	if result.Match {
		t.Fatal("expected no match")
	}
	if result.Diff == "" {
		t.Error("expected diff output for non-matching comparison")
	}
	if !containsString(result.Diff, "-line2") || !containsString(result.Diff, "+line3") {
		t.Errorf("diff should show removed and added lines, got: %s", result.Diff)
	}
}

func TestLineLevenshtein(t *testing.T) {
	tests := []struct {
		name     string
		a        []string
		b        []string
		expected int
	}{
		{"identical", []string{"a", "b", "c"}, []string{"a", "b", "c"}, 0},
		{"one insert", []string{"a", "b"}, []string{"a", "x", "b"}, 1},
		{"one delete", []string{"a", "x", "b"}, []string{"a", "b"}, 1},
		{"one substitute", []string{"a", "b", "c"}, []string{"a", "x", "c"}, 1},
		{"empty a", []string{}, []string{"a", "b"}, 2},
		{"empty b", []string{"a", "b"}, []string{}, 2},
		{"both empty", []string{}, []string{}, 0},
		{"completely different", []string{"a", "b"}, []string{"x", "y"}, 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := lineLevenshtein(tt.a, tt.b); got != tt.expected {
				t.Errorf("lineLevenshtein(%v, %v) = %d, want %d", tt.a, tt.b, got, tt.expected)
			}
		})
	}
}

func TestLevenshtein(t *testing.T) {
	tests := []struct {
		a, b     string
		expected int
	}{
		{"", "", 0},
		{"abc", "", 3},
		{"", "abc", 3},
		{"abc", "abc", 0},
		{"abc", "abd", 1},
		{"kitten", "sitting", 3},
		{"saturday", "sunday", 3},
	}

	for _, tt := range tests {
		t.Run(tt.a+"_"+tt.b, func(t *testing.T) {
			if got := levenshtein(tt.a, tt.b); got != tt.expected {
				t.Errorf("levenshtein(%q, %q) = %d, want %d", tt.a, tt.b, got, tt.expected)
			}
		})
	}
}

func TestLevenshteinUnicode(t *testing.T) {
	// Test Unicode handling
	tests := []struct {
		name     string
		a, b     string
		expected int
	}{
		{"identical ASCII", "cafe", "cafe", 0},
		{"identical", "hello", "hello", 0},
		{"accent difference", "caf\u00e9", "cafe", 1}, // cafe with accent vs plain cafe (1 char different)
		{"one insert", "abc", "abcd", 1},
		{"emoji", "\U0001F600", "\U0001F601", 1}, // different emoji
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := levenshtein(tt.a, tt.b)
			if got != tt.expected {
				t.Errorf("levenshtein(%q, %q) = %d, want %d", tt.a, tt.b, got, tt.expected)
			}
		})
	}
}

func TestGroupByAgreement_EmptyResults(t *testing.T) {
	results := []HarnessResult{}
	groups := GroupByAgreement(results, nil)

	if len(groups) != 0 {
		t.Errorf("expected 0 groups, got %d", len(groups))
	}
}

func TestGroupByAgreement_AllErrors(t *testing.T) {
	results := []HarnessResult{
		{Harness: "h1", Error: errTest},
		{Harness: "h2", Error: errTest},
	}
	groups := GroupByAgreement(results, nil)

	if len(groups) != 0 {
		t.Errorf("expected 0 groups for all errors, got %d", len(groups))
	}
}

var errTest = stringError("test error")

type stringError string

func (e stringError) Error() string { return string(e) }

func TestGroupByAgreement_IdenticalResponses(t *testing.T) {
	results := []HarnessResult{
		{Harness: "claude", Response: "answer", Timestamp: time.Now()},
		{Harness: "gemini", Response: "answer", Timestamp: time.Now()},
		{Harness: "codex", Response: "answer", Timestamp: time.Now()},
	}

	groups := GroupByAgreement(results, nil)

	if len(groups) != 1 {
		t.Fatalf("expected 1 group, got %d", len(groups))
	}
	if len(groups[0].Members) != 3 {
		t.Errorf("expected 3 members, got %d", len(groups[0].Members))
	}
	if groups[0].Confidence != 1.0 {
		t.Errorf("expected confidence 1.0, got %v", groups[0].Confidence)
	}
}

func TestGroupByAgreement_NormalizedMatch(t *testing.T) {
	// All these should normalize to "hello world" (single line, single space)
	results := []HarnessResult{
		{Harness: "claude", Response: "hello world", Timestamp: time.Now()},
		{Harness: "gemini", Response: "hello  world", Timestamp: time.Now()},       // extra space
		{Harness: "codex", Response: "  hello world  ", Timestamp: time.Now()},     // leading/trailing space
		{Harness: "other", Response: "hello world // comment", Timestamp: time.Now()}, // with comment
	}

	groups := GroupByAgreement(results, nil)

	if len(groups) != 1 {
		t.Fatalf("expected 1 group after normalization, got %d", len(groups))
	}
	if len(groups[0].Members) != 4 {
		t.Errorf("expected 4 members, got %d", len(groups[0].Members))
	}
}

func TestGroupByAgreement_DifferentResponses(t *testing.T) {
	results := []HarnessResult{
		{Harness: "claude", Response: "yes", Timestamp: time.Now()},
		{Harness: "gemini", Response: "no", Timestamp: time.Now()},
		{Harness: "codex", Response: "maybe", Timestamp: time.Now()},
	}

	groups := GroupByAgreement(results, nil)

	if len(groups) != 3 {
		t.Fatalf("expected 3 groups for different responses, got %d", len(groups))
	}
}

func TestGroupByAgreement_MixedAgreement(t *testing.T) {
	results := []HarnessResult{
		{Harness: "claude", Response: "yes", Timestamp: time.Now()},
		{Harness: "gemini", Response: "yes", Timestamp: time.Now()},
		{Harness: "codex", Response: "no", Timestamp: time.Now()},
	}

	groups := GroupByAgreement(results, nil)

	if len(groups) != 2 {
		t.Fatalf("expected 2 groups, got %d", len(groups))
	}

	// First group should be largest
	if len(groups[0].Members) != 2 {
		t.Errorf("expected first group to have 2 members, got %d", len(groups[0].Members))
	}
	if groups[0].Canonical != "yes" {
		t.Errorf("expected first group canonical to be 'yes', got %q", groups[0].Canonical)
	}

	if len(groups[1].Members) != 1 {
		t.Errorf("expected second group to have 1 member, got %d", len(groups[1].Members))
	}
}

func TestGroupByAgreement_WithErrors(t *testing.T) {
	results := []HarnessResult{
		{Harness: "claude", Response: "answer", Timestamp: time.Now()},
		{Harness: "gemini", Response: "", Error: errTest, Timestamp: time.Now()},
		{Harness: "codex", Response: "answer", Timestamp: time.Now()},
	}

	groups := GroupByAgreement(results, nil)

	if len(groups) != 1 {
		t.Fatalf("expected 1 group (ignoring errors), got %d", len(groups))
	}
	if len(groups[0].Members) != 2 {
		t.Errorf("expected 2 members (excluding error), got %d", len(groups[0].Members))
	}
}

func TestGroupByAgreement_JSONKeyOrder(t *testing.T) {
	results := []HarnessResult{
		{Harness: "claude", Response: `{"a":1,"b":2}`, Timestamp: time.Now()},
		{Harness: "gemini", Response: `{"b":2,"a":1}`, Timestamp: time.Now()},
	}

	groups := GroupByAgreement(results, nil)

	if len(groups) != 1 {
		t.Fatalf("expected 1 group for same JSON with different key order, got %d", len(groups))
	}
	if len(groups[0].Members) != 2 {
		t.Errorf("expected 2 members, got %d", len(groups[0].Members))
	}
}

func TestGroupByAgreement_CustomComparator(t *testing.T) {
	// Use a stricter comparator that doesn't normalize
	strictNormalizer := ResponseNormalizer{
		StripWhitespace:   false,
		NormalizeLineEnds: false,
		SortJSONKeys:      false,
		IgnoreComments:    false,
	}
	sc := &SemanticComparator{Normalizer: strictNormalizer, Threshold: 0.95}

	results := []HarnessResult{
		{Harness: "claude", Response: "hello world", Timestamp: time.Now()},
		{Harness: "gemini", Response: "hello  world", Timestamp: time.Now()}, // extra space
	}

	groups := GroupByAgreement(results, sc)

	if len(groups) != 2 {
		t.Fatalf("expected 2 groups with strict comparator, got %d", len(groups))
	}
}

func TestGroupByAgreement_ConfidenceTracking(t *testing.T) {
	// Create a comparator with low threshold to allow structural matches
	sc := NewSemanticComparator(0.6)

	results := []HarnessResult{
		{Harness: "claude", Response: "line1\nline2\nline3", Timestamp: time.Now()},
		{Harness: "gemini", Response: "line1\nline2\nline3", Timestamp: time.Now()},      // exact match
		{Harness: "codex", Response: "line1\nlineTWO\nline3", Timestamp: time.Now()},     // 1 line different
	}

	groups := GroupByAgreement(results, sc)

	if len(groups) != 1 {
		t.Fatalf("expected 1 group, got %d", len(groups))
	}

	// Confidence should be less than 1.0 due to the structural match
	if groups[0].Confidence >= 1.0 {
		t.Errorf("expected confidence < 1.0 due to structural match, got %v", groups[0].Confidence)
	}
}

func TestGroupByAgreement_SortedBySize(t *testing.T) {
	results := []HarnessResult{
		{Harness: "h1", Response: "small", Timestamp: time.Now()},
		{Harness: "h2", Response: "large", Timestamp: time.Now()},
		{Harness: "h3", Response: "large", Timestamp: time.Now()},
		{Harness: "h4", Response: "large", Timestamp: time.Now()},
	}

	groups := GroupByAgreement(results, nil)

	if len(groups) != 2 {
		t.Fatalf("expected 2 groups, got %d", len(groups))
	}

	// First group should be largest
	if len(groups[0].Members) != 3 {
		t.Errorf("expected first group to have 3 members (large), got %d", len(groups[0].Members))
	}
	if len(groups[1].Members) != 1 {
		t.Errorf("expected second group to have 1 member (small), got %d", len(groups[1].Members))
	}
}

func TestDefaultComparator(t *testing.T) {
	sc := DefaultComparator()

	if sc.Threshold != 0.95 {
		t.Errorf("expected default threshold 0.95, got %v", sc.Threshold)
	}
	if !sc.Normalizer.StripWhitespace {
		t.Error("expected StripWhitespace to be true by default")
	}
	if !sc.Normalizer.NormalizeLineEnds {
		t.Error("expected NormalizeLineEnds to be true by default")
	}
	if !sc.Normalizer.SortJSONKeys {
		t.Error("expected SortJSONKeys to be true by default")
	}
	if !sc.Normalizer.IgnoreComments {
		t.Error("expected IgnoreComments to be true by default")
	}
}

func TestComparator_Normalize(t *testing.T) {
	sc := DefaultComparator()

	input := "  hello  world  "
	expected := "hello world"

	if got := sc.Normalize(input); got != expected {
		t.Errorf("Normalize(%q) = %q, want %q", input, got, expected)
	}
}

func TestSignificantLines(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{"a\nb\nc", []string{"a", "b", "c"}},
		{"a\n\nb", []string{"a", "b"}},
		{"  a  \n  b  ", []string{"a", "b"}},
		{"", nil},
		{"   \n   \n   ", nil},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := significantLines(tt.input)
			if len(got) != len(tt.expected) {
				t.Errorf("significantLines(%q) = %v, want %v", tt.input, got, tt.expected)
				return
			}
			for i := range got {
				if got[i] != tt.expected[i] {
					t.Errorf("significantLines(%q)[%d] = %q, want %q", tt.input, i, got[i], tt.expected[i])
				}
			}
		})
	}
}

func TestComputeDiff(t *testing.T) {
	diff := computeDiff("line1\nline2", "line1\nline3")

	if !containsString(diff, "--- a") {
		t.Error("diff should contain '--- a' header")
	}
	if !containsString(diff, "+++ b") {
		t.Error("diff should contain '+++ b' header")
	}
	if !containsString(diff, " line1") {
		t.Error("diff should show unchanged line1 with space prefix")
	}
	if !containsString(diff, "-line2") {
		t.Error("diff should show removed line2 with - prefix")
	}
	if !containsString(diff, "+line3") {
		t.Error("diff should show added line3 with + prefix")
	}
}

// Additional tests for edge cases

func TestSortJSONKeysError(t *testing.T) {
	// Invalid JSON should return original string
	result, err := sortJSONKeys("not valid json")
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
	if result != "not valid json" {
		t.Errorf("expected original string returned, got %q", result)
	}
}

func TestStructuralCompareEmptyInputs(t *testing.T) {
	sc := DefaultComparator()

	// Both normalized to empty
	result := sc.Compare("   ", "\n\n")
	if !result.Match {
		t.Error("expected match for both-empty-after-normalization")
	}
	if result.Confidence != 1.0 {
		t.Errorf("expected confidence 1.0, got %v", result.Confidence)
	}
}

func TestSemanticCompareEmptyAfterNormalization(t *testing.T) {
	sc := DefaultComparator()

	// Test where normalization results in empty strings
	result := sc.Compare("//comment only", "/* another comment */")
	if !result.Match {
		t.Error("expected match when both normalize to empty")
	}
}

func TestComputeDiffLongerA(t *testing.T) {
	diff := computeDiff("line1\nline2\nline3", "line1")

	if !containsString(diff, "-line2") {
		t.Error("diff should show removed line2")
	}
	if !containsString(diff, "-line3") {
		t.Error("diff should show removed line3")
	}
}

func TestComputeDiffLongerB(t *testing.T) {
	diff := computeDiff("line1", "line1\nline2\nline3")

	if !containsString(diff, "+line2") {
		t.Error("diff should show added line2")
	}
	if !containsString(diff, "+line3") {
		t.Error("diff should show added line3")
	}
}

func TestSemanticMatchLevel(t *testing.T) {
	// Use a threshold that allows semantic match
	sc := NewSemanticComparator(0.5)

	// These are different enough to not match structurally but similar enough semantically
	a := "hello world"
	b := "hello earth"

	result := sc.Compare(a, b)
	if !result.Match {
		t.Errorf("expected match with low threshold, confidence=%v", result.Confidence)
	}
	// Should be semantic level since structural won't match at character level
	if result.Level != "exact" && result.Level != "structural" && result.Level != "semantic" {
		t.Errorf("expected valid level, got %q", result.Level)
	}
}

func TestSortMapKeysNonContainer(t *testing.T) {
	// Test with primitive values (not map or slice)
	result := sortMapKeys("string value")
	if result != "string value" {
		t.Errorf("expected string to pass through unchanged, got %v", result)
	}

	result = sortMapKeys(42)
	if result != 42 {
		t.Errorf("expected int to pass through unchanged, got %v", result)
	}

	result = sortMapKeys(true)
	if result != true {
		t.Errorf("expected bool to pass through unchanged, got %v", result)
	}

	result = sortMapKeys(nil)
	if result != nil {
		t.Errorf("expected nil to pass through unchanged, got %v", result)
	}
}

// Helper function
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
