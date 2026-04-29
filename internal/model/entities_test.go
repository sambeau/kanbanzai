package model_test

import (
	"testing"
	"time"

	"github.com/sambeau/kanbanzai/internal/model"
	"github.com/sambeau/kanbanzai/internal/testutil"

	"gopkg.in/yaml.v3"
)

// Compile-time Entity interface satisfaction checks.
var _ model.Entity = model.Plan{}
var _ model.Entity = model.Feature{}
var _ model.Entity = model.Task{}
var _ model.Entity = model.Bug{}
var _ model.Entity = model.Decision{}
var _ model.Entity = model.DocumentRecord{}

func TestFeature_YAMLRoundTrip(t *testing.T) {
	t.Parallel()

	ts := time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)

	original := model.Feature{
		ID:           testutil.TestFeatureID,
		Slug:         "initial-kernel",
		Status:       model.FeatureStatusInProgress,
		Summary:      "Start the workflow kernel",
		Created:      ts,
		CreatedBy:    "sam",
		Spec:         "spec/kernel.md",
		Plan:         "plan/kernel.md",
		Tasks:        []string{testutil.TestTaskID, "TASK-01J3KZZZCC5LG"},
		Decisions:    []string{testutil.TestDecisionID},
		Branch:       "feat/kernel",
		Supersedes:   "FEAT-01J3K0SUPER00",
		SupersededBy: testutil.TestFeatureID2,
	}

	data, err := yaml.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	var decoded model.Feature
	if err := yaml.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if decoded.ID != original.ID {
		t.Errorf("ID = %q, want %q", decoded.ID, original.ID)
	}
	if decoded.Slug != original.Slug {
		t.Errorf("Slug = %q, want %q", decoded.Slug, original.Slug)
	}
	if decoded.Status != original.Status {
		t.Errorf("Status = %q, want %q", decoded.Status, original.Status)
	}
	if decoded.Summary != original.Summary {
		t.Errorf("Summary = %q, want %q", decoded.Summary, original.Summary)
	}
	if !decoded.Created.Equal(original.Created) {
		t.Errorf("Created = %v, want %v", decoded.Created, original.Created)
	}
	if decoded.CreatedBy != original.CreatedBy {
		t.Errorf("CreatedBy = %q, want %q", decoded.CreatedBy, original.CreatedBy)
	}
	if decoded.Spec != original.Spec {
		t.Errorf("Spec = %q, want %q", decoded.Spec, original.Spec)
	}
	if decoded.Plan != original.Plan {
		t.Errorf("Plan = %q, want %q", decoded.Plan, original.Plan)
	}
	if len(decoded.Tasks) != len(original.Tasks) {
		t.Fatalf("Tasks length = %d, want %d", len(decoded.Tasks), len(original.Tasks))
	}
	for i, v := range decoded.Tasks {
		if v != original.Tasks[i] {
			t.Errorf("Tasks[%d] = %q, want %q", i, v, original.Tasks[i])
		}
	}
	if len(decoded.Decisions) != len(original.Decisions) {
		t.Fatalf("Decisions length = %d, want %d", len(decoded.Decisions), len(original.Decisions))
	}
	for i, v := range decoded.Decisions {
		if v != original.Decisions[i] {
			t.Errorf("Decisions[%d] = %q, want %q", i, v, original.Decisions[i])
		}
	}
	if decoded.Branch != original.Branch {
		t.Errorf("Branch = %q, want %q", decoded.Branch, original.Branch)
	}
	if decoded.Supersedes != original.Supersedes {
		t.Errorf("Supersedes = %q, want %q", decoded.Supersedes, original.Supersedes)
	}
	if decoded.SupersededBy != original.SupersededBy {
		t.Errorf("SupersededBy = %q, want %q", decoded.SupersededBy, original.SupersededBy)
	}
}

