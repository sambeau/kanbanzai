package actionlog

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestReadEntries_InRange(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeLogFile(t, dir, "2024-03-10", []string{
		`{"timestamp":"2024-03-10T10:00:00Z","tool":"entity","action":null,"entity_id":null,"stage":null,"success":true,"error_type":null}`,
		`{"timestamp":"2024-03-10T11:00:00Z","tool":"doc","action":null,"entity_id":null,"stage":null,"success":true,"error_type":null}`,
	})
	writeLogFile(t, dir, "2024-03-11", []string{
		`{"timestamp":"2024-03-11T10:00:00Z","tool":"status","action":null,"entity_id":null,"stage":null,"success":true,"error_type":null}`,
	})
	writeLogFile(t, dir, "2024-03-15", []string{
		`{"timestamp":"2024-03-15T10:00:00Z","tool":"next","action":null,"entity_id":null,"stage":null,"success":false,"error_type":null}`,
	})

	since := time.Date(2024, 3, 10, 0, 0, 0, 0, time.UTC)
	until := time.Date(2024, 3, 11, 23, 59, 59, 0, time.UTC)

	entries, err := ReadEntries(dir, since, until)
	if err != nil {
		t.Fatalf("ReadEntries: %v", err)
	}

	if len(entries) != 3 {
		t.Errorf("got %d entries, want 3", len(entries))
	}
}

func TestReadEntries_EmptyDir(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	since := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	until := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)

	entries, err := ReadEntries(dir, since, until)
	if err != nil {
		t.Fatalf("ReadEntries: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("got %d entries, want 0", len(entries))
	}
}

func TestReadEntries_SkipsMalformedLines(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeLogFile(t, dir, "2024-03-10", []string{
		`{"timestamp":"2024-03-10T10:00:00Z","tool":"entity","action":null,"entity_id":null,"stage":null,"success":true,"error_type":null}`,
		`not valid json`,
		`{"timestamp":"2024-03-10T11:00:00Z","tool":"doc","action":null,"entity_id":null,"stage":null,"success":true,"error_type":null}`,
	})

	since := time.Date(2024, 3, 10, 0, 0, 0, 0, time.UTC)
	until := time.Date(2024, 3, 10, 23, 59, 59, 0, time.UTC)

	entries, err := ReadEntries(dir, since, until)
	if err != nil {
		t.Fatalf("ReadEntries: %v", err)
	}
	if len(entries) != 2 {
		t.Errorf("got %d entries, want 2", len(entries))
	}
}

func writeLogFile(t *testing.T, dir, date string, lines []string) {
	t.Helper()
	path := filepath.Join(dir, "actions-"+date+".jsonl")
	content := ""
	for _, line := range lines {
		content += line + "\n"
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("writeLogFile: %v", err)
	}
}
