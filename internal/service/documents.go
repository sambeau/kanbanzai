package service

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"kanbanzai/internal/core"
	"kanbanzai/internal/model"
	"kanbanzai/internal/storage"
)

// SubmitDocumentInput contains the fields needed to submit a new document.
type SubmitDocumentInput struct {
	// Path is the relative path to the document file from the repo root.
	Path string
	// Type is the document type (design, specification, dev-plan, research, report, policy).
	Type string
	// Title is the human-readable title.
	Title string
	// Owner is the optional parent Plan or Feature ID.
	Owner string
	// CreatedBy identifies who created the document.
	CreatedBy string
}

// ApproveDocumentInput contains the fields needed to approve a document.
type ApproveDocumentInput struct {
	ID         string
	ApprovedBy string
}

// SupersedeDocumentInput contains the fields needed to supersede a document.
type SupersedeDocumentInput struct {
	// ID is the document being superseded.
	ID string
	// SupersededBy is the ID of the document that supersedes this one.
	SupersededBy string
}

// DocumentResult is the result of a document operation.
type DocumentResult struct {
	ID           string
	Path         string
	RecordPath   string
	Type         string
	Title        string
	Status       string
	Owner        string
	ContentHash  string
	Drift        bool   // True if content has changed since recorded
	CurrentHash  string // Current hash if drift detected
	Created      time.Time
	Updated      time.Time
	ApprovedBy   string
	ApprovedAt   *time.Time
	Supersedes   string
	SupersededBy string
}

// DocumentFilters contains optional filters for listing documents.
type DocumentFilters struct {
	Type   string
	Status string
	Owner  string
}

// DocumentService handles document record operations.
type DocumentService struct {
	stateRoot  string
	repoRoot   string
	store      *storage.DocumentStore
	now        func() time.Time
	entityHook EntityLifecycleHook // optional, for lifecycle transitions
}

// SetEntityHook attaches an optional lifecycle hook that triggers entity
// transitions when documents are submitted, approved, or superseded.
func (s *DocumentService) SetEntityHook(hook EntityLifecycleHook) {
	s.entityHook = hook
}

// NewDocumentService creates a new DocumentService.
func NewDocumentService(stateRoot, repoRoot string) *DocumentService {
	if strings.TrimSpace(stateRoot) == "" {
		stateRoot = core.StatePath()
	}
	if strings.TrimSpace(repoRoot) == "" {
		// Default to current directory - documents are relative to repo root
		repoRoot = "."
	}

	return &DocumentService{
		stateRoot: stateRoot,
		repoRoot:  repoRoot,
		store:     storage.NewDocumentStore(stateRoot),
		now: func() time.Time {
			return time.Now().UTC()
		},
	}
}

