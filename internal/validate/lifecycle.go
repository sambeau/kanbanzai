package validate

import (
	"fmt"
	"sort"
	"strings"

	"github.com/sambeau/kanbanzai/internal/model"
)

type EntityKind = model.EntityKind

const (
	EntityBatch         = model.EntityKindBatch
	EntityFeature       = model.EntityKindFeature
	EntityTask          = model.EntityKindTask
	EntityBug           = model.EntityKindBug
	EntityDecision      = model.EntityKindDecision
	EntityIncident      = model.EntityKindIncident
	EntityStrategicPlan = model.EntityKindStrategicPlan

	// Deprecated: use EntityBatch.
	EntityPlan = EntityBatch
)

var entryStates = map[EntityKind]string{
	EntityBatch:         string(model.BatchStatusProposed),
	EntityFeature:       phase2FeatureEntryState, // Phase 2 entry state (document-driven lifecycle)
	EntityTask:          string(model.TaskStatusQueued),
	EntityBug:           string(model.BugStatusReported),
	EntityDecision:      string(model.DecisionStatusProposed),
	EntityIncident:      string(model.IncidentStatusReported),
	EntityStrategicPlan: string(model.PlanningStatusIdea),
}

// phase2FeatureEntryState is the entry state for Phase 2 features (document-driven lifecycle).
const phase2FeatureEntryState = "proposed"

var terminalStates = map[EntityKind]map[string]struct{}{
	EntityBatch: {
		string(model.BatchStatusSuperseded): {},
		string(model.BatchStatusCancelled):  {},
	},
	EntityStrategicPlan: {
		string(model.PlanningStatusSuperseded): {},
		string(model.PlanningStatusCancelled):  {},
	},
	EntityFeature: {
		string(model.FeatureStatusSuperseded): {},
		string(model.FeatureStatusCancelled):  {},
	},
	EntityTask: {
		string(model.TaskStatusDone):       {},
		string(model.TaskStatusNotPlanned): {},
		string(model.TaskStatusDuplicate):  {},
	},
	EntityBug: {
		string(model.BugStatusClosed):     {},
		string(model.BugStatusDuplicate):  {},
		string(model.BugStatusNotPlanned): {},
	},
	EntityDecision: {
		string(model.DecisionStatusRejected):   {},
		string(model.DecisionStatusSuperseded): {},
	},
	EntityIncident: {
		string(model.IncidentStatusClosed): {},
	},
}

// Deprecated aliases for backward compatibility.
var _ = terminalStates[EntityPlan]

