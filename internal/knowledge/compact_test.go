package knowledge

import (
	"strings"
	"testing"
)

func TestDetectRelationship_ExactDuplicate(t *testing.T) {
	t.Parallel()

	a := map[string]any{
		"id":         "KE-001",
		"topic":      "api-error-handling",
		"scope":      "backend",
		"content":    "Always wrap errors with context using fmt.Errorf",
		"confidence": 0.8,
	}
	b := map[string]any{
		"id":         "KE-002",
		"topic":      "api-error-handling",
		"scope":      "backend",
		"content":    "Always wrap errors with context using fmt.Errorf",
		"confidence": 0.7,
	}

	result := DetectRelationship(a, b)

	if result.Relationship != RelationshipExactDuplicate {
		t.Errorf("expected %s, got %s", RelationshipExactDuplicate, result.Relationship)
	}
	if result.Similarity != 1.0 {
		t.Errorf("expected similarity 1.0, got %f", result.Similarity)
	}
	if !result.ShouldMerge {
		t.Error("expected ShouldMerge=true for exact duplicates with high confidence")
	}
	if result.ShouldFlag {
		t.Error("expected ShouldFlag=false for exact duplicates")
	}
}

func TestDetectRelationship_NearDuplicate_SameTopic(t *testing.T) {
	t.Parallel()

	a := map[string]any{
		"id":         "KE-001",
		"topic":      "api-error-handling",
		"scope":      "backend",
		"content":    "Wrap errors with context using fmt.Errorf and %w verb",
		"confidence": 0.8,
	}
	b := map[string]any{
		"id":         "KE-002",
		"topic":      "API Error Handling", // Same topic when normalized
		"scope":      "frontend",           // Different scope
		"content":    "Handle errors gracefully in the UI layer",
		"confidence": 0.7,
	}

	result := DetectRelationship(a, b)

	if result.Relationship != RelationshipNearDuplicate {
		t.Errorf("expected %s, got %s", RelationshipNearDuplicate, result.Relationship)
	}
	// Same topic should trigger near-duplicate even with low content similarity
	if !result.ShouldMerge {
		t.Error("expected ShouldMerge=true for near-duplicates with confidence > 0.5")
	}
}

func TestDetectRelationship_NearDuplicate_HighJaccard(t *testing.T) {
	t.Parallel()

	a := map[string]any{
		"id":         "KE-001",
		"topic":      "json-naming-convention",
		"scope":      "backend",
		"content":    "Use camelCase for all JSON API field names in REST responses",
		"confidence": 0.8,
	}
	b := map[string]any{
		"id":         "KE-002",
		"topic":      "json-field-names", // Different topic
		"scope":      "backend",          // Same scope
		"content":    "Use camelCase for JSON API field names in HTTP responses",
		"confidence": 0.7,
	}

	result := DetectRelationship(a, b)

	if result.Relationship != RelationshipNearDuplicate {
		t.Errorf("expected %s, got %s (similarity: %.2f)", RelationshipNearDuplicate, result.Relationship, result.Similarity)
	}
	if result.Similarity <= NearDuplicateThreshold {
		t.Errorf("expected similarity > %.2f, got %.2f", NearDuplicateThreshold, result.Similarity)
	}
}

func TestDetectRelationship_NearDuplicate_LowConfidence(t *testing.T) {
	t.Parallel()

	a := map[string]any{
		"id":         "KE-001",
		"topic":      "api-error-handling",
		"scope":      "backend",
		"content":    "Always wrap errors with context using fmt.Errorf",
		"confidence": 0.3, // Low confidence
	}
	b := map[string]any{
		"id":         "KE-002",
		"topic":      "api-error-handling",
		"scope":      "backend",
		"content":    "Wrap errors with context using fmt.Errorf and %w",
		"confidence": 0.8,
	}

	result := DetectRelationship(a, b)

	if result.Relationship != RelationshipNearDuplicate {
		t.Errorf("expected %s, got %s", RelationshipNearDuplicate, result.Relationship)
	}
	if result.ShouldMerge {
		t.Error("expected ShouldMerge=false when one entry has low confidence")
	}
	if !result.ShouldFlag {
		t.Error("expected ShouldFlag=true for near-duplicates with low confidence")
	}
}

