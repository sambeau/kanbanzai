package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/sambeau/kanbanzai/internal/config"
	"github.com/sambeau/kanbanzai/internal/core"
	"github.com/sambeau/kanbanzai/internal/fsutil"
	"github.com/sambeau/kanbanzai/internal/model"
	"github.com/sambeau/kanbanzai/internal/storage"
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
	// AutoApprove, when true, registers and approves the document in one call.
	// Only permitted for document types: dev-plan, research, report (FR-B04).
	AutoApprove bool
}

// MoveDocumentInput contains the parameters for the doc move operation.
type MoveDocumentInput struct {
	// ID is the document record ID to move.
	ID string
	// NewPath is the new relative path for the document file.
	NewPath string
}

// DeleteDocumentInput contains the parameters for the doc delete operation.
type DeleteDocumentInput struct {
	// ID is the document record ID to delete.
	ID string
	// Force, when true, allows deletion of approved documents.
	Force bool
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
	ID                string
	Path              string
	RecordPath        string
	Type              string
	Title             string
	Status            string
	Owner             string
	ContentHash       string
	Drift             bool   // True if content has changed since recorded
	CurrentHash       string // Current hash if drift detected
	Created           time.Time
	Updated           time.Time
	ApprovedBy        string
	ApprovedAt        *time.Time
	Supersedes        string
	SupersededBy      string
	QualityEvaluation *model.QualityEvaluation
	// EntityTransition is set when this operation triggered a lifecycle
	// transition on the owning entity. Nil when no transition occurred.
	// Read by 2.0 tools (Track I) to push status_transition side effects.
	EntityTransition *DocEntityTransition
	// Warnings contains non-blocking advisory messages (e.g. missing sections
	// on registration). Empty when there are no warnings.
	Warnings []string
}

// DocumentFilters contains optional filters for listing documents.
type DocumentFilters struct {
	Type   string
	Status string
	Owner  string
}

// RefreshDocumentInput contains the parameters for refreshing a document record's content hash.
type RefreshDocumentInput struct {
	ID string
}

// RefreshDocumentResult contains the result of a document refresh operation.
type RefreshDocumentResult struct {
	ID               string
	OldHash          string
	NewHash          string
	Status           string
	Changed          bool
	StatusTransition string // populated when an approved document is reset to draft
	Message          string // human-readable explanation of the status transition
}

// DocumentService handles document record operations.
type DocumentService struct {
	stateRoot       string
	repoRoot        string
	store           *storage.DocumentStore
	now             func() time.Time
	configProvider  func() *config.Config // optional; defaults to config.LoadOrDefault
	entityHook      EntityLifecycleHook   // optional, for lifecycle transitions
	intelligenceSvc *IntelligenceService  // optional, for auto-ingest on submit
	// sectionProvider returns the required section names for a given document
	// type string. Returns nil when no required sections are declared (FR-D06).
	sectionProvider func(docType string) []string
	// testHookAfterApprovalWrite, if non-nil, is called in ApproveDocument
	// immediately after the first store.Write (approval record). For tests only.
	testHookAfterApprovalWrite func()
}

// RepoRoot returns the repository root path used by this service.
func (s *DocumentService) RepoRoot() string {
	return s.repoRoot
}

