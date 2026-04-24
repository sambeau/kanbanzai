package docint

import (
	"fmt"
	"time"

	"gopkg.in/yaml.v3"
)

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

// ConceptIntroEntry represents a concept introduced by a section.
// It accepts both plain string and object form in YAML:
//
//	concepts_intro:
//	  - plain-string-concept
//	  - name: concept-name
//	    aliases: [alt-form, another-form]
type ConceptIntroEntry struct {
	Name    string   `json:"name"`              // canonical concept name
	Aliases []string `json:"aliases,omitempty"` // alternative forms (optional)
}

// UnmarshalYAML implements yaml.Unmarshaler.
// A scalar node is treated as a plain name with no aliases.
// A mapping node must have a "name" key and an optional "aliases" key.
func (e *ConceptIntroEntry) UnmarshalYAML(value *yaml.Node) error {
	switch value.Kind {
	case yaml.ScalarNode:
		e.Name = value.Value
		return nil
	case yaml.MappingNode:
		var obj struct {
			Name    string   `yaml:"name"`
			Aliases []string `yaml:"aliases"`
		}
		if err := value.Decode(&obj); err != nil {
			return err
		}
		e.Name = obj.Name
		e.Aliases = obj.Aliases
		return nil
	default:
		return fmt.Errorf("concepts_intro entry must be a string or map, got kind %v", value.Kind)
	}
}

// MarshalYAML implements yaml.Marshaler.
// Entries without aliases are serialised as plain strings for backward compatibility.
func (e ConceptIntroEntry) MarshalYAML() (interface{}, error) {
	if len(e.Aliases) == 0 {
		return e.Name, nil
	}
	return struct {
		Name    string   `yaml:"name"`
		Aliases []string `yaml:"aliases"`
	}{Name: e.Name, Aliases: e.Aliases}, nil
}

// Classification represents an agent-provided fragment classification (Layer 3).
type Classification struct {
	SectionPath   string              `yaml:"section_path"             json:"section_path"`
	Role          string              `yaml:"role"                     json:"role"`       // From FragmentRole taxonomy
	Confidence    string              `yaml:"confidence"               json:"confidence"` // "high", "medium", "low"
	Summary       string              `yaml:"summary,omitempty"        json:"summary,omitempty"`
	ConceptsIntro []ConceptIntroEntry `yaml:"concepts_intro,omitempty" json:"concepts_intro,omitempty"` // Concepts this section introduces
	ConceptsUsed  []string            `yaml:"concepts_used,omitempty"  json:"concepts_used,omitempty"`  // Concepts this section uses
}

// ClassificationEntry pairs a document's content hash with one of its
// classification records. It is the element type returned by
// IntelligenceService.GetClassifications and used by the concept-tagging
// approval gate to evaluate REQ-002, REQ-003, and REQ-005 (content_hash).
type ClassificationEntry struct {
	ContentHash    string         `json:"content_hash"`
	Classification Classification `json:"classification"`
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

// SectionAccessInfo holds access counters for a specific document section.
type SectionAccessInfo struct {
	AccessCount    int        `yaml:"access_count,omitempty"`
	LastAccessedAt *time.Time `yaml:"last_accessed_at,omitempty"`
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

	// Access tracking (Layer 5 — instrumentation)
	AccessCount    int                          `yaml:"access_count,omitempty"`
	LastAccessedAt *time.Time                   `yaml:"last_accessed_at,omitempty"`
	SectionAccess  map[string]SectionAccessInfo `yaml:"section_access,omitempty"`

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
	Aliases      []string `yaml:"aliases,omitempty"`       // Alternative forms (normalised)
	IntroducedIn []string `yaml:"introduced_in,omitempty"` // Section IDs that INTRODUCE this concept
	UsedIn       []string `yaml:"used_in,omitempty"`       // Section IDs that USE this concept
}

// ConceptRegistry is the persistent concept registry file.
type ConceptRegistry struct {
	UpdatedAt time.Time `yaml:"updated_at"`
	Concepts  []Concept `yaml:"concepts"`
}
