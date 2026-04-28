package health

import (
	"fmt"

	"github.com/sambeau/kanbanzai/internal/model"
	"github.com/sambeau/kanbanzai/internal/validate"
)

var featureTerminalOrDone = map[string]struct{}{
	string(model.FeatureStatusDone):       {},
	string(model.FeatureStatusSuperseded): {},
	string(model.FeatureStatusCancelled):  {},
}

var featureEarlyStatuses = map[string]struct{}{
	string(model.FeatureStatusProposed):    {},
	string(model.FeatureStatusDesigning):   {},
	string(model.FeatureStatusSpecifying):  {},
	string(model.FeatureStatusDevPlanning): {},
}

var featureDevelopingStatuses = map[string]struct{}{
	string(model.FeatureStatusDeveloping):  {},
	string(model.FeatureStatusNeedsRework): {},
}

func CheckFeatureChildConsistency(features []map[string]any, tasks []map[string]any) CategoryResult {
	result := NewCategoryResult()
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
		if _, ok := featureTerminalOrDone[featureStatus]; ok && nonTerminalCount > 0 {
			result.AddIssue(Issue{
				Severity: SeverityWarning,
				EntityID: featureID,
				Message:  fmt.Sprintf("feature %s is %s but has %d non-terminal child task(s)", featureID, featureStatus, nonTerminalCount),
			})
		}
		if _, ok := featureEarlyStatuses[featureStatus]; ok && terminalCount == len(childTasks) {
			result.AddIssue(Issue{
				Severity: SeverityWarning,
				EntityID: featureID,
				Message:  fmt.Sprintf("feature %s has all %d child task(s) in terminal state but feature is %s", featureID, terminalCount, featureStatus),
			})
		}
		if _, ok := featureDevelopingStatuses[featureStatus]; ok && terminalCount == len(childTasks) {
			result.AddIssue(Issue{
				Severity: SeverityWarning,
				EntityID: featureID,
				Message:  fmt.Sprintf("feature %s has all %d child task(s) in terminal state but feature is %s", featureID, terminalCount, featureStatus),
			})
		}
	}
	return result
}

func CheckBatchChildConsistency(batches []map[string]any, features []map[string]any) CategoryResult {
	result := NewCategoryResult()
	batchFeatures := make(map[string][]map[string]any)
	for _, f := range features {
		parent, _ := f["parent"].(string)
		if parent == "" {
			continue
		}
		batchFeatures[parent] = append(batchFeatures[parent], f)
	}
	for _, b := range batches {
		batchID, _ := b["id"].(string)
		if batchID == "" {
			continue
		}
		batchStatus, _ := b["status"].(string)
		childFeatures := batchFeatures[batchID]
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
		if batchStatus != string(model.BatchStatusDone) && finishedCount == len(childFeatures) {
			result.AddIssue(Issue{
				Severity: SeverityWarning,
				EntityID: batchID,
				Message:  fmt.Sprintf("batch %s has all %d child feature(s) in finished state but batch is %s", batchID, finishedCount, batchStatus),
			})
		}
		if batchStatus == string(model.BatchStatusDone) && nonFinishedCount > 0 {
			result.AddIssue(Issue{
				Severity: SeverityWarning,
				EntityID: batchID,
				Message:  fmt.Sprintf("batch %s is done but has %d non-finished child feature(s)", batchID, nonFinishedCount),
			})
		}
	}
	return result
}

func CheckPlanChildConsistency(plans []map[string]any, features []map[string]any) CategoryResult {
	return CheckBatchChildConsistency(plans, features)
}
