package mcp

import (
	"strings"
	"testing"

	"github.com/sambeau/kanbanzai/internal/stage"
)

func TestStageIntegration_DevelopingHandoff(t *testing.T) {
	taskState := map[string]any{
		"id":            "TASK-INTEG01",
		"summary":       "Implement feature X",
		"files_planned": []any{"internal/foo.go", "internal/bar.go"},
	}

	actx := assembleContext(asmInput{
		taskState:    taskState,
		featureStage: "developing",
	})

	// Stage-aware fields populated.
	if !actx.stageAware {
		t.Fatal("expected stageAware = true")
	}
	if actx.featureStage != "developing" {
		t.Errorf("featureStage = %q, want developing", actx.featureStage)
	}
	if !strings.Contains(actx.orchestrationText, "multi-agent") {
		t.Errorf("orchestrationText missing 'multi-agent': %s", actx.orchestrationText)
	}
	if !strings.Contains(actx.effortBudgetText, "10") {
		t.Errorf("effortBudgetText missing '10': %s", actx.effortBudgetText)
	}
	if actx.outputConventionText == "" {
		t.Error("outputConventionText should be non-empty for developing")
	}

	// Files included for developing.
	if len(actx.filesContext) != 2 {
		t.Errorf("filesContext has %d entries, want 2", len(actx.filesContext))
	}

	// Render prompt and verify section ordering.
	prompt := renderHandoffPrompt(taskState, actx, "")
	conventionsIdx := strings.Index(prompt, "### Conventions")
	orchestrationIdx := strings.Index(prompt, "## Orchestration")
	taskIdx := strings.Index(prompt, "## Task:")

	if conventionsIdx < 0 {
		t.Fatal("prompt missing ### Conventions")
	}
	if orchestrationIdx < 0 {
		t.Fatal("prompt missing ## Orchestration")
	}
	if taskIdx < 0 {
		t.Fatal("prompt missing ## Task:")
	}
	if conventionsIdx >= orchestrationIdx {
		t.Error("Conventions should appear before Orchestration")
	}
	if orchestrationIdx >= taskIdx {
		t.Error("Orchestration should appear before Task")
	}
}

func TestStageIntegration_SpecifyingHandoff(t *testing.T) {
	taskState := map[string]any{
		"id":            "TASK-INTEG02",
		"summary":       "Write specification",
		"files_planned": []any{"work/spec/foo.md"},
	}

	actx := assembleContext(asmInput{
		taskState:    taskState,
		featureStage: "specifying",
	})

	if !actx.stageAware {
		t.Fatal("expected stageAware = true")
	}
	if !strings.Contains(actx.orchestrationText, "single-agent") {
		t.Errorf("orchestrationText missing 'single-agent': %s", actx.orchestrationText)
	}
	if actx.outputConventionText != "" {
		t.Error("outputConventionText should be empty for specifying")
	}
	// Files excluded for specifying.
	if len(actx.filesContext) != 0 {
		t.Errorf("filesContext has %d entries, want 0 (specifying excludes files)", len(actx.filesContext))
	}

	// Render prompt: no file paths section.
	prompt := renderHandoffPrompt(taskState, actx, "")
	if strings.Contains(prompt, "### Files") {
		t.Error("specifying prompt should not contain ### Files section")
	}
}

func TestStageIntegration_ValidationRejection(t *testing.T) {
	// ValidateFeatureStage with non-working state.
	mock := &mockEntityGetter{status: "done"}
	_, err := ValidateFeatureStage("FEAT-INTEG01", mock)
	if err == nil {
		t.Fatal("expected validation error for done state")
	}
	msg := err.Error()
	if !strings.Contains(msg, "FEAT-INTEG01") {
		t.Error("error missing feature ID")
	}
	if !strings.Contains(msg, "'done'") {
		t.Error("error missing quoted state")
	}
	if !strings.Contains(msg, "entity(action:") {
		t.Error("error missing recovery tool call")
	}
}

func TestStageIntegration_GracefulDegradation(t *testing.T) {
	taskState := map[string]any{
		"id":            "TASK-INTEG03",
		"summary":       "Orphan task",
		"files_planned": []any{"orphan.go"},
	}

	actx := assembleContext(asmInput{
		taskState: taskState,
		// No featureStage — backward compat.
	})

	if actx.stageAware {
		t.Error("expected stageAware = false for no featureStage")
	}
	if actx.orchestrationText != "" {
		t.Error("orchestrationText should be empty without featureStage")
	}
	// Files still included.
	if len(actx.filesContext) != 1 {
		t.Errorf("filesContext has %d entries, want 1", len(actx.filesContext))
	}
}

func TestStageIntegration_AllStagesHaveValidConfig(t *testing.T) {
	for _, s := range stage.AllStages() {
		cfg, ok := stage.ForStage(string(s))
		if !ok {
			t.Errorf("stage %q has no config", s)
			continue
		}

		taskState := map[string]any{
			"id":            "TASK-STAGETEST",
			"summary":       "test task",
			"files_planned": []any{"test.go"},
		}

		actx := assembleContext(asmInput{
			taskState:    taskState,
			featureStage: string(s),
		})

		if !actx.stageAware {
			t.Errorf("stage %q: stageAware = false", s)
		}
		if actx.orchestrationText == "" {
			t.Errorf("stage %q: empty orchestrationText", s)
		}
		if actx.effortBudgetText == "" {
			t.Errorf("stage %q: empty effortBudgetText", s)
		}
		if actx.toolSubsetText == "" {
			t.Errorf("stage %q: empty toolSubsetText", s)
		}

		// OutputConvention only for orchestrator-workers stages.
		if cfg.OutputConvention && actx.outputConventionText == "" {
			t.Errorf("stage %q: expected outputConventionText", s)
		}
		if !cfg.OutputConvention && actx.outputConventionText != "" {
			t.Errorf("stage %q: unexpected outputConventionText", s)
		}

		// File inclusion matches config.
		if cfg.IncludeFilePaths && len(actx.filesContext) == 0 {
			t.Errorf("stage %q: expected filesContext to be populated", s)
		}
		if !cfg.IncludeFilePaths && len(actx.filesContext) != 0 {
			t.Errorf("stage %q: expected filesContext to be empty", s)
		}
	}
}
