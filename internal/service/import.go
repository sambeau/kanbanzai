package service

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/sambeau/kanbanzai/internal/config"
)

// ─── Dry-run types ────────────────────────────────────────────────────────────

// ImportDryRunResult is the result of a dry-run import operation.
// It describes what a live import would register and skip, without making
// any changes to the document store.
type ImportDryRunResult struct {
	WouldImport []DryRunImportEntry `json:"would_import"`
	WouldSkip   []DryRunSkipEntry   `json:"would_skip"`
	Summary     DryRunSummary       `json:"summary"`
}

// DryRunImportEntry describes a file that would be registered by a live import.
type DryRunImportEntry struct {
	// Path is the repository-relative path of the document.
	Path string `json:"path"`
	// Type is the inferred document type.
	Type string `json:"type"`
	// Title is the inferred document title.
	Title string `json:"title"`
	// Owner is the inferred owner (empty string when none).
	Owner string `json:"owner"`
}

// DryRunSkipEntry describes a file that would be skipped by a live import.
type DryRunSkipEntry struct {
	// Path is the repository-relative path of the document.
	Path string `json:"path"`
	// Reason explains why the file would be skipped.
	Reason string `json:"reason"`
}

// DryRunSummary holds aggregate counts for a dry-run import.
type DryRunSummary struct {
	WouldImport int `json:"would_import"`
	WouldSkip   int `json:"would_skip"`
}

// matchGlob reports whether name matches the glob pattern.
// An empty pattern matches everything.
// Supports two matching modes:
//   - If the pattern contains a path separator, it matches against the full relative path
//   - Otherwise, it matches against just the filename (basename)
//
// Note: Go's filepath.Match does not support "**" for recursive matching.
// Use patterns like "design/*.md" or "*.md" instead.
func matchGlob(pattern, name string) bool {
	if pattern == "" {
		return true
	}
	// Normalize to forward slashes for consistent matching
	pattern = filepath.ToSlash(pattern)
	name = filepath.ToSlash(name)

	// If pattern has no path separator, match against basename only
	if !strings.Contains(pattern, "/") {
		name = filepath.Base(name)
	}

	matched, _ := filepath.Match(pattern, name)
	return matched
}

// BatchImportInput contains the parameters for a batch import operation.
type BatchImportInput struct {
	// Path is the directory to scan for documents.
	Path string
	// DefaultType is the fallback document type when no pattern matches.
	DefaultType string
	// Owner is the optional parent Plan or Feature ID for imported documents.
	Owner string
	// CreatedBy is the already-resolved user identity.
	CreatedBy string
	// Glob is an optional glob pattern to filter files.
	// If the pattern contains a path separator (e.g., "design/*.md"), it matches
	// against the relative path from the import directory.
	// If the pattern has no separator (e.g., "*.md"), it matches the filename only.
	// Note: Go's filepath.Match does not support "**" for recursive matching.
	Glob string
}

// BatchImportSkip records a file that was deliberately skipped during import.
type BatchImportSkip struct {
	Path   string
	Reason string
}

// BatchImportError records a file that failed to import.
type BatchImportError struct {
	Path  string
	Error string
}

// BatchImportResult is the outcome of a batch import operation.
type BatchImportResult struct {
	Imported int
	Skipped  []BatchImportSkip
	Errors   []BatchImportError
}

// BatchImportService performs batch document imports using an existing DocumentService.
type BatchImportService struct {
	docSvc *DocumentService
}

// NewBatchImportService creates a new BatchImportService.
func NewBatchImportService(docSvc *DocumentService) *BatchImportService {
	return &BatchImportService{docSvc: docSvc}
}

// Import scans the directory in input.Path, matches .md files, infers document
// types, and creates document records via DocumentService.SubmitDocument.
// Already-imported files are skipped (idempotent). Errors for individual files
// are collected without aborting the batch.
func (s *BatchImportService) Import(cfg *config.Config, input BatchImportInput) (BatchImportResult, error) {
	var result BatchImportResult

	// Verify the scan directory exists before walking.
	if _, err := os.Stat(input.Path); err != nil {
		return result, fmt.Errorf("access import directory: %w", err)
	}

	repoRoot := s.docSvc.RepoRoot()

	// Build a set of already-imported paths for idempotency.
	existing, _ := s.docSvc.ListDocuments(DocumentFilters{})
	existingPaths := make(map[string]bool, len(existing))
	for _, doc := range existing {
		existingPaths[doc.Path] = true
	}

	walkErr := filepath.WalkDir(input.Path, func(absPath string, d fs.DirEntry, err error) error {
		if err != nil {
			result.Errors = append(result.Errors, BatchImportError{
				Path:  absPath,
				Error: err.Error(),
			})
			return nil
		}

		if d.IsDir() {
			return nil
		}

		// Only process Markdown files.
		if !strings.HasSuffix(strings.ToLower(absPath), ".md") {
			return nil
		}

		// Compute path relative to the import directory for glob matching.
		relFromImport, _ := filepath.Rel(input.Path, absPath)
		if relFromImport == "" {
			relFromImport = filepath.Base(absPath)
		}

		// Apply optional glob filter.
		if !matchGlob(input.Glob, relFromImport) {
			return nil
		}

		// Compute a path relative to the repo root so DocumentService can
		// locate the file correctly (it joins repoRoot + relPath internally).
		relPath, relErr := filepath.Rel(repoRoot, absPath)
		if relErr != nil {
			// Fall back to the raw path if Rel fails (e.g. different drives on Windows).
			relPath = absPath
		}

		// Skip already-imported files (compare by relative path).
		if existingPaths[relPath] {
			result.Skipped = append(result.Skipped, BatchImportSkip{
				Path:   relPath,
				Reason: "already imported",
			})
			return nil
		}

		// Infer document type from path conventions.
		docType := inferDocType(cfg, relPath, input.DefaultType)
		if docType == "" {
			result.Skipped = append(result.Skipped, BatchImportSkip{
				Path:   relPath,
				Reason: "no document type available: no matching pattern and no default_type provided",
			})
			return nil
		}

		// Derive a human-readable title from the filename.
		title := deriveTitle(filepath.Base(relPath))

		_, submitErr := s.docSvc.SubmitDocument(SubmitDocumentInput{
			Path:      relPath,
			Type:      docType,
			Title:     title,
			Owner:     input.Owner,
			CreatedBy: input.CreatedBy,
		})
		if submitErr != nil {
			result.Errors = append(result.Errors, BatchImportError{
				Path:  relPath,
				Error: submitErr.Error(),
			})
			return nil
		}

		result.Imported++
		return nil
	})
	if walkErr != nil {
		return result, walkErr
	}

	return result, nil
}

