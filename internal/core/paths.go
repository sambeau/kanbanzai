package core

import (
	"fmt"
	"os"
	"path/filepath"
)

const (
	// InstanceRootDir is the repository-local Kanbanzai instance root.
	InstanceRootDir = ".kbz"

	// StateDir is the canonical entity state directory within the instance root.
	StateDir = "state"

	// InitCompleteFile is the sentinel file written as the final step of
	// a successful kanbanzai init. Its absence when .kbz/ exists indicates
	// a previously interrupted initialisation.
	InitCompleteFile = ".init-complete"
)

// RootPath returns the repository-local Kanbanzai instance root path.
func RootPath() string {
	return InstanceRootDir
}

// StatePath returns the canonical state directory path within the instance root.
func StatePath() string {
	return filepath.Join(InstanceRootDir, StateDir)
}

// CheckInitComplete verifies that the Kanbanzai instance is fully initialised.
// If .kbz/ exists but the .init-complete sentinel is absent, it returns an
// error describing the partial initialisation and suggesting recovery actions.
// If .kbz/ does not exist at all, it returns nil (the caller may handle the
// missing-instance case separately).
func CheckInitComplete() error {
	kbzDir := InstanceRootDir

	if _, err := os.Stat(kbzDir); os.IsNotExist(err) {
		// No .kbz/ directory — not our problem here.
		return nil
	}

	sentinel := filepath.Join(kbzDir, InitCompleteFile)
	if _, err := os.Stat(sentinel); os.IsNotExist(err) {
		return fmt.Errorf(
			"partial initialisation detected: a previous 'kanbanzai init' did not complete successfully. " +
				"Re-run 'kanbanzai init' to complete setup, or remove the '.kbz/' directory and start over",
		)
	}

	return nil
}
