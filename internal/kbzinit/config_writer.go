package kbzinit

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// DocumentRoot represents a single document root entry in the init config.
type DocumentRoot struct {
	Path        string `yaml:"path"`
	DefaultType string `yaml:"default_type"`
}

// DefaultDocumentRoots returns the canonical work/ document roots for new projects.
func DefaultDocumentRoots() []DocumentRoot {
	return []DocumentRoot{
		{Path: "work/design", DefaultType: "design"},
		{Path: "work/spec", DefaultType: "specification"},
		{Path: "work/plan", DefaultType: "plan"},
		{Path: "work/dev", DefaultType: "dev-plan"},
		{Path: "work/research", DefaultType: "research"},
		{Path: "work/report", DefaultType: "report"},
		{Path: "work/review", DefaultType: "report"},
		{Path: "work/retro", DefaultType: "retrospective"},
	}
}

// InferDocType infers a document type from a directory path segment.
// The path basename is matched against known type keywords; everything else
// defaults to "design".
func InferDocType(path string) string {
	switch filepath.Base(path) {
	case "spec":
		return "specification"
	case "plan":
		return "plan"
	case "dev":
		return "dev-plan"
	case "research":
		return "research"
	case "report":
		return "report"
	case "reports":
		return "report"
	case "retro":
		return "retrospective"
	default:
		return "design"
	}
}

// initPrefixEntry is a prefix entry as written by init (prefix + name only).
type initPrefixEntry struct {
	Prefix string `yaml:"prefix"`
	Name   string `yaml:"name"`
}

// initDocuments is the documents section of the init config file.
type initDocuments struct {
	Roots []DocumentRoot `yaml:"roots"`
}

// initFileConfig is the minimal config structure written by kanbanzai init.
// It only includes the fields that init is responsible for: version, prefixes,
// and documents. Optional operational fields (branch_tracking, cleanup, etc.)
// are omitted intentionally — they are added later by other commands or the user.
type initFileConfig struct {
	Version   string            `yaml:"version"`
	Name      string            `yaml:"name"`
	Prefixes  []initPrefixEntry `yaml:"prefixes"`
	Documents initDocuments     `yaml:"documents"`
}

// WriteInitConfig writes a minimal .kbz/config.yaml to kbzDir with the given
// document roots. It creates kbzDir if it does not exist. The file is written
// with 2-space YAML indentation to match the canonical spec layout.
func WriteInitConfig(kbzDir string, name string, roots []DocumentRoot) error {
	cfg := initFileConfig{
		Version: "2",
		Name:    name,
		Prefixes: []initPrefixEntry{
			{Prefix: "P", Name: "Plan"},
		},
		Documents: initDocuments{
			Roots: roots,
		},
	}

	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(cfg); err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	if err := enc.Close(); err != nil {
		return fmt.Errorf("finalise config: %w", err)
	}

	if err := os.MkdirAll(kbzDir, 0o755); err != nil {
		return fmt.Errorf("cannot create '.kbz/' directory: check that the current user has write access to this directory")
	}

	configPath := filepath.Join(kbzDir, "config.yaml")
	if err := os.WriteFile(configPath, buf.Bytes(), 0o644); err != nil {
		return fmt.Errorf("cannot write '%s': check that the current user has write access to this directory", configPath)
	}

	return nil
}
