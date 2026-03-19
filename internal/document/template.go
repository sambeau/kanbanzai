package document

import (
	"fmt"
	"strings"
)

// Template defines the structural requirements for a document type.
type Template struct {
	Type             DocType
	RequiredSections []string
	Description      string
}

// Templates returns the template definition for each recognised document type.
var templates = map[DocType]Template{
	DocTypeProposal: {
		Type:        DocTypeProposal,
		Description: "A small, informal document putting forward a broad idea.",
		RequiredSections: []string{
			"Summary",
			"Problem",
			"Proposal",
		},
	},
	DocTypeResearchReport: {
		Type:        DocTypeResearchReport,
		Description: "A substantive piece of research with findings for use in design work.",
		RequiredSections: []string{
			"Summary",
			"Scope",
			"Findings",
			"Conclusions",
		},
	},
	DocTypeDraftDesign: {
		Type:        DocTypeDraftDesign,
		Description: "A semi-formal design document that shows its working.",
		RequiredSections: []string{
			"Summary",
			"Context",
			"Design",
			"Open Questions",
		},
	},
	DocTypeDesign: {
		Type:        DocTypeDesign,
		Description: "The final design document for a feature.",
		RequiredSections: []string{
			"Purpose",
			"Design",
			"Decisions",
			"Acceptance Criteria",
		},
	},
	DocTypeSpecification: {
		Type:        DocTypeSpecification,
		Description: "The formal specification for a feature.",
		RequiredSections: []string{
			"Purpose",
			"Scope",
			"Requirements",
			"Acceptance Criteria",
		},
	},
	DocTypeImplementationPlan: {
		Type:        DocTypeImplementationPlan,
		Description: "A formal plan for implementing the specification.",
		RequiredSections: []string{
			"Purpose",
			"Scope",
			"Tasks",
			"Verification",
		},
	},
	DocTypeUserDocumentation: {
		Type:        DocTypeUserDocumentation,
		Description: "Documentation delivered to end users of the product.",
		RequiredSections: []string{
			"Overview",
		},
	},
}

// GetTemplate returns the template for a document type.
// Returns an error if the document type is not recognised.
func GetTemplate(docType DocType) (Template, error) {
	t, ok := templates[docType]
	if !ok {
		return Template{}, fmt.Errorf("unknown document type: %s", string(docType))
	}
	return t, nil
}

// Scaffold generates a starter markdown document from a template.
// The title is used as the document heading.
func Scaffold(docType DocType, title string) (string, error) {
	t, err := GetTemplate(docType)
	if err != nil {
		return "", err
	}

	if strings.TrimSpace(title) == "" {
		return "", fmt.Errorf("document title is required")
	}

	var b strings.Builder
	b.WriteString("# ")
	b.WriteString(title)
	b.WriteString("\n\n")

	for _, section := range t.RequiredSections {
		b.WriteString("## ")
		b.WriteString(section)
		b.WriteString("\n\n")
	}

	return b.String(), nil
}
