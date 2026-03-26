package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	"path/filepath"
	"strconv"

	"kanbanzai/internal/cache"
	"kanbanzai/internal/config"
	kbzctx "kanbanzai/internal/context"
	"kanbanzai/internal/core"

	"kanbanzai/internal/id"
	kbzmcp "kanbanzai/internal/mcp"
	"kanbanzai/internal/service"
	"kanbanzai/internal/validate"
)

// version is injected at link time via: go build -ldflags "-X main.version=1.0.0"
var version = "dev"

type entityService interface {
	CreatePlan(service.CreatePlanInput) (service.CreateResult, error)
	CreateEpic(service.CreateEpicInput) (service.CreateResult, error)
	CreateFeature(service.CreateFeatureInput) (service.CreateResult, error)
	CreateTask(service.CreateTaskInput) (service.CreateResult, error)
	CreateBug(service.CreateBugInput) (service.CreateResult, error)
	CreateDecision(service.CreateDecisionInput) (service.CreateResult, error)
	GetPlan(id string) (service.ListResult, error)
	Get(entityType, entityID, slug string) (service.GetResult, error)
	List(entityType string) ([]service.ListResult, error)
	ListPlans(filters service.PlanFilters) ([]service.ListResult, error)
	UpdateStatus(service.UpdateStatusInput) (service.GetResult, error)
	UpdateEntity(service.UpdateEntityInput) (service.GetResult, error)
	ValidateCandidate(entityType string, fields map[string]any) []validate.ValidationError
	HealthCheck() (*validate.HealthReport, error)
	RebuildCache() (int, error)
	SetCache(c *cache.Cache)
}

type dependencies struct {
	stdout           io.Writer
	stdin            io.Reader
	newEntityService func(root string) entityService
}

