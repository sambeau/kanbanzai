package context

import (
	"fmt"
	"strings"
	"testing"

	"github.com/sambeau/kanbanzai/internal/binding"
	"github.com/sambeau/kanbanzai/internal/skill"
)

// ─── Mock implementations ─────────────────────────────────────────────────────

type mockRoleResolver struct {
	roles map[string]*ResolvedRole
	err   error
}

func (m *mockRoleResolver) Resolve(id string) (*ResolvedRole, error) {
	if m.err != nil {
		return nil, m.err
	}
	r, ok := m.roles[id]
	if !ok {
		return nil, fmt.Errorf("role %q not found", id)
	}
	return r, nil
}

type mockSkillResolver struct {
	skills map[string]*skill.Skill
	err    error
}

func (m *mockSkillResolver) Load(name string) (*skill.Skill, error) {
	if m.err != nil {
		return nil, m.err
	}
	s, ok := m.skills[name]
	if !ok {
		return nil, fmt.Errorf("skill %q not found", name)
	}
	return s, nil
}

type mockBindingResolver struct {
	bindings map[string]*binding.StageBinding
	err      error
}

func (m *mockBindingResolver) Lookup(stage string) (*binding.StageBinding, error) {
	if m.err != nil {
		return nil, m.err
	}
	b, ok := m.bindings[stage]
	if !ok {
		return nil, fmt.Errorf("no binding for stage %q", stage)
	}
	return b, nil
}

type mockKnowledgeSurfacer struct {
	entries []SurfacedEntry
	err     error
}

func (m *mockKnowledgeSurfacer) Surface(_ SurfaceInput) ([]SurfacedEntry, error) {
	return m.entries, m.err
}

// ─── Test helpers ─────────────────────────────────────────────────────────────

func testRole() *ResolvedRole {
	return &ResolvedRole{
		ID:       "implementer-go",
		Identity: "Senior Go engineer focused on clean, tested code",
		Vocabulary: []string{
			"acceptance criteria",
			"test coverage",
			"idiomatic Go",
		},
		AntiPatterns: []AntiPattern{
			{
				Name:    "over-engineering",
				Detect:  "adding abstractions for hypothetical futures",
				Because: "unnecessary complexity slows iteration",
				Resolve: "implement only what is required now",
			},
		},
		Tools: []string{"entity", "doc", "status", "knowledge"},
	}
}

func testSkill() *skill.Skill {
	return &skill.Skill{
		Frontmatter: skill.SkillFrontmatter{
			Name: "implement-task",
			Description: skill.SkillDescription{
				Expert:  "Structured task execution",
				Natural: "Guides you through implementing a task",
			},
			Triggers:        []string{"implement a task"},
			Roles:           []string{"implementer", "implementer-go"},
			Stage:           "developing",
			ConstraintLevel: "medium",
		},
		Sections: []skill.BodySection{
			{Heading: "Vocabulary", Content: "- spec-driven\n- incremental\n"},
			{Heading: "Anti-Patterns", Content: "- Skipping tests\n- Ignoring acceptance criteria\n"},
			{Heading: "Procedure", Content: "1. Read the spec\n2. Write tests\n3. Implement\n4. Verify\n"},
			{Heading: "Output Format", Content: "Commit message with task ID reference.\n"},
			{Heading: "Examples", Content: "Example: feat(TASK-001): implement auth flow\n"},
			{Heading: "Evaluation Criteria", Content: "- All acceptance criteria met\n- Tests pass\n"},
			{Heading: "Questions This Skill Answers", Content: "- How do I implement a task?\n"},
		},
	}
}

func testBinding() *binding.StageBinding {
	return &binding.StageBinding{
		Description:   "Development stage",
		Orchestration: "single-agent",
		Roles:         []string{"implementer-go"},
		Skills:        []string{"implement-task"},
		EffortBudget:  "1-3 story points, under 4 hours",
	}
}

func testPipeline() *Pipeline {
	return &Pipeline{
		Roles: &mockRoleResolver{
			roles: map[string]*ResolvedRole{
				"implementer-go": testRole(),
			},
		},
		Skills: &mockSkillResolver{
			skills: map[string]*skill.Skill{
				"implement-task": testSkill(),
			},
		},
		Bindings: &mockBindingResolver{
			bindings: map[string]*binding.StageBinding{
				"developing": testBinding(),
			},
		},
		Knowledge: &NoOpSurfacer{},
	}
}

func testInput() PipelineInput {
	return PipelineInput{
		TaskID: "TASK-01KN5D2PJTMWZ",
		TaskState: map[string]any{
			"id":             "TASK-01KN5D2PJTMWZ",
			"summary":        "Implement the authentication flow",
			"parent_feature": "FEAT-01KN588PE43M6",
		},
		FeatureState: map[string]any{
			"id":     "FEAT-01KN588PE43M6",
			"status": "developing",
		},
	}
}

