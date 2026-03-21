package document

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"kanbanzai/internal/core"
	"kanbanzai/internal/id"
	"kanbanzai/internal/model"
)

// DocsDir is the document storage directory within the instance root.
const DocsDir = "docs"

// DocsPath returns the document storage directory path.
func DocsPath() string {
	return filepath.Join(core.InstanceRootDir, DocsDir)
}

// SubmitInput is the input for submitting a new document.
type SubmitInput struct {
	Type      DocType
	Title     string
	Feature   string // optional, required for design/spec/plan
	Body      string
	CreatedBy string
}

// ApproveInput is the input for approving a document.
type ApproveInput struct {
	Type       DocType
	ID         string
	ApprovedBy string
}

// DocumentResult is the result of a document operation.
type DocumentResult struct {
	ID     string
	Type   DocType
	Title  string
	Status DocStatus
	Path   string
}

// DocService provides document lifecycle operations.
type DocService struct {
	store  *DocStore
	now    func() time.Time
	nextID func() string
}

// NewDocService creates a new DocService using the given root path.
// If root is empty, the default docs path is used.
func NewDocService(root string) *DocService {
	if strings.TrimSpace(root) == "" {
		root = DocsPath()
	}

	store := NewDocStore(root)
	allocator := id.NewAllocator()

	return &DocService{
		store: store,
		now: func() time.Time {
			return time.Now().UTC()
		},
		nextID: func() string {
			idValue, err := allocator.Allocate(model.EntityKindDocument, "", nil)
			if err != nil {
				// Fallback — should not happen in practice
				return fmt.Sprintf("DOC-ERR-%d", time.Now().UnixNano())
			}
			return idValue
		},
	}
}

// ScaffoldDocument generates a starter document from a template.
// Returns the scaffolded markdown content (not yet stored).
func (s *DocService) ScaffoldDocument(docType DocType, title string) (string, error) {
	return Scaffold(docType, title)
}

// Submit creates and stores a new document in submitted state.
func (s *DocService) Submit(input SubmitInput) (DocumentResult, error) {
	if !ValidDocType(string(input.Type)) {
		return DocumentResult{}, fmt.Errorf("invalid document type: %s", input.Type)
	}
	if strings.TrimSpace(input.Title) == "" {
		return DocumentResult{}, errors.New("document title is required")
	}
	if strings.TrimSpace(input.Body) == "" {
		return DocumentResult{}, errors.New("document body is required")
	}
	if strings.TrimSpace(input.CreatedBy) == "" {
		return DocumentResult{}, errors.New("created_by is required")
	}
	if requiresFeatureRef(input.Type) && strings.TrimSpace(input.Feature) == "" {
		return DocumentResult{}, fmt.Errorf("feature reference required for %s documents", input.Type)
	}

	now := s.now()
	doc := Document{
		Meta: DocMeta{
			ID:        s.nextID(),
			Type:      input.Type,
			Title:     input.Title,
			Status:    DocStatusSubmitted,
			Feature:   input.Feature,
			CreatedBy: input.CreatedBy,
			Created:   now,
			Updated:   now,
		},
		Body: input.Body,
	}

	path, err := s.store.Write(doc)
	if err != nil {
		return DocumentResult{}, fmt.Errorf("write document: %w", err)
	}

	return DocumentResult{
		ID:     doc.Meta.ID,
		Type:   doc.Meta.Type,
		Title:  doc.Meta.Title,
		Status: doc.Meta.Status,
		Path:   path,
	}, nil
}

// UpdateBody updates the body of a document and transitions it to normalised state.
// This represents the normalisation step where an agent cleans/restructures the content.
func (s *DocService) UpdateBody(docType DocType, id, newBody string) (DocumentResult, error) {
	doc, path, err := s.findDocument(docType, id)
	if err != nil {
		return DocumentResult{}, err
	}

	if doc.Meta.Status != DocStatusSubmitted {
		return DocumentResult{}, fmt.Errorf("cannot normalise document in %s state (must be submitted)", doc.Meta.Status)
	}

	doc.Meta.Status = DocStatusNormalised
	doc.Meta.Updated = s.now()
	doc.Body = newBody

	newPath, err := s.store.Write(doc)
	if err != nil {
		return DocumentResult{}, fmt.Errorf("write normalised document: %w", err)
	}

	_ = removeIfDifferent(path, newPath)

	return DocumentResult{
		ID:     doc.Meta.ID,
		Type:   doc.Meta.Type,
		Title:  doc.Meta.Title,
		Status: doc.Meta.Status,
		Path:   newPath,
	}, nil
}

// Approve transitions a document to approved state.
// Only normalised documents can be approved.
func (s *DocService) Approve(input ApproveInput) (DocumentResult, error) {
	if strings.TrimSpace(input.ApprovedBy) == "" {
		return DocumentResult{}, errors.New("approved_by is required")
	}

	doc, path, err := s.findDocument(input.Type, input.ID)
	if err != nil {
		return DocumentResult{}, err
	}

	if !ValidDocTransition(doc.Meta.Status, DocStatusApproved) {
		return DocumentResult{}, fmt.Errorf("cannot approve document in %s state (must be normalised)", doc.Meta.Status)
	}

	now := s.now()
	doc.Meta.Status = DocStatusApproved
	doc.Meta.ApprovedBy = input.ApprovedBy
	doc.Meta.ApprovedAt = &now
	doc.Meta.Updated = now

	newPath, err := s.store.Write(doc)
	if err != nil {
		return DocumentResult{}, fmt.Errorf("write approved document: %w", err)
	}

	_ = removeIfDifferent(path, newPath)

	return DocumentResult{
		ID:     doc.Meta.ID,
		Type:   doc.Meta.Type,
		Title:  doc.Meta.Title,
		Status: doc.Meta.Status,
		Path:   newPath,
	}, nil
}

