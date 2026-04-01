// Package service doc_audit.go — document audit service for Kanbanzai 2.5.
//
// AuditDocuments walks the configured document directories, compares the
// found .md files against the document store, and returns:
//   - unregistered: files on disk with no matching store record
//   - missing:      store records whose files no longer exist on disk
//   - registered:   files on disk with a matching store record (optional)
//   - summary:      aggregate counts
//
// The audit action is read-only; it does not create, update, or delete any
// store records or files.
//
// See specification work/spec/doc-audit.md.
package service

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/sambeau/kanbanzai/internal/config"
)

// defaultAuditDirs is the hardcoded set of document directories scanned when
// no path parameter is provided (spec §4, Default Document Directories).
var defaultAuditDirs = []string{
	"work/design",
	"work/spec",
	"work/plan",
	"work/research",
	"work/reports",
	"work/reviews",
	"docs",
}

// DocAuditStore is the minimal interface needed by AuditDocuments to query
// the document store. *DocumentService satisfies this interface.
type DocAuditStore interface {
	ListDocuments(filters DocumentFilters) ([]DocumentResult, error)
}

// UnregisteredFile describes a .md file found on disk that has no matching
// document store record.
type UnregisteredFile struct {
	// Path is the repository-relative path of the file.
	Path string `json:"path"`
	// InferredType is the document type inferred from the file's directory path,
	// using the same mapping as doc import. Empty when no mapping matches.
	InferredType string `json:"inferred_type"`
}

// MissingRecord describes a document store record whose file no longer exists
// on disk.
type MissingRecord struct {
	// Path is the repository-relative path stored in the record.
	Path string `json:"path"`
	// DocID is the document record ID.
	DocID string `json:"doc_id"`
}

// RegisteredFile describes a .md file that has a matching document store record.
// Only populated when AuditDocuments is called with includeRegistered=true.
type RegisteredFile struct {
	// Path is the repository-relative path of the file.
	Path string `json:"path"`
	// DocID is the document record ID.
	DocID string `json:"doc_id"`
}

// AuditSummary holds aggregate counts for an audit operation.
type AuditSummary struct {
	// TotalOnDisk is the total number of .md files found during the walk.
	TotalOnDisk int `json:"total_on_disk"`
	// Registered is the number of files with a matching store record.
	Registered int `json:"registered"`
	// Unregistered is the number of files with no matching store record.
	Unregistered int `json:"unregistered"`
	// Missing is the number of store records with no corresponding file.
	Missing int `json:"missing"`
}

// AuditResult is the full output of an AuditDocuments call.
type AuditResult struct {
	// Unregistered lists files found on disk with no matching store record.
	// Always present (may be empty); never nil.
	Unregistered []UnregisteredFile `json:"unregistered"`
	// Missing lists store records whose files no longer exist on disk.
	// Always present (may be empty); never nil.
	Missing []MissingRecord `json:"missing"`
	// Registered lists files with matching store records.
	// Only populated when includeRegistered is true; nil otherwise.
	Registered []RegisteredFile `json:"registered,omitempty"`
	// Summary contains aggregate counts.
	Summary AuditSummary `json:"summary"`
}

