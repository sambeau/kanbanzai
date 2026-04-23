package mcp

// Tests for the concept-tagging approval gate added to doc(action: "approve").
// Covers all 11 ACs from work/spec/p32-concept-tagging-approval-gate.md.
//
// AC-001: policy/report/research types pass through gate
// AC-002: spec with zero classifications → passes
// AC-003: design with concepts_intro populated → passes
// AC-004: specification, classifications, no concepts_intro → blocked
// AC-005: dev-plan, classifications, no concepts_intro → blocked
// AC-006: error content_hash matches classification entry
// AC-007: nil intel service → gate skipped, approval succeeds
// AC-008: GetClassifications returns empty slice (not error) for unknown doc
// AC-009: document status unchanged after blocked approval
// AC-010: retry after classify with concepts_intro succeeds
// AC-011: nil intel service = identical to pre-feature behavior

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/sambeau/kanbanzai/internal/docint"
	"github.com/sambeau/kanbanzai/internal/service"
)

// callDocApproveWithIntel calls doc(action:"approve") with the given
// intelligence service (may be nil to test the nil-guard path, REQ-007).
func callDocApproveWithIntel(t *testing.T, env *docToolEnv, intelSvc *service.IntelligenceService, docID string) map[string]any {
	t.Helper()
	tool := docTool(env.docSvc, intelSvc, nil)
	req := makeRequest(map[string]any{
		"action": "approve",
		"id":     docID,
	})
	result, err := tool.Handler(context.Background(), req)
	if err != nil {
		t.Fatalf("doc approve handler error: %v", err)
	}
	text := extractText(t, result)
	var parsed map[string]any
	if err := json.Unmarshal([]byte(text), &parsed); err != nil {
		t.Fatalf("unmarshal approve response: %v\nraw: %s", err, text)
	}
	return parsed
}

// registerAndIngestDoc creates a file, registers it, and returns the doc ID
// plus the content hash from the classification nudge (populated by auto-ingest).
func registerAndIngestDoc(t *testing.T, env *docToolEnv, relPath, docType, title, content string) (docID, contentHash string) {
	t.Helper()
	writeDocFile(t, env.repoRoot, relPath, content)
	resp := callDocWithIntel(t, env, map[string]any{
		"action": "register",
		"path":   relPath,
		"type":   docType,
		"title":  title,
	})
	doc, ok := resp["document"].(map[string]any)
	if !ok {
		t.Fatalf("register: expected document field, got: %v", resp)
	}
	docID, _ = doc["id"].(string)
	if docID == "" {
		t.Fatalf("register: document.id is empty; response: %v", resp)
	}
	nudge, _ := resp["classification_nudge"].(map[string]any)
	contentHash, _ = nudge["content_hash"].(string)
	// Wait for the background touchDocumentAccess goroutine spawned by
	// GetDocumentIndex (called during nudge-outline construction) to finish
	// writing the access-count update back to the document-index YAML file.
	// Without this, classifyDocForGate may race: classify writes classifications
	// to the YAML, then the still-running goroutine loads the pre-classify
	// snapshot and saves it back, clobbering the classifications so that the
	// gate's GetClassifications call sees an empty slice and skips the block.
	env.intelSvc.Wait()
	return docID, contentHash
}

// classifyDocForGate calls doc_intel(action:"classify") with the given JSON classifications string.
func classifyDocForGate(t *testing.T, env *docToolEnv, docID, contentHash, classificationsJSON string) {
	t.Helper()
	resp := callDocIntelAction(t, env, map[string]any{
		"action":          "classify",
		"id":              docID,
		"content_hash":    contentHash,
		"model_name":      "test-model",
		"model_version":   "1.0",
		"classifications": classificationsJSON,
	})
	if errVal, hasErr := resp["error"]; hasErr {
		t.Fatalf("classify returned error: %v", errVal)
	}
}

// ─── AC-001: non-gated types (policy, report, research) pass through ─────────

