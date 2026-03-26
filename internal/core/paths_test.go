package core

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRootPath(t *testing.T) {
	t.Parallel()

	if got, want := RootPath(), InstanceRootDir; got != want {
		t.Fatalf("RootPath() = %q, want %q", got, want)
	}
}

func TestStatePath(t *testing.T) {
	t.Parallel()

	if got, want := StatePath(), filepath.Join(InstanceRootDir, StateDir); got != want {
		t.Fatalf("StatePath() = %q, want %q", got, want)
	}
}

func TestStatePath_IsUnderRootPath(t *testing.T) {
	t.Parallel()

	root := RootPath()
	state := StatePath()

	if filepath.Dir(state) != root {
		t.Fatalf("filepath.Dir(StatePath()) = %q, want %q", filepath.Dir(state), root)
	}
}

func TestCheckInitComplete_NoKbzDir(t *testing.T) {
	// When .kbz/ does not exist, CheckInitComplete should return nil.
	dir := t.TempDir()
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(origDir) })

	if err := CheckInitComplete(); err != nil {
		t.Fatalf("CheckInitComplete() with no .kbz/ = %v, want nil", err)
	}
}

func TestCheckInitComplete_FullyInitialised(t *testing.T) {
	// When .kbz/ exists and .init-complete is present, should return nil.
	dir := t.TempDir()
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(origDir) })

	kbzDir := filepath.Join(dir, InstanceRootDir)
	if err := os.MkdirAll(kbzDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(kbzDir, InitCompleteFile), []byte{}, 0o644); err != nil {
		t.Fatal(err)
	}

	if err := CheckInitComplete(); err != nil {
		t.Fatalf("CheckInitComplete() with sentinel present = %v, want nil", err)
	}
}

func TestCheckInitComplete_PartialInit(t *testing.T) {
	// When .kbz/ exists but .init-complete is absent, should return an error.
	dir := t.TempDir()
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(origDir) })

	kbzDir := filepath.Join(dir, InstanceRootDir)
	if err := os.MkdirAll(kbzDir, 0o755); err != nil {
		t.Fatal(err)
	}

	err = CheckInitComplete()
	if err == nil {
		t.Fatal("CheckInitComplete() with partial init = nil, want error")
	}

	msg := err.Error()
	if !contains(msg, "Partial initialisation detected") {
		t.Errorf("error message missing 'Partial initialisation detected': %s", msg)
	}
	if !contains(msg, "kanbanzai init") {
		t.Errorf("error message missing recovery action 'kanbanzai init': %s", msg)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
