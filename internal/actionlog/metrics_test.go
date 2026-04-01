package actionlog

import (
	"testing"
	"time"
)

func TestComputeGateFailureRate(t *testing.T) {
	t.Parallel()

	errGate := ErrorGateFailure
	errOther := ErrorInternalError

	entries := []Entry{
		{Success: false, ErrorType: &errGate},
		{Success: false, ErrorType: &errGate},
		{Success: true},
		{Success: false, ErrorType: &errOther},
	}

	rate := computeGateFailureRate(entries)
	if rate.Count != 2 {
		t.Errorf("Count: got %d, want 2", rate.Count)
	}
	if rate.Total != 4 {
		t.Errorf("Total: got %d, want 4", rate.Total)
	}
	if rate.Rate != 0.5 {
		t.Errorf("Rate: got %f, want 0.5", rate.Rate)
	}
}

func TestComputeTimePerStage(t *testing.T) {
	t.Parallel()

	base := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	features := []FeatureMetricsData{
		{
			FeatureID: "FEAT-001",
			Transitions: []StatusTransition{
				{ToStatus: "developing", At: base},
				{ToStatus: "reviewing", At: base.Add(48 * time.Hour)},
			},
		},
	}

	stages := computeTimePerStage(features)
	if len(stages) == 0 {
		t.Fatal("expected stages, got none")
	}

	var devStage *StageDuration
	for i := range stages {
		if stages[i].Stage == "developing" {
			devStage = &stages[i]
			break
		}
	}
	if devStage == nil {
		t.Fatal("developing stage not found")
	}
	if devStage.Median != 48.0 {
		t.Errorf("Median: got %f, want 48.0", devStage.Median)
	}
}

func TestComputeMetrics_NoLookup(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	input := MetricsInput{
		LogsDir: dir,
		Since:   time.Now().UTC().AddDate(0, 0, -7),
		Until:   time.Now().UTC(),
	}

	result, err := ComputeMetrics(input, nil)
	if err != nil {
		t.Fatalf("ComputeMetrics: %v", err)
	}
	if result == nil {
		t.Fatal("expected result, got nil")
	}
}

func TestComputeMetrics_WithLookup(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	input := MetricsInput{
		LogsDir: dir,
		Since:   time.Now().UTC().AddDate(0, 0, -7),
		Until:   time.Now().UTC(),
	}

	lookup := &stubFeatureLookup{features: []FeatureMetricsData{
		{FeatureID: "FEAT-001", DisplayID: "FEAT-01ABC", ReviewCycles: 2},
	}}

	result, err := ComputeMetrics(input, lookup)
	if err != nil {
		t.Fatalf("ComputeMetrics: %v", err)
	}
	if len(result.RevisionCycleCounts) != 1 {
		t.Errorf("RevisionCycleCounts: got %d, want 1", len(result.RevisionCycleCounts))
	}
}

type stubFeatureLookup struct {
	features []FeatureMetricsData
}

func (s *stubFeatureLookup) ListFeaturesInRange(since, until time.Time, featureID string) ([]FeatureMetricsData, error) {
	return s.features, nil
}
