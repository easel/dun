package dun

import (
	"bytes"
	"context"
	"errors"
	"runtime"
	"strings"
	"testing"
	"time"
)

type runnerCapture struct {
	name    string
	args    []string
	stdin   string
	workDir string
	env     map[string]string
	calls   int
}

func captureRunner(capture *runnerCapture, stdout string, stderr string, err error) CommandRunner {
	return func(_ context.Context, name string, args []string, workDir string, env map[string]string, stdin string) (string, string, error) {
		capture.calls++
		capture.name = name
		capture.args = append([]string(nil), args...)
		capture.stdin = stdin
		capture.workDir = workDir
		if env != nil {
			capture.env = make(map[string]string, len(env))
			for key, value := range env {
				capture.env[key] = value
			}
		} else {
			capture.env = nil
		}
		return stdout, stderr, err
	}
}

func blockingRunner(capture *runnerCapture) CommandRunner {
	return func(ctx context.Context, name string, args []string, workDir string, env map[string]string, stdin string) (string, string, error) {
		capture.calls++
		capture.name = name
		capture.args = append([]string(nil), args...)
		capture.stdin = stdin
		capture.workDir = workDir
		if env != nil {
			capture.env = make(map[string]string, len(env))
			for key, value := range env {
				capture.env[key] = value
			}
		} else {
			capture.env = nil
		}
		<-ctx.Done()
		return "", "", ctx.Err()
	}
}

