package mcp

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/sambeau/kanbanzai/internal/config"
	"github.com/sambeau/kanbanzai/internal/git"
	"github.com/sambeau/kanbanzai/internal/service"
	"github.com/sambeau/kanbanzai/internal/storage"
	"github.com/sambeau/kanbanzai/internal/worktree"
)

// setupPRGateTest creates a worktree store with one record and an entity service
// with one feature entity. The returned LocalConfig has a fake GitHub token so
// the PR block runs. The caller is responsible for injecting loadPRConfigFunc
// and getPRStatusFunc before calling checkMergeReadiness or executeMerge.
func setupPRGateTest(t *testing.T) (
	wtStore *worktree.Store,
	entitySvc *service.EntityService,
	localCfg *config.LocalConfig,
	entityID string,
	branch string,
) {
	t.Helper()

	stateRoot := t.TempDir()
	entityID = "FEAT-01PRGATE00001"
	branch = "feature/test-pr-gate"

	// Write a minimal feature entity directly to the state store.
	estore := storage.NewEntityStore(stateRoot)
	_, err := estore.Write(storage.EntityRecord{
		Type: "feature",
		ID:   entityID,
		Slug: "test-pr-gate",
		Fields: map[string]any{
			"id":         entityID,
			"slug":       "test-pr-gate",
			"parent":     "P1-test",
			"status":     "developing",
			"summary":    "PR gate test feature",
			"created":    "2026-01-01T00:00:00Z",
			"created_by": "test",
		},
	})
	if err != nil {
		t.Fatalf("write test entity: %v", err)
	}

	entitySvc = service.NewEntityService(stateRoot)

	// Create a worktree record for the entity.
	wtStore = worktree.NewStore(stateRoot)
	_, err = wtStore.Create(worktree.Record{
		ID:        "WT-PRGATE001",
		EntityID:  entityID,
		Branch:    branch,
		Path:      "/tmp/test-pr-gate",
		Status:    worktree.StatusActive,
		Created:   time.Now(),
		CreatedBy: "test",
	})
	if err != nil {
		t.Fatalf("create worktree record: %v", err)
	}

	// LocalConfig with a fake GitHub token so getPRStatus block is entered.
	localCfg = &config.LocalConfig{}
	localCfg.GitHub.Token = "ghp_fake_test_token"

	return
}

// truePtr returns a pointer to true for use in MergeConfig.RequireGitHubPR.
func truePtr() *bool { v := true; return &v }

// injectPRConfig replaces loadPRConfigFunc with one that returns a MergeConfig
// with RequireGitHubPR set to true. It restores the original on test cleanup.
func injectPRConfig(t *testing.T) {
	t.Helper()
	old := loadPRConfigFunc
	t.Cleanup(func() { loadPRConfigFunc = old })
	loadPRConfigFunc = func() *config.Config {
		cfg := config.DefaultConfig()
		cfg.Merge.RequireGitHubPR = truePtr()
		return &cfg
	}
}

// ─── PR gate tests ────────────────────────────────────────────────────────────

// TestMergeCheck_RequireGitHubPR_Nil_PassesWithoutPR verifies AC-006:
// when require_github_pr is unset and no GitHub token is configured, the
// merge check response contains no pr_gate key.
func TestMergeCheck_RequireGitHubPR_Nil_PassesWithoutPR(t *testing.T) {
	t.Parallel()

	store := newEmptyWorktreeStore(t)
	entitySvc := service.NewEntityService(t.TempDir())
	thresholds := git.BranchThresholds{}

	result, err := checkMergeReadiness(
		context.Background(),
		store,
		entitySvc,
		t.TempDir(),
		thresholds,
		nil, // no localConfig → no GitHub token → PR block skipped
		"FEAT-001",
	)
	if err != nil {
		t.Fatalf("checkMergeReadiness() error = %v", err)
	}
	if _, hasPRGate := result["pr_gate"]; hasPRGate {
		t.Errorf("response contains unexpected pr_gate key: %v", result["pr_gate"])
	}
}

// TestMergeCheck_RequireGitHubPR_True_NoPR_Fails verifies AC-007:
// when require_github_pr: true and no PR is found, pr_gate.status == "failed".
func TestMergeCheck_RequireGitHubPR_True_NoPR_Fails(t *testing.T) {
	// Not parallel: mutates package-level getPRStatusFunc and loadPRConfigFunc.
	wtStore, entitySvc, localCfg, entityID, _ := setupPRGateTest(t)
	injectPRConfig(t)

	// Mock getPRStatusFunc to return nil (no PR found).
	oldFn := getPRStatusFunc
	t.Cleanup(func() { getPRStatusFunc = oldFn })
	getPRStatusFunc = func(_ context.Context, _, _ string, _ *config.LocalConfig) (map[string]any, error) {
		return nil, nil
	}

	result, err := checkMergeReadiness(
		context.Background(),
		wtStore,
		entitySvc,
		t.TempDir(),
		git.BranchThresholds{},
		localCfg,
		entityID,
	)
	if err != nil {
		t.Fatalf("checkMergeReadiness() error = %v", err)
	}
	gate, ok := result["pr_gate"]
	if !ok {
		t.Fatalf("response missing pr_gate key; full result: %v", result)
	}
	gateMap, ok := gate.(map[string]any)
	if !ok {
		t.Fatalf("pr_gate is not a map: %T", gate)
	}
	if gateMap["status"] != "failed" {
		t.Errorf("pr_gate.status = %q, want \"failed\"", gateMap["status"])
	}
}

