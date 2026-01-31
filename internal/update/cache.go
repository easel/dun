package update

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

const (
	// CacheTTL is how long the cache is valid.
	CacheTTL = 1 * time.Hour

	// CacheFileName is the name of the cache file.
	CacheFileName = "update-cache.json"
)

// Cache stores update check results to avoid frequent API calls.
type Cache struct {
	LastCheck     time.Time `json:"last_check"`
	LatestVersion string    `json:"latest_version"`
	UpdateAvail   bool      `json:"update_available"`
}

// CachePath returns the path to the cache file.
func CachePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".dun", CacheFileName), nil
}

// Load reads the cache from disk.
func (c *Cache) Load() error {
	path, err := CachePath()
	if err != nil {
		return err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// Return empty cache, not an error
			return nil
		}
		return err
	}

	return json.Unmarshal(data, c)
}

// LoadFrom reads the cache from a specific path.
func (c *Cache) LoadFrom(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return json.Unmarshal(data, c)
}

// Save writes the cache to disk.
func (c *Cache) Save() error {
	path, err := CachePath()
	if err != nil {
		return err
	}
	return c.SaveTo(path)
}

// SaveTo writes the cache to a specific path.
func (c *Cache) SaveTo(path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// IsStale returns true if the cache is older than CacheTTL.
func (c *Cache) IsStale() bool {
	if c.LastCheck.IsZero() {
		return true
	}
	return time.Since(c.LastCheck) > CacheTTL
}

// IsStaleWithTTL returns true if the cache is older than the given TTL.
func (c *Cache) IsStaleWithTTL(ttl time.Duration) bool {
	if c.LastCheck.IsZero() {
		return true
	}
	return time.Since(c.LastCheck) > ttl
}

// Update sets the cache values and updates LastCheck.
func (c *Cache) Update(version string, available bool) {
	c.LastCheck = time.Now()
	c.LatestVersion = version
	c.UpdateAvail = available
}

// Clear resets the cache.
func (c *Cache) Clear() {
	c.LastCheck = time.Time{}
	c.LatestVersion = ""
	c.UpdateAvail = false
}
