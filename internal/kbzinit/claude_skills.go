package kbzinit

import (
	"embed"
	"fmt"
)

//go:embed claude_skills
var embeddedClaudeWrappers embed.FS

// installClaudeWrappers installs all ClaudeWrapper artifacts from the Manifest
// into <baseDir>/.claude/skills/<name>/SKILL.md.
//
// REQ-004: The naming convention is encoded in Manifest entries —
// wrappers whose canonical target lives under .agents/skills/kanbanzai-*/ use
// the kanbanzai- prefix; wrappers whose target lives under .kbz/skills/<name>/
// keep the bare name. installOne() uses each entry's InstallPath directly.
func (i *Initializer) installClaudeWrappers(baseDir string) error {
	for _, a := range Manifest {
		if a.Kind != ClaudeWrapper {
			continue
		}
		srcData, err := embeddedClaudeWrappers.ReadFile(a.EmbedPath)
		if err != nil {
			return fmt.Errorf("read embedded claude wrapper %s: %w", a.Name, err)
		}
		transformed, err := transformSkillContent(srcData, i.version)
		if err != nil {
			return fmt.Errorf("transform claude wrapper %s: %w", a.Name, err)
		}
		a.Marker.CurrentValue = i.version
		if err := installArtifact(a, transformed, i.stdout, baseDir); err != nil {
			return err
		}
	}
	return nil
}
