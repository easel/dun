package dun

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRenderPromptIncludesAutomationMode(t *testing.T) {
	dir := t.TempDir()
	promptPath := filepath.Join(dir, "prompt.md")
	if err := os.WriteFile(promptPath, []byte("mode={{ .AutomationMode }}"), 0644); err != nil {
		t.Fatalf("write prompt: %v", err)
	}

	plugin := Plugin{
		FS:   os.DirFS(dir),
		Base: ".",
	}
	check := Check{
		ID:     "test",
		Prompt: "prompt.md",
	}
	text, _, err := renderPromptText(plugin, check, nil, "yolo")
	if err != nil {
		t.Fatalf("render prompt: %v", err)
	}
	if !strings.Contains(text, "mode=yolo") {
		t.Fatalf("expected automation mode in prompt, got %q", text)
	}
}
