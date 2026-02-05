package dun

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// stubBdBinary creates a mock bd binary for testing
func stubBdBinary(t *testing.T) string {
	t.Helper()
	binDir := t.TempDir()
	bdPath := filepath.Join(binDir, "bd")
	script := `#!/bin/sh
set -e

# Check for failure mode
if [ -n "${DUN_BD_EXIT:-}" ]; then
  exit "${DUN_BD_EXIT}"
fi

# Check for custom output
if [ -n "${DUN_BD_OUTPUT:-}" ]; then
  echo "${DUN_BD_OUTPUT}"
  exit 0
fi

# Check for file-based output (for multiline JSON)
if [ -n "${DUN_BD_OUTPUT_FILE:-}" ]; then
  cat "${DUN_BD_OUTPUT_FILE}"
  exit 0
fi

# Default behavior based on subcommand
case "$2" in
  ready)
    if [ -n "${DUN_BD_READY_OUTPUT:-}" ]; then
      echo "${DUN_BD_READY_OUTPUT}"
    else
      echo "[]"
    fi
    ;;
  blocked)
    if [ -n "${DUN_BD_BLOCKED_OUTPUT:-}" ]; then
      echo "${DUN_BD_BLOCKED_OUTPUT}"
    else
      echo "[]"
    fi
    ;;
  *)
    echo "unknown bd subcommand: $2" >&2
    exit 1
    ;;
esac
`
	writeFile(t, bdPath, script)
	if err := os.Chmod(bdPath, 0755); err != nil {
		t.Fatalf("chmod bd: %v", err)
	}
	return binDir
}

// Test toIssue conversion
func TestBeadsIssueToIssue(t *testing.T) {
	bi := beadsIssue{
		ID:          "BEAD-123",
		Title:       "Implement feature X",
		Status:      "open",
		Priority:    1,
		Labels:      []string{"feature", "urgent"},
		BlockedBy:   []string{"BEAD-100"},
		Blocks:      []string{"BEAD-200"},
		Description: "Detailed description",
	}

	issue := bi.toIssue()

	if issue.ID != "BEAD-123" {
		t.Errorf("expected ID 'BEAD-123', got %q", issue.ID)
	}
	if issue.Summary != "Implement feature X" {
		t.Errorf("expected Summary 'Implement feature X', got %q", issue.Summary)
	}
}

func TestBeadsIssueToIssue_Empty(t *testing.T) {
	bi := beadsIssue{}
	issue := bi.toIssue()

	if issue.ID != "" {
		t.Errorf("expected empty ID, got %q", issue.ID)
	}
	if issue.Summary != "" {
		t.Errorf("expected empty Summary, got %q", issue.Summary)
	}
}

// Test toIssues conversion
func TestToIssues(t *testing.T) {
	beads := []beadsIssue{
		{ID: "BEAD-1", Title: "First bead"},
		{ID: "BEAD-2", Title: "Second bead"},
		{ID: "BEAD-3", Title: "Third bead"},
	}

	issues := toIssues(beads)

	if len(issues) != 3 {
		t.Fatalf("expected 3 issues, got %d", len(issues))
	}
	if issues[0].ID != "BEAD-1" {
		t.Errorf("expected ID 'BEAD-1', got %q", issues[0].ID)
	}
	if issues[1].Summary != "Second bead" {
		t.Errorf("expected Summary 'Second bead', got %q", issues[1].Summary)
	}
}

func TestToIssues_Empty(t *testing.T) {
	issues := toIssues([]beadsIssue{})

	if len(issues) != 0 {
		t.Errorf("expected 0 issues, got %d", len(issues))
	}
}

func TestToIssues_Nil(t *testing.T) {
	issues := toIssues(nil)

	if len(issues) != 0 {
		t.Errorf("expected 0 issues for nil input, got %d", len(issues))
	}
}

// Test runBeadsReadyCheck
func TestRunBeadsReadyCheck_Success(t *testing.T) {
	binDir := stubBdBinary(t)
	t.Setenv("PATH", binDir)
	t.Setenv("DUN_BD_READY_OUTPUT", `[{"id":"BEAD-1","title":"Ready task","status":"open","priority":1}]`)

	root := t.TempDir()
	res, err := runBeadsReadyCheck(root, CheckDefinition{ID: "beads-ready"})
	if err != nil {
		t.Fatalf("beads ready check: %v", err)
	}
	if res.Status != "action" {
		t.Errorf("expected status 'action', got %q", res.Status)
	}
	if len(res.Issues) != 1 {
		t.Errorf("expected 1 issue, got %d", len(res.Issues))
	}
	if !strings.Contains(res.Signal, "BEAD-1") {
		t.Errorf("expected signal to contain 'BEAD-1', got %q", res.Signal)
	}
}

