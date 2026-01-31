# TP-008: Git Hygiene Test Plan

**User Story:** US-008 - Keep Git Hygiene and Hook Checks
**Status:** Draft
**Last Updated:** 2026-01-30

## Acceptance Criteria

From US-008:

| ID | Criterion | Description |
|----|-----------|-------------|
| AC1 | Dirty working trees are detected via `git status --porcelain` | System must detect uncommitted changes |
| AC2 | Each dirty path appears as an issue with an actionable next step | Issues must list changed files with guidance |
| AC3 | Lefthook or pre-commit hooks run when configured and installed | Hooks execute if tool is available |
| AC4 | Missing hook tools produce a warning with a clear next step | Warning with install instructions |
| AC5 | If no hook configuration exists, hook checks are skipped | Skip gracefully when no config |

## Test Coverage Matrix

### AC1: Dirty Working Tree Detection

| Test Case | File | Function | Status |
|-----------|------|----------|--------|
| Dirty repo returns warn status | `git_checks_test.go` | `TestGitStatusCheckWarnsWhenDirty` | COVERED |
| Clean repo returns pass status | `git_checks_test.go` | `TestGitStatusCheckPassesWhenClean` | COVERED |
| Uses `git status --porcelain` | `git_checks.go:131` | `gitStatusLines` | COVERED (impl) |
| Handles git status errors | `git_checks_extra_test.go` | `TestRunGitStatusCheckError` | COVERED |
| Handles non-git directories | `git_checks_extra_test.go` | `TestGitStatusLinesError` | COVERED |

### AC2: Dirty Paths as Actionable Issues

| Test Case | File | Function | Status |
|-----------|------|----------|--------|
| Issues list contains dirty files | `git_checks_test.go` | `TestGitStatusCheckWarnsWhenDirty` | COVERED |
| Next step contains `git commit` | `git_checks_test.go` | `TestGitStatusCheckWarnsWhenDirty` | COVERED |
| Parses renamed files correctly | `git_checks_extra_test.go` | `TestParseGitStatusPath` | COVERED |
| Handles no changes | `git_checks_extra_test.go` | `TestCommitNextInstructionNoFiles` | COVERED |
| Handles files with spaces | `git_checks_extra_test.go` | `TestCommitNextInstructionQuotesSpaces` | COVERED |
| Handles many files (>10) | `git_checks_extra_test.go` | `TestCommitNextInstructionManyFiles` | COVERED |
| Skips empty paths | `git_checks_extra_test.go` | `TestRunGitStatusCheckSkipsEmptyAndDuplicate` | COVERED |
| Deduplicates paths | `git_checks_extra_test.go` | `TestRunGitStatusCheckSkipsEmptyAndDuplicate` | COVERED |
| Never suggests `git add -A` | `git_checks_extra_test.go` | `TestCommitNextInstructionWithFiles`, `TestCommitNextInstructionManyFiles` | COVERED |
| Warns against `--force` push | `git_checks_extra_test.go` | `TestCommitNextInstructionWithFiles` | COVERED |

### AC3: Hook Execution When Configured and Installed

| Test Case | File | Function | Status |
|-----------|------|----------|--------|
| Lefthook runs when present | `git_checks_test.go` | `TestHookCheckRunsWhenToolPresent` | COVERED |
| Lefthook passes (exit 0) | `git_checks_extra_test.go` | `TestRunHookCheckPasses` | COVERED |
| Lefthook fails (exit 1) | `git_checks_extra_test.go` | `TestHookCheckFailsWhenToolErrors` | COVERED |
| Detects lefthook.yml | `git_checks.go:104-113` | `detectHookTool` | COVERED (impl) |
| Detects .lefthook dir | `git_checks.go:104-105` | `detectHookTool` | **GAP** |
| Detects pre-commit config | `git_checks_extra_test.go` | `TestDetectHookToolPreCommit` | COVERED |
| Pre-commit runs when present | - | - | **GAP** |
| Pre-commit fails (exit 1) | - | - | **GAP** |

### AC4: Missing Hook Tool Warning

| Test Case | File | Function | Status |
|-----------|------|----------|--------|
| Lefthook config but tool missing | `git_checks_test.go` | `TestHookCheckWarnsWhenToolMissing` | COVERED |
| Warning contains tool name | `git_checks_test.go` | `TestHookCheckWarnsWhenToolMissing` | COVERED |
| Pre-commit config but tool missing | `git_checks_extra_test.go` | `TestDetectHookToolPreCommit` (partial) | PARTIAL |
| Install hint for lefthook | `git_checks.go:112` | - | COVERED (impl) |
| Install hint for pre-commit | `git_checks.go:123` | - | COVERED (impl) |
| Next step is install hint | `git_checks_extra_test.go` | `TestRunHookCheckWarnWhenToolMissing` | COVERED |

### AC5: Skip When No Hook Config

| Test Case | File | Function | Status |
|-----------|------|----------|--------|
| No hook config returns skip | `git_checks_extra_test.go` | `TestRunHookCheckSkip` | COVERED |
| No tool detected | `git_checks_extra_test.go` | `TestDetectHookToolNone` | COVERED |
| Signal indicates no config | `git_checks.go:63` | - | COVERED (impl) |

## Gap Analysis

### Missing Test Cases

1. **`.lefthook` directory detection** (AC3)
   - The implementation checks for `.lefthook` directory but no test verifies this path
   - Priority: LOW (alternative config location)

2. **Pre-commit hook execution** (AC3)
   - Tests exist for detection but not for actual execution
   - Priority: MEDIUM (symmetry with lefthook tests)

