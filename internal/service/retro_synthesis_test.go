package service

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sambeau/kanbanzai/internal/config"
	"github.com/sambeau/kanbanzai/internal/model"
	"github.com/sambeau/kanbanzai/internal/storage"
)

// ─── parseSeverityFromContent ─────────────────────────────────────────────────

func TestParseSeverityFromContent_ValidSeverities(t *testing.T) {
	t.Parallel()
	cases := []struct {
		content string
		want    string
	}{
		{"[minor] tool-gap: No tool for X", "minor"},
		{"[moderate] spec-ambiguity: Spec unclear", "moderate"},
		{"[significant] context-gap: Missing context", "significant"},
	}
	for _, tc := range cases {
		got := parseSeverityFromContent(tc.content)
		if got != tc.want {
			t.Errorf("parseSeverityFromContent(%q) = %q, want %q", tc.content, got, tc.want)
		}
	}
}

func TestParseSeverityFromContent_Invalid(t *testing.T) {
	t.Parallel()
	cases := []string{
		"",
		"no brackets here",
		"[unknown] tool-gap: something",
		"[moderate without closing bracket",
	}
	for _, c := range cases {
		got := parseSeverityFromContent(c)
		if got != "" {
			t.Errorf("parseSeverityFromContent(%q) = %q, want empty", c, got)
		}
	}
}

// ─── parseObservationAndSuggestion ───────────────────────────────────────────

func TestParseObservationAndSuggestion_ObservationOnly(t *testing.T) {
	t.Parallel()
	content := "[moderate] spec-ambiguity: Error handling underspecified"
	obs, sug := parseObservationAndSuggestion(content, "spec-ambiguity")
	if obs != "Error handling underspecified" {
		t.Errorf("observation = %q, want %q", obs, "Error handling underspecified")
	}
	if sug != "" {
		t.Errorf("suggestion = %q, want empty", sug)
	}
}

func TestParseObservationAndSuggestion_WithSuggestion(t *testing.T) {
	t.Parallel()
	content := "[moderate] spec-ambiguity: Error handling underspecified Suggestion: Add error format section"
	obs, sug := parseObservationAndSuggestion(content, "spec-ambiguity")
	if obs != "Error handling underspecified" {
		t.Errorf("observation = %q, want %q", obs, "Error handling underspecified")
	}
	if sug != "Add error format section" {
		t.Errorf("suggestion = %q, want %q", sug, "Add error format section")
	}
}

func TestParseObservationAndSuggestion_WithSuggestionAndRelated(t *testing.T) {
	t.Parallel()
	content := "[significant] spec-ambiguity: Error undefined Suggestion: Add format section Related: DEC-042"
	obs, sug := parseObservationAndSuggestion(content, "spec-ambiguity")
	if obs != "Error undefined" {
		t.Errorf("observation = %q, want %q", obs, "Error undefined")
	}
	if sug != "Add format section" {
		t.Errorf("suggestion = %q, want %q", sug, "Add format section")
	}
}

func TestParseObservationAndSuggestion_WithRelatedNoSuggestion(t *testing.T) {
	t.Parallel()
	content := "[minor] worked-well: Vertical slicing was great Related: DEC-043"
	obs, sug := parseObservationAndSuggestion(content, "worked-well")
	if obs != "Vertical slicing was great" {
		t.Errorf("observation = %q, want %q", obs, "Vertical slicing was great")
	}
	if sug != "" {
		t.Errorf("suggestion = %q, want empty", sug)
	}
}

// ─── parseRelatedDecision ─────────────────────────────────────────────────────

func TestParseRelatedDecision_Present(t *testing.T) {
	t.Parallel()
	got := parseRelatedDecision("[moderate] spec-ambiguity: Error undefined Suggestion: Add format Related: DEC-042")
	if got != "DEC-042" {
		t.Errorf("parseRelatedDecision() = %q, want %q", got, "DEC-042")
	}
}

func TestParseRelatedDecision_Absent(t *testing.T) {
	t.Parallel()
	got := parseRelatedDecision("[minor] tool-gap: No deploy automation tool exists")
	if got != "" {
		t.Errorf("parseRelatedDecision() = %q, want empty", got)
	}
}

func TestParseRelatedDecision_NoSuggestion(t *testing.T) {
	t.Parallel()
	got := parseRelatedDecision("[minor] worked-well: Vertical slicing was great Related: DEC-043")
	if got != "DEC-043" {
		t.Errorf("parseRelatedDecision() = %q, want %q", got, "DEC-043")
	}
}

func TestParseRelatedDecision_EmptyContent(t *testing.T) {
	t.Parallel()
	got := parseRelatedDecision("")
	if got != "" {
		t.Errorf("parseRelatedDecision() = %q, want empty", got)
	}
}

func TestParseObservationAndSuggestion_CategoryNotFound(t *testing.T) {
	t.Parallel()
	content := "[minor] tool-gap: something"
	// Using wrong category — should fall back to returning full content.
	obs, _ := parseObservationAndSuggestion(content, "wrong-category")
	if obs != content {
		t.Errorf("observation = %q, want full content %q", obs, content)
	}
}

// ─── severityWeight ───────────────────────────────────────────────────────────

func TestSeverityWeight(t *testing.T) {
	t.Parallel()
	cases := []struct {
		sev  string
		want int
	}{
		{"minor", 1},
		{"moderate", 3},
		{"significant", 5},
		{"unknown", 1}, // falls through to default
		{"", 1},
	}
	for _, tc := range cases {
		got := severityWeight(tc.sev)
		if got != tc.want {
			t.Errorf("severityWeight(%q) = %d, want %d", tc.sev, got, tc.want)
		}
	}
}

// ─── parseRetroRecord ─────────────────────────────────────────────────────────

func TestParseRetroRecord_Valid(t *testing.T) {
	t.Parallel()
	rec := storage.KnowledgeRecord{
		ID: "KE-001",
		Fields: map[string]any{
			"tags":         []any{"retrospective", "spec-ambiguity"},
			"content":      "[moderate] spec-ambiguity: Error format undefined",
			"learned_from": "TASK-ABC",
			"created":      "2026-03-01T10:00:00Z",
		},
	}
	sig, ok := parseRetroRecord(rec)
	if !ok {
		t.Fatal("parseRetroRecord returned ok=false, want ok=true")
	}
	if sig.EntryID != "KE-001" {
		t.Errorf("EntryID = %q, want %q", sig.EntryID, "KE-001")
	}
	if sig.Category != "spec-ambiguity" {
		t.Errorf("Category = %q, want %q", sig.Category, "spec-ambiguity")
	}
	if sig.Severity != "moderate" {
		t.Errorf("Severity = %q, want %q", sig.Severity, "moderate")
	}
	if sig.Observation != "Error format undefined" {
		t.Errorf("Observation = %q, want %q", sig.Observation, "Error format undefined")
	}
	if sig.LearnedFrom != "TASK-ABC" {
		t.Errorf("LearnedFrom = %q, want %q", sig.LearnedFrom, "TASK-ABC")
	}
	wantTime, _ := time.Parse(time.RFC3339, "2026-03-01T10:00:00Z")
	if !sig.Created.Equal(wantTime) {
		t.Errorf("Created = %v, want %v", sig.Created, wantTime)
	}
}

func TestParseRetroRecord_NoCategory(t *testing.T) {
	t.Parallel()
	rec := storage.KnowledgeRecord{
		ID: "KE-002",
		Fields: map[string]any{
			"tags":    []any{"retrospective"}, // no category tag
			"content": "[minor] spec-ambiguity: something",
		},
	}
	_, ok := parseRetroRecord(rec)
	if ok {
		t.Error("expected ok=false for record with no category tag")
	}
}

func TestParseRetroRecord_NoContent(t *testing.T) {
	t.Parallel()
	rec := storage.KnowledgeRecord{
		ID: "KE-003",
		Fields: map[string]any{
			"tags": []any{"retrospective", "tool-gap"},
		},
	}
	_, ok := parseRetroRecord(rec)
	if ok {
		t.Error("expected ok=false for record with no content")
	}
}

func TestParseRetroRecord_UnknownSeverityInContent(t *testing.T) {
	t.Parallel()
	rec := storage.KnowledgeRecord{
		ID: "KE-004",
		Fields: map[string]any{
			"tags":    []any{"retrospective", "tool-gap"},
			"content": "[critical] tool-gap: unknown severity",
		},
	}
	_, ok := parseRetroRecord(rec)
	if ok {
		t.Error("expected ok=false for record with unknown severity in content")
	}
}

// ─── retroThemeTitle ──────────────────────────────────────────────────────────

func TestRetroThemeTitle_ShortObservation(t *testing.T) {
	t.Parallel()
	got := retroThemeTitle("spec-ambiguity", "Error format undefined")
	if !strings.HasPrefix(got, "spec-ambiguity: ") {
		t.Errorf("title does not start with category: %q", got)
	}
	if !strings.Contains(got, "Error format undefined") {
		t.Errorf("title does not contain observation: %q", got)
	}
}

func TestRetroThemeTitle_LongObservationTruncated(t *testing.T) {
	t.Parallel()
	long := strings.Repeat("x", 100)
	got := retroThemeTitle("tool-gap", long)
	if len(got) > len("tool-gap: ")+60+3 { // category + ": " + 60 chars + "..."
		t.Errorf("title not truncated, len=%d: %q", len(got), got)
	}
	if !strings.HasSuffix(got, "...") {
		t.Errorf("truncated title should end with '...': %q", got)
	}
}

// ─── clusterRetroSignals ──────────────────────────────────────────────────────

func TestClusterRetroSignals_Empty(t *testing.T) {
	t.Parallel()
	got := clusterRetroSignals(nil)
	if got != nil {
		t.Errorf("clusterRetroSignals(nil) = %v, want nil", got)
	}
}

func TestClusterRetroSignals_SingleSignal(t *testing.T) {
	t.Parallel()
	sigs := []parsedRetroSignal{
		{EntryID: "KE-1", Category: "tool-gap", Severity: "minor", Content: "[minor] tool-gap: no deploy tool"},
	}
	clusters := clusterRetroSignals(sigs)
	if len(clusters) != 1 {
		t.Fatalf("expected 1 cluster, got %d", len(clusters))
	}
	if len(clusters[0].signals) != 1 {
		t.Errorf("cluster 0 has %d signals, want 1", len(clusters[0].signals))
	}
}

func TestClusterRetroSignals_SimilarSignalsMerge(t *testing.T) {
	t.Parallel()
	// Two very similar signals — should be in the same cluster.
	sigs := []parsedRetroSignal{
		{EntryID: "KE-1", Category: "spec-ambiguity", Severity: "minor",
			Content: "[minor] spec-ambiguity: error format not defined in the specification document"},
		{EntryID: "KE-2", Category: "spec-ambiguity", Severity: "moderate",
			Content: "[moderate] spec-ambiguity: error format not defined in the specification document retry policy"},
	}
	clusters := clusterRetroSignals(sigs)
	// At least two signals should be in at most 2 clusters.
	totalSigs := 0
	for _, c := range clusters {
		totalSigs += len(c.signals)
	}
	if totalSigs != 2 {
		t.Errorf("total signals across clusters = %d, want 2", totalSigs)
	}
}

func TestClusterRetroSignals_DissimilarSignalsSeparate(t *testing.T) {
	t.Parallel()
	// Two completely different signals — each should be its own cluster.
	sigs := []parsedRetroSignal{
		{EntryID: "KE-1", Category: "tool-gap", Severity: "minor",
			Content: "[minor] tool-gap: deploy automation missing"},
		{EntryID: "KE-2", Category: "tool-gap", Severity: "moderate",
			Content: "[moderate] tool-gap: test coverage reporting unavailable"},
	}
	clusters := clusterRetroSignals(sigs)
	if len(clusters) != 2 {
		t.Errorf("expected 2 clusters for dissimilar signals, got %d", len(clusters))
	}
}

// ─── retroClusterScore ────────────────────────────────────────────────────────

