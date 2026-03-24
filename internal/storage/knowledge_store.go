package storage

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"kanbanzai/internal/fsutil"
)

// KnowledgeDir is the directory name for knowledge entry records within state.
const KnowledgeDir = "knowledge"

// KnowledgeRecord is the storage representation of a knowledge entry.
type KnowledgeRecord struct {
	ID       string
	Fields   map[string]any
	FileHash string // SHA-256 hex digest of file contents at load time; used for optimistic locking
}

// KnowledgeStore handles storage and retrieval of knowledge entry records.
type KnowledgeStore struct {
	root string
}

// NewKnowledgeStore creates a new KnowledgeStore rooted at the given path.
// The path should be the .kbz/state directory.
func NewKnowledgeStore(root string) *KnowledgeStore {
	return &KnowledgeStore{root: root}
}

// Write persists a knowledge record to disk.
func (s *KnowledgeStore) Write(record KnowledgeRecord) (string, error) {
	if err := validateKnowledgeRecord(record); err != nil {
		return "", err
	}

	dir := filepath.Join(s.root, KnowledgeDir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("create knowledge directory: %w", err)
	}

	path := filepath.Join(dir, record.ID+".yaml")

	// Optimistic locking: if FileHash is set, verify the file hasn't changed.
	if record.FileHash != "" {
		current, err := os.ReadFile(path)
		if err == nil {
			h := sha256.Sum256(current)
			if hex.EncodeToString(h[:]) != record.FileHash {
				return "", fmt.Errorf("write knowledge entry %s: %w", record.ID, ErrConflict)
			}
		}
		// If the file doesn't exist (os.ErrNotExist), skip the check — new entry.
	}

	content, err := MarshalCanonicalYAML("knowledge_entry", record.Fields)
	if err != nil {
		return "", fmt.Errorf("marshal knowledge entry: %w", err)
	}

	if err := fsutil.WriteFileAtomic(path, []byte(content), 0o644); err != nil {
		return "", fmt.Errorf("write knowledge entry: %w", err)
	}

	return path, nil
}

// Load reads a knowledge record from disk by ID.
func (s *KnowledgeStore) Load(id string) (KnowledgeRecord, error) {
	record := KnowledgeRecord{ID: id}

	if strings.TrimSpace(id) == "" {
		return record, errors.New("knowledge entry ID is required")
	}

	path := filepath.Join(s.root, KnowledgeDir, id+".yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return record, fmt.Errorf("knowledge entry not found: %s", id)
		}
		return record, fmt.Errorf("read knowledge entry: %w", err)
	}

	fields, err := UnmarshalCanonicalYAML(string(data))
	if err != nil {
		return record, fmt.Errorf("unmarshal knowledge entry: %w", err)
	}

	record.Fields = fields

	h := sha256.Sum256(data)
	record.FileHash = hex.EncodeToString(h[:])

	return record, nil
}

// LoadAll returns all knowledge records in the store.
func (s *KnowledgeStore) LoadAll() ([]KnowledgeRecord, error) {
	dir := filepath.Join(s.root, KnowledgeDir)

	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("read knowledge directory: %w", err)
	}

	var records []KnowledgeRecord
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}

		id := strings.TrimSuffix(entry.Name(), ".yaml")
		record, err := s.Load(id)
		if err != nil {
			continue // Skip unreadable records
		}
		records = append(records, record)
	}

	return records, nil
}

// Delete removes a knowledge record from disk.
func (s *KnowledgeStore) Delete(id string) error {
	if strings.TrimSpace(id) == "" {
		return errors.New("knowledge entry ID is required")
	}

	path := filepath.Join(s.root, KnowledgeDir, id+".yaml")
	if err := os.Remove(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil // Already deleted
		}
		return fmt.Errorf("delete knowledge entry: %w", err)
	}

	return nil
}

// validateKnowledgeRecord checks that a knowledge record is valid for storage.
func validateKnowledgeRecord(record KnowledgeRecord) error {
	if strings.TrimSpace(record.ID) == "" {
		return errors.New("knowledge entry ID is required")
	}
	if len(record.Fields) == 0 {
		return errors.New("knowledge entry fields are required")
	}

	id, ok := record.Fields["id"]
	if !ok {
		return errors.New("knowledge entry fields must include id")
	}
	if fmt.Sprint(id) != record.ID {
		return fmt.Errorf("knowledge entry id mismatch: record=%q fields=%q", record.ID, fmt.Sprint(id))
	}

	return nil
}
