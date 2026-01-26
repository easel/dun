package dun

import (
	"strings"
	"testing"
)

func TestRespondParsesAgentResponse(t *testing.T) {
	input := `{"status":"pass","signal":"ok","detail":"done","next":"none","issues":[{"id":"ISSUE-1","summary":"fix it","path":"docs/foo.md"}]}`
	check, err := Respond("helix-create-architecture", strings.NewReader(input))
	if err != nil {
		t.Fatalf("respond: %v", err)
	}
	if check.ID != "helix-create-architecture" {
		t.Fatalf("expected id set, got %s", check.ID)
	}
	if check.Status != "pass" {
		t.Fatalf("expected pass, got %s", check.Status)
	}
	if check.Signal != "ok" {
		t.Fatalf("expected signal, got %s", check.Signal)
	}
	if len(check.Issues) != 1 {
		t.Fatalf("expected issues, got %v", check.Issues)
	}
}

func TestRespondErrors(t *testing.T) {
	if _, err := Respond("", strings.NewReader(`{}`)); err == nil {
		t.Fatalf("expected missing id error")
	}
	if _, err := Respond("id", strings.NewReader("not-json")); err == nil {
		t.Fatalf("expected parse error")
	}
	if _, err := Respond("id", strings.NewReader(`{"status":"pass"}`)); err == nil {
		t.Fatalf("expected missing fields error")
	}
}
