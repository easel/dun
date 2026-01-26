package dun

import (
	"io/fs"
	"testing"
	"testing/fstest"

	"github.com/easel/dun/internal/plugins/builtin"
)

func TestLoadPluginFSReadError(t *testing.T) {
	_, err := loadPluginFS(fstest.MapFS{}, "missing")
	if err == nil {
		t.Fatalf("expected read error")
	}
}

func TestLoadPluginFSInvalidYAML(t *testing.T) {
	fs := fstest.MapFS{
		"plugin.yaml": {Data: []byte(":")},
	}
	_, err := loadPluginFS(fs, ".")
	if err == nil {
		t.Fatalf("expected parse error")
	}
}

func TestLoadPluginFSMissingFields(t *testing.T) {
	fs := fstest.MapFS{
		"plugin.yaml": {Data: []byte("id: \"\"\nversion: \"\"\nchecks:\n  - id: test\n")},
	}
	_, err := loadPluginFS(fs, ".")
	if err == nil {
		t.Fatalf("expected missing fields error")
	}
}

func TestLoadPluginFSNoChecks(t *testing.T) {
	fs := fstest.MapFS{
		"plugin.yaml": {Data: []byte("id: test\nversion: \"1\"")},
	}
	_, err := loadPluginFS(fs, ".")
	if err == nil {
		t.Fatalf("expected missing checks error")
	}
}

func TestLoadPluginFSSuccess(t *testing.T) {
	fs := fstest.MapFS{
		"plugin.yaml": {Data: []byte("id: test\nversion: \"1\"\nchecks:\n  - id: check\n    type: rule-set\n    description: \"x\"\n")},
	}
	plugin, err := loadPluginFS(fs, ".")
	if err != nil {
		t.Fatalf("load plugin: %v", err)
	}
	if plugin.Manifest.ID != "test" {
		t.Fatalf("unexpected id")
	}
}

func TestLoadBuiltinsError(t *testing.T) {
	orig := builtinPlugins
	builtinPlugins = func() []builtin.Entry {
		return []builtin.Entry{
			{ID: "bad", FS: fstest.MapFS{}, Base: "."},
		}
	}
	t.Cleanup(func() { builtinPlugins = orig })

	if _, err := LoadBuiltins(); err == nil {
		t.Fatalf("expected LoadBuiltins error")
	}
}

func TestLoadBuiltinsSuccess(t *testing.T) {
	orig := builtinPlugins
	builtinPlugins = builtin.Plugins
	t.Cleanup(func() { builtinPlugins = orig })

	plugins, err := LoadBuiltins()
	if err != nil {
		t.Fatalf("load builtins: %v", err)
	}
	if len(plugins) == 0 {
		t.Fatalf("expected plugins")
	}
}

var _ fs.FS = fstest.MapFS{}
