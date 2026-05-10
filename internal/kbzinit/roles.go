package kbzinit

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

//go:embed roles
var embeddedRoles embed.FS

// installRoles installs all Role artifacts from the Manifest into
// <kbzDir>/roles/. base.yaml is scaffold (create once, never overwrite).
// All other roles are managed (version-aware update via installArtifact).
func (i *Initializer) installRoles(kbzDir string) error {
	for _, a := range Manifest {
		if a.Kind != Role {
			continue
		}

		srcData, err := embeddedRoles.ReadFile(a.EmbedPath)
		if err != nil {
			return fmt.Errorf("read embedded role %s: %w", a.Name, err)
		}

		// base.yaml (the only Role with an empty MarkerSpec) is scaffold —
		// write once, never overwrite.
		if a.Marker == (MarkerSpec{}) {
			destPath := filepath.Join(kbzDir, a.InstallPath)
			if _, err := os.Stat(destPath); err == nil {
				continue // already exists
			}
			if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
				return fmt.Errorf("create roles dir: %w", err)
			}
			if err := os.WriteFile(destPath, srcData, 0o644); err != nil {
				return fmt.Errorf("write base.yaml: %w", err)
			}
			fmt.Fprintln(i.stdout, "Created .kbz/roles/base.yaml")
			continue
		}

		// Managed role: replace dev version placeholder with binary version.
		content := strings.ReplaceAll(string(srcData), `version: "dev"`, `version: "`+i.version+`"`)
		a.Marker.CurrentValue = i.version
		if err := installArtifact(a, []byte(content), i.stdout, kbzDir); err != nil {
			return err
		}
	}
	return nil
}

// updateManagedRoles updates all managed role files (excluding base.yaml)
// that are at older versions. It never touches base.yaml.
func (i *Initializer) updateManagedRoles(kbzDir string) error {
	for _, a := range Manifest {
		if a.Kind != Role || a.Marker == (MarkerSpec{}) {
			continue
		}
		destPath := filepath.Join(kbzDir, a.InstallPath)
		if _, err := os.Stat(destPath); os.IsNotExist(err) {
			continue // not yet installed, skip
		}

		srcData, err := embeddedRoles.ReadFile(a.EmbedPath)
		if err != nil {
			return fmt.Errorf("read embedded role %s: %w", a.Name, err)
		}

		content := strings.ReplaceAll(string(srcData), `version: "dev"`, `version: "`+i.version+`"`)
		a.Marker.CurrentValue = i.version
		if err := installArtifact(a, []byte(content), i.stdout, kbzDir); err != nil {
			return err
		}
	}
	return nil
}
