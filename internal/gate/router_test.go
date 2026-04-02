package gate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sambeau/kanbanzai/internal/model"
)

func alwaysSatisfied(from, to string, feature *model.Feature, docSvc DocumentService, entitySvc EntityService) GateResult {
	return GateResult{Stage: to, Satisfied: true, Reason: "hardcoded satisfied"}
}

func alwaysUnsatisfied(from, to string, feature *model.Feature, docSvc DocumentService, entitySvc EntityService) GateResult {
	return GateResult{Stage: to, Satisfied: false, Reason: "no approved design document found"}
}

const routerBindingYAML = `stage_bindings:
  specifying:
    description: "Write specification"
    orchestration: single-agent
    roles: [spec-author]
    skills: [write-spec]
    prerequisites:
      documents:
        - type: design
          status: approved
      override_policy: checkpoint
  designing:
    description: "Design phase"
    orchestration: single-agent
    roles: [architect]
    skills: [write-design]
  developing:
    description: "Development phase"
    orchestration: multi-agent
    roles: [orchestrator]
    skills: [orchestrate-development]
    prerequisites:
      override_policy: agent
`

func writeRouterBindingFile(t *testing.T, dir string) string {
	t.Helper()
	p := filepath.Join(dir, "stage-bindings.yaml")
	if err := os.WriteFile(p, []byte(routerBindingYAML), 0o644); err != nil {
		t.Fatalf("writing test binding file: %v", err)
	}
	return p
}

func TestCheckGate_RegistryProvidesPrereqs_EvaluatorUsed(t *testing.T) {
	dir := t.TempDir()
	path := writeRouterBindingFile(t, dir)
	cache := NewRegistryCache(path)

	docSvc := &mockDocSvc{
		getDoc: func(id string, _ bool) (*DocumentRecord, error) {
			if id == "DOC-design-001" {
				return &DocumentRecord{
					ID:     "DOC-design-001",
					Status: "approved",
					Type:   "design",
					Owner:  "FEAT-001",
				}, nil
			}
			return nil, nil
		},
		listDoc: func(f DocumentFilters) ([]*DocumentRecord, error) {
			return nil, nil
		},
	}

	feat := &model.Feature{
		ID:     "FEAT-001",
		Parent: "P1-plan",
		Design: "DOC-design-001",
	}

	ctx := PrereqEvalContext{
		Feature: feat,
		DocSvc:  docSvc,
	}

	router := NewGateRouter(cache, alwaysUnsatisfied)
	result := router.CheckGate("designing", "specifying", ctx)

	if !result.Satisfied {
		t.Errorf("expected satisfied, got reason: %s", result.Reason)
	}
	if result.Source != "registry" {
		t.Errorf("expected source %q, got %q", "registry", result.Source)
	}
	if result.Stage != "specifying" {
		t.Errorf("expected stage %q, got %q", "specifying", result.Stage)
	}
}

func TestCheckGate_RegistryPrereqsUnsatisfied_ReturnsFirstFailure(t *testing.T) {
	dir := t.TempDir()
	path := writeRouterBindingFile(t, dir)
	cache := NewRegistryCache(path)

	docSvc := &mockDocSvc{
		getDoc: func(_ string, _ bool) (*DocumentRecord, error) {
			return nil, nil
		},
		listDoc: func(_ DocumentFilters) ([]*DocumentRecord, error) {
			return nil, nil
		},
	}

	feat := &model.Feature{
		ID:     "FEAT-001",
		Parent: "P1-plan",
	}

	ctx := PrereqEvalContext{
		Feature: feat,
		DocSvc:  docSvc,
	}

	router := NewGateRouter(cache, alwaysSatisfied)
	result := router.CheckGate("designing", "specifying", ctx)

	if result.Satisfied {
		t.Error("expected not satisfied when document prereq is missing")
	}
	if result.Source != "registry" {
		t.Errorf("expected source %q, got %q", "registry", result.Source)
	}
	if result.Reason == "" {
		t.Error("expected a reason for unsatisfied result")
	}
}

