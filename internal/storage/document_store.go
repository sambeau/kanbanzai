package storage

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"kanbanzai/internal/fsutil"
	"kanbanzai/internal/model"
)

// DocumentDir is the directory name for document records within state.
const DocumentDir = "documents"

// DocumentRecord is the storage representation of a document metadata record.
type DocumentRecord struct {
	ID       string
	Fields   map[string]any
	FileHash string // SHA-256 hex digest of file contents at load time; used for optimistic locking
}

// DocumentStore handles storage and retrieval of document metadata records.
type DocumentStore struct {
	root string
}

// NewDocumentStore creates a new DocumentStore rooted at the given path.
// The path should be the .kbz/state directory.
func NewDocumentStore(root string) *DocumentStore {
	return &DocumentStore{root: root}
}

// Write persists a document record to disk.
func (s *DocumentStore) Write(record DocumentRecord) (string, error) {
	if err := validateDocumentRecord(record); err != nil {
		return "", err
	}

	dir := filepath.Join(s.root, DocumentDir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("create documents directory: %w", err)
	}

	path := filepath.Join(dir, documentFileName(record.ID))

	// Optimistic locking: if FileHash is set, verify the file hasn't changed.
	if record.FileHash != "" {
		current, err := os.ReadFile(path)
		if err == nil {
			h := sha256.Sum256(current)
			if hex.EncodeToString(h[:]) != record.FileHash {
				return "", fmt.Errorf("write document %s: %w", record.ID, ErrConflict)
			}
		}
		// If the file doesn't exist (os.ErrNotExist), skip the check — new document.
	}

	content, err := MarshalCanonicalYAML("document_record", record.Fields)
	if err != nil {
		return "", fmt.Errorf("marshal document record: %w", err)
	}

	if err := fsutil.WriteFileAtomic(path, []byte(content), 0o644); err != nil {
		return "", fmt.Errorf("write document record: %w", err)
	}

	return path, nil
}

// Load reads a document record from disk by ID.
func (s *DocumentStore) Load(id string) (DocumentRecord, error) {
	record := DocumentRecord{ID: id}

	if strings.TrimSpace(id) == "" {
		return record, errors.New("document ID is required")
	}

	path := filepath.Join(s.root, DocumentDir, documentFileName(id))
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return record, fmt.Errorf("document record not found: %s", id)
		}
		return record, fmt.Errorf("read document record: %w", err)
	}

	fields, err := UnmarshalCanonicalYAML(string(data))
	if err != nil {
		return record, fmt.Errorf("unmarshal document record: %w", err)
	}

	record.Fields = fields

	h := sha256.Sum256(data)
	record.FileHash = hex.EncodeToString(h[:])

	return record, nil
}

// Delete removes a document record from disk.
func (s *DocumentStore) Delete(id string) error {
	if strings.TrimSpace(id) == "" {
		return errors.New("document ID is required")
	}

	path := filepath.Join(s.root, DocumentDir, documentFileName(id))
	if err := os.Remove(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil // Already deleted
		}
		return fmt.Errorf("delete document record: %w", err)
	}

	return nil
}

// List returns all document records in the store.
func (s *DocumentStore) List() ([]DocumentRecord, error) {
	dir := filepath.Join(s.root, DocumentDir)

	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("read documents directory: %w", err)
	}

	var records []DocumentRecord
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}

		id := documentIDFromFileName(entry.Name())
		record, err := s.Load(id)
		if err != nil {
			continue // Skip invalid records
		}
		records = append(records, record)
	}

	return records, nil
}

// Exists checks if a document record exists.
func (s *DocumentStore) Exists(id string) bool {
	path := filepath.Join(s.root, DocumentDir, documentFileName(id))
	_, err := os.Stat(path)
	return err == nil
}

// GetFilePath returns the filesystem path to a document record file.
func (s *DocumentStore) GetFilePath(id string) string {
	return filepath.Join(s.root, DocumentDir, documentFileName(id))
}

// validateDocumentRecord checks that a document record is valid for storage.
func validateDocumentRecord(record DocumentRecord) error {
	if strings.TrimSpace(record.ID) == "" {
		return errors.New("document ID is required")
	}
	if len(record.Fields) == 0 {
		return errors.New("document fields are required")
	}

	id, ok := record.Fields["id"]
	if !ok {
		return errors.New("document fields must include id")
	}
	if fmt.Sprint(id) != record.ID {
		return fmt.Errorf("document id mismatch: record=%q fields=%q", record.ID, fmt.Sprint(id))
	}

	// Validate required fields
	requiredFields := []string{"path", "type", "title", "status", "content_hash", "created", "created_by", "updated"}
	for _, field := range requiredFields {
		if _, ok := record.Fields[field]; !ok {
			return fmt.Errorf("document record missing required field: %s", field)
		}
	}

	// Validate document type
	docType, _ := record.Fields["type"].(string)
	if !model.ValidDocumentType(docType) {
		return fmt.Errorf("invalid document type: %s", docType)
	}

	// Validate status
	status, _ := record.Fields["status"].(string)
	switch model.DocumentStatus(status) {
	case model.DocumentStatusDraft, model.DocumentStatusApproved, model.DocumentStatusSuperseded:
		// Valid
	default:
		return fmt.Errorf("invalid document status: %s", status)
	}

	return nil
}

