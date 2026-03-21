package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"kanbanzai/internal/cache"
	"kanbanzai/internal/core"
	"kanbanzai/internal/id"
	"kanbanzai/internal/model"
	"kanbanzai/internal/storage"
	"kanbanzai/internal/validate"
)

type CreateEpicInput struct {
	EpicSlug  string // human-chosen slug for the EPIC-{SLUG} ID; derived from Slug if empty
	Slug      string
	Title     string
	Summary   string
	CreatedBy string
}

type CreateFeatureInput struct {
	Slug      string
	Epic      string
	Summary   string
	CreatedBy string
}

type CreateTaskInput struct {
	ParentFeature string
	Slug          string
	Summary       string
}

type CreateBugInput struct {
	Slug       string
	Title      string
	ReportedBy string
	Observed   string
	Expected   string
	Severity   string
	Priority   string
	Type       string
}

type CreateDecisionInput struct {
	Slug      string
	Summary   string
	Rationale string
	DecidedBy string
}

type UpdateStatusInput struct {
	Type   string
	ID     string
	Slug   string
	Status string
}

type UpdateEntityInput struct {
	Type   string            // entity type: "epic", "feature", "task", "bug", "decision"
	ID     string            // entity ID
	Slug   string            // entity slug
	Fields map[string]string // field name → new value (string values only)
}

type CreateResult struct {
	Type  string
	ID    string
	Slug  string
	Path  string
	State map[string]any
}

type GetResult = CreateResult

type ListResult struct {
	Type  string
	ID    string
	Slug  string
	Path  string
	State map[string]any
}

type EntityService struct {
	root      string
	store     *storage.EntityStore
	allocator *id.Allocator
	now       func() time.Time
	cache     *cache.Cache
}

func NewEntityService(root string) *EntityService {
	if strings.TrimSpace(root) == "" {
		root = core.StatePath()
	}

	return &EntityService{
		root:      root,
		store:     storage.NewEntityStore(root),
		allocator: id.NewAllocator(),
		now: func() time.Time {
			return time.Now().UTC()
		},
	}
}

// SetCache attaches an optional local derived cache.
// When set, mutations update the cache best-effort, and lookups
// may use the cache for acceleration. All operations fall back
// to filesystem if the cache is nil.
func (s *EntityService) SetCache(c *cache.Cache) {
	s.cache = c
}

// RebuildCache scans all canonical entity files and repopulates the cache.
// Returns the number of entities cached.
func (s *EntityService) RebuildCache() (int, error) {
	if s.cache == nil {
		return 0, fmt.Errorf("no cache configured")
	}

	var records []cache.RebuildRecord
	for _, kind := range []string{
		string(model.EntityKindEpic),
		string(model.EntityKindFeature),
		string(model.EntityKindTask),
		string(model.EntityKindBug),
		string(model.EntityKindDecision),
	} {
		results, err := s.List(kind)
		if err != nil {
			return 0, fmt.Errorf("listing %s entities for cache rebuild: %w", kind, err)
		}
		for _, r := range results {
			records = append(records, cache.RebuildRecord{
				EntityType: r.Type,
				ID:         r.ID,
				Slug:       r.Slug,
				FilePath:   r.Path,
				Fields:     r.State,
			})
		}
	}

	return s.cache.Rebuild(records)
}

