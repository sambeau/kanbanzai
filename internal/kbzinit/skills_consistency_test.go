package kbzinit

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestEmbeddedSkillsMatchAgentSkills verifies that embedded workflow skill
// seeds (internal/kbzinit/skills/<name>/SKILL.md) match their corresponding
// files in .agents/skills/kanbanzai-<name>/SKILL.md after normalizing markers.
func TestEmbeddedSkillsMatchAgentSkills(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot resolve test file path")
	}
	repoRoot := filepath.Join(filepath.Dir(thisFile), "..", "..")

	for _, name := range skillNames {
		embeddedPath := filepath.Join(filepath.Dir(thisFile), "skills", name, "SKILL.md")
		agentPath := filepath.Join(repoRoot, ".agents", "skills", "kanbanzai-"+name, "SKILL.md")

		embedded, err := os.ReadFile(embeddedPath)
		if err != nil {
			t.Fatalf("read embedded skill %q: %v", name, err)
		}
		agent, err := os.ReadFile(agentPath)
		if err != nil {
			if os.IsNotExist(err) {
				continue // no .agents/skills/ counterpart
			}
			t.Fatalf("read agent skill %q: %v", name, err)
		}

		embeddedNorm := normalizeSkillMarkers(string(embedded))
		agentNorm := normalizeSkillMarkers(string(agent))

		if embeddedNorm != agentNorm {
			t.Errorf("skill %q: embedded seed differs from .agents/skills/ counterpart.\n"+
				"Update the embedded seed in internal/kbzinit/skills/%s/SKILL.md to match.\n"+
				"See the dual-write rule in AGENTS.md.",
				name, name)
		}
	}
}

// normalizeSkillMarkers strips managed/version comment markers and YAML
// metadata blocks containing kanbanzai-managed so comparison focuses on body
// content. The embedded seeds have these markers added; the project files may
// not (project files are the source of truth, not install targets).
func normalizeSkillMarkers(content string) string {
	lines := strings.Split(content, "\n")
	var out []string
	skipMeta := false
	for _, line := range lines {
		// Strip comment markers (added by Phase 1 fix).
		if strings.HasPrefix(line, "# kanbanzai-managed:") || strings.HasPrefix(line, "# kanbanzai-version:") {
			continue
		}
		trimmed := strings.TrimSpace(line)
		// Skip YAML metadata: blocks containing kanbanzai-managed (old format).
		if trimmed == "metadata:" {
			skipMeta = true
			continue
		}
		if skipMeta {
			if strings.Contains(trimmed, "kanbanzai-managed") || strings.HasPrefix(trimmed, "version:") || trimmed == "" {
				continue
			}
			skipMeta = false
		}
		out = append(out, line)
	}
	return strings.TrimSpace(strings.Join(out, "\n"))
}

// TestEmbeddedTaskSkillsMatchProjectSkills verifies embedded task-execution
// skill seeds match their .kbz/skills/ counterparts.
func TestEmbeddedTaskSkillsMatchProjectSkills(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot resolve test file path")
	}
	repoRoot := filepath.Join(filepath.Dir(thisFile), "..", "..")

	for _, name := range taskSkillNames {
		embeddedPath := filepath.Join(filepath.Dir(thisFile), "skills", "task-execution", name, "SKILL.md")
		projectPath := filepath.Join(repoRoot, ".kbz", "skills", name, "SKILL.md")

		embedded, err := os.ReadFile(embeddedPath)
		if err != nil {
			t.Fatalf("read embedded task skill %q: %v", name, err)
		}
		project, err := os.ReadFile(projectPath)
		if err != nil {
			t.Fatalf("read project task skill %q: %v", name, err)
		}

		embeddedNorm := normalizeSkillMarkers(string(embedded))
		projectNorm := normalizeSkillMarkers(string(project))

		if embeddedNorm != projectNorm {
			t.Errorf("task skill %q: embedded seed differs from .kbz/skills/ counterpart.\n"+
				"Update internal/kbzinit/skills/task-execution/%s/SKILL.md to match.",
				name, name)
		}
	}
}

// TestEmbeddedRolesMatchProjectRoles verifies embedded role files match their
// .kbz/roles/ counterparts after normalizing version markers.
func TestEmbeddedRolesMatchProjectRoles(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot resolve test file path")
	}
	repoRoot := filepath.Join(filepath.Dir(thisFile), "..", "..")

	entries, err := embeddedRoles.ReadDir("roles")
	if err != nil {
		t.Fatalf("read embedded roles dir: %v", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}
		name := strings.TrimSuffix(entry.Name(), ".yaml")

		embeddedPath := filepath.Join(filepath.Dir(thisFile), "roles", entry.Name())
		projectPath := filepath.Join(repoRoot, ".kbz", "roles", entry.Name())

		embedded, err := os.ReadFile(embeddedPath)
		if err != nil {
			t.Fatalf("read embedded role %q: %v", name, err)
		}
		project, err := os.ReadFile(projectPath)
		if err != nil {
			t.Fatalf("read project role %q: %v", name, err)
		}

		embeddedNorm := normalizeRoleMarkers(string(embedded))
		projectNorm := normalizeRoleMarkers(string(project))

		if embeddedNorm != projectNorm {
			t.Errorf("role %q: embedded seed differs from .kbz/roles/ counterpart.\n"+
				"Update internal/kbzinit/roles/%s.yaml to match.",
				name, name)
		}
	}
}

// normalizeRoleMarkers strips managed metadata blocks containing
// kanbanzai-managed (added to embedded seeds during Phase 2).
func normalizeRoleMarkers(content string) string {
	lines := strings.Split(content, "\n")
	var out []string
	skipMeta := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "metadata:" {
			skipMeta = true
			continue
		}
		if skipMeta {
			if strings.Contains(trimmed, "kanbanzai-managed") || strings.HasPrefix(trimmed, "version:") || trimmed == "" {
				continue
			}
			skipMeta = false
		}
		out = append(out, line)
	}
	return strings.TrimSpace(strings.Join(out, "\n"))
}