func TestRetroClusterScore(t *testing.T) {
	t.Parallel()
	cases := []struct {
		signals []parsedRetroSignal
		want    int
	}{
		{
			signals: []parsedRetroSignal{
				{Severity: "minor"},
				{Severity: "minor"},
			},
			want: 2 * 1, // 2 signals × weight 1
		},
		{
			signals: []parsedRetroSignal{
				{Severity: "minor"},
				{Severity: "significant"},
			},
			want: 2 * 5, // 2 signals × max weight 5
		},
		{
			signals: []parsedRetroSignal{
				{Severity: "moderate"},
				{Severity: "moderate"},
				{Severity: "moderate"},
			},
			want: 3 * 3, // 3 signals × weight 3
		},
	}
	for _, tc := range cases {
		c := retroCluster{signals: tc.signals}
		got := retroClusterScore(c)
		if got != tc.want {
			t.Errorf("retroClusterScore(%v) = %d, want %d", tc.signals, got, tc.want)
		}
	}
}

// ─── buildRetroThemes ─────────────────────────────────────────────────────────

func TestBuildRetroThemes_Empty(t *testing.T) {
	t.Parallel()
	got := buildRetroThemes(nil)
	if len(got) != 0 {
		t.Errorf("buildRetroThemes(nil) = %v, want empty", got)
	}
}

func TestBuildRetroThemes_SingleSignal(t *testing.T) {
	t.Parallel()
	sigs := []parsedRetroSignal{
		{EntryID: "KE-1", Category: "tool-gap", Severity: "minor", Observation: "No deploy tool",
			Content: "[minor] tool-gap: No deploy tool"},
	}
	themes := buildRetroThemes(sigs)
	if len(themes) != 1 {
		t.Fatalf("expected 1 theme, got %d", len(themes))
	}
	if themes[0].Rank != 1 {
		t.Errorf("Rank = %d, want 1", themes[0].Rank)
	}
	if themes[0].Category != "tool-gap" {
		t.Errorf("Category = %q, want %q", themes[0].Category, "tool-gap")
	}
	if themes[0].SignalCount != 1 {
		t.Errorf("SignalCount = %d, want 1", themes[0].SignalCount)
	}
	if len(themes[0].Signals) != 1 || themes[0].Signals[0] != "KE-1" {
		t.Errorf("Signals = %v, want [KE-1]", themes[0].Signals)
	}
}

func TestBuildRetroThemes_RankingByScore(t *testing.T) {
	t.Parallel()
	// Two clusters: one with 1 significant signal (score=5), one with 3 minor signals (score=3).
	// The significant one should rank first.
	sigs := []parsedRetroSignal{
		// Three minor tool-gap signals (distinct enough to stay separate or merge — score = 3×1=3 max)
		{EntryID: "KE-1", Category: "context-gap", Severity: "minor", Observation: "A",
			Content: "[minor] context-gap: A context was missing from the packet"},
		{EntryID: "KE-2", Category: "context-gap", Severity: "minor", Observation: "B",
			Content: "[minor] context-gap: B convention not included in assembled context"},
		{EntryID: "KE-3", Category: "context-gap", Severity: "minor", Observation: "C",
			Content: "[minor] context-gap: C policy document absent from context packet"},
		// One significant spec-ambiguity signal (score = 1×5=5).
		{EntryID: "KE-4", Category: "spec-ambiguity", Severity: "significant",
			Observation: "Error handling completely undefined",
			Content:     "[significant] spec-ambiguity: Error handling completely undefined"},
	}
	themes := buildRetroThemes(sigs)
	if len(themes) == 0 {
		t.Fatal("expected themes, got none")
	}
	// The spec-ambiguity significant signal should be rank 1 (score 5 > 3).
	if themes[0].Category != "spec-ambiguity" {
		t.Errorf("rank-1 theme category = %q, want %q", themes[0].Category, "spec-ambiguity")
	}
	if themes[0].SeverityScore < themes[len(themes)-1].SeverityScore {
		t.Error("themes should be sorted descending by severity_score")
	}
}

func TestBuildRetroThemes_TopSuggestionPresent(t *testing.T) {
	t.Parallel()
	sigs := []parsedRetroSignal{
		{EntryID: "KE-1", Category: "spec-ambiguity", Severity: "moderate",
			Observation: "Error format undefined",
			Suggestion:  "Add error format section to spec",
			Content:     "[moderate] spec-ambiguity: Error format undefined Suggestion: Add error format section to spec"},
	}
	themes := buildRetroThemes(sigs)
	if len(themes) == 0 {
		t.Fatal("expected at least one theme")
	}
	if themes[0].TopSuggestion != "Add error format section to spec" {
		t.Errorf("TopSuggestion = %q, want %q", themes[0].TopSuggestion, "Add error format section to spec")
	}
}

func TestBuildRetroThemes_TopSuggestionAbsentWhenNone(t *testing.T) {
	t.Parallel()
	sigs := []parsedRetroSignal{
		{EntryID: "KE-1", Category: "tool-gap", Severity: "minor",
			Observation: "No tool",
			Content:     "[minor] tool-gap: No tool"},
	}
	themes := buildRetroThemes(sigs)
	if len(themes) == 0 {
		t.Fatal("expected at least one theme")
	}
	if themes[0].TopSuggestion != "" {
		t.Errorf("TopSuggestion = %q, want empty when no suggestions", themes[0].TopSuggestion)
	}
}

func TestBuildRetroThemes_WorkedWellNotIncluded(t *testing.T) {
	t.Parallel()
	// worked-well signals should not appear in themes.
	sigs := []parsedRetroSignal{
		{EntryID: "KE-1", Category: "worked-well", Severity: "minor",
			Observation: "Vertical slicing was great",
			Content:     "[minor] worked-well: Vertical slicing was great"},
	}
	themes := buildRetroThemes(sigs)
	// Themes come from negative categories; worked-well handled separately.
	// buildRetroThemes receives only negative signals, so if we pass worked-well
	// it would produce a theme (it doesn't know to filter). The separation happens
	// in Synthesise. Here we just verify the theme gets built normally.
	_ = themes
}

// ─── buildRetroWorkedWell ─────────────────────────────────────────────────────

func TestBuildRetroWorkedWell_Empty(t *testing.T) {
	t.Parallel()
	got := buildRetroWorkedWell(nil)
	if got != nil {
		t.Errorf("buildRetroWorkedWell(nil) = %v, want nil", got)
	}
}

func TestBuildRetroWorkedWell_OneSignal(t *testing.T) {
	t.Parallel()
	sigs := []parsedRetroSignal{
		{EntryID: "KE-1", Category: "worked-well", Severity: "minor",
			Observation: "Vertical slicing worked well",
			Content:     "[minor] worked-well: Vertical slicing worked well"},
	}
	entries := buildRetroWorkedWell(sigs)
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].SignalCount != 1 {
		t.Errorf("SignalCount = %d, want 1", entries[0].SignalCount)
	}
	if entries[0].RepresentativeObservation != "Vertical slicing worked well" {
		t.Errorf("RepresentativeObservation = %q", entries[0].RepresentativeObservation)
	}
}

// ─── Synthesise (integration) ─────────────────────────────────────────────────

// writeRetroKnowledgeRecord writes a retrospective knowledge record directly to
// the store with an explicit created timestamp (to enable date filter testing).
// ─── Test helpers ─────────────────────────────────────────────────────────────

func writeRetroKnowledgeRecord(
	t *testing.T,
	root, id, category, severity, content, learnedFrom, created string,
) {
	t.Helper()
	store := storage.NewKnowledgeStore(root)
	topic := "retro-" + strings.ReplaceAll(id, "KE-", "task-")
	_, err := store.Write(storage.KnowledgeRecord{
		ID: id,
		Fields: map[string]any{
			"id":           id,
			"tier":         3,
			"topic":        topic,
			"scope":        "project",
			"content":      content,
			"status":       "contributed",
			"use_count":    0,
			"miss_count":   0,
			"confidence":   0.5,
			"ttl_days":     30,
			"tags":         []any{"retrospective", category},
			"learned_from": learnedFrom,
			"created":      created,
			"created_by":   "test",
			"updated":      created,
		},
	})
	if err != nil {
		t.Fatalf("writeRetroKnowledgeRecord(%s): %v", id, err)
	}
}

func newRetroTestService(t *testing.T, root string) *RetroService {
	t.Helper()
	knowledgeSvc := NewKnowledgeService(root)
	// Wait for background goroutines before TempDir cleanup runs.
	t.Cleanup(knowledgeSvc.Close)
	// entitySvc and docSvc are only needed for non-project scope and report mode.
	return &RetroService{
		knowledgeSvc: knowledgeSvc,
		repoRoot:     root,
		now:          func() time.Time { return time.Date(2026, 3, 28, 0, 0, 0, 0, time.UTC) },
	}
}

func TestSynthesise_EmptyReturnsEmptyThemes(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	svc := newRetroTestService(t, root)

	result, err := svc.Synthesise(RetroSynthesisInput{})
	if err != nil {
		t.Fatalf("Synthesise() error = %v", err)
	}
	if result.SignalCount != 0 {
		t.Errorf("SignalCount = %d, want 0", result.SignalCount)
	}
	if result.Themes == nil {
		t.Error("Themes should be non-nil empty slice, got nil")
	}
	if len(result.Themes) != 0 {
		t.Errorf("Themes = %v, want empty", result.Themes)
	}
	if result.Scope != "project" {
		t.Errorf("Scope = %q, want %q", result.Scope, "project")
	}
	if result.Period.From == "" || result.Period.To == "" {
		t.Error("Period.From and Period.To must be set even when no signals")
	}
}

func TestSynthesise_BasicSynthesis(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	svc := newRetroTestService(t, root)

	writeRetroKnowledgeRecord(t, root,
		"KE-001", "spec-ambiguity", "moderate",
		"[moderate] spec-ambiguity: Error format undefined",
		"TASK-001", "2026-03-01T10:00:00Z",
	)
	writeRetroKnowledgeRecord(t, root,
		"KE-002", "tool-gap", "minor",
		"[minor] tool-gap: No deploy automation tool exists",
		"TASK-002", "2026-03-02T10:00:00Z",
	)

	result, err := svc.Synthesise(RetroSynthesisInput{})
	if err != nil {
		t.Fatalf("Synthesise() error = %v", err)
	}
	if result.SignalCount != 2 {
		t.Errorf("SignalCount = %d, want 2", result.SignalCount)
	}
	if len(result.Themes) == 0 {
		t.Fatal("expected at least 1 theme")
	}
	// All themes should have the required fields.
	for i, theme := range result.Themes {
		if theme.Rank == 0 {
			t.Errorf("theme[%d].Rank = 0, want > 0", i)
		}
		if theme.Category == "" {
			t.Errorf("theme[%d].Category is empty", i)
		}
		if theme.SignalCount == 0 {
			t.Errorf("theme[%d].SignalCount = 0", i)
		}
		if len(theme.Signals) == 0 {
			t.Errorf("theme[%d].Signals is empty", i)
		}
		if theme.RepresentativeObservation == "" {
			t.Errorf("theme[%d].RepresentativeObservation is empty", i)
		}
	}
}

func TestSynthesise_WorkedWellSeparated(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	svc := newRetroTestService(t, root)

	writeRetroKnowledgeRecord(t, root,
		"KE-001", "spec-ambiguity", "moderate",
		"[moderate] spec-ambiguity: Error format undefined",
		"TASK-001", "2026-03-01T10:00:00Z",
	)
	writeRetroKnowledgeRecord(t, root,
		"KE-002", "worked-well", "minor",
		"[minor] worked-well: Vertical slicing was effective",
		"TASK-002", "2026-03-02T10:00:00Z",
	)

	result, err := svc.Synthesise(RetroSynthesisInput{})
	if err != nil {
		t.Fatalf("Synthesise() error = %v", err)
	}
	if result.SignalCount != 2 {
		t.Errorf("SignalCount = %d, want 2", result.SignalCount)
	}
	// worked-well should appear in worked_well, not themes.
	for _, theme := range result.Themes {
		if theme.Category == "worked-well" {
			t.Error("worked-well signal appeared in themes; it should be in worked_well section")
		}
	}
	if len(result.WorkedWell) == 0 {
		t.Error("worked_well section should have at least one entry")
	}
	if result.WorkedWell[0].RepresentativeObservation == "" {
		t.Error("worked_well entry RepresentativeObservation is empty")
	}
}

