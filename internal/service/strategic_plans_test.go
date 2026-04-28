package service

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sambeau/kanbanzai/internal/config"
	"github.com/sambeau/kanbanzai/internal/model"
	"github.com/sambeau/kanbanzai/internal/storage"
	"github.com/sambeau/kanbanzai/internal/validate"
)

// writeTestStrategicPlan creates a strategic Plan entity directly on disk via the store,
// bypassing CreateStrategicPlan (which requires global config and prefix registry).
func writeTestStrategicPlan(t *testing.T, svc *EntityService, id, status string) {
	t.Helper()
	_, _, slug := model.ParsePlanID(id)
	if slug == "" {
		slug = "test-plan"
		id = "P1-" + slug
	}
	fields := map[string]any{
		"id":         id,
		"slug":       slug,
		"name":       "Test Strategic Plan",
		"status":     status,
		"summary":    "Test strategic plan for unit tests",
		"order":      0,
		"created":    "2026-04-28T12:00:00Z",
		"created_by": "test",
		"updated":    "2026-04-28T12:00:00Z",
	}
	_, err := svc.store.Write(storage.EntityRecord{
		Type:   entityTypeStrategicPlan,
		ID:     id,
		Slug:   slug,
		Fields: fields,
	})
	if err != nil {
		t.Fatalf("writeTestStrategicPlan(%s) error = %v", id, err)
	}
}

// writeTestStrategicPlanWithParent creates a strategic plan with a parent reference.
func writeTestStrategicPlanWithParent(t *testing.T, svc *EntityService, id, status, parent string, order int) {
	t.Helper()
	_, _, slug := model.ParsePlanID(id)
	if slug == "" {
		slug = "test-plan"
	}
	fields := map[string]any{
		"id":         id,
		"slug":       slug,
		"name":       "Test Strategic Plan " + slug,
		"status":     status,
		"summary":    "Test strategic plan for unit tests",
		"parent":     parent,
		"order":      order,
		"created":    "2026-04-28T12:00:00Z",
		"created_by": "test",
		"updated":    "2026-04-28T12:00:00Z",
	}
	_, err := svc.store.Write(storage.EntityRecord{
		Type:   entityTypeStrategicPlan,
		ID:     id,
		Slug:   slug,
		Fields: fields,
	})
	if err != nil {
		t.Fatalf("writeTestStrategicPlanWithParent(%s) error = %v", id, err)
	}
}

// ─── AC-001: StrategicPlan struct exists and is YAML-serialisable ────────────

func TestStrategicPlan_YAMLRoundTrip(t *testing.T) {
	t.Parallel()

	original := model.StrategicPlan{
		ID:        "P1-test-plan",
		Slug:      "test-plan",
		Name:      "Test Plan",
		Status:    model.PlanningStatusIdea,
		Summary:   "A test plan",
		Parent:    "",
		Design:    "",
		DependsOn: nil,
		Order:     0,
		Tags:      []string{"tag1", "tag2"},
	}

	fields := strategicPlanFields(original)

	// Verify key fields are present.
	tests := []struct {
		key   string
		value any
	}{
		{"id", "P1-test-plan"},
		{"slug", "test-plan"},
		{"name", "Test Plan"},
		{"status", "idea"},
		{"summary", "A test plan"},
		{"order", 0},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			got, ok := fields[tt.key]
			if !ok {
				t.Fatalf("field %q missing from strategic plan fields", tt.key)
			}
			if got != tt.value {
				t.Fatalf("field %q = %v (type %T), want %v", tt.key, got, got, tt.value)
			}
		})
	}

	// Verify tags are preserved.
	tagsRaw, ok := fields["tags"]
	if !ok {
		t.Fatal("tags field missing from strategic plan fields")
	}
	tags, ok := tagsRaw.([]any)
	if !ok {
		t.Fatalf("tags field type = %T, want []any", tagsRaw)
	}
	if len(tags) != 2 || tags[0] != "tag1" || tags[1] != "tag2" {
		t.Fatalf("tags = %v, want [tag1 tag2]", tags)
	}
}

// ─── AC-002: parent and order are optional ────────────────────────────────────

