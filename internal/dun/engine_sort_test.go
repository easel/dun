package dun

import (
	"testing"
)

func TestSortPlanByPluginPriority(t *testing.T) {
	// Test that plugins with lower priority come first
	// Also test default priority (0 -> 50)
	plan := []plannedCheck{
		{Plugin: Plugin{Manifest: Manifest{Priority: 100}}, Check: Check{ID: "a"}},
		{Plugin: Plugin{Manifest: Manifest{Priority: 10}}, Check: Check{ID: "b"}},
		{Plugin: Plugin{Manifest: Manifest{Priority: 0}}, Check: Check{ID: "c"}},  // default 50
		{Plugin: Plugin{Manifest: Manifest{Priority: 30}}, Check: Check{ID: "d"}},
	}

	sortPlan(plan)

	expected := []string{"b", "d", "c", "a"} // 10, 30, 50, 100
	for i, id := range expected {
		if plan[i].Check.ID != id {
			t.Errorf("position %d: expected %s, got %s", i, id, plan[i].Check.ID)
		}
	}
}

func TestSortPlanByCheckPriority(t *testing.T) {
	// Test that checks with lower priority come first within same plugin priority
	// Also test default priority (0 -> 50)
	plan := []plannedCheck{
		{Plugin: Plugin{Manifest: Manifest{Priority: 50}}, Check: Check{ID: "a", Priority: 100}},
		{Plugin: Plugin{Manifest: Manifest{Priority: 50}}, Check: Check{ID: "b", Priority: 10}},
		{Plugin: Plugin{Manifest: Manifest{Priority: 50}}, Check: Check{ID: "c", Priority: 0}},  // default 50
		{Plugin: Plugin{Manifest: Manifest{Priority: 50}}, Check: Check{ID: "d", Priority: 30}},
	}

	sortPlan(plan)

	expected := []string{"b", "d", "c", "a"} // 10, 30, 50, 100
	for i, id := range expected {
		if plan[i].Check.ID != id {
			t.Errorf("position %d: expected %s, got %s", i, id, plan[i].Check.ID)
		}
	}
}

func TestSortPlanByPhase(t *testing.T) {
	// Test phase ordering when priorities are equal
	// Phase order: frame=1, design=2, test=3, build=4, deploy=5, iterate=6
	plan := []plannedCheck{
		{Check: Check{ID: "a", Phase: "deploy"}},
		{Check: Check{ID: "b", Phase: "frame"}},
		{Check: Check{ID: "c", Phase: "build"}},
		{Check: Check{ID: "d", Phase: "test"}},
		{Check: Check{ID: "e", Phase: "iterate"}},
		{Check: Check{ID: "f", Phase: "design"}},
	}

	sortPlan(plan)

	expected := []string{"b", "f", "d", "c", "a", "e"} // frame, design, test, build, deploy, iterate
	for i, id := range expected {
		if plan[i].Check.ID != id {
			t.Errorf("position %d: expected %s, got %s", i, id, plan[i].Check.ID)
		}
	}
}

func TestSortPlanByID(t *testing.T) {
	// Test alphabetical ordering when all else is equal
	plan := []plannedCheck{
		{Check: Check{ID: "zebra", Phase: "frame"}},
		{Check: Check{ID: "apple", Phase: "frame"}},
		{Check: Check{ID: "mango", Phase: "frame"}},
		{Check: Check{ID: "banana", Phase: "frame"}},
	}

	sortPlan(plan)

	expected := []string{"apple", "banana", "mango", "zebra"}
	for i, id := range expected {
		if plan[i].Check.ID != id {
			t.Errorf("position %d: expected %s, got %s", i, id, plan[i].Check.ID)
		}
	}
}

func TestSortPlanCombined(t *testing.T) {
	// Test a mix of all criteria
	plan := []plannedCheck{
		// Plugin priority 10, check priority 20, phase frame
		{Plugin: Plugin{Manifest: Manifest{Priority: 10}}, Check: Check{ID: "p10-c20-frame", Priority: 20, Phase: "frame"}},
		// Plugin priority 10, check priority 20, phase design (same plugin & check priority, different phase)
		{Plugin: Plugin{Manifest: Manifest{Priority: 10}}, Check: Check{ID: "p10-c20-design", Priority: 20, Phase: "design"}},
		// Plugin priority 10, check priority 10 (lower check priority)
		{Plugin: Plugin{Manifest: Manifest{Priority: 10}}, Check: Check{ID: "p10-c10", Priority: 10, Phase: "frame"}},
		// Plugin priority 5 (lowest plugin priority, should be first)
		{Plugin: Plugin{Manifest: Manifest{Priority: 5}}, Check: Check{ID: "p5", Priority: 50, Phase: "iterate"}},
		// Plugin priority 0 (default 50), check priority 0 (default 50), same phase - alphabetical
		{Plugin: Plugin{Manifest: Manifest{Priority: 0}}, Check: Check{ID: "zebra", Priority: 0, Phase: "build"}},
		{Plugin: Plugin{Manifest: Manifest{Priority: 0}}, Check: Check{ID: "alpha", Priority: 0, Phase: "build"}},
	}

	sortPlan(plan)

	// Expected order:
	// 1. p5 (plugin priority 5)
	// 2. p10-c10 (plugin priority 10, check priority 10)
	// 3. p10-c20-frame (plugin priority 10, check priority 20, phase frame=1)
	// 4. p10-c20-design (plugin priority 10, check priority 20, phase design=2)
	// 5. alpha (plugin priority 50, check priority 50, phase build, alphabetically first)
	// 6. zebra (plugin priority 50, check priority 50, phase build, alphabetically second)
	expected := []string{"p5", "p10-c10", "p10-c20-frame", "p10-c20-design", "alpha", "zebra"}
	for i, id := range expected {
		if plan[i].Check.ID != id {
			t.Errorf("position %d: expected %s, got %s", i, id, plan[i].Check.ID)
		}
	}
}

func TestSortPlanEmptyPlan(t *testing.T) {
	// Edge case: empty plan should not panic
	var plan []plannedCheck
	sortPlan(plan) // Should not panic
	if len(plan) != 0 {
		t.Error("expected empty plan to remain empty")
	}
}

func TestSortPlanSingleElement(t *testing.T) {
	// Edge case: single element should work
	plan := []plannedCheck{
		{Check: Check{ID: "only"}},
	}
	sortPlan(plan)
	if plan[0].Check.ID != "only" {
		t.Error("expected single element to remain unchanged")
	}
}
