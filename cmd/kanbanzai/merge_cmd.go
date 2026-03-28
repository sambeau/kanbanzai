package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/sambeau/kanbanzai/internal/config"
	"github.com/sambeau/kanbanzai/internal/core"
	kbzgit "github.com/sambeau/kanbanzai/internal/git"
	"github.com/sambeau/kanbanzai/internal/merge"
	"github.com/sambeau/kanbanzai/internal/service"
	"github.com/sambeau/kanbanzai/internal/worktree"
)

func runMerge(args []string, deps dependencies) error {
	if len(args) == 0 {
		return fmt.Errorf("missing merge subcommand\n\n%s", mergeUsageText)
	}

	switch args[0] {
	case "check":
		return runMergeCheck(args[1:], deps)
	case "run":
		return runMergeRun(args[1:], deps)
	default:
		return fmt.Errorf("unknown merge subcommand %q\n\n%s", args[0], mergeUsageText)
	}
}

func runMergeCheck(args []string, deps dependencies) error {
	if len(args) == 0 {
		return fmt.Errorf("missing entity ID\n\n%s", mergeUsageText)
	}

	entityID := args[0]
	store := worktree.NewStore(core.StatePath())
	entitySvc := service.NewEntityService(core.StatePath())

	record, err := store.GetByEntityID(entityID)
	if err != nil {
		return fmt.Errorf("get worktree: %w", err)
	}

	entityType := mergeEntityType(entityID)
	if entityType == "" {
		return fmt.Errorf("invalid entity type: ID must start with FEAT- or BUG-")
	}

	entity, err := entitySvc.Get(entityType, entityID, "")
	if err != nil {
		return fmt.Errorf("get entity: %w", err)
	}

	cfg := config.LoadOrDefault()
	thresholds := kbzgit.BranchThresholds{
		StaleAfterDays:      cfg.BranchTracking.StaleAfterDays,
		DriftWarningCommits: cfg.BranchTracking.DriftWarningCommits,
		DriftErrorCommits:   cfg.BranchTracking.DriftErrorCommits,
	}

	// Gather tasks for the entity
	tasks := gatherTasks(entitySvc, entityID)

	gateCtx := merge.GateContext{
		RepoPath:   ".",
		EntityID:   entityID,
		Branch:     record.Branch,
		Entity:     entity.State,
		Tasks:      tasks,
		Thresholds: thresholds,
	}

	result := merge.CheckGates(gateCtx)

	fmt.Fprintf(deps.stdout, "Entity:  %s\n", result.EntityID)
	fmt.Fprintf(deps.stdout, "Branch:  %s\n", result.Branch)
	fmt.Fprintf(deps.stdout, "Status:  %s\n", result.OverallStatus)
	fmt.Fprintln(deps.stdout)

	fmt.Fprintln(deps.stdout, "Gates:")
	for _, g := range result.Gates {
		icon := "✓"
		if g.Status == merge.GateStatusFailed {
			icon = "✗"
		} else if g.Status == merge.GateStatusWarning {
			icon = "⚠"
		}
		line := fmt.Sprintf("  %s %-25s [%s]", icon, g.Name, g.Severity)
		if g.Message != "" {
			line += fmt.Sprintf("  %s", g.Message)
		}
		fmt.Fprintln(deps.stdout, line)
	}

	return nil
}

