package mcp_test

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/mcptest"

	"kanbanzai/internal/git"
	kbzmcp "kanbanzai/internal/mcp"
	"kanbanzai/internal/service"
	"kanbanzai/internal/storage"
	"kanbanzai/internal/worktree"
)

type worktreeTestEnv struct {
	server    *mcptest.Server
	entitySvc *service.EntityService
	store     *worktree.Store
	gitOps    *worktree.Git
	repoPath  string
	stateRoot string
}

func setupWorktreeTestServer(t *testing.T) *worktreeTestEnv {
	t.Helper()

	// Create temp directories
	repoPath := t.TempDir()
	stateRoot := filepath.Join(repoPath, ".kbz", "state")
	if err := os.MkdirAll(stateRoot, 0755); err != nil {
		t.Fatalf("create state dir: %v", err)
	}

	// Initialize a git repository
	if err := initGitRepo(repoPath); err != nil {
		t.Fatalf("init git repo: %v", err)
	}

	// Create services
	entitySvc := service.NewEntityService(stateRoot)
	store := worktree.NewStore(stateRoot)
	gitOps := worktree.NewGit(repoPath)

	// Create worktree and branch tools
	tools := kbzmcp.WorktreeTools(store, entitySvc, gitOps)
	tools = append(tools, kbzmcp.BranchTools(store, repoPath, git.DefaultBranchThresholds())...)

	ts, err := mcptest.NewServer(t, tools...)
	if err != nil {
		t.Fatalf("start test server: %v", err)
	}

	return &worktreeTestEnv{
		server:    ts,
		entitySvc: entitySvc,
		store:     store,
		gitOps:    gitOps,
		repoPath:  repoPath,
		stateRoot: stateRoot,
	}
}

func initGitRepo(path string) error {
	// git init
	cmd := exec.Command("git", "init")
	cmd.Dir = path
	if err := cmd.Run(); err != nil {
		return err
	}

	// Configure git user for commits
	cmd = exec.Command("git", "config", "user.email", "test@example.com")
	cmd.Dir = path
	if err := cmd.Run(); err != nil {
		return err
	}

	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = path
	if err := cmd.Run(); err != nil {
		return err
	}

	// Create initial commit
	readmePath := filepath.Join(path, "README.md")
	if err := os.WriteFile(readmePath, []byte("# Test Repo\n"), 0644); err != nil {
		return err
	}

	cmd = exec.Command("git", "add", ".")
	cmd.Dir = path
	if err := cmd.Run(); err != nil {
		return err
	}

	cmd = exec.Command("git", "commit", "-m", "Initial commit")
	cmd.Dir = path
	if err := cmd.Run(); err != nil {
		return err
	}

	// Ensure we're on main branch
	cmd = exec.Command("git", "branch", "-M", "main")
	cmd.Dir = path
	return cmd.Run()
}

func callWorktreeTool(t *testing.T, env *worktreeTestEnv, name string, args map[string]any) *mcp.CallToolResult {
	t.Helper()
	ctx := context.Background()
	request := mcp.CallToolRequest{}
	request.Params.Name = name
	request.Params.Arguments = args
	result, err := env.server.Client().CallTool(ctx, request)
	if err != nil {
		t.Fatalf("CallTool(%s): %v", name, err)
	}
	return result
}

func worktreeResultText(t *testing.T, result *mcp.CallToolResult) string {
	t.Helper()
	if len(result.Content) == 0 {
		t.Fatal("result has no content")
	}
	tc, ok := result.Content[0].(mcp.TextContent)
	if !ok {
		t.Fatalf("expected TextContent, got %T", result.Content[0])
	}
	return tc.Text
}

func createTestFeature(t *testing.T, env *worktreeTestEnv, slug string) string {
	t.Helper()

	// Write a plan record directly (bypasses prefix registry validation)
	planID := "P1-test-plan"
	entityStore := storage.NewEntityStore(env.stateRoot)
	_, err := entityStore.Write(storage.EntityRecord{
		Type: "plan",
		ID:   planID,
		Slug: "test-plan",
		Fields: map[string]any{
			"id":         planID,
			"slug":       "test-plan",
			"title":      "Test Plan",
			"status":     "active",
			"summary":    "Test plan for worktree tests",
			"created":    "2026-03-19T12:00:00Z",
			"created_by": "tester",
			"updated":    "2026-03-19T12:00:00Z",
		},
	})
	if err != nil {
		t.Fatalf("create test plan: %v", err)
	}

	// Create a feature under the plan
	result, err := env.entitySvc.CreateFeature(service.CreateFeatureInput{
		Slug:      slug,
		Parent:    planID,
		Summary:   "Test feature for worktree",
		CreatedBy: "tester",
	})
	if err != nil {
		t.Fatalf("create test feature: %v", err)
	}
	return result.ID
}

