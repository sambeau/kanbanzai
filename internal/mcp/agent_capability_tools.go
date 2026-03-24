package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"kanbanzai/internal/docint"
	"kanbanzai/internal/knowledge"
	"kanbanzai/internal/service"
)

// AgentCapabilityTools returns MCP tool definitions for agent-assisting capabilities:
// suggest_links, check_duplicates, and doc_extraction_guide.
func AgentCapabilityTools(
	entitySvc *service.EntityService,
	knowledgeSvc *service.KnowledgeService,
	intelligenceSvc *service.IntelligenceService,
) []server.ServerTool {
	return []server.ServerTool{
		suggestLinksTool(entitySvc, knowledgeSvc),
		checkDuplicatesTool(entitySvc, knowledgeSvc),
		docExtractionGuideTool(intelligenceSvc),
	}
}

func suggestLinksTool(entitySvc *service.EntityService, knowledgeSvc *service.KnowledgeService) server.ServerTool {
	tool := mcp.NewTool("suggest_links",
		mcp.WithDescription("Scan free text for entity ID patterns (FEAT-, TASK-, BUG-, DEC-, KE-, Plan IDs) and look up each found ID in the entity and knowledge stores. Returns a list of confirmed references with their entity type and title."),
		mcp.WithString("text", mcp.Description("Free text to scan for entity references"), mcp.Required()),
		mcp.WithString("scope", mcp.Description("Optional entity type filter (e.g. feature, task, bug, decision, plan, knowledge_entry)")),
	)
	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		text, err := request.RequireString("text")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		scopeFilter := request.GetString("scope", "")

		refs := knowledge.ScanEntityRefs(text)

		type linkResult struct {
			TextSpan     string `json:"text_span"`
			EntityID     string `json:"entity_id"`
			EntityType   string `json:"entity_type"`
			EntityTitle  string `json:"entity_title"`
			MatchQuality string `json:"match_quality"`
		}

		var links []linkResult

		for _, ref := range refs {
			entityType := knowledge.EntityTypeFromID(ref.Span)

			// Apply scope filter if provided
			if scopeFilter != "" && !strings.EqualFold(scopeFilter, entityType) {
				continue
			}

			var title string
			var found bool

			if entityType == "knowledge_entry" {
				rec, err := knowledgeSvc.Get(ref.Span)
				if err == nil {
					found = true
					title = agentTitleFromFields(rec.Fields)
				}
			} else {
				// Pass empty slug — EntityService.Get resolves by prefix internally
				result, err := entitySvc.Get(entityType, ref.Span, "")
				if err == nil {
					found = true
					title = agentTitleFromFields(result.State)
				}
			}

			if found {
				links = append(links, linkResult{
					TextSpan:     ref.Span,
					EntityID:     ref.Span,
					EntityType:   entityType,
					EntityTitle:  title,
					MatchQuality: "exact",
				})
			}
		}

		if links == nil {
			links = []linkResult{}
		}

		resp := map[string]any{
			"success": true,
			"count":   len(links),
			"links":   links,
		}
		return agentMapJSON(resp)
	}
	return server.ServerTool{Tool: tool, Handler: handler}
}

