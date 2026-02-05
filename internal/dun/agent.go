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
	CheckID        string
	Inputs         []PromptInput
	AutomationMode string
}

func runAgentCheck(root string, plugin Plugin, def CheckDefinition, config AgentCheckConfig, opts Options) (CheckResult, error) {
	mode, err := normalizeAgentMode(opts.AgentMode)
	if err != nil {
		return CheckResult{}, err
	}
	automation, err := normalizeAutomationMode(opts.AutomationMode)
	if err != nil {
		return CheckResult{}, err
	}

	envelope, err := buildPromptEnvelope(root, plugin, def, config, automation)
	if err != nil {
		return CheckResult{}, err
	}

	if mode != "auto" {
		return promptResult(def, envelope, "agent prompt ready", def.Description), nil
	}

	agentCmd := opts.AgentCmd
	if agentCmd == "" {
		agentCmd = os.Getenv("DUN_AGENT_CMD")
	}
	if agentCmd == "" {
		return promptResult(def, envelope, "agent not configured", "set --agent-cmd or DUN_AGENT_CMD to run in auto mode"), nil
	}

	timeout := opts.AgentTimeout
	if timeout == 0 {
		timeout = 300 * time.Second
	}

	resp, err := execAgent(agentCmd, envelope.Prompt, timeout)
	if err != nil {
		return CheckResult{}, err
	}

	if resp.Status == "" || resp.Signal == "" {
		return CheckResult{}, fmt.Errorf("agent response missing required fields")
	}

	return CheckResult{
		ID:     def.ID,
		Status: resp.Status,
		Signal: resp.Signal,
		Detail: resp.Detail,
		Next:   resp.Next,
		Issues: resp.Issues,
	}, nil
}

func normalizeAgentMode(mode string) (string, error) {
	switch mode {
	case "", "prompt", "ask":
		return "prompt", nil
	case "auto":
		return "auto", nil
	default:
		return "", fmt.Errorf("unknown agent mode: %s", mode)
	}
}

func normalizeAutomationMode(mode string) (string, error) {
	switch mode {
	case "", "auto":
		return "auto", nil
	case "manual":
		return "manual", nil
	case "plan", "yolo":
		return mode, nil
	default:
		return "", fmt.Errorf("unknown automation mode: %s", mode)
	}
}

func promptResult(def CheckDefinition, envelope PromptEnvelope, signal string, detail string) CheckResult {
	next := envelope.Callback.Command
	if next == "" {
		next = fmt.Sprintf("dun respond --id %s --response -", def.ID)
	}
	return CheckResult{
		ID:     def.ID,
		Status: "prompt",
		Signal: signal,
		Detail: detail,
		Next:   next,
		Prompt: &envelope,
	}
}

func buildPromptEnvelope(root string, plugin Plugin, def CheckDefinition, config AgentCheckConfig, automationMode string) (PromptEnvelope, error) {
	inputs, err := resolveInputs(root, config.Inputs)
	if err != nil {
		return PromptEnvelope{}, err
	}

	promptText, schemaText, err := renderPromptText(plugin, config, def.ID, inputs, automationMode)
	if err != nil {
		return PromptEnvelope{}, err
	}

	inputPaths := make([]string, 0, len(inputs))
	for _, input := range inputs {
		inputPaths = append(inputPaths, input.Path)
	}

	return PromptEnvelope{
		Kind:           "dun.prompt.v1",
		ID:             def.ID,
		Title:          def.Description,
		Summary:        def.Description,
		Prompt:         promptText,
		Inputs:         inputPaths,
		ResponseSchema: schemaText,
		Callback: PromptCallback{
			Command: fmt.Sprintf("dun respond --id %s --response -", def.ID),
			Stdin:   true,
		},
	}, nil
}

func renderPromptText(plugin Plugin, config AgentCheckConfig, checkID string, inputs []PromptInput, automationMode string) (string, string, error) {
	tmplText, err := loadPromptTemplate(plugin, config.Prompt)
	if err != nil {
		return "", "", err
	}

	tmpl, err := template.New("prompt").Parse(tmplText)
	if err != nil {
		return "", "", err
	}

	var buf bytes.Buffer
	ctx := PromptContext{
		CheckID:        checkID,
		Inputs:         inputs,
		AutomationMode: automationMode,
	}
	if err := tmpl.Execute(&buf, ctx); err != nil {
		return "", "", err
	}

	var schemaText string
	if config.ResponseSchema != "" {
		loaded, err := loadPromptTemplate(plugin, config.ResponseSchema)
		if err != nil {
			return "", "", err
		}
		schemaText = loaded
		buf.WriteString("\n\nResponse Schema:\n")
		buf.WriteString(schemaText)
	}

	return buf.String(), schemaText, nil
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
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}
		rel, err := relPath(root, path)
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

var relPath = filepath.Rel

func execAgent(cmdStr, prompt string, timeout time.Duration) (AgentResponse, error) {
	output, err := execAgentOutput(cmdStr, prompt, timeout)
	if err != nil {
		return AgentResponse{}, fmt.Errorf("agent command failed: %w", err)
	}

	var resp AgentResponse
	if err := json.Unmarshal(bytes.TrimSpace(output), &resp); err != nil {
		return AgentResponse{}, fmt.Errorf("agent response parse error: %w", err)
	}
	return resp, nil
}

var execAgentOutput = execAgentShell

func execAgentShell(cmdStr, prompt string, timeout time.Duration) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "/bin/sh", "-c", cmdStr)
	cmd.Stdin = strings.NewReader(prompt)
	return cmd.Output()
}
