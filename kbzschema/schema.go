// Package kbzschema — see types.go for package documentation.
package kbzschema

import (
	"encoding/json"
	"fmt"
)

// jsonSchema is a minimal JSON Schema representation sufficient for generating
// the Kanbanzai entity schema. It is not a general-purpose JSON Schema library.
type jsonSchema struct {
	ID          string                 `json:"$id,omitempty"`
	SchemaURI   string                 `json:"$schema,omitempty"`
	Comment     string                 `json:"$comment,omitempty"`
	Title       string                 `json:"title,omitempty"`
	Description string                 `json:"description,omitempty"`
	Type        string                 `json:"type,omitempty"`
	Properties  map[string]*jsonSchema `json:"properties,omitempty"`
	Required    []string               `json:"required,omitempty"`
	Items       *jsonSchema            `json:"items,omitempty"`
	Defs        map[string]*jsonSchema `json:"$defs,omitempty"`
	Ref         string                 `json:"$ref,omitempty"`
	Enum        []string               `json:"enum,omitempty"`
	Format      string                 `json:"format,omitempty"`
}

// GenerateSchema produces a JSON Schema document covering all public entity
// types. The returned bytes contain a valid, indented JSON Schema file. The
// schema version is encoded in both the $id and the $comment field.
func GenerateSchema() ([]byte, error) {
	root := &jsonSchema{
		ID:        fmt.Sprintf("https://kanbanzai.dev/schema/%s/kanbanzai.schema.json", SchemaVersion),
		SchemaURI: "https://json-schema.org/draft/2020-12/schema",
		Comment:   "schema_version: " + SchemaVersion,
		Title:     "Kanbanzai Entity Schema",
		Description: "JSON Schema for all entity types in a Kanbanzai-managed repository " +
			"(.kbz/state/). Generated from the kbzschema Go package.",
		Defs: map[string]*jsonSchema{
			"Plan":            planSchema(),
			"Feature":         featureSchema(),
			"Task":            taskSchema(),
			"Bug":             bugSchema(),
			"Decision":        decisionSchema(),
			"DocumentRecord":  documentRecordSchema(),
			"KnowledgeEntry":  knowledgeEntrySchema(),
			"HumanCheckpoint": humanCheckpointSchema(),
			"ProjectConfig":   projectConfigSchema(),
			"PrefixEntry":     prefixEntrySchema(),
		},
	}
	return json.MarshalIndent(root, "", "  ")
}

// ────────────────────────────────────────────────────────────────────────────
// Per-type schema builders
// ────────────────────────────────────────────────────────────────────────────

func planSchema() *jsonSchema {
	return &jsonSchema{
		Type:  "object",
		Title: "Plan",
		Description: "A Plan entity record stored in .kbz/state/plans/. " +
			"Plan IDs have the form {prefix}{n}-{slug} (e.g. P1-my-plan).",
		Properties: map[string]*jsonSchema{
			"id":            strProp("Unique Plan identifier (e.g. P1-my-plan)"),
			"slug":          strProp("URL-friendly slug"),
			"title":         strProp("Human-readable title"),
			"status":        enumProp("Lifecycle status", PlanStatusProposed, PlanStatusDesigning, PlanStatusActive, PlanStatusDone, PlanStatusSuperseded, PlanStatusCancelled),
			"summary":       strProp("Brief description of the Plan"),
			"design":        strProp("Reference to design document record ID"),
			"tags":          strArrayProp("Freeform tags"),
			"created":       timestampProp("Creation timestamp (RFC 3339)"),
			"created_by":    strProp("Identity of the creator"),
			"updated":       timestampProp("Last update timestamp (RFC 3339)"),
			"supersedes":    strProp("ID of the Plan this one supersedes"),
			"superseded_by": strProp("ID of the Plan that supersedes this one"),
		},
		Required: []string{"id", "slug", "title", "status", "summary", "created", "created_by", "updated"},
	}
}

