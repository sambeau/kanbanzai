package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/sambeau/kanbanzai/internal/worktree"
)

// ─── helpers ──────────────────────────────────────────────────────────────────

// callAction invokes an ActionHandler directly and returns the parsed response map.
func callAction(t *testing.T, handler ActionHandler, args map[string]any) (map[string]any, error) {
	t.Helper()
	req := mcp.CallToolRequest{}
	req.Params.Arguments = args
	result, err := handler(context.Background(), req)
	if err != nil {
		return nil, err
	}
	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal result: %v", err)
	}
	var resp map[string]any
	if err := json.Unmarshal(data, &resp); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	return resp, nil
}

// lockContendErr simulates a git lock contention error that triggers retries.
var lockContendErr = errors.New("exit status 128: fatal: unable to obtain lock on .git/worktrees/test/lock")

// ─── worktreeAddWithRetry unit tests ─────────────────────────────────────────

// TestWorktreeRetry_TwoFailuresThenSuccess verifies AC-002:
// 2 lock-contention failures followed by success → exactly 3 git add calls,
// backoff delays are 2s then 4s.
func TestWorktreeRetry_TwoFailuresThenSuccess(t *testing.T) {
	// Calls worktreeAddWithRetry directly — no package-level var mutation.
	callCount := 0
	addFn := func(_, _ string) error {
		callCount++
		if callCount < 3 {
			return lockContendErr
		}
		return nil
	}

	var delays []time.Duration
	mockSleep := func(d time.Duration) { delays = append(delays, d) }

	if err := worktreeAddWithRetry(addFn, ".wt/test", "feat/test", mockSleep); err != nil {
		t.Fatalf("expected success on 3rd attempt, got: %v", err)
	}

	if callCount != 3 {
		t.Errorf("call count = %d, want 3", callCount)
	}
	if len(delays) != 2 {
		t.Errorf("sleep count = %d, want 2 (before attempt 2 and 3)", len(delays))
	}
	if len(delays) >= 1 && delays[0] != 2*time.Second {
		t.Errorf("first delay = %v, want 2s", delays[0])
	}
	if len(delays) >= 2 && delays[1] != 4*time.Second {
		t.Errorf("second delay = %v, want 4s", delays[1])
	}
}

// TestWorktreeRetry_FirstAttemptSucceeds verifies AC-003:
// First attempt succeeds → exactly 1 call, no sleep.
func TestWorktreeRetry_FirstAttemptSucceeds(t *testing.T) {
	callCount := 0
	addFn := func(_, _ string) error {
		callCount++
		return nil
	}

	var sleepCalled bool
	mockSleep := func(d time.Duration) { sleepCalled = true }

	if err := worktreeAddWithRetry(addFn, ".wt/test", "feat/test", mockSleep); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if callCount != 1 {
		t.Errorf("call count = %d, want 1", callCount)
	}
	if sleepCalled {
		t.Error("sleep should not be called when first attempt succeeds")
	}
}

// TestWorktreeRetry_ThreeConsecutiveFailures verifies AC-004:
// 3 consecutive lock failures → returned error contains (a) underlying error
// text, (b) "3 attempts", (c) fallback command containing "git worktree add".
func TestWorktreeRetry_ThreeConsecutiveFailures(t *testing.T) {
	underlyingMsg := "unable to obtain lock on .git/worktrees/test/lock"
	addFn := func(path, branch string) error {
		return fmt.Errorf("exit status 128: fatal: %s", underlyingMsg)
	}
	mockSleep := func(d time.Duration) {}

	err := worktreeAddWithRetry(addFn, ".wt/test-path", "feat/test-branch", mockSleep)
	if err == nil {
		t.Fatal("expected error after 3 consecutive failures, got nil")
	}

	errStr := err.Error()

	if !strings.Contains(errStr, underlyingMsg) {
		t.Errorf("error must contain underlying message %q; got: %s", underlyingMsg, errStr)
	}
	if !strings.Contains(errStr, "3 attempts") {
		t.Errorf("error must contain '3 attempts'; got: %s", errStr)
	}
	if !strings.Contains(errStr, "git worktree add") {
		t.Errorf("error must contain 'git worktree add' (fallback command); got: %s", errStr)
	}
}

// TestWorktreeRetry_TotalElapsedUnder30s verifies AC-006:
// Total sleep with 3 lock-failures at max backoff is ≤ 30s (fake clock).
func TestWorktreeRetry_TotalElapsedUnder30s(t *testing.T) {
	addFn := func(_, _ string) error { return lockContendErr }

	var total time.Duration
	mockSleep := func(d time.Duration) { total += d }

	_ = worktreeAddWithRetry(addFn, ".wt/test", "feat/test", mockSleep)

	if total > 30*time.Second {
		t.Errorf("total sleep = %v, exceeds 30s budget", total)
	}
}