// TestMergeCheck_RequireGitHubPR_True_NonOpenPR_Fails verifies AC-008:
// when require_github_pr: true and PR state is not "open", pr_gate.status == "failed"
// and the message includes the actual state.
func TestMergeCheck_RequireGitHubPR_True_NonOpenPR_Fails(t *testing.T) {
	// Not parallel: mutates package-level getPRStatusFunc and loadPRConfigFunc.
	wtStore, entitySvc, localCfg, entityID, _ := setupPRGateTest(t)
	injectPRConfig(t)

	// Mock getPRStatusFunc to return a closed PR.
	oldFn := getPRStatusFunc
	t.Cleanup(func() { getPRStatusFunc = oldFn })
	getPRStatusFunc = func(_ context.Context, _, _ string, _ *config.LocalConfig) (map[string]any, error) {
		return map[string]any{
			"state":         "closed",
			"url":           "https://github.com/example/repo/pull/1",
			"ci_status":     "",
			"review_status": "",
			"has_conflicts": false,
		}, nil
	}

	result, err := checkMergeReadiness(
		context.Background(),
		wtStore,
		entitySvc,
		t.TempDir(),
		git.BranchThresholds{},
		localCfg,
		entityID,
	)
	if err != nil {
		t.Fatalf("checkMergeReadiness() error = %v", err)
	}
	gate, ok := result["pr_gate"]
	if !ok {
		t.Fatalf("response missing pr_gate key; full result: %v", result)
	}
	gateMap, ok := gate.(map[string]any)
	if !ok {
		t.Fatalf("pr_gate is not a map: %T", gate)
	}
	if gateMap["status"] != "failed" {
		t.Errorf("pr_gate.status = %q, want \"failed\"", gateMap["status"])
	}
	msg, _ := gateMap["message"].(string)
	if !strings.Contains(msg, "closed") {
		t.Errorf("pr_gate.message = %q, want it to contain the actual state %q", msg, "closed")
	}
}

// TestMergeExecute_RequireGitHubPR_True_NoPR_Blocked verifies AC-009:
// when require_github_pr: true and no open PR exists, executeMerge returns an error.
func TestMergeExecute_RequireGitHubPR_True_NoPR_Blocked(t *testing.T) {
	// Not parallel: mutates package-level getPRStatusFunc and loadPRConfigFunc.
	wtStore, entitySvc, localCfg, entityID, _ := setupPRGateTest(t)
	injectPRConfig(t)

	// Mock getPRStatusFunc to return nil (no PR found).
	oldFn := getPRStatusFunc
	t.Cleanup(func() { getPRStatusFunc = oldFn })
	getPRStatusFunc = func(_ context.Context, _, _ string, _ *config.LocalConfig) (map[string]any, error) {
		return nil, nil
	}

	// Also stub out mergeCommitFunc to avoid git operations.
	oldCommit := mergeCommitFunc
	t.Cleanup(func() { mergeCommitFunc = oldCommit })
	mergeCommitFunc = func(_, _ string) (bool, error) { return false, nil }

	// Use override=true to bypass the standard merge gates (entity status, verification,
	// default branch) and reach the PR gate check, which is what AC-009 tests.
	_, err := executeMerge(
		wtStore,
		entitySvc,
		t.TempDir(),
		git.BranchThresholds{},
		localCfg,
		entityID,
		true,  // override standard gates so PR gate is reached
		"bypassing standard gates for PR gate test",
		worktree.MergeStrategySquash,
		true,
	)
	if err == nil {
		t.Fatal("executeMerge() returned nil error, want a PR gate error")
	}
	if !strings.Contains(err.Error(), "require_github_pr") {
		t.Errorf("error = %q, want it to mention require_github_pr", err.Error())
	}
}

// TestMergeCheck_RequireGitHubPR_Absent_NoNewFailures verifies AC-010:
// a config without require_github_pr set produces no pr_gate failures.
func TestMergeCheck_RequireGitHubPR_Absent_NoNewFailures(t *testing.T) {
	t.Parallel()

	store := newEmptyWorktreeStore(t)
	entitySvc := service.NewEntityService(t.TempDir())
	thresholds := git.BranchThresholds{}

	// Default config (no require_github_pr) + no GitHub token.
	result, err := checkMergeReadiness(
		context.Background(),
		store,
		entitySvc,
		t.TempDir(),
		thresholds,
		nil,
		"FEAT-001",
	)
	if err != nil {
		t.Fatalf("checkMergeReadiness() error = %v", err)
	}
	if _, hasPRGate := result["pr_gate"]; hasPRGate {
		t.Errorf("response contains unexpected pr_gate key; no new gate failures expected: %v", result["pr_gate"])
	}
}
