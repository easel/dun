package dun

import (
	"io/fs"
	"os"
	"path/filepath"
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

func TestLoadExternalPlugins(t *testing.T) {
	// Create temp directory structure
	tmpDir := t.TempDir()
	pluginDir := filepath.Join(tmpDir, "myplugin")
	if err := os.MkdirAll(pluginDir, 0755); err != nil {
		t.Fatalf("create plugin dir: %v", err)
	}

	manifest := `id: myplugin
version: "1"
description: "Test plugin"
checks:
  - id: mycheck
    type: command
    command: echo hello
`
	if err := os.WriteFile(filepath.Join(pluginDir, "plugin.yaml"), []byte(manifest), 0644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	plugins, err := loadPluginsFromDir(tmpDir)
	if err != nil {
		t.Fatalf("load plugins: %v", err)
	}
	if len(plugins) != 1 {
		t.Fatalf("expected 1 plugin, got %d", len(plugins))
	}
	if plugins[0].Manifest.ID != "myplugin" {
		t.Fatalf("expected id=myplugin, got %s", plugins[0].Manifest.ID)
	}
}

func TestLoadPluginsFromDirMissingDir(t *testing.T) {
	plugins, err := loadPluginsFromDir("/nonexistent/path")
	if err != nil {
		t.Fatalf("expected nil error for missing dir, got %v", err)
	}
	if plugins != nil {
		t.Fatalf("expected nil plugins for missing dir, got %v", plugins)
	}
}

func TestLoadPluginsFromDirInvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	pluginDir := filepath.Join(tmpDir, "badplugin")
	if err := os.MkdirAll(pluginDir, 0755); err != nil {
		t.Fatalf("create plugin dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pluginDir, "plugin.yaml"), []byte(":invalid:"), 0644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	// Should skip invalid plugins without error
	plugins, err := loadPluginsFromDir(tmpDir)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if len(plugins) != 0 {
		t.Fatalf("expected 0 plugins, got %d", len(plugins))
	}
}

func TestLoadPluginFromPath(t *testing.T) {
	tmpDir := t.TempDir()
	manifest := `id: testplugin
version: "2"
description: "A test plugin"
priority: 10
checks:
  - id: testcheck
    type: command
    command: true
`
	if err := os.WriteFile(filepath.Join(tmpDir, "plugin.yaml"), []byte(manifest), 0644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	plugin, err := loadPluginFromPath(tmpDir)
	if err != nil {
		t.Fatalf("load plugin: %v", err)
	}
	if plugin.Manifest.ID != "testplugin" {
		t.Fatalf("expected id=testplugin, got %s", plugin.Manifest.ID)
	}
	if plugin.Manifest.Version != "2" {
		t.Fatalf("expected version=2, got %s", plugin.Manifest.Version)
	}
	if plugin.Base != "." {
		t.Fatalf("expected base=., got %s", plugin.Base)
	}
}

func TestLoadPluginFromPathMissingFile(t *testing.T) {
	tmpDir := t.TempDir()
	_, err := loadPluginFromPath(tmpDir)
	if err == nil {
		t.Fatalf("expected error for missing manifest")
	}
}

func TestLoadPluginFromPathMissingID(t *testing.T) {
	tmpDir := t.TempDir()
	manifest := `version: "1"
checks:
  - id: check
`
	if err := os.WriteFile(filepath.Join(tmpDir, "plugin.yaml"), []byte(manifest), 0644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	_, err := loadPluginFromPath(tmpDir)
	if err == nil {
		t.Fatalf("expected error for missing id")
	}
}

func TestLoadPluginFromPathNoChecks(t *testing.T) {
	tmpDir := t.TempDir()
	manifest := `id: test
version: "1"
`
	if err := os.WriteFile(filepath.Join(tmpDir, "plugin.yaml"), []byte(manifest), 0644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	_, err := loadPluginFromPath(tmpDir)
	if err == nil {
		t.Fatalf("expected error for no checks")
	}
}

func TestProjectOverridesUserPlugin(t *testing.T) {
	// Create user plugins dir with a plugin
	userHome := t.TempDir()
	userPluginDir := filepath.Join(userHome, ".dun", "plugins", "shared")
	if err := os.MkdirAll(userPluginDir, 0755); err != nil {
		t.Fatalf("create user plugin dir: %v", err)
	}
	userManifest := `id: shared
version: "1"
description: "User version"
checks:
  - id: usercheck
    type: command
    command: echo user
`
	if err := os.WriteFile(filepath.Join(userPluginDir, "plugin.yaml"), []byte(userManifest), 0644); err != nil {
		t.Fatalf("write user manifest: %v", err)
	}

	// Create project plugins dir with same plugin ID
	projectDir := t.TempDir()
	projectPluginDir := filepath.Join(projectDir, ".dun", "plugins", "shared")
	if err := os.MkdirAll(projectPluginDir, 0755); err != nil {
		t.Fatalf("create project plugin dir: %v", err)
	}
	projectManifest := `id: shared
version: "2"
description: "Project version"
checks:
  - id: projectcheck
    type: command
    command: echo project
`
	if err := os.WriteFile(filepath.Join(projectPluginDir, "plugin.yaml"), []byte(projectManifest), 0644); err != nil {
		t.Fatalf("write project manifest: %v", err)
	}

	// Change to project dir and test
	origDir, _ := os.Getwd()
	if err := os.Chdir(projectDir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() { os.Chdir(origDir) })

	// Override HOME to use our temp user dir
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", userHome)
	t.Cleanup(func() { os.Setenv("HOME", origHome) })

	plugins, err := LoadExternalPlugins()
	if err != nil {
		t.Fatalf("load external plugins: %v", err)
	}

	if len(plugins) != 1 {
		t.Fatalf("expected 1 plugin (merged), got %d", len(plugins))
	}

	// Project should override user
	if plugins[0].Manifest.Version != "2" {
		t.Fatalf("expected version=2 (project override), got %s", plugins[0].Manifest.Version)
	}
	if plugins[0].Manifest.Description != "Project version" {
		t.Fatalf("expected project description, got %s", plugins[0].Manifest.Description)
	}
}

func TestLoadPluginsFromDirSkipsFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a regular file (not a directory)
	if err := os.WriteFile(filepath.Join(tmpDir, "notadir"), []byte("hello"), 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	plugins, err := loadPluginsFromDir(tmpDir)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if len(plugins) != 0 {
		t.Fatalf("expected 0 plugins, got %d", len(plugins))
	}
}
