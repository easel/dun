package dun

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEvalRulePathExistsAndMissing(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "exists.txt")
	writeFile(t, path, "ok")

	res, err := evalRule(root, Rule{Type: "path-exists", Path: "exists.txt"})
	if err != nil {
		t.Fatalf("path-exists: %v", err)
	}
	if !res.Passed {
		t.Fatalf("expected pass, got %v", res)
	}

	res, err = evalRule(root, Rule{Type: "path-missing", Path: "exists.txt"})
	if err != nil {
		t.Fatalf("path-missing: %v", err)
	}
	if res.Passed {
		t.Fatalf("expected fail when path exists")
	}

	res, err = evalRule(root, Rule{Type: "path-missing", Path: "missing.txt"})
	if err != nil {
		t.Fatalf("path-missing missing: %v", err)
	}
	if !res.Passed {
		t.Fatalf("expected pass when path missing")
	}

	if err := os.Chmod(root, 0000); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	defer os.Chmod(root, 0755)
	_, err = evalRule(root, Rule{Type: "path-exists", Path: "exists.txt"})
	if err == nil {
		t.Fatalf("expected stat error")
	}
	_, err = evalRule(root, Rule{Type: "path-missing", Path: "missing.txt"})
	if err == nil {
		t.Fatalf("expected stat error for path-missing")
	}
}

func TestEvalRuleGlobCounts(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "a.md"), "a")
	writeFile(t, filepath.Join(root, "b.md"), "b")

	res, err := evalRule(root, Rule{Type: "glob-min-count", Path: "*.md", Expected: 2})
	if err != nil {
		t.Fatalf("glob-min-count: %v", err)
	}
	if !res.Passed {
		t.Fatalf("expected pass, got %v", res)
	}

	res, err = evalRule(root, Rule{Type: "glob-max-count", Path: "*.md", Expected: 1})
	if err != nil {
		t.Fatalf("glob-max-count: %v", err)
	}
	if res.Passed {
		t.Fatalf("expected fail for max count")
	}
}

func TestEvalRuleGlobMaxCountError(t *testing.T) {
	_, err := evalRule(t.TempDir(), Rule{Type: "glob-max-count", Path: "[", Expected: 1})
	if err == nil {
		t.Fatalf("expected glob error")
	}
}

func TestEvalRulePatternCount(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "pattern.txt"), "foo foo")

	res, err := evalRule(root, Rule{Type: "pattern-count", Path: "pattern.txt", Pattern: "foo", Expected: 2})
	if err != nil {
		t.Fatalf("pattern-count: %v", err)
	}
	if !res.Passed {
		t.Fatalf("expected pass, got %v", res)
	}

	res, err = evalRule(root, Rule{Type: "pattern-count", Path: "pattern.txt", Pattern: "foo", Expected: 1})
	if err != nil {
		t.Fatalf("pattern-count fail: %v", err)
	}
	if res.Passed {
		t.Fatalf("expected fail for pattern count")
	}
}

func TestEvalRuleUniqueIDs(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "ids.txt"), "ID-1\nID-1\nID-2\n")

	res, err := evalRule(root, Rule{Type: "unique-ids", Path: "ids.txt", Pattern: "ID-[0-9]+"})
	if err != nil {
		t.Fatalf("unique-ids: %v", err)
	}
	if res.Passed {
		t.Fatalf("expected duplicate id fail")
	}

	writeFile(t, filepath.Join(root, "ids.txt"), "ID-1\nID-2\n")
	res, err = evalRule(root, Rule{Type: "unique-ids", Path: "ids.txt", Pattern: "ID-[0-9]+"})
	if err != nil {
		t.Fatalf("unique-ids pass: %v", err)
	}
	if !res.Passed {
		t.Fatalf("expected pass for unique ids")
	}

	writeFile(t, filepath.Join(root, "ids.txt"), "no matches")
	res, err = evalRule(root, Rule{Type: "unique-ids", Path: "ids.txt", Pattern: "ID-[0-9]+"})
	if err != nil {
		t.Fatalf("unique-ids empty: %v", err)
	}
	if !res.Passed {
		t.Fatalf("expected pass for no matches")
	}
}

