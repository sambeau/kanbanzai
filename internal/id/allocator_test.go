package id

import (
	"testing"

	"kanbanzai/internal/model"
)

func TestAllocator_AllocateTypedIDs(t *testing.T) {
	t.Parallel()

	allocator := NewAllocator()

	tests := []struct {
		name       string
		entityKind model.EntityKind
		existing   []string
		want       string
	}{
		{
			name:       "epic starts at one",
			entityKind: model.EntityKindEpic,
			existing:   nil,
			want:       "E-001",
		},
		{
			name:       "feature increments highest existing value",
			entityKind: model.EntityKindFeature,
			existing:   []string{"FEAT-001", "FEAT-003", "FEAT-002"},
			want:       "FEAT-004",
		},
		{
			name:       "bug ignores other entity families",
			entityKind: model.EntityKindBug,
			existing:   []string{"FEAT-009", "BUG-001", "DEC-004", "BUG-010"},
			want:       "BUG-011",
		},
		{
			name:       "decision ignores invalid IDs",
			entityKind: model.EntityKindDecision,
			existing:   []string{"DEC-001", "invalid", "DEC-002", "DEC-two"},
			want:       "DEC-003",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := allocator.Allocate(tt.entityKind, tt.existing, "")
			if err != nil {
				t.Fatalf("Allocate() error = %v", err)
			}

			if got != tt.want {
				t.Fatalf("Allocate() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestAllocator_AllocateTaskID(t *testing.T) {
	t.Parallel()

	allocator := NewAllocator()

	got, err := allocator.Allocate(
		model.EntityKindTask,
		[]string{"FEAT-001.1", "FEAT-001.3", "FEAT-002.7", "invalid"},
		"FEAT-001",
	)
	if err != nil {
		t.Fatalf("Allocate() error = %v", err)
	}

	if got != "FEAT-001.4" {
		t.Fatalf("Allocate() = %q, want %q", got, "FEAT-001.4")
	}
}

func TestAllocator_AllocateTaskID_InvalidFeatureID(t *testing.T) {
	t.Parallel()

	allocator := NewAllocator()

	_, err := allocator.Allocate(model.EntityKindTask, nil, "E-001")
	if err == nil {
		t.Fatal("Allocate() error = nil, want non-nil")
	}
}

func TestAllocator_Validate(t *testing.T) {
	t.Parallel()

	allocator := NewAllocator()

	tests := []struct {
		name       string
		entityKind model.EntityKind
		id         string
		featureID  string
		wantErr    bool
	}{
		{
			name:       "valid epic ID",
			entityKind: model.EntityKindEpic,
			id:         "E-042",
			wantErr:    false,
		},
		{
			name:       "invalid epic prefix",
			entityKind: model.EntityKindEpic,
			id:         "FEAT-042",
			wantErr:    true,
		},
		{
			name:       "valid feature task ID",
			entityKind: model.EntityKindTask,
			id:         "FEAT-002.3",
			featureID:  "FEAT-002",
			wantErr:    false,
		},
		{
			name:       "task ID with wrong feature prefix",
			entityKind: model.EntityKindTask,
			id:         "FEAT-003.3",
			featureID:  "FEAT-002",
			wantErr:    true,
		},
		{
			name:       "task ID with invalid shape",
			entityKind: model.EntityKindTask,
			id:         "TASK-1",
			featureID:  "FEAT-002",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := allocator.Validate(tt.entityKind, tt.id, tt.featureID)
			if tt.wantErr && err == nil {
				t.Fatal("Validate() error = nil, want non-nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("Validate() error = %v, want nil", err)
			}
		})
	}
}

func TestAllocator_SortIDs(t *testing.T) {
	t.Parallel()

	allocator := NewAllocator()

	got := allocator.SortIDs([]string{
		"FEAT-002.10",
		"BUG-010",
		"FEAT-002.2",
		"E-003",
		"FEAT-002",
		"DEC-001",
		"BUG-002",
	})

	want := []string{
		"BUG-002",
		"BUG-010",
		"DEC-001",
		"E-003",
		"FEAT-002",
		"FEAT-002.2",
		"FEAT-002.10",
	}

	if len(got) != len(want) {
		t.Fatalf("SortIDs() length = %d, want %d", len(got), len(want))
	}

	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("SortIDs()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestAllocator_Allocate_UnknownEntityKind(t *testing.T) {
	t.Parallel()

	allocator := NewAllocator()

	_, err := allocator.Allocate(model.EntityKind("unknown"), nil, "")
	if err == nil {
		t.Fatal("Allocate() error = nil, want non-nil")
	}
}
