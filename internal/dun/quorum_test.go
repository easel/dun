package dun

import (
	"testing"
)

func TestParseQuorumFlagsNumeric(t *testing.T) {
	tests := []struct {
		name      string
		quorum    string
		harnesses string
		wantErr   error
		threshold int
	}{
		{"quorum 2 of 3", "2", "a,b,c", nil, 2},
		{"quorum 1 of 3", "1", "a,b,c", nil, 1},
		{"quorum 3 of 3", "3", "a,b,c", nil, 3},
		{"quorum exceeds harness", "4", "a,b,c", ErrQuorumExceedsHarness, 0},
		{"quorum zero", "0", "a,b,c", ErrQuorumZero, 0},
		{"quorum negative", "-1", "a,b,c", ErrQuorumZero, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := ParseQuorumFlags(tt.quorum, tt.harnesses, false, false, "")
			if err != tt.wantErr {
				t.Errorf("ParseQuorumFlags() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil && cfg.Threshold != tt.threshold {
				t.Errorf("threshold = %d, want %d", cfg.Threshold, tt.threshold)
			}
		})
	}
}

func TestParseQuorumFlagsStrategies(t *testing.T) {
	tests := []struct {
		name     string
		quorum   string
		harness  string
		strategy string
		wantErr  error
	}{
		{"any strategy", "any", "a,b,c", "any", nil},
		{"majority strategy", "majority", "a,b,c", "majority", nil},
		{"unanimous strategy", "unanimous", "a,b,c", "unanimous", nil},
		{"invalid strategy", "invalid", "a,b,c", "", ErrQuorumInvalidStrategy},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := ParseQuorumFlags(tt.quorum, tt.harness, false, false, "")
			if err != tt.wantErr {
				t.Errorf("ParseQuorumFlags() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil && cfg.Strategy != tt.strategy {
				t.Errorf("strategy = %q, want %q", cfg.Strategy, tt.strategy)
			}
		})
	}
}

func TestParseQuorumFlagsOptions(t *testing.T) {
	cfg, err := ParseQuorumFlags("2", "a,b,c", true, true, "a")
	if err != nil {
		t.Fatalf("ParseQuorumFlags() error = %v", err)
	}
	if cfg.Mode != "sequential" {
		t.Errorf("Mode = %q, want sequential", cfg.Mode)
	}
	if !cfg.Escalate {
		t.Error("Escalate = false, want true")
	}
	if cfg.Prefer != "a" {
		t.Errorf("Prefer = %q, want a", cfg.Prefer)
	}
}

func TestParseQuorumFlagsHarnesses(t *testing.T) {
	tests := []struct {
		name     string
		harness  string
		expected []string
		count    int
	}{
		{"single harness", "claude", []string{"claude"}, 1},
		{"multiple harnesses", "claude,gemini,codex", []string{"claude", "gemini", "codex"}, 3},
		{"with spaces", "claude, gemini , codex", []string{"claude", "gemini", "codex"}, 3},
		{"empty", "", nil, 0},
		{"trailing comma", "a,b,", []string{"a", "b"}, 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := ParseQuorumFlags("", tt.harness, false, false, "")
			if err != nil {
				t.Fatalf("ParseQuorumFlags() error = %v", err)
			}
			if cfg.TotalHarnesses != tt.count {
				t.Errorf("TotalHarnesses = %d, want %d", cfg.TotalHarnesses, tt.count)
			}
			if len(cfg.Harnesses) != len(tt.expected) {
				t.Errorf("len(Harnesses) = %d, want %d", len(cfg.Harnesses), len(tt.expected))
				return
			}
			for i, h := range cfg.Harnesses {
				if h != tt.expected[i] {
					t.Errorf("Harnesses[%d] = %q, want %q", i, h, tt.expected[i])
				}
			}
		})
	}
}

