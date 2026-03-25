package knowledge

import (
	"testing"
	"time"
)

func TestDefaultPromotionConfig(t *testing.T) {
	config := DefaultPromotionConfig()

	if config.MinUseCount != 5 {
		t.Errorf("MinUseCount = %d, want 5", config.MinUseCount)
	}
	if config.MaxMissCount != 0 {
		t.Errorf("MaxMissCount = %d, want 0", config.MaxMissCount)
	}
	if config.MinConfidence != 0.7 {
		t.Errorf("MinConfidence = %f, want 0.7", config.MinConfidence)
	}
}

func TestCheckPromotionEligibility(t *testing.T) {
	config := DefaultPromotionConfig()

	tests := []struct {
		name         string
		fields       map[string]any
		wantEligible bool
		wantReason   string
	}{
		{
			name: "all criteria met - eligible",
			fields: map[string]any{
				"tier":       3,
				"use_count":  5,
				"miss_count": 0,
				"confidence": 0.7,
			},
			wantEligible: true,
			wantReason:   "use_count=5, miss_count=0, confidence=0.70",
		},
		{
			name: "exceeds all thresholds - eligible",
			fields: map[string]any{
				"tier":       3,
				"use_count":  10,
				"miss_count": 0,
				"confidence": 0.95,
			},
			wantEligible: true,
			wantReason:   "use_count=10, miss_count=0, confidence=0.95",
		},
		{
			name: "tier 1 entry - ineligible",
			fields: map[string]any{
				"tier":       1,
				"use_count":  100,
				"miss_count": 0,
				"confidence": 1.0,
			},
			wantEligible: false,
			wantReason:   "tier 1 entries cannot be promoted (already highest tier)",
		},
		{
			name: "tier 2 entry - ineligible",
			fields: map[string]any{
				"tier":       2,
				"use_count":  50,
				"miss_count": 0,
				"confidence": 0.9,
			},
			wantEligible: false,
			wantReason:   "tier 2 entries are not eligible for promotion",
		},
		{
			name: "use_count too low",
			fields: map[string]any{
				"tier":       3,
				"use_count":  4,
				"miss_count": 0,
				"confidence": 0.8,
			},
			wantEligible: false,
			wantReason:   "use_count=4 is below minimum 5",
		},
		{
			name: "miss_count too high",
			fields: map[string]any{
				"tier":       3,
				"use_count":  10,
				"miss_count": 1,
				"confidence": 0.8,
			},
			wantEligible: false,
			wantReason:   "miss_count=1 exceeds maximum 0",
		},
		{
			name: "confidence too low",
			fields: map[string]any{
				"tier":       3,
				"use_count":  10,
				"miss_count": 0,
				"confidence": 0.69,
			},
			wantEligible: false,
			wantReason:   "confidence=0.69 is below minimum 0.70",
		},
		{
			name:         "missing all fields - uses defaults, tier defaults to 3",
			fields:       map[string]any{},
			wantEligible: false,
			wantReason:   "use_count=0 is below minimum 5",
		},
		{
			name: "missing tier - defaults to 3, becomes eligible",
			fields: map[string]any{
				"use_count":  7,
				"miss_count": 0,
				"confidence": 0.85,
			},
			wantEligible: true,
			wantReason:   "use_count=7, miss_count=0, confidence=0.85",
		},
		{
			name: "missing use_count - defaults to 0",
			fields: map[string]any{
				"tier":       3,
				"miss_count": 0,
				"confidence": 0.8,
			},
			wantEligible: false,
			wantReason:   "use_count=0 is below minimum 5",
		},
		{
			name: "missing miss_count - defaults to 0",
			fields: map[string]any{
				"tier":       3,
				"use_count":  5,
				"confidence": 0.8,
			},
			wantEligible: true,
			wantReason:   "use_count=5, miss_count=0, confidence=0.80",
		},
		{
			name: "missing confidence - defaults to 0.5",
			fields: map[string]any{
				"tier":       3,
				"use_count":  5,
				"miss_count": 0,
			},
			wantEligible: false,
			wantReason:   "confidence=0.50 is below minimum 0.70",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CheckPromotionEligibility(tt.fields, config)

			if got.Eligible != tt.wantEligible {
				t.Errorf("Eligible = %v, want %v", got.Eligible, tt.wantEligible)
			}
			if got.Reason != tt.wantReason {
				t.Errorf("Reason = %q, want %q", got.Reason, tt.wantReason)
			}
		})
	}
}

