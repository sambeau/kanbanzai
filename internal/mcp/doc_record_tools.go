package mcp

import (
	"context"
	"encoding/json"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"kanbanzai/internal/service"
)

// DocRecordTools returns all document record MCP tool definitions with their handlers.
// These are Phase 2a tools for document metadata management, distinct from Phase 1 document tools.
func DocRecordTools(docSvc *service.DocumentService) []server.ServerTool {
	return []server.ServerTool{
		docRecordSubmitTool(docSvc),
		docRecordApproveTool(docSvc),
		docRecordSupersedeTool(docSvc),
		docRecordGetTool(docSvc),
		docRecordGetContentTool(docSvc),
		docRecordListTool(docSvc),
		docRecordValidateTool(docSvc),
		docRecordListPendingTool(docSvc),
	}
}

func docRecordSubmitTool(docSvc *service.DocumentService) server.ServerTool {
	tool := mcp.NewTool("doc_record_submit",
		mcp.WithDescription("Register a document with the system, creating a document record in draft status. This computes the content hash and prepares the document for Layer 1-2 analysis. The document file must already exist at the specified path."),
		mcp.WithString("path", mcp.Description("Relative path to the document file from the repo root"), mcp.Required()),
		mcp.WithString("type", mcp.Description("Document type: design, specification, dev-plan, research, report, or policy"), mcp.Required()),
		mcp.WithString("title", mcp.Description("Human-readable title for the document"), mcp.Required()),
		mcp.WithString("owner", mcp.Description("Optional parent Plan or Feature ID that owns this document")),
		mcp.WithString("created_by", mcp.Description("Who is submitting the document"), mcp.Required()),
	)
	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		path, err := request.RequireString("path")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		docType, err := request.RequireString("type")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		title, err := request.RequireString("title")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		createdBy, err := request.RequireString("created_by")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		owner := request.GetString("owner", "")

		result, err := docSvc.SubmitDocument(service.SubmitDocumentInput{
			Path:      path,
			Type:      docType,
			Title:     title,
			Owner:     owner,
			CreatedBy: createdBy,
		})
		if err != nil {
			return mcp.NewToolResultErrorFromErr("submit document failed", err), nil
		}
		return docRecordResultJSON(result, "Document submitted successfully")
	}
	return server.ServerTool{Tool: tool, Handler: handler}
}

func docRecordApproveTool(docSvc *service.DocumentService) server.ServerTool {
	tool := mcp.NewTool("doc_record_approve",
		mcp.WithDescription("Transition a document from draft to approved status. The content hash must match the current file content. Approval triggers lifecycle transitions on the owning entity."),
		mcp.WithString("id", mcp.Description("Document record ID"), mcp.Required()),
		mcp.WithString("approved_by", mcp.Description("Who is approving the document"), mcp.Required()),
	)
	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		id, err := request.RequireString("id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		approvedBy, err := request.RequireString("approved_by")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		result, err := docSvc.ApproveDocument(service.ApproveDocumentInput{
			ID:         id,
			ApprovedBy: approvedBy,
		})
		if err != nil {
			return mcp.NewToolResultErrorFromErr("approve document failed", err), nil
		}
		return docRecordResultJSON(result, "Document approved successfully")
	}
	return server.ServerTool{Tool: tool, Handler: handler}
}

func docRecordSupersedeTool(docSvc *service.DocumentService) server.ServerTool {
	tool := mcp.NewTool("doc_record_supersede",
		mcp.WithDescription("Transition a document from approved to superseded status, linking to the newer document that replaces it. Supersession may trigger backward lifecycle transitions on the owning entity."),
		mcp.WithString("id", mcp.Description("Document record ID being superseded"), mcp.Required()),
		mcp.WithString("superseded_by", mcp.Description("Document record ID of the document that supersedes this one"), mcp.Required()),
	)
	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		id, err := request.RequireString("id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		supersededBy, err := request.RequireString("superseded_by")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		result, err := docSvc.SupersedeDocument(service.SupersedeDocumentInput{
			ID:           id,
			SupersededBy: supersededBy,
		})
		if err != nil {
			return mcp.NewToolResultErrorFromErr("supersede document failed", err), nil
		}
		return docRecordResultJSON(result, "Document superseded successfully")
	}
	return server.ServerTool{Tool: tool, Handler: handler}
}

func docRecordGetTool(docSvc *service.DocumentService) server.ServerTool {
	tool := mcp.NewTool("doc_record_get",
		mcp.WithDescription("Get a document record by ID. Returns metadata including status, owner, content hash, and drift detection."),
		mcp.WithString("id", mcp.Description("Document record ID"), mcp.Required()),
		mcp.WithBoolean("check_drift", mcp.Description("Whether to check if content has changed since recorded (default: true)")),
	)
	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		id, err := request.RequireString("id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		checkDrift := request.GetBool("check_drift", true)

		result, err := docSvc.GetDocument(id, checkDrift)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("get document failed", err), nil
		}

		message := ""
		if result.Drift {
			message = "WARNING: Document content has changed since recorded"
		}
		return docRecordResultJSON(result, message)
	}
	return server.ServerTool{Tool: tool, Handler: handler}
}

