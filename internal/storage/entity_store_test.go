package storage

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestMarshalCanonicalYAML_DeterministicKeyOrder(t *testing.T) {
	t.Parallel()

	fields := map[string]any{
		"summary": "Kernel storage",
		"id":      "FEAT-001",
		"slug":    "kernel-storage",
		"status":  "draft",
		"tags":    []string{"phase-1", "storage"},
		"meta": map[string]any{
			"owner":   "sam",
			"created": "2026-03-19T00:00:00Z",
		},
	}

	got, err := MarshalCanonicalYAML("feature", fields)
	if err != nil {
		t.Fatalf("MarshalCanonicalYAML() error = %v", err)
	}

	want := "" +
		"id: FEAT-001\n" +
		"slug: kernel-storage\n" +
		"status: draft\n" +
		"summary: Kernel storage\n" +
		"meta:\n" +
		"  created: \"2026-03-19T00:00:00Z\"\n" +
		"  owner: sam\n" +
		"tags:\n" +
		"  - phase-1\n" +
		"  - storage\n"

	if got != want {
		t.Fatalf("MarshalCanonicalYAML() mismatch\nwant:\n%s\ngot:\n%s", want, got)
	}
}

func TestCanonicalYAML_RoundTrip(t *testing.T) {
	t.Parallel()

	input := map[string]any{
		"id":      "BUG-001",
		"slug":    "yaml-round-trip",
		"title":   "Round-trip preserves structure",
		"status":  "reported",
		"quoted":  "needs:quotes",
		"empty":   "",
		"enabled": true,
		"labels":  []string{"storage", "phase-1"},
		"details": map[string]any{
			"observed": "writer output differs",
			"expected": "stable canonical representation",
		},
	}

	content, err := MarshalCanonicalYAML("bug", input)
	if err != nil {
		t.Fatalf("MarshalCanonicalYAML() error = %v", err)
	}

	got, err := UnmarshalCanonicalYAML(content)
	if err != nil {
		t.Fatalf("UnmarshalCanonicalYAML() error = %v", err)
	}

	want := map[string]any{
		"id":      "BUG-001",
		"slug":    "yaml-round-trip",
		"title":   "Round-trip preserves structure",
		"status":  "reported",
		"quoted":  "needs:quotes",
		"empty":   "",
		"enabled": true,
		"labels":  []any{"storage", "phase-1"},
		"details": map[string]any{
			"observed": "writer output differs",
			"expected": "stable canonical representation",
		},
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("round-trip mismatch\nwant: %#v\ngot:  %#v", want, got)
	}
}

func TestEntityStore_WriteAndLoad(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := NewEntityStore(root)

	record := EntityRecord{
		Type: "feature",
		ID:   "FEAT-001",
		Slug: "initial-kernel",
		Fields: map[string]any{
			"id":         "FEAT-001",
			"slug":       "initial-kernel",
			"epic":       "E-001",
			"status":     "draft",
			"summary":    "Start the workflow kernel",
			"created":    "2026-03-19T00:00:00Z",
			"created_by": "sam",
			"tasks":      []string{"FEAT-001.1", "FEAT-001.2"},
		},
	}

	path, err := store.Write(record)
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	wantPath := filepath.Join(root, "features", "FEAT-001-initial-kernel.yaml")
	if path != wantPath {
		t.Fatalf("Write() path mismatch: want %q, got %q", wantPath, path)
	}

	got, err := store.Load("feature", "FEAT-001", "initial-kernel")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	want := map[string]any{
		"created":    "2026-03-19T00:00:00Z",
		"created_by": "sam",
		"epic":       "E-001",
		"id":         "FEAT-001",
		"slug":       "initial-kernel",
		"status":     "draft",
		"summary":    "Start the workflow kernel",
		"tasks":      []any{"FEAT-001.1", "FEAT-001.2"},
	}

	if got.Type != "feature" {
		t.Fatalf("Load() type mismatch: want %q, got %q", "feature", got.Type)
	}
	if got.ID != "FEAT-001" {
		t.Fatalf("Load() id mismatch: want %q, got %q", "FEAT-001", got.ID)
	}
	if got.Slug != "initial-kernel" {
		t.Fatalf("Load() slug mismatch: want %q, got %q", "initial-kernel", got.Slug)
	}
	if !reflect.DeepEqual(got.Fields, want) {
		t.Fatalf("Load() fields mismatch\nwant: %#v\ngot:  %#v", want, got.Fields)
	}
}

