package dun

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

var loadBuiltins = LoadBuiltins

type plannedCheck struct {
	Plugin Plugin
	Check  Check
}

type Plan struct {
	Checks []PlannedCheck
}

type PlannedCheck struct {
	ID          string
	Description string
	Type        string
	Phase       string
	PluginID    string
	Inputs      []string
	Conditions  []Rule
	Prompt      string
	StateRules  string
	GateFiles   []string
}

func CheckRepo(root string, opts Options) (Result, error) {
	plan, err := buildPlanForRoot(root)
	if err != nil {
		return Result{}, err
	}

	var results []CheckResult
	for _, pc := range plan {
		res, err := runCheck(root, pc, opts)
		if err != nil {
			return Result{}, err
		}
		results = append(results, res)
	}

	return Result{Checks: results}, nil
}

func PlanRepo(root string) (Plan, error) {
	plan, err := buildPlanForRoot(root)
	if err != nil {
		return Plan{}, err
	}
	var out []PlannedCheck
	for _, pc := range plan {
		out = append(out, PlannedCheck{
			ID:          pc.Check.ID,
			Description: pc.Check.Description,
			Type:        pc.Check.Type,
			Phase:       pc.Check.Phase,
			PluginID:    pc.Plugin.Manifest.ID,
			Inputs:      pc.Check.Inputs,
			Conditions:  pc.Check.Conditions,
			Prompt:      pc.Check.Prompt,
			StateRules:  pc.Check.StateRules,
			GateFiles:   pc.Check.GateFiles,
		})
	}
	return Plan{Checks: out}, nil
}

func buildPlanForRoot(root string) ([]plannedCheck, error) {
	plugins, err := loadBuiltins()
	if err != nil {
		return nil, err
	}

	active := filterActivePlugins(root, plugins)
	plan, err := buildPlan(root, active)
	if err != nil {
		return nil, err
	}

	sortPlan(plan)
	return plan, nil
}

func filterActivePlugins(root string, plugins []Plugin) []Plugin {
	var active []Plugin
	for _, plugin := range plugins {
		if isPluginActive(root, plugin) {
			active = append(active, plugin)
		}
	}
	return active
}

func isPluginActive(root string, plugin Plugin) bool {
	if len(plugin.Manifest.Triggers) == 0 {
		return true
	}
	for _, trigger := range plugin.Manifest.Triggers {
		if evalTrigger(root, trigger) {
			return true
		}
	}
	return false
}

func evalTrigger(root string, trigger Trigger) bool {
	switch trigger.Type {
	case "path-exists":
		_, err := os.Stat(filepath.Join(root, trigger.Value))
		return err == nil
	case "glob-exists":
		matches, _ := filepath.Glob(filepath.Join(root, trigger.Value))
		return len(matches) > 0
	default:
		return false
	}
}

func buildPlan(root string, plugins []Plugin) ([]plannedCheck, error) {
	var plan []plannedCheck
	for _, plugin := range plugins {
		for _, check := range plugin.Manifest.Checks {
			ok, err := conditionsMet(root, check.Conditions)
			if err != nil {
				return nil, err
			}
			if !ok {
				continue
			}
			plan = append(plan, plannedCheck{Plugin: plugin, Check: check})
		}
	}
	return plan, nil
}

func conditionsMet(root string, rules []Rule) (bool, error) {
	for _, rule := range rules {
		res, err := evalRule(root, rule)
		if err != nil {
			return false, err
		}
		if !res.Passed {
			return false, nil
		}
	}
	return true, nil
}

func sortPlan(plan []plannedCheck) {
	phaseOrder := map[string]int{
		"frame":   1,
		"design":  2,
		"test":    3,
		"build":   4,
		"deploy":  5,
		"iterate": 6,
	}
	sort.Slice(plan, func(i, j int) bool {
		// Plugin priority (default 50)
		pi := plan[i].Plugin.Manifest.Priority
		pj := plan[j].Plugin.Manifest.Priority
		if pi == 0 {
			pi = 50
		}
		if pj == 0 {
			pj = 50
		}
		if pi != pj {
			return pi < pj
		}
		// Check priority (default 50)
		ci := plan[i].Check.Priority
		cj := plan[j].Check.Priority
		if ci == 0 {
			ci = 50
		}
		if cj == 0 {
			cj = 50
		}
		if ci != cj {
			return ci < cj
		}
		// Phase order
		phi := phaseOrder[plan[i].Check.Phase]
		phj := phaseOrder[plan[j].Check.Phase]
		if phi != phj {
			return phi < phj
		}
		// Alphabetical by ID
		return plan[i].Check.ID < plan[j].Check.ID
	})
}

func runCheck(root string, pc plannedCheck, opts Options) (CheckResult, error) {
	handler, ok := LookupCheckType(pc.Check.Type)
	if !ok {
		return CheckResult{}, fmt.Errorf("unknown check type: %s", pc.Check.Type)
	}
	cfg, err := handler.Decode(pc.Check)
	if err != nil {
		return CheckResult{}, err
	}
	def := CheckDefinition{
		ID:          pc.Check.ID,
		Description: pc.Check.Description,
		Type:        pc.Check.Type,
		Phase:       pc.Check.Phase,
		Priority:    pc.Check.Priority,
		Conditions:  pc.Check.Conditions,
		PluginID:    pc.Plugin.Manifest.ID,
	}
	result, err := handler.Run(root, def, cfg, opts, pc.Plugin)
	if err != nil {
		return CheckResult{}, err
	}
	return summarizeResult(result), nil
}
