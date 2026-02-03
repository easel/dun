package dun

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/template"

	"gopkg.in/yaml.v3"
)

type DocNode struct {
	ID          string
	Path        string
	AbsPath     string
	DependsOn   []string
	Prompt      string
	Inputs      []string
	Review      DocReview
	Frontmatter *yaml.Node
	Body        string
	Content     string
	Hash        string
}

type DocGraph struct {
	Root           string
	Nodes          map[string]*DocNode
	NodesByPath    map[string]*DocNode
	RequiredRoots  []string
	IDMap          map[string]string
	PromptDefaults map[string]string
	DefaultPrompt  string
	Issues         []Issue
}

type graphDefaults struct {
	RequiredRoots  []string          `yaml:"required_roots"`
	IDMap          map[string]string `yaml:"id_map"`
	PromptDefaults map[string]string `yaml:"prompt_defaults"`
	DefaultPrompt  string            `yaml:"default_prompt"`
}

type docPromptContext struct {
	DocID   string
	DocPath string
	Reason  string
	Inputs  []PromptInput
}

func runDocDagCheck(root string, plugin Plugin, check Check) (CheckResult, error) {
	graph, err := buildDocGraph(root)
	if err != nil {
		return CheckResult{}, err
	}

	missing := graph.MissingRequiredRoots()
	staleAll := graph.StaleNodes()
	invalidCount := countInvalidDocIssues(graph.Issues)
	stale := staleAll
	if len(missing) > 0 {
		stale = nil
	} else {
		stale = graph.StaleFrontier(staleAll)
	}

	if len(missing) == 0 && len(staleAll) == 0 && invalidCount == 0 {
		return CheckResult{ID: check.ID, Status: "pass", Signal: "all docs up to date"}, nil
	}

	issues := append([]Issue(nil), graph.Issues...)
	issues = append(issues, graph.buildIssues(missing, stale)...)

	prompt, err := graph.buildPromptEnvelope(root, plugin, check, missing, stale)
	if err != nil {
		return CheckResult{}, err
	}

	status := "warn"
	if invalidCount > 0 || len(missing) > 0 {
		status = "fail"
	}

	signal := buildDocDagSignal(invalidCount, len(missing), len(stale))
	detail := "review required"
	if invalidCount > 0 {
		detail = "fix invalid frontmatter or graph files"
	} else if len(missing) > 0 && len(staleAll) > 0 {
		detail = fmt.Sprintf("review required (%d stale deferred until missing roots resolved)", len(staleAll))
	} else if len(staleAll) != len(stale) {
		detail = fmt.Sprintf("review required (%d stale total, %d actionable)", len(staleAll), len(stale))
	}
	next := "Update missing/stale docs and run dun stamp"
	if invalidCount > 0 {
		next = "Fix invalid frontmatter or graph files"
	}

	return CheckResult{
		ID:     check.ID,
		Status: status,
		Signal: signal,
		Detail: detail,
		Next:   next,
		Prompt: prompt,
		Issues: issues,
	}, nil
}

func buildDocGraph(root string) (*DocGraph, error) {
	defaults, graphIssues := loadGraphDefaults(root)
	nodes, nodesByPath, nodeIssues, err := loadDocNodes(root)
	if err != nil {
		return nil, err
	}

	return &DocGraph{
		Root:           root,
		Nodes:          nodes,
		NodesByPath:    nodesByPath,
		RequiredRoots:  defaults.RequiredRoots,
		IDMap:          defaults.IDMap,
		PromptDefaults: defaults.PromptDefaults,
		DefaultPrompt:  defaults.DefaultPrompt,
		Issues:         append(graphIssues, nodeIssues...),
	}, nil
}