func createTestBug(t *testing.T, env *worktreeTestEnv, slug string) string {
	t.Helper()

	result, err := env.entitySvc.CreateBug(service.CreateBugInput{
		Slug:       slug,
		Title:      "Test Bug",
		ReportedBy: "tester",
		Observed:   "Bug observed",
		Expected:   "Bug expected",
		Severity:   "medium",
		Priority:   "medium",
		Type:       "implementation-defect",
	})
	if err != nil {
		t.Fatalf("create test bug: %v", err)
	}
	return result.ID
}

func TestWorktreeCreate(t *testing.T) {
	env := setupWorktreeTestServer(t)
	defer env.server.Close()

	featureID := createTestFeature(t, env, "test-feature")

	result := callWorktreeTool(t, env, "worktree_create", map[string]any{
		"entity_id":  featureID,
		"created_by": "tester",
	})

	if result.IsError {
		t.Fatalf("worktree_create returned error: %s", worktreeResultText(t, result))
	}

	text := worktreeResultText(t, result)
	var parsed map[string]any
	if err := json.Unmarshal([]byte(text), &parsed); err != nil {
		t.Fatalf("failed to parse result JSON: %v\nraw: %s", err, text)
	}

	success, ok := parsed["success"].(bool)
	if !ok || !success {
		t.Fatalf("expected success=true, got: %v", parsed["success"])
	}

	wt, ok := parsed["worktree"].(map[string]any)
	if !ok {
		t.Fatalf("expected worktree map in result, got: %T", parsed["worktree"])
	}

	if wt["entity_id"] != featureID {
		t.Errorf("worktree entity_id = %v, want %v", wt["entity_id"], featureID)
	}

	if wt["status"] != "active" {
		t.Errorf("worktree status = %v, want active", wt["status"])
	}

	branch, ok := wt["branch"].(string)
	if !ok || branch == "" {
		t.Errorf("expected non-empty branch, got: %v", wt["branch"])
	}

	if !strings.HasPrefix(branch, "feature/") {
		t.Errorf("expected branch to start with feature/, got: %s", branch)
	}
}

func TestWorktreeCreate_WithCustomBranch(t *testing.T) {
	env := setupWorktreeTestServer(t)
	defer env.server.Close()

	featureID := createTestFeature(t, env, "custom-branch-feature")

	result := callWorktreeTool(t, env, "worktree_create", map[string]any{
		"entity_id":   featureID,
		"branch_name": "feature/custom-test-branch",
		"created_by":  "tester",
	})

	if result.IsError {
		t.Fatalf("worktree_create returned error: %s", worktreeResultText(t, result))
	}

	text := worktreeResultText(t, result)
	var parsed map[string]any
	if err := json.Unmarshal([]byte(text), &parsed); err != nil {
		t.Fatalf("failed to parse result JSON: %v", err)
	}

	wt := parsed["worktree"].(map[string]any)
	if wt["branch"] != "feature/custom-test-branch" {
		t.Errorf("branch = %v, want feature/custom-test-branch", wt["branch"])
	}
}

func TestWorktreeCreate_EntityNotFound(t *testing.T) {
	env := setupWorktreeTestServer(t)
	defer env.server.Close()

	result := callWorktreeTool(t, env, "worktree_create", map[string]any{
		"entity_id":  "FEAT-01JXNOTEXIST",
		"created_by": "tester",
	})

	if !result.IsError {
		t.Fatalf("expected error for non-existent entity, got success: %s", worktreeResultText(t, result))
	}

	text := worktreeResultText(t, result)
	if !strings.Contains(text, "ENTITY_NOT_FOUND") {
		t.Errorf("expected ENTITY_NOT_FOUND error, got: %s", text)
	}
}

