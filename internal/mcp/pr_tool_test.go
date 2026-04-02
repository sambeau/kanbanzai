package mcp

import (
	"context"
	"testing"

	"github.com/sambeau/kanbanzai/internal/config"
	"github.com/sambeau/kanbanzai/internal/git"
	"github.com/sambeau/kanbanzai/internal/service"
)

// localConfigWithToken builds a minimal LocalConfig that passes the GitHub
// token check so tests can reach the worktree-lookup code path.
func localConfigWithToken(token string) *config.LocalConfig {
	lc := &config.LocalConfig{}
	lc.GitHub.Token = token
	return lc
}

// ─── createPR ─────────────────────────────────────────────────────────────────

func TestCreatePR_NoWorktree_NotApplicable(t *testing.T) {
	t.Parallel()

	store := newEmptyWorktreeStore(t)
	entitySvc := service.NewEntityService(t.TempDir())
	localConfig := localConfigWithToken("ghp_test_token")

	result, err := createPR(
		context.Background(),
		store,
		entitySvc,
		t.TempDir(),
		localConfig,
		"FEAT-001",
		false,
	)

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if result == nil {
		t.Fatal("expected a result map, got nil")
	}
	if result["status"] != "not_applicable" {
		t.Errorf("status = %q, want not_applicable; full result: %v", result["status"], result)
	}
	if result["reason"] == "" {
		t.Error("expected non-empty reason field")
	}
}

func TestCreatePR_NoWorktree_BugEntity_NotApplicable(t *testing.T) {
	t.Parallel()

	store := newEmptyWorktreeStore(t)
	entitySvc := service.NewEntityService(t.TempDir())
	localConfig := localConfigWithToken("ghp_test_token")

	result, err := createPR(
		context.Background(),
		store,
		entitySvc,
		t.TempDir(),
		localConfig,
		"BUG-001",
		false,
	)

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if result["status"] != "not_applicable" {
		t.Errorf("status = %q, want not_applicable", result["status"])
	}
}

func TestCreatePR_InvalidEntityID_ReturnsError(t *testing.T) {
	t.Parallel()

	store := newEmptyWorktreeStore(t)
	entitySvc := service.NewEntityService(t.TempDir())
	localConfig := localConfigWithToken("ghp_test_token")

	_, err := createPR(
		context.Background(),
		store,
		entitySvc,
		t.TempDir(),
		localConfig,
		"TASK-001",
		false,
	)

	if err == nil {
		t.Fatal("expected an error for invalid entity ID, got nil")
	}
}

func TestCreatePR_MissingToken_ReturnsError(t *testing.T) {
	t.Parallel()

	store := newEmptyWorktreeStore(t)
	entitySvc := service.NewEntityService(t.TempDir())

	_, err := createPR(
		context.Background(),
		store,
		entitySvc,
		t.TempDir(),
		nil, // no token
		"FEAT-001",
		false,
	)

	if err == nil {
		t.Fatal("expected an error when GitHub token is missing, got nil")
	}
}

func TestCreatePR_StoreError_Propagated(t *testing.T) {
	t.Parallel()

	store := newBrokenWorktreeStore(t)
	entitySvc := service.NewEntityService(t.TempDir())
	localConfig := localConfigWithToken("ghp_test_token")

	_, err := createPR(
		context.Background(),
		store,
		entitySvc,
		t.TempDir(),
		localConfig,
		"FEAT-001",
		false,
	)

	if err == nil {
		t.Fatal("expected a store error to propagate, got nil")
	}
}

// ─── getPRStatusForEntity ─────────────────────────────────────────────────────

func TestGetPRStatusForEntity_NoWorktree_NotApplicable(t *testing.T) {
	t.Parallel()

	store := newEmptyWorktreeStore(t)
	localConfig := localConfigWithToken("ghp_test_token")

	result, err := getPRStatusForEntity(
		context.Background(),
		store,
		t.TempDir(),
		localConfig,
		"FEAT-001",
	)

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if result == nil {
		t.Fatal("expected a result map, got nil")
	}
	if result["status"] != "not_applicable" {
		t.Errorf("status = %q, want not_applicable; full result: %v", result["status"], result)
	}
	if result["reason"] == "" {
		t.Error("expected non-empty reason field")
	}
}

