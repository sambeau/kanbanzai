package service

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/sambeau/kanbanzai/internal/config"
	"github.com/sambeau/kanbanzai/internal/model"
	"github.com/sambeau/kanbanzai/internal/storage"
	"github.com/sambeau/kanbanzai/internal/validate"
)

// CreatePlanInput contains the fields needed to create a new Plan.
type CreatePlanInput struct {
	// Prefix is the single-character prefix for the Plan ID.
	// Must be declared in the prefix registry.
	Prefix string
	// Slug is the URL-friendly identifier appended after the number.
	Slug string
	// Name is the human-readable name.
	Name string
	// Summary is a brief description of the Plan.
	Summary string
	// CreatedBy identifies who created the Plan.
	CreatedBy string
	// Tags are optional freeform tags for organisation.
	Tags []string
}

// UpdatePlanInput contains the fields that can be updated on a Plan.
type UpdatePlanInput struct {
	ID      string
	Slug    string
	Name    *string
	Summary *string
	Design  *string
	Tags    []string
}

// CreatePlan creates a new Plan entity.
func (s *EntityService) CreatePlan(input CreatePlanInput) (CreateResult, error) {
	if err := validateRequired(
		field("prefix", input.Prefix),
		field("slug", input.Slug),
		field("name", input.Name),
		field("summary", input.Summary),
		field("created_by", input.CreatedBy),
	); err != nil {
		return CreateResult{}, err
	}

	planName, nameErr := validate.ValidateName(input.Name)
	if nameErr != nil {
		return CreateResult{}, nameErr
	}

	// Load and validate prefix registry (fall back to defaults if no config file exists,
	// so that Plan creation works in fresh projects before kbz init has been run).
	cfg := config.LoadOrDefault()

	prefix := strings.TrimSpace(input.Prefix)
	if !cfg.IsActivePrefix(prefix) {
		if cfg.IsValidPrefix(prefix) {
			return CreateResult{}, fmt.Errorf("prefix %q is retired and cannot be used for new Plans", prefix)
		}
		return CreateResult{}, fmt.Errorf("undeclared prefix %q: add it to .kbz/config.yaml prefixes", prefix)
	}

	// Get next available number for this prefix
	nextNum, err := cfg.NextPlanNumber(prefix, func() ([]string, error) {
		return s.listPlanIDs()
	})
	if err != nil {
		return CreateResult{}, fmt.Errorf("allocate plan number: %w", err)
	}

	slug := normalizeSlug(input.Slug)
	idValue := fmt.Sprintf("%s%d-%s", prefix, nextNum, slug)

	now := s.now()
	entity := model.Plan{
		ID:             idValue,
		Slug:           slug,
		Name:           planName,
		Status:         model.PlanStatusProposed,
		Summary:        strings.TrimSpace(input.Summary),
		Tags:           normalizeTags(input.Tags),
		Created:        now,
		CreatedBy:      strings.TrimSpace(input.CreatedBy),
		Updated:        now,
		NextFeatureSeq: 1,
	}

	if err := validate.ValidateInitialState(validate.EntityPlan, string(entity.Status)); err != nil {
		return CreateResult{}, err
	}

	result, err := s.writePlan(entity)
	if err != nil {
		return result, err
	}
	s.cacheUpsertFromResult(result)
	return result, nil
}

// AllocateFeatureDisplayIDInPlan allocates the next feature display ID in a plan
// by incrementing its next_feature_seq counter. Returns the new display ID (e.g. "P37-F5").
// This is used by kbz move Mode 2 when re-parenting a feature to a different plan.
func (s *EntityService) AllocateFeatureDisplayIDInPlan(planID string) (string, error) {
	planResult, err := s.GetPlan(planID)
	if err != nil {
		return "", fmt.Errorf("load plan %s: %w", planID, err)
	}

	seq := intFromState(planResult.State, "next_feature_seq", 1)

	_, planNum, _ := model.ParsePlanID(planID)
	displayID := fmt.Sprintf("P%s-F%d", planNum, seq)

	planResult.State["next_feature_seq"] = seq + 1
	planRecord := storage.EntityRecord{
		Type:   string(model.EntityKindPlan),
		ID:     planResult.ID,
		Slug:   planResult.Slug,
		Fields: planResult.State,
	}
	if _, err := s.store.Write(planRecord); err != nil {
		return "", fmt.Errorf("increment plan sequence for %s: %w", planID, err)
	}

	return displayID, nil
}

