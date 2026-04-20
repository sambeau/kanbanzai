package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
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
// DocIntelTool returns the doc_intel MCP tool registered with the given services.
// knowledgeSvc may be nil; when nil, find(entity_id) returns an empty related_knowledge array.
func DocIntelTool(intelligenceSvc *service.IntelligenceService, docRecordSvc *service.DocumentService, knowledgeSvc *service.KnowledgeService) []server.ServerTool {
	return []server.ServerTool{docIntelTool(intelligenceSvc, docRecordSvc, knowledgeSvc)}
}

func docIntelTool(intelligenceSvc *service.IntelligenceService, docRecordSvc *service.DocumentService, knowledgeSvc *service.KnowledgeService) server.ServerTool {
	tool := mcp.NewTool("doc_intel",
		mcp.WithReadOnlyHintAnnotation(false),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(false),
		mcp.WithOpenWorldHintAnnotation(false),
		mcp.WithTitleAnnotation("Document Intelligence"),
		mcp.WithDescription(
			"Explore, classify, and query the document graph — use for understanding document structure, "+
				"finding content by concept/entity/role, and assessing change impact on sections. "+
				"Do NOT use for document record management (register, approve, supersede) — use doc instead. "+
				"Call guide before manually extracting information from a document. "+
				"The find action routes by parameter: provide exactly one of concept, entity_id, or role. "+
				"id is required for outline, section, classify, and guide. "+
				"Actions: outline, section, classify, find, trace, impact, guide, pending, search.",
		),
		mcp.WithString("action",
			mcp.Required(),
			mcp.Description("Action: outline, section, classify, find, trace, impact, guide, pending, search"),
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
			mcp.Description("Fragment role to search for (requirement, decision, rationale, etc.) — find and search actions"),
		),
		mcp.WithString("scope",
			mcp.Description("Limit role search to a specific document ID — find action with role"),
		),
		// impact
		mcp.WithString("section_id",
			mcp.Description("Section ID in the format 'docID#sectionPath' — required for impact"),
		),
		// guide — uses id (same as outline)
		// search
		mcp.WithString("query",
			mcp.Description("Full-text search query in FTS5 syntax — required for search. total_matches in the response is the count before doc_type/role post-filtering."),
		),
		mcp.WithString("mode",
			mcp.Description("Output level: outline (default), summary, full — search action"),
		),
		mcp.WithNumber("limit",
			mcp.Description("Maximum results (default 10, max 50) — search action"),
		),
		mcp.WithString("doc_type",
			mcp.Description("Filter by document type (e.g. specification, design) — search action"),
		),
	)

	// doc_intel is predominantly read-only; no WithSideEffects wrapper needed
	// except for classify which mutates the document index. We use a plain handler
	// and call SignalMutation inside classify.
	handler := WithSideEffects(func(ctx context.Context, req mcp.CallToolRequest) (any, error) {
		return DispatchAction(ctx, req, map[string]ActionHandler{
			"outline":  docIntelOutlineAction(intelligenceSvc),
			"section":  docIntelSectionAction(intelligenceSvc),
			"classify": docIntelClassifyAction(intelligenceSvc),
			"find":     docIntelFindAction(intelligenceSvc, knowledgeSvc),
			"trace":    docIntelTraceAction(intelligenceSvc),
			"impact":   docIntelImpactAction(intelligenceSvc),
			"guide":    docIntelGuideAction(intelligenceSvc),
			"pending":  docIntelPendingAction(intelligenceSvc),
			"search":   docIntelSearchAction(intelligenceSvc),
		})
	})

	return server.ServerTool{Tool: tool, Handler: handler}
}

// ─── outline ──────────────────────────────────────────────────────────────────