func TestRunBeadsReadyCheck_MultipleReady(t *testing.T) {
	binDir := stubBdBinary(t)
	t.Setenv("PATH", binDir)
	t.Setenv("DUN_BD_READY_OUTPUT", `[{"id":"BEAD-1","title":"First"},{"id":"BEAD-2","title":"Second"},{"id":"BEAD-3","title":"Third"}]`)

	root := t.TempDir()
	res, err := runBeadsReadyCheck(root, CheckDefinition{ID: "beads-ready"})
	if err != nil {
		t.Fatalf("beads ready check: %v", err)
	}
	if res.Status != "action" {
		t.Errorf("expected status 'action', got %q", res.Status)
	}
	if len(res.Issues) != 3 {
		t.Errorf("expected 3 issues, got %d", len(res.Issues))
	}
	if !strings.Contains(res.Signal, "3 ready beads") {
		t.Errorf("expected signal to contain '3 ready beads', got %q", res.Signal)
	}
}

func TestRunBeadsReadyCheck_EmptyArray(t *testing.T) {
	binDir := stubBdBinary(t)
	t.Setenv("PATH", binDir)
	t.Setenv("DUN_BD_READY_OUTPUT", `[]`)

	root := t.TempDir()
	res, err := runBeadsReadyCheck(root, CheckDefinition{ID: "beads-ready"})
	if err != nil {
		t.Fatalf("beads ready check: %v", err)
	}
	if res.Status != "pass" {
		t.Errorf("expected status 'pass', got %q", res.Status)
	}
	if res.Signal != "no ready beads found" {
		t.Errorf("expected signal 'no ready beads found', got %q", res.Signal)
	}
}

func TestRunBeadsReadyCheck_CommandFails(t *testing.T) {
	binDir := stubBdBinary(t)
	t.Setenv("PATH", binDir)
	t.Setenv("DUN_BD_EXIT", "1")

	root := t.TempDir()
	res, err := runBeadsReadyCheck(root, CheckDefinition{ID: "beads-ready"})
	if err != nil {
		t.Fatalf("beads ready check: %v", err)
	}
	if res.Status != "skip" {
		t.Errorf("expected status 'skip', got %q", res.Status)
	}
	if !strings.Contains(res.Signal, "not available") {
		t.Errorf("expected signal to mention 'not available', got %q", res.Signal)
	}
}

func TestRunBeadsReadyCheck_BdNotInstalled(t *testing.T) {
	// Use empty PATH to simulate bd not being installed
	t.Setenv("PATH", "")

	root := t.TempDir()
	res, err := runBeadsReadyCheck(root, CheckDefinition{ID: "beads-ready"})
	if err != nil {
		t.Fatalf("beads ready check: %v", err)
	}
	if res.Status != "skip" {
		t.Errorf("expected status 'skip', got %q", res.Status)
	}
}

func TestRunBeadsReadyCheck_InvalidJSON(t *testing.T) {
	binDir := stubBdBinary(t)
	t.Setenv("PATH", binDir)
	t.Setenv("DUN_BD_READY_OUTPUT", "not valid json")

	root := t.TempDir()
	res, err := runBeadsReadyCheck(root, CheckDefinition{ID: "beads-ready"})
	if err != nil {
		t.Fatalf("beads ready check: %v", err)
	}
	// Should fall back to returning the raw output
	if res.Status != "pass" {
		t.Errorf("expected status 'pass', got %q", res.Status)
	}
	// Output may have trailing newline from shell echo
	if !strings.Contains(res.Signal, "not valid json") {
		t.Errorf("expected signal to contain raw output, got %q", res.Signal)
	}
}

func TestRunBeadsReadyCheck_EmptyOutput(t *testing.T) {
	binDir := stubBdBinary(t)
	t.Setenv("PATH", binDir)
	t.Setenv("DUN_BD_READY_OUTPUT", "")

	root := t.TempDir()
	res, err := runBeadsReadyCheck(root, CheckDefinition{ID: "beads-ready"})
	if err != nil {
		t.Fatalf("beads ready check: %v", err)
	}
	if res.Status != "pass" {
		t.Errorf("expected status 'pass', got %q", res.Status)
	}
	if res.Signal != "no ready beads found" {
		t.Errorf("expected signal 'no ready beads found', got %q", res.Signal)
	}
}

