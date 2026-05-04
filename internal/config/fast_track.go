package config

import (
	"fmt"
	"slices"
)

// Valid tier name constants per REQ-TIER-001.
const (
	TierRetroFix = "retro_fix"
	TierBugFix   = "bug_fix"
	TierFeature  = "feature"
	TierCritical = "critical"
)

// Valid gate mode constants per REQ-TIER-002.
const (
	GateModeAuto        = "auto"
	GateModeHuman       = "human"
	GateModeConditional = "conditional"
)

// validTiers is the set of recognised tier names.
var validTiers = []string{TierRetroFix, TierBugFix, TierFeature, TierCritical}

// validGateModes is the set of recognised gate modes.
var validGateModes = map[string]bool{GateModeAuto: true, GateModeHuman: true, GateModeConditional: true}

// FastTrackConfig holds the fast_track configuration block per REQ-TIER-005.
type FastTrackConfig struct {
	// Enabled controls whether fast-track automation is active.
	// Defaults to true when not explicitly configured.
	Enabled bool `yaml:"enabled"`
	// DefaultTier is the tier assigned when no explicit tier is set (REQ-TIER-006).
	DefaultTier string `yaml:"default_tier"`
	// Tiers maps tier names to their automation configuration.
	Tiers map[string]TierConfig `yaml:"tiers"`
}

// TierConfig defines the automation matrix and cycle limit for a risk tier
// per REQ-TIER-002 and REQ-TIER-003.
type TierConfig struct {
	// Design gate mode: auto, human, or conditional.
	Design string `yaml:"design"`
	// Spec gate mode: auto or human.
	Spec string `yaml:"spec"`
	// DevPlan gate mode: auto or human.
	DevPlan string `yaml:"dev-plan"`
	// Review gate mode: auto, human, or conditional (REQ-TIER-004).
	Review string `yaml:"review"`
	// MaxCycles is the maximum number of auto fix-validate cycles (REQ-TIER-003).
	MaxCycles int `yaml:"max_cycles"`
}

// DefaultFastTrackConfig returns a FastTrackConfig with sensible defaults
// matching the specification matrix (REQ-TIER-001 through REQ-TIER-003).
func DefaultFastTrackConfig() FastTrackConfig {
	return FastTrackConfig{
		Enabled:     true,
		DefaultTier: TierFeature,
		Tiers: map[string]TierConfig{
			TierRetroFix: {
				Design:    GateModeAuto,
				Spec:      GateModeAuto,
				DevPlan:   GateModeAuto,
				Review:    GateModeConditional,
				MaxCycles: 3,
			},
			TierBugFix: {
				Design:    GateModeAuto,
				Spec:      GateModeHuman,
				DevPlan:   GateModeAuto,
				Review:    GateModeAuto,
				MaxCycles: 2,
			},
			TierFeature: {
				Design:    GateModeHuman,
				Spec:      GateModeAuto,
				DevPlan:   GateModeAuto,
				Review:    GateModeAuto,
				MaxCycles: 2,
			},
			TierCritical: {
				Design:    GateModeHuman,
				Spec:      GateModeHuman,
				DevPlan:   GateModeHuman,
				Review:    GateModeHuman,
				MaxCycles: 0,
			},
		},
	}
}

// IsEnabled returns true if fast-track is enabled.
// A zero-value Config (not configured) defaults to enabled.
func (c *FastTrackConfig) IsEnabled() bool {
	if c.DefaultTier == "" && !c.Enabled && len(c.Tiers) == 0 {
		return true // zero config → enabled by default
	}
	return c.Enabled
}

// Validate checks that the FastTrackConfig is valid:
//   - default_tier must be one of the four valid tier names
//   - every tier key must be a valid tier name
//   - every gate mode in each tier config must be a valid gate mode
//   - max_cycles must be >= 0
func (c *FastTrackConfig) Validate() error {
	if c.DefaultTier != "" && !slices.Contains(validTiers, c.DefaultTier) {
		return fmt.Errorf("fast_track.default_tier %q is not a valid tier name (valid: %v)", c.DefaultTier, validTiers)
	}

	for tierName, tc := range c.Tiers {
		if !slices.Contains(validTiers, tierName) {
			return fmt.Errorf("fast_track.tiers: %q is not a valid tier name (valid: %v)", tierName, validTiers)
		}
		if err := validateGateMode(tc.Design, tierName, "design"); err != nil {
			return err
		}
		if err := validateGateMode(tc.Spec, tierName, "spec"); err != nil {
			return err
		}
		if err := validateGateMode(tc.DevPlan, tierName, "dev-plan"); err != nil {
			return err
		}
		if err := validateGateMode(tc.Review, tierName, "review"); err != nil {
			return err
		}
		if tc.MaxCycles < 0 {
			return fmt.Errorf("fast_track.tiers.%s.max_cycles must be >= 0, got %d", tierName, tc.MaxCycles)
		}
	}

	return nil
}

// validateGateMode checks that a gate mode is valid for a given tier and stage.
func validateGateMode(mode, tierName, stage string) error {
	if mode != "" && !validGateModes[mode] {
		return fmt.Errorf("fast_track.tiers.%s.%s: %q is not a valid gate mode (valid: %v)", tierName, stage, mode, validGateModes)
	}
	return nil
}
