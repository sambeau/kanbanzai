package mcp

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/sambeau/kanbanzai/internal/hashvalidate"
	"github.com/sambeau/kanbanzai/internal/worktree"
)

// invokeReadFile calls the read_file tool handler with the given args and
// returns the parsed JSON response as a map.
func invokeReadFile(t *testing.T, repoRoot string, store *worktree.Store, args map[string]any) map[string]any {
	t.Helper()

	tool := readFileTool(repoRoot, store)
	req := mcp.CallToolRequest{}
	req.Params.Arguments = args

	result, err := tool.Handler(context.Background(), req)
	if err != nil {
		t.Fatalf("read_file handler returned unexpected error: %v", err)
	}
	if len(result.Content) == 0 {
		t.Fatal("read_file handler returned empty content")
	}

	tc, ok := result.Content[0].(mcp.TextContent)
	if !ok {
		t.Fatalf("expected mcp.TextContent, got %T", result.Content[0])
	}

	var resp map[string]any
	if err := json.Unmarshal([]byte(tc.Text), &resp); err != nil {
		t.Fatalf("unmarshal response: %v\nraw: %s", err, tc.Text)
	}
	return resp
}

func TestReadFile_PlainText(t *testing.T) {
	repoDir := t.TempDir()
	repoFile := filepath.Join(repoDir, "test.txt")
	content := "line one\nline two\nline three\n"
	if err := os.WriteFile(repoFile, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	resp := invokeReadFile(t, repoDir, nil, map[string]any{
		"path": "test.txt",
	})

	if errObj, _ := resp["error"].(map[string]any); errObj != nil {
		t.Fatalf("unexpected error: %v", resp["error"])
	}

	got := resp["content"].(string)
	if got != content {
		t.Errorf("content = %q, want %q", got, content)
	}
}

func TestReadFile_HashTagged(t *testing.T) {
	repoDir := t.TempDir()
	repoFile := filepath.Join(repoDir, "test.txt")
	content := "line one\nline two\nline three\n"
	if err := os.WriteFile(repoFile, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	resp := invokeReadFile(t, repoDir, nil, map[string]any{
		"path":     "test.txt",
		"hash_tag": true,
	})

	if errObj, _ := resp["error"].(map[string]any); errObj != nil {
		t.Fatalf("unexpected error: %v", resp["error"])
	}

	got := resp["content"].(string)
	lines := strings.Split(strings.TrimRight(got, "\n"), "\n")

	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d:\n%s", len(lines), got)
	}

	// Verify format: "   N#XX| content"
	for i, line := range lines {
		// Each line should have the hash tag format.
		if !strings.Contains(line, "#") || !strings.Contains(line, "|") {
			t.Errorf("line %d missing hash-tag format: %q", i+1, line)
		}

		// Extract the hash from the line to verify it matches HashLine.
		parts := strings.SplitN(line, "#", 2)
		if len(parts) != 2 {
			t.Errorf("line %d missing # separator: %q", i+1, line)
			continue
		}
		hashAndContent := strings.SplitN(parts[1], "| ", 2)
		if len(hashAndContent) != 2 {
			t.Errorf("line %d missing | separator: %q", i+1, line)
			continue
		}
		hash := hashAndContent[0]
		lineContent := hashAndContent[1]

		expectedHash := hashvalidate.HashLine(lineContent)
		if hash != expectedHash {
			t.Errorf("line %d hash = %q, expected %q (content=%q)", i+1, hash, expectedHash, lineContent)
		}
	}
}

func TestReadFile_HashTagged_LineRange(t *testing.T) {
	repoDir := t.TempDir()
	repoFile := filepath.Join(repoDir, "test.txt")
	content := "line one\nline two\nline three\nline four\nline five\n"
	if err := os.WriteFile(repoFile, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	resp := invokeReadFile(t, repoDir, nil, map[string]any{
		"path":       "test.txt",
		"hash_tag":   true,
		"start_line": float64(2),
		"end_line":   float64(4),
	})

	if errObj, _ := resp["error"].(map[string]any); errObj != nil {
		t.Fatalf("unexpected error: %v", resp["error"])
	}

	got := resp["content"].(string)
	lines := strings.Split(strings.TrimRight(got, "\n"), "\n")

	if len(lines) != 3 {
		t.Fatalf("expected 3 lines (2-4), got %d:\n%s", len(lines), got)
	}

	// First line should start with "   2#" (line 2).
	if !strings.HasPrefix(lines[0], "   2#") {
		t.Errorf("first line should be line 2 (prefix '   2#'), got %q", lines[0])
	}
	// Last line should start with "   4#" (line 4).
	if !strings.HasPrefix(lines[2], "   4#") {
		t.Errorf("last line should be line 4 (prefix '   4#'), got %q", lines[2])
	}
}

func TestReadFile_HashTagged_BlankLine(t *testing.T) {
	repoDir := t.TempDir()
	repoFile := filepath.Join(repoDir, "test.txt")
	content := "line one\n\nline three\n"
	if err := os.WriteFile(repoFile, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	resp := invokeReadFile(t, repoDir, nil, map[string]any{
		"path":     "test.txt",
		"hash_tag": true,
	})

	if errObj, _ := resp["error"].(map[string]any); errObj != nil {
		t.Fatalf("unexpected error: %v", resp["error"])
	}

	got := resp["content"].(string)
	// Line 2 should be a blank line with hash tag: "   2#XX| "
	lines := strings.Split(strings.TrimRight(got, "\n"), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d", len(lines))
	}
	if !strings.HasPrefix(lines[1], "   2#") || !strings.Contains(lines[1], "| ") {
		t.Errorf("blank line (line 2) should have hash tag prefix, got %q", lines[1])
	}
}

func TestReadFile_EndToEnd_HashEditFlow(t *testing.T) {
	// Full flow: read with hash_tag → edit with hash_validate.
	repoDir := t.TempDir()
	repoFile := filepath.Join(repoDir, "test.go")
	content := "package main\n\nfunc main() {\n\tprintln(\"hello\")\n}\n"
	if err := os.WriteFile(repoFile, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	// 1. Read with hash tags.
	readResp := invokeReadFile(t, repoDir, nil, map[string]any{
		"path":     "test.go",
		"hash_tag": true,
	})
	readContent := readResp["content"].(string)

	// 2. Parse line 3's hash (the line "func main() {").
	lines := strings.Split(strings.TrimRight(readContent, "\n"), "\n")
	if len(lines) < 3 {
		t.Fatalf("expected at least 3 lines, got %d:\n%s", len(lines), readContent)
	}
	line3 := lines[2] // 0-indexed, so line 3
	parts := strings.SplitN(line3, "#", 2)
	hashAndContent := strings.SplitN(parts[1], "| ", 2)
	hashRef := "3#" + hashAndContent[0]

	// 3. Edit with hash_validate.
	editResp := invokeEditFile(t, repoDir, nil, map[string]any{
		"display_description": "change greeting",
		"path":                "test.go",
		"mode":                "edit",
		"hash_validate":       true,
		"edits": []any{
			map[string]any{
				"hash_ref": hashRef,
				"new_text": "func main() {",
			},
		},
	})
	if errObj, _ := editResp["error"].(map[string]any); errObj != nil {
		t.Fatalf("hash-validated edit failed: %v", editResp["error"])
	}

	// 4. Verify content unchanged (same line content).
	data, _ := os.ReadFile(repoFile)
	if !strings.Contains(string(data), "func main() {") {
		t.Errorf("file should still contain func main()")
	}
}
