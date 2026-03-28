// Package mcp assembly.go — shared context assembly pipeline for next and handoff.
//
// This file contains the types and functions shared by next (Track F) and
// handoff (Track G) for assembling task context. The pipeline gathers spec
// sections from document intelligence, acceptance criteria from spec content,
// knowledge entries from the knowledge store, file paths from the task, and
// role conventions from context profiles.
//
// Both next and handoff call assembleContext with the same inputs. The
// difference is output format: next serialises the result to structured
// JSON/YAML; handoff renders it as a Markdown prompt.
//
// See specification §11.5 and implementation plan §3.4.
package mcp

import (
	"sort"
	"strings"

	kbzctx "kanbanzai/internal/context"
	"kanbanzai/internal/service"
)

const assemblyDefaultBudget = 30720

// ─── Types ────────────────────────────────────────────────────────────────────

// asmExperimentNudge represents an active workflow experiment included in
// context assembly per spec §8.4.
type asmExperimentNudge struct {
	decisionID string
	summary    string
}

// asmSpecSection represents a document section included in assembled context.
type asmSpecSection struct {
	document string
	section  string
	content  string
}

// asmKnowledgeEntry represents a knowledge entry included in assembled context.
type asmKnowledgeEntry struct {
	topic      string
	content    string
	scope      string
	confidence float64
	tier       int
}

// asmFileEntry represents a file path included in assembled context.
type asmFileEntry struct {
	path string
	note string
}

// asmTrimmedEntry records a context item removed to stay within the byte budget.
type asmTrimmedEntry struct {
	entryType string
	topic     string
	sizeBytes int
}

// assembledContext holds the result of the context assembly pipeline.
// Both next (structured JSON) and handoff (Markdown prompt) consume this
// intermediate representation.
type assembledContext struct {
	specSections       []asmSpecSection
	acceptanceCriteria []string
	knowledge          []asmKnowledgeEntry
	filesContext       []asmFileEntry
	constraints        []string
	roleProfile        string
	byteUsage          int
	byteBudget         int
	trimmed            []asmTrimmedEntry
	// experimentNudge lists active workflow-experiment decisions for agents to
	// reference when they encounter friction or success related to an experiment.
	// Not a knowledge entry; does not count against tier-3 budget (spec §8.4).
	experimentNudge []asmExperimentNudge
	// specFallbackPath is set when document intelligence returns no sections.
	// Contains the raw spec document path for the agent to read manually
	// (graceful degradation per spec §24.3).
	specFallbackPath string
}

// ─── Assembly entry point ─────────────────────────────────────────────────────

// asmInput holds parameters for context assembly.
type asmInput struct {
	taskState       map[string]any
	parentFeature   string
	role            string
	profileStore    *kbzctx.ProfileStore
	knowledgeSvc    *service.KnowledgeService
	intelligenceSvc *service.IntelligenceService
	docRecordSvc    *service.DocumentService
	entitySvc       *service.EntityService
}

// assembleContext gathers spec sections, acceptance criteria, knowledge,
// file context, and profile conventions. All sources are best-effort: errors
// produce empty results rather than failures.
//
// This is the shared pipeline described in spec §11.5 and implementation
// plan §3.4.
func assembleContext(input asmInput) assembledContext {
	var actx assembledContext
	actx.byteBudget = assemblyDefaultBudget

	// Role profile conventions.
	if input.profileStore != nil && input.role != "" {
		if profile, err := kbzctx.ResolveProfile(input.profileStore, input.role); err == nil {
			actx.roleProfile = input.role
			actx.constraints = append(actx.constraints, flattenConventions(profile.Conventions)...)
		}
	}

	// Spec/design sections from document intelligence, with automatic
	// Layer 1–2 parsing and graceful degradation.
	actx.specSections, actx.specFallbackPath = asmExtractSpecSections(
		input.parentFeature, input.intelligenceSvc, input.docRecordSvc,
	)

	// Extract acceptance criteria from spec sections.
	actx.acceptanceCriteria = asmExtractCriteria(actx.specSections)

	// Knowledge entries (Tier 2 + Tier 3), scoped to role or project.
	if input.knowledgeSvc != nil {
		actx.knowledge = asmLoadKnowledge(input.knowledgeSvc, input.role)
	}

	// File context from task's files_planned.
	actx.filesContext = asmExtractFiles(input.taskState)

	// Active workflow experiments (Phase 3 context nudge, spec §8.4).
	if input.entitySvc != nil {
		actx.experimentNudge = asmLoadExperimentNudge(input.entitySvc)
	}

	// Byte usage and trim if over budget.
	actx.byteUsage = asmByteCount(actx)
	if actx.byteUsage > actx.byteBudget {
		actx = asmTrimContext(actx)
	}

	return actx
}

// ─── Spec section extraction ──────────────────────────────────────────────────