func TestWorktreeCreate_InvalidEntityType(t *testing.T) {
	env := setupWorktreeTestServer(t)
	defer env.server.Close()

	// Create an epic - epics are not valid for worktrees
	epicResult, err := env.entitySvc.CreateEpic(service.CreateEpicInput{
		Slug:      "wt-epic",
		Title:     "Worktree Epic",
		Summary:   "Test epic",
		CreatedBy: "tester",
	})
	if err != nil {
		t.Fatalf("create epic: %v", err)
	}

	result := callWorktreeTool(t, env, "worktree_create", map[string]any{
		"entity_id":  epicResult.ID,
		"created_by": "tester",
	})

	if !result.IsError {
		t.Fatalf("expected error for epic entity, got success: %s", worktreeResultText(t, result))
	}

	text := worktreeResultText(t, result)
	if !strings.Contains(text, "INVALID_ENTITY_TYPE") {
		t.Errorf("expected INVALID_ENTITY_TYPE error, got: %s", text)
	}
}

func TestWorktreeCreate_Duplicate(t *testing.T) {
	env := setupWorktreeTestServer(t)
	defer env.server.Close()

	featureID := createTestFeature(t, env, "dup-feature")

	// Create first worktree
	result := callWorktreeTool(t, env, "worktree_create", map[string]any{
		"entity_id":  featureID,
		"created_by": "tester",
	})

	if result.IsError {
		t.Fatalf("first worktree_create returned error: %s", worktreeResultText(t, result))
	}

	// Try to create duplicate
	result = callWorktreeTool(t, env, "worktree_create", map[string]any{
		"entity_id":  featureID,
		"created_by": "tester",
	})

	if !result.IsError {
		t.Fatalf("expected error for duplicate worktree, got success: %s", worktreeResultText(t, result))
	}

	text := worktreeResultText(t, result)
	if !strings.Contains(text, "WORKTREE_EXISTS") {
		t.Errorf("expected WORKTREE_EXISTS error, got: %s", text)
	}
}

func TestWorktreeList(t *testing.T) {
	env := setupWorktreeTestServer(t)
	defer env.server.Close()

	// Create a worktree
	featureID := createTestFeature(t, env, "list-feature")
	_ = callWorktreeTool(t, env, "worktree_create", map[string]any{
		"entity_id":  featureID,
		"created_by": "tester",
	})

	// List worktrees
	result := callWorktreeTool(t, env, "worktree_list", map[string]any{})

	if result.IsError {
		t.Fatalf("worktree_list returned error: %s", worktreeResultText(t, result))
	}

	text := worktreeResultText(t, result)
	var parsed map[string]any
	if err := json.Unmarshal([]byte(text), &parsed); err != nil {
		t.Fatalf("failed to parse result JSON: %v", err)
	}

	if !parsed["success"].(bool) {
		t.Fatal("expected success=true")
	}

	worktrees, ok := parsed["worktrees"].([]any)
	if !ok {
		t.Fatalf("expected worktrees array, got: %T", parsed["worktrees"])
	}

	if len(worktrees) != 1 {
		t.Errorf("expected 1 worktree, got %d", len(worktrees))
	}

	count, ok := parsed["count"].(float64)
	if !ok || int(count) != 1 {
		t.Errorf("expected count=1, got: %v", parsed["count"])
	}
}

func TestWorktreeList_FilterByStatus(t *testing.T) {
	env := setupWorktreeTestServer(t)
	defer env.server.Close()

	// Create a worktree (will be active)
	featureID := createTestFeature(t, env, "filter-feature")
	_ = callWorktreeTool(t, env, "worktree_create", map[string]any{
		"entity_id":  featureID,
		"created_by": "tester",
	})

	// List with status filter for merged (should return 0)
	result := callWorktreeTool(t, env, "worktree_list", map[string]any{
		"status": "merged",
	})

	if result.IsError {
		t.Fatalf("worktree_list returned error: %s", worktreeResultText(t, result))
	}

	text := worktreeResultText(t, result)
	var parsed map[string]any
	if err := json.Unmarshal([]byte(text), &parsed); err != nil {
		t.Fatalf("failed to parse result JSON: %v", err)
	}

	worktrees := parsed["worktrees"].([]any)
	if len(worktrees) != 0 {
		t.Errorf("expected 0 merged worktrees, got %d", len(worktrees))
	}

	// List with status filter for active (should return 1)
	result = callWorktreeTool(t, env, "worktree_list", map[string]any{
		"status": "active",
	})

	text = worktreeResultText(t, result)
	if err := json.Unmarshal([]byte(text), &parsed); err != nil {
		t.Fatalf("failed to parse result JSON: %v", err)
	}

	worktrees = parsed["worktrees"].([]any)
	if len(worktrees) != 1 {
		t.Errorf("expected 1 active worktree, got %d", len(worktrees))
	}
}