func TestEntityStore_Write_RejectsMismatchedIdentity(t *testing.T) {
	t.Parallel()

	store := NewEntityStore(t.TempDir())

	record := EntityRecord{
		Type: "epic",
		ID:   "E-001",
		Slug: "phase-1",
		Fields: map[string]any{
			"id":      "E-999",
			"slug":    "phase-1",
			"title":   "Phase 1",
			"status":  "proposed",
			"summary": "Build the kernel",
		},
	}

	if _, err := store.Write(record); err == nil {
		t.Fatal("Write() error = nil, want mismatch error")
	}
}

func TestUnmarshalCanonicalYAML_RejectsUnexpectedIndentation(t *testing.T) {
	t.Parallel()

	content := "" +
		"id: FEAT-001\n" +
		"  slug: bad-indent\n"

	if _, err := UnmarshalCanonicalYAML(content); err == nil {
		t.Fatal("UnmarshalCanonicalYAML() error = nil, want indentation error")
	}
}

func TestCanonicalYAML_FixturesRoundTrip(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		path       string
		entityType string
	}{
		{name: "epic", path: filepath.Join("..", "..", "testdata", "entities", "epic.yaml"), entityType: "epic"},
		{name: "feature", path: filepath.Join("..", "..", "testdata", "entities", "feature.yaml"), entityType: "feature"},
		{name: "task", path: filepath.Join("..", "..", "testdata", "entities", "task.yaml"), entityType: "task"},
		{name: "bug", path: filepath.Join("..", "..", "testdata", "entities", "bug.yaml"), entityType: "bug"},
		{name: "decision", path: filepath.Join("..", "..", "testdata", "entities", "decision.yaml"), entityType: "decision"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			content, err := os.ReadFile(tt.path)
			if err != nil {
				t.Fatalf("ReadFile() error = %v", err)
			}

			fields, err := UnmarshalCanonicalYAML(string(content))
			if err != nil {
				t.Fatalf("UnmarshalCanonicalYAML() error = %v", err)
			}

			got, err := MarshalCanonicalYAML(tt.entityType, fields)
			if err != nil {
				t.Fatalf("MarshalCanonicalYAML() error = %v", err)
			}

			if got != string(content) {
				t.Fatalf("fixture round-trip mismatch\nwant:\n%s\ngot:\n%s", string(content), got)
			}
		})
	}
}