func docRecordGetContentTool(docSvc *service.DocumentService) server.ServerTool {
	tool := mcp.NewTool("doc_record_get_content",
		mcp.WithDescription("Get the content of a document file. For approved documents, this should be verbatim as approved. Includes drift detection warning if content has changed."),
		mcp.WithString("id", mcp.Description("Document record ID"), mcp.Required()),
	)
	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		id, err := request.RequireString("id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		content, result, err := docSvc.GetDocumentContent(id)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("get document content failed", err), nil
		}

		response := map[string]any{
			"success": true,
			"document": map[string]any{
				"id":           result.ID,
				"path":         result.Path,
				"type":         result.Type,
				"title":        result.Title,
				"status":       result.Status,
				"content_hash": result.ContentHash,
			},
			"content": content,
		}

		if result.Drift {
			response["warning"] = "Document content has changed since recorded"
			response["drift"] = true
			response["current_hash"] = result.CurrentHash
		}

		return docRecordMapJSON(response)
	}
	return server.ServerTool{Tool: tool, Handler: handler}
}

func docRecordListTool(docSvc *service.DocumentService) server.ServerTool {
	tool := mcp.NewTool("doc_record_list",
		mcp.WithDescription("List all document records with optional filtering by type, status, or owner."),
		mcp.WithString("type", mcp.Description("Filter by document type: design, specification, dev-plan, research, report, policy")),
		mcp.WithString("status", mcp.Description("Filter by status: draft, approved, superseded")),
		mcp.WithString("owner", mcp.Description("Filter by owner entity ID")),
	)
	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var filters service.DocumentFilters

		filters.Type = request.GetString("type", "")
		filters.Status = request.GetString("status", "")
		filters.Owner = request.GetString("owner", "")

		results, err := docSvc.ListDocuments(filters)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("list documents failed", err), nil
		}

		return docRecordListJSON(results)
	}
	return server.ServerTool{Tool: tool, Handler: handler}
}

func docRecordListPendingTool(docSvc *service.DocumentService) server.ServerTool {
	tool := mcp.NewTool("doc_record_list_pending",
		mcp.WithDescription("List all documents in draft status that are awaiting approval or classification."),
	)
	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		results, err := docSvc.ListPendingDocuments()
		if err != nil {
			return mcp.NewToolResultErrorFromErr("list pending documents failed", err), nil
		}

		return docRecordListJSON(results)
	}
	return server.ServerTool{Tool: tool, Handler: handler}
}

func docRecordValidateTool(docSvc *service.DocumentService) server.ServerTool {
	tool := mcp.NewTool("doc_record_validate",
		mcp.WithDescription("Validate a document record and check content integrity. Returns a list of any issues found."),
		mcp.WithString("id", mcp.Description("Document record ID"), mcp.Required()),
	)
	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		id, err := request.RequireString("id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		issues, err := docSvc.ValidateDocument(id)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("validate document failed", err), nil
		}

		response := map[string]any{
			"success":     true,
			"document_id": id,
			"valid":       len(issues) == 0,
			"issues":      issues,
		}

		if len(issues) == 0 {
			response["message"] = "Document is valid"
		} else {
			response["message"] = "Document has validation issues"
		}

		return docRecordMapJSON(response)
	}
	return server.ServerTool{Tool: tool, Handler: handler}
}

// docRecordResultJSON creates a JSON result for a document operation.
func docRecordResultJSON(result service.DocumentResult, message string) (*mcp.CallToolResult, error) {
	response := map[string]any{
		"success": true,
		"document": map[string]any{
			"id":           result.ID,
			"path":         result.Path,
			"record_path":  result.RecordPath,
			"type":         result.Type,
			"title":        result.Title,
			"status":       result.Status,
			"owner":        result.Owner,
			"content_hash": result.ContentHash,
			"created":      result.Created,
			"updated":      result.Updated,
		},
	}

	if message != "" {
		response["message"] = message
	}

	docMap := response["document"].(map[string]any)
	if result.ApprovedBy != "" {
		docMap["approved_by"] = result.ApprovedBy
	}
	if result.ApprovedAt != nil {
		docMap["approved_at"] = result.ApprovedAt
	}
	if result.Supersedes != "" {
		docMap["supersedes"] = result.Supersedes
	}
	if result.SupersededBy != "" {
		docMap["superseded_by"] = result.SupersededBy
	}
	if result.Drift {
		response["warning"] = "Document content has changed since recorded"
		response["drift"] = true
		response["current_hash"] = result.CurrentHash
	}

	return docRecordMapJSON(response)
}

// docRecordListJSON creates a JSON result for a list of documents.
func docRecordListJSON(results []service.DocumentResult) (*mcp.CallToolResult, error) {
	docs := make([]map[string]any, 0, len(results))
	for _, r := range results {
		doc := map[string]any{
			"id":           r.ID,
			"path":         r.Path,
			"type":         r.Type,
			"title":        r.Title,
			"status":       r.Status,
			"owner":        r.Owner,
			"content_hash": r.ContentHash,
			"created":      r.Created,
			"updated":      r.Updated,
		}
		if r.ApprovedBy != "" {
			doc["approved_by"] = r.ApprovedBy
		}
		if r.ApprovedAt != nil {
			doc["approved_at"] = r.ApprovedAt
		}
		if r.Supersedes != "" {
			doc["supersedes"] = r.Supersedes
		}
		if r.SupersededBy != "" {
			doc["superseded_by"] = r.SupersededBy
		}
		docs = append(docs, doc)
	}

	response := map[string]any{
		"success":   true,
		"count":     len(results),
		"documents": docs,
	}

	return docRecordMapJSON(response)
}

// docRecordMapJSON marshals a map to JSON and returns it as a tool result.
func docRecordMapJSON(v map[string]any) (*mcp.CallToolResult, error) {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return mcp.NewToolResultError("marshal result: " + err.Error()), nil
	}
	return mcp.NewToolResultText(string(data)), nil
}
