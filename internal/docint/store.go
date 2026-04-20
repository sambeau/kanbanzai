package docint

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/sambeau/kanbanzai/internal/fsutil"

	"gopkg.in/yaml.v3"
	_ "modernc.org/sqlite"
)

// IndexStore handles reading and writing document intelligence index files.
// Files live in .kbz/index/documents/ (per-document) and .kbz/index/ (graph, concepts).
type IndexStore struct {
	indexRoot string
	db        *sql.DB
}

// NewIndexStore creates an IndexStore rooted at the given directory.
func NewIndexStore(indexRoot string) *IndexStore {
	return &IndexStore{indexRoot: indexRoot}
}

// Close closes the SQLite database connection if one is open.
func (s *IndexStore) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

// DBPath returns the path to the SQLite database file.
func (s *IndexStore) DBPath() string {
	return filepath.Join(s.indexRoot, "docint.db")
}

// ensureDB opens the SQLite database if it is not already open.
func (s *IndexStore) ensureDB() error {
	if s.db != nil {
		return nil
	}
	dbPath := filepath.Join(s.indexRoot, "docint.db")
	return s.openDB(dbPath)
}

// openDB opens or creates a SQLite database at dbPath and applies the schema.
func (s *IndexStore) openDB(dbPath string) error {
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		return fmt.Errorf("create db directory: %w", err)
	}
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return fmt.Errorf("open sqlite: %w", err)
	}
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		db.Close()
		return fmt.Errorf("set WAL mode: %w", err)
	}
	if err := createSchema(db); err != nil {
		db.Close()
		return fmt.Errorf("create schema: %w", err)
	}
	s.db = db
	return nil
}

// createSchema creates all tables and indexes if they do not already exist.
func createSchema(db *sql.DB) error {
	stmts := []string{
		`CREATE VIRTUAL TABLE IF NOT EXISTS sections_fts USING fts5(
			title, content,
			document_id UNINDEXED,
			section_path UNINDEXED,
			tokenize = 'porter unicode61'
		)`,
		`CREATE TABLE IF NOT EXISTS edges (
			id        INTEGER PRIMARY KEY AUTOINCREMENT,
			from_id   TEXT NOT NULL,
			from_type TEXT NOT NULL,
			to_id     TEXT NOT NULL,
			to_type   TEXT NOT NULL,
			edge_type TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_edges_from ON edges(from_id, from_type)`,
		`CREATE INDEX IF NOT EXISTS idx_edges_to   ON edges(to_id, to_type)`,
		`CREATE INDEX IF NOT EXISTS idx_edges_type ON edges(edge_type)`,
		`CREATE TABLE IF NOT EXISTS entity_refs (
			id           INTEGER PRIMARY KEY AUTOINCREMENT,
			entity_id    TEXT NOT NULL,
			entity_type  TEXT NOT NULL,
			document_id  TEXT NOT NULL,
			section_path TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_entity_refs_entity   ON entity_refs(entity_id)`,
		`CREATE INDEX IF NOT EXISTS idx_entity_refs_document ON entity_refs(document_id)`,
	}
	for _, stmt := range stmts {
		if _, err := db.Exec(stmt); err != nil {
			preview := stmt
			if len(preview) > 40 {
				preview = preview[:40]
			}
			return fmt.Errorf("exec %q: %w", preview, err)
		}
	}
	return nil
}

