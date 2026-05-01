package mcp

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/sambeau/kanbanzai/internal/git"
	"github.com/sambeau/kanbanzai/internal/worktree"
)

func TestFilterDrift_RemovesDriftMessages(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    []string
		expected []string
	}{
		{
			name:     "empty slice",
			input:    []string{},
			expected: []string{},
		},
		{
			name:     "nil slice",
			input:    nil,
			expected: nil,
		},
		{
			name:     "no drift messages",
			input:    []string{"branch is stale: no commits in 14 days"},
			expected: []string{"branch is stale: no commits in 14 days"},
		},
		{
			name:     "drift warning removed",
			input:    []string{"branch is drifting: 60 commits behind main (threshold: 50)"},
			expected: []string{},
		},
		{
			name:     "critical drift error removed",
			input:    []string{"branch has critical drift: 120 commits behind main (threshold: 100)"},
			expected: []string{},
		},
		{
			name: "mixed: drift and non-drift",
			input: []string{
				"branch is stale: no commits in 30 days (threshold: 14 days)",
				"branch is drifting: 60 commits behind main (threshold: 50)",
				"branch has merge conflicts with main",
			},
			expected: []string{
				"branch is stale: no commits in 30 days (threshold: 14 days)",
				"branch has merge conflicts with main",
			},
		},
		{
			name: "all drift messages removed",
			input: []string{
				"branch is drifting: 55 commits behind main (threshold: 50)",
				"branch has critical drift: 110 commits behind main (threshold: 100)",
			},
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := filterDrift(tt.input)
			if len(got) != len(tt.expected) {
				t.Errorf("filterDrift() len = %d, want %d; got = %v, want = %v",
					len(got), len(tt.expected), got, tt.expected)
				return
			}
			for i := range got {
				if got[i] != tt.expected[i] {
					t.Errorf("filterDrift()[%d] = %q, want %q", i, got[i], tt.expected[i])
				}
			}
		})
	}
}

// TestBranchStatusAction_UnmergedStillAlerts verifies AC-005:
// Unmerged branch → merged=false and drift alerts are not suppressed.
func TestBranchStatusAction_UnmergedStillAlerts(t *testing.T) {
	repoDir, wtAbsPath := setupGitRepoForRemove(t, "wt-unmerged-drift")
	store := worktree.NewStore(t.TempDir())

	// Create a worktree record WITHOUT MergedAt (active, unmerged branch).
	entityID := "FEAT-01UNMERGED001234"
	_, err := store.Create(worktree.Record{
		EntityID:  entityID,
		Branch:    "test-branch",
		Path:      wtAbsPath,
		Status:    worktree.StatusActive,
		Created:   timeNowUTC(),
		CreatedBy: "tester",
		// MergedAt intentionally nil — unmerged.
	})
	if err != nil {
		t.Fatalf("store.Create: %v", err)
	}

	// Create drift between main and test-branch (55 commits ahead on main).
	setupDriftBetweenMainAndBranch(t, repoDir, "test-branch", 55)

	thresholds := git.BranchThresholds{
		StaleAfterDays:      14,
		DriftWarningCommits: 50,
		DriftErrorCommits:   100,
	}
	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{
		"entity_id": entityID,
	}

	result, err := branchStatusAction(store, repoDir, thresholds, req)
	if err != nil {
		t.Fatalf("branchStatusAction: %v", err)
	}

	resp := branchStatusResult(t, result)

	// AC-005: unmerged branch should show merged=false.
	if merged, ok := resp["merged"].(bool); !ok || merged {
		t.Errorf("merged = %v, want false", resp["merged"])
	}

	// AC-005: drift alerts must be present (not filtered) when there is drift.
	hasDrift := false
	for _, w := range sliceFromResp(resp, "warnings") {
		if strings.Contains(w, "drift") || strings.Contains(w, "Drift") {
			hasDrift = true
		}
	}
	for _, e := range sliceFromResp(resp, "errors") {
		if strings.Contains(e, "drift") || strings.Contains(e, "Drift") {
			hasDrift = true
		}
	}
	if !hasDrift {
		t.Error("unmerged branch with drift should have drift alerts, but none found")
	}
}

