package service

import (
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"kanbanzai/internal/core"
	"kanbanzai/internal/model"
	"kanbanzai/internal/validate"
)

func TestEntityService_CreateEpic(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	service := newTestEntityService(root, "2026-03-19T12:00:00Z")

	got, err := service.CreateEpic(CreateEpicInput{
		Slug:      "phase 1 kernel",
		Title:     "Phase 1 Kernel",
		Summary:   "Build the initial workflow kernel",
		CreatedBy: "sam",
	})
	if err != nil {
		t.Fatalf("CreateEpic() error = %v", err)
	}

	if got.Type != "epic" {
		t.Fatalf("CreateEpic() type = %q, want %q", got.Type, "epic")
	}
	if got.ID != "E-001" {
		t.Fatalf("CreateEpic() id = %q, want %q", got.ID, "E-001")
	}
	if got.Slug != "phase-1-kernel" {
		t.Fatalf("CreateEpic() slug = %q, want %q", got.Slug, "phase-1-kernel")
	}

	wantPath := filepath.Join(root, "epics", "E-001-phase-1-kernel.yaml")
	if got.Path != wantPath {
		t.Fatalf("CreateEpic() path = %q, want %q", got.Path, wantPath)
	}

	wantState := map[string]any{
		"id":         "E-001",
		"slug":       "phase-1-kernel",
		"title":      "Phase 1 Kernel",
		"status":     "proposed",
		"summary":    "Build the initial workflow kernel",
		"created":    "2026-03-19T12:00:00Z",
		"created_by": "sam",
	}
	if !reflect.DeepEqual(got.State, wantState) {
		t.Fatalf("CreateEpic() state mismatch\nwant: %#v\ngot:  %#v", wantState, got.State)
	}
}

func TestEntityService_CreateFeature_AllocatesSequentialID(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	service := newTestEntityService(root, "2026-03-19T12:00:00Z")

	first, err := service.CreateFeature(CreateFeatureInput{
		Slug:      "storage layer",
		Epic:      "E-001",
		Summary:   "Implement canonical storage",
		CreatedBy: "sam",
	})
	if err != nil {
		t.Fatalf("first CreateFeature() error = %v", err)
	}

	second, err := service.CreateFeature(CreateFeatureInput{
		Slug:      "validation engine",
		Epic:      "E-001",
		Summary:   "Implement lifecycle validation",
		CreatedBy: "sam",
	})
	if err != nil {
		t.Fatalf("second CreateFeature() error = %v", err)
	}

	if first.ID != "FEAT-001" {
		t.Fatalf("first CreateFeature() id = %q, want %q", first.ID, "FEAT-001")
	}
	if second.ID != "FEAT-002" {
		t.Fatalf("second CreateFeature() id = %q, want %q", second.ID, "FEAT-002")
	}
	if second.Slug != "validation-engine" {
		t.Fatalf("second CreateFeature() slug = %q, want %q", second.Slug, "validation-engine")
	}
}

func TestEntityService_CreateTask_AllocatesFeatureLocalID(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	service := newTestEntityService(root, "2026-03-19T12:00:00Z")

	first, err := service.CreateTask(CreateTaskInput{
		Feature: "FEAT-001",
		Slug:    "write entity files",
		Summary: "Write canonical entity files to disk",
	})
	if err != nil {
		t.Fatalf("first CreateTask() error = %v", err)
	}

	second, err := service.CreateTask(CreateTaskInput{
		Feature: "FEAT-001",
		Slug:    "read entity files",
		Summary: "Read canonical entity files from disk",
	})
	if err != nil {
		t.Fatalf("second CreateTask() error = %v", err)
	}

	otherFeature, err := service.CreateTask(CreateTaskInput{
		Feature: "FEAT-002",
		Slug:    "first task",
		Summary: "Start work for another feature",
	})
	if err != nil {
		t.Fatalf("third CreateTask() error = %v", err)
	}

	if first.ID != "FEAT-001.1" {
		t.Fatalf("first CreateTask() id = %q, want %q", first.ID, "FEAT-001.1")
	}
	if second.ID != "FEAT-001.2" {
		t.Fatalf("second CreateTask() id = %q, want %q", second.ID, "FEAT-001.2")
	}
	if otherFeature.ID != "FEAT-002.1" {
		t.Fatalf("third CreateTask() id = %q, want %q", otherFeature.ID, "FEAT-002.1")
	}
}

