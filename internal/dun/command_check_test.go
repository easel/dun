package dun

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"
)

func commandConfigFromCheck(check Check) CommandConfig {
	return CommandConfig{
		Command:      check.Command,
		Parser:       check.Parser,
		SuccessExit:  check.SuccessExit,
		WarnExits:    check.WarnExits,
		Timeout:      check.Timeout,
		Shell:        check.Shell,
		Env:          check.Env,
		IssuePath:    check.IssuePath,
		IssuePattern: check.IssuePattern,
		IssueFields:  check.IssueFields,
	}
}

func runCommandCheckFromSpec(root string, check Check) (CheckResult, error) {
	def := CheckDefinition{ID: check.ID}
	return runCommandCheck(root, def, commandConfigFromCheck(check))
}

type commandCapture struct {
	dir      string
	shell    string
	shellArg string
	command  string
	env      []string
	calls    int
}

func captureCommandRunner(capture *commandCapture, output []byte, exitCode int, err error) commandRunner {
	return func(_ context.Context, dir string, shell string, shellArg string, command string, env []string) ([]byte, int, error) {
		capture.calls++
		capture.dir = dir
		capture.shell = shell
		capture.shellArg = shellArg
		capture.command = command
		capture.env = append([]string(nil), env...)
		return output, exitCode, err
	}
}

func blockingCommandRunner(capture *commandCapture) commandRunner {
	return func(ctx context.Context, dir string, shell string, shellArg string, command string, env []string) ([]byte, int, error) {
		capture.calls++
		capture.dir = dir
		capture.shell = shell
		capture.shellArg = shellArg
		capture.command = command
		capture.env = append([]string(nil), env...)
		<-ctx.Done()
		return nil, -1, ctx.Err()
	}
}


func TestRunCommandCheck_Success(t *testing.T) {
	root := t.TempDir()
	capture := &commandCapture{}
	orig := commandRunnerFn
	commandRunnerFn = captureCommandRunner(capture, []byte("hello"), 0, nil)
	t.Cleanup(func() { commandRunnerFn = orig })

	check := Check{
		ID:      "test-echo",
		Type:    "command",
		Command: "echo hello",
	}

	result, err := runCommandCheckFromSpec(root, check)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != "pass" {
		t.Errorf("expected status 'pass', got %q", result.Status)
	}
	if result.Signal != "command passed" {
		t.Errorf("expected signal 'command passed', got %q", result.Signal)
	}
	if result.Next != "" {
		t.Errorf("expected empty Next for pass, got %q", result.Next)
	}
	if capture.command != check.Command {
		t.Errorf("expected command %q, got %q", check.Command, capture.command)
	}
}

func TestRunCommandCheck_Failure(t *testing.T) {
	root := t.TempDir()
	capture := &commandCapture{}
	orig := commandRunnerFn
	commandRunnerFn = captureCommandRunner(capture, nil, 1, errors.New("exit 1"))
	t.Cleanup(func() { commandRunnerFn = orig })

	check := Check{
		ID:      "test-exit",
		Type:    "command",
		Command: "exit 1",
	}

	result, err := runCommandCheckFromSpec(root, check)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != "fail" {
		t.Errorf("expected status 'fail', got %q", result.Status)
	}
	if result.Signal != "command failed" {
		t.Errorf("expected signal 'command failed', got %q", result.Signal)
	}
	if result.Next != "exit 1" {
		t.Errorf("expected Next to be command, got %q", result.Next)
	}
}

func TestRunCommandCheck_CustomSuccessExit(t *testing.T) {
	root := t.TempDir()
	orig := commandRunnerFn
	commandRunnerFn = captureCommandRunner(&commandCapture{}, nil, 2, errors.New("exit 2"))
	t.Cleanup(func() { commandRunnerFn = orig })

	check := Check{
		ID:          "test-custom-exit",
		Type:        "command",
		Command:     "exit 2",
		SuccessExit: 2,
	}

	result, err := runCommandCheckFromSpec(root, check)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != "pass" {
		t.Errorf("expected status 'pass', got %q", result.Status)
	}
}

