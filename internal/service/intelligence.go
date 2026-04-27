package service

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/sambeau/kanbanzai/internal/docint"
)

// EntityDocMatch represents a document section that references an entity.
type EntityDocMatch struct {
	DocumentID   string `json:"document_id"`
	DocPath      string `json:"doc_path"`
	SectionPath  string `json:"section_path"`
	SectionTitle string `json:"section_title"`
}

// ConceptMatch represents a document section related to a concept.
type ConceptMatch struct {
	DocumentID   string `json:"document_id"`
	SectionPath  string `json:"section_path"`
	SectionTitle string `json:"section_title"`
	Relationship string `json:"relationship"` // "introduces" or "uses"
}

// RoleMatch represents a document fragment with a classified role.
type RoleMatch struct {
	DocumentID   string `json:"document_id"`
	SectionPath  string `json:"section_path"`
	SectionTitle string `json:"section_title"`
	Role         string `json:"role"`
	Confidence   string `json:"confidence"`
	Summary      string `json:"summary,omitempty"`
}

// PendingDocEntry represents a document that is indexed but not yet classified.
type PendingDocEntry struct {
	ID           string
	SectionCount int
}

// countSections recursively counts all sections (including children) from Layer 1.
func countSections(sections []docint.Section) int {
	n := len(sections)
	for _, s := range sections {
		n += countSections(s.Children)
	}
	return n
}

// IntelligenceService coordinates document intelligence operations (Layers 1-4).
type IntelligenceService struct {
	indexStore *docint.IndexStore
	repoRoot   string
	wg         sync.WaitGroup // tracks background goroutines (FR-012 – FR-016)
}

// NewIntelligenceService creates a new IntelligenceService.
func NewIntelligenceService(indexRoot, repoRoot string) *IntelligenceService {
	return &IntelligenceService{
		indexStore: docint.NewIndexStore(indexRoot),
		repoRoot:   repoRoot,
	}
}

// IngestDocument runs Layers 1-2: reads the file, parses structure, extracts
// patterns, and saves the index. Returns the index for optional Layer 3 classification.
func (s *IntelligenceService) IngestDocument(docID, docPath string) (*docint.DocumentIndex, error) {
	fullPath := s.resolveDocPath(docPath)
	content, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, fmt.Errorf("read document file: %w", err)
	}

	// Layer 1: parse structural skeleton
	sections := docint.ParseStructure(content)

	// Layer 2: extract patterns
	extracted := docint.ExtractPatterns(content, sections)

	// Compute content hash
	hash := sha256.Sum256(content)
	contentHash := hex.EncodeToString(hash[:])

	now := time.Now().UTC()
	index := &docint.DocumentIndex{
		DocumentID:        docID,
		DocumentPath:      docPath,
		ContentHash:       contentHash,
		IndexedAt:         now,
		Sections:          sections,
		FrontMatter:       extracted.FrontMatter,
		EntityRefs:        extracted.EntityRefs,
		CrossDocLinks:     extracted.CrossDocLinks,
		ConventionalRoles: extracted.ConventionalRoles,
	}

	// Build and merge graph edges (Layer 4 — structural edges only)
	edges := docint.BuildGraphEdges(index)
	graph, err := s.indexStore.LoadGraph()
	if err != nil {
		return nil, fmt.Errorf("load graph: %w", err)
	}
	graph.Edges = docint.MergeGraphEdges(graph.Edges, docID, edges)
	graph.UpdatedAt = now
	if err := s.indexStore.SaveGraph(graph); err != nil {
		return nil, fmt.Errorf("save graph: %w", err)
	}

	// Persist the index
	if err := s.indexStore.SaveDocumentIndex(index); err != nil {
		return nil, fmt.Errorf("save document index: %w", err)
	}

	// Dual-write to SQLite for FTS and fast queries (graceful degradation)
	if sqlErr := s.indexStore.UpsertDocumentSQLite(docID, sections, content, extracted.EntityRefs, edges); sqlErr != nil {
		log.Printf("warning: SQLite write failed for %s: %v", docID, sqlErr)
	}

	return index, nil
}

