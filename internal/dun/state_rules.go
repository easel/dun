package dun

import (
	"fmt"
	"io/fs"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

type stateRules struct {
	ArtifactPatterns struct {
		Story map[string]artifactPattern `yaml:"story"`
	} `yaml:"artifact_patterns"`
}

type artifactPattern struct {
	Pattern string `yaml:"pattern"`
}

func runStateRules(root string, plugin Plugin, check Check) (CheckResult, error) {
	if check.StateRules == "" {
		return CheckResult{}, fmt.Errorf("state rules path missing")
	}
	raw, err := fs.ReadFile(plugin.FS, path.Join(plugin.Base, check.StateRules))
	if err != nil {
		return CheckResult{}, fmt.Errorf("read state rules: %w", err)
	}

	var rules stateRules
	if err := yaml.Unmarshal(raw, &rules); err != nil {
		return CheckResult{}, fmt.Errorf("parse state rules: %w", err)
	}

	storyPatterns := rules.ArtifactPatterns.Story
	usIDs, err := idsForPattern(root, storyPatterns["frame"])
	if err != nil {
		return CheckResult{}, err
	}
	tdIDs, err := idsForPattern(root, storyPatterns["design"])
	if err != nil {
		return CheckResult{}, err
	}
	tpIDs, err := idsForPattern(root, storyPatterns["test"])
	if err != nil {
		return CheckResult{}, err
	}
	ipIDs, err := idsForPattern(root, storyPatterns["build"])
	if err != nil {
		return CheckResult{}, err
	}

	var missing []string
	missing = append(missing, missingUpstream("TD", tdIDs, "US", usIDs)...)
	missing = append(missing, missingUpstream("TP", tpIDs, "TD", tdIDs)...)
	missing = append(missing, missingUpstream("IP", ipIDs, "TP", tpIDs)...)

	sort.Strings(missing)

	if len(missing) == 0 {
		return CheckResult{
			ID:     check.ID,
			Status: "pass",
			Signal: "workflow progression valid",
		}, nil
	}

	return CheckResult{
		ID:     check.ID,
		Status: "fail",
		Signal: fmt.Sprintf("%d progression gaps", len(missing)),
		Detail: strings.Join(missing, "; "),
		Next:   "Create missing upstream artifacts for listed story IDs",
	}, nil
}

func idsForPattern(root string, pattern artifactPattern) (map[string]bool, error) {
	if pattern.Pattern == "" {
		return map[string]bool{}, nil
	}
	glob := strings.ReplaceAll(pattern.Pattern, "{id}", "*")
	matches, err := filepath.Glob(filepath.Join(root, filepath.FromSlash(glob)))
	if err != nil {
		return nil, err
	}

	prefix := prefixFromPattern(pattern.Pattern)
	ids := make(map[string]bool)
	for _, match := range matches {
		base := filepath.Base(match)
		if id := parseID(base, prefix); id != "" {
			ids[id] = true
		}
	}
	return ids, nil
}

func prefixFromPattern(pattern string) string {
	base := path.Base(pattern)
	idx := strings.Index(base, "{id}")
	if idx == -1 {
		return ""
	}
	prefix := strings.TrimSuffix(base[:idx], "-")
	return prefix
}

func parseID(base, prefix string) string {
	name := strings.TrimSuffix(base, filepath.Ext(base))
	if prefix == "" || !strings.HasPrefix(name, prefix+"-") {
		return ""
	}
	rest := strings.TrimPrefix(name, prefix+"-")
	parts := strings.SplitN(rest, "-", 2)
	return parts[0]
}

func missingUpstream(downPrefix string, downIDs map[string]bool, upPrefix string, upIDs map[string]bool) []string {
	var missing []string
	for id := range downIDs {
		if !upIDs[id] {
			missing = append(missing, fmt.Sprintf("%s-%s missing for %s-%s", upPrefix, id, downPrefix, id))
		}
	}
	return missing
}
