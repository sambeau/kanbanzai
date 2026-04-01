package service

import (
	"testing"
)

// ---------------------------------------------------------------------------
// checkDescriptionPresent
// ---------------------------------------------------------------------------

func TestCheckDescriptionPresent_AllValid(t *testing.T) {
	t.Parallel()
	p := Proposal{Tasks: []ProposedTask{
		{Slug: "task-a", Summary: "Implement login"},
		{Slug: "task-b", Summary: "Add registration form"},
	}}
	findings := checkDescriptionPresent(p)
	if len(findings) != 0 {
		t.Errorf("expected no findings, got %d", len(findings))
	}
}

func TestCheckDescriptionPresent_EmptySummary(t *testing.T) {
	t.Parallel()
	p := Proposal{Tasks: []ProposedTask{
		{Slug: "task-a", Summary: ""},
	}}
	findings := checkDescriptionPresent(p)
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(findings))
	}
	if findings[0].Type != "empty-description" {
		t.Errorf("Type = %q, want %q", findings[0].Type, "empty-description")
	}
	if findings[0].Severity != "error" {
		t.Errorf("Severity = %q, want %q", findings[0].Severity, "error")
	}
	if findings[0].TaskSlug != "task-a" {
		t.Errorf("TaskSlug = %q, want %q", findings[0].TaskSlug, "task-a")
	}
}

func TestCheckDescriptionPresent_WhitespaceOnly(t *testing.T) {
	t.Parallel()
	p := Proposal{Tasks: []ProposedTask{
		{Slug: "task-a", Summary: "   "},
	}}
	findings := checkDescriptionPresent(p)
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding for whitespace-only summary, got %d", len(findings))
	}
	if findings[0].Severity != "error" {
		t.Errorf("Severity = %q, want error", findings[0].Severity)
	}
}

func TestCheckDescriptionPresent_MultipleEmpty(t *testing.T) {
	t.Parallel()
	p := Proposal{Tasks: []ProposedTask{
		{Slug: "task-a", Summary: ""},
		{Slug: "task-b", Summary: "Valid summary here"},
		{Slug: "task-c", Summary: ""},
	}}
	findings := checkDescriptionPresent(p)
	if len(findings) != 2 {
		t.Errorf("expected 2 findings, got %d", len(findings))
	}
}

// ---------------------------------------------------------------------------
// checkTestingCoverage
// ---------------------------------------------------------------------------

func TestCheckTestingCoverage_KeywordInSummary(t *testing.T) {
	t.Parallel()
	p := Proposal{Tasks: []ProposedTask{
		{Slug: "task-a", Summary: "Write integration tests for the API"},
	}}
	findings := checkTestingCoverage(p)
	if len(findings) != 0 {
		t.Errorf("expected no findings, got %d", len(findings))
	}
}

func TestCheckTestingCoverage_KeywordInRationale(t *testing.T) {
	t.Parallel()
	p := Proposal{Tasks: []ProposedTask{
		{Slug: "task-a", Summary: "Implement login", Rationale: "Coverage ensured by unit tests"},
	}}
	findings := checkTestingCoverage(p)
	if len(findings) != 0 {
		t.Errorf("expected no findings, got %d", len(findings))
	}
}

func TestCheckTestingCoverage_NoKeywords(t *testing.T) {
	t.Parallel()
	p := Proposal{Tasks: []ProposedTask{
		{Slug: "task-a", Summary: "Implement login", Rationale: "Needed for auth"},
	}}
	findings := checkTestingCoverage(p)
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(findings))
	}
	if findings[0].Type != "missing-test-coverage" {
		t.Errorf("Type = %q, want missing-test-coverage", findings[0].Type)
	}
	if findings[0].Severity != "warning" {
		t.Errorf("Severity = %q, want warning", findings[0].Severity)
	}
	if findings[0].TaskSlug != "" {
		t.Errorf("TaskSlug = %q, want empty (proposal-level finding)", findings[0].TaskSlug)
	}
}

