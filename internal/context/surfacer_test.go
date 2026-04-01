package context

import (
	"testing"
	"time"

	"github.com/sambeau/kanbanzai/internal/knowledge"
)

func fixedNow() time.Time {
	return time.Date(2025, 6, 15, 12, 0, 0, 0, time.UTC)
}

func makeTestEntry(id, topic, content, scope, status string, confidence float64, tags []string, confirmedDaysAgo int) map[string]any {
	now := fixedNow()
	entry := map[string]any{
		"id":         id,
		"topic":      topic,
		"content":    content,
		"scope":      scope,
		"status":     status,
		"confidence": confidence,
		"created":    now.Add(-time.Duration(confirmedDaysAgo) * 24 * time.Hour).Format(time.RFC3339),
	}
	if confirmedDaysAgo >= 0 && status == "confirmed" {
		entry["last_confirmed"] = now.Add(-time.Duration(confirmedDaysAgo) * 24 * time.Hour).Format(time.RFC3339)
	}
	if len(tags) > 0 {
		anyTags := make([]any, len(tags))
		for i, t := range tags {
			anyTags[i] = t
		}
		entry["tags"] = anyTags
	}
	return entry
}

func staticLoader(entries []map[string]any) EntryLoader {
	return func() ([]map[string]any, error) {
		return entries, nil
	}
}

func TestSurfacer_FullPipeline(t *testing.T) {
	t.Parallel()

	entries := []map[string]any{
		makeTestEntry("KE-001", "error-wrapping", "Always wrap errors with context BECAUSE it aids debugging", "internal/service/", "confirmed", 0.9, nil, 1),
		makeTestEntry("KE-002", "table-tests", "Always use table-driven tests BECAUSE they make adding cases trivial", "project", "confirmed", 0.8, nil, 5),
		makeTestEntry("KE-003", "yaml-tags", "Always add yaml struct tags BECAUSE the serializer requires them", "internal/storage/", "confirmed", 0.7, []string{"go"}, 10),
	}

	s := NewSurfacer(staticLoader(entries), nil, fixedNow)
	result, err := s.Surface(SurfaceInput{
		FilePaths: []string{"internal/service/handler.go"},
		RoleTags:  []string{"go"},
	})
	if err != nil {
		t.Fatalf("Surface() error = %v", err)
	}

	if len(result) != 3 {
		t.Fatalf("expected 3 surfaced entries, got %d", len(result))
	}

	// Verify ascending score order (highest last).
	for i := 1; i < len(result); i++ {
		if result[i].Score < result[i-1].Score {
			t.Errorf("entry %d score (%.4f) < entry %d score (%.4f); expected ascending", i, result[i].Score, i-1, result[i-1].Score)
		}
	}
}

func TestSurfacer_ZeroMatches(t *testing.T) {
	t.Parallel()

	entries := []map[string]any{
		makeTestEntry("KE-001", "topic-a", "content a", "internal/storage/", "confirmed", 0.9, nil, 1),
	}

	s := NewSurfacer(staticLoader(entries), nil, fixedNow)
	result, err := s.Surface(SurfaceInput{
		FilePaths: []string{"cmd/main.go"},
		RoleTags:  []string{"frontend"},
	})
	if err != nil {
		t.Fatalf("Surface() error = %v", err)
	}
	if result != nil {
		t.Fatalf("expected nil result for zero matches, got %d entries", len(result))
	}
}

func TestSurfacer_EmptyKnowledgeBase(t *testing.T) {
	t.Parallel()

	s := NewSurfacer(staticLoader(nil), nil, fixedNow)
	result, err := s.Surface(SurfaceInput{
		FilePaths: []string{"internal/service/handler.go"},
	})
	if err != nil {
		t.Fatalf("Surface() error = %v", err)
	}
	if result != nil {
		t.Fatalf("expected nil result for empty knowledge base, got %d entries", len(result))
	}
}

func TestSurfacer_LoaderError(t *testing.T) {
	t.Parallel()

	loader := func() ([]map[string]any, error) {
		return nil, &testError{"knowledge base unavailable"}
	}
	s := NewSurfacer(loader, nil, fixedNow)
	result, err := s.Surface(SurfaceInput{
		FilePaths: []string{"internal/service/handler.go"},
	})
	if err != nil {
		t.Fatalf("expected nil error on loader failure (graceful degradation), got %v", err)
	}
	if result != nil {
		t.Fatalf("expected nil result on loader failure, got %d entries", len(result))
	}
}

type testError struct{ msg string }

func (e *testError) Error() string { return e.msg }

func TestSurfacer_CapAt10(t *testing.T) {
	t.Parallel()

	var entries []map[string]any
	for i := 0; i < 15; i++ {
		id := "KE-" + string(rune('A'+i))
		entries = append(entries, makeTestEntry(
			id, "topic-"+id, "Always do thing "+id,
			"project", "confirmed", 0.5+float64(i)*0.02, nil, i,
		))
	}

	s := NewSurfacer(staticLoader(entries), nil, fixedNow)
	result, err := s.Surface(SurfaceInput{})
	if err != nil {
		t.Fatalf("Surface() error = %v", err)
	}
	if len(result) != 10 {
		t.Fatalf("expected 10 surfaced entries (cap), got %d", len(result))
	}
}

