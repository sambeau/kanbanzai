package id

import (
	"strings"
	"testing"

	"kanbanzai/internal/model"
)

func TestAllocator_AllocateEpic(t *testing.T) {
	t.Parallel()
	allocator := NewAllocator()

	got, err := allocator.Allocate(model.EntityKindEpic, "my-project", nil)
	if err != nil {
		t.Fatalf("Allocate() error = %v", err)
	}
	if got != "EPIC-MY-PROJECT" {
		t.Fatalf("Allocate() = %q, want %q", got, "EPIC-MY-PROJECT")
	}
}

func TestAllocator_AllocateEpic_DuplicateRejected(t *testing.T) {
	t.Parallel()
	allocator := NewAllocator()

	exists := func(id string) bool { return id == "EPIC-DUPE" }
	_, err := allocator.Allocate(model.EntityKindEpic, "DUPE", exists)
	if err == nil {
		t.Fatal("Allocate() error = nil, want duplicate error")
	}
}

func TestAllocator_AllocateEpic_InvalidSlug(t *testing.T) {
	t.Parallel()
	allocator := NewAllocator()

	_, err := allocator.Allocate(model.EntityKindEpic, "A", nil) // too short
	if err == nil {
		t.Fatal("Allocate() error = nil, want error for short slug")
	}
}

func TestAllocator_AllocateTSIDTypes(t *testing.T) {
	t.Parallel()
	allocator := NewAllocator()

	types := []struct {
		kind   model.EntityKind
		prefix string
	}{
		{model.EntityKindFeature, "FEAT-"},
		{model.EntityKindBug, "BUG-"},
		{model.EntityKindDecision, "DEC-"},
		{model.EntityKindTask, "TASK-"},
		{model.EntityKindDocument, "DOC-"},
	}

	for _, tt := range types {
		tt := tt
		t.Run(string(tt.kind), func(t *testing.T) {
			t.Parallel()
			got, err := allocator.Allocate(tt.kind, "", nil)
			if err != nil {
				t.Fatalf("Allocate(%s) error = %v", tt.kind, err)
			}
			if !strings.HasPrefix(got, tt.prefix) {
				t.Fatalf("Allocate(%s) = %q, want prefix %q", tt.kind, got, tt.prefix)
			}
			// Prefix + 13-char TSID
			expectedLen := len(tt.prefix) + 13
			if len(got) != expectedLen {
				t.Fatalf("Allocate(%s) length = %d, want %d", tt.kind, len(got), expectedLen)
			}
		})
	}
}

func TestAllocator_AllocateUnknownKind(t *testing.T) {
	t.Parallel()
	allocator := NewAllocator()
	_, err := allocator.Allocate(model.EntityKind("unknown"), "", nil)
	if err == nil {
		t.Fatal("Allocate() error = nil, want error")
	}
}

func TestAllocator_CollisionRetry(t *testing.T) {
	allocator := NewAllocator()

	callCount := 0
	exists := func(id string) bool {
		callCount++
		return callCount <= 2 // first 2 collide, 3rd succeeds
	}

	got, err := allocator.Allocate(model.EntityKindFeature, "", exists)
	if err != nil {
		t.Fatalf("Allocate() error = %v", err)
	}
	if !strings.HasPrefix(got, "FEAT-") {
		t.Fatalf("Allocate() = %q, want FEAT- prefix", got)
	}
}

func TestAllocator_CollisionExhausted(t *testing.T) {
	allocator := NewAllocator()

	exists := func(id string) bool { return true } // always collides
	_, err := allocator.Allocate(model.EntityKindFeature, "", exists)
	if err == nil {
		t.Fatal("Allocate() error = nil, want collision error")
	}
}

func TestAllocator_Validate(t *testing.T) {
	t.Parallel()
	allocator := NewAllocator()

	tests := []struct {
		name    string
		kind    model.EntityKind
		id      string
		wantErr bool
	}{
		{"valid epic", model.EntityKindEpic, "EPIC-MYPROJECT", false},
		{"valid feature TSID", model.EntityKindFeature, "FEAT-01J3K7MXP3RT5", false},
		{"valid bug TSID", model.EntityKindBug, "BUG-01J4AR7WHN4F2", false},
		{"valid task TSID", model.EntityKindTask, "TASK-01J3KZZZBB4KF", false},
		{"valid decision TSID", model.EntityKindDecision, "DEC-01J3KABCDE7MX", false},
		{"valid document TSID", model.EntityKindDocument, "DOC-01J3K7MXP3RT5", false},
		{"legacy feature accepted", model.EntityKindFeature, "FEAT-001", false},
		{"legacy bug accepted", model.EntityKindBug, "BUG-002", false},
		{"wrong prefix", model.EntityKindFeature, "BUG-01J3K7MXP3RT5", true},
		{"invalid format", model.EntityKindFeature, "notanid", true},
		{"epic with invalid slug", model.EntityKindEpic, "EPIC-A", true}, // too short
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := allocator.Validate(tt.kind, tt.id)
			if tt.wantErr && err == nil {
				t.Fatal("Validate() error = nil, want non-nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("Validate() error = %v, want nil", err)
			}
		})
	}
}

func TestTypePrefix_AllKinds(t *testing.T) {
	t.Parallel()

	kinds := []struct {
		kind   model.EntityKind
		prefix string
	}{
		{model.EntityKindEpic, "EPIC"},
		{model.EntityKindFeature, "FEAT"},
		{model.EntityKindBug, "BUG"},
		{model.EntityKindDecision, "DEC"},
		{model.EntityKindTask, "TASK"},
		{model.EntityKindDocument, "DOC"},
	}

	for _, tt := range kinds {
		got, err := TypePrefix(tt.kind)
		if err != nil {
			t.Fatalf("TypePrefix(%s) error = %v", tt.kind, err)
		}
		if got != tt.prefix {
			t.Fatalf("TypePrefix(%s) = %q, want %q", tt.kind, got, tt.prefix)
		}

		// Round-trip
		kind, err := EntityKindFromPrefix(got)
		if err != nil {
			t.Fatalf("EntityKindFromPrefix(%s) error = %v", got, err)
		}
		if kind != tt.kind {
			t.Fatalf("EntityKindFromPrefix(%s) = %q, want %q", got, kind, tt.kind)
		}
	}
}

func TestParseCanonicalID(t *testing.T) {
	t.Parallel()

	prefix, ident, err := ParseCanonicalID("FEAT-01J3K7MXP3RT5")
	if err != nil {
		t.Fatalf("ParseCanonicalID() error = %v", err)
	}
	if prefix != "FEAT" || ident != "01J3K7MXP3RT5" {
		t.Fatalf("ParseCanonicalID() = (%q, %q), want (FEAT, 01J3K7MXP3RT5)", prefix, ident)
	}
}

func TestIsLegacyID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		id   string
		want bool
	}{
		{"E-001", true},
		{"FEAT-001", true},
		{"FEAT-001.1", true},
		{"BUG-002", true},
		{"DEC-003", true},
		{"EPIC-MYPROJECT", false},
		{"FEAT-01J3K7MXP3RT5", false},
		{"TASK-01J3KZZZBB4KF", false},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.id, func(t *testing.T) {
			t.Parallel()
			got := IsLegacyID(tt.id)
			if got != tt.want {
				t.Fatalf("IsLegacyID(%q) = %v, want %v", tt.id, got, tt.want)
			}
		})
	}
}
