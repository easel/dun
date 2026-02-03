package dun

import (
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestFrontmatterParseDunBlock(t *testing.T) {
	content := `---
dun:
  id: test.doc
  depends_on:
    - parent.doc
  prompt: prompts/test.md
  inputs:
    - node:parent.doc
  review:
    self_hash: abc123
    deps:
      parent.doc: def456
---
# Title
Body`

	frontmatter, body, err := ParseFrontmatter([]byte(content))
	if err != nil {
		t.Fatalf("parse frontmatter: %v", err)
	}
	if !frontmatter.HasFrontmatter {
		t.Fatalf("expected frontmatter")
	}
	if frontmatter.Dun.ID != "test.doc" {
		t.Fatalf("expected id, got %q", frontmatter.Dun.ID)
	}
	if len(frontmatter.Dun.DependsOn) != 1 || frontmatter.Dun.DependsOn[0] != "parent.doc" {
		t.Fatalf("unexpected depends_on: %#v", frontmatter.Dun.DependsOn)
	}
	if frontmatter.Dun.Prompt != "prompts/test.md" {
		t.Fatalf("unexpected prompt: %q", frontmatter.Dun.Prompt)
	}
	if len(frontmatter.Dun.Inputs) != 1 || frontmatter.Dun.Inputs[0] != "node:parent.doc" {
		t.Fatalf("unexpected inputs: %#v", frontmatter.Dun.Inputs)
	}
	if frontmatter.Dun.Review.SelfHash != "abc123" {
		t.Fatalf("unexpected self hash: %q", frontmatter.Dun.Review.SelfHash)
	}
	if frontmatter.Dun.Review.Deps["parent.doc"] != "def456" {
		t.Fatalf("unexpected deps: %#v", frontmatter.Dun.Review.Deps)
	}
	if !strings.Contains(body, "# Title") {
		t.Fatalf("expected body content, got %q", body)
	}
}

func TestFrontmatterNoFrontmatter(t *testing.T) {
	content := `# Just markdown

No frontmatter here.`

	fm, body, err := ParseFrontmatter([]byte(content))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if fm.HasFrontmatter {
		t.Fatal("expected no frontmatter")
	}
	if !strings.Contains(body, "# Just markdown") {
		t.Fatalf("expected body, got %q", body)
	}
}

func TestFrontmatterInvalidYAML(t *testing.T) {
	content := `---
dun:
  id: [invalid yaml
---
# Body`

	_, _, err := ParseFrontmatter([]byte(content))
	if err == nil {
		t.Fatal("expected error for invalid YAML")
	}
}

func TestFrontmatterNoClosingDelimiter(t *testing.T) {
	content := `---
dun:
  id: test
# No closing delimiter`

	fm, body, err := ParseFrontmatter([]byte(content))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if fm.HasFrontmatter {
		t.Fatal("expected no frontmatter (no closing delimiter)")
	}
	if !strings.Contains(body, "---") {
		t.Fatalf("expected content as body, got %q", body)
	}
}

func TestFrontmatterFirstLineNotPureDashes(t *testing.T) {
	content := `--- yaml
dun:
  id: test
---
# Body`

	fm, body, err := ParseFrontmatter([]byte(content))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if fm.HasFrontmatter {
		t.Fatal("expected no frontmatter (first line has extra content)")
	}
	if !strings.Contains(body, "--- yaml") {
		t.Fatalf("expected content as body, got %q", body)
	}
}

func TestFrontmatterOnlyDelimiter(t *testing.T) {
	content := `---`

	fm, _, err := ParseFrontmatter([]byte(content))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if fm.HasFrontmatter {
		t.Fatal("expected no frontmatter (only opening delimiter)")
	}
}

func TestSetReviewNilRoot(t *testing.T) {
	err := SetReview(nil, DocReview{})
	if err == nil {
		t.Fatal("expected error for nil root")
	}
}

func TestSetReviewNonMapping(t *testing.T) {
	node := &yaml.Node{Kind: yaml.ScalarNode, Value: "scalar"}
	err := SetReview(node, DocReview{})
	if err == nil {
		t.Fatal("expected error for non-mapping root")
	}
}

func TestSetReviewCreatesKeys(t *testing.T) {
	root := &yaml.Node{Kind: yaml.MappingNode, Content: []*yaml.Node{}}
	review := DocReview{
		SelfHash: "hash123",
		Deps:     map[string]string{"dep1": "dep1hash"},
	}

	if err := SetReview(root, review); err != nil {
		t.Fatalf("set review: %v", err)
	}

	encoded, err := EncodeFrontmatter(root)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}

	if !strings.Contains(encoded, "self_hash: hash123") {
		t.Fatalf("expected self_hash in output: %s", encoded)
	}
	if !strings.Contains(encoded, "dep1: dep1hash") {
		t.Fatalf("expected dep in output: %s", encoded)
	}
}

func TestSetReviewWithReviewedAt(t *testing.T) {
	root := &yaml.Node{Kind: yaml.MappingNode, Content: []*yaml.Node{}}
	review := DocReview{
		SelfHash:   "hash",
		Deps:       map[string]string{},
		ReviewedAt: "2025-01-01",
	}

	if err := SetReview(root, review); err != nil {
		t.Fatalf("set review: %v", err)
	}

	encoded, err := EncodeFrontmatter(root)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}

	if !strings.Contains(encoded, "reviewed_at: \"2025-01-01\"") && !strings.Contains(encoded, "reviewed_at: 2025-01-01") {
		t.Fatalf("expected reviewed_at in output: %s", encoded)
	}
}

func TestEncodeFrontmatterNilRoot(t *testing.T) {
	_, err := EncodeFrontmatter(nil)
	if err == nil {
		t.Fatal("expected error for nil root")
	}
}

func TestEncodeFrontmatterPreservesStructure(t *testing.T) {
	content := `---
dun:
  id: test
other_key: value
---
`
	fm, _, err := ParseFrontmatter([]byte(content))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	encoded, err := EncodeFrontmatter(fm.Raw)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}

	if !strings.Contains(encoded, "dun:") {
		t.Fatalf("expected dun key: %s", encoded)
	}
	if !strings.Contains(encoded, "other_key:") {
		t.Fatalf("expected other_key preserved: %s", encoded)
	}
}

func TestEnsureMappingNodeUpdatesExisting(t *testing.T) {
	content := `---
dun:
  id: existing
  review:
    self_hash: old
---
`
	fm, _, err := ParseFrontmatter([]byte(content))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	review := DocReview{
		SelfHash: "newhash",
		Deps:     map[string]string{},
	}
	if err := SetReview(fm.Raw, review); err != nil {
		t.Fatalf("set review: %v", err)
	}

	encoded, err := EncodeFrontmatter(fm.Raw)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}

	if !strings.Contains(encoded, "newhash") {
		t.Fatalf("expected new hash in output: %s", encoded)
	}
	if strings.Contains(encoded, "old") && strings.Contains(encoded, "self_hash: old") {
		t.Fatalf("old hash should be replaced: %s", encoded)
	}
}

func TestSplitFrontmatterWindowsLineEndings(t *testing.T) {
	content := "---\r\ndun:\r\n  id: win\r\n---\r\nBody"

	fm, body, err := ParseFrontmatter([]byte(content))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if !fm.HasFrontmatter {
		t.Fatal("expected frontmatter with CRLF")
	}
	if fm.Dun.ID != "win" {
		t.Fatalf("expected id 'win', got %q", fm.Dun.ID)
	}
	if !strings.Contains(body, "Body") {
		t.Fatalf("expected body, got %q", body)
	}
}
