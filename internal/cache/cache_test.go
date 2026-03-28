package cache

import (
	"path/filepath"
	"testing"

	"github.com/sambeau/kanbanzai/internal/testutil"
)

func openTestCache(t *testing.T) *Cache {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "cache")
	c, err := Open(dir)
	if err != nil {
		t.Fatalf("open cache: %v", err)
	}
	t.Cleanup(func() { c.Close() })
	return c
}

func TestOpen_CreatesDatabase(t *testing.T) {
	c := openTestCache(t)
	if c.Path() == "" {
		t.Fatal("expected non-empty path")
	}
}

func TestUpsert_AndLookupByID(t *testing.T) {
	c := openTestCache(t)

	row := EntityRow{
		EntityType: "epic",
		ID:         testutil.TestEpicID,
		Slug:       "test-epic",
		Status:     "proposed",
		Title:      "Test Epic",
		Summary:    "A test epic",
		ParentRef:  "",
		FilePath:   ".kbz/state/epics/" + testutil.TestEpicID + "-test-epic.yaml",
		FieldsJSON: `{"id":"` + testutil.TestEpicID + `","slug":"test-epic","status":"proposed","title":"Test Epic","summary":"A test epic"}`,
	}

	if err := c.Upsert(row); err != nil {
		t.Fatalf("upsert: %v", err)
	}

	slug, filePath, found := c.LookupByID("epic", testutil.TestEpicID)
	if !found {
		t.Fatal("expected to find entity")
	}
	if slug != "test-epic" {
		t.Errorf("slug = %q, want %q", slug, "test-epic")
	}
	wantPath := ".kbz/state/epics/" + testutil.TestEpicID + "-test-epic.yaml"
	if filePath != wantPath {
		t.Errorf("filePath = %q, want %q", filePath, wantPath)
	}
}

func TestLookupByID_NotFound(t *testing.T) {
	c := openTestCache(t)

	_, _, found := c.LookupByID("epic", "EPIC-01J3KBNXS7VX9")
	if found {
		t.Fatal("expected not found")
	}
}

func TestFindByID_AcrossTypes(t *testing.T) {
	c := openTestCache(t)

	if err := c.Upsert(EntityRow{
		EntityType: "feature",
		ID:         testutil.TestFeatureID,
		Slug:       "my-feature",
		Status:     "draft",
		FilePath:   "features/" + testutil.TestFeatureID + "-my-feature.yaml",
		FieldsJSON: `{}`,
	}); err != nil {
		t.Fatalf("upsert: %v", err)
	}

	entityType, slug, filePath, found := c.FindByID(testutil.TestFeatureID)
	if !found {
		t.Fatal("expected to find entity")
	}
	if entityType != "feature" {
		t.Errorf("entityType = %q, want %q", entityType, "feature")
	}
	if slug != "my-feature" {
		t.Errorf("slug = %q, want %q", slug, "my-feature")
	}
	wantPath := "features/" + testutil.TestFeatureID + "-my-feature.yaml"
	if filePath != wantPath {
		t.Errorf("filePath = %q, want %q", filePath, wantPath)
	}
}

func TestFindByID_NotFound(t *testing.T) {
	c := openTestCache(t)

	_, _, _, found := c.FindByID("NOPE-01J3KNOTFND01")
	if found {
		t.Fatal("expected not found")
	}
}

func TestUpsert_UpdatesExisting(t *testing.T) {
	c := openTestCache(t)

	row := EntityRow{
		EntityType: "epic",
		ID:         testutil.TestEpicID,
		Slug:       "old-slug",
		Status:     "proposed",
		FilePath:   "epics/" + testutil.TestEpicID + "-old-slug.yaml",
		FieldsJSON: `{}`,
	}
	if err := c.Upsert(row); err != nil {
		t.Fatalf("first upsert: %v", err)
	}

	row.Slug = "new-slug"
	row.Status = "approved"
	row.FilePath = "epics/" + testutil.TestEpicID + "-new-slug.yaml"
	if err := c.Upsert(row); err != nil {
		t.Fatalf("second upsert: %v", err)
	}

	slug, filePath, found := c.LookupByID("epic", testutil.TestEpicID)
	if !found {
		t.Fatal("expected to find entity after update")
	}
	if slug != "new-slug" {
		t.Errorf("slug = %q, want %q", slug, "new-slug")
	}
	wantPath := "epics/" + testutil.TestEpicID + "-new-slug.yaml"
	if filePath != wantPath {
		t.Errorf("filePath = %q, want %q", filePath, wantPath)
	}
}

