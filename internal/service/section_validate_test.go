// Package service section_validate_test.go — tests for ValidateSections.
//
// Covers acceptance criteria AC-D01 through AC-D07 from the Workflow
// State Automation specification (FEAT-01KN73BFK4M4Z, Pillar D).
//
// Note: writeTestFile is shared with import_test.go (defined there).
package service

import (
	"os"
	"path/filepath"
	"testing"
)

// writeSVFile is a helper for section_validate tests. It creates dir/name
// and returns the full path (distinct from writeTestFile in import_test.go
// which does not return a path).
func writeSVFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
	return path
}

// AC-D01: Document with all required sections passes validation.
func TestValidateSections_AllPresent_Passes(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := writeSVFile(t, dir, "doc.md", "# Title\n\n## Overview\n\nSome text.\n\n## Scope\n\nMore text.\n\n## Functional Requirements\n\nRequirements here.\n")

	result, err := ValidateSections(path, []string{"Overview", "Scope", "Functional Requirements"})
	if err != nil {
		t.Fatalf("ValidateSections: unexpected error: %v", err)
	}
	if !result.Valid {
		t.Errorf("Valid = false, want true; missing: %v", result.Missing)
	}
	if len(result.Missing) != 0 {
		t.Errorf("Missing = %v, want empty", result.Missing)
	}
}

// AC-D02: Document missing one or more required sections returns them in Missing.
func TestValidateSections_MissingSections_ReturnsMissing(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := writeSVFile(t, dir, "doc.md", "# Title\n\n## Overview\n\nSome text.\n")

	result, err := ValidateSections(path, []string{"Overview", "Scope", "Acceptance Criteria"})
	if err != nil {
		t.Fatalf("ValidateSections: unexpected error: %v", err)
	}
	if result.Valid {
		t.Error("Valid = true, want false (two sections missing)")
	}
	if len(result.Missing) != 2 {
		t.Errorf("len(Missing) = %d, want 2; Missing: %v", len(result.Missing), result.Missing)
	}
	if len(result.Found) != 1 || result.Found[0] != "Overview" {
		t.Errorf("Found = %v, want [Overview]", result.Found)
	}
}

// AC-D03: Section heading matching is case-insensitive.
func TestValidateSections_CaseInsensitive(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := writeSVFile(t, dir, "doc.md", "## overview\n\n## scope\n")

	result, err := ValidateSections(path, []string{"Overview", "Scope"})
	if err != nil {
		t.Fatalf("ValidateSections: unexpected error: %v", err)
	}
	if !result.Valid {
		t.Errorf("Valid = false, want true; missing: %v", result.Missing)
	}
}

// AC-D03 (cont): Required section names use mixed-case; file uses UPPER-CASE.
func TestValidateSections_CaseInsensitive_UpperCase(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := writeSVFile(t, dir, "doc.md", "## OVERVIEW\n\n## SCOPE\n")

	result, err := ValidateSections(path, []string{"Overview", "Scope"})
	if err != nil {
		t.Fatalf("ValidateSections: unexpected error: %v", err)
	}
	if !result.Valid {
		t.Errorf("Valid = false, want true; missing: %v", result.Missing)
	}
}

// AC-D04: Level-1 (#) headings do not match level-2 requirements.
func TestValidateSections_Level1HeadingDoesNotMatch(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := writeSVFile(t, dir, "doc.md", "# Overview\n\nSome content.\n")

	result, err := ValidateSections(path, []string{"Overview"})
	if err != nil {
		t.Fatalf("ValidateSections: unexpected error: %v", err)
	}
	if result.Valid {
		t.Error("Valid = true, want false (level-1 heading should not match level-2 requirement)")
	}
	if len(result.Missing) != 1 || result.Missing[0] != "Overview" {
		t.Errorf("Missing = %v, want [Overview]", result.Missing)
	}
}

