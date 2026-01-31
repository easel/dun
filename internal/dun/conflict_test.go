package dun

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestResolveQuorumMet(t *testing.T) {
	groups := []ResponseGroup{
		{
			Canonical:  "response content",
			Members:    makeHarnessResults("claude", "gemini"),
			Confidence: 1.0,
		},
	}
	config := QuorumConfig{
		Strategy:       "majority",
		TotalHarnesses: 3,
	}

	cr := NewConflictResolver(false, "", nil, nil)
	res := cr.Resolve(groups, config)

	if res.Outcome != "accepted" {
		t.Fatalf("expected accepted, got %s", res.Outcome)
	}
	if res.Response != "response content" {
		t.Fatalf("expected response content, got %s", res.Response)
	}
	if !strings.Contains(res.Reason, "quorum met") {
		t.Fatalf("expected quorum met reason, got %s", res.Reason)
	}
}

func TestResolveQuorumMetUnanimous(t *testing.T) {
	groups := []ResponseGroup{
		{
			Canonical:  "all agree",
			Members:    makeHarnessResults("claude", "gemini", "codex"),
			Confidence: 1.0,
		},
	}
	config := QuorumConfig{
		Strategy:       "unanimous",
		TotalHarnesses: 3,
	}

	cr := NewConflictResolver(false, "", nil, nil)
	res := cr.Resolve(groups, config)

	if res.Outcome != "accepted" {
		t.Fatalf("expected accepted, got %s", res.Outcome)
	}
}

func TestResolveQuorumNotMetDefaultSkip(t *testing.T) {
	groups := []ResponseGroup{
		{
			Canonical:  "response A",
			Members:    makeHarnessResults("claude"),
			Confidence: 1.0,
		},
		{
			Canonical:  "response B",
			Members:    makeHarnessResults("gemini"),
			Confidence: 1.0,
		},
	}
	config := QuorumConfig{
		Strategy:       "unanimous",
		TotalHarnesses: 2,
	}

	cr := NewConflictResolver(false, "", nil, nil)
	res := cr.Resolve(groups, config)

	if res.Outcome != "skipped" {
		t.Fatalf("expected skipped, got %s", res.Outcome)
	}
	if res.Conflict == nil {
		t.Fatalf("expected conflict report")
	}
	if !strings.Contains(res.Reason, "quorum not met") {
		t.Fatalf("expected quorum not met reason, got %s", res.Reason)
	}
}

func TestResolveEscalateUserSelectsOption(t *testing.T) {
	groups := []ResponseGroup{
		{
			Canonical:  "response A",
			Members:    makeHarnessResults("claude"),
			Confidence: 1.0,
		},
		{
			Canonical:  "response B",
			Members:    makeHarnessResults("gemini"),
			Confidence: 1.0,
		},
	}
	config := QuorumConfig{
		Strategy:       "unanimous",
		TotalHarnesses: 2,
	}

	stdin := strings.NewReader("1\n")
	stdout := &bytes.Buffer{}
	cr := NewConflictResolver(true, "", stdin, stdout)
	res := cr.Resolve(groups, config)

	if res.Outcome != "accepted" {
		t.Fatalf("expected accepted, got %s", res.Outcome)
	}
	if res.Response != "response A" {
		t.Fatalf("expected response A, got %s", res.Response)
	}
	if !strings.Contains(res.Reason, "user selected") {
		t.Fatalf("expected user selected reason, got %s", res.Reason)
	}

	// Verify output contains prompts
	output := stdout.String()
	if !strings.Contains(output, "QUORUM CONFLICT") {
		t.Fatalf("expected conflict header in output")
	}
	if !strings.Contains(output, "Option 1") {
		t.Fatalf("expected Option 1 in output")
	}
	if !strings.Contains(output, "Option 2") {
		t.Fatalf("expected Option 2 in output")
	}
}

