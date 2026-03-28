// Package mcp handoff_tool.go — sub-agent prompt generation for Kanbanzai 2.0 (Track G).
//
// handoff(task_id) generates a complete Markdown prompt from an active (or ready,
// needs-rework) task. The prompt is designed for direct use in
// spawn_agent(message=...). The tool is read-only: it does not modify task status.
//
// Context assembly uses the shared pipeline in assembly.go (spec §11.5,
// implementation plan §3.4). The difference from next is output format:
// handoff renders a Markdown prompt; next returns structured data.
//
// Accepted statuses: active, ready, needs-rework.
// Terminal statuses (done, not-planned, duplicate) return an error.
// Other non-accepted statuses (e.g. queued) also return an error.
package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	kbzctx "kanbanzai/internal/context"
	"kanbanzai/internal/id"
	"kanbanzai/internal/service"
)

// HandoffTools returns the `handoff` MCP tool registered in the core group.
func HandoffTools(
	entitySvc *service.EntityService,
	profileStore *kbzctx.ProfileStore,
	knowledgeSvc *service.KnowledgeService,
	intelligenceSvc *service.IntelligenceService,
	docRecordSvc *service.DocumentService,
) []server.ServerTool {
	return []server.ServerTool{handoffTool(entitySvc, profileStore, knowledgeSvc, intelligenceSvc, docRecordSvc)}
}

func handoffTool(
	entitySvc *service.EntityService,
	profileStore *kbzctx.ProfileStore,
	knowledgeSvc *service.KnowledgeService,
	intelligenceSvc *service.IntelligenceService,
	docRecordSvc *service.DocumentService,
) server.ServerTool {
	tool := mcp.NewTool("handoff",
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
		mcp.WithOpenWorldHintAnnotation(false),
		mcp.WithTitleAnnotation("Sub-Agent Prompt Generator"),
		mcp.WithDescription(
			"Generate a complete sub-agent prompt from a task. "+
				"The output is designed to go directly into spawn_agent's message parameter. "+
				"Assembles spec sections, knowledge constraints, file paths, and role conventions "+
				"into a structured Markdown prompt. "+
				"Read-only: does not modify task status. "+
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
			return mcp.NewToolResultError(err.Error()), nil
		}
		role := strings.TrimSpace(req.GetString("role", ""))
		instructions := strings.TrimSpace(req.GetString("instructions", ""))

		// Load the task.
		task, err := entitySvc.Get("task", taskID, "")
		if err != nil {
			return mcp.NewToolResultText(handoffErrorJSON("not_found",
				fmt.Sprintf("Task %s not found", taskID))), nil
		}

		// Validate status.
		status, _ := task.State["status"].(string)
		switch status {
		case "active", "ready", "needs-rework":
			// Accepted — proceed.
		default:
			if isTerminalStatus(status) {
				return mcp.NewToolResultText(handoffErrorJSON("terminal_status", fmt.Sprintf(
					"Task %s is in status %q (terminal). Handoff is only meaningful for active or ready tasks.",
					task.ID, status))), nil
			}
			return mcp.NewToolResultText(handoffErrorJSON("invalid_status", fmt.Sprintf(
				"Task %s is in status %q. Handoff requires active, ready, or needs-rework.",
				task.ID, status))), nil
		}

		// Assemble context using the shared pipeline (assembly.go).
		parentFeature, _ := task.State["parent_feature"].(string)
		actx := assembleContext(asmInput{
			taskState:       task.State,
			parentFeature:   parentFeature,
			role:            role,
			profileStore:    profileStore,
			knowledgeSvc:    knowledgeSvc,
			intelligenceSvc: intelligenceSvc,
			docRecordSvc:    docRecordSvc,
			entitySvc:       entitySvc,
		})

		// Render the Markdown prompt.
		prompt := renderHandoffPrompt(task.State, actx, instructions)

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
				"spec_sections_included":     len(actx.specSections),
				"knowledge_entries_included": len(actx.knowledge),
				"byte_usage":                 actx.byteUsage,
				"byte_budget":                actx.byteBudget,
				"trimmed":                    trimmedOut,
			},
		}

		data, err := json.MarshalIndent(resp, "", "  ")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("marshal result: %s", err)), nil
		}
		return mcp.NewToolResultText(string(data)), nil
	}

	return server.ServerTool{Tool: tool, Handler: handler}
}

// ─── Prompt rendering ─────────────────────────────────────────────────────────

// renderHandoffPrompt builds the Markdown prompt string from assembled context.
func renderHandoffPrompt(taskState map[string]any, actx assembledContext, instructions string) string {
	var sb strings.Builder
	taskID, _ := taskState["id"].(string)
	summary, _ := taskState["summary"].(string)

	// Title and summary.
	fmt.Fprintf(&sb, "## Task: %s\n\n", summary)
	fmt.Fprintf(&sb, "### Summary\n\n%s\n\n", summary)

	// Specification sections from doc intelligence.
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

	// Acceptance criteria extracted from spec sections (spec §13.5, G.2).
	// Populated by asmExtractCriteria from sections whose title contains
	// "acceptance"/"criteria"/"requirement", or whose items contain MUST/SHALL.
	if len(actx.acceptanceCriteria) > 0 {
		sb.WriteString("### Acceptance Criteria\n\n")
		for _, ac := range actx.acceptanceCriteria {
			fmt.Fprintf(&sb, "- %s\n", ac)
		}
		sb.WriteString("\n")
	}

	// Knowledge constraints.
	if len(actx.knowledge) > 0 {
		sb.WriteString("### Known Constraints (from knowledge base)\n\n")
		for _, ke := range actx.knowledge {
			fmt.Fprintf(&sb, "- %s\n", ke.content)
		}
		sb.WriteString("\n")
	}

	// File paths.
	filePaths := make([]string, 0, len(actx.filesContext))
	for _, f := range actx.filesContext {
		filePaths = append(filePaths, f.path)
	}
	if len(filePaths) > 0 {
		sb.WriteString("### Files\n\n")
		for _, f := range filePaths {
			fmt.Fprintf(&sb, "- %s\n", f)
		}
		sb.WriteString("\n")
	}

	// Conventions (role profile + always-present commit format).
	sb.WriteString("### Conventions\n\n")
	for _, c := range actx.constraints {
		fmt.Fprintf(&sb, "- %s\n", c)
	}
	if taskID != "" {
		fmt.Fprintf(&sb, "- Commit format: feat(%s): <description>\n", taskID)
	}
	sb.WriteString("\n")

	// Active workflow experiments (Phase 3 context nudge, spec §8.4).
	if len(actx.experimentNudge) > 0 {
		sb.WriteString("### Active Workflow Experiments\n\n")
		sb.WriteString("The following workflow experiments are active. If you encounter friction or success related to any of these, reference the decision ID in your retrospective signal's `related_decision` field.\n\n")
		for _, exp := range actx.experimentNudge {
			fmt.Fprintf(&sb, "- **%s**: %s\n", exp.decisionID, exp.summary)
		}
		sb.WriteString("\n")
	}

	// Additional orchestrator instructions.
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