var allowedTransitions = map[EntityKind]map[string]map[string]struct{}{
	// StrategicPlan lifecycle: idea → shaping → ready → active → done
	EntityStrategicPlan: {
		string(model.PlanningStatusIdea): {
			string(model.PlanningStatusShaping):    {},
			string(model.PlanningStatusSuperseded): {},
			string(model.PlanningStatusCancelled):  {},
		},
		string(model.PlanningStatusShaping): {
			string(model.PlanningStatusReady):      {},
			string(model.PlanningStatusIdea):       {},
			string(model.PlanningStatusSuperseded): {},
			string(model.PlanningStatusCancelled):  {},
		},
		string(model.PlanningStatusReady): {
			string(model.PlanningStatusActive):     {},
			string(model.PlanningStatusShaping):    {},
			string(model.PlanningStatusSuperseded): {},
			string(model.PlanningStatusCancelled):  {},
		},
		string(model.PlanningStatusActive): {
			string(model.PlanningStatusDone):       {},
			string(model.PlanningStatusShaping):    {},
			string(model.PlanningStatusSuperseded): {},
			string(model.PlanningStatusCancelled):  {},
		},
		string(model.PlanningStatusDone): {
			string(model.PlanningStatusSuperseded): {},
			string(model.PlanningStatusCancelled):  {},
		},
	},
	// Batch lifecycle: proposed → designing → active → reviewing → done
	EntityBatch: {
		string(model.BatchStatusProposed): {
			string(model.BatchStatusDesigning):  {},
			string(model.BatchStatusActive):     {},
			string(model.BatchStatusSuperseded): {},
			string(model.BatchStatusCancelled):  {},
		},
		string(model.BatchStatusDesigning): {
			string(model.BatchStatusActive):     {},
			string(model.BatchStatusSuperseded): {},
			string(model.BatchStatusCancelled):  {},
		},
		string(model.BatchStatusActive): {
			string(model.BatchStatusReviewing):  {},
			string(model.BatchStatusSuperseded): {},
			string(model.BatchStatusCancelled):  {},
		},
		string(model.BatchStatusReviewing): {
			string(model.BatchStatusDone):       {},
			string(model.BatchStatusActive):     {},
			string(model.BatchStatusSuperseded): {},
			string(model.BatchStatusCancelled):  {},
		},
		string(model.BatchStatusDone): {
			string(model.BatchStatusSuperseded): {},
			string(model.BatchStatusCancelled):  {},
		},
	},
	// Feature lifecycle
	//
	// During the Phase 1 to Phase 2 transition, feature statuses interleave two
	// parallel lifecycle entry points:
	//
	//   Phase 1 (legacy): Draft -> InReview -> Approved -> InProgress -> Review -> Done
	//   Phase 2 (current): Proposed -> Designing -> Specifying -> DevPlanning -> Developing -> Reviewing -> Done
	//
	// NeedsRework bridges both: from Review/Reviewing/InReview/InProgress/Developing
	// back to InReview/InProgress/Developing/Reviewing respectively.
	// Superseded and Cancelled are terminal-abort paths reachable from most statuses.
	EntityFeature: {
		string(model.FeatureStatusDraft): {
			string(model.FeatureStatusInReview): {},
		},
		string(model.FeatureStatusInReview): {
			string(model.FeatureStatusApproved):    {},
			string(model.FeatureStatusNeedsRework): {},
		},
		string(model.FeatureStatusApproved): {
			string(model.FeatureStatusInProgress): {},
			string(model.FeatureStatusSuperseded): {},
		},
		string(model.FeatureStatusInProgress): {
			string(model.FeatureStatusReview):      {},
			string(model.FeatureStatusNeedsRework): {},
		},
		string(model.FeatureStatusReview): {
			string(model.FeatureStatusDone):        {},
			string(model.FeatureStatusNeedsRework): {},
		},
		string(model.FeatureStatusNeedsRework): {
			string(model.FeatureStatusInReview):   {},
			string(model.FeatureStatusInProgress): {},
			string(model.FeatureStatusDeveloping): {},
			string(model.FeatureStatusReviewing):  {},
			string(model.FeatureStatusSuperseded): {},
			string(model.FeatureStatusCancelled):  {},
		},
		string(model.FeatureStatusProposed): {
			string(model.FeatureStatusDesigning):  {},
			string(model.FeatureStatusSpecifying): {},
			string(model.FeatureStatusSuperseded): {},
			string(model.FeatureStatusCancelled):  {},
		},
		string(model.FeatureStatusDesigning): {
			string(model.FeatureStatusSpecifying): {},
			string(model.FeatureStatusSuperseded): {},
			string(model.FeatureStatusCancelled):  {},
		},
		string(model.FeatureStatusSpecifying): {
			string(model.FeatureStatusDevPlanning): {},
			string(model.FeatureStatusDesigning):   {},
			string(model.FeatureStatusSuperseded):  {},
			string(model.FeatureStatusCancelled):   {},
		},
		string(model.FeatureStatusDevPlanning): {
			string(model.FeatureStatusDeveloping): {},
			string(model.FeatureStatusSpecifying): {},
			string(model.FeatureStatusSuperseded): {},
			string(model.FeatureStatusCancelled):  {},
		},
		string(model.FeatureStatusDeveloping): {
			string(model.FeatureStatusReviewing):   {},
			string(model.FeatureStatusDevPlanning): {},
			string(model.FeatureStatusSuperseded):  {},
			string(model.FeatureStatusCancelled):   {},
		},
		string(model.FeatureStatusReviewing): {
			string(model.FeatureStatusDone):        {},
			string(model.FeatureStatusNeedsRework): {},
			string(model.FeatureStatusSuperseded):  {},
			string(model.FeatureStatusCancelled):   {},
		},
		string(model.FeatureStatusDone): {
			string(model.FeatureStatusSuperseded): {},
			string(model.FeatureStatusCancelled):  {},
		},
	},
	EntityTask: {
		string(model.TaskStatusQueued): {
			string(model.TaskStatusReady):      {},
			string(model.TaskStatusNotPlanned): {},
			string(model.TaskStatusDuplicate):  {},
		},
		string(model.TaskStatusReady): {
			string(model.TaskStatusActive):     {},
			string(model.TaskStatusNotPlanned): {},
			string(model.TaskStatusDuplicate):  {},
		},
		string(model.TaskStatusActive): {
			string(model.TaskStatusReady):       {},
			string(model.TaskStatusBlocked):     {},
			string(model.TaskStatusNeedsReview): {},
			string(model.TaskStatusNeedsRework): {},
			string(model.TaskStatusDone):        {},
			string(model.TaskStatusNotPlanned):  {},
			string(model.TaskStatusDuplicate):   {},
		},
		string(model.TaskStatusBlocked): {
			string(model.TaskStatusActive): {},
		},
		string(model.TaskStatusNeedsReview): {
			string(model.TaskStatusDone):        {},
			string(model.TaskStatusNeedsRework): {},
		},
		string(model.TaskStatusNeedsRework): {
			string(model.TaskStatusActive): {},
		},
	},
	EntityBug: {
		string(model.BugStatusReported): {
			string(model.BugStatusTriaged):   {},
			string(model.BugStatusDuplicate): {},
		},
		string(model.BugStatusTriaged): {
			string(model.BugStatusReproduced):      {},
			string(model.BugStatusCannotReproduce): {},
			string(model.BugStatusNotPlanned):      {},
			string(model.BugStatusDuplicate):       {},
			string(model.BugStatusPlanned):         {},
		},
		string(model.BugStatusReproduced): {
			string(model.BugStatusPlanned):    {},
			string(model.BugStatusNotPlanned): {},
		},
		string(model.BugStatusPlanned): {
			string(model.BugStatusInProgress): {},
		},
		string(model.BugStatusInProgress): {
			string(model.BugStatusNeedsReview): {},
		},
		string(model.BugStatusNeedsReview): {
			string(model.BugStatusVerified):    {},
			string(model.BugStatusNeedsRework): {},
		},
		string(model.BugStatusNeedsRework): {
			string(model.BugStatusInProgress): {},
		},
		string(model.BugStatusVerified): {
			string(model.BugStatusClosed): {},
		},
		string(model.BugStatusCannotReproduce): {
			string(model.BugStatusTriaged): {},
		},
	},
	EntityDecision: {
		string(model.DecisionStatusProposed): {
			string(model.DecisionStatusAccepted): {},
			string(model.DecisionStatusRejected): {},
		},
		string(model.DecisionStatusAccepted): {
			string(model.DecisionStatusSuperseded): {},
		},
	},
	EntityIncident: {
		string(model.IncidentStatusReported): {
			string(model.IncidentStatusTriaged): {},
			string(model.IncidentStatusClosed):  {},
		},
		string(model.IncidentStatusTriaged): {
			string(model.IncidentStatusInvestigating): {},
			string(model.IncidentStatusClosed):        {},
		},
		string(model.IncidentStatusInvestigating): {
			string(model.IncidentStatusRootCauseIdentified): {},
			string(model.IncidentStatusClosed):              {},
		},
		string(model.IncidentStatusRootCauseIdentified): {
			string(model.IncidentStatusMitigated):     {},
			string(model.IncidentStatusInvestigating): {},
			string(model.IncidentStatusClosed):        {},
		},
		string(model.IncidentStatusMitigated): {
			string(model.IncidentStatusResolved):      {},
			string(model.IncidentStatusInvestigating): {},
			string(model.IncidentStatusClosed):        {},
		},
		string(model.IncidentStatusResolved): {
			string(model.IncidentStatusClosed): {},
		},
	},
}

