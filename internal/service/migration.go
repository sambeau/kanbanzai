package service

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sambeau/kanbanzai/internal/config"
	"github.com/sambeau/kanbanzai/internal/model"
	"github.com/sambeau/kanbanzai/internal/storage"
)

// MigrationResult summarises the outcome of a Phase 2 migration.
type MigrationResult struct {
	PlansCreated    int
	FeaturesUpdated int
	FilesMoved      int
	DirsCreated     int
	Errors          []string
}

// MigratePhase2 converts Phase 1 epic entities to Phase 2 plan entities and
// updates feature references accordingly. The migration is idempotent: if
// plan files already exist the corresponding epic is skipped.
func (s *EntityService) MigratePhase2() (*MigrationResult, error) {
	result := &MigrationResult{}

	// Step 1: Load config and validate prefix registry.
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("prefix registry must be configured before migration: %w", err)
	}
	activePrefixes := cfg.ActivePrefixes()
	if len(activePrefixes) == 0 {
		return nil, fmt.Errorf("prefix registry must be configured before migration")
	}
	prefix := activePrefixes[0].Prefix

	// Step 2: Check if epics directory exists. If not, migration is a no-op.
	epicsDir := filepath.Join(s.root, "epics")
	if _, err := os.Stat(epicsDir); os.IsNotExist(err) {
		return result, nil
	}

	// Step 3: Create target directories if they don't exist.
	for _, dir := range []string{"plans", "documents", "index"} {
		target := filepath.Join(s.root, dir)
		if _, err := os.Stat(target); os.IsNotExist(err) {
			if err := os.MkdirAll(target, 0o755); err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("create directory %s: %v", dir, err))
				continue
			}
			result.DirsCreated++
		}
	}
	// Also create the index directory at the instance root level (.kbz/index).
	indexDir := filepath.Join(filepath.Dir(s.root), "index")
	if _, err := os.Stat(indexDir); os.IsNotExist(err) {
		if err := os.MkdirAll(indexDir, 0o755); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("create index directory: %v", err))
		} else {
			result.DirsCreated++
		}
	}

	// Step 4: Migrate each epic to a plan.
	epicEntries, err := os.ReadDir(epicsDir)
	if err != nil {
		return result, fmt.Errorf("read epics directory: %w", err)
	}

	// Track old epic ID → new plan ID for feature updates.
	idMapping := make(map[string]string)
	seqNum := 0

	// Determine starting sequence number by scanning existing plans.
	existingIDs, _ := s.listPlanIDs()
	maxNum := 0
	for _, pid := range existingIDs {
		p, numStr, _ := model.ParsePlanID(pid)
		if p != prefix {
			continue
		}
		var n int
		if _, err := fmt.Sscanf(numStr, "%d", &n); err == nil && n > maxNum {
			maxNum = n
		}
	}
	seqNum = maxNum

	// Build a map of slug → plan ID for existing plans (for idempotency checks).
	existingSlugToID := make(map[string]string)
	plansDir := filepath.Join(s.root, "plans")
	if planEntries, err := os.ReadDir(plansDir); err == nil {
		for _, pe := range planEntries {
			name := pe.Name()
			if pe.IsDir() || !strings.HasSuffix(name, ".yaml") {
				continue
			}
			// Plan filenames are "{prefix}{num}-{slug}.yaml". Extract the slug
			// by finding the first hyphen after the prefix+number portion.
			base := strings.TrimSuffix(name, ".yaml")
			if idx := strings.Index(base, "-"); idx >= 0 {
				slug := base[idx+1:]
				existingSlugToID[slug] = base
			}
		}
	}

	for _, entry := range epicEntries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}

		epicPath := filepath.Join(epicsDir, entry.Name())
		data, err := os.ReadFile(epicPath)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("read epic %s: %v", entry.Name(), err))
			continue
		}

		fields, err := storage.UnmarshalCanonicalYAML(string(data))
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("parse epic %s: %v", entry.Name(), err))
			continue
		}

		epicID := stringFromState(fields, "id")
		epicSlug := stringFromState(fields, "slug")
		if epicID == "" || epicSlug == "" {
			result.Errors = append(result.Errors, fmt.Sprintf("epic %s missing id or slug", entry.Name()))
			continue
		}

		// Check idempotency: if a plan with this slug already exists, skip.
		if existingPlanID, ok := existingSlugToID[epicSlug]; ok {
			idMapping[epicID] = existingPlanID
			continue
		}

		// Assign new Plan ID.
		seqNum++
		newID := fmt.Sprintf("%s%d-%s", prefix, seqNum, epicSlug)

		// Transform fields: epic → plan.
		fields["id"] = newID
		fields["type"] = string(model.EntityKindPlan)

		// Map epic status to plan status.
		epicStatus := stringFromState(fields, "status")
		fields["status"] = mapEpicStatusToPlan(epicStatus)

		// Remove epic-only fields.
		delete(fields, "features")

		// Add updated timestamp if not present.
		if _, ok := fields["updated"]; !ok {
			if created := stringFromState(fields, "created"); created != "" {
				fields["updated"] = created
			}
		}

		// Write the plan record.
		record := storage.EntityRecord{
			Type:   string(model.EntityKindPlan),
			ID:     newID,
			Slug:   epicSlug,
			Fields: fields,
		}

		if _, err := s.store.Write(record); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("write plan for epic %s: %v", epicID, err))
			continue
		}

		idMapping[epicID] = newID
		result.PlansCreated++

		// Delete the old epic file.
		if err := os.Remove(epicPath); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("remove epic file %s: %v", entry.Name(), err))
		} else {
			result.FilesMoved++
		}
	}

	// Step 5: Update features to reference the new plan IDs.
	featuresDir := filepath.Join(s.root, "features")
	featureEntries, err := os.ReadDir(featuresDir)
	if err != nil && !os.IsNotExist(err) {
		result.Errors = append(result.Errors, fmt.Sprintf("read features directory: %v", err))
		return result, nil
	}

	for _, entry := range featureEntries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}

		featurePath := filepath.Join(featuresDir, entry.Name())
		data, err := os.ReadFile(featurePath)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("read feature %s: %v", entry.Name(), err))
			continue
		}

		fields, err := storage.UnmarshalCanonicalYAML(string(data))
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("parse feature %s: %v", entry.Name(), err))
			continue
		}

		modified := false

		// Rename "epic" → "parent" and update the value.
		if epicRef, ok := fields["epic"]; ok {
			epicRefStr, _ := epicRef.(string)
			if epicRefStr != "" {
				newPlanID, mapped := idMapping[epicRefStr]
				if mapped {
					fields["parent"] = newPlanID
				} else {
					fields["parent"] = epicRefStr
				}
			}
			delete(fields, "epic")
			modified = true
		}

		// Rename "plan" → "dev_plan" (these are different fields in Phase 2).
		if planRef, ok := fields["plan"]; ok {
			fields["dev_plan"] = planRef
			delete(fields, "plan")
			modified = true
		}

		if !modified {
			continue
		}

		// Re-derive ID and slug from the filename for the write.
		featureID := stringFromState(fields, "id")
		featureSlug := stringFromState(fields, "slug")
		if featureID == "" || featureSlug == "" {
			result.Errors = append(result.Errors, fmt.Sprintf("feature %s missing id or slug", entry.Name()))
			continue
		}

		record := storage.EntityRecord{
			Type:   string(model.EntityKindFeature),
			ID:     featureID,
			Slug:   featureSlug,
			Fields: fields,
		}

		if _, err := s.store.Write(record); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("write feature %s: %v", featureID, err))
			continue
		}

		result.FeaturesUpdated++
	}

	// Step 6: Remove epics directory if empty.
	remaining, _ := os.ReadDir(epicsDir)
	if len(remaining) == 0 {
		_ = os.Remove(epicsDir)
	}

	return result, nil
}

// mapEpicStatusToPlan maps a Phase 1 EpicStatus to the closest Phase 2 PlanStatus.
func mapEpicStatusToPlan(epicStatus string) string {
	switch model.EpicStatus(epicStatus) {
	case model.EpicStatusProposed:
		return string(model.PlanStatusProposed)
	case model.EpicStatusApproved:
		return string(model.PlanStatusDesigning)
	case model.EpicStatusActive:
		return string(model.PlanStatusActive)
	case model.EpicStatusOnHold:
		return string(model.PlanStatusActive)
	case model.EpicStatusDone:
		return string(model.PlanStatusDone)
	default:
		return string(model.PlanStatusProposed)
	}
}