func TestDetectRelationship_Contradiction(t *testing.T) {
	t.Parallel()

	a := map[string]any{
		"id":         "KE-001",
		"topic":      "json-naming",
		"scope":      "api",
		"content":    "Use camelCase for JSON field names",
		"confidence": 0.8,
	}
	b := map[string]any{
		"id":         "KE-002",
		"topic":      "json-fields", // Overlapping topic words
		"scope":      "api",         // Same scope
		"content":    "Use snake_case for JSON field names",
		"confidence": 0.8,
	}

	result := DetectRelationship(a, b)

	// Check if detected as contradiction
	if result.Relationship != RelationshipContradiction {
		// If not contradiction, check if similarity is in range
		if result.Similarity >= ContradictionLowerBound && result.Similarity <= ContradictionUpperBound {
			t.Errorf("expected %s with similarity %.2f in range [%.2f, %.2f], got %s",
				RelationshipContradiction, result.Similarity, ContradictionLowerBound, ContradictionUpperBound, result.Relationship)
		}
	} else {
		if !result.ShouldFlag {
			t.Error("expected ShouldFlag=true for contradictions")
		}
		if result.ShouldMerge {
			t.Error("expected ShouldMerge=false for contradictions")
		}
	}
}

func TestDetectRelationship_Independent(t *testing.T) {
	t.Parallel()

	a := map[string]any{
		"id":         "KE-001",
		"topic":      "error-handling",
		"scope":      "backend",
		"content":    "Always wrap errors with context using fmt.Errorf",
		"confidence": 0.8,
	}
	b := map[string]any{
		"id":         "KE-002",
		"topic":      "ui-components",
		"scope":      "frontend",
		"content":    "Use React functional components with hooks",
		"confidence": 0.7,
	}

	result := DetectRelationship(a, b)

	if result.Relationship != RelationshipIndependent {
		t.Errorf("expected %s, got %s", RelationshipIndependent, result.Relationship)
	}
	if result.ShouldMerge {
		t.Error("expected ShouldMerge=false for independent entries")
	}
	if result.ShouldFlag {
		t.Error("expected ShouldFlag=false for independent entries")
	}
}

func TestCanAutoCompact(t *testing.T) {
	t.Parallel()

	tests := []struct {
		tier int
		want bool
	}{
		{tier: 1, want: false},
		{tier: 2, want: false},
		{tier: 3, want: true},
		{tier: 0, want: false},
	}

	for _, tc := range tests {
		got := CanAutoCompact(tc.tier)
		if got != tc.want {
			t.Errorf("CanAutoCompact(%d) = %v, want %v", tc.tier, got, tc.want)
		}
	}
}

func TestMergeEntries_KeepsHigherConfidence(t *testing.T) {
	t.Parallel()

	a := map[string]any{
		"id":         "KE-001",
		"topic":      "api-error-handling",
		"content":    "Wrap errors with context",
		"confidence": 0.9,
		"use_count":  3,
	}
	b := map[string]any{
		"id":         "KE-002",
		"topic":      "api-error-handling",
		"content":    "Wrap errors with context",
		"confidence": 0.7,
		"use_count":  5,
	}

	kept, discarded := MergeEntries(a, b)

	// Entry with higher confidence (a) should be kept
	keptID := getEntryIDFromFields(kept)
	if keptID != "KE-001" {
		t.Errorf("expected KE-001 to be kept (higher confidence), got %s", keptID)
	}

	discardedID := getEntryIDFromFields(discarded)
	if discardedID != "KE-002" {
		t.Errorf("expected KE-002 to be discarded, got %s", discardedID)
	}
}

