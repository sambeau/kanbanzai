package service

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sambeau/kanbanzai/internal/model"
	"github.com/sambeau/kanbanzai/internal/storage"
	"github.com/sambeau/kanbanzai/internal/validate"
)

type CreateBatchInput struct {
	Prefix    string
	Slug      string
	Name      string
	Summary   string
	Parent    string
	CreatedBy string
	Tags      []string
}

type CreatePlanInput = CreateBatchInput

type UpdateBatchInput struct {
	ID      string
	Slug    string
	Name    *string
	Summary *string
	Design  *string
	Parent  *string
	Tags    []string
}

type UpdatePlanInput = UpdateBatchInput

func (s *EntityService) CreateBatch(input CreateBatchInput) (CreateResult, error) {
	if err := validateRequired(
		field("prefix", input.Prefix),
		field("slug", input.Slug),
		field("name", input.Name),
		field("summary", input.Summary),
		field("created_by", input.CreatedBy),
	); err != nil {
		return CreateResult{}, err
	}

	batchName, nameErr := validate.ValidateName(input.Name)
	if nameErr != nil {
		return CreateResult{}, nameErr
	}

	cfg := s.cfg
	prefix := strings.TrimSpace(input.Prefix)
	if !cfg.IsActivePrefix(prefix) {
		if cfg.IsValidPrefix(prefix) {
			return CreateResult{}, fmt.Errorf("prefix %q is retired and cannot be used for new Batches", prefix)
		}
		return CreateResult{}, fmt.Errorf("undeclared prefix %q: add it to .kbz/config.yaml prefixes", prefix)
	}

	slug := normalizeSlug(input.Slug)

	var idValue string
	entityType := "batch_" + prefix
	if s.coordinationDB != nil {
		allocatedID, allocErr := s.coordinationDB.AllocateID(context.Background(), s.cfg.Coordination.ProjectID, entityType, prefix, slug)
		if allocErr != nil {
			fmt.Fprintf(os.Stderr, "warning: coordination database error, falling back to local allocation: %v\n", allocErr)
			// fall through to local allocation
		} else {
			idValue = allocatedID
		}
	}
	if idValue == "" {
		nextNum, err := cfg.NextPlanNumber(prefix, func() ([]string, error) {
			return s.listAllPlanIDs()
		})
		if err != nil {
			return CreateResult{}, fmt.Errorf("allocate batch number: %w", err)
		}
		idValue = fmt.Sprintf("%s%d-%s", prefix, nextNum, slug)
	}
	now := s.now()

	entity := model.Batch{
		ID:             idValue,
		Slug:           slug,
		Name:           batchName,
		Status:         model.BatchStatusProposed,
		Summary:        strings.TrimSpace(input.Summary),
		Parent:         strings.TrimSpace(input.Parent),
		Tags:           normalizeTags(input.Tags),
		Created:        now,
		CreatedBy:      strings.TrimSpace(input.CreatedBy),
		Updated:        now,
		NextFeatureSeq: 1,
	}

	if err := validate.ValidateInitialState(validate.EntityBatch, string(entity.Status)); err != nil {
		return CreateResult{}, err
	}

	result, err := s.writeBatch(entity)
	if err != nil {
		return result, err
	}
	s.cacheUpsertFromResult(result)
	return result, nil
}

func (s *EntityService) CreatePlan(input CreatePlanInput) (CreateResult, error) {
	return s.CreateBatch(CreateBatchInput(input))
}

