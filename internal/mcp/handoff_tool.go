// Package mcp handoff_tool.go — sub-agent prompt generation for Kanbanzai 3.0.
//
// handoff(task_id) generates a complete Markdown prompt from an active (or ready,
// needs-rework) task. The prompt is designed for direct use in
// spawn_agent(message=...). The tool is read-only: it does not modify task status.
//
// Context assembly uses the 3.0 pipeline unconditionally. The pipeline's own
// validation steps (stepValidateLifecycle, stepLookupBinding) handle errors
// for missing parent features and missing stage bindings.
//
// Accepted statuses: active, ready, needs-rework.
// Terminal statuses (done, not-planned, duplicate) return an error.
// Other non-accepted statuses (e.g. queued) also return an error.
package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	kbzctx "github.com/sambeau/kanbanzai/internal/context"
	"github.com/sambeau/kanbanzai/internal/git"
	"github.com/sambeau/kanbanzai/internal/id"
	"github.com/sambeau/kanbanzai/internal/model"
	"github.com/sambeau/kanbanzai/internal/service"
	"github.com/sambeau/kanbanzai/internal/validate"
)

// commitStateFunc is the function called by the handoff handler to commit any
// pending .kbz/state/ changes before dispatching a sub-agent. It is a
// package-level variable so tests can inject a stub without changing public
// APIs. The production value delegates to git.CommitStateIfDirty.
var commitStateFunc = func(repoRoot string) (bool, error) {
	return git.CommitStateIfDirty(repoRoot)
}

// HandoffTools returns the `handoff` MCP tool registered in the core group.
//
// The pipeline parameter provides the 3.0 context assembly pipeline.
func HandoffTools(
	entitySvc *service.EntityService,
	pipeline *kbzctx.Pipeline,
) []server.ServerTool {
	return []server.ServerTool{handoffTool(entitySvc, pipeline)}
}

