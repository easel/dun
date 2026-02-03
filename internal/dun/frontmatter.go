package dun

import (
	"bytes"
	"fmt"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

type Frontmatter struct {
	Dun            DunFrontmatter
	Raw            *yaml.Node
	HasFrontmatter bool
}

type DunFrontmatter struct {
	ID        string    `yaml:"id"`
	DependsOn []string  `yaml:"depends_on"`
	Prompt    string    `yaml:"prompt"`
	Inputs    []string  `yaml:"inputs"`
	Review    DocReview `yaml:"review"`
}

type DocReview struct {
	SelfHash   string            `yaml:"self_hash"`
	Deps       map[string]string `yaml:"deps"`
	ReviewedAt string            `yaml:"reviewed_at"`
}

func ParseFrontmatter(content []byte) (Frontmatter, string, error) {
	frontmatterText, body, ok := splitFrontmatter(content)
	if !ok {
		return Frontmatter{}, string(content), nil
	}

	trimmed := strings.TrimSpace(frontmatterText)
	var wrapper struct {
		Dun DunFrontmatter `yaml:"dun"`
	}
	if err := yaml.Unmarshal([]byte(trimmed), &wrapper); err != nil {
		return Frontmatter{}, "", fmt.Errorf("parse frontmatter: %w", err)
	}

	var node yaml.Node
	if err := yaml.Unmarshal([]byte(trimmed), &node); err != nil {
		return Frontmatter{}, "", fmt.Errorf("parse frontmatter node: %w", err)
	}

	var root *yaml.Node
	if len(node.Content) > 0 {
		root = node.Content[0]
	}

	return Frontmatter{
		Dun:            wrapper.Dun,
		Raw:            root,
		HasFrontmatter: true,
	}, body, nil
}

func splitFrontmatter(content []byte) (string, string, bool) {
	if !bytes.HasPrefix(content, []byte("---")) {
		return "", string(content), false
	}

	firstLineEnd := bytes.IndexByte(content, '\n')
	if firstLineEnd == -1 {
		return "", string(content), false
	}

	firstLine := bytes.TrimRight(content[:firstLineEnd], "\r")
	if !bytes.Equal(firstLine, []byte("---")) {
		return "", string(content), false
	}

	rest := content[firstLineEnd+1:]
	idx := 0
	for idx <= len(rest) {
		lineEnd := bytes.IndexByte(rest[idx:], '\n')
		if lineEnd == -1 {
			lineEnd = len(rest) - idx
		}
		line := rest[idx : idx+lineEnd]
		lineTrimmed := bytes.TrimRight(line, "\r")
		if bytes.Equal(lineTrimmed, []byte("---")) {
			frontmatter := rest[:idx]
			bodyStart := idx + lineEnd
			if idx+lineEnd < len(rest) && rest[idx+lineEnd] == '\n' {
				bodyStart = idx + lineEnd + 1
			}
			return string(frontmatter), string(rest[bodyStart:]), true
		}
		if idx+lineEnd >= len(rest) {
			break
		}
		idx += lineEnd + 1
	}

	return "", string(content), false
}

func SetReview(root *yaml.Node, review DocReview) error {
	if root == nil {
		return fmt.Errorf("frontmatter missing")
	}
	if root.Kind != yaml.MappingNode {
		return fmt.Errorf("frontmatter root must be mapping")
	}

	dunNode := ensureMappingNode(root, "dun")
	reviewNode := ensureMappingNode(dunNode, "review")
	setScalarNode(reviewNode, "self_hash", review.SelfHash)
	setMappingNode(reviewNode, "deps", review.Deps)
	if review.ReviewedAt != "" {
		setScalarNode(reviewNode, "reviewed_at", review.ReviewedAt)
	}
	return nil
}

func EncodeFrontmatter(root *yaml.Node) (string, error) {
	if root == nil {
		return "", fmt.Errorf("frontmatter missing")
	}
	var buf strings.Builder
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(root); err != nil {
		return "", err
	}
	if err := enc.Close(); err != nil {
		return "", err
	}
	return strings.TrimRight(buf.String(), "\n"), nil
}

func ensureMappingNode(parent *yaml.Node, key string) *yaml.Node {
	if parent.Kind != yaml.MappingNode {
		parent.Kind = yaml.MappingNode
	}
	for i := 0; i < len(parent.Content); i += 2 {
		k := parent.Content[i]
		if k.Value == key {
			v := parent.Content[i+1]
			if v.Kind != yaml.MappingNode {
				v.Kind = yaml.MappingNode
				v.Content = nil
			}
			return v
		}
	}
	keyNode := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key}
	valueNode := &yaml.Node{Kind: yaml.MappingNode}
	parent.Content = append(parent.Content, keyNode, valueNode)
	return valueNode
}

func setScalarNode(parent *yaml.Node, key string, value string) {
	for i := 0; i < len(parent.Content); i += 2 {
		k := parent.Content[i]
		if k.Value == key {
			parent.Content[i+1] = &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: value}
			return
		}
	}
	parent.Content = append(parent.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key},
		&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: value},
	)
}

func setMappingNode(parent *yaml.Node, key string, values map[string]string) {
	mapping := &yaml.Node{Kind: yaml.MappingNode}
	if len(values) > 0 {
		keys := make([]string, 0, len(values))
		for k := range values {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			mapping.Content = append(mapping.Content,
				&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: k},
				&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: values[k]},
			)
		}
	}
	for i := 0; i < len(parent.Content); i += 2 {
		k := parent.Content[i]
		if k.Value == key {
			parent.Content[i+1] = mapping
			return
		}
	}
	parent.Content = append(parent.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key},
		mapping,
	)
}
