package dun

import (
	"fmt"
	"io/fs"
	"path"

	"github.com/easel/dun/internal/plugins/builtin"
	"gopkg.in/yaml.v3"
)

var builtinPlugins = builtin.Plugins

func LoadBuiltins() ([]Plugin, error) {
	var plugins []Plugin
	for _, entry := range builtinPlugins() {
		p, err := loadPluginFS(entry.FS, entry.Base)
		if err != nil {
			return nil, err
		}
		plugins = append(plugins, p)
	}
	return plugins, nil
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
