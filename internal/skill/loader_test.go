package skill

import (
	"os"
	"path/filepath"
	"testing"
)

const validSkillContent = `---
name: test-skill
description:
  expert: "Expert description for test skill"
  natural: "A friendly description of the test skill"
triggers:
  - "when testing skills"
  - "when validating loaders"
roles:
  - backend
stage: developing
constraint_level: high
---

## Vocabulary

- **term**: a definition

## Anti-Patterns

- Don't do bad things

## Procedure

1. Do the thing
2. Verify the thing

## Output Format

Return results as JSON.

## Evaluation Criteria

- Correctness
- Completeness

## Questions This Skill Answers

- How do I test skill loading?
`

func writeSkillDir(t *testing.T, root, name string, skillContent string) {
	t.Helper()
	dir := filepath.Join(root, name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(skillContent), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestLoad_ValidSkill(t *testing.T) {
	root := t.TempDir()
	writeSkillDir(t, root, "test-skill", validSkillContent)

	store := NewSkillStore(root)
	sk, warnings, err := store.Load("test-skill")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(warnings) != 0 {
		t.Errorf("expected no warnings, got %v", warnings)
	}
	if sk == nil {
		t.Fatal("expected skill, got nil")
	}
	if sk.Frontmatter.Name != "test-skill" {
		t.Errorf("expected name %q, got %q", "test-skill", sk.Frontmatter.Name)
	}
	if sk.Frontmatter.Stage != "developing" {
		t.Errorf("expected stage %q, got %q", "developing", sk.Frontmatter.Stage)
	}
	if sk.Frontmatter.ConstraintLevel != "high" {
		t.Errorf("expected constraint_level %q, got %q", "high", sk.Frontmatter.ConstraintLevel)
	}
	if sk.Frontmatter.Description.Expert != "Expert description for test skill" {
		t.Errorf("unexpected expert description: %q", sk.Frontmatter.Description.Expert)
	}
	if sk.Frontmatter.Description.Natural != "A friendly description of the test skill" {
		t.Errorf("unexpected natural description: %q", sk.Frontmatter.Description.Natural)
	}
	if len(sk.Frontmatter.Triggers) != 2 {
		t.Errorf("expected 2 triggers, got %d", len(sk.Frontmatter.Triggers))
	}
	if len(sk.Frontmatter.Roles) != 1 || sk.Frontmatter.Roles[0] != "backend" {
		t.Errorf("unexpected roles: %v", sk.Frontmatter.Roles)
	}
	if len(sk.Sections) != 6 {
		t.Errorf("expected 6 sections, got %d", len(sk.Sections))
	}
	expectedPath := filepath.Join(root, "test-skill", "SKILL.md")
	absExpected, _ := filepath.Abs(expectedPath)
	if sk.SourcePath != absExpected {
		t.Errorf("expected SourcePath %q, got %q", absExpected, sk.SourcePath)
	}
	if sk.ReferencePaths != nil {
		t.Errorf("expected nil ReferencePaths, got %v", sk.ReferencePaths)
	}
	if sk.ScriptPaths != nil {
		t.Errorf("expected nil ScriptPaths, got %v", sk.ScriptPaths)
	}
}

func TestLoad_MissingSKILLMD(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "my-skill"), 0o755); err != nil {
		t.Fatal(err)
	}

	store := NewSkillStore(root)
	sk, _, err := store.Load("my-skill")
	if err == nil {
		t.Fatal("expected error for missing SKILL.md, got nil")
	}
	if sk != nil {
		t.Errorf("expected nil skill, got %+v", sk)
	}
}

func TestLoad_NameMismatch(t *testing.T) {
	content := `---
name: other-name
description:
  expert: "Expert"
  natural: "Natural"
triggers:
  - "trigger"
roles:
  - backend
stage: developing
constraint_level: high
---

## Vocabulary

- **term**: def

## Anti-Patterns

- bad

## Procedure

1. step

## Output Format

text

## Evaluation Criteria

- correct

## Questions This Skill Answers

- question
`
	root := t.TempDir()
	writeSkillDir(t, root, "my-skill", content)

	store := NewSkillStore(root)
	sk, _, err := store.Load("my-skill")
	if err == nil {
		t.Fatal("expected error for name mismatch, got nil")
	}
	if sk != nil {
		t.Errorf("expected nil skill, got %+v", sk)
	}
}

