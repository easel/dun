package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/easel/dun/internal/dun"
)

var exit = os.Exit
var checkRepo = dun.CheckRepo
var planRepo = dun.PlanRepo
var respondFn = dun.Respond
var installRepo = dun.InstallRepo

func main() {
	code := run(os.Args[1:], os.Stdout, os.Stderr)
	exit(code)
}

func run(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) < 1 {
		return runCheck(args, stdout, stderr)
	}
	switch args[0] {
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
	default:
		fmt.Fprintf(stderr, "unknown command: %s\n", args[0])
		return 1
	}
}

func runCheck(args []string, stdout io.Writer, stderr io.Writer) int {
	root := resolveRoot(".")
	explicitConfig := findConfigFlag(args)
	opts := dun.DefaultOptions()
	cfg, loaded, err := dun.LoadConfig(root, explicitConfig)
	if err != nil {
		fmt.Fprintf(stderr, "dun check failed: config error: %v\n", err)
		return 4
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
		return 4
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
		return 1
	}

	switch *format {
	case "llm":
		printLLM(stdout, result)
	case "json", "prompt":
		if err := json.NewEncoder(stdout).Encode(result); err != nil {
			fmt.Fprintf(stderr, "encode json: %v\n", err)
			return 1
		}
	default:
		fmt.Fprintf(stderr, "unknown format: %s\n", *format)
		return 1
	}
	return 0
}

func runList(args []string, stdout io.Writer, stderr io.Writer) int {
	root := resolveRoot(".")
	fs := flag.NewFlagSet("list", flag.ContinueOnError)
	fs.SetOutput(stderr)
	format := fs.String("format", "text", "output format (text|json)")
	configPath := fs.String("config", "", "path to config file (default .dun/config.yaml if present)")
	if err := fs.Parse(args); err != nil {
		return 4
	}

	if _, _, err := dun.LoadConfig(root, *configPath); err != nil {
		fmt.Fprintf(stderr, "dun list failed: config error: %v\n", err)
		return 4
	}

	plan, err := planRepo(root)
	if err != nil {
		fmt.Fprintf(stderr, "dun list failed: %v\n", err)
		return 1
	}

	switch *format {
	case "json":
		if err := json.NewEncoder(stdout).Encode(plan); err != nil {
			fmt.Fprintf(stderr, "encode json: %v\n", err)
			return 1
		}
	default:
		for _, check := range plan.Checks {
			fmt.Fprintf(stdout, "%s\t%s\n", check.ID, check.Description)
		}
	}
	return 0
}

func runExplain(args []string, stdout io.Writer, stderr io.Writer) int {
	root := resolveRoot(".")
	fs := flag.NewFlagSet("explain", flag.ContinueOnError)
	fs.SetOutput(stderr)
	format := fs.String("format", "text", "output format (text|json)")
	configPath := fs.String("config", "", "path to config file (default .dun/config.yaml if present)")
	if err := fs.Parse(args); err != nil {
		return 4
	}

	if fs.NArg() < 1 {
		fmt.Fprintln(stderr, "usage: dun explain <check-id>")
		return 4
	}
	target := fs.Arg(0)

	if _, _, err := dun.LoadConfig(root, *configPath); err != nil {
		fmt.Fprintf(stderr, "dun explain failed: config error: %v\n", err)
		return 4
	}

	plan, err := planRepo(root)
	if err != nil {
		fmt.Fprintf(stderr, "dun explain failed: %v\n", err)
		return 1
	}

	for _, check := range plan.Checks {
		if check.ID != target {
			continue
		}
		switch *format {
		case "json":
			if err := json.NewEncoder(stdout).Encode(check); err != nil {
				fmt.Fprintf(stderr, "encode json: %v\n", err)
				return 1
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
		return 0
	}

	fmt.Fprintf(stderr, "unknown check: %s\n", target)
	return 1
}

func runRespond(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("respond", flag.ContinueOnError)
	fs.SetOutput(stderr)
	id := fs.String("id", "", "check id from prompt")
	responsePath := fs.String("response", "-", "response JSON path or - for stdin")
	format := fs.String("format", "json", "output format (json|llm)")
	if err := fs.Parse(args); err != nil {
		return 4
	}

	if *id == "" {
		fmt.Fprintln(stderr, "usage: dun respond --id <check-id> --response <path|->")
		return 4
	}

	var reader io.Reader = os.Stdin
	if *responsePath != "-" {
		file, err := os.Open(*responsePath)
		if err != nil {
			fmt.Fprintf(stderr, "open response: %v\n", err)
			return 1
		}
		defer file.Close()
		reader = file
	}

	check, err := respondFn(*id, reader)
	if err != nil {
		fmt.Fprintf(stderr, "dun respond failed: %v\n", err)
		return 1
	}

	result := dun.Result{Checks: []dun.CheckResult{check}}
	switch *format {
	case "llm":
		printLLM(stdout, result)
	case "json":
		if err := json.NewEncoder(stdout).Encode(check); err != nil {
			fmt.Fprintf(stderr, "encode json: %v\n", err)
			return 1
		}
	default:
		fmt.Fprintf(stderr, "unknown format: %s\n", *format)
		return 1
	}
	return 0
}

func runInstall(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("install", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dryRun := fs.Bool("dry-run", false, "show planned changes without writing")
	if err := fs.Parse(args); err != nil {
		return 4
	}

	result, err := installRepo(".", dun.InstallOptions{DryRun: *dryRun})
	if err != nil {
		fmt.Fprintf(stderr, "dun install failed: %v\n", err)
		return 1
	}

	for _, step := range result.Steps {
		if *dryRun {
			fmt.Fprintf(stdout, "plan: %s %s\n", step.Action, step.Path)
		} else {
			fmt.Fprintf(stdout, "installed: %s %s\n", step.Action, step.Path)
		}
	}

	fmt.Fprintln(stdout, "note: add hooks manually if desired (lefthook/pre-commit)")
	return 0
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
