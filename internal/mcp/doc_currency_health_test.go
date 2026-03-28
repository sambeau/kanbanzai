package mcp

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"kanbanzai/internal/service"
	"kanbanzai/internal/storage"
)

// ─── Tier 1: Tool Name Validation ────────────────────────────────────────────

func TestDocCurrencyHealth_DetectsStaleToolName(t *testing.T) {
	t.Parallel()

	repoRoot := t.TempDir()
	writeSkillFile(t, repoRoot, "example.md", "Use `batch_import_documents` to import.\n")

	toolNames := testToolNameSet()
	checker := DocCurrencyHealthChecker(toolNames, repoRoot, nil, nil)

	report, err := checker()
	if err != nil {
		t.Fatalf("checker error: %v", err)
	}

	if len(report.Warnings) == 0 {
		t.Fatal("expected warning for stale tool name, got none")
	}

	found := false
	for _, w := range report.Warnings {
		if strings.Contains(w.Message, "batch_import_documents") &&
			strings.Contains(w.Message, ".skills/example.md") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected warning about batch_import_documents in .skills/example.md; got: %v", report.Warnings)
	}
}

func TestDocCurrencyHealth_IgnoresValidToolName(t *testing.T) {
	t.Parallel()

	repoRoot := t.TempDir()
	writeSkillFile(t, repoRoot, "example.md", "Use `doc(action: \"refresh\")` to refresh.\n")

	toolNames := testToolNameSet()
	checker := DocCurrencyHealthChecker(toolNames, repoRoot, nil, nil)

	report, err := checker()
	if err != nil {
		t.Fatalf("checker error: %v", err)
	}

	for _, w := range report.Warnings {
		if strings.Contains(w.Message, "doc") && strings.Contains(w.Message, "stale") {
			t.Errorf("valid tool name 'doc' should not be flagged; got: %s", w.Message)
		}
	}
}

func TestDocCurrencyHealth_IgnoresExcludedNames(t *testing.T) {
	t.Parallel()

	repoRoot := t.TempDir()
	content := "Run `go test` and `git status` and `kbz` and `goimports` and `yaml` format.\n"
	writeAgentsMD(t, repoRoot, content)

	toolNames := testToolNameSet()
	checker := DocCurrencyHealthChecker(toolNames, repoRoot, nil, nil)

	report, err := checker()
	if err != nil {
		t.Fatalf("checker error: %v", err)
	}

	for _, w := range report.Warnings {
		for _, excluded := range []string{"go", "git", "kbz", "goimports", "yaml"} {
			if strings.Contains(w.Message, `"`+excluded+`"`) {
				t.Errorf("excluded name %q should not be flagged; got: %s", excluded, w.Message)
			}
		}
	}
}

func TestDocCurrencyHealth_DetectsInAgentsMD(t *testing.T) {
	t.Parallel()

	repoRoot := t.TempDir()
	content := "Call `context_assemble` to gather context.\n"
	writeAgentsMD(t, repoRoot, content)

	toolNames := testToolNameSet()
	checker := DocCurrencyHealthChecker(toolNames, repoRoot, nil, nil)

	report, err := checker()
	if err != nil {
		t.Fatalf("checker error: %v", err)
	}

	found := false
	for _, w := range report.Warnings {
		if strings.Contains(w.Message, "context_assemble") &&
			strings.Contains(w.Message, "AGENTS.md") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected warning about context_assemble in AGENTS.md; got: %v", report.Warnings)
	}
}

func TestDocCurrencyHealth_MultipleStaleRefs(t *testing.T) {
	t.Parallel()

	repoRoot := t.TempDir()
	content := "Use `old_tool_one` and `old_tool_two` and `old_tool_three`.\n"
	writeSkillFile(t, repoRoot, "multi.md", content)

	toolNames := testToolNameSet()
	checker := DocCurrencyHealthChecker(toolNames, repoRoot, nil, nil)

	report, err := checker()
	if err != nil {
		t.Fatalf("checker error: %v", err)
	}

	staleNames := make(map[string]bool)
	for _, w := range report.Warnings {
		for _, name := range []string{"old_tool_one", "old_tool_two", "old_tool_three"} {
			if strings.Contains(w.Message, name) {
				staleNames[name] = true
			}
		}
	}

	for _, name := range []string{"old_tool_one", "old_tool_two", "old_tool_three"} {
		if !staleNames[name] {
			t.Errorf("expected warning for %q, not found in warnings: %v", name, report.Warnings)
		}
	}
}

