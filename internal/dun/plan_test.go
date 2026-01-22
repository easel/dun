package dun

import "testing"

func TestPlanRepoIncludesHelixChecks(t *testing.T) {
	plan, err := PlanRepo(fixturePath(t, "../testdata/repos/helix-missing-architecture"))
	if err != nil {
		t.Fatalf("plan repo: %v", err)
	}

	expectIDs := []string{
		"helix-gates",
		"helix-state-rules",
		"helix-create-architecture",
	}
	for _, id := range expectIDs {
		if !planHas(plan, id) {
			t.Fatalf("expected check %s", id)
		}
	}
}

func planHas(plan Plan, id string) bool {
	for _, check := range plan.Checks {
		if check.ID == id {
			return true
		}
	}
	return false
}
