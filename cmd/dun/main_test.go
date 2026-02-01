package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/easel/dun/internal/dun"
)

func TestCheckUsesConfigAgentAuto(t *testing.T) {
	root := setupRepoFromFixture(t, "helix-alignment")
	agentCmd := "bash " + fixturePath(t, "internal/testdata/agent/agent.sh")
	writeConfig(t, root, agentCmd)

	output := runInDir(t, root, []string{"check"})
	var result dun.Result
	if err := json.Unmarshal(output, &result); err != nil {
		t.Fatalf("decode output: %v", err)
	}

	check := findCheck(t, result, "helix-align-specs")
	if check.Status == "prompt" {
		t.Fatalf("expected auto agent response, got prompt")
	}
	if check.Status != "warn" {
		t.Fatalf("expected warn, got %s", check.Status)
	}
}

func TestCheckResolvesRepoRootFromSubdir(t *testing.T) {
	root := setupRepoFromFixture(t, "helix-alignment")
	agentCmd := "bash " + fixturePath(t, "internal/testdata/agent/agent.sh")
	writeConfig(t, root, agentCmd)

	subdir := filepath.Join(root, "nested", "work")
	if err := os.MkdirAll(subdir, 0755); err != nil {
		t.Fatalf("mkdir subdir: %v", err)
	}

	output := runInDir(t, subdir, []string{"check"})
	var result dun.Result
	if err := json.Unmarshal(output, &result); err != nil {
		t.Fatalf("decode output: %v", err)
	}

	check := findCheck(t, result, "helix-align-specs")
	if check.Status == "prompt" {
		t.Fatalf("expected auto agent response from repo root, got prompt")
	}
}

func TestMainExitCode(t *testing.T) {
	origExit := exit
	origArgs := os.Args
	defer func() {
		exit = origExit
		os.Args = origArgs
	}()

	var code int
	exit = func(c int) { code = c }
	os.Args = []string{"dun", "unknown"}
	main()

	if code != dun.ExitUsageError {
		t.Fatalf("expected exit code %d, got %d", dun.ExitUsageError, code)
	}
}

func TestRunUnknownCommand(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{"unknown"}, &stdout, &stderr)
	if code != dun.ExitUsageError {
		t.Fatalf("expected code %d, got %d", dun.ExitUsageError, code)
	}
	if !strings.Contains(stderr.String(), "unknown command") {
		t.Fatalf("expected unknown command message")
	}
}

func TestRunDefaultsToCheck(t *testing.T) {
	root := setupEmptyRepo(t)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := runInDirWithWriters(t, root, []string{}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected success, got %d", code)
	}
}

func TestRunCheckUnknownFormat(t *testing.T) {
	root := setupEmptyRepo(t)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := runInDirWithWriters(t, root, []string{"check", "--format=bad"}, &stdout, &stderr)
	if code != dun.ExitUsageError {
		t.Fatalf("expected code %d, got %d", dun.ExitUsageError, code)
	}
}

func TestRunCheckRepoError(t *testing.T) {
	root := setupEmptyRepo(t)
	orig := checkRepo
	checkRepo = func(_ string, _ dun.Options) (dun.Result, error) {
		return dun.Result{}, errors.New("boom")
	}
	t.Cleanup(func() { checkRepo = orig })

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := runInDirWithWriters(t, root, []string{"check"}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("expected code 1, got %d", code)
	}
}

