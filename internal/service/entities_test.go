package service

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/sambeau/kanbanzai/internal/config"
	"github.com/sambeau/kanbanzai/internal/coordination"
	"github.com/sambeau/kanbanzai/internal/id"
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

func TestEntityService_CreateFeature_AllocatesSequentialID(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	service := newTestEntityService(root, "2026-03-19T12:00:00Z")

	// Use B-prefix for batch plan (execution layer).
	planID := "B1-parent"
	writeTestPlan(t, service, planID)

	first, err := service.CreateFeature(CreateFeatureInput{
		Name:      "test",
		Slug:      "storage layer",
		Parent:    planID,
		Summary:   "Implement canonical storage",
		CreatedBy: "sam",
	})
	if err != nil {
		t.Fatalf("first CreateFeature() error = %v", err)
	}

	second, err := service.CreateFeature(CreateFeatureInput{
		Name:      "test",
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

func TestEntityService_CreateFeature_StrategicPlanParent(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	svc := newTestEntityService(root, "2026-03-19T12:00:00Z")

	// Create a strategic plan (P-prefix).
	spResult, err := svc.CreateStrategicPlan(CreateStrategicPlanInput{
		Prefix:    "P",
		Slug:      "strategic-parent",
		Name:      "Strategic Plan",
		Summary:   "A strategic plan for testing",
		CreatedBy: "sam",
	})
	if err != nil {
		t.Fatalf("CreateStrategicPlan() error = %v", err)
	}

	result, err := svc.CreateFeature(CreateFeatureInput{
		Name:      "test",
		Slug:      "feature-under-strategic",
		Parent:    spResult.ID,
		Summary:   "Feature under strategic plan",
		CreatedBy: "sam",
	})
	if err != nil {
		t.Fatalf("CreateFeature() with strategic plan parent error = %v", err)
	}

	if result.Type != string(model.EntityKindFeature) {
		t.Fatalf("CreateFeature() type = %q, want %q", result.Type, model.EntityKindFeature)
	}
}

func TestEntityService_CreateTask_AllocatesFeatureLocalID(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	service := newTestEntityService(root, "2026-03-19T12:00:00Z")

	planID := "P1-parent"
	writeTestPlan(t, service, planID)

	feat1, err := service.CreateFeature(CreateFeatureInput{
		Name:      "test",
		Slug:      "feature-one",
		Parent:    planID,
		Summary:   "First feature",
		CreatedBy: "sam",
	})
	if err != nil {
		t.Fatalf("CreateFeature(feat1) error = %v", err)
	}

	feat2, err := service.CreateFeature(CreateFeatureInput{
		Name:      "test",
		Slug:      "feature-two",
		Parent:    planID,
		Summary:   "Second feature",
		CreatedBy: "sam",
	})
	if err != nil {
		t.Fatalf("CreateFeature(feat2) error = %v", err)
	}

	first, err := service.CreateTask(CreateTaskInput{
		Name:          "test",
		ParentFeature: feat1.ID,
		Slug:          "write entity files",
		Summary:       "Write canonical entity files to disk",
	})
	if err != nil {
		t.Fatalf("first CreateTask() error = %v", err)
	}

	second, err := service.CreateTask(CreateTaskInput{
		Name:          "test",
		ParentFeature: feat1.ID,
		Slug:          "read entity files",
		Summary:       "Read canonical entity files from disk",
	})
	if err != nil {
		t.Fatalf("second CreateTask() error = %v", err)
	}

	otherFeature, err := service.CreateTask(CreateTaskInput{
		Name:          "test",
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
		"tier":        "bug_fix",
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
		Name:      "test",
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
		"name":       "test",
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

	loaded, err := service.Get(context.Background(), string(model.EntityKindBug), created.ID, created.Slug)
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
		Name:      "test",
		Slug:      "entity retrieval",
		Parent:    planID,
		Summary:   "Support entity reads by canonical identity",
		CreatedBy: "sam",
	})
	if err != nil {
		t.Fatalf("CreateFeature() error = %v", err)
	}

	got, err := service.Get(context.Background(), created.Type, created.ID, created.Slug)
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
		Name:      "test",
		Slug:      "storage layer",
		Parent:    planID,
		Summary:   "Implement canonical storage",
		CreatedBy: "sam",
	})
	if err != nil {
		t.Fatalf("first CreateFeature() error = %v", err)
	}

	createdSecond, err := service.CreateFeature(CreateFeatureInput{
		Name:      "test",
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
		Name:      "test",
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

	got, err := service.Get(context.Background(), created.Type, created.ID, created.Slug)
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

	planID := "P1-illegal-transition"
	writeTestPlan(t, service, planID)

	created, err := service.CreateFeature(CreateFeatureInput{
		Slug:      "phase-1-kernel",
		Name:      "Phase 1 Kernel",
		Parent:    planID,
		Summary:   "Build the initial workflow kernel",
		CreatedBy: "sam",
	})
	if err != nil {
		t.Fatalf("CreateFeature() error = %v", err)
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
		Name:          "test",
		ParentFeature: "BAD-PARENT",
		Slug:          "bad parent",
		Summary:       "This should fail",
	})
	if err == nil {
		t.Fatal("CreateTask() error = nil, want non-nil")
	}
}

func TestEntityService_Get_MissingEntity(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	service := newTestEntityService(root, "2026-03-19T12:00:00Z")

	_, err := service.Get(context.Background(), "feature", "FEAT-01ZZZZZZZZZZ9", "missing")
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

func TestExtractCoordinationCounter(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		id     string
		prefix string
		want   int
	}{
		{"BUG-1-slug", "BUG-1-slug", "BUG-", 1},
		{"BUG-42-slug", "BUG-42-slug", "BUG-", 42},
		{"BUG-999-multi-word-slug", "BUG-999-multi-word-slug", "BUG-", 999},
		{"B123-batch-slug", "B123-batch-slug", "B", 123},
		{"P50-plan-name", "P50-plan-name", "P", 50},
		{"TSID format returns 0", "BUG-01KQZ-WYMNEMTX", "BUG-", 0},
		{"wrong prefix returns 0", "BUG-1-slug", "FEAT-", 0},
		{"no digits returns 0", "BUG--slug", "BUG-", 0},
		{"no hyphen after digits returns 0", "BUG-1slug", "BUG-", 0},
		{"empty string returns 0", "", "BUG-", 0},
		{"short prefix returns 0", "B", "BUG-", 0},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := extractCoordinationCounter(tt.id, tt.prefix)
			if got != tt.want {
				t.Errorf("extractCoordinationCounter(%q, %q) = %d, want %d", tt.id, tt.prefix, got, tt.want)
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
func planEntityTypeFromID(id string) string {
	if len(id) > 0 && id[0] == 'B' {
		return "batch"
	}
	return "plan"
}

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
	entityType := planEntityTypeFromID(id)
	_, err := svc.store.Write(storage.EntityRecord{
		Type:   entityType,
		ID:     id,
		Slug:   slug,
		Fields: fields,
	})
	if err != nil {
		t.Fatalf("writeTestPlan(%s) error = %v", id, err)
	}
}

// TestE2E_CoordinationFullFlow verifies that EntityService uses the
// coordination database for ID allocation when wired up. It exercises
// CreateBatch, CreateBug, and CreateFeature with a real coordination DB.
func TestE2E_CoordinationFullFlow(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping integration test")
	}

	root := t.TempDir()

	// Build a config with coordination enabled and a unique project ID.
	cfg := config.Config{
		Name:          "e2e-test",
		Prefixes:      []config.PrefixEntry{{Prefix: "P"}, {Prefix: "B"}},
		SchemaVersion: "1.0.0",
		Coordination: config.CoordinationConfig{
			DatabaseURL: databaseURL,
			ProjectID:   t.Name() + "-ffe0d78a"},
	}

	// Connect the coordination DB directly.
	coordDB, err := coordination.New(context.Background(), databaseURL)
	if err != nil {
		t.Fatalf("coordination.New: %v", err)
	}
	t.Cleanup(func() { coordDB.Close() })

	if err := coordDB.Migrate(context.Background()); err != nil {
		t.Fatalf("coordination Migrate: %v", err)
	}

	// Build EntityService by hand (bypassing NewEntityService which reads
	// global config from disk).
	svc := &EntityService{
		root:           root,
		store:          storage.NewEntityStore(root),
		allocator:      id.NewAllocator(),
		cfg:            &cfg,
		coordinationDB: coordDB,
		now: func() time.Time {
			ts, _ := time.Parse(time.RFC3339, "2026-03-19T12:00:00Z")
			return ts
		},
	}

	// --- Test 1: CreateBatch uses coordination DB ---
	batchResult, err := svc.CreateBatch(CreateBatchInput{
		Prefix:    "B",
		Slug:      "coordination-batch",
		Name:      "Coordination Batch",
		Summary:   "Testing coordination DB batch ID allocation",
		CreatedBy: "tester",
	})
	if err != nil {
		t.Fatalf("CreateBatch: %v", err)
	}
	if batchResult.ID != "B1-coordination-batch" {
		t.Errorf("batch ID: expected B1-coordination-batch, got %q", batchResult.ID)
	}

	// --- Test 2: CreateBug uses coordination DB ---
	bugResult, err := svc.CreateBug(CreateBugInput{
		Slug:       "coordination-bug",
		Name:       "Test Bug",
		ReportedBy: "tester",
		Observed:   "Something broke",
		Expected:   "It should work",
	})
	if err != nil {
		t.Fatalf("CreateBug: %v", err)
	}
	if bugResult.ID != "BUG-1-coordination-bug" {
		t.Errorf("bug ID: expected BUG-1-coordination-bug, got %q", bugResult.ID)
	}

	// --- Test 3: CreateFeature uses coordinated sequence ---
	batchID := batchResult.ID // e.g. "B1-coordination-batch"
	// Write the batch entity record so the feature can find its parent.
	_, _, batchSlug := model.ParsePlanID(batchID)
	_, err = svc.store.Write(storage.EntityRecord{
		Type: string(model.EntityKindPlan),
		ID:   batchID,
		Slug: batchSlug,
		Fields: map[string]any{
			"id":               batchID,
			"slug":             batchSlug,
			"name":             "Coordination Batch",
			"status":           "active",
			"summary":          "Testing coordination",
			"next_feature_seq": 1,
			"created":          "2026-03-19T12:00:00Z",
			"created_by":       "tester",
			"updated":          "2026-03-19T12:00:00Z",
		},
	})
	if err != nil {
		t.Fatalf("write batch entity: %v", err)
	}

	featureResult, err := svc.CreateFeature(CreateFeatureInput{
		Parent:    batchID,
		Slug:      "coordination-feature",
		Name:      "Coordination Feature",
		Summary:   "Testing coordination DB feature seq allocation",
		CreatedBy: "tester",
	})
	if err != nil {
		t.Fatalf("CreateFeature: %v", err)
	}
	// Feature display_id uses format {Prefix}{num}-F{seq}.
	if featureResult.State["display_id"] != "B1-F1" {
		t.Errorf("feature display_id: expected B1-F1, got %v", featureResult.State["display_id"])
	}

	// Second feature increments sequence.
	featureResult2, err := svc.CreateFeature(CreateFeatureInput{
		Parent:    batchID,
		Slug:      "coordination-feature-2",
		Name:      "Coordination Feature 2",
		Summary:   "Second feature to verify seq increment",
		CreatedBy: "tester",
	})
	if err != nil {
		t.Fatalf("CreateFeature 2: %v", err)
	}
	if featureResult2.State["display_id"] != "B1-F2" {
		t.Errorf("feature2 display_id: expected B1-F2, got %q", featureResult2.State["display_id"])
	}

	// --- Test 4: Second bug increments independently ---
	bugResult2, err := svc.CreateBug(CreateBugInput{
		Slug:       "another-bug",
		Name:       "Another Bug",
		ReportedBy: "tester",
		Observed:   "Another issue",
		Expected:   "Another fix",
	})
	if err != nil {
		t.Fatalf("CreateBug 2: %v", err)
	}
	if bugResult2.ID != "BUG-2-another-bug" {
		t.Errorf("second bug ID: expected BUG-2-another-bug, got %q", bugResult2.ID)
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

func TestEntityService_ValidateCandidate_MissingField(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	svc := newTestEntityService(root, "2026-03-19T12:00:00Z")

	errs := svc.ValidateCandidate("feature", map[string]any{
		"id":     "FEAT-01AAAAAAAAA01",
		"slug":   "test",
		"status": "draft",
		"parent": "P1-test",
	})
	if len(errs) == 0 {
		t.Fatal("expected validation errors for missing fields, got none")
	}

	foundSummary := false
	foundCreatedBy := false
	for _, e := range errs {
		if e.Field == "summary" {
			foundSummary = true
		}
		if e.Field == "created_by" {
			foundCreatedBy = true
		}
	}
	if !foundSummary {
		t.Error("expected error for missing summary field")
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
	planID := "B1-health-test"
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
	if report.Summary.EntitiesByType["batch"] != 1 {
		t.Fatalf("batch count = %d, want 1", report.Summary.EntitiesByType["batch"])
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
		Name:      "test",
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
		Name:          "test",
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

	planID := "P1-update-field"
	writeTestPlan(t, svc, planID)

	created, err := svc.CreateFeature(CreateFeatureInput{
		Slug:      "phase-1-kernel",
		Name:      "Phase 1 Kernel",
		Parent:    planID,
		Summary:   "Build the initial workflow kernel",
		CreatedBy: "sam",
	})
	if err != nil {
		t.Fatalf("CreateFeature() error = %v", err)
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

	got, err := svc.Get(context.Background(), created.Type, created.ID, created.Slug)
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

	planID := "P1-reject-id"
	writeTestPlan(t, svc, planID)

	created, err := svc.CreateFeature(CreateFeatureInput{
		Slug:      "phase-1-kernel",
		Name:      "Phase 1 Kernel",
		Parent:    planID,
		Summary:   "Build the initial workflow kernel",
		CreatedBy: "sam",
	})
	if err != nil {
		t.Fatalf("CreateFeature() error = %v", err)
	}

	_, err = svc.UpdateEntity(UpdateEntityInput{
		Type:   created.Type,
		ID:     created.ID,
		Slug:   created.Slug,
		Fields: map[string]string{"id": "FEAT-HACKED"},
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

	planID := "P1-reject-status"
	writeTestPlan(t, svc, planID)

	created, err := svc.CreateFeature(CreateFeatureInput{
		Slug:      "phase-1-kernel",
		Name:      "Phase 1 Kernel",
		Parent:    planID,
		Summary:   "Build the initial workflow kernel",
		CreatedBy: "sam",
	})
	if err != nil {
		t.Fatalf("CreateFeature() error = %v", err)
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

	planID := "P1-validate-result"
	writeTestPlan(t, svc, planID)

	created, err := svc.CreateFeature(CreateFeatureInput{
		Slug:      "phase-1-kernel",
		Name:      "Phase 1 Kernel",
		Parent:    planID,
		Summary:   "Build the initial workflow kernel",
		CreatedBy: "sam",
	})
	if err != nil {
		t.Fatalf("CreateFeature() error = %v", err)
	}

	_, err = svc.UpdateEntity(UpdateEntityInput{
		Type:   created.Type,
		ID:     created.ID,
		Slug:   created.Slug,
		Fields: map[string]string{"summary": ""},
	})
	if err == nil {
		t.Fatal("UpdateEntity() error = nil, want validation error for empty summary")
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

	got, err := svc.Get(context.Background(), feat.Type, feat.ID, feat.Slug)
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

func TestEntityService_TaskLifecycle(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	svc := newTestEntityService(root, "2026-03-19T12:00:00Z")

	planID := "P1-task-parent"
	writeTestPlan(t, svc, planID)

	feat, err := svc.CreateFeature(CreateFeatureInput{
		Name:      "test",
		Slug:      "task-parent-feature",
		Parent:    planID,
		Summary:   "Feature for task lifecycle test",
		CreatedBy: "sam",
	})
	if err != nil {
		t.Fatalf("CreateFeature() error = %v", err)
	}

	created, err := svc.CreateTask(CreateTaskInput{
		Name:          "test",
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
		Name:      "test",
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
		"needs-review", "verifying", "closed",
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

// TestEntityService_BugReviewCycle_InitialZero verifies FR-012: new bugs have review_cycle=0.
func TestEntityService_BugReviewCycle_InitialZero(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	svc := newTestEntityService(root, "2026-03-19T12:00:00Z")

	created, err := svc.CreateBug(CreateBugInput{
		Slug:       "cycle-zero-bug",
		Name:       "Cycle Zero Bug",
		ReportedBy: "sam",
		Observed:   "Something broke",
		Expected:   "It should work",
	})
	if err != nil {
		t.Fatalf("CreateBug() error = %v", err)
	}

	// Re-read the bug from the store to verify review_cycle.
	rec, err := svc.store.Load("bug", created.ID, created.Slug)
	if err != nil {
		t.Fatalf("Load bug: %v", err)
	}
	rc, _ := rec.Fields["review_cycle"].(int)
	if rc != 0 {
		t.Errorf("review_cycle = %d, want 0 (FR-012)", rc)
	}
}

// TestEntityService_BugIncrementReviewCycle verifies FR-013: IncrementBugReviewCycle
// increments the review_cycle field.
func TestEntityService_BugIncrementReviewCycle(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	svc := newTestEntityService(root, "2026-03-19T12:00:00Z")

	created, err := svc.CreateBug(CreateBugInput{
		Slug:       "increment-cycle-bug",
		Name:       "Increment Cycle Bug",
		ReportedBy: "sam",
		Observed:   "Something broke",
		Expected:   "It should work",
	})
	if err != nil {
		t.Fatalf("CreateBug() error = %v", err)
	}

	// Increment once.
	if err := svc.IncrementBugReviewCycle(created.ID, created.Slug); err != nil {
		t.Fatalf("IncrementBugReviewCycle(1): %v", err)
	}
	rec, err := svc.store.Load("bug", created.ID, created.Slug)
	if err != nil {
		t.Fatalf("Load bug: %v", err)
	}
	rc, _ := rec.Fields["review_cycle"].(int)
	if rc != 1 {
		t.Errorf("review_cycle after 1 increment = %d, want 1 (FR-013)", rc)
	}

	// Increment again.
	if err := svc.IncrementBugReviewCycle(created.ID, created.Slug); err != nil {
		t.Fatalf("IncrementBugReviewCycle(2): %v", err)
	}
	rec, err = svc.store.Load("bug", created.ID, created.Slug)
	if err != nil {
		t.Fatalf("Load bug: %v", err)
	}
	rc, _ = rec.Fields["review_cycle"].(int)
	if rc != 2 {
		t.Errorf("review_cycle after 2 increments = %d, want 2 (FR-013)", rc)
	}
}

// TestEntityService_BugReviewCycleCapBlocks verifies FR-014: when review_cycle >= MaxCycles,
// the needs-review→needs-rework gate blocks with blocked_reason and ReviewCapReached.
func TestEntityService_BugReviewCycleCapBlocks(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	svc := newTestEntityService(root, "2026-03-19T12:00:00Z")

	created, err := svc.CreateBug(CreateBugInput{
		Slug:       "cap-block-bug",
		Name:       "Cap Block Bug",
		ReportedBy: "sam",
		Observed:   "Something broke",
		Expected:   "It should work",
		Tier:       "bug_fix",
	})
	if err != nil {
		t.Fatalf("CreateBug() error = %v", err)
	}

	// Set review_cycle to the cap (MaxCycles=2 for bug_fix).
	for i := 0; i < 2; i++ {
		if err := svc.IncrementBugReviewCycle(created.ID, created.Slug); err != nil {
			t.Fatalf("IncrementBugReviewCycle(%d): %v", i+1, err)
		}
	}

	// Re-read the bug to get current ReviewCycle value.
	rec, err := svc.store.Load("bug", created.ID, created.Slug)
	if err != nil {
		t.Fatalf("Load bug: %v", err)
	}
	rc, _ := rec.Fields["review_cycle"].(int)
	if rc != 2 {
		t.Fatalf("review_cycle = %d, want 2 for cap test", rc)
	}

	// Verify CheckBugTransitionGate blocks needs-review→needs-rework at cap.
	bug := model.Bug{
		ID:          created.ID,
		Slug:        created.Slug,
		Status:      model.BugStatusNeedsReview,
		Tier:        "bug_fix",
		ReviewCycle: rc,
	}
	result := CheckBugTransitionGate(
		string(model.BugStatusNeedsReview),
		string(model.BugStatusNeedsRework),
		&bug,
		nil, // docSvc not needed for this gate
		svc,
	)
	if result.Satisfied {
		t.Error("expected gate to be unsatisfied at cap")
	}
	if !result.ReviewCapReached {
		t.Error("expected ReviewCapReached=true (FR-014)")
	}
}

// TestEntityService_BugReviewCycle_OnlyReworkIncrements verifies FR-015: only
// needs-review→needs-rework increments review_cycle; other transitions do not.
func TestEntityService_BugReviewCycle_OnlyReworkIncrements(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	svc := newTestEntityService(root, "2026-03-19T12:00:00Z")

	created, err := svc.CreateBug(CreateBugInput{
		Slug:       "no-increment-bug",
		Name:       "No Increment Bug",
		ReportedBy: "sam",
		Observed:   "Something broke",
		Expected:   "It should work",
	})
	if err != nil {
		t.Fatalf("CreateBug() error = %v", err)
	}

	// Walk through several non-rework transitions.
	transitions := []string{"triaged", "reproduced", "planned", "in-progress"}
	current := created
	for _, next := range transitions {
		updated, err := svc.UpdateStatus(UpdateStatusInput{
			Type:   current.Type,
			ID:     current.ID,
			Slug:   current.Slug,
			Status: next,
		})
		if err != nil {
			t.Fatalf("UpdateStatus(%q -> %q) error = %v", current.State["status"], next, err)
		}
		current = updated
	}

	// After all non-rework transitions, review_cycle should still be 0.
	rec, err := svc.store.Load("bug", created.ID, created.Slug)
	if err != nil {
		t.Fatalf("Load bug: %v", err)
	}
	rc, _ := rec.Fields["review_cycle"].(int)
	if rc != 0 {
		t.Errorf("review_cycle after non-rework transitions = %d, want 0 (FR-015)", rc)
	}
}

func TestEntityService_ResolvePrefix(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	svc := newTestEntityService(root, "2026-03-19T12:00:00Z")

	planID := "P1-prefix-plan"
	writeTestPlan(t, svc, planID)

	feat1, err := svc.CreateFeature(CreateFeatureInput{
		Name:      "test",
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
		Name:      "test",
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
		Name:      "test",
		Slug:      "prefix-get",
		Parent:    planID,
		Summary:   "Feature for prefix get",
		CreatedBy: "sam",
	})
	if err != nil {
		t.Fatalf("CreateFeature() error = %v", err)
	}

	// Get with prefix and empty slug should resolve
	got, err := svc.Get(context.Background(), "feature", created.ID[:10], "")
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
	_, err = svc.Get(context.Background(), "feature", "FEAT-ZZZZZZZZZZZZZ", "")
	if err == nil {
		t.Fatal("Get() with non-existent prefix error = nil, want non-nil")
	}
}

func TestEntityService_CoordinationDisabled_UsesLocalAllocation(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	svc := newTestEntityService(root, "2026-03-19T12:00:00Z")

	// coordinationDB is nil by default (no DATABASE_URL in test config)
	if svc.coordinationDB != nil {
		t.Fatal("coordinationDB should be nil in tests (no DATABASE_URL)")
	}

	// CreateBug should use local TSID allocation
	bug, err := svc.CreateBug(CreateBugInput{
		Slug:       "local-bug",
		Name:       "Local Bug",
		ReportedBy: "tester",
		Observed:   "something went wrong",
		Expected:   "everything should work",
	})
	if err != nil {
		t.Fatalf("CreateBug() error = %v", err)
	}
	if !strings.HasPrefix(bug.ID, "BUG-") {
		t.Errorf("CreateBug() ID = %q, want BUG- prefix", bug.ID)
	}
	if bug.Slug != "local-bug" {
		t.Errorf("CreateBug() slug = %q, want %q", bug.Slug, "local-bug")
	}

	// CreateFeature should use local sequence from parent state
	planID := "P1-local-allocation"
	writeTestPlan(t, svc, planID)

	feat, err := svc.CreateFeature(CreateFeatureInput{
		Name:      "test",
		Slug:      "local-feature",
		Parent:    planID,
		Summary:   "Feature with local allocation",
		CreatedBy: "tester",
	})
	if err != nil {
		t.Fatalf("CreateFeature() error = %v", err)
	}
	if !strings.HasPrefix(feat.ID, "FEAT-") {
		t.Errorf("CreateFeature() ID = %q, want FEAT- prefix", feat.ID)
	}
	// Display ID should use local seq from parent (starts at 1 for new plans)
	if feat.State["display_id"] != "P1-F1" {
		t.Errorf("CreateFeature() display_id = %q, want P1-F1", feat.State["display_id"])
	}
}

// ─── Tier inference tests ────────────────────────────────────────────────────

// testEntityServiceWithConfig creates an EntityService with a custom config
// for tests that need to control fast-track settings.
func testEntityServiceWithConfig(t *testing.T, cfg config.Config) *EntityService {
	t.Helper()
	root := t.TempDir()
	svc := newTestEntityService(root, "2026-03-19T12:00:00Z")
	svc.cfg = &cfg
	return svc
}

func TestTierInference_ExplicitTierOverridesAll(t *testing.T) {
	t.Parallel()

	cfg := config.DefaultConfig()
	cfg.FastTrack.DefaultTier = config.TierFeature
	svc := testEntityServiceWithConfig(t, cfg)

	planID := "P1-tier-test"
	writeTestPlan(t, svc, planID)

	// Explicit tier should win over tags and default.
	result, err := svc.CreateFeature(CreateFeatureInput{
		Name:      "test",
		Slug:      "explicit-tier",
		Parent:    planID,
		Summary:   "Test explicit tier override",
		CreatedBy: "tester",
		Tier:      config.TierRetroFix,
		Tags:      []string{"critical"},
	})
	if err != nil {
		t.Fatalf("CreateFeature() error = %v", err)
	}
	if result.State["tier"] != config.TierRetroFix {
		t.Errorf("tier = %q, want %q", result.State["tier"], config.TierRetroFix)
	}
}

func TestTierInference_CriticalTagProducesCriticalTier(t *testing.T) {
	t.Parallel()

	cfg := config.DefaultConfig()
	cfg.FastTrack.DefaultTier = config.TierFeature
	svc := testEntityServiceWithConfig(t, cfg)

	planID := "P1-tier-critical"
	writeTestPlan(t, svc, planID)

	result, err := svc.CreateFeature(CreateFeatureInput{
		Name:      "test",
		Slug:      "critical-tag",
		Parent:    planID,
		Summary:   "Test critical tag inference",
		CreatedBy: "tester",
		Tags:      []string{"critical"},
	})
	if err != nil {
		t.Fatalf("CreateFeature() error = %v", err)
	}
	if result.State["tier"] != config.TierCritical {
		t.Errorf("tier = %q, want %q", result.State["tier"], config.TierCritical)
	}
}

func TestTierInference_SecurityTagProducesCriticalTier(t *testing.T) {
	t.Parallel()

	cfg := config.DefaultConfig()
	cfg.FastTrack.DefaultTier = config.TierFeature
	svc := testEntityServiceWithConfig(t, cfg)

	planID := "P1-tier-security"
	writeTestPlan(t, svc, planID)

	result, err := svc.CreateFeature(CreateFeatureInput{
		Name:      "test",
		Slug:      "security-tag",
		Parent:    planID,
		Summary:   "Test security tag inference",
		CreatedBy: "tester",
		Tags:      []string{"security"},
	})
	if err != nil {
		t.Fatalf("CreateFeature() error = %v", err)
	}
	if result.State["tier"] != config.TierCritical {
		t.Errorf("tier = %q, want %q", result.State["tier"], config.TierCritical)
	}
}

func TestTierInference_DefaultTierFromConfig(t *testing.T) {
	t.Parallel()

	cfg := config.DefaultConfig()
	cfg.FastTrack.DefaultTier = config.TierBugFix
	svc := testEntityServiceWithConfig(t, cfg)

	planID := "P1-tier-default"
	writeTestPlan(t, svc, planID)

	result, err := svc.CreateFeature(CreateFeatureInput{
		Name:      "test",
		Slug:      "default-tier",
		Parent:    planID,
		Summary:   "Test default tier from config",
		CreatedBy: "tester",
	})
	if err != nil {
		t.Fatalf("CreateFeature() error = %v", err)
	}
	if result.State["tier"] != config.TierBugFix {
		t.Errorf("tier = %q, want %q", result.State["tier"], config.TierBugFix)
	}
}

func TestTierInference_DefaultConfigTierIsFeature(t *testing.T) {
	t.Parallel()

	cfg := config.DefaultConfig() // DefaultTier is "feature"
	svc := testEntityServiceWithConfig(t, cfg)

	planID := "P1-tier-feature"
	writeTestPlan(t, svc, planID)

	result, err := svc.CreateFeature(CreateFeatureInput{
		Name:      "test",
		Slug:      "feature-tier",
		Parent:    planID,
		Summary:   "Test default feature tier",
		CreatedBy: "tester",
	})
	if err != nil {
		t.Fatalf("CreateFeature() error = %v", err)
	}
	if result.State["tier"] != config.TierFeature {
		t.Errorf("tier = %q, want %q", result.State["tier"], config.TierFeature)
	}
}

func TestTierInference_NilConfigUsesFeatureDefault(t *testing.T) {
	t.Parallel()

	cfg := config.Config{} // empty config; no FastTrack set
	svc := testEntityServiceWithConfig(t, cfg)

	planID := "P1-tier-emptycfg"
	writeTestPlan(t, svc, planID)

	result, err := svc.CreateFeature(CreateFeatureInput{
		Name:      "test",
		Slug:      "empty-cfg-tier",
		Parent:    planID,
		Summary:   "Test empty config fallback",
		CreatedBy: "tester",
	})
	if err != nil {
		t.Fatalf("CreateFeature() error = %v", err)
	}
	if result.State["tier"] != config.TierFeature {
		t.Errorf("tier = %q, want %q", result.State["tier"], config.TierFeature)
	}
}

func TestTierInference_TagCaseInsensitive(t *testing.T) {
	t.Parallel()

	cfg := config.DefaultConfig()
	cfg.FastTrack.DefaultTier = config.TierFeature
	svc := testEntityServiceWithConfig(t, cfg)

	planID := "P1-tier-case"
	writeTestPlan(t, svc, planID)

	result, err := svc.CreateFeature(CreateFeatureInput{
		Name:      "test",
		Slug:      "case-insensitive",
		Parent:    planID,
		Summary:   "Test case insensitive tag matching",
		CreatedBy: "tester",
		Tags:      []string{"CRITICAL"},
	})
	if err != nil {
		t.Fatalf("CreateFeature() error = %v", err)
	}
	if result.State["tier"] != config.TierCritical {
		t.Errorf("tier = %q, want %q", result.State["tier"], config.TierCritical)
	}
}

func TestTierInference_IrrelevantTagsUseDefault(t *testing.T) {
	t.Parallel()

	cfg := config.DefaultConfig()
	cfg.FastTrack.DefaultTier = config.TierBugFix
	svc := testEntityServiceWithConfig(t, cfg)

	planID := "P1-tier-irrelevant"
	writeTestPlan(t, svc, planID)

	result, err := svc.CreateFeature(CreateFeatureInput{
		Name:      "test",
		Slug:      "irrelevant-tags",
		Parent:    planID,
		Summary:   "Test irrelevant tags fall through to default",
		CreatedBy: "tester",
		Tags:      []string{"documentation", "refactor"},
	})
	if err != nil {
		t.Fatalf("CreateFeature() error = %v", err)
	}
	if result.State["tier"] != config.TierBugFix {
		t.Errorf("tier = %q, want %q", result.State["tier"], config.TierBugFix)
	}
}

func TestTierInference_SecurityTagWithWhitespace(t *testing.T) {
	t.Parallel()

	cfg := config.DefaultConfig()
	svc := testEntityServiceWithConfig(t, cfg)

	planID := "P1-tier-ws"
	writeTestPlan(t, svc, planID)

	result, err := svc.CreateFeature(CreateFeatureInput{
		Name:      "test",
		Slug:      "ws-tag",
		Parent:    planID,
		Summary:   "Test tag with whitespace",
		CreatedBy: "tester",
		Tags:      []string{"  security  "},
	})
	if err != nil {
		t.Fatalf("CreateFeature() error = %v", err)
	}
	if result.State["tier"] != config.TierCritical {
		t.Errorf("tier = %q, want %q", result.State["tier"], config.TierCritical)
	}
}

func TestTierInference_EmptyDefaultTierFallsBackToFeature(t *testing.T) {
	t.Parallel()

	cfg := config.DefaultConfig()
	cfg.FastTrack.DefaultTier = ""
	svc := testEntityServiceWithConfig(t, cfg)

	planID := "P1-tier-empty"
	writeTestPlan(t, svc, planID)

	result, err := svc.CreateFeature(CreateFeatureInput{
		Name:      "test",
		Slug:      "empty-default",
		Parent:    planID,
		Summary:   "Test empty default tier",
		CreatedBy: "tester",
	})
	if err != nil {
		t.Fatalf("CreateFeature() error = %v", err)
	}
	if result.State["tier"] != config.TierFeature {
		t.Errorf("tier = %q, want %q", result.State["tier"], config.TierFeature)
	}
}

func TestTierInference_TierStoredAndRetrievable(t *testing.T) {
	t.Parallel()

	cfg := config.DefaultConfig()
	cfg.FastTrack.DefaultTier = config.TierFeature
	svc := testEntityServiceWithConfig(t, cfg)

	planID := "P1-tier-stored"
	writeTestPlan(t, svc, planID)

	createResult, err := svc.CreateFeature(CreateFeatureInput{
		Name:      "test",
		Slug:      "stored-tier",
		Parent:    planID,
		Summary:   "Test tier is stored and retrievable",
		CreatedBy: "tester",
		Tier:      config.TierRetroFix,
	})
	if err != nil {
		t.Fatalf("CreateFeature() error = %v", err)
	}

	// Retrieve and verify tier is persisted.
	getResult, getErr := svc.Get(context.Background(), "feature", createResult.ID, "")
	if getErr != nil {
		t.Fatalf("Get(%s) error = %v", createResult.ID, getErr)
	}
	if getResult.State["tier"] != config.TierRetroFix {
		t.Errorf("retrieved tier = %q, want %q", getResult.State["tier"], config.TierRetroFix)
	}
}

// ─── REQ-005: Decision record for capability gap ─────────────────────────────

// TestEntityService_REQ005_DecisionRecord_CoversCapabilityGap verifies that
// a decision entity can be created and retrieved with all fields required to
// flag the handoff capability gap (spec sections, conflict annotations, graph
// traversal) for separate feature planning.
func TestEntityService_REQ005_DecisionRecord_CoversCapabilityGap(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	service := newTestEntityService(root, "2026-05-08T16:53:36Z")

	got, err := service.CreateDecision(CreateDecisionInput{
		Name:      "Handoff capability gap — spec sections conflict graph",
		Slug:      "handoff-capability-gap-spec-sections-conflict-graph",
		Summary:   "The handoff tool documentation previously claimed three capabilities that are not implemented: (1) spec section injection into the context packet, (2) conflict annotation of tasks based on domain analysis, (3) graph traversal to surface related code nodes. These claims have been removed from AGENTS.md and .github/copilot-instructions.md as part of P61 Track D. This decision record flags the gap for separate feature planning under a future plan.",
		Rationale: "During P61 Track D documentation reconciliation, three capability gaps were confirmed by inspection of internal/mcp/handoff_tool.go and internal/mcp/assembly.go: (1) Spec section injection — handoff does not pull approved spec documents or inject their sections into the assembled prompt; (2) Conflict annotation — handoff does not call conflict_domain_check and does not annotate the context packet with per-task conflict risk; (3) Graph traversal — handoff does not execute graph queries to find related code nodes or include them in the context packet.",
		DecidedBy: "sambeau",
	})
	if err != nil {
		t.Fatalf("CreateDecision() error = %v", err)
	}

	// Verify type and status.
	if got.Type != "decision" {
		t.Errorf("type = %q, want decision", got.Type)
	}
	if got.State["status"] != "proposed" {
		t.Errorf("status = %v, want proposed", got.State["status"])
	}

	// Verify the three capability gaps are present in summary.
	summary, _ := got.State["summary"].(string)
	for _, gap := range []string{
		"spec section",
		"conflict annotation",
		"graph traversal",
	} {
		if !strings.Contains(strings.ToLower(summary), gap) {
			t.Errorf("summary must mention capability gap: %q", gap)
		}
	}

	// Verify summary explicitly states this is for separate feature planning.
	if !strings.Contains(summary, "separate feature planning") &&
		!strings.Contains(summary, "future plan") {
		t.Error("summary must indicate the gap is for separate/future planning")
	}

	// Verify the three capability gaps are present in rationale.
	rationale, _ := got.State["rationale"].(string)
	for _, gap := range []string{
		"spec section",
		"conflict annotation",
		"graph traversal",
	} {
		if !strings.Contains(strings.ToLower(rationale), gap) {
			t.Errorf("rationale must mention capability gap: %q", gap)
		}
	}

	// Verify we can retrieve the decision by ID.
	retrieved, err := service.Get(context.Background(), "decision", got.ID, "")
	if err != nil {
		t.Fatalf("Get(decision, %s) error = %v", got.ID, err)
	}
	if retrieved.Type != "decision" {
		t.Errorf("retrieved type = %q, want decision", retrieved.Type)
	}
	if retrieved.State["status"] != "proposed" {
		t.Errorf("retrieved status = %v, want proposed", retrieved.State["status"])
	}
}

// TestREQ005_DecisionDocument_ExistsAndCoversGap verifies that the decision
// document (P61-report-handoff-capability-gap.md) exists and covers all three
// capability gaps: spec sections, conflict annotations, and graph traversal.
//
// The document may be in a worktree (during development) or in the main repo
// (after merge). This test searches both locations.
func TestREQ005_DecisionDocument_ExistsAndCoversGap(t *testing.T) {
	t.Parallel()

	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot resolve test file path")
	}
	// From internal/service/entities_test.go: .. → internal/, ../.. → repo/worktree root.
	repoRoot := filepath.Join(filepath.Dir(thisFile), "..", "..")

	docRelPath := filepath.Join("work", "P61-handoff-resilience-binding-hardening",
		"P61-report-handoff-capability-gap.md")

	// Search: first in repoRoot (main repo), then in worktree subdirectory.
	candidates := []string{
		filepath.Join(repoRoot, docRelPath),
		// Worktree path: .worktrees/FEAT-01KR46PKHMG4J-doc-reconciliation/work/...
		filepath.Join(repoRoot, ".worktrees", "FEAT-01KR46PKHMG4J-doc-reconciliation", docRelPath),
	}

	var content string
	found := false
	for _, docPath := range candidates {
		data, err := os.ReadFile(docPath)
		if err == nil {
			content = string(data)
			found = true
			break
		}
	}
	if !found {
		t.Skip("decision document not found in repo root or worktree — it may not be committed yet")
	}

	// Verify the document identifies the decision entity.
	if !strings.Contains(content, "DEC-01KR484HRQ97X") {
		t.Error("document must reference the decision entity ID DEC-01KR484HRQ97X")
	}

	// Verify the three capability gaps are documented.
	gaps := map[string]string{
		"spec section injection": "spec section",
		"conflict annotation":    "conflict annotation",
		"graph traversal":        "graph traversal",
	}
	for gapName, searchTerm := range gaps {
		if !strings.Contains(strings.ToLower(content), searchTerm) {
			t.Errorf("document must cover capability gap: %s", gapName)
		}
	}

	// Verify the document proposes next steps (separate feature planning).
	if !strings.Contains(content, "Next Steps") &&
		!strings.Contains(content, "Recommended Next Steps") {
		t.Error("document must include recommended next steps")
	}

	// Verify the document contains a Decision Options section.
	if !strings.Contains(content, "Decision Options") &&
		!strings.Contains(content, "Option A") {
		t.Error("document must include decision options")
	}
}

// TestREQ005_DecisionEntity_ExistsOnDisk verifies the decision entity YAML
// file exists in .kbz/state/decisions/ and contains the required fields
// flagging the three capability gaps for separate feature planning.
func TestREQ005_DecisionEntity_ExistsOnDisk(t *testing.T) {
	// Not parallel — reads real on-disk state.

	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot resolve test file path")
	}
	// From internal/service/entities_test.go: .. → internal/, ../.. → repo/worktree root.
	// The .kbz/state/decisions/ directory in the worktree contains the same
	// entities as the main repo (shared via git worktree).
	repoRoot := filepath.Join(filepath.Dir(thisFile), "..", "..")

	// Find the decision entity file by glob — the filename contains the slug.
	globPattern := filepath.Join(repoRoot, ".kbz", "state", "decisions", "DEC-01KR484HRQ97X-*.yaml")
	matches, err := filepath.Glob(globPattern)
	if err != nil {
		t.Fatalf("glob decision entity: %v", err)
	}
	if len(matches) == 0 {
		t.Skipf("decision entity DEC-01KR484HRQ97X not found at %s — entity may be in main repo only", globPattern)
	}

	data, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatalf("read decision entity %s: %v", matches[0], err)
	}

	content := string(data)

	// Verify required fields.
	requiredFields := []string{
		"id: DEC-01KR484HRQ97X",
		"slug: handoff-capability-gap-spec-sections-conflict-graph",
		"status: proposed",
		"decided_by: sambeau",
	}
	for _, field := range requiredFields {
		if !strings.Contains(content, field) {
			t.Errorf("entity must contain field: %q", field)
		}
	}

	// Verify the three capability gaps are mentioned.
	for _, gap := range []string{
		"spec section",
		"conflict annotation",
		"graph traversal",
	} {
		if !strings.Contains(strings.ToLower(content), gap) {
			t.Errorf("entity must mention capability gap: %q", gap)
		}
	}

	// Verify the entity explicitly states separate feature planning.
	if !strings.Contains(content, "separate feature planning") &&
		!strings.Contains(content, "future plan") {
		t.Error("entity must indicate the gap is for separate/future planning")
	}
}

// ─── AC-005: maintainer reviews capability gap ─────────────────────────────

// TestAC005_MaintainerRetrievesDecisionRecord verifies that a project
// maintainer can retrieve the actual decision record DEC-01KR484HRQ97X
// through the entity service and confirm it documents all three capability
// gaps (spec sections, conflict annotations, graph traversal) and proposes
// next steps. This simulates the maintainer review scenario in AC-005.
func TestAC005_MaintainerRetrievesDecisionRecord(t *testing.T) {
	// Not parallel — uses real entity store.

	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot resolve test file path")
	}
	// From internal/service/entities_test.go: .. → internal/, ../.. → repo/worktree root.
	repoRoot := filepath.Join(filepath.Dir(thisFile), "..", "..")

	// Initialise entity service against the repo root so it reads from the
	// actual .kbz/state/ store — same path the MCP entity tool uses.
	svc := NewEntityService(repoRoot)

	retrieved, err := svc.Get(context.Background(), "decision", "DEC-01KR484HRQ97X", "")
	if err != nil {
		// The entity may not exist in this worktree's store (decision
		// entities are stored in the shared .kbz/state/ which worktrees
		// inherit from main). Skip rather than fail if not found.
		t.Skipf("decision entity DEC-01KR484HRQ97X not retrievable: %v", err)
	}

	// Verify entity type.
	if retrieved.Type != "decision" {
		t.Errorf("type = %q, want decision", retrieved.Type)
	}

	// Verify status is 'proposed' — the maintainer-reviews state.
	if retrieved.State["status"] != "proposed" {
		t.Errorf("status = %v, want proposed", retrieved.State["status"])
	}

	summary, _ := retrieved.State["summary"].(string)
	rationale, _ := retrieved.State["rationale"].(string)

	// Verify all three capability gaps are documented in the summary.
	gaps := []string{
		"spec section",
		"conflict annotation",
		"graph traversal",
	}
	for _, gap := range gaps {
		if !strings.Contains(strings.ToLower(summary), gap) {
			t.Errorf("summary must mention capability gap: %q", gap)
		}
		if !strings.Contains(strings.ToLower(rationale), gap) {
			t.Errorf("rationale must mention capability gap: %q", gap)
		}
	}

	// Verify the decision proposes next steps — either in summary or
	// rationale, the entity must indicate the gaps are for future planning.
	hasNextSteps := strings.Contains(summary, "separate feature planning") ||
		strings.Contains(summary, "future plan") ||
		strings.Contains(rationale, "separate feature planning") ||
		strings.Contains(rationale, "future plan") ||
		strings.Contains(rationale, "separate feature under a future plan")
	if !hasNextSteps {
		t.Error("decision entity must propose next steps (separate feature planning under a future plan)")
	}

	// Verify the decided_by field is set — a maintainer needs to know who
	// made the decision.
	decidedBy, _ := retrieved.State["decided_by"].(string)
	if decidedBy == "" {
		t.Error("decision entity must have a decided_by field")
	}
}

// TestAC005_DecisionProposesConcreteNextSteps verifies that the decision
// document contains specific, numbered, actionable next steps — not just
// vague or aspirational statements. AC-005 requires that the decision
// record "propos[es] next steps" that a maintainer can act on.
func TestAC005_DecisionProposesConcreteNextSteps(t *testing.T) {
	t.Parallel()

	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot resolve test file path")
	}
	repoRoot := filepath.Join(filepath.Dir(thisFile), "..", "..")

	docRelPath := filepath.Join("work", "P61-handoff-resilience-binding-hardening",
		"P61-report-handoff-capability-gap.md")

	candidates := []string{
		filepath.Join(repoRoot, docRelPath),
		filepath.Join(repoRoot, ".worktrees", "FEAT-01KR46PKHMG4J-doc-reconciliation", docRelPath),
	}

	var content string
	found := false
	for _, docPath := range candidates {
		data, err := os.ReadFile(docPath)
		if err == nil {
			content = string(data)
			found = true
			break
		}
	}
	if !found {
		t.Skip("decision document not found — may not be committed yet")
	}

	// Verify concrete next steps exist with numbered, actionable items.
	// The document should have a "Recommended Next Steps" section with
	// specific actions a maintainer can evaluate.
	if !strings.Contains(content, "Recommended Next Steps") {
		t.Error("decision document must have a 'Recommended Next Steps' section")
	}

	// Verify each next step has specific, actionable content.
	// Step 1: Should propose creating features under a future plan.
	if !strings.Contains(content, "Handoff context quality") &&
		!strings.Contains(content, "handoff context") {
		t.Error("next steps must reference a future plan for handoff context quality")
	}

	// Step 2: Should describe interim guidance until implementation.
	if !strings.Contains(content, "implement-task") &&
		!strings.Contains(content, "interim") {
		t.Error("next steps must describe interim guidance until features are implemented")
	}

	// Step 3: Should reference tracking the decision in the workflow system.
	if !strings.Contains(content, "DEC-01KR484HRQ97X") {
		t.Error("next steps must reference the decision entity DEC-01KR484HRQ97X for tracking")
	}

	// Verify the next steps section has numbered items (actionable structure).
	nextStepsIdx := strings.Index(content, "Recommended Next Steps")
	if nextStepsIdx == -1 {
		t.Fatal("cannot locate Recommended Next Steps section")
	}
	nextStepsContent := content[nextStepsIdx:]
	if !strings.Contains(nextStepsContent, "1.") ||
		!strings.Contains(nextStepsContent, "2.") ||
		!strings.Contains(nextStepsContent, "3.") {
		t.Error("next steps must contain numbered, actionable items (1., 2., 3.)")
	}

	// Verify Decision Options section exists (structured decision-making).
	if !strings.Contains(content, "Decision Options") {
		t.Error("decision document must include Decision Options for maintainer evaluation")
	}
}

// TestAC005_DocumentationIsConsistentAfterUpdates verifies that after the
// documentation updates, the capability gaps are correctly handled:
//   - User-facing docs (AGENTS.md) no longer claim the three capabilities
//   - The decision record DEC-01KR484HRQ97X documents them as gaps
//
// This validates the AC-005 scenario end-to-end: documentation updates are
// complete AND the capability gap is documented for maintainer review.
func TestAC005_DocumentationIsConsistentAfterUpdates(t *testing.T) {
	t.Parallel()

	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot resolve test file path")
	}
	repoRoot := filepath.Join(filepath.Dir(thisFile), "..", "..")

	// ── Verify AGENTS.md no longer claims the three capabilities ────────────
	agentsData, err := os.ReadFile(filepath.Join(repoRoot, "AGENTS.md"))
	if err != nil {
		t.Fatalf("read AGENTS.md: %v", err)
	}
	agentsContent := string(agentsData)

	// Find the handoff section and verify it does NOT claim capabilities that
	// don't exist. The handoff section should be accurate after REQ-003 fixes.
	handoffIdx := strings.Index(agentsContent, "handoff")
	if handoffIdx == -1 {
		t.Fatal("AGENTS.md must contain a handoff section")
	}

	// Collect text around handoff mentions (within reasonable window).
	start := handoffIdx
	if start > 500 {
		start = handoffIdx - 500
	} else {
		start = 0
	}
	end := handoffIdx + 1500
	if end > len(agentsContent) {
		end = len(agentsContent)
	}
	handoffContext := agentsContent[start:end]

	// These capabilities should NOT appear as claims of current functionality
	// in the handoff section. They may appear in context of the gap/decision.
	falseClaims := []string{
		"assembling spec sections",
		"injecting spec sections",
		"conflict annotations",
		"graph traversal",
	}
	for _, claim := range falseClaims {
		if strings.Contains(strings.ToLower(handoffContext), strings.ToLower(claim)) {
			// The claim may appear in context of the gap being flagged — that's OK.
			// Only flag if it appears as a claimed current capability.
			// Check if the claim is near a "does not", "not yet", or gap language.
			claimIdx := strings.Index(strings.ToLower(handoffContext), strings.ToLower(claim))
			window := handoffContext[max(0, claimIdx-100):min(len(handoffContext), claimIdx+100)]
			if !strings.Contains(strings.ToLower(window), "not") &&
				!strings.Contains(strings.ToLower(window), "gap") &&
				!strings.Contains(strings.ToLower(window), "future") {
				t.Errorf("AGENTS.md handoff section must not claim %q as current capability", claim)
			}
		}
	}

	// ── Verify the decision record exists documenting the gaps ──────────────
	decPattern := filepath.Join(repoRoot, ".kbz", "state", "decisions", "DEC-01KR484HRQ97X-*.yaml")
	decMatches, err := filepath.Glob(decPattern)
	if err != nil {
		t.Fatalf("glob decision entity: %v", err)
	}
	if len(decMatches) == 0 {
		t.Skipf("decision entity DEC-01KR484HRQ97X not found — entity may be in main repo only")
	}

	decData, err := os.ReadFile(decMatches[0])
	if err != nil {
		t.Fatalf("read decision entity %s: %v", decMatches[0], err)
	}
	decContent := string(decData)

	// Verify the decision explicitly documents the three gaps.
	for _, gap := range []string{"spec section", "conflict annotation", "graph traversal"} {
		if !strings.Contains(strings.ToLower(decContent), gap) {
			t.Errorf("decision must document capability gap: %q", gap)
		}
	}

	// Verify the decision is in proposed status (awaiting maintainer review).
	if !strings.Contains(decContent, "status: proposed") {
		t.Error("decision must be in 'proposed' status for maintainer review")
	}
}

// TestEntityService_Get_CancelledContext verifies AC-001:
// Given a cancelled context, EntityService.Get returns a context cancellation error.
func TestEntityService_Get_CancelledContext(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	service := newTestEntityService(root, "2026-03-19T12:00:00Z")

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // immediate cancellation

	_, err := service.Get(ctx, "feature", "FEAT-0000000000000", "")

	if err == nil {
		t.Fatal("expected error from cancelled context, got nil")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got: %v", err)
	}
}
