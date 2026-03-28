package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/sambeau/kanbanzai/internal/docint"
	"github.com/sambeau/kanbanzai/internal/service"
)

// DocIntelTool returns the 2.0 consolidated doc_intel tool.
// It consolidates doc_outline, doc_section, doc_classify, doc_find_by_concept,
// doc_find_by_entity, doc_find_by_role, doc_trace, doc_impact, doc_extraction_guide,
// and doc_pending into a single tool with an action parameter (spec §20.1).
func DocIntelTool(intelligenceSvc *service.IntelligenceService, docRecordSvc *service.DocumentService) []server.ServerTool {
	return []server.ServerTool{docIntelTool(intelligenceSvc, docRecordSvc)}
}

func docIntelTool(intelligenceSvc *service.IntelligenceService, docRecordSvc *service.DocumentService) server.ServerTool {
	tool := mcp.NewTool("doc_intel",
		mcp.WithReadOnlyHintAnnotation(false),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(false),
		mcp.WithOpenWorldHintAnnotation(false),
		mcp.WithTitleAnnotation("Document Intelligence"),
		mcp.WithDescription(
			"Document intelligence operations: explore, classify, and query the document graph. "+
				"Consolidates doc_outline, doc_section, doc_classify, doc_find_by_concept, "+
				"doc_find_by_entity, doc_find_by_role, doc_trace, doc_impact, "+
				"doc_extraction_guide, and doc_pending. "+
				"Actions: outline, section, classify, find, trace, impact, guide, pending. "+
				"The find action routes by parameter: concept, entity_id, or role.",
		),
		mcp.WithString("action",
			mcp.Required(),
			mcp.Description("Action: outline, section, classify, find, trace, impact, guide, pending"),
		),
		// outline / section / classify / guide — document identity
		mcp.WithString("id",
			mcp.Description("Document ID — required for outline, section, classify, guide"),
		),
		// section
		mcp.WithString("section_path",
			mcp.Description("Section path e.g. '1', '1.2', '2.3.1' — required for section"),
		),
		// classify
		mcp.WithString("content_hash",
			mcp.Description("Content hash of the document (must match current index) — required for classify"),
		),
		mcp.WithString("model_name",
			mcp.Description("Name of the LLM that produced the classifications — required for classify"),
		),
		mcp.WithString("model_version",
			mcp.Description("Version of the LLM that produced the classifications — required for classify"),
		),
		mcp.WithString("classifications",
			mcp.Description("JSON array of classification objects with section_path, role, confidence, etc. — required for classify"),
		),
		// find — exactly one of concept, entity_id, or role must be provided
		mcp.WithString("concept",
			mcp.Description("Concept name to search for (case-insensitive, normalised) — find action"),
		),
		mcp.WithString("entity_id",
			mcp.Description("Entity ID to search for (e.g. FEAT-001, TASK-042) — find and trace actions"),
		),
		mcp.WithString("role",
			mcp.Description("Fragment role to search for (requirement, decision, rationale, etc.) — find action"),
		),
		mcp.WithString("scope",
			mcp.Description("Limit role search to a specific document ID — find action with role"),
		),
		// impact
		mcp.WithString("section_id",
			mcp.Description("Section ID in the format 'docID#sectionPath' — required for impact"),
		),
		// guide — uses id (same as outline)
	)

	// doc_intel is predominantly read-only; no WithSideEffects wrapper needed
	// except for classify which mutates the document index. We use a plain handler
	// and call SignalMutation inside classify.
	handler := WithSideEffects(func(ctx context.Context, req mcp.CallToolRequest) (any, error) {
		return DispatchAction(ctx, req, map[string]ActionHandler{
			"outline":  docIntelOutlineAction(intelligenceSvc),
			"section":  docIntelSectionAction(intelligenceSvc),
			"classify": docIntelClassifyAction(intelligenceSvc),
			"find":     docIntelFindAction(intelligenceSvc),
			"trace":    docIntelTraceAction(intelligenceSvc),
			"impact":   docIntelImpactAction(intelligenceSvc),
			"guide":    docIntelGuideAction(intelligenceSvc),
			"pending":  docIntelPendingAction(intelligenceSvc),
		})
	})

	return server.ServerTool{Tool: tool, Handler: handler}
}

// ─── outline ──────────────────────────────────────────────────────────────────

func docIntelOutlineAction(svc *service.IntelligenceService) ActionHandler {
	return func(ctx context.Context, req mcp.CallToolRequest) (any, error) {
		id, err := req.RequireString("id")
		if err != nil {
			return inlineErr("missing_parameter", "id is required for outline action")
		}

		sections, err := svc.GetOutline(id)
		if err != nil {
			return nil, fmt.Errorf("get outline: %w", err)
		}

		return map[string]any{
			"document_id": id,
			"sections":    sections,
		}, nil
	}
}

// ─── section ──────────────────────────────────────────────────────────────────

