// Package mcp doc_tool.go — consolidated document operations for Kanbanzai 2.0 (Track I).
//
// doc(action, ...) replaces 11+ document record tools with a single
// resource-oriented interface:
//
//	doc(action: "register", path: "work/spec/foo.md", type: "specification", title: "Foo Spec")
//	doc(action: "approve", id: "DOC-01JX...")
//	doc(action: "get", id: "DOC-01JX...")
//	doc(action: "get", path: "work/spec/foo.md")
//	doc(action: "content", id: "DOC-01JX...")
//	doc(action: "list", status: "draft")
//	doc(action: "gaps", feature_id: "FEAT-01JA...")
//	doc(action: "validate", id: "DOC-01JX...")
//	doc(action: "supersede", id: "DOC-01JX...", superseded_by: "DOC-02JX...")
//	doc(action: "refresh", id: "DOC-01JX...")
//	doc(action: "chain", id: "DOC-01JX...")
//	doc(action: "import", path: "work/")
//
// approve and supersede push SideEffectStatusTransition side effects when
// the operation cascades a feature lifecycle transition (spec §30.2 criterion 7).
// register supports batch via the documents array; approve supports batch via ids.
//
// intelligenceSvc is held by DocTool for future actions that need document
// classification data (e.g., enriched gap analysis via Layer 3 concepts). All
// current actions rely on docSvc directly and do not need it.
package mcp

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"kanbanzai/internal/config"
	"kanbanzai/internal/service"
)

// DocTool returns the consolidated `doc` MCP tool registered in the core group.
//
// intelligenceSvc is retained in the signature so callers (server.go) do not
// need to change when a future action makes use of it. Currently all actions
// call into docSvc directly.
func DocTool(docSvc *service.DocumentService, intelligenceSvc *service.IntelligenceService) []server.ServerTool {
	// intelligenceSvc is intentionally not forwarded to docTool; no current action needs it.
	_ = intelligenceSvc
	return []server.ServerTool{docTool(docSvc)}
}

func docTool(docSvc *service.DocumentService) server.ServerTool {
	tool := mcp.NewTool("doc",
		mcp.WithReadOnlyHintAnnotation(false),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(false),
		mcp.WithOpenWorldHintAnnotation(false),
		mcp.WithTitleAnnotation("Document Records"),
		mcp.WithDescription(
			"Manage document records: register, approve, query, supersede, refresh, chain, validate, and import. "+
				"Use action: get or action: list to query document status — do not read .kbz/state/documents/ files directly. "+
				"Actions: register, approve, get, content, list, gaps, validate, supersede, refresh, chain, import. "+
				"approve and supersede report entity lifecycle cascade side effects.",
		),
		mcp.WithString("action",
			mcp.Required(),
			mcp.Description("Action: register, approve, get, content, list, gaps, validate, supersede, refresh, chain, import"),
		),
		// Common identifier fields.
		mcp.WithString("id", mcp.Description("Document record ID (approve, get, content, validate, supersede, refresh, chain)")),
		mcp.WithArray("ids", mcp.Description("Batch approve: array of document record IDs")),
		// register fields.
		mcp.WithString("path", mcp.Description("Document file path (register: required; get: alternative to id; import: directory to scan)")),
		mcp.WithString("type", mcp.Description("Document type: design, specification, dev-plan, research, report, policy (register, list)")),
		mcp.WithString("title", mcp.Description("Human-readable title (register)")),
		mcp.WithString("owner", mcp.Description("Parent Plan or Feature ID (register, list, import)")),
		mcp.WithArray("documents", mcp.Description("Batch register: array of {path, type, title, owner?} objects")),
		// supersede fields.
		mcp.WithString("superseded_by", mcp.Description("Replacement document record ID (supersede)")),
		// list fields.
		mcp.WithString("status", mcp.Description("Status filter: draft, approved, superseded (list)")),
		mcp.WithBoolean("pending", mcp.Description("If true, list only draft documents awaiting approval — shorthand for status: draft (list)")),
		// gaps fields.
		mcp.WithString("feature_id", mcp.Description("Feature ID to analyse for document gaps (gaps)")),
		// import fields.
		mcp.WithString("glob", mcp.Description("File pattern filter (import)")),
		mcp.WithString("default_type", mcp.Description("Fallback document type when no path pattern matches (import)")),
		// Identity.
		mcp.WithString("created_by", mcp.Description("Who is performing the operation. Auto-resolved from .kbz/local.yaml or git config if not provided.")),
	)

	handler := WithSideEffects(func(ctx context.Context, req mcp.CallToolRequest) (any, error) {
		return DispatchAction(ctx, req, map[string]ActionHandler{
			"register":  docRegisterAction(docSvc),
			"approve":   docApproveAction(docSvc),
			"get":       docGetAction(docSvc),
			"content":   docContentAction(docSvc),
			"list":      docListAction(docSvc),
			"gaps":      docGapsAction(docSvc),
			"validate":  docValidateAction(docSvc),
			"supersede": docSupersedeAction(docSvc),
			"refresh":   docRefreshAction(docSvc),
			"chain":     docChainAction(docSvc),
			"import":    docImportAction(docSvc),
		})
	})

	return server.ServerTool{Tool: tool, Handler: handler}
}