func (s *EntityService) CreateEpic(input CreateEpicInput) (CreateResult, error) {
	if err := validateRequired(
		field("slug", input.Slug),
		field("title", input.Title),
		field("summary", input.Summary),
		field("created_by", input.CreatedBy),
	); err != nil {
		return CreateResult{}, err
	}

	epicSlug := strings.TrimSpace(input.EpicSlug)
	if epicSlug == "" {
		epicSlug = strings.ToUpper(strings.ReplaceAll(normalizeSlug(input.Slug), " ", "-"))
	}

	exists := func(candidateID string) bool {
		return s.entityExists(string(model.EntityKindEpic), candidateID)
	}
	idValue, err := s.allocator.Allocate(model.EntityKindEpic, epicSlug, exists)
	if err != nil {
		return CreateResult{}, err
	}

	entity := model.Epic{
		ID:        idValue,
		Slug:      normalizeSlug(input.Slug),
		Title:     strings.TrimSpace(input.Title),
		Status:    model.EpicStatus("proposed"),
		Summary:   strings.TrimSpace(input.Summary),
		Created:   s.now(),
		CreatedBy: strings.TrimSpace(input.CreatedBy),
	}

	if err := validate.ValidateInitialState(validate.EntityEpic, string(entity.Status)); err != nil {
		return CreateResult{}, err
	}

	result, err := s.write(entity)
	if err != nil {
		return result, err
	}
	s.cacheUpsertFromResult(result)
	return result, nil
}

func (s *EntityService) CreateFeature(input CreateFeatureInput) (CreateResult, error) {
	if err := validateRequired(
		field("slug", input.Slug),
		field("epic", input.Epic),
		field("summary", input.Summary),
		field("created_by", input.CreatedBy),
	); err != nil {
		return CreateResult{}, err
	}

	epicID := strings.TrimSpace(input.Epic)
	if !s.entityExists(string(model.EntityKindEpic), epicID) {
		return CreateResult{}, fmt.Errorf("epic %s: %w", epicID, ErrReferenceNotFound)
	}

	idValue, err := s.allocateID(model.EntityKindFeature)
	if err != nil {
		return CreateResult{}, err
	}

	entity := model.Feature{
		ID:        idValue,
		Slug:      normalizeSlug(input.Slug),
		Epic:      epicID,
		Status:    model.FeatureStatus("draft"),
		Summary:   strings.TrimSpace(input.Summary),
		Created:   s.now(),
		CreatedBy: strings.TrimSpace(input.CreatedBy),
	}

	if err := validate.ValidateInitialState(validate.EntityFeature, string(entity.Status)); err != nil {
		return CreateResult{}, err
	}

	result, err := s.write(entity)
	if err != nil {
		return result, err
	}
	s.cacheUpsertFromResult(result)
	return result, nil
}

func (s *EntityService) CreateTask(input CreateTaskInput) (CreateResult, error) {
	if err := validateRequired(
		field("parent_feature", input.ParentFeature),
		field("slug", input.Slug),
		field("summary", input.Summary),
	); err != nil {
		return CreateResult{}, err
	}

	featureID := strings.TrimSpace(input.ParentFeature)
	if !s.entityExists(string(model.EntityKindFeature), featureID) {
		return CreateResult{}, fmt.Errorf("feature %s: %w", featureID, ErrReferenceNotFound)
	}

	idValue, err := s.allocateID(model.EntityKindTask)
	if err != nil {
		return CreateResult{}, err
	}

	entity := model.Task{
		ID:            idValue,
		ParentFeature: featureID,
		Slug:          normalizeSlug(input.Slug),
		Summary:       strings.TrimSpace(input.Summary),
		Status:        model.TaskStatus("queued"),
	}

	if err := validate.ValidateInitialState(validate.EntityTask, string(entity.Status)); err != nil {
		return CreateResult{}, err
	}

	result, err := s.write(entity)
	if err != nil {
		return result, err
	}
	s.cacheUpsertFromResult(result)
	return result, nil
}

