package kbzinit

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
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
	if err := WriteInitConfig(dir, DefaultDocumentRoots()); err != nil {
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
	if err := WriteInitConfig(dir, DefaultDocumentRoots()); err != nil {
		t.Fatalf("WriteInitConfig: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "config.yaml"))
	if err != nil {
		t.Fatalf("read config: %v", err)
	}

	// Canonical YAML from spec §5.2 — 2-space indent, version quoted.
	want := strings.TrimSpace(`
version: "2"
prefixes:
  - prefix: P
    name: Plan
documents:
  roots:
    - path: work/design
      default_type: design
    - path: work/spec
      default_type: specification
    - path: work/dev
      default_type: dev-plan
    - path: work/research
      default_type: research
    - path: work/reports
      default_type: report`)

	got := strings.TrimSpace(string(data))
	if got != want {
		t.Errorf("config.yaml content mismatch\ngot:\n%s\n\nwant:\n%s", got, want)
	}
}

func TestWriteInitConfig_CreatesDirIfMissing(t *testing.T) {
	parent := t.TempDir()
	kbzDir := filepath.Join(parent, ".kbz")
	// kbzDir does not exist yet.
	if err := WriteInitConfig(kbzDir, DefaultDocumentRoots()); err != nil {
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
	if err := WriteInitConfig(dir, roots); err != nil {
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
	if err := in.Run(Options{}); err != nil {
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
prefixes:
  - prefix: P
    name: Plan
documents:
  roots:
    - path: work/design
      default_type: design
    - path: work/spec
      default_type: specification
    - path: work/dev
      default_type: dev-plan
    - path: work/research
      default_type: research
    - path: work/reports
      default_type: report`)

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
	for _, sub := range []string{"work/design", "work/spec", "work/dev", "work/research", "work/reports"} {
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
	for _, sub := range []string{"work/design", "work/spec", "work/dev", "work/research", "work/reports"} {
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
	for _, sub := range []string{"work/design", "work/spec", "work/dev", "work/research", "work/reports"} {
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
	if err := WriteInitConfig(kbzDir, DefaultDocumentRoots()); err != nil {
		t.Fatalf("pre-create config: %v", err)
	}

	in, _ := newTestInit(dir, "")
	if err := in.Run(Options{}); err != nil {
		t.Fatalf("Run: %v", err)
	}

	for _, sub := range []string{"work/design", "work/spec", "work/dev", "work/research", "work/reports"} {
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
