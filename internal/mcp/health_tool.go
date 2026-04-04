package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/sambeau/kanbanzai/internal/config"
	"github.com/sambeau/kanbanzai/internal/git"
	"github.com/sambeau/kanbanzai/internal/health"
	"github.com/sambeau/kanbanzai/internal/service"
	"github.com/sambeau/kanbanzai/internal/validate"
	"github.com/sambeau/kanbanzai/internal/worktree"
)

// AdditionalHealthChecker is a function that performs additional health checks
// and returns a report to be merged into the main health check result.
type AdditionalHealthChecker func() (*validate.HealthReport, error)

// GateOverrideHealthChecker returns an AdditionalHealthChecker that loads all
// features and calls CheckGateOverrides to flag any that used gate overrides.
func GateOverrideHealthChecker(entitySvc *service.EntityService) AdditionalHealthChecker {
	return func() (*validate.HealthReport, error) {
		report := &validate.HealthReport{
			Summary: validate.HealthSummary{
				EntitiesByType: make(map[string]int),
			},
		}

		features, err := entitySvc.List("feature")
		if err != nil {
			// Best-effort: skip gate override check if features cannot be loaded.
			return report, nil
		}

		featureMaps := make([]map[string]any, len(features))
		for i, f := range features {
			featureMaps[i] = f.State
		}

		overrideResult := health.CheckGateOverrides(featureMaps)
		mergeHealthResult(report, "gate_overrides", overrideResult)

		return report, nil
	}
}

// HealthTool returns the 2.0 health tool.
// It replaces the 1.0 health_check tool with the same behaviour but under the
// 2.0 naming convention (tool name: "health", registered in GroupCore).
func HealthTool(entitySvc *service.EntityService, additionalCheckers ...AdditionalHealthChecker) []server.ServerTool {
	return []server.ServerTool{healthTool(entitySvc, additionalCheckers...)}
}

func healthTool(entitySvc *service.EntityService, additionalCheckers ...AdditionalHealthChecker) server.ServerTool {
	tool := mcp.NewTool("health",
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
		mcp.WithOpenWorldHintAnnotation(false),
		mcp.WithTitleAnnotation("System Health Check"),
		mcp.WithDescription(
			"A comprehensive health check across all entities, knowledge entries, worktrees, "+
				"branches, and context profiles — call periodically or when diagnosing unexpected "+
				"workflow errors. Returns a structured report of errors and warnings with category "+
				"breakdowns. Use INSTEAD OF manually inspecting individual entities for consistency "+
				"issues. Do NOT use for entity-specific queries — use status for dashboards or "+
				"entity(action: \"get\") for individual lookups. No parameters required.",
		),
	)

	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		report, err := entitySvc.HealthCheck()
		if err != nil {
			return mcp.NewToolResultErrorFromErr("Cannot run health check: entity validation failed", err), nil
		}

		for _, checker := range additionalCheckers {
			additional, err := checker()
			if err != nil {
				return mcp.NewToolResultErrorFromErr("Cannot run health check: additional checker failed", err), nil
			}
			if additional != nil {
				mergeHealthReports(report, additional)
			}
		}

		return jsonResult(report)
	}

	return server.ServerTool{Tool: tool, Handler: handler}
}

// mergeHealthReports merges src into dst in-place.
func mergeHealthReports(dst, src *validate.HealthReport) {
	if dst == nil || src == nil {
		return
	}
	dst.Errors = append(dst.Errors, src.Errors...)
	dst.Warnings = append(dst.Warnings, src.Warnings...)

	// Merge entity counts.
	if dst.Summary.EntitiesByType == nil {
		dst.Summary.EntitiesByType = make(map[string]int)
	}
	for k, v := range src.Summary.EntitiesByType {
		dst.Summary.EntitiesByType[k] += v
	}
	dst.Summary.TotalEntities += src.Summary.TotalEntities
}