// TestWorktreeRetry_NonRetryableErrorFailsImmediately verifies that a
// non-retryable error (e.g., "already exists") returns after exactly 1 call
// without any sleep.
func TestWorktreeRetry_NonRetryableErrorFailsImmediately(t *testing.T) {
	callCount := 0
	addFn := func(_, _ string) error {
		callCount++
		return errors.New("fatal: '/wt/test' already exists")
	}

	var sleepCalled bool
	mockSleep := func(d time.Duration) { sleepCalled = true }

	if err := worktreeAddWithRetry(addFn, ".wt/test", "feat/test", mockSleep); err == nil {
		t.Fatal("expected error, got nil")
	}
	if callCount != 1 {
		t.Errorf("call count = %d, want 1 (no retries for non-retryable errors)", callCount)
	}
	if sleepCalled {
		t.Error("sleep should not be called for non-retryable errors")
	}
}

// TestWorktreeRetry_BackoffDoubles verifies the exponential doubling sequence:
// sleeps are 2s, 4s (exactly double), and no sleep after the final attempt.
func TestWorktreeRetry_BackoffDoubles(t *testing.T) {
	addFn := func(_, _ string) error { return lockContendErr }

	var sleepDurations []time.Duration
	mockSleep := func(d time.Duration) { sleepDurations = append(sleepDurations, d) }

	_ = worktreeAddWithRetry(addFn, ".wt/test", "feat/test", mockSleep)

	if len(sleepDurations) != 2 {
		t.Fatalf("sleep count = %d, want 2 (not after last attempt)", len(sleepDurations))
	}
	if sleepDurations[1] != sleepDurations[0]*2 {
		t.Errorf("second sleep %v != double of first sleep %v", sleepDurations[1], sleepDurations[0])
	}
}

// ─── AC-007: response schema stability ────────────────────────────────────────

// TestWorktreeRetry_GetResponseSchemaUnchanged verifies AC-007 (get):
// worktree(action: get) still returns {"worktree": {...}}.
// No live git repository needed — uses in-memory store.
func TestWorktreeRetry_GetResponseSchemaUnchanged(t *testing.T) {
	t.Parallel()

	store := worktree.NewStore(t.TempDir())
	entityID := "FEAT-01AAAAAAAAAAAA0"
	createTestWorktreeRecord(t, store, entityID, "")

	handler := worktreeGetAction(store)
	result, err := callAction(t, handler, map[string]any{"entity_id": entityID})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, ok := result["worktree"]; !ok {
		t.Errorf("get response schema changed: missing 'worktree' key; got: %v", result)
	}
}

// TestWorktreeRetry_ListResponseSchemaUnchanged verifies AC-007 (list):
// worktree(action: list) still returns {"count": N, "worktrees": [...]}.
func TestWorktreeRetry_ListResponseSchemaUnchanged(t *testing.T) {
	t.Parallel()

	store := worktree.NewStore(t.TempDir())

	handler := worktreeListAction(store)
	result, err := callAction(t, handler, map[string]any{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, ok := result["count"]; !ok {
		t.Errorf("list response schema changed: missing 'count' key; got: %v", result)
	}
	if _, ok := result["worktrees"]; !ok {
		t.Errorf("list response schema changed: missing 'worktrees' key; got: %v", result)
	}
}

// TestWorktreeRetry_RemoveResponseSchemaUnchanged verifies AC-007 (remove):
// worktree(action: remove) still returns {"removed": {"id": ..., "path": ...}}.
// Uses a real git repo created in t.TempDir() — no live 34-worktree repo needed.
func TestWorktreeRetry_RemoveResponseSchemaUnchanged(t *testing.T) {
	t.Parallel()

	repoDir, wtAbsPath := setupGitRepoForRemove(t, "wt-schema-check")
	gitOps := worktree.NewGit(repoDir)
	store := worktree.NewStore(t.TempDir())

	entityID := "FEAT-01SCHEMACHECK001"
	_, err := store.Create(worktree.Record{
		EntityID:  entityID,
		Branch:    "test-branch",
		Path:      wtAbsPath,
		Status:    worktree.StatusActive,
		Created:   time.Now().UTC(),
		CreatedBy: "tester",
	})
	if err != nil {
		t.Fatalf("store.Create: %v", err)
	}

	handler := worktreeRemoveAction(store, gitOps)
	result, err := callAction(t, handler, map[string]any{"entity_id": entityID})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	removed, ok := result["removed"].(map[string]any)
	if !ok {
		t.Fatalf("remove response schema changed: missing 'removed' key; got: %v", result)
	}
	if _, ok := removed["id"]; !ok {
		t.Errorf("remove.removed schema changed: missing 'id'; got: %v", removed)
	}
	if _, ok := removed["path"]; !ok {
		t.Errorf("remove.removed schema changed: missing 'path'; got: %v", removed)
	}
}
