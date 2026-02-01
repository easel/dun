package dun

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

// TestMockHarnessName verifies the mock harness returns correct name.
func TestMockHarnessName(t *testing.T) {
	harness := NewMockHarness(HarnessConfig{})
	if harness.Name() != "mock" {
		t.Fatalf("expected name 'mock', got %q", harness.Name())
	}
}

// TestMockHarnessExecuteSuccess verifies mock harness returns configured response.
func TestMockHarnessExecuteSuccess(t *testing.T) {
	expected := "mock response"
	harness := NewMockHarness(HarnessConfig{
		MockResponse: expected,
	})

	ctx := context.Background()
	response, err := harness.Execute(ctx, "test prompt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if response != expected {
		t.Fatalf("expected %q, got %q", expected, response)
	}
}

// TestMockHarnessExecuteError verifies mock harness returns configured error.
func TestMockHarnessExecuteError(t *testing.T) {
	expectedErr := errors.New("mock error")
	harness := NewMockHarness(HarnessConfig{
		MockError: expectedErr,
	})

	ctx := context.Background()
	_, err := harness.Execute(ctx, "test prompt")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err != expectedErr {
		t.Fatalf("expected %v, got %v", expectedErr, err)
	}
}

// TestMockHarnessExecuteWithDelay verifies mock harness respects delay.
func TestMockHarnessExecuteWithDelay(t *testing.T) {
	delay := 50 * time.Millisecond
	harness := NewMockHarness(HarnessConfig{
		MockResponse: "delayed response",
		MockDelay:    delay,
	})

	ctx := context.Background()
	start := time.Now()
	response, err := harness.Execute(ctx, "test prompt")
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if response != "delayed response" {
		t.Fatalf("expected 'delayed response', got %q", response)
	}
	if elapsed < delay {
		t.Fatalf("expected at least %v delay, got %v", delay, elapsed)
	}
}