// inferDocType determines the document type for a file path using the configured
// type mappings. It falls back to defaultType when no pattern matches.
func inferDocType(cfg *config.Config, path, defaultType string) string {
	// Normalize path separators and wrap for segment matching.
	normalized := filepath.ToSlash(path)
	wrapped := "/" + normalized + "/"

	for _, mapping := range cfg.Import.TypeMappings {
		segment := extractGlobSegment(mapping.Glob)
		if segment != "" && strings.Contains(wrapped, segment) {
			return mapping.Type
		}
	}

	return defaultType
}

// extractGlobSegment extracts the fixed directory segment from a glob pattern
// like "*/design/*" or "**/spec/**", returning "/design/" or "/spec/".
func extractGlobSegment(glob string) string {
	parts := strings.Split(filepath.ToSlash(glob), "/")
	for _, part := range parts {
		if part != "" && part != "*" && part != "**" {
			return "/" + part + "/"
		}
	}
	return ""
}

// ImportDryRun runs the full import inference pipeline over input.Path without
// writing any records to the document store. It returns a preview of what a
// live import would register and what it would skip.
//
// The inference logic (directory walking, type inference, title extraction,
// owner inference) is identical to that used by Import, ensuring that
// WouldImport matches the set that a live import would actually register
// (REQ-07, REQ-13 of doc-import-dry-run spec).
//
// Files that already have a document record in the store appear in WouldSkip
// with Reason "already registered" (REQ-12).
func (s *BatchImportService) ImportDryRun(cfg *config.Config, input BatchImportInput) (*ImportDryRunResult, error) {
	result := &ImportDryRunResult{
		WouldImport: []DryRunImportEntry{},
		WouldSkip:   []DryRunSkipEntry{},
	}

	// Verify the scan directory exists before walking.
	if _, err := os.Stat(input.Path); err != nil {
		return nil, fmt.Errorf("access import directory: %w", err)
	}

	repoRoot := s.docSvc.RepoRoot()

	// Build a set of already-registered paths for the skip check.
	existing, _ := s.docSvc.ListDocuments(DocumentFilters{})
	existingPaths := make(map[string]bool, len(existing))
	for _, doc := range existing {
		existingPaths[doc.Path] = true
	}

	walkErr := filepath.WalkDir(input.Path, func(absPath string, d fs.DirEntry, err error) error {
		if err != nil {
			// Skip unreadable entries without aborting the walk.
			return nil
		}
		if d.IsDir() {
			return nil
		}
		// Only process Markdown files.
		if !strings.HasSuffix(strings.ToLower(absPath), ".md") {
			return nil
		}

		// Compute path relative to the import directory for glob matching.
		relFromImport, _ := filepath.Rel(input.Path, absPath)
		if relFromImport == "" {
			relFromImport = filepath.Base(absPath)
		}

		// Apply optional glob filter.
		if !matchGlob(input.Glob, relFromImport) {
			return nil
		}

		// Compute a path relative to the repo root.
		relPath, relErr := filepath.Rel(repoRoot, absPath)
		if relErr != nil {
			relPath = absPath
		}

		// Already-registered files go to WouldSkip (REQ-12).
		if existingPaths[relPath] {
			result.WouldSkip = append(result.WouldSkip, DryRunSkipEntry{
				Path:   relPath,
				Reason: "already registered",
			})
			return nil
		}

		// Infer document type — same logic as live import (REQ-07).
		docType := inferDocType(cfg, relPath, input.DefaultType)
		if docType == "" {
			result.WouldSkip = append(result.WouldSkip, DryRunSkipEntry{
				Path:   relPath,
				Reason: "no document type available: no matching pattern and no default_type provided",
			})
			return nil
		}

		// Derive title — same logic as live import (REQ-07).
		title := deriveTitle(filepath.Base(relPath))

		result.WouldImport = append(result.WouldImport, DryRunImportEntry{
			Path:  relPath,
			Type:  docType,
			Title: title,
			Owner: input.Owner,
		})
		return nil
	})
	if walkErr != nil {
		return nil, walkErr
	}

	result.Summary = DryRunSummary{
		WouldImport: len(result.WouldImport),
		WouldSkip:   len(result.WouldSkip),
	}
	return result, nil
}

// deriveTitle creates a human-readable title from a filename by removing the
// extension and replacing dashes/underscores with spaces.
func deriveTitle(filename string) string {
	name := strings.TrimSuffix(filename, filepath.Ext(filename))
	name = strings.NewReplacer("-", " ", "_", " ").Replace(name)
	if name == "" {
		return "Untitled"
	}
	return strings.ToUpper(name[:1]) + name[1:]
}
