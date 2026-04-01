package structural

import (
	"fmt"
	"strings"

	"github.com/sambeau/kanbanzai/internal/docint"
)

// CheckResult holds the outcome of a single structural check.
type CheckResult struct {
	CheckType    string   `json:"check_type"`
	Gate         string   `json:"gate"`
	DocumentID   string   `json:"document_id"`
	DocumentType string   `json:"document_type"`
	Passed       bool     `json:"passed"`
	Mode         string   `json:"mode"`
	Details      []string `json:"details"`
}

// CheckRequiredSections verifies that the document sections include all required sections
// for the given document type. Only headings at levels 2-3 are checked (the H1 title is
// excluded). Matching is case-insensitive substring match.
func CheckRequiredSections(sections []docint.Section, docType, docID, gate string) CheckResult {
	result := CheckResult{
		CheckType:    "required_sections",
		Gate:         gate,
		DocumentID:   docID,
		DocumentType: docType,
		Passed:       true,
	}

	required := RequiredSections(docType)
	if required == nil {
		return result
	}

	titles := collectTitles(sections)

	for _, req := range required {
		if !matchesAnyKeyword(titles, req.Keywords) {
			result.Passed = false
			result.Details = append(result.Details, fmt.Sprintf("missing required section: %s", req.Label))
		}
	}

	return result
}

// CheckCrossReference verifies that the document references at least one approved design
// document, either via a cross-doc link (matching approvedDesignPaths) or via an entity
// reference (matching approvedDesignIDs).
func CheckCrossReference(extractResult docint.ExtractResult, approvedDesignPaths, approvedDesignIDs []string, docID, gate string) CheckResult {
	result := CheckResult{
		CheckType:  "cross_reference",
		Gate:       gate,
		DocumentID: docID,
		Passed:     false,
	}

	pathSet := make(map[string]bool, len(approvedDesignPaths))
	for _, p := range approvedDesignPaths {
		pathSet[p] = true
	}

	idSet := make(map[string]bool, len(approvedDesignIDs))
	for _, id := range approvedDesignIDs {
		idSet[id] = true
	}

	for _, link := range extractResult.CrossDocLinks {
		if pathSet[link.TargetPath] {
			result.Passed = true
			return result
		}
	}

	for _, ref := range extractResult.EntityRefs {
		if idSet[ref.EntityID] {
			result.Passed = true
			return result
		}
	}

	result.Details = []string{"no reference to an approved design document found"}
	return result
}

// CheckAcceptanceCriteria verifies that the document contains at least one acceptance
// criteria section. It first checks conventionalRoles for sections whose heading matched
// the acceptance criteria keyword, then falls back to scanning all headings directly.
func CheckAcceptanceCriteria(sections []docint.Section, conventionalRoles []docint.ConventionalRole, docID, gate string) CheckResult {
	result := CheckResult{
		CheckType:  "acceptance_criteria",
		Gate:       gate,
		DocumentID: docID,
		Passed:     false,
	}

	titleByPath := make(map[string]string)
	buildTitleByPath(sections, titleByPath)

	for _, role := range conventionalRoles {
		if role.Role != string(docint.RoleRequirement) {
			continue
		}
		title, ok := titleByPath[role.SectionPath]
		if !ok {
			continue
		}
		if strings.Contains(strings.ToLower(title), "acceptance criteria") {
			result.Passed = true
			return result
		}
	}

	all := collectAllTitles(sections)
	if matchesAnyKeyword(all, []string{"acceptance criteria"}) {
		result.Passed = true
		return result
	}

	result.Details = []string{"no acceptance criteria section found"}
	return result
}

func collectTitles(sections []docint.Section) []string {
	var out []string
	var walk func([]docint.Section)
	walk = func(ss []docint.Section) {
		for i := range ss {
			s := &ss[i]
			if s.Level >= 2 && s.Level <= 3 {
				out = append(out, s.Title)
			}
			walk(s.Children)
		}
	}
	walk(sections)
	return out
}

func collectAllTitles(sections []docint.Section) []string {
	var out []string
	var walk func([]docint.Section)
	walk = func(ss []docint.Section) {
		for i := range ss {
			s := &ss[i]
			out = append(out, s.Title)
			walk(s.Children)
		}
	}
	walk(sections)
	return out
}

func buildTitleByPath(sections []docint.Section, m map[string]string) {
	for i := range sections {
		s := &sections[i]
		m[s.Path] = s.Title
		buildTitleByPath(s.Children, m)
	}
}

func matchesAnyKeyword(titles, keywords []string) bool {
	for _, title := range titles {
		lower := strings.ToLower(title)
		for _, kw := range keywords {
			if strings.Contains(lower, strings.ToLower(kw)) {
				return true
			}
		}
	}
	return false
}
