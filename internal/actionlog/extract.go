package actionlog

// StageLookup abstracts entity loading for stage resolution.
type StageLookup interface {
	GetEntityKindAndParent(entityID string) (kind, parentFeatureID string, err error)
	GetFeatureStage(featureID string) (string, error)
}

// ExtractEntityID extracts an entity ID from tool call args.
// Checks "id", then "entity_id", then "task_id". Returns nil if none found.
func ExtractEntityID(args map[string]any) *string {
	for _, key := range []string{"id", "entity_id", "task_id"} {
		if v, ok := args[key]; ok {
			if s, ok := v.(string); ok && s != "" {
				result := s
				return &result
			}
		}
	}
	return nil
}

// ResolveStage determines the lifecycle stage for a log entry by looking up
// the entity and its parent feature. Returns nil on any failure.
func ResolveStage(entityID *string, lookup StageLookup) *string {
	if entityID == nil || lookup == nil {
		return nil
	}

	kind, parentFeatureID, err := lookup.GetEntityKindAndParent(*entityID)
	if err != nil {
		return nil
	}

	// Features: look up their stage directly.
	if kind == "feature" {
		stage, err := lookup.GetFeatureStage(*entityID)
		if err != nil {
			return nil
		}
		result := stage
		return &result
	}

	// Tasks, bugs, etc: look up the parent feature stage.
	if parentFeatureID == "" {
		return nil
	}
	stage, err := lookup.GetFeatureStage(parentFeatureID)
	if err != nil {
		return nil
	}
	result := stage
	return &result
}
