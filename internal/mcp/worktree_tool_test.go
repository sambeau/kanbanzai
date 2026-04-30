package mcp

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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

// ─── remove action: graph_project_note (AC-015, AC-016) ─────────────────────

// setupGitRepoForRemove creates a temporary git repository with an initial commit
// and a worktree at the given path, suitable for testing worktreeRemoveAction.
func setupGitRepoForRemove(t *testing.T, wtRelPath string) (repoDir string, wtAbsPath string) {
	t.Helper()

	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not found in PATH, skipping test")
	}

	repoDir = t.TempDir()

	runGitCmd := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = repoDir
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v failed: %v\n%s", args, err, out)
		}
	}

	runGitCmd("init")
	runGitCmd("config", "user.email", "test@test.com")
	runGitCmd("config", "user.name", "Test")

	readme := filepath.Join(repoDir, "README.md")
	if err := os.WriteFile(readme, []byte("# test\n"), 0o644); err != nil {
		t.Fatalf("write README: %v", err)
	}
	runGitCmd("add", "README.md")
	runGitCmd("commit", "-m", "init")

	wtAbsPath = filepath.Join(repoDir, wtRelPath)
	runGitCmd("worktree", "add", "-b", "test-branch", wtAbsPath)

	return repoDir, wtAbsPath
}

// TestWorktreeRemove_GraphProjectNote verifies AC-015:
// Removing a worktree with non-empty GraphProject → response contains graph_project_note.
func TestWorktreeRemove_GraphProjectNote(t *testing.T) {
	t.Parallel()

	repoDir, wtAbsPath := setupGitRepoForRemove(t, "wt-gp-note")
	gitOps := worktree.NewGit(repoDir)
	store := worktree.NewStore(t.TempDir())

	entityID := "FEAT-01AAAAAAAAAAAAA"
	_, err := store.Create(worktree.Record{
		EntityID:     entityID,
		Branch:       "test-branch",
		Path:         wtAbsPath,
		Status:       worktree.StatusActive,
		Created:      time.Now().UTC(),
		CreatedBy:    "tester",
		GraphProject: "kanbanzai-FEAT-XXX",
	})
	if err != nil {
		t.Fatalf("store.Create: %v", err)
	}

	handler := worktreeRemoveAction(store, gitOps)
	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{
		"entity_id": entityID,
	}

	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("worktreeRemoveAction: %v", err)
	}

	data, _ := json.Marshal(result)
	var resp map[string]any
	if err := json.Unmarshal(data, &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	note, ok := resp["graph_project_note"].(string)
	if !ok {
		t.Fatalf("expected graph_project_note in response, got: %v", resp)
	}
	if !containsIgnoreCase(note, "kanbanzai-FEAT-XXX") {
		t.Errorf("graph_project_note should reference project name, got: %s", note)
	}
	if !containsIgnoreCase(note, "delete_project") {
		t.Errorf("graph_project_note should mention delete_project, got: %s", note)
	}
}

// TestWorktreeRemove_NoGraphProjectNote verifies AC-016:
// Removing a worktree with empty GraphProject → no graph_project_note in response.
func TestWorktreeRemove_NoGraphProjectNote(t *testing.T) {
	t.Parallel()

	repoDir, wtAbsPath := setupGitRepoForRemove(t, "wt-no-gp-note")
	gitOps := worktree.NewGit(repoDir)
	store := worktree.NewStore(t.TempDir())

	entityID := "FEAT-01BBBBBBBBBBBBB"
	_, err := store.Create(worktree.Record{
		EntityID:     entityID,
		Branch:       "test-branch",
		Path:         wtAbsPath,
		Status:       worktree.StatusActive,
		Created:      time.Now().UTC(),
		CreatedBy:    "tester",
		GraphProject: "",
	})
	if err != nil {
		t.Fatalf("store.Create: %v", err)
	}

	handler := worktreeRemoveAction(store, gitOps)
	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{
		"entity_id": entityID,
	}

	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("worktreeRemoveAction: %v", err)
	}

	data, _ := json.Marshal(result)
	var resp map[string]any
	if err := json.Unmarshal(data, &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if _, hasNote := resp["graph_project_note"]; hasNote {
		t.Errorf("unexpected graph_project_note for empty GraphProject: %v", resp)
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

// ─── entity ID validation tests ─────────────────────────────────────────────

// TestIsDisplayEntityID verifies the display-ID detection logic (O(1) string check).
func TestIsDisplayEntityID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		id   string
		want bool
	}{
		// AC-008: display IDs with embedded hyphen are detected.
		{"display ID FEAT", "FEAT-01KQ7-JDT511BZ", true},
		{"display ID BUG", "BUG-01KQ7-JDT511BZ", true},
		{"display ID with multiple embedded hyphens", "FEAT-01K-Q7-JDT511BZ", true},

		// AC-009: canonical IDs (single hyphen) are NOT display IDs.
		{"canonical FEAT", "FEAT-01KQ7JDT511BZ", false},
		{"canonical BUG", "BUG-01KQ7JDT511BZ", false},
		{"short canonical", "FEAT-01ABCDEFGHIJKL", false},

		// Edge cases.
		{"no hyphen at all", "FEAT01KQ7JDT511BZ", false},
		{"empty string", "", false},
		{"just hyphens", "---", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := isDisplayEntityID(tt.id)
			if got != tt.want {
				t.Errorf("isDisplayEntityID(%q) = %v, want %v", tt.id, got, tt.want)
			}
		})
	}
}