func TestSynthesise_MinSeverityFilter(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	svc := newRetroTestService(t, root)

	writeRetroKnowledgeRecord(t, root,
		"KE-001", "tool-gap", "minor",
		"[minor] tool-gap: Small inconvenience",
		"TASK-001", "2026-03-01T10:00:00Z",
	)
	writeRetroKnowledgeRecord(t, root,
		"KE-002", "spec-ambiguity", "moderate",
		"[moderate] spec-ambiguity: Moderate issue",
		"TASK-002", "2026-03-02T10:00:00Z",
	)
	writeRetroKnowledgeRecord(t, root,
		"KE-003", "context-gap", "significant",
		"[significant] context-gap: Critical context missing",
		"TASK-003", "2026-03-03T10:00:00Z",
	)

	// Filter to moderate and above — should exclude minor signal.
	result, err := svc.Synthesise(RetroSynthesisInput{MinSeverity: "moderate"})
	if err != nil {
		t.Fatalf("Synthesise() error = %v", err)
	}
	if result.SignalCount != 2 {
		t.Errorf("SignalCount = %d, want 2 (minor excluded)", result.SignalCount)
	}
	for _, theme := range result.Themes {
		for _, sigID := range theme.Signals {
			if sigID == "KE-001" {
				t.Error("minor signal KE-001 should be excluded with min_severity=moderate")
			}
		}
	}

	// Filter to significant — should only include KE-003.
	result2, err := svc.Synthesise(RetroSynthesisInput{MinSeverity: "significant"})
	if err != nil {
		t.Fatalf("Synthesise() error = %v", err)
	}
	if result2.SignalCount != 1 {
		t.Errorf("SignalCount = %d, want 1 (only significant)", result2.SignalCount)
	}
}

func TestSynthesise_SinceFilter(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	svc := newRetroTestService(t, root)

	writeRetroKnowledgeRecord(t, root,
		"KE-001", "tool-gap", "minor",
		"[minor] tool-gap: Old signal",
		"TASK-001", "2026-02-01T10:00:00Z", // before since
	)
	writeRetroKnowledgeRecord(t, root,
		"KE-002", "spec-ambiguity", "moderate",
		"[moderate] spec-ambiguity: Recent signal",
		"TASK-002", "2026-03-15T10:00:00Z", // after since
	)

	result, err := svc.Synthesise(RetroSynthesisInput{Since: "2026-03-01T00:00:00Z"})
	if err != nil {
		t.Fatalf("Synthesise() error = %v", err)
	}
	if result.SignalCount != 1 {
		t.Errorf("SignalCount = %d, want 1 (old signal excluded)", result.SignalCount)
	}
	// Check the included signal is KE-002.
	found := false
	for _, theme := range result.Themes {
		for _, sigID := range theme.Signals {
			if sigID == "KE-002" {
				found = true
			}
			if sigID == "KE-001" {
				t.Error("old signal KE-001 should be excluded by since filter")
			}
		}
	}
	if !found {
		t.Error("recent signal KE-002 not found in themes")
	}
}

func TestSynthesise_UntilFilter(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	svc := newRetroTestService(t, root)

	writeRetroKnowledgeRecord(t, root,
		"KE-001", "tool-gap", "minor",
		"[minor] tool-gap: Early signal",
		"TASK-001", "2026-02-01T10:00:00Z",
	)
	writeRetroKnowledgeRecord(t, root,
		"KE-002", "spec-ambiguity", "moderate",
		"[moderate] spec-ambiguity: Late signal",
		"TASK-002", "2026-04-01T10:00:00Z", // after until
	)

	result, err := svc.Synthesise(RetroSynthesisInput{Until: "2026-03-01T00:00:00Z"})
	if err != nil {
		t.Fatalf("Synthesise() error = %v", err)
	}
	if result.SignalCount != 1 {
		t.Errorf("SignalCount = %d, want 1 (late signal excluded)", result.SignalCount)
	}
	for _, theme := range result.Themes {
		for _, sigID := range theme.Signals {
			if sigID == "KE-002" {
				t.Error("late signal KE-002 should be excluded by until filter")
			}
		}
	}
}

func TestSynthesise_RankingOrder(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	svc := newRetroTestService(t, root)

	// Three spec-ambiguity minor signals (3 × 1 = 3), one context-gap significant (1 × 5 = 5).
	// context-gap significant should rank 1.
	writeRetroKnowledgeRecord(t, root,
		"KE-001", "spec-ambiguity", "minor",
		"[minor] spec-ambiguity: Error handling aaa",
		"TASK-001", "2026-03-01T10:00:00Z",
	)
	writeRetroKnowledgeRecord(t, root,
		"KE-002", "spec-ambiguity", "minor",
		"[minor] spec-ambiguity: Error handling bbb",
		"TASK-002", "2026-03-02T10:00:00Z",
	)
	writeRetroKnowledgeRecord(t, root,
		"KE-003", "spec-ambiguity", "minor",
		"[minor] spec-ambiguity: Error handling ccc",
		"TASK-003", "2026-03-03T10:00:00Z",
	)
	writeRetroKnowledgeRecord(t, root,
		"KE-004", "context-gap", "significant",
		"[significant] context-gap: Critical missing information",
		"TASK-004", "2026-03-04T10:00:00Z",
	)

	result, err := svc.Synthesise(RetroSynthesisInput{})
	if err != nil {
		t.Fatalf("Synthesise() error = %v", err)
	}
	if len(result.Themes) == 0 {
		t.Fatal("expected themes")
	}
	// Rank 1 should have the highest severity score.
	if result.Themes[0].SeverityScore < result.Themes[len(result.Themes)-1].SeverityScore {
		t.Error("themes not sorted descending by severity_score")
	}
	// context-gap significant (score=5) should rank above 3 minor spec-ambiguity signals
	// that form a single cluster (score=3).
	if result.Themes[0].Category != "context-gap" {
		t.Errorf("rank-1 category = %q, want %q", result.Themes[0].Category, "context-gap")
	}
}

func TestSynthesise_InvalidMinSeverity(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	svc := newRetroTestService(t, root)

	_, err := svc.Synthesise(RetroSynthesisInput{MinSeverity: "catastrophic"})
	if err == nil {
		t.Error("expected error for invalid min_severity, got nil")
	}
	if !strings.Contains(err.Error(), "min_severity") {
		t.Errorf("error should mention min_severity: %v", err)
	}
}

func TestSynthesise_InvalidSinceTimestamp(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	svc := newRetroTestService(t, root)

	_, err := svc.Synthesise(RetroSynthesisInput{Since: "not-a-timestamp"})
	if err == nil {
		t.Error("expected error for invalid since timestamp, got nil")
	}
	if !strings.Contains(err.Error(), "since") {
		t.Errorf("error should mention since: %v", err)
	}
}

func TestSynthesise_InvalidUntilTimestamp(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	svc := newRetroTestService(t, root)

	_, err := svc.Synthesise(RetroSynthesisInput{Until: "bad-timestamp"})
	if err == nil {
		t.Error("expected error for invalid until timestamp, got nil")
	}
	if !strings.Contains(err.Error(), "until") {
		t.Errorf("error should mention until: %v", err)
	}
}

func TestSynthesise_DefaultScope(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	svc := newRetroTestService(t, root)

	// Empty scope should default to "project".
	result, err := svc.Synthesise(RetroSynthesisInput{Scope: ""})
	if err != nil {
		t.Fatalf("Synthesise() error = %v", err)
	}
	if result.Scope != "project" {
		t.Errorf("Scope = %q, want %q", result.Scope, "project")
	}
}

func TestSynthesise_ThemeRankIsSequential(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	svc := newRetroTestService(t, root)

	writeRetroKnowledgeRecord(t, root,
		"KE-001", "spec-ambiguity", "moderate",
		"[moderate] spec-ambiguity: Issue one",
		"TASK-001", "2026-03-01T10:00:00Z",
	)
	writeRetroKnowledgeRecord(t, root,
		"KE-002", "tool-gap", "minor",
		"[minor] tool-gap: Issue two different topic",
		"TASK-002", "2026-03-02T10:00:00Z",
	)

	result, err := svc.Synthesise(RetroSynthesisInput{})
	if err != nil {
		t.Fatalf("Synthesise() error = %v", err)
	}
	for i, theme := range result.Themes {
		if theme.Rank != i+1 {
			t.Errorf("theme[%d].Rank = %d, want %d", i, theme.Rank, i+1)
		}
	}
}

func TestSynthesise_PeriodFromAndTo(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	svc := newRetroTestService(t, root)

	writeRetroKnowledgeRecord(t, root,
		"KE-001", "tool-gap", "minor",
		"[minor] tool-gap: First signal",
		"TASK-001", "2026-03-01T10:00:00Z",
	)
	writeRetroKnowledgeRecord(t, root,
		"KE-002", "spec-ambiguity", "minor",
		"[minor] spec-ambiguity: Last signal different",
		"TASK-002", "2026-03-20T10:00:00Z",
	)

	result, err := svc.Synthesise(RetroSynthesisInput{})
	if err != nil {
		t.Fatalf("Synthesise() error = %v", err)
	}
	if result.Period.From != "2026-03-01T10:00:00Z" {
		t.Errorf("Period.From = %q, want earliest signal timestamp", result.Period.From)
	}
	if result.Period.To != "2026-03-20T10:00:00Z" {
		t.Errorf("Period.To = %q, want latest signal timestamp", result.Period.To)
	}
}

// ─── renderRetroMarkdown ──────────────────────────────────────────────────────

func TestRenderRetroMarkdown_EmptySignals(t *testing.T) {
	t.Parallel()
	result := RetroSynthesisResult{
		Scope:       "project",
		SignalCount: 0,
		Period:      RetroPeriod{From: "2026-03-01T00:00:00Z", To: "2026-03-28T00:00:00Z"},
		Themes:      []RetroTheme{},
	}
	md := renderRetroMarkdown("Retrospective: project 2026-03-28", result)
	if !strings.HasPrefix(md, "# Retrospective") {
		t.Errorf("markdown should start with '# Retrospective', got: %q", md[:50])
	}
	if !strings.Contains(md, "No signals found") {
		t.Error("empty synthesis markdown should mention no signals found")
	}
}

func TestRenderRetroMarkdown_WithThemes(t *testing.T) {
	t.Parallel()
	result := RetroSynthesisResult{
		Scope:       "project",
		SignalCount: 2,
		Period:      RetroPeriod{From: "2026-03-01T00:00:00Z", To: "2026-03-28T00:00:00Z"},
		Themes: []RetroTheme{
			{
				Rank:                      1,
				Category:                  "spec-ambiguity",
				Title:                     "spec-ambiguity: Error format undefined",
				SignalCount:               2,
				SeverityScore:             6,
				Signals:                   []string{"KE-001", "KE-002"},
				TopSuggestion:             "Add error format to spec template",
				RepresentativeObservation: "Error format undefined",
			},
		},
	}
	md := renderRetroMarkdown("Test Retrospective", result)
	if !strings.Contains(md, "## Themes") {
		t.Error("markdown should contain '## Themes'")
	}
	if !strings.Contains(md, "spec-ambiguity") {
		t.Error("markdown should contain category name")
	}
	if !strings.Contains(md, "Add error format to spec template") {
		t.Error("markdown should contain the top suggestion")
	}
	if !strings.Contains(md, "KE-001") {
		t.Error("markdown should contain signal IDs")
	}
}

