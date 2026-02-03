---
dun:
  id: TP-001
  depends_on:
  - TD-001
---
# TP-001: Auto-Discover Repo Checks

## User Story Reference

**US-001**: As an agent operator, I want Dun to detect the right checks from repo signals so I can run `dun check` without configuration.

## Acceptance Criteria Mapping

| Criterion | Test File | Test Function | Status |
|-----------|-----------|---------------|--------|
| Detect Go repositories via `go.mod` | `internal/dun/engine_extra_test.go` | `TestIsPluginActiveWithTrigger` | Covered |
| Detect Go repositories via `go.mod` | `internal/plugins/builtin/go/plugin.yaml` | Plugin trigger definition | Covered |
| Detect Helix workflow via `docs/helix/` | `internal/dun/plan_test.go` | `TestPlanRepoIncludesHelixChecks` | Covered |
| Detect Helix workflow via `docs/helix/` | `internal/plugins/builtin/helix/plugin.yaml` | Plugin trigger definition | Covered |
| Check IDs and ordering are deterministic | `internal/dun/engine_sort_test.go` | `TestSortPlanByPluginPriority` | Covered |
| Check IDs and ordering are deterministic | `internal/dun/engine_sort_test.go` | `TestSortPlanByCheckPriority` | Covered |
| Check IDs and ordering are deterministic | `internal/dun/engine_sort_test.go` | `TestSortPlanByPhase` | Covered |
| Check IDs and ordering are deterministic | `internal/dun/engine_sort_test.go` | `TestSortPlanByID` | Covered |
| Check IDs and ordering are deterministic | `internal/dun/engine_sort_test.go` | `TestSortPlanCombined` | Covered |
| No user configuration required for core discovery | `cmd/dun/main_test.go` | `TestRunDefaultsToCheck` | Partial |
| No user configuration required for core discovery | `internal/dun/engine_extra_test.go` | `TestCheckRepoEmptyRoot` | Partial |

## Gaps

- [ ] Gap 1: No explicit test verifies Go plugin is activated when `go.mod` exists and deactivated when it does not
- [ ] Gap 2: No explicit test verifies Helix plugin is activated when `docs/helix/` exists and deactivated when it does not
- [ ] Gap 3: No test verifies that multiple plugins (Go + Helix + Git) are correctly discovered together in the same repo
- [ ] Gap 4: No test verifies deterministic ordering across multiple runs (same repo state produces identical check order)
- [ ] Gap 5: No test verifies zero-config discovery works end-to-end from CLI without any `.dun/config.yaml`
- [ ] Gap 6: No test verifies that `dun list` output is deterministic for the same repo state
- [ ] Gap 7: No test for edge case: `go.mod` exists but is empty or malformed
- [ ] Gap 8: No test for edge case: `docs/helix` is a file instead of a directory

## Proposed Test Cases

### Test Case 1: Go Plugin Activation via go.mod

**File**: `internal/dun/engine_extra_test.go`

**Description**: Verify that the Go plugin is activated when `go.mod` exists and produces Go-specific checks.

```go
func TestGoPluginActiveWhenGoModExists(t *testing.T) {
    root := t.TempDir()
    writeFile(t, filepath.Join(root, "go.mod"), "module example.com/test")

    plan, err := PlanRepo(root)
    if err != nil {
        t.Fatalf("plan repo: %v", err)
    }

    // Verify Go checks are included
    goCheckIDs := []string{"go-test", "go-coverage", "go-vet", "go-staticcheck"}
    for _, id := range goCheckIDs {
        if !hasCheck(plan, id) {
            t.Errorf("expected check %s when go.mod exists", id)
        }
    }
}

func TestGoPluginInactiveWhenGoModMissing(t *testing.T) {
    root := t.TempDir()
    // No go.mod file

    plan, err := PlanRepo(root)
    if err != nil {
        t.Fatalf("plan repo: %v", err)
    }

    goCheckIDs := []string{"go-test", "go-coverage", "go-vet", "go-staticcheck"}
    for _, id := range goCheckIDs {
        if hasCheck(plan, id) {
            t.Errorf("unexpected check %s when go.mod missing", id)
        }
    }
}
```

### Test Case 2: Helix Plugin Activation via docs/helix/

