package builtin

import "testing"

func TestPluginsIncludesBuiltins(t *testing.T) {
	plugins := Plugins()
	if len(plugins) < 3 {
		t.Fatalf("expected builtin plugins")
	}
	found := map[string]bool{}
	for _, plugin := range plugins {
		found[plugin.ID] = true
		if plugin.Base == "" {
			t.Fatalf("expected base for plugin %s", plugin.ID)
		}
	}
	for _, id := range []string{"helix", "git", "go"} {
		if !found[id] {
			t.Fatalf("expected plugin %s", id)
		}
	}
}
