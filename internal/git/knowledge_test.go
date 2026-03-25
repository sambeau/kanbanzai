package git

import (
	"testing"
	"time"
)

func TestExtractAnchors_StringSlice(t *testing.T) {
	t.Parallel()

	fields := map[string]any{
		"git_anchors": []string{
			"internal/api/handler.go",
			"internal/api/routes.go",
		},
	}

	anchors := ExtractAnchors(fields)

	if len(anchors) != 2 {
		t.Fatalf("ExtractAnchors() len = %d, want 2", len(anchors))
	}
	if anchors[0].Path != "internal/api/handler.go" {
		t.Errorf("ExtractAnchors()[0].Path = %q, want %q", anchors[0].Path, "internal/api/handler.go")
	}
	if anchors[1].Path != "internal/api/routes.go" {
		t.Errorf("ExtractAnchors()[1].Path = %q, want %q", anchors[1].Path, "internal/api/routes.go")
	}
}

func TestExtractAnchors_AnySlice(t *testing.T) {
	t.Parallel()

	// YAML unmarshals to []any, not []string
	fields := map[string]any{
		"git_anchors": []any{
			"internal/api/handler.go",
			"internal/api/routes.go",
		},
	}

	anchors := ExtractAnchors(fields)

	if len(anchors) != 2 {
		t.Fatalf("ExtractAnchors() len = %d, want 2", len(anchors))
	}
	if anchors[0].Path != "internal/api/handler.go" {
		t.Errorf("ExtractAnchors()[0].Path = %q", anchors[0].Path)
	}
}

func TestExtractAnchors_NoField(t *testing.T) {
	t.Parallel()

	fields := map[string]any{
		"id":    "KE-01JTEST",
		"topic": "test-topic",
	}

	anchors := ExtractAnchors(fields)

	if anchors != nil {
		t.Errorf("ExtractAnchors() = %v, want nil", anchors)
	}
}

func TestExtractAnchors_NilField(t *testing.T) {
	t.Parallel()

	fields := map[string]any{
		"git_anchors": nil,
	}

	anchors := ExtractAnchors(fields)

	if anchors != nil {
		t.Errorf("ExtractAnchors() = %v, want nil", anchors)
	}
}

func TestExtractAnchors_NilFields(t *testing.T) {
	t.Parallel()

	anchors := ExtractAnchors(nil)

	if anchors != nil {
		t.Errorf("ExtractAnchors(nil) = %v, want nil", anchors)
	}
}

func TestExtractAnchors_EmptySlice(t *testing.T) {
	t.Parallel()

	fields := map[string]any{
		"git_anchors": []string{},
	}

	anchors := ExtractAnchors(fields)

	if anchors != nil {
		t.Errorf("ExtractAnchors() = %v, want nil", anchors)
	}
}

func TestExtractAnchors_SkipsEmptyPaths(t *testing.T) {
	t.Parallel()

	fields := map[string]any{
		"git_anchors": []any{
			"valid/path.go",
			"",
			"another/path.go",
		},
	}

	anchors := ExtractAnchors(fields)

	if len(anchors) != 2 {
		t.Fatalf("ExtractAnchors() len = %d, want 2", len(anchors))
	}
	if anchors[0].Path != "valid/path.go" {
		t.Errorf("ExtractAnchors()[0].Path = %q", anchors[0].Path)
	}
	if anchors[1].Path != "another/path.go" {
		t.Errorf("ExtractAnchors()[1].Path = %q", anchors[1].Path)
	}
}

func TestExtractAnchors_WrongType(t *testing.T) {
	t.Parallel()

	fields := map[string]any{
		"git_anchors": "not-a-slice",
	}

	anchors := ExtractAnchors(fields)

	if anchors != nil {
		t.Errorf("ExtractAnchors() with wrong type = %v, want nil", anchors)
	}
}

func TestSetLastConfirmed(t *testing.T) {
	t.Parallel()

	fields := make(map[string]any)
	ts := time.Date(2024, 6, 15, 10, 30, 0, 0, time.UTC)

	SetLastConfirmed(fields, ts)

	got, ok := fields["last_confirmed"].(string)
	if !ok {
		t.Fatalf("last_confirmed is not a string: %T", fields["last_confirmed"])
	}

	want := "2024-06-15T10:30:00Z"
	if got != want {
		t.Errorf("SetLastConfirmed() = %q, want %q", got, want)
	}
}