func TestTask_YAMLRoundTrip_WithPointers(t *testing.T) {
	t.Parallel()

	started := time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)
	completed := time.Date(2025, 1, 16, 14, 0, 0, 0, time.UTC)

	original := model.Task{
		ID:            testutil.TestTaskID,
		ParentFeature: testutil.TestFeatureID,
		Slug:          "write-entity-files",
		Summary:       "Write canonical entity files to disk",
		Status:        model.TaskStatusDone,
		Assignee:      "agent-1",
		DependsOn:     []string{"TASK-01J3KZZZCC5LG"},
		FilesPlanned:  []string{"internal/storage/entity_store.go"},
		Started:       &started,
		Completed:     &completed,
		Verification:  "go test ./internal/storage/...",
	}

	data, err := yaml.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	var decoded model.Task
	if err := yaml.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if decoded.ID != original.ID {
		t.Errorf("ID = %q, want %q", decoded.ID, original.ID)
	}
	if decoded.ParentFeature != original.ParentFeature {
		t.Errorf("ParentFeature = %q, want %q", decoded.ParentFeature, original.ParentFeature)
	}
	if decoded.Slug != original.Slug {
		t.Errorf("Slug = %q, want %q", decoded.Slug, original.Slug)
	}
	if decoded.Summary != original.Summary {
		t.Errorf("Summary = %q, want %q", decoded.Summary, original.Summary)
	}
	if decoded.Status != original.Status {
		t.Errorf("Status = %q, want %q", decoded.Status, original.Status)
	}
	if decoded.Assignee != original.Assignee {
		t.Errorf("Assignee = %q, want %q", decoded.Assignee, original.Assignee)
	}
	if len(decoded.DependsOn) != len(original.DependsOn) {
		t.Fatalf("DependsOn length = %d, want %d", len(decoded.DependsOn), len(original.DependsOn))
	}
	for i, v := range decoded.DependsOn {
		if v != original.DependsOn[i] {
			t.Errorf("DependsOn[%d] = %q, want %q", i, v, original.DependsOn[i])
		}
	}
	if len(decoded.FilesPlanned) != len(original.FilesPlanned) {
		t.Fatalf("FilesPlanned length = %d, want %d", len(decoded.FilesPlanned), len(original.FilesPlanned))
	}
	for i, v := range decoded.FilesPlanned {
		if v != original.FilesPlanned[i] {
			t.Errorf("FilesPlanned[%d] = %q, want %q", i, v, original.FilesPlanned[i])
		}
	}
	if decoded.Started == nil {
		t.Fatal("Started = nil, want non-nil")
	}
	if !decoded.Started.Equal(*original.Started) {
		t.Errorf("Started = %v, want %v", *decoded.Started, *original.Started)
	}
	if decoded.Completed == nil {
		t.Fatal("Completed = nil, want non-nil")
	}
	if !decoded.Completed.Equal(*original.Completed) {
		t.Errorf("Completed = %v, want %v", *decoded.Completed, *original.Completed)
	}
	if decoded.Verification != original.Verification {
		t.Errorf("Verification = %q, want %q", decoded.Verification, original.Verification)
	}

	// Verify GetKind/GetID/GetSlug on the round-tripped value.
	if decoded.GetKind() != model.EntityKindTask {
		t.Errorf("GetKind() = %q, want %q", decoded.GetKind(), model.EntityKindTask)
	}
	if decoded.GetID() != original.ID {
		t.Errorf("GetID() = %q, want %q", decoded.GetID(), original.ID)
	}
	if decoded.GetSlug() != original.Slug {
		t.Errorf("GetSlug() = %q, want %q", decoded.GetSlug(), original.Slug)
	}
}