// ClassifyDocument validates and applies Layer 3 classifications to an existing index.
func (s *IntelligenceService) ClassifyDocument(submission docint.ClassificationSubmission) error {
	index, err := s.indexStore.LoadDocumentIndex(submission.DocumentID)
	if err != nil {
		return fmt.Errorf("load document index: %w", err)
	}

	// Validate the submission against the current index
	errs := docint.ValidateClassifications(index, submission)
	if len(errs) > 0 {
		msgs := make([]string, len(errs))
		for i, e := range errs {
			msgs[i] = e.Error()
		}
		return fmt.Errorf("classification validation failed: %s", strings.Join(msgs, "; "))
	}

	// Apply classifications
	docint.ApplyClassifications(index, submission)

	// Update concept registry
	registry, err := s.indexStore.LoadConceptRegistry()
	if err != nil {
		return fmt.Errorf("load concept registry: %w", err)
	}
	docint.RemoveDocumentFromRegistry(registry, submission.DocumentID)
	docint.UpdateConceptRegistry(registry, submission.DocumentID, submission.Classifications)
	registry.UpdatedAt = time.Now().UTC()
	if err := s.indexStore.SaveConceptRegistry(registry); err != nil {
		return fmt.Errorf("save concept registry: %w", err)
	}

	// Rebuild graph edges with classification data
	edges := docint.BuildGraphEdges(index)
	graph, err := s.indexStore.LoadGraph()
	if err != nil {
		return fmt.Errorf("load graph: %w", err)
	}
	graph.Edges = docint.MergeGraphEdges(graph.Edges, submission.DocumentID, edges)
	graph.UpdatedAt = time.Now().UTC()
	if err := s.indexStore.SaveGraph(graph); err != nil {
		return fmt.Errorf("save graph: %w", err)
	}

	// Save updated index
	if err := s.indexStore.SaveDocumentIndex(index); err != nil {
		return fmt.Errorf("save document index: %w", err)
	}

	// Dual-write: update SQLite edges and entity refs after classification (graceful degradation).
	var fileContent []byte
	if index.DocumentPath != "" {
		fullPath := s.resolveDocPath(index.DocumentPath)
		fileContent, _ = os.ReadFile(fullPath)
	}
	if sqlErr := s.indexStore.UpsertDocumentSQLite(submission.DocumentID, index.Sections, fileContent, index.EntityRefs, edges); sqlErr != nil {
		log.Printf("warning: SQLite write failed after classify for %s: %v", submission.DocumentID, sqlErr)
	}

	return nil
}

// GetOutline returns the structural outline (Layer 1 sections) for a document.
func (s *IntelligenceService) GetOutline(docID string) ([]docint.Section, error) {
	index, err := s.indexStore.LoadDocumentIndex(docID)
	if err != nil {
		return nil, fmt.Errorf("load document index: %w", err)
	}
	s.wg.Add(1)
	go func() { defer s.wg.Done(); s.touchDocumentAccess(docID, "") }() // FR-012
	return index.Sections, nil
}

// GetSection returns a specific section's metadata and raw content from the file.
func (s *IntelligenceService) GetSection(docID, sectionPath string) (*docint.Section, []byte, error) {
	index, err := s.indexStore.LoadDocumentIndex(docID)
	if err != nil {
		return nil, nil, fmt.Errorf("load document index: %w", err)
	}

	section := findSection(index.Sections, sectionPath)
	if section == nil {
		return nil, nil, fmt.Errorf("section %q not found in document %s", sectionPath, docID)
	}

	// Read the document file and extract the section content
	fullPath := s.resolveDocPath(index.DocumentPath)
	content, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, nil, fmt.Errorf("read document file: %w", err)
	}

	start := section.ByteOffset
	end := start + section.ByteCount
	if end > len(content) {
		end = len(content)
	}
	if start > len(content) {
		return section, nil, nil
	}

	s.wg.Add(1)
	go func() { defer s.wg.Done(); s.touchDocumentAccess(docID, sectionPath) }() // FR-014
	return section, content[start:end], nil
}