func featureSchema() *jsonSchema {
	return &jsonSchema{
		Type:        "object",
		Title:       "Feature",
		Description: "A Feature entity record stored in .kbz/state/features/.",
		Properties: map[string]*jsonSchema{
			"id":            strProp("Unique Feature identifier (FEAT-…)"),
			"slug":          strProp("URL-friendly slug"),
			"parent":        strProp("Parent Plan ID"),
			"status":        enumProp("Lifecycle status", FeatureStatusProposed, FeatureStatusDesigning, FeatureStatusSpecifying, FeatureStatusDevPlanning, FeatureStatusDeveloping, FeatureStatusDone, FeatureStatusSuperseded, FeatureStatusCancelled),
			"estimate":      numProp("Story point estimate"),
			"summary":       strProp("Brief description of the Feature"),
			"design":        strProp("Design document record ID"),
			"spec":          strProp("Specification document record ID"),
			"dev_plan":      strProp("Dev-plan document record ID"),
			"tasks":         strArrayProp("Task IDs belonging to this Feature"),
			"decisions":     strArrayProp("Decision IDs associated with this Feature"),
			"tags":          strArrayProp("Freeform tags"),
			"branch":        strProp("Git branch name for this Feature"),
			"created":       timestampProp("Creation timestamp (RFC 3339)"),
			"created_by":    strProp("Identity of the creator"),
			"updated":       timestampProp("Last update timestamp (RFC 3339)"),
			"supersedes":    strProp("ID of the Feature this one supersedes"),
			"superseded_by": strProp("ID of the Feature that supersedes this one"),
		},
		Required: []string{"id", "slug", "status", "summary", "created", "created_by"},
	}
}

func taskSchema() *jsonSchema {
	return &jsonSchema{
		Type:        "object",
		Title:       "Task",
		Description: "A Task entity record stored in .kbz/state/tasks/.",
		Properties: map[string]*jsonSchema{
			"id":                 strProp("Unique Task identifier (TASK-…)"),
			"parent_feature":     strProp("Parent Feature ID"),
			"slug":               strProp("URL-friendly slug"),
			"summary":            strProp("Brief description of the Task"),
			"status":             enumProp("Lifecycle status", TaskStatusQueued, TaskStatusReady, TaskStatusActive, TaskStatusBlocked, TaskStatusNeedsReview, TaskStatusNeedsRework, TaskStatusDone, TaskStatusNotPlanned, TaskStatusDuplicate),
			"estimate":           numProp("Story point estimate"),
			"assignee":           strProp("Assigned agent or user identity"),
			"depends_on":         strArrayProp("Task IDs this Task depends on"),
			"files_planned":      strArrayProp("File paths planned to be modified"),
			"started":            timestampProp("Start timestamp (RFC 3339)"),
			"completed":          timestampProp("Completion timestamp (RFC 3339)"),
			"claimed_at":         timestampProp("Claim timestamp (RFC 3339)"),
			"dispatched_to":      strProp("Agent the Task was dispatched to"),
			"dispatched_at":      timestampProp("Dispatch timestamp (RFC 3339)"),
			"dispatched_by":      strProp("Orchestrator that dispatched the Task"),
			"completion_summary": strProp("Summary written at completion"),
			"rework_reason":      strProp("Reason the Task was sent for rework"),
			"verification":       strProp("Verification steps or criteria"),
			"tags":               strArrayProp("Freeform tags"),
		},
		Required: []string{"id", "parent_feature", "slug", "summary", "status"},
	}
}

func bugSchema() *jsonSchema {
	return &jsonSchema{
		Type:        "object",
		Title:       "Bug",
		Description: "A Bug entity record stored in .kbz/state/bugs/.",
		Properties: map[string]*jsonSchema{
			"id":             strProp("Unique Bug identifier (BUG-…)"),
			"slug":           strProp("URL-friendly slug"),
			"title":          strProp("Human-readable bug title"),
			"status":         enumProp("Lifecycle status", BugStatusReported, BugStatusTriaged, BugStatusReproduced, BugStatusPlanned, BugStatusInProgress, BugStatusNeedsReview, BugStatusNeedsRework, BugStatusVerified, BugStatusClosed, BugStatusDuplicate, BugStatusNotPlanned, BugStatusCannotReproduce),
			"estimate":       numProp("Story point estimate"),
			"severity":       enumProp("Bug severity", SeverityLow, SeverityMedium, SeverityHigh, SeverityCritical),
			"priority":       enumProp("Bug priority", PriorityLow, PriorityMedium, PriorityHigh, PriorityCritical),
			"type":           enumProp("Bug classification", BugTypeImplementationDefect, BugTypeSpecificationDefect, BugTypeDesignProblem),
			"reported_by":    strProp("Identity of the reporter"),
			"reported":       timestampProp("Report timestamp (RFC 3339)"),
			"observed":       strProp("Description of the observed (incorrect) behaviour"),
			"expected":       strProp("Description of the expected (correct) behaviour"),
			"affects":        strArrayProp("Feature or entity IDs affected"),
			"origin_feature": strProp("Feature ID where the bug was introduced"),
			"origin_task":    strProp("Task ID where the bug was introduced"),
			"environment":    strProp("Environment description"),
			"reproduction":   strProp("Steps to reproduce"),
			"duplicate_of":   strProp("ID of the canonical duplicate Bug"),
			"fixed_by":       strProp("Commit or PR that fixed the bug"),
			"verified_by":    strProp("Identity of the verifier"),
			"release_target": strProp("Target release for the fix"),
			"tags":           strArrayProp("Freeform tags"),
		},
		Required: []string{"id", "slug", "title", "status", "severity", "priority", "type", "reported_by", "reported", "observed", "expected"},
	}
}