func TestTask_YAMLRoundTrip_WithReworkReason(t *testing.T) {
	t.Parallel()

	started := time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)

	original := model.Task{
		ID:            testutil.TestTaskID,
		ParentFeature: testutil.TestFeatureID,
		Slug:          "rework-reason-task",
		Summary:       "Task with rework reason for round-trip test",
		Status:        model.TaskStatusNeedsRework,
		Assignee:      "agent-1",
		Started:       &started,
		ReworkReason:  "output file missing: internal/auth.go; verification criteria not met",
		Verification:  "go test ./internal/auth/...",
	}

	data, err := yaml.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	var decoded model.Task
	if err := yaml.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if decoded.ReworkReason != original.ReworkReason {
		t.Errorf("ReworkReason = %q, want %q", decoded.ReworkReason, original.ReworkReason)
	}
	if decoded.ID != original.ID {
		t.Errorf("ID = %q, want %q", decoded.ID, original.ID)
	}
	if decoded.Status != original.Status {
		t.Errorf("Status = %q, want %q", decoded.Status, original.Status)
	}
	if decoded.Verification != original.Verification {
		t.Errorf("Verification = %q, want %q", decoded.Verification, original.Verification)
	}

	// Verify round-trip stability: marshal the decoded value and compare.
	data2, err := yaml.Marshal(decoded)
	if err != nil {
		t.Fatalf("second Marshal() error = %v", err)
	}
	if string(data) != string(data2) {
		t.Errorf("round-trip mismatch:\n--- first ---\n%s\n--- second ---\n%s", data, data2)
	}
}

func TestTask_YAMLRoundTrip_NilPointers(t *testing.T) {
	t.Parallel()

	original := model.Task{
		ID:            "TASK-01J3KZZZCC5LG",
		ParentFeature: testutil.TestFeatureID2,
		Slug:          "minimal-task",
		Summary:       "A task with no optional fields",
		Status:        model.TaskStatusQueued,
	}

	data, err := yaml.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	var decoded model.Task
	if err := yaml.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if decoded.ID != original.ID {
		t.Errorf("ID = %q, want %q", decoded.ID, original.ID)
	}
	if decoded.ParentFeature != original.ParentFeature {
		t.Errorf("ParentFeature = %q, want %q", decoded.ParentFeature, original.ParentFeature)
	}
	if decoded.Slug != original.Slug {
		t.Errorf("Slug = %q, want %q", decoded.Slug, original.Slug)
	}
	if decoded.Summary != original.Summary {
		t.Errorf("Summary = %q, want %q", decoded.Summary, original.Summary)
	}
	if decoded.Status != original.Status {
		t.Errorf("Status = %q, want %q", decoded.Status, original.Status)
	}
	if decoded.Started != nil {
		t.Errorf("Started = %v, want nil", decoded.Started)
	}
	if decoded.Completed != nil {
		t.Errorf("Completed = %v, want nil", decoded.Completed)
	}
}