// GetPlan retrieves a Plan by ID.
func (s *EntityService) GetPlan(id string) (ListResult, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return ListResult{}, fmt.Errorf("plan ID is required")
	}

	if !model.IsPlanID(id) {
		return ListResult{}, fmt.Errorf("invalid Plan ID format: %s", id)
	}

	_, _, slug := model.ParsePlanID(id)
	return s.loadPlan(id, slug)
}

// ListPlans returns all Plans, optionally filtered.
func (s *EntityService) ListPlans(filters PlanFilters) ([]ListResult, error) {
	dir := filepath.Join(s.root, "plans")
	entries, err := listDirectory(dir)
	if err != nil {
		return nil, err
	}

	var results []ListResult
	for _, entry := range entries {
		if !strings.HasSuffix(entry, ".yaml") {
			continue
		}

		id, slug, err := parsePlanFileName(entry)
		if err != nil {
			continue
		}

		result, err := s.loadPlan(id, slug)
		if err != nil {
			continue
		}

		// Apply filters
		if !matchesPlanFilters(result, filters) {
			continue
		}

		results = append(results, result)
	}

	return results, nil
}

// PlanFilters contains optional filters for listing Plans.
type PlanFilters struct {
	Status string
	Prefix string
	Tags   []string
}

// UpdatePlanStatus transitions a Plan to a new status.
func (s *EntityService) UpdatePlanStatus(id, slug, newStatus string) (ListResult, error) {
	if err := validateRequired(
		field("id", id),
		field("slug", slug),
		field("status", newStatus),
	); err != nil {
		return ListResult{}, err
	}

	// Load existing plan
	result, err := s.loadPlan(id, slug)
	if err != nil {
		return ListResult{}, err
	}

	currentStatus := stringFromState(result.State, "status")
	if err := validate.ValidateTransition(validate.EntityPlan, currentStatus, newStatus); err != nil {
		return ListResult{}, err
	}

	// proposed → active shortcut: precondition — at least one feature must be in a
	// post-designing state (specifying, dev-planning, developing, reviewing, done).
	if currentStatus == string(model.PlanStatusProposed) && newStatus == string(model.PlanStatusActive) {
		n, err := s.countPostDesigningFeaturesForPlan(id)
		if err != nil {
			return ListResult{}, fmt.Errorf("checking post-designing features: %w", err)
		}
		if n == 0 {
			return ListResult{}, fmt.Errorf(
				"proposed → active shortcut requires at least one feature in post-designing state " +
					"(specifying, dev-planning, developing, reviewing, or done); " +
					"use proposed → designing instead",
			)
		}
		// Append system-generated override record to the plan's audit trail.
		existing := planOverridesFromState(result.State)
		or := model.OverrideRecord{
			FromStatus: currentStatus,
			ToStatus:   newStatus,
			Reason:     fmt.Sprintf("proposed → active shortcut: %d feature(s) in post-designing state at transition time", n),
			Timestamp:  s.now(),
		}
		result.State["overrides"] = overrideRecordsToAny(append(existing, or))
	}

	// Update status and updated timestamp
	result.State["status"] = newStatus
	result.State["updated"] = s.now().Format(time.RFC3339)

	// Write back
	record := storage.EntityRecord{
		Type:   string(model.EntityKindPlan),
		ID:     id,
		Slug:   slug,
		Fields: result.State,
	}

	path, err := s.store.Write(record)
	if err != nil {
		return ListResult{}, fmt.Errorf("write plan: %w", err)
	}

	result.Path = path
	s.cacheUpsertFromResult(CreateResult{
		Type:  result.Type,
		ID:    result.ID,
		Slug:  result.Slug,
		Path:  result.Path,
		State: result.State,
	})

	return result, nil
}

