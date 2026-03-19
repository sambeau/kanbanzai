package core

import (
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