func TestRunCommandCheck_WarnExit(t *testing.T) {
	root := t.TempDir()
	orig := commandRunnerFn
	commandRunnerFn = captureCommandRunner(&commandCapture{}, nil, 2, errors.New("exit 2"))
	t.Cleanup(func() { commandRunnerFn = orig })

	check := Check{
		ID:        "test-warn-exit",
		Type:      "command",
		Command:   "exit 2",
		WarnExits: []int{2, 3},
	}

	result, err := runCommandCheckFromSpec(root, check)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != "warn" {
		t.Errorf("expected status 'warn', got %q", result.Status)
	}
	if result.Signal != "command returned warning" {
		t.Errorf("expected signal 'command returned warning', got %q", result.Signal)
	}
}

func TestRunCommandCheck_Timeout(t *testing.T) {
	root := t.TempDir()
	capture := &commandCapture{}
	orig := commandRunnerFn
	commandRunnerFn = blockingCommandRunner(capture)
	t.Cleanup(func() { commandRunnerFn = orig })

	// Use a command that runs in the shell process itself (not a subprocess)
	// so that the context cancellation properly kills it
	check := Check{
		ID:      "test-timeout",
		Type:    "command",
		Command: "while true; do :; done",
		Timeout: "50ms",
	}

	start := time.Now()
	result, err := runCommandCheckFromSpec(root, check)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != "fail" {
		t.Errorf("expected status 'fail', got %q", result.Status)
	}
	// The command should timeout (context deadline exceeded is detected)
	if result.Signal != "command timed out" {
		t.Errorf("expected signal 'command timed out', got %q", result.Signal)
	}
	// Should complete quickly due to timeout
	if elapsed > 1*time.Second {
		t.Errorf("command should have timed out quickly, took %v", elapsed)
	}
}

func TestRunCommandCheck_WorkingDirectory(t *testing.T) {
	root := t.TempDir()
	capture := &commandCapture{}
	orig := commandRunnerFn
	commandRunnerFn = captureCommandRunner(capture, []byte("test content"), 0, nil)
	t.Cleanup(func() { commandRunnerFn = orig })

	check := Check{
		ID:      "test-pwd",
		Type:    "command",
		Command: "cat testfile.txt",
	}

	result, err := runCommandCheckFromSpec(root, check)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != "pass" {
		t.Errorf("expected status 'pass', got %q", result.Status)
	}
	if capture.dir != root {
		t.Errorf("expected working dir %q, got %q", root, capture.dir)
	}
}

func TestRunCommandCheck_CustomEnv(t *testing.T) {
	root := t.TempDir()
	capture := &commandCapture{}
	orig := commandRunnerFn
	commandRunnerFn = captureCommandRunner(capture, []byte("hello"), 0, nil)
	t.Cleanup(func() { commandRunnerFn = orig })

	check := Check{
		ID:      "test-env",
		Type:    "command",
		Command: "echo $MY_VAR",
		Env:     map[string]string{"MY_VAR": "hello"},
	}

	result, err := runCommandCheckFromSpec(root, check)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != "pass" {
		t.Errorf("expected status 'pass', got %q", result.Status)
	}
	found := false
	for _, entry := range capture.env {
		if entry == "MY_VAR=hello" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected env to include MY_VAR=hello, got %v", capture.env)
	}
}

func TestRunCommandCheck_CustomShell(t *testing.T) {
	root := t.TempDir()
	capture := &commandCapture{}
	orig := commandRunnerFn
	commandRunnerFn = captureCommandRunner(capture, []byte("hello"), 0, nil)
	t.Cleanup(func() { commandRunnerFn = orig })

	check := Check{
		ID:      "test-shell",
		Type:    "command",
		Command: "echo hello",
		Shell:   "bash -c",
	}

	result, err := runCommandCheckFromSpec(root, check)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != "pass" {
		t.Errorf("expected status 'pass', got %q", result.Status)
	}
	if capture.shell != "bash" || capture.shellArg != "-c" {
		t.Errorf("expected shell bash -c, got %q %q", capture.shell, capture.shellArg)
	}
}

func TestRunCommandCheck_LinesParser(t *testing.T) {
	root := t.TempDir()
	orig := commandRunnerFn
	commandRunnerFn = captureCommandRunner(&commandCapture{}, []byte("line1\nline2\n"), 0, nil)
	t.Cleanup(func() { commandRunnerFn = orig })

	check := Check{
		ID:      "test-lines",
		Type:    "command",
		Command: "printf 'line1\\nline2\\n'",
		Parser:  "lines",
	}

	result, err := runCommandCheckFromSpec(root, check)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != "pass" {
		t.Errorf("expected status 'pass', got %q", result.Status)
	}
	if len(result.Issues) != 2 {
		t.Errorf("expected 2 issues, got %d", len(result.Issues))
	}
}

