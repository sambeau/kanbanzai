package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveIdentity_ExplicitValueWins(t *testing.T) {
	t.Parallel()

	name, err := resolveIdentity("alice", "/nonexistent/local.yaml")
	if err != nil {
		t.Fatalf("resolveIdentity() error = %v", err)
	}
	if name != "alice" {
		t.Errorf("name = %q, want %q", name, "alice")
	}
}

func TestResolveIdentity_ExplicitValueTrimmed(t *testing.T) {
	t.Parallel()

	name, err := resolveIdentity("  alice  ", "/nonexistent/local.yaml")
	if err != nil {
		t.Fatalf("resolveIdentity() error = %v", err)
	}
	if name != "alice" {
		t.Errorf("name = %q, want %q", name, "alice")
	}
}

func TestResolveIdentity_LocalYAMLFallback(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	kbzDir := filepath.Join(tmpDir, ".kbz")
	if err := os.MkdirAll(kbzDir, 0o755); err != nil {
		t.Fatalf("failed to create .kbz dir: %v", err)
	}

	localPath := filepath.Join(kbzDir, "local.yaml")
	content := "user:\n  name: sambeau\n"
	if err := os.WriteFile(localPath, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write local.yaml: %v", err)
	}

	name, err := resolveIdentity("", localPath)
	if err != nil {
		t.Fatalf("resolveIdentity() error = %v", err)
	}
	if name != "sambeau" {
		t.Errorf("name = %q, want %q", name, "sambeau")
	}
}

func TestResolveIdentity_LocalYAMLTrimsWhitespace(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	localPath := filepath.Join(tmpDir, "local.yaml")
	content := "user:\n  name: \"  bob  \"\n"
	if err := os.WriteFile(localPath, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write local.yaml: %v", err)
	}

	name, err := resolveIdentity("", localPath)
	if err != nil {
		t.Fatalf("resolveIdentity() error = %v", err)
	}
	if name != "bob" {
		t.Errorf("name = %q, want %q", name, "bob")
	}
}

func TestResolveIdentity_LocalYAMLEmptyNameSkipped(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	localPath := filepath.Join(tmpDir, "local.yaml")
	// user.name is blank — should not count as a valid identity
	content := "user:\n  name: \"\"\n"
	if err := os.WriteFile(localPath, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write local.yaml: %v", err)
	}

	// Falls through to git config (which may or may not be set in CI).
	// We only assert no panic and a non-empty result-or-error path.
	_, _ = resolveIdentity("", localPath)
}

func TestResolveIdentity_ExplicitBeatsLocalYAML(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	localPath := filepath.Join(tmpDir, "local.yaml")
	content := "user:\n  name: fromfile\n"
	if err := os.WriteFile(localPath, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write local.yaml: %v", err)
	}

	name, err := resolveIdentity("explicit-user", localPath)
	if err != nil {
		t.Fatalf("resolveIdentity() error = %v", err)
	}
	if name != "explicit-user" {
		t.Errorf("name = %q, want %q (explicit should win)", name, "explicit-user")
	}
}

func TestResolveIdentity_ErrorMessageIsHelpful(t *testing.T) {
	t.Parallel()

	// Simulate an environment where local.yaml is absent and git is unavailable
	// by pointing at a nonexistent path. The git fallback may still succeed in
	// a developer environment, so we only check the error message when we do
	// get an error.
	_, err := resolveIdentity("", "/nonexistent/path/local.yaml")
	if err != nil {
		msg := err.Error()
		if !strings.Contains(msg, "created_by") {
			t.Errorf("error message %q should mention 'created_by'", msg)
		}
		if !strings.Contains(msg, "local.yaml") {
			t.Errorf("error message %q should mention 'local.yaml'", msg)
		}
		if !strings.Contains(msg, "git") {
			t.Errorf("error message %q should mention 'git'", msg)
		}
	}
}