// UpsertDocumentSQLite writes section FTS, edges, and entity refs for a document.
// It deletes existing rows for the document before inserting new ones.
func (s *IndexStore) UpsertDocumentSQLite(docID string, sections []Section, fileContent []byte, refs []EntityRef, edges []GraphEdge) error {
	if err := s.ensureDB(); err != nil {
		return fmt.Errorf("ensure db: %w", err)
	}
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	// Delete and re-insert FTS rows
	if _, err := tx.Exec(`DELETE FROM sections_fts WHERE document_id = ?`, docID); err != nil {
		return fmt.Errorf("delete sections_fts: %w", err)
	}
	if err := insertSectionsFTS(tx, docID, sections, fileContent); err != nil {
		return fmt.Errorf("insert sections_fts: %w", err)
	}

	// Delete and re-insert edges
	if _, err := tx.Exec(`DELETE FROM edges WHERE from_id = ? OR from_id LIKE ?`, docID, docID+"#%"); err != nil {
		return fmt.Errorf("delete edges: %w", err)
	}
	for _, e := range edges {
		if _, err := tx.Exec(
			`INSERT INTO edges(from_id, from_type, to_id, to_type, edge_type) VALUES (?,?,?,?,?)`,
			e.From, e.FromType, e.To, e.ToType, e.EdgeType,
		); err != nil {
			return fmt.Errorf("insert edge: %w", err)
		}
	}

	// Delete and re-insert entity refs
	if _, err := tx.Exec(`DELETE FROM entity_refs WHERE document_id = ?`, docID); err != nil {
		return fmt.Errorf("delete entity_refs: %w", err)
	}
	for _, ref := range refs {
		if _, err := tx.Exec(
			`INSERT INTO entity_refs(entity_id, entity_type, document_id, section_path) VALUES (?,?,?,?)`,
			ref.EntityID, ref.EntityType, docID, ref.SectionPath,
		); err != nil {
			return fmt.Errorf("insert entity_ref: %w", err)
		}
	}

	return tx.Commit()
}

// insertSectionsFTS recursively inserts sections into the sections_fts table.
func insertSectionsFTS(tx *sql.Tx, docID string, sections []Section, fileContent []byte) error {
	for _, sec := range sections {
		text := extractSectionText(fileContent, sec)
		if _, err := tx.Exec(
			`INSERT INTO sections_fts(title, content, document_id, section_path) VALUES (?,?,?,?)`,
			sec.Title, text, docID, sec.Path,
		); err != nil {
			return fmt.Errorf("insert section %s: %w", sec.Path, err)
		}
		if err := insertSectionsFTS(tx, docID, sec.Children, fileContent); err != nil {
			return err
		}
	}
	return nil
}

var (
	reHeading    = regexp.MustCompile(`(?m)^#{1,6}\s+`)
	reBoldItalic = regexp.MustCompile(`\*\*?|_{1,2}`)
	reInlineCode = regexp.MustCompile("`[^`]+`")
	reLink       = regexp.MustCompile(`\[([^\]]+)\]\([^\)]+\)`)
)

// extractSectionText extracts plain text from a section using its byte offsets.
func extractSectionText(fileContent []byte, sec Section) string {
	if len(fileContent) == 0 || sec.ByteCount == 0 {
		return sec.Title
	}
	start := sec.ByteOffset
	end := start + sec.ByteCount
	if start >= len(fileContent) {
		return sec.Title
	}
	if end > len(fileContent) {
		end = len(fileContent)
	}
	raw := string(fileContent[start:end])
	raw = reHeading.ReplaceAllString(raw, "")
	raw = reLink.ReplaceAllString(raw, "$1")
	raw = reInlineCode.ReplaceAllString(raw, "")
	raw = reBoldItalic.ReplaceAllString(raw, "")
	return strings.TrimSpace(raw)
}

// EntityRefWithDoc pairs an EntityRef with its containing document ID.
type EntityRefWithDoc struct {
	Ref        EntityRef
	DocumentID string
}

