package actionlog

import (
	"cmp"
	"slices"
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
	ActionDistribution   []ToolActionCount   `json:"action_distribution,omitempty"`
	DocTypeFunnel        *DocFunnelMetric    `json:"doc_type_funnel,omitempty"`
	TaskCompletionGap    *TaskGapMetric      `json:"task_completion_gap,omitempty"`
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

// ToolActionCount records the number of times a specific tool-action pair was invoked.
type ToolActionCount struct {
	Tool     string `json:"tool"`
	Action   string `json:"action"`
	Count    int    `json:"calls"`
	Failures int    `json:"failures"`
}

// DocFunnelRecord holds register vs approve counts and approval rate for a document type.
type DocFunnelRecord struct {
	DocType    string  `json:"doc_type"`
	Registered int     `json:"registered"`
	Approved   int     `json:"approved"`
	Rate       float64 `json:"rate"`
}

// DocFunnelMetric summarises document register-to-approve throughput.
type DocFunnelMetric struct {
	Records []DocFunnelRecord `json:"records"`
}

// TaskGapMetric holds paired next-to-finish gap statistics.
type TaskGapMetric struct {
	Count  int     `json:"count"`
	Median float64 `json:"median_hours"`
	P90    float64 `json:"p90_hours"`
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
	DocType(entityID string) (string, error)
}

// ─── Computation ─────────────────────────────────────────────────────────────

// ComputeMetrics aggregates log entries and entity data into a MetricsResult.
func ComputeMetrics(input MetricsInput, lookup StageFeatureLookup) (*MetricsResult, error) {
	entries, err := ReadEntries(input.LogsDir, input.Since, input.Until)
	if err != nil {
		return nil, err
	}

	result := &MetricsResult{
		GateFailureRate:    computeGateFailureRate(entries),
		ActionDistribution: computeActionDistribution(entries),
	}

	if lookup != nil {
		features, err := lookup.ListFeaturesInRange(input.Since, input.Until, input.FeatureID)
		if err != nil {
			return nil, err
		}

		result.TimePerStage = computeTimePerStage(features)
		result.RevisionCycleCounts = computeRevisionCycles(features)

		// DocTypeFunnel: collect entity IDs from doc tool entries during the
		// single entry pass, then batch-resolve document types.
		result.DocTypeFunnel = computeDocTypeFunnel(entries, lookup)

		// TaskCompletionGap: paired next→finish gap calculation.
		result.TaskCompletionGap = computeTaskCompletionGap(entries)
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
		slices.SortFunc(transitions, func(a, b StatusTransition) int {
			return a.At.Compare(b.At)
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
		slices.Sort(durations)
		stages = append(stages, StageDuration{
			Stage:  stage,
			Median: percentile(durations, 50),
			P90:    percentile(durations, 90),
			Count:  len(durations),
		})
	}

	slices.SortFunc(stages, func(a, b StageDuration) int {
		return cmp.Compare(a.Stage, b.Stage)
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

// computeActionDistribution groups log entries by tool and action, tracking
// total calls and failures per pair. Results are sorted by calls descending.
func computeActionDistribution(entries []Entry) []ToolActionCount {
	type key struct {
		tool   string
		action string
	}
	type counts struct {
		total    int
		failures int
	}
	acc := map[key]*counts{}

	for _, e := range entries {
		action := ""
		if e.Action != nil {
			action = *e.Action
		}
		k := key{tool: e.Tool, action: action}
		if acc[k] == nil {
			acc[k] = &counts{}
		}
		acc[k].total++
		if !e.Success {
			acc[k].failures++
		}
	}

	result := make([]ToolActionCount, 0, len(acc))
	for k, c := range acc {
		result = append(result, ToolActionCount{
			Tool:     k.tool,
			Action:   k.action,
			Count:    c.total,
			Failures: c.failures,
		})
	}

	// Sort by calls descending, then tool ascending, then action ascending.
	slices.SortFunc(result, func(a, b ToolActionCount) int {
		if n := cmp.Compare(b.Count, a.Count); n != 0 {
			return n
		}
		if n := cmp.Compare(a.Tool, b.Tool); n != 0 {
			return n
		}
		return cmp.Compare(a.Action, b.Action)
	})

	return result
}

// computeDocTypeFunnel collects entity IDs from doc tool entries during the
// single entry pass, then batch-resolves document types via the lookup. It
// produces a funnel comparing registered vs approved document counts by type.
func computeDocTypeFunnel(entries []Entry, lookup StageFeatureLookup) *DocFunnelMetric {
	// First pass: collect per-entity action counts from doc tool entries.
	type entityActions struct {
		register int
		approve  int
	}
	entityCounts := map[string]*entityActions{}

	for _, e := range entries {
		if e.Tool != "doc" || e.EntityID == nil || e.Action == nil {
			continue
		}
		eid := *e.EntityID
		if entityCounts[eid] == nil {
			entityCounts[eid] = &entityActions{}
		}
		switch *e.Action {
		case "register":
			entityCounts[eid].register++
		case "approve":
			entityCounts[eid].approve++
		}
	}

	if len(entityCounts) == 0 {
		return nil
	}

	// Batch-resolve document types and aggregate by doc type.
	type docTypeAcc struct {
		registered int
		approved   int
	}
	funnel := map[string]*docTypeAcc{}

	for eid, actions := range entityCounts {
		dt, err := lookup.DocType(eid)
		if err != nil || dt == "" {
			continue
		}
		if funnel[dt] == nil {
			funnel[dt] = &docTypeAcc{}
		}
		funnel[dt].registered += actions.register
		funnel[dt].approved += actions.approve
	}

	if len(funnel) == 0 {
		return nil
	}

	// Build flat records with computed rate.
	metric := &DocFunnelMetric{}
	for dt, acc := range funnel {
		rate := 0.0
		if acc.registered > 0 {
			rate = float64(acc.approved) / float64(acc.registered)
		}
		metric.Records = append(metric.Records, DocFunnelRecord{
			DocType:    dt,
			Registered: acc.registered,
			Approved:   acc.approved,
			Rate:       rate,
		})
	}

	// Stable sort by doc type.
	slices.SortFunc(metric.Records, func(a, b DocFunnelRecord) int {
		return cmp.Compare(a.DocType, b.DocType)
	})

	return metric
}

// computeTaskCompletionGap pairs next→finish entries for task entities and
// computes the median and p90 gap duration in hours.
func computeTaskCompletionGap(entries []Entry) *TaskGapMetric {
	// Collect next timestamps keyed by entity ID.
	nextTimes := map[string]time.Time{}

	for _, e := range entries {
		if e.Tool != "next" || e.EntityID == nil {
			continue
		}
		ts, err := time.Parse(time.RFC3339, e.Timestamp)
		if err != nil {
			continue
		}
		// Keep the earliest next call for each entity.
		if existing, ok := nextTimes[*e.EntityID]; !ok || ts.Before(existing) {
			nextTimes[*e.EntityID] = ts
		}
	}

	// Pair with finish timestamps.
	var gaps []float64
	for _, e := range entries {
		if e.Tool != "finish" || e.EntityID == nil {
			continue
		}
		nextTS, ok := nextTimes[*e.EntityID]
		if !ok {
			continue
		}
		finishTS, err := time.Parse(time.RFC3339, e.Timestamp)
		if err != nil {
			continue
		}
		gap := finishTS.Sub(nextTS).Hours()
		if gap >= 0 {
			gaps = append(gaps, gap)
		}
	}

	if len(gaps) == 0 {
		return nil
	}

	slices.Sort(gaps)
	return &TaskGapMetric{
		Count:  len(gaps),
		Median: percentile(gaps, 50),
		P90:    percentile(gaps, 90),
	}
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
