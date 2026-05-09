package registry_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sambeau/kanbanzai/internal/registry"
)

// makeFixture creates a minimal fixture directory under a temp dir with the
// given stage-bindings content and a map of role filename -> content.
// It returns the root path. Set stageContent to "" to skip writing the bindings file.
func makeFixture(t *testing.T, stageContent string, roles map[string]string) string {
	t.Helper()
	root := t.TempDir()

	kbzDir := filepath.Join(root, ".kbz")
	if err := os.MkdirAll(kbzDir, 0o755); err != nil {
		t.Fatalf("setup: mkdir .kbz: %v", err)
	}

	if stageContent != "" {
		if err := os.WriteFile(filepath.Join(kbzDir, "stage-bindings.yaml"), []byte(stageContent), 0o644); err != nil {
			t.Fatalf("setup: write stage-bindings.yaml: %v", err)
		}
	}

	if roles != nil {
		rolesDir := filepath.Join(kbzDir, "roles")
		if err := os.MkdirAll(rolesDir, 0o755); err != nil {
			t.Fatalf("setup: mkdir roles: %v", err)
		}
		for name, content := range roles {
			if err := os.WriteFile(filepath.Join(rolesDir, name), []byte(content), 0o644); err != nil {
				t.Fatalf("setup: write role %s: %v", name, err)
			}
		}
	}

	return root
}

const minimalStageBindings = `
stage_bindings:
  designing:
    description: "Creating or revising a design document"
    orchestration: single-agent
    roles: [architect]
    skills: [write-design]
    human_gate: false
    document_type: design

  specifying:
    description: "Writing a formal specification"
    orchestration: single-agent
    roles: [spec-author]
    skills: [write-spec]
    human_gate: true
    document_type: specification
    prerequisites:
      documents:
        - type: design
          status: approved

  developing:
    description: "Implementing tasks from the dev plan"
    orchestration: orchestrator-workers
    roles: [orchestrator]
    skills: [orchestrate-development]
    human_gate: false
    prerequisites:
      documents:
        - type: dev-plan
          status: approved
      tasks:
        min_count: 1
`

const architectRoleYAML = `
id: architect
inherits: base
identity: "Senior software architect"
`

const specAuthorRoleYAML = `
id: spec-author
inherits: base
identity: "Senior requirements engineer"
`

const orchestratorRoleYAML = `
id: orchestrator
inherits: base
identity: "Orchestrating agent"
`

// TestExtract_NormalLoad verifies that a well-formed fixture produces the
// expected stages and roles (AC-001).
func TestExtract_NormalLoad(t *testing.T) {
	root := makeFixture(t, minimalStageBindings, map[string]string{
		"architect.yaml":   architectRoleYAML,
		"spec-author.yaml": specAuthorRoleYAML,
		"orchestrator.yaml": orchestratorRoleYAML,
	})

	model, err := registry.Extract(root)
	if err != nil {
		t.Fatalf("Extract returned unexpected error: %v", err)
	}

	// Stages: all three should be present.
	if len(model.Stages) != 3 {
		t.Errorf("want 3 stages, got %d", len(model.Stages))
	}

	// Roles: all three should be present.
	if len(model.Roles) != 3 {
		t.Errorf("want 3 roles, got %d", len(model.Roles))
	}

	// Spot-check stage fields.
	if len(model.Stages) >= 1 {
		s := model.Stages[0]
		if s.Name != "designing" {
			t.Errorf("stages[0].Name = %q, want %q", s.Name, "designing")
		}
		if s.Description == "" {
			t.Errorf("stages[0].Description is empty")
		}
		if len(s.Roles) == 0 || s.Roles[0] != "architect" {
			t.Errorf("stages[0].Roles = %v, want [architect]", s.Roles)
		}
		if s.HumanGate {
			t.Errorf("stages[0].HumanGate = true, want false")
		}
		if s.DocumentType != "design" {
			t.Errorf("stages[0].DocumentType = %q, want %q", s.DocumentType, "design")
		}
		if s.SourcePath != ".kbz/stage-bindings.yaml" {
			t.Errorf("stages[0].SourcePath = %q, want %q", s.SourcePath, ".kbz/stage-bindings.yaml")
		}
	}

	// Spot-check stage prerequisites.
	if len(model.Stages) >= 2 {
		s := model.Stages[1]
		if s.Name != "specifying" {
			t.Errorf("stages[1].Name = %q, want %q", s.Name, "specifying")
		}
		if !s.HumanGate {
			t.Errorf("stages[1].HumanGate = false, want true")
		}
		if !strings.Contains(s.Prerequisites, "design:approved") {
			t.Errorf("stages[1].Prerequisites = %q, want to contain %q", s.Prerequisites, "design:approved")
		}
	}

	// Spot-check composite prerequisites on developing stage.
	if len(model.Stages) >= 3 {
		s := model.Stages[2]
		if s.Name != "developing" {
			t.Errorf("stages[2].Name = %q, want %q", s.Name, "developing")
		}
		if !strings.Contains(s.Prerequisites, "dev-plan:approved") {
			t.Errorf("stages[2].Prerequisites = %q, want to contain %q", s.Prerequisites, "dev-plan:approved")
		}
		if !strings.Contains(s.Prerequisites, "tasks:min-1") {
			t.Errorf("stages[2].Prerequisites = %q, want to contain %q", s.Prerequisites, "tasks:min-1")
		}
	}

	// Spot-check role fields.
	arch, ok := model.Roles["architect"]
	if !ok {
		t.Fatalf("roles[architect] missing")
	}
	if arch.Identity == "" {
		t.Errorf("roles[architect].Identity is empty")
	}
	if arch.Inherits != "base" {
		t.Errorf("roles[architect].Inherits = %q, want %q", arch.Inherits, "base")
	}
	if arch.SourcePath != ".kbz/roles/architect.yaml" {
		t.Errorf("roles[architect].SourcePath = %q, want %q", arch.SourcePath, ".kbz/roles/architect.yaml")
	}
}

