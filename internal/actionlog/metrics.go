package actionlog

import (
	"sort"
	"time"
)

// ─── Input / output types ────────────────────────────────────────────────────

// MetricsInput specifies the time range and optional feature filter for metrics.
type MetricsInput struct {
	LogsDir   string
	Since     time.Time
	Until     time.Time
	FeatureID string // optional; empty means all features
}

// MetricsResult holds the computed metrics output.
type MetricsResult struct {
	TimePerStage         []StageDuration     `json:"time_per_stage"`
	RevisionCycleCounts  []FeatureCycleCount `json:"revision_cycle_counts,omitempty"`
	GateFailureRate      GateFailureMetric   `json:"gate_failure_rate"`
	StructuralCheckRate  *PassRateMetric     `json:"structural_check_pass_rate,omitempty"`
	ToolSubsetCompliance *ComplianceMetric   `json:"tool_subset_compliance,omitempty"`
}

// StageDuration holds median and p90 dwell time for a lifecycle stage.
type StageDuration struct {
	Stage  string  `json:"stage"`
	Median float64 `json:"median_hours"`
	P90    float64 `json:"p90_hours"`
	Count  int     `json:"count"`
}

// FeatureCycleCount holds the number of review cycles for a feature.
type FeatureCycleCount struct {
	FeatureID    string `json:"feature_id"`
	DisplayID    string `json:"display_id"`
	ReviewCycles int    `json:"review_cycles"`
}

// GateFailureMetric summarises gate failure rate from log entries.
type GateFailureMetric struct {
	Count int     `json:"count"`
	Total int     `json:"total"`
	Rate  float64 `json:"rate"`
}

// PassRateMetric holds a pass/fail ratio for a check.
type PassRateMetric struct {
	Passed int     `json:"passed"`
	Total  int     `json:"total"`
	Rate   float64 `json:"rate"`
}

// ComplianceMetric holds a compliance ratio.
type ComplianceMetric struct {
	Compliant int     `json:"compliant"`
	Total     int     `json:"total"`
	Rate      float64 `json:"rate"`
}

// FeatureMetricsData carries entity data for metrics computation.
type FeatureMetricsData struct {
	FeatureID    string
	DisplayID    string
	ReviewCycles int
	Transitions  []StatusTransition
}

// StatusTransition records when an entity moved from one status to another.
type StatusTransition struct {
	FromStatus string
	ToStatus   string
	At         time.Time
}

// ─── Lookup interface ────────────────────────────────────────────────────────

// StageFeatureLookup loads feature metrics data from the entity store.
type StageFeatureLookup interface {
	ListFeaturesInRange(since, until time.Time, featureID string) ([]FeatureMetricsData, error)
}

// ─── Computation ─────────────────────────────────────────────────────────────

// ComputeMetrics aggregates log entries and entity data into a MetricsResult.
func ComputeMetrics(input MetricsInput, lookup StageFeatureLookup) (*MetricsResult, error) {
	entries, err := ReadEntries(input.LogsDir, input.Since, input.Until)
	if err != nil {
		return nil, err
	}

	result := &MetricsResult{
		GateFailureRate: computeGateFailureRate(entries),
	}

	if lookup != nil {
		features, err := lookup.ListFeaturesInRange(input.Since, input.Until, input.FeatureID)
		if err != nil {
			return nil, err
		}

		result.TimePerStage = computeTimePerStage(features)
		result.RevisionCycleCounts = computeRevisionCycles(features)
	}

	return result, nil
}

// computeGateFailureRate counts gate_failure errors vs total tool calls.
func computeGateFailureRate(entries []Entry) GateFailureMetric {
	var failures, total int
	for _, e := range entries {
		total++
		if e.ErrorType != nil && *e.ErrorType == ErrorGateFailure {
			failures++
		}
	}
	rate := 0.0
	if total > 0 {
		rate = float64(failures) / float64(total)
	}
	return GateFailureMetric{Count: failures, Total: total, Rate: rate}
}

// computeTimePerStage calculates median and p90 dwell times per lifecycle stage.
func computeTimePerStage(features []FeatureMetricsData) []StageDuration {
	// Collect dwell times per stage from transition pairs.
	stageDurations := map[string][]float64{}

	for _, f := range features {
		transitions := f.Transitions
		sort.Slice(transitions, func(i, j int) bool {
			return transitions[i].At.Before(transitions[j].At)
		})
		for i := 0; i < len(transitions)-1; i++ {
			stage := transitions[i].ToStatus
			duration := transitions[i+1].At.Sub(transitions[i].At).Hours()
			if duration >= 0 {
				stageDurations[stage] = append(stageDurations[stage], duration)
			}
		}
	}

	stages := make([]StageDuration, 0, len(stageDurations))
	for stage, durations := range stageDurations {
		sort.Float64s(durations)
		stages = append(stages, StageDuration{
			Stage:  stage,
			Median: percentile(durations, 50),
			P90:    percentile(durations, 90),
			Count:  len(durations),
		})
	}

	sort.Slice(stages, func(i, j int) bool {
		return stages[i].Stage < stages[j].Stage
	})
	return stages
}

// computeRevisionCycles collects review cycle counts for features.
func computeRevisionCycles(features []FeatureMetricsData) []FeatureCycleCount {
	var result []FeatureCycleCount
	for _, f := range features {
		if f.ReviewCycles > 0 {
			result = append(result, FeatureCycleCount{
				FeatureID:    f.FeatureID,
				DisplayID:    f.DisplayID,
				ReviewCycles: f.ReviewCycles,
			})
		}
	}
	return result
}

// percentile returns the p-th percentile from a pre-sorted slice.
func percentile(sorted []float64, p float64) float64 {
	if len(sorted) == 0 {
		return 0
	}
	idx := (p / 100.0) * float64(len(sorted)-1)
	lo := int(idx)
	hi := lo + 1
	if hi >= len(sorted) {
		return sorted[len(sorted)-1]
	}
	frac := idx - float64(lo)
	return sorted[lo] + frac*(sorted[hi]-sorted[lo])
}