func TestSetLastConfirmed_NilFields(t *testing.T) {
	t.Parallel()

	// Should not panic
	SetLastConfirmed(nil, time.Now())
}

func TestGetLastConfirmed(t *testing.T) {
	t.Parallel()

	fields := map[string]any{
		"last_confirmed": "2024-06-15T10:30:00Z",
	}

	got := GetLastConfirmed(fields)
	want := time.Date(2024, 6, 15, 10, 30, 0, 0, time.UTC)

	if !got.Equal(want) {
		t.Errorf("GetLastConfirmed() = %v, want %v", got, want)
	}
}

func TestGetLastConfirmed_NoField(t *testing.T) {
	t.Parallel()

	fields := map[string]any{
		"id": "KE-01JTEST",
	}

	got := GetLastConfirmed(fields)

	if !got.IsZero() {
		t.Errorf("GetLastConfirmed() = %v, want zero", got)
	}
}

func TestGetLastConfirmed_InvalidFormat(t *testing.T) {
	t.Parallel()

	fields := map[string]any{
		"last_confirmed": "not-a-timestamp",
	}

	got := GetLastConfirmed(fields)

	if !got.IsZero() {
		t.Errorf("GetLastConfirmed() with invalid format = %v, want zero", got)
	}
}

func TestGetLastConfirmed_NilFields(t *testing.T) {
	t.Parallel()

	got := GetLastConfirmed(nil)

	if !got.IsZero() {
		t.Errorf("GetLastConfirmed(nil) = %v, want zero", got)
	}
}

func TestSetLastUsed(t *testing.T) {
	t.Parallel()

	fields := make(map[string]any)
	ts := time.Date(2024, 7, 20, 14, 45, 0, 0, time.UTC)

	SetLastUsed(fields, ts)

	got, ok := fields["last_used"].(string)
	if !ok {
		t.Fatalf("last_used is not a string: %T", fields["last_used"])
	}

	want := "2024-07-20T14:45:00Z"
	if got != want {
		t.Errorf("SetLastUsed() = %q, want %q", got, want)
	}
}

func TestGetLastUsed(t *testing.T) {
	t.Parallel()

	fields := map[string]any{
		"last_used": "2024-07-20T14:45:00Z",
	}

	got := GetLastUsed(fields)
	want := time.Date(2024, 7, 20, 14, 45, 0, 0, time.UTC)

	if !got.Equal(want) {
		t.Errorf("GetLastUsed() = %v, want %v", got, want)
	}
}

func TestGetLastUsed_NoField(t *testing.T) {
	t.Parallel()

	fields := map[string]any{}
	got := GetLastUsed(fields)

	if !got.IsZero() {
		t.Errorf("GetLastUsed() = %v, want zero", got)
	}
}

func TestSetTTL(t *testing.T) {
	t.Parallel()

	fields := make(map[string]any)
	lastUsed := time.Date(2024, 6, 1, 10, 0, 0, 0, time.UTC)

	SetTTL(fields, 30, lastUsed)

	ttlDays, ok := fields["ttl_days"].(int)
	if !ok {
		t.Fatalf("ttl_days is not an int: %T", fields["ttl_days"])
	}
	if ttlDays != 30 {
		t.Errorf("ttl_days = %d, want 30", ttlDays)
	}

	expiresAt, ok := fields["ttl_expires_at"].(string)
	if !ok {
		t.Fatalf("ttl_expires_at is not a string: %T", fields["ttl_expires_at"])
	}

	want := "2024-07-01T10:00:00Z" // 30 days after lastUsed
	if expiresAt != want {
		t.Errorf("ttl_expires_at = %q, want %q", expiresAt, want)
	}
}

func TestSetTTL_NilFields(t *testing.T) {
	t.Parallel()

	// Should not panic
	SetTTL(nil, 30, time.Now())
}

func TestGetTTLDays_Int(t *testing.T) {
	t.Parallel()

	fields := map[string]any{
		"ttl_days": 30,
	}

	got := GetTTLDays(fields)
	if got != 30 {
		t.Errorf("GetTTLDays() = %d, want 30", got)
	}
}

