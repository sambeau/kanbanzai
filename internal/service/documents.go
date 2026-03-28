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

// RefreshInput identifies the document to refresh. ID is preferred; Path is the fallback.
type RefreshInput struct {
	ID   string
	Path string
}

// RefreshResult is the outcome of a RefreshContentHash call.
type RefreshResult struct {
	ID               string
	Path             string
	Changed          bool
	OldHash          string
	NewHash          string
	Status           string // final status after refresh
	StatusTransition string // e.g., "approved → draft", or "" if unchanged
}

// DocumentResult is the result of a document operation.
// DocEntityTransition records an entity lifecycle transition triggered by a
// document operation (approval, supersession). It is used by 2.0 tools to
// report status_transition side effects without modifying the EntityLifecycleHook
// interface (spec §8.3, Track B task B.9 / Track I prerequisite).
type DocEntityTransition struct {
	// EntityID is the owning entity that was transitioned (feature or plan ID).
	EntityID string
	// EntityType is the type of the owning entity ("feature" or "plan").
	EntityType string
	// FromStatus is the entity's status before the transition.
	FromStatus string
	// ToStatus is the entity's status after the transition.
	ToStatus string
}

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
	// EntityTransition is set when this operation triggered a lifecycle
	// transition on the owning entity. Nil when no transition occurred.
	// Read by 2.0 tools (Track I) to push status_transition side effects.
	EntityTransition *DocEntityTransition
}

// DocumentFilters contains optional filters for listing documents.
type DocumentFilters struct {
	Type   string
	Status string
	Owner  string
}

// DocumentService handles document record operations.
type DocumentService struct {
	stateRoot       string
	repoRoot        string
	store           *storage.DocumentStore
	now             func() time.Time
	entityHook      EntityLifecycleHook  // optional, for lifecycle transitions
	intelligenceSvc *IntelligenceService // optional, for auto-ingest on submit
}

// RepoRoot returns the repository root path used by this service.
func (s *DocumentService) RepoRoot() string {
	return s.repoRoot
}

// SetEntityHook attaches an optional lifecycle hook that triggers entity
// transitions when documents are submitted, approved, or superseded.
func (s *DocumentService) SetEntityHook(hook EntityLifecycleHook) {
	s.entityHook = hook
}

