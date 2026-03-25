package main

import (
	"fmt"
	"time"

	"kanbanzai/internal/core"
	"kanbanzai/internal/git"
	"kanbanzai/internal/knowledge"
	"kanbanzai/internal/service"
	"kanbanzai/internal/storage"
)

// runKnowledgeLifecycle handles the Phase 3 knowledge lifecycle subcommands.
// It is called from runKnowledge for subcommands not handled by existing Phase 2b code.
func runKnowledgeLifecycle(subcommand string, args []string, deps dependencies) error {
	switch subcommand {
	case "check":
		return runKnowledgeCheck(args, deps)
	case "confirm":
		return runKnowledgeConfirm(args, deps)
	case "prune":
		return runKnowledgePrune(args, deps)
	case "compact":
		return runKnowledgeCompact(args, deps)
	case "resolve":
		return runKnowledgeResolve(args, deps)
	default:
		return fmt.Errorf("unknown knowledge subcommand %q\n\n%s", subcommand, knowledgeLifecycleUsageText)
	}
}

func runKnowledgeCheck(args []string, deps dependencies) error {
	flags, err := parseFlags(args)
	if err != nil {
		return err
	}

	entryID := flags["entry"]
	scope := flags["scope"]
	repoPath := "."

	knowledgeSvc := service.NewKnowledgeService(core.StatePath())

	var records []storage.KnowledgeRecord

	if entryID != "" {
		record, gerr := knowledgeSvc.Get(entryID)
		if gerr != nil {
			return fmt.Errorf("get knowledge entry: %w", gerr)
		}
		records = []storage.KnowledgeRecord{record}
	} else {
		all, gerr := knowledgeSvc.LoadAllRaw()
		if gerr != nil {
			return fmt.Errorf("load knowledge entries: %w", gerr)
		}
		records = all
	}

	staleCount := 0
	for _, rec := range records {
		// Filter by scope if specified
		if scope != "" {
			s, _ := rec.Fields["scope"].(string)
			if s != scope {
				continue
			}
		}

		// Only check entries with git_anchors
		anchors := git.ExtractAnchors(rec.Fields)
		if len(anchors) == 0 {
			continue
		}

		staleness, err := git.CheckEntryStaleness(repoPath, rec.Fields)
		if err != nil {
			fmt.Fprintf(deps.stdout, "  warning: could not check %s: %v\n", rec.ID, err)
			continue
		}

		if staleness.IsStale {
			staleCount++
			topic, _ := rec.Fields["topic"].(string)
			fmt.Fprintf(deps.stdout, "STALE  %s  topic=%s\n", rec.ID, topic)
			fmt.Fprintf(deps.stdout, "  reason: %s\n", staleness.StaleReason)
			for _, sf := range staleness.StaleFiles {
				fmt.Fprintf(deps.stdout, "  file: %s (modified %s, commit %s)\n",
					sf.Path, sf.ModifiedAt.Format(time.RFC3339), sf.Commit)
			}
			fmt.Fprintf(deps.stdout, "  last confirmed: %s\n", staleness.LastConfirmed.Format(time.RFC3339))
			fmt.Fprintln(deps.stdout)
		}
	}

	if staleCount == 0 {
		fmt.Fprintln(deps.stdout, "no stale knowledge entries found")
	} else {
		fmt.Fprintf(deps.stdout, "%d stale entry(ies) found\n", staleCount)
	}
	return nil
}

func runKnowledgeConfirm(args []string, deps dependencies) error {
	if len(args) == 0 {
		return fmt.Errorf("missing entry ID\n\n%s", knowledgeLifecycleUsageText)
	}

	entryID := args[0]
	knowledgeSvc := service.NewKnowledgeService(core.StatePath())

	record, err := knowledgeSvc.Get(entryID)
	if err != nil {
		return fmt.Errorf("get knowledge entry: %w", err)
	}

	now := time.Now().UTC()
	git.SetLastConfirmed(record.Fields, now)

	// Update status if it was stale
	if status, _ := record.Fields["status"].(string); status == "stale" {
		record.Fields["status"] = "confirmed"
	}

	ks := storage.NewKnowledgeStore(core.StatePath())
	if _, err := ks.Write(record); err != nil {
		return fmt.Errorf("write knowledge entry: %w", err)
	}

	fmt.Fprintf(deps.stdout, "Confirmed %s\n", entryID)
	fmt.Fprintf(deps.stdout, "  last_confirmed: %s\n", now.Format(time.RFC3339))
	return nil
}