func docIntelOutlineAction(svc *service.IntelligenceService) ActionHandler {
	return func(ctx context.Context, req mcp.CallToolRequest) (any, error) {
		id, err := req.RequireString("id")
		if err != nil {
			return inlineErr("missing_parameter", "Cannot get outline: id is missing.\n\nTo resolve:\n  Provide id: doc_intel(action: \"outline\", id: \"DOC-...\")")
		}

		sections, err := svc.GetOutline(id)
		if err != nil {
			return nil, fmt.Errorf("Cannot get outline for document %s: %w.\n\nTo resolve:\n  Verify the document ID is valid and the document has been indexed", id, err)
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
			return inlineErr("missing_parameter", "Cannot get section: id is missing.\n\nTo resolve:\n  Provide id: doc_intel(action: \"section\", id: \"DOC-...\", section_path: \"1.2\")")
		}
		sectionPath, err := req.RequireString("section_path")
		if err != nil {
			return inlineErr("missing_parameter", "Cannot get section: section_path is missing.\n\nTo resolve:\n  Provide section_path: doc_intel(action: \"section\", id: \"DOC-...\", section_path: \"1.2\")")
		}

		section, content, err := svc.GetSection(id, sectionPath)
		if err != nil {
			return nil, fmt.Errorf("Cannot get section %s of document %s: %w.\n\nTo resolve:\n  Verify the document ID and section path are correct using doc_intel(action: \"outline\", id: \"...\")", sectionPath, id, err)
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
			return inlineErr("missing_parameter", "Cannot classify document: id is missing.\n\nTo resolve:\n  Provide id: doc_intel(action: \"classify\", id: \"DOC-...\", ...)")
		}
		contentHash, err := req.RequireString("content_hash")
		if err != nil {
			return inlineErr("missing_parameter", "Cannot classify document: content_hash is missing.\n\nTo resolve:\n  Provide content_hash from doc_intel(action: \"guide\", id: \"DOC-...\") to get the current hash")
		}
		modelName, err := req.RequireString("model_name")
		if err != nil {
			return inlineErr("missing_parameter", "Cannot classify document: model_name is missing.\n\nTo resolve:\n  Provide model_name: doc_intel(action: \"classify\", model_name: \"...\", ...)")
		}
		modelVersion, err := req.RequireString("model_version")
		if err != nil {
			return inlineErr("missing_parameter", "Cannot classify document: model_version is missing.\n\nTo resolve:\n  Provide model_version: doc_intel(action: \"classify\", model_version: \"...\", ...)")
		}
		classificationsJSON, err := req.RequireString("classifications")
		if err != nil {
			return inlineErr("missing_parameter", "Cannot classify document: classifications is missing.\n\nTo resolve:\n  Provide classifications as a JSON array: doc_intel(action: \"classify\", classifications: \"[{...}]\", ...)")
		}

		var classifications []docint.Classification
		if jsonErr := json.Unmarshal([]byte(classificationsJSON), &classifications); jsonErr != nil {
			return inlineErr("invalid_parameter", "Cannot classify document: classifications JSON is invalid: "+jsonErr.Error()+".\n\nTo resolve:\n  Provide a valid JSON array of objects with section_path, role, and confidence fields")
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
			return nil, fmt.Errorf("Cannot classify document %s: %w.\n\nTo resolve:\n  Verify the content_hash matches the current document index using doc_intel(action: \"guide\", id: \"...\")", id, err)
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
func docIntelFindAction(svc *service.IntelligenceService, knowledgeSvc *service.KnowledgeService) ActionHandler {
	return func(ctx context.Context, req mcp.CallToolRequest) (any, error) {
		args := req.GetArguments()

		// Check concept first, then entity_id, then role (spec §20.1 ordering).
		if concept, ok := args["concept"].(string); ok && concept != "" {
			matches, err := svc.FindByConcept(concept)
			if err != nil {
				return nil, fmt.Errorf("Cannot find by concept %q: %w.\n\nTo resolve:\n  Check the concept name and try a broader search term", concept, err)
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
				return nil, fmt.Errorf("Cannot find by entity %s: %w.\n\nTo resolve:\n  Verify the entity ID is valid using entity(action: \"get\", id: \"...\")", entityID, err)
			}
			relatedKnowledge, knowledgeCount := findRelatedKnowledge(knowledgeSvc, svc, entityID)
			return map[string]any{
				"search_type":       "entity_id",
				"entity_id":         entityID,
				"count":             len(matches),
				"matches":           matches,
				"related_knowledge": relatedKnowledge,
				"knowledge_matches": knowledgeCount,
			}, nil
		}

		if role, ok := args["role"].(string); ok && role != "" {
			scope := req.GetString("scope", "")
			matches, err := svc.FindByRole(role, scope)
			if err != nil {
				return nil, fmt.Errorf("Cannot find by role %q: %w.\n\nTo resolve:\n  Verify the role name is valid (e.g. requirement, decision, rationale)", role, err)
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
			"Cannot find document fragments: no search parameter provided.\n\nTo resolve:\n  Provide exactly one of concept, entity_id, or role: doc_intel(action: \"find\", concept: \"...\")")
	}
}

// ─── trace ────────────────────────────────────────────────────────────────────

func docIntelTraceAction(svc *service.IntelligenceService) ActionHandler {
	return func(ctx context.Context, req mcp.CallToolRequest) (any, error) {
		entityID, err := req.RequireString("entity_id")
		if err != nil {
			return inlineErr("missing_parameter", "Cannot trace entity: entity_id is missing.\n\nTo resolve:\n  Provide entity_id: doc_intel(action: \"trace\", entity_id: \"FEAT-...\")")
		}

		matches, err := svc.TraceEntity(entityID)
		if err != nil {
			return nil, fmt.Errorf("Cannot trace entity %s: %w.\n\nTo resolve:\n  Verify the entity ID is valid using entity(action: \"get\", id: \"...\")", entityID, err)
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
			return inlineErr("missing_parameter", "Cannot assess impact: section_id is missing.\n\nTo resolve:\n  Provide section_id in format 'DOC-...#sectionPath': doc_intel(action: \"impact\", section_id: \"DOC-...#1.2\")")
		}

		edges, err := svc.GetImpact(sectionID)
		if err != nil {
			return nil, fmt.Errorf("Cannot assess impact for section %s: %w.\n\nTo resolve:\n  Verify the section ID format is 'DOC-...#sectionPath' using doc_intel(action: \"outline\", id: \"...\")", sectionID, err)
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
			return inlineErr("missing_parameter", "Cannot generate extraction guide: id is missing.\n\nTo resolve:\n  Provide id: doc_intel(action: \"guide\", id: \"DOC-...\")")
		}

		index, err := svc.GetDocumentIndex(documentID)
		if err != nil {
			return nil, fmt.Errorf("Cannot generate extraction guide for document %s: %w.\n\nTo resolve:\n  Verify the document ID is valid and the document has been indexed", documentID, err)
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
			return nil, fmt.Errorf("Cannot list pending classifications: %w.\n\nTo resolve:\n  Check that the document index is intact and re-index if necessary", err)
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

// ─── search ───────────────────────────────────────────────────────────────────

func docIntelSearchAction(svc *service.IntelligenceService) ActionHandler {
	return func(ctx context.Context, req mcp.CallToolRequest) (any, error) {
		query, err := req.RequireString("query")
		if err != nil || query == "" {
			return inlineErr("missing_parameter",
				"Cannot search: query is missing.\n\nTo resolve:\n  Provide query: doc_intel(action: \"search\", query: \"...\")")
		}

		mode := req.GetString("mode", "outline")
		if mode != "outline" && mode != "summary" && mode != "full" {
			mode = "outline"
		}

		limit := int(req.GetFloat("limit", 10))
		if limit <= 0 {
			limit = 10
		}
		if limit > 50 {
			limit = 50
		}

		docType := req.GetString("doc_type", "")
		role := req.GetString("role", "")

		params := docint.SearchParams{
			Query:   query,
			Mode:    mode,
			Limit:   limit,
			DocType: docType,
			Role:    role,
		}

		total, results, err := svc.Search(params)
		if err != nil {
			return nil, fmt.Errorf("search failed: %w", err)
		}
		if results == nil {
			results = []docint.SearchResult{}
		}

		return map[string]any{
			"query":         query,
			"total_matches": total,
			"returned":      len(results),
			"results":       results,
		}, nil
	}
}

// ─── knowledge cross-query helpers ───────────────────────────────────────────

// findRelatedKnowledge returns knowledge entries related to entityID.
// It matches by learned_from, tags, and document-scope.
// Returns (entries, count); degrades gracefully when knowledgeSvc is nil.
func findRelatedKnowledge(
	knowledgeSvc *service.KnowledgeService,
	svc *service.IntelligenceService,
	entityID string,
) ([]map[string]any, int) {
	if knowledgeSvc == nil {
		return []map[string]any{}, 0
	}

	recs, err := knowledgeSvc.List(service.KnowledgeFilters{})
	if err != nil {
		return []map[string]any{}, 0
	}

	entityType := knowledgeEntityType(entityID)

	// Build set of doc paths that reference this entity (for scope matching).
	docPaths := map[string]bool{}
	if svc != nil {
		if docMatches, docErr := svc.FindByEntity(entityID); docErr == nil {
			for _, m := range docMatches {
				docPaths[m.DocPath] = true
			}
		}
	}

	seen := map[string]bool{}
	var result []map[string]any

	for _, rec := range recs {
		id, _ := rec.Fields["id"].(string)
		if id == "" {
			id = rec.ID
		}
		if seen[id] {
			continue
		}

		matched := false

		// FR-002: learned_from matches entity ID.
		if lf, _ := rec.Fields["learned_from"].(string); lf == entityID {
			matched = true
		}

		// FR-003: tags contain entity ID or entity type.
		if !matched {
			switch tags := rec.Fields["tags"].(type) {
			case []any:
				for _, t := range tags {
					if s, ok := t.(string); ok && (s == entityID || (entityType != "" && s == entityType)) {
						matched = true
						break
					}
				}
			case []string:
				for _, s := range tags {
					if s == entityID || (entityType != "" && s == entityType) {
						matched = true
						break
					}
				}
			}
		}

		// FR-004: scope matches a document that references the entity.
		if !matched && len(docPaths) > 0 {
			if scope, _ := rec.Fields["scope"].(string); scope != "" {
				for dp := range docPaths {
					if strings.HasPrefix(dp, scope) {
						matched = true
						break
					}
				}
			}
		}

		if !matched {
			continue
		}

		seen[id] = true

		topic, _ := rec.Fields["topic"].(string)
		recContent, _ := rec.Fields["content"].(string)
		status, _ := rec.Fields["status"].(string)
		confidence := asmFieldFloat(rec.Fields, "confidence")

		result = append(result, map[string]any{
			"id":         id,
			"topic":      topic,
			"content":    recContent,
			"confidence": confidence,
			"status":     status,
		})
	}

	if result == nil {
		result = []map[string]any{}
	}
	return result, len(result)
}

// knowledgeEntityType derives the entity type string from an entity ID prefix.
// Returns empty string for unrecognised prefixes.
func knowledgeEntityType(entityID string) string {
	switch {
	case strings.HasPrefix(entityID, "FEAT-"):
		return "feature"
	case strings.HasPrefix(entityID, "TASK-"):
		return "task"
	case strings.HasPrefix(entityID, "BUG-"):
		return "bug"
	default:
		return ""
	}
}
