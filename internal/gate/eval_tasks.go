package gate

import (
	"fmt"

	"github.com/sambeau/kanbanzai/internal/binding"
	"github.com/sambeau/kanbanzai/internal/model"
)

var terminalStates = map[string]bool{
	"done":        true,
	"not-planned": true,
	"duplicate":   true,
}

func init() {
	RegisterEvaluator("tasks", evalTasks)
}

func evalTasks(prereqs *binding.Prerequisites, stage string, ctx PrereqEvalContext) []GateResult {
	if prereqs.Tasks == nil {
		return nil
	}

	tasks, err := ctx.EntitySvc.List("task")
	if err != nil {
		return []GateResult{{
			Stage:     stage,
			Satisfied: false,
			Reason:    fmt.Sprintf("failed to list tasks: %v", err),
			Source:    "registry",
		}}
	}

	childTasks := filterChildTasks(tasks, ctx.Feature)

	var results []GateResult

	if prereqs.Tasks.MinCount != nil {
		minCount := *prereqs.Tasks.MinCount
		if len(childTasks) >= minCount {
			results = append(results, GateResult{
				Stage:     stage,
				Satisfied: true,
				Reason:    fmt.Sprintf("feature has %d child task(s) (minimum %d)", len(childTasks), minCount),
				Source:    "registry",
			})
		} else {
			results = append(results, GateResult{
				Stage:     stage,
				Satisfied: false,
				Reason:    fmt.Sprintf("feature has %d child task(s) but minimum %d required", len(childTasks), minCount),
				Source:    "registry",
			})
		}
	}

	if prereqs.Tasks.AllTerminal != nil && *prereqs.Tasks.AllTerminal {
		nonTerminal := 0
		for _, t := range childTasks {
			status, _ := t.State["status"].(string)
			if !terminalStates[status] {
				nonTerminal++
			}
		}

		if nonTerminal == 0 {
			results = append(results, GateResult{
				Stage:     stage,
				Satisfied: true,
				Reason:    fmt.Sprintf("all %d child task(s) are in terminal state", len(childTasks)),
				Source:    "registry",
			})
		} else {
			results = append(results, GateResult{
				Stage:     stage,
				Satisfied: false,
				Reason:    fmt.Sprintf("%d of %d child task(s) not in terminal state", nonTerminal, len(childTasks)),
				Source:    "registry",
			})
		}
	}

	return results
}

func filterChildTasks(tasks []EntityResult, feature *model.Feature) []EntityResult {
	var children []EntityResult
	for _, t := range tasks {
		parent, _ := t.State["parent_feature"].(string)
		if parent == feature.ID {
			children = append(children, t)
		}
	}
	return children
}
