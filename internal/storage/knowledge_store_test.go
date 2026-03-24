package storage

import (
	"errors"
	"testing"
)

func TestKnowledgeStore_WriteLoad_RoundTrip(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := NewKnowledgeStore(root)

	id := "KE-01JTEST0001234"
	fields := map[string]any{
		"id":         id,
		"tier":       3,
		"topic":      "api-json-naming-convention",
		"scope":      "project",
		"content":    "Use camelCase for all JSON API field names.",
		"status":     "contributed",
		"use_count":  0,
		"miss_count": 0,
		"confidence": 0.5,
		"ttl_days":   30,
		"created":    "2024-01-01T00:00:00Z",
		"created_by": "test-agent",
		"updated":    "2024-01-01T00:00:00Z",
	}

	record := KnowledgeRecord{
		ID:     id,
		Fields: fields,
	}

	path, err := store.Write(record)
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	if path == "" {
		t.Fatal("Write() returned empty path")
	}

	loaded, err := store.Load(id)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if loaded.ID != id {
		t.Errorf("Load() ID = %q, want %q", loaded.ID, id)
	}
	if loaded.FileHash == "" {
		t.Error("Load() FileHash should not be empty")
	}

	// Verify key fields round-trip correctly
	wantFields := map[string]any{
		"id":         id,
		"topic":      "api-json-naming-convention",
		"scope":      "project",
		"content":    "Use camelCase for all JSON API field names.",
		"status":     "contributed",
		"created_by": "test-agent",
	}
	for k, want := range wantFields {
		got := loaded.Fields[k]
		if got != want {
			t.Errorf("Load() Fields[%q] = %v (%T), want %v (%T)", k, got, got, want, want)
		}
	}

	// Integers should round-trip as int
	if v, ok := loaded.Fields["tier"]; !ok {
		t.Error("Load() Fields missing 'tier'")
	} else if v != 3 {
		t.Errorf("Load() Fields['tier'] = %v (%T), want 3 (int)", v, v)
	}
	if v, ok := loaded.Fields["use_count"]; !ok {
		t.Error("Load() Fields missing 'use_count'")
	} else if v != 0 {
		t.Errorf("Load() Fields['use_count'] = %v (%T), want 0 (int)", v, v)
	}
}

func TestKnowledgeStore_WriteLoad_OptionalFields(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := NewKnowledgeStore(root)

	id := "KE-01JTEST0OPTIONAL"
	fields := map[string]any{
		"id":                id,
		"tier":              2,
		"topic":             "go-error-wrapping",
		"scope":             "project",
		"content":           "Always wrap errors with context using fmt.Errorf.",
		"learned_from":      "TASK-01ABC",
		"status":            "confirmed",
		"use_count":         5,
		"miss_count":        0,
		"confidence":        0.5,
		"ttl_days":          90,
		"promoted_from":     "KE-01JTEST0000001",
		"deprecated_reason": "",
		"tags":              []string{"go", "errors"},
		"created":           "2024-06-01T12:00:00Z",
		"created_by":        "agent-x",
		"updated":           "2024-06-01T12:00:00Z",
	}

	record := KnowledgeRecord{ID: id, Fields: fields}
	if _, err := store.Write(record); err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	loaded, err := store.Load(id)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if v, _ := loaded.Fields["learned_from"].(string); v != "TASK-01ABC" {
		t.Errorf("Fields['learned_from'] = %q, want %q", v, "TASK-01ABC")
	}
	if v, _ := loaded.Fields["promoted_from"].(string); v != "KE-01JTEST0000001" {
		t.Errorf("Fields['promoted_from'] = %q, want %q", v, "KE-01JTEST0000001")
	}

	// Tags should come back as []any after YAML round-trip
	tagsAny, ok := loaded.Fields["tags"]
	if !ok {
		t.Fatal("Load() Fields missing 'tags'")
	}
	tags, ok := tagsAny.([]any)
	if !ok {
		t.Fatalf("Fields['tags'] type = %T, want []any", tagsAny)
	}
	if len(tags) != 2 {
		t.Fatalf("Fields['tags'] len = %d, want 2", len(tags))
	}
	if tags[0] != "go" || tags[1] != "errors" {
		t.Errorf("Fields['tags'] = %v, want [go errors]", tags)
	}
}

func TestKnowledgeStore_Load_NotFound(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := NewKnowledgeStore(root)

	_, err := store.Load("KE-DOESNOTEXIST")
	if err == nil {
		t.Fatal("Load() expected error for missing entry, got nil")
	}
}

func TestKnowledgeStore_Load_EmptyID(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := NewKnowledgeStore(root)

	_, err := store.Load("")
	if err == nil {
		t.Fatal("Load() expected error for empty ID, got nil")
	}
}

func TestKnowledgeStore_LoadAll_Empty(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := NewKnowledgeStore(root)

	records, err := store.LoadAll()
	if err != nil {
		t.Fatalf("LoadAll() error = %v", err)
	}
	if len(records) != 0 {
		t.Errorf("LoadAll() returned %d records, want 0", len(records))
	}
}

