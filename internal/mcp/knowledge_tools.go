package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"kanbanzai/internal/config"
	"kanbanzai/internal/git"
	"kanbanzai/internal/knowledge"
	"kanbanzai/internal/service"
	"kanbanzai/internal/storage"
)

// KnowledgeTools returns all knowledge entry MCP tool definitions with their handlers.
func KnowledgeTools(svc *service.KnowledgeService) []server.ServerTool {
	return []server.ServerTool{
		knowledgeContributeTool(svc),
		knowledgeGetTool(svc),
		knowledgeListTool(svc),
		knowledgeUpdateTool(svc),
		knowledgeConfirmTool(svc),
		knowledgeFlagTool(svc),
		knowledgeRetireTool(svc),
		knowledgePromoteTool(svc),
		knowledgeContextReportTool(svc),
		// Phase 3 lifecycle tools
		knowledgeCheckStalenessTool(svc),
		knowledgePruneTool(svc),
		knowledgeCompactTool(svc),
		knowledgeResolveConflictTool(svc),
	}
}

func knowledgeContributeTool(svc *service.KnowledgeService) server.ServerTool {
	tool := mcp.NewTool("knowledge_contribute",
		mcp.WithDescription("Contribute a new knowledge entry to the shared knowledge base. Topics are normalised (lowercased, hyphenated). Duplicate detection rejects entries with an identical topic or similar content (Jaccard > 0.65) in the same scope."),
		mcp.WithString("topic", mcp.Description("Topic identifier for the knowledge entry (will be normalised)"), mcp.Required()),
		mcp.WithString("content", mcp.Description("Concise, actionable statement of the knowledge"), mcp.Required()),
		mcp.WithString("scope", mcp.Description("Scope of the entry: a profile name or \"project\""), mcp.Required()),
		mcp.WithNumber("tier", mcp.Description("Knowledge tier: 2 (project-level) or 3 (session-level, default)")),
		mcp.WithString("learned_from", mcp.Description("Optional provenance: Task ID or other reference")),
		mcp.WithString("created_by", mcp.Description("Identity of the contributor")),
		mcp.WithArray("tags", mcp.WithStringItems(), mcp.Description("Optional classification tags")),
	)
	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		topic, err := request.RequireString("topic")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		content, err := request.RequireString("content")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		scope, err := request.RequireString("scope")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		tier := int(request.GetFloat("tier", 3))
		learnedFrom := request.GetString("learned_from", "")
		createdByRaw := request.GetString("created_by", "")
		createdBy, err := config.ResolveIdentity(createdByRaw)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		tags := request.GetStringSlice("tags", nil)

		input := service.ContributeInput{
			Topic:       topic,
			Content:     content,
			Scope:       scope,
			Tier:        tier,
			LearnedFrom: learnedFrom,
			CreatedBy:   createdBy,
			Tags:        tags,
		}

		record, duplicate, err := svc.Contribute(input)
		if err != nil {
			if duplicate != nil {
				resp := map[string]any{
					"success":   false,
					"duplicate": true,
					"message":   err.Error(),
					"existing":  duplicate.Fields,
				}
				return knowledgeMapJSON(resp)
			}
			return mcp.NewToolResultErrorFromErr("contribute knowledge entry failed", err), nil
		}

		resp := map[string]any{
			"success": true,
			"message": "Knowledge entry contributed successfully",
			"entry":   record.Fields,
		}
		return knowledgeMapJSON(resp)
	}
	return server.ServerTool{Tool: tool, Handler: handler}
}

func knowledgeGetTool(svc *service.KnowledgeService) server.ServerTool {
	tool := mcp.NewTool("knowledge_get",
		mcp.WithDescription("Get a knowledge entry by ID. Includes staleness information for entries with git_anchors."),
		mcp.WithString("id", mcp.Description("Knowledge entry ID (KE-...)"), mcp.Required()),
	)
	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		id, err := request.RequireString("id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		record, err := svc.Get(id)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("get knowledge entry failed", err), nil
		}

		resp := map[string]any{
			"success": true,
			"entry":   record.Fields,
		}

		// Check staleness for entries with git_anchors
		if staleness := checkEntryStaleness(record.Fields, "."); staleness != nil {
			resp["staleness"] = staleness
		}

		return knowledgeMapJSON(resp)
	}
	return server.ServerTool{Tool: tool, Handler: handler}
}