// AC-D04 (cont): Level-3+ (###) headings do not match level-2 requirements.
func TestValidateSections_Level3HeadingDoesNotMatch(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := writeSVFile(t, dir, "doc.md", "### Overview\n\nSome content.\n\n#### Deeper\n\nEven deeper.\n")

	result, err := ValidateSections(path, []string{"Overview"})
	if err != nil {
		t.Fatalf("ValidateSections: unexpected error: %v", err)
	}
	if result.Valid {
		t.Error("Valid = true, want false (level-3/4 headings should not match level-2 requirement)")
	}
}

// AC-D07: Document type with no declared required sections always passes.
func TestValidateSections_EmptyRequirements_AlwaysPasses(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := writeSVFile(t, dir, "doc.md", "Just some prose, no headings.")

	result, err := ValidateSections(path, nil)
	if err != nil {
		t.Fatalf("ValidateSections: unexpected error: %v", err)
	}
	if !result.Valid {
		t.Error("Valid = false, want true (empty required sections always passes)")
	}
	if len(result.Missing) != 0 {
		t.Errorf("Missing = %v, want empty", result.Missing)
	}
}

// AC-D07 (cont): empty slice also always passes.
func TestValidateSections_EmptySlice_AlwaysPasses(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := writeSVFile(t, dir, "doc.md", "Just prose.")

	result, err := ValidateSections(path, []string{})
	if err != nil {
		t.Fatalf("ValidateSections: unexpected error: %v", err)
	}
	if !result.Valid {
		t.Error("Valid = false, want true")
	}
}

// Error case: file does not exist returns an error (not a validation failure).
func TestValidateSections_FileMissing_ReturnsError(t *testing.T) {
	t.Parallel()
	_, err := ValidateSections("/tmp/kanbanzai-nonexistent-file-xyz.md", []string{"Overview"})
	if err == nil {
		t.Error("expected error for missing file, got nil")
	}
}

// A level-2 heading with a "##" prefix but no space should not match.
func TestValidateSections_DoubleHashNoSpace_DoesNotMatch(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := writeSVFile(t, dir, "doc.md", "##Overview\n\nContent.\n")

	result, err := ValidateSections(path, []string{"Overview"})
	if err != nil {
		t.Fatalf("ValidateSections: unexpected error: %v", err)
	}
	if result.Valid {
		t.Error("Valid = true, want false ('##Overview' without space should not match)")
	}
}

// Numbered headings ("## 1. Overview") should match the unnumbered required name ("Overview").
func TestValidateSections_NumberedHeadings_Match(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	content := "# Title\n\n## 1. Overview\n\nText.\n\n## 2. Scope\n\nText.\n\n## 3. Functional Requirements\n\nText.\n"
	path := writeSVFile(t, dir, "doc.md", content)

	result, err := ValidateSections(path, []string{"Overview", "Scope", "Functional Requirements"})
	if err != nil {
		t.Fatalf("ValidateSections: unexpected error: %v", err)
	}
	if !result.Valid {
		t.Errorf("Valid = false, want true; missing: %v", result.Missing)
	}
	if len(result.Found) != 3 {
		t.Errorf("len(Found) = %d, want 3; Found: %v", len(result.Found), result.Found)
	}
}

// Multi-digit numbered headings ("## 12. Overview") should also match.
func TestValidateSections_MultiDigitNumberedHeadings_Match(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := writeSVFile(t, dir, "doc.md", "## 12. Overview\n\n## 99. Scope\n")

	result, err := ValidateSections(path, []string{"Overview", "Scope"})
	if err != nil {
		t.Fatalf("ValidateSections: unexpected error: %v", err)
	}
	if !result.Valid {
		t.Errorf("Valid = false, want true; missing: %v", result.Missing)
	}
}

// Unnumbered headings still match (regression check).
func TestValidateSections_UnnumberedHeadings_StillMatch(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := writeSVFile(t, dir, "doc.md", "## Overview\n\n## Scope\n")

	result, err := ValidateSections(path, []string{"Overview", "Scope"})
	if err != nil {
		t.Fatalf("ValidateSections: unexpected error: %v", err)
	}
	if !result.Valid {
		t.Errorf("Valid = false, want true; missing: %v", result.Missing)
	}
}
