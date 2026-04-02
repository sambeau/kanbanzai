package gate

import (
	"fmt"
	"testing"

	"github.com/sambeau/kanbanzai/internal/binding"
	"github.com/sambeau/kanbanzai/internal/model"
)

type mockDocSvc struct {
	getDoc  func(id string, loadContent bool) (*DocumentRecord, error)
	listDoc func(filters DocumentFilters) ([]*DocumentRecord, error)
}

func (m *mockDocSvc) GetDocument(id string, lc bool) (*DocumentRecord, error) {
	return m.getDoc(id, lc)
}

func (m *mockDocSvc) ListDocuments(f DocumentFilters) ([]*DocumentRecord, error) {
	return m.listDoc(f)
}

func TestEvalDocuments_SatisfiedByFeatureFieldRef(t *testing.T) {
	docSvc := &mockDocSvc{
		getDoc: func(id string, _ bool) (*DocumentRecord, error) {
			if id == "DOC-design-001" {
				return &DocumentRecord{
					ID:     "DOC-design-001",
					Status: "approved",
					Type:   "design",
					Owner:  "FEAT-001",
				}, nil
			}
			return nil, fmt.Errorf("not found")
		},
		listDoc: func(_ DocumentFilters) ([]*DocumentRecord, error) {
			return nil, nil
		},
	}

	feat := &model.Feature{
		ID:     "FEAT-001",
		Parent: "P1-plan",
		Design: "DOC-design-001",
	}

	prereqs := &binding.Prerequisites{
		Documents: []binding.DocumentPrereq{
			{Type: "design", Status: "approved"},
		},
	}

	ctx := PrereqEvalContext{
		Feature: feat,
		DocSvc:  docSvc,
	}

	results := evalDocuments(prereqs, "specifying", ctx)

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if !results[0].Satisfied {
		t.Errorf("expected satisfied, got reason: %s", results[0].Reason)
	}
	if results[0].Stage != "specifying" {
		t.Errorf("expected stage %q, got %q", "specifying", results[0].Stage)
	}
}

func TestEvalDocuments_SatisfiedByFeatureOwnedDoc(t *testing.T) {
	docSvc := &mockDocSvc{
		getDoc: func(_ string, _ bool) (*DocumentRecord, error) {
			return nil, fmt.Errorf("not found")
		},
		listDoc: func(f DocumentFilters) ([]*DocumentRecord, error) {
			if f.Owner == "FEAT-001" && f.Type == "design" && f.Status == "approved" {
				return []*DocumentRecord{
					{ID: "DOC-002", Status: "approved", Type: "design", Owner: "FEAT-001"},
				}, nil
			}
			return nil, nil
		},
	}

	feat := &model.Feature{
		ID:     "FEAT-001",
		Parent: "P1-plan",
		// No Design field reference set
	}

	prereqs := &binding.Prerequisites{
		Documents: []binding.DocumentPrereq{
			{Type: "design", Status: "approved"},
		},
	}

	ctx := PrereqEvalContext{
		Feature: feat,
		DocSvc:  docSvc,
	}

	results := evalDocuments(prereqs, "specifying", ctx)

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if !results[0].Satisfied {
		t.Errorf("expected satisfied, got reason: %s", results[0].Reason)
	}
}

func TestEvalDocuments_SatisfiedByParentPlanDoc(t *testing.T) {
	docSvc := &mockDocSvc{
		getDoc: func(_ string, _ bool) (*DocumentRecord, error) {
			return nil, fmt.Errorf("not found")
		},
		listDoc: func(f DocumentFilters) ([]*DocumentRecord, error) {
			if f.Owner == "P1-plan" && f.Type == "design" && f.Status == "approved" {
				return []*DocumentRecord{
					{ID: "DOC-003", Status: "approved", Type: "design", Owner: "P1-plan"},
				}, nil
			}
			return nil, nil
		},
	}

	feat := &model.Feature{
		ID:     "FEAT-001",
		Parent: "P1-plan",
	}

	prereqs := &binding.Prerequisites{
		Documents: []binding.DocumentPrereq{
			{Type: "design", Status: "approved"},
		},
	}

	ctx := PrereqEvalContext{
		Feature: feat,
		DocSvc:  docSvc,
	}

	results := evalDocuments(prereqs, "specifying", ctx)

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if !results[0].Satisfied {
		t.Errorf("expected satisfied, got reason: %s", results[0].Reason)
	}
}

func TestEvalDocuments_NotSatisfied(t *testing.T) {
	docSvc := &mockDocSvc{
		getDoc: func(_ string, _ bool) (*DocumentRecord, error) {
			return nil, fmt.Errorf("not found")
		},
		listDoc: func(_ DocumentFilters) ([]*DocumentRecord, error) {
			return nil, nil
		},
	}

	feat := &model.Feature{
		ID:     "FEAT-001",
		Parent: "P1-plan",
	}

	prereqs := &binding.Prerequisites{
		Documents: []binding.DocumentPrereq{
			{Type: "design", Status: "approved"},
		},
	}

	ctx := PrereqEvalContext{
		Feature: feat,
		DocSvc:  docSvc,
	}

	results := evalDocuments(prereqs, "specifying", ctx)

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Satisfied {
		t.Error("expected not satisfied")
	}
	if results[0].Reason == "" {
		t.Error("expected a reason for unsatisfied result")
	}
}

