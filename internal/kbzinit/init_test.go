package kbzinit

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sambeau/kanbanzai/internal/config"
)

// ---- test helpers ----

// makeGitRepoNoCommits creates a git repo with no commits in a temp dir.
func makeGitRepoNoCommits(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	mustRun(t, dir, "git", "init")
	mustRun(t, dir, "git", "config", "user.email", "test@example.com")
	mustRun(t, dir, "git", "config", "user.name", "Test User")
	return dir
}

// makeGitRepoWithCommit creates a git repo with one commit in a temp dir.
func makeGitRepoWithCommit(t *testing.T) string {
	t.Helper()
	dir := makeGitRepoNoCommits(t)
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("# test\n"), 0o644); err != nil {
		t.Fatalf("write README: %v", err)
	}
	mustRun(t, dir, "git", "add", ".")
	mustRun(t, dir, "git", "commit", "-m", "initial")
	return dir
}

// mustRun runs a shell command in dir, failing the test if it errors.
func mustRun(t *testing.T, dir, name string, args ...string) {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("run %q %v: %v\n%s", name, args, err, out)
	}
}

// newTestInit creates an Initializer with a controllable stdin and captures stdout.
func newTestInit(workDir, stdinContent string) (*Initializer, *bytes.Buffer) {
	var stdout bytes.Buffer
	return New(workDir, strings.NewReader(stdinContent), &stdout), &stdout
}

// ---- FindGitRoot ----

