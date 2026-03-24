package validate

import (
	"fmt"
	"math"
)

// KnowledgeInfo holds the fields of a single knowledge entry for health checking.
type KnowledgeInfo struct {
	ID     string
	Fields map[string]any
}

// ProfileInfo holds the minimum metadata of a context profile for health checking.
type ProfileInfo struct {
	ID       string
	Inherits string
}

// CheckKnowledgeHealth validates all knowledge entries.
//
// loadAll returns all knowledge entry infos.
// profileExists returns true if the named profile exists (used for scope validation).
func CheckKnowledgeHealth(
	loadAll func() ([]KnowledgeInfo, error),
	profileExists func(id string) bool,
) (*HealthReport, error) {
	entries, err := loadAll()
	if err != nil {
		return nil, fmt.Errorf("loading knowledge entries: %w", err)
	}

	report := &HealthReport{
		Summary: HealthSummary{
			EntitiesByType: make(map[string]int),
		},
	}

	validStatuses := map[string]struct{}{
		"contributed": {},
		"confirmed":   {},
		"disputed":    {},
		"stale":       {},
		"retired":     {},
	}

	for _, e := range entries {
		report.Summary.TotalEntities++
		report.Summary.EntitiesByType["knowledge_entry"]++

		// Check 1: required fields present.
		for _, field := range []string{"id", "tier", "topic", "scope", "content", "status"} {
			if v := toString(e.Fields[field]); v == "" {
				report.Errors = append(report.Errors, ValidationError{
					EntityType: "knowledge_entry",
					EntityID:   e.ID,
					Field:      field,
					Message:    fmt.Sprintf("required field %q is empty or missing", field),
				})
			}
		}

		// Check 2: status is a known value.
		status := toString(e.Fields["status"])
		if status != "" {
			if _, ok := validStatuses[status]; !ok {
				report.Errors = append(report.Errors, ValidationError{
					EntityType: "knowledge_entry",
					EntityID:   e.ID,
					Field:      "status",
					Message:    fmt.Sprintf("unknown status %q", status),
				})
			}
		}

		// Check 3: tier is 2 or 3.
		tier := toInt(e.Fields["tier"])
		if tier != 0 && tier != 2 && tier != 3 {
			report.Errors = append(report.Errors, ValidationError{
				EntityType: "knowledge_entry",
				EntityID:   e.ID,
				Field:      "tier",
				Message:    fmt.Sprintf("tier must be 2 or 3, got %d", tier),
			})
		}

		// Check 4: confidence consistency — stored value must match Wilson score.
		useCount := toInt(e.Fields["use_count"])
		missCount := toInt(e.Fields["miss_count"])
		stored := toFloat64(e.Fields["confidence"])
		expected := wilsonScoreForHealth(useCount, missCount)
		if stored != 0 && math.Abs(stored-expected) > 0.001 {
			report.Warnings = append(report.Warnings, ValidationWarning{
				EntityType: "knowledge_entry",
				EntityID:   e.ID,
				Field:      "confidence",
				Message: fmt.Sprintf(
					"confidence %.4f does not match computed Wilson score %.4f (use_count=%d, miss_count=%d)",
					stored, expected, useCount, missCount,
				),
			})
		}

		// Check 5: scope must be "project" or reference an existing profile.
		scope := toString(e.Fields["scope"])
		if scope != "" && scope != "project" && profileExists != nil && !profileExists(scope) {
			report.Errors = append(report.Errors, ValidationError{
				EntityType: "knowledge_entry",
				EntityID:   e.ID,
				Field:      "scope",
				Message:    fmt.Sprintf("scope %q does not reference an existing profile", scope),
			})
		}
	}

	report.Summary.ErrorCount = len(report.Errors)
	report.Summary.WarningCount = len(report.Warnings)

	return report, nil
}

// CheckProfileHealth validates all context profiles, including inheritance resolution.
//
// loadAll returns all profile infos (id + inherits).
// resolveProfile attempts full inheritance resolution for a profile by ID,
// returning nil on success or an error describing the failure (cycle, missing ref, etc.).
func CheckProfileHealth(
	loadAll func() ([]ProfileInfo, error),
	resolveProfile func(id string) error,
) (*HealthReport, error) {
	profiles, err := loadAll()
	if err != nil {
		return nil, fmt.Errorf("loading profiles: %w", err)
	}

	report := &HealthReport{
		Summary: HealthSummary{
			EntitiesByType: make(map[string]int),
		},
	}

	knownIDs := make(map[string]struct{}, len(profiles))
	for _, p := range profiles {
		knownIDs[p.ID] = struct{}{}
	}

	for _, p := range profiles {
		report.Summary.TotalEntities++
		report.Summary.EntitiesByType["profile"]++

		// Check 1: inherits reference must resolve to a known profile.
		if p.Inherits != "" {
			if _, ok := knownIDs[p.Inherits]; !ok {
				report.Errors = append(report.Errors, ValidationError{
					EntityType: "profile",
					EntityID:   p.ID,
					Field:      "inherits",
					Message:    fmt.Sprintf("inherits references unknown profile %q", p.Inherits),
				})
			}
		}

		// Check 2: full inheritance chain resolves without cycles.
		if resolveProfile != nil {
			if err := resolveProfile(p.ID); err != nil {
				report.Errors = append(report.Errors, ValidationError{
					EntityType: "profile",
					EntityID:   p.ID,
					Field:      "inherits",
					Message:    fmt.Sprintf("inheritance resolution failed: %v", err),
				})
			}
		}
	}

	report.Summary.ErrorCount = len(report.Errors)
	report.Summary.WarningCount = len(report.Warnings)

	return report, nil
}

// wilsonScoreForHealth computes the Wilson score lower bound for health check comparisons.
// Duplicates the formula from internal/knowledge/confidence.go to avoid import cycles.
func wilsonScoreForHealth(useCount, missCount int) float64 {
	n := useCount + missCount
	if n == 0 {
		return 0.5
	}
	p := float64(useCount) / float64(n)
	z := 1.96
	nf := float64(n)
	z2 := z * z
	numerator := p + z2/(2*nf) - z*math.Sqrt(p*(1-p)/nf+z2/(4*nf*nf))
	denominator := 1 + z2/nf
	v := numerator / denominator
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

// toInt extracts an integer from an any value. Handles int, int64, and float64 (from YAML).
func toInt(v any) int {
	if v == nil {
		return 0
	}
	switch val := v.(type) {
	case int:
		return val
	case int64:
		return int(val)
	case float64:
		return int(val)
	}
	return 0
}

// toFloat64 extracts a float64 from an any value. Handles float64 and int types.
func toFloat64(v any) float64 {
	if v == nil {
		return 0
	}
	switch val := v.(type) {
	case float64:
		return val
	case int:
		return float64(val)
	case int64:
		return float64(val)
	}
	return 0
}
