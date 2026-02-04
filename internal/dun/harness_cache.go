package dun

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"time"
)

const HarnessCacheFileName = "harnesses.json"

// HarnessCache stores availability information about agent harnesses.
type HarnessCache struct {
	LastCheck time.Time       `json:"last_check"`
	Harnesses []HarnessStatus `json:"harnesses"`
}

// HarnessCachePath returns the path to the harness cache file.
func HarnessCachePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".dun", HarnessCacheFileName), nil
}

// LoadHarnessCache reads the harness cache from disk.
func LoadHarnessCache() (HarnessCache, error) {
	var cache HarnessCache
	path, err := HarnessCachePath()
	if err != nil {
		return cache, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cache, nil
		}
		return cache, err
	}
	if err := json.Unmarshal(data, &cache); err != nil {
		return cache, err
	}
	return cache, nil
}

// Save writes the harness cache to disk.
func (c *HarnessCache) Save() error {
	path, err := HarnessCachePath()
	if err != nil {
		return err
	}
	return c.SaveTo(path)
}

// SaveTo writes the harness cache to a specific path.
func (c *HarnessCache) SaveTo(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	sorted := make([]HarnessStatus, 0, len(c.Harnesses))
	sorted = append(sorted, c.Harnesses...)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Name < sorted[j].Name
	})
	copyCache := HarnessCache{
		LastCheck: c.LastCheck,
		Harnesses: sorted,
	}
	data, err := json.MarshalIndent(copyCache, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// AvailableHarnesses returns a sorted list of available harness names.
func (c HarnessCache) AvailableHarnesses() []string {
	var names []string
	for _, harness := range c.Harnesses {
		if harness.Available {
			names = append(names, harness.Name)
		}
	}
	sort.Strings(names)
	return names
}