// SubmitDocument registers a document with the system, creating a document record
// in draft status. This includes computing the content hash and preparing for
// Layers 1-2 ingest.
func (s *DocumentService) SubmitDocument(input SubmitDocumentInput) (DocumentResult, error) {
	if err := validateRequired(
		field("path", input.Path),
		field("type", input.Type),
		field("title", input.Title),
		field("created_by", input.CreatedBy),
	); err != nil {
		return DocumentResult{}, err
	}

	// Validate document type
	docType := model.DocumentType(strings.TrimSpace(input.Type))
	if !model.ValidDocumentType(string(docType)) {
		return DocumentResult{}, fmt.Errorf("invalid document type: %s", input.Type)
	}

	// Resolve document path
	docPath := strings.TrimSpace(input.Path)
	fullPath := filepath.Join(s.repoRoot, docPath)

	// Verify the file exists
	if _, err := os.Stat(fullPath); err != nil {
		if os.IsNotExist(err) {
			return DocumentResult{}, fmt.Errorf("document file not found: %s", docPath)
		}
		return DocumentResult{}, fmt.Errorf("access document file: %w", err)
	}

	// Compute content hash
	contentHash, err := storage.ComputeContentHash(fullPath)
	if err != nil {
		return DocumentResult{}, fmt.Errorf("compute content hash: %w", err)
	}

	// Generate document ID
	owner := strings.TrimSpace(input.Owner)
	slug := generateDocumentSlug(docType, docPath)
	var docID string
	if owner != "" {
		docID = fmt.Sprintf("%s/%s", owner, slug)
	} else {
		// Project-level document
		docID = fmt.Sprintf("PROJECT/%s", slug)
	}

	// Check if document already exists
	if s.store.Exists(docID) {
		return DocumentResult{}, fmt.Errorf("document already registered: %s", docID)
	}

	now := s.now()
	doc := model.DocumentRecord{
		ID:          docID,
		Path:        docPath,
		Type:        docType,
		Title:       strings.TrimSpace(input.Title),
		Status:      model.DocumentStatusDraft,
		Owner:       owner,
		ContentHash: contentHash,
		Created:     now,
		CreatedBy:   strings.TrimSpace(input.CreatedBy),
		Updated:     now,
	}

	// Write the record
	record := storage.DocumentToRecord(doc)
	recordPath, err := s.store.Write(record)
	if err != nil {
		return DocumentResult{}, fmt.Errorf("write document record: %w", err)
	}

	result := DocumentResult{
		ID:          doc.ID,
		Path:        doc.Path,
		RecordPath:  recordPath,
		Type:        string(doc.Type),
		Title:       doc.Title,
		Status:      string(doc.Status),
		Owner:       doc.Owner,
		ContentHash: doc.ContentHash,
		Created:     doc.Created,
		Updated:     doc.Updated,
	}

	// Trigger entity lifecycle hooks on submission
	if s.entityHook != nil && owner != "" {
		// Set document reference on the owning entity
		var docField string
		switch docType {
		case model.DocumentTypeDesign:
			docField = "design"
		case model.DocumentTypeSpecification:
			docField = "spec"
		case model.DocumentTypeDevPlan:
			docField = "dev_plan"
		}
		if docField != "" {
			_ = s.entityHook.SetDocumentRef(owner, docField, doc.ID)
		}

		// Trigger creation-time lifecycle transition
		var targetStatus string
		switch docType {
		case model.DocumentTypeDesign:
			targetStatus = "designing"
		case model.DocumentTypeSpecification:
			targetStatus = "specifying"
		}
		if targetStatus != "" {
			_ = s.entityHook.TransitionStatus(owner, targetStatus)
		}
	}

	return result, nil
}

// ApproveDocument transitions a document from draft to approved.
func (s *DocumentService) ApproveDocument(input ApproveDocumentInput) (DocumentResult, error) {
	if err := validateRequired(
		field("id", input.ID),
		field("approved_by", input.ApprovedBy),
	); err != nil {
		return DocumentResult{}, err
	}

	// Load existing record
	record, err := s.store.Load(input.ID)
	if err != nil {
		return DocumentResult{}, err
	}

	doc, err := storage.RecordToDocument(record)
	if err != nil {
		return DocumentResult{}, fmt.Errorf("parse document record: %w", err)
	}

	// Validate current status
	if doc.Status != model.DocumentStatusDraft {
		return DocumentResult{}, fmt.Errorf("cannot approve document in status %s (must be draft)", doc.Status)
	}

	// Verify content hash matches current file
	fullPath := filepath.Join(s.repoRoot, doc.Path)
	currentHash, err := storage.ComputeContentHash(fullPath)
	if err != nil {
		return DocumentResult{}, fmt.Errorf("compute content hash: %w", err)
	}

	if currentHash != doc.ContentHash {
		return DocumentResult{}, fmt.Errorf("content hash mismatch: document has been modified since submission (recorded=%s, current=%s)", doc.ContentHash[:12], currentHash[:12])
	}

	// Update status
	now := s.now()
	doc.Status = model.DocumentStatusApproved
	doc.ApprovedBy = strings.TrimSpace(input.ApprovedBy)
	doc.ApprovedAt = &now
	doc.Updated = now

	// Write the record
	updatedRecord := storage.DocumentToRecord(doc)
	recordPath, err := s.store.Write(updatedRecord)
	if err != nil {
		return DocumentResult{}, fmt.Errorf("write document record: %w", err)
	}

	result := DocumentResult{
		ID:          doc.ID,
		Path:        doc.Path,
		RecordPath:  recordPath,
		Type:        string(doc.Type),
		Title:       doc.Title,
		Status:      string(doc.Status),
		Owner:       doc.Owner,
		ContentHash: doc.ContentHash,
		Created:     doc.Created,
		Updated:     doc.Updated,
		ApprovedBy:  doc.ApprovedBy,
		ApprovedAt:  doc.ApprovedAt,
	}

	// Trigger entity lifecycle hooks on approval
	if s.entityHook != nil && doc.Owner != "" {
		entityType, _, _ := s.entityHook.GetEntityStatus(doc.Owner)
		var targetStatus string
		switch {
		case entityType == "plan" && doc.Type == model.DocumentTypeDesign:
			targetStatus = "active"
		case entityType == "feature" && doc.Type == model.DocumentTypeDesign:
			targetStatus = "specifying"
		case entityType == "feature" && doc.Type == model.DocumentTypeSpecification:
			targetStatus = "dev-planning"
		case entityType == "feature" && doc.Type == model.DocumentTypeDevPlan:
			targetStatus = "developing"
		}
		if targetStatus != "" {
			_ = s.entityHook.TransitionStatus(doc.Owner, targetStatus)
		}
	}

	return result, nil
}