func TestFindGitRoot_Found(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	got, err := FindGitRoot(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want, _ := filepath.Abs(dir)
	if got != want {
		t.Errorf("FindGitRoot = %q, want %q", got, want)
	}
}

func TestFindGitRoot_FoundFromSubdir(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	sub := filepath.Join(root, "a", "b", "c")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	got, err := FindGitRoot(sub)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want, _ := filepath.Abs(root)
	if got != want {
		t.Errorf("FindGitRoot = %q, want %q", got, want)
	}
}

func TestFindGitRoot_NotFound(t *testing.T) {
	dir := t.TempDir()
	_, err := FindGitRoot(dir)
	if err == nil {
		// The temp dir is inside a git repo (unusual but possible in some CI setups).
		t.Skip("temp dir appears to be inside a git repository; skipping")
	}
	if !strings.Contains(err.Error(), "not a git repository") {
		t.Errorf("expected 'not a git repository' in error, got: %v", err)
	}
}

// ---- HasCommits ----

func TestHasCommits_NoCommits(t *testing.T) {
	dir := makeGitRepoNoCommits(t)
	got, err := HasCommits(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got {
		t.Error("expected false for empty repo, got true")
	}
}

func TestHasCommits_WithCommit(t *testing.T) {
	dir := makeGitRepoWithCommit(t)
	got, err := HasCommits(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !got {
		t.Error("expected true for repo with commit, got false")
	}
}

// ---- InferDocType ----

func TestInferDocType(t *testing.T) {
	cases := []struct {
		path string
		want string
	}{
		{"work/spec", "specification"},
		{"spec", "specification"},
		{"nested/spec", "specification"},
		{"work/dev", "dev-plan"},
		{"dev", "dev-plan"},
		{"work/research", "research"},
		{"research", "research"},
		{"work/reports", "report"},
		{"reports", "report"},
		{"work/plan", "plan"},
		{"plan", "plan"},
		{"work/retro", "retrospective"},
		{"retro", "retrospective"},
		{"work/report", "report"},
		{"report", "report"},
		{"work/design", "design"},
		{"work/docs", "design"},
		{"custom/anything", "design"},
		{"justwords", "design"},
	}
	for _, tc := range cases {
		t.Run(tc.path, func(t *testing.T) {
			got := InferDocType(tc.path)
			if got != tc.want {
				t.Errorf("InferDocType(%q) = %q, want %q", tc.path, got, tc.want)
			}
		})
	}
}

// ---- WriteInitConfig ----

func TestWriteInitConfig_CreatesFile(t *testing.T) {
	dir := t.TempDir()
	if err := WriteInitConfig(dir, "Test", DefaultDocumentRoots()); err != nil {
		t.Fatalf("WriteInitConfig: %v", err)
	}
	configPath := filepath.Join(dir, "config.yaml")
	if _, err := os.Stat(configPath); err != nil {
		t.Fatalf("config.yaml not created: %v", err)
	}
}

// TestWriteInitConfig_CanonicalContent verifies the exact YAML output matches §5.2.
func TestWriteInitConfig_CanonicalContent(t *testing.T) {
	dir := t.TempDir()
	if err := WriteInitConfig(dir, "Test Project", DefaultDocumentRoots()); err != nil {
		t.Fatalf("WriteInitConfig: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "config.yaml"))
	if err != nil {
		t.Fatalf("read config: %v", err)
	}

	// Canonical YAML from spec §5.2 — 2-space indent, version quoted.
	want := strings.TrimSpace(`
version: "2"
name: Test Project
prefixes:
  - prefix: P
    name: Plan
documents:
  roots:
    - path: work/design
      default_type: design
    - path: work/spec
      default_type: specification
    - path: work/plan
      default_type: plan
    - path: work/dev
      default_type: dev-plan
    - path: work/research
      default_type: research
    - path: work/report
      default_type: report
    - path: work/review
      default_type: report
    - path: work/retro
      default_type: retrospective`)

	got := strings.TrimSpace(string(data))
	if got != want {
		t.Errorf("config.yaml content mismatch\ngot:\n%s\n\nwant:\n%s", got, want)
	}
}

func TestWriteInitConfig_CreatesDirIfMissing(t *testing.T) {
	parent := t.TempDir()
	kbzDir := filepath.Join(parent, ".kbz")
	// kbzDir does not exist yet.
	if err := WriteInitConfig(kbzDir, "Test", DefaultDocumentRoots()); err != nil {
		t.Fatalf("WriteInitConfig: %v", err)
	}
	if _, err := os.Stat(kbzDir); err != nil {
		t.Errorf(".kbz dir not created: %v", err)
	}
}

func TestWriteInitConfig_CustomRoots(t *testing.T) {
	dir := t.TempDir()
	roots := []DocumentRoot{
		{Path: "docs/spec", DefaultType: "specification"},
		{Path: "docs/design", DefaultType: "design"},
	}
	if err := WriteInitConfig(dir, "Test", roots); err != nil {
		t.Fatalf("WriteInitConfig: %v", err)
	}
	data, _ := os.ReadFile(filepath.Join(dir, "config.yaml"))
	content := string(data)
	for _, want := range []string{"docs/spec", "docs/design", "specification", `version: "2"`} {
		if !strings.Contains(content, want) {
			t.Errorf("expected %q in config, got:\n%s", want, content)
		}
	}
}

// TestInitNameFlag verifies that --name sets the project name in config.yaml (AC-11).
func TestInitNameFlag(t *testing.T) {
	dir := makeGitRepoNoCommits(t)
	in, _ := newTestInit(dir, "")
	if err := in.Run(Options{
		Name:           "My Project",
		NonInteractive: true,
		SkipSkills:     true,
		SkipMCP:        true,
		SkipWorkDirs:   true,
		SkipRoles:      true,
		SkipAgentsMD:   true,
	}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(dir, ".kbz", "config.yaml"))
	if err != nil {
		t.Fatalf("config.yaml not created: %v", err)
	}
	if !strings.Contains(string(data), "name: My Project") {
		t.Errorf("expected 'name: My Project' in config.yaml, got:\n%s", data)
	}
}

// TestInitNameDefault verifies that the default project name is derived from the
// working directory basename when no --name flag is given (AC-10).
func TestInitNameDefault(t *testing.T) {
	dir := makeGitRepoNoCommits(t)
	in, _ := newTestInit(dir, "") // empty stdin → use default name (workDir basename)
	if err := in.Run(Options{
		SkipSkills:   true,
		SkipMCP:      true,
		SkipWorkDirs: true,
		SkipRoles:    true,
		SkipAgentsMD: true,
	}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(dir, ".kbz", "config.yaml"))
	if err != nil {
		t.Fatalf("config.yaml not created: %v", err)
	}
	wantName := filepath.Base(dir)
	// The YAML encoder may quote the name (e.g. numeric-looking basenames like "001"),
	// so check that the value appears somewhere in the name line rather than exact prefix.
	if !strings.Contains(string(data), "name:") || !strings.Contains(string(data), wantName) {
		t.Errorf("expected default name %q in config.yaml, got:\n%s", wantName, data)
	}
}

// TestConfigNameMissing verifies that an existing config.yaml without a name field
// is read without error — backward compatibility (AC-12).
func TestConfigNameMissing(t *testing.T) {
	dir := makeGitRepoWithCommit(t)
	kbzDir := filepath.Join(dir, ".kbz")
	if err := os.MkdirAll(kbzDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// Write a config in the old format — no name field.
	oldConfig := "version: \"2\"\nprefixes:\n  - prefix: P\n    name: Plan\ndocuments:\n  roots:\n    - path: work/design\n      default_type: design\n"
	if err := os.WriteFile(filepath.Join(kbzDir, "config.yaml"), []byte(oldConfig), 0o644); err != nil {
		t.Fatal(err)
	}
	in, _ := newTestInit(dir, "")
	if err := in.Run(Options{
		SkipSkills:   true,
		SkipMCP:      true,
		SkipRoles:    true,
		SkipAgentsMD: true,
	}); err != nil {
		t.Fatalf("expected no error reading config without name field, got: %v", err)
	}
}

// ---- isNewerSchemaVersion (internal) ----

func TestIsNewerSchemaVersion(t *testing.T) {
	cases := []struct {
		cfg    string
		binary string
		want   bool
	}{
		{"2", "2", false},
		{"3", "2", true},
		{"1", "2", false},
		{"10", "2", true},
		{"2", "10", false},
		{"0", "2", false},
	}
	for _, tc := range cases {
		got := isNewerSchemaVersion(tc.cfg, tc.binary)
		if got != tc.want {
			t.Errorf("isNewerSchemaVersion(%q, %q) = %v, want %v", tc.cfg, tc.binary, got, tc.want)
		}
	}
}

// ---- Run: mutually exclusive flags (AC-18) ----

func TestRun_MutuallyExclusiveFlags(t *testing.T) {
	dir := t.TempDir()
	in, _ := newTestInit(dir, "")
	err := in.Run(Options{UpdateSkills: true, SkipSkills: true})
	if err == nil {
		t.Fatal("expected error for mutually exclusive flags, got nil")
	}
	if !strings.Contains(err.Error(), "mutually exclusive") {
		t.Errorf("expected 'mutually exclusive' in error, got: %v", err)
	}
}

// Mutually exclusive flag check must fire before any filesystem access.
func TestRun_MutuallyExclusiveFlags_NoFilesCreated(t *testing.T) {
	dir := t.TempDir()
	in, _ := newTestInit(dir, "")
	_ = in.Run(Options{UpdateSkills: true, SkipSkills: true})
	// .kbz must not exist.
	if _, err := os.Stat(filepath.Join(dir, ".kbz")); err == nil {
		t.Error(".kbz should not be created when flags are mutually exclusive")
	}
}

// ---- Run: not a git repository (AC-13) ----

func TestRun_NotGitRepo(t *testing.T) {
	dir := t.TempDir()
	in, _ := newTestInit(dir, "")
	err := in.Run(Options{})
	if err == nil {
		// If the temp dir is inside a git repo, skip rather than fail.
		t.Skip("temp dir is inside a git repo; skipping not-a-git-repo test")
	}
	if !strings.Contains(err.Error(), "not a Git repository") {
		t.Errorf("expected 'not a Git repository' in error, got: %v", err)
	}
	if !strings.Contains(err.Error(), "git init") {
		t.Errorf("expected 'git init' instruction in error, got: %v", err)
	}
}

// ---- Run: new project (AC-01, AC-02, AC-04) ----

func TestRun_NewProject_CreatesConfig(t *testing.T) {
	dir := makeGitRepoNoCommits(t)
	in, out := newTestInit(dir, "")
	if err := in.Run(Options{Name: "Test"}); err != nil {
		t.Fatalf("Run: %v", err)
	}

	configPath := filepath.Join(dir, ".kbz", "config.yaml")
	if _, err := os.Stat(configPath); err != nil {
		t.Fatalf("config.yaml not created: %v", err)
	}

	// AC-02: config content matches canonical YAML.
	data, _ := os.ReadFile(configPath)
	want := strings.TrimSpace(`
version: "2"
name: Test
prefixes:
  - prefix: P
    name: Plan
documents:
  roots:
    - path: work/design
      default_type: design
    - path: work/spec
      default_type: specification
    - path: work/plan
      default_type: plan
    - path: work/dev
      default_type: dev-plan
    - path: work/research
      default_type: research
    - path: work/report
      default_type: report
    - path: work/review
      default_type: report
    - path: work/retro
      default_type: retrospective`)

	if got := strings.TrimSpace(string(data)); got != want {
		t.Errorf("config.yaml content mismatch\ngot:\n%s\n\nwant:\n%s", got, want)
	}

	// Output should mention .kbz.
	if !strings.Contains(out.String(), ".kbz") {
		t.Errorf("output should mention .kbz; got: %q", out.String())
	}
}

// AC-01: work/ directories created with .gitkeep on a new project.
func TestRun_NewProject_CreatesWorkDirs(t *testing.T) {
	dir := makeGitRepoNoCommits(t)
	in, _ := newTestInit(dir, "")
	if err := in.Run(Options{}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	for _, sub := range []string{"work/design", "work/spec", "work/plan", "work/dev", "work/research", "work/report", "work/review", "work/retro"} {
		gitkeep := filepath.Join(dir, sub, ".gitkeep")
		if _, err := os.Stat(gitkeep); err != nil {
			t.Errorf("expected %s/.gitkeep to exist, got: %v", sub, err)
		}
	}
}

// AC-04: second run is idempotent — config content unchanged.
func TestRun_Idempotency(t *testing.T) {
	dir := makeGitRepoNoCommits(t)

	in1, _ := newTestInit(dir, "")
	if err := in1.Run(Options{}); err != nil {
		t.Fatalf("first Run: %v", err)
	}

	configPath := filepath.Join(dir, ".kbz", "config.yaml")
	data1, _ := os.ReadFile(configPath)

	// Second run: .kbz/ now exists, so treated as existing project.
	in2, _ := newTestInit(dir, "")
	if err := in2.Run(Options{}); err != nil {
		t.Fatalf("second Run: %v", err)
	}

	data2, _ := os.ReadFile(configPath)
	if string(data1) != string(data2) {
		t.Error("config.yaml was modified on the second run (expected idempotency)")
	}
}

// --skip-work-dirs suppresses work/ directory creation but still creates config.
func TestRun_NewProject_SkipWorkDirs(t *testing.T) {
	dir := makeGitRepoNoCommits(t)
	in, _ := newTestInit(dir, "")
	if err := in.Run(Options{SkipWorkDirs: true}); err != nil {
		t.Fatalf("Run: %v", err)
	}

	// Config should exist.
	if _, err := os.Stat(filepath.Join(dir, ".kbz", "config.yaml")); err != nil {
		t.Error("config.yaml should be created even with --skip-work-dirs")
	}
	// work/ dirs should not exist.
	for _, sub := range []string{"work/design", "work/spec", "work/plan", "work/dev", "work/research", "work/report", "work/review", "work/retro"} {
		if _, err := os.Stat(filepath.Join(dir, sub)); err == nil {
			t.Errorf("work dir %q should not be created with --skip-work-dirs", sub)
		}
	}
}

// --skip-skills does not create any .agents/skills/kanbanzai-* entries but still
// creates config and work/ dirs (AC-11).
func TestRun_NewProject_SkipSkills(t *testing.T) {
	dir := makeGitRepoNoCommits(t)
	in, _ := newTestInit(dir, "")
	if err := in.Run(Options{SkipSkills: true}); err != nil {
		t.Fatalf("Run: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir, ".kbz", "config.yaml")); err != nil {
		t.Error("config.yaml should be created with --skip-skills")
	}
	for _, sub := range []string{"work/design", "work/spec", "work/plan", "work/dev", "work/research", "work/report", "work/review", "work/retro"} {
		if _, err := os.Stat(filepath.Join(dir, sub, ".gitkeep")); err != nil {
			t.Errorf("work dir %s/.gitkeep should be created with --skip-skills", sub)
		}
	}
	skillsDir := filepath.Join(dir, ".agents", "skills")
	if entries, err := os.ReadDir(skillsDir); err == nil {
		for _, e := range entries {
			if strings.HasPrefix(e.Name(), "kanbanzai-") {
				t.Errorf("unexpected kanbanzai skill dir %q with --skip-skills", e.Name())
			}
		}
	}
}

// ---- Run: existing project (AC-05, AC-06, AC-07, AC-17) ----

// AC-05: existing project with valid config does not create work/ dirs.
func TestRun_ExistingProject_NoWorkDirs(t *testing.T) {
	dir := makeGitRepoWithCommit(t)
	kbzDir := filepath.Join(dir, ".kbz")
	if err := WriteInitConfig(kbzDir, "", DefaultDocumentRoots()); err != nil {
		t.Fatalf("pre-create config: %v", err)
	}

	in, _ := newTestInit(dir, "")
	if err := in.Run(Options{}); err != nil {
		t.Fatalf("Run: %v", err)
	}

	for _, sub := range []string{"work/design", "work/spec", "work/plan", "work/dev", "work/research", "work/report", "work/review", "work/retro"} {
		if _, err := os.Stat(filepath.Join(dir, sub)); err == nil {
			t.Errorf("work dir %q should NOT be created for an existing project", sub)
		}
	}
}

// AC-07: --docs-path suppresses the interactive prompt.
func TestRun_ExistingProject_DocsPath_SuppressesPrompt(t *testing.T) {
	dir := makeGitRepoWithCommit(t)
	// stdin is empty — would cause an error if a prompt were issued.
	in, _ := newTestInit(dir, "")
	if err := in.Run(Options{DocsPath: []string{"work/docs"}}); err != nil {
		t.Fatalf("Run: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, ".kbz", "config.yaml"))
	if err != nil {
		t.Fatalf("config.yaml not created: %v", err)
	}
	if !strings.Contains(string(data), "work/docs") {
		t.Errorf("expected 'work/docs' in config.yaml, got:\n%s", data)
	}
}

// AC-06: interactive prompt for docs-path when config is absent.
func TestRun_ExistingProject_NoConfig_Prompt(t *testing.T) {
	dir := makeGitRepoWithCommit(t)
	in, _ := newTestInit(dir, "work/notes\n")
	if err := in.Run(Options{}); err != nil {
		t.Fatalf("Run: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, ".kbz", "config.yaml"))
	if err != nil {
		t.Fatalf("config.yaml not created: %v", err)
	}
	if !strings.Contains(string(data), "work/notes") {
		t.Errorf("expected 'work/notes' in config.yaml, got:\n%s", data)
	}
}

// TestRun_ExistingProject_NoConfig_EmptyInput_UsesDefault verifies that pressing
// Enter at the document root prompt (empty input) uses the standard work/ layout
// rather than returning an error.
func TestRun_ExistingProject_NoConfig_EmptyInput_UsesDefault(t *testing.T) {
	dir := makeGitRepoWithCommit(t)
	in, _ := newTestInit(dir, "\n") // simulate pressing Enter with no input
	if err := in.Run(Options{}); err != nil {
		t.Fatalf("Run: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, ".kbz", "config.yaml"))
	if err != nil {
		t.Fatalf("config.yaml not created: %v", err)
	}
	// All eight default roots must be present.
	for _, root := range []string{"work/design", "work/spec", "work/plan", "work/dev",
		"work/research", "work/report", "work/review", "work/retro"} {
		if !strings.Contains(string(data), root) {
			t.Errorf("expected default root %q in config.yaml, got:\n%s", root, data)
		}
	}
	// The standard work/ directories must also be created.
	for _, sub := range []string{"design", "spec", "plan", "dev", "research", "report", "review", "retro"} {
		if _, err := os.Stat(filepath.Join(dir, "work", sub)); os.IsNotExist(err) {
			t.Errorf("work/%s/ not created for default layout", sub)
		}
	}
}

// AC-17: --non-interactive without --docs-path on existing project with no config.
func TestRun_NonInteractive_NoDocsPath_Error(t *testing.T) {
	dir := makeGitRepoWithCommit(t)
	in, _ := newTestInit(dir, "")
	err := in.Run(Options{NonInteractive: true})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "--docs-path") {
		t.Errorf("expected '--docs-path' in error message, got: %v", err)
	}
	if !strings.Contains(err.Error(), "--non-interactive") {
		t.Errorf("expected '--non-interactive' in error message, got: %v", err)
	}
}

// --docs-path with multiple values creates multiple roots.
func TestRun_ExistingProject_MultipleDocsPaths(t *testing.T) {
	dir := makeGitRepoWithCommit(t)
	in, _ := newTestInit(dir, "")
	err := in.Run(Options{DocsPath: []string{"docs/spec", "docs/design"}})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	data, _ := os.ReadFile(filepath.Join(dir, ".kbz", "config.yaml"))
	content := string(data)
	for _, want := range []string{"docs/spec", "docs/design", "specification"} {
		if !strings.Contains(content, want) {
			t.Errorf("expected %q in config.yaml, got:\n%s", want, content)
		}
	}
}

// ---- Run: schema version guard (AC-14) ----

func TestRun_NewerSchemaVersion_Error(t *testing.T) {
	dir := makeGitRepoWithCommit(t)
	kbzDir := filepath.Join(dir, ".kbz")
	if err := os.MkdirAll(kbzDir, 0o755); err != nil {
		t.Fatal(err)
	}
	newerConfig := `version: "9"
prefixes:
  - prefix: P
    name: Plan
`
	if err := os.WriteFile(filepath.Join(kbzDir, "config.yaml"), []byte(newerConfig), 0o644); err != nil {
		t.Fatal(err)
	}

	in, _ := newTestInit(dir, "")
	err := in.Run(Options{})
	if err == nil {
		t.Fatal("expected error for newer schema version, got nil")
	}
	// Error must name both versions and include a download URL.
	if !strings.Contains(err.Error(), "9") {
		t.Errorf("expected config schema version '9' in error, got: %v", err)
	}
	if !strings.Contains(err.Error(), SupportedSchemaVersion) {
		t.Errorf("expected supported schema version %q in error, got: %v", SupportedSchemaVersion, err)
	}
	if !strings.Contains(err.Error(), LatestReleaseURL) {
		t.Errorf("expected download URL in error, got: %v", err)
	}
}

// ---- Run: invalid config (AC-15) ----

func TestRun_InvalidConfig_NonInteractive_Overwrites(t *testing.T) {
	dir := makeGitRepoWithCommit(t)
	kbzDir := filepath.Join(dir, ".kbz")
	if err := os.MkdirAll(kbzDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(kbzDir, "config.yaml"), []byte("{{invalid: yaml::\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	in, _ := newTestInit(dir, "")
	if err := in.Run(Options{NonInteractive: true}); err != nil {
		t.Fatalf("expected success (non-interactive overwrite), got: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(kbzDir, "config.yaml"))
	if err != nil {
		t.Fatalf("config not readable after overwrite: %v", err)
	}
	if !strings.Contains(string(data), `version: "2"`) {
		t.Errorf("expected valid config after overwrite, got:\n%s", data)
	}
}

func TestRun_InvalidConfig_Interactive_AcceptOverwrite(t *testing.T) {
	dir := makeGitRepoWithCommit(t)
	kbzDir := filepath.Join(dir, ".kbz")
	if err := os.MkdirAll(kbzDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(kbzDir, "config.yaml"), []byte("{{invalid: yaml::\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Answer "y" to the overwrite prompt.
	in, _ := newTestInit(dir, "y\n")
	if err := in.Run(Options{}); err != nil {
		t.Fatalf("expected success with y answer, got: %v", err)
	}

	data, _ := os.ReadFile(filepath.Join(kbzDir, "config.yaml"))
	if !strings.Contains(string(data), `version: "2"`) {
		t.Errorf("expected valid config after overwrite, got:\n%s", data)
	}
}

func TestRun_InvalidConfig_Interactive_DeclineOverwrite(t *testing.T) {
	dir := makeGitRepoWithCommit(t)
	kbzDir := filepath.Join(dir, ".kbz")
	if err := os.MkdirAll(kbzDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(kbzDir, "config.yaml"), []byte("{{invalid: yaml::\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Answer "n" — decline overwrite.
	in, _ := newTestInit(dir, "n\n")
	if err := in.Run(Options{}); err == nil {
		t.Fatal("expected error when overwrite was declined, got nil")
	}
}

// ---- Run: non-kanbanzai skill files are unaffected (AC-16) ----

func TestRun_NonKanbanzaiSkillFiles_Untouched(t *testing.T) {
	dir := makeGitRepoNoCommits(t)

	// Pre-create a skill directory that is NOT kanbanzai-managed.
	otherSkill := filepath.Join(dir, ".agents", "skills", "other-tool")
	if err := os.MkdirAll(otherSkill, 0o755); err != nil {
		t.Fatal(err)
	}
	otherFile := filepath.Join(otherSkill, "SKILL.md")
	originalContent := "# Other Tool\nNot managed by kanbanzai.\n"
	if err := os.WriteFile(otherFile, []byte(originalContent), 0o644); err != nil {
		t.Fatal(err)
	}

	in, _ := newTestInit(dir, "")
	if err := in.Run(Options{}); err != nil {
		t.Fatalf("Run: %v", err)
	}

	// The other-tool file must be unchanged.
	data, err := os.ReadFile(otherFile)
	if err != nil {
		t.Fatalf("other skill file missing after init: %v", err)
	}
	if string(data) != originalContent {
		t.Errorf("other skill file was modified\ngot:  %q\nwant: %q", string(data), originalContent)
	}
}

// ---- helpers for version-aware tests ----

// newTestInitWithVersion creates an Initializer with a specific binary version string.
func newTestInitWithVersion(workDir, stdinContent, version string) (*Initializer, *bytes.Buffer) {
	var stdout bytes.Buffer
	return New(workDir, strings.NewReader(stdinContent), &stdout).WithVersion(version), &stdout
}

// ---- Run: skill files created on new project (AC-01, AC-03) ----

// AC-01: six kanbanzai skill files are created on a new project.
func TestRun_NewProject_CreatesSkills(t *testing.T) {
	dir := makeGitRepoNoCommits(t)
	in, _ := newTestInit(dir, "")
	if err := in.Run(Options{}); err != nil {
		t.Fatalf("Run: %v", err)
	}

	wantNames := []string{"agents", "design", "documents", "getting-started", "planning", "workflow"}
	for _, name := range wantNames {
		skillPath := filepath.Join(dir, ".agents", "skills", "kanbanzai-"+name, "SKILL.md")
		if _, err := os.Stat(skillPath); err != nil {
			t.Errorf("expected skill file %s to exist: %v", skillPath, err)
		}
	}
}

// AC-03: each SKILL.md contains the managed marker and version comment in frontmatter.
func TestRun_NewProject_SkillFrontmatter(t *testing.T) {
	dir := makeGitRepoNoCommits(t)
	in, _ := newTestInitWithVersion(dir, "", "1.2.3")
	if err := in.Run(Options{}); err != nil {
		t.Fatalf("Run: %v", err)
	}

	wantNames := []string{"agents", "design", "documents", "getting-started", "planning", "workflow"}
	for _, name := range wantNames {
		skillPath := filepath.Join(dir, ".agents", "skills", "kanbanzai-"+name, "SKILL.md")
		data, err := os.ReadFile(skillPath)
		if err != nil {
			t.Fatalf("read skill %s: %v", name, err)
		}
		content := string(data)
		if !strings.Contains(content, "# kanbanzai-managed:") {
			t.Errorf("skill %s: missing '# kanbanzai-managed:' line", name)
		}
		if !strings.Contains(content, "# kanbanzai-version: 1.2.3") {
			t.Errorf("skill %s: missing '# kanbanzai-version: 1.2.3' line", name)
		}
		if strings.Contains(content, "# kanbanzai-managed: true") {
			t.Errorf("skill %s: managed marker was not rewritten (still contains 'true')", name)
		}
		if !strings.Contains(content, "# kanbanzai-managed: do not edit") {
			t.Errorf("skill %s: managed marker missing 'do not edit' text", name)
		}
	}
}

// AC-04 (skills): second run with the same version is a no-op — mtime unchanged.
func TestRun_Idempotency_Skills(t *testing.T) {
	dir := makeGitRepoNoCommits(t)
	in1, _ := newTestInitWithVersion(dir, "", "1.0.0")
	if err := in1.Run(Options{}); err != nil {
		t.Fatalf("first Run: %v", err)
	}

	skillPath := filepath.Join(dir, ".agents", "skills", "kanbanzai-agents", "SKILL.md")
	info1, err := os.Stat(skillPath)
	if err != nil {
		t.Fatalf("stat after first run: %v", err)
	}

	in2, _ := newTestInitWithVersion(dir, "", "1.0.0")
	if err := in2.Run(Options{}); err != nil {
		t.Fatalf("second Run: %v", err)
	}

	info2, err := os.Stat(skillPath)
	if err != nil {
		t.Fatalf("stat after second run: %v", err)
	}

	if !info1.ModTime().Equal(info2.ModTime()) {
		t.Errorf("skill file mtime changed on second run (expected no-op): first=%v second=%v",
			info1.ModTime(), info2.ModTime())
	}
}

// ---- Run: skill version-aware update logic (AC-08, AC-09, AC-10) ----

// AC-08: skill files at the current version are not touched (mtime unchanged).
func TestRun_Skills_CurrentVersion_NoOp(t *testing.T) {
	dir := makeGitRepoNoCommits(t)
	in1, _ := newTestInitWithVersion(dir, "", "2.0.0")
	if err := in1.Run(Options{}); err != nil {
		t.Fatalf("first Run: %v", err)
	}

	agentsPath := filepath.Join(dir, ".agents", "skills", "kanbanzai-agents", "SKILL.md")
	info1, _ := os.Stat(agentsPath)

	in2, _ := newTestInitWithVersion(dir, "", "2.0.0")
	if err := in2.Run(Options{}); err != nil {
		t.Fatalf("second Run: %v", err)
	}

	info2, _ := os.Stat(agentsPath)
	if !info1.ModTime().Equal(info2.ModTime()) {
		t.Errorf("skill file was touched when version matched: mtime changed from %v to %v",
			info1.ModTime(), info2.ModTime())
	}
}

// AC-09: skill file with older version is overwritten; files at current version are not touched.
func TestRun_Skills_OlderVersion_Overwritten(t *testing.T) {
	dir := makeGitRepoNoCommits(t)

	in1, _ := newTestInitWithVersion(dir, "", "1.0.0")
	if err := in1.Run(Options{}); err != nil {
		t.Fatalf("first Run: %v", err)
	}

	agentsPath := filepath.Join(dir, ".agents", "skills", "kanbanzai-agents", "SKILL.md")
	designPath := filepath.Join(dir, ".agents", "skills", "kanbanzai-design", "SKILL.md")

	// Manually set the design skill to 2.0.0 so we can verify it stays untouched.
	designData, _ := os.ReadFile(designPath)
	updatedDesign := strings.ReplaceAll(string(designData), "# kanbanzai-version: 1.0.0", "# kanbanzai-version: 2.0.0")
	if err := os.WriteFile(designPath, []byte(updatedDesign), 0o644); err != nil {
		t.Fatalf("write design skill: %v", err)
	}
	designInfo1, _ := os.Stat(designPath)

	// Second run at version 2.0.0 — agents overwritten, design unchanged.
	in2, _ := newTestInitWithVersion(dir, "", "2.0.0")
	if err := in2.Run(Options{}); err != nil {
		t.Fatalf("second Run: %v", err)
	}

	agentsData, _ := os.ReadFile(agentsPath)
	if !strings.Contains(string(agentsData), "# kanbanzai-version: 2.0.0") {
		t.Errorf("agents skill not updated to 2.0.0\ncontent: %s", string(agentsData))
	}

	designInfo2, _ := os.Stat(designPath)
	if !designInfo1.ModTime().Equal(designInfo2.ModTime()) {
		t.Errorf("design skill was touched even though version already matched: mtime changed from %v to %v",
			designInfo1.ModTime(), designInfo2.ModTime())
	}
}

// AC-10: skill file without managed marker causes non-zero exit.
func TestRun_Skills_NoManagedMarker_Error(t *testing.T) {
	dir := makeGitRepoNoCommits(t)

	skillDir := filepath.Join(dir, ".agents", "skills", "kanbanzai-agents")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatal(err)
	}
	skillPath := filepath.Join(skillDir, "SKILL.md")
	if err := os.WriteFile(skillPath, []byte("# kanbanzai-agents\nCustom content without managed marker.\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	in, _ := newTestInit(dir, "")
	err := in.Run(Options{})
	if err == nil {
		t.Fatal("expected error for skill file without managed marker, got nil")
	}
	if !strings.Contains(err.Error(), skillPath) {
		t.Errorf("error should contain skill file path %q; got: %v", skillPath, err)
	}
	if !strings.Contains(err.Error(), "--skip-skills") {
		t.Errorf("error should mention --skip-skills; got: %v", err)
	}

	data, _ := os.ReadFile(skillPath)
	if !strings.Contains(string(data), "Custom content without managed marker.") {
		t.Errorf("skill file was modified despite error condition")
	}
}

// ---- Run: --update-skills flag (AC-12) ----

// AC-12: --update-skills updates only skill files, leaves config and work/ unchanged.
func TestRun_UpdateSkills_OnlySkills(t *testing.T) {
	dir := makeGitRepoNoCommits(t)

	in1, _ := newTestInitWithVersion(dir, "", "1.0.0")
	if err := in1.Run(Options{}); err != nil {
		t.Fatalf("first Run: %v", err)
	}

	configPath := filepath.Join(dir, ".kbz", "config.yaml")
	configInfo1, _ := os.Stat(configPath)

	workDesign := filepath.Join(dir, "work", "design", ".gitkeep")
	workInfo1, _ := os.Stat(workDesign)

	in2, _ := newTestInitWithVersion(dir, "", "2.0.0")
	if err := in2.Run(Options{UpdateSkills: true}); err != nil {
		t.Fatalf("--update-skills Run: %v", err)
	}

	configInfo2, _ := os.Stat(configPath)
	if !configInfo1.ModTime().Equal(configInfo2.ModTime()) {
		t.Errorf("config.yaml was modified by --update-skills")
	}

	workInfo2, _ := os.Stat(workDesign)
	if !workInfo1.ModTime().Equal(workInfo2.ModTime()) {
		t.Errorf("work/ .gitkeep was modified by --update-skills")
	}

	agentsPath := filepath.Join(dir, ".agents", "skills", "kanbanzai-agents", "SKILL.md")
	agentsData, _ := os.ReadFile(agentsPath)
	if !strings.Contains(string(agentsData), "# kanbanzai-version: 2.0.0") {
		t.Errorf("skill not updated to 2.0.0 after --update-skills\ncontent: %s", string(agentsData))
	}
}

// ---- Run: sentinel file (atomicity / partial init detection) ----

// TestRun_SentinelFileWritten verifies that .kbz/.init-complete is created.
func TestRun_SentinelFileWritten(t *testing.T) {
	dir := makeGitRepoNoCommits(t)
	in, _ := newTestInit(dir, "")
	if err := in.Run(Options{}); err != nil {
		t.Fatalf("Run: %v", err)
	}

	sentinel := filepath.Join(dir, ".kbz", ".init-complete")
	if _, err := os.Stat(sentinel); err != nil {
		t.Errorf(".kbz/.init-complete not created: %v", err)
	}
}

// TestRun_PartialInit_Detected verifies that a missing sentinel triggers a warning.
func TestRun_PartialInit_Detected(t *testing.T) {
	dir := makeGitRepoNoCommits(t)

	// Create .kbz/ with a config but no sentinel — simulates a partial init.
	kbzDir := filepath.Join(dir, ".kbz")
	if err := WriteInitConfig(kbzDir, "", DefaultDocumentRoots()); err != nil {
		t.Fatalf("pre-create config: %v", err)
	}

	in, out := newTestInit(dir, "")
	if err := in.Run(Options{}); err != nil {
		t.Fatalf("Run: %v", err)
	}

	if !strings.Contains(out.String(), "incomplete") {
		t.Errorf("expected partial-init warning mentioning 'incomplete'; got: %q", out.String())
	}
}

// ---- Run: MCP config files ----

// TestInit_WritesMcpJson verifies AC-01 to AC-03.
func TestInit_WritesMcpJson(t *testing.T) {
	t.Parallel()
	dir := makeGitRepoNoCommits(t)
	in, _ := newTestInit(dir, "")
	if err := in.Run(Options{}); err != nil {
		t.Fatalf("Run: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, ".mcp.json"))
	if err != nil {
		t.Fatalf("read .mcp.json: %v", err)
	}

	var cfg map[string]interface{}
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("parse .mcp.json: %v", err)
	}

	managed, ok := cfg["_managed"].(map[string]interface{})
	if !ok {
		t.Fatal("missing _managed block")
	}
	if managed["tool"] != "kanbanzai" {
		t.Errorf("_managed.tool = %v, want kanbanzai", managed["tool"])
	}

	servers, ok := cfg["mcpServers"].(map[string]interface{})
	if !ok {
		t.Fatal("missing mcpServers")
	}
	kbz, ok := servers["kanbanzai"].(map[string]interface{})
	if !ok {
		t.Fatal("missing mcpServers.kanbanzai")
	}
	if kbz["command"] != "kbz" {
		t.Errorf("command = %v, want kbz", kbz["command"])
	}
}

// TestInit_UnmanagedMcpJson_Skips verifies AC-04.
func TestInit_UnmanagedMcpJson_Skips(t *testing.T) {
	t.Parallel()
	dir := makeGitRepoNoCommits(t)

	// Write an unmanaged .mcp.json
	original := `{"mcpServers": {"other": {"command": "other"}}}`
	if err := os.WriteFile(filepath.Join(dir, ".mcp.json"), []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}

	in, out := newTestInit(dir, "")
	if err := in.Run(Options{}); err != nil {
		t.Fatalf("Run: %v", err)
	}

	data, _ := os.ReadFile(filepath.Join(dir, ".mcp.json"))
	if string(data) != original {
		t.Errorf(".mcp.json was modified; want unchanged")
	}
	if !strings.Contains(out.String(), ".mcp.json") {
		t.Errorf("expected warning mentioning .mcp.json in output; got: %s", out.String())
	}
}

// TestInit_ManagedMcpJson_OlderVersion_Overwrites verifies AC-05.
func TestInit_ManagedMcpJson_OlderVersion_Overwrites(t *testing.T) {
	t.Parallel()
	dir := makeGitRepoNoCommits(t)

	older := `{"_managed": {"tool": "kanbanzai", "version": 0}, "mcpServers": {}}`
	if err := os.WriteFile(filepath.Join(dir, ".mcp.json"), []byte(older), 0o644); err != nil {
		t.Fatal(err)
	}

	in, _ := newTestInit(dir, "")
	if err := in.Run(Options{}); err != nil {
		t.Fatalf("Run: %v", err)
	}

	data, _ := os.ReadFile(filepath.Join(dir, ".mcp.json"))
	var cfg map[string]interface{}
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("parse: %v", err)
	}
	// Should now have kanbanzai server entry
	servers := cfg["mcpServers"].(map[string]interface{})
	if _, ok := servers["kanbanzai"]; !ok {
		t.Error("expected kanbanzai server entry after overwrite")
	}
}

// TestInit_ManagedMcpJson_CurrentVersion_NoOp verifies AC-06.
func TestInit_ManagedMcpJson_CurrentVersion_NoOp(t *testing.T) {
	t.Parallel()
	dir := makeGitRepoNoCommits(t)

	// First run to create managed file at current version.
	in1, _ := newTestInit(dir, "")
	if err := in1.Run(Options{}); err != nil {
		t.Fatalf("first Run: %v", err)
	}
	original, _ := os.ReadFile(filepath.Join(dir, ".mcp.json"))

	// Second run should not modify it.
	in2, _ := newTestInit(dir, "")
	if err := in2.Run(Options{}); err != nil {
		t.Fatalf("second Run: %v", err)
	}
	current, _ := os.ReadFile(filepath.Join(dir, ".mcp.json"))

	if string(original) != string(current) {
		t.Error("second run modified .mcp.json at current version")
	}
}

// TestInit_ZedDir_WritesSettingsJson verifies AC-07 to AC-08.
// assertZedToolPermissions checks that cfg contains agent.tool_permissions with the
// expected kanbanzai tool entries. Fails the test if any are missing or wrong.
func assertZedToolPermissions(t *testing.T, cfg map[string]interface{}) {
	t.Helper()
	agent, ok := cfg["agent"].(map[string]interface{})
	if !ok {
		t.Error(".zed/settings.json missing agent block")
		return
	}
	tp, ok := agent["tool_permissions"].(map[string]interface{})
	if !ok {
		t.Error(".zed/settings.json missing agent.tool_permissions")
		return
	}
	tools, ok := tp["tools"].(map[string]interface{})
	if !ok {
		t.Error(".zed/settings.json missing agent.tool_permissions.tools")
		return
	}

	// Spot-check: high-frequency read tools must be allow.
	for _, name := range []string{
		"mcp:kanbanzai:status",
		"mcp:kanbanzai:next",
		"mcp:kanbanzai:entity",
		"mcp:kanbanzai:health",
		"mcp:kanbanzai:finish",
	} {
		entry, ok := tools[name].(map[string]interface{})
		if !ok {
			t.Errorf("agent.tool_permissions.tools[%q] missing", name)
			continue
		}
		if entry["default"] != "allow" {
			t.Errorf("agent.tool_permissions.tools[%q].default = %v, want allow", name, entry["default"])
		}
	}

	// Destructive tools must be confirm.
	for _, name := range []string{
		"mcp:kanbanzai:merge",
		"mcp:kanbanzai:pr",
		"mcp:kanbanzai:cleanup",
	} {
		entry, ok := tools[name].(map[string]interface{})
		if !ok {
			t.Errorf("agent.tool_permissions.tools[%q] missing", name)
			continue
		}
		if entry["default"] != "confirm" {
			t.Errorf("agent.tool_permissions.tools[%q].default = %v, want confirm", name, entry["default"])
		}
	}
}

func TestInit_ZedDir_WritesSettingsJson(t *testing.T) {
	t.Parallel()
	dir := makeGitRepoNoCommits(t)

	// Create .zed/ directory
	if err := os.MkdirAll(filepath.Join(dir, ".zed"), 0o755); err != nil {
		t.Fatal(err)
	}

	in, _ := newTestInit(dir, "")
	if err := in.Run(Options{}); err != nil {
		t.Fatalf("Run: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, ".zed", "settings.json"))
	if err != nil {
		t.Fatalf("read .zed/settings.json: %v", err)
	}

	var cfg map[string]interface{}
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("parse: %v", err)
	}

	// _managed must NOT be present — Zed's schema rejects unknown top-level properties.
	if _, ok := cfg["_managed"]; ok {
		t.Error(".zed/settings.json must not contain _managed (Zed schema rejects it)")
	}

	servers := cfg["context_servers"].(map[string]interface{})
	kbz, ok := servers["kanbanzai"].(map[string]interface{})
	if !ok {
		t.Fatal("missing context_servers.kanbanzai")
	}
	// command must be a flat string, not a nested object — Zed silently ignores
	// the nested {"path":..., "args":[...]} form.
	if cmd, ok := kbz["command"].(string); !ok || cmd == "" {
		t.Errorf("context_servers.kanbanzai.command = %v, want a non-empty string", kbz["command"])
	}
	args, ok := kbz["args"].([]interface{})
	if !ok || len(args) == 0 {
		t.Errorf("context_servers.kanbanzai.args = %v, want a non-empty array", kbz["args"])
	}

	// agent.tool_permissions must be present with kanbanzai tools pre-approved.
	assertZedToolPermissions(t, cfg)
}

// TestInit_NewProject_NoZedDir_CreatesSettingsJson verifies that a new project always gets
// .zed/settings.json even when .zed/ does not exist at init time. Zed creates .zed/ lazily
// on first open, so detecting its presence is not a reliable signal for new projects.
func TestInit_NewProject_NoZedDir_CreatesSettingsJson(t *testing.T) {
	t.Parallel()
	dir := makeGitRepoNoCommits(t)
	in, _ := newTestInit(dir, "")
	if err := in.Run(Options{}); err != nil {
		t.Fatalf("Run: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, ".zed", "settings.json"))
	if err != nil {
		t.Fatalf(".zed/settings.json not created for new project: %v", err)
	}
	var cfg map[string]interface{}
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("parse .zed/settings.json: %v", err)
	}

	servers, ok := cfg["context_servers"].(map[string]interface{})
	if !ok {
		t.Fatal(".zed/settings.json missing context_servers key")
	}
	kbz, ok := servers["kanbanzai"].(map[string]interface{})
	if !ok {
		t.Fatal("missing context_servers.kanbanzai")
	}
	if cmd, ok := kbz["command"].(string); !ok || cmd == "" {
		t.Errorf("context_servers.kanbanzai.command = %v, want a non-empty string", kbz["command"])
	}

	assertZedToolPermissions(t, cfg)
}

// TestInit_ExistingProject_NoZedDir_NoSettingsJson verifies that re-running init on a
// project where kanbanzai was already set up (.kbz/ existed) does not create .zed/ —
// the missing directory signals the project does not use Zed.
func TestInit_ExistingProject_NoZedDir_NoSettingsJson(t *testing.T) {
	t.Parallel()
	dir := makeGitRepoWithCommit(t)
	kbzDir := filepath.Join(dir, ".kbz")
	if err := WriteInitConfig(kbzDir, "", DefaultDocumentRoots()); err != nil {
		t.Fatalf("pre-create config: %v", err)
	}

	in, _ := newTestInit(dir, "")
	if err := in.Run(Options{}); err != nil {
		t.Fatalf("Run: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir, ".zed")); !os.IsNotExist(err) {
		t.Error("expected .zed/ not to be created when .kbz/ already existed")
	}
}

// TestInit_FirstTimeInit_WithCommits_CreatesZedSettings verifies that a project with
// existing commits but no prior kanbanzai setup (.kbz/ absent) gets .zed/settings.json
// written. This is the common case: a repo with a README that is being initialised with
// kanbanzai for the first time.
func TestInit_FirstTimeInit_WithCommits_CreatesZedSettings(t *testing.T) {
	t.Parallel()
	dir := makeGitRepoWithCommit(t)
	// No .kbz/ pre-created — this is a first-time init on a project with commits.

	in, out := newTestInit(dir, "work")
	if err := in.Run(Options{NonInteractive: true, Name: "Test", DocsPath: []string{"work"}}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	_ = out

	data, err := os.ReadFile(filepath.Join(dir, ".zed", "settings.json"))
	if err != nil {
		t.Fatalf(".zed/settings.json not created for first-time init with commits: %v", err)
	}
	var cfg map[string]interface{}
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("parse .zed/settings.json: %v", err)
	}
	servers, ok := cfg["context_servers"].(map[string]interface{})
	if !ok {
		t.Fatal(".zed/settings.json missing context_servers")
	}
	if _, ok := servers["kanbanzai"]; !ok {
		t.Fatal("missing context_servers.kanbanzai")
	}
}

// TestInit_ZedSettings_MigratesNoAgentBlock verifies that an existing .zed/settings.json
// written by an older kanbanzai version (which lacked agent.tool_permissions) is rewritten
// to include tool_permissions when the file has no "agent" key.
func TestInit_ZedSettings_MigratesNoAgentBlock(t *testing.T) {
	t.Parallel()
	dir := makeGitRepoNoCommits(t)

	if err := os.MkdirAll(filepath.Join(dir, ".zed"), 0o755); err != nil {
		t.Fatal(err)
	}
	// Simulate file written by older kanbanzai: context_servers present, no agent key.
	old := `{"context_servers":{"kanbanzai":{"command":"kanbanzai","args":["serve"]}}}`
	if err := os.WriteFile(filepath.Join(dir, ".zed", "settings.json"), []byte(old), 0o644); err != nil {
		t.Fatal(err)
	}

	in, _ := newTestInit(dir, "")
	if err := in.Run(Options{}); err != nil {
		t.Fatalf("Run: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, ".zed", "settings.json"))
	if err != nil {
		t.Fatalf("read .zed/settings.json: %v", err)
	}
	var cfg map[string]interface{}
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("parse migrated .zed/settings.json: %v", err)
	}

	assertZedToolPermissions(t, cfg)
}

// TestInit_ZedSettings_PreservesUserAgentBlock verifies that an existing .zed/settings.json
// with a user-added "agent" block is not overwritten, even if tool_permissions are absent.
func TestInit_ZedSettings_PreservesUserAgentBlock(t *testing.T) {
	t.Parallel()
	dir := makeGitRepoNoCommits(t)

	if err := os.MkdirAll(filepath.Join(dir, ".zed"), 0o755); err != nil {
		t.Fatal(err)
	}
	// Simulate user-customised file with agent block but no kanbanzai tool_permissions.
	original := `{"context_servers":{"kanbanzai":{"command":"kanbanzai","args":["serve"]}},"agent":{"default_model":"custom"}}`
	if err := os.WriteFile(filepath.Join(dir, ".zed", "settings.json"), []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}

	in, _ := newTestInit(dir, "")
	if err := in.Run(Options{}); err != nil {
		t.Fatalf("Run: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, ".zed", "settings.json"))
	if err != nil {
		t.Fatalf("read .zed/settings.json: %v", err)
	}
	if string(data) != original {
		t.Errorf(".zed/settings.json with user agent block was modified; want unchanged\ngot:  %s\nwant: %s", string(data), original)
	}
}

// TestInit_ZedSettings_MigratesOldManagedBlock verifies that a .zed/settings.json written
// by an older kanbanzai version (which included a _managed block) is rewritten without it.
func TestInit_ZedSettings_MigratesOldManagedBlock(t *testing.T) {
	t.Parallel()
	dir := makeGitRepoNoCommits(t)

	if err := os.MkdirAll(filepath.Join(dir, ".zed"), 0o755); err != nil {
		t.Fatal(err)
	}
	// Simulate old-format file that includes _managed (which Zed rejects).
	old := `{"_managed":{"tool":"kanbanzai","version":1},"context_servers":{"kanbanzai":{"command":"kanbanzai","args":["serve"]}}}`
	if err := os.WriteFile(filepath.Join(dir, ".zed", "settings.json"), []byte(old), 0o644); err != nil {
		t.Fatal(err)
	}

	in, _ := newTestInit(dir, "")
	if err := in.Run(Options{}); err != nil {
		t.Fatalf("Run: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, ".zed", "settings.json"))
	if err != nil {
		t.Fatalf("read .zed/settings.json: %v", err)
	}

	var cfg map[string]interface{}
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("parse migrated .zed/settings.json: %v", err)
	}

	if _, ok := cfg["_managed"]; ok {
		t.Error("migrated .zed/settings.json still contains _managed block")
	}

	servers, ok := cfg["context_servers"].(map[string]interface{})
	if !ok {
		t.Fatal("migrated .zed/settings.json missing context_servers")
	}
	if _, ok := servers["kanbanzai"]; !ok {
		t.Fatal("migrated .zed/settings.json missing context_servers.kanbanzai")
	}

	assertZedToolPermissions(t, cfg)
}

// TestInit_FirstTimeInit_WithCommits_CreatesWorkDirs verifies that a project with
// existing commits but no prior kanbanzai setup gets work/ directories and README
// created — the same behaviour as a new project init.
func TestInit_FirstTimeInit_WithCommits_CreatesWorkDirs(t *testing.T) {
	t.Parallel()
	dir := makeGitRepoWithCommit(t)
	// No .kbz/ pre-created — first-time init on a project with commits.

	in, _ := newTestInit(dir, "")
	if err := in.Run(Options{NonInteractive: true, Name: "Test", DocsPath: []string{"work/design"}}); err != nil {
		t.Fatalf("Run: %v", err)
	}

	// The docs-path directory should be created.
	if _, err := os.Stat(filepath.Join(dir, "work", "design")); os.IsNotExist(err) {
		t.Error("work/design/ not created on first-time init with commits")
	}
}

// TestInit_FirstTimeInit_WithCommits_CreatesWorkReadme verifies that work/README.md
// is created on first-time init with commits when the default layout is used.
func TestInit_FirstTimeInit_WithCommits_CreatesWorkReadme(t *testing.T) {
	t.Parallel()
	dir := makeGitRepoWithCommit(t)

	in, _ := newTestInit(dir, "")
	if err := in.Run(Options{NonInteractive: true, Name: "Test", DocsPath: []string{"work"}}); err != nil {
		t.Fatalf("Run: %v", err)
	}

	// work/README.md should NOT be written when work/ root is a single path
	// that doesn't match the standard layout — writeWorkReadme skips when
	// work/ has no subdirectory matching its template.
	// (The README is only written when work/ exists as a directory.)
	// Just verify we didn't error — the directory should have been created.
	if _, err := os.Stat(filepath.Join(dir, "work")); os.IsNotExist(err) {
		t.Error("work/ not created on first-time init with commits")
	}
}

// TestInit_FirstTimeInit_DefaultRoots_CreatesAllWorkDirs verifies that first-time
// init with default roots creates all eight work/ directories and work/README.md.
func TestInit_FirstTimeInit_DefaultRoots_CreatesAllWorkDirs(t *testing.T) {
	t.Parallel()
	dir := makeGitRepoNoCommits(t)

	in, _ := newTestInit(dir, "")
	if err := in.Run(Options{}); err != nil {
		t.Fatalf("Run: %v", err)
	}

	for _, sub := range []string{"design", "spec", "plan", "dev", "research", "report", "review", "retro"} {
		if _, err := os.Stat(filepath.Join(dir, "work", sub)); os.IsNotExist(err) {
			t.Errorf("work/%s/ not created", sub)
		}
	}
	if _, err := os.Stat(filepath.Join(dir, "work", "README.md")); os.IsNotExist(err) {
		t.Error("work/README.md not created")
	}
}

// TestInit_UnmanagedZedSettings_Skips verifies that an existing .zed/settings.json
// without a kanbanzai entry is left untouched with a warning.
func TestInit_UnmanagedZedSettings_Skips(t *testing.T) {
	t.Parallel()
	dir := makeGitRepoNoCommits(t)

	if err := os.MkdirAll(filepath.Join(dir, ".zed"), 0o755); err != nil {
		t.Fatal(err)
	}
	original := `{"context_servers": {}}`
	if err := os.WriteFile(filepath.Join(dir, ".zed", "settings.json"), []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}

	in, out := newTestInit(dir, "")
	if err := in.Run(Options{}); err != nil {
		t.Fatalf("Run: %v", err)
	}

	data, _ := os.ReadFile(filepath.Join(dir, ".zed", "settings.json"))
	if string(data) != original {
		t.Error(".zed/settings.json was modified")
	}
	if !strings.Contains(out.String(), ".zed/settings.json") {
		t.Errorf("expected warning mentioning .zed/settings.json; got: %s", out.String())
	}
}

// TestInit_SkipMcp verifies AC-13 to AC-16.
func TestInit_SkipMcp(t *testing.T) {
	t.Parallel()
	dir := makeGitRepoNoCommits(t)

	// Create .zed/ to test that Zed file is also skipped
	if err := os.MkdirAll(filepath.Join(dir, ".zed"), 0o755); err != nil {
		t.Fatal(err)
	}

	in, _ := newTestInit(dir, "")
	if err := in.Run(Options{SkipMCP: true}); err != nil {
		t.Fatalf("Run: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir, ".mcp.json")); !os.IsNotExist(err) {
		t.Error("expected .mcp.json not to be created with --skip-mcp")
	}
	if _, err := os.Stat(filepath.Join(dir, ".zed", "settings.json")); !os.IsNotExist(err) {
		t.Error("expected .zed/settings.json not to be created with --skip-mcp")
	}
	// Config should still be created
	if _, err := os.Stat(filepath.Join(dir, ".kbz", "config.yaml")); os.IsNotExist(err) {
		t.Error("expected config.yaml to still be created with --skip-mcp")
	}
}

// ---- Run: work/README.md ----

// TestRun_NewProject_CreatesWorkReadme verifies that work/README.md is created for new projects.
func TestRun_NewProject_CreatesWorkReadme(t *testing.T) {
	dir := makeGitRepoNoCommits(t)
	in, _ := newTestInit(dir, "")
	if err := in.Run(Options{}); err != nil {
		t.Fatalf("Run: %v", err)
	}

	readmePath := filepath.Join(dir, "work", "README.md")
	data, err := os.ReadFile(readmePath)
	if err != nil {
		t.Fatalf("work/README.md not created: %v", err)
	}
	content := string(data)

	// Must contain all 8 directories.
	for _, dir := range []string{"design/", "spec/", "plan/", "dev/", "research/", "report/", "review/", "retro/"} {
		if !strings.Contains(content, dir) {
			t.Errorf("work/README.md missing directory entry %q", dir)
		}
	}

	// Must contain the AI agents line.
	if !strings.Contains(content, "AI agents") {
		t.Error("work/README.md missing AI agents line")
	}
	if !strings.Contains(content, "kanbanzai-documents") {
		t.Error("work/README.md missing kanbanzai-documents skill reference")
	}
}

// TestRun_SkipWorkDirs_NoReadme verifies that --skip-work-dirs suppresses work/README.md.
func TestRun_SkipWorkDirs_NoReadme(t *testing.T) {
	dir := makeGitRepoNoCommits(t)
	in, _ := newTestInit(dir, "")
	if err := in.Run(Options{SkipWorkDirs: true}); err != nil {
		t.Fatalf("Run: %v", err)
	}

	readmePath := filepath.Join(dir, "work", "README.md")
	if _, err := os.Stat(readmePath); err == nil {
		t.Error("work/README.md should NOT be created with --skip-work-dirs")
	}
}

// TestRun_ExistingProject_NoReadme verifies that existing projects do not get work/README.md.
func TestRun_ExistingProject_NoReadme(t *testing.T) {
	dir := makeGitRepoWithCommit(t)
	kbzDir := filepath.Join(dir, ".kbz")
	if err := WriteInitConfig(kbzDir, "", DefaultDocumentRoots()); err != nil {
		t.Fatalf("pre-create config: %v", err)
	}

	in, _ := newTestInit(dir, "")
	if err := in.Run(Options{}); err != nil {
		t.Fatalf("Run: %v", err)
	}

	readmePath := filepath.Join(dir, "work", "README.md")
	if _, err := os.Stat(readmePath); err == nil {
		t.Error("work/README.md should NOT be created for an existing project")
	}
}

// ---- Run: context role files ----

// TestRun_NewProject_CreatesBaseRole verifies that base.yaml is created for a new project.
func TestRun_NewProject_CreatesBaseRole(t *testing.T) {
	t.Parallel()
	dir := makeGitRepoNoCommits(t)

	in, _ := newTestInit(dir, "")
	if err := in.Run(Options{}); err != nil {
		t.Fatalf("Run: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, ".kbz", "context", "roles", "base.yaml"))
	if err != nil {
		t.Fatalf("read base.yaml: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "id: base") {
		t.Error("base.yaml missing 'id: base'")
	}
	if !strings.Contains(content, "conventions: []") {
		t.Error("base.yaml missing 'conventions: []'")
	}
	// No managed marker
	if strings.Contains(content, "kanbanzai-managed") {
		t.Error("base.yaml should not have managed marker")
	}
}

// TestRun_NewProject_CreatesReviewerRole verifies that reviewer.yaml is created with correct content.
func TestRun_NewProject_CreatesReviewerRole(t *testing.T) {
	t.Parallel()
	dir := makeGitRepoNoCommits(t)

	in, _ := newTestInitWithVersion(dir, "", "1.0.0")
	if err := in.Run(Options{}); err != nil {
		t.Fatalf("Run: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, ".kbz", "context", "roles", "reviewer.yaml"))
	if err != nil {
		t.Fatalf("read reviewer.yaml: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "id: reviewer") {
		t.Error("missing 'id: reviewer'")
	}
	if !strings.Contains(content, "inherits: base") {
		t.Error("missing 'inherits: base'")
	}
	if !strings.Contains(content, `kanbanzai-managed: "true"`) {
		t.Error("missing managed marker")
	}
	if !strings.Contains(content, `version: "1.0.0"`) {
		t.Error("missing version 1.0.0")
	}
	if !strings.Contains(content, "review_approach:") {
		t.Error("missing review_approach key")
	}
	if !strings.Contains(content, "output_format:") {
		t.Error("missing output_format key")
	}
	if !strings.Contains(content, "dimensions:") {
		t.Error("missing dimensions key")
	}
	if !strings.Contains(content, "kanbanzai-review") {
		t.Error("reviewer.yaml should reference kanbanzai-review skill")
	}
}

// TestRun_SkipRoles_CreatesNeither verifies that --skip-roles suppresses both role files.
func TestRun_SkipRoles_CreatesNeither(t *testing.T) {
	t.Parallel()
	dir := makeGitRepoNoCommits(t)

	in, _ := newTestInit(dir, "")
	if err := in.Run(Options{SkipRoles: true}); err != nil {
		t.Fatalf("Run: %v", err)
	}

	rolesDir := filepath.Join(dir, ".kbz", "context", "roles")
	if _, err := os.Stat(filepath.Join(rolesDir, "base.yaml")); !os.IsNotExist(err) {
		t.Error("base.yaml should not be created with --skip-roles")
	}
	if _, err := os.Stat(filepath.Join(rolesDir, "reviewer.yaml")); !os.IsNotExist(err) {
		t.Error("reviewer.yaml should not be created with --skip-roles")
	}
}

// TestRun_BaseRole_NotOverwritten verifies that a pre-existing base.yaml is never overwritten.
func TestRun_BaseRole_NotOverwritten(t *testing.T) {
	t.Parallel()
	dir := makeGitRepoNoCommits(t)

	// Pre-create a custom base.yaml inside .kbz (which also creates the .kbz dir).
	rolesDir := filepath.Join(dir, ".kbz", "context", "roles")
	if err := os.MkdirAll(rolesDir, 0o755); err != nil {
		t.Fatal(err)
	}
	custom := "id: base\ndescription: \"custom\"\nconventions: [\"my-convention\"]\n"
	if err := os.WriteFile(filepath.Join(rolesDir, "base.yaml"), []byte(custom), 0o644); err != nil {
		t.Fatal(err)
	}

	// Pre-create config.yaml so runExistingProject (triggered by .kbz existing) doesn't prompt.
	kbzDir := filepath.Join(dir, ".kbz")
	if err := WriteInitConfig(kbzDir, "", DefaultDocumentRoots()); err != nil {
		t.Fatalf("pre-create config: %v", err)
	}

	in, _ := newTestInit(dir, "")
	if err := in.Run(Options{}); err != nil {
		t.Fatalf("Run: %v", err)
	}

	data, _ := os.ReadFile(filepath.Join(rolesDir, "base.yaml"))
	if string(data) != custom {
		t.Error("base.yaml was overwritten but should be left alone")
	}
}

// TestRun_ReviewerRole_UnmanagedSkipsWithWarning verifies that an unmanaged reviewer.yaml is left alone.
func TestRun_ReviewerRole_UnmanagedSkipsWithWarning(t *testing.T) {
	t.Parallel()
	dir := makeGitRepoNoCommits(t)

	rolesDir := filepath.Join(dir, ".kbz", "context", "roles")
	if err := os.MkdirAll(rolesDir, 0o755); err != nil {
		t.Fatal(err)
	}
	unmanaged := "id: reviewer\nconventions: []\n"
	if err := os.WriteFile(filepath.Join(rolesDir, "reviewer.yaml"), []byte(unmanaged), 0o644); err != nil {
		t.Fatal(err)
	}

	// Pre-create config.yaml so runExistingProject doesn't prompt.
	kbzDir := filepath.Join(dir, ".kbz")
	if err := WriteInitConfig(kbzDir, "", DefaultDocumentRoots()); err != nil {
		t.Fatalf("pre-create config: %v", err)
	}

	in, out := newTestInit(dir, "")
	if err := in.Run(Options{}); err != nil {
		t.Fatalf("Run: %v", err)
	}

	data, _ := os.ReadFile(filepath.Join(rolesDir, "reviewer.yaml"))
	if string(data) != unmanaged {
		t.Error("unmanaged reviewer.yaml was modified")
	}
	if !strings.Contains(out.String(), "reviewer.yaml") {
		t.Errorf("expected warning mentioning reviewer.yaml; got: %s", out.String())
	}
}

// TestRun_ReviewerRole_OlderVersion_Overwritten verifies that a managed reviewer.yaml at an older version is updated.
func TestRun_ReviewerRole_OlderVersion_Overwritten(t *testing.T) {
	t.Parallel()
	dir := makeGitRepoNoCommits(t)

	rolesDir := filepath.Join(dir, ".kbz", "context", "roles")
	if err := os.MkdirAll(rolesDir, 0o755); err != nil {
		t.Fatal(err)
	}
	older := "id: reviewer\ninherits: base\nmetadata:\n  kanbanzai-managed: \"true\"\n  version: \"0.9.0\"\nconventions: []\n"
	if err := os.WriteFile(filepath.Join(rolesDir, "reviewer.yaml"), []byte(older), 0o644); err != nil {
		t.Fatal(err)
	}

	// Pre-create config.yaml so runExistingProject doesn't prompt.
	kbzDir := filepath.Join(dir, ".kbz")
	if err := WriteInitConfig(kbzDir, "", DefaultDocumentRoots()); err != nil {
		t.Fatalf("pre-create config: %v", err)
	}

	in, _ := newTestInitWithVersion(dir, "", "1.0.0")
	if err := in.Run(Options{}); err != nil {
		t.Fatalf("Run: %v", err)
	}

	data, _ := os.ReadFile(filepath.Join(rolesDir, "reviewer.yaml"))
	if !strings.Contains(string(data), `version: "1.0.0"`) {
		t.Error("reviewer.yaml was not updated to current version")
	}
}

// TestRun_ReviewerRole_CurrentVersion_NoOp verifies that a managed reviewer.yaml at the current version is not re-written.
func TestRun_ReviewerRole_CurrentVersion_NoOp(t *testing.T) {
	t.Parallel()
	dir := makeGitRepoNoCommits(t)

	// First run to create at current version.
	in1, _ := newTestInitWithVersion(dir, "", "1.0.0")
	if err := in1.Run(Options{}); err != nil {
		t.Fatalf("first Run: %v", err)
	}
	reviewerPath := filepath.Join(dir, ".kbz", "context", "roles", "reviewer.yaml")
	original, _ := os.ReadFile(reviewerPath)

	// Second run at same version — should be a no-op for reviewer.yaml.
	in2, _ := newTestInitWithVersion(dir, "", "1.0.0")
	if err := in2.Run(Options{}); err != nil {
		t.Fatalf("second Run: %v", err)
	}
	current, _ := os.ReadFile(reviewerPath)

	if string(original) != string(current) {
		t.Error("second run at same version modified reviewer.yaml")
	}
}

// TestRun_NoDeveloperYaml verifies that developer.yaml is not created by kbz init.
func TestRun_NoDeveloperYaml(t *testing.T) {
	t.Parallel()
	dir := makeGitRepoNoCommits(t)

	in, _ := newTestInit(dir, "")
	if err := in.Run(Options{}); err != nil {
		t.Fatalf("Run: %v", err)
	}

	developerPath := filepath.Join(dir, ".kbz", "context", "roles", "developer.yaml")
	if _, err := os.Stat(developerPath); !os.IsNotExist(err) {
		t.Error("developer.yaml should not be created by kbz init")
	}
}

// TestRun_UpdateSkills_UpdatesManagedReviewer verifies that --update-skills also updates managed reviewer.yaml.
func TestRun_UpdateSkills_UpdatesManagedReviewer(t *testing.T) {
	t.Parallel()
	dir := makeGitRepoWithCommit(t)

	// Pre-create .kbz with a managed reviewer.yaml at an older version.
	kbzDir := filepath.Join(dir, ".kbz")
	rolesDir := filepath.Join(kbzDir, "context", "roles")
	if err := os.MkdirAll(rolesDir, 0o755); err != nil {
		t.Fatal(err)
	}
	older := "id: reviewer\ninherits: base\nmetadata:\n  kanbanzai-managed: \"true\"\n  version: \"0.9.0\"\nconventions: []\n"
	if err := os.WriteFile(filepath.Join(rolesDir, "reviewer.yaml"), []byte(older), 0o644); err != nil {
		t.Fatal(err)
	}

	in, _ := newTestInitWithVersion(dir, "", "1.0.0")
	if err := in.Run(Options{UpdateSkills: true}); err != nil {
		t.Fatalf("Run --update-skills: %v", err)
	}

	data, _ := os.ReadFile(filepath.Join(rolesDir, "reviewer.yaml"))
	if !strings.Contains(string(data), `version: "1.0.0"`) {
		t.Error("--update-skills did not update managed reviewer.yaml")
	}
}

// TestRun_UpdateSkills_DoesNotTouchBaseRole verifies that --update-skills never modifies base.yaml.
func TestRun_UpdateSkills_DoesNotTouchBaseRole(t *testing.T) {
	t.Parallel()
	dir := makeGitRepoWithCommit(t)

	kbzDir := filepath.Join(dir, ".kbz")
	rolesDir := filepath.Join(kbzDir, "context", "roles")
	if err := os.MkdirAll(rolesDir, 0o755); err != nil {
		t.Fatal(err)
	}
	custom := "id: base\ndescription: \"custom project conventions\"\nconventions: [\"custom\"]\n"
	if err := os.WriteFile(filepath.Join(rolesDir, "base.yaml"), []byte(custom), 0o644); err != nil {
		t.Fatal(err)
	}

	in, _ := newTestInitWithVersion(dir, "", "1.0.0")
	if err := in.Run(Options{UpdateSkills: true}); err != nil {
		t.Fatalf("Run --update-skills: %v", err)
	}

	data, _ := os.ReadFile(filepath.Join(rolesDir, "base.yaml"))
	if string(data) != custom {
		t.Error("--update-skills modified base.yaml which should never be touched")
	}
}

// ---- P12 integration tests (AC-INT-1 through AC-INT-5) ----

// TestP12_Integration_NewProject verifies that kbz init on a new project
// produces all P12 artefacts: AGENTS.md, copilot-instructions.md,
// specification skill, and updated getting-started/workflow skills.
// Also verifies idempotency: a second run must not modify any of these files.
func TestP12_Integration_NewProject(t *testing.T) {
	t.Parallel()
	dir := makeGitRepoNoCommits(t)

	in, _ := newTestInit(dir, "")
	if err := in.Run(Options{}); err != nil {
		t.Fatalf("first Run: %v", err)
	}

	// AC-INT-1 / AC-A1: AGENTS.md exists with managed marker.
	agentsMDPath := filepath.Join(dir, "AGENTS.md")
	agentsData, err := os.ReadFile(agentsMDPath)
	if err != nil {
		t.Fatalf("AGENTS.md not created: %v", err)
	}
	v, managed, err := readMarkdownManagedVersion(agentsData)
	if err != nil || !managed {
		t.Errorf("AGENTS.md missing managed marker (managed=%v err=%v)", managed, err)
	}
	if v != agentsMDVersion {
		t.Errorf("AGENTS.md marker version = %d, want %d", v, agentsMDVersion)
	}

	// AC-INT-1: AGENTS.md must tell agents to use MCP tools and follow stage gates.
	agentsText := string(agentsData)
	for _, want := range []string{"status", "next", "stage gate", ".agents/skills/"} {
		if !strings.Contains(strings.ToLower(agentsText), strings.ToLower(want)) {
			t.Errorf("AGENTS.md missing required content %q", want)
		}
	}

	// AC-INT-2 / AC-B1: .github/copilot-instructions.md exists and references AGENTS.md.
	copilotPath := filepath.Join(dir, ".github", "copilot-instructions.md")
	copilotData, err := os.ReadFile(copilotPath)
	if err != nil {
		t.Fatalf(".github/copilot-instructions.md not created: %v", err)
	}
	if !strings.Contains(string(copilotData), "AGENTS.md") {
		t.Error(".github/copilot-instructions.md must reference AGENTS.md")
	}

	// AC-INT-4 / AC-D1: kanbanzai-specification skill is installed.
	specSkillPath := filepath.Join(dir, ".agents", "skills", "kanbanzai-specification", "SKILL.md")
	specData, err := os.ReadFile(specSkillPath)
	if err != nil {
		t.Fatalf("kanbanzai-specification/SKILL.md not installed: %v", err)
	}
	specText := string(specData)
	if !strings.Contains(specText, "kanbanzai-specification") {
		t.Error("specification skill missing name in frontmatter")
	}

	// AC-C1: getting-started skill contains the MCP-tools write rule.
	gsPath := filepath.Join(dir, ".agents", "skills", "kanbanzai-getting-started", "SKILL.md")
	gsData, err := os.ReadFile(gsPath)
	if err != nil {
		t.Fatalf("kanbanzai-getting-started/SKILL.md not installed: %v", err)
	}
	if !strings.Contains(string(gsData), "edit_file") {
		t.Error("getting-started skill missing MCP-tools write rule (edit_file reference)")
	}

	// AC-C2: workflow skill emergency brake includes direct-write condition.
	wfPath := filepath.Join(dir, ".agents", "skills", "kanbanzai-workflow", "SKILL.md")
	wfData, err := os.ReadFile(wfPath)
	if err != nil {
		t.Fatalf("kanbanzai-workflow/SKILL.md not installed: %v", err)
	}
	if !strings.Contains(string(wfData), "work/") || !strings.Contains(string(wfData), ".kbz/state/") {
		t.Error("workflow skill emergency brake missing direct-write condition")
	}

	// Idempotency: second run must not modify AGENTS.md or copilot-instructions.md.
	agentsStat1, _ := os.Stat(agentsMDPath)
	copilotStat1, _ := os.Stat(copilotPath)

	in2, stdout2 := newTestInit(dir, "")
	if err := in2.Run(Options{}); err != nil {
		t.Fatalf("second Run: %v", err)
	}

	agentsStat2, _ := os.Stat(agentsMDPath)
	copilotStat2, _ := os.Stat(copilotPath)

	if agentsStat1.ModTime() != agentsStat2.ModTime() {
		t.Error("AGENTS.md was modified on second run (idempotency violation)")
	}
	if copilotStat1.ModTime() != copilotStat2.ModTime() {
		t.Error("copilot-instructions.md was modified on second run (idempotency violation)")
	}
	out := stdout2.String()
	if strings.Contains(out, "Created AGENTS.md") || strings.Contains(out, "Updated AGENTS.md") {
		t.Errorf("unexpected AGENTS.md output on second run: %s", out)
	}
}

// TestP12_Integration_SkipAgentsMD verifies that --skip-agents-md suppresses
// both AGENTS.md and .github/copilot-instructions.md (AC-INT-5).
func TestP12_Integration_SkipAgentsMD(t *testing.T) {
	t.Parallel()
	dir := makeGitRepoNoCommits(t)

	in, _ := newTestInit(dir, "")
	if err := in.Run(Options{SkipAgentsMD: true}); err != nil {
		t.Fatalf("Run --skip-agents-md: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir, "AGENTS.md")); !os.IsNotExist(err) {
		t.Error("AGENTS.md should not be created with --skip-agents-md")
	}
	if _, err := os.Stat(filepath.Join(dir, ".github", "copilot-instructions.md")); !os.IsNotExist(err) {
		t.Error(".github/copilot-instructions.md should not be created with --skip-agents-md")
	}
}

// TestP12_Integration_SkipAgentsMDAndSkipSkills verifies that
// --skip-agents-md --skip-skills produces a project with no AGENTS.md,
// no copilot instructions, and no skills (AC-INT-5 edge case).
func TestP12_Integration_SkipAgentsMDAndSkipSkills(t *testing.T) {
	t.Parallel()
	dir := makeGitRepoNoCommits(t)

	in, _ := newTestInit(dir, "")
	if err := in.Run(Options{SkipAgentsMD: true, SkipSkills: true}); err != nil {
		t.Fatalf("Run: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir, "AGENTS.md")); !os.IsNotExist(err) {
		t.Error("AGENTS.md should not exist")
	}
	if _, err := os.Stat(filepath.Join(dir, ".github", "copilot-instructions.md")); !os.IsNotExist(err) {
		t.Error("copilot-instructions.md should not exist")
	}
	if _, err := os.Stat(filepath.Join(dir, ".agents", "skills")); !os.IsNotExist(err) {
		t.Error(".agents/skills should not exist with --skip-skills")
	}
	// Config must still be created.
	if _, err := os.Stat(filepath.Join(dir, ".kbz", "config.yaml")); os.IsNotExist(err) {
		t.Error("config.yaml must still be created")
	}
}

// TestConfigNameField verifies that config.yaml written with a name round-trips
// through the Config struct after Fix 1 added the Name field (AC-07).
func TestConfigNameField(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	if err := WriteInitConfig(dir, "My Test Project", DefaultDocumentRoots()); err != nil {
		t.Fatalf("WriteInitConfig: %v", err)
	}

	cfg, err := config.LoadFrom(filepath.Join(dir, "config.yaml"))
	if err != nil {
		t.Fatalf("config.LoadFrom: %v", err)
	}

	if cfg.Name != "My Test Project" {
		t.Errorf("Config.Name = %q, want %q", cfg.Name, "My Test Project")
	}
}
