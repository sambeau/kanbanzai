package service

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"kanbanzai/internal/core"
	"kanbanzai/internal/id"
	"kanbanzai/internal/knowledge"
	"kanbanzai/internal/model"
	"kanbanzai/internal/storage"
	"kanbanzai/internal/validate"
)

// ContributeInput holds the parameters for contributing a new knowledge entry.
type ContributeInput struct {
	Topic       string
	Scope       string
	Content     string
	Tier        int
	LearnedFrom string
	CreatedBy   string
	Tags        []string
}

// KnowledgeFilters holds optional filter criteria for listing knowledge entries.
type KnowledgeFilters struct {
	Tier           int
	Scope          string
	Status         string
	Topic          string
	Tags           []string
	MinConfidence  float64
	IncludeRetired bool
}

// FlaggedEntry represents a knowledge entry flagged as incorrect in a context report.
type FlaggedEntry struct {
	EntryID string `json:"entry_id"`
	Reason  string `json:"reason"`
}

// KnowledgeService provides business logic for knowledge entry management.
type KnowledgeService struct {
	root  string
	store *storage.KnowledgeStore
	now   func() time.Time
}

// NewKnowledgeService creates a new KnowledgeService.
// root should be the .kbz/state directory; pass empty string for the default.
func NewKnowledgeService(root string) *KnowledgeService {
	if strings.TrimSpace(root) == "" {
		root = core.StatePath()
	}
	return &KnowledgeService{
		root:  root,
		store: storage.NewKnowledgeStore(root),
		now:   func() time.Time { return time.Now().UTC() },
	}
}

// Contribute creates a new knowledge entry after deduplication checks.
// Returns (new record, nil, nil) on success.
// Returns (zero, &duplicate, error) if rejected due to an existing duplicate.
func (s *KnowledgeService) Contribute(input ContributeInput) (storage.KnowledgeRecord, *storage.KnowledgeRecord, error) {
	if strings.TrimSpace(input.Topic) == "" {
		return storage.KnowledgeRecord{}, nil, fmt.Errorf("topic is required")
	}
	if strings.TrimSpace(input.Content) == "" {
		return storage.KnowledgeRecord{}, nil, fmt.Errorf("content is required")
	}
	if strings.TrimSpace(input.Scope) == "" {
		return storage.KnowledgeRecord{}, nil, fmt.Errorf("scope is required")
	}

	normTopic := knowledge.NormalizeTopic(input.Topic)

	tier := input.Tier
	if tier != 2 && tier != 3 {
		tier = 3
	}

	all, err := s.store.LoadAll()
	if err != nil {
		return storage.KnowledgeRecord{}, nil, fmt.Errorf("load knowledge entries for dedup: %w", err)
	}

	inputWords := knowledge.ContentWords(input.Content)

	for i := range all {
		rec := &all[i]
		recScope, _ := rec.Fields["scope"].(string)
		if recScope != input.Scope {
			continue
		}
		recStatus, _ := rec.Fields["status"].(string)
		if recStatus == string(model.KnowledgeStatusRetired) {
			continue
		}

		// Exact topic match → reject
		recTopic, _ := rec.Fields["topic"].(string)
		if recTopic == normTopic {
			return storage.KnowledgeRecord{}, rec, fmt.Errorf("duplicate topic %q in scope %q: existing entry %s", normTopic, input.Scope, rec.ID)
		}

		// Near-duplicate via Jaccard similarity > 0.65 → reject
		recContent, _ := rec.Fields["content"].(string)
		recWords := knowledge.ContentWords(recContent)
		if sim := knowledge.JaccardSimilarity(inputWords, recWords); sim > 0.65 {
			return storage.KnowledgeRecord{}, rec, fmt.Errorf("near-duplicate content (similarity %.2f) in scope %q: existing entry %s", sim, input.Scope, rec.ID)
		}
	}

	tsid, err := id.GenerateTSID13()
	if err != nil {
		return storage.KnowledgeRecord{}, nil, fmt.Errorf("generate ID: %w", err)
	}
	entryID := "KE-" + tsid

	ttlDays := 30
	if tier == 2 {
		ttlDays = 90
	}

	now := s.now()
	createdBy := strings.TrimSpace(input.CreatedBy)
	if createdBy == "" {
		createdBy = "unknown"
	}

	fields := map[string]any{
		"id":         entryID,
		"tier":       tier,
		"topic":      normTopic,
		"scope":      input.Scope,
		"content":    input.Content,
		"status":     string(model.KnowledgeStatusContributed),
		"use_count":  0,
		"miss_count": 0,
		"confidence": 0.5,
		"ttl_days":   ttlDays,
		"created":    now.Format(time.RFC3339),
		"created_by": createdBy,
		"updated":    now.Format(time.RFC3339),
	}

	if input.LearnedFrom != "" {
		fields["learned_from"] = input.LearnedFrom
	}
	if len(input.Tags) > 0 {
		fields["tags"] = input.Tags
	}

	record := storage.KnowledgeRecord{
		ID:     entryID,
		Fields: fields,
	}

	if _, err := s.store.Write(record); err != nil {
		return storage.KnowledgeRecord{}, nil, fmt.Errorf("write knowledge entry: %w", err)
	}

	return record, nil, nil
}