func knowledgeListTool(svc *service.KnowledgeService) server.ServerTool {
	tool := mcp.NewTool("knowledge_list",
		mcp.WithDescription("List knowledge entries with optional filters. Retired entries are excluded by default."),
		mcp.WithNumber("tier", mcp.Description("Filter by tier: 2 or 3")),
		mcp.WithString("scope", mcp.Description("Filter by scope")),
		mcp.WithString("status", mcp.Description("Filter by status: contributed, confirmed, disputed, stale, retired")),
		mcp.WithString("topic", mcp.Description("Filter by exact normalised topic")),
		mcp.WithNumber("min_confidence", mcp.Description("Minimum confidence score (0.0–1.0)")),
		mcp.WithArray("tags", mcp.WithStringItems(), mcp.Description("Filter: entries must have all of these tags")),
		mcp.WithBoolean("include_retired", mcp.Description("Include retired entries (default: false)")),
	)
	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		filters := service.KnowledgeFilters{
			Tier:           int(request.GetFloat("tier", 0)),
			Scope:          request.GetString("scope", ""),
			Status:         request.GetString("status", ""),
			Topic:          request.GetString("topic", ""),
			MinConfidence:  request.GetFloat("min_confidence", 0),
			Tags:           request.GetStringSlice("tags", nil),
			IncludeRetired: request.GetBool("include_retired", false),
		}

		records, err := svc.List(filters)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("list knowledge entries failed", err), nil
		}

		entries := make([]map[string]any, 0, len(records))
		for _, rec := range records {
			entries = append(entries, rec.Fields)
		}

		resp := map[string]any{
			"success": true,
			"count":   len(records),
			"entries": entries,
		}
		return knowledgeMapJSON(resp)
	}
	return server.ServerTool{Tool: tool, Handler: handler}
}

func knowledgeUpdateTool(svc *service.KnowledgeService) server.ServerTool {
	tool := mcp.NewTool("knowledge_update",
		mcp.WithDescription("Update the content of a knowledge entry. Resets use_count, miss_count, and confidence to defaults."),
		mcp.WithString("id", mcp.Description("Knowledge entry ID (KE-...)"), mcp.Required()),
		mcp.WithString("content", mcp.Description("New content for the entry"), mcp.Required()),
	)
	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		id, err := request.RequireString("id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		content, err := request.RequireString("content")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		record, err := svc.Update(id, content)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("update knowledge entry failed", err), nil
		}

		resp := map[string]any{
			"success": true,
			"message": "Knowledge entry updated successfully",
			"entry":   record.Fields,
		}
		return knowledgeMapJSON(resp)
	}
	return server.ServerTool{Tool: tool, Handler: handler}
}

func knowledgeConfirmTool(svc *service.KnowledgeService) server.ServerTool {
	tool := mcp.NewTool("knowledge_confirm",
		mcp.WithDescription("Manually confirm a knowledge entry, transitioning it from contributed or disputed to confirmed status."),
		mcp.WithString("id", mcp.Description("Knowledge entry ID (KE-...)"), mcp.Required()),
	)
	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		id, err := request.RequireString("id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		record, err := svc.Confirm(id)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("confirm knowledge entry failed", err), nil
		}

		resp := map[string]any{
			"success": true,
			"message": "Knowledge entry confirmed",
			"entry":   record.Fields,
		}
		return knowledgeMapJSON(resp)
	}
	return server.ServerTool{Tool: tool, Handler: handler}
}

func knowledgeFlagTool(svc *service.KnowledgeService) server.ServerTool {
	tool := mcp.NewTool("knowledge_flag",
		mcp.WithDescription("Flag a knowledge entry as incorrect or disputed. Increments miss_count and recomputes confidence. If miss_count reaches 2, the entry is automatically retired."),
		mcp.WithString("id", mcp.Description("Knowledge entry ID (KE-...)"), mcp.Required()),
		mcp.WithString("reason", mcp.Description("Reason for flagging the entry"), mcp.Required()),
	)
	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		id, err := request.RequireString("id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		reason := request.GetString("reason", "")
		if reason == "" {
			return mcp.NewToolResultError("missing required parameter: reason"), nil
		}

		record, err := svc.Flag(id, reason)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("flag knowledge entry failed", err), nil
		}

		resp := map[string]any{
			"success": true,
			"message": "Knowledge entry flagged",
			"entry":   record.Fields,
		}
		return knowledgeMapJSON(resp)
	}
	return server.ServerTool{Tool: tool, Handler: handler}
}

