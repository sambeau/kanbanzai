package service

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/sambeau/kanbanzai/internal/model"
	"github.com/sambeau/kanbanzai/internal/validate"
)

// CreateIncidentInput holds the parameters for creating a new incident.
type CreateIncidentInput struct {
	Slug       string
	Name       string
	Severity   string
	Summary    string
	ReportedBy string
	DetectedAt string // ISO 8601; defaults to now if empty
}

// UpdateIncidentInput holds the parameters for updating an existing incident.
type UpdateIncidentInput struct {
	ID               string
	Slug             string   // optional, used for resolution if ID is a prefix
	Status           string   // optional, new lifecycle status
	Severity         string   // optional
	Summary          string   // optional
	TriagedAt        string   // optional, ISO 8601
	MitigatedAt      string   // optional, ISO 8601
	ResolvedAt       string   // optional, ISO 8601
	AffectedFeatures []string // optional, replaces existing list
}

// LinkBugInput holds the parameters for linking a bug to an incident.
type LinkBugInput struct {
	IncidentID string
	BugID      string
}

// CreateIncident creates a new incident in "reported" status.
func (s *EntityService) CreateIncident(input CreateIncidentInput) (CreateResult, error) {
	if err := validateRequired(
		field("slug", input.Slug),
		field("name", input.Name),
		field("severity", input.Severity),
		field("summary", input.Summary),
		field("reported_by", input.ReportedBy),
	); err != nil {
		return CreateResult{}, err
	}

	incidentName, nameErr := validate.ValidateName(input.Name)
	if nameErr != nil {
		return CreateResult{}, nameErr
	}

	if err := validate.ValidateIncidentSeverity(input.Severity); err != nil {
		return CreateResult{}, err
	}

	idValue, err := s.allocateID(model.EntityKindIncident)
	if err != nil {
		return CreateResult{}, err
	}

	now := s.now()

	detectedAt := now
	if input.DetectedAt != "" {
		parsed, err := time.Parse(time.RFC3339, input.DetectedAt)
		if err != nil {
			return CreateResult{}, fmt.Errorf("invalid detected_at: %w", err)
		}
		detectedAt = parsed
	}

	entity := model.Incident{
		ID:         idValue,
		Slug:       normalizeSlug(input.Slug),
		Name:       incidentName,
		Status:     model.IncidentStatusReported,
		Severity:   model.IncidentSeverity(input.Severity),
		ReportedBy: strings.TrimSpace(input.ReportedBy),
		DetectedAt: detectedAt,
		Summary:    strings.TrimSpace(input.Summary),
		Created:    now,
		CreatedBy:  strings.TrimSpace(input.ReportedBy),
		Updated:    now,
	}

	if err := validate.ValidateInitialState(validate.EntityIncident, string(entity.Status)); err != nil {
		return CreateResult{}, err
	}

	result, err := s.write(entity)
	if err != nil {
		return result, err
	}
	s.cacheUpsertFromResult(result)
	return result, nil
}