func runKnowledgePrune(args []string, deps dependencies) error {
	flags, err := parseFlags(args)
	if err != nil {
		return err
	}

	dryRun := flags["dry-run"] == "true"
	tierFilter := 0
	if flags["tier"] != "" {
		_, _ = fmt.Sscanf(flags["tier"], "%d", &tierFilter)
	}

	knowledgeSvc := service.NewKnowledgeService(core.StatePath())
	ks := storage.NewKnowledgeStore(core.StatePath())

	records, err := knowledgeSvc.LoadAllRaw()
	if err != nil {
		return fmt.Errorf("load knowledge entries: %w", err)
	}

	now := time.Now().UTC()
	ttlConfig := knowledge.DefaultTTLConfig()

	var entries []map[string]any
	for _, rec := range records {
		entries = append(entries, rec.Fields)
	}

	opts := knowledge.PruneOptions{
		DryRun: dryRun,
		Tier:   tierFilter,
	}

	results := knowledge.PruneExpiredEntries(entries, now, ttlConfig, opts)

	if len(results) == 0 {
		fmt.Fprintln(deps.stdout, "no entries eligible for pruning")
		return nil
	}

	if dryRun {
		fmt.Fprintf(deps.stdout, "Would prune %d entry(ies):\n", len(results))
	} else {
		fmt.Fprintf(deps.stdout, "Pruning %d entry(ies):\n", len(results))
	}

	for _, r := range results {
		fmt.Fprintf(deps.stdout, "  %s  tier=%d  topic=%s\n", r.EntryID, r.Tier, r.Topic)
		fmt.Fprintf(deps.stdout, "    reason: %s\n", r.Reason)

		if !dryRun {
			// Transition entry to retired
			rec, gerr := knowledgeSvc.Get(r.EntryID)
			if gerr != nil {
				fmt.Fprintf(deps.stdout, "    warning: could not load entry for retirement: %v\n", gerr)
				continue
			}
			rec.Fields["status"] = "retired"
			if _, werr := ks.Write(rec); werr != nil {
				fmt.Fprintf(deps.stdout, "    warning: could not retire entry: %v\n", werr)
			}
		}
	}
	return nil
}

func runKnowledgeCompact(args []string, deps dependencies) error {
	flags, err := parseFlags(args)
	if err != nil {
		return err
	}

	dryRun := flags["dry-run"] == "true"
	scope := flags["scope"]

	knowledgeSvc := service.NewKnowledgeService(core.StatePath())
	ks := storage.NewKnowledgeStore(core.StatePath())

	records, err := knowledgeSvc.LoadAllRaw()
	if err != nil {
		return fmt.Errorf("load knowledge entries: %w", err)
	}

	var entries []map[string]any
	for _, rec := range records {
		// Skip retired entries
		status, _ := rec.Fields["status"].(string)
		if status == "retired" {
			continue
		}
		entries = append(entries, rec.Fields)
	}

	opts := knowledge.CompactionOptions{
		DryRun: dryRun,
		Scope:  scope,
	}

	var result knowledge.CompactionResult
	var updatedEntries []map[string]any
	if scope != "" {
		result, updatedEntries = knowledge.CompactEntriesInScope(entries, scope, opts)
	} else {
		result, updatedEntries = knowledge.CompactEntries(entries, opts)
	}

	fmt.Fprintf(deps.stdout, "Compaction results:\n")
	fmt.Fprintf(deps.stdout, "  Duplicates merged:      %d\n", result.DuplicatesMerged)
	fmt.Fprintf(deps.stdout, "  Near-duplicates merged:  %d\n", result.NearDuplicatesMerged)
	fmt.Fprintf(deps.stdout, "  Conflicts flagged:       %d\n", result.ConflictsFlagged)

	for _, d := range result.Details {
		switch d.Action {
		case knowledge.CompactionActionMerged:
			fmt.Fprintf(deps.stdout, "\n  merged: kept %s, discarded %s\n", d.Kept, d.Discarded)
			fmt.Fprintf(deps.stdout, "    reason: %s\n", d.Reason)
		case knowledge.CompactionActionDisputed:
			fmt.Fprintf(deps.stdout, "\n  disputed: %v\n", d.Entries)
			fmt.Fprintf(deps.stdout, "    reason: %s\n", d.Reason)
		}
	}

	if !dryRun && len(updatedEntries) > 0 {
		for _, entry := range updatedEntries {
			id, _ := entry["id"].(string)
			if id == "" {
				continue
			}
			rec := storage.KnowledgeRecord{
				ID:     id,
				Fields: entry,
			}
			if _, werr := ks.Write(rec); werr != nil {
				fmt.Fprintf(deps.stdout, "  warning: could not write %s: %v\n", id, werr)
			}
		}
	}

	return nil
}

