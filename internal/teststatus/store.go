package teststatus

import (
	"io/fs"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const statePath = ".kbz/state/test-status.yaml"

// stateFile returns the absolute path to the test-status state file.
func stateFile(repoRoot string) string {
	return filepath.Join(repoRoot, statePath)
}

// ReadRecord parses .kbz/state/test-status.yaml. If the file does not exist it
// returns a zero-value Record with Result set to ResultUnknown.
func ReadRecord(repoRoot string) (Record, error) {
	raw, err := os.ReadFile(stateFile(repoRoot))
	if err != nil {
		if os.IsNotExist(err) {
			return Record{Result: ResultUnknown}, nil
		}
		return Record{}, err
	}

	var rec Record
	if err := yaml.Unmarshal(raw, &rec); err != nil {
		return Record{}, err
	}
	if rec.Result == "" {
		rec.Result = ResultUnknown
	}
	return rec, nil
}

// WriteRecord atomically writes rec to .kbz/state/test-status.yaml by writing to
// a temporary file first and then renaming it into place.
func WriteRecord(repoRoot string, rec Record) error {
	path := stateFile(repoRoot)
	dir := filepath.Dir(path)

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	tmp, err := os.CreateTemp(dir, "test-status-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()

	if err := yaml.NewEncoder(tmp).Encode(rec); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return err
	}
	return os.Rename(tmpName, path)
}

// IsStale returns true if any .go file under repoRoot has a modification time
// newer than rec.LastRun, or if rec.LastRun is nil. It excludes
// .worktrees/, vendor/, .git/, and hidden directories.
func IsStale(repoRoot string, rec Record) (bool, error) {
	if rec.LastRun == nil {
		return true, nil
	}

	cutoff := *rec.LastRun
	found := false

	err := walkGoFiles(repoRoot, func(_ string, info fs.FileInfo) error {
		if info.ModTime().After(cutoff) {
			found = true
			return fs.SkipAll
		}
		return nil
	})
	return found, err
}

// walkGoFiles calls fn for each .go file under root, recursing into
// subdirectories but skipping .worktrees/, vendor/, .git/, and hidden dirs.
func walkGoFiles(root string, fn func(path string, info fs.FileInfo) error) error {
	return filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			base := d.Name()
			if base == ".worktrees" || base == "vendor" || base == ".git" {
				return fs.SkipDir
			}
			if len(base) > 1 && base[0] == '.' {
				return fs.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) != ".go" {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		return fn(path, info)
	})
}