func TestCheckGate_NilCache_HardcodedFallback(t *testing.T) {
	fallbackCalled := false
	fallback := func(from, to string, feature *model.Feature, docSvc DocumentService, entitySvc EntityService) GateResult {
		fallbackCalled = true
		return GateResult{Stage: to, Satisfied: true, Reason: "hardcoded satisfied"}
	}

	feat := &model.Feature{ID: "FEAT-001"}
	ctx := PrereqEvalContext{Feature: feat}

	router := NewGateRouter(nil, fallback)
	result := router.CheckGate("designing", "specifying", ctx)

	if !fallbackCalled {
		t.Error("expected fallback to be called when cache is nil")
	}
	if result.Source != "hardcoded" {
		t.Errorf("expected source %q, got %q", "hardcoded", result.Source)
	}
	if !result.Satisfied {
		t.Error("expected satisfied from fallback")
	}
}

func TestCheckGate_RegistryNoPrereqsForStage_HardcodedFallback(t *testing.T) {
	dir := t.TempDir()
	path := writeRouterBindingFile(t, dir)
	cache := NewRegistryCache(path)

	fallbackCalled := false
	fallback := func(from, to string, feature *model.Feature, docSvc DocumentService, entitySvc EntityService) GateResult {
		fallbackCalled = true
		return GateResult{Stage: to, Satisfied: true, Reason: "hardcoded satisfied"}
	}

	feat := &model.Feature{ID: "FEAT-001"}
	ctx := PrereqEvalContext{Feature: feat}

	// "designing" has no prerequisites in our test YAML
	router := NewGateRouter(cache, fallback)
	result := router.CheckGate("somewhere", "designing", ctx)

	if !fallbackCalled {
		t.Error("expected fallback to be called when registry has no prereqs for stage")
	}
	if result.Source != "hardcoded" {
		t.Errorf("expected source %q, got %q", "hardcoded", result.Source)
	}
}

func TestCheckGate_MissingRegistryFile_HardcodedFallback(t *testing.T) {
	cache := NewRegistryCache(filepath.Join(t.TempDir(), "nonexistent.yaml"))

	fallbackCalled := false
	fallback := func(from, to string, feature *model.Feature, docSvc DocumentService, entitySvc EntityService) GateResult {
		fallbackCalled = true
		return GateResult{Stage: to, Satisfied: false, Reason: "no approved design document found"}
	}

	feat := &model.Feature{ID: "FEAT-001"}
	ctx := PrereqEvalContext{Feature: feat}

	router := NewGateRouter(cache, fallback)
	result := router.CheckGate("designing", "specifying", ctx)

	if !fallbackCalled {
		t.Error("expected fallback when registry file is missing")
	}
	if result.Source != "hardcoded" {
		t.Errorf("expected source %q, got %q", "hardcoded", result.Source)
	}
}

func TestCheckGate_FailureMessage_NoInternalTerms(t *testing.T) {
	dir := t.TempDir()
	path := writeRouterBindingFile(t, dir)
	cache := NewRegistryCache(path)

	docSvc := &mockDocSvc{
		getDoc: func(_ string, _ bool) (*DocumentRecord, error) {
			return nil, nil
		},
		listDoc: func(_ DocumentFilters) ([]*DocumentRecord, error) {
			return nil, nil
		},
	}

	feat := &model.Feature{
		ID:     "FEAT-001",
		Parent: "P1-plan",
	}

	ctx := PrereqEvalContext{
		Feature: feat,
		DocSvc:  docSvc,
	}

	router := NewGateRouter(cache, alwaysSatisfied)
	result := router.CheckGate("designing", "specifying", ctx)

	if result.Satisfied {
		t.Fatal("expected not satisfied for this test")
	}

	forbidden := []string{"binding registry", "hardcoded", "fallback"}
	lower := strings.ToLower(result.Reason)
	for _, term := range forbidden {
		if strings.Contains(lower, term) {
			t.Errorf("gate failure reason contains forbidden term %q: %s", term, result.Reason)
		}
	}
}

