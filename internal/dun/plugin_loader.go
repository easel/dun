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

// LoadBuiltins loads all builtin plugins and external plugins.
// External plugins from the project directory override user plugins with the same ID.
func LoadBuiltins() ([]Plugin, error) {
	var plugins []Plugin
	for _, entry := range builtinPlugins() {
		p, err := loadPluginFS(entry.FS, entry.Base)
		if err != nil {
			return nil, err
		}
		plugins = append(plugins, p)
	}

	external, err := LoadExternalPlugins()
	if err != nil {
		return nil, err
	}
	plugins = append(plugins, external...)

	return plugins, nil
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
