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

func main() {
	if len(os.Args) < 2 {
		runCheck(os.Args[1:])
		return
	}
	switch os.Args[1] {
	case "check":
		runCheck(os.Args[2:])
	case "list":
		runList(os.Args[2:])
	case "explain":
		runExplain(os.Args[2:])
	case "respond":
		runRespond(os.Args[2:])
	case "install":
		runInstall(os.Args[2:])
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", os.Args[1])
		os.Exit(1)
	}
}

func runCheck(args []string) {
	explicitConfig := findConfigFlag(args)
	opts := dun.DefaultOptions()
	cfg, loaded, err := dun.LoadConfig(".", explicitConfig)
	if err != nil {
		fmt.Fprintf(os.Stderr, "dun check failed: config error: %v\n", err)
		os.Exit(4)
	}
	if loaded {
		opts = dun.ApplyConfig(opts, cfg)
	}

	fs := flag.NewFlagSet("check", flag.ExitOnError)
	configPath := fs.String("config", explicitConfig, "path to config file (default .dun/config.yaml if present)")
	format := fs.String("format", "prompt", "output format (prompt|llm|json)")
	agentCmd := fs.String("agent-cmd", opts.AgentCmd, "agent command override")
	agentTimeout := fs.Int("agent-timeout", int(opts.AgentTimeout/time.Second), "agent timeout in seconds")
	agentMode := fs.String("agent-mode", opts.AgentMode, "agent mode (prompt|auto)")
	automation := fs.String("automation", opts.AutomationMode, "automation mode (manual|plan|auto|yolo)")
	fs.Parse(args)
	explicitConfig = *configPath

	opts = dun.Options{
		AgentCmd:       *agentCmd,
		AgentTimeout:   time.Duration(*agentTimeout) * time.Second,
		AgentMode:      *agentMode,
		AutomationMode: *automation,
	}
	result, err := dun.CheckRepo(".", opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "dun check failed: %v\n", err)
		os.Exit(1)
	}

	switch *format {
	case "llm":
		printLLM(result)
	case "json", "prompt":
		if err := json.NewEncoder(os.Stdout).Encode(result); err != nil {
			fmt.Fprintf(os.Stderr, "encode json: %v\n", err)
			os.Exit(1)
		}
	default:
		fmt.Fprintf(os.Stderr, "unknown format: %s\n", *format)
		os.Exit(1)
	}
}

func runList(args []string) {
	fs := flag.NewFlagSet("list", flag.ExitOnError)
	format := fs.String("format", "text", "output format (text|json)")
	configPath := fs.String("config", "", "path to config file (default .dun/config.yaml if present)")
	fs.Parse(args)

	if _, _, err := dun.LoadConfig(".", *configPath); err != nil {
		fmt.Fprintf(os.Stderr, "dun list failed: config error: %v\n", err)
		os.Exit(4)
	}

	plan, err := dun.PlanRepo(".")
	if err != nil {
		fmt.Fprintf(os.Stderr, "dun list failed: %v\n", err)
		os.Exit(1)
	}

	switch *format {
	case "json":
		if err := json.NewEncoder(os.Stdout).Encode(plan); err != nil {
			fmt.Fprintf(os.Stderr, "encode json: %v\n", err)
			os.Exit(1)
		}
	default:
		for _, check := range plan.Checks {
			fmt.Printf("%s\t%s\n", check.ID, check.Description)
		}
	}
}

