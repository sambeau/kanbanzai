package stage

import (
	"testing"
)

func TestAllStagesHaveConfig(t *testing.T) {
	for _, s := range AllStages() {
		cfg, ok := ForStage(string(s))
		if !ok {
			t.Errorf("AllStages includes %q but ForStage returned false", s)
		}
		if cfg.Orchestration == "" {
			t.Errorf("stage %q has empty Orchestration", s)
		}
		if cfg.EffortBudget.Text == "" {
			t.Errorf("stage %q has empty EffortBudget.Text", s)
		}
		if cfg.EffortBudget.Warning == "" {
			t.Errorf("stage %q has empty EffortBudget.Warning", s)
		}
		if len(cfg.PrimaryTools) == 0 {
			t.Errorf("stage %q has no PrimaryTools", s)
		}
		if cfg.SpecMode == "" {
			t.Errorf("stage %q has empty SpecMode", s)
		}
	}
}

func TestIsWorkingState(t *testing.T) {
	workingStates := []string{
		"designing", "specifying", "dev-planning",
		"developing", "reviewing", "needs-rework",
	}
	for _, s := range workingStates {
		if !IsWorkingState(s) {
			t.Errorf("IsWorkingState(%q) = false, want true", s)
		}
	}

	nonWorkingStates := []string{
		"proposed", "done", "superseded", "cancelled",
	}
	for _, s := range nonWorkingStates {
		if IsWorkingState(s) {
			t.Errorf("IsWorkingState(%q) = true, want false", s)
		}
	}
}

func TestForStageNotFound(t *testing.T) {
	_, ok := ForStage("proposed")
	if ok {
		t.Error("ForStage(proposed) should return false")
	}
	_, ok = ForStage("done")
	if ok {
		t.Error("ForStage(done) should return false")
	}
	_, ok = ForStage("superseded")
	if ok {
		t.Error("ForStage(superseded) should return false")
	}
	_, ok = ForStage("cancelled")
	if ok {
		t.Error("ForStage(cancelled) should return false")
	}
	_, ok = ForStage("")
	if ok {
		t.Error("ForStage(empty) should return false")
	}
}

func TestOrchestrationPatterns(t *testing.T) {
	singleAgentStages := []string{"designing", "specifying", "dev-planning"}
	for _, s := range singleAgentStages {
		cfg, ok := ForStage(s)
		if !ok {
			t.Fatalf("ForStage(%q) returned false", s)
		}
		if cfg.Orchestration != SingleAgent {
			t.Errorf("stage %q: Orchestration = %q, want %q", s, cfg.Orchestration, SingleAgent)
		}
	}

	orchestratorStages := []string{"developing", "reviewing", "needs-rework"}
	for _, s := range orchestratorStages {
		cfg, ok := ForStage(s)
		if !ok {
			t.Fatalf("ForStage(%q) returned false", s)
		}
		if cfg.Orchestration != OrchestratorWorkers {
			t.Errorf("stage %q: Orchestration = %q, want %q", s, cfg.Orchestration, OrchestratorWorkers)
		}
	}
}

func TestEffortBudgets(t *testing.T) {
	tests := []struct {
		stage   string
		wantSub string
	}{
		{"designing", "5\u201315 tool calls"},
		{"specifying", "5\u201315 tool calls"},
		{"dev-planning", "5\u201310 tool calls"},
		{"developing", "10\u201350 tool calls per task"},
		{"reviewing", "5\u201310 tool calls per review dimension"},
		{"needs-rework", "10\u201350 tool calls per task"},
	}
	for _, tt := range tests {
		cfg, ok := ForStage(tt.stage)
		if !ok {
			t.Fatalf("ForStage(%q) returned false", tt.stage)
		}
		if !contains(cfg.EffortBudget.Text, tt.wantSub) {
			t.Errorf("stage %q: EffortBudget.Text = %q, want substring %q", tt.stage, cfg.EffortBudget.Text, tt.wantSub)
		}
	}
}

func TestEffortBudgetWarnings(t *testing.T) {
	// Sequential stages use "skip to implementation" warning.
	sequentialStages := []string{"designing", "specifying", "dev-planning", "reviewing"}
	for _, s := range sequentialStages {
		cfg, _ := ForStage(s)
		if !contains(cfg.EffortBudget.Warning, "skip to implementation") {
			t.Errorf("stage %q: Warning = %q, want 'skip to implementation' substring", s, cfg.EffortBudget.Warning)
		}
	}

	// Implementation stages use "skip testing" warning.
	implStages := []string{"developing", "needs-rework"}
	for _, s := range implStages {
		cfg, _ := ForStage(s)
		if !contains(cfg.EffortBudget.Warning, "skip testing") {
			t.Errorf("stage %q: Warning = %q, want 'skip testing' substring", s, cfg.EffortBudget.Warning)
		}
	}
}

