package dun

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDocDagMissingRequiredRoot(t *testing.T) {
	root := t.TempDir()
	graphDir := filepath.Join(root, ".dun", "graphs")
	if err := os.MkdirAll(graphDir, 0755); err != nil {
		t.Fatalf("mkdir graphs: %v", err)
	}
	graph := `required_roots:
  - doc.prd
id_map:
  doc.prd: docs/prd.md
`
	if err := os.WriteFile(filepath.Join(graphDir, "test.yaml"), []byte(graph), 0644); err != nil {
		t.Fatalf("write graph: %v", err)
	}

	graphState, err := buildDocGraph(root)
	if err != nil {
		t.Fatalf("build graph: %v", err)
	}
	missing := graphState.MissingRequiredRoots()
	if len(missing) != 1 || missing[0] != "doc.prd" {
		t.Fatalf("expected missing doc.prd, got %#v", missing)
	}
}

func TestDefaultSelectorsWithDependsOn(t *testing.T) {
	nodes := map[string]*DocNode{
		"parent": {ID: "parent", Path: "docs/parent.md"},
	}
	graph := &DocGraph{Nodes: nodes}

	child := &DocNode{
		ID:        "child",
		DependsOn: []string{"parent", "other"},
	}

	selectors := graph.defaultSelectors(child)
	if len(selectors) != 2 {
		t.Fatalf("expected 2 selectors, got %d: %v", len(selectors), selectors)
	}
	if selectors[0] != "node:parent" || selectors[1] != "node:other" {
		t.Fatalf("unexpected selectors: %v", selectors)
	}
}

func TestDefaultSelectorsNilNode(t *testing.T) {
	graph := &DocGraph{}
	selectors := graph.defaultSelectors(nil)
	if selectors != nil {
		t.Fatalf("expected nil selectors for nil node, got %v", selectors)
	}
}

func TestDefaultSelectorsNoDeps(t *testing.T) {
	graph := &DocGraph{}
	node := &DocNode{ID: "nodeps", DependsOn: nil}
	selectors := graph.defaultSelectors(node)
	if selectors != nil {
		t.Fatalf("expected nil selectors for node without deps, got %v", selectors)
	}
}

func TestPromptForIDFromNode(t *testing.T) {
	nodes := map[string]*DocNode{
		"doc.custom": {
			ID:     "doc.custom",
			Prompt: "prompts/custom.md",
		},
	}
	graph := &DocGraph{Nodes: nodes}

	prompt := graph.promptForID("doc.custom")
	if prompt != "prompts/custom.md" {
		t.Fatalf("expected prompts/custom.md, got %q", prompt)
	}
}

func TestPromptForIDFromDefaults(t *testing.T) {
	graph := &DocGraph{
		Nodes:          map[string]*DocNode{},
		PromptDefaults: map[string]string{"doc.default": "prompts/default.md"},
	}

	prompt := graph.promptForID("doc.default")
	if prompt != "prompts/default.md" {
		t.Fatalf("expected prompts/default.md, got %q", prompt)
	}
}

func TestPromptForIDFallsBackToDefaultPrompt(t *testing.T) {
	graph := &DocGraph{
		Nodes:          map[string]*DocNode{},
		PromptDefaults: map[string]string{},
		DefaultPrompt:  "prompts/fallback.md",
	}

	prompt := graph.promptForID("unknown")
	if prompt != "prompts/fallback.md" {
		t.Fatalf("expected prompts/fallback.md, got %q", prompt)
	}
}

func TestPromptForIDReturnsEmpty(t *testing.T) {
	graph := &DocGraph{
		Nodes:          map[string]*DocNode{},
		PromptDefaults: map[string]string{},
		DefaultPrompt:  "",
	}

	prompt := graph.promptForID("unknown")
	if prompt != "" {
		t.Fatalf("expected empty string, got %q", prompt)
	}
}

func TestExpectedPathFromNode(t *testing.T) {
	nodes := map[string]*DocNode{
		"doc.known": {ID: "doc.known", Path: "docs/known.md"},
	}
	graph := &DocGraph{Nodes: nodes}

	path := graph.expectedPath("doc.known")
	if path != "docs/known.md" {
		t.Fatalf("expected docs/known.md, got %q", path)
	}
}

func TestExpectedPathFromIDMap(t *testing.T) {
	graph := &DocGraph{
		Nodes: map[string]*DocNode{},
		IDMap: map[string]string{"DOC-{id}": "docs/DOC-{id}.md"},
	}

	path := graph.expectedPath("DOC-123")
	if path != "docs/DOC-123.md" {
		t.Fatalf("expected docs/DOC-123.md, got %q", path)
	}
}