// Retrieve returns a document by type and ID.
func (s *DocService) Retrieve(docType DocType, id string) (Document, error) {
	doc, _, err := s.findDocument(docType, id)
	if err != nil {
		return Document{}, err
	}
	return doc, nil
}

// ExtractFromDocument returns approved document metadata and body for agent parsing.
func (s *DocService) ExtractFromDocument(id string) (ExtractedDocument, error) {
	doc, err := s.findDocumentByID(id)
	if err != nil {
		return ExtractedDocument{}, err
	}
	if doc.Meta.Status != DocStatusApproved {
		return ExtractedDocument{}, fmt.Errorf(
			"cannot extract from document in %s state (must be approved)",
			doc.Meta.Status,
		)
	}

	return ExtractedDocument{
		Meta: ExtractedDocumentMeta{
			ID:         doc.Meta.ID,
			Type:       doc.Meta.Type,
			Title:      doc.Meta.Title,
			Status:     doc.Meta.Status,
			Feature:    doc.Meta.Feature,
			CreatedBy:  doc.Meta.CreatedBy,
			Created:    doc.Meta.Created,
			Updated:    doc.Meta.Updated,
			ApprovedBy: doc.Meta.ApprovedBy,
			ApprovedAt: doc.Meta.ApprovedAt,
		},
		Body: doc.Body,
	}, nil
}

// Validate validates a document's structure against its template.
// Returns validation errors (empty if valid).
func (s *DocService) Validate(doc Document) []ValidationError {
	return ValidateDocumentWithRefs(doc, func(entityType, id string) bool {
		if strings.TrimSpace(entityType) != "feature" || strings.TrimSpace(id) == "" {
			return false
		}

		docPaths, err := s.store.ListAll()
		if err != nil {
			return false
		}
		for _, path := range docPaths {
			loadedDoc, loadErr := s.store.LoadByPath(path)
			if loadErr != nil {
				continue
			}
			if loadedDoc.Meta.Feature == id {
				return true
			}
		}

		return false
	})
}

// ListByType lists all documents of a given type.
func (s *DocService) ListByType(docType DocType) ([]DocumentResult, error) {
	paths, err := s.store.List(docType)
	if err != nil {
		return nil, err
	}

	var results []DocumentResult
	for _, path := range paths {
		doc, err := s.store.LoadByPath(path)
		if err != nil {
			continue // skip unreadable files
		}
		results = append(results, DocumentResult{
			ID:     doc.Meta.ID,
			Type:   doc.Meta.Type,
			Title:  doc.Meta.Title,
			Status: doc.Meta.Status,
			Path:   path,
		})
	}
	return results, nil
}

// ListAll lists all documents across all types.
func (s *DocService) ListAll() ([]DocumentResult, error) {
	paths, err := s.store.ListAll()
	if err != nil {
		return nil, err
	}

	var results []DocumentResult
	for _, path := range paths {
		doc, err := s.store.LoadByPath(path)
		if err != nil {
			continue
		}
		results = append(results, DocumentResult{
			ID:     doc.Meta.ID,
			Type:   doc.Meta.Type,
			Title:  doc.Meta.Title,
			Status: doc.Meta.Status,
			Path:   path,
		})
	}
	return results, nil
}

// findDocument locates a document by type and ID.
func (s *DocService) findDocument(docType DocType, id string) (Document, string, error) {
	paths, err := s.store.List(docType)
	if err != nil {
		return Document{}, "", err
	}

	prefix := id + "-"
	for _, path := range paths {
		base := filepath.Base(path)
		if strings.HasPrefix(base, prefix) {
			doc, err := s.store.LoadByPath(path)
			if err != nil {
				return Document{}, "", err
			}
			return doc, path, nil
		}
	}

	return Document{}, "", fmt.Errorf("document not found: %s/%s", docType, id)
}

// findDocumentByID locates a document by ID across all document types.
func (s *DocService) findDocumentByID(id string) (Document, error) {
	paths, err := s.store.ListAll()
	if err != nil {
		return Document{}, err
	}

	prefix := id + "-"
	for _, path := range paths {
		base := filepath.Base(path)
		if !strings.HasPrefix(base, prefix) {
			continue
		}

		doc, err := s.store.LoadByPath(path)
		if err != nil {
			return Document{}, err
		}
		return doc, nil
	}

	return Document{}, fmt.Errorf("document not found: %s", id)
}

// removeIfDifferent removes the old file if the new path is different.
func removeIfDifferent(oldPath, newPath string) error {
	if oldPath == newPath {
		return nil
	}
	if err := os.Remove(oldPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove old document file: %w", err)
	}
	return nil
}
