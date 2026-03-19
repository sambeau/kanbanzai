package document

import (
	"strings"
	"testing"
	"time"
)

// validDoc returns a fully valid Document for use as a baseline in tests.
// Callers can mutate the returned value to create invalid variants.
func validDoc() Document {
	t := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)
	return Document{
		Meta: DocMeta{
			ID:        "DOC-001",
			Type:      DocTypeProposal,
			Title:     "Test Proposal",
			Status:    DocStatusDraft,
			CreatedBy: "alice",
			Created:   t,
			Updated:   t,
		},
		Body: "# Test Proposal\n\n## Summary\n\nA summary.\n\n## Problem\n\nA problem.\n\n## Proposal\n\nA proposal.\n",
	}
}

// validDesignDoc returns a fully valid design Document that requires a feature ref.
func validDesignDoc() Document {
	t := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)
	return Document{
		Meta: DocMeta{
			ID:        "DOC-002",
			Type:      DocTypeDesign,
			Title:     "Test Design",
			Status:    DocStatusDraft,
			Feature:   "FEAT-001",
			CreatedBy: "bob",
			Created:   t,
			Updated:   t,
		},
		Body: "# Test Design\n\n## Purpose\n\nPurpose.\n\n## Design\n\nDesign.\n\n## Decisions\n\nDecisions.\n\n## Acceptance Criteria\n\nCriteria.\n",
	}
}

// hasFieldError returns true if errs contains an error for the given field.
func hasFieldError(t *testing.T, errs []ValidationError, field string) bool {
	t.Helper()
	for _, e := range errs {
		if e.Field == field {
			return true
		}
	}
	return false
}

// requireFieldError fails the test if errs does not contain an error for the given field.
func requireFieldError(t *testing.T, errs []ValidationError, field string) {
	t.Helper()
	if !hasFieldError(t, errs, field) {
		t.Errorf("expected validation error for field %q, got errors: %v", field, errs)
	}
}

// requireNoErrors fails the test if errs is non-empty.
func requireNoErrors(t *testing.T, errs []ValidationError) {
	t.Helper()
	if len(errs) != 0 {
		t.Errorf("expected no validation errors, got: %v", errs)
	}
}

// --- ValidateDocument tests ---

func TestValidateDocument_FullyValid(t *testing.T) {
	doc := validDoc()
	errs := ValidateDocument(doc)
	requireNoErrors(t, errs)
}

func TestValidateDocument_FullyValidDesign(t *testing.T) {
	doc := validDesignDoc()
	errs := ValidateDocument(doc)
	requireNoErrors(t, errs)
}

func TestValidateDocument_InvalidType(t *testing.T) {
	doc := validDoc()
	doc.Meta.Type = "nonsense"
	errs := ValidateDocument(doc)
	if len(errs) != 1 {
		t.Fatalf("expected exactly 1 error for invalid type, got %d: %v", len(errs), errs)
	}
	requireFieldError(t, errs, "type")
}

func TestValidateDocument_MissingRequiredFields(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*Document)
		field  string
	}{
		{"missing id", func(d *Document) { d.Meta.ID = "" }, "id"},
		{"whitespace id", func(d *Document) { d.Meta.ID = "   " }, "id"},
		{"missing title", func(d *Document) { d.Meta.Title = "" }, "title"},
		{"missing status", func(d *Document) { d.Meta.Status = "" }, "status"},
		{"missing created_by", func(d *Document) { d.Meta.CreatedBy = "" }, "created_by"},
		{"zero created", func(d *Document) { d.Meta.Created = time.Time{} }, "created"},
		{"zero updated", func(d *Document) { d.Meta.Updated = time.Time{} }, "updated"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			doc := validDoc()
			tc.mutate(&doc)
			errs := ValidateDocument(doc)
			requireFieldError(t, errs, tc.field)
		})
	}
}

func TestValidateDocument_InvalidStatus(t *testing.T) {
	doc := validDoc()
	doc.Meta.Status = "archived"
	errs := ValidateDocument(doc)
	requireFieldError(t, errs, "status")
	// Check the message mentions "invalid status"
	for _, e := range errs {
		if e.Field == "status" {
			if !strings.Contains(e.Message, "invalid status") {
				t.Errorf("expected message to contain 'invalid status', got: %s", e.Message)
			}
		}
	}
}