// documentFileName converts a document ID to a filename.
// Document IDs have format {owner-id}/{slug}, and we replace / with --
func documentFileName(id string) string {
	// Replace / with -- to create a flat filename
	safe := strings.ReplaceAll(id, "/", "--")
	return safe + ".yaml"
}

// documentIDFromFileName converts a filename back to a document ID.
func documentIDFromFileName(filename string) string {
	// Remove .yaml suffix
	name := strings.TrimSuffix(filename, ".yaml")
	// Replace -- back to /
	return strings.ReplaceAll(name, "--", "/")
}

// ComputeContentHash computes the SHA-256 hash of a file's content.
func ComputeContentHash(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("open file for hashing: %w", err)
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", fmt.Errorf("read file for hashing: %w", err)
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}

// GetFileMtime returns the modification time of a file.
func GetFileMtime(path string) (time.Time, error) {
	info, err := os.Stat(path)
	if err != nil {
		return time.Time{}, err
	}
	return info.ModTime(), nil
}

// CheckContentDrift checks if a document's content has changed since it was recorded.
// Returns (hasDrift, currentHash, error).
func CheckContentDrift(docPath string, recordedHash string, recordedUpdated time.Time) (bool, string, error) {
	// Get file modification time
	mtime, err := GetFileMtime(docPath)
	if err != nil {
		return false, "", fmt.Errorf("get file mtime: %w", err)
	}

	// If file hasn't been modified since record was updated, no drift
	if !mtime.After(recordedUpdated) {
		return false, recordedHash, nil
	}

	// File is newer, recompute hash
	currentHash, err := ComputeContentHash(docPath)
	if err != nil {
		return false, "", fmt.Errorf("compute content hash: %w", err)
	}

	// Compare hashes
	if currentHash != recordedHash {
		return true, currentHash, nil
	}

	return false, currentHash, nil
}

// DocumentToRecord converts a model.DocumentRecord to a storage DocumentRecord.
func DocumentToRecord(doc model.DocumentRecord) DocumentRecord {
	fields := make(map[string]any)

	fields["id"] = doc.ID
	fields["path"] = doc.Path
	fields["type"] = string(doc.Type)
	fields["title"] = doc.Title
	fields["status"] = string(doc.Status)

	if doc.Owner != "" {
		fields["owner"] = doc.Owner
	}
	if doc.ApprovedBy != "" {
		fields["approved_by"] = doc.ApprovedBy
	}
	if doc.ApprovedAt != nil {
		fields["approved_at"] = doc.ApprovedAt.Format(time.RFC3339)
	}

	fields["content_hash"] = doc.ContentHash

	if doc.Supersedes != "" {
		fields["supersedes"] = doc.Supersedes
	}
	if doc.SupersededBy != "" {
		fields["superseded_by"] = doc.SupersededBy
	}

	fields["created"] = doc.Created.Format(time.RFC3339)
	fields["created_by"] = doc.CreatedBy
	fields["updated"] = doc.Updated.Format(time.RFC3339)

	return DocumentRecord{
		ID:     doc.ID,
		Fields: fields,
	}
}

// RecordToDocument converts a storage DocumentRecord to a model.DocumentRecord.
func RecordToDocument(record DocumentRecord) (model.DocumentRecord, error) {
	doc := model.DocumentRecord{}

	doc.ID = record.ID

	if v, ok := record.Fields["path"].(string); ok {
		doc.Path = v
	}
	if v, ok := record.Fields["type"].(string); ok {
		doc.Type = model.DocumentType(v)
	}
	if v, ok := record.Fields["title"].(string); ok {
		doc.Title = v
	}
	if v, ok := record.Fields["status"].(string); ok {
		doc.Status = model.DocumentStatus(v)
	}
	if v, ok := record.Fields["owner"].(string); ok {
		doc.Owner = v
	}
	if v, ok := record.Fields["approved_by"].(string); ok {
		doc.ApprovedBy = v
	}
	if v, ok := record.Fields["approved_at"].(string); ok {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			doc.ApprovedAt = &t
		}
	}
	if v, ok := record.Fields["content_hash"].(string); ok {
		doc.ContentHash = v
	}
	if v, ok := record.Fields["supersedes"].(string); ok {
		doc.Supersedes = v
	}
	if v, ok := record.Fields["superseded_by"].(string); ok {
		doc.SupersededBy = v
	}
	if v, ok := record.Fields["created"].(string); ok {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			doc.Created = t
		}
	}
	if v, ok := record.Fields["created_by"].(string); ok {
		doc.CreatedBy = v
	}
	if v, ok := record.Fields["updated"].(string); ok {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			doc.Updated = t
		}
	}

	return doc, nil
}
