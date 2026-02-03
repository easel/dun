package dun

import (
	"context"
	"strings"
	"testing"
)

// TestRunSelfTestCheckPasses verifies the self-test check passes in normal conditions.
func TestRunSelfTestCheckPasses(t *testing.T) {
	check := Check{
		ID:   "self-test",
		Type: "self-test",
	}

	result, err := runSelfTestCheck(".", check)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Status != "pass" {
		t.Errorf("expected status 'pass', got %q", result.Status)
		t.Logf("Issues: %v", result.Issues)
		t.Logf("Detail: %s", result.Detail)
	}

	if result.ID != "self-test" {
		t.Errorf("expected ID 'self-test', got %q", result.ID)
	}

	if len(result.Issues) > 0 {
		t.Errorf("expected no issues, got %d", len(result.Issues))
		for _, issue := range result.Issues {
			t.Logf("Issue: %s", issue.Summary)
		}
	}
}

// TestRunSelfTestCheckDetails verifies the self-test check reports expected details.
func TestRunSelfTestCheckDetails(t *testing.T) {
	check := Check{
		ID:   "self-test",
		Type: "self-test",
	}

	result, err := runSelfTestCheck(".", check)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedDetails := []string{
		"DefaultRegistry initialized",
		"Harness registered: claude",
		"Harness registered: gemini",
		"Harness registered: codex",
		"Harness registered: pi",
		"Harness registered: mock",
		"Harness creates correctly",
		"automation modes",
		"Mock harness execution works",
		"context cancellation",
		"configured errors",
		"ExecuteHarness function works",
	}

	for _, expected := range expectedDetails {
		if !strings.Contains(result.Detail, expected) {
			t.Errorf("expected detail to contain %q, got: %s", expected, result.Detail)
		}
	}
}

// TestRunSelfTestCheckSignal verifies the self-test check returns correct signal.
func TestRunSelfTestCheckSignal(t *testing.T) {
	check := Check{
		ID:   "self-test",
		Type: "self-test",
	}

	result, err := runSelfTestCheck(".", check)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Status == "pass" && result.Signal != "all self-tests passed" {
		t.Errorf("expected signal 'all self-tests passed', got %q", result.Signal)
	}
}

// TestRunSelfTestCheckWithBrokenRegistry tests behavior when registry is manipulated.
func TestRunSelfTestCheckWithBrokenRegistry(t *testing.T) {
	// Save original registry
	origRegistry := DefaultRegistry
	defer func() { DefaultRegistry = origRegistry }()

	// Create a registry missing some harnesses
	DefaultRegistry = &HarnessRegistry{
		factories: make(map[string]HarnessFactory),
	}
	DefaultRegistry.Register("mock", NewMockHarness)
	// Missing claude, gemini, codex, opencode, pi

	check := Check{
		ID:   "self-test-broken",
		Type: "self-test",
	}

	result, err := runSelfTestCheck(".", check)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Status != "fail" {
		t.Errorf("expected status 'fail' with broken registry, got %q", result.Status)
	}

	if len(result.Issues) < 5 {
		t.Errorf("expected at least 5 issues for missing harnesses, got %d", len(result.Issues))
	}

	// Verify issues mention missing harnesses
	issueText := ""
	for _, issue := range result.Issues {
		issueText += issue.Summary + " "
	}

	for _, missing := range []string{"claude", "gemini", "codex", "opencode", "pi"} {
		if !strings.Contains(issueText, missing) {
			t.Errorf("expected issues to mention missing %q harness", missing)
		}
	}
}

// TestRunSelfTestCheckWithNilRegistry tests behavior when registry is nil.
func TestRunSelfTestCheckWithNilRegistry(t *testing.T) {
	// Save original registry
	origRegistry := DefaultRegistry
	defer func() { DefaultRegistry = origRegistry }()

	DefaultRegistry = nil

	check := Check{
		ID:   "self-test-nil",
		Type: "self-test",
	}

	result, err := runSelfTestCheck(".", check)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Status != "fail" {
		t.Errorf("expected status 'fail' with nil registry, got %q", result.Status)
	}

	// Should have issue about nil registry
	found := false
	for _, issue := range result.Issues {
		if strings.Contains(issue.Summary, "nil") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected issue about nil DefaultRegistry")
	}
}

