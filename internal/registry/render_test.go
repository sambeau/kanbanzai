package registry_test

import (
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sambeau/kanbanzai/internal/registry"
)

var update = flag.Bool("update", false, "update golden files instead of comparing")

// makeRenderModel returns a fixed RegistryModel for golden and determinism tests.
func makeRenderModel() *registry.RegistryModel {
	return &registry.RegistryModel{
		Stages: []registry.StageEntry{
			{
				Name:         "designing",
				Description:  "Creating or revising a design document",
				Roles:        []string{"architect"},
				Skills:       []string{"write-design"},
				HumanGate:    false,
				DocumentType: "design",
				SourcePath:   ".kbz/stage-bindings.yaml",
			},
			{
				Name:          "specifying",
				Description:   "Writing a formal specification",
				Roles:         []string{"spec-author"},
				Skills:        []string{"write-spec"},
				HumanGate:     true,
				DocumentType:  "specification",
				Prerequisites: "design:approved",
				SourcePath:    ".kbz/stage-bindings.yaml",
			},
			{
				Name:          "reviewing",
				Description:   "Evaluating implementation",
				Roles:         []string{"orchestrator", "reviewer-conformance"},
				Skills:        []string{"orchestrate-review", "review-code"},
				HumanGate:     true,
				DocumentType:  "report",
				Prerequisites: "tasks:all-terminal",
				SourcePath:    ".kbz/stage-bindings.yaml",
			},
		},
		Roles: map[string]registry.RoleEntry{
			"architect": {
				ID:         "architect",
				Identity:   "Senior software architect",
				Inherits:   "",
				SourcePath: ".kbz/roles/architect.yaml",
			},
			"orchestrator": {
				ID:         "orchestrator",
				Identity:   "Orchestrating agent",
				Inherits:   "",
				SourcePath: ".kbz/roles/orchestrator.yaml",
			},
			"spec-author": {
				ID:         "spec-author",
				Identity:   "Senior requirements engineer",
				Inherits:   "base",
				SourcePath: ".kbz/roles/spec-author.yaml",
			},
		},
	}
}

func checkGolden(t *testing.T, got, goldenPath string) {
	t.Helper()
	if *update {
		if err := os.MkdirAll(filepath.Dir(goldenPath), 0o755); err != nil {
			t.Fatalf("mkdir golden dir: %v", err)
		}
		if err := os.WriteFile(goldenPath, []byte(got), 0o644); err != nil {
			t.Fatalf("write golden %s: %v", goldenPath, err)
		}
		return
	}
	want, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("read golden %s: %v (run with -update to create)", goldenPath, err)
	}
	if got != string(want) {
		t.Errorf("output mismatch for %s:\ngot:\n%s\nwant:\n%s", goldenPath, got, string(want))
	}
}

// TestRolesAndSkillsContent_Snapshot verifies the roles-and-skills output
// against a golden file (AC-001, AC-002, AC-006, AC-011).
func TestRolesAndSkillsContent_Snapshot(t *testing.T) {
	model := makeRenderModel()
	got := registry.RolesAndSkillsContent(model)
	checkGolden(t, got, filepath.Join("testdata", "golden", "roles-and-skills.golden"))
}

// TestRoleIndexContent_Snapshot verifies the role-index output against a
// golden file (AC-001, AC-002, AC-006).
func TestRoleIndexContent_Snapshot(t *testing.T) {
	model := makeRenderModel()
	got := registry.RoleIndexContent(model)
	checkGolden(t, got, filepath.Join("testdata", "golden", "role-index.golden"))
}

// TestRolesAndSkillsContent_Determinism verifies that RolesAndSkillsContent
// always produces identical bytes for the same input (REQ-NF-003).
func TestRolesAndSkillsContent_Determinism(t *testing.T) {
	model := makeRenderModel()
	first := registry.RolesAndSkillsContent(model)
	for i := 1; i < 5; i++ {
		got := registry.RolesAndSkillsContent(model)
		if got != first {
			t.Errorf("run %d produced different output", i)
		}
	}
}

// TestRoleIndexContent_Determinism verifies that RoleIndexContent always
// produces identical bytes for the same input (REQ-NF-003).
func TestRoleIndexContent_Determinism(t *testing.T) {
	model := makeRenderModel()
	first := registry.RoleIndexContent(model)
	for i := 1; i < 5; i++ {
		got := registry.RoleIndexContent(model)
		if got != first {
			t.Errorf("run %d produced different output", i)
		}
	}
}

// TestRolesAndSkillsContent_NoProcedureBody verifies that the rendered
// roles-and-skills region does not contain full skill procedures, examples,
// or anti-pattern bodies (REQ-NF-004, AC-011).
func TestRolesAndSkillsContent_NoProcedureBody(t *testing.T) {
	model := &registry.RegistryModel{
		Stages: []registry.StageEntry{
			{
				Name:         "designing",
				Description:  "Creating or revising a design document",
				Roles:        []string{"architect"},
				Skills:       []string{"write-design"},
				HumanGate:    false,
				DocumentType: "design",
				SourcePath:   ".kbz/stage-bindings.yaml",
			},
		},
		Roles: map[string]registry.RoleEntry{},
	}

	got := registry.RolesAndSkillsContent(model)

	forbidden := []string{
		"## Procedure",
		"## Examples",
		"anti_patterns:",
		"vocabulary:",
		"## Anti-Patterns",
	}
	for _, f := range forbidden {
		if strings.Contains(got, f) {
			t.Errorf("output contains forbidden content %q", f)
		}
	}
}