// QueryEntityRefsByEntityID returns all entity refs with document IDs for a given entity ID.
func (s *IndexStore) QueryEntityRefsByEntityID(entityID string) ([]EntityRefWithDoc, error) {
	if err := s.ensureDB(); err != nil {
		return nil, fmt.Errorf("ensure db: %w", err)
	}
	rows, err := s.db.Query(
		`SELECT entity_id, entity_type, document_id, section_path FROM entity_refs WHERE entity_id = ?`,
		entityID,
	)
	if err != nil {
		return nil, fmt.Errorf("query entity_refs: %w", err)
	}
	defer rows.Close()
	var results []EntityRefWithDoc
	for rows.Next() {
		var r EntityRefWithDoc
		if err := rows.Scan(&r.Ref.EntityID, &r.Ref.EntityType, &r.DocumentID, &r.Ref.SectionPath); err != nil {
			return nil, fmt.Errorf("scan entity_ref: %w", err)
		}
		results = append(results, r)
	}
	return results, rows.Err()
}

// QueryEdgesByToID returns all graph edges pointing to a given node ID from SQLite.
func (s *IndexStore) QueryEdgesByToID(toID string) ([]GraphEdge, error) {
	if err := s.ensureDB(); err != nil {
		return nil, fmt.Errorf("ensure db: %w", err)
	}
	rows, err := s.db.Query(
		`SELECT from_id, from_type, to_id, to_type, edge_type FROM edges WHERE to_id = ?`,
		toID,
	)
	if err != nil {
		return nil, fmt.Errorf("query edges: %w", err)
	}
	defer rows.Close()
	var edges []GraphEdge
	for rows.Next() {
		var e GraphEdge
		if err := rows.Scan(&e.From, &e.FromType, &e.To, &e.ToType, &e.EdgeType); err != nil {
			return nil, fmt.Errorf("scan edge: %w", err)
		}
		edges = append(edges, e)
	}
	return edges, rows.Err()
}

// SearchParams configures a full-text search query.
type SearchParams struct {
	Query   string
	Mode    string // "outline" (default), "summary", "full"
	Limit   int    // default 10, max 50
	DocType string // optional post-filter
	Role    string // optional post-filter
}

// SearchResult is a single search result.
type SearchResult struct {
	DocumentID   string  `json:"document_id"`
	DocumentPath string  `json:"document_path"`
	SectionPath  string  `json:"section_path"`
	SectionTitle string  `json:"section_title"`
	WordCount    int     `json:"word_count"`
	Role         *string `json:"role"` // null if unclassified
	BM25Score    float64 `json:"bm25_score"`
	Summary      string  `json:"summary,omitempty"` // summary mode
	Content      string  `json:"content,omitempty"` // full mode
}

// SearchSections executes a full-text search and returns ranked results.
func (s *IndexStore) SearchSections(params SearchParams) (total int, results []SearchResult, err error) {
	if err := s.ensureDB(); err != nil {
		return 0, nil, fmt.Errorf("ensure db: %w", err)
	}

	limit := params.Limit
	if limit <= 0 {
		limit = 10
	}
	if limit > 50 {
		limit = 50
	}

	// When filters active, overscan to ensure enough results survive filtering
	scanLimit := limit
	if params.DocType != "" || params.Role != "" {
		scanLimit = limit * 3
	}

	// Count total matches
	countRow := s.db.QueryRow(`SELECT count(*) FROM sections_fts WHERE sections_fts MATCH ?`, params.Query)
	if scanErr := countRow.Scan(&total); scanErr != nil {
		total = 0
	}

	rows, err := s.db.Query(
		`SELECT document_id, section_path, title, bm25(sections_fts) AS score
		 FROM sections_fts
		 WHERE sections_fts MATCH ?
		 ORDER BY score
		 LIMIT ?`,
		params.Query, scanLimit,
	)
	if err != nil {
		return 0, nil, fmt.Errorf("search fts: %w", err)
	}
	defer rows.Close()

	var candidates []SearchResult
	for rows.Next() {
		var r SearchResult
		if scanErr := rows.Scan(&r.DocumentID, &r.SectionPath, &r.SectionTitle, &r.BM25Score); scanErr != nil {
			return 0, nil, fmt.Errorf("scan search result: %w", scanErr)
		}
		candidates = append(candidates, r)
	}
	if rowsErr := rows.Err(); rowsErr != nil {
		return 0, nil, fmt.Errorf("rows error: %w", rowsErr)
	}

	// Enrich results and apply post-filters
	for _, r := range candidates {
		if len(results) >= limit {
			break
		}
		index, loadErr := s.LoadDocumentIndex(r.DocumentID)
		if loadErr != nil {
			continue
		}
		r.DocumentPath = index.DocumentPath

		if params.DocType != "" {
			if index.FrontMatter == nil || index.FrontMatter.Type != params.DocType {
				continue
			}
		}

		sec := findSectionByPath(index.Sections, r.SectionPath)
		if sec != nil {
			r.SectionTitle = sec.Title
			r.WordCount = sec.WordCount
		}

		r.Role = findSectionRole(index, r.SectionPath)

		if params.Role != "" {
			if r.Role == nil || *r.Role != params.Role {
				continue
			}
		}

		if params.Mode == "summary" && sec != nil {
			r.Summary = findSectionSummary(index, r.SectionPath)
		}
		if params.Mode == "full" && sec != nil {
			r.Content = loadSectionContent(index.DocumentPath, sec)
		}

		results = append(results, r)
	}

	return total, results, nil
}