func TestBug_YAMLRoundTrip(t *testing.T) {
	t.Parallel()

	ts := time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)

	original := model.Bug{
		ID:            testutil.TestBugID,
		Slug:          "yaml-output-unstable",
		Name:          "YAML output unstable",
		Status:        model.BugStatusTriaged,
		Severity:      model.BugSeverityHigh,
		Priority:      model.BugPriorityCritical,
		Type:          model.BugTypeImplementationDefect,
		ReportedBy:    "sam",
		Reported:      ts,
		Observed:      "Repeated writes produce different output",
		Expected:      "Repeated writes should be stable",
		Affects:       []string{testutil.TestFeatureID, testutil.TestFeatureID2},
		OriginFeature: testutil.TestFeatureID,
		OriginTask:    testutil.TestTaskID,
		Environment:   "CI pipeline, Go 1.23",
		Reproduction:  "Run write 10 times, compare output",
		DuplicateOf:   "",
		FixedBy:       "",
		VerifiedBy:    "",
		ReleaseTarget: "",
	}

	data, err := yaml.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	var decoded model.Bug
	if err := yaml.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if decoded.ID != original.ID {
		t.Errorf("ID = %q, want %q", decoded.ID, original.ID)
	}
	if decoded.Slug != original.Slug {
		t.Errorf("Slug = %q, want %q", decoded.Slug, original.Slug)
	}
	if decoded.Name != original.Name {
		t.Errorf("Name = %q, want %q", decoded.Name, original.Name)
	}
	if decoded.Status != original.Status {
		t.Errorf("Status = %q, want %q", decoded.Status, original.Status)
	}
	if decoded.Severity != original.Severity {
		t.Errorf("Severity = %q, want %q", decoded.Severity, original.Severity)
	}
	if decoded.Priority != original.Priority {
		t.Errorf("Priority = %q, want %q", decoded.Priority, original.Priority)
	}
	if decoded.Type != original.Type {
		t.Errorf("Type = %q, want %q", decoded.Type, original.Type)
	}
	if decoded.ReportedBy != original.ReportedBy {
		t.Errorf("ReportedBy = %q, want %q", decoded.ReportedBy, original.ReportedBy)
	}
	if !decoded.Reported.Equal(original.Reported) {
		t.Errorf("Reported = %v, want %v", decoded.Reported, original.Reported)
	}
	if decoded.Observed != original.Observed {
		t.Errorf("Observed = %q, want %q", decoded.Observed, original.Observed)
	}
	if decoded.Expected != original.Expected {
		t.Errorf("Expected = %q, want %q", decoded.Expected, original.Expected)
	}
	if len(decoded.Affects) != len(original.Affects) {
		t.Fatalf("Affects length = %d, want %d", len(decoded.Affects), len(original.Affects))
	}
	for i, v := range decoded.Affects {
		if v != original.Affects[i] {
			t.Errorf("Affects[%d] = %q, want %q", i, v, original.Affects[i])
		}
	}
	if decoded.Environment != original.Environment {
		t.Errorf("Environment = %q, want %q", decoded.Environment, original.Environment)
	}
	if decoded.OriginFeature != original.OriginFeature {
		t.Errorf("OriginFeature = %q, want %q", decoded.OriginFeature, original.OriginFeature)
	}
	if decoded.OriginTask != original.OriginTask {
		t.Errorf("OriginTask = %q, want %q", decoded.OriginTask, original.OriginTask)
	}
	if decoded.Reproduction != original.Reproduction {
		t.Errorf("Reproduction = %q, want %q", decoded.Reproduction, original.Reproduction)
	}
}

func TestDecision_YAMLRoundTrip(t *testing.T) {
	t.Parallel()

	ts := time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)

	original := model.Decision{
		ID:           testutil.TestDecisionID,
		Slug:         "strict-yaml-subset",
		Summary:      "Use a strict canonical YAML subset",
		Rationale:    "Deterministic output is required for Git-friendly state",
		DecidedBy:    "sam",
		Date:         ts,
		Status:       model.DecisionStatusAccepted,
		Affects:      []string{testutil.TestFeatureID, "FEAT-01J3K9NPQ5TW7"},
		Supersedes:   "DEC-01J3KABCDG9PZ",
		SupersededBy: "DEC-01J3KABCDF8NY",
	}

	data, err := yaml.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	var decoded model.Decision
	if err := yaml.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if decoded.ID != original.ID {
		t.Errorf("ID = %q, want %q", decoded.ID, original.ID)
	}
	if decoded.Slug != original.Slug {
		t.Errorf("Slug = %q, want %q", decoded.Slug, original.Slug)
	}
	if decoded.Summary != original.Summary {
		t.Errorf("Summary = %q, want %q", decoded.Summary, original.Summary)
	}
	if decoded.Rationale != original.Rationale {
		t.Errorf("Rationale = %q, want %q", decoded.Rationale, original.Rationale)
	}
	if decoded.DecidedBy != original.DecidedBy {
		t.Errorf("DecidedBy = %q, want %q", decoded.DecidedBy, original.DecidedBy)
	}
	if !decoded.Date.Equal(original.Date) {
		t.Errorf("Date = %v, want %v", decoded.Date, original.Date)
	}
	if decoded.Status != original.Status {
		t.Errorf("Status = %q, want %q", decoded.Status, original.Status)
	}
	if len(decoded.Affects) != len(original.Affects) {
		t.Fatalf("Affects length = %d, want %d", len(decoded.Affects), len(original.Affects))
	}
	for i, v := range decoded.Affects {
		if v != original.Affects[i] {
			t.Errorf("Affects[%d] = %q, want %q", i, v, original.Affects[i])
		}
	}
	if decoded.Supersedes != original.Supersedes {
		t.Errorf("Supersedes = %q, want %q", decoded.Supersedes, original.Supersedes)
	}
	if decoded.SupersededBy != original.SupersededBy {
		t.Errorf("SupersededBy = %q, want %q", decoded.SupersededBy, original.SupersededBy)
	}
}

