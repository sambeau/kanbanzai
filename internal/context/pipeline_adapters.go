// Package context pipeline_adapters.go — production adapters wrapping concrete stores
// to satisfy the pipeline's dependency interfaces (RoleResolver, SkillResolver, BindingResolver).
package context

import (
	"fmt"

	"github.com/sambeau/kanbanzai/internal/binding"
	"github.com/sambeau/kanbanzai/internal/skill"
)

// ─── RoleResolver adapter ─────────────────────────────────────────────────────

// RoleStoreAdapter wraps a *RoleStore to satisfy the RoleResolver interface.
type RoleStoreAdapter struct {
	Store *RoleStore
}

// Resolve loads and resolves a role by ID, walking the inheritance chain.
func (a *RoleStoreAdapter) Resolve(id string) (*ResolvedRole, error) {
	return ResolveRole(a.Store, id)
}

// ─── SkillResolver adapter ────────────────────────────────────────────────────

// SkillStoreAdapter wraps a *skill.SkillStore to satisfy the SkillResolver interface.
type SkillStoreAdapter struct {
	Store *skill.SkillStore
}

// Load reads, parses, and validates a skill by name.
// Validation warnings are ignored; only errors are propagated.
func (a *SkillStoreAdapter) Load(name string) (*skill.Skill, error) {
	sk, _, err := a.Store.Load(name)
	return sk, err
}

// ─── BindingResolver adapter ──────────────────────────────────────────────────

// BindingFileAdapter wraps a *binding.BindingFile to satisfy the BindingResolver interface.
type BindingFileAdapter struct {
	File *binding.BindingFile
}

// Lookup retrieves the stage binding for the given lifecycle stage.
func (a *BindingFileAdapter) Lookup(stage string) (*binding.StageBinding, error) {
	if a.File == nil || a.File.StageBindings == nil {
		return nil, fmt.Errorf("no binding registry loaded")
	}
	sb, ok := a.File.StageBindings[stage]
	if !ok {
		return nil, fmt.Errorf("no binding for stage %q", stage)
	}
	return sb, nil
}

// ─── KnowledgeSurfacer stub ───────────────────────────────────────────────────

// NoOpSurfacer is a KnowledgeSurfacer that always returns an empty result.
// Used as a placeholder until the Knowledge Auto-Surfacing feature is implemented.
type NoOpSurfacer struct{}

// Surface returns an empty slice and no error.
func (NoOpSurfacer) Surface(_ SurfaceInput) ([]SurfacedEntry, error) {
	return nil, nil
}
