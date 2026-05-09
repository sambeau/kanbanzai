package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// minimalStageBindings is a minimal .kbz/stage-bindings.yaml for fixture use.
const minimalStageBindings = `stage_bindings:
  designing:
    description: Write the design document
    roles: [architect]
    skills: [write-design]
    human_gate: true
    document_type: design
`

// minimalRoleYAML is a minimal role file for fixture use.
const minimalRoleYAML = `id: architect
identity: You are an architect designing systems.
`

// makeDocsFixture creates a temp directory with the minimal structure needed
// to exercise kbz docs sync and check. rolesAndSkillsContent and
// roleIndexContent are placed as the initial region bodies in each target file.
func makeDocsFixture(t *testing.T, rolesAndSkillsContent, roleIndexContent string) string {
	t.Helper()
	dir := t.TempDir()

	kbzDir := filepath.Join(dir, ".kbz")
	rolesDir := filepath.Join(kbzDir, "roles")
	if err := os.MkdirAll(rolesDir, 0755); err != nil {
		t.Fatal(err)
	}
	githubDir := filepath.Join(dir, ".github")
	if err := os.MkdirAll(githubDir, 0755); err != nil {
		t.Fatal(err)
	}

	writeTestFile(t, filepath.Join(kbzDir, "stage-bindings.yaml"), minimalStageBindings)
	writeTestFile(t, filepath.Join(rolesDir, "architect.yaml"), minimalRoleYAML)

	targetContent := func(ras, ri string) string {
		return "# Header\n\nProse before.\n\n" +
			"<!-- registry-gen:begin:roles-and-skills source=.kbz/stage-bindings.yaml -->\n" +
			ras +
			"<!-- registry-gen:end:roles-and-skills -->\n\n" +
			"Prose between.\n\n" +
			"<!-- registry-gen:begin:role-index source=.kbz/roles -->\n" +
			ri +
			"<!-- registry-gen:end:role-index -->\n\n" +
			"Prose after.\n"
	}

	writeTestFile(t, filepath.Join(dir, "CLAUDE.md"), targetContent(rolesAndSkillsContent, roleIndexContent))
	writeTestFile(t, filepath.Join(githubDir, "copilot-instructions.md"), targetContent(rolesAndSkillsContent, roleIndexContent))
	writeTestFile(t, filepath.Join(dir, "README.md"), targetContent(rolesAndSkillsContent, roleIndexContent))
	writeTestFile(t, filepath.Join(dir, "AGENTS.md"), "# AGENTS\n\nHand-authored content.\n")

	return dir
}

func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func readTestFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

// TestDocsSync_ProducesExpectedOutput verifies sync replaces stale region
// content and preserves surrounding prose byte-for-byte (AC-003).
func TestDocsSync_ProducesExpectedOutput(t *testing.T) {
	dir := makeDocsFixture(t, "stale roles-and-skills\n", "stale role-index\n")

	var out bytes.Buffer
	if err := runDocsSync([]string{"--root", dir}, dependencies{stdout: &out}); err != nil {
		t.Fatalf("runDocsSync: %v", err)
	}

	for _, relPath := range docsTargetFiles {
		absPath := filepath.Join(dir, filepath.FromSlash(relPath))
		content := readTestFile(t, absPath)

		// Generated content marker must be present.
		if !strings.Contains(content, "Generated") {
			t.Errorf("%s: expected generated marker in updated region", relPath)
		}
		// Surrounding prose must be preserved.
		if !strings.Contains(content, "Prose before.") {
			t.Errorf("%s: prose before region not preserved", relPath)
		}
		if !strings.Contains(content, "Prose after.") {
			t.Errorf("%s: prose after region not preserved", relPath)
		}
		if !strings.Contains(content, "Prose between.") {
			t.Errorf("%s: prose between regions not preserved", relPath)
		}
		// Stale content must be gone.
		if strings.Contains(content, "stale roles-and-skills") {
			t.Errorf("%s: old stale roles-and-skills content still present", relPath)
		}
		if strings.Contains(content, "stale role-index") {
			t.Errorf("%s: old stale role-index content still present", relPath)
		}
	}
}

// TestDocsCheck_ExitsNonZeroOnStale verifies check returns an error and names
// the stale file when regions contain outdated content (AC-004).
func TestDocsCheck_ExitsNonZeroOnStale(t *testing.T) {
	dir := makeDocsFixture(t, "stale content\n", "stale content\n")

	var out bytes.Buffer
	err := runDocsCheck([]string{"--root", dir}, dependencies{stdout: &out})
	if err == nil {
		t.Fatal("expected non-zero exit for stale regions, got nil")
	}
	output := out.String()
	if !strings.Contains(output, "stale") {
		t.Errorf("expected stale report in output, got: %q", output)
	}
}

// TestDocsCheck_ExitsZeroOnCurrent verifies check returns nil after sync
// has populated all regions (AC-004).
func TestDocsCheck_ExitsZeroOnCurrent(t *testing.T) {
	dir := makeDocsFixture(t, "stale content\n", "stale content\n")

	// Sync first to populate regions with correct content.
	if err := runDocsSync([]string{"--root", dir}, dependencies{stdout: &bytes.Buffer{}}); err != nil {
		t.Fatalf("sync: %v", err)
	}

	var out bytes.Buffer
	if err := runDocsCheck([]string{"--root", dir}, dependencies{stdout: &out}); err != nil {
		t.Fatalf("check after sync: %v\noutput: %s", err, out.String())
	}
}

// TestDocsSync_AgentsMDNotModified verifies AGENTS.md is never written
// by sync mode (AC-008, REQ-011).
func TestDocsSync_AgentsMDNotModified(t *testing.T) {
	const agentsOriginal = "# AGENTS\n\nHand-authored content.\n"
	dir := makeDocsFixture(t, "stale content\n", "stale content\n")

	if err := runDocsSync([]string{"--root", dir}, dependencies{stdout: &bytes.Buffer{}}); err != nil {
		t.Fatalf("sync: %v", err)
	}

	got := readTestFile(t, filepath.Join(dir, "AGENTS.md"))
	if got != agentsOriginal {
		t.Errorf("AGENTS.md was modified\ngot:  %q\nwant: %q", got, agentsOriginal)
	}
}

// TestDocsCheck_Timing verifies check mode completes under 2 seconds (AC-010,
// REQ-NF-002).
func TestDocsCheck_Timing(t *testing.T) {
	dir := makeDocsFixture(t, "stale content\n", "stale content\n")

	// Populate all regions so check has valid content to compare.
	if err := runDocsSync([]string{"--root", dir}, dependencies{stdout: &bytes.Buffer{}}); err != nil {
		t.Fatalf("sync: %v", err)
	}

	start := time.Now()
	if err := runDocsCheck([]string{"--root", dir}, dependencies{stdout: &bytes.Buffer{}}); err != nil {
		t.Fatalf("check: %v", err)
	}
	elapsed := time.Since(start)

	const limit = 2 * time.Second
	if elapsed > limit {
		t.Errorf("check mode took %v, want < %v", elapsed, limit)
	}
}
