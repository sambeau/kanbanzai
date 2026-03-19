package document

import "time"

// DocType identifies a recognised document type.
type DocType string

const (
	DocTypeProposal           DocType = "proposal"
	DocTypeResearchReport     DocType = "research-report"
	DocTypeDraftDesign        DocType = "draft-design"
	DocTypeDesign             DocType = "design"
	DocTypeSpecification      DocType = "specification"
	DocTypeImplementationPlan DocType = "implementation-plan"
	DocTypeUserDocumentation  DocType = "user-documentation"
)

// AllDocTypes returns the ordered list of recognised document types.
func AllDocTypes() []DocType {
	return []DocType{
		DocTypeProposal,
		DocTypeResearchReport,
		DocTypeDraftDesign,
		DocTypeDesign,
		DocTypeSpecification,
		DocTypeImplementationPlan,
		DocTypeUserDocumentation,
	}
}

// ValidDocType returns true if the given string is a recognised document type.
func ValidDocType(s string) bool {
	for _, dt := range AllDocTypes() {
		if string(dt) == s {
			return true
		}
	}
	return false
}

// DocStatus is the lifecycle state of a document.
type DocStatus string

const (
	DocStatusDraft      DocStatus = "draft"
	DocStatusSubmitted  DocStatus = "submitted"
	DocStatusNormalised DocStatus = "normalised"
	DocStatusApproved   DocStatus = "approved"
)

// DocMeta is the structured metadata for a stored document.
type DocMeta struct {
	ID         string     `yaml:"id"`
	Type       DocType    `yaml:"type"`
	Title      string     `yaml:"title"`
	Status     DocStatus  `yaml:"status"`
	Feature    string     `yaml:"feature,omitempty"`
	CreatedBy  string     `yaml:"created_by"`
	Created    time.Time  `yaml:"created"`
	Updated    time.Time  `yaml:"updated"`
	ApprovedBy string     `yaml:"approved_by,omitempty"`
	ApprovedAt *time.Time `yaml:"approved_at,omitempty"`
}

// Document is a complete document record: metadata plus body content.
type Document struct {
	Meta DocMeta
	Body string
}

// ExtractedDocument is the structured extraction payload returned for approved documents.
type ExtractedDocument struct {
	Meta ExtractedDocumentMeta `json:"meta"`
	Body string                `json:"body"`
}

// ExtractedDocumentMeta is the metadata returned alongside extracted document body content.
type ExtractedDocumentMeta struct {
	ID         string     `json:"id"`
	Type       DocType    `json:"type"`
	Title      string     `json:"title"`
	Status     DocStatus  `json:"status"`
	Feature    string     `json:"feature,omitempty"`
	CreatedBy  string     `json:"created_by"`
	Created    time.Time  `json:"created"`
	Updated    time.Time  `json:"updated"`
	ApprovedBy string     `json:"approved_by,omitempty"`
	ApprovedAt *time.Time `json:"approved_at,omitempty"`
}
