package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

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

	// No target: project overview (existing behaviour).
	if target == "" {
		if err := runHealth(deps); err != nil {
			return err
		}

		stateRoot := core.StatePath()
		entitySvc := deps.newEntityService(stateRoot)
		result, err := entitySvc.WorkQueue(service.WorkQueueInput{})
		if err != nil {
			return err
		}

		fmt.Fprintln(deps.stdout)
		fmt.Fprintf(deps.stdout, "Work queue: %d ready, %d queued\n", len(result.Queue), result.TotalQueued)
		return nil
	}

	// Target provided: disambiguate and route.
	kind := resolution.Disambiguate(target)
	switch kind {
	case resolution.ResolveEntity:
		return runStatusEntity(target, format, deps)
	case resolution.ResolvePlanPrefix:
		return runStatusPlanPrefix(target, format, deps)
	case resolution.ResolvePath:
		return runStatusPath(target, format, deps)
	default:
		// ResolveNone: try entity first, then path, then give up.
		return fmt.Errorf("unrecognised target %q — not an entity ID, plan prefix, or file path", target)
	}
}

// deriveEntityType extracts the entity type string from a target by inspecting
// its prefix. Returns "" if the prefix is unrecognised.
func deriveEntityType(target string) string {
	// Strip the leading prefix (e.g. "FEAT-042" -> "FEAT").
	idx := strings.Index(target, "-")
	if idx <= 0 {
		return ""
	}
	prefix := target[:idx]

	// Map to entity type string used by the service layer.
	kind, err := id.EntityKindFromPrefix(prefix)
	if err != nil {
		return ""
	}
	return string(kind)
}

// runStatusEntity shows status for a resolved entity target.
// Stub rendering — full rendering is the job of F3/F4.
func runStatusEntity(target, format string, deps dependencies) error {
	stateRoot := core.StatePath()
	entitySvc := deps.newEntityService(stateRoot)

	entityType := deriveEntityType(target)
	if entityType == "" {
		return fmt.Errorf("cannot determine entity type for target %q", target)
	}

	result, err := entitySvc.Get(entityType, target, "")
	if err != nil {
		return fmt.Errorf("entity not found: %s: %w", target, err)
	}

	status, _ := result.State["status"].(string)
	switch format {
	case "json":
		_, err = fmt.Fprintf(deps.stdout, `{"entity":%q,"status":%q,"format":"json"}
`, target, status)
	case "plain":
		_, err = fmt.Fprintf(deps.stdout, "%s: %s\n", target, status)
	default:
		_, err = fmt.Fprintf(deps.stdout, "Entity: %s\nStatus: %s\n", target, status)
	}
	return err
}

// runStatusPlanPrefix shows status for a bare plan prefix (e.g. "P1").
// Stub rendering — full rendering is the job of F3/F4.
func runStatusPlanPrefix(target, format string, deps dependencies) error {
	stateRoot := core.StatePath()
	entitySvc := deps.newEntityService(stateRoot)

	result, err := entitySvc.GetPlan(target)
	if err != nil {
		return fmt.Errorf("plan not found for prefix %q: %w", target, err)
	}

	status, _ := result.State["status"].(string)
	switch format {
	case "json":
		_, err = fmt.Fprintf(deps.stdout, `{"plan_prefix":%q,"plan_id":%q,"status":%q,"format":"json"}
`, target, result.ID, status)
	case "plain":
		_, err = fmt.Fprintf(deps.stdout, "%s → %s: %s\n", target, result.ID, status)
	default:
		_, err = fmt.Fprintf(deps.stdout, "Plan prefix: %s\nResolved to: %s\nStatus: %s\n", target, result.ID, status)
	}
	return err
}