func TestRunCommandCheck_JSONParser(t *testing.T) {
	root := t.TempDir()
	orig := commandRunnerFn
	commandRunnerFn = captureCommandRunner(&commandCapture{}, []byte(`{"issues":[{"file":"a.go","message":"error"}]}`), 0, nil)
	t.Cleanup(func() { commandRunnerFn = orig })

	check := Check{
		ID:      "test-json",
		Type:    "command",
		Command: `echo '{"issues":[{"file":"a.go","message":"error"}]}'`,
		Parser:  "json",
		IssueFields: IssueFieldMap{
			File:    "file",
			Message: "message",
		},
		IssuePath: "issues",
	}

	result, err := runCommandCheckFromSpec(root, check)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != "pass" {
		t.Errorf("expected status 'pass', got %q", result.Status)
	}
	if len(result.Issues) != 1 {
		t.Errorf("expected 1 issue, got %d", len(result.Issues))
	}
}

// Parser unit tests

func TestParseTextOutput(t *testing.T) {
	output := []byte("hello world\nline 2\n")
	issues, detail := parseTextOutput(output)

	if len(issues) != 0 {
		t.Errorf("expected no issues, got %d", len(issues))
	}
	if detail == "" {
		t.Error("expected non-empty detail")
	}
}

func TestParseTextOutput_Empty(t *testing.T) {
	output := []byte("")
	issues, detail := parseTextOutput(output)

	if len(issues) != 0 {
		t.Errorf("expected no issues, got %d", len(issues))
	}
	if detail != "hook command failed" {
		t.Errorf("expected 'hook command failed' for empty output, got %q", detail)
	}
}

func TestParseLinesOutput(t *testing.T) {
	output := []byte("error: file1.go:10: missing import\nerror: file2.go:20: unused var\n\n")
	issues, detail := parseLinesOutput(output)

	if len(issues) != 2 {
		t.Fatalf("expected 2 issues, got %d", len(issues))
	}
	if issues[0].Summary != "error: file1.go:10: missing import" {
		t.Errorf("unexpected issue 0 summary: %q", issues[0].Summary)
	}
	if issues[1].Summary != "error: file2.go:20: unused var" {
		t.Errorf("unexpected issue 1 summary: %q", issues[1].Summary)
	}
	if detail == "" {
		t.Error("expected non-empty detail")
	}
}

func TestParseLinesOutput_Empty(t *testing.T) {
	output := []byte("")
	issues, detail := parseLinesOutput(output)

	if len(issues) != 0 {
		t.Errorf("expected no issues, got %d", len(issues))
	}
	if detail != "" {
		t.Errorf("expected empty detail, got %q", detail)
	}
}

func TestParseLinesOutput_WhitespaceOnly(t *testing.T) {
	output := []byte("   \n\n  \n")
	issues, detail := parseLinesOutput(output)

	if len(issues) != 0 {
		t.Errorf("expected no issues, got %d", len(issues))
	}
	if detail != "" {
		t.Errorf("expected empty detail, got %q", detail)
	}
}

func TestParseJSONOutput(t *testing.T) {
	output := []byte(`{
		"issues": [
			{"file": "main.go", "message": "unused import"},
			{"file": "util.go", "message": "missing docs"}
		]
	}`)

	check := Check{
		IssuePath: "$.issues",
		IssueFields: IssueFieldMap{
			File:    "file",
			Message: "message",
		},
	}

	issues, detail := parseJSONOutput(commandConfigFromCheck(check), output)

	if len(issues) != 2 {
		t.Fatalf("expected 2 issues, got %d", len(issues))
	}
	if issues[0].Path != "main.go" {
		t.Errorf("expected path 'main.go', got %q", issues[0].Path)
	}
	if issues[0].Summary != "unused import" {
		t.Errorf("expected summary 'unused import', got %q", issues[0].Summary)
	}
	if detail == "" {
		t.Error("expected non-empty detail")
	}
}

