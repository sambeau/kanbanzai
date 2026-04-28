// Package kbzschema provides exported Go types for reading Kanbanzai entity
// records. These types cover the committed public schema of a
// Kanbanzai-managed repository and allow external Go programs to parse entity
// YAML files without depending on any internal Kanbanzai package.
package kbzschema

const SchemaVersion = "1.0.0"

// Batch status constants
const (
	BatchStatusProposed   = "proposed"
	BatchStatusDesigning  = "designing"
	BatchStatusActive     = "active"
	BatchStatusDone       = "done"
	BatchStatusSuperseded = "superseded"
	BatchStatusCancelled  = "cancelled"
)

// Deprecated: use BatchStatus* constants.
const (
	PlanStatusProposed   = BatchStatusProposed
	PlanStatusDesigning  = BatchStatusDesigning
	PlanStatusActive     = BatchStatusActive
	PlanStatusDone       = BatchStatusDone
	PlanStatusSuperseded = BatchStatusSuperseded
	PlanStatusCancelled  = BatchStatusCancelled
)

// Feature status constants
const (
	FeatureStatusProposed    = "proposed"
	FeatureStatusDesigning   = "designing"
	FeatureStatusSpecifying  = "specifying"
	FeatureStatusDevPlanning = "dev-planning"
	FeatureStatusDeveloping  = "developing"
	FeatureStatusDone        = "done"
	FeatureStatusSuperseded  = "superseded"
	FeatureStatusCancelled   = "cancelled"
)

// Task status constants
const (
	TaskStatusQueued      = "queued"
	TaskStatusReady       = "ready"
	TaskStatusActive      = "active"
	TaskStatusBlocked     = "blocked"
	TaskStatusNeedsReview = "needs-review"
	TaskStatusNeedsRework = "needs-rework"
	TaskStatusDone        = "done"
	TaskStatusNotPlanned  = "not-planned"
	TaskStatusDuplicate   = "duplicate"
)

// Bug status constants
const (
	BugStatusReported        = "reported"
	BugStatusTriaged         = "triaged"
	BugStatusReproduced      = "reproduced"
	BugStatusPlanned         = "planned"
	BugStatusInProgress      = "in-progress"
	BugStatusNeedsReview     = "needs-review"
	BugStatusNeedsRework     = "needs-rework"
	BugStatusVerified        = "verified"
	BugStatusClosed          = "closed"
	BugStatusDuplicate       = "duplicate"
	BugStatusNotPlanned      = "not-planned"
	BugStatusCannotReproduce = "cannot-reproduce"
)

// Bug severity constants
const (SeverityLow = "low"; SeverityMedium = "medium"; SeverityHigh = "high"; SeverityCritical = "critical")

// Bug priority constants
const (PriorityLow = "low"; PriorityMedium = "medium"; PriorityHigh = "high"; PriorityCritical = "critical")

// Bug type constants
const (BugTypeImplementationDefect = "implementation-defect"; BugTypeSpecificationDefect = "specification-defect"; BugTypeDesignProblem = "design-problem")

// Decision status constants
const (DecisionStatusProposed = "proposed"; DecisionStatusAccepted = "accepted"; DecisionStatusRejected = "rejected"; DecisionStatusSuperseded = "superseded")

// Document type constants
const (
	DocTypeDesign = "design"; DocTypeSpecification = "specification"; DocTypeDevPlan = "dev-plan"
	DocTypeResearch = "research"; DocTypeReport = "report"; DocTypePolicy = "policy"; DocTypeRCA = "rca"
)

// Document status constants
const (DocStatusDraft = "draft"; DocStatusApproved = "approved"; DocStatusSuperseded = "superseded")

// Knowledge entry status constants
const (
	KnowledgeStatusContributed = "contributed"; KnowledgeStatusConfirmed = "confirmed"
	KnowledgeStatusDisputed = "disputed"; KnowledgeStatusStale = "stale"; KnowledgeStatusRetired = "retired"
)

const (KnowledgeTier2 = 2; KnowledgeTier3 = 3)

// Checkpoint status constants
const (CheckpointStatusPending = "pending"; CheckpointStatusResponded = "responded")

// Entity types

// Batch represents a Batch entity record stored in .kbz/state/batches/.
type Batch struct {
	ID           string   `yaml:"id" json:"id"`
	Slug         string   `yaml:"slug" json:"slug"`
	Title        string   `yaml:"title" json:"title"`
	Status       string   `yaml:"status" json:"status"`
	Summary      string   `yaml:"summary" json:"summary"`
	Design       string   `yaml:"design,omitempty" json:"design,omitempty"`
	Tags         []string `yaml:"tags,omitempty" json:"tags,omitempty"`
	Created      string   `yaml:"created" json:"created"`
	CreatedBy    string   `yaml:"created_by" json:"created_by"`
	Updated      string   `yaml:"updated" json:"updated"`
	Supersedes   string   `yaml:"supersedes,omitempty" json:"supersedes,omitempty"`
	SupersededBy string   `yaml:"superseded_by,omitempty" json:"superseded_by,omitempty"`
}

