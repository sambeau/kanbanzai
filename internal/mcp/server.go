package mcp

import (
	"path/filepath"

	"github.com/mark3labs/mcp-go/server"

	"kanbanzai/internal/cache"
	"kanbanzai/internal/checkpoint"
	"kanbanzai/internal/config"
	kbzctx "kanbanzai/internal/context"
	"kanbanzai/internal/core"
	"kanbanzai/internal/git"
	"kanbanzai/internal/service"
	"kanbanzai/internal/validate"
	"kanbanzai/internal/worktree"
)

const (
	ServerName    = "kanbanzai"
	ServerVersion = "2.0-dev"
)

// NewServer creates a new MCP server with all tools registered.
// The entityRoot is the root path for entity storage (typically ".kbz/state").
// Pass empty strings to use the default paths.
func NewServer(entityRoot string) *server.MCPServer {
	entitySvc := service.NewEntityService(entityRoot)

	// Create document record service for Phase 2a document management
	stateRoot := entityRoot
	if stateRoot == "" {
		stateRoot = core.StatePath()
	}
	// Documents are stored relative to the repository root (current directory)
	repoRoot := "."

	// Create intelligence service for document intelligence (Layers 1-4)
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

	// Phase 2b: knowledge service
	knowledgeSvc := service.NewKnowledgeService(stateRoot)

	// Phase 2b: context profile store
	profileRoot := filepath.Join(core.InstanceRootDir, "context", "roles")
	profileStore := kbzctx.NewProfileStore(profileRoot)

	mcpServer := server.NewMCPServer(
		ServerName,
		ServerVersion,
		server.WithToolCapabilities(false),
	)

	// Phase 3 worktree store and git ops (needed for health checker and cleanup tools)
	worktreeStore := worktree.NewStore(stateRoot)
	gitOps := worktree.NewGit(repoRoot)
	cfg := config.LoadOrDefault()

	// Resolve effective group configuration (Kanbanzai 2.0 feature group framework).
	groups := resolveServerGroups(cfg)

	// Phase 3: automatic worktree creation on task→active / bug→in-progress
	// Phase 4b: automatic dependency unblocking on task→done/not-planned/duplicate
	entitySvc.SetStatusTransitionHook(
		service.NewCompositeTransitionHook(
			service.NewWorktreeTransitionHook(worktreeStore, gitOps, entitySvc),
			service.NewDependencyUnblockingHook(entitySvc),
		),
	)

	// Phase 3: load local config for GitHub token (best-effort)
	localConfig, _ := config.LoadLocalConfig()

	// Phase 4a: checkpoint store and dispatch service
	checkpointStore := checkpoint.NewStore(stateRoot)
	dispatchSvc := service.NewDispatchService(entitySvc, knowledgeSvc)

	// _legacy group: all 1.0 tool registrations, enabled during the Kanbanzai 2.0
	// dual-registration development period (Tracks A–J). Removed in Track K.
	if groups[GroupLegacy] {
		// Phase 1 entity tools (with Phase 2b, Phase 3, Phase 4a, and Phase 4b health checkers)
		mcpServer.AddTools(EntityTools(entitySvc,
			phase2bKnowledgeHealthChecker(knowledgeSvc, profileStore),
			phase2bProfileHealthChecker(profileStore),
			Phase3HealthChecker(worktreeStore, knowledgeSvc, cfg, repoRoot),
			Phase4aHealthChecker(entitySvc, worktreeStore, checkpointStore, cfg.Dispatch.StallThresholdDays, repoRoot),
			Phase4bHealthChecker(entitySvc, cfg.Incidents.RCALinkWarnAfterDays),
		)...)

		// Phase 2a Plan tools
		mcpServer.AddTools(PlanTools(entitySvc)...)

		// Phase 2a Document record tools
		mcpServer.AddTools(DocRecordTools(docRecordSvc)...)

		// Phase 2a Config tools
		mcpServer.AddTools(ConfigTools()...)

		// Phase 2a Document intelligence tools
		mcpServer.AddTools(DocIntelligenceTools(intelligenceSvc, docRecordSvc)...)

		// Phase 2a Rich query tools
		mcpServer.AddTools(QueryTools(entitySvc, docRecordSvc)...)

		// Phase 2a Migration tools
		mcpServer.AddTools(MigrationTools(entitySvc)...)

		// Phase 2b Knowledge tools
		mcpServer.AddTools(KnowledgeTools(knowledgeSvc)...)

		// Phase 2b Context profile tools
		mcpServer.AddTools(ProfileTools(profileStore)...)

		// Phase 2b Context assembly tools
		mcpServer.AddTools(ContextTools(profileStore, knowledgeSvc, entitySvc, intelligenceSvc)...)

		// Phase 2b Agent capability tools
		mcpServer.AddTools(AgentCapabilityTools(entitySvc, knowledgeSvc, intelligenceSvc)...)

		// Phase 2b Batch import tools
		mcpServer.AddTools(BatchImportTools(docRecordSvc)...)

		// Phase 3 worktree and branch tools
		branchThresholds := git.BranchThresholds{
			StaleAfterDays:      cfg.BranchTracking.StaleAfterDays,
			DriftWarningCommits: cfg.BranchTracking.DriftWarningCommits,
			DriftErrorCommits:   cfg.BranchTracking.DriftErrorCommits,
		}
		mcpServer.AddTools(WorktreeTools(worktreeStore, entitySvc, gitOps)...)
		mcpServer.AddTools(BranchTools(worktreeStore, repoRoot, branchThresholds)...)

		// Phase 3 cleanup tools
		mcpServer.AddTools(CleanupTools(worktreeStore, gitOps, &cfg.Cleanup)...)

		// Phase 3 merge tools
		mcpServer.AddTools(MergeTools(worktreeStore, entitySvc, repoRoot, branchThresholds, localConfig)...)

		// Phase 3 PR tools
		mcpServer.AddTools(PRTools(worktreeStore, entitySvc, repoRoot, branchThresholds, localConfig)...)

		// Phase 4a Queue tools
		mcpServer.AddTools(QueueTools(entitySvc)...)

		// Phase 4a Estimation tools
		mcpServer.AddTools(EstimationTools(entitySvc, knowledgeSvc)...)

		// Phase 4a Dispatch tools
		mcpServer.AddTools(DispatchTools(dispatchSvc, checkpointStore, profileStore, knowledgeSvc, entitySvc, intelligenceSvc)...)

		// Phase 4b Incident tools
		mcpServer.AddTools(IncidentTools(entitySvc)...)

		// Phase 4b Decompose tools
		decomposeSvc := service.NewDecomposeService(entitySvc, docRecordSvc)
		mcpServer.AddTools(DecomposeTools(decomposeSvc)...)

		// Phase 4b Review tools
		reviewSvc := service.NewReviewService(entitySvc, intelligenceSvc, repoRoot)
		mcpServer.AddTools(ReviewTools(reviewSvc)...)

		// Phase 4b Conflict tools
		conflictSvc := service.NewConflictService(entitySvc, newWorktreeBranchLookup(worktreeStore, repoRoot), repoRoot)
		mcpServer.AddTools(ConflictTools(conflictSvc)...)
	}

	// 2.0 core group tools are registered as they are implemented (Tracks B–K).
	// Placeholder: when groups[config.GroupCore] is true, register 2.0 core tools here.

	return mcpServer
}

// Serve starts the MCP server on stdio transport.
func Serve() error {
	mcpServer := NewServer("")
	return server.ServeStdio(mcpServer)
}

// phase2bKnowledgeHealthChecker returns an AdditionalHealthChecker that validates
// all knowledge entries against schema and confidence consistency.
func phase2bKnowledgeHealthChecker(knowledgeSvc *service.KnowledgeService, profileStore *kbzctx.ProfileStore) AdditionalHealthChecker {
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

// phase2bProfileHealthChecker returns an AdditionalHealthChecker that validates
// all context profiles for schema correctness and inheritance resolution.
func phase2bProfileHealthChecker(profileStore *kbzctx.ProfileStore) AdditionalHealthChecker {
	return func() (*validate.HealthReport, error) {
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
}
