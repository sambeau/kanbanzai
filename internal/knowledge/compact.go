// Package knowledge provides knowledge entry lifecycle management.
package knowledge

import (
	"fmt"
	"sort"
	"time"
)

// CompactionAction describes what happened to an entry pair.
type CompactionAction string

const (
	// CompactionActionMerged indicates entries were merged.
	CompactionActionMerged CompactionAction = "merged"
	// CompactionActionDisputed indicates entries have contradictory content.
	CompactionActionDisputed CompactionAction = "disputed"
	// CompactionActionSkipped indicates entries were not modified.
	CompactionActionSkipped CompactionAction = "skipped"
)

// CompactionDetail describes a single compaction action.
type CompactionDetail struct {
	Action    CompactionAction
	Kept      string   // Entry ID that was kept (for merged)
	Discarded string   // Entry ID that was discarded (for merged)
	Entries   []string // Entry IDs involved (for disputed)
	Reason    string   // Human-readable explanation
}

// CompactionResult is the result of a compaction run.
type CompactionResult struct {
	DuplicatesMerged     int
	NearDuplicatesMerged int
	ConflictsFlagged     int
	Details              []CompactionDetail
}

// CompactionOptions configures compaction behavior.
type CompactionOptions struct {
	DryRun bool   // Don't modify, just report what would happen
	Scope  string // Limit to specific scope (optional)
}

// DetectionResult describes what kind of relationship exists between two entries.
type DetectionResult struct {
	Relationship string  // "exact_duplicate", "near_duplicate", "contradiction", "independent"
	Similarity   float64 // Jaccard similarity between content
	ShouldMerge  bool    // True if entries should be auto-merged
	ShouldFlag   bool    // True if entries should be flagged for review
}

// Relationship constants.
const (
	RelationshipExactDuplicate = "exact_duplicate"
	RelationshipNearDuplicate  = "near_duplicate"
	RelationshipContradiction  = "contradiction"
	RelationshipIndependent    = "independent"
)

// Similarity thresholds from spec.
const (
	// NearDuplicateThreshold is the minimum Jaccard similarity for near-duplicates.
	NearDuplicateThreshold = 0.65
	// ContradictionLowerBound is the minimum Jaccard for contradiction detection.
	ContradictionLowerBound = 0.3
	// ContradictionUpperBound is the maximum Jaccard for contradiction detection.
	ContradictionUpperBound = 0.6
	// AutoMergeMinConfidence is the minimum confidence for auto-merge.
	AutoMergeMinConfidence = 0.5
)

// DetectRelationship determines the relationship between two knowledge entries.
// Detection rules from spec §14.3:
// - Exact duplicate: Same topic AND same normalised content
// - Near-duplicate: Same topic OR Jaccard > 0.65 in same scope
// - Contradiction: Same scope AND topic overlap AND Jaccard 0.3–0.6
// - Independent: Different topic AND different scope
func DetectRelationship(a, b map[string]any) DetectionResult {
	topicA := GetTopic(a)
	topicB := GetTopic(b)
	scopeA := GetScope(a)
	scopeB := GetScope(b)
	contentA := GetContent(a)
	contentB := GetContent(b)

	// Normalize topics for comparison
	normalizedTopicA := NormalizeTopic(topicA)
	normalizedTopicB := NormalizeTopic(topicB)
	sameTopic := normalizedTopicA != "" && normalizedTopicA == normalizedTopicB

	// Compute content similarity
	wordsA := ContentWords(contentA)
	wordsB := ContentWords(contentB)
	similarity := JaccardSimilarity(wordsA, wordsB)

	// Check for exact duplicate: same topic AND same normalised content
	if sameTopic && similarity == 1.0 {
		confA := GetConfidence(a)
		confB := GetConfidence(b)
		shouldMerge := confA >= AutoMergeMinConfidence || confB >= AutoMergeMinConfidence
		return DetectionResult{
			Relationship: RelationshipExactDuplicate,
			Similarity:   similarity,
			ShouldMerge:  shouldMerge,
			ShouldFlag:   false,
		}
	}

	sameScope := scopeA != "" && scopeA == scopeB

	// Check for near-duplicate: Same topic OR Jaccard > 0.65 in same scope
	if sameTopic || (sameScope && similarity > NearDuplicateThreshold) {
		confA := GetConfidence(a)
		confB := GetConfidence(b)
		// Auto-merge only if both confidence > 0.5
		shouldMerge := confA > AutoMergeMinConfidence && confB > AutoMergeMinConfidence
		return DetectionResult{
			Relationship: RelationshipNearDuplicate,
			Similarity:   similarity,
			ShouldMerge:  shouldMerge,
			ShouldFlag:   !shouldMerge, // Flag if not auto-mergeable
		}
	}

	// Check for contradiction: Same scope AND topic overlap AND Jaccard 0.3–0.6
	// "Topic overlap" means normalized topics share common words
	topicOverlap := hasTopicOverlap(normalizedTopicA, normalizedTopicB)
	if sameScope && topicOverlap && similarity >= ContradictionLowerBound && similarity <= ContradictionUpperBound {
		return DetectionResult{
			Relationship: RelationshipContradiction,
			Similarity:   similarity,
			ShouldMerge:  false,
			ShouldFlag:   true,
		}
	}

	// Independent: Different topic AND different scope (or no match above)
	return DetectionResult{
		Relationship: RelationshipIndependent,
		Similarity:   similarity,
		ShouldMerge:  false,
		ShouldFlag:   false,
	}
}