func TestRunBeadsReadyCheck_WhitespaceOnlyOutput(t *testing.T) {
	binDir := stubBdBinary(t)
	t.Setenv("PATH", binDir)
	t.Setenv("DUN_BD_READY_OUTPUT", "   \n\n  ")

	root := t.TempDir()
	res, err := runBeadsReadyCheck(root, CheckDefinition{ID: "beads-ready"})
	if err != nil {
		t.Fatalf("beads ready check: %v", err)
	}
	if res.Status != "pass" {
		t.Errorf("expected status 'pass', got %q", res.Status)
	}
}

// Test runBeadsCriticalPathCheck
func TestRunBeadsCriticalPathCheck_Success(t *testing.T) {
	binDir := stubBdBinary(t)
	t.Setenv("PATH", binDir)
	// Create beads where BEAD-1 blocks BEAD-2 and BEAD-3
	t.Setenv("DUN_BD_BLOCKED_OUTPUT", `[{"id":"BEAD-1","title":"Blocker","blocked_by":[]},{"id":"BEAD-2","title":"Blocked 1","blocked_by":["BEAD-1"]},{"id":"BEAD-3","title":"Blocked 2","blocked_by":["BEAD-1"]}]`)

	root := t.TempDir()
	res, err := runBeadsCriticalPathCheck(root, CheckDefinition{ID: "beads-critical-path"})
	if err != nil {
		t.Fatalf("beads critical path check: %v", err)
	}
	if res.Status != "info" {
		t.Errorf("expected status 'info', got %q", res.Status)
	}
	if !strings.Contains(res.Signal, "critical path") {
		t.Errorf("expected signal to contain 'critical path', got %q", res.Signal)
	}
}

func TestRunBeadsCriticalPathCheck_NoBlocked(t *testing.T) {
	binDir := stubBdBinary(t)
	t.Setenv("PATH", binDir)
	t.Setenv("DUN_BD_BLOCKED_OUTPUT", `[]`)

	root := t.TempDir()
	res, err := runBeadsCriticalPathCheck(root, CheckDefinition{ID: "beads-critical-path"})
	if err != nil {
		t.Fatalf("beads critical path check: %v", err)
	}
	if res.Status != "pass" {
		t.Errorf("expected status 'pass', got %q", res.Status)
	}
	if res.Signal != "no blocked beads" {
		t.Errorf("expected signal 'no blocked beads', got %q", res.Signal)
	}
}

func TestRunBeadsCriticalPathCheck_CommandFails(t *testing.T) {
	binDir := stubBdBinary(t)
	t.Setenv("PATH", binDir)
	t.Setenv("DUN_BD_EXIT", "1")

	root := t.TempDir()
	res, err := runBeadsCriticalPathCheck(root, CheckDefinition{ID: "beads-critical-path"})
	if err != nil {
		t.Fatalf("beads critical path check: %v", err)
	}
	if res.Status != "skip" {
		t.Errorf("expected status 'skip', got %q", res.Status)
	}
}

func TestRunBeadsCriticalPathCheck_InvalidJSON(t *testing.T) {
	binDir := stubBdBinary(t)
	t.Setenv("PATH", binDir)
	t.Setenv("DUN_BD_BLOCKED_OUTPUT", "invalid json")

	root := t.TempDir()
	res, err := runBeadsCriticalPathCheck(root, CheckDefinition{ID: "beads-critical-path"})
	if err != nil {
		t.Fatalf("beads critical path check: %v", err)
	}
	if res.Status != "pass" {
		t.Errorf("expected status 'pass', got %q", res.Status)
	}
	if res.Signal != "no blocked beads" {
		t.Errorf("expected signal 'no blocked beads', got %q", res.Signal)
	}
}

func TestRunBeadsCriticalPathCheck_BdNotInstalled(t *testing.T) {
	t.Setenv("PATH", "")

	root := t.TempDir()
	res, err := runBeadsCriticalPathCheck(root, CheckDefinition{ID: "beads-critical-path"})
	if err != nil {
		t.Fatalf("beads critical path check: %v", err)
	}
	if res.Status != "skip" {
		t.Errorf("expected status 'skip', got %q", res.Status)
	}
}

