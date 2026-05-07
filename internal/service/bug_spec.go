package service

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/sambeau/kanbanzai/internal/model"
)

// GenerateBugSpec auto-generates a specification document for a newly created
// bug from its observed/expected fields (FR-101). Returns warnings for
// non-fatal failures (FR-106). It is intended for use as a BugCreationHook.
func GenerateBugSpec(repoRoot string, bug model.Bug, docSvc *DocumentService) []string {
	var warnings []string

	// FR-103: Create work/bugs/<slug>/ directory if it doesn't exist.
	bugDir := filepath.Join(repoRoot, "work", "bugs", bug.Slug)
	if err := os.MkdirAll(bugDir, 0o755); err != nil {
		warning := fmt.Sprintf("spec generation warning: could not create directory %s: %v", bugDir, err)
		log.Printf("WARNING: %s", warning)
		return []string{warning}
	}

	specPath := filepath.Join(bugDir, "spec.md")
	relSpecPath := filepath.Join("work", "bugs", bug.Slug, "spec.md")

	// FR-104: Don't overwrite existing spec files (idempotent re-registration).
	specExists := false
	if _, err := os.Stat(specPath); err == nil {
		specExists = true
	}

	if !specExists {
		content := buildBugSpecContent(bug)
		if err := os.WriteFile(specPath, []byte(content), 0o644); err != nil {
			warning := fmt.Sprintf("spec generation warning: could not write spec file: %v", err)
			log.Printf("WARNING: %s", warning)
			return []string{warning}
		}
	}

	// FR-102: Register as specification document, owned by bug, auto-approved.
	title := "Bug Specification: " + bug.Name
	createdBy := bug.ReportedBy
	result, err := docSvc.SubmitDocument(SubmitDocumentInput{
		Path:        relSpecPath,
		Type:        "specification",
		Title:       title,
		Owner:       bug.ID,
		CreatedBy:   createdBy,
		AutoApprove: true,
	})

	if err != nil {
		// FR-104: If already registered (idempotent), suppress the error.
		if strings.Contains(err.Error(), "already registered") {
			return nil
		}
		warning := fmt.Sprintf("spec generation warning: document registration failed: %v", err)
		log.Printf("WARNING: %s", warning)
		return []string{warning}
	}

	// Collect any non-fatal warnings from registration.
	for _, w := range result.Warnings {
		warnings = append(warnings, fmt.Sprintf("spec generation warning: %s", w))
	}

	return warnings
}

// buildBugSpecContent creates the markdown content for a bug specification
// using the template defined in FR-101.
func buildBugSpecContent(bug model.Bug) string {
	return fmt.Sprintf(`# Bug Specification: %s

## Observed Behaviour
%s

## Expected Behaviour
%s

## Severity
%s | Priority: %s | Type: %s
`, bug.Name, bug.Observed, bug.Expected, bug.Severity, bug.Priority, bug.Type)
}

// GenerateBugFixPlan writes the fix-plan file and registers it as an approved
// dev-plan document when FixPlan is non-empty (FR-109, FR-110).
func GenerateBugFixPlan(repoRoot string, bug model.Bug, docSvc *DocumentService) []string {
	if bug.FixPlan == "" {
		return nil // FR-110: empty fix_plan is not an error.
	}

	var warnings []string

	planDir := filepath.Join(repoRoot, "work", "bugs", bug.Slug)
	if err := os.MkdirAll(planDir, 0o755); err != nil {
		warning := fmt.Sprintf("fix-plan generation warning: could not create directory %s: %v", planDir, err)
		log.Printf("WARNING: %s", warning)
		return []string{warning}
	}

	planPath := filepath.Join(planDir, "fix-plan.md")
	relPlanPath := filepath.Join("work", "bugs", bug.Slug, "fix-plan.md")

	// Do not overwrite existing fix-plan.
	if _, statErr := os.Stat(planPath); statErr != nil {
		if err := os.WriteFile(planPath, []byte(bug.FixPlan+"\n"), 0o644); err != nil {
			warning := fmt.Sprintf("fix-plan generation warning: could not write fix-plan file: %v", err)
			log.Printf("WARNING: %s", warning)
			return []string{warning}
		}
	}

	// FR-109: Register as dev-plan document, owned by bug, auto-approved.
	title := "Fix Plan: " + bug.Name
	createdBy := bug.ReportedBy
	result, err := docSvc.SubmitDocument(SubmitDocumentInput{
		Path:        relPlanPath,
		Type:        "dev-plan",
		Title:       title,
		Owner:       bug.ID,
		CreatedBy:   createdBy,
		AutoApprove: true,
	})

	if err != nil {
		if strings.Contains(err.Error(), "already registered") {
			return nil
		}
		warning := fmt.Sprintf("fix-plan generation warning: document registration failed: %v", err)
		log.Printf("WARNING: %s", warning)
		return []string{warning}
	}

	for _, w := range result.Warnings {
		warnings = append(warnings, fmt.Sprintf("fix-plan generation warning: %s", w))
	}

	return warnings
}
