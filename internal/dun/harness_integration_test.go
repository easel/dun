package dun

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"
	"testing"
	"time"
)

const harnessIntegrationEnv = "DUN_CLI_INTEGRATION"

func TestClaudeHarnessIntegration(t *testing.T) {
	runHarnessIntegration(t, "claude", NewClaudeHarness)
}

func TestGeminiHarnessIntegration(t *testing.T) {
	runHarnessIntegration(t, "gemini", NewGeminiHarness)
}

func TestCodexHarnessIntegration(t *testing.T) {
	runHarnessIntegration(t, "codex", NewCodexHarness)
}

func runHarnessIntegration(t *testing.T, command string, factory HarnessFactory) {
	t.Helper()
	if os.Getenv(harnessIntegrationEnv) == "" {
		t.Skipf("set %s=1 to run CLI integration tests", harnessIntegrationEnv)
	}
	if _, err := exec.LookPath(command); err != nil {
		t.Skipf("%s not found on PATH", command)
	}

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}

	expected, err := firstSortedEntries(wd, 5)
	if err != nil {
		t.Fatalf("list entries: %v", err)
	}
	if len(expected) == 0 {
		t.Fatalf("no files found in %s", wd)
	}

	prompt := buildIntegrationPrompt(wd, expected)
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	harness := factory(HarnessConfig{
		Command:        command,
		AutomationMode: AutomationYolo,
		Timeout:        90 * time.Second,
	})

	response, err := harness.Execute(ctx, prompt)
	if err != nil {
		t.Fatalf("%s integration failed: %v", command, err)
	}
	if strings.TrimSpace(response) == "" {
		t.Fatalf("%s integration returned empty response", command)
	}
	if !strings.Contains(response, "PROOF:") {
		t.Fatalf("%s integration missing PROOF section", command)
	}
	for _, name := range expected {
		if !strings.Contains(response, name) {
			t.Fatalf("%s integration missing expected entry %q in response", command, name)
		}
	}
}

func firstSortedEntries(dir string, count int) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		names = append(names, entry.Name())
	}
	sort.Strings(names)
	if len(names) < count {
		count = len(names)
	}
	return names[:count], nil
}

func buildIntegrationPrompt(dir string, expected []string) string {
	var b strings.Builder
	b.WriteString("You are running inside a CLI agent harness.\n")
	b.WriteString("Your task MUST execute a shell command.\n\n")
	b.WriteString(fmt.Sprintf("Working directory: %s\n", dir))
	b.WriteString("Steps:\n")
	b.WriteString("1) Run: ls -1\n")
	b.WriteString("2) Show the raw command output under PROOF as a fenced code block.\n")
	b.WriteString("3) From that listing, output the first 5 entries alphabetically under FILES.\n")
	b.WriteString("4) Output COUNT: <number of entries printed under FILES>.\n\n")
	b.WriteString("Response format (exact sections):\n")
	b.WriteString("PROOF:\n")
	b.WriteString("```\n")
	b.WriteString("<raw ls -1 output>\n")
	b.WriteString("```\n")
	b.WriteString("FILES:\n")
	b.WriteString("- name\n")
	b.WriteString("COUNT: N\n\n")
	b.WriteString("Note: The first 5 alphabetical entries are expected to include:\n")
	b.WriteString(strings.Join(expected, ", "))
	b.WriteString("\n")
	return b.String()
}
