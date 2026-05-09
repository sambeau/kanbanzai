// Package card_test — golden file tests for the constraint card renderer.
//
// Golden tests render constraint cards from fixture inputs and assert exact
// byte-level match against files in testdata/golden/. Use -update to regenerate:
//
//	go test ./internal/card/ -run Golden -args -update
package card_test

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sambeau/kanbanzai/internal/binding"
	"github.com/sambeau/kanbanzai/internal/card"
	kbzctx "github.com/sambeau/kanbanzai/internal/context"
	"gopkg.in/yaml.v3"
)

var updateGolden = flag.Bool("update", false, "update golden files in testdata/golden/")

// ── golden fixture types ─────────────────────────────────────────────────────

type goldenCase struct {
	name    string
	role    *kbzctx.ResolvedRole
	stage   string
	binding *binding.StageBinding
	entries []card.ConstraintEntry
}

func developingGolden() goldenCase {
	return goldenCase{
		name: "developing",
		role: &kbzctx.ResolvedRole{
			ID:       "implementer-go",
			Identity: "Senior Go engineer",
		},
		stage: "developing",
		binding: &binding.StageBinding{
			Roles:  []string{"implementer-go"},
			Skills: []string{"implement-task"},
		},
		entries: []card.ConstraintEntry{
			{
				ID:   "C-DEV-001",
				Rule: "Use `kanbanzai_edit_file` or `write_file` with entity_id for all writes inside a worktree.",
			},
			{
				ID:   "C-DEV-002",
				Rule: "Check `git status` before starting. Commit or stash prior work first.",
			},
			{
				ID:   "C-DEV-003",
				Rule: "Run `go test ./...` after implementation. File a BUG entity for any failure.",
			},
		},
	}
}

func specifyingGolden() goldenCase {
	return goldenCase{
		name: "specifying",
		role: &kbzctx.ResolvedRole{
			ID:       "spec-author",
			Identity: "Specification author",
		},
		stage: "specifying",
		binding: &binding.StageBinding{
			Roles:  []string{"spec-author"},
			Skills: []string{"write-spec"},
		},
		entries: []card.ConstraintEntry{
			{
				ID:   "C-SPEC-001",
				Rule: "Every acceptance criterion must reference at least one functional requirement.",
			},
			{
				ID:   "C-SPEC-002",
				Rule: "Acceptance criteria must be verifiable by automated test or manual procedure.",
			},
			{
				ID:   "C-SPEC-003",
				Rule: "Approved design document must exist before writing the spec.",
			},
			{
				ID:   "C-SPEC-004",
				Rule: "Include all five required template sections: Overview, Scope, FRs, NFRs, ACs.",
			},
		},
	}
}

func devPlanningGolden() goldenCase {
	return goldenCase{
		name: "dev-planning",
		role: &kbzctx.ResolvedRole{
			ID:       "architect",
			Identity: "Senior software architect",
		},
		stage: "dev-planning",
		binding: &binding.StageBinding{
			Roles:  []string{"architect"},
			Skills: []string{"write-dev-plan", "decompose-feature"},
		},
		entries: []card.ConstraintEntry{
			{
				ID:   "C-PLAN-001",
				Rule: "Decompose into 8–15 tasks. Fewer than 5 signals insufficient breakdown.",
			},
			{
				ID:   "C-PLAN-002",
				Rule: "Each task must be independently testable.",
			},
			{
				ID:   "C-PLAN-003",
				Rule: "Every task must have a story-point estimate on the Modified Fibonacci scale.",
			},
			{
				ID:   "C-PLAN-004",
				Rule: "Include a dependency graph and identify parallel execution waves.",
			},
			{
				ID:   "C-PLAN-005",
				Rule: "Approved specification must exist before writing the dev-plan.",
			},
		},
	}
}

