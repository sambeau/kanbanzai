package context

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"gopkg.in/yaml.v3"
)

// writeTestRole writes a role YAML file into dir and returns its path.
func writeTestRole(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name+".yaml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write test role: %v", err)
	}
	return path
}

// writeTestSkill writes a SKILL.md inside baseDir/<name>/ and returns the skill directory.
func writeTestSkill(t *testing.T, baseDir, name, content string) string {
	t.Helper()
	dir := filepath.Join(baseDir, name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("create skill dir: %v", err)
	}
	path := filepath.Join(dir, "SKILL.md")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write test skill: %v", err)
	}
	return dir
}

func TestRefreshRoleLastVerified_ExistingField(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	content := `id: implementer
identity: "A Go implementation specialist"
vocabulary:
    - Go
    - testing
tools:
    - terminal
    - edit_file
last_verified: "2025-01-01T00:00:00Z"
`
	path := writeTestRole(t, dir, "implementer", content)
	now := time.Date(2025, 7, 30, 14, 0, 0, 0, time.UTC)

	if err := RefreshRoleLastVerified(path, now); err != nil {
		t.Fatalf("RefreshRoleLastVerified: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read back: %v", err)
	}

	var role Role
	if err := yaml.Unmarshal(data, &role); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if role.LastVerified != "2025-07-30T14:00:00Z" {
		t.Errorf("last_verified = %q, want %q", role.LastVerified, "2025-07-30T14:00:00Z")
	}
	if role.ID != "implementer" {
		t.Errorf("id = %q, want %q", role.ID, "implementer")
	}
	if role.Identity != "A Go implementation specialist" {
		t.Errorf("identity changed: %q", role.Identity)
	}
	if len(role.Vocabulary) != 2 || role.Vocabulary[0] != "Go" || role.Vocabulary[1] != "testing" {
		t.Errorf("vocabulary changed: %v", role.Vocabulary)
	}
	if len(role.Tools) != 2 || role.Tools[0] != "terminal" || role.Tools[1] != "edit_file" {
		t.Errorf("tools changed: %v", role.Tools)
	}
}

func TestRefreshRoleLastVerified_MissingField(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	content := `id: reviewer
identity: "A code reviewer"
vocabulary:
    - Go
    - review
`
	path := writeTestRole(t, dir, "reviewer", content)
	now := time.Date(2025, 8, 1, 12, 30, 0, 0, time.UTC)

	if err := RefreshRoleLastVerified(path, now); err != nil {
		t.Fatalf("RefreshRoleLastVerified: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read back: %v", err)
	}

	var role Role
	if err := yaml.Unmarshal(data, &role); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if role.LastVerified != "2025-08-01T12:30:00Z" {
		t.Errorf("last_verified = %q, want %q", role.LastVerified, "2025-08-01T12:30:00Z")
	}
	if role.ID != "reviewer" {
		t.Errorf("id = %q, want %q", role.ID, "reviewer")
	}
	if role.Identity != "A code reviewer" {
		t.Errorf("identity changed: %q", role.Identity)
	}
}

func TestRefreshRoleLastVerified_TimestampFormat(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	content := `id: planner
identity: "A planner"
vocabulary:
    - planning
`
	path := writeTestRole(t, dir, "planner", content)

	// Use a non-UTC time to verify conversion to UTC.
	loc := time.FixedZone("EST", -5*60*60)
	now := time.Date(2025, 6, 15, 10, 0, 0, 0, loc) // 15:00 UTC

	if err := RefreshRoleLastVerified(path, now); err != nil {
		t.Fatalf("RefreshRoleLastVerified: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read back: %v", err)
	}

	var role Role
	if err := yaml.Unmarshal(data, &role); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	want := "2025-06-15T15:00:00Z"
	if role.LastVerified != want {
		t.Errorf("last_verified = %q, want %q (UTC conversion)", role.LastVerified, want)
	}

	// Verify it parses as valid RFC3339.
	parsed, err := time.Parse(time.RFC3339, role.LastVerified)
	if err != nil {
		t.Fatalf("last_verified is not valid RFC3339: %v", err)
	}
	if !parsed.Equal(now) {
		t.Errorf("parsed time %v != original %v", parsed, now)
	}
}