func TestGetTTLDays_Int64(t *testing.T) {
	t.Parallel()

	fields := map[string]any{
		"ttl_days": int64(90),
	}

	got := GetTTLDays(fields)
	if got != 90 {
		t.Errorf("GetTTLDays() = %d, want 90", got)
	}
}

func TestGetTTLDays_Float64(t *testing.T) {
	t.Parallel()

	// JSON unmarshals numbers as float64
	fields := map[string]any{
		"ttl_days": float64(45),
	}

	got := GetTTLDays(fields)
	if got != 45 {
		t.Errorf("GetTTLDays() = %d, want 45", got)
	}
}

func TestGetTTLDays_NoField(t *testing.T) {
	t.Parallel()

	fields := map[string]any{}
	got := GetTTLDays(fields)

	if got != 0 {
		t.Errorf("GetTTLDays() = %d, want 0", got)
	}
}

func TestGetTTLDays_WrongType(t *testing.T) {
	t.Parallel()

	fields := map[string]any{
		"ttl_days": "thirty",
	}

	got := GetTTLDays(fields)
	if got != 0 {
		t.Errorf("GetTTLDays() with string = %d, want 0", got)
	}
}

func TestGetTTLExpiresAt(t *testing.T) {
	t.Parallel()

	fields := map[string]any{
		"ttl_expires_at": "2024-07-01T10:00:00Z",
	}

	got := GetTTLExpiresAt(fields)
	want := time.Date(2024, 7, 1, 10, 0, 0, 0, time.UTC)

	if !got.Equal(want) {
		t.Errorf("GetTTLExpiresAt() = %v, want %v", got, want)
	}
}

func TestGetTTLExpiresAt_NoField(t *testing.T) {
	t.Parallel()

	fields := map[string]any{}
	got := GetTTLExpiresAt(fields)

	if !got.IsZero() {
		t.Errorf("GetTTLExpiresAt() = %v, want zero", got)
	}
}

func TestSetGitAnchors(t *testing.T) {
	t.Parallel()

	fields := make(map[string]any)
	anchors := []GitAnchor{
		{Path: "internal/api/handler.go"},
		{Path: "internal/api/routes.go"},
	}

	SetGitAnchors(fields, anchors)

	paths, ok := fields["git_anchors"].([]string)
	if !ok {
		t.Fatalf("git_anchors is not []string: %T", fields["git_anchors"])
	}

	if len(paths) != 2 {
		t.Fatalf("git_anchors len = %d, want 2", len(paths))
	}
	if paths[0] != "internal/api/handler.go" {
		t.Errorf("paths[0] = %q", paths[0])
	}
	if paths[1] != "internal/api/routes.go" {
		t.Errorf("paths[1] = %q", paths[1])
	}
}

func TestSetGitAnchors_Empty(t *testing.T) {
	t.Parallel()

	fields := map[string]any{
		"git_anchors": []string{"existing.go"},
	}

	SetGitAnchors(fields, nil)

	if _, ok := fields["git_anchors"]; ok {
		t.Error("SetGitAnchors(nil) should delete the field")
	}
}

func TestSetGitAnchors_EmptySlice(t *testing.T) {
	t.Parallel()

	fields := map[string]any{
		"git_anchors": []string{"existing.go"},
	}

	SetGitAnchors(fields, []GitAnchor{})

	if _, ok := fields["git_anchors"]; ok {
		t.Error("SetGitAnchors([]) should delete the field")
	}
}

func TestSetGitAnchors_NilFields(t *testing.T) {
	t.Parallel()

	// Should not panic
	SetGitAnchors(nil, []GitAnchor{{Path: "file.go"}})
}

func TestCheckEntryStaleness(t *testing.T) {
	t.Parallel()

	repo := setupTestRepo(t)

	// Create and commit a file
	createFile(t, repo, "api/handler.go", "package api")
	commitFile(t, repo, "api/handler.go", "Initial")

	// Use a time before the commit
	past := time.Now().Add(-time.Hour)

	fields := map[string]any{
		"id":             "KE-01JTEST",
		"git_anchors":    []any{"api/handler.go"},
		"last_confirmed": past.UTC().Format("2006-01-02T15:04:05Z07:00"),
	}

	info, err := CheckEntryStaleness(repo, fields)
	if err != nil {
		t.Fatalf("CheckEntryStaleness() error = %v", err)
	}

	if !info.IsStale {
		t.Error("CheckEntryStaleness() should detect staleness")
	}
}

