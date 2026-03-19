package model

import "time"

// EntityKind identifies a Phase 1 canonical entity type.
type EntityKind string

const (
	EntityKindEpic     EntityKind = "epic"
	EntityKindFeature  EntityKind = "feature"
	EntityKindTask     EntityKind = "task"
	EntityKindBug      EntityKind = "bug"
	EntityKindDecision EntityKind = "decision"
)

// EpicStatus is the lifecycle state for an Epic.
type EpicStatus string

const (
	EpicStatusProposed EpicStatus = "proposed"
	EpicStatusApproved EpicStatus = "approved"
	EpicStatusActive   EpicStatus = "active"
	EpicStatusOnHold   EpicStatus = "on-hold"
	EpicStatusDone     EpicStatus = "done"
)

// FeatureStatus is the lifecycle state for a Feature.
type FeatureStatus string

const (
	FeatureStatusDraft       FeatureStatus = "draft"
	FeatureStatusInReview    FeatureStatus = "in-review"
	FeatureStatusApproved    FeatureStatus = "approved"
	FeatureStatusInProgress  FeatureStatus = "in-progress"
	FeatureStatusReview      FeatureStatus = "review"
	FeatureStatusNeedsRework FeatureStatus = "needs-rework"
	FeatureStatusDone        FeatureStatus = "done"
	FeatureStatusSuperseded  FeatureStatus = "superseded"
)

// TaskStatus is the lifecycle state for a Task.
type TaskStatus string

const (
	TaskStatusQueued      TaskStatus = "queued"
	TaskStatusReady       TaskStatus = "ready"
	TaskStatusActive      TaskStatus = "active"
	TaskStatusBlocked     TaskStatus = "blocked"
	TaskStatusNeedsReview TaskStatus = "needs-review"
	TaskStatusNeedsRework TaskStatus = "needs-rework"
	TaskStatusDone        TaskStatus = "done"
)

// BugStatus is the lifecycle state for a Bug.
type BugStatus string

const (
	BugStatusReported        BugStatus = "reported"
	BugStatusTriaged         BugStatus = "triaged"
	BugStatusReproduced      BugStatus = "reproduced"
	BugStatusPlanned         BugStatus = "planned"
	BugStatusInProgress      BugStatus = "in-progress"
	BugStatusNeedsReview     BugStatus = "needs-review"
	BugStatusNeedsRework     BugStatus = "needs-rework"
	BugStatusVerified        BugStatus = "verified"
	BugStatusClosed          BugStatus = "closed"
	BugStatusDuplicate       BugStatus = "duplicate"
	BugStatusNotPlanned      BugStatus = "not-planned"
	BugStatusCannotReproduce BugStatus = "cannot-reproduce"
)

// BugSeverity classifies defect severity.
type BugSeverity string

const (
	BugSeverityLow      BugSeverity = "low"
	BugSeverityMedium   BugSeverity = "medium"
	BugSeverityHigh     BugSeverity = "high"
	BugSeverityCritical BugSeverity = "critical"
)

// BugPriority classifies implementation priority.
type BugPriority string

const (
	BugPriorityLow      BugPriority = "low"
	BugPriorityMedium   BugPriority = "medium"
	BugPriorityHigh     BugPriority = "high"
	BugPriorityCritical BugPriority = "critical"
)

// BugType classifies the nature of a bug.
type BugType string

const (
	BugTypeImplementationDefect BugType = "implementation-defect"
	BugTypeSpecificationDefect  BugType = "specification-defect"
	BugTypeDesignProblem        BugType = "design-problem"
)

// DecisionStatus is the lifecycle state for a Decision.
type DecisionStatus string

const (
	DecisionStatusProposed   DecisionStatus = "proposed"
	DecisionStatusAccepted   DecisionStatus = "accepted"
	DecisionStatusRejected   DecisionStatus = "rejected"
	DecisionStatusSuperseded DecisionStatus = "superseded"
)

// Entity is the shared behavior for all canonical Phase 1 entities.
type Entity interface {
	GetKind() EntityKind
	GetID() string
	GetSlug() string
}

// Epic is the canonical Phase 1 representation of an Epic.
type Epic struct {
	ID        string     `yaml:"id"`
	Slug      string     `yaml:"slug"`
	Title     string     `yaml:"title"`
	Status    EpicStatus `yaml:"status"`
	Summary   string     `yaml:"summary"`
	Created   time.Time  `yaml:"created"`
	CreatedBy string     `yaml:"created_by"`

	Features []string `yaml:"features,omitempty"`
}

// GetKind returns the entity kind.
func (Epic) GetKind() EntityKind {
	return EntityKindEpic
}

// GetID returns the canonical ID.
func (e Epic) GetID() string {
	return e.ID
}

