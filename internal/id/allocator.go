package id

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"kanbanzai/internal/model"
)

var (
	typedIDPattern    = regexp.MustCompile(`^(E|FEAT|BUG|DEC)-(\d{3,})$`)
	featureTaskIDExpr = regexp.MustCompile(`^(FEAT-\d{3,})\.(\d+)$`)
)

// Allocator allocates canonical Phase 1 IDs from an existing set.
type Allocator struct{}

// NewAllocator creates a new Allocator.
func NewAllocator() *Allocator {
	return &Allocator{}
}

// Allocate returns the next canonical ID for the given entity type.
//
// For Epics, Features, Bugs, and Decisions, IDs are global typed sequential IDs:
//   - E-001
//   - FEAT-001
//   - BUG-001
//   - DEC-001
//
// For Tasks, IDs are feature-local sub-IDs:
//   - FEAT-001.1
//   - FEAT-001.2
func (a *Allocator) Allocate(entityKind model.EntityKind, existingIDs []string, featureID string) (string, error) {
	switch entityKind {
	case model.EntityKindEpic, model.EntityKindFeature, model.EntityKindBug, model.EntityKindDecision:
		return allocateTypedID(entityKind, existingIDs)
	case model.EntityKindTask:
		return allocateTaskID(existingIDs, featureID)
	default:
		return "", fmt.Errorf("allocate id for %q: unknown entity kind", entityKind)
	}
}

// Validate returns an error if id is not valid for the given entity type.
func (a *Allocator) Validate(entityKind model.EntityKind, id string, featureID string) error {
	switch entityKind {
	case model.EntityKindEpic, model.EntityKindFeature, model.EntityKindBug, model.EntityKindDecision:
		return validateTypedID(entityKind, id)
	case model.EntityKindTask:
		return validateTaskID(id, featureID)
	default:
		return fmt.Errorf("validate id %q: unknown entity kind %q", id, entityKind)
	}
}

// SortIDs returns a stable canonical ordering for a slice of IDs of the same family.
func (a *Allocator) SortIDs(ids []string) []string {
	out := append([]string(nil), ids...)
	sort.Slice(out, func(i, j int) bool {
		return compareID(out[i], out[j]) < 0
	})
	return out
}

func allocateTypedID(entityKind model.EntityKind, existingIDs []string) (string, error) {
	prefix, err := prefixForEntityKind(entityKind)
	if err != nil {
		return "", err
	}

	maxValue := 0
	for _, existing := range existingIDs {
		foundPrefix, value, parseErr := parseTypedID(existing)
		if parseErr != nil {
			continue
		}
		if foundPrefix != prefix {
			continue
		}
		if value > maxValue {
			maxValue = value
		}
	}

	return fmt.Sprintf("%s-%03d", prefix, maxValue+1), nil
}

func allocateTaskID(existingIDs []string, featureID string) (string, error) {
	if err := validateTypedID(model.EntityKindFeature, featureID); err != nil {
		return "", fmt.Errorf("allocate task id: invalid feature id: %w", err)
	}

	maxValue := 0
	for _, existing := range existingIDs {
		parentFeatureID, value, ok := parseTaskID(existing)
		if !ok || parentFeatureID != featureID {
			continue
		}
		if value > maxValue {
			maxValue = value
		}
	}

	return fmt.Sprintf("%s.%d", featureID, maxValue+1), nil
}

func validateTypedID(entityKind model.EntityKind, id string) error {
	expectedPrefix, err := prefixForEntityKind(entityKind)
	if err != nil {
		return err
	}

	prefix, _, parseErr := parseTypedID(id)
	if parseErr != nil {
		return fmt.Errorf("invalid %s id %q: %w", entityKind, id, parseErr)
	}
	if prefix != expectedPrefix {
		return fmt.Errorf("invalid %s id %q: expected prefix %s", entityKind, id, expectedPrefix)
	}

	return nil
}

func validateTaskID(id string, featureID string) error {
	if err := validateTypedID(model.EntityKindFeature, featureID); err != nil {
		return fmt.Errorf("invalid task feature id: %w", err)
	}

	parentFeatureID, value, ok := parseTaskID(id)
	if !ok {
		return fmt.Errorf("invalid task id %q: must match %s.N", id, featureID)
	}
	if parentFeatureID != featureID {
		return fmt.Errorf("invalid task id %q: expected feature prefix %s", id, featureID)
	}
	if value < 1 {
		return fmt.Errorf("invalid task id %q: sequence must be >= 1", id)
	}

	return nil
}

func prefixForEntityKind(entityKind model.EntityKind) (string, error) {
	switch entityKind {
	case model.EntityKindEpic:
		return "E", nil
	case model.EntityKindFeature:
		return "FEAT", nil
	case model.EntityKindBug:
		return "BUG", nil
	case model.EntityKindDecision:
		return "DEC", nil
	default:
		return "", fmt.Errorf("unknown typed entity kind %q", entityKind)
	}
}

func parseTypedID(id string) (string, int, error) {
	matches := typedIDPattern.FindStringSubmatch(strings.TrimSpace(id))
	if matches == nil {
		return "", 0, fmt.Errorf("must match PREFIX-NNN")
	}

	value, err := strconv.Atoi(matches[2])
	if err != nil {
		return "", 0, fmt.Errorf("parse numeric component: %w", err)
	}
	if value < 1 {
		return "", 0, fmt.Errorf("numeric component must be >= 1")
	}

	return matches[1], value, nil
}

func parseTaskID(id string) (string, int, bool) {
	matches := featureTaskIDExpr.FindStringSubmatch(strings.TrimSpace(id))
	if matches == nil {
		return "", 0, false
	}

	value, err := strconv.Atoi(matches[2])
	if err != nil || value < 1 {
		return "", 0, false
	}

	return matches[1], value, true
}

func compareID(left string, right string) int {
	if left == right {
		return 0
	}

	leftFeature, leftTaskValue, leftIsTask := parseTaskID(left)
	rightFeature, rightTaskValue, rightIsTask := parseTaskID(right)

	switch {
	case leftIsTask && rightIsTask:
		if leftFeature != rightFeature {
			return strings.Compare(leftFeature, rightFeature)
		}
		return compareInts(leftTaskValue, rightTaskValue)
	case leftIsTask:
		return 1
	case rightIsTask:
		return -1
	}

	leftPrefix, leftValue, leftErr := parseTypedID(left)
	rightPrefix, rightValue, rightErr := parseTypedID(right)

	switch {
	case leftErr == nil && rightErr == nil:
		if leftPrefix != rightPrefix {
			return strings.Compare(leftPrefix, rightPrefix)
		}
		return compareInts(leftValue, rightValue)
	case leftErr == nil:
		return -1
	case rightErr == nil:
		return 1
	default:
		return strings.Compare(left, right)
	}
}

func compareInts(left int, right int) int {
	switch {
	case left < right:
		return -1
	case left > right:
		return 1
	default:
		return 0
	}
}
