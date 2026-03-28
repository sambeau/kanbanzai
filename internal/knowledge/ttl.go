package knowledge

import (
	"time"
)

// TTLConfig holds TTL settings per tier.
type TTLConfig struct {
	Tier3Days       int // Default 30 days
	Tier2Days       int // Default 90 days
	GracePeriodDays int // Default 7 days - entries younger than this are exempt
}

// DefaultTTLConfig returns the spec defaults.
func DefaultTTLConfig() TTLConfig {
	return TTLConfig{
		Tier3Days:       30,
		Tier2Days:       90,
		GracePeriodDays: 7,
	}
}

// ComputeTTLExpiry calculates ttl_expires_at from last_used and ttl_days.
func ComputeTTLExpiry(lastUsed time.Time, ttlDays int) time.Time {
	return lastUsed.AddDate(0, 0, ttlDays)
}

// GetDefaultTTL returns the default TTL for a tier (30 for tier 3, 90 for tier 2, 0 for tier 1).
func GetDefaultTTL(tier int) int {
	switch tier {
	case 3:
		return 30
	case 2:
		return 90
	default:
		return 0
	}
}

// ResetTTL resets an entry's TTL to its tier default.
// Updates: last_used, ttl_days, ttl_expires_at
// Returns the updated fields map.
func ResetTTL(fields map[string]any, tier int, config TTLConfig, now time.Time) map[string]any {
	result := make(map[string]any)
	for k, v := range fields {
		result[k] = v
	}

	ttlDays := getTTLDaysFromConfig(tier, config)

	result["last_used"] = now.Format(time.RFC3339)
	result["ttl_days"] = ttlDays
	if ttlDays > 0 {
		expiresAt := ComputeTTLExpiry(now, ttlDays)
		result["ttl_expires_at"] = expiresAt.Format(time.RFC3339)
	}

	return result
}

// PruneCondition represents why an entry should be pruned.
type PruneCondition struct {
	ShouldPrune bool
	Reason      string
}

// CheckPruneCondition checks if an entry should be pruned based on TTL rules.
// Rules from spec:
// - Tier 3: TTL expired AND use_count < 3
// - Tier 2: TTL expired AND confidence < 0.5
// - Tier 1: Never (manual retirement only)
// - Entries younger than grace period: exempt
func CheckPruneCondition(fields map[string]any, now time.Time, config TTLConfig) PruneCondition {
	if fields == nil {
		return PruneCondition{ShouldPrune: false, Reason: "no fields"}
	}

	// Check status - don't prune already retired entries
	status := getFieldString(fields, "status")
	if status == "retired" {
		return PruneCondition{ShouldPrune: false, Reason: "already retired"}
	}

	tier := GetTier(fields)

	// Tier 1 entries are never auto-pruned
	if tier == 1 {
		return PruneCondition{ShouldPrune: false, Reason: "tier 1 entries require manual retirement"}
	}

	// Check grace period - entries younger than grace period are exempt
	createdAt := GetCreatedAt(fields)
	if !createdAt.IsZero() {
		gracePeriod := time.Duration(config.GracePeriodDays) * 24 * time.Hour
		if now.Sub(createdAt) < gracePeriod {
			return PruneCondition{ShouldPrune: false, Reason: "within grace period"}
		}
	}

	// Check TTL expiration
	ttlExpiresAt := GetTTLExpiresAt(fields)
	if ttlExpiresAt.IsZero() {
		// No TTL set - compute from last_used or created
		lastUsed := GetLastUsed(fields)
		if lastUsed.IsZero() {
			lastUsed = createdAt
		}
		if lastUsed.IsZero() {
			return PruneCondition{ShouldPrune: false, Reason: "no timestamp to compute TTL"}
		}

		ttlDays := GetTTLDays(fields)
		if ttlDays == 0 {
			ttlDays = getTTLDaysFromConfig(tier, config)
		}
		if ttlDays == 0 {
			return PruneCondition{ShouldPrune: false, Reason: "no TTL configured for tier"}
		}
		ttlExpiresAt = ComputeTTLExpiry(lastUsed, ttlDays)
	}

	// Check if TTL has expired
	if now.Before(ttlExpiresAt) {
		return PruneCondition{ShouldPrune: false, Reason: "TTL not expired"}
	}

	// TTL is expired - check tier-specific conditions
	switch tier {
	case 3:
		// Tier 3: TTL expired AND use_count < 3
		useCount := GetUseCount(fields)
		if useCount >= 3 {
			return PruneCondition{ShouldPrune: false, Reason: "tier 3 with use_count >= 3 is exempt"}
		}
		return PruneCondition{
			ShouldPrune: true,
			Reason:      "tier 3 TTL expired with use_count < 3",
		}

	case 2:
		// Tier 2: TTL expired AND confidence < 0.5
		confidence := GetConfidence(fields)
		if confidence >= 0.5 {
			return PruneCondition{ShouldPrune: false, Reason: "tier 2 with confidence >= 0.5 is exempt"}
		}
		return PruneCondition{
			ShouldPrune: true,
			Reason:      "tier 2 TTL expired with confidence < 0.5",
		}

	default:
		return PruneCondition{ShouldPrune: false, Reason: "unknown tier"}
	}
}

