package service

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/sambeau/kanbanzai/internal/model"
	"github.com/sambeau/kanbanzai/internal/storage"
)

// EstimationScale is the Modified Fibonacci sequence used for story points.
var EstimationScale = []float64{0, 0.5, 1, 2, 3, 5, 8, 13, 20, 40, 100}

// EstimationScaleMeanings maps scale points to their human meanings.
var EstimationScaleMeanings = map[float64]string{
	0:   "No effort required",
	0.5: "Minimal; trivial change",
	1:   "Simple, well-understood; likely done in one day",
	2:   "Requires some thought; routine work",
	3:   "Well-understood with a few extra steps",
	5:   "Complex or infrequent; may need collaboration",
	8:   "Requires research; likely multiple contributors",
	13:  "Highly complex with many unknowns",
	20:  "Roughly one month of work",
	40:  "Roughly two months of work",
	100: "Roughly five months of work",
}

// SoftLimits returns the soft limit for the entity type.
var SoftLimits = map[string]float64{
	"task":    13,
	"bug":     13,
	"feature": 100,
}

// IsValidEstimate returns true if the estimate is in the Modified Fibonacci scale.
func IsValidEstimate(estimate float64) bool {
	for _, v := range EstimationScale {
		if v == estimate {
			return true
		}
	}
	return false
}

// ValidateEstimate returns an error if the estimate is not in the scale.
func ValidateEstimate(estimate float64) error {
	if !IsValidEstimate(estimate) {
		return fmt.Errorf("invalid estimate %.1f: must be one of %v", estimate, EstimationScale)
	}
	return nil
}

// SoftLimitWarning returns a warning string if the estimate exceeds the soft limit for the entity type.
// Returns empty string if within limit.
func SoftLimitWarning(entityType string, estimate float64) string {
	limit, ok := SoftLimits[strings.ToLower(entityType)]
	if !ok {
		return ""
	}
	if estimate > limit {
		return fmt.Sprintf("estimate %.0f exceeds the soft limit of %.0f for %s — consider decomposing into smaller units", estimate, limit, entityType)
	}
	return ""
}

// FeatureRollup holds computed rollup values for a Feature.
type FeatureRollup struct {
	TaskTotal          *float64 // sum of included task estimates (nil if no estimated tasks)
	Progress           float64  // sum of done task estimates
	Delta              *float64 // TaskTotal - Feature.estimate (nil if either absent)
	TaskCount          int
	EstimatedTaskCount int
	ExcludedTaskCount  int // not-planned or duplicate
}

// BatchRollup holds computed rollup values for a Batch.
type BatchRollup struct {
	FeatureTotal          *float64
	Progress              float64
	Delta                 *float64
	FeatureCount          int
	EstimatedFeatureCount int
}

// maxPlanRollupDepth is the maximum recursion depth for ComputePlanRollup.
// When exceeded, the function returns an error rather than looping indefinitely.
// This is a defensive measure; the plan tree is guaranteed acyclic by F2's
// no-cycle enforcement rule.
const maxPlanRollupDepth = 50

// PlanRollup holds computed recursive rollup values for a StrategicPlan.
type PlanRollup struct {
	Total      *float64 // recursive sum of all child batch and child plan totals; nil when no child carries an estimate
	Progress   float64  // recursive sum of progress from child batches and child plans
	BatchCount int      // number of direct child batches
	PlanCount  int      // number of direct child plans
}

// SetEstimate loads an entity, sets its estimate, and saves it.
// Returns the entity's state, a soft limit warning (may be empty), and any error.
func (s *EntityService) SetEstimate(entityType, entityID string, estimate float64) (map[string]any, string, error) {
	entityType = strings.ToLower(strings.TrimSpace(entityType))
	entityID = strings.TrimSpace(entityID)

	if err := ValidateEstimate(estimate); err != nil {
		return nil, "", err
	}

	// Resolve ID
	resolvedID, resolvedSlug, err := s.ResolvePrefix(entityType, entityID)
	if err != nil {
		return nil, "", err
	}

	// Load record
	record, err := s.store.Load(entityType, resolvedID, resolvedSlug)
	if err != nil {
		return nil, "", err
	}

	// Set estimate
	record.Fields["estimate"] = estimate

	// Write
	if _, err := s.store.Write(record); err != nil {
		return nil, "", fmt.Errorf("write entity: %w", err)
	}

	warning := SoftLimitWarning(entityType, estimate)
	return record.Fields, warning, nil
}