func TestExpectedPathUnknown(t *testing.T) {
	graph := &DocGraph{
		Nodes: map[string]*DocNode{},
		IDMap: map[string]string{},
	}

	path := graph.expectedPath("unknown")
	if path != "" {
		t.Fatalf("expected empty string, got %q", path)
	}
}

func TestStaleNodesWithMissingParent(t *testing.T) {
	nodes := map[string]*DocNode{
		"child": {
			ID:        "child",
			DependsOn: []string{"nonexistent"},
			Review:    DocReview{Deps: map[string]string{}},
		},
	}
	graph := &DocGraph{Nodes: nodes}

	stale := graph.StaleNodes()
	if len(stale) != 0 {
		t.Fatalf("expected no stale nodes when parent missing, got %v", stale)
	}
}

func TestStaleNodesCascade(t *testing.T) {
	nodes := map[string]*DocNode{
		"grandparent": {
			ID:     "grandparent",
			Hash:   "newhash",
			Review: DocReview{Deps: map[string]string{}},
		},
		"parent": {
			ID:        "parent",
			DependsOn: []string{"grandparent"},
			Hash:      "parenthash",
			Review:    DocReview{Deps: map[string]string{"grandparent": "oldhash"}},
		},
		"child": {
			ID:        "child",
			DependsOn: []string{"parent"},
			Hash:      "childhash",
			Review:    DocReview{Deps: map[string]string{"parent": "parenthash"}},
		},
	}
	graph := &DocGraph{Nodes: nodes}

	stale := graph.StaleNodes()
	if len(stale) != 2 {
		t.Fatalf("expected 2 stale nodes (parent and child), got %d: %v", len(stale), stale)
	}
}

func TestStaleFrontierSkipsDownstream(t *testing.T) {
	nodes := map[string]*DocNode{
		"grandparent": {
			ID:     "grandparent",
			Hash:   "newhash",
			Review: DocReview{Deps: map[string]string{}},
		},
		"parent": {
			ID:        "parent",
			DependsOn: []string{"grandparent"},
			Hash:      "parenthash",
			Review:    DocReview{Deps: map[string]string{"grandparent": "oldhash"}},
		},
		"child": {
			ID:        "child",
			DependsOn: []string{"parent"},
			Hash:      "childhash",
			Review:    DocReview{Deps: map[string]string{"parent": "parenthash"}},
		},
	}
	graph := &DocGraph{Nodes: nodes}

	stale := graph.StaleNodes()
	frontier := graph.StaleFrontier(stale)
	if len(frontier) != 1 || frontier[0] != "parent" {
		t.Fatalf("expected frontier to include only parent, got %v", frontier)
	}
}

func TestBuildIssuesMissingAndStale(t *testing.T) {
	nodes := map[string]*DocNode{
		"stale.doc": {ID: "stale.doc", Path: "docs/stale.md"},
	}
	graph := &DocGraph{Nodes: nodes}

	issues := graph.buildIssues([]string{"missing.doc"}, []string{"stale.doc"})
	if len(issues) != 2 {
		t.Fatalf("expected 2 issues, got %d", len(issues))
	}
	if issues[0].ID != "missing:missing.doc" {
		t.Fatalf("expected missing issue first, got %s", issues[0].ID)
	}
	if issues[1].ID != "stale:stale.doc" {
		t.Fatalf("expected stale issue second, got %s", issues[1].ID)
	}
}

func TestLoadGraphDefaultsMultipleFiles(t *testing.T) {
	root := t.TempDir()
	graphDir := filepath.Join(root, ".dun", "graphs")
	if err := os.MkdirAll(graphDir, 0755); err != nil {
		t.Fatalf("mkdir graphs: %v", err)
	}

	graph1 := `required_roots:
  - root.one
id_map:
  root.one: docs/one.md
`
	graph2 := `required_roots:
  - root.two
  - root.one
id_map:
  root.two: docs/two.md
default_prompt: prompts/default.md
`

	if err := os.WriteFile(filepath.Join(graphDir, "a.yaml"), []byte(graph1), 0644); err != nil {
		t.Fatalf("write a.yaml: %v", err)
	}
	if err := os.WriteFile(filepath.Join(graphDir, "b.yaml"), []byte(graph2), 0644); err != nil {
		t.Fatalf("write b.yaml: %v", err)
	}

	defaults, issues := loadGraphDefaults(root)
	if len(issues) != 0 {
		t.Fatalf("expected no issues, got %v", issues)
	}
	if len(defaults.RequiredRoots) != 2 {
		t.Fatalf("expected 2 roots (deduplicated), got %d: %v", len(defaults.RequiredRoots), defaults.RequiredRoots)
	}
	if len(defaults.IDMap) != 2 {
		t.Fatalf("expected 2 id map entries, got %d", len(defaults.IDMap))
	}
	if defaults.DefaultPrompt != "prompts/default.md" {
		t.Fatalf("expected default prompt, got %q", defaults.DefaultPrompt)
	}
}