func (s *EntityService) AllocateFeatureDisplayIDInBatch(batchID string) (string, error) {
	batchResult, err := s.GetBatch(batchID)
	if err != nil {
		return "", fmt.Errorf("load batch %s: %w", batchID, err)
	}

	seq := intFromState(batchResult.State, "next_feature_seq", 1)
	batchPrefix, batchNum, _ := model.ParseBatchID(batchID)
	displayID := fmt.Sprintf("%s%s-F%d", batchPrefix, batchNum, seq)

	batchResult.State["next_feature_seq"] = seq + 1
	batchRecord := storage.EntityRecord{
		Type:   string(model.EntityKindBatch),
		ID:     batchResult.ID,
		Slug:   batchResult.Slug,
		Fields: batchResult.State,
	}
	if _, err := s.store.Write(batchRecord); err != nil {
		return "", fmt.Errorf("increment batch sequence for %s: %w", batchID, err)
	}
	return displayID, nil
}

func (s *EntityService) AllocateFeatureDisplayIDInPlan(planID string) (string, error) {
	return s.AllocateFeatureDisplayIDInBatch(planID)
}

func (s *EntityService) GetBatch(id string) (ListResult, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return ListResult{}, fmt.Errorf("batch ID is required")
	}
	if !model.IsBatchID(id) {
		return ListResult{}, fmt.Errorf("invalid Batch ID format: %s", id)
	}
	_, _, slug := model.ParseBatchID(id)
	return s.loadBatch(id, slug)
}

func (s *EntityService) GetPlan(id string) (ListResult, error) {
	return s.GetBatch(id)
}

func (s *EntityService) ListBatches(filters BatchFilters) ([]ListResult, error) {
	dir := filepath.Join(s.root, "batches")
	entries, err := listDirectory(dir)
	if err != nil {
		dir = filepath.Join(s.root, "plans")
		entries, err = listDirectory(dir)
		if err != nil {
			return nil, err
		}
	}

	var results []ListResult
	for _, entry := range entries {
		if !strings.HasSuffix(entry, ".yaml") {
			continue
		}
		id, slug, err := parseBatchFileName(entry)
		if err != nil {
			continue
		}
		result, err := s.loadBatch(id, slug)
		if err != nil {
			continue
		}
		if !matchesBatchFilters(result, filters) {
			continue
		}
		results = append(results, result)
	}

	return results, nil
}

func (s *EntityService) ListPlans(filters PlanFilters) ([]ListResult, error) {
	return s.ListBatches(BatchFilters{
		Status: filters.Status,
		Prefix: filters.Prefix,
		Parent: filters.Parent,
		Tags:   filters.Tags,
	})
}

type BatchFilters struct {
	Status string
	Prefix string
	Parent string // filter by parent plan ID; empty means no filter
	Tags   []string
}

type PlanFilters = BatchFilters

