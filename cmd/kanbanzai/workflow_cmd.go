package main

import (
	"fmt"
	"strings"

	"kanbanzai/internal/core"
	"kanbanzai/internal/id"
	"kanbanzai/internal/service"
)

// ─── status ──────────────────────────────────────────────────────────────────

func runStatus(args []string, deps dependencies) error {
	// Simple project overview: health check + queue summary.
	if err := runHealth(deps); err != nil {
		return err
	}

	stateRoot := core.StatePath()
	entitySvc := service.NewEntityService(stateRoot)
	result, err := entitySvc.WorkQueue(service.WorkQueueInput{})
	if err != nil {
		return err
	}

	fmt.Fprintln(deps.stdout)
	fmt.Fprintf(deps.stdout, "Work queue: %d ready, %d queued\n", len(result.Queue), result.TotalQueued)
	return nil
}

// ─── next ────────────────────────────────────────────────────────────────────

func runNextCmd(args []string, deps dependencies) error {
	stateRoot := core.StatePath()
	entitySvc := service.NewEntityService(stateRoot)

	// With task ID: claim the task (transition to active) and print context.
	if len(args) > 0 && !strings.HasPrefix(args[0], "--") {
		return runNextClaim(args[0], entitySvc, deps)
	}

	// No task ID: show the work queue.
	return runQueue(args, deps)
}

func runNextClaim(taskID string, entitySvc *service.EntityService, deps dependencies) error {
	result, err := entitySvc.Get("task", taskID, "")
	if err != nil {
		return fmt.Errorf("task not found: %s", taskID)
	}

	status, _ := result.State["status"].(string)
	if status != "ready" && status != "active" {
		return fmt.Errorf("task %s is in status %q — only ready or active tasks can be claimed", taskID, status)
	}

	if status == "ready" {
		_, err = entitySvc.UpdateStatus(service.UpdateStatusInput{
			Type:   "task",
			ID:     result.ID,
			Slug:   result.Slug,
			Status: "active",
		})
		if err != nil {
			return fmt.Errorf("transition task to active: %w", err)
		}
	}

	summary, _ := result.State["summary"].(string)
	parentFeature, _ := result.State["parent_feature"].(string)

	_, err = fmt.Fprintf(deps.stdout,
		"## Task: %s (%s)\n\n**Summary:** %s\n**Status:** active\n**Parent Feature:** %s\n\n"+
			"Complete this task and run:\n  kbz finish %s --summary \"<description of what was done>\"\n",
		id.FormatFullDisplay(result.ID), result.Slug,
		summary,
		parentFeature,
		result.ID,
	)
	return err
}

// ─── finish ──────────────────────────────────────────────────────────────────

const finishUsageText = `Usage: kbz finish <task-id> --summary "<text>" [options]

Complete a task and record what was accomplished.

Options:
  --summary <text>       Brief description of what was done (required)
  --files <path,...>     Comma-separated list of files created or modified
  --verification <text>  Testing or verification performed
  --status <status>      Target status: done (default) or needs-review
`

