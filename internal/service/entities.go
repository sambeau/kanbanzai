package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/sambeau/kanbanzai/internal/cache"
	"github.com/sambeau/kanbanzai/internal/core"
	"github.com/sambeau/kanbanzai/internal/id"
	"github.com/sambeau/kanbanzai/internal/model"
	"github.com/sambeau/kanbanzai/internal/storage"
	"github.com/sambeau/kanbanzai/internal/validate"
)

type CreateFeatureInput struct {
	Slug      string
	Parent    string
	Design    string
	Tags      []string
	Summary   string
	CreatedBy string
	Name      string
}

type CreateTaskInput struct {
	ParentFeature string
	Slug          string
	Summary       string
	Name          string
}

type CreateBugInput struct {
	Slug       string
	Name       string
	ReportedBy string
	Observed   string
	Expected   string
	Severity   string
	Priority   string
	Type       string
}

type CreateDecisionInput struct {
	Slug      string
	Name      string
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
	Type       string              // entity type: "feature", "task", "bug", "decision"
	ID         string              // entity ID
	Slug       string              // entity slug
	Fields     map[string]string   // field name → new value (string values only)
	ListFields map[string][]string // field name → new value (list values, e.g. depends_on)
}

type CreateResult struct {
	Type  string
	ID    string
	Slug  string
	Path  string
	State map[string]any

	// WorktreeHookResult is set by the status transition hook when a
	// worktree was automatically created (or attempted) during a status
	// update. It is nil for non-transition operations and for transitions
	// that don't trigger worktree creation.
	WorktreeHookResult *WorktreeResult `json:"-"`
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
	root       string
	store      *storage.EntityStore
	allocator  *id.Allocator
	now        func() time.Time
	cache      *cache.Cache
	statusHook StatusTransitionHook // optional, for automatic worktree creation
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

// SetStatusTransitionHook attaches an optional hook that fires after
// successful status transitions. Used for automatic worktree creation
// when tasks become active or bugs move to in-progress.
func (s *EntityService) SetStatusTransitionHook(hook StatusTransitionHook) {
	s.statusHook = hook
}

// Root returns the state root path for this service.
func (s *EntityService) Root() string {
	return s.root
}

// Store returns the underlying entity store for low-level access in dispatch operations.
func (s *EntityService) Store() *storage.EntityStore {
	return s.store
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
	// plan is intentionally excluded: EntityService.List("plan") is unsupported
	// because plan files use a slug-free naming convention ({id}.yaml) that the
	// generic load path cannot handle. Plans are managed by a separate plan service.
	for _, kind := range []string{
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

func (s *EntityService) CreateFeature(input CreateFeatureInput) (CreateResult, error) {
	if err := validateRequired(
		field("slug", input.Slug),
		field("summary", input.Summary),
		field("created_by", input.CreatedBy),
	); err != nil {
		return CreateResult{}, err
	}

	parentID := strings.TrimSpace(input.Parent)
	if parentID == "" {
		return CreateResult{}, fmt.Errorf("parent plan or batch is required: feature must belong to a plan or batch")
	}

	// Load parent — provides existence check, next_feature_seq, and number.
	planResult, err := s.GetPlan(parentID)
	if err != nil {
		return CreateResult{}, fmt.Errorf("parent %s: %w", parentID, ErrReferenceNotFound)
	}

	// Read next_feature_seq (default 1 if absent).
	seq := intFromState(planResult.State, "next_feature_seq", 1)

	// Compute display_id: {Prefix}{number}-F{seq} (e.g. "B24-F1" for batch, "P37-F5" for legacy plan).
	parentPrefix, planNum, _ := model.ParsePlanID(parentID)
	displayID := fmt.Sprintf("%s%s-F%d", parentPrefix, planNum, seq)

	// Write parent with incremented counter BEFORE writing feature (REQ-006).
	planResult.State["next_feature_seq"] = seq + 1
	planRecord := storage.EntityRecord{
		Type:   string(model.EntityKindBatch),
		ID:     planResult.ID,
		Slug:   planResult.Slug,
		Fields: planResult.State,
	}
	if _, err := s.store.Write(planRecord); err != nil {
		return CreateResult{}, fmt.Errorf("increment sequence for %s: %w", parentID, err)
	}

	idValue, err := s.allocateID(model.EntityKindFeature)
	if err != nil {
		return CreateResult{}, err
	}

	featureName, nameErr := validate.ValidateName(input.Name)
	if nameErr != nil {
		return CreateResult{}, nameErr
	}

	entity := model.Feature{
		ID:        idValue,
		Slug:      normalizeSlug(input.Slug),
		Name:      featureName,
		Parent:    parentID,
		DisplayID: displayID,
		Status:    model.FeatureStatusProposed,
		Summary:   strings.TrimSpace(input.Summary),
		Design:    strings.TrimSpace(input.Design),
		Tags:      append([]string(nil), input.Tags...),
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

	taskName, nameErr := validate.ValidateName(input.Name)
	if nameErr != nil {
		return CreateResult{}, nameErr
	}

	entity := model.Task{
		ID:            idValue,
		ParentFeature: featureID,
		Slug:          normalizeSlug(input.Slug),
		Name:          taskName,
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
		field("name", input.Name),
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

	bugName, nameErr := validate.ValidateName(input.Name)
	if nameErr != nil {
		return CreateResult{}, nameErr
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
		Name:       bugName,
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

	decisionName, nameErr := validate.ValidateName(input.Name)
	if nameErr != nil {
		return CreateResult{}, nameErr
	}

	entity := model.Decision{
		ID:        idValue,
		Slug:      normalizeSlug(input.Slug),
		Name:      decisionName,
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

		// Load Plans via ListPlans (Plans use different filename format).
		plans, err := s.ListPlans(PlanFilters{})
		if err != nil {
			return nil, fmt.Errorf("listing plan entities: %w", err)
		}
		for _, r := range plans {
			all = append(all, validate.EntityInfo{
				Type:   r.Type,
				ID:     r.ID,
				Fields: r.State,
			})
		}

		for _, kind := range []string{
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
		// Plans use different storage, check via file directly.
		if entityType == string(model.EntityKindPlan) {
			return s.entityExists(entityType, id)
		}
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
	// Resolve P{n}-F{m} display ID to canonical FEAT-TSID.
	if entityType == "feature" && IsFeatureDisplayID(entityID) {
		resolvedID, resolvedSlug, err := s.ResolveFeatureDisplayID(entityID)
		if err != nil {
			return GetResult{}, err
		}
		entityID = resolvedID
		slug = resolvedSlug
	}
	if slug == "" {
		// Cache fast path: when cache is warm for this type, resolve slug without
		// a directory scan. Fall through to ResolvePrefix on miss or Load error.
		if s.cache != nil && s.cache.IsWarm(entityType) {
			if cachedSlug, _, found := s.cache.LookupByID(entityType, entityID); found {
				record, err := s.store.Load(entityType, entityID, cachedSlug)
				if err == nil {
					return GetResult{
						Type:  record.Type,
						ID:    record.ID,
						Slug:  record.Slug,
						Path:  filepath.Join(s.root, entityDirectory(record.Type), entityFileName(record.ID, record.Slug)),
						State: record.Fields,
					}, nil
				}
				// Load failed (stale cache entry) — fall through to ResolvePrefix
				log.Printf("[entity] cache hit but Load failed for %s/%s (falling back): %v", entityType, entityID, err)
			}
			// Cache miss — fall through to ResolvePrefix
		}
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

	// Cache fast path: when cache is warm for this type, serve from SQLite
	// instead of scanning the filesystem. Corrupt fields_json returns an error;
	// ListByType error falls through to filepath.Glob.
	if s.cache != nil && s.cache.IsWarm(entityType) {
		rows, err := s.cache.ListByType(entityType)
		if err == nil {
			results := make([]ListResult, 0, len(rows))
			for _, row := range rows {
				var fields map[string]any
				if err := json.Unmarshal([]byte(row.FieldsJSON), &fields); err != nil {
					return nil, fmt.Errorf("list %s: corrupt cache entry for %s: %w", entityType, row.ID, err)
				}
				results = append(results, ListResult{
					Type:  row.EntityType,
					ID:    row.ID,
					Slug:  row.Slug,
					Path:  row.FilePath,
					State: fields,
				})
			}
			return results, nil
		}
		// ListByType error — fall through to filesystem path
		log.Printf("[entity] cache ListByType error for %s (falling back): %v", entityType, err)
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

	// Resolve P{n}-F{m} display ID to canonical FEAT-TSID.
	if entityType == "feature" && IsFeatureDisplayID(entityID) {
		resolvedID, resolvedSlug, err := s.ResolveFeatureDisplayID(entityID)
		if err != nil {
			return GetResult{}, err
		}
		entityID = resolvedID
		slug = resolvedSlug
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

	// Clear rework_reason when a task transitions from needs-rework to active.
	if entityType == "task" && currentStatusText == "needs-rework" && nextStatus == "active" {
		delete(record.Fields, "rework_reason")
	}

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

	// Fire status transition hook (e.g. automatic worktree creation).
	// The hook result is stored on the result for the caller to include
	// in its response. Hook failures never block the transition.
	if s.statusHook != nil {
		wtResult := s.statusHook.OnStatusTransition(entityType, entityID, slug, currentStatusText, nextStatus, record.Fields)
		if wtResult != nil {
			result.WorktreeHookResult = wtResult
		}
	}

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

	// Resolve P{n}-F{m} display ID to canonical FEAT-TSID.
	if entityType == "feature" && IsFeatureDisplayID(entityID) {
		resolvedID, resolvedSlug, err := s.ResolveFeatureDisplayID(entityID)
		if err != nil {
			return GetResult{}, err
		}
		entityID = resolvedID
		slug = resolvedSlug
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

	for k, v := range input.ListFields {
		if len(v) == 0 {
			delete(record.Fields, k)
		} else {
			record.Fields[k] = append([]string(nil), v...)
		}
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
	// Plan files use {id}.yaml (no slug suffix), so check that first.
	if strings.ToLower(strings.TrimSpace(entityType)) == string(model.EntityKindPlan) {
		_, err := os.Stat(filepath.Join(dir, entityID+".yaml"))
		return err == nil
	}
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
	case string(model.EntityKindPlan):
		return validate.EntityPlan, nil
	case string(model.EntityKindFeature):
		return validate.EntityFeature, nil
	case string(model.EntityKindTask):
		return validate.EntityTask, nil
	case string(model.EntityKindBug):
		return validate.EntityBug, nil
	case string(model.EntityKindDecision):
		return validate.EntityDecision, nil
	case string(model.EntityKindIncident):
		return validate.EntityIncident, nil
	default:
		return "", fmt.Errorf("unknown entity type %q", entityType)
	}
}

func parseRecordIdentity(entityType, idPart string) (string, string, error) {
	switch entityType {
	case string(model.EntityKindPlan):
		// Plan files use {id}.yaml with no slug suffix. The entire idPart is the ID.
		if prefix, _, _ := model.ParsePlanID(idPart); prefix != "" {
			return idPart, "", nil
		}
		return "", "", fmt.Errorf("invalid plan record filename %q", idPart)

	case string(model.EntityKindFeature), string(model.EntityKindBug),
		string(model.EntityKindDecision), string(model.EntityKindTask),
		string(model.EntityKindDocument), string(model.EntityKindIncident):
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
	case "incident":
		return "INC"
	default:
		return ""
	}
}

func recordFromEntity(entity model.Entity) (storage.EntityRecord, error) {
	switch e := entity.(type) {
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
	case model.Incident:
		return storage.EntityRecord{
			Type:   string(model.EntityKindIncident),
			ID:     e.ID,
			Slug:   e.Slug,
			Fields: incidentFields(e),
		}, nil
	default:
		return storage.EntityRecord{}, fmt.Errorf("internal error: entity type is not supported for serialisation — this is likely a bug; please report it")
	}
}

func featureFields(e model.Feature) map[string]any {
	fields := map[string]any{
		"id":         e.ID,
		"slug":       e.Slug,
		"parent":     e.Parent,
		"status":     string(e.Status),
		"summary":    e.Summary,
		"created":    e.Created.Format(time.RFC3339),
		"created_by": e.CreatedBy,
	}
	if e.DisplayID != "" {
		fields["display_id"] = e.DisplayID
	}
	if e.Estimate != nil {
		fields["estimate"] = *e.Estimate
	}
	if !e.Updated.IsZero() {
		fields["updated"] = e.Updated.Format(time.RFC3339)
	}
	if e.Design != "" {
		fields["design"] = e.Design
	}
	if e.Spec != "" {
		fields["spec"] = e.Spec
	}
	if e.DevPlan != "" {
		fields["dev_plan"] = e.DevPlan
	}
	if len(e.Tasks) > 0 {
		fields["tasks"] = append([]string(nil), e.Tasks...)
	}
	if len(e.Decisions) > 0 {
		fields["decisions"] = append([]string(nil), e.Decisions...)
	}
	fields["name"] = e.Name
	if len(e.Tags) > 0 {
		fields["tags"] = append([]string(nil), e.Tags...)
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
	if len(e.Overrides) > 0 {
		overrides := make([]any, len(e.Overrides))
		for i, o := range e.Overrides {
			overrides[i] = map[string]any{
				"from_status": o.FromStatus,
				"to_status":   o.ToStatus,
				"reason":      o.Reason,
				"timestamp":   o.Timestamp.Format(time.RFC3339),
			}
		}
		fields["overrides"] = overrides
	}
	return fields
}

// PersistFeatureOverrides writes the given override records to the feature entity
// on disk, replacing any previously stored overrides. Called after each gate
// bypass to ensure override history is durable (FR-014).
func (s *EntityService) PersistFeatureOverrides(featureID, slug string, overrides []model.OverrideRecord) error {
	if slug == "" {
		_, resolvedSlug, err := s.ResolvePrefix("feature", featureID)
		if err != nil {
			return err
		}
		slug = resolvedSlug
	}

	record, err := s.store.Load("feature", featureID, slug)
	if err != nil {
		return err
	}

	if len(overrides) == 0 {
		delete(record.Fields, "overrides")
	} else {
		overrideList := make([]any, len(overrides))
		for i, o := range overrides {
			overrideList[i] = map[string]any{
				"from_status": o.FromStatus,
				"to_status":   o.ToStatus,
				"reason":      o.Reason,
				"timestamp":   o.Timestamp.Format(time.RFC3339),
			}
		}
		record.Fields["overrides"] = overrideList
	}

	_, err = s.store.Write(record)
	return err
}

func taskFields(e model.Task) map[string]any {
	fields := map[string]any{
		"id":             e.ID,
		"parent_feature": e.ParentFeature,
		"slug":           e.Slug,
		"summary":        e.Summary,
		"status":         string(e.Status),
	}
	if e.Estimate != nil {
		fields["estimate"] = *e.Estimate
	}
	fields["name"] = e.Name
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
	if e.ClaimedAt != nil {
		fields["claimed_at"] = e.ClaimedAt.Format(time.RFC3339)
	}
	if e.DispatchedTo != "" {
		fields["dispatched_to"] = e.DispatchedTo
	}
	if e.DispatchedAt != nil {
		fields["dispatched_at"] = e.DispatchedAt.Format(time.RFC3339)
	}
	if e.DispatchedBy != "" {
		fields["dispatched_by"] = e.DispatchedBy
	}
	if e.CompletionSummary != "" {
		fields["completion_summary"] = e.CompletionSummary
	}
	if e.ReworkReason != "" {
		fields["rework_reason"] = e.ReworkReason
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
		"name":        e.Name,
		"status":      string(e.Status),
		"severity":    string(e.Severity),
		"priority":    string(e.Priority),
		"type":        string(e.Type),
		"reported_by": e.ReportedBy,
		"reported":    e.Reported.Format(time.RFC3339),
		"observed":    e.Observed,
		"expected":    e.Expected,
	}
	if e.Estimate != nil {
		fields["estimate"] = *e.Estimate
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
		Title:      stringFromState(result.State, "name"),
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

var featureDisplayIDPattern = regexp.MustCompile(`(?i)^([BP])(\d+)-F(\d+)$`)

// IsFeatureDisplayID reports whether id matches the B{n}-F{m} or P{n}-F{m} display ID pattern.
func IsFeatureDisplayID(id string) bool {
	return featureDisplayIDPattern.MatchString(id)
}

// ResolveFeatureDisplayID resolves a B{n}-F{m} or P{n}-F{m} display ID to (canonicalID, slug).
// Uses the SQLite cache when warm (O(1)); falls back to a filesystem scan.
func (s *EntityService) ResolveFeatureDisplayID(displayID string) (string, string, error) {
	if s.cache != nil && s.cache.IsWarm("feature") {
		if id, slug, _, found := s.cache.LookupByDisplayID(displayID); found {
			return id, slug, nil
		}
		return "", "", fmt.Errorf("feature with display_id %s: %w", displayID, ErrNotFound)
	}
	// Filesystem scan fallback.
	results, err := s.List("feature")
	if err != nil {
		return "", "", fmt.Errorf("resolve display_id %s: %w", displayID, err)
	}
	upper := strings.ToUpper(displayID)
	for _, r := range results {
		if did, _ := r.State["display_id"].(string); strings.ToUpper(did) == upper {
			return r.ID, r.Slug, nil
		}
	}
	return "", "", fmt.Errorf("feature with display_id %s: %w", displayID, ErrNotFound)
}

func intFromState(state map[string]any, key string, defaultVal int) int {
	if state == nil {
		return defaultVal
	}
	v, ok := state[key]
	if !ok {
		return defaultVal
	}
	switch n := v.(type) {
	case int:
		return n
	case float64:
		return int(n)
	default:
		return defaultVal
	}
}

func extractParentRefFromState(entityType string, state map[string]any) string {
	switch strings.ToLower(entityType) {
	case "feature":
		return stringFromState(state, "parent")
	case "task":
		return stringFromState(state, "parent_feature")
	case "bug":
		return stringFromState(state, "origin_feature")
	case "plan":
		return "" // Plans have no parent
	default:
		return ""
	}
}

// listDirectory returns the names of entries in a directory.
// Returns nil, nil if the directory doesn't exist.
func listDirectory(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read directory %s: %w", dir, err)
	}

	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		names = append(names, entry.Name())
	}
	return names, nil
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
	fields["name"] = e.Name
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

func incidentFields(e model.Incident) map[string]any {
	fields := map[string]any{
		"id":          e.ID,
		"slug":        e.Slug,
		"name":        e.Name,
		"status":      string(e.Status),
		"severity":    string(e.Severity),
		"reported_by": e.ReportedBy,
		"detected_at": e.DetectedAt.Format(time.RFC3339),
		"summary":     e.Summary,
		"created":     e.Created.Format(time.RFC3339),
		"created_by":  e.CreatedBy,
		"updated":     e.Updated.Format(time.RFC3339),
	}
	if e.TriagedAt != nil {
		fields["triaged_at"] = e.TriagedAt.Format(time.RFC3339)
	}
	if e.MitigatedAt != nil {
		fields["mitigated_at"] = e.MitigatedAt.Format(time.RFC3339)
	}
	if e.ResolvedAt != nil {
		fields["resolved_at"] = e.ResolvedAt.Format(time.RFC3339)
	}
	if len(e.AffectedFeatures) > 0 {
		fields["affected_features"] = append([]string(nil), e.AffectedFeatures...)
	}
	if len(e.LinkedBugs) > 0 {
		fields["linked_bugs"] = append([]string(nil), e.LinkedBugs...)
	}
	if e.LinkedRCA != "" {
		fields["linked_rca"] = e.LinkedRCA
	}
	return fields
}
