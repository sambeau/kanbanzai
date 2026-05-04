package actionlog

import (
	"encoding/json"
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

func TestComputeActionDistribution(t *testing.T) {
	t.Parallel()

	act1 := "register"
	act2 := "approve"
	entries := []Entry{
		{Tool: "doc", Action: &act1, Success: true},
		{Tool: "doc", Action: &act1, Success: false},
		{Tool: "doc", Action: &act2, Success: true},
		{Tool: "entity", Action: &act1, Success: true},
		{Tool: "entity", Success: true},
	}

	dist := computeActionDistribution(entries)
	if len(dist) != 4 {
		t.Fatalf("expected 4 tool-action pairs, got %d", len(dist))
	}

	// Verify counts and failures.
	counts := map[string]int{}
	failures := map[string]int{}
	for _, d := range dist {
		counts[d.Tool+":"+d.Action] = d.Count
		failures[d.Tool+":"+d.Action] = d.Failures
	}
	if counts["doc:register"] != 2 {
		t.Errorf("doc:register Count: got %d, want 2", counts["doc:register"])
	}
	if failures["doc:register"] != 1 {
		t.Errorf("doc:register Failures: got %d, want 1", failures["doc:register"])
	}
	if counts["doc:approve"] != 1 {
		t.Errorf("doc:approve Count: got %d, want 1", counts["doc:approve"])
	}
	if failures["doc:approve"] != 0 {
		t.Errorf("doc:approve Failures: got %d, want 0", failures["doc:approve"])
	}

	// Verify sort order: doc:register (2) should be first.
	if dist[0].Tool != "doc" || dist[0].Action != "register" {
		t.Errorf("first entry should be doc:register (2 calls), got %s:%s (%d calls)", dist[0].Tool, dist[0].Action, dist[0].Count)
	}
}

func TestComputeDocTypeFunnel(t *testing.T) {
	t.Parallel()

	entityID := "FEAT-001"
	reg := "register"
	app := "approve"
	now := time.Now().UTC().Format(time.RFC3339)

	entries := []Entry{
		{Timestamp: now, Tool: "doc", Action: &reg, EntityID: &entityID},
		{Timestamp: now, Tool: "doc", Action: &app, EntityID: &entityID},
	}

	lookup := &stubFeatureLookup{
		docTypes: map[string]string{"FEAT-001": "specification"},
	}

	funnel := computeDocTypeFunnel(entries, lookup)
	if funnel == nil {
		t.Fatal("expected funnel, got nil")
	}
	// Verify new flat record structure with Rate field.
	if len(funnel.Records) != 1 {
		t.Fatalf("Records len: got %d, want 1", len(funnel.Records))
	}
	rec := funnel.Records[0]
	if rec.DocType != "specification" {
		t.Errorf("DocType: got %q, want specification", rec.DocType)
	}
	if rec.Registered != 1 {
		t.Errorf("Registered: got %d, want 1", rec.Registered)
	}
	if rec.Approved != 1 {
		t.Errorf("Approved: got %d, want 1", rec.Approved)
	}
	if rec.Rate != 1.0 {
		t.Errorf("Rate: got %f, want 1.0", rec.Rate)
	}
}

func TestComputeDocTypeFunnel_Empty(t *testing.T) {
	t.Parallel()

	// No doc tool entries.
	entries := []Entry{
		{Tool: "entity", Action: strPtr("get")},
	}
	lookup := &stubFeatureLookup{}

	funnel := computeDocTypeFunnel(entries, lookup)
	if funnel != nil {
		t.Errorf("expected nil funnel for empty doc entries, got %+v", funnel)
	}
}

func TestComputeTaskCompletionGap(t *testing.T) {
	t.Parallel()

	entityID := "TASK-001"
	base := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)

	entries := []Entry{
		{Timestamp: base.Format(time.RFC3339), Tool: "next", EntityID: &entityID},
		{Timestamp: base.Add(3 * time.Hour).Format(time.RFC3339), Tool: "finish", EntityID: &entityID},
	}

	gap := computeTaskCompletionGap(entries)
	if gap == nil {
		t.Fatal("expected gap, got nil")
	}
	if gap.Count != 1 {
		t.Errorf("Count: got %d, want 1", gap.Count)
	}
	if gap.Median != 3.0 {
		t.Errorf("Median: got %f, want 3.0", gap.Median)
	}
}

func TestComputeTaskCompletionGap_NoFinish(t *testing.T) {
	t.Parallel()

	entityID := "TASK-001"
	base := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)

	// Only next, no finish.
	entries := []Entry{
		{Timestamp: base.Format(time.RFC3339), Tool: "next", EntityID: &entityID},
	}

	gap := computeTaskCompletionGap(entries)
	if gap != nil {
		t.Errorf("expected nil gap without finish, got %+v", gap)
	}
}