// Test runBeadsSuggestCheck
func TestRunBeadsSuggestCheck_Success(t *testing.T) {
	binDir := stubBdBinary(t)
	t.Setenv("PATH", binDir)
	t.Setenv("DUN_BD_READY_OUTPUT", `[{"id":"BEAD-1","title":"Low priority","priority":3},{"id":"BEAD-2","title":"High priority","priority":1,"description":"Important task"}]`)

	root := t.TempDir()
	res, err := runBeadsSuggestCheck(root, CheckDefinition{ID: "beads-suggest"})
	if err != nil {
		t.Fatalf("beads suggest check: %v", err)
	}
	if res.Status != "action" {
		t.Errorf("expected status 'action', got %q", res.Status)
	}
	// Should suggest BEAD-2 (higher priority = lower number)
	if !strings.Contains(res.Signal, "BEAD-2") {
		t.Errorf("expected signal to contain 'BEAD-2', got %q", res.Signal)
	}
	if res.Prompt == nil {
		t.Fatal("expected Prompt to be set")
	}
	if res.Prompt.ID != "BEAD-2" {
		t.Errorf("expected Prompt.ID 'BEAD-2', got %q", res.Prompt.ID)
	}
	if res.Prompt.Kind != "bead" {
		t.Errorf("expected Prompt.Kind 'bead', got %q", res.Prompt.Kind)
	}
	if !strings.Contains(res.Prompt.Prompt, "bd show BEAD-2") {
		t.Errorf("expected prompt to include bd show, got %q", res.Prompt.Prompt)
	}
}

func TestRunBeadsSuggestCheck_NoReadyBeads(t *testing.T) {
	binDir := stubBdBinary(t)
	t.Setenv("PATH", binDir)
	t.Setenv("DUN_BD_READY_OUTPUT", `[]`)

	root := t.TempDir()
	res, err := runBeadsSuggestCheck(root, CheckDefinition{ID: "beads-suggest"})
	if err != nil {
		t.Fatalf("beads suggest check: %v", err)
	}
	if res.Status != "pass" {
		t.Errorf("expected status 'pass', got %q", res.Status)
	}
	if res.Next != "beads-critical-path" {
		t.Errorf("expected Next 'beads-critical-path', got %q", res.Next)
	}
	if !strings.Contains(res.Signal, "check critical path") {
		t.Errorf("expected signal to mention 'check critical path', got %q", res.Signal)
	}
}

func TestRunBeadsSuggestCheck_CommandFails(t *testing.T) {
	binDir := stubBdBinary(t)
	t.Setenv("PATH", binDir)
	t.Setenv("DUN_BD_EXIT", "1")

	root := t.TempDir()
	res, err := runBeadsSuggestCheck(root, CheckDefinition{ID: "beads-suggest"})
	if err != nil {
		t.Fatalf("beads suggest check: %v", err)
	}
	if res.Status != "skip" {
		t.Errorf("expected status 'skip', got %q", res.Status)
	}
}

func TestRunBeadsSuggestCheck_InvalidJSON(t *testing.T) {
	binDir := stubBdBinary(t)
	t.Setenv("PATH", binDir)
	t.Setenv("DUN_BD_READY_OUTPUT", "invalid")

	root := t.TempDir()
	res, err := runBeadsSuggestCheck(root, CheckDefinition{ID: "beads-suggest"})
	if err != nil {
		t.Fatalf("beads suggest check: %v", err)
	}
	if res.Status != "pass" {
		t.Errorf("expected status 'pass', got %q", res.Status)
	}
	if res.Next != "beads-critical-path" {
		t.Errorf("expected Next 'beads-critical-path', got %q", res.Next)
	}
}

func TestRunBeadsSuggestCheck_SingleBead(t *testing.T) {
	binDir := stubBdBinary(t)
	t.Setenv("PATH", binDir)
	t.Setenv("DUN_BD_READY_OUTPUT", `[{"id":"BEAD-ONLY","title":"Only bead","priority":5}]`)

	root := t.TempDir()
	res, err := runBeadsSuggestCheck(root, CheckDefinition{ID: "beads-suggest"})
	if err != nil {
		t.Fatalf("beads suggest check: %v", err)
	}
	if res.Status != "action" {
		t.Errorf("expected status 'action', got %q", res.Status)
	}
	if len(res.Issues) != 1 {
		t.Errorf("expected 1 issue, got %d", len(res.Issues))
	}
	if !strings.Contains(res.Signal, "BEAD-ONLY") {
		t.Errorf("expected signal to contain 'BEAD-ONLY', got %q", res.Signal)
	}
}