**File**: `internal/dun/engine_extra_test.go`

**Description**: Verify that the Helix plugin is activated when `docs/helix/` directory exists.

```go
func TestHelixPluginActiveWhenDocsHelixExists(t *testing.T) {
    root := t.TempDir()
    if err := os.MkdirAll(filepath.Join(root, "docs", "helix"), 0755); err != nil {
        t.Fatalf("mkdir: %v", err)
    }

    plan, err := PlanRepo(root)
    if err != nil {
        t.Fatalf("plan repo: %v", err)
    }

    // Verify Helix checks are included
    if !hasCheck(plan, "helix-gates") {
        t.Error("expected helix-gates check when docs/helix exists")
    }
    if !hasCheck(plan, "helix-state-rules") {
        t.Error("expected helix-state-rules check when docs/helix exists")
    }
}

func TestHelixPluginInactiveWhenDocsHelixMissing(t *testing.T) {
    root := t.TempDir()
    // No docs/helix directory

    plan, err := PlanRepo(root)
    if err != nil {
        t.Fatalf("plan repo: %v", err)
    }

    helixCheckIDs := []string{"helix-gates", "helix-state-rules", "helix-create-architecture"}
    for _, id := range helixCheckIDs {
        if hasCheck(plan, id) {
            t.Errorf("unexpected check %s when docs/helix missing", id)
        }
    }
}
```

### Test Case 3: Multi-Plugin Discovery

**File**: `internal/dun/engine_extra_test.go`

**Description**: Verify that multiple plugins are correctly discovered together when their triggers are satisfied.

```go
func TestMultiPluginDiscovery(t *testing.T) {
    root := tempGitRepo(t) // Creates .git directory
    writeFile(t, filepath.Join(root, "go.mod"), "module example.com/test")
    if err := os.MkdirAll(filepath.Join(root, "docs", "helix"), 0755); err != nil {
        t.Fatalf("mkdir: %v", err)
    }

    plan, err := PlanRepo(root)
    if err != nil {
        t.Fatalf("plan repo: %v", err)
    }

    // Verify checks from all three plugins
    expectedPlugins := map[string][]string{
        "git":   {"git-status", "git-hooks"},
        "go":    {"go-test", "go-coverage", "go-vet", "go-staticcheck"},
        "helix": {"helix-gates", "helix-state-rules"},
    }

    for pluginID, checkIDs := range expectedPlugins {
        for _, checkID := range checkIDs {
            if !hasCheck(plan, checkID) {
                t.Errorf("expected check %s from plugin %s", checkID, pluginID)
            }
        }
    }
}
```

### Test Case 4: Deterministic Ordering Across Runs

**File**: `internal/dun/engine_extra_test.go`

**Description**: Verify that running PlanRepo multiple times on the same repo state produces identical check ordering.

```go
func TestDeterministicOrdering(t *testing.T) {
    root := tempGitRepo(t)
    writeFile(t, filepath.Join(root, "go.mod"), "module example.com/test")
    if err := os.MkdirAll(filepath.Join(root, "docs", "helix"), 0755); err != nil {
        t.Fatalf("mkdir: %v", err)
    }

    // Run PlanRepo multiple times
    var previousIDs []string
    for i := 0; i < 5; i++ {
        plan, err := PlanRepo(root)
        if err != nil {
            t.Fatalf("plan repo run %d: %v", i, err)
        }

        var currentIDs []string
        for _, check := range plan.Checks {
            currentIDs = append(currentIDs, check.ID)
        }

        if previousIDs != nil {
            if !slicesEqual(previousIDs, currentIDs) {
                t.Errorf("run %d: check order differs\nprevious: %v\ncurrent: %v",
                    i, previousIDs, currentIDs)
            }
        }
        previousIDs = currentIDs
    }
}
```

### Test Case 5: Zero-Config CLI Discovery

**File**: `cmd/dun/main_test.go`

**Description**: Verify that `dun check` works without any configuration file.

