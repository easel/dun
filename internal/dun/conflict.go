package dun

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"
)

// Resolution is the outcome of quorum evaluation.
type Resolution struct {
	Outcome  string          `json:"outcome"` // "accepted", "skipped", "aborted"
	Response string          `json:"response,omitempty"`
	Conflict *ConflictReport `json:"conflict,omitempty"`
	Reason   string          `json:"reason"`
}

// ConflictReport contains details about a quorum conflict.
type ConflictReport struct {
	TaskID    string          `json:"task_id"`
	Timestamp time.Time       `json:"timestamp"`
	Groups    []ResponseGroup `json:"groups"`
	Diffs     []GroupDiff     `json:"diffs"`
	Harnesses []string        `json:"harnesses"`
	Quorum    QuorumConfig    `json:"quorum"`
}

// GroupDiff contains the unified diff between two response groups.
type GroupDiff struct {
	GroupA  int    `json:"group_a"`
	GroupB  int    `json:"group_b"`
	Unified string `json:"unified_diff"`
}

// ConflictResolver handles disagreements between harness responses.
type ConflictResolver struct {
	escalate bool
	prefer   string
	stdin    io.Reader
	stdout   io.Writer
}

// NewConflictResolver creates a ConflictResolver with the given options.
func NewConflictResolver(escalate bool, prefer string, stdin io.Reader, stdout io.Writer) *ConflictResolver {
	return &ConflictResolver{
		escalate: escalate,
		prefer:   prefer,
		stdin:    stdin,
		stdout:   stdout,
	}
}

// Resolve determines the outcome based on response groups and quorum configuration.
func (cr *ConflictResolver) Resolve(groups []ResponseGroup, config QuorumConfig) Resolution {
	if len(groups) == 0 {
		return Resolution{
			Outcome: "skipped",
			Reason:  "no response groups",
		}
	}

	largestGroup := groups[0]

	// Check if quorum is met with largest group
	if config.IsMet(len(largestGroup.Members), config.TotalHarnesses) {
		return Resolution{
			Outcome:  "accepted",
			Response: largestGroup.Canonical,
			Reason:   fmt.Sprintf("quorum met: %d/%d agree", len(largestGroup.Members), config.TotalHarnesses),
		}
	}

	// Quorum not met - build conflict report
	conflict := cr.buildConflictReport(groups, config)

	// Strategy 1: Escalate to human
	if cr.escalate {
		return cr.humanReview(conflict)
	}

	// Strategy 2: Prefer specific harness
	if cr.prefer != "" {
		return cr.preferredHarness(groups, cr.prefer, &conflict)
	}

	// Strategy 3: Default - skip task
	return Resolution{
		Outcome:  "skipped",
		Conflict: &conflict,
		Reason:   "quorum not met, no escalation configured",
	}
}

// humanReview prompts the user to select a response.
func (cr *ConflictResolver) humanReview(conflict ConflictReport) Resolution {
	fmt.Fprintf(cr.stdout, "\n=== QUORUM CONFLICT ===\n")
	fmt.Fprintf(cr.stdout, "Task: %s\n\n", conflict.TaskID)

	for i, group := range conflict.Groups {
		fmt.Fprintf(cr.stdout, "Option %d (%d harnesses: %s):\n",
			i+1, len(group.Members), harnessNames(group.Members))
		fmt.Fprintf(cr.stdout, "```\n%s\n```\n\n", truncate(group.Canonical, 500))
	}

	fmt.Fprintf(cr.stdout, "Enter choice (1-%d), 's' to skip, 'q' to quit: ", len(conflict.Groups))

	scanner := bufio.NewScanner(cr.stdin)
	if !scanner.Scan() {
		return Resolution{
			Outcome:  "skipped",
			Conflict: &conflict,
			Reason:   "no input received",
		}
	}

	choice := strings.TrimSpace(scanner.Text())

	switch choice {
	case "s":
		return Resolution{
			Outcome:  "skipped",
			Conflict: &conflict,
			Reason:   "user skipped",
		}
	case "q":
		return Resolution{
			Outcome:  "aborted",
			Conflict: &conflict,
			Reason:   "user quit",
		}
	default:
		idx, err := strconv.Atoi(choice)
		if err != nil || idx < 1 || idx > len(conflict.Groups) {
			return Resolution{
				Outcome:  "skipped",
				Conflict: &conflict,
				Reason:   "invalid choice",
			}
		}
		return Resolution{
			Outcome:  "accepted",
			Response: conflict.Groups[idx-1].Canonical,
			Conflict: &conflict,
			Reason:   "user selected option " + choice,
		}
	}
}