// hasTopicOverlap checks if two normalized topics share any common words.
func hasTopicOverlap(topicA, topicB string) bool {
	if topicA == "" || topicB == "" {
		return false
	}

	// Use ContentWords which handles splitting and normalization
	wordsA := ContentWords(topicA)
	wordsB := ContentWords(topicB)

	for w := range wordsA {
		if _, ok := wordsB[w]; ok {
			return true
		}
	}
	return false
}

// CanAutoCompact returns true if the entry tier allows auto-compaction.
// From spec §14.2:
// - Tier 3: Auto-compacted according to rules
// - Tier 2: Flagged for human review; not auto-modified
// - Tier 1: Never compacted
func CanAutoCompact(tier int) bool {
	return tier == 3
}

// MergeEntries merges two entries, returning the updated kept and discarded entries.
// The kept entry receives:
// - Added usage counts from discarded entry
// - Merged git_anchors (union of both lists)
// - Updated merged_from field with discarded entry ID
// The discarded entry is retired with reason "merged into {kept-id}".
//
// The entry with higher confidence is kept per spec §14.5.
// If confidence is equal, the entry with more use_count is kept.
// If still equal, entry 'a' is kept (arbitrary but deterministic).
func MergeEntries(a, b map[string]any, now time.Time) (updatedKept, updatedDiscarded map[string]any) {
	confA := GetConfidence(a)
	confB := GetConfidence(b)
	useCountA := GetUseCount(a)
	useCountB := GetUseCount(b)

	// Decide which entry to keep
	keepA := true
	if confB > confA {
		keepA = false
	} else if confB == confA && useCountB > useCountA {
		keepA = false
	}

	var kept, discarded map[string]any
	if keepA {
		kept = copyFields(a)
		discarded = copyFields(b)
	} else {
		kept = copyFields(b)
		discarded = copyFields(a)
	}

	// Transfer usage counts from discarded to kept
	keptUseCount := GetUseCount(kept)
	discardedUseCount := GetUseCount(discarded)
	kept["use_count"] = keptUseCount + discardedUseCount

	// Also transfer miss_count
	keptMissCount := GetMissCount(kept)
	discardedMissCount := GetMissCount(discarded)
	kept["miss_count"] = keptMissCount + discardedMissCount

	// Merge git_anchors (union of both lists)
	keptAnchors := GetGitAnchors(kept)
	discardedAnchors := GetGitAnchors(discarded)
	mergedAnchors := mergeAnchors(keptAnchors, discardedAnchors)
	SetGitAnchors(kept, mergedAnchors)

	// Update merged_from field in kept entry
	discardedID := getFieldString(discarded, "id")
	if discardedID == "" {
		discardedID = GetEntryID(discarded)
	}
	SetMergedFrom(kept, discardedID)

	// Retire discarded entry
	keptID := getFieldString(kept, "id")
	if keptID == "" {
		keptID = GetEntryID(kept)
	}
	SetStatus(discarded, "retired")
	discarded["retired_reason"] = fmt.Sprintf("merged into %s", keptID)
	discarded["retired_at"] = now.Format(time.RFC3339)

	return kept, discarded
}

