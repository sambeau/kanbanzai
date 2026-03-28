package checkpoint

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

// CheckpointsDir is the directory name for checkpoint records within state.
const CheckpointsDir = "checkpoints"

// ErrNotFound is returned when a checkpoint does not exist.
var ErrNotFound = errors.New("checkpoint not found")

// Status is the lifecycle status of a checkpoint.
type Status string

const (
	StatusPending   Status = "pending"
	StatusResponded Status = "responded"
)

// Record is a checkpoint record.
type Record struct {
	ID                   string     `yaml:"id"`
	Question             string     `yaml:"question"`
	Context              string     `yaml:"context"`
	OrchestrationSummary string     `yaml:"orchestration_summary"`
	Status               Status     `yaml:"status"`
	CreatedAt            time.Time  `yaml:"created_at"`
	CreatedBy            string     `yaml:"created_by"`
	RespondedAt          *time.Time `yaml:"responded_at"` // null until responded
	Response             *string    `yaml:"response"`     // null until responded

	FileHash string `yaml:"-"` // for optimistic locking
}

// FieldOrder returns the canonical field order for YAML serialisation.
func FieldOrder() []string {
	return []string{
		"id",
		"question",
		"context",
		"orchestration_summary",
		"status",
		"created_at",
		"created_by",
		"responded_at",
		"response",
	}
}

// Fields returns the record as a map for YAML marshalling.
// responded_at and response are always present, using null when not set.
func (r Record) Fields() map[string]any {
	m := map[string]any{
		"id":                    r.ID,
		"question":              r.Question,
		"context":               r.Context,
		"orchestration_summary": r.OrchestrationSummary,
		"status":                string(r.Status),
		"created_at":            r.CreatedAt.UTC().Format(time.RFC3339),
		"created_by":            r.CreatedBy,
		"responded_at":          nil, // explicit null
		"response":              nil, // explicit null
	}
	if r.RespondedAt != nil {
		m["responded_at"] = r.RespondedAt.UTC().Format(time.RFC3339)
	}
	if r.Response != nil {
		m["response"] = *r.Response
	}
	return m
}

// Store handles storage and retrieval of checkpoint records.
type Store struct {
	root string
}

// NewStore creates a new Store rooted at the given state path.
func NewStore(root string) *Store {
	return &Store{root: root}
}

// dir returns the checkpoints directory path.
func (s *Store) dir() string {
	return filepath.Join(s.root, CheckpointsDir)
}

// Create persists a new checkpoint record. Generates a new ID.
func (s *Store) Create(record Record) (Record, error) {
	tsid, err := id.GenerateTSID13()
	if err != nil {
		return record, fmt.Errorf("generate checkpoint ID: %w", err)
	}
	record.ID = "CHK-" + tsid

	if err := s.validateRecord(record); err != nil {
		return record, err
	}

	path, err := s.write(record)
	if err != nil {
		return record, err
	}

	data, _ := os.ReadFile(path)
	h := sha256.Sum256(data)
	record.FileHash = hex.EncodeToString(h[:])

	return record, nil
}

// Get reads a checkpoint record by ID.
func (s *Store) Get(checkpointID string) (Record, error) {
	if strings.TrimSpace(checkpointID) == "" {
		return Record{}, errors.New("checkpoint ID is required")
	}

	path := filepath.Join(s.dir(), checkpointID+".yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Record{}, fmt.Errorf("%w: %s", ErrNotFound, checkpointID)
		}
		return Record{}, fmt.Errorf("read checkpoint record: %w", err)
	}

	record, err := unmarshalRecord(string(data))
	if err != nil {
		return Record{}, fmt.Errorf("parse checkpoint record: %w", err)
	}

	h := sha256.Sum256(data)
	record.FileHash = hex.EncodeToString(h[:])

	return record, nil
}

// List returns all checkpoint records, optionally filtered by status.
// Pass empty string for statusFilter to return all.
func (s *Store) List(statusFilter string) ([]Record, error) {
	entries, err := os.ReadDir(s.dir())
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("read checkpoints directory: %w", err)
	}

	var records []Record
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}
		chkID := strings.TrimSuffix(entry.Name(), ".yaml")
		record, err := s.Get(chkID)
		if err != nil {
			continue
		}
		if statusFilter != "" && string(record.Status) != statusFilter {
			continue
		}
		records = append(records, record)
	}

	sort.Slice(records, func(i, j int) bool {
		return records[i].ID < records[j].ID
	})

	return records, nil
}

// Update persists changes to an existing checkpoint (for responding).
func (s *Store) Update(record Record) (Record, error) {
	if err := s.validateRecord(record); err != nil {
		return record, err
	}

	path, err := s.write(record)
	if err != nil {
		return record, err
	}

	data, _ := os.ReadFile(path)
	h := sha256.Sum256(data)
	record.FileHash = hex.EncodeToString(h[:])

	return record, nil
}