func TestCheckGate_FallbackFailureMessage_NoInternalTerms(t *testing.T) {
	fallback := func(from, to string, feature *model.Feature, docSvc DocumentService, entitySvc EntityService) GateResult {
		return GateResult{Stage: to, Satisfied: false, Reason: "no approved design document found"}
	}

	feat := &model.Feature{ID: "FEAT-001"}
	ctx := PrereqEvalContext{Feature: feat}

	router := NewGateRouter(nil, fallback)
	result := router.CheckGate("designing", "specifying", ctx)

	if result.Satisfied {
		t.Fatal("expected not satisfied for this test")
	}

	forbidden := []string{"binding registry", "hardcoded", "fallback"}
	lower := strings.ToLower(result.Reason)
	for _, term := range forbidden {
		if strings.Contains(lower, term) {
			t.Errorf("gate failure reason contains forbidden term %q: %s", term, result.Reason)
		}
	}
}

func TestOverridePolicy_NilCache_ReturnsAgent(t *testing.T) {
	router := NewGateRouter(nil, alwaysSatisfied)
	policy := router.OverridePolicy("specifying")
	if policy != "agent" {
		t.Errorf("expected %q, got %q", "agent", policy)
	}
}

func TestOverridePolicy_RegistrySpecifiesCheckpoint(t *testing.T) {
	dir := t.TempDir()
	path := writeRouterBindingFile(t, dir)
	cache := NewRegistryCache(path)

	router := NewGateRouter(cache, alwaysSatisfied)
	policy := router.OverridePolicy("specifying")
	if policy != "checkpoint" {
		t.Errorf("expected %q, got %q", "checkpoint", policy)
	}
}

func TestOverridePolicy_NoOverridePolicyField_ReturnsAgent(t *testing.T) {
	dir := t.TempDir()
	path := writeRouterBindingFile(t, dir)
	cache := NewRegistryCache(path)

	// "designing" has no prerequisites at all, so no override_policy
	router := NewGateRouter(cache, alwaysSatisfied)
	policy := router.OverridePolicy("designing")
	if policy != "agent" {
		t.Errorf("expected %q, got %q", "agent", policy)
	}
}

func TestOverridePolicy_UnknownStage_ReturnsAgent(t *testing.T) {
	dir := t.TempDir()
	path := writeRouterBindingFile(t, dir)
	cache := NewRegistryCache(path)

	router := NewGateRouter(cache, alwaysSatisfied)
	policy := router.OverridePolicy("nonexistent-stage")
	if policy != "agent" {
		t.Errorf("expected %q, got %q", "agent", policy)
	}
}

func TestOverridePolicy_MissingFile_ReturnsAgent(t *testing.T) {
	cache := NewRegistryCache(filepath.Join(t.TempDir(), "nonexistent.yaml"))
	router := NewGateRouter(cache, alwaysSatisfied)
	policy := router.OverridePolicy("specifying")
	if policy != "agent" {
		t.Errorf("expected %q, got %q", "agent", policy)
	}
}

func TestOverridePolicy_ExplicitAgent(t *testing.T) {
	dir := t.TempDir()
	path := writeRouterBindingFile(t, dir)
	cache := NewRegistryCache(path)

	// "developing" has override_policy: agent explicitly
	router := NewGateRouter(cache, alwaysSatisfied)
	policy := router.OverridePolicy("developing")
	if policy != "agent" {
		t.Errorf("expected %q, got %q", "agent", policy)
	}
}

func TestCheckGate_FallbackReceivesCorrectArgs(t *testing.T) {
	var gotFrom, gotTo string
	var gotFeature *model.Feature

	fallback := func(from, to string, feature *model.Feature, docSvc DocumentService, entitySvc EntityService) GateResult {
		gotFrom = from
		gotTo = to
		gotFeature = feature
		return GateResult{Stage: to, Satisfied: true}
	}

	feat := &model.Feature{ID: "FEAT-042"}
	ctx := PrereqEvalContext{Feature: feat}

	router := NewGateRouter(nil, fallback)
	router.CheckGate("designing", "specifying", ctx)

	if gotFrom != "designing" {
		t.Errorf("expected from %q, got %q", "designing", gotFrom)
	}
	if gotTo != "specifying" {
		t.Errorf("expected to %q, got %q", "specifying", gotTo)
	}
	if gotFeature == nil || gotFeature.ID != "FEAT-042" {
		t.Errorf("expected feature FEAT-042, got %v", gotFeature)
	}
}
