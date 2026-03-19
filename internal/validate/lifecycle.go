package validate

import (
	"fmt"

	"kanbanzai/internal/model"
)

type EntityKind = model.EntityKind

const (
	EntityEpic     = model.EntityKindEpic
	EntityFeature  = model.EntityKindFeature
	EntityTask     = model.EntityKindTask
	EntityBug      = model.EntityKindBug
	EntityDecision = model.EntityKindDecision
)

var entryStates = map[EntityKind]string{
	EntityEpic:     string(model.EpicStatusProposed),
	EntityFeature:  string(model.FeatureStatusDraft),
	EntityTask:     string(model.TaskStatusQueued),
	EntityBug:      string(model.BugStatusReported),
	EntityDecision: string(model.DecisionStatusProposed),
}

var terminalStates = map[EntityKind]map[string]struct{}{
	EntityEpic: {
		string(model.EpicStatusDone): {},
	},
	EntityFeature: {
		string(model.FeatureStatusDone):       {},
		string(model.FeatureStatusSuperseded): {},
	},
	EntityTask: {
		string(model.TaskStatusDone): {},
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
}

var allowedTransitions = map[EntityKind]map[string]map[string]struct{}{
	EntityEpic: {
		string(model.EpicStatusProposed): {
			string(model.EpicStatusApproved): {},
		},
		string(model.EpicStatusApproved): {
			string(model.EpicStatusActive): {},
		},
		string(model.EpicStatusActive): {
			string(model.EpicStatusOnHold): {},
			string(model.EpicStatusDone):   {},
		},
		string(model.EpicStatusOnHold): {
			string(model.EpicStatusActive): {},
			string(model.EpicStatusDone):   {},
		},
	},
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
		},
		string(model.FeatureStatusDone): {
			string(model.FeatureStatusSuperseded): {},
		},
	},
	EntityTask: {
		string(model.TaskStatusQueued): {
			string(model.TaskStatusReady): {},
		},
		string(model.TaskStatusReady): {
			string(model.TaskStatusActive): {},
		},
		string(model.TaskStatusActive): {
			string(model.TaskStatusBlocked):     {},
			string(model.TaskStatusNeedsReview): {},
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
		return fmt.Errorf("invalid %s transition %q -> %q", kind, from, to)
	}

	return nil
}
