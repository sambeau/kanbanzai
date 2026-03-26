package main

import (
	"bytes"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"kanbanzai/internal/cache"
	"kanbanzai/internal/model"
	"kanbanzai/internal/service"
	"kanbanzai/internal/validate"
)

func TestRun_NoArgs_PrintsUsage(t *testing.T) {
	deps, output := testDependencies()

	if err := run(nil, deps); err != nil {
		t.Fatalf("run(nil) error = %v", err)
	}

	stdout := output.String()
	if !strings.Contains(stdout, "Phase 4b workflow kernel CLI.") {
		t.Fatalf("stdout missing usage header:\n%s", stdout)
	}
	if !strings.Contains(stdout, "create     Create a Phase 1 entity") {
		t.Fatalf("stdout missing create command:\n%s", stdout)
	}
}

func TestRun_Version_PrintsVersion(t *testing.T) {
	deps, output := testDependencies()

	if err := run([]string{"version"}, deps); err != nil {
		t.Fatalf("run(version) error = %v", err)
	}

	if got, want := strings.TrimSpace(output.String()), "kanbanzai dev"; got != want {
		t.Fatalf("version output = %q, want %q", got, want)
	}
}

func TestRun_VersionFlag_PrintsVersion(t *testing.T) {
	deps, output := testDependencies()

	if err := run([]string{"--version"}, deps); err != nil {
		t.Fatalf("run(--version) error = %v", err)
	}

	if got, want := strings.TrimSpace(output.String()), "kanbanzai dev"; got != want {
		t.Fatalf("version output = %q, want %q", got, want)
	}
}

func TestRun_VersionShortFlag_PrintsVersion(t *testing.T) {
	deps, output := testDependencies()

	if err := run([]string{"-v"}, deps); err != nil {
		t.Fatalf("run(-v) error = %v", err)
	}

	if got, want := strings.TrimSpace(output.String()), "kanbanzai dev"; got != want {
		t.Fatalf("version output = %q, want %q", got, want)
	}
}

func TestRunCreate_MissingTarget_ReturnsUsageError(t *testing.T) {
	deps, _ := testDependencies()

	err := run([]string{"create"}, deps)
	if err == nil {
		t.Fatal("run(create) error = nil, want non-nil")
	}

	if !strings.Contains(err.Error(), "missing create target") {
		t.Fatalf("error missing target message: %v", err)
	}
	if !strings.Contains(err.Error(), "kanbanzai create <entity>") {
		t.Fatalf("error missing create usage: %v", err)
	}
}

func TestRunCreate_UnknownTarget_ReturnsUsageError(t *testing.T) {
	deps, _ := testDependencies()

	err := run([]string{"create", "unknown"}, deps)
	if err == nil {
		t.Fatal("run(create unknown) error = nil, want non-nil")
	}

	if !strings.Contains(err.Error(), `unknown create target "unknown"`) {
		t.Fatalf("error missing unknown target message: %v", err)
	}
	if !strings.Contains(err.Error(), "Entities:") {
		t.Fatalf("error missing entity list: %v", err)
	}
}