// LoadAllRaw returns all knowledge records without filtering.
// This is intended for health checks and administrative operations.
func (s *KnowledgeService) LoadAllRaw() ([]storage.KnowledgeRecord, error) {
	return s.store.LoadAll()
}

// Get returns a knowledge entry by ID.
func (s *KnowledgeService) Get(id string) (storage.KnowledgeRecord, error) {
	record, err := s.store.Load(id)
	if err != nil {
		return storage.KnowledgeRecord{}, err
	}
	return record, nil
}

// List returns knowledge entries matching the given filters.
// By default, retired entries are excluded unless IncludeRetired is true.
func (s *KnowledgeService) List(filters KnowledgeFilters) ([]storage.KnowledgeRecord, error) {
	all, err := s.store.LoadAll()
	if err != nil {
		return nil, fmt.Errorf("list knowledge entries: %w", err)
	}

	var result []storage.KnowledgeRecord
	for _, rec := range all {
		status, _ := rec.Fields["status"].(string)

		if !filters.IncludeRetired && status == string(model.KnowledgeStatusRetired) {
			continue
		}
		if filters.Status != "" && status != filters.Status {
			continue
		}
		if filters.Scope != "" {
			scope, _ := rec.Fields["scope"].(string)
			if scope != filters.Scope {
				continue
			}
		}
		if filters.Topic != "" {
			topic, _ := rec.Fields["topic"].(string)
			if topic != knowledge.NormalizeTopic(filters.Topic) {
				continue
			}
		}
		if filters.Tier != 0 {
			if knowledgeFieldInt(rec.Fields, "tier") != filters.Tier {
				continue
			}
		}
		if filters.MinConfidence > 0 {
			if knowledgeFieldFloat(rec.Fields, "confidence") < filters.MinConfidence {
				continue
			}
		}
		if len(filters.Tags) > 0 {
			recTags := knowledgeFieldStrings(rec.Fields, "tags")
			if !knowledgeHasAllTags(recTags, filters.Tags) {
				continue
			}
		}

		result = append(result, rec)
	}

	return result, nil
}

// Update replaces the content of a knowledge entry and resets its confidence counters.
func (s *KnowledgeService) Update(id, content string) (storage.KnowledgeRecord, error) {
	record, err := s.store.Load(id)
	if err != nil {
		return storage.KnowledgeRecord{}, err
	}

	record.Fields["content"] = content
	record.Fields["use_count"] = 0
	record.Fields["miss_count"] = 0
	record.Fields["confidence"] = 0.5
	record.Fields["updated"] = s.now().Format(time.RFC3339)

	if _, err := s.store.Write(record); err != nil {
		return storage.KnowledgeRecord{}, fmt.Errorf("write knowledge entry: %w", err)
	}

	return record, nil
}

