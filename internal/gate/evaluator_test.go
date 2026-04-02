package gate

import (
	"testing"

	"github.com/sambeau/kanbanzai/internal/binding"
	"github.com/sambeau/kanbanzai/internal/model"
)

// mockEvaluator returns a PrereqEvaluator that produces a fixed set of GateResults.
func mockEvaluator(results []GateResult) PrereqEvaluator {
	return func(prereqs *binding.Prerequisites, stage string, ctx PrereqEvalContext) []GateResult {
		return results
	}
}

// withCleanRegistry replaces the global registry for the duration of a test,
// restoring the original afterwards.
func withCleanRegistry(t *testing.T) {
	t.Helper()
	orig := evaluatorRegistry
	evaluatorRegistry = map[string]PrereqEvaluator{}
	t.Cleanup(func() { evaluatorRegistry = orig })
}

func TestEvaluatePrerequisites_NilPrereqs(t *testing.T) {
	results := EvaluatePrerequisites(nil, "designing", PrereqEvalContext{})
	if results != nil {
		t.Fatalf("expected nil, got %v", results)
	}
}

func TestEvaluatePrerequisites_EmptyPrereqs(t *testing.T) {
	prereqs := &binding.Prerequisites{}
	results := EvaluatePrerequisites(prereqs, "designing", PrereqEvalContext{})
	if len(results) != 0 {
		t.Fatalf("expected no results, got %d", len(results))
	}
}

func TestEvaluatePrerequisites_DocumentsDispatched(t *testing.T) {
	withCleanRegistry(t)

	want := []GateResult{{
		Stage:     "specifying",
		Satisfied: true,
		Reason:    "design document approved",
		Source:    "registry",
	}}
	RegisterEvaluator("documents", mockEvaluator(want))

	prereqs := &binding.Prerequisites{
		Documents: []binding.DocumentPrereq{
			{Type: "design", Status: "approved"},
		},
	}
	ctx := PrereqEvalContext{
		Feature: &model.Feature{ID: "FEAT-001"},
	}

	got := EvaluatePrerequisites(prereqs, "specifying", ctx)
	if len(got) != 1 {
		t.Fatalf("expected 1 result, got %d", len(got))
	}
	if got[0] != want[0] {
		t.Fatalf("expected %+v, got %+v", want[0], got[0])
	}
}

func TestEvaluatePrerequisites_TasksDispatched(t *testing.T) {
	withCleanRegistry(t)

	want := []GateResult{{
		Stage:     "reviewing",
		Satisfied: true,
		Reason:    "all tasks terminal",
		Source:    "registry",
	}}
	RegisterEvaluator("tasks", mockEvaluator(want))

	allTerminal := true
	prereqs := &binding.Prerequisites{
		Tasks: &binding.TaskPrereq{AllTerminal: &allTerminal},
	}
	ctx := PrereqEvalContext{
		Feature: &model.Feature{ID: "FEAT-002"},
	}

	got := EvaluatePrerequisites(prereqs, "reviewing", ctx)
	if len(got) != 1 {
		t.Fatalf("expected 1 result, got %d", len(got))
	}
	if got[0] != want[0] {
		t.Fatalf("expected %+v, got %+v", want[0], got[0])
	}
}

func TestEvaluatePrerequisites_BothDocumentsAndTasks(t *testing.T) {
	withCleanRegistry(t)

	docResult := GateResult{
		Stage:     "reviewing",
		Satisfied: true,
		Reason:    "spec approved",
		Source:    "registry",
	}
	taskResult := GateResult{
		Stage:     "reviewing",
		Satisfied: false,
		Reason:    "2 of 5 tasks done",
		Source:    "registry",
	}
	RegisterEvaluator("documents", mockEvaluator([]GateResult{docResult}))
	RegisterEvaluator("tasks", mockEvaluator([]GateResult{taskResult}))

	allTerminal := true
	prereqs := &binding.Prerequisites{
		Documents: []binding.DocumentPrereq{
			{Type: "specification", Status: "approved"},
		},
		Tasks: &binding.TaskPrereq{AllTerminal: &allTerminal},
	}
	ctx := PrereqEvalContext{
		Feature: &model.Feature{ID: "FEAT-003"},
	}

	got := EvaluatePrerequisites(prereqs, "reviewing", ctx)
	if len(got) != 2 {
		t.Fatalf("expected 2 results, got %d", len(got))
	}
	if got[0] != docResult {
		t.Errorf("result[0]: expected %+v, got %+v", docResult, got[0])
	}
	if got[1] != taskResult {
		t.Errorf("result[1]: expected %+v, got %+v", taskResult, got[1])
	}
}

func TestEvaluatePrerequisites_UnregisteredDocuments(t *testing.T) {
	withCleanRegistry(t)

	prereqs := &binding.Prerequisites{
		Documents: []binding.DocumentPrereq{
			{Type: "design", Status: "approved"},
		},
	}

	got := EvaluatePrerequisites(prereqs, "designing", PrereqEvalContext{})
	if len(got) != 1 {
		t.Fatalf("expected 1 result, got %d", len(got))
	}
	if got[0].Satisfied {
		t.Error("expected Satisfied=false for unregistered evaluator")
	}
	wantReason := `unknown prerequisite type "documents" for stage "designing"`
	if got[0].Reason != wantReason {
		t.Errorf("expected reason %q, got %q", wantReason, got[0].Reason)
	}
	if got[0].Stage != "designing" {
		t.Errorf("expected stage %q, got %q", "designing", got[0].Stage)
	}
}

func TestEvaluatePrerequisites_UnregisteredTasks(t *testing.T) {
	withCleanRegistry(t)

	allTerminal := true
	prereqs := &binding.Prerequisites{
		Tasks: &binding.TaskPrereq{AllTerminal: &allTerminal},
	}

	got := EvaluatePrerequisites(prereqs, "reviewing", PrereqEvalContext{})
	if len(got) != 1 {
		t.Fatalf("expected 1 result, got %d", len(got))
	}
	if got[0].Satisfied {
		t.Error("expected Satisfied=false for unregistered evaluator")
	}
	wantReason := `unknown prerequisite type "tasks" for stage "reviewing"`
	if got[0].Reason != wantReason {
		t.Errorf("expected reason %q, got %q", wantReason, got[0].Reason)
	}
	if got[0].Stage != "reviewing" {
		t.Errorf("expected stage %q, got %q", "reviewing", got[0].Stage)
	}
}
