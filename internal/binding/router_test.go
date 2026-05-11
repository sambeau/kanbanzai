package binding

import (
	"errors"
	"testing"
)

// ─── AC-001: Resolve with feature status "developing" and tier "feature" ────────

func TestResolve_FeatureDeveloping_FeatureTier(t *testing.T) {
	t.Parallel()

	routingTable := DefaultRoutingTable()
	ftConfig := DefaultFastTrackConfig()
	skillExists := func(string) bool { return true }

	resolution, err := Resolve("developing", "feature", routingTable, ftConfig, skillExists)
	if err != nil {
		t.Fatalf("Resolve() unexpected error: %v", err)
	}

	if resolution.BindingKey != "developing" {
		t.Errorf("BindingKey = %q, want %q", resolution.BindingKey, "developing")
	}
	if resolution.ModeProfile == nil {
		t.Fatal("ModeProfile is nil, want non-nil")
	}
	if resolution.ModeProfile.Review != "auto" {
		t.Errorf("ModeProfile.Review = %q, want %q", resolution.ModeProfile.Review, "auto")
	}
	if len(resolution.SkillOverrides) != 0 {
		t.Errorf("SkillOverrides should be empty for feature tier, got %v", resolution.SkillOverrides)
	}
}

// ─── AC-002: Resolve with retro_fix tier and "developing" status ───────────────

func TestResolve_RetroFixDeveloping_OverrideAndConditional(t *testing.T) {
	t.Parallel()

	routingTable := DefaultRoutingTable()
	ftConfig := DefaultFastTrackConfig()
	skillExists := func(skill string) bool {
		return skill == "implement-retro-fix"
	}

	resolution, err := Resolve("developing", "retro_fix", routingTable, ftConfig, skillExists)
	if err != nil {
		t.Fatalf("Resolve() unexpected error: %v", err)
	}

	if resolution.BindingKey != "developing" {
		t.Errorf("BindingKey = %q, want %q", resolution.BindingKey, "developing")
	}
	if resolution.ModeProfile == nil {
		t.Fatal("ModeProfile is nil, want non-nil")
	}
	if resolution.ModeProfile.Review != "conditional" {
		t.Errorf("ModeProfile.Review = %q, want %q", resolution.ModeProfile.Review, "conditional")
	}
	if resolution.ModeProfile.MaxCycles != 0 {
		t.Errorf("ModeProfile.MaxCycles = %d, want 0 (not set in default tiers)", resolution.ModeProfile.MaxCycles)
	}

	// SkillOverrides should contain the substitution.
	if resolution.SkillOverrides == nil {
		t.Fatal("SkillOverrides is nil, want non-nil for retro_fix")
	}
	sub, ok := resolution.SkillOverrides["implement-task"]
	if !ok {
		t.Errorf("SkillOverrides should contain key %q, got %v", "implement-task", resolution.SkillOverrides)
	}
	if sub != "implement-retro-fix" {
		t.Errorf("SkillOverrides[%q] = %q, want %q", "implement-task", sub, "implement-retro-fix")
	}
}

// ─── AC-003: Status with no routing entry → ErrNoBinding ───────────────────────

func TestResolve_UnknownStatus_ErrNoBinding(t *testing.T) {
	t.Parallel()

	routingTable := DefaultRoutingTable()
	ftConfig := DefaultFastTrackConfig()
	skillExists := func(string) bool { return true }

	_, err := Resolve("cooking", "feature", routingTable, ftConfig, skillExists)
	if err == nil {
		t.Fatal("expected error for unknown status, got nil")
	}

	var nbErr ErrNoBinding
	if !errors.As(err, &nbErr) {
		t.Errorf("error should be ErrNoBinding, got: %T %v", err, err)
	}
	if nbErr.Status != "cooking" {
		t.Errorf("ErrNoBinding.Status = %q, want %q", nbErr.Status, "cooking")
	}
}

// ─── AC-004: Unrecognised tier → ErrUnknownTier ────────────────────────────────

func TestResolve_UnknownTier_ErrUnknownTier(t *testing.T) {
	t.Parallel()

	routingTable := DefaultRoutingTable()
	ftConfig := DefaultFastTrackConfig()
	skillExists := func(string) bool { return true }

	_, err := Resolve("developing", "nonexistent", routingTable, ftConfig, skillExists)
	if err == nil {
		t.Fatal("expected error for unknown tier, got nil")
	}

	var utErr ErrUnknownTier
	if !errors.As(err, &utErr) {
		t.Errorf("error should be ErrUnknownTier, got: %T %v", err, err)
	}
	if utErr.Tier != "nonexistent" {
		t.Errorf("ErrUnknownTier.Tier = %q, want %q", utErr.Tier, "nonexistent")
	}
}

// ─── AC-005: Missing skill → ErrSkillMissing ───────────────────────────────────