// runStatusPath shows status for a file path target.
//
// Behaviour (per AC-008 through AC-012):
//   - Nonexistent file → exit 1 with "file not found"
//   - Unregistered file (exists on disk, no document record) → exit 0 with
//     "not registered" message + suggested kbz doc register command
//   - Registered file with owner → doc view + entity view
//   - Registered file without owner → doc view only
//   - Leading "./" prefix is normalised away before lookup
func runStatusPath(target, format string, deps dependencies) error {
	// Normalise the path: strip "./" prefix for file-existence check.
	normalised := strings.TrimPrefix(target, "./")

	// Check file exists on disk.
	fullPath := filepath.Join(".", normalised)
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		return fmt.Errorf("file not found: %s", target)
	}

	stateRoot := core.StatePath()
	repoRoot := "."
	docSvc := service.NewDocumentService(stateRoot, repoRoot)

	// Strip "./" again for the service call (idempotent, but explicit).
	lookupPath := strings.TrimPrefix(target, "./")
	doc, err := docSvc.LookupByPath(context.Background(), lookupPath)
	if err != nil {
		return fmt.Errorf("lookup document: %w", err)
	}

	// Unregistered file case.
	if doc.ID == "" {
		switch format {
		case "json":
			_, err = fmt.Fprintf(deps.stdout,
				`{"path":%q,"registered":false,"suggestion":"kbz doc register %s","format":"json"}`+"\n",
				lookupPath, lookupPath)
		case "plain":
			_, err = fmt.Fprintf(deps.stdout,
				"%s: not registered (use `kbz doc register %s` to register)\n",
				lookupPath, lookupPath)
		default:
			_, err = fmt.Fprintf(deps.stdout,
				"File: %s\nStatus: not registered\n\nThis file exists on disk but is not registered in the document store.\nRegister it with:\n  kbz doc register %s\n",
				lookupPath, lookupPath)
		}
		return err
	}

	// Registered file: show document info.
	switch format {
	case "json":
		ownerJSON := "null"
		if doc.Owner != "" {
			ownerJSON = fmt.Sprintf("%q", doc.Owner)
		}
		_, err = fmt.Fprintf(deps.stdout,
			`{"path":%q,"registered":true,"doc_id":%q,"type":%q,"title":%q,"status":%q,"owner":%s,"format":"json"}`+"\n",
			doc.Path, doc.ID, doc.Type, doc.Title, doc.Status, ownerJSON)
	case "plain":
		_, err = fmt.Fprintf(deps.stdout, "%s [%s] %s: %s", doc.Path, doc.Type, doc.ID, doc.Status)
		if doc.Owner != "" {
			_, err = fmt.Fprintf(deps.stdout, " (owner: %s)", doc.Owner)
		}
		_, err = fmt.Fprintf(deps.stdout, "\n")
	default:
		_, err = fmt.Fprintf(deps.stdout, "Document: %s\nID: %s\nType: %s\nTitle: %s\nStatus: %s\nPath: %s\n",
			doc.ID, doc.ID, doc.Type, doc.Title, doc.Status, doc.Path)
		if doc.Owner != "" {
			_, err = fmt.Fprintf(deps.stdout, "Owner: %s\n", doc.Owner)
		}
	}

	if err != nil {
		return err
	}

	// If the document has an owner entity, also show the entity status.
	if doc.Owner != "" {
		entitySvc := deps.newEntityService(stateRoot)

		entityType := deriveEntityType(doc.Owner)
		if entityType == "" {
			// Can't determine entity type — skip entity view.
			return nil
		}

		entity, err := entitySvc.Get(entityType, doc.Owner, "")
		if err != nil {
			// Entity lookup failed — skip entity view but don't error.
			return nil
		}

		entityStatus, _ := entity.State["status"].(string)
		switch format {
		case "json":
			// Entity already inline in JSON output above via owner field.
		case "plain":
			_, err = fmt.Fprintf(deps.stdout, "  %s: %s\n", doc.Owner, entityStatus)
		default:
			_, err = fmt.Fprintf(deps.stdout, "\nOwner entity:\n  ID: %s\n  Type: %s\n  Status: %s\n",
				entity.ID, entity.Type, entityStatus)
		}
		if err != nil {
			return err
		}
	}

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