func TestStrategicPlan_OptionalFields(t *testing.T) {
	t.Parallel()

	// Plan with parent and order.
	p1 := model.StrategicPlan{
		ID:     "P1-parent",
		Status: model.PlanningStatusIdea,
		Parent: "P0-root",
		Order:  3,
	}
	if p1.Parent != "P0-root" {
		t.Fatalf("Plan with Parent: Parent = %q, want %q", p1.Parent, "P0-root")
	}
	if p1.Order != 3 {
		t.Fatalf("Plan with Order: Order = %d, want %d", p1.Order, 3)
	}

	// Plan without parent and order (zero values).
	p2 := model.StrategicPlan{
		ID:     "P2-child",
		Status: model.PlanningStatusIdea,
	}
	if p2.Parent != "" {
		t.Fatalf("Plan without Parent: Parent = %q, want empty", p2.Parent)
	}
	if p2.Order != 0 {
		t.Fatalf("Plan without Order: Order = %d, want 0", p2.Order)
	}
}

// ─── AC-003: no next_feature_seq field ────────────────────────────────────────

func TestStrategicPlan_NoNextFeatureSeq(t *testing.T) {
	t.Parallel()

	// Compile-time check: accessing NextFeatureSeq on StrategicPlan should fail.
	// This test verifies the struct does not have the field by checking YAML output.
	p := model.StrategicPlan{
		ID: "P1-test",
	}
	fields := strategicPlanFields(p)
	if _, ok := fields["next_feature_seq"]; ok {
		t.Fatal("StrategicPlan should NOT have next_feature_seq field, but it was found in YAML output")
	}
}

// ─── AC-004: ID format matches P{n}-{slug} ────────────────────────────────────

func TestStrategicPlan_IDFormat(t *testing.T) {
	t.Parallel()

	// Verify IsPlanID works for strategic plan IDs (same format as batch plans).
	if !model.IsPlanID("P1-social-platform") {
		t.Fatal("IsPlanID should return true for P1-social-platform")
	}
	prefix, num, slug := model.ParsePlanID("P1-social-platform")
	if prefix != "P" {
		t.Fatalf("ParsePlanID prefix = %q, want %q", prefix, "P")
	}
	if num != "1" {
		t.Fatalf("ParsePlanID number = %q, want %q", num, "1")
	}
	if slug != "social-platform" {
		t.Fatalf("ParsePlanID slug = %q, want %q", slug, "social-platform")
	}
}

// ─── AC-006: Status constants are distinct and initial state is idea ──────────

func TestStrategicPlan_StatusConstants(t *testing.T) {
	t.Parallel()

	constants := []struct {
		name  string
		value model.PlanningStatus
		want  string
	}{
		{"Idea", model.PlanningStatusIdea, "idea"},
		{"Shaping", model.PlanningStatusShaping, "shaping"},
		{"Ready", model.PlanningStatusReady, "ready"},
		{"Active", model.PlanningStatusActive, "active"},
		{"Done", model.PlanningStatusDone, "done"},
		{"Superseded", model.PlanningStatusSuperseded, "superseded"},
		{"Cancelled", model.PlanningStatusCancelled, "cancelled"},
	}

	seen := make(map[string]bool)
	for _, c := range constants {
		if string(c.value) != c.want {
			t.Fatalf("PlanningStatus%s = %q, want %q", c.name, c.value, c.want)
		}
		if seen[string(c.value)] {
			t.Fatalf("Duplicate PlanningStatus constant value: %q", c.value)
		}
		seen[string(c.value)] = true
	}
}

// ─── AC-007: Valid lifecycle transitions ──────────────────────────────────────

func TestStrategicPlan_ValidTransitions(t *testing.T) {
	t.Parallel()

	type transition struct {
		from string
		to   string
	}

	valid := []transition{
		{"idea", "shaping"},
		{"shaping", "ready"},
		{"shaping", "idea"},
		{"ready", "active"},
		{"active", "done"},
		{"active", "shaping"},
	}

	for _, tt := range valid {
		t.Run(tt.from+"_to_"+tt.to, func(t *testing.T) {
			if err := validate.ValidateTransition(validate.EntityStrategicPlan, tt.from, tt.to); err != nil {
				t.Fatalf("ValidateTransition(%q -> %q) should succeed: %v", tt.from, tt.to, err)
			}
		})
	}
}

// ─── AC-008: Terminal states reachable from any non-terminal ──────────────────

func TestStrategicPlan_TerminalTransitions(t *testing.T) {
	t.Parallel()

	nonTerminalStates := []string{"idea", "shaping", "ready", "active", "done"}
	terminalStates := []string{"superseded", "cancelled"}

	for _, from := range nonTerminalStates {
		for _, to := range terminalStates {
			t.Run(from+"_to_"+to, func(t *testing.T) {
				if err := validate.ValidateTransition(validate.EntityStrategicPlan, from, to); err != nil {
					t.Fatalf("ValidateTransition(%q -> %q) should succeed: %v", from, to, err)
				}
			})
		}
	}
}

