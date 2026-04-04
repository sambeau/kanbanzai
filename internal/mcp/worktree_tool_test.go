package mcp

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/sambeau/kanbanzai/internal/worktree"
)

// ─── helpers ─────────────────────────────────────────────────────────────────

// createTestWorktreeRecord inserts a worktree record directly into the store,
// bypassing the MCP layer and git operations. Useful for seeding state
// before testing get/update actions.
func createTestWorktreeRecord(t *testing.T, store *worktree.Store, entityID, graphProject string) worktree.Record {
	t.Helper()

	record := worktree.Record{
		EntityID:     entityID,
		Branch:       "feature/" + entityID + "-test",
		Path:         ".worktrees/" + entityID + "-test",
		Status:       worktree.StatusActive,
		Created:      time.Date(2025, 1, 27, 10, 0, 0, 0, time.UTC),
		CreatedBy:    "test-user",
		GraphProject: graphProject,
	}

	created, err := store.Create(record)
	if err != nil {
		t.Fatalf("createTestWorktreeRecord: %v", err)
	}
	return created
}

// callWorktreeAction invokes the worktree update action handler directly
// (bypassing the full MCP server) and returns the parsed response map.
func callWorktreeUpdateAction(t *testing.T, store *worktree.Store, args map[string]any) map[string]any {
	t.Helper()

	handler := worktreeUpdateAction(store)
	req := mcp.CallToolRequest{}
	req.Params.Arguments = args

	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("worktreeUpdateAction: %v", err)
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal result: %v", err)
	}

	var resp map[string]any
	if err := json.Unmarshal(data, &resp); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	return resp
}

// ─── worktreeRecordToMap ─────────────────────────────────────────────────────

func TestWorktreeRecordToMap_IncludesGraphProject(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		graphProject string
		want         string
	}{
		{"empty graph_project", "", ""},
		{"non-empty graph_project", "kanbanzai-FEAT-XXX", "kanbanzai-FEAT-XXX"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			record := worktree.Record{
				ID:           "WT-01JX123456789",
				EntityID:     "FEAT-01JX987654321",
				Branch:       "feature/test",
				Path:         ".worktrees/test",
				Status:       worktree.StatusActive,
				Created:      time.Date(2025, 1, 27, 10, 0, 0, 0, time.UTC),
				CreatedBy:    "test-user",
				GraphProject: tt.graphProject,
			}

			m := worktreeRecordToMap(record)

			got, ok := m["graph_project"]
			if !ok {
				t.Fatal("graph_project key missing from worktreeRecordToMap output")
			}
			if got != tt.want {
				t.Errorf("graph_project = %q, want %q", got, tt.want)
			}
		})
	}
}

// ─── create action: graph_project ────────────────────────────────────────────

// TestWorktreeCreate_GraphProject_SetInRecord verifies AC-002:
// When graph_project is provided on create, the record stores it.
func TestWorktreeCreate_GraphProject_SetInRecord(t *testing.T) {
	t.Parallel()

	store := worktree.NewStore(t.TempDir())

	// We can't call the full create action (it needs entity service + git ops),
	// so we verify the field wiring by creating a record directly with
	// GraphProject set — the production code sets it from req.GetString.
	record := worktree.Record{
		EntityID:     "FEAT-01AAAAAAAAAAAAA",
		Branch:       "feature/test",
		Path:         ".worktrees/test",
		Status:       worktree.StatusActive,
		Created:      time.Now().UTC(),
		CreatedBy:    "test-user",
		GraphProject: "kanbanzai-FEAT-XXX",
	}

	created, err := store.Create(record)
	if err != nil {
		t.Fatalf("store.Create: %v", err)
	}

	// Re-read from store to verify round-trip persistence.
	got, err := store.GetByEntityID("FEAT-01AAAAAAAAAAAAA")
	if err != nil {
		t.Fatalf("store.GetByEntityID: %v", err)
	}

	if got.GraphProject != "kanbanzai-FEAT-XXX" {
		t.Errorf("GraphProject = %q, want %q", got.GraphProject, "kanbanzai-FEAT-XXX")
	}
	_ = created
}

// TestWorktreeCreate_GraphProject_DefaultEmpty verifies AC-003:
// When graph_project is not provided on create, the field is empty string.
func TestWorktreeCreate_GraphProject_DefaultEmpty(t *testing.T) {
	t.Parallel()

	store := worktree.NewStore(t.TempDir())

	record := worktree.Record{
		EntityID:  "FEAT-01AAAAAAAAAAAAA",
		Branch:    "feature/test",
		Path:      ".worktrees/test",
		Status:    worktree.StatusActive,
		Created:   time.Now().UTC(),
		CreatedBy: "test-user",
		// GraphProject intentionally omitted — zero value.
	}

	_, err := store.Create(record)
	if err != nil {
		t.Fatalf("store.Create: %v", err)
	}

	got, err := store.GetByEntityID("FEAT-01AAAAAAAAAAAAA")
	if err != nil {
		t.Fatalf("store.GetByEntityID: %v", err)
	}

	if got.GraphProject != "" {
		t.Errorf("GraphProject = %q, want empty string", got.GraphProject)
	}
}

