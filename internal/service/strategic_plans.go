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

// CreateStrategicPlanInput contains the fields needed to create a new strategic Plan.
type CreateStrategicPlanInput struct {
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
	// Parent is the optional ID of a parent plan for nesting.
	Parent string
	// DependsOn is an optional list of plan IDs this plan depends on.
	DependsOn []string
	// Order is an optional sibling ordering within a parent plan.
	Order int
}

// UpdateStrategicPlanInput contains the fields that can be updated on a strategic Plan.
type UpdateStrategicPlanInput struct {
	ID        string
	Slug      string
	Name      *string
	Summary   *string
	Design    *string
	Parent    *string
	DependsOn []string // nil = no change; empty slice = clear
	Order     *int
	Tags      []string // nil = no change; empty slice = clear
}

// StrategicPlanFilters contains optional filters for listing strategic Plans.
type StrategicPlanFilters struct {
	Status string
	Prefix string
	Parent string // empty means top-level only; "*" means all
	Tags   []string
}

// entityTypeStrategicPlan is the storage type string for strategic plans.
const entityTypeStrategicPlan = "strategic-plan"

// CreateStrategicPlan creates a new strategic Plan entity.
func (s *EntityService) CreateStrategicPlan(input CreateStrategicPlanInput) (CreateResult, error) {
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

	// Validate parent if set.
	parent := strings.TrimSpace(input.Parent)
	if parent != "" {
		if err := s.validateStrategicPlanRef(parent); err != nil {
			return CreateResult{}, fmt.Errorf("parent plan %s: %w", parent, err)
		}
	}

	// Load and validate prefix registry.
	cfg := config.LoadOrDefault()
	prefix := strings.TrimSpace(input.Prefix)
	if !cfg.IsActivePrefix(prefix) {
		if cfg.IsValidPrefix(prefix) {
			return CreateResult{}, fmt.Errorf("prefix %q is retired and cannot be used for new Plans", prefix)
		}
		return CreateResult{}, fmt.Errorf("undeclared prefix %q: add it to .kbz/config.yaml prefixes", prefix)
	}

	// Get next available number for this prefix.
	nextNum, err := cfg.NextPlanNumber(prefix, func() ([]string, error) {
		return s.listAllPlanIDs()
	})
	if err != nil {
		return CreateResult{}, fmt.Errorf("allocate plan number: %w", err)
	}

	slug := normalizeSlug(input.Slug)
	idValue := fmt.Sprintf("%s%d-%s", prefix, nextNum, slug)

	now := s.now()
	entity := model.StrategicPlan{
		ID:        idValue,
		Slug:      slug,
		Name:      planName,
		Status:    model.PlanningStatusIdea,
		Summary:   strings.TrimSpace(input.Summary),
		Parent:    parent,
		DependsOn: append([]string(nil), input.DependsOn...),
		Order:     input.Order,
		Tags:      normalizeTags(input.Tags),
		Created:   now,
		CreatedBy: strings.TrimSpace(input.CreatedBy),
		Updated:   now,
	}

	if err := validate.ValidateInitialState(validate.EntityStrategicPlan, string(entity.Status)); err != nil {
		return CreateResult{}, err
	}

	result, err := s.writeStrategicPlan(entity)
	if err != nil {
		return result, err
	}
	s.cacheUpsertFromResult(result)
	return result, nil
}

// GetStrategicPlan retrieves a strategic Plan by ID.
func (s *EntityService) GetStrategicPlan(id string) (ListResult, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return ListResult{}, fmt.Errorf("strategic plan ID is required")
	}

	if !model.IsPlanID(id) {
		return ListResult{}, fmt.Errorf("invalid Plan ID format: %s", id)
	}

	_, _, slug := model.ParsePlanID(id)
	return s.loadStrategicPlan(id, slug)
}