// ─── register ─────────────────────────────────────────────────────────────────

func docRegisterAction(docSvc *service.DocumentService) ActionHandler {
	return func(ctx context.Context, req mcp.CallToolRequest) (any, error) {
		SignalMutation(ctx)
		args, _ := req.Params.Arguments.(map[string]any)

		// Batch path: documents array of {path, type, title, owner?} objects.
		if IsBatchInput(args, "documents") {
			items, _ := args["documents"].([]any)
			topCreatedBy, _ := args["created_by"].(string)
			return ExecuteBatch(ctx, items, func(ctx context.Context, item any) (string, any, error) {
				doc, ok := item.(map[string]any)
				if !ok {
					return "", nil, fmt.Errorf("each item in documents must be an object with path, type, and title")
				}
				// Inherit top-level created_by when not set per item.
				if _, has := doc["created_by"]; !has && topCreatedBy != "" {
					doc["created_by"] = topCreatedBy
				}
				return docRegisterOne(docSvc, doc)
			})
		}

		// Single path.
		if docArgStr(args, "path") == "" {
			return nil, fmt.Errorf("path is required for register")
		}
		_, result, err := docRegisterOne(docSvc, args)
		return result, err
	}
}

func docRegisterOne(docSvc *service.DocumentService, args map[string]any) (string, any, error) {
	path := docArgStr(args, "path")
	docType := docArgStr(args, "type")
	title := docArgStr(args, "title")
	owner := docArgStr(args, "owner")
	createdByRaw := docArgStr(args, "created_by")

	if path == "" {
		return path, nil, fmt.Errorf("path is required")
	}
	if docType == "" {
		return path, nil, fmt.Errorf("type is required")
	}
	if title == "" {
		return path, nil, fmt.Errorf("title is required")
	}

	createdBy, err := config.ResolveIdentity(createdByRaw)
	if err != nil {
		return path, nil, err
	}

	result, err := docSvc.SubmitDocument(service.SubmitDocumentInput{
		Path:      path,
		Type:      docType,
		Title:     title,
		Owner:     owner,
		CreatedBy: createdBy,
	})
	if err != nil {
		return path, nil, err
	}

	return result.ID, map[string]any{"document": docRecordToMap(result)}, nil
}

// ─── approve ──────────────────────────────────────────────────────────────────

func docApproveAction(docSvc *service.DocumentService) ActionHandler {
	return func(ctx context.Context, req mcp.CallToolRequest) (any, error) {
		SignalMutation(ctx)
		args, _ := req.Params.Arguments.(map[string]any)
		approvedByRaw := docArgStr(args, "created_by")

		// Batch path: ids array of document record ID strings.
		if IsBatchInput(args, "ids") {
			items, _ := args["ids"].([]any)
			return ExecuteBatch(ctx, items, func(ctx context.Context, item any) (string, any, error) {
				docID, _ := item.(string)
				return docApproveOne(ctx, docSvc, docID, approvedByRaw)
			})
		}

		// Single path.
		docID := docArgStr(args, "id")
		if docID == "" {
			return nil, fmt.Errorf("id is required for approve")
		}
		_, result, err := docApproveOne(ctx, docSvc, docID, approvedByRaw)
		return result, err
	}
}