// SetIntelligenceService attaches an optional intelligence service that
// automatically ingests documents (Layers 1-2) when they are submitted.
func (s *DocumentService) SetIntelligenceService(svc *IntelligenceService) {
	s.intelligenceSvc = svc
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

	// Write the record (new document, no fileHash for optimistic locking)
	record := storage.DocumentToRecord(doc, "")
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

	// Best-effort ingest — run Layers 1-2 if intelligence service is available.
	// Don't fail the submission if ingest fails.
	if s.intelligenceSvc != nil {
		s.intelligenceSvc.IngestDocument(doc.ID, doc.Path)
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

	doc := storage.RecordToDocument(record)

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

	// Write the record (preserve fileHash for optimistic locking)
	updatedRecord := storage.DocumentToRecord(doc, record.FileHash)
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

	// Trigger entity lifecycle hooks on approval and record any transition
	// in result.EntityTransition for 2.0 side-effect reporting (Track I).
	if s.entityHook != nil && doc.Owner != "" {
		entityType, currentStatus, getErr := s.entityHook.GetEntityStatus(doc.Owner)
		if getErr == nil {
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
			if targetStatus != "" && currentStatus != targetStatus {
				if err := s.entityHook.TransitionStatus(doc.Owner, targetStatus); err == nil {
					result.EntityTransition = &DocEntityTransition{
						EntityID:   doc.Owner,
						EntityType: entityType,
						FromStatus: currentStatus,
						ToStatus:   targetStatus,
					}
				}
			}
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

	doc := storage.RecordToDocument(record)

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

	supersedesDoc := storage.RecordToDocument(supersedesRecord)

	supersedesDoc.Supersedes = doc.ID
	supersedesDoc.Updated = s.now()

	supersedesUpdatedRecord := storage.DocumentToRecord(supersedesDoc, supersedesRecord.FileHash)
	if _, err := s.store.Write(supersedesUpdatedRecord); err != nil {
		return DocumentResult{}, fmt.Errorf("update superseding document: %w", err)
	}

	// Update this document
	now := s.now()
	doc.Status = model.DocumentStatusSuperseded
	doc.SupersededBy = input.SupersededBy
	doc.Updated = now

	// Write the record (preserve fileHash for optimistic locking)
	updatedRecord := storage.DocumentToRecord(doc, record.FileHash)
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

	// Trigger entity lifecycle hooks on supersession and record any backward
	// transition in result.EntityTransition for 2.0 side-effect reporting (Track I).
	if s.entityHook != nil && doc.Owner != "" {
		entityType, currentStatus, getErr := s.entityHook.GetEntityStatus(doc.Owner)
		if getErr == nil && entityType == "feature" {
			var targetStatus string
			switch doc.Type {
			case model.DocumentTypeDesign:
				targetStatus = "designing"
			case model.DocumentTypeSpecification:
				targetStatus = "specifying"
			case model.DocumentTypeDevPlan:
				targetStatus = "dev-planning"
			}
			if targetStatus != "" && currentStatus != targetStatus {
				if err := s.entityHook.TransitionStatus(doc.Owner, targetStatus); err == nil {
					result.EntityTransition = &DocEntityTransition{
						EntityID:   doc.Owner,
						EntityType: entityType,
						FromStatus: currentStatus,
						ToStatus:   targetStatus,
					}
				}
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

	doc := storage.RecordToDocument(record)

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
		doc := storage.RecordToDocument(record)

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
		if !isValidEntityID(result.Owner) {
			issues = append(issues, fmt.Sprintf("invalid owner reference: %s", result.Owner))
		} else if s.entityHook != nil {
			// Check if owner entity exists
			_, _, err := s.entityHook.GetEntityStatus(result.Owner)
			if err != nil {
				issues = append(issues, fmt.Sprintf("owner entity does not exist: %s", result.Owner))
			}
		}
	}

	return issues, nil
}

// SupersessionChain follows supersedes/superseded_by links to build the full
// chain of document versions. It walks backward (via supersedes) and forward
// (via superseded_by) from the given document, returning results ordered from
// oldest to newest.
func (s *DocumentService) SupersessionChain(docID string) ([]DocumentResult, error) {
	if strings.TrimSpace(docID) == "" {
		return nil, fmt.Errorf("document ID is required")
	}

	// Load the starting document.
	start, err := s.GetDocument(docID, false)
	if err != nil {
		return nil, err
	}

	// Walk backward through supersedes links to find the oldest ancestor.
	visited := map[string]bool{start.ID: true}
	var backward []DocumentResult
	cur := start
	for cur.Supersedes != "" {
		if visited[cur.Supersedes] {
			break // cycle guard
		}
		visited[cur.Supersedes] = true
		prev, err := s.GetDocument(cur.Supersedes, false)
		if err != nil {
			break // broken link — stop walking
		}
		backward = append(backward, prev)
		cur = prev
	}

	// Reverse backward so oldest is first.
	for i, j := 0, len(backward)-1; i < j; i, j = i+1, j-1 {
		backward[i], backward[j] = backward[j], backward[i]
	}

	// Walk forward through superseded_by links.
	var forward []DocumentResult
	cur = start
	for cur.SupersededBy != "" {
		if visited[cur.SupersededBy] {
			break // cycle guard
		}
		visited[cur.SupersededBy] = true
		next, err := s.GetDocument(cur.SupersededBy, false)
		if err != nil {
			break // broken link — stop walking
		}
		forward = append(forward, next)
		cur = next
	}

	// Combine: backward ancestors + start + forward successors.
	chain := make([]DocumentResult, 0, len(backward)+1+len(forward))
	chain = append(chain, backward...)
	chain = append(chain, start)
	chain = append(chain, forward...)

	return chain, nil
}

// DocumentExists checks if a document record exists.
func (s *DocumentService) DocumentExists(id string) bool {
	return s.store.Exists(id)
}

// RefreshContentHash recomputes the content hash of the document's file and
// updates the record if it has changed. If the document was approved, it is
// demoted to draft status.
func (s *DocumentService) RefreshContentHash(input RefreshInput) (RefreshResult, error) {
	id := strings.TrimSpace(input.ID)
	path := strings.TrimSpace(input.Path)

	if id == "" && path == "" {
		return RefreshResult{}, fmt.Errorf("id or path is required")
	}

	var record storage.DocumentRecord
	var err error

	if id != "" {
		record, err = s.store.Load(id)
		if err != nil {
			return RefreshResult{}, err
		}
	} else {
		records, listErr := s.store.List()
		if listErr != nil {
			return RefreshResult{}, fmt.Errorf("list documents: %w", listErr)
		}
		found := false
		for _, r := range records {
			doc := storage.RecordToDocument(r)
			if doc.Path == path {
				record = r
				found = true
				break
			}
		}
		if !found {
			return RefreshResult{}, fmt.Errorf("document not found: %s", path)
		}
	}

	doc := storage.RecordToDocument(record)
	fullPath := filepath.Join(s.repoRoot, doc.Path)

	currentHash, err := storage.ComputeContentHash(fullPath)
	if err != nil {
		return RefreshResult{}, fmt.Errorf("document file not found at path %s; verify the file exists before calling refresh", fullPath)
	}

	if currentHash == doc.ContentHash {
		return RefreshResult{
			Changed: false,
			OldHash: doc.ContentHash,
			NewHash: doc.ContentHash,
			Status:  string(doc.Status),
			ID:      doc.ID,
			Path:    doc.Path,
		}, nil
	}

	oldHash := doc.ContentHash
	doc.ContentHash = currentHash
	doc.Updated = s.now()

	var statusTransition string
	if doc.Status == model.DocumentStatusApproved {
		doc.Status = model.DocumentStatusDraft
		statusTransition = "approved → draft"
	}

	updatedRecord := storage.DocumentToRecord(doc, record.FileHash)
	if _, err := s.store.Write(updatedRecord); err != nil {
		return RefreshResult{}, fmt.Errorf("write document record: %w", err)
	}

	return RefreshResult{
		Changed:          true,
		OldHash:          oldHash,
		NewHash:          currentHash,
		Status:           string(doc.Status),
		StatusTransition: statusTransition,
		ID:               doc.ID,
		Path:             doc.Path,
	}, nil
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
