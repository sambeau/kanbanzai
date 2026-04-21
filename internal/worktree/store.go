package worktree

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/sambeau/kanbanzai/internal/fsutil"
	"github.com/sambeau/kanbanzai/internal/id"
)

const maxCollisionRetries = 3

// WorktreesDir is the directory name for worktree records within state.
const WorktreesDir = "worktrees"

// ErrConflict is returned when a write fails because the file on disk
// has changed since the record was loaded (optimistic-locking violation).
var ErrConflict = errors.New("concurrent modification: file changed since last read")

// ErrNotFound is returned when a requested worktree record does not exist.
var ErrNotFound = errors.New("worktree not found")

// Store handles storage and retrieval of worktree records.
type Store struct {
	root string
}

// NewStore creates a new Store rooted at the given path.
// The path should be the .kbz/state directory.
func NewStore(root string) *Store {
	return &Store{root: root}
}

// Create persists a new worktree record to disk.
// It generates a new ID for the record if not already set.
func (s *Store) Create(record Record) (Record, error) {
	if record.ID == "" {
		newID, err := s.allocateID()
		if err != nil {
			return record, fmt.Errorf("allocate worktree ID: %w", err)
		}
		record.ID = newID
	}

	if err := s.validateRecord(record); err != nil {
		return record, err
	}

	path, err := s.write(record)
	if err != nil {
		return record, err
	}

	// Return record with FileHash for subsequent updates
	data, _ := os.ReadFile(path)
	h := sha256.Sum256(data)
	record.FileHash = hex.EncodeToString(h[:])

	return record, nil
}

// allocateID generates a new worktree ID with collision checking.
func (s *Store) allocateID() (string, error) {
	for attempt := 0; attempt <= maxCollisionRetries; attempt++ {
		tsid, err := id.GenerateTSID13()
		if err != nil {
			return "", fmt.Errorf("generate TSID: %w", err)
		}

		wtID := "WT-" + tsid

		// Check if ID already exists
		_, err = s.Get(wtID)
		if errors.Is(err, ErrNotFound) {
			return wtID, nil
		}
		// If err is nil, ID exists — retry
		// If err is something else, also retry
	}

	return "", fmt.Errorf("ID collision persisted after %d retries", maxCollisionRetries)
}

// Get reads a worktree record from disk by ID.
func (s *Store) Get(wtID string) (Record, error) {
	var record Record

	if strings.TrimSpace(wtID) == "" {
		return record, errors.New("worktree ID is required")
	}

	path := filepath.Join(s.root, WorktreesDir, wtID+".yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return record, fmt.Errorf("%w: %s", ErrNotFound, wtID)
		}
		return record, fmt.Errorf("read worktree record: %w", err)
	}

	fields, err := unmarshalYAML(string(data))
	if err != nil {
		return record, fmt.Errorf("unmarshal worktree record: %w", err)
	}

	record, err = recordFromFields(fields)
	if err != nil {
		return record, fmt.Errorf("parse worktree record: %w", err)
	}

	h := sha256.Sum256(data)
	record.FileHash = hex.EncodeToString(h[:])

	return record, nil
}

// GetByEntityID returns the active worktree record associated with the given entity ID.
// Returns (nil, nil) if no active worktree is associated with the entity.
func (s *Store) GetByEntityID(entityID string) (*Record, error) {
	records, err := s.List()
	if err != nil {
		return nil, err
	}

	for _, r := range records {
		if r.Status == StatusActive && r.EntityID == entityID {
			rec := r
			return &rec, nil
		}
	}

	return nil, nil
}

// List returns all worktree records in the store.
func (s *Store) List() ([]Record, error) {
	dir := filepath.Join(s.root, WorktreesDir)

	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("read worktrees directory: %w", err)
	}

	var records []Record
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}

		wtID := strings.TrimSuffix(entry.Name(), ".yaml")
		record, err := s.Get(wtID)
		if err != nil {
			continue // Skip unreadable records
		}
		records = append(records, record)
	}

	// Sort by ID for deterministic ordering
	sort.Slice(records, func(i, j int) bool {
		return records[i].ID < records[j].ID
	})

	return records, nil
}

// Update persists changes to an existing worktree record.
// The record must include FileHash from a previous Get or Create call
// for optimistic locking.
func (s *Store) Update(record Record) (Record, error) {
	if err := s.validateRecord(record); err != nil {
		return record, err
	}

	path, err := s.write(record)
	if err != nil {
		return record, err
	}

	// Return record with updated FileHash
	data, _ := os.ReadFile(path)
	h := sha256.Sum256(data)
	record.FileHash = hex.EncodeToString(h[:])

	return record, nil
}

// Delete removes a worktree record from disk.
func (s *Store) Delete(wtID string) error {
	if strings.TrimSpace(wtID) == "" {
		return errors.New("worktree ID is required")
	}

	path := filepath.Join(s.root, WorktreesDir, wtID+".yaml")
	if err := os.Remove(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil // Already deleted
		}
		return fmt.Errorf("delete worktree record: %w", err)
	}

	return nil
}

// write persists a worktree record to disk with optimistic locking.
func (s *Store) write(record Record) (string, error) {
	dir := filepath.Join(s.root, WorktreesDir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("create worktrees directory: %w", err)
	}

	path := filepath.Join(dir, record.ID+".yaml")

	// Optimistic locking: if FileHash is set, verify the file hasn't changed.
	if record.FileHash != "" {
		current, err := os.ReadFile(path)
		if err == nil {
			h := sha256.Sum256(current)
			if hex.EncodeToString(h[:]) != record.FileHash {
				return "", fmt.Errorf("write worktree %s: %w", record.ID, ErrConflict)
			}
		}
		// If the file doesn't exist (os.ErrNotExist), skip the check — new record.
	}

	content, err := marshalYAML(record.Fields())
	if err != nil {
		return "", fmt.Errorf("marshal worktree record: %w", err)
	}

	if err := fsutil.WriteFileAtomic(path, []byte(content), 0o644); err != nil {
		return "", fmt.Errorf("write worktree record: %w", err)
	}

	return path, nil
}