// findSectionByPath recursively searches for a section by path.
func findSectionByPath(sections []Section, path string) *Section {
	for i := range sections {
		if sections[i].Path == path {
			return &sections[i]
		}
		if found := findSectionByPath(sections[i].Children, path); found != nil {
			return found
		}
	}
	return nil
}

// findSectionRole returns the role for a section, preferring Layer 3 over Layer 2.
func findSectionRole(index *DocumentIndex, sectionPath string) *string {
	for _, c := range index.Classifications {
		if c.SectionPath == sectionPath {
			role := c.Role
			return &role
		}
	}
	for _, cr := range index.ConventionalRoles {
		if cr.SectionPath == sectionPath {
			role := cr.Role
			return &role
		}
	}
	return nil
}

// findSectionSummary returns the agent-provided summary for a section, or empty if unclassified.
func findSectionSummary(index *DocumentIndex, sectionPath string) string {
	for _, c := range index.Classifications {
		if c.SectionPath == sectionPath {
			return c.Summary
		}
	}
	return ""
}

// loadSectionContent reads a section's raw content from the source file via byte offsets.
func loadSectionContent(docPath string, sec *Section) string {
	if docPath == "" || sec.ByteCount == 0 {
		return ""
	}
	data, err := os.ReadFile(docPath)
	if err != nil {
		return ""
	}
	start := sec.ByteOffset
	end := start + sec.ByteCount
	if start >= len(data) {
		return ""
	}
	if end > len(data) {
		end = len(data)
	}
	return string(data[start:end])
}

// CountFTSSectionsForDoc returns the number of FTS5 rows for a document (for testing and diagnostics).
func (s *IndexStore) CountFTSSectionsForDoc(docID string) (int, error) {
	if err := s.ensureDB(); err != nil {
		return 0, err
	}
	var n int
	err := s.db.QueryRow(`SELECT count(*) FROM sections_fts WHERE document_id = ?`, docID).Scan(&n)
	return n, err
}

// CountEdgesForDoc returns the number of edges originating from a document (for testing and diagnostics).
func (s *IndexStore) CountEdgesForDoc(docID string) (int, error) {
	if err := s.ensureDB(); err != nil {
		return 0, err
	}
	var n int
	err := s.db.QueryRow(`SELECT count(*) FROM edges WHERE from_id = ? OR from_id LIKE ?`, docID, docID+"#%").Scan(&n)
	return n, err
}