func runKnowledgeResolve(args []string, deps dependencies) error {
	flags, err := parseFlags(args)
	if err != nil {
		return err
	}

	keepID := flags["keep"]
	retireID := flags["retire"]
	mergeContent := flags["merge"] == "true"

	if keepID == "" || retireID == "" {
		return fmt.Errorf("both --keep and --retire are required\n\n%s", knowledgeLifecycleUsageText)
	}

	knowledgeSvc := service.NewKnowledgeService(core.StatePath())
	ks := storage.NewKnowledgeStore(core.StatePath())

	keptRec, err := knowledgeSvc.Get(keepID)
	if err != nil {
		return fmt.Errorf("get kept entry: %w", err)
	}

	retiredRec, err := knowledgeSvc.Get(retireID)
	if err != nil {
		return fmt.Errorf("get retired entry: %w", err)
	}

	if mergeContent {
		// Merge content from retired into kept
		retiredContent, _ := retiredRec.Fields["content"].(string)
		keptContent, _ := keptRec.Fields["content"].(string)
		if retiredContent != "" && retiredContent != keptContent {
			keptRec.Fields["content"] = keptContent + "\n\n[Merged from " + retireID + "]: " + retiredContent
		}

		// Merge git_anchors
		keptAnchors := knowledge.GetGitAnchors(keptRec.Fields)
		retiredAnchors := knowledge.GetGitAnchors(retiredRec.Fields)
		seen := make(map[string]bool)
		for _, a := range keptAnchors {
			seen[a] = true
		}
		for _, a := range retiredAnchors {
			if !seen[a] {
				keptAnchors = append(keptAnchors, a)
			}
		}
		if len(keptAnchors) > 0 {
			knowledge.SetGitAnchors(keptRec.Fields, keptAnchors)
		}

		// Transfer usage counts
		keptUseCount := knowledge.GetUseCount(keptRec.Fields)
		retiredUseCount := knowledge.GetUseCount(retiredRec.Fields)
		keptRec.Fields["use_count"] = keptUseCount + retiredUseCount
	}

	// Confirm the kept entry
	keptRec.Fields["status"] = "confirmed"
	knowledge.SetMergedFrom(keptRec.Fields, retireID)

	// Retire the other entry
	retiredRec.Fields["status"] = "retired"

	if _, werr := ks.Write(keptRec); werr != nil {
		return fmt.Errorf("write kept entry: %w", werr)
	}
	if _, werr := ks.Write(retiredRec); werr != nil {
		return fmt.Errorf("write retired entry: %w", werr)
	}

	fmt.Fprintf(deps.stdout, "Resolved conflict:\n")
	fmt.Fprintf(deps.stdout, "  kept:    %s\n", keepID)
	fmt.Fprintf(deps.stdout, "  retired: %s\n", retireID)
	if mergeContent {
		fmt.Fprintf(deps.stdout, "  content merged: true\n")
	}
	return nil
}

const knowledgeLifecycleUsageText = `kanbanzai knowledge <subcommand> [flags]

Knowledge lifecycle commands (Phase 3):

Subcommands:
  list      List knowledge entries (Phase 2b)
  get       Get a knowledge entry by ID (Phase 2b)
  check     Check staleness of anchored knowledge entries
  confirm   Confirm a knowledge entry is still accurate
  prune     Prune expired knowledge entries by TTL
  compact   Run post-merge compaction on knowledge entries
  resolve   Resolve a disputed knowledge entry conflict

Examples:
  kbz knowledge check
  kbz knowledge check --entry=KE-01JX... --scope=backend
  kbz knowledge confirm KE-01JX...
  kbz knowledge prune --dry-run
  kbz knowledge prune --tier=3
  kbz knowledge compact --dry-run
  kbz knowledge compact --scope=backend
  kbz knowledge resolve --keep=KE-01JX... --retire=KE-01JY...
  kbz knowledge resolve --keep=KE-01JX... --retire=KE-01JY... --merge
`
