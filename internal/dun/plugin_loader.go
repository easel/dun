package dun

import (
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path"
	"path/filepath"

	"github.com/easel/dun/internal/plugins/builtin"
	"gopkg.in/yaml.v3"
)

var builtinPlugins = builtin.Plugins

// LoadBuiltins loads all builtin plugins, cached plugins, and external plugins.
// Priority (lowest to highest): builtin < cached < user < project.
// External plugins from the project directory override user plugins with the same ID.
func LoadBuiltins() ([]Plugin, error) {
	var plugins []Plugin
	seen := make(map[string]int) // ID -> index in plugins slice

	// 1. Load builtin plugins (lowest priority)
	for _, entry := range builtinPlugins() {
		p, err := loadPluginFS(entry.FS, entry.Base)
		if err != nil {
			return nil, err
		}
		seen[p.Manifest.ID] = len(plugins)
		plugins = append(plugins, p)
	}

	// 2. Load cached plugins from ~/.cache/ddx/library (overrides builtins)
	cached, err := LoadCachedPlugins()
	if err != nil {
		return nil, err
	}
	for _, p := range cached {
		if idx, ok := seen[p.Manifest.ID]; ok {
			plugins[idx] = p
		} else {
			seen[p.Manifest.ID] = len(plugins)
			plugins = append(plugins, p)
		}
	}

	// 3. Load external plugins (user and project - highest priority)
	external, err := LoadExternalPlugins()
	if err != nil {
		return nil, err
	}
	for _, p := range external {
		if idx, ok := seen[p.Manifest.ID]; ok {
			plugins[idx] = p
		} else {
			seen[p.Manifest.ID] = len(plugins)
			plugins = append(plugins, p)
		}
	}

	return plugins, nil
}

// LoadCachedPlugins loads plugins from the ddx library cache.
// Returns plugins found in ~/.cache/ddx/library/plugins/*/plugin.yaml.
// Returns an empty slice (not an error) if the cache directory doesn't exist.
func LoadCachedPlugins() ([]Plugin, error) {
	cacheDir, err := getCacheDir()
	if err != nil {
		// Can't determine cache dir, skip cached plugins
		return nil, nil
	}

	libraryDir := filepath.Join(cacheDir, "ddx", "library", "plugins")
	if _, err := os.Stat(libraryDir); os.IsNotExist(err) {
		// Cache doesn't exist - log a helpful warning
		slog.Debug("ddx library cache not found; run 'ddx update' to populate", "path", libraryDir)
		return nil, nil
	}

	plugins, err := loadPluginsFromDir(libraryDir)
	if err != nil {
		return nil, err
	}

	if len(plugins) > 0 {
		slog.Debug("loaded cached plugins from ddx library", "count", len(plugins), "path", libraryDir)
	}

	return plugins, nil
}

// getCacheDir returns the user cache directory.
// It uses os.UserCacheDir() which returns:
// - $XDG_CACHE_HOME or ~/.cache on Linux
// - ~/Library/Caches on macOS
// - %LocalAppData% on Windows
func getCacheDir() (string, error) {
	return os.UserCacheDir()
}

// LoadExternalPlugins loads plugins from user and project directories.
// Project plugins override user plugins with the same ID.
func LoadExternalPlugins() ([]Plugin, error) {
	var plugins []Plugin
	seen := make(map[string]int) // ID -> index in plugins slice

	// User plugins: ~/.dun/plugins/*/plugin.yaml
	homeDir, err := os.UserHomeDir()
	if err == nil {
		userDir := filepath.Join(homeDir, ".dun", "plugins")
		userPlugins, _ := loadPluginsFromDir(userDir)
		for _, p := range userPlugins {
			seen[p.Manifest.ID] = len(plugins)
			plugins = append(plugins, p)
		}
	}

	// Project plugins: .dun/plugins/*/plugin.yaml
	projectDir := ".dun/plugins"
	projectPlugins, _ := loadPluginsFromDir(projectDir)
	for _, p := range projectPlugins {
		if idx, ok := seen[p.Manifest.ID]; ok {
			// Project plugin overrides user plugin with same ID
			plugins[idx] = p
		} else {
			seen[p.Manifest.ID] = len(plugins)
			plugins = append(plugins, p)
		}
	}

	return plugins, nil
}

// loadPluginsFromDir loads all plugins from subdirectories of dir.
// Returns nil if the directory doesn't exist.
func loadPluginsFromDir(dir string) ([]Plugin, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, nil // Not an error if dir doesn't exist
	}

	var plugins []Plugin
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		pluginDir := filepath.Join(dir, entry.Name())
		p, err := loadPluginFromPath(pluginDir)
		if err != nil {
			slog.Warn("skipping invalid plugin", "path", pluginDir, "error", err)
			continue
		}
		plugins = append(plugins, p)
	}
	return plugins, nil
}

// loadPluginFromPath loads a plugin from a directory path.
func loadPluginFromPath(pluginPath string) (Plugin, error) {
	manifestPath := filepath.Join(pluginPath, "plugin.yaml")
	raw, err := os.ReadFile(manifestPath)
	if err != nil {
		return Plugin{}, fmt.Errorf("read plugin manifest: %w", err)
	}

	var manifest Manifest
	if err := yaml.Unmarshal(raw, &manifest); err != nil {
		return Plugin{}, fmt.Errorf("parse plugin manifest: %w", err)
	}

	if manifest.ID == "" || manifest.Version == "" {
		return Plugin{}, fmt.Errorf("invalid plugin manifest: missing id or version")
	}
	if len(manifest.Checks) == 0 {
		return Plugin{}, fmt.Errorf("invalid plugin manifest: no checks defined")
	}

	return Plugin{
		Manifest: manifest,
		FS:       os.DirFS(pluginPath),
		Base:     ".",
	}, nil
}

func loadPluginFS(pluginFS fs.FS, base string) (Plugin, error) {
	manifestPath := path.Join(base, "plugin.yaml")
	raw, err := fs.ReadFile(pluginFS, manifestPath)
	if err != nil {
		return Plugin{}, fmt.Errorf("read plugin manifest: %w", err)
	}

	var manifest Manifest
	if err := yaml.Unmarshal(raw, &manifest); err != nil {
		return Plugin{}, fmt.Errorf("parse plugin manifest: %w", err)
	}

	if manifest.ID == "" || manifest.Version == "" {
		return Plugin{}, fmt.Errorf("invalid plugin manifest: missing id or version")
	}
	if len(manifest.Checks) == 0 {
		return Plugin{}, fmt.Errorf("invalid plugin manifest: no checks defined")
	}

	return Plugin{
		Manifest: manifest,
		FS:       pluginFS,
		Base:     base,
	}, nil
}