func TestResolveEscalateUserSelectsSecondOption(t *testing.T) {
	groups := []ResponseGroup{
		{
			Canonical:  "response A",
			Members:    makeHarnessResults("claude"),
			Confidence: 1.0,
		},
		{
			Canonical:  "response B",
			Members:    makeHarnessResults("gemini"),
			Confidence: 1.0,
		},
	}
	config := QuorumConfig{
		Strategy:       "unanimous",
		TotalHarnesses: 2,
	}

	stdin := strings.NewReader("2\n")
	stdout := &bytes.Buffer{}
	cr := NewConflictResolver(true, "", stdin, stdout)
	res := cr.Resolve(groups, config)

	if res.Outcome != "accepted" {
		t.Fatalf("expected accepted, got %s", res.Outcome)
	}
	if res.Response != "response B" {
		t.Fatalf("expected response B, got %s", res.Response)
	}
}

func TestResolveEscalateUserSkips(t *testing.T) {
	groups := []ResponseGroup{
		{
			Canonical:  "response A",
			Members:    makeHarnessResults("claude"),
			Confidence: 1.0,
		},
	}
	config := QuorumConfig{
		Strategy:       "",
		Threshold:      2,
		TotalHarnesses: 2,
	}

	stdin := strings.NewReader("s\n")
	stdout := &bytes.Buffer{}
	cr := NewConflictResolver(true, "", stdin, stdout)
	res := cr.Resolve(groups, config)

	if res.Outcome != "skipped" {
		t.Fatalf("expected skipped, got %s", res.Outcome)
	}
	if !strings.Contains(res.Reason, "user skipped") {
		t.Fatalf("expected user skipped reason, got %s", res.Reason)
	}
}

func TestResolveEscalateUserQuits(t *testing.T) {
	groups := []ResponseGroup{
		{
			Canonical:  "response A",
			Members:    makeHarnessResults("claude"),
			Confidence: 1.0,
		},
	}
	config := QuorumConfig{
		Strategy:       "",
		Threshold:      2,
		TotalHarnesses: 2,
	}

	stdin := strings.NewReader("q\n")
	stdout := &bytes.Buffer{}
	cr := NewConflictResolver(true, "", stdin, stdout)
	res := cr.Resolve(groups, config)

	if res.Outcome != "aborted" {
		t.Fatalf("expected aborted, got %s", res.Outcome)
	}
	if !strings.Contains(res.Reason, "user quit") {
		t.Fatalf("expected user quit reason, got %s", res.Reason)
	}
}

func TestResolveEscalateInvalidChoice(t *testing.T) {
	groups := []ResponseGroup{
		{
			Canonical:  "response A",
			Members:    makeHarnessResults("claude"),
			Confidence: 1.0,
		},
	}
	config := QuorumConfig{
		Strategy:       "",
		Threshold:      2,
		TotalHarnesses: 2,
	}

	stdin := strings.NewReader("99\n")
	stdout := &bytes.Buffer{}
	cr := NewConflictResolver(true, "", stdin, stdout)
	res := cr.Resolve(groups, config)

	if res.Outcome != "skipped" {
		t.Fatalf("expected skipped, got %s", res.Outcome)
	}
	if !strings.Contains(res.Reason, "invalid choice") {
		t.Fatalf("expected invalid choice reason, got %s", res.Reason)
	}
}

func TestResolveEscalateInvalidInput(t *testing.T) {
	groups := []ResponseGroup{
		{
			Canonical:  "response A",
			Members:    makeHarnessResults("claude"),
			Confidence: 1.0,
		},
	}
	config := QuorumConfig{
		Strategy:       "",
		Threshold:      2,
		TotalHarnesses: 2,
	}

	stdin := strings.NewReader("not-a-number\n")
	stdout := &bytes.Buffer{}
	cr := NewConflictResolver(true, "", stdin, stdout)
	res := cr.Resolve(groups, config)

	if res.Outcome != "skipped" {
		t.Fatalf("expected skipped, got %s", res.Outcome)
	}
	if !strings.Contains(res.Reason, "invalid choice") {
		t.Fatalf("expected invalid choice reason, got %s", res.Reason)
	}
}