func reviewingGolden() goldenCase {
	return goldenCase{
		name: "reviewing",
		role: &kbzctx.ResolvedRole{
			ID:       "reviewer-conformance",
			Identity: "Conformance reviewer",
		},
		stage: "reviewing",
		binding: &binding.StageBinding{
			Roles:  []string{"reviewer-conformance"},
			Skills: []string{"review-code"},
		},
		entries: []card.ConstraintEntry{
			{
				ID:   "C-REV-001",
				Rule: "Check each acceptance criterion individually; do not aggregate.",
			},
			{
				ID:   "C-REV-002",
				Rule: "Blocking findings must be resolved before DoD passes.",
			},
		},
	}
}

func unknownStageGolden() goldenCase {
	return goldenCase{
		name: "unknown-stage",
		role: &kbzctx.ResolvedRole{
			ID:       "implementer-go",
			Identity: "Senior Go engineer",
		},
		stage:   "nonexistent-stage",
		binding: nil,
		entries: nil,
	}
}

func allGoldenCases() []goldenCase {
	return []goldenCase{
		developingGolden(),
		specifyingGolden(),
		devPlanningGolden(),
		reviewingGolden(),
		unknownStageGolden(),
	}
}

// ── golden file helpers ──────────────────────────────────────────────────────

func goldenPath(t *testing.T, name string) string {
	t.Helper()
	return filepath.Join("testdata", "golden", name+".golden")
}

// renderGolden renders a golden case and returns the card string.
func renderGolden(t *testing.T, gc goldenCase) string {
	t.Helper()
	got, err := card.Render(gc.role, gc.stage, gc.binding, gc.entries)
	if err != nil {
		t.Fatalf("Render(%s): unexpected error: %v", gc.name, err)
	}
	return got
}

// ── golden tests (REQ-009, AC-006) ───────────────────────────────────────────

func TestGolden_AllStages(t *testing.T) {
	for _, gc := range allGoldenCases() {
		t.Run(gc.name, func(t *testing.T) {
			got := renderGolden(t, gc)
			path := goldenPath(t, gc.name)

			if *updateGolden {
				if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
					t.Fatalf("mkdir golden dir: %v", err)
				}
				if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
					t.Fatalf("write golden: %v", err)
				}
				return
			}

			want, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read golden file %s: %v\nRun with -update to regenerate golden files.", path, err)
			}

			if string(want) != got {
				t.Errorf("golden mismatch for %s:\n--- want:\n%s\n--- got:\n%s\n---\nRun with -update to regenerate.", gc.name, string(want), got)
			}
		})
	}
}

// ── size enforcement across production YAML (AC-008, AC-010) ─────────────────

func TestSizeEnforcement_ProductionRoles(t *testing.T) {
	rolesDir := filepath.Join("..", "..", ".kbz", "roles")
	entries, err := os.ReadDir(rolesDir)
	if err != nil {
		t.Skipf("cannot read roles directory %s: %v", rolesDir, err)
	}

	store := kbzctx.NewRoleStore(rolesDir, rolesDir)

	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".yaml") {
			continue
		}
		roleID := strings.TrimSuffix(e.Name(), ".yaml")

		role, err := kbzctx.ResolveRole(store, roleID)
		if err != nil {
			t.Errorf("ResolveRole(%q): %v", roleID, err)
			continue
		}

		// Render with a minimal binding for each role.
		b := &binding.StageBinding{
			Roles:  []string{roleID},
			Skills: []string{"test-skill"},
		}

		got, renderErr := card.Render(role, "developing", b, nil)
		if renderErr != nil {
			t.Errorf("Render(role=%q): %v", roleID, renderErr)
			continue
		}

		assertSizeLimits(t, fmt.Sprintf("role %q card", roleID), got)
	}
}