// TestRoleIndexContent_NoProcedureBody verifies that the rendered role-index
// region does not include vocabulary lists, anti-pattern bodies, or other role
// detail beyond identity, inherits, and source path (REQ-NF-004, AC-011).
func TestRoleIndexContent_NoProcedureBody(t *testing.T) {
	model := &registry.RegistryModel{
		Stages: []registry.StageEntry{},
		Roles: map[string]registry.RoleEntry{
			"reviewer-security": {
				ID:         "reviewer-security",
				Identity:   "Senior application security engineer",
				Inherits:   "reviewer",
				SourcePath: ".kbz/roles/reviewer-security.yaml",
			},
		},
	}

	got := registry.RoleIndexContent(model)

	// These strings appear only in the full role YAML body, never in metadata.
	forbidden := []string{
		"OWASP",
		"STRIDE",
		"anti_patterns:",
		"vocabulary:",
		"detect:",
		"resolve:",
	}
	for _, f := range forbidden {
		if strings.Contains(got, f) {
			t.Errorf("output contains forbidden content %q", f)
		}
	}
}

// TestRolesAndSkillsContent_StageOrdering verifies that stage rows appear in
// declaration order, not alphabetical order (REQ-NF-003).
func TestRolesAndSkillsContent_StageOrdering(t *testing.T) {
	model := &registry.RegistryModel{
		Stages: []registry.StageEntry{
			{Name: "zzz-stage", Description: "Last alpha, declared first", Roles: []string{}, Skills: []string{}},
			{Name: "aaa-stage", Description: "First alpha, declared last", Roles: []string{}, Skills: []string{}},
		},
		Roles: map[string]registry.RoleEntry{},
	}

	got := registry.RolesAndSkillsContent(model)

	zzIdx := strings.Index(got, "zzz-stage")
	aaIdx := strings.Index(got, "aaa-stage")
	if zzIdx < 0 || aaIdx < 0 {
		t.Fatal("expected both stage names in output")
	}
	if aaIdx < zzIdx {
		t.Errorf("aaa-stage (idx %d) appeared before zzz-stage (idx %d), but zzz-stage was declared first", aaIdx, zzIdx)
	}
}

// TestRoleIndexContent_LexicographicOrder verifies that roles are sorted
// lexicographically by role ID regardless of map iteration order (REQ-NF-003).
func TestRoleIndexContent_LexicographicOrder(t *testing.T) {
	model := &registry.RegistryModel{
		Stages: []registry.StageEntry{},
		Roles: map[string]registry.RoleEntry{
			"zzz-role": {ID: "zzz-role", Identity: "Z", SourcePath: ".kbz/roles/zzz-role.yaml"},
			"aaa-role": {ID: "aaa-role", Identity: "A", SourcePath: ".kbz/roles/aaa-role.yaml"},
			"mmm-role": {ID: "mmm-role", Identity: "M", SourcePath: ".kbz/roles/mmm-role.yaml"},
		},
	}

	got := registry.RoleIndexContent(model)

	aaIdx := strings.Index(got, "aaa-role")
	mmIdx := strings.Index(got, "mmm-role")
	zzIdx := strings.Index(got, "zzz-role")
	if aaIdx < 0 || mmIdx < 0 || zzIdx < 0 {
		t.Fatal("expected all three role names in output")
	}
	if !(aaIdx < mmIdx && mmIdx < zzIdx) {
		t.Errorf("roles not in lexicographic order: aaa@%d mmm@%d zzz@%d", aaIdx, mmIdx, zzIdx)
	}
}

// TestRolesAndSkillsContent_HumanGateFormatting verifies that the Gate column
// shows "auto" for HumanGate=false and "human" for HumanGate=true.
func TestRolesAndSkillsContent_HumanGateFormatting(t *testing.T) {
	model := &registry.RegistryModel{
		Stages: []registry.StageEntry{
			{Name: "auto-stage", Description: "No gate", HumanGate: false, Roles: []string{}, Skills: []string{}},
			{Name: "human-stage", Description: "Human gate", HumanGate: true, Roles: []string{}, Skills: []string{}},
		},
		Roles: map[string]registry.RoleEntry{},
	}

	got := registry.RolesAndSkillsContent(model)

	for _, line := range strings.Split(got, "\n") {
		if strings.Contains(line, "auto-stage") && !strings.Contains(line, "| auto |") {
			t.Errorf("auto-stage row should contain '| auto |' gate column, got: %q", line)
		}
		if strings.Contains(line, "human-stage") && !strings.Contains(line, "| human |") {
			t.Errorf("human-stage row should contain '| human |' gate column, got: %q", line)
		}
	}
}