// AuditDocuments walks the given directories (or the default set when dirs is
// empty) and compares found .md files against the document store.
//
// repoRoot is the absolute path to the repository root; it is prepended to
// each directory path before walking. Stored document record paths are treated
// as relative to repoRoot.
//
// When dirs is non-empty (explicit path provided by the caller), each
// directory must exist; a non-existent explicit path is returned as an error
// (REQ-02). When dirs is empty (default-directory mode), missing directories
// are silently skipped.
//
// When includeRegistered is true, the Registered field of the returned
// AuditResult is populated with registered files.
//
// The returned AuditResult always satisfies:
//
//	Summary.Registered + Summary.Unregistered == Summary.TotalOnDisk
func AuditDocuments(
	_ context.Context,
	store DocAuditStore,
	repoRoot string,
	dirs []string,
	includeRegistered bool,
) (*AuditResult, error) {
	result := &AuditResult{
		Unregistered: []UnregisteredFile{},
		Missing:      []MissingRecord{},
	}
	if includeRegistered {
		result.Registered = []RegisteredFile{}
	}

	// Determine which directories to scan.
	// Track whether the caller supplied an explicit path so we can enforce
	// REQ-02: an explicit path that does not exist is an error.
	explicitPath := len(dirs) > 0
	scanDirs := dirs
	if !explicitPath {
		scanDirs = defaultAuditDirs
	}

	// Build a set of all document record paths for efficient O(1) lookup.
	allRecords, err := store.ListDocuments(DocumentFilters{})
	if err != nil {
		return nil, err
	}
	// recordByPath maps repository-relative path → document ID.
	recordByPath := make(map[string]string, len(allRecords))
	for _, rec := range allRecords {
		recordByPath[rec.Path] = rec.ID
	}

	// Normalise repoRoot for Rel computations.
	if repoRoot == "" {
		repoRoot = "."
	}

	// Load config for type inference (same mapping as doc import, REQ-10).
	cfg := config.LoadOrDefault()

	// onDiskPaths tracks every relative path we find on disk, for the
	// missing-record check after the walk.
	onDiskPaths := make(map[string]bool)

	// Walk each scan directory, collecting on-disk .md files.
	for _, dir := range scanDirs {
		// Resolve to an absolute walk path. If dir is already absolute (e.g.
		// in tests where the caller passes a temp-dir path), use it directly.
		// Otherwise join with repoRoot so that relative paths like "work/spec"
		// are resolved correctly (REQ-02).
		var absDir string
		if filepath.IsAbs(dir) {
			absDir = dir
		} else {
			absDir = filepath.Join(repoRoot, dir)
		}

		// When the caller supplied an explicit path, a missing directory is an
		// error (REQ-02). In default-directory mode, silently skip directories
		// that do not exist (a fresh project may not have all standard dirs).
		if _, statErr := os.Stat(absDir); os.IsNotExist(statErr) {
			if explicitPath {
				return nil, fmt.Errorf("audit path does not exist: %s", dir)
			}
			continue
		}

		walkErr := filepath.WalkDir(absDir, func(absPath string, d fs.DirEntry, walkEntryErr error) error {
			if walkEntryErr != nil {
				// Skip unreadable entries without aborting the walk.
				return nil
			}
			if d.IsDir() {
				return nil
			}
			// Only .md files (REQ-03).
			if !strings.HasSuffix(strings.ToLower(absPath), ".md") {
				return nil
			}

			// Compute repository-relative path.
			relPath, relErr := filepath.Rel(repoRoot, absPath)
			if relErr != nil {
				relPath = absPath
			}
			relPath = filepath.ToSlash(relPath)

			onDiskPaths[relPath] = true
			result.Summary.TotalOnDisk++

			if docID, registered := recordByPath[relPath]; registered {
				// File is in the store.
				result.Summary.Registered++
				if includeRegistered {
					result.Registered = append(result.Registered, RegisteredFile{
						Path:  relPath,
						DocID: docID,
					})
				}
			} else {
				// File is not in the store — unregistered.
				result.Summary.Unregistered++
				inferred := inferDocType(cfg, relPath, "")
				result.Unregistered = append(result.Unregistered, UnregisteredFile{
					Path:         relPath,
					InferredType: inferred,
				})
			}

			return nil
		})
		if walkErr != nil {
			return nil, walkErr
		}
	}

	// Invariant check (documented for clarity): registered + unregistered must
	// equal total_on_disk.
	// result.Summary.Registered + result.Summary.Unregistered == result.Summary.TotalOnDisk

	// Missing-record check: find store records whose paths fall under the
	// scanned directories but have no corresponding on-disk file (REQ-07, REQ-08).
	//
	// A record path is "under" the scanned set when it starts with one of the
	// scan directories (after normalisation to repo-relative forward-slash paths).
	//
	// When a dir entry was absolute, compute its repo-relative equivalent for
	// comparison against stored record paths (which are always repo-relative).
	normalised := make([]string, 0, len(scanDirs))
	for _, d := range scanDirs {
		var relDir string
		if filepath.IsAbs(d) {
			rel, relErr := filepath.Rel(repoRoot, d)
			if relErr != nil {
				// Cannot relativise — skip this dir in the missing check.
				continue
			}
			relDir = filepath.ToSlash(rel)
		} else {
			relDir = filepath.ToSlash(d)
		}
		// Ensure trailing slash so "work/design" does not accidentally match
		// "work/design-old".
		if !strings.HasSuffix(relDir, "/") {
			relDir += "/"
		}
		normalised = append(normalised, relDir)
	}

	for _, rec := range allRecords {
		recPath := filepath.ToSlash(rec.Path)

		// Check whether this record's path falls under one of the scanned dirs.
		underScan := false
		for _, nd := range normalised {
			if strings.HasPrefix(recPath+"/", nd) || strings.HasPrefix(recPath, nd) {
				underScan = true
				break
			}
		}
		if !underScan {
			continue
		}

		// If the on-disk walk did not find this path, the file is missing.
		if !onDiskPaths[recPath] {
			result.Summary.Missing++
			result.Missing = append(result.Missing, MissingRecord{
				Path:  rec.Path,
				DocID: rec.ID,
			})
		}
	}

	return result, nil
}
