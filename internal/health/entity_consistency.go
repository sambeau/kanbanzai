package health

import (
	"fmt"

	"github.com/sambeau/kanbanzai/internal/model"
	"github.com/sambeau/kanbanzai/internal/validate"
)

// featureTerminalOrDone contains feature statuses that imply work is finished.
// This includes the lifecycle terminal states (superseded, cancelled) plus "done".
var featureTerminalOrDone = map[string]struct{}{
	string(model.FeatureStatusDone):       {},
	string(model.FeatureStatusSuperseded): {},
	string(model.FeatureStatusCancelled):  {},
}

// featureEarlyStatuses contains feature statuses that precede active development.
var featureEarlyStatuses = map[string]struct{}{
	string(model.FeatureStatusProposed):    {},
	string(model.FeatureStatusDesigning):   {},
	string(model.FeatureStatusSpecifying):  {},
	string(model.FeatureStatusDevPlanning): {},
}

// CheckFeatureChildConsistency checks for state inconsistencies between
// features and their child tasks.
//
// Detects:
//   - Warning: Feature is done/superseded/cancelled but has non-terminal child tasks
//   - Warning: All child tasks are in terminal state but feature is in an early lifecycle state
//
// features and tasks are slices of entity field maps (as returned by storage).
func CheckFeatureChildConsistency(features []map[string]any, tasks []map[string]any) CategoryResult {
	result := NewCategoryResult()

	// Build map of feature ID → child tasks.
	featureTasks := make(map[string][]map[string]any)
	for _, t := range tasks {
		pf, _ := t["parent_feature"].(string)
		if pf == "" {
			continue
		}
		featureTasks[pf] = append(featureTasks[pf], t)
	}

	for _, f := range features {
		featureID, _ := f["id"].(string)
		if featureID == "" {
			continue
		}

		featureStatus, _ := f["status"].(string)
		childTasks := featureTasks[featureID]

		if len(childTasks) == 0 {
			continue
		}

		// Count terminal vs non-terminal child tasks.
		nonTerminalCount := 0
		terminalCount := 0
		for _, t := range childTasks {
			taskStatus, _ := t["status"].(string)
			if validate.IsTerminalState(model.EntityKindTask, taskStatus) {
				terminalCount++
			} else {
				nonTerminalCount++
			}
		}

		// Check 1: Feature is terminal/done but has non-terminal children.
		if _, ok := featureTerminalOrDone[featureStatus]; ok && nonTerminalCount > 0 {
			result.AddIssue(Issue{
				Severity: SeverityWarning,
				EntityID: featureID,
				Message: fmt.Sprintf(
					"feature %s is %s but has %d non-terminal child task(s)",
					featureID, featureStatus, nonTerminalCount,
				),
			})
		}

		// Check 2: All children terminal but feature in early state.
		if _, ok := featureEarlyStatuses[featureStatus]; ok && terminalCount == len(childTasks) {
			result.AddIssue(Issue{
				Severity: SeverityWarning,
				EntityID: featureID,
				Message: fmt.Sprintf(
					"feature %s has all %d child task(s) in terminal state but feature is %s",
					featureID, terminalCount, featureStatus,
				),
			})
		}
	}

	return result
}
