package dun

import (
	"context"
	"testing"
)

func TestPingHarnessParsesJSON(t *testing.T) {
	result, err := PingHarness(context.Background(), "mock", HarnessConfig{MockResponse: `{"ok":true,"model":"test-model"}`})
	if err != nil {
		t.Fatalf("ping harness: %v", err)
	}
	if !result.Live {
		t.Fatalf("expected live ping")
	}
	if result.Model != "test-model" {
		t.Fatalf("expected model test-model, got %q", result.Model)
	}
	if result.Detail != "" {
		t.Fatalf("expected empty detail, got %q", result.Detail)
	}
}

func TestPingHarnessNonJSONFallback(t *testing.T) {
	result, err := PingHarness(context.Background(), "mock", HarnessConfig{MockResponse: "model: alpha"})
	if err != nil {
		t.Fatalf("ping harness: %v", err)
	}
	if !result.Live {
		t.Fatalf("expected live ping")
	}
	if result.Model != "alpha" {
		t.Fatalf("expected model alpha, got %q", result.Model)
	}
	if result.Detail == "" {
		t.Fatalf("expected detail for non-json response")
	}
}
