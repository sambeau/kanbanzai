package kbzinit

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
)

//go:embed skills/task-execution
var embeddedTaskSkills embed.FS

// taskSkillNames is the ordered list of task-execution skill names.
// These are installed to .kbz/skills/ (not .agents/skills/).
var taskSkillNames = []string{
	"audit-codebase",
	"check-docs",
	"copyedit-docs",
	"decompose-feature",
	"edit-docs",
	"implement-task",
	"orchestrate-development",
	"orchestrate-doc-pipeline",
	"orchestrate-review",
	"review-code",
	"review-plan",
	"style-docs",
	"update-docs",
	"write-design",
	"write-dev-plan",
	"write-docs",
	"write-research",
	"write-skill",
	"write-spec",
}

// installTaskSkills installs all task-execution skill files into
// <baseDir>/.kbz/skills/<name>/SKILL.md.
// It applies the same version-aware create/update/skip logic as installSkills.
func (i *Initializer) installTaskSkills(baseDir string) error {
	for _, name := range taskSkillNames {
		if err := i.installOneTaskSkill(baseDir, name); err != nil {
			return err
		}
	}
	return nil
}

// installOneTaskSkill installs a single task-execution skill file.
func (i *Initializer) installOneTaskSkill(baseDir, name string) error {
	destDir := filepath.Join(baseDir, ".kbz", "skills", name)
	destPath := filepath.Join(destDir, "SKILL.md")

	// Read the embedded source.
	srcPath := "skills/task-execution/" + name + "/SKILL.md"
	srcData, err := embeddedTaskSkills.ReadFile(srcPath)
	if err != nil {
		return fmt.Errorf("read embedded task skill %q: %w", name, err)
	}

	// Transform the frontmatter: inject the proper managed marker and version.
	// Reuse the same transformSkillContent from skills.go — task-execution
	// skills use the same frontmatter format.
	transformed, err := transformSkillContent(srcData, i.version)
	if err != nil {
		return fmt.Errorf("transform task skill %q: %w", name, err)
	}

	// Check whether the destination file already exists.
	existing, readErr := os.ReadFile(destPath)
	if readErr != nil {
		if !os.IsNotExist(readErr) {
			return fmt.Errorf("read existing task skill %q: %w", destPath, readErr)
		}
		// File does not exist — create it.
		if err := os.MkdirAll(destDir, 0o755); err != nil {
			return fmt.Errorf("create task skill dir %q: %w", destDir, err)
		}
		if err := os.WriteFile(destPath, transformed, 0o644); err != nil {
			return fmt.Errorf("write task skill %q: %w", destPath, err)
		}
		fmt.Fprintf(i.stdout, "Created .kbz/skills/%s/SKILL.md\n", name)
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
	if existingVersion == i.version && i.version != "dev" {
		return nil
	}

	// Version is different, or this is a dev build — overwrite.
	if err := os.WriteFile(destPath, transformed, 0o644); err != nil {
		return fmt.Errorf("update task skill %q: %w", destPath, err)
	}
	fmt.Fprintf(i.stdout, "Updated .kbz/skills/%s/SKILL.md\n", name)
	return nil
}
