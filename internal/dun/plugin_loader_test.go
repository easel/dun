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

func TestLoadCachedPlugins(t *testing.T) {
	// Create temp cache directory structure
	tmpCache := t.TempDir()
	libraryDir := filepath.Join(tmpCache, "ddx", "library", "plugins", "helix")
	if err := os.MkdirAll(libraryDir, 0755); err != nil {
		t.Fatalf("create library dir: %v", err)
	}

	manifest := `id: helix
version: "1"
description: "Helix workflow template"
checks:
  - id: helix-check
    type: command
    command: echo helix
`
	if err := os.WriteFile(filepath.Join(libraryDir, "plugin.yaml"), []byte(manifest), 0644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	// Override XDG_CACHE_HOME to use our temp dir
	origCache := os.Getenv("XDG_CACHE_HOME")
	os.Setenv("XDG_CACHE_HOME", tmpCache)
	t.Cleanup(func() { os.Setenv("XDG_CACHE_HOME", origCache) })

	plugins, err := LoadCachedPlugins()
	if err != nil {
		t.Fatalf("load cached plugins: %v", err)
	}
	if len(plugins) != 1 {
		t.Fatalf("expected 1 plugin, got %d", len(plugins))
	}
	if plugins[0].Manifest.ID != "helix" {
		t.Fatalf("expected id=helix, got %s", plugins[0].Manifest.ID)
	}
}

func TestLoadCachedPluginsMissingCache(t *testing.T) {
	// Use a non-existent cache directory
	tmpCache := t.TempDir()
	origCache := os.Getenv("XDG_CACHE_HOME")
	os.Setenv("XDG_CACHE_HOME", tmpCache)
	t.Cleanup(func() { os.Setenv("XDG_CACHE_HOME", origCache) })

	// No ddx/library directory exists - should return empty slice, not error
	plugins, err := LoadCachedPlugins()
	if err != nil {
		t.Fatalf("expected nil error for missing cache, got %v", err)
	}
	if len(plugins) != 0 {
		t.Fatalf("expected 0 plugins for missing cache, got %d", len(plugins))
	}
}

func TestCachedPluginPriority(t *testing.T) {
	// Setup: create cached plugin
	tmpCache := t.TempDir()
	cachedPluginDir := filepath.Join(tmpCache, "ddx", "library", "plugins", "shared")
	if err := os.MkdirAll(cachedPluginDir, 0755); err != nil {
		t.Fatalf("create cached plugin dir: %v", err)
	}
	cachedManifest := `id: shared
version: "1"
description: "Cached version"
checks:
  - id: cachedcheck
    type: command
    command: echo cached
`
	if err := os.WriteFile(filepath.Join(cachedPluginDir, "plugin.yaml"), []byte(cachedManifest), 0644); err != nil {
		t.Fatalf("write cached manifest: %v", err)
	}

	// Setup: create project plugin with same ID (should override cached)
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

	// Override environment
	origCache := os.Getenv("XDG_CACHE_HOME")
	os.Setenv("XDG_CACHE_HOME", tmpCache)
	t.Cleanup(func() { os.Setenv("XDG_CACHE_HOME", origCache) })

	origHome := os.Getenv("HOME")
	emptyHome := t.TempDir() // Empty home so no user plugins
	os.Setenv("HOME", emptyHome)
	t.Cleanup(func() { os.Setenv("HOME", origHome) })

	origDir, _ := os.Getwd()
	if err := os.Chdir(projectDir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() { os.Chdir(origDir) })

	// Override builtins to empty
	origBuiltins := builtinPlugins
	builtinPlugins = func() []builtin.Entry { return nil }
	t.Cleanup(func() { builtinPlugins = origBuiltins })

	plugins, err := LoadBuiltins()
	if err != nil {
		t.Fatalf("load builtins: %v", err)
	}

	// Should have 1 plugin (merged)
	if len(plugins) != 1 {
		t.Fatalf("expected 1 plugin (merged), got %d", len(plugins))
	}

	// Project should override cached
	if plugins[0].Manifest.Version != "2" {
		t.Fatalf("expected version=2 (project override), got %s", plugins[0].Manifest.Version)
	}
	if plugins[0].Manifest.Description != "Project version" {
		t.Fatalf("expected project description, got %s", plugins[0].Manifest.Description)
	}
}

func TestCachedPluginOverridesBuiltin(t *testing.T) {
	// Setup: create cached plugin
	tmpCache := t.TempDir()
	cachedPluginDir := filepath.Join(tmpCache, "ddx", "library", "plugins", "testplugin")
	if err := os.MkdirAll(cachedPluginDir, 0755); err != nil {
		t.Fatalf("create cached plugin dir: %v", err)
	}
	cachedManifest := `id: testplugin
version: "2"
description: "Cached version"
checks:
  - id: cachedcheck
    type: command
    command: echo cached
`
	if err := os.WriteFile(filepath.Join(cachedPluginDir, "plugin.yaml"), []byte(cachedManifest), 0644); err != nil {
		t.Fatalf("write cached manifest: %v", err)
	}

	// Override environment
	origCache := os.Getenv("XDG_CACHE_HOME")
	os.Setenv("XDG_CACHE_HOME", tmpCache)
	t.Cleanup(func() { os.Setenv("XDG_CACHE_HOME", origCache) })

	origHome := os.Getenv("HOME")
	emptyHome := t.TempDir()
	os.Setenv("HOME", emptyHome)
	t.Cleanup(func() { os.Setenv("HOME", origHome) })

	emptyProject := t.TempDir()
	origDir, _ := os.Getwd()
	if err := os.Chdir(emptyProject); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() { os.Chdir(origDir) })

	// Create a fake builtin with same ID
	origBuiltins := builtinPlugins
	builtinPlugins = func() []builtin.Entry {
		return []builtin.Entry{
			{
				ID:   "testplugin",
				Base: ".",
				FS: fstest.MapFS{
					"plugin.yaml": {Data: []byte(`id: testplugin
version: "1"
description: "Builtin version"
checks:
  - id: builtincheck
    type: command
    command: echo builtin
`)},
				},
			},
		}
	}
	t.Cleanup(func() { builtinPlugins = origBuiltins })

	plugins, err := LoadBuiltins()
	if err != nil {
		t.Fatalf("load builtins: %v", err)
	}

	// Should have 1 plugin
	if len(plugins) != 1 {
		t.Fatalf("expected 1 plugin, got %d", len(plugins))
	}

	// Cached should override builtin
	if plugins[0].Manifest.Version != "2" {
		t.Fatalf("expected version=2 (cached override), got %s", plugins[0].Manifest.Version)
	}
	if plugins[0].Manifest.Description != "Cached version" {
		t.Fatalf("expected cached description, got %s", plugins[0].Manifest.Description)
	}
}
