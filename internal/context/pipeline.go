// Package context pipeline.go — 10-step context assembly pipeline for roles, skills, and bindings.
//
// The pipeline integrates the role system (role.go, role_resolve.go), skill system
// (internal/skill), and binding registry (internal/binding) into an attention-curve-ordered
// prompt. It is invoked by the handoff tool when a stage binding exists for the task's
// parent feature lifecycle stage.
//
// Design: work/design/skills-system-redesign-v2.md §3.4, §6.1, §6.2
// Spec: work/spec/3.0-context-assembly-pipeline.md
package context

import (
	"fmt"
	"strings"
	"time"

	"github.com/sambeau/kanbanzai/internal/binding"
	"github.com/sambeau/kanbanzai/internal/health"
	"github.com/sambeau/kanbanzai/internal/skill"
)

// ─── Pipeline position constants ──────────────────────────────────────────────

// Section positions in the attention-curve-ordered output (FR-016).
const (
	PositionIdentity          = 1
	PositionRoleIdentity      = 2
	PositionOrchestration     = 3
	PositionVocabulary        = 4
	PositionAntiPatterns      = 5
	PositionProcedure         = 6
	PositionOutputAndExamples = 7
	PositionKnowledge         = 8
	PositionEvalCriteria      = 9
	PositionRetrievalAnchors  = 10
)

// Progressive disclosure layer assignments (FR-018).
const (
	LayerAlways     = 1 // ~300–500 tokens: identity, constraints, vocabulary
	LayerTask       = 2 // ~500–2000 tokens: procedure, anti-patterns, output format, examples
	LayerOnDemand   = 3 // 2000+ tokens: full spec sections, reference documents
	LayerCompressed = 4 // variable: summaries of large documents
)

// Token budget thresholds relative to context window (FR-014, FR-015).
const (
	DefaultContextWindowTokens = 200_000
	BudgetWarnRatio            = 0.40
	BudgetRefuseRatio          = 0.60
)

// ─── Dependency interfaces (NFR-005: testability) ─────────────────────────────

// RoleResolver resolves a role by ID, walking the inheritance chain.
// The production implementation wraps RoleStore + ResolveRole.
type RoleResolver interface {
	Resolve(id string) (*ResolvedRole, error)
}

// SkillResolver loads a parsed skill by name.
// The production implementation wraps skill.SkillStore.
type SkillResolver interface {
	Load(name string) (*skill.Skill, error)
}

// BindingResolver looks up a stage binding from the binding registry.
// The production implementation wraps binding.BindingFile.
type BindingResolver interface {
	Lookup(stage string) (*binding.StageBinding, error)
}

// KnowledgeSurfacer retrieves relevant knowledge entries for a task context.
// Step 7 of the pipeline delegates to this interface.
type KnowledgeSurfacer interface {
	Surface(input SurfaceInput) ([]SurfacedEntry, error)
}

// SurfaceInput is the input for knowledge auto-surfacing.
type SurfaceInput struct {
	TaskID    string
	FilePaths []string
	RoleTags  []string
	SkillName string
}

// SurfacedEntry is a single knowledge entry selected for inclusion.
type SurfacedEntry struct {
	ID      string
	Topic   string
	Content string
	Score   float64
}

// ─── Pipeline data types ──────────────────────────────────────────────────────

// PipelineInput holds the parameters for a pipeline run.
type PipelineInput struct {
	TaskID       string
	TaskState    map[string]any
	FeatureState map[string]any
	Role         string // optional role override from the caller
	Instructions string // optional orchestrator instructions
}

// PipelineSection represents one section in the attention-curve-ordered output.
type PipelineSection struct {
	Position int    // 1–10 per the attention-curve table
	Label    string // section heading (e.g. "Identity", "Vocabulary")
	Content  string // rendered Markdown content
	Layer    int    // progressive disclosure layer (1–4)
	Tokens   int    // estimated token count for this section
}

// PipelineResult is the output of the 10-step pipeline.
type PipelineResult struct {
	Sections         []PipelineSection // ordered by Position
	TotalTokens      int
	TokenWarning     string   // non-empty when > 40% of context window
	MetadataWarnings []string // e.g. staleness warnings (for Freshness Tracking feature)
}