// Test findCriticalPath
func TestFindCriticalPath_Basic(t *testing.T) {
	// BEAD-1 blocks BEAD-2 and BEAD-3
	// BEAD-2 blocks BEAD-4
	issues := []beadsIssue{
		{ID: "BEAD-1", Title: "Root blocker"},
		{ID: "BEAD-2", Title: "Blocked by 1", BlockedBy: []string{"BEAD-1"}},
		{ID: "BEAD-3", Title: "Blocked by 1", BlockedBy: []string{"BEAD-1"}},
		{ID: "BEAD-4", Title: "Blocked by 2", BlockedBy: []string{"BEAD-2"}},
	}

	path := findCriticalPath(issues)

	// BEAD-1 should be in the critical path (blocks 2 beads)
	found := false
	for _, p := range path {
		if p.ID == "BEAD-1" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected BEAD-1 to be in critical path")
	}
}

func TestFindCriticalPath_Empty(t *testing.T) {
	path := findCriticalPath([]beadsIssue{})

	if len(path) != 0 {
		t.Errorf("expected empty critical path, got %d items", len(path))
	}
}

func TestFindCriticalPath_NoBlockers(t *testing.T) {
	issues := []beadsIssue{
		{ID: "BEAD-1", Title: "Independent 1"},
		{ID: "BEAD-2", Title: "Independent 2"},
	}

	path := findCriticalPath(issues)

	if len(path) != 0 {
		t.Errorf("expected empty critical path for independent beads, got %d items", len(path))
	}
}

func TestFindCriticalPath_LimitsToFive(t *testing.T) {
	// Create more than 5 distinct blockers to trigger the limit
	issues := []beadsIssue{
		// 6 different blockers, each blocking at least one bead
		{ID: "BEAD-1"},
		{ID: "BEAD-2"},
		{ID: "BEAD-3"},
		{ID: "BEAD-4"},
		{ID: "BEAD-5"},
		{ID: "BEAD-6"},
		{ID: "BEAD-7"},
		// Beads blocked by the 7 blockers above
		{ID: "BEAD-A", BlockedBy: []string{"BEAD-1"}},
		{ID: "BEAD-B", BlockedBy: []string{"BEAD-2"}},
		{ID: "BEAD-C", BlockedBy: []string{"BEAD-3"}},
		{ID: "BEAD-D", BlockedBy: []string{"BEAD-4"}},
		{ID: "BEAD-E", BlockedBy: []string{"BEAD-5"}},
		{ID: "BEAD-F", BlockedBy: []string{"BEAD-6"}},
		{ID: "BEAD-G", BlockedBy: []string{"BEAD-7"}},
	}

	path := findCriticalPath(issues)

	// There are 7 blockers, but should be limited to 5
	if len(path) > 5 {
		t.Errorf("expected at most 5 items in critical path, got %d", len(path))
	}
	if len(path) != 5 {
		t.Errorf("expected exactly 5 items (limited from 7 blockers), got %d", len(path))
	}
}

func TestFindCriticalPath_BlockerNotInList(t *testing.T) {
	// BEAD-2 is blocked by BEAD-EXTERNAL which is not in the list
	issues := []beadsIssue{
		{ID: "BEAD-1", Title: "In list"},
		{ID: "BEAD-2", Title: "Blocked by external", BlockedBy: []string{"BEAD-EXTERNAL"}},
	}

	path := findCriticalPath(issues)

	// BEAD-EXTERNAL won't be in the path because it's not in the issueMap
	for _, p := range path {
		if p.ID == "BEAD-EXTERNAL" {
			t.Error("should not include external blocker not in issue list")
		}
	}
}

// Test suggestNextBead
func TestSuggestNextBead_PicksHighestPriority(t *testing.T) {
	issues := []beadsIssue{
		{ID: "BEAD-LOW", Priority: 10},
		{ID: "BEAD-HIGH", Priority: 1},
		{ID: "BEAD-MED", Priority: 5},
	}

	suggested := suggestNextBead(issues)

	if suggested.ID != "BEAD-HIGH" {
		t.Errorf("expected BEAD-HIGH (priority 1), got %q (priority %d)", suggested.ID, suggested.Priority)
	}
}