func TestParseQuorumFlagsEmptyQuorum(t *testing.T) {
	cfg, err := ParseQuorumFlags("", "a,b,c", false, false, "")
	if err != nil {
		t.Fatalf("ParseQuorumFlags() error = %v", err)
	}
	if cfg.Strategy != "" {
		t.Errorf("Strategy = %q, want empty", cfg.Strategy)
	}
	if cfg.Threshold != 0 {
		t.Errorf("Threshold = %d, want 0", cfg.Threshold)
	}
}

func TestQuorumConfigIsMet(t *testing.T) {
	tests := []struct {
		name       string
		strategy   string
		threshold  int
		agreements int
		total      int
		want       bool
	}{
		// any strategy
		{"any: 1 of 3", "any", 0, 1, 3, true},
		{"any: 0 of 3", "any", 0, 0, 3, false},
		{"any: 3 of 3", "any", 0, 3, 3, true},

		// majority strategy
		{"majority: 2 of 3", "majority", 0, 2, 3, true},
		{"majority: 1 of 3", "majority", 0, 1, 3, false},
		{"majority: 3 of 3", "majority", 0, 3, 3, true},
		{"majority: 1 of 2", "majority", 0, 1, 2, false},
		{"majority: 2 of 2", "majority", 0, 2, 2, true},
		{"majority: 3 of 5", "majority", 0, 3, 5, true},
		{"majority: 2 of 5", "majority", 0, 2, 5, false},
		{"majority: 1 of 1", "majority", 0, 1, 1, true},

		// unanimous strategy
		{"unanimous: 3 of 3", "unanimous", 0, 3, 3, true},
		{"unanimous: 2 of 3", "unanimous", 0, 2, 3, false},
		{"unanimous: 1 of 1", "unanimous", 0, 1, 1, true},
		{"unanimous: 0 of 1", "unanimous", 0, 0, 1, false},

		// numeric threshold
		{"numeric: 2 of 3 (threshold 2)", "", 2, 2, 3, true},
		{"numeric: 1 of 3 (threshold 2)", "", 2, 1, 3, false},
		{"numeric: 3 of 3 (threshold 2)", "", 2, 3, 3, true},
		{"numeric: 1 of 3 (threshold 1)", "", 1, 1, 3, true},

		// zero total
		{"zero total", "any", 0, 0, 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := QuorumConfig{
				Strategy:  tt.strategy,
				Threshold: tt.threshold,
			}
			got := cfg.IsMet(tt.agreements, tt.total)
			if got != tt.want {
				t.Errorf("IsMet(%d, %d) = %v, want %v", tt.agreements, tt.total, got, tt.want)
			}
		})
	}
}

func TestQuorumConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     QuorumConfig
		wantErr error
	}{
		{
			name:    "no harnesses",
			cfg:     QuorumConfig{Strategy: "any", TotalHarnesses: 0},
			wantErr: nil,
		},
		{
			name:    "any with harnesses",
			cfg:     QuorumConfig{Strategy: "any", TotalHarnesses: 3},
			wantErr: nil,
		},
		{
			name:    "majority with harnesses",
			cfg:     QuorumConfig{Strategy: "majority", TotalHarnesses: 3},
			wantErr: nil,
		},
		{
			name:    "unanimous with harnesses",
			cfg:     QuorumConfig{Strategy: "unanimous", TotalHarnesses: 3},
			wantErr: nil,
		},
		{
			name:    "numeric threshold valid",
			cfg:     QuorumConfig{Threshold: 2, TotalHarnesses: 3},
			wantErr: nil,
		},
		{
			name:    "numeric threshold exceeds",
			cfg:     QuorumConfig{Threshold: 4, TotalHarnesses: 3},
			wantErr: ErrQuorumExceedsHarness,
		},
		{
			name:    "numeric threshold zero",
			cfg:     QuorumConfig{Threshold: 0, TotalHarnesses: 3},
			wantErr: ErrQuorumZero,
		},
		{
			name:    "threshold equals harness count",
			cfg:     QuorumConfig{Threshold: 3, TotalHarnesses: 3},
			wantErr: nil,
		},
		{
			name:    "single harness with threshold 1",
			cfg:     QuorumConfig{Threshold: 1, TotalHarnesses: 1},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if err != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestQuorumConfigIsActive(t *testing.T) {
	tests := []struct {
		name string
		cfg  QuorumConfig
		want bool
	}{
		{"no harnesses", QuorumConfig{Strategy: "any"}, false},
		{"harnesses but no quorum", QuorumConfig{TotalHarnesses: 3}, false},
		{"strategy set", QuorumConfig{Strategy: "any", TotalHarnesses: 3}, true},
		{"threshold set", QuorumConfig{Threshold: 2, TotalHarnesses: 3}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.cfg.IsActive()
			if got != tt.want {
				t.Errorf("IsActive() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseHarnessList(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{"a,b,c", []string{"a", "b", "c"}},
		{"a", []string{"a"}},
		{"", []string{}},
		{" a , b , c ", []string{"a", "b", "c"}},
		{"a,,b", []string{"a", "b"}},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := parseHarnessList(tt.input)
			if len(got) != len(tt.expected) {
				t.Errorf("parseHarnessList(%q) = %v, want %v", tt.input, got, tt.expected)
				return
			}
			for i := range got {
				if got[i] != tt.expected[i] {
					t.Errorf("parseHarnessList(%q)[%d] = %q, want %q", tt.input, i, got[i], tt.expected[i])
				}
			}
		})
	}
}

func TestEffectiveThreshold(t *testing.T) {
	tests := []struct {
		name     string
		cfg      QuorumConfig
		expected int
	}{
		{"any", QuorumConfig{Strategy: "any", TotalHarnesses: 5}, 1},
		{"majority 3", QuorumConfig{Strategy: "majority", TotalHarnesses: 3}, 2},
		{"majority 5", QuorumConfig{Strategy: "majority", TotalHarnesses: 5}, 3},
		{"majority 2", QuorumConfig{Strategy: "majority", TotalHarnesses: 2}, 2},
		{"unanimous 3", QuorumConfig{Strategy: "unanimous", TotalHarnesses: 3}, 3},
		{"unanimous 1", QuorumConfig{Strategy: "unanimous", TotalHarnesses: 1}, 1},
		{"numeric 2", QuorumConfig{Threshold: 2, TotalHarnesses: 3}, 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.cfg.effectiveThreshold()
			if got != tt.expected {
				t.Errorf("effectiveThreshold() = %d, want %d", got, tt.expected)
			}
		})
	}
}

func TestQuorumConfigDefaultMode(t *testing.T) {
	cfg, err := ParseQuorumFlags("2", "a,b,c", false, false, "")
	if err != nil {
		t.Fatalf("ParseQuorumFlags() error = %v", err)
	}
	if cfg.Mode != "parallel" {
		t.Errorf("Mode = %q, want parallel", cfg.Mode)
	}
}

func TestQuorumEdgeCases(t *testing.T) {
	// Single harness with any strategy
	cfg, err := ParseQuorumFlags("any", "claude", false, false, "")
	if err != nil {
		t.Fatalf("ParseQuorumFlags() error = %v", err)
	}
	if !cfg.IsMet(1, 1) {
		t.Error("any with single harness should pass with 1 agreement")
	}

	// Single harness with unanimous
	cfg, err = ParseQuorumFlags("unanimous", "claude", false, false, "")
	if err != nil {
		t.Fatalf("ParseQuorumFlags() error = %v", err)
	}
	if !cfg.IsMet(1, 1) {
		t.Error("unanimous with single harness should pass with 1 agreement")
	}

	// Two harnesses with majority (needs 2/2)
	cfg, err = ParseQuorumFlags("majority", "a,b", false, false, "")
	if err != nil {
		t.Fatalf("ParseQuorumFlags() error = %v", err)
	}
	if cfg.IsMet(1, 2) {
		t.Error("majority with 2 harnesses should not pass with 1 agreement")
	}
	if !cfg.IsMet(2, 2) {
		t.Error("majority with 2 harnesses should pass with 2 agreements")
	}
}
