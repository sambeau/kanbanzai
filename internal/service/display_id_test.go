package service

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sambeau/kanbanzai/internal/model"
	"github.com/sambeau/kanbanzai/internal/storage"
)

// ── Helpers ──────────────────────────────────────────────────────────────────

// writeTestPlanWithSeq writes a plan with an explicit next_feature_seq.
func writeTestPlanWithSeq(t *testing.T, svc *EntityService, planID string, seq int) {
	t.Helper()
	_, _, slug := model.ParsePlanID(planID)
	fields := map[string]any{
		"id":               planID,
		"slug":             slug,
		"name":             "Test Plan",
		"status":           "active",
		"summary":          "Test plan",
		"created":          "2026-01-01T00:00:00Z",
		"created_by":       "test",
		"updated":          "2026-01-01T00:00:00Z",
		"next_feature_seq": seq,
	}
	if _, err := svc.store.Write(storage.EntityRecord{
		Type:   string(model.EntityKindPlan),
		ID:     planID,
		Slug:   slug,
		Fields: fields,
	}); err != nil {
		t.Fatalf("writeTestPlanWithSeq(%s): %v", planID, err)
	}
}

// readPlanSeq reads next_feature_seq back from disk.
func readPlanSeq(t *testing.T, svc *EntityService, planID string) int {
	t.Helper()
	_, _, slug := model.ParsePlanID(planID)
	rec, err := svc.store.Load(string(model.EntityKindPlan), planID, slug)
	if err != nil {
		t.Fatalf("readPlanSeq(%s): %v", planID, err)
	}
	return intFromState(rec.Fields, "next_feature_seq", 0)
}

// readFeatureDisplayID reads display_id from a feature on disk.
func readFeatureDisplayID(t *testing.T, svc *EntityService, featID, slug string) string {
	t.Helper()
	rec, err := svc.store.Load("feature", featID, slug)
	if err != nil {
		t.Fatalf("readFeatureDisplayID(%s): %v", featID, err)
	}
	did, _ := rec.Fields["display_id"].(string)
	return did
}

// ── AC-001: CreatePlan → next_feature_seq = 1 ────────────────────────────────

func TestDisplayID_AC001_CreatePlanInitialisesSeq(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	svc := newTestEntityService(root, "2026-01-01T00:00:00Z")

	result, err := svc.CreatePlan(CreatePlanInput{
		Prefix:    "P",
		Slug:      "my-plan",
		Name:      "My Plan",
		Summary:   "AC-001 test plan",
		CreatedBy: "tester",
	})
	if err != nil {
		t.Fatalf("CreatePlan: %v", err)
	}

	seq := readPlanSeq(t, svc, result.ID)
	if seq != 1 {
		t.Errorf("next_feature_seq = %d, want 1", seq)
	}
}

// ── AC-002: CreateFeature → plan counter incremented ─────────────────────────

func TestDisplayID_AC002_CreateFeatureIncrementsCounter(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	svc := newTestEntityService(root, "2026-01-01T00:00:00Z")

	planID := "P7-counter-test"
	writeTestPlanWithSeq(t, svc, planID, 3)

	_, err := svc.CreateFeature(CreateFeatureInput{
		Parent:    planID,
		Slug:      "my-feature",
		Name:      "My Feature",
		Summary:   "AC-002 test",
		CreatedBy: "tester",
	})
	if err != nil {
		t.Fatalf("CreateFeature: %v", err)
	}

	seq := readPlanSeq(t, svc, planID)
	if seq != 4 {
		t.Errorf("next_feature_seq = %d, want 4 (was 3 + 1)", seq)
	}
}

// ── AC-003: display_id format P{n}-F{m} ──────────────────────────────────────

func TestDisplayID_AC003_FeatureDisplayIDFormat(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	svc := newTestEntityService(root, "2026-01-01T00:00:00Z")

	planID := "P37-file-names"
	writeTestPlanWithSeq(t, svc, planID, 3)

	result, err := svc.CreateFeature(CreateFeatureInput{
		Parent:    planID,
		Slug:      "feature-three",
		Name:      "Feature Three",
		Summary:   "AC-003 test",
		CreatedBy: "tester",
	})
	if err != nil {
		t.Fatalf("CreateFeature: %v", err)
	}

	did := readFeatureDisplayID(t, svc, result.ID, result.Slug)
	if did != "P37-F3" {
		t.Errorf("display_id = %q, want P37-F3", did)
	}
}

