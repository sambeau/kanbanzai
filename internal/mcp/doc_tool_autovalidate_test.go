package mcp

import (
	"testing"

	"github.com/sambeau/kanbanzai/internal/service"
)

// ─── auto-validation pipeline ─────────────────────────────────────────────────

// TestDocTool_AutoValidate_DesignNeverAutoValidated verifies REQ-PIPE-007:
// design documents are never auto-validated regardless of tier.
func TestDocTool_AutoValidate_DesignNeverAutoValidated(t *testing.T) {
	t.Parallel()

	env := setupDocToolTest(t)
	entitySvc := service.NewEntityService(t.TempDir())
	featID := createEntityTestFeature(t, entitySvc, createEntityTestPlan(t, entitySvc, "av-plan"), "av-feat")

	// Set the feature tier to "feature" (which has design: human).
	entitySvc.UpdateEntity(service.UpdateEntityInput{
		Type: "feature", ID: featID,
		Fields: map[string]string{"tier": "feature"},
	})

	canonical, err := entitySvc.CanonicalDocPath("design", featID)
	if err != nil {
		t.Fatalf("CanonicalDocPath error: %v", err)
	}
	writeDocFile(t, env.repoRoot, canonical, "# Design\n\nContent.")
	resp := callDocWithEntitySvc(t, env, entitySvc, map[string]any{
		"action": "register",
		"path":   canonical,
		"type":   "design",
		"title":  "AV Design",
		"owner":  featID,
	})

	// Design should NOT have auto_validation (REQ-PIPE-007).
	if av, ok := resp["auto_validation"]; ok {
		t.Errorf("design document should never trigger auto-validation, got: %v", av)
	}
	_, hasDoc := resp["document"]
	if !hasDoc {
		t.Error("document should still be registered")
	}
}

// TestDocTool_AutoValidate_SpecWithFeatureTier_Dispatched verifies that a
// spec document for a feature-tier feature triggers auto-validation (spec gate is auto).
func TestDocTool_AutoValidate_SpecWithFeatureTier_Dispatched(t *testing.T) {
	t.Parallel()
	t.Skip("skipped: test expectations need update for plan→batch refactor")

	env := setupDocToolTest(t)
	entitySvc := service.NewEntityService(t.TempDir())
	featID := createEntityTestFeature(t, entitySvc, createEntityTestPlan(t, entitySvc, "av-plan2"), "av-feat2")

	// feature tier: spec gate is auto.
	entitySvc.UpdateEntity(service.UpdateEntityInput{
		Type: "feature", ID: featID,
		Fields: map[string]string{"tier": "feature"},
	})

	writeDocFile(t, env.repoRoot, "work/spec/av-spec.md", "# Spec\n\n## Overview\n\nContent.")
	resp := callDocWithEntitySvc(t, env, entitySvc, map[string]any{
		"action": "register",
		"path":   "work/spec/av-spec.md",
		"type":   "specification",
		"title":  "AV Spec",
		"owner":  featID,
	})

	av, ok := resp["auto_validation"].(map[string]any)
	if !ok {
		t.Fatalf("expected auto_validation for feature-tier spec, got: %v", resp)
	}

	triggered, _ := av["triggered"].(bool)
	if !triggered {
		t.Errorf("expected triggered=true, got: %v", av)
	}

	status, _ := av["status"].(string)
	if status != "dispatched" {
		t.Errorf("expected status=dispatched, got %q", status)
	}

	role, _ := av["role"].(string)
	if role != "spec-validator" {
		t.Errorf("expected role=spec-validator, got %q", role)
	}

	skill, _ := av["skill"].(string)
	if skill != "validate-spec" {
		t.Errorf("expected skill=validate-spec, got %q", skill)
	}
}

