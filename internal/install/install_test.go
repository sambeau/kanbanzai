package install

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestWriteReadRoundTrip(t *testing.T) {
	t.Parallel()

	root := t.TempDir()

	// Create the .kbz directory so WriteRecord can write into it.
	if err := os.MkdirAll(filepath.Join(root, ".kbz"), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	beforeWrite := time.Now().UTC().Truncate(time.Second)

	err := WriteRecord(root, "abc123def456", "/usr/local/bin/kanbanzai", "makefile")
	if err != nil {
		t.Fatalf("WriteRecord() error = %v", err)
	}

	afterWrite := time.Now().UTC().Add(time.Second).Truncate(time.Second)

	rec, err := ReadRecord(root)
	if err != nil {
		t.Fatalf("ReadRecord() error = %v", err)
	}
	if rec == nil {
		t.Fatal("ReadRecord() returned nil, want non-nil")
	}

	if rec.GitSHA != "abc123def456" {
		t.Errorf("GitSHA = %q, want %q", rec.GitSHA, "abc123def456")
	}
	if rec.InstalledBy != "makefile" {
		t.Errorf("InstalledBy = %q, want %q", rec.InstalledBy, "makefile")
	}
	if rec.BinaryPath != "/usr/local/bin/kanbanzai" {
		t.Errorf("BinaryPath = %q, want %q", rec.BinaryPath, "/usr/local/bin/kanbanzai")
	}
	if rec.InstalledAt.Before(beforeWrite) || rec.InstalledAt.After(afterWrite) {
		t.Errorf("InstalledAt = %v, want between %v and %v", rec.InstalledAt, beforeWrite, afterWrite)
	}
	if rec.InstalledAt.Location().String() != "UTC" {
		t.Errorf("InstalledAt location = %v, want UTC", rec.InstalledAt.Location())
	}
}

func TestReadRecord_Missing(t *testing.T) {
	t.Parallel()

	root := t.TempDir()

	rec, err := ReadRecord(root)
	if err != nil {
		t.Fatalf("ReadRecord() error = %v, want nil", err)
	}
	if rec != nil {
		t.Errorf("ReadRecord() = %+v, want nil", rec)
	}
}

func TestWriteRecord_AtomicWrite(t *testing.T) {
	t.Parallel()

	root := t.TempDir()

	if err := os.MkdirAll(filepath.Join(root, ".kbz"), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	err := WriteRecord(root, "deadbeef", "/opt/bin/kanbanzai", "go-install")
	if err != nil {
		t.Fatalf("WriteRecord() error = %v", err)
	}

	expectedPath := filepath.Join(root, ".kbz", "last-install.yaml")
	info, err := os.Stat(expectedPath)
	if err != nil {
		t.Fatalf("Stat(%q) error = %v", expectedPath, err)
	}
	if info.Size() == 0 {
		t.Error("file is empty after WriteRecord")
	}

	// Verify file content is valid YAML by reading it back.
	rec, err := ReadRecord(root)
	if err != nil {
		t.Fatalf("ReadRecord() after atomic write error = %v", err)
	}
	if rec == nil {
		t.Fatal("ReadRecord() returned nil after WriteRecord")
	}
	if rec.GitSHA != "deadbeef" {
		t.Errorf("GitSHA = %q, want %q", rec.GitSHA, "deadbeef")
	}
}

func TestWriteRecord_FieldOrder(t *testing.T) {
	t.Parallel()

	root := t.TempDir()

	if err := os.MkdirAll(filepath.Join(root, ".kbz"), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	err := WriteRecord(root, "abc123", "/usr/local/bin/kbz", "makefile")
	if err != nil {
		t.Fatalf("WriteRecord() error = %v", err)
	}

	data, err := os.ReadFile(filepath.Join(root, ".kbz", "last-install.yaml"))
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	content := string(data)

	// Verify canonical YAML field order: git_sha, installed_at, installed_by, binary_path.
	gitSHAIdx := indexOf(content, "git_sha:")
	installedAtIdx := indexOf(content, "installed_at:")
	installedByIdx := indexOf(content, "installed_by:")
	binaryPathIdx := indexOf(content, "binary_path:")

	if gitSHAIdx < 0 || installedAtIdx < 0 || installedByIdx < 0 || binaryPathIdx < 0 {
		t.Fatalf("missing expected fields in YAML output:\n%s", content)
	}

	if !(gitSHAIdx < installedAtIdx && installedAtIdx < installedByIdx && installedByIdx < binaryPathIdx) {
		t.Errorf("field order is not git_sha < installed_at < installed_by < binary_path in:\n%s", content)
	}

	// Verify trailing newline.
	if len(data) == 0 || data[len(data)-1] != '\n' {
		t.Error("YAML output does not end with trailing newline")
	}
}

func indexOf(s, substr string) int {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
