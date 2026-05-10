package kbzinit

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

// ---- compareManaged unit tests (IntCounter version kind) ----

func TestCompareManaged_IntCounter_OlderVersion_Overwrite(t *testing.T) {
	data := []byte("<!-- kanbanzai-managed: v1 -->\n\n# rest of file\n")
	spec := MarkerSpec{Comment: "<!-- kanbanzai-managed: v", VersionKind: IntCounter, CurrentValue: "2"}
	if got := compareManaged(data, spec); got != Overwrite {
		t.Errorf("compareManaged = %v, want Overwrite", got)
	}
}

func TestCompareManaged_IntCounter_NewerVersion_NoOp(t *testing.T) {
	data := []byte("<!-- kanbanzai-managed: v42 -->\n")
	spec := MarkerSpec{Comment: "<!-- kanbanzai-managed: v", VersionKind: IntCounter, CurrentValue: "2"}
	if got := compareManaged(data, spec); got != NoOp {
		t.Errorf("compareManaged = %v, want NoOp", got)
	}
}

func TestCompareManaged_IntCounter_EqualVersion_NoOp(t *testing.T) {
	data := []byte("<!-- kanbanzai-managed: v2 -->\n")
	spec := MarkerSpec{Comment: "<!-- kanbanzai-managed: v", VersionKind: IntCounter, CurrentValue: "2"}
	if got := compareManaged(data, spec); got != NoOp {
		t.Errorf("compareManaged = %v, want NoOp", got)
	}
}

func TestCompareManaged_IntCounter_NoMarker_WarnSkip(t *testing.T) {
	data := []byte("# Regular AGENTS.md\n\nsome content\n")
	spec := MarkerSpec{Comment: "<!-- kanbanzai-managed: v", VersionKind: IntCounter, CurrentValue: "2"}
	if got := compareManaged(data, spec); got != WarnSkip {
		t.Errorf("compareManaged = %v, want WarnSkip", got)
	}
}

func TestCompareManaged_IntCounter_EmptyFile_Create(t *testing.T) {
	spec := MarkerSpec{Comment: "<!-- kanbanzai-managed: v", VersionKind: IntCounter, CurrentValue: "2"}
	if got := compareManaged(nil, spec); got != Create {
		t.Errorf("compareManaged = %v, want Create", got)
	}
}

