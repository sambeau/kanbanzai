package mcp

// Tests for the structured classification_nudge field returned by doc(action: "register").
// Covers AC-001 through AC-006 and AC-009 (NFR benchmark).
// See: internal/mcp/doc_tool.go — classificationNudge / classificationNudgeSection structs.

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/sambeau/kanbanzai/internal/service"
)

// ─── helpers ─────────────────────────────────────────────────────────────────

// callDocWithIntel invokes the doc tool with the intelligence service wired in
// so that auto-ingest and outline population are active during registration.
func callDocWithIntel(t *testing.T, env *docToolEnv, args map[string]any) map[string]any {
	t.Helper()
	tool := docTool(env.docSvc, env.intelSvc, nil)
	req := makeRequest(args)
	result, err := tool.Handler(context.Background(), req)
	if err != nil {
		t.Fatalf("doc handler error: %v", err)
	}
	text := extractText(t, result)
	var parsed map[string]any
	if err := json.Unmarshal([]byte(text), &parsed); err != nil {
		t.Fatalf("unmarshal response: %v\nraw: %s", err, text)
	}
	return parsed
}

// callDocIntelAction invokes the doc_intel tool with the given args.
func callDocIntelAction(t *testing.T, env *docToolEnv, args map[string]any) map[string]any {
	t.Helper()
	tool := docIntelTool(env.intelSvc, nil, nil)
	req := makeRequest(args)
	result, err := tool.Handler(context.Background(), req)
	if err != nil {
		t.Fatalf("doc_intel handler error: %v", err)
	}
	text := extractText(t, result)
	var parsed map[string]any
	if err := json.Unmarshal([]byte(text), &parsed); err != nil {
		t.Fatalf("unmarshal doc_intel response: %v\nraw: %s", err, text)
	}
	return parsed
}

// makeFiftySectionContent creates a markdown document with 50 level-2 sections.
func makeFiftySectionContent() string {
	var sb strings.Builder
	sb.WriteString("# Fixture Document\n\nOverview paragraph with several introductory words.\n\n")
	for i := 1; i <= 50; i++ {
		sb.WriteString(fmt.Sprintf("## Section %d\n\nContent for section %d with several words here.\n\n", i, i))
	}
	return sb.String()
}

// ─── AC-001: nudge is a JSON object, not a plain string ──────────────────────

func TestDocTool_ClassificationNudge_IsJSONObject(t *testing.T) {
	t.Parallel()
	env := setupDocToolTest(t)
	writeDocFile(t, env.repoRoot, "work/spec/ac001.md", "# AC001\n\nContent.")

	tool := docTool(env.docSvc, nil, nil)
	req := makeRequest(map[string]any{
		"action": "register",
		"path":   "work/spec/ac001.md",
		"type":   "specification",
		"title":  "AC-001 Test",
	})
	res, err := tool.Handler(context.Background(), req)
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}
	text := extractText(t, res)

	// Check via raw JSON that the field is an object ('{'), not a string ('"').
	var raw map[string]json.RawMessage
	if err := json.Unmarshal([]byte(text), &raw); err != nil {
		t.Fatalf("unmarshal raw response: %v", err)
	}
	nudgeRaw, ok := raw["classification_nudge"]
	if !ok {
		t.Fatal("classification_nudge field missing from response")
	}
	trimmed := strings.TrimSpace(string(nudgeRaw))
	if len(trimmed) == 0 || trimmed[0] != '{' {
		t.Errorf("classification_nudge is not a JSON object; got: %s", trimmed)
	}

	// All three required keys must be present.
	var nudge map[string]any
	if err := json.Unmarshal(nudgeRaw, &nudge); err != nil {
		t.Fatalf("unmarshal classification_nudge object: %v", err)
	}
	for _, key := range []string{"message", "content_hash", "outline"} {
		if _, has := nudge[key]; !has {
			t.Errorf("classification_nudge missing required key %q", key)
		}
	}
}

// ─── AC-002: message equals the previous instructional string ────────────────

