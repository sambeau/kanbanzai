package knowledge

import (
	"testing"
	"time"
)

func TestDefaultTTLConfig(t *testing.T) {
	config := DefaultTTLConfig()

	if config.Tier3Days != 30 {
		t.Errorf("Tier3Days = %d, want 30", config.Tier3Days)
	}
	if config.Tier2Days != 90 {
		t.Errorf("Tier2Days = %d, want 90", config.Tier2Days)
	}
	if config.GracePeriodDays != 7 {
		t.Errorf("GracePeriodDays = %d, want 7", config.GracePeriodDays)
	}
}

func TestGetDefaultTTL(t *testing.T) {
	tests := []struct {
		tier int
		want int
	}{
		{tier: 3, want: 30},
		{tier: 2, want: 90},
		{tier: 1, want: 0},
		{tier: 0, want: 0},
		{tier: 4, want: 0},
	}

	for _, tt := range tests {
		got := GetDefaultTTL(tt.tier)
		if got != tt.want {
			t.Errorf("GetDefaultTTL(%d) = %d, want %d", tt.tier, got, tt.want)
		}
	}
}

func TestComputeTTLExpiry(t *testing.T) {
	base := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name     string
		lastUsed time.Time
		ttlDays  int
		want     time.Time
	}{
		{
			name:     "30 days",
			lastUsed: base,
			ttlDays:  30,
			want:     time.Date(2024, 2, 14, 12, 0, 0, 0, time.UTC),
		},
		{
			name:     "90 days",
			lastUsed: base,
			ttlDays:  90,
			want:     time.Date(2024, 4, 14, 12, 0, 0, 0, time.UTC),
		},
		{
			name:     "0 days",
			lastUsed: base,
			ttlDays:  0,
			want:     base,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ComputeTTLExpiry(tt.lastUsed, tt.ttlDays)
			if !got.Equal(tt.want) {
				t.Errorf("ComputeTTLExpiry() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestResetTTL(t *testing.T) {
	config := DefaultTTLConfig()

	t.Run("tier 3 reset", func(t *testing.T) {
		fields := map[string]any{
			"id":   "KE-123",
			"tier": 3,
		}

		now := time.Now().UTC()
		result := ResetTTL(fields, 3, config, now)

		if result["ttl_days"] != 30 {
			t.Errorf("ttl_days = %v, want 30", result["ttl_days"])
		}

		lastUsed, ok := result["last_used"].(string)
		if !ok || lastUsed == "" {
			t.Error("last_used should be set")
		}

		ttlExpiresAt, ok := result["ttl_expires_at"].(string)
		if !ok || ttlExpiresAt == "" {
			t.Error("ttl_expires_at should be set")
		}
	})

	t.Run("tier 2 reset", func(t *testing.T) {
		fields := map[string]any{
			"id":   "KE-456",
			"tier": 2,
		}

		result := ResetTTL(fields, 2, config, time.Now().UTC())

		if result["ttl_days"] != 90 {
			t.Errorf("ttl_days = %v, want 90", result["ttl_days"])
		}
	})

	t.Run("tier 1 reset", func(t *testing.T) {
		fields := map[string]any{
			"id":   "KE-789",
			"tier": 1,
		}

		result := ResetTTL(fields, 1, config, time.Now().UTC())

		if result["ttl_days"] != 0 {
			t.Errorf("ttl_days = %v, want 0", result["ttl_days"])
		}

		// ttl_expires_at should not be set for tier 1
		if _, ok := result["ttl_expires_at"]; ok {
			t.Error("ttl_expires_at should not be set for tier 1")
		}
	})

	t.Run("nil fields creates new map", func(t *testing.T) {
		result := ResetTTL(nil, 3, config, time.Now().UTC())

		if result == nil {
			t.Error("result should not be nil")
		}
		if result["ttl_days"] != 30 {
			t.Errorf("ttl_days = %v, want 30", result["ttl_days"])
		}
	})
}

func TestCheckPruneCondition_Tier3(t *testing.T) {
	config := DefaultTTLConfig()
	now := time.Date(2024, 3, 1, 12, 0, 0, 0, time.UTC)
	expiredDate := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC) // Well before now
	notExpiredDate := time.Date(2024, 3, 1, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name        string
		fields      map[string]any
		shouldPrune bool
		reasonHint  string
	}{
		{
			name: "tier 3 expired with low use count",
			fields: map[string]any{
				"tier":           3,
				"use_count":      2,
				"ttl_expires_at": expiredDate.Format(time.RFC3339),
				"created":        expiredDate.Add(-30 * 24 * time.Hour).Format(time.RFC3339),
			},
			shouldPrune: true,
			reasonHint:  "tier 3 TTL expired",
		},
		{
			name: "tier 3 expired but high use count exempt",
			fields: map[string]any{
				"tier":           3,
				"use_count":      3,
				"ttl_expires_at": expiredDate.Format(time.RFC3339),
				"created":        expiredDate.Add(-30 * 24 * time.Hour).Format(time.RFC3339),
			},
			shouldPrune: false,
			reasonHint:  "exempt",
		},
		{
			name: "tier 3 not expired",
			fields: map[string]any{
				"tier":           3,
				"use_count":      1,
				"ttl_expires_at": notExpiredDate.Add(24 * time.Hour).Format(time.RFC3339),
				"created":        expiredDate.Add(-10 * 24 * time.Hour).Format(time.RFC3339),
			},
			shouldPrune: false,
			reasonHint:  "not expired",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CheckPruneCondition(tt.fields, now, config)
			if result.ShouldPrune != tt.shouldPrune {
				t.Errorf("ShouldPrune = %v, want %v (reason: %s)", result.ShouldPrune, tt.shouldPrune, result.Reason)
			}
		})
	}
}

func TestCheckPruneCondition_Tier2(t *testing.T) {
	config := DefaultTTLConfig()
	now := time.Date(2024, 3, 1, 12, 0, 0, 0, time.UTC)
	expiredDate := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name        string
		fields      map[string]any
		shouldPrune bool
	}{
		{
			name: "tier 2 expired with low confidence",
			fields: map[string]any{
				"tier":           2,
				"confidence":     0.4,
				"ttl_expires_at": expiredDate.Format(time.RFC3339),
				"created":        expiredDate.Add(-90 * 24 * time.Hour).Format(time.RFC3339),
			},
			shouldPrune: true,
		},
		{
			name: "tier 2 expired but high confidence exempt",
			fields: map[string]any{
				"tier":           2,
				"confidence":     0.5,
				"ttl_expires_at": expiredDate.Format(time.RFC3339),
				"created":        expiredDate.Add(-90 * 24 * time.Hour).Format(time.RFC3339),
			},
			shouldPrune: false,
		},
		{
			name: "tier 2 expired with exactly 0.5 confidence is exempt",
			fields: map[string]any{
				"tier":           2,
				"confidence":     0.5,
				"ttl_expires_at": expiredDate.Format(time.RFC3339),
				"created":        expiredDate.Add(-90 * 24 * time.Hour).Format(time.RFC3339),
			},
			shouldPrune: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CheckPruneCondition(tt.fields, now, config)
			if result.ShouldPrune != tt.shouldPrune {
				t.Errorf("ShouldPrune = %v, want %v (reason: %s)", result.ShouldPrune, tt.shouldPrune, result.Reason)
			}
		})
	}
}

