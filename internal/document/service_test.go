package document

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// submitProposal is a test helper that submits a proposal document with sensible defaults.
func submitProposal(t *testing.T, svc *DocService, title, body string) DocumentResult {
	t.Helper()
	result, err := svc.Submit(SubmitInput{
		Type:      DocTypeProposal,
		Title:     title,
		Body:      body,
		CreatedBy: "test-user",
	})
	if err != nil {
		t.Fatalf("submit proposal %q: %v", title, err)
	}
	return result
}

// submitDesign is a test helper that submits a design document (which requires a feature ref).
func submitDesign(t *testing.T, svc *DocService, title, body, feature string) DocumentResult {
	t.Helper()
	result, err := svc.Submit(SubmitInput{
		Type:      DocTypeDesign,
		Title:     title,
		Feature:   feature,
		Body:      body,
		CreatedBy: "test-user",
	})
	if err != nil {
		t.Fatalf("submit design %q: %v", title, err)
	}
	return result
}

// normalise is a test helper that normalises (UpdateBody) a submitted document.
func normalise(t *testing.T, svc *DocService, docType DocType, id, newBody string) DocumentResult {
	t.Helper()
	result, err := svc.UpdateBody(docType, id, newBody)
	if err != nil {
		t.Fatalf("normalise %s/%s: %v", docType, id, err)
	}
	return result
}

// approve is a test helper that approves a normalised document.
func approve(t *testing.T, svc *DocService, docType DocType, id string) DocumentResult {
	t.Helper()
	result, err := svc.Approve(ApproveInput{
		Type:       docType,
		ID:         id,
		ApprovedBy: "approver",
	})
	if err != nil {
		t.Fatalf("approve %s/%s: %v", docType, id, err)
	}
	return result
}

func TestDocService_FullLifecycle(t *testing.T) {
	svc := NewDocService(t.TempDir())

	body := "# My Proposal\n\n## Summary\n\nA summary.\n\n## Problem\n\nA problem.\n\n## Proposal\n\nA proposal.\n"

	// Submit
	submitted := submitProposal(t, svc, "My Proposal", body)
	if submitted.Status != DocStatusSubmitted {
		t.Errorf("expected status %s, got %s", DocStatusSubmitted, submitted.Status)
	}
	if submitted.ID == "" {
		t.Fatal("expected non-empty ID")
	}

	// Normalise (UpdateBody)
	normalisedBody := "# My Proposal (Normalised)\n\n## Summary\n\nCleaned summary.\n\n## Problem\n\nCleaned problem.\n\n## Proposal\n\nCleaned proposal.\n"
	normalised := normalise(t, svc, DocTypeProposal, submitted.ID, normalisedBody)
	if normalised.Status != DocStatusNormalised {
		t.Errorf("expected status %s, got %s", DocStatusNormalised, normalised.Status)
	}

	// Approve
	approved := approve(t, svc, DocTypeProposal, submitted.ID)
	if approved.Status != DocStatusApproved {
		t.Errorf("expected status %s, got %s", DocStatusApproved, approved.Status)
	}

	// Retrieve and verify body is verbatim
	doc, err := svc.Retrieve(DocTypeProposal, submitted.ID)
	if err != nil {
		t.Fatalf("retrieve: %v", err)
	}
	if doc.Body != normalisedBody {
		t.Errorf("body mismatch after full lifecycle\ngot:  %q\nwant: %q", doc.Body, normalisedBody)
	}
	if doc.Meta.Status != DocStatusApproved {
		t.Errorf("expected approved status, got %s", doc.Meta.Status)
	}
	if doc.Meta.ApprovedBy != "approver" {
		t.Errorf("expected approved_by=approver, got %s", doc.Meta.ApprovedBy)
	}
	if doc.Meta.ApprovedAt == nil {
		t.Error("expected approved_at to be set")
	}
}

