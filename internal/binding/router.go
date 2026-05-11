// Package binding router.go — pure-function routing resolver for the P44 contract.
//
// Resolve maps a feature's lifecycle status and tier to a BindingResolution,
// which the 3.0 pipeline's stepLookupBinding consumes. It is a pure function:
// no file I/O, no logging, no network calls in the hot path.
//
// Spec: work/P64-binding-governance/P64-spec-phase3-router-extraction-p44-contract.md

//go:generate go run ./gen routing.yaml router_gen.go

package binding

import "fmt"

// ─── Routing data types ──────────────────────────────────────────────────────

// FastTrackTier carries the tier-derived gate mode and max-cycles cap.
// Returned as part of BindingResolution.ModeProfile.
type FastTrackTier struct {
	Review    string // "auto", "human", "conditional"
	Design    string
	Spec      string
	DevPlan   string
	MaxCycles int
}

// FastTrackConfig maps tier names to their FastTrack configuration.
// This is the config struct that Resolve accepts to derive ModeProfile.
type FastTrackConfig struct {
	DefaultTier string
	Tiers       map[string]*FastTrackTier
}

// BindingResolution is the result of resolving a feature's routing.
// It exposes the binding key, any tier-conditional skill overrides,
// and the FastTrack-derived mode profile (may be nil for tiers without
// FastTrack entries).
type BindingResolution struct {
	BindingKey     string            // key into the binding registry (e.g., "developing")
	SkillOverrides map[string]string // tier-conditional skill substitution map (e.g., "implement-task" → "implement-retro-fix")
	ModeProfile    *FastTrackTier    // nil when the tier has no FastTrack entry
}

// RoutingTable maps feature status values to binding keys.
// The canonical source is internal/binding/routing.yaml.
type RoutingTable map[string]string

// SkillChecker checks whether a named skill exists on disk.
// nil means "skip skill checks" — callers that cannot provide a real checker
// (e.g., tests, offline contexts) pass nil to bypass disk I/O.
type SkillChecker func(skillName string) bool

// ─── Error types ─────────────────────────────────────────────────────────────

// ErrNoBinding is returned when a feature's status has no entry in the routing table.
type ErrNoBinding struct {
	Status string
}

func (e ErrNoBinding) Error() string {
	return fmt.Sprintf("no binding for status %q", e.Status)
}

// ErrUnknownTier is returned when a feature's tier is not a recognised value.
type ErrUnknownTier struct {
	Tier string
}

func (e ErrUnknownTier) Error() string {
	return fmt.Sprintf("unknown tier %q", e.Tier)
}

// Is enables errors.Is comparison against any ErrUnknownTier regardless of Tier value.
func (e ErrUnknownTier) Is(target error) bool {
	_, ok := target.(ErrUnknownTier)
	return ok
}

// ErrSkillMissing is returned when a skill referenced by the resolved binding
// is not present on disk.
type ErrSkillMissing struct {
	Skill      string
	BindingKey string
}

func (e ErrSkillMissing) Error() string {
	return fmt.Sprintf("skill %q referenced by binding %q is not present on disk", e.Skill, e.BindingKey)
}

// ─── Tier constants ──────────────────────────────────────────────────────────

// validTiers is resolved from the generated ValidTiers map (router_gen.go).

// ─── Routing table ───────────────────────────────────────────────────────────

// DefaultRoutingTable returns the routing table generated from routing.yaml.
func DefaultRoutingTable() RoutingTable {
	return routingTable
}

// ─── Default FastTrackConfig ─────────────────────────────────────────────────

// defaultFastTrackTiers provides the default gate modes for each tier.
// Feature/bug_fix tiers use "auto" review; retro_fix uses "conditional".
var defaultFastTrackTiers = map[string]*FastTrackTier{
	"feature": {
		Review:  "auto",
		Design:  "auto",
		Spec:    "auto",
		DevPlan: "auto",
	},
	"bug_fix": {
		Review:  "auto",
		Design:  "auto",
		Spec:    "auto",
		DevPlan: "auto",
	},
	"retro_fix": {
		Review:  "conditional",
		Design:  "auto",
		Spec:    "auto",
		DevPlan: "auto",
	},
	"critical": {
		Review:  "human",
		Design:  "human",
		Spec:    "human",
		DevPlan: "human",
	},
}

// DefaultFastTrackConfig returns the built-in FastTrack configuration.
func DefaultFastTrackConfig() *FastTrackConfig {
	tiers := make(map[string]*FastTrackTier, len(defaultFastTrackTiers))
	for k, v := range defaultFastTrackTiers {
		tiers[k] = v
	}
	return &FastTrackConfig{Tiers: tiers}
}

// ─── Skill overrides ─────────────────────────────────────────────────────────

// defaultSkillOverrides maps tier → original skill → substitute skill.
// This implements the tier-conditional skill substitution described in REQ-002.
var defaultSkillOverrides = map[string]map[string]string{
	"retro_fix": {
		"implement-task": "implement-retro-fix",
	},
}

// ─── Resolve ─────────────────────────────────────────────────────────────────

// Resolve resolves a feature's routing to a BindingResolution.
//
// It is a pure function: no file I/O, no logging, no network calls.
// skillChecker is called only when non-nil; pass nil to skip disk checks.
//
// Returns:
//   - ErrUnknownTier when tier is not in validTiers
//   - ErrNoBinding when status has no entry in the routing table
//   - ErrSkillMissing when skillChecker reports a required skill is absent
func Resolve(status string, tier string, routingTable RoutingTable, config *FastTrackConfig, skillChecker SkillChecker) (*BindingResolution, error) {
	// Normalise tier: empty tier uses the configured DefaultTier, or falls
	// back to "feature" when no DefaultTier is configured.
	if tier == "" {
		if config != nil && config.DefaultTier != "" {
			tier = config.DefaultTier
		} else {
			tier = "feature"
		}
	}

	// Validate tier.
	if !ValidTiers[tier] {
		return nil, ErrUnknownTier{Tier: tier}
	}

	// Look up the binding key from the routing table.
	bindingKey, ok := routingTable[status]
	if !ok {
		return nil, ErrNoBinding{Status: status}
	}

	// Build resolution.
	resolution := &BindingResolution{
		BindingKey:     bindingKey,
		SkillOverrides: nil,
		ModeProfile:    nil,
	}

	// Apply tier-conditional skill overrides.
	if overrides, ok := defaultSkillOverrides[tier]; ok && len(overrides) > 0 {
		resolution.SkillOverrides = overrides
	}

	// Check skills if a checker is provided.
	if skillChecker != nil {
		// Check the skills from the overrides. If any override target skill
		// is missing, return ErrSkillMissing.
		for _, substituteSkill := range resolution.SkillOverrides {
			if !skillChecker(substituteSkill) {
				return nil, ErrSkillMissing{Skill: substituteSkill, BindingKey: bindingKey}
			}
		}
	}

	// Derive ModeProfile from FastTrackConfig.
	if config != nil {
		if ft, ok := config.Tiers[tier]; ok {
			resolution.ModeProfile = ft
		}
		// If tier not in config, ModeProfile stays nil (caller interprets as
		// "no FastTrack entry" per AC-012).
	}

	return resolution, nil
}