```go
func TestCheckWorksWithoutConfig(t *testing.T) {
    root := setupEmptyRepo(t)
    writeFile(t, filepath.Join(root, "go.mod"), "module example.com/test\n")

    // Ensure no .dun/config.yaml exists
    dunDir := filepath.Join(root, ".dun")
    if _, err := os.Stat(dunDir); err == nil {
        t.Fatalf(".dun directory should not exist")
    }

    var stdout bytes.Buffer
    var stderr bytes.Buffer
    code := runInDirWithWriters(t, root, []string{"check"}, &stdout, &stderr)
    if code != 0 {
        t.Fatalf("expected success without config, got %d: %s", code, stderr.String())
    }

    // Verify Go checks are in the output
    output := stdout.String()
    if !strings.Contains(output, "go-test") {
        t.Error("expected go-test check in output")
    }
}
```

### Test Case 6: Deterministic List Output

**File**: `cmd/dun/main_test.go`

**Description**: Verify that `dun list` produces deterministic output for the same repo state.

```go
func TestListOutputDeterministic(t *testing.T) {
    root := setupEmptyRepo(t)
    writeFile(t, filepath.Join(root, "go.mod"), "module example.com/test\n")

    var outputs []string
    for i := 0; i < 3; i++ {
        var stdout bytes.Buffer
        var stderr bytes.Buffer
        code := runInDirWithWriters(t, root, []string{"list"}, &stdout, &stderr)
        if code != 0 {
            t.Fatalf("list run %d failed: %s", i, stderr.String())
        }
        outputs = append(outputs, stdout.String())
    }

    for i := 1; i < len(outputs); i++ {
        if outputs[i] != outputs[0] {
            t.Errorf("list output differs between runs\nrun 0: %s\nrun %d: %s",
                outputs[0], i, outputs[i])
        }
    }
}
```

### Test Case 7: Edge Case - Empty go.mod

**File**: `internal/dun/engine_extra_test.go`

**Description**: Verify that an empty `go.mod` file still triggers the Go plugin.

```go
func TestGoPluginActiveWithEmptyGoMod(t *testing.T) {
    root := t.TempDir()
    writeFile(t, filepath.Join(root, "go.mod"), "") // Empty file

    plan, err := PlanRepo(root)
    if err != nil {
        t.Fatalf("plan repo: %v", err)
    }

    // Plugin activation is based on file existence, not content
    if !hasCheck(plan, "go-test") {
        t.Error("expected go-test check even with empty go.mod")
    }
}
```

### Test Case 8: Edge Case - docs/helix is a File

**File**: `internal/dun/engine_extra_test.go`

**Description**: Verify that the Helix plugin is activated when `docs/helix` exists as a file (not directory).

```go
func TestHelixPluginActiveWhenDocsHelixIsFile(t *testing.T) {
    root := t.TempDir()
    if err := os.MkdirAll(filepath.Join(root, "docs"), 0755); err != nil {
        t.Fatalf("mkdir: %v", err)
    }
    writeFile(t, filepath.Join(root, "docs", "helix"), "placeholder")

    plan, err := PlanRepo(root)
    if err != nil {
        t.Fatalf("plan repo: %v", err)
    }

    // path-exists trigger succeeds for files too
    if !hasCheck(plan, "helix-gates") {
        t.Error("expected helix-gates check when docs/helix exists as file")
    }
}
```

## Helper Functions Needed

```go
func hasCheck(plan Plan, id string) bool {
    for _, check := range plan.Checks {
        if check.ID == id {
            return true
        }
    }
    return false
}

func slicesEqual(a, b []string) bool {
    if len(a) != len(b) {
        return false
    }
    for i := range a {
        if a[i] != b[i] {
            return false
        }
    }
    return true
}
```

## Test Priority

| Test Case | Priority | Rationale |
|-----------|----------|-----------|
| Test Case 1: Go Plugin Activation | High | Core acceptance criterion |
| Test Case 2: Helix Plugin Activation | High | Core acceptance criterion |
| Test Case 3: Multi-Plugin Discovery | High | Real-world usage pattern |
| Test Case 4: Deterministic Ordering | High | Core acceptance criterion |
| Test Case 5: Zero-Config CLI Discovery | Medium | User experience validation |
| Test Case 6: Deterministic List Output | Medium | CLI contract verification |
| Test Case 7: Edge Case - Empty go.mod | Low | Edge case handling |
| Test Case 8: Edge Case - docs/helix is File | Low | Edge case handling |
