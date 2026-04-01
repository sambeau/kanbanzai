package structural

// SectionRequirement describes a required section in a document, identified by
// a human-readable label and a set of keywords to match against section headings.
type SectionRequirement struct {
	Label    string
	Keywords []string
}

// RequiredSections returns the ordered list of required sections for a given
// document type. Returns nil for document types with no structural requirements.
func RequiredSections(docType string) []SectionRequirement {
	switch docType {
	case "design":
		return []SectionRequirement{
			{Label: "overview/purpose/summary", Keywords: []string{"overview", "purpose", "summary"}},
			{Label: "design", Keywords: []string{"design"}},
		}
	case "specification":
		return []SectionRequirement{
			{Label: "overview", Keywords: []string{"overview"}},
			{Label: "scope", Keywords: []string{"scope"}},
			{Label: "functional requirements", Keywords: []string{"functional requirements"}},
			{Label: "acceptance criteria", Keywords: []string{"acceptance criteria"}},
		}
	case "dev-plan":
		return []SectionRequirement{
			{Label: "overview", Keywords: []string{"overview"}},
			{Label: "task", Keywords: []string{"task"}},
		}
	default:
		return nil
	}
}
