package health

import (
	"fmt"
	"time"
)

// CheckUnlinkedResolvedIncidents checks for incidents in "resolved" or
// "root-cause-identified" status that have no linked_rca and whose
// resolved_at (or updated) timestamp is older than the configured threshold.
//
// incidents is a slice of incident field maps.
// rcaLinkWarnAfterDays is the threshold in days (0 disables the check).
// now is the current time, passed in for testability.
func CheckUnlinkedResolvedIncidents(incidents []map[string]any, rcaLinkWarnAfterDays int, now time.Time) CategoryResult {
	result := NewCategoryResult()

	if rcaLinkWarnAfterDays <= 0 {
		return result // disabled
	}

	threshold := time.Duration(rcaLinkWarnAfterDays) * 24 * time.Hour

	for _, inc := range incidents {
		status, _ := inc["status"].(string)
		if status != "resolved" && status != "root-cause-identified" {
			continue
		}

		// Skip if a linked RCA already exists.
		if linkedRCA, _ := inc["linked_rca"].(string); linkedRCA != "" {
			continue
		}

		incidentID, _ := inc["id"].(string)

		// Determine the reference timestamp: prefer resolved_at, fall back to updated.
		refTime, ok := parseTimestamp(inc, "resolved_at")
		if !ok {
			refTime, ok = parseTimestamp(inc, "updated")
		}
		if !ok {
			// No usable timestamp — flag it anyway since we can't tell how old it is.
			result.AddIssue(Issue{
				Severity: SeverityWarning,
				EntityID: incidentID,
				Message: fmt.Sprintf(
					"%s is in %s status with no linked RCA and no parseable timestamp",
					incidentID, status,
				),
			})
			continue
		}

		age := now.Sub(refTime)
		if age >= threshold {
			dayCount := int(age.Hours() / 24)
			result.AddIssue(Issue{
				Severity: SeverityWarning,
				EntityID: incidentID,
				Message: fmt.Sprintf(
					"%s has been in %s status for %d day(s) with no linked RCA",
					incidentID, status, dayCount,
				),
			})
		}
	}

	return result
}

// parseTimestamp tries to parse an RFC3339 timestamp from a field map entry.
func parseTimestamp(fields map[string]any, key string) (time.Time, bool) {
	val, ok := fields[key]
	if !ok {
		return time.Time{}, false
	}
	s, ok := val.(string)
	if !ok || s == "" {
		return time.Time{}, false
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Time{}, false
	}
	return t, true
}
