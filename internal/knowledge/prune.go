package knowledge

import (
	"time"
)

// PruneResult contains the result of a pruning operation.
type PruneResult struct {
	EntryID string
	Topic   string
	Tier    int
	Reason  string
}

// PruneOptions configures pruning behavior.
type PruneOptions struct {
	DryRun bool // If true, don't actually modify entries
	Tier   int  // If non-zero, only prune entries of this tier
}

// PruneExpiredEntries finds and returns entries that should be pruned based on TTL rules.
// entries: the knowledge records to check (each as a map with fields)
// Returns list of entries to prune (or would-prune if dry-run).
//
// Note: This function only identifies entries to prune and returns the results.
// The actual status change to "retired" should be performed by the caller
// (typically the service layer) based on the returned results.
func PruneExpiredEntries(entries []map[string]any, now time.Time, config TTLConfig, opts PruneOptions) []PruneResult {
	var results []PruneResult

	for _, fields := range entries {
		if fields == nil {
			continue
		}

		// Skip if tier filter is set and doesn't match
		tier := GetTier(fields)
		if opts.Tier != 0 && tier != opts.Tier {
			continue
		}

		// Check if entry should be pruned
		condition := CheckPruneCondition(fields, now, config)
		if !condition.ShouldPrune {
			continue
		}

		// Extract entry info for result
		entryID := getFieldString(fields, "id")
		topic := getFieldString(fields, "topic")

		results = append(results, PruneResult{
			EntryID: entryID,
			Topic:   topic,
			Tier:    tier,
			Reason:  condition.Reason,
		})
	}

	return results
}

// ExtractFields extracts the fields map from a knowledge record structure.
// This is a helper for adapting storage.KnowledgeRecord to map[string]any.
// The caller should pass record.Fields directly.
func ExtractFields(fields map[string]any) map[string]any {
	return fields
}

// CollectFieldsFromRecords converts a slice of records (with Fields maps) to a slice of field maps.
// This is useful when the caller has []storage.KnowledgeRecord and needs []map[string]any.
func CollectFieldsFromRecords[T interface{ GetFields() map[string]any }](records []T) []map[string]any {
	result := make([]map[string]any, len(records))
	for i, rec := range records {
		result[i] = rec.GetFields()
	}
	return result
}

// PruneStats contains aggregate statistics about a pruning operation.
type PruneStats struct {
	TotalChecked int
	TotalPruned  int
	Tier2Pruned  int
	Tier3Pruned  int
}

// ComputeStats calculates aggregate statistics from pruning results.
func ComputeStats(results []PruneResult, totalChecked int) PruneStats {
	stats := PruneStats{
		TotalChecked: totalChecked,
		TotalPruned:  len(results),
	}

	for _, r := range results {
		switch r.Tier {
		case 2:
			stats.Tier2Pruned++
		case 3:
			stats.Tier3Pruned++
		}
	}

	return stats
}
