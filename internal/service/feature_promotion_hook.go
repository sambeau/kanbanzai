package service

import (
	"log"
	"strings"
)

// FeaturePromotionHook implements StatusTransitionHook to automatically
// promote queued tasks to ready when a feature transitions to "developing".
//
// Per spec FR-003: hook failures are logged and never propagate — the
// feature transition has already been persisted when the hook fires.
type FeaturePromotionHook struct {
	svc *EntityService
}

// NewFeaturePromotionHook creates a hook that calls PromoteQueuedTasks when a
// feature transitions to developing.
func NewFeaturePromotionHook(svc *EntityService) *FeaturePromotionHook {
	return &FeaturePromotionHook{svc: svc}
}

// OnStatusTransition fires PromoteQueuedTasks when a feature transitions to
// "developing". All other transitions are ignored (FR-002).
func (h *FeaturePromotionHook) OnStatusTransition(entityType, entityID, slug, fromStatus, toStatus string, state map[string]any) *WorktreeResult {
	if !strings.EqualFold(entityType, "feature") || toStatus != "developing" {
		return nil
	}
	if err := h.svc.PromoteQueuedTasks(entityID); err != nil {
		log.Printf("feature promotion hook: PromoteQueuedTasks(%s) failed: %v", entityID, err)
	}
	return nil
}
