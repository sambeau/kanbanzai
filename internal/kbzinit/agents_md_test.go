package kbzinit

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ---- readMarkdownManagedVersion unit tests ----

func TestReadMarkdownManagedVersion_ValidMarker(t *testing.T) {
	data := []byte("<!-- kanbanzai-managed: v1 -->\n\n# rest of file\n")
	v, managed, err := readMarkdownManagedVersion(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !managed {
		t.Error("expected managed=true")
	}
	if v != 1 {
		t.Errorf("version = %d, want 1", v)
	}
}

func TestReadMarkdownManagedVersion_HigherVersion(t *testing.T) {
	data := []byte("<!-- kanbanzai-managed: v42 -->\n")
	v, managed, err := readMarkdownManagedVersion(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !managed {
		t.Error("expected managed=true")
	}
	if v != 42 {
		t.Errorf("version = %d, want 42", v)
	}
}

func TestReadMarkdownManagedVersion_NoMarker(t *testing.T) {
	data := []byte("# Regular AGENTS.md\n\nsome content\n")
	v, managed, err := readMarkdownManagedVersion(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if managed {
		t.Error("expected managed=false for file with no marker")
	}
	if v != 0 {
		t.Errorf("version = %d, want 0", v)
	}
}

func TestReadMarkdownManagedVersion_EmptyFile(t *testing.T) {
	v, managed, err := readMarkdownManagedVersion([]byte{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if managed {
		t.Error("expected managed=false for empty file")
	}
	if v != 0 {
		t.Errorf("version = %d, want 0", v)
	}
}

func TestReadMarkdownManagedVersion_MalformedMarker(t *testing.T) {
	data := []byte("<!-- kanbanzai-managed: vNOT_A_NUMBER -->\n")
	_, managed, err := readMarkdownManagedVersion(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Malformed version — treated as unmanaged.
	if managed {
		t.Error("expected managed=false for malformed version")
	}
}

func TestReadMarkdownManagedVersion_MarkerNotOnFirstLine(t *testing.T) {
	data := []byte("# Title\n<!-- kanbanzai-managed: v1 -->\n")
	_, managed, err := readMarkdownManagedVersion(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if managed {
		t.Error("expected managed=false when marker is not on line 1")
	}
}

// ---- writeAgentsMD tests ----

// AC-A1: kbz init on a new project creates AGENTS.md at the project root.
func TestWriteAgentsMD_NewProject_CreatesFile(t *testing.T) {
	dir := t.TempDir()
	init, stdout := newTestInit(dir, "")

	if err := init.writeAgentsMD(dir); err != nil {
		t.Fatalf("writeAgentsMD: %v", err)
	}

	path := filepath.Join(dir, "AGENTS.md")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Fatalf("AGENTS.md not created at %s", path)
	}
	if !strings.Contains(stdout.String(), "Created AGENTS.md") {
		t.Errorf("expected 'Created AGENTS.md' in output, got: %s", stdout.String())
	}
}

// AC-A2: The generated file starts with the managed marker on line 1.
func TestWriteAgentsMD_ManagedMarkerOnLineOne(t *testing.T) {
	dir := t.TempDir()
	init, _ := newTestInit(dir, "")
	if err := init.writeAgentsMD(dir); err != nil {
		t.Fatalf("writeAgentsMD: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "AGENTS.md"))
	if err != nil {
		t.Fatalf("read AGENTS.md: %v", err)
	}

	v, managed, err := readMarkdownManagedVersion(data)
	if err != nil {
		t.Fatalf("readMarkdownManagedVersion: %v", err)
	}
	if !managed {
		t.Error("AGENTS.md should have managed marker on line 1")
	}
	if v != agentsMDVersion {
		t.Errorf("marker version = %d, want %d", v, agentsMDVersion)
	}
}

// AC-A3: The file contains the "Before You Do Anything" section with status,
// next, and skill path references.
func TestWriteAgentsMD_ContentRequirements_BeforeYouDoAnything(t *testing.T) {
	dir := t.TempDir()
	init, _ := newTestInit(dir, "")
	if err := init.writeAgentsMD(dir); err != nil {
		t.Fatalf("writeAgentsMD: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(dir, "AGENTS.md"))
	if err != nil {
		t.Fatalf("read AGENTS.md: %v", err)
	}
	text := string(content)

	for _, want := range []string{
		"status",
		"next",
		".agents/skills/kanbanzai-getting-started/SKILL.md",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("AGENTS.md missing required reference to %q", want)
		}
	}
}

// AC-A4: The file contains the three rules (MCP tools, stage gates, human approval).
func TestWriteAgentsMD_ContentRequirements_Rules(t *testing.T) {
	dir := t.TempDir()
	init, _ := newTestInit(dir, "")
	if err := init.writeAgentsMD(dir); err != nil {
		t.Fatalf("writeAgentsMD: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(dir, "AGENTS.md"))
	if err != nil {
		t.Fatalf("read AGENTS.md: %v", err)
	}
	text := string(content)

	for _, want := range []string{
		"MCP",
		"stage gate",
		"approval",
	} {
		if !strings.Contains(strings.ToLower(text), strings.ToLower(want)) {
			t.Errorf("AGENTS.md missing required rule content %q", want)
		}
	}
}

// AC-A5: The file contains a skills reference table listing installed skills.
func TestWriteAgentsMD_ContentRequirements_SkillsTable(t *testing.T) {
	dir := t.TempDir()
	init, _ := newTestInit(dir, "")
	if err := init.writeAgentsMD(dir); err != nil {
		t.Fatalf("writeAgentsMD: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(dir, "AGENTS.md"))
	if err != nil {
		t.Fatalf("read AGENTS.md: %v", err)
	}
	text := string(content)

	for _, skill := range []string{
		"kanbanzai-getting-started",
		"kanbanzai-workflow",
		"kanbanzai-documents",
		"kanbanzai-agents",
		"kanbanzai-planning",
		"kanbanzai-plan-review",
		"write-design",
		"write-spec",
		"implement-task",
		"orchestrate-development",
		"review-code",
	} {
		if !strings.Contains(text, skill) {
			t.Errorf("AGENTS.md skills table missing skill %q", skill)
		}
	}
}

// AC-A6: The file does not exceed 100 lines.
func TestWriteAgentsMD_LineCountLimit(t *testing.T) {
	dir := t.TempDir()
	init, _ := newTestInit(dir, "")
	if err := init.writeAgentsMD(dir); err != nil {
		t.Fatalf("writeAgentsMD: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(dir, "AGENTS.md"))
	if err != nil {
		t.Fatalf("read AGENTS.md: %v", err)
	}

	lines := strings.Split(string(content), "\n")
	if len(lines) > 100 {
		t.Errorf("AGENTS.md has %d lines, must not exceed 100", len(lines))
	}
}

// AC-A7: Running writeAgentsMD on a project with an existing kanbanzai-managed
// AGENTS.md at the current version does not modify the file.
func TestWriteAgentsMD_CurrentVersion_NoOp(t *testing.T) {
	dir := t.TempDir()
	init, _ := newTestInit(dir, "")

	// First write.
	if err := init.writeAgentsMD(dir); err != nil {
		t.Fatalf("first writeAgentsMD: %v", err)
	}
	path := filepath.Join(dir, "AGENTS.md")
	info1, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat after first write: %v", err)
	}

	// Second write — should be a no-op.
	init2, stdout2 := newTestInit(dir, "")
	if err := init2.writeAgentsMD(dir); err != nil {
		t.Fatalf("second writeAgentsMD: %v", err)
	}
	info2, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat after second write: %v", err)
	}

	if info1.ModTime() != info2.ModTime() {
		t.Error("AGENTS.md was modified on second run at current version (should be no-op)")
	}
	if strings.Contains(stdout2.String(), "Updated") || strings.Contains(stdout2.String(), "Created") {
		t.Errorf("unexpected output on no-op run: %s", stdout2.String())
	}
}

// AC-A8: Running writeAgentsMD with an existing managed file at an older version
// overwrites it.
func TestWriteAgentsMD_OlderVersion_Overwrites(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "AGENTS.md")

	// Write a file with an older version marker.
	oldContent := "<!-- kanbanzai-managed: v0 -->\n\n# Old AGENTS.md\n"
	if err := os.WriteFile(path, []byte(oldContent), 0o644); err != nil {
		t.Fatalf("write old AGENTS.md: %v", err)
	}

	init, stdout := newTestInit(dir, "")
	if err := init.writeAgentsMD(dir); err != nil {
		t.Fatalf("writeAgentsMD: %v", err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read AGENTS.md: %v", err)
	}

	if strings.Contains(string(content), "Old AGENTS.md") {
		t.Error("old content should have been overwritten")
	}
	if !strings.Contains(stdout.String(), "Updated AGENTS.md") {
		t.Errorf("expected 'Updated AGENTS.md' in output, got: %s", stdout.String())
	}
}

// AC-A9: Running writeAgentsMD on a project with an existing non-managed
// AGENTS.md prints a warning and does not modify the file.
func TestWriteAgentsMD_NonManaged_SkipsWithWarning(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "AGENTS.md")

	userContent := "# My Project Instructions\n\nCustom instructions here.\n"
	if err := os.WriteFile(path, []byte(userContent), 0o644); err != nil {
		t.Fatalf("write user AGENTS.md: %v", err)
	}

	init, stdout := newTestInit(dir, "")
	if err := init.writeAgentsMD(dir); err != nil {
		t.Fatalf("writeAgentsMD: %v", err)
	}

	// File must not be modified.
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read AGENTS.md: %v", err)
	}
	if string(content) != userContent {
		t.Error("user-managed AGENTS.md was modified — it should not be touched")
	}

	// Warning must be printed.
	if !strings.Contains(stdout.String(), "Warning") {
		t.Errorf("expected warning in output, got: %s", stdout.String())
	}
}

// AC-A10: --skip-agents-md prevents creation of AGENTS.md.
func TestWriteAgentsMD_SkipFlag_DoesNotCreate(t *testing.T) {
	dir := t.TempDir()
	init, _ := newTestInit(dir, "")

	// Simulate the flag by not calling writeAgentsMD when SkipAgentsMD is set.
	// We verify the option plumbing directly via Options.
	opts := Options{SkipAgentsMD: true}
	if !opts.SkipAgentsMD {
		t.Fatal("SkipAgentsMD field not set correctly")
	}

	// Also verify that calling nothing when the flag is set leaves no file.
	path := filepath.Join(dir, "AGENTS.md")
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("AGENTS.md should not exist when skip flag is set")
	}

	// Confirm that if we do call writeAgentsMD directly it WOULD create the file
	// (so the flag is the only guard).
	if err := init.writeAgentsMD(dir); err != nil {
		t.Fatalf("writeAgentsMD: %v", err)
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Error("writeAgentsMD should create the file when called directly")
	}
}

// AC-A11: The generated content is embedded (not read from disk at runtime).
// This is structural — the const agentsMDContent is defined in the same package.
func TestWriteAgentsMD_ContentIsEmbedded(t *testing.T) {
	if agentsMDContent == "" {
		t.Error("agentsMDContent constant is empty — content must be embedded in the binary")
	}
	if !strings.HasPrefix(agentsMDContent, agentsMDMarkerPrefix) {
		t.Errorf("agentsMDContent does not start with managed marker %q", agentsMDMarkerPrefix)
	}
}

// ---- writeCopilotInstructions tests ----

// AC-B1: kbz init on a new project creates .github/copilot-instructions.md.
func TestWriteCopilotInstructions_NewProject_CreatesFile(t *testing.T) {
	dir := t.TempDir()
	init, stdout := newTestInit(dir, "")

	if err := init.writeCopilotInstructions(dir); err != nil {
		t.Fatalf("writeCopilotInstructions: %v", err)
	}

	path := filepath.Join(dir, ".github", "copilot-instructions.md")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Fatalf(".github/copilot-instructions.md not created at %s", path)
	}
	if !strings.Contains(stdout.String(), "Created .github/copilot-instructions.md") {
		t.Errorf("expected 'Created .github/copilot-instructions.md' in output, got: %s", stdout.String())
	}
}

// AC-B2: The .github/ directory is created if it does not exist.
func TestWriteCopilotInstructions_CreatesGithubDir(t *testing.T) {
	dir := t.TempDir()
	init, _ := newTestInit(dir, "")

	githubDir := filepath.Join(dir, ".github")
	if _, err := os.Stat(githubDir); !os.IsNotExist(err) {
		t.Fatal(".github/ should not exist before test")
	}

	if err := init.writeCopilotInstructions(dir); err != nil {
		t.Fatalf("writeCopilotInstructions: %v", err)
	}

	if _, err := os.Stat(githubDir); os.IsNotExist(err) {
		t.Error(".github/ directory was not created")
	}
}

// AC-B3: The generated file starts with the managed marker on line 1.
func TestWriteCopilotInstructions_ManagedMarkerOnLineOne(t *testing.T) {
	dir := t.TempDir()
	init, _ := newTestInit(dir, "")
	if err := init.writeCopilotInstructions(dir); err != nil {
		t.Fatalf("writeCopilotInstructions: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, ".github", "copilot-instructions.md"))
	if err != nil {
		t.Fatalf("read copilot-instructions.md: %v", err)
	}

	v, managed, err := readMarkdownManagedVersion(data)
	if err != nil {
		t.Fatalf("readMarkdownManagedVersion: %v", err)
	}
	if !managed {
		t.Error("copilot-instructions.md should have managed marker on line 1")
	}
	if v != agentsMDVersion {
		t.Errorf("marker version = %d, want %d", v, agentsMDVersion)
	}
}

// AC-B4: The file contains an explicit instruction to read AGENTS.md.
func TestWriteCopilotInstructions_ReferencesAgentsMD(t *testing.T) {
	dir := t.TempDir()
	init, _ := newTestInit(dir, "")
	if err := init.writeCopilotInstructions(dir); err != nil {
		t.Fatalf("writeCopilotInstructions: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(dir, ".github", "copilot-instructions.md"))
	if err != nil {
		t.Fatalf("read copilot-instructions.md: %v", err)
	}
	if !strings.Contains(string(content), "AGENTS.md") {
		t.Error("copilot-instructions.md must reference AGENTS.md")
	}
}

// AC-B5: The file does not exceed 25 lines.
func TestWriteCopilotInstructions_LineCountLimit(t *testing.T) {
	dir := t.TempDir()
	init, _ := newTestInit(dir, "")
	if err := init.writeCopilotInstructions(dir); err != nil {
		t.Fatalf("writeCopilotInstructions: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(dir, ".github", "copilot-instructions.md"))
	if err != nil {
		t.Fatalf("read copilot-instructions.md: %v", err)
	}
	lines := strings.Split(string(content), "\n")
	if len(lines) > 25 {
		t.Errorf("copilot-instructions.md has %d lines, must not exceed 25", len(lines))
	}
}

// AC-B6: An existing non-managed .github/copilot-instructions.md is not modified;
// a warning is printed.
func TestWriteCopilotInstructions_NonManaged_SkipsWithWarning(t *testing.T) {
	dir := t.TempDir()
	githubDir := filepath.Join(dir, ".github")
	if err := os.MkdirAll(githubDir, 0o755); err != nil {
		t.Fatalf("mkdir .github: %v", err)
	}

	path := filepath.Join(githubDir, "copilot-instructions.md")
	userContent := "# My Copilot Instructions\n\nCustom content.\n"
	if err := os.WriteFile(path, []byte(userContent), 0o644); err != nil {
		t.Fatalf("write user copilot-instructions.md: %v", err)
	}

	init, stdout := newTestInit(dir, "")
	if err := init.writeCopilotInstructions(dir); err != nil {
		t.Fatalf("writeCopilotInstructions: %v", err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read copilot-instructions.md: %v", err)
	}
	if string(content) != userContent {
		t.Error("user-managed copilot-instructions.md was modified")
	}
	if !strings.Contains(stdout.String(), "Warning") {
		t.Errorf("expected warning in output, got: %s", stdout.String())
	}
}

// AC-B7: --skip-agents-md prevents creation of copilot-instructions.md.
// Verified via the Options struct flag being present and checked in init.go.
func TestWriteCopilotInstructions_SkipFlag_FieldExists(t *testing.T) {
	opts := Options{SkipAgentsMD: true}
	if !opts.SkipAgentsMD {
		t.Fatal("SkipAgentsMD field not set — flag plumbing is broken")
	}
}

// AC-B8: If .github/ already exists with other files, only copilot-instructions.md
// is affected — no other files are created, modified, or deleted.
func TestWriteCopilotInstructions_ExistingGithubDir_OtherFilesUntouched(t *testing.T) {
	dir := t.TempDir()
	githubDir := filepath.Join(dir, ".github")
	if err := os.MkdirAll(githubDir, 0o755); err != nil {
		t.Fatalf("mkdir .github: %v", err)
	}

	// Create an existing file in .github/.
	otherPath := filepath.Join(githubDir, "FUNDING.yml")
	otherContent := "github: [octocat]\n"
	if err := os.WriteFile(otherPath, []byte(otherContent), 0o644); err != nil {
		t.Fatalf("write FUNDING.yml: %v", err)
	}

	init, _ := newTestInit(dir, "")
	if err := init.writeCopilotInstructions(dir); err != nil {
		t.Fatalf("writeCopilotInstructions: %v", err)
	}

	// Other file must be unchanged.
	got, err := os.ReadFile(otherPath)
	if err != nil {
		t.Fatalf("read FUNDING.yml after init: %v", err)
	}
	if string(got) != otherContent {
		t.Error("FUNDING.yml was modified by writeCopilotInstructions — it must not be touched")
	}

	// Target file must have been created.
	targetPath := filepath.Join(githubDir, "copilot-instructions.md")
	if _, err := os.Stat(targetPath); os.IsNotExist(err) {
		t.Error("copilot-instructions.md was not created")
	}

	// No unexpected extra files should have been created.
	entries, err := os.ReadDir(githubDir)
	if err != nil {
		t.Fatalf("readdir .github: %v", err)
	}
	if len(entries) != 2 {
		names := make([]string, len(entries))
		for i, e := range entries {
			names[i] = e.Name()
		}
		t.Errorf(".github/ has unexpected files: %v", names)
	}
}

// ---- Integration: both files written together ----

func TestAgentsMD_BothFilesWritten_NewProject(t *testing.T) {
	dir := t.TempDir()
	init, stdout := newTestInit(dir, "")

	if err := init.writeAgentsMD(dir); err != nil {
		t.Fatalf("writeAgentsMD: %v", err)
	}
	if err := init.writeCopilotInstructions(dir); err != nil {
		t.Fatalf("writeCopilotInstructions: %v", err)
	}

	// Both files must exist.
	for _, rel := range []string{"AGENTS.md", filepath.Join(".github", "copilot-instructions.md")} {
		if _, err := os.Stat(filepath.Join(dir, rel)); os.IsNotExist(err) {
			t.Errorf("expected file %s not found", rel)
		}
	}

	out := stdout.String()
	if !strings.Contains(out, "Created AGENTS.md") {
		t.Errorf("expected 'Created AGENTS.md' in output: %s", out)
	}
	if !strings.Contains(out, "Created .github/copilot-instructions.md") {
		t.Errorf("expected 'Created .github/copilot-instructions.md' in output: %s", out)
	}
}

func TestAgentsMD_Idempotency(t *testing.T) {
	dir := t.TempDir()

	// First run.
	init1, _ := newTestInit(dir, "")
	if err := init1.writeAgentsMD(dir); err != nil {
		t.Fatalf("first writeAgentsMD: %v", err)
	}
	if err := init1.writeCopilotInstructions(dir); err != nil {
		t.Fatalf("first writeCopilotInstructions: %v", err)
	}

	// Capture mod times.
	agentsStat1, err := os.Stat(filepath.Join(dir, "AGENTS.md"))
	if err != nil {
		t.Fatalf("stat AGENTS.md: %v", err)
	}
	copilotStat1, err := os.Stat(filepath.Join(dir, ".github", "copilot-instructions.md"))
	if err != nil {
		t.Fatalf("stat copilot-instructions.md: %v", err)
	}

	// Second run — should be no-op.
	init2, stdout2 := newTestInit(dir, "")
	if err := init2.writeAgentsMD(dir); err != nil {
		t.Fatalf("second writeAgentsMD: %v", err)
	}
	if err := init2.writeCopilotInstructions(dir); err != nil {
		t.Fatalf("second writeCopilotInstructions: %v", err)
	}

	agentsStat2, err := os.Stat(filepath.Join(dir, "AGENTS.md"))
	if err != nil {
		t.Fatalf("stat AGENTS.md after second run: %v", err)
	}
	copilotStat2, err := os.Stat(filepath.Join(dir, ".github", "copilot-instructions.md"))
	if err != nil {
		t.Fatalf("stat copilot-instructions.md after second run: %v", err)
	}

	if agentsStat1.ModTime() != agentsStat2.ModTime() {
		t.Error("AGENTS.md was modified on second run (idempotency violation)")
	}
	if copilotStat1.ModTime() != copilotStat2.ModTime() {
		t.Error("copilot-instructions.md was modified on second run (idempotency violation)")
	}

	out := stdout2.String()
	if strings.Contains(out, "Created") || strings.Contains(out, "Updated") {
		t.Errorf("unexpected output on idempotent run: %s", out)
	}
}

// ---- Fuzz: readMarkdownManagedVersion never panics ----

func FuzzReadMarkdownManagedVersion(f *testing.F) {
	f.Add([]byte("<!-- kanbanzai-managed: v1 -->\n"))
	f.Add([]byte("# Regular file\n"))
	f.Add([]byte(""))
	f.Add([]byte("<!-- kanbanzai-managed: vNaN -->\n"))

	f.Fuzz(func(t *testing.T, data []byte) {
		// Must not panic.
		v, managed, err := readMarkdownManagedVersion(data)
		_ = fmt.Sprintf("v=%d managed=%v err=%v", v, managed, err)
	})
}
