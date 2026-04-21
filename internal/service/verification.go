package service

import (
	"fmt"
	"os"
	"strings"
)

// VerificationAggregationResult holds the outcome of AggregateTaskVerification.
type VerificationAggregationResult struct {
	Status  string // "passed", "partial", or "none"
	Written bool   // true if a write to the feature entity was performed
}

// AggregateTaskVerification aggregates per-task verification fields into a
// feature-level summary and writes the result to the feature entity.
//
// Status values:
//   - "passed"  — all done tasks have non-empty verification
//   - "partial" — at least one done task has verification and at least one does not
//   - "none"    — no done tasks have non-empty verification
//
// If status is "none" the feature entity is not updated and Written is false.
// If the UpdateEntity write fails the error is logged to stderr but not returned.
func (s *DispatchService) AggregateTaskVerification(featureID string) (*VerificationAggregationResult, error) {
	featureID = strings.TrimSpace(featureID)
	if featureID == "" {
		return nil, fmt.Errorf("feature_id is required")
	}

	// List all tasks for this feature.
	tasks, err := s.entitySvc.ListEntitiesFiltered(ListFilteredInput{
		Type:   "task",
		Parent: featureID,
	})
	if err != nil {
		return nil, fmt.Errorf("list tasks for feature %s: %w", featureID, err)
	}

	// Collect verification values from done tasks only.
	// wont_do / not-planned tasks are excluded naturally by this filter.
	type taskVerif struct {
		id           string
		verification string
	}
	var done []taskVerif
	for _, t := range tasks {
		status := stringFromState(t.State, "status")
		if status != "done" {
			continue
		}
		done = append(done, taskVerif{
			id:           t.ID,
			verification: stringFromState(t.State, "verification"),
		})
	}

	// Derive verification status.
	var withVerif, withoutVerif int
	for _, tv := range done {
		if tv.verification != "" {
			withVerif++
		} else {
			withoutVerif++
		}
	}

	var status string
	switch {
	case withVerif == 0:
		status = "none"
	case withoutVerif == 0:
		status = "passed"
	default:
		status = "partial"
	}

	if status == "none" {
		return &VerificationAggregationResult{Status: "none", Written: false}, nil
	}

	// Build newline-separated summary: "<TASK-ID>: <verification text>"
	lines := make([]string, 0, len(done))
	for _, tv := range done {
		v := tv.verification
		if v == "" {
			v = "(no verification recorded)"
		}
		lines = append(lines, tv.id+": "+v)
	}
	summary := strings.Join(lines, "\n")

	// Write verification fields to the feature entity (best-effort).
	_, writeErr := s.entitySvc.UpdateEntity(UpdateEntityInput{
		Type: "feature",
		ID:   featureID,
		Fields: map[string]string{
			"verification":        summary,
			"verification_status": status,
		},
	})
	if writeErr != nil {
		fmt.Fprintf(os.Stderr, "AggregateTaskVerification: failed to update feature %s: %v\n", featureID, writeErr)
	}

	return &VerificationAggregationResult{Status: status, Written: writeErr == nil}, nil
}