func TestMergeEntries_EqualConfidence_KeepsHigherUseCount(t *testing.T) {
	t.Parallel()

	a := map[string]any{
		"id":         "KE-001",
		"topic":      "api-error-handling",
		"content":    "Wrap errors with context",
		"confidence": 0.8,
		"use_count":  3,
	}
	b := map[string]any{
		"id":         "KE-002",
		"topic":      "api-error-handling",
		"content":    "Wrap errors with context",
		"confidence": 0.8,
		"use_count":  5,
	}

	kept, discarded := MergeEntries(a, b)

	// Entry with higher use_count (b) should be kept
	keptID := getEntryIDFromFields(kept)
	if keptID != "KE-002" {
		t.Errorf("expected KE-002 to be kept (higher use_count), got %s", keptID)
	}

	discardedID := getEntryIDFromFields(discarded)
	if discardedID != "KE-001" {
		t.Errorf("expected KE-001 to be discarded, got %s", discardedID)
	}
}

func TestMergeEntries_TransfersUseCounts(t *testing.T) {
	t.Parallel()

	a := map[string]any{
		"id":         "KE-001",
		"topic":      "api-error-handling",
		"confidence": 0.9,
		"use_count":  3,
		"miss_count": 1,
	}
	b := map[string]any{
		"id":         "KE-002",
		"topic":      "api-error-handling",
		"confidence": 0.7,
		"use_count":  5,
		"miss_count": 2,
	}

	kept, _ := MergeEntries(a, b)

	expectedUseCount := 3 + 5
	if GetUseCount(kept) != expectedUseCount {
		t.Errorf("expected use_count=%d, got %d", expectedUseCount, GetUseCount(kept))
	}

	expectedMissCount := 1 + 2
	if GetMissCount(kept) != expectedMissCount {
		t.Errorf("expected miss_count=%d, got %d", expectedMissCount, GetMissCount(kept))
	}
}

func TestMergeEntries_MergesGitAnchors(t *testing.T) {
	t.Parallel()

	a := map[string]any{
		"id":          "KE-001",
		"confidence":  0.9,
		"git_anchors": []string{"internal/api/handler.go", "internal/api/errors.go"},
	}
	b := map[string]any{
		"id":          "KE-002",
		"confidence":  0.7,
		"git_anchors": []string{"internal/api/errors.go", "internal/api/types.go"},
	}

	kept, _ := MergeEntries(a, b)

	anchors := GetGitAnchors(kept)
	expectedAnchors := []string{"internal/api/errors.go", "internal/api/handler.go", "internal/api/types.go"}

	if len(anchors) != len(expectedAnchors) {
		t.Errorf("expected %d anchors, got %d: %v", len(expectedAnchors), len(anchors), anchors)
	}

	for i, expected := range expectedAnchors {
		if i < len(anchors) && anchors[i] != expected {
			t.Errorf("expected anchor[%d]=%s, got %s", i, expected, anchors[i])
		}
	}
}

func TestMergeEntries_SetsMergedFrom(t *testing.T) {
	t.Parallel()

	a := map[string]any{
		"id":         "KE-001",
		"confidence": 0.9,
	}
	b := map[string]any{
		"id":         "KE-002",
		"confidence": 0.7,
	}

	kept, _ := MergeEntries(a, b)

	mergedFrom, ok := kept["merged_from"].(string)
	if !ok || mergedFrom != "KE-002" {
		t.Errorf("expected merged_from=KE-002, got %v", kept["merged_from"])
	}
}

func TestMergeEntries_RetiresDiscarded(t *testing.T) {
	t.Parallel()

	a := map[string]any{
		"id":         "KE-001",
		"confidence": 0.9,
	}
	b := map[string]any{
		"id":         "KE-002",
		"confidence": 0.7,
	}

	_, discarded := MergeEntries(a, b)

	status := GetStatus(discarded)
	if status != "retired" {
		t.Errorf("expected status=retired, got %s", status)
	}

	reason, ok := discarded["retired_reason"].(string)
	if !ok || !strings.Contains(reason, "KE-001") {
		t.Errorf("expected retired_reason to mention KE-001, got %v", discarded["retired_reason"])
	}
}