func TestValidateDocument_EmptyBody(t *testing.T) {
	doc := validDoc()
	doc.Body = ""
	errs := ValidateDocument(doc)
	requireFieldError(t, errs, "body")
}

func TestValidateDocument_WhitespaceOnlyBody(t *testing.T) {
	doc := validDoc()
	doc.Body = "   \n\n  \t  "
	errs := ValidateDocument(doc)
	requireFieldError(t, errs, "body")
}

func TestValidateDocument_DesignMissingFeatureRef(t *testing.T) {
	doc := validDesignDoc()
	doc.Meta.Feature = ""
	errs := ValidateDocument(doc)
	requireFieldError(t, errs, "feature")
}

func TestValidateDocument_SpecificationMissingFeatureRef(t *testing.T) {
	now := time.Now()
	doc := Document{
		Meta: DocMeta{
			ID:        "DOC-003",
			Type:      DocTypeSpecification,
			Title:     "Test Spec",
			Status:    DocStatusDraft,
			Feature:   "",
			CreatedBy: "carol",
			Created:   now,
			Updated:   now,
		},
		Body: "# Test Spec\n\n## Purpose\n\nP.\n\n## Scope\n\nS.\n\n## Requirements\n\nR.\n\n## Acceptance Criteria\n\nAC.\n",
	}
	errs := ValidateDocument(doc)
	requireFieldError(t, errs, "feature")
}

func TestValidateDocument_ImplementationPlanMissingFeatureRef(t *testing.T) {
	now := time.Now()
	doc := Document{
		Meta: DocMeta{
			ID:        "DOC-004",
			Type:      DocTypeImplementationPlan,
			Title:     "Test Plan",
			Status:    DocStatusDraft,
			Feature:   "",
			CreatedBy: "dave",
			Created:   now,
			Updated:   now,
		},
		Body: "# Test Plan\n\n## Purpose\n\nP.\n\n## Scope\n\nS.\n\n## Tasks\n\nT.\n\n## Verification\n\nV.\n",
	}
	errs := ValidateDocument(doc)
	requireFieldError(t, errs, "feature")
}

func TestValidateDocument_ProposalWithoutFeatureRef(t *testing.T) {
	doc := validDoc()
	doc.Meta.Feature = "" // proposals don't require feature ref
	errs := ValidateDocument(doc)
	if hasFieldError(t, errs, "feature") {
		t.Error("proposal should not require a feature reference")
	}
}

func TestValidateDocument_ApprovedMissingApprovalMetadata(t *testing.T) {
	doc := validDoc()
	doc.Meta.Status = DocStatusApproved
	// No ApprovedBy or ApprovedAt set
	errs := ValidateDocument(doc)
	requireFieldError(t, errs, "approved_by")
	requireFieldError(t, errs, "approved_at")
}

func TestValidateDocument_ApprovedWithApprovalMetadata(t *testing.T) {
	doc := validDoc()
	doc.Meta.Status = DocStatusApproved
	doc.Meta.ApprovedBy = "manager"
	approvedAt := time.Date(2025, 2, 1, 12, 0, 0, 0, time.UTC)
	doc.Meta.ApprovedAt = &approvedAt
	errs := ValidateDocument(doc)
	if hasFieldError(t, errs, "approved_by") {
		t.Error("should not have approved_by error when field is set")
	}
	if hasFieldError(t, errs, "approved_at") {
		t.Error("should not have approved_at error when field is set")
	}
}

func TestValidateDocument_MissingRequiredSections(t *testing.T) {
	doc := validDoc()
	// Body with only the title heading, missing Summary, Problem, Proposal
	doc.Body = "# Test Proposal\n\nSome text without headings.\n"
	errs := ValidateDocument(doc)
	// Should have errors for missing sections
	bodyErrors := 0
	for _, e := range errs {
		if e.Field == "body" && strings.Contains(e.Message, "missing required section") {
			bodyErrors++
		}
	}
	if bodyErrors != 3 {
		t.Errorf("expected 3 missing section errors for proposal, got %d (errors: %v)", bodyErrors, errs)
	}
}