func TestDocTool_ClassificationNudge_MessageExactFormat(t *testing.T) {
	t.Parallel()
	env := setupDocToolTest(t)
	writeDocFile(t, env.repoRoot, "work/spec/ac002.md", "# AC002\n\nContent.")

	resp := callDoc(t, env, map[string]any{
		"action": "register",
		"path":   "work/spec/ac002.md",
		"type":   "specification",
		"title":  "AC-002 Test",
	})

	doc, _ := resp["document"].(map[string]any)
	docID, _ := doc["id"].(string)
	if docID == "" {
		t.Fatal("document.id missing")
	}
	nudge, ok := resp["classification_nudge"].(map[string]any)
	if !ok {
		t.Fatal("classification_nudge not an object")
	}
	msg, _ := nudge["message"].(string)
	want := fmt.Sprintf(
		"Layer 3 classification pending for %s.\nCall doc_intel(action: \"guide\", id: \"%s\") then read the section outline.\nThen call doc_intel(action: \"classify\", id: \"%s\", content_hash: \"...\", ...) to classify.",
		docID, docID, docID,
	)
	if msg != want {
		t.Errorf("message mismatch:\n got: %q\nwant: %q", msg, want)
	}
}

// ─── AC-003: content_hash passes to classify without hash-mismatch ───────────

func TestDocTool_ClassificationNudge_ContentHashPassesClassify(t *testing.T) {
	t.Parallel()
	env := setupDocToolTest(t)
	env.docSvc.SetIntelligenceService(env.intelSvc)

	writeDocFile(t, env.repoRoot, "work/spec/ac003.md", "# AC003\n\nContent.\n\n## Section\n\nWords here.")
	resp := callDocWithIntel(t, env, map[string]any{
		"action": "register",
		"path":   "work/spec/ac003.md",
		"type":   "specification",
		"title":  "AC-003 Test",
	})

	doc, _ := resp["document"].(map[string]any)
	docID, _ := doc["id"].(string)
	if docID == "" {
		t.Fatal("document.id missing")
	}
	nudge, ok := resp["classification_nudge"].(map[string]any)
	if !ok {
		t.Fatal("classification_nudge not an object")
	}
	contentHash, _ := nudge["content_hash"].(string)
	if contentHash == "" {
		t.Fatal("content_hash is empty in nudge")
	}

	classifyResp := callDocIntelAction(t, env, map[string]any{
		"action":          "classify",
		"id":              docID,
		"content_hash":    contentHash,
		"model_name":      "test-model",
		"model_version":   "1.0",
		"classifications": "[]",
	})

	if errVal, hasErr := classifyResp["error"]; hasErr {
		t.Errorf("classify returned error (hash mismatch or other): %v", errVal)
	}
	if msg, _ := classifyResp["message"].(string); !strings.Contains(msg, "Classifications applied") {
		t.Errorf("unexpected classify response: %v", classifyResp)
	}
}

// ─── AC-004: outline deep-equals doc_intel guide response sections ────────────

func TestDocTool_ClassificationNudge_OutlineMatchesGuide(t *testing.T) {
	t.Parallel()
	env := setupDocToolTest(t)
	env.docSvc.SetIntelligenceService(env.intelSvc)

	content := "# Title\n\nIntro.\n\n## Section A\n\nContent A.\n\n## Section B\n\nContent B.\n"
	writeDocFile(t, env.repoRoot, "work/spec/ac004.md", content)

	resp := callDocWithIntel(t, env, map[string]any{
		"action": "register",
		"path":   "work/spec/ac004.md",
		"type":   "specification",
		"title":  "AC-004 Test",
	})

	doc, _ := resp["document"].(map[string]any)
	docID, _ := doc["id"].(string)
	if docID == "" {
		t.Fatal("document.id missing")
	}
	nudge, ok := resp["classification_nudge"].(map[string]any)
	if !ok {
		t.Fatal("classification_nudge not an object")
	}
	nudgeOutline, _ := nudge["outline"].([]any)
	if len(nudgeOutline) == 0 {
		t.Fatal("nudge.outline is empty; check intelligence service auto-ingest")
	}

	guideResp := callDocIntelAction(t, env, map[string]any{
		"action": "guide",
		"id":     docID,
	})
	guideOutline, _ := guideResp["outline"].([]any)

	if len(nudgeOutline) != len(guideOutline) {
		t.Fatalf("outline length mismatch: nudge=%d, guide=%d", len(nudgeOutline), len(guideOutline))
	}
	for i, ns := range nudgeOutline {
		nudgeSec, _ := ns.(map[string]any)
		guideSec, _ := guideOutline[i].(map[string]any)
		if nudgeSec["path"] != guideSec["path"] {
			t.Errorf("[%d] path: nudge=%v guide=%v", i, nudgeSec["path"], guideSec["path"])
		}
		if nudgeSec["title"] != guideSec["title"] {
			t.Errorf("[%d] title: nudge=%v guide=%v", i, nudgeSec["title"], guideSec["title"])
		}
		if nudgeSec["level"] != guideSec["level"] {
			t.Errorf("[%d] level: nudge=%v guide=%v", i, nudgeSec["level"], guideSec["level"])
		}
		if _, has := nudgeSec["word_count"]; !has {
			t.Errorf("[%d] nudge section missing word_count field", i)
		}
	}
}

