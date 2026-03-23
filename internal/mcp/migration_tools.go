package mcp

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"kanbanzai/internal/service"
)

// MigrationTools returns MCP tools for data migration operations.
func MigrationTools(svc *service.EntityService) []server.ServerTool {
	return []server.ServerTool{
		migratePhase2Tool(svc),
	}
}

func migratePhase2Tool(svc *service.EntityService) server.ServerTool {
	tool := mcp.NewTool("migrate_phase2",
		mcp.WithDescription("Migrate Phase 1 epic entities to Phase 2 plan entities. Converts epics to plans, updates feature references, and creates required directories. The migration is idempotent: re-running it skips already-migrated entities. Requires a configured prefix registry in .kbz/config.yaml."),
	)
	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		result, err := svc.MigratePhase2()
		if err != nil {
			return mcp.NewToolResultErrorFromErr("migration failed", err), nil
		}

		response := map[string]any{
			"success":          true,
			"plans_created":    result.PlansCreated,
			"features_updated": result.FeaturesUpdated,
			"files_moved":      result.FilesMoved,
			"dirs_created":     result.DirsCreated,
		}

		if len(result.Errors) > 0 {
			response["errors"] = result.Errors
			response["message"] = "migration completed with errors"
		} else {
			response["message"] = "migration completed successfully"
		}

		return jsonResult(response)
	}
	return server.ServerTool{Tool: tool, Handler: handler}
}
