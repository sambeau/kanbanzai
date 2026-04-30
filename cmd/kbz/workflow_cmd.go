package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sambeau/kanbanzai/internal/cli/render"
	"github.com/sambeau/kanbanzai/internal/cli/status"
	"github.com/sambeau/kanbanzai/internal/core"
	"github.com/sambeau/kanbanzai/internal/id"
	"github.com/sambeau/kanbanzai/internal/resolution"
	"github.com/sambeau/kanbanzai/internal/service"
)

// ─── status ──────────────────────────────────────────────────────────────────

const statusUsageText = `Usage: kbz status [<target>] [options]

Show project overview or entity status.

Arguments:
  <target>                Optional entity ID, plan prefix, or file path

Options:
  --format, -f <fmt>      Output format: human, plain, json (default: human)
`

var validStatusFormats = map[string]bool{
	"human": true,
	"plain": true,
	"json":  true,
}

func validStatusFormatsList() string {
	return "human, plain, json"
}

func runStatus(args []string, deps dependencies) error {
	var format string
	var target string
	var positionalCount int

	// Parse args manually for --format/-f flag and positional target.
	for i := 0; i < len(args); i++ {
		a := args[i]

		// Handle --format=<value> and -f=<value> compact syntax.
		if before, val, found := strings.Cut(a, "="); found {
			switch before {
			case "--format", "-f":
				if format != "" {
					return fmt.Errorf("--format specified more than once")
				}
				format = val
				continue
			default:
				return fmt.Errorf("unknown flag %q\n\n%s", a, statusUsageText)
			}
		}

		switch a {
		case "--format", "-f":
			if format != "" {
				return fmt.Errorf("--format specified more than once")
			}
			if i+1 >= len(args) {
				return fmt.Errorf("--format requires a value (human, plain, json)\n\n%s", statusUsageText)
			}
			i++
			format = args[i]
		default:
			// Anything that starts with -- or - is an unknown flag.
			if strings.HasPrefix(a, "--") || strings.HasPrefix(a, "-") {
				return fmt.Errorf("unknown flag %q\n\n%s", a, statusUsageText)
			}
			// Positional argument.
			positionalCount++
			if positionalCount > 1 {
				return fmt.Errorf("expected at most one target argument, got multiple\n\n%s", statusUsageText)
			}
			target = a
		}
	}

	// Default format.
	if format == "" {
		format = "human"
	}

	// Validate format value.
	if !validStatusFormats[format] {
		return fmt.Errorf("invalid format %q — valid formats: %s", format, validStatusFormatsList())
	}

	// Build renderer with real TTY detection (stdout fd).
	r := buildRenderer(deps)

	// No target: project overview.
	if target == "" {
		return runStatusProjectOverview(format, r, deps)
	}

	// Target provided: disambiguate and route.
	kind := resolution.Disambiguate(target)
	switch kind {
	case resolution.ResolveEntity:
		return runStatusEntity(target, format, r, deps)
	case resolution.ResolvePlanPrefix:
		return runStatusPlanPrefix(target, format, r, deps)
	case resolution.ResolvePath:
		return runStatusPath(target, format, r, deps)
	default:
		// ResolveNone: try entity first, then path, then give up.
		return fmt.Errorf("unrecognised target %q — not an entity ID, plan prefix, or file path", target)
	}
}

// buildRenderer creates a renderer using the dependencies' factory and/or
// falls back to a non-TTY renderer.
func buildRenderer(deps dependencies) *render.Renderer {
	if deps.newRenderer != nil {
		return deps.newRenderer(render.NewTermTTY(int(os.Stdout.Fd())))
	}
	return render.NewRenderer(render.StaticTTY{Value: false})
}

// deriveEntityType extracts the entity type string from a target by inspecting
// its prefix. Returns "" if the prefix is unrecognised.
func deriveEntityType(target string) string {
	idx := strings.Index(target, "-")
	if idx <= 0 {
		return ""
	}
	prefix := target[:idx]
	kind, err := id.EntityKindFromPrefix(prefix)
	if err != nil {
		return ""
	}
	return string(kind)
}

