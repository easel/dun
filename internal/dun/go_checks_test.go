package dun

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGoTestCheckPasses(t *testing.T) {
	binDir := stubGoBinary(t)
	t.Setenv("PATH", binDir)

	root := t.TempDir()
	res, err := runGoTestCheck(root, Check{ID: "go-test"})
	if err != nil {
		t.Fatalf("go test check: %v", err)
	}
	if res.Status != "pass" {
		t.Fatalf("expected pass, got %s", res.Status)
	}
}

func TestGoTestCheckFails(t *testing.T) {
	binDir := stubGoBinary(t)
	t.Setenv("PATH", binDir)
	t.Setenv("DUN_GO_TEST_EXIT", "1")

	root := t.TempDir()
	res, err := runGoTestCheck(root, Check{ID: "go-test"})
	if err != nil {
		t.Fatalf("go test check: %v", err)
	}
	if res.Status != "fail" {
		t.Fatalf("expected fail, got %s", res.Status)
	}
	if !strings.Contains(res.Next, "go test") {
		t.Fatalf("expected next go test, got %q", res.Next)
	}
}

func TestGoCoverageCheckFailsBelowThreshold(t *testing.T) {
	binDir := stubGoBinary(t)
	t.Setenv("PATH", binDir)
	t.Setenv("DUN_COVER_PCT", "72.0")

	root := t.TempDir()
	check := Check{
		ID: "go-coverage",
		Rules: []Rule{
			{Type: "coverage-min", Expected: 100},
		},
	}
	res, err := runGoCoverageCheck(root, check, Options{})
	if err != nil {
		t.Fatalf("coverage check: %v", err)
	}
	if res.Status != "fail" {
		t.Fatalf("expected fail, got %s", res.Status)
	}
	if !strings.Contains(res.Detail, "72.0") {
		t.Fatalf("expected coverage detail, got %q", res.Detail)
	}
}

func TestGoCoverageCheckPassesAtCustomThreshold(t *testing.T) {
	binDir := stubGoBinary(t)
	t.Setenv("PATH", binDir)
	t.Setenv("DUN_COVER_PCT", "80.0")

	root := t.TempDir()
	check := Check{
		ID: "go-coverage",
		Rules: []Rule{
			{Type: "coverage-min", Expected: 75},
		},
	}
	res, err := runGoCoverageCheck(root, check, Options{})
	if err != nil {
		t.Fatalf("coverage check: %v", err)
	}
	if res.Status != "pass" {
		t.Fatalf("expected pass, got %s", res.Status)
	}
}

func TestGoVetCheckFails(t *testing.T) {
	binDir := stubGoBinary(t)
	t.Setenv("PATH", binDir)
	t.Setenv("DUN_GO_VET_EXIT", "1")

	root := t.TempDir()
	res, err := runGoVetCheck(root, Check{ID: "go-vet"})
	if err != nil {
		t.Fatalf("go vet check: %v", err)
	}
	if res.Status != "fail" {
		t.Fatalf("expected fail, got %s", res.Status)
	}
}

func TestGoVetCheckPasses(t *testing.T) {
	binDir := stubGoBinary(t)
	t.Setenv("PATH", binDir)

	root := t.TempDir()
	res, err := runGoVetCheck(root, Check{ID: "go-vet"})
	if err != nil {
		t.Fatalf("go vet check: %v", err)
	}
	if res.Status != "pass" {
		t.Fatalf("expected pass, got %s", res.Status)
	}
}

func TestGoStaticcheckWarnsWhenMissing(t *testing.T) {
	binDir := stubGoBinary(t)
	t.Setenv("PATH", binDir)

	root := t.TempDir()
	res, err := runGoStaticcheck(root, Check{ID: "go-staticcheck"})
	if err != nil {
		t.Fatalf("staticcheck: %v", err)
	}
	if res.Status != "warn" {
		t.Fatalf("expected warn, got %s", res.Status)
	}
}

func TestGoStaticcheckFailsWhenToolErrors(t *testing.T) {
	binDir := stubGoBinary(t)
	staticcheckPath := filepath.Join(binDir, "staticcheck")
	writeFile(t, staticcheckPath, "#!/bin/sh\nexit 1\n")
	if err := os.Chmod(staticcheckPath, 0755); err != nil {
		t.Fatalf("chmod staticcheck: %v", err)
	}
	t.Setenv("PATH", binDir)

	root := t.TempDir()
	res, err := runGoStaticcheck(root, Check{ID: "go-staticcheck"})
	if err != nil {
		t.Fatalf("staticcheck: %v", err)
	}
	if res.Status != "fail" {
		t.Fatalf("expected fail, got %s", res.Status)
	}
}

func TestGoStaticcheckPassesWhenToolOK(t *testing.T) {
	binDir := stubGoBinary(t)
	staticcheckPath := filepath.Join(binDir, "staticcheck")
	writeFile(t, staticcheckPath, "#!/bin/sh\nexit 0\n")
	if err := os.Chmod(staticcheckPath, 0755); err != nil {
		t.Fatalf("chmod staticcheck: %v", err)
	}
	t.Setenv("PATH", binDir)

	root := t.TempDir()
	res, err := runGoStaticcheck(root, Check{ID: "go-staticcheck"})
	if err != nil {
		t.Fatalf("staticcheck: %v", err)
	}
	if res.Status != "pass" {
		t.Fatalf("expected pass, got %s", res.Status)
	}
}

func TestRunGoToolCoverError(t *testing.T) {
	binDir := stubGoBinary(t)
	t.Setenv("PATH", binDir)
	t.Setenv("DUN_TOOL_COVER_EXIT", "1")

	_, err := runGoToolCover(t.TempDir(), "coverage.out")
	if err == nil {
		t.Fatalf("expected go tool cover error")
	}
}