// TestRolesAndSkillsContent_EmptyDocType verifies that a missing DocumentType
// renders as an em dash rather than an empty cell.
func TestRolesAndSkillsContent_EmptyDocType(t *testing.T) {
	model := &registry.RegistryModel{
		Stages: []registry.StageEntry{
			{Name: "nodoc-stage", Description: "No doc type", HumanGate: false, Roles: []string{}, Skills: []string{}, DocumentType: ""},
		},
		Roles: map[string]registry.RoleEntry{},
	}

	got := registry.RolesAndSkillsContent(model)

	for _, line := range strings.Split(got, "\n") {
		if strings.Contains(line, "nodoc-stage") {
			// Last column before the closing pipe should be an em dash.
			if !strings.HasSuffix(strings.TrimRight(line, " "), "| — |") {
				t.Errorf("empty DocumentType should render as em dash, got row: %q", line)
			}
		}
	}
}

// TestRolesAndSkillsContent_EmptyModel verifies that an empty stage list
// produces the header rows and warning without panicking.
func TestRolesAndSkillsContent_EmptyModel(t *testing.T) {
	model := &registry.RegistryModel{
		Stages: []registry.StageEntry{},
		Roles:  map[string]registry.RoleEntry{},
	}

	got := registry.RolesAndSkillsContent(model)
	if !strings.Contains(got, "| Stage |") {
		t.Error("expected table header line in output")
	}
	if !strings.Contains(got, ".kbz/stage-bindings.yaml") {
		t.Error("expected canonical source reference in output")
	}
}

// TestRoleIndexContent_EmptyModel verifies that an empty role map produces
// the header rows and warning without panicking.
func TestRoleIndexContent_EmptyModel(t *testing.T) {
	model := &registry.RegistryModel{
		Stages: []registry.StageEntry{},
		Roles:  map[string]registry.RoleEntry{},
	}

	got := registry.RoleIndexContent(model)
	if !strings.Contains(got, "| Role |") {
		t.Error("expected table header line in output")
	}
	if !strings.Contains(got, ".kbz/roles/*.yaml") {
		t.Error("expected canonical source reference in output")
	}
}

// TestRolesAndSkillsContent_SkillLinks verifies that skill names are rendered
// as Markdown links pointing to the canonical SKILL.md path (AC-006).
func TestRolesAndSkillsContent_SkillLinks(t *testing.T) {
	model := &registry.RegistryModel{
		Stages: []registry.StageEntry{
			{
				Name:        "designing",
				Description: "Design stage",
				Roles:       []string{"architect"},
				Skills:      []string{"write-design"},
				SourcePath:  ".kbz/stage-bindings.yaml",
			},
		},
		Roles: map[string]registry.RoleEntry{},
	}

	got := registry.RolesAndSkillsContent(model)

	const wantLink = "[write-design](.kbz/skills/write-design/SKILL.md)"
	if !strings.Contains(got, wantLink) {
		t.Errorf("expected skill link %q in output, got:\n%s", wantLink, got)
	}
}

// TestRoleIndexContent_SourcePaths verifies that the Source column in the
// role-index table points to the role's SourcePath (AC-006).
func TestRoleIndexContent_SourcePaths(t *testing.T) {
	model := &registry.RegistryModel{
		Stages: []registry.StageEntry{},
		Roles: map[string]registry.RoleEntry{
			"architect": {
				ID:         "architect",
				Identity:   "Senior software architect",
				SourcePath: ".kbz/roles/architect.yaml",
			},
		},
	}

	got := registry.RoleIndexContent(model)

	const wantPath = "`.kbz/roles/architect.yaml`"
	if !strings.Contains(got, wantPath) {
		t.Errorf("expected source path %q in output, got:\n%s", wantPath, got)
	}
}

// TestRoleIndexContent_InheritsFormatting verifies that the Inherits column
// shows a backtick-quoted parent ID when set, and an em dash when absent.
func TestRoleIndexContent_InheritsFormatting(t *testing.T) {
	model := &registry.RegistryModel{
		Stages: []registry.StageEntry{},
		Roles: map[string]registry.RoleEntry{
			"child": {ID: "child", Identity: "Child role", Inherits: "parent", SourcePath: ".kbz/roles/child.yaml"},
			"root":  {ID: "root", Identity: "Root role", Inherits: "", SourcePath: ".kbz/roles/root.yaml"},
		},
	}

	got := registry.RoleIndexContent(model)

	for _, line := range strings.Split(got, "\n") {
		if strings.Contains(line, "| `child`") {
			if !strings.Contains(line, "| `parent` |") {
				t.Errorf("child role should show parent in Inherits column, got: %q", line)
			}
		}
		if strings.Contains(line, "| `root`") {
			if !strings.Contains(line, "| — |") {
				t.Errorf("root role should show em dash in Inherits column, got: %q", line)
			}
		}
	}
}
