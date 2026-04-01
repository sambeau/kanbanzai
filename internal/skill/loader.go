package skill

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// SkillStore reads skill definitions from the filesystem.
// Skills are stored as directories under the root, each containing a SKILL.md file.
type SkillStore struct {
	root string // path to .kbz/skills/
}

// NewSkillStore creates a SkillStore rooted at the given directory (.kbz/skills/).
func NewSkillStore(root string) *SkillStore {
	return &SkillStore{root: root}
}

// Load reads, parses, and validates a single skill by name.
// The skill directory must be at {root}/{name}/SKILL.md.
// Returns the Skill and any validation warnings. Returns an error if
// the skill has validation errors (warnings alone do not cause an error).
func (s *SkillStore) Load(name string) (*Skill, []ValidationMessage, error) {
	if !skillNameRegexp.MatchString(name) {
		return nil, nil, fmt.Errorf("invalid skill name %q: must be lowercase alphanumeric and hyphens, 2-40 chars", name)
	}

	skillFile := filepath.Join(s.root, name, "SKILL.md")
	data, err := os.ReadFile(skillFile)
	if err != nil {
		return nil, nil, fmt.Errorf("reading skill %q: %w", name, err)
	}

	parsed, parseErrs := parseSKILLMD(data)
	if len(parseErrs) > 0 {
		return nil, nil, fmt.Errorf("parsing skill %q: %w", name, errors.Join(parseErrs...))
	}

	fmErrs := validateFrontmatter(&parsed.Frontmatter, name)
	if len(fmErrs) > 0 {
		return nil, nil, fmt.Errorf("validating skill %q frontmatter: %w", name, errors.Join(fmErrs...))
	}

	sections := parseSections(parsed.BodyRaw)
	sectionMsgs := validateSections(sections, parsed.Frontmatter.ConstraintLevel)

	var warnings []ValidationMessage
	var validationErrors []ValidationMessage
	for _, msg := range sectionMsgs {
		if msg.Level == "error" {
			validationErrors = append(validationErrors, msg)
		} else {
			warnings = append(warnings, msg)
		}
	}

	refPaths := discoverReferences(filepath.Join(s.root, name))
	scriptPaths := discoverScripts(filepath.Join(s.root, name))

	absPath, err := filepath.Abs(skillFile)
	if err != nil {
		return nil, nil, fmt.Errorf("resolving absolute path for skill %q: %w", name, err)
	}

	sk := &Skill{
		Frontmatter:    parsed.Frontmatter,
		Sections:       sections,
		ReferencePaths: refPaths,
		ScriptPaths:    scriptPaths,
		SourcePath:     absPath,
	}

	if len(validationErrors) > 0 {
		var errMsgs []string
		for _, e := range validationErrors {
			errMsgs = append(errMsgs, e.Message)
		}
		return nil, warnings, fmt.Errorf("validating skill %q sections: %s", name, strings.Join(errMsgs, "; "))
	}

	return sk, warnings, nil
}

// LoadAll reads and validates all skills in the root directory.
// If the root directory does not exist, returns an empty slice without error.
func (s *SkillStore) LoadAll() ([]*Skill, error) {
	entries, err := os.ReadDir(s.root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading skills directory: %w", err)
	}

	var skills []*Skill
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		sk, _, err := s.Load(entry.Name())
		if err != nil {
			return nil, fmt.Errorf("loading skill %q: %w", entry.Name(), err)
		}
		skills = append(skills, sk)
	}
	return skills, nil
}

// discoverReferences lists .md files in the skill's references/ subdirectory.
// Returns paths relative to the skill directory. Returns nil if the directory doesn't exist.
func discoverReferences(skillDir string) []string {
	refDir := filepath.Join(skillDir, "references")
	entries, err := os.ReadDir(refDir)
	if err != nil {
		return nil
	}

	var paths []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if strings.ToLower(filepath.Ext(entry.Name())) == ".md" {
			paths = append(paths, filepath.Join("references", entry.Name()))
		}
	}
	return paths
}

// discoverScripts lists all files in the skill's scripts/ subdirectory.
// Returns paths relative to the skill directory. Returns nil if the directory doesn't exist.
func discoverScripts(skillDir string) []string {
	scriptDir := filepath.Join(skillDir, "scripts")
	entries, err := os.ReadDir(scriptDir)
	if err != nil {
		return nil
	}

	var paths []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		paths = append(paths, filepath.Join("scripts", entry.Name()))
	}
	return paths
}
