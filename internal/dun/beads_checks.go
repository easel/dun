package dun

import (
	"encoding/json"
	"os/exec"
	"strconv"
	"strings"
)

// beadsIssue is the internal representation of a beads issue
type beadsIssue struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Status      string   `json:"status"`
	Priority    string   `json:"priority"`
	Labels      []string `json:"labels"`
	BlockedBy   []string `json:"blocked_by"`
	Blocks      []string `json:"blocks"`
	Description string   `json:"description,omitempty"`
}

// toIssue converts a beadsIssue to the dun Issue type
func (b beadsIssue) toIssue() Issue {
	return Issue{
		ID:      b.ID,
		Summary: b.Title,
	}
}

// toIssues converts a slice of beadsIssues to dun Issues
func toIssues(beads []beadsIssue) []Issue {
	issues := make([]Issue, len(beads))
	for i, b := range beads {
		issues[i] = b.toIssue()
	}
	return issues
}

// runBeadsReadyCheck finds workable beads (no blockers, not in progress)
func runBeadsReadyCheck(root string, check Check) (CheckResult, error) {
	// Run bd ready to get workable beads
	cmd := exec.Command("bd", "ready", "--json")
	cmd.Dir = root
	output, err := cmd.Output()
	if err != nil {
		// bd ready might not exist or fail - that's ok
		return CheckResult{
			ID:     check.ID,
			Status: "skip",
			Signal: "beads not available or no ready issues",
		}, nil
	}

	var issues []beadsIssue
	if err := json.Unmarshal(output, &issues); err != nil {
		// Try line-by-line parsing if not JSON array
		lines := strings.Split(strings.TrimSpace(string(output)), "\n")
		if len(lines) == 0 || (len(lines) == 1 && lines[0] == "") {
			return CheckResult{
				ID:     check.ID,
				Status: "pass",
				Signal: "no ready beads found",
			}, nil
		}
		return CheckResult{
			ID:     check.ID,
			Status: "pass",
			Signal: string(output),
		}, nil
	}

	if len(issues) == 0 {
		return CheckResult{
			ID:     check.ID,
			Status: "pass",
			Signal: "no ready beads found",
		}, nil
	}

	return CheckResult{
		ID:     check.ID,
		Status: "action",
		Signal: formatReadySignal(issues),
		Issues: toIssues(issues),
	}, nil
}

// runBeadsCriticalPathCheck identifies the critical path through blocked beads
func runBeadsCriticalPathCheck(root string, check Check) (CheckResult, error) {
	// Run bd blocked to get blocked beads
	cmd := exec.Command("bd", "blocked", "--json")
	cmd.Dir = root
	output, err := cmd.Output()
	if err != nil {
		return CheckResult{
			ID:     check.ID,
			Status: "skip",
			Signal: "beads not available or no blocked issues",
		}, nil
	}

	var issues []beadsIssue
	if err := json.Unmarshal(output, &issues); err != nil {
		return CheckResult{
			ID:     check.ID,
			Status: "pass",
			Signal: "no blocked beads",
		}, nil
	}

	if len(issues) == 0 {
		return CheckResult{
			ID:     check.ID,
			Status: "pass",
			Signal: "no blocked beads",
		}, nil
	}

	// Find the critical path - beads that block the most other beads
	criticalPath := findCriticalPath(issues)

	return CheckResult{
		ID:     check.ID,
		Status: "info",
		Signal: formatCriticalPathSignal(criticalPath),
		Issues: toIssues(criticalPath),
	}, nil
}

// runBeadsSuggestCheck suggests the next bead to work on
func runBeadsSuggestCheck(root string, check Check) (CheckResult, error) {
	// Get ready beads first
	cmd := exec.Command("bd", "ready", "--json")
	cmd.Dir = root
	output, err := cmd.Output()
	if err != nil {
		return CheckResult{
			ID:     check.ID,
			Status: "skip",
			Signal: "beads not available",
		}, nil
	}

	var issues []beadsIssue
	if err := json.Unmarshal(output, &issues); err != nil || len(issues) == 0 {
		// No ready beads - check what's blocked
		return CheckResult{
			ID:     check.ID,
			Status: "pass",
			Signal: "no workable beads - check critical path",
			Next:   "beads-critical-path",
		}, nil
	}

	// Find highest priority ready bead
	suggested := suggestNextBead(issues)

	return CheckResult{
		ID:     check.ID,
		Status: "action",
		Signal: formatSuggestion(suggested),
		Issues: toIssues([]beadsIssue{suggested}),
		Prompt: &PromptEnvelope{
			Kind:    "bead",
			ID:      suggested.ID,
			Title:   suggested.Title,
			Summary: suggested.Description,
			Prompt:  "Work on this bead: " + suggested.ID + " - " + suggested.Title,
		},
	}, nil
}

// findCriticalPath finds beads that block the most other beads
func findCriticalPath(issues []beadsIssue) []beadsIssue {
	// Count how many beads each bead blocks
	blockCount := make(map[string]int)
	issueMap := make(map[string]beadsIssue)

	for _, issue := range issues {
		issueMap[issue.ID] = issue
		for _, blocked := range issue.BlockedBy {
			blockCount[blocked]++
		}
	}

	// Sort by block count (most blocking first)
	var critical []beadsIssue
	for id, count := range blockCount {
		if issue, ok := issueMap[id]; ok && count > 0 {
			critical = append(critical, issue)
		}
	}

	// Limit to top 5
	if len(critical) > 5 {
		critical = critical[:5]
	}

	return critical
}

// suggestNextBead picks the best bead to work on next
func suggestNextBead(issues []beadsIssue) beadsIssue {
	if len(issues) == 0 {
		return beadsIssue{}
	}

	// Priority order: P0 > P1 > P2 > P3
	priorityOrder := map[string]int{"P0": 0, "P1": 1, "P2": 2, "P3": 3, "": 4}

	best := issues[0]
	bestPriority := priorityOrder[best.Priority]

	for _, issue := range issues[1:] {
		p := priorityOrder[issue.Priority]
		if p < bestPriority {
			best = issue
			bestPriority = p
		}
	}

	return best
}

func formatReadySignal(issues []beadsIssue) string {
	if len(issues) == 0 {
		return "no ready beads"
	}
	if len(issues) == 1 {
		return "1 ready bead: " + issues[0].ID
	}
	ids := make([]string, len(issues))
	for i, issue := range issues {
		ids[i] = issue.ID
	}
	return strconv.Itoa(len(issues)) + " ready beads: " + strings.Join(ids, ", ")
}

func formatCriticalPathSignal(issues []beadsIssue) string {
	if len(issues) == 0 {
		return "no critical path blockers"
	}
	ids := make([]string, len(issues))
	for i, issue := range issues {
		ids[i] = issue.ID
	}
	return "critical path: " + strings.Join(ids, " → ")
}

func formatSuggestion(issue beadsIssue) string {
	if issue.ID == "" {
		return "no suggestion"
	}
	return "suggested: " + issue.ID + " [" + issue.Priority + "] " + issue.Title
}