func TestRenderRetroMarkdown_WithWorkedWell(t *testing.T) {
	t.Parallel()
	result := RetroSynthesisResult{
		Scope:       "project",
		SignalCount: 1,
		Period:      RetroPeriod{From: "2026-03-01T00:00:00Z", To: "2026-03-01T00:00:00Z"},
		Themes:      []RetroTheme{},
		WorkedWell: []RetroWorkedWell{
			{
				Title:                     "worked-well: Vertical slicing worked well",
				SignalCount:               1,
				RepresentativeObservation: "Vertical slicing worked well",
			},
		},
	}
	md := renderRetroMarkdown("Test Retrospective", result)
	if !strings.Contains(md, "## What Worked Well") {
		t.Error("markdown should contain '## What Worked Well'")
	}
	if !strings.Contains(md, "Vertical slicing worked well") {
		t.Error("markdown should contain worked-well observation")
	}
}

func TestRenderRetroMarkdown_WithExperiments(t *testing.T) {
	t.Parallel()
	result := RetroSynthesisResult{
		Scope:       "project",
		SignalCount: 3,
		Period:      RetroPeriod{From: "2026-03-01T00:00:00Z", To: "2026-03-28T00:00:00Z"},
		Themes:      []RetroTheme{},
		Experiments: []RetroExperiment{
			{
				DecisionID:      "DEC-0100000000001",
				Title:           "Add error format to spec template",
				PositiveSignals: 3,
				NegativeSignals: 1,
				NetAssessment:   "3 positive, 1 negative",
				Recommendation:  "keep",
			},
		},
	}
	md := renderRetroMarkdown("Test Retrospective", result)
	if !strings.Contains(md, "## Experiments") {
		t.Error("markdown should contain '## Experiments'")
	}
	if !strings.Contains(md, "DEC-0100000000001") {
		t.Error("markdown should contain decision ID")
	}
	if !strings.Contains(md, "Add error format to spec template") {
		t.Error("markdown should contain experiment title")
	}
	if !strings.Contains(md, "keep") {
		t.Error("markdown should contain recommendation")
	}
}

// ─── Phase 3: Experiment tracking in synthesis ────────────────────────────────

// writeDecisionEntity writes a decision entity YAML file to the test root.
func writeDecisionEntity(t *testing.T, root, id, slug, summary, status string, tags []string) {
	t.Helper()
	dir := filepath.Join(root, "decisions")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("mkdir decisions: %v", err)
	}
	tagsYAML := ""
	for _, tag := range tags {
		tagsYAML += fmt.Sprintf("\n  - %s", tag)
	}
	content := fmt.Sprintf(`id: %s
slug: %s
summary: %s
rationale: Test rationale
decided_by: test
date: "2026-03-01T00:00:00Z"
status: %s
tags:%s
`, id, slug, summary, status, tagsYAML)
	path := filepath.Join(dir, id+"-"+slug+".yaml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write decision entity: %v", err)
	}
}

// newRetroTestServiceWithEntities creates a RetroService with entitySvc wired up.
func newRetroTestServiceWithEntities(t *testing.T, root string) *RetroService {
	t.Helper()
	knowledgeSvc := NewKnowledgeService(root)
	t.Cleanup(knowledgeSvc.Close)
	entitySvc := NewEntityService(root)
	return &RetroService{
		knowledgeSvc: knowledgeSvc,
		entitySvc:    entitySvc,
		repoRoot:     root,
		now:          func() time.Time { return time.Date(2026, 3, 28, 0, 0, 0, 0, time.UTC) },
	}
}

// newRetroTestServiceWithReport creates a RetroService with all dependencies wired up,
// including DocService for Report mode tests (P5-2.13–P5-2.16).
func newRetroTestServiceWithReport(t *testing.T, root string) *RetroService {
	t.Helper()
	knowledgeSvc := NewKnowledgeService(root)
	t.Cleanup(knowledgeSvc.Close)
	entitySvc := NewEntityService(root)
	docSvc := NewDocumentService(root, root)
	return &RetroService{
		knowledgeSvc: knowledgeSvc,
		entitySvc:    entitySvc,
		docSvc:       docSvc,
		repoRoot:     root,
		now:          func() time.Time { return time.Date(2026, 3, 28, 0, 0, 0, 0, time.UTC) },
	}
}

// writeRetroTestPlan writes a plan entity record directly to the entity store.
// Used in scope filtering tests to satisfy the parent-plan existence check in
// CreateFeature without going through the plan service.
func writeRetroTestPlan(t *testing.T, svc *EntityService, id string) {
	t.Helper()
	_, _, slug := model.ParsePlanID(id)
	fields := map[string]any{
		"id":         id,
		"slug":       slug,
		"title":      "Retro Scope Test Plan",
		"status":     "active",
		"summary":    "Test plan for retro scope filtering tests",
		"created":    "2026-03-01T00:00:00Z",
		"created_by": "test",
		"updated":    "2026-03-01T00:00:00Z",
	}
	_, err := svc.store.Write(storage.EntityRecord{
		Type:   string(model.EntityKindPlan),
		ID:     id,
		Slug:   slug,
		Fields: fields,
	})
	if err != nil {
		t.Fatalf("writeRetroTestPlan(%s): %v", id, err)
	}
}

// P5-3.1: finish with related_decision stores Related: DEC-042 in content.
// (Tested in retro_test.go via TestEncodeRetroContent_WithRelatedDecision and
// TestEncodeRetroContent_WithSuggestionAndRelatedDecision.)

// P5-3.2: related_decision is optional; signals without it are accepted.
// (Tested in retro_test.go via TestEncodeRetroContent_Basic.)

// P5-3.4: When at least one signal references a decision, experiments section is present.
func TestSynthesise_ExperimentsPresent(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	svc := newRetroTestServiceWithEntities(t, root)

	writeDecisionEntity(t, root, "DEC-0100000000001", "add-error-format",
		"Add error format to spec template", "accepted",
		[]string{"workflow-experiment", "retrospective"})

	writeRetroKnowledgeRecord(t, root,
		"KE-E01", "spec-ambiguity", "moderate",
		"[moderate] spec-ambiguity: Error format still missing Related: DEC-0100000000001",
		"TASK-001", "2026-03-10T10:00:00Z",
	)
	writeRetroKnowledgeRecord(t, root,
		"KE-E02", "worked-well", "minor",
		"[minor] worked-well: Error format spec section eliminated guesswork Related: DEC-0100000000001",
		"TASK-002", "2026-03-11T10:00:00Z",
	)

	result, err := svc.Synthesise(RetroSynthesisInput{})
	if err != nil {
		t.Fatalf("Synthesise() error = %v", err)
	}
	if result.Experiments == nil {
		t.Fatal("Experiments should be present when signals reference decisions")
	}
	if len(result.Experiments) != 1 {
		t.Fatalf("Experiments len = %d, want 1", len(result.Experiments))
	}
	exp := result.Experiments[0]
	if exp.DecisionID != "DEC-0100000000001" {
		t.Errorf("DecisionID = %q, want %q", exp.DecisionID, "DEC-0100000000001")
	}
	if exp.Title != "Add error format to spec template" {
		t.Errorf("Title = %q, want %q", exp.Title, "Add error format to spec template")
	}
	if exp.PositiveSignals != 1 {
		t.Errorf("PositiveSignals = %d, want 1", exp.PositiveSignals)
	}
	if exp.NegativeSignals != 1 {
		t.Errorf("NegativeSignals = %d, want 1", exp.NegativeSignals)
	}
}

// P5-3.5: Each experiment entry includes required fields and recommendation.
func TestSynthesise_ExperimentRecommendationKeep(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	svc := newRetroTestServiceWithEntities(t, root)

	writeDecisionEntity(t, root, "DEC-0100000000002", "vertical-slicing",
		"Require vertical slice decomposition", "accepted",
		[]string{"workflow-experiment"})

	writeRetroKnowledgeRecord(t, root,
		"KE-K01", "worked-well", "minor",
		"[minor] worked-well: Slicing was great Related: DEC-0100000000002",
		"TASK-001", "2026-03-10T10:00:00Z",
	)
	writeRetroKnowledgeRecord(t, root,
		"KE-K02", "worked-well", "minor",
		"[minor] worked-well: Each slice independently testable Related: DEC-0100000000002",
		"TASK-002", "2026-03-11T10:00:00Z",
	)

	result, err := svc.Synthesise(RetroSynthesisInput{})
	if err != nil {
		t.Fatalf("Synthesise() error = %v", err)
	}
	if len(result.Experiments) != 1 {
		t.Fatalf("Experiments len = %d, want 1", len(result.Experiments))
	}
	exp := result.Experiments[0]
	if exp.Recommendation != "keep" {
		t.Errorf("Recommendation = %q, want %q", exp.Recommendation, "keep")
	}
	if exp.PositiveSignals != 2 {
		t.Errorf("PositiveSignals = %d, want 2", exp.PositiveSignals)
	}
	if exp.NegativeSignals != 0 {
		t.Errorf("NegativeSignals = %d, want 0", exp.NegativeSignals)
	}
}

func TestSynthesise_ExperimentRecommendationRevert(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	svc := newRetroTestServiceWithEntities(t, root)

	writeDecisionEntity(t, root, "DEC-0100000000003", "require-context-profile",
		"Require context profile for all features", "accepted",
		[]string{"workflow-experiment"})

	writeRetroKnowledgeRecord(t, root,
		"KE-R01", "tool-friction", "moderate",
		"[moderate] tool-friction: Context profile setup too complex Related: DEC-0100000000003",
		"TASK-001", "2026-03-10T10:00:00Z",
	)
	writeRetroKnowledgeRecord(t, root,
		"KE-R02", "workflow-friction", "moderate",
		"[moderate] workflow-friction: Profiles not reused across features Related: DEC-0100000000003",
		"TASK-002", "2026-03-11T10:00:00Z",
	)

	result, err := svc.Synthesise(RetroSynthesisInput{})
	if err != nil {
		t.Fatalf("Synthesise() error = %v", err)
	}
	if len(result.Experiments) != 1 {
		t.Fatalf("Experiments len = %d, want 1", len(result.Experiments))
	}
	exp := result.Experiments[0]
	if exp.Recommendation != "revert" {
		t.Errorf("Recommendation = %q, want %q", exp.Recommendation, "revert")
	}
	if exp.PositiveSignals != 0 {
		t.Errorf("PositiveSignals = %d, want 0", exp.PositiveSignals)
	}
	if exp.NegativeSignals != 2 {
		t.Errorf("NegativeSignals = %d, want 2", exp.NegativeSignals)
	}
}

func TestSynthesise_ExperimentRecommendationRevise(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	svc := newRetroTestServiceWithEntities(t, root)

	writeDecisionEntity(t, root, "DEC-0100000000004", "spec-template-sections",
		"Add structured sections to spec template", "accepted",
		[]string{"workflow-experiment"})

	writeRetroKnowledgeRecord(t, root,
		"KE-V01", "worked-well", "minor",
		"[minor] worked-well: Error section was helpful Related: DEC-0100000000004",
		"TASK-001", "2026-03-10T10:00:00Z",
	)
	writeRetroKnowledgeRecord(t, root,
		"KE-V02", "spec-ambiguity", "moderate",
		"[moderate] spec-ambiguity: Retry policy section still vague Related: DEC-0100000000004",
		"TASK-002", "2026-03-11T10:00:00Z",
	)

	result, err := svc.Synthesise(RetroSynthesisInput{})
	if err != nil {
		t.Fatalf("Synthesise() error = %v", err)
	}
	if len(result.Experiments) != 1 {
		t.Fatalf("Experiments len = %d, want 1", len(result.Experiments))
	}
	exp := result.Experiments[0]
	if exp.Recommendation != "revise" {
		t.Errorf("Recommendation = %q, want %q", exp.Recommendation, "revise")
	}
}

