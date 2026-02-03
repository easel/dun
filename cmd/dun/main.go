// dun CLI
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/easel/dun/internal/dun"
	"github.com/easel/dun/internal/update"
	"github.com/easel/dun/internal/version"
)

// Quorum-related sentinel errors.
var (
	errQuorumConflict = errors.New("quorum conflict")
	errQuorumAborted  = errors.New("quorum aborted")
)

var exit = os.Exit
var checkRepo = dun.CheckRepo
var planRepo = dun.PlanRepo
var respondFn = dun.Respond
var installRepo = dun.InstallRepo
var callHarnessFn = callHarnessImpl
var callHarnessStreamingFn = callHarnessStreamingImpl

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
	case "review":
		return runReview(args[1:], stdout, stderr)
	case "stamp":
		return runStamp(args[1:], stdout, stderr)
	case "install":
		return runInstall(args[1:], stdout, stderr)
	case "loop":
		return runLoop(args[1:], stdout, stderr)
	case "version":
		return runVersion(args[1:], stdout, stderr)
	case "update":
		return runUpdate(args[1:], stdout, stderr)
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
  review     Run multi-agent review with synthesis
  stamp      Update doc review stamps
  install    Install dun config and agent documentation
  loop       Run autonomous loop with an agent harness
  version    Show version information
  update     Update dun to the latest version

REVIEW MODE:
  dun review [options] [doc paths...]

  Runs a multi-agent review of the provided document(s) and synthesizes a
  single response using the synthesis harness.

  Options:
    --principles  Path to principles document (default docs/helix/01-frame/principles.md)
    --harnesses   Comma-separated list of review harnesses (default: codex,claude,gemini)
    --synth-harness Harness to synthesize final review (default: first harness)
    --automation  Mode: manual, plan, auto, yolo (default: auto)
    --dry-run     Print prompt without calling harnesses
    --verbose     Print individual harness reviews

CHECK MODE:
  dun check [options]

  Options:
    --prompt     Output the loop prompt for the current repo state
    --all        Include passing checks in prompt output
    --format     Output format: prompt, llm, json
    --automation Mode: manual, plan, auto, yolo (default: auto)
    --ignore-version  Skip .ddx-version check

LOOP MODE:
  dun loop [options]

  The loop command runs checks, generates prompts, calls an agent harness,
  and repeats until all checks pass or max iterations is reached.

  Options:
    --harness     Agent to use: codex, claude, gemini (default: from config)
    --automation  Mode: manual, plan, auto, yolo (default: auto)
    --max-iterations  Safety limit (default: 100)
    --dry-run     Show prompt without calling agent
    --verbose     Print prompts sent to harnesses and responses received
    --only        Comma-separated check IDs to include (supports * suffix)
    --ignore-version  Skip .ddx-version check

  Quorum Options (multi-agent consensus):
    --quorum      Strategy: any, majority, unanimous, or number (e.g., 2)
    --harnesses   Comma-separated list of harnesses (e.g., codex,claude,gemini)
    --cost-mode   Run harnesses sequentially to minimize cost
    --escalate    Pause for human review on conflict
    --prefer      Preferred harness on conflict (e.g., codex)
    --similarity  Similarity threshold for conflict detection (default: 0.8)

  Examples:
    dun loop                              # Run with default harness
    dun loop --harness gemini             # Run with gemini
    dun loop --automation yolo            # Allow autonomous edits
    dun loop --dry-run                    # Preview prompt
    dun loop --verbose                    # Show prompt and responses
    dun loop --quorum majority --harnesses codex,claude,gemini
    dun loop --quorum 2 --harnesses codex,claude --prefer codex

VERSION:
  dun version [options]

  Options:
    --json        Output version as JSON
    --check       Check for available updates

UPDATE:
  dun update [options]

  Options:
    --dry-run     Show what would be updated without applying
    --force       Force update even if already on latest version

AGENT DOCUMENTATION:
  Run 'dun install' to add AGENTS.md with instructions for AI agents.
  Agents can then run 'dun check --prompt' or 'dun loop' to work autonomously.

