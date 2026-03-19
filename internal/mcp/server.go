package mcp

import (
	"github.com/mark3labs/mcp-go/server"

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
