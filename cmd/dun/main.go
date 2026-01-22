package main

import (
	"encoding/json"
	"flag"
	"fmt"
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
	case "install":
		runInstall(os.Args[2:])
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", os.Args[1])
		os.Exit(1)
	}
}

func runCheck(args []string) {
	fs := flag.NewFlagSet("check", flag.ExitOnError)
	format := fs.String("format", "llm", "output format (llm|json)")
	agentCmd := fs.String("agent-cmd", "", "agent command override")
	agentTimeout := fs.Int("agent-timeout", 300, "agent timeout in seconds")
	agentMode := fs.String("agent-mode", "ask", "agent mode (ask|auto)")
	fs.Parse(args)

	opts := dun.Options{
		AgentCmd:     *agentCmd,
		AgentTimeout: time.Duration(*agentTimeout) * time.Second,
		AgentMode:    *agentMode,
	}
	result, err := dun.CheckRepo(".", opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "dun check failed: %v\n", err)
		os.Exit(1)
	}

	switch *format {
	case "json":
		if err := json.NewEncoder(os.Stdout).Encode(result); err != nil {
			fmt.Fprintf(os.Stderr, "encode json: %v\n", err)
			os.Exit(1)
		}
	default:
		printLLM(result)
	}
}

func runList(args []string) {
	fs := flag.NewFlagSet("list", flag.ExitOnError)
	format := fs.String("format", "text", "output format (text|json)")
	fs.Parse(args)

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
	fs.Parse(args)

	if fs.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "usage: dun explain <check-id>")
		os.Exit(1)
	}
	target := fs.Arg(0)

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

func printLLM(result dun.Result) {
	for _, check := range result.Checks {
		fmt.Printf("check:%s status:%s\n", check.ID, check.Status)
		fmt.Printf("signal: %s\n", check.Signal)
		if check.Detail != "" {
			fmt.Printf("detail: %s\n", check.Detail)
		}
		if check.Next != "" {
			fmt.Printf("next: %s\n", check.Next)
		}
		fmt.Println()
	}
}
