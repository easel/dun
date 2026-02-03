---
dun:
  id: TP-004
  depends_on:
  - TD-004
---
# TP-004: Install Command Test Plan

**User Story**: US-004 - Install AGENTS Guidance
**Status**: In Progress
**Last Updated**: 2026-01-30

## 1. Acceptance Criteria

From US-004:

| AC# | Criterion | Description |
|-----|-----------|-------------|
| AC1 | `dun install` writes AGENTS.md if it is missing | Command creates AGENTS.md file when it does not exist |
| AC2 | `dun install --dry-run` shows planned changes without writing files | Dry run mode previews changes without filesystem modifications |
| AC3 | Re-running `dun install` is safe and idempotent | Multiple runs produce identical results |
| AC4 | The insertion uses marker blocks and avoids destructive edits | Uses `<!-- DUN:BEGIN -->` / `<!-- DUN:END -->` markers to protect existing content |

## 2. Test Coverage Mapping

### AC1: Creates AGENTS.md if Missing

| Test Case | File | Function | Status |
|-----------|------|----------|--------|
| Creates AGENTS.md in empty repo | `install_test.go` | `TestInstallCreatesAgentsFile` | COVERED |
| Verifies marker block is present | `install_test.go` | `TestInstallCreatesAgentsFile` | COVERED |
| Verifies tool line is present | `install_test.go` | `TestInstallCreatesAgentsFile` | COVERED |
| Creates config file alongside | `install_test.go` | `TestInstallCreatesAgentsFile` | COVERED |
| CLI outputs "installed:" message | `main_test.go` | `TestRunInstallOutputsInstalled` | COVERED |

### AC2: Dry Run Mode

| Test Case | File | Function | Status |
|-----------|------|----------|--------|
| Dry run does not create AGENTS.md | `install_test.go` | `TestInstallDryRunDoesNotWrite` | COVERED |
| Dry run does not create config | `install_test.go` | `TestInstallDryRunDoesNotWrite` | COVERED |
| CLI dry run outputs "plan:" | `main_test.go` | `TestRunInstallDryRunAndError` | COVERED |
| Dry run shows what would be created | - | - | GAP |
| Dry run shows what would be updated | - | - | GAP |

### AC3: Idempotent Behavior

| Test Case | File | Function | Status |
|-----------|------|----------|--------|
| Second run produces identical AGENTS.md | `install_test.go` | `TestInstallIsIdempotent` | COVERED |
| Second run produces identical config | - | - | GAP |
| Existing content is preserved | `install_test.go` | `TestInstallInsertsUnderToolsHeader` | COVERED |
| Noop action returned when already installed | `install_test.go` | `TestEnsureConfigFileNoop` | COVERED |

### AC4: Marker Block Insertion

| Test Case | File | Function | Status |
|-----------|------|----------|--------|
| Inserts under ## Tools header | `install_test.go` | `TestInstallInsertsUnderToolsHeader` | COVERED |
| Preserves existing tools | `install_test.go` | `TestInstallInsertsUnderToolsHeader` | COVERED |
| Updates existing marker block | `install_test.go` | `TestUpsertAgentsContentWithMarkers` | COVERED |
| Preserves preface content | `install_test.go` | `TestUpsertAgentsContentWithPreface` | COVERED |
| Handles missing ## Tools header | `install_test.go` | `TestInsertAfterToolsNoHeader` | COVERED |
| Detects malformed markers | `install_test.go` | `TestReplaceMarkerBlockError` | COVERED |
| Detects malformed markers in content | `install_test.go` | `TestUpsertAgentsContentMalformedMarkers` | COVERED |

## 3. Error Handling Coverage

| Test Case | File | Function | Status |
|-----------|------|----------|--------|
| Error when no .git directory | `install_test.go` | `TestInstallRepoErrorWhenNoGit` | COVERED |
| Error when .dun is a file not dir | `install_test.go` | `TestInstallRepoConfigError` | COVERED |
| Error when AGENTS.md is a directory | `install_test.go` | `TestInstallRepoAgentsFileError` | COVERED |
| Error on config stat failure | `install_test.go` | `TestEnsureConfigFileStatError` | COVERED |
| Error on config mkdir failure | `install_test.go` | `TestEnsureConfigFileMkdirError` | COVERED |
| Error on config write failure | `install_test.go` | `TestEnsureConfigFileWriteError` | COVERED |
| Error on agents read failure | `install_test.go` | `TestUpsertAgentsFileReadError` | COVERED |
| Error on agents content failure | `install_test.go` | `TestUpsertAgentsFileContentError` | COVERED |
| Error on agents write failure | `install_test.go` | `TestUpsertAgentsFileWriteError` | COVERED |
| FindRepoRoot error handling | `install_test.go` | `TestFindRepoRootError` | COVERED |
| FindRepoRoot abs path error | `install_test.go` | `TestFindRepoRootAbsError` | COVERED |
| CLI parse error for bad flags | `main_test.go` | `TestRunInstallParseError` | COVERED |
| CLI error for non-repo directory | `main_test.go` | `TestRunInstallDryRunAndError` | COVERED |