// copyFields creates a shallow copy of the fields map.
func copyFields(fields map[string]any) map[string]any {
	if fields == nil {
		return make(map[string]any)
	}
	result := make(map[string]any, len(fields))
	for k, v := range fields {
		result[k] = v
	}
	return result
}

// mergeAnchors returns the union of two anchor lists, removing duplicates.
func mergeAnchors(a, b []string) []string {
	seen := make(map[string]struct{})
	var result []string

	for _, anchor := range a {
		if _, ok := seen[anchor]; !ok {
			seen[anchor] = struct{}{}
			result = append(result, anchor)
		}
	}
	for _, anchor := range b {
		if _, ok := seen[anchor]; !ok {
			seen[anchor] = struct{}{}
			result = append(result, anchor)
		}
	}

	// Sort for deterministic output
	sort.Strings(result)
	return result
}

// CompactEntries runs compaction on a set of knowledge entries.
// Returns the result and list of entries that need to be updated.
// Entries are compared pairwise; an entry may be involved in multiple comparisons.
func CompactEntries(entries []map[string]any, opts CompactionOptions) (CompactionResult, []map[string]any) {
	if opts.Scope != "" {
		return CompactEntriesInScope(entries, opts.Scope, opts)
	}

	// Filter out already-retired entries
	var active []map[string]any
	for _, e := range entries {
		status := getFieldString(e, "status")
		if status != "retired" {
			active = append(active, e)
		}
	}

	return compactActiveEntries(active, opts)
}

// CompactEntriesInScope runs compaction on entries within a scope.
func CompactEntriesInScope(entries []map[string]any, scope string, opts CompactionOptions) (CompactionResult, []map[string]any) {
	// Filter to entries in the specified scope
	var scoped []map[string]any
	for _, e := range entries {
		if GetScope(e) == scope {
			status := getFieldString(e, "status")
			if status != "retired" {
				scoped = append(scoped, e)
			}
		}
	}

	return compactActiveEntries(scoped, opts)
}

// compactActiveEntries performs compaction on pre-filtered active entries.
func compactActiveEntries(entries []map[string]any, opts CompactionOptions) (CompactionResult, []map[string]any) {
	result := CompactionResult{}
	updatedEntries := make(map[string]map[string]any) // ID -> updated fields
	processed := make(map[string]bool)                // Track processed entry IDs
	disputed := make(map[string][]string)             // Track disputed groups

	n := len(entries)
	for i := 0; i < n; i++ {
		entryA := entries[i]
		idA := getEntryIDFromFields(entryA)

		// Skip if already processed (merged into another entry)
		if processed[idA] {
			continue
		}

		tierA := GetTier(entryA)

		// Tier 1 entries are never compacted
		if tierA == 1 {
			continue
		}

		for j := i + 1; j < n; j++ {
			entryB := entries[j]
			idB := getEntryIDFromFields(entryB)

			// Skip if already processed
			if processed[idB] {
				continue
			}

			tierB := GetTier(entryB)

			// Tier 1 entries are never compacted
			if tierB == 1 {
				continue
			}

			detection := DetectRelationship(entryA, entryB)

			switch detection.Relationship {
			case RelationshipExactDuplicate:
				if detection.ShouldMerge && CanAutoCompact(tierA) && CanAutoCompact(tierB) {
					if !opts.DryRun {
						kept, discarded := MergeEntries(entryA, entryB, time.Now().UTC())
						keptID := getEntryIDFromFields(kept)
						discardedID := getEntryIDFromFields(discarded)
						updatedEntries[keptID] = kept
						updatedEntries[discardedID] = discarded
						processed[discardedID] = true
						// Update entryA reference for subsequent comparisons
						entries[i] = kept
						entryA = kept
					}
					result.DuplicatesMerged++
					result.Details = append(result.Details, CompactionDetail{
						Action:    CompactionActionMerged,
						Kept:      idA,
						Discarded: idB,
						Reason:    "Exact duplicate",
					})
				} else if !CanAutoCompact(tierA) || !CanAutoCompact(tierB) {
					// Tier 2 entries: flag for review
					key := disputeKey(idA, idB)
					disputed[key] = []string{idA, idB}
				}

			case RelationshipNearDuplicate:
				if detection.ShouldMerge && CanAutoCompact(tierA) && CanAutoCompact(tierB) {
					if !opts.DryRun {
						kept, discarded := MergeEntries(entryA, entryB, time.Now().UTC())
						keptID := getEntryIDFromFields(kept)
						discardedID := getEntryIDFromFields(discarded)
						updatedEntries[keptID] = kept
						updatedEntries[discardedID] = discarded
						processed[discardedID] = true
						entries[i] = kept
						entryA = kept
					}
					result.NearDuplicatesMerged++
					result.Details = append(result.Details, CompactionDetail{
						Action:    CompactionActionMerged,
						Kept:      idA,
						Discarded: idB,
						Reason:    fmt.Sprintf("Near-duplicate (similarity: %.2f)", detection.Similarity),
					})
				} else if detection.ShouldFlag || !CanAutoCompact(tierA) || !CanAutoCompact(tierB) {
					// Flag for review (low confidence or tier 2)
					key := disputeKey(idA, idB)
					disputed[key] = []string{idA, idB}
				}

			case RelationshipContradiction:
				// Always flag contradictions
				key := disputeKey(idA, idB)
				disputed[key] = []string{idA, idB}
				if !opts.DryRun {
					// Mark both entries as disputed
					a := copyFields(entryA)
					b := copyFields(entryB)
					a["status"] = "disputed"
					b["status"] = "disputed"
					updatedEntries[idA] = a
					updatedEntries[idB] = b
				}
				result.ConflictsFlagged++
				result.Details = append(result.Details, CompactionDetail{
					Action:  CompactionActionDisputed,
					Entries: []string{idA, idB},
					Reason:  fmt.Sprintf("Contradictory content in same scope (similarity: %.2f)", detection.Similarity),
				})

			case RelationshipIndependent:
				// No action needed
			}
		}
	}

	// Collect all updated entries
	var updates []map[string]any
	for _, fields := range updatedEntries {
		updates = append(updates, fields)
	}

	// Sort updates by ID for deterministic output
	sort.Slice(updates, func(i, j int) bool {
		return getEntryIDFromFields(updates[i]) < getEntryIDFromFields(updates[j])
	})

	return result, updates
}

