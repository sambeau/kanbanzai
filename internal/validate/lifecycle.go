package validate

import (
	"fmt"
	"sort"
	"strings"

	"github.com/sambeau/kanbanzai/internal/model"
)

type EntityKind = model.EntityKind

const (
	EntityPlan     = model.EntityKindPlan
	EntityFeature  = model.EntityKindFeature
	EntityTask     = model.EntityKindTask
	EntityBug      = model.EntityKindBug
	EntityDecision = model.EntityKindDecision
	EntityIncident = model.EntityKindIncident
)

var entryStates = map[EntityKind]string{
	EntityPlan:     string(model.PlanStatusProposed),
	EntityFeature:  phase2FeatureEntryState, // Phase 2 entry state (document-driven lifecycle)
	EntityTask:     string(model.TaskStatusQueued),
	EntityBug:      string(model.BugStatusReported),
	EntityDecision: string(model.DecisionStatusProposed),
	EntityIncident: string(model.IncidentStatusReported),
}

// phase2FeatureEntryState is the entry state for Phase 2 features (document-driven lifecycle).
const phase2FeatureEntryState = "proposed"

var terminalStates = map[EntityKind]map[string]struct{}{
	EntityPlan: {
		string(model.PlanStatusSuperseded): {},
		string(model.PlanStatusCancelled):  {},
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

var allowedTransitions = map[EntityKind]map[string]map[string]struct{}{
	// Plan lifecycle: proposed → designing → active → reviewing → done
	// Terminal: superseded, cancelled (from any non-terminal)
	EntityPlan: {
		string(model.PlanStatusProposed): {
			string(model.PlanStatusDesigning):  {},
			string(model.PlanStatusSuperseded): {},
			string(model.PlanStatusCancelled):  {},
		},
		string(model.PlanStatusDesigning): {
			string(model.PlanStatusActive):     {},
			string(model.PlanStatusSuperseded): {},
			string(model.PlanStatusCancelled):  {},
		},
		string(model.PlanStatusActive): {
			string(model.PlanStatusReviewing):  {},
			string(model.PlanStatusSuperseded): {},
			string(model.PlanStatusCancelled):  {},
		},
		string(model.PlanStatusReviewing): {
			string(model.PlanStatusDone):       {},
			string(model.PlanStatusActive):     {},
			string(model.PlanStatusSuperseded): {},
			string(model.PlanStatusCancelled):  {},
		},
		string(model.PlanStatusDone): {
			string(model.PlanStatusSuperseded): {},
			string(model.PlanStatusCancelled):  {},
		},
	},
	// Feature lifecycle supports both Phase 1 and Phase 2 states for backward compatibility.
	//
	// Phase 1 (legacy): draft → in-review → approved → in-progress → review → done
	// Phase 2 (document-driven): proposed → designing → specifying → dev-planning → developing → reviewing → done
	EntityFeature: {
		// Phase 1 transitions (backward compatibility)
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
			// Phase 2 targets
			string(model.FeatureStatusDeveloping): {},
			string(model.FeatureStatusReviewing):  {},
			string(model.FeatureStatusSuperseded): {},
			string(model.FeatureStatusCancelled):  {},
		},
		// Phase 2 transitions (document-driven lifecycle)
		string(model.FeatureStatusProposed): {
			string(model.FeatureStatusDesigning):  {},
			string(model.FeatureStatusSpecifying): {}, // Shortcut: spec without design
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
			string(model.FeatureStatusDesigning):   {}, // Backward: design superseded
			string(model.FeatureStatusSuperseded):  {},
			string(model.FeatureStatusCancelled):   {},
		},
		string(model.FeatureStatusDevPlanning): {
			string(model.FeatureStatusDeveloping): {},
			string(model.FeatureStatusSpecifying): {}, // Backward: spec superseded
			string(model.FeatureStatusSuperseded): {},
			string(model.FeatureStatusCancelled):  {},
		},
		string(model.FeatureStatusDeveloping): {
			string(model.FeatureStatusReviewing):   {},
			string(model.FeatureStatusDevPlanning): {}, // Backward: dev plan superseded
			string(model.FeatureStatusSuperseded):  {},
			string(model.FeatureStatusCancelled):   {},
		},
		string(model.FeatureStatusReviewing): {
			string(model.FeatureStatusDone):        {},
			string(model.FeatureStatusNeedsRework): {},
			string(model.FeatureStatusSuperseded):  {},
			string(model.FeatureStatusCancelled):   {},
		},
		// Shared terminal states
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
			string(model.TaskStatusReady):       {}, // unclaim / crash recovery
			string(model.TaskStatusBlocked):     {},
			string(model.TaskStatusNeedsReview): {},
			string(model.TaskStatusNeedsRework): {}, // review_task_output fail from active
			string(model.TaskStatusDone):        {}, // direct completion via complete_task
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
			string(model.IncidentStatusClosed):  {}, // trivial / false alarm
		},
		string(model.IncidentStatusTriaged): {
			string(model.IncidentStatusInvestigating): {},
			string(model.IncidentStatusClosed):        {}, // not reproducible
		},
		string(model.IncidentStatusInvestigating): {
			string(model.IncidentStatusRootCauseIdentified): {},
			string(model.IncidentStatusClosed):              {},
		},
		string(model.IncidentStatusRootCauseIdentified): {
			string(model.IncidentStatusMitigated):     {},
			string(model.IncidentStatusInvestigating): {}, // root cause revised
			string(model.IncidentStatusClosed):        {},
		},
		string(model.IncidentStatusMitigated): {
			string(model.IncidentStatusResolved):      {},
			string(model.IncidentStatusInvestigating): {}, // mitigation incomplete
			string(model.IncidentStatusClosed):        {},
		},
		string(model.IncidentStatusResolved): {
			string(model.IncidentStatusClosed): {}, // after RCA approved
		},
	},
}

