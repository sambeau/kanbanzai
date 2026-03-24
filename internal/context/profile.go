package context

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// idRegexp validates profile IDs: lowercase alphanumeric and hyphens, 2-30 chars.
// The first alternative handles 2-30 char IDs that may contain hyphens (but not at start/end).
// The second alternative handles exactly 2-char purely alphanumeric IDs.
var idRegexp = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,28}[a-z0-9]$|^[a-z0-9]{2}$`)

// Profile is a context profile definition as loaded from a YAML file.
type Profile struct {
	ID           string        `yaml:"id"`
	Inherits     string        `yaml:"inherits,omitempty"`
	Description  string        `yaml:"description"`
	Packages     []string      `yaml:"packages,omitempty"`
	Conventions  []string      `yaml:"conventions,omitempty"`
	Architecture *Architecture `yaml:"architecture,omitempty"`
}

// Architecture holds optional architectural context within a Profile.
type Architecture struct {
	Summary       string   `yaml:"summary,omitempty"`
	KeyInterfaces []string `yaml:"key_interfaces,omitempty"`
}

// ResolvedProfile is the result of walking the inheritance chain and applying
// leaf-level replace semantics. The id field comes from the leaf profile;
// inherits is omitted since it belongs to the definition, not the resolved output.
type ResolvedProfile struct {
	ID           string
	Description  string
	Packages     []string
	Conventions  []string
	Architecture *Architecture
}

// ProfileStore reads context profiles from a directory on the filesystem.
type ProfileStore struct {
	root string // path to .kbz/context/roles/
}

// NewProfileStore creates a ProfileStore rooted at the given directory.
func NewProfileStore(root string) *ProfileStore {
	return &ProfileStore{root: root}
}

// Load reads and validates the profile with the given id from the store.
// Returns an error if the file does not exist, is malformed, or fails validation.
func (s *ProfileStore) Load(id string) (*Profile, error) {
	if !idRegexp.MatchString(id) {
		return nil, fmt.Errorf("invalid profile id %q: must be lowercase alphanumeric and hyphens, 2-30 chars", id)
	}

	path := filepath.Join(s.root, id+".yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("profile %q not found", id)
		}
		return nil, fmt.Errorf("read profile %q: %w", id, err)
	}

	var p Profile
	if err := yaml.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("parse profile %q: %w", id, err)
	}

	if err := validateProfile(&p, id); err != nil {
		return nil, err
	}

	return &p, nil
}

// LoadAll reads and validates all profiles in the store directory.
// If the directory does not exist, it returns an empty slice without error.
func (s *ProfileStore) LoadAll() ([]*Profile, error) {
	entries, err := os.ReadDir(s.root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read profiles directory: %w", err)
	}

	var profiles []*Profile
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasSuffix(name, ".yaml") {
			continue
		}
		id := strings.TrimSuffix(name, ".yaml")
		p, err := s.Load(id)
		if err != nil {
			return nil, err
		}
		profiles = append(profiles, p)
	}
	return profiles, nil
}

// validateProfile checks that a parsed profile meets all invariants.
func validateProfile(p *Profile, expectedID string) error {
	if p.ID == "" {
		return fmt.Errorf("profile file %q: missing required field 'id'", expectedID)
	}
	if !idRegexp.MatchString(p.ID) {
		return fmt.Errorf("profile %q: invalid id %q: must be lowercase alphanumeric and hyphens, 2-30 chars", expectedID, p.ID)
	}
	if p.ID != expectedID {
		return fmt.Errorf("profile id %q does not match filename %q", p.ID, expectedID)
	}
	if p.Description == "" {
		return fmt.Errorf("profile %q: missing required field 'description'", p.ID)
	}
	return nil
}