// TestExtract_MissingRolesDir verifies that a missing .kbz/roles directory
// returns an error naming the directory.
func TestExtract_MissingRolesDir(t *testing.T) {
	// Create a fixture with stage-bindings but no roles directory.
	root := t.TempDir()
	kbzDir := filepath.Join(root, ".kbz")
	if err := os.MkdirAll(kbzDir, 0o755); err != nil {
		t.Fatalf("setup: %v", err)
	}
	if err := os.WriteFile(filepath.Join(kbzDir, "stage-bindings.yaml"), []byte(minimalStageBindings), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}
	// Intentionally do NOT create .kbz/roles/

	_, err := registry.Extract(root)
	if err == nil {
		t.Fatal("Extract expected to return an error for missing roles dir, got nil")
	}
	if !strings.Contains(err.Error(), ".kbz/roles") {
		t.Errorf("error %q does not name the missing directory (.kbz/roles)", err.Error())
	}
}

// TestExtract_MalformedStageBindings verifies that a parse failure in
// stage-bindings.yaml returns an error naming the file.
func TestExtract_MalformedStageBindings(t *testing.T) {
	root := makeFixture(t, "stage_bindings: }{invalid yaml{{", map[string]string{
		"architect.yaml": architectRoleYAML,
	})

	_, err := registry.Extract(root)
	if err == nil {
		t.Fatal("Extract expected to return an error for malformed stage-bindings.yaml, got nil")
	}
	if !strings.Contains(err.Error(), ".kbz/stage-bindings.yaml") {
		t.Errorf("error %q does not name the file (.kbz/stage-bindings.yaml)", err.Error())
	}
}

// TestExtract_MalformedRoleFile verifies that a parse failure in a role YAML
// returns an error naming that file.
func TestExtract_MalformedRoleFile(t *testing.T) {
	root := makeFixture(t, minimalStageBindings, map[string]string{
		"architect.yaml": "id: }{invalid yaml{{",
	})

	_, err := registry.Extract(root)
	if err == nil {
		t.Fatal("Extract expected to return an error for malformed role file, got nil")
	}
	if !strings.Contains(err.Error(), ".kbz/roles/architect.yaml") {
		t.Errorf("error %q does not name the file (.kbz/roles/architect.yaml)", err.Error())
	}
}

// TestExtract_EmptyCorpus verifies that an empty stage_bindings mapping and
// an empty roles directory produce a valid model with no entries.
func TestExtract_EmptyCorpus(t *testing.T) {
	const emptyBindings = `stage_bindings: {}`

	root := makeFixture(t, emptyBindings, map[string]string{})
	// map[string]string{} creates the roles directory but writes no files.

	model, err := registry.Extract(root)
	if err != nil {
		t.Fatalf("Extract returned unexpected error: %v", err)
	}
	if len(model.Stages) != 0 {
		t.Errorf("want 0 stages, got %d", len(model.Stages))
	}
	if len(model.Roles) != 0 {
		t.Errorf("want 0 roles, got %d", len(model.Roles))
	}
}