// OrchestrationMeta holds extracted orchestration metadata from a stage binding (step 4).
type OrchestrationMeta struct {
	Pattern         string // e.g. "single-agent", "orchestrator-workers"
	EffortBudget    string
	Prerequisites   []string
	MaxReviewCycles int
	HumanGate       bool
}

// InclusionStrategy holds the stage-specific include/exclude configuration (step 3).
type InclusionStrategy struct {
	// IncludeSpec controls whether full spec sections are included.
	IncludeSpec bool
	// IncludeKnowledge controls whether knowledge entries are included.
	IncludeKnowledge bool
	// IncludeExamples controls whether skill examples are included.
	IncludeExamples bool
	// IncludeReferences controls whether reference documents are included.
	IncludeReferences bool
}

// PipelineState accumulates results as the pipeline steps execute.
type PipelineState struct {
	Input             PipelineInput
	Stage             string                // from step 1
	Binding           *binding.StageBinding // from step 2
	Inclusion         InclusionStrategy     // from step 3
	Orchestration     OrchestrationMeta     // from step 4
	Role              *ResolvedRole         // from step 5
	Skill             *skill.Skill          // from step 6
	MergedVocab       []string              // from steps 5+6
	MergedAnti        []AntiPatternEntry    // from steps 5+6
	Knowledge         []SurfacedEntry       // from step 7
	ToolGuidance      string                // from step 8
	Sections          []PipelineSection     // accumulated by step 10
	TokenEstimate     int                   // from step 9
	StalenessWarnings []string              // from freshness check after steps 5+6
}

// AntiPatternEntry is a flattened anti-pattern for rendering.
// Combines the structured AntiPattern from roles with plain-text anti-patterns from skills.
type AntiPatternEntry struct {
	Name    string
	Detect  string
	Because string
	Resolve string
}

// ─── Pipeline orchestrator ────────────────────────────────────────────────────

// Pipeline orchestrates the 10-step context assembly.
type Pipeline struct {
	Roles               RoleResolver
	Skills              SkillResolver
	Bindings            BindingResolver
	Knowledge           KnowledgeSurfacer
	WindowSize          int // context window in tokens; 0 means DefaultContextWindowTokens
	StalenessWindowDays int // 0 means 30 (default)
}

// stalenessWindow returns the effective staleness window in days.
func (p *Pipeline) stalenessWindow() int {
	if p.StalenessWindowDays > 0 {
		return p.StalenessWindowDays
	}
	return 30
}

// windowTokens returns the effective context window size.
func (p *Pipeline) windowTokens() int {
	if p.WindowSize > 0 {
		return p.WindowSize
	}
	return DefaultContextWindowTokens
}

// Run executes the 10-step pipeline and returns the assembled result.
// Each step either advances the PipelineState or returns an error with
// step name, entity ID, and remediation hint (NFR-004).
func (p *Pipeline) Run(input PipelineInput) (*PipelineResult, error) {
	state := &PipelineState{Input: input}

	// Step 0: Lifecycle state validation.
	if err := p.stepValidateLifecycle(state); err != nil {
		return nil, err
	}

	// Step 1: Task-to-stage resolution.
	if err := p.stepResolveStage(state); err != nil {
		return nil, err
	}

	// Step 2: Stage binding lookup.
	if err := p.stepLookupBinding(state); err != nil {
		return nil, err
	}

	// Step 3: Stage-specific inclusion/exclusion.
	p.stepApplyInclusion(state)

	// Step 4: Orchestration metadata extraction.
	p.stepExtractOrchestration(state)

	// Step 5: Role resolution with inheritance.
	if err := p.stepResolveRole(state); err != nil {
		return nil, err
	}

	// Step 6: Skill loading.
	if err := p.stepLoadSkill(state); err != nil {
		return nil, err
	}

	// Freshness check: warn if role or skill is stale or never-verified.
	p.stepCheckFreshness(state)

	// Step 7: Knowledge entry integration.
	p.stepSurfaceKnowledge(state)

	// Step 8: Tool subset guidance.
	p.stepToolGuidance(state)

	// Step 9: Token budget estimation.
	// (Populates state.Sections first, then estimates.)
	p.stepAssembleSections(state)
	if err := p.stepTokenBudget(state); err != nil {
		return nil, err
	}

	// Step 10: Build the final result with attention-curve ordering.
	return p.stepBuildResult(state), nil
}