func TestParseJSONOutput_CommonFieldNames(t *testing.T) {
	output := []byte(`{
		"errors": [
			{"path": "main.go", "msg": "error 1"},
			{"filename": "util.go", "description": "error 2"}
		]
	}`)

	check := Check{
		IssuePath:   "errors",
		IssueFields: IssueFieldMap{},
	}

	issues, _ := parseJSONOutput(commandConfigFromCheck(check), output)

	if len(issues) != 2 {
		t.Fatalf("expected 2 issues, got %d", len(issues))
	}
	if issues[0].Path != "main.go" {
		t.Errorf("expected path 'main.go', got %q", issues[0].Path)
	}
	if issues[0].Summary != "error 1" {
		t.Errorf("expected summary 'error 1', got %q", issues[0].Summary)
	}
}

func TestParseJSONOutput_InvalidJSON(t *testing.T) {
	output := []byte(`not valid json`)

	check := Check{
		IssuePath: "issues",
	}

	issues, detail := parseJSONOutput(commandConfigFromCheck(check), output)

	if len(issues) != 0 {
		t.Errorf("expected no issues for invalid JSON, got %d", len(issues))
	}
	if detail == "" {
		t.Error("expected fallback to text detail")
	}
}

func TestParseJSONOutput_Empty(t *testing.T) {
	output := []byte("")

	check := Check{
		IssuePath: "issues",
	}

	issues, detail := parseJSONOutput(commandConfigFromCheck(check), output)

	if len(issues) != 0 {
		t.Errorf("expected no issues, got %d", len(issues))
	}
	if detail != "" {
		t.Errorf("expected empty detail, got %q", detail)
	}
}

func TestParseJSONLinesOutput(t *testing.T) {
	output := []byte(`{"file": "a.go", "message": "error 1"}
{"file": "b.go", "message": "error 2"}
`)

	check := Check{
		IssueFields: IssueFieldMap{
			File:    "file",
			Message: "message",
		},
	}

	issues, detail := parseJSONLinesOutput(commandConfigFromCheck(check), output)

	if len(issues) != 2 {
		t.Fatalf("expected 2 issues, got %d", len(issues))
	}
	if issues[0].Path != "a.go" {
		t.Errorf("expected path 'a.go', got %q", issues[0].Path)
	}
	if issues[1].Summary != "error 2" {
		t.Errorf("expected summary 'error 2', got %q", issues[1].Summary)
	}
	if detail == "" {
		t.Error("expected non-empty detail")
	}
}

func TestParseJSONLinesOutput_MixedValid(t *testing.T) {
	output := []byte(`{"message": "valid 1"}
invalid json line
{"message": "valid 2"}
`)

	check := Check{
		IssueFields: IssueFieldMap{
			Message: "message",
		},
	}

	issues, _ := parseJSONLinesOutput(commandConfigFromCheck(check), output)

	if len(issues) != 2 {
		t.Fatalf("expected 2 issues (skipping invalid), got %d", len(issues))
	}
}

func TestParseJSONLinesOutput_Empty(t *testing.T) {
	output := []byte("")

	check := Check{}

	issues, detail := parseJSONLinesOutput(commandConfigFromCheck(check), output)

	if len(issues) != 0 {
		t.Errorf("expected no issues, got %d", len(issues))
	}
	if detail != "" {
		t.Errorf("expected empty detail, got %q", detail)
	}
}

func TestParseRegexOutput(t *testing.T) {
	output := []byte(`main.go:10: unused variable
util.go:20: missing return
`)

	check := Check{
		IssuePattern: `(?P<file>[^:]+):(?P<line>\d+): (?P<message>.+)`,
	}

	issues, detail := parseRegexOutput(commandConfigFromCheck(check), output)

	if len(issues) != 2 {
		t.Fatalf("expected 2 issues, got %d", len(issues))
	}
	if issues[0].Path != "main.go" {
		t.Errorf("expected path 'main.go', got %q", issues[0].Path)
	}
	if issues[0].Summary != "unused variable" {
		t.Errorf("expected summary 'unused variable', got %q", issues[0].Summary)
	}
	if detail == "" {
		t.Error("expected non-empty detail")
	}
}

func TestParseRegexOutput_NoPattern(t *testing.T) {
	output := []byte("some output")

	check := Check{
		IssuePattern: "",
	}

	issues, detail := parseRegexOutput(commandConfigFromCheck(check), output)

	if len(issues) != 0 {
		t.Errorf("expected no issues without pattern, got %d", len(issues))
	}
	if detail == "" {
		t.Error("expected text fallback")
	}
}

