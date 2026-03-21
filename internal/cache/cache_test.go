package cache

import (
	"path/filepath"
	"testing"
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
		ID:         "E-001",
		Slug:       "test-epic",
		Status:     "proposed",
		Title:      "Test Epic",
		Summary:    "A test epic",
		ParentRef:  "",
		FilePath:   ".kbz/state/epics/E-001-test-epic.yaml",
		FieldsJSON: `{"id":"E-001","slug":"test-epic","status":"proposed","title":"Test Epic","summary":"A test epic"}`,
	}

	if err := c.Upsert(row); err != nil {
		t.Fatalf("upsert: %v", err)
	}

	slug, filePath, found := c.LookupByID("epic", "E-001")
	if !found {
		t.Fatal("expected to find entity")
	}
	if slug != "test-epic" {
		t.Errorf("slug = %q, want %q", slug, "test-epic")
	}
	if filePath != ".kbz/state/epics/E-001-test-epic.yaml" {
		t.Errorf("filePath = %q, want %q", filePath, ".kbz/state/epics/E-001-test-epic.yaml")
	}
}

func TestLookupByID_NotFound(t *testing.T) {
	c := openTestCache(t)

	_, _, found := c.LookupByID("epic", "E-999")
	if found {
		t.Fatal("expected not found")
	}
}

func TestFindByID_AcrossTypes(t *testing.T) {
	c := openTestCache(t)

	if err := c.Upsert(EntityRow{
		EntityType: "feature",
		ID:         "FEAT-001",
		Slug:       "my-feature",
		Status:     "draft",
		FilePath:   "features/FEAT-001-my-feature.yaml",
		FieldsJSON: `{}`,
	}); err != nil {
		t.Fatalf("upsert: %v", err)
	}

	entityType, slug, filePath, found := c.FindByID("FEAT-001")
	if !found {
		t.Fatal("expected to find entity")
	}
	if entityType != "feature" {
		t.Errorf("entityType = %q, want %q", entityType, "feature")
	}
	if slug != "my-feature" {
		t.Errorf("slug = %q, want %q", slug, "my-feature")
	}
	if filePath != "features/FEAT-001-my-feature.yaml" {
		t.Errorf("filePath = %q, want %q", filePath, "features/FEAT-001-my-feature.yaml")
	}
}

func TestFindByID_NotFound(t *testing.T) {
	c := openTestCache(t)

	_, _, _, found := c.FindByID("NOPE-999")
	if found {
		t.Fatal("expected not found")
	}
}

func TestUpsert_UpdatesExisting(t *testing.T) {
	c := openTestCache(t)

	row := EntityRow{
		EntityType: "epic",
		ID:         "E-001",
		Slug:       "old-slug",
		Status:     "proposed",
		FilePath:   "epics/E-001-old-slug.yaml",
		FieldsJSON: `{}`,
	}
	if err := c.Upsert(row); err != nil {
		t.Fatalf("first upsert: %v", err)
	}

	row.Slug = "new-slug"
	row.Status = "approved"
	row.FilePath = "epics/E-001-new-slug.yaml"
	if err := c.Upsert(row); err != nil {
		t.Fatalf("second upsert: %v", err)
	}

	slug, filePath, found := c.LookupByID("epic", "E-001")
	if !found {
		t.Fatal("expected to find entity after update")
	}
	if slug != "new-slug" {
		t.Errorf("slug = %q, want %q", slug, "new-slug")
	}
	if filePath != "epics/E-001-new-slug.yaml" {
		t.Errorf("filePath = %q, want %q", filePath, "epics/E-001-new-slug.yaml")
	}
}

func TestDelete(t *testing.T) {
	c := openTestCache(t)

	if err := c.Upsert(EntityRow{
		EntityType: "bug",
		ID:         "BUG-001",
		Slug:       "test-bug",
		FieldsJSON: `{}`,
	}); err != nil {
		t.Fatalf("upsert: %v", err)
	}

	if err := c.Delete("bug", "BUG-001"); err != nil {
		t.Fatalf("delete: %v", err)
	}

	_, _, found := c.LookupByID("bug", "BUG-001")
	if found {
		t.Fatal("expected not found after delete")
	}
}

func TestDelete_NonExistent(t *testing.T) {
	c := openTestCache(t)

	// Should not error when deleting a non-existent entity.
	if err := c.Delete("epic", "E-999"); err != nil {
		t.Fatalf("delete non-existent: %v", err)
	}
}