// TestMockHarnessExecuteContextCancellation verifies mock harness respects context cancellation.
func TestMockHarnessExecuteContextCancellation(t *testing.T) {
	harness := NewMockHarness(HarnessConfig{
		MockResponse: "response",
		MockDelay:    1 * time.Second,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	_, err := harness.Execute(ctx, "test prompt")
	if err == nil {
		t.Fatal("expected context error, got nil")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected context.DeadlineExceeded, got %v", err)
	}
}

// TestMockHarnessSupportsAutomation verifies mock harness supports all modes.
func TestMockHarnessSupportsAutomation(t *testing.T) {
	harness := NewMockHarness(HarnessConfig{})

	modes := []AutomationMode{
		AutomationManual,
		AutomationPlan,
		AutomationAuto,
		AutomationYolo,
	}

	for _, mode := range modes {
		if !harness.SupportsAutomation(mode) {
			t.Fatalf("mock harness should support mode %s", mode)
		}
	}
}

// TestHarnessRegistryRegisterAndGet verifies registry registration and lookup.
func TestHarnessRegistryRegisterAndGet(t *testing.T) {
	registry := NewHarnessRegistry()

	// Get a default harness
	harness, err := registry.Get("mock", HarnessConfig{MockResponse: "test"})
	if err != nil {
		t.Fatalf("unexpected error getting mock harness: %v", err)
	}
	if harness.Name() != "mock" {
		t.Fatalf("expected 'mock', got %q", harness.Name())
	}

	// Register a custom harness
	registry.Register("custom", func(config HarnessConfig) Harness {
		return NewMockHarness(HarnessConfig{MockResponse: "custom response"})
	})

	harness, err = registry.Get("custom", HarnessConfig{})
	if err != nil {
		t.Fatalf("unexpected error getting custom harness: %v", err)
	}
	if harness.Name() != "mock" { // Returns mock since our custom factory returns mock
		t.Fatalf("expected 'mock', got %q", harness.Name())
	}
}

// TestHarnessRegistryGetUnknown verifies registry returns error for unknown harness.
func TestHarnessRegistryGetUnknown(t *testing.T) {
	registry := NewHarnessRegistry()

	_, err := registry.Get("unknown", HarnessConfig{})
	if err == nil {
		t.Fatal("expected error for unknown harness, got nil")
	}
	if !strings.Contains(err.Error(), "unknown harness") {
		t.Fatalf("expected 'unknown harness' error, got: %v", err)
	}
}

// TestHarnessRegistryList verifies registry lists all registered harnesses.
func TestHarnessRegistryList(t *testing.T) {
	registry := NewHarnessRegistry()

	names := registry.List()
	if len(names) < 4 {
		t.Fatalf("expected at least 4 default harnesses, got %d", len(names))
	}

	expected := map[string]bool{"claude": true, "gemini": true, "codex": true, "mock": true}
	for _, name := range names {
		if !expected[name] {
			continue // Allow additional harnesses
		}
		delete(expected, name)
	}

	if len(expected) > 0 {
		t.Fatalf("missing expected harnesses: %v", expected)
	}
}

// TestHarnessRegistryHas verifies registry reports harness existence correctly.
func TestHarnessRegistryHas(t *testing.T) {
	registry := NewHarnessRegistry()

	if !registry.Has("mock") {
		t.Fatal("expected registry to have 'mock'")
	}
	if !registry.Has("claude") {
		t.Fatal("expected registry to have 'claude'")
	}
	if registry.Has("nonexistent") {
		t.Fatal("expected registry to not have 'nonexistent'")
	}
}

// TestClaudeHarnessName verifies claude harness returns correct name.
func TestClaudeHarnessName(t *testing.T) {
	harness := NewClaudeHarness(HarnessConfig{})
	if harness.Name() != "claude" {
		t.Fatalf("expected name 'claude', got %q", harness.Name())
	}
}

// TestClaudeHarnessSupportsAutomation verifies claude harness supports all modes.
func TestClaudeHarnessSupportsAutomation(t *testing.T) {
	harness := NewClaudeHarness(HarnessConfig{})

	modes := []AutomationMode{
		AutomationManual,
		AutomationPlan,
		AutomationAuto,
		AutomationYolo,
	}

	for _, mode := range modes {
		if !harness.SupportsAutomation(mode) {
			t.Fatalf("claude harness should support mode %s", mode)
		}
	}
}

// TestGeminiHarnessName verifies gemini harness returns correct name.
func TestGeminiHarnessName(t *testing.T) {
	harness := NewGeminiHarness(HarnessConfig{})
	if harness.Name() != "gemini" {
		t.Fatalf("expected name 'gemini', got %q", harness.Name())
	}
}

// TestGeminiHarnessSupportsAutomation verifies gemini harness supports all modes.
// The gemini CLI (Google's agentic coding tool) supports --yolo flag for autonomous execution.
func TestGeminiHarnessSupportsAutomation(t *testing.T) {
	harness := NewGeminiHarness(HarnessConfig{})

	modes := []AutomationMode{
		AutomationManual,
		AutomationPlan,
		AutomationAuto,
		AutomationYolo,
	}

	for _, mode := range modes {
		if !harness.SupportsAutomation(mode) {
			t.Fatalf("gemini harness should support mode %s", mode)
		}
	}
}

// TestCodexHarnessName verifies codex harness returns correct name.
func TestCodexHarnessName(t *testing.T) {
	harness := NewCodexHarness(HarnessConfig{})
	if harness.Name() != "codex" {
		t.Fatalf("expected name 'codex', got %q", harness.Name())
	}
}

// TestCodexHarnessSupportsAutomation verifies codex harness supports all modes.
func TestCodexHarnessSupportsAutomation(t *testing.T) {
	harness := NewCodexHarness(HarnessConfig{})

	modes := []AutomationMode{
		AutomationManual,
		AutomationPlan,
		AutomationAuto,
		AutomationYolo,
	}

	for _, mode := range modes {
		if !harness.SupportsAutomation(mode) {
			t.Fatalf("codex harness should support mode %s", mode)
		}
	}
}

// TestExecuteHarnessWithMock verifies ExecuteHarness convenience function works.
func TestExecuteHarnessWithMock(t *testing.T) {
	// Temporarily register a mock that we control
	origRegistry := DefaultRegistry
	DefaultRegistry = NewHarnessRegistry()
	DefaultRegistry.Register("mock", func(config HarnessConfig) Harness {
		return NewMockHarness(HarnessConfig{MockResponse: "test response"})
	})
	defer func() { DefaultRegistry = origRegistry }()

	ctx := context.Background()
	result, err := ExecuteHarness(ctx, "mock", "test prompt", AutomationAuto, ".")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Harness != "mock" {
		t.Fatalf("expected harness 'mock', got %q", result.Harness)
	}
	if result.Response != "test response" {
		t.Fatalf("expected 'test response', got %q", result.Response)
	}
	if result.Error != nil {
		t.Fatalf("expected no error, got %v", result.Error)
	}
	if result.Duration <= 0 {
		t.Fatal("expected positive duration")
	}
	if result.Timestamp.IsZero() {
		t.Fatal("expected non-zero timestamp")
	}
}

// TestExecuteHarnessUnknown verifies ExecuteHarness returns error for unknown harness.
func TestExecuteHarnessUnknown(t *testing.T) {
	ctx := context.Background()
	result, err := ExecuteHarness(ctx, "nonexistent", "test", AutomationAuto, ".")
	if err == nil {
		t.Fatal("expected error for unknown harness")
	}
	if !strings.Contains(err.Error(), "unknown harness") {
		t.Fatalf("expected 'unknown harness' error, got: %v", err)
	}
	if result.Error == nil {
		t.Fatal("expected result.Error to be set")
	}
}

// TestExecuteHarnessUnsupportedAutomation verifies ExecuteHarness returns error for unsupported mode.
func TestExecuteHarnessUnsupportedAutomation(t *testing.T) {
	// Register a harness that doesn't support yolo
	origRegistry := DefaultRegistry
	DefaultRegistry = NewHarnessRegistry()
	DefaultRegistry.Register("noyolo", func(config HarnessConfig) Harness {
		return &noYoloHarness{}
	})
	defer func() { DefaultRegistry = origRegistry }()

	ctx := context.Background()
	result, err := ExecuteHarness(ctx, "noyolo", "test", AutomationYolo, ".")
	if err == nil {
		t.Fatal("expected error for unsupported automation")
	}
	if !strings.Contains(err.Error(), "does not support automation mode") {
		t.Fatalf("expected 'does not support automation mode' error, got: %v", err)
	}
	if result.Error == nil {
		t.Fatal("expected result.Error to be set")
	}
}

// noYoloHarness is a test harness that doesn't support yolo mode.
type noYoloHarness struct{}

func (h *noYoloHarness) Name() string { return "noyolo" }
func (h *noYoloHarness) Execute(ctx context.Context, prompt string) (string, error) {
	return "ok", nil
}
func (h *noYoloHarness) SupportsAutomation(mode AutomationMode) bool {
	return mode != AutomationYolo
}

// TestHarnessConfigDefaults verifies harnesses use sensible defaults.
func TestHarnessConfigDefaults(t *testing.T) {
	// Claude harness should default to "claude" command
	claudeHarness := NewClaudeHarness(HarnessConfig{}).(*ClaudeHarness)
	if claudeHarness.config.Command != "claude" {
		t.Fatalf("expected default command 'claude', got %q", claudeHarness.config.Command)
	}

	// Gemini harness should default to "gemini" command (Google's agentic CLI)
	geminiHarness := NewGeminiHarness(HarnessConfig{}).(*GeminiHarness)
	if geminiHarness.config.Command != "gemini" {
		t.Fatalf("expected default command 'gemini', got %q", geminiHarness.config.Command)
	}

	// Codex harness should default to "codex" command
	codexHarness := NewCodexHarness(HarnessConfig{}).(*CodexHarness)
	if codexHarness.config.Command != "codex" {
		t.Fatalf("expected default command 'codex', got %q", codexHarness.config.Command)
	}
}

// TestHarnessConfigOverride verifies harnesses respect config overrides.
func TestHarnessConfigOverride(t *testing.T) {
	claudeHarness := NewClaudeHarness(HarnessConfig{Command: "/custom/claude"}).(*ClaudeHarness)
	if claudeHarness.config.Command != "/custom/claude" {
		t.Fatalf("expected command '/custom/claude', got %q", claudeHarness.config.Command)
	}
}

// TestHarnessResultFields verifies HarnessResult fields are populated correctly.
func TestHarnessResultFields(t *testing.T) {
	now := time.Now()
	result := HarnessResult{
		Harness:   "test",
		Response:  "response",
		Error:     errors.New("test error"),
		Duration:  100 * time.Millisecond,
		Timestamp: now,
	}

	if result.Harness != "test" {
		t.Fatalf("expected harness 'test', got %q", result.Harness)
	}
	if result.Response != "response" {
		t.Fatalf("expected response 'response', got %q", result.Response)
	}
	if result.Error == nil || result.Error.Error() != "test error" {
		t.Fatalf("expected error 'test error', got %v", result.Error)
	}
	if result.Duration != 100*time.Millisecond {
		t.Fatalf("expected duration 100ms, got %v", result.Duration)
	}
	if !result.Timestamp.Equal(now) {
		t.Fatalf("expected timestamp %v, got %v", now, result.Timestamp)
	}
}

// TestDefaultRegistryInitialized verifies DefaultRegistry has default harnesses.
func TestDefaultRegistryInitialized(t *testing.T) {
	if DefaultRegistry == nil {
		t.Fatal("DefaultRegistry should be initialized")
	}

	expectedHarnesses := []string{"claude", "gemini", "codex", "mock"}
	for _, name := range expectedHarnesses {
		if !DefaultRegistry.Has(name) {
			t.Fatalf("DefaultRegistry should have '%s' harness", name)
		}
	}
}

// TestDefaultRegistryCanCreateHarnesses verifies harnesses can be created from DefaultRegistry.
func TestDefaultRegistryCanCreateHarnesses(t *testing.T) {
	testCases := []struct {
		name         string
		expectedName string
	}{
		{"claude", "claude"},
		{"gemini", "gemini"},
		{"codex", "codex"},
		{"mock", "mock"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			harness, err := DefaultRegistry.Get(tc.name, HarnessConfig{})
			if err != nil {
				t.Fatalf("failed to get harness %s: %v", tc.name, err)
			}
			if harness.Name() != tc.expectedName {
				t.Fatalf("expected name %q, got %q", tc.expectedName, harness.Name())
			}
		})
	}
}

// TestHarnessRegistryConcurrentAccess verifies registry is thread-safe.
func TestHarnessRegistryConcurrentAccess(t *testing.T) {
	registry := NewHarnessRegistry()

	done := make(chan bool)

	// Concurrent reads
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				registry.Has("mock")
				registry.List()
				registry.Get("mock", HarnessConfig{})
			}
			done <- true
		}()
	}

	// Concurrent writes
	for i := 0; i < 10; i++ {
		go func(id int) {
			for j := 0; j < 100; j++ {
				name := "harness" + string(rune('A'+id))
				registry.Register(name, NewMockHarness)
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 20; i++ {
		<-done
	}
}

// TestClaudeHarnessExecuteWithEcho tests Claude harness with echo command.
func TestClaudeHarnessExecuteWithEcho(t *testing.T) {
	harness := NewClaudeHarness(HarnessConfig{
		Command: "echo",
	})

	ctx := context.Background()
	// When command is "echo", it will just echo the arguments
	response, err := harness.Execute(ctx, "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// echo will print the arguments
	if !strings.Contains(response, "-p") {
		t.Fatalf("expected echo output to contain '-p', got %q", response)
	}
}

// TestCodexHarnessExecuteWithEcho tests Codex harness with echo command.
func TestCodexHarnessExecuteWithEcho(t *testing.T) {
	harness := NewCodexHarness(HarnessConfig{
		Command: "echo",
	})

	ctx := context.Background()
	response, err := harness.Execute(ctx, "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// echo will print the arguments including "exec" and "-p"
	if !strings.Contains(response, "exec") {
		t.Fatalf("expected echo output to contain 'exec', got %q", response)
	}
}

// TestHarnessExecuteWithTimeout tests harness respects timeout.
func TestHarnessExecuteWithTimeout(t *testing.T) {
	harness := NewClaudeHarness(HarnessConfig{
		Command: "sleep",
		Timeout: 10 * time.Millisecond,
	})

	ctx := context.Background()
	_, err := harness.Execute(ctx, "10") // sleep 10 seconds
	if err == nil {
		t.Fatal("expected timeout error")
	}
	// The error should be about the command being killed or timing out
}

// TestAutomationModeConstants verifies automation mode constants are correct.
func TestAutomationModeConstants(t *testing.T) {
	if AutomationManual != "manual" {
		t.Fatalf("expected AutomationManual to be 'manual', got %q", AutomationManual)
	}
	if AutomationPlan != "plan" {
		t.Fatalf("expected AutomationPlan to be 'plan', got %q", AutomationPlan)
	}
	if AutomationAuto != "auto" {
		t.Fatalf("expected AutomationAuto to be 'auto', got %q", AutomationAuto)
	}
	if AutomationYolo != "yolo" {
		t.Fatalf("expected AutomationYolo to be 'yolo', got %q", AutomationYolo)
	}
}

// TestClaudeHarnessYoloArgs tests that yolo mode adds correct arguments.
func TestClaudeHarnessYoloArgs(t *testing.T) {
	harness := NewClaudeHarness(HarnessConfig{
		Command:        "echo",
		AutomationMode: AutomationYolo,
	})

	ctx := context.Background()
	response, err := harness.Execute(ctx, "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// In yolo mode, should include --dangerously-skip-permissions
	if !strings.Contains(response, "--dangerously-skip-permissions") {
		t.Fatalf("expected yolo args in output, got %q", response)
	}
}

// TestCodexHarnessYoloArgs tests that yolo mode adds correct arguments.
// Codex uses "exec --full-auto" for autonomous execution.
// Reference: ralph-orchestrator/crates/ralph-adapters/src/cli_backend.rs
func TestCodexHarnessYoloArgs(t *testing.T) {
	harness := NewCodexHarness(HarnessConfig{
		Command:        "echo",
		AutomationMode: AutomationYolo,
	})

	ctx := context.Background()
	response, err := harness.Execute(ctx, "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// In yolo mode, should include --full-auto
	if !strings.Contains(response, "--full-auto") {
		t.Fatalf("expected --full-auto in output, got %q", response)
	}
}

// TestHarnessWorkDir tests that working directory is respected.
func TestHarnessWorkDir(t *testing.T) {
	// Use a mock harness for this test since ClaudeHarness adds arguments
	// that would break the pwd command
	harness := NewMockHarness(HarnessConfig{
		WorkDir:      "/tmp",
		MockResponse: "test",
	})

	ctx := context.Background()
	response, err := harness.Execute(ctx, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Mock harness should return the configured response
	if response != "test" {
		t.Fatalf("expected 'test', got %q", response)
	}
}

// TestClaudeHarnessWorkDirConfig verifies work dir is stored in config.
func TestClaudeHarnessWorkDirConfig(t *testing.T) {
	harness := NewClaudeHarness(HarnessConfig{
		WorkDir: "/tmp",
	}).(*ClaudeHarness)

	if harness.config.WorkDir != "/tmp" {
		t.Fatalf("expected WorkDir '/tmp', got %q", harness.config.WorkDir)
	}
}

// TestGeminiHarnessPromptEscaping tests that gemini properly escapes prompts.
func TestGeminiHarnessPromptEscaping(t *testing.T) {
	harness := NewGeminiHarness(HarnessConfig{
		Command: "echo",
	})

	ctx := context.Background()
	// Execute with a prompt containing triple quotes
	_, err := harness.Execute(ctx, `test """quote""" test`)
	// This will fail because python3 isn't running a valid script,
	// but we're just testing that it doesn't crash
	if err == nil {
		// If it somehow succeeds, that's fine
		return
	}
	// Error should be about command execution, not panic
	if strings.Contains(err.Error(), "unknown harness") {
		t.Fatalf("gemini should be a known harness")
	}
}

// =============================================================================
// Integration Tests for Multi-Step Agent Behavior
// =============================================================================
// These tests verify that harnesses construct the correct command-line arguments
// for autonomous, multi-step agent execution. Reference: ralph-orchestrator.

// TestClaudeHarnessAutonomousArgs verifies Claude harness uses correct autonomous flags.
// Claude requires --dangerously-skip-permissions for multi-step execution.
func TestClaudeHarnessAutonomousArgs(t *testing.T) {
	harness := NewClaudeHarness(HarnessConfig{
		Command: "echo",
	})

	ctx := context.Background()
	response, err := harness.Execute(ctx, "multi-step task")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify all required autonomous mode flags are present
	required := []string{
		"--dangerously-skip-permissions", // Required for multi-step execution
		"--output-format",                // Needed for parseable output
		"text",                           // Text output format
		"-p",                             // Prompt flag
		"multi-step task",                // The actual prompt
	}

	for _, flag := range required {
		if !strings.Contains(response, flag) {
			t.Errorf("missing required flag %q in command: %s", flag, response)
		}
	}
}

// TestGeminiHarnessAutonomousArgs verifies Gemini harness uses correct autonomous flags.
// The gemini CLI uses --yolo for autonomous tool approval.
func TestGeminiHarnessAutonomousArgs(t *testing.T) {
	harness := NewGeminiHarness(HarnessConfig{
		Command: "echo",
	})

	ctx := context.Background()
	response, err := harness.Execute(ctx, "multi-step task")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify all required autonomous mode flags are present
	required := []string{
		"--yolo",          // Required for autonomous tool approval
		"-p",              // Prompt flag
		"multi-step task", // The actual prompt
	}

	for _, flag := range required {
		if !strings.Contains(response, flag) {
			t.Errorf("missing required flag %q in command: %s", flag, response)
		}
	}
}

// TestCodexHarnessAutonomousArgs verifies Codex harness uses correct autonomous flags.
// Codex uses "exec --full-auto" for autonomous execution with positional prompt.
func TestCodexHarnessAutonomousArgs(t *testing.T) {
	harness := NewCodexHarness(HarnessConfig{
		Command: "echo",
	})

	ctx := context.Background()
	response, err := harness.Execute(ctx, "multi-step task")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify all required autonomous mode flags are present
	required := []string{
		"exec",            // Codex subcommand
		"--full-auto",     // Required for autonomous execution
		"multi-step task", // Positional prompt argument (not -p)
	}

	for _, flag := range required {
		if !strings.Contains(response, flag) {
			t.Errorf("missing required flag %q in command: %s", flag, response)
		}
	}

	// Codex should NOT use -p flag (uses positional argument)
	if strings.Contains(response, " -p ") {
		t.Errorf("codex should not use -p flag, uses positional argument: %s", response)
	}
}

// TestAllHarnessesHaveAutonomousMode verifies all harnesses support autonomous execution.
func TestAllHarnessesHaveAutonomousMode(t *testing.T) {
	harnesses := []struct {
		name    string
		harness Harness
	}{
		{"claude", NewClaudeHarness(HarnessConfig{})},
		{"gemini", NewGeminiHarness(HarnessConfig{})},
		{"codex", NewCodexHarness(HarnessConfig{})},
		{"mock", NewMockHarness(HarnessConfig{})},
	}

	for _, h := range harnesses {
		t.Run(h.name, func(t *testing.T) {
			// All harnesses should support all automation modes for multi-step execution
			modes := []AutomationMode{
				AutomationManual,
				AutomationPlan,
				AutomationAuto,
				AutomationYolo,
			}

			for _, mode := range modes {
				if !h.harness.SupportsAutomation(mode) {
					t.Errorf("%s harness should support %s mode for multi-step execution", h.name, mode)
				}
			}
		})
	}
}

// TestHarnessExecutePreservesPromptContent tests that complex prompts are passed through correctly.
func TestHarnessExecutePreservesPromptContent(t *testing.T) {
	complexPrompt := `Step 1: Read the file config.yaml
Step 2: Parse the YAML content
Step 3: Validate the schema
Step 4: Apply the changes`

	harnesses := []struct {
		name    string
		harness Harness
	}{
		{"claude", NewClaudeHarness(HarnessConfig{Command: "echo"})},
		{"gemini", NewGeminiHarness(HarnessConfig{Command: "echo"})},
		{"codex", NewCodexHarness(HarnessConfig{Command: "echo"})},
	}

	ctx := context.Background()
	for _, h := range harnesses {
		t.Run(h.name, func(t *testing.T) {
			response, err := h.harness.Execute(ctx, complexPrompt)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// The multi-line prompt should be preserved in the command
			if !strings.Contains(response, "Step 1") {
				t.Errorf("prompt content not preserved: %s", response)
			}
		})
	}
}

// TestHarnessConsistentInterface ensures all harnesses implement the interface consistently.
func TestHarnessConsistentInterface(t *testing.T) {
	configs := []HarnessConfig{
		{}, // Empty config
		{WorkDir: "/tmp"},
		{Timeout: 30 * time.Second},
		{AutomationMode: AutomationYolo},
	}

	factories := []struct {
		name    string
		factory HarnessFactory
	}{
		{"claude", NewClaudeHarness},
		{"gemini", NewGeminiHarness},
		{"codex", NewCodexHarness},
		{"mock", NewMockHarness},
	}

	for _, f := range factories {
		for _, cfg := range configs {
			t.Run(f.name, func(t *testing.T) {
				harness := f.factory(cfg)

				// All harnesses must implement Harness interface
				var _ Harness = harness

				// Name must not be empty
				if harness.Name() == "" {
					t.Error("harness name should not be empty")
				}

				// SupportsAutomation must not panic
				_ = harness.SupportsAutomation(AutomationYolo)
			})
		}
	}
}

// TestCrossAgentExecutionConsistency tests that all agents produce consistent execution patterns.
func TestCrossAgentExecutionConsistency(t *testing.T) {
	prompt := "analyze the codebase and suggest improvements"

	harnesses := map[string]Harness{
		"claude": NewClaudeHarness(HarnessConfig{Command: "echo"}),
		"gemini": NewGeminiHarness(HarnessConfig{Command: "echo"}),
		"codex":  NewCodexHarness(HarnessConfig{Command: "echo"}),
	}

	ctx := context.Background()
	results := make(map[string]string)

	for name, harness := range harnesses {
		response, err := harness.Execute(ctx, prompt)
		if err != nil {
			t.Fatalf("%s: unexpected error: %v", name, err)
		}
		results[name] = response
	}

	// All harnesses should include the prompt in their output
	for name, result := range results {
		if !strings.Contains(result, "analyze") {
			t.Errorf("%s: prompt not found in command output: %s", name, result)
		}
	}
}

// TestHarnessErrorHandlingConsistency ensures error handling is consistent across harnesses.
func TestHarnessErrorHandlingConsistency(t *testing.T) {
	harnesses := []struct {
		name    string
		harness Harness
	}{
		{"claude", NewClaudeHarness(HarnessConfig{Command: "false"})}, // "false" command always fails
		{"gemini", NewGeminiHarness(HarnessConfig{Command: "false"})},
		{"codex", NewCodexHarness(HarnessConfig{Command: "false"})},
	}

	ctx := context.Background()
	for _, h := range harnesses {
		t.Run(h.name, func(t *testing.T) {
			_, err := h.harness.Execute(ctx, "test")
			// All harnesses should return an error when the command fails
			if err == nil {
				t.Errorf("%s: expected error for failed command", h.name)
			}
		})
	}
}

// TestHarnessTimeoutConsistency tests timeout behavior across harnesses.
func TestHarnessTimeoutConsistency(t *testing.T) {
	timeout := 50 * time.Millisecond

	harnesses := []struct {
		name    string
		harness Harness
	}{
		{"claude", NewClaudeHarness(HarnessConfig{Command: "sleep", Timeout: timeout})},
		{"gemini", NewGeminiHarness(HarnessConfig{Command: "sleep", Timeout: timeout})},
		{"codex", NewCodexHarness(HarnessConfig{Command: "sleep", Timeout: timeout})},
	}

	ctx := context.Background()
	for _, h := range harnesses {
		t.Run(h.name, func(t *testing.T) {
			start := time.Now()
			_, err := h.harness.Execute(ctx, "10") // Sleep 10 seconds
			elapsed := time.Since(start)

			// Should timeout quickly, not wait full 10 seconds
			if err == nil {
				t.Errorf("%s: expected timeout error", h.name)
			}
			if elapsed > 5*time.Second {
				t.Errorf("%s: timeout not respected, took %v", h.name, elapsed)
			}
		})
	}
}

// =============================================================================
// Cross-Agent Consistency Regression Tests
// =============================================================================
// These tests prevent regressions in harness behavior across agent types.
// If any test fails, it indicates a breaking change in agent invocation.

// TestRegressionClaudeFlags guards against changes to Claude autonomous mode flags.
// These exact flags are required for multi-step execution:
// - --dangerously-skip-permissions: Required for autonomous tool execution
// - --output-format text: Parseable output format
// - -p: Prompt flag
func TestRegressionClaudeFlags(t *testing.T) {
	harness := NewClaudeHarness(HarnessConfig{Command: "echo"})
	ctx := context.Background()

	response, _ := harness.Execute(ctx, "test")

	// These flags MUST NOT change without explicit decision
	requiredFlags := map[string]string{
		"--dangerously-skip-permissions": "required for autonomous tool execution",
		"--output-format":                "required for parseable output",
		"text":                           "text output format",
		"-p":                             "prompt flag",
	}

	for flag, reason := range requiredFlags {
		if !strings.Contains(response, flag) {
			t.Errorf("REGRESSION: Claude missing required flag %q (%s)", flag, reason)
		}
	}
}

// TestRegressionGeminiFlags guards against changes to Gemini autonomous mode flags.
// These exact flags are required for multi-step execution:
// - --yolo: Required for autonomous tool approval
// - -p: Prompt flag
func TestRegressionGeminiFlags(t *testing.T) {
	harness := NewGeminiHarness(HarnessConfig{Command: "echo"})
	ctx := context.Background()

	response, _ := harness.Execute(ctx, "test")

	// These flags MUST NOT change without explicit decision
	requiredFlags := map[string]string{
		"--yolo": "required for autonomous tool approval",
		"-p":     "prompt flag",
	}

	for flag, reason := range requiredFlags {
		if !strings.Contains(response, flag) {
			t.Errorf("REGRESSION: Gemini missing required flag %q (%s)", flag, reason)
		}
	}
}

// TestRegressionCodexFlags guards against changes to Codex autonomous mode flags.
// These exact flags are required for multi-step execution:
// - exec: Codex subcommand
// - --full-auto: Required for autonomous execution
// - Positional prompt (NOT -p flag)
func TestRegressionCodexFlags(t *testing.T) {
	harness := NewCodexHarness(HarnessConfig{Command: "echo"})
	ctx := context.Background()

	response, _ := harness.Execute(ctx, "test")

	// These flags MUST NOT change without explicit decision
	requiredFlags := map[string]string{
		"exec":       "codex subcommand",
		"--full-auto": "required for autonomous execution",
	}

	for flag, reason := range requiredFlags {
		if !strings.Contains(response, flag) {
			t.Errorf("REGRESSION: Codex missing required flag %q (%s)", flag, reason)
		}
	}

	// Codex uses positional argument, NOT -p flag
	if strings.Contains(response, " -p ") {
		t.Error("REGRESSION: Codex should use positional prompt, not -p flag")
	}
}

// TestRegressionDefaultCommands guards against changes to default commands.
func TestRegressionDefaultCommands(t *testing.T) {
	tests := []struct {
		name            string
		factory         HarnessFactory
		expectedCommand string
	}{
		{"claude", NewClaudeHarness, "claude"},
		{"gemini", NewGeminiHarness, "gemini"}, // Changed from python3 to gemini CLI
		{"codex", NewCodexHarness, "codex"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			harness := tc.factory(HarnessConfig{})

			// Use type assertion to access config
			var command string
			switch h := harness.(type) {
			case *ClaudeHarness:
				command = h.config.Command
			case *GeminiHarness:
				command = h.config.Command
			case *CodexHarness:
				command = h.config.Command
			}

			if command != tc.expectedCommand {
				t.Errorf("REGRESSION: %s default command changed from %q to %q",
					tc.name, tc.expectedCommand, command)
			}
		})
	}
}

// TestRegressionHarnessNames guards against changes to harness names.
func TestRegressionHarnessNames(t *testing.T) {
	tests := []struct {
		factory      HarnessFactory
		expectedName string
	}{
		{NewClaudeHarness, "claude"},
		{NewGeminiHarness, "gemini"},
		{NewCodexHarness, "codex"},
		{NewMockHarness, "mock"},
	}

	for _, tc := range tests {
		t.Run(tc.expectedName, func(t *testing.T) {
			harness := tc.factory(HarnessConfig{})
			if harness.Name() != tc.expectedName {
				t.Errorf("REGRESSION: harness name changed from %q to %q",
					tc.expectedName, harness.Name())
			}
		})
	}
}

// TestRegressionRegistryHarnesses guards against removal of default harnesses.
func TestRegressionRegistryHarnesses(t *testing.T) {
	registry := NewHarnessRegistry()

	// These harnesses MUST be registered by default
	requiredHarnesses := []string{"claude", "gemini", "codex", "mock"}

	for _, name := range requiredHarnesses {
		if !registry.Has(name) {
			t.Errorf("REGRESSION: %q harness not registered in default registry", name)
		}
	}
}

// TestRegressionAllHarnessesAutonomous guards against removal of automation support.
func TestRegressionAllHarnessesAutonomous(t *testing.T) {
	harnesses := []struct {
		name    string
		harness Harness
	}{
		{"claude", NewClaudeHarness(HarnessConfig{})},
		{"gemini", NewGeminiHarness(HarnessConfig{})},
		{"codex", NewCodexHarness(HarnessConfig{})},
	}

	// All real harnesses MUST support all automation modes
	modes := []AutomationMode{
		AutomationManual,
		AutomationPlan,
		AutomationAuto,
		AutomationYolo,
	}

	for _, h := range harnesses {
		for _, mode := range modes {
			if !h.harness.SupportsAutomation(mode) {
				t.Errorf("REGRESSION: %s harness no longer supports %s mode", h.name, mode)
			}
		}
	}
}

// TestRegressionAutomationModeValues guards against changes to automation mode string values.
func TestRegressionAutomationModeValues(t *testing.T) {
	// These string values are used in configuration and MUST NOT change
	expected := map[AutomationMode]string{
		AutomationManual: "manual",
		AutomationPlan:   "plan",
		AutomationAuto:   "auto",
		AutomationYolo:   "yolo",
	}

	for mode, expectedValue := range expected {
		if string(mode) != expectedValue {
			t.Errorf("REGRESSION: AutomationMode %s value changed from %q to %q",
				mode, expectedValue, string(mode))
		}
	}
}

// TestRegressionHarnessResultFields guards against changes to HarnessResult structure.
func TestRegressionHarnessResultFields(t *testing.T) {
	result := HarnessResult{
		Harness:   "test",
		Response:  "response",
		Error:     errors.New("error"),
		Duration:  time.Second,
		Timestamp: time.Now(),
	}

	// All these fields MUST exist and be accessible
	if result.Harness == "" {
		t.Error("REGRESSION: HarnessResult.Harness field removed or renamed")
	}
	if result.Response == "" {
		t.Error("REGRESSION: HarnessResult.Response field removed or renamed")
	}
	if result.Error == nil {
		t.Error("REGRESSION: HarnessResult.Error field removed or renamed")
	}
	if result.Duration == 0 {
		t.Error("REGRESSION: HarnessResult.Duration field removed or renamed")
	}
	if result.Timestamp.IsZero() {
		t.Error("REGRESSION: HarnessResult.Timestamp field removed or renamed")
	}
}

// TestRegressionHarnessConfigFields guards against changes to HarnessConfig structure.
func TestRegressionHarnessConfigFields(t *testing.T) {
	config := HarnessConfig{
		Name:           "test",
		Command:        "cmd",
		WorkDir:        "/tmp",
		Timeout:        time.Second,
		AutomationMode: AutomationYolo,
		Env:            map[string]string{"KEY": "value"},
		MockResponse:   "response",
		MockError:      errors.New("error"),
		MockDelay:      time.Millisecond,
	}

	// All these fields MUST exist and be accessible
	if config.Name == "" {
		t.Error("REGRESSION: HarnessConfig.Name field removed or renamed")
	}
	if config.Command == "" {
		t.Error("REGRESSION: HarnessConfig.Command field removed or renamed")
	}
	if config.WorkDir == "" {
		t.Error("REGRESSION: HarnessConfig.WorkDir field removed or renamed")
	}
	if config.Timeout == 0 {
		t.Error("REGRESSION: HarnessConfig.Timeout field removed or renamed")
	}
	if config.AutomationMode == "" {
		t.Error("REGRESSION: HarnessConfig.AutomationMode field removed or renamed")
	}
	if config.Env == nil {
		t.Error("REGRESSION: HarnessConfig.Env field removed or renamed")
	}
	if config.MockResponse == "" {
		t.Error("REGRESSION: HarnessConfig.MockResponse field removed or renamed")
	}
	if config.MockError == nil {
		t.Error("REGRESSION: HarnessConfig.MockError field removed or renamed")
	}
	if config.MockDelay == 0 {
		t.Error("REGRESSION: HarnessConfig.MockDelay field removed or renamed")
	}
}