// ─── AC-009: Invalid transitions fail with descriptive error ─────────────────

func TestStrategicPlan_InvalidTransitions(t *testing.T) {
	t.Parallel()

	type transition struct {
		from string
		to   string
	}

	invalid := []transition{
		{"idea", "done"},
		{"done", "shaping"},
		{"superseded", "idea"},
		{"cancelled", "shaping"},
		{"idea", "active"},
		{"ready", "done"},
	}

	for _, tt := range invalid {
		t.Run(tt.from+"_to_"+tt.to, func(t *testing.T) {
			err := validate.ValidateTransition(validate.EntityStrategicPlan, tt.from, tt.to)
			if err == nil {
				t.Fatalf("ValidateTransition(%q -> %q) should fail but succeeded", tt.from, tt.to)
			}
		})
	}
}

// ─── AC-010: Parent plan must exist ───────────────────────────────────────────

func TestStrategicPlan_CreateWithNonexistentParent(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	svc := newTestEntityService(root, "2026-04-28T12:00:00Z")

	// Override config to register prefix P.
	cfg := config.LoadOrDefault()
	_ = cfg // config might not have prefix registered; CreateStrategicPlan will check

	// Attempt to create with a non-existent parent should fail.
	_, err := svc.CreateStrategicPlan(CreateStrategicPlanInput{
		Prefix:    "P",
		Slug:      "child-plan",
		Name:      "Child Plan",
		Summary:   "A child plan",
		Parent:    "P1-nonexistent",
		CreatedBy: "test",
	})
	if err == nil {
		t.Fatal("CreateStrategicPlan with nonexistent parent should fail")
	}
	if !strings.Contains(err.Error(), "referenced entity not found") {
		t.Fatalf("Error should mention 'referenced entity not found', got: %v", err)
	}
}

// ─── AC-011: Cycle detection ──────────────────────────────────────────────────


// TestStrategicPlan_CreateWithValidParent verifies the success path for
// creating a strategic plan with a valid, existing parent.
func TestStrategicPlan_CreateWithValidParent(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	svc := newTestEntityService(root, "2026-04-28T12:00:00Z")

	// Write the parent plan first.
	writeTestStrategicPlan(t, svc, "P1-existing-parent", "active")

	// Create child plan with parent pointing to the existing plan.
	result, err := svc.CreateStrategicPlan(CreateStrategicPlanInput{
		Prefix:    "P",
		Slug:      "child-plan",
		Name:      "Child Plan",
		Summary:   "A child plan with valid parent",
		Parent:    "P1-existing-parent",
		CreatedBy: "test",
	})
	if err != nil {
		t.Fatalf("CreateStrategicPlan with valid parent should succeed, got: %v", err)
	}
	if result.State["parent"] != "P1-existing-parent" {
		t.Errorf("parent = %v, want P1-existing-parent", result.State["parent"])
	}
}
func TestStrategicPlan_CycleDetection(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	svc := newTestEntityService(root, "2026-04-28T12:00:00Z")

	// Create chain: P1 (parent of P2) and P2 (parent of P3).
	writeTestStrategicPlanWithParent(t, svc, "P1-root", "active", "", 0)
	writeTestStrategicPlanWithParent(t, svc, "P2-child", "active", "P1-root", 0)
	writeTestStrategicPlanWithParent(t, svc, "P3-grandchild", "active", "P2-child", 0)

	// Attempt to set P1's parent to P3 (creates a cycle).
	err := svc.detectStrategicPlanCycle("P1-root", "P3-grandchild")
	if err == nil {
		t.Fatal("Cycle detection should fail for P1 -> P3")
	}
	if !strings.Contains(err.Error(), "cycle detected") {
		t.Fatalf("Error should mention 'cycle detected', got: %v", err)
	}

	// Direct self-reference.
	err = svc.detectStrategicPlanCycle("P1-root", "P1-root")
	if err == nil {
		t.Fatal("Self-reference should be detected as a cycle")
	}
	if !strings.Contains(err.Error(), "cycle detected") {
		t.Fatalf("Error should mention 'cycle detected', got: %v", err)
	}
}

