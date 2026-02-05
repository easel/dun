package dun

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func runAgentCheckFromSpec(root string, plugin Plugin, check Check, opts Options) (CheckResult, error) {
	def := CheckDefinition{ID: check.ID, Description: check.Description}
	config := AgentCheckConfig{Prompt: check.Prompt, Inputs: check.Inputs, ResponseSchema: check.ResponseSchema}
	return runAgentCheck(root, plugin, def, config, opts)
}

func TestNormalizeAgentModeInvalid(t *testing.T) {
	if _, err := normalizeAgentMode("nope"); err == nil {
		t.Fatalf("expected error for invalid agent mode")
	}
}

func TestNormalizeAutomationModeInvalid(t *testing.T) {
	if _, err := normalizeAutomationMode("nope"); err == nil {
		t.Fatalf("expected error for invalid automation mode")
	}
}

func TestNormalizeAutomationModeVariants(t *testing.T) {
	cases := map[string]string{
		"":       "auto",
		"auto":   "auto",
		"manual": "manual",
		"plan":       "plan",
		"yolo": "yolo",
	}
	for input, expected := range cases {
		got, err := normalizeAutomationMode(input)
		if err != nil {
			t.Fatalf("normalize %q: %v", input, err)
		}
		if got != expected {
			t.Fatalf("expected %q, got %q", expected, got)
		}
	}
}

func TestPromptResultDefaultsAndOverridesNext(t *testing.T) {
	check := Check{ID: "check-1"}
	envelope := PromptEnvelope{}
	res := promptResult(CheckDefinition{ID: check.ID, Description: check.Description}, envelope, "signal", "detail")
	if !strings.Contains(res.Next, "dun respond") {
		t.Fatalf("expected default next, got %q", res.Next)
	}

	envelope.Callback.Command = "custom command"
	res = promptResult(CheckDefinition{ID: check.ID, Description: check.Description}, envelope, "signal", "detail")
	if res.Next != "custom command" {
		t.Fatalf("expected custom next, got %q", res.Next)
	}
}

func TestLoadPromptTemplateFallback(t *testing.T) {
	dir := t.TempDir()
	plugin := Plugin{FS: os.DirFS(dir), Base: "."}
	text, err := loadPromptTemplate(plugin, "missing.md")
	if err != nil {
		t.Fatalf("load prompt fallback: %v", err)
	}
	if text != "missing.md" {
		t.Fatalf("expected fallback path, got %q", text)
	}
}

func TestRenderPromptTextWithSchema(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "prompt.md"), "hello {{ .CheckID }}")
	writeFile(t, filepath.Join(dir, "schema.json"), `{"type":"object"}`)
	plugin := Plugin{FS: os.DirFS(dir), Base: "."}
	check := Check{ID: "id", Prompt: "prompt.md", ResponseSchema: "schema.json"}

	text, schema, err := renderPromptText(plugin, AgentCheckConfig{Prompt: check.Prompt, Inputs: check.Inputs, ResponseSchema: check.ResponseSchema}, check.ID, nil, "auto")
	if err != nil {
		t.Fatalf("render prompt: %v", err)
	}
	if !strings.Contains(text, "Response Schema:") {
		t.Fatalf("expected schema section in prompt")
	}
	if schema == "" {
		t.Fatalf("expected schema text")
	}
}

func TestResolveInputsGlobAndMissing(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "a.txt"), "A")
	writeFile(t, filepath.Join(dir, "b.txt"), "B")

	inputs, err := resolveInputs(dir, []string{"*.txt"})
	if err != nil {
		t.Fatalf("resolve inputs: %v", err)
	}
	if len(inputs) != 2 {
		t.Fatalf("expected 2 inputs, got %d", len(inputs))
	}

	inputs, err = resolveInputs(dir, []string{"missing.txt"})
	if err != nil {
		t.Fatalf("resolve missing input: %v", err)
	}
	if len(inputs) != 0 {
		t.Fatalf("expected no inputs for missing file, got %d", len(inputs))
	}
}

func TestResolveInputsBadGlob(t *testing.T) {
	_, err := resolveInputs(t.TempDir(), []string{"["})
	if err == nil {
		t.Fatalf("expected glob error")
	}
}

