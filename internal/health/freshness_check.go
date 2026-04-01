package health

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// lastVerifiedHolder is a minimal struct for extracting just the last_verified
// field from role YAML or skill frontmatter.
type lastVerifiedHolder struct {
	LastVerified string `yaml:"last_verified"`
}

// FreshnessSummary holds aggregate counts of fresh, stale, and never-verified
// roles and skills.
type FreshnessSummary struct {
	FreshRoles          int
	StaleRoles          int
	NeverVerifiedRoles  int
	FreshSkills         int
	StaleSkills         int
	NeverVerifiedSkills int
}

// ParseRoleLastVerified reads a role YAML file and extracts the last_verified
// timestamp. Returns zero time with nil error if the field is absent or empty.
func ParseRoleLastVerified(path string) (time.Time, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return time.Time{}, fmt.Errorf("reading role file: %w", err)
	}
	var h lastVerifiedHolder
	if err := yaml.Unmarshal(data, &h); err != nil {
		return time.Time{}, fmt.Errorf("parsing role YAML: %w", err)
	}
	if h.LastVerified == "" {
		return time.Time{}, nil
	}
	t, err := time.Parse(time.RFC3339, h.LastVerified)
	if err != nil {
		return time.Time{}, fmt.Errorf("parsing last_verified %q: %w", h.LastVerified, err)
	}
	return t, nil
}

// ParseSkillLastVerified reads a SKILL.md file and extracts the last_verified
// timestamp from its YAML frontmatter. Returns zero time with nil error if the
// field is absent or empty.
func ParseSkillLastVerified(skillDir string) (time.Time, error) {
	path := filepath.Join(skillDir, "SKILL.md")
	data, err := os.ReadFile(path)
	if err != nil {
		return time.Time{}, fmt.Errorf("reading skill file: %w", err)
	}
	fm, err := extractFrontmatter(string(data))
	if err != nil {
		return time.Time{}, err
	}
	var h lastVerifiedHolder
	if err := yaml.Unmarshal([]byte(fm), &h); err != nil {
		return time.Time{}, fmt.Errorf("parsing skill frontmatter YAML: %w", err)
	}
	if h.LastVerified == "" {
		return time.Time{}, nil
	}
	t, err := time.Parse(time.RFC3339, h.LastVerified)
	if err != nil {
		return time.Time{}, fmt.Errorf("parsing last_verified %q: %w", h.LastVerified, err)
	}
	return t, nil
}

// extractFrontmatter returns the YAML content between the first pair of "---"
// delimiters in a Markdown file.
func extractFrontmatter(content string) (string, error) {
	const delim = "---"
	idx := strings.Index(content, delim)
	if idx == -1 {
		return "", fmt.Errorf("no frontmatter delimiter found")
	}
	rest := content[idx+len(delim):]
	end := strings.Index(rest, delim)
	if end == -1 {
		return "", fmt.Errorf("no closing frontmatter delimiter found")
	}
	return rest[:end], nil
}

// CheckRoleFreshness scans all role YAML files in rolesDir and returns a
// CategoryResult with warnings for stale or never-verified roles.
func CheckRoleFreshness(rolesDir string, window int, now time.Time) CategoryResult {
	result := NewCategoryResult()

	matches, err := filepath.Glob(filepath.Join(rolesDir, "*.yaml"))
	if err != nil || len(matches) == 0 {
		return result
	}

	for _, path := range matches {
		name := strings.TrimSuffix(filepath.Base(path), ".yaml")
		lv, err := ParseRoleLastVerified(path)
		if err != nil {
			result.AddIssue(Issue{
				Severity: SeverityWarning,
				EntityID: name,
				Message:  fmt.Sprintf("role %q: failed to parse last_verified: %v", name, err),
			})
			continue
		}

		detail := ClassifyFreshness(lv, lv.IsZero(), window, now)
		switch detail.Status {
		case StatusStale:
			result.AddIssue(Issue{
				Severity: SeverityWarning,
				EntityID: name,
				Message:  fmt.Sprintf("role %q last verified %s (%d days overdue)", name, lv.Format(time.RFC3339), detail.DaysOverdue),
			})
		case StatusNeverVerified:
			result.AddIssue(Issue{
				Severity: SeverityWarning,
				EntityID: name,
				Message:  fmt.Sprintf("role %q has never been verified", name),
			})
		}
	}

	return result
}

// CheckSkillFreshness scans all skill directories under skillsDir (each
// containing a SKILL.md) and returns a CategoryResult with warnings for stale
// or never-verified skills.
func CheckSkillFreshness(skillsDir string, window int, now time.Time) CategoryResult {
	result := NewCategoryResult()

	matches, err := filepath.Glob(filepath.Join(skillsDir, "*", "SKILL.md"))
	if err != nil || len(matches) == 0 {
		return result
	}

	for _, path := range matches {
		dir := filepath.Dir(path)
		name := filepath.Base(dir)
		lv, err := ParseSkillLastVerified(dir)
		if err != nil {
			result.AddIssue(Issue{
				Severity: SeverityWarning,
				EntityID: name,
				Message:  fmt.Sprintf("skill %q: failed to parse last_verified: %v", name, err),
			})
			continue
		}

		detail := ClassifyFreshness(lv, lv.IsZero(), window, now)
		switch detail.Status {
		case StatusStale:
			result.AddIssue(Issue{
				Severity: SeverityWarning,
				EntityID: name,
				Message:  fmt.Sprintf("skill %q last verified %s (%d days overdue)", name, lv.Format(time.RFC3339), detail.DaysOverdue),
			})
		case StatusNeverVerified:
			result.AddIssue(Issue{
				Severity: SeverityWarning,
				EntityID: name,
				Message:  fmt.Sprintf("skill %q has never been verified", name),
			})
		}
	}

	return result
}

// ComputeFreshnessSummary scans roles and skills directories and returns
// aggregate counts by freshness status.
func ComputeFreshnessSummary(rolesDir, skillsDir string, window int, now time.Time) FreshnessSummary {
	var s FreshnessSummary

	// Roles.
	roleMatches, _ := filepath.Glob(filepath.Join(rolesDir, "*.yaml"))
	for _, path := range roleMatches {
		lv, err := ParseRoleLastVerified(path)
		if err != nil {
			s.NeverVerifiedRoles++
			continue
		}
		detail := ClassifyFreshness(lv, lv.IsZero(), window, now)
		switch detail.Status {
		case StatusFresh:
			s.FreshRoles++
		case StatusStale:
			s.StaleRoles++
		case StatusNeverVerified:
			s.NeverVerifiedRoles++
		}
	}

	// Skills.
	skillMatches, _ := filepath.Glob(filepath.Join(skillsDir, "*", "SKILL.md"))
	for _, path := range skillMatches {
		dir := filepath.Dir(path)
		lv, err := ParseSkillLastVerified(dir)
		if err != nil {
			s.NeverVerifiedSkills++
			continue
		}
		detail := ClassifyFreshness(lv, lv.IsZero(), window, now)
		switch detail.Status {
		case StatusFresh:
			s.FreshSkills++
		case StatusStale:
			s.StaleSkills++
		case StatusNeverVerified:
			s.NeverVerifiedSkills++
		}
	}

	return s
}