// CountEntityRefsForDoc returns the number of entity refs for a document (for testing and diagnostics).
func (s *IndexStore) CountEntityRefsForDoc(docID string) (int, error) {
	if err := s.ensureDB(); err != nil {
		return 0, err
	}
	var n int
	err := s.db.QueryRow(`SELECT count(*) FROM entity_refs WHERE document_id = ?`, docID).Scan(&n)
	return n, err
}

// ResetDB closes and clears the database reference so the next operation reinitialises it.
func (s *IndexStore) ResetDB() {
	if s.db != nil {
		s.db.Close()
		s.db = nil
	}
}

// SaveDocumentIndex saves a per-document index file.
func (s *IndexStore) SaveDocumentIndex(index *DocumentIndex) error {
	dir := filepath.Join(s.indexRoot, "documents")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create index directory: %w", err)
	}
	path := filepath.Join(dir, indexFileName(index.DocumentID))
	return writeYAMLFile(path, index)
}

// LoadDocumentIndex loads a per-document index file.
func (s *IndexStore) LoadDocumentIndex(docID string) (*DocumentIndex, error) {
	path := filepath.Join(s.indexRoot, "documents", indexFileName(docID))
	var index DocumentIndex
	if err := readYAMLFile(path, &index); err != nil {
		return nil, err
	}
	return &index, nil
}

// ListDocumentIndexes returns all indexed document IDs.
func (s *IndexStore) ListDocumentIndexes() ([]string, error) {
	dir := filepath.Join(s.indexRoot, "documents")
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var ids []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".yaml") {
			continue
		}
		id := strings.TrimSuffix(e.Name(), ".yaml")
		id = strings.ReplaceAll(id, "--", "/")
		ids = append(ids, id)
	}
	return ids, nil
}

// SaveGraph saves the document graph.
func (s *IndexStore) SaveGraph(graph *DocumentGraph) error {
	path := filepath.Join(s.indexRoot, "graph.yaml")
	return writeYAMLFile(path, graph)
}

// LoadGraph loads the document graph. Returns an empty graph if the file does not exist.
func (s *IndexStore) LoadGraph() (*DocumentGraph, error) {
	path := filepath.Join(s.indexRoot, "graph.yaml")
	var graph DocumentGraph
	if err := readYAMLFile(path, &graph); err != nil {
		if os.IsNotExist(err) {
			return &DocumentGraph{}, nil
		}
		return nil, err
	}
	return &graph, nil
}

// SaveConceptRegistry saves the concept registry.
func (s *IndexStore) SaveConceptRegistry(registry *ConceptRegistry) error {
	path := filepath.Join(s.indexRoot, "concepts.yaml")
	return writeYAMLFile(path, registry)
}

// LoadConceptRegistry loads the concept registry. Returns an empty registry if the file does not exist.
func (s *IndexStore) LoadConceptRegistry() (*ConceptRegistry, error) {
	path := filepath.Join(s.indexRoot, "concepts.yaml")
	var registry ConceptRegistry
	if err := readYAMLFile(path, &registry); err != nil {
		if os.IsNotExist(err) {
			return &ConceptRegistry{}, nil
		}
		return nil, err
	}
	return &registry, nil
}

// DocumentIndexExists checks if an index exists for a document.
func (s *IndexStore) DocumentIndexExists(docID string) bool {
	path := filepath.Join(s.indexRoot, "documents", indexFileName(docID))
	_, err := os.Stat(path)
	return err == nil
}

func indexFileName(docID string) string {
	safe := strings.ReplaceAll(docID, "/", "--")
	return safe + ".yaml"
}

func writeYAMLFile(path string, v any) error {
	data, err := yaml.Marshal(v)
	if err != nil {
		return fmt.Errorf("marshal YAML: %w", err)
	}

	// Ensure trailing newline
	if len(data) == 0 || data[len(data)-1] != '\n' {
		data = append(data, '\n')
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}

	return fsutil.WriteFileAtomic(path, data, 0o644)
}

func readYAMLFile(path string, v any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return yaml.Unmarshal(data, v)
}
