package kbzinit

import (
	"bufio"
	"bytes"
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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

const managedMarker = "# kanbanzai-managed:"
const versionMarker = "# kanbanzai-version:"

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

// transformSkillContent rewrites the embedded skill content with the proper
// managed marker text and the actual binary version.
//
// The embedded files contain:
//
//	# kanbanzai-managed: true
//	# kanbanzai-version: dev
//
// These are replaced with:
//
//	# kanbanzai-managed: do not edit. Regenerate with: kanbanzai init --update-skills
//	# kanbanzai-version: <version>
func transformSkillContent(src []byte, version string) ([]byte, error) {
	var buf bytes.Buffer
	scanner := bufio.NewScanner(bytes.NewReader(src))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, managedMarker) {
			buf.WriteString("# kanbanzai-managed: do not edit. Regenerate with: kanbanzai init --update-skills\n")
			continue
		}
		if strings.HasPrefix(line, versionMarker) {
			buf.WriteString("# kanbanzai-version: " + version + "\n")
			continue
		}
		buf.WriteString(line)
		buf.WriteByte('\n')
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// hasLine reports whether data contains a line that starts with prefix.
func hasLine(data []byte, prefix string) bool {
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		if strings.HasPrefix(scanner.Text(), prefix) {
			return true
		}
	}
	return false
}

// extractVersion returns the version string from the first line that starts
// with "# kanbanzai-version:". Returns empty string if not found.
func extractVersion(data []byte) string {
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, versionMarker) {
			v := strings.TrimPrefix(line, versionMarker)
			return strings.TrimSpace(v)
		}
	}
	return ""
}
