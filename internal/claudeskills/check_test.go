package claudeskills_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/sambeau/kanbanzai/internal/claudeskills"
)

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot resolve test file path")
	}
	// internal/claudeskills/ is two levels down from the repo root
	return filepath.Join(filepath.Dir(file), "..", "..")
}

// TestClaudeSkillsWrappers validates that all .claude/skills/ wrappers on disk
// match the expected content defined in ExpectedWrappers.
func TestClaudeSkillsWrappers(t *testing.T) {
	skillsDir := filepath.Join(repoRoot(t), ".claude", "skills")
	stale := claudeskills.CheckAll(skillsDir)
	for _, path := range stale {
		t.Errorf("stale or missing wrapper: %s", path)
	}
}

// TestClaudeSkillsDriftDetected is a fixture-based integration test.
// It creates a temporary directory containing a single stale wrapper and
// verifies that CheckAll returns that path (non-zero stale list).
func TestClaudeSkillsDriftDetected(t *testing.T) {
	tmpDir := t.TempDir()
	spec := claudeskills.ExpectedWrappers[0]

	skillDir := filepath.Join(tmpDir, spec.Skill)
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	// Write stale content — wrong description, missing generated marker.
	staleContent := "---\nname: " + spec.Skill + "\ndescription: \"old description\"\n---\n\nThis is stale content.\n"
	path := filepath.Join(skillDir, "SKILL.md")
	if err := os.WriteFile(path, []byte(staleContent), 0644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	stale := claudeskills.CheckAll(tmpDir)
	if len(stale) == 0 {
		t.Fatal("expected stale wrapper to be detected, got none")
	}
	if stale[0] != path {
		t.Errorf("expected stale path %q, got %q", path, stale[0])
	}
}

// TestClaudeSkillsDriftClean verifies that correctly-generated wrappers pass
// the check (exit-zero case).
func TestClaudeSkillsDriftClean(t *testing.T) {
	tmpDir := t.TempDir()

	for _, spec := range claudeskills.ExpectedWrappers {
		skillDir := filepath.Join(tmpDir, spec.Skill)
		if err := os.MkdirAll(skillDir, 0755); err != nil {
			t.Fatalf("mkdir %q: %v", spec.Skill, err)
		}
		path := filepath.Join(skillDir, "SKILL.md")
		if err := os.WriteFile(path, []byte(spec.ExpectedContent()), 0644); err != nil {
			t.Fatalf("write %q: %v", spec.Skill, err)
		}
	}

	stale := claudeskills.CheckAll(tmpDir)
	if len(stale) != 0 {
		t.Errorf("expected no stale wrappers, got: %v", stale)
	}
}