func TestDelete(t *testing.T) {
	c := openTestCache(t)

	if err := c.Upsert(EntityRow{
		EntityType: "bug",
		ID:         testutil.TestBugID,
		Slug:       "test-bug",
		FieldsJSON: `{}`,
	}); err != nil {
		t.Fatalf("upsert: %v", err)
	}

	if err := c.Delete("bug", testutil.TestBugID); err != nil {
		t.Fatalf("delete: %v", err)
	}

	_, _, found := c.LookupByID("bug", testutil.TestBugID)
	if found {
		t.Fatal("expected not found after delete")
	}
}

func TestDelete_NonExistent(t *testing.T) {
	c := openTestCache(t)

	// Should not error when deleting a non-existent entity.
	if err := c.Delete("epic", "EPIC-01J3KBNXS7VX9"); err != nil {
		t.Fatalf("delete non-existent: %v", err)
	}
}

func TestListByType(t *testing.T) {
	c := openTestCache(t)

	epicID2 := "EPIC-01J3K9PYR5TV7"

	for _, row := range []EntityRow{
		{EntityType: "epic", ID: testutil.TestEpicID, Slug: "alpha", FieldsJSON: `{}`},
		{EntityType: "epic", ID: epicID2, Slug: "beta", FieldsJSON: `{}`},
		{EntityType: "feature", ID: testutil.TestFeatureID, Slug: "gamma", FieldsJSON: `{}`},
	} {
		if err := c.Upsert(row); err != nil {
			t.Fatalf("upsert %s: %v", row.ID, err)
		}
	}

	epics, err := c.ListByType("epic")
	if err != nil {
		t.Fatalf("list epics: %v", err)
	}
	if len(epics) != 2 {
		t.Fatalf("expected 2 epics, got %d", len(epics))
	}
	if epics[0].ID != epicID2 || epics[1].ID != testutil.TestEpicID {
		t.Errorf("expected sorted by ID: got %s, %s", epics[0].ID, epics[1].ID)
	}

	features, err := c.ListByType("feature")
	if err != nil {
		t.Fatalf("list features: %v", err)
	}
	if len(features) != 1 {
		t.Fatalf("expected 1 feature, got %d", len(features))
	}
}

func TestListByType_Empty(t *testing.T) {
	c := openTestCache(t)

	rows, err := c.ListByType("task")
	if err != nil {
		t.Fatalf("list tasks: %v", err)
	}
	if len(rows) != 0 {
		t.Fatalf("expected 0 tasks, got %d", len(rows))
	}
}

func TestListAll(t *testing.T) {
	c := openTestCache(t)

	for _, row := range []EntityRow{
		{EntityType: "epic", ID: testutil.TestEpicID, Slug: "a", FieldsJSON: `{}`},
		{EntityType: "bug", ID: testutil.TestBugID, Slug: "b", FieldsJSON: `{}`},
		{EntityType: "task", ID: testutil.TestTaskID, Slug: "c", FieldsJSON: `{}`},
	} {
		if err := c.Upsert(row); err != nil {
			t.Fatalf("upsert %s: %v", row.ID, err)
		}
	}

	all, err := c.ListAll()
	if err != nil {
		t.Fatalf("list all: %v", err)
	}
	if len(all) != 3 {
		t.Fatalf("expected 3, got %d", len(all))
	}
	// Should be ordered by entity_type then id.
	if all[0].EntityType != "bug" {
		t.Errorf("first entity type = %q, want %q", all[0].EntityType, "bug")
	}
}

func TestCount(t *testing.T) {
	c := openTestCache(t)

	epicID2 := "EPIC-01J3K9PYR5TV7"

	for _, row := range []EntityRow{
		{EntityType: "epic", ID: testutil.TestEpicID, Slug: "a", FieldsJSON: `{}`},
		{EntityType: "epic", ID: epicID2, Slug: "b", FieldsJSON: `{}`},
		{EntityType: "feature", ID: testutil.TestFeatureID, Slug: "c", FieldsJSON: `{}`},
	} {
		if err := c.Upsert(row); err != nil {
			t.Fatalf("upsert %s: %v", row.ID, err)
		}
	}

	total, err := c.Count("")
	if err != nil {
		t.Fatalf("count all: %v", err)
	}
	if total != 3 {
		t.Errorf("total = %d, want 3", total)
	}

	epicCount, err := c.Count("epic")
	if err != nil {
		t.Fatalf("count epics: %v", err)
	}
	if epicCount != 2 {
		t.Errorf("epic count = %d, want 2", epicCount)
	}
}

