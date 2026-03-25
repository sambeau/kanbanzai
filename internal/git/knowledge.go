package git

import (
	"fmt"
	"time"
)

// KnowledgeFieldGitAnchors is the field name for git anchors in knowledge entries.
const KnowledgeFieldGitAnchors = "git_anchors"

// KnowledgeFieldLastConfirmed is the field name for last confirmed timestamp.
const KnowledgeFieldLastConfirmed = "last_confirmed"

// KnowledgeFieldLastUsed is the field name for last used timestamp.
const KnowledgeFieldLastUsed = "last_used"

// KnowledgeFieldTTLDays is the field name for TTL in days.
const KnowledgeFieldTTLDays = "ttl_days"

// KnowledgeFieldTTLExpiresAt is the field name for TTL expiration timestamp.
const KnowledgeFieldTTLExpiresAt = "ttl_expires_at"

// ExtractAnchors extracts git anchors from knowledge entry fields.
// Returns an empty slice if no anchors are present or if the field is malformed.
func ExtractAnchors(fields map[string]any) []GitAnchor {
	if fields == nil {
		return nil
	}

	anchorsRaw, ok := fields[KnowledgeFieldGitAnchors]
	if !ok || anchorsRaw == nil {
		return nil
	}

	var anchors []GitAnchor

	switch v := anchorsRaw.(type) {
	case []string:
		// Direct string slice
		for _, path := range v {
			if path != "" {
				anchors = append(anchors, GitAnchor{Path: path})
			}
		}
	case []any:
		// YAML unmarshals to []any, so handle that case
		for _, item := range v {
			if path, ok := item.(string); ok && path != "" {
				anchors = append(anchors, GitAnchor{Path: path})
			}
		}
	}

	return anchors
}

// SetLastConfirmed updates the last_confirmed timestamp in fields.
// The timestamp is stored as an RFC3339 string.
func SetLastConfirmed(fields map[string]any, timestamp time.Time) {
	if fields == nil {
		return
	}
	fields[KnowledgeFieldLastConfirmed] = timestamp.UTC().Format(time.RFC3339)
}

// GetLastConfirmed extracts the last_confirmed timestamp from fields.
// Returns zero time if not present or if the field is malformed.
func GetLastConfirmed(fields map[string]any) time.Time {
	if fields == nil {
		return time.Time{}
	}

	raw, ok := fields[KnowledgeFieldLastConfirmed]
	if !ok || raw == nil {
		return time.Time{}
	}

	str, ok := raw.(string)
	if !ok || str == "" {
		return time.Time{}
	}

	t, err := time.Parse(time.RFC3339, str)
	if err != nil {
		return time.Time{}
	}

	return t
}

// SetLastUsed updates the last_used timestamp in fields.
// The timestamp is stored as an RFC3339 string.
func SetLastUsed(fields map[string]any, timestamp time.Time) {
	if fields == nil {
		return
	}
	fields[KnowledgeFieldLastUsed] = timestamp.UTC().Format(time.RFC3339)
}

// GetLastUsed extracts the last_used timestamp from fields.
// Returns zero time if not present or if the field is malformed.
func GetLastUsed(fields map[string]any) time.Time {
	if fields == nil {
		return time.Time{}
	}

	raw, ok := fields[KnowledgeFieldLastUsed]
	if !ok || raw == nil {
		return time.Time{}
	}

	str, ok := raw.(string)
	if !ok || str == "" {
		return time.Time{}
	}

	t, err := time.Parse(time.RFC3339, str)
	if err != nil {
		return time.Time{}
	}

	return t
}

// SetTTL updates the TTL-related fields in a knowledge entry.
// It sets ttl_days and computes ttl_expires_at based on lastUsed + ttlDays.
func SetTTL(fields map[string]any, ttlDays int, lastUsed time.Time) {
	if fields == nil {
		return
	}
	fields[KnowledgeFieldTTLDays] = ttlDays
	expiresAt := lastUsed.AddDate(0, 0, ttlDays)
	fields[KnowledgeFieldTTLExpiresAt] = expiresAt.UTC().Format(time.RFC3339)
}

// GetTTLDays extracts the TTL in days from fields.
// Returns 0 if not present or if the field is malformed.
func GetTTLDays(fields map[string]any) int {
	if fields == nil {
		return 0
	}

	raw, ok := fields[KnowledgeFieldTTLDays]
	if !ok || raw == nil {
		return 0
	}

	switch v := raw.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	default:
		return 0
	}
}

// GetTTLExpiresAt extracts the TTL expiration timestamp from fields.
// Returns zero time if not present or if the field is malformed.
func GetTTLExpiresAt(fields map[string]any) time.Time {
	if fields == nil {
		return time.Time{}
	}

	raw, ok := fields[KnowledgeFieldTTLExpiresAt]
	if !ok || raw == nil {
		return time.Time{}
	}

	str, ok := raw.(string)
	if !ok || str == "" {
		return time.Time{}
	}

	t, err := time.Parse(time.RFC3339, str)
	if err != nil {
		return time.Time{}
	}

	return t
}

// SetGitAnchors sets the git_anchors field in a knowledge entry.
func SetGitAnchors(fields map[string]any, anchors []GitAnchor) {
	if fields == nil {
		return
	}
	if len(anchors) == 0 {
		delete(fields, KnowledgeFieldGitAnchors)
		return
	}

	paths := make([]string, len(anchors))
	for i, a := range anchors {
		paths[i] = a.Path
	}
	fields[KnowledgeFieldGitAnchors] = paths
}

// CheckEntryStaleness checks if a knowledge entry is stale based on its fields.
// This is a convenience function that extracts anchors and last_confirmed from
// fields and calls CheckStaleness.
func CheckEntryStaleness(repoPath string, fields map[string]any) (StalenessInfo, error) {
	anchors := ExtractAnchors(fields)
	lastConfirmed := GetLastConfirmed(fields)
	return CheckStaleness(repoPath, anchors, lastConfirmed)
}

// ValidateAnchorPaths validates that all anchor paths are non-empty and
// don't contain invalid characters.
func ValidateAnchorPaths(anchors []GitAnchor) error {
	for _, anchor := range anchors {
		if anchor.Path == "" {
			return fmt.Errorf("anchor path cannot be empty")
		}
		// Paths should be relative (not start with /)
		if len(anchor.Path) > 0 && anchor.Path[0] == '/' {
			return fmt.Errorf("anchor path must be relative: %s", anchor.Path)
		}
	}
	return nil
}
