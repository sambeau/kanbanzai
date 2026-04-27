package service

import (
	"fmt"
	"sort"
	"time"

	"github.com/sambeau/kanbanzai/internal/model"
	"github.com/sambeau/kanbanzai/internal/storage"
)

// MigrateDisplayIDs assigns display_id values to all existing features that do
// not have one (REQ-014). Within each plan, features are assigned sequence
// numbers in ascending order of their created timestamp (REQ-015). After
// backfilling all features under a plan the plan's next_feature_seq is set to
// max_seq+1. Returns an error if any write fails; already-backfilled features
// are skipped (idempotent).
func MigrateDisplayIDs(svc *EntityService) error {
	plans, err := svc.ListPlans(PlanFilters{})
	if err != nil {
		return fmt.Errorf("migrate display_ids: list plans: %w", err)
	}

	allFeatures, err := svc.List("feature")
	if err != nil {
		return fmt.Errorf("migrate display_ids: list features: %w", err)
	}

	// Group features without a display_id by parent plan.
	type featureEntry struct {
		id      string
		slug    string
		created time.Time
		state   map[string]any
	}
	byPlan := make(map[string][]featureEntry, len(plans))
	for _, f := range allFeatures {
		parent, _ := f.State["parent"].(string)
		if parent == "" {
			continue
		}
		if did, _ := f.State["display_id"].(string); did != "" {
			continue // already has a display_id
		}
		createdStr, _ := f.State["created"].(string)
		created, _ := time.Parse(time.RFC3339, createdStr)
		byPlan[parent] = append(byPlan[parent], featureEntry{
			id: f.ID, slug: f.Slug, created: created, state: f.State,
		})
	}

	for _, planResult := range plans {
		planID := planResult.ID
		_, planNum, _ := model.ParsePlanID(planID)
		if planNum == "" {
			continue
		}
		toBackfill := byPlan[planID]
		if len(toBackfill) == 0 {
			continue
		}

		// Sort by created timestamp ascending (oldest gets lowest seq number).
		sort.Slice(toBackfill, func(i, j int) bool {
			return toBackfill[i].created.Before(toBackfill[j].created)
		})

		seq := intFromState(planResult.State, "next_feature_seq", 1)
		for _, fe := range toBackfill {
			fe.state["display_id"] = fmt.Sprintf("P%s-F%d", planNum, seq)
			record := storage.EntityRecord{
				Type:   "feature",
				ID:     fe.id,
				Slug:   fe.slug,
				Fields: fe.state,
			}
			if _, err := svc.store.Write(record); err != nil {
				return fmt.Errorf("migrate display_ids: write feature %s: %w", fe.id, err)
			}
			seq++
		}

		// Write plan with updated counter.
		planResult.State["next_feature_seq"] = seq
		planRecord := storage.EntityRecord{
			Type:   string(model.EntityKindPlan),
			ID:     planResult.ID,
			Slug:   planResult.Slug,
			Fields: planResult.State,
		}
		if _, err := svc.store.Write(planRecord); err != nil {
			return fmt.Errorf("migrate display_ids: write plan %s: %w", planID, err)
		}
	}

	return nil
}