func TestDocCurrencyHealth_ActionInvocationSyntax(t *testing.T) {
	t.Parallel()

	repoRoot := t.TempDir()
	// "entity" is a valid tool, "fake_tool" is not.
	content := "Call entity(action: \"list\") and fake_tool(action: \"do\").\n"
	writeSkillFile(t, repoRoot, "action.md", content)

	toolNames := testToolNameSet()
	checker := DocCurrencyHealthChecker(toolNames, repoRoot, nil, nil)

	report, err := checker()
	if err != nil {
		t.Fatalf("checker error: %v", err)
	}

	foundFake := false
	foundEntity := false
	for _, w := range report.Warnings {
		if strings.Contains(w.Message, "fake_tool") {
			foundFake = true
		}
		if strings.Contains(w.Message, `"entity"`) {
			foundEntity = true
		}
	}

	if !foundFake {
		t.Error("expected warning for fake_tool(action:), not found")
	}
	if foundEntity {
		t.Error("valid tool 'entity' should not be flagged via action invocation syntax")
	}
}

func TestDocCurrencyHealth_WarningCategoryIsDocCurrency(t *testing.T) {
	t.Parallel()

	repoRoot := t.TempDir()
	writeSkillFile(t, repoRoot, "cat.md", "Use `stale_thing` to do stuff.\n")

	toolNames := testToolNameSet()
	checker := DocCurrencyHealthChecker(toolNames, repoRoot, nil, nil)

	report, err := checker()
	if err != nil {
		t.Fatalf("checker error: %v", err)
	}

	if len(report.Warnings) == 0 {
		t.Fatal("expected at least one warning")
	}
	for _, w := range report.Warnings {
		if w.EntityType != "doc_currency" {
			t.Errorf("warning EntityType = %q, want %q", w.EntityType, "doc_currency")
		}
	}
}

// ─── Tier 2: Plan Completion Documentation ───────────────────────────────────

func TestDocCurrencyHealth_DetectsMissingProjectStatus(t *testing.T) {
	t.Parallel()

	repoRoot := t.TempDir()
	entityRoot := t.TempDir()
	entitySvc := service.NewEntityService(entityRoot)

	// Write a done plan.
	writePlanRecord(t, entitySvc, "P9-my-cool-plan", "done")

	// AGENTS.md without the slug in Project Status.
	agentsContent := "# Agent Instructions\n\n## Project Status\n\nSome other plans.\n\n## Scope Guard\n\nP9 is complete.\n"
	writeAgentsMD(t, repoRoot, agentsContent)

	toolNames := testToolNameSet()
	checker := DocCurrencyHealthChecker(toolNames, repoRoot, entitySvc, nil)

	report, err := checker()
	if err != nil {
		t.Fatalf("checker error: %v", err)
	}

	found := false
	for _, w := range report.Warnings {
		if strings.Contains(w.Message, "P9-my-cool-plan") &&
			strings.Contains(w.Message, "Project Status") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected warning about missing Project Status mention; got: %v", report.Warnings)
	}
}

func TestDocCurrencyHealth_DetectsMissingScopeGuard(t *testing.T) {
	t.Parallel()

	repoRoot := t.TempDir()
	entityRoot := t.TempDir()
	entitySvc := service.NewEntityService(entityRoot)

	writePlanRecord(t, entitySvc, "P9-my-cool-plan", "done")

	// AGENTS.md with slug in Project Status but not in Scope Guard.
	agentsContent := "# Agent Instructions\n\n## Project Status\n\nmy-cool-plan is done.\n\n## Scope Guard\n\nNothing here about the plan.\n"
	writeAgentsMD(t, repoRoot, agentsContent)

	toolNames := testToolNameSet()
	checker := DocCurrencyHealthChecker(toolNames, repoRoot, entitySvc, nil)

	report, err := checker()
	if err != nil {
		t.Fatalf("checker error: %v", err)
	}

	found := false
	for _, w := range report.Warnings {
		if strings.Contains(w.Message, "P9-my-cool-plan") &&
			strings.Contains(w.Message, "Scope Guard") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected warning about missing Scope Guard mention; got: %v", report.Warnings)
	}
}

