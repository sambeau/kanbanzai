package mcp

import (
	"time"

	"kanbanzai/internal/health"
	"kanbanzai/internal/service"
	"kanbanzai/internal/validate"
)

// Phase4bHealthChecker returns an AdditionalHealthChecker that validates
// incident health, including unlinked resolved incidents missing an RCA.
func Phase4bHealthChecker(
	entitySvc *service.EntityService,
	rcaLinkWarnAfterDays int,
) AdditionalHealthChecker {
	return func() (*validate.HealthReport, error) {
		report := &validate.HealthReport{
			Summary: validate.HealthSummary{
				EntitiesByType: make(map[string]int),
			},
		}

		// Load all incidents (best-effort; skip if none exist).
		incidents, err := entitySvc.ListIncidents("", "")
		if err != nil {
			// No incidents directory yet is not an error — just skip.
			return report, nil
		}

		incidentMaps := make([]map[string]any, len(incidents))
		for i, inc := range incidents {
			incidentMaps[i] = inc.State
		}

		report.Summary.EntitiesByType["incident"] = len(incidents)

		// Check for resolved/root-cause-identified incidents without linked RCA.
		unlinkedResult := health.CheckUnlinkedResolvedIncidents(incidentMaps, rcaLinkWarnAfterDays, time.Now())
		mergeHealthResult(report, "unlinked_resolved_incidents", unlinkedResult)

		return report, nil
	}
}
