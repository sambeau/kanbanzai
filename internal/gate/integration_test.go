package gate_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/sambeau/kanbanzai/internal/binding"
	"github.com/sambeau/kanbanzai/internal/checkpoint"
	"github.com/sambeau/kanbanzai/internal/gate"
	"github.com/sambeau/kanbanzai/internal/model"
)

// ---------------------------------------------------------------------------
// Test service mocks
// ---------------------------------------------------------------------------

type testDocSvc struct {
	docs map[string]*gate.DocumentRecord // keyed by ID
}

func (s *testDocSvc) GetDocument(id string, _ bool) (*gate.DocumentRecord, error) {
	if s.docs == nil {
		return nil, fmt.Errorf("not found: %s", id)
	}
	d, ok := s.docs[id]
	if !ok {
		return nil, fmt.Errorf("not found: %s", id)
	}
	return d, nil
}

func (s *testDocSvc) ListDocuments(f gate.DocumentFilters) ([]*gate.DocumentRecord, error) {
	var results []*gate.DocumentRecord
	for _, d := range s.docs {
		if f.Owner != "" && d.Owner != f.Owner {
			continue
		}
		if f.Type != "" && d.Type != f.Type {
			continue
		}
		if f.Status != "" && d.Status != f.Status {
			continue
		}
		results = append(results, d)
	}
	return results, nil
}

type testEntitySvc struct {
	tasks []gate.EntityResult
}