// ─── AC-005: batch register 3 docs, each has structured nudge ────────────────

func TestDocTool_ClassificationNudge_Batch3Docs(t *testing.T) {
	t.Parallel()
	env := setupDocToolTest(t)

	paths := []string{"work/spec/b3a.md", "work/spec/b3b.md", "work/spec/b3c.md"}
	for i, p := range paths {
		writeDocFile(t, env.repoRoot, p, fmt.Sprintf("# Doc %d\n\nContent.\n", i+1))
	}
	docs := make([]any, len(paths))
	for i, p := range paths {
		docs[i] = map[string]any{
			"path":  p,
			"type":  "specification",
			"title": fmt.Sprintf("Batch Doc %d", i+1),
		}
	}

	resp := callDoc(t, env, map[string]any{
		"action":    "register",
		"documents": docs,
	})

	results, ok := resp["results"].([]any)
	if !ok || len(results) != 3 {
		t.Fatalf("expected 3 results, got: %v", resp)
	}
	for i, r := range results {
		item, _ := r.(map[string]any)
		if item["status"] != "ok" {
			t.Errorf("result[%d] status = %v, want ok", i, item["status"])
			continue
		}
		data, _ := item["data"].(map[string]any)
		if data == nil {
			t.Errorf("result[%d] missing data field", i)
			continue
		}
		nudge, ok := data["classification_nudge"].(map[string]any)
		if !ok || nudge == nil {
			t.Errorf("result[%d] classification_nudge missing or not an object", i)
			continue
		}
		for _, key := range []string{"message", "content_hash", "outline"} {
			if _, has := nudge[key]; !has {
				t.Errorf("result[%d] nudge missing key %q", i, key)
			}
		}
		msg, _ := nudge["message"].(string)
		if msg == "" {
			t.Errorf("result[%d] nudge.message is empty", i)
		}
		docObj, _ := data["document"].(map[string]any)
		docID, _ := docObj["id"].(string)
		if docID != "" && !strings.Contains(msg, docID) {
			t.Errorf("result[%d] nudge.message %q does not contain docID %q", i, msg, docID)
		}
	}
}

// ─── AC-006: register + classify in 2 calls, no intermediate guide call ───────

func TestDocTool_ClassificationNudge_RegisterThenClassifyNoGuide(t *testing.T) {
	t.Parallel()
	env := setupDocToolTest(t)
	env.docSvc.SetIntelligenceService(env.intelSvc)

	writeDocFile(t, env.repoRoot, "work/spec/ac006.md", "# AC006\n\nIntro.\n\n## Section\n\nContent here.\n")

	// Call 1: register — extract content_hash from nudge.
	resp := callDocWithIntel(t, env, map[string]any{
		"action": "register",
		"path":   "work/spec/ac006.md",
		"type":   "specification",
		"title":  "AC-006 Test",
	})
	doc, _ := resp["document"].(map[string]any)
	docID, _ := doc["id"].(string)
	if docID == "" {
		t.Fatal("document.id missing")
	}
	nudge, ok := resp["classification_nudge"].(map[string]any)
	if !ok {
		t.Fatal("classification_nudge not an object")
	}
	contentHash, _ := nudge["content_hash"].(string)
	if contentHash == "" {
		t.Fatal("content_hash empty in nudge")
	}

	// Call 2: classify using only values from the register response; no guide call.
	classifyResp := callDocIntelAction(t, env, map[string]any{
		"action":          "classify",
		"id":              docID,
		"content_hash":    contentHash,
		"model_name":      "test-model",
		"model_version":   "1.0",
		"classifications": "[]",
	})

	if errVal, hasErr := classifyResp["error"]; hasErr {
		t.Errorf("classify returned error (2-call workflow failed): %v", errVal)
	}
	if _, ok := classifyResp["document_id"]; !ok {
		t.Errorf("classify response missing document_id; got: %v", classifyResp)
	}
}

// ─── AC-009 (NFR): p99 latency benchmark — 50-section fixture ────────────────

// benchP99 collects per-iteration timings and returns the p99 value.
func benchP99(times []time.Duration) time.Duration {
	if len(times) == 0 {
		return 0
	}
	sorted := make([]time.Duration, len(times))
	copy(sorted, times)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })
	return sorted[int(float64(len(sorted))*0.99)]
}

