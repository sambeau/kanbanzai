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
	"log"
	"strconv"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/sambeau/kanbanzai/internal/config"
	"github.com/sambeau/kanbanzai/internal/git"
	"github.com/sambeau/kanbanzai/internal/model"
	"github.com/sambeau/kanbanzai/internal/service"
	"github.com/sambeau/kanbanzai/internal/structural"
)

// docCommitFunc is the function called after doc register and approve to commit
// state changes with a custom message. Package-level variable for test injection.
// Production value delegates to git.CommitStateWithMessage (FR-A12, FR-B01).
var docCommitFunc = func(repoRoot, message string) (bool, error) {
	return git.CommitStateWithMessage(repoRoot, message)
}

// docCommitPathsFunc is the function called by doc register, move, and delete
// to commit state changes plus extra file paths atomically.
// Package-level variable for test injection. (FR-B01, FR-B13, FR-B19).
var docCommitPathsFunc = func(repoRoot, message string, extraPaths ...string) (bool, error) {
	return git.CommitStateAndPaths(repoRoot, message, extraPaths...)
}

// DocTool returns the consolidated `doc` MCP tool registered in the core group.
//
// intelligenceSvc is retained in the signature so callers (server.go) do not
// need to change when a future action makes use of it. Currently all actions
// call into docSvc directly.
func DocTool(docSvc *service.DocumentService, intelligenceSvc *service.IntelligenceService, entitySvc *service.EntityService) []server.ServerTool {
	// intelligenceSvc is intentionally not forwarded to docTool; no current action needs it.
	_ = intelligenceSvc
	return []server.ServerTool{docTool(docSvc, entitySvc)}
}

