package dun

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

type plannedCheck struct {
	Plugin Plugin
	Check  Check
}

func CheckRepo(root string, opts Options) (Result, error) {
	plugins, err := LoadBuiltins()
	if err != nil {
		return Result{}, err
	}

	active := filterActivePlugins(root, plugins)
	plan, err := buildPlan(root, active)
	if err != nil {
		return Result{}, err
	}

	sortPlan(plan)

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
		pi := phaseOrder[plan[i].Check.Phase]
		pj := phaseOrder[plan[j].Check.Phase]
		if pi != pj {
			return pi < pj
		}
		return plan[i].Check.ID < plan[j].Check.ID
	})
}

func runCheck(root string, pc plannedCheck, opts Options) (CheckResult, error) {
	switch pc.Check.Type {
	case "rule-set":
		return runRuleSet(root, pc.Check)
	case "state-rules":
		return runStateRules(root, pc.Plugin, pc.Check)
	case "agent":
		return runAgentCheck(root, pc.Plugin, pc.Check, opts)
	case "command":
		return CheckResult{}, errors.New("command checks not implemented")
	default:
		return CheckResult{}, fmt.Errorf("unknown check type: %s", pc.Check.Type)
	}
}
