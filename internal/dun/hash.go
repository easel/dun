package dun

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

func HashDocument(frontmatter *yaml.Node, body string) (string, error) {
	frontmatterText := ""
	if frontmatter != nil {
		clone := cloneNode(frontmatter)
		removeReview(clone)
		sortMappingNodes(clone)
		encoded, err := encodeYAML(clone)
		if err != nil {
			return "", fmt.Errorf("encode frontmatter: %w", err)
		}
		frontmatterText = normalizeNewlines(encoded)
	}

	normalizedBody := normalizeNewlines(body)
	content := normalizedBody
	if frontmatterText != "" {
		content = frontmatterText + "\n\n" + normalizedBody
	}

	sum := sha256.Sum256([]byte(content))
	return hex.EncodeToString(sum[:]), nil
}

func normalizeNewlines(value string) string {
	value = strings.ReplaceAll(value, "\r\n", "\n")
	value = strings.ReplaceAll(value, "\r", "\n")
	return value
}

func encodeYAML(node *yaml.Node) (string, error) {
	var buf strings.Builder
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(node); err != nil {
		return "", err
	}
	if err := enc.Close(); err != nil {
		return "", err
	}
	return strings.TrimRight(buf.String(), "\n"), nil
}

func cloneNode(node *yaml.Node) *yaml.Node {
	if node == nil {
		return nil
	}
	clone := *node
	if len(node.Content) > 0 {
		clone.Content = make([]*yaml.Node, len(node.Content))
		for i, child := range node.Content {
			clone.Content[i] = cloneNode(child)
		}
	}
	return &clone
}

func removeReview(root *yaml.Node) {
	if root == nil || root.Kind != yaml.MappingNode {
		return
	}
	dunNode := findMappingValue(root, "dun")
	if dunNode == nil || dunNode.Kind != yaml.MappingNode {
		return
	}
	removeMappingKey(dunNode, "review")
}

func sortMappingNodes(node *yaml.Node) {
	if node == nil {
		return
	}
	if node.Kind == yaml.MappingNode {
		type pair struct {
			key   *yaml.Node
			value *yaml.Node
		}
		pairs := make([]pair, 0, len(node.Content)/2)
		for i := 0; i < len(node.Content); i += 2 {
			pairs = append(pairs, pair{key: node.Content[i], value: node.Content[i+1]})
		}
		sort.SliceStable(pairs, func(i, j int) bool {
			return pairs[i].key.Value < pairs[j].key.Value
		})
		node.Content = node.Content[:0]
		for _, p := range pairs {
			node.Content = append(node.Content, p.key, p.value)
		}
	}
	for _, child := range node.Content {
		sortMappingNodes(child)
	}
}

func findMappingValue(node *yaml.Node, key string) *yaml.Node {
	if node == nil || node.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i < len(node.Content); i += 2 {
		k := node.Content[i]
		if k.Value == key {
			return node.Content[i+1]
		}
	}
	return nil
}

func removeMappingKey(node *yaml.Node, key string) {
	if node == nil || node.Kind != yaml.MappingNode {
		return
	}
	for i := 0; i < len(node.Content); i += 2 {
		k := node.Content[i]
		if k.Value == key {
			node.Content = append(node.Content[:i], node.Content[i+2:]...)
			return
		}
	}
}
