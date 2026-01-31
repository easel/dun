package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/easel/dun/internal/dun"
)

var exit = os.Exit
var checkRepo = dun.CheckRepo
var planRepo = dun.PlanRepo
var respondFn = dun.Respond
var installRepo = dun.InstallRepo
var callHarnessFn = callHarnessImpl

func main() {
	code := run(os.Args[1:], os.Stdout, os.Stderr)
	exit(code)
}

func run(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) < 1 {
		return runCheck(args, stdout, stderr)
	}
	switch args[0] {
	case "help", "--help", "-h":
		return runHelp(stdout)
	case "check":
		return runCheck(args[1:], stdout, stderr)
	case "list":
		return runList(args[1:], stdout, stderr)
	case "explain":
		return runExplain(args[1:], stdout, stderr)
	case "respond":
		return runRespond(args[1:], stdout, stderr)
	case "install":
		return runInstall(args[1:], stdout, stderr)
	case "iterate":
		return runIterate(args[1:], stdout, stderr)
	case "loop":
		return runLoop(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown command: %s\n", args[0])
		return dun.ExitUsageError
	}
}

func runHelp(stdout io.Writer) int {
	help := `dun - Development quality checks and autonomous iteration

USAGE:
  dun [command] [options]

COMMANDS:
  check      Run all checks and report status (default)
  list       List available checks
  explain    Show details for a specific check
  respond    Process agent response for a check
  install    Install dun config and agent documentation
  iterate    Output work list as a prompt for an agent
  loop       Run autonomous loop with an agent harness

LOOP MODE:
  dun loop [options]

  The loop command runs checks, generates prompts, calls an agent harness,
  and repeats until all checks pass or max iterations is reached.

  Options:
    --harness     Agent to use: claude, gemini, codex (default: claude)
    --automation  Mode: manual, plan, auto, yolo (default: auto)
    --max-iterations  Safety limit (default: 100)
    --dry-run     Show prompt without calling agent

  Examples:
    dun loop                              # Run with claude
    dun loop --harness gemini             # Run with gemini
    dun loop --automation yolo            # Allow autonomous edits
    dun loop --dry-run                    # Preview prompt

ITERATE MODE:
  dun iterate [options]

  Outputs a prompt listing available work for an external agent.
  Use this with a bash loop:

    while :; do dun iterate | claude -p "$(cat -)"; done

AGENT DOCUMENTATION:
  Run 'dun install' to add AGENTS.md with instructions for AI agents.
  Agents can then run 'dun iterate' or 'dun loop' to work autonomously.

EXIT CODES:
  0  Success / all checks pass
  1  Check failed
  2  Configuration error
  3  Runtime error
  4  Usage error
`
	fmt.Fprint(stdout, help)
	return dun.ExitSuccess
}

func runCheck(args []string, stdout io.Writer, stderr io.Writer) int {
	root := resolveRoot(".")
	explicitConfig := findConfigFlag(args)
	opts := dun.DefaultOptions()
	cfg, loaded, err := dun.LoadConfig(root, explicitConfig)
	if err != nil {
		fmt.Fprintf(stderr, "dun check failed: config error: %v\n", err)
		return dun.ExitConfigError
	}
	if loaded {
		opts = dun.ApplyConfig(opts, cfg)
	}

	fs := flag.NewFlagSet("check", flag.ContinueOnError)
	fs.SetOutput(stderr)
	configPath := fs.String("config", explicitConfig, "path to config file (default .dun/config.yaml if present)")
	format := fs.String("format", "prompt", "output format (prompt|llm|json)")
	agentCmd := fs.String("agent-cmd", opts.AgentCmd, "agent command override")
	agentTimeout := fs.Int("agent-timeout", int(opts.AgentTimeout/time.Second), "agent timeout in seconds")
	agentMode := fs.String("agent-mode", opts.AgentMode, "agent mode (prompt|auto)")
	automation := fs.String("automation", opts.AutomationMode, "automation mode (manual|plan|auto|yolo)")
	if err := fs.Parse(args); err != nil {
		return dun.ExitUsageError
	}
	explicitConfig = *configPath

	opts = dun.Options{
		AgentCmd:       *agentCmd,
		AgentTimeout:   time.Duration(*agentTimeout) * time.Second,
		AgentMode:      *agentMode,
		AutomationMode: *automation,
	}
	result, err := checkRepo(root, opts)
	if err != nil {
		fmt.Fprintf(stderr, "dun check failed: %v\n", err)
		return dun.ExitCheckFailed
	}

	switch *format {
	case "llm":
		printLLM(stdout, result)
	case "json", "prompt":
		if err := json.NewEncoder(stdout).Encode(result); err != nil {
			fmt.Fprintf(stderr, "encode json: %v\n", err)
			return dun.ExitCheckFailed
		}
	default:
		fmt.Fprintf(stderr, "unknown format: %s\n", *format)
		return dun.ExitUsageError
	}
	return dun.ExitSuccess
}

