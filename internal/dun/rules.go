package dun

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

type RuleEval struct {
	Passed  bool
	Message string
}

func evalRule(root string, rule Rule) (RuleEval, error) {
	switch rule.Type {
	case "path-exists":
		_, err := os.Stat(filepath.Join(root, rule.Path))
		if err == nil {
			return RuleEval{Passed: true}, nil
		}
		if os.IsNotExist(err) {
			return RuleEval{Passed: false, Message: fmt.Sprintf("missing path: %s", rule.Path)}, nil
		}
		return RuleEval{}, err
	case "path-missing":
		_, err := os.Stat(filepath.Join(root, rule.Path))
		if err == nil {
			return RuleEval{Passed: false, Message: fmt.Sprintf("path exists: %s", rule.Path)}, nil
		}
		if os.IsNotExist(err) {
			return RuleEval{Passed: true}, nil
		}
		return RuleEval{}, err
	case "glob-min-count":
		matches, err := filepath.Glob(filepath.Join(root, rule.Path))
		if err != nil {
			return RuleEval{}, err
		}
		if len(matches) >= rule.Expected {
			return RuleEval{Passed: true}, nil
		}
		return RuleEval{Passed: false, Message: fmt.Sprintf("glob %s expected >= %d, got %d", rule.Path, rule.Expected, len(matches))}, nil
	case "glob-max-count":
		matches, err := filepath.Glob(filepath.Join(root, rule.Path))
		if err != nil {
			return RuleEval{}, err
		}
		if len(matches) <= rule.Expected {
			return RuleEval{Passed: true}, nil
		}
		return RuleEval{Passed: false, Message: fmt.Sprintf("glob %s expected <= %d, got %d", rule.Path, rule.Expected, len(matches))}, nil
	case "pattern-count":
		return evalPatternCount(root, rule)
	case "unique-ids":
		return evalUniqueIDs(root, rule)
	case "cross-reference":
		return evalCrossReference(root, rule)
	default:
		return RuleEval{}, fmt.Errorf("unknown rule type: %s", rule.Type)
	}
}

func runRuleSet(root string, check Check) (CheckResult, error) {
	var fails []string
	var warns []string
	for _, rule := range check.Rules {
		res, err := evalRule(root, rule)
		if err != nil {
			return CheckResult{}, err
		}
		if res.Passed {
			continue
		}
		if rule.Severity == "warn" {
			warns = append(warns, res.Message)
		} else {
			fails = append(fails, res.Message)
		}
	}

	status := "pass"
	var detail string
	if len(fails) > 0 {
		status = "fail"
		detail = strings.Join(fails, "; ")
	} else if len(warns) > 0 {
		status = "warn"
		detail = strings.Join(warns, "; ")
	}

	signal := fmt.Sprintf("%d rules failed, %d warnings", len(fails), len(warns))
	if status == "pass" {
		signal = "all rules passed"
	}

	return CheckResult{
		ID:     check.ID,
		Status: status,
		Signal: signal,
		Detail: detail,
	}, nil
}

func evalPatternCount(root string, rule Rule) (RuleEval, error) {
	path := filepath.Join(root, rule.Path)
	content, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return RuleEval{Passed: false, Message: fmt.Sprintf("missing path: %s", rule.Path)}, nil
		}
		return RuleEval{}, err
	}
	re, err := regexp.Compile(rule.Pattern)
	if err != nil {
		return RuleEval{}, err
	}
	count := len(re.FindAll(content, -1))
	if count == rule.Expected {
		return RuleEval{Passed: true}, nil
	}
	return RuleEval{Passed: false, Message: fmt.Sprintf("pattern %s expected %d, got %d", rule.Pattern, rule.Expected, count)}, nil
}

func evalUniqueIDs(root string, rule Rule) (RuleEval, error) {
	path := filepath.Join(root, rule.Path)
	content, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return RuleEval{Passed: false, Message: fmt.Sprintf("missing path: %s", rule.Path)}, nil
		}
		return RuleEval{}, err
	}
	re, err := regexp.Compile(rule.Pattern)
	if err != nil {
		return RuleEval{}, err
	}
	matches := re.FindAllString(string(content), -1)
	if len(matches) == 0 {
		return RuleEval{Passed: true}, nil
	}
	sort.Strings(matches)
	for i := 1; i < len(matches); i++ {
		if matches[i] == matches[i-1] {
			return RuleEval{Passed: false, Message: fmt.Sprintf("duplicate id: %s", matches[i])}, nil
		}
	}
	return RuleEval{Passed: true}, nil
}

func evalCrossReference(root string, rule Rule) (RuleEval, error) {
	path := filepath.Join(root, rule.Path)
	content, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return RuleEval{Passed: false, Message: fmt.Sprintf("missing path: %s", rule.Path)}, nil
		}
		return RuleEval{}, err
	}
	if strings.Contains(string(content), rule.Pattern) {
		return RuleEval{Passed: true}, nil
	}
	return RuleEval{Passed: false, Message: fmt.Sprintf("missing reference: %s", rule.Pattern)}, nil
}
