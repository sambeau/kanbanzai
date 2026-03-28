package mcp

import (
	"context"
	"fmt"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/sambeau/kanbanzai/internal/config"
	"github.com/sambeau/kanbanzai/internal/git"
	"github.com/sambeau/kanbanzai/internal/knowledge"
	"github.com/sambeau/kanbanzai/internal/service"
	"github.com/sambeau/kanbanzai/internal/storage"
)

// KnowledgeTool returns the 2.0 consolidated knowledge tool.
// It consolidates all 12 knowledge management operations into a single tool
// with an action parameter (spec §18.1).
func KnowledgeTool(svc *service.KnowledgeService) []server.ServerTool {
	return []server.ServerTool{knowledgeTool(svc)}
}

func knowledgeTool(svc *service.KnowledgeService) server.ServerTool {
	tool := mcp.NewTool("knowledge",
		mcp.WithReadOnlyHintAnnotation(false),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(false),
		mcp.WithOpenWorldHintAnnotation(false),
		mcp.WithTitleAnnotation("Knowledge Base"),
		mcp.WithDescription(
			"Query and manage the project knowledge base. "+
				"Use action: list to find knowledge entries by topic, tag, or status — "+
				"this surfaces information that may not be in your context window. "+
				"Routine contribution happens via finish; use this tool for direct management: "+
				"confirming entries, resolving conflicts, checking staleness, pruning. "+
				"Actions: list, get, contribute, confirm, flag, retire, update, promote, "+
				"compact, prune, resolve, staleness.",
		),
		mcp.WithString("action",
			mcp.Required(),
			mcp.Description("Action: list, get, contribute, confirm, flag, retire, update, promote, compact, prune, resolve, staleness"),
		),
		// Shared single-entry parameters
		mcp.WithString("id",
			mcp.Description("Knowledge entry ID (KE-...) — required for get, confirm, flag, retire, update, promote"),
		),
		// contribute parameters
		mcp.WithString("topic",
			mcp.Description("Topic identifier for the knowledge entry (contribute)"),
		),
		mcp.WithString("content",
			mcp.Description("Concise, actionable statement of the knowledge (contribute, update)"),
		),
		mcp.WithString("scope",
			mcp.Description("Scope: a profile name or 'project' (contribute, compact, staleness filter)"),
		),
		mcp.WithNumber("tier",
			mcp.Description("Knowledge tier: 2 (project-level) or 3 (session-level, default) (contribute, prune filter)"),
		),
		mcp.WithString("learned_from",
			mcp.Description("Optional provenance: Task ID or other reference (contribute)"),
		),
		mcp.WithString("created_by",
			mcp.Description("Identity of the contributor — auto-resolved if omitted (contribute)"),
		),
		mcp.WithArray("tags",
			mcp.WithStringItems(),
			mcp.Description("Classification tags (contribute) or tag filter (list)"),
		),
		// flag / retire parameters
		mcp.WithString("reason",
			mcp.Description("Reason for flagging or retiring the entry (flag, retire)"),
		),
		// list parameters
		mcp.WithString("status",
			mcp.Description("Filter by status: contributed, confirmed, disputed, stale, retired (list)"),
		),
		mcp.WithString("topic_filter",
			mcp.Description("Filter by exact normalised topic (list) — use 'topic' for contribute, 'topic_filter' for list"),
		),
		mcp.WithNumber("min_confidence",
			mcp.Description("Minimum confidence score 0.0–1.0 (list)"),
		),
		mcp.WithBoolean("include_retired",
			mcp.Description("Include retired entries in list results (default: false)"),
		),
		// compact / prune
		mcp.WithBoolean("dry_run",
			mcp.Description("Simulate without making changes (compact, prune)"),
		),
		// resolve
		mcp.WithString("keep",
			mcp.Description("ID of the entry to keep (resolve)"),
		),
		mcp.WithString("retire_id",
			mcp.Description("ID of the entry to retire (resolve)"),
		),
		mcp.WithBoolean("merge_content",
			mcp.Description("Merge usage counts and git_anchors from the retired entry (resolve, default: false)"),
		),
		// staleness
		mcp.WithString("entry_id",
			mcp.Description("Optional specific entry ID to check for staleness (staleness)"),
		),
	)

	handler := WithSideEffects(func(ctx context.Context, req mcp.CallToolRequest) (any, error) {
		return DispatchAction(ctx, req, map[string]ActionHandler{
			"list":       knowledgeListAction(svc),
			"get":        knowledgeGetAction(svc),
			"contribute": knowledgeContributeAction(svc),
			"confirm":    knowledgeConfirmAction(svc),
			"flag":       knowledgeFlagAction(svc),
			"retire":     knowledgeRetireAction(svc),
			"update":     knowledgeUpdateAction(svc),
			"promote":    knowledgePromoteAction(svc),
			"compact":    knowledgeCompactAction(svc),
			"prune":      knowledgePruneAction(svc),
			"resolve":    knowledgeResolveAction(svc),
			"staleness":  knowledgeStalenessAction(svc),
		})
	})

	return server.ServerTool{Tool: tool, Handler: handler}
}

