package mcp

import (
	"github.com/sambeau/kanbanzai/internal/gate"
	"github.com/sambeau/kanbanzai/internal/health"
	"github.com/sambeau/kanbanzai/internal/service"
	"github.com/sambeau/kanbanzai/internal/validate"
)

// allGatedStages is the canonical list of stages that have gate prerequisites.
var allGatedStages = []string{
	"designing",
	"specifying",
	"dev-planning",
	"developing",
	"reviewing",
}

// GateSourceHealthChecker returns an AdditionalHealthChecker that reports
// whether each gated stage draws its prerequisites from the registry file
// or from hardcoded defaults. This helps operators track migration progress
// from hardcoded gates to registry-driven gates.
func GateSourceHealthChecker(registryCache *gate.RegistryCache) AdditionalHealthChecker {
	return func() (*validate.HealthReport, error) {
		report := &validate.HealthReport{
			Summary: validate.HealthSummary{
				EntitiesByType: make(map[string]int),
			},
		}

		var registryStages []string

		bf, err := registryCache.Get()
		if err == nil && bf != nil {
			for name := range bf.StageBindings {
				registryStages = append(registryStages, name)
			}
		}
		// If bf is nil (no registry file), registryStages stays empty →
		// all stages will be reported as hardcoded.

		result := health.CheckGateSources(registryStages, allGatedStages)
		mergeHealthResult(report, "gate_source", result)

		return report, nil
	}
}

// CheckpointOverrideHealthChecker returns an AdditionalHealthChecker that
// flags features whose gate overrides are still pending on a human checkpoint.
// These are overrides that used the "checkpoint" override policy and are
// awaiting human confirmation.
func CheckpointOverrideHealthChecker(entitySvc *service.EntityService) AdditionalHealthChecker {
	return func() (*validate.HealthReport, error) {
		report := &validate.HealthReport{
			Summary: validate.HealthSummary{
				EntitiesByType: make(map[string]int),
			},
		}

		features, err := entitySvc.List("feature")
		if err != nil {
			// Best-effort: skip checkpoint override check if features cannot be loaded.
			return report, nil
		}

		featureMaps := make([]map[string]any, len(features))
		for i, f := range features {
			featureMaps[i] = f.State
		}

		result := health.CheckCheckpointOverrides(featureMaps)
		mergeHealthResult(report, "checkpoint_overrides", result)

		return report, nil
	}
}
