package mcp

import (
	"strings"
	"testing"
	"time"

	"github.com/sambeau/kanbanzai/internal/git"
	"github.com/sambeau/kanbanzai/internal/service"
	"github.com/sambeau/kanbanzai/internal/storage"
	"github.com/sambeau/kanbanzai/internal/worktree"
)

// setupReviewingFeatureTest creates a worktree store and entity service with a
// feature entity in "reviewing" status. Follows the same pattern as
// setupPRGateTest in merge_tool_pr_gate_test.go.
func setupReviewingFeatureTest(t *testing.T) (
	wtStore *worktree.Store,
	entitySvc *service.EntityService,
	entityID string,
) {
	t.Helper()

	stateRoot := t.TempDir()
	entityID = "FEAT-01RGTESTINGRV1"
	branch := "feature/FEAT-01RGTESTINGRV1-review-gate-test"

	estore := storage.NewEntityStore(stateRoot)
	_, err := estore.Write(storage.EntityRecord{
		Type: "feature",
		ID:   entityID,
		Slug: "review-gate-test",
		Fields: map[string]any{
			"id":         entityID,
			"slug":       "review-gate-test",
			"parent":     "P1-test",
			"status":     "reviewing",
			"summary":    "Review gate integration test fixture",
			"created":    "2026-01-01T00:00:00Z",
			"created_by": "test",
		},
	})
	if err != nil {
		t.Fatalf("setupReviewingFeatureTest: write entity: %v", err)
	}

	entitySvc = service.NewEntityService(stateRoot)

	wtStore = worktree.NewStore(stateRoot)
	_, err = wtStore.Create(worktree.Record{
		ID:        "WT-RGTEST001",
		EntityID:  entityID,
		Branch:    branch,
		Path:      "/tmp/review-gate-test",
		Status:    worktree.StatusActive,
		Created:   time.Now(),
		CreatedBy: "test",
	})
	if err != nil {
		t.Fatalf("setupReviewingFeatureTest: create worktree record: %v", err)
	}

	return
}

// ─── ReviewReportExistsGate integration: executeMerge level ──────────────────

// TestExecuteMerge_ReviewingFeature_NoReport_OverrideRejected verifies AC-002:
// when a feature is in "reviewing" status and no report document is registered,
// executeMerge must reject the merge even when override: true is supplied.
// The non-bypassable gate must not be circumvented.
func TestExecuteMerge_ReviewingFeature_NoReport_OverrideRejected(t *testing.T) {
	t.Parallel()

	wtStore, entitySvc, entityID := setupReviewingFeatureTest(t)

	// DocumentService backed by empty state — no report documents registered.
	docSvc := service.NewDocumentService(t.TempDir(), t.TempDir())

	_, err := executeMerge(
		wtStore,
		entitySvc,
		docSvc,
		t.TempDir(),
		git.BranchThresholds{},
		nil, // no local config
		entityID,
		true, // override: true — must be rejected by non-bypassable gate
		"attempting to override non-bypassable review gate",
		worktree.MergeStrategySquash,
		true,
	)

	if err == nil {
		t.Fatal("executeMerge() returned nil error; expected rejection because ReviewReportExistsGate is non-bypassable")
	}
	if !strings.Contains(err.Error(), "cannot be bypassed") {
		t.Errorf("error = %q; expected it to contain %q", err.Error(), "cannot be bypassed")
	}
}

// TestExecuteMerge_ReviewingFeature_NilDocSvc_FailsOpen verifies AC-007:
// when the document service is unavailable (nil), ReviewReportExistsGate must
// fail open and must NOT produce a non-bypassable blocking failure. The merge
// may still be blocked by other gates (e.g. EntityDoneGate), but the rejection
// must not be attributable to the review report gate.
func TestExecuteMerge_ReviewingFeature_NilDocSvc_FailsOpen(t *testing.T) {
	t.Parallel()

	wtStore, entitySvc, entityID := setupReviewingFeatureTest(t)

	_, err := executeMerge(
		wtStore,
		entitySvc,
		nil, // doc service unavailable — ReviewReportExistsGate must fail open
		t.TempDir(),
		git.BranchThresholds{},
		nil,
		entityID,
		false,
		"",
		worktree.MergeStrategySquash,
		true,
	)

	// The merge may be blocked by other bypassable gates (e.g. EntityDoneGate
	// because status is "reviewing", not "done"), but it must NOT be rejected
	// with a non-bypassable error from ReviewReportExistsGate.
	if err != nil && strings.Contains(err.Error(), "cannot be bypassed") {
		t.Errorf("ReviewReportExistsGate must fail open when DocSvc is nil, but got: %v", err)
	}
}