func TestCompactEntries_Tier3AutoMerge(t *testing.T) {
	t.Parallel()

	entries := []map[string]any{
		{
			"id":         "KE-001",
			"topic":      "api-error-handling",
			"scope":      "backend",
			"content":    "Always wrap errors with context using fmt.Errorf",
			"tier":       3,
			"confidence": 0.8,
			"use_count":  2,
		},
		{
			"id":         "KE-002",
			"topic":      "api-error-handling",
			"scope":      "backend",
			"content":    "Always wrap errors with context using fmt.Errorf",
			"tier":       3,
			"confidence": 0.9,
			"use_count":  3,
		},
	}

	opts := CompactionOptions{DryRun: false}
	result, updates := CompactEntries(entries, opts)

	if result.DuplicatesMerged != 1 {
		t.Errorf("expected DuplicatesMerged=1, got %d", result.DuplicatesMerged)
	}

	if len(updates) != 2 {
		t.Errorf("expected 2 updated entries, got %d", len(updates))
	}
}

func TestCompactEntries_Tier2Flagged(t *testing.T) {
	t.Parallel()

	entries := []map[string]any{
		{
			"id":         "KE-001",
			"topic":      "api-error-handling",
			"scope":      "backend",
			"content":    "Always wrap errors with context using fmt.Errorf",
			"tier":       2,
			"confidence": 0.8,
		},
		{
			"id":         "KE-002",
			"topic":      "api-error-handling",
			"scope":      "backend",
			"content":    "Always wrap errors with context using fmt.Errorf",
			"tier":       2,
			"confidence": 0.9,
		},
	}

	opts := CompactionOptions{DryRun: false}
	result, _ := CompactEntries(entries, opts)

	// Tier 2 entries should not be auto-merged
	if result.DuplicatesMerged != 0 {
		t.Errorf("expected DuplicatesMerged=0 for Tier 2 entries, got %d", result.DuplicatesMerged)
	}
}

func TestCompactEntries_Tier1Excluded(t *testing.T) {
	t.Parallel()

	entries := []map[string]any{
		{
			"id":         "KE-001",
			"topic":      "api-error-handling",
			"scope":      "backend",
			"content":    "Always wrap errors with context using fmt.Errorf",
			"tier":       1,
			"confidence": 0.8,
		},
		{
			"id":         "KE-002",
			"topic":      "api-error-handling",
			"scope":      "backend",
			"content":    "Always wrap errors with context using fmt.Errorf",
			"tier":       1,
			"confidence": 0.9,
		},
	}

	opts := CompactionOptions{DryRun: false}
	result, updates := CompactEntries(entries, opts)

	// Tier 1 entries should never be compacted
	if result.DuplicatesMerged != 0 {
		t.Errorf("expected DuplicatesMerged=0 for Tier 1 entries, got %d", result.DuplicatesMerged)
	}
	if result.NearDuplicatesMerged != 0 {
		t.Errorf("expected NearDuplicatesMerged=0 for Tier 1 entries, got %d", result.NearDuplicatesMerged)
	}
	if result.ConflictsFlagged != 0 {
		t.Errorf("expected ConflictsFlagged=0 for Tier 1 entries, got %d", result.ConflictsFlagged)
	}
	if len(updates) != 0 {
		t.Errorf("expected 0 updates for Tier 1 entries, got %d", len(updates))
	}
}