func TestWorktreeGet(t *testing.T) {
	env := setupWorktreeTestServer(t)
	defer env.server.Close()

	// Create a worktree
	featureID := createTestFeature(t, env, "get-feature")
	createResult := callWorktreeTool(t, env, "worktree_create", map[string]any{
		"entity_id":  featureID,
		"created_by": "tester",
	})

	// Get the worktree
	result := callWorktreeTool(t, env, "worktree_get", map[string]any{
		"entity_id": featureID,
	})

	if result.IsError {
		t.Fatalf("worktree_get returned error: %s", worktreeResultText(t, result))
	}

	text := worktreeResultText(t, result)
	var parsed map[string]any
	if err := json.Unmarshal([]byte(text), &parsed); err != nil {
		t.Fatalf("failed to parse result JSON: %v", err)
	}

	if !parsed["success"].(bool) {
		t.Fatal("expected success=true")
	}

	wt := parsed["worktree"].(map[string]any)
	if wt["entity_id"] != featureID {
		t.Errorf("entity_id = %v, want %v", wt["entity_id"], featureID)
	}

	// Verify it matches the created worktree
	var createdParsed map[string]any
	json.Unmarshal([]byte(worktreeResultText(t, createResult)), &createdParsed)
	createdWt := createdParsed["worktree"].(map[string]any)

	if wt["id"] != createdWt["id"] {
		t.Errorf("id = %v, want %v", wt["id"], createdWt["id"])
	}
}

func TestWorktreeGet_NotFound(t *testing.T) {
	env := setupWorktreeTestServer(t)
	defer env.server.Close()

	result := callWorktreeTool(t, env, "worktree_get", map[string]any{
		"entity_id": "FEAT-01JXNOTEXIST",
	})

	if !result.IsError {
		t.Fatalf("expected error for non-existent worktree, got success: %s", worktreeResultText(t, result))
	}

	text := worktreeResultText(t, result)
	if !strings.Contains(text, "NO_WORKTREE") {
		t.Errorf("expected NO_WORKTREE error, got: %s", text)
	}
}

func TestWorktreeRemove(t *testing.T) {
	env := setupWorktreeTestServer(t)
	defer env.server.Close()

	// Create a worktree
	featureID := createTestFeature(t, env, "remove-feature")
	createResult := callWorktreeTool(t, env, "worktree_create", map[string]any{
		"entity_id":  featureID,
		"created_by": "tester",
	})

	var createdParsed map[string]any
	json.Unmarshal([]byte(worktreeResultText(t, createResult)), &createdParsed)
	createdWt := createdParsed["worktree"].(map[string]any)
	wtID := createdWt["id"].(string)
	wtPath := createdWt["path"].(string)

	// Remove the worktree
	result := callWorktreeTool(t, env, "worktree_remove", map[string]any{
		"entity_id": featureID,
	})

	if result.IsError {
		t.Fatalf("worktree_remove returned error: %s", worktreeResultText(t, result))
	}

	text := worktreeResultText(t, result)
	var parsed map[string]any
	if err := json.Unmarshal([]byte(text), &parsed); err != nil {
		t.Fatalf("failed to parse result JSON: %v", err)
	}

	if !parsed["success"].(bool) {
		t.Fatal("expected success=true")
	}

	removed := parsed["removed"].(map[string]any)
	if removed["id"] != wtID {
		t.Errorf("removed id = %v, want %v", removed["id"], wtID)
	}
	if removed["path"] != wtPath {
		t.Errorf("removed path = %v, want %v", removed["path"], wtPath)
	}

	// Verify worktree no longer exists
	result = callWorktreeTool(t, env, "worktree_get", map[string]any{
		"entity_id": featureID,
	})

	if !result.IsError {
		t.Fatal("expected error getting removed worktree")
	}
}

func TestWorktreeRemove_NotFound(t *testing.T) {
	env := setupWorktreeTestServer(t)
	defer env.server.Close()

	result := callWorktreeTool(t, env, "worktree_remove", map[string]any{
		"entity_id": "FEAT-01JXNOTEXIST",
	})

	if !result.IsError {
		t.Fatalf("expected error for non-existent worktree, got success: %s", worktreeResultText(t, result))
	}

	text := worktreeResultText(t, result)
	if !strings.Contains(text, "NO_WORKTREE") {
		t.Errorf("expected NO_WORKTREE error, got: %s", text)
	}
}