func defaultDependencies() dependencies {
	return dependencies{
		stdout: os.Stdout,
		stdin:  os.Stdin,
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
	if deps.stdin == nil {
		deps.stdin = os.Stdin
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
	case "version", "--version", "-v":
		fmt.Fprintln(deps.stdout, "kanbanzai "+version)
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

	case "health":
		return runHealth(deps)
	case "validate":
		return runValidate(args[1:], deps)
	case "cache":
		return runCache(args[1:], deps)
	case "import":
		return runImport(args[1:], deps)
	case "knowledge":
		return runKnowledge(args[1:], deps)
	case "profile":
		return runProfile(args[1:], deps)
	case "context":
		return runContext(args[1:], deps)
	case "worktree":
		return runWorktree(args[1:], deps)
	case "branch":
		return runBranch(args[1:], deps)
	case "merge":
		return runMerge(args[1:], deps)
	case "pr":
		return runPR(args[1:], deps)
	case "cleanup":
		return runCleanup(args[1:], deps)
	case "feature":
		return runFeature(args[1:], deps)
	case "task":
		return runTask(args[1:], deps)
	case "queue":
		return runQueue(args[1:], deps)
	case "incident":
		return runIncident(args[1:], deps)
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
	case "plan":
		values, err := parseFlags(args[1:])
		if err != nil {
			return err
		}
		result, err := svc.CreatePlan(service.CreatePlanInput{
			Prefix:    values["prefix"],
			Slug:      values["slug"],
			Title:     values["title"],
			Summary:   values["summary"],
			CreatedBy: values["created_by"],
		})
		if err != nil {
			return err
		}
		return printCreateResult(deps.stdout, result)
	case "epic":
		return fmt.Errorf("'epic' is deprecated; use 'create plan' instead")
	case "feature":
		values, err := parseFlags(args[1:])
		if err != nil {
			return err
		}
		result, err := svc.CreateFeature(service.CreateFeatureInput{
			Slug:      values["slug"],
			Parent:    values["parent"],
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
	case "plans":
		results, err := svc.ListPlans(service.PlanFilters{})
		if err != nil {
			return fmt.Errorf("list plans: %w", err)
		}
		fmt.Fprintln(deps.stdout, "listed plan")
		for _, r := range results {
			status, _ := r.State["status"].(string)
			fmt.Fprintf(deps.stdout, "%s\t%s\t%s\t%s\n", r.ID, r.Slug, r.Path, status)
		}
		return nil
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

func runHealth(deps dependencies) error {
	svc := deps.newEntityService("")
	report, err := svc.HealthCheck()
	if err != nil {
		return err
	}

	// Phase 2b: knowledge health checks
	stateRoot := core.StatePath()
	knowledgeSvc := service.NewKnowledgeService(stateRoot)
	profileRoot := filepath.Join(core.InstanceRootDir, "context", "roles")
	profileStore := kbzctx.NewProfileStore(profileRoot)

	knowledgeReport, err := runKnowledgeHealthCheck(knowledgeSvc, profileStore)
	if err != nil {
		return err
	}
	report = validate.MergeReports(report, knowledgeReport)

	// Phase 2b: profile health checks
	profileReport, err := runProfileHealthCheck(profileStore)
	if err != nil {
		return err
	}
	report = validate.MergeReports(report, profileReport)

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

func runKnowledgeHealthCheck(knowledgeSvc *service.KnowledgeService, profileStore *kbzctx.ProfileStore) (*validate.HealthReport, error) {
	loadAll := func() ([]validate.KnowledgeInfo, error) {
		records, err := knowledgeSvc.LoadAllRaw()
		if err != nil {
			return nil, err
		}
		infos := make([]validate.KnowledgeInfo, len(records))
		for i, r := range records {
			infos[i] = validate.KnowledgeInfo{ID: r.ID, Fields: r.Fields}
		}
		return infos, nil
	}
	profileExists := func(id string) bool {
		p, err := profileStore.Load(id)
		return err == nil && p != nil
	}
	return validate.CheckKnowledgeHealth(loadAll, profileExists)
}

func runProfileHealthCheck(profileStore *kbzctx.ProfileStore) (*validate.HealthReport, error) {
	loadAll := func() ([]validate.ProfileInfo, error) {
		profiles, err := profileStore.LoadAll()
		if err != nil {
			return nil, err
		}
		infos := make([]validate.ProfileInfo, len(profiles))
		for i, p := range profiles {
			infos[i] = validate.ProfileInfo{ID: p.ID, Inherits: p.Inherits}
		}
		return infos, nil
	}
	resolveProfile := func(id string) error {
		_, err := kbzctx.ResolveProfile(profileStore, id)
		return err
	}
	return validate.CheckProfileHealth(loadAll, resolveProfile)
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

Phase 4b workflow kernel CLI.

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

  health     Run a health check against canonical state
  validate   Validate a candidate entity without persisting it
  cache      Manage the local derived cache
  import     Batch import document records from a directory
  knowledge  Manage knowledge entries
  profile    Manage context profiles
  context    Assemble agent context packets
  worktree   Manage Git worktrees for feature/bug development
  branch     Check branch health for worktree branches
  merge      Check merge readiness and execute merges
  pr         Manage GitHub pull requests
  cleanup    Manage post-merge cleanup of worktrees and branches
  queue      Show the current work queue with optional conflict checking
  feature    Decompose features into tasks
  task       Review completed task output
  incident   Create, list, and show incidents

Notes:
  - Kanbanzai is MCP-first; the CLI is a secondary, strict interface.
`

const createUsageText = `kanbanzai create <entity> [flags]

Entities:
  plan
    --prefix
    --slug
    --title
    --summary
    --created_by

  feature
    --slug
    --parent
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
  plan
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
  plans
  features
  tasks
  bugs
  decisions
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
  rebuild    Rebuild the local derived cache from canonical entity files
`

// runKnowledge handles the `kbz knowledge` subcommands.
func runKnowledge(args []string, deps dependencies) error {
	if len(args) == 0 {
		return fmt.Errorf("missing knowledge subcommand\n\n%s", knowledgeUsageText)
	}

	switch args[0] {
	case "list":
		return runKnowledgeList(args[1:], deps)
	case "get":
		return runKnowledgeGet(args[1:], deps)
	case "check", "confirm", "prune", "compact", "resolve":
		return runKnowledgeLifecycle(args[0], args[1:], deps)
	default:
		return fmt.Errorf("unknown knowledge subcommand %q\n\n%s", args[0], knowledgeUsageText)
	}
}

func runKnowledgeList(args []string, deps dependencies) error {
	flags, err := parseFlags(args)
	if err != nil {
		return err
	}

	svc := service.NewKnowledgeService("")

	filters := service.KnowledgeFilters{}
	if t := flags["tier"]; t != "" {
		n, err := strconv.Atoi(t)
		if err != nil {
			return fmt.Errorf("--tier must be an integer: %w", err)
		}
		filters.Tier = n
	}
	filters.Scope = flags["scope"]
	filters.Status = flags["status"]

	records, err := svc.List(filters)
	if err != nil {
		return fmt.Errorf("list knowledge entries: %w", err)
	}

	if len(records) == 0 {
		fmt.Fprintln(deps.stdout, "no knowledge entries found")
		return nil
	}

	for _, r := range records {
		id := toString2(r.Fields["id"])
		topic := toString2(r.Fields["topic"])
		status := toString2(r.Fields["status"])
		tier := toAny(r.Fields["tier"])
		scope := toString2(r.Fields["scope"])
		fmt.Fprintf(deps.stdout, "%s  tier:%v  %s  scope:%s  status:%s\n", id, tier, topic, scope, status)
	}
	return nil
}

func runKnowledgeGet(args []string, deps dependencies) error {
	if len(args) == 0 {
		return fmt.Errorf("knowledge get requires an ID")
	}
	entryID := args[0]

	svc := service.NewKnowledgeService("")
	record, err := svc.Get(entryID)
	if err != nil {
		return fmt.Errorf("get knowledge entry: %w", err)
	}

	fmt.Fprintf(deps.stdout, "id:          %s\n", toString2(record.Fields["id"]))
	fmt.Fprintf(deps.stdout, "tier:        %v\n", toAny(record.Fields["tier"]))
	fmt.Fprintf(deps.stdout, "topic:       %s\n", toString2(record.Fields["topic"]))
	fmt.Fprintf(deps.stdout, "scope:       %s\n", toString2(record.Fields["scope"]))
	fmt.Fprintf(deps.stdout, "status:      %s\n", toString2(record.Fields["status"]))
	fmt.Fprintf(deps.stdout, "confidence:  %.4f\n", toFloat2(record.Fields["confidence"]))
	fmt.Fprintf(deps.stdout, "use_count:   %v\n", toAny(record.Fields["use_count"]))
	fmt.Fprintf(deps.stdout, "miss_count:  %v\n", toAny(record.Fields["miss_count"]))
	fmt.Fprintf(deps.stdout, "content:\n  %s\n", toString2(record.Fields["content"]))
	return nil
}

// runProfile handles the `kbz profile` subcommands.
func runProfile(args []string, deps dependencies) error {
	if len(args) == 0 {
		return fmt.Errorf("missing profile subcommand\n\n%s", profileUsageText)
	}

	switch args[0] {
	case "list":
		return runProfileList(deps)
	case "get":
		return runProfileGet(args[1:], deps)
	default:
		return fmt.Errorf("unknown profile subcommand %q\n\n%s", args[0], profileUsageText)
	}
}

func runProfileList(deps dependencies) error {
	profileRoot := filepath.Join(core.InstanceRootDir, "context", "roles")
	store := kbzctx.NewProfileStore(profileRoot)

	profiles, err := store.LoadAll()
	if err != nil {
		return fmt.Errorf("list profiles: %w", err)
	}

	if len(profiles) == 0 {
		fmt.Fprintln(deps.stdout, "no profiles found in .kbz/context/roles/")
		return nil
	}

	for _, p := range profiles {
		if p.Inherits != "" {
			fmt.Fprintf(deps.stdout, "%s (inherits: %s) — %s\n", p.ID, p.Inherits, p.Description)
		} else {
			fmt.Fprintf(deps.stdout, "%s — %s\n", p.ID, p.Description)
		}
	}
	return nil
}

func runProfileGet(args []string, deps dependencies) error {
	flags, err := parseFlags(args)
	if err != nil {
		return err
	}

	profileID := flags["id"]
	if profileID == "" && len(args) > 0 && !strings.HasPrefix(args[0], "--") {
		profileID = args[0]
	}
	if profileID == "" {
		return fmt.Errorf("profile get requires a profile ID (--id <id> or positional)")
	}

	profileRoot := filepath.Join(core.InstanceRootDir, "context", "roles")
	store := kbzctx.NewProfileStore(profileRoot)

	raw := flags["raw"] == "true"
	if raw {
		p, err := store.Load(profileID)
		if err != nil {
			return fmt.Errorf("get profile: %w", err)
		}
		fmt.Fprintf(deps.stdout, "id:          %s\n", p.ID)
		if p.Inherits != "" {
			fmt.Fprintf(deps.stdout, "inherits:    %s\n", p.Inherits)
		}
		fmt.Fprintf(deps.stdout, "description: %s\n", p.Description)
		if len(p.Packages) > 0 {
			fmt.Fprintf(deps.stdout, "packages:    %s\n", strings.Join(p.Packages, ", "))
		}
		if len(p.Conventions) > 0 {
			fmt.Fprintf(deps.stdout, "conventions:\n")
			for _, c := range p.Conventions {
				fmt.Fprintf(deps.stdout, "  - %s\n", c)
			}
		}
		return nil
	}

	resolved, err := kbzctx.ResolveProfile(store, profileID)
	if err != nil {
		return fmt.Errorf("resolve profile: %w", err)
	}
	fmt.Fprintf(deps.stdout, "id:          %s (resolved)\n", resolved.ID)
	fmt.Fprintf(deps.stdout, "description: %s\n", resolved.Description)
	if len(resolved.Packages) > 0 {
		fmt.Fprintf(deps.stdout, "packages:    %s\n", strings.Join(resolved.Packages, ", "))
	}
	if len(resolved.Conventions) > 0 {
		fmt.Fprintf(deps.stdout, "conventions:\n")
		for _, c := range resolved.Conventions {
			fmt.Fprintf(deps.stdout, "  - %s\n", c)
		}
	}
	if resolved.Architecture != nil {
		fmt.Fprintf(deps.stdout, "architecture:\n")
		if resolved.Architecture.Summary != "" {
			fmt.Fprintf(deps.stdout, "  summary: %s\n", resolved.Architecture.Summary)
		}
		for _, ki := range resolved.Architecture.KeyInterfaces {
			fmt.Fprintf(deps.stdout, "  - %s\n", ki)
		}
	}
	return nil
}

// runContext handles the `kbz context` subcommands.
func runContext(args []string, deps dependencies) error {
	if len(args) == 0 {
		return fmt.Errorf("missing context subcommand\n\n%s", contextUsageText)
	}

	switch args[0] {
	case "assemble":
		return runContextAssemble(args[1:], deps)
	default:
		return fmt.Errorf("unknown context subcommand %q\n\n%s", args[0], contextUsageText)
	}
}

func runContextAssemble(args []string, deps dependencies) error {
	flags, err := parseFlags(args)
	if err != nil {
		return err
	}

	role := flags["role"]
	if role == "" {
		return fmt.Errorf("--role is required\n\n%s", contextUsageText)
	}

	taskID := flags["task"]
	maxBytes := 30720
	if mb := flags["max-bytes"]; mb != "" {
		n, err := strconv.Atoi(mb)
		if err != nil {
			return fmt.Errorf("--max-bytes must be an integer: %w", err)
		}
		maxBytes = n
	}

	profileRoot := filepath.Join(core.InstanceRootDir, "context", "roles")
	profileStore := kbzctx.NewProfileStore(profileRoot)
	knowledgeSvc := service.NewKnowledgeService("")
	entitySvc := service.NewEntityService("")
	indexRoot := filepath.Join(core.InstanceRootDir, "index")
	intelligenceSvc := service.NewIntelligenceService(indexRoot, ".")

	result, err := kbzctx.Assemble(kbzctx.AssemblyInput{
		Role:     role,
		TaskID:   taskID,
		MaxBytes: maxBytes,
	}, profileStore, knowledgeSvc, entitySvc, intelligenceSvc)
	if err != nil {
		return fmt.Errorf("context assemble: %w", err)
	}

	fmt.Fprintf(deps.stdout, "role: %s\n", result.Role)
	if result.TaskID != "" {
		fmt.Fprintf(deps.stdout, "task: %s\n", result.TaskID)
	}
	fmt.Fprintf(deps.stdout, "items: %d  bytes: %d", len(result.Items), result.ByteCount)
	if result.Trimmed > 0 {
		fmt.Fprintf(deps.stdout, "  trimmed: %d", result.Trimmed)
	}
	fmt.Fprintln(deps.stdout)
	fmt.Fprintln(deps.stdout)
	for _, item := range result.Items {
		fmt.Fprintf(deps.stdout, "--- [%s] ---\n%s\n\n", item.Source, item.Content)
	}
	return nil
}

// toString2 is a local helper (avoids shadowing the validate package's toString).
func toString2(v any) string {
	if v == nil {
		return ""
	}
	s, _ := v.(string)
	return s
}

// toFloat2 extracts a float64 from an any value.
func toFloat2(v any) float64 {
	if v == nil {
		return 0
	}
	f, _ := v.(float64)
	return f
}

// toAny formats an any value for display.
func toAny(v any) any {
	if v == nil {
		return ""
	}
	return v
}

const knowledgeUsageText = `kanbanzai knowledge <subcommand> [flags]

Subcommands:
  list    List knowledge entries
    [--tier <2|3>]
    [--scope <name>]
    [--status <status>]

  get <id>    Get a knowledge entry by ID
`

const profileUsageText = `kanbanzai profile <subcommand> [flags]

Subcommands:
  list    List available context profiles

  get <id> [--raw]    Get a context profile (resolved by default)
    --raw    Return the raw profile without inheritance resolution
`

const contextUsageText = `kanbanzai context <subcommand> [flags]

Subcommands:
  assemble    Assemble a context packet for a role
    --role <name>        Profile ID (required)
    [--task <id>]        Task ID for task-specific context
    [--max-bytes <n>]    Byte ceiling (default: 30720)
`

func runImport(args []string, deps dependencies) error {
	if len(args) == 0 {
		return fmt.Errorf("path is required\n\n%s", importUsageText)
	}

	path := args[0]

	flags, err := parseFlags(args[1:])
	if err != nil {
		return err
	}

	defaultType := flags["type"]
	owner := flags["owner"]
	glob := flags["glob"]
	createdByRaw := flags["created_by"]
	if createdByRaw == "" {
		createdByRaw = flags["created-by"]
	}

	createdBy, err := config.ResolveIdentity(createdByRaw)
	if err != nil {
		return err
	}

	cfg := config.LoadOrDefault()
	docSvc := service.NewDocumentService(core.StatePath(), ".")
	importSvc := service.NewBatchImportService(docSvc)

	result, err := importSvc.Import(cfg, service.BatchImportInput{
		Path:        path,
		DefaultType: defaultType,
		Owner:       owner,
		CreatedBy:   createdBy,
		Glob:        glob,
	})
	if err != nil {
		return err
	}

	fmt.Fprintf(deps.stdout, "imported: %d\n", result.Imported)
	if len(result.Skipped) > 0 {
		fmt.Fprintf(deps.stdout, "skipped:  %d\n", len(result.Skipped))
		for _, s := range result.Skipped {
			fmt.Fprintf(deps.stdout, "  skip  %s: %s\n", s.Path, s.Reason)
		}
	}
	if len(result.Errors) > 0 {
		fmt.Fprintf(deps.stdout, "errors:   %d\n", len(result.Errors))
		for _, e := range result.Errors {
			fmt.Fprintf(deps.stdout, "  error %s: %s\n", e.Path, e.Error)
		}
	}
	return nil
}

const importUsageText = `kanbanzai import <path> [flags]

Import document records from a directory. Scans recursively for Markdown files
and creates document records. Already-imported files are skipped (idempotent).

Arguments:
  <path>    Directory to scan for documents

Flags:
  --type    <type>      Default document type when no path pattern matches
                        (design, specification, dev-plan, research, report, policy)
  --owner   <id>        Optional parent Plan or Feature ID for imported documents
  --glob    <pattern>   Only import files matching this glob pattern (e.g. "*.md", "design-*.md")
`