func TestApproveGate_AC001_NonGatedTypes(t *testing.T) {
	t.Parallel()
	for _, docType := range []string{"policy", "report", "research"} {
		t.Run(docType, func(t *testing.T) {
			t.Parallel()
			env := setupDocToolTest(t)
			env.docSvc.SetIntelligenceService(env.intelSvc)

			relPath := fmt.Sprintf("work/docs/ac001-%s.md", docType)
			content := fmt.Sprintf("# AC001 %s\n\n## Section\n\nContent here.\n", docType)
			docID, contentHash := registerAndIngestDoc(t, env, relPath, docType, "AC001 "+docType, content)

			if contentHash != "" {
				// Classify with no concepts_intro — would block if this were a gated type.
				classifyDocForGate(t, env, docID, contentHash,
					`[{"section_path":"1","role":"requirement","confidence":"high"}]`)
			}

			resp := callDocApproveWithIntel(t, env, env.intelSvc, docID)

			// Must NOT return concept_tagging_required.
			if errCode, _ := resp["error"].(string); errCode == "concept_tagging_required" {
				t.Errorf("type %q must not be gated, got concept_tagging_required", docType)
			}
			doc, _ := resp["document"].(map[string]any)
			if doc == nil {
				t.Fatalf("expected document in response; got: %v", resp)
			}
			if doc["status"] != "approved" {
				t.Errorf("status = %q, want approved", doc["status"])
			}
		})
	}
}

// ─── AC-002: specification with zero classifications passes ──────────────────

func TestApproveGate_AC002_NoClassifications(t *testing.T) {
	t.Parallel()
	env := setupDocToolTest(t)
	env.docSvc.SetIntelligenceService(env.intelSvc)

	docID, _ := registerAndIngestDoc(t, env, "work/spec/ac002.md", "specification", "AC002",
		"# AC002\n\n## Section\n\nContent.\n")
	// No classify call — zero classification entries.

	resp := callDocApproveWithIntel(t, env, env.intelSvc, docID)

	if errCode, _ := resp["error"].(string); errCode == "concept_tagging_required" {
		t.Errorf("zero classifications must not trigger gate")
	}
	doc, _ := resp["document"].(map[string]any)
	if doc == nil {
		t.Fatalf("expected document in response; got: %v", resp)
	}
	if doc["status"] != "approved" {
		t.Errorf("status = %q, want approved", doc["status"])
	}
}

// ─── AC-003: design with concepts_intro populated passes ─────────────────────

func TestApproveGate_AC003_ConceptsIntroPresent(t *testing.T) {
	t.Parallel()
	env := setupDocToolTest(t)
	env.docSvc.SetIntelligenceService(env.intelSvc)

	docID, contentHash := registerAndIngestDoc(t, env, "work/design/ac003.md", "design", "AC003",
		"# AC003\n\n## Section A\n\nContent.\n")

	if contentHash == "" {
		t.Skip("no content_hash from ingest — intel service not active")
	}

	// Classify with concepts_intro — gate must NOT fire.
	classifyDocForGate(t, env, docID, contentHash,
		`[{"section_path":"1","role":"requirement","confidence":"high","concepts_intro":[{"name":"my-concept"}]}]`)

	resp := callDocApproveWithIntel(t, env, env.intelSvc, docID)

	if errCode, _ := resp["error"].(string); errCode == "concept_tagging_required" {
		t.Errorf("concepts_intro present — gate must not fire")
	}
	doc, _ := resp["document"].(map[string]any)
	if doc == nil {
		t.Fatalf("expected document in response; got: %v", resp)
	}
	if doc["status"] != "approved" {
		t.Errorf("status = %q, want approved", doc["status"])
	}
}

// ─── AC-004: specification, classifications, no concepts_intro → blocked ─────