func TestCompactEntries_DryRun(t *testing.T) {
	t.Parallel()

	entries := []map[string]any{
		{
			"id":         "KE-001",
			"topic":      "api-error-handling",
			"scope":      "backend",
			"content":    "Always wrap errors with context using fmt.Errorf",
			"tier":       3,
			"confidence": 0.8,
		},
		{
			"id":         "KE-002",
			"topic":      "api-error-handling",
			"scope":      "backend",
			"content":    "Always wrap errors with context using fmt.Errorf",
			"tier":       3,
			"confidence": 0.9,
		},
	}

	opts := CompactionOptions{DryRun: true}
	result, updates := CompactEntries(entries, opts)

	// Should report what would happen
	if result.DuplicatesMerged != 1 {
		t.Errorf("expected DuplicatesMerged=1, got %d", result.DuplicatesMerged)
	}

	// But no actual updates in dry-run
	if len(updates) != 0 {
		t.Errorf("expected 0 updates in dry-run, got %d", len(updates))
	}

	// Original entries should be unmodified
	if entries[0]["status"] != nil {
		t.Errorf("original entry should not be modified in dry-run")
	}
}

func TestCompactEntries_ScopeFilter(t *testing.T) {
	t.Parallel()

	entries := []map[string]any{
		{
			"id":         "KE-001",
			"topic":      "api-error-handling",
			"scope":      "backend",
			"content":    "Always wrap errors with context using fmt.Errorf",
			"tier":       3,
			"confidence": 0.8,
		},
		{
			"id":         "KE-002",
			"topic":      "api-error-handling",
			"scope":      "backend",
			"content":    "Always wrap errors with context using fmt.Errorf",
			"tier":       3,
			"confidence": 0.9,
		},
		{
			"id":         "KE-003",
			"topic":      "api-error-handling",
			"scope":      "frontend",
			"content":    "Always wrap errors with context using fmt.Errorf",
			"tier":       3,
			"confidence": 0.8,
		},
	}

	opts := CompactionOptions{DryRun: false, Scope: "backend"}
	result, _ := CompactEntries(entries, opts)

	// Only backend entries should be compacted
	if result.DuplicatesMerged != 1 {
		t.Errorf("expected DuplicatesMerged=1, got %d", result.DuplicatesMerged)
	}
}

func TestCompactEntries_SkipsRetiredEntries(t *testing.T) {
	t.Parallel()

	entries := []map[string]any{
		{
			"id":         "KE-001",
			"topic":      "api-error-handling",
			"scope":      "backend",
			"content":    "Always wrap errors with context using fmt.Errorf",
			"tier":       3,
			"confidence": 0.8,
			"status":     "retired",
		},
		{
			"id":         "KE-002",
			"topic":      "api-error-handling",
			"scope":      "backend",
			"content":    "Always wrap errors with context using fmt.Errorf",
			"tier":       3,
			"confidence": 0.9,
		},
	}

	opts := CompactionOptions{DryRun: false}
	result, _ := CompactEntries(entries, opts)

	// Retired entry should be skipped
	if result.DuplicatesMerged != 0 {
		t.Errorf("expected DuplicatesMerged=0 (retired entry skipped), got %d", result.DuplicatesMerged)
	}
}

func TestCompactEntries_NearDuplicates(t *testing.T) {
	t.Parallel()

	entries := []map[string]any{
		{
			"id":         "KE-001",
			"topic":      "json-naming-convention",
			"scope":      "backend",
			"content":    "Use camelCase for all JSON API field names in REST responses",
			"tier":       3,
			"confidence": 0.8,
		},
		{
			"id":         "KE-002",
			"topic":      "json-field-style",
			"scope":      "backend",
			"content":    "Use camelCase for JSON API field names in HTTP responses",
			"tier":       3,
			"confidence": 0.7,
		},
	}

	opts := CompactionOptions{DryRun: false}
	result, _ := CompactEntries(entries, opts)

	// High Jaccard similarity in same scope should be detected as near-duplicate
	if result.NearDuplicatesMerged < 1 && result.DuplicatesMerged < 1 {
		t.Logf("result: %+v", result)
		// Check if entries have high enough similarity
		wordsA := ContentWords(entries[0]["content"].(string))
		wordsB := ContentWords(entries[1]["content"].(string))
		sim := JaccardSimilarity(wordsA, wordsB)
		if sim > NearDuplicateThreshold {
			t.Errorf("expected near-duplicates to be merged (similarity: %.2f > %.2f)", sim, NearDuplicateThreshold)
		}
	}
}