func TestLoadGraphDefaultsWithYmlExtension(t *testing.T) {
	root := t.TempDir()
	graphDir := filepath.Join(root, ".dun", "graphs")
	if err := os.MkdirAll(graphDir, 0755); err != nil {
		t.Fatalf("mkdir graphs: %v", err)
	}

	graph := `required_roots:
  - yml.doc
`
	if err := os.WriteFile(filepath.Join(graphDir, "test.yml"), []byte(graph), 0644); err != nil {
		t.Fatalf("write test.yml: %v", err)
	}

	defaults, issues := loadGraphDefaults(root)
	if len(issues) != 0 {
		t.Fatalf("expected no issues, got %v", issues)
	}
	if len(defaults.RequiredRoots) != 1 || defaults.RequiredRoots[0] != "yml.doc" {
		t.Fatalf("expected yml.doc, got %v", defaults.RequiredRoots)
	}
}

func TestLoadGraphDefaultsInvalidYAMLProducesIssue(t *testing.T) {
	root := t.TempDir()
	graphDir := filepath.Join(root, ".dun", "graphs")
	if err := os.MkdirAll(graphDir, 0755); err != nil {
		t.Fatalf("mkdir graphs: %v", err)
	}
	if err := os.WriteFile(filepath.Join(graphDir, "bad.yaml"), []byte("invalid: [yaml"), 0644); err != nil {
		t.Fatalf("write bad.yaml: %v", err)
	}

	_, issues := loadGraphDefaults(root)
	if len(issues) != 1 {
		t.Fatalf("expected 1 issue, got %d", len(issues))
	}
	if issues[0].ID != "invalid-graph" {
		t.Fatalf("expected invalid-graph issue, got %q", issues[0].ID)
	}
}

func TestBuildDocGraphInvalidFrontmatterProducesIssue(t *testing.T) {
	root := t.TempDir()
	docDir := filepath.Join(root, "docs")
	if err := os.MkdirAll(docDir, 0755); err != nil {
		t.Fatalf("mkdir docs: %v", err)
	}
	content := `---
dun:
  id: [invalid yaml
---
# Body`
	if err := os.WriteFile(filepath.Join(docDir, "bad.md"), []byte(content), 0644); err != nil {
		t.Fatalf("write bad.md: %v", err)
	}

	graph, err := buildDocGraph(root)
	if err != nil {
		t.Fatalf("build graph: %v", err)
	}
	if countInvalidDocIssues(graph.Issues) != 1 {
		t.Fatalf("expected invalid issue, got %#v", graph.Issues)
	}
}

func TestBuildDocGraphSkipsParkingLotDocs(t *testing.T) {
	root := t.TempDir()
	parkedDir := filepath.Join(root, "docs", "helix", "02-design", "adr")
	if err := os.MkdirAll(parkedDir, 0755); err != nil {
		t.Fatalf("mkdir parked dir: %v", err)
	}
	activeDir := filepath.Join(root, "docs", "helix", "01-frame")
	if err := os.MkdirAll(activeDir, 0755); err != nil {
		t.Fatalf("mkdir active dir: %v", err)
	}

	parked := `---
dun:
  id: ADR-008
  parking_lot: true
  depends_on:
    - PRD-001
---
# Parked ADR
`
	if err := os.WriteFile(filepath.Join(parkedDir, "ADR-008-parked.md"), []byte(parked), 0644); err != nil {
		t.Fatalf("write parked: %v", err)
	}

	active := `---
dun:
  id: PRD-001
---
# PRD
`
	if err := os.WriteFile(filepath.Join(activeDir, "prd.md"), []byte(active), 0644); err != nil {
		t.Fatalf("write active: %v", err)
	}

	graph, err := buildDocGraph(root)
	if err != nil {
		t.Fatalf("build graph: %v", err)
	}
	if _, ok := graph.Nodes["ADR-008"]; ok {
		t.Fatalf("expected parking lot doc to be skipped")
	}
	if _, ok := graph.Nodes["PRD-001"]; !ok {
		t.Fatalf("expected active doc to be included")
	}
}