func decisionSchema() *jsonSchema {
	return &jsonSchema{
		Type:        "object",
		Title:       "Decision",
		Description: "A Decision entity record stored in .kbz/state/decisions/.",
		Properties: map[string]*jsonSchema{
			"id":            strProp("Unique Decision identifier (DEC-…)"),
			"slug":          strProp("URL-friendly slug"),
			"summary":       strProp("Brief summary of the decision"),
			"rationale":     strProp("Rationale and reasoning"),
			"decided_by":    strProp("Identity of the decision maker"),
			"date":          timestampProp("Decision timestamp (RFC 3339)"),
			"status":        enumProp("Lifecycle status", DecisionStatusProposed, DecisionStatusAccepted, DecisionStatusRejected, DecisionStatusSuperseded),
			"affects":       strArrayProp("Entity IDs affected by this decision"),
			"supersedes":    strProp("ID of the Decision this one supersedes"),
			"superseded_by": strProp("ID of the Decision that supersedes this one"),
			"tags":          strArrayProp("Freeform tags"),
		},
		Required: []string{"id", "slug", "summary", "rationale", "decided_by", "date", "status"},
	}
}

func documentRecordSchema() *jsonSchema {
	return &jsonSchema{
		Type:        "object",
		Title:       "DocumentRecord",
		Description: "A document metadata record stored in .kbz/state/documents/.",
		Properties: map[string]*jsonSchema{
			"id":            strProp("Document record ID in the form {owner}/{slug}"),
			"path":          strProp("Relative path to the document file from the repository root"),
			"type":          enumProp("Document type", DocTypeDesign, DocTypeSpecification, DocTypeDevPlan, DocTypeResearch, DocTypeReport, DocTypePolicy, DocTypeRCA),
			"title":         strProp("Human-readable document title"),
			"status":        enumProp("Document status", DocStatusDraft, DocStatusApproved, DocStatusSuperseded),
			"owner":         strProp("Owning Plan or Feature ID"),
			"approved_by":   strProp("Identity of the approver"),
			"approved_at":   timestampProp("Approval timestamp (RFC 3339)"),
			"content_hash":  strProp("SHA-256 hex digest of the document file content at last registration"),
			"supersedes":    strProp("ID of the DocumentRecord this one supersedes"),
			"superseded_by": strProp("ID of the DocumentRecord that supersedes this one"),
			"created":       timestampProp("Creation timestamp (RFC 3339)"),
			"created_by":    strProp("Identity of the creator"),
			"updated":       timestampProp("Last update timestamp (RFC 3339)"),
		},
		Required: []string{"id", "path", "type", "title", "status", "content_hash", "created", "created_by", "updated"},
	}
}