// TestBranchStatusAction_MergedSkipsDrift verifies AC-004:
// Merged branch with drift → no drift alerts in warnings or errors.
// Uses a real git repo so EvaluateBranchStatus produces real results.
func TestBranchStatusAction_MergedSkipsDrift(t *testing.T) {
	repoDir, wtAbsPath := setupGitRepoForRemove(t, "wt-merged-drift")
	store := worktree.NewStore(t.TempDir())

	// Create a worktree record with MergedAt set (simulating a merged branch).
	entityID := "FEAT-01MERGEDDRIFT001"
	_, err := store.Create(worktree.Record{
		EntityID:  entityID,
		Branch:    "test-branch",
		Path:      wtAbsPath,
		Status:    worktree.StatusActive,
		Created:   timeNowUTC(),
		CreatedBy: "tester",
		MergedAt:  timeNowPtr(),
	})
	if err != nil {
		t.Fatalf("store.Create: %v", err)
	}

	// Create drift between main and test-branch (60 commits ahead on main).
	setupDriftBetweenMainAndBranch(t, repoDir, "test-branch", 60)

	thresholds := git.BranchThresholds{
		StaleAfterDays:      14,
		DriftWarningCommits: 50,
		DriftErrorCommits:   100,
	}
	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{
		"entity_id": entityID,
	}

	result, err := branchStatusAction(store, repoDir, thresholds, req)
	if err != nil {
		t.Fatalf("branchStatusAction: %v", err)
	}

	resp := branchStatusResult(t, result)

	// AC-004: merged branch should show merged=true.
	if merged, ok := resp["merged"].(bool); !ok || !merged {
		t.Errorf("merged = %v, want true", resp["merged"])
	}

	// AC-004: drift alerts must NOT appear in warnings or errors.
	for _, w := range sliceFromResp(resp, "warnings") {
		if strings.Contains(w, "drift") || strings.Contains(w, "Drift") {
			t.Errorf("merged branch should not have drift warning: %q", w)
		}
	}
	for _, e := range sliceFromResp(resp, "errors") {
		if strings.Contains(e, "drift") || strings.Contains(e, "Drift") {
			t.Errorf("merged branch should not have drift error: %q", e)
		}
	}
}

// ─── helpers ─────────────────────────────────────────────────────────────────

func timeNowUTC() time.Time { return time.Now().UTC() }

func timeNowPtr() *time.Time {
	t := timeNowUTC()
	return &t
}

// setupDriftBetweenMainAndBranch adds n commits to main so that branch is behind.
func setupDriftBetweenMainAndBranch(t *testing.T, repoDir, branch string, n int) {
	t.Helper()
	runGit(t, repoDir, "checkout", "main")
	for i := 0; i < n; i++ {
		filename := fmt.Sprintf("drift-file-%d.txt", i)
		createTestFile(t, repoDir, filename, fmt.Sprintf("content %d", i))
		runGit(t, repoDir, "add", filename)
		runGit(t, repoDir, "commit", "-m", fmt.Sprintf("drift commit %d", i))
	}
}

func createTestFile(t *testing.T, repoDir, name, content string) {
	t.Helper()
	path := filepath.Join(repoDir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
}

func branchStatusResult(t *testing.T, result *mcp.CallToolResult) map[string]any {
	t.Helper()
	if len(result.Content) == 0 {
		t.Fatal("result has no content")
	}
	textContent, ok := result.Content[0].(mcp.TextContent)
	if !ok {
		t.Fatalf("unexpected content type: %T", result.Content[0])
	}
	var resp map[string]any
	if err := json.Unmarshal([]byte(textContent.Text), &resp); err != nil {
		t.Fatalf("unmarshal text content: %v", err)
	}
	return resp
}

func sliceFromResp(resp map[string]any, key string) []string {
	raw, ok := resp[key]
	if !ok {
		return nil
	}
	arr, ok := raw.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(arr))
	for _, v := range arr {
		if s, ok := v.(string); ok {
			out = append(out, s)
		}
	}
	return out
}