func TestCheckPruneCondition_Tier1(t *testing.T) {
	config := DefaultTTLConfig()
	now := time.Date(2024, 3, 1, 12, 0, 0, 0, time.UTC)
	expiredDate := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

	fields := map[string]any{
		"tier":           1,
		"use_count":      0,
		"confidence":     0.1,
		"ttl_expires_at": expiredDate.Format(time.RFC3339),
		"created":        expiredDate.Add(-365 * 24 * time.Hour).Format(time.RFC3339),
	}

	result := CheckPruneCondition(fields, now, config)
	if result.ShouldPrune {
		t.Errorf("Tier 1 should never be auto-pruned, got ShouldPrune=true")
	}
	if result.Reason != "tier 1 entries require manual retirement" {
		t.Errorf("unexpected reason: %s", result.Reason)
	}
}

func TestCheckPruneCondition_GracePeriod(t *testing.T) {
	config := DefaultTTLConfig()
	now := time.Date(2024, 3, 1, 12, 0, 0, 0, time.UTC)

	// Entry created 3 days ago (within 7-day grace period)
	recentCreated := now.Add(-3 * 24 * time.Hour)
	expiredDate := now.Add(-1 * 24 * time.Hour)

	fields := map[string]any{
		"tier":           3,
		"use_count":      0,
		"ttl_expires_at": expiredDate.Format(time.RFC3339),
		"created":        recentCreated.Format(time.RFC3339),
	}

	result := CheckPruneCondition(fields, now, config)
	if result.ShouldPrune {
		t.Errorf("Entry within grace period should not be pruned")
	}
	if result.Reason != "within grace period" {
		t.Errorf("expected grace period reason, got: %s", result.Reason)
	}
}

func TestCheckPruneCondition_AlreadyRetired(t *testing.T) {
	config := DefaultTTLConfig()
	now := time.Date(2024, 3, 1, 12, 0, 0, 0, time.UTC)
	expiredDate := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

	fields := map[string]any{
		"tier":           3,
		"status":         "retired",
		"use_count":      0,
		"ttl_expires_at": expiredDate.Format(time.RFC3339),
		"created":        expiredDate.Add(-30 * 24 * time.Hour).Format(time.RFC3339),
	}

	result := CheckPruneCondition(fields, now, config)
	if result.ShouldPrune {
		t.Errorf("Already retired entry should not be pruned again")
	}
}