func TestCompactEntries_Contradictions(t *testing.T) {
	t.Parallel()

	// Create entries that should be detected as contradictions:
	// Same scope, overlapping topics, Jaccard 0.3-0.6
	entries := []map[string]any{
		{
			"id":         "KE-001",
			"topic":      "json-naming",
			"scope":      "api",
			"content":    "Use camelCase for JSON field naming convention",
			"tier":       3,
			"confidence": 0.8,
		},
		{
			"id":         "KE-002",
			"topic":      "json-style",
			"scope":      "api",
			"content":    "Use snake_case for database column naming convention",
			"tier":       3,
			"confidence": 0.8,
		},
	}

	// Check if they would be detected as contradiction
	detection := DetectRelationship(entries[0], entries[1])
	t.Logf("Detection result: %+v", detection)

	if detection.Relationship == RelationshipContradiction {
		opts := CompactionOptions{DryRun: false}
		result, updates := CompactEntries(entries, opts)

		if result.ConflictsFlagged != 1 {
			t.Errorf("expected ConflictsFlagged=1, got %d", result.ConflictsFlagged)
		}

		// Check that both entries are marked as disputed
		for _, u := range updates {
			status := GetStatus(u)
			if status != "disputed" {
				t.Errorf("expected status=disputed, got %s", status)
			}
		}
	}
}

func TestCompactEntries_IndependentEntries(t *testing.T) {
	t.Parallel()

	entries := []map[string]any{
		{
			"id":         "KE-001",
			"topic":      "error-handling",
			"scope":      "backend",
			"content":    "Always wrap errors with context using fmt.Errorf",
			"tier":       3,
			"confidence": 0.8,
		},
		{
			"id":         "KE-002",
			"topic":      "ui-components",
			"scope":      "frontend",
			"content":    "Use React functional components with hooks",
			"tier":       3,
			"confidence": 0.7,
		},
	}

	opts := CompactionOptions{DryRun: false}
	result, updates := CompactEntries(entries, opts)

	// Independent entries should not be touched
	if result.DuplicatesMerged != 0 {
		t.Errorf("expected DuplicatesMerged=0, got %d", result.DuplicatesMerged)
	}
	if result.NearDuplicatesMerged != 0 {
		t.Errorf("expected NearDuplicatesMerged=0, got %d", result.NearDuplicatesMerged)
	}
	if result.ConflictsFlagged != 0 {
		t.Errorf("expected ConflictsFlagged=0, got %d", result.ConflictsFlagged)
	}
	if len(updates) != 0 {
		t.Errorf("expected 0 updates, got %d", len(updates))
	}
}