// EntryState, EntryStateOrPanic, IsTerminalState, IsKnownState, CanTransition,
// ValidateInitialState, ValidateTransition, ValidNextStates, DependencyTerminalStates,
// IsTaskDependencySatisfied, NextStates, ValidateTaskQueuedToReady

func EntryState(kind EntityKind) (string, bool) {
	state, ok := entryStates[kind]
	return state, ok
}

func EntryStateOrPanic(kind EntityKind) string {
	state, ok := EntryState(kind)
	if !ok {
		panic(fmt.Sprintf("unknown entity kind %q", kind))
	}
	return state
}

func IsTerminalState(kind EntityKind, state string) bool {
	states, ok := terminalStates[kind]
	if !ok {
		return false
	}
	_, ok = states[state]
	return ok
}

func IsKnownState(kind EntityKind, state string) bool {
	if entry, ok := entryStates[kind]; ok && entry == state {
		return true
	}
	transitions, ok := allowedTransitions[kind]
	if !ok {
		return false
	}
	if _, ok := transitions[state]; ok {
		return true
	}
	for _, nextStates := range transitions {
		if _, ok := nextStates[state]; ok {
			return true
		}
	}
	return IsTerminalState(kind, state)
}

func CanTransition(kind EntityKind, from, to string) bool {
	if from == to {
		return false
	}
	if !IsKnownState(kind, from) || !IsKnownState(kind, to) {
		return false
	}
	if IsTerminalState(kind, from) {
		return false
	}
	nextStates, ok := allowedTransitions[kind][from]
	if !ok {
		return false
	}
	_, ok = nextStates[to]
	return ok
}

