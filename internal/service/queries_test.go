package service

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/sambeau/kanbanzai/internal/model"
	"github.com/sambeau/kanbanzai/internal/storage"
)

func writeTestEntity(t *testing.T, root, entityType, id, slug string, fields map[string]any) {
	t.Helper()
	store := storage.NewEntityStore(root)
	record := storage.EntityRecord{
		Type:   entityType,
		ID:     id,
		Slug:   slug,
		Fields: fields,
	}
	if _, err := store.Write(record); err != nil {
		t.Fatalf("writeTestEntity(%s, %s): %v", entityType, id, err)
	}
}

func makeFeatureFields(id, slug, parent, status string, tags []any) map[string]any {
	fields := map[string]any{
		"id":         id,
		"slug":       slug,
		"status":     status,
		"summary":    "test feature",
		"created":    "2026-01-01T00:00:00Z",
		"created_by": "tester",
	}
	if parent != "" {
		fields["parent"] = parent
	}
	if len(tags) > 0 {
		fields["tags"] = tags
	}
	return fields
}

func makeTaskFields(id, slug, parentFeature, status string, tags []any) map[string]any {
	fields := map[string]any{
		"id":             id,
		"slug":           slug,
		"parent_feature": parentFeature,
		"summary":        "test task",
		"status":         status,
	}
	if len(tags) > 0 {
		fields["tags"] = tags
	}
	return fields
}

func makePlanFields(id, slug, status string, tags []any) map[string]any {
	fields := map[string]any{
		"id":         id,
		"slug":       slug,
		"title":      "Test Plan",
		"status":     status,
		"summary":    "test plan",
		"created":    "2026-01-01T00:00:00Z",
		"created_by": "tester",
		"updated":    "2026-01-01T00:00:00Z",
	}
	if len(tags) > 0 {
		fields["tags"] = tags
	}
	return fields
}

func makeBugFields(id, slug, status string, tags []any) map[string]any {
	fields := map[string]any{
		"id":          id,
		"slug":        slug,
		"title":       "Test Bug",
		"status":      status,
		"severity":    "medium",
		"priority":    "medium",
		"type":        "implementation-defect",
		"reported_by": "tester",
		"reported":    "2026-01-01T00:00:00Z",
		"observed":    "something broke",
		"expected":    "it should work",
	}
	if len(tags) > 0 {
		fields["tags"] = tags
	}
	return fields
}

func TestListAllTags_Empty(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	svc := NewEntityService(root)

	tags, err := svc.ListAllTags()
	if err != nil {
		t.Fatalf("ListAllTags() error = %v", err)
	}
	if len(tags) != 0 {
		t.Fatalf("expected 0 tags, got %d", len(tags))
	}
}

func TestListAllTags_CollectsFromMultipleTypes(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	svc := NewEntityService(root)

	// Create a feature with tags
	writeTestEntity(t, root, "feature", "FEAT-01AAAAAAAAA01", "feat-one",
		makeFeatureFields("FEAT-01AAAAAAAAA01", "feat-one", "", "draft", []any{"backend", "auth"}))

	// Create a bug with overlapping and new tags
	writeTestEntity(t, root, "bug", "BUG-01BBBBBBBBB01", "bug-one",
		makeBugFields("BUG-01BBBBBBBBB01", "bug-one", "reported", []any{"auth", "critical"}))

	tags, err := svc.ListAllTags()
	if err != nil {
		t.Fatalf("ListAllTags() error = %v", err)
	}

	// Should have 3 unique tags: auth, backend, critical (sorted)
	if len(tags) != 3 {
		t.Fatalf("expected 3 tags, got %d: %v", len(tags), tags)
	}
	want := []string{"auth", "backend", "critical"}
	for i, tag := range want {
		if tags[i] != tag {
			t.Errorf("tags[%d] = %q, want %q", i, tags[i], tag)
		}
	}
}