func knowledgeRetireTool(svc *service.KnowledgeService) server.ServerTool {
	tool := mcp.NewTool("knowledge_retire",
		mcp.WithDescription("Manually retire a knowledge entry, marking it as no longer valid. Retired entries are excluded from listing by default."),
		mcp.WithString("id", mcp.Description("Knowledge entry ID (KE-...)"), mcp.Required()),
		mcp.WithString("reason", mcp.Description("Reason for retiring the entry"), mcp.Required()),
	)
	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		id, err := request.RequireString("id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		reason := request.GetString("reason", "")
		if reason == "" {
			return mcp.NewToolResultError("missing required parameter: reason"), nil
		}

		record, err := svc.Retire(id, reason)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("retire knowledge entry failed", err), nil
		}

		resp := map[string]any{
			"success": true,
			"message": "Knowledge entry retired",
			"entry":   record.Fields,
		}
		return knowledgeMapJSON(resp)
	}
	return server.ServerTool{Tool: tool, Handler: handler}
}

func knowledgePromoteTool(svc *service.KnowledgeService) server.ServerTool {
	tool := mcp.NewTool("knowledge_promote",
		mcp.WithDescription("Promote a tier-3 knowledge entry to tier 2 in place, extending its TTL from 30 to 90 days."),
		mcp.WithString("id", mcp.Description("Knowledge entry ID (KE-...)"), mcp.Required()),
	)
	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		id, err := request.RequireString("id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		record, err := svc.Promote(id)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("promote knowledge entry failed", err), nil
		}

		resp := map[string]any{
			"success": true,
			"message": "Knowledge entry promoted to tier 2",
			"entry":   record.Fields,
		}
		return knowledgeMapJSON(resp)
	}
	return server.ServerTool{Tool: tool, Handler: handler}
}

func knowledgeContextReportTool(svc *service.KnowledgeService) server.ServerTool {
	tool := mcp.NewTool("context_report",
		mcp.WithDescription("Report knowledge entry usage from a completed task. For each used entry: increments use_count and updates last_used; auto-confirms if use_count >= 3 and miss_count == 0. For each flagged entry: increments miss_count; auto-retires if miss_count >= 2."),
		mcp.WithString("task_id", mcp.Description("ID of the task that consumed the knowledge entries"), mcp.Required()),
		mcp.WithArray("used", mcp.WithStringItems(), mcp.Description("List of knowledge entry IDs that were used and found helpful"), mcp.Required()),
		mcp.WithString("flagged", mcp.Description("JSON array of flagged entries: [{\"entry_id\": \"KE-...\", \"reason\": \"...\"}]")),
	)
	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		taskID, err := request.RequireString("task_id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		used := request.GetStringSlice("used", nil)
		if len(used) == 0 {
			return mcp.NewToolResultError("missing required parameter: used (list of knowledge entry IDs)"), nil
		}

		flaggedRaw := request.GetString("flagged", "")
		flagged, err := service.ParseFlaggedEntries(flaggedRaw)
		if err != nil {
			return mcp.NewToolResultError("parse flagged entries: " + err.Error()), nil
		}

		if err := svc.ContextReport(taskID, used, flagged); err != nil {
			return mcp.NewToolResultErrorFromErr("context report failed", err), nil
		}

		resp := map[string]any{
			"success":       true,
			"task_id":       taskID,
			"used_count":    len(used),
			"flagged_count": len(flagged),
			"message":       "Context report processed successfully",
		}
		return knowledgeMapJSON(resp)
	}
	return server.ServerTool{Tool: tool, Handler: handler}
}

// knowledgeMapJSON marshals a map to JSON and returns it as a tool result.
func knowledgeMapJSON(v map[string]any) (*mcp.CallToolResult, error) {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return mcp.NewToolResultError("marshal result: " + err.Error()), nil
	}
	return mcp.NewToolResultText(string(data)), nil
}

