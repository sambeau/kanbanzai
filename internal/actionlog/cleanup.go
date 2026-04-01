package actionlog

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"time"
)

// logFilePattern matches action log file names: actions-YYYY-MM-DD.jsonl
var logFilePattern = regexp.MustCompile(`^actions-(\d{4}-\d{2}-\d{2})\.jsonl$`)

// Cleanup deletes log files older than 30 days.
// Only files matching the actions-YYYY-MM-DD.jsonl pattern are deleted.
// A missing directory is not an error. Per-file failures are logged to stderr.
func Cleanup(logsDir string, now time.Time) error {
	entries, err := os.ReadDir(logsDir)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("cleanup logs: read dir: %w", err)
	}

	cutoff := now.UTC().AddDate(0, 0, -30)
	cutoffDate := cutoff.Format("2006-01-02")

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		m := logFilePattern.FindStringSubmatch(entry.Name())
		if m == nil {
			continue
		}
		fileDate := m[1]
		if fileDate >= cutoffDate {
			continue
		}
		path := filepath.Join(logsDir, entry.Name())
		if err := os.Remove(path); err != nil {
			fmt.Fprintf(os.Stderr, "actionlog cleanup: remove %s: %v\n", path, err)
		}
	}
	return nil
}
