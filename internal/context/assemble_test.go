package context

import (
	"strings"
	"testing"

	"kanbanzai/internal/service"
	"kanbanzai/internal/storage"
)

func TestAssemble_RoleOnly_ProfileAndKnowledge(t *testing.T) {
	t.Parallel()

	profileDir := t.TempDir()
	stateDir := t.TempDir()

	writeProfileFile(t, profileDir, "backend.yaml", `
id: backend
description: "Backend development role"
packages:
  - internal/service
conventions:
  - "Use idiomatic Go"
`)
	store := NewProfileStore(profileDir)
	knowledgeSvc := service.NewKnowledgeService(stateDir)

	// T3 scoped to role
	_, _, err := knowledgeSvc.Contribute(service.ContributeInput{
		Topic: "error-handling", Scope: "backend",
		Content: "Always wrap errors with fmt.Errorf and %w", Tier: 3,
	})
	if err != nil {
		t.Fatalf("Contribute backend T3: %v", err)
	}
	// T3 scoped to project — must appear
	_, _, err = knowledgeSvc.Contribute(service.ContributeInput{
		Topic: "project-wide-tip", Scope: "project",
		Content: "Use t.TempDir() for all filesystem tests", Tier: 3,
	})
	if err != nil {
		t.Fatalf("Contribute project T3: %v", err)
	}
	// T3 scoped to a different role — must NOT appear
	_, _, err = knowledgeSvc.Contribute(service.ContributeInput{
		Topic: "frontend-tip", Scope: "frontend",
		Content: "Use React hooks for state management", Tier: 3,
	})
	if err != nil {
		t.Fatalf("Contribute frontend T3: %v", err)
	}

	result, err := Assemble(AssemblyInput{Role: "backend"}, store, knowledgeSvc, nil, nil)
	if err != nil {
		t.Fatalf("Assemble() error = %v", err)
	}

	if result.Role != "backend" {
		t.Errorf("Role = %q, want %q", result.Role, "backend")
	}
	if result.TaskID != "" {
		t.Errorf("TaskID = %q, want empty", result.TaskID)
	}
	if result.Trimmed != 0 {
		t.Errorf("Trimmed = %d, want 0", result.Trimmed)
	}

	var profileCount, t2Count, t3Count, taskCount int
	for _, item := range result.Items {
		switch item.Source {
		case SourceProfile:
			profileCount++
		case SourceKnowledgeT2:
			t2Count++
		case SourceKnowledgeT3:
			t3Count++
		case SourceTask:
			taskCount++
		}
	}

	if profileCount != 1 {
		t.Errorf("profile items = %d, want 1", profileCount)
	}
	if t3Count != 2 {
		t.Errorf("T3 items = %d, want 2 (backend-scoped + project-scoped)", t3Count)
	}
	if t2Count != 0 {
		t.Errorf("T2 items = %d, want 0", t2Count)
	}
	if taskCount != 0 {
		t.Errorf("task items = %d, want 0", taskCount)
	}

	// Profile must be first
	if len(result.Items) > 0 && result.Items[0].Source != SourceProfile {
		t.Errorf("first item source = %q, want %q", result.Items[0].Source, SourceProfile)
	}
	// Profile content must mention the role ID
	if len(result.Items) > 0 && !strings.Contains(result.Items[0].Content, "backend") {
		t.Errorf("profile content does not contain role ID")
	}

	// ByteCount must match the sum of item content sizes
	wantBytes := 0
	for _, item := range result.Items {
		wantBytes += len(item.Content)
	}
	if result.ByteCount != wantBytes {
		t.Errorf("ByteCount = %d, want %d", result.ByteCount, wantBytes)
	}
}