// ─── Pipeline step implementations ───────────────────────────────────────────

// stepValidateLifecycle checks that the parent feature is in a workable state (step 0).
func (p *Pipeline) stepValidateLifecycle(state *PipelineState) error {
	if state.Input.FeatureState == nil {
		return pipelineError(0, "lifecycle-validation",
			fmt.Sprintf("task %s has no parent feature state", state.Input.TaskID),
			"ensure the task has a parent_feature that exists")
	}

	status, _ := state.Input.FeatureState["status"].(string)
	if !isWorkableFeatureStatus(status) {
		return pipelineError(0, "lifecycle-validation",
			fmt.Sprintf("feature is in status %q; pipeline requires one of: %s",
				status, strings.Join(workableStatuses, ", ")),
			"advance the feature to a workable status before generating context")
	}
	return nil
}

// stepResolveStage resolves the task's parent feature lifecycle stage (step 1).
func (p *Pipeline) stepResolveStage(state *PipelineState) error {
	status, _ := state.Input.FeatureState["status"].(string)
	if status == "" {
		return pipelineError(1, "stage-resolution",
			fmt.Sprintf("task %s: parent feature has no status", state.Input.TaskID),
			"ensure the parent feature has a valid lifecycle status")
	}
	state.Stage = status
	return nil
}

// stepLookupBinding retrieves the stage binding for the resolved stage (step 2).
func (p *Pipeline) stepLookupBinding(state *PipelineState) error {
	if p.Bindings == nil {
		return pipelineError(2, "binding-lookup",
			fmt.Sprintf("no binding resolver configured for stage %q", state.Stage),
			"ensure a stage-bindings.yaml exists and the pipeline is configured with a BindingResolver")
	}

	b, err := p.Bindings.Lookup(state.Stage)
	if err != nil {
		return pipelineError(2, "binding-lookup",
			fmt.Sprintf("no binding configured for stage %q: %v", state.Stage, err),
			"add a binding for this stage in stage-bindings.yaml")
	}
	state.Binding = b
	return nil
}

// stepApplyInclusion derives what content categories to include/exclude (step 3).
func (p *Pipeline) stepApplyInclusion(state *PipelineState) {
	// Default: include everything.
	state.Inclusion = InclusionStrategy{
		IncludeSpec:       true,
		IncludeKnowledge:  true,
		IncludeExamples:   true,
		IncludeReferences: true,
	}

	// The binding can restrict categories. For now, the binding model doesn't
	// have explicit include/exclude fields — we derive from the stage semantics.
	// Reviewing stages exclude full spec (reviewers read it separately).
	// This is extensible when the binding model gains explicit category fields.
	switch state.Stage {
	case "reviewing", "plan-reviewing":
		state.Inclusion.IncludeReferences = false
	}
}

// stepExtractOrchestration extracts orchestration metadata from the binding (step 4).
func (p *Pipeline) stepExtractOrchestration(state *PipelineState) {
	b := state.Binding
	state.Orchestration = OrchestrationMeta{
		Pattern:      b.Orchestration,
		EffortBudget: b.EffortBudget,
		HumanGate:    b.HumanGate,
	}
	if b.MaxReviewCycles != nil {
		state.Orchestration.MaxReviewCycles = *b.MaxReviewCycles
	}
	if b.Prerequisites != nil {
		for _, dp := range b.Prerequisites.Documents {
			state.Orchestration.Prerequisites = append(state.Orchestration.Prerequisites,
				fmt.Sprintf("%s must be %s", dp.Type, dp.Status))
		}
	}
}

