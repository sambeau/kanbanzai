package structural

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/sambeau/kanbanzai/internal/fsutil"
	"gopkg.in/yaml.v3"
)

const (
	promotionStateFile       = "structural-check-state.yaml"
	promotionThreshold       = 5
	modeWarning              = "warning"
	modeHardGate             = "hard_gate"
)

// CheckKey identifies a (check_type, document_type) promotion state entry.
type CheckKey struct {
	CheckType    string
	DocumentType string
}

// PromotionEntry tracks the promotion state for a single check+doc type pair.
type PromotionEntry struct {
	Mode               string     `yaml:"mode"`
	ConsecutiveClean   int        `yaml:"consecutive_clean"`
	PromotedAt         *time.Time `yaml:"promoted_at,omitempty"`
	DemotedAt          *time.Time `yaml:"demoted_at,omitempty"`
	FalsePositiveCount int        `yaml:"false_positive_count"`
}

// promotionStateYAML is the on-disk format.
type promotionStateYAML struct {
	Entries map[string]PromotionEntry `yaml:"entries"`
}

// PromotionState manages the promotion state for structural checks.
type PromotionState struct {
	path    string
	entries map[string]PromotionEntry
}

// entryKey converts a CheckKey to a string map key.
func entryKey(key CheckKey) string {
	return key.CheckType + "/" + key.DocumentType
}

// LoadPromotionState loads the promotion state from disk.
// If the file does not exist, an empty state is returned.
func LoadPromotionState(stateRoot string) (*PromotionState, error) {
	path := filepath.Join(stateRoot, promotionStateFile)
	ps := &PromotionState{
		path:    path,
		entries: make(map[string]PromotionEntry),
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return ps, nil
		}
		return nil, fmt.Errorf("read promotion state: %w", err)
	}

	var raw promotionStateYAML
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse promotion state: %w", err)
	}

	if raw.Entries != nil {
		ps.entries = raw.Entries
	}

	return ps, nil
}

// Save persists the promotion state to disk atomically.
func (ps *PromotionState) Save() error {
	raw := promotionStateYAML{Entries: ps.entries}
	data, err := yaml.Marshal(raw)
	if err != nil {
		return fmt.Errorf("marshal promotion state: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(ps.path), 0o755); err != nil {
		return fmt.Errorf("create state dir: %w", err)
	}

	if err := fsutil.WriteFileAtomic(ps.path, data, 0o644); err != nil {
		return fmt.Errorf("write promotion state: %w", err)
	}

	return nil
}

// GetMode returns the current mode ("warning" or "hard_gate") for the given check key.
// Defaults to "warning" if no entry exists.
func (ps *PromotionState) GetMode(key CheckKey) string {
	entry, ok := ps.entries[entryKey(key)]
	if !ok {
		return modeWarning
	}
	return entry.Mode
}

// RecordPass records a successful (clean) run for the given check key.
// After promotionThreshold consecutive clean passes, the mode is promoted to hard_gate.
func (ps *PromotionState) RecordPass(key CheckKey) {
	k := entryKey(key)
	entry := ps.entries[k]

	if entry.Mode == "" {
		entry.Mode = modeWarning
	}

	entry.ConsecutiveClean++

	if entry.Mode == modeWarning && entry.ConsecutiveClean >= promotionThreshold {
		entry.Mode = modeHardGate
		now := time.Now().UTC()
		entry.PromotedAt = &now
	}

	ps.entries[k] = entry
}

// RecordFalsePositive records a false positive for the given check key,
// incrementing the false positive count and resetting consecutive clean passes.
// If the check was previously promoted to hard_gate, it is demoted back to warning.
func (ps *PromotionState) RecordFalsePositive(key CheckKey, description string) {
	k := entryKey(key)
	entry := ps.entries[k]

	if entry.Mode == "" {
		entry.Mode = modeWarning
	}

	entry.FalsePositiveCount++
	entry.ConsecutiveClean = 0

	if entry.Mode == modeHardGate {
		entry.Mode = modeWarning
		now := time.Now().UTC()
		entry.DemotedAt = &now
	}

	ps.entries[k] = entry
}
