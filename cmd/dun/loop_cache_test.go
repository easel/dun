package main

import (
	"bytes"
	"testing"
	"time"

	"github.com/easel/dun/internal/dun"
)

func TestRunLoopQuorumUsesCachedHarnesses(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	cache := dun.HarnessCache{
		LastCheck: time.Now(),
		Harnesses: []dun.HarnessStatus{{Name: "codex", Command: "codex", Available: true, Live: true}},
	}
	if err := cache.Save(); err != nil {
		t.Fatalf("save cache: %v", err)
	}

	root := setupEmptyRepo(t)
	origCheck := checkRepo
	checkRepo = func(_ string, _ dun.Options) (dun.Result, error) {
		return dun.Result{Checks: []dun.CheckResult{{ID: "fail-check", Status: "fail", Signal: "fail"}}}, nil
	}
	t.Cleanup(func() { checkRepo = origCheck })

	origHarness := callHarnessFn
	var gotHarness string
	callHarnessFn = func(harness, _ string, _ string) (string, error) {
		gotHarness = harness
		return "EXIT_SIGNAL: true", nil
	}
	t.Cleanup(func() { callHarnessFn = origHarness })

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := runInDirWithWriters(t, root, []string{"loop", "--max-iterations", "1", "--quorum", "any"}, &stdout, &stderr)
	if code != dun.ExitSuccess {
		t.Fatalf("expected success, got %d: %s", code, stderr.String())
	}
	if gotHarness != "codex" {
		t.Fatalf("expected cached harness codex, got %q", gotHarness)
	}
}