// FindByEntity finds all sections across all documents that reference an entity.
// It queries the SQLite entity_refs table for performance; falls back to YAML scan on error.
func (s *IntelligenceService) FindByEntity(entityID string) ([]EntityDocMatch, error) {
	refs, sqlErr := s.indexStore.QueryEntityRefsByEntityID(entityID)
	if sqlErr == nil {
		var matches []EntityDocMatch
		for _, r := range refs {
			index, err := s.indexStore.LoadDocumentIndex(r.DocumentID)
			if err != nil {
				continue
			}
			title := sectionTitle(index.Sections, r.Ref.SectionPath)
			matches = append(matches, EntityDocMatch{
				DocumentID:   r.DocumentID,
				DocPath:      index.DocumentPath,
				SectionPath:  r.Ref.SectionPath,
				SectionTitle: title,
			})
		}
		s.wg.Add(1)
		go func(m []EntityDocMatch) { defer s.wg.Done(); s.touchDistinctDocuments(m) }(matches) // FR-015
		return matches, nil
	}
	log.Printf("warning: SQLite FindByEntity fallback for %s: %v", entityID, sqlErr)

	// Fallback: scan YAML indexes
	docIDs, err := s.indexStore.ListDocumentIndexes()
	if err != nil {
		return nil, fmt.Errorf("list document indexes: %w", err)
	}
	var matches []EntityDocMatch
	for _, id := range docIDs {
		index, err := s.indexStore.LoadDocumentIndex(id)
		if err != nil {
			continue
		}
		for _, ref := range index.EntityRefs {
			if ref.EntityID == entityID {
				title := sectionTitle(index.Sections, ref.SectionPath)
				matches = append(matches, EntityDocMatch{
					DocumentID:   index.DocumentID,
					DocPath:      index.DocumentPath,
					SectionPath:  ref.SectionPath,
					SectionTitle: title,
				})
			}
		}
	}
	s.wg.Add(1)
	go func(m []EntityDocMatch) { defer s.wg.Done(); s.touchDistinctDocuments(m) }(matches) // FR-015
	return matches, nil
}

// FindByConcept finds all sections that introduce or use a concept.
func (s *IntelligenceService) FindByConcept(concept string) ([]ConceptMatch, error) {
	registry, err := s.indexStore.LoadConceptRegistry()
	if err != nil {
		return nil, fmt.Errorf("load concept registry: %w", err)
	}

	c := docint.FindConcept(registry, concept)
	if c == nil {
		return nil, nil
	}

	var matches []ConceptMatch

	for _, ref := range c.IntroducedIn {
		docID, sectionPath := parseSectionRef(ref)
		if docID == "" {
			continue
		}
		title := s.lookupSectionTitle(docID, sectionPath)
		matches = append(matches, ConceptMatch{
			DocumentID:   docID,
			SectionPath:  sectionPath,
			SectionTitle: title,
			Relationship: "introduces",
		})
	}

	for _, ref := range c.UsedIn {
		docID, sectionPath := parseSectionRef(ref)
		if docID == "" {
			continue
		}
		title := s.lookupSectionTitle(docID, sectionPath)
		matches = append(matches, ConceptMatch{
			DocumentID:   docID,
			SectionPath:  sectionPath,
			SectionTitle: title,
			Relationship: "uses",
		})
	}

	// FR-015: increment access counters for distinct documents in non-empty results.
	if len(matches) > 0 {
		docIDs := make([]string, 0, len(matches))
		seen := make(map[string]bool)
		for _, m := range matches {
			if !seen[m.DocumentID] {
				seen[m.DocumentID] = true
				docIDs = append(docIDs, m.DocumentID)
			}
		}
		s.wg.Add(1)
		go func(ids []string) {
			defer s.wg.Done()
			for _, id := range ids {
				s.touchDocumentAccess(id, "")
			}
		}(docIDs)
	}

	return matches, nil
}