func TestCheckPromotionEligibility_BoundaryValues(t *testing.T) {
	config := DefaultPromotionConfig()

	tests := []struct {
		name         string
		fields       map[string]any
		wantEligible bool
	}{
		{
			name: "use_count exactly at boundary (5) - eligible",
			fields: map[string]any{
				"tier":       3,
				"use_count":  5,
				"miss_count": 0,
				"confidence": 0.7,
			},
			wantEligible: true,
		},
		{
			name: "use_count just below boundary (4) - ineligible",
			fields: map[string]any{
				"tier":       3,
				"use_count":  4,
				"miss_count": 0,
				"confidence": 0.7,
			},
			wantEligible: false,
		},
		{
			name: "confidence exactly at boundary (0.70) - eligible",
			fields: map[string]any{
				"tier":       3,
				"use_count":  5,
				"miss_count": 0,
				"confidence": 0.70,
			},
			wantEligible: true,
		},
		{
			name: "confidence just below boundary (0.699) - ineligible",
			fields: map[string]any{
				"tier":       3,
				"use_count":  5,
				"miss_count": 0,
				"confidence": 0.699,
			},
			wantEligible: false,
		},
		{
			name: "miss_count exactly at boundary (0) - eligible",
			fields: map[string]any{
				"tier":       3,
				"use_count":  5,
				"miss_count": 0,
				"confidence": 0.7,
			},
			wantEligible: true,
		},
		{
			name: "miss_count just above boundary (1) - ineligible",
			fields: map[string]any{
				"tier":       3,
				"use_count":  5,
				"miss_count": 1,
				"confidence": 0.7,
			},
			wantEligible: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CheckPromotionEligibility(tt.fields, config)

			if got.Eligible != tt.wantEligible {
				t.Errorf("Eligible = %v, want %v (reason: %s)", got.Eligible, tt.wantEligible, got.Reason)
			}
		})
	}
}

func TestCheckPromotionEligibility_CustomConfig(t *testing.T) {
	customConfig := PromotionConfig{
		MinUseCount:   3,
		MaxMissCount:  2,
		MinConfidence: 0.6,
	}

	tests := []struct {
		name         string
		fields       map[string]any
		wantEligible bool
	}{
		{
			name: "passes with custom lower thresholds",
			fields: map[string]any{
				"tier":       3,
				"use_count":  3,
				"miss_count": 2,
				"confidence": 0.6,
			},
			wantEligible: true,
		},
		{
			name: "fails default config but passes custom",
			fields: map[string]any{
				"tier":       3,
				"use_count":  4,
				"miss_count": 1,
				"confidence": 0.65,
			},
			wantEligible: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CheckPromotionEligibility(tt.fields, customConfig)

			if got.Eligible != tt.wantEligible {
				t.Errorf("Eligible = %v, want %v", got.Eligible, tt.wantEligible)
			}
		})
	}
}

func TestApplyPromotion(t *testing.T) {
	now := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	lastUsed := time.Date(2024, 1, 10, 8, 0, 0, 0, time.UTC)
	tier2TTLDays := 90

	fields := map[string]any{
		"entry_id":   "KE-001",
		"topic":      "Test Topic",
		"tier":       3,
		"use_count":  7,
		"miss_count": 0,
		"confidence": 0.85,
		"last_used":  lastUsed.Format(time.RFC3339),
		"ttl_days":   30,
	}

	result := ApplyPromotion(fields, "KE-001", now, tier2TTLDays)

	// Check tier updated
	if result["tier"] != 2 {
		t.Errorf("tier = %v, want 2", result["tier"])
	}

	// Check promoted_from set
	if result["promoted_from"] != "KE-001" {
		t.Errorf("promoted_from = %v, want KE-001", result["promoted_from"])
	}

	// Check promoted_at set
	if result["promoted_at"] != now.Format(time.RFC3339) {
		t.Errorf("promoted_at = %v, want %s", result["promoted_at"], now.Format(time.RFC3339))
	}

	// Check ttl_days updated
	if result["ttl_days"] != tier2TTLDays {
		t.Errorf("ttl_days = %v, want %d", result["ttl_days"], tier2TTLDays)
	}

	// Check ttl_expires_at recomputed from last_used + 90 days
	expectedExpires := lastUsed.AddDate(0, 0, tier2TTLDays).Format(time.RFC3339)
	if result["ttl_expires_at"] != expectedExpires {
		t.Errorf("ttl_expires_at = %v, want %s", result["ttl_expires_at"], expectedExpires)
	}

	// Check original fields preserved
	if result["topic"] != "Test Topic" {
		t.Errorf("topic = %v, want Test Topic", result["topic"])
	}
	if result["use_count"] != 7 {
		t.Errorf("use_count = %v, want 7", result["use_count"])
	}
}

