package service

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/sambeau/kanbanzai/internal/model"
	"github.com/sambeau/kanbanzai/internal/storage"
	"github.com/sambeau/kanbanzai/internal/validate"
)

// setupDecomposeTest creates entity and document services with a feature
// that has a linked spec document containing the given content.
// Returns the decompose service, the feature ID, and the feature slug.
func setupDecomposeTest(t *testing.T, specContent string) (*DecomposeService, string, string) {
	t.Helper()

	stateRoot := t.TempDir()
	repoRoot := t.TempDir()

	entitySvc := NewEntityService(stateRoot)
	docSvc := NewDocumentService(stateRoot, repoRoot)

	// Write a plan directly to disk, bypassing the prefix registry/allocator.
	planID := "P1-decompose-plan"
	writeDecomposeTestPlan(t, entitySvc, planID)

	// Create a feature under the plan.
	featResult, err := entitySvc.CreateFeature(CreateFeatureInput{
		Slug:      "test-feature",
		Parent:    planID,
		Summary:   "Test feature for decompose",
		CreatedBy: "tester",
		Name:      "Test feature",
	})
	if err != nil {
		t.Fatalf("create feature: %v", err)
	}

	featureID := featResult.ID
	featureSlug := "test-feature"

	if specContent != "" {
		// Write the spec document to disk.
		specPath := "work/spec/test-spec.md"
		fullPath := filepath.Join(repoRoot, specPath)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
			t.Fatalf("mkdir for spec: %v", err)
		}
		if err := os.WriteFile(fullPath, []byte(specContent), 0o644); err != nil {
			t.Fatalf("write spec file: %v", err)
		}

		// Submit the spec document via the document service.
		docResult, err := docSvc.SubmitDocument(SubmitDocumentInput{
			Path:      specPath,
			Type:      "specification",
			Title:     "Test Specification",
			Owner:     featureID,
			CreatedBy: "tester",
		})
		if err != nil {
			t.Fatalf("submit spec document: %v", err)
		}

		// Approve the spec so it passes the approval gate in DecomposeFeature.
		if _, err := docSvc.ApproveDocument(ApproveDocumentInput{
			ID:         docResult.ID,
			ApprovedBy: "tester",
		}); err != nil {
			t.Fatalf("approve spec document: %v", err)
		}

		// Manually link the spec document to the feature via UpdateEntity.
		_, err = entitySvc.UpdateEntity(UpdateEntityInput{
			Type:   "feature",
			ID:     featureID,
			Slug:   featureSlug,
			Fields: map[string]string{"spec": docResult.ID},
		})
		if err != nil {
			t.Fatalf("link spec to feature: %v", err)
		}
	}

	decomposeSvc := NewDecomposeService(entitySvc, docSvc)
	return decomposeSvc, featureID, featureSlug
}

// ---------------------------------------------------------------------------
// B.10: decompose_feature tests
// ---------------------------------------------------------------------------

func TestDecomposeFeature_NoSpecRegistered(t *testing.T) {
	t.Parallel()

	// Set up a feature with no spec linked (pass empty content to skip doc creation).
	svc, featureID, _ := setupDecomposeTest(t, "")

	_, err := svc.DecomposeFeature(DecomposeInput{FeatureID: featureID})
	if err == nil {
		t.Fatal("expected error when feature has no linked spec, got nil")
	}
	want := "has no linked specification document"
	if got := err.Error(); !contains(got, want) {
		t.Errorf("error = %q, want it to contain %q", got, want)
	}
}

func TestDecomposeFeature_ProposalProduced(t *testing.T) {
	t.Parallel()

	specContent := `# Feature Spec

## Authentication

### Acceptance Criteria

- [ ] Users can log in with email and password
- [ ] Users can reset their password via email
- [ ] Sessions expire after 24 hours of inactivity

## Authorization

- [ ] Role-based access control is enforced on all API endpoints
- [ ] Admin users can manage other users
`

	svc, featureID, _ := setupDecomposeTest(t, specContent)

	result, err := svc.DecomposeFeature(DecomposeInput{FeatureID: featureID})
	if err != nil {
		t.Fatalf("DecomposeFeature() error = %v", err)
	}

	// Should not write any tasks — just return a proposal.
	if result.FeatureID == "" {
		t.Error("result.FeatureID is empty")
	}
	if result.SpecDocumentID == "" {
		t.Error("result.SpecDocumentID is empty")
	}

	// Proposal should contain tasks derived from the 5 acceptance criteria.
	// With section-based grouping (3 ACs in Authentication, 2 in Authorization),
	// we get 2 grouped tasks + 1 test task = 3 total.
	proposal := result.Proposal
	if proposal.TotalTasks < 2 {
		t.Errorf("TotalTasks = %d, want at least 2 (grouped tasks from acceptance criteria)", proposal.TotalTasks)
	}

	// Each proposed task must have slug, summary, rationale, and a non-empty name.
	for i, task := range proposal.Tasks {
		if task.Slug == "" {
			t.Errorf("task[%d].Slug is empty", i)
		}
		if task.Summary == "" {
			t.Errorf("task[%d].Summary is empty", i)
		}
		if task.Rationale == "" {
			t.Errorf("task[%d].Rationale is empty", i)
		}
		if task.Name == "" {
			t.Errorf("task[%d].Name is empty (AC-01: every proposed task must have a non-empty name)", i)
		}
	}

	// Slices should be identified from level-2 headers.
	if len(proposal.Slices) == 0 {
		t.Error("expected non-empty Slices, got none")
	}
	foundAuth := false
	foundAuthz := false
	for _, s := range proposal.Slices {
		if s == "Authentication" {
			foundAuth = true
		}
		if s == "Authorization" {
			foundAuthz = true
		}
	}
	if !foundAuth {
		t.Errorf("Slices = %v, want it to contain %q", proposal.Slices, "Authentication")
	}
	if !foundAuthz {
		t.Errorf("Slices = %v, want it to contain %q", proposal.Slices, "Authorization")
	}
}