// ─── list ─────────────────────────────────────────────────────────────────────

func knowledgeListAction(svc *service.KnowledgeService) ActionHandler {
	return func(ctx context.Context, req mcp.CallToolRequest) (any, error) {
		filters := service.KnowledgeFilters{
			Tier:           int(req.GetFloat("tier", 0)),
			Scope:          req.GetString("scope", ""),
			Status:         req.GetString("status", ""),
			Topic:          req.GetString("topic_filter", ""),
			MinConfidence:  req.GetFloat("min_confidence", 0),
			Tags:           req.GetStringSlice("tags", nil),
			IncludeRetired: req.GetBool("include_retired", false),
		}

		records, err := svc.List(filters)
		if err != nil {
			return nil, fmt.Errorf("list knowledge entries: %w", err)
		}

		entries := make([]map[string]any, 0, len(records))
		for _, rec := range records {
			entries = append(entries, rec.Fields)
		}

		return map[string]any{
			"count":   len(records),
			"entries": entries,
		}, nil
	}
}

// ─── get ──────────────────────────────────────────────────────────────────────

func knowledgeGetAction(svc *service.KnowledgeService) ActionHandler {
	return func(ctx context.Context, req mcp.CallToolRequest) (any, error) {
		id, err := req.RequireString("id")
		if err != nil {
			return nil, fmt.Errorf("id is required for get action")
		}

		record, err := svc.Get(id)
		if err != nil {
			return nil, fmt.Errorf("get knowledge entry: %w", err)
		}

		resp := map[string]any{
			"entry": record.Fields,
		}

		// Check staleness for entries with git_anchors.
		if staleness := checkEntryStaleness(record.Fields, "."); staleness != nil {
			resp["staleness"] = staleness
		}

		return resp, nil
	}
}

// ─── contribute ───────────────────────────────────────────────────────────────

func knowledgeContributeAction(svc *service.KnowledgeService) ActionHandler {
	return func(ctx context.Context, req mcp.CallToolRequest) (any, error) {
		SignalMutation(ctx)

		topic, err := req.RequireString("topic")
		if err != nil {
			return nil, fmt.Errorf("topic is required for contribute action")
		}
		content, err := req.RequireString("content")
		if err != nil {
			return nil, fmt.Errorf("content is required for contribute action")
		}
		scope, err := req.RequireString("scope")
		if err != nil {
			return nil, fmt.Errorf("scope is required for contribute action")
		}

		tier := int(req.GetFloat("tier", 3))
		learnedFrom := req.GetString("learned_from", "")
		createdByRaw := req.GetString("created_by", "")
		createdBy, err := config.ResolveIdentity(createdByRaw)
		if err != nil {
			return nil, err
		}
		tags := req.GetStringSlice("tags", nil)

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
				return map[string]any{
					"accepted":  false,
					"duplicate": true,
					"message":   err.Error(),
					"existing":  duplicate.Fields,
				}, nil
			}
			return nil, fmt.Errorf("contribute knowledge entry: %w", err)
		}

		return map[string]any{
			"accepted": true,
			"message":  "Knowledge entry contributed successfully",
			"entry":    record.Fields,
		}, nil
	}
}

// ─── confirm ──────────────────────────────────────────────────────────────────

func knowledgeConfirmAction(svc *service.KnowledgeService) ActionHandler {
	return func(ctx context.Context, req mcp.CallToolRequest) (any, error) {
		SignalMutation(ctx)

		id, err := req.RequireString("id")
		if err != nil {
			return nil, fmt.Errorf("id is required for confirm action")
		}

		record, err := svc.Confirm(id)
		if err != nil {
			return nil, fmt.Errorf("confirm knowledge entry: %w", err)
		}

		return map[string]any{
			"message": "Knowledge entry confirmed",
			"entry":   record.Fields,
		}, nil
	}
}

// ─── flag ─────────────────────────────────────────────────────────────────────