func TestEntityStore_Load_FixtureFiles(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := NewEntityStore(root)

	fixtures := []struct {
		name         string
		sourcePath   string
		entityType   string
		id           string
		slug         string
		wantFields   map[string]any
		wantFilePath string
	}{
		{
			name:       "epic",
			sourcePath: filepath.Join("..", "..", "testdata", "entities", "epic.yaml"),
			entityType: "epic",
			id:         "E-001",
			slug:       "phase-1-kernel",
			wantFields: map[string]any{
				"id":         "E-001",
				"slug":       "phase-1-kernel",
				"title":      "Phase 1 Kernel",
				"status":     "proposed",
				"summary":    "Build the initial workflow kernel",
				"created":    "2026-03-19T12:00:00Z",
				"created_by": "sam",
				"features":   []any{"FEAT-001", "FEAT-002"},
			},
			wantFilePath: filepath.Join(root, "epics", "E-001-phase-1-kernel.yaml"),
		},
		{
			name:       "feature",
			sourcePath: filepath.Join("..", "..", "testdata", "entities", "feature.yaml"),
			entityType: "feature",
			id:         "FEAT-001",
			slug:       "initial-kernel",
			wantFields: map[string]any{
				"id":         "FEAT-001",
				"slug":       "initial-kernel",
				"epic":       "E-001",
				"status":     "draft",
				"summary":    "Start the workflow kernel",
				"created":    "2026-03-19T12:00:00Z",
				"created_by": "sam",
				"spec":       "work/spec/phase-1-specification.md",
				"plan":       "work/plan/phase-1-implementation-plan.md",
				"tasks":      []any{"FEAT-001.1", "FEAT-001.2"},
				"decisions":  []any{"DEC-001"},
				"branch":     "feat/feat-001-initial-kernel",
			},
			wantFilePath: filepath.Join(root, "features", "FEAT-001-initial-kernel.yaml"),
		},
		{
			name:       "task",
			sourcePath: filepath.Join("..", "..", "testdata", "entities", "task.yaml"),
			entityType: "task",
			id:         "FEAT-001.1",
			slug:       "write-entity-files",
			wantFields: map[string]any{
				"id":            "FEAT-001.1",
				"feature":       "FEAT-001",
				"slug":          "write-entity-files",
				"summary":       "Write canonical entity files to disk",
				"status":        "queued",
				"assignee":      "sam",
				"depends_on":    []any{"FEAT-001.0"},
				"files_planned": []any{"internal/storage/entity_store.go", "internal/storage/entity_store_test.go"},
				"verification":  "go test ./...",
			},
			wantFilePath: filepath.Join(root, "tasks", "FEAT-001.1-write-entity-files.yaml"),
		},
		{
			name:       "bug",
			sourcePath: filepath.Join("..", "..", "testdata", "entities", "bug.yaml"),
			entityType: "bug",
			id:         "BUG-001",
			slug:       "bad-yaml-output",
			wantFields: map[string]any{
				"id":          "BUG-001",
				"slug":        "bad-yaml-output",
				"title":       "Writer produces unstable YAML",
				"status":      "reported",
				"severity":    "medium",
				"priority":    "medium",
				"type":        "implementation-defect",
				"reported_by": "sam",
				"reported":    "2026-03-19T12:00:00Z",
				"observed":    "Repeated writes produce different output",
				"expected":    "Repeated writes should be stable",
			},
			wantFilePath: filepath.Join(root, "bugs", "BUG-001-bad-yaml-output.yaml"),
		},
		{
			name:       "decision",
			sourcePath: filepath.Join("..", "..", "testdata", "entities", "decision.yaml"),
			entityType: "decision",
			id:         "DEC-001",
			slug:       "strict-yaml-subset",
			wantFields: map[string]any{
				"id":         "DEC-001",
				"slug":       "strict-yaml-subset",
				"summary":    "Use a strict canonical YAML subset",
				"rationale":  "Deterministic output is required for Git-friendly state",
				"decided_by": "sam",
				"date":       "2026-03-19T12:00:00Z",
				"status":     "proposed",
			},
			wantFilePath: filepath.Join(root, "decisions", "DEC-001-strict-yaml-subset.yaml"),
		},
	}

	for _, fixture := range fixtures {
		fixture := fixture
		t.Run(fixture.name, func(t *testing.T) {
			t.Parallel()

			content, err := os.ReadFile(fixture.sourcePath)
			if err != nil {
				t.Fatalf("ReadFile() error = %v", err)
			}

			if err := os.MkdirAll(filepath.Dir(fixture.wantFilePath), 0o755); err != nil {
				t.Fatalf("MkdirAll() error = %v", err)
			}
			if err := os.WriteFile(fixture.wantFilePath, content, 0o644); err != nil {
				t.Fatalf("WriteFile() error = %v", err)
			}

			got, err := store.Load(fixture.entityType, fixture.id, fixture.slug)
			if err != nil {
				t.Fatalf("Load() error = %v", err)
			}

			if got.Type != fixture.entityType {
				t.Fatalf("Load() type = %q, want %q", got.Type, fixture.entityType)
			}
			if got.ID != fixture.id {
				t.Fatalf("Load() id = %q, want %q", got.ID, fixture.id)
			}
			if got.Slug != fixture.slug {
				t.Fatalf("Load() slug = %q, want %q", got.Slug, fixture.slug)
			}
			if !reflect.DeepEqual(got.Fields, fixture.wantFields) {
				t.Fatalf("Load() fields mismatch\nwant: %#v\ngot:  %#v", fixture.wantFields, got.Fields)
			}
		})
	}
}

func TestMarshalCanonicalYAML_NestedMapsInsideListItems(t *testing.T) {
	t.Parallel()

	fields := map[string]any{
		"id":   "BUG-001",
		"slug": "nested-list-maps",
		"steps": []any{
			map[string]any{
				"action": "open app",
				"meta": map[string]any{
					"attempt": 1,
					"owner":   "sam",
				},
			},
			map[string]any{
				"action": "click save",
				"meta": map[string]any{
					"attempt": 2,
					"owner":   "sam",
				},
			},
		},
	}

	got, err := MarshalCanonicalYAML("bug", fields)
	if err != nil {
		t.Fatalf("MarshalCanonicalYAML() error = %v", err)
	}

	want := "" +
		"id: BUG-001\n" +
		"slug: nested-list-maps\n" +
		"steps:\n" +
		"  -\n" +
		"    action: open app\n" +
		"    meta:\n" +
		"      attempt: 1\n" +
		"      owner: sam\n" +
		"  -\n" +
		"    action: click save\n" +
		"    meta:\n" +
		"      attempt: 2\n" +
		"      owner: sam\n"

	if got != want {
		t.Fatalf("MarshalCanonicalYAML() mismatch\nwant:\n%s\ngot:\n%s", want, got)
	}
}