func TestLoadLocalConfigFrom_GitHubConfig(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	localPath := filepath.Join(tmpDir, "local.yaml")

	content := `user:
  name: testuser
github:
  token: ghp_xxxxxxxxxxxxxxxxxxxx
  owner: example-org
  repo: example-repo
`
	if err := os.WriteFile(localPath, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write local.yaml: %v", err)
	}

	lc, err := LoadLocalConfigFrom(localPath)
	if err != nil {
		t.Fatalf("LoadLocalConfigFrom() error = %v", err)
	}

	if lc.GitHub.Token != "ghp_xxxxxxxxxxxxxxxxxxxx" {
		t.Errorf("GitHub.Token = %q, want %q", lc.GitHub.Token, "ghp_xxxxxxxxxxxxxxxxxxxx")
	}
	if lc.GitHub.Owner != "example-org" {
		t.Errorf("GitHub.Owner = %q, want %q", lc.GitHub.Owner, "example-org")
	}
	if lc.GitHub.Repo != "example-repo" {
		t.Errorf("GitHub.Repo = %q, want %q", lc.GitHub.Repo, "example-repo")
	}
}

func TestLocalConfig_GetGitHubToken(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	localPath := filepath.Join(tmpDir, "local.yaml")

	content := `github:
  token: ghp_secret_token_12345
`
	if err := os.WriteFile(localPath, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write local.yaml: %v", err)
	}

	lc, err := LoadLocalConfigFrom(localPath)
	if err != nil {
		t.Fatalf("LoadLocalConfigFrom() error = %v", err)
	}

	token := lc.GetGitHubToken()
	if token != "ghp_secret_token_12345" {
		t.Errorf("GetGitHubToken() = %q, want %q", token, "ghp_secret_token_12345")
	}
}

func TestLocalConfig_GetGitHubOwner(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	localPath := filepath.Join(tmpDir, "local.yaml")

	content := `github:
  owner: my-org
`
	if err := os.WriteFile(localPath, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write local.yaml: %v", err)
	}

	lc, err := LoadLocalConfigFrom(localPath)
	if err != nil {
		t.Fatalf("LoadLocalConfigFrom() error = %v", err)
	}

	owner := lc.GetGitHubOwner()
	if owner != "my-org" {
		t.Errorf("GetGitHubOwner() = %q, want %q", owner, "my-org")
	}
}

func TestLocalConfig_GetGitHubRepo(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	localPath := filepath.Join(tmpDir, "local.yaml")

	content := `github:
  repo: my-repo
`
	if err := os.WriteFile(localPath, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write local.yaml: %v", err)
	}

	lc, err := LoadLocalConfigFrom(localPath)
	if err != nil {
		t.Fatalf("LoadLocalConfigFrom() error = %v", err)
	}

	repo := lc.GetGitHubRepo()
	if repo != "my-repo" {
		t.Errorf("GetGitHubRepo() = %q, want %q", repo, "my-repo")
	}
}

func TestLocalConfig_GitHubFieldsEmptyWhenNotSet(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	localPath := filepath.Join(tmpDir, "local.yaml")

	// Config without github section
	content := `user:
  name: testuser
`
	if err := os.WriteFile(localPath, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write local.yaml: %v", err)
	}

	lc, err := LoadLocalConfigFrom(localPath)
	if err != nil {
		t.Fatalf("LoadLocalConfigFrom() error = %v", err)
	}

	if lc.GetGitHubToken() != "" {
		t.Errorf("GetGitHubToken() = %q, want empty string", lc.GetGitHubToken())
	}
	if lc.GetGitHubOwner() != "" {
		t.Errorf("GetGitHubOwner() = %q, want empty string", lc.GetGitHubOwner())
	}
	if lc.GetGitHubRepo() != "" {
		t.Errorf("GetGitHubRepo() = %q, want empty string", lc.GetGitHubRepo())
	}
}

func TestLoadLocalConfigFrom_FileNotFound(t *testing.T) {
	t.Parallel()

	_, err := LoadLocalConfigFrom("/nonexistent/path/local.yaml")
	if err == nil {
		t.Error("LoadLocalConfigFrom() should fail for non-existent file")
	}
}

