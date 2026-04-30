package main

import (
	"os"
	"strings"
	"testing"

	"github.com/sambeau/kanbanzai/internal/cache"
	"github.com/sambeau/kanbanzai/internal/service"
	"github.com/sambeau/kanbanzai/internal/validate"
)

func TestRunStatus_NoArgs_ShowsProjectOverview(t *testing.T) {
	fake := newFakeEntityService()
	deps, output := testDependenciesWithService(fake)

	err := runStatus(nil, deps)
	if err != nil {
		t.Fatalf("runStatus(nil) error = %v", err)
	}

	stdout := output.String()
	if !strings.Contains(stdout, "health check") {
		t.Fatalf("stdout missing health check:\n%s", stdout)
	}
	if !strings.Contains(stdout, "Work queue:") {
		t.Fatalf("stdout missing work queue:\n%s", stdout)
	}
}

func TestRunStatus_InvalidFormat_ReturnsError(t *testing.T) {
	fake := newFakeEntityService()
	deps, _ := testDependenciesWithService(fake)

	err := runStatus([]string{"--format", "xml"}, deps)
	if err == nil {
		t.Fatal("runStatus(--format xml) error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "invalid format") {
		t.Fatalf("error missing 'invalid format': %v", err)
	}
	if !strings.Contains(err.Error(), "human, plain, json") {
		t.Fatalf("error missing valid formats list: %v", err)
	}
}

func TestRunStatus_InvalidFormat_CompactEquals(t *testing.T) {
	fake := newFakeEntityService()
	deps, _ := testDependenciesWithService(fake)

	err := runStatus([]string{"--format=xml"}, deps)
	if err == nil {
		t.Fatal("runStatus(--format=xml) error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "invalid format") {
		t.Fatalf("error missing 'invalid format': %v", err)
	}
}

func TestRunStatus_MultiplePositionalArgs_ReturnsError(t *testing.T) {
	fake := newFakeEntityService()
	deps, _ := testDependenciesWithService(fake)

	err := runStatus([]string{"FEAT-042", "FEAT-043"}, deps)
	if err == nil {
		t.Fatal("runStatus(FEAT-042 FEAT-043) error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "at most one target") {
		t.Fatalf("error missing target count message: %v", err)
	}
}

func TestRunStatus_UnknownFlag_ReturnsError(t *testing.T) {
	fake := newFakeEntityService()
	deps, _ := testDependenciesWithService(fake)

	err := runStatus([]string{"--unknown-flag"}, deps)
	if err == nil {
		t.Fatal("runStatus(--unknown-flag) error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "unknown flag") {
		t.Fatalf("error missing 'unknown flag': %v", err)
	}
}

func TestRunStatus_UnknownFlag_CompactEquals(t *testing.T) {
	fake := newFakeEntityService()
	deps, _ := testDependenciesWithService(fake)

	err := runStatus([]string{"--unknown=value"}, deps)
	if err == nil {
		t.Fatal("runStatus(--unknown=value) error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "unknown flag") {
		t.Fatalf("error missing 'unknown flag': %v", err)
	}
}

func TestRunStatus_FormatRequiresValue(t *testing.T) {
	fake := newFakeEntityService()
	deps, _ := testDependenciesWithService(fake)

	err := runStatus([]string{"--format"}, deps)
	if err == nil {
		t.Fatal("runStatus(--format) error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "--format requires a value") {
		t.Fatalf("error missing required value message: %v", err)
	}
}

func TestRunStatus_ShortFlag(t *testing.T) {
	fake := newFakeEntityService()
	deps, _ := testDependenciesWithService(fake)

	err := runStatus([]string{"-f", "json"}, deps)
	if err != nil {
		t.Fatalf("runStatus(-f json) error = %v", err)
	}
	// No target — should show project overview.
}

func TestRunStatus_ShortFlag_CompactEquals(t *testing.T) {
	fake := newFakeEntityService()
	deps, _ := testDependenciesWithService(fake)

	err := runStatus([]string{"-f=json"}, deps)
	if err != nil {
		t.Fatalf("runStatus(-f=json) error = %v", err)
	}
	// No target — should show project overview.
}

func TestRunStatus_DuplicateFormat_ReturnsError(t *testing.T) {
	fake := newFakeEntityService()
	deps, _ := testDependenciesWithService(fake)

	err := runStatus([]string{"--format", "json", "--format", "human"}, deps)
	if err == nil {
		t.Fatal("runStatus with duplicate --format error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "specified more than once") {
		t.Fatalf("error missing duplicate message: %v", err)
	}
}

func TestRunStatus_ValidFormats_Accepted(t *testing.T) {
	fake := newFakeEntityService()
	deps, _ := testDependenciesWithService(fake)

	for _, fmt := range []string{"human", "plain", "json"} {
		t.Run(fmt, func(t *testing.T) {
			err := runStatus([]string{"--format", fmt}, deps)
			if err != nil {
				t.Fatalf("runStatus(--format %s) error = %v", fmt, err)
			}
		})
	}
}

func TestRunStatus_EntityTarget_DisambiguatesAndRoutes(t *testing.T) {
	fake := newFakeEntityService()
	deps, output := testDependenciesWithService(fake)

	err := runStatus([]string{"FEAT-01J3K7MXP3RT5"}, deps)
	if err != nil {
		t.Fatalf("runStatus(FEAT-01J3K7MXP3RT5) error = %v", err)
	}

	stdout := output.String()
	if !strings.Contains(stdout, "Entity: FEAT-01J3K7MXP3RT5") {
		t.Fatalf("stdout missing entity output:\n%s", stdout)
	}
}

func TestRunStatus_EntityTarget_JSONFormat(t *testing.T) {
	fake := newFakeEntityService()
	deps, output := testDependenciesWithService(fake)

	err := runStatus([]string{"FEAT-01J3K7MXP3RT5", "--format", "json"}, deps)
	if err != nil {
		t.Fatalf("runStatus(FEAT-01J3K7MXP3RT5 --format json) error = %v", err)
	}

	stdout := output.String()
	if !strings.Contains(stdout, `"format":"json"`) {
		t.Fatalf("stdout missing json format marker:\n%s", stdout)
	}
	if !strings.Contains(stdout, `"entity":"FEAT-01J3K7MXP3RT5"`) {
		t.Fatalf("stdout missing entity field:\n%s", stdout)
	}
}

func TestRunStatus_EntityTarget_PlainFormat(t *testing.T) {
	fake := newFakeEntityService()
	deps, output := testDependenciesWithService(fake)

	err := runStatus([]string{"FEAT-01J3K7MXP3RT5", "--format", "plain"}, deps)
	if err != nil {
		t.Fatalf("runStatus(FEAT-01J3K7MXP3RT5 --format plain) error = %v", err)
	}

	stdout := output.String()
	if !strings.Contains(stdout, "FEAT-01J3K7MXP3RT5:") {
		t.Fatalf("stdout missing plain entity output:\n%s", stdout)
	}
}

func TestRunStatus_PlanPrefixTarget_DisambiguatesAndRoutes(t *testing.T) {
	fake := newFakeEntityService()
	deps, output := testDependenciesWithService(fake)

	err := runStatus([]string{"P1"}, deps)
	if err != nil {
		t.Fatalf("runStatus(P1) error = %v", err)
	}

	stdout := output.String()
	if !strings.Contains(stdout, "Plan prefix: P1") {
		t.Fatalf("stdout missing plan prefix output:\n%s", stdout)
	}
	if !strings.Contains(stdout, "Resolved to:") {
		t.Fatalf("stdout missing resolved plan ID:\n%s", stdout)
	}
}

func TestRunStatus_PlanPrefixTarget_JSONFormat(t *testing.T) {
	fake := newFakeEntityService()
	deps, output := testDependenciesWithService(fake)

	err := runStatus([]string{"P1", "--format", "json"}, deps)
	if err != nil {
		t.Fatalf("runStatus(P1 --format json) error = %v", err)
	}

	stdout := output.String()
	if !strings.Contains(stdout, `"plan_prefix":"P1"`) {
		t.Fatalf("stdout missing plan_prefix field:\n%s", stdout)
	}
	if !strings.Contains(stdout, `"format":"json"`) {
		t.Fatalf("stdout missing json format:\n%s", stdout)
	}
}

// ─── File path resolution tests ──────────────────────────────────────────────

func TestRunStatus_FilePathTarget_NonexistentFile(t *testing.T) {
	fake := newFakeEntityService()
	deps, _ := testDependenciesWithService(fake)

	err := runStatus([]string{"work/design/nonexistent.md"}, deps)
	if err == nil {
		t.Fatal("runStatus(work/design/nonexistent.md) error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "file not found") {
		t.Fatalf("error missing 'file not found': %v", err)
	}
}

func TestRunStatus_FilePathTarget_UnregisteredFile(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)
	os.Chdir(tmpDir)

	f, err := os.Create("unregistered.md")
	if err != nil {
		t.Fatal(err)
	}
	f.Close()

	fake := newFakeEntityService()
	deps, output := testDependenciesWithService(fake)

	err = runStatus([]string{"unregistered.md"}, deps)
	if err != nil {
		t.Fatalf("runStatus(unregistered.md) error = %v", err)
	}

	stdout := output.String()
	if !strings.Contains(stdout, "not registered") {
		t.Fatalf("stdout missing 'not registered':\n%s", stdout)
	}
	if !strings.Contains(stdout, "kbz doc register") {
		t.Fatalf("stdout missing register suggestion:\n%s", stdout)
	}
}

func TestRunStatus_FilePathTarget_UnregisteredFile_DotSlashPrefix(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)
	os.Chdir(tmpDir)

	f, err := os.Create("unregistered.md")
	if err != nil {
		t.Fatal(err)
	}
	f.Close()

	fake := newFakeEntityService()
	deps, output := testDependenciesWithService(fake)

	err = runStatus([]string{"./unregistered.md"}, deps)
	if err != nil {
		t.Fatalf("runStatus(./unregistered.md) error = %v", err)
	}

	stdout := output.String()
	if !strings.Contains(stdout, "not registered") {
		t.Fatalf("stdout missing 'not registered' for ./ prefix:\n%s", stdout)
	}
}

func TestRunStatus_FilePathTarget_UnregisteredFile_JSONFormat(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)
	os.Chdir(tmpDir)

	f, err := os.Create("unregistered.md")
	if err != nil {
		t.Fatal(err)
	}
	f.Close()

	fake := newFakeEntityService()
	deps, output := testDependenciesWithService(fake)

	err = runStatus([]string{"unregistered.md", "--format", "json"}, deps)
	if err != nil {
		t.Fatalf("runStatus(unregistered.md --format json) error = %v", err)
	}

	stdout := output.String()
	if !strings.Contains(stdout, `"registered":false`) {
		t.Fatalf("stdout missing registered:false:\n%s", stdout)
	}
	if !strings.Contains(stdout, `"format":"json"`) {
		t.Fatalf("stdout missing json format:\n%s", stdout)
	}
}

// ─── End file path resolution tests ──────────────────────────────────────────

func TestRunStatus_UnrecognisedTarget_ReturnsError(t *testing.T) {
	fake := newFakeEntityService()
	deps, _ := testDependenciesWithService(fake)

	err := runStatus([]string{"sometoken"}, deps)
	if err == nil {
		t.Fatal("runStatus(sometoken) error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "unrecognised target") {
		t.Fatalf("error missing 'unrecognised target': %v", err)
	}
}

func TestRunStatus_FormatBeforeTarget(t *testing.T) {
	fake := newFakeEntityService()
	deps, output := testDependenciesWithService(fake)

	err := runStatus([]string{"--format", "json", "FEAT-01J3K7MXP3RT5"}, deps)
	if err != nil {
		t.Fatalf("runStatus(--format json FEAT-01J3K7MXP3RT5) error = %v", err)
	}

	stdout := output.String()
	if !strings.Contains(stdout, `"entity":"FEAT-01J3K7MXP3RT5"`) {
		t.Fatalf("stdout missing entity with flag-first order:\n%s", stdout)
	}
}

func TestRunStatus_FormatAfterTarget(t *testing.T) {
	fake := newFakeEntityService()
	deps, output := testDependenciesWithService(fake)

	err := runStatus([]string{"FEAT-01J3K7MXP3RT5", "--format", "json"}, deps)
	if err != nil {
		t.Fatalf("runStatus(FEAT-01J3K7MXP3RT5 --format json) error = %v", err)
	}

	stdout := output.String()
	if !strings.Contains(stdout, `"entity":"FEAT-01J3K7MXP3RT5"`) {
		t.Fatalf("stdout missing entity with flag-last order:\n%s", stdout)
	}
}

func TestRunStatus_ErrorExitCode(t *testing.T) {
	// Errors from runStatus should be non-nil (exit code 2 in main).
	fake := newFakeEntityService()
	deps, _ := testDependenciesWithService(fake)

	// Multiple positional args → error
	err := runStatus([]string{"FEAT-042", "FEAT-043"}, deps)
	if err == nil {
		t.Fatal("runStatus with multiple targets should error")
	}

	// Unknown flag → error
	err = runStatus([]string{"--no-such-flag"}, deps)
	if err == nil {
		t.Fatal("runStatus with unknown flag should error")
	}

	// Invalid format → error
	err = runStatus([]string{"--format", "xml"}, deps)
	if err == nil {
		t.Fatal("runStatus with invalid format should error")
	}
}

func TestRunStatus_ViaMain(t *testing.T) {
	// Integration: test that `kbz status` wired through `run` works.
	deps, output := testDependencies()

	err := run([]string{"status"}, deps)
	if err != nil {
		t.Fatalf("run(status) error = %v", err)
	}

	stdout := output.String()
	if !strings.Contains(stdout, "health check") {
		t.Fatalf("stdout missing health check:\n%s", stdout)
	}
}

func TestRunStatus_ViaMain_WithTarget(t *testing.T) {
	deps, output := testDependencies()

	err := run([]string{"status", "FEAT-01J3K7MXP3RT5"}, deps)
	if err != nil {
		t.Fatalf("run(status FEAT-01J3K7MXP3RT5) error = %v", err)
	}

	stdout := output.String()
	if !strings.Contains(stdout, "Entity: FEAT-01J3K7MXP3RT5") {
		t.Fatalf("stdout missing entity output:\n%s", stdout)
	}
}

func TestRunStatus_ViaMain_InvalidFormat(t *testing.T) {
	deps, _ := testDependencies()

	err := run([]string{"status", "--format", "xml"}, deps)
	if err == nil {
		t.Fatal("run(status --format xml) error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "invalid format") {
		t.Fatalf("error missing 'invalid format': %v", err)
	}
}

// ─── Entity not found tests (AC-013) ────────────────────────────────────────

func TestRunStatus_EntityTarget_NotFound(t *testing.T) {
	fake := newFakeEntityService()
	deps, _ := testDependenciesWithService(fake)

	// AC-013: entity ID that matches the pattern but doesn't exist in the store.
	err := runStatus([]string{"FEAT-01ZZZZZZZZZZZ"}, deps)
	if err == nil {
		t.Fatal("runStatus(FEAT-01ZZZZZZZZZZZ) error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "entity not found") {
		t.Fatalf("error missing 'entity not found': %v", err)
	}
}

// ─── Bug entity routing (AC-015) ────────────────────────────────────────────

func TestRunStatus_EntityTarget_BugRouting(t *testing.T) {
	fake := newFakeEntityService()
	// Add a bug with display-format ID to getResults.
	fake.getResults["bug:BUG-007:login-bypass"] = service.GetResult{
		Type: "bug",
		ID:   "BUG-007",
		Slug: "login-bypass",
		Path: "test/state/bugs/BUG-007-login-bypass.yaml",
		State: map[string]any{
			"status":   "reported",
			"severity": "high",
		},
	}
	deps, output := testDependenciesWithService(fake)

	err := runStatus([]string{"BUG-007"}, deps)
	if err != nil {
		t.Fatalf("runStatus(BUG-007) error = %v", err)
	}

	stdout := output.String()
	if !strings.Contains(stdout, "Entity: BUG-007") {
		t.Fatalf("stdout missing bug entity output:\n%s", stdout)
	}
	if !strings.Contains(stdout, "Status: reported") {
		t.Fatalf("stdout missing bug status:\n%s", stdout)
	}
}

func TestRunStatus_EntityTarget_BugNotFound(t *testing.T) {
	fake := newFakeEntityService()
	deps, _ := testDependenciesWithService(fake)

	err := runStatus([]string{"BUG-999"}, deps)
	if err == nil {
		t.Fatal("runStatus(BUG-999) error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "entity not found") {
		t.Fatalf("error missing 'entity not found': %v", err)
	}
}

// ─── State store error (AC-020) ─────────────────────────────────────────────

func TestRunStatus_StateStoreError_HealthCheckFails(t *testing.T) {
	fake := &faultyEntityService{err: &testError{"state store unavailable"}}
	deps, _ := testDependenciesWithService(fake)

	err := runStatus(nil, deps)
	if err == nil {
		t.Fatal("runStatus with faulty health check error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "state store unavailable") {
		t.Fatalf("error missing state store message: %v", err)
	}
}

// ─── Exit code verification (AC-019) ────────────────────────────────────────

func TestRunStatus_ExitCodes(t *testing.T) {
	fake := newFakeEntityService()

	t.Run("no_args_success", func(t *testing.T) {
		deps, _ := testDependenciesWithService(fake)
		if err := runStatus(nil, deps); err != nil {
			t.Errorf("runStatus(nil) error = %v, want nil (exit 0)", err)
		}
	})

	t.Run("entity_found_success", func(t *testing.T) {
		deps, _ := testDependenciesWithService(fake)
		if err := runStatus([]string{"FEAT-01J3K7MXP3RT5"}, deps); err != nil {
			t.Errorf("runStatus(FEAT-...) error = %v, want nil (exit 0)", err)
		}
	})

	t.Run("plan_prefix_success", func(t *testing.T) {
		deps, _ := testDependenciesWithService(fake)
		if err := runStatus([]string{"P1"}, deps); err != nil {
			t.Errorf("runStatus(P1) error = %v, want nil (exit 0)", err)
		}
	})

	t.Run("invalid_format_error", func(t *testing.T) {
		deps, _ := testDependenciesWithService(fake)
		if err := runStatus([]string{"--format", "xml"}, deps); err == nil {
			t.Error("runStatus(--format xml) error = nil, want non-nil (exit 1)")
		}
	})

	t.Run("unknown_flag_error", func(t *testing.T) {
		deps, _ := testDependenciesWithService(fake)
		if err := runStatus([]string{"--bogus"}, deps); err == nil {
			t.Error("runStatus(--bogus) error = nil, want non-nil (exit 1)")
		}
	})

	t.Run("multiple_args_error", func(t *testing.T) {
		deps, _ := testDependenciesWithService(fake)
		if err := runStatus([]string{"FEAT-042", "FEAT-043"}, deps); err == nil {
			t.Error("runStatus(two targets) error = nil, want non-nil (exit 1)")
		}
	})

	t.Run("entity_not_found_error", func(t *testing.T) {
		deps, _ := testDependenciesWithService(fake)
		if err := runStatus([]string{"FEAT-01ZZZZZZZZZZZ"}, deps); err == nil {
			t.Error("runStatus(nonexistent) error = nil, want non-nil (exit 1)")
		}
	})

	t.Run("file_not_found_error", func(t *testing.T) {
		deps, _ := testDependenciesWithService(fake)
		if err := runStatus([]string{"work/design/nonexistent.md"}, deps); err == nil {
			t.Error("runStatus(nonexistent file) error = nil, want non-nil (exit 1)")
		}
	})

	t.Run("unrecognised_target_error", func(t *testing.T) {
		deps, _ := testDependenciesWithService(fake)
		if err := runStatus([]string{"sometoken"}, deps); err == nil {
			t.Error("runStatus(sometoken) error = nil, want non-nil (exit 1)")
		}
	})
}

// ─── Doc approve integration (AC-021 through AC-024 via main) ───────────────

func TestRunDocApprove_ViaMain(t *testing.T) {
	deps, _ := testDependencies()

	err := run([]string{"doc", "approve"}, deps)
	if err == nil {
		t.Fatal("run(doc approve) with no args error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "missing document ID or path") {
		t.Fatalf("error missing expected message: %v", err)
	}
}

// ─── faultyEntityService for testing error paths ─────────────────────────────

type faultyEntityService struct {
	err error
}

func (f *faultyEntityService) CreatePlan(input service.CreatePlanInput) (service.CreateResult, error) {
	return service.CreateResult{}, f.err
}
func (f *faultyEntityService) CreateFeature(input service.CreateFeatureInput) (service.CreateResult, error) {
	return service.CreateResult{}, f.err
}
func (f *faultyEntityService) CreateTask(input service.CreateTaskInput) (service.CreateResult, error) {
	return service.CreateResult{}, f.err
}
func (f *faultyEntityService) CreateBug(input service.CreateBugInput) (service.CreateResult, error) {
	return service.CreateResult{}, f.err
}
func (f *faultyEntityService) CreateDecision(input service.CreateDecisionInput) (service.CreateResult, error) {
	return service.CreateResult{}, f.err
}
func (f *faultyEntityService) GetPlan(id string) (service.ListResult, error) {
	return service.ListResult{}, f.err
}
func (f *faultyEntityService) Get(entityType, entityID, slug string) (service.GetResult, error) {
	return service.GetResult{}, f.err
}
func (f *faultyEntityService) List(entityType string) ([]service.ListResult, error) {
	return nil, f.err
}
func (f *faultyEntityService) ListPlans(filters service.PlanFilters) ([]service.ListResult, error) {
	return nil, f.err
}
func (f *faultyEntityService) UpdateStatus(input service.UpdateStatusInput) (service.GetResult, error) {
	return service.GetResult{}, f.err
}
func (f *faultyEntityService) UpdateEntity(input service.UpdateEntityInput) (service.GetResult, error) {
	return service.GetResult{}, f.err
}
func (f *faultyEntityService) ValidateCandidate(entityType string, fields map[string]any) []validate.ValidationError {
	return nil
}
func (f *faultyEntityService) HealthCheck() (*validate.HealthReport, error) {
	return nil, f.err
}
func (f *faultyEntityService) RebuildCache() (int, error) {
	return 0, f.err
}
func (f *faultyEntityService) SetCache(c *cache.Cache) {
}
func (f *faultyEntityService) WorkQueue(input service.WorkQueueInput) (service.WorkQueueResult, error) {
	return service.WorkQueueResult{}, f.err
}

var _ entityService = (*faultyEntityService)(nil)