// getTTLDaysFromConfig returns the TTL in days for a given tier from config.
func getTTLDaysFromConfig(tier int, config TTLConfig) int {
	switch tier {
	case 3:
		return config.Tier3Days
	case 2:
		return config.Tier2Days
	default:
		return 0
	}
}

// GetTier extracts tier as int from fields.
// Defaults to 3 (lowest tier) if missing, matching new entry behavior.
func GetTier(fields map[string]any) int {
	if fields == nil {
		return 3
	}
	v, ok := fields["tier"]
	if !ok {
		return 3
	}
	switch typed := v.(type) {
	case int:
		return typed
	case int64:
		return int(typed)
	case float64:
		return int(typed)
	}
	return 3
}

// GetUseCount extracts use_count as int from fields.
func GetUseCount(fields map[string]any) int {
	return getFieldInt(fields, "use_count")
}

// GetConfidence extracts confidence as float64 from fields.
// Defaults to 0.5 (neutral Wilson score) if missing.
func GetConfidence(fields map[string]any) float64 {
	if fields == nil {
		return 0.5
	}
	v, ok := fields["confidence"]
	if !ok {
		return 0.5
	}
	switch typed := v.(type) {
	case float64:
		return typed
	case int:
		return float64(typed)
	case int64:
		return float64(typed)
	}
	return 0.5
}

// GetCreatedAt extracts created timestamp from fields.
func GetCreatedAt(fields map[string]any) time.Time {
	return getFieldTime(fields, "created")
}

// GetLastUsed extracts last_used timestamp from fields.
func GetLastUsed(fields map[string]any) time.Time {
	return getFieldTime(fields, "last_used")
}

// GetTTLDays extracts ttl_days as int from fields.
func GetTTLDays(fields map[string]any) int {
	return getFieldInt(fields, "ttl_days")
}

// GetTTLExpiresAt extracts ttl_expires_at timestamp from fields.
func GetTTLExpiresAt(fields map[string]any) time.Time {
	return getFieldTime(fields, "ttl_expires_at")
}

// getFieldInt reads an integer value from the fields map.
// Handles int, int64, float64 representations (from YAML round-trips).
func getFieldInt(fields map[string]any, key string) int {
	if fields == nil {
		return 0
	}
	v := fields[key]
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

// getFieldTime reads a time.Time value from the fields map.
// Expects RFC3339 string format.
func getFieldTime(fields map[string]any, key string) time.Time {
	if fields == nil {
		return time.Time{}
	}
	v := fields[key]
	str, ok := v.(string)
	if !ok || str == "" {
		return time.Time{}
	}
	t, err := time.Parse(time.RFC3339, str)
	if err != nil {
		return time.Time{}
	}
	return t
}

// getFieldString reads a string value from the fields map.
func getFieldString(fields map[string]any, key string) string {
	if fields == nil {
		return ""
	}
	v := fields[key]
	str, _ := v.(string)
	return str
}
