// Package mcp handoff_tool.go — sub-agent prompt generation for Kanbanzai 2.0 (Track G).
//
// handoff(task_id) generates a complete Markdown prompt from an active (or ready,
// needs-rework) task. The prompt is designed for direct use in
// spawn_agent(message=...). The tool is read-only: it does not modify task status.
//
// Context assembly uses two paths:
//  1. New 10-step pipeline (3.0): when a stage binding exists for the task's parent
//     feature lifecycle stage, the pipeline assembles attention-curve-ordered context
//     with roles, skills, vocabulary, anti-patterns, and token budget management.
//  2. Legacy assembly (2.0): when no stage binding exists or the pipeline is not
//     configured, falls back to the shared assembleContext() in assembly.go.
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
	"github.com/sambeau/kanbanzai/internal/service"
	"github.com/sambeau/kanbanzai/internal/stage"
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
// The pipeline parameter is optional. When non-nil, the handler attempts the
// 10-step assembly pipeline for tasks whose parent feature has a stage binding.
// When nil (or when no binding exists for the stage), the handler falls back to
// the legacy assembleContext() path, preserving full backward compatibility
// (NFR-003).
func HandoffTools(
	entitySvc *service.EntityService,
	profileStore *kbzctx.ProfileStore,
	knowledgeSvc *service.KnowledgeService,
	intelligenceSvc *service.IntelligenceService,
	docRecordSvc *service.DocumentService,
	pipeline *kbzctx.Pipeline,
) []server.ServerTool {
	return []server.ServerTool{handoffTool(entitySvc, profileStore, knowledgeSvc, intelligenceSvc, docRecordSvc, pipeline)}
}

func handoffTool(
	entitySvc *service.EntityService,
	profileStore *kbzctx.ProfileStore,
	knowledgeSvc *service.KnowledgeService,
	intelligenceSvc *service.IntelligenceService,
	docRecordSvc *service.DocumentService,
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
				"Accepts tasks in active, ready, or needs-rework status.",
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
			return mcp.NewToolResultError(fmt.Sprintf(
				"Cannot generate handoff prompt: task_id is required.\n\nTo resolve:\n  Provide a task_id parameter (e.g. TASK-xxx) for the task to hand off.")), nil
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
			if isTerminalStatus(status) {
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

		// ── Attempt 3.0 pipeline assembly ──────────────────────────────────
		//
		// The pipeline is used when:
		//   1. A *Pipeline is configured (non-nil).
		//   2. The task has a parent feature.
		//   3. The parent feature's lifecycle stage has a binding in the registry.
		//
		// If any of these conditions fail, we fall back to the legacy 2.0 path.
		// Pipeline errors (missing role, missing skill, token budget exceeded)
		// are returned as tool errors — they indicate misconfiguration that the
		// user should fix, not something to silently degrade through.

		parentFeature, _ := task.State["parent_feature"].(string)

		// Inject re-review guidance when the parent feature is in a focused
		// re-review cycle (review_cycle >= 2 while in reviewing status — FR-008).
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

		if pipelineResult, used := tryPipeline(pipeline, entitySvc, task.State, parentFeature, role, instructions); used {
			if pipelineResult.err != nil {
				return mcp.NewToolResultText(handoffErrorJSON("pipeline_error",
					fmt.Sprintf("Cannot assemble handoff for task %s: pipeline error: %v.\n\nTo resolve:\n  Check that the role profile exists and skill files are present. Review the feature's stage binding configuration.",
						task.ID, pipelineResult.err))), nil
			}
			return buildPipelineResponse(task, pipelineResult.result)
		}

		// ── Legacy 2.0 assembly fallback ───────────────────────────────────

		// Stage-aware lifecycle validation (FR-001).
		featureStage, valErr := ValidateFeatureStage(parentFeature, entitySvc)
		if valErr != nil {
			return mcp.NewToolResultText(handoffErrorJSON("stage_validation",
				fmt.Sprintf("Cannot assemble context for task %s: %v", task.ID, valErr))), nil
		}

		actx := assembleContext(asmInput{
			taskState:       task.State,
			parentFeature:   parentFeature,
			featureStage:    featureStage,
			role:            role,
			profileStore:    profileStore,
			knowledgeSvc:    knowledgeSvc,
			intelligenceSvc: intelligenceSvc,
			docRecordSvc:    docRecordSvc,
			entitySvc:       entitySvc,
		})

		prompt := renderHandoffPrompt(task.State, actx, instructions)
		return buildLegacyResponse(task, actx, prompt)
	}

	return server.ServerTool{Tool: tool, Handler: handler}
}

