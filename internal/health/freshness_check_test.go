package health

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

var checkNow = time.Date(2025, 7, 30, 12, 0, 0, 0, time.UTC)

// ── ParseRoleLastVerified ────────────────────────────────────────────────────

func TestParseRoleLastVerified_Valid(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "implementer.yaml")
	content := `id: implementer
identity: "A Go implementer"
vocabulary:
  - "Go"
last_verified: "2025-06-15T12:00:00Z"
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := ParseRoleLastVerified(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := time.Date(2025, 6, 15, 12, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestParseRoleLastVerified_Missing(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "basic.yaml")
	content := `id: basic
identity: "A basic role"
vocabulary:
  - "Go"
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := ParseRoleLastVerified(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !got.IsZero() {
		t.Fatalf("expected zero time, got %v", got)
	}
}

// ── ParseSkillLastVerified ───────────────────────────────────────────────────

func TestParseSkillLastVerified_Valid(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	skillDir := filepath.Join(dir, "implement-task")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatal(err)
	}
	content := `---
name: implement-task
description:
  expert: Implement a task
  natural: Implement a task
triggers:
  - implement
roles:
  - implementer
stage: developing
constraint_level: high
last_verified: "2025-06-15T12:00:00Z"
---
## Procedure
Do the thing.
`
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := ParseSkillLastVerified(skillDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := time.Date(2025, 6, 15, 12, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestParseSkillLastVerified_Missing(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	skillDir := filepath.Join(dir, "basic-skill")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatal(err)
	}
	content := `---
name: basic-skill
description:
  expert: A basic skill
  natural: A basic skill
triggers:
  - basic
roles:
  - implementer
stage: developing
constraint_level: high
---
## Procedure
Do the thing.
`
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := ParseSkillLastVerified(skillDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !got.IsZero() {
		t.Fatalf("expected zero time, got %v", got)
	}
}

// ── CheckRoleFreshness ──────────────────────────────────────────────────────

func TestCheckRoleFreshness_StaleRole(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	// Verified 45 days ago, 30-day window → 15 days overdue.
	staleTime := checkNow.AddDate(0, 0, -45).Format(time.RFC3339)
	content := "id: implementer\nidentity: \"A Go implementer\"\nvocabulary:\n  - \"Go\"\nlast_verified: \"" + staleTime + "\"\n"
	if err := os.WriteFile(filepath.Join(dir, "implementer.yaml"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	result := CheckRoleFreshness(dir, 30, checkNow)
	if len(result.Issues) != 1 {
		t.Fatalf("expected 1 issue, got %d", len(result.Issues))
	}
	issue := result.Issues[0]
	if issue.Severity != SeverityWarning {
		t.Fatalf("expected warning, got %v", issue.Severity)
	}
	if issue.EntityID != "implementer" {
		t.Fatalf("expected entity implementer, got %q", issue.EntityID)
	}
	// 45 - 30 = 15 days overdue.
	if want := "15 days overdue"; !contains(issue.Message, want) {
		t.Fatalf("message %q should contain %q", issue.Message, want)
	}
}

func TestCheckRoleFreshness_FreshRole(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	freshTime := checkNow.AddDate(0, 0, -10).Format(time.RFC3339)
	content := "id: implementer\nidentity: \"A Go implementer\"\nvocabulary:\n  - \"Go\"\nlast_verified: \"" + freshTime + "\"\n"
	if err := os.WriteFile(filepath.Join(dir, "implementer.yaml"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	result := CheckRoleFreshness(dir, 30, checkNow)
	if len(result.Issues) != 0 {
		t.Fatalf("expected 0 issues, got %d: %v", len(result.Issues), result.Issues)
	}
	if result.Status != SeverityOK {
		t.Fatalf("expected OK status, got %v", result.Status)
	}
}

func TestCheckRoleFreshness_NeverVerifiedRole(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	content := "id: implementer\nidentity: \"A Go implementer\"\nvocabulary:\n  - \"Go\"\n"
	if err := os.WriteFile(filepath.Join(dir, "implementer.yaml"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	result := CheckRoleFreshness(dir, 30, checkNow)
	if len(result.Issues) != 1 {
		t.Fatalf("expected 1 issue, got %d", len(result.Issues))
	}
	issue := result.Issues[0]
	if issue.Severity != SeverityWarning {
		t.Fatalf("expected warning, got %v", issue.Severity)
	}
	if want := "has never been verified"; !contains(issue.Message, want) {
		t.Fatalf("message %q should contain %q", issue.Message, want)
	}
}

func TestCheckRoleFreshness_EmptyDir(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	result := CheckRoleFreshness(dir, 30, checkNow)
	if len(result.Issues) != 0 {
		t.Fatalf("expected 0 issues, got %d", len(result.Issues))
	}
	if result.Status != SeverityOK {
		t.Fatalf("expected OK status, got %v", result.Status)
	}
}

// ── CheckSkillFreshness ─────────────────────────────────────────────────────

func TestCheckSkillFreshness_StaleSkill(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	skillDir := filepath.Join(dir, "implement-task")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// Verified 60 days ago, 30-day window → 30 days overdue.
	staleTime := checkNow.AddDate(0, 0, -60).Format(time.RFC3339)
	content := "---\nname: implement-task\ndescription:\n  expert: Implement\n  natural: Implement\ntriggers:\n  - implement\nroles:\n  - implementer\nstage: developing\nconstraint_level: high\nlast_verified: \"" + staleTime + "\"\n---\n## Procedure\nDo.\n"
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	result := CheckSkillFreshness(dir, 30, checkNow)
	if len(result.Issues) != 1 {
		t.Fatalf("expected 1 issue, got %d", len(result.Issues))
	}
	issue := result.Issues[0]
	if issue.Severity != SeverityWarning {
		t.Fatalf("expected warning, got %v", issue.Severity)
	}
	if issue.EntityID != "implement-task" {
		t.Fatalf("expected entity implement-task, got %q", issue.EntityID)
	}
	if want := "30 days overdue"; !contains(issue.Message, want) {
		t.Fatalf("message %q should contain %q", issue.Message, want)
	}
}

func TestCheckSkillFreshness_FreshSkill(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	skillDir := filepath.Join(dir, "implement-task")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatal(err)
	}
	freshTime := checkNow.AddDate(0, 0, -25).Format(time.RFC3339)
	content := "---\nname: implement-task\ndescription:\n  expert: Implement\n  natural: Implement\ntriggers:\n  - implement\nroles:\n  - implementer\nstage: developing\nconstraint_level: high\nlast_verified: \"" + freshTime + "\"\n---\n## Procedure\nDo.\n"
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	result := CheckSkillFreshness(dir, 30, checkNow)
	if len(result.Issues) != 0 {
		t.Fatalf("expected 0 issues, got %d: %v", len(result.Issues), result.Issues)
	}
	if result.Status != SeverityOK {
		t.Fatalf("expected OK status, got %v", result.Status)
	}
}

func TestCheckSkillFreshness_NeverVerifiedSkill(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	skillDir := filepath.Join(dir, "implement-task")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatal(err)
	}
	content := "---\nname: implement-task\ndescription:\n  expert: Implement\n  natural: Implement\ntriggers:\n  - implement\nroles:\n  - implementer\nstage: developing\nconstraint_level: high\n---\n## Procedure\nDo.\n"
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	result := CheckSkillFreshness(dir, 30, checkNow)
	if len(result.Issues) != 1 {
		t.Fatalf("expected 1 issue, got %d", len(result.Issues))
	}
	issue := result.Issues[0]
	if issue.Severity != SeverityWarning {
		t.Fatalf("expected warning, got %v", issue.Severity)
	}
	if want := "has never been verified"; !contains(issue.Message, want) {
		t.Fatalf("message %q should contain %q", issue.Message, want)
	}
}

// ── ComputeFreshnessSummary ─────────────────────────────────────────────────

func TestComputeFreshnessSummary(t *testing.T) {
	t.Parallel()
	rolesDir := t.TempDir()
	skillsDir := t.TempDir()

	freshTime := checkNow.AddDate(0, 0, -10).Format(time.RFC3339)
	staleTime := checkNow.AddDate(0, 0, -45).Format(time.RFC3339)

	// 3 roles: 2 fresh, 1 stale.
	writeRole(t, rolesDir, "fresh-one", freshTime)
	writeRole(t, rolesDir, "fresh-two", freshTime)
	writeRole(t, rolesDir, "stale-one", staleTime)

	// 4 skills: 3 fresh, 1 never-verified.
	writeSkill(t, skillsDir, "skill-a", freshTime)
	writeSkill(t, skillsDir, "skill-b", freshTime)
	writeSkill(t, skillsDir, "skill-c", freshTime)
	writeSkill(t, skillsDir, "skill-d", "") // no last_verified

	s := ComputeFreshnessSummary(rolesDir, skillsDir, 30, checkNow)

	if s.FreshRoles != 2 {
		t.Errorf("FreshRoles: got %d, want 2", s.FreshRoles)
	}
	if s.StaleRoles != 1 {
		t.Errorf("StaleRoles: got %d, want 1", s.StaleRoles)
	}
	if s.NeverVerifiedRoles != 0 {
		t.Errorf("NeverVerifiedRoles: got %d, want 0", s.NeverVerifiedRoles)
	}
	if s.FreshSkills != 3 {
		t.Errorf("FreshSkills: got %d, want 3", s.FreshSkills)
	}
	if s.StaleSkills != 0 {
		t.Errorf("StaleSkills: got %d, want 0", s.StaleSkills)
	}
	if s.NeverVerifiedSkills != 1 {
		t.Errorf("NeverVerifiedSkills: got %d, want 1", s.NeverVerifiedSkills)
	}
}

// ── helpers ─────────────────────────────────────────────────────────────────

func writeRole(t *testing.T, dir, name, lastVerified string) {
	t.Helper()
	content := "id: " + name + "\nidentity: \"A role\"\nvocabulary:\n  - \"Go\"\n"
	if lastVerified != "" {
		content += "last_verified: \"" + lastVerified + "\"\n"
	}
	if err := os.WriteFile(filepath.Join(dir, name+".yaml"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func writeSkill(t *testing.T, dir, name, lastVerified string) {
	t.Helper()
	skillDir := filepath.Join(dir, name)
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatal(err)
	}
	content := "---\nname: " + name + "\ndescription:\n  expert: A skill\n  natural: A skill\ntriggers:\n  - do\nroles:\n  - implementer\nstage: developing\nconstraint_level: high\n"
	if lastVerified != "" {
		content += "last_verified: \"" + lastVerified + "\"\n"
	}
	content += "---\n## Procedure\nDo.\n"
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