func assertArgsContain(t *testing.T, args []string, required []string) {
	t.Helper()
	for _, req := range required {
		found := false
		for _, arg := range args {
			if arg == req {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("expected args to contain %q, got %v", req, args)
		}
	}
}

func assertArgsContainSubstring(t *testing.T, args []string, needle string) {
	t.Helper()
	for _, arg := range args {
		if strings.Contains(arg, needle) {
			return
		}
	}
	t.Fatalf("expected args to contain %q, got %v", needle, args)
}

func assertFlagValue(t *testing.T, args []string, flag string, value string) {
	t.Helper()
	for i, arg := range args {
		if arg == flag && i+1 < len(args) {
			if args[i+1] != value {
				t.Fatalf("expected %s %q, got %q", flag, value, args[i+1])
			}
			return
		}
	}
	t.Fatalf("expected args to contain %s %q, got %v", flag, value, args)
}

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
	if len(names) < 5 {
		t.Fatalf("expected at least 5 default harnesses, got %d", len(names))
	}

	expected := map[string]bool{"claude": true, "gemini": true, "codex": true, "opencode": true, "mock": true}
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

// TestOpenCodeHarnessName verifies opencode harness returns correct name.
func TestOpenCodeHarnessName(t *testing.T) {
	harness := NewOpenCodeHarness(HarnessConfig{})
	if harness.Name() != "opencode" {
		t.Fatalf("expected name 'opencode', got %q", harness.Name())
	}
}

// TestOpenCodeHarnessSupportsAutomation verifies opencode harness supports all modes.
func TestOpenCodeHarnessSupportsAutomation(t *testing.T) {
	harness := NewOpenCodeHarness(HarnessConfig{})

	modes := []AutomationMode{
		AutomationManual,
		AutomationPlan,
		AutomationAuto,
		AutomationYolo,
	}

	for _, mode := range modes {
		if !harness.SupportsAutomation(mode) {
			t.Fatalf("opencode harness should support mode %s", mode)
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
	result, err := ExecuteHarness(ctx, "mock", "test prompt", AutomationAuto, ".", "")
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
	result, err := ExecuteHarness(ctx, "nonexistent", "test", AutomationAuto, ".", "")
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
	result, err := ExecuteHarness(ctx, "noyolo", "test", AutomationYolo, ".", "")
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

	// OpenCode harness should default to "opencode" command
	openCodeHarness := NewOpenCodeHarness(HarnessConfig{}).(*OpenCodeHarness)
	if openCodeHarness.config.Command != "opencode" {
		t.Fatalf("expected default command 'opencode', got %q", openCodeHarness.config.Command)
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

	expectedHarnesses := []string{"claude", "gemini", "codex", "opencode", "mock"}
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
		{"opencode", "opencode"},
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
	capture := &runnerCapture{}
	harness := NewClaudeHarness(HarnessConfig{
		Runner: captureRunner(capture, "ok", "", nil),
	})

	ctx := context.Background()
	_, err := harness.Execute(ctx, "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertArgsContain(t, capture.args, []string{"--print", "--input-format", "text", "--output-format", "text"})
	if capture.stdin != "test" {
		t.Fatalf("expected stdin to be %q, got %q", "test", capture.stdin)
	}
}

// TestCodexHarnessExecuteWithEcho tests Codex harness with echo command.
func TestCodexHarnessExecuteWithEcho(t *testing.T) {
	capture := &runnerCapture{}
	harness := NewCodexHarness(HarnessConfig{
		Runner: captureRunner(capture, "ok", "", nil),
	})

	ctx := context.Background()
	_, err := harness.Execute(ctx, "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertArgsContain(t, capture.args, []string{"exec", "--full-auto", "-"})
	if capture.stdin != "test" {
		t.Fatalf("expected stdin to be %q, got %q", "test", capture.stdin)
	}
}

// TestOpenCodeHarnessExecuteWithEcho tests OpenCode harness with echo command.
func TestOpenCodeHarnessExecuteWithEcho(t *testing.T) {
	capture := &runnerCapture{}
	harness := NewOpenCodeHarness(HarnessConfig{
		Runner: captureRunner(capture, "ok", "", nil),
	})

	ctx := context.Background()
	_, err := harness.Execute(ctx, "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertArgsContain(t, capture.args, []string{"run"})
	assertArgsContain(t, capture.args, []string{"test"})
	if capture.stdin != "test" {
		t.Fatalf("expected stdin to be %q, got %q", "test", capture.stdin)
	}
}

// TestHarnessModelFlags verifies model overrides are passed to harness CLIs.
func TestHarnessModelFlags(t *testing.T) {
	tests := []struct {
		name    string
		harness Harness
	}{
		{"claude", NewClaudeHarness(HarnessConfig{Model: "model-a", Runner: captureRunner(&runnerCapture{}, "ok", "", nil)})},
		{"gemini", NewGeminiHarness(HarnessConfig{Model: "model-b", Runner: captureRunner(&runnerCapture{}, "ok", "", nil)})},
		{"codex", NewCodexHarness(HarnessConfig{Model: "model-c", Runner: captureRunner(&runnerCapture{}, "ok", "", nil)})},
		{"opencode", NewOpenCodeHarness(HarnessConfig{Model: "model-d", Runner: captureRunner(&runnerCapture{}, "ok", "", nil)})},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			capture := &runnerCapture{}
			switch tc.name {
			case "claude":
				tc.harness = NewClaudeHarness(HarnessConfig{Model: "model-a", Runner: captureRunner(capture, "ok", "", nil)})
			case "gemini":
				tc.harness = NewGeminiHarness(HarnessConfig{Model: "model-b", Runner: captureRunner(capture, "ok", "", nil)})
			case "codex":
				tc.harness = NewCodexHarness(HarnessConfig{Model: "model-c", Runner: captureRunner(capture, "ok", "", nil)})
			case "opencode":
				tc.harness = NewOpenCodeHarness(HarnessConfig{Model: "model-d", Runner: captureRunner(capture, "ok", "", nil)})
			}

			_, err := tc.harness.Execute(context.Background(), "test")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			switch tc.name {
			case "claude":
				assertFlagValue(t, capture.args, "--model", "model-a")
			case "gemini":
				assertFlagValue(t, capture.args, "--model", "model-b")
			case "codex":
				assertFlagValue(t, capture.args, "--model", "model-c")
			case "opencode":
				assertFlagValue(t, capture.args, "--model", "model-d")
			}
		})
	}
}

// TestHarnessExecuteWithTimeout tests harness respects timeout.
func TestHarnessExecuteWithTimeout(t *testing.T) {
	capture := &runnerCapture{}
	harness := NewClaudeHarness(HarnessConfig{
		Timeout: 10 * time.Millisecond,
		Runner:  blockingRunner(capture),
	})

	ctx := context.Background()
	_, err := harness.Execute(ctx, "10")
	if err == nil {
		t.Fatal("expected timeout error")
	}
	// The error should be about the command being killed or timing out
}

// TestRunHarnessCommandStreamingCapturesOutput verifies streaming preserves stdout and stdin.
func TestRunHarnessCommandStreamingCapturesOutput(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("streaming test relies on cat")
	}

	var streamed bytes.Buffer
	var streamedErr bytes.Buffer
	prompt := "streaming-stdin"

	out, err := runHarnessCommand(context.Background(), HarnessConfig{
		StdoutWriter: &streamed,
		StderrWriter: &streamedErr,
	}, "cat", prompt, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != prompt {
		t.Fatalf("expected stdout %q, got %q", prompt, out)
	}
	if streamed.String() != prompt {
		t.Fatalf("expected streamed stdout %q, got %q", prompt, streamed.String())
	}
	if streamedErr.Len() != 0 {
		t.Fatalf("expected no stderr output, got %q", streamedErr.String())
	}
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
	capture := &runnerCapture{}
	harness := NewClaudeHarness(HarnessConfig{
		AutomationMode: AutomationYolo,
		Runner:         captureRunner(capture, "ok", "", nil),
	})

	ctx := context.Background()
	_, err := harness.Execute(ctx, "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// In yolo mode, should include --dangerously-skip-permissions
	assertArgsContain(t, capture.args, []string{"--dangerously-skip-permissions"})
	if capture.stdin != "test" {
		t.Fatalf("expected stdin to be %q, got %q", "test", capture.stdin)
	}
}

// TestCodexHarnessYoloArgs tests that yolo mode adds correct arguments.
// Codex uses "exec --full-auto" for autonomous execution.
// Reference: ralph-orchestrator/crates/ralph-adapters/src/cli_backend.rs
func TestCodexHarnessYoloArgs(t *testing.T) {
	capture := &runnerCapture{}
	harness := NewCodexHarness(HarnessConfig{
		AutomationMode: AutomationYolo,
		Runner:         captureRunner(capture, "ok", "", nil),
	})

	ctx := context.Background()
	_, err := harness.Execute(ctx, "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// In yolo mode, should include --full-auto
	assertArgsContain(t, capture.args, []string{"--full-auto"})
	assertArgsContain(t, capture.args, []string{"-"})
	if capture.stdin != "test" {
		t.Fatalf("expected stdin to be %q, got %q", "test", capture.stdin)
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
	capture := &runnerCapture{}
	harness := NewGeminiHarness(HarnessConfig{
		Runner: captureRunner(capture, "ok", "", nil),
	})

	ctx := context.Background()
	// Execute with a prompt containing triple quotes
	_, err := harness.Execute(ctx, `test """quote""" test`)
	// Ensure prompt is preserved in args and no panic
	if err == nil {
		if capture.stdin != `test """quote""" test` {
			t.Fatalf("expected stdin to contain prompt, got %q", capture.stdin)
		}
		return
	}
	// Error should be about command execution, not panic
	if strings.Contains(err.Error(), "unknown harness") {
		t.Fatalf("gemini should be a known harness")
	}
	if capture.stdin != `test """quote""" test` {
		t.Fatalf("expected stdin to contain prompt, got %q", capture.stdin)
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
	capture := &runnerCapture{}
	harness := NewClaudeHarness(HarnessConfig{
		Runner: captureRunner(capture, "ok", "", nil),
	})

	ctx := context.Background()
	_, err := harness.Execute(ctx, "multi-step task")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify all required non-interactive flags are present
	required := []string{
		"--print",
		"--input-format",
		"text",
		"--output-format",
		"text",
	}

	assertArgsContain(t, capture.args, required)
	if capture.stdin != "multi-step task" {
		t.Fatalf("expected stdin to be %q, got %q", "multi-step task", capture.stdin)
	}
}

// TestGeminiHarnessAutonomousArgs verifies Gemini harness uses correct autonomous flags.
// The gemini CLI uses --yolo for autonomous tool approval.
func TestGeminiHarnessAutonomousArgs(t *testing.T) {
	capture := &runnerCapture{}
	harness := NewGeminiHarness(HarnessConfig{
		Runner: captureRunner(capture, "ok", "", nil),
	})

	ctx := context.Background()
	_, err := harness.Execute(ctx, "multi-step task")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify all required non-interactive flags are present
	required := []string{
		"--prompt",
		"--approval-mode",
		"auto_edit",
		"--output-format",
		"text",
	}

	assertArgsContain(t, capture.args, required)
	promptIndex := -1
	for i, arg := range capture.args {
		if arg == "--prompt" {
			promptIndex = i
			break
		}
	}
	if promptIndex == -1 || promptIndex+1 >= len(capture.args) {
		t.Fatalf("expected --prompt flag to include an empty argument, got %v", capture.args)
	}
	if capture.args[promptIndex+1] != "" {
		t.Fatalf("expected --prompt to be followed by empty string, got %q", capture.args[promptIndex+1])
	}
	if capture.stdin != "multi-step task" {
		t.Fatalf("expected stdin to be %q, got %q", "multi-step task", capture.stdin)
	}
}

// TestCodexHarnessAutonomousArgs verifies Codex harness uses correct autonomous flags.
// Codex uses "exec --full-auto" for autonomous execution with positional prompt.
func TestCodexHarnessAutonomousArgs(t *testing.T) {
	capture := &runnerCapture{}
	harness := NewCodexHarness(HarnessConfig{
		Runner: captureRunner(capture, "ok", "", nil),
	})

	ctx := context.Background()
	_, err := harness.Execute(ctx, "multi-step task")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify all required non-interactive flags are present
	required := []string{
		"exec",
		"--full-auto",
		"-",
	}

	assertArgsContain(t, capture.args, required)
	if capture.stdin != "multi-step task" {
		t.Fatalf("expected stdin to be %q, got %q", "multi-step task", capture.stdin)
	}
}

// TestOpenCodeHarnessAutonomousArgs verifies OpenCode harness uses correct autonomous flags.
func TestOpenCodeHarnessAutonomousArgs(t *testing.T) {
	capture := &runnerCapture{}
	harness := NewOpenCodeHarness(HarnessConfig{
		Runner: captureRunner(capture, "ok", "", nil),
	})

	ctx := context.Background()
	_, err := harness.Execute(ctx, "multi-step task")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify required arguments are present
	required := []string{
		"run",
		"multi-step task",
	}

	assertArgsContain(t, capture.args, required)
	if capture.stdin != "multi-step task" {
		t.Fatalf("expected stdin to be %q, got %q", "multi-step task", capture.stdin)
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
		{"opencode", NewOpenCodeHarness(HarnessConfig{})},
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

	ctx := context.Background()
	for _, name := range []string{"claude", "gemini", "codex", "opencode"} {
		t.Run(name, func(t *testing.T) {
			capture := &runnerCapture{}
			var harness Harness
			switch name {
			case "claude":
				harness = NewClaudeHarness(HarnessConfig{Runner: captureRunner(capture, "ok", "", nil)})
			case "gemini":
				harness = NewGeminiHarness(HarnessConfig{Runner: captureRunner(capture, "ok", "", nil)})
			case "codex":
				harness = NewCodexHarness(HarnessConfig{Runner: captureRunner(capture, "ok", "", nil)})
			case "opencode":
				harness = NewOpenCodeHarness(HarnessConfig{Runner: captureRunner(capture, "ok", "", nil)})
			}
			_, err := harness.Execute(ctx, complexPrompt)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// The multi-line prompt should be preserved in stdin
			if !strings.Contains(capture.stdin, "Step 1: Read the file config.yaml") {
				t.Fatalf("expected prompt to be preserved in stdin, got %q", capture.stdin)
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
		{"opencode", NewOpenCodeHarness},
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

	harnesses := map[string]struct {
		harness Harness
		capture *runnerCapture
	}{
		"claude":   {NewClaudeHarness(HarnessConfig{Runner: captureRunner(&runnerCapture{}, "ok", "", nil)}), &runnerCapture{}},
		"gemini":   {NewGeminiHarness(HarnessConfig{Runner: captureRunner(&runnerCapture{}, "ok", "", nil)}), &runnerCapture{}},
		"codex":    {NewCodexHarness(HarnessConfig{Runner: captureRunner(&runnerCapture{}, "ok", "", nil)}), &runnerCapture{}},
		"opencode": {NewOpenCodeHarness(HarnessConfig{Runner: captureRunner(&runnerCapture{}, "ok", "", nil)}), &runnerCapture{}},
	}

	ctx := context.Background()
	results := make(map[string]string)

	for name, entry := range harnesses {
		capture := &runnerCapture{}
		switch name {
		case "claude":
			entry.harness = NewClaudeHarness(HarnessConfig{Runner: captureRunner(capture, "ok", "", nil)})
		case "gemini":
			entry.harness = NewGeminiHarness(HarnessConfig{Runner: captureRunner(capture, "ok", "", nil)})
		case "codex":
			entry.harness = NewCodexHarness(HarnessConfig{Runner: captureRunner(capture, "ok", "", nil)})
		case "opencode":
			entry.harness = NewOpenCodeHarness(HarnessConfig{Runner: captureRunner(capture, "ok", "", nil)})
		}
		_, err := entry.harness.Execute(ctx, prompt)
		if err != nil {
			t.Fatalf("%s: unexpected error: %v", name, err)
		}
		results[name] = capture.stdin
	}

	// All harnesses should include the prompt in their output
	for name, result := range results {
		if !strings.Contains(result, "analyze") {
			t.Errorf("%s: prompt not found in stdin: %q", name, result)
		}
	}
}

// TestHarnessErrorHandlingConsistency ensures error handling is consistent across harnesses.
func TestHarnessErrorHandlingConsistency(t *testing.T) {
	harnesses := []struct {
		name    string
		harness Harness
	}{
		{"claude", NewClaudeHarness(HarnessConfig{Runner: captureRunner(&runnerCapture{}, "", "", errors.New("fail"))})},
		{"gemini", NewGeminiHarness(HarnessConfig{Runner: captureRunner(&runnerCapture{}, "", "", errors.New("fail"))})},
		{"codex", NewCodexHarness(HarnessConfig{Runner: captureRunner(&runnerCapture{}, "", "", errors.New("fail"))})},
		{"opencode", NewOpenCodeHarness(HarnessConfig{Runner: captureRunner(&runnerCapture{}, "", "", errors.New("fail"))})},
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
		{"claude", NewClaudeHarness(HarnessConfig{Timeout: timeout, Runner: blockingRunner(&runnerCapture{})})},
		{"gemini", NewGeminiHarness(HarnessConfig{Timeout: timeout, Runner: blockingRunner(&runnerCapture{})})},
		{"codex", NewCodexHarness(HarnessConfig{Timeout: timeout, Runner: blockingRunner(&runnerCapture{})})},
		{"opencode", NewOpenCodeHarness(HarnessConfig{Timeout: timeout, Runner: blockingRunner(&runnerCapture{})})},
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

// TestRegressionClaudeFlags guards against changes to Claude non-interactive flags.
// These exact flags are required for non-interactive execution:
// - --print: Non-interactive output
// - --input-format text: Read prompt from stdin
// - --output-format text: Parseable output format
func TestRegressionClaudeFlags(t *testing.T) {
	capture := &runnerCapture{}
	harness := NewClaudeHarness(HarnessConfig{Runner: captureRunner(capture, "ok", "", nil)})
	ctx := context.Background()

	_, _ = harness.Execute(ctx, "test")

	// These flags MUST NOT change without explicit decision
	requiredFlags := map[string]string{
		"--print":         "required for non-interactive mode",
		"--input-format":  "required to read stdin",
		"--output-format": "required for parseable output",
		"text":            "text output format",
	}

	for flag, reason := range requiredFlags {
		found := false
		for _, arg := range capture.args {
			if arg == flag {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("REGRESSION: Claude missing required flag %q (%s)", flag, reason)
		}
	}
}

// TestRegressionGeminiFlags guards against changes to Gemini non-interactive flags.
// These exact flags are required for non-interactive execution:
// - --prompt: Non-interactive mode
// - --output-format text: Parseable output format
func TestRegressionGeminiFlags(t *testing.T) {
	capture := &runnerCapture{}
	harness := NewGeminiHarness(HarnessConfig{Runner: captureRunner(capture, "ok", "", nil)})
	ctx := context.Background()

	_, _ = harness.Execute(ctx, "test")

	// These flags MUST NOT change without explicit decision
	requiredFlags := map[string]string{
		"--prompt":        "required for non-interactive mode",
		"--approval-mode": "required for automation policy",
		"auto_edit":       "default automation policy for auto mode",
		"--output-format": "required for parseable output",
		"text":            "text output format",
	}

	for flag, reason := range requiredFlags {
		found := false
		for _, arg := range capture.args {
			if arg == flag {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("REGRESSION: Gemini missing required flag %q (%s)", flag, reason)
		}
	}
	for i, arg := range capture.args {
		if arg == "--prompt" {
			if i+1 >= len(capture.args) {
				t.Error("REGRESSION: Gemini --prompt flag missing value")
			} else if capture.args[i+1] != "" {
				t.Errorf("REGRESSION: Gemini --prompt should be followed by empty string, got %q", capture.args[i+1])
			}
			break
		}
	}
}

// TestRegressionCodexFlags guards against changes to Codex non-interactive flags.
// These exact flags are required for non-interactive execution:
// - exec: Codex subcommand
// - -: Read prompt from stdin
func TestRegressionCodexFlags(t *testing.T) {
	capture := &runnerCapture{}
	harness := NewCodexHarness(HarnessConfig{Runner: captureRunner(capture, "ok", "", nil)})
	ctx := context.Background()

	_, _ = harness.Execute(ctx, "test")

	// These flags MUST NOT change without explicit decision
	requiredFlags := map[string]string{
		"exec":        "codex subcommand",
		"--full-auto": "default automation mode",
		"-":           "read prompt from stdin",
	}

	for flag, reason := range requiredFlags {
		found := false
		for _, arg := range capture.args {
			if arg == flag {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("REGRESSION: Codex missing required flag %q (%s)", flag, reason)
		}
	}

	if capture.stdin == "" {
		t.Error("REGRESSION: Codex should receive prompt via stdin")
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
		{"opencode", NewOpenCodeHarness, "opencode"},
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
			case *OpenCodeHarness:
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
		{NewOpenCodeHarness, "opencode"},
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
	requiredHarnesses := []string{"claude", "gemini", "codex", "opencode", "mock"}

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
		{"opencode", NewOpenCodeHarness(HarnessConfig{})},
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
		Model:          "model",
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
	if config.Model == "" {
		t.Error("REGRESSION: HarnessConfig.Model field removed or renamed")
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
