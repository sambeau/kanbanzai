package mcp_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/mark3labs/mcp-go/mcptest"

	kbzmcp "kanbanzai/internal/mcp"
	"kanbanzai/internal/service"
)

// setupDocRecordServer creates a test MCP server with only DocRecordTools registered.
// It returns the test environment and the repoRoot path so tests can create document files.
func setupDocRecordServer(t *testing.T) (*testEnv, string) {
	t.Helper()
	repoRoot := t.TempDir()
	stateRoot := t.TempDir()
	indexRoot := t.TempDir()

	docSvc := service.NewDocumentService(stateRoot, repoRoot)
	intelSvc := service.NewIntelligenceService(indexRoot, repoRoot)
	docSvc.SetIntelligenceService(intelSvc)

	tools := kbzmcp.DocRecordTools(docSvc)
	ts, err := mcptest.NewServer(t, tools...)
	if err != nil {
		t.Fatalf("start doc record test server: %v", err)
	}
	return &testEnv{server: ts}, repoRoot
}

// submitTestDoc creates a file at relPath inside repoRoot and registers it via
// doc_record_submit. Returns the document record ID.
func submitTestDoc(t *testing.T, env *testEnv, repoRoot, relPath, content string) string {
	t.Helper()
	fullPath := filepath.Join(repoRoot, relPath)
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		t.Fatalf("create parent dirs for %s: %v", relPath, err)
	}
	if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write test doc %s: %v", relPath, err)
	}

	result := callTool(t, env, "doc_record_submit", map[string]any{
		"path":       relPath,
		"type":       "design",
		"title":      "Test Document",
		"created_by": "tester",
	})
	if result.IsError {
		t.Fatalf("doc_record_submit returned error: %s", resultText(t, result))
	}

	var parsed map[string]any
	if err := json.Unmarshal([]byte(resultText(t, result)), &parsed); err != nil {
		t.Fatalf("parse doc_record_submit result: %v", err)
	}
	doc, ok := parsed["document"].(map[string]any)
	if !ok {
		t.Fatalf("expected 'document' in submit result, got: %v", parsed)
	}
	id, ok := doc["id"].(string)
	if !ok || id == "" {
		t.Fatalf("expected non-empty document id, got: %v", doc["id"])
	}
	return id
}

// AC-13: Calling doc_record_refresh on a document whose file has not changed
// returns success with changed=false and does not modify status or timestamp.
func TestDocRecordRefresh_UnchangedFile_ReturnsChangedFalse(t *testing.T) {
	env, repoRoot := setupDocRecordServer(t)
	defer env.server.Close()

	docID := submitTestDoc(t, env, repoRoot, "docs/test.md", "# Test\n\nOriginal content.\n")

	result := callTool(t, env, "doc_record_refresh", map[string]any{"id": docID})
	if result.IsError {
		t.Fatalf("doc_record_refresh returned error: %s", resultText(t, result))
	}

	var resp map[string]any
	if err := json.Unmarshal([]byte(resultText(t, result)), &resp); err != nil {
		t.Fatalf("parse refresh response: %v", err)
	}

	if resp["success"] != true {
		t.Errorf("expected success=true, got %v", resp["success"])
	}
	if resp["changed"] != false {
		t.Errorf("expected changed=false for unchanged file, got %v", resp["changed"])
	}
	if resp["id"] != docID {
		t.Errorf("expected id=%q, got %v", docID, resp["id"])
	}
	// Status must remain draft (was never approved)
	if resp["status"] != "draft" {
		t.Errorf("expected status=draft, got %v", resp["status"])
	}
	// old_hash and new_hash must both be present and equal
	oldHash, _ := resp["old_hash"].(string)
	newHash, _ := resp["new_hash"].(string)
	if oldHash == "" {
		t.Error("expected non-empty old_hash")
	}
	if newHash == "" {
		t.Error("expected non-empty new_hash")
	}
	if oldHash != newHash {
		t.Errorf("expected old_hash == new_hash for unchanged file, got old=%s new=%s", oldHash, newHash)
	}
	// Must not have a status_transition
	if _, hasTransition := resp["status_transition"]; hasTransition {
		t.Errorf("expected no status_transition for unchanged file, got %v", resp["status_transition"])
	}
}