// AC-017: ComputeMetrics makes a single ReadEntries call. We verify this by checking
// that all metrics categories (GateFailureRate, ActionDistribution, DocTypeFunnel,
// TaskCompletionGap) are computed from the same set of entries. If ComputeMetrics
// made multiple ReadEntries calls, each might return different subsets due to
// file-system races; a single call guarantees consistency.
func TestComputeMetrics_SingleReadEntriesCall(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	// Write entries that span multiple metric categories in a single log file.
	entityID := "TASK-001"
	base := time.Date(2024, 6, 1, 10, 0, 0, 0, time.UTC)
	actReg := "register"
	actApp := "approve"
	errGate := ErrorGateFailure
	docEntity := "DOC-001"

	lines := []string{
		// A gate failure entry (for GateFailureRate).
		mustMarshal(Entry{Timestamp: base.Format(time.RFC3339), Tool: "entity", Action: strPtr("transition"), ServerVersion: "2.0", Success: false, ErrorType: &errGate}),
		// A doc register entry (for DocTypeFunnel).
		mustMarshal(Entry{Timestamp: base.Add(1 * time.Hour).Format(time.RFC3339), Tool: "doc", Action: &actReg, EntityID: &docEntity, ServerVersion: "2.0", Success: true}),
		// A doc approve entry (for DocTypeFunnel).
		mustMarshal(Entry{Timestamp: base.Add(2 * time.Hour).Format(time.RFC3339), Tool: "doc", Action: &actApp, EntityID: &docEntity, ServerVersion: "2.0", Success: true}),
		// A next entry (for TaskCompletionGap).
		mustMarshal(Entry{Timestamp: base.Add(3 * time.Hour).Format(time.RFC3339), Tool: "next", EntityID: &entityID, ServerVersion: "2.0", Success: true}),
		// A finish entry (for TaskCompletionGap).
		mustMarshal(Entry{Timestamp: base.Add(6 * time.Hour).Format(time.RFC3339), Tool: "finish", EntityID: &entityID, ServerVersion: "2.0", Success: true}),
	}
	writeLogFile(t, dir, "2024-06-01", lines)

	input := MetricsInput{
		LogsDir: dir,
		Since:   base,
		Until:   base.Add(24 * time.Hour),
	}

	lookup := &stubFeatureLookup{
		docTypes: map[string]string{"DOC-001": "specification"},
	}

	result, err := ComputeMetrics(input, lookup)
	if err != nil {
		t.Fatalf("ComputeMetrics: %v", err)
	}

	// All metrics should be populated from the same single read pass.
	// GateFailureRate: 1 failure out of 5 entries.
	if result.GateFailureRate.Count != 1 {
		t.Errorf("GateFailureRate.Count: got %d, want 1 (AC-017)", result.GateFailureRate.Count)
	}
	if result.GateFailureRate.Total != 5 {
		t.Errorf("GateFailureRate.Total: got %d, want 5 (AC-017)", result.GateFailureRate.Total)
	}

	// ActionDistribution: should have doc:register, doc:approve, entity:transition, next:, finish:
	if len(result.ActionDistribution) != 5 {
		t.Errorf("ActionDistribution len: got %d, want 5 (AC-017)", len(result.ActionDistribution))
	}

	// DocTypeFunnel: specification with 1 register + 1 approve, rate 1.0
	if result.DocTypeFunnel == nil {
		t.Fatal("DocTypeFunnel: got nil, want non-nil (AC-017)")
	}
	if len(result.DocTypeFunnel.Records) != 1 {
		t.Fatalf("DocTypeFunnel.Records len: got %d, want 1 (AC-017)", len(result.DocTypeFunnel.Records))
	}
	rec := result.DocTypeFunnel.Records[0]
	if rec.Registered != 1 || rec.Approved != 1 || rec.Rate != 1.0 {
		t.Errorf("DocTypeFunnel record: got reg=%d app=%d rate=%f, want reg=1 app=1 rate=1.0 (AC-017)", rec.Registered, rec.Approved, rec.Rate)
	}

	// TaskCompletionGap: 3-hour gap (from 10:00+3h=13:00 next to 10:00+6h=16:00 finish)
	if result.TaskCompletionGap == nil {
		t.Fatal("TaskCompletionGap: got nil, want non-nil (AC-017)")
	}
	if result.TaskCompletionGap.Count != 1 {
		t.Errorf("TaskCompletionGap.Count: got %d, want 1 (AC-017)", result.TaskCompletionGap.Count)
	}
	if result.TaskCompletionGap.Median != 3.0 {
		t.Errorf("TaskCompletionGap.Median: got %f, want 3.0 (AC-017)", result.TaskCompletionGap.Median)
	}
}

// mustMarshal is a test helper that marshals an Entry to JSON string or panics.
func mustMarshal(e Entry) string {
	b, err := json.Marshal(e)
	if err != nil {
		panic(err)
	}
	return string(b)
}

func strPtr(s string) *string {
	return &s
}

type stubFeatureLookup struct {
	features []FeatureMetricsData
	docTypes map[string]string
}

func (s *stubFeatureLookup) ListFeaturesInRange(since, until time.Time, featureID string) ([]FeatureMetricsData, error) {
	return s.features, nil
}

func (s *stubFeatureLookup) DocType(entityID string) (string, error) {
	if s.docTypes == nil {
		return "", nil
	}
	return s.docTypes[entityID], nil
}
