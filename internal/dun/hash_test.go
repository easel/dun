package dun

import (
	"strings"
	"testing"
)

func hashFromContent(content string) (string, error) {
	fm, body, err := ParseFrontmatter([]byte(content))
	if err != nil {
		return "", err
	}
	return HashDocument(fm.Raw, body)
}

func TestHashExcludesReviewSection(t *testing.T) {
	contentA := `---
dun:
  id: test.doc
  review:
    self_hash: aaa
    deps:
      parent.doc: old
---
# Title
Body`

	contentB := `---
dun:
  id: test.doc
  review:
    self_hash: bbb
    deps:
      parent.doc: new
---
# Title
Body`

	hashA, err := hashFromContent(contentA)
	if err != nil {
		t.Fatalf("hash A: %v", err)
	}
	hashB, err := hashFromContent(contentB)
	if err != nil {
		t.Fatalf("hash B: %v", err)
	}

	if hashA != hashB {
		t.Fatalf("expected hashes to match, got %s vs %s", hashA, hashB)
	}
}

func TestHashCanonicalizesFrontmatter(t *testing.T) {
	contentA := `---
dun:
  id: test.doc
  depends_on:
    - parent.doc
  inputs:
    - node:parent.doc
meta:
  z: 1
  a:
    c: 3
    b: 2
---
# Title
Body`

	contentB := `---
meta:
  a:
    b: 2
    c: 3
  z: 1
dun:
  inputs:
    - node:parent.doc
  depends_on:
    - parent.doc
  id: test.doc
---
# Title
Body`

	hashA, err := hashFromContent(contentA)
	if err != nil {
		t.Fatalf("hash A: %v", err)
	}
	hashB, err := hashFromContent(contentB)
	if err != nil {
		t.Fatalf("hash B: %v", err)
	}

	if hashA != hashB {
		t.Fatalf("expected hashes to match, got %s vs %s", hashA, hashB)
	}
}

func TestHashRejectsInvalidFrontmatter(t *testing.T) {
	content := `---
dun:
  id: [invalid yaml
---
# Title
Body`

	_, err := hashFromContent(content)
	if err == nil {
		t.Fatal("expected error for invalid YAML frontmatter")
	}
	if !strings.Contains(err.Error(), "parse frontmatter") {
		t.Fatalf("expected parse frontmatter error, got %v", err)
	}
}