func TestExecAgentErrorsAndSuccess(t *testing.T) {
	orig := execAgentOutput
	execAgentOutput = func(cmdStr, prompt string, timeout time.Duration) ([]byte, error) {
		switch cmdStr {
		case "fail":
			return nil, errors.New("command failed")
		case "badjson":
			return []byte("not-json"), nil
		case "ok":
			return []byte(`{"status":"pass","signal":"ok"}`), nil
		default:
			return nil, errors.New("unexpected cmd")
		}
	}
	t.Cleanup(func() { execAgentOutput = orig })

	_, err := execAgent("fail", "prompt", 1*time.Second)
	if err == nil {
		t.Fatalf("expected command error")
	}

	_, err = execAgent("badjson", "prompt", 1*time.Second)
	if err == nil {
		t.Fatalf("expected json error")
	}

	parsed, err := execAgent("ok", "prompt", 1*time.Second)
	if err != nil {
		t.Fatalf("exec agent: %v", err)
	}
	if parsed.Status != "pass" {
		t.Fatalf("expected pass, got %s", parsed.Status)
	}
}

func TestRunAgentCheckMissingCmd(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "prompt.md"), "hi")
	plugin := Plugin{FS: os.DirFS(dir), Base: "."}
	check := Check{ID: "test", Prompt: "prompt.md", Description: "desc"}
	opts := Options{AgentMode: "auto", AutomationMode: "auto"}

	res, err := runAgentCheckFromSpec(".", plugin, check, opts)
	if err != nil {
		t.Fatalf("run agent check: %v", err)
	}
	if res.Status != "prompt" {
		t.Fatalf("expected prompt, got %s", res.Status)
	}
	if !strings.Contains(res.Signal, "agent not configured") {
		t.Fatalf("expected agent not configured, got %q", res.Signal)
	}
}

func TestRunAgentCheckInvalidMode(t *testing.T) {
	plugin := Plugin{FS: os.DirFS(t.TempDir()), Base: "."}
	check := Check{ID: "test", Prompt: "missing.md"}
	opts := Options{AgentMode: "nope", AutomationMode: "auto"}
	_, err := runAgentCheckFromSpec(".", plugin, check, opts)
	if err == nil {
		t.Fatalf("expected invalid agent mode error")
	}
}

func TestRunAgentCheckUsesEnvCmd(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "prompt.md"), "hi")
	plugin := Plugin{FS: os.DirFS(dir), Base: "."}
	check := Check{ID: "test", Prompt: "prompt.md", Description: "desc"}
	orig := execAgentOutput
	execAgentOutput = func(cmdStr, prompt string, timeout time.Duration) ([]byte, error) {
		if cmdStr != "ok" {
			return nil, errors.New("unexpected cmd")
		}
		return []byte(`{"status":"pass","signal":"ok"}`), nil
	}
	t.Cleanup(func() { execAgentOutput = orig })

	t.Setenv("DUN_AGENT_CMD", "ok")
	opts := Options{AgentMode: "auto", AutomationMode: "auto", AgentTimeout: time.Second}
	res, err := runAgentCheckFromSpec(".", plugin, check, opts)
	if err != nil {
		t.Fatalf("run agent: %v", err)
	}
	if res.Status != "pass" {
		t.Fatalf("expected pass, got %s", res.Status)
	}
}

func TestRunAgentCheckDefaultTimeout(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "prompt.md"), "hi")
	plugin := Plugin{FS: os.DirFS(dir), Base: "."}
	check := Check{ID: "test", Prompt: "prompt.md", Description: "desc"}
	orig := execAgentOutput
	execAgentOutput = func(cmdStr, prompt string, timeout time.Duration) ([]byte, error) {
		if cmdStr != "ok" {
			return nil, errors.New("unexpected cmd")
		}
		return []byte(`{"status":"pass","signal":"ok"}`), nil
	}
	t.Cleanup(func() { execAgentOutput = orig })
	opts := Options{
		AgentMode:      "auto",
		AutomationMode: "auto",
		AgentCmd:       "ok",
	}
	res, err := runAgentCheckFromSpec(".", plugin, check, opts)
	if err != nil {
		t.Fatalf("run agent: %v", err)
	}
	if res.Status != "pass" {
		t.Fatalf("expected pass")
	}
}

func TestRunAgentCheckInvalidAutomationMode(t *testing.T) {
	plugin := Plugin{FS: os.DirFS(t.TempDir()), Base: "."}
	check := Check{ID: "test", Prompt: "missing.md"}
	opts := Options{AgentMode: "prompt", AutomationMode: "nope"}
	_, err := runAgentCheckFromSpec(".", plugin, check, opts)
	if err == nil {
		t.Fatalf("expected error for invalid automation mode")
	}
}

