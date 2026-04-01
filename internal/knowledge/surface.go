package knowledge

import (
	"sort"
	"strings"
	"time"

	"github.com/sambeau/kanbanzai/internal/git"
)

// MatchInput holds the context used to match knowledge entries against a task.
type MatchInput struct {
	FilePaths []string
	RoleTags  []string
}

// MatchedEntry is a knowledge entry that matched a task's context.
type MatchedEntry struct {
	ID          string
	Topic       string
	Content     string
	Scope       string
	Tags        []string
	Status      string
	Confidence  float64
	ConfirmedAt time.Time
	CreatedAt   time.Time
}

// MatchEntries filters and returns knowledge entries relevant to the given input.
// Entries match if their scope is a prefix of any file path, if they share a tag
// with the role, if they are tagged "always", or if their scope is "project".
// Retired entries are excluded. Results are deduplicated by ID and sorted.
func MatchEntries(entries []map[string]any, input MatchInput) []MatchedEntry {
	seen := make(map[string]struct{})
	var result []MatchedEntry

	for _, entry := range entries {
		status := getFieldString(entry, "status")
		if status == "retired" {
			continue
		}

		id := getFieldString(entry, "id")
		if id == "" {
			continue
		}
		if _, dup := seen[id]; dup {
			continue
		}

		scope := getFieldString(entry, "scope")
		tags := extractTags(entry)

		if !matchesFilePath(scope, input.FilePaths) &&
			!matchesTags(tags, input.RoleTags) &&
			!matchesAlways(scope, tags) {
			continue
		}

		seen[id] = struct{}{}
		result = append(result, toMatchedEntry(entry, id, scope, status, tags))
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].ID < result[j].ID
	})

	if result == nil {
		result = []MatchedEntry{}
	}
	return result
}

func matchesFilePath(scope string, filePaths []string) bool {
	if scope == "" || scope == "project" {
		return false
	}
	for _, fp := range filePaths {
		if strings.HasPrefix(fp, scope) {
			return true
		}
	}
	return false
}

func matchesTags(entryTags, roleTags []string) bool {
	for _, et := range entryTags {
		for _, rt := range roleTags {
			if strings.EqualFold(et, rt) {
				return true
			}
		}
	}
	return false
}

func matchesAlways(scope string, tags []string) bool {
	if scope == "project" {
		return true
	}
	for _, t := range tags {
		if strings.EqualFold(t, "always") {
			return true
		}
	}
	return false
}

func extractTags(entry map[string]any) []string {
	raw, ok := entry["tags"]
	if !ok || raw == nil {
		return nil
	}
	switch v := raw.(type) {
	case []string:
		return v
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				out = append(out, s)
			}
		}
		return out
	}
	return nil
}

func toMatchedEntry(entry map[string]any, id, scope, status string, tags []string) MatchedEntry {
	confidence := GetConfidence(entry)
	confirmedAt := git.GetLastConfirmed(entry)
	createdAt := GetCreatedAt(entry)

	return MatchedEntry{
		ID:          id,
		Topic:       getFieldString(entry, "topic"),
		Content:     getFieldString(entry, "content"),
		Scope:       scope,
		Tags:        tags,
		Status:      status,
		Confidence:  confidence,
		ConfirmedAt: confirmedAt,
		CreatedAt:   createdAt,
	}
}