// knowledgeCheckStalenessTool checks staleness of knowledge entries with git_anchors.
func knowledgeCheckStalenessTool(svc *service.KnowledgeService) server.ServerTool {
	tool := mcp.NewTool("knowledge_check_staleness",
		mcp.WithDescription("Check staleness of knowledge entries that have git_anchors. An entry is stale if any anchored file was modified after the entry was last confirmed."),
		mcp.WithString("entry_id", mcp.Description("Optional: check a specific knowledge entry ID (KE-...). If omitted, checks all anchored entries.")),
		mcp.WithString("scope", mcp.Description("Optional: filter entries by scope")),
	)
	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		entryID := request.GetString("entry_id", "")
		scope := request.GetString("scope", "")
		repoPath := "." // Current directory as repo root

		var entries []storage.KnowledgeRecord
		var err error

		if entryID != "" {
			// Check a specific entry
			record, gerr := svc.Get(entryID)
			if gerr != nil {
				return mcp.NewToolResultErrorFromErr("get knowledge entry failed", gerr), nil
			}
			entries = []storage.KnowledgeRecord{record}
		} else {
			// Check all entries
			entries, err = svc.LoadAllRaw()
			if err != nil {
				return mcp.NewToolResultErrorFromErr("load knowledge entries failed", err), nil
			}
		}

		var staleEntries []map[string]any

		for _, rec := range entries {
			// Filter by scope if specified
			if scope != "" {
				entryScope, _ := rec.Fields["scope"].(string)
				if entryScope != scope {
					continue
				}
			}

			// Skip entries without git_anchors
			anchors := knowledge.GetGitAnchors(rec.Fields)
			if len(anchors) == 0 {
				continue
			}

			// Check staleness
			staleness := checkEntryStaleness(rec.Fields, repoPath)
			if staleness != nil && staleness["is_stale"] == true {
				staleEntry := map[string]any{
					"entry_id":  rec.ID,
					"topic":     rec.Fields["topic"],
					"staleness": staleness,
				}
				staleEntries = append(staleEntries, staleEntry)
			}
		}

		resp := map[string]any{
			"success":       true,
			"stale_entries": staleEntries,
			"total_checked": len(entries),
		}
		return knowledgeMapJSON(resp)
	}
	return server.ServerTool{Tool: tool, Handler: handler}
}

// knowledgePruneTool prunes expired knowledge entries based on TTL rules.
func knowledgePruneTool(svc *service.KnowledgeService) server.ServerTool {
	tool := mcp.NewTool("knowledge_prune",
		mcp.WithDescription("Prune expired knowledge entries based on TTL rules. Tier-3 entries expire after 30 days without use, tier-2 after 90 days."),
		mcp.WithBoolean("dry_run", mcp.Description("If true, report what would be pruned without actually pruning (default: false)")),
		mcp.WithNumber("tier", mcp.Description("Optional: only prune entries of this tier (2 or 3)")),
	)
	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		dryRun := request.GetBool("dry_run", false)
		tier := int(request.GetFloat("tier", 0))

		// Load all entries
		entries, err := svc.LoadAllRaw()
		if err != nil {
			return mcp.NewToolResultErrorFromErr("load knowledge entries failed", err), nil
		}

		// Convert to field maps for pruning
		fieldMaps := make([]map[string]any, len(entries))
		for i, rec := range entries {
			fieldMaps[i] = rec.Fields
		}

		// Get prune results
		now := time.Now().UTC()
		config := knowledge.DefaultTTLConfig()
		opts := knowledge.PruneOptions{
			DryRun: dryRun,
			Tier:   tier,
		}
		results := knowledge.PruneExpiredEntries(fieldMaps, now, config, opts)

		// If not dry run, actually retire the entries
		if !dryRun {
			for _, r := range results {
				_, _ = svc.Retire(r.EntryID, r.Reason)
			}
		}

		// Build response
		var prunedList []map[string]any
		for _, r := range results {
			prunedList = append(prunedList, map[string]any{
				"entry_id": r.EntryID,
				"topic":    r.Topic,
				"tier":     r.Tier,
				"reason":   r.Reason,
			})
		}

		resp := map[string]any{
			"success": true,
			"dry_run": dryRun,
		}
		if dryRun {
			resp["would_prune"] = prunedList
		} else {
			resp["pruned"] = prunedList
		}

		return knowledgeMapJSON(resp)
	}
	return server.ServerTool{Tool: tool, Handler: handler}
}

