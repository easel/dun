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

	var missingRequired []string
	var missingOptional []string
	var manualRequired []string

	for _, gatePath := range check.GateFiles {
		gates, err := loadGateFile(plugin, gatePath)
		if err != nil {
			return CheckResult{}, err
		}
		items := append(gates.ExitGates, gates.InputGates...)
		for _, gate := range items {
			if gate.Evidence == "" {
				if gate.Required {
					manualRequired = append(manualRequired, fmt.Sprintf("%s (manual)", gate.Criteria))
				}
				continue
			}
			if exists(filepath.Join(root, filepath.FromSlash(gate.Evidence))) {
				continue
			}
			if gate.Required {
				missingRequired = append(missingRequired, gate.Evidence)
			} else {
				missingOptional = append(missingOptional, gate.Evidence)
			}
		}
	}

	sort.Strings(missingRequired)
	sort.Strings(missingOptional)
	sort.Strings(manualRequired)

	if len(missingRequired) == 0 && len(missingOptional) == 0 && len(manualRequired) == 0 {
		return CheckResult{
			ID:     check.ID,
			Status: "pass",
			Signal: "all gates satisfied",
		}, nil
	}

	status := "warn"
	if len(missingRequired) > 0 {
		status = "fail"
	}

	var detailParts []string
	if len(missingRequired) > 0 {
		detailParts = append(detailParts, "missing required: "+strings.Join(missingRequired, ", "))
	}
	if len(missingOptional) > 0 {
		detailParts = append(detailParts, "missing optional: "+strings.Join(missingOptional, ", "))
	}
	if len(manualRequired) > 0 {
		detailParts = append(detailParts, "manual confirmation: "+strings.Join(manualRequired, ", "))
	}

	signal := fmt.Sprintf("%d required missing, %d manual", len(missingRequired), len(manualRequired))
	if status == "warn" {
		signal = fmt.Sprintf("%d gates need attention", len(missingOptional)+len(manualRequired))
	}

	return CheckResult{
		ID:     check.ID,
		Status: status,
		Signal: signal,
		Detail: strings.Join(detailParts, "; "),
		Next:   "Fill required evidence files or confirm manual gates",
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