// BenchmarkDocTool_Register_50Sections measures the p99 latency delta between
// baseline (nil intel svc passed to docTool) and enhanced (intel svc passed to
// docTool, triggering GetDocumentIndex + buildNudgeSections) for a 50-section
// document. Both runs use identical docSvc+intelSvc setup so that
// IngestDocument overhead is equal in both; the delta isolates only the
// GetDocumentIndex+buildNudgeSections cost. REQ-NF-001 requires delta ≤ 50 ms.
// Auto-commit is disabled to isolate register-handler overhead.
func BenchmarkDocTool_Register_50Sections(b *testing.B) {
	// Disable auto-commit noise so timings reflect only register-handler work.
	orig := docCommitPathsFunc
	docCommitPathsFunc = func(_, _ string, _ ...string) (bool, error) { return false, nil }
	defer func() { docCommitPathsFunc = orig }()

	content := makeFiftySectionContent()

	// mkTmpDir creates a temp dir and schedules cleanup via os.RemoveAll (not
	// b.TempDir) to avoid the SQLite WAL-file cleanup race on macOS.
	mkTmpDir := func() string {
		dir, err := os.MkdirTemp("", "bench-nudge-*")
		if err != nil {
			b.Fatalf("MkdirTemp: %v", err)
		}
		b.Cleanup(func() { os.RemoveAll(dir) })
		return dir
	}

	// Both runs share the same intelSvc so IngestDocument overhead is identical.
	// The only difference is whether docTool receives the intel svc (and thus
	// calls GetDocumentIndex + buildNudgeSections to populate the nudge outline).
	stateRootBase := mkTmpDir()
	repoRootBase := mkTmpDir()
	indexRootBase := mkTmpDir()
	docSvcBase := service.NewDocumentService(stateRootBase, repoRootBase)
	intelSvcBase := service.NewIntelligenceService(indexRootBase, repoRootBase)
	docSvcBase.SetIntelligenceService(intelSvcBase)

	stateRootEnh := mkTmpDir()
	repoRootEnh := mkTmpDir()
	indexRootEnh := mkTmpDir()
	docSvcEnh := service.NewDocumentService(stateRootEnh, repoRootEnh)
	intelSvcEnh := service.NewIntelligenceService(indexRootEnh, repoRootEnh)
	docSvcEnh.SetIntelligenceService(intelSvcEnh)

	// Pre-create all fixture files so file I/O is excluded from timed loop.
	for i := 0; i < b.N; i++ {
		for _, root := range []string{repoRootBase, repoRootEnh} {
			fp := filepath.Join(root, fmt.Sprintf("work/spec/bench-%d.md", i))
			_ = os.MkdirAll(filepath.Dir(fp), 0o755)
			_ = os.WriteFile(fp, []byte(content), 0o644)
		}
	}

	runRegister := func(docSvc *service.DocumentService, toolIntelSvc *service.IntelligenceService, offset int) []time.Duration {
		times := make([]time.Duration, 0, b.N)
		for i := 0; i < b.N; i++ {
			docPath := fmt.Sprintf("work/spec/bench-%d.md", offset+i)
			tool := docTool(docSvc, toolIntelSvc, nil)
			req := makeRequest(map[string]any{
				"action": "register",
				"path":   docPath,
				"type":   "specification",
				"title":  fmt.Sprintf("Bench Doc %d", offset+i),
			})
			start := time.Now()
			tool.Handler(context.Background(), req) //nolint:errcheck
			times = append(times, time.Since(start))
		}
		return times
	}

	// Baseline: docTool receives nil intel svc — no GetDocumentIndex call.
	baselineTimes := runRegister(docSvcBase, nil, 0)
	// Enhanced: docTool receives intel svc — calls GetDocumentIndex + buildNudgeSections.
	enhancedTimes := runRegister(docSvcEnh, intelSvcEnh, 0)

	b.StopTimer()

	baselineP99 := benchP99(baselineTimes)
	enhancedP99 := benchP99(enhancedTimes)
	delta := enhancedP99 - baselineP99
	if delta < 0 {
		delta = 0
	}

	b.ReportMetric(float64(baselineP99.Milliseconds()), "baseline-p99-ms")
	b.ReportMetric(float64(enhancedP99.Milliseconds()), "enhanced-p99-ms")
	b.ReportMetric(float64(delta.Milliseconds()), "delta-p99-ms")

	if len(baselineTimes) >= 100 && len(enhancedTimes) >= 100 {
		if delta > 50*time.Millisecond {
			b.Errorf("p99 latency delta %v exceeds 50ms budget (baseline=%v, enhanced=%v)",
				delta, baselineP99, enhancedP99)
		}
	}
}
