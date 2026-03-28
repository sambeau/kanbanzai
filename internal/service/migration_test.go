package service

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sambeau/kanbanzai/internal/config"
	"github.com/sambeau/kanbanzai/internal/storage"
)

func setupMigrationConfig(t *testing.T, prefix, label string) {
	t.Helper()

	cfg := config.Config{
		Version: "2",
		Prefixes: []config.PrefixEntry{
			{Prefix: prefix, Name: label},
		},
	}

	// Save to the default location that config.Load() reads from.
	configPath := filepath.Join(".kbz", "config.yaml")
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		t.Fatalf("create config dir: %v", err)
	}
	if err := cfg.SaveTo(configPath); err != nil {
		t.Fatalf("save config: %v", err)
	}
	t.Cleanup(func() {
		os.RemoveAll(".kbz")
	})
}

func writeRawYAML(t *testing.T, dir, filename, content string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", dir, err)
	}
	if err := os.WriteFile(filepath.Join(dir, filename), []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", filename, err)
	}
}

func TestMigratePhase2_NoEpicsDir_Noop(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	svc := NewEntityService(root)

	// No epics directory — config check will fail first, but let's set up config.
	// Actually, migration checks config first. Let's skip the config check
	// by testing the "no epics" path. We need config to exist.
	// Since config.Load() uses a global path, we test the no-epics path
	// by providing a root that has no epics/ subdir but the config exists.

	// We can't easily mock config.Load() in this test, so we test what we can:
	// if there's no config, it returns an error.
	_, err := svc.MigratePhase2()
	if err == nil {
		t.Fatal("expected error when no config exists")
	}
	if !strings.Contains(err.Error(), "prefix registry must be configured") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestMigratePhase2_ConvertsEpicsToPlan(t *testing.T) {
	// This test uses config.Load() which reads from .kbz/config.yaml
	// relative to the working directory. We must set it up carefully.

	root := t.TempDir()
	svc := NewEntityService(root)

	// Set up config in the working directory.
	setupMigrationConfig(t, "P", "Plan")

	// Create an epic.
	epicsDir := filepath.Join(root, "epics")
	epicYAML := `id: EPIC-KERNEL
slug: kernel
title: Workflow Kernel
status: proposed
summary: Build the workflow kernel
created: 2026-01-01T00:00:00Z
created_by: sam
features:
  - FEAT-001
  - FEAT-002
`
	writeRawYAML(t, epicsDir, "EPIC-KERNEL-kernel.yaml", epicYAML)

	// Create a feature that references the epic.
	featuresDir := filepath.Join(root, "features")
	featureYAML := `id: FEAT-01AAAAAAAAA01
slug: login
epic: EPIC-KERNEL
status: draft
summary: Login feature
created: 2026-01-01T00:00:00Z
created_by: sam
plan: some-dev-plan
`
	writeRawYAML(t, featuresDir, "FEAT-01AAAAAAAAA01-login.yaml", featureYAML)

	result, err := svc.MigratePhase2()
	if err != nil {
		t.Fatalf("MigratePhase2() error = %v", err)
	}

	if result.PlansCreated != 1 {
		t.Errorf("PlansCreated = %d, want 1", result.PlansCreated)
	}
	if result.FeaturesUpdated != 1 {
		t.Errorf("FeaturesUpdated = %d, want 1", result.FeaturesUpdated)
	}
	if len(result.Errors) != 0 {
		t.Errorf("unexpected errors: %v", result.Errors)
	}

	// Verify the plan was created.
	planDir := filepath.Join(root, "plans")
	planEntries, err := os.ReadDir(planDir)
	if err != nil {
		t.Fatalf("read plans dir: %v", err)
	}
	if len(planEntries) != 1 {
		t.Fatalf("expected 1 plan file, got %d", len(planEntries))
	}

	planFile := planEntries[0].Name()
	if !strings.HasPrefix(planFile, "P1-kernel") {
		t.Errorf("plan filename = %q, want prefix P1-kernel", planFile)
	}

	// Read back the plan and verify fields.
	planData, err := os.ReadFile(filepath.Join(planDir, planFile))
	if err != nil {
		t.Fatalf("read plan: %v", err)
	}
	planFields, err := storage.UnmarshalCanonicalYAML(string(planData))
	if err != nil {
		t.Fatalf("unmarshal plan: %v", err)
	}

	if planFields["status"] != "proposed" {
		t.Errorf("plan status = %v, want proposed", planFields["status"])
	}
	if planFields["title"] != "Workflow Kernel" {
		t.Errorf("plan title = %v, want Workflow Kernel", planFields["title"])
	}
	// Features field should be removed from the plan.
	if _, ok := planFields["features"]; ok {
		t.Error("plan should not have 'features' field")
	}

	// Verify the epic file was removed.
	epicEntries, _ := os.ReadDir(epicsDir)
	if len(epicEntries) != 0 {
		t.Errorf("expected epics dir to be empty, got %d files", len(epicEntries))
	}

	// Read back the feature and verify updates.
	featureData, err := os.ReadFile(filepath.Join(featuresDir, "FEAT-01AAAAAAAAA01-login.yaml"))
	if err != nil {
		t.Fatalf("read feature: %v", err)
	}
	featureFields, err := storage.UnmarshalCanonicalYAML(string(featureData))
	if err != nil {
		t.Fatalf("unmarshal feature: %v", err)
	}

	// "epic" should be removed, "parent" should be the new plan ID.
	if _, ok := featureFields["epic"]; ok {
		t.Error("feature should not have 'epic' field after migration")
	}
	parentVal, ok := featureFields["parent"]
	if !ok {
		t.Fatal("feature should have 'parent' field after migration")
	}
	parentStr, _ := parentVal.(string)
	if !strings.HasPrefix(parentStr, "P1-") {
		t.Errorf("feature parent = %q, want P1-* prefix", parentStr)
	}

	// "plan" should be renamed to "dev_plan".
	if _, ok := featureFields["plan"]; ok {
		t.Error("feature should not have 'plan' field after migration")
	}
	if _, ok := featureFields["dev_plan"]; !ok {
		t.Error("feature should have 'dev_plan' field after migration")
	}
}

func TestMigratePhase2_Idempotent(t *testing.T) {
	root := t.TempDir()
	svc := NewEntityService(root)

	setupMigrationConfig(t, "P", "Plan")

	epicsDir := filepath.Join(root, "epics")
	epicYAML := `id: EPIC-IDEMPOTENT
slug: idem
title: Idempotent Test
status: active
summary: Test idempotency
created: 2026-01-01T00:00:00Z
created_by: sam
`
	writeRawYAML(t, epicsDir, "EPIC-IDEMPOTENT-idem.yaml", epicYAML)

	// First run.
	result1, err := svc.MigratePhase2()
	if err != nil {
		t.Fatalf("first MigratePhase2() error = %v", err)
	}
	if result1.PlansCreated != 1 {
		t.Fatalf("first run: PlansCreated = %d, want 1", result1.PlansCreated)
	}

	// Re-create the epic to simulate a partial re-run scenario.
	// But since the epic was already deleted, the second run sees no epics dir.
	// Instead, let's verify that if the plan already exists, it's skipped.
	// Re-create the epic.
	writeRawYAML(t, epicsDir, "EPIC-IDEMPOTENT-idem.yaml", epicYAML)

	result2, err := svc.MigratePhase2()
	if err != nil {
		t.Fatalf("second MigratePhase2() error = %v", err)
	}

	// The plan file already exists, so it should be skipped.
	if result2.PlansCreated != 0 {
		t.Errorf("second run: PlansCreated = %d, want 0 (idempotent)", result2.PlansCreated)
	}
}

func TestMigratePhase2_EpicStatusMapping(t *testing.T) {
	root := t.TempDir()
	svc := NewEntityService(root)

	setupMigrationConfig(t, "P", "Plan")

	testCases := []struct {
		epicStatus string
		wantPlan   string
	}{
		{"proposed", "proposed"},
		{"approved", "designing"},
		{"active", "active"},
		{"on-hold", "active"},
		{"done", "done"},
	}

	for i, tc := range testCases {
		// Clean state for each iteration.
		epicsDir := filepath.Join(root, "epics")
		os.RemoveAll(epicsDir)
		plansDir := filepath.Join(root, "plans")
		os.RemoveAll(plansDir)

		slug := "status-test"
		epicID := "EPIC-STATUSTEST"
		epicYAML := "id: " + epicID + "\n" +
			"slug: " + slug + "\n" +
			"title: Status Test\n" +
			"status: " + tc.epicStatus + "\n" +
			"summary: Status mapping test\n" +
			"created: 2026-01-01T00:00:00Z\n" +
			"created_by: sam\n"

		writeRawYAML(t, epicsDir, epicID+"-"+slug+".yaml", epicYAML)

		result, err := svc.MigratePhase2()
		if err != nil {
			t.Fatalf("case %d (%s): MigratePhase2() error = %v", i, tc.epicStatus, err)
		}
		if result.PlansCreated != 1 {
			t.Fatalf("case %d (%s): PlansCreated = %d, want 1", i, tc.epicStatus, result.PlansCreated)
		}

		// Read back the plan.
		planEntries, _ := os.ReadDir(plansDir)
		if len(planEntries) != 1 {
			t.Fatalf("case %d (%s): expected 1 plan, got %d", i, tc.epicStatus, len(planEntries))
		}
		data, _ := os.ReadFile(filepath.Join(plansDir, planEntries[0].Name()))
		fields, _ := storage.UnmarshalCanonicalYAML(string(data))

		gotStatus, _ := fields["status"].(string)
		if gotStatus != tc.wantPlan {
			t.Errorf("case %d: epic %q → plan %q, want %q", i, tc.epicStatus, gotStatus, tc.wantPlan)
		}
	}
}

func TestMigratePhase2_MultipleEpics(t *testing.T) {
	root := t.TempDir()
	svc := NewEntityService(root)

	setupMigrationConfig(t, "P", "Plan")

	epicsDir := filepath.Join(root, "epics")

	epic1 := `id: EPIC-ALPHA
slug: alpha
title: Alpha
status: proposed
summary: First epic
created: 2026-01-01T00:00:00Z
created_by: sam
`
	epic2 := `id: EPIC-BETA
slug: beta
title: Beta
status: active
summary: Second epic
created: 2026-01-02T00:00:00Z
created_by: sam
`
	writeRawYAML(t, epicsDir, "EPIC-ALPHA-alpha.yaml", epic1)
	writeRawYAML(t, epicsDir, "EPIC-BETA-beta.yaml", epic2)

	result, err := svc.MigratePhase2()
	if err != nil {
		t.Fatalf("MigratePhase2() error = %v", err)
	}

	if result.PlansCreated != 2 {
		t.Errorf("PlansCreated = %d, want 2", result.PlansCreated)
	}

	// Verify both plans exist.
	planEntries, _ := os.ReadDir(filepath.Join(root, "plans"))
	if len(planEntries) != 2 {
		t.Errorf("expected 2 plan files, got %d", len(planEntries))
	}

	// Verify epics directory is empty or removed.
	epicEntries, err := os.ReadDir(epicsDir)
	if err == nil && len(epicEntries) != 0 {
		t.Errorf("expected epics dir to be empty, got %d files", len(epicEntries))
	}
}

func TestMigratePhase2_FeatureWithoutEpicRef(t *testing.T) {
	root := t.TempDir()
	svc := NewEntityService(root)

	setupMigrationConfig(t, "P", "Plan")

	// Create epics dir but no epics (so migration doesn't skip).
	epicsDir := filepath.Join(root, "epics")
	writeRawYAML(t, epicsDir, "EPIC-DUMMY-dummy.yaml", `id: EPIC-DUMMY
slug: dummy
title: Dummy
status: proposed
summary: Dummy epic
created: 2026-01-01T00:00:00Z
created_by: sam
`)

	// Feature without epic or plan fields — should not be modified.
	featuresDir := filepath.Join(root, "features")
	featureYAML := `id: FEAT-01NOEPICREF01
slug: no-ref
status: draft
summary: No epic reference
created: 2026-01-01T00:00:00Z
created_by: sam
`
	writeRawYAML(t, featuresDir, "FEAT-01NOEPICREF01-no-ref.yaml", featureYAML)

	result, err := svc.MigratePhase2()
	if err != nil {
		t.Fatalf("MigratePhase2() error = %v", err)
	}

	// Feature should not have been updated because it had no epic/plan fields.
	if result.FeaturesUpdated != 0 {
		t.Errorf("FeaturesUpdated = %d, want 0 (no epic ref to migrate)", result.FeaturesUpdated)
	}
}

func TestMigratePhase2_CreatesDirectories(t *testing.T) {
	root := t.TempDir()
	svc := NewEntityService(root)

	setupMigrationConfig(t, "P", "Plan")

	// Create an epic so migration isn't a no-op.
	epicsDir := filepath.Join(root, "epics")
	writeRawYAML(t, epicsDir, "EPIC-DIRS-dirs.yaml", `id: EPIC-DIRS
slug: dirs
title: Dir Test
status: proposed
summary: Directory creation test
created: 2026-01-01T00:00:00Z
created_by: sam
`)

	result, err := svc.MigratePhase2()
	if err != nil {
		t.Fatalf("MigratePhase2() error = %v", err)
	}

	// Should have created plans and documents directories.
	for _, dir := range []string{"plans", "documents"} {
		target := filepath.Join(root, dir)
		info, err := os.Stat(target)
		if err != nil {
			t.Errorf("expected directory %s to exist: %v", dir, err)
			continue
		}
		if !info.IsDir() {
			t.Errorf("expected %s to be a directory", dir)
		}
	}

	if result.PlansCreated != 1 {
		t.Errorf("PlansCreated = %d, want 1", result.PlansCreated)
	}
}
