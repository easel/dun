package main

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"io"
	"os/exec"
	"strconv"
	"strings"

	"github.com/easel/dun/internal/dun"
)

const (
	maxTasksPerCategory = 10
	maxTaskSummaryBytes = 200
	maxTaskReasonBytes  = 160
	taskIDSeparator     = "#"
)

type taskItem struct {
	ID      string
	Summary string
	Why     string
}

type taskGroup struct {
	Check           dun.CheckResult
	Tasks           []taskItem
	Total           int
	PromptAvailable bool
}

type taskRef struct {
	CheckID    string
	IssueIndex int
	State      string
}

var repoStateHashFn = repoStateHash

func runTask(args []string, stdout io.Writer, stderr io.Writer) int {
	root := resolveRoot(".")
	fs := flag.NewFlagSet("task", flag.ContinueOnError)
	fs.SetOutput(stderr)
	configPath := fs.String("config", "", "path to config file (default .dun/config.yaml if present; also loads user config)")
	printPrompt := fs.Bool("prompt", false, "print the full task prompt (if available)")
	flagArgs, positionals, err := splitTaskArgs(args)
	if err != nil {
		fmt.Fprintln(stderr, err.Error())
		return dun.ExitUsageError
	}
	if err := fs.Parse(flagArgs); err != nil {
		return dun.ExitUsageError
	}

	if len(positionals) < 1 {
		fmt.Fprintln(stderr, "usage: dun task <task-id> [--prompt]")
		return dun.ExitUsageError
	}

	ref, err := parseTaskID(positionals[0])
	if err != nil {
		fmt.Fprintf(stderr, "invalid task id: %v\n", err)
		return dun.ExitUsageError
	}

	currentState := repoStateHashFn(root)
	if ref.State == "" {
		fmt.Fprintln(stderr, "invalid task id: missing repo state hash")
		return dun.ExitUsageError
	}
	if currentState != "" && ref.State != currentState {
		fmt.Fprintf(stderr, "task id is stale (expected state %s)\n", currentState)
		return dun.ExitCheckFailed
	}

	opts := dun.DefaultOptions()
	cfg, loaded, err := dun.LoadConfig(root, *configPath)
	if err != nil {
		fmt.Fprintf(stderr, "dun task failed: config error: %v\n", err)
		return dun.ExitConfigError
	}
	if loaded {
		opts = dun.ApplyConfig(opts, cfg)
	}
	opts.AgentMode = "prompt"

	result, err := checkRepo(root, opts)
	if err != nil {
		fmt.Fprintf(stderr, "dun task failed: %v\n", err)
		return dun.ExitCheckFailed
	}

	var match *dun.CheckResult
	for i := range result.Checks {
		if result.Checks[i].ID == ref.CheckID {
			match = &result.Checks[i]
			break
		}
	}
	if match == nil {
		fmt.Fprintf(stderr, "unknown check for task: %s\n", ref.CheckID)
		return dun.ExitCheckFailed
	}

	if *printPrompt {
		if match.Prompt == nil || strings.TrimSpace(match.Prompt.Prompt) == "" {
			fmt.Fprintf(stderr, "no prompt available for task %s\n", ref.CheckID)
			return dun.ExitCheckFailed
		}
		fmt.Fprintln(stdout, match.Prompt.Prompt)
		return dun.ExitSuccess
	}

	summary := taskSummaryForCheck(*match)
	if ref.IssueIndex > 0 {
		if ref.IssueIndex > len(match.Issues) {
			fmt.Fprintf(stderr, "task %s refers to issue %d but only %d issues are available\n", ref.CheckID, ref.IssueIndex, len(match.Issues))
			return dun.ExitCheckFailed
		}
		issue := match.Issues[ref.IssueIndex-1]
		summary = issueSummary(issue)
	}
	fmt.Fprintf(stdout, "task: %s\n", positionals[0])
	fmt.Fprintf(stdout, "check: %s\n", match.ID)
	fmt.Fprintf(stdout, "status: %s\n", match.Status)
	if match.Signal != "" {
		fmt.Fprintf(stdout, "signal: %s\n", match.Signal)
	}
	if match.Detail != "" {
		fmt.Fprintf(stdout, "detail: %s\n", match.Detail)
	}
	fmt.Fprintf(stdout, "summary: %s\n", summary)
	if match.Next != "" {
		fmt.Fprintf(stdout, "next: %s\n", match.Next)
	}
	if match.Prompt != nil {
		fmt.Fprintf(stdout, "prompt: available (run `dun task %s --prompt`)\n", taskIDForCheck(match.ID, ref.State))
	}
	return dun.ExitSuccess
}

