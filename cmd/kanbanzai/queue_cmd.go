package main

import (
	"fmt"
	"strings"

	"kanbanzai/internal/core"
	"kanbanzai/internal/service"
)

const queueUsageText = `Usage: kbz queue [options]

Show the current work queue. Promotes eligible queued tasks to ready status,
then displays all ready tasks sorted by priority.

Options:
  --role <profile>      Filter by role profile
  --conflict-check      Annotate each task with conflict risk against active tasks
`

func runQueue(args []string, deps dependencies) error {
	if wantsHelp(args) {
		fmt.Fprint(deps.stdout, queueUsageText)
		return nil
	}

	var role string
	var conflictCheck bool

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--role":
			if i+1 >= len(args) {
				return fmt.Errorf("--role requires a value")
			}
			i++
			role = args[i]
		case "--conflict-check":
			conflictCheck = true
		default:
			return fmt.Errorf("unknown flag %q\n\n%s", args[i], queueUsageText)
		}
	}

	stateRoot := core.StatePath()
	entitySvc := service.NewEntityService(stateRoot)

	result, err := entitySvc.WorkQueue(service.WorkQueueInput{
		Role:          role,
		ConflictCheck: conflictCheck,
	})
	if err != nil {
		return err
	}

	if result.PromotedCount > 0 {
		fmt.Fprintf(deps.stdout, "Promoted %d task(s) to ready\n", result.PromotedCount)
	}
	if result.TotalQueued > 0 {
		fmt.Fprintf(deps.stdout, "%d task(s) still queued (blocked)\n", result.TotalQueued)
	}

	if len(result.Queue) == 0 {
		fmt.Fprintln(deps.stdout, "No ready tasks in queue")
		return nil
	}

	fmt.Fprintf(deps.stdout, "\nReady tasks: %d\n\n", len(result.Queue))
	for _, item := range result.Queue {
		est := "—"
		if item.Estimate != nil {
			est = fmt.Sprintf("%.1f", *item.Estimate)
		}
		fmt.Fprintf(deps.stdout, "  %s  %s  [est: %s, age: %dd]\n", item.TaskID, item.Slug, est, item.AgeDays)
		fmt.Fprintf(deps.stdout, "    %s\n", item.Summary)
		if item.ParentFeature != "" {
			feat := item.ParentFeature
			if item.FeatureSlug != "" {
				feat = fmt.Sprintf("%s (%s)", item.ParentFeature, item.FeatureSlug)
			}
			fmt.Fprintf(deps.stdout, "    feature: %s\n", feat)
		}
		if conflictCheck && item.ConflictRisk != "" {
			fmt.Fprintf(deps.stdout, "    conflict: %s", item.ConflictRisk)
			if len(item.ConflictWith) > 0 {
				fmt.Fprintf(deps.stdout, " (with %s)", strings.Join(item.ConflictWith, ", "))
			}
			fmt.Fprintln(deps.stdout)
		}
	}

	return nil
}
