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

	hook := NewHook(wr, nil)

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

	hook := NewHook(wr, nil)

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

	hook := NewHook(wr, nil)

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

func TestHookWrap_TimestampFormat(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	wr := NewWriter(dir)
	defer wr.Close()

	hook := NewHook(wr, nil)
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
