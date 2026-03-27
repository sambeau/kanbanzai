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

const decomposeUsageText = `Usage: kbz decompose <feature-id> [options]

Propose a task decomposition for a feature based on its linked specification.

Options:
  --apply     Write proposed tasks after human confirmation
  --review    Review an existing proposal against the specification
  --slice     Perform vertical slice analysis instead of full decomposition
`

func runDecompose(args []string, deps dependencies) error {
	if len(args) == 0 {
		return fmt.Errorf("missing feature ID\n\n%s", decomposeUsageText)
	}

	featureID := args[0]
	if strings.HasPrefix(featureID, "--") {
		return fmt.Errorf("unknown flag %q\n\n%s", featureID, decomposeUsageText)
	}
	remaining := args[1:]

	var apply, review, slice bool
	for _, arg := range remaining {
		switch arg {
		case "--apply":
			apply = true
		case "--review":
			review = true
		case "--slice":
			slice = true
		default:
			return fmt.Errorf("unknown flag %q\n\n%s", arg, decomposeUsageText)
		}
	}

	stateRoot := core.StatePath()
	repoRoot := "."
	entitySvc := service.NewEntityService(stateRoot)
	docSvc := service.NewDocumentService(stateRoot, repoRoot)
	decomposeSvc := service.NewDecomposeService(entitySvc, docSvc)

	if slice {
		return runDecomposeSlice(featureID, decomposeSvc, deps)
	}

	result, err := decomposeSvc.DecomposeFeature(service.DecomposeInput{
		FeatureID: featureID,
	})
	if err != nil {
		return err
	}

	if review {
		return runDecomposeReview(featureID, result, decomposeSvc, deps)
	}

	// Print the proposal.
	printProposal(deps.stdout, result)

	if !apply {
		return nil
	}

	// --apply: create a human checkpoint then create tasks.
	return runDecomposeApply(deps, entitySvc, result)
}

// ─── slice ───────────────────────────────────────────────────────────────────

func runDecomposeSlice(featureID string, decomposeSvc *service.DecomposeService, deps dependencies) error {
	result, err := decomposeSvc.SliceAnalysis(service.SliceAnalysisInput{
		FeatureID: featureID,
	})
	if err != nil {
		return err
	}

	fmt.Fprintf(deps.stdout, "Feature:  %s (%s)\n", result.FeatureID, result.FeatureSlug)
	fmt.Fprintf(deps.stdout, "Slices:   %d\n\n", len(result.Slices))

	for i, s := range result.Slices {
		fmt.Fprintf(deps.stdout, "  %d. %s\n", i+1, s.Name)
		if s.Rationale != "" {
			fmt.Fprintf(deps.stdout, "     %s\n", s.Rationale)
		}
		if len(s.Layers) > 0 {
			fmt.Fprintf(deps.stdout, "     Layers: %s\n", strings.Join(s.Layers, ", "))
		}
		if len(s.Outcomes) > 0 {
			fmt.Fprintf(deps.stdout, "     Outcomes: %s\n", strings.Join(s.Outcomes, "; "))
		}
		if s.Estimate != "" {
			fmt.Fprintf(deps.stdout, "     Estimate: %s\n", s.Estimate)
		}
		if len(s.DependsOn) > 0 {
			fmt.Fprintf(deps.stdout, "     Depends on: %s\n", strings.Join(s.DependsOn, ", "))
		}
	}

	if result.AnalysisNotes != "" {
		fmt.Fprintln(deps.stdout)
		fmt.Fprintf(deps.stdout, "Notes: %s\n", result.AnalysisNotes)
	}

	return nil
}

// ─── review ──────────────────────────────────────────────────────────────────

func runDecomposeReview(featureID string, result service.DecomposeResult, decomposeSvc *service.DecomposeService, deps dependencies) error {
	reviewResult, err := decomposeSvc.ReviewProposal(service.DecomposeReviewInput{
		FeatureID: featureID,
		Proposal:  result.Proposal,
	})
	if err != nil {
		return err
	}

	fmt.Fprintf(deps.stdout, "Review:   %s\n", reviewResult.Status)
	fmt.Fprintf(deps.stdout, "Findings: %d (blocking: %d)\n\n", reviewResult.TotalFindings, reviewResult.BlockingCount)

	for _, f := range reviewResult.Findings {
		icon := "⚠"
		if f.Type == "gap" || f.Type == "cycle" {
			icon = "✗"
		}
		fmt.Fprintf(deps.stdout, "  %s [%s] %s\n", icon, f.Type, f.Detail)
	}

	return nil
}

// ─── apply ───────────────────────────────────────────────────────────────────

func runDecomposeApply(deps dependencies, entitySvc *service.EntityService, result service.DecomposeResult) error {
	stateRoot := core.StatePath()
	chkStore := checkpoint.NewStore(stateRoot)

	chkRecord := checkpoint.Record{
		Question: fmt.Sprintf("Confirm creation of %d tasks for feature %s?",
			result.Proposal.TotalTasks, result.FeatureID),
		Context: fmt.Sprintf("Feature: %s (%s)\nSpec: %s\nTasks: %d",
			result.FeatureID, result.FeatureSlug, result.SpecDocumentID, result.Proposal.TotalTasks),
		OrchestrationSummary: "CLI decompose --apply: awaiting human approval before task creation",
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

// ─── shared ──────────────────────────────────────────────────────────────────

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
