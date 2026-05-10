package kbzinit

import (
	"io/fs"
	"strings"
	"testing"
)

// TestEmbeddedSkillsAllHaveMarker covers AC-001: every SKILL.md file in the
// embedded skill corpora must contain a "# kanbanzai-managed:" marker line.
// This is the key regression test — a markerless embedded source causes
// `kbz init` to abort on the second run.
func TestEmbeddedSkillsAllHaveMarker(t *testing.T) {
	corpora := []struct {
		name string
		fsys fs.FS
	}{
		{"embeddedSkills", embeddedSkills},
		{"embeddedTaskSkills", embeddedTaskSkills},
	}

	for _, c := range corpora {
		t.Run(c.name, func(t *testing.T) {
			err := fs.WalkDir(c.fsys, ".", func(path string, d fs.DirEntry, err error) error {
				if err != nil {
					return err
				}
				if d.IsDir() || d.Name() != "SKILL.md" {
					return nil
				}
				data, readErr := fs.ReadFile(c.fsys, path)
				if readErr != nil {
					t.Errorf("read %q: %v", path, readErr)
					return nil
				}
				if !strings.Contains(string(data), "# kanbanzai-managed:") {
					t.Errorf("embedded %q is missing '# kanbanzai-managed:' marker — "+
						"add the marker or the installer will abort on the second run", path)
				}
				return nil
			})
			if err != nil {
				t.Fatalf("walk %s: %v", c.name, err)
			}
		})
	}
}

// TestOrchestrateReviewSkillExists covers AC-007: the embedded
// skills/orchestrate-review/SKILL.md must not be deleted.
// It is a dead workflow skill retained to satisfy the constraint in
// P62-F1-spec-init-unblock.md ("Must NOT delete").
func TestOrchestrateReviewSkillExists(t *testing.T) {
	const path = "skills/orchestrate-review/SKILL.md"
	data, err := embeddedSkills.ReadFile(path)
	if err != nil {
		t.Fatalf("%s missing from embedded FS: %v\n"+
			"This file must not be deleted (P62-F1 constraint)", path, err)
	}
	if len(data) == 0 {
		t.Errorf("%s exists in embedded FS but is empty", path)
	}
}