// ─── project overview ───────────────────────────────────────────────────────

func runStatusProjectOverview(format string, r *render.Renderer, deps dependencies) error {
	stateRoot := core.StatePath()
	entitySvc := deps.newEntityService(stateRoot)

	health, err := entitySvc.HealthCheck()
	if err != nil {
		return fmt.Errorf("health check: %w", err)
	}
	wq, err := entitySvc.WorkQueue(service.WorkQueueInput{})
	if err != nil {
		return fmt.Errorf("work queue: %w", err)
	}
	plans, err := entitySvc.ListPlans(service.PlanFilters{})
	if err != nil {
		return fmt.Errorf("list plans: %w", err)
	}

	// Build project input (same as human path).
	input := render.ProjectInput{
		Name: "Kanbanzai",
		Health: &render.StatusHealthSummary{
			Errors:   health.Summary.ErrorCount,
			Warnings: health.Summary.WarningCount,
		},
		WorkQueue: render.ProjectWorkQueue{
			Ready:  len(wq.Queue),
			Active: wq.TotalQueued - len(wq.Queue),
		},
	}
	for _, p := range plans {
		pStatus, _ := p.State["status"].(string)
		fActive, _ := toInt(p.State["features_active"])
		fTotal, _ := toInt(p.State["features_total"])
		input.Plans = append(input.Plans, render.ProjectPlanInput{
			DisplayID:      p.ID,
			Status:         pStatus,
			FeaturesActive: fActive,
			FeaturesTotal:  fTotal,
		})
	}

	// Build attention from health errors/warnings.
	for _, e := range health.Errors {
		input.Attention = append(input.Attention, render.AttentionItem{
			Type:     "error",
			Severity: "error",
			Message:  e.Error(),
		})
	}
	for _, w := range health.Warnings {
		input.Attention = append(input.Attention, render.AttentionItem{
			Type:     "warning",
			Severity: "warning",
			Message:  w.Error(),
		})
	}

	switch format {
	case "json":
		jr := &status.JSONRenderer{}
		b, err := jr.RenderProject(&input)
		if err != nil {
			return err
		}
		_, err = fmt.Fprintln(deps.stdout, string(b))
		return err
	case "plain":
		pr := &status.PlainRenderer{}
		return pr.RenderProject(deps.stdout, &input)
	default: // human
		return r.RenderProject(deps.stdout, &input)
	}
}

// ─── entity status ──────────────────────────────────────────────────────────

func runStatusEntity(target, format string, r *render.Renderer, deps dependencies) error {
	stateRoot := core.StatePath()
	entitySvc := deps.newEntityService(stateRoot)

	entityType := deriveEntityType(target)
	if entityType == "" {
		return fmt.Errorf("cannot determine entity type for target %q", target)
	}

	result, err := entitySvc.Get(entityType, target, "")
	if err != nil {
		// Entity not found in store — informational message, not an error.
		// Per D-6: it's a query tool, so exit 0.
		return nil
	}

	// Human format delegates to the existing human renderer.
	if format == "human" {
		return runStatusEntityHuman(target, entityType, &result, r, deps)
	}

	// Plain and JSON formats: use status renderers.
	status, _ := result.State["status"].(string)
	slug := result.Slug

	switch entityType {
	case "feature":
		return runStatusFeatureFormatted(format, target, &result, status, slug, deps)
	case "task":
		parentFeature, _ := result.State["parent_feature"].(string)
		return runStatusTaskFormatted(format, target, slug, status, parentFeature, deps)
	case "bug":
		severity, _ := result.State["severity"].(string)
		return runStatusBugFormatted(format, target, slug, status, severity, deps)
	default:
		// Fallback for unknown entity types: simple key:value / JSON.
		if format == "json" {
			_, err = fmt.Fprintf(deps.stdout, `{"entity":%q,"status":%q,"format":"json"}`+"\n", target, status)
		} else {
			_, err = fmt.Fprintf(deps.stdout, "%s: %s\n", target, status)
		}
		return err
	}
}