func TestDocService_Submit_EmptyTitle(t *testing.T) {
	svc := NewDocService(t.TempDir())

	_, err := svc.Submit(SubmitInput{
		Type:      DocTypeProposal,
		Title:     "",
		Body:      "some body",
		CreatedBy: "user",
	})
	if err == nil {
		t.Fatal("expected error for empty title")
	}
	if !strings.Contains(err.Error(), "title") {
		t.Errorf("error should mention title: %v", err)
	}
}

func TestDocService_Submit_EmptyBody(t *testing.T) {
	svc := NewDocService(t.TempDir())

	_, err := svc.Submit(SubmitInput{
		Type:      DocTypeProposal,
		Title:     "A Title",
		Body:      "",
		CreatedBy: "user",
	})
	if err == nil {
		t.Fatal("expected error for empty body")
	}
	if !strings.Contains(err.Error(), "body") {
		t.Errorf("error should mention body: %v", err)
	}
}

func TestDocService_Submit_EmptyCreatedBy(t *testing.T) {
	svc := NewDocService(t.TempDir())

	_, err := svc.Submit(SubmitInput{
		Type:      DocTypeProposal,
		Title:     "A Title",
		Body:      "some body",
		CreatedBy: "",
	})
	if err == nil {
		t.Fatal("expected error for empty created_by")
	}
	if !strings.Contains(err.Error(), "created_by") {
		t.Errorf("error should mention created_by: %v", err)
	}
}

func TestDocService_Submit_InvalidType(t *testing.T) {
	svc := NewDocService(t.TempDir())

	_, err := svc.Submit(SubmitInput{
		Type:      DocType("nonsense"),
		Title:     "A Title",
		Body:      "some body",
		CreatedBy: "user",
	})
	if err == nil {
		t.Fatal("expected error for invalid type")
	}
	if !strings.Contains(err.Error(), "invalid") {
		t.Errorf("error should mention invalid: %v", err)
	}
}

func TestDocService_Submit_DesignWithoutFeature(t *testing.T) {
	svc := NewDocService(t.TempDir())

	_, err := svc.Submit(SubmitInput{
		Type:      DocTypeDesign,
		Title:     "A Design",
		Body:      "some body",
		CreatedBy: "user",
		Feature:   "",
	})
	if err == nil {
		t.Fatal("expected error for design without feature")
	}
	if !strings.Contains(err.Error(), "feature") {
		t.Errorf("error should mention feature: %v", err)
	}
}

func TestDocService_Submit_SpecWithoutFeature(t *testing.T) {
	svc := NewDocService(t.TempDir())

	_, err := svc.Submit(SubmitInput{
		Type:      DocTypeSpecification,
		Title:     "A Spec",
		Body:      "some body",
		CreatedBy: "user",
		Feature:   "",
	})
	if err == nil {
		t.Fatal("expected error for specification without feature")
	}
}

func TestDocService_Submit_ImplementationPlanWithoutFeature(t *testing.T) {
	svc := NewDocService(t.TempDir())

	_, err := svc.Submit(SubmitInput{
		Type:      DocTypeImplementationPlan,
		Title:     "A Plan",
		Body:      "some body",
		CreatedBy: "user",
		Feature:   "",
	})
	if err == nil {
		t.Fatal("expected error for implementation-plan without feature")
	}
}

func TestDocService_Submit_ProposalWithoutFeature(t *testing.T) {
	svc := NewDocService(t.TempDir())

	result, err := svc.Submit(SubmitInput{
		Type:      DocTypeProposal,
		Title:     "A Proposal",
		Body:      "some body",
		CreatedBy: "user",
	})
	if err != nil {
		t.Fatalf("proposal should not require feature: %v", err)
	}
	if result.ID == "" {
		t.Error("expected non-empty ID")
	}
}

func TestDocService_Submit_WhitespaceOnlyTitle(t *testing.T) {
	svc := NewDocService(t.TempDir())

	_, err := svc.Submit(SubmitInput{
		Type:      DocTypeProposal,
		Title:     "   ",
		Body:      "some body",
		CreatedBy: "user",
	})
	if err == nil {
		t.Fatal("expected error for whitespace-only title")
	}
}