// AC-11: Calling doc_record_refresh after editing the file updates the stored
// content hash and returns both old and new hashes.
func TestDocRecordRefresh_ChangedFile_UpdatesHash(t *testing.T) {
	env, repoRoot := setupDocRecordServer(t)
	defer env.server.Close()

	relPath := "docs/changeable.md"
	docID := submitTestDoc(t, env, repoRoot, relPath, "# Original\n\nVersion 1.\n")

	// Record the hash that was stored at submit time.
	getResult := callTool(t, env, "doc_record_get", map[string]any{
		"id":          docID,
		"check_drift": false,
	})
	if getResult.IsError {
		t.Fatalf("doc_record_get returned error: %s", resultText(t, getResult))
	}
	var getResp map[string]any
	if err := json.Unmarshal([]byte(resultText(t, getResult)), &getResp); err != nil {
		t.Fatalf("parse doc_record_get response: %v", err)
	}
	docFields, _ := getResp["document"].(map[string]any)
	submittedHash, _ := docFields["content_hash"].(string)
	if submittedHash == "" {
		t.Fatal("submitted hash is empty")
	}

	// Modify the file.
	fullPath := filepath.Join(repoRoot, relPath)
	if err := os.WriteFile(fullPath, []byte("# Modified\n\nVersion 2 — content changed.\n"), 0o644); err != nil {
		t.Fatalf("modify test file: %v", err)
	}

	// Refresh.
	result := callTool(t, env, "doc_record_refresh", map[string]any{"id": docID})
	if result.IsError {
		t.Fatalf("doc_record_refresh returned error: %s", resultText(t, result))
	}

	var resp map[string]any
	if err := json.Unmarshal([]byte(resultText(t, result)), &resp); err != nil {
		t.Fatalf("parse refresh response: %v", err)
	}

	if resp["success"] != true {
		t.Errorf("expected success=true, got %v", resp["success"])
	}
	if resp["changed"] != true {
		t.Errorf("expected changed=true after file edit, got %v", resp["changed"])
	}
	oldHash, _ := resp["old_hash"].(string)
	newHash, _ := resp["new_hash"].(string)
	if oldHash == "" {
		t.Error("expected non-empty old_hash")
	}
	if newHash == "" {
		t.Error("expected non-empty new_hash")
	}
	if oldHash == newHash {
		t.Errorf("expected old_hash != new_hash after content change, both are %q", oldHash)
	}
	if oldHash != submittedHash {
		t.Errorf("old_hash=%q does not match hash at submission time %q", oldHash, submittedHash)
	}
	// status stays draft (was never approved)
	if resp["status"] != "draft" {
		t.Errorf("expected status=draft, got %v", resp["status"])
	}

	// A second refresh of the same (now updated) file must return changed=false.
	result2 := callTool(t, env, "doc_record_refresh", map[string]any{"id": docID})
	if result2.IsError {
		t.Fatalf("second doc_record_refresh returned error: %s", resultText(t, result2))
	}
	var resp2 map[string]any
	if err := json.Unmarshal([]byte(resultText(t, result2)), &resp2); err != nil {
		t.Fatalf("parse second refresh response: %v", err)
	}
	if resp2["changed"] != false {
		t.Errorf("expected changed=false on second refresh (no further edit), got %v", resp2["changed"])
	}
}

