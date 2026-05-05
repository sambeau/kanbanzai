package service

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sambeau/kanbanzai/internal/model"
	"github.com/sambeau/kanbanzai/internal/storage"
)

// setupCanonicalDocPathTest creates a temp directory with plan, batch, and feature entities.
func setupCanonicalDocPathTest(t *testing.T) *EntityService {
	t.Helper()
	dir := t.TempDir()
	kbzDir := filepath.Join(dir, ".kbz", "state", "entities")
	os.MkdirAll(kbzDir, 0o755)

	storeRoot := filepath.Join(dir, ".kbz", "state")
	store := storage.NewEntityStore(storeRoot)

	svc := &EntityService{
		root:  storeRoot,
		store: store,
		now: func() time.Time {
			return time.Now().UTC()
		},
	}

	// Write entities with the correct type. Plans/batches use EntityKindPlan which
	// is "batch" (the plan→batch rename).
	writeBatch(t, svc, "P50-retro-may-2026", "retro-may-2026", "")
	writeBatch(t, svc, "P1-master-plan", "master-plan", "")
	writeBatch(t, svc, "B49-execution", "execution", "P50-retro-may-2026")
	writeBatch(t, svc, "B42-test-batch", "test-batch", "P1-master-plan")
	writeBatch(t, svc, "B99-standalone", "standalone", "")

	writeFeature(t, svc, "FEAT-01KQTNYN00HZA", "doc-path-tool", "B49-execution")
	writeFeature(t, svc, "FEAT-01ABCDEF12345", "direct-feature", "P50-retro-may-2026")

	return svc
}

func writeBatch(t *testing.T, svc *EntityService, id, slug, parent string) {
	t.Helper()
	fields := map[string]any{
		"id":     id,
		"slug":   slug,
		"name":   "Test " + id,
		"status": "active",
	}
	if parent != "" {
		fields["parent"] = parent
	}
	_, err := svc.store.Write(storage.EntityRecord{
		Type:   "batch",
		ID:     id,
		Slug:   slug,
		Fields: fields,
	})
	if err != nil {
		t.Fatalf("write batch %s: %v", id, err)
	}
}

func writeFeature(t *testing.T, svc *EntityService, id, slug, parent string) {
	t.Helper()
	_, err := svc.store.Write(storage.EntityRecord{
		Type: "feature",
		ID:   id,
		Slug: slug,
		Fields: map[string]any{
			"id":     id,
			"slug":   slug,
			"name":   "Test " + id,
			"status": "active",
			"parent": parent,
		},
	})
	if err != nil {
		t.Fatalf("write feature %s: %v", id, err)
	}
}

// --- tests ---

func TestCanonicalDocPath_PlanParent(t *testing.T) {
	t.Parallel()
	svc := setupCanonicalDocPathTest(t)

	tests := []struct {
		docType string
		parent  string
		want    string
	}{
		{"design", "P50-retro-may-2026", "work/retro-may-2026/P50-retro-may-2026-design-retro-may-2026.md"},
		{"specification", "P50-retro-may-2026", "work/retro-may-2026/P50-retro-may-2026-spec-retro-may-2026.md"},
		{"dev-plan", "P50-retro-may-2026", "work/retro-may-2026/P50-retro-may-2026-dev-plan-retro-may-2026.md"},
		{"research", "P50-retro-may-2026", "work/retro-may-2026/P50-retro-may-2026-research-retro-may-2026.md"},
		{"report", "P50-retro-may-2026", "work/retro-may-2026/P50-retro-may-2026-report-retro-may-2026.md"},
		{"policy", "P50-retro-may-2026", "work/retro-may-2026/P50-retro-may-2026-policy-retro-may-2026.md"},
		{"design", "P1-master-plan", "work/master-plan/P1-master-plan-design-master-plan.md"},
	}

	for _, tt := range tests {
		got, err := svc.CanonicalDocPath(tt.docType, tt.parent)
		if err != nil {
			t.Errorf("canonicalDocPath(%q, %q): unexpected error: %v", tt.docType, tt.parent, err)
			continue
		}
		if got != tt.want {
			t.Errorf("canonicalDocPath(%q, %q) = %q, want %q", tt.docType, tt.parent, got, tt.want)
		}
	}
}

