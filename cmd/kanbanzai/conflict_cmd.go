package main

import (
	"fmt"

	"kanbanzai/internal/core"
	"kanbanzai/internal/service"
)

const conflictUsageText = `Usage: kbz conflict <task-id> <task-id> [<task-id> ...]

Analyse conflict risk between two or more tasks that might run in parallel.
Checks file overlap, dependency ordering, and architectural boundary crossing.
`

func runConflict(args []string, deps dependencies) error {
	if len(args) < 2 {
		return fmt.Errorf("conflict check requires at least two task IDs\n\n%s", conflictUsageText)
	}

	stateRoot := core.StatePath()
	entitySvc := service.NewEntityService(stateRoot)
	conflictSvc := service.NewConflictService(entitySvc, nil, "")

	result, err := conflictSvc.Check(service.ConflictCheckInput{
		TaskIDs: args,
	})
	if err != nil {
		return err
	}

	_, err = fmt.Fprintf(deps.stdout, "conflict analysis\ntasks: %v\noverall risk: %s\n\n",
		result.TaskIDs, result.OverallRisk)
	if err != nil {
		return err
	}

	for _, pair := range result.Pairs {
		_, err = fmt.Fprintf(deps.stdout, "  %s vs %s\n    risk: %s\n    recommendation: %s\n",
			pair.TaskA, pair.TaskB, pair.Risk, pair.Recommendation)
		if err != nil {
			return err
		}
	}

	return nil
}
