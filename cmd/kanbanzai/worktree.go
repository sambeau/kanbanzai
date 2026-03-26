package main

import (
	"fmt"
	"strings"
	"time"

	"kanbanzai/internal/config"
	"kanbanzai/internal/core"
	"kanbanzai/internal/git"
	"kanbanzai/internal/service"
	"kanbanzai/internal/worktree"
)

// runWorktree handles the worktree subcommand.
func runWorktree(args []string, deps dependencies) error {
	if len(args) == 0 || wantsHelp(args) {
		fmt.Fprint(deps.stdout, worktreeUsageText)
		return nil
	}

	switch args[0] {
	case "list":
		return runWorktreeList(args[1:], deps)
	case "create":
		return runWorktreeCreate(args[1:], deps)
	case "show":
		return runWorktreeShow(args[1:], deps)
	case "remove":
		return runWorktreeRemove(args[1:], deps)
	default:
		return fmt.Errorf("unknown worktree subcommand %q\n\n%s", args[0], worktreeUsageText)
	}
}

// runBranch handles the branch subcommand.
func runBranch(args []string, deps dependencies) error {
	if len(args) == 0 || wantsHelp(args) {
		fmt.Fprint(deps.stdout, branchUsageText)
		return nil
	}

	switch args[0] {
	case "status":
		return runBranchStatus(args[1:], deps)
	case "list":
		return runBranchList(deps)
	default:
		return fmt.Errorf("unknown branch subcommand %q\n\n%s", args[0], branchUsageText)
	}
}

func runWorktreeList(args []string, deps dependencies) error {
	flags, err := parseFlags(args)
	if err != nil {
		return err
	}

	statusFilter := flags["status"]
	if statusFilter == "" {
		statusFilter = "all"
	}

	store := worktree.NewStore(core.StatePath())
	records, err := store.List()
	if err != nil {
		return fmt.Errorf("list worktrees: %w", err)
	}

	// Apply filters
	var filtered []worktree.Record
	for _, r := range records {
		if statusFilter != "all" && string(r.Status) != statusFilter {
			continue
		}
		filtered = append(filtered, r)
	}

	if len(filtered) == 0 {
		fmt.Fprintln(deps.stdout, "no worktrees found")
		return nil
	}

	fmt.Fprintf(deps.stdout, "%-20s  %-20s  %-40s  %s\n", "ID", "ENTITY", "BRANCH", "STATUS")
	for _, r := range filtered {
		fmt.Fprintf(deps.stdout, "%-20s  %-20s  %-40s  %s\n", r.ID, r.EntityID, r.Branch, r.Status)
	}
	return nil
}

func runWorktreeCreate(args []string, deps dependencies) error {
	if len(args) == 0 {
		return fmt.Errorf("missing entity ID\n\n%s", worktreeUsageText)
	}

	entityID := args[0]
	flags, err := parseFlags(args[1:])
	if err != nil {
		return err
	}

	branchName := flags["branch"]
	slug := flags["slug"]
	createdByRaw := flags["created_by"]
	if createdByRaw == "" {
		createdByRaw = flags["created-by"]
	}

	// Validate entity type
	entityType := worktreeEntityType(entityID)
	if entityType == "" {
		return fmt.Errorf("invalid entity type: ID must start with FEAT- or BUG-")
	}

	// Verify entity exists
	entitySvc := service.NewEntityService(core.StatePath())
	entity, err := entitySvc.Get(entityType, entityID, "")
	if err != nil {
		return fmt.Errorf("get entity: %w", err)
	}

	store := worktree.NewStore(core.StatePath())
	gitOps := worktree.NewGit(".")

	// Check if worktree already exists
	existing, err := store.GetByEntityID(entityID)
	if err == nil && existing.ID != "" {
		return fmt.Errorf("worktree already exists for entity %s: %s", entityID, existing.ID)
	}

	// Resolve identity
	createdBy, err := config.ResolveIdentity(createdByRaw)
	if err != nil {
		return err
	}

	// Get slug from entity if not provided
	if slug == "" {
		if s, ok := entity.State["slug"].(string); ok {
			slug = s
		}
	}

	// Generate branch name and path
	if branchName == "" {
		branchName = worktree.GenerateBranchName(entityID, slug)
	}
	wtPath := worktree.GenerateWorktreePath(entityID, slug)

	// Create the git worktree with a new branch
	if err := gitOps.CreateWorktreeNewBranch(wtPath, branchName, ""); err != nil {
		return fmt.Errorf("create git worktree: %w", err)
	}

	// Create the worktree record
	record := worktree.Record{
		EntityID:  entityID,
		Branch:    branchName,
		Path:      wtPath,
		Status:    worktree.StatusActive,
		Created:   time.Now().UTC(),
		CreatedBy: createdBy,
	}

	created, err := store.Create(record)
	if err != nil {
		// Try to clean up the git worktree if record creation fails
		_ = gitOps.RemoveWorktree(wtPath, true)
		return fmt.Errorf("create worktree record: %w", err)
	}

	fmt.Fprintf(deps.stdout, "created worktree %s\n", created.ID)
	fmt.Fprintf(deps.stdout, "  entity: %s\n", created.EntityID)
	fmt.Fprintf(deps.stdout, "  branch: %s\n", created.Branch)
	fmt.Fprintf(deps.stdout, "  path:   %s\n", created.Path)
	return nil
}