func runFinish(args []string, deps dependencies) error {
	if len(args) == 0 {
		return fmt.Errorf("missing task ID\n\n%s", finishUsageText)
	}

	taskID := args[0]
	remaining := args[1:]

	var summary, verification, toStatus string
	var filesModified []string

	for i := 0; i < len(remaining); i++ {
		switch remaining[i] {
		case "--summary":
			if i+1 >= len(remaining) {
				return fmt.Errorf("--summary requires a value")
			}
			i++
			summary = remaining[i]
		case "--files":
			if i+1 >= len(remaining) {
				return fmt.Errorf("--files requires a value")
			}
			i++
			filesModified = strings.Split(remaining[i], ",")
		case "--verification":
			if i+1 >= len(remaining) {
				return fmt.Errorf("--verification requires a value")
			}
			i++
			verification = remaining[i]
		case "--status":
			if i+1 >= len(remaining) {
				return fmt.Errorf("--status requires a value")
			}
			i++
			toStatus = remaining[i]
		default:
			return fmt.Errorf("unknown flag %q\n\n%s", remaining[i], finishUsageText)
		}
	}

	if summary == "" {
		return fmt.Errorf("--summary is required\n\n%s", finishUsageText)
	}

	stateRoot := core.StatePath()
	entitySvc := service.NewEntityService(stateRoot)
	knowledgeSvc := service.NewKnowledgeService(stateRoot)
	dispatchSvc := service.NewDispatchService(entitySvc, knowledgeSvc)

	// Auto-transition ready → active if needed.
	task, err := entitySvc.Get("task", taskID, "")
	if err != nil {
		return fmt.Errorf("task not found: %s", taskID)
	}
	taskStatus, _ := task.State["status"].(string)
	if taskStatus == "ready" {
		_, err = entitySvc.UpdateStatus(service.UpdateStatusInput{
			Type:   "task",
			ID:     task.ID,
			Slug:   task.Slug,
			Status: "active",
		})
		if err != nil {
			return fmt.Errorf("auto-transition to active: %w", err)
		}
	}

	result, err := dispatchSvc.CompleteTask(service.CompleteInput{
		TaskID:                taskID,
		Summary:               summary,
		ToStatus:              toStatus,
		FilesModified:         filesModified,
		VerificationPerformed: verification,
	})
	if err != nil {
		return err
	}

	taskResult := result.Task
	finalStatus, _ := taskResult["status"].(string)
	_, err = fmt.Fprintf(deps.stdout, "completed task\nid: %s\nstatus: %s\n",
		taskID, finalStatus)
	if err != nil {
		return err
	}

	if result.KnowledgeContributions.TotalAccepted > 0 {
		_, err = fmt.Fprintf(deps.stdout, "knowledge entries contributed: %d\n",
			result.KnowledgeContributions.TotalAccepted)
		if err != nil {
			return err
		}
	}

	for _, u := range result.UnblockedTasks {
		_, err = fmt.Fprintf(deps.stdout, "unblocked: %s → %s\n", u.TaskID, u.Status)
		if err != nil {
			return err
		}
	}

	return nil
}

// ─── handoff ─────────────────────────────────────────────────────────────────

const handoffUsageText = `Usage: kbz handoff <task-id>

Print a sub-agent prompt for the given task, including task summary,
parent feature context, and completion instructions.
`

func runHandoff(args []string, deps dependencies) error {
	if len(args) == 0 {
		return fmt.Errorf("missing task ID\n\n%s", handoffUsageText)
	}

	taskID := args[0]
	stateRoot := core.StatePath()
	entitySvc := service.NewEntityService(stateRoot)

	task, err := entitySvc.Get("task", taskID, "")
	if err != nil {
		return fmt.Errorf("task not found: %s", taskID)
	}

	summary, _ := task.State["summary"].(string)
	status, _ := task.State["status"].(string)
	parentFeature, _ := task.State["parent_feature"].(string)

	var featureSummary string
	if parentFeature != "" {
		feat, ferr := entitySvc.Get("feature", parentFeature, "")
		if ferr == nil {
			featureSummary, _ = feat.State["summary"].(string)
		}
	}

	_, err = fmt.Fprintf(deps.stdout, "# Task Handoff: %s\n\n## Task\n- **ID:** %s\n- **Slug:** %s\n- **Status:** %s\n- **Summary:** %s\n\n## Parent Feature\n- **ID:** %s\n- **Summary:** %s\n\n## Instructions\n\nComplete this task according to the task summary above.\n\nWhen done, run:\n  kbz finish %s --summary \"<brief description of what was accomplished>\"\n",
		id.FormatFullDisplay(task.ID),
		id.FormatFullDisplay(task.ID),
		task.Slug,
		status,
		summary,
		parentFeature,
		featureSummary,
		task.ID,
	)
	return err
}