func TestDocService_Submit_IDsAreSequential(t *testing.T) {
	svc := NewDocService(t.TempDir())

	r1 := submitProposal(t, svc, "First", "body one")
	r2 := submitProposal(t, svc, "Second", "body two")
	r3 := submitProposal(t, svc, "Third", "body three")

	if r1.ID != "DOC-001" {
		t.Errorf("expected DOC-001, got %s", r1.ID)
	}
	if r2.ID != "DOC-002" {
		t.Errorf("expected DOC-002, got %s", r2.ID)
	}
	if r3.ID != "DOC-003" {
		t.Errorf("expected DOC-003, got %s", r3.ID)
	}
}

func TestDocService_UpdateBody_SubmittedDocument(t *testing.T) {
	svc := NewDocService(t.TempDir())

	submitted := submitProposal(t, svc, "Test Doc", "original body")
	result, err := svc.UpdateBody(DocTypeProposal, submitted.ID, "normalised body")
	if err != nil {
		t.Fatalf("normalise submitted document: %v", err)
	}
	if result.Status != DocStatusNormalised {
		t.Errorf("expected status %s, got %s", DocStatusNormalised, result.Status)
	}
}

func TestDocService_UpdateBody_ApprovedDocument(t *testing.T) {
	svc := NewDocService(t.TempDir())

	submitted := submitProposal(t, svc, "Test Doc", "original body")
	normalise(t, svc, DocTypeProposal, submitted.ID, "normalised body")
	approve(t, svc, DocTypeProposal, submitted.ID)

	_, err := svc.UpdateBody(DocTypeProposal, submitted.ID, "re-normalised body")
	if err == nil {
		t.Fatal("expected error when normalising approved document")
	}
	if !strings.Contains(err.Error(), "approved") {
		t.Errorf("error should mention approved state: %v", err)
	}
}

func TestDocService_UpdateBody_NormalisedDocument(t *testing.T) {
	svc := NewDocService(t.TempDir())

	submitted := submitProposal(t, svc, "Test Doc", "original body")
	normalise(t, svc, DocTypeProposal, submitted.ID, "normalised body")

	_, err := svc.UpdateBody(DocTypeProposal, submitted.ID, "re-normalised body")
	if err == nil {
		t.Fatal("expected error when normalising already-normalised document")
	}
	if !strings.Contains(err.Error(), "normalised") {
		t.Errorf("error should mention normalised state: %v", err)
	}
}

func TestDocService_UpdateBody_NonExistentDocument(t *testing.T) {
	svc := NewDocService(t.TempDir())

	_, err := svc.UpdateBody(DocTypeProposal, "DOC-999", "body")
	if err == nil {
		t.Fatal("expected error for non-existent document")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error should mention not found: %v", err)
	}
}

func TestDocService_Approve_NormalisedDocument(t *testing.T) {
	svc := NewDocService(t.TempDir())

	submitted := submitProposal(t, svc, "Test Doc", "body")
	normalise(t, svc, DocTypeProposal, submitted.ID, "normalised body")

	result, err := svc.Approve(ApproveInput{
		Type:       DocTypeProposal,
		ID:         submitted.ID,
		ApprovedBy: "boss",
	})
	if err != nil {
		t.Fatalf("approve normalised document: %v", err)
	}
	if result.Status != DocStatusApproved {
		t.Errorf("expected status %s, got %s", DocStatusApproved, result.Status)
	}
}

func TestDocService_Approve_SubmittedDocument(t *testing.T) {
	svc := NewDocService(t.TempDir())

	submitted := submitProposal(t, svc, "Test Doc", "body")

	_, err := svc.Approve(ApproveInput{
		Type:       DocTypeProposal,
		ID:         submitted.ID,
		ApprovedBy: "boss",
	})
	if err == nil {
		t.Fatal("expected error when approving submitted document")
	}
	if !strings.Contains(err.Error(), "submitted") {
		t.Errorf("error should mention submitted state: %v", err)
	}
}