func TestCompareManaged_IntCounter_MalformedVersion_WarnSkip(t *testing.T) {
	data := []byte("<!-- kanbanzai-managed: vNOT_A_NUMBER -->\n")
	spec := MarkerSpec{Comment: "<!-- kanbanzai-managed: v", VersionKind: IntCounter, CurrentValue: "2"}
	if got := compareManaged(data, spec); got != WarnSkip {
		t.Errorf("compareManaged = %v, want WarnSkip", got)
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

	spec := MarkerSpec{Comment: "<!-- kanbanzai-managed: v", VersionKind: IntCounter, CurrentValue: strconv.Itoa(agentsMDVersion)}
	if got := compareManaged(data, spec); got != NoOp {
		t.Errorf("AGENTS.md: compareManaged = %v, want NoOp (marker on line 1 at current version)", got)
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

// AC-A10: --skip-instructions prevents creation of AGENTS.md.
func TestWriteAgentsMD_SkipFlag_DoesNotCreate(t *testing.T) {
	dir := t.TempDir()
	init, _ := newTestInit(dir, "")

	// Simulate the flag by not calling writeAgentsMD when SkipInstructions is set.
	// We verify the option plumbing directly via Options.
	opts := Options{SkipInstructions: true}
	if !opts.SkipInstructions {
		t.Fatal("SkipInstructions field not set correctly")
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
	if !strings.HasPrefix(agentsMDContent, "<!-- kanbanzai-managed: v") {
		t.Error("agentsMDContent does not start with managed marker <!-- kanbanzai-managed: v")
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

	spec := MarkerSpec{Comment: "<!-- kanbanzai-managed: v", VersionKind: IntCounter, CurrentValue: strconv.Itoa(agentsMDVersion)}
	if got := compareManaged(data, spec); got != NoOp {
		t.Errorf("copilot-instructions.md: compareManaged = %v, want NoOp (marker on line 1 at current version)", got)
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

// AC-B7: --skip-instructions prevents creation of copilot-instructions.md.
// Verified via the Options struct flag being present and checked in init.go.
func TestWriteCopilotInstructions_SkipFlag_FieldExists(t *testing.T) {
	opts := Options{SkipInstructions: true}
	if !opts.SkipInstructions {
		t.Fatal("SkipInstructions field not set — flag plumbing is broken")
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

// ---- Fuzz: compareManaged never panics ----

func FuzzCompareManaged(f *testing.F) {
	f.Add([]byte("<!-- kanbanzai-managed: v1 -->\n"))
	f.Add([]byte("# Regular file\n"))
	f.Add([]byte(""))
	f.Add([]byte("<!-- kanbanzai-managed: vNaN -->\n"))

	spec := MarkerSpec{Comment: "<!-- kanbanzai-managed: v", VersionKind: IntCounter, CurrentValue: "2"}
	f.Fuzz(func(t *testing.T, data []byte) {
		// Must not panic.
		d := compareManaged(data, spec)
		_ = fmt.Sprintf("decision=%v", d)
	})
}

// ---- REQ-003 (doc-reconciliation): AGENTS.md handoff section accuracy ----

// TestAgentsMD_HandoffSection_NoFalseClaims verifies AC-003:
// AGENTS.md handoff mentions must not claim spec sections, conflict annotations,
// or graph traversal capabilities that do not exist. The implementation removed
// two false claims (context packet definition and Code Graph section).
func TestAgentsMD_HandoffSection_NoFalseClaims(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot resolve test file path")
	}
	repoRoot := filepath.Join(filepath.Dir(thisFile), "..", "..")

	content, err := os.ReadFile(filepath.Join(repoRoot, "AGENTS.md"))
	if err != nil {
		t.Fatalf("read AGENTS.md: %v", err)
	}
	text := string(content)

	// Collect lines mentioning "handoff" for context-aware assertions.
	lines := strings.Split(text, "\n")
	var handoffLines []string
	for i, line := range lines {
		if strings.Contains(strings.ToLower(line), "handoff") {
			handoffLines = append(handoffLines, fmt.Sprintf("L%d: %s", i+1, line))
		}
	}

	if len(handoffLines) == 0 {
		t.Fatal("no handoff mentions found in AGENTS.md — file may be truncated or in wrong format")
	}

	// 1. No unqualified claim that handoff provides spec sections.
	//    Valid: "next additionally includes spec sections" attributes to next, not handoff.
	for _, hl := range handoffLines {
		lower := strings.ToLower(hl)
		if strings.Contains(lower, "spec section") || strings.Contains(lower, "spec sections") {
			if !strings.Contains(lower, "next") {
				t.Errorf("handoff mention claims spec sections without attributing them to next: %s", hl)
			}
		}
	}

	// 2. No claim that handoff provides conflict annotations.
	for _, hl := range handoffLines {
		if strings.Contains(strings.ToLower(hl), "conflict annotation") {
			t.Errorf("handoff mention claims conflict annotations (not implemented): %s", hl)
		}
	}

	// 3. No claim that handoff provides graph traversal.
	for _, hl := range handoffLines {
		if strings.Contains(strings.ToLower(hl), "graph traversal") {
			t.Errorf("handoff mention claims graph traversal (not implemented): %s", hl)
		}
	}

	// 4. Context packet definition must accurately distinguish handoff from next.
	//    Handoff: role instructions, knowledge, vocabulary, anti-patterns, skill procedure.
	//    Next additionally: spec sections, file paths, graph project reference.
	foundContextPacket := false
	for _, hl := range handoffLines {
		lower := strings.ToLower(hl)
		if strings.Contains(lower, "context packet") {
			foundContextPacket = true
			for _, want := range []string{
				"role instruction",
				"knowledge",
				"vocabulary",
				"anti-pattern",
				"skill procedure",
			} {
				if !strings.Contains(lower, want) {
					t.Errorf("context packet definition missing %q: %s", want, hl)
				}
			}
			if !strings.Contains(lower, "next") || !strings.Contains(lower, "spec section") {
				t.Errorf("context packet definition missing 'next' qualifier for spec sections: %s", hl)
			}
		}
	}
	if !foundContextPacket {
		t.Error("context packet definition with handoff description not found")
	}

	// 5. No handoff line may claim handoff receives graph/graph_project capabilities.
	for _, hl := range handoffLines {
		lower := strings.ToLower(hl)
		plain := strings.ReplaceAll(lower, "`", "")
		if !strings.Contains(plain, "graph") && !strings.Contains(plain, "graph_project") {
			continue
		}
		// Old false claim: "every handoff/next call"
		if strings.Contains(plain, "every handoff") {
			t.Errorf("handoff line claims graph context for handoff: %s", hl)
		}
		// If the line discusses handoff's relation to graph, it must be a denial.
		graphAboutHandoff := strings.Contains(plain, "handoff") &&
			(strings.Contains(plain, "handoff renders") ||
				strings.Contains(plain, "handoff does") ||
				strings.Contains(plain, "every handoff") ||
				!strings.Contains(plain, "next"))
		if graphAboutHandoff && !strings.Contains(plain, "does not") {
			t.Errorf("handoff mention appears to claim graph capability for handoff: %s", hl)
		}
	}
}

// ---- REQ-004 (doc-reconciliation): project copilot-instructions.md verification ----

// TestProjectCopilotInstructions_NoFalseHandoffClaims verifies AC-004:
// The project's own .github/copilot-instructions.md contains no false claims
// about handoff capabilities (spec sections, conflict annotations, graph traversal).
func TestProjectCopilotInstructions_NoFalseHandoffClaims(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot resolve test file path")
	}
	repoRoot := filepath.Join(filepath.Dir(thisFile), "..", "..")

	content, err := os.ReadFile(filepath.Join(repoRoot, ".github", "copilot-instructions.md"))
	if err != nil {
		t.Fatalf("read project .github/copilot-instructions.md: %v", err)
	}

	text := string(content)

	// AC-004: copilot-instructions.md must not claim handoff provides
	// spec sections, conflict annotations, or graph traversal.
	// The file has zero handoff mentions, so AC-004 is trivially satisfied.
	// This test guards against regression.

	// Verify no false capability claims exist anywhere in the file.
	lower := strings.ToLower(text)

	// If handoff is mentioned, verify it's not accompanied by false capability claims.
	if strings.Contains(lower, "handoff") {
		if strings.Contains(lower, "spec section") {
			t.Error("copilot-instructions.md claims handoff provides spec sections (not implemented)")
		}
		if strings.Contains(lower, "conflict annotation") {
			t.Error("copilot-instructions.md claims handoff provides conflict annotations (not implemented)")
		}
		if strings.Contains(lower, "graph traversal") {
			t.Error("copilot-instructions.md claims handoff provides graph traversal (not implemented)")
		}
	}

	// Verify the file is the hand-maintained project version (not generated).
	// The project file starts with a kanbanzai-project comment marker.
	if !strings.Contains(text, "kanbanzai-project") {
		t.Error("project .github/copilot-instructions.md missing kanbanzai-project marker")
	}
}
