package checkpoint

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	dir := t.TempDir()
	return NewStore(dir)
}

func TestCreate(t *testing.T) {
	store := newTestStore(t)

	record := Record{
		Question:             "Should I prioritise cache invalidation or pagination?",
		Context:              "Both tasks are ready. Cache invalidation blocks TASK-002.",
		OrchestrationSummary: "Track A: 3/10 tasks complete.",
		Status:               StatusPending,
		CreatedAt:            time.Now().UTC().Truncate(time.Second),
		CreatedBy:            "orchestrator-session-abc",
	}

	created, err := store.Create(record)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if created.ID == "" {
		t.Error("expected non-empty ID")
	}
	if !strings.HasPrefix(created.ID, "CHK-") {
		t.Errorf("expected ID to start with CHK-, got %q", created.ID)
	}
	if created.Status != StatusPending {
		t.Errorf("expected status pending, got %q", created.Status)
	}
	if created.RespondedAt != nil {
		t.Error("expected responded_at to be nil for new checkpoint")
	}
	if created.Response != nil {
		t.Error("expected response to be nil for new checkpoint")
	}
	if created.FileHash == "" {
		t.Error("expected non-empty FileHash after create")
	}
}

func TestCreateRequiresQuestion(t *testing.T) {
	store := newTestStore(t)

	record := Record{
		Question:  "",
		Context:   "some context",
		Status:    StatusPending,
		CreatedAt: time.Now().UTC(),
		CreatedBy: "test-agent",
	}

	_, err := store.Create(record)
	if err == nil {
		t.Error("expected error for empty question, got nil")
	}
}