func docApproveOne(ctx context.Context, docSvc *service.DocumentService, docID, approvedByRaw string) (string, any, error) {
	approvedBy, err := config.ResolveIdentity(approvedByRaw)
	if err != nil {
		return docID, nil, err
	}

	result, err := docSvc.ApproveDocument(service.ApproveDocumentInput{
		ID:         docID,
		ApprovedBy: approvedBy,
	})
	if err != nil {
		return docID, nil, err
	}

	// Report entity lifecycle cascade triggered by document approval (spec §30.2 criterion 7).
	if t := result.EntityTransition; t != nil {
		PushSideEffect(ctx, SideEffect{
			Type:       SideEffectStatusTransition,
			EntityID:   t.EntityID,
			EntityType: t.EntityType,
			FromStatus: t.FromStatus,
			ToStatus:   t.ToStatus,
			Trigger:    fmt.Sprintf("Document %s approved", result.ID),
		})
	}

	return result.ID, map[string]any{"document": docRecordToMap(result)}, nil
}

// ─── get ──────────────────────────────────────────────────────────────────────

func docGetAction(docSvc *service.DocumentService) ActionHandler {
	return func(ctx context.Context, req mcp.CallToolRequest) (any, error) {
		args, _ := req.Params.Arguments.(map[string]any)
		docID := docArgStr(args, "id")
		path := docArgStr(args, "path")

		if docID == "" && path == "" {
			return nil, fmt.Errorf("either id or path is required for get")
		}

		// Path-based lookup: scan all records for one whose path matches.
		if docID == "" {
			all, err := docSvc.ListDocuments(service.DocumentFilters{})
			if err != nil {
				return nil, fmt.Errorf("resolve path: %w", err)
			}
			for _, d := range all {
				if d.Path == path {
					docID = d.ID
					break
				}
			}
			if docID == "" {
				return nil, fmt.Errorf("no document found at path %q", path)
			}
		}

		result, err := docSvc.GetDocument(docID, true)
		if err != nil {
			return nil, err
		}

		out := map[string]any{"document": docRecordToMap(result)}
		if result.Drift {
			out["drift"] = true
			out["current_hash"] = result.CurrentHash
			out["warning"] = "Document content has changed since recorded"
		}
		return out, nil
	}
}

// ─── content ──────────────────────────────────────────────────────────────────

func docContentAction(docSvc *service.DocumentService) ActionHandler {
	return func(ctx context.Context, req mcp.CallToolRequest) (any, error) {
		args, _ := req.Params.Arguments.(map[string]any)
		docID := docArgStr(args, "id")
		if docID == "" {
			return nil, fmt.Errorf("id is required for content")
		}

		content, result, err := docSvc.GetDocumentContent(docID)
		if err != nil {
			return nil, err
		}

		out := map[string]any{
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
			out["drift"] = true
			out["current_hash"] = result.CurrentHash
			out["warning"] = "Document content has changed since recorded"
		}
		return out, nil
	}
}

// ─── list ─────────────────────────────────────────────────────────────────────

func docListAction(docSvc *service.DocumentService) ActionHandler {
	return func(ctx context.Context, req mcp.CallToolRequest) (any, error) {
		args, _ := req.Params.Arguments.(map[string]any)

		filters := service.DocumentFilters{
			Type:  docArgStr(args, "type"),
			Owner: docArgStr(args, "owner"),
		}
		// pending: true is shorthand for status: "draft" (spec §15.7).
		if pending, _ := args["pending"].(bool); pending {
			filters.Status = "draft"
		} else {
			filters.Status = docArgStr(args, "status")
		}

		results, err := docSvc.ListDocuments(filters)
		if err != nil {
			return nil, err
		}

		docs := make([]map[string]any, 0, len(results))
		for _, r := range results {
			docs = append(docs, docRecordToMap(r))
		}
		return map[string]any{
			"documents": docs,
			"total":     len(results),
		}, nil
	}
}

