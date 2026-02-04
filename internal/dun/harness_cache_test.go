package dun

import (
	"reflect"
	"testing"
	"time"
)

func TestHarnessCacheSaveLoad(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	cache := HarnessCache{
		LastCheck: time.Now().UTC(),
		Harnesses: []HarnessStatus{
			{Name: "codex", Command: "codex", Available: true, Live: true},
			{Name: "gemini", Command: "gemini", Available: false, Live: false},
		},
	}
	if err := cache.Save(); err != nil {
		t.Fatalf("save cache: %v", err)
	}

	loaded, err := LoadHarnessCache()
	if err != nil {
		t.Fatalf("load cache: %v", err)
	}
	if len(loaded.Harnesses) != 2 {
		t.Fatalf("expected 2 harnesses, got %d", len(loaded.Harnesses))
	}
	got := loaded.AvailableHarnesses()
	want := []string{"codex"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("available harnesses mismatch: got %v want %v", got, want)
	}
}