func runMergeRun(args []string, deps dependencies) error {
	if len(args) == 0 {
		return fmt.Errorf("missing entity ID\n\n%s", mergeUsageText)
	}

	entityID := args[0]
	flags, err := parseFlags(args[1:])
	if err != nil {
		return err
	}

	override := flags["override"] == "true"
	overrideReason := flags["reason"]
	strategy := flags["strategy"]
	if strategy == "" {
		strategy = "squash"
	}
	deleteBranch := flags["delete-branch"] != "false"

	if override && overrideReason == "" {
		return fmt.Errorf("--reason is required when using --override")
	}

	store := worktree.NewStore(core.StatePath())
	entitySvc := service.NewEntityService(core.StatePath())
	gitOps := worktree.NewGit(".")

	record, err := store.GetByEntityID(entityID)
	if err != nil {
		return fmt.Errorf("get worktree: %w", err)
	}

	entityType := mergeEntityType(entityID)
	if entityType == "" {
		return fmt.Errorf("invalid entity type: ID must start with FEAT- or BUG-")
	}

	entity, err := entitySvc.Get(entityType, entityID, "")
	if err != nil {
		return fmt.Errorf("get entity: %w", err)
	}

	cfg := config.LoadOrDefault()
	thresholds := kbzgit.BranchThresholds{
		StaleAfterDays:      cfg.BranchTracking.StaleAfterDays,
		DriftWarningCommits: cfg.BranchTracking.DriftWarningCommits,
		DriftErrorCommits:   cfg.BranchTracking.DriftErrorCommits,
	}

	tasks := gatherTasks(entitySvc, entityID)

	gateCtx := merge.GateContext{
		RepoPath:   ".",
		EntityID:   entityID,
		Branch:     record.Branch,
		Entity:     entity.State,
		Tasks:      tasks,
		Thresholds: thresholds,
	}

	result := merge.CheckGates(gateCtx)

	if result.OverallStatus == "blocked" && !override {
		fmt.Fprintln(deps.stdout, "Merge blocked by failing gates:")
		for _, g := range result.Gates {
			if g.Status == merge.GateStatusFailed && g.Severity == merge.GateSeverityBlocking {
				fmt.Fprintf(deps.stdout, "  ✗ %s: %s\n", g.Name, g.Message)
			}
		}
		fmt.Fprintln(deps.stdout)
		fmt.Fprintln(deps.stdout, "Use --override --reason=\"...\" to force merge.")
		return fmt.Errorf("merge blocked by failing gates")
	}

	// Perform the merge
	var mergeStrategy worktree.MergeStrategy
	switch strategy {
	case "squash":
		mergeStrategy = worktree.MergeStrategySquash
	case "merge":
		mergeStrategy = worktree.MergeStrategyMerge
	case "rebase":
		mergeStrategy = worktree.MergeStrategyRebase
	default:
		return fmt.Errorf("unknown merge strategy %q; use squash, merge, or rebase", strategy)
	}

	commitMsg := fmt.Sprintf("feat(%s): merge %s", entityID, record.Branch)
	mergeResult, mergeErr := gitOps.MergeBranch(record.Branch, mergeStrategy, commitMsg)
	if mergeErr != nil {
		return fmt.Errorf("git merge failed: %w", mergeErr)
	}

	now := time.Now().UTC()

	// Update worktree record
	record.MarkMerged(now, cfg.Cleanup.GracePeriodDays)
	if _, err := store.Update(record); err != nil {
		return fmt.Errorf("update worktree record: %w", err)
	}

	// Log override if applicable
	if override {
		createdBy, _ := config.ResolveIdentity("")
		fmt.Fprintf(deps.stdout, "Override applied by %s: %s\n", createdBy, overrideReason)
	}

	fmt.Fprintf(deps.stdout, "Merged %s (%s) into main\n", entityID, record.Branch)
	fmt.Fprintf(deps.stdout, "  strategy: %s\n", strategy)
	fmt.Fprintf(deps.stdout, "  merge commit: %s\n", mergeResult.MergeCommit)
	fmt.Fprintf(deps.stdout, "  merged at: %s\n", now.Format(time.RFC3339))
	if record.CleanupAfter != nil {
		fmt.Fprintf(deps.stdout, "  cleanup after: %s\n", record.CleanupAfter.Format(time.RFC3339))
	}

	// Delete branch if requested
	if deleteBranch {
		if err := gitOps.DeleteBranch(record.Branch, false); err != nil {
			fmt.Fprintf(deps.stdout, "  warning: could not delete branch: %v\n", err)
		} else {
			fmt.Fprintf(deps.stdout, "  branch deleted: %s\n", record.Branch)
		}
	}

	return nil
}

// gatherTasks returns task state maps for an entity's child tasks.
func gatherTasks(entitySvc *service.EntityService, entityID string) []map[string]any {
	results, err := entitySvc.List("task")
	if err != nil {
		return nil
	}

	var tasks []map[string]any
	for _, r := range results {
		parent, _ := r.State["parent_id"].(string)
		if parent == entityID {
			tasks = append(tasks, r.State)
		}
	}
	return tasks
}

func mergeEntityType(id string) string {
	upper := strings.ToUpper(id)
	if strings.HasPrefix(upper, "FEAT-") {
		return "feature"
	}
	if strings.HasPrefix(upper, "BUG-") {
		return "bug"
	}
	return ""
}

const mergeUsageText = `kanbanzai merge <subcommand> [flags]

Check merge readiness and execute merges.

Subcommands:
  check   Check merge gate status for an entity
  run     Execute merge to main

Examples:
  kbz merge check FEAT-01JX...
  kbz merge run FEAT-01JX...
  kbz merge run FEAT-01JX... --strategy=squash
  kbz merge run FEAT-01JX... --override --reason="Hotfix - will backfill tests"
  kbz merge run FEAT-01JX... --delete-branch=false
`
