package context

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// RoleStore reads role definitions from the filesystem.
// It checks newRoot (.kbz/roles/) first, falling back to legacyRoot
// (.kbz/context/roles/) for backward compatibility (NFR-004).
type RoleStore struct {
	newRoot    string // .kbz/roles/
	legacyRoot string // .kbz/context/roles/
}

// NewRoleStore creates a RoleStore. newRoot is .kbz/roles/, legacyRoot is
// .kbz/context/roles/. Either path may not exist on disk.
func NewRoleStore(newRoot, legacyRoot string) *RoleStore {
	return &RoleStore{newRoot: newRoot, legacyRoot: legacyRoot}
}

// Load reads and validates a single role by ID.
// It checks the new location first, then falls back to the legacy location.
func (s *RoleStore) Load(id string) (*Role, error) {
	if !idRegexp.MatchString(id) {
		return nil, fmt.Errorf("invalid role id %q: must be lowercase alphanumeric and hyphens, 2-30 chars", id)
	}

	path, err := s.resolve(id)
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read role %q: %w", id, err)
	}

	var role Role
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	if err := dec.Decode(&role); err != nil {
		return nil, fmt.Errorf("parse role %q: %w", id, err)
	}

	if err := validateRole(&role, id); err != nil {
		return nil, err
	}

	return &role, nil
}

// LoadAll reads and validates all roles from both locations.
// Roles in the new location take precedence over legacy roles with the same ID.
// If neither directory exists, it returns an empty slice without error.
//
// The legacy directory (.kbz/context/roles/) is shared with ProfileStore, which
// writes old-format YAML files that do not match the Role struct. LoadAll handles
// this by loading the legacy directory leniently: files whose IDs are already
// present in the new location are skipped without parsing, and files that fail
// to parse or validate are silently skipped rather than hard-failing. Only the
// new location (.kbz/roles/) is loaded strictly.
func (s *RoleStore) LoadAll() ([]*Role, error) {
	seen := make(map[string]bool)
	var roles []*Role

	// Load from new location first (takes precedence). Strict mode: any parse
	// or validation error in the new location is surfaced immediately.
	newRoles, err := s.loadDir(s.newRoot, nil, false)
	if err != nil {
		return nil, err
	}
	for _, r := range newRoles {
		seen[r.ID] = true
		roles = append(roles, r)
	}

	// Load from legacy location leniently. Pass seen so that superseded IDs are
	// skipped before parsing. Old-format files (written by ProfileStore / kbz init)
	// that fail strict parsing are silently skipped rather than crashing the caller.
	legacyRoles, err := s.loadDir(s.legacyRoot, seen, true)
	if err != nil {
		return nil, err
	}
	roles = append(roles, legacyRoles...)

	return roles, nil
}

// RolePath returns the filesystem path for a role by ID.
// It checks the new location first, then falls back to the legacy location.
// Returns an error if the role is not found.
func (s *RoleStore) RolePath(id string) (string, error) {
	return s.resolve(id)
}

// Exists returns true if a role file exists for the given ID in either location.
func (s *RoleStore) Exists(id string) bool {
	if !idRegexp.MatchString(id) {
		return false
	}
	_, err := s.resolve(id)
	return err == nil
}

// resolve returns the filesystem path to the role file for the given ID.
// New location takes precedence over legacy.
func (s *RoleStore) resolve(id string) (string, error) {
	filename := id + ".yaml"

	newPath := filepath.Join(s.newRoot, filename)
	if _, err := os.Stat(newPath); err == nil {
		return newPath, nil
	}

	legacyPath := filepath.Join(s.legacyRoot, filename)
	if _, err := os.Stat(legacyPath); err == nil {
		return legacyPath, nil
	}

	return "", fmt.Errorf("role %q not found", id)
}

// loadDir reads and validates all role YAML files from the given directory.
// Returns an empty slice without error if the directory does not exist.
//
// skip is an optional set of role IDs to skip before parsing (may be nil).
// Files whose stem matches a key in skip are ignored without reading or parsing.
//
// lenient controls error handling for individual files. When false (strict mode),
// any parse or validation failure returns an error immediately. When true (lenient
// mode), files that fail to parse or validate are silently skipped — this is
// appropriate for the legacy directory where old-format ProfileStore files may
// coexist alongside new-format Role files.
func (s *RoleStore) loadDir(dir string, skip map[string]bool, lenient bool) ([]*Role, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read roles directory %q: %w", dir, err)
	}

	var roles []*Role
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasSuffix(name, ".yaml") {
			continue
		}
		id := strings.TrimSuffix(name, ".yaml")

		// Skip IDs already loaded from a higher-priority location.
		if skip[id] {
			continue
		}

		data, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			if lenient {
				continue
			}
			return nil, fmt.Errorf("read role file %q: %w", name, err)
		}

		var role Role
		dec := yaml.NewDecoder(bytes.NewReader(data))
		dec.KnownFields(true)
		if err := dec.Decode(&role); err != nil {
			if lenient {
				// Old-format files (e.g. ProfileStore files with description/conventions)
				// coexist in the legacy directory. Silently skip rather than hard-fail.
				continue
			}
			return nil, fmt.Errorf("parse role %q: %w", id, err)
		}

		if err := validateRole(&role, id); err != nil {
			if lenient {
				continue
			}
			return nil, err
		}

		roles = append(roles, &role)
	}

	return roles, nil
}