func TestParseRegexOutput_InvalidPattern(t *testing.T) {
	output := []byte("some output")

	check := Check{
		IssuePattern: "[invalid(regex",
	}

	issues, detail := parseRegexOutput(commandConfigFromCheck(check), output)

	if len(issues) != 0 {
		t.Errorf("expected no issues with invalid pattern, got %d", len(issues))
	}
	if detail == "" {
		t.Error("expected text fallback")
	}
}

func TestParseRegexOutput_NoMatches(t *testing.T) {
	output := []byte("no matches here")

	check := Check{
		IssuePattern: `(?P<file>\w+\.go):(?P<message>.+)`,
	}

	issues, detail := parseRegexOutput(commandConfigFromCheck(check), output)

	if len(issues) != 0 {
		t.Errorf("expected no issues when no matches, got %d", len(issues))
	}
	if detail == "" {
		t.Error("expected text fallback")
	}
}

func TestParseRegexOutput_PartialGroups(t *testing.T) {
	output := []byte("error in main.go")

	check := Check{
		IssuePattern: `error in (?P<file>\w+\.go)`,
	}

	issues, _ := parseRegexOutput(commandConfigFromCheck(check), output)

	if len(issues) != 1 {
		t.Fatalf("expected 1 issue, got %d", len(issues))
	}
	if issues[0].Path != "main.go" {
		t.Errorf("expected path 'main.go', got %q", issues[0].Path)
	}
}

func TestParseRegexOutput_IDGroup(t *testing.T) {
	output := []byte("[ERR001] something went wrong")

	check := Check{
		IssuePattern: `\[(?P<id>\w+)\] (?P<message>.+)`,
	}

	issues, _ := parseRegexOutput(commandConfigFromCheck(check), output)

	if len(issues) != 1 {
		t.Fatalf("expected 1 issue, got %d", len(issues))
	}
	if issues[0].ID != "ERR001" {
		t.Errorf("expected ID 'ERR001', got %q", issues[0].ID)
	}
	if issues[0].Summary != "something went wrong" {
		t.Errorf("expected summary 'something went wrong', got %q", issues[0].Summary)
	}
}

// Helper function tests

func TestCommandTimeout(t *testing.T) {
	tests := []struct {
		name     string
		timeout  string
		expected time.Duration
	}{
		{"empty", "", defaultCommandTimeout},
		{"valid", "30s", 30 * time.Second},
		{"minutes", "2m", 2 * time.Minute},
		{"invalid", "notaduration", defaultCommandTimeout},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			check := Check{Timeout: tt.timeout}
			got := commandTimeout(commandConfigFromCheck(check))
			if got != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, got)
			}
		})
	}
}

func TestShellCommand(t *testing.T) {
	tests := []struct {
		name        string
		shell       string
		expectedCmd string
		expectedArg string
	}{
		{"empty", "", "sh", "-c"},
		{"bash", "bash -c", "bash", "-c"},
		{"single", "zsh", "zsh", ""},
		{"with flags", "bash --norc -c", "bash", "--norc -c"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			check := Check{Shell: tt.shell}
			cmd, arg := shellCommand(commandConfigFromCheck(check))
			if cmd != tt.expectedCmd {
				t.Errorf("expected cmd %q, got %q", tt.expectedCmd, cmd)
			}
			if arg != tt.expectedArg {
				t.Errorf("expected arg %q, got %q", tt.expectedArg, arg)
			}
		})
	}
}

func TestExitCodeFromError(t *testing.T) {
	if got := exitCodeFromError(nil); got != 0 {
		t.Errorf("expected 0 for nil error, got %d", got)
	}

	if got := exitCodeFromError(os.ErrNotExist); got != -1 {
		t.Errorf("expected -1 for non-ExitError, got %d", got)
	}
}

func TestStatusFromExitCode(t *testing.T) {
	tests := []struct {
		name        string
		successExit int
		warnExits   []int
		exitCode    int
		expected    string
	}{
		{"pass default", 0, nil, 0, "pass"},
		{"fail", 0, nil, 1, "fail"},
		{"custom success", 2, nil, 2, "pass"},
		{"warn exit", 0, []int{2, 3}, 2, "warn"},
		{"warn exit second", 0, []int{2, 3}, 3, "warn"},
		{"fail not warn", 0, []int{2, 3}, 4, "fail"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			check := Check{
				SuccessExit: tt.successExit,
				WarnExits:   tt.warnExits,
			}
			got := statusFromExitCode(commandConfigFromCheck(check), tt.exitCode)
			if got != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, got)
			}
		})
	}
}

