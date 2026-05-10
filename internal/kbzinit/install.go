package kbzinit

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// installArtifact writes srcData to targetDir/<a.InstallPath> using the
// compareManaged decision logic. w receives status messages.
//
// When a.Marker.CurrentValue is empty, the caller must populate it before
// calling (e.g. from the binary version for skills/roles/stage-bindings).
func installArtifact(a Artifact, srcData []byte, w io.Writer, targetDir string) error {
	destPath := filepath.Join(targetDir, a.InstallPath)

	if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
		return fmt.Errorf("create directory for %s: %w", a.InstallPath, err)
	}

	existing, readErr := os.ReadFile(destPath)
	if readErr != nil {
		if !os.IsNotExist(readErr) {
			return fmt.Errorf("read %s: %w", a.InstallPath, readErr)
		}
		existing = nil
	}

	decision := compareManaged(existing, a.Marker)

	switch decision {
	case Create:
		if err := os.WriteFile(destPath, srcData, 0o644); err != nil {
			return fmt.Errorf("write %s: %w", a.InstallPath, err)
		}
		fmt.Fprintf(w, "Created %s\n", a.InstallPath)
	case Overwrite:
		if err := os.WriteFile(destPath, srcData, 0o644); err != nil {
			return fmt.Errorf("write %s: %w", a.InstallPath, err)
		}
		fmt.Fprintf(w, "Updated %s\n", a.InstallPath)
	case NoOp:
		// File exists, same or newer version — nothing to do.
	case WarnSkip:
		fmt.Fprintf(w, "Warning: %s exists but is not managed by kanbanzai. Skipping.\n", a.InstallPath)
	}

	return nil
}