func knowledgeFlagAction(svc *service.KnowledgeService) ActionHandler {
	return func(ctx context.Context, req mcp.CallToolRequest) (any, error) {
		SignalMutation(ctx)

		id, err := req.RequireString("id")
		if err != nil {
			return nil, fmt.Errorf("id is required for flag action")
		}
		reason := req.GetString("reason", "")
		if reason == "" {
			return inlineErr("missing_parameter", "reason is required for flag action")
		}

		record, err := svc.Flag(id, reason)
		if err != nil {
			return nil, fmt.Errorf("flag knowledge entry: %w", err)
		}

		return map[string]any{
			"message": "Knowledge entry flagged",
			"entry":   record.Fields,
		}, nil
	}
}

// ─── retire ───────────────────────────────────────────────────────────────────

func knowledgeRetireAction(svc *service.KnowledgeService) ActionHandler {
	return func(ctx context.Context, req mcp.CallToolRequest) (any, error) {
		SignalMutation(ctx)

		id, err := req.RequireString("id")
		if err != nil {
			return nil, fmt.Errorf("id is required for retire action")
		}
		reason := req.GetString("reason", "")
		if reason == "" {
			return inlineErr("missing_parameter", "reason is required for retire action")
		}

		record, err := svc.Retire(id, reason)
		if err != nil {
			return nil, fmt.Errorf("retire knowledge entry: %w", err)
		}

		return map[string]any{
			"message": "Knowledge entry retired",
			"entry":   record.Fields,
		}, nil
	}
}

// ─── update ───────────────────────────────────────────────────────────────────

func knowledgeUpdateAction(svc *service.KnowledgeService) ActionHandler {
	return func(ctx context.Context, req mcp.CallToolRequest) (any, error) {
		SignalMutation(ctx)

		id, err := req.RequireString("id")
		if err != nil {
			return nil, fmt.Errorf("id is required for update action")
		}
		content, err := req.RequireString("content")
		if err != nil {
			return nil, fmt.Errorf("content is required for update action")
		}

		record, err := svc.Update(id, content)
		if err != nil {
			return nil, fmt.Errorf("update knowledge entry: %w", err)
		}

		return map[string]any{
			"message": "Knowledge entry updated successfully",
			"entry":   record.Fields,
		}, nil
	}
}

// ─── promote ──────────────────────────────────────────────────────────────────

func knowledgePromoteAction(svc *service.KnowledgeService) ActionHandler {
	return func(ctx context.Context, req mcp.CallToolRequest) (any, error) {
		SignalMutation(ctx)

		id, err := req.RequireString("id")
		if err != nil {
			return nil, fmt.Errorf("id is required for promote action")
		}

		record, err := svc.Promote(id)
		if err != nil {
			return nil, fmt.Errorf("promote knowledge entry: %w", err)
		}

		return map[string]any{
			"message": "Knowledge entry promoted to tier 2",
			"entry":   record.Fields,
		}, nil
	}
}

// ─── compact ──────────────────────────────────────────────────────────────────

func knowledgeCompactAction(svc *service.KnowledgeService) ActionHandler {
	return func(ctx context.Context, req mcp.CallToolRequest) (any, error) {
		dryRun := req.GetBool("dry_run", false)
		scope := req.GetString("scope", "")

		if !dryRun {
			SignalMutation(ctx)
		}

		entries, err := svc.LoadAllRaw()
		if err != nil {
			return nil, fmt.Errorf("load knowledge entries: %w", err)
		}

		fieldMaps := make([]map[string]any, len(entries))
		for i, rec := range entries {
			fieldMaps[i] = rec.Fields
		}

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
					if _, err := svc.UpdateFields(id, fields); err != nil {
						retireErrors = append(retireErrors, fmt.Sprintf("update %s: %v", id, err))
					}
				}
			}
		}

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

		return resp, nil
	}
}

// ─── prune ────────────────────────────────────────────────────────────────────

func knowledgePruneAction(svc *service.KnowledgeService) ActionHandler {
	return func(ctx context.Context, req mcp.CallToolRequest) (any, error) {
		dryRun := req.GetBool("dry_run", false)
		tier := int(req.GetFloat("tier", 0))

		if !dryRun {
			SignalMutation(ctx)
		}

		entries, err := svc.LoadAllRaw()
		if err != nil {
			return nil, fmt.Errorf("load knowledge entries: %w", err)
		}

		fieldMaps := make([]map[string]any, len(entries))
		for i, rec := range entries {
			fieldMaps[i] = rec.Fields
		}

		now := time.Now().UTC()
		ttlCfg := knowledge.DefaultTTLConfig()
		opts := knowledge.PruneOptions{
			DryRun: dryRun,
			Tier:   tier,
		}
		results := knowledge.PruneExpiredEntries(fieldMaps, now, ttlCfg, opts)

		if !dryRun {
			for _, r := range results {
				_, _ = svc.Retire(r.EntryID, r.Reason)
			}
		}

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
			"dry_run": dryRun,
		}
		if dryRun {
			resp["would_prune"] = prunedList
		} else {
			resp["pruned"] = prunedList
		}

		return resp, nil
	}
}