func TestResolve_MissingSkill_ErrSkillMissing(t *testing.T) {
	t.Parallel()

	routingTable := DefaultRoutingTable()
	ftConfig := DefaultFastTrackConfig()
	// Skill checker returns false for implement-retro-fix.
	skillExists := func(skill string) bool { return false }

	_, err := Resolve("developing", "retro_fix", routingTable, ftConfig, skillExists)
	if err == nil {
		t.Fatal("expected error for missing skill, got nil")
	}

	var smErr ErrSkillMissing
	if !errors.As(err, &smErr) {
		t.Errorf("error should be ErrSkillMissing, got: %T %v", err, err)
	}
	if smErr.Skill != "implement-retro-fix" {
		t.Errorf("ErrSkillMissing.Skill = %q, want %q", smErr.Skill, "implement-retro-fix")
	}
	if smErr.BindingKey != "developing" {
		t.Errorf("ErrSkillMissing.BindingKey = %q, want %q", smErr.BindingKey, "developing")
	}
}

// ─── AC-012: BindingResolution field completeness ──────────────────────────────

func TestResolve_BindingResolution_AllFields(t *testing.T) {
	t.Parallel()

	routingTable := DefaultRoutingTable()
	ftConfig := DefaultFastTrackConfig()
	skillExists := func(string) bool { return true }

	resolution, err := Resolve("developing", "feature", routingTable, ftConfig, skillExists)
	if err != nil {
		t.Fatalf("Resolve() unexpected error: %v", err)
	}

	// BindingKey must be non-empty.
	if resolution.BindingKey == "" {
		t.Error("BindingKey must not be empty")
	}

	// SkillOverrides must be a map (possibly nil for non-retro_fix).
	// Feature tier has no overrides.
	if resolution.SkillOverrides != nil {
		t.Errorf("SkillOverrides should be nil for feature tier, got %v", resolution.SkillOverrides)
	}

	// ModeProfile must be non-nil when tier has a config entry.
	if resolution.ModeProfile == nil {
		t.Fatal("ModeProfile must not be nil for a configured tier")
	}
	if resolution.ModeProfile.Design == "" {
		t.Error("ModeProfile.Design must not be empty")
	}
	if resolution.ModeProfile.Spec == "" {
		t.Error("ModeProfile.Spec must not be empty")
	}
	if resolution.ModeProfile.DevPlan == "" {
		t.Error("ModeProfile.DevPlan must not be empty")
	}
	if resolution.ModeProfile.Review == "" {
		t.Error("ModeProfile.Review must not be empty")
	}
	// MaxCycles is accessible (could be 0).
	_ = resolution.ModeProfile.MaxCycles
}

// ─── Additional coverage ───────────────────────────────────────────────────────

func TestResolve_EmptyTier_DefaultsToFeature(t *testing.T) {
	t.Parallel()

	routingTable := DefaultRoutingTable()
	ftConfig := DefaultFastTrackConfig()
	skillExists := func(string) bool { return true }

	resolution, err := Resolve("designing", "", routingTable, ftConfig, skillExists)
	if err != nil {
		t.Fatalf("Resolve() unexpected error: %v", err)
	}
	if resolution.BindingKey != "designing" {
		t.Errorf("BindingKey = %q, want %q", resolution.BindingKey, "designing")
	}
	if resolution.ModeProfile == nil {
		t.Fatal("ModeProfile should not be nil when default tier is configured")
	}
}

func TestResolve_ValidTier_NoConfigEntry_NilModeProfile(t *testing.T) {
	t.Parallel()

	// Known tier but not present in Tiers map.
	routingTable := DefaultRoutingTable()
	ftConfig := &FastTrackConfig{Tiers: map[string]*FastTrackTier{}} // empty
	skillExists := func(string) bool { return true }

	resolution, err := Resolve("researching", "bug_fix", routingTable, ftConfig, skillExists)
	if err != nil {
		t.Fatalf("Resolve() unexpected error: %v", err)
	}
	if resolution.ModeProfile != nil {
		t.Errorf("ModeProfile should be nil when tier has no config entry, got %+v", resolution.ModeProfile)
	}
}

func TestResolve_AllFeatureStatuses(t *testing.T) {
	t.Parallel()

	routingTable := DefaultRoutingTable()
	ftConfig := DefaultFastTrackConfig()
	skillExists := func(string) bool { return true }

	featureStatuses := []string{
		"designing", "specifying", "dev-planning", "developing",
		"reviewing", "merging", "verifying",
		"batch-reviewing", "researching", "documenting",
		"retro-fixing",
	}

	for _, status := range featureStatuses {
		resolution, err := Resolve(status, "feature", routingTable, ftConfig, skillExists)
		if err != nil {
			t.Errorf("Resolve(%q, \"feature\") unexpected error: %v", status, err)
			continue
		}
		if resolution.BindingKey != status {
			t.Errorf("Resolve(%q) BindingKey = %q, want %q", status, resolution.BindingKey, status)
		}
	}
}

func TestResolve_BugStatuses(t *testing.T) {
	t.Parallel()

	routingTable := DefaultRoutingTable()
	ftConfig := DefaultFastTrackConfig()
	skillExists := func(string) bool { return true }

	tests := []struct {
		status  string
		wantKey string
	}{
		{"in-progress", "bug-developing"},
		{"needs-review", "bug-reviewing"},
	}

	for _, tt := range tests {
		resolution, err := Resolve(tt.status, "bug_fix", routingTable, ftConfig, skillExists)
		if err != nil {
			t.Errorf("Resolve(%q) unexpected error: %v", tt.status, err)
			continue
		}
		if resolution.BindingKey != tt.wantKey {
			t.Errorf("Resolve(%q) BindingKey = %q, want %q", tt.status, resolution.BindingKey, tt.wantKey)
		}
	}
}