func handoffTool(
	entitySvc *service.EntityService,
	pipeline *kbzctx.Pipeline,
) server.ServerTool {
	tool := mcp.NewTool("handoff",
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
		mcp.WithOpenWorldHintAnnotation(false),
		mcp.WithTitleAnnotation("Sub-Agent Prompt Generator"),
		mcp.WithDescription(
			"Use when delegating a task to a sub-agent — generates a complete, ready-to-use prompt "+
				"by assembling spec sections, knowledge constraints, file paths, and role conventions "+
				"from the task and its parent feature. The output goes directly into spawn_agent's "+
				"message parameter. Call AFTER next(id) claims the task, BEFORE spawn_agent dispatches "+
				"the sub-agent. Read-only: does not modify task status or claim the task. "+
				"For structured JSON context instead of a rendered Markdown prompt, use next(id) which "+
				"returns machine-readable data. Do NOT use to claim tasks — use next for that. "+
				"Accepts tasks in active, ready, or needs-rework status. "+
				"Use INSTEAD OF calling spawn_agent directly — this is the only safe dispatch path; "+
				"direct spawn_agent bypasses context assembly and stage gate enforcement (INV-001).",
		),
		mcp.WithString("task_id",
			mcp.Required(),
			mcp.Description("Task ID (should be in active status; also accepts ready or needs-rework)"),
		),
		mcp.WithString("role",
			mcp.Description("Role profile ID for context shaping (e.g. backend, frontend)"),
		),
		mcp.WithString("instructions",
			mcp.Description("Additional orchestrator instructions to include in the prompt"),
		),
	)

	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		taskID, err := req.RequireString("task_id")
		if err != nil {
			return mcp.NewToolResultError("Cannot generate handoff prompt: task_id is required.\n\nTo resolve:\n  Provide a task_id parameter (e.g. TASK-xxx) for the task to hand off."), nil
		}
		role := strings.TrimSpace(req.GetString("role", ""))
		instructions := strings.TrimSpace(req.GetString("instructions", ""))

		// Load the task.
		task, err := entitySvc.Get("task", taskID, "")
		if err != nil {
			return mcp.NewToolResultText(handoffErrorJSON("not_found",
				fmt.Sprintf("Cannot generate handoff for task %s: task not found.\n\nTo resolve:\n  Verify the task ID exists with entity(action: \"get\", id: %q) or list tasks with entity(action: \"list\", type: \"task\").", taskID, taskID))), nil
		}

		// Validate status.
		status, _ := task.State["status"].(string)
		switch status {
		case "active", "ready", "needs-rework":
			// Accepted — proceed.
		default:
			if validate.IsTerminalState(model.EntityKindTask, status) {
				return mcp.NewToolResultText(handoffErrorJSON("terminal_status", fmt.Sprintf(
					"Cannot generate handoff for task %s: status is %q (terminal).\n\nTo resolve:\n  Handoff is only valid for active, ready, or needs-rework tasks. Create a new task if further work is needed.",
					task.ID, status))), nil
			}
			return mcp.NewToolResultText(handoffErrorJSON("invalid_status", fmt.Sprintf(
				"Cannot generate handoff for task %s: status is %q.\n\nTo resolve:\n  Transition the task to ready or active first, or claim it with next(id: %q).",
				task.ID, status, task.ID))), nil
		}

		// Pre-dispatch state commit: persist any uncommitted .kbz/state/ changes
		// before the sub-agent is dispatched. This protects workflow state from
		// being destroyed by sub-agent git operations (stash, checkout, reset).
		// The commit is best-effort — failure logs a warning but does not block
		// the handoff (REQ-06, REQ-07 of sub-agent-state-isolation spec).
		// commitStateFunc is a package-level variable (see top of file) so
		// tests can inject a stub to verify this path without a real git repo.
		if committed, commitErr := commitStateFunc("."); commitErr != nil {
			log.Printf("[handoff] WARNING: pre-dispatch state commit failed: %v", commitErr)
		} else if committed {
			log.Printf("[handoff] pre-dispatch state commit created for task %s", taskID)
		}

		// Inject re-review guidance when the parent feature is in a focused
		// re-review cycle (review_cycle >= 2 while in reviewing status — FR-008).
		parentFeature, _ := task.State["parent_feature"].(string)
		if parentFeature != "" {
			if feat, featErr := entitySvc.Get("feature", parentFeature, ""); featErr == nil {
				fstatus, _ := feat.State["status"].(string)
				frc, _ := feat.State["review_cycle"].(int)
				if fstatus == "reviewing" && frc >= 2 {
					guidance := fmt.Sprintf(
						"## Re-Review Guidance (Cycle %d of %d)\n\n"+
							"This is a **focused re-review** (cycle %d). Narrow your scope:\n"+
							"- Review only rework tasks created since the previous review and the changes they made.\n"+
							"- Check that each finding from the previous review has been addressed.\n"+
							"- Read rework task descriptions to understand what was supposed to change.\n"+
							"- Do NOT re-review unchanged implementation from earlier cycles.",
						frc, service.DefaultMaxReviewCycles, frc,
					)
					if instructions != "" {
						instructions = guidance + "\n\n" + instructions
					} else {
						instructions = guidance
					}
				}
			}
		}

		// ── 3.0 pipeline assembly ──────────────────────────────────────────
		// The pipeline's stepValidateLifecycle and stepLookupBinding handle
		// errors for missing parent features and missing stage bindings.

		input := kbzctx.PipelineInput{
			TaskID:       task.ID,
			TaskState:    task.State,
			FeatureState: nil,
			Role:         role,
			Instructions: instructions,
		}

		if parentFeature != "" {
			if feat, featErr := entitySvc.Get("feature", parentFeature, ""); featErr == nil {
				input.FeatureState = feat.State
			}
		}

		result, runErr := pipeline.Run(input)
		if runErr != nil {
			return mcp.NewToolResultText(handoffErrorJSON("pipeline_error",
				fmt.Sprintf("Cannot assemble handoff for task %s: pipeline error: %v.\n\nTo resolve:\n  Check that the role profile exists and skill files are present. Review the feature's stage binding configuration.",
					task.ID, runErr))), nil
		}

		// Render the prompt and build the response.
		prompt := kbzctx.RenderPrompt(result)

		displayID := id.FormatFullDisplay(task.ID)
		slug, _ := task.State["slug"].(string)
		label, _ := task.State["label"].(string)

		sectionLabels := make([]string, len(result.Sections))
		for i, s := range result.Sections {
			sectionLabels[i] = s.Label
		}

		resp := map[string]any{
			"task_id":    task.ID,
			"display_id": displayID,
			"entity_ref": id.FormatEntityRef(displayID, slug, label),
			"prompt":     prompt,
			"context_metadata": map[string]any{
				"assembly_path":     "pipeline-3.0",
				"sections":          sectionLabels,
				"total_tokens":      result.TotalTokens,
				"token_warning":     result.TokenWarning,
				"metadata_warnings": result.MetadataWarnings,
			},
		}

		data, err := json.MarshalIndent(resp, "", "  ")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Cannot serialise handoff response for task %s: %s.\n\nTo resolve:\n  This is an internal error — report it as a bug with the task ID.", task.ID, err)), nil
		}
		return mcp.NewToolResultText(string(data)), nil
	}

	return server.ServerTool{Tool: tool, Handler: handler}
}

// handoffErrorJSON produces the standard error JSON string for handoff responses.
func handoffErrorJSON(code, message string) string {
	b, _ := json.Marshal(map[string]any{
		"error": map[string]string{
			"code":    code,
			"message": message,
		},
	})
	return string(b)
}
