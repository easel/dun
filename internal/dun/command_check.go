package dun

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

const defaultCommandTimeout = 5 * time.Minute

// runCommandCheck executes a generic shell command check.
func runCommandCheck(root string, check Check) (CheckResult, error) {
	timeout := commandTimeout(check)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	shell, shellArg := shellCommand(check)

	var cmd *exec.Cmd
	if shellArg != "" {
		cmd = exec.CommandContext(ctx, shell, shellArg, check.Command)
	} else {
		cmd = exec.CommandContext(ctx, shell, check.Command)
	}
	cmd.Dir = root
	cmd.Env = buildCommandEnv(check)

	output, err := cmd.CombinedOutput()

	// Check for context timeout - must check before examining error
	// since killed processes return an ExitError
	if ctx.Err() == context.DeadlineExceeded {
		return CheckResult{
			ID:     check.ID,
			Status: "fail",
			Signal: "command timed out",
			Detail: "Command exceeded timeout of " + timeout.String(),
			Next:   check.Command,
		}, nil
	}
	if ctx.Err() == context.Canceled {
		return CheckResult{
			ID:     check.ID,
			Status: "fail",
			Signal: "command canceled",
			Detail: "Command was canceled",
			Next:   check.Command,
		}, nil
	}

	exitCode := exitCodeFromError(err)
	if exitCode == -1 && err != nil {
		return CheckResult{
			ID:     check.ID,
			Status: "fail",
			Signal: "command failed to run",
			Detail: err.Error(),
			Next:   check.Command,
		}, nil
	}

	status := statusFromExitCode(check, exitCode)
	signal := signalFromStatus(status, exitCode)
	issues, detail := parseOutput(check, output)

	return CheckResult{
		ID:     check.ID,
		Status: status,
		Signal: signal,
		Detail: detail,
		Issues: issues,
		Next:   nextFromStatus(status, check),
	}, nil
}

// commandTimeout returns the timeout duration for a command check.
func commandTimeout(check Check) time.Duration {
	if check.Timeout == "" {
		return defaultCommandTimeout
	}
	d, err := time.ParseDuration(check.Timeout)
	if err != nil {
		return defaultCommandTimeout
	}
	return d
}

// shellCommand returns the shell and argument to use for command execution.
func shellCommand(check Check) (string, string) {
	if check.Shell != "" {
		parts := strings.Fields(check.Shell)
		if len(parts) > 1 {
			return parts[0], strings.Join(parts[1:], " ")
		}
		return parts[0], ""
	}
	return "sh", "-c"
}

// buildCommandEnv builds the environment for command execution.
func buildCommandEnv(check Check) []string {
	env := os.Environ()
	for k, v := range check.Env {
		env = append(env, k+"="+v)
	}
	return env
}

// exitCodeFromError extracts the exit code from an exec error.
func exitCodeFromError(err error) int {
	if err == nil {
		return 0
	}
	if exitErr, ok := err.(*exec.ExitError); ok {
		return exitErr.ExitCode()
	}
	return -1
}

// statusFromExitCode determines the check status from the exit code.
func statusFromExitCode(check Check, exitCode int) string {
	if exitCode == check.SuccessExit {
		return "pass"
	}
	for _, warnExit := range check.WarnExits {
		if exitCode == warnExit {
			return "warn"
		}
	}
	return "fail"
}

// signalFromStatus generates a signal message based on status.
func signalFromStatus(status string, exitCode int) string {
	switch status {
	case "pass":
		return "command passed"
	case "warn":
		return "command returned warning"
	default:
		if exitCode == -1 {
			return "command failed to execute"
		}
		return "command failed"
	}
}

// nextFromStatus generates a next action message based on status.
func nextFromStatus(status string, check Check) string {
	if status == "pass" {
		return ""
	}
	return check.Command
}

// parseOutput parses command output based on the parser type.
func parseOutput(check Check, output []byte) ([]Issue, string) {
	switch check.Parser {
	case "json":
		return parseJSONOutput(check, output)
	case "json-lines":
		return parseJSONLinesOutput(check, output)
	case "lines":
		return parseLinesOutput(output)
	case "regex":
		return parseRegexOutput(check, output)
	default: // "text" or empty
		return parseTextOutput(output)
	}
}

// parseTextOutput returns the raw output as detail with no issues.
func parseTextOutput(output []byte) ([]Issue, string) {
	return nil, trimOutput(output)
}

// parseLinesOutput converts each non-empty line to an issue.
func parseLinesOutput(output []byte) ([]Issue, string) {
	text := strings.TrimSpace(string(output))
	if text == "" {
		return nil, ""
	}

	lines := strings.Split(text, "\n")
	var issues []Issue
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		issues = append(issues, Issue{
			Summary: line,
		})
	}
	return issues, trimOutput(output)
}