func docIntelSectionAction(svc *service.IntelligenceService) ActionHandler {
	return func(ctx context.Context, req mcp.CallToolRequest) (any, error) {
		id, err := req.RequireString("id")
		if err != nil {
			return inlineErr("missing_parameter", "id is required for section action")
		}
		sectionPath, err := req.RequireString("section_path")
		if err != nil {
			return inlineErr("missing_parameter", "section_path is required for section action")
		}

		section, content, err := svc.GetSection(id, sectionPath)
		if err != nil {
			return nil, fmt.Errorf("get section: %w", err)
		}

		return map[string]any{
			"document_id": id,
			"section": map[string]any{
				"path":         section.Path,
				"level":        section.Level,
				"title":        section.Title,
				"word_count":   section.WordCount,
				"content_hash": section.ContentHash,
			},
			"content": string(content),
		}, nil
	}
}

// ─── classify ─────────────────────────────────────────────────────────────────

func docIntelClassifyAction(svc *service.IntelligenceService) ActionHandler {
	return func(ctx context.Context, req mcp.CallToolRequest) (any, error) {
		SignalMutation(ctx)

		id, err := req.RequireString("id")
		if err != nil {
			return inlineErr("missing_parameter", "id is required for classify action")
		}
		contentHash, err := req.RequireString("content_hash")
		if err != nil {
			return inlineErr("missing_parameter", "content_hash is required for classify action")
		}
		modelName, err := req.RequireString("model_name")
		if err != nil {
			return inlineErr("missing_parameter", "model_name is required for classify action")
		}
		modelVersion, err := req.RequireString("model_version")
		if err != nil {
			return inlineErr("missing_parameter", "model_version is required for classify action")
		}
		classificationsJSON, err := req.RequireString("classifications")
		if err != nil {
			return inlineErr("missing_parameter", "classifications is required for classify action")
		}

		var classifications []docint.Classification
		if jsonErr := json.Unmarshal([]byte(classificationsJSON), &classifications); jsonErr != nil {
			return inlineErr("invalid_parameter", "invalid classifications JSON: "+jsonErr.Error())
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
			return nil, fmt.Errorf("classify document: %w", err)
		}

		return map[string]any{
			"document_id": id,
			"message":     "Classifications applied successfully",
			"count":       len(classifications),
		}, nil
	}
}

// ─── find ─────────────────────────────────────────────────────────────────────

// docIntelFindAction routes the find action based on which discriminator
// parameter is present: concept → FindByConcept, entity_id → FindByEntity,
// role → FindByRole. Returns an error if none are provided (spec §20.1).
func docIntelFindAction(svc *service.IntelligenceService) ActionHandler {
	return func(ctx context.Context, req mcp.CallToolRequest) (any, error) {
		args := req.GetArguments()

		// Check concept first, then entity_id, then role (spec §20.1 ordering).
		if concept, ok := args["concept"].(string); ok && concept != "" {
			matches, err := svc.FindByConcept(concept)
			if err != nil {
				return nil, fmt.Errorf("find by concept: %w", err)
			}
			return map[string]any{
				"search_type": "concept",
				"concept":     concept,
				"count":       len(matches),
				"matches":     matches,
			}, nil
		}

		if entityID, ok := args["entity_id"].(string); ok && entityID != "" {
			matches, err := svc.FindByEntity(entityID)
			if err != nil {
				return nil, fmt.Errorf("find by entity: %w", err)
			}
			return map[string]any{
				"search_type": "entity_id",
				"entity_id":   entityID,
				"count":       len(matches),
				"matches":     matches,
			}, nil
		}

		if role, ok := args["role"].(string); ok && role != "" {
			scope := req.GetString("scope", "")
			matches, err := svc.FindByRole(role, scope)
			if err != nil {
				return nil, fmt.Errorf("find by role: %w", err)
			}
			result := map[string]any{
				"search_type": "role",
				"role":        role,
				"count":       len(matches),
				"matches":     matches,
			}
			if scope != "" {
				result["scope"] = scope
			}
			return result, nil
		}

		return inlineErr("missing_parameter",
			"find action requires exactly one of: concept, entity_id, or role")
	}
}

// ─── trace ────────────────────────────────────────────────────────────────────

func docIntelTraceAction(svc *service.IntelligenceService) ActionHandler {
	return func(ctx context.Context, req mcp.CallToolRequest) (any, error) {
		entityID, err := req.RequireString("entity_id")
		if err != nil {
			return inlineErr("missing_parameter", "entity_id is required for trace action")
		}

		matches, err := svc.TraceEntity(entityID)
		if err != nil {
			return nil, fmt.Errorf("trace entity: %w", err)
		}

		return map[string]any{
			"entity_id": entityID,
			"count":     len(matches),
			"matches":   matches,
		}, nil
	}
}

// ─── impact ───────────────────────────────────────────────────────────────────