// asmExtractSpecSections retrieves spec sections for a feature using document
// intelligence. Implements:
//   - Automatic Layer 1–2 parsing for unindexed documents (spec §24.4)
//   - Graceful degradation to raw document path when no sections are
//     extracted (spec §24.3)
//
// Returns:
//   - sections: extracted spec sections (empty if no index or no matches)
//   - fallbackPath: the raw document path when no sections were extracted,
//     enabling the agent to read the document manually
func asmExtractSpecSections(
	parentFeature string,
	intelligenceSvc *service.IntelligenceService,
	docRecordSvc *service.DocumentService,
) (sections []asmSpecSection, fallbackPath string) {
	if parentFeature == "" {
		return nil, ""
	}

	// Try document intelligence first.
	if intelligenceSvc != nil {
		sections = asmTraceEntitySections(parentFeature, intelligenceSvc)
		if len(sections) > 0 {
			return sections, ""
		}
	}

	// No sections found. Check for registered spec documents and attempt
	// automatic Layer 1–2 parsing (§24.4) if they haven't been indexed.
	if docRecordSvc != nil && intelligenceSvc != nil {
		specDocs, _ := docRecordSvc.ListDocuments(service.DocumentFilters{
			Owner: parentFeature,
			Type:  "specification",
		})

		// Auto-parse any unindexed documents.
		for _, doc := range specDocs {
			if _, err := intelligenceSvc.GetOutline(doc.ID); err != nil {
				// Not yet indexed — trigger synchronous Layer 1–2 parse.
				_, _ = intelligenceSvc.IngestDocument(doc.ID, doc.Path)
			}
		}

		// Also try design documents owned by the feature.
		designDocs, _ := docRecordSvc.ListDocuments(service.DocumentFilters{
			Owner: parentFeature,
			Type:  "design",
		})
		for _, doc := range designDocs {
			if _, err := intelligenceSvc.GetOutline(doc.ID); err != nil {
				_, _ = intelligenceSvc.IngestDocument(doc.ID, doc.Path)
			}
		}

		// Retry extraction after indexing.
		sections = asmTraceEntitySections(parentFeature, intelligenceSvc)
		if len(sections) > 0 {
			return sections, ""
		}

		// Graceful degradation (§24.3): return the document path so the
		// agent receives "read this document" guidance rather than nothing.
		if len(specDocs) > 0 {
			return nil, specDocs[0].Path
		}
		if len(designDocs) > 0 {
			return nil, designDocs[0].Path
		}
	}

	// Last resort: if we have a docRecordSvc but no intelligenceSvc,
	// still provide the document path for graceful degradation.
	if docRecordSvc != nil && intelligenceSvc == nil {
		specDocs, _ := docRecordSvc.ListDocuments(service.DocumentFilters{
			Owner: parentFeature,
			Type:  "specification",
		})
		if len(specDocs) > 0 {
			return nil, specDocs[0].Path
		}
		designDocs, _ := docRecordSvc.ListDocuments(service.DocumentFilters{
			Owner: parentFeature,
			Type:  "design",
		})
		if len(designDocs) > 0 {
			return nil, designDocs[0].Path
		}
	}

	return nil, ""
}

// asmTraceEntitySections calls IntelligenceService.TraceEntity and converts
// the results to assembly spec sections.
func asmTraceEntitySections(entityID string, svc *service.IntelligenceService) []asmSpecSection {
	matches, err := svc.TraceEntity(entityID)
	if err != nil || len(matches) == 0 {
		return nil
	}

	var sections []asmSpecSection
	for _, match := range matches {
		_, content, err := svc.GetSection(match.DocumentID, match.SectionPath)
		if err != nil || len(content) == 0 {
			continue
		}
		title := match.SectionTitle
		if title == "" {
			title = match.SectionPath
		}
		sections = append(sections, asmSpecSection{
			document: match.DocumentID,
			section:  title,
			content:  string(content),
		})
	}
	return sections
}

// ─── Acceptance criteria extraction ───────────────────────────────────────────