func TestValidateDocument_SectionMatchingCaseInsensitive(t *testing.T) {
	doc := validDoc()
	// Use different casing for headings
	doc.Body = "# Test Proposal\n\n## SUMMARY\n\nA summary.\n\n## problem\n\nA problem.\n\n## Proposal\n\nA proposal.\n"
	errs := ValidateDocument(doc)
	requireNoErrors(t, errs)
}

func TestValidateDocument_MultipleErrors(t *testing.T) {
	doc := Document{
		Meta: DocMeta{
			Type: DocTypeProposal,
			// Everything else empty/zero
		},
		Body: "",
	}
	errs := ValidateDocument(doc)
	// Should have errors for id, title, status, created_by, created, updated, body
	if len(errs) < 5 {
		t.Errorf("expected multiple validation errors, got %d: %v", len(errs), errs)
	}
}

// --- ValidDocTransition tests ---

func TestValidDocTransition_ValidTransitions(t *testing.T) {
	tests := []struct {
		name string
		from DocStatus
		to   DocStatus
	}{
		{"draft to submitted", DocStatusDraft, DocStatusSubmitted},
		{"submitted to normalised", DocStatusSubmitted, DocStatusNormalised},
		{"normalised to approved", DocStatusNormalised, DocStatusApproved},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if !ValidDocTransition(tc.from, tc.to) {
				t.Errorf("expected %s → %s to be valid", tc.from, tc.to)
			}
		})
	}
}

func TestValidDocTransition_InvalidTransitions(t *testing.T) {
	tests := []struct {
		name string
		from DocStatus
		to   DocStatus
	}{
		{"approved to draft", DocStatusApproved, DocStatusDraft},
		{"approved to submitted", DocStatusApproved, DocStatusSubmitted},
		{"approved to normalised", DocStatusApproved, DocStatusNormalised},
		{"approved to approved", DocStatusApproved, DocStatusApproved},
		{"draft to normalised (skip)", DocStatusDraft, DocStatusNormalised},
		{"draft to approved (skip)", DocStatusDraft, DocStatusApproved},
		{"submitted to approved (skip)", DocStatusSubmitted, DocStatusApproved},
		{"submitted to draft (backwards)", DocStatusSubmitted, DocStatusDraft},
		{"normalised to draft (backwards)", DocStatusNormalised, DocStatusDraft},
		{"normalised to submitted (backwards)", DocStatusNormalised, DocStatusSubmitted},
		{"draft to draft (same)", DocStatusDraft, DocStatusDraft},
		{"unknown from", DocStatus("unknown"), DocStatusDraft},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if ValidDocTransition(tc.from, tc.to) {
				t.Errorf("expected %s → %s to be invalid", tc.from, tc.to)
			}
		})
	}
}

// --- requiresFeatureRef tests ---

func TestRequiresFeatureRef(t *testing.T) {
	tests := []struct {
		docType  DocType
		expected bool
	}{
		{DocTypeProposal, false},
		{DocTypeResearchReport, false},
		{DocTypeDraftDesign, false},
		{DocTypeDesign, true},
		{DocTypeSpecification, true},
		{DocTypeImplementationPlan, true},
		{DocTypeUserDocumentation, false},
	}

	for _, tc := range tests {
		t.Run(string(tc.docType), func(t *testing.T) {
			got := requiresFeatureRef(tc.docType)
			if got != tc.expected {
				t.Errorf("requiresFeatureRef(%s) = %v, want %v", tc.docType, got, tc.expected)
			}
		})
	}
}

// --- extractHeadings tests ---

func TestExtractHeadings_VariousLevels(t *testing.T) {
	body := "# Heading One\n\nSome text.\n\n## Heading Two\n\nMore text.\n\n### Heading Three\n\nEven more.\n"
	headings := extractHeadings(body)

	expected := []string{"Heading One", "Heading Two", "Heading Three"}
	if len(headings) != len(expected) {
		t.Fatalf("expected %d headings, got %d: %v", len(expected), len(headings), headings)
	}
	for i, h := range headings {
		if h != expected[i] {
			t.Errorf("heading[%d] = %q, want %q", i, h, expected[i])
		}
	}
}

func TestExtractHeadings_EmptyBody(t *testing.T) {
	headings := extractHeadings("")
	if len(headings) != 0 {
		t.Errorf("expected no headings from empty body, got: %v", headings)
	}
}