func parseTaskID(taskID string) (taskRef, error) {
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return taskRef{}, errors.New("empty task id")
	}
	stateSplit := strings.Split(taskID, "@")
	if len(stateSplit) != 2 {
		return taskRef{}, errors.New("task id must include @<state>")
	}
	base := strings.TrimSpace(stateSplit[0])
	state := strings.TrimSpace(stateSplit[1])
	if base == "" || state == "" {
		return taskRef{}, errors.New("task id missing base or state")
	}

	parts := strings.SplitN(base, taskIDSeparator, 2)
	ref := taskRef{CheckID: parts[0], State: state}
	if ref.CheckID == "" {
		return taskRef{}, errors.New("missing check id")
	}
	if len(parts) == 1 {
		return ref, nil
	}
	if parts[1] == "" {
		return taskRef{}, errors.New("missing issue index")
	}
	idx, err := strconv.Atoi(parts[1])
	if err != nil || idx <= 0 {
		return taskRef{}, errors.New("issue index must be a positive integer")
	}
	ref.IssueIndex = idx
	return ref, nil
}

func splitTaskArgs(args []string) ([]string, []string, error) {
	if len(args) == 0 {
		return nil, nil, nil
	}
	var flagArgs []string
	var positionals []string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if strings.HasPrefix(arg, "-") {
			flagArgs = append(flagArgs, arg)
			if arg == "--config" {
				if i+1 >= len(args) {
					return nil, nil, errors.New("missing value for --config")
				}
				flagArgs = append(flagArgs, args[i+1])
				i++
			}
			continue
		}
		positionals = append(positionals, arg)
	}
	return flagArgs, positionals, nil
}

func buildTaskGroup(check dun.CheckResult, stateHash string) taskGroup {
	group := taskGroup{
		Check: check,
	}
	if check.Prompt != nil {
		group.PromptAvailable = true
		group.Total = 1
		group.Tasks = []taskItem{
			{
				ID:      taskIDForCheck(check.ID, stateHash),
				Summary: truncateText(taskSummaryForCheck(check), maxTaskSummaryBytes),
				Why:     truncateText(taskReasonForCheck(check), maxTaskReasonBytes),
			},
		}
		return group
	}
	if len(check.Issues) > 0 {
		group.Total = len(check.Issues)
		limit := len(check.Issues)
		if limit > maxTasksPerCategory {
			limit = maxTasksPerCategory
		}
		for i := 0; i < limit; i++ {
			issue := check.Issues[i]
			group.Tasks = append(group.Tasks, taskItem{
				ID:      taskIDForIssue(check.ID, i+1, stateHash),
				Summary: truncateText(issueSummary(issue), maxTaskSummaryBytes),
				Why:     truncateText(taskReasonForCheck(check), maxTaskReasonBytes),
			})
		}
		return group
	}
	group.Total = 1
	group.Tasks = []taskItem{
		{
			ID:      taskIDForCheck(check.ID, stateHash),
			Summary: truncateText(taskSummaryForCheck(check), maxTaskSummaryBytes),
			Why:     truncateText(taskReasonForCheck(check), maxTaskReasonBytes),
		},
	}
	return group
}

func taskIDForIssue(checkID string, index int, stateHash string) string {
	return fmt.Sprintf("%s%s%d@%s", checkID, taskIDSeparator, index, stateHash)
}

func taskIDForCheck(checkID string, stateHash string) string {
	return fmt.Sprintf("%s@%s", checkID, stateHash)
}

func issueSummary(issue dun.Issue) string {
	summary := strings.TrimSpace(issue.Summary)
	if summary == "" {
		summary = strings.TrimSpace(issue.ID)
	}
	if issue.Path != "" {
		summary = fmt.Sprintf("%s (%s)", summary, issue.Path)
	}
	if summary == "" {
		summary = "issue"
	}
	return summary
}

func taskSummaryForCheck(check dun.CheckResult) string {
	if strings.TrimSpace(check.Detail) != "" {
		return check.Detail
	}
	if strings.TrimSpace(check.Signal) != "" {
		return check.Signal
	}
	return check.ID
}

func taskReasonForCheck(check dun.CheckResult) string {
	base := ""
	switch check.Status {
	case "error", "fail":
		base = "blocking"
	case "warn":
		base = "needs attention"
	case "prompt":
		base = "agent prompt ready"
	case "action":
		base = "action required"
	case "info":
		base = "informational"
	case "skip":
		base = "low priority"
	case "pass":
		base = "complete"
	default:
		base = "status " + check.Status
	}
	if strings.TrimSpace(check.Signal) != "" {
		return base + ": " + check.Signal
	}
	if strings.TrimSpace(check.Detail) != "" {
		return base + ": " + check.Detail
	}
	return base
}

func truncateText(value string, maxBytes int) string {
	if maxBytes <= 0 || len(value) <= maxBytes {
		return value
	}
	if maxBytes <= 3 {
		return value[:maxBytes]
	}
	return value[:maxBytes-3] + "..."
}

func repoStateHash(root string) string {
	head, err := gitOutput(root, "rev-parse", "HEAD")
	if err != nil {
		return ""
	}
	status, err := gitOutput(root, "status", "--porcelain")
	if err != nil {
		return ""
	}
	payload := strings.TrimSpace(head) + "\n" + status
	sum := sha256.Sum256([]byte(payload))
	return hex.EncodeToString(sum[:])[:8]
}

func gitOutput(root string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}