func runWorktreeShow(args []string, deps dependencies) error {
	if len(args) == 0 {
		return fmt.Errorf("missing entity ID\n\n%s", worktreeUsageText)
	}

	entityID := args[0]
	store := worktree.NewStore(core.StatePath())

	record, err := store.GetByEntityID(entityID)
	if err != nil {
		return fmt.Errorf("get worktree: %w", err)
	}

	fmt.Fprintf(deps.stdout, "ID:         %s\n", record.ID)
	fmt.Fprintf(deps.stdout, "Entity:     %s\n", record.EntityID)
	fmt.Fprintf(deps.stdout, "Branch:     %s\n", record.Branch)
	fmt.Fprintf(deps.stdout, "Path:       %s\n", record.Path)
	fmt.Fprintf(deps.stdout, "Status:     %s\n", record.Status)
	fmt.Fprintf(deps.stdout, "Created:    %s\n", record.Created.Format(time.RFC3339))
	fmt.Fprintf(deps.stdout, "Created By: %s\n", record.CreatedBy)
	if record.MergedAt != nil {
		fmt.Fprintf(deps.stdout, "Merged At:  %s\n", record.MergedAt.Format(time.RFC3339))
	}
	if record.CleanupAfter != nil {
		fmt.Fprintf(deps.stdout, "Cleanup After: %s\n", record.CleanupAfter.Format(time.RFC3339))
	}
	return nil
}

func runWorktreeRemove(args []string, deps dependencies) error {
	if len(args) == 0 {
		return fmt.Errorf("missing entity ID\n\n%s", worktreeUsageText)
	}

	entityID := args[0]
	flags, err := parseFlags(args[1:])
	if err != nil {
		return err
	}

	force := flags["force"] == "true"

	store := worktree.NewStore(core.StatePath())
	gitOps := worktree.NewGit(".")

	record, err := store.GetByEntityID(entityID)
	if err != nil {
		return fmt.Errorf("get worktree: %w", err)
	}

	// Remove the git worktree
	if err := gitOps.RemoveWorktree(record.Path, force); err != nil {
		errStr := err.Error()
		if !force && (strings.Contains(errStr, "uncommitted") ||
			strings.Contains(errStr, "untracked") ||
			strings.Contains(errStr, "changes")) {
			return fmt.Errorf("worktree has uncommitted changes, use --force to remove anyway")
		}
		return fmt.Errorf("remove git worktree: %w", err)
	}

	// Delete the worktree record
	if err := store.Delete(record.ID); err != nil {
		return fmt.Errorf("delete worktree record: %w", err)
	}

	fmt.Fprintf(deps.stdout, "removed worktree %s (path: %s)\n", record.ID, record.Path)
	return nil
}