func TestDocCurrencyHealth_DetectsDraftSpec(t *testing.T) {
	t.Parallel()

	repoRoot := t.TempDir()
	entityRoot := t.TempDir()
	docStateRoot := t.TempDir()

	entitySvc := service.NewEntityService(entityRoot)
	docSvc := service.NewDocumentService(docStateRoot, repoRoot)

	planID := "P9-draft-spec-plan"
	writePlanRecord(t, entitySvc, planID, "done")

	feat := createTestFeature(t, entitySvc, planID, "draft-spec-feat")
	advanceToDone(t, entitySvc, feat)

	// Register a spec document owned by the feature, still in draft.
	specPath := "work/spec/my-spec.md"
	writeDocCurrencyFile(t, repoRoot, specPath, "# My Spec\n\nContent.\n")
	result, err := docSvc.SubmitDocument(service.SubmitDocumentInput{
		Path:      specPath,
		Type:      "specification",
		Title:     "My Spec",
		Owner:     feat.ID,
		CreatedBy: "tester",
	})
	if err != nil {
		t.Fatalf("SubmitDocument: %v", err)
	}

	// AGENTS.md mentions the plan in both sections to avoid Tier 2 check 1/2 noise.
	agentsContent := "# Agent Instructions\n\n## Project Status\n\ndraft-spec-plan is done.\n\n## Scope Guard\n\nP9 is listed.\n"
	writeAgentsMD(t, repoRoot, agentsContent)

	toolNames := testToolNameSet()
	checker := DocCurrencyHealthChecker(toolNames, repoRoot, entitySvc, docSvc)

	report, err := checker()
	if err != nil {
		t.Fatalf("checker error: %v", err)
	}

	found := false
	for _, w := range report.Warnings {
		if strings.Contains(w.Message, result.ID) &&
			strings.Contains(w.Message, "draft") &&
			strings.Contains(w.Message, planID) {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected warning about draft spec for done plan; got: %v", report.Warnings)
	}
}

func TestDocCurrencyHealth_IgnoresActivePlan(t *testing.T) {
	t.Parallel()

	repoRoot := t.TempDir()
	entityRoot := t.TempDir()
	entitySvc := service.NewEntityService(entityRoot)

	writePlanRecord(t, entitySvc, "P9-active-plan", "active")

	// AGENTS.md with no mention at all — but since the plan is active, no warning.
	agentsContent := "# Agent Instructions\n\n## Project Status\n\nNothing.\n\n## Scope Guard\n\nNothing.\n"
	writeAgentsMD(t, repoRoot, agentsContent)

	toolNames := testToolNameSet()
	checker := DocCurrencyHealthChecker(toolNames, repoRoot, entitySvc, nil)

	report, err := checker()
	if err != nil {
		t.Fatalf("checker error: %v", err)
	}

	for _, w := range report.Warnings {
		if strings.Contains(w.Message, "P9-active-plan") {
			t.Errorf("active plan should not be flagged; got: %s", w.Message)
		}
	}
}

func TestDocCurrencyHealth_PassesWhenMentioned(t *testing.T) {
	t.Parallel()

	repoRoot := t.TempDir()
	entityRoot := t.TempDir()
	entitySvc := service.NewEntityService(entityRoot)

	writePlanRecord(t, entitySvc, "P9-well-documented", "done")

	agentsContent := "# Agent Instructions\n\n## Project Status\n\nwell-documented is done.\n\n## Scope Guard\n\nP9 is handled.\n"
	writeAgentsMD(t, repoRoot, agentsContent)

	toolNames := testToolNameSet()
	checker := DocCurrencyHealthChecker(toolNames, repoRoot, entitySvc, nil)

	report, err := checker()
	if err != nil {
		t.Fatalf("checker error: %v", err)
	}

	for _, w := range report.Warnings {
		if strings.Contains(w.Message, "P9-well-documented") {
			t.Errorf("well-mentioned plan should not be flagged; got: %s", w.Message)
		}
	}
}

func TestDocCurrencyHealth_ScopeGuardMatchesByPrefix(t *testing.T) {
	t.Parallel()

	repoRoot := t.TempDir()
	entityRoot := t.TempDir()
	entitySvc := service.NewEntityService(entityRoot)

	writePlanRecord(t, entitySvc, "P9-prefix-test", "done")

	// Scope Guard mentions "P9" but not the full slug.
	agentsContent := "# Agent Instructions\n\n## Project Status\n\nprefix-test is done.\n\n## Scope Guard\n\nP9 is handled.\n"
	writeAgentsMD(t, repoRoot, agentsContent)

	toolNames := testToolNameSet()
	checker := DocCurrencyHealthChecker(toolNames, repoRoot, entitySvc, nil)

	report, err := checker()
	if err != nil {
		t.Fatalf("checker error: %v", err)
	}

	for _, w := range report.Warnings {
		if strings.Contains(w.Message, "Scope Guard") && strings.Contains(w.Message, "P9-prefix-test") {
			t.Errorf("plan with prefix P9 in Scope Guard should not be flagged; got: %s", w.Message)
		}
	}
}

// ─── Test helpers ────────────────────────────────────────────────────────────

// testToolNameSet returns the canonical 22-tool set for the Kanbanzai 2.0 server.
func testToolNameSet() map[string]bool {
	return map[string]bool{
		"status":      true,
		"entity":      true,
		"next":        true,
		"finish":      true,
		"handoff":     true,
		"health":      true,
		"server_info": true,
		"decompose":   true,
		"estimate":    true,
		"conflict":    true,
		"knowledge":   true,
		"profile":     true,
		"worktree":    true,
		"merge":       true,
		"pr":          true,
		"branch":      true,
		"cleanup":     true,
		"doc":         true,
		"doc_intel":   true,
		"incident":    true,
		"checkpoint":  true,
		"retro":       true,
	}
}

func writeSkillFile(t *testing.T, repoRoot, name, content string) {
	t.Helper()
	dir := filepath.Join(repoRoot, ".skills")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir .skills: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatalf("write skill file: %v", err)
	}
}

