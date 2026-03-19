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
	ServerVersion = "phase-1-dev"
)

// NewServer creates a new MCP server with all Phase 1 tools registered.
// The entityRoot is the root path for entity storage (typically ".kbz/state").
// The docsRoot is the root path for document storage (typically ".kbz/docs").
// Pass empty strings to use the default paths.
func NewServer(entityRoot, docsRoot string) *server.MCPServer {
	entitySvc := service.NewEntityService(entityRoot)
	docSvc := document.NewDocService(docsRoot)

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

	mcpServer.AddTools(EntityTools(entitySvc)...)
	mcpServer.AddTools(DocumentTools(docSvc)...)

	return mcpServer
}

// Serve starts the MCP server on stdio transport.
func Serve() error {
	mcpServer := NewServer("", "")
	return server.ServeStdio(mcpServer)
}
