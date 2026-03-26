package main

import (
	"fmt"
	"io"
	"strings"
	"time"

	"kanbanzai/internal/checkpoint"
	"kanbanzai/internal/core"
	"kanbanzai/internal/service"
)

const featureUsageText = `Usage: kbz feature <subcommand> [options]

Subcommands:
  decompose <feature-id>             Propose a task decomposition for a feature
  decompose <feature-id> --confirm   Write proposed tasks after human confirmation
`

func runFeature(args []string, deps dependencies) error {
	if len(args) == 0 || wantsHelp(args) {
		fmt.Fprint(deps.stdout, featureUsageText)
		return nil
	}

	switch args[0] {
	case "decompose":
		return runFeatureDecompose(args[1:], deps)
	default:
		return fmt.Errorf("unknown feature subcommand %q\n\n%s", args[0], featureUsageText)
	}
}

func runFeatureDecompose(args []string, deps dependencies) error {
	if len(args) == 0 {
		return fmt.Errorf("missing feature ID\n\nUsage: kbz feature decompose <feature-id> [--confirm]")
	}

	featureID := args[0]
	remaining := args[1:]

	// Check for --confirm flag.
	confirm := false
	for _, arg := range remaining {
		if arg == "--confirm" {
			confirm = true
		}
	}

	stateRoot := core.StatePath()
	repoRoot := "."
	entitySvc := service.NewEntityService(stateRoot)
	docSvc := service.NewDocumentService(stateRoot, repoRoot)
	decomposeSvc := service.NewDecomposeService(entitySvc, docSvc)

	result, err := decomposeSvc.DecomposeFeature(service.DecomposeInput{
		FeatureID: featureID,
	})
	if err != nil {
		return err
	}

	// Print the proposal.
	printProposal(deps.stdout, result)

	if !confirm {
		return nil
	}

	// --confirm: create a human checkpoint then create tasks.
	return runFeatureDecomposeConfirm(deps, entitySvc, result)
}

func printProposal(w io.Writer, result service.DecomposeResult) {
	fmt.Fprintf(w, "Feature:  %s (%s)\n", result.FeatureID, result.FeatureSlug)
	fmt.Fprintf(w, "Spec:     %s\n", result.SpecDocumentID)
	fmt.Fprintf(w, "Tasks:    %d\n", result.Proposal.TotalTasks)
	if result.Proposal.EstimatedTotal != nil {
		fmt.Fprintf(w, "Estimate: %.0f points\n", *result.Proposal.EstimatedTotal)
	}
	fmt.Fprintln(w)

	if len(result.Proposal.Slices) > 0 {
		fmt.Fprintf(w, "Slices: %s\n", strings.Join(result.Proposal.Slices, ", "))
	}
	if len(result.Proposal.Warnings) > 0 {
		fmt.Fprintln(w, "Warnings:")
		for _, warn := range result.Proposal.Warnings {
			fmt.Fprintf(w, "  ⚠ %s\n", warn)
		}
	}
	fmt.Fprintln(w)

	fmt.Fprintln(w, "Proposed tasks:")
	for i, task := range result.Proposal.Tasks {
		fmt.Fprintf(w, "  %d. [%s] %s\n", i+1, task.Slug, task.Summary)
		if task.Rationale != "" {
			fmt.Fprintf(w, "     Rationale: %s\n", task.Rationale)
		}
		if task.Estimate != nil {
			fmt.Fprintf(w, "     Estimate: %.0f points\n", *task.Estimate)
		}
		if len(task.DependsOn) > 0 {
			fmt.Fprintf(w, "     Depends on: %s\n", strings.Join(task.DependsOn, ", "))
		}
	}
	fmt.Fprintln(w)

	fmt.Fprintf(w, "Guidance applied: %s\n", strings.Join(result.GuidanceApplied, ", "))
}

func runFeatureDecomposeConfirm(deps dependencies, entitySvc *service.EntityService, result service.DecomposeResult) error {
	// Create a formal human checkpoint before writing any tasks.
	stateRoot := core.StatePath()
	chkStore := checkpoint.NewStore(stateRoot)

	chkRecord := checkpoint.Record{
		Question: fmt.Sprintf("Confirm creation of %d tasks for feature %s?",
			result.Proposal.TotalTasks, result.FeatureID),
		Context: fmt.Sprintf("Feature: %s (%s)\nSpec: %s\nTasks: %d",
			result.FeatureID, result.FeatureSlug, result.SpecDocumentID, result.Proposal.TotalTasks),
		OrchestrationSummary: "CLI decompose --confirm: awaiting human approval before task creation",
		Status:               checkpoint.StatusPending,
		CreatedAt:            time.Now().UTC(),
		CreatedBy:            "cli",
	}

	created, err := chkStore.Create(chkRecord)
	if err != nil {
		return fmt.Errorf("create checkpoint: %w", err)
	}

	fmt.Fprintln(deps.stdout, "---")
	fmt.Fprintf(deps.stdout, "Checkpoint %s created.\n", created.ID)
	fmt.Fprintf(deps.stdout, "About to create %d tasks for feature %s.\n", result.Proposal.TotalTasks, result.FeatureID)
	fmt.Fprint(deps.stdout, "Confirm? (y/n): ")

	var response string
	if _, err := fmt.Fscan(deps.stdin, &response); err != nil {
		return fmt.Errorf("read confirmation: %w", err)
	}

	response = strings.TrimSpace(strings.ToLower(response))

	// Record the response on the checkpoint.
	now := time.Now().UTC()
	created.Status = checkpoint.StatusResponded
	created.RespondedAt = &now
	created.Response = &response
	if _, err := chkStore.Update(created); err != nil {
		return fmt.Errorf("update checkpoint: %w", err)
	}

	if response != "y" && response != "yes" {
		fmt.Fprintln(deps.stdout, "Aborted.")
		return nil
	}

	// Create tasks from the proposal.
	tasksCreated := 0
	for _, task := range result.Proposal.Tasks {
		_, err := entitySvc.CreateTask(service.CreateTaskInput{
			ParentFeature: result.FeatureID,
			Slug:          task.Slug,
			Summary:       task.Summary,
		})
		if err != nil {
			return fmt.Errorf("create task %q: %w", task.Slug, err)
		}
		tasksCreated++
	}

	fmt.Fprintf(deps.stdout, "Created %d tasks.\n", tasksCreated)
	return nil
}