func TestSuggestNextBead_Empty(t *testing.T) {
	suggested := suggestNextBead([]beadsIssue{})

	if suggested.ID != "" {
		t.Errorf("expected empty bead for empty input, got %q", suggested.ID)
	}
}

func TestSuggestNextBead_Single(t *testing.T) {
	issues := []beadsIssue{
		{ID: "BEAD-ONLY", Priority: 3},
	}

	suggested := suggestNextBead(issues)

	if suggested.ID != "BEAD-ONLY" {
		t.Errorf("expected BEAD-ONLY, got %q", suggested.ID)
	}
}

func TestSuggestNextBead_SamePriority(t *testing.T) {
	issues := []beadsIssue{
		{ID: "BEAD-FIRST", Priority: 5},
		{ID: "BEAD-SECOND", Priority: 5},
	}

	suggested := suggestNextBead(issues)

	// Should return the first one when priorities are equal
	if suggested.ID != "BEAD-FIRST" {
		t.Errorf("expected BEAD-FIRST (first with same priority), got %q", suggested.ID)
	}
}

func TestSuggestNextBead_ZeroPriority(t *testing.T) {
	issues := []beadsIssue{
		{ID: "BEAD-NONZERO", Priority: 5},
		{ID: "BEAD-ZERO", Priority: 0}, // 0 is highest priority
	}

	suggested := suggestNextBead(issues)

	if suggested.ID != "BEAD-ZERO" {
		t.Errorf("expected BEAD-ZERO (priority 0 is highest), got %q", suggested.ID)
	}
}

func TestSuggestNextBead_NegativePriority(t *testing.T) {
	issues := []beadsIssue{
		{ID: "BEAD-ZERO", Priority: 0},
		{ID: "BEAD-NEGATIVE", Priority: -1}, // Negative would be even higher priority
	}

	suggested := suggestNextBead(issues)

	if suggested.ID != "BEAD-NEGATIVE" {
		t.Errorf("expected BEAD-NEGATIVE (lowest number = highest priority), got %q", suggested.ID)
	}
}

// Test formatReadySignal
func TestFormatReadySignal_Empty(t *testing.T) {
	signal := formatReadySignal([]beadsIssue{})

	if signal != "no ready beads" {
		t.Errorf("expected 'no ready beads', got %q", signal)
	}
}

func TestFormatReadySignal_Single(t *testing.T) {
	issues := []beadsIssue{
		{ID: "BEAD-123"},
	}

	signal := formatReadySignal(issues)

	if signal != "1 ready bead: BEAD-123" {
		t.Errorf("expected '1 ready bead: BEAD-123', got %q", signal)
	}
}

func TestFormatReadySignal_Multiple(t *testing.T) {
	issues := []beadsIssue{
		{ID: "BEAD-1"},
		{ID: "BEAD-2"},
		{ID: "BEAD-3"},
	}

	signal := formatReadySignal(issues)

	if !strings.HasPrefix(signal, "3 ready beads: ") {
		t.Errorf("expected signal to start with '3 ready beads: ', got %q", signal)
	}
	if !strings.Contains(signal, "BEAD-1") {
		t.Errorf("expected signal to contain 'BEAD-1', got %q", signal)
	}
	if !strings.Contains(signal, "BEAD-2") {
		t.Errorf("expected signal to contain 'BEAD-2', got %q", signal)
	}
	if !strings.Contains(signal, "BEAD-3") {
		t.Errorf("expected signal to contain 'BEAD-3', got %q", signal)
	}
}

// Test formatCriticalPathSignal
func TestFormatCriticalPathSignal_Empty(t *testing.T) {
	signal := formatCriticalPathSignal([]beadsIssue{})

	if signal != "no critical path blockers" {
		t.Errorf("expected 'no critical path blockers', got %q", signal)
	}
}

func TestFormatCriticalPathSignal_Single(t *testing.T) {
	issues := []beadsIssue{
		{ID: "BEAD-BLOCKER"},
	}

	signal := formatCriticalPathSignal(issues)

	if signal != "critical path: BEAD-BLOCKER" {
		t.Errorf("expected 'critical path: BEAD-BLOCKER', got %q", signal)
	}
}

func TestFormatCriticalPathSignal_Multiple(t *testing.T) {
	issues := []beadsIssue{
		{ID: "BEAD-1"},
		{ID: "BEAD-2"},
		{ID: "BEAD-3"},
	}

	signal := formatCriticalPathSignal(issues)

	expected := "critical path: BEAD-1 \u2192 BEAD-2 \u2192 BEAD-3"
	if signal != expected {
		t.Errorf("expected %q, got %q", expected, signal)
	}
}