func (s *EntityService) CreateBug(input CreateBugInput) (CreateResult, error) {
	if err := validateRequired(
		field("slug", input.Slug),
		field("title", input.Title),
		field("reported_by", input.ReportedBy),
		field("observed", input.Observed),
		field("expected", input.Expected),
	); err != nil {
		return CreateResult{}, err
	}

	idValue, err := s.allocateID(model.EntityKindBug)
	if err != nil {
		return CreateResult{}, err
	}

	severity := defaultString(input.Severity, string(model.BugSeverityMedium))
	if err := validate.ValidateBugSeverity(severity); err != nil {
		return CreateResult{}, err
	}

	priority := defaultString(input.Priority, string(model.BugPriorityMedium))
	if err := validate.ValidateBugPriority(priority); err != nil {
		return CreateResult{}, err
	}

	bugType := defaultString(input.Type, string(model.BugTypeImplementationDefect))
	if err := validate.ValidateBugType(bugType); err != nil {
		return CreateResult{}, err
	}

	entity := model.Bug{
		ID:         idValue,
		Slug:       normalizeSlug(input.Slug),
		Title:      strings.TrimSpace(input.Title),
		Status:     model.BugStatus("reported"),
		Severity:   model.BugSeverity(severity),
		Priority:   model.BugPriority(priority),
		Type:       model.BugType(bugType),
		ReportedBy: strings.TrimSpace(input.ReportedBy),
		Reported:   s.now(),
		Observed:   strings.TrimSpace(input.Observed),
		Expected:   strings.TrimSpace(input.Expected),
	}

	if err := validate.ValidateInitialState(validate.EntityBug, string(entity.Status)); err != nil {
		return CreateResult{}, err
	}

	result, err := s.write(entity)
	if err != nil {
		return result, err
	}
	s.cacheUpsertFromResult(result)
	return result, nil
}

func (s *EntityService) CreateDecision(input CreateDecisionInput) (CreateResult, error) {
	if err := validateRequired(
		field("slug", input.Slug),
		field("summary", input.Summary),
		field("rationale", input.Rationale),
		field("decided_by", input.DecidedBy),
	); err != nil {
		return CreateResult{}, err
	}

	idValue, err := s.allocateID(model.EntityKindDecision)
	if err != nil {
		return CreateResult{}, err
	}

	entity := model.Decision{
		ID:        idValue,
		Slug:      normalizeSlug(input.Slug),
		Summary:   strings.TrimSpace(input.Summary),
		Rationale: strings.TrimSpace(input.Rationale),
		DecidedBy: strings.TrimSpace(input.DecidedBy),
		Date:      s.now(),
		Status:    model.DecisionStatus("proposed"),
	}

	if err := validate.ValidateInitialState(validate.EntityDecision, string(entity.Status)); err != nil {
		return CreateResult{}, err
	}

	result, err := s.write(entity)
	if err != nil {
		return result, err
	}
	s.cacheUpsertFromResult(result)
	return result, nil
}

// ValidateCandidate validates candidate entity data without persisting it.
// It returns a list of validation errors, or an empty slice if the data is valid.
func (s *EntityService) ValidateCandidate(entityType string, fields map[string]any) []validate.ValidationError {
	return validate.ValidateRecord(entityType, fields)
}

// HealthCheck runs a comprehensive health check across all entities in the store.
func (s *EntityService) HealthCheck() (*validate.HealthReport, error) {
	loadAll := func() ([]validate.EntityInfo, error) {
		var all []validate.EntityInfo
		for _, kind := range []string{
			string(model.EntityKindEpic),
			string(model.EntityKindFeature),
			string(model.EntityKindTask),
			string(model.EntityKindBug),
			string(model.EntityKindDecision),
		} {
			results, err := s.List(kind)
			if err != nil {
				return nil, fmt.Errorf("listing %s entities: %w", kind, err)
			}
			for _, r := range results {
				all = append(all, validate.EntityInfo{
					Type:   r.Type,
					ID:     r.ID,
					Fields: r.State,
				})
			}
		}
		return all, nil
	}

	entityExists := func(entityType, id string) bool {
		results, err := s.List(entityType)
		if err != nil {
			return false
		}
		for _, r := range results {
			if r.ID == id {
				return true
			}
		}
		return false
	}

	return validate.CheckHealth(loadAll, entityExists)
}