func TestRunCheckLLMOutput(t *testing.T) {
	root := setupEmptyRepo(t)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := runInDirWithWriters(t, root, []string{"check", "--format=llm"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected success, got %d", code)
	}
	if !strings.Contains(stdout.String(), "check:git-status") {
		t.Fatalf("expected llm output")
	}
}

func TestRunCheckJSONEncodeError(t *testing.T) {
	root := setupEmptyRepo(t)
	errWriter := &failWriter{err: errors.New("write failed")}
	var stderr bytes.Buffer
	code := runInDirWithWriters(t, root, []string{"check", "--format=json"}, errWriter, &stderr)
	if code != 1 {
		t.Fatalf("expected code 1, got %d", code)
	}
}

func TestRunCheckConfigError(t *testing.T) {
	root := setupEmptyRepo(t)
	cfgPath := filepath.Join(root, ".dun", "config.yaml")
	if err := os.MkdirAll(filepath.Dir(cfgPath), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(cfgPath, []byte("agent:\n  cmd: ["), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := runInDirWithWriters(t, root, []string{"check"}, &stdout, &stderr)
	if code != dun.ExitConfigError {
		t.Fatalf("expected code %d, got %d", dun.ExitConfigError, code)
	}
}

func TestRunCheckParseError(t *testing.T) {
	root := setupEmptyRepo(t)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := runInDirWithWriters(t, root, []string{"check", "--agent-timeout=bad"}, &stdout, &stderr)
	if code != 4 {
		t.Fatalf("expected code 4, got %d", code)
	}
}

func TestRunListParseError(t *testing.T) {
	root := setupEmptyRepo(t)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := runInDirWithWriters(t, root, []string{"list", "--badflag"}, &stdout, &stderr)
	if code != 4 {
		t.Fatalf("expected code 4, got %d", code)
	}
}

func TestRunListPlanError(t *testing.T) {
	root := setupEmptyRepo(t)
	orig := planRepo
	planRepo = func(_ string) (dun.Plan, error) {
		return dun.Plan{}, errors.New("boom")
	}
	t.Cleanup(func() { planRepo = orig })

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := runInDirWithWriters(t, root, []string{"list"}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("expected code 1, got %d", code)
	}
}

func TestRunListJSONEncodeError(t *testing.T) {
	root := setupEmptyRepo(t)
	errWriter := &failWriter{err: errors.New("write failed")}
	var stderr bytes.Buffer
	code := runInDirWithWriters(t, root, []string{"list", "--format=json"}, errWriter, &stderr)
	if code != 1 {
		t.Fatalf("expected code 1, got %d", code)
	}
}

func TestRunListTextAndJSON(t *testing.T) {
	root := setupEmptyRepo(t)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := runInDirWithWriters(t, root, []string{"list"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected success, got %d", code)
	}
	if !strings.Contains(stdout.String(), "git-status") {
		t.Fatalf("expected list output")
	}

	stdout.Reset()
	code = runInDirWithWriters(t, root, []string{"list", "--format=json"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected success, got %d", code)
	}
	if !strings.Contains(stdout.String(), "\"Checks\"") {
		t.Fatalf("expected json output")
	}
}

func TestRunListConfigError(t *testing.T) {
	root := setupEmptyRepo(t)
	cfgPath := filepath.Join(root, "bad.yaml")
	if err := os.WriteFile(cfgPath, []byte("agent:\n  cmd: ["), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := runInDirWithWriters(t, root, []string{"list", "--config", "bad.yaml"}, &stdout, &stderr)
	if code != dun.ExitConfigError {
		t.Fatalf("expected code %d, got %d", dun.ExitConfigError, code)
	}
}

func TestRunExplainParseError(t *testing.T) {
	root := setupEmptyRepo(t)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := runInDirWithWriters(t, root, []string{"explain", "--badflag"}, &stdout, &stderr)
	if code != 4 {
		t.Fatalf("expected code 4, got %d", code)
	}
}

func TestRunExplainConfigError(t *testing.T) {
	root := setupEmptyRepo(t)
	cfgPath := filepath.Join(root, "bad.yaml")
	if err := os.WriteFile(cfgPath, []byte(":"), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := runInDirWithWriters(t, root, []string{"explain", "--config", "bad.yaml", "git-status"}, &stdout, &stderr)
	if code != dun.ExitConfigError {
		t.Fatalf("expected code %d, got %d", dun.ExitConfigError, code)
	}
}

func TestRunExplainPlanError(t *testing.T) {
	root := setupEmptyRepo(t)
	orig := planRepo
	planRepo = func(_ string) (dun.Plan, error) {
		return dun.Plan{}, errors.New("boom")
	}
	t.Cleanup(func() { planRepo = orig })

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := runInDirWithWriters(t, root, []string{"explain", "git-status"}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("expected code 1, got %d", code)
	}
}

func TestRunExplainJSONEncodeError(t *testing.T) {
	root := setupEmptyRepo(t)
	orig := planRepo
	planRepo = func(_ string) (dun.Plan, error) {
		return dun.Plan{
			Checks: []dun.PlannedCheck{{ID: "check", Description: "desc", Type: "rule-set"}},
		}, nil
	}
	t.Cleanup(func() { planRepo = orig })

	errWriter := &failWriter{err: errors.New("write failed")}
	var stderr bytes.Buffer
	code := runInDirWithWriters(t, root, []string{"explain", "--format=json", "check"}, errWriter, &stderr)
	if code != 1 {
		t.Fatalf("expected code 1, got %d", code)
	}
}

func TestRunExplainOutputsExtraFields(t *testing.T) {
	root := setupEmptyRepo(t)
	orig := planRepo
	planRepo = func(_ string) (dun.Plan, error) {
		return dun.Plan{
			Checks: []dun.PlannedCheck{
				{
					ID:          "demo",
					Description: "demo",
					Type:        "agent",
					Phase:       "frame",
					PluginID:    "plugin",
					Conditions:  []dun.Rule{{Type: "path-exists", Path: "file.txt"}},
					Inputs:      []string{"docs/a.md"},
					GateFiles:   []string{"gates.yml"},
					StateRules:  "state.yml",
					Prompt:      "prompt.md",
				},
			},
		}, nil
	}
	t.Cleanup(func() { planRepo = orig })

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := runInDirWithWriters(t, root, []string{"explain", "demo"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected success, got %d", code)
	}
	output := stdout.String()
	for _, needle := range []string{"conditions:", "inputs:", "gate_files:", "state_rules:", "prompt:"} {
		if !strings.Contains(output, needle) {
			t.Fatalf("expected %q in output", needle)
		}
	}
}

func TestRunExplainUsageAndUnknown(t *testing.T) {
	root := setupEmptyRepo(t)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := runInDirWithWriters(t, root, []string{"explain"}, &stdout, &stderr)
	if code != 4 {
		t.Fatalf("expected code 4, got %d", code)
	}
	code = runInDirWithWriters(t, root, []string{"explain", "nope"}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("expected code 1, got %d", code)
	}
}

func TestRunExplainJSON(t *testing.T) {
	root := setupEmptyRepo(t)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := runInDirWithWriters(t, root, []string{"explain", "git-status", "--format=json"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected success, got %d", code)
	}
	if !strings.Contains(stdout.String(), "git-status") {
		t.Fatalf("expected json explain output")
	}
}

func TestRunRespondVariants(t *testing.T) {
	root := setupEmptyRepo(t)
	response := filepath.Join(root, "response.json")
	if err := os.WriteFile(response, []byte(`{"status":"pass","signal":"ok"}`), 0644); err != nil {
		t.Fatalf("write response: %v", err)
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := runInDirWithWriters(t, root, []string{"respond", "--id", "check", "--response", response, "--format=llm"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected success, got %d", code)
	}
	if !strings.Contains(stdout.String(), "check:check") {
		t.Fatalf("expected llm output")
	}

	stdout.Reset()
	code = runInDirWithWriters(t, root, []string{"respond", "--id", "check", "--response", response, "--format=bad"}, &stdout, &stderr)
	if code != dun.ExitUsageError {
		t.Fatalf("expected code %d, got %d", dun.ExitUsageError, code)
	}
}

func TestRunRespondParseError(t *testing.T) {
	root := setupEmptyRepo(t)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := runInDirWithWriters(t, root, []string{"respond", "--badflag"}, &stdout, &stderr)
	if code != 4 {
		t.Fatalf("expected code 4, got %d", code)
	}
}

func TestRunRespondHandleError(t *testing.T) {
	root := setupEmptyRepo(t)
	orig := respondFn
	respondFn = func(_ string, _ io.Reader) (dun.CheckResult, error) {
		return dun.CheckResult{}, errors.New("boom")
	}
	t.Cleanup(func() { respondFn = orig })

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := runInDirWithWriters(t, root, []string{"respond", "--id", "check", "--response", "-"}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("expected code 1, got %d", code)
	}
}

func TestRunRespondJSONEncodeError(t *testing.T) {
	root := setupEmptyRepo(t)
	orig := respondFn
	respondFn = func(_ string, _ io.Reader) (dun.CheckResult, error) {
		return dun.CheckResult{ID: "check", Status: "pass", Signal: "ok"}, nil
	}
	t.Cleanup(func() { respondFn = orig })

	errWriter := &failWriter{err: errors.New("write failed")}
	var stderr bytes.Buffer
	code := runInDirWithWriters(t, root, []string{"respond", "--id", "check", "--response", "-", "--format=json"}, errWriter, &stderr)
	if code != 1 {
		t.Fatalf("expected code 1, got %d", code)
	}
}

func TestRunRespondErrors(t *testing.T) {
	root := setupEmptyRepo(t)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := runInDirWithWriters(t, root, []string{"respond"}, &stdout, &stderr)
	if code != 4 {
		t.Fatalf("expected code 4, got %d", code)
	}
	code = runInDirWithWriters(t, root, []string{"respond", "--id", "x", "--response", "missing.json"}, &stdout, &stderr)
	if code != dun.ExitRuntimeError {
		t.Fatalf("expected code %d, got %d", dun.ExitRuntimeError, code)
	}
}

func TestRunInstallParseError(t *testing.T) {
	root := setupEmptyRepo(t)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := runInDirWithWriters(t, root, []string{"install", "--badflag"}, &stdout, &stderr)
	if code != 4 {
		t.Fatalf("expected code 4, got %d", code)
	}
}

func TestRunInstallOutputsInstalled(t *testing.T) {
	root := setupEmptyRepo(t)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := runInDirWithWriters(t, root, []string{"install"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected success, got %d", code)
	}
	if !strings.Contains(stdout.String(), "installed:") {
		t.Fatalf("expected installed output")
	}
}

func TestRunInstallDryRunAndError(t *testing.T) {
	root := setupEmptyRepo(t)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := runInDirWithWriters(t, root, []string{"install", "--dry-run"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected success, got %d", code)
	}
	if !strings.Contains(stdout.String(), "plan:") {
		t.Fatalf("expected plan output")
	}

	badDir := t.TempDir()
	code = runInDirWithWriters(t, badDir, []string{"install"}, &stdout, &stderr)
	if code != dun.ExitRuntimeError {
		t.Fatalf("expected code %d, got %d", dun.ExitRuntimeError, code)
	}
}

func TestFindConfigFlag(t *testing.T) {
	if got := findConfigFlag([]string{"--config", "path.yaml"}); got != "path.yaml" {
		t.Fatalf("expected config path, got %q", got)
	}
	if got := findConfigFlag([]string{"--config=path.yaml"}); got != "path.yaml" {
		t.Fatalf("expected config path, got %q", got)
	}
	if got := findConfigFlag([]string{}); got != "" {
		t.Fatalf("expected empty config")
	}
	if got := findConfigFlag([]string{"--config"}); got != "" {
		t.Fatalf("expected empty config with missing value")
	}
}

func TestResolveRootFallback(t *testing.T) {
	dir := t.TempDir()
	if got := resolveRoot(dir); got != dir {
		t.Fatalf("expected fallback root")
	}
}

func TestFormatRules(t *testing.T) {
	rules := []dun.Rule{{Type: "path-exists", Path: "file.txt"}, {Type: "pattern-count", Pattern: "x", Expected: 2}}
	out := formatRules(rules)
	if !strings.Contains(out, "path-exists") || !strings.Contains(out, "pattern") {
		t.Fatalf("unexpected format output: %q", out)
	}
}

func TestPrintLLM(t *testing.T) {
	var stdout bytes.Buffer
	printLLM(&stdout, dun.Result{
		Checks: []dun.CheckResult{
			{ID: "id", Status: "pass", Signal: "ok", Detail: "detail", Next: "next", Issues: []dun.Issue{{Summary: "issue", Path: "file"}, {Summary: "loose"}}},
		},
	})
	text := stdout.String()
	if !strings.Contains(text, "check:id") || !strings.Contains(text, "issue: issue") {
		t.Fatalf("expected llm content")
	}
}

func runInDir(t *testing.T, dir string, args []string) []byte {
	t.Helper()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(cwd)
	})

	code := run(args, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("run failed (%d): %s", code, stderr.String())
	}
	return stdout.Bytes()
}

func runInDirWithWriters(t *testing.T, dir string, args []string, stdout io.Writer, stderr io.Writer) int {
	t.Helper()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(cwd)
	})
	return run(args, stdout, stderr)
}

func writeConfig(t *testing.T, root string, agentCmd string) {
	t.Helper()
	path := filepath.Join(root, ".dun", "config.yaml")
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("mkdir config dir: %v", err)
	}
	content := []byte("version: \"1\"\nagent:\n  automation: auto\n  mode: auto\n  timeout_ms: 5000\n  cmd: \"" + agentCmd + "\"\n")
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}
}

func setupRepoFromFixture(t *testing.T, name string) string {
	t.Helper()
	root := t.TempDir()
	if err := initGitRepo(root); err != nil {
		t.Fatalf("init git: %v", err)
	}
	src := fixturePath(t, filepath.Join("internal/testdata/repos", name))
	if err := copyDir(src, root); err != nil {
		t.Fatalf("copy fixture: %v", err)
	}
	return root
}

func setupEmptyRepo(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if err := initGitRepo(root); err != nil {
		t.Fatalf("init git: %v", err)
	}
	return root
}

func fixturePath(t *testing.T, rel string) string {
	t.Helper()
	root, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	repo := filepath.Dir(filepath.Dir(root))
	return filepath.Join(repo, rel)
}

func copyDir(src string, dst string) error {
	return filepath.WalkDir(src, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0644)
	})
}

func initGitRepo(path string) error {
	cmd := exec.Command("git", "init")
	cmd.Dir = path
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Run()
}

type failWriter struct {
	err error
}

func (w *failWriter) Write(_ []byte) (int, error) {
	return 0, w.err
}

func findCheck(t *testing.T, result dun.Result, id string) dun.CheckResult {
	t.Helper()
	for _, check := range result.Checks {
		if check.ID == id {
			return check
		}
	}
	t.Fatalf("check %s not found", id)
	return dun.CheckResult{}
}

// Tests for runIterate

func TestRunIterateParseError(t *testing.T) {
	root := setupEmptyRepo(t)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := runInDirWithWriters(t, root, []string{"iterate", "--badflag"}, &stdout, &stderr)
	if code != dun.ExitUsageError {
		t.Fatalf("expected code %d, got %d", dun.ExitUsageError, code)
	}
}

func TestRunIterateConfigError(t *testing.T) {
	root := setupEmptyRepo(t)
	cfgPath := filepath.Join(root, ".dun", "config.yaml")
	if err := os.MkdirAll(filepath.Dir(cfgPath), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(cfgPath, []byte("agent:\n  cmd: ["), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := runInDirWithWriters(t, root, []string{"iterate"}, &stdout, &stderr)
	if code != dun.ExitConfigError {
		t.Fatalf("expected code %d, got %d", dun.ExitConfigError, code)
	}
}

func TestRunIterateCheckError(t *testing.T) {
	root := setupEmptyRepo(t)
	orig := checkRepo
	checkRepo = func(_ string, _ dun.Options) (dun.Result, error) {
		return dun.Result{}, errors.New("boom")
	}
	t.Cleanup(func() { checkRepo = orig })

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := runInDirWithWriters(t, root, []string{"iterate"}, &stdout, &stderr)
	if code != dun.ExitCheckFailed {
		t.Fatalf("expected code %d, got %d", dun.ExitCheckFailed, code)
	}
}

func TestRunIterateAllPass(t *testing.T) {
	root := setupEmptyRepo(t)
	orig := checkRepo
	checkRepo = func(_ string, _ dun.Options) (dun.Result, error) {
		return dun.Result{
			Checks: []dun.CheckResult{
				{ID: "test", Status: "pass", Signal: "ok"},
			},
		}, nil
	}
	t.Cleanup(func() { checkRepo = orig })

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := runInDirWithWriters(t, root, []string{"iterate"}, &stdout, &stderr)
	if code != dun.ExitSuccess {
		t.Fatalf("expected code %d, got %d", dun.ExitSuccess, code)
	}
	output := stdout.String()
	if !strings.Contains(output, "STATUS: ALL_PASS") {
		t.Fatalf("expected ALL_PASS status in output")
	}
	if !strings.Contains(output, "EXIT_SIGNAL: true") {
		t.Fatalf("expected EXIT_SIGNAL in output")
	}
}

func TestRunIterateWithActionable(t *testing.T) {
	root := setupEmptyRepo(t)
	orig := checkRepo
	checkRepo = func(_ string, _ dun.Options) (dun.Result, error) {
		return dun.Result{
			Checks: []dun.CheckResult{
				{ID: "pass-check", Status: "pass", Signal: "ok"},
				{ID: "fail-check", Status: "fail", Signal: "failed", Detail: "something failed", Next: "fix it"},
				{ID: "warn-check", Status: "warn", Signal: "warning"},
				{ID: "error-check", Status: "error", Signal: "error"},
				{ID: "skip-check", Status: "skip", Signal: "skipped"},
			},
		}, nil
	}
	t.Cleanup(func() { checkRepo = orig })

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := runInDirWithWriters(t, root, []string{"iterate", "--automation", "yolo"}, &stdout, &stderr)
	if code != dun.ExitSuccess {
		t.Fatalf("expected code %d, got %d", dun.ExitSuccess, code)
	}
	output := stdout.String()
	// Should have iteration prompt, not all pass
	if strings.Contains(output, "STATUS: ALL_PASS") {
		t.Fatalf("should not have ALL_PASS when actionable items exist")
	}
	// Should list the actionable checks
	if !strings.Contains(output, "fail-check") {
		t.Fatalf("expected fail-check in output")
	}
	if !strings.Contains(output, "[HIGH]") {
		t.Fatalf("expected HIGH priority for error status")
	}
	if !strings.Contains(output, "[LOW]") {
		t.Fatalf("expected LOW priority for skip status")
	}
}

// Tests for runLoop

func TestRunLoopParseError(t *testing.T) {
	root := setupEmptyRepo(t)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := runInDirWithWriters(t, root, []string{"loop", "--badflag"}, &stdout, &stderr)
	if code != dun.ExitUsageError {
		t.Fatalf("expected code %d, got %d", dun.ExitUsageError, code)
	}
}

func TestRunLoopConfigError(t *testing.T) {
	root := setupEmptyRepo(t)
	cfgPath := filepath.Join(root, ".dun", "config.yaml")
	if err := os.MkdirAll(filepath.Dir(cfgPath), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(cfgPath, []byte("agent:\n  cmd: ["), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := runInDirWithWriters(t, root, []string{"loop"}, &stdout, &stderr)
	if code != dun.ExitConfigError {
		t.Fatalf("expected code %d, got %d", dun.ExitConfigError, code)
	}
}

func TestRunLoopDryRun(t *testing.T) {
	root := setupEmptyRepo(t)
	orig := checkRepo
	checkRepo = func(_ string, _ dun.Options) (dun.Result, error) {
		return dun.Result{
			Checks: []dun.CheckResult{
				{ID: "fail-check", Status: "fail", Signal: "failed"},
			},
		}, nil
	}
	t.Cleanup(func() { checkRepo = orig })

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := runInDirWithWriters(t, root, []string{"loop", "--dry-run", "--max-iterations", "1"}, &stdout, &stderr)
	if code != dun.ExitSuccess {
		t.Fatalf("expected code %d, got %d", dun.ExitSuccess, code)
	}
	output := stdout.String()
	if !strings.Contains(output, "DRY RUN") {
		t.Fatalf("expected DRY RUN in output")
	}
}

func TestRunLoopAllPass(t *testing.T) {
	root := setupEmptyRepo(t)
	orig := checkRepo
	checkRepo = func(_ string, _ dun.Options) (dun.Result, error) {
		return dun.Result{
			Checks: []dun.CheckResult{
				{ID: "pass-check", Status: "pass", Signal: "ok"},
			},
		}, nil
	}
	t.Cleanup(func() { checkRepo = orig })

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := runInDirWithWriters(t, root, []string{"loop", "--max-iterations", "1"}, &stdout, &stderr)
	if code != dun.ExitSuccess {
		t.Fatalf("expected code %d, got %d", dun.ExitSuccess, code)
	}
	output := stdout.String()
	if !strings.Contains(output, "All checks pass") {
		t.Fatalf("expected all pass message")
	}
}

func TestRunLoopCheckError(t *testing.T) {
	root := setupEmptyRepo(t)
	orig := checkRepo
	checkRepo = func(_ string, _ dun.Options) (dun.Result, error) {
		return dun.Result{}, errors.New("boom")
	}
	t.Cleanup(func() { checkRepo = orig })

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := runInDirWithWriters(t, root, []string{"loop", "--max-iterations", "1"}, &stdout, &stderr)
	if code != dun.ExitCheckFailed {
		t.Fatalf("expected code %d, got %d", dun.ExitCheckFailed, code)
	}
}

func TestRunLoopWithConfig(t *testing.T) {
	root := setupEmptyRepo(t)
	cfgPath := filepath.Join(root, ".dun", "config.yaml")
	if err := os.MkdirAll(filepath.Dir(cfgPath), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(cfgPath, []byte("version: \"1\"\nagent:\n  mode: auto\n"), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	orig := checkRepo
	checkRepo = func(_ string, _ dun.Options) (dun.Result, error) {
		return dun.Result{
			Checks: []dun.CheckResult{
				{ID: "pass-check", Status: "pass", Signal: "ok"},
			},
		}, nil
	}
	t.Cleanup(func() { checkRepo = orig })

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := runInDirWithWriters(t, root, []string{"loop", "--max-iterations", "1"}, &stdout, &stderr)
	if code != dun.ExitSuccess {
		t.Fatalf("expected code %d, got %d: %s", dun.ExitSuccess, code, stderr.String())
	}
}

func TestRunLoopMaxIterations(t *testing.T) {
	root := setupEmptyRepo(t)
	callCount := 0
	origCheck := checkRepo
	checkRepo = func(_ string, _ dun.Options) (dun.Result, error) {
		return dun.Result{
			Checks: []dun.CheckResult{
				{ID: "fail-check", Status: "fail", Signal: "failed"},
			},
		}, nil
	}
	t.Cleanup(func() { checkRepo = origCheck })

	origHarness := callHarnessFn
	callHarnessFn = func(harness, prompt, automation string) (string, error) {
		callCount++
		return "no exit signal", nil
	}
	t.Cleanup(func() { callHarnessFn = origHarness })

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := runInDirWithWriters(t, root, []string{"loop", "--max-iterations", "2"}, &stdout, &stderr)
	if code != dun.ExitSuccess {
		t.Fatalf("expected code %d, got %d", dun.ExitSuccess, code)
	}
	if callCount != 2 {
		t.Fatalf("expected 2 harness calls, got %d", callCount)
	}
	if !strings.Contains(stdout.String(), "Max iterations (2) reached") {
		t.Fatalf("expected max iterations message")
	}
}

func TestRunLoopExitSignal(t *testing.T) {
	root := setupEmptyRepo(t)
	origCheck := checkRepo
	checkRepo = func(_ string, _ dun.Options) (dun.Result, error) {
		return dun.Result{
			Checks: []dun.CheckResult{
				{ID: "fail-check", Status: "fail", Signal: "failed"},
			},
		}, nil
	}
	t.Cleanup(func() { checkRepo = origCheck })

	origHarness := callHarnessFn
	callHarnessFn = func(harness, prompt, automation string) (string, error) {
		return "---DUN_STATUS---\nEXIT_SIGNAL: true\n---END_DUN_STATUS---", nil
	}
	t.Cleanup(func() { callHarnessFn = origHarness })

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := runInDirWithWriters(t, root, []string{"loop", "--max-iterations", "10"}, &stdout, &stderr)
	if code != dun.ExitSuccess {
		t.Fatalf("expected code %d, got %d", dun.ExitSuccess, code)
	}
	if !strings.Contains(stdout.String(), "Exit signal received") {
		t.Fatalf("expected exit signal message")
	}
}

func TestRunLoopHarnessError(t *testing.T) {
	root := setupEmptyRepo(t)
	callCount := 0
	origCheck := checkRepo
	checkRepo = func(_ string, _ dun.Options) (dun.Result, error) {
		callCount++
		if callCount > 2 {
			// Return all pass to exit loop
			return dun.Result{
				Checks: []dun.CheckResult{
					{ID: "pass-check", Status: "pass", Signal: "ok"},
				},
			}, nil
		}
		return dun.Result{
			Checks: []dun.CheckResult{
				{ID: "fail-check", Status: "fail", Signal: "failed"},
			},
		}, nil
	}
	t.Cleanup(func() { checkRepo = origCheck })

	origHarness := callHarnessFn
	callHarnessFn = func(harness, prompt, automation string) (string, error) {
		return "", errors.New("harness failed")
	}
	t.Cleanup(func() { callHarnessFn = origHarness })

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := runInDirWithWriters(t, root, []string{"loop", "--max-iterations", "10"}, &stdout, &stderr)
	if code != dun.ExitSuccess {
		t.Fatalf("expected code %d, got %d", dun.ExitSuccess, code)
	}
	// Should continue despite harness error and eventually reach all pass
	if !strings.Contains(stderr.String(), "harness call failed") {
		t.Fatalf("expected harness error message in stderr")
	}
}

// Tests for callHarness

func TestCallHarnessUnknown(t *testing.T) {
	_, err := callHarness("unknown", "prompt", "auto")
	if err == nil {
		t.Fatalf("expected error for unknown harness")
	}
	if !strings.Contains(err.Error(), "unknown harness") {
		t.Fatalf("expected unknown harness error, got: %v", err)
	}
}

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

// Tests for printIteratePrompt

func TestPrintIteratePromptVariants(t *testing.T) {
	checks := []dun.CheckResult{
		{
			ID:     "error-check",
			Status: "error",
			Signal: "error signal",
			Detail: "error detail",
			Next:   "fix error",
			Issues: []dun.Issue{
				{Summary: "issue1", Path: "file1.go"},
				{Summary: "issue2"}, // no path
			},
			Prompt: &dun.PromptEnvelope{},
		},
		{
			ID:     "skip-check",
			Status: "skip",
			Signal: "skip signal",
		},
		{
			ID:     "warn-check",
			Status: "warn",
			Signal: "warn signal",
		},
	}

	var buf bytes.Buffer
	printIteratePrompt(&buf, checks, "yolo", "/test/root")
	output := buf.String()

	// Check header
	if !strings.Contains(output, "# Dun Iteration") {
		t.Fatalf("expected header")
	}
	if !strings.Contains(output, "You are working in: /test/root") {
		t.Fatalf("expected working directory")
	}
	if !strings.Contains(output, "Automation mode: yolo") {
		t.Fatalf("expected automation mode")
	}

	// Check priority labels
	if !strings.Contains(output, "[HIGH]") {
		t.Fatalf("expected HIGH priority for error")
	}
	if !strings.Contains(output, "[LOW]") {
		t.Fatalf("expected LOW priority for skip")
	}
	if !strings.Contains(output, "[MEDIUM]") {
		t.Fatalf("expected MEDIUM priority for warn")
	}

	// Check issue formatting
	if !strings.Contains(output, "issue1 (file1.go)") {
		t.Fatalf("expected issue with path")
	}
	if !strings.Contains(output, "- issue2\n") {
		t.Fatalf("expected issue without path")
	}

	// Check prompt indicator
	if !strings.Contains(output, "Prompt available:") {
		t.Fatalf("expected prompt available note")
	}

	// Check instructions section
	if !strings.Contains(output, "## Instructions") {
		t.Fatalf("expected instructions section")
	}
	if !strings.Contains(output, "---DUN_STATUS---") {
		t.Fatalf("expected status block template")
	}
}

// Tests for help command coverage (AC-8)

func TestRunHelpIncludesIterate(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{"help"}, &stdout, &stderr)
	if code != dun.ExitSuccess {
		t.Fatalf("expected success, got %d", code)
	}
	output := stdout.String()
	if !strings.Contains(output, "iterate") {
		t.Fatalf("help should document iterate command")
	}
	if !strings.Contains(output, "dun iterate") {
		t.Fatalf("help should show iterate usage")
	}
}

func TestRunHelpIncludesLoop(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{"help"}, &stdout, &stderr)
	if code != dun.ExitSuccess {
		t.Fatalf("expected success, got %d", code)
	}
	output := stdout.String()
	if !strings.Contains(output, "loop") {
		t.Fatalf("help should document loop command")
	}
	if !strings.Contains(output, "--harness") {
		t.Fatalf("help should document harness option")
	}
	if !strings.Contains(output, "--max-iterations") {
		t.Fatalf("help should document max-iterations option")
	}
	if !strings.Contains(output, "claude, gemini, codex") {
		t.Fatalf("help should list available harnesses")
	}
}

func TestRunHelpIncludesExamples(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{"help"}, &stdout, &stderr)
	if code != dun.ExitSuccess {
		t.Fatalf("expected success, got %d", code)
	}
	output := stdout.String()
	if !strings.Contains(output, "dun loop") {
		t.Fatalf("help should include loop examples")
	}
	if !strings.Contains(output, "--dry-run") {
		t.Fatalf("help should document dry-run option")
	}
}

// Tests for AC-4: Deterministic Output

// TC-006: Deterministic Output - verify same input produces same output
func TestOutputDeterminism(t *testing.T) {
	root := setupEmptyRepo(t)

	// Run check multiple times
	outputs := make([]string, 3)
	for i := 0; i < 3; i++ {
		var stdout bytes.Buffer
		var stderr bytes.Buffer
		code := runInDirWithWriters(t, root, []string{"check", "--format=json"}, &stdout, &stderr)
		if code != 0 {
			t.Fatalf("run %d: expected success, got %d: %s", i, code, stderr.String())
		}
		outputs[i] = stdout.String()
	}

	// All outputs should be identical
	for i := 1; i < len(outputs); i++ {
		if outputs[i] != outputs[0] {
			t.Fatalf("output %d differs from output 0:\n--- output 0 ---\n%s\n--- output %d ---\n%s",
				i, outputs[0], i, outputs[i])
		}
	}
}

// TC-007: Check Ordering Consistency - verify check order is stable across runs
func TestCheckOrderingConsistency(t *testing.T) {
	root := setupEmptyRepo(t)

	// Run multiple times and verify order
	var prevOrder []string
	for i := 0; i < 3; i++ {
		var stdout bytes.Buffer
		var stderr bytes.Buffer
		code := runInDirWithWriters(t, root, []string{"check", "--format=json"}, &stdout, &stderr)
		if code != 0 {
			t.Fatalf("run %d: expected success, got %d: %s", i, code, stderr.String())
		}

		var result dun.Result
		if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
			t.Fatalf("decode run %d: %v", i, err)
		}

		var order []string
		for _, check := range result.Checks {
			order = append(order, check.ID)
		}

		if prevOrder != nil {
			if len(order) != len(prevOrder) {
				t.Fatalf("check count changed: %d vs %d", len(prevOrder), len(order))
			}
			for j, id := range order {
				if prevOrder[j] != id {
					t.Fatalf("check order changed at position %d: %s vs %s", j, prevOrder[j], id)
				}
			}
		}
		prevOrder = order
	}
}

// TestOutputDeterminismWithFixture tests determinism with a more complex fixture
func TestOutputDeterminismWithFixture(t *testing.T) {
	root := setupRepoFromFixture(t, "helix-alignment")
	agentCmd := "bash " + fixturePath(t, "internal/testdata/agent/agent.sh")
	writeConfig(t, root, agentCmd)

	// Run check multiple times
	outputs := make([]string, 3)
	for i := 0; i < 3; i++ {
		var stdout bytes.Buffer
		var stderr bytes.Buffer
		code := runInDirWithWriters(t, root, []string{"check", "--format=json"}, &stdout, &stderr)
		if code != 0 {
			t.Fatalf("run %d: expected success, got %d: %s", i, code, stderr.String())
		}
		outputs[i] = stdout.String()
	}

	// All outputs should be identical
	for i := 1; i < len(outputs); i++ {
		if outputs[i] != outputs[0] {
			t.Fatalf("output %d differs from output 0:\n--- output 0 ---\n%s\n--- output %d ---\n%s",
				i, outputs[0], i, outputs[i])
		}
	}
}

// TestIssueOrderingConsistency verifies issues within checks maintain stable ordering
func TestIssueOrderingConsistency(t *testing.T) {
	orig := checkRepo
	checkRepo = func(_ string, _ dun.Options) (dun.Result, error) {
		return dun.Result{
			Checks: []dun.CheckResult{
				{
					ID:     "check-with-issues",
					Status: "fail",
					Signal: "failed",
					Issues: []dun.Issue{
						{Summary: "Issue A", Path: "a.go"},
						{Summary: "Issue B", Path: "b.go"},
						{Summary: "Issue C", Path: "c.go"},
					},
				},
			},
		}, nil
	}
	t.Cleanup(func() { checkRepo = orig })

	root := setupEmptyRepo(t)

	// Run multiple times and verify issue order
	for i := 0; i < 3; i++ {
		var stdout bytes.Buffer
		var stderr bytes.Buffer
		code := runInDirWithWriters(t, root, []string{"check", "--format=json"}, &stdout, &stderr)
		if code != 0 {
			t.Fatalf("run %d: expected success, got %d", i, code)
		}

		var result dun.Result
		if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
			t.Fatalf("decode run %d: %v", i, err)
		}

		if len(result.Checks) != 1 {
			t.Fatalf("expected 1 check, got %d", len(result.Checks))
		}

		issues := result.Checks[0].Issues
		if len(issues) != 3 {
			t.Fatalf("expected 3 issues, got %d", len(issues))
		}

		// Verify order is always A, B, C
		expected := []string{"Issue A", "Issue B", "Issue C"}
		for j, issue := range issues {
			if issue.Summary != expected[j] {
				t.Fatalf("run %d: issue order changed at position %d: expected %q, got %q",
					i, j, expected[j], issue.Summary)
			}
		}
	}
}

// TestNoTimestampsInOutput verifies output contains no non-deterministic fields like timestamps
func TestNoTimestampsInOutput(t *testing.T) {
	root := setupEmptyRepo(t)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := runInDirWithWriters(t, root, []string{"check", "--format=json"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected success, got %d", code)
	}

	output := stdout.String()

	// These patterns would indicate non-deterministic fields
	nonDeterministic := []string{
		"\"timestamp\"",
		"\"time\"",
		"\"duration\"",
		"\"elapsed\"",
		"\"created_at\"",
		"\"updated_at\"",
	}

	for _, pattern := range nonDeterministic {
		if strings.Contains(output, pattern) {
			t.Fatalf("output contains non-deterministic field %q which would break determinism", pattern)
		}
	}
}

// Tests for runVersion command

func TestRunVersionBasic(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{"version"}, &stdout, &stderr)
	if code != dun.ExitSuccess {
		t.Fatalf("expected success, got %d", code)
	}
	output := stdout.String()
	// Should contain version info
	if !strings.Contains(output, "dun") || !strings.Contains(output, "dev") {
		t.Fatalf("expected version output, got: %s", output)
	}
}

func TestRunVersionJSON(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{"version", "--json"}, &stdout, &stderr)
	if code != dun.ExitSuccess {
		t.Fatalf("expected success, got %d", code)
	}
	output := stdout.String()
	// Should be valid JSON with version field
	var result map[string]string
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("expected valid JSON: %v", err)
	}
	if _, ok := result["version"]; !ok {
		t.Fatalf("expected version field in JSON output")
	}
}

func TestRunVersionParseError(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{"version", "--badflag"}, &stdout, &stderr)
	if code != dun.ExitUsageError {
		t.Fatalf("expected code %d, got %d", dun.ExitUsageError, code)
	}
}

func TestRunVersionJSONWriteError(t *testing.T) {
	errWriter := &failWriter{err: errors.New("write failed")}
	var stderr bytes.Buffer
	code := run([]string{"version", "--json"}, errWriter, &stderr)
	if code != dun.ExitRuntimeError {
		t.Fatalf("expected code %d, got %d", dun.ExitRuntimeError, code)
	}
}

func TestRunVersionCheck(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	// The --check flag tries to check for updates, may fail due to network
	code := run([]string{"version", "--check"}, &stdout, &stderr)
	// Either succeeds or fails with runtime error (network issue)
	if code != dun.ExitSuccess && code != dun.ExitRuntimeError {
		t.Fatalf("expected success or runtime error, got %d", code)
	}
	// Should have version info at least
	output := stdout.String()
	if !strings.Contains(output, "dun") {
		t.Fatalf("expected version output, got: %s", output)
	}
}

// Tests for runUpdate command

func TestRunUpdateParseError(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{"update", "--badflag"}, &stdout, &stderr)
	if code != dun.ExitUsageError {
		t.Fatalf("expected code %d, got %d", dun.ExitUsageError, code)
	}
}

func TestRunUpdateDryRun(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	// Dry run mode should output plan without applying
	code := run([]string{"update", "--dry-run"}, &stdout, &stderr)
	// May succeed or fail depending on network - just verify it runs
	if code != dun.ExitSuccess && code != dun.ExitRuntimeError {
		t.Fatalf("expected success or runtime error, got %d", code)
	}
}

// Tests for quorumStrategyName helper

func TestQuorumStrategyName(t *testing.T) {
	tests := []struct {
		cfg  dun.QuorumConfig
		want string
	}{
		{dun.QuorumConfig{Strategy: "majority"}, "majority"},
		{dun.QuorumConfig{Threshold: 3}, "3"},
		{dun.QuorumConfig{}, "default"},
		{dun.QuorumConfig{Strategy: "custom", Threshold: 2}, "custom"}, // Strategy takes precedence
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := quorumStrategyName(tt.cfg)
			if got != tt.want {
				t.Errorf("quorumStrategyName(%+v) = %q, want %q", tt.cfg, got, tt.want)
			}
		})
	}
}