func TestEntityKind_Values(t *testing.T) {
	t.Parallel()

	if model.EntityKindFeature != "feature" {
		t.Errorf("EntityKindFeature = %q, want %q", model.EntityKindFeature, "feature")
	}
	if model.EntityKindTask != "task" {
		t.Errorf("EntityKindTask = %q, want %q", model.EntityKindTask, "task")
	}
	if model.EntityKindBug != "bug" {
		t.Errorf("EntityKindBug = %q, want %q", model.EntityKindBug, "bug")
	}
	if model.EntityKindDecision != "decision" {
		t.Errorf("EntityKindDecision = %q, want %q", model.EntityKindDecision, "decision")
	}
	if model.EntityKindDocument != "document" {
		t.Errorf("EntityKindDocument = %q, want %q", model.EntityKindDocument, "document")
	}
}

func TestEnumStringValues(t *testing.T) {
	t.Parallel()

	// FeatureStatus
	if model.FeatureStatusDraft != "draft" {
		t.Errorf("FeatureStatusDraft = %q, want %q", model.FeatureStatusDraft, "draft")
	}
	if model.FeatureStatusInReview != "in-review" {
		t.Errorf("FeatureStatusInReview = %q, want %q", model.FeatureStatusInReview, "in-review")
	}
	if model.FeatureStatusApproved != "approved" {
		t.Errorf("FeatureStatusApproved = %q, want %q", model.FeatureStatusApproved, "approved")
	}
	if model.FeatureStatusInProgress != "in-progress" {
		t.Errorf("FeatureStatusInProgress = %q, want %q", model.FeatureStatusInProgress, "in-progress")
	}
	if model.FeatureStatusReview != "review" {
		t.Errorf("FeatureStatusReview = %q, want %q", model.FeatureStatusReview, "review")
	}
	if model.FeatureStatusReviewing != "reviewing" {
		t.Errorf("FeatureStatusReviewing = %q, want %q", model.FeatureStatusReviewing, "reviewing")
	}
	if model.FeatureStatusNeedsRework != "needs-rework" {
		t.Errorf("FeatureStatusNeedsRework = %q, want %q", model.FeatureStatusNeedsRework, "needs-rework")
	}
	if model.FeatureStatusDone != "done" {
		t.Errorf("FeatureStatusDone = %q, want %q", model.FeatureStatusDone, "done")
	}
	if model.FeatureStatusSuperseded != "superseded" {
		t.Errorf("FeatureStatusSuperseded = %q, want %q", model.FeatureStatusSuperseded, "superseded")
	}

	// TaskStatus
	if model.TaskStatusQueued != "queued" {
		t.Errorf("TaskStatusQueued = %q, want %q", model.TaskStatusQueued, "queued")
	}
	if model.TaskStatusReady != "ready" {
		t.Errorf("TaskStatusReady = %q, want %q", model.TaskStatusReady, "ready")
	}
	if model.TaskStatusActive != "active" {
		t.Errorf("TaskStatusActive = %q, want %q", model.TaskStatusActive, "active")
	}
	if model.TaskStatusBlocked != "blocked" {
		t.Errorf("TaskStatusBlocked = %q, want %q", model.TaskStatusBlocked, "blocked")
	}
	if model.TaskStatusNeedsReview != "needs-review" {
		t.Errorf("TaskStatusNeedsReview = %q, want %q", model.TaskStatusNeedsReview, "needs-review")
	}
	if model.TaskStatusNeedsRework != "needs-rework" {
		t.Errorf("TaskStatusNeedsRework = %q, want %q", model.TaskStatusNeedsRework, "needs-rework")
	}
	if model.TaskStatusDone != "done" {
		t.Errorf("TaskStatusDone = %q, want %q", model.TaskStatusDone, "done")
	}

	// BugStatus
	if model.BugStatusReported != "reported" {
		t.Errorf("BugStatusReported = %q, want %q", model.BugStatusReported, "reported")
	}
	if model.BugStatusTriaged != "triaged" {
		t.Errorf("BugStatusTriaged = %q, want %q", model.BugStatusTriaged, "triaged")
	}
	if model.BugStatusReproduced != "reproduced" {
		t.Errorf("BugStatusReproduced = %q, want %q", model.BugStatusReproduced, "reproduced")
	}
	if model.BugStatusPlanned != "planned" {
		t.Errorf("BugStatusPlanned = %q, want %q", model.BugStatusPlanned, "planned")
	}
	if model.BugStatusInProgress != "in-progress" {
		t.Errorf("BugStatusInProgress = %q, want %q", model.BugStatusInProgress, "in-progress")
	}
	if model.BugStatusNeedsReview != "needs-review" {
		t.Errorf("BugStatusNeedsReview = %q, want %q", model.BugStatusNeedsReview, "needs-review")
	}
	if model.BugStatusNeedsRework != "needs-rework" {
		t.Errorf("BugStatusNeedsRework = %q, want %q", model.BugStatusNeedsRework, "needs-rework")
	}
	if model.BugStatusVerified != "verified" {
		t.Errorf("BugStatusVerified = %q, want %q", model.BugStatusVerified, "verified")
	}
	if model.BugStatusClosed != "closed" {
		t.Errorf("BugStatusClosed = %q, want %q", model.BugStatusClosed, "closed")
	}
	if model.BugStatusDuplicate != "duplicate" {
		t.Errorf("BugStatusDuplicate = %q, want %q", model.BugStatusDuplicate, "duplicate")
	}
	if model.BugStatusNotPlanned != "not-planned" {
		t.Errorf("BugStatusNotPlanned = %q, want %q", model.BugStatusNotPlanned, "not-planned")
	}
	if model.BugStatusCannotReproduce != "cannot-reproduce" {
		t.Errorf("BugStatusCannotReproduce = %q, want %q", model.BugStatusCannotReproduce, "cannot-reproduce")
	}

	// BugSeverity
	if model.BugSeverityLow != "low" {
		t.Errorf("BugSeverityLow = %q, want %q", model.BugSeverityLow, "low")
	}
	if model.BugSeverityMedium != "medium" {
		t.Errorf("BugSeverityMedium = %q, want %q", model.BugSeverityMedium, "medium")
	}
	if model.BugSeverityHigh != "high" {
		t.Errorf("BugSeverityHigh = %q, want %q", model.BugSeverityHigh, "high")
	}
	if model.BugSeverityCritical != "critical" {
		t.Errorf("BugSeverityCritical = %q, want %q", model.BugSeverityCritical, "critical")
	}

	// BugPriority
	if model.BugPriorityLow != "low" {
		t.Errorf("BugPriorityLow = %q, want %q", model.BugPriorityLow, "low")
	}
	if model.BugPriorityMedium != "medium" {
		t.Errorf("BugPriorityMedium = %q, want %q", model.BugPriorityMedium, "medium")
	}
	if model.BugPriorityHigh != "high" {
		t.Errorf("BugPriorityHigh = %q, want %q", model.BugPriorityHigh, "high")
	}
	if model.BugPriorityCritical != "critical" {
		t.Errorf("BugPriorityCritical = %q, want %q", model.BugPriorityCritical, "critical")
	}

	// BugType
	if model.BugTypeImplementationDefect != "implementation-defect" {
		t.Errorf("BugTypeImplementationDefect = %q, want %q", model.BugTypeImplementationDefect, "implementation-defect")
	}
	if model.BugTypeSpecificationDefect != "specification-defect" {
		t.Errorf("BugTypeSpecificationDefect = %q, want %q", model.BugTypeSpecificationDefect, "specification-defect")
	}
	if model.BugTypeDesignProblem != "design-problem" {
		t.Errorf("BugTypeDesignProblem = %q, want %q", model.BugTypeDesignProblem, "design-problem")
	}

	// DecisionStatus
	if model.DecisionStatusProposed != "proposed" {
		t.Errorf("DecisionStatusProposed = %q, want %q", model.DecisionStatusProposed, "proposed")
	}
	if model.DecisionStatusAccepted != "accepted" {
		t.Errorf("DecisionStatusAccepted = %q, want %q", model.DecisionStatusAccepted, "accepted")
	}
	if model.DecisionStatusRejected != "rejected" {
		t.Errorf("DecisionStatusRejected = %q, want %q", model.DecisionStatusRejected, "rejected")
	}
	if model.DecisionStatusSuperseded != "superseded" {
		t.Errorf("DecisionStatusSuperseded = %q, want %q", model.DecisionStatusSuperseded, "superseded")
	}
}

