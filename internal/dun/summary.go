package dun

import "fmt"

func summarizeResult(result CheckResult) CheckResult {
	if result.Summary == "" {
		result.Summary = defaultSummary(result)
	}
	if result.Score == nil {
		if score := defaultScore(result.Status); score != nil {
			result.Score = score
		}
	}
	return result
}

func defaultSummary(result CheckResult) string {
	base := result.Signal
	if base == "" {
		base = result.Detail
	}
	if base == "" {
		base = result.Status
	}
	if len(result.Issues) > 0 {
		base = fmt.Sprintf("%s (%d issues)", base, len(result.Issues))
	}
	return base
}

func defaultScore(status string) *CheckScore {
	switch status {
	case "pass":
		return &CheckScore{Value: 100, Reason: "pass"}
	case "warn":
		return &CheckScore{Value: 70, Reason: "warn"}
	case "fail":
		return &CheckScore{Value: 30, Reason: "fail"}
	case "error":
		return &CheckScore{Value: 10, Reason: "error"}
	default:
		return nil
	}
}
