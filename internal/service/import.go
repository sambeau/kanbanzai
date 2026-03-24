package service

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"kanbanzai/internal/config"
)

// matchGlob reports whether name matches the glob pattern using filepath.Match.
// An empty pattern matches everything.
func matchGlob(pattern, name string) bool {
	if pattern == "" {
		return true
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
	// Glob is an optional filename glob pattern (e.g. "design-*.md").
	// When non-empty, only files whose base name matches are imported.
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

		// Apply optional glob filter against the base filename.
		if !matchGlob(input.Glob, filepath.Base(absPath)) {
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