func loadGraphDefaults(root string) (graphDefaults, []Issue) {
	var out graphDefaults
	paths := append(globGraphFiles(root, "*.yaml"), globGraphFiles(root, "*.yml")...)
	if len(paths) == 0 {
		return out, nil
	}

	seenRequired := make(map[string]bool)
	out.IDMap = make(map[string]string)
	out.PromptDefaults = make(map[string]string)
	var issues []Issue

	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			issues = append(issues, invalidGraphIssue(root, path, err))
			continue
		}
		var cfg graphDefaults
		if err := yaml.Unmarshal(data, &cfg); err != nil {
			issues = append(issues, invalidGraphIssue(root, path, err))
			continue
		}
		for _, id := range cfg.RequiredRoots {
			if seenRequired[id] {
				continue
			}
			seenRequired[id] = true
			out.RequiredRoots = append(out.RequiredRoots, id)
		}
		for key, value := range cfg.IDMap {
			out.IDMap[key] = value
		}
		for key, value := range cfg.PromptDefaults {
			out.PromptDefaults[key] = value
		}
		if cfg.DefaultPrompt != "" {
			out.DefaultPrompt = cfg.DefaultPrompt
		}
	}

	sort.Strings(out.RequiredRoots)
	return out, issues
}

func globGraphFiles(root, pattern string) []string {
	pattern = filepath.Join(root, ".dun", "graphs", pattern)
	matches, err := filepath.Glob(pattern)
	if err != nil || len(matches) == 0 {
		return nil
	}
	sort.Strings(matches)
	return matches
}

func loadDocNodes(root string) (map[string]*DocNode, map[string]*DocNode, []Issue, error) {
	nodes := make(map[string]*DocNode)
	nodesByPath := make(map[string]*DocNode)
	var issues []Issue

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			issues = append(issues, invalidFrontmatterIssue(root, path, err))
			return nil
		}
		if d.IsDir() {
			if shouldSkipDocDir(path, d.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(d.Name()) != ".md" {
			return nil
		}
		content, err := os.ReadFile(path)
		if err != nil {
			issues = append(issues, invalidFrontmatterIssue(root, path, err))
			return nil
		}
		frontmatter, body, err := ParseFrontmatter(content)
		if err != nil {
			issues = append(issues, invalidFrontmatterIssue(root, path, err))
			return nil
		}
		if !frontmatter.HasFrontmatter || strings.TrimSpace(frontmatter.Dun.ID) == "" {
			return nil
		}
		if frontmatter.Dun.ParkingLot {
			return nil
		}
		if frontmatter.Dun.Review.Deps == nil {
			frontmatter.Dun.Review.Deps = make(map[string]string)
		}
		if _, exists := nodes[frontmatter.Dun.ID]; exists {
			return fmt.Errorf("duplicate doc id: %s", frontmatter.Dun.ID)
		}
		rel, err := relPath(root, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		hash, err := HashDocument(frontmatter.Raw, body)
		if err != nil {
			return err
		}
		node := &DocNode{
			ID:          frontmatter.Dun.ID,
			Path:        rel,
			AbsPath:     path,
			DependsOn:   append([]string(nil), frontmatter.Dun.DependsOn...),
			Prompt:      frontmatter.Dun.Prompt,
			Inputs:      append([]string(nil), frontmatter.Dun.Inputs...),
			Review:      frontmatter.Dun.Review,
			Frontmatter: frontmatter.Raw,
			Body:        body,
			Content:     string(content),
			Hash:        hash,
		}
		nodes[node.ID] = node
		nodesByPath[node.Path] = node
		return nil
	})

	if err != nil {
		return nil, nil, nil, err
	}
	return nodes, nodesByPath, issues, nil
}

func shouldSkipDocDir(path, name string) bool {
	switch name {
	case ".git", ".dun", "node_modules", "vendor", "testdata":
		return true
	}
	return false
}

func (g *DocGraph) MissingRequiredRoots() []string {
	var missing []string
	for _, id := range g.RequiredRoots {
		if g.Nodes[id] == nil {
			missing = append(missing, id)
		}
	}
	sort.Strings(missing)
	return missing
}