func TestRunCreate_CreatesEntities(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		wantEntity string
		wantSlug   string
		wantPath   string
	}{
		{
			name: "plan",
			args: []string{
				"create", "plan",
				"--prefix", "P",
				"--slug", "phase 1 kernel",
				"--title", "Phase 1 Kernel",
				"--summary", "Build the initial workflow kernel",
				"--created_by", "sam",
			},
			wantEntity: "plan",
			wantSlug:   "phase-1-kernel",
			wantPath:   "test/state/plans/",
		},
		{
			name: "feature",
			args: []string{
				"create", "feature",
				"--slug", "storage layer",
				"--parent", "P1-phase-1-kernel",
				"--summary", "Implement canonical storage",
				"--created_by", "sam",
			},
			wantEntity: "feature",
			wantSlug:   "storage-layer",
			wantPath:   "test/state/features/",
		},
		{
			name: "task",
			args: []string{
				"create", "task",
				"--slug", "write entity files",
				"--parent_feature", "FEAT-01J3K7MXP3RT5",
				"--summary", "Write canonical entity files to disk",
			},
			wantEntity: "task",
			wantSlug:   "write-entity-files",
			wantPath:   "test/state/tasks/",
		},
		{
			name: "bug",
			args: []string{
				"create", "bug",
				"--slug", "bad-yaml-output",
				"--title", "Writer produces unstable YAML",
				"--reported_by", "sam",
				"--observed", "Repeated writes produce different output",
				"--expected", "Repeated writes should be stable",
			},
			wantEntity: "bug",
			wantSlug:   "bad-yaml-output",
			wantPath:   "test/state/bugs/",
		},
		{
			name: "decision",
			args: []string{
				"create", "decision",
				"--slug", "strict-yaml-subset",
				"--summary", "Use a strict canonical YAML subset",
				"--rationale", "Deterministic output is required for Git-friendly state",
				"--decided_by", "sam",
			},
			wantEntity: "decision",
			wantSlug:   "strict-yaml-subset",
			wantPath:   "test/state/decisions/",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			fake := newFakeEntityService()
			deps, output := testDependenciesWithService(fake)

			if err := run(tt.args, deps); err != nil {
				t.Fatalf("run(%v) error = %v", tt.args, err)
			}

			stdout := output.String()
			if !strings.Contains(stdout, "created "+tt.wantEntity) {
				t.Fatalf("stdout missing created message for %q:\n%s", tt.wantEntity, stdout)
			}
			if !strings.Contains(stdout, "slug: "+tt.wantSlug) {
				t.Fatalf("stdout missing slug %q:\n%s", tt.wantSlug, stdout)
			}
			if !strings.Contains(stdout, "path: "+tt.wantPath) {
				t.Fatalf("stdout missing path prefix %q:\n%s", tt.wantPath, stdout)
			}

			lines := strings.Split(strings.TrimSpace(stdout), "\n")
			var idLine string
			for _, line := range lines {
				if strings.HasPrefix(line, "id: ") {
					idLine = line
					break
				}
			}
			if idLine == "" {
				t.Fatalf("stdout missing id line:\n%s", stdout)
			}

			idValue := strings.TrimPrefix(idLine, "id: ")
			if idValue == "" {
				t.Fatalf("stdout contained empty id line:\n%s", stdout)
			}

			switch tt.wantEntity {
			case "plan":
				if !model.IsPlanID(idValue) {
					t.Fatalf("plan id %q does not match plan ID format:\n%s", idValue, stdout)
				}
			case "feature":
				if !strings.HasPrefix(idValue, "FEAT-") {
					t.Fatalf("feature id %q does not have FEAT- prefix:\n%s", idValue, stdout)
				}
			case "task":
				if !strings.HasPrefix(idValue, "TASK-") {
					t.Fatalf("task id %q does not have TASK- prefix:\n%s", idValue, stdout)
				}
			case "bug":
				if !strings.HasPrefix(idValue, "BUG-") {
					t.Fatalf("bug id %q does not have BUG- prefix:\n%s", idValue, stdout)
				}
			case "decision":
				if !strings.HasPrefix(idValue, "DEC-") {
					t.Fatalf("decision id %q does not have DEC- prefix:\n%s", idValue, stdout)
				}
			}

			pathLinePrefix := "path: " + tt.wantPath
			if !strings.Contains(stdout, pathLinePrefix) {
				t.Fatalf("stdout missing path prefix %q:\n%s", pathLinePrefix, stdout)
			}
		})
	}
}

func TestRunGet_MissingTarget_ReturnsUsageError(t *testing.T) {
	deps, _ := testDependencies()

	err := run([]string{"get"}, deps)
	if err == nil {
		t.Fatal("run(get) error = nil, want non-nil")
	}

	if !strings.Contains(err.Error(), "missing get target") {
		t.Fatalf("error missing target message: %v", err)
	}
}

func TestRunGet_UnknownTarget_ReturnsUsageError(t *testing.T) {
	deps, _ := testDependencies()

	err := run([]string{"get", "unknown"}, deps)
	if err == nil {
		t.Fatal("run(get unknown) error = nil, want non-nil")
	}

	if !strings.Contains(err.Error(), `unknown get target "unknown"`) {
		t.Fatalf("error missing unknown target message: %v", err)
	}
}