// ─── Pipeline integration ─────────────────────────────────────────────────────

// pipelineAttempt holds the result of trying the 3.0 pipeline.
type pipelineAttempt struct {
	result *kbzctx.PipelineResult
	err    error
}

// tryPipeline attempts to use the 3.0 pipeline for context assembly.
// Returns (result, true) if the pipeline was used (success or error).
// Returns (nil, false) if the pipeline should not be used and the caller
// should fall back to legacy assembly.
func tryPipeline(
	pipeline *kbzctx.Pipeline,
	entitySvc *service.EntityService,
	taskState map[string]any,
	parentFeature string,
	role string,
	instructions string,
) (pipelineAttempt, bool) {
	if pipeline == nil {
		return pipelineAttempt{}, false
	}
	if parentFeature == "" {
		return pipelineAttempt{}, false
	}

	// Load the parent feature to get its lifecycle status.
	feat, err := entitySvc.Get("feature", parentFeature, "")
	if err != nil {
		// Can't resolve feature — fall back to legacy.
		log.Printf("[handoff] pipeline: cannot load feature %s, falling back to legacy: %v", parentFeature, err)
		return pipelineAttempt{}, false
	}

	featureStatus, _ := feat.State["status"].(string)

	// Check whether a binding exists for this stage before committing to the
	// pipeline. If no binding exists, we fall back to legacy gracefully.
	if pipeline.Bindings != nil {
		if _, bindErr := pipeline.Bindings.Lookup(featureStatus); bindErr != nil {
			// No binding for this stage — fall back to legacy.
			log.Printf("[handoff] pipeline: no binding for stage %q, falling back to legacy", featureStatus)
			return pipelineAttempt{}, false
		}
	} else {
		// No binding resolver at all — fall back to legacy.
		return pipelineAttempt{}, false
	}

	// Pipeline is applicable — run it. From this point, errors are returned
	// to the caller rather than triggering a fallback, because they indicate
	// real problems (missing role file, missing skill, budget exceeded).
	input := kbzctx.PipelineInput{
		TaskID:       taskState["id"].(string),
		TaskState:    taskState,
		FeatureState: feat.State,
		Role:         role,
		Instructions: instructions,
	}

	result, runErr := pipeline.Run(input)
	return pipelineAttempt{result: result, err: runErr}, true
}

// buildPipelineResponse constructs the MCP tool response from a pipeline result.
func buildPipelineResponse(task service.GetResult, result *kbzctx.PipelineResult) (*mcp.CallToolResult, error) {
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

// buildLegacyResponse constructs the MCP tool response from legacy assembly.
func buildLegacyResponse(task service.GetResult, actx assembledContext, prompt string) (*mcp.CallToolResult, error) {
	trimmedOut := make([]map[string]any, len(actx.trimmed))
	for i, te := range actx.trimmed {
		trimmedOut[i] = map[string]any{
			"type":       te.entryType,
			"topic":      te.topic,
			"size_bytes": te.sizeBytes,
		}
	}

	displayID := id.FormatFullDisplay(task.ID)
	slug, _ := task.State["slug"].(string)
	label, _ := task.State["label"].(string)
	resp := map[string]any{
		"task_id":    task.ID,
		"display_id": displayID,
		"entity_ref": id.FormatEntityRef(displayID, slug, label),
		"prompt":     prompt,
		"context_metadata": map[string]any{
			"assembly_path":              "legacy-2.0",
			"stage_aware":                actx.stageAware,
			"feature_stage":              actx.featureStage,
			"spec_sections_included":     len(actx.specSections),
			"knowledge_entries_included": len(actx.knowledge),
			"byte_usage":                 actx.byteUsage,
			"byte_budget":                actx.byteBudget,
			"trimmed":                    trimmedOut,
		},
	}

	data, err := json.MarshalIndent(resp, "", "  ")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Cannot serialise handoff response for task %s: %s.\n\nTo resolve:\n  This is an internal error — report it as a bug with the task ID.", task.ID, err)), nil
	}
	return mcp.NewToolResultText(string(data)), nil
}

// ─── Prompt rendering (legacy 2.0 path) ───────────────────────────────────────