// FindByRole finds all fragments with a given role across the corpus.
// If scope is provided, filters to that document ID.
func (s *IntelligenceService) FindByRole(role string, scope string) ([]RoleMatch, error) {
	docIDs, err := s.indexStore.ListDocumentIndexes()
	if err != nil {
		return nil, fmt.Errorf("list document indexes: %w", err)
	}

	var matches []RoleMatch
	for _, id := range docIDs {
		if scope != "" && id != scope {
			continue
		}
		index, err := s.indexStore.LoadDocumentIndex(id)
		if err != nil {
			continue
		}

		// Deduplicate by (DocumentID, SectionPath) — Layer 3 takes precedence
		type key struct {
			docID       string
			sectionPath string
		}
		seen := make(map[key]bool)

		// Check Layer 3 agent classifications
		for _, c := range index.Classifications {
			if c.Role == role {
				k := key{index.DocumentID, c.SectionPath}
				seen[k] = true
				title := sectionTitle(index.Sections, c.SectionPath)
				matches = append(matches, RoleMatch{
					DocumentID:   index.DocumentID,
					SectionPath:  c.SectionPath,
					SectionTitle: title,
					Role:         c.Role,
					Confidence:   c.Confidence,
					Summary:      c.Summary,
				})
			}
		}

		// Also check Layer 2 conventional roles (skip if Layer 3 already matched)
		for _, cr := range index.ConventionalRoles {
			if cr.Role == role {
				k := key{index.DocumentID, cr.SectionPath}
				if seen[k] {
					continue // Skip duplicates — Layer 3 takes precedence
				}
				title := sectionTitle(index.Sections, cr.SectionPath)
				matches = append(matches, RoleMatch{
					DocumentID:   index.DocumentID,
					SectionPath:  cr.SectionPath,
					SectionTitle: title,
					Role:         cr.Role,
					Confidence:   cr.Confidence,
				})
			}
		}
	}

	// FR-015: increment access counters for distinct documents in non-empty results.
	if len(matches) > 0 {
		seenDocs := make(map[string]bool)
		var distinctIDs []string
		for _, m := range matches {
			if !seenDocs[m.DocumentID] {
				seenDocs[m.DocumentID] = true
				distinctIDs = append(distinctIDs, m.DocumentID)
			}
		}
		s.wg.Add(1)
		go func(ids []string) {
			defer s.wg.Done()
			for _, id := range ids {
				s.touchDocumentAccess(id, "")
			}
		}(distinctIDs)
	}

	return matches, nil
}

// GetPendingClassification returns entries for documents that are indexed but not classified.
// Each entry includes the document ID and its Layer 1 section count.
func (s *IntelligenceService) GetPendingClassification() ([]PendingDocEntry, error) {
	docIDs, err := s.indexStore.ListDocumentIndexes()
	if err != nil {
		return nil, fmt.Errorf("list document indexes: %w", err)
	}

	var pending []PendingDocEntry
	for _, id := range docIDs {
		index, err := s.indexStore.LoadDocumentIndex(id)
		if err != nil {
			continue
		}
		if !index.Classified {
			pending = append(pending, PendingDocEntry{
				ID:           id,
				SectionCount: countSections(index.Sections),
			})
		}
	}

	return pending, nil
}

// TraceEntity traces an entity through the refinement chain: finds all documents
// that reference it, ordered by document type (design → spec → dev-plan).
func (s *IntelligenceService) TraceEntity(entityID string) ([]EntityDocMatch, error) {
	matches, err := s.FindByEntity(entityID)
	if err != nil {
		return nil, err
	}

	// Build a type-order map for sorting
	typeOrder := map[string]int{
		"design":        0,
		"specification": 1, // legacy alias
		"spec":          1,
		"dev-plan":      2,
		"research":      3,
		"report":        4,
		"policy":        5,
	}

	// Load front matter to get document types for ordering
	type matchWithType struct {
		match   EntityDocMatch
		docType string
	}
	var typed []matchWithType
	for _, m := range matches {
		index, err := s.indexStore.LoadDocumentIndex(m.DocumentID)
		if err != nil {
			typed = append(typed, matchWithType{match: m, docType: ""})
			continue
		}
		dt := ""
		if index.FrontMatter != nil && index.FrontMatter.Type != "" {
			dt = index.FrontMatter.Type
		}
		typed = append(typed, matchWithType{match: m, docType: dt})
	}

	// Sort by document type order (stable, preserving original order within same type)
	for i := 1; i < len(typed); i++ {
		for j := i; j > 0; j-- {
			oi := typeOrder[typed[j].docType]
			oj := typeOrder[typed[j-1].docType]
			if oi < oj {
				typed[j], typed[j-1] = typed[j-1], typed[j]
			} else {
				break
			}
		}
	}

	sorted := make([]EntityDocMatch, len(typed))
	for i, t := range typed {
		sorted[i] = t.match
	}
	return sorted, nil
}

// AnalyzeGaps determines what document types are missing for a feature.
// It checks whether design, specification, and dev-plan documents exist.
func (s *IntelligenceService) AnalyzeGaps(featureID string, docSvc *DocumentService) ([]string, error) {
	docs, err := docSvc.ListDocumentsByOwner(featureID)
	if err != nil {
		return nil, fmt.Errorf("list documents for %s: %w", featureID, err)
	}

	existingTypes := make(map[string]bool)
	for _, d := range docs {
		existingTypes[d.Type] = true
	}

	// The expected document types for a feature
	expected := []string{"design", "spec", "dev-plan"}
	var gaps []string
	for _, t := range expected {
		if !existingTypes[t] {
			gaps = append(gaps, t)
		}
	}

	return gaps, nil
}