// disputeKey creates a deterministic key for a pair of entry IDs.
func disputeKey(a, b string) string {
	if a < b {
		return a + ":" + b
	}
	return b + ":" + a
}

// getEntryIDFromFields extracts the entry ID from fields, checking both "id" and "entry_id".
func getEntryIDFromFields(fields map[string]any) string {
	if id := getFieldString(fields, "id"); id != "" {
		return id
	}
	return GetEntryID(fields)
}

// GetScope extracts the scope from knowledge entry fields.
func GetScope(fields map[string]any) string {
	return getFieldString(fields, "scope")
}

// GetContent extracts the content from knowledge entry fields.
func GetContent(fields map[string]any) string {
	return getFieldString(fields, "content")
}

// GetGitAnchors extracts git_anchors as a string slice from fields.
func GetGitAnchors(fields map[string]any) []string {
	if fields == nil {
		return nil
	}

	raw, ok := fields["git_anchors"]
	if !ok || raw == nil {
		return nil
	}

	switch v := raw.(type) {
	case []string:
		return v
	case []any:
		var result []string
		for _, item := range v {
			if s, ok := item.(string); ok && s != "" {
				result = append(result, s)
			}
		}
		return result
	}
	return nil
}

// SetGitAnchors sets the git_anchors field in a knowledge entry.
func SetGitAnchors(fields map[string]any, anchors []string) {
	if fields == nil {
		return
	}
	if len(anchors) == 0 {
		delete(fields, "git_anchors")
		return
	}
	fields["git_anchors"] = anchors
}

// SetMergedFrom sets the merged_from field in a knowledge entry.
func SetMergedFrom(fields map[string]any, entryID string) {
	if fields == nil {
		return
	}
	if entryID == "" {
		return
	}
	fields["merged_from"] = entryID
}

// SetStatus sets the status field in a knowledge entry.
func SetStatus(fields map[string]any, status string) {
	if fields == nil {
		return
	}
	fields["status"] = status
}

// GetStatus extracts the status from knowledge entry fields.
func GetStatus(fields map[string]any) string {
	return getFieldString(fields, "status")
}