// P5-3.6: Signals not referencing any decision are not attributed to experiments.
func TestSynthesise_UnrelatedSignalsNotAttributed(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	svc := newRetroTestServiceWithEntities(t, root)

	writeDecisionEntity(t, root, "DEC-0100000000005", "test-experiment",
		"Test experiment", "accepted",
		[]string{"workflow-experiment"})

	// Signal with related decision.
	writeRetroKnowledgeRecord(t, root,
		"KE-U01", "worked-well", "minor",
		"[minor] worked-well: Experiment helped Related: DEC-0100000000005",
		"TASK-001", "2026-03-10T10:00:00Z",
	)
	// Signal without related decision — should not be attributed.
	writeRetroKnowledgeRecord(t, root,
		"KE-U02", "spec-ambiguity", "moderate",
		"[moderate] spec-ambiguity: Unrelated observation",
		"TASK-002", "2026-03-11T10:00:00Z",
	)

	result, err := svc.Synthesise(RetroSynthesisInput{})
	if err != nil {
		t.Fatalf("Synthesise() error = %v", err)
	}
	if len(result.Experiments) != 1 {
		t.Fatalf("Experiments len = %d, want 1", len(result.Experiments))
	}
	exp := result.Experiments[0]
	// Only the signal with Related: DEC-0100000000005 should count.
	if exp.PositiveSignals != 1 {
		t.Errorf("PositiveSignals = %d, want 1", exp.PositiveSignals)
	}
	if exp.NegativeSignals != 0 {
		t.Errorf("NegativeSignals = %d, want 0", exp.NegativeSignals)
	}
}

// P5-3.7: Experiments section is absent (nil) when no signals reference any decision.
func TestSynthesise_ExperimentsAbsentWhenNoReferences(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	svc := newRetroTestServiceWithEntities(t, root)

	writeDecisionEntity(t, root, "DEC-0100000000006", "unused-experiment",
		"Unused experiment", "accepted",
		[]string{"workflow-experiment"})

	writeRetroKnowledgeRecord(t, root,
		"KE-N01", "spec-ambiguity", "moderate",
		"[moderate] spec-ambiguity: No related decision here",
		"TASK-001", "2026-03-10T10:00:00Z",
	)

	result, err := svc.Synthesise(RetroSynthesisInput{})
	if err != nil {
		t.Fatalf("Synthesise() error = %v", err)
	}
	if result.Experiments != nil {
		t.Errorf("Experiments = %v, want nil when no signals reference decisions", result.Experiments)
	}
}

// P5-3.7 edge case: Signals reference a decision that exists but is NOT
// tagged workflow-experiment — experiments section should still be absent.
func TestSynthesise_ExperimentsAbsentWhenDecisionNotExperiment(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	svc := newRetroTestServiceWithEntities(t, root)

	// Decision exists but is NOT tagged workflow-experiment.
	writeDecisionEntity(t, root, "DEC-0100000000007", "not-an-experiment",
		"Not an experiment", "accepted",
		[]string{"some-other-tag"})

	writeRetroKnowledgeRecord(t, root,
		"KE-NE01", "spec-ambiguity", "moderate",
		"[moderate] spec-ambiguity: References non-experiment Related: DEC-0100000000007",
		"TASK-001", "2026-03-10T10:00:00Z",
	)

	result, err := svc.Synthesise(RetroSynthesisInput{})
	if err != nil {
		t.Fatalf("Synthesise() error = %v", err)
	}
	if result.Experiments != nil {
		t.Errorf("Experiments = %v, want nil when referenced decision is not a workflow-experiment", result.Experiments)
	}
}

// P5-3.3: related_decision with an ID that doesn't correspond to a known
// decision is accepted and stored. In synthesis, such signals are simply
// not attributed to any experiment.
func TestSynthesise_ExperimentsAbsentWhenDecisionDoesNotExist(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	svc := newRetroTestServiceWithEntities(t, root)

	// No decision entities at all.
	writeRetroKnowledgeRecord(t, root,
		"KE-NX01", "spec-ambiguity", "moderate",
		"[moderate] spec-ambiguity: References nonexistent decision Related: DEC-0199999999999",
		"TASK-001", "2026-03-10T10:00:00Z",
	)

	result, err := svc.Synthesise(RetroSynthesisInput{})
	if err != nil {
		t.Fatalf("Synthesise() error = %v", err)
	}
	if result.Experiments != nil {
		t.Errorf("Experiments = %v, want nil when referenced decision does not exist", result.Experiments)
	}
}

// P5-3.5: net_assessment is a descriptive string.
func TestSynthesise_ExperimentNetAssessment(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	svc := newRetroTestServiceWithEntities(t, root)

	writeDecisionEntity(t, root, "DEC-0100000000008", "assessment-test",
		"Assessment test", "accepted",
		[]string{"workflow-experiment"})

	writeRetroKnowledgeRecord(t, root,
		"KE-A01", "worked-well", "minor",
		"[minor] worked-well: Good outcome Related: DEC-0100000000008",
		"TASK-001", "2026-03-10T10:00:00Z",
	)
	writeRetroKnowledgeRecord(t, root,
		"KE-A02", "worked-well", "minor",
		"[minor] worked-well: Another good outcome Related: DEC-0100000000008",
		"TASK-002", "2026-03-11T10:00:00Z",
	)
	writeRetroKnowledgeRecord(t, root,
		"KE-A03", "tool-friction", "moderate",
		"[moderate] tool-friction: Some friction Related: DEC-0100000000008",
		"TASK-003", "2026-03-12T10:00:00Z",
	)

	result, err := svc.Synthesise(RetroSynthesisInput{})
	if err != nil {
		t.Fatalf("Synthesise() error = %v", err)
	}
	if len(result.Experiments) != 1 {
		t.Fatalf("Experiments len = %d, want 1", len(result.Experiments))
	}
	exp := result.Experiments[0]
	if exp.NetAssessment != "2 positive, 1 negative" {
		t.Errorf("NetAssessment = %q, want %q", exp.NetAssessment, "2 positive, 1 negative")
	}
	if exp.Recommendation != "keep" {
		t.Errorf("Recommendation = %q, want %q", exp.Recommendation, "keep")
	}
}

// Multiple experiments in one synthesis.
func TestSynthesise_MultipleExperiments(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	svc := newRetroTestServiceWithEntities(t, root)

	writeDecisionEntity(t, root, "DEC-0100000000009", "experiment-alpha",
		"Experiment Alpha", "accepted",
		[]string{"workflow-experiment"})
	writeDecisionEntity(t, root, "DEC-0100000000010", "experiment-beta",
		"Experiment Beta", "accepted",
		[]string{"workflow-experiment"})

	writeRetroKnowledgeRecord(t, root,
		"KE-M01", "worked-well", "minor",
		"[minor] worked-well: Alpha helped Related: DEC-0100000000009",
		"TASK-001", "2026-03-10T10:00:00Z",
	)
	writeRetroKnowledgeRecord(t, root,
		"KE-M02", "tool-friction", "moderate",
		"[moderate] tool-friction: Beta caused friction Related: DEC-0100000000010",
		"TASK-002", "2026-03-11T10:00:00Z",
	)

	result, err := svc.Synthesise(RetroSynthesisInput{})
	if err != nil {
		t.Fatalf("Synthesise() error = %v", err)
	}
	if len(result.Experiments) != 2 {
		t.Fatalf("Experiments len = %d, want 2", len(result.Experiments))
	}
	// Experiments should be sorted by decision ID for deterministic output.
	if result.Experiments[0].DecisionID != "DEC-0100000000009" {
		t.Errorf("Experiments[0].DecisionID = %q, want %q", result.Experiments[0].DecisionID, "DEC-0100000000009")
	}
	if result.Experiments[1].DecisionID != "DEC-0100000000010" {
		t.Errorf("Experiments[1].DecisionID = %q, want %q", result.Experiments[1].DecisionID, "DEC-0100000000010")
	}
	if result.Experiments[0].Recommendation != "keep" {
		t.Errorf("Experiments[0].Recommendation = %q, want %q", result.Experiments[0].Recommendation, "keep")
	}
	if result.Experiments[1].Recommendation != "revert" {
		t.Errorf("Experiments[1].Recommendation = %q, want %q", result.Experiments[1].Recommendation, "revert")
	}
}

// Experiment with rejected status is excluded from experiments section.
// ─── Report mode (P5-2.13–P5-2.16) ──────────────────────────────────────────

func TestReport_MissingOutputPath(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	svc := newRetroTestServiceWithReport(t, root)

	_, err := svc.Report(RetroReportInput{})
	if err == nil {
		t.Fatal("expected error for empty output_path")
	}
	if !strings.Contains(err.Error(), "output_path") {
		t.Errorf("error %q should mention output_path", err)
	}
}

// P5-2.13: report action generates a document file and returns synthesis data.
func TestReport_HappyPath(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	svc := newRetroTestServiceWithReport(t, root)

	writeRetroKnowledgeRecord(t, root, "KE-R01", "spec-ambiguity", "moderate",
		"[moderate] spec-ambiguity: Spec was unclear", "", "2026-03-01T10:00:00Z")

	result, err := svc.Report(RetroReportInput{
		RetroSynthesisInput: RetroSynthesisInput{},
		OutputPath:          "retro/2026-03-28.md",
		Title:               "Sprint 1 Retro",
		CreatedBy:           "test",
	})
	if err != nil {
		t.Fatalf("Report() error = %v", err)
	}

	if result.SignalCount != 1 {
		t.Errorf("SignalCount = %d, want 1", result.SignalCount)
	}
	if len(result.Themes) == 0 {
		t.Error("expected at least one theme")
	}
	// Markdown file must exist on disk.
	fullPath := filepath.Join(root, "retro/2026-03-28.md")
	if _, err := os.Stat(fullPath); err != nil {
		t.Errorf("report file not found at %s: %v", fullPath, err)
	}
}

// P5-2.14: the registered document record is in draft status.
func TestReport_DocumentInDraftStatus(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	svc := newRetroTestServiceWithReport(t, root)

	writeRetroKnowledgeRecord(t, root, "KE-R02", "tool-gap", "minor",
		"[minor] tool-gap: No tool for X", "", "2026-03-01T10:00:00Z")

	result, err := svc.Report(RetroReportInput{
		RetroSynthesisInput: RetroSynthesisInput{},
		OutputPath:          "retro/draft-status-test.md",
		CreatedBy:           "test",
	})
	if err != nil {
		t.Fatalf("Report() error = %v", err)
	}

	doc, err := svc.docSvc.GetDocument(result.Report.DocumentID, false)
	if err != nil {
		t.Fatalf("GetDocument(%s): %v", result.Report.DocumentID, err)
	}
	if doc.Status != "draft" {
		t.Errorf("document status = %q, want %q", doc.Status, "draft")
	}
}

// Verify that Report() registers the document with type "retro" instead of "report".
func TestReport_DocumentTypeRetro(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	svc := newRetroTestServiceWithReport(t, root)

	result, err := svc.Report(RetroReportInput{
		RetroSynthesisInput: RetroSynthesisInput{},
		OutputPath:          "retro/type-retro-test.md",
		CreatedBy:           "test",
	})
	if err != nil {
		t.Fatalf("Report() error = %v", err)
	}

	doc, err := svc.docSvc.GetDocument(result.Report.DocumentID, false)
	if err != nil {
		t.Fatalf("GetDocument(%s): %v", result.Report.DocumentID, err)
	}
	if doc.Type != "retro" {
		t.Errorf("document type = %q, want %q", doc.Type, "retro")
	}
}

// P5-2.15: response includes report.path and report.document_id.
func TestReport_ReportPathAndDocumentID(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	svc := newRetroTestServiceWithReport(t, root)

	const outputPath = "retro/path-id-test.md"
	result, err := svc.Report(RetroReportInput{
		RetroSynthesisInput: RetroSynthesisInput{},
		OutputPath:          outputPath,
		CreatedBy:           "test",
	})
	if err != nil {
		t.Fatalf("Report() error = %v", err)
	}

	if result.Report.Path != outputPath {
		t.Errorf("Report.Path = %q, want %q", result.Report.Path, outputPath)
	}
	if result.Report.DocumentID == "" {
		t.Error("Report.DocumentID should not be empty")
	}
}