// ─── Step 0: Lifecycle Validation ─────────────────────────────────────────────

func TestStepValidateLifecycle_ValidStatus(t *testing.T) {
	t.Parallel()
	p := testPipeline()
	for _, status := range workableStatuses {
		state := &PipelineState{
			Input: PipelineInput{
				TaskID:       "TASK-001",
				FeatureState: map[string]any{"status": status},
			},
		}
		if err := p.stepValidateLifecycle(state); err != nil {
			t.Errorf("status %q should be valid, got error: %v", status, err)
		}
	}
}

func TestStepValidateLifecycle_InvalidStatus(t *testing.T) {
	t.Parallel()
	p := testPipeline()
	state := &PipelineState{
		Input: PipelineInput{
			TaskID:       "TASK-001",
			FeatureState: map[string]any{"status": "draft"},
		},
	}
	err := p.stepValidateLifecycle(state)
	if err == nil {
		t.Fatal("expected error for draft status")
	}
	if !strings.Contains(err.Error(), "step 0") {
		t.Errorf("error should mention step 0: %v", err)
	}
	if !strings.Contains(err.Error(), "draft") {
		t.Errorf("error should mention current status: %v", err)
	}
	if !strings.Contains(err.Error(), "Hint:") {
		t.Errorf("error should contain remediation hint: %v", err)
	}
}

func TestStepValidateLifecycle_NilFeatureState(t *testing.T) {
	t.Parallel()
	p := testPipeline()
	state := &PipelineState{
		Input: PipelineInput{
			TaskID:       "TASK-001",
			FeatureState: nil,
		},
	}
	err := p.stepValidateLifecycle(state)
	if err == nil {
		t.Fatal("expected error for nil feature state")
	}
	if !strings.Contains(err.Error(), "no parent feature") {
		t.Errorf("error should mention missing parent: %v", err)
	}
}

// ─── Step 1: Stage Resolution ─────────────────────────────────────────────────

func TestStepResolveStage_Success(t *testing.T) {
	t.Parallel()
	p := testPipeline()
	state := &PipelineState{
		Input: PipelineInput{
			FeatureState: map[string]any{"status": "developing"},
		},
	}
	if err := p.stepResolveStage(state); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if state.Stage != "developing" {
		t.Errorf("stage = %q, want %q", state.Stage, "developing")
	}
}

func TestStepResolveStage_EmptyStatus(t *testing.T) {
	t.Parallel()
	p := testPipeline()
	state := &PipelineState{
		Input: PipelineInput{
			TaskID:       "TASK-001",
			FeatureState: map[string]any{},
		},
	}
	err := p.stepResolveStage(state)
	if err == nil {
		t.Fatal("expected error for empty status")
	}
	if !strings.Contains(err.Error(), "step 1") {
		t.Errorf("error should mention step 1: %v", err)
	}
}

// ─── Step 2: Binding Lookup ───────────────────────────────────────────────────

func TestStepLookupBinding_Success(t *testing.T) {
	t.Parallel()
	p := testPipeline()
	state := &PipelineState{Stage: "developing"}
	if err := p.stepLookupBinding(state); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if state.Binding == nil {
		t.Fatal("binding should not be nil")
	}
	if state.Binding.Orchestration != "single-agent" {
		t.Errorf("orchestration = %q, want %q", state.Binding.Orchestration, "single-agent")
	}
}

func TestStepLookupBinding_UnknownStage(t *testing.T) {
	t.Parallel()
	p := testPipeline()
	state := &PipelineState{Stage: "nonexistent-stage"}
	err := p.stepLookupBinding(state)
	if err == nil {
		t.Fatal("expected error for unknown stage")
	}
	if !strings.Contains(err.Error(), "step 2") {
		t.Errorf("error should mention step 2: %v", err)
	}
	if !strings.Contains(err.Error(), "nonexistent-stage") {
		t.Errorf("error should mention the stage: %v", err)
	}
}

func TestStepLookupBinding_NilBindings(t *testing.T) {
	t.Parallel()
	p := &Pipeline{}
	state := &PipelineState{Stage: "developing"}
	err := p.stepLookupBinding(state)
	if err == nil {
		t.Fatal("expected error for nil bindings resolver")
	}
	if !strings.Contains(err.Error(), "no binding resolver") {
		t.Errorf("error should mention missing resolver: %v", err)
	}
}

// ─── Step 3: Inclusion/Exclusion ──────────────────────────────────────────────

func TestStepApplyInclusion_DefaultsToIncludeAll(t *testing.T) {
	t.Parallel()
	p := testPipeline()
	state := &PipelineState{Stage: "developing", Binding: testBinding()}
	p.stepApplyInclusion(state)

	if !state.Inclusion.IncludeSpec {
		t.Error("IncludeSpec should default to true")
	}
	if !state.Inclusion.IncludeKnowledge {
		t.Error("IncludeKnowledge should default to true")
	}
	if !state.Inclusion.IncludeExamples {
		t.Error("IncludeExamples should default to true")
	}
	if !state.Inclusion.IncludeReferences {
		t.Error("IncludeReferences should default to true")
	}
}