func TestListByType(t *testing.T) {
	c := openTestCache(t)

	for _, row := range []EntityRow{
		{EntityType: "epic", ID: "E-001", Slug: "alpha", FieldsJSON: `{}`},
		{EntityType: "epic", ID: "E-002", Slug: "beta", FieldsJSON: `{}`},
		{EntityType: "feature", ID: "FEAT-001", Slug: "gamma", FieldsJSON: `{}`},
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
	if epics[0].ID != "E-001" || epics[1].ID != "E-002" {
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
		{EntityType: "epic", ID: "E-001", Slug: "a", FieldsJSON: `{}`},
		{EntityType: "bug", ID: "BUG-001", Slug: "b", FieldsJSON: `{}`},
		{EntityType: "task", ID: "FEAT-001.1", Slug: "c", FieldsJSON: `{}`},
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

	for _, row := range []EntityRow{
		{EntityType: "epic", ID: "E-001", Slug: "a", FieldsJSON: `{}`},
		{EntityType: "epic", ID: "E-002", Slug: "b", FieldsJSON: `{}`},
		{EntityType: "feature", ID: "FEAT-001", Slug: "c", FieldsJSON: `{}`},
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
		ID:         "DEC-001",
		Slug:       "my-dec",
		FieldsJSON: `{}`,
	}); err != nil {
		t.Fatalf("upsert: %v", err)
	}

	if !c.EntityExists("decision", "DEC-001") {
		t.Error("expected entity to exist")
	}
	if c.EntityExists("decision", "DEC-999") {
		t.Error("expected entity to not exist")
	}
	if c.EntityExists("epic", "DEC-001") {
		t.Error("expected wrong-type lookup to not exist")
	}
}

func TestClear(t *testing.T) {
	c := openTestCache(t)

	if err := c.Upsert(EntityRow{
		EntityType: "epic",
		ID:         "E-001",
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

	// Pre-populate with stale data that should be cleared.
	if err := c.Upsert(EntityRow{
		EntityType: "epic",
		ID:         "E-STALE",
		Slug:       "stale",
		FieldsJSON: `{}`,
	}); err != nil {
		t.Fatalf("upsert stale: %v", err)
	}

	records := []RebuildRecord{
		{
			EntityType: "epic",
			ID:         "E-001",
			Slug:       "alpha",
			FilePath:   "epics/E-001-alpha.yaml",
			Fields: map[string]any{
				"id":      "E-001",
				"slug":    "alpha",
				"title":   "Alpha Epic",
				"status":  "proposed",
				"summary": "Alpha summary",
			},
		},
		{
			EntityType: "feature",
			ID:         "FEAT-001",
			Slug:       "beta",
			FilePath:   "features/FEAT-001-beta.yaml",
			Fields: map[string]any{
				"id":      "FEAT-001",
				"slug":    "beta",
				"epic":    "E-001",
				"status":  "draft",
				"summary": "Beta summary",
			},
		},
		{
			EntityType: "task",
			ID:         "FEAT-001.1",
			Slug:       "gamma",
			FilePath:   "tasks/FEAT-001.1-gamma.yaml",
			Fields: map[string]any{
				"id":             "FEAT-001.1",
				"parent_feature": "FEAT-001",
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
	if c.EntityExists("epic", "E-STALE") {
		t.Error("stale entity should have been cleared")
	}

	// New entities should be present.
	slug, _, found := c.LookupByID("epic", "E-001")
	if !found {
		t.Fatal("E-001 not found after rebuild")
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
	if features[0].ParentRef != "E-001" {
		t.Errorf("feature parent_ref = %q, want %q", features[0].ParentRef, "E-001")
	}

	tasks, err := c.ListByType("task")
	if err != nil {
		t.Fatalf("list tasks: %v", err)
	}
	if len(tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(tasks))
	}
	if tasks[0].ParentRef != "FEAT-001" {
		t.Errorf("task parent_ref = %q, want %q", tasks[0].ParentRef, "FEAT-001")
	}
}

func TestGetFields(t *testing.T) {
	c := openTestCache(t)

	if err := c.Upsert(EntityRow{
		EntityType: "epic",
		ID:         "E-001",
		Slug:       "test",
		FieldsJSON: `{"id":"E-001","slug":"test","title":"Test","status":"proposed","summary":"A test"}`,
	}); err != nil {
		t.Fatalf("upsert: %v", err)
	}

	fields, err := c.GetFields("epic", "E-001")
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

	_, err := c.GetFields("epic", "E-999")
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
		{"feature", map[string]any{"epic": "E-001"}, "E-001"},
		{"task", map[string]any{"parent_feature": "FEAT-001"}, "FEAT-001"},
		{"bug", map[string]any{"origin_feature": "FEAT-002"}, "FEAT-002"},
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

	// First rebuild with two entities.
	_, err := c.Rebuild([]RebuildRecord{
		{EntityType: "epic", ID: "E-001", Slug: "a", Fields: map[string]any{"id": "E-001"}},
		{EntityType: "epic", ID: "E-002", Slug: "b", Fields: map[string]any{"id": "E-002"}},
	})
	if err != nil {
		t.Fatalf("first rebuild: %v", err)
	}

	// Second rebuild with only one entity.
	count, err := c.Rebuild([]RebuildRecord{
		{EntityType: "epic", ID: "E-003", Slug: "c", Fields: map[string]any{"id": "E-003"}},
	})
	if err != nil {
		t.Fatalf("second rebuild: %v", err)
	}
	if count != 1 {
		t.Errorf("count = %d, want 1", count)
	}

	// Old entities should be gone.
	if c.EntityExists("epic", "E-001") {
		t.Error("E-001 should be gone after rebuild")
	}
	if c.EntityExists("epic", "E-002") {
		t.Error("E-002 should be gone after rebuild")
	}
	if !c.EntityExists("epic", "E-003") {
		t.Error("E-003 should exist after rebuild")
	}
}