func TestEvalRuleUniqueIDsReadError(t *testing.T) {
	res, err := evalRule(t.TempDir(), Rule{Type: "unique-ids", Path: "missing.txt", Pattern: "ID-[0-9]+"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Passed {
		t.Fatalf("expected missing file to fail rule")
	}
}

func TestEvalRuleCrossReference(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "doc.md"), "see REF-1")

	res, err := evalRule(root, Rule{Type: "cross-reference", Path: "doc.md", Pattern: "REF-1"})
	if err != nil {
		t.Fatalf("cross-reference: %v", err)
	}
	if !res.Passed {
		t.Fatalf("expected pass")
	}

	res, err = evalRule(root, Rule{Type: "cross-reference", Path: "doc.md", Pattern: "MISSING"})
	if err != nil {
		t.Fatalf("cross-reference missing: %v", err)
	}
	if res.Passed {
		t.Fatalf("expected fail for missing reference")
	}
}

func TestEvalRuleUnknownType(t *testing.T) {
	_, err := evalRule(t.TempDir(), Rule{Type: "unknown"})
	if err == nil {
		t.Fatalf("expected error for unknown rule type")
	}
}

func TestRunRuleSetStatuses(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "ok.txt"), "ok")

	passDef := CheckDefinition{ID: "rules-pass"}
	passConfig := RuleSetConfig{Rules: []Rule{{Type: "path-exists", Path: "ok.txt"}}}
	res, err := runRuleSet(root, passDef, passConfig)
	if err != nil {
		t.Fatalf("rule-set pass: %v", err)
	}
	if res.Status != "pass" {
		t.Fatalf("expected pass, got %s", res.Status)
	}

	warnDef := CheckDefinition{ID: "rules-warn"}
	warnConfig := RuleSetConfig{Rules: []Rule{{Type: "path-missing", Path: "ok.txt", Severity: "warn"}}}
	res, err = runRuleSet(root, warnDef, warnConfig)
	if err != nil {
		t.Fatalf("rule-set warn: %v", err)
	}
	if res.Status != "warn" {
		t.Fatalf("expected warn, got %s", res.Status)
	}

	failDef := CheckDefinition{ID: "rules-fail"}
	failConfig := RuleSetConfig{Rules: []Rule{{Type: "path-missing", Path: "ok.txt"}}}
	res, err = runRuleSet(root, failDef, failConfig)
	if err != nil {
		t.Fatalf("rule-set fail: %v", err)
	}
	if res.Status != "fail" {
		t.Fatalf("expected fail, got %s", res.Status)
	}

	badDef := CheckDefinition{ID: "rules-error"}
	badConfig := RuleSetConfig{Rules: []Rule{{Type: "pattern-count", Path: "pattern.txt", Pattern: "("}}}
	writeFile(t, filepath.Join(root, "pattern.txt"), "x")
	if _, err := runRuleSet(root, badDef, badConfig); err == nil {
		t.Fatalf("expected error from bad rule")
	}
}

func TestEvalRulePatternCountErrors(t *testing.T) {
	root := t.TempDir()
	res, err := evalPatternCount(root, Rule{Path: "missing.txt", Pattern: "("})
	if err != nil {
		t.Fatalf("unexpected error for missing file: %v", err)
	}
	if res.Passed {
		t.Fatalf("expected missing file to fail rule")
	}

	writeFile(t, filepath.Join(root, "pattern.txt"), "x")
	_, err = evalPatternCount(root, Rule{Path: "pattern.txt", Pattern: "("})
	if err == nil {
		t.Fatalf("expected error for invalid regex")
	}
}

func TestEvalRuleUniqueIDsRegexError(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "ids.txt"), "ID-1")
	_, err := evalUniqueIDs(root, Rule{Path: "ids.txt", Pattern: "("})
	if err == nil {
		t.Fatalf("expected regex error")
	}
}

func TestEvalRuleCrossReferenceErrors(t *testing.T) {
	root := t.TempDir()
	res, err := evalCrossReference(root, Rule{Path: "missing.txt", Pattern: "REF"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Passed {
		t.Fatalf("expected missing file to fail rule")
	}
}

func TestEvalRuleUnknownGlobErrors(t *testing.T) {
	root := t.TempDir()
	_, err := evalRule(root, Rule{Type: "glob-min-count", Path: "[", Expected: 1})
	if err == nil {
		t.Fatalf("expected glob error")
	}
}