func TestApproveGate_AC004_SpecificationBlocked(t *testing.T) {
	t.Parallel()
	env := setupDocToolTest(t)
	env.docSvc.SetIntelligenceService(env.intelSvc)

	docID, contentHash := registerAndIngestDoc(t, env, "work/spec/ac004.md", "specification", "AC004",
		"# AC004\n\n## Section\n\nContent.\n")

	if contentHash == "" {
		t.Skip("no content_hash — intel service did not ingest")
	}

	classifyDocForGate(t, env, docID, contentHash,
		`[{"section_path":"1","role":"requirement","confidence":"high"}]`)

	resp := callDocApproveWithIntel(t, env, env.intelSvc, docID)
	errCode, _ := resp["error"].(string)
	if errCode != "concept_tagging_required" {
		t.Fatalf("expected concept_tagging_required gate block; got resp: %v", resp)
	}
	message, _ := resp["message"].(string)
	if !strings.Contains(message, "concepts_intro") {
		t.Errorf("message must mention concepts_intro; got: %q", message)
	}
	if !strings.Contains(message, docID) {
		t.Errorf("message must contain docID %q; got: %q", docID, message)
	}
}

// ─── AC-005: dev-plan, classifications, no concepts_intro → blocked ───────────

func TestApproveGate_AC005_DevPlanBlocked(t *testing.T) {
	t.Parallel()
	env := setupDocToolTest(t)
	env.docSvc.SetIntelligenceService(env.intelSvc)

	// Register dev-plan without auto_approve so it stays as draft.
	writeDocFile(t, env.repoRoot, "work/plan/ac005.md", "# AC005\n\n## Section\n\nContent.\n")
	resp := callDocWithIntel(t, env, map[string]any{
		"action": "register",
		"path":   "work/plan/ac005.md",
		"type":   "dev-plan",
		"title":  "AC005",
	})
	doc, _ := resp["document"].(map[string]any)
	docID, _ := doc["id"].(string)
	if docID == "" {
		t.Fatalf("register: document.id is empty; response: %v", resp)
	}
	nudge, _ := resp["classification_nudge"].(map[string]any)
	contentHash, _ := nudge["content_hash"].(string)
	if contentHash == "" {
		t.Skip("no content_hash — intel service did not ingest")
	}

	// If auto-approved, we can't test the gate. Check current status.
	current, err := env.docSvc.GetDocument(docID, false)
	if err != nil {
		t.Fatalf("GetDocument: %v", err)
	}
	if current.Status == "approved" {
		t.Skip("dev-plan auto-approved during register — gate test requires draft status")
	}

	classifyDocForGate(t, env, docID, contentHash,
		`[{"section_path":"1","role":"requirement","confidence":"high"}]`)

	blockResp := callDocApproveWithIntel(t, env, env.intelSvc, docID)
	blockErrCode, _ := blockResp["error"].(string)
	if blockErrCode != "concept_tagging_required" {
		t.Fatalf("expected concept_tagging_required gate block for dev-plan; got resp: %v", blockResp)
	}
	blockMsg, _ := blockResp["message"].(string)
	if !strings.Contains(blockMsg, docID) {
		t.Errorf("message must contain docID %q; got: %q", docID, blockMsg)
	}
}

// ─── AC-006: error content_hash matches classification entry ─────────────────

