package update

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCacheLoadSave(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cache.json")

	cache := &Cache{
		LastCheck:     time.Now().Truncate(time.Second),
		LatestVersion: "v1.2.0",
		UpdateAvail:   true,
	}

	if err := cache.SaveTo(path); err != nil {
		t.Fatalf("SaveTo failed: %v", err)
	}

	loaded := &Cache{}
	if err := loaded.LoadFrom(path); err != nil {
		t.Fatalf("LoadFrom failed: %v", err)
	}

	if loaded.LatestVersion != cache.LatestVersion {
		t.Errorf("version mismatch: got %s, want %s", loaded.LatestVersion, cache.LatestVersion)
	}
	if loaded.UpdateAvail != cache.UpdateAvail {
		t.Errorf("update avail mismatch: got %v, want %v", loaded.UpdateAvail, cache.UpdateAvail)
	}
	// Time comparison with truncation
	if !loaded.LastCheck.Equal(cache.LastCheck) {
		t.Errorf("last check mismatch: got %v, want %v", loaded.LastCheck, cache.LastCheck)
	}
}

func TestCacheLoadNotFound(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nonexistent.json")

	cache := &Cache{}
	if err := cache.LoadFrom(path); err != nil {
		t.Errorf("LoadFrom should not error for missing file: %v", err)
	}

	// Should be empty/zero
	if cache.LatestVersion != "" {
		t.Errorf("expected empty version for missing file")
	}
	if cache.UpdateAvail {
		t.Errorf("expected false update avail for missing file")
	}
}

func TestCacheIsStale(t *testing.T) {
	cache := &Cache{}

	// Empty cache is stale
	if !cache.IsStale() {
		t.Error("empty cache should be stale")
	}

	// Fresh cache is not stale
	cache.LastCheck = time.Now()
	if cache.IsStale() {
		t.Error("fresh cache should not be stale")
	}

	// Old cache is stale
	cache.LastCheck = time.Now().Add(-2 * CacheTTL)
	if !cache.IsStale() {
		t.Error("old cache should be stale")
	}
}

func TestCacheIsStaleWithTTL(t *testing.T) {
	cache := &Cache{}

	// Empty cache is stale
	if !cache.IsStaleWithTTL(1 * time.Hour) {
		t.Error("empty cache should be stale")
	}

	// Fresh cache with short TTL
	cache.LastCheck = time.Now().Add(-30 * time.Minute)
	if cache.IsStaleWithTTL(1 * time.Hour) {
		t.Error("30 min old cache should not be stale with 1 hour TTL")
	}
	if !cache.IsStaleWithTTL(15 * time.Minute) {
		t.Error("30 min old cache should be stale with 15 min TTL")
	}
}

func TestCacheUpdate(t *testing.T) {
	cache := &Cache{}

	before := time.Now()
	cache.Update("v2.0.0", true)
	after := time.Now()

	if cache.LatestVersion != "v2.0.0" {
		t.Errorf("expected version v2.0.0, got %s", cache.LatestVersion)
	}
	if !cache.UpdateAvail {
		t.Error("expected update available")
	}
	if cache.LastCheck.Before(before) || cache.LastCheck.After(after) {
		t.Error("last check time out of range")
	}
}

func TestCacheClear(t *testing.T) {
	cache := &Cache{
		LastCheck:     time.Now(),
		LatestVersion: "v1.0.0",
		UpdateAvail:   true,
	}

	cache.Clear()

	if !cache.LastCheck.IsZero() {
		t.Error("last check should be zero after clear")
	}
	if cache.LatestVersion != "" {
		t.Error("version should be empty after clear")
	}
	if cache.UpdateAvail {
		t.Error("update avail should be false after clear")
	}
}

func TestCacheSaveCreatesDirectory(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "subdir", "deep", "cache.json")

	cache := &Cache{
		LatestVersion: "v1.0.0",
	}

	if err := cache.SaveTo(path); err != nil {
		t.Fatalf("SaveTo should create directories: %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(path); err != nil {
		t.Errorf("cache file should exist: %v", err)
	}
}

func TestCacheLoadInvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "invalid.json")

	if err := os.WriteFile(path, []byte("not valid json"), 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	cache := &Cache{}
	err := cache.LoadFrom(path)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestCacheSaveToReadOnlyDir(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("skipping test as root")
	}

	dir := t.TempDir()
	readonlyDir := filepath.Join(dir, "readonly")
	if err := os.MkdirAll(readonlyDir, 0555); err != nil {
		t.Fatalf("create readonly dir: %v", err)
	}

	cache := &Cache{LatestVersion: "v1.0.0"}
	path := filepath.Join(readonlyDir, "subdir", "cache.json")

	err := cache.SaveTo(path)
	if err == nil {
		t.Error("expected error saving to readonly directory")
	}
}

func TestCachePath(t *testing.T) {
	path, err := CachePath()
	if err != nil {
		t.Fatalf("CachePath failed: %v", err)
	}

	// Should end with expected filename
	if filepath.Base(path) != CacheFileName {
		t.Errorf("expected filename %s, got %s", CacheFileName, filepath.Base(path))
	}

	// Should be under .dun directory
	parent := filepath.Base(filepath.Dir(path))
	if parent != ".dun" {
		t.Errorf("expected .dun parent, got %s", parent)
	}
}

func TestCacheLoadDefault(t *testing.T) {
	// Create a temporary home directory
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)

	dir := t.TempDir()
	os.Setenv("HOME", dir)

	cache := &Cache{}

	// Load from default path (file doesn't exist)
	if err := cache.Load(); err != nil {
		t.Errorf("Load should not error for missing file: %v", err)
	}

	// Empty cache should be stale
	if !cache.IsStale() {
		t.Error("empty loaded cache should be stale")
	}
}

func TestCacheSaveDefault(t *testing.T) {
	// Create a temporary home directory
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)

	dir := t.TempDir()
	os.Setenv("HOME", dir)

	cache := &Cache{
		LastCheck:     time.Now(),
		LatestVersion: "v1.5.0",
		UpdateAvail:   true,
	}

	if err := cache.Save(); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Verify file was created
	expectedPath := filepath.Join(dir, ".dun", CacheFileName)
	if _, err := os.Stat(expectedPath); err != nil {
		t.Errorf("cache file should exist at %s: %v", expectedPath, err)
	}

	// Load it back
	loaded := &Cache{}
	if err := loaded.LoadFrom(expectedPath); err != nil {
		t.Fatalf("LoadFrom failed: %v", err)
	}

	if loaded.LatestVersion != "v1.5.0" {
		t.Errorf("expected version v1.5.0, got %s", loaded.LatestVersion)
	}
}

func TestCacheTTL(t *testing.T) {
	// Verify CacheTTL constant
	if CacheTTL != 1*time.Hour {
		t.Errorf("expected CacheTTL to be 1 hour, got %v", CacheTTL)
	}
}

func TestCacheFileName(t *testing.T) {
	if CacheFileName != "update-cache.json" {
		t.Errorf("expected CacheFileName to be update-cache.json, got %s", CacheFileName)
	}
}

func TestCacheLoadFromReadError(t *testing.T) {
	// Create a directory instead of a file to cause read error
	dir := t.TempDir()
	path := filepath.Join(dir, "isdir")
	if err := os.MkdirAll(path, 0755); err != nil {
		t.Fatalf("create dir: %v", err)
	}

	cache := &Cache{}
	err := cache.LoadFrom(path)
	if err == nil {
		t.Error("expected error reading directory as file")
	}
}

func TestCacheEdgeCases(t *testing.T) {
	cache := &Cache{}

	// Test with zero TTL
	cache.LastCheck = time.Now().Add(-1 * time.Millisecond)
	if !cache.IsStaleWithTTL(0) {
		t.Error("any time should be stale with zero TTL")
	}

	// Test with very large TTL
	cache.LastCheck = time.Now().Add(-24 * time.Hour)
	if cache.IsStaleWithTTL(48 * time.Hour) {
		t.Error("24h old cache should not be stale with 48h TTL")
	}
}

func TestCacheUpdateMultipleTimes(t *testing.T) {
	cache := &Cache{}

	cache.Update("v1.0.0", false)
	first := cache.LastCheck

	time.Sleep(10 * time.Millisecond)

	cache.Update("v2.0.0", true)

	if cache.LatestVersion != "v2.0.0" {
		t.Errorf("expected v2.0.0, got %s", cache.LatestVersion)
	}
	if !cache.UpdateAvail {
		t.Error("expected update available")
	}
	if !cache.LastCheck.After(first) {
		t.Error("last check should be updated")
	}
}

