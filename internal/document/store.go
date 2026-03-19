package document

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"kanbanzai/internal/fsutil"
)

// DocStore handles filesystem storage of documents.
type DocStore struct {
	root string // the docs directory path, e.g. ".kbz/docs"
}

// NewDocStore creates a new DocStore rooted at the given directory.
func NewDocStore(root string) *DocStore {
	return &DocStore{root: root}
}

// Write writes a document to disk. The document's Meta.ID and Meta.Title must be set.
// Returns the path where the document was written.
func (s *DocStore) Write(doc Document) (string, error) {
	if strings.TrimSpace(doc.Meta.ID) == "" {
		return "", errors.New("document id is required")
	}
	if strings.TrimSpace(doc.Meta.Title) == "" {
		return "", errors.New("document title is required")
	}
	if !ValidDocType(string(doc.Meta.Type)) {
		return "", fmt.Errorf("invalid document type: %s", doc.Meta.Type)
	}

	typeDir := filepath.Join(s.root, string(doc.Meta.Type))
	if err := os.MkdirAll(typeDir, 0o755); err != nil {
		return "", fmt.Errorf("create document directory: %w", err)
	}

	content := marshalDocument(doc)

	path := filepath.Join(typeDir, docFileName(doc.Meta))
	if err := fsutil.WriteFileAtomic(path, []byte(content), 0o644); err != nil {
		return "", fmt.Errorf("write document file: %w", err)
	}

	return path, nil
}

// Load reads a document from disk by type, id, and slug.
func (s *DocStore) Load(docType DocType, id, slug string) (Document, error) {
	if !ValidDocType(string(docType)) {
		return Document{}, fmt.Errorf("invalid document type: %s", docType)
	}

	path := filepath.Join(s.root, string(docType), docFileName(DocMeta{ID: id, Title: slug}))
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Document{}, fmt.Errorf("document not found: %s/%s-%s", docType, id, slug)
		}
		return Document{}, fmt.Errorf("read document file: %w", err)
	}

	return unmarshalDocument(string(data))
}

// LoadByPath reads a document from disk by its full path.
func (s *DocStore) LoadByPath(path string) (Document, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Document{}, fmt.Errorf("read document file: %w", err)
	}

	return unmarshalDocument(string(data))
}

// List returns paths of all documents of a given type.
func (s *DocStore) List(docType DocType) ([]string, error) {
	if !ValidDocType(string(docType)) {
		return nil, fmt.Errorf("invalid document type: %s", docType)
	}

	typeDir := filepath.Join(s.root, string(docType))
	entries, err := filepath.Glob(filepath.Join(typeDir, "*.md"))
	if err != nil {
		return nil, fmt.Errorf("list documents: %w", err)
	}

	return entries, nil
}

// ListAll returns paths of all documents across all types.
func (s *DocStore) ListAll() ([]string, error) {
	var all []string
	for _, dt := range AllDocTypes() {
		paths, err := s.List(dt)
		if err != nil {
			return nil, err
		}
		all = append(all, paths...)
	}
	return all, nil
}

// docFileName returns the file name for a document.
func docFileName(meta DocMeta) string {
	slug := slugify(meta.Title)
	return fmt.Sprintf("%s-%s.md", meta.ID, slug)
}

// slugify converts a title to a file-name-safe slug.
func slugify(title string) string {
	s := strings.ToLower(strings.TrimSpace(title))
	s = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			return r
		}
		return '-'
	}, s)
	// collapse multiple dashes
	for strings.Contains(s, "--") {
		s = strings.ReplaceAll(s, "--", "-")
	}
	s = strings.Trim(s, "-")
	return s
}