// GetDocumentIndex returns the full document index for a given document ID.
// This is used by doc_extraction_guide to access sections, entity refs, and classifications.
func (s *IntelligenceService) GetDocumentIndex(docID string) (*docint.DocumentIndex, error) {
	index, err := s.indexStore.LoadDocumentIndex(docID)
	if err != nil {
		return nil, fmt.Errorf("load document index: %w", err)
	}
	s.wg.Add(1)
	go func() { defer s.wg.Done(); s.touchDocumentAccess(docID, "") }() // FR-013
	return index, nil
}

// GetImpact finds all graph edges pointing to a given section ID.
// It queries the SQLite edges table for performance; falls back to YAML scan on error.
func (s *IntelligenceService) GetImpact(sectionID string) ([]docint.GraphEdge, error) {
	edges, sqlErr := s.indexStore.QueryEdgesByToID(sectionID)
	if sqlErr == nil {
		return edges, nil
	}
	log.Printf("warning: SQLite GetImpact fallback for %s: %v", sectionID, sqlErr)

	// Fallback: scan graph.yaml
	graph, err := s.indexStore.LoadGraph()
	if err != nil {
		return nil, fmt.Errorf("load graph: %w", err)
	}
	var impacted []docint.GraphEdge
	for _, edge := range graph.Edges {
		if edge.To == sectionID {
			impacted = append(impacted, edge)
		}
	}
	return impacted, nil
}

// Search executes a full-text search over section content.
func (s *IntelligenceService) Search(params docint.SearchParams) (int, []docint.SearchResult, error) {
	total, results, err := s.indexStore.SearchSections(params)
	// FR-016: increment access counters for distinct documents in non-empty results.
	if err == nil && len(results) > 0 {
		seen := make(map[string]bool)
		var distinctIDs []string
		for _, r := range results {
			if !seen[r.DocumentID] {
				seen[r.DocumentID] = true
				distinctIDs = append(distinctIDs, r.DocumentID)
			}
		}
		s.wg.Add(1)
		go func(ids []string) {
			defer s.wg.Done()
			for _, id := range ids {
				s.touchDocumentAccess(id, "")
			}
		}(distinctIDs)
	}
	return total, results, err
}

// RebuildStats summarises the result of a full index rebuild.
type RebuildStats struct {
	Documents   int
	Edges       int
	EntityRefs  int
	FTSSections int
	Failed      int
}

// RebuildIndex deletes the SQLite database and rebuilds it from all per-document YAML indexes.
func (s *IntelligenceService) RebuildIndex() (RebuildStats, error) {
	var stats RebuildStats

	// Delete and reset the database
	dbPath := s.indexStore.DBPath()
	if err := os.Remove(dbPath); err != nil && !os.IsNotExist(err) {
		return stats, fmt.Errorf("remove db: %w", err)
	}
	// Also remove WAL and shared-memory sidecar files (they belong to the deleted DB).
	os.Remove(dbPath + "-wal") //nolint:errcheck
	os.Remove(dbPath + "-shm") //nolint:errcheck
	s.indexStore.ResetDB()

	// Enumerate all per-document YAML index files
	docIDs, err := s.indexStore.ListDocumentIndexes()
	if err != nil {
		return stats, fmt.Errorf("list document indexes: %w", err)
	}

	for _, docID := range docIDs {
		index, err := s.indexStore.LoadDocumentIndex(docID)
		if err != nil {
			continue
		}

		// Read file content for FTS
		var fileContent []byte
		if index.DocumentPath != "" {
			fullPath := s.resolveDocPath(index.DocumentPath)
			fileContent, _ = os.ReadFile(fullPath)
		}

		edges := docint.BuildGraphEdges(index)

		if err := s.indexStore.UpsertDocumentSQLite(docID, index.Sections, fileContent, index.EntityRefs, edges); err != nil {
			log.Printf("warning: rebuild skip %s: %v", docID, err)
			stats.Failed++
			continue
		}

		sectionCount, _ := s.indexStore.CountFTSSectionsForDoc(docID)
		refCount, _ := s.indexStore.CountEntityRefsForDoc(docID)
		edgeCount, _ := s.indexStore.CountEdgesForDoc(docID)

		stats.Documents++
		stats.FTSSections += sectionCount
		stats.EntityRefs += refCount
		stats.Edges += edgeCount
	}

	return stats, nil
}