func TestCheckTestingCoverage_WholeWordOnly(t *testing.T) {
	// "contest" should NOT match keyword "test".
	t.Parallel()
	p := Proposal{Tasks: []ProposedTask{
		{Slug: "task-a", Summary: "Run a contest for users"},
	}}
	findings := checkTestingCoverage(p)
	if len(findings) == 0 {
		t.Error("expected a missing-test-coverage finding (contest is not a keyword match), got none")
	}
}

func TestCheckTestingCoverage_SpecKeyword(t *testing.T) {
	t.Parallel()
	p := Proposal{Tasks: []ProposedTask{
		{Slug: "task-a", Summary: "Implement per spec requirements"},
	}}
	findings := checkTestingCoverage(p)
	if len(findings) != 0 {
		t.Errorf("expected no findings (spec is a keyword), got %d", len(findings))
	}
}

// ---------------------------------------------------------------------------
// checkDependenciesDeclared
// ---------------------------------------------------------------------------

func TestCheckDependenciesDeclared_NoCrossReferences(t *testing.T) {
	t.Parallel()
	p := Proposal{Tasks: []ProposedTask{
		{Slug: "auth", Summary: "Implement authentication service"},
		{Slug: "profile", Summary: "Build user profile page"},
	}}
	findings := checkDependenciesDeclared(p)
	if len(findings) != 0 {
		t.Errorf("expected no findings, got %d", len(findings))
	}
}

func TestCheckDependenciesDeclared_SlugInSummaryNoDep(t *testing.T) {
	t.Parallel()
	p := Proposal{Tasks: []ProposedTask{
		{Slug: "auth", Summary: "Implement authentication after setup-db is ready"},
		{Slug: "setup-db", Summary: "Set up the database"},
	}}
	findings := checkDependenciesDeclared(p)
	if len(findings) != 1 {
		t.Fatalf("expected 1 undeclared-dependency finding, got %d", len(findings))
	}
	if findings[0].Type != "undeclared-dependency" {
		t.Errorf("Type = %q, want undeclared-dependency", findings[0].Type)
	}
	if findings[0].Severity != "warning" {
		t.Errorf("Severity = %q, want warning", findings[0].Severity)
	}
	if findings[0].TaskSlug != "auth" {
		t.Errorf("TaskSlug = %q, want auth", findings[0].TaskSlug)
	}
}

func TestCheckDependenciesDeclared_SlugInRationaleWithDep(t *testing.T) {
	t.Parallel()
	p := Proposal{Tasks: []ProposedTask{
		{Slug: "auth", Summary: "Implement authentication", Rationale: "Requires setup-db",
			DependsOn: []string{"setup-db"}},
		{Slug: "setup-db", Summary: "Set up the database"},
	}}
	findings := checkDependenciesDeclared(p)
	if len(findings) != 0 {
		t.Errorf("expected no findings (dependency declared), got %d", len(findings))
	}
}

func TestCheckDependenciesDeclared_ReverseDep(t *testing.T) {
	// B depends on A satisfies A referencing B in its summary.
	t.Parallel()
	p := Proposal{Tasks: []ProposedTask{
		{Slug: "auth", Summary: "Implement auth, needed by profile"},
		{Slug: "profile", Summary: "Build profile page", DependsOn: []string{"auth"}},
	}}
	findings := checkDependenciesDeclared(p)
	if len(findings) != 0 {
		t.Errorf("expected no findings (reverse dep satisfies), got %d", len(findings))
	}
}

func TestCheckDependenciesDeclared_CaseInsensitive(t *testing.T) {
	t.Parallel()
	p := Proposal{Tasks: []ProposedTask{
		{Slug: "auth", Summary: "Implement auth after Setup-Database is ready"},
		{Slug: "setup-database", Summary: "Set up database"},
	}}
	findings := checkDependenciesDeclared(p)
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding (case-insensitive match), got %d", len(findings))
	}
	if findings[0].TaskSlug != "auth" {
		t.Errorf("TaskSlug = %q, want auth", findings[0].TaskSlug)
	}
}

