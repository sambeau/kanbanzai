// Package mcp handoff_tool.go — sub-agent prompt generation for Kanbanzai 2.0 (Track G).
//
// handoff(task_id) generates a complete Markdown prompt from an active (or ready,
// needs-rework) task. The prompt is designed for direct use in
// spawn_agent(message=...). The tool is read-only: it does not modify task status.
//
// Accepted statuses: active, ready, needs-rework.
// Terminal statuses (done, not-planned, duplicate) return an error.
// Other non-accepted statuses (e.g. queued) also return an error.
package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	kbzctx "kanbanzai/internal/context"
	"kanbanzai/internal/service"
)

const handoffDefaultBudget = 30720

// HandoffTools returns the `handoff` MCP tool registered in the core group.
func HandoffTools(
	entitySvc *service.EntityService,
	profileStore *kbzctx.ProfileStore,
	knowledgeSvc *service.KnowledgeService,
	intelligenceSvc *service.IntelligenceService,
) []server.ServerTool {
	return []server.ServerTool{handoffTool(entitySvc, profileStore, knowledgeSvc, intelligenceSvc)}
}

func handoffTool(
	entitySvc *service.EntityService,
	profileStore *kbzctx.ProfileStore,
	knowledgeSvc *service.KnowledgeService,
	intelligenceSvc *service.IntelligenceService,
) server.ServerTool {
	tool := mcp.NewTool("handoff",
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

		// Assemble context and render prompt.
		hctx := assembleHandoffContext(task.State, role, profileStore, knowledgeSvc, intelligenceSvc)
		prompt := renderHandoffPrompt(task.State, hctx, instructions)

		trimmedOut := make([]map[string]any, len(hctx.trimmed))
		for i, te := range hctx.trimmed {
			trimmedOut[i] = map[string]any{
				"type":       te.entryType,
				"topic":      te.topic,
				"size_bytes": te.sizeBytes,
			}
		}

		resp := map[string]any{
			"task_id": task.ID,
			"prompt":  prompt,
			"context_metadata": map[string]any{
				"spec_sections_included":     len(hctx.specSections),
				"knowledge_entries_included": len(hctx.knowledge),
				"byte_usage":                 hctx.byteUsage,
				"byte_budget":                hctx.byteBudget,
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

// ─── Assembly types ───────────────────────────────────────────────────────────

// handoffSpecSection is a single spec or design section included in the prompt.
type handoffSpecSection struct {
	document string
	section  string
	content  string
}

// handoffKnowledgeEntry is a single knowledge entry included in the prompt.
type handoffKnowledgeEntry struct {
	topic      string
	content    string
	confidence float64
	tier       int
}

// handoffTrimmedEntry records a context item that was removed to stay within budget.
type handoffTrimmedEntry struct {
	entryType string
	topic     string
	sizeBytes int
}

// handoffContextData holds the assembled context for renderHandoffPrompt.
type handoffContextData struct {
	specSections []handoffSpecSection
	knowledge    []handoffKnowledgeEntry
	conventions  []string
	filesPlanned []string
	trimmed      []handoffTrimmedEntry
	byteUsage    int
	byteBudget   int
}

// ─── Assembly ─────────────────────────────────────────────────────────────────

// assembleHandoffContext gathers spec sections, knowledge entries, and profile
// conventions for the prompt. All sources are best-effort: service errors
// produce empty sections rather than failures.
func assembleHandoffContext(
	taskState map[string]any,
	role string,
	profileStore *kbzctx.ProfileStore,
	knowledgeSvc *service.KnowledgeService,
	intelligenceSvc *service.IntelligenceService,
) handoffContextData {
	hctx := handoffContextData{byteBudget: handoffDefaultBudget}

	// files_planned from task state.
	hctx.filesPlanned = handoffStringSlice(taskState, "files_planned")

	// Spec/design sections from document intelligence.
	if intelligenceSvc != nil {
		parentFeature, _ := taskState["parent_feature"].(string)
		if parentFeature != "" {
			if matches, err := intelligenceSvc.TraceEntity(parentFeature); err == nil {
				for _, match := range matches {
					_, content, err := intelligenceSvc.GetSection(match.DocumentID, match.SectionPath)
					if err != nil || len(content) == 0 {
						continue
					}
					title := match.SectionTitle
					if title == "" {
						title = match.SectionPath
					}
					hctx.specSections = append(hctx.specSections, handoffSpecSection{
						document: match.DocumentID,
						section:  title,
						content:  string(content),
					})
				}
			}
		}
	}

	// Role profile conventions.
	if profileStore != nil && role != "" {
		if profile, err := kbzctx.ResolveProfile(profileStore, role); err == nil {
			hctx.conventions = profile.Conventions
		}
	}

	// Knowledge entries (Tier 2 + Tier 3), scoped to role or "project".
	if knowledgeSvc != nil {
		hctx.knowledge = loadHandoffKnowledge(knowledgeSvc, role)
	}

	// Compute byte usage and trim if over budget.
	hctx.byteUsage = handoffByteCount(hctx, taskState)
	if hctx.byteUsage > handoffDefaultBudget {
		hctx = trimHandoffContext(hctx, taskState)
	}

	return hctx
}

// loadHandoffKnowledge loads knowledge entries for the handoff prompt.
// Returns entries scoped to the given role or "project", covering T2+T3,
// sorted by confidence descending (highest confidence first).
func loadHandoffKnowledge(svc *service.KnowledgeService, role string) []handoffKnowledgeEntry {
	var entries []handoffKnowledgeEntry

	tierConfig := []struct {
		tier    int
		minConf float64
	}{
		{2, 0.3},
		{3, 0.5},
	}

	for _, tc := range tierConfig {
		recs, err := svc.List(service.KnowledgeFilters{
			Tier:          tc.tier,
			MinConfidence: tc.minConf,
		})
		if err != nil {
			continue
		}
		for _, rec := range recs {
			scope, _ := rec.Fields["scope"].(string)
			if scope != "project" && scope != role {
				continue
			}
			topic, _ := rec.Fields["topic"].(string)
			content, _ := rec.Fields["content"].(string)
			conf := handoffFloat(rec.Fields, "confidence")
			tier := handoffInt(rec.Fields, "tier")
			entries = append(entries, handoffKnowledgeEntry{
				topic:      topic,
				content:    content,
				confidence: conf,
				tier:       tier,
			})
		}
	}

	// Highest confidence first.
	sort.SliceStable(entries, func(i, j int) bool {
		return entries[i].confidence > entries[j].confidence
	})
	return entries
}

// handoffByteCount estimates the byte size of the assembled prompt.
func handoffByteCount(hctx handoffContextData, taskState map[string]any) int {
	total := 0
	summary, _ := taskState["summary"].(string)
	total += len(summary) * 2 // header + summary paragraph
	for _, s := range hctx.specSections {
		total += len(s.content) + len(s.document) + len(s.section) + 60
	}
	for _, ke := range hctx.knowledge {
		total += len(ke.content) + len(ke.topic) + 10
	}
	for _, c := range hctx.conventions {
		total += len(c) + 3
	}
	for _, f := range hctx.filesPlanned {
		total += len(f) + 3
	}
	return total
}

// trimHandoffContext removes items to stay within the byte budget.
// Trim order: T3 lowest-confidence first, then T2 lowest-confidence, then
// spec sections from the end.
func trimHandoffContext(hctx handoffContextData, taskState map[string]any) handoffContextData {
	var t3, t2 []handoffKnowledgeEntry
	for _, ke := range hctx.knowledge {
		if ke.tier == 3 {
			t3 = append(t3, ke)
		} else {
			t2 = append(t2, ke)
		}
	}
	// Sort ascending so we cut the lowest-confidence entries first.
	sort.SliceStable(t3, func(i, j int) bool { return t3[i].confidence < t3[j].confidence })
	sort.SliceStable(t2, func(i, j int) bool { return t2[i].confidence < t2[j].confidence })

	current := handoffByteCount(hctx, taskState)

	// Trim T3 first.
	for len(t3) > 0 && current > handoffDefaultBudget {
		cut := t3[0]
		t3 = t3[1:]
		sz := len(cut.content) + len(cut.topic) + 10
		current -= sz
		hctx.trimmed = append(hctx.trimmed, handoffTrimmedEntry{
			entryType: "knowledge",
			topic:     cut.topic,
			sizeBytes: sz,
		})
	}

	// Trim T2 next.
	for len(t2) > 0 && current > handoffDefaultBudget {
		cut := t2[0]
		t2 = t2[1:]
		sz := len(cut.content) + len(cut.topic) + 10
		current -= sz
		hctx.trimmed = append(hctx.trimmed, handoffTrimmedEntry{
			entryType: "knowledge",
			topic:     cut.topic,
			sizeBytes: sz,
		})
	}

	// Trim spec sections from the end.
	for len(hctx.specSections) > 0 && current > handoffDefaultBudget {
		cut := hctx.specSections[len(hctx.specSections)-1]
		hctx.specSections = hctx.specSections[:len(hctx.specSections)-1]
		sz := len(cut.content) + len(cut.document) + len(cut.section) + 60
		current -= sz
		hctx.trimmed = append(hctx.trimmed, handoffTrimmedEntry{
			entryType: "spec",
			topic:     cut.section,
			sizeBytes: sz,
		})
	}

	// Rebuild knowledge list: T2 and T3 both re-sorted descending, T2 first.
	sort.SliceStable(t3, func(i, j int) bool { return t3[i].confidence > t3[j].confidence })
	sort.SliceStable(t2, func(i, j int) bool { return t2[i].confidence > t2[j].confidence })
	hctx.knowledge = append(t2, t3...)

	hctx.byteUsage = current
	return hctx
}

// ─── Prompt rendering ─────────────────────────────────────────────────────────

// renderHandoffPrompt builds the Markdown prompt string from assembled context.
func renderHandoffPrompt(taskState map[string]any, hctx handoffContextData, instructions string) string {
	var sb strings.Builder
	taskID, _ := taskState["id"].(string)
	summary, _ := taskState["summary"].(string)

	// Title and summary.
	fmt.Fprintf(&sb, "## Task: %s\n\n", summary)
	fmt.Fprintf(&sb, "### Summary\n\n%s\n\n", summary)

	// Specification sections from doc intelligence.
	for _, s := range hctx.specSections {
		ref := s.document
		if s.section != "" {
			ref = fmt.Sprintf("%s §%s", s.document, s.section)
		}
		fmt.Fprintf(&sb, "### Specification (from %s)\n\n%s\n\n", ref, strings.TrimSpace(s.content))
	}

	// Knowledge constraints.
	if len(hctx.knowledge) > 0 {
		sb.WriteString("### Known Constraints (from knowledge base)\n\n")
		for _, ke := range hctx.knowledge {
			fmt.Fprintf(&sb, "- %s\n", ke.content)
		}
		sb.WriteString("\n")
	}

	// File paths.
	if len(hctx.filesPlanned) > 0 {
		sb.WriteString("### Files\n\n")
		for _, f := range hctx.filesPlanned {
			fmt.Fprintf(&sb, "- %s\n", f)
		}
		sb.WriteString("\n")
	}

	// Conventions (role profile + always-present commit format).
	sb.WriteString("### Conventions\n\n")
	for _, c := range hctx.conventions {
		fmt.Fprintf(&sb, "- %s\n", c)
	}
	if taskID != "" {
		fmt.Fprintf(&sb, "- Commit format: feat(%s): <description>\n", taskID)
	}
	sb.WriteString("\n")

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

// handoffStringSlice extracts a string slice from a task state map field.
func handoffStringSlice(state map[string]any, key string) []string {
	raw, ok := state[key]
	if !ok {
		return nil
	}
	switch v := raw.(type) {
	case []string:
		return v
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				out = append(out, s)
			}
		}
		return out
	}
	return nil
}

// handoffFloat reads a float64 from a fields map.
func handoffFloat(fields map[string]any, key string) float64 {
	v, ok := fields[key]
	if !ok {
		return 0
	}
	switch typed := v.(type) {
	case float64:
		return typed
	case int:
		return float64(typed)
	case int64:
		return float64(typed)
	}
	return 0
}

// handoffInt reads an int from a fields map.
func handoffInt(fields map[string]any, key string) int {
	v, ok := fields[key]
	if !ok {
		return 0
	}
	switch typed := v.(type) {
	case int:
		return typed
	case int64:
		return int(typed)
	case float64:
		return int(typed)
	}
	return 0
}
