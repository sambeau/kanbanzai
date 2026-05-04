package config

import (
	"testing"

	"gopkg.in/yaml.v3"
)

func TestDefaultFastTrackConfig(t *testing.T) {
	t.Parallel()
	cfg := DefaultFastTrackConfig()

	if !cfg.IsEnabled() {
		t.Error("DefaultFastTrackConfig: enabled should be true")
	}
	if cfg.DefaultTier != TierFeature {
		t.Errorf("DefaultFastTrackConfig: default_tier = %q, want %q", cfg.DefaultTier, TierFeature)
	}

	// Verify all four tiers exist per REQ-TIER-001.
	for _, tier := range validTiers {
		if _, ok := cfg.Tiers[tier]; !ok {
			t.Errorf("DefaultFastTrackConfig: tiers[%q] missing", tier)
		}
	}

	// Verify tier matrix per REQ-TIER-002.
	tests := []struct {
		tier    string
		design  string
		spec    string
		devPlan string
		review  string
	}{
		{TierRetroFix, GateModeAuto, GateModeAuto, GateModeAuto, GateModeConditional},
		{TierBugFix, GateModeAuto, GateModeHuman, GateModeAuto, GateModeAuto},
		{TierFeature, GateModeHuman, GateModeAuto, GateModeAuto, GateModeAuto},
		{TierCritical, GateModeHuman, GateModeHuman, GateModeHuman, GateModeHuman},
	}

	for _, tc := range tests {
		t.Run(tc.tier, func(t *testing.T) {
			got := cfg.Tiers[tc.tier]
			if got.Design != tc.design {
				t.Errorf("design = %q, want %q", got.Design, tc.design)
			}
			if got.Spec != tc.spec {
				t.Errorf("spec = %q, want %q", got.Spec, tc.spec)
			}
			if got.DevPlan != tc.devPlan {
				t.Errorf("dev-plan = %q, want %q", got.DevPlan, tc.devPlan)
			}
			if got.Review != tc.review {
				t.Errorf("review = %q, want %q", got.Review, tc.review)
			}
		})
	}

	// Verify max_cycles per REQ-TIER-003.
	cycleTests := []struct {
		tier      string
		maxCycles int
	}{
		{TierRetroFix, 3},
		{TierBugFix, 2},
		{TierFeature, 2},
		{TierCritical, 0},
	}

	for _, tc := range cycleTests {
		t.Run(tc.tier+"_cycles", func(t *testing.T) {
			got := cfg.Tiers[tc.tier].MaxCycles
			if got != tc.maxCycles {
				t.Errorf("max_cycles = %d, want %d", got, tc.maxCycles)
			}
		})
	}
}

