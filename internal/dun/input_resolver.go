package dun

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

type InputResolver struct {
	root  string
	nodes map[string]*DocNode
	idMap []idMapEntry
}

type idMapEntry struct {
	key            string
	value          string
	re             *regexp.Regexp
	hasPlaceholder bool
}

func NewInputResolver(root string, nodes map[string]*DocNode, idMap map[string]string) *InputResolver {
	entries := compileIDMap(idMap)
	return &InputResolver{root: root, nodes: nodes, idMap: entries}
}

func (r *InputResolver) Resolve(selectors []string) ([]string, error) {
	seen := make(map[string]bool)
	var resolved []string

	for _, selector := range selectors {
		selector = strings.TrimSpace(selector)
		if selector == "" {
			continue
		}
		var paths []string
		switch {
		case strings.HasPrefix(selector, "node:"):
			id := strings.TrimPrefix(selector, "node:")
			paths = r.resolveIDPaths(id)
		case strings.HasPrefix(selector, "refs:"):
			id := strings.TrimPrefix(selector, "refs:")
			paths = r.resolveRefs(id)
		case strings.HasPrefix(selector, "code_refs:"):
			id := strings.TrimPrefix(selector, "code_refs:")
			matches, err := findCodeRefs(r.root, id)
			if err != nil {
				return nil, err
			}
			paths = matches
		case strings.HasPrefix(selector, "paths:"):
			pattern := strings.TrimPrefix(selector, "paths:")
			paths = globPaths(r.root, pattern)
		default:
			if hasGlob(selector) {
				paths = globPaths(r.root, selector)
			} else {
				paths = fileIfExists(r.root, selector)
			}
		}
		for _, path := range paths {
			if path == "" || seen[path] {
				continue
			}
			seen[path] = true
			resolved = append(resolved, path)
		}
	}

	sort.Strings(resolved)
	return resolved, nil
}

func (r *InputResolver) resolveIDPaths(id string) []string {
	if node := r.nodes[id]; node != nil {
		return []string{node.Path}
	}
	for _, entry := range r.idMap {
		if path, ok := entry.matchID(id); ok {
			return globPaths(r.root, path)
		}
	}
	return nil
}

func (r *InputResolver) resolveRefs(id string) []string {
	content := r.nodeContent(id)
	if content == "" {
		return nil
	}
	refs := r.extractRefs(content)
	seen := make(map[string]bool)
	var paths []string
	for _, ref := range refs {
		for _, path := range r.resolveIDPaths(ref) {
			if path == "" || seen[path] {
				continue
			}
			seen[path] = true
			paths = append(paths, path)
		}
	}
	sort.Strings(paths)
	return paths
}

func (r *InputResolver) nodeContent(id string) string {
	node := r.nodes[id]
	if node == nil {
		return ""
	}
	if node.Content != "" {
		return node.Content
	}
	abs := filepath.Join(r.root, filepath.FromSlash(node.Path))
	data, err := os.ReadFile(abs)
	if err != nil {
		return ""
	}
	return string(data)
}

func (r *InputResolver) extractRefs(content string) []string {
	seen := make(map[string]bool)
	var refs []string
	for _, entry := range r.idMap {
		if entry.hasPlaceholder {
			matches := entry.re.FindAllStringSubmatch(content, -1)
			for _, match := range matches {
				if len(match) < 2 {
					continue
				}
				id := strings.ReplaceAll(entry.key, "{id}", match[1])
				if !seen[id] {
					seen[id] = true
					refs = append(refs, id)
				}
			}
			continue
		}
		if entry.re.MatchString(content) {
			if !seen[entry.key] {
				seen[entry.key] = true
				refs = append(refs, entry.key)
			}
		}
	}
	sort.Strings(refs)
	return refs
}

func compileIDMap(idMap map[string]string) []idMapEntry {
	if len(idMap) == 0 {
		return nil
	}
	keys := make([]string, 0, len(idMap))
	for k := range idMap {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	entries := make([]idMapEntry, 0, len(keys))
	for _, key := range keys {
		value := idMap[key]
		hasPlaceholder := strings.Contains(key, "{id}")
		pattern := regexp.QuoteMeta(key)
		if hasPlaceholder {
			pattern = strings.ReplaceAll(pattern, "\\{id\\}", "([A-Za-z0-9._-]+)")
		}
		re, err := regexp.Compile(pattern)
		if err != nil {
			continue
		}
		entries = append(entries, idMapEntry{
			key:            key,
			value:          value,
			re:             re,
			hasPlaceholder: hasPlaceholder,
		})
	}
	return entries
}

func (e idMapEntry) matchID(id string) (string, bool) {
	if e.hasPlaceholder {
		matches := e.re.FindStringSubmatch(id)
		if len(matches) < 2 {
			return "", false
		}
		return strings.ReplaceAll(e.value, "{id}", matches[1]), true
	}
	if id == e.key {
		return e.value, true
	}
	return "", false
}

func globPaths(root, pattern string) []string {
	pattern = filepath.Join(root, filepath.FromSlash(pattern))
	matches, err := filepath.Glob(pattern)
	if err != nil || len(matches) == 0 {
		return nil
	}
	var paths []string
	for _, match := range matches {
		rel, err := relPath(root, match)
		if err != nil {
			continue
		}
		paths = append(paths, filepath.ToSlash(rel))
	}
	sort.Strings(paths)
	return paths
}

func fileIfExists(root, path string) []string {
	full := filepath.Join(root, filepath.FromSlash(path))
	if _, err := os.Stat(full); err != nil {
		return nil
	}
	rel, err := relPath(root, full)
	if err != nil {
		return nil
	}
	return []string{filepath.ToSlash(rel)}
}

func findCodeRefs(root, needle string) ([]string, error) {
	var matches []string
	needleBytes := []byte(needle)
	if len(needleBytes) == 0 {
		return nil, nil
	}
	allowedExt := map[string]bool{
		".go":    true,
		".ts":    true,
		".tsx":   true,
		".js":    true,
		".jsx":   true,
		".py":    true,
		".rb":    true,
		".rs":    true,
		".java":  true,
		".kt":    true,
		".cs":    true,
		".cpp":   true,
		".c":     true,
		".h":     true,
		".hpp":   true,
		".swift": true,
		".sql":   true,
		".proto": true,
	}

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		name := d.Name()
		if d.IsDir() {
			if shouldSkipDir(name) {
				return filepath.SkipDir
			}
			return nil
		}
		if !allowedExt[filepath.Ext(name)] {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return nil
		}
		if info.Size() > 1<<20 {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if bytes.Contains(data, needleBytes) {
			rel, err := relPath(root, path)
			if err != nil {
				return nil
			}
			matches = append(matches, filepath.ToSlash(rel))
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("code refs: %w", err)
	}

	sort.Strings(matches)
	return matches, nil
}

func shouldSkipDir(name string) bool {
	switch name {
	case ".git", ".dun", "node_modules", "vendor", "testdata":
		return true
	default:
		return false
	}
}