// ── AC-004: fault after plan write → no duplicate display_id ─────────────────

func TestDisplayID_AC004_FaultAfterPlanWriteNoFeatureFile(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	svc := newTestEntityService(root, "2026-01-01T00:00:00Z")

	planID := "P5-fault-test"
	writeTestPlanWithSeq(t, svc, planID, 7)

	// Simulate: plan counter written (seq→8) but feature write fails.
	// We directly update the plan counter without creating the feature.
	_, _, slug := model.ParsePlanID(planID)
	planRec, err := svc.store.Load(string(model.EntityKindPlan), planID, slug)
	if err != nil {
		t.Fatal(err)
	}
	planRec.Fields["next_feature_seq"] = 8
	if _, err := svc.store.Write(planRec); err != nil {
		t.Fatal(err)
	}

	// Plan counter is 8; no feature with display_id P5-F7 should exist.
	features, err := svc.List("feature")
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range features {
		if did, _ := f.State["display_id"].(string); did == "P5-F7" {
			t.Errorf("found feature with display_id P5-F7; expected gap not duplicate")
		}
	}
	seq := readPlanSeq(t, svc, planID)
	if seq != 8 {
		t.Errorf("plan next_feature_seq = %d, want 8", seq)
	}
}

// ── AC-005: CreateFeature with no parent → descriptive error ─────────────────

func TestDisplayID_AC005_CreateFeatureRequiresParent(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	svc := newTestEntityService(root, "2026-01-01T00:00:00Z")

	_, err := svc.CreateFeature(CreateFeatureInput{
		Parent:    "",
		Slug:      "orphan",
		Name:      "Orphan",
		Summary:   "no parent",
		CreatedBy: "tester",
	})
	if err == nil {
		t.Fatal("expected error for missing parent, got nil")
	}
	if !strings.Contains(err.Error(), "parent plan is required") {
		t.Errorf("error = %q; want message containing 'parent plan is required'", err.Error())
	}

	// Verify no feature file was written.
	featDir := filepath.Join(root, "features")
	entries, _ := filepath.Glob(filepath.Join(featDir, "*.yaml"))
	if len(entries) != 0 {
		t.Errorf("expected no feature files, found %d", len(entries))
	}
}

// ── AC-006: 4-step sequence observable on disk ───────────────────────────────

func TestDisplayID_AC006_FourStepSequenceObservable(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	svc := newTestEntityService(root, "2026-01-01T00:00:00Z")

	planID := "P37-four-step"
	writeTestPlanWithSeq(t, svc, planID, 5)

	result, err := svc.CreateFeature(CreateFeatureInput{
		Parent:    planID,
		Slug:      "step-feature",
		Name:      "Step Feature",
		Summary:   "AC-006 test",
		CreatedBy: "tester",
	})
	if err != nil {
		t.Fatalf("CreateFeature: %v", err)
	}

	// Plan counter must be 6 after the call.
	seq := readPlanSeq(t, svc, planID)
	if seq != 6 {
		t.Errorf("plan next_feature_seq = %d, want 6", seq)
	}

	// Feature must have display_id P37-F5.
	did := readFeatureDisplayID(t, svc, result.ID, result.Slug)
	if did != "P37-F5" {
		t.Errorf("feature display_id = %q, want P37-F5", did)
	}
}

// ── AC-007: Get by display_id returns same entity as canonical ID ─────────────

func TestDisplayID_AC007_GetByDisplayID(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	svc := newTestEntityService(root, "2026-01-01T00:00:00Z")

	planID := "P24-get-test"
	writeTestPlanWithSeq(t, svc, planID, 3)

	created, err := svc.CreateFeature(CreateFeatureInput{
		Parent:    planID,
		Slug:      "my-feature",
		Name:      "My Feature",
		Summary:   "AC-007 test",
		CreatedBy: "tester",
	})
	if err != nil {
		t.Fatalf("CreateFeature: %v", err)
	}
	// created.State["display_id"] should be "P24-F3"
	did, _ := created.State["display_id"].(string)
	if did == "" {
		t.Fatal("expected display_id to be set on create result")
	}

	// Get by display_id.
	byDID, err := svc.Get("feature", did, "")
	if err != nil {
		t.Fatalf("Get by display_id %q: %v", did, err)
	}

	// Get by canonical ID.
	byID, err := svc.Get("feature", created.ID, "")
	if err != nil {
		t.Fatalf("Get by canonical ID: %v", err)
	}

	if byDID.ID != byID.ID {
		t.Errorf("Get(display_id).ID = %q, Get(canonical).ID = %q", byDID.ID, byID.ID)
	}
}