// Test formatSuggestion
func TestFormatSuggestion_Empty(t *testing.T) {
	signal := formatSuggestion(beadsIssue{})

	if signal != "no suggestion" {
		t.Errorf("expected 'no suggestion', got %q", signal)
	}
}

func TestFormatSuggestion_Complete(t *testing.T) {
	issue := beadsIssue{
		ID:       "BEAD-456",
		Title:    "Implement feature",
		Priority: 2,
	}

	signal := formatSuggestion(issue)

	if !strings.Contains(signal, "BEAD-456") {
		t.Errorf("expected signal to contain 'BEAD-456', got %q", signal)
	}
	if !strings.Contains(signal, "[P2]") {
		t.Errorf("expected signal to contain '[P2]', got %q", signal)
	}
	if !strings.Contains(signal, "Implement feature") {
		t.Errorf("expected signal to contain 'Implement feature', got %q", signal)
	}
}

func TestFormatSuggestion_ZeroPriority(t *testing.T) {
	issue := beadsIssue{
		ID:       "BEAD-789",
		Title:    "Urgent task",
		Priority: 0,
	}

	signal := formatSuggestion(issue)

	if !strings.Contains(signal, "[P0]") {
		t.Errorf("expected signal to contain '[P0]', got %q", signal)
	}
}

func TestFormatSuggestion_EmptyTitle(t *testing.T) {
	issue := beadsIssue{
		ID:       "BEAD-NOTITLE",
		Priority: 1,
	}

	signal := formatSuggestion(issue)

	expected := "suggested: BEAD-NOTITLE [P1] "
	if signal != expected {
		t.Errorf("expected %q, got %q", expected, signal)
	}
}

// Integration-style tests

func TestBeadsCheckFlow_ReadyToSuggest(t *testing.T) {
	binDir := stubBdBinary(t)
	t.Setenv("PATH", binDir)

	// Multiple ready beads with different priorities
	t.Setenv("DUN_BD_READY_OUTPUT", `[
		{"id":"BEAD-LOW","title":"Low priority task","priority":10},
		{"id":"BEAD-URGENT","title":"Urgent task","priority":0,"description":"This is critical"},
		{"id":"BEAD-NORMAL","title":"Normal task","priority":5}
	]`)

	root := t.TempDir()

	// First check ready beads
	readyRes, err := runBeadsReadyCheck(root, CheckDefinition{ID: "beads-ready"})
	if err != nil {
		t.Fatalf("ready check: %v", err)
	}
	if readyRes.Status != "action" {
		t.Errorf("expected ready status 'action', got %q", readyRes.Status)
	}
	if len(readyRes.Issues) != 3 {
		t.Errorf("expected 3 ready issues, got %d", len(readyRes.Issues))
	}

	// Then get suggestion (should pick BEAD-URGENT with priority 0)
	suggestRes, err := runBeadsSuggestCheck(root, CheckDefinition{ID: "beads-suggest"})
	if err != nil {
		t.Fatalf("suggest check: %v", err)
	}
	if suggestRes.Status != "action" {
		t.Errorf("expected suggest status 'action', got %q", suggestRes.Status)
	}
	if suggestRes.Prompt == nil {
		t.Fatal("expected Prompt to be set")
	}
	if suggestRes.Prompt.ID != "BEAD-URGENT" {
		t.Errorf("expected suggested BEAD-URGENT, got %q", suggestRes.Prompt.ID)
	}
}

func TestBeadsCheckFlow_CriticalPathWhenBlocked(t *testing.T) {
	binDir := stubBdBinary(t)
	t.Setenv("PATH", binDir)

	// No ready beads, but blocked beads exist
	t.Setenv("DUN_BD_READY_OUTPUT", `[]`)
	t.Setenv("DUN_BD_BLOCKED_OUTPUT", `[
		{"id":"BEAD-ROOT","title":"Root blocker"},
		{"id":"BEAD-A","title":"Blocked A","blocked_by":["BEAD-ROOT"]},
		{"id":"BEAD-B","title":"Blocked B","blocked_by":["BEAD-ROOT"]}
	]`)

	root := t.TempDir()

	// Suggest should point to critical path
	suggestRes, err := runBeadsSuggestCheck(root, CheckDefinition{ID: "beads-suggest"})
	if err != nil {
		t.Fatalf("suggest check: %v", err)
	}
	if suggestRes.Status != "pass" {
		t.Errorf("expected suggest status 'pass', got %q", suggestRes.Status)
	}
	if suggestRes.Next != "beads-critical-path" {
		t.Errorf("expected Next 'beads-critical-path', got %q", suggestRes.Next)
	}

	// Critical path should identify ROOT as the blocker
	critRes, err := runBeadsCriticalPathCheck(root, CheckDefinition{ID: "beads-critical-path"})
	if err != nil {
		t.Fatalf("critical path check: %v", err)
	}
	if critRes.Status != "info" {
		t.Errorf("expected critical path status 'info', got %q", critRes.Status)
	}
	if !strings.Contains(critRes.Signal, "BEAD-ROOT") {
		t.Errorf("expected critical path to contain 'BEAD-ROOT', got %q", critRes.Signal)
	}
}