// stepResolveRole resolves the role (step 5) and merges vocabulary/anti-patterns.
func (p *Pipeline) stepResolveRole(state *PipelineState) error {
	// Determine which role to resolve: caller override > binding's first role.
	roleID := state.Input.Role
	if roleID == "" && len(state.Binding.Roles) > 0 {
		roleID = state.Binding.Roles[0]
	}
	if roleID == "" {
		return pipelineError(5, "role-resolution",
			fmt.Sprintf("no role specified for stage %q", state.Stage),
			"specify a role in the binding or pass a role parameter")
	}

	if p.Roles == nil {
		return pipelineError(5, "role-resolution",
			fmt.Sprintf("no role resolver configured; cannot resolve role %q", roleID),
			"ensure the pipeline is configured with a RoleResolver")
	}

	resolved, err := p.Roles.Resolve(roleID)
	if err != nil {
		return pipelineError(5, "role-resolution",
			fmt.Sprintf("failed to resolve role %q: %v", roleID, err),
			"check that the role file exists in .kbz/roles/")
	}
	state.Role = resolved

	// Start building merged vocabulary with role vocabulary.
	state.MergedVocab = append(state.MergedVocab, resolved.Vocabulary...)

	// Start building merged anti-patterns from role.
	for _, ap := range resolved.AntiPatterns {
		state.MergedAnti = append(state.MergedAnti, AntiPatternEntry{
			Name:    ap.Name,
			Detect:  ap.Detect,
			Because: ap.Because,
			Resolve: ap.Resolve,
		})
	}

	return nil
}

// stepLoadSkill loads the skill and appends its vocabulary/anti-patterns (step 6).
func (p *Pipeline) stepLoadSkill(state *PipelineState) error {
	skillName := ""
	if len(state.Binding.Skills) > 0 {
		skillName = state.Binding.Skills[0]
	}
	if skillName == "" {
		return pipelineError(6, "skill-loading",
			fmt.Sprintf("no skill specified for stage %q", state.Stage),
			"add a skill to the binding for this stage in stage-bindings.yaml")
	}

	if p.Skills == nil {
		return pipelineError(6, "skill-loading",
			fmt.Sprintf("no skill resolver configured; cannot load skill %q", skillName),
			"ensure the pipeline is configured with a SkillResolver")
	}

	sk, err := p.Skills.Load(skillName)
	if err != nil {
		return pipelineError(6, "skill-loading",
			fmt.Sprintf("skill %q not found: %v", skillName, err),
			"check that the skill directory exists in .kbz/skills/")
	}
	state.Skill = sk

	// Append skill vocabulary after role vocabulary (FR-009).
	skillVocab := extractSkillSection(sk, "Vocabulary")
	if skillVocab != "" {
		for _, line := range splitNonEmpty(skillVocab) {
			state.MergedVocab = append(state.MergedVocab, strings.TrimPrefix(line, "- "))
		}
	}

	// Append skill anti-patterns after role anti-patterns (FR-010).
	skillAnti := extractSkillSection(sk, "Anti-Patterns")
	if skillAnti != "" {
		for _, line := range splitNonEmpty(skillAnti) {
			trimmed := strings.TrimPrefix(line, "- ")
			state.MergedAnti = append(state.MergedAnti, AntiPatternEntry{
				Name: trimmed,
			})
		}
	}

	return nil
}

// stepSurfaceKnowledge invokes the knowledge surfacer at step 7.
func (p *Pipeline) stepSurfaceKnowledge(state *PipelineState) {
	if p.Knowledge == nil || !state.Inclusion.IncludeKnowledge {
		return
	}

	var filePaths []string
	if paths, ok := state.Input.TaskState["files_modified"].([]any); ok {
		for _, fp := range paths {
			if s, ok := fp.(string); ok {
				filePaths = append(filePaths, s)
			}
		}
	}

	var roleTags []string
	if state.Role != nil {
		roleTags = append(roleTags, state.Role.ID)
	}

	skillName := ""
	if state.Skill != nil {
		skillName = state.Skill.Frontmatter.Name
	}

	entries, err := p.Knowledge.Surface(SurfaceInput{
		TaskID:    state.Input.TaskID,
		FilePaths: filePaths,
		RoleTags:  roleTags,
		SkillName: skillName,
	})
	if err != nil {
		// Knowledge surfacing is best-effort; failures don't block assembly.
		return
	}
	state.Knowledge = entries
}

// stepToolGuidance generates tool subset guidance text (step 8).
func (p *Pipeline) stepToolGuidance(state *PipelineState) {
	if state.Role == nil || len(state.Role.Tools) == 0 {
		return
	}
	state.ToolGuidance = fmt.Sprintf("**Preferred tools for this role:** %s",
		strings.Join(state.Role.Tools, ", "))
}

