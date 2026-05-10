package kbzinit

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
)

//go:embed skills
var embeddedSkills embed.FS

// skillNames is the ordered list of skill names embedded in this binary.
var skillNames = []string{
	"agents",
	"design",
	"documents",
	"getting-started",
	"plan-review",
	"planning",
	"review",
	"specification",
	"workflow",
}

// installSkills installs all six kanbanzai skill files into
// <baseDir>/.agents/skills/kanbanzai-<name>/SKILL.md.
// It applies version-aware create/update/skip/conflict logic per spec §7.
func (i *Initializer) installSkills(baseDir string) error {
	for _, name := range skillNames {
		if err := i.installOneSkill(baseDir, name); err != nil {
			return err
		}
	}
	return nil
}

// installOneSkill installs a single skill file, applying the version-aware rules.
func (i *Initializer) installOneSkill(baseDir, name string) error {
	destDir := filepath.Join(baseDir, ".agents", "skills", "kanbanzai-"+name)
	destPath := filepath.Join(destDir, "SKILL.md")

	// Read the embedded source.
	srcPath := "skills/" + name + "/SKILL.md"
	srcData, err := embeddedSkills.ReadFile(srcPath)
	if err != nil {
		return fmt.Errorf("read embedded skill %q: %w", name, err)
	}

	// Transform the frontmatter: inject the proper managed marker and version.
	transformed, err := transformSkillContent(srcData, i.version)
	if err != nil {
		return fmt.Errorf("transform skill %q: %w", name, err)
	}

	// Check whether the destination file already exists.
	existing, readErr := os.ReadFile(destPath)
	if readErr != nil {
		if !os.IsNotExist(readErr) {
			return fmt.Errorf("read existing skill %q: %w", destPath, readErr)
		}
		// File does not exist — create it.
		if err := os.MkdirAll(destDir, 0o755); err != nil {
			return fmt.Errorf("create skill dir %q: %w", destDir, err)
		}
		if err := os.WriteFile(destPath, transformed, 0o644); err != nil {
			return fmt.Errorf("write skill %q: %w", destPath, err)
		}
		fmt.Fprintf(i.stdout, "Created .agents/skills/kanbanzai-%s/SKILL.md\n", name)
		return nil
	}

	// File exists — check for the managed marker.
	if !hasLine(existing, managedMarker) {
		return fmt.Errorf(
			"skill file '%s' exists but is not managed by Kanbanzai (no '%s' marker found in frontmatter). "+
				"Remove or rename the file manually, or re-run with --skip-skills to bypass skill installation",
			destPath, managedMarker,
		)
	}

	// Has managed marker — check version.
	existingVersion := extractVersion(existing)
	// In dev builds, always overwrite — "dev" means "latest from source".
	// In release builds, only overwrite when versions differ.
	if existingVersion == i.version && i.version != "dev" {
		// Already at current version — no-op, do not touch file.
		return nil
	}

	// Version is different, or this is a dev build — overwrite.
	if err := os.WriteFile(destPath, transformed, 0o644); err != nil {
		return fmt.Errorf("update skill %q: %w", destPath, err)
	}
	fmt.Fprintf(i.stdout, "Updated .agents/skills/kanbanzai-%s/SKILL.md\n", name)
	return nil
}