func TestResolveEscalateNoInput(t *testing.T) {
	groups := []ResponseGroup{
		{
			Canonical:  "response A",
			Members:    makeHarnessResults("claude"),
			Confidence: 1.0,
		},
	}
	config := QuorumConfig{
		Strategy:       "",
		Threshold:      2,
		TotalHarnesses: 2,
	}

	stdin := strings.NewReader("")
	stdout := &bytes.Buffer{}
	cr := NewConflictResolver(true, "", stdin, stdout)
	res := cr.Resolve(groups, config)

	if res.Outcome != "skipped" {
		t.Fatalf("expected skipped, got %s", res.Outcome)
	}
	if !strings.Contains(res.Reason, "no input") {
		t.Fatalf("expected no input reason, got %s", res.Reason)
	}
}

func TestResolvePreferredHarness(t *testing.T) {
	groups := []ResponseGroup{
		{
			Canonical:  "response A",
			Members:    makeHarnessResultsWithResponse("claude", "response A"),
			Confidence: 1.0,
		},
		{
			Canonical:  "response B",
			Members:    makeHarnessResultsWithResponse("gemini", "response B"),
			Confidence: 1.0,
		},
	}
	config := QuorumConfig{
		Strategy:       "unanimous",
		TotalHarnesses: 2,
	}

	cr := NewConflictResolver(false, "gemini", nil, nil)
	res := cr.Resolve(groups, config)

	if res.Outcome != "accepted" {
		t.Fatalf("expected accepted, got %s", res.Outcome)
	}
	if res.Response != "response B" {
		t.Fatalf("expected response B (gemini's response), got %s", res.Response)
	}
	if !strings.Contains(res.Reason, "preferred harness") {
		t.Fatalf("expected preferred harness reason, got %s", res.Reason)
	}
}

func TestResolvePreferredHarnessNotFound(t *testing.T) {
	groups := []ResponseGroup{
		{
			Canonical:  "response A",
			Members:    makeHarnessResults("claude"),
			Confidence: 1.0,
		},
	}
	config := QuorumConfig{
		Strategy:       "",
		Threshold:      2,
		TotalHarnesses: 2,
	}

	cr := NewConflictResolver(false, "nonexistent", nil, nil)
	res := cr.Resolve(groups, config)

	if res.Outcome != "skipped" {
		t.Fatalf("expected skipped, got %s", res.Outcome)
	}
	if !strings.Contains(res.Reason, "not found") {
		t.Fatalf("expected not found reason, got %s", res.Reason)
	}
}

func TestResolveEmptyGroups(t *testing.T) {
	cr := NewConflictResolver(false, "", nil, nil)
	res := cr.Resolve(nil, QuorumConfig{})

	if res.Outcome != "skipped" {
		t.Fatalf("expected skipped, got %s", res.Outcome)
	}
	if !strings.Contains(res.Reason, "no response groups") {
		t.Fatalf("expected no response groups reason, got %s", res.Reason)
	}
}

func TestBuildConflictReport(t *testing.T) {
	groups := []ResponseGroup{
		{
			Canonical:  "line1\nline2",
			Members:    makeHarnessResults("claude", "codex"),
			Confidence: 1.0,
		},
		{
			Canonical:  "line1\nline3",
			Members:    makeHarnessResults("gemini"),
			Confidence: 1.0,
		},
	}
	config := QuorumConfig{
		Strategy:       "unanimous",
		TotalHarnesses: 3,
	}

	cr := NewConflictResolver(false, "", nil, nil)
	report := cr.buildConflictReport(groups, config)

	if len(report.Groups) != 2 {
		t.Fatalf("expected 2 groups, got %d", len(report.Groups))
	}
	if len(report.Harnesses) != 3 {
		t.Fatalf("expected 3 harnesses, got %d", len(report.Harnesses))
	}
	if len(report.Diffs) != 1 {
		t.Fatalf("expected 1 diff, got %d", len(report.Diffs))
	}
	if report.Diffs[0].GroupA != 0 || report.Diffs[0].GroupB != 1 {
		t.Fatalf("expected diff between groups 0 and 1")
	}
	if report.Quorum.Strategy != "unanimous" {
		t.Fatalf("expected quorum config in report")
	}
}

