package dun

import (
	"strings"
	"testing"
)

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

// Gap-1: Negative test for Helix plugin not activating without docs/helix/
func TestHelixPluginInactiveWithoutDocsHelix(t *testing.T) {
	root := tempGitRepo(t)
	// No docs/helix/ directory - Helix plugin should not activate
	plan, err := PlanRepo(root)
	if err != nil {
		t.Fatalf("plan repo: %v", err)
	}
	for _, check := range plan.Checks {
		if strings.HasPrefix(check.ID, "helix-") {
			t.Fatalf("helix check %s should not be active without docs/helix/", check.ID)
		}
	}
}
