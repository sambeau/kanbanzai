package actionlog

import (
	"errors"

	"github.com/sambeau/kanbanzai/internal/service"
)

// ClassifyError maps an error to one of the 5 error type constants.
// It inspects the error chain for known sentinel types before falling
// back to ErrorInternalError.
func ClassifyError(err error) string {
	if err == nil {
		return ""
	}

	// Gate failures: prerequisite not satisfied for a lifecycle transition.
	// Advance failures are reported via fmt.Errorf with no sentinel currently,
	// so we look for keyword patterns as a heuristic.
	msg := err.Error()
	if containsAny(msg, "gate", "prerequisite", "not satisfied", "missing required document", "missing required", "required document") {
		return ErrorGateFailure
	}

	// Not-found errors.
	if errors.Is(err, service.ErrNotFound) || errors.Is(err, service.ErrReferenceNotFound) {
		return ErrorNotFound
	}

	// Validation errors.
	if errors.Is(err, service.ErrValidationFailed) || errors.Is(err, service.ErrInvalidTransition) {
		return ErrorValidationError
	}

	// Precondition errors (immutable fields, duplicate prevention, etc.).
	if errors.Is(err, service.ErrImmutableField) {
		return ErrorPreconditionError
	}
	if containsAny(msg, "precondition", "already exists", "immutable", "cannot be changed") {
		return ErrorPreconditionError
	}

	// Validation keyword patterns.
	if containsAny(msg, "invalid", "validation", "required field", "missing required") {
		return ErrorValidationError
	}

	return ErrorInternalError
}

// containsAny reports whether s contains any of the given substrings.
func containsAny(s string, subs ...string) bool {
	sl := toLower(s)
	for _, sub := range subs {
		if indexOf(sl, sub) >= 0 {
			return true
		}
	}
	return false
}

// toLower is a thin wrapper so we do not import strings at package level
// (avoids importing strings just for classify).
func toLower(s string) string {
	b := make([]byte, len(s))
	for i := range s {
		c := s[i]
		if c >= 65 && c <= 90 { // A-Z
			c += 32
		}
		b[i] = c
	}
	return string(b)
}

// indexOf returns the first index of substr in s, or -1.
func indexOf(s, substr string) int {
	if len(substr) == 0 {
		return 0
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
