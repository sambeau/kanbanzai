package validate

import "fmt"

// knowledgeTransitions defines the allowed lifecycle transitions for knowledge entries.
var knowledgeTransitions = map[string]map[string]struct{}{
	"contributed": {
		"confirmed": {},
		"disputed":  {},
		"retired":   {},
	},
	"confirmed": {
		"disputed": {},
		"stale":    {},
		"retired":  {},
	},
	"disputed": {
		"confirmed": {},
		"retired":   {},
	},
	"stale": {
		"confirmed": {},
		"retired":   {},
	},
}

// knowledgeTerminalStates defines terminal states for knowledge entries.
var knowledgeTerminalStates = map[string]struct{}{
	"retired": {},
}

// IsKnownKnowledgeStatus returns true if the status is a recognised knowledge entry status.
func IsKnownKnowledgeStatus(status string) bool {
	if _, ok := knowledgeTransitions[status]; ok {
		return true
	}
	if _, ok := knowledgeTerminalStates[status]; ok {
		return true
	}
	return false
}

// CanTransitionKnowledge reports whether a knowledge lifecycle transition is valid.
func CanTransitionKnowledge(from, to string) bool {
	if from == to {
		return false
	}
	if _, ok := knowledgeTerminalStates[from]; ok {
		return false
	}
	nextStates, ok := knowledgeTransitions[from]
	if !ok {
		return false
	}
	_, ok = nextStates[to]
	return ok
}

// ValidateKnowledgeTransition checks whether a proposed lifecycle transition is valid.
func ValidateKnowledgeTransition(from, to string) error {
	if from == to {
		return fmt.Errorf("invalid knowledge entry transition: self-transition %q is not allowed", from)
	}
	if !IsKnownKnowledgeStatus(from) {
		return fmt.Errorf("unknown knowledge entry status %q", from)
	}
	if !IsKnownKnowledgeStatus(to) {
		return fmt.Errorf("unknown knowledge entry status %q", to)
	}
	if _, ok := knowledgeTerminalStates[from]; ok {
		return fmt.Errorf("invalid knowledge entry transition from terminal status %q", from)
	}
	if !CanTransitionKnowledge(from, to) {
		return fmt.Errorf("invalid knowledge entry transition %q -> %q", from, to)
	}
	return nil
}

// ValidateKnowledgeFields validates the required fields for a knowledge entry.
func ValidateKnowledgeFields(fields map[string]any) error {
	required := []string{"id", "tier", "topic", "scope", "content", "status", "created", "created_by", "updated"}
	for _, f := range required {
		if _, ok := fields[f]; !ok {
			return fmt.Errorf("knowledge entry missing required field: %s", f)
		}
	}

	status, _ := fields["status"].(string)
	if !IsKnownKnowledgeStatus(status) {
		return fmt.Errorf("invalid knowledge entry status: %s", status)
	}

	return nil
}