func TestConflictReportDiffContent(t *testing.T) {
	groups := []ResponseGroup{
		{
			Canonical:  "same\ndifferent A\nend",
			Members:    makeHarnessResults("claude"),
			Confidence: 1.0,
		},
		{
			Canonical:  "same\ndifferent B\nend",
			Members:    makeHarnessResults("gemini"),
			Confidence: 1.0,
		},
	}
	config := QuorumConfig{TotalHarnesses: 2}

	cr := NewConflictResolver(false, "", nil, nil)
	report := cr.buildConflictReport(groups, config)

	diff := report.Diffs[0].Unified
	if !strings.Contains(diff, "- different A") {
		t.Fatalf("expected removed line in diff")
	}
	if !strings.Contains(diff, "+ different B") {
		t.Fatalf("expected added line in diff")
	}
	if !strings.Contains(diff, "  same") {
		t.Fatalf("expected unchanged line in diff")
	}
}

func TestHarnessNames(t *testing.T) {
	members := makeHarnessResults("claude", "gemini", "codex")
	names := harnessNames(members)

	if names != "claude, gemini, codex" {
		t.Fatalf("expected 'claude, gemini, codex', got '%s'", names)
	}
}

func TestTruncate(t *testing.T) {
	short := "hello"
	if truncate(short, 10) != "hello" {
		t.Fatalf("short string should not be truncated")
	}

	long := "hello world this is a long string"
	truncated := truncate(long, 15)
	if truncated != "hello world ..." {
		t.Fatalf("expected 'hello world ...', got '%s'", truncated)
	}
}

func TestUnifiedDiff(t *testing.T) {
	a := "line1\nline2\nline3"
	b := "line1\nmodified\nline3"

	diff := unifiedDiff(a, b)

	if !strings.Contains(diff, "--- a") {
		t.Fatalf("expected diff header")
	}
	if !strings.Contains(diff, "+++ b") {
		t.Fatalf("expected diff header")
	}
	if !strings.Contains(diff, "- line2") {
		t.Fatalf("expected removed line")
	}
	if !strings.Contains(diff, "+ modified") {
		t.Fatalf("expected added line")
	}
}

func TestUnifiedDiffDifferentLengths(t *testing.T) {
	a := "line1\nline2"
	b := "line1\nline2\nline3"

	diff := unifiedDiff(a, b)

	if !strings.Contains(diff, "+ line3") {
		t.Fatalf("expected added line3")
	}
}

func TestResolveWithNumericThreshold(t *testing.T) {
	groups := []ResponseGroup{
		{
			Canonical:  "response",
			Members:    makeHarnessResults("claude", "gemini"),
			Confidence: 1.0,
		},
	}
	config := QuorumConfig{
		Strategy:       "",
		Threshold:      2,
		TotalHarnesses: 3,
	}

	cr := NewConflictResolver(false, "", nil, nil)
	res := cr.Resolve(groups, config)

	if res.Outcome != "accepted" {
		t.Fatalf("expected accepted, got %s", res.Outcome)
	}
}

func TestResolveWithMajority(t *testing.T) {
	// 2 out of 3 is majority
	groups := []ResponseGroup{
		{
			Canonical:  "response",
			Members:    makeHarnessResults("claude", "gemini"),
			Confidence: 1.0,
		},
		{
			Canonical:  "other",
			Members:    makeHarnessResults("codex"),
			Confidence: 1.0,
		},
	}
	config := QuorumConfig{
		Strategy:       "majority",
		TotalHarnesses: 3,
	}

	cr := NewConflictResolver(false, "", nil, nil)
	res := cr.Resolve(groups, config)

	if res.Outcome != "accepted" {
		t.Fatalf("expected accepted, got %s", res.Outcome)
	}
}

