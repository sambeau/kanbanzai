package mcp

import (
	"fmt"
	"log"
	"path/filepath"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/server"

	"github.com/sambeau/kanbanzai/internal/binding"
	"github.com/sambeau/kanbanzai/internal/cache"
	"github.com/sambeau/kanbanzai/internal/checkpoint"
	"github.com/sambeau/kanbanzai/internal/config"
	kbzctx "github.com/sambeau/kanbanzai/internal/context"
	"github.com/sambeau/kanbanzai/internal/core"
	"github.com/sambeau/kanbanzai/internal/gate"
	"github.com/sambeau/kanbanzai/internal/git"
	"github.com/sambeau/kanbanzai/internal/health"
	"github.com/sambeau/kanbanzai/internal/knowledge"
	"github.com/sambeau/kanbanzai/internal/model"
	"github.com/sambeau/kanbanzai/internal/service"
	"github.com/sambeau/kanbanzai/internal/skill"
	"github.com/sambeau/kanbanzai/internal/validate"
	"github.com/sambeau/kanbanzai/internal/worktree"

	"github.com/sambeau/kanbanzai/internal/actionlog"
)

const (
	ServerName    = "kanbanzai"
	ServerVersion = "2.0"
)

// NewServer creates a new MCP server with all tools registered.
// Group registration is controlled by the mcp.groups / mcp.preset section in
// .kbz/config.yaml. If no mcp section is present, all groups are enabled
// (preset: full). The entityRoot is the path for entity storage (typically
// ".kbz/state"); pass an empty string to use the default.
func NewServer(entityRoot string) *server.MCPServer {
	return newServerWithConfig(entityRoot, config.LoadOrDefault())
}

