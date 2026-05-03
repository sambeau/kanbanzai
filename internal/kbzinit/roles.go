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

const yamlManagedMarker = `kanbanzai-managed: "true"`

// installRoles installs all embedded role files into <kbzDir>/roles/.
// base.yaml is scaffold (create once, never overwrite).
// All other roles are managed (version-aware update via writeManagedRole).
func (i *Initializer) installRoles(kbzDir string) error {
	rolesDir := filepath.Join(kbzDir, "roles")
	if err := os.MkdirAll(rolesDir, 0o755); err != nil {
		return fmt.Errorf("create roles dir: %w", err)
	}

	entries, err := embeddedRoles.ReadDir("roles")
	if err != nil {
		return fmt.Errorf("read embedded roles: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}

		name := strings.TrimSuffix(entry.Name(), ".yaml")
		srcPath := "roles/" + entry.Name()
		srcData, err := embeddedRoles.ReadFile(srcPath)
		if err != nil {
			return fmt.Errorf("read embedded role %q: %w", name, err)
		}

		if name == "base" {
			if err := i.writeBaseRole(rolesDir, srcData); err != nil {
				return err
			}
		} else {
			if err := i.writeManagedRole(rolesDir, name, srcData); err != nil {
				return err
			}
		}
	}
	return nil
}

// writeBaseRole writes base.yaml only if it does not already exist.
// It accepts the embedded data as the source (not a hardcoded template).
func (i *Initializer) writeBaseRole(rolesDir string, srcData []byte) error {
	destPath := filepath.Join(rolesDir, "base.yaml")
	if _, err := os.Stat(destPath); err == nil {
		// Already exists — leave it alone.
		return nil
	}
	if err := os.WriteFile(destPath, srcData, 0o644); err != nil {
		return fmt.Errorf("write base.yaml: %w", err)
	}
	fmt.Fprintln(i.stdout, "Created .kbz/roles/base.yaml")
	return nil
}

// writeManagedRole writes a role file using version-aware managed logic.
// The embedded source carries "version: \"dev\"" which is replaced with
// the binary version at install time.
func (i *Initializer) writeManagedRole(rolesDir, name string, srcData []byte) error {
	destPath := filepath.Join(rolesDir, name+".yaml")

	// Replace the version placeholder in the embedded source.
	content := strings.ReplaceAll(string(srcData), `version: "dev"`, `version: "`+i.version+`"`)

	existing, readErr := os.ReadFile(destPath)
	if readErr != nil {
		if !os.IsNotExist(readErr) {
			return fmt.Errorf("read %s.yaml: %w", name, readErr)
		}
		// Create new.
		if err := os.WriteFile(destPath, []byte(content), 0o644); err != nil {
			return fmt.Errorf("write %s.yaml: %w", name, err)
		}
		fmt.Fprintf(i.stdout, "Created .kbz/roles/%s.yaml\n", name)
		return nil
	}

	// File exists — check managed marker.
	if !strings.Contains(string(existing), yamlManagedMarker) {
		fmt.Fprintf(i.stdout, "Warning: .kbz/roles/%s.yaml exists but is not managed by kanbanzai (no managed marker). Skipping.\n", name)
		return nil
	}

	// Extract existing version.
	existingVersion := extractYAMLVersion(existing)
	if existingVersion == i.version && i.version != "dev" {
		// At current version — no-op.
		return nil
	}

	// Older managed version, or dev build — overwrite.
	if err := os.WriteFile(destPath, []byte(content), 0o644); err != nil {
		return fmt.Errorf("update %s.yaml: %w", name, err)
	}
	fmt.Fprintf(i.stdout, "Updated .kbz/roles/%s.yaml\n", name)
	return nil
}

// updateManagedRoles updates all managed role files at older versions.
// It never touches base.yaml.
func (i *Initializer) updateManagedRoles(kbzDir string) error {
	rolesDir := filepath.Join(kbzDir, "roles")

	entries, err := embeddedRoles.ReadDir("roles")
	if err != nil {
		return fmt.Errorf("read embedded roles: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}

		name := strings.TrimSuffix(entry.Name(), ".yaml")
		if name == "base" {
			continue // never touch base.yaml
		}

		destPath := filepath.Join(rolesDir, name+".yaml")
		if _, err := os.Stat(destPath); os.IsNotExist(err) {
			continue // not yet installed, skip
		}

		srcPath := "roles/" + entry.Name()
		srcData, err := embeddedRoles.ReadFile(srcPath)
		if err != nil {
			return fmt.Errorf("read embedded role %q: %w", name, err)
		}

		if err := i.writeManagedRole(rolesDir, name, srcData); err != nil {
			return err
		}
	}
	return nil
}

// extractYAMLVersion extracts the version value from a role YAML file.
// Looks for a line like:   version: "1.0.0"
func extractYAMLVersion(data []byte) string {
	for _, line := range strings.Split(string(data), "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "version:") {
			v := strings.TrimPrefix(trimmed, "version:")
			v = strings.TrimSpace(v)
			v = strings.Trim(v, `"`)
			return v
		}
	}
	return ""
}