// StateRoot returns the state root path used by this service.
func (s *DocumentService) StateRoot() string {
	return s.stateRoot
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

// SetSectionProvider attaches a function that returns required section names
// for a given document type string. Called during section validation on
// register (warnings) and approve (hard error). FR-D07.
func (s *DocumentService) SetSectionProvider(fn func(docType string) []string) {
	s.sectionProvider = fn
}

// SetConfigProvider overrides the configuration loader used by the service.
// Intended for testing; production code should leave this at the default.
func (s *DocumentService) SetConfigProvider(fn func() *config.Config) {
	s.configProvider = fn
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

	// Normalise legacy type synonyms (specification->spec, retrospective->retro).
	docType := model.NormaliseDocumentType(model.DocumentType(strings.TrimSpace(input.Type)))
	// Validate the normalised type -- only 8 user-facing types plus policy and rca are accepted.
	if !model.ValidDocumentTypeForRegistration(string(docType)) {
		return DocumentResult{}, fmt.Errorf(
			"invalid document type %q: accepted types are design, spec, dev-plan, review, report, research, retro, proposal",
			input.Type,
		)
	}

	// Validate filename and folder placement (REQ-005 through REQ-010).
	if err := validateDocumentFilename(strings.TrimSpace(input.Path)); err != nil {
		return DocumentResult{}, err
	}
	if err := validateDocumentFolder(strings.TrimSpace(input.Path)); err != nil {
		return DocumentResult{}, err
	}

	// Resolve document path
	docPath := strings.TrimSpace(input.Path)
	fullPath := filepath.Join(s.repoRoot, docPath)

	// Verify the file exists
	if _, err := os.Stat(fullPath); err != nil {
		if os.IsNotExist(err) {
			return DocumentResult{}, fmt.Errorf("document file not found at %q — ensure the file exists at that path before registering it", docPath)
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
		return DocumentResult{}, fmt.Errorf("document %q is already registered. Use doc_record_get to view the existing record", docID)
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

	// Section validation (FR-D04): validate required sections and return
	// any missing sections as warnings. Registration proceeds regardless.
	if s.sectionProvider != nil {
		requiredSections := s.sectionProvider(string(docType))
		if len(requiredSections) > 0 {
			validation, valErr := ValidateSections(fullPath, requiredSections)
			if valErr == nil && len(validation.Missing) > 0 {
				for _, ms := range validation.Missing {
					result.Warnings = append(result.Warnings, fmt.Sprintf("missing required section: %q", ms))
				}
			}
		}
	}

	// Auto-approve: if requested and type is whitelisted, approve immediately (FR-B02–FR-B05).
	if input.AutoApprove {
		autoApproveWhitelist := map[model.DocumentType]bool{
			model.DocumentTypeDevPlan:  true,
			model.DocumentTypeResearch: true,
			model.DocumentTypeReport:   true,
		}
		if !autoApproveWhitelist[docType] {
			// Clean up the just-written draft record so we don't leave orphans.
			_ = s.store.Delete(doc.ID)
			return DocumentResult{}, fmt.Errorf("auto_approve is not permitted for %s documents", string(docType))
		}
		approvedNow := s.now()
		doc.Status = model.DocumentStatusApproved
		doc.ApprovedBy = strings.TrimSpace(input.CreatedBy)
		doc.ApprovedAt = &approvedNow
		doc.Updated = approvedNow
		approvedRecord := storage.DocumentToRecord(doc, "")
		approvedRecordPath, approveErr := s.store.Write(approvedRecord)
		if approveErr != nil {
			return DocumentResult{}, fmt.Errorf("write approved document record: %w", approveErr)
		}
		result.Status = string(doc.Status)
		result.ApprovedBy = doc.ApprovedBy
		result.ApprovedAt = doc.ApprovedAt
		result.RecordPath = approvedRecordPath
	}

	// Trigger entity lifecycle hooks on submission
	if s.entityHook != nil && owner != "" {
		// Set document reference on the owning entity
		var docField string
		switch docType {
		case model.DocumentTypeDesign:
			docField = "design"
		case model.DocumentTypeSpec:
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
		case model.DocumentTypeSpec:
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
		return DocumentResult{}, fmt.Errorf("cannot approve a document with status %q — only draft documents can be approved. If the document was previously approved, use doc_record_get to check its current status", doc.Status)
	}

	// Verify file exists and auto-refresh hash if it has changed (FR-B06, FR-B07).
	fullPath := filepath.Join(s.repoRoot, doc.Path)
	currentHash, err := storage.ComputeContentHash(fullPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return DocumentResult{}, fmt.Errorf("document file not found at %q — the file must exist on disk to be approved", doc.Path)
		}
		return DocumentResult{}, fmt.Errorf("compute content hash: %w", err)
	}

	// If the hash has changed since registration, update it before approving.
	// This eliminates the need for a separate doc refresh step (FR-B06).
	if currentHash != doc.ContentHash {
		doc.ContentHash = currentHash
	}

	// Quality evaluation gate: check if RequireForApproval is enabled.
	var cfg *config.Config
	if s.configProvider != nil {
		cfg = s.configProvider()
	} else {
		cfg = config.LoadOrDefault()
	}
	if cfg.QualityEvaluation.RequireForApproval {
		threshold := cfg.QualityEvaluation.Threshold
		if threshold == 0 {
			threshold = 0.7
		}
		if doc.QualityEvaluation == nil {
			return DocumentResult{}, fmt.Errorf(
				"cannot approve document %s: quality evaluation required but no quality evaluation found.\n\nTo resolve:\n1. Run the quality evaluation skill on the document.\n2. Attach the result: doc(action: \"evaluate\", id: \"%s\", evaluation: {...})\n3. Retry approval: doc(action: \"approve\", id: \"%s\")",
				input.ID, input.ID, input.ID)
		}
		if !doc.QualityEvaluation.Pass {
			var sb strings.Builder
			fmt.Fprintf(&sb, "cannot approve document %s: quality evaluation did not pass (pass=false, overall_score=%g).\n\nDimension scores:\n", input.ID, doc.QualityEvaluation.OverallScore)
			dimKeys := make([]string, 0, len(doc.QualityEvaluation.Dimensions))
			for k := range doc.QualityEvaluation.Dimensions {
				dimKeys = append(dimKeys, k)
			}
			sort.Strings(dimKeys)
			for _, k := range dimKeys {
				fmt.Fprintf(&sb, "  - %s: %g\n", k, doc.QualityEvaluation.Dimensions[k])
			}
			fmt.Fprintf(&sb, "\nTo improve scores, re-evaluate and re-attach: doc(action: \"evaluate\", id: \"%s\", evaluation: {...})", input.ID)
			return DocumentResult{}, fmt.Errorf("%s", sb.String())
		}
		if doc.QualityEvaluation.OverallScore < threshold {
			return DocumentResult{}, fmt.Errorf(
				"cannot approve document %s: quality score %g is below threshold %g.\n\nTo resolve, re-evaluate and re-attach: doc(action: \"evaluate\", id: \"%s\", evaluation: {...})",
				input.ID, doc.QualityEvaluation.OverallScore, threshold, input.ID)
		}
	}

	// Section validation gate (FR-D05): reject approval if required sections
	// are missing. Must run before status update so we don't write a partially
	// approved record. FR-D06: types with no declared sections always pass.
	if s.sectionProvider != nil {
		requiredSections := s.sectionProvider(string(doc.Type))
		if len(requiredSections) > 0 {
			validation, valErr := ValidateSections(fullPath, requiredSections)
			if valErr != nil {
				return DocumentResult{}, fmt.Errorf("validate document sections: %w", valErr)
			}
			if !validation.Valid {
				return DocumentResult{}, fmt.Errorf(
					"cannot approve document %s: missing required sections: %s",
					input.ID, strings.Join(validation.Missing, ", "),
				)
			}
		}
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

	// Invoke test hook if set (for AC-011 test only; no-op in production).
	if s.testHookAfterApprovalWrite != nil {
		s.testHookAfterApprovalWrite()
	}

	// Patch the Status field in the source file (best-effort; FR-006).
	patchOK, patchErr := fsutil.PatchStatusField(fullPath, "approved")
	if patchErr != nil {
		log.Printf("[doc] WARNING: could not patch status field in %s: %v", fullPath, patchErr)
	} else if patchOK {
		// Re-compute and store the updated content hash.
		newHash, hashErr := storage.ComputeContentHash(fullPath)
		if hashErr != nil {
			log.Printf("[doc] WARNING: could not compute content hash after status patch in %s: %v", fullPath, hashErr)
		} else {
			doc.ContentHash = newHash
			hashRecord := storage.DocumentToRecord(doc, "") // no optimistic locking for hash-refresh write
			if _, writeErr := s.store.Write(hashRecord); writeErr != nil {
				log.Printf("[doc] WARNING: could not update content hash record after status patch in %s: %v", fullPath, writeErr)
			}
		}
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
			case entityType == "feature" && doc.Type == model.DocumentTypeSpec:
				targetStatus = "dev-planning"
				// FR-C01: dev-plan approval does NOT cascade the feature to developing.
				// The transition from dev-planning → developing requires an explicit
				// entity(action: transition, status: "developing") call.
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
		return DocumentResult{}, fmt.Errorf("cannot supersede a document with status %q — only approved documents can be superseded. Approve the document first using doc_record_approve", doc.Status)
	}

	// Verify the superseding document exists
	if !s.store.Exists(input.SupersededBy) {
		return DocumentResult{}, fmt.Errorf("superseding document %q not found. Register the new document with doc_record_submit before superseding the old one", input.SupersededBy)
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
			case model.DocumentTypeSpec:
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
		ID:                doc.ID,
		Path:              doc.Path,
		RecordPath:        s.store.GetFilePath(id),
		Type:              string(doc.Type),
		Title:             doc.Title,
		Status:            string(doc.Status),
		Owner:             doc.Owner,
		ContentHash:       doc.ContentHash,
		Created:           doc.Created,
		Updated:           doc.Updated,
		ApprovedBy:        doc.ApprovedBy,
		ApprovedAt:        doc.ApprovedAt,
		Supersedes:        doc.Supersedes,
		SupersededBy:      doc.SupersededBy,
		QualityEvaluation: doc.QualityEvaluation,
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

// LookupByPath finds a document record by its repository-relative file path.
// The path is normalised (leading "./" stripped) before lookup. If no record
// exists for the path, the returned DocumentResult has an empty ID and no error
// is returned — callers should check result.ID == "" to detect the unregistered
// case.
func (s *DocumentService) LookupByPath(ctx context.Context, path string) (DocumentResult, error) {
	// Normalise: strip leading "./" if present.
	path = strings.TrimPrefix(path, "./")

	// List all records and find by exact path match.
	records, err := s.store.List()
	if err != nil {
		return DocumentResult{}, fmt.Errorf("lookup document by path: %w", err)
	}

	for _, record := range records {
		doc := storage.RecordToDocument(record)
		if doc.Path == path {
			return DocumentResult{
				ID:                doc.ID,
				Path:              doc.Path,
				RecordPath:        s.store.GetFilePath(doc.ID),
				Type:              string(doc.Type),
				Title:             doc.Title,
				Status:            string(doc.Status),
				Owner:             doc.Owner,
				ContentHash:       doc.ContentHash,
				Created:           doc.Created,
				Updated:           doc.Updated,
				ApprovedBy:        doc.ApprovedBy,
				ApprovedAt:        doc.ApprovedAt,
				Supersedes:        doc.Supersedes,
				SupersededBy:      doc.SupersededBy,
				QualityEvaluation: doc.QualityEvaluation,
			}, nil
		}
	}

	// No record found — not an error; empty ID signals unregistered.
	return DocumentResult{}, nil
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
// RefreshDocument updates the stored content hash to match the current file on disk.
// If the document was approved and the content has changed, the status is reset to draft.
func (s *DocumentService) RefreshDocument(input RefreshDocumentInput) (RefreshDocumentResult, error) {
	if strings.TrimSpace(input.ID) == "" {
		return RefreshDocumentResult{}, fmt.Errorf("document ID is required")
	}

	record, err := s.store.Load(input.ID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return RefreshDocumentResult{}, fmt.Errorf("no document record found with ID %q. Use doc_record_list to see available records", input.ID)
		}
		return RefreshDocumentResult{}, fmt.Errorf("load document record: %w", err)
	}

	doc := storage.RecordToDocument(record)

	if doc.Status == model.DocumentStatusSuperseded {
		return RefreshDocumentResult{}, fmt.Errorf("document %q has been superseded and cannot be refreshed. Update the superseding document instead", input.ID)
	}

	fullPath := filepath.Join(s.repoRoot, doc.Path)
	if _, statErr := os.Stat(fullPath); statErr != nil {
		if os.IsNotExist(statErr) {
			return RefreshDocumentResult{}, fmt.Errorf("the file %q no longer exists. If the file was moved, delete this record and re-register the document at its new path", doc.Path)
		}
		return RefreshDocumentResult{}, fmt.Errorf("cannot access %q: check that the current user has read access to this file", doc.Path)
	}

	currentHash, err := storage.ComputeContentHash(fullPath)
	if err != nil {
		if errors.Is(err, os.ErrPermission) {
			return RefreshDocumentResult{}, fmt.Errorf("cannot read %q: permission denied. Check that the current user has read access to this file", doc.Path)
		}
		return RefreshDocumentResult{}, fmt.Errorf("compute content hash for %q: %w", doc.Path, err)
	}

	oldHash := doc.ContentHash

	if currentHash == oldHash {
		return RefreshDocumentResult{
			ID:      doc.ID,
			OldHash: oldHash,
			NewHash: currentHash,
			Status:  string(doc.Status),
			Changed: false,
		}, nil
	}

	// Content has changed — update the hash and (if approved) reset to draft.
	doc.ContentHash = currentHash
	doc.Updated = s.now()

	result := RefreshDocumentResult{
		ID:      doc.ID,
		OldHash: oldHash,
		NewHash: currentHash,
		Changed: true,
	}

	if doc.Status == model.DocumentStatusApproved {
		doc.Status = model.DocumentStatusDraft
		result.StatusTransition = "approved → draft"
		result.Message = "Document content has changed; status reset to draft for re-review. Use doc_record_approve to re-approve after reviewing the changes."
	}

	result.Status = string(doc.Status)

	updatedRecord := storage.DocumentToRecord(doc, record.FileHash)
	if _, err := s.store.Write(updatedRecord); err != nil {
		return RefreshDocumentResult{}, fmt.Errorf("write document record: %w", err)
	}

	return result, nil
}

func isValidEntityID(id string) bool {
	if id == "" {
		return false
	}

	// Check for Plan ID format
	if model.IsPlanID(id) {
		return true
	}

	// Check for standard entity ID formats (FEAT-xxx, TASK-xxx, etc.)
	prefixes := []string{"FEAT-", "TASK-", "BUG-", "DEC-"}
	for _, prefix := range prefixes {
		if strings.HasPrefix(id, prefix) {
			return true
		}
	}

	return false
}

// AttachEvaluationInput contains the parameters for attaching a quality evaluation.
type AttachEvaluationInput struct {
	ID         string
	Evaluation model.QualityEvaluation
}

// AttachQualityEvaluation validates and attaches a quality evaluation to a document record.
// Works on both draft and approved documents. Replaces any existing evaluation.
func (s *DocumentService) AttachQualityEvaluation(input AttachEvaluationInput) (DocumentResult, error) {
	if err := validateRequired(field("id", input.ID)); err != nil {
		return DocumentResult{}, err
	}

	eval := input.Evaluation
	if eval.Evaluator == "" {
		return DocumentResult{}, fmt.Errorf("evaluator is required")
	}
	if len(eval.Dimensions) == 0 {
		return DocumentResult{}, fmt.Errorf("dimensions must not be empty")
	}
	if eval.EvaluatedAt.IsZero() {
		return DocumentResult{}, fmt.Errorf("evaluated_at is required")
	}
	if eval.OverallScore < 0.0 || eval.OverallScore > 1.0 {
		return DocumentResult{}, fmt.Errorf("overall_score must be in [0.0, 1.0], got %g", eval.OverallScore)
	}
	for dim, score := range eval.Dimensions {
		if score < 0.0 || score > 1.0 {
			return DocumentResult{}, fmt.Errorf("dimension %q score must be in [0.0, 1.0], got %g", dim, score)
		}
	}

	record, err := s.store.Load(input.ID)
	if err != nil {
		return DocumentResult{}, err
	}

	doc := storage.RecordToDocument(record)
	if doc.Status == model.DocumentStatusSuperseded {
		return DocumentResult{}, fmt.Errorf("cannot attach evaluation to superseded document %q", input.ID)
	}

	now := s.now()
	doc.QualityEvaluation = &eval
	doc.Updated = now

	updatedRecord := storage.DocumentToRecord(doc, record.FileHash)
	recordPath, err := s.store.Write(updatedRecord)
	if err != nil {
		return DocumentResult{}, fmt.Errorf("write document record: %w", err)
	}

	return DocumentResult{
		ID:                record.ID,
		Path:              doc.Path,
		RecordPath:        recordPath,
		Type:              string(doc.Type),
		Title:             doc.Title,
		Status:            string(doc.Status),
		Owner:             doc.Owner,
		ContentHash:       doc.ContentHash,
		Created:           doc.Created,
		Updated:           doc.Updated,
		ApprovedBy:        doc.ApprovedBy,
		ApprovedAt:        doc.ApprovedAt,
		Supersedes:        doc.Supersedes,
		SupersededBy:      doc.SupersededBy,
		QualityEvaluation: doc.QualityEvaluation,
	}, nil
}

// MoveDocument moves a document file to a new path, updates the record, and
// recomputes the content hash. The document ID, approval status, owner, and
// cross-references are preserved (FR-B09 through FR-B15).
func (s *DocumentService) MoveDocument(input MoveDocumentInput) (DocumentResult, error) {
	if err := validateRequired(
		field("id", input.ID),
		field("new_path", input.NewPath),
	); err != nil {
		return DocumentResult{}, err
	}

	record, err := s.store.Load(input.ID)
	if err != nil {
		return DocumentResult{}, err
	}

	doc := storage.RecordToDocument(record)
	oldPath := doc.Path

	oldFullPath := filepath.Join(s.repoRoot, oldPath)
	newFullPath := filepath.Join(s.repoRoot, input.NewPath)

	// Verify source file exists (FR-B14).
	if _, statErr := os.Stat(oldFullPath); statErr != nil {
		if os.IsNotExist(statErr) {
			return DocumentResult{}, fmt.Errorf("document file not found at %q — cannot move a file that does not exist", oldPath)
		}
		return DocumentResult{}, fmt.Errorf("stat source file: %w", statErr)
	}

	// Ensure destination directory exists.
	if mkdirErr := os.MkdirAll(filepath.Dir(newFullPath), 0o755); mkdirErr != nil {
		return DocumentResult{}, fmt.Errorf("create destination directory: %w", mkdirErr)
	}

	// Move the file on disk.
	if renameErr := os.Rename(oldFullPath, newFullPath); renameErr != nil {
		return DocumentResult{}, fmt.Errorf("move document file: %w", renameErr)
	}

	// Update path and recompute hash.
	doc.Path = input.NewPath
	newHash, hashErr := storage.ComputeContentHash(newFullPath)
	if hashErr != nil {
		// Roll back the rename on hash failure.
		_ = os.Rename(newFullPath, oldFullPath)
		return DocumentResult{}, fmt.Errorf("compute content hash after move: %w", hashErr)
	}
	doc.ContentHash = newHash

	// Update type if the new path implies a different document type (FR-B11).
	if inferredType := inferDocTypeFromPath(input.NewPath); inferredType != "" && inferredType != string(doc.Type) {
		doc.Type = model.DocumentType(inferredType)
	}

	doc.Updated = s.now()

	updatedRecord := storage.DocumentToRecord(doc, record.FileHash)
	recordPath, writeErr := s.store.Write(updatedRecord)
	if writeErr != nil {
		return DocumentResult{}, fmt.Errorf("write document record: %w", writeErr)
	}

	return DocumentResult{
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
		// OldPath is returned so callers can pass both old and new paths to
		// CommitStateAndPaths for the atomic commit (FR-B13).
	}, nil
}

// UpdateDocumentPathAndOwner updates the path and/or owner of an existing
// document record without touching the file on disk. It recomputes the
// content hash from the new path location. Called by kbz move after git mv.
func (s *DocumentService) UpdateDocumentPathAndOwner(id, newPath, newOwner string) (DocumentResult, error) {
	if id == "" {
		return DocumentResult{}, fmt.Errorf("id is required")
	}

	record, err := s.store.Load(id)
	if err != nil {
		return DocumentResult{}, err
	}

	doc := storage.RecordToDocument(record)

	if newPath != "" {
		doc.Path = newPath
		fullPath := filepath.Join(s.repoRoot, newPath)
		if newHash, hashErr := storage.ComputeContentHash(fullPath); hashErr == nil {
			doc.ContentHash = newHash
		}
	}
	if newOwner != "" {
		doc.Owner = newOwner
	}
	doc.Updated = s.now()

	updatedRecord := storage.DocumentToRecord(doc, "")
	recordPath, writeErr := s.store.Write(updatedRecord)
	if writeErr != nil {
		return DocumentResult{}, fmt.Errorf("write document record: %w", writeErr)
	}

	result := documentToResult(doc)
	result.RecordPath = recordPath
	return result, nil
}

// DeleteDocument removes a document file, clears entity references, and
// removes the state and index records (FR-B16 through FR-B21).
func (s *DocumentService) DeleteDocument(input DeleteDocumentInput) (DocumentResult, error) {
	if err := validateRequired(field("id", input.ID)); err != nil {
		return DocumentResult{}, err
	}

	record, err := s.store.Load(input.ID)
	if err != nil {
		return DocumentResult{}, err
	}

	doc := storage.RecordToDocument(record)

	// Guard approved documents unless force is set (FR-B17).
	if doc.Status == model.DocumentStatusApproved && !input.Force {
		return DocumentResult{}, fmt.Errorf(
			"cannot delete approved document %q without force: set force: true to confirm deletion of an approved document",
			input.ID,
		)
	}

	// Remove the file from disk; ignore "not exists" (FR-B20).
	fullPath := filepath.Join(s.repoRoot, doc.Path)
	if removeErr := os.Remove(fullPath); removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
		return DocumentResult{}, fmt.Errorf("remove document file: %w", removeErr)
	}

	// Clear the entity's document reference field (FR-B18 item 2).
	if s.entityHook != nil && doc.Owner != "" {
		var docField string
		switch doc.Type {
		case model.DocumentTypeDesign:
			docField = "design"
		case model.DocumentTypeSpec:
			docField = "spec"
		case model.DocumentTypeDevPlan:
			docField = "dev_plan"
		}
		if docField != "" {
			_ = s.entityHook.SetDocumentRef(doc.Owner, docField, "")
		}
	}

	// Remove the state record (FR-B18 item 3).
	if deleteErr := s.store.Delete(input.ID); deleteErr != nil {
		return DocumentResult{}, fmt.Errorf("delete document record: %w", deleteErr)
	}

	// Remove any corresponding index file best-effort (FR-B18 item 4).
	// Index files live at .kbz/index/documents/<safe-id>.yaml
	instanceRoot := filepath.Dir(s.stateRoot)
	safeID := strings.ReplaceAll(input.ID, "/", "--")
	indexFilePath := filepath.Join(instanceRoot, "index", "documents", safeID+".yaml")
	_ = os.Remove(indexFilePath) // best-effort; index may not exist

	return DocumentResult{
		ID:    doc.ID,
		Path:  doc.Path,
		Type:  string(doc.Type),
		Title: doc.Title,
		Owner: doc.Owner,
	}, nil
}

// inferDocTypeFromPath infers the document type from the path's directory
// component. Returns an empty string if no type can be inferred, in which
// case the caller should keep the existing type.
// documentToResult converts a model.DocumentRecord to a DocumentResult.
// The RecordPath field is not set here; callers must set it after the fact.
func documentToResult(doc model.DocumentRecord) DocumentResult {
	return DocumentResult{
		ID:           doc.ID,
		Path:         doc.Path,
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
}

func inferDocTypeFromPath(path string) string {
	lower := strings.ToLower(filepath.ToSlash(path))
	switch {
	case strings.Contains(lower, "/design/"):
		return string(model.DocumentTypeDesign)
	case strings.Contains(lower, "/spec/"):
		return string(model.DocumentTypeSpec)
	case strings.Contains(lower, "/plan/"):
		return string(model.DocumentTypeDevPlan)
	case strings.Contains(lower, "/research/"):
		return string(model.DocumentTypeResearch)
	case strings.Contains(lower, "/reports/"), strings.Contains(lower, "/reviews/"):
		return string(model.DocumentTypeReport)
	case strings.Contains(lower, "/policy/"):
		return string(model.DocumentTypePolicy)
	default:
		return ""
	}
}