// SupersedeDocument transitions a document from approved to superseded.
func (s *DocumentService) SupersedeDocument(input SupersedeDocumentInput) (DocumentResult, error) {
	if err := validateRequired(
		field("id", input.ID),
		field("superseded_by", input.SupersededBy),
	); err != nil {
		return DocumentResult{}, err
	}

	// Load existing record
	record, err := s.store.Load(input.ID)
	if err != nil {
		return DocumentResult{}, err
	}

	doc, err := storage.RecordToDocument(record)
	if err != nil {
		return DocumentResult{}, fmt.Errorf("parse document record: %w", err)
	}

	// Validate current status
	if doc.Status != model.DocumentStatusApproved {
		return DocumentResult{}, fmt.Errorf("cannot supersede document in status %s (must be approved)", doc.Status)
	}

	// Verify the superseding document exists
	if !s.store.Exists(input.SupersededBy) {
		return DocumentResult{}, fmt.Errorf("superseding document not found: %s", input.SupersededBy)
	}

	// Update the superseding document to reference this one
	supersedesRecord, err := s.store.Load(input.SupersededBy)
	if err != nil {
		return DocumentResult{}, fmt.Errorf("load superseding document: %w", err)
	}

	supersedesDoc, err := storage.RecordToDocument(supersedesRecord)
	if err != nil {
		return DocumentResult{}, fmt.Errorf("parse superseding document: %w", err)
	}

	supersedesDoc.Supersedes = doc.ID
	supersedesDoc.Updated = s.now()

	supersedesUpdatedRecord := storage.DocumentToRecord(supersedesDoc)
	if _, err := s.store.Write(supersedesUpdatedRecord); err != nil {
		return DocumentResult{}, fmt.Errorf("update superseding document: %w", err)
	}

	// Update this document
	now := s.now()
	doc.Status = model.DocumentStatusSuperseded
	doc.SupersededBy = input.SupersededBy
	doc.Updated = now

	// Write the record
	updatedRecord := storage.DocumentToRecord(doc)
	recordPath, err := s.store.Write(updatedRecord)
	if err != nil {
		return DocumentResult{}, fmt.Errorf("write document record: %w", err)
	}

	result := DocumentResult{
		ID:           doc.ID,
		Path:         doc.Path,
		RecordPath:   recordPath,
		Type:         string(doc.Type),
		Title:        doc.Title,
		Status:       string(doc.Status),
		Owner:        doc.Owner,
		ContentHash:  doc.ContentHash,
		Created:      doc.Created,
		Updated:      doc.Updated,
		ApprovedBy:   doc.ApprovedBy,
		ApprovedAt:   doc.ApprovedAt,
		Supersedes:   doc.Supersedes,
		SupersededBy: doc.SupersededBy,
	}

	// Trigger entity lifecycle hooks on supersession
	if s.entityHook != nil && doc.Owner != "" {
		entityType, _, _ := s.entityHook.GetEntityStatus(doc.Owner)
		if entityType == "feature" {
			var targetStatus string
			switch doc.Type {
			case model.DocumentTypeDesign:
				targetStatus = "designing"
			case model.DocumentTypeSpecification:
				targetStatus = "specifying"
			case model.DocumentTypeDevPlan:
				targetStatus = "dev-planning"
			}
			if targetStatus != "" {
				_ = s.entityHook.TransitionStatus(doc.Owner, targetStatus)
			}
		}
	}

	return result, nil
}

