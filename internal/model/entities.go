package model

import (
	"time"
	"unicode"
)

// EntityKind identifies a canonical entity type.
type EntityKind string

const (
	EntityKindBatch         EntityKind = "batch"
	EntityKindFeature       EntityKind = "feature"
	EntityKindTask          EntityKind = "task"
	EntityKindBug           EntityKind = "bug"
	EntityKindDecision      EntityKind = "decision"
	EntityKindDocument      EntityKind = "document"
	EntityKindIncident      EntityKind = "incident"
	EntityKindStrategicPlan EntityKind = "strategic-plan"

	// Deprecated: use EntityKindBatch.
	EntityKindPlan EntityKind = EntityKindBatch
)

// BatchStatus is the lifecycle state for a Batch (execution-level plan).
type BatchStatus string

const (
	BatchStatusProposed   BatchStatus = "proposed"
	BatchStatusDesigning  BatchStatus = "designing"
	BatchStatusActive     BatchStatus = "active"
	BatchStatusReviewing  BatchStatus = "reviewing"
	BatchStatusDone       BatchStatus = "done"
	BatchStatusSuperseded BatchStatus = "superseded"
	BatchStatusCancelled  BatchStatus = "cancelled"
)

// Deprecated: use BatchStatus.
type PlanStatus = BatchStatus

// Deprecated constants.
const (
	PlanStatusProposed   = BatchStatusProposed
	PlanStatusDesigning  = BatchStatusDesigning
	PlanStatusActive     = BatchStatusActive
	PlanStatusReviewing  = BatchStatusReviewing
	PlanStatusDone       = BatchStatusDone
	PlanStatusSuperseded = BatchStatusSuperseded
	PlanStatusCancelled  = BatchStatusCancelled
)

// PlanningStatus is the lifecycle state for a strategic Plan entity.
// After P38-F3 renames PlanStatus to BatchStatus, this type will become PlanStatus.
type PlanningStatus string

const (
	PlanningStatusIdea       PlanningStatus = "idea"
	PlanningStatusShaping    PlanningStatus = "shaping"
	PlanningStatusReady      PlanningStatus = "ready"
	PlanningStatusActive     PlanningStatus = "active"
	PlanningStatusDone       PlanningStatus = "done"
	PlanningStatusSuperseded PlanningStatus = "superseded"
	PlanningStatusCancelled  PlanningStatus = "cancelled"
)

// FeatureStatus is the lifecycle state for a Feature.
type FeatureStatus string

