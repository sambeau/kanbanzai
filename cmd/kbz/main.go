package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	"path/filepath"
	"strconv"

	_ "github.com/sambeau/kanbanzai/internal/buildinfo"
	"github.com/sambeau/kanbanzai/internal/cache"
	"github.com/sambeau/kanbanzai/internal/config"
	kbzctx "github.com/sambeau/kanbanzai/internal/context"
	"github.com/sambeau/kanbanzai/internal/core"

	kbzmcp "github.com/sambeau/kanbanzai/internal/mcp"
	"github.com/sambeau/kanbanzai/internal/service"
	"github.com/sambeau/kanbanzai/internal/validate"
)

// version is set at link time via -ldflags "-X main.version=<semver>".
// It defaults to "dev" for local builds.
var version = "dev"

type entityService interface {
	CreatePlan(service.CreatePlanInput) (service.CreateResult, error)
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
	version          string
	newEntityService func(root string) entityService
}

func defaultDependencies() dependencies {
	return dependencies{
		stdout:  os.Stdout,
		stdin:   os.Stdin,
		version: version,
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
	case "init":
		return runInit(args[1:], deps)

	// ── Core workflow commands (spec §23.2) ──────────────────────────────
	case "status":
		return runStatus(args[1:], deps)
	case "next":
		return runNextCmd(args[1:], deps)
	case "finish":
		return runFinish(args[1:], deps)
	case "handoff":
		return runHandoff(args[1:], deps)
	case "entity":
		return runEntity(args[1:], deps)
	case "doc":
		return runDoc(args[1:], deps)
	case "delete":
		return runDelete(args[1:], deps)
	case "health":
		return runHealth(deps)

	// ── Feature group commands (spec §23.3) ──────────────────────────────
	case "decompose":
		return runDecompose(args[1:], deps)
	case "estimate":
		return runEstimate(args[1:], deps)
	case "conflict":
		return runConflict(args[1:], deps)
	case "knowledge":
		return runKnowledge(args[1:], deps)
	case "profile":
		return runProfile(args[1:], deps)
	case "worktree":
		return runWorktree(args[1:], deps)
	case "merge":
		return runMerge(args[1:], deps)
	case "pr":
		return runPR(args[1:], deps)
	case "branch":
		return runBranch(args[1:], deps)
	case "cleanup":
		return runCleanup(args[1:], deps)
	case "incident":
		return runIncident(args[1:], deps)
	case "checkpoint":
		return runCheckpoint(args[1:], deps)
	case "task":
		return runTask(args[1:], deps)
	case "metrics":
		return runMetrics(args[1:], deps)

	// ── Utility commands ─────────────────────────────────────────────────
	case "validate":
		return runValidate(args[1:], deps)
	case "cache":
		return runCache(args[1:], deps)
	case "import":
		return runImport(args[1:], deps)
	case "install-record":
		return runInstallRecord(args[1:], deps)
	case "rebuild-index":
		return runRebuildIndex(args[1:], deps)

	default:
		return fmt.Errorf("unknown command %q\n\n%s", args[0], usageText)
	}
}

// ─── Health ──────────────────────────────────────────────────────────────────

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

// ─── Validate ────────────────────────────────────────────────────────────────

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

// ─── Usage ───────────────────────────────────────────────────────────────────

func wantsHelp(args []string) bool {
	return len(args) > 0 && (args[0] == "-h" || args[0] == "--help")
}

func printUsage(w io.Writer) {
	_, _ = fmt.Fprint(w, usageText)
}

const usageText = `kanbanzai 2.0

Resource-oriented workflow CLI.

Usage:
  kbz <command> [options]

Setup commands:
  init                       Initialise a Git repository for use with Kanbanzai

Core commands:
  status [<id>]              Project overview or entity dashboard
  next [<task-id>]           Show the ready queue or claim a task
  finish <task-id> [opts]    Complete a task
  handoff <task-id>          Print a sub-agent prompt
  entity <action> [opts]     Create, get, list, or transition entities
  doc <action> [opts]        Register, approve, or list documents
  health                     Run a health check

Feature group commands:
  decompose <feat-id>        Propose task decomposition
  estimate <id> [<pts>]      Query or set story point estimate
  conflict <id> <id> [...]   Analyse conflict risk between tasks
  knowledge <action> [opts]  Manage knowledge entries
  profile <action> [opts]    Manage context profiles
  worktree <action> [opts]   Manage Git worktrees
  merge [check] <id>         Check merge readiness or execute merge
  pr <action> <id>           Manage GitHub pull requests
  branch <id>                Check branch health
  cleanup [opts]             Manage post-merge cleanup
  incident <action> [opts]   Create, list, and show incidents
  checkpoint <action> [opts] Create or respond to human checkpoints
  task review <id>           Review completed task output
  metrics [opts]             Show action-pattern metrics

Utility commands:
  validate [flags]           Validate a candidate entity
  cache rebuild              Rebuild the local derived cache
  import <path> [flags]      Batch import document records
  install-record write       Write binary install record

Other:
  serve                      Start the MCP server (stdio transport)
  help                       Show this help text
  version                    Show the version
`