func TestApproveGate_AC006_ContentHashMatchesEntry(t *testing.T) {
	t.Parallel()
	env := setupDocToolTest(t)
	env.docSvc.SetIntelligenceService(env.intelSvc)

	docID, contentHash := registerAndIngestDoc(t, env, "work/spec/ac006.md", "specification", "AC006",
		"# AC006\n\n## Section\n\nContent.\n")

	if contentHash == "" {
		t.Skip("no content_hash — intel service did not ingest")
	}

	classifyDocForGate(t, env, docID, contentHash,
		`[{"section_path":"1","role":"requirement","confidence":"high"}]`)

	blockResp := callDocApproveWithIntel(t, env, env.intelSvc, docID)
	if errCode, _ := blockResp["error"].(string); errCode != "concept_tagging_required" {
		t.Fatalf("expected gate to fire; got resp: %v", blockResp)
	}
	responseHash, _ := blockResp["content_hash"].(string)
	if responseHash == "" {
		t.Fatal("content_hash must be non-empty in gate response")
	}

	// Verify hash matches the last entry from GetClassifications.
	entries, err := env.intelSvc.GetClassifications(docID)
	if err != nil {
		t.Fatalf("GetClassifications error: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("expected classification entries in index")
	}
	lastEntry := entries[len(entries)-1]
	if responseHash != lastEntry.ContentHash {
		t.Errorf("content_hash in gate response = %q, want %q (from last classification entry)",
			responseHash, lastEntry.ContentHash)
	}
}

// ─── AC-007: nil intel service — gate skipped ────────────────────────────────

func TestApproveGate_AC007_NilIntelService(t *testing.T) {
	t.Parallel()
	env := setupDocToolTest(t)
	env.docSvc.SetIntelligenceService(env.intelSvc)

	docID, contentHash := registerAndIngestDoc(t, env, "work/spec/ac007.md", "specification", "AC007",
		"# AC007\n\n## Section\n\nContent.\n")

	if contentHash != "" {
		// Classify with no concepts_intro — would block with a real intel service.
		classifyDocForGate(t, env, docID, contentHash,
			`[{"section_path":"1","role":"requirement","confidence":"high"}]`)
	}

	// Pass nil intel service — gate must be skipped entirely (REQ-007).
	resp := callDocApproveWithIntel(t, env, nil, docID)

	if errCode, _ := resp["error"].(string); errCode == "concept_tagging_required" {
		t.Errorf("nil intel service must skip gate; got concept_tagging_required")
	}
	doc, _ := resp["document"].(map[string]any)
	if doc == nil {
		t.Fatalf("expected document in response; got: %v", resp)
	}
	if doc["status"] != "approved" {
		t.Errorf("status = %q, want approved", doc["status"])
	}
}

// ─── AC-008: GetClassifications returns empty slice for unknown doc ───────────

func TestApproveGate_AC008_GetClassificationsEmptySlice(t *testing.T) {
	t.Parallel()
	env := setupDocToolTest(t)

	entries, err := env.intelSvc.GetClassifications("DOC-DOES-NOT-EXIST")
	if err != nil {
		t.Errorf("GetClassifications must return nil error for unknown doc; got: %v", err)
	}
	if entries == nil {
		t.Error("GetClassifications must return non-nil empty slice, not nil")
	}
	if len(entries) != 0 {
		t.Errorf("GetClassifications must return empty slice, got %d entries", len(entries))
	}
}

// ─── AC-009: document status unchanged after blocked approval ────────────────

func TestApproveGate_AC009_StatusUnchangedAfterBlock(t *testing.T) {
	t.Parallel()
	env := setupDocToolTest(t)
	env.docSvc.SetIntelligenceService(env.intelSvc)

	docID, contentHash := registerAndIngestDoc(t, env, "work/spec/ac009.md", "specification", "AC009",
		"# AC009\n\n## Section\n\nContent.\n")

	if contentHash == "" {
		t.Skip("no content_hash — intel service did not ingest")
	}

	classifyDocForGate(t, env, docID, contentHash,
		`[{"section_path":"1","role":"requirement","confidence":"high"}]`)

	// Record status before attempted approval.
	before, err := env.docSvc.GetDocument(docID, false)
	if err != nil {
		t.Fatalf("GetDocument before: %v", err)
	}
	statusBefore := before.Status

	// Attempt approval — gate should block.
	blockResp := callDocApproveWithIntel(t, env, env.intelSvc, docID)
	if errCode, _ := blockResp["error"].(string); errCode != "concept_tagging_required" {
		t.Fatalf("expected gate to fire; got resp: %v", blockResp)
	}

	// Status must be unchanged.
	after, err := env.docSvc.GetDocument(docID, false)
	if err != nil {
		t.Fatalf("GetDocument after: %v", err)
	}
	if after.Status != statusBefore {
		t.Errorf("status changed from %q to %q after blocked approval", statusBefore, after.Status)
	}
}

// ─── AC-010: retry after classify with concepts_intro succeeds ───────────────

func TestApproveGate_AC010_RetrySucceedsAfterClassify(t *testing.T) {
	t.Parallel()
	env := setupDocToolTest(t)
	env.docSvc.SetIntelligenceService(env.intelSvc)

	docID, contentHash := registerAndIngestDoc(t, env, "work/spec/ac010.md", "specification", "AC010",
		"# AC010\n\n## Section\n\nContent.\n")

	if contentHash == "" {
		t.Skip("no content_hash — intel service did not ingest")
	}

	// First classify: no concepts_intro → gate fires.
	classifyDocForGate(t, env, docID, contentHash,
		`[{"section_path":"1","role":"requirement","confidence":"high"}]`)

	resp1 := callDocApproveWithIntel(t, env, env.intelSvc, docID)
	if errCode1, _ := resp1["error"].(string); errCode1 != "concept_tagging_required" {
		t.Fatalf("expected first approve to be blocked; got resp: %v", resp1)
	}

	// Re-classify with concepts_intro.
	classifyDocForGate(t, env, docID, contentHash,
		`[{"section_path":"1","role":"requirement","confidence":"high","concepts_intro":[{"name":"my-concept"}]}]`)

	// Retry approve — must succeed.
	resp2 := callDocApproveWithIntel(t, env, env.intelSvc, docID)
	doc, _ := resp2["document"].(map[string]any)
	if doc == nil {
		t.Fatalf("expected document in retry response; got: %v", resp2)
	}
	if doc["status"] != "approved" {
		t.Errorf("status = %q, want approved", doc["status"])
	}
}

// ─── AC-011: nil intel service = identical to pre-feature behavior ───────────

func TestApproveGate_AC011_NilServiceIdenticalBehavior(t *testing.T) {
	t.Parallel()
	for _, docType := range []string{"specification", "design", "dev-plan", "policy", "report", "research"} {
		t.Run(docType, func(t *testing.T) {
			t.Parallel()
			env := setupDocToolTest(t)
			// No intelligence service set on docSvc.

			relPath := fmt.Sprintf("work/docs/ac011-%s.md", docType)
			writeDocFile(t, env.repoRoot, relPath, "# AC011\n\nContent.\n")
			regResp := callDoc(t, env, map[string]any{
				"action": "register",
				"path":   relPath,
				"type":   docType,
				"title":  "AC011 " + docType,
			})
			doc, _ := regResp["document"].(map[string]any)
			docID, _ := doc["id"].(string)
			if docID == "" {
				t.Fatalf("register: document.id is empty; response: %v", regResp)
			}

			// Pass nil intel service.
			approveResp := callDocApproveWithIntel(t, env, nil, docID)

			if errCode, _ := approveResp["error"].(string); errCode != "" {
				t.Errorf("nil intel service must produce no gate error; got %q", errCode)
			}
			approvedDoc, _ := approveResp["document"].(map[string]any)
			if approvedDoc == nil {
				t.Fatalf("expected document in approve response; got: %v", approveResp)
			}
			if approvedDoc["status"] != "approved" {
				t.Errorf("status = %q, want approved", approvedDoc["status"])
			}
		})
	}
}

// Compile-time checks: ensure the gate test uses real types.
var _ []docint.ClassificationEntry
var _ *service.IntelligenceService

// ─── Batch approve with a gate-blocked document ───────────────────────────────

func TestApproveGate_BatchWithBlockedDoc(t *testing.T) {
	t.Parallel()
	env := setupDocToolTest(t)
	env.docSvc.SetIntelligenceService(env.intelSvc)

	// doc1: gated type, classified with no concepts_intro → will be blocked
	doc1ID, doc1Hash := registerAndIngestDoc(t, env, "work/spec/batch-blocked.md", "specification", "Batch Blocked",
		"# Spec\n\n## Requirements\n\nContent.\n")
	if doc1Hash == "" {
		t.Skip("no content_hash for doc1 — intel service did not ingest")
	}
	classifyDocForGate(t, env, doc1ID, doc1Hash,
		`[{"section_path":"1","role":"requirement","confidence":"high"}]`)

	// doc2: gated type, classified with concepts_intro → will be approved
	doc2ID, doc2Hash := registerAndIngestDoc(t, env, "work/spec/batch-ok.md", "specification", "Batch OK",
		"# Spec\n\n## Requirements\n\nContent.\n")
	if doc2Hash == "" {
		t.Skip("no content_hash for doc2 — intel service did not ingest")
	}
	classifyDocForGate(t, env, doc2ID, doc2Hash,
		`[{"section_path":"1","role":"requirement","confidence":"high","concepts_intro":[{"name":"batch-concept"}]}]`)

	// Batch approve both in one call.
	tool := docTool(env.docSvc, env.intelSvc, nil)
	req := makeRequest(map[string]any{
		"action": "approve",
		"ids":    []any{doc1ID, doc2ID},
	})
	result, err := tool.Handler(context.Background(), req)
	if err != nil {
		t.Fatalf("batch approve handler returned top-level error: %v", err)
	}
	text := extractText(t, result)
	var parsed map[string]any
	if err := json.Unmarshal([]byte(text), &parsed); err != nil {
		t.Fatalf("unmarshal batch response: %v\nraw: %s", err, text)
	}

	// The batch result must show doc1 as an error (gate blocked).
	// ExecuteBatch serialises errors as ItemResult.Error (*ErrorDetail → {"code":..., "message":...}).
	results, _ := parsed["results"].([]any)
	if len(results) != 2 {
		t.Fatalf("expected 2 results in batch result, got %d: %v", len(results), parsed)
	}
	errorCount := 0
	for _, item := range results {
		m, _ := item.(map[string]any)
		if m["error"] != nil {
			errorCount++
			// m["error"] is ErrorDetail marshalled as {"code":"item_error","message":"..."}
			errDetail, _ := m["error"].(map[string]any)
			errMsg, _ := errDetail["message"].(string)
			if !strings.Contains(errMsg, "concept_tagging_required") {
				t.Errorf("blocked doc error message = %q, want to contain concept_tagging_required", errMsg)
			}
		}
	}
	if errorCount != 1 {
		t.Errorf("expected 1 blocked document in batch, got %d error items: %v", errorCount, results)
	}

	// doc2 must be approved despite doc1 being blocked.
	doc2, docErr := env.docSvc.GetDocument(doc2ID, false)
	if docErr != nil {
		t.Fatalf("GetDocument(doc2): %v", docErr)
	}
	if doc2.Status != "approved" {
		t.Errorf("doc2 status = %q, want approved", doc2.Status)
	}

	// doc1 must still be in draft (not approved).
	doc1, docErr := env.docSvc.GetDocument(doc1ID, false)
	if docErr != nil {
		t.Fatalf("GetDocument(doc1): %v", docErr)
	}
	if doc1.Status != "draft" {
		t.Errorf("doc1 status = %q, want draft (gate should have blocked approval)", doc1.Status)
	}
}

// ─── auto_approve is disabled for gated doc types ─────────────────────────────

func TestApproveGate_AutoApproveDisabledForGatedTypes(t *testing.T) {
	t.Parallel()
	for _, docType := range []string{"specification", "design", "dev-plan"} {
		t.Run(docType, func(t *testing.T) {
			t.Parallel()
			env := setupDocToolTest(t)
			// Close the SQLite-backed intelligence service before TempDir cleanup
			// to prevent WAL-file "directory not empty" errors on macOS.
			t.Cleanup(func() { _ = env.intelSvc.Close() })
			env.docSvc.SetIntelligenceService(env.intelSvc)

			relPath := fmt.Sprintf("work/docs/autoapprove-%s.md", docType)
			content := fmt.Sprintf("# AutoApprove %s\n\n## Section\n\nContent.\n", docType)
			writeDocFile(t, env.repoRoot, relPath, content)

			// Register with auto_approve: true for a gated type.
			resp := callDocWithIntel(t, env, map[string]any{
				"action":       "register",
				"path":         relPath,
				"type":         docType,
				"title":        "AutoApprove " + docType,
				"auto_approve": true,
			})
			doc, _ := resp["document"].(map[string]any)
			if doc == nil {
				t.Fatalf("register response missing document; got: %v", resp)
			}
			// Document must be draft, not approved.
			if doc["status"] != "draft" {
				t.Errorf("type %q with auto_approve:true: status = %q, want draft (gate should prevent auto-approval)", docType, doc["status"])
			}
		})
	}
}