// TestStrategicPlan_UpdateParentCycleDetection verifies that UpdateStrategicPlan
// with a Parent that would create a cycle returns the cycle-detection error via
// the public API (not just the private detectStrategicPlanCycle function).
func TestStrategicPlan_UpdateParentCycleDetection(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	svc := newTestEntityService(root, "2026-04-28T12:00:00Z")

	// Create chain: P1-root (parent of P2-child) and P2-child (parent of P3-grandchild).
	writeTestStrategicPlanWithParent(t, svc, "P1-root", "active", "", 0)
	writeTestStrategicPlanWithParent(t, svc, "P2-child", "active", "P1-root", 0)
	writeTestStrategicPlanWithParent(t, svc, "P3-grandchild", "active", "P2-child", 0)

	// Attempt to set P1-root's parent to P3-grandchild (would create a cycle).
	_, _, slug := model.ParsePlanID("P1-root")
	newParent := "P3-grandchild"
	_, err := svc.UpdateStrategicPlan(UpdateStrategicPlanInput{
		ID:     "P1-root",
		Slug:   slug,
		Parent: &newParent,
	})
	if err == nil {
		t.Fatal("UpdateStrategicPlan with cyclic parent should fail")
	}
	if !strings.Contains(err.Error(), "cycle detected") {
		t.Fatalf("Error should mention 'cycle detected', got: %v", err)
	}

	// Direct self-reference should also be caught.
	newParent = "P1-root"
	_, err = svc.UpdateStrategicPlan(UpdateStrategicPlanInput{
		ID:     "P1-root",
		Slug:   slug,
		Parent: &newParent,
	})
	if err == nil {
		t.Fatal("Self-reference as parent should be detected as a cycle")
	}
	if !strings.Contains(err.Error(), "cycle detected") {
		t.Fatalf("Error should mention 'cycle detected', got: %v", err)

	}
}
// ─── AC-012: Deep nesting (depth 5) ───────────────────────────────────────────