func (g *DocGraph) StaleNodes() []string {
	stale := make(map[string]bool)
	for _, node := range g.Nodes {
		for _, parentID := range node.DependsOn {
			parent := g.Nodes[parentID]
			if parent == nil {
				continue
			}
			stamp := node.Review.Deps[parentID]
			if stamp == "" || stamp != parent.Hash {
				stale[node.ID] = true
				break
			}
		}
	}

	children := g.childrenByParent()
	queue := make([]string, 0, len(stale))
	for id := range stale {
		queue = append(queue, id)
	}
	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
		for _, child := range children[id] {
			if stale[child] {
				continue
			}
			stale[child] = true
			queue = append(queue, child)
		}
	}

	ids := make([]string, 0, len(stale))
	for id := range stale {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

func (g *DocGraph) StaleFrontier(stale []string) []string {
	if len(stale) == 0 {
		return nil
	}
	staleSet := make(map[string]bool, len(stale))
	for _, id := range stale {
		staleSet[id] = true
	}

	var frontier []string
	for _, id := range stale {
		node := g.Nodes[id]
		if node == nil {
			continue
		}
		hasStaleParent := false
		for _, parentID := range node.DependsOn {
			if staleSet[parentID] {
				hasStaleParent = true
				break
			}
		}
		if !hasStaleParent {
			frontier = append(frontier, id)
		}
	}
	sort.Strings(frontier)
	return frontier
}

func (g *DocGraph) childrenByParent() map[string][]string {
	children := make(map[string][]string)
	for _, node := range g.Nodes {
		for _, parentID := range node.DependsOn {
			children[parentID] = append(children[parentID], node.ID)
		}
	}
	for id := range children {
		sort.Strings(children[id])
	}
	return children
}

func (g *DocGraph) buildIssues(missing, stale []string) []Issue {
	issues := make([]Issue, 0, len(missing)+len(stale))
	for _, id := range missing {
		path := g.expectedPath(id)
		issues = append(issues, Issue{
			ID:      "missing:" + id,
			Summary: fmt.Sprintf("Missing required doc %s", id),
			Path:    path,
		})
	}
	for _, id := range stale {
		node := g.Nodes[id]
		path := ""
		if node != nil {
			path = node.Path
		}
		issues = append(issues, Issue{
			ID:      "stale:" + id,
			Summary: fmt.Sprintf("Doc %s is stale", id),
			Path:    path,
		})
	}
	return issues
}

func (g *DocGraph) buildPromptEnvelope(root string, plugin Plugin, check Check, missing, stale []string) (*PromptEnvelope, error) {
	var targetID string
	var reason string
	if len(missing) > 0 {
		targetID = missing[0]
		reason = "missing"
	} else if len(stale) > 0 {
		targetID = stale[0]
		reason = "stale"
	} else {
		return nil, nil
	}

	promptPath := g.promptForID(targetID)
	if promptPath == "" {
		return nil, nil
	}

	var inputs []string
	if node := g.Nodes[targetID]; node != nil {
		selectors := node.Inputs
		if len(selectors) == 0 {
			selectors = g.defaultSelectors(node)
		}
		resolver := NewInputResolver(root, g.Nodes, g.IDMap)
		resolved, err := resolver.Resolve(selectors)
		if err != nil {
			return nil, err
		}
		inputs = resolved
	}

	promptInputs, err := loadPromptInputs(root, inputs)
	if err != nil {
		return nil, err
	}

	promptText, schemaText, err := renderDocPromptText(plugin, promptPath, docPromptContext{
		DocID:   targetID,
		DocPath: g.expectedPath(targetID),
		Reason:  reason,
		Inputs:  promptInputs,
	})
	if err != nil {
		return nil, err
	}

	return &PromptEnvelope{
		Kind:           "dun.prompt.v1",
		ID:             check.ID,
		Title:          check.Description,
		Summary:        check.Description,
		Prompt:         promptText,
		Inputs:         inputs,
		ResponseSchema: schemaText,
		Callback: PromptCallback{
			Command: fmt.Sprintf("dun respond --id %s --response -", check.ID),
			Stdin:   true,
		},
	}, nil
}

func (g *DocGraph) promptForID(id string) string {
	if node := g.Nodes[id]; node != nil && node.Prompt != "" {
		return node.Prompt
	}
	if g.PromptDefaults[id] != "" {
		return g.PromptDefaults[id]
	}
	return g.DefaultPrompt
}

func (g *DocGraph) expectedPath(id string) string {
	if node := g.Nodes[id]; node != nil {
		return node.Path
	}
	for _, entry := range compileIDMap(g.IDMap) {
		if path, ok := entry.matchID(id); ok {
			return filepath.ToSlash(path)
		}
	}
	return ""
}

func (g *DocGraph) defaultSelectors(node *DocNode) []string {
	if node == nil || len(node.DependsOn) == 0 {
		return nil
	}
	selectors := make([]string, 0, len(node.DependsOn))
	for _, id := range node.DependsOn {
		selectors = append(selectors, "node:"+id)
	}
	return selectors
}

func loadPromptInputs(root string, paths []string) ([]PromptInput, error) {
	if len(paths) == 0 {
		return nil, nil
	}
	resolved := make([]PromptInput, 0, len(paths))
	for _, path := range paths {
		full := filepath.Join(root, filepath.FromSlash(path))
		content, err := os.ReadFile(full)
		if err != nil {
			return nil, err
		}
		resolved = append(resolved, PromptInput{
			Path:    path,
			Content: strings.TrimSpace(string(content)),
		})
	}
	return resolved, nil
}

func renderDocPromptText(plugin Plugin, promptPath string, ctx docPromptContext) (string, string, error) {
	tmplText, err := loadPromptTemplate(plugin, promptPath)
	if err != nil {
		return "", "", err
	}

	tmpl, err := template.New("doc-prompt").Parse(tmplText)
	if err != nil {
		return "", "", err
	}

	var buf strings.Builder
	if err := tmpl.Execute(&buf, ctx); err != nil {
		return "", "", err
	}

	schemaText, err := loadPromptTemplate(plugin, "responses/agent-default.json")
	if err != nil {
		return "", "", err
	}
	buf.WriteString("\n\nResponse Schema:\n")
	buf.WriteString(schemaText)

	return buf.String(), schemaText, nil
}

func invalidGraphIssue(root, path string, err error) Issue {
	rel := path
	if root != "" {
		if rp, relErr := relPath(root, path); relErr == nil {
			rel = filepath.ToSlash(rp)
		}
	}
	return Issue{
		ID:      "invalid-graph",
		Path:    rel,
		Summary: fmt.Sprintf("Invalid graph file: %v", err),
	}
}

func invalidFrontmatterIssue(root, path string, err error) Issue {
	rel := path
	if root != "" {
		if rp, relErr := relPath(root, path); relErr == nil {
			rel = filepath.ToSlash(rp)
		}
	}
	return Issue{
		ID:      "invalid-frontmatter",
		Path:    rel,
		Summary: fmt.Sprintf("Invalid frontmatter: %v", err),
	}
}

func countInvalidDocIssues(issues []Issue) int {
	count := 0
	for _, issue := range issues {
		if issue.ID == "invalid-frontmatter" || issue.ID == "invalid-graph" {
			count++
		}
	}
	return count
}

func buildDocDagSignal(invalidCount, missingCount, staleCount int) string {
	parts := make([]string, 0, 3)
	if invalidCount > 0 {
		parts = append(parts, fmt.Sprintf("%d invalid", invalidCount))
	}
	if missingCount > 0 {
		parts = append(parts, fmt.Sprintf("%d missing", missingCount))
	}
	if staleCount > 0 {
		parts = append(parts, fmt.Sprintf("%d stale docs", staleCount))
	}
	if len(parts) == 0 {
		return "all docs up to date"
	}
	return strings.Join(parts, ", ")
}