func TestCheckDependenciesDeclared_WordBoundary(t *testing.T) {
	// slug "api" should not match word "capital".
	t.Parallel()
	p := Proposal{Tasks: []ProposedTask{
		{Slug: "frontend", Summary: "Build the capital city frontend"},
		{Slug: "api", Summary: "Implement the API service"},
	}}
	findings := checkDependenciesDeclared(p)
	if len(findings) != 0 {
		t.Errorf("expected no findings (api not in capital), got %d: %v", len(findings), findings)
	}
}

// ---------------------------------------------------------------------------
// checkOrphanTasks
// ---------------------------------------------------------------------------

func TestCheckOrphanTasks_NoDependencies(t *testing.T) {
	t.Parallel()
	p := Proposal{Tasks: []ProposedTask{
		{Slug: "task-a", Summary: "Do A"},
		{Slug: "task-b", Summary: "Do B"},
	}}
	findings := checkOrphanTasks(p)
	if len(findings) != 0 {
		t.Errorf("expected no findings (no deps, skip check), got %d", len(findings))
	}
}

func TestCheckOrphanTasks_AllConnected(t *testing.T) {
	t.Parallel()
	p := Proposal{Tasks: []ProposedTask{
		{Slug: "task-a", Summary: "Do A", DependsOn: []string{"task-b"}},
		{Slug: "task-b", Summary: "Do B"},
	}}
	findings := checkOrphanTasks(p)
	if len(findings) != 0 {
		t.Errorf("expected no findings (all connected), got %d", len(findings))
	}
}

func TestCheckOrphanTasks_DisconnectedTask(t *testing.T) {
	t.Parallel()
	p := Proposal{Tasks: []ProposedTask{
		{Slug: "task-a", Summary: "Do A", DependsOn: []string{"task-b"}},
		{Slug: "task-b", Summary: "Do B"},
		{Slug: "task-c", Summary: "Do C standalone"},
	}}
	findings := checkOrphanTasks(p)
	if len(findings) != 1 {
		t.Fatalf("expected 1 orphan finding, got %d", len(findings))
	}
	if findings[0].TaskSlug != "task-c" {
		t.Errorf("TaskSlug = %q, want task-c", findings[0].TaskSlug)
	}
	if findings[0].Severity != "warning" {
		t.Errorf("Severity = %q, want warning", findings[0].Severity)
	}
}

func TestCheckOrphanTasks_TwoSeparateChains(t *testing.T) {
	t.Parallel()
	p := Proposal{Tasks: []ProposedTask{
		{Slug: "task-a", Summary: "Do A", DependsOn: []string{"task-b"}},
		{Slug: "task-b", Summary: "Do B"},
		{Slug: "task-c", Summary: "Do C", DependsOn: []string{"task-d"}},
		{Slug: "task-d", Summary: "Do D"},
	}}
	findings := checkOrphanTasks(p)
	if len(findings) != 0 {
		t.Errorf("expected no findings (all participate in a chain), got %d", len(findings))
	}
}

func TestCheckOrphanTasks_SingleTask(t *testing.T) {
	t.Parallel()
	p := Proposal{Tasks: []ProposedTask{
		{Slug: "task-a", Summary: "Do A"},
	}}
	findings := checkOrphanTasks(p)
	if len(findings) != 0 {
		t.Errorf("expected no findings for single task, got %d", len(findings))
	}
}

func TestCheckOrphanTasks_CrossFeatureDepIgnored(t *testing.T) {
	t.Parallel()
	p := Proposal{Tasks: []ProposedTask{
		{Slug: "task-a", Summary: "Do A", DependsOn: []string{"external-task"}},
		{Slug: "task-b", Summary: "Do B"},
	}}
	findings := checkOrphanTasks(p)
	if len(findings) != 0 {
		t.Errorf("expected no findings (cross-feature dep ignored, no intra-proposal edges), got %d", len(findings))
	}
}