func runList(args []string, stdout io.Writer, stderr io.Writer) int {
	root := resolveRoot(".")
	fs := flag.NewFlagSet("list", flag.ContinueOnError)
	fs.SetOutput(stderr)
	format := fs.String("format", "text", "output format (text|json)")
	configPath := fs.String("config", "", "path to config file (default .dun/config.yaml if present)")
	if err := fs.Parse(args); err != nil {
		return dun.ExitUsageError
	}

	if _, _, err := dun.LoadConfig(root, *configPath); err != nil {
		fmt.Fprintf(stderr, "dun list failed: config error: %v\n", err)
		return dun.ExitConfigError
	}

	plan, err := planRepo(root)
	if err != nil {
		fmt.Fprintf(stderr, "dun list failed: %v\n", err)
		return dun.ExitCheckFailed
	}

	switch *format {
	case "json":
		if err := json.NewEncoder(stdout).Encode(plan); err != nil {
			fmt.Fprintf(stderr, "encode json: %v\n", err)
			return dun.ExitCheckFailed
		}
	default:
		for _, check := range plan.Checks {
			fmt.Fprintf(stdout, "%s\t%s\n", check.ID, check.Description)
		}
	}
	return dun.ExitSuccess
}

func runExplain(args []string, stdout io.Writer, stderr io.Writer) int {
	root := resolveRoot(".")
	fs := flag.NewFlagSet("explain", flag.ContinueOnError)
	fs.SetOutput(stderr)
	format := fs.String("format", "text", "output format (text|json)")
	configPath := fs.String("config", "", "path to config file (default .dun/config.yaml if present)")
	if err := fs.Parse(args); err != nil {
		return dun.ExitUsageError
	}

	if fs.NArg() < 1 {
		fmt.Fprintln(stderr, "usage: dun explain <check-id>")
		return dun.ExitUsageError
	}
	target := fs.Arg(0)

	if _, _, err := dun.LoadConfig(root, *configPath); err != nil {
		fmt.Fprintf(stderr, "dun explain failed: config error: %v\n", err)
		return dun.ExitConfigError
	}

	plan, err := planRepo(root)
	if err != nil {
		fmt.Fprintf(stderr, "dun explain failed: %v\n", err)
		return dun.ExitCheckFailed
	}

	for _, check := range plan.Checks {
		if check.ID != target {
			continue
		}
		switch *format {
		case "json":
			if err := json.NewEncoder(stdout).Encode(check); err != nil {
				fmt.Fprintf(stderr, "encode json: %v\n", err)
				return dun.ExitCheckFailed
			}
		default:
			fmt.Fprintf(stdout, "id: %s\n", check.ID)
			fmt.Fprintf(stdout, "description: %s\n", check.Description)
			fmt.Fprintf(stdout, "plugin: %s\n", check.PluginID)
			fmt.Fprintf(stdout, "type: %s\n", check.Type)
			if check.Phase != "" {
				fmt.Fprintf(stdout, "phase: %s\n", check.Phase)
			}
			if len(check.Conditions) > 0 {
				fmt.Fprintf(stdout, "conditions: %s\n", formatRules(check.Conditions))
			}
			if len(check.Inputs) > 0 {
				fmt.Fprintf(stdout, "inputs: %s\n", strings.Join(check.Inputs, ", "))
			}
			if len(check.GateFiles) > 0 {
				fmt.Fprintf(stdout, "gate_files: %s\n", strings.Join(check.GateFiles, ", "))
			}
			if check.StateRules != "" {
				fmt.Fprintf(stdout, "state_rules: %s\n", check.StateRules)
			}
			if check.Prompt != "" {
				fmt.Fprintf(stdout, "prompt: %s\n", check.Prompt)
			}
		}
		return dun.ExitSuccess
	}

	fmt.Fprintf(stderr, "unknown check: %s\n", target)
	return dun.ExitCheckFailed
}