// Tests for runQuorum function

func TestRunQuorumNoHarnesses(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cfg := dun.QuorumConfig{Harnesses: []string{}}
	_, err := runQuorum(cfg, "test prompt", "auto", &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error for no harnesses")
	}
	if !strings.Contains(err.Error(), "no harnesses configured") {
		t.Fatalf("expected 'no harnesses' error, got: %v", err)
	}
}

func TestRunQuorumSequential(t *testing.T) {
	// Mock harness calls
	origHarness := callHarnessFn
	callCount := 0
	callHarnessFn = func(harness, prompt, automation string) (string, error) {
		callCount++
		return "mock response", nil
	}
	defer func() { callHarnessFn = origHarness }()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cfg := dun.QuorumConfig{
		Harnesses: []string{"mock1", "mock2"},
		Mode:      "sequential",
		Strategy:  "majority",
	}

	response, err := runQuorum(cfg, "test prompt", "auto", &stdout, &stderr)
	if err != nil {
		t.Fatalf("runQuorum failed: %v", err)
	}
	if callCount != 2 {
		t.Fatalf("expected 2 harness calls, got %d", callCount)
	}
	if response == "" {
		t.Fatal("expected non-empty response")
	}
	if !strings.Contains(stdout.String(), "sequentially") {
		t.Fatalf("expected sequential message in output")
	}
}