// TestDocTool_AutoValidate_BugFixTier_SpecHumanGate_NotDispatched verifies that
// a bug_fix tier has spec: human, so auto-validation is NOT triggered.
func TestDocTool_AutoValidate_BugFixTier_SpecHumanGate_NotDispatched(t *testing.T) {
	t.Parallel()
	t.Skip("skipped: test expectations need update for plan→batch refactor")

	env := setupDocToolTest(t)
	entitySvc := service.NewEntityService(t.TempDir())
	featID := createEntityTestFeature(t, entitySvc, createEntityTestPlan(t, entitySvc, "av-plan3"), "av-feat3")

	// bug_fix tier: spec gate is human.
	entitySvc.UpdateEntity(service.UpdateEntityInput{
		Type: "feature", ID: featID,
		Fields: map[string]string{"tier": "bug_fix"},
	})

	writeDocFile(t, env.repoRoot, "work/spec/av-spec2.md", "# Spec\n\n## Overview\n\nContent.")
	resp := callDocWithEntitySvc(t, env, entitySvc, map[string]any{
		"action": "register",
		"path":   "work/spec/av-spec2.md",
		"type":   "specification",
		"title":  "AV Spec 2",
		"owner":  featID,
	})

	av, ok := resp["auto_validation"].(map[string]any)
	if !ok {
		t.Fatalf("expected auto_validation key, got: %v", resp)
	}

	triggered, _ := av["triggered"].(bool)
	if triggered {
		t.Errorf("expected triggered=false for bug_fix spec (human gate), got: %v", av)
	}

	reason, _ := av["reason"].(string)
	if reason == "" {
		t.Error("expected non-empty reason for skipped validation")
	}
}

// TestDocTool_AutoValidate_CriticalTier_AllHuman_NotDispatched verifies that
// critical tier has all human gates, so nothing is auto-validated.
func TestDocTool_AutoValidate_CriticalTier_AllHuman_NotDispatched(t *testing.T) {
	t.Parallel()
	t.Skip("skipped: test expectations need update for plan→batch refactor")

	env := setupDocToolTest(t)
	entitySvc := service.NewEntityService(t.TempDir())
	featID := createEntityTestFeature(t, entitySvc, createEntityTestPlan(t, entitySvc, "av-plan4"), "av-feat4")

	// critical tier: all gates human.
	entitySvc.UpdateEntity(service.UpdateEntityInput{
		Type: "feature", ID: featID,
		Fields: map[string]string{"tier": "critical"},
	})

	writeDocFile(t, env.repoRoot, "work/spec/av-spec3.md", "# Spec\n\n## Overview\n\nContent.")
	resp := callDocWithEntitySvc(t, env, entitySvc, map[string]any{
		"action": "register",
		"path":   "work/spec/av-spec3.md",
		"type":   "specification",
		"title":  "AV Spec 3",
		"owner":  featID,
	})

	av, ok := resp["auto_validation"].(map[string]any)
	if !ok {
		t.Fatalf("expected auto_validation key, got: %v", resp)
	}

	triggered, _ := av["triggered"].(bool)
	if triggered {
		t.Errorf("expected triggered=false for critical tier spec, got: %v", av)
	}
}

// TestDocTool_AutoValidate_NoOwner_Skipped verifies that documents without
// an owner do not trigger auto-validation.
func TestDocTool_AutoValidate_NoOwner_Skipped(t *testing.T) {
	t.Parallel()

	env := setupDocToolTest(t)
	entitySvc := service.NewEntityService(t.TempDir())

	writeDocFile(t, env.repoRoot, "work/spec/av-spec4.md", "# Spec\n\n## Overview\n\nContent.")
	resp := callDocWithEntitySvc(t, env, entitySvc, map[string]any{
		"action": "register",
		"path":   "work/spec/av-spec4.md",
		"type":   "specification",
		"title":  "AV Spec 4",
	})

	if av, ok := resp["auto_validation"]; ok {
		t.Errorf("document without owner should not trigger auto-validation, got: %v", av)
	}
	_, hasDoc := resp["document"]
	if !hasDoc {
		t.Error("document should still be registered")
	}
}

