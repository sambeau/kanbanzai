package actionlog

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/sambeau/kanbanzai/internal/core"
)

const logsSubDir = "logs"

// LogsDir returns the canonical logs directory path (.kbz/logs).
func LogsDir() string {
	return filepath.Join(core.InstanceRootDir, logsSubDir)
}

// Writer appends log entries as JSON lines to date-partitioned JSONL files.
// It is safe for concurrent use.
type Writer struct {
	mu      sync.Mutex
	logsDir string
	date    string // current UTC date: YYYY-MM-DD
	file    *os.File
	buf     *bufio.Writer
}

// NewWriter creates a Writer that writes to logsDir.
func NewWriter(logsDir string) *Writer {
	return &Writer{logsDir: logsDir}
}

// Log appends e as a JSON line to the current day log file.
// Callers must NOT fail if Log returns an error — logging is best-effort.
func (w *Writer) Log(e Entry) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	now := time.Now().UTC()
	date := now.Format("2006-01-02")

	if err := w.rotate(date); err != nil {
		return fmt.Errorf("actionlog rotate: %w", err)
	}

	b, err := json.Marshal(e)
	if err != nil {
		return fmt.Errorf("actionlog marshal: %w", err)
	}

	if _, err := w.buf.Write(b); err != nil {
		return fmt.Errorf("actionlog write: %w", err)
	}
	if err := w.buf.WriteByte('\n'); err != nil {
		return fmt.Errorf("actionlog write newline: %w", err)
	}
	if err := w.buf.Flush(); err != nil {
		return fmt.Errorf("actionlog flush: %w", err)
	}
	return nil
}

// rotate opens a new log file when the UTC date has changed or no file is open.
func (w *Writer) rotate(date string) error {
	if w.file != nil && w.date == date {
		return nil
	}

	if w.file != nil {
		_ = w.buf.Flush()
		_ = w.file.Close()
		w.file = nil
		w.buf = nil
	}

	if err := os.MkdirAll(w.logsDir, 0o755); err != nil {
		return fmt.Errorf("create logs dir: %w", err)
	}

	path := filepath.Join(w.logsDir, "actions-"+date+".jsonl")
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("open log file: %w", err)
	}

	w.file = f
	w.buf = bufio.NewWriter(f)
	w.date = date
	return nil
}

// Close flushes any buffered data and closes the current log file.
// After Close, the Writer must not be used.
func (w *Writer) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.file == nil {
		return nil
	}
	if err := w.buf.Flush(); err != nil {
		_ = w.file.Close()
		return err
	}
	err := w.file.Close()
	w.file = nil
	w.buf = nil
	return err
}