// UpdatePlan updates mutable fields on a Plan.
func (s *EntityService) UpdatePlan(input UpdatePlanInput) (ListResult, error) {
	if err := validateRequired(
		field("id", input.ID),
		field("slug", input.Slug),
	); err != nil {
		return ListResult{}, err
	}

	// Load existing plan
	result, err := s.loadPlan(input.ID, input.Slug)
	if err != nil {
		return ListResult{}, err
	}

	// Apply updates
	if input.Name != nil {
		result.State["name"] = strings.TrimSpace(*input.Name)
	}
	if input.Summary != nil {
		result.State["summary"] = strings.TrimSpace(*input.Summary)
	}
	if input.Design != nil {
		if *input.Design == "" {
			delete(result.State, "design")
		} else {
			result.State["design"] = strings.TrimSpace(*input.Design)
		}
	}
	if input.Tags != nil {
		tags := normalizeTags(input.Tags)
		if len(tags) == 0 {
			delete(result.State, "tags")
		} else {
			result.State["tags"] = tagsToAny(tags)
		}
	}

	result.State["updated"] = s.now().Format(time.RFC3339)

	// Write back
	record := storage.EntityRecord{
		Type:   string(model.EntityKindPlan),
		ID:     input.ID,
		Slug:   input.Slug,
		Fields: result.State,
	}

	path, err := s.store.Write(record)
	if err != nil {
		return ListResult{}, fmt.Errorf("write plan: %w", err)
	}

	result.Path = path
	s.cacheUpsertFromResult(CreateResult{
		Type:  result.Type,
		ID:    result.ID,
		Slug:  result.Slug,
		Path:  result.Path,
		State: result.State,
	})

	return result, nil
}

// writePlan persists a new Plan entity.
func (s *EntityService) writePlan(entity model.Plan) (CreateResult, error) {
	fields := planFields(entity)
	record := storage.EntityRecord{
		Type:   string(model.EntityKindPlan),
		ID:     entity.ID,
		Slug:   entity.Slug,
		Fields: fields,
	}

	path, err := s.store.Write(record)
	if err != nil {
		return CreateResult{}, fmt.Errorf("write plan: %w", err)
	}

	return CreateResult{
		Type:  string(model.EntityKindPlan),
		ID:    entity.ID,
		Slug:  entity.Slug,
		Path:  path,
		State: fields,
	}, nil
}

// loadPlan reads a Plan from storage.
func (s *EntityService) loadPlan(id, slug string) (ListResult, error) {
	record, err := s.store.Load(string(model.EntityKindPlan), id, slug)
	if err != nil {
		return ListResult{}, fmt.Errorf("load plan %s: %w", id, err)
	}

	return ListResult{
		Type:  string(model.EntityKindPlan),
		ID:    id,
		Slug:  slug,
		Path:  filepath.Join(s.root, "plans", id+".yaml"),
		State: record.Fields,
	}, nil
}

// listPlanIDs returns all existing Plan IDs.
func (s *EntityService) listPlanIDs() ([]string, error) {
	dir := filepath.Join(s.root, "plans")
	entries, err := listDirectory(dir)
	if err != nil {
		return nil, nil // Directory doesn't exist yet
	}

	var ids []string
	for _, entry := range entries {
		if !strings.HasSuffix(entry, ".yaml") {
			continue
		}
		id, _, err := parsePlanFileName(entry)
		if err != nil {
			continue
		}
		ids = append(ids, id)
	}

	return ids, nil
}

// countPostDesigningFeaturesForPlan returns the number of features belonging to planID
// that are in a post-designing state (specifying, dev-planning, developing, reviewing, done).
// It short-circuits on the first qualifying feature for O(1) happy-path performance, but
// returns the full count for the override record message.
func (s *EntityService) countPostDesigningFeaturesForPlan(planID string) (int, error) {
	postDesigning := map[string]struct{}{
		string(model.FeatureStatusSpecifying):  {},
		string(model.FeatureStatusDevPlanning): {},
		string(model.FeatureStatusDeveloping):  {},
		string(model.FeatureStatusReviewing):   {},
		string(model.FeatureStatusDone):        {},
	}

	features, err := s.List("feature")
	if err != nil {
		return 0, err
	}

	count := 0
	for _, f := range features {
		if stringFromState(f.State, "parent") != planID {
			continue
		}
		status := stringFromState(f.State, "status")
		if _, ok := postDesigning[status]; ok {
			count++
		}
	}
	return count, nil
}

