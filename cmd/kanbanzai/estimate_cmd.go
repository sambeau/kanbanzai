package main

import (
	"fmt"
	"strconv"

	"kanbanzai/internal/core"
	"kanbanzai/internal/service"
)

const estimateUsageText = `Usage: kbz estimate <entity-id> [<points>]

Query or set a story point estimate for a task, feature, bug, or plan.

  kbz estimate <entity-id>           Query the current estimate
  kbz estimate <entity-id> <points>  Set the estimate (Modified Fibonacci: 0, 0.5, 1, 2, 3, 5, 8, 13, 20, 40, 100)
`

func runEstimate(args []string, deps dependencies) error {
	if len(args) == 0 {
		return fmt.Errorf("missing entity ID\n\n%s", estimateUsageText)
	}

	entityID := args[0]
	stateRoot := core.StatePath()
	entitySvc := service.NewEntityService(stateRoot)

	// Query mode: no second argument.
	if len(args) == 1 {
		result, err := entitySvc.Get("task", entityID, "")
		if err != nil {
			// Try other entity types.
			for _, t := range []string{"feature", "bug", "plan"} {
				result, err = entitySvc.Get(t, entityID, "")
				if err == nil {
					break
				}
			}
			if err != nil {
				return fmt.Errorf("entity not found: %s", entityID)
			}
		}

		est := service.GetEstimateFromFields(result.State)
		if est == nil {
			_, err = fmt.Fprintf(deps.stdout, "entity: %s\nestimate: (not set)\n", result.ID)
		} else {
			_, err = fmt.Fprintf(deps.stdout, "entity: %s\nestimate: %.1f\n", result.ID, *est)
		}
		return err
	}

	// Set mode: second argument is the points value.
	points, err := strconv.ParseFloat(args[1], 64)
	if err != nil {
		return fmt.Errorf("invalid points value %q: must be a number", args[1])
	}

	// Detect entity type from prefix.
	entityType := detectEntityTypeFromID(entityID)

	_, warning, err := entitySvc.SetEstimate(entityType, entityID, points)
	if err != nil {
		return err
	}

	_, err = fmt.Fprintf(deps.stdout, "set estimate\nentity: %s\nestimate: %.1f\n", entityID, points)
	if err != nil {
		return err
	}
	if warning != "" {
		_, err = fmt.Fprintf(deps.stdout, "warning: %s\n", warning)
	}
	return err
}

// detectEntityTypeFromID infers entity type from ID prefix (best-effort).
func detectEntityTypeFromID(entityID string) string {
	switch {
	case len(entityID) >= 5 && entityID[:5] == "TASK-":
		return "task"
	case len(entityID) >= 5 && entityID[:5] == "FEAT-":
		return "feature"
	case len(entityID) >= 4 && entityID[:4] == "BUG-":
		return "bug"
	default:
		return "task"
	}
}