func TestCanonicalDocPath_BatchParent(t *testing.T) {
	t.Parallel()
	svc := setupCanonicalDocPathTest(t)

	got, err := svc.CanonicalDocPath("specification", "B49-execution")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "work/retro-may-2026/P50-retro-may-2026-spec-retro-may-2026.md"
	if got != want {
		t.Errorf("canonicalDocPath(specification, B49-execution) = %q, want %q", got, want)
	}
}

func TestCanonicalDocPath_FeatureParent(t *testing.T) {
	t.Parallel()
	svc := setupCanonicalDocPathTest(t)

	got, err := svc.CanonicalDocPath("design", "FEAT-01KQTNYN00HZA")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "work/retro-may-2026/P50-retro-may-2026-design-retro-may-2026.md"
	if got != want {
		t.Errorf("canonicalDocPath(design, FEAT-01KQTNYN00HZA) = %q, want %q", got, want)
	}

	got, err = svc.CanonicalDocPath("design", "FEAT-01ABCDEF12345")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want = "work/retro-may-2026/P50-retro-may-2026-design-retro-may-2026.md"
	if got != want {
		t.Errorf("canonicalDocPath(design, FEAT-01ABCDEF12345) = %q, want %q", got, want)
	}
}

func TestCanonicalDocPath_NoParent(t *testing.T) {
	t.Parallel()
	svc := setupCanonicalDocPathTest(t)

	_, err := svc.CanonicalDocPath("design", "")
	if err == nil {
		t.Fatal("expected error for empty parent, got nil")
	}
	if err.Error() != "cannot determine path: no parent entity provided. Specify a parent plan, batch, or feature ID" {
		t.Errorf("error: %q", err.Error())
	}
}

func TestCanonicalDocPath_NonexistentParent(t *testing.T) {
	t.Parallel()
	svc := setupCanonicalDocPathTest(t)

	_, err := svc.CanonicalDocPath("design", "P999-nonexist")
	if err == nil {
		t.Fatal("expected error for non-existent parent, got nil")
	}
	// REQ-005 / AC-005: error must state the parent entity was not found.
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error = %q, want message containing 'not found'", err.Error())
	}
	if !strings.Contains(err.Error(), "P999-nonexist") {
		t.Errorf("error = %q, want message containing parent ID 'P999-nonexist'", err.Error())
	}
}

func TestCanonicalDocPath_FeatureNoParent(t *testing.T) {
	t.Parallel()
	svc := setupCanonicalDocPathTest(t)

	// Write a feature with no parent field.
	_, err := svc.store.Write(storage.EntityRecord{
		Type: "feature",
		ID:   "FEAT-99NOPARENT",
		Slug: "no-parent",
		Fields: map[string]any{
			"id":     "FEAT-99NOPARENT",
			"slug":   "no-parent",
			"name":   "Test orphan feature",
			"status": "active",
		},
	})
	if err != nil {
		t.Fatalf("write orphan feature: %v", err)
	}

	_, err = svc.CanonicalDocPath("design", "FEAT-99NOPARENT")
	if err == nil {
		t.Fatal("expected error for feature with no parent, got nil")
	}
	if !strings.Contains(err.Error(), "no parent") && !strings.Contains(err.Error(), "has no parent") {
		t.Errorf("error = %q, want message about missing parent", err.Error())
	}
}

func TestCanonicalDocPath_UnknownIDFormat(t *testing.T) {
	t.Parallel()
	svc := setupCanonicalDocPathTest(t)

	// An ID that is neither a plan/batch ID nor a feature ID.
	_, err := svc.CanonicalDocPath("design", "NOT-A-VALID-ID")
	if err == nil {
		t.Fatal("expected error for unknown ID format, got nil")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error = %q, want message containing 'not found'", err.Error())
	}
}

func TestCanonicalDocPath_StandaloneBatch(t *testing.T) {
	t.Parallel()
	svc := setupCanonicalDocPathTest(t)

	got, err := svc.CanonicalDocPath("design", "B99-standalone")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "work/standalone/B99-standalone-design-standalone.md"
	if got != want {
		t.Errorf("canonicalDocPath(design, B99-standalone) = %q, want %q", got, want)
	}
}

func TestCanonicalDocPath_PromptType(t *testing.T) {
	t.Parallel()
	svc := setupCanonicalDocPathTest(t)

	got, err := svc.CanonicalDocPath("prompt", "P50-retro-may-2026")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "work/retro-may-2026/prompts/retro-may-2026.md"
	if got != want {
		t.Errorf("canonicalDocPath(prompt, P50-retro-may-2026) = %q, want %q", got, want)
	}
}