3. **Pre-commit hook failure** (AC3)
   - No test for pre-commit returning non-zero exit
   - Priority: MEDIUM (error path coverage)

4. **Pre-commit tool missing warning** (AC4)
   - Detection test exists but doesn't verify warn status when tool missing
   - Priority: MEDIUM (matches lefthook warning test)

5. **Integration test via CLI** (All AC)
   - `main_test.go` has no specific git-hygiene integration tests
   - Priority: LOW (unit tests provide adequate coverage)

6. **Output format verification** (AC2)
   - No test verifies the exact format of the `Next` instruction for different file counts
   - Priority: LOW (behavior tested, format is implementation detail)

## Proposed Test Cases

### High Priority

None - all critical paths are covered.

### Medium Priority

```go
// git_checks_extra_test.go

func TestDetectHookToolLefthookDir(t *testing.T) {
    root := tempGitRepo(t)
    if err := os.MkdirAll(filepath.Join(root, ".lefthook"), 0755); err != nil {
        t.Fatalf("mkdir .lefthook: %v", err)
    }
    t.Setenv("PATH", "")
    tool, err := detectHookTool(root)
    if err != nil {
        t.Fatalf("detect hook tool: %v", err)
    }
    if tool.Name != "lefthook" {
        t.Fatalf("expected lefthook, got %q", tool.Name)
    }
}

func TestPreCommitHookRunsWhenToolPresent(t *testing.T) {
    root := tempGitRepo(t)
    writeFile(t, filepath.Join(root, ".pre-commit-config.yaml"), "repos: []")

    binDir := t.TempDir()
    toolPath := filepath.Join(binDir, "pre-commit")
    script := "#!/bin/sh\nexit 0\n"
    writeFile(t, toolPath, script)
    if err := os.Chmod(toolPath, 0755); err != nil {
        t.Fatalf("chmod pre-commit: %v", err)
    }

    origPath := os.Getenv("PATH")
    t.Setenv("PATH", binDir+string(os.PathListSeparator)+origPath)

    res, err := runHookCheck(root, Check{ID: "git-hooks"})
    if err != nil {
        t.Fatalf("hook check: %v", err)
    }
    if res.Status != "pass" {
        t.Fatalf("expected pass, got %s", res.Status)
    }
    if !strings.Contains(res.Signal, "pre-commit") {
        t.Fatalf("expected pre-commit in signal, got %q", res.Signal)
    }
}

func TestPreCommitHookFailsWhenToolErrors(t *testing.T) {
    root := tempGitRepo(t)
    writeFile(t, filepath.Join(root, ".pre-commit-config.yaml"), "repos: []")

    binDir := t.TempDir()
    toolPath := filepath.Join(binDir, "pre-commit")
    writeFile(t, toolPath, "#!/bin/sh\nexit 1\n")
    if err := os.Chmod(toolPath, 0755); err != nil {
        t.Fatalf("chmod pre-commit: %v", err)
    }
    origPath := os.Getenv("PATH")
    t.Setenv("PATH", binDir+string(os.PathListSeparator)+origPath)

    res, err := runHookCheck(root, Check{ID: "git-hooks"})
    if err != nil {
        t.Fatalf("hook check: %v", err)
    }
    if res.Status != "fail" {
        t.Fatalf("expected fail, got %s", res.Status)
    }
}

func TestPreCommitMissingToolWarning(t *testing.T) {
    root := tempGitRepo(t)
    writeFile(t, filepath.Join(root, ".pre-commit-config.yaml"), "repos: []")
    t.Setenv("PATH", "")

    res, err := runHookCheck(root, Check{ID: "git-hooks"})
    if err != nil {
        t.Fatalf("hook check: %v", err)
    }
    if res.Status != "warn" {
        t.Fatalf("expected warn, got %s", res.Status)
    }
    if !strings.Contains(res.Detail, "pre-commit") {
        t.Fatalf("expected pre-commit in detail, got %q", res.Detail)
    }
    if !strings.Contains(res.Next, "pre-commit.com") {
        t.Fatalf("expected install hint, got %q", res.Next)
    }
}
```

### Low Priority

```go
// git_checks_extra_test.go

func TestDetectHookToolLefthookPrecedence(t *testing.T) {
    // Verify lefthook takes precedence when both configs exist
    root := tempGitRepo(t)
    writeFile(t, filepath.Join(root, "lefthook.yml"), "pre-commit: {}")
    writeFile(t, filepath.Join(root, ".pre-commit-config.yaml"), "repos: []")
    t.Setenv("PATH", "")

    tool, err := detectHookTool(root)
    if err != nil {
        t.Fatalf("detect hook tool: %v", err)
    }
    if tool.Name != "lefthook" {
        t.Fatalf("expected lefthook precedence, got %q", tool.Name)
    }
}
```

## Test Execution

Run all git hygiene tests:

```bash
go test -v ./internal/dun/... -run "Git|Hook|Commit"
```

Run with coverage:

```bash
go test -coverprofile=coverage.out ./internal/dun/... -run "Git|Hook|Commit"
go tool cover -html=coverage.out
```

## Summary

| Category | Total | Covered | Gaps |
|----------|-------|---------|------|
| AC1: Dirty tree detection | 5 | 5 | 0 |
| AC2: Actionable issues | 10 | 10 | 0 |
| AC3: Hook execution | 8 | 5 | 3 |
| AC4: Missing tool warning | 6 | 5 | 1 |
| AC5: Skip when no config | 3 | 3 | 0 |
| **Total** | **32** | **28** | **4** |

**Coverage: 87.5%**

All critical acceptance criteria have test coverage. The identified gaps are for edge cases and alternative configurations that provide redundant coverage.