func TestResolveWithAnyStrategy(t *testing.T) {
	groups := []ResponseGroup{
		{
			Canonical:  "response",
			Members:    makeHarnessResults("claude"),
			Confidence: 1.0,
		},
	}
	config := QuorumConfig{
		Strategy:       "any",
		TotalHarnesses: 3,
	}

	cr := NewConflictResolver(false, "", nil, nil)
	res := cr.Resolve(groups, config)

	if res.Outcome != "accepted" {
		t.Fatalf("expected accepted, got %s", res.Outcome)
	}
}

func TestEscalateShowsTruncatedResponse(t *testing.T) {
	longResponse := strings.Repeat("a", 600)
	groups := []ResponseGroup{
		{
			Canonical:  longResponse,
			Members:    makeHarnessResults("claude"),
			Confidence: 1.0,
		},
	}
	config := QuorumConfig{
		Strategy:       "",
		Threshold:      2,
		TotalHarnesses: 2,
	}

	stdin := strings.NewReader("s\n")
	stdout := &bytes.Buffer{}
	cr := NewConflictResolver(true, "", stdin, stdout)
	cr.Resolve(groups, config)

	output := stdout.String()
	// Should be truncated to 500 chars + "..."
	if strings.Contains(output, strings.Repeat("a", 600)) {
		t.Fatalf("expected response to be truncated")
	}
	if !strings.Contains(output, "...") {
		t.Fatalf("expected truncation indicator")
	}
}

func TestPreferHarnessFromSecondGroup(t *testing.T) {
	groups := []ResponseGroup{
		{
			Canonical:  "response A",
			Members:    makeHarnessResultsWithResponse("claude", "response A"),
			Confidence: 1.0,
		},
		{
			Canonical:  "response B",
			Members:    makeHarnessResultsWithResponse2("gemini", "codex", "response B"),
			Confidence: 1.0,
		},
	}
	config := QuorumConfig{
		Strategy:       "unanimous",
		TotalHarnesses: 3,
	}

	// Prefer codex which is in the second group
	cr := NewConflictResolver(false, "codex", nil, nil)
	res := cr.Resolve(groups, config)

	if res.Outcome != "accepted" {
		t.Fatalf("expected accepted, got %s", res.Outcome)
	}
	if res.Response != "response B" {
		t.Fatalf("expected response B, got %s", res.Response)
	}
}

// makeHarnessResults creates HarnessResult entries for testing.
func makeHarnessResults(names ...string) []HarnessResult {
	results := make([]HarnessResult, len(names))
	for i, name := range names {
		results[i] = HarnessResult{
			Harness:   name,
			Response:  "response from " + name,
			Timestamp: time.Now(),
			Duration:  100 * time.Millisecond,
		}
	}
	return results
}

// makeHarnessResultsWithResponse creates HarnessResult entries with a specific response.
func makeHarnessResultsWithResponse(name, response string) []HarnessResult {
	return []HarnessResult{
		{
			Harness:   name,
			Response:  response,
			Timestamp: time.Now(),
			Duration:  100 * time.Millisecond,
		},
	}
}

// makeHarnessResultsWithResponse2 creates two HarnessResult entries with the same response.
func makeHarnessResultsWithResponse2(name1, name2, response string) []HarnessResult {
	return []HarnessResult{
		{
			Harness:   name1,
			Response:  response,
			Timestamp: time.Now(),
			Duration:  100 * time.Millisecond,
		},
		{
			Harness:   name2,
			Response:  response,
			Timestamp: time.Now(),
			Duration:  100 * time.Millisecond,
		},
	}
}