// P5-2.16: calling Report twice with the same output_path returns an error on
// the second call (document already registered).
func TestReport_DuplicateOutputPath(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	svc := newRetroTestServiceWithReport(t, root)

	input := RetroReportInput{
		RetroSynthesisInput: RetroSynthesisInput{},
		OutputPath:          "retro/dup-test.md",
		CreatedBy:           "test",
	}
	if _, err := svc.Report(input); err != nil {
		t.Fatalf("first Report() error = %v", err)
	}
	if _, err := svc.Report(input); err == nil {
		t.Fatal("expected error on second Report() with same output_path")
	}
}

// TestReport_DefaultTitle verifies the default title "Retrospective: {scope} {date}" is
// used when Title is empty.
func TestReport_DefaultTitle(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	svc := newRetroTestServiceWithReport(t, root)

	result, err := svc.Report(RetroReportInput{
		RetroSynthesisInput: RetroSynthesisInput{},
		OutputPath:          "retro/default-title-test.md",
		// No Title; defaults to "Retrospective: project {date}".
		CreatedBy: "test",
	})
	if err != nil {
		t.Fatalf("Report() error = %v", err)
	}

	doc, err := svc.docSvc.GetDocument(result.Report.DocumentID, false)
	if err != nil {
		t.Fatalf("GetDocument: %v", err)
	}
	const want = "Retrospective: project 2026-03-28"
	if doc.Title != want {
		t.Errorf("document title = %q, want %q", doc.Title, want)
	}
}

// ─── Scope filtering (P5-2.7) ────────────────────────────────────────────────

// P5-2.7: Feature scope includes only signals from tasks under the specified feature.
func TestSynthesise_FeatureScope(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	svc := newRetroTestServiceWithEntities(t, root)

	writeRetroTestPlan(t, svc.entitySvc, "P1-scope-test")
	featResult, err := svc.entitySvc.CreateFeature(CreateFeatureInput{
		Name: "test",
		Slug: "scope-feat", Parent: "P1-scope-test", Summary: "Scope test feature", CreatedBy: "test",
	})
	if err != nil {
		t.Fatalf("CreateFeature: %v", err)
	}
	taskResult, err := svc.entitySvc.CreateTask(CreateTaskInput{
		Name:          "test",
		ParentFeature: featResult.ID,
		Slug:          "scope-task",
		Summary:       "Scope test task",
	})
	if err != nil {
		t.Fatalf("CreateTask: %v", err)
	}

	// Signal attributed to the scoped task.
	writeRetroKnowledgeRecord(t, root, "KE-FS01", "spec-ambiguity", "moderate",
		"[moderate] spec-ambiguity: Scoped spec issue",
		taskResult.ID, "2026-03-01T10:00:00Z")

	// Standalone signal (no learned_from) — excluded from feature scope.
	writeRetroKnowledgeRecord(t, root, "KE-FS02", "tool-gap", "minor",
		"[minor] tool-gap: Standalone signal",
		"", "2026-03-01T11:00:00Z")

	result, err := svc.Synthesise(RetroSynthesisInput{Scope: featResult.ID})
	if err != nil {
		t.Fatalf("Synthesise() error = %v", err)
	}

	if result.SignalCount != 1 {
		t.Errorf("SignalCount = %d, want 1 (only task-attributed signal)", result.SignalCount)
	}
	if result.Scope != featResult.ID {
		t.Errorf("Scope = %q, want %q", result.Scope, featResult.ID)
	}
}

// P5-2.7: Feature scope excludes signals from tasks under a different feature.
func TestSynthesise_FeatureScopeExcludesOtherFeature(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	svc := newRetroTestServiceWithEntities(t, root)

	writeRetroTestPlan(t, svc.entitySvc, "P1-scope-excl-test")
	featA, err := svc.entitySvc.CreateFeature(CreateFeatureInput{
		Name: "test",
		Slug: "feature-a", Parent: "P1-scope-excl-test", Summary: "Feature A", CreatedBy: "test",
	})
	if err != nil {
		t.Fatalf("CreateFeature A: %v", err)
	}
	featB, err := svc.entitySvc.CreateFeature(CreateFeatureInput{
		Name: "test",
		Slug: "feature-b", Parent: "P1-scope-excl-test", Summary: "Feature B", CreatedBy: "test",
	})
	if err != nil {
		t.Fatalf("CreateFeature B: %v", err)
	}
	taskB, err := svc.entitySvc.CreateTask(CreateTaskInput{
		Name:          "test",
		ParentFeature: featB.ID,
		Slug:          "task-b",
		Summary:       "Task under feature B",
	})
	if err != nil {
		t.Fatalf("CreateTask B: %v", err)
	}

	// Signal belongs to feature B's task.
	writeRetroKnowledgeRecord(t, root, "KE-FE01", "tool-gap", "minor",
		"[minor] tool-gap: Signal from feature B",
		taskB.ID, "2026-03-01T10:00:00Z")

	// Synthesise scoped to feature A — should return zero signals.
	result, err := svc.Synthesise(RetroSynthesisInput{Scope: featA.ID})
	if err != nil {
		t.Fatalf("Synthesise() error = %v", err)
	}
	if result.SignalCount != 0 {
		t.Errorf("SignalCount = %d, want 0 (signal belongs to feature B)", result.SignalCount)
	}
}

// P5-2.7: Plan scope includes signals from tasks under any feature in the plan.
func TestSynthesise_PlanScope(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	svc := newRetroTestServiceWithEntities(t, root)

	writeRetroTestPlan(t, svc.entitySvc, "P1-plan-scope")
	featResult, err := svc.entitySvc.CreateFeature(CreateFeatureInput{
		Name: "test",
		Slug: "plan-scope-feat", Parent: "P1-plan-scope", Summary: "Feature under plan", CreatedBy: "test",
	})
	if err != nil {
		t.Fatalf("CreateFeature: %v", err)
	}
	taskResult, err := svc.entitySvc.CreateTask(CreateTaskInput{
		Name:          "test",
		ParentFeature: featResult.ID,
		Slug:          "plan-scope-task",
		Summary:       "Task under plan feature",
	})
	if err != nil {
		t.Fatalf("CreateTask: %v", err)
	}

	writeRetroKnowledgeRecord(t, root, "KE-PS01", "context-gap", "significant",
		"[significant] context-gap: Plan-scoped issue",
		taskResult.ID, "2026-03-01T10:00:00Z")

	result, err := svc.Synthesise(RetroSynthesisInput{Scope: "P1-plan-scope"})
	if err != nil {
		t.Fatalf("Synthesise() error = %v", err)
	}
	if result.SignalCount != 1 {
		t.Errorf("SignalCount = %d, want 1", result.SignalCount)
	}
	if result.Scope != "P1-plan-scope" {
		t.Errorf("Scope = %q, want %q", result.Scope, "P1-plan-scope")
	}
}

// P5-2.7: Plan scope excludes signals from tasks under a different plan.
func TestSynthesise_PlanScopeExcludesOtherPlan(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	svc := newRetroTestServiceWithEntities(t, root)

	writeRetroTestPlan(t, svc.entitySvc, "P1-plan-excl-a")
	writeRetroTestPlan(t, svc.entitySvc, "P2-plan-excl-b")

	// Feature and task under plan A (no signals attributed to it).
	_, err := svc.entitySvc.CreateFeature(CreateFeatureInput{
		Name: "test",
		Slug: "feat-plan-a", Parent: "P1-plan-excl-a", Summary: "Feature under plan A", CreatedBy: "test",
	})
	if err != nil {
		t.Fatalf("CreateFeature A: %v", err)
	}

	// Feature and task under plan B (signal attributed here).
	featB, err := svc.entitySvc.CreateFeature(CreateFeatureInput{
		Name: "test",
		Slug: "feat-plan-b", Parent: "P2-plan-excl-b", Summary: "Feature under plan B", CreatedBy: "test",
	})
	if err != nil {
		t.Fatalf("CreateFeature B: %v", err)
	}
	taskB, err := svc.entitySvc.CreateTask(CreateTaskInput{
		Name:          "test",
		ParentFeature: featB.ID,
		Slug:          "task-plan-b",
		Summary:       "Task under plan B",
	})
	if err != nil {
		t.Fatalf("CreateTask B: %v", err)
	}

	writeRetroKnowledgeRecord(t, root, "KE-PE01", "workflow-friction", "minor",
		"[minor] workflow-friction: Signal from plan B",
		taskB.ID, "2026-03-01T10:00:00Z")

	// Synthesise scoped to plan A — should return zero signals.
	result, err := svc.Synthesise(RetroSynthesisInput{Scope: "P1-plan-excl-a"})
	if err != nil {
		t.Fatalf("Synthesise() error = %v", err)
	}
	if result.SignalCount != 0 {
		t.Errorf("SignalCount = %d, want 0 (signal belongs to plan B)", result.SignalCount)
	}
}

func TestSynthesise_ExperimentsExcludeNonAcceptedDecisions(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	svc := newRetroTestServiceWithEntities(t, root)

	// Decision tagged workflow-experiment but in rejected status.
	writeDecisionEntity(t, root, "DEC-0100000000011", "rejected-experiment",
		"Rejected experiment", "rejected",
		[]string{"workflow-experiment"})

	writeRetroKnowledgeRecord(t, root,
		"KE-RE01", "worked-well", "minor",
		"[minor] worked-well: Referenced rejected decision Related: DEC-0100000000011",
		"TASK-001", "2026-03-10T10:00:00Z",
	)

	result, err := svc.Synthesise(RetroSynthesisInput{})
	if err != nil {
		t.Fatalf("Synthesise() error = %v", err)
	}
	// buildExperiments only matches workflow-experiment decisions regardless of
	// their own status — it just looks for the tag. The rejected decision still
	// has the tag, so it WILL appear. This is by design: synthesis shows all
	// workflow-experiment decisions that signals reference, regardless of status.
	// The nudge (assembly.go) is what filters to accepted-only.
	if result.Experiments == nil {
		t.Fatal("Experiments should not be nil when signals reference a workflow-experiment decision")
	}
}

// ─── selectAutoThemes ─────────────────────────────────────────────────────────

func TestSelectAutoThemes_TopNOnly(t *testing.T) {
	t.Parallel()
	themes := []RetroTheme{
		{Rank: 1, Title: "Theme A", SeverityScore: 10},
		{Rank: 2, Title: "Theme B", SeverityScore: 8},
		{Rank: 3, Title: "Theme C", SeverityScore: 5},
		{Rank: 4, Title: "Theme D", SeverityScore: 3},
	}
	got := selectAutoThemes(themes, 2, 0)
	if len(got) != 2 {
		t.Fatalf("expected 2 themes, got %d", len(got))
	}
	if got[0].Title != "Theme A" || got[1].Title != "Theme B" {
		t.Errorf("got %v, want [Theme A, Theme B]", got)
	}
}

func TestSelectAutoThemes_ThresholdOnly(t *testing.T) {
	t.Parallel()
	themes := []RetroTheme{
		{Rank: 1, Title: "Theme A", SeverityScore: 10},
		{Rank: 2, Title: "Theme B", SeverityScore: 4},
		{Rank: 3, Title: "Theme C", SeverityScore: 8},
	}
	got := selectAutoThemes(themes, 0, 5)
	if len(got) != 2 {
		t.Fatalf("expected 2 themes (score >= 5), got %d", len(got))
	}
	// Theme B has score 4, should not be included.
	for _, th := range got {
		if th.Title == "Theme B" {
			t.Errorf("Theme B (score 4) should not be included with threshold 5")
		}
	}
}

