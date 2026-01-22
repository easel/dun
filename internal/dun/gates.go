package dun

import (
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

type gateFile struct {
	InputGates []gateItem `yaml:"input_gates"`
	ExitGates  []gateItem `yaml:"exit_gates"`
}

type gateItem struct {
	Criteria string `yaml:"criteria"`
	Required bool   `yaml:"required"`
	Evidence string `yaml:"evidence"`
	Source   string `yaml:"source"`
}

func runGateCheck(root string, plugin Plugin, check Check) (CheckResult, error) {
	if len(check.GateFiles) == 0 {
		return CheckResult{}, fmt.Errorf("gate files missing")
	}

	requiredActions := map[string]string{}
	optionalActions := map[string]string{}
	manualActions := map[string]string{}
	issuesByKey := map[string]Issue{}

	for _, gatePath := range check.GateFiles {
		gates, err := loadGateFile(plugin, gatePath)
		if err != nil {
			return CheckResult{}, err
		}
		items := append(gates.ExitGates, gates.InputGates...)
		for _, gate := range items {
			if gate.Evidence == "" {
				if gate.Required {
					action := fmt.Sprintf("Confirm manual gate: %s", gate.Criteria)
					manualActions[action] = action
					key := "manual:" + gate.Criteria
					issuesByKey[key] = Issue{
						ID:      key,
						Summary: action,
					}
				}
				continue
			}
			pathPart, anchor := splitEvidence(gate.Evidence)
			resolvedPath := filepath.Join(root, filepath.FromSlash(pathPart))
			missing, missingAnchor, err := evidenceMissing(resolvedPath, anchor)
			if err != nil {
				return CheckResult{}, err
			}
			if !missing {
				continue
			}
			action := buildGateAction(pathPart, anchor, missingAnchor, gate.Criteria)
			target := gate.Evidence
			if gate.Required {
				requiredActions[action] = action
			} else {
				optionalActions[action] = action
			}
			key := gate.Evidence
			if gate.Required {
				key = "required:" + key
			} else {
				key = "optional:" + key
			}
			issuesByKey[key] = Issue{
				ID:      key,
				Summary: action,
				Path:    target,
			}
		}
	}

	requiredList := sortedKeys(requiredActions)
	optionalList := sortedKeys(optionalActions)
	manualList := sortedKeys(manualActions)
	issues := sortedIssues(issuesByKey)

	if len(requiredList) == 0 && len(optionalList) == 0 && len(manualList) == 0 {
		return CheckResult{
			ID:     check.ID,
			Status: "pass",
			Signal: "all gates satisfied",
		}, nil
	}

	status := "warn"
	if len(requiredList) > 0 {
		status = "fail"
	}

	var detailParts []string
	if len(requiredList) > 0 {
		detailParts = append(detailParts, "required actions: "+strings.Join(requiredList, "; "))
	}
	if len(optionalList) > 0 {
		detailParts = append(detailParts, "optional actions: "+strings.Join(optionalList, "; "))
	}
	if len(manualList) > 0 {
		detailParts = append(detailParts, "manual confirmations: "+strings.Join(manualList, "; "))
	}

	signal := fmt.Sprintf("%d required missing, %d manual", len(requiredList), len(manualList))
	if status == "warn" {
		signal = fmt.Sprintf("%d gates need attention", len(optionalList)+len(manualList))
	}

	return CheckResult{
		ID:     check.ID,
		Status: status,
		Signal: signal,
		Detail: strings.Join(detailParts, "; "),
		Next:   "Resolve required actions and confirm manual gates, then rerun `dun check`",
		Issues: issues,
	}, nil
}

func loadGateFile(plugin Plugin, relPath string) (gateFile, error) {
	raw, err := fs.ReadFile(plugin.FS, path.Join(plugin.Base, relPath))
	if err != nil {
		return gateFile{}, fmt.Errorf("read gate file %s: %w", relPath, err)
	}
	var gates gateFile
	if err := yaml.Unmarshal(raw, &gates); err != nil {
		return gateFile{}, fmt.Errorf("parse gate file %s: %w", relPath, err)
	}
	return gates, nil
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func splitEvidence(evidence string) (string, string) {
	parts := strings.SplitN(evidence, "#", 2)
	if len(parts) == 2 {
		return parts[0], strings.TrimSpace(parts[1])
	}
	return evidence, ""
}

func evidenceMissing(path string, anchor string) (bool, bool, error) {
	if anchor == "" {
		return !exists(path), false, nil
	}
	if !exists(path) {
		return true, false, nil
	}
	found, err := hasMarkdownAnchor(path, anchor)
	if err != nil {
		return false, false, err
	}
	if found {
		return false, false, nil
	}
	return true, true, nil
}

func hasMarkdownAnchor(path string, anchor string) (bool, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return false, err
	}
	target := slugify(anchor)
	for _, line := range strings.Split(string(content), "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "#") {
			continue
		}
		title := strings.TrimSpace(strings.TrimLeft(line, "#"))
		if title == "" {
			continue
		}
		if slugify(title) == target {
			return true, nil
		}
	}
	return false, nil
}

func slugify(value string) string {
	var b strings.Builder
	sep := false
	for _, r := range strings.ToLower(strings.TrimSpace(value)) {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
			sep = false
		case r >= '0' && r <= '9':
			b.WriteRune(r)
			sep = false
		default:
			if !sep {
				b.WriteRune('-')
				sep = true
			}
		}
	}
	out := b.String()
	out = strings.Trim(out, "-")
	out = strings.ReplaceAll(out, "--", "-")
	return out
}

func buildGateAction(path string, anchor string, missingAnchor bool, criteria string) string {
	action := ""
	if anchor != "" && missingAnchor {
		action = fmt.Sprintf("Add section '%s' to %s", anchor, path)
	} else if anchor != "" {
		action = fmt.Sprintf("Create %s with section '%s'", path, anchor)
	} else if strings.HasSuffix(path, "/") {
		action = fmt.Sprintf("Create directory %s", path)
	} else {
		action = fmt.Sprintf("Create %s", path)
	}
	if criteria != "" {
		action = fmt.Sprintf("%s (%s)", action, criteria)
	}
	return action
}

func sortedKeys(values map[string]string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func sortedIssues(values map[string]Issue) []Issue {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]Issue, 0, len(keys))
	for _, key := range keys {
		out = append(out, values[key])
	}
	return out
}
