// Package context surfacer.go — concrete KnowledgeSurfacer implementation.
//
// Replaces the NoOpSurfacer for production use. Loads knowledge entries via an
// injected loader function, matches by file path / tag / "always" criteria,
// scores with recency-weighted confidence, caps at 10, and returns entries
// ordered ascending by score (highest last, exploiting recency bias).
//
// Caching: when a GenReader is provided, Surface() caches the last loaded
// entry set keyed by generation token. On each call it compares the current
// generation with the cached one; on mismatch it reloads via EntryLoader and
// updates the cache. This satisfies REQ-002 and REQ-003 (AC-002 through AC-004).
package context

import (
	"context"

	"strings"
	"log/slog"
	"sync"
	"time"

	"github.com/sambeau/kanbanzai/internal/knowledge"
)

const defaultMaxSurfacedEntries = 10

// EntryLoader loads all non-retired knowledge entries as raw field maps.
type EntryLoader func() ([]map[string]any, error)

// GenReader returns an O(1) generation token for the knowledge store.
// When it returns an error, caching is skipped for that call.
// May be nil, in which case entries are always reloaded (no caching).
type GenReader func() (string, error)

// Surfacer implements the KnowledgeSurfacer interface using the matching engine,
// recency-weighted scoring, and cap logic from internal/knowledge.
type Surfacer struct {
	loadEntries EntryLoader
	capTracker  *knowledge.CapTracker
	now         func() time.Time
	genReader   GenReader

	mu            sync.Mutex
	cachedGen     string
	cachedEntries []map[string]any
}

// NewSurfacer creates a production KnowledgeSurfacer.
// capTracker may be nil (cap tracking is skipped).
// now may be nil (defaults to time.Now).
// genReader may be nil (caching disabled — entries reloaded on every call).
func NewSurfacer(loader EntryLoader, capTracker *knowledge.CapTracker, now func() time.Time, genReader GenReader) *Surfacer {
	if now == nil {
		now = time.Now
	}
	return &Surfacer{
		loadEntries: loader,
		capTracker:  capTracker,
		now:         now,
		genReader:   genReader,
	}
}

// Surface implements KnowledgeSurfacer.Surface.
func (s *Surfacer) Surface(ctx context.Context, input SurfaceInput) ([]SurfacedEntry, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	entries, err := s.resolveEntries()
	if err != nil {
		// Graceful degradation (NFR-004): return empty result on load failure.
		return nil, nil
	}
	if len(entries) == 0 {
		return nil, nil
	}

	matched := knowledge.MatchEntries(entries, knowledge.MatchInput{
		FilePaths: input.FilePaths,
		RoleTags:  input.RoleTags,
	})
	if len(matched) == 0 {
		s.recordCap("", false)
		return nil, nil
	}

	now := s.now()
	surfaced, excluded := knowledge.RankAndCap(matched, now, defaultMaxSurfacedEntries)

	capHit := len(excluded) > 0
	scope := deriveScopeForTracking(input.FilePaths)
	s.recordCap(scope, capHit)

	if len(excluded) > 0 {
		for _, ex := range excluded {
			slog.Info("excluded entry due to cap", "component", "knowledge-surfacer", "id", ex.ID, "topic", ex.Topic)
		}
	}

	result := make([]SurfacedEntry, len(surfaced))
	for i, se := range surfaced {
		result[i] = SurfacedEntry{
			ID:      se.ID,
			Topic:   se.Topic,
			Content: se.Content,
			Score:   se.Score,
		}
	}
	return result, nil
}

// resolveEntries returns the current entry set, using the cache when the
// generation token matches. Falls back to a fresh load on any error.
func (s *Surfacer) resolveEntries() ([]map[string]any, error) {
	if s.genReader == nil {
		// No generation reader — always reload (caching disabled).
		return s.loadEntries()
	}

	gen, err := s.genReader()
	if err != nil {
		// Generation read failed — load fresh without updating cache.
		return s.loadEntries()
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if gen != "" && gen == s.cachedGen {
		// Cache hit: generation token unchanged.
		return s.cachedEntries, nil
	}

	// Cache miss: reload and store.
	entries, err := s.loadEntries()
	if err != nil {
		return nil, err
	}

	s.cachedGen = gen
	s.cachedEntries = entries
	return entries, nil
}

// recordCap delegates to the cap tracker if available.
func (s *Surfacer) recordCap(scope string, capHit bool) {
	if s.capTracker == nil || scope == "" {
		return
	}
	if err := s.capTracker.RecordAssembly(scope, capHit); err != nil {
		slog.Info("cap tracker error", "component", "knowledge-surfacer", "error", err)
	}
}

// deriveScopeForTracking picks the most specific common scope from the task's
// file paths. If all paths share a common directory prefix, that prefix is used.
// Otherwise, "mixed" is used. If there are no paths, returns empty string.
func deriveScopeForTracking(filePaths []string) string {
	if len(filePaths) == 0 {
		return ""
	}
	if len(filePaths) == 1 {
		return dirPrefix(filePaths[0])
	}
	prefix := dirPrefix(filePaths[0])
	for _, fp := range filePaths[1:] {
		d := dirPrefix(fp)
		for prefix != "" && prefix != d {
			if len(prefix) > len(d) {
				prefix = dirPrefix(strings.TrimSuffix(prefix, "/"))
			} else {
				d = dirPrefix(strings.TrimSuffix(d, "/"))
			}
		}
		if prefix == "" {
			return "mixed"
		}
	}
	return prefix
}

// dirPrefix returns the directory portion of a path, with trailing slash.
func dirPrefix(path string) string {
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' {
			return path[:i+1]
		}
	}
	return ""
}