// ─── resolve ──────────────────────────────────────────────────────────────────

func knowledgeResolveAction(svc *service.KnowledgeService) ActionHandler {
	return func(ctx context.Context, req mcp.CallToolRequest) (any, error) {
		SignalMutation(ctx)

		keepID, err := req.RequireString("keep")
		if err != nil {
			return nil, fmt.Errorf("keep is required for resolve action")
		}
		retireID, err := req.RequireString("retire_id")
		if err != nil {
			return nil, fmt.Errorf("retire_id is required for resolve action")
		}
		mergeContent := req.GetBool("merge_content", false)

		keepRec, err := svc.Get(keepID)
		if err != nil {
			return nil, fmt.Errorf("get keep entry: %w", err)
		}
		retireRec, err := svc.Get(retireID)
		if err != nil {
			return nil, fmt.Errorf("get retire entry: %w", err)
		}

		if mergeContent {
			keepUseCount := knowledgeFieldInt(keepRec.Fields, "use_count")
			retireUseCount := knowledgeFieldInt(retireRec.Fields, "use_count")
			keepMissCount := knowledgeFieldInt(keepRec.Fields, "miss_count")
			retireMissCount := knowledgeFieldInt(retireRec.Fields, "miss_count")

			mergedFields := map[string]any{
				"use_count":   keepUseCount + retireUseCount,
				"miss_count":  keepMissCount + retireMissCount,
				"merged_from": retireID,
			}

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
				return nil, fmt.Errorf("update kept entry with merged data: %w", err)
			}
		}

		if _, err := svc.Retire(retireID, "resolved conflict: merged into "+keepID); err != nil {
			return nil, fmt.Errorf("retire entry: %w", err)
		}

		keepStatus, _ := keepRec.Fields["status"].(string)
		if keepStatus == "disputed" {
			_, _ = svc.Confirm(keepID)
		}

		return map[string]any{
			"resolved": map[string]any{
				"kept":    keepID,
				"retired": retireID,
				"merged":  mergeContent,
			},
		}, nil
	}
}

// ─── staleness ────────────────────────────────────────────────────────────────

func knowledgeStalenessAction(svc *service.KnowledgeService) ActionHandler {
	return func(ctx context.Context, req mcp.CallToolRequest) (any, error) {
		entryID := req.GetString("entry_id", "")
		scope := req.GetString("scope", "")
		repoPath := "."

		var entries []storage.KnowledgeRecord
		var err error

		if entryID != "" {
			record, gerr := svc.Get(entryID)
			if gerr != nil {
				return nil, fmt.Errorf("get knowledge entry: %w", gerr)
			}
			entries = []storage.KnowledgeRecord{record}
		} else {
			entries, err = svc.LoadAllRaw()
			if err != nil {
				return nil, fmt.Errorf("load knowledge entries: %w", err)
			}
		}

		var staleEntries []map[string]any
		for _, rec := range entries {
			if scope != "" {
				entryScope, _ := rec.Fields["scope"].(string)
				if entryScope != scope {
					continue
				}
			}

			anchors := knowledge.GetGitAnchors(rec.Fields)
			if len(anchors) == 0 {
				continue
			}

			staleness := checkEntryStaleness(rec.Fields, repoPath)
			if staleness != nil && staleness["is_stale"] == true {
				staleEntries = append(staleEntries, map[string]any{
					"entry_id":  rec.ID,
					"topic":     rec.Fields["topic"],
					"staleness": staleness,
				})
			}
		}

		return map[string]any{
			"stale_entries": staleEntries,
			"total_checked": len(entries),
		}, nil
	}
}

// ─── helpers (moved from knowledge_tools.go) ──────────────────────────────────

func checkEntryStaleness(fields map[string]any, repoPath string) map[string]any {
	anchorPaths := knowledge.GetGitAnchors(fields)
	if len(anchorPaths) == 0 {
		return nil
	}
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
	info, err := git.CheckStaleness(repoPath, anchors, lastConfirmed)
	if err != nil {
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
