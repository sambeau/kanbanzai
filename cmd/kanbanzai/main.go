package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	"path/filepath"

	"kanbanzai/internal/cache"
	"kanbanzai/internal/core"
	"kanbanzai/internal/document"
	kbzmcp "kanbanzai/internal/mcp"
	"kanbanzai/internal/service"
	"kanbanzai/internal/validate"
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
	UpdateEntity(service.UpdateEntityInput) (service.GetResult, error)
	ValidateCandidate(entityType string, fields map[string]any) []validate.ValidationError
	HealthCheck() (*validate.HealthReport, error)
	RebuildCache() (int, error)
	SetCache(c *cache.Cache)
}

type docService interface {
	ScaffoldDocument(docType document.DocType, title string) (string, error)
	Submit(input document.SubmitInput) (document.DocumentResult, error)
	Approve(input document.ApproveInput) (document.DocumentResult, error)
	Retrieve(docType document.DocType, id string) (document.Document, error)
	Validate(doc document.Document) []document.ValidationError
	ListByType(docType document.DocType) ([]document.DocumentResult, error)
	ListAll() ([]document.DocumentResult, error)
}

type dependencies struct {
	stdout           io.Writer
	stdin            io.Reader
	newEntityService func(root string) entityService
	newDocService    func(root string) docService
}

