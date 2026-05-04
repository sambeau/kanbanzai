package mcp

// nextAction describes the exact MCP tool call an agent should make to resolve
// a gate failure or continue an incomplete workflow. It is embedded in
// composite action responses to replace prose recovery instructions.
type nextAction struct {
	Tool        string         `json:"tool"`
	Action      string         `json:"action"`
	Params      map[string]any `json:"params"`
	Description string         `json:"description"`
}

// nextActionForMissingDocument returns a next_action instructing the agent to
// register a document of the given type for the specified owner.
func nextActionForMissingDocument(docType, owner string) nextAction {
	return nextAction{
		Tool:   "doc",
		Action: "register",
		Params: map[string]any{
			"type":  docType,
			"owner": owner,
		},
		Description: "Register the missing " + docType + " document for " + owner,
	}
}

// nextActionForMissingApproval returns a next_action instructing the agent to
// approve a specific document.
func nextActionForMissingApproval(docID string) nextAction {
	return nextAction{
		Tool:   "doc",
		Action: "approve",
		Params: map[string]any{
			"id": docID,
		},
		Description: "Approve document " + docID,
	}
}

// nextActionForNonTerminalTasks returns a next_action instructing the agent to
// complete the remaining non-terminal tasks before proceeding.
func nextActionForNonTerminalTasks(featureID string) nextAction {
	return nextAction{
		Tool:   "status",
		Action: "",
		Params: map[string]any{
			"id": featureID,
		},
		Description: "Check status of " + featureID + " — non-terminal tasks remain",
	}
}

// nextActionForHumanGate returns a next_action indicating the human must
// approve before the workflow can continue.
func nextActionForHumanGate(stage string) nextAction {
	return nextAction{
		Tool:        "checkpoint",
		Action:      "create",
		Params:      map[string]any{},
		Description: "Human gate at stage '" + stage + "' — human approval required before proceeding",
	}
}

// nextActionForClassification returns a next_action instructing the agent to
// classify and then approve a document (the two-step fallback when publish
// is called without classifications).
func nextActionForClassification(docID string) nextAction {
	return nextAction{
		Tool:   "doc_intel",
		Action: "classify",
		Params: map[string]any{
			"id": docID,
		},
		Description: "Classify document " + docID + ", then call doc(action: \"approve\")",
	}
}