func TestLoad_InvalidName(t *testing.T) {
	root := t.TempDir()

	store := NewSkillStore(root)
	sk, _, err := store.Load("INVALID_NAME!")
	if err == nil {
		t.Fatal("expected error for invalid name, got nil")
	}
	if sk != nil {
		t.Errorf("expected nil skill, got %+v", sk)
	}
}

func TestLoad_ReferencesDiscovered(t *testing.T) {
	root := t.TempDir()
	writeSkillDir(t, root, "test-skill", validSkillContent)

	refDir := filepath.Join(root, "test-skill", "references")
	if err := os.MkdirAll(refDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(refDir, "guide.md"), []byte("# Guide"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(refDir, "notes.md"), []byte("# Notes"), 0o644); err != nil {
		t.Fatal(err)
	}

	store := NewSkillStore(root)
	sk, _, err := store.Load("test-skill")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(sk.ReferencePaths) != 2 {
		t.Fatalf("expected 2 reference paths, got %d: %v", len(sk.ReferencePaths), sk.ReferencePaths)
	}

	found := make(map[string]bool)
	for _, p := range sk.ReferencePaths {
		found[p] = true
	}
	if !found[filepath.Join("references", "guide.md")] {
		t.Errorf("expected references/guide.md in paths, got %v", sk.ReferencePaths)
	}
	if !found[filepath.Join("references", "notes.md")] {
		t.Errorf("expected references/notes.md in paths, got %v", sk.ReferencePaths)
	}
}

func TestLoad_ReferencesIgnoresNonMD(t *testing.T) {
	root := t.TempDir()
	writeSkillDir(t, root, "test-skill", validSkillContent)

	refDir := filepath.Join(root, "test-skill", "references")
	if err := os.MkdirAll(refDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(refDir, "guide.md"), []byte("# Guide"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(refDir, "image.png"), []byte("fake png"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(refDir, "data.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}

	store := NewSkillStore(root)
	sk, _, err := store.Load("test-skill")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(sk.ReferencePaths) != 1 {
		t.Fatalf("expected 1 reference path (only .md), got %d: %v", len(sk.ReferencePaths), sk.ReferencePaths)
	}
	if sk.ReferencePaths[0] != filepath.Join("references", "guide.md") {
		t.Errorf("expected references/guide.md, got %q", sk.ReferencePaths[0])
	}
}

func TestLoad_ScriptsDiscovered(t *testing.T) {
	root := t.TempDir()
	writeSkillDir(t, root, "test-skill", validSkillContent)

	scriptDir := filepath.Join(root, "test-skill", "scripts")
	if err := os.MkdirAll(scriptDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(scriptDir, "validate.sh"), []byte("#!/bin/bash"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(scriptDir, "check.py"), []byte("print('ok')"), 0o644); err != nil {
		t.Fatal(err)
	}

	store := NewSkillStore(root)
	sk, _, err := store.Load("test-skill")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(sk.ScriptPaths) != 2 {
		t.Fatalf("expected 2 script paths, got %d: %v", len(sk.ScriptPaths), sk.ScriptPaths)
	}

	found := make(map[string]bool)
	for _, p := range sk.ScriptPaths {
		found[p] = true
	}
	if !found[filepath.Join("scripts", "validate.sh")] {
		t.Errorf("expected scripts/validate.sh in paths, got %v", sk.ScriptPaths)
	}
	if !found[filepath.Join("scripts", "check.py")] {
		t.Errorf("expected scripts/check.py in paths, got %v", sk.ScriptPaths)
	}
}

func TestLoad_WarningsReturnedWithoutError(t *testing.T) {
	// Add an unknown section to trigger a warning but not an error.
	content := `---
name: test-skill
description:
  expert: "Expert description"
  natural: "Natural description"
triggers:
  - "trigger"
roles:
  - backend
stage: developing
constraint_level: high
---

## Vocabulary

- **term**: def

## Anti-Patterns

- bad

## Procedure

1. step

## Output Format

text

## Custom Section

This is an unknown section and should produce a warning.

## Evaluation Criteria

- correct

## Questions This Skill Answers

- question
`
	root := t.TempDir()
	writeSkillDir(t, root, "test-skill", content)

	store := NewSkillStore(root)
	sk, warnings, err := store.Load("test-skill")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sk == nil {
		t.Fatal("expected skill, got nil")
	}
	if len(warnings) == 0 {
		t.Fatal("expected at least one warning for unknown section")
	}

	foundWarning := false
	for _, w := range warnings {
		if w.Level == "warning" {
			foundWarning = true
		}
		if w.Level == "error" {
			t.Errorf("unexpected error-level message in warnings: %s", w.Message)
		}
	}
	if !foundWarning {
		t.Errorf("expected warning-level message, got %v", warnings)
	}
}

func TestLoadAll_ValidSkills(t *testing.T) {
	root := t.TempDir()
	writeSkillDir(t, root, "skill-aa", validSkillContent)

	// Write a second skill with its own name.
	content2 := `---
name: skill-bb
description:
  expert: "Expert"
  natural: "Natural"
triggers:
  - "trigger"
roles:
  - frontend
stage: developing
constraint_level: high
---

## Vocabulary

- **term**: def

## Anti-Patterns

- bad

## Procedure

1. step

## Output Format

text

## Evaluation Criteria

- correct

## Questions This Skill Answers

- question
`
	// Fix the first skill to match its directory name.
	content1 := `---
name: skill-aa
description:
  expert: "Expert"
  natural: "Natural"
triggers:
  - "trigger"
roles:
  - backend
stage: developing
constraint_level: high
---

## Vocabulary

- **term**: def

## Anti-Patterns

- bad

## Procedure

1. step

## Output Format

text

## Evaluation Criteria

- correct

## Questions This Skill Answers

- question
`
	// Overwrite skill-aa with matching name.
	writeSkillDir(t, root, "skill-aa", content1)
	writeSkillDir(t, root, "skill-bb", content2)

	store := NewSkillStore(root)
	skills, err := store.LoadAll()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(skills) != 2 {
		t.Fatalf("expected 2 skills, got %d", len(skills))
	}

	names := make(map[string]bool)
	for _, sk := range skills {
		names[sk.Frontmatter.Name] = true
	}
	if !names["skill-aa"] {
		t.Error("missing skill-aa")
	}
	if !names["skill-bb"] {
		t.Error("missing skill-bb")
	}
}

func TestLoadAll_NonExistentRoot(t *testing.T) {
	store := NewSkillStore(filepath.Join(t.TempDir(), "does-not-exist"))
	skills, err := store.LoadAll()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(skills) != 0 {
		t.Errorf("expected empty slice, got %d skills", len(skills))
	}
}

func TestLoadAll_EmptyRoot(t *testing.T) {
	root := t.TempDir()

	store := NewSkillStore(root)
	skills, err := store.LoadAll()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(skills) != 0 {
		t.Errorf("expected empty slice, got %d skills", len(skills))
	}
}

func TestLoadAll_InvalidSkillReturnsError(t *testing.T) {
	root := t.TempDir()

	// Create a skill directory with no SKILL.md.
	if err := os.MkdirAll(filepath.Join(root, "bad-skill"), 0o755); err != nil {
		t.Fatal(err)
	}

	store := NewSkillStore(root)
	skills, err := store.LoadAll()
	if err == nil {
		t.Fatal("expected error for invalid skill, got nil")
	}
	if skills != nil {
		t.Errorf("expected nil skills, got %v", skills)
	}
}

func TestLoadAll_SkipsNonDirectories(t *testing.T) {
	root := t.TempDir()

	content := `---
name: real-skill
description:
  expert: "Expert"
  natural: "Natural"
triggers:
  - "trigger"
roles:
  - backend
stage: developing
constraint_level: high
---

## Vocabulary

- **term**: def

## Anti-Patterns

- bad

## Procedure

1. step

## Output Format

text

## Evaluation Criteria

- correct

## Questions This Skill Answers

- question
`
	writeSkillDir(t, root, "real-skill", content)

	// Create a regular file at root level — should be skipped.
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("# hi"), 0o644); err != nil {
		t.Fatal(err)
	}

	store := NewSkillStore(root)
	skills, err := store.LoadAll()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(skills) != 1 {
		t.Fatalf("expected 1 skill, got %d", len(skills))
	}
	if skills[0].Frontmatter.Name != "real-skill" {
		t.Errorf("expected skill name %q, got %q", "real-skill", skills[0].Frontmatter.Name)
	}
}
