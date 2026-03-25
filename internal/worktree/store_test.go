package worktree

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestStore_Create(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := NewStore(root)

	record := Record{
		EntityID:  "FEAT-01JX987654321",
		Branch:    "feature/FEAT-01JX987654321-user-profiles",
		Path:      ".worktrees/FEAT-01JX987654321-user-profiles",
		Status:    StatusActive,
		Created:   time.Date(2025, 1, 27, 10, 0, 0, 0, time.UTC),
		CreatedBy: "sambeau",
	}

	created, err := store.Create(record)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Verify ID was allocated
	if created.ID == "" {
		t.Error("Create() should allocate an ID")
	}
	if !strings.HasPrefix(created.ID, "WT-") {
		t.Errorf("Create() ID = %q, want WT- prefix", created.ID)
	}

	// Verify FileHash was set
	if created.FileHash == "" {
		t.Error("Create() should set FileHash for optimistic locking")
	}

	// Verify file was written
	path := filepath.Join(root, WorktreesDir, created.ID+".yaml")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Errorf("Create() did not write file at %s", path)
	}
}

func TestStore_Create_WithExistingID(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := NewStore(root)

	record := Record{
		ID:        "WT-01JXCUSTOM1234",
		EntityID:  "FEAT-01JX987654321",
		Branch:    "feature/FEAT-01JX987654321-user-profiles",
		Path:      ".worktrees/FEAT-01JX987654321-user-profiles",
		Status:    StatusActive,
		Created:   time.Date(2025, 1, 27, 10, 0, 0, 0, time.UTC),
		CreatedBy: "sambeau",
	}

	created, err := store.Create(record)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if created.ID != "WT-01JXCUSTOM1234" {
		t.Errorf("Create() ID = %q, want %q", created.ID, "WT-01JXCUSTOM1234")
	}
}

func TestStore_Create_ValidationErrors(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := NewStore(root)

	baseRecord := Record{
		ID:        "WT-01JX123456789",
		EntityID:  "FEAT-01JX987654321",
		Branch:    "feature/FEAT-01JX987654321-user-profiles",
		Path:      ".worktrees/FEAT-01JX987654321-user-profiles",
		Status:    StatusActive,
		Created:   time.Date(2025, 1, 27, 10, 0, 0, 0, time.UTC),
		CreatedBy: "sambeau",
	}

	tests := []struct {
		name   string
		modify func(*Record)
		errMsg string
	}{
		{
			name:   "missing entity_id",
			modify: func(r *Record) { r.EntityID = "" },
			errMsg: "entity_id is required",
		},
		{
			name:   "missing branch",
			modify: func(r *Record) { r.Branch = "" },
			errMsg: "branch is required",
		},
		{
			name:   "missing path",
			modify: func(r *Record) { r.Path = "" },
			errMsg: "path is required",
		},
		{
			name:   "invalid status",
			modify: func(r *Record) { r.Status = Status("invalid") },
			errMsg: "invalid worktree status",
		},
		{
			name:   "missing created",
			modify: func(r *Record) { r.Created = time.Time{} },
			errMsg: "created timestamp is required",
		},
		{
			name:   "missing created_by",
			modify: func(r *Record) { r.CreatedBy = "" },
			errMsg: "created_by is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			record := baseRecord
			tt.modify(&record)

			_, err := store.Create(record)
			if err == nil {
				t.Fatal("Create() expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("Create() error = %q, want containing %q", err.Error(), tt.errMsg)
			}
		})
	}
}

