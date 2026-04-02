package health

import "fmt"

// CheckGateSources reports which gate source (registry vs hardcoded) is active
// for each stage. This helps operators track migration progress from hardcoded
// gates to registry-driven gates.
//
// registryStages contains stage names present in the binding registry.
// allStages contains the full set of gated stage names to check.
func CheckGateSources(registryStages []string, allStages []string) CategoryResult {
	result := NewCategoryResult()

	registrySet := make(map[string]bool, len(registryStages))
	for _, s := range registryStages {
		registrySet[s] = true
	}

	for _, stage := range allStages {
		if registrySet[stage] {
			result.AddIssue(Issue{
				Severity: SeverityInfo,
				Message:  fmt.Sprintf("stage %s: gate source is registry", stage),
			})
		} else {
			result.AddIssue(Issue{
				Severity: SeverityInfo,
				Message:  fmt.Sprintf("stage %s: gate source is hardcoded", stage),
			})
		}
	}

	return result
}

// CheckCheckpointOverrides scans serialised feature field maps for gate
// override records that have a checkpoint_id field, indicating a checkpoint
// override that is still pending human approval. Each such record produces
// a Warning issue.
//
// Regular overrides (without checkpoint_id) are ignored — those are reported
// by CheckGateOverrides instead.
func CheckCheckpointOverrides(features []map[string]any) CategoryResult {
	result := NewCategoryResult()

	for _, f := range features {
		featureID, _ := f["id"].(string)
		if featureID == "" {
			continue
		}

		rawOverrides, ok := f["overrides"]
		if !ok {
			continue
		}

		overrides, ok := rawOverrides.([]any)
		if !ok || len(overrides) == 0 {
			continue
		}

		for _, item := range overrides {
			override, ok := item.(map[string]any)
			if !ok {
				continue
			}

			checkpointID, _ := override["checkpoint_id"].(string)
			if checkpointID == "" {
				continue
			}

			from, _ := override["from_status"].(string)
			to, _ := override["to_status"].(string)

			result.AddIssue(Issue{
				Severity: SeverityWarning,
				EntityID: featureID,
				Message:  fmt.Sprintf("feature %s: checkpoint override pending on %s→%s (checkpoint %s)", featureID, from, to, checkpointID),
			})
		}
	}

	return result
}