func TestRunGet_MissingFlags_ReturnsValidationError(t *testing.T) {
	deps, _ := testDependenciesWithService(newFakeEntityService())

	err := run([]string{"get", "feature"}, deps)
	if err == nil {
		t.Fatal("run(get feature missing id) error = nil, want non-nil")
	}

	if !strings.Contains(err.Error(), "entity id is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunGet_PrintsEntityDetails(t *testing.T) {
	fake := newFakeEntityService()
	deps, output := testDependenciesWithService(fake)

	if err := run([]string{"get", "feature", "--id", "FEAT-01J3K7MXP3RT5", "--slug", "storage-layer"}, deps); err != nil {
		t.Fatalf("run(get feature) error = %v", err)
	}

	stdout := output.String()
	if !strings.Contains(stdout, "type: feature") {
		t.Fatalf("stdout missing type:\n%s", stdout)
	}
	if !strings.Contains(stdout, "id: FEAT-01J3K-7MXP3RT5") {
		t.Fatalf("stdout missing id:\n%s", stdout)
	}
	if !strings.Contains(stdout, "slug: storage-layer") {
		t.Fatalf("stdout missing slug:\n%s", stdout)
	}
	if !strings.Contains(stdout, "status: draft") {
		t.Fatalf("stdout missing status:\n%s", stdout)
	}
}

func TestRunGet_PrefixResolution(t *testing.T) {
	fake := newFakeEntityService()
	deps, output := testDependenciesWithService(fake)

	// Get feature with --id only (no --slug) — prefix resolution in fake service
	if err := run([]string{"get", "feature", "--id", "FEAT-01J3K7"}, deps); err != nil {
		t.Fatalf("run(get feature with prefix) error = %v", err)
	}

	stdout := output.String()
	if !strings.Contains(stdout, "type: feature") {
		t.Fatalf("stdout missing type:\n%s", stdout)
	}
	if !strings.Contains(stdout, "id: FEAT-01J3K-7MXP3RT5") {
		t.Fatalf("stdout missing id:\n%s", stdout)
	}
	if !strings.Contains(stdout, "slug: storage-layer") {
		t.Fatalf("stdout missing slug:\n%s", stdout)
	}
}

func TestRunList_MissingTarget_ReturnsUsageError(t *testing.T) {
	deps, _ := testDependencies()

	err := run([]string{"list"}, deps)
	if err == nil {
		t.Fatal("run(list) error = nil, want non-nil")
	}

	if !strings.Contains(err.Error(), "missing list target") {
		t.Fatalf("error missing target message: %v", err)
	}
}

func TestRunList_UnknownTarget_ReturnsUsageError(t *testing.T) {
	deps, _ := testDependencies()

	err := run([]string{"list", "unknown"}, deps)
	if err == nil {
		t.Fatal("run(list unknown) error = nil, want non-nil")
	}

	if !strings.Contains(err.Error(), `unknown list target "unknown"`) {
		t.Fatalf("error missing unknown target message: %v", err)
	}
}

func TestRunList_PrintsEntityCountAndEntries(t *testing.T) {
	fake := newFakeEntityService()
	deps, output := testDependenciesWithService(fake)

	if err := run([]string{"list", "features"}, deps); err != nil {
		t.Fatalf("run(list features) error = %v", err)
	}

	stdout := output.String()
	if !strings.Contains(stdout, "listed feature") {
		t.Fatalf("stdout missing list header:\n%s", stdout)
	}
	if !strings.Contains(stdout, "storage-layer") {
		t.Fatalf("stdout missing first feature slug:\n%s", stdout)
	}
	if !strings.Contains(stdout, "validation-engine") {
		t.Fatalf("stdout missing second feature slug:\n%s", stdout)
	}
}

func TestRunHealth_PrintsSummary(t *testing.T) {
	fake := newFakeEntityService()
	deps, output := testDependenciesWithService(fake)

	if err := run([]string{"health"}, deps); err != nil {
		t.Fatalf("run(health) error = %v", err)
	}

	stdout := output.String()
	if !strings.Contains(stdout, "health check") {
		t.Fatalf("stdout missing health header:\n%s", stdout)
	}
	if !strings.Contains(stdout, "entities: 2") {
		t.Fatalf("stdout missing entity count:\n%s", stdout)
	}
}

func TestRunValidate_PrintsCandidateValidationResult(t *testing.T) {
	fake := newFakeEntityService()
	deps, output := testDependenciesWithService(fake)

	err := run([]string{
		"validate",
		"--type", "feature",
		"--id", "FEAT-01J3K7MXP3RT5",
		"--slug", "storage-layer",
		"--epic", "EPIC-TESTEPIC",
		"--status", "draft",
		"--summary", "Implement storage",
		"--created", "2025-01-15",
		"--created_by", "sam",
	}, deps)
	if err != nil {
		t.Fatalf("run(validate) error = %v", err)
	}

	stdout := output.String()
	if !strings.Contains(stdout, "candidate is valid") {
		t.Fatalf("stdout missing validation success:\n%s", stdout)
	}
}

func TestRunUpdateStatus_MissingTarget_ReturnsUsageError(t *testing.T) {
	deps, _ := testDependencies()

	err := run([]string{"update", "status"}, deps)
	if err == nil {
		t.Fatal("run(update status) error = nil, want non-nil")
	}

	if !strings.Contains(err.Error(), "type is required") {
		t.Fatalf("error missing target message: %v", err)
	}
}

func TestRunUpdateStatus_UnknownTarget_ReturnsUsageError(t *testing.T) {
	deps, _ := testDependencies()

	err := run([]string{"update", "status", "unknown"}, deps)
	if err == nil {
		t.Fatal("run(update status unknown) error = nil, want non-nil")
	}

	if !strings.Contains(err.Error(), `unexpected argument "unknown"`) {
		t.Fatalf("error missing unknown target message: %v", err)
	}
}

func TestRunUpdateStatus_MissingRequiredFlags_ReturnsValidationError(t *testing.T) {
	deps, _ := testDependenciesWithService(newFakeEntityService())

	err := run([]string{
		"update", "status",
		"--type", "feature",
		"--id", "FEAT-01J3K7MXP3RT5",
		"--slug", "storage-layer",
	}, deps)
	if err == nil {
		t.Fatal("run(update status feature missing status) error = nil, want non-nil")
	}

	if !strings.Contains(err.Error(), "status is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunUpdateStatus_UpdatesEntityStatus(t *testing.T) {
	fake := newFakeEntityService()
	deps, output := testDependenciesWithService(fake)

	err := run([]string{
		"update", "status",
		"--type", "feature",
		"--id", "FEAT-01J3K9ABC5DE7",
		"--slug", "status-updates",
		"--status", "in-review",
	}, deps)
	if err != nil {
		t.Fatalf("run(update status feature) error = %v", err)
	}

	updateOutput := output.String()
	if !strings.Contains(updateOutput, "updated feature") {
		t.Fatalf("stdout missing update header:\n%s", updateOutput)
	}
	if !strings.Contains(updateOutput, "status: in-review") {
		t.Fatalf("stdout missing updated status:\n%s", updateOutput)
	}
	if !strings.Contains(updateOutput, "id: FEAT-01J3K-9ABC5DE7") {
		t.Fatalf("stdout missing updated id:\n%s", updateOutput)
	}
}

func TestRunUpdateStatus_RejectsIllegalTransition(t *testing.T) {
	fake := newFakeEntityService()
	deps, _ := testDependenciesWithService(fake)

	err := run([]string{
		"update", "status",
		"--type", "epic",
		"--id", "EPIC-TESTEPIC",
		"--slug", "phase-1-kernel",
		"--status", "done",
	}, deps)
	if err == nil {
		t.Fatal("run(update status epic invalid jump) error = nil, want non-nil")
	}

	if !strings.Contains(err.Error(), `invalid epic transition "proposed" -> "done"`) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunFeature_MissingSubcommand_ReturnsUsageError(t *testing.T) {
	deps, _ := testDependencies()

	err := run([]string{"feature"}, deps)
	if err == nil {
		t.Fatal("run(feature) error = nil, want non-nil")
	}

	if !strings.Contains(err.Error(), "missing feature subcommand") {
		t.Fatalf("error missing subcommand message: %v", err)
	}
}

func TestRunFeature_UnknownSubcommand_ReturnsUsageError(t *testing.T) {
	deps, _ := testDependencies()

	err := run([]string{"feature", "explode"}, deps)
	if err == nil {
		t.Fatal("run(feature explode) error = nil, want non-nil")
	}

	if !strings.Contains(err.Error(), "unknown feature subcommand") {
		t.Fatalf("error missing unknown subcommand message: %v", err)
	}
}

func TestRunIncident_MissingSubcommand_ReturnsUsageError(t *testing.T) {
	deps, _ := testDependencies()

	err := run([]string{"incident"}, deps)
	if err == nil {
		t.Fatal("run(incident) error = nil, want non-nil")
	}

	if !strings.Contains(err.Error(), "missing incident subcommand") {
		t.Fatalf("error missing subcommand message: %v", err)
	}
}

func TestRunIncident_UnknownSubcommand_ReturnsUsageError(t *testing.T) {
	deps, _ := testDependencies()

	err := run([]string{"incident", "explode"}, deps)
	if err == nil {
		t.Fatal("run(incident explode) error = nil, want non-nil")
	}

	if !strings.Contains(err.Error(), "unknown incident subcommand") {
		t.Fatalf("error missing unknown subcommand message: %v", err)
	}
}

func TestRunIncidentCreate_MissingSlug_ReturnsUsageError(t *testing.T) {
	deps, _ := testDependencies()

	err := run([]string{"incident", "create", "--title", "Test"}, deps)
	if err == nil {
		t.Fatal("run(incident create missing slug) error = nil, want non-nil")
	}

	if !strings.Contains(err.Error(), "--slug is required") {
		t.Fatalf("error missing slug message: %v", err)
	}
}

func TestRunIncidentShow_MissingID_ReturnsUsageError(t *testing.T) {
	deps, _ := testDependencies()

	err := run([]string{"incident", "show"}, deps)
	if err == nil {
		t.Fatal("run(incident show) error = nil, want non-nil")
	}

	if !strings.Contains(err.Error(), "missing incident ID") {
		t.Fatalf("error missing incident ID message: %v", err)
	}
}

func TestRunFeatureDecompose_MissingID_ReturnsUsageError(t *testing.T) {
	deps, _ := testDependencies()

	err := run([]string{"feature", "decompose"}, deps)
	if err == nil {
		t.Fatal("run(feature decompose) error = nil, want non-nil")
	}

	if !strings.Contains(err.Error(), "missing feature ID") {
		t.Fatalf("error missing feature ID message: %v", err)
	}
}

func TestParseFlags(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		want    map[string]string
		wantErr bool
	}{
		{
			name: "space separated",
			args: []string{"--slug", "phase-1", "--summary", "Build kernel"},
			want: map[string]string{
				"slug":    "phase-1",
				"summary": "Build kernel",
			},
		},
		{
			name: "equals syntax",
			args: []string{"--slug=phase-1", "--created_by=sam"},
			want: map[string]string{
				"slug":       "phase-1",
				"created_by": "sam",
			},
		},
		{
			name:    "missing value",
			args:    []string{"--slug"},
			wantErr: true,
		},
		{
			name:    "unexpected positional argument",
			args:    []string{"slug"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseFlags(tt.args)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("parseFlags(%v) error = nil, want non-nil", tt.args)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseFlags(%v) error = %v", tt.args, err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("parseFlags(%v) = %#v, want %#v", tt.args, got, tt.want)
			}
		})
	}
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	fn()

	return currentTestStdout.String()
}

func testDependencies() (dependencies, *bytes.Buffer) {
	return testDependenciesWithService(newFakeEntityService())
}

func testDependenciesWithService(svc entityService) (dependencies, *bytes.Buffer) {
	buf := &bytes.Buffer{}
	currentTestStdout = buf

	return dependencies{
		stdout: buf,
		stdin:  strings.NewReader(""),
		newEntityService: func(root string) entityService {
			return svc
		},
	}, buf
}

var currentTestStdout *bytes.Buffer

type fakeEntityService struct {
	listResults map[string][]service.ListResult
	getResults  map[string]service.GetResult
}

func newFakeEntityService() *fakeEntityService {
	return &fakeEntityService{
		listResults: map[string][]service.ListResult{
			"feature": {
				{
					Type: "feature",
					ID:   "FEAT-01J3K7MXP3RT5",
					Slug: "storage-layer",
					Path: "test/state/features/FEAT-01J3K7MXP3RT5-storage-layer.yaml",
					State: map[string]any{
						"status": "draft",
					},
				},
				{
					Type: "feature",
					ID:   "FEAT-01J3K8NPQ4RS6",
					Slug: "validation-engine",
					Path: "test/state/features/FEAT-01J3K8NPQ4RS6-validation-engine.yaml",
					State: map[string]any{
						"status": "draft",
					},
				},
			},
		},
		getResults: map[string]service.GetResult{
			"feature:FEAT-01J3K7MXP3RT5:storage-layer": {
				Type: "feature",
				ID:   "FEAT-01J3K7MXP3RT5",
				Slug: "storage-layer",
				Path: "test/state/features/FEAT-01J3K7MXP3RT5-storage-layer.yaml",
				State: map[string]any{
					"status": "draft",
				},
			},
			"epic:EPIC-TESTEPIC:phase-1-kernel": {
				Type: "epic",
				ID:   "EPIC-TESTEPIC",
				Slug: "phase-1-kernel",
				Path: "test/state/epics/EPIC-TESTEPIC-phase-1-kernel.yaml",
				State: map[string]any{
					"status": "proposed",
				},
			},
			"feature:FEAT-01J3K9ABC5DE7:status-updates": {
				Type: "feature",
				ID:   "FEAT-01J3K9ABC5DE7",
				Slug: "status-updates",
				Path: "test/state/features/FEAT-01J3K9ABC5DE7-status-updates.yaml",
				State: map[string]any{
					"status": "draft",
				},
			},
		},
	}
}

func (f *fakeEntityService) CreatePlan(input service.CreatePlanInput) (service.CreateResult, error) {
	slug := normalizeTestSlug(input.Slug)
	return service.CreateResult{
		Type: "plan",
		ID:   "P1-" + slug,
		Slug: slug,
		Path: "test/state/plans/P1-" + slug + ".yaml",
		State: map[string]any{
			"status": "proposed",
		},
	}, nil
}

func (f *fakeEntityService) GetPlan(id string) (service.ListResult, error) {
	return service.ListResult{
		Type:  "plan",
		ID:    id,
		Slug:  "phase-1-kernel",
		Path:  "test/state/plans/" + id + ".yaml",
		State: map[string]any{"status": "proposed"},
	}, nil
}

func (f *fakeEntityService) ListPlans(filters service.PlanFilters) ([]service.ListResult, error) {
	return nil, nil
}

func (f *fakeEntityService) CreateEpic(input service.CreateEpicInput) (service.CreateResult, error) {
	return service.CreateResult{
		Type: "epic",
		ID:   "EPIC-TESTEPIC",
		Slug: "phase-1-kernel",
		Path: "test/state/epics/EPIC-TESTEPIC-phase-1-kernel.yaml",
		State: map[string]any{
			"status": "proposed",
		},
	}, nil
}

func (f *fakeEntityService) CreateFeature(input service.CreateFeatureInput) (service.CreateResult, error) {
	slug := normalizeTestSlug(input.Slug)
	result := service.CreateResult{
		Type: "feature",
		ID:   "FEAT-01J3K7MXP3RT5",
		Slug: slug,
		Path: "test/state/features/FEAT-01J3K7MXP3RT5-" + slug + ".yaml",
		State: map[string]any{
			"status": "draft",
		},
	}
	if slug == "validation-engine" {
		result.ID = "FEAT-01J3K8NPQ4RS6"
		result.Path = "test/state/features/FEAT-01J3K8NPQ4RS6-validation-engine.yaml"
	}
	if slug == "status-updates" {
		result.ID = "FEAT-01J3K9ABC5DE7"
		result.Path = "test/state/features/FEAT-01J3K9ABC5DE7-status-updates.yaml"
	}
	return result, nil
}

func (f *fakeEntityService) CreateTask(input service.CreateTaskInput) (service.CreateResult, error) {
	slug := normalizeTestSlug(input.Slug)
	return service.CreateResult{
		Type: "task",
		ID:   "TASK-01J3KZZZBB4KF",
		Slug: slug,
		Path: "test/state/tasks/TASK-01J3KZZZBB4KF-" + slug + ".yaml",
		State: map[string]any{
			"status": "queued",
		},
	}, nil
}

func (f *fakeEntityService) CreateBug(input service.CreateBugInput) (service.CreateResult, error) {
	slug := normalizeTestSlug(input.Slug)
	return service.CreateResult{
		Type: "bug",
		ID:   "BUG-01J4AR7WHN4F2",
		Slug: slug,
		Path: "test/state/bugs/BUG-01J4AR7WHN4F2-" + slug + ".yaml",
		State: map[string]any{
			"status": "reported",
		},
	}, nil
}

func (f *fakeEntityService) CreateDecision(input service.CreateDecisionInput) (service.CreateResult, error) {
	slug := normalizeTestSlug(input.Slug)
	return service.CreateResult{
		Type: "decision",
		ID:   "DEC-01J3KABCDE7MX",
		Slug: slug,
		Path: "test/state/decisions/DEC-01J3KABCDE7MX-" + slug + ".yaml",
		State: map[string]any{
			"status": "proposed",
		},
	}, nil
}

func (f *fakeEntityService) Get(entityType, entityID, slug string) (service.GetResult, error) {
	if strings.TrimSpace(entityType) == "" {
		return service.GetResult{}, &testError{"entity type is required"}
	}
	if strings.TrimSpace(entityID) == "" {
		return service.GetResult{}, &testError{"entity id is required"}
	}

	if strings.TrimSpace(slug) != "" {
		key := entityType + ":" + entityID + ":" + slug
		if result, ok := f.getResults[key]; ok {
			return result, nil
		}
		return service.GetResult{}, &testError{"entity not found"}
	}

	// Prefix resolution: find by type and ID prefix
	var matches []service.GetResult
	for key, result := range f.getResults {
		parts := strings.SplitN(key, ":", 3)
		if parts[0] == entityType && strings.HasPrefix(parts[1], entityID) {
			matches = append(matches, result)
		}
	}
	switch len(matches) {
	case 0:
		return service.GetResult{}, &testError{fmt.Sprintf("no %s entity found matching prefix %q", entityType, entityID)}
	case 1:
		return matches[0], nil
	default:
		return service.GetResult{}, &testError{fmt.Sprintf("ambiguous prefix %q for %s", entityID, entityType)}
	}
}

func (f *fakeEntityService) List(entityType string) ([]service.ListResult, error) {
	if strings.TrimSpace(entityType) == "" {
		return nil, &testError{"entity type is required"}
	}
	return f.listResults[entityType], nil
}

func (f *fakeEntityService) UpdateStatus(input service.UpdateStatusInput) (service.GetResult, error) {
	if strings.TrimSpace(input.Type) == "" {
		return service.GetResult{}, &testError{"type is required"}
	}
	if strings.TrimSpace(input.ID) == "" {
		return service.GetResult{}, &testError{"id is required"}
	}
	if strings.TrimSpace(input.Status) == "" {
		return service.GetResult{}, &testError{"status is required"}
	}

	if input.Type == "epic" && input.ID == "EPIC-TESTEPIC" && input.Slug == "phase-1-kernel" && input.Status == "done" {
		return service.GetResult{}, &testError{`invalid epic transition "proposed" -> "done"`}
	}

	result, err := f.Get(input.Type, input.ID, input.Slug)
	if err != nil {
		return service.GetResult{}, err
	}
	result.State["status"] = input.Status
	return result, nil
}

func (f *fakeEntityService) UpdateEntity(input service.UpdateEntityInput) (service.GetResult, error) {
	if strings.TrimSpace(input.Type) == "" {
		return service.GetResult{}, &testError{"type is required"}
	}
	if strings.TrimSpace(input.ID) == "" {
		return service.GetResult{}, &testError{"id is required"}
	}
	if _, ok := input.Fields["id"]; ok {
		return service.GetResult{}, &testError{"cannot update id: field is immutable"}
	}
	if _, ok := input.Fields["status"]; ok {
		return service.GetResult{}, &testError{"cannot update status: use update_status instead"}
	}

	result, err := f.Get(input.Type, input.ID, input.Slug)
	if err != nil {
		return service.GetResult{}, err
	}
	for k, v := range input.Fields {
		result.State[k] = v
	}
	return result, nil
}

func (f *fakeEntityService) RebuildCache() (int, error) {
	return 0, nil
}

func (f *fakeEntityService) SetCache(c *cache.Cache) {
}

func (f *fakeEntityService) ValidateCandidate(entityType string, fields map[string]any) []validate.ValidationError {
	return nil
}

func (f *fakeEntityService) HealthCheck() (*validate.HealthReport, error) {
	return &validate.HealthReport{
		Summary: validate.HealthSummary{
			TotalEntities: 2,
			ErrorCount:    0,
			WarningCount:  0,
			EntitiesByType: map[string]int{
				"feature": 2,
			},
		},
	}, nil
}

func normalizeTestSlug(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	value = strings.ReplaceAll(value, " ", "-")
	for strings.Contains(value, "--") {
		value = strings.ReplaceAll(value, "--", "-")
	}
	return strings.Trim(value, "-")
}

type testError struct {
	message string
}

func (e *testError) Error() string {
	return e.message
}