func TestEvalDocuments_MultiplePrereqsOneFails(t *testing.T) {
	docSvc := &mockDocSvc{
		getDoc: func(id string, _ bool) (*DocumentRecord, error) {
			if id == "DOC-design-001" {
				return &DocumentRecord{
					ID:     "DOC-design-001",
					Status: "approved",
					Type:   "design",
					Owner:  "FEAT-001",
				}, nil
			}
			return nil, fmt.Errorf("not found")
		},
		listDoc: func(f DocumentFilters) ([]*DocumentRecord, error) {
			// No specification documents anywhere
			return nil, nil
		},
	}

	feat := &model.Feature{
		ID:     "FEAT-001",
		Parent: "P1-plan",
		Design: "DOC-design-001",
		// No Spec field reference
	}

	prereqs := &binding.Prerequisites{
		Documents: []binding.DocumentPrereq{
			{Type: "design", Status: "approved"},
			{Type: "specification", Status: "approved"},
		},
	}

	ctx := PrereqEvalContext{
		Feature: feat,
		DocSvc:  docSvc,
	}

	results := evalDocuments(prereqs, "dev-planning", ctx)

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	var satisfiedCount, unsatisfiedCount int
	for _, r := range results {
		if r.Satisfied {
			satisfiedCount++
		} else {
			unsatisfiedCount++
		}
	}
	if satisfiedCount != 1 {
		t.Errorf("expected 1 satisfied result, got %d", satisfiedCount)
	}
	if unsatisfiedCount != 1 {
		t.Errorf("expected 1 unsatisfied result, got %d", unsatisfiedCount)
	}
}

func TestEvalDocuments_UnknownDocType(t *testing.T) {
	docSvc := &mockDocSvc{
		getDoc: func(_ string, _ bool) (*DocumentRecord, error) {
			return nil, fmt.Errorf("not found")
		},
		listDoc: func(f DocumentFilters) ([]*DocumentRecord, error) {
			if f.Owner == "FEAT-001" && f.Type == "research" && f.Status == "approved" {
				return []*DocumentRecord{
					{ID: "DOC-research-01", Status: "approved", Type: "research", Owner: "FEAT-001"},
				}, nil
			}
			return nil, nil
		},
	}

	feat := &model.Feature{
		ID:     "FEAT-001",
		Parent: "P1-plan",
	}

	prereqs := &binding.Prerequisites{
		Documents: []binding.DocumentPrereq{
			{Type: "research", Status: "approved"},
		},
	}

	ctx := PrereqEvalContext{
		Feature: feat,
		DocSvc:  docSvc,
	}

	results := evalDocuments(prereqs, "specifying", ctx)

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	// "research" has no feature field mapping, so Level 1 is skipped.
	// Level 2 (feature-owned) should find it.
	if !results[0].Satisfied {
		t.Errorf("expected satisfied for unknown doc type via Level 2, got reason: %s", results[0].Reason)
	}
}

func TestEvalDocuments_FieldRefWrongStatus(t *testing.T) {
	// Feature field ref points to a document that exists but has wrong status.
	// Should fall through to Level 2/3.
	docSvc := &mockDocSvc{
		getDoc: func(id string, _ bool) (*DocumentRecord, error) {
			if id == "DOC-design-001" {
				return &DocumentRecord{
					ID:     "DOC-design-001",
					Status: "draft", // not approved
					Type:   "design",
					Owner:  "FEAT-001",
				}, nil
			}
			return nil, fmt.Errorf("not found")
		},
		listDoc: func(_ DocumentFilters) ([]*DocumentRecord, error) {
			return nil, nil
		},
	}

	feat := &model.Feature{
		ID:     "FEAT-001",
		Parent: "P1-plan",
		Design: "DOC-design-001",
	}

	prereqs := &binding.Prerequisites{
		Documents: []binding.DocumentPrereq{
			{Type: "design", Status: "approved"},
		},
	}

	ctx := PrereqEvalContext{
		Feature: feat,
		DocSvc:  docSvc,
	}

	results := evalDocuments(prereqs, "specifying", ctx)

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Satisfied {
		t.Error("expected not satisfied when field ref doc has wrong status and no fallback docs exist")
	}
}

func TestEvalDocuments_NoParentSkipsLevel3(t *testing.T) {
	docSvc := &mockDocSvc{
		getDoc: func(_ string, _ bool) (*DocumentRecord, error) {
			return nil, fmt.Errorf("not found")
		},
		listDoc: func(f DocumentFilters) ([]*DocumentRecord, error) {
			if f.Owner == "" {
				t.Error("ListDocuments should not be called with empty owner for Level 3")
			}
			return nil, nil
		},
	}

	feat := &model.Feature{
		ID: "FEAT-001",
		// No Parent
	}

	prereqs := &binding.Prerequisites{
		Documents: []binding.DocumentPrereq{
			{Type: "design", Status: "approved"},
		},
	}

	ctx := PrereqEvalContext{
		Feature: feat,
		DocSvc:  docSvc,
	}

	results := evalDocuments(prereqs, "specifying", ctx)

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Satisfied {
		t.Error("expected not satisfied when no parent and no docs found")
	}
}

func TestEvalDocuments_NilDocumentsSlice(t *testing.T) {
	prereqs := &binding.Prerequisites{
		Documents: nil,
	}

	ctx := PrereqEvalContext{
		Feature: &model.Feature{ID: "FEAT-001"},
	}

	results := evalDocuments(prereqs, "specifying", ctx)

	if len(results) != 0 {
		t.Fatalf("expected 0 results for nil documents, got %d", len(results))
	}
}
