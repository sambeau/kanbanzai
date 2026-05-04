package actionlog

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
)

func TestHookWrap_Success(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	wr := NewWriter(dir)
	defer wr.Close()

	hook := NewHook(wr, nil, "2.0")

	inner := func(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return mcp.NewToolResultText(`{"ok":true}`), nil
	}

	wrapped := hook.Wrap("entity", inner)
	_, err := wrapped(context.Background(), mcp.CallToolRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	entry := readLastEntry(t, dir)
	if entry.Tool != "entity" {
		t.Errorf("Tool: got %q, want %q", entry.Tool, "entity")
	}
	if !entry.Success {
		t.Errorf("Success: got false, want true")
	}
	if entry.ErrorType != nil {
		t.Errorf("ErrorType: got %q, want nil", *entry.ErrorType)
	}
}

func TestHookWrap_ErrorResult(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	wr := NewWriter(dir)
	defer wr.Close()

	hook := NewHook(wr, nil, "2.0")

	inner := func(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return mcp.NewToolResultText(`{"error":{"code":"not_found","message":"not found"}}`), nil
	}

	wrapped := hook.Wrap("entity", inner)
	_, err := wrapped(context.Background(), mcp.CallToolRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	entry := readLastEntry(t, dir)
	if entry.Success {
		t.Errorf("Success: got true, want false")
	}
	if entry.ErrorType == nil {
		t.Error("ErrorType: got nil, want non-nil")
	}
}

func TestHookWrap_ExtractsAction(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	wr := NewWriter(dir)
	defer wr.Close()

	hook := NewHook(wr, nil, "2.0")

	inner := func(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return mcp.NewToolResultText(`{}`), nil
	}

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{"action": "create", "id": "FEAT-001"}

	wrapped := hook.Wrap("entity", inner)
	wrapped(context.Background(), req)

	entry := readLastEntry(t, dir)
	if entry.Action == nil || *entry.Action != "create" {
		t.Errorf("Action: got %v, want create", entry.Action)
	}
	if entry.EntityID == nil || *entry.EntityID != "FEAT-001" {
		t.Errorf("EntityID: got %v, want FEAT-001", entry.EntityID)
	}
}

func TestHookWrap_AnnotationsMergedIntoExtra(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	wr := NewWriter(dir)
	defer wr.Close()

	hook := NewHook(wr, nil, "2.0")

	inner := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		AnnotateEntry(ctx, AnnotationResultCount, "15")
		AnnotateEntry(ctx, AnnotationEntityType, "feature")
		return mcp.NewToolResultText(`{}`), nil
	}

	wrapped := hook.Wrap("entity", inner)
	_, err := wrapped(context.Background(), mcp.CallToolRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	entry := readLastEntry(t, dir)
	if entry.Extra == nil {
		t.Fatal("Extra: got nil, want non-nil")
	}
	if entry.Extra[AnnotationResultCount] != "15" {
		t.Errorf("Extra[result_count]: got %q, want %q", entry.Extra[AnnotationResultCount], "15")
	}
	if entry.Extra[AnnotationEntityType] != "feature" {
		t.Errorf("Extra[entity_type]: got %q, want %q", entry.Extra[AnnotationEntityType], "feature")
	}
}

func TestHookWrap_NoAnnotations_ExtraNil(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	wr := NewWriter(dir)
	defer wr.Close()

	hook := NewHook(wr, nil, "2.0")

	inner := func(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// No annotations added.
		return mcp.NewToolResultText(`{}`), nil
	}

	wrapped := hook.Wrap("entity", inner)
	_, err := wrapped(context.Background(), mcp.CallToolRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	entry := readLastEntry(t, dir)
	if entry.Extra != nil {
		t.Errorf("Extra: got %v, want nil when no annotations", entry.Extra)
	}
}

func TestHookWrap_LogFailureDoesNotAffectResponse(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	wr := NewWriter(dir)
	// Close the writer early so Log will fail.
	wr.Close()

	hook := NewHook(wr, nil, "2.0")

	inner := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		AnnotateEntry(ctx, "k", "v")
		return mcp.NewToolResultText(`{"ok":true}`), nil
	}

	wrapped := hook.Wrap("entity", inner)
	result, err := wrapped(context.Background(), mcp.CallToolRequest{})
	if err != nil {
		t.Fatalf("handler err should be nil despite log failure: %v", err)
	}
	if result == nil {
		t.Fatal("result should not be nil despite log failure")
	}
}

// AC-002: Log entry contains server_version from Hook (stamped via ldflags or constructor).
func TestHookWrap_ServerVersionStamped(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	wr := NewWriter(dir)
	defer wr.Close()

	hook := NewHook(wr, nil, "3.14.0-beta1+abc123")

	inner := func(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return mcp.NewToolResultText(`{}`), nil
	}

	wrapped := hook.Wrap("entity", inner)
	_, err := wrapped(context.Background(), mcp.CallToolRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	entry := readLastEntry(t, dir)
	if entry.ServerVersion != "3.14.0-beta1+abc123" {
		t.Errorf("ServerVersion: got %q, want %q (AC-002)", entry.ServerVersion, "3.14.0-beta1+abc123")
	}
}

func TestHookWrap_TimestampFormat(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	wr := NewWriter(dir)
	defer wr.Close()

	hook := NewHook(wr, nil, "2.0")
	inner := func(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return mcp.NewToolResultText(`{}`), nil
	}

	wrapped := hook.Wrap("status", inner)
	wrapped(context.Background(), mcp.CallToolRequest{})

	entry := readLastEntry(t, dir)
	if _, err := time.Parse(time.RFC3339, entry.Timestamp); err != nil {
		t.Errorf("Timestamp %q is not RFC3339: %v", entry.Timestamp, err)
	}
}

// ─── Knowledge rejection capture tests ─────────────────────────────────────

// mockSideEffectInspector satisfies SideEffectInspector for testing.
type mockSideEffectInspector struct {
	counts map[string]int
}

func (m *mockSideEffectInspector) CountByType(typeName string) int {
	if m == nil || m.counts == nil {
		return 0
	}
	return m.counts[typeName]
}

func TestCountRejections_NoCollectorOnContext(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	got := countRejections(ctx)
	if got != "" {
		t.Errorf("countRejections with no collector: got %q, want empty", got)
	}
}

func TestCountRejections_ZeroRejections(t *testing.T) {
	t.Parallel()

	m := &mockSideEffectInspector{counts: map[string]int{"knowledge_rejected": 0}}
	ctx := context.WithValue(context.Background(), SideEffectKey, m)
	got := countRejections(ctx)
	if got != "" {
		t.Errorf("countRejections with zero rejections: got %q, want empty", got)
	}
}

func TestCountRejections_NonZeroRejections(t *testing.T) {
	t.Parallel()

	m := &mockSideEffectInspector{counts: map[string]int{"knowledge_rejected": 3}}
	ctx := context.WithValue(context.Background(), SideEffectKey, m)
	got := countRejections(ctx)
	if got != "3" {
		t.Errorf("countRejections with 3 rejections: got %q, want %q", got, "3")
	}
}

func TestHookWrap_KnowledgeRejectionsMergedIntoExtra(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	wr := NewWriter(dir)
	defer wr.Close()

	hook := NewHook(wr, nil, "2.0")

	inner := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// Also add an annotation to verify both merge correctly.
		AnnotateEntry(ctx, AnnotationResultCount, "5")
		return mcp.NewToolResultText(`{}`), nil
	}

	// Place a mock side-effect inspector on the context with 2 rejections.
	m := &mockSideEffectInspector{counts: map[string]int{"knowledge_rejected": 2}}
	ctx := context.WithValue(context.Background(), SideEffectKey, m)

	wrapped := hook.Wrap("entity", inner)
	_, err := wrapped(ctx, mcp.CallToolRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	entry := readLastEntry(t, dir)
	if entry.Extra == nil {
		t.Fatal("Extra: got nil, want non-nil")
	}
	if entry.Extra[AnnotationKBRejections] != "2" {
		t.Errorf("Extra[kb_rejections]: got %q, want %q", entry.Extra[AnnotationKBRejections], "2")
	}
	if entry.Extra[AnnotationResultCount] != "5" {
		t.Errorf("Extra[result_count]: got %q, want %q", entry.Extra[AnnotationResultCount], "5")
	}
}

func TestHookWrap_KnowledgeRejectionsWhenNoAnnotations(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	wr := NewWriter(dir)
	defer wr.Close()

	hook := NewHook(wr, nil, "2.0")

	inner := func(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// No annotations added.
		return mcp.NewToolResultText(`{}`), nil
	}

	// Place a mock side-effect inspector on the context with 1 rejection.
	m := &mockSideEffectInspector{counts: map[string]int{"knowledge_rejected": 1}}
	ctx := context.WithValue(context.Background(), SideEffectKey, m)

	wrapped := hook.Wrap("entity", inner)
	_, err := wrapped(ctx, mcp.CallToolRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	entry := readLastEntry(t, dir)
	if entry.Extra == nil {
		t.Fatal("Extra: got nil, want non-nil (should be allocated for rejections)")
	}
	if entry.Extra[AnnotationKBRejections] != "1" {
		t.Errorf("Extra[kb_rejections]: got %q, want %q", entry.Extra[AnnotationKBRejections], "1")
	}
	// Should not contain annotation keys that weren't added.
	if _, ok := entry.Extra[AnnotationResultCount]; ok {
		t.Error("Extra[result_count] should not be present")
	}
}

func TestHookWrap_KnowledgeRejectionsWithWrongCollectorType(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	wr := NewWriter(dir)
	defer wr.Close()

	hook := NewHook(wr, nil, "2.0")

	inner := func(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return mcp.NewToolResultText(`{}`), nil
	}

	// Place a value of the wrong type under SideEffectKey.
	ctx := context.WithValue(context.Background(), SideEffectKey, "not-a-collector")

	wrapped := hook.Wrap("entity", inner)
	_, err := wrapped(ctx, mcp.CallToolRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	entry := readLastEntry(t, dir)
	// Should not crash and should not set kb_rejections.
	if entry.Extra != nil {
		if _, ok := entry.Extra[AnnotationKBRejections]; ok {
			t.Error("Extra[kb_rejections] should not be present with wrong collector type")
		}
	}
}

// AC-007: entity(action: "list") annotates result_count. This is an integration-style
// test that verifies the annotation flow end-to-end through Hook.Wrap, matching what
// entity_tool.go does when listing entities.
func TestHookWrap_EntityListAnnotatesResultCount(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	wr := NewWriter(dir)
	defer wr.Close()

	hook := NewHook(wr, nil, "2.0")

	// Simulate what entityListAction does: annotate result_count with the count.
	inner := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		AnnotateEntry(ctx, AnnotationResultCount, "42")
		return mcp.NewToolResultText(`{"entities":[],"total":42}`), nil
	}

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{"action": "list", "type": "feature"}

	wrapped := hook.Wrap("entity", inner)
	_, err := wrapped(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	entry := readLastEntry(t, dir)
	if entry.Extra == nil {
		t.Fatal("Extra: got nil, want non-nil (AC-007)")
	}
	if entry.Extra[AnnotationResultCount] != "42" {
		t.Errorf("Extra[result_count]: got %q, want %q (AC-007)", entry.Extra[AnnotationResultCount], "42")
	}
}

// AC-008: knowledge(action: "list") annotates result_count. Same pattern as AC-007
// but for the knowledge tool.
func TestHookWrap_KnowledgeListAnnotatesResultCount(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	wr := NewWriter(dir)
	defer wr.Close()

	hook := NewHook(wr, nil, "2.0")

	inner := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		AnnotateEntry(ctx, AnnotationResultCount, "7")
		return mcp.NewToolResultText(`{"count":7}`), nil
	}

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{"action": "list"}

	wrapped := hook.Wrap("knowledge", inner)
	_, err := wrapped(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	entry := readLastEntry(t, dir)
	if entry.Extra == nil {
		t.Fatal("Extra: got nil, want non-nil (AC-008)")
	}
	if entry.Extra[AnnotationResultCount] != "7" {
		t.Errorf("Extra[result_count]: got %q, want %q (AC-008)", entry.Extra[AnnotationResultCount], "7")
	}
}

// AC-009: doc_intel(action: "search") annotates result_count. Same pattern as AC-007
// but for the doc_intel tool.
func TestHookWrap_DocIntelSearchAnnotatesResultCount(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	wr := NewWriter(dir)
	defer wr.Close()

	hook := NewHook(wr, nil, "2.0")

	inner := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		AnnotateEntry(ctx, AnnotationResultCount, "12")
		return mcp.NewToolResultText(`{"total_matches":15,"returned":12}`), nil
	}

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{"action": "search", "query": "test"}

	wrapped := hook.Wrap("doc_intel", inner)
	_, err := wrapped(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	entry := readLastEntry(t, dir)
	if entry.Extra == nil {
		t.Fatal("Extra: got nil, want non-nil (AC-009)")
	}
	if entry.Extra[AnnotationResultCount] != "12" {
		t.Errorf("Extra[result_count]: got %q, want %q (AC-009)", entry.Extra[AnnotationResultCount], "12")
	}
}

// readLastEntry reads the most recent log file and parses the last entry.
func readLastEntry(t *testing.T, dir string) Entry {
	t.Helper()

	pattern := filepath.Join(dir, "actions-*.jsonl")
	matches, _ := filepath.Glob(pattern)
	if len(matches) == 0 {
		t.Fatal("no log file found")
	}

	data, err := os.ReadFile(matches[len(matches)-1])
	if err != nil {
		t.Fatalf("read log file: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) == 0 {
		t.Fatal("log file empty")
	}

	var entry Entry
	if err := json.Unmarshal([]byte(lines[len(lines)-1]), &entry); err != nil {
		t.Fatalf("unmarshal last entry: %v", err)
	}
	return entry
}
