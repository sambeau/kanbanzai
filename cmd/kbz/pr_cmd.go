package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/sambeau/kanbanzai/internal/config"
	"github.com/sambeau/kanbanzai/internal/core"
	"github.com/sambeau/kanbanzai/internal/github"
	"github.com/sambeau/kanbanzai/internal/service"
	"github.com/sambeau/kanbanzai/internal/worktree"
)

func runPR(args []string, deps dependencies) error {
	if len(args) == 0 || wantsHelp(args) {
		fmt.Fprint(deps.stdout, prUsageText)
		return nil
	}

	switch args[0] {
	case "create":
		return runPRCreate(args[1:], deps)
	case "update":
		return runPRUpdate(args[1:], deps)
	case "status":
		return runPRStatus(args[1:], deps)
	default:
		return fmt.Errorf("unknown pr subcommand %q\n\n%s", args[0], prUsageText)
	}
}

func runPRCreate(args []string, deps dependencies) error {
	if len(args) == 0 {
		return fmt.Errorf("missing entity ID\n\n%s", prUsageText)
	}

	entityID := args[0]
	flags, err := parseFlags(args[1:])
	if err != nil {
		return err
	}

	draft := flags["draft"] == "true"

	localConfig, err := config.LoadLocalConfig()
	if err != nil || localConfig.GetGitHubToken() == "" {
		return fmt.Errorf("GitHub token not configured. Set github.token in .kbz/local.yaml")
	}

	store := worktree.NewStore(core.StatePath())
	entitySvc := service.NewEntityService(core.StatePath())

	record, err := store.GetByEntityID(entityID)
	if err != nil {
		return fmt.Errorf("get worktree: %w (does entity %s have a worktree?)", err, entityID)
	}

	entityType := prEntityType(entityID)
	if entityType == "" {
		return fmt.Errorf("invalid entity type: ID must start with FEAT- or BUG-")
	}

	entity, err := entitySvc.Get(entityType, entityID, "")
	if err != nil {
		return fmt.Errorf("get entity: %w", err)
	}

	client := github.NewClient(localConfig.GetGitHubToken())

	repo, err := github.DetectRepo(".", localConfig)
	if err != nil {
		return fmt.Errorf("detect repository: %w", err)
	}

	title := prStringFromEntityState(entity.State, "title")
	if title == "" {
		title = entityID
	}

	description := buildPRDescription(&entity, entityID, entitySvc, record.Branch)

	pr, err := client.CreatePR(context.Background(), repo, record.Branch, "main", title, description, draft)
	if err != nil {
		return fmt.Errorf("create PR: %w", err)
	}

	fmt.Fprintf(deps.stdout, "Created PR #%d\n", pr.Number)
	fmt.Fprintf(deps.stdout, "  URL:   %s\n", pr.URL)
	fmt.Fprintf(deps.stdout, "  Title: %s\n", pr.Title)
	fmt.Fprintf(deps.stdout, "  State: %s\n", pr.State)
	if pr.Draft {
		fmt.Fprintf(deps.stdout, "  Draft: true\n")
	}
	return nil
}

func runPRUpdate(args []string, deps dependencies) error {
	if len(args) == 0 {
		return fmt.Errorf("missing entity ID\n\n%s", prUsageText)
	}

	entityID := args[0]

	localConfig, err := config.LoadLocalConfig()
	if err != nil || localConfig.GetGitHubToken() == "" {
		return fmt.Errorf("GitHub token not configured. Set github.token in .kbz/local.yaml")
	}

	store := worktree.NewStore(core.StatePath())
	entitySvc := service.NewEntityService(core.StatePath())

	record, err := store.GetByEntityID(entityID)
	if err != nil {
		return fmt.Errorf("get worktree: %w", err)
	}

	entityType := prEntityType(entityID)
	if entityType == "" {
		return fmt.Errorf("invalid entity type: ID must start with FEAT- or BUG-")
	}

	entity, err := entitySvc.Get(entityType, entityID, "")
	if err != nil {
		return fmt.Errorf("get entity: %w", err)
	}

	client := github.NewClient(localConfig.GetGitHubToken())

	repo, err := github.DetectRepo(".", localConfig)
	if err != nil {
		return fmt.Errorf("detect repository: %w", err)
	}

	// Find existing PR by branch
	existingPR, err := client.GetPRByBranch(context.Background(), repo, record.Branch)
	if err != nil {
		return fmt.Errorf("find PR for branch %s: %w", record.Branch, err)
	}

	title := prStringFromEntityState(entity.State, "title")
	if title == "" {
		title = entityID
	}

	description := buildPRDescription(&entity, entityID, entitySvc, record.Branch)

	updatedPR, err := client.UpdatePR(context.Background(), repo, existingPR.Number, title, description)
	if err != nil {
		return fmt.Errorf("update PR: %w", err)
	}

	fmt.Fprintf(deps.stdout, "Updated PR #%d\n", updatedPR.Number)
	fmt.Fprintf(deps.stdout, "  URL: %s\n", updatedPR.URL)
	return nil
}