func TestEntityService_CreateBug_AppliesDefaults(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	service := newTestEntityService(root, "2026-03-19T12:00:00Z")

	got, err := service.CreateBug(CreateBugInput{
		Slug:       "bad-yaml-output",
		Title:      "Writer produces unstable YAML",
		ReportedBy: "sam",
		Observed:   "Repeated writes produce different output",
		Expected:   "Repeated writes should be stable",
	})
	if err != nil {
		t.Fatalf("CreateBug() error = %v", err)
	}

	wantState := map[string]any{
		"id":          "BUG-001",
		"slug":        "bad-yaml-output",
		"title":       "Writer produces unstable YAML",
		"status":      "reported",
		"severity":    "medium",
		"priority":    "medium",
		"type":        "implementation-defect",
		"reported_by": "sam",
		"reported":    "2026-03-19T12:00:00Z",
		"observed":    "Repeated writes produce different output",
		"expected":    "Repeated writes should be stable",
	}
	if !reflect.DeepEqual(got.State, wantState) {
		t.Fatalf("CreateBug() state mismatch\nwant: %#v\ngot:  %#v", wantState, got.State)
	}
}

func TestEntityService_CreateDecision(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	service := newTestEntityService(root, "2026-03-19T12:00:00Z")

	got, err := service.CreateDecision(CreateDecisionInput{
		Slug:      "strict-yaml-subset",
		Summary:   "Use a strict canonical YAML subset",
		Rationale: "Deterministic output is required for Git-friendly state",
		DecidedBy: "sam",
	})
	if err != nil {
		t.Fatalf("CreateDecision() error = %v", err)
	}

	wantState := map[string]any{
		"id":         "DEC-001",
		"slug":       "strict-yaml-subset",
		"summary":    "Use a strict canonical YAML subset",
		"rationale":  "Deterministic output is required for Git-friendly state",
		"decided_by": "sam",
		"date":       "2026-03-19T12:00:00Z",
		"status":     "proposed",
	}
	if !reflect.DeepEqual(got.State, wantState) {
		t.Fatalf("CreateDecision() state mismatch\nwant: %#v\ngot:  %#v", wantState, got.State)
	}
}

func TestEntityService_UpdateStatus_ReopensCannotReproduceBug(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	service := newTestEntityService(root, "2026-03-19T12:00:00Z")

	created, err := service.CreateBug(CreateBugInput{
		Slug:       "flaky-repro",
		Title:      "Flaky reproduction steps",
		ReportedBy: "sam",
		Observed:   "Bug appears intermittently",
		Expected:   "Bug should reproduce consistently",
	})
	if err != nil {
		t.Fatalf("CreateBug() error = %v", err)
	}

	triaged, err := service.UpdateStatus(UpdateStatusInput{
		Type:   string(model.EntityKindBug),
		ID:     created.ID,
		Slug:   created.Slug,
		Status: string(model.BugStatusTriaged),
	})
	if err != nil {
		t.Fatalf("UpdateStatus() to triaged error = %v", err)
	}
	if got := triaged.State["status"]; got != string(model.BugStatusTriaged) {
		t.Fatalf("UpdateStatus() triaged status = %v, want %q", got, model.BugStatusTriaged)
	}

	cannotReproduce, err := service.UpdateStatus(UpdateStatusInput{
		Type:   string(model.EntityKindBug),
		ID:     created.ID,
		Slug:   created.Slug,
		Status: string(model.BugStatusCannotReproduce),
	})
	if err != nil {
		t.Fatalf("UpdateStatus() to cannot-reproduce error = %v", err)
	}
	if got := cannotReproduce.State["status"]; got != string(model.BugStatusCannotReproduce) {
		t.Fatalf("UpdateStatus() cannot-reproduce status = %v, want %q", got, model.BugStatusCannotReproduce)
	}

	reopened, err := service.UpdateStatus(UpdateStatusInput{
		Type:   string(model.EntityKindBug),
		ID:     created.ID,
		Slug:   created.Slug,
		Status: string(model.BugStatusTriaged),
	})
	if err != nil {
		t.Fatalf("UpdateStatus() reopen to triaged error = %v", err)
	}

	if got := reopened.State["status"]; got != string(model.BugStatusTriaged) {
		t.Fatalf("UpdateStatus() reopened status = %v, want %q", got, model.BugStatusTriaged)
	}

	loaded, err := service.Get(string(model.EntityKindBug), created.ID, created.Slug)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got := loaded.State["status"]; got != string(model.BugStatusTriaged) {
		t.Fatalf("Get() reopened status = %v, want %q", got, model.BugStatusTriaged)
	}
}

