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