// stepAssembleSections builds the ordered PipelineSection slice (part of step 9/10).
func (p *Pipeline) stepAssembleSections(state *PipelineState) {
	// Position 1: Project identity and hard constraints (Layer 1).
	taskID, _ := state.Input.TaskState["id"].(string)
	summary, _ := state.Input.TaskState["summary"].(string)
	identityContent := fmt.Sprintf("## Task: %s\n\n%s", taskID, summary)
	if state.Input.Instructions != "" {
		identityContent += "\n\n### Additional Instructions\n\n" + state.Input.Instructions
	}
	state.Sections = append(state.Sections, PipelineSection{
		Position: PositionIdentity,
		Label:    "Identity",
		Content:  identityContent,
		Layer:    LayerAlways,
		Tokens:   estimateTokens(identityContent),
	})

	// Position 2: Role identity (Layer 1).
	if state.Role != nil {
		roleContent := fmt.Sprintf("**Role:** %s\n\n%s", state.Role.ID, state.Role.Identity)
		if state.ToolGuidance != "" {
			roleContent += "\n\n" + state.ToolGuidance
		}
		state.Sections = append(state.Sections, PipelineSection{
			Position: PositionRoleIdentity,
			Label:    "Role",
			Content:  roleContent,
			Layer:    LayerAlways,
			Tokens:   estimateTokens(roleContent),
		})
	}

	// Position 3: Orchestration metadata (Layer 1).
	orchContent := renderOrchestration(state.Orchestration, taskID)
	if orchContent != "" {
		state.Sections = append(state.Sections, PipelineSection{
			Position: PositionOrchestration,
			Label:    "Orchestration",
			Content:  orchContent,
			Layer:    LayerAlways,
			Tokens:   estimateTokens(orchContent),
		})
	}

	// Position 4: Combined vocabulary (Layer 1).
	if len(state.MergedVocab) > 0 {
		vocabContent := renderList("Vocabulary", state.MergedVocab)
		state.Sections = append(state.Sections, PipelineSection{
			Position: PositionVocabulary,
			Label:    "Vocabulary",
			Content:  vocabContent,
			Layer:    LayerAlways,
			Tokens:   estimateTokens(vocabContent),
		})
	}

	// Position 5: Combined anti-patterns (Layer 2).
	if len(state.MergedAnti) > 0 {
		antiContent := renderAntiPatterns(state.MergedAnti)
		state.Sections = append(state.Sections, PipelineSection{
			Position: PositionAntiPatterns,
			Label:    "Anti-Patterns",
			Content:  antiContent,
			Layer:    LayerTask,
			Tokens:   estimateTokens(antiContent),
		})
	}

	// Position 6: Skill procedure (Layer 2).
	if state.Skill != nil {
		proc := extractSkillSection(state.Skill, "Procedure")
		if proc == "" {
			proc = extractSkillSection(state.Skill, "Checklist")
		}
		if proc != "" {
			state.Sections = append(state.Sections, PipelineSection{
				Position: PositionProcedure,
				Label:    "Procedure",
				Content:  proc,
				Layer:    LayerTask,
				Tokens:   estimateTokens(proc),
			})
		}
	}

	// Position 7: Output format and examples (Layer 2).
	if state.Skill != nil && state.Inclusion.IncludeExamples {
		outputFmt := extractSkillSection(state.Skill, "Output Format")
		examples := extractSkillSection(state.Skill, "Examples")
		combined := joinNonEmpty(outputFmt, examples)
		if combined != "" {
			state.Sections = append(state.Sections, PipelineSection{
				Position: PositionOutputAndExamples,
				Label:    "Output Format and Examples",
				Content:  combined,
				Layer:    LayerTask,
				Tokens:   estimateTokens(combined),
			})
		}
	}

	// Position 8: Knowledge entries (Layer 2).
	if len(state.Knowledge) > 0 {
		var lines []string
		for _, ke := range state.Knowledge {
			lines = append(lines, fmt.Sprintf("- **%s:** %s", ke.Topic, ke.Content))
		}
		knowledgeContent := strings.Join(lines, "\n")
		state.Sections = append(state.Sections, PipelineSection{
			Position: PositionKnowledge,
			Label:    "Knowledge",
			Content:  knowledgeContent,
			Layer:    LayerTask,
			Tokens:   estimateTokens(knowledgeContent),
		})
	}

	// Position 9: Evaluation criteria (Layer 2).
	if state.Skill != nil {
		evalContent := extractSkillSection(state.Skill, "Evaluation Criteria")
		if evalContent != "" {
			state.Sections = append(state.Sections, PipelineSection{
				Position: PositionEvalCriteria,
				Label:    "Evaluation Criteria",
				Content:  evalContent,
				Layer:    LayerTask,
				Tokens:   estimateTokens(evalContent),
			})
		}
	}

	// Position 10: Retrieval anchors (Layer 2).
	if state.Skill != nil {
		anchors := extractSkillSection(state.Skill, "Questions This Skill Answers")
		if anchors != "" {
			state.Sections = append(state.Sections, PipelineSection{
				Position: PositionRetrievalAnchors,
				Label:    "Retrieval Anchors",
				Content:  anchors,
				Layer:    LayerTask,
				Tokens:   estimateTokens(anchors),
			})
		}
	}
}