func knowledgeEntrySchema() *jsonSchema {
	return &jsonSchema{
		Type:        "object",
		Title:       "KnowledgeEntry",
		Description: "A knowledge entry record stored in .kbz/state/knowledge/.",
		Properties: map[string]*jsonSchema{
			"id":                strProp("Unique knowledge entry identifier (KE-…)"),
			"tier":              &jsonSchema{Type: "integer", Description: "Knowledge tier (2 = project-level, 3 = session-level)", Enum: []string{"2", "3"}},
			"topic":             strProp("Normalised topic identifier"),
			"scope":             strProp("Scope of the entry (profile name or 'project')"),
			"content":           strProp("Concise, actionable statement of the knowledge"),
			"learned_from":      strProp("Provenance: Task ID or other reference"),
			"status":            enumProp("Lifecycle status", KnowledgeStatusContributed, KnowledgeStatusConfirmed, KnowledgeStatusDisputed, KnowledgeStatusStale, KnowledgeStatusRetired),
			"use_count":         &jsonSchema{Type: "integer", Description: "Number of times this entry was used by an agent"},
			"miss_count":        &jsonSchema{Type: "integer", Description: "Number of times this entry was flagged as incorrect"},
			"confidence":        &jsonSchema{Type: "number", Description: "Confidence score in [0, 1] computed from use/miss counts"},
			"last_used":         timestampProp("Timestamp of last use (RFC 3339)"),
			"ttl_days":          &jsonSchema{Type: "integer", Description: "Time-to-live in days (0 = exempt)"},
			"promoted_from":     strProp("Original entry ID if this was promoted from tier 3"),
			"merged_from":       strArrayProp("Entry IDs that were merged into this one"),
			"deprecated_reason": strProp("Reason for deprecation or retirement"),
			"git_anchors":       strArrayProp("File paths that anchor this entry for staleness detection"),
			"tags":              strArrayProp("Freeform tags"),
			"created":           timestampProp("Creation timestamp (RFC 3339)"),
			"created_by":        strProp("Identity of the contributor"),
			"updated":           timestampProp("Last update timestamp (RFC 3339)"),
		},
		Required: []string{"id", "tier", "topic", "scope", "content", "status", "use_count", "miss_count", "confidence", "ttl_days", "created", "created_by", "updated"},
	}
}

func humanCheckpointSchema() *jsonSchema {
	return &jsonSchema{
		Type:        "object",
		Title:       "HumanCheckpoint",
		Description: "A human checkpoint record stored in .kbz/state/checkpoints/.",
		Properties: map[string]*jsonSchema{
			"id":                    strProp("Unique checkpoint identifier (CHK-…)"),
			"question":              strProp("The question or decision requiring human input"),
			"context":               strProp("Background information for the human"),
			"orchestration_summary": strProp("State of the orchestration session at checkpoint time"),
			"status":                enumProp("Checkpoint status", CheckpointStatusPending, CheckpointStatusResponded),
			"created_at":            timestampProp("Creation timestamp (RFC 3339)"),
			"created_by":            strProp("Identity of the orchestrating agent"),
			"responded_at":          timestampProp("Response timestamp (RFC 3339); absent until responded"),
			"response":              strProp("Human's answer or decision; absent until responded"),
		},
		Required: []string{"id", "question", "context", "orchestration_summary", "status", "created_at", "created_by"},
	}
}

func projectConfigSchema() *jsonSchema {
	return &jsonSchema{
		Type:        "object",
		Title:       "ProjectConfig",
		Description: "Project configuration stored in .kbz/config.yaml.",
		Properties: map[string]*jsonSchema{
			"version":        strProp("Config format version (legacy field; preserved for backwards compatibility)"),
			"schema_version": strProp("Public schema version in MAJOR.MINOR.PATCH format (e.g. '1.0.0')"),
			"prefixes":       {Type: "array", Items: &jsonSchema{Ref: "#/$defs/PrefixEntry"}, Description: "Plan ID prefix registry"},
		},
		Required: []string{"version", "prefixes"},
	}
}

func prefixEntrySchema() *jsonSchema {
	return &jsonSchema{
		Type:        "object",
		Title:       "PrefixEntry",
		Description: "A single entry in the Plan ID prefix registry.",
		Properties: map[string]*jsonSchema{
			"prefix":      strProp("Single non-digit character used as Plan ID prefix"),
			"name":        strProp("Human-readable name for the prefix"),
			"description": strProp("Optional longer description of the prefix purpose"),
			"retired":     {Type: "boolean", Description: "Whether this prefix is retired (no longer used for new Plans)"},
		},
		Required: []string{"prefix", "name"},
	}
}

// ────────────────────────────────────────────────────────────────────────────
// Schema property helpers
// ────────────────────────────────────────────────────────────────────────────

func strProp(description string) *jsonSchema {
	return &jsonSchema{Type: "string", Description: description}
}

func timestampProp(description string) *jsonSchema {
	return &jsonSchema{Type: "string", Format: "date-time", Description: description}
}

func numProp(description string) *jsonSchema {
	return &jsonSchema{Type: "number", Description: description}
}

func strArrayProp(description string) *jsonSchema {
	return &jsonSchema{
		Type:        "array",
		Description: description,
		Items:       &jsonSchema{Type: "string"},
	}
}

func enumProp(description string, values ...string) *jsonSchema {
	return &jsonSchema{
		Type:        "string",
		Description: description,
		Enum:        values,
	}
}
