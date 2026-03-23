package docint

import "time"

// Section represents a headed section in a Markdown document (Layer 1).
type Section struct {
	Path        string    `yaml:"path"`         // Hierarchical path e.g. "1.2.3"
	Level       int       `yaml:"level"`        // Heading level (1-6)
	Title       string    `yaml:"title"`        // Heading text
	ByteOffset  int       `yaml:"byte_offset"`  // Byte offset in source file
	ByteCount   int       `yaml:"byte_count"`   // Byte length of section content
	WordCount   int       `yaml:"word_count"`   // Word count of section content
	ContentHash string    `yaml:"content_hash"` // SHA-256 of section content
	Children    []Section `yaml:"children,omitempty"`
}

// EntityRef represents a workflow entity reference found in text (Layer 2).
type EntityRef struct {
	EntityID    string `yaml:"entity_id"`    // e.g. "FEAT-xxxx", "TASK-xxxx", "P1-basic-ui"
	EntityType  string `yaml:"entity_type"`  // "feature", "task", "bug", "decision", "plan"
	SectionPath string `yaml:"section_path"` // Which section contains this reference
	ByteOffset  int    `yaml:"byte_offset"`  // Location in document
}

// CrossDocLink represents a link to another document (Layer 2).
type CrossDocLink struct {
	TargetPath  string `yaml:"target_path"`  // Path to linked document
	LinkText    string `yaml:"link_text"`    // Markdown link text
	SectionPath string `yaml:"section_path"` // Which section contains this link
}

// ConventionalRole represents a section classified by its heading keywords (Layer 2).
type ConventionalRole struct {
	SectionPath string `yaml:"section_path"`
	Role        string `yaml:"role"`       // From FragmentRole taxonomy
	Confidence  string `yaml:"confidence"` // "high" for exact keyword match
}

// FrontMatter holds extracted front matter fields (Layer 2).
type FrontMatter struct {
	Type    string            `yaml:"type,omitempty"`
	Status  string            `yaml:"status,omitempty"`
	Date    string            `yaml:"date,omitempty"`
	Related []string          `yaml:"related,omitempty"`
	Extra   map[string]string `yaml:"extra,omitempty"`
}

// Classification represents an agent-provided fragment classification (Layer 3).
type Classification struct {
	SectionPath   string   `yaml:"section_path"`
	Role          string   `yaml:"role"`                     // From FragmentRole taxonomy
	Confidence    string   `yaml:"confidence"`               // "high", "medium", "low"
	Summary       string   `yaml:"summary,omitempty"`        // One-line characterisation
	ConceptsIntro []string `yaml:"concepts_intro,omitempty"` // Concepts this section introduces
	ConceptsUsed  []string `yaml:"concepts_used,omitempty"`  // Concepts this section uses
}

// ClassificationSubmission is the input for Layer 3 classification.
type ClassificationSubmission struct {
	DocumentID      string           `yaml:"document_id"`
	ContentHash     string           `yaml:"content_hash"` // Must match current doc hash
	ModelName       string           `yaml:"model_name"`   // Which LLM classified
	ModelVersion    string           `yaml:"model_version"`
	ClassifiedAt    time.Time        `yaml:"classified_at"`
	Classifications []Classification `yaml:"classifications"`
}

// DocumentIndex is the persistent per-document index file (Layers 1-3).
type DocumentIndex struct {
	DocumentID   string    `yaml:"document_id"`
	DocumentPath string    `yaml:"document_path"`
	ContentHash  string    `yaml:"content_hash"`
	IndexedAt    time.Time `yaml:"indexed_at"`

	// Layer 1
	Sections []Section `yaml:"sections"`

	// Layer 2
	FrontMatter       *FrontMatter       `yaml:"front_matter,omitempty"`
	EntityRefs        []EntityRef        `yaml:"entity_refs,omitempty"`
	CrossDocLinks     []CrossDocLink     `yaml:"cross_doc_links,omitempty"`
	ConventionalRoles []ConventionalRole `yaml:"conventional_roles,omitempty"`

	// Layer 3 (populated by agent classification)
	Classified        bool             `yaml:"classified"`
	ClassifiedAt      *time.Time       `yaml:"classified_at,omitempty"`
	ClassifiedBy      string           `yaml:"classified_by,omitempty"` // model name
	ClassifierVersion string           `yaml:"classifier_version,omitempty"`
	Classifications   []Classification `yaml:"classifications,omitempty"`
}

// GraphEdge represents a single edge in the document graph (Layer 4).
type GraphEdge struct {
	From     string `yaml:"from"`      // Node ID
	FromType string `yaml:"from_type"` // "document", "section", "fragment", "entity_ref", "concept"
	To       string `yaml:"to"`        // Node ID
	ToType   string `yaml:"to_type"`
	EdgeType string `yaml:"edge_type"` // CONTAINS, REFERENCES, LINKS_TO, etc.
}

// DocumentGraph is the persistent graph file.
type DocumentGraph struct {
	UpdatedAt time.Time   `yaml:"updated_at"`
	Edges     []GraphEdge `yaml:"edges"`
}

// Concept represents an entry in the concept registry.
type Concept struct {
	Name         string   `yaml:"name"`                    // Canonical name (lowercase, hyphenated)
	Aliases      []string `yaml:"aliases,omitempty"`       // Alternative forms
	IntroducedIn []string `yaml:"introduced_in,omitempty"` // Section IDs that INTRODUCE this concept
	UsedIn       []string `yaml:"used_in,omitempty"`       // Section IDs that USE this concept
}

// ConceptRegistry is the persistent concept registry file.
type ConceptRegistry struct {
	UpdatedAt time.Time `yaml:"updated_at"`
	Concepts  []Concept `yaml:"concepts"`
}