// Confirm manually transitions a knowledge entry to confirmed status.
func (s *KnowledgeService) Confirm(id string) (storage.KnowledgeRecord, error) {
	record, err := s.store.Load(id)
	if err != nil {
		return storage.KnowledgeRecord{}, err
	}

	status, _ := record.Fields["status"].(string)
	if err := validate.ValidateKnowledgeTransition(status, string(model.KnowledgeStatusConfirmed)); err != nil {
		return storage.KnowledgeRecord{}, err
	}

	record.Fields["status"] = string(model.KnowledgeStatusConfirmed)
	record.Fields["updated"] = s.now().Format(time.RFC3339)

	if _, err := s.store.Write(record); err != nil {
		return storage.KnowledgeRecord{}, fmt.Errorf("write knowledge entry: %w", err)
	}

	return record, nil
}

// Flag manually flags a knowledge entry as incorrect or disputed.
// Increments miss_count and recomputes confidence. If miss_count reaches 2 or above,
// the entry is automatically retired. Otherwise it transitions to disputed
// (if the current status allows it).
func (s *KnowledgeService) Flag(id, reason string) (storage.KnowledgeRecord, error) {
	record, err := s.store.Load(id)
	if err != nil {
		return storage.KnowledgeRecord{}, err
	}

	status, _ := record.Fields["status"].(string)
	if status == string(model.KnowledgeStatusRetired) {
		return storage.KnowledgeRecord{}, fmt.Errorf("knowledge entry %s is already retired", id)
	}

	missCount := knowledgeFieldInt(record.Fields, "miss_count") + 1
	useCount := knowledgeFieldInt(record.Fields, "use_count")
	conf := knowledge.WilsonScore(useCount, missCount)

	record.Fields["miss_count"] = missCount
	record.Fields["confidence"] = conf
	record.Fields["updated"] = s.now().Format(time.RFC3339)

	if missCount >= 2 {
		record.Fields["status"] = string(model.KnowledgeStatusRetired)
		if reason != "" {
			record.Fields["deprecated_reason"] = reason
		}
	} else if validate.CanTransitionKnowledge(status, string(model.KnowledgeStatusDisputed)) {
		record.Fields["status"] = string(model.KnowledgeStatusDisputed)
	}

	if _, err := s.store.Write(record); err != nil {
		return storage.KnowledgeRecord{}, fmt.Errorf("write knowledge entry: %w", err)
	}

	return record, nil
}

// Retire manually retires a knowledge entry with an optional reason.
func (s *KnowledgeService) Retire(id, reason string) (storage.KnowledgeRecord, error) {
	record, err := s.store.Load(id)
	if err != nil {
		return storage.KnowledgeRecord{}, err
	}

	status, _ := record.Fields["status"].(string)
	if err := validate.ValidateKnowledgeTransition(status, string(model.KnowledgeStatusRetired)); err != nil {
		return storage.KnowledgeRecord{}, err
	}

	record.Fields["status"] = string(model.KnowledgeStatusRetired)
	if reason != "" {
		record.Fields["deprecated_reason"] = reason
	}
	record.Fields["updated"] = s.now().Format(time.RFC3339)

	if _, err := s.store.Write(record); err != nil {
		return storage.KnowledgeRecord{}, fmt.Errorf("write knowledge entry: %w", err)
	}

	return record, nil
}

// Promote promotes a tier-3 knowledge entry to tier 2 in place.
// Updates tier, ttl_days, and records the promotion.
func (s *KnowledgeService) Promote(id string) (storage.KnowledgeRecord, error) {
	record, err := s.store.Load(id)
	if err != nil {
		return storage.KnowledgeRecord{}, err
	}

	tier := knowledgeFieldInt(record.Fields, "tier")
	if tier != 3 {
		return storage.KnowledgeRecord{}, fmt.Errorf("only tier-3 entries can be promoted (entry %s is tier %d)", id, tier)
	}

	record.Fields["tier"] = 2
	record.Fields["ttl_days"] = 90
	record.Fields["promoted_from"] = id
	record.Fields["updated"] = s.now().Format(time.RFC3339)

	if _, err := s.store.Write(record); err != nil {
		return storage.KnowledgeRecord{}, fmt.Errorf("write knowledge entry: %w", err)
	}

	return record, nil
}