func TestSurfacer_Deterministic(t *testing.T) {
	t.Parallel()

	entries := []map[string]any{
		makeTestEntry("KE-001", "topic-a", "content a", "project", "confirmed", 0.9, nil, 1),
		makeTestEntry("KE-002", "topic-b", "content b", "project", "confirmed", 0.8, nil, 2),
		makeTestEntry("KE-003", "topic-c", "content c", "project", "confirmed", 0.7, nil, 3),
	}

	s := NewSurfacer(staticLoader(entries), nil, fixedNow)
	input := SurfaceInput{FilePaths: []string{"internal/service/handler.go"}}

	result1, _ := s.Surface(input)
	result2, _ := s.Surface(input)

	if len(result1) != len(result2) {
		t.Fatalf("different lengths: %d vs %d", len(result1), len(result2))
	}
	for i := range result1 {
		if result1[i].ID != result2[i].ID {
			t.Errorf("entry %d: ID %s vs %s", i, result1[i].ID, result2[i].ID)
		}
		if result1[i].Score != result2[i].Score {
			t.Errorf("entry %d: Score %f vs %f", i, result1[i].Score, result2[i].Score)
		}
	}
}

func TestSurfacer_CapTrackerIntegration(t *testing.T) {
	t.Parallel()

	cacheDir := t.TempDir()
	tracker := knowledge.NewCapTracker(cacheDir)

	// Create 12 project-scoped entries so the cap is hit.
	var entries []map[string]any
	for i := 0; i < 12; i++ {
		id := "KE-" + string(rune('A'+i))
		entries = append(entries, makeTestEntry(
			id, "topic-"+id, "Always do "+id,
			"project", "confirmed", 0.5+float64(i)*0.01, nil, i,
		))
	}

	s := NewSurfacer(staticLoader(entries), tracker, fixedNow)
	input := SurfaceInput{FilePaths: []string{"internal/service/handler.go"}}

	// 3 consecutive cap-hit assemblies.
	for i := 0; i < 3; i++ {
		_, err := s.Surface(input)
		if err != nil {
			t.Fatalf("Surface() call %d error = %v", i, err)
		}
	}

	scopes := tracker.ScopesNeedingCompaction()
	if len(scopes) == 0 {
		t.Fatal("expected cap tracker to report scope needing compaction after 3 cap-hits")
	}
}

func TestSurfacer_OutputContainsExpectedFields(t *testing.T) {
	t.Parallel()

	entries := []map[string]any{
		makeTestEntry("KE-042", "yaml-marshal", "Always validate YAML output BECAUSE silent corruption is costly", "project", "confirmed", 0.85, nil, 2),
	}

	s := NewSurfacer(staticLoader(entries), nil, fixedNow)
	result, err := s.Surface(SurfaceInput{})
	if err != nil {
		t.Fatalf("Surface() error = %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(result))
	}

	e := result[0]
	if e.ID != "KE-042" {
		t.Errorf("ID = %q, want KE-042", e.ID)
	}
	if e.Topic != "yaml-marshal" {
		t.Errorf("Topic = %q, want yaml-marshal", e.Topic)
	}
	if e.Content != "Always validate YAML output BECAUSE silent corruption is costly" {
		t.Errorf("Content = %q, unexpected", e.Content)
	}
	if e.Score <= 0 {
		t.Errorf("Score = %f, expected > 0", e.Score)
	}
}

func TestSurfacer_HighestScoreLast(t *testing.T) {
	t.Parallel()

	entries := []map[string]any{
		makeTestEntry("KE-001", "low-conf", "content a", "project", "confirmed", 0.3, nil, 30),
		makeTestEntry("KE-002", "high-conf-recent", "content b", "project", "confirmed", 0.95, nil, 0),
		makeTestEntry("KE-003", "mid-conf", "content c", "project", "confirmed", 0.6, nil, 10),
	}

	s := NewSurfacer(staticLoader(entries), nil, fixedNow)
	result, err := s.Surface(SurfaceInput{})
	if err != nil {
		t.Fatalf("Surface() error = %v", err)
	}
	if len(result) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(result))
	}

	last := result[len(result)-1]
	if last.ID != "KE-002" {
		t.Errorf("highest score should be last; got ID %s (score %.4f)", last.ID, last.Score)
	}
	first := result[0]
	if first.Score > last.Score {
		t.Errorf("first score (%.4f) > last score (%.4f); expected ascending order", first.Score, last.Score)
	}
}

func TestDeriveScopeForTracking(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		filePaths []string
		want      string
	}{
		{"no paths", nil, ""},
		{"single file", []string{"internal/service/handler.go"}, "internal/service/"},
		{"same dir", []string{"internal/service/a.go", "internal/service/b.go"}, "internal/service/"},
		{"different dirs", []string{"internal/service/a.go", "cmd/main.go"}, "mixed"},
		{"nested same parent", []string{"internal/service/sub/a.go", "internal/service/b.go"}, "internal/service/"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := deriveScopeForTracking(tt.filePaths)
			if got != tt.want {
				t.Errorf("deriveScopeForTracking(%v) = %q, want %q", tt.filePaths, got, tt.want)
			}
		})
	}
}