STAMP:
  dun stamp [--all] [paths...]

  Options:
    --all        Stamp all docs with dun frontmatter

EXIT CODES:
  0  Success / all checks pass
  1  Check failed
  2  Configuration error
  3  Runtime error
  4  Usage error
  5  Update error
  6  Quorum conflict (no consensus reached)
  7  Quorum aborted (user intervention)
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
	promptOut := fs.Bool("prompt", false, "output loop prompt")
	allChecks := fs.Bool("all", false, "include passing checks in prompt output")
	automation := fs.String("automation", opts.AutomationMode, "automation mode (manual|plan|auto|yolo)")
	ignoreVersion := fs.Bool("ignore-version", false, "skip .ddx-version check")
	if err := fs.Parse(args); err != nil {
		return dun.ExitUsageError
	}
	explicitConfig = *configPath

	opts.AgentMode = "prompt"
	opts.AutomationMode = *automation
	if !*ignoreVersion {
		if warn := checkDDXVersion(root); warn != "" {
			fmt.Fprintln(stderr, warn)
		}
	}
	result, err := checkRepo(root, opts)
	if err != nil {
		fmt.Fprintf(stderr, "dun check failed: %v\n", err)
		return dun.ExitCheckFailed
	}

	if *promptOut {
		checks := result.Checks
		if !*allChecks {
			var actionable []dun.CheckResult
			for _, check := range checks {
				if check.Status != "pass" {
					actionable = append(actionable, check)
				}
			}
			if len(actionable) == 0 {
				passCount := 0
				for _, check := range checks {
					if check.Status == "pass" {
						passCount++
					}
				}
				plugins, err := activePlugins(root)
				pluginsLine := "unknown"
				if err == nil {
					pluginsLine = strings.Join(plugins, ", ")
				}
				fmt.Fprintln(stdout, "---DUN_PROMPT---")
				fmt.Fprintln(stdout, "STATUS: ALL_PASS")
				fmt.Fprintln(stdout, "EXIT_SIGNAL: true")
				fmt.Fprintf(stdout, "CHECKS_PASSED: %d\n", passCount)
				fmt.Fprintf(stdout, "PLUGINS_ACTIVE: %s\n", pluginsLine)
				fmt.Fprintln(stdout, "MESSAGE: All checks pass. No work remaining.")
				fmt.Fprintln(stdout, "---END_DUN_PROMPT---")
				return dun.ExitSuccess
			}
			checks = actionable
		}
		printPrompt(stdout, checks, *automation, root)
		return dun.ExitSuccess
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

func runStamp(args []string, stdout io.Writer, stderr io.Writer) int {
	root := resolveRoot(".")
	fs := flag.NewFlagSet("stamp", flag.ContinueOnError)
	fs.SetOutput(stderr)
	all := fs.Bool("all", false, "stamp all docs with dun frontmatter")
	if err := fs.Parse(args); err != nil {
		return dun.ExitUsageError
	}
	if *all && fs.NArg() > 0 {
		fmt.Fprintln(stderr, "usage: dun stamp [--all] [paths...]")
		return dun.ExitUsageError
	}

	var stamped []string
	var err error
	if *all {
		stamped, err = dun.StampAll(root)
	} else {
		if fs.NArg() == 0 {
			fmt.Fprintln(stderr, "usage: dun stamp [--all] [paths...]")
			return dun.ExitUsageError
		}
		stamped, err = dun.StampDocs(root, fs.Args())
	}
	if err != nil {
		fmt.Fprintf(stderr, "dun stamp failed: %v\n", err)
		return dun.ExitRuntimeError
	}
	for _, path := range stamped {
		fmt.Fprintf(stdout, "stamped: %s\n", path)
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
	harness := fs.String("harness", "", "agent harness (codex|claude|gemini); default from config")
	automation := fs.String("automation", opts.AutomationMode, "automation mode (manual|plan|auto|yolo)")
	maxIterations := fs.Int("max-iterations", 100, "maximum iterations before stopping")
	dryRun := fs.Bool("dry-run", false, "print prompt without calling harness")
	verbose := fs.Bool("verbose", false, "print prompts and harness responses")
	only := fs.String("only", "", "comma-separated list of check IDs to include")
	ignoreVersion := fs.Bool("ignore-version", false, "skip .ddx-version check")

	// Quorum flags
	quorumFlag := fs.String("quorum", "", "quorum strategy: any, majority, unanimous, or number")
	harnessesFlag := fs.String("harnesses", "", "comma-separated list of harnesses for quorum")
	costMode := fs.Bool("cost-mode", false, "run harnesses sequentially to minimize cost")
	escalate := fs.Bool("escalate", false, "pause for human review on conflict")
	prefer := fs.String("prefer", "", "preferred harness on conflict")
	similarity := fs.Float64("similarity", 0.8, "similarity threshold for conflict detection")

	if err := fs.Parse(args); err != nil {
		return dun.ExitUsageError
	}
	_ = *configPath
	_ = *similarity // Reserved for future use in conflict detection
	if *harness == "" {
		if opts.AgentHarness != "" {
			*harness = opts.AgentHarness
		} else {
			*harness = "codex"
		}
	}

	// Parse quorum configuration if specified
	var quorumCfg dun.QuorumConfig
	if *quorumFlag != "" || *harnessesFlag != "" {
		quorumCfg, err = dun.ParseQuorumFlags(*quorumFlag, *harnessesFlag, *costMode, *escalate, *prefer)
		if err != nil {
			fmt.Fprintf(stderr, "dun loop failed: quorum config error: %v\n", err)
			return dun.ExitUsageError
		}
	}

	// Log startup info
	if quorumCfg.IsActive() {
		fmt.Fprintf(stdout, "Starting dun loop (quorum=%s, harnesses=%v, automation=%s, max=%d)\n",
			quorumStrategyName(quorumCfg), quorumCfg.Harnesses, *automation, *maxIterations)
	} else {
		fmt.Fprintf(stdout, "Starting dun loop (harness=%s, automation=%s, max=%d)\n",
			*harness, *automation, *maxIterations)
	}

	if !*ignoreVersion {
		if warn := checkDDXVersion(root); warn != "" {
			fmt.Fprintln(stderr, warn)
		}
	}

	for i := 1; i <= *maxIterations; i++ {
		fmt.Fprintf(stdout, "\n=== Iteration %d/%d ===\n", i, *maxIterations)

		// Run check to get work list
		opts.AgentMode = "prompt"
		opts.AutomationMode = *automation
		result, err := checkRepo(root, opts)
		if err != nil {
			fmt.Fprintf(stderr, "check failed: %v\n", err)
			return dun.ExitCheckFailed
		}

		checks := result.Checks
		if *only != "" {
			checks = filterChecksByID(checks, *only)
		}

		// Filter to actionable items
		var actionable []dun.CheckResult
		for _, check := range checks {
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
		printPrompt(&promptBuf, actionable, *automation, root)
		prompt := promptBuf.String()
		if *only != "" && len(actionable) == 1 && actionable[0].Prompt != nil {
			trimmed := strings.TrimSpace(actionable[0].Prompt.Prompt)
			if trimmed != "" {
				prompt = trimmed
			}
		}

		if *dryRun {
			fmt.Fprintln(stdout, "--- DRY RUN: Would send this prompt ---")
			fmt.Fprintln(stdout, prompt)
			fmt.Fprintln(stdout, "--- END DRY RUN ---")
			return dun.ExitSuccess
		}

		if *verbose {
			fmt.Fprintln(stdout, "--- PROMPT (to harnesses) ---")
			fmt.Fprintln(stdout, prompt)
			fmt.Fprintln(stdout, "--- END PROMPT ---")
		}

		var response string
		if quorumCfg.IsActive() {
			// Run quorum-based execution
			response, err = runQuorum(quorumCfg, prompt, *automation, stdout, stderr, *verbose)
			if err != nil {
				if errors.Is(err, errQuorumAborted) {
					fmt.Fprintln(stderr, "Quorum aborted by user.")
					return dun.ExitQuorumAborted
				}
				if errors.Is(err, errQuorumConflict) {
					fmt.Fprintln(stderr, "Quorum conflict: harnesses could not reach consensus.")
					return dun.ExitQuorumConflict
				}
				fmt.Fprintf(stderr, "quorum failed: %v\n", err)
				continue
			}
		} else {
			// Single harness call
			start := time.Now()
			if *verbose {
				fmt.Fprintf(stdout, "Calling %s...\n", *harness)
				response, err = callHarnessStreaming(*harness, prompt, *automation, stdout, stderr)
			} else {
				response, err = callHarness(*harness, prompt, *automation)
			}
			if *verbose {
				fmt.Fprintf(stdout, "%s completed in %s\n", *harness, time.Since(start).Round(time.Millisecond))
			}
			if err != nil {
				fmt.Fprintf(stderr, "harness call failed: %v\n", err)
				// Don't exit on harness failure - circuit breaker would handle this
				continue
			}
		}

		if *verbose {
			if !quorumCfg.IsActive() {
				fmt.Fprintf(stdout, "--- RESPONSE (%s) ---\n", *harness)
				fmt.Fprintln(stdout, response)
				fmt.Fprintln(stdout, "--- END RESPONSE ---")
			}
		} else {
			fmt.Fprintf(stdout, "Harness response:\n%s\n", response)
		}

		maybeAutoStampDocDag(root, actionable, response, stdout, stderr)

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

func maybeAutoStampDocDag(root string, actionable []dun.CheckResult, response string, stdout, stderr io.Writer) {
	if len(actionable) != 1 {
		return
	}
	check := actionable[0]
	if check.ID != "helix-doc-dag" || check.Prompt == nil {
		return
	}
	if status := parseAgentStatus(response); status == "fail" {
		return
	}
	docPath := extractDocPath(check.Prompt.Prompt)
	if docPath == "" {
		return
	}
	stamped, err := dun.StampDocs(root, []string{docPath})
	if err != nil {
		fmt.Fprintf(stderr, "auto-stamp failed for %s: %v\n", docPath, err)
		return
	}
	for _, path := range stamped {
		fmt.Fprintf(stdout, "auto-stamped: %s\n", path)
	}
}

func parseAgentStatus(response string) string {
	trimmed := strings.TrimSpace(response)
	if trimmed == "" {
		return ""
	}
	start := strings.Index(trimmed, "{")
	end := strings.LastIndex(trimmed, "}")
	if start == -1 || end == -1 || end <= start {
		return ""
	}
	var resp dun.AgentResponse
	if err := json.Unmarshal([]byte(trimmed[start:end+1]), &resp); err != nil {
		return ""
	}
	return resp.Status
}

func extractDocPath(prompt string) string {
	idx := strings.Index(prompt, "document `")
	if idx == -1 {
		return ""
	}
	start := idx + len("document `")
	if start >= len(prompt) {
		return ""
	}
	rest := prompt[start:]
	end := strings.Index(rest, "`")
	if end == -1 {
		return ""
	}
	return rest[:end]
}

func callHarness(harness, prompt, automation string) (string, error) {
	return callHarnessFn(harness, prompt, automation)
}

func callHarnessStreaming(harness, prompt, automation string, stdout, stderr io.Writer) (string, error) {
	return callHarnessStreamingFn(harness, prompt, automation, stdout, stderr)
}

func callHarnessImpl(harnessName, prompt, automation string) (string, error) {
	// Convert automation string to AutomationMode
	var mode dun.AutomationMode
	switch automation {
	case "manual":
		mode = dun.AutomationManual
	case "plan":
		mode = dun.AutomationPlan
	case "auto", "":
		mode = dun.AutomationAuto
	case "yolo":
		mode = dun.AutomationYolo
	default:
		mode = dun.AutomationAuto
	}

	// Use a timeout context to prevent hanging on unresponsive harnesses
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	result, err := dun.ExecuteHarness(ctx, harnessName, prompt, mode, ".")
	if err != nil {
		return "", err
	}

	return result.Response, nil
}

func callHarnessStreamingImpl(harnessName, prompt, automation string, stdout, stderr io.Writer) (string, error) {
	// Convert automation string to AutomationMode
	var mode dun.AutomationMode
	switch automation {
	case "manual":
		mode = dun.AutomationManual
	case "plan":
		mode = dun.AutomationPlan
	case "auto", "":
		mode = dun.AutomationAuto
	case "yolo":
		mode = dun.AutomationYolo
	default:
		mode = dun.AutomationAuto
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	result, err := dun.ExecuteHarnessWithOutput(ctx, harnessName, prompt, mode, ".", stdout, stderr)
	if err != nil {
		return "", err
	}

	return result.Response, nil
}

func printPrompt(w io.Writer, checks []dun.CheckResult, automation string, root string) {
	fmt.Fprintln(w, "# Dun Prompt")
	fmt.Fprintln(w)
	fmt.Fprintf(w, "You are working in: %s\n", root)
	fmt.Fprintf(w, "Automation mode: %s\n", automation)
	fmt.Fprintln(w)
	fmt.Fprintln(w, "## Available Work")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Pick ONE task from this list. Choose the one with highest impact.")
	fmt.Fprintln(w)

	if candidates := collectBeadsCandidates(checks); len(candidates) > 0 {
		fmt.Fprintln(w, "### Beads Candidates")
		for _, candidate := range candidates {
			line := candidate.ID
			if line == "" {
				line = candidate.Summary
			} else if candidate.Summary != "" {
				line = line + " - " + candidate.Summary
			}
			fmt.Fprintf(w, "- %s\n", line)
		}
		fmt.Fprintln(w, "For details, run: `bd show <id>` (optional: `bd comments <id>`)")
		fmt.Fprintln(w)
	}

	for i, check := range checks {
		priority := "MEDIUM"
		if check.Status == "error" {
			priority = "HIGH"
		} else if check.Status == "pass" {
			priority = "DONE"
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
			if strings.TrimSpace(check.Prompt.Prompt) != "" {
				fmt.Fprintln(w, "**Prompt:**")
				fmt.Fprintln(w, check.Prompt.Prompt)
			}
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

type beadsCandidate struct {
	ID      string
	Summary string
}

func collectBeadsCandidates(checks []dun.CheckResult) []beadsCandidate {
	var suggest []beadsCandidate
	var ready []beadsCandidate

	for _, check := range checks {
		switch check.ID {
		case "beads-suggest":
			suggest = issuesToBeadsCandidates(check.Issues)
		case "beads-ready":
			ready = issuesToBeadsCandidates(check.Issues)
		}
	}

	candidates := suggest
	if len(candidates) == 0 {
		candidates = ready
	}
	if len(candidates) > 3 {
		candidates = candidates[:3]
	}
	return candidates
}

func issuesToBeadsCandidates(issues []dun.Issue) []beadsCandidate {
	if len(issues) == 0 {
		return nil
	}
	candidates := make([]beadsCandidate, 0, len(issues))
	for _, issue := range issues {
		if issue.ID == "" && issue.Summary == "" {
			continue
		}
		candidates = append(candidates, beadsCandidate{
			ID:      issue.ID,
			Summary: issue.Summary,
		})
	}
	return candidates
}

func filterChecksByID(checks []dun.CheckResult, only string) []dun.CheckResult {
	parts := splitCSV(only)
	if len(parts) == 0 {
		return checks
	}
	out := make([]dun.CheckResult, 0, len(checks))
	for _, check := range checks {
		if matchesAny(check.ID, parts) {
			out = append(out, check)
		}
	}
	return out
}

func splitCSV(value string) []string {
	raw := strings.Split(value, ",")
	out := make([]string, 0, len(raw))
	for _, part := range raw {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func matchesAny(id string, patterns []string) bool {
	for _, pattern := range patterns {
		if pattern == id {
			return true
		}
		if strings.HasSuffix(pattern, "*") {
			prefix := strings.TrimSuffix(pattern, "*")
			if strings.HasPrefix(id, prefix) {
				return true
			}
		}
	}
	return false
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

func activePlugins(root string) ([]string, error) {
	plan, err := planRepo(root)
	if err != nil {
		return nil, err
	}
	seen := make(map[string]struct{})
	for _, check := range plan.Checks {
		if check.PluginID != "" {
			seen[check.PluginID] = struct{}{}
		}
	}
	plugins := make([]string, 0, len(seen))
	for id := range seen {
		plugins = append(plugins, id)
	}
	sort.Strings(plugins)
	return plugins, nil
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

func runVersion(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("version", flag.ContinueOnError)
	fs.SetOutput(stderr)
	jsonOutput := fs.Bool("json", false, "output version as JSON")
	checkUpdate := fs.Bool("check", false, "check for available updates")
	if err := fs.Parse(args); err != nil {
		return dun.ExitUsageError
	}

	info := version.Get()

	if *jsonOutput {
		out := map[string]string{
			"version":    info.Version,
			"commit":     info.Commit,
			"build_date": info.BuildDate,
			"go_version": info.GoVersion,
			"platform":   info.Platform,
		}
		if err := json.NewEncoder(stdout).Encode(out); err != nil {
			fmt.Fprintf(stderr, "encode json: %v\n", err)
			return dun.ExitRuntimeError
		}
		return dun.ExitSuccess
	}

	fmt.Fprintln(stdout, info.String())

	if *checkUpdate {
		fmt.Fprintln(stdout)
		updater := update.DefaultUpdater(info.Version)
		release, hasUpdate, err := updater.CheckForUpdate()
		if err != nil {
			fmt.Fprintf(stderr, "check for update failed: %v\n", err)
			return dun.ExitRuntimeError
		}
		if hasUpdate {
			fmt.Fprintf(stdout, "Update available: %s (current: %s)\n", release.TagName, info.Version)
			fmt.Fprintln(stdout, "Run 'dun update' to install the latest version.")
		} else {
			fmt.Fprintln(stdout, "You are running the latest version.")
		}
	}

	return dun.ExitSuccess
}

func runUpdate(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("update", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dryRun := fs.Bool("dry-run", false, "show what would be updated without applying")
	force := fs.Bool("force", false, "force update even if already on latest version")
	if err := fs.Parse(args); err != nil {
		return dun.ExitUsageError
	}

	info := version.Get()
	fmt.Fprintf(stdout, "Current version: %s\n", info.Version)

	updater := update.DefaultUpdater(info.Version)
	release, hasUpdate, err := updater.CheckForUpdate()
	if err != nil {
		fmt.Fprintf(stderr, "check for update failed: %v\n", err)
		return dun.ExitRuntimeError
	}

	if !hasUpdate && !*force {
		fmt.Fprintln(stdout, "Already running the latest version.")
		return dun.ExitSuccess
	}

	if release == nil {
		fmt.Fprintln(stderr, "No releases found.")
		return dun.ExitRuntimeError
	}

	fmt.Fprintf(stdout, "Latest version: %s\n", release.TagName)

	if *dryRun {
		fmt.Fprintln(stdout, "Dry run: would download and install the update.")
		return dun.ExitSuccess
	}

	fmt.Fprintln(stdout, "Downloading...")
	downloadPath, err := updater.DownloadRelease(release)
	if err != nil {
		fmt.Fprintf(stderr, "download failed: %v\n", err)
		return dun.ExitRuntimeError
	}

	fmt.Fprintln(stdout, "Installing...")
	if err := updater.ApplyUpdate(downloadPath); err != nil {
		fmt.Fprintf(stderr, "install failed: %v\n", err)
		return dun.ExitRuntimeError
	}

	fmt.Fprintf(stdout, "Successfully updated to %s\n", release.TagName)
	return dun.ExitSuccess
}

// quorumStrategyName returns a human-readable name for the quorum strategy.
func quorumStrategyName(cfg dun.QuorumConfig) string {
	if cfg.Strategy != "" {
		return cfg.Strategy
	}
	if cfg.Threshold > 0 {
		return fmt.Sprintf("%d", cfg.Threshold)
	}
	return "default"
}

// harnessResponse holds the result of a single harness call.
type harnessResponse struct {
	Harness  string
	Response string
	Err      error
}

// runQuorum executes the prompt against multiple harnesses and resolves consensus.
func runQuorum(cfg dun.QuorumConfig, prompt, automation string, stdout, stderr io.Writer, verbose bool) (string, error) {
	if len(cfg.Harnesses) == 0 {
		return "", errors.New("no harnesses configured for quorum")
	}

	responses := make([]harnessResponse, len(cfg.Harnesses))

	if cfg.Mode == "sequential" {
		// Sequential execution (cost mode)
		fmt.Fprintln(stdout, "Running harnesses sequentially (cost mode)...")
		for i, h := range cfg.Harnesses {
			fmt.Fprintf(stdout, "  Calling %s...\n", h)
			resp, err := callHarness(h, prompt, automation)
			responses[i] = harnessResponse{Harness: h, Response: resp, Err: err}
			if err != nil {
				fmt.Fprintf(stderr, "  %s failed: %v\n", h, err)
			} else {
				fmt.Fprintf(stdout, "  %s completed.\n", h)
			}
		}
	} else {
		// Parallel execution
		fmt.Fprintln(stdout, "Running harnesses in parallel...")
		var wg sync.WaitGroup
		for i, h := range cfg.Harnesses {
			wg.Add(1)
			go func(idx int, harness string) {
				defer wg.Done()
				resp, err := callHarness(harness, prompt, automation)
				responses[idx] = harnessResponse{Harness: harness, Response: resp, Err: err}
			}(i, h)
		}
		wg.Wait()

		// Report results
		for _, r := range responses {
			if r.Err != nil {
				fmt.Fprintf(stderr, "  %s failed: %v\n", r.Harness, r.Err)
			} else {
				fmt.Fprintf(stdout, "  %s completed.\n", r.Harness)
			}
		}
	}

	if verbose {
		for _, r := range responses {
			if r.Err != nil {
				continue
			}
			fmt.Fprintf(stdout, "--- RESPONSE (%s) ---\n", r.Harness)
			fmt.Fprintln(stdout, r.Response)
			fmt.Fprintln(stdout, "--- END RESPONSE ---")
		}
	}

	// Collect successful responses
	var successful []harnessResponse
	for _, r := range responses {
		if r.Err == nil {
			successful = append(successful, r)
		}
	}

	if len(successful) == 0 {
		return "", errors.New("all harnesses failed")
	}

	// Check if quorum is met
	if !cfg.IsMet(len(successful), len(cfg.Harnesses)) {
		return "", fmt.Errorf("quorum not met: %d/%d successful", len(successful), len(cfg.Harnesses))
	}

	// Check for conflicts (simple check: all responses should be similar)
	// For now, we use a simple check: if all responses contain EXIT_SIGNAL, consider them agreeing
	exitSignalCount := 0
	for _, r := range successful {
		if strings.Contains(r.Response, "EXIT_SIGNAL: true") {
			exitSignalCount++
		}
	}

	// Detect conflict: some say exit, some don't
	hasConflict := exitSignalCount > 0 && exitSignalCount < len(successful)

	if hasConflict {
		fmt.Fprintln(stdout, "Conflict detected: harnesses disagree on exit signal.")
		if cfg.Escalate {
			fmt.Fprintln(stderr, "Escalating to human review due to conflict.")
			return "", errQuorumAborted
		}
		if cfg.Prefer != "" {
			// Use preferred harness response
			for _, r := range successful {
				if r.Harness == cfg.Prefer {
					fmt.Fprintf(stdout, "Using preferred harness response: %s\n", cfg.Prefer)
					return r.Response, nil
				}
			}
		}
		// No preferred harness found, report conflict
		return "", errQuorumConflict
	}

	// No conflict or all agree - use first successful response
	// Prefer the preferred harness if specified
	if cfg.Prefer != "" {
		for _, r := range successful {
			if r.Harness == cfg.Prefer {
				return r.Response, nil
			}
		}
	}

	return successful[0].Response, nil
}