// ResolvePrefix resolves an ID prefix to the unique (id, slug) pair for the
// given entity type. It scans filenames without loading YAML.
func (s *EntityService) ResolvePrefix(entityType, prefix string) (resolvedID, resolvedSlug string, err error) {
	entityType = strings.ToLower(strings.TrimSpace(entityType))
	prefix = strings.ToUpper(strings.TrimSpace(prefix))
	prefix = id.StripBreakHyphens(prefix)

	dir := filepath.Join(s.root, entityDirectory(entityType))
	entries, err := filepath.Glob(filepath.Join(dir, "*.yaml"))
	if err != nil {
		return "", "", fmt.Errorf("resolve prefix for %s: %w", entityType, err)
	}

	type match struct {
		id   string
		slug string
	}
	var matches []match

	for _, entry := range entries {
		base := filepath.Base(entry)
		baseName := strings.TrimSuffix(base, ".yaml")

		fileID, fileSlug, err := parseRecordIdentity(entityType, baseName)
		if err != nil {
			continue
		}

		normalizedID := strings.ToUpper(fileID)
		if strings.HasPrefix(normalizedID, prefix) {
			matches = append(matches, match{id: fileID, slug: fileSlug})
		}
	}

	switch len(matches) {
	case 0:
		return "", "", fmt.Errorf("no %s entity found matching prefix %q", entityType, prefix)
	case 1:
		return matches[0].id, matches[0].slug, nil
	default:
		ids := make([]string, len(matches))
		for i, m := range matches {
			ids[i] = m.id
		}
		return "", "", fmt.Errorf("ambiguous prefix %q for %s: matches %s", prefix, entityType, strings.Join(ids, ", "))
	}
}

func (s *EntityService) Get(entityType, entityID, slug string) (GetResult, error) {
	entityType = strings.ToLower(strings.TrimSpace(entityType))
	entityID = strings.TrimSpace(entityID)
	slug = normalizeSlug(slug)

	if entityType == "" {
		return GetResult{}, fmt.Errorf("entity type is required")
	}
	if entityID == "" {
		return GetResult{}, fmt.Errorf("entity id is required")
	}
	if slug == "" {
		resolvedID, resolvedSlug, err := s.ResolvePrefix(entityType, entityID)
		if err != nil {
			return GetResult{}, err
		}
		entityID = resolvedID
		slug = resolvedSlug
	}

	record, err := s.store.Load(entityType, entityID, slug)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return GetResult{}, fmt.Errorf("get %s %s: %w", entityType, entityID, ErrNotFound)
		}
		return GetResult{}, err
	}

	return GetResult{
		Type:  record.Type,
		ID:    record.ID,
		Slug:  record.Slug,
		Path:  filepath.Join(s.root, entityDirectory(record.Type), entityFileName(record.ID, record.Slug)),
		State: record.Fields,
	}, nil
}

func (s *EntityService) List(entityType string) ([]ListResult, error) {
	entityType = strings.ToLower(strings.TrimSpace(entityType))
	if entityType == "" {
		return nil, fmt.Errorf("entity type is required")
	}

	dir := filepath.Join(s.root, entityDirectory(entityType))
	entries, err := filepath.Glob(filepath.Join(dir, "*.yaml"))
	if err != nil {
		return nil, fmt.Errorf("list %s entities: %w", entityType, err)
	}

	sort.Strings(entries)

	results := make([]ListResult, 0, len(entries))
	for _, entry := range entries {
		record, err := s.loadRecordFromPath(entityType, entry)
		if err != nil {
			return nil, err
		}

		results = append(results, ListResult{
			Type:  record.Type,
			ID:    record.ID,
			Slug:  record.Slug,
			Path:  entry,
			State: record.Fields,
		})
	}

	return results, nil
}

