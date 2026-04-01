package knowledge

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"sync"
)

// ScopeCompactionInfo describes a scope that has hit the auto-surfacing cap
// enough consecutive times to warrant compaction.
type ScopeCompactionInfo struct {
	Scope           string
	ConsecutiveHits int
}

// CapTracker records consecutive cap-hit events per scope for health
// compaction recommendations. State is persisted to a JSON file under
// the .kbz cache directory.
type CapTracker struct {
	path string
	mu   sync.Mutex
	data map[string]int
}

// NewCapTracker creates a CapTracker that persists state to cacheDir.
func NewCapTracker(cacheDir string) *CapTracker {
	p := filepath.Join(cacheDir, "knowledge-cap-tracker.json")
	data := make(map[string]int)

	if raw, err := os.ReadFile(p); err == nil {
		_ = json.Unmarshal(raw, &data)
		if data == nil {
			data = make(map[string]int)
		}
	}

	return &CapTracker{
		path: p,
		data: data,
	}
}

// RecordAssembly records whether the auto-surfacing cap was hit for a scope.
// Consecutive hits increment the counter; a below-cap assembly resets it.
func (t *CapTracker) RecordAssembly(scope string, capHit bool) error {
	if scope == "" {
		return nil
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	if capHit {
		t.data[scope]++
	} else {
		delete(t.data, scope)
	}

	return t.persist()
}

// ScopesNeedingCompaction returns scopes that have hit the cap 3 or more
// consecutive times, sorted by scope name.
func (t *CapTracker) ScopesNeedingCompaction() []ScopeCompactionInfo {
	t.mu.Lock()
	defer t.mu.Unlock()

	var result []ScopeCompactionInfo
	for scope, count := range t.data {
		if count >= 3 {
			result = append(result, ScopeCompactionInfo{
				Scope:           scope,
				ConsecutiveHits: count,
			})
		}
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Scope < result[j].Scope
	})

	return result
}

func (t *CapTracker) persist() error {
	if err := os.MkdirAll(filepath.Dir(t.path), 0o755); err != nil {
		return err
	}

	raw, err := json.Marshal(t.data)
	if err != nil {
		return err
	}

	tmp := t.path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o644); err != nil {
		return err
	}

	return os.Rename(tmp, t.path)
}
