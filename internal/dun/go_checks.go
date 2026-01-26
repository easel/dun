package dun

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

const defaultCoverageThreshold = 100

var closeCoverageFile = func(f *os.File) error {
	return f.Close()
}

func runGoTestCheck(root string, check Check) (CheckResult, error) {
	output, err := runGoCommand(root, "test", "./...")
	if err != nil {
		return CheckResult{
			ID:     check.ID,
			Status: "fail",
			Signal: "go test failed",
			Detail: trimOutput(output),
			Next:   "go test ./...",
		}, nil
	}
	return CheckResult{
		ID:     check.ID,
		Status: "pass",
		Signal: "go test passed",
	}, nil
}

func runGoCoverageCheck(root string, check Check) (CheckResult, error) {
	coveragePath, err := writeCoverageProfile(root)
	if err != nil {
		return CheckResult{}, err
	}
	defer os.Remove(coveragePath)

	output, err := runGoCommand(root, "test", "./...", "-coverprofile", coveragePath)
	if err != nil {
		return CheckResult{
			ID:     check.ID,
			Status: "fail",
			Signal: "go test failed",
			Detail: trimOutput(output),
			Next:   "go test ./...",
		}, nil
	}

	coverageOutput, err := runGoToolCover(root, coveragePath)
	if err != nil {
		return CheckResult{
			ID:     check.ID,
			Status: "fail",
			Signal: "coverage parsing failed",
			Detail: err.Error(),
			Next:   "go test ./... -coverprofile=coverage.out && go tool cover -func=coverage.out",
		}, nil
	}

	coverage, err := parseCoveragePercent(coverageOutput)
	if err != nil {
		return CheckResult{
			ID:     check.ID,
			Status: "fail",
			Signal: "coverage parsing failed",
			Detail: err.Error(),
			Next:   "go test ./... -coverprofile=coverage.out && go tool cover -func=coverage.out",
		}, nil
	}

	threshold := coverageThreshold(check)
	if coverage < float64(threshold) {
		return CheckResult{
			ID:     check.ID,
			Status: "fail",
			Signal: "coverage below threshold",
			Detail: fmt.Sprintf("total coverage %.1f%% (target %d%%)", coverage, threshold),
			Next:   fmt.Sprintf("Increase tests to reach %d%%. Run `go test ./... -coverprofile=coverage.out`.", threshold),
		}, nil
	}

	return CheckResult{
		ID:     check.ID,
		Status: "pass",
		Signal: fmt.Sprintf("coverage %.1f%%", coverage),
	}, nil
}

func runGoVetCheck(root string, check Check) (CheckResult, error) {
	output, err := runGoCommand(root, "vet", "./...")
	if err != nil {
		return CheckResult{
			ID:     check.ID,
			Status: "fail",
			Signal: "go vet failed",
			Detail: trimOutput(output),
			Next:   "go vet ./...",
		}, nil
	}
	return CheckResult{
		ID:     check.ID,
		Status: "pass",
		Signal: "go vet passed",
	}, nil
}

func runGoStaticcheck(root string, check Check) (CheckResult, error) {
	if _, err := exec.LookPath("staticcheck"); err != nil {
		return CheckResult{
			ID:     check.ID,
			Status: "warn",
			Signal: "staticcheck missing",
			Detail: "staticcheck not found on PATH",
			Next:   "Install staticcheck (https://staticcheck.io)",
		}, nil
	}

	cmd := exec.Command("staticcheck", "./...")
	cmd.Dir = root
	output, err := cmd.CombinedOutput()
	if err != nil {
		return CheckResult{
			ID:     check.ID,
			Status: "fail",
			Signal: "staticcheck failed",
			Detail: trimOutput(output),
			Next:   "staticcheck ./...",
		}, nil
	}

	return CheckResult{
		ID:     check.ID,
		Status: "pass",
		Signal: "staticcheck passed",
	}, nil
}

func runGoCommand(root string, args ...string) ([]byte, error) {
	cmd := exec.Command("go", args...)
	cmd.Dir = root
	output, err := cmd.CombinedOutput()
	if err != nil {
		return output, err
	}
	return output, nil
}

func runGoToolCover(root string, coveragePath string) ([]byte, error) {
	cmd := exec.Command("go", "tool", "cover", "-func", coveragePath)
	cmd.Dir = root
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("go tool cover: %w", err)
	}
	return output, nil
}

func writeCoverageProfile(root string) (string, error) {
	path, err := os.CreateTemp(root, "dun-cover-*.out")
	if err != nil {
		return "", err
	}
	if err := closeCoverageFile(path); err != nil {
		return "", err
	}
	return path.Name(), nil
}

func parseCoveragePercent(output []byte) (float64, error) {
	text := strings.TrimSpace(string(output))
	if text == "" {
		return 0, errors.New("coverage output empty")
	}
	lines := strings.Split(text, "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if !strings.HasPrefix(line, "total:") {
			continue
		}
		parts := strings.Fields(line)
		last := parts[len(parts)-1]
		last = strings.TrimSuffix(last, "%")
		percent, err := strconv.ParseFloat(last, 64)
		if err != nil {
			return 0, fmt.Errorf("invalid coverage percent: %w", err)
		}
		return percent, nil
	}
	return 0, errors.New("coverage summary not found")
}

func coverageThreshold(check Check) int {
	for _, rule := range check.Rules {
		if rule.Type == "coverage-min" && rule.Expected > 0 {
			return rule.Expected
		}
	}
	return defaultCoverageThreshold
}
