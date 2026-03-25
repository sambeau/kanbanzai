package service

import (
	"os"
	"path/filepath"
	"testing"

	"kanbanzai/internal/model"
	"kanbanzai/internal/storage"
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
	// Plus a test task added by the test-tasks-explicit guidance.
	proposal := result.Proposal
	if proposal.TotalTasks < 5 {
		t.Errorf("TotalTasks = %d, want at least 5 (one per acceptance criterion)", proposal.TotalTasks)
	}

	// Each proposed task must have slug, summary, and rationale.
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

	expectedRules := []string{
		"one-ac-per-task",
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

func TestDecomposeFeature_NoACs_FallsBackToSections(t *testing.T) {
	t.Parallel()

	specContent := `# Feature Spec

## Database Layer

Design the database schema.

## API Layer

Implement REST endpoints.

## UI Layer

Build the user interface.
`
	svc, featureID, _ := setupDecomposeTest(t, specContent)

	result, err := svc.DecomposeFeature(DecomposeInput{FeatureID: featureID})
	if err != nil {
		t.Fatalf("DecomposeFeature() error = %v", err)
	}

	// Should derive tasks from sections since no checkboxes exist.
	if result.Proposal.TotalTasks == 0 {
		t.Fatal("expected non-empty proposal from section-based fallback")
	}

	// Should warn about missing acceptance criteria.
	found := false
	for _, w := range result.Proposal.Warnings {
		if contains(w, "No acceptance criteria") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Warnings = %v, want warning about missing acceptance criteria", result.Proposal.Warnings)
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
				Rationale: "Covers acceptance criterion: users can log in",
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
		"title":      "Test Plan",
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
