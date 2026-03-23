package mcp

import (
	"path/filepath"

	"github.com/mark3labs/mcp-go/server"

	"kanbanzai/internal/cache"
	"kanbanzai/internal/core"
	"kanbanzai/internal/document"
	"kanbanzai/internal/service"
)

const (
	ServerName    = "kanbanzai"
	ServerVersion = "phase-2a-dev"
)

// NewServer creates a new MCP server with all Phase 1 and Phase 2a tools registered.
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
	docRecordSvc := service.NewDocumentService(stateRoot, repoRoot)
	docRecordSvc.SetEntityHook(service.NewEntityLifecycleHook(entitySvc))

	// Open the local derived cache best-effort. If it fails, the service
	// operates without cache acceleration — all queries fall back to
	// filesystem reads.
	cacheDir := filepath.Join(core.InstanceRootDir, cache.CacheDir)
	if c, err := cache.Open(cacheDir); err == nil {
		entitySvc.SetCache(c)
	}

	mcpServer := server.NewMCPServer(
		ServerName,
		ServerVersion,
		server.WithToolCapabilities(false),
	)

	// Phase 1 entity tools
	mcpServer.AddTools(EntityTools(entitySvc)...)

	// Phase 1 document tools (legacy)
	mcpServer.AddTools(DocumentTools(docSvc)...)

	// Phase 2a Plan tools
	mcpServer.AddTools(PlanTools(entitySvc)...)

	// Phase 2a Document record tools
	mcpServer.AddTools(DocRecordTools(docRecordSvc)...)

	// Phase 2a Config tools
	mcpServer.AddTools(ConfigTools()...)

	return mcpServer
}

// Serve starts the MCP server on stdio transport.
func Serve() error {
	mcpServer := NewServer("", "")
	return server.ServeStdio(mcpServer)
}