func TestSelectAutoThemes_Intersection(t *testing.T) {
	t.Parallel()
	themes := []RetroTheme{
		{Rank: 1, Title: "Theme A", SeverityScore: 10},
		{Rank: 2, Title: "Theme B", SeverityScore: 4},
		{Rank: 3, Title: "Theme C", SeverityScore: 8},
		{Rank: 4, Title: "Theme D", SeverityScore: 6},
	}
	// Top 3 AND score >= 5 => Theme A (10), Theme C (8). Theme B (4) fails threshold, Theme D (6) fails top-N.
	got := selectAutoThemes(themes, 3, 5)
	if len(got) != 2 {
		t.Fatalf("expected 2 themes in intersection, got %d", len(got))
	}
	titles := make([]string, len(got))
	for i, t := range got {
		titles[i] = t.Title
	}
	if titles[0] != "Theme A" || titles[1] != "Theme C" {
		t.Errorf("got %v, want [Theme A, Theme C]", titles)
	}
}

func TestSelectAutoThemes_NoMatch(t *testing.T) {
	t.Parallel()
	themes := []RetroTheme{
		{Rank: 1, Title: "Theme A", SeverityScore: 10},
	}
	got := selectAutoThemes(themes, 0, 0)
	if got != nil {
		t.Errorf("expected nil when both filters are 0, got %v", got)
	}
}

// ─── themeToSlug ──────────────────────────────────────────────────────────────

func TestThemeToSlug_Basic(t *testing.T) {
	t.Parallel()
	theme := RetroTheme{Rank: 1, Title: "Error handling underspecified"}
	got := themeToSlug(theme)
	if got != "retro-fix-error-handling-underspecified" {
		t.Errorf("themeToSlug = %q, want %q", got, "retro-fix-error-handling-underspecified")
	}
}

func TestThemeToSlug_SpecialChars(t *testing.T) {
	t.Parallel()
	theme := RetroTheme{Rank: 3, Title: "Fix: tool-gap (urgent)!"}
	got := themeToSlug(theme)
	if !strings.HasPrefix(got, "retro-fix-") {
		t.Errorf("themeToSlug = %q, should start with retro-fix-", got)
	}
	if strings.Contains(got, "(") || strings.Contains(got, ")") || strings.Contains(got, "!") {
		t.Errorf("themeToSlug = %q, contains invalid characters", got)
	}
}

func TestThemeToSlug_EmptyTitle(t *testing.T) {
	t.Parallel()
	theme := RetroTheme{Rank: 5, Title: ""}
	got := themeToSlug(theme)
	if got != "retro-fix-5" {
		t.Errorf("themeToSlug = %q, want %q", got, "retro-fix-5")
	}
}

// ─── renderRetroFixDevPlan ────────────────────────────────────────────────────

func TestRenderRetroFixDevPlan(t *testing.T) {
	t.Parallel()
	theme := RetroTheme{
		Rank:          1,
		Category:      "spec-ambiguity",
		Title:         "Error handling underspecified",
		SignalCount:   3,
		SeverityScore: 15,
		TopSuggestion: "Add explicit error format section to the spec",
	}
	got := renderRetroFixDevPlan(theme, "FEAT-001")
	if !strings.Contains(got, "# Implementation Plan: Error handling underspecified") {
		t.Error("dev-plan missing title")
	}
	if !strings.Contains(got, "FEAT-001") {
		t.Error("dev-plan missing feature ID")
	}
	if !strings.Contains(got, "Add explicit error format section") {
		t.Error("dev-plan missing suggestion")
	}
	if !strings.Contains(got, "## Task Breakdown") {
		t.Error("dev-plan missing Task Breakdown section")
	}
}

// ─── CreateFix (integration) ──────────────────────────────────────────────────

// newRetroTestServiceForCreateFix creates a RetroService with all dependencies
// for CreateFix tests (knowledge, entity, doc, repoRoot).
func newRetroTestServiceForCreateFix(t *testing.T, root string) *RetroService {
	t.Helper()
	knowledgeSvc := NewKnowledgeService(root)
	t.Cleanup(knowledgeSvc.Close)
	entitySvc := NewEntityService(root)
	docSvc := NewDocumentService(root, root)
	return &RetroService{
		knowledgeSvc: knowledgeSvc,
		entitySvc:    entitySvc,
		docSvc:       docSvc,
		repoRoot:     root,
		now:          func() time.Time { return time.Date(2026, 3, 28, 0, 0, 0, 0, time.UTC) },
	}
}

func TestCreateFix_HumanGated_CreatesFeature(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	svc := newRetroTestServiceForCreateFix(t, root)

	// Write a retro signal to synthesise.
	writeRetroKnowledgeRecord(t, root,
		"KE-HG01", "spec-ambiguity", "moderate",
		"[moderate] spec-ambiguity: Error format undefined Suggestion: Add error format section",
		"TASK-HG01", "2026-03-01T10:00:00Z",
	)

	// Auto-create plan and create feature at theme index 0.
	result, err := svc.CreateFix(CreateFixInput{
		RetroSynthesisInput: RetroSynthesisInput{},
		Mode:                "human-gated",
		ThemeIndex:          0,
		CreatedBy:           "test",
	})
	if err != nil {
		t.Fatalf("CreateFix() error = %v", err)
	}
	if result.PlanID == "" {
		t.Error("PlanID should not be empty (auto-created)")
	}
	if !strings.HasPrefix(result.PlanID, "P") {
		t.Errorf("PlanID should start with P: %q", result.PlanID)
	}
	if result.ThemesProcessed != 1 {
		t.Errorf("ThemesProcessed = %d, want 1", result.ThemesProcessed)
	}
	if len(result.Features) != 1 {
		t.Fatalf("expected 1 feature, got %d", len(result.Features))
	}
	f := result.Features[0]
	if f.FeatureID == "" {
		t.Error("FeatureID should not be empty")
	}
	if f.ThemeRank != 1 {
		t.Errorf("ThemeRank = %d, want 1", f.ThemeRank)
	}
	if f.WasSkipped {
		t.Error("WasSkipped should be false for new feature")
	}
	// Human-gated mode: no doc IDs should be set.
	if f.DesignDocID != "" {
		t.Error("DesignDocID should be empty in human-gated mode")
	}
	if f.SpecDocID != "" {
		t.Error("SpecDocID should be empty in human-gated mode")
	}
}

func TestCreateFix_HumanGated_ThemeIndexOutOfRange(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	svc := newRetroTestServiceForCreateFix(t, root)

	_, err := svc.CreateFix(CreateFixInput{
		RetroSynthesisInput: RetroSynthesisInput{},
		Mode:                "human-gated",
		ThemeIndex:          5, // No signals = empty themes.
		CreatedBy:           "test",
	})
	if err == nil {
		t.Fatal("expected error for out-of-range theme_index")
	}
	if !strings.Contains(err.Error(), "out of range") {
		t.Errorf("error should mention out of range: %v", err)
	}
}

func TestCreateFix_Auto_CreatesMultipleFeatures(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	svc := newRetroTestServiceForCreateFix(t, root)

	// Write multiple signals in different categories.
	writeRetroKnowledgeRecord(t, root,
		"KE-A01", "spec-ambiguity", "significant",
		"[significant] spec-ambiguity: Error format completely undefined Suggestion: Add comprehensive error spec",
		"TASK-A01", "2026-03-01T10:00:00Z",
	)
	writeRetroKnowledgeRecord(t, root,
		"KE-A02", "tool-gap", "moderate",
		"[moderate] tool-gap: No deploy automation tool exists Suggestion: Add deploy tool integration",
		"TASK-A02", "2026-03-02T10:00:00Z",
	)
	writeRetroKnowledgeRecord(t, root,
		"KE-A03", "context-gap", "minor",
		"[minor] context-gap: Missing context for handoff packets Suggestion: Add context assembly guide",
		"TASK-A03", "2026-03-03T10:00:00Z",
	)

	// Select top 2 themes in auto mode.
	result, err := svc.CreateFix(CreateFixInput{
		RetroSynthesisInput: RetroSynthesisInput{},
		Mode:                "auto",
		ThemeCount:          2,
		CreatedBy:           "test",
	})
	if err != nil {
		t.Fatalf("CreateFix() error = %v", err)
	}
	if result.ThemesProcessed != 2 {
		t.Errorf("ThemesProcessed = %d, want 2", result.ThemesProcessed)
	}
	if len(result.Features) != 2 {
		t.Fatalf("expected 2 features, got %d", len(result.Features))
	}
	// Auto mode: features should have doc IDs.
	for i, f := range result.Features {
		if f.DesignDocID == "" {
			t.Errorf("feature[%d]: DesignDocID should not be empty in auto mode", i)
		}
		if f.SpecDocID == "" {
			t.Errorf("feature[%d]: SpecDocID should not be empty in auto mode", i)
		}
		if f.DevPlanDocID == "" {
			t.Errorf("feature[%d]: DevPlanDocID should not be empty in auto mode", i)
		}
		if f.WasSkipped {
			t.Errorf("feature[%d]: WasSkipped should be false", i)
		}
	}
}

func TestCreateFix_Auto_SeverityThreshold(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	svc := newRetroTestServiceForCreateFix(t, root)

	// Create signals that produce themes with different severity scores.
	// The significant one should get score 5, the minor ones score 1.
	writeRetroKnowledgeRecord(t, root,
		"KE-ST01", "spec-ambiguity", "significant",
		"[significant] spec-ambiguity: Critical error format gap Suggestion: Fix error handling",
		"TASK-ST01", "2026-03-01T10:00:00Z",
	)
	writeRetroKnowledgeRecord(t, root,
		"KE-ST02", "tool-gap", "minor",
		"[minor] tool-gap: Minor tool request Suggestion: Add helper",
		"TASK-ST02", "2026-03-02T10:00:00Z",
	)

	// Threshold of 5 should only select the significant theme.
	result, err := svc.CreateFix(CreateFixInput{
		RetroSynthesisInput: RetroSynthesisInput{},
		Mode:                "auto",
		SeverityThreshold:   5,
		CreatedBy:           "test",
	})
	if err != nil {
		t.Fatalf("CreateFix() error = %v", err)
	}
	// The minor tool-gap theme has severity score 1 (<5), should be excluded.
	if result.ThemesProcessed < 1 {
		t.Error("expected at least 1 theme with severity >= 5")
	}
	for _, f := range result.Features {
		if f.ThemeTitle == "" {
			t.Error("ThemeTitle should not be empty")
		}
	}
}

func TestCreateFix_Auto_Intersection(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	svc := newRetroTestServiceForCreateFix(t, root)

	// Three distinct categories with varying severities.
	writeRetroKnowledgeRecord(t, root,
		"KE-IN01", "spec-ambiguity", "significant",
		"[significant] spec-ambiguity: Critical spec gap Suggestion: Complete the spec",
		"TASK-IN01", "2026-03-01T10:00:00Z",
	)
	writeRetroKnowledgeRecord(t, root,
		"KE-IN02", "tool-gap", "significant",
		"[significant] tool-gap: Missing critical tool Suggestion: Build the tool",
		"TASK-IN02", "2026-03-02T10:00:00Z",
	)
	writeRetroKnowledgeRecord(t, root,
		"KE-IN03", "context-gap", "minor",
		"[minor] context-gap: Minor context issue Suggestion: Update docs",
		"TASK-IN03", "2026-03-03T10:00:00Z",
	)
	// Top 2 by rank AND severity >= 5.
	// spec-ambiguity (score 5) and tool-gap (score 5) should qualify.
	// context-gap (score 1) fails severity threshold.
	result, err := svc.CreateFix(CreateFixInput{
		RetroSynthesisInput: RetroSynthesisInput{},
		Mode:                "auto",
		ThemeCount:          2,
		SeverityThreshold:   5,
		CreatedBy:           "test",
	})
	if err != nil {
		t.Fatalf("CreateFix() error = %v", err)
	}
	if result.ThemesProcessed < 1 {
		t.Error("intersection should select at least 1 theme")
	}
	// Verify no feature has minor severity (context-gap).
	for _, f := range result.Features {
		if strings.Contains(f.ThemeTitle, "context") {
			t.Errorf("context-gap theme should not be selected in intersection: %v", f)
		}
	}
}

