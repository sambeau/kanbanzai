package context

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// RefreshRoleLastVerified reads a role YAML file, updates its last_verified
// timestamp to now (UTC, RFC 3339), and writes it back atomically.
func RefreshRoleLastVerified(path string, now time.Time) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read role file: %w", err)
	}

	var role Role
	if err := yaml.Unmarshal(data, &role); err != nil {
		return fmt.Errorf("parse role file: %w", err)
	}

	role.LastVerified = now.UTC().Format(time.RFC3339)

	out, err := yaml.Marshal(&role)
	if err != nil {
		return fmt.Errorf("marshal role file: %w", err)
	}

	return atomicWriteFile(path, out)
}

// RefreshSkillLastVerified reads the SKILL.md in skillDir, updates the
// last_verified field in the YAML frontmatter to now (UTC, RFC 3339), and
// writes it back atomically. The markdown body is preserved verbatim.
func RefreshSkillLastVerified(skillDir string, now time.Time) error {
	path := filepath.Join(skillDir, "SKILL.md")
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read skill file: %w", err)
	}

	content := string(data)
	timestamp := now.UTC().Format(time.RFC3339)
	newLine := "last_verified: \"" + timestamp + "\""

	// Find frontmatter boundaries (opening and closing ---).
	lines := strings.Split(content, "\n")
	fmStart := -1
	fmEnd := -1
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "---" {
			if fmStart == -1 {
				fmStart = i
			} else {
				fmEnd = i
				break
			}
		}
	}
	if fmStart == -1 || fmEnd == -1 {
		return fmt.Errorf("skill file %s: no YAML frontmatter found", path)
	}

	// Replace existing last_verified line, or insert before closing ---.
	replaced := false
	for i := fmStart + 1; i < fmEnd; i++ {
		if strings.HasPrefix(strings.TrimSpace(lines[i]), "last_verified:") {
			lines[i] = newLine
			replaced = true
			break
		}
	}
	if !replaced {
		expanded := make([]string, 0, len(lines)+1)
		expanded = append(expanded, lines[:fmEnd]...)
		expanded = append(expanded, newLine)
		expanded = append(expanded, lines[fmEnd:]...)
		lines = expanded
	}

	return atomicWriteFile(path, []byte(strings.Join(lines, "\n")))
}

// atomicWriteFile writes data to path by first writing to a temporary file in
// the same directory, then renaming. This ensures the target is never left in
// a partially-written state (NFR-002 atomicity).
func atomicWriteFile(path string, data []byte) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".refresh-*.tmp")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath) // clean up on failure; no-op after successful rename

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("write temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp file: %w", err)
	}
	return os.Rename(tmpPath, path)
}