// renderHandoffPrompt builds the Markdown prompt string from assembled context.
func renderHandoffPrompt(taskState map[string]any, actx assembledContext, instructions string) string {
	var sb strings.Builder
	taskID, _ := taskState["id"].(string)
	summary, _ := taskState["summary"].(string)

	// 1. Conventions (role profile + commit format) — high attention zone.
	sb.WriteString("### Conventions\n\n")
	for _, c := range actx.constraints {
		fmt.Fprintf(&sb, "- %s\n", c)
	}
	if taskID != "" {
		fmt.Fprintf(&sb, "- Commit format: feat(%s): <description>\n", taskID)
	}
	sb.WriteString("\n")

	// 2. Stage-aware sections (FR-006, FR-007, FR-008, FR-009).
	if actx.orchestrationText != "" {
		sb.WriteString(actx.orchestrationText)
		sb.WriteString("\n\n")
	}
	if actx.effortBudgetText != "" {
		sb.WriteString(actx.effortBudgetText)
		sb.WriteString("\n\n")
	}
	if actx.toolSubsetText != "" {
		sb.WriteString(actx.toolSubsetText)
		sb.WriteString("\n\n")
	}
	if actx.outputConventionText != "" {
		sb.WriteString(actx.outputConventionText)
		sb.WriteString("\n\n")
	}
	if actx.reviewRubricText != "" {
		sb.WriteString(actx.reviewRubricText)
		sb.WriteString("\n\n")
	}
	if actx.testExpectText != "" {
		sb.WriteString(actx.testExpectText)
		sb.WriteString("\n\n")
	}
	if actx.implGuidanceText != "" {
		sb.WriteString(actx.implGuidanceText)
		sb.WriteString("\n\n")
	}
	if actx.planGuidanceText != "" {
		sb.WriteString(actx.planGuidanceText)
		sb.WriteString("\n\n")
	}

	// 3. Title and summary.
	fmt.Fprintf(&sb, "## Task: %s\n\n", summary)
	fmt.Fprintf(&sb, "### Summary\n\n%s\n\n", summary)

	// 4. Specification sections from doc intelligence.
	for _, s := range actx.specSections {
		ref := s.document
		if s.section != "" {
			ref = fmt.Sprintf("%s §%s", s.document, s.section)
		}
		fmt.Fprintf(&sb, "### Specification (from %s)\n\n%s\n\n", ref, strings.TrimSpace(s.content))
	}

	// Fallback: when no spec sections were extracted, point the agent to
	// the raw document path (graceful degradation per §24.3).
	if len(actx.specSections) == 0 && actx.specFallbackPath != "" {
		fmt.Fprintf(&sb, "### Specification\n\nRefer to: %s\n\n", actx.specFallbackPath)
	}

	// 5. Acceptance criteria.
	if len(actx.acceptanceCriteria) > 0 {
		sb.WriteString("### Acceptance Criteria\n\n")
		for _, ac := range actx.acceptanceCriteria {
			fmt.Fprintf(&sb, "- %s\n", ac)
		}
		sb.WriteString("\n")
	}

	// 6. Knowledge constraints.
	if len(actx.knowledge) > 0 {
		sb.WriteString("### Known Constraints (from knowledge base)\n\n")
		for _, ke := range actx.knowledge {
			fmt.Fprintf(&sb, "- %s\n", ke.content)
		}
		sb.WriteString("\n")
	}

	// 7. File paths — excluded for designing/specifying stages per FR-005.
	if len(actx.filesContext) > 0 {
		includeFiles := true
		if actx.stageAware {
			if cfg, ok := stage.ForStage(actx.featureStage); ok && !cfg.IncludeFilePaths {
				includeFiles = false
			}
		}
		if includeFiles {
			sb.WriteString("### Files\n\n")
			for _, f := range actx.filesContext {
				fmt.Fprintf(&sb, "- %s\n", f.path)
			}
			sb.WriteString("\n")
		}
	}

	// 8. Active workflow experiments.
	if len(actx.experimentNudge) > 0 {
		sb.WriteString("### Active Workflow Experiments\n\n")
		sb.WriteString("The following workflow experiments are active. If you encounter friction or success related to any of these, reference the decision ID in your retrospective signal's `related_decision` field.\n\n")
		for _, exp := range actx.experimentNudge {
			fmt.Fprintf(&sb, "- **%s**: %s\n", exp.decisionID, exp.summary)
		}
		sb.WriteString("\n")
	}

	// 9. Additional orchestrator instructions.
	if instructions != "" {
		fmt.Fprintf(&sb, "### Additional Instructions\n\n%s\n\n", instructions)
	}

	return sb.String()
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

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