func TestApplyPromotion_MissingLastUsed(t *testing.T) {
	now := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	tier2TTLDays := 90

	fields := map[string]any{
		"entry_id": "KE-002",
		"tier":     3,
	}

	result := ApplyPromotion(fields, "KE-002", now, tier2TTLDays)

	// When last_used is missing, should use 'now' as the base
	expectedExpires := now.AddDate(0, 0, tier2TTLDays).Format(time.RFC3339)
	if result["ttl_expires_at"] != expectedExpires {
		t.Errorf("ttl_expires_at = %v, want %s", result["ttl_expires_at"], expectedExpires)
	}
}

func TestApplyPromotion_DoesNotMutateInput(t *testing.T) {
	now := time.Now()
	fields := map[string]any{
		"entry_id": "KE-003",
		"tier":     3,
	}

	_ = ApplyPromotion(fields, "KE-003", now, 90)

	// Original should still be tier 3
	if fields["tier"] != 3 {
		t.Errorf("original tier = %v, want 3 (should not be mutated)", fields["tier"])
	}
	if _, exists := fields["promoted_from"]; exists {
		t.Error("original should not have promoted_from field")
	}
}

func TestFindPromotionCandidates(t *testing.T) {
	config := DefaultPromotionConfig()

	entries := []map[string]any{
		{
			"entry_id":   "KE-001",
			"topic":      "Eligible Entry 1",
			"tier":       3,
			"use_count":  7,
			"miss_count": 0,
			"confidence": 0.85,
		},
		{
			"entry_id":   "KE-002",
			"topic":      "Tier 1 Entry",
			"tier":       1,
			"use_count":  100,
			"miss_count": 0,
			"confidence": 1.0,
		},
		{
			"entry_id":   "KE-003",
			"topic":      "Tier 2 Entry",
			"tier":       2,
			"use_count":  50,
			"miss_count": 0,
			"confidence": 0.9,
		},
		{
			"entry_id":   "KE-004",
			"topic":      "Low Use Count",
			"tier":       3,
			"use_count":  2,
			"miss_count": 0,
			"confidence": 0.8,
		},
		{
			"entry_id":   "KE-005",
			"topic":      "Eligible Entry 2",
			"tier":       3,
			"use_count":  5,
			"miss_count": 0,
			"confidence": 0.7,
		},
		{
			"entry_id":   "KE-006",
			"topic":      "Has Misses",
			"tier":       3,
			"use_count":  10,
			"miss_count": 1,
			"confidence": 0.9,
		},
	}

	candidates := FindPromotionCandidates(entries, config)

	if len(candidates) != 2 {
		t.Fatalf("got %d candidates, want 2", len(candidates))
	}

	// Verify first candidate
	if candidates[0].EntryID != "KE-001" {
		t.Errorf("candidates[0].EntryID = %s, want KE-001", candidates[0].EntryID)
	}
	if candidates[0].Topic != "Eligible Entry 1" {
		t.Errorf("candidates[0].Topic = %s, want Eligible Entry 1", candidates[0].Topic)
	}
	if !candidates[0].Eligibility.Eligible {
		t.Error("candidates[0] should be eligible")
	}

	// Verify second candidate
	if candidates[1].EntryID != "KE-005" {
		t.Errorf("candidates[1].EntryID = %s, want KE-005", candidates[1].EntryID)
	}
}