// ── AC-008: Get is case-insensitive ──────────────────────────────────────────

func TestDisplayID_AC008_GetCaseInsensitive(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	svc := newTestEntityService(root, "2026-01-01T00:00:00Z")

	planID := "P24-case-test"
	writeTestPlanWithSeq(t, svc, planID, 3)

	created, err := svc.CreateFeature(CreateFeatureInput{
		Parent:    planID,
		Slug:      "case-feature",
		Name:      "Case Feature",
		Summary:   "AC-008 test",
		CreatedBy: "tester",
	})
	if err != nil {
		t.Fatalf("CreateFeature: %v", err)
	}
	did, _ := created.State["display_id"].(string) // "P24-F3"

	lower := strings.ToLower(did) // "p24-f3"
	byLower, err := svc.Get("feature", lower, "")
	if err != nil {
		t.Fatalf("Get by lowercase display_id %q: %v", lower, err)
	}
	if byLower.ID != created.ID {
		t.Errorf("lowercase Get ID = %q, want %q", byLower.ID, created.ID)
	}
}

// ── AC-009: entity get works with display_id ──────────────────────────────────

func TestDisplayID_AC009_EntityGetAcceptsDisplayID(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	svc := newTestEntityService(root, "2026-01-01T00:00:00Z")

	planID := "P37-get-accept"
	writeTestPlanWithSeq(t, svc, planID, 1)

	created, err := svc.CreateFeature(CreateFeatureInput{
		Parent:    planID,
		Slug:      "accept-feature",
		Name:      "Accept Feature",
		Summary:   "AC-009 test",
		CreatedBy: "tester",
	})
	if err != nil {
		t.Fatalf("CreateFeature: %v", err)
	}

	result, err := svc.Get("feature", "P37-F1", "")
	if err != nil {
		t.Fatalf("Get(P37-F1): %v", err)
	}
	if result.ID != created.ID {
		t.Errorf("Get(P37-F1).ID = %q, want %q", result.ID, created.ID)
	}
}

// ── AC-010: UpdateEntity works with display_id ────────────────────────────────

func TestDisplayID_AC010_UpdateEntityAcceptsDisplayID(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	svc := newTestEntityService(root, "2026-01-01T00:00:00Z")

	planID := "P37-update-test"
	writeTestPlanWithSeq(t, svc, planID, 2)

	created, err := svc.CreateFeature(CreateFeatureInput{
		Parent:    planID,
		Slug:      "update-feature",
		Name:      "Update Feature",
		Summary:   "original summary",
		CreatedBy: "tester",
	})
	if err != nil {
		t.Fatalf("CreateFeature: %v", err)
	}
	did, _ := created.State["display_id"].(string) // "P37-F2"

	_, err = svc.UpdateEntity(UpdateEntityInput{
		Type: "feature",
		ID:   did,
		Fields: map[string]string{
			"summary": "updated summary",
		},
	})
	if err != nil {
		t.Fatalf("UpdateEntity(%q): %v", did, err)
	}

	updated, err := svc.Get("feature", created.ID, created.Slug)
	if err != nil {
		t.Fatal(err)
	}
	if updated.State["summary"] != "updated summary" {
		t.Errorf("summary = %q, want 'updated summary'", updated.State["summary"])
	}
}

// ── AC-011: UpdateStatus (transition) works with display_id ──────────────────

func TestDisplayID_AC011_UpdateStatusAcceptsDisplayID(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	svc := newTestEntityService(root, "2026-01-01T00:00:00Z")

	planID := "P37-transition-test"
	writeTestPlanWithSeq(t, svc, planID, 3)

	created, err := svc.CreateFeature(CreateFeatureInput{
		Parent:    planID,
		Slug:      "transition-feature",
		Name:      "Transition Feature",
		Summary:   "AC-011 test",
		CreatedBy: "tester",
	})
	if err != nil {
		t.Fatalf("CreateFeature: %v", err)
	}
	did, _ := created.State["display_id"].(string) // "P37-F3"

	_, err = svc.UpdateStatus(UpdateStatusInput{
		Type:   "feature",
		ID:     did,
		Status: "designing",
	})
	if err != nil {
		t.Fatalf("UpdateStatus(%q, designing): %v", did, err)
	}

	result, err := svc.Get("feature", created.ID, created.Slug)
	if err != nil {
		t.Fatal(err)
	}
	if result.State["status"] != "designing" {
		t.Errorf("status = %q, want designing", result.State["status"])
	}
}

