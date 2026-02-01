package dun

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"sync"
	"time"
)

// AutomationMode represents the level of automation allowed for a harness.
type AutomationMode string

const (
	AutomationManual AutomationMode = "manual"
	AutomationPlan   AutomationMode = "plan"
	AutomationAuto   AutomationMode = "auto"
	AutomationYolo   AutomationMode = "yolo"
)

// Harness defines the interface for agent execution harnesses.
// A harness wraps an LLM agent CLI (claude, gemini, codex, etc.) and provides
// a consistent interface for executing prompts and retrieving responses.
type Harness interface {
	// Name returns the unique identifier for this harness.
	Name() string

	// Execute sends a prompt to the agent and returns the response.
	// The context can be used for cancellation and timeout.
	Execute(ctx context.Context, prompt string) (string, error)

	// SupportsAutomation returns true if this harness supports the given automation mode.
	SupportsAutomation(mode AutomationMode) bool
}

// HarnessResult captures the outcome of a harness execution.
type HarnessResult struct {
	Harness   string        // Name of the harness used
	Response  string        // Response text from the agent
	Error     error         // Error if execution failed
	Duration  time.Duration // Time taken for execution
	Timestamp time.Time     // When the execution started
}

// HarnessConfig holds configuration for initializing a harness.
type HarnessConfig struct {
	// Name is the harness identifier (e.g., "claude", "gemini", "codex")
	Name string

	// Command is the base command to execute (optional, uses default if empty)
	Command string

	// WorkDir is the working directory for command execution
	WorkDir string

	// Timeout is the maximum execution time (0 means no timeout)
	Timeout time.Duration

	// AutomationMode is the current automation level
	AutomationMode AutomationMode

	// Environment variables to set for command execution
	Env map[string]string

	// MockResponse is used by MockHarness for testing
	MockResponse string

	// MockError is used by MockHarness for testing
	MockError error

	// MockDelay is used by MockHarness to simulate execution time
	MockDelay time.Duration
}

// HarnessFactory is a function that creates a Harness from configuration.
type HarnessFactory func(config HarnessConfig) Harness

// HarnessRegistry manages harness factories and creates harness instances.
type HarnessRegistry struct {
	mu        sync.RWMutex
	factories map[string]HarnessFactory
}

// NewHarnessRegistry creates a new registry with default harnesses registered.
func NewHarnessRegistry() *HarnessRegistry {
	r := &HarnessRegistry{
		factories: make(map[string]HarnessFactory),
	}
	// Register default harnesses
	r.Register("claude", NewClaudeHarness)
	r.Register("gemini", NewGeminiHarness)
	r.Register("codex", NewCodexHarness)
	r.Register("mock", NewMockHarness)
	return r
}

// Register adds a harness factory to the registry.
func (r *HarnessRegistry) Register(name string, factory HarnessFactory) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.factories[name] = factory
}

// Get creates a harness instance using the registered factory.
// Returns an error if no factory is registered for the given name.
func (r *HarnessRegistry) Get(name string, config HarnessConfig) (Harness, error) {
	r.mu.RLock()
	factory, ok := r.factories[name]
	r.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("unknown harness: %s", name)
	}

	config.Name = name
	return factory(config), nil
}

// List returns the names of all registered harnesses.
func (r *HarnessRegistry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.factories))
	for name := range r.factories {
		names = append(names, name)
	}
	return names
}

// Has returns true if a harness with the given name is registered.
func (r *HarnessRegistry) Has(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.factories[name]
	return ok
}

// ClaudeHarness wraps the Claude CLI for agent execution.
type ClaudeHarness struct {
	config HarnessConfig
}

// NewClaudeHarness creates a new Claude harness.
func NewClaudeHarness(config HarnessConfig) Harness {
	if config.Command == "" {
		config.Command = "claude"
	}
	return &ClaudeHarness{config: config}
}

// Name returns "claude".
func (h *ClaudeHarness) Name() string {
	return "claude"
}

// Execute runs the Claude CLI with the given prompt.
// Uses --dangerously-skip-permissions for autonomous execution.
// Reference: ralph-orchestrator/crates/ralph-adapters/src/cli_backend.rs
func (h *ClaudeHarness) Execute(ctx context.Context, prompt string) (string, error) {
	args := []string{
		"--dangerously-skip-permissions",
		"--output-format", "text",
		"-p", prompt,
	}

	return h.runCommand(ctx, h.config.Command, args...)
}

// SupportsAutomation returns true for all automation modes.
func (h *ClaudeHarness) SupportsAutomation(mode AutomationMode) bool {
	return true
}

func (h *ClaudeHarness) runCommand(ctx context.Context, name string, args ...string) (string, error) {
	if h.config.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, h.config.Timeout)
		defer cancel()
	}

	cmd := exec.CommandContext(ctx, name, args...)

	if h.config.WorkDir != "" {
		cmd.Dir = h.config.WorkDir
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("%v: %s", err, stderr.String())
	}

	return stdout.String(), nil
}

// GeminiHarness wraps the Gemini API via Python for agent execution.
type GeminiHarness struct {
	config HarnessConfig
}