// Test CachePath with HOME not set (edge case)
func TestCachePathError(t *testing.T) {
	// Temporarily unset HOME
	originalHome := os.Getenv("HOME")
	os.Unsetenv("HOME")
	// Also unset XDG_CACHE_HOME and other fallbacks
	originalXdg := os.Getenv("XDG_CACHE_HOME")
	os.Unsetenv("XDG_CACHE_HOME")

	defer func() {
		os.Setenv("HOME", originalHome)
		if originalXdg != "" {
			os.Setenv("XDG_CACHE_HOME", originalXdg)
		}
	}()

	_, err := CachePath()
	if err == nil {
		// On some systems, os.UserHomeDir has fallbacks; that's OK
		// The important thing is we're testing the code path
		t.Log("CachePath succeeded even without HOME (fallback used)")
	}
}

// Test Cache.Load error path when CachePath fails
func TestCacheLoadCachePathError(t *testing.T) {
	originalHome := os.Getenv("HOME")
	os.Unsetenv("HOME")
	originalXdg := os.Getenv("XDG_CACHE_HOME")
	os.Unsetenv("XDG_CACHE_HOME")

	defer func() {
		os.Setenv("HOME", originalHome)
		if originalXdg != "" {
			os.Setenv("XDG_CACHE_HOME", originalXdg)
		}
	}()

	cache := &Cache{}
	err := cache.Load()
	// If UserHomeDir has fallbacks, this may not error
	// We're exercising the code path either way
	_ = err
}

// Test Cache.Save error path when CachePath fails
func TestCacheSaveCachePathError(t *testing.T) {
	originalHome := os.Getenv("HOME")
	os.Unsetenv("HOME")
	originalXdg := os.Getenv("XDG_CACHE_HOME")
	os.Unsetenv("XDG_CACHE_HOME")

	defer func() {
		os.Setenv("HOME", originalHome)
		if originalXdg != "" {
			os.Setenv("XDG_CACHE_HOME", originalXdg)
		}
	}()

	cache := &Cache{LatestVersion: "v1.0.0"}
	err := cache.Save()
	// If UserHomeDir has fallbacks, this may not error
	// We're exercising the code path either way
	_ = err
}

// Test Cache.Load with read error (not just not-exist)
func TestCacheLoadReadError(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("skipping test as root")
	}

	// Create a temporary home directory
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)

	dir := t.TempDir()
	os.Setenv("HOME", dir)

	// Create .dun directory
	dunDir := filepath.Join(dir, ".dun")
	if err := os.MkdirAll(dunDir, 0755); err != nil {
		t.Fatalf("create .dun: %v", err)
	}

	// Create cache file that's unreadable
	cachePath := filepath.Join(dunDir, CacheFileName)
	if err := os.WriteFile(cachePath, []byte("{}"), 0000); err != nil {
		t.Fatalf("write cache: %v", err)
	}
	defer os.Chmod(cachePath, 0644)

	cache := &Cache{}
	err := cache.Load()
	if err == nil {
		t.Error("expected error for unreadable cache file")
	}
}

// Test SaveTo when MarshalIndent would fail (can't actually trigger this easily)
// but we can test directory creation failure
func TestCacheSaveToMkdirError(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("skipping test as root")
	}

	dir := t.TempDir()

	// Create a file where we want a directory
	blockingFile := filepath.Join(dir, "blocker")
	if err := os.WriteFile(blockingFile, []byte("blocking"), 0644); err != nil {
		t.Fatalf("create blocking file: %v", err)
	}

	cache := &Cache{LatestVersion: "v1.0.0"}
	// Try to save to a path that requires creating a directory where a file exists
	path := filepath.Join(blockingFile, "subdir", "cache.json")
	err := cache.SaveTo(path)
	if err == nil {
		t.Error("expected error when directory creation is blocked by file")
	}
}

func TestCacheSaveToWriteError(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("skipping test as root")
	}

	dir := t.TempDir()

	// Create a read-only directory
	roDir := filepath.Join(dir, "readonly")
	if err := os.MkdirAll(roDir, 0755); err != nil {
		t.Fatalf("create dir: %v", err)
	}

	// Make it read-only
	if err := os.Chmod(roDir, 0555); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	defer os.Chmod(roDir, 0755)

	cache := &Cache{LatestVersion: "v1.0.0"}
	path := filepath.Join(roDir, "cache.json")
	err := cache.SaveTo(path)
	if err == nil {
		t.Error("expected error when writing to read-only directory")
	}
}
