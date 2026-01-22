package dun

import (
	"strings"
	"testing"
)

func TestRespondParsesAgentResponse(t *testing.T) {
	input := `{"status":"pass","signal":"ok","detail":"done","next":"none"}`
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
}
