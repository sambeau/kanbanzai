package card_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/sambeau/kanbanzai/internal/binding"
	"github.com/sambeau/kanbanzai/internal/card"
	kbzctx "github.com/sambeau/kanbanzai/internal/context"
)

// ── helpers ──────────────────────────────────────────────────────────────────

// roleFixture returns a minimal valid ResolvedRole for testing.
func roleFixture(id, identity string) *kbzctx.ResolvedRole {
	return &kbzctx.ResolvedRole{
		ID:       id,
		Identity: identity,
	}
}

// bindingFixture returns a StageBinding with the given skill names.
func bindingFixture(skills ...string) *binding.StageBinding {
	return &binding.StageBinding{
		Roles:  []string{"implementer-go"},
		Skills: skills,
	}
}

// makeEntries creates n ConstraintEntry values with distinct IDs and rules.
func makeEntries(n int) []card.ConstraintEntry {
	entries := make([]card.ConstraintEntry, n)
	for i := range entries {
		entries[i] = card.ConstraintEntry{
			ID:   fmt.Sprintf("C-%03d", i+1),
			Rule: fmt.Sprintf("Rule number %d: do something specific and actionable.", i+1),
		}
	}
	return entries
}

// nonEmptyLines counts lines with at least one non-whitespace character.
func nonEmptyLines(s string) int {
	count := 0
	for _, line := range strings.Split(s, "\n") {
		if strings.TrimSpace(line) != "" {
			count++
		}
	}
	return count
}

// ── input validation (REQ-007, AC-006) ───────────────────────────────────────

func TestRender_NilRole_Error(t *testing.T) {
	_, err := card.Render(nil, "developing", bindingFixture("implement-task"), nil)
	if err == nil {
		t.Fatal("expected error for nil role, got nil")
	}
	if !strings.Contains(err.Error(), "role") {
		t.Errorf("error must name missing input \"role\", got: %v", err)
	}
}

func TestRender_MissingIdentity_NamesField(t *testing.T) {
	role := roleFixture("implementer-go", "") // empty identity
	_, err := card.Render(role, "developing", bindingFixture("implement-task"), nil)
	if err == nil {
		t.Fatal("expected error for missing identity, got nil")
	}
	if !strings.Contains(err.Error(), "identity") {
		t.Errorf("error must name missing field \"identity\", got: %v", err)
	}
}

// ── unknown stage (REQ-008, AC-007) ──────────────────────────────────────────