func TestMarshalCanonicalYAML_SerializesNilValue(t *testing.T) {
	t.Parallel()

	fields := map[string]any{
		"id":      "BUG-001",
		"slug":    "nil-value",
		"comment": nil,
	}

	got, err := MarshalCanonicalYAML("bug", fields)
	if err != nil {
		t.Fatalf("MarshalCanonicalYAML() error = %v", err)
	}

	want := "" +
		"id: BUG-001\n" +
		"slug: nil-value\n" +
		"comment: null\n"

	if got != want {
		t.Fatalf("MarshalCanonicalYAML() mismatch\nwant:\n%s\ngot:\n%s", want, got)
	}
}

func TestUnmarshalCanonicalYAML_ParsesNumericScalars(t *testing.T) {
	t.Parallel()

	content := "" +
		"id: BUG-001\n" +
		"slug: numeric-scalars\n" +
		"attempts: 2\n" +
		"ratio: 3.5\n" +
		"items:\n" +
		"  - 1\n" +
		"  - 2.25\n"

	got, err := UnmarshalCanonicalYAML(content)
	if err != nil {
		t.Fatalf("UnmarshalCanonicalYAML() error = %v", err)
	}

	want := map[string]any{
		"id":       "BUG-001",
		"slug":     "numeric-scalars",
		"attempts": 2,
		"ratio":    3.5,
		"items":    []any{1, 2.25},
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("UnmarshalCanonicalYAML() mismatch\nwant: %#v\ngot:  %#v", want, got)
	}
}

func TestMarshalCanonicalYAML_UsesStringer(t *testing.T) {
	t.Parallel()

	fields := map[string]any{
		"id":      "BUG-001",
		"slug":    "stringer-value",
		"comment": fixtureStringer("needs:quotes"),
	}

	got, err := MarshalCanonicalYAML("bug", fields)
	if err != nil {
		t.Fatalf("MarshalCanonicalYAML() error = %v", err)
	}

	want := "" +
		"id: BUG-001\n" +
		"slug: stringer-value\n" +
		"comment: \"needs:quotes\"\n"

	if got != want {
		t.Fatalf("MarshalCanonicalYAML() mismatch\nwant:\n%s\ngot:\n%s", want, got)
	}
}

func TestMarshalCanonicalYAML_IdempotentWrite(t *testing.T) {
	t.Parallel()

	fields := map[string]any{
		"id":       "BUG-001",
		"slug":     "idempotent-write",
		"title":    "Idempotent output",
		"status":   "reported",
		"severity": "medium",
		"priority": "medium",
		"type":     "implementation-defect",
		"details": map[string]any{
			"attempts": 2,
			"owner":    "sam",
		},
		"labels": []any{"storage", "phase-1"},
	}

	first, err := MarshalCanonicalYAML("bug", fields)
	if err != nil {
		t.Fatalf("first MarshalCanonicalYAML() error = %v", err)
	}

	parsed, err := UnmarshalCanonicalYAML(first)
	if err != nil {
		t.Fatalf("UnmarshalCanonicalYAML() error = %v", err)
	}

	second, err := MarshalCanonicalYAML("bug", parsed)
	if err != nil {
		t.Fatalf("second MarshalCanonicalYAML() error = %v", err)
	}

	if second != first {
		t.Fatalf("MarshalCanonicalYAML() not idempotent\nfirst:\n%s\nsecond:\n%s", first, second)
	}
}

func TestMarshalCanonicalYAML_BackslashEscapingRoundTrip(t *testing.T) {
	t.Parallel()

	fields := map[string]any{
		"id":      "BUG-001",
		"slug":    "backslash-round-trip",
		"comment": `C:\temp\kbz\state`,
	}

	content, err := MarshalCanonicalYAML("bug", fields)
	if err != nil {
		t.Fatalf("MarshalCanonicalYAML() error = %v", err)
	}

	got, err := UnmarshalCanonicalYAML(content)
	if err != nil {
		t.Fatalf("UnmarshalCanonicalYAML() error = %v", err)
	}

	if got["comment"] != `C:\temp\kbz\state` {
		t.Fatalf("backslash round-trip mismatch: got %v", got["comment"])
	}
}