func TestSignalFromStatus(t *testing.T) {
	tests := []struct {
		status   string
		exitCode int
		expected string
	}{
		{"pass", 0, "command passed"},
		{"warn", 2, "command returned warning"},
		{"fail", 1, "command failed"},
		{"fail", -1, "command failed to execute"},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			got := signalFromStatus(tt.status, tt.exitCode)
			if got != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, got)
			}
		})
	}
}

func TestNextFromStatus(t *testing.T) {
	check := Check{Command: "test cmd"}

	tests := []struct {
		status   string
		expected string
	}{
		{"pass", ""},
		{"fail", "test cmd"},
		{"warn", "test cmd"},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			got := nextFromStatus(tt.status, commandConfigFromCheck(check))
			if got != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, got)
			}
		})
	}
}

func TestBuildCommandEnv(t *testing.T) {
	check := Check{
		Env: map[string]string{
			"FOO": "bar",
			"BAZ": "qux",
		},
	}

	env := buildCommandEnv(commandConfigFromCheck(check))

	found := 0
	for _, e := range env {
		if e == "FOO=bar" || e == "BAZ=qux" {
			found++
		}
	}
	if found != 2 {
		t.Errorf("expected to find 2 custom env vars, found %d", found)
	}
}

func TestBuildCommandEnv_NoEnv(t *testing.T) {
	check := Check{}
	env := buildCommandEnv(commandConfigFromCheck(check))

	if len(env) == 0 {
		t.Error("expected system env vars to be present")
	}
}

func TestResolveJSONPath(t *testing.T) {
	data := map[string]interface{}{
		"issues": []interface{}{
			map[string]interface{}{"msg": "error 1"},
		},
		"nested": map[string]interface{}{
			"items": []interface{}{"a", "b"},
		},
	}

	tests := []struct {
		name     string
		path     string
		hasValue bool
	}{
		{"empty path", "", true},
		{"simple", "issues", true},
		{"nested", "nested.items", true},
		{"jsonpath prefix", "$.issues", true},
		{"dollar only", "$", true},
		{"not found", "missing", false},
		{"deep not found", "nested.missing", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := resolveJSONPath(data, tt.path)
			if tt.hasValue && result == nil {
				t.Error("expected value, got nil")
			}
			if !tt.hasValue && result != nil {
				t.Errorf("expected nil, got %v", result)
			}
		})
	}
}

func TestResolveJSONPath_NonMap(t *testing.T) {
	data := "not a map"
	result := resolveJSONPath(data, "something")
	if result != nil {
		t.Errorf("expected nil for non-map data, got %v", result)
	}
}

func TestParseOutput_DefaultToText(t *testing.T) {
	output := []byte("some text output")
	check := Check{Parser: ""}

	issues, detail := parseOutput(commandConfigFromCheck(check), output)

	if len(issues) != 0 {
		t.Errorf("expected no issues for text parser, got %d", len(issues))
	}
	if detail == "" {
		t.Error("expected non-empty detail")
	}
}

func TestParseOutput_TextParser(t *testing.T) {
	output := []byte("some text output")
	check := Check{Parser: "text"}

	issues, detail := parseOutput(commandConfigFromCheck(check), output)

	if len(issues) != 0 {
		t.Errorf("expected no issues for text parser, got %d", len(issues))
	}
	if detail == "" {
		t.Error("expected non-empty detail")
	}
}

func TestExtractSingleIssue_NotMap(t *testing.T) {
	check := Check{}
	result := extractSingleIssue(commandConfigFromCheck(check), "not a map")
	if result != nil {
		t.Errorf("expected nil for non-map data, got %v", result)
	}
}

func TestExtractSingleIssue_EmptyResult(t *testing.T) {
	check := Check{
		IssueFields: IssueFieldMap{
			File:    "nonexistent",
			Message: "nonexistent",
		},
	}
	data := map[string]interface{}{
		"other": "value",
	}
	result := extractSingleIssue(commandConfigFromCheck(check), data)
	if result != nil {
		t.Errorf("expected nil when no fields match, got %v", result)
	}
}