func TestListEntitiesByTag_ReturnsMatchingEntities(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	svc := NewEntityService(root)

	writeTestEntity(t, root, "feature", "FEAT-01AAAAAAAAA02", "feat-tagged",
		makeFeatureFields("FEAT-01AAAAAAAAA02", "feat-tagged", "", "draft", []any{"api", "v2"}))

	writeTestEntity(t, root, "feature", "FEAT-01AAAAAAAAA03", "feat-other",
		makeFeatureFields("FEAT-01AAAAAAAAA03", "feat-other", "", "draft", []any{"frontend"}))

	writeTestEntity(t, root, "bug", "BUG-01BBBBBBBBB02", "bug-tagged",
		makeBugFields("BUG-01BBBBBBBBB02", "bug-tagged", "reported", []any{"api"}))

	results, err := svc.ListEntitiesByTag("api")
	if err != nil {
		t.Fatalf("ListEntitiesByTag() error = %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	// Both should have "api" tag — one feature, one bug
	types := map[string]int{}
	for _, r := range results {
		types[r.Type]++
	}
	if types["feature"] != 1 {
		t.Errorf("expected 1 feature, got %d", types["feature"])
	}
	if types["bug"] != 1 {
		t.Errorf("expected 1 bug, got %d", types["bug"])
	}
}

func TestListEntitiesByTag_EmptyTag(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	svc := NewEntityService(root)

	_, err := svc.ListEntitiesByTag("")
	if err == nil {
		t.Fatal("expected error for empty tag")
	}
}

func TestListEntitiesByTag_NoMatches(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	svc := NewEntityService(root)

	writeTestEntity(t, root, "feature", "FEAT-01AAAAAAAAA04", "feat-nm",
		makeFeatureFields("FEAT-01AAAAAAAAA04", "feat-nm", "", "draft", []any{"backend"}))

	results, err := svc.ListEntitiesByTag("nonexistent")
	if err != nil {
		t.Fatalf("ListEntitiesByTag() error = %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("expected 0 results, got %d", len(results))
	}
}

func TestListEntitiesFiltered_ByStatus(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	svc := NewEntityService(root)

	writeTestEntity(t, root, "feature", "FEAT-01AAAAAAAAA05", "feat-draft",
		makeFeatureFields("FEAT-01AAAAAAAAA05", "feat-draft", "", "draft", nil))

	writeTestEntity(t, root, "feature", "FEAT-01AAAAAAAAA06", "feat-done",
		makeFeatureFields("FEAT-01AAAAAAAAA06", "feat-done", "", "done", nil))

	results, err := svc.ListEntitiesFiltered(ListFilteredInput{
		Type:   "feature",
		Status: "draft",
	})
	if err != nil {
		t.Fatalf("ListEntitiesFiltered() error = %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].ID != "FEAT-01AAAAAAAAA05" {
		t.Errorf("unexpected ID: %s", results[0].ID)
	}
}

func TestListEntitiesFiltered_ByTags(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	svc := NewEntityService(root)

	writeTestEntity(t, root, "feature", "FEAT-01AAAAAAAAA07", "feat-a",
		makeFeatureFields("FEAT-01AAAAAAAAA07", "feat-a", "", "draft", []any{"api", "auth"}))

	writeTestEntity(t, root, "feature", "FEAT-01AAAAAAAAA08", "feat-b",
		makeFeatureFields("FEAT-01AAAAAAAAA08", "feat-b", "", "draft", []any{"frontend"}))

	// Filter with tag "api" — should match feat-a
	results, err := svc.ListEntitiesFiltered(ListFilteredInput{
		Type: "feature",
		Tags: []string{"api"},
	})
	if err != nil {
		t.Fatalf("ListEntitiesFiltered() error = %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].ID != "FEAT-01AAAAAAAAA07" {
		t.Errorf("unexpected ID: %s", results[0].ID)
	}
}

func TestListEntitiesFiltered_ByLabel(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	svc := NewEntityService(root)

	fields1 := makeFeatureFields("FEAT-01AAAAAAAAA13", "feat-labeled-g", "", "draft", nil)
	fields1["label"] = "G"
	writeTestEntity(t, root, "feature", "FEAT-01AAAAAAAAA13", "feat-labeled-g", fields1)

	fields2 := makeFeatureFields("FEAT-01AAAAAAAAA14", "feat-labeled-q", "", "draft", nil)
	fields2["label"] = "Q"
	writeTestEntity(t, root, "feature", "FEAT-01AAAAAAAAA14", "feat-labeled-q", fields2)

	writeTestEntity(t, root, "feature", "FEAT-01AAAAAAAAA15", "feat-unlabeled",
		makeFeatureFields("FEAT-01AAAAAAAAA15", "feat-unlabeled", "", "draft", nil))

	results, err := svc.ListEntitiesFiltered(ListFilteredInput{
		Type:  "feature",
		Label: "G",
	})
	if err != nil {
		t.Fatalf("ListEntitiesFiltered() error = %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].ID != "FEAT-01AAAAAAAAA13" {
		t.Errorf("unexpected ID: %s", results[0].ID)
	}
}

func TestListEntitiesFiltered_ByParent(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	svc := NewEntityService(root)

	writeTestEntity(t, root, "feature", "FEAT-01AAAAAAAAA09", "feat-pa",
		makeFeatureFields("FEAT-01AAAAAAAAA09", "feat-pa", "P1-my-plan", "draft", nil))

	writeTestEntity(t, root, "feature", "FEAT-01AAAAAAAAA10", "feat-pb",
		makeFeatureFields("FEAT-01AAAAAAAAA10", "feat-pb", "P2-other", "draft", nil))

	results, err := svc.ListEntitiesFiltered(ListFilteredInput{
		Type:   "feature",
		Parent: "P1-my-plan",
	})
	if err != nil {
		t.Fatalf("ListEntitiesFiltered() error = %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].ID != "FEAT-01AAAAAAAAA09" {
		t.Errorf("unexpected ID: %s", results[0].ID)
	}
}

func TestListEntitiesFiltered_ByDateRange(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	svc := NewEntityService(root)

	fieldsOld := makeFeatureFields("FEAT-01AAAAAAAAA11", "feat-old", "", "draft", nil)
	fieldsOld["created"] = "2025-01-01T00:00:00Z"
	writeTestEntity(t, root, "feature", "FEAT-01AAAAAAAAA11", "feat-old", fieldsOld)

	fieldsNew := makeFeatureFields("FEAT-01AAAAAAAAA12", "feat-new", "", "draft", nil)
	fieldsNew["created"] = "2026-06-01T00:00:00Z"
	writeTestEntity(t, root, "feature", "FEAT-01AAAAAAAAA12", "feat-new", fieldsNew)

	after := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	results, err := svc.ListEntitiesFiltered(ListFilteredInput{
		Type:         "feature",
		CreatedAfter: &after,
	})
	if err != nil {
		t.Fatalf("ListEntitiesFiltered() error = %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].ID != "FEAT-01AAAAAAAAA12" {
		t.Errorf("unexpected ID: %s", results[0].ID)
	}
}

func TestListEntitiesFiltered_ByParent_Task(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	svc := NewEntityService(root)

	writeTestEntity(t, root, "task", "TASK-01AAAAAAAAA20", "task-a",
		makeTaskFields("TASK-01AAAAAAAAA20", "task-a", "FEAT-01AAAAAAAAA09", "queued", nil))

	writeTestEntity(t, root, "task", "TASK-01AAAAAAAAA21", "task-b",
		makeTaskFields("TASK-01AAAAAAAAA21", "task-b", "FEAT-01AAAAAAAAA09", "queued", nil))

	writeTestEntity(t, root, "task", "TASK-01AAAAAAAAA22", "task-c",
		makeTaskFields("TASK-01AAAAAAAAA22", "task-c", "FEAT-01AAAAAAAAA10", "queued", nil))

	results, err := svc.ListEntitiesFiltered(ListFilteredInput{
		Type:   "task",
		Parent: "FEAT-01AAAAAAAAA09",
	})
	if err != nil {
		t.Fatalf("ListEntitiesFiltered() error = %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 tasks for FEAT-01AAAAAAAAA09, got %d", len(results))
	}

	ids := map[string]bool{}
	for _, r := range results {
		ids[r.ID] = true
	}
	if !ids["TASK-01AAAAAAAAA20"] || !ids["TASK-01AAAAAAAAA21"] {
		t.Errorf("unexpected task IDs: %v", ids)
	}
}

func TestListEntitiesFiltered_ByParent_TaskWithStatus(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	svc := NewEntityService(root)

	writeTestEntity(t, root, "task", "TASK-01AAAAAAAAA23", "task-d",
		makeTaskFields("TASK-01AAAAAAAAA23", "task-d", "FEAT-01AAAAAAAAA09", "queued", nil))

	writeTestEntity(t, root, "task", "TASK-01AAAAAAAAA24", "task-e",
		makeTaskFields("TASK-01AAAAAAAAA24", "task-e", "FEAT-01AAAAAAAAA09", "done", nil))

	results, err := svc.ListEntitiesFiltered(ListFilteredInput{
		Type:   "task",
		Parent: "FEAT-01AAAAAAAAA09",
		Status: "queued",
	})
	if err != nil {
		t.Fatalf("ListEntitiesFiltered() error = %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 queued task, got %d", len(results))
	}
	if results[0].ID != "TASK-01AAAAAAAAA23" {
		t.Errorf("unexpected ID: %s", results[0].ID)
	}
}

func TestListEntitiesFiltered_ByParent_Task_NoMatch(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	svc := NewEntityService(root)

	writeTestEntity(t, root, "task", "TASK-01AAAAAAAAA25", "task-x",
		makeTaskFields("TASK-01AAAAAAAAA25", "task-x", "FEAT-01AAAAAAAAA09", "queued", nil))

	writeTestEntity(t, root, "task", "TASK-01AAAAAAAAA26", "task-y",
		makeTaskFields("TASK-01AAAAAAAAA26", "task-y", "FEAT-01AAAAAAAAA09", "queued", nil))

	results, err := svc.ListEntitiesFiltered(ListFilteredInput{
		Type:   "task",
		Parent: "FEAT-01ZZZZZZZZZZZ",
	})
	if err != nil {
		t.Fatalf("ListEntitiesFiltered() error = %v", err)
	}

	if len(results) != 0 {
		t.Fatalf("expected 0 tasks for nonexistent feature, got %d", len(results))
	}
}

func TestListEntitiesFiltered_MissingType(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	svc := NewEntityService(root)

	_, err := svc.ListEntitiesFiltered(ListFilteredInput{})
	if err == nil {
		t.Fatal("expected error for missing type")
	}
}

func TestCrossEntityQuery_FindsTasksForPlan(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	svc := NewEntityService(root)

	planID := "P1-test-plan"

	// Create plan
	planDir := filepath.Join(root, "plans")
	if err := os.MkdirAll(planDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeTestEntity(t, root, string(model.EntityKindPlan), planID, "test-plan",
		makePlanFields(planID, "test-plan", "active", nil))

	// Create features under this plan
	writeTestEntity(t, root, "feature", "FEAT-01CCCCCCCCC01", "feat-x",
		makeFeatureFields("FEAT-01CCCCCCCCC01", "feat-x", planID, "developing", nil))

	writeTestEntity(t, root, "feature", "FEAT-01CCCCCCCCC02", "feat-y",
		makeFeatureFields("FEAT-01CCCCCCCCC02", "feat-y", "P2-other", "developing", nil))

	// Create tasks
	writeTestEntity(t, root, "task", "TASK-01DDDDDDDDD01", "task-a",
		makeTaskFields("TASK-01DDDDDDDDD01", "task-a", "FEAT-01CCCCCCCCC01", "queued", nil))

	writeTestEntity(t, root, "task", "TASK-01DDDDDDDDD02", "task-b",
		makeTaskFields("TASK-01DDDDDDDDD02", "task-b", "FEAT-01CCCCCCCCC01", "active", nil))

	// Task under the other feature — should NOT appear
	writeTestEntity(t, root, "task", "TASK-01DDDDDDDDD03", "task-c",
		makeTaskFields("TASK-01DDDDDDDDD03", "task-c", "FEAT-01CCCCCCCCC02", "queued", nil))

	results, err := svc.CrossEntityQuery(planID)
	if err != nil {
		t.Fatalf("CrossEntityQuery() error = %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(results))
	}

	ids := map[string]bool{}
	for _, r := range results {
		ids[r.ID] = true
		if r.Type != "task" {
			t.Errorf("expected type task, got %s", r.Type)
		}
	}
	if !ids["TASK-01DDDDDDDDD01"] || !ids["TASK-01DDDDDDDDD02"] {
		t.Errorf("unexpected task IDs: %v", ids)
	}
}

func TestCrossEntityQuery_NoFeatures(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	svc := NewEntityService(root)

	// Create a feature directory so List doesn't fail
	if err := os.MkdirAll(filepath.Join(root, "features"), 0o755); err != nil {
		t.Fatal(err)
	}

	results, err := svc.CrossEntityQuery("P1-nonexistent")
	if err != nil {
		t.Fatalf("CrossEntityQuery() error = %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("expected 0 results, got %d", len(results))
	}
}

func TestCrossEntityQuery_EmptyPlanID(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	svc := NewEntityService(root)

	_, err := svc.CrossEntityQuery("")
	if err == nil {
		t.Fatal("expected error for empty plan_id")
	}
}