// ---------------------------------------------------------------------------
// checkSingleAgentSizing
// ---------------------------------------------------------------------------

func TestCheckSingleAgentSizing_SingleVerb(t *testing.T) {
	t.Parallel()
	p := Proposal{Tasks: []ProposedTask{
		{Slug: "task-a", Summary: "Implement user authentication"},
	}}
	findings := checkSingleAgentSizing(p)
	if len(findings) != 0 {
		t.Errorf("expected no findings, got %d", len(findings))
	}
}

func TestCheckSingleAgentSizing_TwoVerbsWithAnd(t *testing.T) {
	t.Parallel()
	p := Proposal{Tasks: []ProposedTask{
		{Slug: "task-a", Summary: "Implement login and add registration"},
	}}
	findings := checkSingleAgentSizing(p)
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(findings))
	}
	if findings[0].Type != "multi-agent-sizing" {
		t.Errorf("Type = %q, want multi-agent-sizing", findings[0].Type)
	}
	if findings[0].Severity != "warning" {
		t.Errorf("Severity = %q, want warning", findings[0].Severity)
	}
}

func TestCheckSingleAgentSizing_TwoVerbsWithSemicolon(t *testing.T) {
	t.Parallel()
	p := Proposal{Tasks: []ProposedTask{
		{Slug: "task-a", Summary: "Refactor authentication module; migrate legacy sessions"},
	}}
	findings := checkSingleAgentSizing(p)
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding (semicolon separator), got %d", len(findings))
	}
}

func TestCheckSingleAgentSizing_TwoVerbsWithAsWellAs(t *testing.T) {
	t.Parallel()
	p := Proposal{Tasks: []ProposedTask{
		{Slug: "task-a", Summary: "Build the CI pipeline as well as create Docker images"},
	}}
	findings := checkSingleAgentSizing(p)
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(findings))
	}
}

func TestCheckSingleAgentSizing_NounConjunction(t *testing.T) {
	// "and" joins nouns, not verb clauses — only 1 action verb.
	t.Parallel()
	p := Proposal{Tasks: []ProposedTask{
		{Slug: "task-a", Summary: "Implement request and response handling"},
	}}
	findings := checkSingleAgentSizing(p)
	if len(findings) != 0 {
		t.Errorf("expected no finding (and joins nouns not verb clauses), got %d", len(findings))
	}
}

func TestCheckSingleAgentSizing_VerbNotInList(t *testing.T) {
	// "verify" is not in the action verb list.
	t.Parallel()
	p := Proposal{Tasks: []ProposedTask{
		{Slug: "task-a", Summary: "Implement the service and verify the output"},
	}}
	findings := checkSingleAgentSizing(p)
	if len(findings) != 0 {
		t.Errorf("expected no finding (verify not in action verb list), got %d: %v", len(findings), findings)
	}
}

func TestCheckSingleAgentSizing_SubstringNonMatch(t *testing.T) {
	// "irreplaceable" should NOT match verb "replace".
	t.Parallel()
	p := Proposal{Tasks: []ProposedTask{
		{Slug: "task-a", Summary: "Implement irreplaceable core logic"},
	}}
	findings := checkSingleAgentSizing(p)
	if len(findings) != 0 {
		t.Errorf("expected no finding (irreplaceable is not a verb), got %d", len(findings))
	}
}

func TestCheckSingleAgentSizing_SetUpPhrase(t *testing.T) {
	t.Parallel()
	p := Proposal{Tasks: []ProposedTask{
		{Slug: "task-a", Summary: "Set up the environment and implement the service"},
	}}
	findings := checkSingleAgentSizing(p)
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding (set up + implement), got %d", len(findings))
	}
}
