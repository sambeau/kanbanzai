package main

import (
	"os"
	"strings"
	"testing"
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
