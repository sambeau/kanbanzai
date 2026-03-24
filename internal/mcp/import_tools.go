package mcp

import (
	"context"
	"encoding/json"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"kanbanzai/internal/config"
	"kanbanzai/internal/service"
)

// BatchImportTools returns the MCP tool definitions for batch document import.
func BatchImportTools(docSvc *service.DocumentService) []server.ServerTool {
	return []server.ServerTool{
		batchImportDocumentsTool(docSvc),
	}
}

func batchImportDocumentsTool(docSvc *service.DocumentService) server.ServerTool {
	tool := mcp.NewTool("batch_import_documents",
		mcp.WithDescription("Batch import document records from a directory. Scans for matching Markdown files and creates document records idempotently. Already-imported files are skipped."),
		mcp.WithString("path", mcp.Description("Directory path to scan for documents"), mcp.Required()),
		mcp.WithString("default_type", mcp.Description("Fallback document type when no path pattern matches (design, specification, dev-plan, research, report, policy)")),
		mcp.WithString("owner", mcp.Description("Optional parent Plan or Feature ID to assign as owner of imported documents")),
		mcp.WithString("created_by", mcp.Description("Who is importing the documents. Auto-resolved from .kbz/local.yaml or git config if not provided.")),
	)

	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		path, err := request.RequireString("path")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		createdByRaw := request.GetString("created_by", "")
		createdBy, err := config.ResolveIdentity(createdByRaw)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		cfg := config.LoadOrDefault()
		importSvc := service.NewBatchImportService(docSvc)

		result, importErr := importSvc.Import(cfg, service.BatchImportInput{
			Path:        path,
			DefaultType: request.GetString("default_type", ""),
			Owner:       request.GetString("owner", ""),
			CreatedBy:   createdBy,
		})
		if importErr != nil {
			return mcp.NewToolResultErrorFromErr("batch import failed", importErr), nil
		}

		skipped := make([]map[string]any, 0, len(result.Skipped))
		for _, s := range result.Skipped {
			skipped = append(skipped, map[string]any{
				"path":   s.Path,
				"reason": s.Reason,
			})
		}

		errors := make([]map[string]any, 0, len(result.Errors))
		for _, e := range result.Errors {
			errors = append(errors, map[string]any{
				"path":  e.Path,
				"error": e.Error,
			})
		}

		response := map[string]any{
			"success":  true,
			"imported": result.Imported,
			"skipped":  skipped,
			"errors":   errors,
		}

		data, marshalErr := json.MarshalIndent(response, "", "  ")
		if marshalErr != nil {
			return mcp.NewToolResultError("marshal result: " + marshalErr.Error()), nil
		}
		return mcp.NewToolResultText(string(data)), nil
	}

	return server.ServerTool{Tool: tool, Handler: handler}
}