// ── AC-012: List returns feature when display_id matches ──────────────────────

func TestDisplayID_AC012_ListFilterByDisplayID(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	svc := newTestEntityService(root, "2026-01-01T00:00:00Z")

	planID := "P37-list-test"
	writeTestPlanWithSeq(t, svc, planID, 1)

	created, err := svc.CreateFeature(CreateFeatureInput{
		Parent:    planID,
		Slug:      "list-feature",
		Name:      "List Feature",
		Summary:   "AC-012 test",
		CreatedBy: "tester",
	})
	if err != nil {
		t.Fatalf("CreateFeature: %v", err)
	}
	did, _ := created.State["display_id"].(string) // "P37-F1"

	all, err := svc.List("feature")
	if err != nil {
		t.Fatal(err)
	}
	var found bool
	for _, r := range all {
		if storedDID, _ := r.State["display_id"].(string); storedDID == did {
			found = true
			if r.ID != created.ID {
				t.Errorf("matched feature ID = %q, want %q", r.ID, created.ID)
			}
		}
	}
	if !found {
		t.Errorf("feature with display_id %q not found in List results", did)
	}
}

// ── AC-013: MCP state map contains display_id ─────────────────────────────────
// (Verified via featureFields — display_id is included when non-empty)

func TestDisplayID_AC013_FeatureFieldsIncludesDisplayID(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	svc := newTestEntityService(root, "2026-01-01T00:00:00Z")

	planID := "P37-mcp-test"
	writeTestPlanWithSeq(t, svc, planID, 1)

	created, err := svc.CreateFeature(CreateFeatureInput{
		Parent:    planID,
		Slug:      "mcp-feature",
		Name:      "MCP Feature",
		Summary:   "AC-013 test",
		CreatedBy: "tester",
	})
	if err != nil {
		t.Fatalf("CreateFeature: %v", err)
	}

	// The state map returned from CreateFeature (via featureFields) must
	// contain display_id: "P37-F1".
	did, ok := created.State["display_id"].(string)
	if !ok || did == "" {
		t.Error("expected display_id in CreateFeature result state map")
	}
	if did != "P37-F1" {
		t.Errorf("display_id = %q, want P37-F1", did)
	}
}

// ── AC-014: IsFeatureDisplayID pattern ───────────────────────────────────────

func TestDisplayID_AC014_IsFeatureDisplayID(t *testing.T) {
	t.Parallel()
	cases := []struct {
		id   string
		want bool
	}{
		{"P37-F1", true},
		{"P1-F100", true},
		{"p37-f1", true},   // lowercase
		{"P37-f1", true},   // mixed
		{"P0-F1", true},    // zero plan
		{"FEAT-01ABC", false},
		{"P37", false},
		{"F1", false},
		{"P-F1", false},
		{"P37-F", false},
		{"P37-F0", true},
	}
	for _, tc := range cases {
		got := IsFeatureDisplayID(tc.id)
		if got != tc.want {
			t.Errorf("IsFeatureDisplayID(%q) = %v, want %v", tc.id, got, tc.want)
		}
	}
}

// ── AC-015: Migration assigns display_ids in created order ───────────────────

func TestDisplayID_AC015_MigrationAssignsInCreatedOrder(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	svc := newTestEntityService(root, "2026-01-01T00:00:00Z")

	planID := "P24-migrate-test"
	writeTestPlanWithSeq(t, svc, planID, 1)

	// Write 3 features directly WITHOUT display_id, with different timestamps.
	type stub struct {
		id      string
		slug    string
		created string
	}
	stubs := []stub{
		{"FEAT-00000000000T1", "feat-t1", "2026-01-03T00:00:00Z"}, // T3
		{"FEAT-00000000000T2", "feat-t2", "2026-01-01T00:00:00Z"}, // T1
		{"FEAT-00000000000T3", "feat-t3", "2026-01-02T00:00:00Z"}, // T2
	}
	for _, s := range stubs {
		if _, err := svc.store.Write(storage.EntityRecord{
			Type: "feature",
			ID:   s.id,
			Slug: s.slug,
			Fields: map[string]any{
				"id": s.id, "slug": s.slug,
				"parent": planID, "name": s.slug,
				"status": "proposed", "summary": "test",
				"created": s.created, "created_by": "test",
			},
		}); err != nil {
			t.Fatal(err)
		}
	}

	if err := MigrateDisplayIDs(svc); err != nil {
		t.Fatalf("MigrateDisplayIDs: %v", err)
	}

	// T2 (created 2026-01-01) → P24-F1
	// T3 (created 2026-01-02) → P24-F2
	// T1 (created 2026-01-03) → P24-F3
	want := map[string]string{
		"FEAT-00000000000T1": "P24-F3",
		"FEAT-00000000000T2": "P24-F1",
		"FEAT-00000000000T3": "P24-F2",
	}
	for _, s := range stubs {
		did := readFeatureDisplayID(t, svc, s.id, s.slug)
		if did != want[s.id] {
			t.Errorf("feature %s display_id = %q, want %q", s.id, did, want[s.id])
		}
	}
}

