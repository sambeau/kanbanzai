package main

import (
	"fmt"
	"io"
	"strings"

	"github.com/sambeau/kanbanzai/internal/id"
	"github.com/sambeau/kanbanzai/internal/service"
)

const entityUsageText = `Usage: kbz entity <subcommand> [options]

Subcommands:
  create <type>              Create a new entity (plan, feature, task, bug, decision)
    --slug, --name, --summary, --parent, etc.

  get <type> --id <id>       Get an entity by ID

  list <type>                List all entities of a type
    (types: plans, features, tasks, bugs, decisions)

  transition --type <type> --id <id> --status <status>
                             Transition entity lifecycle status
`

func runEntity(args []string, deps dependencies) error {
	if len(args) == 0 {
		return fmt.Errorf("missing entity subcommand\n\n%s", entityUsageText)
	}

	switch args[0] {
	case "create":
		return runEntityCreate(args[1:], deps)
	case "get":
		return runEntityGet(args[1:], deps)
	case "list":
		return runEntityList(args[1:], deps)
	case "transition":
		return runEntityTransition(args[1:], deps)
	default:
		return fmt.Errorf("unknown entity subcommand %q\n\n%s", args[0], entityUsageText)
	}
}

// ─── create ──────────────────────────────────────────────────────────────────

func runEntityCreate(args []string, deps dependencies) error {
	if len(args) == 0 {
		return fmt.Errorf("missing create type\n\n%s", entityUsageText)
	}

	svc := deps.newEntityService("")

	switch args[0] {
	case "plan":
		values, err := parseFlags(args[1:])
		if err != nil {
			return err
		}
		result, err := svc.CreatePlan(service.CreatePlanInput{
			Prefix:    values["prefix"],
			Slug:      values["slug"],
			Name:      values["name"],
			Summary:   values["summary"],
			CreatedBy: values["created_by"],
		})
		if err != nil {
			return err
		}
		return printCreateResult(deps.stdout, result)
	case "feature":
		values, err := parseFlags(args[1:])
		if err != nil {
			return err
		}
		result, err := svc.CreateFeature(service.CreateFeatureInput{
			Slug:      values["slug"],
			Parent:    values["parent"],
			Name:      values["name"],
			Summary:   values["summary"],
			CreatedBy: values["created_by"],
		})
		if err != nil {
			return err
		}
		return printCreateResult(deps.stdout, result)
	case "task":
		values, err := parseFlags(args[1:])
		if err != nil {
			return err
		}
		result, err := svc.CreateTask(service.CreateTaskInput{
			ParentFeature: values["parent_feature"],
			Slug:          values["slug"],
			Name:          values["name"],
			Summary:       values["summary"],
		})
		if err != nil {
			return err
		}
		return printCreateResult(deps.stdout, result)
	case "bug":
		values, err := parseFlags(args[1:])
		if err != nil {
			return err
		}
		result, err := svc.CreateBug(service.CreateBugInput{
			Slug:       values["slug"],
			Name:       values["name"],
			ReportedBy: values["reported_by"],
			Observed:   values["observed"],
			Expected:   values["expected"],
			Severity:   values["severity"],
			Priority:   values["priority"],
			Type:       values["type"],
		})
		if err != nil {
			return err
		}
		return printCreateResult(deps.stdout, result)
	case "decision":
		values, err := parseFlags(args[1:])
		if err != nil {
			return err
		}
		result, err := svc.CreateDecision(service.CreateDecisionInput{
			Slug:      values["slug"],
			Name:      values["name"],
			Summary:   values["summary"],
			Rationale: values["rationale"],
			DecidedBy: values["decided_by"],
		})
		if err != nil {
			return err
		}
		return printCreateResult(deps.stdout, result)
	default:
		return fmt.Errorf("unknown entity type %q\n\n%s", args[0], entityUsageText)
	}
}

// ─── get ─────────────────────────────────────────────────────────────────────

func runEntityGet(args []string, deps dependencies) error {
	if len(args) == 0 {
		return fmt.Errorf("missing get type\n\n%s", entityUsageText)
	}

	svc := deps.newEntityService("")

	switch args[0] {
	case "plan":
		values, err := parseFlags(args[1:])
		if err != nil {
			return err
		}
		listResult, err := svc.GetPlan(values["id"])
		if err != nil {
			return err
		}
		return printGetResult(deps.stdout, service.GetResult{
			Type:  listResult.Type,
			ID:    listResult.ID,
			Slug:  listResult.Slug,
			Path:  listResult.Path,
			State: listResult.State,
		})
	case "feature", "task", "bug", "decision":
		values, err := parseFlags(args[1:])
		if err != nil {
			return err
		}
		result, err := svc.Get(args[0], values["id"], values["slug"])
		if err != nil {
			return err
		}
		return printGetResult(deps.stdout, result)
	default:
		return fmt.Errorf("unknown entity type %q\n\n%s", args[0], entityUsageText)
	}
}

