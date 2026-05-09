// Package registry provides the data model and extractor for the role/skill
// registry derived from .kbz/stage-bindings.yaml and .kbz/roles/*.yaml.
package registry

// StageEntry holds the registry metadata for a single workflow stage.
// Fields are extracted from .kbz/stage-bindings.yaml.
type StageEntry struct {
	// Name is the stage key as declared in stage-bindings.yaml.
	Name string
	// Description is the human-readable description of the stage.
	Description string
	// Roles lists the primary roles bound to this stage.
	Roles []string
	// Skills lists the skills bound to this stage.
	Skills []string
	// HumanGate indicates whether the stage requires a human checkpoint.
	HumanGate bool
	// DocumentType is the document_type field value, or empty if absent.
	DocumentType string
	// Prerequisites is a brief summary of entry prerequisites, or empty if absent.
	// Format: comma-separated items, e.g. "design:approved, tasks:min-1".
	Prerequisites string
	// SourcePath is the path to stage-bindings.yaml relative to the repo root.
	SourcePath string
}

// RoleEntry holds the registry metadata for a single role.
// Fields are extracted from .kbz/roles/<id>.yaml.
type RoleEntry struct {
	// ID is the role identifier (matches the id field in the YAML file).
	ID string
	// Identity is the role's identity string.
	Identity string
	// Inherits is the ID of the parent role, or empty if the role has no parent.
	Inherits string
	// SourcePath is the path to the role file relative to the repo root.
	SourcePath string
}

// RegistryModel is the complete extracted registry derived from canonical sources.
// Stages are ordered by their declaration order in stage-bindings.yaml.
// Roles are keyed by role ID and sorted lexicographically by source filename
// when accessed via RolesSorted.
type RegistryModel struct {
	// Stages is the ordered list of stage entries.
	Stages []StageEntry
	// Roles maps role ID to role entry.
	Roles map[string]RoleEntry
}
