package main

import (
	"bytes"
	"reflect"
	"strings"
	"testing"
	"time"

	"kanbanzai/internal/cache"
	"kanbanzai/internal/document"
	"kanbanzai/internal/service"
	"kanbanzai/internal/validate"
)

func TestRun_NoArgs_PrintsUsage(t *testing.T) {
	deps, output := testDependencies()

	if err := run(nil, deps); err != nil {
		t.Fatalf("run(nil) error = %v", err)
	}

	stdout := output.String()
	if !strings.Contains(stdout, "Phase 1 workflow kernel CLI.") {
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

	if got, want := strings.TrimSpace(output.String()), "kanbanzai phase-1-dev"; got != want {
		t.Fatalf("version output = %q, want %q", got, want)
	}
}

func TestRun_VersionFlag_PrintsVersion(t *testing.T) {
	deps, output := testDependencies()

	if err := run([]string{"--version"}, deps); err != nil {
		t.Fatalf("run(--version) error = %v", err)
	}

	if got, want := strings.TrimSpace(output.String()), "kanbanzai phase-1-dev"; got != want {
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
			name: "epic",
			args: []string{
				"create", "epic",
				"--slug", "phase 1 kernel",
				"--title", "Phase 1 Kernel",
				"--summary", "Build the initial workflow kernel",
				"--created_by", "sam",
			},
			wantEntity: "epic",
			wantSlug:   "phase-1-kernel",
			wantPath:   "test/state/epics/",
		},
		{
			name: "feature",
			args: []string{
				"create", "feature",
				"--slug", "storage layer",
				"--epic", "E-001",
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
				"--feature", "FEAT-001",
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
			case "epic":
				if !strings.HasPrefix(idValue, "E-") {
					t.Fatalf("epic id %q does not have E- prefix:\n%s", idValue, stdout)
				}
			case "feature":
				if !strings.HasPrefix(idValue, "FEAT-") {
					t.Fatalf("feature id %q does not have FEAT- prefix:\n%s", idValue, stdout)
				}
			case "task":
				if !strings.HasPrefix(idValue, "FEAT-") || !strings.Contains(idValue, ".") {
					t.Fatalf("task id %q is not feature-local:\n%s", idValue, stdout)
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

	err := run([]string{"get", "feature", "--id", "FEAT-001"}, deps)
	if err == nil {
		t.Fatal("run(get feature missing slug) error = nil, want non-nil")
	}

	if !strings.Contains(err.Error(), "entity slug is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunGet_PrintsEntityDetails(t *testing.T) {
	fake := newFakeEntityService()
	deps, output := testDependenciesWithService(fake)

	if err := run([]string{"get", "feature", "--id", "FEAT-001", "--slug", "storage-layer"}, deps); err != nil {
		t.Fatalf("run(get feature) error = %v", err)
	}

	stdout := output.String()
	if !strings.Contains(stdout, "type: feature") {
		t.Fatalf("stdout missing type:\n%s", stdout)
	}
	if !strings.Contains(stdout, "id: FEAT-001") {
		t.Fatalf("stdout missing id:\n%s", stdout)
	}
	if !strings.Contains(stdout, "slug: storage-layer") {
		t.Fatalf("stdout missing slug:\n%s", stdout)
	}
	if !strings.Contains(stdout, "status: draft") {
		t.Fatalf("stdout missing status:\n%s", stdout)
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

func TestRunDoc_SubmitApproveRetrieveValidateAndList(t *testing.T) {
	fakeEntity := newFakeEntityService()
	fakeDoc := newFakeDocService()
	deps, output := testDependenciesWithServices(fakeEntity, fakeDoc)

	submitArgs := []string{
		"doc", "submit",
		"--type", "proposal",
		"--title", "Test Proposal",
		"--created_by", "sam",
		"--body", "# Test Proposal\n\n## Summary\n\nA summary.\n\n## Problem\n\nA problem.\n\n## Proposal\n\nA proposal.\n",
	}
	if err := run(submitArgs, deps); err != nil {
		t.Fatalf("run(doc submit) error = %v", err)
	}

	stdout := output.String()
	if !strings.Contains(stdout, "submitted document") {
		t.Fatalf("stdout missing submit header:\n%s", stdout)
	}
	if !strings.Contains(stdout, "id: DOC-001") {
		t.Fatalf("stdout missing submitted doc id:\n%s", stdout)
	}
	if !strings.Contains(stdout, "type: proposal") {
		t.Fatalf("stdout missing submitted doc type:\n%s", stdout)
	}

	output.Reset()

	approveArgs := []string{
		"doc", "approve",
		"--type", "proposal",
		"--id", "DOC-001",
		"--approved_by", "reviewer",
	}
	if err := run(approveArgs, deps); err != nil {
		t.Fatalf("run(doc approve) error = %v", err)
	}

	stdout = output.String()
	if !strings.Contains(stdout, "approved document") {
		t.Fatalf("stdout missing approve header:\n%s", stdout)
	}
	if !strings.Contains(stdout, "status: approved") {
		t.Fatalf("stdout missing approved status:\n%s", stdout)
	}

	output.Reset()

	retrieveArgs := []string{
		"doc", "retrieve",
		"--type", "proposal",
		"--id", "DOC-001",
	}
	if err := run(retrieveArgs, deps); err != nil {
		t.Fatalf("run(doc retrieve) error = %v", err)
	}

	stdout = output.String()
	if !strings.Contains(stdout, "## Summary") {
		t.Fatalf("stdout missing retrieved body:\n%s", stdout)
	}

	output.Reset()

	validateArgs := []string{
		"doc", "validate",
		"--type", "proposal",
		"--id", "DOC-001",
	}
	if err := run(validateArgs, deps); err != nil {
		t.Fatalf("run(doc validate) error = %v", err)
	}

	stdout = output.String()
	if !strings.Contains(stdout, "document is valid") {
		t.Fatalf("stdout missing validation success:\n%s", stdout)
	}

	output.Reset()

	listArgs := []string{
		"doc", "list",
		"--type", "proposal",
	}
	if err := run(listArgs, deps); err != nil {
		t.Fatalf("run(doc list) error = %v", err)
	}

	stdout = output.String()
	if !strings.Contains(stdout, "listed documents") {
		t.Fatalf("stdout missing document list header:\n%s", stdout)
	}
	if !strings.Contains(stdout, "DOC-001") {
		t.Fatalf("stdout missing listed doc id:\n%s", stdout)
	}
}

func TestRunDoc_Scaffold_PrintsTemplate(t *testing.T) {
	fakeEntity := newFakeEntityService()
	fakeDoc := newFakeDocService()
	deps, output := testDependenciesWithServices(fakeEntity, fakeDoc)

	err := run([]string{
		"doc", "scaffold",
		"--type", "proposal",
		"--title", "Scaffolded Proposal",
	}, deps)
	if err != nil {
		t.Fatalf("run(doc scaffold) error = %v", err)
	}

	stdout := output.String()
	if !strings.Contains(stdout, "# Scaffolded Proposal") {
		t.Fatalf("stdout missing scaffold title:\n%s", stdout)
	}
	if !strings.Contains(stdout, "## Summary") {
		t.Fatalf("stdout missing scaffold section:\n%s", stdout)
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
		"--id", "FEAT-001",
		"--slug", "storage-layer",
		"--epic", "E-001",
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
		"--id", "FEAT-001",
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
		"--id", "FEAT-007",
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
	if !strings.Contains(updateOutput, "id: FEAT-007") {
		t.Fatalf("stdout missing updated id:\n%s", updateOutput)
	}
}

func TestRunUpdateStatus_RejectsIllegalTransition(t *testing.T) {
	fake := newFakeEntityService()
	deps, _ := testDependenciesWithService(fake)

	err := run([]string{
		"update", "status",
		"--type", "epic",
		"--id", "E-001",
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
	return testDependenciesWithServices(svc, newFakeDocService())
}

func testDependenciesWithServices(entitySvc entityService, docSvc docService) (dependencies, *bytes.Buffer) {
	buf := &bytes.Buffer{}
	currentTestStdout = buf

	return dependencies{
		stdout: buf,
		stdin:  strings.NewReader(""),
		newEntityService: func(root string) entityService {
			return entitySvc
		},
		newDocService: func(root string) docService {
			return docSvc
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
					ID:   "FEAT-001",
					Slug: "storage-layer",
					Path: "test/state/features/FEAT-001-storage-layer.yaml",
					State: map[string]any{
						"status": "draft",
					},
				},
				{
					Type: "feature",
					ID:   "FEAT-002",
					Slug: "validation-engine",
					Path: "test/state/features/FEAT-002-validation-engine.yaml",
					State: map[string]any{
						"status": "draft",
					},
				},
			},
		},
		getResults: map[string]service.GetResult{
			"feature:FEAT-001:storage-layer": {
				Type: "feature",
				ID:   "FEAT-001",
				Slug: "storage-layer",
				Path: "test/state/features/FEAT-001-storage-layer.yaml",
				State: map[string]any{
					"status": "draft",
				},
			},
			"epic:E-001:phase-1-kernel": {
				Type: "epic",
				ID:   "E-001",
				Slug: "phase-1-kernel",
				Path: "test/state/epics/E-001-phase-1-kernel.yaml",
				State: map[string]any{
					"status": "proposed",
				},
			},
			"feature:FEAT-007:status-updates": {
				Type: "feature",
				ID:   "FEAT-007",
				Slug: "status-updates",
				Path: "test/state/features/FEAT-007-status-updates.yaml",
				State: map[string]any{
					"status": "draft",
				},
			},
		},
	}
}

func (f *fakeEntityService) CreateEpic(input service.CreateEpicInput) (service.CreateResult, error) {
	return service.CreateResult{
		Type: "epic",
		ID:   "E-001",
		Slug: "phase-1-kernel",
		Path: "test/state/epics/E-001-phase-1-kernel.yaml",
		State: map[string]any{
			"status": "proposed",
		},
	}, nil
}

func (f *fakeEntityService) CreateFeature(input service.CreateFeatureInput) (service.CreateResult, error) {
	slug := normalizeTestSlug(input.Slug)
	result := service.CreateResult{
		Type: "feature",
		ID:   "FEAT-001",
		Slug: slug,
		Path: "test/state/features/FEAT-001-" + slug + ".yaml",
		State: map[string]any{
			"status": "draft",
		},
	}
	if slug == "validation-engine" {
		result.ID = "FEAT-002"
		result.Path = "test/state/features/FEAT-002-validation-engine.yaml"
	}
	if slug == "status-updates" {
		result.ID = "FEAT-007"
		result.Path = "test/state/features/FEAT-007-status-updates.yaml"
	}
	return result, nil
}

func (f *fakeEntityService) CreateTask(input service.CreateTaskInput) (service.CreateResult, error) {
	slug := normalizeTestSlug(input.Slug)
	return service.CreateResult{
		Type: "task",
		ID:   "FEAT-001.1",
		Slug: slug,
		Path: "test/state/tasks/FEAT-001.1-" + slug + ".yaml",
		State: map[string]any{
			"status": "queued",
		},
	}, nil
}

func (f *fakeEntityService) CreateBug(input service.CreateBugInput) (service.CreateResult, error) {
	slug := normalizeTestSlug(input.Slug)
	return service.CreateResult{
		Type: "bug",
		ID:   "BUG-001",
		Slug: slug,
		Path: "test/state/bugs/BUG-001-" + slug + ".yaml",
		State: map[string]any{
			"status": "reported",
		},
	}, nil
}

func (f *fakeEntityService) CreateDecision(input service.CreateDecisionInput) (service.CreateResult, error) {
	slug := normalizeTestSlug(input.Slug)
	return service.CreateResult{
		Type: "decision",
		ID:   "DEC-001",
		Slug: slug,
		Path: "test/state/decisions/DEC-001-" + slug + ".yaml",
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
	if strings.TrimSpace(slug) == "" {
		return service.GetResult{}, &testError{"entity slug is required"}
	}

	key := entityType + ":" + entityID + ":" + slug
	if result, ok := f.getResults[key]; ok {
		return result, nil
	}

	return service.GetResult{}, &testError{"entity not found"}
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
	if strings.TrimSpace(input.Slug) == "" {
		return service.GetResult{}, &testError{"slug is required"}
	}
	if strings.TrimSpace(input.Status) == "" {
		return service.GetResult{}, &testError{"status is required"}
	}

	if input.Type == "epic" && input.ID == "E-001" && input.Slug == "phase-1-kernel" && input.Status == "done" {
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
	if strings.TrimSpace(input.Slug) == "" {
		return service.GetResult{}, &testError{"slug is required"}
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

type fakeDocService struct {
	docs map[string]document.Document
}

func newFakeDocService() *fakeDocService {
	created := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)
	approvedAt := time.Date(2025, 1, 16, 10, 0, 0, 0, time.UTC)

	return &fakeDocService{
		docs: map[string]document.Document{
			"proposal:DOC-001": {
				Meta: document.DocMeta{
					ID:         "DOC-001",
					Type:       document.DocTypeProposal,
					Title:      "Test Proposal",
					Status:     document.DocStatusApproved,
					CreatedBy:  "sam",
					Created:    created,
					Updated:    approvedAt,
					ApprovedBy: "reviewer",
					ApprovedAt: &approvedAt,
				},
				Body: "# Test Proposal\n\n## Summary\n\nA summary.\n\n## Problem\n\nA problem.\n\n## Proposal\n\nA proposal.\n",
			},
		},
	}
}

func (f *fakeDocService) ScaffoldDocument(docType document.DocType, title string) (string, error) {
	return "# " + title + "\n\n## Summary\n\n", nil
}

func (f *fakeDocService) Submit(input document.SubmitInput) (document.DocumentResult, error) {
	key := string(input.Type) + ":DOC-001"
	now := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)
	f.docs[key] = document.Document{
		Meta: document.DocMeta{
			ID:        "DOC-001",
			Type:      input.Type,
			Title:     input.Title,
			Status:    document.DocStatusSubmitted,
			Feature:   input.Feature,
			CreatedBy: input.CreatedBy,
			Created:   now,
			Updated:   now,
		},
		Body: input.Body,
	}
	return document.DocumentResult{
		ID:     "DOC-001",
		Type:   input.Type,
		Title:  input.Title,
		Status: document.DocStatusSubmitted,
		Path:   "test/docs/DOC-001-test-proposal.md",
	}, nil
}

func (f *fakeDocService) Approve(input document.ApproveInput) (document.DocumentResult, error) {
	key := string(input.Type) + ":" + input.ID
	doc := f.docs[key]
	now := time.Date(2025, 1, 16, 10, 0, 0, 0, time.UTC)
	doc.Meta.Status = document.DocStatusApproved
	doc.Meta.ApprovedBy = input.ApprovedBy
	doc.Meta.ApprovedAt = &now
	doc.Meta.Updated = now
	f.docs[key] = doc

	return document.DocumentResult{
		ID:     input.ID,
		Type:   input.Type,
		Title:  doc.Meta.Title,
		Status: document.DocStatusApproved,
		Path:   "test/docs/DOC-001-test-proposal.md",
	}, nil
}

func (f *fakeDocService) Retrieve(docType document.DocType, id string) (document.Document, error) {
	return f.docs[string(docType)+":"+id], nil
}

func (f *fakeDocService) Validate(doc document.Document) []document.ValidationError {
	return nil
}

func (f *fakeDocService) ListByType(docType document.DocType) ([]document.DocumentResult, error) {
	return []document.DocumentResult{
		{
			ID:     "DOC-001",
			Type:   docType,
			Title:  "Test Proposal",
			Status: document.DocStatusApproved,
			Path:   "test/docs/DOC-001-test-proposal.md",
		},
	}, nil
}

func (f *fakeDocService) ListAll() ([]document.DocumentResult, error) {
	return []document.DocumentResult{
		{
			ID:     "DOC-001",
			Type:   document.DocTypeProposal,
			Title:  "Test Proposal",
			Status: document.DocStatusApproved,
			Path:   "test/docs/DOC-001-test-proposal.md",
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
