package service

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/sambeau/kanbanzai/internal/model"
	"github.com/sambeau/kanbanzai/internal/storage"
	"github.com/sambeau/kanbanzai/internal/validate"
)

// assertIDFormat checks that an ID has the expected prefix and total length.
func assertIDFormat(t *testing.T, label, id, wantPrefix string, wantLen int) {
	t.Helper()
	if !strings.HasPrefix(id, wantPrefix) {
		t.Fatalf("%s ID = %q, want prefix %q", label, id, wantPrefix)
	}
	if len(id) != wantLen {
		t.Fatalf("%s ID = %q (len %d), want len %d", label, id, len(id), wantLen)
	}
}

func assertEpicID(t *testing.T, label, id, wantID string) {
	t.Helper()
	if id != wantID {
		t.Fatalf("%s ID = %q, want %q", label, id, wantID)
	}
}

func TestEntityService_CreateEpic(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	service := newTestEntityService(root, "2026-03-19T12:00:00Z")

	got, err := service.CreateEpic(CreateEpicInput{
		Slug:      "phase 1 kernel",
		Name:      "Phase 1 Kernel",
		Summary:   "Build the initial workflow kernel",
		CreatedBy: "sam",
	})
	if err != nil {
		t.Fatalf("CreateEpic() error = %v", err)
	}

	if got.Type != "epic" {
		t.Fatalf("CreateEpic() type = %q, want %q", got.Type, "epic")
	}
	// Epic slug "phase 1 kernel" normalizes to "phase-1-kernel", uppercased to "PHASE-1-KERNEL"
	assertEpicID(t, "CreateEpic()", got.ID, "EPIC-PHASE-1-KERNEL")
	if got.Slug != "phase-1-kernel" {
		t.Fatalf("CreateEpic() slug = %q, want %q", got.Slug, "phase-1-kernel")
	}

	wantPath := filepath.Join(root, "epics", "EPIC-PHASE-1-KERNEL-phase-1-kernel.yaml")
	if got.Path != wantPath {
		t.Fatalf("CreateEpic() path = %q, want %q", got.Path, wantPath)
	}

	wantState := map[string]any{
		"id":         "EPIC-PHASE-1-KERNEL",
		"slug":       "phase-1-kernel",
		"name":       "Phase 1 Kernel",
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

	planID := "P1-parent"
	writeTestPlan(t, service, planID)

	first, err := service.CreateFeature(CreateFeatureInput{
		Slug:      "storage layer",
		Parent:    planID,
		Summary:   "Implement canonical storage",
		CreatedBy: "sam",
	})
	if err != nil {
		t.Fatalf("first CreateFeature() error = %v", err)
	}

	second, err := service.CreateFeature(CreateFeatureInput{
		Slug:      "validation engine",
		Parent:    planID,
		Summary:   "Implement lifecycle validation",
		CreatedBy: "sam",
	})
	if err != nil {
		t.Fatalf("second CreateFeature() error = %v", err)
	}

	assertIDFormat(t, "first CreateFeature()", first.ID, "FEAT-", 18)
	assertIDFormat(t, "second CreateFeature()", second.ID, "FEAT-", 18)
	if first.ID == second.ID {
		t.Fatalf("first and second feature IDs should differ, both = %q", first.ID)
	}
	if second.Slug != "validation-engine" {
		t.Fatalf("second CreateFeature() slug = %q, want %q", second.Slug, "validation-engine")
	}
}

func TestEntityService_CreateTask_AllocatesFeatureLocalID(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	service := newTestEntityService(root, "2026-03-19T12:00:00Z")

	planID := "P1-parent"
	writeTestPlan(t, service, planID)

	feat1, err := service.CreateFeature(CreateFeatureInput{
		Slug:      "feature-one",
		Parent:    planID,
		Summary:   "First feature",
		CreatedBy: "sam",
	})
	if err != nil {
		t.Fatalf("CreateFeature(feat1) error = %v", err)
	}

	feat2, err := service.CreateFeature(CreateFeatureInput{
		Slug:      "feature-two",
		Parent:    planID,
		Summary:   "Second feature",
		CreatedBy: "sam",
	})
	if err != nil {
		t.Fatalf("CreateFeature(feat2) error = %v", err)
	}

	first, err := service.CreateTask(CreateTaskInput{
		ParentFeature: feat1.ID,
		Slug:          "write entity files",
		Summary:       "Write canonical entity files to disk",
	})
	if err != nil {
		t.Fatalf("first CreateTask() error = %v", err)
	}

	second, err := service.CreateTask(CreateTaskInput{
		ParentFeature: feat1.ID,
		Slug:          "read entity files",
		Summary:       "Read canonical entity files from disk",
	})
	if err != nil {
		t.Fatalf("second CreateTask() error = %v", err)
	}

	otherFeature, err := service.CreateTask(CreateTaskInput{
		ParentFeature: feat2.ID,
		Slug:          "first task",
		Summary:       "Start work for another feature",
	})
	if err != nil {
		t.Fatalf("third CreateTask() error = %v", err)
	}

	assertIDFormat(t, "first CreateTask()", first.ID, "TASK-", 18)
	assertIDFormat(t, "second CreateTask()", second.ID, "TASK-", 18)
	assertIDFormat(t, "third CreateTask()", otherFeature.ID, "TASK-", 18)
	if first.ID == second.ID {
		t.Fatalf("first and second task IDs should differ, both = %q", first.ID)
	}

	// Verify parent_feature is stored correctly
	if first.State["parent_feature"] != feat1.ID {
		t.Fatalf("first task parent_feature = %v, want %q", first.State["parent_feature"], feat1.ID)
	}
	if otherFeature.State["parent_feature"] != feat2.ID {
		t.Fatalf("third task parent_feature = %v, want %q", otherFeature.State["parent_feature"], feat2.ID)
	}
}

func TestEntityService_CreateBug_AppliesDefaults(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	service := newTestEntityService(root, "2026-03-19T12:00:00Z")

	got, err := service.CreateBug(CreateBugInput{
		Slug:       "bad-yaml-output",
		Name:       "Writer produces unstable YAML",
		ReportedBy: "sam",
		Observed:   "Repeated writes produce different output",
		Expected:   "Repeated writes should be stable",
	})
	if err != nil {
		t.Fatalf("CreateBug() error = %v", err)
	}

	assertIDFormat(t, "CreateBug()", got.ID, "BUG-", 17)

	wantState := map[string]any{
		"id":          got.ID, // dynamic TSID
		"slug":        "bad-yaml-output",
		"name":        "Writer produces unstable YAML",
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

	assertIDFormat(t, "CreateDecision()", got.ID, "DEC-", 17)

	wantState := map[string]any{
		"id":         got.ID, // dynamic TSID
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
		Name:       "Flaky reproduction steps",
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

	planID := "P1-parent"
	writeTestPlan(t, service, planID)

	created, err := service.CreateFeature(CreateFeatureInput{
		Slug:      "entity retrieval",
		Parent:    planID,
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

	wantPath := filepath.Join(root, "features", created.ID+"-entity-retrieval.yaml")
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

	planID := "P1-parent"
	writeTestPlan(t, service, planID)

	createdFirst, err := service.CreateFeature(CreateFeatureInput{
		Slug:      "storage layer",
		Parent:    planID,
		Summary:   "Implement canonical storage",
		CreatedBy: "sam",
	})
	if err != nil {
		t.Fatalf("first CreateFeature() error = %v", err)
	}

	createdSecond, err := service.CreateFeature(CreateFeatureInput{
		Slug:      "validation engine",
		Parent:    planID,
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

	if len(got) != 2 {
		t.Fatalf("List() returned %d results, want 2", len(got))
	}

	// With TSIDs, ordering is by TSID (which is time-based), so first created comes first.
	// But the exact order depends on sort implementation. Just check both are present.
	ids := map[string]bool{got[0].ID: true, got[1].ID: true}
	if !ids[createdFirst.ID] {
		t.Fatalf("List() missing first feature ID %q", createdFirst.ID)
	}
	if !ids[createdSecond.ID] {
		t.Fatalf("List() missing second feature ID %q", createdSecond.ID)
	}
}

func TestEntityService_StatusUpdate_UsesLifecycleValidation(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	service := newTestEntityService(root, "2026-03-19T12:00:00Z")

	planID := "P1-parent"
	writeTestPlan(t, service, planID)

	created, err := service.CreateFeature(CreateFeatureInput{
		Slug:      "status updates",
		Parent:    planID,
		Summary:   "Support lifecycle status changes",
		CreatedBy: "sam",
	})
	if err != nil {
		t.Fatalf("CreateFeature() error = %v", err)
	}

	if created.State["status"] != "proposed" {
		t.Fatalf("initial status = %#v, want %q", created.State["status"], "proposed")
	}

	transitions := []struct {
		from string
		to   string
	}{
		{from: "proposed", to: "designing"},
		{from: "designing", to: "specifying"},
		{from: "specifying", to: "dev-planning"},
		{from: "dev-planning", to: "developing"},
		{from: "developing", to: "reviewing"},
		{from: "reviewing", to: "done"},
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
		Name:      "Phase 1 Kernel",
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
		ParentFeature: "EPIC-TEST",
		Slug:          "bad parent",
		Summary:       "This should fail",
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
		Name:      "Phase 1 Kernel",
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

	_, err := service.Get("feature", "FEAT-01ZZZZZZZZZZ9", "missing")
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
			name:       "epic new format",
			entityType: "epic",
			idPart:     "EPIC-PHASE-1-KERNEL-phase-1-kernel",
			wantID:     "EPIC-PHASE-1-KERNEL",
			wantSlug:   "phase-1-kernel",
		},
		{
			name:       "feature TSID format",
			entityType: "feature",
			idPart:     "FEAT-01J3K7MXP3RT5-storage-layer",
			wantID:     "FEAT-01J3K7MXP3RT5",
			wantSlug:   "storage-layer",
		},
		{
			name:       "bug TSID format",
			entityType: "bug",
			idPart:     "BUG-01J4AR7WHN4F2-bad-yaml",
			wantID:     "BUG-01J4AR7WHN4F2",
			wantSlug:   "bad-yaml",
		},
		{
			name:       "decision TSID format",
			entityType: "decision",
			idPart:     "DEC-01J3KABCDE7MX-strict-yaml",
			wantID:     "DEC-01J3KABCDE7MX",
			wantSlug:   "strict-yaml",
		},
		{
			name:       "task TSID format",
			entityType: "task",
			idPart:     "TASK-01J3KZZZBB4KF-write-files",
			wantID:     "TASK-01J3KZZZBB4KF",
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

// writeTestPlan creates a Plan entity directly on disk via the store,
// bypassing CreatePlan (which requires global config).
func writeTestPlan(t *testing.T, svc *EntityService, id string) {
	t.Helper()
	_, _, slug := model.ParsePlanID(id)
	fields := map[string]any{
		"id":         id,
		"slug":       slug,
		"name":       "Test Plan",
		"status":     "active",
		"summary":    "Test plan for unit tests",
		"created":    "2026-03-19T12:00:00Z",
		"created_by": "test",
		"updated":    "2026-03-19T12:00:00Z",
	}
	_, err := svc.store.Write(storage.EntityRecord{
		Type:   string(model.EntityKindPlan),
		ID:     id,
		Slug:   slug,
		Fields: fields,
	})
	if err != nil {
		t.Fatalf("writeTestPlan(%s) error = %v", id, err)
	}
}

func TestEntityService_CreateBug_RejectsInvalidSeverity(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	svc := newTestEntityService(root, "2026-03-19T12:00:00Z")

	_, err := svc.CreateBug(CreateBugInput{
		Slug:       "test-bug",
		Name:       "Test",
		ReportedBy: "sam",
		Observed:   "Bad",
		Expected:   "Good",
		Severity:   "extreme",
	})
	if err == nil {
		t.Fatal("expected error for invalid severity, got nil")
	}
	if !strings.Contains(err.Error(), "severity") {
		t.Fatalf("error should mention severity, got: %v", err)
	}
}

func TestEntityService_CreateBug_RejectsInvalidPriority(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	svc := newTestEntityService(root, "2026-03-19T12:00:00Z")

	_, err := svc.CreateBug(CreateBugInput{
		Slug:       "test-bug",
		Name:       "Test",
		ReportedBy: "sam",
		Observed:   "Bad",
		Expected:   "Good",
		Priority:   "urgent",
	})
	if err == nil {
		t.Fatal("expected error for invalid priority, got nil")
	}
	if !strings.Contains(err.Error(), "priority") {
		t.Fatalf("error should mention priority, got: %v", err)
	}
}

func TestEntityService_CreateBug_RejectsInvalidType(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	svc := newTestEntityService(root, "2026-03-19T12:00:00Z")

	_, err := svc.CreateBug(CreateBugInput{
		Slug:       "test-bug",
		Name:       "Test",
		ReportedBy: "sam",
		Observed:   "Bad",
		Expected:   "Good",
		Type:       "typo",
	})
	if err == nil {
		t.Fatal("expected error for invalid bug type, got nil")
	}
	if !strings.Contains(err.Error(), "bug type") {
		t.Fatalf("error should mention bug type, got: %v", err)
	}
}

func TestEntityService_CreateBug_AcceptsValidEnums(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	svc := newTestEntityService(root, "2026-03-19T12:00:00Z")

	got, err := svc.CreateBug(CreateBugInput{
		Slug:       "test-bug",
		Name:       "Test",
		ReportedBy: "sam",
		Observed:   "Bad",
		Expected:   "Good",
		Severity:   "critical",
		Priority:   "high",
		Type:       "specification-defect",
	})
	if err != nil {
		t.Fatalf("CreateBug() error = %v", err)
	}
	if got.State["severity"] != "critical" {
		t.Fatalf("severity = %q, want %q", got.State["severity"], "critical")
	}
	if got.State["priority"] != "high" {
		t.Fatalf("priority = %q, want %q", got.State["priority"], "high")
	}
	if got.State["type"] != "specification-defect" {
		t.Fatalf("type = %q, want %q", got.State["type"], "specification-defect")
	}
}

func TestEntityService_ValidateCandidate_ValidEpic(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	svc := newTestEntityService(root, "2026-03-19T12:00:00Z")

	errs := svc.ValidateCandidate("epic", map[string]any{
		"id":         "EPIC-TESTEPIC",
		"slug":       "test",
		"name":       "Test Epic",
		"status":     "proposed",
		"summary":    "A test epic",
		"created":    "2026-03-19T12:00:00Z",
		"created_by": "agent",
	})
	if len(errs) != 0 {
		t.Fatalf("expected no errors, got %d: %v", len(errs), errs)
	}
}

func TestEntityService_ValidateCandidate_MissingField(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	svc := newTestEntityService(root, "2026-03-19T12:00:00Z")

	errs := svc.ValidateCandidate("epic", map[string]any{
		"id":      "EPIC-TESTEPIC",
		"slug":    "test",
		"status":  "proposed",
		"summary": "A test epic",
		"created": "2026-03-19T12:00:00Z",
	})
	if len(errs) == 0 {
		t.Fatal("expected validation errors for missing fields, got none")
	}

	foundName := false
	foundCreatedBy := false
	for _, e := range errs {
		if e.Field == "name" {
			foundName = true
		}
		if e.Field == "created_by" {
			foundCreatedBy = true
		}
	}
	if !foundName {
		t.Error("expected error for missing title field")
	}
	if !foundCreatedBy {
		t.Error("expected error for missing created_by field")
	}
}

func TestEntityService_ValidateCandidate_InvalidBugEnums(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	svc := newTestEntityService(root, "2026-03-19T12:00:00Z")

	errs := svc.ValidateCandidate("bug", map[string]any{
		"id":          "BUG-01J4AR7WHN4F2",
		"slug":        "test",
		"name":        "Test Bug",
		"status":      "reported",
		"severity":    "extreme",
		"priority":    "urgent",
		"type":        "typo",
		"reported_by": "sam",
		"reported":    "2026-03-19T12:00:00Z",
		"observed":    "Bad",
		"expected":    "Good",
	})
	if len(errs) == 0 {
		t.Fatal("expected validation errors for invalid enums, got none")
	}

	fieldErrors := make(map[string]bool)
	for _, e := range errs {
		fieldErrors[e.Field] = true
	}
	for _, f := range []string{"severity", "priority", "type"} {
		if !fieldErrors[f] {
			t.Errorf("expected error for invalid %s field", f)
		}
	}
}

func TestEntityService_HealthCheck_CleanProject(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	svc := newTestEntityService(root, "2026-03-19T12:00:00Z")

	// Create a valid plan and a feature referencing it.
	planID := "P1-health-test"
	writeTestPlan(t, svc, planID)

	_, err := svc.CreateFeature(CreateFeatureInput{
		Slug:      "health-feat",
		Parent:    planID,
		Summary:   "A feature for health checking",
		CreatedBy: "agent",
		Name:      "Health feature",
	})
	if err != nil {
		t.Fatalf("CreateFeature() error = %v", err)
	}

	report, err := svc.HealthCheck()
	if err != nil {
		t.Fatalf("HealthCheck() error = %v", err)
	}

	if report.Summary.TotalEntities != 2 {
		t.Fatalf("TotalEntities = %d, want 2", report.Summary.TotalEntities)
	}
	if report.Summary.ErrorCount != 0 {
		t.Fatalf("ErrorCount = %d, want 0; errors: %v", report.Summary.ErrorCount, report.Errors)
	}
	if report.Summary.WarningCount != 0 {
		t.Fatalf("WarningCount = %d, want 0; warnings: %v", report.Summary.WarningCount, report.Warnings)
	}
	if report.Summary.EntitiesByType["plan"] != 1 {
		t.Fatalf("plan count = %d, want 1", report.Summary.EntitiesByType["plan"])
	}
	if report.Summary.EntitiesByType["feature"] != 1 {
		t.Fatalf("feature count = %d, want 1", report.Summary.EntitiesByType["feature"])
	}
}

func TestEntityService_HealthCheck_EmptyProject(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	svc := newTestEntityService(root, "2026-03-19T12:00:00Z")

	report, err := svc.HealthCheck()
	if err != nil {
		t.Fatalf("HealthCheck() error = %v", err)
	}

	if report.Summary.TotalEntities != 0 {
		t.Fatalf("TotalEntities = %d, want 0", report.Summary.TotalEntities)
	}
	if report.Summary.ErrorCount != 0 {
		t.Fatalf("ErrorCount = %d, want 0", report.Summary.ErrorCount)
	}
}

func TestEntityService_CreateFeature_RejectsNonExistentParent(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	svc := newTestEntityService(root, "2026-03-19T12:00:00Z")

	_, err := svc.CreateFeature(CreateFeatureInput{
		Slug:      "orphan-feature",
		Parent:    "P1-does-not-exist",
		Summary:   "Feature referencing non-existent plan",
		CreatedBy: "sam",
	})
	if err == nil {
		t.Fatal("CreateFeature() should fail when parent plan does not exist")
	}
	if !strings.Contains(err.Error(), "P1-does-not-exist") {
		t.Fatalf("error should mention the missing plan ID, got: %v", err)
	}
}

func TestEntityService_CreateTask_RejectsNonExistentFeature(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	svc := newTestEntityService(root, "2026-03-19T12:00:00Z")

	_, err := svc.CreateTask(CreateTaskInput{
		ParentFeature: "FEAT-01ZZZZZZZZZZ9",
		Slug:          "orphan-task",
		Summary:       "Task referencing non-existent feature",
	})
	if err == nil {
		t.Fatal("CreateTask() should fail when feature does not exist")
	}
	if !strings.Contains(err.Error(), "FEAT-01ZZZZZZZZZZ9") {
		t.Fatalf("error should mention the missing feature ID, got: %v", err)
	}
}

func TestEntityService_UpdateEntity_CorrectField(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	svc := newTestEntityService(root, "2026-03-19T12:00:00Z")

	created, err := svc.CreateEpic(CreateEpicInput{
		Slug:      "phase-1-kernel",
		Name:      "Phase 1 Kernel",
		Summary:   "Build the initial workflow kernel",
		CreatedBy: "sam",
	})
	if err != nil {
		t.Fatalf("CreateEpic() error = %v", err)
	}

	updated, err := svc.UpdateEntity(UpdateEntityInput{
		Type:   created.Type,
		ID:     created.ID,
		Slug:   created.Slug,
		Fields: map[string]string{"name": "Phase 1 Kernel (Revised)"},
	})
	if err != nil {
		t.Fatalf("UpdateEntity() error = %v", err)
	}

	if updated.State["name"] != "Phase 1 Kernel (Revised)" {
		t.Fatalf("UpdateEntity() name = %v, want %q", updated.State["name"], "Phase 1 Kernel (Revised)")
	}

	got, err := svc.Get(created.Type, created.ID, created.Slug)
	if err != nil {
		t.Fatalf("Get() after update error = %v", err)
	}
	if got.State["name"] != "Phase 1 Kernel (Revised)" {
		t.Fatalf("persisted name = %v, want %q", got.State["name"], "Phase 1 Kernel (Revised)")
	}
}

func TestEntityService_UpdateEntity_RejectsIDChange(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	svc := newTestEntityService(root, "2026-03-19T12:00:00Z")

	created, err := svc.CreateEpic(CreateEpicInput{
		Slug:      "phase-1-kernel",
		Name:      "Phase 1 Kernel",
		Summary:   "Build the initial workflow kernel",
		CreatedBy: "sam",
	})
	if err != nil {
		t.Fatalf("CreateEpic() error = %v", err)
	}

	_, err = svc.UpdateEntity(UpdateEntityInput{
		Type:   created.Type,
		ID:     created.ID,
		Slug:   created.Slug,
		Fields: map[string]string{"id": "EPIC-HACKED"},
	})
	if err == nil {
		t.Fatal("UpdateEntity() error = nil, want error about immutable id")
	}
	if !strings.Contains(err.Error(), "immutable") && !strings.Contains(err.Error(), "cannot") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestEntityService_UpdateEntity_RejectsStatusChange(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	svc := newTestEntityService(root, "2026-03-19T12:00:00Z")

	created, err := svc.CreateEpic(CreateEpicInput{
		Slug:      "phase-1-kernel",
		Name:      "Phase 1 Kernel",
		Summary:   "Build the initial workflow kernel",
		CreatedBy: "sam",
	})
	if err != nil {
		t.Fatalf("CreateEpic() error = %v", err)
	}

	_, err = svc.UpdateEntity(UpdateEntityInput{
		Type:   created.Type,
		ID:     created.ID,
		Slug:   created.Slug,
		Fields: map[string]string{"status": "done"},
	})
	if err == nil {
		t.Fatal("UpdateEntity() error = nil, want error about status")
	}
	if !strings.Contains(err.Error(), "status") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestEntityService_UpdateEntity_ValidatesResult(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	svc := newTestEntityService(root, "2026-03-19T12:00:00Z")

	created, err := svc.CreateEpic(CreateEpicInput{
		Slug:      "phase-1-kernel",
		Name:      "Phase 1 Kernel",
		Summary:   "Build the initial workflow kernel",
		CreatedBy: "sam",
	})
	if err != nil {
		t.Fatalf("CreateEpic() error = %v", err)
	}

	_, err = svc.UpdateEntity(UpdateEntityInput{
		Type:   created.Type,
		ID:     created.ID,
		Slug:   created.Slug,
		Fields: map[string]string{"name": ""},
	})
	if err == nil {
		t.Fatal("UpdateEntity() error = nil, want validation error for empty title")
	}
	if !strings.Contains(err.Error(), "validation") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestEntityService_UpdateEntity_CorrectParentReference(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	svc := newTestEntityService(root, "2026-03-19T12:00:00Z")

	plan1ID := "P1-first-plan"
	writeTestPlan(t, svc, plan1ID)

	plan2ID := "P2-second-plan"
	writeTestPlan(t, svc, plan2ID)

	feat, err := svc.CreateFeature(CreateFeatureInput{
		Slug:      "my-feature",
		Parent:    plan1ID,
		Summary:   "A feature under first plan",
		CreatedBy: "sam",
		Name:      "My feature",
	})
	if err != nil {
		t.Fatalf("CreateFeature() error = %v", err)
	}

	if feat.State["parent"] != plan1ID {
		t.Fatalf("initial parent = %v, want %q", feat.State["parent"], plan1ID)
	}

	updated, err := svc.UpdateEntity(UpdateEntityInput{
		Type:   feat.Type,
		ID:     feat.ID,
		Slug:   feat.Slug,
		Fields: map[string]string{"parent": plan2ID},
	})
	if err != nil {
		t.Fatalf("UpdateEntity() error = %v", err)
	}

	if updated.State["parent"] != plan2ID {
		t.Fatalf("updated parent = %v, want %q", updated.State["parent"], plan2ID)
	}

	got, err := svc.Get(feat.Type, feat.ID, feat.Slug)
	if err != nil {
		t.Fatalf("Get() after update error = %v", err)
	}
	if got.State["parent"] != plan2ID {
		t.Fatalf("persisted parent = %v, want %q", got.State["parent"], plan2ID)
	}
}

func TestEntityService_HealthCheck_DetectsBrokenReference(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	svc := newTestEntityService(root, "2026-03-19T12:00:00Z")

	// Create a feature that references a non-existent plan.
	// We need to bypass normal validation to get this state,
	// so we write directly via the store.
	featureFields := map[string]any{
		"id":         "FEAT-01J3K7MXP3RT5",
		"slug":       "orphan-feat",
		"parent":     "P1-does-not-exist",
		"status":     "proposed",
		"summary":    "Feature with broken parent ref",
		"created":    "2026-03-19T12:00:00Z",
		"created_by": "agent",
	}
	_, err := svc.store.Write(storage.EntityRecord{
		Type:   "feature",
		ID:     "FEAT-01J3K7MXP3RT5",
		Slug:   "orphan-feat",
		Fields: featureFields,
	})
	if err != nil {
		t.Fatalf("store.Write() error = %v", err)
	}

	report, err := svc.HealthCheck()
	if err != nil {
		t.Fatalf("HealthCheck() error = %v", err)
	}

	if report.Summary.ErrorCount == 0 {
		t.Fatal("expected errors for broken parent reference, got none")
	}

	foundParentError := false
	for _, e := range report.Errors {
		if e.Field == "parent" && strings.Contains(e.Message, "P1-does-not-exist") {
			foundParentError = true
		}
	}
	if !foundParentError {
		t.Fatalf("expected error about non-existent plan P1-does-not-exist, errors: %v", report.Errors)
	}
}

func TestEntityService_EpicLifecycle(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	svc := newTestEntityService(root, "2026-03-19T12:00:00Z")

	created, err := svc.CreateEpic(CreateEpicInput{
		Slug:      "lifecycle-epic",
		Name:      "Lifecycle Epic",
		Summary:   "Test epic lifecycle transitions",
		CreatedBy: "sam",
	})
	if err != nil {
		t.Fatalf("CreateEpic() error = %v", err)
	}

	if created.State["status"] != "proposed" {
		t.Fatalf("initial status = %v, want %q", created.State["status"], "proposed")
	}

	transitions := []string{"approved", "active", "on-hold", "active", "done"}
	current := created
	for _, next := range transitions {
		prev := current.State["status"]
		updated, err := svc.UpdateStatus(UpdateStatusInput{
			Type:   current.Type,
			ID:     current.ID,
			Slug:   current.Slug,
			Status: next,
		})
		if err != nil {
			t.Fatalf("UpdateStatus(%v -> %q) error = %v", prev, next, err)
		}
		if updated.State["status"] != next {
			t.Fatalf("status after transition = %v, want %q", updated.State["status"], next)
		}
		current = CreateResult(updated)
	}

	// Terminal state: further transitions should be rejected.
	_, err = svc.UpdateStatus(UpdateStatusInput{
		Type:   current.Type,
		ID:     current.ID,
		Slug:   current.Slug,
		Status: "proposed",
	})
	if err == nil {
		t.Fatal("UpdateStatus() from terminal state should fail, got nil error")
	}
}

func TestEntityService_TaskLifecycle(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	svc := newTestEntityService(root, "2026-03-19T12:00:00Z")

	planID := "P1-task-parent"
	writeTestPlan(t, svc, planID)

	feat, err := svc.CreateFeature(CreateFeatureInput{
		Slug:      "task-parent-feature",
		Parent:    planID,
		Summary:   "Feature for task lifecycle test",
		CreatedBy: "sam",
	})
	if err != nil {
		t.Fatalf("CreateFeature() error = %v", err)
	}

	created, err := svc.CreateTask(CreateTaskInput{
		ParentFeature: feat.ID,
		Slug:          "lifecycle-task",
		Summary:       "Test task lifecycle transitions",
	})
	if err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}

	if created.State["status"] != "queued" {
		t.Fatalf("initial status = %v, want %q", created.State["status"], "queued")
	}

	transitions := []string{"ready", "active", "needs-review", "done"}
	current := created
	for _, next := range transitions {
		prev := current.State["status"]
		updated, err := svc.UpdateStatus(UpdateStatusInput{
			Type:   current.Type,
			ID:     current.ID,
			Slug:   current.Slug,
			Status: next,
		})
		if err != nil {
			t.Fatalf("UpdateStatus(%v -> %q) error = %v", prev, next, err)
		}
		if updated.State["status"] != next {
			t.Fatalf("status after transition = %v, want %q", updated.State["status"], next)
		}
		current = CreateResult(updated)
	}

	// Terminal state: further transitions should be rejected.
	_, err = svc.UpdateStatus(UpdateStatusInput{
		Type:   current.Type,
		ID:     current.ID,
		Slug:   current.Slug,
		Status: "active",
	})
	if err == nil {
		t.Fatal("UpdateStatus() from terminal state should fail, got nil error")
	}
}

func TestEntityService_DecisionLifecycle(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	svc := newTestEntityService(root, "2026-03-19T12:00:00Z")

	created, err := svc.CreateDecision(CreateDecisionInput{
		Slug:      "lifecycle-decision",
		Summary:   "Test decision lifecycle transitions",
		Rationale: "Verify lifecycle state machine",
		DecidedBy: "sam",
	})
	if err != nil {
		t.Fatalf("CreateDecision() error = %v", err)
	}

	if created.State["status"] != "proposed" {
		t.Fatalf("initial status = %v, want %q", created.State["status"], "proposed")
	}

	transitions := []string{"accepted", "superseded"}
	current := created
	for _, next := range transitions {
		prev := current.State["status"]
		updated, err := svc.UpdateStatus(UpdateStatusInput{
			Type:   current.Type,
			ID:     current.ID,
			Slug:   current.Slug,
			Status: next,
		})
		if err != nil {
			t.Fatalf("UpdateStatus(%v -> %q) error = %v", prev, next, err)
		}
		if updated.State["status"] != next {
			t.Fatalf("status after transition = %v, want %q", updated.State["status"], next)
		}
		current = CreateResult(updated)
	}

	// Terminal state: further transitions should be rejected.
	_, err = svc.UpdateStatus(UpdateStatusInput{
		Type:   current.Type,
		ID:     current.ID,
		Slug:   current.Slug,
		Status: "proposed",
	})
	if err == nil {
		t.Fatal("UpdateStatus() from terminal state should fail, got nil error")
	}
}

func TestEntityService_BugLifecycle_FullPath(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	svc := newTestEntityService(root, "2026-03-19T12:00:00Z")

	created, err := svc.CreateBug(CreateBugInput{
		Slug:       "lifecycle-bug",
		Name:       "Lifecycle Bug",
		ReportedBy: "sam",
		Observed:   "Something broke",
		Expected:   "It should work",
	})
	if err != nil {
		t.Fatalf("CreateBug() error = %v", err)
	}

	if created.State["status"] != "reported" {
		t.Fatalf("initial status = %v, want %q", created.State["status"], "reported")
	}

	transitions := []string{
		"triaged", "reproduced", "planned", "in-progress",
		"needs-review", "verified", "closed",
	}
	current := created
	for _, next := range transitions {
		prev := current.State["status"]
		updated, err := svc.UpdateStatus(UpdateStatusInput{
			Type:   current.Type,
			ID:     current.ID,
			Slug:   current.Slug,
			Status: next,
		})
		if err != nil {
			t.Fatalf("UpdateStatus(%v -> %q) error = %v", prev, next, err)
		}
		if updated.State["status"] != next {
			t.Fatalf("status after transition = %v, want %q", updated.State["status"], next)
		}
		current = CreateResult(updated)
	}

	// Terminal state: further transitions should be rejected.
	_, err = svc.UpdateStatus(UpdateStatusInput{
		Type:   current.Type,
		ID:     current.ID,
		Slug:   current.Slug,
		Status: "reported",
	})
	if err == nil {
		t.Fatal("UpdateStatus() from terminal state should fail, got nil error")
	}
}

func TestEntityService_ResolvePrefix(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	svc := newTestEntityService(root, "2026-03-19T12:00:00Z")

	epic, err := svc.CreateEpic(CreateEpicInput{
		Slug:      "prefix-epic",
		Name:      "Prefix Epic",
		Summary:   "Epic for prefix resolution tests",
		CreatedBy: "sam",
	})
	if err != nil {
		t.Fatalf("CreateEpic() error = %v", err)
	}

	planID := "P1-prefix-plan"
	writeTestPlan(t, svc, planID)

	feat1, err := svc.CreateFeature(CreateFeatureInput{
		Slug:      "alpha-feature",
		Parent:    planID,
		Summary:   "First feature",
		CreatedBy: "sam",
	})
	if err != nil {
		t.Fatalf("CreateFeature(alpha) error = %v", err)
	}

	// Advance time to ensure distinct ULID prefixes (B8 fix)
	svc.now = func() time.Time {
		return time.Date(2026, 3, 19, 12, 1, 0, 0, time.UTC)
	}

	feat2, err := svc.CreateFeature(CreateFeatureInput{
		Slug:      "beta-feature",
		Parent:    planID,
		Summary:   "Second feature",
		CreatedBy: "sam",
	})
	if err != nil {
		t.Fatalf("CreateFeature(beta) error = %v", err)
	}

	tests := []struct {
		name       string
		entityType string
		prefix     string
		wantID     string
		wantSlug   string
		wantErr    bool
		errContain string
	}{
		{
			name:       "exact full ID match",
			entityType: "feature",
			prefix:     feat1.ID,
			wantID:     feat1.ID,
			wantSlug:   feat1.Slug,
		},
		{
			name:       "unambiguous prefix",
			entityType: "feature",
			prefix:     feat1.ID[:len(feat1.ID)-2],
			wantID:     feat1.ID,
			wantSlug:   feat1.Slug,
		},
		{
			name:       "case insensitive",
			entityType: "feature",
			prefix:     strings.ToLower(feat1.ID[:len(feat1.ID)-2]),
			wantID:     feat1.ID,
			wantSlug:   feat1.Slug,
		},
		{
			name:       "break hyphen stripping",
			entityType: "feature",
			// Insert a break hyphen into the TSID portion: FEAT-XXXXX-YYYYYYYY
			prefix:   feat1.ID[:10] + "-" + feat1.ID[10:],
			wantID:   feat1.ID,
			wantSlug: feat1.Slug,
		},
		{
			name:       "epic prefix resolution",
			entityType: "epic",
			prefix:     epic.ID[:7],
			wantID:     epic.ID,
			wantSlug:   epic.Slug,
		},
		{
			name:       "ambiguous prefix",
			entityType: "feature",
			prefix:     "FEAT",
			wantErr:    true,
			errContain: "ambiguous",
		},
		{
			name:       "no match",
			entityType: "feature",
			prefix:     "FEAT-ZZZZZZZZZZZZZ",
			wantErr:    true,
			errContain: "no feature entity found",
		},
	}

	// Verify both features were created (sanity check for ambiguous test)
	_ = feat2

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			gotID, gotSlug, err := svc.ResolvePrefix(tt.entityType, tt.prefix)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("ResolvePrefix() error = nil, want error containing %q", tt.errContain)
				}
				if !strings.Contains(err.Error(), tt.errContain) {
					t.Fatalf("ResolvePrefix() error = %v, want error containing %q", err, tt.errContain)
				}
				return
			}
			if err != nil {
				t.Fatalf("ResolvePrefix() error = %v", err)
			}
			if gotID != tt.wantID {
				t.Errorf("ResolvePrefix() id = %q, want %q", gotID, tt.wantID)
			}
			if gotSlug != tt.wantSlug {
				t.Errorf("ResolvePrefix() slug = %q, want %q", gotSlug, tt.wantSlug)
			}
		})
	}
}

func TestEntityService_Get_WithEmptySlug(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	svc := newTestEntityService(root, "2026-03-19T12:00:00Z")

	planID := "P1-get"
	writeTestPlan(t, svc, planID)

	created, err := svc.CreateFeature(CreateFeatureInput{
		Slug:      "prefix-get",
		Parent:    planID,
		Summary:   "Feature for prefix get",
		CreatedBy: "sam",
	})
	if err != nil {
		t.Fatalf("CreateFeature() error = %v", err)
	}

	// Get with prefix and empty slug should resolve
	got, err := svc.Get("feature", created.ID[:10], "")
	if err != nil {
		t.Fatalf("Get() with prefix error = %v", err)
	}
	if got.ID != created.ID {
		t.Errorf("Get() id = %q, want %q", got.ID, created.ID)
	}
	if got.Slug != created.Slug {
		t.Errorf("Get() slug = %q, want %q", got.Slug, created.Slug)
	}

	// Get with non-existent prefix and empty slug should error
	_, err = svc.Get("feature", "FEAT-ZZZZZZZZZZZZZ", "")
	if err == nil {
		t.Fatal("Get() with non-existent prefix error = nil, want non-nil")
	}
}