// stepTokenBudget estimates total tokens and enforces budget thresholds (step 9).
func (p *Pipeline) stepTokenBudget(state *PipelineState) error {
	total := 0
	for _, s := range state.Sections {
		total += s.Tokens
	}
	state.TokenEstimate = total

	windowSize := p.windowTokens()
	refuseThreshold := int(float64(windowSize) * BudgetRefuseRatio)
	if total > refuseThreshold {
		return pipelineError(9, "token-budget",
			fmt.Sprintf("assembled context is %d tokens, exceeding 60%% of %d-token window (%d tokens)",
				total, windowSize, refuseThreshold),
			"split the work unit into smaller tasks to reduce context size")
	}

	return nil
}

// stepCheckFreshness checks whether the resolved role and skill are stale
// or never-verified, and populates StalenessWarnings on the state.
// Stale content is NOT blocked — assembly proceeds with warnings (FR-009).
func (p *Pipeline) stepCheckFreshness(state *PipelineState) {
	window := p.stalenessWindow()
	now := time.Now()

	if state.Role != nil {
		lv := state.Role.LastVerified
		lvTime, isZero := parseLastVerified(lv)
		detail := health.ClassifyFreshness(lvTime, isZero, window, now)
		switch detail.Status {
		case health.StatusStale:
			state.StalenessWarnings = append(state.StalenessWarnings,
				fmt.Sprintf("role %q last verified %s (%d days overdue)",
					state.Role.ID, lv, detail.DaysOverdue))
		case health.StatusNeverVerified:
			state.StalenessWarnings = append(state.StalenessWarnings,
				fmt.Sprintf("role %q has never been verified", state.Role.ID))
		}
	}

	if state.Skill != nil {
		lv := state.Skill.Frontmatter.LastVerified
		lvTime, isZero := parseLastVerified(lv)
		detail := health.ClassifyFreshness(lvTime, isZero, window, now)
		switch detail.Status {
		case health.StatusStale:
			state.StalenessWarnings = append(state.StalenessWarnings,
				fmt.Sprintf("skill %q last verified %s (%d days overdue)",
					state.Skill.Frontmatter.Name, lv, detail.DaysOverdue))
		case health.StatusNeverVerified:
			state.StalenessWarnings = append(state.StalenessWarnings,
				fmt.Sprintf("skill %q has never been verified", state.Skill.Frontmatter.Name))
		}
	}
}

// parseLastVerified parses a last_verified string into a time.
// Returns (zero time, true) if the string is empty (never verified).
func parseLastVerified(lv string) (time.Time, bool) {
	if lv == "" {
		return time.Time{}, true
	}
	t, err := time.Parse(time.RFC3339, lv)
	if err != nil {
		return time.Time{}, true // treat unparseable as never-verified
	}
	return t, false
}