func TestEntityService_Get_ReturnsStoredEntity(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	service := newTestEntityService(root, "2026-03-19T12:00:00Z")

	created, err := service.CreateFeature(CreateFeatureInput{
		Slug:      "entity retrieval",
		Epic:      "E-001",
		Summary:   "Support entity reads by canonical identity",
		CreatedBy: "sam",
	})
	if err != nil {
		t.Fatalf("CreateFeature() error = %v", err)
	}

	got, err := service.Get(created.Type, created.ID, created.Slug)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if got.Type != created.Type {
		t.Fatalf("Get() type = %q, want %q", got.Type, created.Type)
	}
	if got.ID != created.ID {
		t.Fatalf("Get() id = %q, want %q", got.ID, created.ID)
	}
	if got.Slug != created.Slug {
		t.Fatalf("Get() slug = %q, want %q", got.Slug, created.Slug)
	}

	wantPath := filepath.Join(core.StatePath(), "features", "FEAT-001-entity-retrieval.yaml")
	if got.Path != wantPath {
		t.Fatalf("Get() path = %q, want %q", got.Path, wantPath)
	}
	if !reflect.DeepEqual(got.State, created.State) {
		t.Fatalf("Get() state mismatch\nwant: %#v\ngot:  %#v", created.State, got.State)
	}
}

func TestEntityService_List_ReturnsEntitiesSortedByID(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	service := newTestEntityService(root, "2026-03-19T12:00:00Z")

	createdFirst, err := service.CreateFeature(CreateFeatureInput{
		Slug:      "storage layer",
		Epic:      "E-001",
		Summary:   "Implement canonical storage",
		CreatedBy: "sam",
	})
	if err != nil {
		t.Fatalf("first CreateFeature() error = %v", err)
	}

	createdSecond, err := service.CreateFeature(CreateFeatureInput{
		Slug:      "validation engine",
		Epic:      "E-001",
		Summary:   "Implement lifecycle validation",
		CreatedBy: "sam",
	})
	if err != nil {
		t.Fatalf("second CreateFeature() error = %v", err)
	}

	got, err := service.List("feature")
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	want := []ListResult{
		{
			Type:  createdFirst.Type,
			ID:    createdFirst.ID,
			Slug:  createdFirst.Slug,
			Path:  createdFirst.Path,
			State: createdFirst.State,
		},
		{
			Type:  createdSecond.Type,
			ID:    createdSecond.ID,
			Slug:  createdSecond.Slug,
			Path:  createdSecond.Path,
			State: createdSecond.State,
		},
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("List() mismatch\nwant: %#v\ngot:  %#v", want, got)
	}
}

func TestEntityService_StatusUpdate_UsesLifecycleValidation(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	service := newTestEntityService(root, "2026-03-19T12:00:00Z")

	created, err := service.CreateFeature(CreateFeatureInput{
		Slug:      "status updates",
		Epic:      "E-001",
		Summary:   "Support lifecycle status changes",
		CreatedBy: "sam",
	})
	if err != nil {
		t.Fatalf("CreateFeature() error = %v", err)
	}

	if created.State["status"] != "draft" {
		t.Fatalf("initial status = %#v, want %q", created.State["status"], "draft")
	}

	transitions := []struct {
		from string
		to   string
	}{
		{from: "draft", to: "in-review"},
		{from: "in-review", to: "approved"},
		{from: "approved", to: "in-progress"},
		{from: "in-progress", to: "review"},
		{from: "review", to: "done"},
	}

	current := created
	for _, transition := range transitions {
		updated, err := service.UpdateStatus(UpdateStatusInput{
			Type:   current.Type,
			ID:     current.ID,
			Slug:   current.Slug,
			Status: transition.to,
		})
		if err != nil {
			t.Fatalf("UpdateStatus(%q -> %q) error = %v", transition.from, transition.to, err)
		}

		if updated.State["status"] != transition.to {
			t.Fatalf(
				"updated status after %q -> %q = %#v, want %q",
				transition.from,
				transition.to,
				updated.State["status"],
				transition.to,
			)
		}

		current = CreateResult(updated)
	}

	got, err := service.Get(created.Type, created.ID, created.Slug)
	if err != nil {
		t.Fatalf("Get() after status updates error = %v", err)
	}

	if got.State["status"] != "done" {
		t.Fatalf("final persisted status = %#v, want %q", got.State["status"], "done")
	}
}

func TestEntityService_StatusUpdate_RejectsIllegalTransition(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	service := newTestEntityService(root, "2026-03-19T12:00:00Z")

	created, err := service.CreateEpic(CreateEpicInput{
		Slug:      "phase 1 kernel",
		Title:     "Phase 1 Kernel",
		Summary:   "Build the initial workflow kernel",
		CreatedBy: "sam",
	})
	if err != nil {
		t.Fatalf("CreateEpic() error = %v", err)
	}

	if created.State["status"] != "proposed" {
		t.Fatalf("initial status = %#v, want %q", created.State["status"], "proposed")
	}

	_, err = service.UpdateStatus(UpdateStatusInput{
		Type:   created.Type,
		ID:     created.ID,
		Slug:   created.Slug,
		Status: "done",
	})
	if err == nil {
		t.Fatal("UpdateStatus() error = nil, want non-nil")
	}
}

