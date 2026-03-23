package id

import (
	"fmt"
	"strings"

	"kanbanzai/internal/model"
)

const maxCollisionRetries = 3

// ExistsFunc checks whether a given ID already exists in the store.
type ExistsFunc func(id string) bool

// Allocator generates canonical IDs for all entity types.
type Allocator struct{}

// NewAllocator creates a new Allocator.
func NewAllocator() *Allocator {
	return &Allocator{}
}

// Allocate returns a new canonical ID for the given entity type.
//
// For Epics, epicSlug must be provided. It is validated per §8 rules and
// formatted as EPIC-{SLUG}. The exists function is used to check uniqueness.
//
// For all other entity types (Feature, Bug, Decision, Task, Document),
// a TSID13 is generated and formatted as {TYPE}-{TSID13}. The exists
// function is used for local collision checking with retry.
//
// epicSlug is ignored for non-Epic types.
// exists may be nil if collision checking is not available.
func (a *Allocator) Allocate(entityKind model.EntityKind, epicSlug string, exists ExistsFunc) (string, error) {
	if entityKind == model.EntityKindEpic {
		return a.allocateEpic(epicSlug, exists)
	}

	prefix, err := TypePrefix(entityKind)
	if err != nil {
		return "", err
	}

	return a.allocateTSID(prefix, exists)
}

// Validate returns an error if id is not a valid canonical ID for the given entity type.
func (a *Allocator) Validate(entityKind model.EntityKind, id string) error {
	if entityKind == model.EntityKindPlan {
		if !model.IsPlanID(id) {
			return fmt.Errorf("invalid plan ID %q: must match {prefix}{number}-{slug} format", id)
		}
		return nil
	}

	if entityKind == model.EntityKindEpic {
		return validateEpicID(id)
	}

	prefix, err := TypePrefix(entityKind)
	if err != nil {
		return fmt.Errorf("validate id %q: %w", id, err)
	}

	return validateTSIDBasedID(prefix, id)
}

// TypePrefix returns the ID type prefix for an entity kind.
func TypePrefix(entityKind model.EntityKind) (string, error) {
	switch entityKind {
	case model.EntityKindEpic:
		return "EPIC", nil
	case model.EntityKindFeature:
		return "FEAT", nil
	case model.EntityKindBug:
		return "BUG", nil
	case model.EntityKindDecision:
		return "DEC", nil
	case model.EntityKindTask:
		return "TASK", nil
	case model.EntityKindDocument:
		return "DOC", nil
	default:
		return "", fmt.Errorf("unknown entity kind %q", entityKind)
	}
}

// EntityKindFromPrefix returns the entity kind for a given type prefix.
func EntityKindFromPrefix(prefix string) (model.EntityKind, error) {
	switch strings.ToUpper(prefix) {
	case "EPIC":
		return model.EntityKindEpic, nil
	case "FEAT":
		return model.EntityKindFeature, nil
	case "BUG":
		return model.EntityKindBug, nil
	case "DEC":
		return model.EntityKindDecision, nil
	case "TASK":
		return model.EntityKindTask, nil
	case "DOC":
		return model.EntityKindDocument, nil
	default:
		return "", fmt.Errorf("unknown type prefix %q", prefix)
	}
}

// ParseCanonicalID splits a canonical ID into its type prefix and the identifier portion.
// For "FEAT-01J3K7MXP3RT5" returns ("FEAT", "01J3K7MXP3RT5", nil).
// For "EPIC-MYPROJECT" returns ("EPIC", "MYPROJECT", nil).
func ParseCanonicalID(id string) (prefix, ident string, err error) {
	idx := strings.Index(id, "-")
	if idx <= 0 || idx >= len(id)-1 {
		return "", "", fmt.Errorf("invalid ID format %q: missing type prefix", id)
	}
	return id[:idx], id[idx+1:], nil
}

// IsLegacyID returns true if the ID uses the old sequential format (e.g., FEAT-001, E-001, FEAT-001.1).
func IsLegacyID(id string) bool {
	// Old format: E-NNN, FEAT-NNN, BUG-NNN, DEC-NNN, FEAT-NNN.N
	if strings.HasPrefix(id, "E-") {
		rest := id[2:]
		return isAllDigits(rest)
	}
	for _, prefix := range []string{"FEAT-", "BUG-", "DEC-"} {
		if strings.HasPrefix(id, prefix) {
			rest := id[len(prefix):]
			// Check for FEAT-NNN.N (task) or FEAT-NNN
			if dotIdx := strings.Index(rest, "."); dotIdx > 0 {
				return isAllDigits(rest[:dotIdx]) && isAllDigits(rest[dotIdx+1:])
			}
			return isAllDigits(rest)
		}
	}
	return false
}

func (a *Allocator) allocateEpic(slug string, exists ExistsFunc) (string, error) {
	normalized, err := ValidateEpicSlug(slug)
	if err != nil {
		return "", err
	}

	id := "EPIC-" + normalized
	if exists != nil && exists(id) {
		return "", fmt.Errorf("epic slug %q is already in use", normalized)
	}

	return id, nil
}

func (a *Allocator) allocateTSID(prefix string, exists ExistsFunc) (string, error) {
	for attempt := 0; attempt <= maxCollisionRetries; attempt++ {
		tsid, err := GenerateTSID13()
		if err != nil {
			return "", fmt.Errorf("generate TSID: %w", err)
		}

		id := prefix + "-" + tsid

		if exists == nil || !exists(id) {
			return id, nil
		}
	}

	return "", fmt.Errorf("ID collision persisted after %d retries", maxCollisionRetries)
}

func validateEpicID(id string) error {
	if !strings.HasPrefix(id, "EPIC-") {
		return fmt.Errorf("invalid epic ID %q: must start with EPIC-", id)
	}

	slug := id[5:]
	_, err := ValidateEpicSlug(slug)
	if err != nil {
		return fmt.Errorf("invalid epic ID %q: %w", id, err)
	}
	return nil
}

func validateTSIDBasedID(expectedPrefix, id string) error {
	prefix, ident, err := ParseCanonicalID(id)
	if err != nil {
		return fmt.Errorf("invalid %s ID %q: %w", expectedPrefix, id, err)
	}

	if strings.ToUpper(prefix) != expectedPrefix {
		return fmt.Errorf("invalid %s ID %q: wrong prefix %q", expectedPrefix, id, prefix)
	}

	if err := ValidateTSID13(ident); err != nil {
		// Accept legacy IDs for robustness
		if IsLegacyID(id) {
			return nil
		}
		return fmt.Errorf("invalid %s ID %q: %w", expectedPrefix, id, err)
	}

	return nil
}

func isAllDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}