func TestEntityExists(t *testing.T) {
	c := openTestCache(t)

	if err := c.Upsert(EntityRow{
		EntityType: "decision",
		ID:         testutil.TestDecisionID,
		Slug:       "my-dec",
		FieldsJSON: `{}`,
	}); err != nil {
		t.Fatalf("upsert: %v", err)
	}

	if !c.EntityExists("decision", testutil.TestDecisionID) {
		t.Error("expected entity to exist")
	}
	if c.EntityExists("decision", "DEC-01J3KNOTFND01") {
		t.Error("expected entity to not exist")
	}
	if c.EntityExists("epic", testutil.TestDecisionID) {
		t.Error("expected wrong-type lookup to not exist")
	}
}

func TestClear(t *testing.T) {
	c := openTestCache(t)

	if err := c.Upsert(EntityRow{
		EntityType: "epic",
		ID:         testutil.TestEpicID,
		Slug:       "a",
		FieldsJSON: `{}`,
	}); err != nil {
		t.Fatalf("upsert: %v", err)
	}

	if err := c.Clear(); err != nil {
		t.Fatalf("clear: %v", err)
	}

	count, err := c.Count("")
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 0 {
		t.Errorf("count after clear = %d, want 0", count)
	}
}

func TestRebuild(t *testing.T) {
	c := openTestCache(t)

	staleEpicID := "EPIC-01J3KCSTALE01"

	// Pre-populate with stale data that should be cleared.
	if err := c.Upsert(EntityRow{
		EntityType: "epic",
		ID:         staleEpicID,
		Slug:       "stale",
		FieldsJSON: `{}`,
	}); err != nil {
		t.Fatalf("upsert stale: %v", err)
	}

	records := []RebuildRecord{
		{
			EntityType: "epic",
			ID:         testutil.TestEpicID,
			Slug:       "alpha",
			FilePath:   "epics/" + testutil.TestEpicID + "-alpha.yaml",
			Fields: map[string]any{
				"id":      testutil.TestEpicID,
				"slug":    "alpha",
				"title":   "Alpha Epic",
				"status":  "proposed",
				"summary": "Alpha summary",
			},
		},
		{
			EntityType: "feature",
			ID:         testutil.TestFeatureID,
			Slug:       "beta",
			FilePath:   "features/" + testutil.TestFeatureID + "-beta.yaml",
			Fields: map[string]any{
				"id":      testutil.TestFeatureID,
				"slug":    "beta",
				"epic":    testutil.TestEpicID,
				"status":  "draft",
				"summary": "Beta summary",
			},
		},
		{
			EntityType: "task",
			ID:         testutil.TestTaskID,
			Slug:       "gamma",
			FilePath:   "tasks/" + testutil.TestTaskID + "-gamma.yaml",
			Fields: map[string]any{
				"id":             testutil.TestTaskID,
				"parent_feature": testutil.TestFeatureID,
				"slug":           "gamma",
				"status":         "queued",
				"summary":        "Gamma task",
			},
		},
	}

	count, err := c.Rebuild(records)
	if err != nil {
		t.Fatalf("rebuild: %v", err)
	}
	if count != 3 {
		t.Errorf("rebuild count = %d, want 3", count)
	}

	// Stale entity should be gone.
	if c.EntityExists("epic", staleEpicID) {
		t.Error("stale entity should have been cleared")
	}

	// New entities should be present.
	slug, _, found := c.LookupByID("epic", testutil.TestEpicID)
	if !found {
		t.Fatalf("%s not found after rebuild", testutil.TestEpicID)
	}
	if slug != "alpha" {
		t.Errorf("slug = %q, want %q", slug, "alpha")
	}

	// Check parent_ref extraction.
	features, err := c.ListByType("feature")
	if err != nil {
		t.Fatalf("list features: %v", err)
	}
	if len(features) != 1 {
		t.Fatalf("expected 1 feature, got %d", len(features))
	}
	if features[0].ParentRef != testutil.TestEpicID {
		t.Errorf("feature parent_ref = %q, want %q", features[0].ParentRef, testutil.TestEpicID)
	}

	tasks, err := c.ListByType("task")
	if err != nil {
		t.Fatalf("list tasks: %v", err)
	}
	if len(tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(tasks))
	}
	if tasks[0].ParentRef != testutil.TestFeatureID {
		t.Errorf("task parent_ref = %q, want %q", tasks[0].ParentRef, testutil.TestFeatureID)
	}
}