// preferredHarness returns the response from the preferred harness.
func (cr *ConflictResolver) preferredHarness(groups []ResponseGroup, prefer string, conflict *ConflictReport) Resolution {
	for _, group := range groups {
		for _, member := range group.Members {
			if member.Harness == prefer {
				return Resolution{
					Outcome:  "accepted",
					Response: member.Response,
					Conflict: conflict,
					Reason:   fmt.Sprintf("using preferred harness: %s", prefer),
				}
			}
		}
	}

	return Resolution{
		Outcome:  "skipped",
		Conflict: conflict,
		Reason:   fmt.Sprintf("preferred harness %s not found in responses", prefer),
	}
}

// buildConflictReport generates a detailed conflict report.
func (cr *ConflictResolver) buildConflictReport(groups []ResponseGroup, config QuorumConfig) ConflictReport {
	report := ConflictReport{
		TaskID:    "", // Will be set by caller if needed
		Timestamp: time.Now(),
		Groups:    groups,
		Quorum:    config,
	}

	// Collect all harness names
	seen := make(map[string]bool)
	for _, group := range groups {
		for _, member := range group.Members {
			if !seen[member.Harness] {
				seen[member.Harness] = true
				report.Harnesses = append(report.Harnesses, member.Harness)
			}
		}
	}

	// Generate diffs between groups
	report.Diffs = cr.generateDiffs(groups)

	return report
}

// generateDiffs creates unified diffs between all pairs of response groups.
func (cr *ConflictResolver) generateDiffs(groups []ResponseGroup) []GroupDiff {
	var diffs []GroupDiff

	for i := 0; i < len(groups); i++ {
		for j := i + 1; j < len(groups); j++ {
			diff := GroupDiff{
				GroupA:  i,
				GroupB:  j,
				Unified: unifiedDiff(groups[i].Canonical, groups[j].Canonical),
			}
			diffs = append(diffs, diff)
		}
	}

	return diffs
}

// harnessNames extracts harness names from a list of results.
func harnessNames(members []HarnessResult) string {
	names := make([]string, 0, len(members))
	for _, m := range members {
		names = append(names, m.Harness)
	}
	return strings.Join(names, ", ")
}

// truncate shortens a string to maxLen, adding "..." if truncated.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// unifiedDiff generates a simple unified diff between two strings.
func unifiedDiff(a, b string) string {
	linesA := strings.Split(a, "\n")
	linesB := strings.Split(b, "\n")

	var result strings.Builder
	result.WriteString("--- a\n")
	result.WriteString("+++ b\n")

	// Simple line-by-line diff (not a true unified diff, but sufficient for display)
	maxLines := len(linesA)
	if len(linesB) > maxLines {
		maxLines = len(linesB)
	}

	for i := 0; i < maxLines; i++ {
		lineA := ""
		lineB := ""
		if i < len(linesA) {
			lineA = linesA[i]
		}
		if i < len(linesB) {
			lineB = linesB[i]
		}

		if lineA != lineB {
			if lineA != "" {
				result.WriteString("- ")
				result.WriteString(lineA)
				result.WriteString("\n")
			}
			if lineB != "" {
				result.WriteString("+ ")
				result.WriteString(lineB)
				result.WriteString("\n")
			}
		} else if lineA != "" {
			result.WriteString("  ")
			result.WriteString(lineA)
			result.WriteString("\n")
		}
	}

	return result.String()
}
