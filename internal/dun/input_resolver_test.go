package dun

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestInputSelectorsResolveDeterministically(t *testing.T) {
	root := t.TempDir()
	writeFile := func(path, content string) {
		t.Helper()
		full := filepath.Join(root, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(full, []byte(content), 0644); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}

	docContent := "Requirements: REQ-100 and REQ-200"
	writeFile("docs/doc1.md", docContent)
	writeFile("docs/reqs/REQ-100.md", "Req 100")
	writeFile("docs/reqs/REQ-200.md", "Req 200")
	writeFile("extras/extra.md", "Extra")
	writeFile("internal/sample.go", "// DOC-1 referenced here")

	nodes := map[string]*DocNode{
		"DOC-1": {
			ID:      "DOC-1",
			Path:    "docs/doc1.md",
			Content: docContent,
		},
	}
	idMap := map[string]string{
		"DOC-1":    "docs/doc1.md",
		"REQ-{id}": "docs/reqs/REQ-{id}.md",
	}

	resolver := NewInputResolver(root, nodes, idMap)
	selectors := []string{
		"node:DOC-1",
		"refs:DOC-1",
		"code_refs:DOC-1",
		"paths:extras/*.md",
	}

	resolved, err := resolver.Resolve(selectors)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}

	expected := []string{
		"docs/doc1.md",
		"docs/reqs/REQ-100.md",
		"docs/reqs/REQ-200.md",
		"extras/extra.md",
		"internal/sample.go",
	}
	if !reflect.DeepEqual(resolved, expected) {
		t.Fatalf("unexpected resolved inputs\nexpected: %#v\nactual: %#v", expected, resolved)
	}
}

func TestNodeContentReadsFile(t *testing.T) {
	root := t.TempDir()
	writeFile := func(path, content string) {
		t.Helper()
		full := filepath.Join(root, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(full, []byte(content), 0644); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}

	fileContent := "File content from disk"
	writeFile("docs/disk.md", fileContent)

	nodes := map[string]*DocNode{
		"DISK-DOC": {
			ID:      "DISK-DOC",
			Path:    "docs/disk.md",
			Content: "",
		},
	}

	resolver := NewInputResolver(root, nodes, nil)
	content := resolver.nodeContent("DISK-DOC")
	if content != fileContent {
		t.Fatalf("expected %q, got %q", fileContent, content)
	}
}

func TestNodeContentReturnsEmpty(t *testing.T) {
	root := t.TempDir()
	nodes := map[string]*DocNode{}
	resolver := NewInputResolver(root, nodes, nil)
	content := resolver.nodeContent("nonexistent")
	if content != "" {
		t.Fatalf("expected empty string, got %q", content)
	}
}

func TestNodeContentUsesInMemoryContent(t *testing.T) {
	root := t.TempDir()
	inMemContent := "In memory content"
	nodes := map[string]*DocNode{
		"MEM-DOC": {
			ID:      "MEM-DOC",
			Path:    "docs/mem.md",
			Content: inMemContent,
		},
	}

	resolver := NewInputResolver(root, nodes, nil)
	content := resolver.nodeContent("MEM-DOC")
	if content != inMemContent {
		t.Fatalf("expected %q, got %q", inMemContent, content)
	}
}

func TestFileIfExistsReturnsPath(t *testing.T) {
	root := t.TempDir()
	writeFile := func(path, content string) {
		t.Helper()
		full := filepath.Join(root, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(full, []byte(content), 0644); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}

	writeFile("docs/exists.md", "content")

	paths := fileIfExists(root, "docs/exists.md")
	if len(paths) != 1 {
		t.Fatalf("expected 1 path, got %d", len(paths))
	}
	if paths[0] != "docs/exists.md" {
		t.Fatalf("expected docs/exists.md, got %q", paths[0])
	}
}

func TestFileIfExistsReturnsNilForMissing(t *testing.T) {
	root := t.TempDir()
	paths := fileIfExists(root, "nonexistent.md")
	if paths != nil {
		t.Fatalf("expected nil, got %v", paths)
	}
}

func TestResolveEmptySelectors(t *testing.T) {
	root := t.TempDir()
	resolver := NewInputResolver(root, nil, nil)
	resolved, err := resolver.Resolve([]string{})
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if len(resolved) != 0 {
		t.Fatalf("expected empty, got %v", resolved)
	}
}

func TestResolveSkipsEmptySelectors(t *testing.T) {
	root := t.TempDir()
	writeFile := func(path, content string) {
		t.Helper()
		full := filepath.Join(root, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(full, []byte(content), 0644); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}

	writeFile("docs/file.md", "content")
	resolver := NewInputResolver(root, nil, nil)
	resolved, err := resolver.Resolve([]string{"", "  ", "docs/file.md"})
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if len(resolved) != 1 {
		t.Fatalf("expected 1, got %d: %v", len(resolved), resolved)
	}
}

func TestResolveDeduplicates(t *testing.T) {
	root := t.TempDir()
	writeFile := func(path, content string) {
		t.Helper()
		full := filepath.Join(root, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(full, []byte(content), 0644); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}

	writeFile("docs/file.md", "content")
	resolver := NewInputResolver(root, nil, nil)
	resolved, err := resolver.Resolve([]string{"docs/file.md", "docs/file.md"})
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if len(resolved) != 1 {
		t.Fatalf("expected 1 (deduplicated), got %d: %v", len(resolved), resolved)
	}
}

func TestResolveWithGlobSelector(t *testing.T) {
	root := t.TempDir()
	writeFile := func(path, content string) {
		t.Helper()
		full := filepath.Join(root, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(full, []byte(content), 0644); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}

	writeFile("docs/a.md", "a")
	writeFile("docs/b.md", "b")

	resolver := NewInputResolver(root, nil, nil)
	resolved, err := resolver.Resolve([]string{"docs/*.md"})
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if len(resolved) != 2 {
		t.Fatalf("expected 2, got %d: %v", len(resolved), resolved)
	}
}

func TestResolveRefsWithNonPlaceholderMatch(t *testing.T) {
	root := t.TempDir()
	writeFile := func(path, content string) {
		t.Helper()
		full := filepath.Join(root, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(full, []byte(content), 0644); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}

	docContent := "References FIXED-ID in the content"
	writeFile("docs/doc.md", docContent)
	writeFile("docs/fixed.md", "fixed doc content")

	nodes := map[string]*DocNode{
		"DOC": {
			ID:      "DOC",
			Path:    "docs/doc.md",
			Content: docContent,
		},
	}
	idMap := map[string]string{
		"FIXED-ID": "docs/fixed.md",
	}

	resolver := NewInputResolver(root, nodes, idMap)
	resolved, err := resolver.Resolve([]string{"refs:DOC"})
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if len(resolved) != 1 || resolved[0] != "docs/fixed.md" {
		t.Fatalf("expected [docs/fixed.md], got %v", resolved)
	}
}
