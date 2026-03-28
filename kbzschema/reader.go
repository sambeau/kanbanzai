// Package kbzschema — see types.go for package documentation.
package kbzschema

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// kbzDir is the instance root directory name within a repository.
const kbzDir = ".kbz"

// stateDir is the state subdirectory within the instance root.
const stateDir = "state"

// ErrNotFound is returned when an entity record does not exist.
var ErrNotFound = errors.New("not found")

// Reader provides read-only access to a Kanbanzai repository's committed state.
// It reads directly from .kbz/state/ YAML files and does not depend on any
// internal Kanbanzai package or a running Kanbanzai server.
//
// All methods are safe to call concurrently; the Reader holds no mutable state.
type Reader struct {
	repoRoot  string // absolute path to repository root
	stateRoot string // absolute path to .kbz/state/
}

// NewReader creates a Reader rooted at the given repository root directory.
// repoRoot must be the directory that contains the .kbz/ subdirectory.
// An error is returned if the .kbz directory does not exist.
func NewReader(repoRoot string) (*Reader, error) {
	abs, err := filepath.Abs(repoRoot)
	if err != nil {
		return nil, fmt.Errorf("resolve repo root: %w", err)
	}

	kbzPath := filepath.Join(abs, kbzDir)
	if _, err := os.Stat(kbzPath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("not a kanbanzai repository: %s does not exist", kbzPath)
		}
		return nil, fmt.Errorf("stat %s: %w", kbzPath, err)
	}

	return &Reader{
		repoRoot:  abs,
		stateRoot: filepath.Join(abs, kbzDir, stateDir),
	}, nil
}

// ────────────────────────────────────────────────────────────────────────────
// Plan operations
// ────────────────────────────────────────────────────────────────────────────

// GetPlan returns the Plan with the given ID.
// Plan files are named {id}.yaml (e.g. P1-my-plan.yaml).
func (r *Reader) GetPlan(id string) (Plan, error) {
	if id == "" {
		return Plan{}, errors.New("plan ID is required")
	}
	path := filepath.Join(r.stateRoot, "plans", id+".yaml")
	var p Plan
	if err := readYAML(path, &p); err != nil {
		return Plan{}, fmt.Errorf("get plan %q: %w", id, err)
	}
	return p, nil
}

// ListPlans returns all Plan records in the repository.
func (r *Reader) ListPlans() ([]Plan, error) {
	dir := filepath.Join(r.stateRoot, "plans")
	entries, err := readDir(dir)
	if err != nil {
		return nil, fmt.Errorf("list plans: %w", err)
	}
	plans := make([]Plan, 0, len(entries))
	for _, name := range entries {
		id := strings.TrimSuffix(name, ".yaml")
		p, err := r.GetPlan(id)
		if err != nil {
			continue // skip unreadable records
		}
		plans = append(plans, p)
	}
	return plans, nil
}

// ────────────────────────────────────────────────────────────────────────────
// Feature operations
// ────────────────────────────────────────────────────────────────────────────

// GetFeature returns the Feature with the given ID.
// Feature files are named {id}-{slug}.yaml; the method scans for a prefix match.
func (r *Reader) GetFeature(id string) (Feature, error) {
	if id == "" {
		return Feature{}, errors.New("feature ID is required")
	}
	path, err := r.findEntityFile("features", id)
	if err != nil {
		return Feature{}, fmt.Errorf("get feature %q: %w", id, err)
	}
	var f Feature
	if err := readYAML(path, &f); err != nil {
		return Feature{}, fmt.Errorf("get feature %q: %w", id, err)
	}
	return f, nil
}

// ListFeaturesByPlan returns all Features whose parent field equals planID.
func (r *Reader) ListFeaturesByPlan(planID string) ([]Feature, error) {
	all, err := r.listFeatures()
	if err != nil {
		return nil, fmt.Errorf("list features by plan: %w", err)
	}
	var out []Feature
	for _, f := range all {
		if f.Parent == planID {
			out = append(out, f)
		}
	}
	return out, nil
}

// listFeatures returns every Feature in the repository.
func (r *Reader) listFeatures() ([]Feature, error) {
	dir := filepath.Join(r.stateRoot, "features")
	entries, err := readDir(dir)
	if err != nil {
		return nil, fmt.Errorf("list features: %w", err)
	}
	features := make([]Feature, 0, len(entries))
	for _, name := range entries {
		path := filepath.Join(dir, name)
		var f Feature
		if err := readYAML(path, &f); err != nil {
			continue
		}
		features = append(features, f)
	}
	return features, nil
}

// ────────────────────────────────────────────────────────────────────────────
// Task operations
// ────────────────────────────────────────────────────────────────────────────

// GetTask returns the Task with the given ID.
func (r *Reader) GetTask(id string) (Task, error) {
	if id == "" {
		return Task{}, errors.New("task ID is required")
	}
	path, err := r.findEntityFile("tasks", id)
	if err != nil {
		return Task{}, fmt.Errorf("get task %q: %w", id, err)
	}
	var t Task
	if err := readYAML(path, &t); err != nil {
		return Task{}, fmt.Errorf("get task %q: %w", id, err)
	}
	return t, nil
}

// ListTasksByFeature returns all Tasks whose parent_feature field equals featureID.
func (r *Reader) ListTasksByFeature(featureID string) ([]Task, error) {
	dir := filepath.Join(r.stateRoot, "tasks")
	entries, err := readDir(dir)
	if err != nil {
		return nil, fmt.Errorf("list tasks by feature: %w", err)
	}
	var out []Task
	for _, name := range entries {
		path := filepath.Join(dir, name)
		var t Task
		if err := readYAML(path, &t); err != nil {
			continue
		}
		if t.ParentFeature == featureID {
			out = append(out, t)
		}
	}
	return out, nil
}