func runRespond(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("respond", flag.ContinueOnError)
	fs.SetOutput(stderr)
	id := fs.String("id", "", "check id from prompt")
	responsePath := fs.String("response", "-", "response JSON path or - for stdin")
	format := fs.String("format", "json", "output format (json|llm)")
	if err := fs.Parse(args); err != nil {
		return dun.ExitUsageError
	}

	if *id == "" {
		fmt.Fprintln(stderr, "usage: dun respond --id <check-id> --response <path|->")
		return dun.ExitUsageError
	}

	var reader io.Reader = os.Stdin
	if *responsePath != "-" {
		file, err := os.Open(*responsePath)
		if err != nil {
			fmt.Fprintf(stderr, "open response: %v\n", err)
			return dun.ExitRuntimeError
		}
		defer file.Close()
		reader = file
	}

	check, err := respondFn(*id, reader)
	if err != nil {
		fmt.Fprintf(stderr, "dun respond failed: %v\n", err)
		return dun.ExitCheckFailed
	}

	result := dun.Result{Checks: []dun.CheckResult{check}}
	switch *format {
	case "llm":
		printLLM(stdout, result)
	case "json":
		if err := json.NewEncoder(stdout).Encode(check); err != nil {
			fmt.Fprintf(stderr, "encode json: %v\n", err)
			return dun.ExitCheckFailed
		}
	default:
		fmt.Fprintf(stderr, "unknown format: %s\n", *format)
		return dun.ExitUsageError
	}
	return dun.ExitSuccess
}