func TestRunQuorumParallel(t *testing.T) {
	// Mock harness calls
	origHarness := callHarnessFn
	callCount := 0
	callHarnessFn = func(harness, prompt, automation string) (string, error) {
		callCount++
		return "mock response", nil
	}
	defer func() { callHarnessFn = origHarness }()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cfg := dun.QuorumConfig{
		Harnesses: []string{"mock1", "mock2"},
		Mode:      "parallel",
		Strategy:  "majority",
	}

	response, err := runQuorum(cfg, "test prompt", "auto", &stdout, &stderr)
	if err != nil {
		t.Fatalf("runQuorum failed: %v", err)
	}
	if callCount != 2 {
		t.Fatalf("expected 2 harness calls, got %d", callCount)
	}
	if response == "" {
		t.Fatal("expected non-empty response")
	}
	if !strings.Contains(stdout.String(), "parallel") {
		t.Fatalf("expected parallel message in output")
	}
}

func TestRunQuorumWithErrors(t *testing.T) {
	// Mock harness calls - first one fails, but quorum still met (2/3 succeed)
	origHarness := callHarnessFn
	callCount := 0
	callHarnessFn = func(harness, prompt, automation string) (string, error) {
		callCount++
		if callCount == 1 {
			return "", errors.New("harness error")
		}
		return "success response", nil
	}
	defer func() { callHarnessFn = origHarness }()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cfg := dun.QuorumConfig{
		Harnesses: []string{"mock1", "mock2", "mock3"}, // 3 harnesses, 2 succeed = majority met
		Mode:      "sequential",
		Strategy:  "majority",
	}

	response, err := runQuorum(cfg, "test prompt", "auto", &stdout, &stderr)
	if err != nil {
		t.Fatalf("runQuorum failed: %v", err)
	}
	// Should still get a response from the successful harness
	if response == "" {
		t.Fatal("expected non-empty response despite one failure")
	}
	// Should have error message in stderr
	if !strings.Contains(stderr.String(), "failed") {
		t.Fatalf("expected error message in stderr")
	}
}

