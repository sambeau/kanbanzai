package main

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/sambeau/kanbanzai/internal/core"
	"github.com/sambeau/kanbanzai/internal/service"
)

const taskUsageText = `Usage: kbz task <subcommand> [options]

Subcommands:
  review <task-id>   Run worker review on a completed task
    --files <path,...>       Paths of files produced or modified
    --summary "<text>"       Agent's description of what was done
`

func runTask(args []string, deps dependencies) error {
	if len(args) == 0 || wantsHelp(args) {
		fmt.Fprint(deps.stdout, taskUsageText)
		return nil
	}

	switch args[0] {
	case "review":
		return runTaskReview(args[1:], deps)
	default:
		return fmt.Errorf("unknown task subcommand %q\n\n%s", args[0], taskUsageText)
	}
}

func runTaskReview(args []string, deps dependencies) error {
	if len(args) == 0 {
		return fmt.Errorf("missing task ID\n\nUsage: kbz task review <task-id> [--files <path,...>] [--summary \"<text>\"]")
	}

	taskID := args[0]
	remaining := args[1:]

	var outputFiles []string
	var outputSummary string

	for i := 0; i < len(remaining); i++ {
		switch remaining[i] {
		case "--files":
			if i+1 >= len(remaining) {
				return fmt.Errorf("--files requires a value")
			}
			i++
			outputFiles = strings.Split(remaining[i], ",")
		case "--summary":
			if i+1 >= len(remaining) {
				return fmt.Errorf("--summary requires a value")
			}
			i++
			outputSummary = remaining[i]
		default:
			return fmt.Errorf("unknown flag %q\n\nUsage: kbz task review <task-id> [--files <path,...>] [--summary \"<text>\"]", remaining[i])
		}
	}

	stateRoot := core.StatePath()
	repoRoot := "."
	indexRoot := filepath.Join(core.InstanceRootDir, "index")

	entitySvc := service.NewEntityService(stateRoot)
	intelSvc := service.NewIntelligenceService(indexRoot, repoRoot)
	reviewSvc := service.NewReviewService(entitySvc, intelSvc, repoRoot)

	result, err := reviewSvc.ReviewTaskOutput(service.ReviewInput{
		TaskID:        taskID,
		OutputFiles:   outputFiles,
		OutputSummary: outputSummary,
	})
	if err != nil {
		return err
	}

	printReviewResult(deps.stdout, result)
	return nil
}

func printReviewResult(w interface{ Write([]byte) (int, error) }, result service.ReviewResult) {
	fmt.Fprintf(w, "Task:     %s (%s)\n", result.TaskID, result.TaskSlug)
	fmt.Fprintf(w, "Status:   %s\n", result.Status)
	fmt.Fprintf(w, "Findings: %d total, %d blocking\n", result.TotalFindings, result.BlockingCount)

	if len(result.Findings) > 0 {
		fmt.Fprintln(w)
		for _, f := range result.Findings {
			icon := "⚠"
			if f.Severity == "error" {
				icon = "✗"
			}
			fmt.Fprintf(w, "  %s [%s] %s: %s\n", icon, f.Severity, f.Type, f.Detail)
		}
	}
}