func TestAssemble_WithTaskID_IncludesTaskInstructions(t *testing.T) {
	t.Parallel()

	profileDir := t.TempDir()
	stateDir := t.TempDir()

	writeProfileFile(t, profileDir, "backend.yaml", `
id: backend
description: "Backend development role"
`)
	store := NewProfileStore(profileDir)
	knowledgeSvc := service.NewKnowledgeService(stateDir)
	entitySvc := service.NewEntityService(stateDir)

	// Write a task record directly to avoid the plan→feature→task creation chain.
	const taskID = "TASK-0123456789012"
	const taskSlug = "login-endpoint"
	entityStore := storage.NewEntityStore(stateDir)
	_, err := entityStore.Write(storage.EntityRecord{
		Type: "task",
		ID:   taskID,
		Slug: taskSlug,
		Fields: map[string]any{
			"id":             taskID,
			"parent_feature": "FEAT-0000000000001",
			"slug":           taskSlug,
			"summary":        "Implement login endpoint",
			"status":         "in-progress",
			"verification":   "POST /login returns 200 with valid credentials",
		},
	})
	if err != nil {
		t.Fatalf("write task record: %v", err)
	}

	result, err := Assemble(
		AssemblyInput{Role: "backend", TaskID: taskID},
		store, knowledgeSvc, entitySvc, nil,
	)
	if err != nil {
		t.Fatalf("Assemble() error = %v", err)
	}

	if result.TaskID != taskID {
		t.Errorf("TaskID = %q, want %q", result.TaskID, taskID)
	}

	var taskCount int
	var taskContent string
	for _, item := range result.Items {
		if item.Source == SourceTask {
			taskCount++
			taskContent = item.Content
		}
	}

	if taskCount != 1 {
		t.Fatalf("task items = %d, want 1", taskCount)
	}
	if !strings.Contains(taskContent, "Implement login endpoint") {
		t.Errorf("task content does not contain summary; got: %q", taskContent)
	}
	if !strings.Contains(taskContent, taskID) {
		t.Errorf("task content does not contain task ID; got: %q", taskContent)
	}
	if !strings.Contains(taskContent, "POST /login returns 200") {
		t.Errorf("task content does not contain verification; got: %q", taskContent)
	}

	// Task must be the last item.
	last := result.Items[len(result.Items)-1]
	if last.Source != SourceTask {
		t.Errorf("last item source = %q, want %q", last.Source, SourceTask)
	}
}

func TestAssemble_ByteBudget_T3TrimmedFirst(t *testing.T) {
	t.Parallel()

	profileDir := t.TempDir()
	stateDir := t.TempDir()

	writeProfileFile(t, profileDir, "be.yaml", `
id: be
description: "Test role"
`)
	store := NewProfileStore(profileDir)
	knowledgeSvc := service.NewKnowledgeService(stateDir)

	t2Content := strings.Repeat("T2content.", 15) // 150 bytes of body
	t3Content := strings.Repeat("T3content.", 15) // 150 bytes of body

	_, _, err := knowledgeSvc.Contribute(service.ContributeInput{
		Topic: "t2-topic", Scope: "be", Content: t2Content, Tier: 2,
	})
	if err != nil {
		t.Fatalf("Contribute T2: %v", err)
	}
	_, _, err = knowledgeSvc.Contribute(service.ContributeInput{
		Topic: "t3-topic", Scope: "be", Content: t3Content, Tier: 3,
	})
	if err != nil {
		t.Fatalf("Contribute T3: %v", err)
	}

	// Sizing pass with unlimited budget
	big, err := Assemble(AssemblyInput{Role: "be", MaxBytes: 1000000}, store, knowledgeSvc, nil, nil)
	if err != nil {
		t.Fatalf("Assemble (sizing pass): %v", err)
	}

	var profileSize, t2Size int
	for _, item := range big.Items {
		switch item.Source {
		case SourceProfile:
			profileSize = len(item.Content)
		case SourceKnowledgeT2:
			t2Size = len(item.Content)
		}
	}
	if t2Size == 0 {
		t.Fatal("sizing pass: T2 item not found")
	}

	// Budget fits profile + T2 exactly; T3 must be trimmed
	budget := profileSize + t2Size

	result, err := Assemble(AssemblyInput{Role: "be", MaxBytes: budget}, store, knowledgeSvc, nil, nil)
	if err != nil {
		t.Fatalf("Assemble() error = %v", err)
	}

	if result.Trimmed < 1 {
		t.Errorf("Trimmed = %d, want >= 1", result.Trimmed)
	}

	var t2Count, t3Count int
	for _, item := range result.Items {
		switch item.Source {
		case SourceKnowledgeT2:
			t2Count++
		case SourceKnowledgeT3:
			t3Count++
		}
	}

	if t3Count != 0 {
		t.Errorf("T3 items in result = %d, want 0 (T3 should be trimmed first)", t3Count)
	}
	if t2Count != 1 {
		t.Errorf("T2 items in result = %d, want 1 (T2 must not be trimmed)", t2Count)
	}
	if result.ByteCount > budget {
		t.Errorf("ByteCount = %d exceeds budget %d", result.ByteCount, budget)
	}
}