// ─── gaps ─────────────────────────────────────────────────────────────────────

func docGapsAction(docSvc *service.DocumentService) ActionHandler {
	return func(ctx context.Context, req mcp.CallToolRequest) (any, error) {
		args, _ := req.Params.Arguments.(map[string]any)
		featureID := docArgStr(args, "feature_id")
		if featureID == "" {
			return nil, fmt.Errorf("feature_id is required for gaps")
		}

		owned, err := docSvc.ListDocumentsByOwner(featureID)
		if err != nil {
			return nil, fmt.Errorf("list documents for %s: %w", featureID, err)
		}

		// Build lookup: type → best record (approved beats draft).
		byType := make(map[string]service.DocumentResult)
		for _, d := range owned {
			existing, found := byType[d.Type]
			if !found || (d.Status == "approved" && existing.Status != "approved") {
				byType[d.Type] = d
			}
		}

		expected := []string{"design", "specification", "dev-plan"}
		gaps := make([]map[string]any, 0)
		present := make([]map[string]any, 0)

		for _, docType := range expected {
			d, found := byType[docType]
			if !found {
				gaps = append(gaps, map[string]any{
					"type":   docType,
					"status": "missing",
				})
			} else if d.Status == "approved" {
				present = append(present, map[string]any{
					"type":   docType,
					"status": d.Status,
					"id":     d.ID,
				})
			} else {
				// Exists but not yet approved — still a gap.
				gaps = append(gaps, map[string]any{
					"type":   docType,
					"status": d.Status,
					"id":     d.ID,
				})
			}
		}

		return map[string]any{
			"feature_id": featureID,
			"gaps":       gaps,
			"present":    present,
		}, nil
	}
}

// ─── validate ─────────────────────────────────────────────────────────────────

func docValidateAction(docSvc *service.DocumentService) ActionHandler {
	return func(ctx context.Context, req mcp.CallToolRequest) (any, error) {
		args, _ := req.Params.Arguments.(map[string]any)
		docID := docArgStr(args, "id")
		if docID == "" {
			return nil, fmt.Errorf("id is required for validate")
		}

		issues, err := docSvc.ValidateDocument(docID)
		if err != nil {
			return nil, err
		}

		if issues == nil {
			issues = []string{}
		}
		return map[string]any{
			"document_id": docID,
			"valid":       len(issues) == 0,
			"issues":      issues,
		}, nil
	}
}

// ─── supersede ────────────────────────────────────────────────────────────────

func docSupersedeAction(docSvc *service.DocumentService) ActionHandler {
	return func(ctx context.Context, req mcp.CallToolRequest) (any, error) {
		SignalMutation(ctx)
		args, _ := req.Params.Arguments.(map[string]any)
		docID := docArgStr(args, "id")
		supersededBy := docArgStr(args, "superseded_by")
		if docID == "" {
			return nil, fmt.Errorf("id is required for supersede")
		}
		if supersededBy == "" {
			return nil, fmt.Errorf("superseded_by is required for supersede")
		}

		result, err := docSvc.SupersedeDocument(service.SupersedeDocumentInput{
			ID:           docID,
			SupersededBy: supersededBy,
		})
		if err != nil {
			return nil, err
		}

		// Report entity lifecycle cascade triggered by supersession (spec §30.2 criterion 7).
		if t := result.EntityTransition; t != nil {
			PushSideEffect(ctx, SideEffect{
				Type:       SideEffectStatusTransition,
				EntityID:   t.EntityID,
				EntityType: t.EntityType,
				FromStatus: t.FromStatus,
				ToStatus:   t.ToStatus,
				Trigger:    fmt.Sprintf("Document %s superseded by %s", result.ID, supersededBy),
			})
		}

		return map[string]any{"document": docRecordToMap(result)}, nil
	}
}

// ─── refresh ──────────────────────────────────────────────────────────────────