func runInstall(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("install", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dryRun := fs.Bool("dry-run", false, "show planned changes without writing")
	if err := fs.Parse(args); err != nil {
		return dun.ExitUsageError
	}

	result, err := installRepo(".", dun.InstallOptions{DryRun: *dryRun})
	if err != nil {
		fmt.Fprintf(stderr, "dun install failed: %v\n", err)
		return dun.ExitRuntimeError
	}

	for _, step := range result.Steps {
		if *dryRun {
			fmt.Fprintf(stdout, "plan: %s %s\n", step.Action, step.Path)
		} else {
			fmt.Fprintf(stdout, "installed: %s %s\n", step.Action, step.Path)
		}
	}

	fmt.Fprintln(stdout, "note: add hooks manually if desired (lefthook/pre-commit)")
	return dun.ExitSuccess
}

func runIterate(args []string, stdout io.Writer, stderr io.Writer) int {
	root := resolveRoot(".")
	explicitConfig := findConfigFlag(args)
	opts := dun.DefaultOptions()
	cfg, loaded, err := dun.LoadConfig(root, explicitConfig)
	if err != nil {
		fmt.Fprintf(stderr, "dun iterate failed: config error: %v\n", err)
		return dun.ExitConfigError
	}
	if loaded {
		opts = dun.ApplyConfig(opts, cfg)
	}

	fs := flag.NewFlagSet("iterate", flag.ContinueOnError)
	fs.SetOutput(stderr)
	configPath := fs.String("config", explicitConfig, "path to config file")
	automation := fs.String("automation", opts.AutomationMode, "automation mode (manual|plan|auto|yolo)")
	if err := fs.Parse(args); err != nil {
		return dun.ExitUsageError
	}
	explicitConfig = *configPath

	// Force prompt mode for iterate - we detect work, don't execute it
	opts.AgentMode = "prompt"
	opts.AutomationMode = *automation
	result, err := checkRepo(root, opts)
	if err != nil {
		fmt.Fprintf(stderr, "dun iterate failed: %v\n", err)
		return dun.ExitCheckFailed
	}

	// Filter to actionable items (non-pass checks with prompts or issues)
	var actionable []dun.CheckResult
	for _, check := range result.Checks {
		if check.Status != "pass" {
			actionable = append(actionable, check)
		}
	}

	// Check for exit condition: all checks pass
	if len(actionable) == 0 {
		fmt.Fprintln(stdout, "---DUN_ITERATE---")
		fmt.Fprintln(stdout, "STATUS: ALL_PASS")
		fmt.Fprintln(stdout, "EXIT_SIGNAL: true")
		fmt.Fprintln(stdout, "MESSAGE: All checks pass. No work remaining.")
		fmt.Fprintln(stdout, "---END_DUN_ITERATE---")
		return dun.ExitSuccess
	}

	// Generate iteration prompt
	printIteratePrompt(stdout, actionable, *automation, root)
	return dun.ExitSuccess
}

func runLoop(args []string, stdout io.Writer, stderr io.Writer) int {
	root := resolveRoot(".")
	explicitConfig := findConfigFlag(args)
	opts := dun.DefaultOptions()
	cfg, loaded, err := dun.LoadConfig(root, explicitConfig)
	if err != nil {
		fmt.Fprintf(stderr, "dun loop failed: config error: %v\n", err)
		return dun.ExitConfigError
	}
	if loaded {
		opts = dun.ApplyConfig(opts, cfg)
	}

	fs := flag.NewFlagSet("loop", flag.ContinueOnError)
	fs.SetOutput(stderr)
	configPath := fs.String("config", explicitConfig, "path to config file")
	harness := fs.String("harness", "claude", "agent harness (claude|gemini|codex)")
	automation := fs.String("automation", opts.AutomationMode, "automation mode (manual|plan|auto|yolo)")
	maxIterations := fs.Int("max-iterations", 100, "maximum iterations before stopping")
	dryRun := fs.Bool("dry-run", false, "print prompt without calling harness")
	if err := fs.Parse(args); err != nil {
		return dun.ExitUsageError
	}
	_ = *configPath

	fmt.Fprintf(stdout, "Starting dun loop (harness=%s, automation=%s, max=%d)\n",
		*harness, *automation, *maxIterations)

	for i := 1; i <= *maxIterations; i++ {
		fmt.Fprintf(stdout, "\n=== Iteration %d/%d ===\n", i, *maxIterations)

		// Run iterate to get work list
		opts.AgentMode = "prompt"
		opts.AutomationMode = *automation
		result, err := checkRepo(root, opts)
		if err != nil {
			fmt.Fprintf(stderr, "check failed: %v\n", err)
			return dun.ExitCheckFailed
		}

		// Filter to actionable items
		var actionable []dun.CheckResult
		for _, check := range result.Checks {
			if check.Status != "pass" {
				actionable = append(actionable, check)
			}
		}

		// Exit condition: all checks pass
		if len(actionable) == 0 {
			fmt.Fprintln(stdout, "All checks pass. Loop complete.")
			return dun.ExitSuccess
		}

		// Generate prompt
		var promptBuf strings.Builder
		printIteratePrompt(&promptBuf, actionable, *automation, root)
		prompt := promptBuf.String()

		if *dryRun {
			fmt.Fprintln(stdout, "--- DRY RUN: Would send this prompt ---")
			fmt.Fprintln(stdout, prompt)
			fmt.Fprintln(stdout, "--- END DRY RUN ---")
			return dun.ExitSuccess
		}

		// Call harness
		response, err := callHarness(*harness, prompt, *automation)
		if err != nil {
			fmt.Fprintf(stderr, "harness call failed: %v\n", err)
			// Don't exit on harness failure - circuit breaker would handle this
			continue
		}

		fmt.Fprintf(stdout, "Harness response:\n%s\n", response)

		// Check for exit signal in response
		if strings.Contains(response, "EXIT_SIGNAL: true") {
			fmt.Fprintln(stdout, "Exit signal received. Loop complete.")
			return dun.ExitSuccess
		}

		// Brief pause between iterations
		time.Sleep(2 * time.Second)
	}

	fmt.Fprintf(stdout, "Max iterations (%d) reached. Stopping.\n", *maxIterations)
	return dun.ExitSuccess
}

func callHarness(harness, prompt, automation string) (string, error) {
	return callHarnessFn(harness, prompt, automation)
}

func callHarnessImpl(harness, prompt, automation string) (string, error) {
	var cmd *exec.Cmd

	switch harness {
	case "claude":
		args := []string{"-p", prompt, "--output-format", "text"}
		// Add yolo-mode permissions if in yolo mode
		if automation == "yolo" {
			args = append(args, "--dangerously-skip-permissions")
		}
		cmd = exec.Command("claude", args...)

	case "gemini":
		// Gemini doesn't have a standard CLI, use API via Python
		script := fmt.Sprintf(`
import google.generativeai as genai
import os
genai.configure(api_key=os.environ.get("GOOGLE_API_KEY", ""))
model = genai.GenerativeModel("gemini-1.5-flash")
response = model.generate_content("""%s""")
print(response.text)
`, strings.ReplaceAll(prompt, `"""`, `\"\"\"`))
		cmd = exec.Command("python3", "-c", script)

	case "codex":
		args := []string{"exec", "-p", prompt}
		if automation == "yolo" {
			args = append(args, "--ask-for-approval", "never")
		}
		cmd = exec.Command("codex", args...)

	default:
		return "", fmt.Errorf("unknown harness: %s", harness)
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	cmd.Dir = "."

	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("%v: %s", err, stderr.String())
	}

	return stdout.String(), nil
}

func printIteratePrompt(w io.Writer, checks []dun.CheckResult, automation string, root string) {
	fmt.Fprintln(w, "# Dun Iteration")
	fmt.Fprintln(w)
	fmt.Fprintf(w, "You are working in: %s\n", root)
	fmt.Fprintf(w, "Automation mode: %s\n", automation)
	fmt.Fprintln(w)
	fmt.Fprintln(w, "## Available Work")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Pick ONE task from this list. Choose the one with highest impact.")
	fmt.Fprintln(w)

	for i, check := range checks {
		priority := "MEDIUM"
		if check.Status == "error" {
			priority = "HIGH"
		} else if check.Status == "skip" {
			priority = "LOW"
		}

		fmt.Fprintf(w, "### %d. %s [%s]\n", i+1, check.ID, priority)
		fmt.Fprintf(w, "**Status:** %s\n", check.Status)
		if check.Signal != "" {
			fmt.Fprintf(w, "**Signal:** %s\n", check.Signal)
		}
		if check.Detail != "" {
			fmt.Fprintf(w, "**Detail:** %s\n", check.Detail)
		}
		if len(check.Issues) > 0 {
			fmt.Fprintln(w, "**Issues:**")
			for _, issue := range check.Issues {
				if issue.Path != "" {
					fmt.Fprintf(w, "- %s (%s)\n", issue.Summary, issue.Path)
				} else {
					fmt.Fprintf(w, "- %s\n", issue.Summary)
				}
			}
		}
		if check.Next != "" {
			fmt.Fprintf(w, "**Action:** %s\n", check.Next)
		}
		if check.Prompt != nil {
			fmt.Fprintf(w, "**Prompt available:** Use `dun explain %s` for details\n", check.ID)
		}
		fmt.Fprintln(w)
	}

	fmt.Fprintln(w, "---")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "## Instructions")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "1. Review the work items above")
	fmt.Fprintln(w, "2. Pick ONE task (highest priority or biggest impact)")
	fmt.Fprintln(w, "3. Complete that task fully:")
	fmt.Fprintln(w, "   - Edit files as needed")
	fmt.Fprintln(w, "   - Run tests to verify (`go test ./...`)")
	fmt.Fprintln(w, "   - Fix any issues that arise")
	fmt.Fprintln(w, "4. When done with that ONE task, EXIT")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Do NOT try to complete multiple tasks. The loop will call you again.")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "## Before You Exit")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Output this status block so the loop knows what happened:")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "```")
	fmt.Fprintln(w, "---DUN_STATUS---")
	fmt.Fprintln(w, "TASK_COMPLETED: <check-id>")
	fmt.Fprintln(w, "STATUS: COMPLETE | IN_PROGRESS | BLOCKED")
	fmt.Fprintln(w, "FILES_MODIFIED: <count>")
	fmt.Fprintln(w, "NEXT_RECOMMENDATION: <what to do next>")
	fmt.Fprintln(w, "---END_DUN_STATUS---")
	fmt.Fprintln(w, "```")
}