func TestRunQuorumQuorumNotMet(t *testing.T) {
	// Mock harness calls - only 1/2 succeed, quorum not met
	origHarness := callHarnessFn
	callCount := 0
	callHarnessFn = func(harness, prompt, automation string) (string, error) {
		callCount++
		if callCount == 1 {
			return "", errors.New("harness error")
		}
		return "success response", nil
	}
	defer func() { callHarnessFn = origHarness }()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cfg := dun.QuorumConfig{
		Harnesses: []string{"mock1", "mock2"},
		Mode:      "sequential",
		Strategy:  "majority",
	}

	_, err := runQuorum(cfg, "test prompt", "auto", &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error when quorum not met")
	}
	if !strings.Contains(err.Error(), "quorum not met") {
		t.Fatalf("expected 'quorum not met' error, got: %v", err)
	}
}

func TestRunQuorumAllFail(t *testing.T) {
	// Mock harness calls - all fail
	origHarness := callHarnessFn
	callHarnessFn = func(harness, prompt, automation string) (string, error) {
		return "", errors.New("harness error")
	}
	defer func() { callHarnessFn = origHarness }()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cfg := dun.QuorumConfig{
		Harnesses: []string{"mock1", "mock2"},
		Mode:      "sequential",
		Strategy:  "majority",
	}

	_, err := runQuorum(cfg, "test prompt", "auto", &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error when all harnesses fail")
	}
	if !strings.Contains(err.Error(), "all harnesses failed") {
		t.Fatalf("expected 'all harnesses failed' error, got: %v", err)
	}
}