// marshalDocument serialises a document to YAML frontmatter + markdown body.
func marshalDocument(doc Document) string {
	var b strings.Builder
	b.WriteString("---\n")

	b.WriteString("id: ")
	b.WriteString(doc.Meta.ID)
	b.WriteString("\n")

	b.WriteString("type: ")
	b.WriteString(string(doc.Meta.Type))
	b.WriteString("\n")

	b.WriteString("title: ")
	b.WriteString(yamlQuoteIfNeeded(doc.Meta.Title))
	b.WriteString("\n")

	b.WriteString("status: ")
	b.WriteString(string(doc.Meta.Status))
	b.WriteString("\n")

	if doc.Meta.Feature != "" {
		b.WriteString("feature: ")
		b.WriteString(doc.Meta.Feature)
		b.WriteString("\n")
	}

	b.WriteString("created_by: ")
	b.WriteString(doc.Meta.CreatedBy)
	b.WriteString("\n")

	b.WriteString("created: ")
	b.WriteString(doc.Meta.Created.Format(time.RFC3339))
	b.WriteString("\n")

	b.WriteString("updated: ")
	b.WriteString(doc.Meta.Updated.Format(time.RFC3339))
	b.WriteString("\n")

	if doc.Meta.ApprovedBy != "" {
		b.WriteString("approved_by: ")
		b.WriteString(doc.Meta.ApprovedBy)
		b.WriteString("\n")
	}

	if doc.Meta.ApprovedAt != nil {
		b.WriteString("approved_at: ")
		b.WriteString(doc.Meta.ApprovedAt.Format(time.RFC3339))
		b.WriteString("\n")
	}

	b.WriteString("---\n")
	b.WriteString(doc.Body)

	return b.String()
}

// unmarshalDocument parses a document from YAML frontmatter + markdown body.
func unmarshalDocument(content string) (Document, error) {
	if !strings.HasPrefix(content, "---\n") {
		return Document{}, errors.New("document missing frontmatter delimiter")
	}

	rest := content[4:] // skip opening "---\n"
	endIdx := strings.Index(rest, "\n---\n")
	if endIdx < 0 {
		return Document{}, errors.New("document missing closing frontmatter delimiter")
	}

	frontmatter := rest[:endIdx]
	body := rest[endIdx+5:] // skip "\n---\n"

	meta, err := parseFrontmatter(frontmatter)
	if err != nil {
		return Document{}, fmt.Errorf("parse frontmatter: %w", err)
	}

	return Document{
		Meta: meta,
		Body: body,
	}, nil
}

// parseFrontmatter parses the YAML frontmatter block into DocMeta.
// This is a simple key-value parser, not a full YAML parser.
func parseFrontmatter(fm string) (DocMeta, error) {
	var meta DocMeta
	lines := strings.Split(fm, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		// strip quotes if present
		if len(value) >= 2 && value[0] == '"' && value[len(value)-1] == '"' {
			value = value[1 : len(value)-1]
		}

		switch key {
		case "id":
			meta.ID = value
		case "type":
			meta.Type = DocType(value)
		case "title":
			meta.Title = value
		case "status":
			meta.Status = DocStatus(value)
		case "feature":
			meta.Feature = value
		case "created_by":
			meta.CreatedBy = value
		case "created":
			t, err := time.Parse(time.RFC3339, value)
			if err != nil {
				return meta, fmt.Errorf("parse created time: %w", err)
			}
			meta.Created = t
		case "updated":
			t, err := time.Parse(time.RFC3339, value)
			if err != nil {
				return meta, fmt.Errorf("parse updated time: %w", err)
			}
			meta.Updated = t
		case "approved_by":
			meta.ApprovedBy = value
		case "approved_at":
			t, err := time.Parse(time.RFC3339, value)
			if err != nil {
				return meta, fmt.Errorf("parse approved_at time: %w", err)
			}
			meta.ApprovedAt = &t
		}
	}

	return meta, nil
}

// yamlQuoteIfNeeded wraps the value in double quotes if it contains
// characters that could cause YAML ambiguity.
func yamlQuoteIfNeeded(value string) string {
	if value == "" {
		return `""`
	}
	for _, r := range value {
		switch r {
		case ':', '#', '{', '}', '[', ']', ',', '&', '*', '!', '|', '>', '@', '`', '"', '\'':
			return `"` + strings.ReplaceAll(value, `"`, `\"`) + `"`
		}
	}
	if strings.HasPrefix(value, "-") || strings.HasPrefix(value, "?") {
		return `"` + strings.ReplaceAll(value, `"`, `\"`) + `"`
	}
	return value
}