func ValidateInitialState(kind EntityKind, state string) error {
	entry, ok := EntryState(kind)
	if !ok {
		return fmt.Errorf("unknown entity kind %q", kind)
	}
	if state != entry {
		return fmt.Errorf("invalid initial state %q for %s: must be %q", state, kind, entry)
	}
	return nil
}

func ValidateTransition(kind EntityKind, from, to string) error {
	if from == to {
		return fmt.Errorf("invalid %s transition: self-transition %q is not allowed", kind, from)
	}
	if _, ok := entryStates[kind]; !ok {
		return fmt.Errorf("unknown entity kind %q", kind)
	}
	if !IsKnownState(kind, from) {
		return fmt.Errorf("unknown %s state %q", kind, from)
	}
	if !IsKnownState(kind, to) {
		return fmt.Errorf("unknown %s state %q", kind, to)
	}
	if IsTerminalState(kind, from) {
		return fmt.Errorf("invalid %s transition from terminal state %q", kind, from)
	}
	if !CanTransition(kind, from, to) {
		valid := ValidNextStates(kind, from)
		if len(valid) > 0 {
			return fmt.Errorf("invalid %s transition %q -> %q; valid transitions from %q: %s", kind, from, to, from, strings.Join(valid, ", "))
		}
		return fmt.Errorf("invalid %s transition %q -> %q", kind, from, to)
	}
	return nil
}

func ValidNextStates(kind EntityKind, from string) []string {
	nextStates, ok := allowedTransitions[kind][from]
	if !ok {
		return nil
	}
	states := make([]string, 0, len(nextStates))
	for s := range nextStates {
		states = append(states, s)
	}
	sort.Strings(states)
	return states
}

func DependencyTerminalStates() map[string]struct{} {
	return map[string]struct{}{
		string(model.TaskStatusDone):       {},
		string(model.TaskStatusNotPlanned): {},
		string(model.TaskStatusDuplicate):  {},
	}
}

func IsTaskDependencySatisfied(status string) bool {
	_, ok := DependencyTerminalStates()[status]
	return ok
}

func NextStates(kind EntityKind, from string) []string {
	if IsTerminalState(kind, from) {
		return nil
	}
	nextMap, ok := allowedTransitions[kind][from]
	if !ok {
		return nil
	}
	var states []string
	for state := range nextMap {
		states = append(states, state)
	}
	return states
}

func ValidateTaskQueuedToReady(dependsOn []string, depStatuses map[string]string) error {
	for _, depID := range dependsOn {
		status, ok := depStatuses[depID]
		if !ok {
			return fmt.Errorf("dependency %s not found", depID)
		}
		if !IsTaskDependencySatisfied(status) {
			return fmt.Errorf("dependency %s is blocking (status: %s) — must reach done, not-planned, or duplicate", depID, status)
		}
	}
	return nil
}
