package card

import (
	"cmp"
	"fmt"
	"os"
	"slices"

	"gopkg.in/yaml.v3"
)

// AppliesToFilter declares the roles and stages a constraint applies to.
// Both fields must be non-empty; an empty list means "no match", not "all".
type AppliesToFilter struct {
	Roles  []string `yaml:"roles"`
	Stages []string `yaml:"stages"`
}

// ConstraintEntry is a single rule in the constraint registry.
// ID must be stable across versions; Rule must be a concise, actionable statement.
type ConstraintEntry struct {
	ID        string          `yaml:"id"`
	Rule      string          `yaml:"rule"`
	AppliesTo AppliesToFilter `yaml:"applies_to"`
}

// ConstraintRegistry holds all loaded and validated constraint entries in
// deterministic sorted order. Use LoadConstraintRegistry to create one.
type ConstraintRegistry struct {
	entries []ConstraintEntry
}

// constraintFile is the top-level structure of constraints.yaml.
type constraintFile struct {
	Constraints []ConstraintEntry `yaml:"constraints"`
}

// LoadConstraintRegistry reads constraints.yaml at path, validates every entry,
// and sorts the result by ID (REQ-NF-004). It returns a loud, actionable error
// that names the first missing required field (REQ-007).
func LoadConstraintRegistry(path string) (*ConstraintRegistry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("constraint registry: read %s: %w", path, err)
	}

	var f constraintFile
	if err := yaml.Unmarshal(data, &f); err != nil {
		return nil, fmt.Errorf("constraint registry: decode %s: %w", path, err)
	}

	for i, e := range f.Constraints {
		if e.ID == "" {
			return nil, fmt.Errorf("constraint registry: entry %d: missing required field \"id\"", i)
		}
		if e.Rule == "" {
			return nil, fmt.Errorf("constraint registry: entry %d (%s): missing required field \"rule\"", i, e.ID)
		}
		if len(e.AppliesTo.Roles) == 0 {
			return nil, fmt.Errorf("constraint registry: entry %d (%s): missing required field \"applies_to.roles\"", i, e.ID)
		}
		if len(e.AppliesTo.Stages) == 0 {
			return nil, fmt.Errorf("constraint registry: entry %d (%s): missing required field \"applies_to.stages\"", i, e.ID)
		}
	}

	// Sort by ID for deterministic ordering (REQ-NF-004).
	slices.SortFunc(f.Constraints, func(a, b ConstraintEntry) int {
		return cmp.Compare(a.ID, b.ID)
	})

	return &ConstraintRegistry{entries: f.Constraints}, nil
}

// Select returns all constraints that apply to both the given role and stage,
// preserving their sorted order. Repeated calls with the same inputs return
// identical results (REQ-NF-004).
func (r *ConstraintRegistry) Select(role, stage string) []ConstraintEntry {
	var result []ConstraintEntry
	for _, e := range r.entries {
		if containsString(e.AppliesTo.Roles, role) && containsString(e.AppliesTo.Stages, stage) {
			result = append(result, e)
		}
	}
	return result
}

// Entries returns a copy of all constraint entries in their sorted order.
func (r *ConstraintRegistry) Entries() []ConstraintEntry {
	out := make([]ConstraintEntry, len(r.entries))
	copy(out, r.entries)
	return out
}

func containsString(ss []string, s string) bool {
	for _, v := range ss {
		if v == s {
			return true
		}
	}
	return false
}