// runStatusFeatureFormatted outputs a feature in plain or JSON format.
func runStatusFeatureFormatted(format, target string, result *service.GetResult, entityStatus, slug string, deps dependencies) error {
	stateRoot := core.StatePath()
	entitySvc := deps.newEntityService(stateRoot)

	summary, _ := result.State["summary"].(string)
	planID, _ := result.State["plan"].(string)
	planName, _ := result.State["plan_name"].(string)

	input := render.FeatureInput{
		DisplayID: target,
		ID:        result.ID,
		Slug:      slug,
		Summary:   summary,
		Status:    entityStatus,
		PlanID:    planID,
		PlanName:  planName,
	}

	// Count tasks.
	tasks, err := entitySvc.List("task")
	if err == nil {
		for _, t := range tasks {
			parentFeature, _ := t.State["parent_feature"].(string)
			if parentFeature == target || parentFeature == result.ID {
				input.TasksTotal++
				ts, _ := t.State["status"].(string)
				switch ts {
				case "active", "developing":
					input.TasksActive++
				case "ready":
					input.TasksReady++
				case "done", "closed":
					input.TasksDone++
				}
			}
		}
	}

	// Extract document references.
	if docs, ok := result.State["documents"]; ok {
		if docList, ok := docs.([]any); ok {
			for _, d := range docList {
				if dm, ok := d.(map[string]any); ok {
					t, _ := dm["type"].(string)
					p, _ := dm["path"].(string)
					s, _ := dm["status"].(string)
					input.Documents = append(input.Documents, render.DocInput{
						Type: t, Path: p, Status: s,
					})
				}
			}
		}
	}

	switch format {
	case "json":
		jr := &status.JSONRenderer{}
		b, err := jr.RenderFeature(&input)
		if err != nil {
			return err
		}
		_, err = fmt.Fprintln(deps.stdout, string(b))
		return err
	default: // plain
		pr := &status.PlainRenderer{}
		return pr.RenderFeature(deps.stdout, &input)
	}
}

// runStatusTaskFormatted outputs a task in plain or JSON format.
func runStatusTaskFormatted(format, id, slug, entityStatus, parentFeature string, deps dependencies) error {
	switch format {
	case "json":
		jr := &status.JSONRenderer{}
		b, err := jr.RenderTask(id, slug, entityStatus, parentFeature, nil)
		if err != nil {
			return err
		}
		_, err = fmt.Fprintln(deps.stdout, string(b))
		return err
	default: // plain
		pr := &status.PlainRenderer{}
		return pr.RenderTask(deps.stdout, id, slug, entityStatus, parentFeature, nil)
	}
}

// runStatusBugFormatted outputs a bug in plain or JSON format.
func runStatusBugFormatted(format, id, slug, entityStatus, severity string, deps dependencies) error {
	switch format {
	case "json":
		jr := &status.JSONRenderer{}
		b, err := jr.RenderBug(id, slug, entityStatus, severity, nil)
		if err != nil {
			return err
		}
		_, err = fmt.Fprintln(deps.stdout, string(b))
		return err
	default: // plain
		pr := &status.PlainRenderer{}
		return pr.RenderBug(deps.stdout, id, slug, entityStatus, severity, nil)
	}
}

