package cache

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite"
)

// CacheDir is the cache directory name within the instance root.
const CacheDir = "cache"

// CacheFile is the SQLite database filename.
const CacheFile = "kbz.db"

// EntityRow represents a cached entity record.
type EntityRow struct {
	EntityType string
	ID         string
	Slug       string
	Status     string
	Title      string
	Summary    string
	ParentRef  string
	FilePath   string
	FieldsJSON string
}

// Cache provides a local derived SQLite cache for entity query acceleration.
// All data in the cache is derived from canonical YAML files and can be
// rebuilt at any time. The cache is not canonical and must not be committed
// to Git.
type Cache struct {
	db   *sql.DB
	path string
}

// Open opens or creates a cache database at the given directory.
// The directory is created if it does not exist.
func Open(cacheDir string) (*Cache, error) {
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return nil, fmt.Errorf("create cache directory: %w", err)
	}

	dbPath := filepath.Join(cacheDir, CacheFile)
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open cache database: %w", err)
	}

	// Enable WAL mode for better concurrent read performance.
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		db.Close()
		return nil, fmt.Errorf("set WAL mode: %w", err)
	}

	c := &Cache{db: db, path: dbPath}
	if err := c.ensureSchema(); err != nil {
		db.Close()
		return nil, err
	}

	return c, nil
}

// Close closes the cache database.
func (c *Cache) Close() error {
	if c.db == nil {
		return nil
	}
	return c.db.Close()
}

// Path returns the path to the cache database file.
func (c *Cache) Path() string {
	return c.path
}

func (c *Cache) ensureSchema() error {
	schema := `
CREATE TABLE IF NOT EXISTS entities (
	entity_type TEXT NOT NULL,
	id          TEXT NOT NULL,
	slug        TEXT NOT NULL,
	status      TEXT NOT NULL DEFAULT '',
	title       TEXT NOT NULL DEFAULT '',
	summary     TEXT NOT NULL DEFAULT '',
	parent_ref  TEXT NOT NULL DEFAULT '',
	file_path   TEXT NOT NULL DEFAULT '',
	fields_json TEXT NOT NULL DEFAULT '{}',
	PRIMARY KEY (entity_type, id)
);

CREATE INDEX IF NOT EXISTS idx_entities_type ON entities(entity_type);
CREATE INDEX IF NOT EXISTS idx_entities_id ON entities(id);
CREATE INDEX IF NOT EXISTS idx_entities_status ON entities(entity_type, status);
CREATE INDEX IF NOT EXISTS idx_entities_parent ON entities(entity_type, parent_ref);
`
	if _, err := c.db.Exec(schema); err != nil {
		return fmt.Errorf("create cache schema: %w", err)
	}
	return nil
}

// Clear removes all cached data.
func (c *Cache) Clear() error {
	if _, err := c.db.Exec("DELETE FROM entities"); err != nil {
		return fmt.Errorf("clear cache: %w", err)
	}
	return nil
}

// Upsert inserts or replaces an entity in the cache.
func (c *Cache) Upsert(row EntityRow) error {
	const query = `
INSERT INTO entities (entity_type, id, slug, status, title, summary, parent_ref, file_path, fields_json)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(entity_type, id) DO UPDATE SET
	slug = excluded.slug,
	status = excluded.status,
	title = excluded.title,
	summary = excluded.summary,
	parent_ref = excluded.parent_ref,
	file_path = excluded.file_path,
	fields_json = excluded.fields_json
`
	_, err := c.db.Exec(query,
		row.EntityType, row.ID, row.Slug, row.Status,
		row.Title, row.Summary, row.ParentRef, row.FilePath,
		row.FieldsJSON,
	)
	if err != nil {
		return fmt.Errorf("upsert entity %s %s: %w", row.EntityType, row.ID, err)
	}
	return nil
}

// Delete removes an entity from the cache.
func (c *Cache) Delete(entityType, id string) error {
	_, err := c.db.Exec("DELETE FROM entities WHERE entity_type = ? AND id = ?", entityType, id)
	if err != nil {
		return fmt.Errorf("delete entity %s %s: %w", entityType, id, err)
	}
	return nil
}

// LookupByID finds an entity by type and ID, returning its slug and file path.
// Returns empty strings and false if not found.
func (c *Cache) LookupByID(entityType, id string) (slug, filePath string, found bool) {
	row := c.db.QueryRow(
		"SELECT slug, file_path FROM entities WHERE entity_type = ? AND id = ?",
		entityType, id,
	)
	if err := row.Scan(&slug, &filePath); err != nil {
		return "", "", false
	}
	return slug, filePath, true
}

// FindByID finds an entity by ID alone (across all types).
// Returns the entity type, slug, and file path if found.
func (c *Cache) FindByID(id string) (entityType, slug, filePath string, found bool) {
	row := c.db.QueryRow(
		"SELECT entity_type, slug, file_path FROM entities WHERE id = ?",
		id,
	)
	if err := row.Scan(&entityType, &slug, &filePath); err != nil {
		return "", "", "", false
	}
	return entityType, slug, filePath, true
}