func TestFastTrackConfig_Validate(t *testing.T) {
	t.Parallel()
	validCfg := DefaultFastTrackConfig()
	if err := validCfg.Validate(); err != nil {
		t.Fatalf("DefaultFastTrackConfig should be valid, got: %v", err)
	}

	tests := []struct {
		name    string
		modify  func(*FastTrackConfig)
		wantErr string
	}{
		{
			name: "invalid default_tier",
			modify: func(c *FastTrackConfig) {
				c.DefaultTier = "unknown"
			},
			wantErr: `fast_track.default_tier "unknown" is not a valid tier name`,
		},
		{
			name: "invalid tier key",
			modify: func(c *FastTrackConfig) {
				c.Tiers["garbage"] = TierConfig{
					Design:    GateModeAuto,
					Spec:      GateModeAuto,
					DevPlan:   GateModeAuto,
					Review:    GateModeAuto,
					MaxCycles: 1,
				}
			},
			wantErr: `is not a valid tier name`,
		},
		{
			name: "invalid gate mode — design",
			modify: func(c *FastTrackConfig) {
				tc := c.Tiers[TierFeature]
				tc.Design = "invalid_mode"
				c.Tiers[TierFeature] = tc
			},
			wantErr: `is not a valid gate mode`,
		},
		{
			name: "invalid gate mode — spec",
			modify: func(c *FastTrackConfig) {
				tc := c.Tiers[TierFeature]
				tc.Spec = "invalid_mode"
				c.Tiers[TierFeature] = tc
			},
			wantErr: `is not a valid gate mode`,
		},
		{
			name: "invalid gate mode — dev-plan",
			modify: func(c *FastTrackConfig) {
				tc := c.Tiers[TierFeature]
				tc.DevPlan = "invalid_mode"
				c.Tiers[TierFeature] = tc
			},
			wantErr: `is not a valid gate mode`,
		},
		{
			name: "invalid gate mode — review",
			modify: func(c *FastTrackConfig) {
				tc := c.Tiers[TierFeature]
				tc.Review = "invalid_mode"
				c.Tiers[TierFeature] = tc
			},
			wantErr: `is not a valid gate mode`,
		},
		{
			name: "negative max_cycles",
			modify: func(c *FastTrackConfig) {
				tc := c.Tiers[TierFeature]
				tc.MaxCycles = -1
				c.Tiers[TierFeature] = tc
			},
			wantErr: `max_cycles must be >= 0`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultFastTrackConfig()
			tt.modify(&cfg)
			err := cfg.Validate()
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !contains(err.Error(), tt.wantErr) {
				t.Errorf("error %q does not contain %q", err.Error(), tt.wantErr)
			}
		})
	}

	// Edge case: empty default_tier is valid (will be merged from defaults).
	t.Run("empty default_tier ok", func(t *testing.T) {
		cfg := DefaultFastTrackConfig()
		cfg.DefaultTier = ""
		if err := cfg.Validate(); err != nil {
			t.Errorf("empty default_tier should be valid: %v", err)
		}
	})

	// Edge case: empty tiers map is valid (will be merged from defaults).
	t.Run("empty tiers ok", func(t *testing.T) {
		cfg := DefaultFastTrackConfig()
		cfg.Tiers = nil
		if err := cfg.Validate(); err != nil {
			t.Errorf("empty tiers should be valid: %v", err)
		}
	})

	// Edge case: enabled=false is valid.
	t.Run("disabled ok", func(t *testing.T) {
		cfg := DefaultFastTrackConfig()
		cfg.Enabled = false
		if err := cfg.Validate(); err != nil {
			t.Errorf("disabled config should be valid: %v", err)
		}
	})
}

func TestFastTrackConfig_YAMLRoundTrip(t *testing.T) {
	t.Parallel()
	// Verify that a complete fast_track config survives YAML marshal → unmarshal.
	yamlInput := `
enabled: true
default_tier: feature
tiers:
  retro_fix:
    design: auto
    spec: auto
    dev-plan: auto
    review: conditional
    max_cycles: 3
  bug_fix:
    design: auto
    spec: human
    dev-plan: auto
    review: auto
    max_cycles: 2
  feature:
    design: human
    spec: auto
    dev-plan: auto
    review: auto
    max_cycles: 2
  critical:
    design: human
    spec: human
    dev-plan: human
    review: human
    max_cycles: 0
`

	var cfg FastTrackConfig
	if err := yaml.Unmarshal([]byte(yamlInput), &cfg); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if !cfg.IsEnabled() {
		t.Error("enabled should be true")
	}
	if cfg.DefaultTier != TierFeature {
		t.Errorf("default_tier = %q, want %q", cfg.DefaultTier, TierFeature)
	}
	if len(cfg.Tiers) != 4 {
		t.Fatalf("tiers count = %d, want 4", len(cfg.Tiers))
	}

	// Spot-check retro_fix
	rf := cfg.Tiers[TierRetroFix]
	if rf.Design != GateModeAuto || rf.Spec != GateModeAuto || rf.DevPlan != GateModeAuto || rf.Review != GateModeConditional || rf.MaxCycles != 3 {
		t.Errorf("retro_fix mismatch: %+v", rf)
	}

	// Spot-check critical
	cr := cfg.Tiers[TierCritical]
	if cr.Design != GateModeHuman || cr.Spec != GateModeHuman || cr.DevPlan != GateModeHuman || cr.Review != GateModeHuman || cr.MaxCycles != 0 {
		t.Errorf("critical mismatch: %+v", cr)
	}

	if err := cfg.Validate(); err != nil {
		t.Fatalf("round-tripped config should be valid: %v", err)
	}
}