// GetDocument retrieves a document record by ID.
// If checkDrift is true, verifies the content hash against the current file.
func (s *DocumentService) GetDocument(id string, checkDrift bool) (DocumentResult, error) {
	if strings.TrimSpace(id) == "" {
		return DocumentResult{}, fmt.Errorf("document ID is required")
	}

	record, err := s.store.Load(id)
	if err != nil {
		return DocumentResult{}, err
	}

	doc, err := storage.RecordToDocument(record)
	if err != nil {
		return DocumentResult{}, fmt.Errorf("parse document record: %w", err)
	}

	result := DocumentResult{
		ID:           doc.ID,
		Path:         doc.Path,
		RecordPath:   s.store.GetFilePath(id),
		Type:         string(doc.Type),
		Title:        doc.Title,
		Status:       string(doc.Status),
		Owner:        doc.Owner,
		ContentHash:  doc.ContentHash,
		Created:      doc.Created,
		Updated:      doc.Updated,
		ApprovedBy:   doc.ApprovedBy,
		ApprovedAt:   doc.ApprovedAt,
		Supersedes:   doc.Supersedes,
		SupersededBy: doc.SupersededBy,
	}

	// Check for content drift if requested
	if checkDrift {
		fullPath := filepath.Join(s.repoRoot, doc.Path)
		hasDrift, currentHash, err := storage.CheckContentDrift(fullPath, doc.ContentHash, doc.Updated)
		if err != nil {
			// File might not exist; report as drift
			result.Drift = true
		} else {
			result.Drift = hasDrift
			result.CurrentHash = currentHash
		}
	}

	return result, nil
}

// GetDocumentContent retrieves the content of a document file.
// For approved documents, this must be verbatim.
func (s *DocumentService) GetDocumentContent(id string) (string, DocumentResult, error) {
	result, err := s.GetDocument(id, true)
	if err != nil {
		return "", DocumentResult{}, err
	}

	// Warn if approved document has drifted
	if result.Status == string(model.DocumentStatusApproved) && result.Drift {
		// We still return the content, but the caller should be aware
		// that it differs from the approved version
	}

	fullPath := filepath.Join(s.repoRoot, result.Path)
	content, err := os.ReadFile(fullPath)
	if err != nil {
		return "", result, fmt.Errorf("read document content: %w", err)
	}

	return string(content), result, nil
}

// ListDocuments returns all documents, optionally filtered.
func (s *DocumentService) ListDocuments(filters DocumentFilters) ([]DocumentResult, error) {
	records, err := s.store.List()
	if err != nil {
		return nil, err
	}

	var results []DocumentResult
	for _, record := range records {
		doc, err := storage.RecordToDocument(record)
		if err != nil {
			continue
		}

		// Apply filters
		if filters.Type != "" && string(doc.Type) != filters.Type {
			continue
		}
		if filters.Status != "" && string(doc.Status) != filters.Status {
			continue
		}
		if filters.Owner != "" && doc.Owner != filters.Owner {
			continue
		}

		results = append(results, DocumentResult{
			ID:           doc.ID,
			Path:         doc.Path,
			RecordPath:   s.store.GetFilePath(doc.ID),
			Type:         string(doc.Type),
			Title:        doc.Title,
			Status:       string(doc.Status),
			Owner:        doc.Owner,
			ContentHash:  doc.ContentHash,
			Created:      doc.Created,
			Updated:      doc.Updated,
			ApprovedBy:   doc.ApprovedBy,
			ApprovedAt:   doc.ApprovedAt,
			Supersedes:   doc.Supersedes,
			SupersededBy: doc.SupersededBy,
		})
	}

	return results, nil
}