// ─── list ────────────────────────────────────────────────────────────────────

func runEntityList(args []string, deps dependencies) error {
	if len(args) == 0 {
		return fmt.Errorf("missing list type\n\n%s", entityUsageText)
	}

	svc := deps.newEntityService("")

	switch args[0] {
	case "plans":
		results, err := svc.ListPlans(service.PlanFilters{})
		if err != nil {
			return fmt.Errorf("list plans: %w", err)
		}
		fmt.Fprintln(deps.stdout, "listed plans")
		for _, r := range results {
			status, _ := r.State["status"].(string)
			fmt.Fprintf(deps.stdout, "%s\t%s\t%s\t%s\n", r.ID, r.Slug, r.Path, status)
		}
		return nil
	case "features":
		return printListResults(deps.stdout, "feature", svc)
	case "tasks":
		return printListResults(deps.stdout, "task", svc)
	case "bugs":
		return printListResults(deps.stdout, "bug", svc)
	case "decisions":
		return printListResults(deps.stdout, "decision", svc)
	default:
		return fmt.Errorf("unknown entity list type %q\n\n%s", args[0], entityUsageText)
	}
}

// ─── transition ──────────────────────────────────────────────────────────────

func runEntityTransition(args []string, deps dependencies) error {
	values, err := parseFlags(args)
	if err != nil {
		return err
	}

	entityType := values["type"]
	if entityType == "" {
		return fmt.Errorf("--type is required\n\nUsage: kbz entity transition --type <type> --id <id> --status <status>")
	}
	if values["status"] == "" {
		return fmt.Errorf("--status is required\n\nUsage: kbz entity transition --type <type> --id <id> --status <status>")
	}

	svc := deps.newEntityService("")
	result, err := svc.UpdateStatus(service.UpdateStatusInput{
		Type:   entityType,
		ID:     values["id"],
		Slug:   values["slug"],
		Status: values["status"],
	})
	if err != nil {
		return err
	}

	return printStatusUpdateResult(deps.stdout, result)
}

// ─── Shared print helpers ────────────────────────────────────────────────────

func printCreateResult(w io.Writer, result service.CreateResult) error {
	_, err := fmt.Fprintf(
		w,
		"created %s\nid: %s\nslug: %s\npath: %s\n",
		result.Type,
		id.FormatFullDisplay(result.ID),
		result.Slug,
		result.Path,
	)
	return err
}

func printGetResult(w io.Writer, result service.GetResult) error {
	_, err := fmt.Fprintf(
		w,
		"type: %s\nid: %s\nslug: %s\npath: %s\nstatus: %v\n",
		result.Type,
		id.FormatFullDisplay(result.ID),
		result.Slug,
		result.Path,
		result.State["status"],
	)
	return err
}

func printListResults(w io.Writer, entityType string, svc entityService) error {
	results, err := svc.List(entityType)
	if err != nil {
		return err
	}

	if _, err := fmt.Fprintf(w, "listed %s\n", entityType); err != nil {
		return err
	}

	for _, result := range results {
		if _, err := fmt.Fprintf(
			w,
			"%s\t%s\t%s\t%v\n",
			id.FormatFullDisplay(result.ID),
			result.Slug,
			result.Path,
			result.State["status"],
		); err != nil {
			return err
		}
	}

	return nil
}

func printStatusUpdateResult(w io.Writer, result service.GetResult) error {
	_, err := fmt.Fprintf(
		w,
		"updated %s\nid: %s\nslug: %s\npath: %s\nstatus: %v\n",
		result.Type,
		id.FormatFullDisplay(result.ID),
		result.Slug,
		result.Path,
		result.State["status"],
	)
	return err
}

// parseFlags parses --key value and --key=value pairs from args into a string map.
func parseFlags(args []string) (map[string]string, error) {
	values := map[string]string{}

	for i := 0; i < len(args); i++ {
		arg := args[i]
		if !strings.HasPrefix(arg, "--") {
			return nil, fmt.Errorf("unexpected argument %q", arg)
		}

		name := strings.TrimPrefix(arg, "--")
		if name == "" {
			return nil, fmt.Errorf("empty flag name")
		}

		if strings.Contains(name, "=") {
			parts := strings.SplitN(name, "=", 2)
			values[parts[0]] = parts[1]
			continue
		}

		if i+1 >= len(args) {
			return nil, fmt.Errorf("missing value for --%s", name)
		}

		values[name] = args[i+1]
		i++
	}

	return values, nil
}