// ContextReport processes a usage report from an agent.
// For each used entry, increments use_count and updates last_used; may auto-confirm.
// For each flagged entry, increments miss_count; may auto-retire.
// Errors from individual entry updates are silently skipped (best-effort).
func (s *KnowledgeService) ContextReport(taskID string, used []string, flagged []FlaggedEntry) error {
	now := s.now()

	for _, entryID := range used {
		record, err := s.store.Load(entryID)
		if err != nil {
			continue // Entry may not exist; skip
		}

		useCount := knowledgeFieldInt(record.Fields, "use_count") + 1
		missCount := knowledgeFieldInt(record.Fields, "miss_count")
		conf := knowledge.WilsonScore(useCount, missCount)

		record.Fields["use_count"] = useCount
		record.Fields["last_used"] = now.Format(time.RFC3339)
		record.Fields["confidence"] = conf
		record.Fields["updated"] = now.Format(time.RFC3339)

		// Auto-confirm: contributed + use_count >= 3 + miss_count == 0
		status, _ := record.Fields["status"].(string)
		if status == string(model.KnowledgeStatusContributed) && useCount >= 3 && missCount == 0 {
			record.Fields["status"] = string(model.KnowledgeStatusConfirmed)
		}

		s.store.Write(record) //nolint:errcheck // best-effort
	}

	for _, f := range flagged {
		record, err := s.store.Load(f.EntryID)
		if err != nil {
			continue
		}

		missCount := knowledgeFieldInt(record.Fields, "miss_count") + 1
		useCount := knowledgeFieldInt(record.Fields, "use_count")
		conf := knowledge.WilsonScore(useCount, missCount)

		record.Fields["miss_count"] = missCount
		record.Fields["confidence"] = conf
		record.Fields["updated"] = now.Format(time.RFC3339)

		status, _ := record.Fields["status"].(string)
		if missCount >= 2 && status != string(model.KnowledgeStatusRetired) {
			record.Fields["status"] = string(model.KnowledgeStatusRetired)
			if f.Reason != "" {
				record.Fields["deprecated_reason"] = f.Reason
			}
		}

		s.store.Write(record) //nolint:errcheck // best-effort
	}

	return nil
}

// ParseFlaggedEntries parses a JSON array of flagged entries from a string.
// Returns an empty slice (not an error) for empty or blank input.
func ParseFlaggedEntries(raw string) ([]FlaggedEntry, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	var entries []FlaggedEntry
	if err := json.Unmarshal([]byte(raw), &entries); err != nil {
		return nil, fmt.Errorf("parse flagged entries: %w", err)
	}
	return entries, nil
}

// knowledgeFieldInt reads an integer value from the Fields map.
// Handles int, float64, and string representations (from YAML round-trips).
func knowledgeFieldInt(fields map[string]any, key string) int {
	v := fields[key]
	switch typed := v.(type) {
	case int:
		return typed
	case int64:
		return int(typed)
	case float64:
		return int(typed)
	case string:
		if i, err := strconv.Atoi(typed); err == nil {
			return i
		}
	}
	return 0
}

// knowledgeFieldFloat reads a float64 value from the Fields map.
// Handles float64, int, and string representations (from YAML round-trips).
func knowledgeFieldFloat(fields map[string]any, key string) float64 {
	v := fields[key]
	switch typed := v.(type) {
	case float64:
		return typed
	case int:
		return float64(typed)
	case string:
		if f, err := strconv.ParseFloat(typed, 64); err == nil {
			return f
		}
	}
	return 0
}

// knowledgeFieldStrings reads a string slice from the Fields map.
// Handles both []string and []any (from YAML round-trips).
func knowledgeFieldStrings(fields map[string]any, key string) []string {
	v := fields[key]
	switch typed := v.(type) {
	case []string:
		return typed
	case []any:
		result := make([]string, 0, len(typed))
		for _, item := range typed {
			if s, ok := item.(string); ok {
				result = append(result, s)
			}
		}
		return result
	}
	return nil
}

// knowledgeHasAllTags returns true if haystack contains every element of needles.
func knowledgeHasAllTags(haystack, needles []string) bool {
	set := make(map[string]struct{}, len(haystack))
	for _, t := range haystack {
		set[t] = struct{}{}
	}
	for _, t := range needles {
		if _, ok := set[t]; !ok {
			return false
		}
	}
	return true
}