func TestFindPromotionCandidates_EmptyInput(t *testing.T) {
	config := DefaultPromotionConfig()

	candidates := FindPromotionCandidates(nil, config)
	if len(candidates) != 0 {
		t.Errorf("got %d candidates for nil input, want 0", len(candidates))
	}

	candidates = FindPromotionCandidates([]map[string]any{}, config)
	if len(candidates) != 0 {
		t.Errorf("got %d candidates for empty input, want 0", len(candidates))
	}
}

func TestFindPromotionCandidates_AllTier1(t *testing.T) {
	config := DefaultPromotionConfig()

	entries := []map[string]any{
		{"entry_id": "KE-001", "tier": 1, "use_count": 100, "miss_count": 0, "confidence": 1.0},
		{"entry_id": "KE-002", "tier": 1, "use_count": 50, "miss_count": 0, "confidence": 0.9},
	}

	candidates := FindPromotionCandidates(entries, config)
	if len(candidates) != 0 {
		t.Errorf("got %d candidates for all tier 1 entries, want 0", len(candidates))
	}
}

func TestFindPromotionCandidates_MissingTierDefaultsToTier3(t *testing.T) {
	config := DefaultPromotionConfig()

	// Entry with missing tier should default to 3 and be eligible if criteria met
	entries := []map[string]any{
		{
			"entry_id":   "KE-001",
			"topic":      "No Tier Field",
			"use_count":  10,
			"miss_count": 0,
			"confidence": 0.9,
		},
	}

	candidates := FindPromotionCandidates(entries, config)
	if len(candidates) != 1 {
		t.Fatalf("got %d candidates, want 1 (missing tier should default to 3)", len(candidates))
	}
	if candidates[0].EntryID != "KE-001" {
		t.Errorf("candidates[0].EntryID = %s, want KE-001", candidates[0].EntryID)
	}
}

func TestGetMissCount(t *testing.T) {
	tests := []struct {
		name   string
		fields map[string]any
		want   int
	}{
		{
			name:   "miss_count present as int",
			fields: map[string]any{"miss_count": 3},
			want:   3,
		},
		{
			name:   "miss_count present as int64",
			fields: map[string]any{"miss_count": int64(5)},
			want:   5,
		},
		{
			name:   "miss_count present as float64",
			fields: map[string]any{"miss_count": float64(2)},
			want:   2,
		},
		{
			name:   "miss_count missing - defaults to 0",
			fields: map[string]any{},
			want:   0,
		},
		{
			name:   "miss_count is string - defaults to 0",
			fields: map[string]any{"miss_count": "three"},
			want:   0,
		},
		{
			name:   "nil fields",
			fields: nil,
			want:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetMissCount(tt.fields); got != tt.want {
				t.Errorf("GetMissCount() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestGetTopic(t *testing.T) {
	tests := []struct {
		name   string
		fields map[string]any
		want   string
	}{
		{
			name:   "topic present",
			fields: map[string]any{"topic": "Go error handling"},
			want:   "Go error handling",
		},
		{
			name:   "topic missing - defaults to empty",
			fields: map[string]any{},
			want:   "",
		},
		{
			name:   "topic is int - defaults to empty",
			fields: map[string]any{"topic": 123},
			want:   "",
		},
		{
			name:   "nil fields",
			fields: nil,
			want:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetTopic(tt.fields); got != tt.want {
				t.Errorf("GetTopic() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGetEntryID(t *testing.T) {
	tests := []struct {
		name   string
		fields map[string]any
		want   string
	}{
		{
			name:   "entry_id present",
			fields: map[string]any{"entry_id": "KE-001"},
			want:   "KE-001",
		},
		{
			name:   "entry_id missing - defaults to empty",
			fields: map[string]any{},
			want:   "",
		},
		{
			name:   "entry_id is int - defaults to empty",
			fields: map[string]any{"entry_id": 123},
			want:   "",
		},
		{
			name:   "nil fields",
			fields: nil,
			want:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetEntryID(tt.fields); got != tt.want {
				t.Errorf("GetEntryID() = %q, want %q", got, tt.want)
			}
		})
	}
}
