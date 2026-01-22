package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
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
	fs.Parse(args)

	opts := dun.Options{
		AgentCmd:     *agentCmd,
		AgentTimeout: time.Duration(*agentTimeout) * time.Second,
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
