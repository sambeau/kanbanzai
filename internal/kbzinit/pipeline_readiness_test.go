package kbzinit

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sambeau/kanbanzai/internal/binding"
	"github.com/sambeau/kanbanzai/internal/context"
	"github.com/sambeau/kanbanzai/internal/skill"
)

// TestPipelineReadiness_NewProject verifies that kbz init on a fresh git repo
// produces a project whose artifacts are structurally ready for the 3.0 pipeline.
// Covers: FR-008 (AC-007), FR-010 (AC-009), and portions of FR-001/FR-002/FR-003.
func TestPipelineReadiness_NewProject(t *testing.T) {
	t.Parallel()
	dir := makeGitRepoNoCommits(t)

	in, _ := newTestInit(dir, "")
	if err := in.Run(Options{Name: "test-project"}); err != nil {
		t.Fatalf("Run: %v", err)
	}

	kbzDir := filepath.Join(dir, ".kbz")

	// --- AC-001: stage-bindings.yaml exists and passes validation ---
	bindingPath := filepath.Join(kbzDir, "stage-bindings.yaml")
	if _, err := os.Stat(bindingPath); os.IsNotExist(err) {
		t.Fatal("stage-bindings.yaml not created")
	}
	bf, errs := binding.LoadBindingFile(bindingPath)
	if len(errs) > 0 {
		for _, e := range errs {
			t.Errorf("stage-bindings.yaml validation error: %v", e)
		}
		t.Fatal("stage-bindings.yaml failed validation")
	}
	if bf == nil {
		t.Fatal("stage-bindings.yaml parsed to nil")
	}

	// --- AC-002: all 19 task-execution skills exist on disk ---
	skillsDir := filepath.Join(kbzDir, "skills")
	for _, name := range taskSkillNames {
		skillPath := filepath.Join(skillsDir, name, "SKILL.md")
		if _, err := os.Stat(skillPath); os.IsNotExist(err) {
			t.Errorf("task skill %q not installed: %s", name, skillPath)
		}
	}

	// --- AC-003: all 18 role files exist on disk ---
	rolesDir := filepath.Join(kbzDir, "roles")
	entries, err := os.ReadDir(rolesDir)
	if err != nil {
		t.Fatalf("read roles dir: %v", err)
	}
	roleCount := 0
	for _, e := range entries {
		if !e.IsDir() && filepath.Ext(e.Name()) == ".yaml" {
			roleCount++
		}
	}
	if roleCount != 18 {
		t.Errorf("found %d role files, want 18", roleCount)
	}

	// --- AC-004: AGENTS.md exists with v3 managed marker and pipeline content ---
	agentsPath := filepath.Join(dir, "AGENTS.md")
	agentsData, err := os.ReadFile(agentsPath)
	if err != nil {
		t.Fatalf("AGENTS.md not created: %v", err)
	}
	v, managed, err := readMarkdownManagedVersion(agentsData)
	if err != nil || !managed {
		t.Errorf("AGENTS.md missing managed marker (managed=%v err=%v)", managed, err)
	}
	if v != agentsMDVersion {
		t.Errorf("AGENTS.md version = %d, want %d", v, agentsMDVersion)
	}
	agentsText := string(agentsData)
	for _, want := range []string{"Task-Execution Skills", "Roles", "Stage Bindings", "stage-bindings.yaml"} {
		if !stringsContainsFold(agentsText, want) {
			t.Errorf("AGENTS.md missing expected content: %q", want)
		}
	}

	// --- AC-004-extra: copilot-instructions.md references stage-bindings ---
	copilotPath := filepath.Join(dir, ".github", "copilot-instructions.md")
	copilotData, err := os.ReadFile(copilotPath)
	if err != nil {
		t.Fatalf("copilot-instructions.md not created: %v", err)
	}
	if !stringsContainsFold(string(copilotData), "stage-bindings.yaml") {
		t.Error("copilot-instructions.md missing stage-bindings.yaml reference")
	}

	// --- AC-007/AC-009: pipeline construction prerequisites are in place ---
	// The stage-bindings loaded above.

	// AC-009: SkillStore.LoadAll() returns all 19 skills without error.
	skillStore := skill.NewSkillStore(skillsDir)
	loadedSkills, err := skillStore.LoadAll()
	if err != nil {
		t.Fatalf("SkillStore.LoadAll: %v", err)
	}
	if len(loadedSkills) != len(taskSkillNames) {
		t.Errorf("SkillStore.LoadAll returned %d skills, want %d", len(loadedSkills), len(taskSkillNames))
	}

	// AC-009: RoleStore.LoadAll() returns all 18 roles without error.
	roleStore := context.NewRoleStore(
		filepath.Join(kbzDir, "roles"),
		filepath.Join(kbzDir, "context", "roles"),
	)
	loadedRoles, err := roleStore.LoadAll()
	if err != nil {
		t.Fatalf("RoleStore.LoadAll: %v", err)
	}
	if len(loadedRoles) != 18 {
		t.Errorf("RoleStore.LoadAll returned %d roles, want 18", len(loadedRoles))
	}

	// AC-008: the pipeline would activate (stage-bindings loaded;
	// skills and roles loadable without error). The server log check
	// for "3.0 context assembly pipeline loaded with N stage bindings"
	// requires a running server, but the prerequisites are verified here.
}

// stringsContainsFold is a case-insensitive string contains check.
func stringsContainsFold(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	if len(s) < len(substr) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if stringsEqualFold(s[i:i+len(substr)], substr) {
			return true
		}
	}
	return false
}

func stringsEqualFold(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		ca, cb := a[i], b[i]
		if ca >= 'A' && ca <= 'Z' {
			ca += 'a' - 'A'
		}
		if cb >= 'A' && cb <= 'Z' {
			cb += 'a' - 'A'
		}
		if ca != cb {
			return false
		}
	}
	return true
}
