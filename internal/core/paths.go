package core

import "path/filepath"

const (
	// InstanceRootDir is the repository-local Kanbanzai instance root.
	InstanceRootDir = ".kbz"

	// StateDir is the canonical entity state directory within the instance root.
	StateDir = "state"
)

// RootPath returns the repository-local Kanbanzai instance root path.
func RootPath() string {
	return InstanceRootDir
}

// StatePath returns the canonical state directory path within the instance root.
func StatePath() string {
	return filepath.Join(InstanceRootDir, StateDir)
}