func runStatusEntityHuman(target, entityType string, result *service.GetResult, r *render.Renderer, deps dependencies) error {
	status, _ := result.State["status"].(string)
	slug := result.Slug
	summary, _ := result.State["summary"].(string)
	planID, _ := result.State["plan"].(string)
	planName, _ := result.State["plan_name"].(string)

	input := render.FeatureInput{
		DisplayID: target,
		ID:        result.ID,
		Slug:      slug,
		Summary:   summary,
		Status:    status,
		PlanID:    planID,
		PlanName:  planName,
	}

	// Count tasks for this feature from the entity service.
	stateRoot := core.StatePath()
	entitySvc := deps.newEntityService(stateRoot)
	tasks, err := entitySvc.List("task")
	if err == nil {
		for _, t := range tasks {
			parentFeature, _ := t.State["parent_feature"].(string)
			if parentFeature == target || parentFeature == result.ID {
				input.TasksTotal++
				ts, _ := t.State["status"].(string)
				switch ts {
				case "active", "developing":
					input.TasksActive++
				case "ready":
					input.TasksReady++
				case "done", "closed":
					input.TasksDone++
				}
			}
		}
	}

	// Extract document references from state.
	if docs, ok := result.State["documents"]; ok {
		if docList, ok := docs.([]any); ok {
			for _, d := range docList {
				if dm, ok := d.(map[string]any); ok {
					t, _ := dm["type"].(string)
					p, _ := dm["path"].(string)
					s, _ := dm["status"].(string)
					input.Documents = append(input.Documents, render.DocInput{
						Type: t, Path: p, Status: s,
					})
				}
			}
		}
	}

	return r.RenderFeature(deps.stdout, &input)
}

// ─── plan prefix status ─────────────────────────────────────────────────────

func runStatusPlanPrefix(target, format string, r *render.Renderer, deps dependencies) error {
	stateRoot := core.StatePath()
	entitySvc := deps.newEntityService(stateRoot)

	result, err := entitySvc.GetPlan(target)
	if err != nil {
		// Plan not found — informational, exit 0.
		return nil
	}

	if format == "human" {
		return runStatusPlanHuman(target, &result, r, deps)
	}

	// Build plan input for plain/JSON renderers.
	planStatus, _ := result.State["status"].(string)

	input := render.PlanInput{
		DisplayID: target,
		ID:        result.ID,
		Slug:      result.Slug,
		Name:      result.Slug,
		Status:    planStatus,
	}

	features, err := entitySvc.List("feature")
	if err == nil {
		for _, f := range features {
			featPlan, _ := f.State["plan"].(string)
			if featPlan != target && featPlan != result.ID {
				continue
			}
			fStatus, _ := f.State["status"].(string)
			input.Features = append(input.Features, render.PlanFeatureInput{
				DisplayID: f.ID,
				Slug:      f.Slug,
				Status:    fStatus,
			})
		}
	}

	switch format {
	case "json":
		jr := &status.JSONRenderer{}
		b, err := jr.RenderPlan(&input)
		if err != nil {
			return err
		}
		_, err = fmt.Fprintln(deps.stdout, string(b))
		return err
	case "plain":
		pr := &status.PlainRenderer{}
		return pr.RenderPlan(deps.stdout, &input)
	default:
		return runStatusPlanHuman(target, &result, r, deps)
	}
}

func runStatusPlanHuman(target string, planResult *service.ListResult, r *render.Renderer, deps dependencies) error {
	status, _ := planResult.State["status"].(string)

	input := render.PlanInput{
		DisplayID: target,
		ID:        planResult.ID,
		Slug:      planResult.Slug,
		Name:      planResult.Slug,
		Status:    status,
	}

	// Load features belonging to this plan.
	stateRoot := core.StatePath()
	entitySvc := deps.newEntityService(stateRoot)
	features, err := entitySvc.List("feature")
	if err == nil {
		for _, f := range features {
			featPlan, _ := f.State["plan"].(string)
			if featPlan != target && featPlan != planResult.ID {
				continue
			}
			fStatus, _ := f.State["status"].(string)
			input.Features = append(input.Features, render.PlanFeatureInput{
				DisplayID: f.ID,
				Slug:      f.Slug,
				Status:    fStatus,
			})
		}
	}

	tasks, err := entitySvc.List("task")
	if err == nil {
		for _, t := range tasks {
			parentFeature, _ := t.State["parent_feature"].(string)
			// Check if this task's feature belongs to this plan.
			for _, f := range input.Features {
				if parentFeature == f.DisplayID || parentFeature == f.Slug {
					input.TasksTotal++
					ts, _ := t.State["status"].(string)
					switch ts {
					case "active", "developing":
						input.TasksActive++
					case "ready":
						input.TasksReady++
					case "done", "closed":
						input.TasksDone++
					}
					break
				}
			}
		}
	}

	return r.RenderPlan(deps.stdout, &input)
}