// TestSelfTestCheckViaEngine tests the self-test check runs through the engine.
func TestSelfTestCheckViaEngine(t *testing.T) {
	pc := plannedCheck{
		Check: Check{
			ID:   "self-test-engine",
			Type: "self-test",
		},
	}

	result, err := runCheck(".", pc, Options{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Status != "pass" {
		t.Errorf("expected status 'pass', got %q", result.Status)
	}
}

// TestSelfTestCheckRegressionPrevention ensures self-test catches regressions.
func TestSelfTestCheckRegressionPrevention(t *testing.T) {
	// This test verifies that the self-test check would catch common regressions:
	// 1. Missing harness registration
	// 2. Harness name changes
	// 3. Automation mode support changes
	// 4. Registry initialization failures

	check := Check{
		ID:   "regression-test",
		Type: "self-test",
	}

	result, err := runSelfTestCheck(".", check)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// If this test fails, it means a regression was introduced
	if result.Status != "pass" {
		t.Fatalf("Self-test detected regression: %v", result.Issues)
	}
}

// TestSelfTestCheckWithWrongHarnessName tests detection of harness name mismatch.
func TestSelfTestCheckWithWrongHarnessName(t *testing.T) {
	// Save original registry
	origRegistry := DefaultRegistry
	defer func() { DefaultRegistry = origRegistry }()

	// Create a registry with a harness that returns the wrong name
	DefaultRegistry = &HarnessRegistry{
		factories: make(map[string]HarnessFactory),
	}
	DefaultRegistry.Register("claude", func(config HarnessConfig) Harness {
		return &wrongNameHarness{name: "wrong-name"}
	})
	DefaultRegistry.Register("gemini", NewGeminiHarness)
	DefaultRegistry.Register("codex", NewCodexHarness)
	DefaultRegistry.Register("mock", NewMockHarness)

	check := Check{
		ID:   "wrong-name-test",
		Type: "self-test",
	}

	result, err := runSelfTestCheck(".", check)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Status != "fail" {
		t.Errorf("expected status 'fail' with wrong harness name, got %q", result.Status)
	}

	// Should detect wrong name
	found := false
	for _, issue := range result.Issues {
		if strings.Contains(issue.Summary, "wrong name") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected issue about wrong harness name")
	}
}

// wrongNameHarness is a test harness that returns a different name.
type wrongNameHarness struct {
	name string
}

func (h *wrongNameHarness) Name() string { return h.name }
func (h *wrongNameHarness) Execute(ctx context.Context, prompt string) (string, error) {
	return "ok", nil
}
func (h *wrongNameHarness) SupportsAutomation(mode AutomationMode) bool { return true }

// TestSelfTestCheckWithUnexpectedResponse tests detection of unexpected mock response.
func TestSelfTestCheckWithUnexpectedResponse(t *testing.T) {
	// Save original registry
	origRegistry := DefaultRegistry
	defer func() { DefaultRegistry = origRegistry }()

	// Create a registry where mock harness ignores config
	DefaultRegistry = &HarnessRegistry{
		factories: make(map[string]HarnessFactory),
	}
	DefaultRegistry.Register("claude", NewClaudeHarness)
	DefaultRegistry.Register("gemini", NewGeminiHarness)
	DefaultRegistry.Register("codex", NewCodexHarness)
	DefaultRegistry.Register("mock", func(config HarnessConfig) Harness {
		// Ignore MockResponse config, always return different value
		return &badMockHarness{}
	})

	check := Check{
		ID:   "bad-mock-test",
		Type: "self-test",
	}

	result, err := runSelfTestCheck(".", check)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Status != "fail" {
		t.Errorf("expected status 'fail' with unexpected response, got %q", result.Status)
	}

	// Should detect unexpected response
	found := false
	for _, issue := range result.Issues {
		if strings.Contains(issue.Summary, "unexpected response") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected issue about unexpected response")
	}
}

// badMockHarness always returns "wrong-response" regardless of config.
type badMockHarness struct{}

func (h *badMockHarness) Name() string { return "mock" }
func (h *badMockHarness) Execute(ctx context.Context, prompt string) (string, error) {
	return "wrong-response", nil
}
func (h *badMockHarness) SupportsAutomation(mode AutomationMode) bool { return true }

// TestSelfTestCheckWithNonCancellingHarness tests detection of harness that ignores cancellation.
func TestSelfTestCheckWithNonCancellingHarness(t *testing.T) {
	// Save original registry
	origRegistry := DefaultRegistry
	defer func() { DefaultRegistry = origRegistry }()

	// Create a registry where mock harness ignores context cancellation
	DefaultRegistry = &HarnessRegistry{
		factories: make(map[string]HarnessFactory),
	}
	DefaultRegistry.Register("claude", NewClaudeHarness)
	DefaultRegistry.Register("gemini", NewGeminiHarness)
	DefaultRegistry.Register("codex", NewCodexHarness)
	DefaultRegistry.Register("mock", func(config HarnessConfig) Harness {
		return &nonCancellingHarness{response: config.MockResponse}
	})

	check := Check{
		ID:   "non-cancelling-test",
		Type: "self-test",
	}

	result, err := runSelfTestCheck(".", check)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Status != "fail" {
		t.Errorf("expected status 'fail' with non-cancelling harness, got %q", result.Status)
	}

	// Should detect context not respected
	found := false
	for _, issue := range result.Issues {
		if strings.Contains(issue.Summary, "context cancellation") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected issue about context cancellation")
	}
}

// nonCancellingHarness ignores context cancellation.
type nonCancellingHarness struct {
	response string
}

func (h *nonCancellingHarness) Name() string { return "mock" }
func (h *nonCancellingHarness) Execute(ctx context.Context, prompt string) (string, error) {
	return h.response, nil
}
func (h *nonCancellingHarness) SupportsAutomation(mode AutomationMode) bool { return true }

// TestSelfTestCheckWithNoErrorHarness tests detection of harness that ignores configured error.
func TestSelfTestCheckWithNoErrorHarness(t *testing.T) {
	// Save original registry
	origRegistry := DefaultRegistry
	defer func() { DefaultRegistry = origRegistry }()

	// Create a registry where mock harness ignores configured error
	DefaultRegistry = &HarnessRegistry{
		factories: make(map[string]HarnessFactory),
	}
	DefaultRegistry.Register("claude", NewClaudeHarness)
	DefaultRegistry.Register("gemini", NewGeminiHarness)
	DefaultRegistry.Register("codex", NewCodexHarness)
	DefaultRegistry.Register("mock", func(config HarnessConfig) Harness {
		return &noErrorHarness{}
	})

	check := Check{
		ID:   "no-error-test",
		Type: "self-test",
	}

	result, err := runSelfTestCheck(".", check)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Status != "fail" {
		t.Errorf("expected status 'fail' with no-error harness, got %q", result.Status)
	}

	// Should detect error not returned
	found := false
	for _, issue := range result.Issues {
		if strings.Contains(issue.Summary, "configured error") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected issue about configured error not returned")
	}
}

// noErrorHarness ignores MockError config - never returns errors.
type noErrorHarness struct{}

func (h *noErrorHarness) Name() string { return "mock" }
func (h *noErrorHarness) Execute(ctx context.Context, prompt string) (string, error) {
	return "ok", nil
}
func (h *noErrorHarness) SupportsAutomation(mode AutomationMode) bool { return true }

// TestSelfTestCheckWithUnsupportedModes tests detection of harness without mode support.
func TestSelfTestCheckWithUnsupportedModes(t *testing.T) {
	// Save original registry
	origRegistry := DefaultRegistry
	defer func() { DefaultRegistry = origRegistry }()

	// Create a registry with a harness that doesn't support all modes
	DefaultRegistry = &HarnessRegistry{
		factories: make(map[string]HarnessFactory),
	}
	DefaultRegistry.Register("claude", func(config HarnessConfig) Harness {
		return &limitedModeHarness{}
	})
	DefaultRegistry.Register("gemini", NewGeminiHarness)
	DefaultRegistry.Register("codex", NewCodexHarness)
	DefaultRegistry.Register("mock", NewMockHarness)

	check := Check{
		ID:   "limited-mode-test",
		Type: "self-test",
	}

	result, err := runSelfTestCheck(".", check)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Status != "fail" {
		t.Errorf("expected status 'fail' with limited mode harness, got %q", result.Status)
	}

	// Should detect unsupported mode
	found := false
	for _, issue := range result.Issues {
		if strings.Contains(issue.Summary, "does not support") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected issue about unsupported mode")
	}
}

// limitedModeHarness only supports manual mode.
type limitedModeHarness struct{}

func (h *limitedModeHarness) Name() string { return "claude" }
func (h *limitedModeHarness) Execute(ctx context.Context, prompt string) (string, error) {
	return "ok", nil
}
func (h *limitedModeHarness) SupportsAutomation(mode AutomationMode) bool {
	return mode == AutomationManual // Only supports manual
}