// GetEstimateFromFields returns the estimate for an entity, or nil if not set.
// Handles float64, int, and string representations (from YAML round-trips).
func GetEstimateFromFields(fields map[string]any) *float64 {
	v, ok := fields["estimate"]
	if !ok || v == nil {
		return nil
	}
	switch typed := v.(type) {
	case float64:
		return &typed
	case int:
		f := float64(typed)
		return &f
	case string:
		if f, err := strconv.ParseFloat(typed, 64); err == nil {
			return &f
		}
	}
	return nil
}

// ComputeFeatureRollup computes rollup statistics for a Feature.
// Tasks are loaded from the entity service.
func (s *EntityService) ComputeFeatureRollup(featureID string) (FeatureRollup, error) {
	// Load all tasks for this feature
	allTasks, err := s.List("task")
	if err != nil {
		return FeatureRollup{}, fmt.Errorf("list tasks: %w", err)
	}

	var rollup FeatureRollup
	var taskTotal float64
	hasEstimatedTasks := false

	for _, t := range allTasks {
		pf, _ := t.State["parent_feature"].(string)
		if pf != featureID {
			continue
		}

		status, _ := t.State["status"].(string)

		// Excluded states
		if status == string(model.TaskStatusNotPlanned) || status == string(model.TaskStatusDuplicate) {
			rollup.ExcludedTaskCount++
			continue
		}

		rollup.TaskCount++

		est := GetEstimateFromFields(t.State)
		if est != nil {
			rollup.EstimatedTaskCount++
			taskTotal += *est
			hasEstimatedTasks = true

			// Progress: done tasks
			if status == string(model.TaskStatusDone) {
				rollup.Progress += *est
			}
		}
	}

	if hasEstimatedTasks {
		rollup.TaskTotal = &taskTotal
	}

	return rollup, nil
}

// ComputeBatchRollup computes rollup statistics for a Batch.
func (s *EntityService) ComputeBatchRollup(batchID string) (BatchRollup, error) {
	// Load all features for this batch
	allFeatures, err := s.List("feature")
	if err != nil {
		return BatchRollup{}, fmt.Errorf("list features: %w", err)
	}

	var rollup BatchRollup
	var featureTotal float64
	hasEstimatedFeatures := false

	for _, f := range allFeatures {
		parent, _ := f.State["parent"].(string)
		if parent != batchID {
			continue
		}

		rollup.FeatureCount++

		// Feature effective estimate: task total if available, else own estimate
		featureRollup, err := s.ComputeFeatureRollup(f.ID)
		if err != nil {
			continue
		}

		var effectiveEstimate *float64
		if featureRollup.TaskTotal != nil {
			effectiveEstimate = featureRollup.TaskTotal
		} else {
			effectiveEstimate = GetEstimateFromFields(f.State)
		}

		if effectiveEstimate != nil {
			rollup.EstimatedFeatureCount++
			featureTotal += *effectiveEstimate
			hasEstimatedFeatures = true
		}

		rollup.Progress += featureRollup.Progress
	}

	if hasEstimatedFeatures {
		rollup.FeatureTotal = &featureTotal
	}

	return rollup, nil
}

// ComputePlanRollup computes recursive rollup statistics for a StrategicPlan.
// It aggregates progress across direct child batches (via ComputeBatchRollup)
// and direct child plans (via recursive self-invocation).
// A depth guard is included as a defensive measure against corrupt states;
// cycle-freedom is guaranteed by F2's no-cycle enforcement rule.
func (s *EntityService) ComputePlanRollup(planID string) (PlanRollup, error) {
	return s.computePlanRollupRecursive(planID, 0)
}