// ListByType returns all cached entities of a given type.
func (c *Cache) ListByType(entityType string) ([]EntityRow, error) {
	rows, err := c.db.Query(
		"SELECT entity_type, id, slug, status, title, summary, parent_ref, file_path, fields_json FROM entities WHERE entity_type = ? ORDER BY id",
		entityType,
	)
	if err != nil {
		return nil, fmt.Errorf("list %s entities from cache: %w", entityType, err)
	}
	defer rows.Close()

	return scanRows(rows)
}

// ListAll returns all cached entities across all types.
func (c *Cache) ListAll() ([]EntityRow, error) {
	rows, err := c.db.Query(
		"SELECT entity_type, id, slug, status, title, summary, parent_ref, file_path, fields_json FROM entities ORDER BY entity_type, id",
	)
	if err != nil {
		return nil, fmt.Errorf("list all entities from cache: %w", err)
	}
	defer rows.Close()

	return scanRows(rows)
}

// Count returns the number of cached entities, optionally filtered by type.
func (c *Cache) Count(entityType string) (int, error) {
	var count int
	var err error
	if entityType == "" {
		err = c.db.QueryRow("SELECT COUNT(*) FROM entities").Scan(&count)
	} else {
		err = c.db.QueryRow("SELECT COUNT(*) FROM entities WHERE entity_type = ?", entityType).Scan(&count)
	}
	if err != nil {
		return 0, fmt.Errorf("count entities: %w", err)
	}
	return count, nil
}

// EntityExists checks whether an entity with the given type and ID exists.
func (c *Cache) EntityExists(entityType, id string) bool {
	var count int
	err := c.db.QueryRow(
		"SELECT COUNT(*) FROM entities WHERE entity_type = ? AND id = ?",
		entityType, id,
	).Scan(&count)
	return err == nil && count > 0
}

// Rebuild clears the cache and repopulates it from the provided entity records.
// Each record should include entity_type, id, slug, and the full fields map.
type RebuildRecord struct {
	EntityType string
	ID         string
	Slug       string
	FilePath   string
	Fields     map[string]any
}

// Rebuild clears the cache and repopulates it from canonical state.
func (c *Cache) Rebuild(records []RebuildRecord) (int, error) {
	tx, err := c.db.Begin()
	if err != nil {
		return 0, fmt.Errorf("begin rebuild transaction: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.Exec("DELETE FROM entities"); err != nil {
		return 0, fmt.Errorf("clear cache for rebuild: %w", err)
	}

	stmt, err := tx.Prepare(`
INSERT INTO entities (entity_type, id, slug, status, title, summary, parent_ref, file_path, fields_json)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
`)
	if err != nil {
		return 0, fmt.Errorf("prepare rebuild insert: %w", err)
	}
	defer stmt.Close()

	count := 0
	for _, rec := range records {
		status := stringFromFields(rec.Fields, "status")
		title := stringFromFields(rec.Fields, "title")
		summary := stringFromFields(rec.Fields, "summary")
		parentRef := extractParentRef(rec.EntityType, rec.Fields)

		fieldsJSON, err := json.Marshal(rec.Fields)
		if err != nil {
			return count, fmt.Errorf("marshal fields for %s %s: %w", rec.EntityType, rec.ID, err)
		}

		if _, err := stmt.Exec(
			rec.EntityType, rec.ID, rec.Slug, status,
			title, summary, parentRef, rec.FilePath,
			string(fieldsJSON),
		); err != nil {
			return count, fmt.Errorf("insert %s %s: %w", rec.EntityType, rec.ID, err)
		}
		count++
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit rebuild: %w", err)
	}

	return count, nil
}

// GetFields returns the full fields map for an entity, parsed from cached JSON.
func (c *Cache) GetFields(entityType, id string) (map[string]any, error) {
	var fieldsJSON string
	err := c.db.QueryRow(
		"SELECT fields_json FROM entities WHERE entity_type = ? AND id = ?",
		entityType, id,
	).Scan(&fieldsJSON)
	if err != nil {
		return nil, fmt.Errorf("get fields for %s %s: %w", entityType, id, err)
	}

	var fields map[string]any
	if err := json.Unmarshal([]byte(fieldsJSON), &fields); err != nil {
		return nil, fmt.Errorf("unmarshal fields for %s %s: %w", entityType, id, err)
	}

	return fields, nil
}

func scanRows(rows *sql.Rows) ([]EntityRow, error) {
	var result []EntityRow
	for rows.Next() {
		var r EntityRow
		if err := rows.Scan(
			&r.EntityType, &r.ID, &r.Slug, &r.Status,
			&r.Title, &r.Summary, &r.ParentRef, &r.FilePath,
			&r.FieldsJSON,
		); err != nil {
			return nil, fmt.Errorf("scan entity row: %w", err)
		}
		result = append(result, r)
	}
	return result, rows.Err()
}

func stringFromFields(fields map[string]any, key string) string {
	if fields == nil {
		return ""
	}
	v, ok := fields[key]
	if !ok {
		return ""
	}
	s, ok := v.(string)
	if !ok {
		return fmt.Sprint(v)
	}
	return s
}

func extractParentRef(entityType string, fields map[string]any) string {
	switch strings.ToLower(entityType) {
	case "feature":
		return stringFromFields(fields, "parent")
	case "task":
		return stringFromFields(fields, "parent_feature")
	case "bug":
		return stringFromFields(fields, "origin_feature")
	default:
		return ""
	}
}
