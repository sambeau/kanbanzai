package actionlog

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestWriterLog(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	wr := NewWriter(dir)
	defer wr.Close()

	action := "create"
	e := Entry{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Tool:      "entity",
		Action:    &action,
		Success:   true,
	}

	if err := wr.Log(e); err != nil {
		t.Fatalf("Log: %v", err)
	}

	pattern := filepath.Join(dir, "actions-*.jsonl")
	matches, err := filepath.Glob(pattern)
	if err != nil || len(matches) == 0 {
		t.Fatalf("no log file found: %v", err)
	}

	data, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatalf("read log file: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 1 {
		t.Fatalf("expected 1 line, got %d", len(lines))
	}

	var got Entry
	if err := json.Unmarshal([]byte(lines[0]), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.Tool != "entity" {
		t.Errorf("Tool: got %q, want %q", got.Tool, "entity")
	}
}

func TestWriterMultipleEntries(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	wr := NewWriter(dir)
	defer wr.Close()

	for i := 0; i < 3; i++ {
		e := Entry{
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			Tool:      "status",
			Success:   true,
		}
		if err := wr.Log(e); err != nil {
			t.Fatalf("Log %d: %v", i, err)
		}
	}

	pattern := filepath.Join(dir, "actions-*.jsonl")
	matches, _ := filepath.Glob(pattern)
	if len(matches) == 0 {
		t.Fatal("no log file found")
	}

	data, _ := os.ReadFile(matches[0])
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 3 {
		t.Errorf("expected 3 lines, got %d", len(lines))
	}
}

func TestLogsDirName(t *testing.T) {
	t.Parallel()

	dir := LogsDir()
	if !strings.HasSuffix(dir, "/logs") && !strings.HasSuffix(dir, filepath.Join(".kbz", "logs")) {
		t.Errorf("LogsDir() = %q, expected to end with .kbz/logs", dir)
	}
}