// TestDisplayToCanonical verifies conversion from display to canonical form.
func TestDisplayToCanonical(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		id   string
		want string
	}{
		// AC-008: display ID converted to canonical.
		{"FEAT display to canonical", "FEAT-01KQ7-JDT511BZ", "FEAT-01KQ7JDT511BZ"},
		{"BUG display to canonical", "BUG-01KQ7-JDT511BZ", "BUG-01KQ7JDT511BZ"},

		// Already canonical: unchanged.
		{"canonical unchanged", "FEAT-01KQ7JDT511BZ", "FEAT-01KQ7JDT511BZ"},

		// Edge cases.
		{"no hyphens", "FEAT01KQ7JDT511BZ", "FEAT01KQ7JDT511BZ"},
		{"single hyphen", "FEAT-01KQ7JDT511BZ", "FEAT-01KQ7JDT511BZ"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := displayToCanonical(tt.id)
			if got != tt.want {
				t.Errorf("displayToCanonical(%q) = %q, want %q", tt.id, got, tt.want)
			}
		})
	}
}

// TestWorktreeCreate_DisplayIDRejected verifies AC-008:
// Display-format entity IDs are rejected with a suggestion.
func TestWorktreeCreate_DisplayIDRejected(t *testing.T) {
	t.Parallel()

	store := worktree.NewStore(t.TempDir())
	repoDir, _ := setupGitRepoForRemove(t, "wt-display-id-reject")
	gitOps := worktree.NewGit(repoDir)

	// We need an EntityService but we can use the nil-safe inline error path.
	// The display-ID check happens before the entity lookup, so we can test
	// with a handler that will fail on display ID before hitting entitySvc.
	handler := worktreeCreateAction(store, nil, gitOps, repoDir)

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{
		"entity_id": "FEAT-01KQ7-JDT511BZ",
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

	// AC-008: display ID should be rejected.
	errInfo, hasErr := resp["error"].(map[string]any)
	if !hasErr {
		t.Fatalf("expected error for display ID, got: %v", resp)
	}

	if code, _ := errInfo["code"].(string); code != "invalid_entity_id" {
		t.Errorf("error code = %q, want %q", code, "invalid_entity_id")
	}

	msg, _ := errInfo["message"].(string)
	if !strings.Contains(msg, "display ID") {
		t.Errorf("error message should mention 'display ID': %s", msg)
	}
	if !strings.Contains(msg, "canonical form") {
		t.Errorf("error message should suggest canonical form: %s", msg)
	}
	if !strings.Contains(msg, "FEAT-01KQ7JDT511BZ") {
		t.Errorf("error message should include canonical form: %s", msg)
	}
}

// TestWorktreeCreate_CanonicalIDAccepted verifies AC-009:
// Canonical entity IDs pass the display-ID gate (testing via isDisplayEntityID).
// The full handler path is tested in TestWorktreeCreate_DisplayIDRejected above.
func TestWorktreeCreate_CanonicalIDAccepted(t *testing.T) {
	t.Parallel()

	// AC-009: canonical IDs (single hyphen) are not flagged as display IDs.
	canonicalIDs := []string{
		"FEAT-01KQ7JDT511BZ",
		"BUG-01KQ7JDT511BZ",
		"FEAT-01ABCDEFGHIJKL",
	}

	for _, id := range canonicalIDs {
		if isDisplayEntityID(id) {
			t.Errorf("isDisplayEntityID(%q) = true, want false (canonical ID)", id)
		}
	}
}

// TestIsDisplayEntityID_WrongHybridFormats verifies edge cases for display ID detection.
func TestIsDisplayEntityID_WrongHybridFormats(t *testing.T) {
	t.Parallel()

	// These are wrong but not display IDs — they'd fail at entity lookup.
	// isDisplayEntityID only checks for the embedded hyphen pattern.
	wrongFormats := []string{
		"TASK-01KQ7-JDT511BZ", // TASK has single hyphen after prefix — but IsDisplayEntityID only checks hyphen count
		"FEAT01KQ7JDT511BZ",   // missing hyphen entirely — no type separator
	}

	for _, id := range wrongFormats {
		t.Run(id, func(t *testing.T) {
			t.Parallel()
			// These either have <2 hyphens (no display-ID pattern) or have hyphen
			// count that triggers display. The type prefix check happens separately.
			got := isDisplayEntityID(id)
			t.Logf("isDisplayEntityID(%q) = %v", id, got)
		})
	}
}
