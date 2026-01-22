package dun

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"text/template"
	"time"
)

type AgentResponse struct {
	Status string  `json:"status"`
	Signal string  `json:"signal"`
	Detail string  `json:"detail"`
	Next   string  `json:"next"`
	Issues []Issue `json:"issues"`
}

type Issue struct {
	ID      string `json:"id"`
	Summary string `json:"summary"`
	Path    string `json:"path"`
}

type PromptInput struct {
	Path    string
	Content string
}

type PromptContext struct {
	CheckID string
	Inputs  []PromptInput
}

func runAgentCheck(root string, plugin Plugin, check Check, opts Options) (CheckResult, error) {
	mode := opts.AgentMode
	if mode == "" {
		mode = "ask"
	}

	if mode == "ask" {
		return CheckResult{
			ID:     check.ID,
			Status: "warn",
			Signal: "agent approval required",
			Detail: "Rerun with --agent-mode=auto to execute agent checks",
			Next:   "dun check --agent-mode=auto",
		}, nil
	}

	agentCmd := opts.AgentCmd
	if agentCmd == "" {
		agentCmd = os.Getenv("DUN_AGENT_CMD")
	}
	if agentCmd == "" {
		return CheckResult{
			ID:     check.ID,
			Status: "warn",
			Signal: "agent not configured",
			Detail: "Set DUN_AGENT_CMD to enable agent checks",
			Next:   "export DUN_AGENT_CMD=\"<agent command>\"",
		}, nil
	}

	prompt, err := renderPrompt(root, plugin, check)
	if err != nil {
		return CheckResult{}, err
	}

	timeout := opts.AgentTimeout
	if timeout == 0 {
		timeout = 300 * time.Second
	}

	resp, err := execAgent(agentCmd, prompt, timeout)
	if err != nil {
		return CheckResult{}, err
	}

	if resp.Status == "" || resp.Signal == "" {
		return CheckResult{}, fmt.Errorf("agent response missing required fields")
	}

	return CheckResult{
		ID:     check.ID,
		Status: resp.Status,
		Signal: resp.Signal,
		Detail: resp.Detail,
		Next:   resp.Next,
	}, nil
}

func renderPrompt(root string, plugin Plugin, check Check) (string, error) {
	inputs, err := resolveInputs(root, check.Inputs)
	if err != nil {
		return "", err
	}

	tmplText, err := loadPromptTemplate(plugin, check.Prompt)
	if err != nil {
		return "", err
	}

	tmpl, err := template.New("prompt").Parse(tmplText)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	ctx := PromptContext{
		CheckID: check.ID,
		Inputs:  inputs,
	}
	if err := tmpl.Execute(&buf, ctx); err != nil {
		return "", err
	}

	if check.ResponseSchema != "" {
		schemaText, err := loadPromptTemplate(plugin, check.ResponseSchema)
		if err != nil {
			return "", err
		}
		buf.WriteString("\n\nResponse Schema:\n")
		buf.WriteString(schemaText)
	}

	return buf.String(), nil
}

func loadPromptTemplate(plugin Plugin, promptPath string) (string, error) {
	if promptPath == "" {
		return "", fmt.Errorf("prompt path missing")
	}
	embeddedPath := path.Join(plugin.Base, promptPath)
	raw, err := fs.ReadFile(plugin.FS, embeddedPath)
	if err == nil {
		return string(raw), nil
	}
	if errors.Is(err, fs.ErrNotExist) {
		return promptPath, nil
	}
	return "", err
}

func resolveInputs(root string, inputs []string) ([]PromptInput, error) {
	var files []string
	for _, input := range inputs {
		if hasGlob(input) {
			matches, err := filepath.Glob(filepath.Join(root, input))
			if err != nil {
				return nil, err
			}
			files = append(files, matches...)
			continue
		}
		files = append(files, filepath.Join(root, input))
	}

	sort.Strings(files)
	var resolved []PromptInput
	for _, path := range files {
		content, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			rel = path
		}
		resolved = append(resolved, PromptInput{
			Path:    filepath.ToSlash(rel),
			Content: strings.TrimSpace(string(content)),
		})
	}
	return resolved, nil
}

func hasGlob(path string) bool {
	return strings.ContainsAny(path, "*?[")
}

func execAgent(cmdStr, prompt string, timeout time.Duration) (AgentResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "/bin/sh", "-c", cmdStr)
	cmd.Stdin = strings.NewReader(prompt)
	output, err := cmd.Output()
	if err != nil {
		return AgentResponse{}, fmt.Errorf("agent command failed: %w", err)
	}

	var resp AgentResponse
	if err := json.Unmarshal(bytes.TrimSpace(output), &resp); err != nil {
		return AgentResponse{}, fmt.Errorf("agent response parse error: %w", err)
	}
	return resp, nil
}
