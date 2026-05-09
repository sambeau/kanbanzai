// Package mcp develop_tool.go — top-level tool for development dispatch
// operations introduced by the Composite Tools feature (B43).
//
//	develop(action: "dispatch", feature_id: "FEAT-...", role: "...")
//
// The dispatch action identifies the ready task frontier for a feature in
// developing status, runs conflict analysis on ready tasks, transitions
// conflict-safe tasks from ready to active, and returns handoff prompts
// for each dispatched task. It does NOT spawn sub-agents — the calling agent
// is responsible for that.
package mcp

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/sambeau/kanbanzai/internal/service"
)

// DevelopTool returns the `develop` MCP tool registered unconditionally.
// The tool is new in Kanbanzai 2.1 (B43 — Composite Tools).
func DevelopTool(entitySvc *service.EntityService, conflictSvc *service.ConflictService) []server.ServerTool {
	tool := mcp.NewTool("develop",
		mcp.WithTitleAnnotation("Development Dispatch"),
		mcp.WithDescription(
			"Orchestrator dispatch tool — identifies the ready task frontier for a feature, "+
				"runs conflict analysis, transitions conflict-safe tasks to active, and returns "+
				"handoff prompts for each dispatched task. Does NOT spawn sub-agents. "+
				"Actions: dispatch.",
		),
		mcp.WithString("action", mcp.Required(), mcp.Description("Action: dispatch")),
		mcp.WithString("feature_id", mcp.Required(), mcp.Description("Feature ID to dispatch for")),
		mcp.WithString("role", mcp.Description("Role profile ID for context assembly")),
		mcp.WithString("instructions", mcp.Description("Additional orchestrator instructions")),
	)

	handler := WithSideEffects(func(ctx context.Context, req mcp.CallToolRequest) (any, error) {
		return DispatchAction(ctx, req, map[string]ActionHandler{
			"dispatch": developDispatchAction(entitySvc, conflictSvc),
		})
	})

	return []server.ServerTool{{Tool: tool, Handler: handler}}
}