func TestValidDocumentType_NewTypes(t *testing.T) {
	t.Parallel()

	cases := []struct {
		input string
		want  bool
	}{
		{"plan", true},
		{"retrospective", true},
		{"design", true},
		{"specification", true},
		{"dev-plan", true},
		{"research", true},
		{"report", true},
		{"policy", true},
		{"rca", true},
		{"unknown", false},
		{"", false},
	}
	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			got := model.ValidDocumentType(tc.input)
			if got != tc.want {
				t.Errorf("ValidDocumentType(%q) = %v, want %v", tc.input, got, tc.want)
			}
		})
	}
}

func TestDocumentType_NewConstants(t *testing.T) {
	t.Parallel()

	if model.DocumentTypePlan != "plan" {
		t.Errorf("DocumentTypePlan = %q, want %q", model.DocumentTypePlan, "plan")
	}
	if model.DocumentTypeRetrospective != "retrospective" {
		t.Errorf("DocumentTypeRetrospective = %q, want %q", model.DocumentTypeRetrospective, "retrospective")
	}
}

func TestEntity_GetKind(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		e    model.Entity
		want model.EntityKind
	}{
		{"Feature", model.Feature{ID: testutil.TestFeatureID, Slug: "test"}, model.EntityKindFeature},
		{"Task", model.Task{ID: testutil.TestTaskID, Slug: "test"}, model.EntityKindTask},
		{"Bug", model.Bug{ID: testutil.TestBugID, Slug: "test"}, model.EntityKindBug},
		{"Decision", model.Decision{ID: testutil.TestDecisionID, Slug: "test"}, model.EntityKindDecision},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.e.GetKind(); got != tt.want {
				t.Errorf("GetKind() = %q, want %q", got, tt.want)
			}
			if got := tt.e.GetID(); got == "" {
				t.Error("GetID() returned empty string")
			}
			if got := tt.e.GetSlug(); got != "test" {
				t.Errorf("GetSlug() = %q, want %q", got, "test")
			}
		})
	}
}

