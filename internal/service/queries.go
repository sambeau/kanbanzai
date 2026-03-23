package service

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"kanbanzai/internal/model"
)

// ListFilteredInput contains filters for generic entity listing.
type ListFilteredInput struct {
	Type          string     // entity type (required)
	Status        string     // optional status filter
	Tags          []string   // optional tag filter (match any)
	CreatedAfter  *time.Time // optional date range
	CreatedBefore *time.Time
	UpdatedAfter  *time.Time
	UpdatedBefore *time.Time
	Parent        string // optional parent filter (for features)
}

// ListEntitiesFiltered returns entities of a given type matching the provided filters.
func (s *EntityService) ListEntitiesFiltered(input ListFilteredInput) ([]ListResult, error) {
	entityType := strings.ToLower(strings.TrimSpace(input.Type))
	if entityType == "" {
		return nil, errRequired("type")
	}

	all, err := s.List(entityType)
	if err != nil {
		return nil, err
	}

	var results []ListResult
	for _, r := range all {
		if !matchesFilteredInput(r, input) {
			continue
		}
		results = append(results, r)
	}

	return results, nil
}

// ListAllTags scans all entity types and returns a sorted list of unique tags.
func (s *EntityService) ListAllTags() ([]string, error) {
	seen := make(map[string]bool)

	for _, kind := range allQueryableKinds() {
		results, err := s.List(kind)
		if err != nil {
			// Directory may not exist for this kind yet — skip.
			continue
		}
		for _, r := range results {
			for _, tag := range tagsFromState(r.State) {
				seen[tag] = true
			}
		}
	}

	tags := make([]string, 0, len(seen))
	for tag := range seen {
		tags = append(tags, tag)
	}
	sort.Strings(tags)
	return tags, nil
}

// ListEntitiesByTag scans all entity types and returns entities that have the given tag.
func (s *EntityService) ListEntitiesByTag(tag string) ([]ListResult, error) {
	tag = strings.ToLower(strings.TrimSpace(tag))
	if tag == "" {
		return nil, errRequired("tag")
	}

	var results []ListResult
	for _, kind := range allQueryableKinds() {
		entities, err := s.List(kind)
		if err != nil {
			continue
		}
		for _, r := range entities {
			for _, t := range tagsFromState(r.State) {
				if t == tag {
					results = append(results, r)
					break
				}
			}
		}
	}

	return results, nil
}

// CrossEntityQuery finds all tasks belonging to features under the given plan.
// It loads all features, filters by parent == planID, then loads tasks for each feature.
func (s *EntityService) CrossEntityQuery(planID string) ([]ListResult, error) {
	planID = strings.TrimSpace(planID)
	if planID == "" {
		return nil, errRequired("plan_id")
	}

	features, err := s.List(string(model.EntityKindFeature))
	if err != nil {
		return nil, err
	}

	// Collect feature IDs that belong to this plan.
	featureIDs := make(map[string]bool)
	for _, f := range features {
		parent := stringFromState(f.State, "parent")
		if parent == "" {
			parent = stringFromState(f.State, "epic")
		}
		if parent == planID {
			featureIDs[f.ID] = true
		}
	}

	if len(featureIDs) == 0 {
		return nil, nil
	}

	tasks, err := s.List(string(model.EntityKindTask))
	if err != nil {
		return nil, err
	}

	var results []ListResult
	for _, t := range tasks {
		parentFeature := stringFromState(t.State, "parent_feature")
		if featureIDs[parentFeature] {
			results = append(results, t)
		}
	}

	return results, nil
}

// matchesFilteredInput checks whether a ListResult matches all the filters in a ListFilteredInput.
func matchesFilteredInput(r ListResult, input ListFilteredInput) bool {
	if input.Status != "" {
		status := stringFromState(r.State, "status")
		if status != input.Status {
			return false
		}
	}

	if len(input.Tags) > 0 {
		entityTags := tagsFromState(r.State)
		if !matchesAnyTag(entityTags, input.Tags) {
			return false
		}
	}

	if input.Parent != "" {
		parent := stringFromState(r.State, "parent")
		if parent == "" {
			parent = stringFromState(r.State, "epic")
		}
		if parent != input.Parent {
			return false
		}
	}

	if input.CreatedAfter != nil || input.CreatedBefore != nil {
		created := parseTimeFromState(r.State, "created")
		if created.IsZero() {
			return false
		}
		if input.CreatedAfter != nil && created.Before(*input.CreatedAfter) {
			return false
		}
		if input.CreatedBefore != nil && created.After(*input.CreatedBefore) {
			return false
		}
	}

	if input.UpdatedAfter != nil || input.UpdatedBefore != nil {
		updated := parseTimeFromState(r.State, "updated")
		if updated.IsZero() {
			return false
		}
		if input.UpdatedAfter != nil && updated.Before(*input.UpdatedAfter) {
			return false
		}
		if input.UpdatedBefore != nil && updated.After(*input.UpdatedBefore) {
			return false
		}
	}

	return true
}

// matchesAnyTag returns true if any of filterTags appears in entityTags.
func matchesAnyTag(entityTags, filterTags []string) bool {
	set := make(map[string]bool, len(entityTags))
	for _, t := range entityTags {
		set[t] = true
	}
	for _, t := range filterTags {
		if set[t] {
			return true
		}
	}
	return false
}

// parseTimeFromState parses an RFC3339 timestamp from the entity state map.
func parseTimeFromState(state map[string]any, key string) time.Time {
	s := stringFromState(state, key)
	if s == "" {
		return time.Time{}
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Time{}
	}
	return t
}

// allQueryableKinds returns the entity kinds to scan for cross-type queries.
func allQueryableKinds() []string {
	return []string{
		string(model.EntityKindPlan),
		string(model.EntityKindEpic),
		string(model.EntityKindFeature),
		string(model.EntityKindTask),
		string(model.EntityKindBug),
		string(model.EntityKindDecision),
	}
}

func errRequired(name string) error {
	return fmt.Errorf("%s is required", name)
}