func TestFastTrackConfig_MergePhase4bDefaults(t *testing.T) {
	t.Parallel()
	t.Run("empty config merges defaults", func(t *testing.T) {
		var cfg Config
		cfg.mergePhase4bDefaults()

		if !cfg.FastTrack.IsEnabled() {
			t.Error("FastTrack.Enabled should be true after merge")
		}
		if cfg.FastTrack.DefaultTier != TierFeature {
			t.Errorf("FastTrack.DefaultTier = %q, want %q", cfg.FastTrack.DefaultTier, TierFeature)
		}
		if len(cfg.FastTrack.Tiers) != 4 {
			t.Errorf("FastTrack.Tiers count = %d, want 4", len(cfg.FastTrack.Tiers))
		}
	})

	t.Run("partial config preserves explicit values", func(t *testing.T) {
		cfg := Config{
			FastTrack: FastTrackConfig{
				Enabled:     false,
				DefaultTier: TierCritical,
				Tiers: map[string]TierConfig{
					TierRetroFix: {
						Design:    GateModeAuto,
						Spec:      GateModeAuto,
						DevPlan:   GateModeAuto,
						Review:    GateModeConditional,
						MaxCycles: 5,
					},
				},
			},
		}
		cfg.mergePhase4bDefaults()

		if cfg.FastTrack.IsEnabled() {
			t.Error("FastTrack.Enabled should remain false when explicitly set")
		}
		if cfg.FastTrack.DefaultTier != TierCritical {
			t.Errorf("FastTrack.DefaultTier should remain %q, got %q", TierCritical, cfg.FastTrack.DefaultTier)
		}
		if len(cfg.FastTrack.Tiers) != 1 {
			t.Errorf("FastTrack.Tiers should not be overwritten when non-empty, got %d entries", len(cfg.FastTrack.Tiers))
		}
	})
}

func TestConfig_FastTrackInFullConfig(t *testing.T) {
	t.Parallel()
	yamlInput := minimalValidYAML + `
fast_track:
  enabled: true
  default_tier: bug_fix
  tiers:
    bug_fix:
      design: auto
      spec: human
      dev-plan: auto
      review: auto
      max_cycles: 2
`

	path := writeTempConfig(t, []byte(yamlInput))
	cfg, err := LoadFrom(path)
	if err != nil {
		t.Fatalf("LoadFrom: %v", err)
	}

	if !cfg.FastTrack.IsEnabled() {
		t.Error("FastTrack.Enabled should be true")
	}
	if cfg.FastTrack.DefaultTier != TierBugFix {
		t.Errorf("FastTrack.DefaultTier = %q, want %q", cfg.FastTrack.DefaultTier, TierBugFix)
	}
	if len(cfg.FastTrack.Tiers) != 1 {
		t.Errorf("FastTrack.Tiers count = %d, want 1", len(cfg.FastTrack.Tiers))
	}
}

func TestFastTrackConfig_EmptyGateModesAreValid(t *testing.T) {
	t.Parallel()
	// Gate modes may be empty strings in YAML (omitted fields); these should be valid.
	// They'll be filled in by mergePhase4bDefaults if the entire tier is missing,
	// but if a tier exists with some empty modes, that is also valid (no enforcement).
	cfg := DefaultFastTrackConfig()
	tc := cfg.Tiers[TierFeature]
	tc.Design = ""
	tc.Spec = ""
	cfg.Tiers[TierFeature] = tc

	if err := cfg.Validate(); err != nil {
		t.Errorf("empty gate modes should be valid: %v", err)
	}
}

// contains checks if s contains substr (simple substring match for tests).
func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
