package main

import (
	"sort"
	"strings"

	"github.com/easel/dun/internal/dun"
)

var defaultQuorumHarnesses = []string{"codex", "claude", "gemini"}
var harnessPreferenceOrder = []string{"codex", "claude", "gemini", "opencode"}

func resolveHarnessesForQuorum(explicit string) []string {
	if strings.TrimSpace(explicit) != "" {
		return parseHarnessCSV(explicit)
	}
	if cached := cachedHarnesses(); len(cached) > 0 {
		return cached
	}
	return append([]string{}, defaultQuorumHarnesses...)
}

func resolveHarnessesForReview(explicit string) []string {
	return resolveHarnessesForQuorum(explicit)
}

func parseHarnessCSV(value string) []string {
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			continue
		}
		out = append(out, trimmed)
	}
	return out
}

func cachedHarnesses() []string {
	cache, err := dun.LoadHarnessCache()
	if err != nil {
		return nil
	}
	available := cache.AvailableHarnesses()
	if len(available) == 0 {
		return nil
	}
	filtered := filterKnownHarnesses(available)
	if len(filtered) == 0 {
		return nil
	}
	return orderHarnesses(filtered)
}

func filterKnownHarnesses(names []string) []string {
	var out []string
	for _, name := range names {
		if name == "mock" {
			continue
		}
		if !dun.DefaultRegistry.Has(name) {
			continue
		}
		out = append(out, name)
	}
	return out
}

func orderHarnesses(names []string) []string {
	seen := make(map[string]bool, len(names))
	for _, name := range names {
		seen[name] = true
	}
	var ordered []string
	for _, name := range harnessPreferenceOrder {
		if seen[name] {
			ordered = append(ordered, name)
			delete(seen, name)
		}
	}
	if len(seen) == 0 {
		return ordered
	}
	var rest []string
	for name := range seen {
		rest = append(rest, name)
	}
	sort.Strings(rest)
	return append(ordered, rest...)
}
