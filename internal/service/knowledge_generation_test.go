package service

import (
	"strings"
	"testing"
)

// ─────────────────────────────────────────────────────────────────────────────
// Generation (REQ-001 / AC-001)
// ─────────────────────────────────────────────────────────────────────────────

func TestKnowledgeService_Generation_EmptyDirectory(t *testing.T) {
	t.Parallel()
	svc := newTestKnowledgeService(t)

	// Knowledge directory does not exist yet — should return "0/0".
	gen, err := svc.Generation()
	if err != nil {
		t.Fatalf("Generation() error = %v", err)
	}
	if gen != "0/0" {
		t.Errorf("Generation() = %q, want \"0/0\" when directory is absent", gen)
	}
}

func TestKnowledgeService_Generation_TokenFormat(t *testing.T) {
	t.Parallel()
	svc := newTestKnowledgeService(t)

	// Seed one entry so the directory is created.
	if _, _, err := svc.Contribute(ContributeInput{
		Topic:   "generation-format-test",
		Content: "content",
		Scope:   "project",
		Tier:    3,
	}); err != nil {
		t.Fatalf("Contribute() error = %v", err)
	}

	gen, err := svc.Generation()
	if err != nil {
		t.Fatalf("Generation() error = %v", err)
	}
	// Token must be "<mtime_ns>/<count>" — two parts separated by "/".
	parts := strings.SplitN(gen, "/", 2)
	if len(parts) != 2 {
		t.Fatalf("Generation() = %q: expected \"<ns>/<count>\" format", gen)
	}
	if parts[0] == "0" && parts[1] == "0" {
		t.Errorf("Generation() = %q: non-empty directory should not return 0/0", gen)
	}
}

func TestKnowledgeService_Generation_ChangesAfterContribute(t *testing.T) {
	t.Parallel()
	svc := newTestKnowledgeService(t)

	// Before any entries.
	gen0, err := svc.Generation()
	if err != nil {
		t.Fatalf("Generation() before contribute: %v", err)
	}

	// Add first entry.
	if _, _, err := svc.Contribute(ContributeInput{
		Topic:   "gen-change-test-1",
		Content: "content 1",
		Scope:   "project",
		Tier:    3,
	}); err != nil {
		t.Fatalf("Contribute #1 error = %v", err)
	}

	gen1, err := svc.Generation()
	if err != nil {
		t.Fatalf("Generation() after contribute #1: %v", err)
	}

	// Generation must change after adding an entry.
	if gen1 == gen0 {
		t.Errorf("Generation did not change after contribute: before=%q after=%q", gen0, gen1)
	}

	// Add a second entry.
	if _, _, err := svc.Contribute(ContributeInput{
		Topic:   "gen-change-test-2",
		Content: "content 2",
		Scope:   "project",
		Tier:    3,
	}); err != nil {
		t.Fatalf("Contribute #2 error = %v", err)
	}

	gen2, err := svc.Generation()
	if err != nil {
		t.Fatalf("Generation() after contribute #2: %v", err)
	}

	if gen2 == gen1 {
		t.Errorf("Generation did not change after second contribute: before=%q after=%q", gen1, gen2)
	}
}

func TestKnowledgeService_Generation_CountReflectsFileCount(t *testing.T) {
	t.Parallel()
	svc := newTestKnowledgeService(t)

	contributions := []struct {
		topic   string
		content string
	}{
		{"gen-count-alpha", "Always use transactions when updating multiple related records in the same datastore to maintain consistency."},
		{"gen-count-bravo", "Prefer table-driven tests because they make it trivial to add new test cases without duplicating boilerplate setup."},
		{"gen-count-charlie", "Use context propagation through all service calls so cancellation signals are correctly forwarded to downstream dependencies."},
	}
	for i, c := range contributions {
		if _, _, err := svc.Contribute(ContributeInput{
			Topic:   c.topic,
			Content: c.content,
			Scope:   "project",
			Tier:    3,
		}); err != nil {
			t.Fatalf("Contribute #%d error = %v", i, err)
		}
	}

	gen, err := svc.Generation()
	if err != nil {
		t.Fatalf("Generation() error = %v", err)
	}

	parts := strings.SplitN(gen, "/", 2)
	if len(parts) != 2 {
		t.Fatalf("Generation() = %q: expected \"<ns>/<count>\" format", gen)
	}
	if parts[1] != "3" {
		t.Errorf("Generation() count part = %q, want \"3\" for %d entries", parts[1], len(contributions))
	}
}

func TestKnowledgeService_Generation_StableForSameState(t *testing.T) {
	t.Parallel()
	svc := newTestKnowledgeService(t)

	if _, _, err := svc.Contribute(ContributeInput{
		Topic:   "stable-gen-test",
		Content: "content",
		Scope:   "project",
		Tier:    3,
	}); err != nil {
		t.Fatalf("Contribute error = %v", err)
	}

	gen1, err := svc.Generation()
	if err != nil {
		t.Fatalf("Generation() #1 error = %v", err)
	}
	gen2, err := svc.Generation()
	if err != nil {
		t.Fatalf("Generation() #2 error = %v", err)
	}

	// Two consecutive calls with no modification must return the same token.
	if gen1 != gen2 {
		t.Errorf("Generation() not stable: %q vs %q", gen1, gen2)
	}
}