func (s *EntityService) UpdateStatus(input UpdateStatusInput) (GetResult, error) {
	entityType := strings.ToLower(strings.TrimSpace(input.Type))
	entityID := strings.TrimSpace(input.ID)
	slug := normalizeSlug(input.Slug)
	nextStatus := strings.TrimSpace(input.Status)

	if err := validateRequired(
		field("type", entityType),
		field("id", entityID),
		field("status", nextStatus),
	); err != nil {
		return GetResult{}, err
	}

	if slug == "" {
		resolvedID, resolvedSlug, err := s.ResolvePrefix(entityType, entityID)
		if err != nil {
			return GetResult{}, err
		}
		entityID = resolvedID
		slug = resolvedSlug
	}

	record, err := s.store.Load(entityType, entityID, slug)
	if err != nil {
		return GetResult{}, err
	}

	currentStatus, ok := record.Fields["status"]
	if !ok {
		return GetResult{}, fmt.Errorf("%s %s has no status field", entityType, entityID)
	}

	kind, err := validateKindForType(entityType)
	if err != nil {
		return GetResult{}, err
	}

	currentStatusText := strings.TrimSpace(fmt.Sprint(currentStatus))
	if err := validate.ValidateTransition(kind, currentStatusText, nextStatus); err != nil {
		return GetResult{}, fmt.Errorf("%s: %w", err.Error(), ErrInvalidTransition)
	}

	record.Fields["status"] = nextStatus
	path, err := s.store.Write(record)
	if err != nil {
		return GetResult{}, err
	}

	result := GetResult{
		Type:  record.Type,
		ID:    record.ID,
		Slug:  record.Slug,
		Path:  path,
		State: record.Fields,
	}
	s.cacheUpsertFromResult(result)
	return result, nil
}

// UpdateEntity updates fields of an existing entity for error correction.
// It cannot change the id (immutable) or status (use UpdateStatus instead).
func (s *EntityService) UpdateEntity(input UpdateEntityInput) (GetResult, error) {
	entityType := strings.ToLower(strings.TrimSpace(input.Type))
	entityID := strings.TrimSpace(input.ID)
	slug := normalizeSlug(input.Slug)

	if err := validateRequired(
		field("type", entityType),
		field("id", entityID),
	); err != nil {
		return GetResult{}, err
	}

	if slug == "" {
		resolvedID, resolvedSlug, err := s.ResolvePrefix(entityType, entityID)
		if err != nil {
			return GetResult{}, err
		}
		entityID = resolvedID
		slug = resolvedSlug
	}

	if _, ok := input.Fields["id"]; ok {
		return GetResult{}, fmt.Errorf("cannot update id: field is immutable")
	}
	if _, ok := input.Fields["status"]; ok {
		return GetResult{}, fmt.Errorf("cannot update status: use update_status instead")
	}

	record, err := s.store.Load(entityType, entityID, slug)
	if err != nil {
		return GetResult{}, err
	}

	oldSlug := record.Slug

	for k, v := range input.Fields {
		record.Fields[k] = v
	}

	if newSlug, ok := input.Fields["slug"]; ok {
		record.Slug = normalizeSlug(newSlug)
		record.Fields["slug"] = record.Slug
	}

	errs := validate.ValidateRecord(entityType, record.Fields)
	if len(errs) > 0 {
		msgs := make([]string, len(errs))
		for i, e := range errs {
			msgs[i] = fmt.Sprintf("%s: %s", e.Field, e.Message)
		}
		return GetResult{}, fmt.Errorf("validation failed: %s", strings.Join(msgs, "; "))
	}

	path, err := s.store.Write(record)
	if err != nil {
		return GetResult{}, err
	}

	if record.Slug != oldSlug {
		oldPath := filepath.Join(s.root, entityDirectory(entityType), entityFileName(entityID, oldSlug))
		_ = os.Remove(oldPath)
	}

	result := GetResult{
		Type:  record.Type,
		ID:    record.ID,
		Slug:  record.Slug,
		Path:  path,
		State: record.Fields,
	}
	s.cacheUpsertFromResult(result)
	return result, nil
}

func (s *EntityService) write(entity model.Entity) (CreateResult, error) {
	record, err := recordFromEntity(entity)
	if err != nil {
		return CreateResult{}, err
	}

	path, err := s.store.Write(record)
	if err != nil {
		return CreateResult{}, err
	}

	return CreateResult{
		Type:  record.Type,
		ID:    record.ID,
		Slug:  record.Slug,
		Path:  path,
		State: record.Fields,
	}, nil
}

func (s *EntityService) allocateID(entityKind model.EntityKind) (string, error) {
	exists := func(candidateID string) bool {
		return s.entityExists(string(entityKind), candidateID)
	}
	return s.allocator.Allocate(entityKind, "", exists)
}

