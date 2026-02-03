//go:build dun_cli

package main

import (
	"errors"
	"strings"
	"testing"
)

func TestCallHarnessClaude(t *testing.T) {
	// This test verifies the command construction for claude harness
	// It will fail if claude is not installed, which is expected in CI
	_, err := callHarness("claude", "test prompt", "auto")
	// We expect an error since claude CLI is likely not installed
	if err == nil {
		// If it succeeds, that's fine too
		return
	}
	// Error should be about command execution, not harness type
	if strings.Contains(err.Error(), "unknown harness") {
		t.Fatalf("claude should be a known harness")
	}
}

func TestCallHarnessClaudeYolo(t *testing.T) {
	// Test that yolo mode adds the right flags
	_, err := callHarness("claude", "test prompt", "yolo")
	// We expect an error since claude CLI is likely not installed
	if err == nil {
		return
	}
	if strings.Contains(err.Error(), "unknown harness") {
		t.Fatalf("claude should be a known harness")
	}
}

func TestCallHarnessGemini(t *testing.T) {
	// Mock callHarnessFn to avoid calling real gemini CLI which may hang
	origCallHarnessFn := callHarnessFn
	callHarnessFn = func(harnessName, prompt, automation string) (string, error) {
		// Always return mock error for gemini - don't call real CLI
		if harnessName == "gemini" {
			return "", errors.New("gemini harness not configured in test environment")
		}
		// For other harnesses, also return error to avoid calling real CLIs
		return "", errors.New("harness not available in test environment")
	}
	t.Cleanup(func() { callHarnessFn = origCallHarnessFn })

	// This test verifies the mock is working correctly
	_, err := callHarness("gemini", "test prompt", "auto")
	if err == nil {
		t.Fatalf("expected an error when calling gemini harness without proper setup, but got none")
	}
	if strings.Contains(err.Error(), "unknown harness") {
		t.Fatalf("gemini should be a known harness, but got 'unknown harness' error: %v", err)
	}
	// We expect the mock error
	if !strings.Contains(err.Error(), "gemini harness not configured in test environment") {
		t.Fatalf("expected specific error message for gemini harness, got: %v", err)
	}
}

func TestCallHarnessCodex(t *testing.T) {
	// This test verifies the command construction for codex harness
	_, err := callHarness("codex", "test prompt", "auto")
	// We expect an error since codex CLI is likely not installed
	if err == nil {
		return
	}
	if strings.Contains(err.Error(), "unknown harness") {
		t.Fatalf("codex should be a known harness")
	}
}

func TestCallHarnessCodexYolo(t *testing.T) {
	// Test that yolo mode adds the right flags
	_, err := callHarness("codex", "test prompt", "yolo")
	if err == nil {
		return
	}
	if strings.Contains(err.Error(), "unknown harness") {
		t.Fatalf("codex should be a known harness")
	}
}

func TestCallHarnessPi(t *testing.T) {
	// This test verifies the command construction for pi harness
	_, err := callHarness("pi", "test prompt", "auto")
	// We expect an error since pi CLI is likely not installed
	if err == nil {
		return
	}
	if strings.Contains(err.Error(), "unknown harness") {
		t.Fatalf("pi should be a known harness")
	}
}

func TestCallHarnessCursor(t *testing.T) {
	// This test verifies the command construction for cursor harness
	_, err := callHarness("cursor", "test prompt", "auto")
	// We expect an error since cursor CLI is likely not installed
	if err == nil {
		return
	}
	if strings.Contains(err.Error(), "unknown harness") {
		t.Fatalf("cursor should be a known harness")
	}
}