func TestAssemble_ByteBudget_T2OnlyTrimmedAfterT3Exhausted(t *testing.T) {
	t.Parallel()

	profileDir := t.TempDir()
	stateDir := t.TempDir()

	writeProfileFile(t, profileDir, "be.yaml", `
id: be
description: "Test role"
`)
	store := NewProfileStore(profileDir)
	knowledgeSvc := service.NewKnowledgeService(stateDir)

	body := strings.Repeat("Xcontent.", 20) // 180 bytes of body each

	// 1 T2 entry, 2 T3 entries — all same confidence (0.5)
	_, _, err := knowledgeSvc.Contribute(service.ContributeInput{
		Topic: "t2-entry", Scope: "be", Content: body, Tier: 2,
	})
	if err != nil {
		t.Fatalf("Contribute T2: %v", err)
	}
	bodyA := strings.Repeat("T3alpha.", 20) // 160 bytes, distinct from bodyB
	bodyB := strings.Repeat("T3beta!!", 20) // 160 bytes, distinct from bodyA

	_, _, err = knowledgeSvc.Contribute(service.ContributeInput{
		Topic: "t3-entry-a", Scope: "be", Content: bodyA, Tier: 3,
	})
	if err != nil {
		t.Fatalf("Contribute T3-A: %v", err)
	}
	_, _, err = knowledgeSvc.Contribute(service.ContributeInput{
		Topic: "t3-entry-b", Scope: "be", Content: bodyB, Tier: 3,
	})
	if err != nil {
		t.Fatalf("Contribute T3-B: %v", err)
	}

	// Sizing pass
	big, err := Assemble(AssemblyInput{Role: "be", MaxBytes: 1000000}, store, knowledgeSvc, nil, nil)
	if err != nil {
		t.Fatalf("Assemble (sizing pass): %v", err)
	}

	var profileSize, t2Size int
	for _, item := range big.Items {
		if item.Source == SourceProfile {
			profileSize = len(item.Content)
		} else if item.Source == SourceKnowledgeT2 {
			t2Size = len(item.Content)
		}
	}
	if t2Size == 0 {
		t.Fatal("sizing pass: T2 item not found")
	}

	// Budget: exactly profile + T2 — forces both T3 entries to be trimmed, T2 survives
	budget := profileSize + t2Size

	result, err := Assemble(AssemblyInput{Role: "be", MaxBytes: budget}, store, knowledgeSvc, nil, nil)
	if err != nil {
		t.Fatalf("Assemble() error = %v", err)
	}

	if result.Trimmed < 2 {
		t.Errorf("Trimmed = %d, want >= 2 (both T3 entries must be trimmed)", result.Trimmed)
	}

	var t2Count, t3Count int
	for _, item := range result.Items {
		switch item.Source {
		case SourceKnowledgeT2:
			t2Count++
		case SourceKnowledgeT3:
			t3Count++
		}
	}

	if t3Count != 0 {
		t.Errorf("T3 items = %d, want 0 (all T3 trimmed before any T2)", t3Count)
	}
	if t2Count != 1 {
		t.Errorf("T2 items = %d, want 1 (T2 must not be trimmed while T3 remains)", t2Count)
	}
	if result.ByteCount > budget {
		t.Errorf("ByteCount = %d exceeds budget %d", result.ByteCount, budget)
	}
}

func TestAssemble_EmptyKnowledgeStore_ProfileOnly(t *testing.T) {
	t.Parallel()

	profileDir := t.TempDir()
	stateDir := t.TempDir()

	writeProfileFile(t, profileDir, "be.yaml", `
id: be
description: "Test role"
`)
	store := NewProfileStore(profileDir)
	knowledgeSvc := service.NewKnowledgeService(stateDir)

	result, err := Assemble(AssemblyInput{Role: "be"}, store, knowledgeSvc, nil, nil)
	if err != nil {
		t.Fatalf("Assemble() error = %v", err)
	}

	if len(result.Items) != 1 {
		t.Fatalf("Items count = %d, want 1 (profile only)", len(result.Items))
	}
	if result.Items[0].Source != SourceProfile {
		t.Errorf("item source = %q, want %q", result.Items[0].Source, SourceProfile)
	}
	if result.Trimmed != 0 {
		t.Errorf("Trimmed = %d, want 0", result.Trimmed)
	}
	if result.ByteCount != len(result.Items[0].Content) {
		t.Errorf("ByteCount = %d, want %d", result.ByteCount, len(result.Items[0].Content))
	}
}

func TestAssemble_UnknownRole_ReturnsError(t *testing.T) {
	t.Parallel()

	profileDir := t.TempDir()
	stateDir := t.TempDir()

	store := NewProfileStore(profileDir)
	knowledgeSvc := service.NewKnowledgeService(stateDir)

	_, err := Assemble(AssemblyInput{Role: "nonexistent"}, store, knowledgeSvc, nil, nil)
	if err == nil {
		t.Fatal("Assemble() with unknown role: expected error, got nil")
	}
}
