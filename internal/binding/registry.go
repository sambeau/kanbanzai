package binding

import (
	"fmt"
	"sort"
	"strings"
)

// BindingRegistry is the loaded and validated binding registry.
type BindingRegistry struct {
	bindingPath string
	roleChecker RoleChecker
	bindings    *BindingFile
	warnings    []string
	loaded      bool
}

// NewBindingRegistry creates an unloaded registry.
func NewBindingRegistry(bindingPath string, roleChecker RoleChecker) *BindingRegistry {
	return &BindingRegistry{
		bindingPath: bindingPath,
		roleChecker: roleChecker,
	}
}

// Load reads, parses, and validates the binding file.
// Returns an error if there are any validation errors.
// Warnings are stored and accessible via Warnings().
func (r *BindingRegistry) Load() error {
	bf, loadErrs := LoadBindingFile(r.bindingPath)
	if len(loadErrs) > 0 {
		msgs := make([]string, len(loadErrs))
		for i, e := range loadErrs {
			msgs[i] = e.Error()
		}
		return fmt.Errorf("loading binding file: %s", strings.Join(msgs, "; "))
	}

	result := ValidateBindingFile(bf, r.roleChecker)
	if len(result.Errors) > 0 {
		msgs := make([]string, len(result.Errors))
		for i, e := range result.Errors {
			msgs[i] = e.Error()
		}
		return fmt.Errorf("validating binding file: %s", strings.Join(msgs, "; "))
	}

	r.bindings = bf
	r.warnings = result.Warnings
	r.loaded = true
	return nil
}

// Lookup returns the binding for the given stage name.
// Returns an error if not loaded or if the stage has no binding.
func (r *BindingRegistry) Lookup(stage string) (*StageBinding, error) {
	if !r.loaded {
		return nil, fmt.Errorf("registry not loaded")
	}
	if stage == "" {
		return nil, fmt.Errorf("stage name must not be empty")
	}
	sb, ok := r.bindings.StageBindings[stage]
	if !ok {
		return nil, fmt.Errorf("no binding for stage %q", stage)
	}
	return sb, nil
}

// Stages returns the sorted list of all stage names with bindings.
func (r *BindingRegistry) Stages() []string {
	if !r.loaded || r.bindings == nil {
		return nil
	}
	stages := make([]string, 0, len(r.bindings.StageBindings))
	for name := range r.bindings.StageBindings {
		stages = append(stages, name)
	}
	sort.Strings(stages)
	return stages
}

// Warnings returns any non-fatal warnings from the last Load.
func (r *BindingRegistry) Warnings() []string {
	return r.warnings
}
