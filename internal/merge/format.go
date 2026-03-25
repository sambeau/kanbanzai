package merge

// FormatGateResults returns a YAML-friendly structured output.
func FormatGateResults(result GateCheckResult) map[string]any {
	output := map[string]any{
		"entity_id":      result.EntityID,
		"branch":         result.Branch,
		"overall_status": result.OverallStatus,
	}

	gates := make([]map[string]any, 0, len(result.Gates))
	for _, g := range result.Gates {
		gate := map[string]any{
			"name":     g.Name,
			"status":   string(g.Status),
			"severity": string(g.Severity),
		}
		if g.Message != "" {
			gate["message"] = g.Message
		}
		gates = append(gates, gate)
	}
	output["gates"] = gates

	// Add summary counts
	passed, failed, warning := CountByStatus(result.Gates)
	output["summary"] = map[string]any{
		"total":   len(result.Gates),
		"passed":  passed,
		"failed":  failed,
		"warning": warning,
	}

	return output
}

// FormatGateResultsCompact returns a minimal output suitable for quick status checks.
func FormatGateResultsCompact(result GateCheckResult) map[string]any {
	output := map[string]any{
		"entity_id": result.EntityID,
		"status":    result.OverallStatus,
	}

	// Only include failed/warning gates
	var issues []map[string]any
	for _, g := range result.Gates {
		if g.Status != GateStatusPassed {
			issues = append(issues, map[string]any{
				"gate":     g.Name,
				"status":   string(g.Status),
				"severity": string(g.Severity),
				"message":  g.Message,
			})
		}
	}

	if len(issues) > 0 {
		output["issues"] = issues
	}

	return output
}

// FormatOverrides returns a YAML-friendly representation of overrides.
func FormatOverrides(overrides []Override) []map[string]any {
	if len(overrides) == 0 {
		return nil
	}

	result := make([]map[string]any, 0, len(overrides))
	for _, o := range overrides {
		result = append(result, map[string]any{
			"gate":          o.Gate,
			"reason":        o.Reason,
			"overridden_by": o.OverriddenBy,
			"overridden_at": o.OverriddenAt,
		})
	}
	return result
}
