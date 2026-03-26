package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	kbzctx "kanbanzai/internal/context"
	"kanbanzai/internal/git"
	"kanbanzai/internal/knowledge"
	"kanbanzai/internal/service"
)

// ContextTools returns the context assembly MCP tool.
func ContextTools(
	profileStore *kbzctx.ProfileStore,
	knowledgeSvc *service.KnowledgeService,
	entitySvc *service.EntityService,
	intelligenceSvc *service.IntelligenceService,
) []server.ServerTool {
	return []server.ServerTool{
		contextAssembleTool(profileStore, knowledgeSvc, entitySvc, intelligenceSvc),
	}
}

func contextAssembleTool(
	profileStore *kbzctx.ProfileStore,
	knowledgeSvc *service.KnowledgeService,
	entitySvc *service.EntityService,
	intelligenceSvc *service.IntelligenceService,
) server.ServerTool {
	tool := mcp.NewTool("context_assemble",
		mcp.WithDescription("Assemble a context packet for an agent role. Returns the role profile, relevant knowledge entries (Tier 2 and Tier 3 scoped to the role or project), design context from document intelligence (if task_id provided), and task instructions — all within the byte budget. Profile and task instructions are never trimmed. When over budget, lowest-confidence Tier 3 is trimmed first, then Tier 2, then design context."),
		mcp.WithString("role", mcp.Description("Profile ID for the agent role (e.g. \"backend\", \"frontend\")"), mcp.Required()),
		mcp.WithString("task_id", mcp.Description("Optional task entity ID to include task instructions and design context")),
		mcp.WithNumber("max_bytes", mcp.Description("Maximum byte budget for the assembled packet (default: 30720)")),
		mcp.WithString("orchestration_context", mcp.Description("Optional handoff note from the orchestrating agent, injected as an ephemeral Tier 3 entry into the context packet. Not persisted to the knowledge store.")),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
	)
	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		role, err := request.RequireString("role")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		taskID := request.GetString("task_id", "")
		maxBytes := int(request.GetFloat("max_bytes", 0))
		orchestrationContext := request.GetString("orchestration_context", "")

		input := kbzctx.AssemblyInput{
			Role:                 role,
			TaskID:               taskID,
			MaxBytes:             maxBytes,
			OrchestrationContext: orchestrationContext,
		}

		result, err := kbzctx.Assemble(input, profileStore, knowledgeSvc, entitySvc, intelligenceSvc)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("context_assemble failed", err), nil
		}

		type stalenessInfo struct {
			IsStale    bool     `json:"is_stale"`
			StaleFiles []string `json:"stale_files,omitempty"`
		}

		type responseItem struct {
			Source     string         `json:"source"`
			EntryID    string         `json:"entry_id,omitempty"`
			Priority   string         `json:"priority"`
			Confidence float64        `json:"confidence,omitempty"`
			Content    string         `json:"content"`
			Staleness  *stalenessInfo `json:"staleness,omitempty"`
		}

		// Build a map of entry IDs to their fields for staleness checking
		entryFieldsMap := make(map[string]map[string]any)
		if knowledgeSvc != nil {
			for _, item := range result.Items {
				if item.EntryID != "" {
					rec, err := knowledgeSvc.Get(item.EntryID)
					if err == nil {
						entryFieldsMap[item.EntryID] = rec.Fields
					}
				}
			}
		}

		items := make([]responseItem, 0, len(result.Items))
		for _, item := range result.Items {
			ri := responseItem{
				Source:   string(item.Source),
				Priority: item.Priority,
				Content:  item.Content,
			}
			if item.EntryID != "" {
				ri.EntryID = item.EntryID

				// Check staleness for knowledge entries with git_anchors
				if fields, ok := entryFieldsMap[item.EntryID]; ok {
					anchorPaths := knowledge.GetGitAnchors(fields)
					if len(anchorPaths) > 0 {
						anchors := make([]git.GitAnchor, len(anchorPaths))
						for i, path := range anchorPaths {
							anchors[i] = git.GitAnchor{Path: path}
						}

						var lastConfirmed time.Time
						if confirmedStr, ok := fields["last_confirmed"].(string); ok && confirmedStr != "" {
							lastConfirmed, _ = time.Parse(time.RFC3339, confirmedStr)
						} else if updatedStr, ok := fields["updated"].(string); ok && updatedStr != "" {
							lastConfirmed, _ = time.Parse(time.RFC3339, updatedStr)
						}

						info, err := git.CheckStaleness(".", anchors, lastConfirmed)
						if err == nil && info.IsStale {
							var staleFilePaths []string
							for _, sf := range info.StaleFiles {
								staleFilePaths = append(staleFilePaths, sf.Path)
							}
							ri.Staleness = &stalenessInfo{
								IsStale:    true,
								StaleFiles: staleFilePaths,
							}
						}
					}
				}
			}
			if item.Confidence > 0 {
				ri.Confidence = item.Confidence
			}
			items = append(items, ri)
		}

		type trimmedEntry struct {
			EntryID   string `json:"entry_id,omitempty"`
			Type      string `json:"type"`
			Topic     string `json:"topic,omitempty"`
			Tier      int    `json:"tier,omitempty"`
			SizeBytes int    `json:"size_bytes"`
		}

		trimmedItems := make([]trimmedEntry, len(result.TrimmedEntries))
		for i, te := range result.TrimmedEntries {
			trimmedItems[i] = trimmedEntry{
				EntryID:   te.EntryID,
				Type:      te.Type,
				Topic:     te.Topic,
				Tier:      te.Tier,
				SizeBytes: te.SizeBytes,
			}
		}

		resp := map[string]any{
			"success":    true,
			"role":       result.Role,
			"byte_count": result.ByteCount,
			"trimmed":    trimmedItems,
			"items":      items,
		}
		if result.TaskID != "" {
			resp["task_id"] = result.TaskID
		}

		return contextMapJSON(resp)
	}
	return server.ServerTool{Tool: tool, Handler: handler}
}

// contextMapJSON marshals a map to JSON and returns it as a tool result.
func contextMapJSON(v map[string]any) (*mcp.CallToolResult, error) {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("marshal result: %s", err)), nil
	}
	return mcp.NewToolResultText(string(data)), nil
}