const validateUsageText = `kanbanzai validate [flags]

Flags:
  --type
  --<field_name> <value>

All flags other than --type are treated as candidate entity fields.
`

// ─── Cache ───────────────────────────────────────────────────────────────────

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

const cacheUsageText = `kanbanzai cache <subcommand>

Subcommands:
  rebuild    Rebuild the local derived cache from canonical entity files
`

// ─── Knowledge ───────────────────────────────────────────────────────────────

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

const knowledgeUsageText = `kanbanzai knowledge <subcommand> [flags]

Subcommands:
  list    List knowledge entries
    [--tier <2|3>]
    [--scope <name>]
    [--status <status>]

  get <id>    Get a knowledge entry by ID
`

// ─── Profile ─────────────────────────────────────────────────────────────────

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
	newRoot := filepath.Join(core.InstanceRootDir, "roles")
	legacyRoot := filepath.Join(core.InstanceRootDir, "context", "roles")
	store := kbzctx.NewRoleStore(newRoot, legacyRoot)

	roles, err := store.LoadAll()
	if err != nil {
		return fmt.Errorf("list profiles: %w", err)
	}

	if len(roles) == 0 {
		fmt.Fprintln(deps.stdout, "no roles found in .kbz/roles/")
		return nil
	}

	for _, r := range roles {
		if r.Inherits != "" {
			fmt.Fprintf(deps.stdout, "%s (inherits: %s) — %s\n", r.ID, r.Inherits, r.Identity)
		} else {
			fmt.Fprintf(deps.stdout, "%s — %s\n", r.ID, r.Identity)
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

	newRoot := filepath.Join(core.InstanceRootDir, "roles")
	legacyRoot := filepath.Join(core.InstanceRootDir, "context", "roles")
	store := kbzctx.NewRoleStore(newRoot, legacyRoot)

	raw := flags["raw"] == "true"
	if raw {
		r, err := store.Load(profileID)
		if err != nil {
			return fmt.Errorf("get profile: %w", err)
		}
		fmt.Fprintf(deps.stdout, "id:       %s\n", r.ID)
		if r.Inherits != "" {
			fmt.Fprintf(deps.stdout, "inherits: %s\n", r.Inherits)
		}
		fmt.Fprintf(deps.stdout, "identity: %s\n", r.Identity)
		if len(r.Vocabulary) > 0 {
			fmt.Fprintf(deps.stdout, "vocabulary:\n")
			for _, v := range r.Vocabulary {
				fmt.Fprintf(deps.stdout, "  - %s\n", v)
			}
		}
		if len(r.AntiPatterns) > 0 {
			fmt.Fprintf(deps.stdout, "anti_patterns:\n")
			for _, ap := range r.AntiPatterns {
				fmt.Fprintf(deps.stdout, "  - name: %s\n", ap.Name)
			}
		}
		if len(r.Tools) > 0 {
			fmt.Fprintf(deps.stdout, "tools:    %s\n", strings.Join(r.Tools, ", "))
		}
		return nil
	}

	resolved, err := kbzctx.ResolveRole(store, profileID)
	if err != nil {
		return fmt.Errorf("resolve profile: %w", err)
	}
	fmt.Fprintf(deps.stdout, "id:       %s (resolved)\n", resolved.ID)
	fmt.Fprintf(deps.stdout, "identity: %s\n", resolved.Identity)
	if len(resolved.Vocabulary) > 0 {
		fmt.Fprintf(deps.stdout, "vocabulary:\n")
		for _, v := range resolved.Vocabulary {
			fmt.Fprintf(deps.stdout, "  - %s\n", v)
		}
	}
	if len(resolved.Tools) > 0 {
		fmt.Fprintf(deps.stdout, "tools:    %s\n", strings.Join(resolved.Tools, ", "))
	}
	return nil
}

const profileUsageText = `kanbanzai profile <subcommand> [flags]

Subcommands:
  list    List available context profiles

  get <id> [--raw]    Get a context profile (resolved by default)
    --raw    Return the raw profile without inheritance resolution
`

// ─── Helpers ─────────────────────────────────────────────────────────────────

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

// ─── Import ──────────────────────────────────────────────────────────────────

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