func TestStepApplyInclusion_ReviewExcludesReferences(t *testing.T) {
	t.Parallel()
	p := testPipeline()
	state := &PipelineState{Stage: "reviewing", Binding: testBinding()}
	p.stepApplyInclusion(state)

	if state.Inclusion.IncludeReferences {
		t.Error("IncludeReferences should be false for reviewing stage")
	}
	// Other categories should still be included.
	if !state.Inclusion.IncludeSpec {
		t.Error("IncludeSpec should still be true for reviewing")
	}
}

// ─── Step 4: Orchestration Metadata ───────────────────────────────────────────

func TestStepExtractOrchestration_Basic(t *testing.T) {
	t.Parallel()
	p := testPipeline()
	state := &PipelineState{Binding: testBinding()}
	p.stepExtractOrchestration(state)

	if state.Orchestration.Pattern != "single-agent" {
		t.Errorf("pattern = %q, want %q", state.Orchestration.Pattern, "single-agent")
	}
	if state.Orchestration.EffortBudget != "1-3 story points, under 4 hours" {
		t.Errorf("effort budget mismatch: %q", state.Orchestration.EffortBudget)
	}
}

func TestStepExtractOrchestration_WithMaxReviewCycles(t *testing.T) {
	t.Parallel()
	p := testPipeline()
	cycles := 3
	b := testBinding()
	b.MaxReviewCycles = &cycles
	state := &PipelineState{Binding: b}
	p.stepExtractOrchestration(state)

	if state.Orchestration.MaxReviewCycles != 3 {
		t.Errorf("max_review_cycles = %d, want 3", state.Orchestration.MaxReviewCycles)
	}
}

func TestStepExtractOrchestration_WithPrerequisites(t *testing.T) {
	t.Parallel()
	p := testPipeline()
	b := testBinding()
	b.Prerequisites = &binding.Prerequisites{
		Documents: []binding.DocumentPrereq{
			{Type: "specification", Status: "approved"},
		},
	}
	state := &PipelineState{Binding: b}
	p.stepExtractOrchestration(state)

	if len(state.Orchestration.Prerequisites) != 1 {
		t.Fatalf("prerequisites len = %d, want 1", len(state.Orchestration.Prerequisites))
	}
	if !strings.Contains(state.Orchestration.Prerequisites[0], "specification") {
		t.Errorf("prerequisite should mention specification: %q", state.Orchestration.Prerequisites[0])
	}
}

// ─── Step 5: Role Resolution ──────────────────────────────────────────────────

func TestStepResolveRole_Success(t *testing.T) {
	t.Parallel()
	p := testPipeline()
	state := &PipelineState{
		Binding: testBinding(),
	}
	if err := p.stepResolveRole(state); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if state.Role == nil {
		t.Fatal("role should not be nil")
	}
	if state.Role.ID != "implementer-go" {
		t.Errorf("role ID = %q, want %q", state.Role.ID, "implementer-go")
	}
	if len(state.MergedVocab) != 3 {
		t.Errorf("merged vocab len = %d, want 3", len(state.MergedVocab))
	}
	if len(state.MergedAnti) != 1 {
		t.Errorf("merged anti-patterns len = %d, want 1", len(state.MergedAnti))
	}
}

func TestStepResolveRole_CallerOverride(t *testing.T) {
	t.Parallel()
	p := &Pipeline{
		Roles: &mockRoleResolver{
			roles: map[string]*ResolvedRole{
				"custom-role": {
					ID:         "custom-role",
					Identity:   "A custom role",
					Vocabulary: []string{"custom-term"},
				},
			},
		},
	}
	state := &PipelineState{
		Input:   PipelineInput{Role: "custom-role"},
		Binding: testBinding(),
	}
	if err := p.stepResolveRole(state); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if state.Role.ID != "custom-role" {
		t.Errorf("role ID = %q, want %q (caller override)", state.Role.ID, "custom-role")
	}
}

func TestStepResolveRole_MissingRole(t *testing.T) {
	t.Parallel()
	p := testPipeline()
	b := testBinding()
	b.Roles = []string{"nonexistent-role"}
	state := &PipelineState{Binding: b}
	err := p.stepResolveRole(state)
	if err == nil {
		t.Fatal("expected error for missing role")
	}
	if !strings.Contains(err.Error(), "step 5") {
		t.Errorf("error should mention step 5: %v", err)
	}
	if !strings.Contains(err.Error(), "nonexistent-role") {
		t.Errorf("error should mention the role ID: %v", err)
	}
}

func TestStepResolveRole_NoRoleSpecified(t *testing.T) {
	t.Parallel()
	p := testPipeline()
	b := testBinding()
	b.Roles = nil
	state := &PipelineState{Binding: b}
	err := p.stepResolveRole(state)
	if err == nil {
		t.Fatal("expected error when no role is specified")
	}
	if !strings.Contains(err.Error(), "no role specified") {
		t.Errorf("error should mention no role specified: %v", err)
	}
}

