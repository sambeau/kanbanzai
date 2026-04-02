package actionlog

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadScenario_Valid(t *testing.T) {
	t.Parallel()

	yaml := `name: "Test scenario"
description: "Tests something useful"
category: happy-path
starting_state:
  feature_stage: proposed
  documents: []
expected_pattern:
  tool_sequence: [entity, doc]
success_criteria:
  - "Feature reaches reviewing stage"
`
	f := writeTempYAML(t, yaml)
	s, err := LoadScenario(f)
	if err != nil {
		t.Fatalf("LoadScenario: %v", err)
	}
	if s.Name != "Test scenario" {
		t.Errorf("Name: got %q, want %q", s.Name, "Test scenario")
	}
	if s.Category != "happy-path" {
		t.Errorf("Category: got %q, want %q", s.Category, "happy-path")
	}
	if len(s.SuccessCriteria) != 1 {
		t.Errorf("SuccessCriteria len: got %d, want 1", len(s.SuccessCriteria))
	}
}

func TestLoadScenario_MissingName(t *testing.T) {
	t.Parallel()

	yaml := `description: "Tests something"
category: happy-path
success_criteria: ["criterion"]
`
	f := writeTempYAML(t, yaml)
	_, err := LoadScenario(f)
	if err == nil {
		t.Error("expected error for missing name, got nil")
	}
}

func TestLoadScenario_InvalidCategory(t *testing.T) {
	t.Parallel()

	yaml := `name: "Test"
description: "Tests something"
category: unknown-category
success_criteria: ["criterion"]
`
	f := writeTempYAML(t, yaml)
	_, err := LoadScenario(f)
	if err == nil {
		t.Error("expected error for invalid category, got nil")
	}
}

func TestLoadScenario_AllCategories(t *testing.T) {
	t.Parallel()

	categories := []string{
		"happy-path",
		"gate-failure-recovery",
		"review-rework-loop",
		"multi-feature-orchestration",
		"edge-case",
	}

	for _, cat := range categories {
		cat := cat
		t.Run(cat, func(t *testing.T) {
			t.Parallel()
			yaml := "name: Test\ndescription: Desc\ncategory: " + cat + "\nstarting_state:\n  feature_stage: proposed\nexpected_pattern:\n  tool_sequence: [entity]\nsuccess_criteria: [x]\n"
			f := writeTempYAML(t, yaml)
			s, err := LoadScenario(f)
			if err != nil {
				t.Fatalf("LoadScenario(%q): %v", cat, err)
			}
			if s.Category != cat {
				t.Errorf("Category: got %q, want %q", s.Category, cat)
			}
		})
	}
}

func TestLoadScenario_FileNotFound(t *testing.T) {
	t.Parallel()

	_, err := LoadScenario("/nonexistent/path/scenario.yaml")
	if err == nil {
		t.Error("expected error for missing file, got nil")
	}
}

func writeTempYAML(t *testing.T, content string) string {
	t.Helper()
	f := filepath.Join(t.TempDir(), "scenario.yaml")
	if err := os.WriteFile(f, []byte(content), 0o644); err != nil {
		t.Fatalf("writeTempYAML: %v", err)
	}
	return f
}