// asmExtractCriteria extracts testable acceptance criteria from spec sections.
//
// Heuristic rules:
//  1. From sections whose title contains "acceptance", "criteria", or
//     "requirement": include all non-empty bullet/numbered list items.
//  2. From all other sections: include bullet/numbered list items whose text
//     contains "MUST", "SHALL", "MUST NOT", or "SHALL NOT".
func asmExtractCriteria(sections []asmSpecSection) []string {
	var criteria []string
	seen := make(map[string]bool)

	addCriterion := func(s string) {
		s = strings.TrimSpace(s)
		if s != "" && !seen[s] {
			seen[s] = true
			criteria = append(criteria, s)
		}
	}

	for _, s := range sections {
		titleLower := strings.ToLower(s.section)
		isAcceptanceSection := strings.Contains(titleLower, "acceptance") ||
			strings.Contains(titleLower, "criteria") ||
			strings.Contains(titleLower, "requirement")

		for _, line := range strings.Split(s.content, "\n") {
			// Strip list marker to get the bare text.
			trimmed := strings.TrimSpace(line)
			text := trimmed
			for _, marker := range []string{"- ", "* ", "+ ", "• "} {
				if strings.HasPrefix(text, marker) {
					text = strings.TrimSpace(text[len(marker):])
					break
				}
			}
			// Numbered list: "1. ", "2. ", etc.
			if len(text) >= 3 && text[0] >= '0' && text[0] <= '9' {
				if idx := strings.Index(text, ". "); idx > 0 && idx < 4 {
					text = strings.TrimSpace(text[idx+2:])
				}
			}

			if text == "" || text == trimmed {
				// No list marker was stripped — not a list item; skip.
				continue
			}

			if isAcceptanceSection {
				addCriterion(text)
			} else {
				upper := strings.ToUpper(text)
				if strings.Contains(upper, " MUST ") || strings.HasSuffix(upper, " MUST") ||
					strings.Contains(upper, " SHALL ") || strings.HasSuffix(upper, " SHALL") ||
					strings.Contains(upper, " MUST NOT ") || strings.Contains(upper, " SHALL NOT ") {
					addCriterion(text)
				}
			}
		}
	}
	return criteria
}

// ─── Knowledge loading ────────────────────────────────────────────────────────

// asmLoadKnowledge loads knowledge entries scoped to the role or project.
// Returns entries sorted by confidence descending (highest first).
func asmLoadKnowledge(svc *service.KnowledgeService, role string) []asmKnowledgeEntry {
	var entries []asmKnowledgeEntry

	for _, tc := range []struct {
		tier    int
		minConf float64
	}{
		{2, 0.3},
		{3, 0.5},
	} {
		recs, err := svc.List(service.KnowledgeFilters{
			Tier:          tc.tier,
			MinConfidence: tc.minConf,
		})
		if err != nil {
			continue
		}
		for _, rec := range recs {
			scope, _ := rec.Fields["scope"].(string)
			if scope != "project" && scope != role {
				continue
			}
			topic, _ := rec.Fields["topic"].(string)
			content, _ := rec.Fields["content"].(string)
			conf := asmFieldFloat(rec.Fields, "confidence")
			tier := asmFieldInt(rec.Fields, "tier")
			entries = append(entries, asmKnowledgeEntry{
				topic:      topic,
				content:    content,
				scope:      scope,
				confidence: conf,
				tier:       tier,
			})
		}
	}

	// Highest confidence first.
	sort.SliceStable(entries, func(i, j int) bool {
		return entries[i].confidence > entries[j].confidence
	})
	return entries
}

// ─── File context extraction ──────────────────────────────────────────────────

// asmExtractFiles extracts file paths from a task's files_planned field.
func asmExtractFiles(taskState map[string]any) []asmFileEntry {
	if taskState == nil {
		return nil
	}
	var files []asmFileEntry
	switch fp := taskState["files_planned"].(type) {
	case []any:
		for _, item := range fp {
			if p, ok := item.(string); ok && p != "" {
				files = append(files, asmFileEntry{path: p})
			}
		}
	case []string:
		for _, p := range fp {
			if p != "" {
				files = append(files, asmFileEntry{path: p})
			}
		}
	}
	return files
}

// ─── Byte counting and trimming ───────────────────────────────────────────────

// flattenConventions converts the any-typed Conventions field to a []string.
// Handles both []interface{} (flat list) and map[string]interface{} (named sub-keys).
func flattenConventions(v any) []string {
	if v == nil {
		return nil
	}
	switch c := v.(type) {
	case []interface{}:
		out := make([]string, 0, len(c))
		for _, item := range c {
			if s, ok := item.(string); ok {
				out = append(out, s)
			}
		}
		return out
	case []string:
		return c
	case map[string]interface{}:
		var out []string
		for _, val := range c {
			if items, ok := val.([]interface{}); ok {
				for _, item := range items {
					if s, ok := item.(string); ok {
						out = append(out, s)
					}
				}
			}
		}
		return out
	}
	return nil
}

// asmByteCount estimates the byte size of assembled context.
// The overhead constants approximate JSON/YAML structural framing per entry.
func asmByteCount(actx assembledContext) int {
	total := 0
	for _, s := range actx.specSections {
		total += len(s.content) + len(s.document) + len(s.section) + 40
	}
	for _, ke := range actx.knowledge {
		total += len(ke.content) + len(ke.topic) + 30
	}
	for _, c := range actx.constraints {
		total += len(c) + 3
	}
	for _, f := range actx.filesContext {
		total += len(f.path) + 20
	}
	for _, cr := range actx.acceptanceCriteria {
		total += len(cr) + 3
	}
	if actx.specFallbackPath != "" {
		total += len(actx.specFallbackPath) + 30
	}
	for _, exp := range actx.experimentNudge {
		total += len(exp.decisionID) + len(exp.summary) + 30
	}
	return total
}