// write persists a checkpoint record to disk.
func (s *Store) write(record Record) (string, error) {
	if err := os.MkdirAll(s.dir(), 0o755); err != nil {
		return "", fmt.Errorf("create checkpoints directory: %w", err)
	}

	path := filepath.Join(s.dir(), record.ID+".yaml")

	// Optimistic locking
	if record.FileHash != "" {
		current, err := os.ReadFile(path)
		if err == nil {
			h := sha256.Sum256(current)
			if hex.EncodeToString(h[:]) != record.FileHash {
				return "", fmt.Errorf("write checkpoint %s: concurrent modification", record.ID)
			}
		}
	}

	content, err := marshalRecord(record)
	if err != nil {
		return "", fmt.Errorf("marshal checkpoint record: %w", err)
	}

	if err := fsutil.WriteFileAtomic(path, []byte(content), 0o644); err != nil {
		return "", fmt.Errorf("write checkpoint file: %w", err)
	}

	return path, nil
}

// validateRecord checks required fields.
func (s *Store) validateRecord(record Record) error {
	if strings.TrimSpace(record.ID) == "" {
		return errors.New("checkpoint ID is required")
	}
	if strings.TrimSpace(record.Question) == "" {
		return errors.New("checkpoint question is required")
	}
	if strings.TrimSpace(record.CreatedBy) == "" {
		return errors.New("checkpoint created_by is required")
	}
	if record.CreatedAt.IsZero() {
		return errors.New("checkpoint created_at is required")
	}
	return nil
}

// marshalRecord serialises a checkpoint record to YAML.
// responded_at and response are always written (as null if not set).
func marshalRecord(record Record) (string, error) {
	fields := record.Fields()
	order := FieldOrder()

	var b strings.Builder
	seen := make(map[string]struct{})

	for _, key := range order {
		v, ok := fields[key]
		if !ok {
			continue
		}
		seen[key] = struct{}{}
		writeField(&b, key, v)
	}

	// Extra fields alphabetically
	var extras []string
	for k := range fields {
		if _, ok := seen[k]; !ok {
			extras = append(extras, k)
		}
	}
	sort.Strings(extras)
	for _, k := range extras {
		writeField(&b, k, fields[k])
	}

	return b.String(), nil
}

// writeField writes a single key: value line. Handles nil (writes null).
func writeField(b *strings.Builder, key string, value any) {
	b.WriteString(key)
	b.WriteString(": ")
	switch v := value.(type) {
	case nil:
		b.WriteString("null")
	case string:
		if needsQuotes(v) {
			b.WriteString(quoteString(v))
		} else {
			b.WriteString(v)
		}
	case bool:
		if v {
			b.WriteString("true")
		} else {
			b.WriteString("false")
		}
	default:
		s := fmt.Sprint(value)
		if needsQuotes(s) {
			b.WriteString(quoteString(s))
		} else {
			b.WriteString(s)
		}
	}
	b.WriteString("\n")
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

// unmarshalRecord parses a YAML checkpoint file into a Record.
func unmarshalRecord(content string) (Record, error) {
	var record Record
	lines := strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n")

	fields := make(map[string]any)
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, ": ", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])

		if val == "null" {
			fields[key] = nil
		} else if len(val) >= 2 && strings.HasPrefix(val, `"`) && strings.HasSuffix(val, `"`) {
			unquoted := val[1 : len(val)-1]
			unquoted = strings.ReplaceAll(unquoted, `\"`, `"`)
			unquoted = strings.ReplaceAll(unquoted, `\\`, `\`)
			unquoted = strings.ReplaceAll(unquoted, `\n`, "\n")
			fields[key] = unquoted
		} else {
			fields[key] = val
		}
	}

	if v, ok := fields["id"].(string); ok {
		record.ID = v
	}
	if v, ok := fields["question"].(string); ok {
		record.Question = v
	}
	if v, ok := fields["context"].(string); ok {
		record.Context = v
	}
	if v, ok := fields["orchestration_summary"].(string); ok {
		record.OrchestrationSummary = v
	}
	if v, ok := fields["status"].(string); ok {
		record.Status = Status(v)
	}
	if v, ok := fields["created_at"].(string); ok {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			record.CreatedAt = t
		}
	}
	if v, ok := fields["created_by"].(string); ok {
		record.CreatedBy = v
	}
	if v, ok := fields["responded_at"].(string); ok && v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			record.RespondedAt = &t
		}
	}
	if v, ok := fields["response"].(string); ok && v != "" {
		record.Response = &v
	}

	return record, nil
}