// ─── Step 6: Skill Loading ────────────────────────────────────────────────────

func TestStepLoadSkill_Success(t *testing.T) {
	t.Parallel()
	p := testPipeline()
	state := &PipelineState{
		Binding: testBinding(),
		// Pre-populate role vocab to test merge ordering.
		MergedVocab: []string{"role-term-1"},
	}
	if err := p.stepLoadSkill(state); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if state.Skill == nil {
		t.Fatal("skill should not be nil")
	}
	if state.Skill.Frontmatter.Name != "implement-task" {
		t.Errorf("skill name = %q, want %q", state.Skill.Frontmatter.Name, "implement-task")
	}
	// Merged vocab should have role term first, then skill terms (FR-009).
	if len(state.MergedVocab) < 2 {
		t.Fatalf("merged vocab len = %d, want >= 2", len(state.MergedVocab))
	}
	if state.MergedVocab[0] != "role-term-1" {
		t.Errorf("first vocab term should be role term, got %q", state.MergedVocab[0])
	}
}

func TestStepLoadSkill_MissingSkill(t *testing.T) {
	t.Parallel()
	p := testPipeline()
	b := testBinding()
	b.Skills = []string{"nonexistent-skill"}
	state := &PipelineState{Binding: b}
	err := p.stepLoadSkill(state)
	if err == nil {
		t.Fatal("expected error for missing skill")
	}
	if !strings.Contains(err.Error(), "step 6") {
		t.Errorf("error should mention step 6: %v", err)
	}
	if !strings.Contains(err.Error(), "nonexistent-skill") {
		t.Errorf("error should mention the skill name: %v", err)
	}
}

func TestStepLoadSkill_NoSkillSpecified(t *testing.T) {
	t.Parallel()
	p := testPipeline()
	b := testBinding()
	b.Skills = nil
	state := &PipelineState{Binding: b, Stage: "developing"}
	err := p.stepLoadSkill(state)
	if err == nil {
		t.Fatal("expected error when no skill is specified")
	}
	if !strings.Contains(err.Error(), "no skill specified") {
		t.Errorf("error should mention no skill: %v", err)
	}
}

// ─── Step 7: Knowledge Surfacing ──────────────────────────────────────────────

func TestStepSurfaceKnowledge_WithEntries(t *testing.T) {
	t.Parallel()
	entries := []SurfacedEntry{
		{ID: "KE-001", Topic: "testing", Content: "Always write tests first", Score: 0.9},
		{ID: "KE-002", Topic: "errors", Content: "Wrap errors with context", Score: 0.8},
	}
	p := &Pipeline{
		Knowledge: &mockKnowledgeSurfacer{entries: entries},
	}
	state := &PipelineState{
		Input: PipelineInput{
			TaskState: map[string]any{"id": "TASK-001"},
		},
		Role:      testRole(),
		Skill:     testSkill(),
		Inclusion: InclusionStrategy{IncludeKnowledge: true},
	}
	p.stepSurfaceKnowledge(state)

	if len(state.Knowledge) != 2 {
		t.Errorf("knowledge entries = %d, want 2", len(state.Knowledge))
	}
}

func TestStepSurfaceKnowledge_NilSurfacer(t *testing.T) {
	t.Parallel()
	p := &Pipeline{Knowledge: nil}
	state := &PipelineState{
		Inclusion: InclusionStrategy{IncludeKnowledge: true},
	}
	p.stepSurfaceKnowledge(state)
	if len(state.Knowledge) != 0 {
		t.Error("knowledge should be empty with nil surfacer")
	}
}

func TestStepSurfaceKnowledge_ExcludedByInclusion(t *testing.T) {
	t.Parallel()
	entries := []SurfacedEntry{
		{ID: "KE-001", Topic: "testing", Content: "some content", Score: 0.9},
	}
	p := &Pipeline{
		Knowledge: &mockKnowledgeSurfacer{entries: entries},
	}
	state := &PipelineState{
		Inclusion: InclusionStrategy{IncludeKnowledge: false},
	}
	p.stepSurfaceKnowledge(state)
	if len(state.Knowledge) != 0 {
		t.Error("knowledge should be empty when IncludeKnowledge is false")
	}
}

func TestStepSurfaceKnowledge_ErrorIsSwallowed(t *testing.T) {
	t.Parallel()
	p := &Pipeline{
		Knowledge: &mockKnowledgeSurfacer{err: fmt.Errorf("db unavailable")},
	}
	state := &PipelineState{
		Input: PipelineInput{
			TaskState: map[string]any{"id": "TASK-001"},
		},
		Inclusion: InclusionStrategy{IncludeKnowledge: true},
	}
	p.stepSurfaceKnowledge(state)
	if len(state.Knowledge) != 0 {
		t.Error("knowledge should be empty when surfacer returns error")
	}
}