func TestUnmarshalCanonicalYAML_ParsesBareListItems(t *testing.T) {
	t.Parallel()

	// Bare `-` at end of input (i+1 >= len(lines) branch)
	// and bare `-` followed by same-indent line (nextIndent <= indent branch)
	content := "" +
		"id: BUG-001\n" +
		"slug: bare-list\n" +
		"steps:\n" +
		"  -\n" +
		"    action: first\n" +
		"  -\n" +
		"  -\n"

	got, err := UnmarshalCanonicalYAML(content)
	if err != nil {
		t.Fatalf("UnmarshalCanonicalYAML() error = %v", err)
	}

	want := map[string]any{
		"id":   "BUG-001",
		"slug": "bare-list",
		"steps": []any{
			map[string]any{"action": "first"},
			map[string]any{},
			map[string]any{},
		},
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("UnmarshalCanonicalYAML() mismatch\nwant: %#v\ngot:  %#v", want, got)
	}
}

func TestUnmarshalCanonicalYAML_RejectsInvalidListItem(t *testing.T) {
	t.Parallel()

	content := "" +
		"id: BUG-001\n" +
		"slug: bad-list\n" +
		"items:\n" +
		"  - good\n" +
		"  not-a-list-item\n"

	if _, err := UnmarshalCanonicalYAML(content); err == nil {
		t.Fatal("UnmarshalCanonicalYAML() error = nil, want invalid list item error")
	}
}

func TestUnmarshalCanonicalYAML_RejectsUnexpectedListIndentation(t *testing.T) {
	t.Parallel()

	content := "" +
		"id: BUG-001\n" +
		"slug: bad-indent\n" +
		"items:\n" +
		"  - good\n" +
		"      - over-indented\n"

	if _, err := UnmarshalCanonicalYAML(content); err == nil {
		t.Fatal("UnmarshalCanonicalYAML() error = nil, want unexpected indentation error")
	}
}

func TestMarshalCanonicalYAML_FormatsIntegerValues(t *testing.T) {
	t.Parallel()

	fields := map[string]any{
		"id":       "BUG-001",
		"slug":     "int-values",
		"attempts": int(42),
	}

	got, err := MarshalCanonicalYAML("bug", fields)
	if err != nil {
		t.Fatalf("MarshalCanonicalYAML() error = %v", err)
	}

	want := "" +
		"id: BUG-001\n" +
		"slug: int-values\n" +
		"attempts: 42\n"

	if got != want {
		t.Fatalf("MarshalCanonicalYAML() mismatch\nwant:\n%s\ngot:\n%s", want, got)
	}
}

func TestMarshalCanonicalYAML_FormatsFloatValues(t *testing.T) {
	t.Parallel()

	fields := map[string]any{
		"id":    "BUG-001",
		"slug":  "float-values",
		"ratio": float64(3.14),
	}

	got, err := MarshalCanonicalYAML("bug", fields)
	if err != nil {
		t.Fatalf("MarshalCanonicalYAML() error = %v", err)
	}

	want := "" +
		"id: BUG-001\n" +
		"slug: float-values\n" +
		"ratio: \"3.14\"\n"

	if got != want {
		t.Fatalf("MarshalCanonicalYAML() mismatch\nwant:\n%s\ngot:\n%s", want, got)
	}
}

func TestMarshalCanonicalYAML_FormatsBooleanFalse(t *testing.T) {
	t.Parallel()

	fields := map[string]any{
		"id":      "BUG-001",
		"slug":    "bool-false",
		"enabled": false,
	}

	got, err := MarshalCanonicalYAML("bug", fields)
	if err != nil {
		t.Fatalf("MarshalCanonicalYAML() error = %v", err)
	}

	want := "" +
		"id: BUG-001\n" +
		"slug: bool-false\n" +
		"enabled: false\n"

	if got != want {
		t.Fatalf("MarshalCanonicalYAML() mismatch\nwant:\n%s\ngot:\n%s", want, got)
	}
}

func TestMarshalCanonicalYAML_FormatsUnknownType(t *testing.T) {
	t.Parallel()

	type customStruct struct {
		X int
		Y int
	}

	fields := map[string]any{
		"id":    "BUG-001",
		"slug":  "unknown-type",
		"point": customStruct{X: 1, Y: 2},
	}

	got, err := MarshalCanonicalYAML("bug", fields)
	if err != nil {
		t.Fatalf("MarshalCanonicalYAML() error = %v", err)
	}

	want := "" +
		"id: BUG-001\n" +
		"slug: unknown-type\n" +
		"point: \"{1 2}\"\n"

	if got != want {
		t.Fatalf("MarshalCanonicalYAML() mismatch\nwant:\n%s\ngot:\n%s", want, got)
	}
}

