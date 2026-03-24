package mcp

import (
	"path/filepath"

	"github.com/mark3labs/mcp-go/server"

	"kanbanzai/internal/cache"
	kbzctx "kanbanzai/internal/context"
	"kanbanzai/internal/core"
	"kanbanzai/internal/document"
	"kanbanzai/internal/service"
	"kanbanzai/internal/validate"
)

const (
	ServerName    = "kanbanzai"
	ServerVersion = "phase-2b-dev"
)

// NewServer creates a new MCP server with all Phase 1, Phase 2a, and Phase 2b tools registered.
// The entityRoot is the root path for entity storage (typically ".kbz/state").
// The docsRoot is the root path for document storage (typically ".kbz/docs").
// Pass empty strings to use the default paths.
func NewServer(entityRoot, docsRoot string) *server.MCPServer {
	entitySvc := service.NewEntityService(entityRoot)
	docSvc := document.NewDocService(docsRoot)

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

	// Phase 1 entity tools (with Phase 2b health checkers)
	mcpServer.AddTools(EntityTools(entitySvc,
		phase2bKnowledgeHealthChecker(knowledgeSvc, profileStore),
		phase2bProfileHealthChecker(profileStore),
	)...)

	// Phase 1 document tools (legacy)
	mcpServer.AddTools(DocumentTools(docSvc)...)

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

	// Phase 2b Knowledge tools (contribute, get, list, update, confirm, flag, retire, promote, context_report)
	mcpServer.AddTools(KnowledgeTools(knowledgeSvc)...)

	// Phase 2b Context profile tools (profile_get, profile_list)
	mcpServer.AddTools(ProfileTools(profileStore)...)

	// Phase 2b Context assembly tools (context_assemble)
	mcpServer.AddTools(ContextTools(profileStore, knowledgeSvc, entitySvc, intelligenceSvc)...)

	// Phase 2b Agent capability tools (suggest_links, check_duplicates, doc_extraction_guide)
	mcpServer.AddTools(AgentCapabilityTools(entitySvc, knowledgeSvc, intelligenceSvc)...)

	// Phase 2b Batch import tools (batch_import_documents)
	mcpServer.AddTools(BatchImportTools(docRecordSvc)...)

	return mcpServer
}

// Serve starts the MCP server on stdio transport.
func Serve() error {
	mcpServer := NewServer("", "")
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