// stepBuildResult produces the final PipelineResult (step 10).
func (p *Pipeline) stepBuildResult(state *PipelineState) *PipelineResult {
	result := &PipelineResult{
		Sections:         state.Sections, // already ordered by Position during assembly
		TotalTokens:      state.TokenEstimate,
		MetadataWarnings: state.StalenessWarnings,
	}

	// Check warning threshold.
	windowSize := p.windowTokens()
	warnThreshold := int(float64(windowSize) * BudgetWarnRatio)
	if state.TokenEstimate > warnThreshold {
		result.TokenWarning = fmt.Sprintf(
			"context size %d tokens exceeds 40%% of %d-token window (%d tokens)",
			state.TokenEstimate, windowSize, warnThreshold)
	}

	return result
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

var workableStatuses = []string{
	"designing", "specifying", "dev-planning",
	"developing", "reviewing", "plan-reviewing",
	"researching", "documenting",
}

func isWorkableFeatureStatus(status string) bool {
	for _, s := range workableStatuses {
		if s == status {
			return true
		}
	}
	return false
}

// pipelineError creates a formatted error with step, entity, and remediation hint (NFR-004).
func pipelineError(step int, stepName, detail, hint string) error {
	return fmt.Errorf("pipeline step %d (%s): %s. Hint: %s", step, stepName, detail, hint)
}

// estimateTokens approximates token count from text (chars / 4, ±10% acceptable per spec §6).
func estimateTokens(text string) int {
	if len(text) == 0 {
		return 0
	}
	return (len(text) + 3) / 4 // ceiling division
}

// extractSkillSection finds a section by heading from a loaded skill.
func extractSkillSection(sk *skill.Skill, heading string) string {
	lower := strings.ToLower(heading)
	for _, s := range sk.Sections {
		if strings.ToLower(s.Heading) == lower {
			return strings.TrimSpace(s.Content)
		}
	}
	return ""
}

// splitNonEmpty splits text by newlines and returns non-empty, trimmed lines.
func splitNonEmpty(text string) []string {
	var result []string
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			result = append(result, line)
		}
	}
	return result
}

// joinNonEmpty concatenates non-empty strings with double newlines.
func joinNonEmpty(parts ...string) string {
	var nonEmpty []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			nonEmpty = append(nonEmpty, p)
		}
	}
	return strings.Join(nonEmpty, "\n\n")
}

// renderOrchestration formats orchestration metadata as Markdown.
func renderOrchestration(meta OrchestrationMeta, taskID string) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "**Orchestration:** %s", meta.Pattern)
	if meta.EffortBudget != "" {
		fmt.Fprintf(&sb, "\n**Effort budget:** %s", meta.EffortBudget)
	}
	if meta.MaxReviewCycles > 0 {
		fmt.Fprintf(&sb, "\n**Max review cycles:** %d", meta.MaxReviewCycles)
	}
	if meta.HumanGate {
		sb.WriteString("\n**Human gate:** required before advancing")
	}
	if len(meta.Prerequisites) > 0 {
		sb.WriteString("\n**Prerequisites:** ")
		sb.WriteString(strings.Join(meta.Prerequisites, "; "))
	}
	if taskID != "" {
		fmt.Fprintf(&sb, "\n**Commit format:** feat(%s): <description>", taskID)
	}
	return sb.String()
}

// renderList formats a string slice as a Markdown bulleted list under a heading.
func renderList(heading string, items []string) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "### %s\n\n", heading)
	for _, item := range items {
		fmt.Fprintf(&sb, "- %s\n", item)
	}
	return sb.String()
}

// renderAntiPatterns formats anti-pattern entries as Markdown.
func renderAntiPatterns(entries []AntiPatternEntry) string {
	var sb strings.Builder
	sb.WriteString("### Anti-Patterns\n\n")
	for _, ap := range entries {
		if ap.Detect != "" {
			// Structured anti-pattern from role.
			fmt.Fprintf(&sb, "- **%s**: Detect: %s. Because: %s. Resolve: %s.\n",
				ap.Name, ap.Detect, ap.Because, ap.Resolve)
		} else {
			// Plain-text anti-pattern from skill.
			fmt.Fprintf(&sb, "- %s\n", ap.Name)
		}
	}
	return sb.String()
}

// RenderPrompt renders the pipeline result as a single Markdown prompt string.
// This is the output format consumed by the handoff tool.
func RenderPrompt(result *PipelineResult) string {
	var sb strings.Builder
	for _, s := range result.Sections {
		sb.WriteString(s.Content)
		sb.WriteString("\n\n")
	}
	if result.TokenWarning != "" {
		fmt.Fprintf(&sb, "---\n⚠️ %s\n\n", result.TokenWarning)
	}
	return strings.TrimSpace(sb.String())
}