func runBranchStatus(args []string, deps dependencies) error {
	if len(args) == 0 {
		return fmt.Errorf("missing entity ID\n\n%s", branchUsageText)
	}

	entityID := args[0]
	store := worktree.NewStore(core.StatePath())

	record, err := store.GetByEntityID(entityID)
	if err != nil {
		return fmt.Errorf("get worktree: %w", err)
	}

	cfg := config.LoadOrDefault()
	thresholds := git.BranchThresholds{
		StaleAfterDays:      cfg.BranchTracking.StaleAfterDays,
		DriftWarningCommits: cfg.BranchTracking.DriftWarningCommits,
		DriftErrorCommits:   cfg.BranchTracking.DriftErrorCommits,
	}

	status, err := git.EvaluateBranchStatus(".", record.Branch, thresholds)
	if err != nil {
		return fmt.Errorf("evaluate branch status: %w", err)
	}

	fmt.Fprintf(deps.stdout, "Branch: %s\n", record.Branch)
	fmt.Fprintf(deps.stdout, "\nMetrics:\n")
	fmt.Fprintf(deps.stdout, "  Branch age (days):     %d\n", status.Metrics.BranchAgeDays)
	fmt.Fprintf(deps.stdout, "  Commits behind main:   %d\n", status.Metrics.CommitsBehindMain)
	fmt.Fprintf(deps.stdout, "  Commits ahead of main: %d\n", status.Metrics.CommitsAheadOfMain)
	fmt.Fprintf(deps.stdout, "  Last commit:           %s\n", status.Metrics.LastCommitAt.Format(time.RFC3339))
	fmt.Fprintf(deps.stdout, "  Last commit age (days): %d\n", status.Metrics.LastCommitAgeDays)
	fmt.Fprintf(deps.stdout, "  Has conflicts:         %t\n", status.Metrics.HasConflicts)

	if len(status.Warnings) > 0 {
		fmt.Fprintf(deps.stdout, "\nWarnings:\n")
		for _, w := range status.Warnings {
			fmt.Fprintf(deps.stdout, "  - %s\n", w)
		}
	}

	if len(status.Errors) > 0 {
		fmt.Fprintf(deps.stdout, "\nErrors:\n")
		for _, e := range status.Errors {
			fmt.Fprintf(deps.stdout, "  - %s\n", e)
		}
	}

	return nil
}

func runBranchList(deps dependencies) error {
	store := worktree.NewStore(core.StatePath())
	records, err := store.List()
	if err != nil {
		return fmt.Errorf("list worktrees: %w", err)
	}

	// Filter to active worktrees only
	var active []worktree.Record
	for _, r := range records {
		if r.Status == worktree.StatusActive {
			active = append(active, r)
		}
	}

	if len(active) == 0 {
		fmt.Fprintln(deps.stdout, "no active worktrees with branches")
		return nil
	}

	fmt.Fprintf(deps.stdout, "%-20s  %s\n", "ENTITY", "BRANCH")
	for _, r := range active {
		fmt.Fprintf(deps.stdout, "%-20s  %s\n", r.EntityID, r.Branch)
	}
	return nil
}

// worktreeEntityType extracts the entity type from an entity ID.
func worktreeEntityType(id string) string {
	upper := strings.ToUpper(id)
	if strings.HasPrefix(upper, "FEAT-") {
		return "feature"
	}
	if strings.HasPrefix(upper, "BUG-") {
		return "bug"
	}
	return ""
}

const worktreeUsageText = `kanbanzai worktree <subcommand> [flags]

Manage Git worktrees for feature and bug development.

Subcommands:
  list     List all worktrees
  create   Create a new worktree for an entity
  show     Show details of a worktree
  remove   Remove a worktree

Examples:
  kbz worktree list
  kbz worktree list --status=active
  kbz worktree create FEAT-01JX...
  kbz worktree create FEAT-01JX... --branch=feature/custom-name
  kbz worktree show FEAT-01JX...
  kbz worktree remove FEAT-01JX...
  kbz worktree remove FEAT-01JX... --force
`

const branchUsageText = `kanbanzai branch <subcommand> [flags]

Check branch health for worktree branches.

Subcommands:
  status   Show branch health metrics for an entity's worktree
  list     List branches for all active worktrees

Examples:
  kbz branch status FEAT-01JX...
  kbz branch list
`