func TestToolSubsets(t *testing.T) {
	tests := []struct {
		stage        string
		wantPrimary  []string
		wantExcluded []string
	}{
		{
			stage:        "designing",
			wantPrimary:  []string{"entity", "doc", "doc_intel", "knowledge", "status"},
			wantExcluded: []string{"decompose", "merge", "pr", "worktree", "finish"},
		},
		{
			stage:        "developing",
			wantPrimary:  []string{"entity", "handoff", "next", "finish", "knowledge", "status", "branch", "worktree", "write_file"},
			wantExcluded: []string{"decompose", "doc_intel"},
		},
		{
			stage:        "reviewing",
			wantPrimary:  []string{"entity", "doc", "doc_intel", "knowledge", "finish", "status"},
			wantExcluded: []string{"decompose", "merge", "worktree", "handoff"},
		},
	}
	for _, tt := range tests {
		cfg, ok := ForStage(tt.stage)
		if !ok {
			t.Fatalf("ForStage(%q) returned false", tt.stage)
		}
		if !sliceEqual(cfg.PrimaryTools, tt.wantPrimary) {
			t.Errorf("stage %q: PrimaryTools = %v, want %v", tt.stage, cfg.PrimaryTools, tt.wantPrimary)
		}
		if !sliceEqual(cfg.ExcludedTools, tt.wantExcluded) {
			t.Errorf("stage %q: ExcludedTools = %v, want %v", tt.stage, cfg.ExcludedTools, tt.wantExcluded)
		}
	}
}

func TestOutputConvention(t *testing.T) {
	// OutputConvention true only for orchestrator-workers stages.
	wantTrue := []string{"developing", "reviewing", "needs-rework"}
	for _, s := range wantTrue {
		cfg, _ := ForStage(s)
		if !cfg.OutputConvention {
			t.Errorf("stage %q: OutputConvention = false, want true", s)
		}
	}

	wantFalse := []string{"designing", "specifying", "dev-planning"}
	for _, s := range wantFalse {
		cfg, _ := ForStage(s)
		if cfg.OutputConvention {
			t.Errorf("stage %q: OutputConvention = true, want false", s)
		}
	}
}

func TestContentInclusion(t *testing.T) {
	// Designing and specifying should not include file paths.
	for _, s := range []string{"designing", "specifying"} {
		cfg, _ := ForStage(s)
		if cfg.IncludeFilePaths {
			t.Errorf("stage %q: IncludeFilePaths = true, want false", s)
		}
		if cfg.IncludeImplGuidance {
			t.Errorf("stage %q: IncludeImplGuidance = true, want false", s)
		}
	}

	// Developing should include file paths and test expectations.
	cfg, _ := ForStage("developing")
	if !cfg.IncludeFilePaths {
		t.Error("developing: IncludeFilePaths = false, want true")
	}
	if !cfg.IncludeTestExpect {
		t.Error("developing: IncludeTestExpect = false, want true")
	}
	if cfg.IncludeReviewRubric {
		t.Error("developing: IncludeReviewRubric = true, want false")
	}

	// Reviewing should include review rubric but not impl guidance.
	cfg, _ = ForStage("reviewing")
	if !cfg.IncludeReviewRubric {
		t.Error("reviewing: IncludeReviewRubric = false, want true")
	}
	if cfg.IncludeImplGuidance {
		t.Error("reviewing: IncludeImplGuidance = true, want false")
	}
}

func TestSpecMode(t *testing.T) {
	fullStages := []string{"designing", "specifying", "dev-planning"}
	for _, s := range fullStages {
		cfg, _ := ForStage(s)
		if cfg.SpecMode != "full" {
			t.Errorf("stage %q: SpecMode = %q, want %q", s, cfg.SpecMode, "full")
		}
	}

	relevantStages := []string{"developing", "reviewing", "needs-rework"}
	for _, s := range relevantStages {
		cfg, _ := ForStage(s)
		if cfg.SpecMode != "relevant-sections" {
			t.Errorf("stage %q: SpecMode = %q, want %q", s, cfg.SpecMode, "relevant-sections")
		}
	}
}

func TestAllStagesCount(t *testing.T) {
	stages := AllStages()
	if len(stages) != 6 {
		t.Errorf("AllStages() returned %d stages, want 6", len(stages))
	}
}

func TestReviewingDoesNotIncludeHandoffAsPrimary(t *testing.T) {
	cfg, _ := ForStage("reviewing")
	for _, tool := range cfg.PrimaryTools {
		if tool == "handoff" {
			t.Error("reviewing: PrimaryTools should not include handoff")
		}
	}
}

// helpers

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchSubstring(s, substr)
}

func searchSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func sliceEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