func TestStore_Get(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := NewStore(root)

	original := Record{
		ID:        "WT-01JX123456789",
		EntityID:  "FEAT-01JX987654321",
		Branch:    "feature/FEAT-01JX987654321-user-profiles",
		Path:      ".worktrees/FEAT-01JX987654321-user-profiles",
		Status:    StatusActive,
		Created:   time.Date(2025, 1, 27, 10, 0, 0, 0, time.UTC),
		CreatedBy: "sambeau",
	}

	created, err := store.Create(original)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	loaded, err := store.Get(created.ID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	// Verify all fields
	if loaded.ID != original.ID {
		t.Errorf("Get() ID = %q, want %q", loaded.ID, original.ID)
	}
	if loaded.EntityID != original.EntityID {
		t.Errorf("Get() EntityID = %q, want %q", loaded.EntityID, original.EntityID)
	}
	if loaded.Branch != original.Branch {
		t.Errorf("Get() Branch = %q, want %q", loaded.Branch, original.Branch)
	}
	if loaded.Path != original.Path {
		t.Errorf("Get() Path = %q, want %q", loaded.Path, original.Path)
	}
	if loaded.Status != original.Status {
		t.Errorf("Get() Status = %q, want %q", loaded.Status, original.Status)
	}
	if !loaded.Created.Equal(original.Created) {
		t.Errorf("Get() Created = %v, want %v", loaded.Created, original.Created)
	}
	if loaded.CreatedBy != original.CreatedBy {
		t.Errorf("Get() CreatedBy = %q, want %q", loaded.CreatedBy, original.CreatedBy)
	}

	// FileHash should be set for optimistic locking
	if loaded.FileHash == "" {
		t.Error("Get() should set FileHash")
	}
}

func TestStore_Get_WithOptionalFields(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := NewStore(root)

	mergedAt := time.Date(2025, 1, 28, 15, 30, 0, 0, time.UTC)
	cleanupAfter := time.Date(2025, 2, 28, 0, 0, 0, 0, time.UTC)

	original := Record{
		ID:           "WT-01JX123456789",
		EntityID:     "FEAT-01JX987654321",
		Branch:       "feature/FEAT-01JX987654321-user-profiles",
		Path:         ".worktrees/FEAT-01JX987654321-user-profiles",
		Status:       StatusMerged,
		Created:      time.Date(2025, 1, 27, 10, 0, 0, 0, time.UTC),
		CreatedBy:    "sambeau",
		MergedAt:     &mergedAt,
		CleanupAfter: &cleanupAfter,
	}

	_, err := store.Create(original)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	loaded, err := store.Get(original.ID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if loaded.MergedAt == nil {
		t.Fatal("Get() MergedAt should not be nil")
	}
	if !loaded.MergedAt.Equal(mergedAt) {
		t.Errorf("Get() MergedAt = %v, want %v", loaded.MergedAt, mergedAt)
	}

	if loaded.CleanupAfter == nil {
		t.Fatal("Get() CleanupAfter should not be nil")
	}
	if !loaded.CleanupAfter.Equal(cleanupAfter) {
		t.Errorf("Get() CleanupAfter = %v, want %v", loaded.CleanupAfter, cleanupAfter)
	}
}

func TestStore_Get_NotFound(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := NewStore(root)

	_, err := store.Get("WT-NONEXISTENT123")
	if err == nil {
		t.Fatal("Get() expected error, got nil")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("Get() error = %v, want ErrNotFound", err)
	}
}

func TestStore_Get_EmptyID(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := NewStore(root)

	_, err := store.Get("")
	if err == nil {
		t.Fatal("Get() expected error, got nil")
	}
	if !strings.Contains(err.Error(), "ID is required") {
		t.Errorf("Get() error = %q, want containing 'ID is required'", err.Error())
	}
}

func TestStore_GetByEntityID(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := NewStore(root)

	// Create two worktrees for different entities
	record1 := Record{
		ID:        "WT-01JX000000001",
		EntityID:  "FEAT-01JX987654321",
		Branch:    "feature/FEAT-01JX987654321-feature-a",
		Path:      ".worktrees/FEAT-01JX987654321-feature-a",
		Status:    StatusActive,
		Created:   time.Date(2025, 1, 27, 10, 0, 0, 0, time.UTC),
		CreatedBy: "user1",
	}
	record2 := Record{
		ID:        "WT-01JX000000002",
		EntityID:  "BUG-01JX123123123",
		Branch:    "bugfix/BUG-01JX123123123-fix-crash",
		Path:      ".worktrees/BUG-01JX123123123-fix-crash",
		Status:    StatusActive,
		Created:   time.Date(2025, 1, 27, 11, 0, 0, 0, time.UTC),
		CreatedBy: "user2",
	}

	if _, err := store.Create(record1); err != nil {
		t.Fatalf("Create(record1) error = %v", err)
	}
	if _, err := store.Create(record2); err != nil {
		t.Fatalf("Create(record2) error = %v", err)
	}

	// Find by entity ID
	found, err := store.GetByEntityID("FEAT-01JX987654321")
	if err != nil {
		t.Fatalf("GetByEntityID() error = %v", err)
	}
	if found.ID != "WT-01JX000000001" {
		t.Errorf("GetByEntityID() ID = %q, want %q", found.ID, "WT-01JX000000001")
	}

	// Find bug worktree
	found, err = store.GetByEntityID("BUG-01JX123123123")
	if err != nil {
		t.Fatalf("GetByEntityID() error = %v", err)
	}
	if found.ID != "WT-01JX000000002" {
		t.Errorf("GetByEntityID() ID = %q, want %q", found.ID, "WT-01JX000000002")
	}
}

func TestStore_GetByEntityID_NotFound(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := NewStore(root)

	_, err := store.GetByEntityID("FEAT-NONEXISTENT")
	if err == nil {
		t.Fatal("GetByEntityID() expected error, got nil")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("GetByEntityID() error = %v, want ErrNotFound", err)
	}
}

func TestStore_List(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := NewStore(root)

	// Initially empty
	records, err := store.List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(records) != 0 {
		t.Errorf("List() returned %d records, want 0", len(records))
	}

	// Create some records
	for i, id := range []string{"WT-01JX000000003", "WT-01JX000000001", "WT-01JX000000002"} {
		record := Record{
			ID:        id,
			EntityID:  "FEAT-" + id[3:],
			Branch:    "feature/test",
			Path:      ".worktrees/test",
			Status:    StatusActive,
			Created:   time.Date(2025, 1, 27, 10+i, 0, 0, 0, time.UTC),
			CreatedBy: "user",
		}
		if _, err := store.Create(record); err != nil {
			t.Fatalf("Create() error = %v", err)
		}
	}

	// List should return all records sorted by ID
	records, err = store.List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(records) != 3 {
		t.Fatalf("List() returned %d records, want 3", len(records))
	}

	// Verify sorted order
	if records[0].ID != "WT-01JX000000001" {
		t.Errorf("List()[0].ID = %q, want %q", records[0].ID, "WT-01JX000000001")
	}
	if records[1].ID != "WT-01JX000000002" {
		t.Errorf("List()[1].ID = %q, want %q", records[1].ID, "WT-01JX000000002")
	}
	if records[2].ID != "WT-01JX000000003" {
		t.Errorf("List()[2].ID = %q, want %q", records[2].ID, "WT-01JX000000003")
	}
}

func TestStore_List_EmptyDirectory(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := NewStore(root)

	// Directory doesn't exist yet
	records, err := store.List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(records) != 0 {
		t.Errorf("List() returned %v, want nil or empty", records)
	}
}

func TestStore_Update(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := NewStore(root)

	original := Record{
		ID:        "WT-01JX123456789",
		EntityID:  "FEAT-01JX987654321",
		Branch:    "feature/FEAT-01JX987654321-user-profiles",
		Path:      ".worktrees/FEAT-01JX987654321-user-profiles",
		Status:    StatusActive,
		Created:   time.Date(2025, 1, 27, 10, 0, 0, 0, time.UTC),
		CreatedBy: "sambeau",
	}

	created, err := store.Create(original)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Update the status
	created.Status = StatusMerged
	mergedAt := time.Date(2025, 1, 28, 15, 30, 0, 0, time.UTC)
	created.MergedAt = &mergedAt

	updated, err := store.Update(created)
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	// Verify FileHash changed
	if updated.FileHash == created.FileHash {
		t.Error("Update() should update FileHash")
	}

	// Verify changes persisted
	loaded, err := store.Get(original.ID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if loaded.Status != StatusMerged {
		t.Errorf("Get() Status = %q, want %q", loaded.Status, StatusMerged)
	}
	if loaded.MergedAt == nil || !loaded.MergedAt.Equal(mergedAt) {
		t.Errorf("Get() MergedAt = %v, want %v", loaded.MergedAt, mergedAt)
	}
}

func TestStore_Update_OptimisticLock(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := NewStore(root)

	original := Record{
		ID:        "WT-01JX123456789",
		EntityID:  "FEAT-01JX987654321",
		Branch:    "feature/FEAT-01JX987654321-user-profiles",
		Path:      ".worktrees/FEAT-01JX987654321-user-profiles",
		Status:    StatusActive,
		Created:   time.Date(2025, 1, 27, 10, 0, 0, 0, time.UTC),
		CreatedBy: "sambeau",
	}

	created, err := store.Create(original)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Load the record again (simulates another process)
	loaded, err := store.Get(created.ID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	// Update the first copy
	created.Status = StatusMerged
	_, err = store.Update(created)
	if err != nil {
		t.Fatalf("Update(created) error = %v", err)
	}

	// Try to update the stale copy - should fail with ErrConflict
	loaded.Status = StatusAbandoned
	_, err = store.Update(loaded)
	if err == nil {
		t.Fatal("Update() with stale FileHash should return error")
	}
	if !errors.Is(err, ErrConflict) {
		t.Errorf("Update() error = %v, want ErrConflict", err)
	}
}

func TestStore_Delete(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := NewStore(root)

	record := Record{
		ID:        "WT-01JX123456789",
		EntityID:  "FEAT-01JX987654321",
		Branch:    "feature/test",
		Path:      ".worktrees/test",
		Status:    StatusActive,
		Created:   time.Date(2025, 1, 27, 10, 0, 0, 0, time.UTC),
		CreatedBy: "user",
	}

	if _, err := store.Create(record); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Delete the record
	if err := store.Delete(record.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	// Verify it's gone
	_, err := store.Get(record.ID)
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("Get() after Delete() error = %v, want ErrNotFound", err)
	}
}

func TestStore_Delete_NotFound(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := NewStore(root)

	// Delete non-existent record should not error
	err := store.Delete("WT-NONEXISTENT123")
	if err != nil {
		t.Errorf("Delete() non-existent record error = %v, want nil", err)
	}
}

func TestStore_Delete_EmptyID(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := NewStore(root)

	err := store.Delete("")
	if err == nil {
		t.Fatal("Delete() expected error, got nil")
	}
	if !strings.Contains(err.Error(), "ID is required") {
		t.Errorf("Delete() error = %q, want containing 'ID is required'", err.Error())
	}
}

func TestStore_YAMLRoundTrip(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := NewStore(root)

	mergedAt := time.Date(2025, 1, 28, 15, 30, 0, 0, time.UTC)
	cleanupAfter := time.Date(2025, 2, 28, 0, 0, 0, 0, time.UTC)

	original := Record{
		ID:           "WT-01JX123456789",
		EntityID:     "FEAT-01JX987654321",
		Branch:       "feature/FEAT-01JX987654321-user-profiles",
		Path:         ".worktrees/FEAT-01JX987654321-user-profiles",
		Status:       StatusMerged,
		Created:      time.Date(2025, 1, 27, 10, 0, 0, 0, time.UTC),
		CreatedBy:    "sambeau",
		MergedAt:     &mergedAt,
		CleanupAfter: &cleanupAfter,
	}

	// Write
	_, err := store.Create(original)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Read
	loaded, err := store.Get(original.ID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	// Compare all fields
	if loaded.ID != original.ID {
		t.Errorf("ID = %q, want %q", loaded.ID, original.ID)
	}
	if loaded.EntityID != original.EntityID {
		t.Errorf("EntityID = %q, want %q", loaded.EntityID, original.EntityID)
	}
	if loaded.Branch != original.Branch {
		t.Errorf("Branch = %q, want %q", loaded.Branch, original.Branch)
	}
	if loaded.Path != original.Path {
		t.Errorf("Path = %q, want %q", loaded.Path, original.Path)
	}
	if loaded.Status != original.Status {
		t.Errorf("Status = %q, want %q", loaded.Status, original.Status)
	}
	if !loaded.Created.Equal(original.Created) {
		t.Errorf("Created = %v, want %v", loaded.Created, original.Created)
	}
	if loaded.CreatedBy != original.CreatedBy {
		t.Errorf("CreatedBy = %q, want %q", loaded.CreatedBy, original.CreatedBy)
	}
	if loaded.MergedAt == nil || !loaded.MergedAt.Equal(*original.MergedAt) {
		t.Errorf("MergedAt = %v, want %v", loaded.MergedAt, original.MergedAt)
	}
	if loaded.CleanupAfter == nil || !loaded.CleanupAfter.Equal(*original.CleanupAfter) {
		t.Errorf("CleanupAfter = %v, want %v", loaded.CleanupAfter, original.CleanupAfter)
	}

	// Write again (with loaded record)
	loaded.Status = StatusAbandoned
	_, err = store.Update(loaded)
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	// Read again and verify idempotency
	reloaded, err := store.Get(original.ID)
	if err != nil {
		t.Fatalf("Get() after Update() error = %v", err)
	}
	if reloaded.Status != StatusAbandoned {
		t.Errorf("Status after update = %q, want %q", reloaded.Status, StatusAbandoned)
	}
}

func TestStore_YAMLFormat(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := NewStore(root)

	record := Record{
		ID:        "WT-01JX123456789",
		EntityID:  "FEAT-01JX987654321",
		Branch:    "feature/FEAT-01JX987654321-user-profiles",
		Path:      ".worktrees/FEAT-01JX987654321-user-profiles",
		Status:    StatusActive,
		Created:   time.Date(2025, 1, 27, 10, 0, 0, 0, time.UTC),
		CreatedBy: "sambeau",
	}

	if _, err := store.Create(record); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Read the raw file
	path := filepath.Join(root, WorktreesDir, record.ID+".yaml")
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	// Verify YAML format
	yaml := string(content)

	// Check field order (id should come first)
	idIdx := strings.Index(yaml, "id:")
	entityIdx := strings.Index(yaml, "entity_id:")
	branchIdx := strings.Index(yaml, "branch:")
	pathIdx := strings.Index(yaml, "path:")
	statusIdx := strings.Index(yaml, "status:")

	if idIdx < 0 || entityIdx < 0 || branchIdx < 0 || pathIdx < 0 || statusIdx < 0 {
		t.Fatalf("Missing expected fields in YAML:\n%s", yaml)
	}

	// Verify canonical order
	if !(idIdx < entityIdx && entityIdx < branchIdx && branchIdx < pathIdx && pathIdx < statusIdx) {
		t.Errorf("Fields not in canonical order:\n%s", yaml)
	}

	// Verify no trailing whitespace or extra blank lines
	lines := strings.Split(yaml, "\n")
	for i, line := range lines {
		if strings.TrimRight(line, " \t") != line {
			t.Errorf("Line %d has trailing whitespace: %q", i+1, line)
		}
	}
}