func TestEntityService_CreateTask_InvalidFeatureID(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	service := newTestEntityService(root, "2026-03-19T12:00:00Z")

	_, err := service.CreateTask(CreateTaskInput{
		Feature: "E-001",
		Slug:    "bad parent",
		Summary: "This should fail",
	})
	if err == nil {
		t.Fatal("CreateTask() error = nil, want non-nil")
	}
}

func TestEntityService_CreateEpic_MissingRequiredField(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	service := newTestEntityService(root, "2026-03-19T12:00:00Z")

	_, err := service.CreateEpic(CreateEpicInput{
		Slug:      "",
		Title:     "Phase 1 Kernel",
		Summary:   "Build the initial workflow kernel",
		CreatedBy: "sam",
	})
	if err == nil {
		t.Fatal("CreateEpic() error = nil, want non-nil")
	}
}

func TestEntityService_Get_MissingEntity(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	service := newTestEntityService(root, "2026-03-19T12:00:00Z")

	_, err := service.Get("feature", "FEAT-999", "missing")
	if err == nil {
		t.Fatal("Get() error = nil, want non-nil")
	}
}

func TestNormalizeSlug(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "lowercases and replaces spaces",
			input: "Phase 1 Kernel",
			want:  "phase-1-kernel",
		},
		{
			name:  "collapses repeated dashes from repeated spaces",
			input: "phase   1   kernel",
			want:  "phase-1-kernel",
		},
		{
			name:  "trims surrounding dashes",
			input: "  -already-slugged-  ",
			want:  "already-slugged",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := normalizeSlug(tt.input)
			if got != tt.want {
				t.Fatalf("normalizeSlug(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestValidateKindForType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		input     string
		want      validate.EntityKind
		wantError bool
	}{
		{
			name:  "epic",
			input: "epic",
			want:  validate.EntityEpic,
		},
		{
			name:  "feature",
			input: "feature",
			want:  validate.EntityFeature,
		},
		{
			name:  "task",
			input: "task",
			want:  validate.EntityTask,
		},
		{
			name:  "bug",
			input: "bug",
			want:  validate.EntityBug,
		},
		{
			name:  "decision",
			input: "decision",
			want:  validate.EntityDecision,
		},
		{
			name:      "unknown type returns error",
			input:     "unknown",
			wantError: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := validateKindForType(tt.input)
			if tt.wantError {
				if err == nil {
					t.Fatal("validateKindForType() error = nil, want non-nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("validateKindForType() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("validateKindForType(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestParseRecordIdentity(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		entityType string
		idPart     string
		wantID     string
		wantSlug   string
		wantError  bool
	}{
		{
			name:       "epic",
			entityType: "epic",
			idPart:     "E-001-phase-1-kernel",
			wantID:     "E-001",
			wantSlug:   "phase-1-kernel",
		},
		{
			name:       "feature",
			entityType: "feature",
			idPart:     "FEAT-001-storage-layer",
			wantID:     "FEAT-001",
			wantSlug:   "storage-layer",
		},
		{
			name:       "bug",
			entityType: "bug",
			idPart:     "BUG-001-bad-yaml",
			wantID:     "BUG-001",
			wantSlug:   "bad-yaml",
		},
		{
			name:       "decision",
			entityType: "decision",
			idPart:     "DEC-001-strict-yaml",
			wantID:     "DEC-001",
			wantSlug:   "strict-yaml",
		},
		{
			name:       "task",
			entityType: "task",
			idPart:     "FEAT-001.1-write-files",
			wantID:     "FEAT-001.1",
			wantSlug:   "write-files",
		},
		{
			name:       "epic with no dashes returns error",
			entityType: "epic",
			idPart:     "nodashes",
			wantError:  true,
		},
		{
			name:       "feature with only one dash segment returns error",
			entityType: "feature",
			idPart:     "FEAT",
			wantError:  true,
		},
		{
			name:       "task with no dashes returns error",
			entityType: "task",
			idPart:     "nodashes",
			wantError:  true,
		},
		{
			name:       "unknown entity type returns error",
			entityType: "unknown",
			idPart:     "X-001-slug",
			wantError:  true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			gotID, gotSlug, err := parseRecordIdentity(tt.entityType, tt.idPart)
			if tt.wantError {
				if err == nil {
					t.Fatal("parseRecordIdentity() error = nil, want non-nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("parseRecordIdentity() error = %v", err)
			}
			if gotID != tt.wantID {
				t.Fatalf("parseRecordIdentity() id = %q, want %q", gotID, tt.wantID)
			}
			if gotSlug != tt.wantSlug {
				t.Fatalf("parseRecordIdentity() slug = %q, want %q", gotSlug, tt.wantSlug)
			}
		})
	}
}

func newTestEntityService(root string, now string) *EntityService {
	svc := NewEntityService(root)

	parsed, err := time.Parse(time.RFC3339, now)
	if err != nil {
		panic(err)
	}

	svc.now = func() time.Time {
		return parsed
	}

	return svc
}