// entityExists checks whether an entity with the given type and ID exists on disk.
func (s *EntityService) entityExists(entityType, entityID string) bool {
	dir := filepath.Join(s.root, entityDirectory(entityType))
	pattern := filepath.Join(dir, entityID+"-*.yaml")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return false
	}
	return len(matches) > 0
}

func (s *EntityService) loadRecordFromPath(entityType, path string) (storage.EntityRecord, error) {
	base := filepath.Base(path)
	idPart := strings.TrimSuffix(base, filepath.Ext(base))

	idValue, slug, err := parseRecordIdentity(entityType, idPart)
	if err != nil {
		return storage.EntityRecord{}, err
	}

	record, err := s.store.Load(entityType, idValue, slug)
	if err != nil {
		return storage.EntityRecord{}, err
	}

	return record, nil
}

func entityDirectory(entityType string) string {
	return strings.ToLower(strings.TrimSpace(entityType)) + "s"
}

func entityFileName(idValue, slug string) string {
	return fmt.Sprintf("%s-%s.yaml", idValue, slug)
}

type requiredField struct {
	name  string
	value string
}

func field(name, value string) requiredField {
	return requiredField{name: name, value: value}
}

func validateRequired(fields ...requiredField) error {
	for _, f := range fields {
		if strings.TrimSpace(f.value) == "" {
			return fmt.Errorf("%s is required", f.name)
		}
	}
	return nil
}

func normalizeSlug(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	value = strings.ReplaceAll(value, " ", "-")
	for strings.Contains(value, "--") {
		value = strings.ReplaceAll(value, "--", "-")
	}
	return strings.Trim(value, "-")
}

func defaultString(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}

func validateKindForType(entityType string) (validate.EntityKind, error) {
	switch entityType {
	case string(model.EntityKindEpic):
		return validate.EntityEpic, nil
	case string(model.EntityKindFeature):
		return validate.EntityFeature, nil
	case string(model.EntityKindTask):
		return validate.EntityTask, nil
	case string(model.EntityKindBug):
		return validate.EntityBug, nil
	case string(model.EntityKindDecision):
		return validate.EntityDecision, nil
	default:
		return "", fmt.Errorf("unknown entity type %q", entityType)
	}
}

func parseRecordIdentity(entityType, idPart string) (string, string, error) {
	switch entityType {
	case string(model.EntityKindEpic):
		// New format: EPIC-{EPICSLUG}-{filename-slug}
		// Epic slug is uppercase letters, digits, hyphens.
		// Filename slug starts with a lowercase letter.
		if strings.HasPrefix(idPart, "EPIC-") {
			rest := idPart[5:] // after "EPIC-"
			for i := 0; i < len(rest); i++ {
				c := rest[i]
				if c >= 'a' && c <= 'z' {
					if i > 0 && rest[i-1] == '-' {
						return idPart[:5+i-1], rest[i:], nil
					}
					break
				}
			}
		}
		// Fall back to legacy format
		return parseLegacyRecordIdentity(entityType, idPart)

	case string(model.EntityKindFeature), string(model.EntityKindBug),
		string(model.EntityKindDecision), string(model.EntityKindTask),
		string(model.EntityKindDocument):
		// New format: {PREFIX}-{13-char-TSID}-{filename-slug}
		prefix := typePrefixForEntityType(entityType)
		if prefix == "" {
			return "", "", fmt.Errorf("unknown entity type %q", entityType)
		}
		prefixWithDash := prefix + "-"
		if strings.HasPrefix(idPart, prefixWithDash) {
			afterPrefix := idPart[len(prefixWithDash):]
			if len(afterPrefix) >= 14 && afterPrefix[13] == '-' {
				return idPart[:len(prefixWithDash)+13], afterPrefix[14:], nil
			}
		}
		// Fall back to legacy format
		return parseLegacyRecordIdentity(entityType, idPart)

	default:
		return "", "", fmt.Errorf("unknown entity type %q", entityType)
	}
}

