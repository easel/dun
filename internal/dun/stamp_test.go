package dun

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStampUpdatesReviewDeps(t *testing.T) {
	root := t.TempDir()
	write := func(rel, content string) {
		t.Helper()
		path := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("write %s: %v", rel, err)
		}
	}

	parent := `---
dun:
  id: parent.doc
---
# Parent
`
	child := `---
dun:
  id: child.doc
  depends_on:
    - parent.doc
  review:
    self_hash: ""
    deps: {}
---
# Child
`

	write("docs/parent.md", parent)
	write("docs/child.md", child)

	stamped, err := StampDocs(root, []string{"docs/child.md"})
	if err != nil {
		t.Fatalf("stamp docs: %v", err)
	}
	if len(stamped) != 1 {
		t.Fatalf("expected 1 stamped doc, got %d", len(stamped))
	}

	parentContent, err := os.ReadFile(filepath.Join(root, "docs", "parent.md"))
	if err != nil {
		t.Fatalf("read parent: %v", err)
	}
	parentFM, parentBody, err := ParseFrontmatter(parentContent)
	if err != nil {
		t.Fatalf("parse parent: %v", err)
	}
	parentHash, err := HashDocument(parentFM.Raw, parentBody)
	if err != nil {
		t.Fatalf("hash parent: %v", err)
	}

	childContent, err := os.ReadFile(filepath.Join(root, "docs", "child.md"))
	if err != nil {
		t.Fatalf("read child: %v", err)
	}
	childFM, _, err := ParseFrontmatter(childContent)
	if err != nil {
		t.Fatalf("parse child: %v", err)
	}

	if childFM.Dun.Review.Deps["parent.doc"] != parentHash {
		t.Fatalf("expected review dep %s, got %q", parentHash, childFM.Dun.Review.Deps["parent.doc"])
	}
}

func TestStampAll(t *testing.T) {
	root := t.TempDir()
	write := func(rel, content string) {
		t.Helper()
		path := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("write %s: %v", rel, err)
		}
	}

	doc1 := `---
dun:
  id: doc.one
---
# Doc One
`
	doc2 := `---
dun:
  id: doc.two
  depends_on:
    - doc.one
---
# Doc Two
`
	write("docs/one.md", doc1)
	write("docs/two.md", doc2)

	stamped, err := StampAll(root)
	if err != nil {
		t.Fatalf("stamp all: %v", err)
	}
	if len(stamped) != 2 {
		t.Fatalf("expected 2 stamped docs, got %d: %v", len(stamped), stamped)
	}
}

func TestStampDocsEmptyPaths(t *testing.T) {
	root := t.TempDir()
	_, err := StampDocs(root, []string{})
	if err == nil {
		t.Fatal("expected error for empty paths")
	}
}

func TestStampDocsNotFound(t *testing.T) {
	root := t.TempDir()
	_, err := StampDocs(root, []string{"nonexistent.md"})
	if err == nil {
		t.Fatal("expected error for nonexistent doc")
	}
}

func TestStampDocsSkipsEmptyPath(t *testing.T) {
	root := t.TempDir()
	write := func(rel, content string) {
		t.Helper()
		path := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("write %s: %v", rel, err)
		}
	}

	doc := `---
dun:
  id: doc.test
---
# Test
`
	write("docs/test.md", doc)

	stamped, err := StampDocs(root, []string{"", "docs/test.md", ""})
	if err != nil {
		t.Fatalf("stamp docs: %v", err)
	}
	if len(stamped) != 1 {
		t.Fatalf("expected 1 stamped doc, got %d", len(stamped))
	}
}

func TestNodeForPathAbsolutePath(t *testing.T) {
	root := t.TempDir()
	write := func(rel, content string) {
		t.Helper()
		path := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("write %s: %v", rel, err)
		}
	}

	doc := `---
dun:
  id: doc.abs
---
# Abs test
`
	write("docs/abs.md", doc)

	graph, err := buildDocGraph(root)
	if err != nil {
		t.Fatalf("build graph: %v", err)
	}

	absPath := filepath.Join(root, "docs", "abs.md")
	relPath, node := graph.nodeForPath(absPath)
	if node == nil {
		t.Fatal("expected node for absolute path")
	}
	if relPath != "docs/abs.md" {
		t.Fatalf("expected relative path, got %q", relPath)
	}
}

func TestNodeForPathByID(t *testing.T) {
	root := t.TempDir()
	write := func(rel, content string) {
		t.Helper()
		path := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("write %s: %v", rel, err)
		}
	}

	doc := `---
dun:
  id: my.doc
---
# My doc
`
	write("docs/mydoc.md", doc)

	graph, err := buildDocGraph(root)
	if err != nil {
		t.Fatalf("build graph: %v", err)
	}

	relPath, node := graph.nodeForPath("my.doc")
	if node == nil {
		t.Fatal("expected node for ID lookup")
	}
	if relPath != "docs/mydoc.md" {
		t.Fatalf("expected path docs/mydoc.md, got %q", relPath)
	}
}

func TestNodeForPathNotFound(t *testing.T) {
	root := t.TempDir()

	graph, err := buildDocGraph(root)
	if err != nil {
		t.Fatalf("build graph: %v", err)
	}

	relPath, node := graph.nodeForPath("nonexistent")
	if node != nil {
		t.Fatal("expected nil node for nonexistent path")
	}
	if relPath != "" {
		t.Fatalf("expected empty relPath, got %q", relPath)
	}
}