func TestRunAgentCheckPromptMode(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "prompt.md"), "Check-ID: {{ .CheckID }}")
	plugin := Plugin{FS: os.DirFS(dir), Base: "."}
	check := Check{ID: "check", Prompt: "prompt.md", Description: "desc"}
	opts := Options{AgentMode: "prompt", AutomationMode: "auto"}

	res, err := runAgentCheckFromSpec(".", plugin, check, opts)
	if err != nil {
		t.Fatalf("run agent check prompt: %v", err)
	}
	if res.Status != "prompt" {
		t.Fatalf("expected prompt, got %s", res.Status)
	}
	if res.Prompt == nil || !strings.Contains(res.Prompt.Prompt, "Check-ID: check") {
		t.Fatalf("expected prompt content")
	}
}

func TestRunAgentCheckAutoSuccessAndInvalidResponse(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "prompt.md"), "hi")
	plugin := Plugin{FS: os.DirFS(dir), Base: "."}
	check := Check{ID: "test", Prompt: "prompt.md", Description: "desc"}

	orig := execAgentOutput
	execAgentOutput = func(cmdStr, prompt string, timeout time.Duration) ([]byte, error) {
		switch cmdStr {
		case "ok":
			return []byte(`{"status":"pass","signal":"ok"}`), nil
		case "missing-signal":
			return []byte(`{"status":"pass"}`), nil
		default:
			return nil, errors.New("unexpected cmd")
		}
	}
	t.Cleanup(func() { execAgentOutput = orig })

	opts := Options{
		AgentMode:      "auto",
		AutomationMode: "auto",
		AgentCmd:       "ok",
		AgentTimeout:   time.Second,
	}
	res, err := runAgentCheckFromSpec(".", plugin, check, opts)
	if err != nil {
		t.Fatalf("run agent auto: %v", err)
	}
	if res.Status != "pass" {
		t.Fatalf("expected pass, got %s", res.Status)
	}

	opts.AgentCmd = "missing-signal"
	_, err = runAgentCheckFromSpec(".", plugin, check, opts)
	if err == nil {
		t.Fatalf("expected error for missing signal")
	}
}

func TestRunAgentCheckBuildPromptEnvelopeSkipsMissingInputs(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "prompt.md"), "hi")
	plugin := Plugin{FS: os.DirFS(dir), Base: "."}
	check := Check{ID: "test", Prompt: "prompt.md", Inputs: []string{"missing.txt"}}
	opts := Options{AgentMode: "prompt", AutomationMode: "auto"}
	res, err := runAgentCheckFromSpec(dir, plugin, check, opts)
	if err != nil {
		t.Fatalf("run agent check: %v", err)
	}
	if res.Status != "prompt" {
		t.Fatalf("expected prompt, got %s", res.Status)
	}
	if res.Prompt == nil {
		t.Fatalf("expected prompt envelope")
	}
	if len(res.Prompt.Inputs) != 0 {
		t.Fatalf("expected no inputs, got %v", res.Prompt.Inputs)
	}
}

func TestRunAgentCheckExecAgentError(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "prompt.md"), "hi")
	plugin := Plugin{FS: os.DirFS(dir), Base: "."}
	check := Check{ID: "test", Prompt: "prompt.md", Description: "desc"}
	orig := execAgentOutput
	execAgentOutput = func(cmdStr, prompt string, timeout time.Duration) ([]byte, error) {
		return nil, errors.New("boom")
	}
	t.Cleanup(func() { execAgentOutput = orig })
	opts := Options{
		AgentMode:      "auto",
		AutomationMode: "auto",
		AgentCmd:       "fail",
		AgentTimeout:   time.Second,
	}
	if _, err := runAgentCheckFromSpec(dir, plugin, check, opts); err == nil {
		t.Fatalf("expected agent command error")
	}
}

func TestBuildPromptEnvelopeInputs(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "prompt.md"), "hello")
	writeFile(t, filepath.Join(dir, "a.txt"), "A")
	plugin := Plugin{FS: os.DirFS(dir), Base: "."}
	check := Check{ID: "id", Prompt: "prompt.md", Inputs: []string{"a.txt"}}
	envelope, err := buildPromptEnvelope(dir, plugin, CheckDefinition{ID: check.ID, Description: check.Description}, AgentCheckConfig{Prompt: check.Prompt, Inputs: check.Inputs, ResponseSchema: check.ResponseSchema}, "auto")
	if err != nil {
		t.Fatalf("build prompt envelope: %v", err)
	}
	if len(envelope.Inputs) != 1 || envelope.Inputs[0] != "a.txt" {
		t.Fatalf("expected input path, got %v", envelope.Inputs)
	}
}