const (
	// Phase 2 Feature statuses (document-driven lifecycle)
	FeatureStatusProposed    FeatureStatus = "proposed"
	FeatureStatusDesigning   FeatureStatus = "designing"
	FeatureStatusSpecifying  FeatureStatus = "specifying"
	FeatureStatusDevPlanning FeatureStatus = "dev-planning"
	FeatureStatusDeveloping  FeatureStatus = "developing"
	FeatureStatusReviewing   FeatureStatus = "reviewing"
	FeatureStatusNeedsRework FeatureStatus = "needs-rework"
	FeatureStatusDone        FeatureStatus = "done"
	FeatureStatusSuperseded  FeatureStatus = "superseded"
	FeatureStatusCancelled   FeatureStatus = "cancelled"

	// Phase 1 Feature statuses (deprecated, for migration compatibility)
	FeatureStatusDraft      FeatureStatus = "draft"
	FeatureStatusInReview   FeatureStatus = "in-review"
	FeatureStatusApproved   FeatureStatus = "approved"
	FeatureStatusInProgress FeatureStatus = "in-progress"
	FeatureStatusReview     FeatureStatus = "review"
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
	TaskStatusNotPlanned  TaskStatus = "not-planned"
	TaskStatusDuplicate   TaskStatus = "duplicate"
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

// IncidentStatus is the lifecycle state for an Incident.
type IncidentStatus string

const (
	IncidentStatusReported            IncidentStatus = "reported"
	IncidentStatusTriaged             IncidentStatus = "triaged"
	IncidentStatusInvestigating       IncidentStatus = "investigating"
	IncidentStatusRootCauseIdentified IncidentStatus = "root-cause-identified"
	IncidentStatusMitigated           IncidentStatus = "mitigated"
	IncidentStatusResolved            IncidentStatus = "resolved"
	IncidentStatusClosed              IncidentStatus = "closed"
)

// IncidentSeverity classifies incident severity.
type IncidentSeverity string

const (
	IncidentSeverityCritical IncidentSeverity = "critical"
	IncidentSeverityHigh     IncidentSeverity = "high"
	IncidentSeverityMedium   IncidentSeverity = "medium"
	IncidentSeverityLow      IncidentSeverity = "low"
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

// DocumentType identifies a recognised document type.
type DocumentType string

const (
	// User-facing document types (REQ-001).
	DocumentTypeDesign   DocumentType = "design"
	DocumentTypeSpec     DocumentType = "spec"
	DocumentTypeDevPlan  DocumentType = "dev-plan"
	DocumentTypeReview   DocumentType = "review"
	DocumentTypeReport   DocumentType = "report"
	DocumentTypeResearch DocumentType = "research"
	DocumentTypeRetro    DocumentType = "retro"
	DocumentTypeProposal DocumentType = "proposal"

	// Internal types accepted by doc register (REQ-004).
	DocumentTypePolicy DocumentType = "policy"
	DocumentTypeRCA    DocumentType = "rca"

	// Legacy synonyms kept for backward compatibility (REQ-002, REQ-003).
	DocumentTypeSpecification DocumentType = "specification"
	DocumentTypeRetrospective DocumentType = "retrospective"

	// Legacy internal type kept for storage backward compatibility (REQ-014, C-005).
	DocumentTypePlan DocumentType = "plan"
)

// AllDocumentTypes returns the eight user-facing document types in canonical order (REQ-001).
func AllDocumentTypes() []DocumentType {
	return []DocumentType{
		DocumentTypeDesign,
		DocumentTypeSpec,
		DocumentTypeDevPlan,
		DocumentTypeReview,
		DocumentTypeReport,
		DocumentTypeResearch,
		DocumentTypeRetro,
		DocumentTypeProposal,
	}
}

// ValidDocumentType returns true if the string is any recognised type, including
// legacy types (specification, retrospective, plan). Used in the storage layer for
// backward compatibility.
func ValidDocumentType(s string) bool {
	switch DocumentType(s) {
	case DocumentTypeDesign, DocumentTypeSpec, DocumentTypeDevPlan,
		DocumentTypeReview, DocumentTypeReport, DocumentTypeResearch,
		DocumentTypeRetro, DocumentTypeProposal,
		DocumentTypePolicy, DocumentTypeRCA,
		DocumentTypeSpecification, DocumentTypeRetrospective, DocumentTypePlan:
		return true
	}
	return false
}

// ValidDocumentTypeForRegistration returns true if the string is a type accepted
// by doc register: the eight user-facing types plus policy and rca (REQ-004).
// Legacy synonyms (specification, retrospective) are NOT accepted here because
// they are normalised before this check is called.
func ValidDocumentTypeForRegistration(s string) bool {
	switch DocumentType(s) {
	case DocumentTypeDesign, DocumentTypeSpec, DocumentTypeDevPlan,
		DocumentTypeReview, DocumentTypeReport, DocumentTypeResearch,
		DocumentTypeRetro, DocumentTypeProposal,
		DocumentTypePolicy, DocumentTypeRCA:
		return true
	}
	return false
}

// NormaliseDocumentType maps legacy type synonyms to their canonical short forms.
// specification → spec, retrospective → retro; all other types are returned unchanged.
func NormaliseDocumentType(t DocumentType) DocumentType {
	switch t {
	case DocumentTypeSpecification:
		return DocumentTypeSpec
	case DocumentTypeRetrospective:
		return DocumentTypeRetro
	default:
		return t
	}
}

// DocumentStatus is the lifecycle state of a document record.
type DocumentStatus string

const (
	DocumentStatusDraft      DocumentStatus = "draft"
	DocumentStatusApproved   DocumentStatus = "approved"
	DocumentStatusSuperseded DocumentStatus = "superseded"
)

// Entity is the shared behavior for all canonical entities.
type Entity interface {
	GetKind() EntityKind
	GetID() string
	GetSlug() string
}

// Batch is the canonical representation of a Batch (execution-level plan).
// A Batch coordinates a body of work, provides direction through its design
// document, and organises Features.
type Batch struct {
	ID             string      `yaml:"id"`
	Slug           string      `yaml:"slug"`
	Name           string      `yaml:"name"`
	Status         BatchStatus `yaml:"status"`
	Summary        string      `yaml:"summary"`
	Parent         string      `yaml:"parent,omitempty"`
	Design         string      `yaml:"design,omitempty"`
	Tags           []string    `yaml:"tags,omitempty"`
	Created        time.Time   `yaml:"created"`
	CreatedBy      string      `yaml:"created_by"`
	Updated        time.Time   `yaml:"updated"`
	NextFeatureSeq int         `yaml:"next_feature_seq"`

	Supersedes   string `yaml:"supersedes,omitempty"`
	SupersededBy string `yaml:"superseded_by,omitempty"`
}

// Deprecated: use Batch.
type Plan = Batch

// GetKind returns the entity kind.
func (Batch) GetKind() EntityKind {
	return EntityKindBatch
}

// GetID returns the canonical ID.
func (b Batch) GetID() string {
	return b.ID
}

// GetSlug returns the human-readable slug.
func (b Batch) GetSlug() string {
	return b.Slug
}

// StrategicPlan is the recursive entity for strategic planning.
// A StrategicPlan represents what needs to be built — a strategic direction, system decomposition,
// or themed group of work. Plans can contain child plans, batches, or both.
type StrategicPlan struct {
	ID           string         `yaml:"id"`
	Slug         string         `yaml:"slug"`
	Name         string         `yaml:"name"`
	Status       PlanningStatus `yaml:"status"`
	Summary      string         `yaml:"summary"`
	Parent       string         `yaml:"parent,omitempty"`
	Design       string         `yaml:"design,omitempty"`
	DependsOn    []string       `yaml:"depends_on,omitempty"`
	Order        int            `yaml:"order,omitempty"`
	Tags         []string       `yaml:"tags,omitempty"`
	Created      time.Time      `yaml:"created"`
	CreatedBy    string         `yaml:"created_by"`
	Updated      time.Time      `yaml:"updated"`
	Supersedes   string         `yaml:"supersedes,omitempty"`
	SupersededBy string         `yaml:"superseded_by,omitempty"`
}

// GetKind returns the entity kind.
func (StrategicPlan) GetKind() EntityKind {
	return EntityKindStrategicPlan
}

// GetID returns the canonical ID.
func (p StrategicPlan) GetID() string {
	return p.ID
}

// GetSlug returns the human-readable slug.
func (p StrategicPlan) GetSlug() string {
	return p.Slug
}

// OverrideRecord captures a single gate override performed on a feature
// transition. It is appended to Feature.Overrides each time a gate is
// bypassed via the override mechanism (FR-014, FR-016).
type OverrideRecord struct {
	FromStatus   string    `yaml:"from_status"`
	ToStatus     string    `yaml:"to_status"`
	Reason       string    `yaml:"reason"`
	Timestamp    time.Time `yaml:"timestamp"`
	CheckpointID string    `yaml:"checkpoint_id,omitempty"`
}

// Feature is the canonical representation of a Feature.
// In Phase 2, Feature lifecycle is driven by document approvals.
type Feature struct {
	ID            string        `yaml:"id"`
	Slug          string        `yaml:"slug"`
	Name          string        `yaml:"name"`
	DisplayID     string        `yaml:"display_id,omitempty"`
	Parent        string        `yaml:"parent,omitempty"` // Parent Batch/Plan ID
	Status        FeatureStatus `yaml:"status"`
	ReviewCycle   int           `yaml:"review_cycle,omitempty"`
	BlockedReason string        `yaml:"blocked_reason,omitempty"`
	Estimate      *float64      `yaml:"estimate,omitempty"`
	Summary       string        `yaml:"summary"`
	Created       time.Time     `yaml:"created"`
	CreatedBy     string        `yaml:"created_by"`
	Updated       time.Time     `yaml:"updated,omitempty"`

	// Document references (Phase 2)
	Design  string `yaml:"design,omitempty"`   // Reference to design document record
	Spec    string `yaml:"spec,omitempty"`     // Reference to specification document record
	DevPlan string `yaml:"dev_plan,omitempty"` // Reference to dev plan document record

	// Tags for cross-cutting organisational metadata
	Tags []string `yaml:"tags,omitempty"`

	// Legacy fields (Phase 1 compatibility)
	Plan string `yaml:"plan,omitempty"` // Deprecated: use DevPlan

	Tasks        []string         `yaml:"tasks,omitempty"`
	Decisions    []string         `yaml:"decisions,omitempty"`
	Branch       string           `yaml:"branch,omitempty"`
	Supersedes   string           `yaml:"supersedes,omitempty"`
	SupersededBy string           `yaml:"superseded_by,omitempty"`
	Overrides    []OverrideRecord `yaml:"overrides,omitempty"`
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

// Task is the canonical representation of a Task.
type Task struct {
	ID            string     `yaml:"id"`
	ParentFeature string     `yaml:"parent_feature"`
	Slug          string     `yaml:"slug"`
	Name          string     `yaml:"name"`
	Summary       string     `yaml:"summary"`
	Status        TaskStatus `yaml:"status"`
	Estimate      *float64   `yaml:"estimate,omitempty"`

	Assignee     string     `yaml:"assignee,omitempty"`
	DependsOn    []string   `yaml:"depends_on,omitempty"`
	FilesPlanned []string   `yaml:"files_planned,omitempty"`
	Started      *time.Time `yaml:"started,omitempty"`
	Completed    *time.Time `yaml:"completed,omitempty"`

	ClaimedAt         *time.Time `yaml:"claimed_at,omitempty"`
	DispatchedTo      string     `yaml:"dispatched_to,omitempty"`
	DispatchedAt      *time.Time `yaml:"dispatched_at,omitempty"`
	DispatchedBy      string     `yaml:"dispatched_by,omitempty"`
	CompletionSummary string     `yaml:"completion_summary,omitempty"`
	ReworkReason      string     `yaml:"rework_reason,omitempty"`

	Verification string   `yaml:"verification,omitempty"`
	Tags         []string `yaml:"tags,omitempty"`
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

// Bug is the canonical representation of a Bug.
type Bug struct {
	ID         string      `yaml:"id"`
	Slug       string      `yaml:"slug"`
	Name       string      `yaml:"name"`
	Status     BugStatus   `yaml:"status"`
	Estimate   *float64    `yaml:"estimate,omitempty"`
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
	Tags          []string `yaml:"tags,omitempty"`
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

// Decision is the canonical representation of a Decision.
type Decision struct {
	ID        string         `yaml:"id"`
	Slug      string         `yaml:"slug"`
	Name      string         `yaml:"name"`
	Summary   string         `yaml:"summary"`
	Rationale string         `yaml:"rationale"`
	DecidedBy string         `yaml:"decided_by"`
	Date      time.Time      `yaml:"date"`
	Status    DecisionStatus `yaml:"status"`

	Affects      []string `yaml:"affects,omitempty"`
	Supersedes   string   `yaml:"supersedes,omitempty"`
	SupersededBy string   `yaml:"superseded_by,omitempty"`
	Tags         []string `yaml:"tags,omitempty"`
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

// QualityEvaluation holds the result of an agent-provided quality evaluation for a document.
type QualityEvaluation struct {
	OverallScore float64            `yaml:"overall_score"`
	Pass         bool               `yaml:"pass"`
	EvaluatedAt  time.Time          `yaml:"evaluated_at"`
	Evaluator    string             `yaml:"evaluator"`
	Dimensions   map[string]float64 `yaml:"dimensions"`
}

// DocumentRecord is the metadata record for a tracked document.
// The document content stays at its canonical path; this record
// contains metadata only and is stored in .kbz/state/documents/.
type DocumentRecord struct {
	ID                string             `yaml:"id"`              // Format: {owner-id}/{slug}
	Path              string             `yaml:"path"`            // Relative path to the document file
	Type              DocumentType       `yaml:"type"`            // One of: design, specification, dev-plan, research, report, policy
	Title             string             `yaml:"title"`           // Human-readable title
	Status            DocumentStatus     `yaml:"status"`          // One of: draft, approved, superseded
	Owner             string             `yaml:"owner,omitempty"` // Parent Plan or Feature ID
	ApprovedBy        string             `yaml:"approved_by,omitempty"`
	ApprovedAt        *time.Time         `yaml:"approved_at,omitempty"`
	ContentHash       string             `yaml:"content_hash"` // SHA-256 hash of file content
	Supersedes        string             `yaml:"supersedes,omitempty"`
	SupersededBy      string             `yaml:"superseded_by,omitempty"`
	Created           time.Time          `yaml:"created"`
	CreatedBy         string             `yaml:"created_by"`
	Updated           time.Time          `yaml:"updated"`
	QualityEvaluation *QualityEvaluation `yaml:"quality_evaluation,omitempty"`
}

// GetKind returns the entity kind.
func (DocumentRecord) GetKind() EntityKind {
	return EntityKindDocument
}

// GetID returns the canonical ID.
func (d DocumentRecord) GetID() string {
	return d.ID
}

// GetSlug returns the document slug (derived from ID).
func (d DocumentRecord) GetSlug() string {
	// ID format is {owner-id}/{slug}, extract the slug part
	for i := len(d.ID) - 1; i >= 0; i-- {
		if d.ID[i] == '/' {
			return d.ID[i+1:]
		}
	}
	return d.ID
}

// Incident is the canonical representation of an Incident.
type Incident struct {
	ID               string           `yaml:"id"`
	Slug             string           `yaml:"slug"`
	Name             string           `yaml:"name"`
	Status           IncidentStatus   `yaml:"status"`
	Severity         IncidentSeverity `yaml:"severity"`
	ReportedBy       string           `yaml:"reported_by"`
	DetectedAt       time.Time        `yaml:"detected_at"`
	TriagedAt        *time.Time       `yaml:"triaged_at,omitempty"`
	MitigatedAt      *time.Time       `yaml:"mitigated_at,omitempty"`
	ResolvedAt       *time.Time       `yaml:"resolved_at,omitempty"`
	AffectedFeatures []string         `yaml:"affected_features,omitempty"`
	LinkedBugs       []string         `yaml:"linked_bugs,omitempty"`
	LinkedRCA        string           `yaml:"linked_rca,omitempty"`
	Summary          string           `yaml:"summary"`
	Created          time.Time        `yaml:"created"`
	CreatedBy        string           `yaml:"created_by"`
	Updated          time.Time        `yaml:"updated"`
}

// GetKind returns the entity kind.
func (Incident) GetKind() EntityKind {
	return EntityKindIncident
}

// GetID returns the canonical ID.
func (i Incident) GetID() string {
	return i.ID
}

// GetSlug returns the human-readable slug.
func (i Incident) GetSlug() string {
	return i.Slug
}

// IsBatchID returns true if the given ID matches the Batch/Plan ID pattern.
// IDs have the format: {X}{n}-{slug} where {X} is a single non-digit
// Unicode rune, {n} is one or more digits, and {slug} is a lowercase slug.
func IsBatchID(id string) bool {
	if len(id) < 4 { // Minimum: X1-a
		return false
	}

	// First character must be a non-digit
	runes := []rune(id)
	if len(runes) < 4 {
		return false
	}
	if unicode.IsDigit(runes[0]) {
		return false
	}

	// Find where digits start (position 1)
	digitStart := 1
	digitEnd := digitStart

	// Find extent of digits
	for digitEnd < len(runes) && unicode.IsDigit(runes[digitEnd]) {
		digitEnd++
	}

	// Must have at least one digit
	if digitEnd == digitStart {
		return false
	}

	// Must have a hyphen after digits
	if digitEnd >= len(runes) || runes[digitEnd] != '-' {
		return false
	}

	// Must have something after the hyphen
	if digitEnd+1 >= len(runes) {
		return false
	}

	return true
}

// Deprecated: use IsBatchID.
func IsPlanID(id string) bool {
	return IsBatchID(id)
}

// ParseBatchID extracts the prefix, number, and slug from a Batch/Plan ID.
// Returns empty strings if the ID is not a valid Batch/Plan ID.
func ParseBatchID(id string) (prefix string, number string, slug string) {
	if !IsBatchID(id) {
		return "", "", ""
	}

	runes := []rune(id)
	prefix = string(runes[0])

	// Find extent of digits
	digitEnd := 1
	for digitEnd < len(runes) && unicode.IsDigit(runes[digitEnd]) {
		digitEnd++
	}

	number = string(runes[1:digitEnd])
	slug = string(runes[digitEnd+1:]) // Skip the hyphen

	return prefix, number, slug
}

// Deprecated: use ParseBatchID.
func ParsePlanID(id string) (prefix string, number string, slug string) {
	return ParseBatchID(id)
}