// UpdateIncident updates fields of an existing incident.
// Status changes go through lifecycle validation.
// Field updates (severity, summary, timestamps, affected_features) are applied directly.
func (s *EntityService) UpdateIncident(input UpdateIncidentInput) (GetResult, error) {
	entityType := string(model.EntityKindIncident)
	entityID := strings.TrimSpace(input.ID)
	slug := normalizeSlug(input.Slug)

	if err := validateRequired(field("id", entityID)); err != nil {
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

	// Handle status transition if requested
	if input.Status != "" {
		currentStatus, ok := record.Fields["status"]
		if !ok {
			return GetResult{}, fmt.Errorf("incident %s has no status field", entityID)
		}
		currentStatusText := strings.TrimSpace(fmt.Sprint(currentStatus))
		nextStatus := strings.TrimSpace(input.Status)

		if err := validate.ValidateTransition(validate.EntityIncident, currentStatusText, nextStatus); err != nil {
			return GetResult{}, fmt.Errorf("%s: %w", err.Error(), ErrInvalidTransition)
		}
		record.Fields["status"] = nextStatus
	}

	// Apply optional field updates
	if input.Severity != "" {
		if err := validate.ValidateIncidentSeverity(input.Severity); err != nil {
			return GetResult{}, err
		}
		record.Fields["severity"] = input.Severity
	}

	if input.Summary != "" {
		record.Fields["summary"] = strings.TrimSpace(input.Summary)
	}

	if input.TriagedAt != "" {
		if _, err := time.Parse(time.RFC3339, input.TriagedAt); err != nil {
			return GetResult{}, fmt.Errorf("invalid triaged_at: %w", err)
		}
		record.Fields["triaged_at"] = input.TriagedAt
	}

	if input.MitigatedAt != "" {
		if _, err := time.Parse(time.RFC3339, input.MitigatedAt); err != nil {
			return GetResult{}, fmt.Errorf("invalid mitigated_at: %w", err)
		}
		record.Fields["mitigated_at"] = input.MitigatedAt
	}

	if input.ResolvedAt != "" {
		if _, err := time.Parse(time.RFC3339, input.ResolvedAt); err != nil {
			return GetResult{}, fmt.Errorf("invalid resolved_at: %w", err)
		}
		record.Fields["resolved_at"] = input.ResolvedAt
	}

	if input.AffectedFeatures != nil {
		if len(input.AffectedFeatures) > 0 {
			record.Fields["affected_features"] = append([]string(nil), input.AffectedFeatures...)
		} else {
			delete(record.Fields, "affected_features")
		}
	}

	record.Fields["updated"] = s.now().Format(time.RFC3339)

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

// ListIncidents lists incidents with optional status and severity filters.
func (s *EntityService) ListIncidents(statusFilter, severityFilter string) ([]ListResult, error) {
	entityType := string(model.EntityKindIncident)
	dir := filepath.Join(s.root, entityDirectory(entityType))

	entries, err := filepath.Glob(filepath.Join(dir, "*.yaml"))
	if err != nil {
		return nil, fmt.Errorf("list incidents: %w", err)
	}
	sort.Strings(entries)

	var results []ListResult
	for _, path := range entries {
		record, err := s.loadRecordFromPath(entityType, path)
		if err != nil {
			continue
		}

		if statusFilter != "" {
			if status, ok := record.Fields["status"]; ok {
				if fmt.Sprint(status) != statusFilter {
					continue
				}
			}
		}

		if severityFilter != "" {
			if severity, ok := record.Fields["severity"]; ok {
				if fmt.Sprint(severity) != severityFilter {
					continue
				}
			}
		}

		results = append(results, ListResult{
			Type:  record.Type,
			ID:    record.ID,
			Slug:  record.Slug,
			Path:  path,
			State: record.Fields,
		})
	}

	return results, nil
}

// LinkBug adds a bug ID to an incident's linked_bugs list. Idempotent.
func (s *EntityService) LinkBug(input LinkBugInput) (GetResult, error) {
	entityType := string(model.EntityKindIncident)
	incidentID := strings.TrimSpace(input.IncidentID)
	bugID := strings.TrimSpace(input.BugID)

	if err := validateRequired(
		field("incident_id", incidentID),
		field("bug_id", bugID),
	); err != nil {
		return GetResult{}, err
	}

	// Verify the bug exists
	bugType := string(model.EntityKindBug)
	_, _, err := s.ResolvePrefix(bugType, bugID)
	if err != nil {
		return GetResult{}, fmt.Errorf("bug %s not found: %w", bugID, err)
	}

	// Resolve incident
	resolvedID, slug, err := s.ResolvePrefix(entityType, incidentID)
	if err != nil {
		return GetResult{}, err
	}

	record, err := s.store.Load(entityType, resolvedID, slug)
	if err != nil {
		return GetResult{}, err
	}

	// Extract current linked_bugs
	var linkedBugs []string
	if existing, ok := record.Fields["linked_bugs"]; ok {
		switch v := existing.(type) {
		case []string:
			linkedBugs = v
		case []any:
			for _, item := range v {
				if s, ok := item.(string); ok {
					linkedBugs = append(linkedBugs, s)
				}
			}
		}
	}

	// Check for idempotency
	for _, existingBug := range linkedBugs {
		if existingBug == bugID {
			// Already linked, return current state
			return GetResult{
				Type:  record.Type,
				ID:    record.ID,
				Slug:  record.Slug,
				State: record.Fields,
			}, nil
		}
	}

	linkedBugs = append(linkedBugs, bugID)
	record.Fields["linked_bugs"] = linkedBugs
	record.Fields["updated"] = s.now().Format(time.RFC3339)

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