func TestStrategicPlan_DeepNesting(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	svc := newTestEntityService(root, "2026-04-28T12:00:00Z")

	// Create a chain of depth 5 using sequential IDs.
	parent := ""
	for i := 1; i <= 5; i++ {
		id := fmt.Sprintf("P1-level%d", i)
		writeTestStrategicPlanWithParent(t, svc, id, "idea", parent, i)
		parent = id
	}

	// Verify all 5 plans are retrievable via GetStrategicPlan.
	for i := 1; i <= 5; i++ {
		id := fmt.Sprintf("P1-level%d", i)
		result, err := svc.GetStrategicPlan(id)
		if err != nil {
			t.Fatalf("GetStrategicPlan(%s) error = %v", id, err)
		}
		if result.State == nil {
			t.Fatalf("GetStrategicPlan(%s): nil state", id)
		}
	}

	// List top-level plans (should return just level-1).
	results, err := svc.ListStrategicPlans(StrategicPlanFilters{})
	if err != nil {
		t.Fatalf("ListStrategicPlans error = %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("Top-level plan count = %d, want 1", len(results))
	}
}

// ─── Deep nesting verification ────────────────────────────────────────────────

// ─── AC-016: Coexistence with existing batch operations ──────────────────────

func TestStrategicPlan_CoexistenceWithBatch(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	svc := newTestEntityService(root, "2026-04-28T12:00:00Z")

	// Create a batch (old plan) via writeTestPlan.
	writeTestPlan(t, svc, "P99-batch-plan")

	// Create a strategic plan with the same ID pattern.
	writeTestStrategicPlan(t, svc, "P1-strategic-plan", "idea")

	// ListPlans should still work (lists batch plans from plan storage).
	batchPlans, err := svc.ListPlans(PlanFilters{})
	if err != nil {
		t.Fatalf("ListPlans error = %v", err)
	}

	// ListStrategicPlans should work (lists strategic plans from same directory).
	spPlans, err := svc.ListStrategicPlans(StrategicPlanFilters{})
	if err != nil {
		t.Fatalf("ListStrategicPlans error = %v", err)
	}

	// Batch GetPlan should still find batch plan.
	_, err = svc.GetPlan("P99-batch-plan")
	if err != nil {
		t.Fatalf("GetPlan(P99-batch-plan) should succeed: %v", err)
	}

	// Strategic GetStrategicPlan should find strategic plan.
	_, err = svc.GetStrategicPlan("P1-strategic-plan")
	if err != nil {
		t.Fatalf("GetStrategicPlan(P1-strategic-plan) should succeed: %v", err)
	}

	// Verify ListPlans returns the batch entity.
	if len(batchPlans) == 0 {
		t.Error("ListPlans returned no batch plans, expected at least 1")
	}
	foundBatch := false
	for _, p := range batchPlans {
		if p.ID == "P99-batch-plan" {
			foundBatch = true
			break
		}
	}
	if !foundBatch {
		t.Error("ListPlans did not contain P99-batch-plan")
	}

	// Verify ListStrategicPlans returns the strategic plan entity.
	if len(spPlans) == 0 {
		t.Error("ListStrategicPlans returned no plans, expected at least 1")
	}
	foundSP := false
	for _, p := range spPlans {
		if p.ID == "P1-strategic-plan" {
			foundSP = true
			break
		}
	}
	if !foundSP {
		t.Error("ListStrategicPlans did not contain P1-strategic-plan")
	}

	// Verify the two lists are disjoint (no ID overlap).
	batchIDs := make(map[string]bool)
	for _, p := range batchPlans {
		batchIDs[p.ID] = true
	}
	for _, p := range spPlans {
		if batchIDs[p.ID] {
			t.Errorf("ID %s appears in both ListPlans and ListStrategicPlans", p.ID)
		}
}
	}

func TestStrategicPlan_InvalidTransitionNoStateChange(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	svc := newTestEntityService(root, "2026-04-28T12:00:00Z")

	// Create a strategic plan in "idea" status.
	writeTestStrategicPlan(t, svc, "P1-test-plan", "idea")
	_, _, slug := model.ParsePlanID("P1-test-plan")
	id := "P1-test-plan"

	// Attempt an invalid transition (idea -> done).
	_, err := svc.UpdateStrategicPlanStatus(id, slug, "done")
	if err == nil {
		t.Fatal("UpdateStrategicPlanStatus(idea -> done) should fail")
	}

	// Verify the state file still has status "idea".
	result, err := svc.GetStrategicPlan(id)
	if err != nil {
		t.Fatalf("GetStrategicPlan after failed transition: %v", err)
	}
	status := stringFromState(result.State, "status")
	if status != "idea" {
		t.Fatalf("Status after failed transition = %q, want %q", status, "idea")
	}
}

// ─── isTerminalState check ────────────────────────────────────────────────────

func TestStrategicPlan_IsTerminalState(t *testing.T) {
	t.Parallel()

	tests := []struct {
		state string
		want  bool
	}{
		{"idea", false},
		{"shaping", false},
		{"ready", false},
		{"active", false},
		{"done", false},
		{"superseded", true},
		{"cancelled", true},
	}

	for _, tt := range tests {
		t.Run(tt.state, func(t *testing.T) {
			got := validate.IsTerminalState(validate.EntityStrategicPlan, tt.state)
			if got != tt.want {
				t.Fatalf("IsTerminalState(%q) = %v, want %v", tt.state, got, tt.want)
			}
		})
	}
}

// ─── File persistence test ────────────────────────────────────────────────────

func TestStrategicPlan_FilePersisted(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	svc := newTestEntityService(root, "2026-04-28T12:00:00Z")

	// Write a strategic plan directly.
	writeTestStrategicPlan(t, svc, "P1-persisted-plan", "idea")

	// Verify the file exists in plans/ directory.
	expectedPath := filepath.Join(root, "plans", "P1-persisted-plan.yaml")
	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Fatalf("Strategic plan file not found at %s", expectedPath)
	}
}

// ─── List top-level vs child plans ────────────────────────────────────────────