func (s *testEntitySvc) List(entityType string) ([]gate.EntityResult, error) {
	if entityType == "task" {
		return s.tasks, nil
	}
	return nil, nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func writeBindingFile(t *testing.T, dir string, yaml string) string {
	t.Helper()
	path := filepath.Join(dir, "stage-bindings.yaml")
	if err := os.WriteFile(path, []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

// fallbackGate is a known hardcoded fallback for tests that need one.
func fallbackGate(from, to string, _ *model.Feature, _ gate.DocumentService, _ gate.EntityService) gate.GateResult {
	return gate.GateResult{
		Stage:     to,
		Satisfied: false,
		Reason:    "hardcoded: no approved design document found",
	}
}

// ---------------------------------------------------------------------------
// Binding YAML fragments
// ---------------------------------------------------------------------------

const designPrereqYAML = `stage_bindings:
  designing:
    description: "Design phase"
    orchestration: single-agent
    roles: [architect]
    skills: [write-design]
    prerequisites:
      documents:
        - type: design
          status: approved
      override_policy: agent
`

const twoStageYAML = `stage_bindings:
  designing:
    description: "Design phase"
    orchestration: single-agent
    roles: [architect]
    skills: [write-design]
    prerequisites:
      documents:
        - type: design
          status: approved
      override_policy: agent
  specifying:
    description: "Specification phase"
    orchestration: single-agent
    roles: [spec-author]
    skills: [write-spec]
    prerequisites:
      documents:
        - type: specification
          status: approved
      override_policy: checkpoint
`

const checkpointPolicyYAML = `stage_bindings:
  designing:
    description: "Design phase"
    orchestration: single-agent
    roles: [architect]
    skills: [write-design]
    prerequisites:
      documents:
        - type: design
          status: approved
      override_policy: checkpoint
`

const agentPolicyYAML = `stage_bindings:
  designing:
    description: "Design phase"
    orchestration: single-agent
    roles: [architect]
    skills: [write-design]
    prerequisites:
      documents:
        - type: design
          status: approved
      override_policy: agent
`

// ---------------------------------------------------------------------------
// Test 1: Registry-sourced gate blocks when prereq not met
// ---------------------------------------------------------------------------

func TestIntegration_RegistryGateBlocks_WhenPrereqNotMet(t *testing.T) {
	dir := t.TempDir()
	path := writeBindingFile(t, dir, designPrereqYAML)

	cache := gate.NewRegistryCache(path)
	router := gate.NewGateRouter(cache, fallbackGate)

	feat := &model.Feature{
		ID:     "FEAT-001",
		Parent: "P1-plan",
		// No Design field set — no approved design doc exists.
	}
	ctx := gate.PrereqEvalContext{
		Feature:   feat,
		DocSvc:    &testDocSvc{},
		EntitySvc: &testEntitySvc{},
	}

	result := router.CheckGate("proposed", "designing", ctx)

	if result.Satisfied {
		t.Fatal("expected gate to block transition when document prereq is not met")
	}
	if result.Source != "registry" {
		t.Errorf("expected source %q, got %q", "registry", result.Source)
	}
	if result.Stage != "designing" {
		t.Errorf("expected stage %q, got %q", "designing", result.Stage)
	}
	if result.Reason == "" {
		t.Error("expected a non-empty reason for blocked gate")
	}
}

// ---------------------------------------------------------------------------
// Test 2: Registry-sourced gate allows when prereq is met
// ---------------------------------------------------------------------------

func TestIntegration_RegistryGateAllows_WhenPrereqMet(t *testing.T) {
	dir := t.TempDir()
	path := writeBindingFile(t, dir, designPrereqYAML)

	cache := gate.NewRegistryCache(path)
	router := gate.NewGateRouter(cache, fallbackGate)

	docSvc := &testDocSvc{
		docs: map[string]*gate.DocumentRecord{
			"DOC-design-001": {
				ID:     "DOC-design-001",
				Status: "approved",
				Type:   "design",
				Owner:  "FEAT-001",
			},
		},
	}

	feat := &model.Feature{
		ID:     "FEAT-001",
		Parent: "P1-plan",
		Design: "DOC-design-001",
	}
	ctx := gate.PrereqEvalContext{
		Feature:   feat,
		DocSvc:    docSvc,
		EntitySvc: &testEntitySvc{},
	}

	result := router.CheckGate("proposed", "designing", ctx)

	if !result.Satisfied {
		t.Fatalf("expected gate to allow transition, got reason: %s", result.Reason)
	}
	if result.Source != "registry" {
		t.Errorf("expected source %q, got %q", "registry", result.Source)
	}
	if result.Stage != "designing" {
		t.Errorf("expected stage %q, got %q", "designing", result.Stage)
	}
}

// ---------------------------------------------------------------------------
// Test 3: Hot-reload — edited registry file is picked up on next call
// ---------------------------------------------------------------------------

func TestIntegration_HotReload_UpdatedPrereqsUsed(t *testing.T) {
	dir := t.TempDir()

	// Start with only a designing stage that requires design doc.
	path := writeBindingFile(t, dir, designPrereqYAML)

	cache := gate.NewRegistryCache(path)
	router := gate.NewGateRouter(cache, fallbackGate)

	feat := &model.Feature{
		ID:     "FEAT-001",
		Parent: "P1-plan",
	}
	ctx := gate.PrereqEvalContext{
		Feature:   feat,
		DocSvc:    &testDocSvc{},
		EntitySvc: &testEntitySvc{},
	}

	// First call: designing has document prereq → unsatisfied.
	result := router.CheckGate("proposed", "designing", ctx)
	if result.Satisfied {
		t.Fatal("first call: expected designing gate to be unsatisfied")
	}
	if result.Source != "registry" {
		t.Errorf("first call: expected source %q, got %q", "registry", result.Source)
	}

	// Specifying stage doesn't exist yet → should fall back to hardcoded.
	result = router.CheckGate("designing", "specifying", ctx)
	if result.Source != "hardcoded" {
		t.Errorf("before reload: expected source %q for specifying, got %q", "hardcoded", result.Source)
	}

	// Rewrite the binding file to add a specifying stage.
	writeBindingFile(t, dir, twoStageYAML)

	// Bump the mtime to guarantee the cache detects the change.
	// Some filesystems have 1-second mtime resolution.
	future := time.Now().Add(2 * time.Second)
	if err := os.Chtimes(path, future, future); err != nil {
		t.Fatalf("chtimes: %v", err)
	}

	// After reload: specifying stage should now use registry.
	result = router.CheckGate("designing", "specifying", ctx)
	if result.Source != "registry" {
		t.Errorf("after reload: expected source %q for specifying, got %q", "registry", result.Source)
	}
	if result.Satisfied {
		t.Error("after reload: expected specifying gate to be unsatisfied (no spec doc)")
	}
}

// ---------------------------------------------------------------------------
// Test 4: Delete registry file → fallback produces hardcoded results
// ---------------------------------------------------------------------------

func TestIntegration_DeleteRegistryFile_FallbackUsed(t *testing.T) {
	dir := t.TempDir()
	path := writeBindingFile(t, dir, designPrereqYAML)

	cache := gate.NewRegistryCache(path)
	router := gate.NewGateRouter(cache, fallbackGate)

	feat := &model.Feature{
		ID:     "FEAT-001",
		Parent: "P1-plan",
	}
	ctx := gate.PrereqEvalContext{
		Feature:   feat,
		DocSvc:    &testDocSvc{},
		EntitySvc: &testEntitySvc{},
	}

	// First call — registry is available.
	result := router.CheckGate("proposed", "designing", ctx)
	if result.Source != "registry" {
		t.Errorf("before delete: expected source %q, got %q", "registry", result.Source)
	}

	// Remove the registry file.
	if err := os.Remove(path); err != nil {
		t.Fatalf("removing binding file: %v", err)
	}

	// Second call — file gone, should fall back to hardcoded.
	result = router.CheckGate("proposed", "designing", ctx)
	if result.Source != "hardcoded" {
		t.Errorf("after delete: expected source %q, got %q", "hardcoded", result.Source)
	}
}

// ---------------------------------------------------------------------------
// Test 5: Agent override policy from router
// ---------------------------------------------------------------------------

func TestIntegration_AgentOverridePolicy(t *testing.T) {
	dir := t.TempDir()
	path := writeBindingFile(t, dir, agentPolicyYAML)

	cache := gate.NewRegistryCache(path)
	router := gate.NewGateRouter(cache, fallbackGate)

	policy := router.OverridePolicy("designing")
	if policy != "agent" {
		t.Errorf("expected override policy %q, got %q", "agent", policy)
	}
}

// ---------------------------------------------------------------------------
// Test 6: Checkpoint override policy → checkpoint created
// ---------------------------------------------------------------------------

func TestIntegration_CheckpointOverridePolicy_CreatesCheckpoint(t *testing.T) {
	dir := t.TempDir()
	path := writeBindingFile(t, dir, checkpointPolicyYAML)

	cache := gate.NewRegistryCache(path)
	router := gate.NewGateRouter(cache, fallbackGate)

	// Verify the router reports checkpoint policy.
	policy := router.OverridePolicy("designing")
	if policy != "checkpoint" {
		t.Fatalf("expected override policy %q, got %q", "checkpoint", policy)
	}

	// Create a real checkpoint store in a temp directory.
	stateDir := t.TempDir()
	store := checkpoint.NewStore(stateDir)

	cpResult, err := gate.HandleCheckpointOverride(gate.CheckpointOverrideParams{
		FeatureID:       "FEAT-001",
		FromStatus:      "proposed",
		ToStatus:        "designing",
		GateDescription: "no approved design document found",
		OverrideReason:  "design is in progress, will be completed soon",
		AgentIdentity:   "test-agent",
		CheckpointStore: store,
	})
	if err != nil {
		t.Fatalf("HandleCheckpointOverride: %v", err)
	}

	if !cpResult.CheckpointCreated {
		t.Fatal("expected checkpoint to be created")
	}
	if cpResult.CheckpointID == "" {
		t.Fatal("expected non-empty checkpoint ID")
	}

	// Read the checkpoint back and verify it's pending.
	record, err := store.Get(cpResult.CheckpointID)
	if err != nil {
		t.Fatalf("reading checkpoint: %v", err)
	}
	if record.Status != checkpoint.StatusPending {
		t.Errorf("expected checkpoint status %q, got %q", checkpoint.StatusPending, record.Status)
	}
	if record.CreatedBy != "test-agent" {
		t.Errorf("expected created_by %q, got %q", "test-agent", record.CreatedBy)
	}
	if record.Question == "" {
		t.Error("expected non-empty checkpoint question")
	}
}

// ---------------------------------------------------------------------------
// Test 7: Checkpoint approval resolved
// ---------------------------------------------------------------------------

func TestIntegration_CheckpointApproval_Resolved(t *testing.T) {
	approved := gate.ResolveCheckpointResponse("approved")
	if !approved {
		t.Error("expected 'approved' to resolve as true (approval)")
	}

	// Also check other approval-like responses.
	for _, resp := range []string{"yes", "approve", "looks good", "ok"} {
		if !gate.ResolveCheckpointResponse(resp) {
			t.Errorf("expected %q to resolve as approval, got rejection", resp)
		}
	}
}

// ---------------------------------------------------------------------------
// Test 8: Checkpoint rejection resolved
// ---------------------------------------------------------------------------

func TestIntegration_CheckpointRejection_Resolved(t *testing.T) {
	rejected := gate.ResolveCheckpointResponse("no")
	if rejected {
		t.Error("expected 'no' to resolve as false (rejection)")
	}

	// Also check other rejection-like responses.
	for _, resp := range []string{"reject", "rejected", "denied"} {
		if gate.ResolveCheckpointResponse(resp) {
			t.Errorf("expected %q to resolve as rejection, got approval", resp)
		}
	}
}

// ---------------------------------------------------------------------------
// Test 10: No binding file at all → hardcoded fallback is seamless
// (Placed before test 9 because test 9 modifies global evaluator state.)
// ---------------------------------------------------------------------------

func TestIntegration_NoBindingFile_FallbackSeamless(t *testing.T) {
	// Point the cache at a path that has never existed.
	cache := gate.NewRegistryCache(filepath.Join(t.TempDir(), "nonexistent", "stage-bindings.yaml"))

	fallbackCalled := false
	fb := func(from, to string, _ *model.Feature, _ gate.DocumentService, _ gate.EntityService) gate.GateResult {
		fallbackCalled = true
		return gate.GateResult{
			Stage:     to,
			Satisfied: true,
			Reason:    "hardcoded: all clear",
		}
	}

	router := gate.NewGateRouter(cache, fb)

	feat := &model.Feature{ID: "FEAT-099"}
	ctx := gate.PrereqEvalContext{
		Feature:   feat,
		DocSvc:    &testDocSvc{},
		EntitySvc: &testEntitySvc{},
	}

	result := router.CheckGate("proposed", "designing", ctx)

	if !fallbackCalled {
		t.Error("expected hardcoded fallback to be called when no binding file exists")
	}
	if result.Source != "hardcoded" {
		t.Errorf("expected source %q, got %q", "hardcoded", result.Source)
	}
	if !result.Satisfied {
		t.Error("expected satisfied from fallback")
	}
}

// ---------------------------------------------------------------------------
// Test 9: Extensibility — register a custom evaluator
//
// NOTE: This test replaces the "documents" evaluator in the global registry.
// It is placed last in the file because the override persists for the
// remainder of the test binary run.
// ---------------------------------------------------------------------------

func TestIntegration_Extensibility_CustomEvaluator(t *testing.T) {
	customCalled := false

	gate.RegisterEvaluator("documents", func(
		prereqs *binding.Prerequisites,
		stage string,
		ctx gate.PrereqEvalContext,
	) []gate.GateResult {
		customCalled = true
		return []gate.GateResult{{
			Stage:     stage,
			Satisfied: true,
			Reason:    "custom evaluator approved",
			Source:    "registry",
		}}
	})

	dir := t.TempDir()
	path := writeBindingFile(t, dir, designPrereqYAML)

	cache := gate.NewRegistryCache(path)
	router := gate.NewGateRouter(cache, fallbackGate)

	feat := &model.Feature{ID: "FEAT-001"}
	ctx := gate.PrereqEvalContext{
		Feature:   feat,
		DocSvc:    &testDocSvc{},
		EntitySvc: &testEntitySvc{},
	}

	result := router.CheckGate("proposed", "designing", ctx)

	if !customCalled {
		t.Error("expected custom evaluator to be dispatched")
	}
	if !result.Satisfied {
		t.Errorf("expected custom evaluator to satisfy gate, got reason: %s", result.Reason)
	}
	if result.Source != "registry" {
		t.Errorf("expected source %q, got %q", "registry", result.Source)
	}
}
