// Package context surfacer.go — concrete KnowledgeSurfacer implementation.
//
// Replaces the NoOpSurfacer for production use. Loads knowledge entries via an
// injected loader function, matches by file path / tag / "always" criteria,
// scores with recency-weighted confidence, caps at 10, and returns entries
// ordered ascending by score (highest last, exploiting recency bias).
package context

import (
	"log"
	"time"

	"github.com/sambeau/kanbanzai/internal/knowledge"
)

const defaultMaxSurfacedEntries = 10

// EntryLoader loads all non-retired knowledge entries as raw field maps.
type EntryLoader func() ([]map[string]any, error)

// Surfacer implements the KnowledgeSurfacer interface using the matching engine,
// recency-weighted scoring, and cap logic from internal/knowledge.
type Surfacer struct {
	loadEntries EntryLoader
	capTracker  *knowledge.CapTracker
	now         func() time.Time
}

// NewSurfacer creates a production KnowledgeSurfacer.
// capTracker may be nil (cap tracking is skipped).
// now may be nil (defaults to time.Now).
func NewSurfacer(loader EntryLoader, capTracker *knowledge.CapTracker, now func() time.Time) *Surfacer {
	if now == nil {
		now = time.Now
	}
	return &Surfacer{
		loadEntries: loader,
		capTracker:  capTracker,
		now:         now,
	}
}

// Surface implements KnowledgeSurfacer.Surface.
func (s *Surfacer) Surface(input SurfaceInput) ([]SurfacedEntry, error) {
	entries, err := s.loadEntries()
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
			log.Printf("[knowledge-surfacer] excluded entry %s (topic: %s) due to cap", ex.ID, ex.Topic)
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

// recordCap delegates to the cap tracker if available.
func (s *Surfacer) recordCap(scope string, capHit bool) {
	if s.capTracker == nil || scope == "" {
		return
	}
	if err := s.capTracker.RecordAssembly(scope, capHit); err != nil {
		log.Printf("[knowledge-surfacer] cap tracker error: %v", err)
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
				prefix = dirPrefix(trimTrailingSlash(prefix))
			} else {
				d = dirPrefix(trimTrailingSlash(d))
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

func trimTrailingSlash(s string) string {
	if len(s) > 0 && s[len(s)-1] == '/' {
		return s[:len(s)-1]
	}
	return s
}
