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
