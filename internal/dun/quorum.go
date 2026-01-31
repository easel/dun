package dun

import (
	"errors"
	"strconv"
	"strings"
)

// QuorumConfig holds quorum settings parsed from flags.
type QuorumConfig struct {
	Strategy       string   // "any", "majority", "unanimous", or "" (numeric)
	Threshold      int      // Numeric threshold when Strategy is ""
	Harnesses      []string // List of harness names
	TotalHarnesses int      // Computed count
	Mode           string   // "parallel" or "sequential"
	Prefer         string   // Preferred harness on conflict
	Escalate       bool     // Pause for human review on conflict
}

var (
	ErrQuorumZero            = errors.New("quorum threshold must be at least 1")
	ErrQuorumExceedsHarness  = errors.New("quorum threshold exceeds harness count")
	ErrQuorumInvalidStrategy = errors.New("unknown quorum strategy")
)

// ParseQuorumFlags parses quorum-related command line flags into a QuorumConfig.
// It returns an error if the configuration is invalid.
func ParseQuorumFlags(quorum string, harnesses string, costMode bool, escalate bool, prefer string) (QuorumConfig, error) {
	cfg := QuorumConfig{
		Escalate: escalate,
		Prefer:   prefer,
		Mode:     "parallel",
	}

	if costMode {
		cfg.Mode = "sequential"
	}

	// Parse harnesses
	if harnesses != "" {
		cfg.Harnesses = parseHarnessList(harnesses)
	}
	cfg.TotalHarnesses = len(cfg.Harnesses)

	// Parse quorum value
	if quorum == "" {
		return cfg, nil
	}

	switch quorum {
	case "any":
		cfg.Strategy = "any"
		cfg.Threshold = 1
	case "majority":
		cfg.Strategy = "majority"
	case "unanimous":
		cfg.Strategy = "unanimous"
	default:
		n, err := strconv.Atoi(quorum)
		if err != nil {
			return QuorumConfig{}, ErrQuorumInvalidStrategy
		}
		if n <= 0 {
			return QuorumConfig{}, ErrQuorumZero
		}
		cfg.Threshold = n
	}

	// Validate quorum against harness count
	if err := cfg.Validate(); err != nil {
		return QuorumConfig{}, err
	}

	return cfg, nil
}

// parseHarnessList splits a comma-separated list of harness names.
func parseHarnessList(harnesses string) []string {
	parts := strings.Split(harnesses, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

// Validate checks that the QuorumConfig is internally consistent.
func (q *QuorumConfig) Validate() error {
	if q.TotalHarnesses == 0 {
		return nil // No harnesses configured, quorum not active
	}

	threshold := q.effectiveThreshold()
	if threshold <= 0 {
		return ErrQuorumZero
	}
	if threshold > q.TotalHarnesses {
		return ErrQuorumExceedsHarness
	}

	return nil
}

// effectiveThreshold returns the minimum agreements needed for the configured strategy.
func (q *QuorumConfig) effectiveThreshold() int {
	switch q.Strategy {
	case "any":
		return 1
	case "majority":
		return (q.TotalHarnesses / 2) + 1
	case "unanimous":
		return q.TotalHarnesses
	default:
		return q.Threshold
	}
}

// IsMet returns true if the number of agreements satisfies the quorum requirement.
func (q *QuorumConfig) IsMet(agreements, total int) bool {
	if total == 0 {
		return false
	}

	switch q.Strategy {
	case "any":
		return agreements >= 1
	case "majority":
		return agreements > total/2
	case "unanimous":
		return agreements == total
	default:
		// Numeric threshold
		return agreements >= q.Threshold
	}
}

// IsActive returns true if quorum checking is enabled.
func (q *QuorumConfig) IsActive() bool {
	return q.TotalHarnesses > 0 && (q.Strategy != "" || q.Threshold > 0)
}