func TestRunQuorumConflict(t *testing.T) {
	// Mock harness calls - conflicting exit signals
	origHarness := callHarnessFn
	callCount := 0
	callHarnessFn = func(harness, prompt, automation string) (string, error) {
		callCount++
		if callCount == 1 {
			return "---DUN_STATUS---\nEXIT_SIGNAL: true\n---END_DUN_STATUS---", nil
		}
		return "no exit signal", nil // Conflict: different response
	}
	defer func() { callHarnessFn = origHarness }()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cfg := dun.QuorumConfig{
		Harnesses: []string{"mock1", "mock2"},
		Mode:      "sequential",
		Strategy:  "any", // Any is always met
	}

	_, err := runQuorum(cfg, "test prompt", "auto", &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error when conflict detected")
	}
	if !strings.Contains(stdout.String(), "Conflict detected") {
		t.Fatalf("expected conflict message in stdout")
	}
}

func TestRunQuorumConflictWithPrefer(t *testing.T) {
	// Mock harness calls - conflicting exit signals, but prefer specified
	origHarness := callHarnessFn
	callHarnessFn = func(harness, prompt, automation string) (string, error) {
		if harness == "preferred" {
			return "preferred response with EXIT_SIGNAL: true", nil
		}
		return "no exit signal", nil // Conflict
	}
	defer func() { callHarnessFn = origHarness }()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cfg := dun.QuorumConfig{
		Harnesses: []string{"other", "preferred"},
		Mode:      "sequential",
		Strategy:  "any",
		Prefer:    "preferred",
	}

	response, err := runQuorum(cfg, "test prompt", "auto", &stdout, &stderr)
	if err != nil {
		t.Fatalf("runQuorum failed: %v", err)
	}
	if !strings.Contains(response, "preferred response") {
		t.Fatalf("expected preferred response, got: %s", response)
	}
}