func writeAgentsMD(t *testing.T, repoRoot, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(repoRoot, "AGENTS.md"), []byte(content), 0o644); err != nil {
		t.Fatalf("write AGENTS.md: %v", err)
	}
}

func writeDocCurrencyFile(t *testing.T, repoRoot, relPath, content string) {
	t.Helper()
	full := filepath.Join(repoRoot, relPath)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatalf("mkdir for %s: %v", relPath, err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", relPath, err)
	}
}

// writePlanRecord writes a plan entity record directly via the store, bypassing
// config-dependent CreatePlan so tests work without .kbz/config.yaml.
func writePlanRecord(t *testing.T, entitySvc *service.EntityService, id, status string) {
	t.Helper()
	now := time.Now().UTC().Format(time.RFC3339)
	slug := extractPlanSlug(id)
	record := storage.EntityRecord{
		Type: "plan",
		ID:   id,
		Slug: slug,
		Fields: map[string]any{
			"id":         id,
			"slug":       slug,
			"title":      "Test Plan " + id,
			"status":     status,
			"summary":    "Test plan for doc currency health check",
			"created":    now,
			"created_by": "tester",
			"updated":    now,
		},
	}
	if _, err := entitySvc.Store().Write(record); err != nil {
		t.Fatalf("writePlanRecord(%s): %v", id, err)
	}
}

// extractPlanSlug gets the slug portion from a plan ID like "P9-my-plan" → "my-plan".
func extractPlanSlug(id string) string {
	if idx := strings.Index(id[1:], "-"); idx >= 0 {
		return id[idx+2:]
	}
	return id
}

// createTestFeature creates a feature under the given plan and returns its CreateResult.
func createTestFeature(t *testing.T, entitySvc *service.EntityService, planID, slug string) service.CreateResult {
	t.Helper()
	result, err := entitySvc.CreateFeature(service.CreateFeatureInput{
		Slug:      slug,
		Parent:    planID,
		Summary:   "Test feature " + slug,
		CreatedBy: "tester",
	})
	if err != nil {
		t.Fatalf("CreateFeature(%s): %v", slug, err)
	}
	return result
}

// advanceToDone transitions a feature through the lifecycle states to done.
// The path is: proposed → designing → specifying → dev-planning → developing → reviewing → done.
func advanceToDone(t *testing.T, entitySvc *service.EntityService, feat service.CreateResult) {
	t.Helper()
	transitions := []string{"designing", "specifying", "dev-planning", "developing", "reviewing", "done"}
	for _, status := range transitions {
		if _, err := entitySvc.UpdateStatus(service.UpdateStatusInput{
			Type: "feature", ID: feat.ID, Slug: feat.Slug, Status: status,
		}); err != nil {
			t.Fatalf("advance feature to %s: %v", status, err)
		}
	}
}