// knowledgeCompactTool compacts knowledge entries by merging duplicates.
func knowledgeCompactTool(svc *service.KnowledgeService) server.ServerTool {
	tool := mcp.NewTool("knowledge_compact",
		mcp.WithDescription("Compact knowledge entries by merging duplicates and near-duplicates, and flagging contradictions. Tier-3 entries are auto-merged; tier-2 entries are flagged for review."),
		mcp.WithBoolean("dry_run", mcp.Description("If true, report what would be compacted without actually compacting (default: false)")),
		mcp.WithString("scope", mcp.Description("Optional: only compact entries in this scope")),
	)
	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		dryRun := request.GetBool("dry_run", false)
		scope := request.GetString("scope", "")

		// Load all entries
		entries, err := svc.LoadAllRaw()
		if err != nil {
			return mcp.NewToolResultErrorFromErr("load knowledge entries failed", err), nil
		}

		// Convert to field maps for compaction
		fieldMaps := make([]map[string]any, len(entries))
		for i, rec := range entries {
			fieldMaps[i] = rec.Fields
		}

		// Run compaction
		opts := knowledge.CompactionOptions{
			DryRun: dryRun,
			Scope:  scope,
		}
		result, updates := knowledge.CompactEntries(fieldMaps, opts)

		var retireErrors []string
		if !dryRun {
			for _, fields := range updates {
				id, _ := fields["id"].(string)
				if id == "" {
					continue
				}
				status, _ := fields["status"].(string)
				if status == "retired" {
					reason, _ := fields["retired_reason"].(string)
					if reason == "" {
						reason, _ = fields["deprecated_reason"].(string)
					}
					if _, err := svc.Retire(id, reason); err != nil {
						retireErrors = append(retireErrors, fmt.Sprintf("%s: %v", id, err))
					}
				} else {
					// Persist kept/disputed entries with their updated fields
					if _, err := svc.UpdateFields(id, fields); err != nil {
						retireErrors = append(retireErrors, fmt.Sprintf("update %s: %v", id, err))
					}
				}
			}
		}

		// Build response details
		var details []map[string]any
		for _, d := range result.Details {
			detail := map[string]any{
				"action": string(d.Action),
				"reason": d.Reason,
			}
			if d.Kept != "" {
				detail["kept"] = d.Kept
			}
			if d.Discarded != "" {
				detail["discarded"] = d.Discarded
			}
			if len(d.Entries) > 0 {
				detail["entries"] = d.Entries
			}
			details = append(details, detail)
		}

		resp := map[string]any{
			"success": true,
			"dry_run": dryRun,
			"compaction_result": map[string]any{
				"duplicates_merged":      result.DuplicatesMerged,
				"near_duplicates_merged": result.NearDuplicatesMerged,
				"conflicts_flagged":      result.ConflictsFlagged,
				"details":                details,
			},
		}
		if len(retireErrors) > 0 {
			resp["warnings"] = retireErrors
		}

		return knowledgeMapJSON(resp)
	}
	return server.ServerTool{Tool: tool, Handler: handler}
}

