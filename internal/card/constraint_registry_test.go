package card_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sambeau/kanbanzai/internal/card"
)

// writeTempYAML writes content to a temp file and returns the path.
func writeTempYAML(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "constraints.yaml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("writeTempYAML: %v", err)
	}
	return path
}

// ── loading ──────────────────────────────────────────────────────────────────

func TestLoadConstraintRegistry_ValidYAML(t *testing.T) {
	yaml := `constraints:
  - id: C-002
    rule: "second rule"
    applies_to:
      roles: [implementer-go]
      stages: [developing]
  - id: C-001
    rule: "first rule"
    applies_to:
      roles: [spec-author]
      stages: [specifying]
`
	path := writeTempYAML(t, yaml)
	r, err := card.LoadConstraintRegistry(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	entries := r.Entries()
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	// Entries must be sorted by ID after load.
	if entries[0].ID != "C-001" || entries[1].ID != "C-002" {
		t.Errorf("entries not sorted: got [%s, %s], want [C-001, C-002]",
			entries[0].ID, entries[1].ID)
	}
}

func TestLoadConstraintRegistry_EmptyConstraintsList(t *testing.T) {
	path := writeTempYAML(t, "constraints: []\n")
	r, err := card.LoadConstraintRegistry(path)
	if err != nil {
		t.Fatalf("unexpected error for empty list: %v", err)
	}
	if len(r.Entries()) != 0 {
		t.Errorf("expected empty entries")
	}
}

func TestLoadConstraintRegistry_FileNotFound(t *testing.T) {
	_, err := card.LoadConstraintRegistry("nonexistent/path/constraints.yaml")
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}

func TestLoadConstraintRegistry_InvalidYAML(t *testing.T) {
	path := writeTempYAML(t, "constraints: [\n  unclosed bracket\n")
	_, err := card.LoadConstraintRegistry(path)
	if err == nil {
		t.Fatal("expected error for invalid YAML, got nil")
	}
}

// ── validation: required fields ──────────────────────────────────────────────

func TestLoadConstraintRegistry_MissingID(t *testing.T) {
	yaml := `constraints:
  - rule: "some rule"
    applies_to:
      roles: [implementer-go]
      stages: [developing]
`
	path := writeTempYAML(t, yaml)
	_, err := card.LoadConstraintRegistry(path)
	if err == nil {
		t.Fatal("expected error for missing id, got nil")
	}
	if !strings.Contains(err.Error(), `"id"`) {
		t.Errorf(`error must name missing field "id", got: %v`, err)
	}
}

func TestLoadConstraintRegistry_MissingRule(t *testing.T) {
	yaml := `constraints:
  - id: C-001
    applies_to:
      roles: [implementer-go]
      stages: [developing]
`
	path := writeTempYAML(t, yaml)
	_, err := card.LoadConstraintRegistry(path)
	if err == nil {
		t.Fatal("expected error for missing rule, got nil")
	}
	if !strings.Contains(err.Error(), `"rule"`) {
		t.Errorf(`error must name missing field "rule", got: %v`, err)
	}
}

func TestLoadConstraintRegistry_MissingAppliesTo_Roles(t *testing.T) {
	yaml := `constraints:
  - id: C-001
    rule: "a rule"
    applies_to:
      stages: [developing]
`
	path := writeTempYAML(t, yaml)
	_, err := card.LoadConstraintRegistry(path)
	if err == nil {
		t.Fatal("expected error for missing applies_to.roles, got nil")
	}
	if !strings.Contains(err.Error(), "applies_to.roles") {
		t.Errorf("error must name missing field applies_to.roles, got: %v", err)
	}
}

func TestLoadConstraintRegistry_MissingAppliesTo_Stages(t *testing.T) {
	yaml := `constraints:
  - id: C-001
    rule: "a rule"
    applies_to:
      roles: [implementer-go]
`
	path := writeTempYAML(t, yaml)
	_, err := card.LoadConstraintRegistry(path)
	if err == nil {
		t.Fatal("expected error for missing applies_to.stages, got nil")
	}
	if !strings.Contains(err.Error(), "applies_to.stages") {
		t.Errorf("error must name missing field applies_to.stages, got: %v", err)
	}
}

// Second entry failing validation should also be caught.
func TestLoadConstraintRegistry_SecondEntryMissingRule(t *testing.T) {
	yaml := `constraints:
  - id: C-001
    rule: "valid rule"
    applies_to:
      roles: [implementer-go]
      stages: [developing]
  - id: C-002
    applies_to:
      roles: [spec-author]
      stages: [specifying]
`
	path := writeTempYAML(t, yaml)
	_, err := card.LoadConstraintRegistry(path)
	if err == nil {
		t.Fatal("expected error for second entry missing rule, got nil")
	}
	if !strings.Contains(err.Error(), "C-002") {
		t.Errorf("error must mention the entry id C-002, got: %v", err)
	}
}

// ── Select ────────────────────────────────────────────────────────────────────

func TestConstraintRegistry_Select_FiltersRoleAndStage(t *testing.T) {
	yaml := `constraints:
  - id: C-DEV-001
    rule: "rule for implementer-go in developing"
    applies_to:
      roles: [implementer-go]
      stages: [developing]
  - id: C-SPEC-001
    rule: "rule for spec-author in specifying"
    applies_to:
      roles: [spec-author]
      stages: [specifying]
  - id: C-CROSS-001
    rule: "rule for implementer-go in specifying — should not match developing"
    applies_to:
      roles: [implementer-go]
      stages: [specifying]
`
	path := writeTempYAML(t, yaml)
	r, err := card.LoadConstraintRegistry(path)
	if err != nil {
		t.Fatalf("unexpected load error: %v", err)
	}

	got := r.Select("implementer-go", "developing")
	if len(got) != 1 {
		t.Fatalf("Select(implementer-go, developing): expected 1 entry, got %d", len(got))
	}
	if got[0].ID != "C-DEV-001" {
		t.Errorf("expected C-DEV-001, got %s", got[0].ID)
	}
}

func TestConstraintRegistry_Select_MultiRoleEntry(t *testing.T) {
	yaml := `constraints:
  - id: C-001
    rule: "shared rule"
    applies_to:
      roles: [orchestrator, implementer-go]
      stages: [developing]
`
	path := writeTempYAML(t, yaml)
	r, err := card.LoadConstraintRegistry(path)
	if err != nil {
		t.Fatalf("unexpected load error: %v", err)
	}

	if got := r.Select("orchestrator", "developing"); len(got) != 1 {
		t.Errorf("orchestrator should match: got %d entries", len(got))
	}
	if got := r.Select("implementer-go", "developing"); len(got) != 1 {
		t.Errorf("implementer-go should match: got %d entries", len(got))
	}
	if got := r.Select("spec-author", "developing"); len(got) != 0 {
		t.Errorf("spec-author should not match: got %d entries", len(got))
	}
}

func TestConstraintRegistry_Select_NoMatch(t *testing.T) {
	yaml := `constraints:
  - id: C-001
    rule: "a rule"
    applies_to:
      roles: [implementer-go]
      stages: [developing]
`
	path := writeTempYAML(t, yaml)
	r, err := card.LoadConstraintRegistry(path)
	if err != nil {
		t.Fatalf("unexpected load error: %v", err)
	}
	if got := r.Select("architect", "dev-planning"); len(got) != 0 {
		t.Errorf("expected no match, got %d entries", len(got))
	}
}

// ── determinism (REQ-NF-004) ──────────────────────────────────────────────────

func TestConstraintRegistry_Select_Determinism(t *testing.T) {
	// Entries are intentionally written out of order to verify that sorting
	// on load produces stable results regardless of file order.
	yaml := `constraints:
  - id: C-003
    rule: "third rule"
    applies_to:
      roles: [implementer-go]
      stages: [developing]
  - id: C-001
    rule: "first rule"
    applies_to:
      roles: [implementer-go]
      stages: [developing]
  - id: C-002
    rule: "second rule"
    applies_to:
      roles: [implementer-go]
      stages: [developing]
`
	path := writeTempYAML(t, yaml)
	r, err := card.LoadConstraintRegistry(path)
	if err != nil {
		t.Fatalf("unexpected load error: %v", err)
	}

	var firstOrder []string
	for iter := 0; iter < 20; iter++ {
		got := r.Select("implementer-go", "developing")
		if len(got) != 3 {
			t.Fatalf("iter %d: expected 3 entries, got %d", iter, len(got))
		}
		if iter == 0 {
			for _, e := range got {
				firstOrder = append(firstOrder, e.ID)
			}
			continue
		}
		for i, e := range got {
			if e.ID != firstOrder[i] {
				t.Errorf("iter %d: position %d changed: %s != %s", iter, i, e.ID, firstOrder[i])
			}
		}
	}
	// Confirm sorted order: C-001, C-002, C-003.
	want := []string{"C-001", "C-002", "C-003"}
	for i, id := range want {
		if firstOrder[i] != id {
			t.Errorf("sorted position %d: want %s, got %s", i, id, firstOrder[i])
		}
	}
}

// ── production YAML coverage ──────────────────────────────────────────────────

// TestLoadConstraintRegistry_Production loads the actual .kbz/constraints.yaml
// and verifies that each of the four required stage/role pairs has at least one
// entry, satisfying AC-001 (partial) and REQ-001.
func TestLoadConstraintRegistry_Production(t *testing.T) {
	path := filepath.Join("..", "..", ".kbz", "constraints.yaml")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Skipf("production constraints.yaml not found at %s", path)
	}

	r, err := card.LoadConstraintRegistry(path)
	if err != nil {
		t.Fatalf("failed to load production constraints.yaml: %v", err)
	}

	required := []struct {
		role  string
		stage string
	}{
		{"implementer-go", "developing"},
		{"spec-author", "specifying"},
		{"architect", "dev-planning"},
		{"reviewer-conformance", "reviewing"},
	}
	for _, req := range required {
		entries := r.Select(req.role, req.stage)
		if len(entries) == 0 {
			t.Errorf("no constraints for role=%s stage=%s in production constraints.yaml",
				req.role, req.stage)
		}
	}
}