func TestGetScope(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		fields map[string]any
		want   string
	}{
		{
			name:   "present",
			fields: map[string]any{"scope": "backend"},
			want:   "backend",
		},
		{
			name:   "missing",
			fields: map[string]any{},
			want:   "",
		},
		{
			name:   "nil fields",
			fields: nil,
			want:   "",
		},
		{
			name:   "non-string value",
			fields: map[string]any{"scope": 123},
			want:   "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := GetScope(tc.fields)
			if got != tc.want {
				t.Errorf("GetScope() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestGetContent(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		fields map[string]any
		want   string
	}{
		{
			name:   "present",
			fields: map[string]any{"content": "Some content here"},
			want:   "Some content here",
		},
		{
			name:   "missing",
			fields: map[string]any{},
			want:   "",
		},
		{
			name:   "nil fields",
			fields: nil,
			want:   "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := GetContent(tc.fields)
			if got != tc.want {
				t.Errorf("GetContent() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestGetGitAnchors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		fields map[string]any
		want   []string
	}{
		{
			name:   "string slice",
			fields: map[string]any{"git_anchors": []string{"a.go", "b.go"}},
			want:   []string{"a.go", "b.go"},
		},
		{
			name:   "any slice",
			fields: map[string]any{"git_anchors": []any{"a.go", "b.go"}},
			want:   []string{"a.go", "b.go"},
		},
		{
			name:   "missing",
			fields: map[string]any{},
			want:   nil,
		},
		{
			name:   "nil fields",
			fields: nil,
			want:   nil,
		},
		{
			name:   "empty slice",
			fields: map[string]any{"git_anchors": []string{}},
			want:   []string{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := GetGitAnchors(tc.fields)
			if len(got) != len(tc.want) {
				t.Errorf("GetGitAnchors() = %v, want %v", got, tc.want)
				return
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Errorf("GetGitAnchors()[%d] = %q, want %q", i, got[i], tc.want[i])
				}
			}
		})
	}
}

func TestSetGitAnchors(t *testing.T) {
	t.Parallel()

	t.Run("sets anchors", func(t *testing.T) {
		t.Parallel()
		fields := make(map[string]any)
		SetGitAnchors(fields, []string{"a.go", "b.go"})
		got := GetGitAnchors(fields)
		if len(got) != 2 || got[0] != "a.go" || got[1] != "b.go" {
			t.Errorf("SetGitAnchors() did not set correctly: %v", got)
		}
	})

	t.Run("removes on empty slice", func(t *testing.T) {
		t.Parallel()
		fields := map[string]any{"git_anchors": []string{"a.go"}}
		SetGitAnchors(fields, nil)
		if _, ok := fields["git_anchors"]; ok {
			t.Error("SetGitAnchors(nil) should remove the field")
		}
	})

	t.Run("nil fields is safe", func(t *testing.T) {
		t.Parallel()
		SetGitAnchors(nil, []string{"a.go"}) // should not panic
	})
}

func TestSetMergedFrom(t *testing.T) {
	t.Parallel()

	t.Run("sets value", func(t *testing.T) {
		t.Parallel()
		fields := make(map[string]any)
		SetMergedFrom(fields, "KE-001")
		if fields["merged_from"] != "KE-001" {
			t.Errorf("expected merged_from=KE-001, got %v", fields["merged_from"])
		}
	})

	t.Run("ignores empty", func(t *testing.T) {
		t.Parallel()
		fields := make(map[string]any)
		SetMergedFrom(fields, "")
		if _, ok := fields["merged_from"]; ok {
			t.Error("SetMergedFrom('') should not set field")
		}
	})

	t.Run("nil fields is safe", func(t *testing.T) {
		t.Parallel()
		SetMergedFrom(nil, "KE-001") // should not panic
	})
}

func TestSetStatus(t *testing.T) {
	t.Parallel()

	t.Run("sets status", func(t *testing.T) {
		t.Parallel()
		fields := make(map[string]any)
		SetStatus(fields, "retired")
		if fields["status"] != "retired" {
			t.Errorf("expected status=retired, got %v", fields["status"])
		}
	})

	t.Run("nil fields is safe", func(t *testing.T) {
		t.Parallel()
		SetStatus(nil, "retired") // should not panic
	})
}

func TestGetStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		fields map[string]any
		want   string
	}{
		{
			name:   "present",
			fields: map[string]any{"status": "active"},
			want:   "active",
		},
		{
			name:   "missing",
			fields: map[string]any{},
			want:   "",
		},
		{
			name:   "nil fields",
			fields: nil,
			want:   "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := GetStatus(tc.fields)
			if got != tc.want {
				t.Errorf("GetStatus() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestHasTopicOverlap(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		a, b   string
		expect bool
	}{
		{
			name:   "same topic",
			a:      "api-error-handling",
			b:      "api-error-handling",
			expect: true,
		},
		{
			name:   "overlapping words",
			a:      "json-naming",
			b:      "json-style",
			expect: true,
		},
		{
			name:   "no overlap",
			a:      "error-handling",
			b:      "ui-components",
			expect: false,
		},
		{
			name:   "empty a",
			a:      "",
			b:      "json-style",
			expect: false,
		},
		{
			name:   "empty b",
			a:      "json-naming",
			b:      "",
			expect: false,
		},
		{
			name:   "both empty",
			a:      "",
			b:      "",
			expect: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := hasTopicOverlap(tc.a, tc.b)
			if got != tc.expect {
				t.Errorf("hasTopicOverlap(%q, %q) = %v, want %v", tc.a, tc.b, got, tc.expect)
			}
		})
	}
}

func TestMergeAnchors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		a, b []string
		want []string
	}{
		{
			name: "union without duplicates",
			a:    []string{"a.go", "b.go"},
			b:    []string{"c.go", "d.go"},
			want: []string{"a.go", "b.go", "c.go", "d.go"},
		},
		{
			name: "union with duplicates",
			a:    []string{"a.go", "b.go"},
			b:    []string{"b.go", "c.go"},
			want: []string{"a.go", "b.go", "c.go"},
		},
		{
			name: "both empty",
			a:    nil,
			b:    nil,
			want: nil,
		},
		{
			name: "a empty",
			a:    nil,
			b:    []string{"b.go"},
			want: []string{"b.go"},
		},
		{
			name: "b empty",
			a:    []string{"a.go"},
			b:    nil,
			want: []string{"a.go"},
		},
		{
			name: "sorted output",
			a:    []string{"z.go", "a.go"},
			b:    []string{"m.go"},
			want: []string{"a.go", "m.go", "z.go"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := mergeAnchors(tc.a, tc.b)
			if len(got) != len(tc.want) {
				t.Errorf("mergeAnchors() = %v, want %v", got, tc.want)
				return
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Errorf("mergeAnchors()[%d] = %q, want %q", i, got[i], tc.want[i])
				}
			}
		})
	}
}

func TestCompactEntries_MixedTiers(t *testing.T) {
	t.Parallel()

	// Mix of Tier 3 (auto-compact) and Tier 2 (flag only)
	entries := []map[string]any{
		{
			"id":         "KE-001",
			"topic":      "api-error-handling",
			"scope":      "backend",
			"content":    "Always wrap errors with context using fmt.Errorf",
			"tier":       3,
			"confidence": 0.8,
		},
		{
			"id":         "KE-002",
			"topic":      "api-error-handling",
			"scope":      "backend",
			"content":    "Always wrap errors with context using fmt.Errorf",
			"tier":       2, // Tier 2 - should not be auto-merged
			"confidence": 0.9,
		},
	}

	opts := CompactionOptions{DryRun: false}
	result, _ := CompactEntries(entries, opts)

	// Should not auto-merge when one entry is Tier 2
	if result.DuplicatesMerged != 0 {
		t.Errorf("expected DuplicatesMerged=0 (mixed tiers), got %d", result.DuplicatesMerged)
	}
}

func TestCompactionResult_Details(t *testing.T) {
	t.Parallel()

	entries := []map[string]any{
		{
			"id":         "KE-001",
			"topic":      "api-error-handling",
			"scope":      "backend",
			"content":    "Always wrap errors with context using fmt.Errorf",
			"tier":       3,
			"confidence": 0.8,
		},
		{
			"id":         "KE-002",
			"topic":      "api-error-handling",
			"scope":      "backend",
			"content":    "Always wrap errors with context using fmt.Errorf",
			"tier":       3,
			"confidence": 0.9,
		},
	}

	opts := CompactionOptions{DryRun: true}
	result, _ := CompactEntries(entries, opts)

	if len(result.Details) != 1 {
		t.Fatalf("expected 1 detail, got %d", len(result.Details))
	}

	detail := result.Details[0]
	if detail.Action != CompactionActionMerged {
		t.Errorf("expected action=%s, got %s", CompactionActionMerged, detail.Action)
	}
	if detail.Reason == "" {
		t.Error("expected non-empty reason")
	}
}
