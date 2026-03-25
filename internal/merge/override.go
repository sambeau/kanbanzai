package merge

import (
	"errors"
	"fmt"
	"time"
)

// Override records a gate override event.
type Override struct {
	// Gate is the name of the gate that was overridden.
	Gate string

	// Reason is the human-provided justification for the override.
	Reason string

	// OverriddenBy identifies who performed the override.
	OverriddenBy string

	// OverriddenAt is when the override was recorded.
	OverriddenAt time.Time
}

// OverrideRequest contains the parameters for an override.
type OverrideRequest struct {
	// EntityID is the ID of the entity being overridden.
	EntityID string

	// Gates lists which gates to override.
	// If empty, all blocking gates are overridden.
	Gates []string

	// Reason is required and must explain why the override is justified.
	Reason string

	// OverriddenBy identifies the user performing the override.
	OverriddenBy string
}

// Override validation errors.
var (
	ErrOverrideNoEntityID     = errors.New("entity_id is required")
	ErrOverrideNoReason       = errors.New("reason is required")
	ErrOverrideNoOverriddenBy = errors.New("overridden_by is required")
	ErrOverrideReasonTooShort = errors.New("reason must be at least 10 characters")
)

// ValidateOverride checks if an override request is valid.
// Returns nil if valid, or an error describing the validation failure.
func ValidateOverride(req OverrideRequest) error {
	if req.EntityID == "" {
		return ErrOverrideNoEntityID
	}

	if req.Reason == "" {
		return ErrOverrideNoReason
	}

	// Require a meaningful reason (at least 10 characters)
	if len(req.Reason) < 10 {
		return ErrOverrideReasonTooShort
	}

	if req.OverriddenBy == "" {
		return ErrOverrideNoOverriddenBy
	}

	return nil
}

// CreateOverrides creates Override records from a validated request.
// If no gates are specified, it creates overrides for all blocking failures.
func CreateOverrides(req OverrideRequest, blockingFailures []GateResult, now time.Time) []Override {
	var overrides []Override

	// Determine which gates to override
	var gateNames []string
	if len(req.Gates) > 0 {
		gateNames = req.Gates
	} else {
		// Override all blocking failures
		for _, f := range blockingFailures {
			gateNames = append(gateNames, f.Name)
		}
	}

	for _, gateName := range gateNames {
		overrides = append(overrides, Override{
			Gate:         gateName,
			Reason:       req.Reason,
			OverriddenBy: req.OverriddenBy,
			OverriddenAt: now,
		})
	}

	return overrides
}

// FormatOverride returns a human-readable description of an override.
func FormatOverride(o Override) string {
	return fmt.Sprintf("gate %q overridden by %s at %s: %s",
		o.Gate,
		o.OverriddenBy,
		o.OverriddenAt.Format(time.RFC3339),
		o.Reason,
	)
}