// EntryState returns the required initial lifecycle state for the entity kind.
func EntryState(kind EntityKind) (string, bool) {
	state, ok := entryStates[kind]
	return state, ok
}

// EntryStateOrPanic returns the required initial lifecycle state for the entity
// kind and panics if the kind is unknown.
func EntryStateOrPanic(kind EntityKind) string {
	state, ok := EntryState(kind)
	if !ok {
		panic(fmt.Sprintf("unknown entity kind %q", kind))
	}
	return state
}

// IsTerminalState reports whether the given state is terminal for the entity kind.
func IsTerminalState(kind EntityKind, state string) bool {
	states, ok := terminalStates[kind]
	if !ok {
		return false
	}

	_, ok = states[state]
	return ok
}

// IsKnownState reports whether the given state exists anywhere in the lifecycle
// definition for the entity kind.
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

// CanTransition reports whether a lifecycle transition is legal.
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

// ValidateInitialState checks that a new entity starts in its required entry state.
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

// ValidateTransition checks whether a proposed lifecycle transition is valid.
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

// ValidNextStates returns the sorted list of states reachable from the given
// state via a single valid transition. Returns nil if the state is terminal,
// unknown, or has no outgoing transitions.
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

// AllStates returns all known states for the given entity kind.
func AllStates(kind EntityKind) []string {
	var states []string
	seen := make(map[string]bool)

	// Add entry state
	if entry, ok := entryStates[kind]; ok {
		states = append(states, entry)
		seen[entry] = true
	}

	// Add states from transitions
	if transitions, ok := allowedTransitions[kind]; ok {
		for from := range transitions {
			if !seen[from] {
				states = append(states, from)
				seen[from] = true
			}
			for to := range transitions[from] {
				if !seen[to] {
					states = append(states, to)
					seen[to] = true
				}
			}
		}
	}

	// Add terminal states
	if terminals, ok := terminalStates[kind]; ok {
		for state := range terminals {
			if !seen[state] {
				states = append(states, state)
				seen[state] = true
			}
		}
	}

	return states
}

// DependencyTerminalStates returns the set of task states that satisfy a dependency.
// A task in one of these states is considered "resolved" as a dependency.
func DependencyTerminalStates() map[string]struct{} {
	return map[string]struct{}{
		string(model.TaskStatusDone):       {},
		string(model.TaskStatusNotPlanned): {},
		string(model.TaskStatusDuplicate):  {},
	}
}

// IsTaskDependencySatisfied returns true if the given task status satisfies a dependency.
func IsTaskDependencySatisfied(status string) bool {
	_, ok := DependencyTerminalStates()[status]
	return ok
}

// NextStates returns the valid next states from the given state for the entity kind.
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

// ValidateTaskQueuedToReady validates the dependency gate for queued→ready task transitions.
// dependsOn is the list of task IDs this task depends on.
// depStatuses maps dependency task IDs to their current status strings.
// Returns an error if any dependency is not in a terminal state.
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
