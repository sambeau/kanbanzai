package mcp_test

import (
	"testing"

	chk "kanbanzai/internal/checkpoint"
	kbzconfig "kanbanzai/internal/config"
	kbzctx "kanbanzai/internal/context"
	"kanbanzai/internal/git"
	kbzmcp "kanbanzai/internal/mcp"
	"kanbanzai/internal/service"
	"kanbanzai/internal/worktree"
)

// TestAllTools_HaveAnnotations is a canary test that verifies every registered MCP
// tool has explicit values for readOnlyHint, destructiveHint, and idempotentHint.
//
// If this test fails, a newly added tool is missing one or more safety annotations.
// Add mcp.WithReadOnlyHintAnnotation, mcp.WithDestructiveHintAnnotation, and
// mcp.WithIdempotentHintAnnotation to the tool's mcp.NewTool(...) call, referring
// to the classification table in work/spec/hardening.md §10.4 for correct values.
func TestAllTools_HaveAnnotations(t *testing.T) {
	// Minimal service instances — temp dirs only, no real Git or network needed.
	entityRoot := t.TempDir()
	stateRoot := t.TempDir()
	profileRoot := t.TempDir()
	checkpointRoot := t.TempDir()
	indexRoot := t.TempDir()
	repoRoot := t.TempDir()

	entitySvc := service.NewEntityService(entityRoot)
	knowledgeSvc := service.NewKnowledgeService(stateRoot)
	dispatchSvc := service.NewDispatchService(entitySvc, knowledgeSvc)
	checkpointStore := chk.NewStore(checkpointRoot)
	profileStore := kbzctx.NewProfileStore(profileRoot)
	intelligenceSvc := service.NewIntelligenceService(indexRoot, repoRoot)
	docSvc := service.NewDocumentService(stateRoot, repoRoot)
	worktreeStore := worktree.NewStore(stateRoot)
	gitOps := worktree.NewGit(repoRoot)
	decomposeSvc := service.NewDecomposeService(entitySvc, docSvc)
	reviewSvc := service.NewReviewService(entitySvc, intelligenceSvc, repoRoot)
	conflictSvc := service.NewConflictService(entitySvc, nil, repoRoot)

	branchThresholds := git.BranchThresholds{}
	cleanupCfg := &kbzconfig.CleanupConfig{}
	localConfig := &kbzconfig.LocalConfig{}

	// Collect every tool from every registered tool group — mirrors the
	// registration order in server.go so additions there show up here.
	var all []string // tool names for diagnostics
	type annotated struct {
		name            string
		readOnlyHint    *bool
		destructiveHint *bool
		idempotentHint  *bool
	}
	var tools []annotated

	add := func(name string, ro, d, i *bool) {
		all = append(all, name)
		tools = append(tools, annotated{name, ro, d, i})
	}

	for _, st := range kbzmcp.EntityTools(entitySvc) {
		ann := st.Tool.Annotations
		add(st.Tool.Name, ann.ReadOnlyHint, ann.DestructiveHint, ann.IdempotentHint)
	}
	for _, st := range kbzmcp.PlanTools(entitySvc) {
		ann := st.Tool.Annotations
		add(st.Tool.Name, ann.ReadOnlyHint, ann.DestructiveHint, ann.IdempotentHint)
	}
	for _, st := range kbzmcp.DocRecordTools(docSvc) {
		ann := st.Tool.Annotations
		add(st.Tool.Name, ann.ReadOnlyHint, ann.DestructiveHint, ann.IdempotentHint)
	}
	for _, st := range kbzmcp.ConfigTools() {
		ann := st.Tool.Annotations
		add(st.Tool.Name, ann.ReadOnlyHint, ann.DestructiveHint, ann.IdempotentHint)
	}
	for _, st := range kbzmcp.DocIntelligenceTools(intelligenceSvc, docSvc) {
		ann := st.Tool.Annotations
		add(st.Tool.Name, ann.ReadOnlyHint, ann.DestructiveHint, ann.IdempotentHint)
	}
	for _, st := range kbzmcp.QueryTools(entitySvc, docSvc) {
		ann := st.Tool.Annotations
		add(st.Tool.Name, ann.ReadOnlyHint, ann.DestructiveHint, ann.IdempotentHint)
	}
	for _, st := range kbzmcp.MigrationTools(entitySvc) {
		ann := st.Tool.Annotations
		add(st.Tool.Name, ann.ReadOnlyHint, ann.DestructiveHint, ann.IdempotentHint)
	}
	for _, st := range kbzmcp.KnowledgeTools(knowledgeSvc) {
		ann := st.Tool.Annotations
		add(st.Tool.Name, ann.ReadOnlyHint, ann.DestructiveHint, ann.IdempotentHint)
	}
	for _, st := range kbzmcp.ProfileTools(profileStore) {
		ann := st.Tool.Annotations
		add(st.Tool.Name, ann.ReadOnlyHint, ann.DestructiveHint, ann.IdempotentHint)
	}
	for _, st := range kbzmcp.ContextTools(profileStore, knowledgeSvc, entitySvc, intelligenceSvc) {
		ann := st.Tool.Annotations
		add(st.Tool.Name, ann.ReadOnlyHint, ann.DestructiveHint, ann.IdempotentHint)
	}
	for _, st := range kbzmcp.AgentCapabilityTools(entitySvc, knowledgeSvc, intelligenceSvc) {
		ann := st.Tool.Annotations
		add(st.Tool.Name, ann.ReadOnlyHint, ann.DestructiveHint, ann.IdempotentHint)
	}
	for _, st := range kbzmcp.BatchImportTools(docSvc) {
		ann := st.Tool.Annotations
		add(st.Tool.Name, ann.ReadOnlyHint, ann.DestructiveHint, ann.IdempotentHint)
	}
	for _, st := range kbzmcp.WorktreeTools(worktreeStore, entitySvc, gitOps) {
		ann := st.Tool.Annotations
		add(st.Tool.Name, ann.ReadOnlyHint, ann.DestructiveHint, ann.IdempotentHint)
	}
	for _, st := range kbzmcp.BranchTools(worktreeStore, repoRoot, branchThresholds) {
		ann := st.Tool.Annotations
		add(st.Tool.Name, ann.ReadOnlyHint, ann.DestructiveHint, ann.IdempotentHint)
	}
	for _, st := range kbzmcp.CleanupTools(worktreeStore, gitOps, cleanupCfg) {
		ann := st.Tool.Annotations
		add(st.Tool.Name, ann.ReadOnlyHint, ann.DestructiveHint, ann.IdempotentHint)
	}
	for _, st := range kbzmcp.MergeTools(worktreeStore, entitySvc, repoRoot, branchThresholds, localConfig) {
		ann := st.Tool.Annotations
		add(st.Tool.Name, ann.ReadOnlyHint, ann.DestructiveHint, ann.IdempotentHint)
	}
	for _, st := range kbzmcp.PRTools(worktreeStore, entitySvc, repoRoot, branchThresholds, localConfig) {
		ann := st.Tool.Annotations
		add(st.Tool.Name, ann.ReadOnlyHint, ann.DestructiveHint, ann.IdempotentHint)
	}
	for _, st := range kbzmcp.QueueTools(entitySvc) {
		ann := st.Tool.Annotations
		add(st.Tool.Name, ann.ReadOnlyHint, ann.DestructiveHint, ann.IdempotentHint)
	}
	for _, st := range kbzmcp.EstimationTools(entitySvc, knowledgeSvc) {
		ann := st.Tool.Annotations
		add(st.Tool.Name, ann.ReadOnlyHint, ann.DestructiveHint, ann.IdempotentHint)
	}
	for _, st := range kbzmcp.DispatchTools(dispatchSvc, checkpointStore, profileStore, knowledgeSvc, entitySvc, intelligenceSvc) {
		ann := st.Tool.Annotations
		add(st.Tool.Name, ann.ReadOnlyHint, ann.DestructiveHint, ann.IdempotentHint)
	}
	for _, st := range kbzmcp.IncidentTools(entitySvc) {
		ann := st.Tool.Annotations
		add(st.Tool.Name, ann.ReadOnlyHint, ann.DestructiveHint, ann.IdempotentHint)
	}
	for _, st := range kbzmcp.DecomposeTools(decomposeSvc) {
		ann := st.Tool.Annotations
		add(st.Tool.Name, ann.ReadOnlyHint, ann.DestructiveHint, ann.IdempotentHint)
	}
	for _, st := range kbzmcp.ReviewTools(reviewSvc) {
		ann := st.Tool.Annotations
		add(st.Tool.Name, ann.ReadOnlyHint, ann.DestructiveHint, ann.IdempotentHint)
	}
	for _, st := range kbzmcp.ConflictTools(conflictSvc) {
		ann := st.Tool.Annotations
		add(st.Tool.Name, ann.ReadOnlyHint, ann.DestructiveHint, ann.IdempotentHint)
	}

	if len(tools) == 0 {
		t.Fatal("no tools collected — check that all tool groups are included in this test")
	}

	t.Logf("checking annotations on %d tools: %v", len(tools), all)

	for _, tool := range tools {
		tool := tool
		t.Run(tool.name, func(t *testing.T) {
			if tool.readOnlyHint == nil {
				t.Errorf("tool %q is missing readOnlyHint annotation — add mcp.WithReadOnlyHintAnnotation(...) to its mcp.NewTool call", tool.name)
			}
			if tool.destructiveHint == nil {
				t.Errorf("tool %q is missing destructiveHint annotation — add mcp.WithDestructiveHintAnnotation(...) to its mcp.NewTool call", tool.name)
			}
			if tool.idempotentHint == nil {
				t.Errorf("tool %q is missing idempotentHint annotation — add mcp.WithIdempotentHintAnnotation(...) to its mcp.NewTool call", tool.name)
			}
		})
	}
}