func TestSizeEnforcement_ProductionStageBindings(t *testing.T) {
	bindingPath := filepath.Join("..", "..", ".kbz", "stage-bindings.yaml")
	if _, err := os.Stat(bindingPath); os.IsNotExist(err) {
		t.Skipf("stage-bindings.yaml not found at %s", bindingPath)
	}

	// Use a lenient loader that tolerates unknown fields (the production
	// file has retro-fixing with profile/tier/modes/verifying that aren't
	// in the StageBinding struct).
	bf := loadBindingFileLenient(t, bindingPath)
	if bf == nil {
		t.Fatal("lenient load returned nil")
	}

	for stageName, sb := range bf.StageBindings {
		// Resolve the first role for this stage binding.
		if len(sb.Roles) == 0 {
			continue
		}
		roleID := sb.Roles[0]

		rolesDir := filepath.Join("..", "..", ".kbz", "roles")
		store := kbzctx.NewRoleStore(rolesDir, rolesDir)
		role, err := kbzctx.ResolveRole(store, roleID)
		if err != nil {
			t.Logf("skipping stage %q: cannot resolve role %q: %v", stageName, roleID, err)
			continue
		}

		got, renderErr := card.Render(role, stageName, sb, nil)
		if renderErr != nil {
			t.Errorf("Render(stage=%q, role=%q): %v", stageName, roleID, renderErr)
			continue
		}

		assertSizeLimits(t, fmt.Sprintf("stage %q binding card", stageName), got)
	}
}

// loadBindingFileLenient reads and decodes a stage-bindings.yaml without
// KnownFields(true), so it tolerates extra fields like profile/tier/modes
// that exist in the production file but aren't in the StageBinding struct.
func loadBindingFileLenient(t *testing.T, path string) *binding.BindingFile {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read binding file: %v", err)
	}
	var bf binding.BindingFile
	if err := yaml.Unmarshal(data, &bf); err != nil {
		t.Fatalf("decode binding file: %v", err)
	}
	return &bf
}

func assertSizeLimits(t *testing.T, label, card string) {
	t.Helper()
	nonEmpty := 0
	for _, line := range strings.Split(card, "\n") {
		if strings.TrimSpace(line) != "" {
			nonEmpty++
		}
	}
	if nonEmpty > 25 {
		t.Errorf("%s: %d non-empty lines (max 25)", label, nonEmpty)
	}
	if len(card) > 2500 {
		t.Errorf("%s: %d bytes (max 2500)", label, len(card))
	}
}

// ── determinism (AC-007, REQ-NF-004) ─────────────────────────────────────────

func TestRender_Determinism_100Iterations(t *testing.T) {
	gc := developingGolden()

	var first string
	for i := 0; i < 100; i++ {
		got, err := card.Render(gc.role, gc.stage, gc.binding, gc.entries)
		if err != nil {
			t.Fatalf("iter %d: unexpected error: %v", i, err)
		}
		if i == 0 {
			first = got
			continue
		}
		if got != first {
			t.Errorf("iter %d: output differs from iteration 0", i)
			t.Logf("first:\n%s", first)
			t.Logf("iter %d:\n%s", i, got)
			t.Fatal("determinism violated — stopping")
		}
	}
}

// ── unknown stage (AC-009) ───────────────────────────────────────────────────

func TestRender_UnknownStage_GoldenPresent(t *testing.T) {
	gc := unknownStageGolden()
	got := renderGolden(t, gc)

	if !strings.Contains(got, "UNKNOWN STAGE") {
		t.Error("unknown-stage card missing UNKNOWN STAGE text")
	}
	if !strings.Contains(got, ".kbz/stage-bindings.yaml") {
		t.Error("unknown-stage card missing manual-load path reference")
	}
}

// ── missing-role validation (AC-010, REQ-007) ────────────────────────────────

func TestRender_MissingIdentity_NamesFieldPrecisely(t *testing.T) {
	tests := []struct {
		name   string
		role   *kbzctx.ResolvedRole
		errKey string
	}{
		{
			name:   "nil role",
			role:   nil,
			errKey: `"role"`,
		},
		{
			name: "empty identity",
			role: &kbzctx.ResolvedRole{
				ID:       "some-role",
				Identity: "",
			},
			errKey: `"identity"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := card.Render(tt.role, "developing", &binding.StageBinding{
				Roles:  []string{"implementer-go"},
				Skills: []string{"implement-task"},
			}, nil)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.errKey) {
				t.Errorf("error must name missing field %s, got: %v", tt.errKey, err)
			}
		})
	}
}
