//go:build ignore

// Option B: Go shelling out to Claude CLI
//
// Pros:
// - Structured error handling
// - Can parse JSON output
// - Integrates with dun's existing Go codebase
// - Fresh context each iteration
//
// Cons:
// - Subprocess overhead
// - Depends on claude CLI being installed
// - Output parsing can be fragile

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

const prompt = `Here is a list of 10 words. Pick your favorite word, make it **bold**, and move it to OUTPUT.

INPUT: apple banana cherry date elderberry fig grape honeydew kiwi lemon
OUTPUT:

Respond with ONLY the updated INPUT/OUTPUT block, nothing else.`

type ClaudeResponse struct {
	Result string `json:"result"`
	Error  string `json:"error,omitempty"`
}

func runClaude(prompt string) (string, error) {
	cmd := exec.Command("claude",
		"-p", prompt,
		"--output-format", "text",
	)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("claude failed: %v\nstderr: %s", err, stderr.String())
	}

	return strings.TrimSpace(stdout.String()), nil
}

func parseResult(output string) (input, boldWord string, err error) {
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "INPUT:") {
			input = strings.TrimPrefix(line, "INPUT:")
			input = strings.TrimSpace(input)
		}
		if strings.HasPrefix(line, "OUTPUT:") {
			boldWord = strings.TrimPrefix(line, "OUTPUT:")
			boldWord = strings.TrimSpace(boldWord)
		}
	}
	if boldWord == "" {
		return "", "", fmt.Errorf("no OUTPUT found in response")
	}
	return input, boldWord, nil
}

func main() {
	fmt.Println("=== Option B: Go + Claude CLI ===")
	fmt.Println("Running claude...")
	fmt.Println()

	result, err := runClaude(prompt)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Raw output:")
	fmt.Println(result)
	fmt.Println()

	input, boldWord, err := parseResult(result)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Parse error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Parsed:")
	fmt.Printf("  INPUT: %s\n", input)
	fmt.Printf("  OUTPUT (bold word): %s\n", boldWord)

	// Could output as JSON for further processing
	out, _ := json.MarshalIndent(map[string]string{
		"input":     input,
		"bold_word": boldWord,
	}, "", "  ")
	fmt.Println()
	fmt.Println("JSON:")
	fmt.Println(string(out))
}