func TestExtractIssuesFromJSON_SingleItem(t *testing.T) {
	check := Check{
		IssuePath: "error",
		IssueFields: IssueFieldMap{
			Message: "message",
		},
	}
	data := map[string]interface{}{
		"error": map[string]interface{}{
			"message": "single error",
		},
	}

	issues := extractIssuesFromJSON(commandConfigFromCheck(check), data)
	if len(issues) != 1 {
		t.Fatalf("expected 1 issue, got %d", len(issues))
	}
	if issues[0].Summary != "single error" {
		t.Errorf("expected 'single error', got %q", issues[0].Summary)
	}
}

func TestExtractIssuesFromJSON_NoPath(t *testing.T) {
	check := Check{
		IssuePath: "",
		IssueFields: IssueFieldMap{
			Message: "message",
		},
	}
	data := map[string]interface{}{
		"message": "top level error",
	}

	issues := extractIssuesFromJSON(commandConfigFromCheck(check), data)
	if len(issues) != 1 {
		t.Fatalf("expected 1 issue from top-level, got %d", len(issues))
	}
	if issues[0].Summary != "top level error" {
		t.Errorf("expected 'top level error', got %q", issues[0].Summary)
	}
}

func TestExtractIssuesFromJSON_NilItems(t *testing.T) {
	check := Check{
		IssuePath: "nonexistent",
	}
	data := map[string]interface{}{
		"other": "value",
	}

	issues := extractIssuesFromJSON(commandConfigFromCheck(check), data)
	if len(issues) != 0 {
		t.Errorf("expected 0 issues for missing path, got %d", len(issues))
	}
}

func TestExtractIssuesFromJSON_NotArray(t *testing.T) {
	check := Check{
		IssuePath: "error",
		IssueFields: IssueFieldMap{
			Message: "msg",
		},
	}
	data := map[string]interface{}{
		"error": map[string]interface{}{
			"msg": "single",
		},
	}

	issues := extractIssuesFromJSON(commandConfigFromCheck(check), data)
	if len(issues) != 1 {
		t.Fatalf("expected 1 issue for non-array, got %d", len(issues))
	}
}

func TestExtractSingleIssue_PathOnly(t *testing.T) {
	check := Check{
		IssueFields: IssueFieldMap{
			File: "file",
		},
	}
	data := map[string]interface{}{
		"file": "main.go",
	}

	issue := extractSingleIssue(commandConfigFromCheck(check), data)
	if issue == nil {
		t.Fatal("expected issue, got nil")
	}
	if issue.Path != "main.go" {
		t.Errorf("expected path 'main.go', got %q", issue.Path)
	}
}

func TestExtractSingleIssue_NestedFields(t *testing.T) {
	check := Check{
		IssueFields: IssueFieldMap{
			File:    "location.file",
			Message: "details.msg",
		},
	}
	data := map[string]interface{}{
		"location": map[string]interface{}{
			"file": "nested.go",
		},
		"details": map[string]interface{}{
			"msg": "nested message",
		},
	}

	issue := extractSingleIssue(commandConfigFromCheck(check), data)
	if issue == nil {
		t.Fatal("expected issue, got nil")
	}
	if issue.Path != "nested.go" {
		t.Errorf("expected path 'nested.go', got %q", issue.Path)
	}
	if issue.Summary != "nested message" {
		t.Errorf("expected summary 'nested message', got %q", issue.Summary)
	}
}

func TestExtractSingleIssue_NonStringValues(t *testing.T) {
	check := Check{
		IssueFields: IssueFieldMap{
			File:    "file",
			Message: "message",
		},
	}
	data := map[string]interface{}{
		"file":    123,
		"message": true,
	}

	issue := extractSingleIssue(commandConfigFromCheck(check), data)
	if issue != nil {
		t.Errorf("expected nil for non-string values, got %v", issue)
	}
}