func defaultDependencies() dependencies {
	return dependencies{
		stdout: os.Stdout,
		stdin:  os.Stdin,
		newEntityService: func(root string) entityService {
			return service.NewEntityService(root)
		},
		newDocService: func(root string) docService {
			return document.NewDocService(root)
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
	if deps.stdin == nil {
		deps.stdin = os.Stdin
	}
	if deps.newEntityService == nil {
		deps.newEntityService = func(root string) entityService {
			return service.NewEntityService(root)
		}
	}
	if deps.newDocService == nil {
		deps.newDocService = func(root string) docService {
			return document.NewDocService(root)
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
	case "doc":
		return runDoc(args[1:], deps)
	case "health":
		return runHealth(deps)
	case "validate":
		return runValidate(args[1:], deps)
	case "cache":
		return runCache(args[1:], deps)
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
			EpicSlug:  values["epic_slug"],
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
			ParentFeature: values["parent_feature"],
			Slug:          values["slug"],
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

	switch args[0] {
	case "status":
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

	case "fields":
		values, err := parseFlags(args[1:])
		if err != nil {
			return err
		}

		entityType := values["type"]
		if entityType == "" {
			return fmt.Errorf("type is required\n\n%s", updateUsageText)
		}
		id := values["id"]
		if id == "" {
			return fmt.Errorf("id is required\n\n%s", updateUsageText)
		}
		slug := values["slug"]

		fields := make(map[string]string, len(values))
		for k, v := range values {
			if k == "type" || k == "id" || k == "slug" {
				continue
			}
			fields[k] = v
		}

		svc := deps.newEntityService("")
		result, err := svc.UpdateEntity(service.UpdateEntityInput{
			Type:   entityType,
			ID:     id,
			Slug:   slug,
			Fields: fields,
		})
		if err != nil {
			return err
		}

		return printGetResult(deps.stdout, result)

	default:
		return fmt.Errorf("unknown update target %q\n\n%s", args[0], updateUsageText)
	}
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

func runDoc(args []string, deps dependencies) error {
	if len(args) == 0 {
		return fmt.Errorf("missing doc subcommand\n\n%s", docUsageText)
	}

	svc := deps.newDocService("")

	switch args[0] {
	case "scaffold":
		values, err := parseFlags(args[1:])
		if err != nil {
			return err
		}
		docType := values["type"]
		if docType == "" {
			return fmt.Errorf("type is required\n\n%s", docUsageText)
		}
		title := values["title"]
		if title == "" {
			title = "Untitled"
		}
		content, err := svc.ScaffoldDocument(document.DocType(docType), title)
		if err != nil {
			return err
		}
		_, err = fmt.Fprint(deps.stdout, content)
		return err

	case "submit":
		values, err := parseFlags(args[1:])
		if err != nil {
			return err
		}
		docType := values["type"]
		if docType == "" {
			return fmt.Errorf("type is required\n\n%s", docUsageText)
		}
		title := values["title"]
		if title == "" {
			return fmt.Errorf("title is required\n\n%s", docUsageText)
		}
		createdBy := values["created_by"]
		if createdBy == "" {
			createdBy = values["created-by"]
		}
		if createdBy == "" {
			return fmt.Errorf("created_by is required\n\n%s", docUsageText)
		}
		body := values["body"]
		if body == "" {
			data, readErr := io.ReadAll(deps.stdin)
			if readErr != nil {
				return fmt.Errorf("read document body: %w", readErr)
			}
			body = string(data)
		}
		result, err := svc.Submit(document.SubmitInput{
			Type:      document.DocType(docType),
			Title:     title,
			Feature:   values["feature"],
			Body:      body,
			CreatedBy: createdBy,
		})
		if err != nil {
			return err
		}
		return printDocumentResult(deps.stdout, "submitted", result)

	case "approve":
		values, err := parseFlags(args[1:])
		if err != nil {
			return err
		}
		docType := values["type"]
		if docType == "" {
			return fmt.Errorf("type is required\n\n%s", docUsageText)
		}
		id := values["id"]
		if id == "" {
			return fmt.Errorf("id is required\n\n%s", docUsageText)
		}
		approvedBy := values["approved_by"]
		if approvedBy == "" {
			approvedBy = values["approved-by"]
		}
		if approvedBy == "" {
			return fmt.Errorf("approved_by is required\n\n%s", docUsageText)
		}
		result, err := svc.Approve(document.ApproveInput{
			Type:       document.DocType(docType),
			ID:         id,
			ApprovedBy: approvedBy,
		})
		if err != nil {
			return err
		}
		return printDocumentResult(deps.stdout, "approved", result)

	case "retrieve":
		values, err := parseFlags(args[1:])
		if err != nil {
			return err
		}
		docType := values["type"]
		if docType == "" {
			return fmt.Errorf("type is required\n\n%s", docUsageText)
		}
		id := values["id"]
		if id == "" {
			return fmt.Errorf("id is required\n\n%s", docUsageText)
		}
		doc, err := svc.Retrieve(document.DocType(docType), id)
		if err != nil {
			return err
		}
		_, err = fmt.Fprint(deps.stdout, doc.Body)
		return err

	case "validate":
		values, err := parseFlags(args[1:])
		if err != nil {
			return err
		}
		docType := values["type"]
		if docType == "" {
			return fmt.Errorf("type is required\n\n%s", docUsageText)
		}
		id := values["id"]
		if id == "" {
			return fmt.Errorf("id is required\n\n%s", docUsageText)
		}
		doc, err := svc.Retrieve(document.DocType(docType), id)
		if err != nil {
			return err
		}
		return printDocumentValidationResults(deps.stdout, svc.Validate(doc))

	case "list":
		values, err := parseFlags(args[1:])
		if err != nil {
			return err
		}
		docType := values["type"]
		if docType == "" {
			results, err := svc.ListAll()
			if err != nil {
				return err
			}
			return printDocumentListResults(deps.stdout, results)
		}
		results, err := svc.ListByType(document.DocType(docType))
		if err != nil {
			return err
		}
		return printDocumentListResults(deps.stdout, results)

	default:
		return fmt.Errorf("unknown doc subcommand %q\n\n%s", args[0], docUsageText)
	}
}

func runHealth(deps dependencies) error {
	svc := deps.newEntityService("")
	report, err := svc.HealthCheck()
	if err != nil {
		return err
	}

	if _, err := fmt.Fprintf(
		deps.stdout,
		"health check\nentities: %d\nerrors: %d\nwarnings: %d\n",
		report.Summary.TotalEntities,
		report.Summary.ErrorCount,
		report.Summary.WarningCount,
	); err != nil {
		return err
	}

	for _, validationErr := range report.Errors {
		if _, err := fmt.Fprintf(deps.stdout, "error: %s\n", validationErr.Error()); err != nil {
			return err
		}
	}
	for _, warning := range report.Warnings {
		if _, err := fmt.Fprintf(deps.stdout, "%s\n", warning.Error()); err != nil {
			return err
		}
	}

	return nil
}

func runValidate(args []string, deps dependencies) error {
	values, err := parseFlags(args)
	if err != nil {
		return err
	}

	entityType := values["type"]
	if entityType == "" {
		return fmt.Errorf("type is required\n\n%s", validateUsageText)
	}

	fields := make(map[string]any, len(values)-1)
	for k, v := range values {
		if k == "type" {
			continue
		}
		fields[k] = v
	}

	svc := deps.newEntityService("")
	return printValidationResults(deps.stdout, svc.ValidateCandidate(entityType, fields))
}

func printDocumentResult(w io.Writer, action string, result document.DocumentResult) error {
	_, err := fmt.Fprintf(
		w,
		"%s document\nid: %s\ntype: %s\ntitle: %s\nstatus: %s\npath: %s\n",
		action,
		result.ID,
		result.Type,
		result.Title,
		result.Status,
		result.Path,
	)
	return err
}

func printDocumentValidationResults(w io.Writer, errs []document.ValidationError) error {
	if len(errs) == 0 {
		_, err := fmt.Fprintln(w, "document is valid")
		return err
	}

	if _, err := fmt.Fprintf(w, "document validation errors: %d\n", len(errs)); err != nil {
		return err
	}
	for _, validationErr := range errs {
		if _, err := fmt.Fprintf(w, "%s\n", validationErr.Error()); err != nil {
			return err
		}
	}
	return nil
}

func printDocumentListResults(w io.Writer, results []document.DocumentResult) error {
	if _, err := fmt.Fprintln(w, "listed documents"); err != nil {
		return err
	}
	for _, result := range results {
		if _, err := fmt.Fprintf(
			w,
			"%s\t%s\t%s\t%s\t%s\n",
			result.ID,
			result.Type,
			result.Title,
			result.Status,
			result.Path,
		); err != nil {
			return err
		}
	}
	return nil
}

func printValidationResults(w io.Writer, errs []validate.ValidationError) error {
	if len(errs) == 0 {
		_, err := fmt.Fprintln(w, "candidate is valid")
		return err
	}

	if _, err := fmt.Fprintf(w, "validation errors: %d\n", len(errs)); err != nil {
		return err
	}
	for _, validationErr := range errs {
		if _, err := fmt.Fprintf(w, "%s\n", validationErr.Error()); err != nil {
			return err
		}
	}
	return nil
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
  doc        Manage Phase 1 documents
  health     Run a health check against canonical state
  validate   Validate a candidate entity without persisting it
  cache      Manage the local derived cache

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
    [--epic_slug]

  feature
    --slug
    --epic
    --summary
    --created_by

  task
    --slug
    --parent_feature
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

const docUsageText = `kanbanzai doc <subcommand> [flags]

Subcommands:
  scaffold
    --type
    [--title]

  submit
    --type
    --title
    --created_by
    [--feature]
    [--body]

  approve
    --type
    --id
    --approved_by

  retrieve
    --type
    --id

  validate
    --type
    --id

  list
    [--type]
`

const validateUsageText = `kanbanzai validate [flags]

Flags:
  --type
  --<field_name> <value>

All flags other than --type are treated as candidate entity fields.
`

func runCache(args []string, deps dependencies) error {
	if len(args) == 0 {
		return fmt.Errorf("missing cache subcommand\n\n%s", cacheUsageText)
	}

	if args[0] != "rebuild" {
		return fmt.Errorf("unknown cache subcommand %q\n\n%s", args[0], cacheUsageText)
	}

	svc := deps.newEntityService("")
	cacheDir := filepath.Join(core.InstanceRootDir, cache.CacheDir)
	c, err := cache.Open(cacheDir)
	if err != nil {
		return fmt.Errorf("open cache: %w", err)
	}
	defer c.Close()

	svc.SetCache(c)
	count, err := svc.RebuildCache()
	if err != nil {
		return fmt.Errorf("rebuild cache: %w", err)
	}

	fmt.Fprintf(deps.stdout, "cache rebuilt: %d entities cached\npath: %s\n", count, c.Path())
	return nil
}

const updateUsageText = `kanbanzai update status [flags]

Subcommands:
  status    Update the lifecycle status of an entity
  fields    Update fields of an existing entity (error correction)

status flags:
  --type
  --id
  --slug
  --status

fields flags:
  --type
  --id
  --slug
  --<field_name> <value>   (any other flags become field updates)

Cannot change id (immutable) or status (use update status).
`

const cacheUsageText = `kanbanzai cache <subcommand>

Subcommands:
  rebuild   Rebuild the local derived cache from canonical entity files

The cache accelerates queries but is not required for correctness.
It is stored in .kbz/cache/ and is not committed to Git.
`
