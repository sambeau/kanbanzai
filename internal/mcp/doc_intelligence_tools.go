package mcp

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"kanbanzai/internal/docint"
	"kanbanzai/internal/service"
)

// DocIntelligenceTools returns all document intelligence MCP tool definitions with their handlers.
func DocIntelligenceTools(svc *service.IntelligenceService, docSvc *service.DocumentService) []server.ServerTool {
	return []server.ServerTool{
		docClassifyTool(svc),
		docOutlineTool(svc),
		docSectionTool(svc),
		docFindByEntityTool(svc),
		docFindByConceptTool(svc),
		docFindByRoleTool(svc),
		docTraceTool(svc),
		docGapsTool(svc, docSvc),
		docPendingTool(svc),
		docImpactTool(svc),
	}
}

func docClassifyTool(svc *service.IntelligenceService) server.ServerTool {
	tool := mcp.NewTool("doc_classify",
		mcp.WithDescription("Submit agent-provided classifications (Layer 3) for a previously indexed document. The content_hash must match the current index to prevent stale classifications. Classifications assign semantic roles (requirement, decision, rationale, etc.) to document sections."),
		mcp.WithString("id", mcp.Description("Document ID to classify"), mcp.Required()),
		mcp.WithString("content_hash", mcp.Description("Content hash of the document (must match current index to prevent stale classification)"), mcp.Required()),
		mcp.WithString("model_name", mcp.Description("Name of the LLM that produced the classifications"), mcp.Required()),
		mcp.WithString("model_version", mcp.Description("Version of the LLM that produced the classifications"), mcp.Required()),
		mcp.WithString("classifications", mcp.Description("JSON array of classification objects, each with: section_path, role, confidence, summary (optional), concepts_intro (optional), concepts_used (optional)"), mcp.Required()),
		mcp.WithReadOnlyHintAnnotation(false),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(false),
	)
	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		id, err := request.RequireString("id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		contentHash, err := request.RequireString("content_hash")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		modelName, err := request.RequireString("model_name")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		modelVersion, err := request.RequireString("model_version")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		classificationsJSON, err := request.RequireString("classifications")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		var classifications []docint.Classification
		if err := json.Unmarshal([]byte(classificationsJSON), &classifications); err != nil {
			return mcp.NewToolResultError("invalid classifications JSON: " + err.Error()), nil
		}

		submission := docint.ClassificationSubmission{
			DocumentID:      id,
			ContentHash:     contentHash,
			ModelName:       modelName,
			ModelVersion:    modelVersion,
			ClassifiedAt:    time.Now().UTC(),
			Classifications: classifications,
		}

		if err := svc.ClassifyDocument(submission); err != nil {
			return mcp.NewToolResultErrorFromErr("classify document failed", err), nil
		}

		return docIntMapJSON(map[string]any{
			"success":     true,
			"document_id": id,
			"message":     "Classifications applied successfully",
			"count":       len(classifications),
		})
	}
	return server.ServerTool{Tool: tool, Handler: handler}
}

func docOutlineTool(svc *service.IntelligenceService) server.ServerTool {
	tool := mcp.NewTool("doc_outline",
		mcp.WithDescription("Get the structural outline (Layer 1) of an indexed document. Returns the section tree with paths, titles, levels, word counts, and content hashes."),
		mcp.WithString("id", mcp.Description("Document ID"), mcp.Required()),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
	)
	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		id, err := request.RequireString("id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		sections, err := svc.GetOutline(id)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("get outline failed", err), nil
		}

		return docIntMapJSON(map[string]any{
			"success":     true,
			"document_id": id,
			"sections":    sections,
		})
	}
	return server.ServerTool{Tool: tool, Handler: handler}
}

func docSectionTool(svc *service.IntelligenceService) server.ServerTool {
	tool := mcp.NewTool("doc_section",
		mcp.WithDescription("Get a specific section's metadata and raw content from an indexed document. Use the section_path from doc_outline to identify sections."),
		mcp.WithString("id", mcp.Description("Document ID"), mcp.Required()),
		mcp.WithString("section_path", mcp.Description("Section path (e.g. '1', '1.2', '2.3.1')"), mcp.Required()),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
	)
	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		id, err := request.RequireString("id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		sectionPath, err := request.RequireString("section_path")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		section, content, err := svc.GetSection(id, sectionPath)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("get section failed", err), nil
		}

		result := map[string]any{
			"success":     true,
			"document_id": id,
			"section": map[string]any{
				"path":         section.Path,
				"level":        section.Level,
				"title":        section.Title,
				"word_count":   section.WordCount,
				"content_hash": section.ContentHash,
			},
			"content": string(content),
		}

		return docIntMapJSON(result)
	}
	return server.ServerTool{Tool: tool, Handler: handler}
}

func docFindByEntityTool(svc *service.IntelligenceService) server.ServerTool {
	tool := mcp.NewTool("doc_find_by_entity",
		mcp.WithDescription("Find all document sections across the corpus that reference a specific entity (FEAT-xxx, TASK-xxx, BUG-xxx, DEC-xxx, EPIC-xxx, or Plan IDs)."),
		mcp.WithString("entity_id", mcp.Description("Entity ID to search for (e.g. FEAT-001, TASK-042, P1-basic-ui)"), mcp.Required()),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
	)
	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		entityID, err := request.RequireString("entity_id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		matches, err := svc.FindByEntity(entityID)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("find by entity failed", err), nil
		}

		return docIntMapJSON(map[string]any{
			"success":   true,
			"entity_id": entityID,
			"count":     len(matches),
			"matches":   matches,
		})
	}
	return server.ServerTool{Tool: tool, Handler: handler}
}