func TestGetPRStatusForEntity_InvalidEntityID_ReturnsError(t *testing.T) {
	t.Parallel()

	store := newEmptyWorktreeStore(t)
	localConfig := localConfigWithToken("ghp_test_token")

	_, err := getPRStatusForEntity(
		context.Background(),
		store,
		t.TempDir(),
		localConfig,
		"TASK-001",
	)

	if err == nil {
		t.Fatal("expected an error for invalid entity ID, got nil")
	}
}

func TestGetPRStatusForEntity_MissingToken_ReturnsError(t *testing.T) {
	t.Parallel()

	store := newEmptyWorktreeStore(t)

	_, err := getPRStatusForEntity(
		context.Background(),
		store,
		t.TempDir(),
		nil, // no token
		"FEAT-001",
	)

	if err == nil {
		t.Fatal("expected an error when GitHub token is missing, got nil")
	}
}

func TestGetPRStatusForEntity_StoreError_Propagated(t *testing.T) {
	t.Parallel()

	store := newBrokenWorktreeStore(t)
	localConfig := localConfigWithToken("ghp_test_token")

	_, err := getPRStatusForEntity(
		context.Background(),
		store,
		t.TempDir(),
		localConfig,
		"FEAT-001",
	)

	if err == nil {
		t.Fatal("expected a store error to propagate, got nil")
	}
}

// ─── updatePR ─────────────────────────────────────────────────────────────────

func TestUpdatePR_NoWorktree_NotApplicable(t *testing.T) {
	t.Parallel()

	store := newEmptyWorktreeStore(t)
	entitySvc := service.NewEntityService(t.TempDir())
	localConfig := localConfigWithToken("ghp_test_token")
	thresholds := git.BranchThresholds{}

	result, err := updatePR(
		context.Background(),
		store,
		entitySvc,
		t.TempDir(),
		thresholds,
		localConfig,
		"FEAT-001",
	)

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if result == nil {
		t.Fatal("expected a result map, got nil")
	}
	if result["status"] != "not_applicable" {
		t.Errorf("status = %q, want not_applicable; full result: %v", result["status"], result)
	}
	if result["reason"] == "" {
		t.Error("expected non-empty reason field")
	}
}

func TestUpdatePR_InvalidEntityID_ReturnsError(t *testing.T) {
	t.Parallel()

	store := newEmptyWorktreeStore(t)
	entitySvc := service.NewEntityService(t.TempDir())
	localConfig := localConfigWithToken("ghp_test_token")
	thresholds := git.BranchThresholds{}

	_, err := updatePR(
		context.Background(),
		store,
		entitySvc,
		t.TempDir(),
		thresholds,
		localConfig,
		"TASK-001",
	)

	if err == nil {
		t.Fatal("expected an error for invalid entity ID, got nil")
	}
}

func TestUpdatePR_MissingToken_ReturnsError(t *testing.T) {
	t.Parallel()

	store := newEmptyWorktreeStore(t)
	entitySvc := service.NewEntityService(t.TempDir())
	thresholds := git.BranchThresholds{}

	_, err := updatePR(
		context.Background(),
		store,
		entitySvc,
		t.TempDir(),
		thresholds,
		nil, // no token
		"FEAT-001",
	)

	if err == nil {
		t.Fatal("expected an error when GitHub token is missing, got nil")
	}
}

func TestUpdatePR_StoreError_Propagated(t *testing.T) {
	t.Parallel()

	store := newBrokenWorktreeStore(t)
	entitySvc := service.NewEntityService(t.TempDir())
	localConfig := localConfigWithToken("ghp_test_token")
	thresholds := git.BranchThresholds{}

	_, err := updatePR(
		context.Background(),
		store,
		entitySvc,
		t.TempDir(),
		thresholds,
		localConfig,
		"FEAT-001",
	)

	if err == nil {
		t.Fatal("expected a store error to propagate, got nil")
	}
}
