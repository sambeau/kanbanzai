// Package actionlog records MCP tool invocations to date-partitioned JSONL
// files under .kbz/logs/. It provides the entry model, a thread-safe writer,
// entity-ID extraction helpers, a logging hook for the MCP server, log
// cleanup, and metrics aggregation over the collected data.
package actionlog

// Entry represents one MCP tool invocation log record.
// Scalar fields are always present in JSON output (nulls are not omitted).
// Map fields use omitempty to keep logs compact when empty.
type Entry struct {
	Timestamp     string            `json:"timestamp"` // RFC 3339
	Tool          string            `json:"tool"`
	Action        *string           `json:"action"`    // null if tool has no action param
	EntityID      *string           `json:"entity_id"` // null if no entity referenced
	Stage         *string           `json:"stage"`     // null if no entity/no parent feature
	ServerVersion string            `json:"server_version,omitempty"`
	Success       bool              `json:"success"`
	ErrorType     *string           `json:"error_type"` // null if success
	Extra         map[string]string `json:"extra,omitempty"`
}

// Error type constants classify failures captured in Entry.ErrorType.
const (
	ErrorGateFailure       = "gate_failure"
	ErrorValidationError   = "validation_error"
	ErrorNotFound          = "not_found"
	ErrorPreconditionError = "precondition_error"
	ErrorInternalError     = "internal_error"
)
