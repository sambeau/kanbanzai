package model_test

import (
	"testing"
	"time"

	"kanbanzai/internal/model"

	"gopkg.in/yaml.v3"
)

// Compile-time Entity interface satisfaction checks.
var _ model.Entity = model.Epic{}
var _ model.Entity = model.Feature{}
var _ model.Entity = model.Task{}
var _ model.Entity = model.Bug{}
var _ model.Entity = model.Decision{}

func TestEpic_YAMLRoundTrip(t *testing.T) {
	t.Parallel()

	ts := time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)

	original := model.Epic{
		ID:        "E-001",
		Slug:      "phase-1-kernel",
		Title:     "Phase 1 Kernel",
		Status:    model.EpicStatusActive,
		Summary:   "Build the initial workflow kernel",
		Created:   ts,
		CreatedBy: "sam",
		Features:  []string{"FEAT-001", "FEAT-002"},
	}

	data, err := yaml.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	var decoded model.Epic
	if err := yaml.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if decoded.ID != original.ID {
		t.Errorf("ID = %q, want %q", decoded.ID, original.ID)
	}
	if decoded.Slug != original.Slug {
		t.Errorf("Slug = %q, want %q", decoded.Slug, original.Slug)
	}
	if decoded.Title != original.Title {
		t.Errorf("Title = %q, want %q", decoded.Title, original.Title)
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
	if len(decoded.Features) != len(original.Features) {
		t.Fatalf("Features length = %d, want %d", len(decoded.Features), len(original.Features))
	}
	for i, f := range decoded.Features {
		if f != original.Features[i] {
			t.Errorf("Features[%d] = %q, want %q", i, f, original.Features[i])
		}
	}
}

func TestFeature_YAMLRoundTrip(t *testing.T) {
	t.Parallel()

	ts := time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)

	original := model.Feature{
		ID:           "FEAT-001",
		Slug:         "initial-kernel",
		Epic:         "E-001",
		Status:       model.FeatureStatusInProgress,
		Summary:      "Start the workflow kernel",
		Created:      ts,
		CreatedBy:    "sam",
		Spec:         "spec/kernel.md",
		Plan:         "plan/kernel.md",
		Tasks:        []string{"FEAT-001.1", "FEAT-001.2"},
		Decisions:    []string{"DEC-001"},
		Branch:       "feat/kernel",
		Supersedes:   "FEAT-000",
		SupersededBy: "FEAT-002",
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
	if decoded.Epic != original.Epic {
		t.Errorf("Epic = %q, want %q", decoded.Epic, original.Epic)
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
		ID:           "FEAT-001.1",
		Feature:      "FEAT-001",
		Slug:         "write-entity-files",
		Summary:      "Write canonical entity files to disk",
		Status:       model.TaskStatusDone,
		Assignee:     "agent-1",
		DependsOn:    []string{"FEAT-001.2"},
		FilesPlanned: []string{"internal/storage/entity_store.go"},
		Started:      &started,
		Completed:    &completed,
		Verification: "go test ./internal/storage/...",
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
	if decoded.Feature != original.Feature {
		t.Errorf("Feature = %q, want %q", decoded.Feature, original.Feature)
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

func TestTask_YAMLRoundTrip_NilPointers(t *testing.T) {
	t.Parallel()

	original := model.Task{
		ID:      "FEAT-002.1",
		Feature: "FEAT-002",
		Slug:    "minimal-task",
		Summary: "A task with no optional fields",
		Status:  model.TaskStatusQueued,
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
	if decoded.Feature != original.Feature {
		t.Errorf("Feature = %q, want %q", decoded.Feature, original.Feature)
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
		ID:            "BUG-001",
		Slug:          "yaml-output-unstable",
		Title:         "Writer produces unstable YAML",
		Status:        model.BugStatusTriaged,
		Severity:      model.BugSeverityHigh,
		Priority:      model.BugPriorityCritical,
		Type:          model.BugTypeImplementationDefect,
		ReportedBy:    "sam",
		Reported:      ts,
		Observed:      "Repeated writes produce different output",
		Expected:      "Repeated writes should be stable",
		Affects:       []string{"FEAT-001", "FEAT-002"},
		OriginFeature: "FEAT-001",
		OriginTask:    "FEAT-001.1",
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
	if decoded.Title != original.Title {
		t.Errorf("Title = %q, want %q", decoded.Title, original.Title)
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
		ID:           "DEC-001",
		Slug:         "strict-yaml-subset",
		Summary:      "Use a strict canonical YAML subset",
		Rationale:    "Deterministic output is required for Git-friendly state",
		DecidedBy:    "sam",
		Date:         ts,
		Status:       model.DecisionStatusAccepted,
		Affects:      []string{"FEAT-001", "FEAT-003"},
		Supersedes:   "DEC-000",
		SupersededBy: "DEC-002",
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

	if model.EntityKindEpic != "epic" {
		t.Errorf("EntityKindEpic = %q, want %q", model.EntityKindEpic, "epic")
	}
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
}

func TestEnumStringValues(t *testing.T) {
	t.Parallel()

	// EpicStatus
	if model.EpicStatusProposed != "proposed" {
		t.Errorf("EpicStatusProposed = %q, want %q", model.EpicStatusProposed, "proposed")
	}
	if model.EpicStatusApproved != "approved" {
		t.Errorf("EpicStatusApproved = %q, want %q", model.EpicStatusApproved, "approved")
	}
	if model.EpicStatusActive != "active" {
		t.Errorf("EpicStatusActive = %q, want %q", model.EpicStatusActive, "active")
	}
	if model.EpicStatusOnHold != "on-hold" {
		t.Errorf("EpicStatusOnHold = %q, want %q", model.EpicStatusOnHold, "on-hold")
	}
	if model.EpicStatusDone != "done" {
		t.Errorf("EpicStatusDone = %q, want %q", model.EpicStatusDone, "done")
	}

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

func TestEntity_GetKind(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		e    model.Entity
		want model.EntityKind
	}{
		{"Epic", model.Epic{ID: "E-001", Slug: "test"}, model.EntityKindEpic},
		{"Feature", model.Feature{ID: "FEAT-001", Slug: "test"}, model.EntityKindFeature},
		{"Task", model.Task{ID: "FEAT-001.1", Slug: "test"}, model.EntityKindTask},
		{"Bug", model.Bug{ID: "BUG-001", Slug: "test"}, model.EntityKindBug},
		{"Decision", model.Decision{ID: "DEC-001", Slug: "test"}, model.EntityKindDecision},
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