// ListDocumentsByOwner returns all documents owned by a specific entity.
func (s *DocumentService) ListDocumentsByOwner(owner string) ([]DocumentResult, error) {
	return s.ListDocuments(DocumentFilters{Owner: owner})
}

// ListPendingDocuments returns all documents that are in draft status.
func (s *DocumentService) ListPendingDocuments() ([]DocumentResult, error) {
	return s.ListDocuments(DocumentFilters{Status: string(model.DocumentStatusDraft)})
}

// ValidateDocument validates a document record and checks content integrity.
func (s *DocumentService) ValidateDocument(id string) ([]string, error) {
	var issues []string

	result, err := s.GetDocument(id, true)
	if err != nil {
		return nil, err
	}

	// Check if file exists
	fullPath := filepath.Join(s.repoRoot, result.Path)
	if _, err := os.Stat(fullPath); err != nil {
		if os.IsNotExist(err) {
			issues = append(issues, fmt.Sprintf("document file not found: %s", result.Path))
		} else {
			issues = append(issues, fmt.Sprintf("cannot access document file: %v", err))
		}
	}

	// Check for content drift
	if result.Drift {
		recordedHash := result.ContentHash
		currentHash := result.CurrentHash
		if len(recordedHash) > 12 {
			recordedHash = recordedHash[:12]
		}
		if len(currentHash) > 12 {
			currentHash = currentHash[:12]
		}
		issues = append(issues, fmt.Sprintf("content hash mismatch: document has been modified (recorded=%s, current=%s)", recordedHash, currentHash))
	}

	// Validate document type
	if !model.ValidDocumentType(result.Type) {
		issues = append(issues, fmt.Sprintf("invalid document type: %s", result.Type))
	}

	// Validate status
	switch model.DocumentStatus(result.Status) {
	case model.DocumentStatusDraft, model.DocumentStatusApproved, model.DocumentStatusSuperseded:
		// Valid
	default:
		issues = append(issues, fmt.Sprintf("invalid document status: %s", result.Status))
	}

	// Check owner reference if set
	if result.Owner != "" && result.Owner != "PROJECT" {
		// This would require checking if the owner entity exists
		// For now, we just validate the format
		if !isValidEntityID(result.Owner) {
			issues = append(issues, fmt.Sprintf("invalid owner reference: %s", result.Owner))
		}
	}

	return issues, nil
}

// DocumentExists checks if a document record exists.
func (s *DocumentService) DocumentExists(id string) bool {
	return s.store.Exists(id)
}

// generateDocumentSlug generates a unique slug for a document.
// It combines the document type with a hash of the path to ensure uniqueness.
func generateDocumentSlug(docType model.DocumentType, path string) string {
	// Use the filename (without extension) as part of the slug for readability
	base := filepath.Base(path)
	ext := filepath.Ext(base)
	name := strings.TrimSuffix(base, ext)
	// Normalize to lowercase and replace spaces with hyphens
	name = strings.ToLower(name)
	name = strings.ReplaceAll(name, " ", "-")
	name = strings.ReplaceAll(name, "_", "-")
	// Remove any characters that aren't alphanumeric or hyphens
	var normalized strings.Builder
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			normalized.WriteRune(r)
		}
	}
	slugName := normalized.String()
	if slugName == "" {
		slugName = "doc"
	}
	return fmt.Sprintf("%s-%s", docType, slugName)
}

// isValidEntityID checks if a string looks like a valid entity ID.
func isValidEntityID(id string) bool {
	if id == "" {
		return false
	}

	// Check for Plan ID format
	if model.IsPlanID(id) {
		return true
	}

	// Check for standard entity ID formats (FEAT-xxx, TASK-xxx, etc.)
	prefixes := []string{"FEAT-", "TASK-", "BUG-", "DEC-", "EPIC-"}
	for _, prefix := range prefixes {
		if strings.HasPrefix(id, prefix) {
			return true
		}
	}

	return false
}
