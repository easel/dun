package dun

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRunStateRulesMissingPath(t *testing.T) {
	plugin := Plugin{}
	_, err := runStateRules(".", plugin, Check{ID: "state"})
	if err == nil {
		t.Fatalf("expected missing state rules error")
	}
}

func TestRunStateRulesReadError(t *testing.T) {
	plugin := Plugin{FS: os.DirFS(t.TempDir()), Base: "."}
	_, err := runStateRules(".", plugin, Check{ID: "state", StateRules: "missing.yml"})
	if err == nil {
		t.Fatalf("expected read error")
	}
}

func TestRunStateRulesParseError(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "rules.yml"), ":")
	plugin := Plugin{FS: os.DirFS(dir), Base: "."}
	_, err := runStateRules(".", plugin, Check{ID: "state", StateRules: "rules.yml"})
	if err == nil {
		t.Fatalf("expected parse error")
	}
}

func TestRunStateRulesPassAndFail(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "rules.yml"), `artifact_patterns:
  story:
    frame: { pattern: "frame/US-{id}.md" }
    design: { pattern: "design/TD-{id}.md" }
    test: { pattern: "test/TP-{id}.md" }
    build: { pattern: "build/IP-{id}.md" }
`)
	if err := os.MkdirAll(filepath.Join(dir, "frame"), 0755); err != nil {
		t.Fatalf("mkdir frame: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "design"), 0755); err != nil {
		t.Fatalf("mkdir design: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "test"), 0755); err != nil {
		t.Fatalf("mkdir test: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "build"), 0755); err != nil {
		t.Fatalf("mkdir build: %v", err)
	}
	writeFile(t, filepath.Join(dir, "frame", "US-1.md"), "US-1")
	writeFile(t, filepath.Join(dir, "design", "TD-1.md"), "TD-1")
	writeFile(t, filepath.Join(dir, "test", "TP-1.md"), "TP-1")
	writeFile(t, filepath.Join(dir, "build", "IP-1.md"), "IP-1")

	plugin := Plugin{FS: os.DirFS(dir), Base: "."}
	check := Check{ID: "state", StateRules: "rules.yml"}
	res, err := runStateRules(dir, plugin, check)
	if err != nil {
		t.Fatalf("run state rules: %v", err)
	}
	if res.Status != "pass" {
		t.Fatalf("expected pass, got %s", res.Status)
	}

	writeFile(t, filepath.Join(dir, "test", "TP-2.md"), "TP-2")
	res, err = runStateRules(dir, plugin, check)
	if err != nil {
		t.Fatalf("run state rules fail: %v", err)
	}
	if res.Status != "fail" {
		t.Fatalf("expected fail, got %s", res.Status)
	}
}

func TestIdsForPatternEmpty(t *testing.T) {
	ids, err := idsForPattern(t.TempDir(), artifactPattern{})
	if err != nil {
		t.Fatalf("ids for pattern: %v", err)
	}
	if len(ids) != 0 {
		t.Fatalf("expected empty ids")
	}
}

func TestPrefixAndParseID(t *testing.T) {
	if prefix := prefixFromPattern("foo/bar/US-{id}.md"); prefix != "US" {
		t.Fatalf("expected US prefix, got %q", prefix)
	}
	if prefix := prefixFromPattern("foo/bar/no-id.md"); prefix != "" {
		t.Fatalf("expected empty prefix")
	}
	if id := parseID("US-123-something.md", "US"); id != "123" {
		t.Fatalf("expected id 123, got %q", id)
	}
	if id := parseID("TD-1.md", "US"); id != "" {
		t.Fatalf("expected empty id")
	}
	if id := parseID("US-1.md", ""); id != "" {
		t.Fatalf("expected empty id for missing prefix")
	}
}

func TestIdsForPatternGlobError(t *testing.T) {
	_, err := idsForPattern(t.TempDir(), artifactPattern{Pattern: "[{id}.md"})
	if err == nil {
		t.Fatalf("expected glob error")
	}
}

func TestRunStateRulesFrameGlobError(t *testing.T) {
	dir := t.TempDir()
	writeStateRules(t, dir, "[", "design/TD-{id}.md", "test/TP-{id}.md", "build/IP-{id}.md")
	plugin := Plugin{FS: os.DirFS(dir), Base: "."}
	if _, err := runStateRules(dir, plugin, Check{ID: "state", StateRules: "rules.yml"}); err == nil {
		t.Fatalf("expected frame glob error")
	}
}

func TestRunStateRulesDesignGlobError(t *testing.T) {
	dir := t.TempDir()
	writeStateRules(t, dir, "frame/US-{id}.md", "[", "test/TP-{id}.md", "build/IP-{id}.md")
	plugin := Plugin{FS: os.DirFS(dir), Base: "."}
	if _, err := runStateRules(dir, plugin, Check{ID: "state", StateRules: "rules.yml"}); err == nil {
		t.Fatalf("expected design glob error")
	}
}

func TestRunStateRulesTestGlobError(t *testing.T) {
	dir := t.TempDir()
	writeStateRules(t, dir, "frame/US-{id}.md", "design/TD-{id}.md", "[", "build/IP-{id}.md")
	plugin := Plugin{FS: os.DirFS(dir), Base: "."}
	if _, err := runStateRules(dir, plugin, Check{ID: "state", StateRules: "rules.yml"}); err == nil {
		t.Fatalf("expected test glob error")
	}
}

func TestRunStateRulesBuildGlobError(t *testing.T) {
	dir := t.TempDir()
	writeStateRules(t, dir, "frame/US-{id}.md", "design/TD-{id}.md", "test/TP-{id}.md", "[")
	plugin := Plugin{FS: os.DirFS(dir), Base: "."}
	if _, err := runStateRules(dir, plugin, Check{ID: "state", StateRules: "rules.yml"}); err == nil {
		t.Fatalf("expected build glob error")
	}
}

func writeStateRules(t *testing.T, dir, frame, design, test, build string) {
	t.Helper()
	content := `artifact_patterns:
  story:
    frame: { pattern: "` + frame + `" }
    design: { pattern: "` + design + `" }
    test: { pattern: "` + test + `" }
    build: { pattern: "` + build + `" }
`
	writeFile(t, filepath.Join(dir, "rules.yml"), content)
}
