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

type commandRunner func(ctx context.Context, dir string, shell string, shellArg string, command string, env []string) ([]byte, int, error)

var commandRunnerFn commandRunner = runCommandWithExec

// runCommandCheck executes a generic shell command check.
func runCommandCheck(root string, def CheckDefinition, config CommandConfig) (CheckResult, error) {
	timeout := commandTimeout(config)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	shell, shellArg := shellCommand(config)
	env := buildCommandEnv(config)

	output, exitCode, err := commandRunnerFn(ctx, root, shell, shellArg, config.Command, env)

	// Check for context timeout - must check before examining error
	// since killed processes return an ExitError
	if ctx.Err() == context.DeadlineExceeded {
		return CheckResult{
			ID:     def.ID,
			Status: "fail",
			Signal: "command timed out",
			Detail: "Command exceeded timeout of " + timeout.String(),
			Next:   config.Command,
		}, nil
	}
	if ctx.Err() == context.Canceled {
		return CheckResult{
			ID:     def.ID,
			Status: "fail",
			Signal: "command canceled",
			Detail: "Command was canceled",
			Next:   config.Command,
		}, nil
	}

	if exitCode == -1 && err != nil {
		return CheckResult{
			ID:     def.ID,
			Status: "fail",
			Signal: "command failed to run",
			Detail: err.Error(),
			Next:   config.Command,
		}, nil
	}

	status := statusFromExitCode(config, exitCode)
	signal := signalFromStatus(status, exitCode)
	issues, detail := parseOutput(config, output)

	return CheckResult{
		ID:     def.ID,
		Status: status,
		Signal: signal,
		Detail: detail,
		Issues: issues,
		Next:   nextFromStatus(status, config),
	}, nil
}

func runCommandWithExec(ctx context.Context, dir string, shell string, shellArg string, command string, env []string) ([]byte, int, error) {
	var cmd *exec.Cmd
	if shellArg != "" {
		cmd = exec.CommandContext(ctx, shell, shellArg, command)
	} else {
		cmd = exec.CommandContext(ctx, shell, command)
	}
	cmd.Dir = dir
	cmd.Env = env

	output, err := cmd.CombinedOutput()
	return output, exitCodeFromError(err), err
}

// commandTimeout returns the timeout duration for a command check.
func commandTimeout(config CommandConfig) time.Duration {
	if config.Timeout == "" {
		return defaultCommandTimeout
	}
	d, err := time.ParseDuration(config.Timeout)
	if err != nil {
		return defaultCommandTimeout
	}
	return d
}

// shellCommand returns the shell and argument to use for command execution.
func shellCommand(config CommandConfig) (string, string) {
	if config.Shell != "" {
		parts := strings.Fields(config.Shell)
		if len(parts) > 1 {
			return parts[0], strings.Join(parts[1:], " ")
		}
		return parts[0], ""
	}
	return "sh", "-c"
}

// buildCommandEnv builds the environment for command execution.
func buildCommandEnv(config CommandConfig) []string {
	env := os.Environ()
	for k, v := range config.Env {
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
func statusFromExitCode(config CommandConfig, exitCode int) string {
	if exitCode == config.SuccessExit {
		return "pass"
	}
	for _, warnExit := range config.WarnExits {
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
func nextFromStatus(status string, config CommandConfig) string {
	if status == "pass" {
		return ""
	}
	return config.Command
}

// parseOutput parses command output based on the parser type.
func parseOutput(config CommandConfig, output []byte) ([]Issue, string) {
	switch config.Parser {
	case "json":
		return parseJSONOutput(config, output)
	case "json-lines":
		return parseJSONLinesOutput(config, output)
	case "lines":
		return parseLinesOutput(output)
	case "regex":
		return parseRegexOutput(config, output)
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
func parseJSONOutput(config CommandConfig, output []byte) ([]Issue, string) {
	text := strings.TrimSpace(string(output))
	if text == "" {
		return nil, ""
	}

	var data interface{}
	if err := json.Unmarshal(output, &data); err != nil {
		return nil, trimOutput(output)
	}

	issues := extractIssuesFromJSON(config, data)
	return issues, trimOutput(output)
}

// parseJSONLinesOutput parses newline-delimited JSON and extracts issues.
func parseJSONLinesOutput(config CommandConfig, output []byte) ([]Issue, string) {
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
		issues := extractIssuesFromJSON(config, data)
		allIssues = append(allIssues, issues...)
	}
	return allIssues, trimOutput(output)
}

// parseRegexOutput extracts issues using a regex pattern with named groups.
func parseRegexOutput(config CommandConfig, output []byte) ([]Issue, string) {
	if config.IssuePattern == "" {
		return nil, trimOutput(output)
	}

	re, err := regexp.Compile(config.IssuePattern)
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
func extractIssuesFromJSON(config CommandConfig, data interface{}) []Issue {
	items := resolveJSONPath(data, config.IssuePath)
	if items == nil {
		if issue := extractSingleIssue(config, data); issue != nil {
			return []Issue{*issue}
		}
		return nil
	}

	arr, ok := items.([]interface{})
	if !ok {
		if issue := extractSingleIssue(config, items); issue != nil {
			return []Issue{*issue}
		}
		return nil
	}

	var issues []Issue
	for _, item := range arr {
		if issue := extractSingleIssue(config, item); issue != nil {
			issues = append(issues, *issue)
		}
	}
	return issues
}

// extractSingleIssue extracts a single issue from a JSON object.
func extractSingleIssue(config CommandConfig, data interface{}) *Issue {
	obj, ok := data.(map[string]interface{})
	if !ok {
		return nil
	}

	issue := &Issue{}
	fields := config.IssueFields

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
