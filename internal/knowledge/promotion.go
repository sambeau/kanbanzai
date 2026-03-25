// Package knowledge provides knowledge entry lifecycle management.
package knowledge

import (
	"fmt"
	"time"
)

// PromotionConfig holds thresholds for automatic promotion.
type PromotionConfig struct {
	MinUseCount   int     // Default 5
	MaxMissCount  int     // Default 0
	MinConfidence float64 // Default 0.7
}

// DefaultPromotionConfig returns the spec defaults.
func DefaultPromotionConfig() PromotionConfig {
	return PromotionConfig{
		MinUseCount:   5,
		MaxMissCount:  0,
		MinConfidence: 0.7,
	}
}

// PromotionEligibility describes why an entry is/isn't eligible for promotion.
type PromotionEligibility struct {
	Eligible bool
	Reason   string // Explains eligibility or what's missing
}

// CheckPromotionEligibility checks if a Tier 3 entry should be promoted to Tier 2.
// Criteria from spec:
// - use_count >= 5 AND
// - miss_count = 0 AND
// - confidence >= 0.7
//
// Returns ineligible with reason if:
// - Entry is not Tier 3
// - Any criterion not met
func CheckPromotionEligibility(fields map[string]any, config PromotionConfig) PromotionEligibility {
	tier := getTierWithDefault(fields, 3)
	if tier == 1 {
		return PromotionEligibility{
			Eligible: false,
			Reason:   "tier 1 entries cannot be promoted (already highest tier)",
		}
	}
	if tier == 2 {
		return PromotionEligibility{
			Eligible: false,
			Reason:   "tier 2 entries are not eligible for promotion",
		}
	}
	if tier != 3 {
		return PromotionEligibility{
			Eligible: false,
			Reason:   fmt.Sprintf("unexpected tier %d; only tier 3 can be promoted", tier),
		}
	}

	useCount := GetUseCount(fields)
	missCount := GetMissCount(fields)
	confidence := getConfidenceWithDefault(fields, 0.5)

	// Check each criterion and report the first failure
	if useCount < config.MinUseCount {
		return PromotionEligibility{
			Eligible: false,
			Reason:   fmt.Sprintf("use_count=%d is below minimum %d", useCount, config.MinUseCount),
		}
	}

	if missCount > config.MaxMissCount {
		return PromotionEligibility{
			Eligible: false,
			Reason:   fmt.Sprintf("miss_count=%d exceeds maximum %d", missCount, config.MaxMissCount),
		}
	}

	if confidence < config.MinConfidence {
		return PromotionEligibility{
			Eligible: false,
			Reason:   fmt.Sprintf("confidence=%.2f is below minimum %.2f", confidence, config.MinConfidence),
		}
	}

	return PromotionEligibility{
		Eligible: true,
		Reason:   fmt.Sprintf("use_count=%d, miss_count=%d, confidence=%.2f", useCount, missCount, confidence),
	}
}

// PromotionResult contains the result of promoting an entry.
type PromotionResult struct {
	EntryID    string
	Topic      string
	FromTier   int
	ToTier     int
	PromotedAt time.Time
	Reason     string // e.g., "use_count=7, miss_count=0, confidence=0.85"
}

// ApplyPromotion updates the fields map to reflect promotion.
// Updates:
// - tier: 2
// - promoted_from: entry_id (self-reference for audit)
// - promoted_at: timestamp
// - ttl_days: tier2TTLDays
// - ttl_expires_at: recomputed from last_used + ttl_days
// Returns the updated fields map.
func ApplyPromotion(fields map[string]any, entryID string, now time.Time, tier2TTLDays int) map[string]any {
	// Create a copy to avoid mutating the input
	result := make(map[string]any, len(fields)+5)
	for k, v := range fields {
		result[k] = v
	}

	result["tier"] = 2
	result["promoted_from"] = entryID
	result["promoted_at"] = now.Format(time.RFC3339)
	result["ttl_days"] = tier2TTLDays

	// Compute ttl_expires_at from last_used + ttl_days
	lastUsed := GetLastUsed(fields)
	if lastUsed.IsZero() {
		lastUsed = now
	}
	expiresAt := lastUsed.AddDate(0, 0, tier2TTLDays)
	result["ttl_expires_at"] = expiresAt.Format(time.RFC3339)

	return result
}

// PromotionCandidate identifies an entry that could be promoted.
type PromotionCandidate struct {
	EntryID     string
	Topic       string
	UseCount    int
	MissCount   int
	Confidence  float64
	Eligibility PromotionEligibility
}

// FindPromotionCandidates scans entries for promotion-eligible Tier 3 entries.
func FindPromotionCandidates(entries []map[string]any, config PromotionConfig) []PromotionCandidate {
	var candidates []PromotionCandidate

	for _, entry := range entries {
		tier := getTierWithDefault(entry, 3)
		if tier != 3 {
			continue
		}

		eligibility := CheckPromotionEligibility(entry, config)
		if !eligibility.Eligible {
			continue
		}

		candidates = append(candidates, PromotionCandidate{
			EntryID:     GetEntryID(entry),
			Topic:       GetTopic(entry),
			UseCount:    GetUseCount(entry),
			MissCount:   GetMissCount(entry),
			Confidence:  getConfidenceWithDefault(entry, 0.5),
			Eligibility: eligibility,
		})
	}

	return candidates
}

// GetMissCount extracts miss_count from fields (default 0).
func GetMissCount(fields map[string]any) int {
	if fields == nil {
		return 0
	}
	v := fields["miss_count"]
	switch typed := v.(type) {
	case int:
		return typed
	case int64:
		return int(typed)
	case float64:
		return int(typed)
	}
	return 0
}

// GetTopic extracts topic from fields.
func GetTopic(fields map[string]any) string {
	if fields == nil {
		return ""
	}
	if s, ok := fields["topic"].(string); ok {
		return s
	}
	return ""
}

// GetEntryID extracts the entry ID from fields.
// Checks "id" first (canonical field name), then "entry_id" for backward compatibility.
func GetEntryID(fields map[string]any) string {
	if fields == nil {
		return ""
	}
	if s, ok := fields["id"].(string); ok {
		return s
	}
	// Fall back to entry_id for backward compatibility
	if s, ok := fields["entry_id"].(string); ok {
		return s
	}
	return ""
}

// getTierWithDefault extracts tier from fields with a specified default.
// This is used for promotion logic where missing tier should default to 3.
func getTierWithDefault(fields map[string]any, defaultVal int) int {
	if fields == nil {
		return defaultVal
	}
	v, ok := fields["tier"]
	if !ok {
		return defaultVal
	}
	switch typed := v.(type) {
	case int:
		return typed
	case int64:
		return int(typed)
	case float64:
		return int(typed)
	}
	return defaultVal
}

// getConfidenceWithDefault extracts confidence from fields with a specified default.
// This is used for promotion logic where missing confidence should default to 0.5.
func getConfidenceWithDefault(fields map[string]any, defaultVal float64) float64 {
	if fields == nil {
		return defaultVal
	}
	v, ok := fields["confidence"]
	if !ok {
		return defaultVal
	}
	switch typed := v.(type) {
	case float64:
		return typed
	case int:
		return float64(typed)
	case int64:
		return float64(typed)
	}
	return defaultVal
}
