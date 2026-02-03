package dun

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

func StampDocs(root string, paths []string) ([]string, error) {
	if len(paths) == 0 {
		return nil, fmt.Errorf("no paths provided")
	}
	graph, err := buildDocGraph(root)
	if err != nil {
		return nil, err
	}

	var stamped []string
	for _, path := range paths {
		if path == "" {
			continue
		}
		rel, node := graph.nodeForPath(path)
		if node == nil {
			return nil, fmt.Errorf("doc not found or missing frontmatter: %s", path)
		}
		deps := make(map[string]string)
		for _, parentID := range node.DependsOn {
			parent := graph.Nodes[parentID]
			if parent == nil {
				continue
			}
			deps[parentID] = parent.Hash
		}
		review := DocReview{
			SelfHash: node.Hash,
			Deps:     deps,
		}
		if err := SetReview(node.Frontmatter, review); err != nil {
			return nil, err
		}
		frontmatterText, err := EncodeFrontmatter(node.Frontmatter)
		if err != nil {
			return nil, err
		}
		updated := "---\n" + frontmatterText + "\n---\n" + node.Body
		if err := os.WriteFile(node.AbsPath, []byte(updated), 0644); err != nil {
			return nil, err
		}
		stamped = append(stamped, rel)
	}

	sort.Strings(stamped)
	return stamped, nil
}

func StampAll(root string) ([]string, error) {
	graph, err := buildDocGraph(root)
	if err != nil {
		return nil, err
	}
	paths := make([]string, 0, len(graph.Nodes))
	for _, node := range graph.Nodes {
		paths = append(paths, node.Path)
	}
	return StampDocs(root, paths)
}

func (g *DocGraph) nodeForPath(path string) (string, *DocNode) {
	if filepath.IsAbs(path) {
		rel, err := relPath(g.Root, path)
		if err != nil {
			return "", nil
		}
		path = rel
	}
	path = filepath.ToSlash(path)
	if node := g.NodesByPath[path]; node != nil {
		return path, node
	}
	if node := g.Nodes[path]; node != nil {
		return node.Path, node
	}
	return "", nil
}