// planOverridesFromState extracts override records stored in a plan's state map.
func planOverridesFromState(state map[string]any) []model.OverrideRecord {
	rawSlice, ok := state["overrides"].([]any)
	if !ok || len(rawSlice) == 0 {
		return nil
	}
	result := make([]model.OverrideRecord, 0, len(rawSlice))
	for _, item := range rawSlice {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		fromStatus, _ := m["from_status"].(string)
		toStatus, _ := m["to_status"].(string)
		reason, _ := m["reason"].(string)
		tsStr, _ := m["timestamp"].(string)
		ts, _ := time.Parse(time.RFC3339, tsStr)
		result = append(result, model.OverrideRecord{
			FromStatus: fromStatus,
			ToStatus:   toStatus,
			Reason:     reason,
			Timestamp:  ts,
		})
	}
	return result
}

// overrideRecordsToAny converts a slice of OverrideRecord to []any for YAML storage,
// using the same wire format as feature override records.
func overrideRecordsToAny(records []model.OverrideRecord) []any {
	out := make([]any, len(records))
	for i, r := range records {
		out[i] = map[string]any{
			"from_status": r.FromStatus,
			"to_status":   r.ToStatus,
			"reason":      r.Reason,
			"timestamp":   r.Timestamp.Format(time.RFC3339),
		}
	}
	return out
}

// planFields converts a Plan entity to a map of fields for storage.
func planFields(p model.Plan) map[string]any {
	fields := map[string]any{
		"id":               p.ID,
		"slug":             p.Slug,
		"name":             p.Name,
		"status":           string(p.Status),
		"summary":          p.Summary,
		"created":          p.Created.Format(time.RFC3339),
		"created_by":       p.CreatedBy,
		"updated":          p.Updated.Format(time.RFC3339),
		"next_feature_seq": p.NextFeatureSeq,
	}

	if p.Design != "" {
		fields["design"] = p.Design
	}
	if len(p.Tags) > 0 {
		fields["tags"] = tagsToAny(p.Tags)
	}
	if p.Supersedes != "" {
		fields["supersedes"] = p.Supersedes
	}
	if p.SupersededBy != "" {
		fields["superseded_by"] = p.SupersededBy
	}

	return fields
}

// matchesPlanFilters checks if a Plan result matches the given filters.
func matchesPlanFilters(result ListResult, filters PlanFilters) bool {
	if filters.Status != "" {
		status := stringFromState(result.State, "status")
		if status != filters.Status {
			return false
		}
	}

	if filters.Prefix != "" {
		prefix, _, _ := model.ParsePlanID(result.ID)
		if prefix != filters.Prefix {
			return false
		}
	}

	if len(filters.Tags) > 0 {
		resultTags := tagsFromState(result.State)
		for _, filterTag := range filters.Tags {
			found := false
			for _, resultTag := range resultTags {
				if resultTag == filterTag {
					found = true
					break
				}
			}
			if !found {
				return false
			}
		}
	}

	return true
}

// parsePlanFileName extracts ID and slug from a Plan filename.
// Plan filenames have format: {prefix}{number}-{slug}.yaml
func parsePlanFileName(filename string) (id, slug string, err error) {
	name := strings.TrimSuffix(filename, ".yaml")
	if name == filename {
		return "", "", fmt.Errorf("not a yaml file: %s", filename)
	}

	// For Plan files, the ID is the entire filename minus extension
	// But we need to validate it's a valid Plan ID format
	if !model.IsPlanID(name) {
		return "", "", fmt.Errorf("not a valid plan ID: %s", name)
	}

	_, _, slug = model.ParsePlanID(name)
	return name, slug, nil
}

// normalizeTags normalizes a slice of tags to lowercase and removes duplicates.
func normalizeTags(tags []string) []string {
	if len(tags) == 0 {
		return nil
	}

	seen := make(map[string]bool)
	var result []string

	for _, tag := range tags {
		t := strings.ToLower(strings.TrimSpace(tag))
		if t == "" {
			continue
		}
		if seen[t] {
			continue
		}
		seen[t] = true
		result = append(result, t)
	}

	return result
}

// tagsToAny converts a string slice to an any slice for YAML storage.
func tagsToAny(tags []string) []any {
	result := make([]any, len(tags))
	for i, t := range tags {
		result[i] = t
	}
	return result
}

// tagsFromState extracts tags from an entity state map.
func tagsFromState(state map[string]any) []string {
	v, ok := state["tags"]
	if !ok {
		return nil
	}

	switch typed := v.(type) {
	case []any:
		result := make([]string, 0, len(typed))
		for _, item := range typed {
			if s, ok := item.(string); ok {
				result = append(result, s)
			}
		}
		return result
	case []string:
		return typed
	default:
		return nil
	}
}