func TestQualityEvaluation_YAMLRoundTrip(t *testing.T) {
	t.Parallel()

	ts := time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC)

	original := model.QualityEvaluation{
		OverallScore: 0.85,
		Pass:         true,
		EvaluatedAt:  ts,
		Evaluator:    "claude-sonnet",
		Dimensions: map[string]float64{
			"clarity":      0.9,
			"completeness": 0.8,
		},
	}

	data, err := yaml.Marshal(original)
	if err != nil {
		t.Fatalf("yaml.Marshal: %v", err)
	}

	var got model.QualityEvaluation
	if err := yaml.Unmarshal(data, &got); err != nil {
		t.Fatalf("yaml.Unmarshal: %v", err)
	}

	if got.OverallScore != original.OverallScore {
		t.Errorf("OverallScore = %g, want %g", got.OverallScore, original.OverallScore)
	}
	if got.Pass != original.Pass {
		t.Errorf("Pass = %v, want %v", got.Pass, original.Pass)
	}
	if got.Evaluator != original.Evaluator {
		t.Errorf("Evaluator = %q, want %q", got.Evaluator, original.Evaluator)
	}
	if !got.EvaluatedAt.Equal(original.EvaluatedAt) {
		t.Errorf("EvaluatedAt = %v, want %v", got.EvaluatedAt, original.EvaluatedAt)
	}
	if len(got.Dimensions) != len(original.Dimensions) {
		t.Errorf("len(Dimensions) = %d, want %d", len(got.Dimensions), len(original.Dimensions))
	}
	for k, v := range original.Dimensions {
		if got.Dimensions[k] != v {
			t.Errorf("Dimensions[%q] = %g, want %g", k, got.Dimensions[k], v)
		}
	}
}

