// Package service retro.go — retrospective signal types, validation, and
// encoding helpers for Phase 1 of the Workflow Retrospective system (P5).
//
// Retrospective signals are stored as tagged knowledge entries. This file
// provides the pure data-layer functions:
//   - RetroSignalInput struct and valid-value sets
//   - ValidateRetroSignal — field-level validation with informative errors
//   - EncodeRetroContent  — canonical content string per spec §5.5
//   - RetroSignalTopic    — topic naming with per-task sequence suffix per spec §5.4
//
// See work/spec/workflow-retrospective.md §5 for the full schema definition.
package service

import (
	"fmt"
	"sort"
	"strings"
)

// ValidRetroCategories is the complete set of accepted retrospective signal
// categories. See work/spec/workflow-retrospective.md §5.2.
var ValidRetroCategories = map[string]bool{
	"workflow-friction":   true,
	"tool-gap":            true,
	"tool-friction":       true,
	"spec-ambiguity":      true,
	"context-gap":         true,
	"decomposition-issue": true,
	"design-gap":          true,
	"worked-well":         true,
}

// ValidRetroSeverities is the complete set of accepted severity levels.
// See work/spec/workflow-retrospective.md §5.3.
var ValidRetroSeverities = map[string]bool{
	"minor":       true,
	"moderate":    true,
	"significant": true,
}

// RetroSignalInput represents a single retrospective signal submitted at task
// completion. See work/spec/workflow-retrospective.md §5.1.
type RetroSignalInput struct {
	// Category classifies the type of process observation.
	// Must be one of the keys in ValidRetroCategories.
	Category string

	// Observation is a one- or two-sentence description of what happened.
	// Required.
	Observation string

	// Severity describes how much friction the observation caused.
	// Must be one of the keys in ValidRetroSeverities.
	Severity string

	// Suggestion is an optional note on what could be done differently.
	Suggestion string

	// RelatedDecision is an optional Decision ID (e.g. "DEC-042") referencing
	// an active workflow-experiment decision. Phase 3 field — stored in the
	// content string but not acted on in Phase 1.
	RelatedDecision string
}

// RetroSignalValidationError is returned when a signal fails validation.
// It carries the original signal and a human-readable reason, enabling
// per-signal rejection without blocking overall task completion.
type RetroSignalValidationError struct {
	Signal RetroSignalInput
	Reason string
}

func (e *RetroSignalValidationError) Error() string {
	return e.Reason
}

// ValidateRetroSignal validates a single retrospective signal.
// Returns a *RetroSignalValidationError on the first failing constraint,
// or nil if the signal is valid.
func ValidateRetroSignal(s RetroSignalInput) error {
	if strings.TrimSpace(s.Category) == "" {
		return &RetroSignalValidationError{Signal: s, Reason: "category is required"}
	}
	if !ValidRetroCategories[s.Category] {
		return &RetroSignalValidationError{
			Signal: s,
			Reason: fmt.Sprintf(
				"unknown category %q; valid categories: %s",
				s.Category,
				strings.Join(sortedRetroKeys(ValidRetroCategories), ", "),
			),
		}
	}
	if strings.TrimSpace(s.Observation) == "" {
		return &RetroSignalValidationError{Signal: s, Reason: "observation is required"}
	}
	if strings.TrimSpace(s.Severity) == "" {
		return &RetroSignalValidationError{Signal: s, Reason: "severity is required"}
	}
	if !ValidRetroSeverities[s.Severity] {
		return &RetroSignalValidationError{
			Signal: s,
			Reason: fmt.Sprintf(
				"unknown severity %q; valid values: minor, moderate, significant",
				s.Severity,
			),
		}
	}
	return nil
}

// EncodeRetroContent formats a signal as the canonical knowledge entry content
// string. See work/spec/workflow-retrospective.md §5.5.
//
// Format when suggestion is absent:
//
//	[{severity}] {category}: {observation}
//
// Format when suggestion is present:
//
//	[{severity}] {category}: {observation} Suggestion: {suggestion}
//
// Format when related_decision is also present:
//
//	[{severity}] {category}: {observation} Suggestion: {suggestion} Related: {decision_id}
//
// The function assumes the signal has already passed ValidateRetroSignal.
func EncodeRetroContent(s RetroSignalInput) string {
	var b strings.Builder
	fmt.Fprintf(&b, "[%s] %s: %s", s.Severity, s.Category, s.Observation)
	if suggestion := strings.TrimSpace(s.Suggestion); suggestion != "" {
		fmt.Fprintf(&b, " Suggestion: %s", suggestion)
	}
	if related := strings.TrimSpace(s.RelatedDecision); related != "" {
		fmt.Fprintf(&b, " Related: %s", related)
	}
	return b.String()
}

// RetroSignalTopic returns the knowledge entry topic for the nth signal
// (1-based) from a given task. See work/spec/workflow-retrospective.md §5.4.
//
// The first validated signal from a task uses topic "retro-{taskID}".
// Subsequent validated signals use "retro-{taskID}-2", "retro-{taskID}-3", etc.
// n must be >= 1.
func RetroSignalTopic(taskID string, n int) string {
	if n <= 1 {
		return "retro-" + taskID
	}
	return fmt.Sprintf("retro-%s-%d", taskID, n)
}

// sortedRetroKeys returns the keys of a bool map sorted alphabetically.
// Used to produce deterministic error messages listing valid category names.
func sortedRetroKeys(m map[string]bool) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