func TestCanonicalDocPath_InvalidType(t *testing.T) {
	t.Parallel()
	svc := setupCanonicalDocPathTest(t)

	_, err := svc.CanonicalDocPath("nonexistent-type", "P50-retro-may-2026")
	if err == nil {
		t.Fatal("expected error for invalid doc type, got nil")
	}
}

func TestCanonicalDocPath_RetroType(t *testing.T) {
	t.Parallel()
	svc := setupCanonicalDocPathTest(t)

	got, err := svc.CanonicalDocPath("retro", "P50-retro-may-2026")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "work/retro-may-2026/P50-retro-may-2026-retro-retro-may-2026.md"
	if got != want {
		t.Errorf("canonicalDocPath(retro, P50-retro-may-2026) = %q, want %q", got, want)
	}
}

func TestCanonicalDocPath_CaseInsensitiveTypes(t *testing.T) {
	t.Parallel()
	svc := setupCanonicalDocPathTest(t)

	got, err := svc.CanonicalDocPath("DESIGN", "P50-retro-may-2026")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "work/retro-may-2026/P50-retro-may-2026-design-retro-may-2026.md"
	if got != want {
		t.Errorf("canonicalDocPath(DESIGN, ...) = %q, want %q", got, want)
	}
}

func TestResolveToPlan_PlanDirectly(t *testing.T) {
	t.Parallel()
	svc := setupCanonicalDocPathTest(t)

	planID, planSlug, err := svc.resolveToPlan("P50-retro-may-2026")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if planID != "P50-retro-may-2026" {
		t.Errorf("planID = %q, want P50-retro-may-2026", planID)
	}
	if planSlug != "retro-may-2026" {
		t.Errorf("planSlug = %q, want retro-may-2026", planSlug)
	}
}

func TestResolveToPlan_BatchUpward(t *testing.T) {
	t.Parallel()
	svc := setupCanonicalDocPathTest(t)

	planID, planSlug, err := svc.resolveToPlan("B49-execution")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if planID != "P50-retro-may-2026" {
		t.Errorf("planID = %q, want P50-retro-may-2026", planID)
	}
	if planSlug != "retro-may-2026" {
		t.Errorf("planSlug = %q, want retro-may-2026", planSlug)
	}
}

func TestResolveToPlan_FeatureUpward(t *testing.T) {
	t.Parallel()
	svc := setupCanonicalDocPathTest(t)

	planID, planSlug, err := svc.resolveToPlan("FEAT-01KQTNYN00HZA")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if planID != "P50-retro-may-2026" {
		t.Errorf("planID = %q, want P50-retro-may-2026", planID)
	}
	if planSlug != "retro-may-2026" {
		t.Errorf("planSlug = %q, want retro-may-2026", planSlug)
	}
}

func TestResolveToPlan_Nonexistent(t *testing.T) {
	t.Parallel()
	svc := setupCanonicalDocPathTest(t)

	_, _, err := svc.resolveToPlan("P999-nonexist")
	if err == nil {
		t.Fatal("expected error for non-existent plan, got nil")
	}
}

func TestResolveToPlan_StandaloneBatch(t *testing.T) {
	t.Parallel()
	svc := setupCanonicalDocPathTest(t)

	planID, planSlug, err := svc.resolveToPlan("B99-standalone")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if planID != "B99-standalone" {
		t.Errorf("planID = %q, want B99-standalone", planID)
	}
	if planSlug != "standalone" {
		t.Errorf("planSlug = %q, want standalone", planSlug)
	}
}

func TestDocTypeAbbreviations(t *testing.T) {
	t.Parallel()

	cases := map[model.DocumentType]string{
		model.DocumentTypeDesign:        "design",
		model.DocumentTypeSpecification: "spec",
		model.DocumentTypeDevPlan:       "dev-plan",
		model.DocumentTypeResearch:      "research",
		model.DocumentTypeReport:        "report",
		model.DocumentTypePolicy:        "policy",
	}

	for dt, want := range cases {
		got, ok := docTypeAbbreviations[string(dt)]
		if !ok {
			t.Errorf("docTypeAbbreviations missing key for %q", dt)
			continue
		}
		if got != want {
			t.Errorf("docTypeAbbreviations[%q] = %q, want %q", dt, got, want)
		}
	}
}