func (s *EntityService) UpdateBatchStatus(id, slug, newStatus string) (ListResult, error) {
	if err := validateRequired(
		field("id", id),
		field("slug", slug),
		field("status", newStatus),
	); err != nil {
		return ListResult{}, err
	}

	result, err := s.loadBatch(id, slug)
	if err != nil {
		return ListResult{}, err
	}

	currentStatus := stringFromState(result.State, "status")
	if err := validate.ValidateTransition(validate.EntityBatch, currentStatus, newStatus); err != nil {
		return ListResult{}, err
	}

	if currentStatus == string(model.BatchStatusProposed) && newStatus == string(model.BatchStatusActive) {
		n, err := s.countPostDesigningFeaturesForBatch(id)
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
		existing := batchOverridesFromState(result.State)
		or := model.OverrideRecord{
			FromStatus: currentStatus,
			ToStatus:   newStatus,
			Reason:     fmt.Sprintf("proposed → active shortcut: %d feature(s) in post-designing state at transition time", n),
			Timestamp:  s.now(),
		}
		result.State["overrides"] = overrideRecordsToAny(append(existing, or))
	}

	result.State["status"] = newStatus
	result.State["updated"] = s.now().Format(time.RFC3339)

	record := storage.EntityRecord{
		Type:   string(model.EntityKindBatch),
		ID:     id,
		Slug:   slug,
		Fields: result.State,
	}
	path, err := s.store.Write(record)
	if err != nil {
		return ListResult{}, fmt.Errorf("write batch: %w", err)
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

func (s *EntityService) UpdatePlanStatus(id, slug, newStatus string) (ListResult, error) {
	return s.UpdateBatchStatus(id, slug, newStatus)
}

func (s *EntityService) UpdateBatch(input UpdateBatchInput) (ListResult, error) {
	if err := validateRequired(
		field("id", input.ID),
		field("slug", input.Slug),
	); err != nil {
		return ListResult{}, err
	}

	result, err := s.loadBatch(input.ID, input.Slug)
	if err != nil {
		return ListResult{}, err
	}

	if input.Name != nil {
		result.State["name"] = strings.TrimSpace(*input.Name)
	}
	if input.Summary != nil {
		result.State["summary"] = strings.TrimSpace(*input.Summary)
	}
	if input.Parent != nil {
		if *input.Parent == "" {
			delete(result.State, "parent")
		} else {
			result.State["parent"] = strings.TrimSpace(*input.Parent)
		}
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

	record := storage.EntityRecord{
		Type:   string(model.EntityKindBatch),
		ID:     input.ID,
		Slug:   input.Slug,
		Fields: result.State,
	}
	path, err := s.store.Write(record)
	if err != nil {
		return ListResult{}, fmt.Errorf("write batch: %w", err)
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

func (s *EntityService) UpdatePlan(input UpdatePlanInput) (ListResult, error) {
	return s.UpdateBatch(UpdateBatchInput(input))
}

func (s *EntityService) writeBatch(entity model.Batch) (CreateResult, error) {
	fields := batchFields(entity)
	record := storage.EntityRecord{
		Type:   string(model.EntityKindBatch),
		ID:     entity.ID,
		Slug:   entity.Slug,
		Fields: fields,
	}
	path, err := s.store.Write(record)
	if err != nil {
		return CreateResult{}, fmt.Errorf("write batch: %w", err)
	}
	return CreateResult{
		Type:  string(model.EntityKindBatch),
		ID:    entity.ID,
		Slug:  entity.Slug,
		Path:  path,
		State: fields,
	}, nil
}

func (s *EntityService) loadBatch(id, slug string) (ListResult, error) {
	record, err := s.store.Load(string(model.EntityKindBatch), id, slug)
	entityType := string(model.EntityKindBatch)
	fallbackDir := "batches"
	if err != nil {
		log.Printf("INFO: batch %s not found in batches/ directory, falling back to plans/ (deprecated legacy path)", id)
		record, err = s.store.Load(string(model.EntityKindStrategicPlan), id, slug)
		fallbackDir = "plans"
		if err != nil {
			return ListResult{}, fmt.Errorf("load batch %s: %w", id, err)
		}
		// Plans loaded from the legacy plans/ directory may be StrategicPlan
		// entities with planning-only statuses (idea, shaping, ready, active).
		// Detect these and return the correct entity type so downstream
		// validation uses the StrategicPlan lifecycle machine.
		if isStrategicPlanStatus(stringFromState(record.Fields, "status")) {
			entityType = string(model.EntityKindStrategicPlan)
		}
	}

	return ListResult{
		Type:  entityType,
		ID:    id,
		Slug:  slug,
		Path:  filepath.Join(s.root, fallbackDir, id+".yaml"),
		State: record.Fields,
	}, nil
}

// isStrategicPlanStatus returns true if status is exclusive to the StrategicPlan
// lifecycle (not valid in the Batch lifecycle). Shared terminal statuses
// (done, superseded, cancelled) are not strategic-plan–only.
func isStrategicPlanStatus(status string) bool {
	switch status {
	case string(model.PlanningStatusIdea),
		string(model.PlanningStatusShaping),
		string(model.PlanningStatusReady),
		string(model.PlanningStatusActive):
		return true
	}
	return false
}

func (s *EntityService) listAllPlanIDs() ([]string, error) {
	var ids []string

	batchDir := filepath.Join(s.root, "batches")
	batchEntries, err := listDirectory(batchDir)
	if err == nil {
		for _, entry := range batchEntries {
			if !strings.HasSuffix(entry, ".yaml") {
				continue
			}
			name := strings.TrimSuffix(entry, ".yaml")
			if model.IsBatchID(name) {
				ids = append(ids, name)
			}
		}
	}

	planDir := filepath.Join(s.root, "plans")
	planEntries, err := listDirectory(planDir)
	if err == nil {
		for _, entry := range planEntries {
			if !strings.HasSuffix(entry, ".yaml") {
				continue
			}
			name := strings.TrimSuffix(entry, ".yaml")
			if model.IsBatchID(name) && !stringSliceContains(ids, name) {
				ids = append(ids, name)
			}
		}
	}

	return ids, nil
}

func (s *EntityService) listPlanIDs() ([]string, error) {
	return s.listAllPlanIDs()
}

func (s *EntityService) listBatchIDs() ([]string, error) {
	return s.listAllPlanIDs()
}

func (s *EntityService) countPostDesigningFeaturesForBatch(batchID string) (int, error) {
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
		if stringFromState(f.State, "parent") != batchID {
			continue
		}
		status := stringFromState(f.State, "status")
		if _, ok := postDesigning[status]; ok {
			count++
		}
	}
	return count, nil
}

func batchOverridesFromState(state map[string]any) []model.OverrideRecord {
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

func batchFields(b model.Batch) map[string]any {
	fields := map[string]any{
		"id":               b.ID,
		"slug":             b.Slug,
		"name":             b.Name,
		"status":           string(b.Status),
		"summary":          b.Summary,
		"created":          b.Created.Format(time.RFC3339),
		"created_by":       b.CreatedBy,
		"updated":          b.Updated.Format(time.RFC3339),
		"next_feature_seq": b.NextFeatureSeq,
	}
	if b.Parent != "" {
		fields["parent"] = b.Parent
	}
	if b.Design != "" {
		fields["design"] = b.Design
	}
	if len(b.Tags) > 0 {
		fields["tags"] = tagsToAny(b.Tags)
	}
	if b.Supersedes != "" {
		fields["supersedes"] = b.Supersedes
	}
	if b.SupersededBy != "" {
		fields["superseded_by"] = b.SupersededBy
	}
	return fields
}

func matchesBatchFilters(result ListResult, filters BatchFilters) bool {
	if filters.Status != "" {
		status := stringFromState(result.State, "status")
		if status != filters.Status {
			return false
		}
	}
	if filters.Prefix != "" {
		prefix, _, _ := model.ParseBatchID(result.ID)
		if prefix != filters.Prefix {
			return false
		}
	}
	if filters.Parent != "" {
		parent := stringFromState(result.State, "parent")
		if parent != filters.Parent {
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

// Deprecated: use matchesBatchFilters.
func matchesPlanFilters(result ListResult, filters PlanFilters) bool {
	return matchesBatchFilters(result, BatchFilters(filters))
}

func parseBatchFileName(filename string) (id, slug string, err error) {
	name := strings.TrimSuffix(filename, ".yaml")
	if name == filename {
		return "", "", fmt.Errorf("not a yaml file: %s", filename)
	}
	if !model.IsBatchID(name) {
		return "", "", fmt.Errorf("not a valid batch ID: %s", name)
	}
	_, _, slug = model.ParseBatchID(name)
	return name, slug, nil
}

func parsePlanFileName(filename string) (id, slug string, err error) {
	return parseBatchFileName(filename)
}

func stringSliceContains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// ─── Shared helpers (used by multiple service files) ─────────────────────────

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

func tagsToAny(tags []string) []any {
	result := make([]any, len(tags))
	for i, t := range tags {
		result[i] = t
	}
	return result
}

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