func TestExtractHeadings_NoHeadings(t *testing.T) {
	body := "Just some text.\nAnother line.\n"
	headings := extractHeadings(body)
	if len(headings) != 0 {
		t.Errorf("expected no headings, got: %v", headings)
	}
}

func TestExtractHeadings_HeadingWithExtraSpaces(t *testing.T) {
	body := "##   Spaced Heading   \n"
	headings := extractHeadings(body)
	if len(headings) != 1 {
		t.Fatalf("expected 1 heading, got %d: %v", len(headings), headings)
	}
	if headings[0] != "Spaced Heading" {
		t.Errorf("heading = %q, want %q", headings[0], "Spaced Heading")
	}
}

func TestExtractHeadings_HashOnlyLine(t *testing.T) {
	body := "###\n\n## Real Heading\n"
	headings := extractHeadings(body)
	// "###" alone has no text after stripping, so it should be skipped
	if len(headings) != 1 {
		t.Fatalf("expected 1 heading, got %d: %v", len(headings), headings)
	}
	if headings[0] != "Real Heading" {
		t.Errorf("heading = %q, want %q", headings[0], "Real Heading")
	}
}

// --- validDocStatus tests ---

func TestValidDocStatus_ValidStatuses(t *testing.T) {
	valid := []string{"draft", "submitted", "normalised", "approved"}
	for _, s := range valid {
		t.Run(s, func(t *testing.T) {
			if !validDocStatus(s) {
				t.Errorf("validDocStatus(%q) = false, want true", s)
			}
		})
	}
}

func TestValidDocStatus_InvalidStatuses(t *testing.T) {
	invalid := []string{"", "archived", "Draft", "SUBMITTED", "pending", "rejected"}
	for _, s := range invalid {
		name := s
		if name == "" {
			name = "empty"
		}
		t.Run(name, func(t *testing.T) {
			if validDocStatus(s) {
				t.Errorf("validDocStatus(%q) = true, want false", s)
			}
		})
	}
}

// --- GetTemplate tests ---

func TestGetTemplate_AllDocTypes(t *testing.T) {
	for _, dt := range AllDocTypes() {
		t.Run(string(dt), func(t *testing.T) {
			tmpl, err := GetTemplate(dt)
			if err != nil {
				t.Fatalf("GetTemplate(%s) returned error: %v", dt, err)
			}
			if tmpl.Type != dt {
				t.Errorf("template type = %s, want %s", tmpl.Type, dt)
			}
			if len(tmpl.RequiredSections) == 0 {
				t.Errorf("template for %s has no required sections", dt)
			}
			if tmpl.Description == "" {
				t.Errorf("template for %s has empty description", dt)
			}
		})
	}
}

func TestGetTemplate_UnknownType(t *testing.T) {
	_, err := GetTemplate(DocType("unknown-type"))
	if err == nil {
		t.Error("expected error for unknown document type, got nil")
	}
}

func TestGetTemplate_TemplatesMapCoversAllTypes(t *testing.T) {
	allTypes := AllDocTypes()
	if len(templates) != len(allTypes) {
		t.Errorf("templates map has %d entries, AllDocTypes returns %d types", len(templates), len(allTypes))
	}
	for _, dt := range allTypes {
		if _, ok := templates[dt]; !ok {
			t.Errorf("templates map missing entry for %s", dt)
		}
	}
}

// --- Scaffold tests ---

func TestScaffold_ProducesRequiredSections(t *testing.T) {
	for _, dt := range AllDocTypes() {
		t.Run(string(dt), func(t *testing.T) {
			output, err := Scaffold(dt, "My Document")
			if err != nil {
				t.Fatalf("Scaffold(%s, ...) returned error: %v", dt, err)
			}

			// Should start with a title heading
			if !strings.HasPrefix(output, "# My Document\n") {
				t.Error("scaffold output should start with '# My Document'")
			}

			// Should contain all required sections
			tmpl, _ := GetTemplate(dt)
			for _, section := range tmpl.RequiredSections {
				expected := "## " + section + "\n"
				if !strings.Contains(output, expected) {
					t.Errorf("scaffold output missing required section %q", section)
				}
			}
		})
	}
}