// computePlanRollupRecursive is the internal recursive implementation.
// depth tracks the current recursion level for the depth guard.
func (s *EntityService) computePlanRollupRecursive(planID string, depth int) (PlanRollup, error) {
	if depth > maxPlanRollupDepth {
		return PlanRollup{}, fmt.Errorf("plan rollup depth limit exceeded at plan %s", planID)
	}

	var rollup PlanRollup
	var total float64
	hasEstimate := false

	// Aggregate direct child batches.
	childBatches, err := s.ListBatches(BatchFilters{Parent: planID})
	if err != nil {
		return PlanRollup{}, fmt.Errorf("list child batches for plan %s: %w", planID, err)
	}
	for _, b := range childBatches {
		rollup.BatchCount++
		batchRollup, err := s.ComputeBatchRollup(b.ID)
		if err != nil {
			continue
		}
		if batchRollup.FeatureTotal != nil {
			total += *batchRollup.FeatureTotal
			hasEstimate = true
		}
		rollup.Progress += batchRollup.Progress
	}

	// Aggregate direct child plans recursively.
	childPlans, err := s.ListStrategicPlans(StrategicPlanFilters{Parent: planID})
	if err != nil {
		return PlanRollup{}, fmt.Errorf("list child plans for plan %s: %w", planID, err)
	}
	for _, p := range childPlans {
		rollup.PlanCount++
		childRollup, err := s.computePlanRollupRecursive(p.ID, depth+1)
		if err != nil {
			return PlanRollup{}, err
		}
		if childRollup.Total != nil {
			total += *childRollup.Total
			hasEstimate = true
		}
		rollup.Progress += childRollup.Progress
	}

	if hasEstimate {
		rollup.Total = &total
	}

	return rollup, nil
}

// GetEstimationReferences returns all knowledge entries tagged "estimation-reference".
func (s *KnowledgeService) GetEstimationReferences() ([]storage.KnowledgeRecord, error) {
	return s.List(KnowledgeFilters{
		Tags:           []string{"estimation-reference"},
		IncludeRetired: false,
	})
}

// AddEstimationReference adds a calibration reference example for an entity.
// Returns the created knowledge record.
func (s *KnowledgeService) AddEstimationReference(entityID, content, createdBy string) (storage.KnowledgeRecord, error) {
	topic := "estimation-ref-" + entityID

	// Contribute as Tier 2 knowledge with ttl_days=0 (exempt from TTL pruning)
	input := ContributeInput{
		Topic:       topic,
		Content:     content,
		Scope:       "project",
		Tier:        2,
		LearnedFrom: entityID,
		CreatedBy:   createdBy,
		Tags:        []string{"estimation-reference"},
	}

	record, _, err := s.Contribute(input)
	if err != nil {
		return storage.KnowledgeRecord{}, err
	}

	// Override ttl_days to 0 (exempt from TTL pruning)
	record.Fields["ttl_days"] = 0
	if _, err := s.store.Write(record); err != nil {
		return storage.KnowledgeRecord{}, fmt.Errorf("update ttl_days: %w", err)
	}

	return record, nil
}

// RemoveEstimationReference retires the reference knowledge entry for an entity.
func (s *KnowledgeService) RemoveEstimationReference(entityID string) (string, error) {
	topic := "estimation-ref-" + entityID

	all, err := s.List(KnowledgeFilters{
		Tags:           []string{"estimation-reference"},
		IncludeRetired: false,
	})
	if err != nil {
		return "", fmt.Errorf("list estimation references: %w", err)
	}

	for _, rec := range all {
		recTopic, _ := rec.Fields["topic"].(string)
		if recTopic == topic {
			_, err := s.Retire(rec.ID, "removed via estimate_reference_remove")
			if err != nil {
				return "", err
			}
			return rec.ID, nil
		}
	}

	return "", fmt.Errorf("no estimation reference found for entity %s", entityID)
}

// ScaleEntry is a single entry in the estimation scale.
type ScaleEntry struct {
	Points  float64 `json:"points"`
	Meaning string  `json:"meaning"`
}

// GetScaleEntries returns all scale entries.
func GetScaleEntries() []ScaleEntry {
	entries := make([]ScaleEntry, len(EstimationScale))
	for i, pts := range EstimationScale {
		entries[i] = ScaleEntry{
			Points:  pts,
			Meaning: EstimationScaleMeanings[pts],
		}
	}
	return entries
}