func TestRender_NilBinding_UnknownStageWarning(t *testing.T) {
	role := roleFixture("implementer-go", "Senior Go engineer")
	got, err := card.Render(role, "some-unknown-stage", nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(got, "UNKNOWN STAGE") {
		t.Errorf("card must contain \"UNKNOWN STAGE\", got:\n%s", got)
	}
	if !strings.Contains(got, ".kbz/stage-bindings.yaml") {
		t.Errorf("card must mention .kbz/stage-bindings.yaml, got:\n%s", got)
	}
}

func TestRender_NilBinding_ContainsManualLoadInstruction(t *testing.T) {
	role := roleFixture("architect", "Senior software architect")
	got, err := card.Render(role, "", nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(got, "manually") {
		t.Errorf("card must instruct agent to load manually, got:\n%s", got)
	}
}

// ── normal render (AC-001, AC-002) ───────────────────────────────────────────

func TestRender_ContainsRoleIdentity(t *testing.T) {
	role := roleFixture("implementer-go", "Senior Go engineer")
	got, err := card.Render(role, "developing", bindingFixture("implement-task"), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(got, "implementer-go") {
		t.Errorf("card must contain role ID, got:\n%s", got)
	}
	if !strings.Contains(got, "Senior Go engineer") {
		t.Errorf("card must contain role identity, got:\n%s", got)
	}
}

func TestRender_ContainsStage(t *testing.T) {
	role := roleFixture("implementer-go", "Senior Go engineer")
	got, err := card.Render(role, "developing", bindingFixture("implement-task"), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(got, "developing") {
		t.Errorf("card must contain stage name, got:\n%s", got)
	}
}

func TestRender_ContainsSkills(t *testing.T) {
	role := roleFixture("implementer-go", "Senior Go engineer")
	b := bindingFixture("implement-task", "orchestrate-development")
	got, err := card.Render(role, "developing", b, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(got, "implement-task") {
		t.Errorf("card must contain skill name, got:\n%s", got)
	}
	if !strings.Contains(got, "orchestrate-development") {
		t.Errorf("card must contain second skill name, got:\n%s", got)
	}
}

func TestRender_ContainsConstraints(t *testing.T) {
	role := roleFixture("implementer-go", "Senior Go engineer")
	entries := []card.ConstraintEntry{
		{ID: "C-001", Rule: "Always check git status before starting work."},
		{ID: "C-002", Rule: "Commit and test before marking done."},
	}
	got, err := card.Render(role, "developing", bindingFixture("implement-task"), entries)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(got, "Always check git status before starting work.") {
		t.Errorf("card must contain first constraint rule, got:\n%s", got)
	}
	if !strings.Contains(got, "Commit and test before marking done.") {
		t.Errorf("card must contain second constraint rule, got:\n%s", got)
	}
}

func TestRender_ContainsToolRoutingReminder(t *testing.T) {
	role := roleFixture("implementer-go", "Senior Go engineer")
	got, err := card.Render(role, "developing", bindingFixture("implement-task"), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(got, "handoff") {
		t.Errorf("card must contain tool-routing reminder mentioning handoff, got:\n%s", got)
	}
}

// TestRender_DevelopingImplementer exercises the full developing/implementer-go
// card to confirm it contains all required elements (AC-002).
func TestRender_DevelopingImplementer(t *testing.T) {
	role := roleFixture("implementer-go", "Senior Go engineer")
	b := &binding.StageBinding{
		Roles:  []string{"implementer-go"},
		Skills: []string{"implement-task"},
	}
	entries := []card.ConstraintEntry{
		{ID: "C-DEV-001", Rule: "Use kanbanzai_edit_file or write_file with entity_id for worktree writes."},
		{ID: "C-DEV-002", Rule: "Check git status before starting. Commit or stash prior work first."},
	}

	got, err := card.Render(role, "developing", b, entries)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	checks := []struct {
		desc string
		want string
	}{
		{"role identity", "implementer-go"},
		{"role description", "Senior Go engineer"},
		{"stage name", "developing"},
		{"skill name", "implement-task"},
		{"first constraint", "kanbanzai_edit_file"},
		{"second constraint", "git status"},
		{"tool-routing reminder", "handoff"},
	}
	for _, c := range checks {
		if !strings.Contains(got, c.want) {
			t.Errorf("developing card missing %s (%q); card:\n%s", c.desc, c.want, got)
		}
	}
}

// TestRender_GeneratedFromTypedInputs confirms the card comes entirely from
// the typed inputs, not from hand-written fixtures (AC-001).
func TestRender_GeneratedFromTypedInputs(t *testing.T) {
	role := roleFixture("test-role", "A uniquely identifiable identity string X7K9")
	b := bindingFixture("a-unique-skill-name-Q3Z8")
	entries := []card.ConstraintEntry{
		{ID: "C-UNIQUE", Rule: "A uniquely identifiable constraint rule R5J2."},
	}
	got, err := card.Render(role, "test-stage-W1M4", b, entries)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(got, "X7K9") {
		t.Errorf("card does not contain identity token from typed input")
	}
	if !strings.Contains(got, "Q3Z8") {
		t.Errorf("card does not contain skill token from typed input")
	}
	if !strings.Contains(got, "R5J2") {
		t.Errorf("card does not contain constraint token from typed input")
	}
	if !strings.Contains(got, "W1M4") {
		t.Errorf("card does not contain stage token from typed input")
	}
}

// ── size enforcement (REQ-NF-001, REQ-NF-002, AC-010) ────────────────────────

func TestRender_LineLimitEnforced(t *testing.T) {
	role := roleFixture("implementer-go", "Senior Go engineer")
	// Base card uses ~7 non-empty lines; 19 entries pushes the total over 25.
	entries := makeEntries(19)
	_, err := card.Render(role, "developing", bindingFixture("implement-task"), entries)
	if err == nil {
		t.Fatal("expected error for exceeding non-empty line limit, got nil")
	}
	if !strings.Contains(err.Error(), "non-empty lines") {
		t.Errorf("error must mention non-empty lines, got: %v", err)
	}
}

func TestRender_ByteLimitEnforced(t *testing.T) {
	role := roleFixture("implementer-go", "Senior Go engineer")
	// A single rule of 2500 bytes pushes the card over the 2500-byte budget.
	longRule := strings.Repeat("X", 2500)
	entries := []card.ConstraintEntry{
		{ID: "C-BIG", Rule: longRule},
	}
	_, err := card.Render(role, "developing", bindingFixture("implement-task"), entries)
	if err == nil {
		t.Fatal("expected error for exceeding byte limit, got nil")
	}
	if !strings.Contains(err.Error(), "bytes") {
		t.Errorf("error must mention bytes, got: %v", err)
	}
}

// TestRender_BudgetRespected confirms a normal card stays within limits.
func TestRender_BudgetRespected(t *testing.T) {
	role := roleFixture("implementer-go", "Senior Go engineer")
	entries := makeEntries(5)
	got, err := card.Render(role, "developing", bindingFixture("implement-task"), entries)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n := nonEmptyLines(got); n > 25 {
		t.Errorf("card has %d non-empty lines, want ≤25", n)
	}
	if b := len(got); b > 2500 {
		t.Errorf("card is %d bytes, want ≤2500", b)
	}
}

// ── edge cases ────────────────────────────────────────────────────────────────

func TestRender_NoSkillsInBinding(t *testing.T) {
	role := roleFixture("implementer-go", "Senior Go engineer")
	b := &binding.StageBinding{Roles: []string{"implementer-go"}} // no skills
	got, err := card.Render(role, "developing", b, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(got, "Skills:") {
		t.Errorf("card must not emit Skills line when binding has no skills, got:\n%s", got)
	}
}

func TestRender_NoConstraintEntries(t *testing.T) {
	role := roleFixture("implementer-go", "Senior Go engineer")
	got, err := card.Render(role, "developing", bindingFixture("implement-task"), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(got, "Constraints:") {
		t.Errorf("card must not emit Constraints header when entries is nil, got:\n%s", got)
	}
}

func TestRender_RoleWithNoID(t *testing.T) {
	role := &kbzctx.ResolvedRole{ID: "", Identity: "Anonymous role identity"}
	got, err := card.Render(role, "developing", bindingFixture("implement-task"), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(got, "Anonymous role identity") {
		t.Errorf("card must contain identity even when ID is empty, got:\n%s", got)
	}
}
