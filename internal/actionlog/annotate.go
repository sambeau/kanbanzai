package actionlog

import "context"

// Annotation key constants for well-known entry annotations.
const (
	AnnotationResultCount  = "result_count"
	AnnotationKBRejections = "kb_rejections"
	AnnotationEntityType   = "entity_type"
	AnnotationDocType      = "doc_type"
)

// ctxKey is an unexported type for context keys to avoid collisions.
type ctxKey struct{ name string }

var collectorKey = ctxKey{name: "actionlog-annotations"}

// annotationCollector carries key-value annotations placed on the context
// during a tool invocation. It is not safe for concurrent use under the
// assumption that annotations happen within the handler goroutine.
type annotationCollector struct {
	pairs map[string]string
}

// AnnotateEntry records a key-value annotation on ctx. Annotations are merged
// into Entry.Extra when the Hook writes the log entry. Duplicate keys are
// overwritten (last-write wins). A nil or no-op context is safe: the call
// is silently ignored if no collector is present.
func AnnotateEntry(ctx context.Context, key, value string) {
	if ctx == nil {
		return
	}
	c, ok := ctx.Value(collectorKey).(*annotationCollector)
	if !ok || c == nil {
		return
	}
	if c.pairs == nil {
		c.pairs = make(map[string]string)
	}
	c.pairs[key] = value
}

// contextWithCollector returns a child context carrying a new annotationCollector.
func contextWithCollector(ctx context.Context) context.Context {
	return context.WithValue(ctx, collectorKey, &annotationCollector{})
}

// drainCollector extracts and returns the annotations from ctx, consuming them.
// Returns nil if no collector is present or it has no entries.
func drainCollector(ctx context.Context) map[string]string {
	c, ok := ctx.Value(collectorKey).(*annotationCollector)
	if !ok || c == nil || len(c.pairs) == 0 {
		return nil
	}
	pairs := c.pairs
	c.pairs = nil
	return pairs
}
