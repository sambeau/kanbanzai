package service

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"kanbanzai/internal/model"
	"kanbanzai/internal/storage"
)

func writeTestDocument(t *testing.T, stateRoot, repoRoot string, doc model.DocumentRecord) {
	t.Helper()

	// Write the actual file so content hash works.
	fullPath := filepath.Join(repoRoot, doc.Path)
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		t.Fatalf("create doc dir: %v", err)
	}
	if err := os.WriteFile(fullPath, []byte("# "+doc.Title+"\n"), 0o644); err != nil {
		t.Fatalf("write doc file: %v", err)
	}

	hash, err := storage.ComputeContentHash(fullPath)
	if err != nil {
		t.Fatalf("compute hash: %v", err)
	}
	doc.ContentHash = hash

	store := storage.NewDocumentStore(stateRoot)
	record := storage.DocumentToRecord(doc)
	if _, err := store.Write(record); err != nil {
		t.Fatalf("write doc record %s: %v", doc.ID, err)
	}
}

func newTestDocService(stateRoot, repoRoot string) *DocumentService {
	svc := NewDocumentService(stateRoot, repoRoot)
	svc.now = func() time.Time {
		return time.Date(2026, 3, 19, 12, 0, 0, 0, time.UTC)
	}
	return svc
}

func TestSupersessionChain_SingleDocument(t *testing.T) {
	t.Parallel()

	stateRoot := t.TempDir()
	repoRoot := t.TempDir()
	svc := newTestDocService(stateRoot, repoRoot)

	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	writeTestDocument(t, stateRoot, repoRoot, model.DocumentRecord{
		ID:          "FEAT-123/design-v1",
		Path:        "work/design/v1.md",
		Type:        model.DocumentTypeDesign,
		Title:       "Design v1",
		Status:      model.DocumentStatusDraft,
		ContentHash: "",
		Created:     now,
		CreatedBy:   "tester",
		Updated:     now,
	})

	chain, err := svc.SupersessionChain("FEAT-123/design-v1")
	if err != nil {
		t.Fatalf("SupersessionChain() error = %v", err)
	}

	if len(chain) != 1 {
		t.Fatalf("expected chain length 1, got %d", len(chain))
	}
	if chain[0].ID != "FEAT-123/design-v1" {
		t.Errorf("chain[0].ID = %q, want %q", chain[0].ID, "FEAT-123/design-v1")
	}
}

func TestSupersessionChain_LinearChain(t *testing.T) {
	t.Parallel()

	stateRoot := t.TempDir()
	repoRoot := t.TempDir()
	svc := newTestDocService(stateRoot, repoRoot)

	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	// v1 superseded by v2
	writeTestDocument(t, stateRoot, repoRoot, model.DocumentRecord{
		ID:           "FEAT-A/design-v1",
		Path:         "work/design/a-v1.md",
		Type:         model.DocumentTypeDesign,
		Title:        "Design v1",
		Status:       model.DocumentStatusSuperseded,
		SupersededBy: "FEAT-A/design-v2",
		Created:      now,
		CreatedBy:    "tester",
		Updated:      now,
	})

	// v2 supersedes v1, superseded by v3
	writeTestDocument(t, stateRoot, repoRoot, model.DocumentRecord{
		ID:           "FEAT-A/design-v2",
		Path:         "work/design/a-v2.md",
		Type:         model.DocumentTypeDesign,
		Title:        "Design v2",
		Status:       model.DocumentStatusSuperseded,
		Supersedes:   "FEAT-A/design-v1",
		SupersededBy: "FEAT-A/design-v3",
		Created:      now,
		CreatedBy:    "tester",
		Updated:      now,
	})

	// v3 supersedes v2
	writeTestDocument(t, stateRoot, repoRoot, model.DocumentRecord{
		ID:         "FEAT-A/design-v3",
		Path:       "work/design/a-v3.md",
		Type:       model.DocumentTypeDesign,
		Title:      "Design v3",
		Status:     model.DocumentStatusDraft,
		Supersedes: "FEAT-A/design-v2",
		Created:    now,
		CreatedBy:  "tester",
		Updated:    now,
	})

	// Start from the middle (v2) — should get the full chain.
	chain, err := svc.SupersessionChain("FEAT-A/design-v2")
	if err != nil {
		t.Fatalf("SupersessionChain() error = %v", err)
	}

	if len(chain) != 3 {
		t.Fatalf("expected chain length 3, got %d", len(chain))
	}

	wantOrder := []string{"FEAT-A/design-v1", "FEAT-A/design-v2", "FEAT-A/design-v3"}
	for i, want := range wantOrder {
		if chain[i].ID != want {
			t.Errorf("chain[%d].ID = %q, want %q", i, chain[i].ID, want)
		}
	}
}