// GetSlug returns the human-readable slug.
func (e Epic) GetSlug() string {
	return e.Slug
}

// Feature is the canonical Phase 1 representation of a Feature.
type Feature struct {
	ID        string        `yaml:"id"`
	Slug      string        `yaml:"slug"`
	Epic      string        `yaml:"epic"`
	Status    FeatureStatus `yaml:"status"`
	Summary   string        `yaml:"summary"`
	Created   time.Time     `yaml:"created"`
	CreatedBy string        `yaml:"created_by"`

	Spec         string   `yaml:"spec,omitempty"`
	Plan         string   `yaml:"plan,omitempty"`
	Tasks        []string `yaml:"tasks,omitempty"`
	Decisions    []string `yaml:"decisions,omitempty"`
	Branch       string   `yaml:"branch,omitempty"`
	Supersedes   string   `yaml:"supersedes,omitempty"`
	SupersededBy string   `yaml:"superseded_by,omitempty"`
}

// GetKind returns the entity kind.
func (Feature) GetKind() EntityKind {
	return EntityKindFeature
}

// GetID returns the canonical ID.
func (f Feature) GetID() string {
	return f.ID
}

// GetSlug returns the human-readable slug.
func (f Feature) GetSlug() string {
	return f.Slug
}

// Task is the canonical Phase 1 representation of a Task.
type Task struct {
	ID      string     `yaml:"id"`
	Feature string     `yaml:"feature"`
	Slug    string     `yaml:"slug"`
	Summary string     `yaml:"summary"`
	Status  TaskStatus `yaml:"status"`

	Assignee     string     `yaml:"assignee,omitempty"`
	DependsOn    []string   `yaml:"depends_on,omitempty"`
	FilesPlanned []string   `yaml:"files_planned,omitempty"`
	Started      *time.Time `yaml:"started,omitempty"`
	Completed    *time.Time `yaml:"completed,omitempty"`
	Verification string     `yaml:"verification,omitempty"`
}

// GetKind returns the entity kind.
func (Task) GetKind() EntityKind {
	return EntityKindTask
}

// GetID returns the canonical ID.
func (t Task) GetID() string {
	return t.ID
}

// GetSlug returns the human-readable slug.
func (t Task) GetSlug() string {
	return t.Slug
}

// Bug is the canonical Phase 1 representation of a Bug.
type Bug struct {
	ID         string      `yaml:"id"`
	Slug       string      `yaml:"slug"`
	Title      string      `yaml:"title"`
	Status     BugStatus   `yaml:"status"`
	Severity   BugSeverity `yaml:"severity"`
	Priority   BugPriority `yaml:"priority"`
	Type       BugType     `yaml:"type"`
	ReportedBy string      `yaml:"reported_by"`
	Reported   time.Time   `yaml:"reported"`
	Observed   string      `yaml:"observed"`
	Expected   string      `yaml:"expected"`

	Affects       []string `yaml:"affects,omitempty"`
	OriginFeature string   `yaml:"origin_feature,omitempty"`
	OriginTask    string   `yaml:"origin_task,omitempty"`
	Environment   string   `yaml:"environment,omitempty"`
	Reproduction  string   `yaml:"reproduction,omitempty"`
	DuplicateOf   string   `yaml:"duplicate_of,omitempty"`
	FixedBy       string   `yaml:"fixed_by,omitempty"`
	VerifiedBy    string   `yaml:"verified_by,omitempty"`
	ReleaseTarget string   `yaml:"release_target,omitempty"`
}

// GetKind returns the entity kind.
func (Bug) GetKind() EntityKind {
	return EntityKindBug
}

// GetID returns the canonical ID.
func (b Bug) GetID() string {
	return b.ID
}

// GetSlug returns the human-readable slug.
func (b Bug) GetSlug() string {
	return b.Slug
}

// Decision is the canonical Phase 1 representation of a Decision.
type Decision struct {
	ID        string         `yaml:"id"`
	Slug      string         `yaml:"slug"`
	Summary   string         `yaml:"summary"`
	Rationale string         `yaml:"rationale"`
	DecidedBy string         `yaml:"decided_by"`
	Date      time.Time      `yaml:"date"`
	Status    DecisionStatus `yaml:"status"`

	Affects      []string `yaml:"affects,omitempty"`
	Supersedes   string   `yaml:"supersedes,omitempty"`
	SupersededBy string   `yaml:"superseded_by,omitempty"`
}

// GetKind returns the entity kind.
func (Decision) GetKind() EntityKind {
	return EntityKindDecision
}

// GetID returns the canonical ID.
func (d Decision) GetID() string {
	return d.ID
}

// GetSlug returns the human-readable slug.
func (d Decision) GetSlug() string {
	return d.Slug
}
