package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	kbzmcp "kanbanzai/internal/mcp"
	"kanbanzai/internal/service"
)

type entityService interface {
	CreateEpic(service.CreateEpicInput) (service.CreateResult, error)
	CreateFeature(service.CreateFeatureInput) (service.CreateResult, error)
	CreateTask(service.CreateTaskInput) (service.CreateResult, error)
	CreateBug(service.CreateBugInput) (service.CreateResult, error)
	CreateDecision(service.CreateDecisionInput) (service.CreateResult, error)
	Get(entityType, entityID, slug string) (service.GetResult, error)
	List(entityType string) ([]service.ListResult, error)
	UpdateStatus(service.UpdateStatusInput) (service.GetResult, error)
}

type dependencies struct {
	stdout           io.Writer
	newEntityService func(root string) entityService
}

func defaultDependencies() dependencies {
	return dependencies{
		stdout: os.Stdout,
		newEntityService: func(root string) entityService {
			return service.NewEntityService(root)
		},
	}
}

func main() {
	if err := run(os.Args[1:], defaultDependencies()); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string, deps dependencies) error {
	if deps.stdout == nil {
		deps.stdout = os.Stdout
	}
	if deps.newEntityService == nil {
		deps.newEntityService = func(root string) entityService {
			return service.NewEntityService(root)
		}
	}

	if len(args) == 0 {
		printUsage(deps.stdout)
		return nil
	}

	switch args[0] {
	case "help", "-h", "--help":
		printUsage(deps.stdout)
		return nil
	case "version", "--version":
		fmt.Fprintln(deps.stdout, "kanbanzai phase-1-dev")
		return nil
	case "serve":
		return kbzmcp.Serve()
	case "create":
		return runCreate(args[1:], deps)
	case "get":
		return runGet(args[1:], deps)
	case "list":
		return runList(args[1:], deps)
	case "update":
		return runUpdate(args[1:], deps)
	default:
		return fmt.Errorf("unknown command %q\n\n%s", args[0], usageText)
	}
}

func runCreate(args []string, deps dependencies) error {
	if len(args) == 0 {
		return fmt.Errorf("missing create target\n\n%s", createUsageText)
	}

	svc := deps.newEntityService("")

	switch args[0] {
	case "epic":
		values, err := parseFlags(args[1:])
		if err != nil {
			return err
		}
		result, err := svc.CreateEpic(service.CreateEpicInput{
			Slug:      values["slug"],
			Title:     values["title"],
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
			Epic:      values["epic"],
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
			Feature: values["feature"],
			Slug:    values["slug"],
			Summary: values["summary"],
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
			Title:      values["title"],
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
			Summary:   values["summary"],
			Rationale: values["rationale"],
			DecidedBy: values["decided_by"],
		})
		if err != nil {
			return err
		}
		return printCreateResult(deps.stdout, result)
	default:
		return fmt.Errorf("unknown create target %q\n\n%s", args[0], createUsageText)
	}
}

func runGet(args []string, deps dependencies) error {
	if len(args) == 0 {
		return fmt.Errorf("missing get target\n\n%s", getUsageText)
	}

	svc := deps.newEntityService("")

	switch args[0] {
	case "epic", "feature", "task", "bug", "decision":
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
		return fmt.Errorf("unknown get target %q\n\n%s", args[0], getUsageText)
	}
}

func runList(args []string, deps dependencies) error {
	if len(args) == 0 {
		return fmt.Errorf("missing list target\n\n%s", listUsageText)
	}

	svc := deps.newEntityService("")

	switch args[0] {
	case "epics":
		return printListResults(deps.stdout, "epic", svc)
	case "features":
		return printListResults(deps.stdout, "feature", svc)
	case "tasks":
		return printListResults(deps.stdout, "task", svc)
	case "bugs":
		return printListResults(deps.stdout, "bug", svc)
	case "decisions":
		return printListResults(deps.stdout, "decision", svc)
	default:
		return fmt.Errorf("unknown list target %q\n\n%s", args[0], listUsageText)
	}
}

func runUpdate(args []string, deps dependencies) error {
	if len(args) == 0 {
		return fmt.Errorf("missing update target\n\n%s", updateUsageText)
	}

	if args[0] != "status" {
		return fmt.Errorf("unknown update target %q\n\n%s", args[0], updateUsageText)
	}

	values, err := parseFlags(args[1:])
	if err != nil {
		return err
	}

	entityType := values["type"]
	if entityType == "" {
		entityType = values["entity"]
	}
	if entityType == "" {
		return fmt.Errorf("type is required\n\n%s", updateUsageText)
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

func printCreateResult(w io.Writer, result service.CreateResult) error {
	_, err := fmt.Fprintf(
		w,
		"created %s\nid: %s\nslug: %s\npath: %s\n",
		result.Type,
		result.ID,
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
		result.ID,
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
			result.ID,
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
		result.ID,
		result.Slug,
		result.Path,
		result.State["status"],
	)
	return err
}

func printUsage(w io.Writer) {
	_, _ = fmt.Fprint(w, usageText)
}

const usageText = `kanbanzai

Phase 1 workflow kernel CLI.

Usage:
  kanbanzai <command>

Commands:
  help       Show this help text
  version    Show the current development version
  serve      Start the MCP server (stdio transport)
  create     Create a Phase 1 entity
  get        Get a Phase 1 entity
  list       List Phase 1 entities
  update     Update Phase 1 entity state

Notes:
  - Phase 1 is MCP-first; the CLI is a secondary, strict interface.
  - This entrypoint is intentionally minimal while the kernel is being built.
`

const createUsageText = `kanbanzai create <entity> [flags]

Entities:
  epic
    --slug
    --title
    --summary
    --created_by

  feature
    --slug
    --epic
    --summary
    --created_by

  task
    --slug
    --feature
    --summary

  bug
    --slug
    --title
    --reported_by
    --observed
    --expected
    [--severity]
    [--priority]
    [--type]

  decision
    --slug
    --summary
    --rationale
    --decided_by
`

const getUsageText = `kanbanzai get <entity> [flags]

Entities:
  epic
  feature
  task
  bug
  decision

Flags:
  --id
  --slug
`

const listUsageText = `kanbanzai list <collection>

Collections:
  epics
  features
  tasks
  bugs
  decisions
`

const updateUsageText = `kanbanzai update status [flags]

Flags:
  --type
  --id
  --slug
  --status
`