// knowledgeResolveConflictTool resolves a conflict between two knowledge entries.
func knowledgeResolveConflictTool(svc *service.KnowledgeService) server.ServerTool {
	tool := mcp.NewTool("knowledge_resolve_conflict",
		mcp.WithDescription("Resolve a conflict between two knowledge entries by keeping one and retiring the other. Optionally merge content from the retired entry into the kept entry."),
		mcp.WithString("keep", mcp.Description("ID of the knowledge entry to keep (KE-...)"), mcp.Required()),
		mcp.WithString("retire", mcp.Description("ID of the knowledge entry to retire (KE-...)"), mcp.Required()),
		mcp.WithBoolean("merge_content", mcp.Description("If true, merge usage counts and git_anchors from the retired entry into the kept entry (default: false)")),
	)
	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		keepID, err := request.RequireString("keep")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		retireID, err := request.RequireString("retire")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		mergeContent := request.GetBool("merge_content", false)

		// Load both entries
		keepRec, err := svc.Get(keepID)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("get keep entry failed", err), nil
		}
		retireRec, err := svc.Get(retireID)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("get retire entry failed", err), nil
		}

		if mergeContent {
			// Transfer usage counts and anchors from retire to keep
			keepUseCount := knowledgeFieldInt(keepRec.Fields, "use_count")
			retireUseCount := knowledgeFieldInt(retireRec.Fields, "use_count")

			keepMissCount := knowledgeFieldInt(keepRec.Fields, "miss_count")
			retireMissCount := knowledgeFieldInt(retireRec.Fields, "miss_count")

			mergedFields := map[string]any{
				"use_count":   keepUseCount + retireUseCount,
				"miss_count":  keepMissCount + retireMissCount,
				"merged_from": retireID,
			}

			// Merge git_anchors (union)
			keepAnchors := knowledge.GetGitAnchors(keepRec.Fields)
			retireAnchors := knowledge.GetGitAnchors(retireRec.Fields)
			if len(retireAnchors) > 0 {
				seen := make(map[string]struct{})
				for _, a := range keepAnchors {
					seen[a] = struct{}{}
				}
				for _, a := range retireAnchors {
					if _, ok := seen[a]; !ok {
						keepAnchors = append(keepAnchors, a)
					}
				}
				mergedFields["git_anchors"] = keepAnchors
			}

			if _, err := svc.UpdateFields(keepID, mergedFields); err != nil {
				return mcp.NewToolResultErrorFromErr("update kept entry with merged data failed", err), nil
			}
		}

		// Retire the entry
		reason := "resolved conflict: merged into " + keepID
		_, err = svc.Retire(retireID, reason)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("retire entry failed", err), nil
		}

		// Confirm the kept entry if it was disputed
		keepStatus, _ := keepRec.Fields["status"].(string)
		if keepStatus == "disputed" {
			_, _ = svc.Confirm(keepID)
		}

		resp := map[string]any{
			"success": true,
			"resolved": map[string]any{
				"kept":    keepID,
				"retired": retireID,
				"merged":  mergeContent,
			},
		}
		return knowledgeMapJSON(resp)
	}
	return server.ServerTool{Tool: tool, Handler: handler}
}

// checkEntryStaleness checks if a knowledge entry is stale based on its git_anchors.
// Returns a map with staleness info, or nil if the entry has no anchors.
func checkEntryStaleness(fields map[string]any, repoPath string) map[string]any {
	anchorPaths := knowledge.GetGitAnchors(fields)
	if len(anchorPaths) == 0 {
		return nil
	}

	// Convert string paths to GitAnchor structs
	anchors := make([]git.GitAnchor, len(anchorPaths))
	for i, path := range anchorPaths {
		anchors[i] = git.GitAnchor{Path: path}
	}

	// Get last confirmed time
	var lastConfirmed time.Time
	if confirmedStr, ok := fields["last_confirmed"].(string); ok && confirmedStr != "" {
		lastConfirmed, _ = time.Parse(time.RFC3339, confirmedStr)
	} else if updatedStr, ok := fields["updated"].(string); ok && updatedStr != "" {
		// Fall back to updated time if no explicit confirmation
		lastConfirmed, _ = time.Parse(time.RFC3339, updatedStr)
	}

	info, err := git.CheckStaleness(repoPath, anchors, lastConfirmed)
	if err != nil {
		// Git error - return stale with error reason
		return map[string]any{
			"is_stale":     true,
			"stale_reason": "git check failed: " + err.Error(),
		}
	}

	result := map[string]any{
		"is_stale":             info.IsStale,
		"entry_last_confirmed": lastConfirmed.Format(time.RFC3339),
	}

	if info.IsStale {
		result["stale_reason"] = info.StaleReason
		var staleFiles []map[string]any
		for _, sf := range info.StaleFiles {
			staleFiles = append(staleFiles, map[string]any{
				"path":        sf.Path,
				"modified_at": sf.ModifiedAt.Format(time.RFC3339),
				"commit":      sf.Commit,
			})
		}
		result["stale_files"] = staleFiles
	}

	return result
}

// knowledgeFieldInt reads an integer value from a Fields map.
func knowledgeFieldInt(fields map[string]any, key string) int {
	v := fields[key]
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