func docRefreshAction(docSvc *service.DocumentService) ActionHandler {
	return func(ctx context.Context, req mcp.CallToolRequest) (any, error) {
		SignalMutation(ctx)
		args, _ := req.Params.Arguments.(map[string]any)
		id := docArgStr(args, "id")
		path := docArgStr(args, "path")
		result, err := docSvc.RefreshContentHash(service.RefreshInput{ID: id, Path: path})
		if err != nil {
			return nil, err
		}
		return map[string]any{
			"id":                result.ID,
			"path":              result.Path,
			"changed":           result.Changed,
			"old_hash":          result.OldHash,
			"new_hash":          result.NewHash,
			"status":            result.Status,
			"status_transition": result.StatusTransition,
		}, nil
	}
}

// ─── chain ────────────────────────────────────────────────────────────────────

func docChainAction(docSvc *service.DocumentService) ActionHandler {
	return func(ctx context.Context, req mcp.CallToolRequest) (any, error) {
		args, _ := req.Params.Arguments.(map[string]any)
		id := docArgStr(args, "id")
		if id == "" {
			return nil, fmt.Errorf("id is required for action: chain")
		}
		chain, err := docSvc.SupersessionChain(id)
		if err != nil {
			return nil, err
		}
		items := make([]map[string]any, len(chain))
		for i, doc := range chain {
			items[i] = map[string]any{
				"id":            doc.ID,
				"path":          doc.Path,
				"type":          doc.Type,
				"title":         doc.Title,
				"status":        doc.Status,
				"superseded_by": doc.SupersededBy,
			}
		}
		return map[string]any{
			"chain":  items,
			"length": len(chain),
		}, nil
	}
}

// ─── import ───────────────────────────────────────────────────────────────────

func docImportAction(docSvc *service.DocumentService) ActionHandler {
	return func(ctx context.Context, req mcp.CallToolRequest) (any, error) {
		SignalMutation(ctx)
		args, _ := req.Params.Arguments.(map[string]any)
		path := docArgStr(args, "path")
		if path == "" {
			return nil, fmt.Errorf("path is required for import")
		}

		createdByRaw := docArgStr(args, "created_by")
		createdBy, err := config.ResolveIdentity(createdByRaw)
		if err != nil {
			return nil, err
		}

		cfg := config.LoadOrDefault()
		importSvc := service.NewBatchImportService(docSvc)

		result, err := importSvc.Import(cfg, service.BatchImportInput{
			Path:        path,
			DefaultType: docArgStr(args, "default_type"),
			Owner:       docArgStr(args, "owner"),
			CreatedBy:   createdBy,
			Glob:        docArgStr(args, "glob"),
		})
		if err != nil {
			return nil, err
		}

		skipped := make([]map[string]any, 0, len(result.Skipped))
		for _, s := range result.Skipped {
			skipped = append(skipped, map[string]any{"path": s.Path, "reason": s.Reason})
		}
		errors := make([]map[string]any, 0, len(result.Errors))
		for _, e := range result.Errors {
			errors = append(errors, map[string]any{"path": e.Path, "error": e.Error})
		}

		return map[string]any{
			"imported": result.Imported,
			"skipped":  skipped,
			"errors":   errors,
		}, nil
	}
}

// ─── helpers ──────────────────────────────────────────────────────────────────

// docRecordToMap converts a DocumentResult to the canonical map shape used in
// doc tool responses. Optional fields are omitted when empty.
func docRecordToMap(r service.DocumentResult) map[string]any {
	m := map[string]any{
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
		m["approved_by"] = r.ApprovedBy
	}
	if r.ApprovedAt != nil {
		m["approved_at"] = r.ApprovedAt
	}
	if r.Supersedes != "" {
		m["supersedes"] = r.Supersedes
	}
	if r.SupersededBy != "" {
		m["superseded_by"] = r.SupersededBy
	}
	return m
}

// docArgStr safely extracts a string from an args map.
func docArgStr(args map[string]any, key string) string {
	v, _ := args[key].(string)
	return v
}
