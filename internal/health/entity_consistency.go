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

// featureDevelopingStatuses contains feature statuses that represent active development
// or rework — all child tasks being terminal while the feature is still in these states
// is a signal that the feature should be advanced.
var featureDevelopingStatuses = map[string]struct{}{
	string(model.FeatureStatusDeveloping):  {},
	string(model.FeatureStatusNeedsRework): {},
}

// CheckFeatureChildConsistency checks for state inconsistencies between
// features and their child tasks.
//
// Detects:
//   - Warning: Feature is done/superseded/cancelled but has non-terminal child tasks
//   - Warning: All child tasks are in terminal state but feature is in an early lifecycle state
//   - Warning: All child tasks are in terminal state but feature is still developing or needs-rework
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

		// Check 3: All children terminal but feature is still developing or needs-rework.
		if _, ok := featureDevelopingStatuses[featureStatus]; ok && terminalCount == len(childTasks) {
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

// CheckPlanChildConsistency checks for state inconsistencies between plans and their child features.
//
// Detects:
//   - Warning: All child features are in a finished state but plan is not done
//   - Warning: Plan is done but has non-finished child features
//
// plans and features are slices of entity field maps (as returned by storage).
func CheckPlanChildConsistency(plans []map[string]any, features []map[string]any) CategoryResult {
	result := NewCategoryResult()

	// Build map of plan ID → child features.
	planFeatures := make(map[string][]map[string]any)
	for _, f := range features {
		parent, _ := f["parent"].(string)
		if parent == "" {
			continue
		}
		planFeatures[parent] = append(planFeatures[parent], f)
	}

	for _, p := range plans {
		planID, _ := p["id"].(string)
		if planID == "" {
			continue
		}

		planStatus, _ := p["status"].(string)
		childFeatures := planFeatures[planID]

		if len(childFeatures) == 0 {
			continue
		}

		finishedCount := 0
		nonFinishedCount := 0
		for _, f := range childFeatures {
			fstatus, _ := f["status"].(string)
			if _, ok := featureTerminalOrDone[fstatus]; ok {
				finishedCount++
			} else {
				nonFinishedCount++
			}
		}

		// Check 1: All features finished but plan not done.
		if planStatus != string(model.PlanStatusDone) && finishedCount == len(childFeatures) {
			result.AddIssue(Issue{
				Severity: SeverityWarning,
				EntityID: planID,
				Message: fmt.Sprintf(
					"plan %s has all %d child feature(s) in finished state but plan is %s",
					planID, finishedCount, planStatus,
				),
			})
		}

		// Check 2: Plan is done but has non-finished features.
		if planStatus == string(model.PlanStatusDone) && nonFinishedCount > 0 {
			result.AddIssue(Issue{
				Severity: SeverityWarning,
				EntityID: planID,
				Message: fmt.Sprintf(
					"plan %s is done but has %d non-finished child feature(s)",
					planID, nonFinishedCount,
				),
			})
		}
	}

	return result
}