func formatRules(rules []dun.Rule) string {
	var parts []string
	for _, rule := range rules {
		desc := rule.Type
		if rule.Path != "" {
			desc += " path=" + rule.Path
		}
		if rule.Pattern != "" {
			desc += " pattern=" + rule.Pattern
		}
		if rule.Expected != 0 {
			desc += fmt.Sprintf(" expected=%d", rule.Expected)
		}
		parts = append(parts, desc)
	}
	return strings.Join(parts, "; ")
}

func resolveRoot(start string) string {
	root, err := dun.FindRepoRoot(start)
	if err != nil {
		return start
	}
	return root
}

func findConfigFlag(args []string) string {
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--config" {
			if i+1 < len(args) {
				return args[i+1]
			}
			return ""
		}
		if strings.HasPrefix(arg, "--config=") {
			return strings.TrimPrefix(arg, "--config=")
		}
	}
	return ""
}

func printLLM(stdout io.Writer, result dun.Result) {
	for _, check := range result.Checks {
		fmt.Fprintf(stdout, "check:%s status:%s\n", check.ID, check.Status)
		fmt.Fprintf(stdout, "signal: %s\n", check.Signal)
		if check.Detail != "" {
			fmt.Fprintf(stdout, "detail: %s\n", check.Detail)
		}
		if len(check.Issues) > 0 {
			for _, issue := range check.Issues {
				if issue.Path != "" {
					fmt.Fprintf(stdout, "issue: %s (%s)\n", issue.Summary, issue.Path)
				} else {
					fmt.Fprintf(stdout, "issue: %s\n", issue.Summary)
				}
			}
		}
		if check.Next != "" {
			fmt.Fprintf(stdout, "next: %s\n", check.Next)
		}
		fmt.Fprintln(stdout)
	}
}