// Phase3HealthChecker returns an AdditionalHealthChecker that validates
// worktrees, branches, knowledge entries, and cleanup status.
func Phase3HealthChecker(
	worktreeStore *worktree.Store,
	knowledgeSvc *service.KnowledgeService,
	cfg *config.Config,
	repoPath string,
) AdditionalHealthChecker {
	return func() (*validate.HealthReport, error) {
		report := &validate.HealthReport{
			Summary: validate.HealthSummary{
				EntitiesByType: make(map[string]int),
			},
		}

		// Get worktree records
		worktrees, err := worktreeStore.List()
		if err != nil {
			report.Errors = append(report.Errors, validate.ValidationError{
				EntityType: "worktree",
				Message:    "failed to list worktrees: " + err.Error(),
			})
			report.Summary.ErrorCount++
			return report, nil
		}

		// Check worktree state consistency
		worktreeResult := health.CheckWorktree(repoPath, worktrees)
		mergeHealthResult(report, "worktree", worktreeResult)

		// Check branch health
		thresholds := git.BranchThresholds{
			StaleAfterDays:      cfg.BranchTracking.StaleAfterDays,
			DriftWarningCommits: cfg.BranchTracking.DriftWarningCommits,
			DriftErrorCommits:   cfg.BranchTracking.DriftErrorCommits,
		}
		branchResult := health.CheckBranch(repoPath, worktrees, thresholds)
		mergeHealthResult(report, "branch", branchResult)

		// Check cleanup status
		now := time.Now()
		cleanupResult := health.CheckCleanup(worktrees, now)
		mergeHealthResult(report, "cleanup", cleanupResult)

		// Check worktrees whose branch is already merged into main (best-effort).
		worktreeMergedResult := health.CheckWorktreeBranchMerged(repoPath, worktrees)
		mergeHealthResult(report, "worktree_branch_merged", worktreeMergedResult)

		// Get knowledge entries for knowledge health checks
		records, err := knowledgeSvc.LoadAllRaw()
		if err != nil {
			report.Errors = append(report.Errors, validate.ValidationError{
				EntityType: "knowledge",
				Message:    "failed to load knowledge entries: " + err.Error(),
			})
			report.Summary.ErrorCount++
			return report, nil
		}

		// Convert records to the format expected by health checks
		entries := make([]map[string]any, len(records))
		for i, r := range records {
			entries[i] = r.Fields
		}

		// Check knowledge staleness
		stalenessResult := health.CheckKnowledgeStaleness(repoPath, entries)
		mergeHealthResult(report, "knowledge_staleness", stalenessResult)

		// Check knowledge TTL
		ttlResult := health.CheckKnowledgeTTL(entries, now)
		mergeHealthResult(report, "knowledge_ttl", ttlResult)

		// Check knowledge conflicts
		conflictsResult := health.CheckKnowledgeConflicts(entries)
		mergeHealthResult(report, "knowledge_conflicts", conflictsResult)

		return report, nil
	}
}

// mergeHealthResult converts a health.CategoryResult to errors/warnings and adds them to the report.
func mergeHealthResult(report *validate.HealthReport, category string, result health.CategoryResult) {
	for _, issue := range result.Issues {
		switch issue.Severity {
		case health.SeverityError:
			report.Errors = append(report.Errors, validate.ValidationError{
				EntityType: category,
				EntityID:   coalesce(issue.EntityID, issue.EntryID),
				Field:      "",
				Message:    issue.Message,
			})
			report.Summary.ErrorCount++
		case health.SeverityWarning, health.SeverityInfo:
			report.Warnings = append(report.Warnings, validate.ValidationWarning{
				EntityType: category,
				EntityID:   coalesce(issue.EntityID, issue.EntryID),
				Field:      "",
				Message:    issue.Message,
			})
			report.Summary.WarningCount++
		}
	}
}

// coalesce returns the first non-empty string.
func coalesce(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

func jsonResult(v any) (*mcp.CallToolResult, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("Cannot format health check result: JSON serialisation failed.\n\nTo resolve:\n  Report this as a bug — the health check data could not be serialised", err), nil
	}
	return mcp.NewToolResultText(string(data)), nil
}

// ToolHintsHealthChecker returns an AdditionalHealthChecker that reports
// the merged tool hints configuration (FR-018, FR-019).
func ToolHintsHealthChecker(mergedHints map[string]string) AdditionalHealthChecker {
	return func() (*validate.HealthReport, error) {
		report := &validate.HealthReport{
			Summary: validate.HealthSummary{
				EntitiesByType: make(map[string]int),
			},
		}

		if len(mergedHints) == 0 {
			report.Warnings = append(report.Warnings, validate.ValidationWarning{
				EntityType: "tool_hints",
				Message:    "No tool hints configured. Add tool_hints to .kbz/local.yaml to guide agents toward available MCP tools.",
			})
			report.Summary.WarningCount = len(report.Warnings)
			return report, nil
		}

		for role, hint := range mergedHints {
			display := hint
			if len(display) > 80 {
				display = display[:77] + "..."
			}
			report.Warnings = append(report.Warnings, validate.ValidationWarning{
				EntityType: "tool_hints",
				Message:    fmt.Sprintf("Role %q: %s", role, display),
			})
		}

		report.Summary.WarningCount = len(report.Warnings)
		return report, nil
	}
}