func runPRStatus(args []string, deps dependencies) error {
	if len(args) == 0 {
		return fmt.Errorf("missing entity ID\n\n%s", prUsageText)
	}

	entityID := args[0]

	localConfig, err := config.LoadLocalConfig()
	if err != nil || localConfig.GetGitHubToken() == "" {
		return fmt.Errorf("GitHub token not configured. Set github.token in .kbz/local.yaml")
	}

	store := worktree.NewStore(core.StatePath())

	record, err := store.GetByEntityID(entityID)
	if err != nil {
		return fmt.Errorf("get worktree: %w", err)
	}

	client := github.NewClient(localConfig.GetGitHubToken())

	repo, err := github.DetectRepo(".", localConfig)
	if err != nil {
		return fmt.Errorf("detect repository: %w", err)
	}

	pr, err := client.GetPRByBranch(context.Background(), repo, record.Branch)
	if err != nil {
		return fmt.Errorf("get PR for branch %s: %w", record.Branch, err)
	}

	fmt.Fprintf(deps.stdout, "PR #%d\n", pr.Number)
	fmt.Fprintf(deps.stdout, "  URL:           %s\n", pr.URL)
	fmt.Fprintf(deps.stdout, "  State:         %s\n", pr.State)
	fmt.Fprintf(deps.stdout, "  Draft:         %t\n", pr.Draft)
	fmt.Fprintf(deps.stdout, "  CI Status:     %s\n", valueOrNone(pr.CIStatus))
	fmt.Fprintf(deps.stdout, "  Review Status: %s\n", valueOrNone(pr.ReviewStatus))
	fmt.Fprintf(deps.stdout, "  Has Conflicts: %t\n", pr.HasConflicts)
	fmt.Fprintf(deps.stdout, "  Mergeable:     %t\n", pr.Mergeable)

	if len(pr.Reviews) > 0 {
		fmt.Fprintln(deps.stdout, "\n  Reviews:")
		for _, r := range pr.Reviews {
			fmt.Fprintf(deps.stdout, "    %s: %s\n", r.User, r.State)
		}
	}

	return nil
}

// buildPRDescription generates a PR description from entity state.
func buildPRDescription(entity *service.GetResult, entityID string, entitySvc *service.EntityService, branch string) string {
	var b strings.Builder

	title := prStringFromEntityState(entity.State, "title")
	b.WriteString(fmt.Sprintf("## %s\n\n", title))

	desc := prStringFromEntityState(entity.State, "description")
	if desc != "" {
		b.WriteString(desc)
		b.WriteString("\n\n")
	}

	// Tasks section
	tasks := gatherTasksForPR(entitySvc, entityID)
	if len(tasks) > 0 {
		b.WriteString("### Tasks\n\n")
		for _, t := range tasks {
			taskID, _ := t["id"].(string)
			taskTitle, _ := t["title"].(string)
			status, _ := t["status"].(string)
			checked := " "
			if status == "done" || status == "complete" || status == "completed" {
				checked = "x"
			}
			b.WriteString(fmt.Sprintf("- [%s] %s: %s (%s)\n", checked, taskID, taskTitle, status))
		}
		b.WriteString("\n")
	}

	// Verification section
	verification := prStringFromEntityState(entity.State, "verification")
	if verification != "" {
		b.WriteString("### Verification\n\n")
		b.WriteString(verification)
		b.WriteString("\n\n")
		verificationStatus := prStringFromEntityState(entity.State, "verification_status")
		if verificationStatus != "" {
			b.WriteString(fmt.Sprintf("**Status:** %s\n\n", verificationStatus))
		}
	}

	// Workflow section
	b.WriteString("### Workflow\n\n")
	b.WriteString(fmt.Sprintf("- **Entity:** %s\n", entityID))
	created := prStringFromEntityState(entity.State, "created")
	if created != "" {
		b.WriteString(fmt.Sprintf("- **Created:** %s\n", created))
	}
	b.WriteString(fmt.Sprintf("- **Branch:** %s\n", branch))
	b.WriteString("\n---\n*This description is managed by Kanbanzai. Manual edits may be overwritten.*\n")

	return b.String()
}

// gatherTasksForPR returns task state maps for an entity's child tasks.
func gatherTasksForPR(entitySvc *service.EntityService, entityID string) []map[string]any {
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

func prEntityType(id string) string {
	upper := strings.ToUpper(id)
	if strings.HasPrefix(upper, "FEAT-") {
		return "feature"
	}
	if strings.HasPrefix(upper, "BUG-") {
		return "bug"
	}
	return ""
}

func prStringFromEntityState(state map[string]any, key string) string {
	v, ok := state[key]
	if !ok {
		return ""
	}
	s, ok := v.(string)
	if !ok {
		return fmt.Sprint(v)
	}
	return s
}

func valueOrNone(s string) string {
	if s == "" {
		return "none"
	}
	return s
}

const prUsageText = `kanbanzai pr <subcommand> [flags]

Manage GitHub pull requests for feature and bug entities.

Subcommands:
  create   Create a new PR for an entity's worktree branch
  update   Update an existing PR's description and labels
  status   Show PR status (CI, reviews, conflicts)

Examples:
  kbz pr create FEAT-01JX...
  kbz pr create FEAT-01JX... --draft
  kbz pr update FEAT-01JX...
  kbz pr status FEAT-01JX...

Notes:
  Requires github.token in .kbz/local.yaml
`