// AC-12: Calling doc_record_refresh on an approved document whose file content
// has changed transitions the record from approved to draft and communicates
// the transition clearly in the response.
func TestDocRecordRefresh_ApprovedDocChanged_TransitionsToDraft(t *testing.T) {
	env, repoRoot := setupDocRecordServer(t)
	defer env.server.Close()

	relPath := "docs/approved.md"
	docID := submitTestDoc(t, env, repoRoot, relPath, "# Approved Doc\n\nOriginal approved content.\n")

	// Approve the document.
	approveResult := callTool(t, env, "doc_record_approve", map[string]any{
		"id":          docID,
		"approved_by": "tester",
	})
	if approveResult.IsError {
		t.Fatalf("doc_record_approve returned error: %s", resultText(t, approveResult))
	}
	var approveResp map[string]any
	if err := json.Unmarshal([]byte(resultText(t, approveResult)), &approveResp); err != nil {
		t.Fatalf("parse approve response: %v", err)
	}
	approvedDoc, _ := approveResp["document"].(map[string]any)
	if approvedDoc["status"] != "approved" {
		t.Fatalf("expected document to be approved, got status=%v", approvedDoc["status"])
	}

	// Modify the file.
	fullPath := filepath.Join(repoRoot, relPath)
	if err := os.WriteFile(fullPath, []byte("# Approved Doc\n\nContent changed after approval.\n"), 0o644); err != nil {
		t.Fatalf("modify approved doc file: %v", err)
	}

	// Refresh the document.
	result := callTool(t, env, "doc_record_refresh", map[string]any{"id": docID})
	if result.IsError {
		t.Fatalf("doc_record_refresh returned error: %s", resultText(t, result))
	}

	var resp map[string]any
	if err := json.Unmarshal([]byte(resultText(t, result)), &resp); err != nil {
		t.Fatalf("parse refresh response: %v", err)
	}

	if resp["success"] != true {
		t.Errorf("expected success=true, got %v", resp["success"])
	}
	if resp["changed"] != true {
		t.Errorf("expected changed=true, got %v", resp["changed"])
	}

	// Status must have transitioned to draft — not remain approved.
	if resp["status"] != "draft" {
		t.Errorf("expected status=draft after refresh of approved doc with changed content, got %v", resp["status"])
	}

	// The response must contain a status_transition field that communicates the change.
	transition, _ := resp["status_transition"].(string)
	if transition == "" {
		t.Error("expected non-empty status_transition when approved doc is refreshed with changed content")
	}
	// Must mention both approved and draft.
	if !containsAll(transition, "approved", "draft") {
		t.Errorf("status_transition %q should mention both 'approved' and 'draft'", transition)
	}

	// The response must include an explanatory message.
	msg, _ := resp["message"].(string)
	if msg == "" {
		t.Error("expected non-empty message explaining the status transition")
	}

	// Old and new hashes must be present and different.
	oldHash, _ := resp["old_hash"].(string)
	newHash, _ := resp["new_hash"].(string)
	if oldHash == "" || newHash == "" {
		t.Errorf("expected non-empty old_hash and new_hash, got old=%q new=%q", oldHash, newHash)
	}
	if oldHash == newHash {
		t.Errorf("expected old_hash != new_hash after content change, both are %q", oldHash)
	}

	// Verify by fetching the record directly: status must be draft.
	getResult := callTool(t, env, "doc_record_get", map[string]any{
		"id":          docID,
		"check_drift": false,
	})
	if getResult.IsError {
		t.Fatalf("doc_record_get returned error: %s", resultText(t, getResult))
	}
	var getResp map[string]any
	if err := json.Unmarshal([]byte(resultText(t, getResult)), &getResp); err != nil {
		t.Fatalf("parse doc_record_get response: %v", err)
	}
	docFields, _ := getResp["document"].(map[string]any)
	if docFields["status"] != "draft" {
		t.Errorf("doc_record_get: expected status=draft after refresh, got %v", docFields["status"])
	}
	if docFields["content_hash"] != newHash {
		t.Errorf("doc_record_get: expected content_hash=%q after refresh, got %v", newHash, docFields["content_hash"])
	}
}

// TestDocRecordRefresh_RecordNotFound returns an error with a helpful message
// when the document ID does not exist.
func TestDocRecordRefresh_RecordNotFound(t *testing.T) {
	env, _ := setupDocRecordServer(t)
	defer env.server.Close()

	result := callTool(t, env, "doc_record_refresh", map[string]any{
		"id": "PROJECT/does-not-exist",
	})
	if !result.IsError {
		t.Fatalf("expected error for non-existent document ID, got success: %s", resultText(t, result))
	}
	msg := resultText(t, result)
	if !containsAll(msg, "no document record found") && !containsAll(msg, "not found") {
		t.Errorf("error message %q should indicate that the record was not found", msg)
	}
}

// TestDocRecordRefresh_FileNotFound returns an error when the registered file
// no longer exists on disk.
func TestDocRecordRefresh_FileNotFound(t *testing.T) {
	env, repoRoot := setupDocRecordServer(t)
	defer env.server.Close()

	relPath := "docs/to-be-deleted.md"
	docID := submitTestDoc(t, env, repoRoot, relPath, "# Will be deleted.\n")

	// Delete the file.
	if err := os.Remove(filepath.Join(repoRoot, relPath)); err != nil {
		t.Fatalf("remove test file: %v", err)
	}

	result := callTool(t, env, "doc_record_refresh", map[string]any{"id": docID})
	if !result.IsError {
		t.Fatalf("expected error when file has been deleted, got success: %s", resultText(t, result))
	}
	msg := resultText(t, result)
	if !containsAll(msg, "no longer exists") && !containsAll(msg, "not found") {
		t.Errorf("error message %q should indicate the file no longer exists", msg)
	}
}

// containsAll reports whether s contains all of the given substrings
// (case-insensitive).
func containsAll(s string, substrings ...string) bool {
	lower := toLower(s)
	for _, sub := range substrings {
		if !containsString(lower, toLower(sub)) {
			return false
		}
	}
	return true
}

func containsString(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 || indexString(s, sub) >= 0)
}

func indexString(s, sub string) int {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

func toLower(s string) string {
	b := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		b[i] = c
	}
	return string(b)
}