func runExplain(args []string) {
	fs := flag.NewFlagSet("explain", flag.ExitOnError)
	format := fs.String("format", "text", "output format (text|json)")
	configPath := fs.String("config", "", "path to config file (default .dun/config.yaml if present)")
	fs.Parse(args)

	if fs.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "usage: dun explain <check-id>")
		os.Exit(1)
	}
	target := fs.Arg(0)

	if _, _, err := dun.LoadConfig(".", *configPath); err != nil {
		fmt.Fprintf(os.Stderr, "dun explain failed: config error: %v\n", err)
		os.Exit(4)
	}

	plan, err := dun.PlanRepo(".")
	if err != nil {
		fmt.Fprintf(os.Stderr, "dun explain failed: %v\n", err)
		os.Exit(1)
	}

	for _, check := range plan.Checks {
		if check.ID != target {
			continue
		}
		switch *format {
		case "json":
			if err := json.NewEncoder(os.Stdout).Encode(check); err != nil {
				fmt.Fprintf(os.Stderr, "encode json: %v\n", err)
				os.Exit(1)
			}
		default:
			fmt.Printf("id: %s\n", check.ID)
			fmt.Printf("description: %s\n", check.Description)
			fmt.Printf("plugin: %s\n", check.PluginID)
			fmt.Printf("type: %s\n", check.Type)
			if check.Phase != "" {
				fmt.Printf("phase: %s\n", check.Phase)
			}
			if len(check.Conditions) > 0 {
				fmt.Printf("conditions: %s\n", formatRules(check.Conditions))
			}
			if len(check.Inputs) > 0 {
				fmt.Printf("inputs: %s\n", strings.Join(check.Inputs, ", "))
			}
			if len(check.GateFiles) > 0 {
				fmt.Printf("gate_files: %s\n", strings.Join(check.GateFiles, ", "))
			}
			if check.StateRules != "" {
				fmt.Printf("state_rules: %s\n", check.StateRules)
			}
			if check.Prompt != "" {
				fmt.Printf("prompt: %s\n", check.Prompt)
			}
		}
		return
	}

	fmt.Fprintf(os.Stderr, "unknown check: %s\n", target)
	os.Exit(1)
}

func runRespond(args []string) {
	fs := flag.NewFlagSet("respond", flag.ExitOnError)
	id := fs.String("id", "", "check id from prompt")
	responsePath := fs.String("response", "-", "response JSON path or - for stdin")
	format := fs.String("format", "json", "output format (json|llm)")
	fs.Parse(args)

	if *id == "" {
		fmt.Fprintln(os.Stderr, "usage: dun respond --id <check-id> --response <path|->")
		os.Exit(1)
	}

	var reader io.Reader = os.Stdin
	if *responsePath != "-" {
		file, err := os.Open(*responsePath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "open response: %v\n", err)
			os.Exit(1)
		}
		defer file.Close()
		reader = file
	}

	check, err := dun.Respond(*id, reader)
	if err != nil {
		fmt.Fprintf(os.Stderr, "dun respond failed: %v\n", err)
		os.Exit(1)
	}

	result := dun.Result{Checks: []dun.CheckResult{check}}
	switch *format {
	case "llm":
		printLLM(result)
	case "json":
		if err := json.NewEncoder(os.Stdout).Encode(check); err != nil {
			fmt.Fprintf(os.Stderr, "encode json: %v\n", err)
			os.Exit(1)
		}
	default:
		fmt.Fprintf(os.Stderr, "unknown format: %s\n", *format)
		os.Exit(1)
	}
}

func runInstall(args []string) {
	fs := flag.NewFlagSet("install", flag.ExitOnError)
	dryRun := fs.Bool("dry-run", false, "show planned changes without writing")
	fs.Parse(args)

	result, err := dun.InstallRepo(".", dun.InstallOptions{DryRun: *dryRun})
	if err != nil {
		fmt.Fprintf(os.Stderr, "dun install failed: %v\n", err)
		os.Exit(1)
	}

	for _, step := range result.Steps {
		if *dryRun {
			fmt.Printf("plan: %s %s\n", step.Action, step.Path)
		} else {
			fmt.Printf("installed: %s %s\n", step.Action, step.Path)
		}
	}

	fmt.Println("note: add hooks manually if desired (lefthook/pre-commit)")
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

func printLLM(result dun.Result) {
	for _, check := range result.Checks {
		fmt.Printf("check:%s status:%s\n", check.ID, check.Status)
		fmt.Printf("signal: %s\n", check.Signal)
		if check.Detail != "" {
			fmt.Printf("detail: %s\n", check.Detail)
		}
		if len(check.Issues) > 0 {
			for _, issue := range check.Issues {
				if issue.Path != "" {
					fmt.Printf("issue: %s (%s)\n", issue.Summary, issue.Path)
				} else {
					fmt.Printf("issue: %s\n", issue.Summary)
				}
			}
		}
		if check.Next != "" {
			fmt.Printf("next: %s\n", check.Next)
		}
		fmt.Println()
	}
}
