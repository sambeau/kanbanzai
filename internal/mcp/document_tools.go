package mcp

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"kanbanzai/internal/document"
)

// docTypeEnum is the list of valid document type values for MCP tool parameters.
var docTypeEnum = []string{
	string(document.DocTypeProposal),
	string(document.DocTypeResearchReport),
	string(document.DocTypeDraftDesign),
	string(document.DocTypeDesign),
	string(document.DocTypeSpecification),
	string(document.DocTypeImplementationPlan),
	string(document.DocTypeUserDocumentation),
}

// DocumentTools returns all document-related MCP tool definitions with their handlers.
func DocumentTools(svc *document.DocService) []server.ServerTool {
	return []server.ServerTool{
		scaffoldDocumentTool(svc),
		submitDocumentTool(svc),
		updateDocumentBodyTool(svc),
		approveDocumentTool(svc),
		retrieveDocumentTool(svc),
		listDocumentsTool(svc),
		validateDocumentTool(svc),
	}
}

func scaffoldDocumentTool(svc *document.DocService) server.ServerTool {
	tool := mcp.NewTool("scaffold_document",
		mcp.WithDescription("Generate a starter document from a template. Returns the scaffolded markdown content (not yet stored)."),
		mcp.WithString("doc_type",
			mcp.Description("Document type"),
			mcp.Required(),
			mcp.Enum(docTypeEnum...),
		),
		mcp.WithString("title",
			mcp.Description("Document title"),
			mcp.Required(),
		),
	)

	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		docType, err := request.RequireString("doc_type")
		if err != nil {
			return mcp.NewToolResultError("doc_type is required"), nil
		}
		title, err := request.RequireString("title")
		if err != nil {
			return mcp.NewToolResultError("title is required"), nil
		}

		content, err := svc.ScaffoldDocument(document.DocType(docType), title)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("scaffold failed", err), nil
		}

		return mcp.NewToolResultText(content), nil
	}

	return server.ServerTool{Tool: tool, Handler: handler}
}

func submitDocumentTool(svc *document.DocService) server.ServerTool {
	tool := mcp.NewTool("submit_document",
		mcp.WithDescription("Create and store a new document in submitted state."),
		mcp.WithString("doc_type",
			mcp.Description("Document type"),
			mcp.Required(),
			mcp.Enum(docTypeEnum...),
		),
		mcp.WithString("title",
			mcp.Description("Document title"),
			mcp.Required(),
		),
		mcp.WithString("body",
			mcp.Description("Document body content"),
			mcp.Required(),
		),
		mcp.WithString("created_by",
			mcp.Description("Author of the document"),
			mcp.Required(),
		),
		mcp.WithString("feature",
			mcp.Description("Feature reference (required for design, specification, and implementation-plan types)"),
		),
	)

	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		docType, err := request.RequireString("doc_type")
		if err != nil {
			return mcp.NewToolResultError("doc_type is required"), nil
		}
		title, err := request.RequireString("title")
		if err != nil {
			return mcp.NewToolResultError("title is required"), nil
		}
		body, err := request.RequireString("body")
		if err != nil {
			return mcp.NewToolResultError("body is required"), nil
		}
		createdBy, err := request.RequireString("created_by")
		if err != nil {
			return mcp.NewToolResultError("created_by is required"), nil
		}
		feature := request.GetString("feature", "")

		result, err := svc.Submit(document.SubmitInput{
			Type:      document.DocType(docType),
			Title:     title,
			Body:      body,
			CreatedBy: createdBy,
			Feature:   feature,
		})
		if err != nil {
			return mcp.NewToolResultErrorFromErr("submit failed", err), nil
		}

		return jsonResult(result)
	}

	return server.ServerTool{Tool: tool, Handler: handler}
}

func updateDocumentBodyTool(svc *document.DocService) server.ServerTool {
	tool := mcp.NewTool("update_document_body",
		mcp.WithDescription("Update the body of a submitted document and transition it to normalised state."),
		mcp.WithString("doc_type",
			mcp.Description("Document type"),
			mcp.Required(),
			mcp.Enum(docTypeEnum...),
		),
		mcp.WithString("id",
			mcp.Description("Document ID"),
			mcp.Required(),
		),
		mcp.WithString("body",
			mcp.Description("New document body content"),
			mcp.Required(),
		),
	)

	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		docType, err := request.RequireString("doc_type")
		if err != nil {
			return mcp.NewToolResultError("doc_type is required"), nil
		}
		id, err := request.RequireString("id")
		if err != nil {
			return mcp.NewToolResultError("id is required"), nil
		}
		body, err := request.RequireString("body")
		if err != nil {
			return mcp.NewToolResultError("body is required"), nil
		}

		result, err := svc.UpdateBody(document.DocType(docType), id, body)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("update body failed", err), nil
		}

		return jsonResult(result)
	}

	return server.ServerTool{Tool: tool, Handler: handler}
}

