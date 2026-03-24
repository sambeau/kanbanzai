package model

// KnowledgeStatus is the lifecycle state for a KnowledgeEntry.
type KnowledgeStatus string

const (
	KnowledgeStatusContributed KnowledgeStatus = "contributed"
	KnowledgeStatusConfirmed   KnowledgeStatus = "confirmed"
	KnowledgeStatusDisputed    KnowledgeStatus = "disputed"
	KnowledgeStatusStale       KnowledgeStatus = "stale"
	KnowledgeStatusRetired     KnowledgeStatus = "retired"
)

// EntityKindKnowledgeEntry is the entity kind for a KnowledgeEntry.
const EntityKindKnowledgeEntry EntityKind = "knowledge_entry"