func TestDocService_Approve_DraftDocument(t *testing.T) {
	// Draft documents can't exist through the service API (Submit creates in submitted state),
	// so we test that approving a non-normalised document fails. We use a freshly submitted doc
	// which is the closest reachable non-normalised state.
	svc := NewDocService(t.TempDir())

	submitted := submitProposal(t, svc, "Test Doc", "body")

	_, err := svc.Approve(ApproveInput{
		Type:       DocTypeProposal,
		ID:         submitted.ID,
		ApprovedBy: "boss",
	})
	if err == nil {
		t.Fatal("expected error when approving non-normalised document")
	}
}

func TestDocService_Approve_EmptyApprovedBy(t *testing.T) {
	svc := NewDocService(t.TempDir())

	submitted := submitProposal(t, svc, "Test Doc", "body")
	normalise(t, svc, DocTypeProposal, submitted.ID, "normalised body")

	_, err := svc.Approve(ApproveInput{
		Type:       DocTypeProposal,
		ID:         submitted.ID,
		ApprovedBy: "",
	})
	if err == nil {
		t.Fatal("expected error for empty approved_by")
	}
	if !strings.Contains(err.Error(), "approved_by") {
		t.Errorf("error should mention approved_by: %v", err)
	}
}

func TestDocService_Approve_NonExistentDocument(t *testing.T) {
	svc := NewDocService(t.TempDir())

	_, err := svc.Approve(ApproveInput{
		Type:       DocTypeProposal,
		ID:         "DOC-999",
		ApprovedBy: "boss",
	})
	if err == nil {
		t.Fatal("expected error for non-existent document")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error should mention not found: %v", err)
	}
}

func TestDocService_Retrieve_NonExistentDocument(t *testing.T) {
	svc := NewDocService(t.TempDir())

	_, err := svc.Retrieve(DocTypeProposal, "DOC-999")
	if err == nil {
		t.Fatal("expected error for non-existent document")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error should mention not found: %v", err)
	}
}

func TestDocService_Retrieve_BodyMatchesStored(t *testing.T) {
	svc := NewDocService(t.TempDir())

	body := "This is the exact body content."
	submitted := submitProposal(t, svc, "Exact Body", body)

	doc, err := svc.Retrieve(DocTypeProposal, submitted.ID)
	if err != nil {
		t.Fatalf("retrieve: %v", err)
	}
	if doc.Body != body {
		t.Errorf("body mismatch\ngot:  %q\nwant: %q", doc.Body, body)
	}
}

func TestDocService_VerbatimRoundTrip(t *testing.T) {
	svc := NewDocService(t.TempDir())

	// Body with special characters, multiple lines, markdown, unicode
	body := "# Proposal Title\n\n" +
		"## Summary\n\n" +
		"This has **bold**, *italic*, and `code`.\n\n" +
		"## Problem\n\n" +
		"Special chars: <angle> & \"quotes\" 'apostrophes' — em-dash – en-dash\n" +
		"Unicode: café, naïve, résumé, über, 日本語\n" +
		"Symbols: @#$%^&*(){}[]|\\;:',.<>?/~`!\n\n" +
		"## Proposal\n\n" +
		"```go\nfunc main() {\n\tfmt.Println(\"hello\")\n}\n```\n\n" +
		"- bullet one\n- bullet two\n  - nested\n\n" +
		"1. numbered\n2. list\n\n" +
		"> blockquote with **formatting**\n\n" +
		"| col1 | col2 |\n|------|------|\n| a    | b    |\n"

	// Submit
	submitted := submitProposal(t, svc, "Verbatim Test", body)

	// Normalise with the same body (no changes)
	normalise(t, svc, DocTypeProposal, submitted.ID, body)

	// Approve
	approve(t, svc, DocTypeProposal, submitted.ID)

	// Retrieve and verify verbatim round-trip
	doc, err := svc.Retrieve(DocTypeProposal, submitted.ID)
	if err != nil {
		t.Fatalf("retrieve: %v", err)
	}
	if doc.Body != body {
		t.Errorf("verbatim round-trip failed\ngot:  %q\nwant: %q", doc.Body, body)
	}
}