func TestCheckPruneCondition_MissingFields(t *testing.T) {
	config := DefaultTTLConfig()
	now := time.Date(2024, 3, 1, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name        string
		fields      map[string]any
		shouldPrune bool
	}{
		{
			name:        "nil fields",
			fields:      nil,
			shouldPrune: false,
		},
		{
			name:        "empty fields",
			fields:      map[string]any{},
			shouldPrune: false,
		},
		{
			name: "missing tier defaults to 0",
			fields: map[string]any{
				"use_count": 0,
			},
			shouldPrune: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CheckPruneCondition(tt.fields, now, config)
			if result.ShouldPrune != tt.shouldPrune {
				t.Errorf("ShouldPrune = %v, want %v", result.ShouldPrune, tt.shouldPrune)
			}
		})
	}
}

func TestCheckPruneCondition_ComputesTTLFromLastUsed(t *testing.T) {
	config := DefaultTTLConfig()
	now := time.Date(2024, 3, 1, 12, 0, 0, 0, time.UTC)

	// last_used 60 days ago, with 30 day TTL = expired
	lastUsed := now.Add(-60 * 24 * time.Hour)
	created := now.Add(-90 * 24 * time.Hour)

	fields := map[string]any{
		"tier":      3,
		"use_count": 1,
		"last_used": lastUsed.Format(time.RFC3339),
		"created":   created.Format(time.RFC3339),
		"ttl_days":  30,
		// Note: no ttl_expires_at - should be computed
	}

	result := CheckPruneCondition(fields, now, config)
	if !result.ShouldPrune {
		t.Errorf("Entry with computed TTL expiry should be pruned, reason: %s", result.Reason)
	}
}

func TestFieldAccessors(t *testing.T) {
	t.Run("GetTier", func(t *testing.T) {
		if got := GetTier(map[string]any{"tier": 3}); got != 3 {
			t.Errorf("GetTier with int = %d, want 3", got)
		}
		if got := GetTier(map[string]any{"tier": float64(2)}); got != 2 {
			t.Errorf("GetTier with float64 = %d, want 2", got)
		}
		// Missing tier defaults to 3 (lowest tier, new entry default)
		if got := GetTier(map[string]any{}); got != 3 {
			t.Errorf("GetTier missing = %d, want 3", got)
		}
		if got := GetTier(nil); got != 3 {
			t.Errorf("GetTier nil = %d, want 3", got)
		}
	})

	t.Run("GetUseCount", func(t *testing.T) {
		if got := GetUseCount(map[string]any{"use_count": 5}); got != 5 {
			t.Errorf("GetUseCount = %d, want 5", got)
		}
		if got := GetUseCount(map[string]any{"use_count": int64(7)}); got != 7 {
			t.Errorf("GetUseCount int64 = %d, want 7", got)
		}
	})

	t.Run("GetConfidence", func(t *testing.T) {
		if got := GetConfidence(map[string]any{"confidence": 0.75}); got != 0.75 {
			t.Errorf("GetConfidence = %f, want 0.75", got)
		}
		if got := GetConfidence(map[string]any{"confidence": 1}); got != 1.0 {
			t.Errorf("GetConfidence int = %f, want 1.0", got)
		}
		// Missing confidence defaults to 0.5 (neutral Wilson score)
		if got := GetConfidence(map[string]any{}); got != 0.5 {
			t.Errorf("GetConfidence missing = %f, want 0.5", got)
		}
		if got := GetConfidence(nil); got != 0.5 {
			t.Errorf("GetConfidence nil = %f, want 0.5", got)
		}
	})

	t.Run("GetCreatedAt", func(t *testing.T) {
		ts := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)
		fields := map[string]any{"created": ts.Format(time.RFC3339)}
		got := GetCreatedAt(fields)
		if !got.Equal(ts) {
			t.Errorf("GetCreatedAt = %v, want %v", got, ts)
		}

		// Invalid time
		if got := GetCreatedAt(map[string]any{"created": "invalid"}); !got.IsZero() {
			t.Errorf("GetCreatedAt invalid = %v, want zero", got)
		}
	})

	t.Run("GetLastUsed", func(t *testing.T) {
		ts := time.Date(2024, 2, 20, 10, 30, 0, 0, time.UTC)
		fields := map[string]any{"last_used": ts.Format(time.RFC3339)}
		got := GetLastUsed(fields)
		if !got.Equal(ts) {
			t.Errorf("GetLastUsed = %v, want %v", got, ts)
		}
	})

	t.Run("GetTTLDays", func(t *testing.T) {
		if got := GetTTLDays(map[string]any{"ttl_days": 45}); got != 45 {
			t.Errorf("GetTTLDays = %d, want 45", got)
		}
	})

	t.Run("GetTTLExpiresAt", func(t *testing.T) {
		ts := time.Date(2024, 4, 15, 12, 0, 0, 0, time.UTC)
		fields := map[string]any{"ttl_expires_at": ts.Format(time.RFC3339)}
		got := GetTTLExpiresAt(fields)
		if !got.Equal(ts) {
			t.Errorf("GetTTLExpiresAt = %v, want %v", got, ts)
		}
	})
}