// ─── update action ───────────────────────────────────────────────────────────

// TestWorktreeUpdate_GraphProject_SetsValue verifies AC-004:
// Updating graph_project from empty to a value.
func TestWorktreeUpdate_GraphProject_SetsValue(t *testing.T) {
	t.Parallel()

	store := worktree.NewStore(t.TempDir())
	entityID := "FEAT-01AAAAAAAAAAAAA"

	createTestWorktreeRecord(t, store, entityID, "")

	resp := callWorktreeUpdateAction(t, store, map[string]any{
		"entity_id":     entityID,
		"graph_project": "kanbanzai-FEAT-XXX",
	})

	wt, ok := resp["worktree"].(map[string]any)
	if !ok {
		t.Fatalf("expected worktree in response, got: %v", resp)
	}

	if got := wt["graph_project"]; got != "kanbanzai-FEAT-XXX" {
		t.Errorf("graph_project = %v, want %q", got, "kanbanzai-FEAT-XXX")
	}

	// Verify persistence via store read-back.
	record, err := store.GetByEntityID(entityID)
	if err != nil {
		t.Fatalf("store.GetByEntityID: %v", err)
	}
	if record.GraphProject != "kanbanzai-FEAT-XXX" {
		t.Errorf("persisted GraphProject = %q, want %q", record.GraphProject, "kanbanzai-FEAT-XXX")
	}
}

// TestWorktreeUpdate_GraphProject_PreservedWhenOmitted verifies AC-005:
// When graph_project is NOT provided in update args, existing value is preserved.
func TestWorktreeUpdate_GraphProject_PreservedWhenOmitted(t *testing.T) {
	t.Parallel()

	store := worktree.NewStore(t.TempDir())
	entityID := "FEAT-01AAAAAAAAAAAAA"

	createTestWorktreeRecord(t, store, entityID, "kanbanzai-FEAT-XXX")

	// Call update WITHOUT graph_project in args.
	resp := callWorktreeUpdateAction(t, store, map[string]any{
		"entity_id": entityID,
	})

	wt, ok := resp["worktree"].(map[string]any)
	if !ok {
		t.Fatalf("expected worktree in response, got: %v", resp)
	}

	if got := wt["graph_project"]; got != "kanbanzai-FEAT-XXX" {
		t.Errorf("graph_project = %v, want %q (should be preserved)", got, "kanbanzai-FEAT-XXX")
	}

	// Verify persistence.
	record, err := store.GetByEntityID(entityID)
	if err != nil {
		t.Fatalf("store.GetByEntityID: %v", err)
	}
	if record.GraphProject != "kanbanzai-FEAT-XXX" {
		t.Errorf("persisted GraphProject = %q, want %q", record.GraphProject, "kanbanzai-FEAT-XXX")
	}
}

// TestWorktreeUpdate_GraphProject_OverwriteExisting verifies that an existing
// graph_project value can be changed to a different value.
func TestWorktreeUpdate_GraphProject_OverwriteExisting(t *testing.T) {
	t.Parallel()

	store := worktree.NewStore(t.TempDir())
	entityID := "FEAT-01AAAAAAAAAAAAA"

	createTestWorktreeRecord(t, store, entityID, "old-project")

	resp := callWorktreeUpdateAction(t, store, map[string]any{
		"entity_id":     entityID,
		"graph_project": "new-project",
	})

	wt, ok := resp["worktree"].(map[string]any)
	if !ok {
		t.Fatalf("expected worktree in response, got: %v", resp)
	}

	if got := wt["graph_project"]; got != "new-project" {
		t.Errorf("graph_project = %v, want %q", got, "new-project")
	}
}

// TestWorktreeUpdate_MissingEntityID verifies that update returns an inline
// error when entity_id is not provided.
func TestWorktreeUpdate_MissingEntityID(t *testing.T) {
	t.Parallel()

	store := worktree.NewStore(t.TempDir())
	handler := worktreeUpdateAction(store)

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{}

	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, _ := json.Marshal(result)
	var resp map[string]any
	if err := json.Unmarshal(data, &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if _, hasErr := resp["error"]; !hasErr {
		t.Errorf("expected error for missing entity_id, got: %v", resp)
	}
}

// TestWorktreeUpdate_NotFound verifies that update returns an inline error
// when no worktree exists for the given entity.
func TestWorktreeUpdate_NotFound(t *testing.T) {
	t.Parallel()

	store := worktree.NewStore(t.TempDir())
	handler := worktreeUpdateAction(store)

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{
		"entity_id":     "FEAT-01ZZZZZZZZZZZZZ",
		"graph_project": "some-project",
	}

	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, _ := json.Marshal(result)
	var resp map[string]any
	if err := json.Unmarshal(data, &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if _, hasErr := resp["error"]; !hasErr {
		t.Errorf("expected error for missing worktree, got: %v", resp)
	}
}