func TestGet(t *testing.T) {
	store := newTestStore(t)

	created, err := store.Create(Record{
		Question:             "Test question?",
		Context:              "Test context.",
		OrchestrationSummary: "Test summary.",
		Status:               StatusPending,
		CreatedAt:            time.Now().UTC().Truncate(time.Second),
		CreatedBy:            "agent-1",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := store.Get(created.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}

	if got.ID != created.ID {
		t.Errorf("ID: got %q, want %q", got.ID, created.ID)
	}
	if got.Question != created.Question {
		t.Errorf("Question: got %q, want %q", got.Question, created.Question)
	}
	if got.Status != StatusPending {
		t.Errorf("Status: got %q, want pending", got.Status)
	}
	if got.RespondedAt != nil {
		t.Error("RespondedAt should be nil for pending checkpoint")
	}
	if got.Response != nil {
		t.Error("Response should be nil for pending checkpoint")
	}
}

func TestGetNotFound(t *testing.T) {
	store := newTestStore(t)

	_, err := store.Get("CHK-NOTEXIST")
	if err == nil {
		t.Fatal("expected error for non-existent checkpoint")
	}
}

func TestUpdate_Respond(t *testing.T) {
	store := newTestStore(t)

	created, err := store.Create(Record{
		Question:             "Which task first?",
		Context:              "Context details.",
		OrchestrationSummary: "Summary here.",
		Status:               StatusPending,
		CreatedAt:            time.Now().UTC().Truncate(time.Second),
		CreatedBy:            "agent-1",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Respond to the checkpoint.
	now := time.Now().UTC().Truncate(time.Second)
	response := "Prioritise the cache task."
	created.Status = StatusResponded
	created.RespondedAt = &now
	created.Response = &response

	updated, err := store.Update(created)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}

	if updated.Status != StatusResponded {
		t.Errorf("Status: got %q, want responded", updated.Status)
	}
	if updated.RespondedAt == nil {
		t.Error("RespondedAt should not be nil after responding")
	}
	if updated.Response == nil || *updated.Response != response {
		t.Errorf("Response: got %v, want %q", updated.Response, response)
	}

	// Reload and verify persistence.
	reloaded, err := store.Get(created.ID)
	if err != nil {
		t.Fatalf("Get after update: %v", err)
	}
	if reloaded.Status != StatusResponded {
		t.Errorf("reloaded Status: got %q, want responded", reloaded.Status)
	}
	if reloaded.Response == nil || *reloaded.Response != response {
		t.Errorf("reloaded Response: got %v, want %q", reloaded.Response, response)
	}
}

func TestList(t *testing.T) {
	store := newTestStore(t)

	base := Record{
		Question:  "Q?",
		Context:   "C",
		Status:    StatusPending,
		CreatedAt: time.Now().UTC(),
		CreatedBy: "agent-1",
	}

	r1, _ := store.Create(base)
	r2, _ := store.Create(base)

	// Respond to r2.
	now := time.Now().UTC()
	resp := "Answer."
	r2.Status = StatusResponded
	r2.RespondedAt = &now
	r2.Response = &resp
	store.Update(r2) //nolint:errcheck

	// List all.
	all, err := store.List("")
	if err != nil {
		t.Fatalf("List all: %v", err)
	}
	if len(all) != 2 {
		t.Errorf("expected 2 checkpoints, got %d", len(all))
	}

	// List pending only.
	pending, err := store.List("pending")
	if err != nil {
		t.Fatalf("List pending: %v", err)
	}
	if len(pending) != 1 {
		t.Errorf("expected 1 pending checkpoint, got %d", len(pending))
	}
	if pending[0].ID != r1.ID {
		t.Errorf("expected pending checkpoint ID %q, got %q", r1.ID, pending[0].ID)
	}

	// List responded only.
	responded, err := store.List("responded")
	if err != nil {
		t.Fatalf("List responded: %v", err)
	}
	if len(responded) != 1 {
		t.Errorf("expected 1 responded checkpoint, got %d", len(responded))
	}
	if responded[0].ID != r2.ID {
		t.Errorf("expected responded checkpoint ID %q, got %q", r2.ID, responded[0].ID)
	}
}

func TestListEmpty(t *testing.T) {
	store := newTestStore(t)

	records, err := store.List("")
	if err != nil {
		t.Fatalf("List on empty store: %v", err)
	}
	if records != nil && len(records) != 0 {
		t.Errorf("expected empty list, got %d records", len(records))
	}
}

func TestYAMLRoundTrip_ExplicitNulls(t *testing.T) {
	// Create a checkpoint and verify the YAML contains explicit null for
	// responded_at and response (§11.5 of spec: explicit nulls required).
	store := newTestStore(t)

	created, err := store.Create(Record{
		Question:             "Round-trip test?",
		Context:              "Context for round-trip.",
		OrchestrationSummary: "Summary.",
		Status:               StatusPending,
		CreatedAt:            time.Date(2026, 3, 25, 10, 0, 0, 0, time.UTC),
		CreatedBy:            "test-agent",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Read the YAML file directly.
	path := filepath.Join(store.dir(), created.ID+".yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read YAML file: %v", err)
	}

	content := string(data)

	if !strings.Contains(content, "responded_at: null") {
		t.Errorf("expected 'responded_at: null' in YAML; got:\n%s", content)
	}
	if !strings.Contains(content, "response: null") {
		t.Errorf("expected 'response: null' in YAML; got:\n%s", content)
	}
}

func TestYAMLRoundTrip_AfterResponse(t *testing.T) {
	store := newTestStore(t)

	created, err := store.Create(Record{
		Question:  "Q?",
		Context:   "C",
		Status:    StatusPending,
		CreatedAt: time.Date(2026, 3, 25, 10, 0, 0, 0, time.UTC),
		CreatedBy: "agent",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	respondedAt := time.Date(2026, 3, 25, 11, 0, 0, 0, time.UTC)
	resp := "Do the cache task first."
	created.Status = StatusResponded
	created.RespondedAt = &respondedAt
	created.Response = &resp

	updated, err := store.Update(created)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}

	// Reload and compare.
	reloaded, err := store.Get(updated.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}

	if reloaded.Status != StatusResponded {
		t.Errorf("Status: got %q", reloaded.Status)
	}
	if reloaded.RespondedAt == nil {
		t.Error("RespondedAt is nil after round-trip")
	} else if !reloaded.RespondedAt.Equal(respondedAt) {
		t.Errorf("RespondedAt: got %v, want %v", reloaded.RespondedAt, respondedAt)
	}
	if reloaded.Response == nil || *reloaded.Response != resp {
		t.Errorf("Response: got %v, want %q", reloaded.Response, resp)
	}
}

func TestFieldOrderInYAML(t *testing.T) {
	// Verifies the canonical field order: id, question, context, orchestration_summary,
	// status, created_at, created_by, responded_at, response (§11.5 of spec).
	store := newTestStore(t)

	created, err := store.Create(Record{
		Question:             "Field order test?",
		Context:              "Context.",
		OrchestrationSummary: "Summary.",
		Status:               StatusPending,
		CreatedAt:            time.Date(2026, 3, 25, 10, 0, 0, 0, time.UTC),
		CreatedBy:            "agent",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	path := filepath.Join(store.dir(), created.ID+".yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read file: %v", err)
	}

	wantOrder := []string{
		"id:",
		"question:",
		"context:",
		"orchestration_summary:",
		"status:",
		"created_at:",
		"created_by:",
		"responded_at:",
		"response:",
	}

	content := string(data)
	lastIdx := -1
	for _, key := range wantOrder {
		idx := strings.Index(content, "\n"+key)
		if idx == -1 {
			// Try at the start of the file (first line).
			if strings.HasPrefix(content, key) {
				idx = 0
			} else {
				t.Errorf("key %q not found in YAML", key)
				continue
			}
		}
		if idx < lastIdx {
			t.Errorf("key %q appears before expected position (after idx %d)", key, lastIdx)
		}
		lastIdx = idx
	}
}