func TestRunGoCoverageCheckToolCoverError(t *testing.T) {
	binDir := stubGoBinary(t)
	t.Setenv("PATH", binDir)
	t.Setenv("DUN_TOOL_COVER_EXIT", "1")

	root := t.TempDir()
	res, err := runGoCoverageCheck(root, Check{ID: "go-coverage"}, Options{})
	if err != nil {
		t.Fatalf("coverage check: %v", err)
	}
	if res.Status != "fail" {
		t.Fatalf("expected fail, got %s", res.Status)
	}
}

func TestRunGoCoverageCheckParseError(t *testing.T) {
	binDir := stubGoBinary(t)
	t.Setenv("PATH", binDir)
	t.Setenv("DUN_COVER_PCT", "bad")

	root := t.TempDir()
	res, err := runGoCoverageCheck(root, Check{ID: "go-coverage"}, Options{})
	if err != nil {
		t.Fatalf("coverage check: %v", err)
	}
	if res.Status != "fail" {
		t.Fatalf("expected fail, got %s", res.Status)
	}
}

func TestParseCoveragePercentErrors(t *testing.T) {
	if _, err := parseCoveragePercent([]byte("")); err == nil {
		t.Fatalf("expected empty output error")
	}
	if _, err := parseCoveragePercent([]byte("no totals here")); err == nil {
		t.Fatalf("expected missing summary error")
	}
	if _, err := parseCoveragePercent([]byte("total: (statements) not-a-percent")); err == nil {
		t.Fatalf("expected invalid percent error")
	}
}

func TestWriteCoverageProfileCloseError(t *testing.T) {
	orig := closeCoverageFile
	closeCoverageFile = func(f *os.File) error {
		return os.ErrInvalid
	}
	t.Cleanup(func() { closeCoverageFile = orig })

	_, err := writeCoverageProfile(t.TempDir())
	if err == nil {
		t.Fatalf("expected close error")
	}
}

func TestCoverageThresholdDefault(t *testing.T) {
	if threshold := coverageThreshold(Check{}, Options{}); threshold != defaultCoverageThreshold {
		t.Fatalf("expected default threshold %d, got %d", defaultCoverageThreshold, threshold)
	}
}

func TestCoverageThresholdFromOptions(t *testing.T) {
	opts := Options{CoverageThreshold: 95}
	if threshold := coverageThreshold(Check{}, opts); threshold != 95 {
		t.Fatalf("expected options threshold 95, got %d", threshold)
	}
}

func TestCoverageThresholdOptionsOverrideRule(t *testing.T) {
	opts := Options{CoverageThreshold: 95}
	check := Check{Rules: []Rule{{Type: "coverage-min", Expected: 100}}}
	if threshold := coverageThreshold(check, opts); threshold != 95 {
		t.Fatalf("expected options threshold 95, got %d", threshold)
	}
}

func TestWriteCoverageProfileCreateError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing", "dir")
	_, err := writeCoverageProfile(path)
	if err == nil {
		t.Fatalf("expected create temp error")
	}
}

func TestRunGoCoverageCheckHandlesGoTestFailure(t *testing.T) {
	binDir := stubGoBinary(t)
	t.Setenv("PATH", binDir)
	t.Setenv("DUN_GO_TEST_EXIT", "1")

	root := t.TempDir()
	res, err := runGoCoverageCheck(root, Check{ID: "go-coverage"}, Options{})
	if err != nil {
		t.Fatalf("coverage check: %v", err)
	}
	if res.Status != "fail" {
		t.Fatalf("expected fail, got %s", res.Status)
	}
}

func TestRunGoCoverageCheckWriteProfileError(t *testing.T) {
	root := filepath.Join(t.TempDir(), "missing")
	if _, err := runGoCoverageCheck(root, Check{ID: "go-coverage"}, Options{}); err == nil {
		t.Fatalf("expected write profile error")
	}
}

func stubGoBinary(t *testing.T) string {
	t.Helper()
	binDir := t.TempDir()
	goPath := filepath.Join(binDir, "go")
	script := `#!/bin/sh
set -e

if [ "$1" = "test" ]; then
  cover=""
  prev=""
  for arg in "$@"; do
    if [ "$prev" = "-coverprofile" ]; then
      cover="$arg"
      prev=""
      continue
    fi
    case "$arg" in
      -coverprofile=*)
        cover="${arg#-coverprofile=}"
        ;;
      -coverprofile)
        prev="-coverprofile"
        ;;
    esac
  done
  if [ -n "$cover" ]; then
    echo "mode: set" > "$cover"
    echo "example.go:1.1,1.2 1 1" >> "$cover"
  fi
  exit ${DUN_GO_TEST_EXIT:-0}
fi

if [ "$1" = "vet" ]; then
  exit ${DUN_GO_VET_EXIT:-0}
fi

if [ "$1" = "tool" ] && [ "$2" = "cover" ]; then
  if [ -n "${DUN_TOOL_COVER_EXIT:-}" ]; then
    exit "${DUN_TOOL_COVER_EXIT}"
  fi
  pct="${DUN_COVER_PCT:-100.0}"
  echo "example.go:1.1,1.2 1 1"
  echo "total: (statements) ${pct}%"
  exit 0
fi

echo "unknown go args: $@" >&2
exit 1
`
	writeFile(t, goPath, script)
	if err := os.Chmod(goPath, 0755); err != nil {
		t.Fatalf("chmod go: %v", err)
	}
	return binDir
}