func parseLegacyRecordIdentity(entityType, idPart string) (string, string, error) {
	// Legacy format: {PREFIX}-{NNN}-{slug}
	firstDash := strings.Index(idPart, "-")
	if firstDash <= 0 {
		return "", "", fmt.Errorf("invalid %s record filename %q", entityType, idPart)
	}
	secondDash := strings.Index(idPart[firstDash+1:], "-")
	if secondDash <= 0 {
		return "", "", fmt.Errorf("invalid %s record filename %q", entityType, idPart)
	}
	idEnd := firstDash + 1 + secondDash
	return idPart[:idEnd], idPart[idEnd+1:], nil
}

func typePrefixForEntityType(entityType string) string {
	switch entityType {
	case "feature":
		return "FEAT"
	case "bug":
		return "BUG"
	case "decision":
		return "DEC"
	case "task":
		return "TASK"
	case "document":
		return "DOC"
	default:
		return ""
	}
}

func recordFromEntity(entity model.Entity) (storage.EntityRecord, error) {
	switch e := entity.(type) {
	case model.Epic:
		return storage.EntityRecord{
			Type:   string(model.EntityKindEpic),
			ID:     e.ID,
			Slug:   e.Slug,
			Fields: epicFields(e),
		}, nil
	case model.Feature:
		return storage.EntityRecord{
			Type:   string(model.EntityKindFeature),
			ID:     e.ID,
			Slug:   e.Slug,
			Fields: featureFields(e),
		}, nil
	case model.Task:
		return storage.EntityRecord{
			Type:   string(model.EntityKindTask),
			ID:     e.ID,
			Slug:   e.Slug,
			Fields: taskFields(e),
		}, nil
	case model.Bug:
		return storage.EntityRecord{
			Type:   string(model.EntityKindBug),
			ID:     e.ID,
			Slug:   e.Slug,
			Fields: bugFields(e),
		}, nil
	case model.Decision:
		return storage.EntityRecord{
			Type:   string(model.EntityKindDecision),
			ID:     e.ID,
			Slug:   e.Slug,
			Fields: decisionFields(e),
		}, nil
	default:
		return storage.EntityRecord{}, fmt.Errorf("unsupported entity type %T", entity)
	}
}

func epicFields(e model.Epic) map[string]any {
	fields := map[string]any{
		"id":         e.ID,
		"slug":       e.Slug,
		"title":      e.Title,
		"status":     string(e.Status),
		"summary":    e.Summary,
		"created":    e.Created.Format(time.RFC3339),
		"created_by": e.CreatedBy,
	}
	if len(e.Features) > 0 {
		fields["features"] = append([]string(nil), e.Features...)
	}
	return fields
}

func featureFields(e model.Feature) map[string]any {
	fields := map[string]any{
		"id":         e.ID,
		"slug":       e.Slug,
		"epic":       e.Epic,
		"status":     string(e.Status),
		"summary":    e.Summary,
		"created":    e.Created.Format(time.RFC3339),
		"created_by": e.CreatedBy,
	}
	if e.Spec != "" {
		fields["spec"] = e.Spec
	}
	if e.Plan != "" {
		fields["plan"] = e.Plan
	}
	if len(e.Tasks) > 0 {
		fields["tasks"] = append([]string(nil), e.Tasks...)
	}
	if len(e.Decisions) > 0 {
		fields["decisions"] = append([]string(nil), e.Decisions...)
	}
	if e.Branch != "" {
		fields["branch"] = e.Branch
	}
	if e.Supersedes != "" {
		fields["supersedes"] = e.Supersedes
	}
	if e.SupersededBy != "" {
		fields["superseded_by"] = e.SupersededBy
	}
	return fields
}