// NewGeminiHarness creates a new Gemini harness.
// Uses the gemini CLI (google's agentic coding tool) instead of raw API.
func NewGeminiHarness(config HarnessConfig) Harness {
	if config.Command == "" {
		config.Command = "gemini"
	}
	return &GeminiHarness{config: config}
}

// Name returns "gemini".
func (h *GeminiHarness) Name() string {
	return "gemini"
}

// Execute runs the Gemini CLI with the given prompt.
// Uses --yolo flag for autonomous execution (auto-approve all tool calls).
// Reference: ralph-orchestrator/crates/ralph-adapters/src/cli_backend.rs
func (h *GeminiHarness) Execute(ctx context.Context, prompt string) (string, error) {
	args := []string{
		"--yolo",
		"-p", prompt,
	}

	return h.runCommand(ctx, h.config.Command, args...)
}

// SupportsAutomation returns true for all automation modes.
// The gemini CLI supports autonomous execution via --yolo flag.
func (h *GeminiHarness) SupportsAutomation(mode AutomationMode) bool {
	return true
}

func (h *GeminiHarness) runCommand(ctx context.Context, name string, args ...string) (string, error) {
	if h.config.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, h.config.Timeout)
		defer cancel()
	}

	cmd := exec.CommandContext(ctx, name, args...)

	if h.config.WorkDir != "" {
		cmd.Dir = h.config.WorkDir
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("%v: %s", err, stderr.String())
	}

	return stdout.String(), nil
}

// CodexHarness wraps the Codex CLI for agent execution.
type CodexHarness struct {
	config HarnessConfig
}

// NewCodexHarness creates a new Codex harness.
func NewCodexHarness(config HarnessConfig) Harness {
	if config.Command == "" {
		config.Command = "codex"
	}
	return &CodexHarness{config: config}
}

// Name returns "codex".
func (h *CodexHarness) Name() string {
	return "codex"
}

// Execute runs the Codex CLI with the given prompt.
// Uses exec --full-auto for autonomous execution.
// Reference: ralph-orchestrator/crates/ralph-adapters/src/cli_backend.rs
func (h *CodexHarness) Execute(ctx context.Context, prompt string) (string, error) {
	args := []string{
		"exec",
		"--full-auto",
		prompt, // Codex uses positional argument, not -p flag
	}

	return h.runCommand(ctx, h.config.Command, args...)
}

// SupportsAutomation returns true for all automation modes.
func (h *CodexHarness) SupportsAutomation(mode AutomationMode) bool {
	return true
}

func (h *CodexHarness) runCommand(ctx context.Context, name string, args ...string) (string, error) {
	if h.config.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, h.config.Timeout)
		defer cancel()
	}

	cmd := exec.CommandContext(ctx, name, args...)

	if h.config.WorkDir != "" {
		cmd.Dir = h.config.WorkDir
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("%v: %s", err, stderr.String())
	}

	return stdout.String(), nil
}

// MockHarness is a harness for testing that returns configurable responses.
type MockHarness struct {
	config HarnessConfig
}

// NewMockHarness creates a new mock harness for testing.
func NewMockHarness(config HarnessConfig) Harness {
	return &MockHarness{config: config}
}

// Name returns "mock".
func (h *MockHarness) Name() string {
	return "mock"
}

// Execute returns the configured mock response or error.
func (h *MockHarness) Execute(ctx context.Context, prompt string) (string, error) {
	// Simulate execution delay if configured
	if h.config.MockDelay > 0 {
		select {
		case <-time.After(h.config.MockDelay):
		case <-ctx.Done():
			return "", ctx.Err()
		}
	}

	if h.config.MockError != nil {
		return "", h.config.MockError
	}

	return h.config.MockResponse, nil
}

// SupportsAutomation returns true for all automation modes.
func (h *MockHarness) SupportsAutomation(mode AutomationMode) bool {
	return true
}

// DefaultRegistry is the global harness registry with default harnesses.
var DefaultRegistry = NewHarnessRegistry()

// ExecuteHarness is a convenience function that executes a prompt using a harness from the default registry.
func ExecuteHarness(ctx context.Context, harnessName, prompt string, automationMode AutomationMode, workDir string) (HarnessResult, error) {
	start := time.Now()

	config := HarnessConfig{
		Name:           harnessName,
		WorkDir:        workDir,
		AutomationMode: automationMode,
	}

	harness, err := DefaultRegistry.Get(harnessName, config)
	if err != nil {
		return HarnessResult{
			Harness:   harnessName,
			Error:     err,
			Duration:  time.Since(start),
			Timestamp: start,
		}, err
	}

	if !harness.SupportsAutomation(automationMode) {
		err := fmt.Errorf("harness %s does not support automation mode %s", harnessName, automationMode)
		return HarnessResult{
			Harness:   harnessName,
			Error:     err,
			Duration:  time.Since(start),
			Timestamp: start,
		}, err
	}

	response, err := harness.Execute(ctx, prompt)
	return HarnessResult{
		Harness:   harnessName,
		Response:  response,
		Error:     err,
		Duration:  time.Since(start),
		Timestamp: start,
	}, err
}
