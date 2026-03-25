package git

import (
	"errors"
	"fmt"
	"time"
)

// CheckStaleness checks if a knowledge entry is stale based on its git anchors.
// An entry is stale if any anchored file was modified after lastConfirmed.
//
// If lastConfirmed is zero, the entry is considered never confirmed and will
// be marked stale if any anchored files exist and have commits.
//
// Returns StalenessInfo with IsStale=false if:
// - No anchors are provided
// - All anchored files were modified before lastConfirmed
// - All anchored files have no commits (new/untracked files are ignored)
func CheckStaleness(repoPath string, anchors []GitAnchor, lastConfirmed time.Time) (StalenessInfo, error) {
	info := StalenessInfo{
		LastConfirmed: lastConfirmed,
	}

	if len(anchors) == 0 {
		return info, nil
	}

	var staleFiles []StaleFile
	var firstErr error

	for _, anchor := range anchors {
		commit, modifiedAt, err := GetFileLastModified(repoPath, anchor.Path)
		if err != nil {
			// File not found or no commits means the anchor is invalid/stale
			// but not an error condition - the knowledge may reference a deleted file
			if errors.Is(err, ErrFileNotFound) {
				staleFiles = append(staleFiles, StaleFile{
					Path:       anchor.Path,
					ModifiedAt: time.Time{},
					Commit:     "",
				})
				continue
			}
			// Real errors (not a repo, git failure) should be reported
			if firstErr == nil {
				firstErr = fmt.Errorf("check anchor %q: %w", anchor.Path, err)
			}
			continue
		}

		// If never confirmed (zero time), any existing file makes entry stale
		// Otherwise, check if file was modified after last confirmation
		if lastConfirmed.IsZero() || modifiedAt.After(lastConfirmed) {
			staleFiles = append(staleFiles, StaleFile{
				Path:       anchor.Path,
				ModifiedAt: modifiedAt,
				Commit:     commit,
			})
		}
	}

	if firstErr != nil {
		return info, firstErr
	}

	if len(staleFiles) > 0 {
		info.IsStale = true
		info.StaleFiles = staleFiles
		info.StaleReason = buildStaleReason(staleFiles, lastConfirmed)
	}

	return info, nil
}

// buildStaleReason creates a human-readable explanation of staleness.
func buildStaleReason(staleFiles []StaleFile, lastConfirmed time.Time) string {
	if len(staleFiles) == 0 {
		return ""
	}

	// Check for deleted/missing files
	var missingFiles []string
	var modifiedFiles []string
	for _, f := range staleFiles {
		if f.Commit == "" {
			missingFiles = append(missingFiles, f.Path)
		} else {
			modifiedFiles = append(modifiedFiles, f.Path)
		}
	}

	if len(missingFiles) > 0 && len(modifiedFiles) == 0 {
		if len(missingFiles) == 1 {
			return fmt.Sprintf("Anchored file not found: %s", missingFiles[0])
		}
		return fmt.Sprintf("Anchored files not found: %d files", len(missingFiles))
	}

	if len(modifiedFiles) > 0 && len(missingFiles) == 0 {
		if lastConfirmed.IsZero() {
			if len(modifiedFiles) == 1 {
				return "Anchored file modified (entry never confirmed)"
			}
			return fmt.Sprintf("Anchored files modified (entry never confirmed): %d files", len(modifiedFiles))
		}
		if len(modifiedFiles) == 1 {
			return "Anchored file modified"
		}
		return fmt.Sprintf("Anchored files modified: %d files", len(modifiedFiles))
	}

	// Mix of missing and modified
	return fmt.Sprintf("Anchored files changed: %d modified, %d not found",
		len(modifiedFiles), len(missingFiles))
}