func taskFields(e model.Task) map[string]any {
	fields := map[string]any{
		"id":             e.ID,
		"parent_feature": e.ParentFeature,
		"slug":           e.Slug,
		"summary":        e.Summary,
		"status":         string(e.Status),
	}
	if e.Assignee != "" {
		fields["assignee"] = e.Assignee
	}
	if len(e.DependsOn) > 0 {
		fields["depends_on"] = append([]string(nil), e.DependsOn...)
	}
	if len(e.FilesPlanned) > 0 {
		fields["files_planned"] = append([]string(nil), e.FilesPlanned...)
	}
	if e.Started != nil {
		fields["started"] = e.Started.Format(time.RFC3339)
	}
	if e.Completed != nil {
		fields["completed"] = e.Completed.Format(time.RFC3339)
	}
	if e.Verification != "" {
		fields["verification"] = e.Verification
	}
	return fields
}

func bugFields(e model.Bug) map[string]any {
	fields := map[string]any{
		"id":          e.ID,
		"slug":        e.Slug,
		"title":       e.Title,
		"status":      string(e.Status),
		"severity":    string(e.Severity),
		"priority":    string(e.Priority),
		"type":        string(e.Type),
		"reported_by": e.ReportedBy,
		"reported":    e.Reported.Format(time.RFC3339),
		"observed":    e.Observed,
		"expected":    e.Expected,
	}
	if len(e.Affects) > 0 {
		fields["affects"] = append([]string(nil), e.Affects...)
	}
	if e.OriginFeature != "" {
		fields["origin_feature"] = e.OriginFeature
	}
	if e.OriginTask != "" {
		fields["origin_task"] = e.OriginTask
	}
	if e.Environment != "" {
		fields["environment"] = e.Environment
	}
	if e.Reproduction != "" {
		fields["reproduction"] = e.Reproduction
	}
	if e.DuplicateOf != "" {
		fields["duplicate_of"] = e.DuplicateOf
	}
	if e.FixedBy != "" {
		fields["fixed_by"] = e.FixedBy
	}
	if e.VerifiedBy != "" {
		fields["verified_by"] = e.VerifiedBy
	}
	if e.ReleaseTarget != "" {
		fields["release_target"] = e.ReleaseTarget
	}
	return fields
}

// cacheUpsertFromResult updates the cache with the entity from a create/update result.
// Failures are silently ignored — the cache is derived and non-essential.
func (s *EntityService) cacheUpsertFromResult(result CreateResult) {
	if s.cache == nil {
		return
	}
	fieldsJSON, err := json.Marshal(result.State)
	if err != nil {
		return
	}
	_ = s.cache.Upsert(cache.EntityRow{
		EntityType: result.Type,
		ID:         result.ID,
		Slug:       result.Slug,
		Status:     stringFromState(result.State, "status"),
		Title:      stringFromState(result.State, "title"),
		Summary:    stringFromState(result.State, "summary"),
		ParentRef:  extractParentRefFromState(result.Type, result.State),
		FilePath:   result.Path,
		FieldsJSON: string(fieldsJSON),
	})
}

func stringFromState(state map[string]any, key string) string {
	if state == nil {
		return ""
	}
	v, ok := state[key]
	if !ok {
		return ""
	}
	s, ok := v.(string)
	if !ok {
		return fmt.Sprint(v)
	}
	return s
}

func extractParentRefFromState(entityType string, state map[string]any) string {
	switch strings.ToLower(entityType) {
	case "feature":
		return stringFromState(state, "epic")
	case "task":
		return stringFromState(state, "parent_feature")
	case "bug":
		return stringFromState(state, "origin_feature")
	default:
		return ""
	}
}

func decisionFields(e model.Decision) map[string]any {
	fields := map[string]any{
		"id":         e.ID,
		"slug":       e.Slug,
		"summary":    e.Summary,
		"rationale":  e.Rationale,
		"decided_by": e.DecidedBy,
		"date":       e.Date.Format(time.RFC3339),
		"status":     string(e.Status),
	}
	if len(e.Affects) > 0 {
		fields["affects"] = append([]string(nil), e.Affects...)
	}
	if e.Supersedes != "" {
		fields["supersedes"] = e.Supersedes
	}
	if e.SupersededBy != "" {
		fields["superseded_by"] = e.SupersededBy
	}
	return fields
}
