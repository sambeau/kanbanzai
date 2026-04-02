package gate

import (
	"testing"

	"github.com/sambeau/kanbanzai/internal/binding"
	"github.com/sambeau/kanbanzai/internal/model"
)

type mockEntitySvc struct {
	listFn func(entityType string) ([]EntityResult, error)
}

func (m *mockEntitySvc) List(et string) ([]EntityResult, error) { return m.listFn(et) }

func intPtr(n int) *int    { return &n }
func boolPtr(b bool) *bool { return &b }

func TestEvalTasks_NilPrereqs(t *testing.T) {
	ctx := PrereqEvalContext{
		Feature:   &model.Feature{ID: "FEAT-001"},
		EntitySvc: &mockEntitySvc{listFn: func(string) ([]EntityResult, error) { return nil, nil }},
	}
	results := evalTasks(&binding.Prerequisites{Tasks: nil}, "developing", ctx)
	if results != nil {
		t.Fatalf("expected nil results for nil Tasks prereq, got %v", results)
	}
}

func TestEvalTasks_MinCountSatisfied(t *testing.T) {
	ctx := PrereqEvalContext{
		Feature: &model.Feature{ID: "FEAT-001"},
		EntitySvc: &mockEntitySvc{listFn: func(string) ([]EntityResult, error) {
			return []EntityResult{
				{ID: "TASK-001", State: map[string]any{"parent_feature": "FEAT-001", "status": "ready"}},
			}, nil
		}},
	}
	prereqs := &binding.Prerequisites{
		Tasks: &binding.TaskPrereq{MinCount: intPtr(1)},
	}
	results := evalTasks(prereqs, "developing", ctx)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if !results[0].Satisfied {
		t.Errorf("expected satisfied, got reason: %s", results[0].Reason)
	}
}

func TestEvalTasks_MinCountNotSatisfiedZeroTasks(t *testing.T) {
	ctx := PrereqEvalContext{
		Feature: &model.Feature{ID: "FEAT-001"},
		EntitySvc: &mockEntitySvc{listFn: func(string) ([]EntityResult, error) {
			return []EntityResult{}, nil
		}},
	}
	prereqs := &binding.Prerequisites{
		Tasks: &binding.TaskPrereq{MinCount: intPtr(1)},
	}
	results := evalTasks(prereqs, "developing", ctx)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Satisfied {
		t.Error("expected not satisfied for zero child tasks with min_count=1")
	}
}

func TestEvalTasks_MinCountNotSatisfiedTooFew(t *testing.T) {
	ctx := PrereqEvalContext{
		Feature: &model.Feature{ID: "FEAT-001"},
		EntitySvc: &mockEntitySvc{listFn: func(string) ([]EntityResult, error) {
			return []EntityResult{
				{ID: "TASK-001", State: map[string]any{"parent_feature": "FEAT-001", "status": "ready"}},
				{ID: "TASK-002", State: map[string]any{"parent_feature": "FEAT-001", "status": "active"}},
			}, nil
		}},
	}
	prereqs := &binding.Prerequisites{
		Tasks: &binding.TaskPrereq{MinCount: intPtr(3)},
	}
	results := evalTasks(prereqs, "developing", ctx)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Satisfied {
		t.Error("expected not satisfied for 2 child tasks with min_count=3")
	}
}

func TestEvalTasks_MinCountFiltersByFeature(t *testing.T) {
	ctx := PrereqEvalContext{
		Feature: &model.Feature{ID: "FEAT-001"},
		EntitySvc: &mockEntitySvc{listFn: func(string) ([]EntityResult, error) {
			return []EntityResult{
				{ID: "TASK-001", State: map[string]any{"parent_feature": "FEAT-001", "status": "ready"}},
				{ID: "TASK-002", State: map[string]any{"parent_feature": "FEAT-999", "status": "ready"}},
			}, nil
		}},
	}
	prereqs := &binding.Prerequisites{
		Tasks: &binding.TaskPrereq{MinCount: intPtr(2)},
	}
	results := evalTasks(prereqs, "developing", ctx)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Satisfied {
		t.Error("expected not satisfied: only 1 task belongs to FEAT-001")
	}
}

func TestEvalTasks_AllTerminalSatisfied(t *testing.T) {
	ctx := PrereqEvalContext{
		Feature: &model.Feature{ID: "FEAT-001"},
		EntitySvc: &mockEntitySvc{listFn: func(string) ([]EntityResult, error) {
			return []EntityResult{
				{ID: "TASK-001", State: map[string]any{"parent_feature": "FEAT-001", "status": "done"}},
				{ID: "TASK-002", State: map[string]any{"parent_feature": "FEAT-001", "status": "not-planned"}},
				{ID: "TASK-003", State: map[string]any{"parent_feature": "FEAT-001", "status": "duplicate"}},
			}, nil
		}},
	}
	prereqs := &binding.Prerequisites{
		Tasks: &binding.TaskPrereq{AllTerminal: boolPtr(true)},
	}
	results := evalTasks(prereqs, "reviewing", ctx)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if !results[0].Satisfied {
		t.Errorf("expected satisfied, got reason: %s", results[0].Reason)
	}
}

func TestEvalTasks_AllTerminalNotSatisfied(t *testing.T) {
	ctx := PrereqEvalContext{
		Feature: &model.Feature{ID: "FEAT-001"},
		EntitySvc: &mockEntitySvc{listFn: func(string) ([]EntityResult, error) {
			return []EntityResult{
				{ID: "TASK-001", State: map[string]any{"parent_feature": "FEAT-001", "status": "done"}},
				{ID: "TASK-002", State: map[string]any{"parent_feature": "FEAT-001", "status": "active"}},
			}, nil
		}},
	}
	prereqs := &binding.Prerequisites{
		Tasks: &binding.TaskPrereq{AllTerminal: boolPtr(true)},
	}
	results := evalTasks(prereqs, "reviewing", ctx)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Satisfied {
		t.Error("expected not satisfied with an active task")
	}
}

func TestEvalTasks_AllTerminalNoTasks(t *testing.T) {
	ctx := PrereqEvalContext{
		Feature: &model.Feature{ID: "FEAT-001"},
		EntitySvc: &mockEntitySvc{listFn: func(string) ([]EntityResult, error) {
			return []EntityResult{}, nil
		}},
	}
	prereqs := &binding.Prerequisites{
		Tasks: &binding.TaskPrereq{AllTerminal: boolPtr(true)},
	}
	results := evalTasks(prereqs, "reviewing", ctx)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if !results[0].Satisfied {
		t.Error("expected vacuously true when no child tasks exist")
	}
}