// TestExtract_DeterministicOrdering verifies that:
//   - Stages are returned in declaration order from stage-bindings.yaml
//     (not alphabetical or map-iteration order).
//   - Calling Extract twice on the same input returns identical stage slices.
func TestExtract_DeterministicOrdering(t *testing.T) {
	// Declare stages in reverse-alphabetical order to confirm they are NOT
	// reordered alphabetically.
	const reverseOrderBindings = `
stage_bindings:
  zebra-stage:
    description: "Last alphabetically, declared first"
    orchestration: single-agent
    roles: [orchestrator]
    skills: [orchestrate-development]
    human_gate: false

  mango-stage:
    description: "Middle"
    orchestration: single-agent
    roles: [architect]
    skills: [write-design]
    human_gate: false

  apple-stage:
    description: "First alphabetically, declared last"
    orchestration: single-agent
    roles: [spec-author]
    skills: [write-spec]
    human_gate: true
`

	root := makeFixture(t, reverseOrderBindings, map[string]string{
		"architect.yaml":   architectRoleYAML,
		"spec-author.yaml": specAuthorRoleYAML,
		"orchestrator.yaml": orchestratorRoleYAML,
	})

	wantOrder := []string{"zebra-stage", "mango-stage", "apple-stage"}

	for run := 0; run < 2; run++ {
		model, err := registry.Extract(root)
		if err != nil {
			t.Fatalf("run %d: Extract error: %v", run, err)
		}
		if len(model.Stages) != 3 {
			t.Fatalf("run %d: want 3 stages, got %d", run, len(model.Stages))
		}
		for i, want := range wantOrder {
			if model.Stages[i].Name != want {
				t.Errorf("run %d: stages[%d].Name = %q, want %q", run, i, model.Stages[i].Name, want)
			}
		}
	}
}

// TestExtract_RolesSortedLexicographically verifies that roles are keyed
// correctly when loaded from files that have lexicographic ordering, and that
// the source paths reflect the actual filenames.
func TestExtract_RolesSortedLexicographically(t *testing.T) {
	root := makeFixture(t, minimalStageBindings, map[string]string{
		"zzz-role.yaml": "id: zzz-role\nidentity: Last\n",
		"aaa-role.yaml": "id: aaa-role\nidentity: First\n",
		"mmm-role.yaml": "id: mmm-role\nidentity: Middle\n",
	})

	model, err := registry.Extract(root)
	if err != nil {
		t.Fatalf("Extract error: %v", err)
	}
	if len(model.Roles) != 3 {
		t.Fatalf("want 3 roles, got %d", len(model.Roles))
	}

	// Verify all three roles are present with correct source paths.
	for _, id := range []string{"aaa-role", "mmm-role", "zzz-role"} {
		r, ok := model.Roles[id]
		if !ok {
			t.Errorf("role %q missing from model", id)
			continue
		}
		want := ".kbz/roles/" + id + ".yaml"
		if r.SourcePath != want {
			t.Errorf("roles[%s].SourcePath = %q, want %q", id, r.SourcePath, want)
		}
	}
}

// TestExtract_AllTerminalPrerequisite verifies that an all_terminal task
// prerequisite is summarised as "tasks:all-terminal".
func TestExtract_AllTerminalPrerequisite(t *testing.T) {
	const bindings = `
stage_bindings:
  reviewing:
    description: "Reviewing implementation"
    orchestration: orchestrator-workers
    roles: [reviewer]
    skills: [review-code]
    human_gate: true
    prerequisites:
      tasks:
        all_terminal: true
`
	root := makeFixture(t, bindings, map[string]string{
		"reviewer.yaml": "id: reviewer\nidentity: Reviewer\n",
	})

	model, err := registry.Extract(root)
	if err != nil {
		t.Fatalf("Extract error: %v", err)
	}
	if len(model.Stages) != 1 {
		t.Fatalf("want 1 stage, got %d", len(model.Stages))
	}
	if !strings.Contains(model.Stages[0].Prerequisites, "tasks:all-terminal") {
		t.Errorf("Prerequisites = %q, want to contain %q", model.Stages[0].Prerequisites, "tasks:all-terminal")
	}
}