// newServerWithConfig creates an MCP server using the provided configuration.
// Separated from NewServer to allow config injection in tests.
func newServerWithConfig(entityRoot string, cfg *config.Config) *server.MCPServer {
	entitySvc := service.NewEntityService(entityRoot)

	// Documents are stored relative to the repository root (current directory).
	stateRoot := entityRoot
	if stateRoot == "" {
		stateRoot = core.StatePath()
	}
	repoRoot := "."

	// Intelligence service for document intelligence (Layers 1-4).
	indexRoot := filepath.Join(core.InstanceRootDir, "index")
	intelligenceSvc := service.NewIntelligenceService(indexRoot, repoRoot)

	docRecordSvc := service.NewDocumentService(stateRoot, repoRoot)
	docRecordSvc.SetEntityHook(service.NewEntityLifecycleHook(entitySvc))
	docRecordSvc.SetIntelligenceService(intelligenceSvc)

	// Open the local derived cache best-effort. If it fails, the service
	// operates without cache acceleration — all queries fall back to
	// filesystem reads.
	cacheDir := filepath.Join(core.InstanceRootDir, cache.CacheDir)
	if c, err := cache.Open(cacheDir); err == nil {
		entitySvc.SetCache(c)
	}

	knowledgeSvc := service.NewKnowledgeService(stateRoot)

	profileRoot := filepath.Join(core.InstanceRootDir, "context", "roles")
	profileStore := kbzctx.NewProfileStore(profileRoot)

	// 3.0 role store: checks .kbz/roles/ first, falls back to .kbz/context/roles/.
	roleNewRoot := filepath.Join(core.InstanceRootDir, "roles")
	roleLegacyRoot := profileRoot
	roleStore := kbzctx.NewRoleStore(roleNewRoot, roleLegacyRoot)

	// 3.0 skill store: reads from .kbz/skills/.
	skillRoot := filepath.Join(core.InstanceRootDir, "skills")
	skillStore := skill.NewSkillStore(skillRoot)

	// 3.0 context assembly pipeline: constructed if a stage-bindings.yaml exists.
	// When nil, handoff falls back to the legacy 2.0 assembly path (NFR-003).
	var capTracker *knowledge.CapTracker
	var pipeline *kbzctx.Pipeline
	bindingPath := filepath.Join(core.InstanceRootDir, "stage-bindings.yaml")
	if bf, errs := binding.LoadBindingFile(bindingPath); bf != nil && len(errs) == 0 {
		capTracker = knowledge.NewCapTracker(cacheDir)
		entryLoader := func() ([]map[string]any, error) {
			records, err := knowledgeSvc.List(service.KnowledgeFilters{})
			if err != nil {
				return nil, err
			}
			fields := make([]map[string]any, len(records))
			for i, r := range records {
				fields[i] = r.Fields
			}
			return fields, nil
		}
		pipeline = &kbzctx.Pipeline{
			Roles:               &kbzctx.RoleStoreAdapter{Store: roleStore},
			Skills:              &kbzctx.SkillStoreAdapter{Store: skillStore},
			Bindings:            &kbzctx.BindingFileAdapter{File: bf},
			Knowledge:           kbzctx.NewSurfacer(entryLoader, capTracker, nil),
			StalenessWindowDays: cfg.Freshness.StalenessWindowDays,
		}
		log.Printf("[server] 3.0 context assembly pipeline loaded with %d stage bindings", len(bf.StageBindings))
	}

	mcpServer := server.NewMCPServer(
		ServerName,
		ServerVersion,
		server.WithToolCapabilities(false),
	)

	// Worktree store and git ops (needed for transition hooks, git group tools, and health).
	worktreeStore := worktree.NewStore(stateRoot)
	gitOps := worktree.NewGit(repoRoot)

	// Resolve effective group configuration (Kanbanzai 2.0 feature group framework).
	groups := resolveServerGroups(cfg)

	// Automatic worktree creation on task→active / bug→in-progress.
	// Automatic dependency unblocking on task→done/not-planned/duplicate.
	entitySvc.SetStatusTransitionHook(
		service.NewCompositeTransitionHook(
			service.NewWorktreeTransitionHook(worktreeStore, gitOps, entitySvc),
			service.NewDependencyUnblockingHook(entitySvc),
		),
	)

	// Action-pattern logging: create writer and hook.
	// Writer appends JSONL to .kbz/logs/; hook wraps every tool handler.
	logWriter := actionlog.NewWriter(actionlog.LogsDir())
	logHook := actionlog.NewHook(logWriter, &entityStageLookup{svc: entitySvc})

	// Load local config for GitHub token (best-effort).
	localConfig, _ := config.LoadLocalConfig()

	// Checkpoint store and dispatch service.
	checkpointStore := checkpoint.NewStore(stateRoot)
	dispatchSvc := service.NewDispatchService(entitySvc, knowledgeSvc)

	// Gate router: registry-driven gate evaluation with hardcoded fallback.
	registryCache := gate.NewRegistryCache(bindingPath)
	gateRouter := gate.NewGateRouter(registryCache, func(from, to string, feature *model.Feature, _ gate.DocumentService, _ gate.EntityService) gate.GateResult {
		// Hardcoded fallback: call service.CheckTransitionGate directly with
		// the concrete service types (captured in this closure).
		svcResult := service.CheckTransitionGate(from, to, feature, docRecordSvc, entitySvc)
		return gate.GateResult{
			Stage:     svcResult.Stage,
			Satisfied: svcResult.Satisfied,
			Reason:    svcResult.Reason,
		}
	})

	// Shared services used by both GroupPlanning and GroupGit.
	decomposeSvc := service.NewDecomposeService(entitySvc, docRecordSvc)
	conflictSvc := service.NewConflictService(entitySvc, newWorktreeBranchLookup(worktreeStore, repoRoot), repoRoot)
	branchThresholds := git.BranchThresholds{
		StaleAfterDays:      cfg.BranchTracking.StaleAfterDays,
		DriftWarningCommits: cfg.BranchTracking.DriftWarningCommits,
		DriftErrorCommits:   cfg.BranchTracking.DriftErrorCommits,
	}

	// server_info is unconditional — present regardless of mcp.groups or mcp.preset config.
	mcpServer.AddTools(ServerInfoTool()...)

	// 2.0 core group tools.
	if groups[config.GroupCore] {
		// Track D: status synthesis dashboard
		mcpServer.AddTools(StatusTools(entitySvc, docRecordSvc, worktreeStore)...)
		// Track E: finish — completion + inline knowledge + lenient lifecycle
		mcpServer.AddTools(FinishTools(entitySvc, dispatchSvc)...)
		// Track F: next — work queue inspection and task claiming
		mcpServer.AddTools(NextTools(entitySvc, dispatchSvc, profileStore, knowledgeSvc, intelligenceSvc, docRecordSvc)...)
		// Track G: handoff — sub-agent prompt generation (3.0 pipeline + legacy fallback)
		mcpServer.AddTools(HandoffTools(entitySvc, profileStore, knowledgeSvc, intelligenceSvc, docRecordSvc, pipeline)...)
		// Track H: entity — consolidated entity CRUD
		mcpServer.AddTools(EntityTool(entitySvc, docRecordSvc, gateRouter, checkpointStore)...)
		// Track I: doc — consolidated document operations
		mcpServer.AddTools(DocTool(docRecordSvc, intelligenceSvc)...)

	}

	// GroupPlanning: decompose, estimate, conflict, retro.
	if groups[config.GroupPlanning] {
		mcpServer.AddTools(DecomposeTool(decomposeSvc, entitySvc)...)
		mcpServer.AddTools(EstimateTool(entitySvc, knowledgeSvc)...)
		mcpServer.AddTools(ConflictTool(conflictSvc)...)
		retroSvc := service.NewRetroService(knowledgeSvc, entitySvc, docRecordSvc, repoRoot)
		mcpServer.AddTools(RetroTool(retroSvc)...)
	}

	// GroupKnowledge: knowledge, profile.
	if groups[config.GroupKnowledge] {
		mcpServer.AddTools(KnowledgeTool(knowledgeSvc, repoRoot)...)
		mcpServer.AddTools(ProfileTool(roleStore)...)
	}

	// GroupGit: worktree, merge, pr, branch, cleanup.
	if groups[config.GroupGit] {
		mcpServer.AddTools(WorktreeTool(worktreeStore, entitySvc, gitOps)...)
		mcpServer.AddTools(MergeTool(worktreeStore, entitySvc, repoRoot, branchThresholds, localConfig)...)
		mcpServer.AddTools(PRTool(worktreeStore, entitySvc, repoRoot, branchThresholds, localConfig)...)
		mcpServer.AddTools(BranchTool(worktreeStore, repoRoot, branchThresholds)...)
		mcpServer.AddTools(CleanupTool(worktreeStore, gitOps, &cfg.Cleanup)...)
	}

	// GroupDocuments: doc_intel.
	if groups[config.GroupDocuments] {
		mcpServer.AddTools(DocIntelTool(intelligenceSvc, docRecordSvc)...)
	}

	// GroupIncidents: incident.
	if groups[config.GroupIncidents] {
		mcpServer.AddTools(IncidentTool(entitySvc)...)
	}

	// GroupCheckpoints: checkpoint.
	if groups[config.GroupCheckpoints] {
		mcpServer.AddTools(CheckpointTool(checkpointStore)...)
	}

	// Register health tool last so DocCurrencyHealthChecker can see all tool names.
	if groups[config.GroupCore] {
		toolNames := make(map[string]bool)
		for name := range mcpServer.ListTools() {
			toolNames[name] = true
		}
		mcpServer.AddTools(HealthTool(entitySvc,
			knowledgeHealthChecker(knowledgeSvc, profileStore),
			profileHealthChecker(roleStore),
			Phase3HealthChecker(worktreeStore, knowledgeSvc, cfg, repoRoot),
			Phase4aHealthChecker(entitySvc, worktreeStore, checkpointStore, cfg.Dispatch.StallThresholdDays, repoRoot),
			Phase4bHealthChecker(entitySvc, cfg.Incidents.RCALinkWarnAfterDays),
			DocCurrencyHealthChecker(toolNames, repoRoot, entitySvc, docRecordSvc),
			capSaturationHealthChecker(capTracker),
			freshnessHealthChecker(cfg),
			GateOverrideHealthChecker(entitySvc),
			GateSourceHealthChecker(registryCache),
			CheckpointOverrideHealthChecker(entitySvc),
		)...)
	}

	// Wrap all registered tool handlers with the action-pattern logging hook.
	wrapAllTools(mcpServer, logHook)

	return mcpServer
}

