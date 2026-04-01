package actionlog

import (
	"fmt"
	"testing"
)

func TestExtractEntityID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		args map[string]any
		want *string
	}{
		{
			name: "id field",
			args: map[string]any{"id": "FEAT-001"},
			want: ptr("FEAT-001"),
		},
		{
			name: "entity_id field",
			args: map[string]any{"entity_id": "TASK-001"},
			want: ptr("TASK-001"),
		},
		{
			name: "task_id field",
			args: map[string]any{"task_id": "TASK-002"},
			want: ptr("TASK-002"),
		},
		{
			name: "id takes priority over entity_id",
			args: map[string]any{"id": "FEAT-001", "entity_id": "TASK-001"},
			want: ptr("FEAT-001"),
		},
		{
			name: "no id fields",
			args: map[string]any{"action": "list"},
			want: nil,
		},
		{
			name: "empty args",
			args: map[string]any{},
			want: nil,
		},
		{
			name: "nil args",
			args: nil,
			want: nil,
		},
		{
			name: "empty string id",
			args: map[string]any{"id": ""},
			want: nil,
		},
		{
			name: "non-string id",
			args: map[string]any{"id": 42},
			want: nil,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := ExtractEntityID(tc.args)
			if tc.want == nil {
				if got != nil {
					t.Errorf("got %q, want nil", *got)
				}
				return
			}
			if got == nil {
				t.Errorf("got nil, want %q", *tc.want)
				return
			}
			if *got != *tc.want {
				t.Errorf("got %q, want %q", *got, *tc.want)
			}
		})
	}
}

func TestResolveStage_NilInputs(t *testing.T) {
	t.Parallel()

	if got := ResolveStage(nil, nil); got != nil {
		t.Errorf("ResolveStage(nil, nil) = %v, want nil", *got)
	}

	entityID := "FEAT-001"
	if got := ResolveStage(&entityID, nil); got != nil {
		t.Errorf("ResolveStage(id, nil) = %v, want nil", *got)
	}

	if got := ResolveStage(nil, &stubLookup{}); got != nil {
		t.Errorf("ResolveStage(nil, lookup) = %v, want nil", *got)
	}
}

func TestResolveStage_Feature(t *testing.T) {
	t.Parallel()

	entityID := "FEAT-001"
	lookup := &stubLookup{
		kindMap:  map[string]string{"FEAT-001": "feature"},
		stageMap: map[string]string{"FEAT-001": "developing"},
	}

	got := ResolveStage(&entityID, lookup)
	if got == nil {
		t.Fatal("got nil, want stage")
	}
	if *got != "developing" {
		t.Errorf("got %q, want %q", *got, "developing")
	}
}

func TestResolveStage_Task(t *testing.T) {
	t.Parallel()

	entityID := "TASK-001"
	lookup := &stubLookup{
		kindMap:   map[string]string{"TASK-001": "task"},
		parentMap: map[string]string{"TASK-001": "FEAT-001"},
		stageMap:  map[string]string{"FEAT-001": "developing"},
	}

	got := ResolveStage(&entityID, lookup)
	if got == nil {
		t.Fatal("got nil, want stage")
	}
	if *got != "developing" {
		t.Errorf("got %q, want %q", *got, "developing")
	}
}

func TestResolveStage_LookupError(t *testing.T) {
	t.Parallel()

	entityID := "FEAT-MISSING"
	lookup := &stubLookup{} // no entries

	got := ResolveStage(&entityID, lookup)
	if got != nil {
		t.Errorf("got %q, want nil on error", *got)
	}
}

// stubLookup implements StageLookup for tests.
type stubLookup struct {
	kindMap   map[string]string
	parentMap map[string]string
	stageMap  map[string]string
}

func (s *stubLookup) GetEntityKindAndParent(entityID string) (kind, parent string, err error) {
	if s.kindMap == nil {
		return "", "", fmt.Errorf("not found")
	}
	k, ok := s.kindMap[entityID]
	if !ok {
		return "", "", fmt.Errorf("not found: %s", entityID)
	}
	p := ""
	if s.parentMap != nil {
		p = s.parentMap[entityID]
	}
	return k, p, nil
}

func (s *stubLookup) GetFeatureStage(featureID string) (string, error) {
	if s.stageMap == nil {
		return "", fmt.Errorf("not found")
	}
	stage, ok := s.stageMap[featureID]
	if !ok {
		return "", fmt.Errorf("not found: %s", featureID)
	}
	return stage, nil
}

func ptr(s string) *string { return &s }
