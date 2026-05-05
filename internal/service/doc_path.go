// Package service implements the kanbanzai MCP service layer.
//
// doc_path.go implements canonicalDocPath — the core logic for
// computing standard document file paths from a document type and
// a parent entity ID (plan, batch, or feature).
//
// See: REQ-001 through REQ-003 of work/P50-retro-may-2026/P50-spec-doc-path-tool.md
package service

import (
	"fmt"
	"strings"

	"github.com/sambeau/kanbanzai/internal/model"
)

// docTypeAbbreviations maps user-facing document types to their
// path-component abbreviations (REQ-002).
var docTypeAbbreviations = map[string]string{
	"design":        "design",
	"specification": "spec",
	"dev-plan":      "dev-plan",
	"research":      "research",
	"report":        "report",
	"policy":        "policy",
	"prompt":        "",      // prompts use a different path convention
	"retro":         "retro", // retrospective (legacy normalised form)
}

// CanonicalDocPath returns the canonical file path for a document based on
// its type and parent entity. The parent is resolved upward to a plan so
// that the path follows the convention:
//
//	work/{plan-slug}/{plan-id}-{type-abbrev}-{topic-slug}.md
//
// REQ-001: returns a canonical path for any supported doc type + parent combo.
// REQ-002: uses abbreviated type forms (spec, dev-plan, etc.).
// REQ-003: resolves batches and features upward to their owning plan.
func (s *EntityService) CanonicalDocPath(docType string, parentEntityID string) (string, error) {
	docType = strings.ToLower(strings.TrimSpace(docType))
	parentEntityID = strings.TrimSpace(parentEntityID)

	if parentEntityID == "" {
		return "", fmt.Errorf("cannot determine path: no parent entity provided. Specify a parent plan, batch, or feature ID")
	}

	// Resolve to the owning plan.
	planID, planSlug, err := s.resolveToPlan(parentEntityID)
	if err != nil {
		return "", fmt.Errorf("cannot determine path: %w", err)
	}

	// Handle prompts specially (REQ-007): they go under work/{plan-slug}/prompts/
	if docType == "prompt" {
		return fmt.Sprintf("work/%s/prompts/%s.md", planSlug, planSlug), nil
	}

	// Look up the type abbreviation (REQ-002).
	abbrev, ok := docTypeAbbreviations[docType]
	if !ok {
		return "", fmt.Errorf("unsupported document type %q — valid types are: design, specification, dev-plan, research, report, policy", docType)
	}

	return fmt.Sprintf("work/%s/%s-%s-%s.md", planSlug, planID, abbrev, planSlug), nil
}

// resolveToPlan resolves an entity ID upward to its owning plan ID and slug.
func (s *EntityService) resolveToPlan(entityID string) (planID, planSlug string, err error) {
	// Check features first (FEAT- prefix).
	if isFeatureID(entityID) {
		return s.resolveFeatureToPlan(entityID)
	}

	// Plan/batch IDs: resolve upward through the batch hierarchy.
	// Both plans and batches share the same ID pattern (IsPlanID = IsBatchID),
	// so we always resolve through resolveBatchToPlan which checks for a
	// parent plan.
	if model.IsPlanID(entityID) {
		return s.resolveBatchToPlan(entityID)
	}

	return "", "", fmt.Errorf("parent entity %s not found", entityID)
}

// resolveBatchToPlan looks up a batch entity and returns its parent plan's ID and slug.
// If the entity has no parent plan, it returns itself as the owning plan.
func (s *EntityService) resolveBatchToPlan(batchID string) (string, string, error) {
	batch, err := s.GetPlan(batchID)
	if err != nil {
		return "", "", fmt.Errorf("parent entity %s not found", batchID)
	}
	parent, _ := batch.State["parent"].(string)
	if parent == "" || !model.IsPlanID(parent) {
		// No parent plan: this entity IS the owning plan.
		_, _, slug := model.ParsePlanID(batch.ID)
		return batch.ID, slug, nil
	}
	// Recursively resolve upward.
	return s.resolveBatchToPlan(parent)
}

// resolveFeatureToPlan looks up a feature entity, finds its parent batch,
// then resolves the batch to its parent plan.
func (s *EntityService) resolveFeatureToPlan(featureID string) (string, string, error) {
	feat, err := s.Get("feature", featureID, "")
	if err != nil {
		return "", "", fmt.Errorf("parent entity %s not found", featureID)
	}
	parent, _ := feat.State["parent"].(string)
	if parent == "" {
		return "", "", fmt.Errorf("feature %s has no parent batch or plan", featureID)
	}
	// The parent could be a plan or a batch; resolveBatchToPlan handles both.
	if model.IsPlanID(parent) {
		return s.resolveBatchToPlan(parent)
	}
	return "", "", fmt.Errorf("feature %s parent %s is not a plan or batch ID", featureID, parent)
}

// isFeatureID reports whether entityID looks like a feature ID (FEAT-...).
func isFeatureID(entityID string) bool {
	return strings.HasPrefix(strings.ToUpper(entityID), "FEAT-")
}