// validateRecord checks that a worktree record is valid for storage.
func (s *Store) validateRecord(record Record) error {
	if strings.TrimSpace(record.ID) == "" {
		return errors.New("worktree ID is required")
	}
	if strings.TrimSpace(record.EntityID) == "" {
		return errors.New("worktree entity_id is required")
	}
	if strings.TrimSpace(record.Branch) == "" {
		return errors.New("worktree branch is required")
	}
	if strings.TrimSpace(record.Path) == "" {
		return errors.New("worktree path is required")
	}
	if !ValidStatus(record.Status) {
		return fmt.Errorf("invalid worktree status: %q", record.Status)
	}
	if record.Created.IsZero() {
		return errors.New("worktree created timestamp is required")
	}
	if strings.TrimSpace(record.CreatedBy) == "" {
		return errors.New("worktree created_by is required")
	}

	return nil
}

// marshalYAML produces canonical YAML output for a worktree record.
func marshalYAML(fields map[string]any) (string, error) {
	if len(fields) == 0 {
		return "", errors.New("fields are required")
	}

	var b strings.Builder
	order := FieldOrder()

	// Write fields in canonical order
	seen := make(map[string]struct{})
	for _, key := range order {
		if value, ok := fields[key]; ok {
			writeField(&b, key, value)
			seen[key] = struct{}{}
		}
	}

	// Write any extra fields in alphabetical order
	var extras []string
	for key := range fields {
		if _, ok := seen[key]; !ok {
			extras = append(extras, key)
		}
	}
	sort.Strings(extras)
	for _, key := range extras {
		writeField(&b, key, fields[key])
	}

	return b.String(), nil
}

func writeField(b *strings.Builder, key string, value any) {
	b.WriteString(key)
	b.WriteString(": ")
	b.WriteString(formatScalar(value))
	b.WriteString("\n")
}

func formatScalar(value any) string {
	switch v := value.(type) {
	case nil:
		return "null"
	case string:
		if needsQuotes(v) {
			return quoteString(v)
		}
		return v
	case bool:
		if v {
			return "true"
		}
		return "false"
	default:
		s := fmt.Sprint(value)
		if needsQuotes(s) {
			return quoteString(s)
		}
		return s
	}
}

func needsQuotes(value string) bool {
	if value == "" {
		return true
	}

	lower := strings.ToLower(value)
	switch lower {
	case "true", "false", "null", "yes", "no", "on", "off", "~":
		return true
	}

	if strings.TrimSpace(value) != value {
		return true
	}

	if strings.ContainsAny(value, "\n\r:") {
		return true
	}

	for _, r := range value {
		switch r {
		case '#', '{', '}', '[', ']', ',', '&', '*', '!', '|', '>', '@', '`', '"', '\'':
			return true
		}
	}

	if strings.HasPrefix(value, "-") || strings.HasPrefix(value, "?") {
		return true
	}

	return false
}

func quoteString(value string) string {
	replacer := strings.NewReplacer(
		`\`, `\\`,
		`"`, `\"`,
		"\n", `\n`,
	)
	return `"` + replacer.Replace(value) + `"`
}

// unmarshalYAML parses canonical YAML content into a map.
func unmarshalYAML(content string) (map[string]any, error) {
	result := make(map[string]any)
	lines := strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n")

	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		if value == "" || value == "null" {
			result[key] = nil
			continue
		}

		result[key] = parseScalar(value)
	}

	return result, nil
}

func parseScalar(value string) any {
	value = strings.TrimSpace(value)

	switch value {
	case "true":
		return true
	case "false":
		return false
	case "null":
		return nil
	}

	// Handle quoted strings
	if len(value) >= 2 && strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"") {
		unquoted := value[1 : len(value)-1]
		unquoted = strings.ReplaceAll(unquoted, "\\\"", "\"")
		unquoted = strings.ReplaceAll(unquoted, "\\\\", "\\")
		unquoted = strings.ReplaceAll(unquoted, "\\n", "\n")
		return unquoted
	}

	return value
}

// recordFromFields parses a map of fields into a Record.
func recordFromFields(fields map[string]any) (Record, error) {
	var record Record

	if v, ok := fields["id"].(string); ok {
		record.ID = v
	}
	if v, ok := fields["entity_id"].(string); ok {
		record.EntityID = v
	}
	if v, ok := fields["branch"].(string); ok {
		record.Branch = v
	}
	if v, ok := fields["path"].(string); ok {
		record.Path = v
	}
	if v, ok := fields["status"].(string); ok {
		record.Status = Status(v)
	}
	if v, ok := fields["created"].(string); ok {
		t, err := time.Parse(time.RFC3339, v)
		if err == nil {
			record.Created = t
		}
	}
	if v, ok := fields["created_by"].(string); ok {
		record.CreatedBy = v
	}
	if v, ok := fields["merged_at"].(string); ok {
		t, err := time.Parse(time.RFC3339, v)
		if err == nil {
			record.MergedAt = &t
		}
	}
	if v, ok := fields["cleanup_after"].(string); ok {
		t, err := time.Parse(time.RFC3339, v)
		if err == nil {
			record.CleanupAfter = &t
		}
	}
	if v, ok := fields["graph_project"].(string); ok {
		record.GraphProject = v
	}

	return record, nil
}