func docFindByConceptTool(svc *service.IntelligenceService) server.ServerTool {
	tool := mcp.NewTool("doc_find_by_concept",
		mcp.WithDescription("Find all document sections that introduce or use a specific concept. Concepts are identified during Layer 3 classification."),
		mcp.WithString("concept", mcp.Description("Concept name to search for (case-insensitive, will be normalized)"), mcp.Required()),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
	)
	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		concept, err := request.RequireString("concept")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		matches, err := svc.FindByConcept(concept)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("find by concept failed", err), nil
		}

		return docIntMapJSON(map[string]any{
			"success": true,
			"concept": concept,
			"count":   len(matches),
			"matches": matches,
		})
	}
	return server.ServerTool{Tool: tool, Handler: handler}
}

func docFindByRoleTool(svc *service.IntelligenceService) server.ServerTool {
	tool := mcp.NewTool("doc_find_by_role",
		mcp.WithDescription("Find all document fragments with a given semantic role across the corpus. Valid roles: requirement, decision, rationale, constraint, assumption, risk, question, definition, example, alternative, narrative."),
		mcp.WithString("role", mcp.Description("Fragment role to search for"), mcp.Required()),
		mcp.WithString("scope", mcp.Description("Optional: limit search to a specific document ID")),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
	)
	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		role, err := request.RequireString("role")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		scope := request.GetString("scope", "")

		matches, err := svc.FindByRole(role, scope)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("find by role failed", err), nil
		}

		result := map[string]any{
			"success": true,
			"role":    role,
			"count":   len(matches),
			"matches": matches,
		}
		if scope != "" {
			result["scope"] = scope
		}

		return docIntMapJSON(result)
	}
	return server.ServerTool{Tool: tool, Handler: handler}
}

func docTraceTool(svc *service.IntelligenceService) server.ServerTool {
	tool := mcp.NewTool("doc_trace",
		mcp.WithDescription("Trace an entity through the document refinement chain. Returns all document sections that reference the entity, ordered by document type (design → specification → dev-plan)."),
		mcp.WithString("entity_id", mcp.Description("Entity ID to trace through the refinement chain"), mcp.Required()),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
	)
	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		entityID, err := request.RequireString("entity_id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		matches, err := svc.TraceEntity(entityID)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("trace entity failed", err), nil
		}

		return docIntMapJSON(map[string]any{
			"success":   true,
			"entity_id": entityID,
			"count":     len(matches),
			"matches":   matches,
		})
	}
	return server.ServerTool{Tool: tool, Handler: handler}
}

func docGapsTool(svc *service.IntelligenceService, docSvc *service.DocumentService) server.ServerTool {
	tool := mcp.NewTool("doc_gaps",
		mcp.WithDescription("Analyze what document types are missing for a feature. Checks whether design, specification, and dev-plan documents exist for the given feature."),
		mcp.WithString("feature_id", mcp.Description("Feature ID to analyze for document gaps"), mcp.Required()),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
	)
	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		featureID, err := request.RequireString("feature_id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		gaps, err := svc.AnalyzeGaps(featureID, docSvc)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("analyze gaps failed", err), nil
		}

		message := "All expected document types exist"
		if len(gaps) > 0 {
			message = "Missing document types: " + strings.Join(gaps, ", ")
		}

		return docIntMapJSON(map[string]any{
			"success":    true,
			"feature_id": featureID,
			"complete":   len(gaps) == 0,
			"gaps":       gaps,
			"message":    message,
		})
	}
	return server.ServerTool{Tool: tool, Handler: handler}
}

func docPendingTool(svc *service.IntelligenceService) server.ServerTool {
	tool := mcp.NewTool("doc_pending",
		mcp.WithDescription("List document IDs that have been indexed (Layers 1-2) but not yet classified (Layer 3). These documents are ready for agent classification."),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
	)
	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		pending, err := svc.GetPendingClassification()
		if err != nil {
			return mcp.NewToolResultErrorFromErr("get pending classification failed", err), nil
		}

		return docIntMapJSON(map[string]any{
			"success":      true,
			"count":        len(pending),
			"document_ids": pending,
		})
	}
	return server.ServerTool{Tool: tool, Handler: handler}
}

func docImpactTool(svc *service.IntelligenceService) server.ServerTool {
	tool := mcp.NewTool("doc_impact",
		mcp.WithDescription("Find what references or depends on a given section. Returns all graph edges where the target matches the section ID. Section IDs have the format 'docID#sectionPath'."),
		mcp.WithString("section_id", mcp.Description("Section ID in the format 'docID#sectionPath' (e.g. 'PROJECT/design-workflow#1.2')"), mcp.Required()),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
	)
	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		sectionID, err := request.RequireString("section_id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		edges, err := svc.GetImpact(sectionID)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("get impact failed", err), nil
		}

		edgeMaps := make([]map[string]any, len(edges))
		for i, e := range edges {
			edgeMaps[i] = map[string]any{
				"from":      e.From,
				"from_type": e.FromType,
				"to":        e.To,
				"to_type":   e.ToType,
				"edge_type": e.EdgeType,
			}
		}

		return docIntMapJSON(map[string]any{
			"success":    true,
			"section_id": sectionID,
			"count":      len(edges),
			"edges":      edgeMaps,
		})
	}
	return server.ServerTool{Tool: tool, Handler: handler}
}

// docIntMapJSON marshals a map to JSON and returns it as a tool result.
func docIntMapJSON(v map[string]any) (*mcp.CallToolResult, error) {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return mcp.NewToolResultError("marshal result: " + err.Error()), nil
	}
	return mcp.NewToolResultText(string(data)), nil
}