func TestDocumentRecord_WithQualityEvaluation_YAMLRoundTrip(t *testing.T) {
	t.Parallel()

	ts := time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC)

	original := model.DocumentRecord{
		ID:          "FEAT-abc/my-design",
		Path:        "work/design/my-design.md",
		Type:        model.DocumentTypeDesign,
		Title:       "My Design",
		Status:      model.DocumentStatusDraft,
		ContentHash: "abc123",
		Created:     ts,
		CreatedBy:   "tester",
		Updated:     ts,
		QualityEvaluation: &model.QualityEvaluation{
			OverallScore: 0.75,
			Pass:         true,
			EvaluatedAt:  ts,
			Evaluator:    "model-v1",
			Dimensions:   map[string]float64{"clarity": 0.8},
		},
	}

	data, err := yaml.Marshal(original)
	if err != nil {
		t.Fatalf("yaml.Marshal: %v", err)
	}

	var got model.DocumentRecord
	if err := yaml.Unmarshal(data, &got); err != nil {
		t.Fatalf("yaml.Unmarshal: %v", err)
	}

	if got.QualityEvaluation == nil {
		t.Fatal("QualityEvaluation is nil after round-trip")
	}
	if got.QualityEvaluation.OverallScore != 0.75 {
		t.Errorf("OverallScore = %g, want 0.75", got.QualityEvaluation.OverallScore)
	}
	if got.QualityEvaluation.Evaluator != "model-v1" {
		t.Errorf("Evaluator = %q, want model-v1", got.QualityEvaluation.Evaluator)
	}
}

func TestParseShortPlanRef(t *testing.T) {
	tests := []struct {
		input      string
		wantPrefix string
		wantNumber string
		wantOK     bool
	}{
		// AC-006: basic ASCII prefix
		{"P30", "P", "30", true},
		// AC-010: non-ASCII Unicode prefix (ñ = U+00F1)
		{"ñ5", "ñ", "5", true},
		// additional valid inputs
		{"Z1", "Z", "1", true},
		{"a999", "a", "999", true},
		// AC-007: hyphen present
		{"P30-foo", "", "", false},
		// AC-008: no leading non-digit rune
		{"30", "", "", false},
		// AC-009: empty string
		{"", "", "", false},
		// no digits after prefix
		{"P", "", "", false},
		// trailing non-digit char
		{"P30X", "", "", false},
	}
	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			gotPrefix, gotNumber, gotOK := model.ParseShortPlanRef(tc.input)
			if gotOK != tc.wantOK {
				t.Errorf("ParseShortPlanRef(%q) ok = %v, want %v", tc.input, gotOK, tc.wantOK)
			}
			if gotPrefix != tc.wantPrefix {
				t.Errorf("ParseShortPlanRef(%q) prefix = %q, want %q", tc.input, gotPrefix, tc.wantPrefix)
			}
			if gotNumber != tc.wantNumber {
				t.Errorf("ParseShortPlanRef(%q) number = %q, want %q", tc.input, gotNumber, tc.wantNumber)
			}
		})
	}
}
