package binding

import (
	"fmt"
	"sort"
	"strings"
)

// RoleChecker tests whether a role ID has a corresponding role file.
type RoleChecker func(id string) bool

// ValidationResult holds errors (blocking) and warnings (non-blocking) separately.
type ValidationResult struct {
	Errors   []error
	Warnings []string
}

// ValidateBindingFile checks stage names, cross-references, and consistency rules
// across all binding entries. roleChecker may be nil (skips role fallback checks).
func ValidateBindingFile(bf *BindingFile, roleChecker RoleChecker) *ValidationResult {
	result := &ValidationResult{}

	// Collect sorted valid stage names for error messages.
	validList := make([]string, 0, len(validStages))
	for s := range validStages {
		validList = append(validList, s)
	}
	sort.Strings(validList)

	for stageName, binding := range bf.StageBindings {
		// 1. Check stage name is valid.
		if !validStages[stageName] {
			result.Errors = append(result.Errors, fmt.Errorf(
				"invalid stage name %q; valid stages: %s",
				stageName, strings.Join(validList, ", "),
			))
		}

		// 2. Run per-binding validation.
		if errs := ValidateBinding(binding, stageName); len(errs) > 0 {
			result.Errors = append(result.Errors, errs...)
		}

		// 3. Role existence checks with fallback (only when roleChecker is provided).
		if roleChecker != nil {
			checkRoles(roleChecker, binding.Roles, stageName, result)
			if binding.SubAgents != nil {
				checkRoles(roleChecker, binding.SubAgents.Roles, stageName, result)
			}
		}
	}

	return result
}

// checkRoles verifies each role via roleChecker, attempting a single-level
// fallback by stripping the last hyphen segment when the exact ID is not found.
func checkRoles(rc RoleChecker, roles []string, stageName string, result *ValidationResult) {
	for _, role := range roles {
		if rc(role) {
			continue
		}

		// Attempt fallback: strip the last hyphen segment.
		if idx := strings.LastIndex(role, "-"); idx > 0 {
			fallback := role[:idx]
			if rc(fallback) {
				result.Warnings = append(result.Warnings, fmt.Sprintf(
					"role %q not found, resolved via fallback to %q in stage %q",
					role, fallback, stageName,
				))
				continue
			}
		}

		result.Warnings = append(result.Warnings, fmt.Sprintf(
			"role %q not found in stage %q",
			role, stageName,
		))
	}
}