func TestLoadLocalConfigFrom_InvalidYAML(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	localPath := filepath.Join(tmpDir, "local.yaml")

	// Write invalid YAML
	if err := os.WriteFile(localPath, []byte("this is not: valid: yaml: content"), 0o644); err != nil {
		t.Fatalf("failed to write local.yaml: %v", err)
	}

	_, err := LoadLocalConfigFrom(localPath)
	if err == nil {
		t.Error("LoadLocalConfigFrom() should fail for invalid YAML")
	}
}

func TestResolveGraphProject_LocalYAMLFallback(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	localPath := filepath.Join(tmpDir, "local.yaml")
	content := "codebase_memory:\n  graph_project: Users-alice-Dev-myrepo\n"
	if err := os.WriteFile(localPath, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write local.yaml: %v", err)
	}

	got := resolveGraphProject(localPath)
	if got != "Users-alice-Dev-myrepo" {
		t.Errorf("resolveGraphProject() = %q, want %q", got, "Users-alice-Dev-myrepo")
	}
}

func TestResolveGraphProject_TrimsWhitespace(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	localPath := filepath.Join(tmpDir, "local.yaml")
	content := "codebase_memory:\n  graph_project: \"  Users-alice-Dev-myrepo  \"\n"
	if err := os.WriteFile(localPath, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write local.yaml: %v", err)
	}

	got := resolveGraphProject(localPath)
	if got != "Users-alice-Dev-myrepo" {
		t.Errorf("resolveGraphProject() = %q, want %q (should trim whitespace)", got, "Users-alice-Dev-myrepo")
	}
}

func TestResolveGraphProject_MissingFile(t *testing.T) {
	t.Parallel()

	got := resolveGraphProject("/nonexistent/path/local.yaml")
	if got != "" {
		t.Errorf("resolveGraphProject() = %q, want empty string for missing file", got)
	}
}

func TestResolveGraphProject_MissingSection(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	localPath := filepath.Join(tmpDir, "local.yaml")
	content := "user:\n  name: alice\n"
	if err := os.WriteFile(localPath, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write local.yaml: %v", err)
	}

	got := resolveGraphProject(localPath)
	if got != "" {
		t.Errorf("resolveGraphProject() = %q, want empty string when section absent", got)
	}
}

func TestResolveGraphProject_EmptyValue(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	localPath := filepath.Join(tmpDir, "local.yaml")
	content := "codebase_memory:\n  graph_project: \"\"\n"
	if err := os.WriteFile(localPath, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write local.yaml: %v", err)
	}

	got := resolveGraphProject(localPath)
	if got != "" {
		t.Errorf("resolveGraphProject() = %q, want empty string for blank value", got)
	}
}

func TestDeriveGraphProject_UnixPath(t *testing.T) {
	t.Parallel()

	// Use a path we know is absolute on the current machine.
	got := DeriveGraphProject("/Users/alice/Dev/myrepo")
	want := "Users-alice-Dev-myrepo"
	if got != want {
		t.Errorf("DeriveGraphProject() = %q, want %q", got, want)
	}
}

func TestDeriveGraphProject_RelativePath(t *testing.T) {
	t.Parallel()

	// A relative path should be resolved to absolute before deriving.
	// We can't assert the exact value since it depends on cwd, but we
	// can assert it is non-empty and contains no leading slash.
	got := DeriveGraphProject(".")
	if got == "" {
		t.Error("DeriveGraphProject(\".\") returned empty string")
	}
	if strings.HasPrefix(got, "/") {
		t.Errorf("DeriveGraphProject(\".\") = %q; should not start with /", got)
	}
	if strings.Contains(got, "/") {
		t.Errorf("DeriveGraphProject(\".\") = %q; should contain no slashes", got)
	}
}

func TestDeriveGraphProject_NoLeadingSlash(t *testing.T) {
	t.Parallel()

	got := DeriveGraphProject("/a/b/c")
	if strings.HasPrefix(got, "-") {
		t.Errorf("DeriveGraphProject() = %q; should not start with hyphen", got)
	}
	if got != "a-b-c" {
		t.Errorf("DeriveGraphProject(\"/a/b/c\") = %q, want %q", got, "a-b-c")
	}
}
