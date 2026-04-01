package actionlog

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// ReadEntries reads all log entries from files in logsDir whose date falls
// within [since, until] (inclusive). Malformed JSON lines are silently skipped.
func ReadEntries(logsDir string, since, until time.Time) ([]Entry, error) {
	sinceDate := since.UTC().Format("2006-01-02")
	untilDate := until.UTC().Format("2006-01-02")

	pattern := filepath.Join(logsDir, "actions-*.jsonl")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("read entries: glob: %w", err)
	}

	var entries []Entry
	for _, path := range matches {
		base := filepath.Base(path)
		m := logFilePattern.FindStringSubmatch(base)
		if m == nil {
			continue
		}
		fileDate := m[1]
		if fileDate < sinceDate || fileDate > untilDate {
			continue
		}

		fileEntries, err := readJSONLFile(path)
		if err != nil {
			return nil, fmt.Errorf("read entries from %s: %w", path, err)
		}
		entries = append(entries, fileEntries...)
	}
	return entries, nil
}

// readJSONLFile reads all valid JSON lines from a .jsonl file.
func readJSONLFile(path string) ([]Entry, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var entries []Entry
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var e Entry
		if err := json.Unmarshal(line, &e); err != nil {
			continue // skip malformed lines
		}
		entries = append(entries, e)
	}
	return entries, scanner.Err()
}