func TestDecomposeFeature_GuidanceApplied(t *testing.T) {
	t.Parallel()

	specContent := `# Spec

## Data Layer

- [ ] Database schema is created
- [ ] Migration scripts run without errors
`

	svc, featureID, _ := setupDecomposeTest(t, specContent)

	result, err := svc.DecomposeFeature(DecomposeInput{FeatureID: featureID})
	if err != nil {
		t.Fatalf("DecomposeFeature() error = %v", err)
	}

	// guidance_applied should list the decomposition rules that influenced output.
	if len(result.GuidanceApplied) == 0 {
		t.Fatal("GuidanceApplied is empty, want at least one rule")
	}

	// 2 ACs in one section → section-based grouping → "group-by-section" rule.
	expectedRules := []string{
		"group-by-section",
		"size-soft-limit-8",
		"explicit-dependencies",
		"role-assignment",
		"test-tasks-explicit",
	}
	for _, rule := range expectedRules {
		found := false
		for _, applied := range result.GuidanceApplied {
			if applied == rule {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("GuidanceApplied = %v, want it to contain %q", result.GuidanceApplied, rule)
		}
	}
}

func TestDecomposeFeature_EmptyFeatureID(t *testing.T) {
	t.Parallel()

	svc, _, _ := setupDecomposeTest(t, "")
	_, err := svc.DecomposeFeature(DecomposeInput{FeatureID: ""})
	if err == nil {
		t.Fatal("expected error for empty feature_id, got nil")
	}
}

func TestDecomposeFeature_ContextPassed(t *testing.T) {
	t.Parallel()

	specContent := `# Spec
- [ ] Basic functionality works
`
	svc, featureID, _ := setupDecomposeTest(t, specContent)

	result, err := svc.DecomposeFeature(DecomposeInput{
		FeatureID: featureID,
		Context:   "Focus on API endpoints first",
	})
	if err != nil {
		t.Fatalf("DecomposeFeature() error = %v", err)
	}

	// Warnings should include a note about the additional context.
	found := false
	for _, w := range result.Proposal.Warnings {
		if contains(w, "Focus on API endpoints first") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Warnings = %v, want it to mention the provided context", result.Proposal.Warnings)
	}
}

func TestDecomposeFeature_DraftSpec_ReturnsError(t *testing.T) {
	t.Parallel()

	stateRoot := t.TempDir()
	repoRoot := t.TempDir()

	entitySvc := NewEntityService(stateRoot)
	docSvc := NewDocumentService(stateRoot, repoRoot)

	planID := "P1-decompose-plan"
	writeDecomposeTestPlan(t, entitySvc, planID)

	featResult, err := entitySvc.CreateFeature(CreateFeatureInput{
		Slug:      "test-feature",
		Parent:    planID,
		Summary:   "Test feature",
		CreatedBy: "tester",
		Name:      "Test feature",
	})
	if err != nil {
		t.Fatalf("create feature: %v", err)
	}

	specPath := "work/spec/draft-spec.md"
	fullPath := repoRoot + "/" + specPath
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(fullPath, []byte("# Spec\n- [ ] Something works\n"), 0o644); err != nil {
		t.Fatalf("write spec: %v", err)
	}

	// Submit but deliberately do NOT approve.
	docResult, err := docSvc.SubmitDocument(SubmitDocumentInput{
		Path:      specPath,
		Type:      "specification",
		Title:     "Draft Spec",
		Owner:     featResult.ID,
		CreatedBy: "tester",
	})
	if err != nil {
		t.Fatalf("submit spec: %v", err)
	}

	if _, err := entitySvc.UpdateEntity(UpdateEntityInput{
		Type:   "feature",
		ID:     featResult.ID,
		Slug:   "test-feature",
		Fields: map[string]string{"spec": docResult.ID},
	}); err != nil {
		t.Fatalf("link spec: %v", err)
	}

	svc := NewDecomposeService(entitySvc, docSvc)
	_, err = svc.DecomposeFeature(DecomposeInput{FeatureID: featResult.ID})
	if err == nil {
		t.Fatal("expected error for draft spec, got nil")
	}
	if !contains(err.Error(), "approve the spec before decomposing") {
		t.Errorf("error = %q, want it to contain %q", err.Error(), "approve the spec before decomposing")
	}
}

func TestDecomposeFeature_NoACs_ReturnsError(t *testing.T) {
	t.Parallel()

	specContent := `# Feature Spec

## Database Layer

Design the database schema.

## API Layer

Implement REST endpoints.
`
	svc, featureID, _ := setupDecomposeTest(t, specContent)

	_, err := svc.DecomposeFeature(DecomposeInput{FeatureID: featureID})
	if err == nil {
		t.Fatal("expected error when spec has no acceptance criteria, got nil")
	}
	if !contains(err.Error(), "no acceptance criteria found in spec") {
		t.Errorf("error = %q, want it to contain %q", err.Error(), "no acceptance criteria found in spec")
	}
}

func TestDecomposeFeature_TestTaskAdded(t *testing.T) {
	t.Parallel()

	specContent := `# Spec
- [ ] User registration works correctly
- [ ] Email verification is sent
`
	svc, featureID, _ := setupDecomposeTest(t, specContent)

	result, err := svc.DecomposeFeature(DecomposeInput{FeatureID: featureID})
	if err != nil {
		t.Fatalf("DecomposeFeature() error = %v", err)
	}

	// Should include an explicit test task since none of the ACs mention "test".
	hasTestTask := false
	for _, task := range result.Proposal.Tasks {
		if contains(task.Summary, "test") || contains(task.Summary, "Test") {
			hasTestTask = true
			break
		}
	}
	if !hasTestTask {
		t.Error("expected an explicit test task in proposal, found none")
	}
}

// ---------------------------------------------------------------------------
// B.11: decompose_review tests
// ---------------------------------------------------------------------------

func TestReviewProposal_Pass(t *testing.T) {
	t.Parallel()

	specContent := `# Spec
- [ ] Users can log in
- [ ] Users can log out
`
	svc, featureID, _ := setupDecomposeTest(t, specContent)

	proposal := Proposal{
		Tasks: []ProposedTask{
			{
				Slug:      "login",
				Summary:   "Implement user login with email and password",
				Rationale: "Covers acceptance criterion: users can log in. Verified by testing.",
			},
			{
				Slug:      "logout",
				Summary:   "Implement user logout and session cleanup",
				Rationale: "Covers acceptance criterion: users can log out",
			},
		},
		TotalTasks: 2,
	}

	result, err := svc.ReviewProposal(DecomposeReviewInput{
		FeatureID: featureID,
		Proposal:  proposal,
	})
	if err != nil {
		t.Fatalf("ReviewProposal() error = %v", err)
	}

	if result.Status != "pass" {
		t.Errorf("Status = %q, want %q", result.Status, "pass")
	}
	if result.BlockingCount != 0 {
		t.Errorf("BlockingCount = %d, want 0", result.BlockingCount)
	}
}

func TestReviewProposal_GapFinding(t *testing.T) {
	t.Parallel()

	specContent := `# Spec
- [ ] Users can log in with email and password
- [ ] Users can reset their password
- [ ] Sessions expire after inactivity
`
	svc, featureID, _ := setupDecomposeTest(t, specContent)

	// Proposal only covers login — missing password reset and session expiry.
	proposal := Proposal{
		Tasks: []ProposedTask{
			{
				Slug:      "login",
				Summary:   "Implement user login with email and password",
				Rationale: "Covers: users can log in with email and password",
			},
		},
		TotalTasks: 1,
	}

	result, err := svc.ReviewProposal(DecomposeReviewInput{
		FeatureID: featureID,
		Proposal:  proposal,
	})
	if err != nil {
		t.Fatalf("ReviewProposal() error = %v", err)
	}

	if result.Status != "fail" {
		t.Errorf("Status = %q, want %q (gaps are blocking)", result.Status, "fail")
	}

	gapCount := 0
	for _, f := range result.Findings {
		if f.Type == "gap" {
			gapCount++
			if f.Severity != "error" {
				t.Errorf("gap finding severity = %q, want %q", f.Severity, "error")
			}
		}
	}
	if gapCount < 2 {
		t.Errorf("gap findings = %d, want at least 2 (password reset, session expiry)", gapCount)
	}
	if result.BlockingCount < 2 {
		t.Errorf("BlockingCount = %d, want at least 2", result.BlockingCount)
	}
}

func TestReviewProposal_OversizedFinding(t *testing.T) {
	t.Parallel()

	specContent := `# Spec
- [ ] Feature is implemented
`
	svc, featureID, _ := setupDecomposeTest(t, specContent)

	bigEstimate := 13.0
	proposal := Proposal{
		Tasks: []ProposedTask{
			{
				Slug:      "big-task",
				Summary:   "Implement the entire feature in one monolithic task that is implemented",
				Estimate:  &bigEstimate,
				Rationale: "Covers: feature is implemented",
			},
		},
		TotalTasks: 1,
	}

	result, err := svc.ReviewProposal(DecomposeReviewInput{
		FeatureID: featureID,
		Proposal:  proposal,
	})
	if err != nil {
		t.Fatalf("ReviewProposal() error = %v", err)
	}

	oversizedCount := 0
	for _, f := range result.Findings {
		if f.Type == "oversized" {
			oversizedCount++
			if f.TaskSlug != "big-task" {
				t.Errorf("oversized finding task_slug = %q, want %q", f.TaskSlug, "big-task")
			}
			if f.Severity != "warning" {
				t.Errorf("oversized finding severity = %q, want %q", f.Severity, "warning")
			}
		}
	}
	if oversizedCount != 1 {
		t.Errorf("oversized findings = %d, want 1", oversizedCount)
	}
}

func TestReviewProposal_CycleFinding(t *testing.T) {
	t.Parallel()

	specContent := `# Spec
- [ ] Task alpha is done
- [ ] Task beta is done
- [ ] Task gamma is done
`
	svc, featureID, _ := setupDecomposeTest(t, specContent)

	// Create a dependency cycle: alpha → beta → gamma → alpha
	proposal := Proposal{
		Tasks: []ProposedTask{
			{
				Slug:      "alpha",
				Summary:   "Task alpha is done and depends on gamma",
				DependsOn: []string{"gamma"},
				Rationale: "Covers: task alpha is done",
			},
			{
				Slug:      "beta",
				Summary:   "Task beta is done and depends on alpha",
				DependsOn: []string{"alpha"},
				Rationale: "Covers: task beta is done",
			},
			{
				Slug:      "gamma",
				Summary:   "Task gamma is done and depends on beta",
				DependsOn: []string{"beta"},
				Rationale: "Covers: task gamma is done",
			},
		},
		TotalTasks: 3,
	}

	result, err := svc.ReviewProposal(DecomposeReviewInput{
		FeatureID: featureID,
		Proposal:  proposal,
	})
	if err != nil {
		t.Fatalf("ReviewProposal() error = %v", err)
	}

	if result.Status != "fail" {
		t.Errorf("Status = %q, want %q (cycles are blocking)", result.Status, "fail")
	}

	cycleCount := 0
	for _, f := range result.Findings {
		if f.Type == "cycle" {
			cycleCount++
			if f.Severity != "error" {
				t.Errorf("cycle finding severity = %q, want %q", f.Severity, "error")
			}
		}
	}
	if cycleCount == 0 {
		t.Error("expected at least one cycle finding, got none")
	}
}

func TestReviewProposal_FailWhenBlockingFindings(t *testing.T) {
	t.Parallel()

	specContent := `# Spec
- [ ] Critical feature A is implemented
- [ ] Critical feature B is implemented
`
	svc, featureID, _ := setupDecomposeTest(t, specContent)

	// Empty proposal — all ACs are gaps.
	proposal := Proposal{
		Tasks:      []ProposedTask{},
		TotalTasks: 0,
	}

	result, err := svc.ReviewProposal(DecomposeReviewInput{
		FeatureID: featureID,
		Proposal:  proposal,
	})
	if err != nil {
		t.Fatalf("ReviewProposal() error = %v", err)
	}

	if result.Status != "fail" {
		t.Errorf("Status = %q, want %q", result.Status, "fail")
	}
	if result.BlockingCount == 0 {
		t.Error("BlockingCount = 0, want > 0")
	}
}

func TestReviewProposal_WarnForNonBlockingFindings(t *testing.T) {
	t.Parallel()

	specContent := `# Spec
- [ ] Feature is implemented
`
	svc, featureID, _ := setupDecomposeTest(t, specContent)

	estimate := 10.0
	proposal := Proposal{
		Tasks: []ProposedTask{
			{
				Slug:      "the-task",
				Summary:   "Implement the feature that is implemented correctly",
				Estimate:  &estimate,
				Rationale: "Covers: feature is implemented",
			},
		},
		TotalTasks: 1,
	}

	result, err := svc.ReviewProposal(DecomposeReviewInput{
		FeatureID: featureID,
		Proposal:  proposal,
	})
	if err != nil {
		t.Fatalf("ReviewProposal() error = %v", err)
	}

	// Oversized is non-blocking, so status should be "warn" not "fail".
	if result.Status != "warn" {
		t.Errorf("Status = %q, want %q", result.Status, "warn")
	}
	if result.BlockingCount != 0 {
		t.Errorf("BlockingCount = %d, want 0", result.BlockingCount)
	}
	if result.TotalFindings == 0 {
		t.Error("TotalFindings = 0, want > 0 (oversized)")
	}
}

func TestReviewProposal_AmbiguousSummary(t *testing.T) {
	t.Parallel()

	specContent := `# Spec
- [ ] Fix it
`
	svc, featureID, _ := setupDecomposeTest(t, specContent)

	proposal := Proposal{
		Tasks: []ProposedTask{
			{
				Slug:      "fix",
				Summary:   "Fix it",
				Rationale: "Covers: fix it",
			},
		},
		TotalTasks: 1,
	}

	result, err := svc.ReviewProposal(DecomposeReviewInput{
		FeatureID: featureID,
		Proposal:  proposal,
	})
	if err != nil {
		t.Fatalf("ReviewProposal() error = %v", err)
	}

	ambiguousCount := 0
	for _, f := range result.Findings {
		if f.Type == "ambiguous" {
			ambiguousCount++
			if f.Severity != "warning" {
				t.Errorf("ambiguous finding severity = %q, want %q", f.Severity, "warning")
			}
		}
	}
	if ambiguousCount == 0 {
		t.Error("expected ambiguous finding for very short summary, got none")
	}
}

// ---------------------------------------------------------------------------
// Spec structure parsing unit tests
// ---------------------------------------------------------------------------

func TestParseSpecStructure_Checkboxes(t *testing.T) {
	t.Parallel()

	content := `# My Spec

## Section One

- [ ] First criterion
- [x] Second criterion (already done)
- [ ] Third criterion

## Section Two

- [ ] Fourth criterion
`
	spec := parseSpecStructure(content)

	if len(spec.acceptanceCriteria) != 4 {
		t.Fatalf("acceptance criteria count = %d, want 4", len(spec.acceptanceCriteria))
	}

	want := []string{"First criterion", "Second criterion (already done)", "Third criterion", "Fourth criterion"}
	for i, ac := range spec.acceptanceCriteria {
		if ac.text != want[i] {
			t.Errorf("ac[%d].text = %q, want %q", i, ac.text, want[i])
		}
	}

	// Check section association.
	if spec.acceptanceCriteria[0].section != "Section One" {
		t.Errorf("ac[0].section = %q, want %q", spec.acceptanceCriteria[0].section, "Section One")
	}
	if spec.acceptanceCriteria[3].section != "Section Two" {
		t.Errorf("ac[3].section = %q, want %q", spec.acceptanceCriteria[3].section, "Section Two")
	}
}

func TestParseSpecStructure_NumberedInACSection(t *testing.T) {
	t.Parallel()

	content := `# Spec

## Acceptance Criteria

1. First requirement
2. Second requirement

## Implementation

This section has numbered steps but is not an AC section.

1. Step one
2. Step two
`
	spec := parseSpecStructure(content)

	if len(spec.acceptanceCriteria) != 2 {
		t.Fatalf("acceptance criteria count = %d, want 2 (only from AC section)", len(spec.acceptanceCriteria))
	}
}

func TestParseSpecStructure_Sections(t *testing.T) {
	t.Parallel()

	content := `# Title

## First Section

### Subsection

## Second Section
`
	spec := parseSpecStructure(content)

	if len(spec.sections) != 4 {
		t.Fatalf("sections count = %d, want 4", len(spec.sections))
	}

	if spec.sections[0].title != "Title" || spec.sections[0].level != 1 {
		t.Errorf("section[0] = {%q, %d}, want {%q, %d}", spec.sections[0].title, spec.sections[0].level, "Title", 1)
	}
	if spec.sections[1].title != "First Section" || spec.sections[1].level != 2 {
		t.Errorf("section[1] = {%q, %d}, want {%q, %d}", spec.sections[1].title, spec.sections[1].level, "First Section", 2)
	}
}

func TestSlugify(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input string
		want  string
	}{
		{"Users can log in", "users-can-log-in"},
		{"Role-based access control", "role-based-access-control"},
		{"  spaces  everywhere  ", "spaces-everywhere"},
		{"UPPERCASE", "uppercase"},
		{"", ""},
		{"a!b@c#d$e", "a-b-c-d-e"},
	}

	for _, tc := range tests {
		got := slugify(tc.input)
		if got != tc.want {
			t.Errorf("slugify(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestCheckCycles_NoCycle(t *testing.T) {
	t.Parallel()

	proposal := Proposal{
		Tasks: []ProposedTask{
			{Slug: "a", DependsOn: []string{}},
			{Slug: "b", DependsOn: []string{"a"}},
			{Slug: "c", DependsOn: []string{"b"}},
		},
	}
	findings := checkCycles(proposal)
	if len(findings) != 0 {
		t.Errorf("expected no cycle findings, got %d: %v", len(findings), findings)
	}
}

func TestCheckCycles_WithCycle(t *testing.T) {
	t.Parallel()

	proposal := Proposal{
		Tasks: []ProposedTask{
			{Slug: "a", DependsOn: []string{"c"}},
			{Slug: "b", DependsOn: []string{"a"}},
			{Slug: "c", DependsOn: []string{"b"}},
		},
	}
	findings := checkCycles(proposal)
	if len(findings) == 0 {
		t.Error("expected cycle finding, got none")
	}
	if findings[0].Type != "cycle" {
		t.Errorf("finding type = %q, want %q", findings[0].Type, "cycle")
	}
}

func TestCheckOversized(t *testing.T) {
	t.Parallel()

	small := 5.0
	big := 13.0
	proposal := Proposal{
		Tasks: []ProposedTask{
			{Slug: "small-task", Estimate: &small},
			{Slug: "big-task", Estimate: &big},
			{Slug: "no-estimate"},
		},
	}

	findings := checkOversized(proposal)
	if len(findings) != 1 {
		t.Fatalf("oversized findings = %d, want 1", len(findings))
	}
	if findings[0].TaskSlug != "big-task" {
		t.Errorf("finding task_slug = %q, want %q", findings[0].TaskSlug, "big-task")
	}
}

func TestEstimatedTotal(t *testing.T) {
	t.Parallel()

	specContent := `# Spec
- [ ] First task
- [ ] Second task
`
	svc, featureID, _ := setupDecomposeTest(t, specContent)

	result, err := svc.DecomposeFeature(DecomposeInput{FeatureID: featureID})
	if err != nil {
		t.Fatalf("DecomposeFeature() error = %v", err)
	}

	// Initial proposal has no estimates, so EstimatedTotal should be nil.
	if result.Proposal.EstimatedTotal != nil {
		t.Errorf("EstimatedTotal = %v, want nil (no estimates assigned)", result.Proposal.EstimatedTotal)
	}
}

// ---------------------------------------------------------------------------
// Track F: Vertical slice analysis tests (§16.5)
// ---------------------------------------------------------------------------

func TestSliceAnalysis_NoSpecReturnsError(t *testing.T) {
	t.Parallel()
	svc, featureID, _ := setupDecomposeTest(t, "")

	_, err := svc.SliceAnalysis(SliceAnalysisInput{FeatureID: featureID})
	if err == nil {
		t.Fatal("expected error when feature has no linked spec, got nil")
	}
	want := "has no linked specification document"
	if got := err.Error(); !contains(got, want) {
		t.Errorf("error = %q, want it to contain %q", got, want)
	}
}

func TestSliceAnalysis_MultiCriterionSpec(t *testing.T) {
	t.Parallel()

	specContent := `# User Management

## Authentication

### Acceptance Criteria

- [ ] Users can log in with email and password
- [ ] Users can reset their password via email link
- [ ] Sessions expire after 24 hours of inactivity

## Profile Management

### Acceptance Criteria

- [ ] Users can update their display name
- [ ] Users can upload an avatar image to the storage service

## Admin Dashboard

### Acceptance Criteria

- [ ] Admin users can list all users via the API endpoint
- [ ] Admin users can disable accounts through the CLI command
- [ ] Admin users can view login history from the database
`

	svc, featureID, _ := setupDecomposeTest(t, specContent)

	result, err := svc.SliceAnalysis(SliceAnalysisInput{FeatureID: featureID})
	if err != nil {
		t.Fatalf("SliceAnalysis() error = %v", err)
	}

	if result.FeatureID != featureID {
		t.Errorf("FeatureID = %q, want %q", result.FeatureID, featureID)
	}

	// §16.5: identifies at least one slice for a multi-criterion spec
	if result.TotalSlices < 1 {
		t.Fatalf("TotalSlices = %d, want at least 1", result.TotalSlices)
	}

	// §16.5: each slice includes name, outcomes, layers, estimate, rationale
	for i, s := range result.Slices {
		if s.Name == "" {
			t.Errorf("slice[%d].Name is empty", i)
		}
		if s.Estimate == "" {
			t.Errorf("slice[%d] %q: Estimate is empty", i, s.Name)
		}
		if s.Estimate != "small" && s.Estimate != "medium" && s.Estimate != "large" {
			t.Errorf("slice[%d] %q: Estimate = %q, want small|medium|large", i, s.Name, s.Estimate)
		}
		if s.Rationale == "" {
			t.Errorf("slice[%d] %q: Rationale is empty", i, s.Name)
		}
	}

	// Check that we got the expected slices
	sliceByName := make(map[string]AnalysisSlice)
	for _, s := range result.Slices {
		sliceByName[s.Name] = s
	}

	auth, ok := sliceByName["Authentication"]
	if !ok {
		t.Fatal("expected slice named 'Authentication'")
	}
	if len(auth.Outcomes) < 2 {
		t.Errorf("Authentication outcomes = %d, want at least 2", len(auth.Outcomes))
	}

	admin, ok := sliceByName["Admin Dashboard"]
	if !ok {
		t.Fatal("expected slice named 'Admin Dashboard'")
	}
	// Admin Dashboard mentions database, API, CLI → multiple layers
	if len(admin.Layers) < 2 {
		t.Errorf("Admin Dashboard layers = %v, want at least 2", admin.Layers)
	}
}

func TestSliceAnalysis_InterSliceDependency(t *testing.T) {
	t.Parallel()

	// The "Notifications" section references "Authentication" by name,
	// so slice analysis should detect a dependency.
	specContent := `# Messaging Feature

## Authentication

### Acceptance Criteria

- [ ] Users can authenticate with the API using JWT tokens
- [ ] Tokens are validated on every request handler

## Notifications

### Acceptance Criteria

- [ ] The system sends email notifications after Authentication is complete
- [ ] Notification preferences are stored in the database
`

	svc, featureID, _ := setupDecomposeTest(t, specContent)

	result, err := svc.SliceAnalysis(SliceAnalysisInput{FeatureID: featureID})
	if err != nil {
		t.Fatalf("SliceAnalysis() error = %v", err)
	}

	// §16.5: inter-slice dependencies are identified
	sliceByName := make(map[string]AnalysisSlice)
	for _, s := range result.Slices {
		sliceByName[s.Name] = s
	}

	notif, ok := sliceByName["Notifications"]
	if !ok {
		t.Fatal("expected slice named 'Notifications'")
	}

	foundDep := false
	for _, dep := range notif.DependsOn {
		if dep == "Authentication" {
			foundDep = true
			break
		}
	}
	if !foundDep {
		t.Errorf("Notifications.DependsOn = %v, want it to contain 'Authentication'", notif.DependsOn)
	}
}

func TestSliceAnalysis_EmptyFeatureID(t *testing.T) {
	t.Parallel()
	svc, _, _ := setupDecomposeTest(t, "")

	_, err := svc.SliceAnalysis(SliceAnalysisInput{FeatureID: ""})
	if err == nil {
		t.Fatal("expected error for empty feature_id, got nil")
	}
}

func TestDecomposeFeature_SliceDetailsPopulated(t *testing.T) {
	t.Parallel()

	specContent := `# Feature Spec

## Storage Layer

### Acceptance Criteria

- [ ] Data is persisted in a database table
- [ ] Records can be queried by ID

## API Layer

### Acceptance Criteria

- [ ] Endpoint accepts POST requests to create records
- [ ] Endpoint returns JSON responses from the handler
`

	svc, featureID, _ := setupDecomposeTest(t, specContent)

	result, err := svc.DecomposeFeature(DecomposeInput{FeatureID: featureID})
	if err != nil {
		t.Fatalf("DecomposeFeature() error = %v", err)
	}

	// §F.5/F.8: decompose_feature includes slice details in the proposal
	if len(result.Proposal.SliceDetails) == 0 {
		t.Error("expected non-empty SliceDetails in proposal")
	}

	for i, s := range result.Proposal.SliceDetails {
		if s.Name == "" {
			t.Errorf("SliceDetails[%d].Name is empty", i)
		}
		if s.Estimate == "" {
			t.Errorf("SliceDetails[%d].Estimate is empty", i)
		}
		if s.Rationale == "" {
			t.Errorf("SliceDetails[%d].Rationale is empty", i)
		}
	}
}

// contains reports whether s contains substr (case-insensitive-ish helper).
func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// writeDecomposeTestPlan creates a Plan entity directly on disk for decompose tests,
// bypassing the prefix registry and ID allocator.
func writeDecomposeTestPlan(t *testing.T, svc *EntityService, id string) {
	t.Helper()
	_, _, slug := model.ParsePlanID(id)
	fields := map[string]any{
		"id":         id,
		"slug":       slug,
		"name":       "Test Plan",
		"status":     "active",
		"summary":    "Test plan for decompose tests",
		"created":    "2026-03-19T12:00:00Z",
		"created_by": "test",
		"updated":    "2026-03-19T12:00:00Z",
	}
	_, err := svc.store.Write(storage.EntityRecord{
		Type:   string(model.EntityKindPlan),
		ID:     id,
		Slug:   slug,
		Fields: fields,
	})
	if err != nil {
		t.Fatalf("writeDecomposeTestPlan(%s) error = %v", id, err)
	}
}

// ─── AC-06: decompose propose on bold-AC spec ─────────────────────────────────

// AC-06: When decompose propose is called on a specification file that uses
// exclusively the **AC-NN.** bold-identifier format in its acceptance criteria
// section, the generated task summaries are derived from the criterion text,
// not from section headings.
//
// This is the integration test for FEAT-01KN4ZPCMJ1FP (docint-ac-pattern-recognition).
// It verifies that parseSpecStructure correctly extracts bold-identifier criteria
// and that generateProposal uses them to produce criterion-derived summaries.
func TestDecomposeFeature_BoldACSpec_ProducesCriterionDerivedSummaries(t *testing.T) {
	t.Parallel()

	// A spec that uses exclusively the **AC-NN.** bold-identifier format.
	// No checkbox or numbered-list criteria are present.
	specContent := `# Feature Specification: Bold AC Test

## 1. Purpose

This feature tests the bold-identifier extraction path in decompose propose.

## 2. Acceptance Criteria

**AC-01.** The handler MUST validate input before processing the request.

**AC-02.** The handler MUST return a structured error when validation fails.

**AC-03.** The response MUST include the request identifier in all cases.

**AC-04.** The system MUST log all errors at WARNING level or above.

**AC-05.** The handler MUST handle concurrent requests without data races.
`

	svc, featureID, _ := setupDecomposeTest(t, specContent)

	result, err := svc.DecomposeFeature(DecomposeInput{FeatureID: featureID})
	if err != nil {
		t.Fatalf("DecomposeFeature() on bold-AC spec returned error: %v", err)
	}

	proposal := result.Proposal

	// REQ-13 / AC-06: The proposal must contain tasks derived from the 5 AC lines.
	// 5 ACs in one section → individual tasks (5+ threshold). Plus a test task.
	if proposal.TotalTasks < 5 {
		t.Errorf("TotalTasks = %d, want at least 5 (one per bold-AC criterion)", proposal.TotalTasks)
	}

	// Collect all task summaries for assertion.
	summaries := make([]string, len(proposal.Tasks))
	for i, task := range proposal.Tasks {
		summaries[i] = task.Summary
	}

	// AC-06 assertion 1: No task summary matches the pattern "Implement <section heading>"
	// where the section heading is from the spec document.
	sectionHeadings := []string{
		"Purpose",
		"Acceptance Criteria",
		"1. Purpose",
		"2. Acceptance Criteria",
	}
	for _, summary := range summaries {
		for _, heading := range sectionHeadings {
			if summary == "Implement "+heading || summary == heading {
				t.Errorf("task summary %q looks like a section-heading fallback, not a criterion-derived summary", summary)
			}
		}
	}

	// AC-06 assertion 2: At least one task summary contains text from a bold-AC criterion.
	// The extracted criterion format is "AC-NN: <text>", so summaries should contain "AC-0".
	criterionDerivedCount := 0
	for _, summary := range summaries {
		// Check for criterion-derived content: contains the identifier prefix "AC-"
		// or text from one of the criterion bodies.
		if contains(summary, "AC-0") ||
			contains(summary, "validate input") ||
			contains(summary, "structured error") ||
			contains(summary, "request identifier") ||
			contains(summary, "log all errors") {
			criterionDerivedCount++
		}
	}
	if criterionDerivedCount == 0 {
		t.Errorf("no task summaries contain criterion-derived text; summaries: %v", summaries)
	}

	// REQ-13 assertion: the set of tasks covers the AC criteria, not just section names.
	// Verify the feature ID and spec document are correctly linked in the result.
	if result.FeatureID != featureID {
		t.Errorf("result.FeatureID = %q, want %q", result.FeatureID, featureID)
	}
	if result.SpecDocumentID == "" {
		t.Error("result.SpecDocumentID is empty; spec document must be linked")
	}
}

// TestDecomposeFeature_BoldACSpec_ZeroFallbackToCheckbox verifies that a spec
// using bold-identifier format produces tasks without requiring checkbox format.
// This is a regression guard: before the fix, such specs returned an error.
func TestDecomposeFeature_BoldACSpec_NoLongerReturnsError(t *testing.T) {
	t.Parallel()

	specContent := `# Spec With Only Bold ACs

## Acceptance Criteria

**REQ-01.** The service MUST accept JSON input.

**REQ-02.** The service MUST reject malformed payloads with HTTP 400.
`

	svc, featureID, _ := setupDecomposeTest(t, specContent)

	_, err := svc.DecomposeFeature(DecomposeInput{FeatureID: featureID})
	if err != nil {
		t.Errorf("DecomposeFeature() returned error for bold-AC spec (should succeed after fix): %v", err)
	}
}

// TestParseSpecStructure_BoldIdent_InACSection verifies that parseSpecStructure
// correctly extracts bold-identifier criteria from an acceptance criteria section.
func TestParseSpecStructure_BoldIdent_InACSection(t *testing.T) {
	t.Parallel()

	content := `# Test Spec

## Acceptance Criteria

**AC-01.** The system must do X.
**AC-02.** The system must do Y.
**C-03.** No side effects allowed.
`

	spec := parseSpecStructure(content)

	if len(spec.acceptanceCriteria) != 3 {
		t.Fatalf("len(acceptanceCriteria) = %d, want 3; criteria: %v",
			len(spec.acceptanceCriteria),
			spec.acceptanceCriteria)
	}
	if spec.acceptanceCriteria[0].text != "AC-01: The system must do X." {
		t.Errorf("criterion[0].text = %q, want %q",
			spec.acceptanceCriteria[0].text, "AC-01: The system must do X.")
	}
	if spec.acceptanceCriteria[1].text != "AC-02: The system must do Y." {
		t.Errorf("criterion[1].text = %q, want %q",
			spec.acceptanceCriteria[1].text, "AC-02: The system must do Y.")
	}
	if spec.acceptanceCriteria[2].text != "C-03: No side effects allowed." {
		t.Errorf("criterion[2].text = %q, want %q",
			spec.acceptanceCriteria[2].text, "C-03: No side effects allowed.")
	}
}

// TestParseSpecStructure_BoldIdent_OutsideACSection verifies that
// bold-identifier lines outside acceptance criteria sections are NOT extracted.
func TestParseSpecStructure_BoldIdent_OutsideACSection(t *testing.T) {
	t.Parallel()

	content := `# Test Spec

## Background

**AC-01.** This appears in a non-AC section and should NOT be extracted.

## Purpose

Some prose about the purpose.
`

	spec := parseSpecStructure(content)

	if len(spec.acceptanceCriteria) != 0 {
		t.Errorf("len(acceptanceCriteria) = %d, want 0 (bold-ident outside AC section must not be extracted); got: %v",
			len(spec.acceptanceCriteria), spec.acceptanceCriteria)
	}
}

// TestParseSpecStructure_BoldIdent_MixedWithCheckbox verifies that bold-identifier
// and checkbox criteria can coexist in the same document.
func TestParseSpecStructure_BoldIdent_MixedWithCheckbox(t *testing.T) {
	t.Parallel()

	content := `# Test Spec

## Acceptance Criteria

- [ ] Checkbox criterion one.
- [ ] Checkbox criterion two.
**AC-01.** Bold-identifier criterion three.
`

	spec := parseSpecStructure(content)

	// All three criteria should be extracted.
	if len(spec.acceptanceCriteria) != 3 {
		t.Fatalf("len(acceptanceCriteria) = %d, want 3; criteria: %v",
			len(spec.acceptanceCriteria), spec.acceptanceCriteria)
	}
}

// ---------------------------------------------------------------------------
// Integration tests (Task 5): full ReviewProposal path with new checks
// ---------------------------------------------------------------------------

func TestReviewProposal_EmptyDescription_Fail(t *testing.T) {
	t.Parallel()

	specContent := `# Spec
- [ ] Feature is implemented
`
	svc, featureID, _ := setupDecomposeTest(t, specContent)

	proposal := Proposal{
		Tasks: []ProposedTask{
			{
				Slug:      "main-task",
				Summary:   "",
				Rationale: "Covers: feature is implemented. Testing: verified by coverage.",
			},
		},
		TotalTasks: 1,
	}

	result, err := svc.ReviewProposal(DecomposeReviewInput{
		FeatureID: featureID,
		Proposal:  proposal,
	})
	if err != nil {
		t.Fatalf("ReviewProposal() error = %v", err)
	}

	if result.Status != "fail" {
		t.Errorf("Status = %q, want fail (empty-description is blocking)", result.Status)
	}
	if result.BlockingCount < 1 {
		t.Errorf("BlockingCount = %d, want >= 1", result.BlockingCount)
	}

	emptyDescCount := 0
	for _, f := range result.Findings {
		if f.Type == "empty-description" {
			emptyDescCount++
			if f.Severity != "error" {
				t.Errorf("empty-description Severity = %q, want error", f.Severity)
			}
		}
	}
	if emptyDescCount == 0 {
		t.Error("expected at least one empty-description finding, got none")
	}
}

func TestReviewProposal_WarningsOnly_Warn(t *testing.T) {
	t.Parallel()

	specContent := `# Spec
- [ ] Feature is implemented
`
	svc, featureID, _ := setupDecomposeTest(t, specContent)

	bigEstimate := 10.0
	proposal := Proposal{
		Tasks: []ProposedTask{
			{
				Slug:      "main-task",
				Summary:   "Implement the main feature that is implemented correctly",
				Estimate:  &bigEstimate,
				Rationale: "Covers: feature is implemented",
			},
		},
		TotalTasks: 1,
	}

	result, err := svc.ReviewProposal(DecomposeReviewInput{
		FeatureID: featureID,
		Proposal:  proposal,
	})
	if err != nil {
		t.Fatalf("ReviewProposal() error = %v", err)
	}

	if result.Status != "warn" {
		t.Errorf("Status = %q, want warn (oversized+missing-test-coverage are warnings only)", result.Status)
	}
	if result.BlockingCount != 0 {
		t.Errorf("BlockingCount = %d, want 0", result.BlockingCount)
	}
	if result.TotalFindings == 0 {
		t.Error("TotalFindings = 0, want > 0")
	}
}

func TestReviewProposal_MixedOldAndNew(t *testing.T) {
	t.Parallel()

	specContent := `# Spec
- [ ] Alpha feature is working
- [ ] Beta feature is working
- [ ] Gamma service is available
`
	svc, featureID, _ := setupDecomposeTest(t, specContent)

	// alpha and beta covered; gamma is NOT — creates a gap (error).
	// delta has no deps while alpha->beta chain exists — creates an orphan (warning).
	proposal := Proposal{
		Tasks: []ProposedTask{
			{
				Slug:      "alpha",
				Summary:   "Implement alpha feature that is working",
				DependsOn: []string{"beta"},
				Rationale: "Covers alpha. Test coverage verified.",
			},
			{
				Slug:      "beta",
				Summary:   "Implement beta feature that is working",
				Rationale: "Covers beta feature.",
			},
			{
				Slug:      "delta",
				Summary:   "Implement delta standalone component",
				Rationale: "Extra delta task.",
			},
		},
		TotalTasks: 3,
	}

	result, err := svc.ReviewProposal(DecomposeReviewInput{
		FeatureID: featureID,
		Proposal:  proposal,
	})
	if err != nil {
		t.Fatalf("ReviewProposal() error = %v", err)
	}

	if result.Status != "fail" {
		t.Errorf("Status = %q, want fail (gap is blocking)", result.Status)
	}

	var hasGap, hasOrphan bool
	for _, f := range result.Findings {
		if f.Type == "gap" {
			hasGap = true
		}
		if f.Type == "orphan-task" {
			hasOrphan = true
		}
	}
	if !hasGap {
		t.Error("expected a gap finding, got none")
	}
	if !hasOrphan {
		t.Error("expected an orphan-task finding, got none")
	}
}

func TestReviewProposal_AllChecksClear_Pass(t *testing.T) {
	t.Parallel()

	specContent := `# Spec
- [ ] Users can log in
- [ ] Users can log out
`
	svc, featureID, _ := setupDecomposeTest(t, specContent)

	proposal := Proposal{
		Tasks: []ProposedTask{
			{
				Slug:      "login",
				Summary:   "Implement user authentication for login",
				Rationale: "Covers: users can log in. Verified by integration testing.",
			},
			{
				Slug:      "logout",
				Summary:   "Implement session cleanup for logout",
				Rationale: "Covers: users can log out.",
			},
		},
		TotalTasks: 2,
	}

	result, err := svc.ReviewProposal(DecomposeReviewInput{
		FeatureID: featureID,
		Proposal:  proposal,
	})
	if err != nil {
		t.Fatalf("ReviewProposal() error = %v", err)
	}

	if result.Status != "pass" {
		t.Errorf("Status = %q, want pass; findings: %v", result.Status, result.Findings)
	}
	if result.TotalFindings != 0 {
		t.Errorf("TotalFindings = %d, want 0; findings: %v", result.TotalFindings, result.Findings)
	}
}

func TestReviewProposal_ErrorAndWarningCombined(t *testing.T) {
	t.Parallel()

	specContent := `# Spec
- [ ] Feature is implemented
`
	svc, featureID, _ := setupDecomposeTest(t, specContent)

	// Empty summary triggers empty-description (error).
	// No testing keyword triggers missing-test-coverage (warning).
	proposal := Proposal{
		Tasks: []ProposedTask{
			{
				Slug:      "main-task",
				Summary:   "",
				Rationale: "Covers: feature is implemented",
			},
		},
		TotalTasks: 1,
	}

	result, err := svc.ReviewProposal(DecomposeReviewInput{
		FeatureID: featureID,
		Proposal:  proposal,
	})
	if err != nil {
		t.Fatalf("ReviewProposal() error = %v", err)
	}

	if result.Status != "fail" {
		t.Errorf("Status = %q, want fail", result.Status)
	}

	// BlockingCount must equal the number of error-severity findings only.
	errorCount := 0
	warningCount := 0
	for _, f := range result.Findings {
		switch f.Severity {
		case "error":
			errorCount++
		case "warning":
			warningCount++
		default:
			t.Errorf("finding %q has unexpected Severity %q", f.Type, f.Severity)
		}
		if f.Severity == "" {
			t.Errorf("finding %q has empty Severity", f.Type)
		}
	}
	if result.BlockingCount != errorCount {
		t.Errorf("BlockingCount = %d, want %d (count of error findings)", result.BlockingCount, errorCount)
	}
	if result.TotalFindings != len(result.Findings) {
		t.Errorf("TotalFindings = %d, want %d", result.TotalFindings, len(result.Findings))
	}
	if warningCount == 0 {
		t.Error("expected at least one warning finding (missing-test-coverage), got none")
	}
}

// ---------------------------------------------------------------------------
// P13 Feature 5: Decomposition Grouping tests
// ---------------------------------------------------------------------------

// TestGrouping_Thresholds verifies the section-based grouping thresholds:
//   - 1 AC  → 1 individual task, Covers has 1 element
//   - 2–4 ACs → 1 grouped task, Covers has n elements
//   - 5+ ACs → individual tasks, each with 1-element Covers
func TestGrouping_Thresholds(t *testing.T) {
	t.Parallel()

	const featureSlug = "feat"
	const section = "Auth"

	makeSpec := func(n int) specStructure {
		var acs []acceptanceCriterion
		for i := 0; i < n; i++ {
			acs = append(acs, acceptanceCriterion{
				text:     fmt.Sprintf("criterion %d works correctly", i+1),
				section:  section,
				parentL2: section,
			})
		}
		return specStructure{acceptanceCriteria: acs}
	}

	cases := []struct {
		n           int
		wantACTasks int
		wantGrouped bool
	}{
		{1, 1, false},
		{2, 1, true},
		{3, 1, true},
		{4, 1, true},
		{5, 5, false},
		{6, 6, false},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(fmt.Sprintf("n=%d", tc.n), func(t *testing.T) {
			t.Parallel()
			spec := makeSpec(tc.n)
			proposal, guidance := generateProposal(spec, featureSlug, "", 0)

			// Count AC tasks (exclude the auto-added test-companion task).
			var acTasks []ProposedTask
			for _, task := range proposal.Tasks {
				if task.Slug != featureSlug+"-tests" {
					acTasks = append(acTasks, task)
				}
			}

			if len(acTasks) != tc.wantACTasks {
				t.Errorf("n=%d: AC task count = %d, want %d", tc.n, len(acTasks), tc.wantACTasks)
			}

			hasGroupBy := false
			hasOneAC := false
			for _, g := range guidance {
				switch g {
				case "group-by-section":
					hasGroupBy = true
				case "one-ac-per-task":
					hasOneAC = true
				}
			}

			if tc.wantGrouped {
				if !hasGroupBy {
					t.Errorf("n=%d: expected 'group-by-section' in guidance %v", tc.n, guidance)
				}
				if hasOneAC {
					t.Errorf("n=%d: did not expect 'one-ac-per-task' in guidance when grouped: %v", tc.n, guidance)
				}
				// The single grouped task's Covers should have n elements.
				if len(acTasks) > 0 && len(acTasks[0].Covers) != tc.n {
					t.Errorf("n=%d: grouped task Covers length = %d, want %d", tc.n, len(acTasks[0].Covers), tc.n)
				}
			} else {
				if !hasOneAC {
					t.Errorf("n=%d: expected 'one-ac-per-task' in guidance %v", tc.n, guidance)
				}
				// Each individual task should have exactly 1 element in Covers.
				for i, task := range acTasks {
					if len(task.Covers) != 1 {
						t.Errorf("n=%d: task[%d] Covers length = %d, want 1", tc.n, i, len(task.Covers))
					}
				}
			}
		})
	}
}

// TestGrouping_MixedSections verifies that sections are grouped independently:
// a section with 3 ACs produces 1 grouped task; a section with 7 ACs produces
// 7 individual tasks.
func TestGrouping_MixedSections(t *testing.T) {
	t.Parallel()

	const featureSlug = "feat"

	var acs []acceptanceCriterion
	// Section A: 3 ACs → grouped (2–4 range).
	for i := 0; i < 3; i++ {
		acs = append(acs, acceptanceCriterion{
			text:     fmt.Sprintf("section-a criterion %d", i+1),
			section:  "Section A",
			parentL2: "Section A",
		})
	}
	// Section B: 7 ACs → individual (5+ range).
	for i := 0; i < 7; i++ {
		acs = append(acs, acceptanceCriterion{
			text:     fmt.Sprintf("section-b criterion %d", i+1),
			section:  "Section B",
			parentL2: "Section B",
		})
	}
	spec := specStructure{acceptanceCriteria: acs}

	proposal, guidance := generateProposal(spec, featureSlug, "", 0)

	// Expect: 1 grouped (Section A) + 7 individual (Section B) = 8 AC tasks + test task.
	acTaskCount := 0
	for _, task := range proposal.Tasks {
		if task.Slug != featureSlug+"-tests" {
			acTaskCount++
		}
	}
	if acTaskCount != 8 {
		t.Errorf("AC task count = %d, want 8 (1 grouped + 7 individual)", acTaskCount)
	}

	// Guidance must contain "group-by-section" (Section A triggers it).
	hasGroupBy := false
	for _, g := range guidance {
		if g == "group-by-section" {
			hasGroupBy = true
		}
	}
	if !hasGroupBy {
		t.Errorf("expected 'group-by-section' in guidance %v", guidance)
	}

	// Find the Section A grouped task (slug = featureSlug + "-section-a").
	var sectionATask *ProposedTask
	for i := range proposal.Tasks {
		if proposal.Tasks[i].Slug == featureSlug+"-section-a" {
			sectionATask = &proposal.Tasks[i]
			break
		}
	}
	if sectionATask == nil {
		t.Fatalf("expected task with slug %q; tasks: %v", featureSlug+"-section-a", proposal.Tasks)
	}
	if len(sectionATask.Covers) != 3 {
		t.Errorf("Section A grouped task Covers length = %d, want 3", len(sectionATask.Covers))
	}

	// AC-13: grouped task summary format.
	wantSummary := "Implement Section A (3 criteria)"
	if sectionATask.Summary != wantSummary {
		t.Errorf("Section A grouped task Summary = %q, want %q", sectionATask.Summary, wantSummary)
	}

	// AC-14: grouped task rationale lists all AC texts.
	for _, acText := range []string{"section-a criterion 1", "section-a criterion 2", "section-a criterion 3"} {
		if !strings.Contains(sectionATask.Rationale, acText) {
			t.Errorf("Section A grouped task Rationale missing %q; Rationale: %q", acText, sectionATask.Rationale)
		}
	}

	// Section B tasks should each have exactly 1 Covers entry.
	for _, task := range proposal.Tasks {
		if task.Slug == featureSlug+"-tests" || task.Slug == featureSlug+"-section-a" {
			continue
		}
		if len(task.Covers) != 1 {
			t.Errorf("Section B task %q Covers length = %d, want 1", task.Slug, len(task.Covers))
		}
	}
}

// TestGrouping_TestCompanionHasNoCovers verifies that the automatically added
// test-companion task has nil/empty Covers.
func TestGrouping_TestCompanionHasNoCovers(t *testing.T) {
	t.Parallel()

	spec := specStructure{
		acceptanceCriteria: []acceptanceCriterion{
			{text: "feature works correctly", section: "S", parentL2: "S"},
		},
	}
	proposal, _ := generateProposal(spec, "feat", "", 0)

	for _, task := range proposal.Tasks {
		if task.Slug == "feat-tests" {
			if len(task.Covers) != 0 {
				t.Errorf("test-companion task Covers = %v, want nil/empty", task.Covers)
			}
			return
		}
	}
	t.Error("test-companion task 'feat-tests' not found in proposal")
}

// TestProposedTask_CoversOmittedFromJSON verifies that a nil Covers slice is
// omitted from JSON encoding (json:"covers,omitempty"), and that a non-empty
// Covers slice is included.
func TestProposedTask_CoversOmittedFromJSON(t *testing.T) {
	t.Parallel()

	// Nil Covers — must NOT appear in JSON output.
	task := ProposedTask{
		Slug:      "my-task",
		Summary:   "Do something",
		Rationale: "Because",
	}
	data, err := json.Marshal(task)
	if err != nil {
		t.Fatalf("json.Marshal error = %v", err)
	}
	if contains(string(data), `"covers"`) {
		t.Errorf("JSON contains 'covers' key for task with nil Covers; got: %s", data)
	}

	// Non-empty Covers — must appear in JSON output.
	task.Covers = []string{"criterion one"}
	data, err = json.Marshal(task)
	if err != nil {
		t.Fatalf("json.Marshal error = %v", err)
	}
	if !contains(string(data), `"covers"`) {
		t.Errorf("JSON missing 'covers' key for task with non-empty Covers; got: %s", data)
	}
}

// TestCheckGaps_ExactMatchViaCovers verifies the updated isACCovered logic:
//  1. Task with Covers containing the AC text → exact match, covered.
//  2. Task with Covers NOT containing the AC text → NOT covered, even if
//     keywords in Summary/Rationale overlap (no heuristic fallback for that task).
//  3. Task with nil Covers → keyword overlap heuristic applies.
func TestCheckGaps_ExactMatchViaCovers(t *testing.T) {
	t.Parallel()

	acText := "validate that the authentication token is refreshed before expiry"
	ac := acceptanceCriterion{text: acText, section: "S", parentL2: "S"}

	// Task with exact Covers match → must cover the AC.
	taskExact := ProposedTask{
		Slug:      "exact",
		Summary:   "unrelated summary",
		Rationale: "unrelated rationale",
		Covers:    []string{acText},
	}
	if !isACCovered(ac, []ProposedTask{taskExact}) {
		t.Error("task with exact Covers match should cover the AC")
	}

	// Task with Covers NOT containing the AC, but whose Summary/Rationale carry
	// enough keywords that the old heuristic would have matched. The new code
	// must NOT fall back to keyword overlap for this task.
	taskWrongCovers := ProposedTask{
		Slug:      "wrong-covers",
		Summary:   "validate authentication token refresh expiry",
		Rationale: "authentication token refresh expiry before validation",
		Covers:    []string{"some other unrelated criterion"},
	}
	if isACCovered(ac, []ProposedTask{taskWrongCovers}) {
		t.Error("task whose Covers doesn't contain the AC should NOT cover it, even when keywords overlap")
	}

	// Task with nil Covers → keyword heuristic applies.
	// Summary and Rationale contain enough AC keywords to exceed the 2/3 threshold.
	taskKeyword := ProposedTask{
		Slug:      "keyword",
		Summary:   "validate authentication token refresh expiry logic",
		Rationale: "authentication token refreshed before expiry deadline",
	}
	if !isACCovered(ac, []ProposedTask{taskKeyword}) {
		t.Error("task without Covers but with sufficient keyword overlap should cover the AC")
	}
}

// TestParseSpecStructure_TableRows verifies that markdown tables within
// acceptance-criteria sections are extracted as acceptance criteria, with
// cells joined by " — ".
func TestParseSpecStructure_TableRows(t *testing.T) {
	t.Parallel()

	content := `# Spec

## Acceptance Criteria

| Criterion | Description |
| --- | --- |
| CR-1 | User can log in |
| CR-2 | User can log out |
| CR-3 | Session expires after timeout |
`

	spec := parseSpecStructure(content)

	if len(spec.acceptanceCriteria) != 3 {
		t.Fatalf("acceptance criteria count = %d, want 3 (3 data rows); got: %v",
			len(spec.acceptanceCriteria), spec.acceptanceCriteria)
	}

	want := []string{
		"CR-1 — User can log in",
		"CR-2 — User can log out",
		"CR-3 — Session expires after timeout",
	}
	for i, ac := range spec.acceptanceCriteria {
		if ac.text != want[i] {
			t.Errorf("ac[%d].text = %q, want %q", i, ac.text, want[i])
		}
		if ac.section != "Acceptance Criteria" {
			t.Errorf("ac[%d].section = %q, want %q", i, ac.section, "Acceptance Criteria")
		}
	}
}

// TestParseSpecStructure_TableNotInACSection verifies that tables outside
// acceptance-criteria sections are NOT parsed as criteria.
func TestParseSpecStructure_TableNotInACSection(t *testing.T) {
	t.Parallel()

	content := `# Spec

## Background

| Column A | Column B |
| --- | --- |
| Row 1A | Row 1B |
| Row 2A | Row 2B |

## Acceptance Criteria

- [ ] Single checkbox criterion
`

	spec := parseSpecStructure(content)

	// Only the checkbox criterion should be extracted; Background table rows are ignored.
	if len(spec.acceptanceCriteria) != 1 {
		t.Fatalf("acceptance criteria count = %d, want 1; got: %v",
			len(spec.acceptanceCriteria), spec.acceptanceCriteria)
	}
	if spec.acceptanceCriteria[0].text != "Single checkbox criterion" {
		t.Errorf("ac[0].text = %q, want %q",
			spec.acceptanceCriteria[0].text, "Single checkbox criterion")
	}
}

// TestReviewProposal_BackwardCompatibility verifies that a Proposal deserialized
// from JSON without a "covers" key (legacy format) has nil Covers on its tasks,
// and that ReviewProposal falls back to keyword-overlap gap detection correctly.
func TestReviewProposal_BackwardCompatibility(t *testing.T) {
	t.Parallel()

	// Simulate a legacy JSON proposal that predates the Covers field.
	legacyJSON := `{
		"tasks": [
			{
				"slug": "login",
				"name": "",
				"summary": "Implement user login with email and password",
				"rationale": "Covers: users can log in"
			}
		],
		"total_tasks": 1,
		"slices": [],
		"warnings": []
	}`

	var proposal Proposal
	if err := json.Unmarshal([]byte(legacyJSON), &proposal); err != nil {
		t.Fatalf("json.Unmarshal error = %v", err)
	}

	// Covers must be nil when the field is absent from JSON.
	if proposal.Tasks[0].Covers != nil {
		t.Errorf("Covers = %v, want nil for legacy JSON without 'covers' key",
			proposal.Tasks[0].Covers)
	}

	// ReviewProposal must not report a gap for an AC that is keyword-covered.
	specContent := `# Spec
- [ ] Users can log in
`
	svc, featureID, _ := setupDecomposeTest(t, specContent)

	result, err := svc.ReviewProposal(DecomposeReviewInput{
		FeatureID: featureID,
		Proposal:  proposal,
	})
	if err != nil {
		t.Fatalf("ReviewProposal() error = %v", err)
	}

	for _, f := range result.Findings {
		if f.Type == "gap" {
			t.Errorf("unexpected gap finding for legacy proposal with keyword-matching rationale: %v", f)
		}
	}
}

func TestParseSpecStructure_ListItemBoldIdent(t *testing.T) {
	// Before fix: reBoldIdent requires line to start with "**"; a line starting
	// with "- " never matches, so acceptanceCriteria is empty. This test FAILS
	// (0 criteria) before the fix and PASSES (2 criteria) after.
	content := `# Spec

## Acceptance Criteria

- **AC-01.** The system must reject requests without auth tokens.
- **AC-02.** The system must return HTTP 401 in that case.
`
	spec := parseSpecStructure(content)
	if len(spec.acceptanceCriteria) != 2 {
		t.Fatalf("acceptanceCriteria len = %d, want 2; got: %v",
			len(spec.acceptanceCriteria), spec.acceptanceCriteria)
	}
	want0 := "AC-01: The system must reject requests without auth tokens."
	if spec.acceptanceCriteria[0].text != want0 {
		t.Errorf("criteria[0] = %q, want %q", spec.acceptanceCriteria[0].text, want0)
	}
	want1 := "AC-02: The system must return HTTP 401 in that case."
	if spec.acceptanceCriteria[1].text != want1 {
		t.Errorf("criteria[1] = %q, want %q", spec.acceptanceCriteria[1].text, want1)
	}
}

func TestDecomposeFeature_RichDiagnostic_BoldOutsideSection(t *testing.T) {
	t.Parallel()
	// Spec has bold-ident lines but they are OUTSIDE an AC section.
	// Before fix: error is generic, does not mention bold-idents outside AC section.
	// After fix: error mentions "outside an Acceptance Criteria section".
	specContent := `# Feature Spec

## Design Details

**AC-01.** The system MUST handle requests.
**AC-02.** The system MUST respond within 200ms.
`
	svc, featureID, _ := setupDecomposeTest(t, specContent)
	_, err := svc.DecomposeFeature(DecomposeInput{FeatureID: featureID})
	if err == nil {
		t.Fatal("expected error when no AC criteria parsed, got nil")
	}
	if !contains(err.Error(), "outside an Acceptance Criteria section") {
		t.Errorf("error = %q\nwant it to mention bold-idents outside AC section", err.Error())
	}
}

// ---------------------------------------------------------------------------
// AC-01 through AC-05: deriveTaskName and decompose apply tests
// ---------------------------------------------------------------------------

// TestDeriveTaskName_BoldIdentPrefix covers AC-02, AC-04, and AC-05.
// It exercises deriveTaskName directly (same package) with a table of inputs.
func TestDeriveTaskName_BoldIdentPrefix(t *testing.T) {
	t.Parallel()

	longText := strings.Repeat("x", 70) // 70 chars — exceeds 60 char limit

	tests := []struct {
		name     string
		text     string
		fallback string
		// wantExact: if non-empty, output must equal this value exactly.
		wantExact string
		// wantMaxLen: if > 0, output must be <= this length.
		wantMaxLen int
		// wantNoColon: if true, output must not contain ":".
		wantNoColon bool
		// wantValidateName: if true, output must pass validate.ValidateName.
		wantValidateName bool
		// wantMatchPattern: if non-empty, output must match this regexp.
		wantMatchPattern string
	}{
		{
			// AC-02 case 1: bold-ident prefix is stripped; result has no colon and passes ValidateName.
			name:             "bold-ident prefix stripped",
			text:             "AC-01: The service MUST accept JSON input",
			fallback:         "fallback",
			wantNoColon:      true,
			wantValidateName: true,
		},
		{
			// AC-02 / AC-05: plain prose with colon — not a bold-ident prefix → not stripped.
			name:      "plain prose with colon not stripped",
			text:      "Login: users can authenticate",
			fallback:  "fallback",
			wantExact: "Login: users can authenticate",
		},
		{
			// AC-02 case 3: empty input → fallback returned.
			name:      "empty text returns fallback",
			text:      "",
			fallback:  "fallback",
			wantExact: "fallback",
		},
		{
			// AC-02 case 4: text longer than 60 chars → output truncated to ≤60 chars.
			name:       "long text truncated to 60 chars",
			text:       longText,
			fallback:   "fallback",
			wantMaxLen: 60,
		},
		{
			// AC-04: empty AC text with fallback matching the Implement AC-NNN pattern.
			name:             "empty text uses Implement AC fallback pattern",
			text:             "",
			fallback:         "Implement AC-001",
			wantMatchPattern: `^Implement AC-\d{3}$`,
		},
		{
			// AC-05: prose with colon like "Setup: configure the database" is returned verbatim.
			name:      "Setup colon prose not stripped",
			text:      "Setup: configure the database",
			fallback:  "fallback",
			wantExact: "Setup: configure the database",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := deriveTaskName(tc.text, tc.fallback)

			if tc.wantExact != "" && got != tc.wantExact {
				t.Errorf("deriveTaskName(%q, %q) = %q, want %q", tc.text, tc.fallback, got, tc.wantExact)
			}
			if tc.wantMaxLen > 0 && len(got) > tc.wantMaxLen {
				t.Errorf("deriveTaskName(%q, %q) = %q (len %d), want len <= %d", tc.text, tc.fallback, got, len(got), tc.wantMaxLen)
			}
			if tc.wantNoColon && strings.Contains(got, ":") {
				t.Errorf("deriveTaskName(%q, %q) = %q, output must not contain a colon", tc.text, tc.fallback, got)
			}
			if tc.wantValidateName {
				if _, err := validate.ValidateName(got); err != nil {
					t.Errorf("deriveTaskName(%q, %q) = %q, validate.ValidateName failed: %v", tc.text, tc.fallback, got, err)
				}
			}
			if tc.wantMatchPattern != "" {
				re := regexp.MustCompile(tc.wantMatchPattern)
				if !re.MatchString(got) {
					t.Errorf("deriveTaskName(%q, %q) = %q, want match %q", tc.text, tc.fallback, got, tc.wantMatchPattern)
				}
			}
		})
	}
}

// TestDecomposeApply_SucceedsWithProposedNames covers AC-03.
// It calls DecomposeFeature to get a proposal, then simulates the apply path
// by calling entitySvc.CreateTask for each proposed task.
// All tasks must be created without a "name must not be empty" error.
func TestDecomposeApply_SucceedsWithProposedNames(t *testing.T) {
	t.Parallel()

	// Use a bold-ident spec so deriveTaskName's stripping logic is exercised.
	specContent := `# Feature Spec: Apply Names Test

## Acceptance Criteria

**AC-01.** The handler MUST validate all required fields before persisting.

**AC-02.** The handler MUST return HTTP 400 when validation fails.

**AC-03.** The handler MUST return HTTP 201 on successful creation.

**AC-04.** The service MUST emit an audit event after every write.

**AC-05.** The service MUST roll back on partial failure.
`

	svc, featureID, _ := setupDecomposeTest(t, specContent)

	result, err := svc.DecomposeFeature(DecomposeInput{FeatureID: featureID})
	if err != nil {
		t.Fatalf("DecomposeFeature() error = %v", err)
	}

	proposal := result.Proposal
	if len(proposal.Tasks) == 0 {
		t.Fatal("proposal contains no tasks; cannot test apply path")
	}

	// Simulate the apply path: create each proposed task via entitySvc.CreateTask.
	// This mirrors what decomposeApply does in internal/mcp/decompose_tool.go.
	for _, pt := range proposal.Tasks {
		_, createErr := svc.entitySvc.CreateTask(CreateTaskInput{
			ParentFeature: featureID,
			Slug:          pt.Slug,
			Name:          pt.Name,
			Summary:       pt.Summary,
		})
		if createErr != nil {
			if strings.Contains(createErr.Error(), "name must not be empty") {
				t.Errorf("CreateTask(%q) failed with 'name must not be empty'; Name field from proposal was %q", pt.Slug, pt.Name)
			} else {
				t.Errorf("CreateTask(%q) unexpected error: %v", pt.Slug, createErr)
			}
		}
	}
}

func TestDecomposeFeature_RichDiagnostic_NoBoldIdents(t *testing.T) {
	t.Parallel()
	// Spec has sections but no bold-ident lines and no checkboxes.
	// Before fix: error is generic. After fix: error mentions "no bold-identifier lines".
	specContent := `# Feature Spec

## Overview

This feature does X and Y.

## Design

The design approach is Z.
`
	svc, featureID, _ := setupDecomposeTest(t, specContent)
	_, err := svc.DecomposeFeature(DecomposeInput{FeatureID: featureID})
	if err == nil {
		t.Fatal("expected error when no AC criteria parsed, got nil")
	}
	if !contains(err.Error(), "no bold-identifier lines") {
		t.Errorf("error = %q\nwant it to mention no bold-identifier lines", err.Error())
	}
}

// ---------------------------------------------------------------------------
// Dev-plan-aware decomposition tests (FEAT-01KPQ08YBJ5AK)
// ---------------------------------------------------------------------------

// devPlanWithTwoTasks is a valid dev plan with a two-task breakdown.
// Task 2 depends on Task 1.
const devPlanWithTwoTasks = `# Dev Plan for Test Feature

## Task Breakdown

### Task 1: Implement data model
- **Effort:** Small
- **Spec requirements:** FR-001, FR-002
- **Depends on:** None

### Task 2: Implement API endpoints
- **Effort:** Large
- **Spec requirements:** FR-003
- **Depends on:** Task 1
`

// devPlanNoTaskBreakdown is an approved dev plan that lacks a Task Breakdown section.
const devPlanNoTaskBreakdown = `# Dev Plan for Test Feature

## Overview

This plan describes the work but has no task breakdown section yet.
`

// specWithACs is a minimal spec that has parseable acceptance criteria under L2 sections.
const specWithACs = `# Feature Spec

## Data Layer

### Acceptance Criteria

- [ ] Data is persisted correctly
- [ ] Records can be retrieved by ID

## API Layer

### Acceptance Criteria

- [ ] Endpoint accepts POST requests
`

// specWithNoACs is a spec that has no acceptance criteria whatsoever.
const specWithNoACs = `# Feature Spec

## Overview

This spec describes a feature but has no acceptance criteria.

## Architecture

High-level architecture notes only.
`

// setupDecomposeTestWithDevPlan creates an entity + document service, a feature
// with an approved spec, and a dev plan document owned by the feature.
// approve controls whether the dev plan is in approved or draft status.
// Returns: *DecomposeService, *EntityService, featureID, featureSlug.
func setupDecomposeTestWithDevPlan(t *testing.T, specContent, devPlanContent string, approve bool) (*DecomposeService, *EntityService, string, string) {
	t.Helper()

	stateRoot := t.TempDir()
	repoRoot := t.TempDir()

	entitySvc := NewEntityService(stateRoot)
	docSvc := NewDocumentService(stateRoot, repoRoot)

	planID := "P1-decompose-plan"
	writeDecomposeTestPlan(t, entitySvc, planID)

	featResult, err := entitySvc.CreateFeature(CreateFeatureInput{
		Slug:      "test-feature",
		Parent:    planID,
		Summary:   "Test feature for dev-plan-aware decompose",
		CreatedBy: "tester",
		Name:      "Test feature",
	})
	if err != nil {
		t.Fatalf("create feature: %v", err)
	}
	featureID := featResult.ID
	featureSlug := "test-feature"

	// Write and approve spec.
	specPath := "work/spec/test-spec.md"
	specFull := filepath.Join(repoRoot, specPath)
	if err := os.MkdirAll(filepath.Dir(specFull), 0o755); err != nil {
		t.Fatalf("mkdir for spec: %v", err)
	}
	if err := os.WriteFile(specFull, []byte(specContent), 0o644); err != nil {
		t.Fatalf("write spec: %v", err)
	}
	specDoc, err := docSvc.SubmitDocument(SubmitDocumentInput{
		Path:      specPath,
		Type:      "specification",
		Title:     "Test Specification",
		Owner:     featureID,
		CreatedBy: "tester",
	})
	if err != nil {
		t.Fatalf("submit spec: %v", err)
	}
	if _, err := docSvc.ApproveDocument(ApproveDocumentInput{
		ID:         specDoc.ID,
		ApprovedBy: "tester",
	}); err != nil {
		t.Fatalf("approve spec: %v", err)
	}
	_, err = entitySvc.UpdateEntity(UpdateEntityInput{
		Type:   "feature",
		ID:     featureID,
		Slug:   featureSlug,
		Fields: map[string]string{"spec": specDoc.ID},
	})
	if err != nil {
		t.Fatalf("link spec to feature: %v", err)
	}

	// Write and optionally approve dev plan.
	devPlanPath := "work/dev-plan/test-dev-plan.md"
	devPlanFull := filepath.Join(repoRoot, devPlanPath)
	if err := os.MkdirAll(filepath.Dir(devPlanFull), 0o755); err != nil {
		t.Fatalf("mkdir for dev plan: %v", err)
	}
	if err := os.WriteFile(devPlanFull, []byte(devPlanContent), 0o644); err != nil {
		t.Fatalf("write dev plan: %v", err)
	}
	devPlanDoc, err := docSvc.SubmitDocument(SubmitDocumentInput{
		Path:      devPlanPath,
		Type:      "dev-plan",
		Title:     "Test Dev Plan",
		Owner:     featureID,
		CreatedBy: "tester",
	})
	if err != nil {
		t.Fatalf("submit dev plan: %v", err)
	}
	if approve {
		if _, err := docSvc.ApproveDocument(ApproveDocumentInput{
			ID:         devPlanDoc.ID,
			ApprovedBy: "tester",
		}); err != nil {
			t.Fatalf("approve dev plan: %v", err)
		}
	}

	decomposeSvc := NewDecomposeService(entitySvc, docSvc)
	return decomposeSvc, entitySvc, featureID, featureSlug
}

// AC-001: Feature with approved dev plan + valid Task Breakdown -> tasks from dev plan.
func TestDecomposeFeature_DevPlanPath_TasksFromDevPlan(t *testing.T) {
	t.Parallel()

	svc, _, featureID, featureSlug := setupDecomposeTestWithDevPlan(t, specWithACs, devPlanWithTwoTasks, true)

	result, err := svc.DecomposeFeature(DecomposeInput{FeatureID: featureID})
	if err != nil {
		t.Fatalf("DecomposeFeature() error = %v", err)
	}

	// Should produce exactly two tasks (from the dev plan, not AC heuristic).
	if got := len(result.Proposal.Tasks); got != 2 {
		t.Fatalf("TotalTasks = %d, want 2 (from dev plan)", got)
	}

	task1 := result.Proposal.Tasks[0]
	task2 := result.Proposal.Tasks[1]

	// Task 1: correct name and slug.
	wantTask1Name := "Implement data model"
	if task1.Name != wantTask1Name {
		t.Errorf("Tasks[0].Name = %q, want %q", task1.Name, wantTask1Name)
	}
	wantTask1Slug := featureSlug + "-" + "implement-data-model"
	if task1.Slug != wantTask1Slug {
		t.Errorf("Tasks[0].Slug = %q, want %q", task1.Slug, wantTask1Slug)
	}

	// Task 2 depends on Task 1.
	if len(task2.DependsOn) != 1 || task2.DependsOn[0] != task1.Slug {
		t.Errorf("Tasks[1].DependsOn = %v, want [%q]", task2.DependsOn, task1.Slug)
	}

	// Task 1 has no dependencies.
	if len(task1.DependsOn) != 0 {
		t.Errorf("Tasks[0].DependsOn = %v, want nil", task1.DependsOn)
	}
}

// AC-002: Draft dev plan -> falls back to AC heuristic, no error.
func TestDecomposeFeature_DraftDevPlan_FallsBackToAC(t *testing.T) {
	t.Parallel()

	// approve=false: dev plan stays in draft; should be ignored.
	svc, _, featureID, _ := setupDecomposeTestWithDevPlan(t, specWithACs, devPlanWithTwoTasks, false)

	result, err := svc.DecomposeFeature(DecomposeInput{FeatureID: featureID})
	if err != nil {
		t.Fatalf("DecomposeFeature() error = %v (draft dev plan should not block)", err)
	}

	// AC heuristic path: guidance should NOT contain "dev-plan-tasks".
	for _, g := range result.GuidanceApplied {
		if g == "dev-plan-tasks" {
			t.Errorf("GuidanceApplied = %v, must NOT contain %q on AC heuristic path", result.GuidanceApplied, "dev-plan-tasks")
			break
		}
	}

	// Should have at least one task from the AC heuristic.
	if len(result.Proposal.Tasks) == 0 {
		t.Error("expected tasks from AC heuristic fallback, got none")
	}
}

// AC-003: Approved dev plan without ## Task Breakdown -> AC heuristic + warning.
func TestDecomposeFeature_DevPlanMissingTaskBreakdown_FallsBackToAC(t *testing.T) {
	t.Parallel()

	svc, _, featureID, _ := setupDecomposeTestWithDevPlan(t, specWithACs, devPlanNoTaskBreakdown, true)

	result, err := svc.DecomposeFeature(DecomposeInput{FeatureID: featureID})
	if err != nil {
		t.Fatalf("DecomposeFeature() error = %v", err)
	}

	// Should fall back to AC heuristic.
	for _, g := range result.GuidanceApplied {
		if g == "dev-plan-tasks" {
			t.Errorf("GuidanceApplied must NOT contain \"dev-plan-tasks\" on fallback path; got %v", result.GuidanceApplied)
			break
		}
	}

	// Warnings must mention the Task Breakdown missing/empty.
	warnFound := false
	for _, w := range result.Proposal.Warnings {
		if contains(w, "Task Breakdown absent or empty") {
			warnFound = true
			break
		}
	}
	if !warnFound {
		t.Errorf("Warnings = %v, want one entry mentioning \"Task Breakdown absent or empty\"", result.Proposal.Warnings)
	}

	// Should still produce tasks from AC heuristic.
	if len(result.Proposal.Tasks) == 0 {
		t.Error("expected tasks from AC heuristic fallback, got none")
	}
}

// AC-004: Approved dev plan + spec with no ACs -> valid proposal, no zero-criteria error.
func TestDecomposeFeature_DevPlanPath_NoACsInSpec(t *testing.T) {
	t.Parallel()

	svc, _, featureID, _ := setupDecomposeTestWithDevPlan(t, specWithNoACs, devPlanWithTwoTasks, true)

	result, err := svc.DecomposeFeature(DecomposeInput{FeatureID: featureID})
	if err != nil {
		t.Fatalf("DecomposeFeature() error = %v; want no error when dev plan has task breakdown", err)
	}

	// Should have 2 tasks from dev plan (zero-criteria gate must not fire).
	if got := len(result.Proposal.Tasks); got != 2 {
		t.Errorf("TotalTasks = %d, want 2", got)
	}
}

// AC-005: No dev plan + spec with no ACs -> zero-criteria error (unchanged behaviour).
func TestDecomposeFeature_NoDevPlan_NoACs_ZeroCriteriaError(t *testing.T) {
	t.Parallel()

	// Use setupDecomposeTest (no dev plan) with a spec that has no ACs.
	svc, featureID, _ := setupDecomposeTest(t, specWithNoACs)

	_, err := svc.DecomposeFeature(DecomposeInput{FeatureID: featureID})
	if err == nil {
		t.Fatal("expected zero-criteria error, got nil")
	}
	if !contains(err.Error(), "no acceptance criteria found in spec") {
		t.Errorf("error = %q, want it to contain \"no acceptance criteria found in spec\"", err.Error())
	}
}

// AC-006: parseDevPlanTasks with Medium effort maps to Estimate == 3.0.
func TestParseDevPlanTasks_EffortMapping(t *testing.T) {
	t.Parallel()

	devPlan := []byte(`# Dev Plan

## Task Breakdown

### Task 1: Widget implementation
- **Effort:** Medium
- **Depends on:** None
`)

	tasks, ok := parseDevPlanTasks("my-feature", devPlan)
	if !ok {
		t.Fatal("parseDevPlanTasks returned false, want true")
	}
	if len(tasks) != 1 {
		t.Fatalf("len(tasks) = %d, want 1", len(tasks))
	}
	if tasks[0].Estimate == nil {
		t.Fatal("Estimate is nil, want 3.0")
	}
	if *tasks[0].Estimate != 3.0 {
		t.Errorf("Estimate = %v, want 3.0", *tasks[0].Estimate)
	}
}

// AC-007: parseDevPlanTasks with no Spec requirements field -> Covers == nil.
func TestParseDevPlanTasks_CoversNilWhenAbsent(t *testing.T) {
	t.Parallel()

	devPlan := []byte(`# Dev Plan

## Task Breakdown

### Task 1: Widget implementation
- **Effort:** Small
- **Depends on:** None
`)

	tasks, ok := parseDevPlanTasks("my-feature", devPlan)
	if !ok {
		t.Fatal("parseDevPlanTasks returned false, want true")
	}
	if len(tasks) != 1 {
		t.Fatalf("len(tasks) = %d, want 1", len(tasks))
	}
	if tasks[0].Covers != nil {
		t.Errorf("Covers = %v, want nil (no Spec requirements field)", tasks[0].Covers)
	}
}

// AC-008: GuidanceApplied contains "dev-plan-tasks", NOT "test-tasks-explicit".
func TestDecomposeFeature_DevPlanPath_GuidanceApplied(t *testing.T) {
	t.Parallel()

	svc, _, featureID, _ := setupDecomposeTestWithDevPlan(t, specWithACs, devPlanWithTwoTasks, true)

	result, err := svc.DecomposeFeature(DecomposeInput{FeatureID: featureID})
	if err != nil {
		t.Fatalf("DecomposeFeature() error = %v", err)
	}

	foundDevPlan := false
	for _, g := range result.GuidanceApplied {
		if g == "dev-plan-tasks" {
			foundDevPlan = true
		}
		if g == "test-tasks-explicit" {
			t.Errorf("GuidanceApplied = %v, must NOT contain \"test-tasks-explicit\" on dev plan path", result.GuidanceApplied)
		}
	}
	if !foundDevPlan {
		t.Errorf("GuidanceApplied = %v, want it to contain \"dev-plan-tasks\"", result.GuidanceApplied)
	}
}

// AC-009: SliceDetails are populated on the dev plan path.
func TestDecomposeFeature_DevPlanPath_SliceDetails(t *testing.T) {
	t.Parallel()

	// specWithACs has L2 sections (Data Layer, API Layer) which produce slices.
	svc, _, featureID, _ := setupDecomposeTestWithDevPlan(t, specWithACs, devPlanWithTwoTasks, true)

	result, err := svc.DecomposeFeature(DecomposeInput{FeatureID: featureID})
	if err != nil {
		t.Fatalf("DecomposeFeature() error = %v", err)
	}

	if len(result.Proposal.SliceDetails) == 0 {
		t.Error("expected non-empty SliceDetails on dev plan path")
	}
	for i, s := range result.Proposal.SliceDetails {
		if s.Name == "" {
			t.Errorf("SliceDetails[%d].Name is empty", i)
		}
	}
}

// AC-010: A dev plan proposal can be fully applied (all tasks created without error).
func TestDecomposeApply_DevPlanProposal(t *testing.T) {
	t.Parallel()

	svc, entitySvc, featureID, _ := setupDecomposeTestWithDevPlan(t, specWithACs, devPlanWithTwoTasks, true)

	result, err := svc.DecomposeFeature(DecomposeInput{FeatureID: featureID})
	if err != nil {
		t.Fatalf("DecomposeFeature() error = %v", err)
	}

	// Apply: create each proposed task using the entity service.
	for _, pt := range result.Proposal.Tasks {
		_, createErr := entitySvc.CreateTask(CreateTaskInput{
			ParentFeature: featureID,
			Slug:          pt.Slug,
			Name:          pt.Name,
			Summary:       pt.Summary,
		})
		if createErr != nil {
			t.Errorf("CreateTask(%q) error = %v", pt.Slug, createErr)
		}
	}
}

// ---------------------------------------------------------------------------
// parseDevPlanTasks unit tests: dependency resolution
// ---------------------------------------------------------------------------

func TestParseDevPlanTasks_DependsOnTask1_ResolvesToFirstSlug(t *testing.T) {
	t.Parallel()

	devPlan := []byte(`# Dev Plan

## Task Breakdown

### Task 1: Setup environment
- **Effort:** Small
- **Depends on:** None

### Task 2: Write service
- **Effort:** Medium
- **Depends on:** Task 1
`)

	tasks, ok := parseDevPlanTasks("feat", devPlan)
	if !ok {
		t.Fatal("parseDevPlanTasks returned false")
	}
	if len(tasks) != 2 {
		t.Fatalf("len(tasks) = %d, want 2", len(tasks))
	}
	// Task 2 should depend on the slug of Task 1.
	if len(tasks[1].DependsOn) != 1 || tasks[1].DependsOn[0] != tasks[0].Slug {
		t.Errorf("Tasks[1].DependsOn = %v, want [%q]", tasks[1].DependsOn, tasks[0].Slug)
	}
}

func TestParseDevPlanTasks_DependsOnNone_IsNil(t *testing.T) {
	t.Parallel()

	devPlan := []byte(`# Dev Plan

## Task Breakdown

### Task 1: Setup environment
- **Effort:** Small
- **Depends on:** None
`)

	tasks, ok := parseDevPlanTasks("feat", devPlan)
	if !ok {
		t.Fatal("parseDevPlanTasks returned false")
	}
	if tasks[0].DependsOn != nil {
		t.Errorf("DependsOn = %v, want nil for \"None\"", tasks[0].DependsOn)
	}
}

func TestParseDevPlanTasks_DependsOnNoneParens_IsNil(t *testing.T) {
	t.Parallel()

	devPlan := []byte(`# Dev Plan

## Task Breakdown

### Task 1: Setup environment
- **Effort:** Small
- **Depends on:** None (independent)
`)

	tasks, ok := parseDevPlanTasks("feat", devPlan)
	if !ok {
		t.Fatal("parseDevPlanTasks returned false")
	}
	if tasks[0].DependsOn != nil {
		t.Errorf("DependsOn = %v, want nil for \"None (independent)\"", tasks[0].DependsOn)
	}
}