// ────────────────────────────────────────────────────────────────────────────
// Bug operations
// ────────────────────────────────────────────────────────────────────────────

// GetBug returns the Bug with the given ID.
func (r *Reader) GetBug(id string) (Bug, error) {
	if id == "" {
		return Bug{}, errors.New("bug ID is required")
	}
	path, err := r.findEntityFile("bugs", id)
	if err != nil {
		return Bug{}, fmt.Errorf("get bug %q: %w", id, err)
	}
	var b Bug
	if err := readYAML(path, &b); err != nil {
		return Bug{}, fmt.Errorf("get bug %q: %w", id, err)
	}
	return b, nil
}

// ListBugs returns all Bug records in the repository.
func (r *Reader) ListBugs() ([]Bug, error) {
	dir := filepath.Join(r.stateRoot, "bugs")
	entries, err := readDir(dir)
	if err != nil {
		return nil, fmt.Errorf("list bugs: %w", err)
	}
	bugs := make([]Bug, 0, len(entries))
	for _, name := range entries {
		path := filepath.Join(dir, name)
		var b Bug
		if err := readYAML(path, &b); err != nil {
			continue
		}
		bugs = append(bugs, b)
	}
	return bugs, nil
}

// ────────────────────────────────────────────────────────────────────────────
// Document operations
// ────────────────────────────────────────────────────────────────────────────

// GetDocumentRecord returns the DocumentRecord with the given ID.
// Document IDs have the form {owner}/{slug}; files are stored as
// {owner}--{slug}.yaml (with / replaced by --).
func (r *Reader) GetDocumentRecord(id string) (DocumentRecord, error) {
	if id == "" {
		return DocumentRecord{}, errors.New("document ID is required")
	}
	path := filepath.Join(r.stateRoot, "documents", documentFileName(id))
	var d DocumentRecord
	if err := readYAML(path, &d); err != nil {
		return DocumentRecord{}, fmt.Errorf("get document record %q: %w", id, err)
	}
	return d, nil
}

// ListDocumentRecords returns all DocumentRecord entries in the repository.
func (r *Reader) ListDocumentRecords() ([]DocumentRecord, error) {
	dir := filepath.Join(r.stateRoot, "documents")
	entries, err := readDir(dir)
	if err != nil {
		return nil, fmt.Errorf("list document records: %w", err)
	}
	docs := make([]DocumentRecord, 0, len(entries))
	for _, name := range entries {
		path := filepath.Join(dir, name)
		var d DocumentRecord
		if err := readYAML(path, &d); err != nil {
			continue
		}
		docs = append(docs, d)
	}
	return docs, nil
}

// GetDocumentContent reads the file referenced by the document record with the
// given ID and returns its content. If the file's current SHA-256 hash differs
// from the hash recorded in the document record, a non-empty driftWarning is
// returned describing the discrepancy. Drift is never returned as an error;
// callers that do not care about drift may ignore the warning.
func (r *Reader) GetDocumentContent(id string) (content string, driftWarning string, err error) {
	rec, err := r.GetDocumentRecord(id)
	if err != nil {
		return "", "", err
	}

	docPath := filepath.Join(r.repoRoot, rec.Path)
	data, err := os.ReadFile(docPath)
	if err != nil {
		return "", "", fmt.Errorf("read document file %q: %w", docPath, err)
	}

	content = string(data)

	if rec.ContentHash != "" {
		h := sha256.New()
		_, _ = io.WriteString(h, content)
		currentHash := hex.EncodeToString(h.Sum(nil))
		if currentHash != rec.ContentHash {
			driftWarning = fmt.Sprintf(
				"document %q has drifted: recorded hash %s, current hash %s",
				id, rec.ContentHash, currentHash,
			)
		}
	}

	return content, driftWarning, nil
}

// ────────────────────────────────────────────────────────────────────────────
// Helpers
// ────────────────────────────────────────────────────────────────────────────

// findEntityFile scans the given subdirectory of stateRoot for a file whose
// name starts with "{id}-". Returns the full path or an ErrNotFound-wrapped
// error.
func (r *Reader) findEntityFile(subdir, id string) (string, error) {
	dir := filepath.Join(r.stateRoot, subdir)
	prefix := id + "-"

	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", fmt.Errorf("%w: %s directory does not exist", ErrNotFound, subdir)
		}
		return "", fmt.Errorf("read %s directory: %w", subdir, err)
	}

	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if strings.HasPrefix(e.Name(), prefix) && strings.HasSuffix(e.Name(), ".yaml") {
			return filepath.Join(dir, e.Name()), nil
		}
	}

	return "", fmt.Errorf("%w: %s/%s", ErrNotFound, subdir, id)
}

// readYAML reads and unmarshals a YAML file into v. Returns ErrNotFound if the
// file does not exist.
func readYAML(path string, v any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("%w: %s", ErrNotFound, path)
		}
		return fmt.Errorf("read %s: %w", path, err)
	}
	if err := yaml.Unmarshal(data, v); err != nil {
		return fmt.Errorf("parse %s: %w", path, err)
	}
	return nil
}

// readDir returns sorted YAML filenames from dir, silently returning nil if
// the directory does not exist.
func readDir(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("read directory %s: %w", dir, err)
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".yaml") {
			names = append(names, e.Name())
		}
	}
	return names, nil
}

// documentFileName converts a document ID (e.g. "FEAT-01ABC/my-design") to
// its on-disk filename (e.g. "FEAT-01ABC--my-design.yaml").
func documentFileName(id string) string {
	return strings.ReplaceAll(id, "/", "--") + ".yaml"
}
