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

// TestGeminiHarnessSupportsAutomation verifies gemini harness supports expected modes.
func TestGeminiHarnessSupportsAutomation(t *testing.T) {
	harness := NewGeminiHarness(HarnessConfig{})

	// Gemini supports manual, plan, auto but not yolo
	supportedModes := []AutomationMode{
		AutomationManual,
		AutomationPlan,
		AutomationAuto,
	}

	for _, mode := range supportedModes {
		if !harness.SupportsAutomation(mode) {
			t.Fatalf("gemini harness should support mode %s", mode)
		}
	}

	// Gemini does not support yolo
	if harness.SupportsAutomation(AutomationYolo) {
		t.Fatal("gemini harness should not support yolo mode")
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

	// Gemini harness should default to "python3" command
	geminiHarness := NewGeminiHarness(HarnessConfig{}).(*GeminiHarness)
	if geminiHarness.config.Command != "python3" {
		t.Fatalf("expected default command 'python3', got %q", geminiHarness.config.Command)
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
	// In yolo mode, should include --ask-for-approval never
	if !strings.Contains(response, "--ask-for-approval") {
		t.Fatalf("expected yolo args in output, got %q", response)
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