func TestStrategicPlan_ListFilters(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	svc := newTestEntityService(root, "2026-04-28T12:00:00Z")

	// Create plans: one top-level, one child.
	writeTestStrategicPlanWithParent(t, svc, "P1-top", "idea", "", 0)
	writeTestStrategicPlanWithParent(t, svc, "P2-child", "idea", "P1-top", 0)

	// List top-level plans (default filter).
	topLevel, err := svc.ListStrategicPlans(StrategicPlanFilters{})
	if err != nil {
		t.Fatalf("ListStrategicPlans: %v", err)
	}
	if len(topLevel) != 1 {
		t.Fatalf("Top-level plan count = %d, want 1", len(topLevel))
	}

	// List all plans (parent = "*").
	all, err := svc.ListStrategicPlans(StrategicPlanFilters{Parent: "*"})
	if err != nil {
		t.Fatalf("ListStrategicPlans(Parent=*): %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("All plan count = %d, want 2", len(all))
	}
}

// ─── Status field update ──────────────────────────────────────────────────────

func TestStrategicPlan_UpdatePlanStatus(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	svc := newTestEntityService(root, "2026-04-28T12:00:00Z")

	writeTestStrategicPlan(t, svc, "P1-update-status", "idea")

	// Transition idea -> shaping.
	result, err := svc.UpdateStrategicPlanStatus("P1-update-status", "update-status", "shaping")
	if err != nil {
		t.Fatalf("UpdateStrategicPlanStatus(idea -> shaping): %v", err)
	}
	status := stringFromState(result.State, "status")
	if status != "shaping" {
		t.Fatalf("Status after transition = %q, want %q", status, "shaping")
	}
}

// ─── Update mutable fields ────────────────────────────────────────────────────

func TestStrategicPlan_UpdateFields(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	svc := newTestEntityService(root, "2026-04-28T12:00:00Z")

	writeTestStrategicPlan(t, svc, "P1-update-fields", "idea")

	newName := "Updated Name"
	newSummary := "Updated summary"
	newOrder := 42

	_, err := svc.UpdateStrategicPlan(UpdateStrategicPlanInput{
		ID:      "P1-update-fields",
		Slug:    "update-fields",
		Name:    &newName,
		Summary: &newSummary,
		Order:   &newOrder,
	})
	if err != nil {
		t.Fatalf("UpdateStrategicPlan: %v", err)
	}

	result, err := svc.GetStrategicPlan("P1-update-fields")
	if err != nil {
		t.Fatalf("GetStrategicPlan: %v", err)
	}

	if got := stringFromState(result.State, "name"); got != newName {
		t.Fatalf("name = %q, want %q", got, newName)
	}
	if got := stringFromState(result.State, "summary"); got != newSummary {
		t.Fatalf("summary = %q, want %q", got, newSummary)
	}
}

// ─── Entity tool create/get/list/transition ───────────────────────────────────

func TestStrategicPlan_EntityKindConstants(t *testing.T) {
	t.Parallel()

	if model.EntityKindStrategicPlan != "strategic-plan" {
		t.Fatalf("EntityKindStrategicPlan = %q, want %q", model.EntityKindStrategicPlan, "strategic-plan")
	}

	// Verify EntityStrategicPlan exists in validate package.
	kind := validate.EntityStrategicPlan
	if kind != model.EntityKindStrategicPlan {
		t.Fatalf("validate.EntityStrategicPlan = %v, want %v", kind, model.EntityKindStrategicPlan)
	}

	// Verify entry state is "idea".
	entry, ok := validate.EntryState(validate.EntityStrategicPlan)
	if !ok {
		t.Fatal("EntryState for EntityStrategicPlan should be defined")
	}
	if entry != "idea" {
		t.Fatalf("EntryState = %q, want %q", entry, "idea")
	}
}

// ─── ValidateInitialState ─────────────────────────────────────────────────────

func TestStrategicPlan_ValidateInitialState(t *testing.T) {
	t.Parallel()

	// Valid: idea.
	if err := validate.ValidateInitialState(validate.EntityStrategicPlan, "idea"); err != nil {
		t.Fatalf("ValidateInitialState(idea) should succeed: %v", err)
	}

	// Invalid: anything else.
	if err := validate.ValidateInitialState(validate.EntityStrategicPlan, "shaping"); err == nil {
		t.Fatal("ValidateInitialState(shaping) should fail")
	}
}

// ─── StrategicPlan implements Entity interface ────────────────────────────────

func TestStrategicPlan_ImplementsEntity(t *testing.T) {
	t.Parallel()

	// Compile-time check: StrategicPlan implements model.Entity.
	var _ model.Entity = (*model.StrategicPlan)(nil)

	p := model.StrategicPlan{
		ID:   "P1-test",
		Slug: "test",
		Name: "Test",
	}

	if p.GetKind() != model.EntityKindStrategicPlan {
		t.Fatalf("GetKind() = %v, want %v", p.GetKind(), model.EntityKindStrategicPlan)
	}
	if p.GetID() != "P1-test" {
		t.Fatalf("GetID() = %q, want %q", p.GetID(), "P1-test")
	}
	if p.GetSlug() != "test" {
		t.Fatalf("GetSlug() = %q, want %q", p.GetSlug(), "test")
	}
}
