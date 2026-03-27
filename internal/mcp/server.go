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

	// Load local config for GitHub token (best-effort).
	localConfig, _ := config.LoadLocalConfig()

	// Checkpoint store and dispatch service.
	checkpointStore := checkpoint.NewStore(stateRoot)
	dispatchSvc := service.NewDispatchService(entitySvc, knowledgeSvc)

	// Shared services used by both GroupPlanning and GroupGit.
	decomposeSvc := service.NewDecomposeService(entitySvc, docRecordSvc)
	conflictSvc := service.NewConflictService(entitySvc, newWorktreeBranchLookup(worktreeStore, repoRoot), repoRoot)
	branchThresholds := git.BranchThresholds{
		StaleAfterDays:      cfg.BranchTracking.StaleAfterDays,
		DriftWarningCommits: cfg.BranchTracking.DriftWarningCommits,
		DriftErrorCommits:   cfg.BranchTracking.DriftErrorCommits,
	}

	// 2.0 core group tools.
	if groups[config.GroupCore] {
		// Track D: status synthesis dashboard
		mcpServer.AddTools(StatusTools(entitySvc, docRecordSvc)...)
		// Track E: finish — completion + inline knowledge + lenient lifecycle
		mcpServer.AddTools(FinishTools(entitySvc, dispatchSvc)...)
		// Track F: next — work queue inspection and task claiming
		mcpServer.AddTools(NextTools(entitySvc, dispatchSvc, profileStore, knowledgeSvc, intelligenceSvc)...)
		// Track G: handoff — sub-agent prompt generation
		mcpServer.AddTools(HandoffTools(entitySvc, profileStore, knowledgeSvc, intelligenceSvc)...)
		// Track H: entity — consolidated entity CRUD
		mcpServer.AddTools(EntityTool(entitySvc)...)
		// Track I: doc — consolidated document operations
		mcpServer.AddTools(DocTool(docRecordSvc, intelligenceSvc)...)
		// Track K: health — consolidated health check
		mcpServer.AddTools(HealthTool(entitySvc,
			knowledgeHealthChecker(knowledgeSvc, profileStore),
			profileHealthChecker(profileStore),
			Phase3HealthChecker(worktreeStore, knowledgeSvc, cfg, repoRoot),
			Phase4aHealthChecker(entitySvc, worktreeStore, checkpointStore, cfg.Dispatch.StallThresholdDays, repoRoot),
			Phase4bHealthChecker(entitySvc, cfg.Incidents.RCALinkWarnAfterDays),
		)...)
	}

	// GroupPlanning: decompose, estimate, conflict.
	if groups[config.GroupPlanning] {
		mcpServer.AddTools(DecomposeTool(decomposeSvc, entitySvc)...)
		mcpServer.AddTools(EstimateTool(entitySvc, knowledgeSvc)...)
		mcpServer.AddTools(ConflictTool(conflictSvc)...)
	}

	// GroupKnowledge: knowledge, profile.
	if groups[config.GroupKnowledge] {
		mcpServer.AddTools(KnowledgeTool(knowledgeSvc)...)
		mcpServer.AddTools(ProfileTool(profileStore)...)
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

	return mcpServer
}

// Serve starts the MCP server on stdio transport.
func Serve() error {
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
// all context profiles for schema correctness and inheritance resolution.
func profileHealthChecker(profileStore *kbzctx.ProfileStore) AdditionalHealthChecker {
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
