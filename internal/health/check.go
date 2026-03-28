package health

import (
	"time"

	"github.com/sambeau/kanbanzai/internal/git"
	"github.com/sambeau/kanbanzai/internal/worktree"
)

// CheckOptions configures the health check.
type CheckOptions struct {
	// RepoPath is the path to the Git repository root.
	RepoPath string

	// BranchThresholds configures staleness and drift thresholds for branch evaluation.
	BranchThresholds git.BranchThresholds

	// IncludeOK includes categories with no issues in the result.
	IncludeOK bool

	// SkipBranchCheck skips branch health checks (useful when git is unavailable).
	SkipBranchCheck bool

	// SkipStalenessCheck skips knowledge staleness checks (useful when git is unavailable).
	SkipStalenessCheck bool

	// Features is a slice of feature entity field maps for entity consistency checks.
	Features []map[string]any

	// Tasks is a slice of task entity field maps for entity consistency checks.
	Tasks []map[string]any
}

// DefaultCheckOptions returns default options with sensible defaults.
func DefaultCheckOptions() CheckOptions {
	return CheckOptions{
		BranchThresholds: git.DefaultBranchThresholds(),
		IncludeOK:        false,
	}
}

// RunHealthCheck runs all health checks and returns combined result.
func RunHealthCheck(
	worktrees []worktree.Record,
	entries []map[string]any,
	now time.Time,
	opts CheckOptions,
) HealthResult {
	categories := make(map[string]CategoryResult)

	// Run worktree checks
	worktreeResult := CheckWorktree(opts.RepoPath, worktrees)
	if opts.IncludeOK || worktreeResult.Status != SeverityOK {
		categories["worktree"] = worktreeResult
	}

	// Run branch checks (requires git)
	if !opts.SkipBranchCheck {
		branchResult := CheckBranch(opts.RepoPath, worktrees, opts.BranchThresholds)
		if opts.IncludeOK || branchResult.Status != SeverityOK {
			categories["branch"] = branchResult
		}
	}

	// Run knowledge staleness checks (requires git)
	if !opts.SkipStalenessCheck {
		stalenessResult := CheckKnowledgeStaleness(opts.RepoPath, entries)
		if opts.IncludeOK || stalenessResult.Status != SeverityOK {
			categories["knowledge_staleness"] = stalenessResult
		}
	}

	// Run knowledge TTL checks
	ttlResult := CheckKnowledgeTTL(entries, now)
	if opts.IncludeOK || ttlResult.Status != SeverityOK {
		categories["knowledge_ttl"] = ttlResult
	}

	// Run knowledge conflict checks
	conflictResult := CheckKnowledgeConflicts(entries)
	if opts.IncludeOK || conflictResult.Status != SeverityOK {
		categories["knowledge_conflicts"] = conflictResult
	}

	// Run cleanup checks
	cleanupResult := CheckCleanup(worktrees, now)
	if opts.IncludeOK || cleanupResult.Status != SeverityOK {
		categories["cleanup"] = cleanupResult
	}

	// Run feature-child state consistency checks
	featureChildResult := CheckFeatureChildConsistency(opts.Features, opts.Tasks)
	if opts.IncludeOK || featureChildResult.Status != SeverityOK {
		categories["feature_child_consistency"] = featureChildResult
	}

	// Run worktree-branch merged checks (best-effort, requires git)
	worktreeMergedResult := CheckWorktreeBranchMerged(opts.RepoPath, worktrees)
	if opts.IncludeOK || worktreeMergedResult.Status != SeverityOK {
		categories["worktree_branch_merged"] = worktreeMergedResult
	}

	return HealthResult{
		Status:     DetermineOverallStatus(categories),
		Categories: categories,
	}
}

// DetermineOverallStatus returns the worst status from all categories.
func DetermineOverallStatus(categories map[string]CategoryResult) Severity {
	overall := SeverityOK

	for _, cat := range categories {
		overall = WorstSeverity(overall, cat.Status)
		// Short-circuit if we already found an error
		if overall == SeverityError {
			return overall
		}
	}

	return overall
}
