// Package install manages the install record stored at .kbz/last-install.yaml.
package install

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"

	"kanbanzai/internal/fsutil"
)

// InstallRecord holds metadata about the most recent binary installation.
type InstallRecord struct {
	GitSHA      string    `yaml:"git_sha"`
	InstalledAt time.Time `yaml:"installed_at"`
	InstalledBy string    `yaml:"installed_by"`
	BinaryPath  string    `yaml:"binary_path"`
}

const recordFile = "last-install.yaml"

// WriteRecord writes an install record to .kbz/last-install.yaml atomically.
func WriteRecord(root, gitSHA, binaryPath, installedBy string) error {
	rec := InstallRecord{
		GitSHA:      gitSHA,
		InstalledAt: time.Now().UTC(),
		InstalledBy: installedBy,
		BinaryPath:  binaryPath,
	}

	data, err := yaml.Marshal(&rec)
	if err != nil {
		return fmt.Errorf("marshal install record: %w", err)
	}

	// Ensure trailing newline
	if len(data) == 0 || data[len(data)-1] != '\n' {
		data = append(data, '\n')
	}

	dir := filepath.Join(root, ".kbz")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create .kbz directory: %w", err)
	}

	path := filepath.Join(dir, recordFile)
	if err := fsutil.WriteFileAtomic(path, data, 0o644); err != nil {
		return fmt.Errorf("write install record: %w", err)
	}

	return nil
}

// ReadRecord reads the install record from .kbz/last-install.yaml.
// Returns nil, nil if the file does not exist.
func ReadRecord(root string) (*InstallRecord, error) {
	path := filepath.Join(root, ".kbz", recordFile)

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("read install record: %w", err)
	}

	var rec InstallRecord
	if err := yaml.Unmarshal(data, &rec); err != nil {
		return nil, fmt.Errorf("unmarshal install record: %w", err)
	}

	return &rec, nil
}
