package health

import "fmt"

// CheckGateOverrides scans a slice of serialised feature field maps for gate
// override records and produces one Warning Issue per record found. This
// surfaces any features that bypassed a stage gate so that the health tool
// can flag them as attention items (FR-015).
//
// Each feature map is expected to contain an optional "overrides" field whose
// value is a []any of maps with keys: from_status, to_status, reason, timestamp.
// This matches the serialisation written by PersistFeatureOverrides / featureFields.
func CheckGateOverrides(features []map[string]any) CategoryResult {
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

			from, _ := override["from_status"].(string)
			to, _ := override["to_status"].(string)
			reason, _ := override["reason"].(string)

			result.AddIssue(Issue{
				Severity: SeverityWarning,
				EntityID: featureID,
				Message:  fmt.Sprintf("feature %s: gate override on %s→%s: %s", featureID, from, to, reason),
			})
		}
	}

	return result
}