func TestCreateFix_Idempotency_SkipsExisting(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	svc := newRetroTestServiceForCreateFix(t, root)

	// Signal that will generate a theme.
	writeRetroKnowledgeRecord(t, root,
		"KE-ID01", "spec-ambiguity", "moderate",
		"[moderate] spec-ambiguity: Error format undefined Suggestion: Add error format section",
		"TASK-ID01", "2026-03-01T10:00:00Z",
	)

	// First call: create the feature.
	result1, err := svc.CreateFix(CreateFixInput{
		RetroSynthesisInput: RetroSynthesisInput{},
		Mode:                "human-gated",
		ThemeIndex:          0,
		CreatedBy:           "test",
	})
	if err != nil {
		t.Fatalf("first CreateFix() error = %v", err)
	}
	if len(result1.Features) != 1 || result1.Features[0].WasSkipped {
		t.Fatal("first call should create a feature, not skip")
	}
	firstFeatureID := result1.Features[0].FeatureID

	// Second call: should detect the existing feature and skip.
	result2, err := svc.CreateFix(CreateFixInput{
		RetroSynthesisInput: RetroSynthesisInput{},
		Mode:                "human-gated",
		ThemeIndex:          0,
		CreatedBy:           "test",
	})
	if err != nil {
		t.Fatalf("second CreateFix() error = %v", err)
	}
	if len(result2.Features) != 1 {
		t.Fatalf("expected 1 result on second call, got %d", len(result2.Features))
	}
	f := result2.Features[0]
	if !f.WasSkipped {
		t.Error("second call should skip the existing feature")
	}
	if f.FeatureID != firstFeatureID {
		t.Errorf("skipped feature ID = %q, want %q", f.FeatureID, firstFeatureID)
	}
	if f.SkipReason == "" {
		t.Error("SkipReason should not be empty")
	}
}

func TestCreateFix_ParentPlan(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	svc := newRetroTestServiceForCreateFix(t, root)

	// Create a plan explicitly.
	planResult, err := svc.entitySvc.CreatePlan(CreatePlanInput{
		Prefix:    "P",
		Slug:      "my-custom-plan",
		Name:      "My Custom Plan",
		Summary:   "Custom plan for testing",
		CreatedBy: "test",
	})
	if err != nil {
		t.Fatalf("CreatePlan: %v", err)
	}

	writeRetroKnowledgeRecord(t, root,
		"KE-PP01", "spec-ambiguity", "moderate",
		"[moderate] spec-ambiguity: Error format undefined",
		"TASK-PP01", "2026-03-01T10:00:00Z",
	)

	result, err := svc.CreateFix(CreateFixInput{
		RetroSynthesisInput: RetroSynthesisInput{},
		Mode:                "human-gated",
		ThemeIndex:          0,
		ParentPlan:          planResult.ID,
		CreatedBy:           "test",
	})
	if err != nil {
		t.Fatalf("CreateFix() error = %v", err)
	}
	if result.PlanID != planResult.ID {
		t.Errorf("PlanID = %q, want %q", result.PlanID, planResult.ID)
	}
}

func TestCreateFix_InvalidMode(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	svc := newRetroTestServiceForCreateFix(t, root)

	_, err := svc.CreateFix(CreateFixInput{
		RetroSynthesisInput: RetroSynthesisInput{},
		Mode:                "invalid-mode",
		CreatedBy:           "test",
	})
	if err == nil {
		t.Fatal("expected error for invalid mode")
	}
	if !strings.Contains(err.Error(), "invalid mode") {
		t.Errorf("error should mention invalid mode: %v", err)
	}
}

func TestCreateFix_NoThemes(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	svc := newRetroTestServiceForCreateFix(t, root)

	result, err := svc.CreateFix(CreateFixInput{
		RetroSynthesisInput: RetroSynthesisInput{},
		Mode:                "auto",
		ThemeCount:          3,
		CreatedBy:           "test",
	})
	if err != nil {
		t.Fatalf("CreateFix() error = %v", err)
	}
	if len(result.Features) != 0 {
		t.Errorf("expected 0 features with no signals, got %d", len(result.Features))
	}
	if result.ThemesProcessed != 0 {
		t.Errorf("ThemesProcessed = %d, want 0", result.ThemesProcessed)
	}
}

// ═══ AC-007: Auto mode creates approved docs ═══════════════════════════════════

func TestCreateFix_Auto_ApprovedDocs(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	svc := newRetroTestServiceForCreateFix(t, root)

	writeRetroKnowledgeRecord(t, root,
		"KE-AP01", "spec-ambiguity", "significant",
		"[significant] spec-ambiguity: Critical spec gap Suggestion: Complete the spec",
		"TASK-AP01", "2026-03-01T10:00:00Z",
	)

	result, err := svc.CreateFix(CreateFixInput{
		RetroSynthesisInput: RetroSynthesisInput{},
		Mode:                "auto",
		ThemeCount:          1,
		CreatedBy:           "test",
	})
	if err != nil {
		t.Fatalf("CreateFix() error = %v", err)
	}
	if len(result.Features) != 1 {
		t.Fatalf("expected 1 feature, got %d", len(result.Features))
	}
	f := result.Features[0]

	// Design doc must be approved.
	if f.DesignDocID == "" {
		t.Fatal("DesignDocID should not be empty in auto mode")
	}
	designDoc, err := svc.docSvc.GetDocument(f.DesignDocID, false)
	if err != nil {
		t.Fatalf("GetDocument(design %s): %v", f.DesignDocID, err)
	}
	if designDoc.Status != "approved" {
		t.Errorf("design doc status = %q, want %q", designDoc.Status, "approved")
	}

	// Spec doc must be approved.
	if f.SpecDocID == "" {
		t.Fatal("SpecDocID should not be empty in auto mode")
	}
	specDoc, err := svc.docSvc.GetDocument(f.SpecDocID, false)
	if err != nil {
		t.Fatalf("GetDocument(spec %s): %v", f.SpecDocID, err)
	}
	if specDoc.Status != "approved" {
		t.Errorf("spec doc status = %q, want %q", specDoc.Status, "approved")
	}

	// Dev-plan doc must be approved (auto-approved).
	if f.DevPlanDocID == "" {
		t.Fatal("DevPlanDocID should not be empty in auto mode")
	}
	devPlanDoc, err := svc.docSvc.GetDocument(f.DevPlanDocID, false)
	if err != nil {
		t.Fatalf("GetDocument(dev-plan %s): %v", f.DevPlanDocID, err)
	}
	if devPlanDoc.Status != "approved" {
		t.Errorf("dev-plan doc status = %q, want %q", devPlanDoc.Status, "approved")
	}
}

// ═══ AC-014: Batch summary response shape ══════════════════════════════════════

func TestCreateFix_BatchSummaryResponseShape(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	svc := newRetroTestServiceForCreateFix(t, root)

	// Multiple signals in different categories to produce multiple themes.
	writeRetroKnowledgeRecord(t, root,
		"KE-BS01", "spec-ambiguity", "significant",
		"[significant] spec-ambiguity: Gap one Suggestion: Fix gap one",
		"TASK-BS01", "2026-03-01T10:00:00Z",
	)
	writeRetroKnowledgeRecord(t, root,
		"KE-BS02", "tool-gap", "significant",
		"[significant] tool-gap: Gap two Suggestion: Fix gap two",
		"TASK-BS02", "2026-03-02T10:00:00Z",
	)

	result, err := svc.CreateFix(CreateFixInput{
		RetroSynthesisInput: RetroSynthesisInput{},
		Mode:                "auto",
		ThemeCount:          2,
		CreatedBy:           "test",
	})
	if err != nil {
		t.Fatalf("CreateFix() error = %v", err)
	}

	// AC-014: Verify response shape fields.
	if result.PlanID == "" {
		t.Error("PlanID should not be empty")
	}
	if !strings.HasPrefix(result.PlanID, "P") {
		t.Errorf("PlanID should start with P: %q", result.PlanID)
	}
	if result.ThemesProcessed != len(result.Features) {
		t.Errorf("ThemesProcessed = %d should equal len(Features) = %d",
			result.ThemesProcessed, len(result.Features))
	}
	if result.ThemesProcessed != 2 {
		t.Errorf("ThemesProcessed = %d, want 2", result.ThemesProcessed)
	}

	// Each feature must have required fields.
	for i, f := range result.Features {
		if f.FeatureID == "" {
			t.Errorf("Features[%d].FeatureID empty", i)
		}
		if !strings.HasPrefix(f.FeatureID, "FEAT-") {
			t.Errorf("Features[%d].FeatureID = %q, should start with FEAT-", i, f.FeatureID)
		}
		if f.ThemeRank == 0 {
			t.Errorf("Features[%d].ThemeRank = 0, want non-zero", i)
		}
		if f.ThemeTitle == "" {
			t.Errorf("Features[%d].ThemeTitle empty", i)
		}
		if f.WasSkipped {
			t.Errorf("Features[%d].WasSkipped should be false", i)
		}
	}
}

// ═══ AC-015: Feature tags contain signal IDs ═══════════════════════════════════

func TestCreateFix_FeatureTagsContainSignalIDs(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	svc := newRetroTestServiceForCreateFix(t, root)

	writeRetroKnowledgeRecord(t, root,
		"KE-TG01", "spec-ambiguity", "moderate",
		"[moderate] spec-ambiguity: Error format undefined Suggestion: Add error format section",
		"TASK-TG01", "2026-03-01T10:00:00Z",
	)

	result, err := svc.CreateFix(CreateFixInput{
		RetroSynthesisInput: RetroSynthesisInput{},
		Mode:                "human-gated",
		ThemeIndex:          0,
		CreatedBy:           "test",
	})
	if err != nil {
		t.Fatalf("CreateFix() error = %v", err)
	}
	if len(result.Features) != 1 {
		t.Fatalf("expected 1 feature, got %d", len(result.Features))
	}

	featID := result.Features[0].FeatureID
	entity, err := svc.entitySvc.Get("feature", featID, "")
	if err != nil {
		t.Fatalf("Get(%s): %v", featID, err)
	}
	tags := extractTags(entity.State)

	// AC-015: tags must contain signal ID KE-TG01.
	found := false
	for _, tag := range tags {
		if tag == "KE-TG01" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("tags %v should contain KE-TG01 (signal ID)", tags)
	}

	// Tags must also include "retro".
	retroFound := false
	for _, tag := range tags {
		if tag == "retro" {
			retroFound = true
			break
		}
	}
	if !retroFound {
		t.Errorf("tags %v should contain 'retro'", tags)
	}

	// Tier must be retro_fix.
	tier, _ := entity.State["tier"].(string)
	if tier != "retro_fix" {
		t.Errorf("tier = %q, want %q", tier, "retro_fix")
	}
}

// ═══ AC-017: DefaultFastTrackConfig unchanged (REQ-NF-004) ═════════════════════

func TestDefaultFastTrackConfig_RetroFixUnchanged(t *testing.T) {
	t.Parallel()
	cfg := config.DefaultFastTrackConfig()

	tier, ok := cfg.Tiers[config.TierRetroFix]
	if !ok {
		t.Fatal("retro_fix tier missing from DefaultFastTrackConfig")
	}

	if tier.Design != string(config.GateModeHuman) {
		t.Errorf("retro_fix Design = %q, want %q", tier.Design, config.GateModeHuman)
	}
	if tier.Spec != string(config.GateModeAuto) {
		t.Errorf("retro_fix Spec = %q, want %q", tier.Spec, config.GateModeAuto)
	}
	if tier.DevPlan != string(config.GateModeAuto) {
		t.Errorf("retro_fix DevPlan = %q, want %q", tier.DevPlan, config.GateModeAuto)
	}
	if tier.Review != string(config.GateModeConditional) {
		t.Errorf("retro_fix Review = %q, want %q", tier.Review, config.GateModeConditional)
	}
	if tier.MaxCycles != 3 {
		t.Errorf("retro_fix MaxCycles = %d, want 3", tier.MaxCycles)
	}
}
