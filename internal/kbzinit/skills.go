package kbzinit

import (
	"embed"
	"fmt"
)

//go:embed skills
var embeddedSkills embed.FS

// installSkills installs all WorkflowSkill artifacts from the Manifest into
// <baseDir>/.agents/skills/kanbanzai-<name>/SKILL.md.
func (i *Initializer) installSkills(baseDir string) error {
	for _, a := range Manifest {
		if a.Kind != WorkflowSkill {
			continue
		}
		srcData, err := embeddedSkills.ReadFile(a.EmbedPath)
		if err != nil {
			return fmt.Errorf("read embedded skill %s: %w", a.Name, err)
		}
		transformed, err := transformSkillContent(srcData, i.version)
		if err != nil {
			return fmt.Errorf("transform skill %s: %w", a.Name, err)
		}
		a.Marker.CurrentValue = i.version
		if err := installArtifact(a, transformed, i.stdout, baseDir); err != nil {
			return err
		}
	}
	return nil
}