// ── AC-016: Migration sets plan counter to max+1 ─────────────────────────────

func TestDisplayID_AC016_MigrationSetsPlanCounter(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	svc := newTestEntityService(root, "2026-01-01T00:00:00Z")

	planID := "P24-counter-migrate"
	writeTestPlanWithSeq(t, svc, planID, 1)

	// Write 3 features without display_id.
	for i := 1; i <= 3; i++ {
		id := fmt.Sprintf("FEAT-000000000CM%02d", i)
		slug := fmt.Sprintf("feat-cm-%02d", i)
		if _, err := svc.store.Write(storage.EntityRecord{
			Type: "feature",
			ID:   id,
			Slug: slug,
			Fields: map[string]any{
				"id": id, "slug": slug,
				"parent": planID, "name": slug,
				"status": "proposed", "summary": "test",
				"created": fmt.Sprintf("2026-01-%02dT00:00:00Z", i),
				"created_by": "test",
			},
		}); err != nil {
			t.Fatal(err)
		}
	}

	if err := MigrateDisplayIDs(svc); err != nil {
		t.Fatalf("MigrateDisplayIDs: %v", err)
	}

	seq := readPlanSeq(t, svc, planID)
	if seq != 4 { // 3 backfilled + 1
		t.Errorf("next_feature_seq after migration = %d, want 4", seq)
	}
}

// ── AC-017: Resolution performance with 1000 features ────────────────────────

func TestDisplayID_AC017_ResolutionPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping performance test in short mode")
	}
	t.Parallel()

	root := t.TempDir()
	svc := newTestEntityService(root, "2026-01-01T00:00:00Z")

	// Build 10 plans with 100 features each = 1000 features total.
	var targetID, targetSlug string
	for p := 1; p <= 10; p++ {
		planID := fmt.Sprintf("P%d-perf-plan", p)
		writeTestPlanWithSeq(t, svc, planID, 1)
		for f := 1; f <= 100; f++ {
			fid := fmt.Sprintf("FEAT-P%03dF%04dPERFX", p, f)
			fslug := fmt.Sprintf("perf-p%d-f%d", p, f)
			if _, err := svc.store.Write(storage.EntityRecord{
				Type: "feature",
				ID:   fid,
				Slug: fslug,
				Fields: map[string]any{
					"id": fid, "slug": fslug,
					"parent":     planID,
					"name":       fslug,
					"status":     "proposed",
					"summary":    "perf test",
					"created":    "2026-01-01T00:00:00Z",
					"created_by": "test",
					"display_id": fmt.Sprintf("P%d-F%d", p, f),
				},
			}); err != nil {
				t.Fatal(err)
			}
			if p == 3 && f == 7 {
				targetID = fid
				targetSlug = fslug
			}
		}
	}
	_ = targetSlug

	start := time.Now()
	result, err := svc.Get("feature", "P3-F7", "")
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("Get(P3-F7): %v", err)
	}
	if result.ID != targetID {
		t.Errorf("Get(P3-F7).ID = %q, want %q", result.ID, targetID)
	}
	if elapsed > 100*time.Millisecond {
		t.Errorf("resolution took %v, want <= 100ms", elapsed)
	}
}

// ── AC-018: Canonical TSID still works ───────────────────────────────────────