func TestGetFields(t *testing.T) {
	c := openTestCache(t)

	if err := c.Upsert(EntityRow{
		EntityType: "epic",
		ID:         testutil.TestEpicID,
		Slug:       "test",
		FieldsJSON: `{"id":"` + testutil.TestEpicID + `","slug":"test","title":"Test","status":"proposed","summary":"A test"}`,
	}); err != nil {
		t.Fatalf("upsert: %v", err)
	}

	fields, err := c.GetFields("epic", testutil.TestEpicID)
	if err != nil {
		t.Fatalf("get fields: %v", err)
	}
	if fields["title"] != "Test" {
		t.Errorf("title = %v, want %q", fields["title"], "Test")
	}
	if fields["status"] != "proposed" {
		t.Errorf("status = %v, want %q", fields["status"], "proposed")
	}
}

func TestGetFields_NotFound(t *testing.T) {
	c := openTestCache(t)

	_, err := c.GetFields("epic", "EPIC-01J3KBNXS7VX9")
	if err == nil {
		t.Fatal("expected error for missing entity")
	}
}

func TestExtractParentRef(t *testing.T) {
	tests := []struct {
		entityType string
		fields     map[string]any
		want       string
	}{
		{"feature", map[string]any{"epic": testutil.TestEpicID}, testutil.TestEpicID},
		{"task", map[string]any{"parent_feature": testutil.TestFeatureID}, testutil.TestFeatureID},
		{"bug", map[string]any{"origin_feature": testutil.TestFeatureID2}, testutil.TestFeatureID2},
		{"epic", map[string]any{"title": "No parent"}, ""},
		{"decision", map[string]any{}, ""},
		{"feature", map[string]any{}, ""},
	}

	for _, tt := range tests {
		got := extractParentRef(tt.entityType, tt.fields)
		if got != tt.want {
			t.Errorf("extractParentRef(%q, %v) = %q, want %q", tt.entityType, tt.fields, got, tt.want)
		}
	}
}

func TestStringFromFields(t *testing.T) {
	tests := []struct {
		name   string
		fields map[string]any
		key    string
		want   string
	}{
		{"string value", map[string]any{"title": "hello"}, "title", "hello"},
		{"missing key", map[string]any{"title": "hello"}, "status", ""},
		{"nil map", nil, "title", ""},
		{"int value", map[string]any{"count": 42}, "count", "42"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stringFromFields(tt.fields, tt.key)
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRebuild_EmptyRecords(t *testing.T) {
	c := openTestCache(t)

	count, err := c.Rebuild(nil)
	if err != nil {
		t.Fatalf("rebuild with nil: %v", err)
	}
	if count != 0 {
		t.Errorf("count = %d, want 0", count)
	}
}

func TestRebuild_ClearsOldData(t *testing.T) {
	c := openTestCache(t)

	epicID2 := "EPIC-01J3K9PYR5TV7"
	epicID3 := "EPIC-01J3KANXR6UW8"

	// First rebuild with two entities.
	_, err := c.Rebuild([]RebuildRecord{
		{EntityType: "epic", ID: testutil.TestEpicID, Slug: "a", Fields: map[string]any{"id": testutil.TestEpicID}},
		{EntityType: "epic", ID: epicID2, Slug: "b", Fields: map[string]any{"id": epicID2}},
	})
	if err != nil {
		t.Fatalf("first rebuild: %v", err)
	}

	// Second rebuild with only one entity.
	count, err := c.Rebuild([]RebuildRecord{
		{EntityType: "epic", ID: epicID3, Slug: "c", Fields: map[string]any{"id": epicID3}},
	})
	if err != nil {
		t.Fatalf("second rebuild: %v", err)
	}
	if count != 1 {
		t.Errorf("count = %d, want 1", count)
	}

	// Old entities should be gone.
	if c.EntityExists("epic", testutil.TestEpicID) {
		t.Errorf("%s should be gone after rebuild", testutil.TestEpicID)
	}
	if c.EntityExists("epic", epicID2) {
		t.Errorf("%s should be gone after rebuild", epicID2)
	}
	if !c.EntityExists("epic", epicID3) {
		t.Errorf("%s should exist after rebuild", epicID3)
	}
}