func checkDuplicatesTool(entitySvc *service.EntityService, knowledgeSvc *service.KnowledgeService) server.ServerTool {
	tool := mcp.NewTool("check_duplicates",
		mcp.WithDescription("Check whether an entity being created would duplicate an existing one. Computes Jaccard similarity between the candidate's title+summary and existing entities. Returns advisory candidates with similarity >= 0.5. Does NOT block creation."),
		mcp.WithString("entity_type", mcp.Description("Entity type to check: feature, task, bug, decision, plan, knowledge_entry"), mcp.Required()),
		mcp.WithString("title", mcp.Description("Title or topic of the candidate entity"), mcp.Required()),
		mcp.WithString("summary", mcp.Description("Optional summary of the candidate entity")),
	)
	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		entityType, err := request.RequireString("entity_type")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		title, err := request.RequireString("title")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		summary := request.GetString("summary", "")

		var existing []knowledge.ExistingEntity

		if strings.EqualFold(entityType, "knowledge_entry") {
			records, err := knowledgeSvc.List(service.KnowledgeFilters{IncludeRetired: false})
			if err != nil {
				return mcp.NewToolResultErrorFromErr("list knowledge entries failed", err), nil
			}
			for _, rec := range records {
				id, _ := rec.Fields["id"].(string)
				t, _ := rec.Fields["topic"].(string)
				c, _ := rec.Fields["content"].(string)
				existing = append(existing, knowledge.ExistingEntity{
					ID:      id,
					Type:    "knowledge_entry",
					Title:   t,
					Summary: c,
				})
			}
		} else {
			results, err := entitySvc.List(entityType)
			if err != nil {
				// Directory may not exist for this type yet — treat as empty
				results = nil
			}
			for _, r := range results {
				existing = append(existing, knowledge.ExistingEntity{
					ID:      r.ID,
					Type:    r.Type,
					Title:   agentTitleFromFields(r.State),
					Summary: agentSummaryFromFields(r.State),
				})
			}
		}

		candidates := knowledge.FindDuplicateCandidates(title, summary, existing, 0.5)

		type candidateResult struct {
			EntityID   string  `json:"entity_id"`
			EntityType string  `json:"entity_type"`
			Title      string  `json:"title"`
			Similarity float64 `json:"similarity"`
		}

		out := make([]candidateResult, 0, len(candidates))
		for _, c := range candidates {
			out = append(out, candidateResult{
				EntityID:   c.EntityID,
				EntityType: c.EntityType,
				Title:      c.Title,
				Similarity: c.Similarity,
			})
		}

		resp := map[string]any{
			"success":     true,
			"advisory":    true,
			"entity_type": entityType,
			"count":       len(out),
			"candidates":  out,
			"message":     fmt.Sprintf("Found %d candidate duplicate(s). This check is advisory — creation is not blocked.", len(out)),
		}
		return agentMapJSON(resp)
	}
	return server.ServerTool{Tool: tool, Handler: handler}
}

func docExtractionGuideTool(intelligenceSvc *service.IntelligenceService) server.ServerTool {
	tool := mcp.NewTool("doc_extraction_guide",
		mcp.WithDescription("Return an extraction guide for a document: its structural outline with section roles, entity references already found, and classification hints (if Layer 3 analysis has been run). Use this before extracting entities or decisions from a document."),
		mcp.WithString("document_id", mcp.Description("Document record ID"), mcp.Required()),
	)
	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		documentID, err := request.RequireString("document_id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		index, err := intelligenceSvc.GetDocumentIndex(documentID)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("get document index failed", err), nil
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

		resp := map[string]any{
			"success":          true,
			"document_id":      documentID,
			"document_path":    index.DocumentPath,
			"content_hash":     index.ContentHash,
			"classified":       index.Classified,
			"outline":          outline,
			"entity_refs":      entityRefs,
			"extraction_hints": extractionHints(index),
		}
		return agentMapJSON(resp)
	}
	return server.ServerTool{Tool: tool, Handler: handler}
}

// buildClassifiedRoleMap builds a lookup of sectionPath → role from Layer 3
// classifications (preferred) and Layer 2 conventional roles (fallback).
func buildClassifiedRoleMap(index *docint.DocumentIndex) map[string]string {
	roles := make(map[string]string)
	for _, cr := range index.ConventionalRoles {
		roles[cr.SectionPath] = cr.Role
	}
	// Layer 3 classifications override Layer 2
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

// flattenSections recursively converts a []docint.Section tree into a
// []sectionGuide tree, annotating each section with its classified role.
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

// extractionHints returns a list of actionable extraction prompts based on what
// the index contains. These guide an agent in extracting structured entities.
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

// agentMapJSON marshals a map to JSON and returns it as a tool result.
func agentMapJSON(v map[string]any) (*mcp.CallToolResult, error) {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return mcp.NewToolResultError("marshal result: " + err.Error()), nil
	}
	return mcp.NewToolResultText(string(data)), nil
}

// agentTitleFromFields extracts a human-readable title from an entity's fields map.
// Tries "title", then "summary", then "topic", then "content" (truncated), then "slug".
func agentTitleFromFields(fields map[string]any) string {
	for _, key := range []string{"title", "summary", "topic", "slug"} {
		if v, ok := fields[key]; ok {
			if s, ok := v.(string); ok && s != "" {
				return s
			}
		}
	}
	// Fallback: truncated content
	if v, ok := fields["content"]; ok {
		if s, ok := v.(string); ok && s != "" {
			if len(s) > 80 {
				return s[:80] + "..."
			}
			return s
		}
	}
	return ""
}

// agentSummaryFromFields extracts a summary string from an entity's fields map.
func agentSummaryFromFields(fields map[string]any) string {
	for _, key := range []string{"summary", "content", "description"} {
		if v, ok := fields[key]; ok {
			if s, ok := v.(string); ok {
				return s
			}
		}
	}
	return ""
}