## 4. Identified Gaps

### 4.1 High Priority Gaps

| Gap ID | Description | Proposed Test |
|--------|-------------|---------------|
| G1 | Dry run output does not verify what would be created/updated | `TestInstallDryRunShowsPlannedActions` |
| G2 | No test for config idempotency specifically | `TestInstallConfigIsIdempotent` |
| G3 | No integration test via CLI for successful install | `TestRunInstallSuccess` (exists partially) |

### 4.2 Medium Priority Gaps

| Gap ID | Description | Proposed Test |
|--------|-------------|---------------|
| G4 | No test for install from subdirectory of repo | `TestInstallFromSubdirectory` |
| G5 | No test for concurrent install calls | `TestInstallConcurrentSafety` |
| G6 | No test for very large existing AGENTS.md | `TestInstallLargeAgentsFile` |

### 4.3 Low Priority Gaps

| Gap ID | Description | Proposed Test |
|--------|-------------|---------------|
| G7 | No test for unicode content in existing AGENTS.md | `TestInstallPreservesUnicodeContent` |
| G8 | No test for Windows line endings (CRLF) | `TestInstallHandlesCRLF` |

## 5. Proposed Test Cases

### G1: TestInstallDryRunShowsPlannedActions

```go
func TestInstallDryRunShowsPlannedActions(t *testing.T) {
    root := tempRepo(t)

    result, err := InstallRepo(root, InstallOptions{DryRun: true})
    if err != nil {
        t.Fatalf("install dry run: %v", err)
    }

    // Verify result contains planned actions
    if len(result.Steps) != 2 {
        t.Fatalf("expected 2 planned steps, got %d", len(result.Steps))
    }

    configStep := findStep(result, "config")
    if configStep == nil || configStep.Action != "create" {
        t.Fatalf("expected config create action in dry run")
    }

    agentsStep := findStep(result, "agents")
    if agentsStep == nil || agentsStep.Action != "create" {
        t.Fatalf("expected agents create action in dry run")
    }
}
```

### G2: TestInstallConfigIsIdempotent

```go
func TestInstallConfigIsIdempotent(t *testing.T) {
    root := tempRepo(t)

    if _, err := InstallRepo(root, InstallOptions{}); err != nil {
        t.Fatalf("install: %v", err)
    }
    firstConfig := readFile(t, filepath.Join(root, DefaultConfigPath))

    if _, err := InstallRepo(root, InstallOptions{}); err != nil {
        t.Fatalf("install again: %v", err)
    }
    secondConfig := readFile(t, filepath.Join(root, DefaultConfigPath))

    if firstConfig != secondConfig {
        t.Fatalf("expected idempotent config install")
    }
}
```

### G4: TestInstallFromSubdirectory

```go
func TestInstallFromSubdirectory(t *testing.T) {
    root := tempRepo(t)
    subdir := filepath.Join(root, "nested", "work")
    if err := os.MkdirAll(subdir, 0755); err != nil {
        t.Fatalf("mkdir: %v", err)
    }

    // Install should work from subdirectory and write to repo root
    result, err := InstallRepo(subdir, InstallOptions{})
    if err != nil {
        t.Fatalf("install from subdir: %v", err)
    }

    // Files should be at repo root, not subdir
    if _, err := os.Stat(filepath.Join(root, "AGENTS.md")); err != nil {
        t.Fatalf("expected AGENTS.md at repo root")
    }
    if _, err := os.Stat(filepath.Join(subdir, "AGENTS.md")); err == nil {
        t.Fatalf("should not create AGENTS.md in subdir")
    }
}
```

## 6. Test Execution Notes

### Running Install Tests

```bash
# Run all install tests
go test -v ./internal/dun -run Install

# Run with coverage
go test -v -coverprofile=coverage.out ./internal/dun -run Install
go tool cover -html=coverage.out

# Run CLI install tests
go test -v ./cmd/dun -run Install
```

### Test Environment Requirements

- Tests use `t.TempDir()` for isolated filesystem
- Tests create `.git` directory to simulate repository
- No external dependencies required
- Tests run in parallel safely (isolated temp directories)

## 7. Coverage Summary

| Category | Covered | Total | Percentage |
|----------|---------|-------|------------|
| AC1 (Create AGENTS.md) | 5 | 5 | 100% |
| AC2 (Dry Run) | 3 | 5 | 60% |
| AC3 (Idempotent) | 3 | 4 | 75% |
| AC4 (Marker Blocks) | 7 | 7 | 100% |
| Error Handling | 14 | 14 | 100% |
| **Total** | **32** | **35** | **91%** |

## 8. Recommendations

1. **Add G1 and G2 tests** to complete acceptance criteria coverage
2. **Consider G4** for improved subdirectory handling validation
3. **Skip G5-G8** unless specific issues are reported (low risk areas)
4. Current test coverage is strong for core functionality and error paths