func docIntelImpactAction(svc *service.IntelligenceService) ActionHandler {
	return func(ctx context.Context, req mcp.CallToolRequest) (any, error) {
		sectionID, err := req.RequireString("section_id")
		if err != nil {
			return inlineErr("missing_parameter", "section_id is required for impact action")
		}

		edges, err := svc.GetImpact(sectionID)
		if err != nil {
			return nil, fmt.Errorf("get impact: %w", err)
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

		return map[string]any{
			"section_id": sectionID,
			"count":      len(edges),
			"edges":      edgeMaps,
		}, nil
	}
}

// ─── guide ────────────────────────────────────────────────────────────────────

func docIntelGuideAction(svc *service.IntelligenceService) ActionHandler {
	return func(ctx context.Context, req mcp.CallToolRequest) (any, error) {
		documentID, err := req.RequireString("id")
		if err != nil {
			return inlineErr("missing_parameter", "id is required for guide action")
		}

		index, err := svc.GetDocumentIndex(documentID)
		if err != nil {
			return nil, fmt.Errorf("get document index: %w", err)
		}

		outline := flattenSections(index.Sections, buildClassifiedRoleMap(index))

		type entityRefEntry struct {
			EntityID    string `json:"entity_id"`
			EntityType  string `json:"entity_type"`
			SectionPath string `json:"section_path"`
		}

		entityRefs := make([]entityRefEntry, 0, len(index.EntityRefs))
		for _, ref := range index.EntityRefs {
			entityRefs = append(entityRefs, entityRefEntry{
				EntityID:    ref.EntityID,
				EntityType:  ref.EntityType,
				SectionPath: ref.SectionPath,
			})
		}

		return map[string]any{
			"document_id":      documentID,
			"document_path":    index.DocumentPath,
			"content_hash":     index.ContentHash,
			"classified":       index.Classified,
			"outline":          outline,
			"entity_refs":      entityRefs,
			"extraction_hints": extractionHints(index),
		}, nil
	}
}

// ─── pending ──────────────────────────────────────────────────────────────────

func docIntelPendingAction(svc *service.IntelligenceService) ActionHandler {
	return func(ctx context.Context, req mcp.CallToolRequest) (any, error) {
		pending, err := svc.GetPendingClassification()
		if err != nil {
			return nil, fmt.Errorf("get pending classification: %w", err)
		}

		return map[string]any{
			"count":        len(pending),
			"document_ids": pending,
		}, nil
	}
}

// ─── helpers (moved from agent_capability_tools.go) ───────────────────────────

func buildClassifiedRoleMap(index *docint.DocumentIndex) map[string]string {
	roles := make(map[string]string)
	for _, cr := range index.ConventionalRoles {
		roles[cr.SectionPath] = cr.Role
	}
	for _, c := range index.Classifications {
		roles[c.SectionPath] = c.Role
	}
	return roles
}

type sectionGuide struct {
	Path     string         `json:"path"`
	Title    string         `json:"title"`
	Level    int            `json:"level"`
	Role     string         `json:"role,omitempty"`
	Children []sectionGuide `json:"children,omitempty"`
}

func flattenSections(sections []docint.Section, roles map[string]string) []sectionGuide {
	if len(sections) == 0 {
		return nil
	}
	result := make([]sectionGuide, 0, len(sections))
	for _, s := range sections {
		sg := sectionGuide{
			Path:  s.Path,
			Title: s.Title,
			Level: s.Level,
			Role:  roles[s.Path],
		}
		if len(s.Children) > 0 {
			sg.Children = flattenSections(s.Children, roles)
		}
		result = append(result, sg)
	}
	return result
}

func extractionHints(index *docint.DocumentIndex) []string {
	var hints []string
	if len(index.EntityRefs) > 0 {
		hints = append(hints, fmt.Sprintf("%d entity reference(s) already detected — consider cross-checking against entity store", len(index.EntityRefs)))
	}
	if index.Classified {
		hints = append(hints, "Layer 3 classifications are available — use section roles to target extraction")
	} else {
		hints = append(hints, "Document has not been Layer 3 classified yet — consider running doc_classify first for richer extraction guidance")
	}
	roleCount := make(map[string]int)
	for _, c := range index.Classifications {
		roleCount[c.Role]++
	}
	for _, cr := range index.ConventionalRoles {
		if _, alreadyCounted := roleCount[cr.Role]; !alreadyCounted {
			roleCount[cr.Role]++
		}
	}
	if n := roleCount["requirement"]; n > 0 {
		hints = append(hints, fmt.Sprintf("%d requirement section(s) found — extract Tasks or acceptance criteria", n))
	}
	if n := roleCount["decision"]; n > 0 {
		hints = append(hints, fmt.Sprintf("%d decision section(s) found — extract Decision entities", n))
	}
	if n := roleCount["rationale"]; n > 0 {
		hints = append(hints, fmt.Sprintf("%d rationale section(s) found — link to corresponding Decision entities", n))
	}
	return hints
}
