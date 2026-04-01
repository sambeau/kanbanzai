package knowledge

import (
	"testing"
	"time"
)

var fixedTime = time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC).Format(time.RFC3339)

func makeEntry(id, topic, content, scope, status string, confidence float64, tags []string) map[string]any {
	entry := map[string]any{
		"id":         id,
		"topic":      topic,
		"content":    content,
		"scope":      scope,
		"status":     status,
		"confidence": confidence,
		"created":    fixedTime,
	}
	if tags != nil {
		anyTags := make([]any, len(tags))
		for i, t := range tags {
			anyTags[i] = t
		}
		entry["tags"] = anyTags
	}
	return entry
}

func TestMatchEntries_FilePathMatch(t *testing.T) {
	t.Parallel()

	entry := makeEntry("KE-01", "storage-topic", "storage info", "internal/storage/", "confirmed", 0.8, nil)
	entries := []map[string]any{entry}

	tests := []struct {
		name      string
		filePaths []string
		wantIDs   []string
	}{
		{
			name:      "matching file path",
			filePaths: []string{"internal/storage/yaml.go"},
			wantIDs:   []string{"KE-01"},
		},
		{
			name:      "non-matching file path",
			filePaths: []string{"internal/mcp/handoff.go"},
			wantIDs:   []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := MatchEntries(entries, MatchInput{FilePaths: tt.filePaths})
			assertIDs(t, got, tt.wantIDs)
		})
	}
}

func TestMatchEntries_TagMatch(t *testing.T) {
	t.Parallel()

	entry := makeEntry("KE-02", "sec-topic", "security info", "internal/auth/", "confirmed", 0.9, []string{"security"})
	entries := []map[string]any{entry}

	tests := []struct {
		name     string
		roleTags []string
		wantIDs  []string
	}{
		{
			name:     "matching role tag",
			roleTags: []string{"security", "go"},
			wantIDs:  []string{"KE-02"},
		},
		{
			name:     "case insensitive match",
			roleTags: []string{"SECURITY"},
			wantIDs:  []string{"KE-02"},
		},
		{
			name:     "non-matching role tag",
			roleTags: []string{"testing"},
			wantIDs:  []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := MatchEntries(entries, MatchInput{RoleTags: tt.roleTags})
			assertIDs(t, got, tt.wantIDs)
		})
	}
}

func TestMatchEntries_AlwaysTag(t *testing.T) {
	t.Parallel()

	entry := makeEntry("KE-03", "global-topic", "always relevant", "internal/foo/", "confirmed", 0.7, []string{"always"})
	entries := []map[string]any{entry}

	got := MatchEntries(entries, MatchInput{})
	assertIDs(t, got, []string{"KE-03"})
}

func TestMatchEntries_ProjectScope(t *testing.T) {
	t.Parallel()

	entry := makeEntry("KE-04", "proj-topic", "project-wide info", "project", "confirmed", 0.8, nil)
	entries := []map[string]any{entry}

	got := MatchEntries(entries, MatchInput{})
	assertIDs(t, got, []string{"KE-04"})
}

func TestMatchEntries_Dedup(t *testing.T) {
	t.Parallel()

	entry := makeEntry("KE-05", "multi-match", "matches everything", "project", "confirmed", 0.9, []string{"always", "go"})
	entries := []map[string]any{entry}

	got := MatchEntries(entries, MatchInput{
		FilePaths: []string{"project/something.go"},
		RoleTags:  []string{"go"},
	})
	assertIDs(t, got, []string{"KE-05"})
}

func TestMatchEntries_RetiredExcluded(t *testing.T) {
	t.Parallel()

	entries := []map[string]any{
		makeEntry("KE-06", "retired-topic", "old info", "project", "retired", 0.5, nil),
		makeEntry("KE-07", "contributed-topic", "new info", "project", "contributed", 0.5, nil),
		makeEntry("KE-08", "confirmed-topic", "good info", "project", "confirmed", 0.8, nil),
		makeEntry("KE-09", "disputed-topic", "disputed info", "project", "disputed", 0.3, nil),
	}

	got := MatchEntries(entries, MatchInput{})
	assertIDs(t, got, []string{"KE-07", "KE-08", "KE-09"})
}

func TestMatchEntries_EmptyFilePaths(t *testing.T) {
	t.Parallel()

	entries := []map[string]any{
		makeEntry("KE-10", "scoped", "scoped info", "internal/storage/", "confirmed", 0.8, nil),
		makeEntry("KE-11", "tagged", "tagged info", "some/scope/", "confirmed", 0.8, []string{"go"}),
		makeEntry("KE-12", "always-entry", "always info", "some/scope/", "confirmed", 0.8, []string{"always"}),
	}

	got := MatchEntries(entries, MatchInput{RoleTags: []string{"go"}})
	assertIDs(t, got, []string{"KE-11", "KE-12"})
}

func TestMatchEntries_EmptyRoleTags(t *testing.T) {
	t.Parallel()

	entries := []map[string]any{
		makeEntry("KE-13", "tagged-only", "tagged info", "other/", "confirmed", 0.8, []string{"security"}),
		makeEntry("KE-14", "path-match", "path info", "internal/storage/", "confirmed", 0.8, nil),
		makeEntry("KE-15", "always-entry", "always info", "other/", "confirmed", 0.8, []string{"always"}),
	}

	got := MatchEntries(entries, MatchInput{FilePaths: []string{"internal/storage/yaml.go"}})
	assertIDs(t, got, []string{"KE-14", "KE-15"})
}

func TestMatchEntries_NoMatches(t *testing.T) {
	t.Parallel()

	entries := []map[string]any{
		makeEntry("KE-16", "scoped", "info", "internal/storage/", "confirmed", 0.8, []string{"go"}),
	}

	got := MatchEntries(entries, MatchInput{
		FilePaths: []string{"cmd/main.go"},
		RoleTags:  []string{"python"},
	})

	if got == nil {
		t.Fatal("expected empty slice, got nil")
	}
	if len(got) != 0 {
		t.Errorf("expected 0 matches, got %d", len(got))
	}
}

func TestMatchEntries_Deterministic(t *testing.T) {
	t.Parallel()

	entries := []map[string]any{
		makeEntry("KE-20", "topic-c", "c info", "project", "confirmed", 0.8, nil),
		makeEntry("KE-18", "topic-a", "a info", "project", "confirmed", 0.9, nil),
		makeEntry("KE-19", "topic-b", "b info", "project", "confirmed", 0.7, nil),
	}

	input := MatchInput{FilePaths: []string{"some/file.go"}}

	first := MatchEntries(entries, input)
	second := MatchEntries(entries, input)

	if len(first) != len(second) {
		t.Fatalf("different lengths: %d vs %d", len(first), len(second))
	}
	for i := range first {
		if first[i].ID != second[i].ID {
			t.Errorf("index %d: %s != %s", i, first[i].ID, second[i].ID)
		}
	}

	for i := 1; i < len(first); i++ {
		if first[i].ID < first[i-1].ID {
			t.Errorf("results not sorted: %s before %s", first[i-1].ID, first[i].ID)
		}
	}
}

func assertIDs(t *testing.T, got []MatchedEntry, wantIDs []string) {
	t.Helper()
	if len(got) != len(wantIDs) {
		ids := make([]string, len(got))
		for i, e := range got {
			ids[i] = e.ID
		}
		t.Fatalf("got %d entries %v, want %d %v", len(got), ids, len(wantIDs), wantIDs)
	}
	for i, want := range wantIDs {
		if got[i].ID != want {
			t.Errorf("index %d: got ID %s, want %s", i, got[i].ID, want)
		}
	}
}