// ─── Step 8: Tool Guidance ────────────────────────────────────────────────────

func TestStepToolGuidance_WithTools(t *testing.T) {
	t.Parallel()
	p := testPipeline()
	state := &PipelineState{Role: testRole()}
	p.stepToolGuidance(state)

	if state.ToolGuidance == "" {
		t.Fatal("tool guidance should not be empty")
	}
	if !strings.Contains(state.ToolGuidance, "entity") {
		t.Errorf("guidance should list entity tool: %q", state.ToolGuidance)
	}
	if !strings.Contains(state.ToolGuidance, "knowledge") {
		t.Errorf("guidance should list knowledge tool: %q", state.ToolGuidance)
	}
}

func TestStepToolGuidance_NoTools(t *testing.T) {
	t.Parallel()
	p := testPipeline()
	role := testRole()
	role.Tools = nil
	state := &PipelineState{Role: role}
	p.stepToolGuidance(state)

	if state.ToolGuidance != "" {
		t.Errorf("tool guidance should be empty when no tools, got %q", state.ToolGuidance)
	}
}

func TestStepToolGuidance_NilRole(t *testing.T) {
	t.Parallel()
	p := testPipeline()
	state := &PipelineState{Role: nil}
	p.stepToolGuidance(state)
	if state.ToolGuidance != "" {
		t.Errorf("tool guidance should be empty with nil role, got %q", state.ToolGuidance)
	}
}

// ─── Step 9: Token Budget ─────────────────────────────────────────────────────

func TestStepTokenBudget_UnderThreshold(t *testing.T) {
	t.Parallel()
	p := testPipeline()
	state := &PipelineState{
		Sections: []PipelineSection{
			{Position: 1, Content: "short", Tokens: 2},
		},
	}
	if err := p.stepTokenBudget(state); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if state.TokenEstimate != 2 {
		t.Errorf("token estimate = %d, want 2", state.TokenEstimate)
	}
}

func TestStepTokenBudget_RefuseAbove60Percent(t *testing.T) {
	t.Parallel()
	p := &Pipeline{WindowSize: 1000}
	state := &PipelineState{
		Sections: []PipelineSection{
			{Position: 1, Content: "a", Tokens: 700}, // 70% of 1000
		},
	}
	err := p.stepTokenBudget(state)
	if err == nil {
		t.Fatal("expected error when exceeding 60% threshold")
	}
	if !strings.Contains(err.Error(), "step 9") {
		t.Errorf("error should mention step 9: %v", err)
	}
	if !strings.Contains(err.Error(), "60%") {
		t.Errorf("error should mention 60%%: %v", err)
	}
	if !strings.Contains(err.Error(), "split") {
		t.Errorf("error should suggest splitting: %v", err)
	}
}

func TestStepTokenBudget_AllowsAt60Percent(t *testing.T) {
	t.Parallel()
	p := &Pipeline{WindowSize: 1000}
	state := &PipelineState{
		Sections: []PipelineSection{
			{Position: 1, Content: "a", Tokens: 600}, // exactly 60%
		},
	}
	if err := p.stepTokenBudget(state); err != nil {
		t.Fatalf("60%% should be allowed (threshold is >, not >=): %v", err)
	}
}

// ─── Step 10: Build Result ────────────────────────────────────────────────────

func TestStepBuildResult_Warning(t *testing.T) {
	t.Parallel()
	p := &Pipeline{WindowSize: 1000}
	state := &PipelineState{
		Sections: []PipelineSection{
			{Position: 1, Content: "test", Tokens: 450}, // 45% > 40%
		},
		TokenEstimate: 450,
	}
	result := p.stepBuildResult(state)

	if result.TokenWarning == "" {
		t.Error("expected token warning when exceeding 40% threshold")
	}
	if !strings.Contains(result.TokenWarning, "40%") {
		t.Errorf("warning should mention 40%%: %q", result.TokenWarning)
	}
}

func TestStepBuildResult_NoWarning(t *testing.T) {
	t.Parallel()
	p := &Pipeline{WindowSize: 1000}
	state := &PipelineState{
		Sections: []PipelineSection{
			{Position: 1, Content: "test", Tokens: 100}, // 10% < 40%
		},
		TokenEstimate: 100,
	}
	result := p.stepBuildResult(state)

	if result.TokenWarning != "" {
		t.Errorf("unexpected token warning: %q", result.TokenWarning)
	}
}

// ─── Full Pipeline Run ────────────────────────────────────────────────────────

