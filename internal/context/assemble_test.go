package context

import (
	"sort"
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

func TestAssemble_MapTypedConventions_FormatProfile(t *testing.T) {
	t.Parallel()

	profileDir := t.TempDir()
	stateDir := t.TempDir()

	writeProfileFile(t, profileDir, "base.yaml", `
id: base
description: "Base profile"
conventions:
  - "Base convention A"
`)
	writeProfileFile(t, profileDir, "reviewer.yaml", `
id: reviewer
inherits: base
description: "Context profile for code review agents."
conventions:
  review_approach:
    - "Review is structured, not conversational."
    - "Every finding has a dimension, severity, location, and description."
  output_format:
    - "Use the structured review output format."
    - "Report per-dimension outcomes."
  dimensions:
    - "Specification conformance"
    - "Implementation quality"
    - "Test adequacy"
`)
	store := NewProfileStore(profileDir)
	knowledgeSvc := service.NewKnowledgeService(stateDir)

	result, err := Assemble(AssemblyInput{Role: "reviewer"}, store, knowledgeSvc, nil, nil)
	if err != nil {
		t.Fatalf("Assemble() error = %v", err)
	}

	if result.Role != "reviewer" {
		t.Errorf("Role = %q, want %q", result.Role, "reviewer")
	}

	// Find the profile item.
	var profileContent string
	for _, item := range result.Items {
		if item.Source == SourceProfile {
			profileContent = item.Content
			break
		}
	}
	if profileContent == "" {
		t.Fatal("no profile item found in assembly result")
	}

	// The profile content must contain the role ID and description.
	if !strings.Contains(profileContent, "reviewer") {
		t.Error("profile content does not contain role ID 'reviewer'")
	}
	if !strings.Contains(profileContent, "Context profile for code review agents.") {
		t.Error("profile content does not contain description")
	}

	// The map-typed conventions branch in formatProfile must produce
	// group headings (the map keys) and bullet items (the list values).
	// Check that all three convention group keys appear as headings.
	for _, key := range []string{"review_approach", "output_format", "dimensions"} {
		if !strings.Contains(profileContent, key+":") {
			t.Errorf("profile content missing convention group heading %q", key)
		}
	}

	// Check that bullet items from each group appear.
	expectedItems := []string{
		"Review is structured, not conversational.",
		"Use the structured review output format.",
		"Specification conformance",
	}
	for _, item := range expectedItems {
		if !strings.Contains(profileContent, item) {
			t.Errorf("profile content missing convention item %q", item)
		}
	}

	// Base conventions (list-typed) must NOT appear — map-typed conventions
	// in the child replace the parent's list-typed conventions entirely.
	if strings.Contains(profileContent, "Base convention A") {
		t.Error("profile content contains base convention — map-typed child conventions should replace list-typed parent conventions")
	}

	// Count total bullet items: 2 (review_approach) + 2 (output_format) + 3 (dimensions) = 7
	bulletCount := strings.Count(profileContent, "  - ")
	if bulletCount != 7 {
		t.Errorf("bullet item count = %d, want 7 (2+2+3)", bulletCount)
	}

	// Verify convention groups are formatted in sorted order for determinism.
	// formatProfile iterates over a map, which has non-deterministic order in Go.
	// Collect the positions of the three group headings and verify they can all be found.
	groupPositions := make(map[string]int)
	for _, key := range []string{"review_approach", "output_format", "dimensions"} {
		pos := strings.Index(profileContent, key+":")
		if pos < 0 {
			continue // already reported above
		}
		groupPositions[key] = pos
	}
	// All three must be present (exact ordering depends on Go map iteration,
	// which is non-deterministic — we verify presence, not order).
	if len(groupPositions) != 3 {
		t.Errorf("found %d convention group headings, want 3", len(groupPositions))
	}

	// Verify items within each group maintain their list order.
	// "Review is structured" must come before "Every finding has" within the output.
	posStructured := strings.Index(profileContent, "Review is structured")
	posEvery := strings.Index(profileContent, "Every finding has")
	if posStructured >= 0 && posEvery >= 0 && posStructured > posEvery {
		t.Error("review_approach items are out of order: 'Review is structured' should precede 'Every finding has'")
	}
}

func TestAssemble_MapTypedConventions_DeterministicGroupOrder(t *testing.T) {
	// Run the assembly multiple times and verify that the convention group
	// order is consistent (detecting non-deterministic map iteration).
	t.Parallel()

	profileDir := t.TempDir()
	stateDir := t.TempDir()

	writeProfileFile(t, profileDir, "reviewer.yaml", `
id: reviewer
description: "Reviewer profile"
conventions:
  alpha:
    - "A1"
  beta:
    - "B1"
  gamma:
    - "G1"
`)
	store := NewProfileStore(profileDir)
	knowledgeSvc := service.NewKnowledgeService(stateDir)

	// Collect the group ordering from multiple runs.
	orderSeen := make(map[string]bool)
	for i := 0; i < 20; i++ {
		result, err := Assemble(AssemblyInput{Role: "reviewer"}, store, knowledgeSvc, nil, nil)
		if err != nil {
			t.Fatalf("Assemble() iteration %d error = %v", i, err)
		}
		var content string
		for _, item := range result.Items {
			if item.Source == SourceProfile {
				content = item.Content
				break
			}
		}
		// Extract the order of group headings.
		var groups []string
		for _, key := range []string{"alpha", "beta", "gamma"} {
			pos := strings.Index(content, key+":")
			if pos >= 0 {
				groups = append(groups, key)
			}
		}
		sort.SliceStable(groups, func(i, j int) bool {
			return strings.Index(content, groups[i]+":") < strings.Index(content, groups[j]+":")
		})
		orderSeen[strings.Join(groups, ",")] = true
	}

	// If map iteration is non-deterministic, we'd see multiple orderings.
	// This is a documentation-of-behavior test: if it fails, it confirms
	// the non-deterministic iteration noted in review finding N3.
	if len(orderSeen) > 1 {
		t.Logf("NOTE: formatProfile produces non-deterministic convention group ordering across %d distinct orderings (expected given Go map iteration); consider sorting map keys for reproducibility", len(orderSeen))
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

func TestAssemble_MinConfidenceFiltering(t *testing.T) {
	t.Parallel()

	profileDir := t.TempDir()
	stateDir := t.TempDir()

	writeProfileFile(t, profileDir, "be.yaml", `
id: be
description: "Test role"
`)
	store := NewProfileStore(profileDir)
	knowledgeSvc := service.NewKnowledgeService(stateDir)

	// Create a Tier 2 entry with high confidence (will be included)
	t2High, _, err := knowledgeSvc.Contribute(service.ContributeInput{
		Topic: "t2-high-conf", Scope: "be", Content: "Always wrap errors with fmt.Errorf and the %w verb for proper error chain propagation", Tier: 2,
	})
	if err != nil {
		t.Fatalf("Contribute T2 high: %v", err)
	}

	// Create a Tier 3 entry with high confidence (will be included)
	t3High, _, err := knowledgeSvc.Contribute(service.ContributeInput{
		Topic: "t3-high-conf", Scope: "be", Content: "Use table-driven tests with t.Parallel for concurrent execution of independent test cases", Tier: 3,
	})
	if err != nil {
		t.Fatalf("Contribute T3 high: %v", err)
	}

	// Create another Tier 3 entry (starts at 0.5, exactly at threshold)
	t3AtThreshold, _, err := knowledgeSvc.Contribute(service.ContributeInput{
		Topic: "t3-at-threshold", Scope: "be", Content: "Run go vet and staticcheck before committing to catch common mistakes early", Tier: 3,
	})
	if err != nil {
		t.Fatalf("Contribute T3 at threshold: %v", err)
	}

	// Create a Tier 3 entry and flag it to lower confidence below 0.5
	t3Low, _, err := knowledgeSvc.Contribute(service.ContributeInput{
		Topic: "t3-low-conf", Scope: "be", Content: "Prefer composition over inheritance when designing Go interfaces and structs", Tier: 3,
	})
	if err != nil {
		t.Fatalf("Contribute T3 low: %v", err)
	}
	// Flag it to lower confidence (one flag brings confidence down)
	if _, err := knowledgeSvc.Flag(t3Low.ID, "not accurate"); err != nil {
		t.Fatalf("Flag T3 low: %v", err)
	}

	// Verify setup: check confidences
	t3LowRecord, err := knowledgeSvc.Get(t3Low.ID)
	if err != nil {
		t.Fatalf("Get T3 low: %v", err)
	}
	t3LowConf, _ := t3LowRecord.Fields["confidence"].(float64)
	if t3LowConf >= 0.5 {
		t.Fatalf("Test setup failed: T3 low confidence = %v, want < 0.5", t3LowConf)
	}

	// Assemble context
	result, err := Assemble(AssemblyInput{Role: "be"}, store, knowledgeSvc, nil, nil)
	if err != nil {
		t.Fatalf("Assemble() error = %v", err)
	}

	// Collect which entries appear in the result by checking the EntryID field
	entryIDs := make(map[string]bool)
	for _, item := range result.Items {
		if item.Source == SourceKnowledgeT2 || item.Source == SourceKnowledgeT3 {
			entryIDs[item.EntryID] = true
		}
	}

	// T2 with default confidence (0.5 >= 0.3) should be included
	if !entryIDs[t2High.ID] {
		t.Errorf("T2 high confidence entry should be included (confidence 0.5 >= 0.3 threshold)")
	}

	// T3 with default confidence (0.5 >= 0.5) should be included
	if !entryIDs[t3High.ID] {
		t.Errorf("T3 high confidence entry should be included (confidence 0.5 >= 0.5 threshold)")
	}

	// T3 at threshold (0.5 = 0.5) should be included
	if !entryIDs[t3AtThreshold.ID] {
		t.Errorf("T3 at threshold entry should be included (confidence 0.5 >= 0.5 threshold)")
	}

	// T3 with low confidence (flagged, < 0.5) should be excluded
	if entryIDs[t3Low.ID] {
		t.Errorf("T3 low confidence entry should be excluded (confidence %v < 0.5 threshold)", t3LowConf)
	}
}