// ListStrategicPlans returns all strategic Plans, optionally filtered.
func (s *EntityService) ListStrategicPlans(filters StrategicPlanFilters) ([]ListResult, error) {
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

		// Load the record to check its entity type.
		record, err := s.store.Load(entityTypeStrategicPlan, id, slug)
		if err != nil {
			// Not a strategic plan (e.g., a batch plan) - skip.
			continue
		}

		// Verify it's a strategic plan.
		if strings.ToLower(strings.TrimSpace(record.Type)) != entityTypeStrategicPlan {
			continue
		}

		result := ListResult{
			Type:  entityTypeStrategicPlan,
			ID:    id,
			Slug:  slug,
			Path:  filepath.Join(s.root, "plans", id+".yaml"),
			State: record.Fields,
		}

		// Apply filters.
		if !matchesStrategicPlanFilters(result, filters) {
			continue
		}

		results = append(results, result)
	}

	return results, nil
}

// UpdateStrategicPlanStatus transitions a strategic Plan to a new status.
func (s *EntityService) UpdateStrategicPlanStatus(id, slug, newStatus string) (ListResult, error) {
	if err := validateRequired(
		field("id", id),
		field("slug", slug),
		field("status", newStatus),
	); err != nil {
		return ListResult{}, err
	}

	// Load existing plan.
	result, err := s.loadStrategicPlan(id, slug)
	if err != nil {
		return ListResult{}, err
	}

	currentStatus := stringFromState(result.State, "status")
	if err := validate.ValidateTransition(validate.EntityStrategicPlan, currentStatus, newStatus); err != nil {
		return ListResult{}, err
	}

	// Update status and updated timestamp.
	result.State["status"] = newStatus
	result.State["updated"] = s.now().Format(time.RFC3339)

	// Write back.
	record := storage.EntityRecord{
		Type:   entityTypeStrategicPlan,
		ID:     id,
		Slug:   slug,
		Fields: result.State,
	}

	path, err := s.store.Write(record)
	if err != nil {
		return ListResult{}, fmt.Errorf("write strategic plan: %w", err)
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

// UpdateStrategicPlan updates mutable fields on a strategic Plan.
// Validates parent existence and detects cycles when parent is changed.
func (s *EntityService) UpdateStrategicPlan(input UpdateStrategicPlanInput) (ListResult, error) {
	if err := validateRequired(
		field("id", input.ID),
		field("slug", input.Slug),
	); err != nil {
		return ListResult{}, err
	}

	// Load existing plan.
	result, err := s.loadStrategicPlan(input.ID, input.Slug)
	if err != nil {
		return ListResult{}, err
	}

	// Apply string field updates.
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

	// Update parent with validation.
	if input.Parent != nil {
		newParent := strings.TrimSpace(*input.Parent)
		if newParent != "" {
			// Validate parent exists.
			if err := s.validateStrategicPlanRef(newParent); err != nil {
				return ListResult{}, fmt.Errorf("parent plan %s: %w", newParent, err)
			}
			// Detect cycles: ensure new parent is not a descendant of this plan.
			if err := s.detectStrategicPlanCycle(input.ID, newParent); err != nil {
				return ListResult{}, err
			}
		}
		if newParent == "" {
			delete(result.State, "parent")
		} else {
			result.State["parent"] = newParent
		}
	}

	// Update order.
	if input.Order != nil {
		result.State["order"] = *input.Order
	}

	// Update depends_on.
	if input.DependsOn != nil {
		deps := cleanStringSlice(input.DependsOn)
		if len(deps) == 0 {
			delete(result.State, "depends_on")
		} else {
			result.State["depends_on"] = deps
		}
	}

	// Update tags.
	if input.Tags != nil {
		tags := normalizeTags(input.Tags)
		if len(tags) == 0 {
			delete(result.State, "tags")
		} else {
			result.State["tags"] = tagsToAny(tags)
		}
	}

	result.State["updated"] = s.now().Format(time.RFC3339)

	// Write back.
	record := storage.EntityRecord{
		Type:   entityTypeStrategicPlan,
		ID:     input.ID,
		Slug:   input.Slug,
		Fields: result.State,
	}

	path, err := s.store.Write(record)
	if err != nil {
		return ListResult{}, fmt.Errorf("write strategic plan: %w", err)
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

// ─── private helpers ──────────────────────────────────────────────────────────

// validateStrategicPlanRef checks that a plan ID references an existing strategic plan.
func (s *EntityService) validateStrategicPlanRef(id string) error {
	_, err := s.GetStrategicPlan(id)
	if err != nil {
		return ErrReferenceNotFound
	}
	return nil
}

// detectStrategicPlanCycle checks if setting planID's parent to newParentID would
// create a cycle. Walks the ancestor chain from newParentID upward.
func (s *EntityService) detectStrategicPlanCycle(planID, newParentID string) error {
	if planID == newParentID {
		return fmt.Errorf("cycle detected: plan %q cannot be its own parent", planID)
	}

	current := newParentID
	visited := map[string]bool{planID: true}
	for current != "" {
		if visited[current] {
			return fmt.Errorf("cycle detected: plan %q is already an ancestor of %q", current, planID)
		}
		visited[current] = true

		// Load the ancestor to find its parent.
		ancestor, err := s.loadStrategicPlan(current, "")
		if err != nil {
			// If it doesn't exist or isn't a strategic plan, stop walking.
			break
		}
		current = stringFromState(ancestor.State, "parent")
	}

	return nil
}

// writeStrategicPlan persists a new strategic Plan entity.
func (s *EntityService) writeStrategicPlan(entity model.StrategicPlan) (CreateResult, error) {
	fields := strategicPlanFields(entity)
	record := storage.EntityRecord{
		Type:   entityTypeStrategicPlan,
		ID:     entity.ID,
		Slug:   entity.Slug,
		Fields: fields,
	}

	path, err := s.store.Write(record)
	if err != nil {
		return CreateResult{}, fmt.Errorf("write strategic plan: %w", err)
	}

	return CreateResult{
		Type:  entityTypeStrategicPlan,
		ID:    entity.ID,
		Slug:  entity.Slug,
		Path:  path,
		State: fields,
	}, nil
}

// loadStrategicPlan reads a strategic Plan from storage.
func (s *EntityService) loadStrategicPlan(id, slug string) (ListResult, error) {
	if slug == "" {
		_, _, slug = model.ParsePlanID(id)
	}

	record, err := s.store.Load(entityTypeStrategicPlan, id, slug)
	if err != nil {
		return ListResult{}, fmt.Errorf("load strategic plan %s: %w", id, err)
	}

	return ListResult{
		Type:  entityTypeStrategicPlan,
		ID:    id,
		Slug:  slug,
		Path:  filepath.Join(s.root, "plans", id+".yaml"),
		State: record.Fields,
	}, nil
}

// strategicPlanFields converts a StrategicPlan entity to a map of fields for storage.
func strategicPlanFields(p model.StrategicPlan) map[string]any {
	fields := map[string]any{
		"id":         p.ID,
		"slug":       p.Slug,
		"name":       p.Name,
		"status":     string(p.Status),
		"summary":    p.Summary,
		"order":      p.Order,
		"created":    p.Created.Format(time.RFC3339),
		"created_by": p.CreatedBy,
		"updated":    p.Updated.Format(time.RFC3339),
	}

	if p.Parent != "" {
		fields["parent"] = p.Parent
	}
	if p.Design != "" {
		fields["design"] = p.Design
	}
	if len(p.DependsOn) > 0 {
		fields["depends_on"] = p.DependsOn
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

// matchesStrategicPlanFilters checks if a strategic Plan result matches the given filters.
func matchesStrategicPlanFilters(result ListResult, filters StrategicPlanFilters) bool {
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

	if filters.Parent != "" {
		if filters.Parent == "*" {
			// "*" means all (no parent filter).
		} else {
			parent := stringFromState(result.State, "parent")
			if parent != filters.Parent {
				return false
			}
		}
	} else {
		// Default: only return top-level plans (no parent).
		parent := stringFromState(result.State, "parent")
		if parent != "" {
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

// cleanStringSlice removes empty strings from a slice and returns it.
func cleanStringSlice(s []string) []string {
	if len(s) == 0 {
		return nil
	}
	var out []string
	for _, v := range s {
		if strings.TrimSpace(v) != "" {
			out = append(out, strings.TrimSpace(v))
		}
	}
	return out
}
