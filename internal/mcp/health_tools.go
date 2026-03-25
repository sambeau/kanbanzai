package mcp

import (
	"time"

	"kanbanzai/internal/config"
	"kanbanzai/internal/git"
	"kanbanzai/internal/health"
	"kanbanzai/internal/service"
	"kanbanzai/internal/validate"
	"kanbanzai/internal/worktree"
)

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
		case health.SeverityWarning:
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
