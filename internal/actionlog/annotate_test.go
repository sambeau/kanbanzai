package actionlog

import (
	"context"
	"testing"
)

// AC-006: Annotation constants are defined in the actionlog package with expected values.
func TestAnnotationConstants(t *testing.T) {
	t.Parallel()

	if AnnotationResultCount != "result_count" {
		t.Errorf("AnnotationResultCount: got %q, want %q", AnnotationResultCount, "result_count")
	}
	if AnnotationKBRejections != "kb_rejections" {
		t.Errorf("AnnotationKBRejections: got %q, want %q", AnnotationKBRejections, "kb_rejections")
	}
	if AnnotationEntityType != "entity_type" {
		t.Errorf("AnnotationEntityType: got %q, want %q", AnnotationEntityType, "entity_type")
	}
	if AnnotationDocType != "doc_type" {
		t.Errorf("AnnotationDocType: got %q, want %q", AnnotationDocType, "doc_type")
	}
}

func TestAnnotateEntry_SetsAndRetrieves(t *testing.T) {
	t.Parallel()

	ctx := contextWithCollector(context.Background())
	AnnotateEntry(ctx, AnnotationResultCount, "42")
	AnnotateEntry(ctx, AnnotationEntityType, "feature")

	pairs := drainCollector(ctx)
	if pairs == nil {
		t.Fatal("drainCollector returned nil")
	}
	if pairs[AnnotationResultCount] != "42" {
		t.Errorf("result_count: got %q, want %q", pairs[AnnotationResultCount], "42")
	}
	if pairs[AnnotationEntityType] != "feature" {
		t.Errorf("entity_type: got %q, want %q", pairs[AnnotationEntityType], "feature")
	}
}

func TestAnnotateEntry_NoCollector_NoOp(t *testing.T) {
	t.Parallel()

	// Should not panic when no collector is on the context.
	AnnotateEntry(context.Background(), "k", "v")
}

func TestAnnotateEntry_NilContext_NoOp(t *testing.T) {
	t.Parallel()

	// nil context should not panic.
	AnnotateEntry(nil, "k", "v") //nolint:staticcheck // intentionally testing nil
}

func TestAnnotateEntry_DuplicateKeyOverwrites(t *testing.T) {
	t.Parallel()

	ctx := contextWithCollector(context.Background())
	AnnotateEntry(ctx, "key", "first")
	AnnotateEntry(ctx, "key", "second")

	pairs := drainCollector(ctx)
	if pairs["key"] != "second" {
		t.Errorf("key: got %q, want %q", pairs["key"], "second")
	}
}

func TestDrainCollector_NoCollector_ReturnsNil(t *testing.T) {
	t.Parallel()

	if pairs := drainCollector(context.Background()); pairs != nil {
		t.Errorf("drainCollector without collector: got %v, want nil", pairs)
	}
}

func TestDrainCollector_EmptyCollector_ReturnsNil(t *testing.T) {
	t.Parallel()

	ctx := contextWithCollector(context.Background())
	// Drain without any annotations.
	pairs := drainCollector(ctx)
	if pairs != nil {
		t.Errorf("drainCollector on empty collector: got %v, want nil", pairs)
	}
}

func TestDrainCollector_ConsumesPairs(t *testing.T) {
	t.Parallel()

	ctx := contextWithCollector(context.Background())
	AnnotateEntry(ctx, "k", "v")

	_ = drainCollector(ctx)

	// Second drain should return nil — pairs are consumed.
	if pairs := drainCollector(ctx); pairs != nil {
		t.Errorf("second drainCollector: got %v, want nil", pairs)
	}
}