func TestRefreshRoleLastVerified_FileNotFound(t *testing.T) {
	t.Parallel()

	err := RefreshRoleLastVerified("/nonexistent/path/role.yaml", time.Now())
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
	if !strings.Contains(err.Error(), "read role file") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRefreshRoleLastVerified_AtomicWrite(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	content := `id: base
identity: "Base role"
vocabulary:
    - general
`
	path := writeTestRole(t, dir, "base", content)
	now := time.Date(2025, 7, 30, 0, 0, 0, 0, time.UTC)

	if err := RefreshRoleLastVerified(path, now); err != nil {
		t.Fatalf("RefreshRoleLastVerified: %v", err)
	}

	// Verify no temp files are left behind.
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read dir: %v", err)
	}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".refresh-") {
			t.Errorf("temp file left behind: %s", e.Name())
		}
	}

	// Verify the file is valid YAML.
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read back: %v", err)
	}
	var role Role
	if err := yaml.Unmarshal(data, &role); err != nil {
		t.Fatalf("written file is not valid YAML: %v", err)
	}
	if role.LastVerified != "2025-07-30T00:00:00Z" {
		t.Errorf("last_verified = %q, want %q", role.LastVerified, "2025-07-30T00:00:00Z")
	}
}

func TestRefreshSkillLastVerified_ExistingField(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	content := `---
name: implement-task
description:
  expert: "Expert desc"
  natural: "Natural desc"
triggers:
  - implement
roles:
  - implementer
stage: developing
constraint_level: high
last_verified: "2025-01-01T00:00:00Z"
---
## Procedure
1. Do the thing.
2. Test the thing.
`
	skillDir := writeTestSkill(t, dir, "implement-task", content)
	now := time.Date(2025, 7, 30, 14, 0, 0, 0, time.UTC)

	if err := RefreshSkillLastVerified(skillDir, now); err != nil {
		t.Fatalf("RefreshSkillLastVerified: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(skillDir, "SKILL.md"))
	if err != nil {
		t.Fatalf("read back: %v", err)
	}

	got := string(data)

	// Verify timestamp updated.
	if !strings.Contains(got, `last_verified: "2025-07-30T14:00:00Z"`) {
		t.Errorf("last_verified not updated in output:\n%s", got)
	}

	// Verify old timestamp removed.
	if strings.Contains(got, "2025-01-01T00:00:00Z") {
		t.Error("old timestamp still present")
	}

	// Verify body preserved.
	if !strings.Contains(got, "## Procedure") {
		t.Error("body heading lost")
	}
	if !strings.Contains(got, "1. Do the thing.") {
		t.Error("body content lost")
	}
}

func TestRefreshSkillLastVerified_MissingField(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	content := `---
name: review-code
description:
  expert: "Expert desc"
  natural: "Natural desc"
triggers:
  - review
roles:
  - reviewer
stage: reviewing
constraint_level: medium
---
## Checklist
- Check tests.
`
	skillDir := writeTestSkill(t, dir, "review-code", content)
	now := time.Date(2025, 8, 1, 9, 0, 0, 0, time.UTC)

	if err := RefreshSkillLastVerified(skillDir, now); err != nil {
		t.Fatalf("RefreshSkillLastVerified: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(skillDir, "SKILL.md"))
	if err != nil {
		t.Fatalf("read back: %v", err)
	}

	got := string(data)

	// Field was inserted.
	if !strings.Contains(got, `last_verified: "2025-08-01T09:00:00Z"`) {
		t.Errorf("last_verified not inserted:\n%s", got)
	}

	// Inserted before closing ---, so frontmatter is still valid.
	if !strings.Contains(got, "constraint_level: medium") {
		t.Error("constraint_level lost")
	}

	// Body preserved.
	if !strings.Contains(got, "## Checklist") {
		t.Error("body heading lost")
	}
	if !strings.Contains(got, "- Check tests.") {
		t.Error("body content lost")
	}

	// Verify the last_verified line appears before the closing ---.
	lines := strings.Split(got, "\n")
	lvIdx := -1
	closingIdx := -1
	dashCount := 0
	for i, line := range lines {
		if strings.TrimSpace(line) == "---" {
			dashCount++
			if dashCount == 2 {
				closingIdx = i
			}
		}
		if strings.HasPrefix(strings.TrimSpace(line), "last_verified:") {
			lvIdx = i
		}
	}
	if lvIdx == -1 {
		t.Fatal("last_verified line not found")
	}
	if closingIdx == -1 {
		t.Fatal("closing --- not found")
	}
	if lvIdx >= closingIdx {
		t.Errorf("last_verified at line %d is not before closing --- at line %d", lvIdx, closingIdx)
	}
}

func TestRefreshSkillLastVerified_BodyPreserved(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	body := `## Procedure
1. Read the task specification.
2. Check out the feature branch.
3. Implement the changes.

## Vocabulary
- Go
- testing
- refactoring

## Anti-Patterns
Don't skip tests.

### Code blocks in body
` + "```go" + `
func main() {
    fmt.Println("hello")
}
` + "```" + `
`

	content := "---\nname: full-skill\ndescription:\n  expert: \"E\"\n  natural: \"N\"\ntriggers:\n  - implement\nroles:\n  - implementer\nstage: developing\nconstraint_level: high\nlast_verified: \"2025-01-01T00:00:00Z\"\n---\n" + body

	skillDir := writeTestSkill(t, dir, "full-skill", content)
	now := time.Date(2025, 7, 30, 14, 0, 0, 0, time.UTC)

	if err := RefreshSkillLastVerified(skillDir, now); err != nil {
		t.Fatalf("RefreshSkillLastVerified: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(skillDir, "SKILL.md"))
	if err != nil {
		t.Fatalf("read back: %v", err)
	}

	got := string(data)

	// All body sections must be preserved exactly.
	for _, want := range []string{
		"## Procedure",
		"1. Read the task specification.",
		"2. Check out the feature branch.",
		"3. Implement the changes.",
		"## Vocabulary",
		"- Go",
		"- testing",
		"- refactoring",
		"## Anti-Patterns",
		"Don't skip tests.",
		"### Code blocks in body",
		`fmt.Println("hello")`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("body content lost: %q not found in output:\n%s", want, got)
		}
	}
}

func TestRefreshSkillLastVerified_NoFrontmatter(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	content := `# Just a plain markdown file
No frontmatter here.
`
	skillDir := writeTestSkill(t, dir, "no-frontmatter", content)

	err := RefreshSkillLastVerified(skillDir, time.Now())
	if err == nil {
		t.Fatal("expected error for file without frontmatter")
	}
	if !strings.Contains(err.Error(), "no YAML frontmatter found") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRefreshSkillLastVerified_SingleDelimiter(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	content := `---
name: broken
`
	skillDir := writeTestSkill(t, dir, "broken", content)

	err := RefreshSkillLastVerified(skillDir, time.Now())
	if err == nil {
		t.Fatal("expected error for file with only one ---")
	}
	if !strings.Contains(err.Error(), "no YAML frontmatter found") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRefreshSkillLastVerified_FileNotFound(t *testing.T) {
	t.Parallel()

	err := RefreshSkillLastVerified("/nonexistent/skill-dir", time.Now())
	if err == nil {
		t.Fatal("expected error for nonexistent skill dir")
	}
	if !strings.Contains(err.Error(), "read skill file") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRefreshSkillLastVerified_TimestampUTC(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	content := `---
name: utc-test
description:
  expert: "E"
  natural: "N"
triggers:
  - test
roles:
  - tester
stage: developing
constraint_level: low
---
Body.
`
	skillDir := writeTestSkill(t, dir, "utc-test", content)

	// Use a non-UTC time to verify conversion.
	loc := time.FixedZone("PST", -8*60*60)
	now := time.Date(2025, 12, 25, 16, 0, 0, 0, loc) // 2025-12-26T00:00:00Z

	if err := RefreshSkillLastVerified(skillDir, now); err != nil {
		t.Fatalf("RefreshSkillLastVerified: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(skillDir, "SKILL.md"))
	if err != nil {
		t.Fatalf("read back: %v", err)
	}

	if !strings.Contains(string(data), `last_verified: "2025-12-26T00:00:00Z"`) {
		t.Errorf("timestamp not in UTC:\n%s", string(data))
	}
}

func TestRefreshRoleLastVerified_InheritedRole(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	content := `id: implementer
inherits: base
identity: "A Go implementation specialist"
vocabulary:
    - Go
    - testing
anti_patterns:
    - name: skip-tests
      detect: "No test file found"
      because: "Tests are required"
      resolve: "Add tests"
tools:
    - terminal
    - edit_file
`
	path := writeTestRole(t, dir, "implementer", content)
	now := time.Date(2025, 7, 30, 14, 0, 0, 0, time.UTC)

	if err := RefreshRoleLastVerified(path, now); err != nil {
		t.Fatalf("RefreshRoleLastVerified: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read back: %v", err)
	}

	var role Role
	if err := yaml.Unmarshal(data, &role); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if role.LastVerified != "2025-07-30T14:00:00Z" {
		t.Errorf("last_verified = %q, want %q", role.LastVerified, "2025-07-30T14:00:00Z")
	}
	if role.Inherits != "base" {
		t.Errorf("inherits = %q, want %q", role.Inherits, "base")
	}
	if len(role.AntiPatterns) != 1 || role.AntiPatterns[0].Name != "skip-tests" {
		t.Errorf("anti_patterns changed: %v", role.AntiPatterns)
	}
	if len(role.Tools) != 2 {
		t.Errorf("tools changed: %v", role.Tools)
	}
}