// Deprecated: use Batch.
type Plan = Batch

// Feature represents a Feature entity record.
type Feature struct {
	ID           string   `yaml:"id" json:"id"`
	Slug         string   `yaml:"slug" json:"slug"`
	Parent       string   `yaml:"parent,omitempty" json:"parent,omitempty"`
	Status       string   `yaml:"status" json:"status"`
	Estimate     *float64 `yaml:"estimate,omitempty" json:"estimate,omitempty"`
	Summary      string   `yaml:"summary" json:"summary"`
	Design       string   `yaml:"design,omitempty" json:"design,omitempty"`
	Spec         string   `yaml:"spec,omitempty" json:"spec,omitempty"`
	DevPlan      string   `yaml:"dev_plan,omitempty" json:"dev_plan,omitempty"`
	Tasks        []string `yaml:"tasks,omitempty" json:"tasks,omitempty"`
	Decisions    []string `yaml:"decisions,omitempty" json:"decisions,omitempty"`
	Tags         []string `yaml:"tags,omitempty" json:"tags,omitempty"`
	Branch       string   `yaml:"branch,omitempty" json:"branch,omitempty"`
	Created      string   `yaml:"created" json:"created"`
	CreatedBy    string   `yaml:"created_by" json:"created_by"`
	Updated      string   `yaml:"updated" json:"updated"`
	Supersedes   string   `yaml:"supersedes,omitempty" json:"supersedes,omitempty"`
	SupersededBy string   `yaml:"superseded_by,omitempty" json:"superseded_by,omitempty"`
}

// Task represents a Task entity record.
type Task struct {
	ID                string   `yaml:"id" json:"id"`
	ParentFeature     string   `yaml:"parent_feature" json:"parent_feature"`
	Slug              string   `yaml:"slug" json:"slug"`
	Summary           string   `yaml:"summary" json:"summary"`
	Status            string   `yaml:"status" json:"status"`
	Estimate          *float64 `yaml:"estimate,omitempty" json:"estimate,omitempty"`
	Assignee          string   `yaml:"assignee,omitempty" json:"assignee,omitempty"`
	DependsOn         []string `yaml:"depends_on,omitempty" json:"depends_on,omitempty"`
	FilesPlanned      []string `yaml:"files_planned,omitempty" json:"files_planned,omitempty"`
	Started           string   `yaml:"started,omitempty" json:"started,omitempty"`
	Completed         string   `yaml:"completed,omitempty" json:"completed,omitempty"`
	ClaimedAt         string   `yaml:"claimed_at,omitempty" json:"claimed_at,omitempty"`
	DispatchedTo      string   `yaml:"dispatched_to,omitempty" json:"dispatched_to,omitempty"`
	DispatchedAt      string   `yaml:"dispatched_at,omitempty" json:"dispatched_at,omitempty"`
	DispatchedBy      string   `yaml:"dispatched_by,omitempty" json:"dispatched_by,omitempty"`
	CompletionSummary string   `yaml:"completion_summary,omitempty" json:"completion_summary,omitempty"`
	ReworkReason      string   `yaml:"rework_reason,omitempty" json:"rework_reason,omitempty"`
	Verification      string   `yaml:"verification,omitempty" json:"verification,omitempty"`
	Tags              []string `yaml:"tags,omitempty" json:"tags,omitempty"`
}

// Bug represents a Bug entity record.
type Bug struct {
	ID            string   `yaml:"id" json:"id"`
	Slug          string   `yaml:"slug" json:"slug"`
	Title         string   `yaml:"title" json:"title"`
	Status        string   `yaml:"status" json:"status"`
	Estimate      *float64 `yaml:"estimate,omitempty" json:"estimate,omitempty"`
	Severity      string   `yaml:"severity" json:"severity"`
	Priority      string   `yaml:"priority" json:"priority"`
	Type          string   `yaml:"type" json:"type"`
	ReportedBy    string   `yaml:"reported_by" json:"reported_by"`
	Reported      string   `yaml:"reported" json:"reported"`
	Observed      string   `yaml:"observed" json:"observed"`
	Expected      string   `yaml:"expected" json:"expected"`
	Affects       []string `yaml:"affects,omitempty" json:"affects,omitempty"`
	OriginFeature string   `yaml:"origin_feature,omitempty" json:"origin_feature,omitempty"`
	OriginTask    string   `yaml:"origin_task,omitempty" json:"origin_task,omitempty"`
	Environment   string   `yaml:"environment,omitempty" json:"environment,omitempty"`
	Reproduction  string   `yaml:"reproduction,omitempty" json:"reproduction,omitempty"`
	DuplicateOf   string   `yaml:"duplicate_of,omitempty" json:"duplicate_of,omitempty"`
	FixedBy       string   `yaml:"fixed_by,omitempty" json:"fixed_by,omitempty"`
	VerifiedBy    string   `yaml:"verified_by,omitempty" json:"verified_by,omitempty"`
	ReleaseTarget string   `yaml:"release_target,omitempty" json:"release_target,omitempty"`
	Tags          []string `yaml:"tags,omitempty" json:"tags,omitempty"`
}