// ─── file path status ───────────────────────────────────────────────────────

func runStatusPath(target, format string, r *render.Renderer, deps dependencies) error {
	// Normalise "./" prefix away.
	normalised := strings.TrimPrefix(target, "./")

	// Check file exists on disk.
	fullPath := filepath.Join(".", normalised)
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		return fmt.Errorf("file not found: %s", target)
	}

	stateRoot := core.StatePath()
	repoRoot := "."
	docSvc := deps.newDocumentService(stateRoot, repoRoot)

	lookupPath := strings.TrimPrefix(target, "./")
	doc, err := docSvc.LookupByPath(context.Background(), lookupPath)
	if err != nil {
		return fmt.Errorf("lookup document: %w", err)
	}

	// Unregistered file case.
	if doc.ID == "" {
		switch format {
		case "json":
			jr := &status.JSONRenderer{}
			b, err := jr.RenderDocument(&doc)
			if err != nil {
				return err
			}
			_, err = fmt.Fprintln(deps.stdout, string(b))
			return err
		case "plain":
			pr := &status.PlainRenderer{}
			return pr.RenderDocument(deps.stdout, &doc)
		default:
			_, err = fmt.Fprintf(deps.stdout,
				"File: %s\nStatus: not registered\n\nThis file exists on disk but is not registered in the document store.\nRegister it with:\n  kbz doc register %s\n",
				lookupPath, lookupPath)
			return err
		}
	}

	// Registered file: show document info.
	switch format {
	case "json":
		jr := &status.JSONRenderer{}
		b, err := jr.RenderDocument(&doc)
		if err != nil {
			return err
		}
		_, err = fmt.Fprintln(deps.stdout, string(b))
		return err
	case "plain":
		pr := &status.PlainRenderer{}
		return pr.RenderDocument(deps.stdout, &doc)
	default:
		_, err = fmt.Fprintf(deps.stdout, "Document: %s\nID: %s\nType: %s\nTitle: %s\nStatus: %s\nPath: %s\n",
			doc.ID, doc.ID, doc.Type, doc.Title, doc.Status, doc.Path)
		if doc.Owner != "" {
			fmt.Fprintf(deps.stdout, "Owner: %s\n", doc.Owner)
		}
		if err != nil {
			return err
		}
	}

	// If the document has an owner entity, also show the entity status.
	if doc.Owner != "" {
		entitySvc := deps.newEntityService(stateRoot)
		entityType := deriveEntityType(doc.Owner)
		if entityType == "" {
			return nil
		}
		entity, err := entitySvc.Get(entityType, doc.Owner, "")
		if err != nil {
			return nil
		}
		entityStatus, _ := entity.State["status"].(string)
		switch format {
		case "json":
			// Already inline in JSON via owner field.
		case "plain":
			fmt.Fprintf(deps.stdout, "  %s: %s\n", doc.Owner, entityStatus)
		default:
			fmt.Fprintf(deps.stdout, "\nOwner entity:\n  ID: %s\n  Type: %s\n  Status: %s\n",
				entity.ID, entity.Type, entityStatus)
		}
	}

	return nil
}

// ─── helpers ────────────────────────────────────────────────────────────────

// toInt converts an any to int, returning 0 and false if not an int.
func toInt(v any) (int, bool) {
	switch n := v.(type) {
	case int:
		return n, true
	case int64:
		return int(n), true
	case float64:
		return int(n), true
	default:
		return 0, false
	}
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