func approveDocumentTool(svc *document.DocService) server.ServerTool {
	tool := mcp.NewTool("approve_document",
		mcp.WithDescription("Approve a normalised document, transitioning it to approved state."),
		mcp.WithString("doc_type",
			mcp.Description("Document type"),
			mcp.Required(),
			mcp.Enum(docTypeEnum...),
		),
		mcp.WithString("id",
			mcp.Description("Document ID"),
			mcp.Required(),
		),
		mcp.WithString("approved_by",
			mcp.Description("Name of the approver"),
			mcp.Required(),
		),
	)

	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		docType, err := request.RequireString("doc_type")
		if err != nil {
			return mcp.NewToolResultError("doc_type is required"), nil
		}
		id, err := request.RequireString("id")
		if err != nil {
			return mcp.NewToolResultError("id is required"), nil
		}
		approvedBy, err := request.RequireString("approved_by")
		if err != nil {
			return mcp.NewToolResultError("approved_by is required"), nil
		}

		result, err := svc.Approve(document.ApproveInput{
			Type:       document.DocType(docType),
			ID:         id,
			ApprovedBy: approvedBy,
		})
		if err != nil {
			return mcp.NewToolResultErrorFromErr("approve failed", err), nil
		}

		return jsonResult(result)
	}

	return server.ServerTool{Tool: tool, Handler: handler}
}

func retrieveDocumentTool(svc *document.DocService) server.ServerTool {
	tool := mcp.NewTool("retrieve_document",
		mcp.WithDescription("Retrieve a document by type and ID. Returns the document body as plain text."),
		mcp.WithString("doc_type",
			mcp.Description("Document type"),
			mcp.Required(),
			mcp.Enum(docTypeEnum...),
		),
		mcp.WithString("id",
			mcp.Description("Document ID"),
			mcp.Required(),
		),
	)

	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		docType, err := request.RequireString("doc_type")
		if err != nil {
			return mcp.NewToolResultError("doc_type is required"), nil
		}
		id, err := request.RequireString("id")
		if err != nil {
			return mcp.NewToolResultError("id is required"), nil
		}

		doc, err := svc.Retrieve(document.DocType(docType), id)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("retrieve failed", err), nil
		}

		return mcp.NewToolResultText(doc.Body), nil
	}

	return server.ServerTool{Tool: tool, Handler: handler}
}

func listDocumentsTool(svc *document.DocService) server.ServerTool {
	tool := mcp.NewTool("list_documents",
		mcp.WithDescription("List documents. If doc_type is provided, lists only that type; otherwise lists all documents."),
		mcp.WithString("doc_type",
			mcp.Description("Document type to filter by (omit to list all)"),
			mcp.Enum(docTypeEnum...),
		),
	)

	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		docType := request.GetString("doc_type", "")

		var results []document.DocumentResult
		var err error

		if docType != "" {
			results, err = svc.ListByType(document.DocType(docType))
		} else {
			results, err = svc.ListAll()
		}
		if err != nil {
			return mcp.NewToolResultErrorFromErr("list failed", err), nil
		}

		if results == nil {
			results = []document.DocumentResult{}
		}

		return jsonResult(results)
	}

	return server.ServerTool{Tool: tool, Handler: handler}
}

func validateDocumentTool(svc *document.DocService) server.ServerTool {
	tool := mcp.NewTool("validate_document",
		mcp.WithDescription("Retrieve and validate a document against its template. Returns validation errors (empty array if valid)."),
		mcp.WithString("doc_type",
			mcp.Description("Document type"),
			mcp.Required(),
			mcp.Enum(docTypeEnum...),
		),
		mcp.WithString("id",
			mcp.Description("Document ID"),
			mcp.Required(),
		),
	)

	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		docType, err := request.RequireString("doc_type")
		if err != nil {
			return mcp.NewToolResultError("doc_type is required"), nil
		}
		id, err := request.RequireString("id")
		if err != nil {
			return mcp.NewToolResultError("id is required"), nil
		}

		doc, err := svc.Retrieve(document.DocType(docType), id)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("retrieve failed", err), nil
		}

		validationErrors := svc.Validate(doc)
		if validationErrors == nil {
			validationErrors = []document.ValidationError{}
		}

		return jsonResult(validationErrors)
	}

	return server.ServerTool{Tool: tool, Handler: handler}
}