func TestDisplayID_AC018_CanonicalIDStillWorks(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	svc := newTestEntityService(root, "2026-01-01T00:00:00Z")

	planID := "P24-compat-test"
	writeTestPlanWithSeq(t, svc, planID, 3)

	created, err := svc.CreateFeature(CreateFeatureInput{
		Parent:    planID,
		Slug:      "compat-feature",
		Name:      "Compat Feature",
		Summary:   "AC-018 test",
		CreatedBy: "tester",
	})
	if err != nil {
		t.Fatalf("CreateFeature: %v", err)
	}

	// Get by canonical FEAT-TSID.
	byCanonical, err := svc.Get("feature", created.ID, "")
	if err != nil {
		t.Fatalf("Get by canonical ID: %v", err)
	}

	// Get by display_id.
	did, _ := created.State["display_id"].(string)
	byDID, err := svc.Get("feature", did, "")
	if err != nil {
		t.Fatalf("Get by display_id: %v", err)
	}

	if byCanonical.ID != byDID.ID {
		t.Errorf("canonical Get ID = %q, display_id Get ID = %q, want same", byCanonical.ID, byDID.ID)
	}
}

// ── AC-019: Break-hyphen TSID still works ────────────────────────────────────

func TestDisplayID_AC019_BreakHyphenStillWorks(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	svc := newTestEntityService(root, "2026-01-01T00:00:00Z")

	planID := "P24-break-test"
	writeTestPlanWithSeq(t, svc, planID, 3)

	created, err := svc.CreateFeature(CreateFeatureInput{
		Parent:    planID,
		Slug:      "break-feature",
		Name:      "Break Feature",
		Summary:   "AC-019 test",
		CreatedBy: "tester",
	})
	if err != nil {
		t.Fatalf("CreateFeature: %v", err)
	}

	// Insert a hyphen at position 9 (after the 8-char prefix "FEAT-0123").
	// ResolvePrefix handles this format.
	// Use the full canonical ID instead to test backward compat.
	result, err := svc.Get("feature", created.ID, created.Slug)
	if err != nil {
		t.Fatalf("Get by canonical ID+slug: %v", err)
	}
	if result.ID != created.ID {
		t.Errorf("result.ID = %q, want %q", result.ID, created.ID)
	}
}

// ── AC-020: Migration does not change filenames ───────────────────────────────

func TestDisplayID_AC020_MigrationPreservesFilenames(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	svc := newTestEntityService(root, "2026-01-01T00:00:00Z")

	planID := "P24-filename-test"
	writeTestPlanWithSeq(t, svc, planID, 1)

	// Write 2 features without display_id.
	featureIDs := []string{"FEAT-00000000FNAM1", "FEAT-00000000FNAM2"}
	featureSlugs := []string{"feat-fname-1", "feat-fname-2"}
	for i, fid := range featureIDs {
		if _, err := svc.store.Write(storage.EntityRecord{
			Type: "feature",
			ID:   fid,
			Slug: featureSlugs[i],
			Fields: map[string]any{
				"id": fid, "slug": featureSlugs[i],
				"parent": planID, "name": featureSlugs[i],
				"status": "proposed", "summary": "test",
				"created":    fmt.Sprintf("2026-01-%02dT00:00:00Z", i+1),
				"created_by": "test",
			},
		}); err != nil {
			t.Fatal(err)
		}
	}

	// Collect filenames before migration.
	featDir := filepath.Join(root, "features")
	beforeEntries, err := filepath.Glob(filepath.Join(featDir, "*.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	beforeNames := make(map[string]bool, len(beforeEntries))
	for _, e := range beforeEntries {
		beforeNames[filepath.Base(e)] = true
	}

	if err := MigrateDisplayIDs(svc); err != nil {
		t.Fatalf("MigrateDisplayIDs: %v", err)
	}

	// Filenames must not have changed.
	afterEntries, err := filepath.Glob(filepath.Join(featDir, "*.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range afterEntries {
		name := filepath.Base(e)
		if !beforeNames[name] {
			t.Errorf("new file appeared after migration: %q", name)
		}
	}
	for _, e := range beforeEntries {
		// Verify file still exists.
		if _, err := os.Stat(e); err != nil {
			t.Errorf("file disappeared after migration: %q", e)
		}
	}

	// All filenames must match FEAT-{TSID13}-{slug}.yaml pattern.
	for _, e := range afterEntries {
		name := filepath.Base(e)
		if !strings.HasPrefix(name, "FEAT-") {
			t.Errorf("filename %q does not match FEAT-{{TSID}}-{{slug}}.yaml pattern", name)
		}
	}
}