// Serve starts the MCP server on stdio transport.
func Serve() error {
	// Best-effort log cleanup — remove log files older than 30 days.
	_ = actionlog.Cleanup(actionlog.LogsDir(), time.Now().UTC())

	mcpServer := NewServer("")
	return server.ServeStdio(mcpServer)
}

// knowledgeHealthChecker returns an AdditionalHealthChecker that validates
// all knowledge entries against schema and confidence consistency.
func knowledgeHealthChecker(knowledgeSvc *service.KnowledgeService, profileStore *kbzctx.ProfileStore) AdditionalHealthChecker {
	return func() (*validate.HealthReport, error) {
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
}

// worktreeBranchLookup adapts a worktree.Store into the service.BranchLookup interface.
type worktreeBranchLookup struct {
	store *worktree.Store
}

func newWorktreeBranchLookup(store *worktree.Store, repoRoot string) *worktreeBranchLookup {
	return &worktreeBranchLookup{store: store}
}

func (w *worktreeBranchLookup) GetBranchForEntity(entityID string) (string, error) {
	rec, err := w.store.GetByEntityID(entityID)
	if err != nil {
		return "", err
	}
	return rec.Branch, nil
}

func (w *worktreeBranchLookup) GetFilesOnBranch(repoRoot, branch string) ([]string, error) {
	return git.GetFilesChangedOnBranch(repoRoot, branch)
}

// profileHealthChecker returns an AdditionalHealthChecker that validates
// all roles for schema correctness and inheritance resolution.
// Uses RoleStore so that roles in .kbz/roles/ are visible to health validation.
func profileHealthChecker(roleStore *kbzctx.RoleStore) AdditionalHealthChecker {
	return func() (*validate.HealthReport, error) {
		loadAll := func() ([]validate.ProfileInfo, error) {
			roles, err := roleStore.LoadAll()
			if err != nil {
				return nil, err
			}
			infos := make([]validate.ProfileInfo, len(roles))
			for i, r := range roles {
				infos[i] = validate.ProfileInfo{ID: r.ID, Inherits: r.Inherits}
			}
			return infos, nil
		}
		resolveProfile := func(id string) error {
			_, err := kbzctx.ResolveRole(roleStore, id)
			return err
		}
		return validate.CheckProfileHealth(loadAll, resolveProfile)
	}
}

// freshnessHealthChecker returns an AdditionalHealthChecker that detects
// stale and never-verified role and skill files.
func freshnessHealthChecker(cfg *config.Config) AdditionalHealthChecker {
	return func() (*validate.HealthReport, error) {
		report := &validate.HealthReport{
			Summary: validate.HealthSummary{
				EntitiesByType: make(map[string]int),
			},
		}
		window := cfg.Freshness.StalenessWindowDays
		if window <= 0 {
			window = 30
		}
		now := time.Now()

		rolesDir := filepath.Join(core.InstanceRootDir, "roles")
		roleResult := health.CheckRoleFreshness(rolesDir, window, now)
		for _, issue := range roleResult.Issues {
			report.Warnings = append(report.Warnings, validate.ValidationWarning{
				EntityType: "role",
				EntityID:   issue.EntityID,
				Field:      "last_verified",
				Message:    issue.Message,
			})
		}

		skillsDir := filepath.Join(core.InstanceRootDir, "skills")
		skillResult := health.CheckSkillFreshness(skillsDir, window, now)
		for _, issue := range skillResult.Issues {
			report.Warnings = append(report.Warnings, validate.ValidationWarning{
				EntityType: "skill",
				EntityID:   issue.EntityID,
				Field:      "last_verified",
				Message:    issue.Message,
			})
		}

		report.Summary.WarningCount = len(report.Warnings)
		return report, nil
	}
}

// capSaturationHealthChecker returns an AdditionalHealthChecker that flags
// scopes where the knowledge auto-surfacing cap (10 entries) is routinely
// exceeded (3+ consecutive assemblies). Recommends knowledge compaction.
func capSaturationHealthChecker(tracker *knowledge.CapTracker) AdditionalHealthChecker {
	return func() (*validate.HealthReport, error) {
		report := &validate.HealthReport{
			Summary: validate.HealthSummary{
				EntitiesByType: make(map[string]int),
			},
		}
		if tracker == nil {
			return report, nil
		}
		scopes := tracker.ScopesNeedingCompaction()
		for _, sc := range scopes {
			report.Warnings = append(report.Warnings, validate.ValidationWarning{
				EntityType: "knowledge",
				Field:      "cap_saturation",
				Message: fmt.Sprintf(
					"scope %q exceeded the 10-entry auto-surfacing cap on %d consecutive assemblies; consider running knowledge compact for this scope",
					sc.Scope, sc.ConsecutiveHits,
				),
			})
		}
		report.Summary.WarningCount = len(report.Warnings)
		return report, nil
	}
}

// ─── Action-log integration ───────────────────────────────────────────────────

// entityStageLookup adapts *service.EntityService to actionlog.StageLookup.
type entityStageLookup struct {
	svc *service.EntityService
}

// GetEntityKindAndParent returns the entity kind and parent feature ID for entityID.
// Kind is inferred from the ID prefix; parent_feature is read from the entity record.
func (l *entityStageLookup) GetEntityKindAndParent(entityID string) (kind, parentFeatureID string, err error) {
	entityType := entityTypeFromPrefix(entityID)
	if entityType == "" {
		return "", "", fmt.Errorf("unknown entity ID prefix: %s", entityID)
	}
	result, err := l.svc.Get(entityType, entityID, "")
	if err != nil {
		return "", "", err
	}
	parent, _ := result.State["parent_feature"].(string)
	return entityType, parent, nil
}

// GetFeatureStage returns the lifecycle status for featureID.
func (l *entityStageLookup) GetFeatureStage(featureID string) (string, error) {
	result, err := l.svc.Get("feature", featureID, "")
	if err != nil {
		return "", err
	}
	stage, _ := result.State["status"].(string)
	return stage, nil
}

// entityTypeFromPrefix determines an entity's service type from its ID prefix.
func entityTypeFromPrefix(id string) string {
	switch {
	case strings.HasPrefix(id, "FEAT-"):
		return "feature"
	case strings.HasPrefix(id, "TASK-"):
		return "task"
	case strings.HasPrefix(id, "BUG-"):
		return "bug"
	case strings.HasPrefix(id, "DEC-"):
		return "decision"
	case strings.HasPrefix(id, "INC-"):
		return "incident"
	default:
		return ""
	}
}

// wrapAllTools wraps every tool handler registered in mcpServer with the
// action-pattern logging hook. It uses ListTools + SetTools to apply the
// wrapping after all tools have been registered.
func wrapAllTools(mcpServer *server.MCPServer, hook *actionlog.Hook) {
	if hook == nil {
		return
	}
	toolMap := mcpServer.ListTools()
	wrapped := make([]server.ServerTool, 0, len(toolMap))
	for _, t := range toolMap {
		inner := t.Handler
		name := t.Tool.Name
		t.Handler = server.ToolHandlerFunc(hook.Wrap(name, actionlog.HandlerFunc(inner)))
		wrapped = append(wrapped, *t)
	}
	mcpServer.SetTools(wrapped...)
}