func TestScaffold_EmptyTitle(t *testing.T) {
	tests := []struct {
		name  string
		title string
	}{
		{"empty string", ""},
		{"whitespace only", "   "},
		{"tabs and spaces", " \t "},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Scaffold(DocTypeProposal, tc.title)
			if err == nil {
				t.Error("expected error for empty title, got nil")
			}
		})
	}
}

func TestScaffold_InvalidType(t *testing.T) {
	_, err := Scaffold(DocType("not-a-type"), "Title")
	if err == nil {
		t.Error("expected error for invalid document type, got nil")
	}
}

func TestScaffold_OutputPassesValidation(t *testing.T) {
	// A scaffolded document body should contain all required sections
	// when combined with valid metadata.
	now := time.Now()
	for _, dt := range AllDocTypes() {
		t.Run(string(dt), func(t *testing.T) {
			body, err := Scaffold(dt, "Test Doc")
			if err != nil {
				t.Fatalf("Scaffold error: %v", err)
			}

			doc := Document{
				Meta: DocMeta{
					ID:        "DOC-100",
					Type:      dt,
					Title:     "Test Doc",
					Status:    DocStatusDraft,
					CreatedBy: "tester",
					Created:   now,
					Updated:   now,
				},
				Body: body,
			}

			// Set feature ref for types that require it
			if requiresFeatureRef(dt) {
				doc.Meta.Feature = "FEAT-001"
			}

			errs := ValidateDocument(doc)
			requireNoErrors(t, errs)
		})
	}
}

// --- ValidDocType tests ---

func TestValidDocType_AllValid(t *testing.T) {
	valid := []string{
		"proposal",
		"research-report",
		"draft-design",
		"design",
		"specification",
		"implementation-plan",
		"user-documentation",
	}
	for _, s := range valid {
		t.Run(s, func(t *testing.T) {
			if !ValidDocType(s) {
				t.Errorf("ValidDocType(%q) = false, want true", s)
			}
		})
	}
}

func TestValidDocType_Invalid(t *testing.T) {
	invalid := []string{
		"",
		"Proposal",
		"DESIGN",
		"research_report",
		"unknown",
		"draft",
		"bug",
		"feature",
	}
	for _, s := range invalid {
		name := s
		if name == "" {
			name = "empty"
		}
		t.Run(name, func(t *testing.T) {
			if ValidDocType(s) {
				t.Errorf("ValidDocType(%q) = true, want false", s)
			}
		})
	}
}

// --- AllDocTypes tests ---

func TestAllDocTypes_ReturnsSevenTypes(t *testing.T) {
	types := AllDocTypes()
	if len(types) != 7 {
		t.Errorf("AllDocTypes() returned %d types, want 7", len(types))
	}
}

func TestAllDocTypes_ContainsAllExpected(t *testing.T) {
	expected := map[DocType]bool{
		DocTypeProposal:           true,
		DocTypeResearchReport:     true,
		DocTypeDraftDesign:        true,
		DocTypeDesign:             true,
		DocTypeSpecification:      true,
		DocTypeImplementationPlan: true,
		DocTypeUserDocumentation:  true,
	}

	types := AllDocTypes()
	for _, dt := range types {
		if !expected[dt] {
			t.Errorf("unexpected doc type in AllDocTypes: %s", dt)
		}
		delete(expected, dt)
	}
	for dt := range expected {
		t.Errorf("missing doc type from AllDocTypes: %s", dt)
	}
}

func TestAllDocTypes_NoDuplicates(t *testing.T) {
	seen := make(map[DocType]bool)
	for _, dt := range AllDocTypes() {
		if seen[dt] {
			t.Errorf("duplicate doc type in AllDocTypes: %s", dt)
		}
		seen[dt] = true
	}
}

// --- ValidationError.Error tests ---

func TestValidationError_ErrorWithField(t *testing.T) {
	e := ValidationError{Field: "title", Message: "required"}
	got := e.Error()
	if got != "title: required" {
		t.Errorf("Error() = %q, want %q", got, "title: required")
	}
}

func TestValidationError_ErrorWithoutField(t *testing.T) {
	e := ValidationError{Message: "something went wrong"}
	got := e.Error()
	if got != "something went wrong" {
		t.Errorf("Error() = %q, want %q", got, "something went wrong")
	}
}