func TestDocService_ListByType_MultipleDocuments(t *testing.T) {
	svc := NewDocService(t.TempDir())

	submitProposal(t, svc, "Proposal One", "body one")
	submitProposal(t, svc, "Proposal Two", "body two")
	submitProposal(t, svc, "Proposal Three", "body three")

	results, err := svc.ListByType(DocTypeProposal)
	if err != nil {
		t.Fatalf("list by type: %v", err)
	}
	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}

	ids := make(map[string]bool)
	for _, r := range results {
		ids[r.ID] = true
		if r.Type != DocTypeProposal {
			t.Errorf("expected type %s, got %s", DocTypeProposal, r.Type)
		}
	}
	for _, id := range []string{"DOC-001", "DOC-002", "DOC-003"} {
		if !ids[id] {
			t.Errorf("missing expected ID %s in results", id)
		}
	}
}

func TestDocService_ListByType_EmptyResult(t *testing.T) {
	svc := NewDocService(t.TempDir())

	results, err := svc.ListByType(DocTypeProposal)
	if err != nil {
		t.Fatalf("list by type: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

func TestDocService_ListByType_DoesNotIncludeOtherTypes(t *testing.T) {
	svc := NewDocService(t.TempDir())

	submitProposal(t, svc, "A Proposal", "body")
	submitDesign(t, svc, "A Design", "body", "FEAT-001")

	results, err := svc.ListByType(DocTypeProposal)
	if err != nil {
		t.Fatalf("list by type: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Type != DocTypeProposal {
		t.Errorf("expected type %s, got %s", DocTypeProposal, results[0].Type)
	}
}

func TestDocService_ListAll_MultipleTypes(t *testing.T) {
	svc := NewDocService(t.TempDir())

	submitProposal(t, svc, "Proposal", "proposal body")

	_, err := svc.Submit(SubmitInput{
		Type:      DocTypeResearchReport,
		Title:     "Research",
		Body:      "research body",
		CreatedBy: "user",
	})
	if err != nil {
		t.Fatalf("submit research report: %v", err)
	}

	submitDesign(t, svc, "Design", "design body", "FEAT-001")

	results, err := svc.ListAll()
	if err != nil {
		t.Fatalf("list all: %v", err)
	}
	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}

	types := make(map[DocType]bool)
	for _, r := range results {
		types[r.Type] = true
	}
	if !types[DocTypeProposal] {
		t.Error("missing proposal in results")
	}
	if !types[DocTypeResearchReport] {
		t.Error("missing research-report in results")
	}
	if !types[DocTypeDesign] {
		t.Error("missing design in results")
	}
}

func TestDocService_ListAll_Empty(t *testing.T) {
	svc := NewDocService(t.TempDir())

	results, err := svc.ListAll()
	if err != nil {
		t.Fatalf("list all: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

func TestDocService_ScaffoldDocument_ValidType(t *testing.T) {
	svc := NewDocService(t.TempDir())

	content, err := svc.ScaffoldDocument(DocTypeProposal, "My Proposal")
	if err != nil {
		t.Fatalf("scaffold: %v", err)
	}
	if !strings.Contains(content, "# My Proposal") {
		t.Error("scaffold should contain title heading")
	}
	if !strings.Contains(content, "## Summary") {
		t.Error("scaffold should contain Summary section")
	}
	if !strings.Contains(content, "## Problem") {
		t.Error("scaffold should contain Problem section")
	}
	if !strings.Contains(content, "## Proposal") {
		t.Error("scaffold should contain Proposal section")
	}
}

func TestDocService_ScaffoldDocument_InvalidType(t *testing.T) {
	svc := NewDocService(t.TempDir())

	_, err := svc.ScaffoldDocument(DocType("invalid"), "Title")
	if err == nil {
		t.Fatal("expected error for invalid type")
	}
}

func TestDocService_ScaffoldDocument_EmptyTitle(t *testing.T) {
	svc := NewDocService(t.TempDir())

	_, err := svc.ScaffoldDocument(DocTypeProposal, "")
	if err == nil {
		t.Fatal("expected error for empty title")
	}
}

func TestDocService_Submit_DesignWithFeature(t *testing.T) {
	svc := NewDocService(t.TempDir())

	result, err := svc.Submit(SubmitInput{
		Type:      DocTypeDesign,
		Title:     "Design Doc",
		Feature:   "FEAT-001",
		Body:      "design body",
		CreatedBy: "user",
	})
	if err != nil {
		t.Fatalf("submit design with feature: %v", err)
	}
	if result.Status != DocStatusSubmitted {
		t.Errorf("expected status %s, got %s", DocStatusSubmitted, result.Status)
	}

	doc, err := svc.Retrieve(DocTypeDesign, result.ID)
	if err != nil {
		t.Fatalf("retrieve: %v", err)
	}
	if doc.Meta.Feature != "FEAT-001" {
		t.Errorf("expected feature FEAT-001, got %s", doc.Meta.Feature)
	}
}

func TestDocService_Retrieve_MetadataAfterSubmit(t *testing.T) {
	svc := NewDocService(t.TempDir())

	result := submitProposal(t, svc, "Meta Test", "test body")

	doc, err := svc.Retrieve(DocTypeProposal, result.ID)
	if err != nil {
		t.Fatalf("retrieve: %v", err)
	}
	if doc.Meta.Title != "Meta Test" {
		t.Errorf("expected title %q, got %q", "Meta Test", doc.Meta.Title)
	}
	if doc.Meta.CreatedBy != "test-user" {
		t.Errorf("expected created_by %q, got %q", "test-user", doc.Meta.CreatedBy)
	}
	if doc.Meta.Status != DocStatusSubmitted {
		t.Errorf("expected status %s, got %s", DocStatusSubmitted, doc.Meta.Status)
	}
	if doc.Meta.Created.IsZero() {
		t.Error("expected created to be set")
	}
	if doc.Meta.Updated.IsZero() {
		t.Error("expected updated to be set")
	}
}

func TestDocService_Submit_ResultFields(t *testing.T) {
	svc := NewDocService(t.TempDir())

	result := submitProposal(t, svc, "Result Fields", "body")

	if result.ID == "" {
		t.Error("expected non-empty ID")
	}
	if result.Type != DocTypeProposal {
		t.Errorf("expected type %s, got %s", DocTypeProposal, result.Type)
	}
	if result.Title != "Result Fields" {
		t.Errorf("expected title %q, got %q", "Result Fields", result.Title)
	}
	if result.Status != DocStatusSubmitted {
		t.Errorf("expected status %s, got %s", DocStatusSubmitted, result.Status)
	}
	if result.Path == "" {
		t.Error("expected non-empty path")
	}
}

func TestDocService_Validate_ValidDocument(t *testing.T) {
	svc := NewDocService(t.TempDir())

	body := "# Test\n\n## Summary\n\nA summary.\n\n## Problem\n\nA problem.\n\n## Proposal\n\nA proposal.\n"
	submitted := submitProposal(t, svc, "Valid Doc", body)

	doc, err := svc.Retrieve(DocTypeProposal, submitted.ID)
	if err != nil {
		t.Fatalf("retrieve: %v", err)
	}

	errs := svc.Validate(doc)
	if len(errs) != 0 {
		t.Errorf("expected no validation errors, got %d: %v", len(errs), errs)
	}
}

func TestDocService_Validate_MissingSections(t *testing.T) {
	svc := NewDocService(t.TempDir())

	// Body missing required sections for proposal
	body := "# No Sections Here\n\nJust some text.\n"
	submitted := submitProposal(t, svc, "Invalid Sections", body)

	doc, err := svc.Retrieve(DocTypeProposal, submitted.ID)
	if err != nil {
		t.Fatalf("retrieve: %v", err)
	}

	errs := svc.Validate(doc)
	if len(errs) == 0 {
		t.Error("expected validation errors for missing sections")
	}

	errStr := ""
	for _, e := range errs {
		errStr += e.Error() + " "
	}
	if !strings.Contains(errStr, "Summary") {
		t.Error("expected error about missing Summary section")
	}
}

func TestDocService_ExtractFromDocument_ApprovedSuccess(t *testing.T) {
	svc := NewDocService(t.TempDir())

	body := "# Approved Proposal\n\n## Summary\n\nReady for extraction.\n"
	submitted := submitProposal(t, svc, "Approved Proposal", body)
	normalise(t, svc, DocTypeProposal, submitted.ID, body)
	approve(t, svc, DocTypeProposal, submitted.ID)

	doc, err := svc.ExtractFromDocument(submitted.ID)
	if err != nil {
		t.Fatalf("extract from approved document: %v", err)
	}
	if doc.Meta.ID != submitted.ID {
		t.Errorf("expected ID %q, got %q", submitted.ID, doc.Meta.ID)
	}
	if doc.Meta.Status != DocStatusApproved {
		t.Errorf("expected status %s, got %s", DocStatusApproved, doc.Meta.Status)
	}
	if doc.Body != body {
		t.Errorf("body mismatch\ngot:  %q\nwant: %q", doc.Body, body)
	}
}

func TestDocService_ExtractFromDocument_NonApprovedError(t *testing.T) {
	svc := NewDocService(t.TempDir())

	submitted := submitProposal(t, svc, "Submitted Proposal", "body")

	_, err := svc.ExtractFromDocument(submitted.ID)
	if err == nil {
		t.Fatal("expected error for non-approved document")
	}
	if !strings.Contains(err.Error(), "approved") {
		t.Errorf("error should mention approved state: %v", err)
	}
}

func TestDocService_ExtractFromDocument_NotFoundError(t *testing.T) {
	svc := NewDocService(t.TempDir())

	_, err := svc.ExtractFromDocument("DOC-999")
	if err == nil {
		t.Fatal("expected error for non-existent document")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error should mention not found: %v", err)
	}
}

func TestDocService_UpdateBody_PreservesMetadata(t *testing.T) {
	svc := NewDocService(t.TempDir())

	submitted := submitProposal(t, svc, "Preserve Meta", "original body")

	docBefore, err := svc.Retrieve(DocTypeProposal, submitted.ID)
	if err != nil {
		t.Fatalf("retrieve before: %v", err)
	}

	normalise(t, svc, DocTypeProposal, submitted.ID, "new body")

	docAfter, err := svc.Retrieve(DocTypeProposal, submitted.ID)
	if err != nil {
		t.Fatalf("retrieve after: %v", err)
	}

	if docAfter.Meta.ID != docBefore.Meta.ID {
		t.Error("ID changed after normalisation")
	}
	if docAfter.Meta.Title != docBefore.Meta.Title {
		t.Error("title changed after normalisation")
	}
	if docAfter.Meta.CreatedBy != docBefore.Meta.CreatedBy {
		t.Error("created_by changed after normalisation")
	}
	if docAfter.Meta.Created != docBefore.Meta.Created {
		t.Error("created time changed after normalisation")
	}
	if docAfter.Body != "new body" {
		t.Errorf("body not updated: got %q", docAfter.Body)
	}
}

func TestDocService_Approve_PreservesBody(t *testing.T) {
	svc := NewDocService(t.TempDir())

	submitted := submitProposal(t, svc, "Preserve Body", "original body")
	normalise(t, svc, DocTypeProposal, submitted.ID, "normalised body content")
	approve(t, svc, DocTypeProposal, submitted.ID)

	doc, err := svc.Retrieve(DocTypeProposal, submitted.ID)
	if err != nil {
		t.Fatalf("retrieve: %v", err)
	}
	if doc.Body != "normalised body content" {
		t.Errorf("body changed after approval: got %q", doc.Body)
	}
}

func TestDocService_ListByType_StatusReflectsLifecycle(t *testing.T) {
	svc := NewDocService(t.TempDir())

	r1 := submitProposal(t, svc, "Doc One", "body one")
	r2 := submitProposal(t, svc, "Doc Two", "body two")
	normalise(t, svc, DocTypeProposal, r2.ID, "normalised two")

	results, err := svc.ListByType(DocTypeProposal)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	statuses := make(map[string]DocStatus)
	for _, r := range results {
		statuses[r.ID] = r.Status
	}

	if statuses[r1.ID] != DocStatusSubmitted {
		t.Errorf("expected %s status %s, got %s", r1.ID, DocStatusSubmitted, statuses[r1.ID])
	}
	if statuses[r2.ID] != DocStatusNormalised {
		t.Errorf("expected %s status %s, got %s", r2.ID, DocStatusNormalised, statuses[r2.ID])
	}
}

func TestDocService_Approve_AlreadyApproved(t *testing.T) {
	svc := NewDocService(t.TempDir())

	submitted := submitProposal(t, svc, "Test Doc", "body")
	normalise(t, svc, DocTypeProposal, submitted.ID, "normalised body")
	approve(t, svc, DocTypeProposal, submitted.ID)

	_, err := svc.Approve(ApproveInput{
		Type:       DocTypeProposal,
		ID:         submitted.ID,
		ApprovedBy: "another-boss",
	})
	if err == nil {
		t.Fatal("expected error when approving already-approved document")
	}
}

func TestRemoveIfDifferent_RemovesOldFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	oldPath := filepath.Join(dir, "old-file.md")
	newPath := filepath.Join(dir, "new-file.md")

	if err := os.WriteFile(oldPath, []byte("old"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	if err := os.WriteFile(newPath, []byte("new"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	if err := removeIfDifferent(oldPath, newPath); err != nil {
		t.Fatalf("removeIfDifferent() error = %v", err)
	}

	if _, err := os.Stat(oldPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("old file should have been removed, but Stat error = %v", err)
	}

	if _, err := os.Stat(newPath); err != nil {
		t.Fatalf("new file should still exist, but Stat error = %v", err)
	}
}

func TestRemoveIfDifferent_SamePathIsNoOp(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "same-file.md")

	if err := os.WriteFile(path, []byte("content"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	if err := removeIfDifferent(path, path); err != nil {
		t.Fatalf("removeIfDifferent() error = %v", err)
	}

	if _, err := os.Stat(path); err != nil {
		t.Fatalf("file should still exist when paths are the same, but Stat error = %v", err)
	}
}

func TestDocService_IDsDoNotCollideAcrossInstances(t *testing.T) {
	t.Parallel()

	root := t.TempDir()

	// First service instance creates two documents.
	svc1 := NewDocService(root)
	body := "# Test\n\n## Summary\n\nTest.\n\n## Problem\n\nTest.\n\n## Proposal\n\nTest.\n"
	_, err := svc1.Submit(SubmitInput{
		Type:      DocTypeProposal,
		Title:     "First",
		Body:      body,
		CreatedBy: "sam",
	})
	if err != nil {
		t.Fatalf("Submit() error = %v", err)
	}
	_, err = svc1.Submit(SubmitInput{
		Type:      DocTypeProposal,
		Title:     "Second",
		Body:      body,
		CreatedBy: "sam",
	})
	if err != nil {
		t.Fatalf("Submit() error = %v", err)
	}

	// Second service instance (simulating restart) should continue from DOC-003.
	svc2 := NewDocService(root)
	result, err := svc2.Submit(SubmitInput{
		Type:      DocTypeProposal,
		Title:     "Third",
		Body:      body,
		CreatedBy: "sam",
	})
	if err != nil {
		t.Fatalf("Submit() error = %v", err)
	}

	if result.ID != "DOC-003" {
		t.Fatalf("expected DOC-003 after restart, got %s", result.ID)
	}
}
