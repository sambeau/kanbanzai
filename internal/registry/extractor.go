package registry

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	stageBindingsRelPath = ".kbz/stage-bindings.yaml"
	rolesRelDir          = ".kbz/roles"
)

// stageRaw is the minimal subset of a stage binding needed for registry extraction.
type stageRaw struct {
	Description   string     `yaml:"description"`
	Roles         []string   `yaml:"roles"`
	Skills        []string   `yaml:"skills"`
	HumanGate     bool       `yaml:"human_gate"`
	DocumentType  *string    `yaml:"document_type,omitempty"`
	Prerequisites *prereqRaw `yaml:"prerequisites,omitempty"`
}

type prereqRaw struct {
	Documents []docPrereqRaw `yaml:"documents,omitempty"`
	Tasks     *taskPrereqRaw `yaml:"tasks,omitempty"`
}

type docPrereqRaw struct {
	Type   string `yaml:"type"`
	Status string `yaml:"status"`
}

type taskPrereqRaw struct {
	MinCount    *int  `yaml:"min_count,omitempty"`
	AllTerminal *bool `yaml:"all_terminal,omitempty"`
}

// roleRaw is the minimal subset of a role YAML needed for registry extraction.
type roleRaw struct {
	ID       string `yaml:"id"`
	Identity string `yaml:"identity"`
	Inherits string `yaml:"inherits,omitempty"`
}

// Extract reads stage bindings and role YAML files from root and returns a
// RegistryModel. Stage ordering preserves declaration order from
// stage-bindings.yaml (yaml.Node walk, not map decode). Roles are sorted
// lexicographically by filename. Error messages name the file that caused
// any parse failure.
func Extract(root string) (*RegistryModel, error) {
	stages, err := extractStages(root)
	if err != nil {
		return nil, err
	}
	roles, err := extractRoles(root)
	if err != nil {
		return nil, err
	}
	return &RegistryModel{
		Stages: stages,
		Roles:  roles,
	}, nil
}

// extractStages reads .kbz/stage-bindings.yaml and returns an ordered slice
// of StageEntry preserving the declaration order of the stage_bindings mapping.
func extractStages(root string) ([]StageEntry, error) {
	absPath := filepath.Join(root, ".kbz", "stage-bindings.yaml")
	data, err := os.ReadFile(absPath)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", stageBindingsRelPath, err)
	}

	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("%s: %w", stageBindingsRelPath, err)
	}

	// An empty file produces a zero-value Document node with no content.
	if doc.Kind != yaml.DocumentNode || len(doc.Content) == 0 {
		return nil, nil
	}

	topNode := doc.Content[0]
	if topNode.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("%s: expected mapping at document root", stageBindingsRelPath)
	}

	// Find the "stage_bindings" key while preserving top-level order.
	var bindingsNode *yaml.Node
	for i := 0; i+1 < len(topNode.Content); i += 2 {
		if topNode.Content[i].Value == "stage_bindings" {
			bindingsNode = topNode.Content[i+1]
			break
		}
	}
	if bindingsNode == nil {
		return nil, nil
	}
	if bindingsNode.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("%s: stage_bindings must be a mapping", stageBindingsRelPath)
	}

	// Walk the mapping node in declaration order to preserve stage ordering.
	var stages []StageEntry
	for i := 0; i+1 < len(bindingsNode.Content); i += 2 {
		stageName := bindingsNode.Content[i].Value
		stageNode := bindingsNode.Content[i+1]

		var raw stageRaw
		if err := stageNode.Decode(&raw); err != nil {
			return nil, fmt.Errorf("%s: stage %q: %w", stageBindingsRelPath, stageName, err)
		}

		docType := ""
		if raw.DocumentType != nil {
			docType = *raw.DocumentType
		}

		stages = append(stages, StageEntry{
			Name:          stageName,
			Description:   raw.Description,
			Roles:         raw.Roles,
			Skills:        raw.Skills,
			HumanGate:     raw.HumanGate,
			DocumentType:  docType,
			Prerequisites: summarizePrereqs(raw.Prerequisites),
			SourcePath:    stageBindingsRelPath,
		})
	}

	return stages, nil
}

// summarizePrereqs builds a brief comma-separated summary of prerequisites
// suitable for inclusion in a registry table cell.
func summarizePrereqs(p *prereqRaw) string {
	if p == nil {
		return ""
	}
	var parts []string
	for _, d := range p.Documents {
		parts = append(parts, fmt.Sprintf("%s:%s", d.Type, d.Status))
	}
	if p.Tasks != nil {
		switch {
		case p.Tasks.AllTerminal != nil && *p.Tasks.AllTerminal:
			parts = append(parts, "tasks:all-terminal")
		case p.Tasks.MinCount != nil:
			parts = append(parts, fmt.Sprintf("tasks:min-%d", *p.Tasks.MinCount))
		}
	}
	return strings.Join(parts, ", ")
}

// extractRoles reads all *.yaml files from .kbz/roles/, sorted lexicographically
// by filename, and returns a map of role ID to RoleEntry.
func extractRoles(root string) (map[string]RoleEntry, error) {
	rolesDir := filepath.Join(root, ".kbz", "roles")
	entries, err := os.ReadDir(rolesDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("%s: directory not found", rolesRelDir)
		}
		return nil, fmt.Errorf("%s: %w", rolesRelDir, err)
	}

	// Collect .yaml files and sort explicitly by filename for determinism.
	var yamlFiles []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".yaml") {
			yamlFiles = append(yamlFiles, e.Name())
		}
	}
	slices.Sort(yamlFiles)

	roles := make(map[string]RoleEntry, len(yamlFiles))
	for _, name := range yamlFiles {
		relPath := rolesRelDir + "/" + name
		absPath := filepath.Join(rolesDir, name)

		data, err := os.ReadFile(absPath)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", relPath, err)
		}

		var raw roleRaw
		if err := yaml.Unmarshal(data, &raw); err != nil {
			return nil, fmt.Errorf("%s: %w", relPath, err)
		}

		// Derive the ID from the filename stem when the id field is absent.
		id := raw.ID
		if id == "" {
			id = strings.TrimSuffix(name, ".yaml")
		}

		roles[id] = RoleEntry{
			ID:         id,
			Identity:   raw.Identity,
			Inherits:   raw.Inherits,
			SourcePath: relPath,
		}
	}
	return roles, nil
}