// Decision represents a Decision entity record.
type Decision struct {
	ID           string   `yaml:"id" json:"id"`
	Slug         string   `yaml:"slug" json:"slug"`
	Summary      string   `yaml:"summary" json:"summary"`
	Rationale    string   `yaml:"rationale" json:"rationale"`
	DecidedBy    string   `yaml:"decided_by" json:"decided_by"`
	Date         string   `yaml:"date" json:"date"`
	Status       string   `yaml:"status" json:"status"`
	Affects      []string `yaml:"affects,omitempty" json:"affects,omitempty"`
	Supersedes   string   `yaml:"supersedes,omitempty" json:"supersedes,omitempty"`
	SupersededBy string   `yaml:"superseded_by,omitempty" json:"superseded_by,omitempty"`
	Tags         []string `yaml:"tags,omitempty" json:"tags,omitempty"`
}

// DocumentRecord represents a document record.
type DocumentRecord struct {
	ID           string   `yaml:"id" json:"id"`
	Path         string   `yaml:"path" json:"path"`
	Type         string   `yaml:"type" json:"type"`
	Title        string   `yaml:"title" json:"title"`
	Status       string   `yaml:"status" json:"status"`
	Owner        string   `yaml:"owner,omitempty" json:"owner,omitempty"`
	ApprovedBy   string   `yaml:"approved_by,omitempty" json:"approved_by,omitempty"`
	ApprovedAt   string   `yaml:"approved_at,omitempty" json:"approved_at,omitempty"`
	ContentHash  string   `yaml:"content_hash" json:"content_hash"`
	Supersedes   string   `yaml:"supersedes,omitempty" json:"supersedes,omitempty"`
	SupersededBy string   `yaml:"superseded_by,omitempty" json:"superseded_by,omitempty"`
	Created      string   `yaml:"created" json:"created"`
	CreatedBy    string   `yaml:"created_by" json:"created_by"`
	Updated      string   `yaml:"updated" json:"updated"`
	Tags         []string `yaml:"tags,omitempty" json:"tags,omitempty"`
}

// KnowledgeEntry represents a knowledge entry.
type KnowledgeEntry struct {
	ID               string   `yaml:"id" json:"id"`
	Tier             int      `yaml:"tier" json:"tier"`
	Topic            string   `yaml:"topic" json:"topic"`
	Scope            string   `yaml:"scope" json:"scope"`
	Content          string   `yaml:"content" json:"content"`
	LearnedFrom      string   `yaml:"learned_from" json:"learned_from"`
	Status           string   `yaml:"status" json:"status"`
	UseCount         int      `yaml:"use_count" json:"use_count"`
	MissCount        int      `yaml:"miss_count" json:"miss_count"`
	Confidence       float64  `yaml:"confidence" json:"confidence"`
	LastUsed         string   `yaml:"last_used" json:"last_used"`
	TTLDays          int      `yaml:"ttl_days" json:"ttl_days"`
	PromotedFrom     string   `yaml:"promoted_from,omitempty" json:"promoted_from,omitempty"`
	MergedFrom       []string `yaml:"merged_from,omitempty" json:"merged_from,omitempty"`
	DeprecatedReason string   `yaml:"deprecated_reason,omitempty" json:"deprecated_reason,omitempty"`
	GitAnchors       []string `yaml:"git_anchors,omitempty" json:"git_anchors,omitempty"`
	Tags             []string `yaml:"tags,omitempty" json:"tags,omitempty"`
	Created          string   `yaml:"created" json:"created"`
	CreatedBy        string   `yaml:"created_by" json:"created_by"`
	Updated          string   `yaml:"updated" json:"updated"`
}

// HumanCheckpoint represents a human checkpoint record.
type HumanCheckpoint struct {
	ID                   string `yaml:"id" json:"id"`
	Question             string `yaml:"question" json:"question"`
	Context              string `yaml:"context" json:"context"`
	OrchestrationSummary string `yaml:"orchestration_summary,omitempty" json:"orchestration_summary,omitempty"`
	Status               string `yaml:"status" json:"status"`
	CreatedAt            string `yaml:"created_at" json:"created_at"`
	CreatedBy            string `yaml:"created_by" json:"created_by"`
	RespondedAt          string `yaml:"responded_at,omitempty" json:"responded_at,omitempty"`
	Response             string `yaml:"response,omitempty" json:"response,omitempty"`
}

// ProjectConfig represents the project configuration.
type ProjectConfig struct {
	Version      string        `yaml:"version" json:"version"`
	SchemaVersion string       `yaml:"schema_version" json:"schema_version"`
	Prefixes     []PrefixEntry `yaml:"prefixes,omitempty" json:"prefixes,omitempty"`
}

// PrefixEntry represents a single prefix entry.
type PrefixEntry struct {
	Prefix      string `yaml:"prefix" json:"prefix"`
	Name        string `yaml:"name" json:"name"`
	Description string `yaml:"description" json:"description"`
	Retired     bool   `yaml:"retired,omitempty" json:"retired,omitempty"`
}