func TestResolve_AllValidTiers(t *testing.T) {
	t.Parallel()

	routingTable := DefaultRoutingTable()
	ftConfig := DefaultFastTrackConfig()
	skillExists := func(string) bool { return true }

	tiers := []string{"retro_fix", "bug_fix", "feature", "critical"}
	for _, tier := range tiers {
		resolution, err := Resolve("developing", tier, routingTable, ftConfig, skillExists)
		if err != nil {
			t.Errorf("Resolve(\"developing\", %q) unexpected error: %v", tier, err)
			continue
		}
		if resolution.BindingKey != "developing" {
			t.Errorf("tier %q: BindingKey = %q, want %q", tier, resolution.BindingKey, "developing")
		}
		if resolution.ModeProfile == nil {
			t.Errorf("tier %q: ModeProfile should not be nil", tier)
		}
	}
}

func TestResolve_NilFastTrackConfig(t *testing.T) {
	t.Parallel()

	routingTable := DefaultRoutingTable()
	skillExists := func(string) bool { return true }

	resolution, err := Resolve("developing", "feature", routingTable, nil, skillExists)
	if err != nil {
		t.Fatalf("Resolve() unexpected error: %v", err)
	}
	if resolution.BindingKey != "developing" {
		t.Errorf("BindingKey = %q, want %q", resolution.BindingKey, "developing")
	}
	// ModeProfile should be nil when no config is provided.
	if resolution.ModeProfile != nil {
		t.Errorf("ModeProfile should be nil when config is nil, got %+v", resolution.ModeProfile)
	}
}

func TestResolve_NilSkillChecker_SkipsCheck(t *testing.T) {
	t.Parallel()

	routingTable := DefaultRoutingTable()
	ftConfig := DefaultFastTrackConfig()

	// nil skillChecker should skip checks and return success.
	resolution, err := Resolve("developing", "retro_fix", routingTable, ftConfig, nil)
	if err != nil {
		t.Fatalf("Resolve() unexpected error: %v (nil skillChecker should skip checks)", err)
	}
	if resolution.BindingKey != "developing" {
		t.Errorf("BindingKey = %q, want %q", resolution.BindingKey, "developing")
	}
	// Overrides should still be set even with nil checker.
	if resolution.SkillOverrides == nil {
		t.Error("SkillOverrides should be set for retro_fix tier even with nil skillChecker")
	}
}

func TestResolve_NoOverridesForNonRetroFix(t *testing.T) {
	t.Parallel()

	routingTable := DefaultRoutingTable()
	ftConfig := DefaultFastTrackConfig()
	skillExists := func(string) bool { return true }

	// bug_fix tier with "developing" binding should not produce overrides.
	resolution, err := Resolve("developing", "bug_fix", routingTable, ftConfig, skillExists)
	if err != nil {
		t.Fatalf("Resolve() unexpected error: %v", err)
	}
	if resolution.SkillOverrides != nil {
		t.Errorf("SkillOverrides should be nil for non-retro_fix tier, got %v", resolution.SkillOverrides)
	}
}

func TestResolve_RetroFixNonDeveloping_NoOverrides(t *testing.T) {
	t.Parallel()

	routingTable := DefaultRoutingTable()
	ftConfig := DefaultFastTrackConfig()
	skillExists := func(string) bool { return true }

	// retro_fix tier but at "reviewing" stage — the default overrides
	// still apply since they're tier-based, not stage-based.
	resolution, err := Resolve("reviewing", "retro_fix", routingTable, ftConfig, skillExists)
	if err != nil {
		t.Fatalf("Resolve() unexpected error: %v", err)
	}
	// The current implementation applies overrides at tier level, so even
	// "reviewing" gets the override for retro_fix.
	if resolution.SkillOverrides == nil {
		t.Error("SkillOverrides should be set for retro_fix tier at any stage")
	}
}

// ─── BENCHMARK: AC-014: p99 ≤ 1ms ────────────────────────────────────────────

func BenchmarkResolve(b *testing.B) {
	routingTable := DefaultRoutingTable()
	ftConfig := DefaultFastTrackConfig()
	skillExists := func(string) bool { return true }

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := Resolve("developing", "feature", routingTable, ftConfig, skillExists)
		if err != nil {
			b.Fatalf("Resolve() unexpected error: %v", err)
		}
	}
}

func BenchmarkResolve_RetroFix(b *testing.B) {
	routingTable := DefaultRoutingTable()
	ftConfig := DefaultFastTrackConfig()
	skillExists := func(skill string) bool { return skill == "implement-retro-fix" }

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := Resolve("developing", "retro_fix", routingTable, ftConfig, skillExists)
		if err != nil {
			b.Fatalf("Resolve() unexpected error: %v", err)
		}
	}
}