// asmTrimContext removes items to stay within the byte budget.
// Trim order per spec §11.5 (referencing Phase 4a §9.1):
// lowest-confidence Tier 3 knowledge first, then Tier 2 knowledge,
// then spec sections from end. Profile and task instructions are never trimmed.
func asmTrimContext(actx assembledContext) assembledContext {
	var t3, t2 []asmKnowledgeEntry
	for _, ke := range actx.knowledge {
		if ke.tier == 3 {
			t3 = append(t3, ke)
		} else {
			t2 = append(t2, ke)
		}
	}
	// Sort ascending so we cut lowest-confidence entries first.
	sort.SliceStable(t3, func(i, j int) bool { return t3[i].confidence < t3[j].confidence })
	sort.SliceStable(t2, func(i, j int) bool { return t2[i].confidence < t2[j].confidence })

	current := asmByteCount(actx)

	// Trim T3 knowledge first.
	for len(t3) > 0 && current > actx.byteBudget {
		cut := t3[0]
		t3 = t3[1:]
		sz := len(cut.content) + len(cut.topic) + 30
		current -= sz
		actx.trimmed = append(actx.trimmed, asmTrimmedEntry{
			entryType: "knowledge",
			topic:     cut.topic,
			sizeBytes: sz,
		})
	}

	// Trim T2 knowledge next.
	for len(t2) > 0 && current > actx.byteBudget {
		cut := t2[0]
		t2 = t2[1:]
		sz := len(cut.content) + len(cut.topic) + 30
		current -= sz
		actx.trimmed = append(actx.trimmed, asmTrimmedEntry{
			entryType: "knowledge",
			topic:     cut.topic,
			sizeBytes: sz,
		})
	}

	// Trim spec sections from end.
	for len(actx.specSections) > 0 && current > actx.byteBudget {
		cut := actx.specSections[len(actx.specSections)-1]
		actx.specSections = actx.specSections[:len(actx.specSections)-1]
		sz := len(cut.content) + len(cut.document) + len(cut.section) + 40
		current -= sz
		actx.trimmed = append(actx.trimmed, asmTrimmedEntry{
			entryType: "spec",
			topic:     cut.section,
			sizeBytes: sz,
		})
	}

	// Rebuild knowledge list: T2 then T3, both descending by confidence.
	sort.SliceStable(t3, func(i, j int) bool { return t3[i].confidence > t3[j].confidence })
	sort.SliceStable(t2, func(i, j int) bool { return t2[i].confidence > t2[j].confidence })
	actx.knowledge = append(t2, t3...)
	actx.byteUsage = current
	return actx
}

// ─── Experiment nudge loading ─────────────────────────────────────────────────

// asmLoadExperimentNudge loads active workflow-experiment decisions for the
// context nudge (spec §8.4). Returns nil when no active experiments exist.
func asmLoadExperimentNudge(entitySvc *service.EntityService) []asmExperimentNudge {
	decisions, err := entitySvc.List("decision")
	if err != nil {
		return nil
	}
	var nudges []asmExperimentNudge
	for _, dec := range decisions {
		status, _ := dec.State["status"].(string)
		if status != "accepted" {
			continue
		}
		if !hasTag(dec.State, "workflow-experiment") {
			continue
		}
		summary, _ := dec.State["summary"].(string)
		nudges = append(nudges, asmExperimentNudge{
			decisionID: dec.ID,
			summary:    summary,
		})
	}
	return nudges
}

// hasTag checks if an entity state map contains a specific tag.
func hasTag(state map[string]any, tag string) bool {
	switch tags := state["tags"].(type) {
	case []any:
		for _, t := range tags {
			if s, ok := t.(string); ok && s == tag {
				return true
			}
		}
	case []string:
		for _, t := range tags {
			if t == tag {
				return true
			}
		}
	}
	return false
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

// asmFieldFloat reads a float64 value from a fields map.
func asmFieldFloat(fields map[string]any, key string) float64 {
	v, ok := fields[key]
	if !ok {
		return 0
	}
	switch typed := v.(type) {
	case float64:
		return typed
	case int:
		return float64(typed)
	case int64:
		return float64(typed)
	}
	return 0
}

// asmFieldInt reads an int value from a fields map.
func asmFieldInt(fields map[string]any, key string) int {
	v, ok := fields[key]
	if !ok {
		return 0
	}
	switch typed := v.(type) {
	case int:
		return typed
	case float64:
		return int(typed)
	case int64:
		return int(typed)
	}
	return 0
}