type fixtureStringer string

func (s fixtureStringer) String() string {
	return string(s)
}

func TestMarshalCanonicalYAML_ConsecutiveBackslashRoundTrip(t *testing.T) {
	t.Parallel()

	fields := map[string]any{
		"id":      "BUG-001",
		"slug":    "double-backslash",
		"comment": `path\\with\\double`,
	}

	content, err := MarshalCanonicalYAML("bug", fields)
	if err != nil {
		t.Fatalf("MarshalCanonicalYAML() error = %v", err)
	}

	got, err := UnmarshalCanonicalYAML(content)
	if err != nil {
		t.Fatalf("UnmarshalCanonicalYAML() error = %v", err)
	}

	if got["comment"] != `path\\with\\double` {
		t.Fatalf("consecutive backslash round-trip mismatch: got %q, want %q", got["comment"], `path\\with\\double`)
	}
}

func TestMarshalCanonicalYAML_EmbeddedNewlineRoundTrip(t *testing.T) {
	t.Parallel()

	fields := map[string]any{
		"id":      "BUG-001",
		"slug":    "embedded-newline",
		"comment": "hello\nworld",
	}

	content, err := MarshalCanonicalYAML("bug", fields)
	if err != nil {
		t.Fatalf("MarshalCanonicalYAML() error = %v", err)
	}

	got, err := UnmarshalCanonicalYAML(content)
	if err != nil {
		t.Fatalf("UnmarshalCanonicalYAML() error = %v", err)
	}

	if got["comment"] != "hello\nworld" {
		t.Fatalf("embedded newline round-trip mismatch: got %q, want %q", got["comment"], "hello\nworld")
	}
}

func TestMarshalCanonicalYAML_YAMLBooleanVariantsAreQuoted(t *testing.T) {
	t.Parallel()

	variants := []string{"True", "FALSE", "Yes", "no", "ON", "off", "~"}
	for _, v := range variants {
		fields := map[string]any{
			"id":      "BUG-001",
			"slug":    "bool-variant",
			"comment": v,
		}

		content, err := MarshalCanonicalYAML("bug", fields)
		if err != nil {
			t.Fatalf("MarshalCanonicalYAML(%q) error = %v", v, err)
		}

		got, err := UnmarshalCanonicalYAML(content)
		if err != nil {
			t.Fatalf("UnmarshalCanonicalYAML(%q) error = %v", v, err)
		}

		if got["comment"] != v {
			t.Fatalf("YAML boolean variant round-trip mismatch for %q: got %v (type %T)", v, got["comment"], got["comment"])
		}
	}
}

func TestEntityStore_Load_NonExistentFile(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := NewEntityStore(root)

	_, err := store.Load("epic", "E-999", "does-not-exist")
	if err == nil {
		t.Fatal("Load() error = nil, want non-nil for non-existent file")
	}

	errMsg := err.Error()
	if !strings.Contains(errMsg, "not exist") && !strings.Contains(errMsg, "no such file") {
		t.Fatalf("Load() error = %v, want error indicating file does not exist", err)
	}
}

func TestEntityStore_Load_CorruptYAML(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := NewEntityStore(root)

	dir := filepath.Join(root, "epics")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "E-001-corrupt.yaml"), []byte("{{{{"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	_, err := store.Load("epic", "E-001", "corrupt")
	if err == nil {
		t.Fatal("Load() error = nil, want non-nil for corrupt YAML")
	}

	errMsg := err.Error()
	if !strings.Contains(errMsg, "unmarshal") && !strings.Contains(errMsg, "parse") && !strings.Contains(errMsg, "invalid") {
		t.Fatalf("Load() error = %v, want error about parsing/unmarshalling", err)
	}
}

func TestEntityStore_Load_EmptyFile(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := NewEntityStore(root)

	dir := filepath.Join(root, "epics")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "E-001-empty.yaml"), []byte(""), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	got, err := store.Load("epic", "E-001", "empty")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if len(got.Fields) != 0 {
		t.Fatalf("Load() fields = %v, want empty map for empty file", got.Fields)
	}
}
