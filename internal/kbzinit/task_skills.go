package kbzinit

import (
	"embed"
	"fmt"
)

//go:embed skills/task-execution
var embeddedTaskSkills embed.FS

// installTaskSkills installs all TaskSkill artifacts from the Manifest into
// <baseDir>/.kbz/skills/<name>/SKILL.md.
func (i *Initializer) installTaskSkills(baseDir string) error {
	for _, a := range Manifest {
		if a.Kind != TaskSkill {
			continue
		}
		srcData, err := embeddedTaskSkills.ReadFile(a.EmbedPath)
		if err != nil {
			return fmt.Errorf("read embedded task skill %s: %w", a.Name, err)
		}
		// Task-execution skills use the same frontmatter format as workflow skills.
		transformed, err := transformSkillContent(srcData, i.version)
		if err != nil {
			return fmt.Errorf("transform task skill %s: %w", a.Name, err)
		}
		a.Marker.CurrentValue = i.version
		if err := installArtifact(a, transformed, i.stdout, baseDir); err != nil {
			return err
		}
	}
	return nil
}
