package gate

import (
	"github.com/sambeau/kanbanzai/internal/binding"
	"github.com/sambeau/kanbanzai/internal/model"
)

// HardcodedGateFunc is the signature for the legacy hardcoded gate checker.
// The caller adapts service.CheckTransitionGate into this signature so the
// gate package does not import the service package.
type HardcodedGateFunc func(from, to string, feature *model.Feature, docSvc DocumentService, entitySvc EntityService) GateResult

// GateRouter decides whether to evaluate gate prerequisites from the binding
// registry or fall back to the hardcoded gate checker.
type GateRouter struct {
	cache    *RegistryCache
	fallback HardcodedGateFunc
}

// NewGateRouter creates a router that prefers registry-defined prerequisites
// and falls back to the hardcoded gate function when the registry is
// unavailable or has no prerequisites for a stage.
func NewGateRouter(cache *RegistryCache, fallback HardcodedGateFunc) *GateRouter {
	return &GateRouter{
		cache:    cache,
		fallback: fallback,
	}
}

// CheckGate evaluates whether the transition to the target stage is allowed.
//
// Resolution order:
//  1. If a registry cache is available and defines prerequisites for the
//     target stage, evaluate them via EvaluatePrerequisites.
//  2. Otherwise, delegate to the hardcoded fallback function.
func (r *GateRouter) CheckGate(from, to string, ctx PrereqEvalContext) GateResult {
	if r.cache != nil {
		prereqs, ok := r.cache.LookupPrereqs(to)
		if ok && prereqs != nil {
			return r.evaluateRegistry(prereqs, to, ctx)
		}
	}

	return r.callFallback(from, to, ctx)
}

// OverridePolicy returns the override policy for the target stage.
// Returns "agent" when the cache is nil or the stage has no explicit policy.
func (r *GateRouter) OverridePolicy(to string) string {
	if r.cache == nil {
		return "agent"
	}
	policy, _ := r.cache.LookupOverridePolicy(to)
	return policy
}

// evaluateRegistry runs the registered prerequisite evaluators and combines
// results into a single GateResult.
func (r *GateRouter) evaluateRegistry(prereqs *binding.Prerequisites, stage string, ctx PrereqEvalContext) GateResult {
	results := EvaluatePrerequisites(prereqs, stage, ctx)

	for _, res := range results {
		if !res.Satisfied {
			return GateResult{
				Stage:     stage,
				Satisfied: false,
				Reason:    res.Reason,
				Source:    "registry",
			}
		}
	}

	return GateResult{
		Stage:     stage,
		Satisfied: true,
		Source:    "registry",
	}
}

// callFallback delegates to the hardcoded gate function and tags the result.
func (r *GateRouter) callFallback(from, to string, ctx PrereqEvalContext) GateResult {
	result := r.fallback(from, to, ctx.Feature, ctx.DocSvc, ctx.EntitySvc)
	result.Source = "hardcoded"
	return result
}
