package storage

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// validDocumentFields returns a minimal set of fields that pass document record validation.
func validDocumentFields(id string) map[string]any {
	return map[string]any{
		"id":           id,
		"path":         "work/design/test-doc.md",
		"type":         "design",
		"title":        "Test Document",
		"status":       "draft",
		"content_hash": "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
		"created":      "2026-03-19T00:00:00Z",
		"created_by":   "sam",
		"updated":      "2026-03-19T00:00:00Z",
	}
}

func TestDocumentStore_Load_PopulatesFileHash(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := NewDocumentStore(root)

	id := "FEAT-01HASH/hash-test"
	record := DocumentRecord{
		ID:     id,
		Fields: validDocumentFields(id),
	}

	if _, err := store.Write(record); err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	got, err := store.Load(id)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if got.FileHash == "" {
		t.Fatal("Load() FileHash is empty, expected SHA-256 hex digest")
	}
	if len(got.FileHash) != 64 {
		t.Fatalf("Load() FileHash length = %d, want 64 hex chars", len(got.FileHash))
	}
}

func TestDocumentStore_Write_SucceedsWithCorrectFileHash(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := NewDocumentStore(root)

	id := "FEAT-01LOCK/lock-ok"
	record := DocumentRecord{
		ID:     id,
		Fields: validDocumentFields(id),
	}

	if _, err := store.Write(record); err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	loaded, err := store.Load(id)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Modify a field and write back with the correct FileHash.
	loaded.Fields["title"] = "Updated Title"
	if _, err := store.Write(loaded); err != nil {
		t.Fatalf("Write() with correct FileHash error = %v", err)
	}
}

func TestDocumentStore_Write_ReturnsErrConflictOnStaleFileHash(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := NewDocumentStore(root)

	id := "FEAT-01STALE/lock-stale"
	record := DocumentRecord{
		ID:     id,
		Fields: validDocumentFields(id),
	}

	if _, err := store.Write(record); err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	loaded, err := store.Load(id)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Simulate a concurrent modification by writing directly.
	stale := DocumentRecord{
		ID:       loaded.ID,
		Fields:   copyFields(loaded.Fields),
		FileHash: loaded.FileHash,
	}
	loaded.Fields["title"] = "Concurrent update"
	loaded.FileHash = "" // bypass locking for the concurrent write
	if _, err := store.Write(loaded); err != nil {
		t.Fatalf("concurrent Write() error = %v", err)
	}

	// Now try to write with the stale hash.
	stale.Fields["title"] = "Late update"
	_, err = store.Write(stale)
	if err == nil {
		t.Fatal("Write() with stale FileHash should return error")
	}
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("Write() error = %v, want ErrConflict", err)
	}
}

func TestDocumentStore_Write_SucceedsWithEmptyFileHash(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := NewDocumentStore(root)

	id := "FEAT-01EMPTY/no-hash"
	record := DocumentRecord{
		ID:     id,
		Fields: validDocumentFields(id),
	}

	// First write — no FileHash, file doesn't exist yet.
	if _, err := store.Write(record); err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	// Second write — no FileHash, file already exists. Should succeed without checking.
	record.Fields["title"] = "Overwritten without hash"
	if _, err := store.Write(record); err != nil {
		t.Fatalf("Write() with empty FileHash on existing file error = %v", err)
	}
}

func TestDocumentStore_Write_NewDocumentWithFileHashSucceeds(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := NewDocumentStore(root)

	id := "FEAT-01NEW/brand-new"
	record := DocumentRecord{
		ID:       id,
		FileHash: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		Fields:   validDocumentFields(id),
	}

	// File doesn't exist, so the hash check should be skipped even though
	// FileHash is set.
	if _, err := store.Write(record); err != nil {
		t.Fatalf("Write() new document with FileHash error = %v", err)
	}

	// Verify the file was actually created.
	path := filepath.Join(root, DocumentDir, documentFileName(id))
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected file to exist at %s: %v", path, err)
	}
}

// copyFields creates a shallow copy of a fields map.
func copyFields(src map[string]any) map[string]any {
	dst := make(map[string]any, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}