func TestPipelineRun_FullSuccess(t *testing.T) {
	t.Parallel()
	p := testPipeline()
	input := testInput()

	result, err := p.Run(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result == nil {
		t.Fatal("result should not be nil")
	}

	// Must have sections.
	if len(result.Sections) == 0 {
		t.Fatal("result should have sections")
	}

	// Token estimate must be positive (FR-013).
	if result.TotalTokens <= 0 {
		t.Errorf("total tokens = %d, want > 0", result.TotalTokens)
	}

	// Verify section ordering: positions must be non-decreasing (FR-016).
	prevPos := 0
	for _, s := range result.Sections {
		if s.Position < prevPos {
			t.Errorf("section %q at position %d is out of order (previous was %d)",
				s.Label, s.Position, prevPos)
		}
		prevPos = s.Position
	}
}

func TestPipelineRun_SectionLabels(t *testing.T) {
	t.Parallel()
	p := testPipeline()
	input := testInput()

	result, err := p.Run(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	labels := make(map[string]bool)
	for _, s := range result.Sections {
		labels[s.Label] = true
	}

	// Layer 1 sections must always be present.
	for _, required := range []string{"Identity", "Role", "Orchestration", "Vocabulary"} {
		if !labels[required] {
			t.Errorf("missing required Layer 1 section %q", required)
		}
	}
}

func TestPipelineRun_Deterministic(t *testing.T) {
	t.Parallel()
	p := testPipeline()
	input := testInput()

	result1, err := p.Run(input)
	if err != nil {
		t.Fatalf("run 1 error: %v", err)
	}
	result2, err := p.Run(input)
	if err != nil {
		t.Fatalf("run 2 error: %v", err)
	}

	prompt1 := RenderPrompt(result1)
	prompt2 := RenderPrompt(result2)

	if prompt1 != prompt2 {
		t.Error("two runs with identical inputs produced different outputs (NFR-002)")
	}
}

func TestPipelineRun_InvalidFeatureStatus(t *testing.T) {
	t.Parallel()
	p := testPipeline()
	input := testInput()
	input.FeatureState["status"] = "draft"

	_, err := p.Run(input)
	if err == nil {
		t.Fatal("expected error for invalid feature status")
	}
	if !strings.Contains(err.Error(), "step 0") {
		t.Errorf("error should identify step 0: %v", err)
	}
}

func TestPipelineRun_NoBindingForStage(t *testing.T) {
	t.Parallel()
	p := testPipeline()
	input := testInput()
	input.FeatureState["status"] = "designing"

	_, err := p.Run(input)
	if err == nil {
		t.Fatal("expected error for missing binding")
	}
	if !strings.Contains(err.Error(), "step 2") {
		t.Errorf("error should identify step 2: %v", err)
	}
}

func TestPipelineRun_WithKnowledge(t *testing.T) {
	t.Parallel()
	p := testPipeline()
	p.Knowledge = &mockKnowledgeSurfacer{
		entries: []SurfacedEntry{
			{ID: "KE-001", Topic: "testing", Content: "Always write tests", Score: 0.9},
		},
	}
	input := testInput()

	result, err := p.Run(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Knowledge section should appear at position 8.
	found := false
	for _, s := range result.Sections {
		if s.Label == "Knowledge" {
			found = true
			if s.Position != PositionKnowledge {
				t.Errorf("knowledge position = %d, want %d", s.Position, PositionKnowledge)
			}
			if !strings.Contains(s.Content, "Always write tests") {
				t.Error("knowledge content should contain the surfaced entry")
			}
		}
	}
	if !found {
		t.Error("knowledge section should be present when entries exist")
	}
}

func TestPipelineRun_NoKnowledgeSection_WhenEmpty(t *testing.T) {
	t.Parallel()
	p := testPipeline()
	p.Knowledge = &NoOpSurfacer{}
	input := testInput()

	result, err := p.Run(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, s := range result.Sections {
		if s.Label == "Knowledge" {
			t.Error("knowledge section should not appear when no entries (FR-011)")
		}
	}
}

func TestPipelineRun_WithInstructions(t *testing.T) {
	t.Parallel()
	p := testPipeline()
	input := testInput()
	input.Instructions = "Focus only on the auth middleware"

	result, err := p.Run(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	prompt := RenderPrompt(result)
	if !strings.Contains(prompt, "Focus only on the auth middleware") {
		t.Error("instructions should appear in the rendered prompt")
	}
}

func TestPipelineRun_CallerRoleOverridesBinding(t *testing.T) {
	t.Parallel()
	customRole := &ResolvedRole{
		ID:         "custom-reviewer",
		Identity:   "A custom reviewer",
		Vocabulary: []string{"custom-vocab"},
		Tools:      []string{"review"},
	}
	p := testPipeline()
	p.Roles = &mockRoleResolver{
		roles: map[string]*ResolvedRole{
			"custom-reviewer": customRole,
			"implementer-go":  testRole(),
		},
	}
	input := testInput()
	input.Role = "custom-reviewer"

	result, err := p.Run(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	prompt := RenderPrompt(result)
	if !strings.Contains(prompt, "custom-reviewer") {
		t.Error("prompt should use the caller's role override")
	}
}

// ─── Vocabulary and Anti-Pattern Merge Order ──────────────────────────────────

func TestVocabMergeOrder_RoleBeforeSkill(t *testing.T) {
	t.Parallel()
	p := testPipeline()
	input := testInput()

	result, err := p.Run(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var vocabSection *PipelineSection
	for i := range result.Sections {
		if result.Sections[i].Label == "Vocabulary" {
			vocabSection = &result.Sections[i]
			break
		}
	}
	if vocabSection == nil {
		t.Fatal("vocabulary section should exist")
	}

	content := vocabSection.Content
	// Role vocabulary terms should appear before skill terms (FR-009).
	roleTermIdx := strings.Index(content, "idiomatic Go")
	skillTermIdx := strings.Index(content, "spec-driven")
	if roleTermIdx < 0 {
		t.Fatal("role term 'idiomatic Go' not found in vocabulary")
	}
	if skillTermIdx < 0 {
		t.Fatal("skill term 'spec-driven' not found in vocabulary")
	}
	if roleTermIdx > skillTermIdx {
		t.Error("role vocabulary should appear before skill vocabulary (FR-009)")
	}
}

func TestAntiPatternMergeOrder_RoleBeforeSkill(t *testing.T) {
	t.Parallel()
	p := testPipeline()
	input := testInput()

	result, err := p.Run(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var antiSection *PipelineSection
	for i := range result.Sections {
		if result.Sections[i].Label == "Anti-Patterns" {
			antiSection = &result.Sections[i]
			break
		}
	}
	if antiSection == nil {
		t.Fatal("anti-patterns section should exist")
	}

	content := antiSection.Content
	// Role anti-patterns (structured) should appear before skill anti-patterns (plain).
	roleIdx := strings.Index(content, "over-engineering")
	skillIdx := strings.Index(content, "Skipping tests")
	if roleIdx < 0 {
		t.Fatal("role anti-pattern 'over-engineering' not found")
	}
	if skillIdx < 0 {
		t.Fatal("skill anti-pattern 'Skipping tests' not found")
	}
	if roleIdx > skillIdx {
		t.Error("role anti-patterns should appear before skill anti-patterns (FR-010)")
	}
}

// ─── Progressive Disclosure Layer Assignment ──────────────────────────────────

func TestLayerAssignment(t *testing.T) {
	t.Parallel()
	p := testPipeline()
	input := testInput()

	result, err := p.Run(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, s := range result.Sections {
		switch s.Label {
		case "Identity", "Role", "Orchestration", "Vocabulary":
			if s.Layer != LayerAlways {
				t.Errorf("section %q layer = %d, want %d (LayerAlways)", s.Label, s.Layer, LayerAlways)
			}
		case "Anti-Patterns", "Procedure", "Output Format and Examples",
			"Knowledge", "Evaluation Criteria", "Retrieval Anchors":
			if s.Layer != LayerTask {
				t.Errorf("section %q layer = %d, want %d (LayerTask)", s.Label, s.Layer, LayerTask)
			}
		}
	}
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

func TestEstimateTokens(t *testing.T) {
	t.Parallel()
	tests := []struct {
		input string
		want  int
	}{
		{"", 0},
		{"a", 1},
		{"abcd", 1},
		{"abcde", 2},
		{"abcdefgh", 2},
		{strings.Repeat("x", 100), 25},
	}
	for _, tt := range tests {
		got := estimateTokens(tt.input)
		if got != tt.want {
			t.Errorf("estimateTokens(%q) = %d, want %d", tt.input, got, tt.want)
		}
	}
}

func TestExtractSkillSection(t *testing.T) {
	t.Parallel()
	sk := testSkill()

	proc := extractSkillSection(sk, "Procedure")
	if !strings.Contains(proc, "Read the spec") {
		t.Errorf("procedure section missing expected content: %q", proc)
	}

	missing := extractSkillSection(sk, "Nonexistent")
	if missing != "" {
		t.Errorf("nonexistent section should return empty, got %q", missing)
	}

	// Case-insensitive match.
	vocab := extractSkillSection(sk, "vocabulary")
	if vocab == "" {
		t.Error("case-insensitive lookup should find Vocabulary section")
	}
}

func TestRenderPrompt(t *testing.T) {
	t.Parallel()
	result := &PipelineResult{
		Sections: []PipelineSection{
			{Position: 1, Label: "Identity", Content: "## Task: Test\n\nTest summary"},
			{Position: 2, Label: "Role", Content: "**Role:** tester\n\nTest role"},
		},
		TotalTokens: 50,
	}

	prompt := RenderPrompt(result)
	if !strings.Contains(prompt, "## Task: Test") {
		t.Error("prompt should contain identity section")
	}
	if !strings.Contains(prompt, "**Role:** tester") {
		t.Error("prompt should contain role section")
	}
}

func TestRenderPrompt_WithWarning(t *testing.T) {
	t.Parallel()
	result := &PipelineResult{
		Sections: []PipelineSection{
			{Position: 1, Label: "Identity", Content: "## Task: Test"},
		},
		TotalTokens:  90000,
		TokenWarning: "context size 90000 tokens exceeds 40% of 200000-token window",
	}

	prompt := RenderPrompt(result)
	if !strings.Contains(prompt, "⚠️") {
		t.Error("prompt should contain warning indicator")
	}
	if !strings.Contains(prompt, "90000") {
		t.Error("prompt should contain token count")
	}
}

func TestWindowTokensDefault(t *testing.T) {
	t.Parallel()
	p := &Pipeline{}
	if p.windowTokens() != DefaultContextWindowTokens {
		t.Errorf("default window = %d, want %d", p.windowTokens(), DefaultContextWindowTokens)
	}
}

func TestWindowTokensCustom(t *testing.T) {
	t.Parallel()
	p := &Pipeline{WindowSize: 50000}
	if p.windowTokens() != 50000 {
		t.Errorf("custom window = %d, want 50000", p.windowTokens())
	}
}

// ─── Adapter tests ────────────────────────────────────────────────────────────

func TestNoOpSurfacer(t *testing.T) {
	t.Parallel()
	s := NoOpSurfacer{}
	entries, err := s.Surface(SurfaceInput{TaskID: "TASK-001"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("expected empty entries, got %d", len(entries))
	}
}

func TestBindingFileAdapter_Success(t *testing.T) {
	t.Parallel()
	sb := testBinding()
	adapter := &BindingFileAdapter{
		File: &binding.BindingFile{
			StageBindings: map[string]*binding.StageBinding{
				"developing": sb,
			},
		},
	}

	got, err := adapter.Lookup("developing")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != sb {
		t.Error("should return the same binding pointer")
	}
}

func TestBindingFileAdapter_NotFound(t *testing.T) {
	t.Parallel()
	adapter := &BindingFileAdapter{
		File: &binding.BindingFile{
			StageBindings: map[string]*binding.StageBinding{},
		},
	}

	_, err := adapter.Lookup("designing")
	if err == nil {
		t.Fatal("expected error for missing stage")
	}
	if !strings.Contains(err.Error(), "designing") {
		t.Errorf("error should mention the stage: %v", err)
	}
}

func TestBindingFileAdapter_NilFile(t *testing.T) {
	t.Parallel()
	adapter := &BindingFileAdapter{File: nil}
	_, err := adapter.Lookup("developing")
	if err == nil {
		t.Fatal("expected error for nil binding file")
	}
}

// ─── Rendering helpers ────────────────────────────────────────────────────────

func TestRenderOrchestration(t *testing.T) {
	t.Parallel()
	meta := OrchestrationMeta{
		Pattern:         "orchestrator-workers",
		EffortBudget:    "5-8 points",
		MaxReviewCycles: 2,
		HumanGate:       true,
		Prerequisites:   []string{"spec must be approved"},
	}
	content := renderOrchestration(meta, "TASK-001")

	checks := []string{
		"orchestrator-workers",
		"5-8 points",
		"**Max review cycles:** 2",
		"**Human gate:** required before advancing",
		"spec must be approved",
		"feat(TASK-001)",
	}
	for _, c := range checks {
		if !strings.Contains(content, c) {
			t.Errorf("orchestration should contain %q: got %q", c, content)
		}
	}
}

func TestRenderAntiPatterns_Structured(t *testing.T) {
	t.Parallel()
	entries := []AntiPatternEntry{
		{
			Name:    "gold-plating",
			Detect:  "adding unrequested features",
			Because: "scope creep",
			Resolve: "stick to spec",
		},
	}
	content := renderAntiPatterns(entries)
	if !strings.Contains(content, "gold-plating") {
		t.Error("should render structured anti-pattern name")
	}
	if !strings.Contains(content, "Detect:") {
		t.Error("should render detect field")
	}
}

func TestRenderAntiPatterns_PlainText(t *testing.T) {
	t.Parallel()
	entries := []AntiPatternEntry{
		{Name: "Don't skip code review"},
	}
	content := renderAntiPatterns(entries)
	if !strings.Contains(content, "Don't skip code review") {
		t.Error("should render plain text anti-pattern")
	}
	if strings.Contains(content, "Detect:") {
		t.Error("should not render Detect for plain text entry")
	}
}

func TestSplitNonEmpty(t *testing.T) {
	t.Parallel()
	result := splitNonEmpty("  hello \n\n  world  \n  ")
	if len(result) != 2 {
		t.Fatalf("len = %d, want 2", len(result))
	}
	if result[0] != "hello" || result[1] != "world" {
		t.Errorf("got %v, want [hello world]", result)
	}
}

func TestJoinNonEmpty(t *testing.T) {
	t.Parallel()
	result := joinNonEmpty("hello", "", "world", "  ")
	if result != "hello\n\nworld" {
		t.Errorf("got %q, want %q", result, "hello\n\nworld")
	}
}
