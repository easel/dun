package dun

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunGateCheckMissingGateFiles(t *testing.T) {
	plugin := Plugin{}
	_, err := runGateCheck(".", plugin, Check{ID: "gates"})
	if err == nil {
		t.Fatalf("expected error for missing gate files")
	}
}

func TestRunGateCheckPassesWhenSatisfied(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "gate.yml"), "input_gates:\n  - criteria: \"Gate\"\n    required: true\n    evidence: \"docs/a.md#Section\"\n")
	if err := os.MkdirAll(filepath.Join(dir, "docs"), 0755); err != nil {
		t.Fatalf("mkdir docs: %v", err)
	}
	writeFile(t, filepath.Join(dir, "docs", "a.md"), "# Section\n")

	plugin := Plugin{FS: os.DirFS(dir), Base: "."}
	check := Check{ID: "gates", GateFiles: []string{"gate.yml"}}
	res, err := runGateCheck(dir, plugin, check)
	if err != nil {
		t.Fatalf("run gate check: %v", err)
	}
	if res.Status != "pass" {
		t.Fatalf("expected pass, got %s", res.Status)
	}
}

func TestRunGateCheckFailsWhenRequiredMissing(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "gate.yml"), "input_gates:\n  - criteria: \"Gate\"\n    required: true\n    evidence: \"docs/missing.md\"\n")

	plugin := Plugin{FS: os.DirFS(dir), Base: "."}
	check := Check{ID: "gates", GateFiles: []string{"gate.yml"}}
	res, err := runGateCheck(dir, plugin, check)
	if err != nil {
		t.Fatalf("run gate check: %v", err)
	}
	if res.Status != "fail" {
		t.Fatalf("expected fail, got %s", res.Status)
	}
	if len(res.Issues) == 0 {
		t.Fatalf("expected issues")
	}
}

func TestRunGateCheckWarnsWhenOptionalMissing(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "gate.yml"), "input_gates:\n  - criteria: \"Gate\"\n    required: false\n    evidence: \"docs/missing.md\"\n")
	plugin := Plugin{FS: os.DirFS(dir), Base: "."}
	check := Check{ID: "gates", GateFiles: []string{"gate.yml"}}
	res, err := runGateCheck(dir, plugin, check)
	if err != nil {
		t.Fatalf("run gate check: %v", err)
	}
	if res.Status != "warn" {
		t.Fatalf("expected warn, got %s", res.Status)
	}
}

func TestRunGateCheckMissingGateFile(t *testing.T) {
	dir := t.TempDir()
	plugin := Plugin{FS: os.DirFS(dir), Base: "."}
	check := Check{ID: "gates", GateFiles: []string{"missing.yml"}}
	if _, err := runGateCheck(dir, plugin, check); err == nil {
		t.Fatalf("expected missing gate file error")
	}
}

func TestRunGateCheckEvidenceError(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "gate.yml"), "input_gates:\n  - criteria: \"Gate\"\n    required: true\n    evidence: \"docs#Section\"\n")
	if err := os.MkdirAll(filepath.Join(dir, "docs"), 0755); err != nil {
		t.Fatalf("mkdir docs: %v", err)
	}
	plugin := Plugin{FS: os.DirFS(dir), Base: "."}
	check := Check{ID: "gates", GateFiles: []string{"gate.yml"}}
	if _, err := runGateCheck(dir, plugin, check); err == nil {
		t.Fatalf("expected evidence error")
	}
}

func TestLoadGateFileErrors(t *testing.T) {
	dir := t.TempDir()
	plugin := Plugin{FS: os.DirFS(dir), Base: "."}
	if _, err := loadGateFile(plugin, "missing.yml"); err == nil {
		t.Fatalf("expected read error")
	}
	writeFile(t, filepath.Join(dir, "bad.yml"), ":")
	if _, err := loadGateFile(plugin, "bad.yml"); err == nil {
		t.Fatalf("expected parse error")
	}
}

func TestEvidenceMissingWithAnchor(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "doc.md")
	writeFile(t, path, "# Title\n")

	missing, missingAnchor, err := evidenceMissing(path, "Missing")
	if err != nil {
		t.Fatalf("evidence missing: %v", err)
	}
	if !missing || !missingAnchor {
		t.Fatalf("expected missing anchor")
	}

	missing, missingAnchor, err = evidenceMissing(path, "Title")
	if err != nil {
		t.Fatalf("evidence missing: %v", err)
	}
	if missing || missingAnchor {
		t.Fatalf("expected anchor present")
	}
}

func TestEvidenceMissingNoAnchor(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "missing.md")
	missing, missingAnchor, err := evidenceMissing(path, "")
	if err != nil {
		t.Fatalf("evidence missing: %v", err)
	}
	if !missing || missingAnchor {
		t.Fatalf("expected missing without anchor")
	}
}

func TestEvidenceMissingAnchorError(t *testing.T) {
	dir := t.TempDir()
	missing, missingAnchor, err := evidenceMissing(dir, "Anchor")
	if err == nil {
		t.Fatalf("expected anchor read error")
	}
	if missing || missingAnchor {
		t.Fatalf("expected no missing flags on error")
	}
}

func TestSplitEvidence(t *testing.T) {
	path, anchor := splitEvidence("docs/a.md#Section One")
	if path != "docs/a.md" || anchor != "Section One" {
		t.Fatalf("unexpected split: %s %s", path, anchor)
	}
	path, anchor = splitEvidence("docs/a.md")
	if anchor != "" {
		t.Fatalf("expected empty anchor")
	}
}

func TestHasMarkdownAnchor(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "doc.md")
	writeFile(t, path, "#\n## Hello World\n")
	found, err := hasMarkdownAnchor(path, "Hello World")
	if err != nil {
		t.Fatalf("has anchor: %v", err)
	}
	if !found {
		t.Fatalf("expected anchor found")
	}

	_, err = hasMarkdownAnchor(dir, "Nope")
	if err == nil {
		t.Fatalf("expected error for directory read")
	}
}

func TestSlugify(t *testing.T) {
	if slugify("Hello, World!") != "hello-world" {
		t.Fatalf("unexpected slugify")
	}
	if slugify("  123 ABC ") != "123-abc" {
		t.Fatalf("unexpected slugify for numbers")
	}
}

func TestBuildGateActionBranches(t *testing.T) {
	if !strings.Contains(buildGateAction("docs/a.md", "Section", true, ""), "Add section") {
		t.Fatalf("expected add section action")
	}
	if !strings.Contains(buildGateAction("docs/a.md", "Section", false, ""), "Create docs/a.md") {
		t.Fatalf("expected create file action")
	}
	if !strings.Contains(buildGateAction("docs/", "", false, ""), "Create directory") {
		t.Fatalf("expected create directory action")
	}
	if !strings.Contains(buildGateAction("docs/a.md", "", false, "Gate"), "(Gate)") {
		t.Fatalf("expected criteria in action")
	}
}

func TestSortedHelpers(t *testing.T) {
	keys := sortedKeys(map[string]string{"b": "b", "a": "a"})
	if len(keys) != 2 || keys[0] != "a" {
		t.Fatalf("expected sorted keys")
	}
	issues := sortedIssues(map[string]Issue{
		"b": {ID: "b"},
		"a": {ID: "a"},
	})
	if len(issues) != 2 || issues[0].ID != "a" {
		t.Fatalf("expected sorted issues")
	}
}