func TestBranchStatus(t *testing.T) {
	env := setupWorktreeTestServer(t)
	defer env.server.Close()

	// Create a worktree
	featureID := createTestFeature(t, env, "branch-status-feature")
	_ = callWorktreeTool(t, env, "worktree_create", map[string]any{
		"entity_id":  featureID,
		"created_by": "tester",
	})

	// Get branch status
	result := callWorktreeTool(t, env, "branch_status", map[string]any{
		"entity_id": featureID,
	})

	if result.IsError {
		t.Fatalf("branch_status returned error: %s", worktreeResultText(t, result))
	}

	text := worktreeResultText(t, result)
	var parsed map[string]any
	if err := json.Unmarshal([]byte(text), &parsed); err != nil {
		t.Fatalf("failed to parse result JSON: %v", err)
	}

	if !parsed["success"].(bool) {
		t.Fatal("expected success=true")
	}

	branch, ok := parsed["branch"].(string)
	if !ok || branch == "" {
		t.Errorf("expected non-empty branch, got: %v", parsed["branch"])
	}

	metrics, ok := parsed["metrics"].(map[string]any)
	if !ok {
		t.Fatalf("expected metrics map, got: %T", parsed["metrics"])
	}

	// Verify expected metric fields exist
	expectedFields := []string{
		"branch_age_days",
		"commits_behind_main",
		"commits_ahead_of_main",
		"last_commit_at",
		"last_commit_age_days",
		"has_conflicts",
	}

	for _, field := range expectedFields {
		if _, ok := metrics[field]; !ok {
			t.Errorf("missing metric field: %s", field)
		}
	}

	// A newly created branch should have 0 commits behind main
	behind := metrics["commits_behind_main"].(float64)
	if int(behind) != 0 {
		t.Errorf("commits_behind_main = %v, want 0 for new branch", behind)
	}

	// Should have no conflicts
	hasConflicts := metrics["has_conflicts"].(bool)
	if hasConflicts {
		t.Error("expected has_conflicts=false for new branch")
	}

	// Warnings and errors should be arrays (may be empty)
	if _, ok := parsed["warnings"].([]any); !ok {
		// Could be nil
		if parsed["warnings"] != nil {
			t.Errorf("expected warnings array or nil, got: %T", parsed["warnings"])
		}
	}

	if _, ok := parsed["errors"].([]any); !ok {
		// Could be nil
		if parsed["errors"] != nil {
			t.Errorf("expected errors array or nil, got: %T", parsed["errors"])
		}
	}
}

func TestBranchStatus_NoWorktree(t *testing.T) {
	env := setupWorktreeTestServer(t)
	defer env.server.Close()

	result := callWorktreeTool(t, env, "branch_status", map[string]any{
		"entity_id": "FEAT-01JXNOTEXIST",
	})

	if !result.IsError {
		t.Fatalf("expected error for non-existent worktree, got success: %s", worktreeResultText(t, result))
	}

	text := worktreeResultText(t, result)
	if !strings.Contains(text, "NO_WORKTREE") {
		t.Errorf("expected NO_WORKTREE error, got: %s", text)
	}
}

func TestWorktreeCreate_ForBug(t *testing.T) {
	env := setupWorktreeTestServer(t)
	defer env.server.Close()

	bugID := createTestBug(t, env, "test-bug")

	result := callWorktreeTool(t, env, "worktree_create", map[string]any{
		"entity_id":  bugID,
		"created_by": "tester",
	})

	if result.IsError {
		t.Fatalf("worktree_create returned error: %s", worktreeResultText(t, result))
	}

	text := worktreeResultText(t, result)
	var parsed map[string]any
	if err := json.Unmarshal([]byte(text), &parsed); err != nil {
		t.Fatalf("failed to parse result JSON: %v", err)
	}

	wt := parsed["worktree"].(map[string]any)
	branch := wt["branch"].(string)

	// Bug branches should use bugfix/ prefix
	if !strings.HasPrefix(branch, "bugfix/") {
		t.Errorf("expected bug branch to start with bugfix/, got: %s", branch)
	}
}