func TestRunQuorumConflictWithEscalate(t *testing.T) {
	// Mock harness calls - conflicting exit signals, escalate to human
	origHarness := callHarnessFn
	callCount := 0
	callHarnessFn = func(harness, prompt, automation string) (string, error) {
		callCount++
		if callCount == 1 {
			return "---DUN_STATUS---\nEXIT_SIGNAL: true\n---END_DUN_STATUS---", nil
		}
		return "no exit signal", nil
	}
	defer func() { callHarnessFn = origHarness }()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cfg := dun.QuorumConfig{
		Harnesses: []string{"mock1", "mock2"},
		Mode:      "sequential",
		Strategy:  "any",
		Escalate:  true,
	}

	_, err := runQuorum(cfg, "test prompt", "auto", &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error when escalating")
	}
	if !strings.Contains(stderr.String(), "Escalating") {
		t.Fatalf("expected escalation message in stderr")
	}
}

func TestRunQuorumPreferredHarnessNoConflict(t *testing.T) {
	// Mock harness calls - no conflict, but prefer specified
	origHarness := callHarnessFn
	callHarnessFn = func(harness, prompt, automation string) (string, error) {
		if harness == "preferred" {
			return "preferred response", nil
		}
		return "other response", nil
	}
	defer func() { callHarnessFn = origHarness }()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cfg := dun.QuorumConfig{
		Harnesses: []string{"other", "preferred"},
		Mode:      "sequential",
		Strategy:  "any",
		Prefer:    "preferred",
	}

	response, err := runQuorum(cfg, "test prompt", "auto", &stdout, &stderr)
	if err != nil {
		t.Fatalf("runQuorum failed: %v", err)
	}
	if response != "preferred response" {
		t.Fatalf("expected preferred response, got: %s", response)
	}
}