func TestKnowledgeStore_LoadAll_MultipleRecords(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := NewKnowledgeStore(root)

	ids := []string{"KE-01JTEST0AAA001", "KE-01JTEST0BBB002", "KE-01JTEST0CCC003"}
	for _, id := range ids {
		record := KnowledgeRecord{
			ID: id,
			Fields: map[string]any{
				"id":         id,
				"tier":       3,
				"topic":      "topic-" + id,
				"scope":      "project",
				"content":    "Content for " + id,
				"status":     "contributed",
				"use_count":  0,
				"miss_count": 0,
				"confidence": 0.5,
				"ttl_days":   30,
				"created":    "2024-01-01T00:00:00Z",
				"created_by": "test",
				"updated":    "2024-01-01T00:00:00Z",
			},
		}
		if _, err := store.Write(record); err != nil {
			t.Fatalf("Write(%s) error = %v", id, err)
		}
	}

	records, err := store.LoadAll()
	if err != nil {
		t.Fatalf("LoadAll() error = %v", err)
	}
	if len(records) != 3 {
		t.Errorf("LoadAll() returned %d records, want 3", len(records))
	}
}

func TestKnowledgeStore_Write_ValidationErrors(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := NewKnowledgeStore(root)

	t.Run("empty ID", func(t *testing.T) {
		t.Parallel()
		_, err := store.Write(KnowledgeRecord{ID: "", Fields: map[string]any{"id": ""}})
		if err == nil {
			t.Fatal("Write() expected error for empty ID")
		}
	})

	t.Run("empty fields", func(t *testing.T) {
		t.Parallel()
		_, err := store.Write(KnowledgeRecord{ID: "KE-01JTEST", Fields: nil})
		if err == nil {
			t.Fatal("Write() expected error for nil fields")
		}
	})

	t.Run("id mismatch", func(t *testing.T) {
		t.Parallel()
		_, err := store.Write(KnowledgeRecord{
			ID:     "KE-01JTEST",
			Fields: map[string]any{"id": "KE-WRONG"},
		})
		if err == nil {
			t.Fatal("Write() expected error for id mismatch")
		}
	})
}

func TestKnowledgeStore_Write_OptimisticLock(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := NewKnowledgeStore(root)

	id := "KE-01JTEST0LOCKME1"
	fields := map[string]any{
		"id":         id,
		"tier":       3,
		"topic":      "lock-test",
		"scope":      "project",
		"content":    "Original content.",
		"status":     "contributed",
		"use_count":  0,
		"miss_count": 0,
		"confidence": 0.5,
		"ttl_days":   30,
		"created":    "2024-01-01T00:00:00Z",
		"created_by": "test",
		"updated":    "2024-01-01T00:00:00Z",
	}

	// Write initial record
	if _, err := store.Write(KnowledgeRecord{ID: id, Fields: fields}); err != nil {
		t.Fatalf("initial Write() error = %v", err)
	}

	// Load to get FileHash
	loaded, err := store.Load(id)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Modify on disk by writing without the hash (simulate concurrent write)
	fields2 := copyKnowledgeFields(fields)
	fields2["content"] = "Concurrently modified."
	if _, err := store.Write(KnowledgeRecord{ID: id, Fields: fields2}); err != nil {
		t.Fatalf("concurrent Write() error = %v", err)
	}

	// Now try to write with stale FileHash — should fail with ErrConflict
	loaded.Fields["content"] = "My update."
	_, err = store.Write(loaded)
	if err == nil {
		t.Fatal("Write() with stale FileHash should return error")
	}
	if !errors.Is(err, ErrConflict) {
		t.Errorf("Write() error = %v, want ErrConflict", err)
	}
}

func TestKnowledgeStore_Delete(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := NewKnowledgeStore(root)

	id := "KE-01JTEST0DELETE1"
	record := KnowledgeRecord{
		ID: id,
		Fields: map[string]any{
			"id":         id,
			"tier":       3,
			"topic":      "delete-test",
			"scope":      "project",
			"content":    "To be deleted.",
			"status":     "contributed",
			"use_count":  0,
			"miss_count": 0,
			"confidence": 0.5,
			"ttl_days":   30,
			"created":    "2024-01-01T00:00:00Z",
			"created_by": "test",
			"updated":    "2024-01-01T00:00:00Z",
		},
	}

	if _, err := store.Write(record); err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	if _, err := store.Load(id); err != nil {
		t.Fatalf("Load() before delete error = %v", err)
	}

	if err := store.Delete(id); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	// Second delete should be a no-op (not an error)
	if err := store.Delete(id); err != nil {
		t.Errorf("Delete() on missing record error = %v, want nil", err)
	}

	if _, err := store.Load(id); err == nil {
		t.Fatal("Load() after delete should return error")
	}
}

// copyKnowledgeFields creates a shallow copy of a fields map.
func copyKnowledgeFields(fields map[string]any) map[string]any {
	out := make(map[string]any, len(fields))
	for k, v := range fields {
		out[k] = v
	}
	return out
}