// Edge case tests

func TestRunBeadsReadyCheck_MalformedJSONArray(t *testing.T) {
	binDir := stubBdBinary(t)
	t.Setenv("PATH", binDir)
	// Valid JSON but not an array of beadsIssue
	t.Setenv("DUN_BD_READY_OUTPUT", `{"error": "not an array"}`)

	root := t.TempDir()
	res, err := runBeadsReadyCheck(root, CheckDefinition{ID: "beads-ready"})
	if err != nil {
		t.Fatalf("beads ready check: %v", err)
	}
	// Should fall back to pass with raw output as signal
	if res.Status != "pass" {
		t.Errorf("expected status 'pass', got %q", res.Status)
	}
}

func TestRunBeadsSuggestCheck_PromptFields(t *testing.T) {
	binDir := stubBdBinary(t)
	t.Setenv("PATH", binDir)
	t.Setenv("DUN_BD_READY_OUTPUT", `[{"id":"BEAD-TEST","title":"Test Title","description":"Test Description","priority":2}]`)

	root := t.TempDir()
	res, err := runBeadsSuggestCheck(root, CheckDefinition{ID: "beads-suggest"})
	if err != nil {
		t.Fatalf("beads suggest check: %v", err)
	}

	if res.Prompt == nil {
		t.Fatal("expected Prompt to be set")
	}
	if res.Prompt.Kind != "bead" {
		t.Errorf("expected Kind 'bead', got %q", res.Prompt.Kind)
	}
	if res.Prompt.ID != "BEAD-TEST" {
		t.Errorf("expected ID 'BEAD-TEST', got %q", res.Prompt.ID)
	}
	if res.Prompt.Title != "Test Title" {
		t.Errorf("expected Title 'Test Title', got %q", res.Prompt.Title)
	}
	if res.Prompt.Summary != "Test Description" {
		t.Errorf("expected Summary 'Test Description', got %q", res.Prompt.Summary)
	}
	if !strings.Contains(res.Prompt.Prompt, "BEAD-TEST") {
		t.Errorf("expected Prompt to contain 'BEAD-TEST', got %q", res.Prompt.Prompt)
	}
}

func TestFindCriticalPath_ComplexDependencies(t *testing.T) {
	// Create a complex dependency graph:
	// BEAD-1 blocks BEAD-2, BEAD-3, BEAD-4 (3 dependents)
	// BEAD-2 blocks BEAD-5, BEAD-6 (2 dependents)
	// BEAD-3 blocks BEAD-7 (1 dependent)
	issues := []beadsIssue{
		{ID: "BEAD-1"},
		{ID: "BEAD-2", BlockedBy: []string{"BEAD-1"}},
		{ID: "BEAD-3", BlockedBy: []string{"BEAD-1"}},
		{ID: "BEAD-4", BlockedBy: []string{"BEAD-1"}},
		{ID: "BEAD-5", BlockedBy: []string{"BEAD-2"}},
		{ID: "BEAD-6", BlockedBy: []string{"BEAD-2"}},
		{ID: "BEAD-7", BlockedBy: []string{"BEAD-3"}},
	}

	path := findCriticalPath(issues)

	// Should include BEAD-1 (blocks 3) and BEAD-2 (blocks 2)
	ids := make(map[string]bool)
	for _, p := range path {
		ids[p.ID] = true
	}

	if !ids["BEAD-1"] {
		t.Error("expected BEAD-1 in critical path (blocks 3 beads)")
	}
	if !ids["BEAD-2"] {
		t.Error("expected BEAD-2 in critical path (blocks 2 beads)")
	}
}