func TestRunCommandCheck_InvalidCommand(t *testing.T) {
	root := t.TempDir()
	capture := &commandCapture{}
	orig := commandRunnerFn
	commandRunnerFn = captureCommandRunner(capture, nil, -1, errors.New("no such file"))
	t.Cleanup(func() { commandRunnerFn = orig })

	check := Check{
		ID:      "test-invalid",
		Type:    "command",
		Command: "echo test",
		Shell:   "/nonexistent/shell",
	}

	result, err := runCommandCheckFromSpec(root, check)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != "fail" {
		t.Errorf("expected status 'fail', got %q", result.Status)
	}
	if result.Signal != "command failed to run" {
		t.Errorf("expected signal 'command failed to run', got %q", result.Signal)
	}
	if capture.shell != "/nonexistent/shell" {
		t.Errorf("expected shell %q, got %q", "/nonexistent/shell", capture.shell)
	}
}

func TestRunCommandCheck_RegexParser(t *testing.T) {
	root := t.TempDir()
	orig := commandRunnerFn
	commandRunnerFn = captureCommandRunner(&commandCapture{}, []byte("main.go:10: error\nutil.go:20: warning"), 0, nil)
	t.Cleanup(func() { commandRunnerFn = orig })

	check := Check{
		ID:           "test-regex",
		Type:         "command",
		Command:      `printf 'main.go:10: error\nutil.go:20: warning'`,
		Parser:       "regex",
		IssuePattern: `(?P<file>[^:]+):(?P<line>\d+): (?P<message>.+)`,
	}

	result, err := runCommandCheckFromSpec(root, check)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != "pass" {
		t.Errorf("expected status 'pass', got %q", result.Status)
	}
	if len(result.Issues) != 2 {
		t.Errorf("expected 2 issues, got %d", len(result.Issues))
	}
}

func TestRunCommandCheck_JSONLinesParser(t *testing.T) {
	root := t.TempDir()
	orig := commandRunnerFn
	commandRunnerFn = captureCommandRunner(&commandCapture{}, []byte("{\"file\":\"a.go\",\"message\":\"err1\"}\n{\"file\":\"b.go\",\"message\":\"err2\"}"), 0, nil)
	t.Cleanup(func() { commandRunnerFn = orig })

	check := Check{
		ID:      "test-jsonlines",
		Type:    "command",
		Command: `printf '{"file":"a.go","message":"err1"}\n{"file":"b.go","message":"err2"}'`,
		Parser:  "json-lines",
		IssueFields: IssueFieldMap{
			File:    "file",
			Message: "message",
		},
	}

	result, err := runCommandCheckFromSpec(root, check)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != "pass" {
		t.Errorf("expected status 'pass', got %q", result.Status)
	}
	if len(result.Issues) != 2 {
		t.Errorf("expected 2 issues, got %d", len(result.Issues))
	}
}

func TestCommandTimeout_Hours(t *testing.T) {
	check := Check{Timeout: "1h30m"}
	got := commandTimeout(commandConfigFromCheck(check))
	expected := 90 * time.Minute
	if got != expected {
		t.Errorf("expected %v, got %v", expected, got)
	}
}

func TestParseLinesOutput_SingleLine(t *testing.T) {
	output := []byte("single line only")
	issues, detail := parseLinesOutput(output)

	if len(issues) != 1 {
		t.Errorf("expected 1 issue, got %d", len(issues))
	}
	if issues[0].Summary != "single line only" {
		t.Errorf("unexpected summary: %q", issues[0].Summary)
	}
	if detail == "" {
		t.Error("expected non-empty detail")
	}
}

func TestExtractIssuesFromJSON_EmptyArray(t *testing.T) {
	check := Check{
		IssuePath: "items",
	}
	data := map[string]interface{}{
		"items": []interface{}{},
	}

	issues := extractIssuesFromJSON(commandConfigFromCheck(check), data)
	if len(issues) != 0 {
		t.Errorf("expected 0 issues for empty array, got %d", len(issues))
	}
}

func TestExtractIssuesFromJSON_NilData(t *testing.T) {
	check := Check{
		IssuePath: "items",
		IssueFields: IssueFieldMap{
			Message: "msg",
		},
	}

	issues := extractIssuesFromJSON(commandConfigFromCheck(check), nil)
	if len(issues) != 0 {
		t.Errorf("expected no issues for nil data, got %v", issues)
	}
}

func TestResolveJSONPath_EmptyParts(t *testing.T) {
	data := map[string]interface{}{
		"a": map[string]interface{}{
			"b": "value",
		},
	}

	// Path with extra dots
	result := resolveJSONPath(data, "a..b")
	if result == nil {
		t.Error("expected value for path with empty parts")
	}
}