// Wait waits for all background goroutine operations to complete without
// closing the index store. Use in tests to ensure counter updates are
// persisted before making assertions.
func (s *IntelligenceService) Wait() {
	s.wg.Wait()
}

// Close waits for all background goroutines to finish, then closes the
// underlying index store (and its SQLite connection).
func (s *IntelligenceService) Close() error {
	s.wg.Wait()
	return s.indexStore.Close()
}

// GetClassifications returns all classification entries for the given document.
// Each entry pairs the document's current content_hash with one Classification.
// Returns an empty (non-nil) slice when the document has no classification entries
// or has not been indexed — never returns an error for the "no entries" case (REQ-008).
func (s *IntelligenceService) GetClassifications(docID string) ([]docint.ClassificationEntry, error) {
	index, err := s.indexStore.LoadDocumentIndex(docID)
	if err != nil {
		// Document not indexed yet — treat as zero classifications (REQ-002, REQ-008).
		return []docint.ClassificationEntry{}, nil
	}
	if len(index.Classifications) == 0 {
		return []docint.ClassificationEntry{}, nil
	}
	entries := make([]docint.ClassificationEntry, len(index.Classifications))
	for i, c := range index.Classifications {
		entries[i] = docint.ClassificationEntry{
			ContentHash:    index.ContentHash,
			Classification: c,
		}
	}
	return entries, nil
}

// touchDocumentAccess increments AccessCount and updates LastAccessedAt for
// the given document. If sectionPath is non-empty, also increments the
// SectionAccess entry for that path. Errors are silently absorbed (NFR-002).
// Must be called in a goroutine for non-blocking behaviour (NFR-001).
func (s *IntelligenceService) touchDocumentAccess(docID string, sectionPath string) {
	index, err := s.indexStore.LoadDocumentIndex(docID)
	if err != nil {
		return
	}
	now := time.Now().UTC()
	index.AccessCount++
	index.LastAccessedAt = &now

	if sectionPath != "" {
		if index.SectionAccess == nil {
			index.SectionAccess = make(map[string]docint.SectionAccessInfo)
		}
		info := index.SectionAccess[sectionPath]
		info.AccessCount++
		info.LastAccessedAt = &now
		index.SectionAccess[sectionPath] = info
	}

	s.indexStore.SaveDocumentIndex(index) //nolint:errcheck // best-effort
}

// touchDistinctDocuments increments access counters for every distinct
// DocumentID found in a []EntityDocMatch result set (FR-015).
func (s *IntelligenceService) touchDistinctDocuments(matches []EntityDocMatch) {
	seen := make(map[string]bool)
	for _, m := range matches {
		if !seen[m.DocumentID] {
			seen[m.DocumentID] = true
			s.touchDocumentAccess(m.DocumentID, "")
		}
	}
}

// resolveDocPath resolves a document path relative to the repo root.
func (s *IntelligenceService) resolveDocPath(docPath string) string {
	if s.repoRoot == "" || s.repoRoot == "." {
		return docPath
	}
	return s.repoRoot + "/" + docPath
}

// lookupSectionTitle loads an index and returns the title for a section path.
func (s *IntelligenceService) lookupSectionTitle(docID, sectionPath string) string {
	index, err := s.indexStore.LoadDocumentIndex(docID)
	if err != nil {
		return ""
	}
	return sectionTitle(index.Sections, sectionPath)
}

// findSection recursively searches for a section by path.
func findSection(sections []docint.Section, path string) *docint.Section {
	for i := range sections {
		if sections[i].Path == path {
			return &sections[i]
		}
		if found := findSection(sections[i].Children, path); found != nil {
			return found
		}
	}
	return nil
}

// sectionTitle returns the title for a section path, or empty string if not found.
func sectionTitle(sections []docint.Section, path string) string {
	s := findSection(sections, path)
	if s == nil {
		return ""
	}
	return s.Title
}

// parseSectionRef splits a "docID#sectionPath" reference into its parts.
func parseSectionRef(ref string) (docID, sectionPath string) {
	idx := strings.Index(ref, "#")
	if idx < 0 {
		return ref, ""
	}
	return ref[:idx], ref[idx+1:]
}
