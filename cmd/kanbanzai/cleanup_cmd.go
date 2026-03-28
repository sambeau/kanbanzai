package main

import (
	"fmt"
	"time"

	"github.com/sambeau/kanbanzai/internal/cleanup"
	"github.com/sambeau/kanbanzai/internal/config"
	"github.com/sambeau/kanbanzai/internal/core"
	"github.com/sambeau/kanbanzai/internal/worktree"
)

func runCleanup(args []string, deps dependencies) error {
	if len(args) == 0 {
		return fmt.Errorf("missing cleanup subcommand\n\n%s", cleanupUsageText)
	}

	switch args[0] {
	case "list":
		return runCleanupList(args[1:], deps)
	case "run":
		return runCleanupRun(args[1:], deps)
	default:
		return fmt.Errorf("unknown cleanup subcommand %q\n\n%s", args[0], cleanupUsageText)
	}
}

func runCleanupList(args []string, deps dependencies) error {
	flags, err := parseFlags(args)
	if err != nil {
		return err
	}

	store := worktree.NewStore(core.StatePath())
	records, err := store.List()
	if err != nil {
		return fmt.Errorf("list worktrees: %w", err)
	}

	now := time.Now().UTC()

	showPending := flags["pending"] != "false"
	showScheduled := flags["scheduled"] != "false"
	showAbandoned := flags["abandoned"] != "false"

	opts := cleanup.ListOptions{
		IncludePending:   showPending,
		IncludeScheduled: showScheduled,
		IncludeAbandoned: showAbandoned,
	}

	items := cleanup.ListCleanupItems(records, now, opts)

	if len(items) == 0 {
		fmt.Fprintln(deps.stdout, "no items pending cleanup")
		return nil
	}

	fmt.Fprintf(deps.stdout, "%-20s  %-20s  %-12s  %-25s  %s\n",
		"WORKTREE", "ENTITY", "STATUS", "CLEANUP AFTER", "BRANCH")
	for _, item := range items {
		cleanupAfterStr := ""
		if !item.CleanupAfter.IsZero() {
			cleanupAfterStr = item.CleanupAfter.Format(time.RFC3339)
		}
		fmt.Fprintf(deps.stdout, "%-20s  %-20s  %-12s  %-25s  %s\n",
			item.WorktreeID, item.EntityID, item.Status, cleanupAfterStr, item.Branch)
	}

	return nil
}

func runCleanupRun(args []string, deps dependencies) error {
	flags, err := parseFlags(args)
	if err != nil {
		return err
	}

	dryRun := flags["dry-run"] == "true"
	worktreeID := flags["worktree"]

	store := worktree.NewStore(core.StatePath())
	gitOps := worktree.NewGit(".")
	cfg := config.LoadOrDefault()

	now := time.Now().UTC()

	if worktreeID != "" {
		// Clean up a specific worktree
		record, gerr := store.Get(worktreeID)
		if gerr != nil {
			return fmt.Errorf("get worktree %s: %w", worktreeID, gerr)
		}

		if dryRun {
			fmt.Fprintf(deps.stdout, "Would clean up:\n")
			fmt.Fprintf(deps.stdout, "  Worktree: %s\n", record.ID)
			fmt.Fprintf(deps.stdout, "  Branch:   %s\n", record.Branch)
			fmt.Fprintf(deps.stdout, "  Path:     %s\n", record.Path)
			return nil
		}

		opts := cleanup.CleanupOptions{
			DeleteRemoteBranch: cfg.Cleanup.AutoDeleteRemoteBranch,
		}
		result := cleanup.ExecuteCleanup(store, gitOps, record, opts)
		if !result.Success {
			return fmt.Errorf("cleanup failed for %s: %v", worktreeID, result.Error)
		}

		fmt.Fprintf(deps.stdout, "Cleaned up %s\n", result.WorktreeID)
		fmt.Fprintf(deps.stdout, "  Branch: %s\n", result.Branch)
		fmt.Fprintf(deps.stdout, "  Path:   %s\n", result.Path)
		if result.RemoteBranchDeleted {
			fmt.Fprintf(deps.stdout, "  Remote branch deleted: true\n")
		}
		return nil
	}

	// Clean up all ready items
	records, err := store.List()
	if err != nil {
		return fmt.Errorf("list worktrees: %w", err)
	}

	readyItems := cleanup.ListCleanupItems(records, now, cleanup.ListOptions{
		IncludePending:   true,
		IncludeAbandoned: true,
	})

	if len(readyItems) == 0 {
		fmt.Fprintln(deps.stdout, "no items ready for cleanup")
		return nil
	}

	if dryRun {
		fmt.Fprintf(deps.stdout, "Would clean up %d item(s):\n", len(readyItems))
		for _, item := range readyItems {
			fmt.Fprintf(deps.stdout, "  %s  %s  %s\n", item.WorktreeID, item.EntityID, item.Branch)
		}
		return nil
	}

	opts := cleanup.CleanupOptions{
		DeleteRemoteBranch: cfg.Cleanup.AutoDeleteRemoteBranch,
	}

	cleaned := 0
	for _, item := range readyItems {
		record, gerr := store.Get(item.WorktreeID)
		if gerr != nil {
			fmt.Fprintf(deps.stdout, "  skip %s: %v\n", item.WorktreeID, gerr)
			continue
		}
		result := cleanup.ExecuteCleanup(store, gitOps, record, opts)
		if result.Success {
			fmt.Fprintf(deps.stdout, "  cleaned %s (%s)\n", result.WorktreeID, result.Branch)
			cleaned++
		} else {
			fmt.Fprintf(deps.stdout, "  failed %s: %v\n", result.WorktreeID, result.Error)
		}
	}

	fmt.Fprintf(deps.stdout, "\n%d of %d items cleaned\n", cleaned, len(readyItems))
	return nil
}

const cleanupUsageText = `kanbanzai cleanup <subcommand> [flags]

Manage post-merge cleanup of worktrees and branches.

Subcommands:
  list   List items pending cleanup
  run    Execute cleanup of ready items

Examples:
  kbz cleanup list
  kbz cleanup list --pending --scheduled
  kbz cleanup run
  kbz cleanup run --dry-run
  kbz cleanup run --worktree=WT-01JX...
`