func TestSupersessionChain_StartFromOldest(t *testing.T) {
	t.Parallel()

	stateRoot := t.TempDir()
	repoRoot := t.TempDir()
	svc := newTestDocService(stateRoot, repoRoot)

	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	writeTestDocument(t, stateRoot, repoRoot, model.DocumentRecord{
		ID:           "FEAT-B/spec-v1",
		Path:         "work/spec/b-v1.md",
		Type:         model.DocumentTypeSpecification,
		Title:        "Spec v1",
		Status:       model.DocumentStatusSuperseded,
		SupersededBy: "FEAT-B/spec-v2",
		Created:      now,
		CreatedBy:    "tester",
		Updated:      now,
	})

	writeTestDocument(t, stateRoot, repoRoot, model.DocumentRecord{
		ID:         "FEAT-B/spec-v2",
		Path:       "work/spec/b-v2.md",
		Type:       model.DocumentTypeSpecification,
		Title:      "Spec v2",
		Status:     model.DocumentStatusDraft,
		Supersedes: "FEAT-B/spec-v1",
		Created:    now,
		CreatedBy:  "tester",
		Updated:    now,
	})

	// Start from the oldest (v1).
	chain, err := svc.SupersessionChain("FEAT-B/spec-v1")
	if err != nil {
		t.Fatalf("SupersessionChain() error = %v", err)
	}

	if len(chain) != 2 {
		t.Fatalf("expected chain length 2, got %d", len(chain))
	}
	if chain[0].ID != "FEAT-B/spec-v1" {
		t.Errorf("chain[0].ID = %q, want oldest", chain[0].ID)
	}
	if chain[1].ID != "FEAT-B/spec-v2" {
		t.Errorf("chain[1].ID = %q, want newest", chain[1].ID)
	}
}

func TestSupersessionChain_StartFromNewest(t *testing.T) {
	t.Parallel()

	stateRoot := t.TempDir()
	repoRoot := t.TempDir()
	svc := newTestDocService(stateRoot, repoRoot)

	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	writeTestDocument(t, stateRoot, repoRoot, model.DocumentRecord{
		ID:           "FEAT-C/dp-v1",
		Path:         "work/dp/c-v1.md",
		Type:         model.DocumentTypeDevPlan,
		Title:        "Dev Plan v1",
		Status:       model.DocumentStatusSuperseded,
		SupersededBy: "FEAT-C/dp-v2",
		Created:      now,
		CreatedBy:    "tester",
		Updated:      now,
	})

	writeTestDocument(t, stateRoot, repoRoot, model.DocumentRecord{
		ID:         "FEAT-C/dp-v2",
		Path:       "work/dp/c-v2.md",
		Type:       model.DocumentTypeDevPlan,
		Title:      "Dev Plan v2",
		Status:     model.DocumentStatusDraft,
		Supersedes: "FEAT-C/dp-v1",
		Created:    now,
		CreatedBy:  "tester",
		Updated:    now,
	})

	// Start from the newest (v2).
	chain, err := svc.SupersessionChain("FEAT-C/dp-v2")
	if err != nil {
		t.Fatalf("SupersessionChain() error = %v", err)
	}

	if len(chain) != 2 {
		t.Fatalf("expected chain length 2, got %d", len(chain))
	}
	if chain[0].ID != "FEAT-C/dp-v1" {
		t.Errorf("chain[0].ID = %q, want oldest", chain[0].ID)
	}
	if chain[1].ID != "FEAT-C/dp-v2" {
		t.Errorf("chain[1].ID = %q, want newest", chain[1].ID)
	}
}

func TestSupersessionChain_BrokenLink(t *testing.T) {
	t.Parallel()

	stateRoot := t.TempDir()
	repoRoot := t.TempDir()
	svc := newTestDocService(stateRoot, repoRoot)

	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	// v2 claims to supersede v1, but v1 doesn't exist.
	writeTestDocument(t, stateRoot, repoRoot, model.DocumentRecord{
		ID:         "FEAT-D/res-v2",
		Path:       "work/res/d-v2.md",
		Type:       model.DocumentTypeResearch,
		Title:      "Research v2",
		Status:     model.DocumentStatusDraft,
		Supersedes: "FEAT-D/res-v1",
		Created:    now,
		CreatedBy:  "tester",
		Updated:    now,
	})

	chain, err := svc.SupersessionChain("FEAT-D/res-v2")
	if err != nil {
		t.Fatalf("SupersessionChain() error = %v", err)
	}

	// Should stop at the broken link and just return v2.
	if len(chain) != 1 {
		t.Fatalf("expected chain length 1 (broken backward link), got %d", len(chain))
	}
	if chain[0].ID != "FEAT-D/res-v2" {
		t.Errorf("chain[0].ID = %q, want %q", chain[0].ID, "FEAT-D/res-v2")
	}
}

func TestSupersessionChain_EmptyID(t *testing.T) {
	t.Parallel()

	stateRoot := t.TempDir()
	repoRoot := t.TempDir()
	svc := newTestDocService(stateRoot, repoRoot)

	_, err := svc.SupersessionChain("")
	if err == nil {
		t.Fatal("expected error for empty document ID")
	}
}

func TestSupersessionChain_NonexistentDocument(t *testing.T) {
	t.Parallel()

	stateRoot := t.TempDir()
	repoRoot := t.TempDir()
	svc := newTestDocService(stateRoot, repoRoot)

	_, err := svc.SupersessionChain("FEAT-X/does-not-exist")
	if err == nil {
		t.Fatal("expected error for nonexistent document")
	}
}