func TestCheckEntryStaleness_NoAnchors(t *testing.T) {
	t.Parallel()

	repo := setupTestRepo(t)

	createFile(t, repo, "file.go", "content")
	commitFile(t, repo, "file.go", "Initial")

	fields := map[string]any{
		"id": "KE-01JTEST",
	}

	info, err := CheckEntryStaleness(repo, fields)
	if err != nil {
		t.Fatalf("CheckEntryStaleness() error = %v", err)
	}

	if info.IsStale {
		t.Error("CheckEntryStaleness() with no anchors should not be stale")
	}
}

func TestValidateAnchorPaths_Valid(t *testing.T) {
	t.Parallel()

	anchors := []GitAnchor{
		{Path: "internal/api/handler.go"},
		{Path: "pkg/util/helper.go"},
		{Path: "main.go"},
	}

	err := ValidateAnchorPaths(anchors)
	if err != nil {
		t.Errorf("ValidateAnchorPaths() error = %v", err)
	}
}

func TestValidateAnchorPaths_EmptyPath(t *testing.T) {
	t.Parallel()

	anchors := []GitAnchor{
		{Path: "valid/path.go"},
		{Path: ""},
	}

	err := ValidateAnchorPaths(anchors)
	if err == nil {
		t.Error("ValidateAnchorPaths() expected error for empty path")
	}
}

func TestValidateAnchorPaths_AbsolutePath(t *testing.T) {
	t.Parallel()

	anchors := []GitAnchor{
		{Path: "/absolute/path.go"},
	}

	err := ValidateAnchorPaths(anchors)
	if err == nil {
		t.Error("ValidateAnchorPaths() expected error for absolute path")
	}
}

func TestValidateAnchorPaths_Empty(t *testing.T) {
	t.Parallel()

	err := ValidateAnchorPaths(nil)
	if err != nil {
		t.Errorf("ValidateAnchorPaths(nil) error = %v", err)
	}

	err = ValidateAnchorPaths([]GitAnchor{})
	if err != nil {
		t.Errorf("ValidateAnchorPaths([]) error = %v", err)
	}
}

func TestRoundTrip_LastConfirmed(t *testing.T) {
	t.Parallel()

	fields := make(map[string]any)
	original := time.Date(2024, 6, 15, 10, 30, 45, 0, time.UTC)

	SetLastConfirmed(fields, original)
	retrieved := GetLastConfirmed(fields)

	// Compare with second precision (RFC3339 doesn't preserve nanoseconds)
	if !retrieved.Equal(original) {
		t.Errorf("Round-trip failed: got %v, want %v", retrieved, original)
	}
}

func TestRoundTrip_TTL(t *testing.T) {
	t.Parallel()

	fields := make(map[string]any)
	lastUsed := time.Date(2024, 6, 1, 10, 0, 0, 0, time.UTC)
	ttlDays := 30

	SetTTL(fields, ttlDays, lastUsed)

	gotDays := GetTTLDays(fields)
	if gotDays != ttlDays {
		t.Errorf("TTL days round-trip: got %d, want %d", gotDays, ttlDays)
	}

	gotExpires := GetTTLExpiresAt(fields)
	wantExpires := lastUsed.AddDate(0, 0, ttlDays)
	if !gotExpires.Equal(wantExpires) {
		t.Errorf("TTL expires round-trip: got %v, want %v", gotExpires, wantExpires)
	}
}

func TestRoundTrip_Anchors(t *testing.T) {
	t.Parallel()

	fields := make(map[string]any)
	original := []GitAnchor{
		{Path: "internal/api/handler.go"},
		{Path: "internal/api/routes.go"},
	}

	SetGitAnchors(fields, original)
	retrieved := ExtractAnchors(fields)

	if len(retrieved) != len(original) {
		t.Fatalf("Anchor round-trip len: got %d, want %d", len(retrieved), len(original))
	}

	for i := range original {
		if retrieved[i].Path != original[i].Path {
			t.Errorf("Anchor[%d] round-trip: got %q, want %q", i, retrieved[i].Path, original[i].Path)
		}
	}
}