// developDispatchAction implements develop(action: "dispatch", ...).
func developDispatchAction(entitySvc *service.EntityService, conflictSvc *service.ConflictService) ActionHandler {
	return func(ctx context.Context, req mcp.CallToolRequest) (any, error) {
		SignalMutation(ctx)
		args, _ := req.Params.Arguments.(map[string]any)

		featureID := docArgStr(args, "feature_id")
		if featureID == "" {
			return nil, fmt.Errorf("Cannot dispatch: feature_id is missing.\n\nTo resolve:\n  Provide feature_id: develop(action: \"dispatch\", feature_id: \"FEAT-...\")")
		}

		// Verify feature exists and is in developing status.
		feat, err := entitySvc.Get(ctx, "feature", featureID, "")
		if err != nil {
			return nil, fmt.Errorf("Cannot dispatch: feature %s not found: %w", featureID, err)
		}
		featStatus, _ := feat.State["status"].(string)
		if featStatus != "developing" {
			return map[string]any{
				"error":       fmt.Sprintf("Feature %s is in %q, not developing", featureID, featStatus),
				"empty_queue": true,
			}, nil
		}

		// List tasks for this feature by filtering all tasks.
		allTasks, listErr := entitySvc.List("task")
		if listErr != nil {
			return nil, fmt.Errorf("Cannot dispatch: %w", listErr)
		}

		// Build a set of terminal task IDs for dependency checking.
		terminalIDs := map[string]bool{}
		for _, t := range allTasks {
			s, _ := t.State["status"].(string)
			if s == "done" || s == "not-planned" || s == "duplicate" {
				terminalIDs[t.ID] = true
			}
		}

		tasks := make([]service.ListResult, 0)
		for _, t := range allTasks {
			pf, _ := t.State["parent_feature"].(string)
			if pf == featureID {
				tasks = append(tasks, t)
			}
		}

		// Separate tasks by status.
		type taskInfo struct {
			ID        string
			Slug      string
			Summary   string
			DependsOn []string
		}
		var readyTasks []taskInfo
		var activeTasks []taskInfo
		var blockedTasks []map[string]any

		for _, t := range tasks {
			status, _ := t.State["status"].(string)
			summary, _ := t.State["summary"].(string)
			var deps []string
			if raw, ok := t.State["depends_on"]; ok {
				switch v := raw.(type) {
				case []any:
					for _, d := range v {
						if ds, ok := d.(string); ok {
							deps = append(deps, ds)
						}
					}
				case []string:
					deps = v
				}
			}
			info := taskInfo{ID: t.ID, Slug: t.Slug, Summary: summary, DependsOn: deps}

			switch status {
			case "ready":
				readyTasks = append(readyTasks, info)
			case "active":
				activeTasks = append(activeTasks, info)
			case "queued":
				blockedTasks = append(blockedTasks, map[string]any{
					"task_id": t.ID,
					"slug":    t.Slug,
					"summary": summary,
					"reason":  "queued — dependencies not yet satisfied",
				})
			case "done", "not-planned", "duplicate":
				// terminal — exclude from blocked
			default:
				blockedTasks = append(blockedTasks, map[string]any{
					"task_id": t.ID,
					"slug":    t.Slug,
					"summary": summary,
					"reason":  fmt.Sprintf("status is %q, not ready", status),
				})
			}
		}

		// Filter ready tasks to those with all depends_on satisfied.
		var dispatchable []taskInfo
		var unsatisfiedDeps []map[string]any
		for _, t := range readyTasks {
			allSatisfied := true
			var missing []string
			for _, dep := range t.DependsOn {
				if !terminalIDs[dep] {
					allSatisfied = false
					missing = append(missing, dep)
				}
			}
			if allSatisfied {
				dispatchable = append(dispatchable, t)
			} else {
				unsatisfiedDeps = append(unsatisfiedDeps, map[string]any{
					"task_id":      t.ID,
					"slug":         t.Slug,
					"summary":      t.Summary,
					"reason":       fmt.Sprintf("depends_on not satisfied: %v", missing),
					"missing_deps": missing,
				})
			}
		}

		// Add unsatisfied-dependency tasks to blocked.
		blockedTasks = append(blockedTasks, unsatisfiedDeps...)

		if len(dispatchable) == 0 {
			return map[string]any{
				"dispatched":  []any{},
				"conflicting": []any{},
				"blocked":     blockedTasks,
				"empty_queue": true,
			}, nil
		}

		// Run conflict analysis on dispatchable tasks.
		var conflicting []map[string]any
		var safeToDispatch []taskInfo
		if conflictSvc != nil && len(dispatchable) > 1 {
			taskIDs := make([]string, len(dispatchable))
			for i, t := range dispatchable {
				taskIDs[i] = t.ID
			}
			conflictResult, cErr := conflictSvc.Check(service.ConflictCheckInput{
				TaskIDs: taskIDs,
			})
			if cErr == nil {
				conflictTaskIDs := map[string]bool{}
				for _, pair := range conflictResult.Pairs {
					if pair.Recommendation != "safe_to_parallelise" {
						conflictTaskIDs[pair.TaskA] = true
						conflictTaskIDs[pair.TaskB] = true
					}
				}
				for _, t := range dispatchable {
					if conflictTaskIDs[t.ID] {
						conflicting = append(conflicting, map[string]any{
							"task_id": t.ID,
							"slug":    t.Slug,
							"summary": t.Summary,
							"reason":  "conflict detected — serialise or checkpoint",
						})
					} else {
						safeToDispatch = append(safeToDispatch, t)
					}
				}
			} else {
				// Conflict analysis failed — fall back to dispatching all.
				safeToDispatch = dispatchable
			}
		} else {
			safeToDispatch = dispatchable
		}

		// Transition conflict-safe tasks to active and generate handoff prompts.
		dispatched := make([]map[string]any, 0)
		instructions := docArgStr(args, "instructions")
		for _, t := range safeToDispatch {
			_, err := entitySvc.UpdateStatus(service.UpdateStatusInput{
				Type:   "task",
				ID:     t.ID,
				Status: "active",
			})
			if err != nil {
				// Task may have been claimed by another dispatcher — skip.
				continue
			}

			// Generate a handoff prompt for this task.
			handoffPrompt := generateDispatchHandoff(entitySvc, t.ID, instructions)

			dispatched = append(dispatched, map[string]any{
				"task_id":        t.ID,
				"slug":           t.Slug,
				"summary":        t.Summary,
				"handoff_prompt": handoffPrompt,
			})
		}

		return map[string]any{
				"dispatched":  dispatched,
				"conflicting": conflicting,
				"blocked":     blockedTasks,
				"empty_queue": false,
			},
			nil
	}
}

// generateDispatchHandoff produces a handoff prompt for a task by calling the
// handoff assembly pipeline. When the pipeline is unavailable, it returns a
// minimal hint directing the agent to use the handoff tool.
func generateDispatchHandoff(entitySvc *service.EntityService, taskID, instructions string) string {
	task, err := entitySvc.Get(context.Background(), "task", taskID, "")
	if err != nil {
		return fmt.Sprintf("<!-- handoff: %s -->", taskID)
	}

	summary, _ := task.State["summary"].(string)

	// Build a minimal handoff prompt pointing the agent to use the full
	// handoff tool. The handoff pipeline requires extensive context assembly
	// (profiles, skills, docs, knowledge) that the dispatch action does not
	// have direct access to. Instead, we produce a prompt that tells the
	// calling agent to invoke handoff for the full assembled context.
	prompt := fmt.Sprintf("Task: %s — %s\n\n", taskID, summary)
	prompt += fmt.Sprintf("Use handoff(task_id: %q) to get the assembled context and dispatch this task.", taskID)
	if instructions != "" {
		prompt += fmt.Sprintf("\n\nAdditional instructions: %s", instructions)
	}
	return prompt
}