// parseJSONOutput parses JSON output and extracts issues using configured paths.
func parseJSONOutput(check Check, output []byte) ([]Issue, string) {
	text := strings.TrimSpace(string(output))
	if text == "" {
		return nil, ""
	}

	var data interface{}
	if err := json.Unmarshal(output, &data); err != nil {
		return nil, trimOutput(output)
	}

	issues := extractIssuesFromJSON(check, data)
	return issues, trimOutput(output)
}

// parseJSONLinesOutput parses newline-delimited JSON and extracts issues.
func parseJSONLinesOutput(check Check, output []byte) ([]Issue, string) {
	text := strings.TrimSpace(string(output))
	if text == "" {
		return nil, ""
	}

	lines := strings.Split(text, "\n")
	var allIssues []Issue
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var data interface{}
		if err := json.Unmarshal([]byte(line), &data); err != nil {
			continue
		}
		issues := extractIssuesFromJSON(check, data)
		allIssues = append(allIssues, issues...)
	}
	return allIssues, trimOutput(output)
}

// parseRegexOutput extracts issues using a regex pattern with named groups.
func parseRegexOutput(check Check, output []byte) ([]Issue, string) {
	if check.IssuePattern == "" {
		return nil, trimOutput(output)
	}

	re, err := regexp.Compile(check.IssuePattern)
	if err != nil {
		return nil, trimOutput(output)
	}

	text := string(output)
	matches := re.FindAllStringSubmatch(text, -1)
	if len(matches) == 0 {
		return nil, trimOutput(output)
	}

	names := re.SubexpNames()
	var issues []Issue
	for _, match := range matches {
		issue := Issue{}
		for i, name := range names {
			if i >= len(match) {
				continue
			}
			value := match[i]
			switch name {
			case "file":
				issue.Path = value
			case "message":
				issue.Summary = value
			case "id":
				issue.ID = value
			}
		}
		if issue.Summary != "" || issue.Path != "" {
			issues = append(issues, issue)
		}
	}
	return issues, trimOutput(output)
}

// extractIssuesFromJSON extracts issues from parsed JSON data.
func extractIssuesFromJSON(check Check, data interface{}) []Issue {
	items := resolveJSONPath(data, check.IssuePath)
	if items == nil {
		if issue := extractSingleIssue(check, data); issue != nil {
			return []Issue{*issue}
		}
		return nil
	}

	arr, ok := items.([]interface{})
	if !ok {
		if issue := extractSingleIssue(check, items); issue != nil {
			return []Issue{*issue}
		}
		return nil
	}

	var issues []Issue
	for _, item := range arr {
		if issue := extractSingleIssue(check, item); issue != nil {
			issues = append(issues, *issue)
		}
	}
	return issues
}

// extractSingleIssue extracts a single issue from a JSON object.
func extractSingleIssue(check Check, data interface{}) *Issue {
	obj, ok := data.(map[string]interface{})
	if !ok {
		return nil
	}

	issue := &Issue{}
	fields := check.IssueFields

	if fields.File != "" {
		if val := resolveJSONPath(obj, fields.File); val != nil {
			if s, ok := val.(string); ok {
				issue.Path = s
			}
		}
	}

	if fields.Message != "" {
		if val := resolveJSONPath(obj, fields.Message); val != nil {
			if s, ok := val.(string); ok {
				issue.Summary = s
			}
		}
	}

	if issue.Summary == "" {
		for _, key := range []string{"message", "msg", "summary", "description", "text"} {
			if val, ok := obj[key]; ok {
				if s, ok := val.(string); ok {
					issue.Summary = s
					break
				}
			}
		}
	}
	if issue.Path == "" {
		for _, key := range []string{"file", "path", "filename", "location"} {
			if val, ok := obj[key]; ok {
				if s, ok := val.(string); ok {
					issue.Path = s
					break
				}
			}
		}
	}

	if issue.Summary == "" && issue.Path == "" {
		return nil
	}
	return issue
}

// resolveJSONPath resolves a simple dot-notation path in JSON data.
func resolveJSONPath(data interface{}, path string) interface{} {
	if path == "" {
		return data
	}

	path = strings.TrimPrefix(path, "$.")
	path = strings.TrimPrefix(path, "$")

	if path == "" {
		return data
	}

	parts := strings.Split(path, ".")
	current := data

	for _, part := range parts {
		if part == "" {
			continue
		}
		obj, ok := current.(map[string]interface{})
		if !ok {
			return nil
		}
		val, exists := obj[part]
		if !exists {
			return nil
		}
		current = val
	}
	return current
}