func docTool(docSvc *service.DocumentService, entitySvc *service.EntityService) server.ServerTool {
	tool := mcp.NewTool("doc",
		mcp.WithReadOnlyHintAnnotation(false),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(false),
		mcp.WithOpenWorldHintAnnotation(false),
		mcp.WithTitleAnnotation("Document Records"),
		mcp.WithDescription(
			"Use when you need to register, approve, query, or manage document records tracking specs, "+
				"designs, and plans. Use INSTEAD OF reading .kbz/state/documents/ files directly — "+
				"get and list return structured metadata with approval status. "+
				"Do NOT use for content analysis — use doc_intel instead. "+
				"Actions: register, approve, get, content, list, gaps, validate, supersede, refresh, chain, import, audit, evaluate, record_false_positive, move, delete. "+
				"For register: path, type, title required. For approve/get/content/validate/supersede/refresh/chain: "+
				"id required (or ids for batch approve). "+
				"Call register after writing a document, approve before advancing past a stage gate. "+
				"approve and supersede report entity lifecycle cascade side effects. "+
				"For record_false_positive: id and description required.",
		),
		mcp.WithString("action",
			mcp.Required(),
			mcp.Description("Action: register, approve, get, content, list, gaps, validate, supersede, refresh, chain, import, audit, evaluate, record_false_positive, move, delete"),
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
		mcp.WithBoolean("auto_approve", mcp.Description("When true, registers and approves the document in one call. Permitted for: dev-plan, research, report (register only)")),
		// move fields.
		mcp.WithString("new_path", mcp.Description("New relative file path for the document (move only)")),
		// delete fields.
		mcp.WithBoolean("force", mcp.Description("When true, allows deletion of approved documents (delete only)")),
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
		mcp.WithBoolean("dry_run", mcp.Description("Preview what would be registered without writing to the store (import)")),
		// audit fields.
		mcp.WithBoolean("include_registered", mcp.Description("Include full registered file list in response (audit)")),
		// Identity.
		mcp.WithString("created_by", mcp.Description("Who is performing the operation. Auto-resolved from .kbz/local.yaml or git config if not provided.")),
		// evaluate fields.
		mcp.WithObject("evaluation", mcp.Description("Quality evaluation object for evaluate action: {overall_score, pass, evaluated_at, evaluator, dimensions}")),
		// record_false_positive fields.
		mcp.WithString("description", mcp.Description("False positive description for record_false_positive action")),
		mcp.WithString("check_type", mcp.Description("Specific check type for record_false_positive (optional; omit to apply to all check types)")),
	)

	handler := WithSideEffects(func(ctx context.Context, req mcp.CallToolRequest) (any, error) {
		return DispatchAction(ctx, req, map[string]ActionHandler{
			"register":              docRegisterAction(docSvc),
			"approve":               docApproveAction(docSvc),
			"move":                  docMoveAction(docSvc),
			"delete":                docDeleteAction(docSvc),
			"get":                   docGetAction(docSvc),
			"content":               docContentAction(docSvc),
			"list":                  docListAction(docSvc),
			"gaps":                  docGapsAction(docSvc, entitySvc),
			"validate":              docValidateAction(docSvc),
			"supersede":             docSupersedeAction(docSvc),
			"refresh":               docRefreshAction(docSvc),
			"chain":                 docChainAction(docSvc),
			"import":                docImportAction(docSvc),
			"audit":                 docAuditAction(docSvc),
			"evaluate":              docEvaluateAction(docSvc),
			"record_false_positive": docRecordFalsePositiveAction(docSvc),
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
					return "", nil, fmt.Errorf("Cannot register document: each item in the documents array must be an object with path, type, and title.\n\nTo resolve:\n  Provide documents as [{\"path\": \"...\", \"type\": \"...\", \"title\": \"...\"}]")
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
			return nil, fmt.Errorf("Cannot register document: path is missing.\n\nTo resolve:\n  Provide path: doc(action: \"register\", path: \"work/spec/foo.md\", type: \"...\", title: \"...\")")
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
		return path, nil, fmt.Errorf("Cannot register document: path is missing.\n\nTo resolve:\n  Provide path: doc(action: \"register\", path: \"work/spec/foo.md\", type: \"...\", title: \"...\")")
	}
	if docType == "" {
		return path, nil, fmt.Errorf("Cannot register document %q: type is missing.\n\nTo resolve:\n  Add the type parameter (design, specification, dev-plan, research, report, policy).", path)
	}
	if title == "" {
		return path, nil, fmt.Errorf("Cannot register document %q: title is missing.\n\nTo resolve:\n  Add a human-readable title parameter.", path)
	}

	createdBy, err := config.ResolveIdentity(createdByRaw)
	if err != nil {
		return path, nil, err
	}

	autoApprove, _ := args["auto_approve"].(bool)

	result, err := docSvc.SubmitDocument(service.SubmitDocumentInput{
		Path:        path,
		Type:        docType,
		Title:       title,
		Owner:       owner,
		CreatedBy:   createdBy,
		AutoApprove: autoApprove,
	})
	if err != nil {
		return path, nil, err
	}

	// Auto-commit: stage both the state record and the document file atomically (FR-B01).
	// Best-effort: commit failure is logged but does not prevent the result.
	repoRoot := docSvc.RepoRoot()
	registerMsg := fmt.Sprintf("workflow(%s): register %s", result.ID, result.Type)
	if _, commitErr := docCommitPathsFunc(repoRoot, registerMsg, path); commitErr != nil {
		log.Printf("[doc] WARNING: auto-commit after register %s failed: %v", result.ID, commitErr)
	}

	out := map[string]any{"document": docRecordToMap(result)}
	if len(result.Warnings) > 0 {
		out["warnings"] = result.Warnings
	}
	return result.ID, out, nil
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
			return nil, fmt.Errorf("Cannot approve document: id is missing.\n\nTo resolve:\n  Provide the document record ID: doc(action: \"approve\", id: \"DOC-...\")")
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

	// Auto-commit the approved document's state record (FR-A12). Best-effort.
	repoRoot := docSvc.RepoRoot()
	approveMsg := fmt.Sprintf("workflow(%s): approve %s", result.ID, result.Type)
	if _, commitErr := docCommitFunc(repoRoot, approveMsg); commitErr != nil {
		log.Printf("[doc] WARNING: auto-commit after approve %s failed: %v", result.ID, commitErr)
	}

	return result.ID, map[string]any{"document": docRecordToMap(result)}, nil
}

// ─── move ─────────────────────────────────────────────────────────────────────

func docMoveAction(docSvc *service.DocumentService) ActionHandler {
	return func(ctx context.Context, req mcp.CallToolRequest) (any, error) {
		SignalMutation(ctx)
		args, _ := req.Params.Arguments.(map[string]any)

		docID := docArgStr(args, "id")
		if docID == "" {
			return nil, fmt.Errorf("Cannot move document: id is missing.\n\nTo resolve:\n  Provide the document record ID: doc(action: \"move\", id: \"DOC-...\", new_path: \"work/...\")")
		}
		newPath := docArgStr(args, "new_path")
		if newPath == "" {
			return nil, fmt.Errorf("Cannot move document %s: new_path is missing.\n\nTo resolve:\n  Provide the destination path: doc(action: \"move\", id: \"%s\", new_path: \"work/...\")", docID, docID)
		}

		// Load the old path before moving so we can include it in the commit.
		oldDoc, err := docSvc.GetDocument(docID, false)
		if err != nil {
			return nil, fmt.Errorf("Cannot move document %s: %w", docID, err)
		}
		oldPath := oldDoc.Path

		result, err := docSvc.MoveDocument(service.MoveDocumentInput{
			ID:      docID,
			NewPath: newPath,
		})
		if err != nil {
			return nil, fmt.Errorf("Cannot move document %s: %w", docID, err)
		}

		// Commit state record + old path (deletion) + new path (addition) atomically (FR-B13).
		repoRoot := docSvc.RepoRoot()
		moveMsg := fmt.Sprintf("workflow(%s): move to %s", docID, newPath)
		if _, commitErr := docCommitPathsFunc(repoRoot, moveMsg, oldPath, newPath); commitErr != nil {
			log.Printf("[doc] WARNING: auto-commit after move %s failed: %v", docID, commitErr)
		}

		return map[string]any{"document": docRecordToMap(result)}, nil
	}
}

// ─── delete ───────────────────────────────────────────────────────────────────

func docDeleteAction(docSvc *service.DocumentService) ActionHandler {
	return func(ctx context.Context, req mcp.CallToolRequest) (any, error) {
		SignalMutation(ctx)
		args, _ := req.Params.Arguments.(map[string]any)

		docID := docArgStr(args, "id")
		if docID == "" {
			return nil, fmt.Errorf("Cannot delete document: id is missing.\n\nTo resolve:\n  Provide the document record ID: doc(action: \"delete\", id: \"DOC-...\")")
		}
		force, _ := args["force"].(bool)

		// Load the document path before deletion so we can include it in the commit.
		oldDoc, err := docSvc.GetDocument(docID, false)
		if err != nil {
			return nil, fmt.Errorf("Cannot delete document %s: %w", docID, err)
		}
		filePath := oldDoc.Path

		result, err := docSvc.DeleteDocument(service.DeleteDocumentInput{
			ID:    docID,
			Force: force,
		})
		if err != nil {
			return nil, fmt.Errorf("Cannot delete document %s: %w", docID, err)
		}

		// Commit state record removal + document file removal atomically (FR-B19).
		repoRoot := docSvc.RepoRoot()
		deleteMsg := fmt.Sprintf("workflow(%s): delete %s", docID, result.Type)
		if _, commitErr := docCommitPathsFunc(repoRoot, deleteMsg, filePath); commitErr != nil {
			log.Printf("[doc] WARNING: auto-commit after delete %s failed: %v", docID, commitErr)
		}

		return map[string]any{
			"deleted": map[string]any{
				"id":    result.ID,
				"path":  result.Path,
				"type":  result.Type,
				"title": result.Title,
				"owner": result.Owner,
			},
		}, nil
	}
}

// ─── get ──────────────────────────────────────────────────────────────────────

func docGetAction(docSvc *service.DocumentService) ActionHandler {
	return func(ctx context.Context, req mcp.CallToolRequest) (any, error) {
		args, _ := req.Params.Arguments.(map[string]any)
		docID := docArgStr(args, "id")
		path := docArgStr(args, "path")

		if docID == "" && path == "" {
			return nil, fmt.Errorf("Cannot get document: neither id nor path was provided.\n\nTo resolve:\n  Provide id or path: doc(action: \"get\", id: \"DOC-...\") or doc(action: \"get\", path: \"work/...\")")
		}

		// Path-based lookup: scan all records for one whose path matches.
		if docID == "" {
			all, err := docSvc.ListDocuments(service.DocumentFilters{})
			if err != nil {
				return nil, fmt.Errorf("Cannot get document by path %q: document listing failed.\n\nTo resolve:\n  Verify the path is correct, or use id instead: doc(action: \"get\", id: \"DOC-...\")\n\nCause: %w", path, err)
			}
			for _, d := range all {
				if d.Path == path {
					docID = d.ID
					break
				}
			}
			if docID == "" {
				return nil, fmt.Errorf("Cannot get document: no document found at path %q.\n\nTo resolve:\n  Check the path is correct, or register it first: doc(action: \"register\", path: \"...\", type: \"...\", title: \"...\")", path)
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
			return nil, fmt.Errorf("Cannot retrieve document content: id is missing.\n\nTo resolve:\n  Provide the document record ID: doc(action: \"content\", id: \"DOC-...\")")
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

func docGapsAction(docSvc *service.DocumentService, entitySvc *service.EntityService) ActionHandler {
	return func(ctx context.Context, req mcp.CallToolRequest) (any, error) {
		args, _ := req.Params.Arguments.(map[string]any)
		featureID := docArgStr(args, "feature_id")
		if featureID == "" {
			return nil, fmt.Errorf("Cannot analyse document gaps: feature_id is missing.\n\nTo resolve:\n  Provide the feature ID: doc(action: \"gaps\", feature_id: \"FEAT-...\")")
		}

		owned, err := docSvc.ListDocumentsByOwner(featureID)
		if err != nil {
			return nil, fmt.Errorf("Cannot analyse document gaps for %s: failed to list owned documents.\n\nTo resolve:\n  Verify the feature ID exists and is correct.\n\nCause: %w", featureID, err)
		}

		// Build lookup: type → best record (approved beats draft).
		byType := make(map[string]service.DocumentResult)
		for _, d := range owned {
			existing, found := byType[d.Type]
			if !found || (d.Status == "approved" && existing.Status != "approved") {
				byType[d.Type] = d
			}
		}

		// Get parent plan ID for inheritance fallback.
		var planID string
		if entitySvc != nil {
			if feat, err := entitySvc.Get("feature", featureID, ""); err == nil {
				planID, _ = feat.State["parent"].(string)
			}
		}

		// Load plan-level approved docs for fallback.
		var planByType map[string]service.DocumentResult
		if planID != "" && docSvc != nil {
			planDocs, _ := docSvc.ListDocumentsByOwner(planID)
			planByType = make(map[string]service.DocumentResult)
			for _, d := range planDocs {
				if d.Status == "approved" {
					if _, exists := planByType[d.Type]; !exists {
						planByType[d.Type] = d
					}
				}
			}
		}

		expected := []string{"design", "specification", "dev-plan"}
		gaps := make([]map[string]any, 0)
		present := make([]map[string]any, 0)

		for _, docType := range expected {
			d, found := byType[docType]
			if !found {
				// Try inheritance from plan.
				if pd, inherited := planByType[docType]; inherited {
					present = append(present, map[string]any{
						"type":      docType,
						"status":    pd.Status,
						"id":        pd.ID,
						"inherited": true,
					})
				} else {
					gaps = append(gaps, map[string]any{
						"type":   docType,
						"status": "missing",
					})
				}
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
			return nil, fmt.Errorf("Cannot validate document: id is missing.\n\nTo resolve:\n  Provide the document record ID: doc(action: \"validate\", id: \"DOC-...\")")
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

// ─── record_false_positive ────────────────────────────────────────────────────

func docRecordFalsePositiveAction(docSvc *service.DocumentService) ActionHandler {
	return func(ctx context.Context, req mcp.CallToolRequest) (any, error) {
		SignalMutation(ctx)
		args, _ := req.Params.Arguments.(map[string]any)
		docID := docArgStr(args, "id")
		description := docArgStr(args, "description")
		checkType := docArgStr(args, "check_type")

		if docID == "" {
			return nil, fmt.Errorf("Cannot record false positive: id is missing.\n\nTo resolve:\n  Provide the document record ID: doc(action: \"record_false_positive\", id: \"DOC-...\", description: \"...\")")
		}
		if description == "" {
			return nil, fmt.Errorf("Cannot record false positive: description is missing.\n\nTo resolve:\n  Provide a description explaining the false positive: doc(action: \"record_false_positive\", id: \"DOC-...\", description: \"...\")")
		}

		doc, err := docSvc.GetDocument(docID, false)
		if err != nil {
			return nil, fmt.Errorf("Cannot record false positive: document %q not found.\n\nTo resolve:\n  Verify the document ID with doc(action: \"get\", id: \"DOC-...\")", docID)
		}

		ps, err := structural.LoadPromotionState(docSvc.StateRoot())
		if err != nil {
			return nil, fmt.Errorf("Cannot record false positive: failed to load promotion state.\n\nTo resolve:\n  Check the state root directory is accessible: %v", err)
		}

		knownCheckTypes := []string{"required_sections", "cross_reference", "acceptance_criteria"}
		var targets []string
		if checkType != "" {
			targets = []string{checkType}
		} else {
			targets = knownCheckTypes
		}

		var affected []string
		for _, ct := range targets {
			key := structural.CheckKey{CheckType: ct, DocumentType: doc.Type}
			ps.RecordFalsePositive(key, description)
			affected = append(affected, ct+"/"+doc.Type)
		}

		if err := ps.Save(); err != nil {
			return nil, fmt.Errorf("Cannot record false positive: failed to save promotion state.\n\nTo resolve:\n  Check the state root directory is writable: %v", err)
		}

		return map[string]any{
			"document_id":   docID,
			"document_type": doc.Type,
			"description":   description,
			"affected_keys": affected,
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
			return nil, fmt.Errorf("Cannot supersede document: id is missing.\n\nTo resolve:\n  Provide the original document record ID: doc(action: \"supersede\", id: \"DOC-...\", superseded_by: \"DOC-...\")")
		}
		if supersededBy == "" {
			return nil, fmt.Errorf("Cannot supersede document %q: superseded_by is missing.\n\nTo resolve:\n  Provide the replacement document record ID: doc(action: \"supersede\", id: \"...\", superseded_by: \"DOC-...\")", docID)
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
			return nil, fmt.Errorf("Cannot retrieve supersession chain: id is missing.\n\nTo resolve:\n  Provide the document record ID: doc(action: \"chain\", id: \"DOC-...\")")
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
		args, _ := req.Params.Arguments.(map[string]any)
		path := docArgStr(args, "path")
		if path == "" {
			return nil, fmt.Errorf("Cannot import documents: path is missing.\n\nTo resolve:\n  Provide the directory to scan: doc(action: \"import\", path: \"work/\")")
		}

		cfg := config.LoadOrDefault()
		importSvc := service.NewBatchImportService(docSvc)

		// Dry-run mode: run the full inference pipeline without writing to the store.
		dryRun, _ := args["dry_run"].(bool)
		if dryRun {
			result, err := importSvc.ImportDryRun(cfg, service.BatchImportInput{
				Path:        path,
				DefaultType: docArgStr(args, "default_type"),
				Owner:       docArgStr(args, "owner"),
				Glob:        docArgStr(args, "glob"),
			})
			if err != nil {
				return nil, err
			}
			return result, nil
		}

		// Live import path (unchanged).
		SignalMutation(ctx)

		createdByRaw := docArgStr(args, "created_by")
		createdBy, err := config.ResolveIdentity(createdByRaw)
		if err != nil {
			return nil, err
		}

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

// ─── audit ────────────────────────────────────────────────────────────────────

// docAuditAction implements doc(action: "audit"). It is read-only: it walks
// document directories and compares against the store without modifying either.
func docAuditAction(docSvc *service.DocumentService) ActionHandler {
	return func(ctx context.Context, req mcp.CallToolRequest) (any, error) {
		args, _ := req.Params.Arguments.(map[string]any)

		includeRegistered, _ := args["include_registered"].(bool)

		// Determine which directories to scan.
		var dirs []string
		if p := docArgStr(args, "path"); p != "" {
			dirs = []string{p}
		}

		result, err := service.AuditDocuments(ctx, docSvc, docSvc.RepoRoot(), dirs, includeRegistered)
		if err != nil {
			return nil, err
		}

		// Build the response map.
		resp := map[string]any{
			"unregistered": auditUnregisteredToMaps(result.Unregistered),
			"missing":      auditMissingToMaps(result.Missing),
			"summary": map[string]any{
				"total_on_disk": result.Summary.TotalOnDisk,
				"registered":    result.Summary.Registered,
				"unregistered":  result.Summary.Unregistered,
				"missing":       result.Summary.Missing,
			},
		}

		// Include the registered list only when the flag was set (REQ-16).
		if includeRegistered {
			resp["registered"] = auditRegisteredToMaps(result.Registered)
		}

		return resp, nil
	}
}

func auditUnregisteredToMaps(files []service.UnregisteredFile) []map[string]any {
	out := make([]map[string]any, len(files))
	for i, f := range files {
		out[i] = map[string]any{
			"path":          f.Path,
			"inferred_type": f.InferredType,
		}
	}
	return out
}

func auditMissingToMaps(records []service.MissingRecord) []map[string]any {
	out := make([]map[string]any, len(records))
	for i, r := range records {
		out[i] = map[string]any{
			"path":   r.Path,
			"doc_id": r.DocID,
		}
	}
	return out
}

func auditRegisteredToMaps(files []service.RegisteredFile) []map[string]any {
	out := make([]map[string]any, len(files))
	for i, f := range files {
		out[i] = map[string]any{
			"path":   f.Path,
			"doc_id": f.DocID,
		}
	}
	return out
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
	if r.QualityEvaluation != nil {
		qe := r.QualityEvaluation
		dims := make(map[string]any, len(qe.Dimensions))
		for k, v := range qe.Dimensions {
			dims[k] = v
		}
		m["quality_evaluation"] = map[string]any{
			"overall_score": qe.OverallScore,
			"pass":          qe.Pass,
			"evaluated_at":  qe.EvaluatedAt,
			"evaluator":     qe.Evaluator,
			"dimensions":    dims,
		}
	}
	return m
}

// docArgStr safely extracts a string from an args map.
func docArgStr(args map[string]any, key string) string {
	v, _ := args[key].(string)
	return v
}

// ─── evaluate ─────────────────────────────────────────────────────────────────

func docEvaluateAction(docSvc *service.DocumentService) ActionHandler {
	return func(ctx context.Context, req mcp.CallToolRequest) (any, error) {
		SignalMutation(ctx)
		args, _ := req.Params.Arguments.(map[string]any)

		docID := docArgStr(args, "id")
		if docID == "" {
			return nil, fmt.Errorf("Cannot evaluate document: id is missing.\n\nTo resolve:\n  Provide the document record ID: doc(action: \"evaluate\", id: \"DOC-...\", evaluation: {...})")
		}

		evalRaw, ok := args["evaluation"].(map[string]any)
		if !ok || evalRaw == nil {
			return nil, fmt.Errorf("Cannot evaluate document %q: evaluation object is missing.\n\nTo resolve:\n  Provide the evaluation: doc(action: \"evaluate\", id: \"...\", evaluation: {\"overall_score\": 7.5, \"pass\": true, \"evaluated_at\": \"...\", \"evaluator\": \"...\"})", docID)
		}

		eval, err := parseEvaluationMap(evalRaw)
		if err != nil {
			return nil, fmt.Errorf("Cannot evaluate document %q: evaluation object is invalid.\n\nTo resolve:\n  Check the evaluation fields (overall_score, pass, evaluated_at, evaluator, dimensions).\n\nCause: %w", docID, err)
		}

		result, err := docSvc.AttachQualityEvaluation(service.AttachEvaluationInput{
			ID:         docID,
			Evaluation: eval,
		})
		if err != nil {
			return nil, err
		}

		return map[string]any{"document": docRecordToMap(result)}, nil
	}
}

// parseEvaluationMap converts a map[string]any from the MCP call into a
// model.QualityEvaluation, performing type coercions for numeric fields.
func parseEvaluationMap(m map[string]any) (model.QualityEvaluation, error) {
	var eval model.QualityEvaluation

	// overall_score
	switch v := m["overall_score"].(type) {
	case float64:
		eval.OverallScore = v
	case int:
		eval.OverallScore = float64(v)
	case string:
		f, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return eval, fmt.Errorf("Cannot parse overall_score: value is not a valid number.\n\nTo resolve:\n  Provide overall_score as a number (e.g., 7.5).\n\nCause: %w", err)
		}
		eval.OverallScore = f
	default:
		return eval, fmt.Errorf("Cannot parse evaluation: overall_score is missing or has an invalid type.\n\nTo resolve:\n  Provide overall_score as a number (e.g., 7.5).")
	}

	// pass
	if v, ok := m["pass"].(bool); ok {
		eval.Pass = v
	}

	// evaluator
	eval.Evaluator, _ = m["evaluator"].(string)

	// evaluated_at
	if v, ok := m["evaluated_at"].(string); ok {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			return eval, fmt.Errorf("Cannot parse evaluation: evaluated_at is not valid RFC3339.\n\nTo resolve:\n  Provide evaluated_at in RFC3339 format (e.g., \"2025-01-15T10:30:00Z\").\n\nCause: %w", err)
		}
		eval.EvaluatedAt = t
	} else {
		return eval, fmt.Errorf("Cannot parse evaluation: evaluated_at is missing.\n\nTo resolve:\n  Provide evaluated_at in RFC3339 format (e.g., \"2025-01-15T10:30:00Z\").")
	}

	// dimensions
	if dimsRaw, ok := m["dimensions"].(map[string]any); ok && len(dimsRaw) > 0 {
		eval.Dimensions = make(map[string]float64, len(dimsRaw))
		for k, dv := range dimsRaw {
			switch tv := dv.(type) {
			case float64:
				eval.Dimensions[k] = tv
			case int:
				eval.Dimensions[k] = float64(tv)
			case string:
				f, err := strconv.ParseFloat(tv, 64)
				if err != nil {
					return eval, fmt.Errorf("Cannot parse evaluation: dimension %q has an invalid value.\n\nTo resolve:\n  Provide dimension scores as numbers (e.g., \"clarity\": 8.0).\n\nCause: %w", k, err)
				}
				eval.Dimensions[k] = f
			}
		}
	}

	return eval, nil
}
