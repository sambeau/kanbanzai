package service

import (
	"fmt"

	"github.com/sambeau/kanbanzai/internal/model"
	"github.com/sambeau/kanbanzai/internal/validate"
)

// CountNonTerminalTasks returns the count of tasks with parent_feature == featureID
// that are NOT in a terminal state (done, not-planned, duplicate).
func (s *EntityService) CountNonTerminalTasks(featureID string) (int, error) {
	tasks, err := s.List("task")
	if err != nil {
		return 0, err
	}

	count := 0
	for _, t := range tasks {
		if stringFromState(t.State, "parent_feature") != featureID {
			continue
		}
		status := stringFromState(t.State, "status")
		if !validate.IsTerminalState(model.EntityKindTask, status) {
			count++
		}
	}
	return count, nil
}

// CountNonTerminalFeatures returns the count of features with parent == planID
// that are NOT in a terminal state (done, superseded, cancelled).
func (s *EntityService) CountNonTerminalFeatures(planID string) (int, error) {
	features, err := s.List("feature")
	if err != nil {
		return 0, err
	}

	count := 0
	for _, f := range features {
		if stringFromState(f.State, "parent") != planID {
			continue
		}
		status := stringFromState(f.State, "status")
		if !isFeatureEffectivelyTerminal(status) {
			count++
		}
	}
	return count, nil
}

// CheckAllTasksTerminal returns whether all tasks for the feature are in a
// terminal state AND whether at least one is in "done" status.
// If there are no tasks, returns allTerminal=true, hasOneDone=false.
func (s *EntityService) CheckAllTasksTerminal(featureID string) (allTerminal bool, hasOneDone bool, err error) {
	tasks, err := s.List("task")
	if err != nil {
		return false, false, err
	}

	for _, t := range tasks {
		if stringFromState(t.State, "parent_feature") != featureID {
			continue
		}
		status := stringFromState(t.State, "status")
		if !validate.IsTerminalState(model.EntityKindTask, status) {
			return false, false, nil
		}
		if status == string(model.TaskStatusDone) {
			hasOneDone = true
		}
	}
	return true, hasOneDone, nil
}

// CheckAllFeaturesTerminal returns whether all features for the plan are in a
// terminal state (done, superseded, or cancelled) AND whether at least one is
// in "done" status.
// If there are no features, returns allTerminal=true, hasOneDone=false.
func (s *EntityService) CheckAllFeaturesTerminal(planID string) (allTerminal bool, hasOneDone bool, err error) {
	features, err := s.List("feature")
	if err != nil {
		return false, false, err
	}

	for _, f := range features {
		if stringFromState(f.State, "parent") != planID {
			continue
		}
		status := stringFromState(f.State, "status")
		if !isFeatureEffectivelyTerminal(status) {
			return false, false, nil
		}
		if status == string(model.FeatureStatusDone) {
			hasOneDone = true
		}
	}
	return true, hasOneDone, nil
}

// FeatureParentPlan returns the plan ID for a feature, or "" if the feature
// does not exist or has no parent plan. Best-effort: errors are suppressed.
func (s *EntityService) FeatureParentPlan(featureID string) string {
	feat, err := s.Get("feature", featureID, "")
	if err != nil {
		return ""
	}
	parent, _ := feat.State["parent"].(string)
	if !model.IsPlanID(parent) {
		return ""
	}
	return parent
}

// isFeatureEffectivelyTerminal reports whether a feature status is terminal for
// the purpose of plan-level completion checks. A feature is effectively terminal
// when it is done, superseded, or cancelled. Note that validate.IsTerminalState
// covers superseded and cancelled; done is added explicitly because features can
// still transition out of done (to superseded/cancelled) but are considered
// complete from a progress perspective.
func isFeatureEffectivelyTerminal(status string) bool {
	return status == string(model.FeatureStatusDone) ||
		validate.IsTerminalState(model.EntityKindFeature, status)
}

// MaybeAutoAdvanceFeature checks whether all tasks for a feature are terminal
// with at least one done, and if so automatically transitions the feature from
// developing or needs-rework to reviewing (REQ-008, REQ-009).
//
// Returns (true, nil) when the advance fired, (false, nil) when conditions were
// not met, and (false, err) when a non-blocking error occurred. Callers MUST
// surface errors as warnings without failing the primary operation (REQ-012).
func (s *EntityService) MaybeAutoAdvanceFeature(featureID string) (bool, error) {
	feat, err := s.Get("feature", featureID, "")
	if err != nil {
		return false, fmt.Errorf("auto-advance: load feature %s: %w", featureID, err)
	}

	currentStatus, _ := feat.State["status"].(string)
	if currentStatus != string(model.FeatureStatusDeveloping) &&
		currentStatus != string(model.FeatureStatusNeedsRework) {
		// Feature is not in an auto-advanceable state.
		return false, nil
	}

	allTerminal, hasOneDone, err := s.CheckAllTasksTerminal(featureID)
	if err != nil {
		return false, fmt.Errorf("auto-advance: check tasks for %s: %w", featureID, err)
	}
	if !allTerminal || !hasOneDone {
		return false, nil
	}

	_, err = s.UpdateStatus(UpdateStatusInput{
		Type:   "feature",
		ID:     featureID,
		Status: string(model.FeatureStatusReviewing),
	})
	if err != nil {
		return false, fmt.Errorf("auto-advance: transition feature %s to reviewing: %w", featureID, err)
	}
	return true, nil
}

// MaybeAutoAdvancePlan checks whether all features for a plan are in a terminal
// state with at least one done, and if so automatically transitions the plan
// from active to done (REQ-013, REQ-014).
//
// The plan lifecycle requires active → reviewing → done; this function chains
// both transitions internally so callers see an atomic advance to done.
//
// Returns (true, nil) when the advance fired, (false, nil) when conditions were
// not met, and (false, err) when a non-blocking error occurred. Callers MUST
// surface errors as warnings without failing the primary operation (REQ-016).
func (s *EntityService) MaybeAutoAdvancePlan(planID string) (bool, error) {
	plan, err := s.GetPlan(planID)
	if err != nil {
		return false, fmt.Errorf("auto-advance: load plan %s: %w", planID, err)
	}

	planStatus, _ := plan.State["status"].(string)
	if planStatus != string(model.PlanStatusActive) {
		return false, nil
	}

	allTerminal, hasOneDone, err := s.CheckAllFeaturesTerminal(planID)
	if err != nil {
		return false, fmt.Errorf("auto-advance: check features for plan %s: %w", planID, err)
	}
	if !allTerminal || !hasOneDone {
		return false, nil
	}

	_, _, slug := model.ParsePlanID(planID)

	// The plan lifecycle is active → reviewing → done. Chain both transitions
	// so the auto-advance lands at done as required by REQ-013.
	if _, err = s.UpdatePlanStatus(planID, slug, string(model.PlanStatusReviewing)); err != nil {
		return false, fmt.Errorf("auto-advance: transition plan %s to reviewing: %w", planID, err)
	}
	if _, err = s.UpdatePlanStatus(planID, slug, string(model.PlanStatusDone)); err != nil {
		return false, fmt.Errorf("auto-advance: transition plan %s to done: %w", planID, err)
	}
	return true, nil
}
