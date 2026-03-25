package mcp

import (
	"time"

	chk "kanbanzai/internal/checkpoint"
	"kanbanzai/internal/health"
	"kanbanzai/internal/service"
	"kanbanzai/internal/validate"
	"kanbanzai/internal/worktree"
)

// Phase4aHealthChecker returns an AdditionalHealthChecker that validates
// dependency cycles, stalled dispatches, and estimation coverage.
func Phase4aHealthChecker(
	entitySvc *service.EntityService,
	worktreeStore *worktree.Store,
	checkpointStore *chk.Store,
	stallThresholdDays int,
	repoPath string,
) AdditionalHealthChecker {
	return func() (*validate.HealthReport, error) {
		report := &validate.HealthReport{
			Summary: validate.HealthSummary{
				EntitiesByType: make(map[string]int),
			},
		}

		// Load all tasks.
		tasks, err := entitySvc.List("task")
		if err != nil {
			report.Warnings = append(report.Warnings, validate.ValidationWarning{
				EntityType: "dependency_cycles",
				Message:    "failed to load tasks for health checks: " + err.Error(),
			})
			report.Summary.WarningCount++
			return report, nil
		}

		taskMaps := make([]map[string]any, len(tasks))
		for i, t := range tasks {
			taskMaps[i] = t.State
		}

		// Load all features (best-effort; skip checks if unavailable).
		features, _ := entitySvc.List("feature")
		featureMaps := make([]map[string]any, len(features))
		for i, f := range features {
			featureMaps[i] = f.State
		}

		// Build worktree branch map: entity ID → branch name.
		worktreeBranches := make(map[string]string)
		if worktrees, err := worktreeStore.List(); err == nil {
			for _, wt := range worktrees {
				worktreeBranches[wt.EntityID] = wt.Branch
			}
		}

		// Dependency cycle detection.
		cycleResult := health.CheckDependencyCycles(taskMaps)
		mergeHealthResult(report, "dependency_cycles", cycleResult)

		// Stalled dispatch detection.
		stalledResult := health.CheckStalledDispatches(taskMaps, worktreeBranches, repoPath, stallThresholdDays)
		mergeHealthResult(report, "stalled_dispatches", stalledResult)

		// Estimation coverage check: features in developing or in-progress status.
		activeStatuses := map[string]struct{}{
			"developing":  {},
			"in-progress": {},
		}
		coverageResult := health.CheckEstimationCoverage(featureMaps, taskMaps, activeStatuses)
		mergeHealthResult(report, "estimation_coverage", coverageResult)

		// checkpointStore reserved for future checkpoint health checks.
		_ = checkpointStore
		// time imported for future time-based checks.
		_ = time.Now()

		return report, nil
	}
}