func TestBuildPromptEnvelopeMissingPrompt(t *testing.T) {
	plugin := Plugin{FS: os.DirFS(t.TempDir()), Base: "."}
	check := Check{ID: "id"}
	_, err := buildPromptEnvelope(".", plugin, CheckDefinition{ID: check.ID, Description: check.Description}, AgentCheckConfig{Prompt: check.Prompt, Inputs: check.Inputs, ResponseSchema: check.ResponseSchema}, "auto")
	if err == nil {
		t.Fatalf("expected error for missing prompt")
	}
}

func TestBuildPromptEnvelopeMissingInput(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "prompt.md"), "hello")
	plugin := Plugin{FS: os.DirFS(dir), Base: "."}
	check := Check{ID: "id", Prompt: "prompt.md", Inputs: []string{"missing.txt"}}
	envelope, err := buildPromptEnvelope(dir, plugin, CheckDefinition{ID: check.ID, Description: check.Description}, AgentCheckConfig{Prompt: check.Prompt, Inputs: check.Inputs, ResponseSchema: check.ResponseSchema}, "auto")
	if err != nil {
		t.Fatalf("build prompt envelope: %v", err)
	}
	if len(envelope.Inputs) != 0 {
		t.Fatalf("expected no inputs, got %v", envelope.Inputs)
	}
}

func TestRenderPromptTextNoSchema(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "prompt.md"), "hello")
	plugin := Plugin{FS: os.DirFS(dir), Base: "."}
	check := Check{ID: "id", Prompt: "prompt.md"}
	text, schema, err := renderPromptText(plugin, AgentCheckConfig{Prompt: check.Prompt, Inputs: check.Inputs, ResponseSchema: check.ResponseSchema}, check.ID, nil, "auto")
	if err != nil {
		t.Fatalf("render prompt: %v", err)
	}
	if schema != "" {
		t.Fatalf("expected empty schema")
	}
	if text == "" {
		t.Fatalf("expected prompt text")
	}
}

func TestRenderPromptTextParseError(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "prompt.md"), "{{ .CheckID")
	plugin := Plugin{FS: os.DirFS(dir), Base: "."}
	check := Check{ID: "id", Prompt: "prompt.md"}
	if _, _, err := renderPromptText(plugin, AgentCheckConfig{Prompt: check.Prompt, Inputs: check.Inputs, ResponseSchema: check.ResponseSchema}, check.ID, nil, "auto"); err == nil {
		t.Fatalf("expected parse error")
	}
}

func TestRenderPromptTextExecuteError(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "prompt.md"), "{{ index .Inputs 0 }}")
	plugin := Plugin{FS: os.DirFS(dir), Base: "."}
	check := Check{ID: "id", Prompt: "prompt.md"}
	if _, _, err := renderPromptText(plugin, AgentCheckConfig{Prompt: check.Prompt, Inputs: check.Inputs, ResponseSchema: check.ResponseSchema}, check.ID, nil, "auto"); err == nil {
		t.Fatalf("expected execute error")
	}
}

func TestRenderPromptTextSchemaError(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "prompt.md"), "hello")
	plugin := Plugin{FS: selectiveFS{root: dir, deny: "schema.json"}, Base: "."}
	check := Check{ID: "id", Prompt: "prompt.md", ResponseSchema: "schema.json"}
	if _, _, err := renderPromptText(plugin, AgentCheckConfig{Prompt: check.Prompt, Inputs: check.Inputs, ResponseSchema: check.ResponseSchema}, check.ID, nil, "auto"); err == nil {
		t.Fatalf("expected schema load error")
	}
}

func TestLoadPromptTemplateError(t *testing.T) {
	plugin := Plugin{FS: errFS{}, Base: "."}
	_, err := loadPromptTemplate(plugin, "prompt.md")
	if err == nil {
		t.Fatalf("expected fs error")
	}
}

type errFS struct{}

func (errFS) Open(name string) (fs.File, error) {
	return nil, os.ErrPermission
}

type selectiveFS struct {
	root string
	deny string
}

func (fsys selectiveFS) Open(name string) (fs.File, error) {
	if name == fsys.deny {
		return nil, os.ErrPermission
	}
	return os.Open(filepath.Join(fsys.root, name))
}

func TestResolveInputsRelErrorFallback(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "a.txt")
	writeFile(t, file, "A")
	orig := relPath
	relPath = func(string, string) (string, error) {
		return "", os.ErrInvalid
	}
	t.Cleanup(func() { relPath = orig })

	inputs, err := resolveInputs(dir, []string{"a.txt"})
	if err != nil {
		t.Fatalf("resolve inputs: %v", err)
	}
	if len(inputs) != 1 {
		t.Fatalf("expected one input")
	}
	if inputs[0].Path != filepath.ToSlash(file) {
		t.Fatalf("expected absolute fallback, got %q", inputs[0].Path)
	}
}