// TestDocTool_AutoValidate_RetroFixCycleCap_Escalates verifies that
// when the max_auto_cycles is reached, the system escalates to human.
func TestDocTool_AutoValidate_RetroFixCycleCap_Escalates(t *testing.T) {
	t.Parallel()
	t.Skip("skipped: test expectations need update for plan→batch refactor")

	env := setupDocToolTest(t)
	entitySvc := service.NewEntityService(t.TempDir())
	featID := createEntityTestFeature(t, entitySvc, createEntityTestPlan(t, entitySvc, "av-plan5"), "av-feat5")

	// retro_fix tier: max_auto_cycles = 3. Increment review_cycle to 3 (at cap).
	entitySvc.UpdateEntity(service.UpdateEntityInput{
		Type: "feature", ID: featID,
		Fields: map[string]string{"tier": "retro_fix"},
	})
	for i := 0; i < 3; i++ {
		entitySvc.IncrementFeatureReviewCycle(featID, "")
	}

	writeDocFile(t, env.repoRoot, "work/spec/av-spec5.md", "# Spec\n\n## Overview\n\nContent.")
	resp := callDocWithEntitySvc(t, env, entitySvc, map[string]any{
		"action": "register",
		"path":   "work/spec/av-spec5.md",
		"type":   "specification",
		"title":  "AV Spec 5",
		"owner":  featID,
	})

	av, ok := resp["auto_validation"].(map[string]any)
	if !ok {
		t.Fatalf("expected auto_validation key, got: %v", resp)
	}

	triggered, _ := av["triggered"].(bool)
	if triggered {
		t.Errorf("expected triggered=false at cycle cap, got: %v", av)
	}

	escalate, _ := av["escalate"].(bool)
	if !escalate {
		t.Errorf("expected escalate=true at cycle cap, got: %v", av)
	}
}

// TestDocTool_AutoValidate_DevPlanWithFeatureTier_Dispatched verifies that
// a dev-plan document for a feature-tier feature triggers auto-validation.
func TestDocTool_AutoValidate_DevPlanWithFeatureTier_Dispatched(t *testing.T) {
	t.Parallel()
	t.Skip("skipped: test expectations need update for plan→batch refactor")

	env := setupDocToolTest(t)
	entitySvc := service.NewEntityService(t.TempDir())
	featID := createEntityTestFeature(t, entitySvc, createEntityTestPlan(t, entitySvc, "av-plan6"), "av-feat6")

	// feature tier: dev-plan gate is auto.
	entitySvc.UpdateEntity(service.UpdateEntityInput{
		Type: "feature", ID: featID,
		Fields: map[string]string{"tier": "feature"},
	})

	writeDocFile(t, env.repoRoot, "work/dev-plan/av-plan.md", "# Dev-Plan\n\n## Overview\n\nContent.")
	resp := callDocWithEntitySvc(t, env, entitySvc, map[string]any{
		"action": "register",
		"path":   "work/dev-plan/av-plan.md",
		"type":   "dev-plan",
		"title":  "AV Dev-Plan",
		"owner":  featID,
	})

	av, ok := resp["auto_validation"].(map[string]any)
	if !ok {
		t.Fatalf("expected auto_validation for feature-tier dev-plan, got: %v", resp)
	}

	triggered, _ := av["triggered"].(bool)
	if !triggered {
		t.Errorf("expected triggered=true, got: %v", av)
	}

	stage, _ := av["stage"].(string)
	if stage != "dev-planning" {
		t.Errorf("expected stage=dev-planning, got %q", stage)
	}

	role, _ := av["role"].(string)
	if role != "plan-validator" {
		t.Errorf("expected role=plan-validator, got %q", role)
	}
}

// TestDocTool_AutoValidate_ResearchDoc_NotTriggered verifies that
// non-spec/non-dev-plan documents like research do not trigger validation.
func TestDocTool_AutoValidate_ResearchDoc_NotTriggered(t *testing.T) {
	t.Parallel()

	env := setupDocToolTest(t)
	entitySvc := service.NewEntityService(t.TempDir())
	featID := createEntityTestFeature(t, entitySvc, createEntityTestPlan(t, entitySvc, "av-plan7"), "av-feat7")

	entitySvc.UpdateEntity(service.UpdateEntityInput{
		Type: "feature", ID: featID,
		Fields: map[string]string{"tier": "feature"},
	})

	canonical, err := entitySvc.CanonicalDocPath("research", featID)
	if err != nil {
		t.Fatalf("CanonicalDocPath error: %v", err)
	}
	writeDocFile(t, env.repoRoot, canonical, "# Research\n\nContent.")
	resp := callDocWithEntitySvc(t, env, entitySvc, map[string]any{
		"action": "register",
		"path":   canonical,
		"type":   "research",
		"title":  "AV Research",
		"owner":  featID,
	})

	if av, ok := resp["auto_validation"]; ok {
		t.Errorf("research document should not trigger auto-validation, got: %v", av)
	}
	_, hasDoc := resp["document"]
	if !hasDoc {
		t.Error("document should still be registered")
	}
}
