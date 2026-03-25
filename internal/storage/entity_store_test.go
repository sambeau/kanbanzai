package storage

import (
	"errors"
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
		"id":      "FEAT-01J3K7MXP3RT5",
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
		"id: FEAT-01J3K7MXP3RT5\n" +
		"slug: kernel-storage\n" +
		"status: draft\n" +
		"summary: Kernel storage\n" +
		"tags:\n" +
		"  - phase-1\n" +
		"  - storage\n" +
		"meta:\n" +
		"  created: \"2026-03-19T00:00:00Z\"\n" +
		"  owner: sam\n"

	if got != want {
		t.Fatalf("MarshalCanonicalYAML() mismatch\nwant:\n%s\ngot:\n%s", want, got)
	}
}

func TestCanonicalYAML_RoundTrip(t *testing.T) {
	t.Parallel()

	input := map[string]any{
		"id":      "BUG-01J4AR7WHN4F2",
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
		"id":      "BUG-01J4AR7WHN4F2",
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
		ID:   "FEAT-01J3K7MXP3RT5",
		Slug: "initial-kernel",
		Fields: map[string]any{
			"id":         "FEAT-01J3K7MXP3RT5",
			"slug":       "initial-kernel",
			"epic":       "EPIC-TESTEPIC",
			"status":     "draft",
			"summary":    "Start the workflow kernel",
			"created":    "2026-03-19T00:00:00Z",
			"created_by": "sam",
			"tasks":      []string{"TASK-01J3KZZZBB4KF", "TASK-01J3L0AACC5LG"},
		},
	}

	path, err := store.Write(record)
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	wantPath := filepath.Join(root, "features", "FEAT-01J3K7MXP3RT5-initial-kernel.yaml")
	if path != wantPath {
		t.Fatalf("Write() path mismatch: want %q, got %q", wantPath, path)
	}

	got, err := store.Load("feature", "FEAT-01J3K7MXP3RT5", "initial-kernel")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	want := map[string]any{
		"created":    "2026-03-19T00:00:00Z",
		"created_by": "sam",
		"epic":       "EPIC-TESTEPIC",
		"id":         "FEAT-01J3K7MXP3RT5",
		"slug":       "initial-kernel",
		"status":     "draft",
		"summary":    "Start the workflow kernel",
		"tasks":      []any{"TASK-01J3KZZZBB4KF", "TASK-01J3L0AACC5LG"},
	}

	if got.Type != "feature" {
		t.Fatalf("Load() type mismatch: want %q, got %q", "feature", got.Type)
	}
	if got.ID != "FEAT-01J3K7MXP3RT5" {
		t.Fatalf("Load() id mismatch: want %q, got %q", "FEAT-01J3K7MXP3RT5", got.ID)
	}
	if got.Slug != "initial-kernel" {
		t.Fatalf("Load() slug mismatch: want %q, got %q", "initial-kernel", got.Slug)
	}
	if !reflect.DeepEqual(got.Fields, want) {
		t.Fatalf("Load() fields mismatch\nwant: %#v\ngot:  %#v", want, got.Fields)
	}
}

func TestEntityStore_WriteAndLoad_Plan(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := NewEntityStore(root)

	record := EntityRecord{
		Type: "plan",
		ID:   "P1-basic-ui",
		Slug: "basic-ui",
		Fields: map[string]any{
			"id":         "P1-basic-ui",
			"slug":       "basic-ui",
			"title":      "Basic UI",
			"status":     "proposed",
			"summary":    "Build the basic UI",
			"created":    "2026-03-22T12:00:00Z",
			"created_by": "sam",
			"updated":    "2026-03-22T12:00:00Z",
		},
	}

	path, err := store.Write(record)
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	// Plan files must use {id}.yaml, not {id}-{slug}.yaml, because the
	// Plan ID already contains the slug (spec §15.1).
	wantPath := filepath.Join(root, "plans", "P1-basic-ui.yaml")
	if path != wantPath {
		t.Fatalf("Write() path = %q, want %q", path, wantPath)
	}

	// Verify the file actually exists at that path
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("Plan file does not exist at %q: %v", path, err)
	}

	got, err := store.Load("plan", "P1-basic-ui", "basic-ui")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if got.Type != "plan" {
		t.Errorf("Load() type = %q, want %q", got.Type, "plan")
	}
	if got.ID != "P1-basic-ui" {
		t.Errorf("Load() id = %q, want %q", got.ID, "P1-basic-ui")
	}
	if got.Slug != "basic-ui" {
		t.Errorf("Load() slug = %q, want %q", got.Slug, "basic-ui")
	}

	want := map[string]any{
		"id":         "P1-basic-ui",
		"slug":       "basic-ui",
		"title":      "Basic UI",
		"status":     "proposed",
		"summary":    "Build the basic UI",
		"created":    "2026-03-22T12:00:00Z",
		"created_by": "sam",
		"updated":    "2026-03-22T12:00:00Z",
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
		ID:   "EPIC-TESTEPIC",
		Slug: "phase-1",
		Fields: map[string]any{
			"id":      "EPIC-MISMATCH",
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
		"id: FEAT-01J3K7MXP3RT5\n" +
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
		{name: "plan", path: filepath.Join("..", "..", "testdata", "entities", "plan.yaml"), entityType: "plan"},
		{name: "epic", path: filepath.Join("..", "..", "testdata", "entities", "epic.yaml"), entityType: "epic"},
		{name: "feature", path: filepath.Join("..", "..", "testdata", "entities", "feature.yaml"), entityType: "feature"},
		{name: "task", path: filepath.Join("..", "..", "testdata", "entities", "task.yaml"), entityType: "task"},
		{name: "bug", path: filepath.Join("..", "..", "testdata", "entities", "bug.yaml"), entityType: "bug"},
		{name: "decision", path: filepath.Join("..", "..", "testdata", "entities", "decision.yaml"), entityType: "decision"},
		{name: "incident", path: filepath.Join("..", "..", "testdata", "entities", "incident.yaml"), entityType: "incident"},
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
			name:       "plan",
			sourcePath: filepath.Join("..", "..", "testdata", "entities", "plan.yaml"),
			entityType: "plan",
			id:         "P1-initial-kernel",
			slug:       "initial-kernel",
			wantFields: map[string]any{
				"id":         "P1-initial-kernel",
				"slug":       "initial-kernel",
				"title":      "Phase 1 Kernel",
				"status":     "active",
				"summary":    "Build the initial workflow kernel",
				"design":     "work/design/workflow-design-basis.md",
				"tags":       []any{"phase:1", "core"},
				"created":    "2026-03-19T12:00:00Z",
				"created_by": "sam",
				"updated":    "2026-03-20T10:00:00Z",
			},
			wantFilePath: filepath.Join(root, "plans", "P1-initial-kernel.yaml"),
		},
		{
			name:       "epic",
			sourcePath: filepath.Join("..", "..", "testdata", "entities", "epic.yaml"),
			entityType: "epic",
			id:         "EPIC-PHASE1KERNEL",
			slug:       "phase-1-kernel",
			wantFields: map[string]any{
				"id":         "EPIC-PHASE1KERNEL",
				"slug":       "phase-1-kernel",
				"title":      "Phase 1 Kernel",
				"status":     "proposed",
				"summary":    "Build the initial workflow kernel",
				"created":    "2026-03-19T12:00:00Z",
				"created_by": "sam",
				"features":   []any{"FEAT-01J3K7MXP3RT5", "FEAT-01J3K8NYQ4SU6"},
			},
			wantFilePath: filepath.Join(root, "epics", "EPIC-PHASE1KERNEL-phase-1-kernel.yaml"),
		},
		{
			name:       "feature",
			sourcePath: filepath.Join("..", "..", "testdata", "entities", "feature.yaml"),
			entityType: "feature",
			id:         "FEAT-01J3K7MXP3RT5",
			slug:       "initial-kernel",
			wantFields: map[string]any{
				"id":         "FEAT-01J3K7MXP3RT5",
				"slug":       "initial-kernel",
				"parent":     "EPIC-PHASE1KERNEL",
				"status":     "draft",
				"summary":    "Start the workflow kernel",
				"created":    "2026-03-19T12:00:00Z",
				"created_by": "sam",
				"spec":       "work/spec/phase-1-specification.md",
				"dev_plan":   "work/plan/phase-1-implementation-plan.md",
				"tasks":      []any{"TASK-01J3KZZZBB4KF", "TASK-01J3L0AACC5LG"},
				"decisions":  []any{"DEC-01J3KABCDE7MX"},
				"branch":     "feat/feat-01j3k7mxp3rt5-initial-kernel",
			},
			wantFilePath: filepath.Join(root, "features", "FEAT-01J3K7MXP3RT5-initial-kernel.yaml"),
		},
		{
			name:       "task",
			sourcePath: filepath.Join("..", "..", "testdata", "entities", "task.yaml"),
			entityType: "task",
			id:         "TASK-01J3KZZZBB4KF",
			slug:       "write-entity-files",
			wantFields: map[string]any{
				"id":             "TASK-01J3KZZZBB4KF",
				"parent_feature": "FEAT-01J3K7MXP3RT5",
				"slug":           "write-entity-files",
				"summary":        "Write canonical entity files to disk",
				"status":         "queued",
				"assignee":       "sam",
				"depends_on":     []any{"TASK-01J3KYYY9A3JE"},
				"files_planned":  []any{"internal/storage/entity_store.go", "internal/storage/entity_store_test.go"},
				"verification":   "go test ./...",
			},
			wantFilePath: filepath.Join(root, "tasks", "TASK-01J3KZZZBB4KF-write-entity-files.yaml"),
		},
		{
			name:       "bug",
			sourcePath: filepath.Join("..", "..", "testdata", "entities", "bug.yaml"),
			entityType: "bug",
			id:         "BUG-01J4AR7WHN4F2",
			slug:       "bad-yaml-output",
			wantFields: map[string]any{
				"id":          "BUG-01J4AR7WHN4F2",
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
			wantFilePath: filepath.Join(root, "bugs", "BUG-01J4AR7WHN4F2-bad-yaml-output.yaml"),
		},
		{
			name:       "decision",
			sourcePath: filepath.Join("..", "..", "testdata", "entities", "decision.yaml"),
			entityType: "decision",
			id:         "DEC-01J3KABCDE7MX",
			slug:       "strict-yaml-subset",
			wantFields: map[string]any{
				"id":         "DEC-01J3KABCDE7MX",
				"slug":       "strict-yaml-subset",
				"summary":    "Use a strict canonical YAML subset",
				"rationale":  "Deterministic output is required for Git-friendly state",
				"decided_by": "sam",
				"date":       "2026-03-19T12:00:00Z",
				"status":     "proposed",
			},
			wantFilePath: filepath.Join(root, "decisions", "DEC-01J3KABCDE7MX-strict-yaml-subset.yaml"),
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
		"id":   "BUG-01J4AR7WHN4F2",
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
		"id: BUG-01J4AR7WHN4F2\n" +
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
		"id":      "BUG-01J4AR7WHN4F2",
		"slug":    "nil-value",
		"comment": nil,
	}

	got, err := MarshalCanonicalYAML("bug", fields)
	if err != nil {
		t.Fatalf("MarshalCanonicalYAML() error = %v", err)
	}

	want := "" +
		"id: BUG-01J4AR7WHN4F2\n" +
		"slug: nil-value\n" +
		"comment: null\n"

	if got != want {
		t.Fatalf("MarshalCanonicalYAML() mismatch\nwant:\n%s\ngot:\n%s", want, got)
	}
}

func TestUnmarshalCanonicalYAML_ParsesNumericScalars(t *testing.T) {
	t.Parallel()

	content := "" +
		"id: BUG-01J4AR7WHN4F2\n" +
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
		"id":       "BUG-01J4AR7WHN4F2",
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
		"id":      "BUG-01J4AR7WHN4F2",
		"slug":    "stringer-value",
		"comment": fixtureStringer("needs:quotes"),
	}

	got, err := MarshalCanonicalYAML("bug", fields)
	if err != nil {
		t.Fatalf("MarshalCanonicalYAML() error = %v", err)
	}

	want := "" +
		"id: BUG-01J4AR7WHN4F2\n" +
		"slug: stringer-value\n" +
		"comment: \"needs:quotes\"\n"

	if got != want {
		t.Fatalf("MarshalCanonicalYAML() mismatch\nwant:\n%s\ngot:\n%s", want, got)
	}
}

func TestMarshalCanonicalYAML_IdempotentWrite(t *testing.T) {
	t.Parallel()

	fields := map[string]any{
		"id":       "BUG-01J4AR7WHN4F2",
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
		"id":      "BUG-01J4AR7WHN4F2",
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
		"id: BUG-01J4AR7WHN4F2\n" +
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
		"id":   "BUG-01J4AR7WHN4F2",
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
		"id: BUG-01J4AR7WHN4F2\n" +
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
		"id: BUG-01J4AR7WHN4F2\n" +
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
		"id":       "BUG-01J4AR7WHN4F2",
		"slug":     "int-values",
		"attempts": int(42),
	}

	got, err := MarshalCanonicalYAML("bug", fields)
	if err != nil {
		t.Fatalf("MarshalCanonicalYAML() error = %v", err)
	}

	want := "" +
		"id: BUG-01J4AR7WHN4F2\n" +
		"slug: int-values\n" +
		"attempts: 42\n"

	if got != want {
		t.Fatalf("MarshalCanonicalYAML() mismatch\nwant:\n%s\ngot:\n%s", want, got)
	}
}

func TestMarshalCanonicalYAML_FormatsFloatValues(t *testing.T) {
	t.Parallel()

	fields := map[string]any{
		"id":    "BUG-01J4AR7WHN4F2",
		"slug":  "float-values",
		"ratio": float64(3.14),
	}

	got, err := MarshalCanonicalYAML("bug", fields)
	if err != nil {
		t.Fatalf("MarshalCanonicalYAML() error = %v", err)
	}

	want := "" +
		"id: BUG-01J4AR7WHN4F2\n" +
		"slug: float-values\n" +
		"ratio: \"3.14\"\n"

	if got != want {
		t.Fatalf("MarshalCanonicalYAML() mismatch\nwant:\n%s\ngot:\n%s", want, got)
	}
}

func TestMarshalCanonicalYAML_FormatsBooleanFalse(t *testing.T) {
	t.Parallel()

	fields := map[string]any{
		"id":      "BUG-01J4AR7WHN4F2",
		"slug":    "bool-false",
		"enabled": false,
	}

	got, err := MarshalCanonicalYAML("bug", fields)
	if err != nil {
		t.Fatalf("MarshalCanonicalYAML() error = %v", err)
	}

	want := "" +
		"id: BUG-01J4AR7WHN4F2\n" +
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
		"id":    "BUG-01J4AR7WHN4F2",
		"slug":  "unknown-type",
		"point": customStruct{X: 1, Y: 2},
	}

	got, err := MarshalCanonicalYAML("bug", fields)
	if err != nil {
		t.Fatalf("MarshalCanonicalYAML() error = %v", err)
	}

	want := "" +
		"id: BUG-01J4AR7WHN4F2\n" +
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
		"id":      "BUG-01J4AR7WHN4F2",
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
		"id":      "BUG-01J4AR7WHN4F2",
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
			"id":      "BUG-01J4AR7WHN4F2",
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

	_, err := store.Load("epic", "EPIC-NONEXIST", "does-not-exist")
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
	if err := os.WriteFile(filepath.Join(dir, "EPIC-TESTEPIC-corrupt.yaml"), []byte("{{{{"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	_, err := store.Load("epic", "EPIC-TESTEPIC", "corrupt")
	if err == nil {
		t.Fatal("Load() error = nil, want non-nil for corrupt YAML")
	}

	errMsg := err.Error()
	if !strings.Contains(errMsg, "unmarshal") && !strings.Contains(errMsg, "parse") && !strings.Contains(errMsg, "invalid") {
		t.Fatalf("Load() error = %v, want error about parsing/unmarshalling", err)
	}
}

func TestEntityStore_Load_LegacySequentialID(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	store := NewEntityStore(dir)

	// Write a feature file using the legacy sequential ID format (FEAT-001).
	// The system must read it without error per ID spec §13.6 / §14.13.
	legacyFields := map[string]any{
		"id":         "FEAT-001",
		"slug":       "legacy-feature",
		"epic":       "EPIC-TESTEPIC",
		"status":     "draft",
		"summary":    "A feature with a legacy sequential ID",
		"created":    "2025-01-01T00:00:00Z",
		"created_by": "test",
	}

	record := EntityRecord{
		Type:   "feature",
		ID:     "FEAT-001",
		Slug:   "legacy-feature",
		Fields: legacyFields,
	}

	path, err := store.Write(record)
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	loaded, err := store.Load("feature", "FEAT-001", "legacy-feature")
	if err != nil {
		t.Fatalf("Load() error = %v (legacy sequential IDs must be readable)", err)
	}

	if loaded.ID != "FEAT-001" {
		t.Fatalf("Load() ID = %q, want %q", loaded.ID, "FEAT-001")
	}

	if loaded.Fields["id"] != "FEAT-001" {
		t.Fatalf("Load() fields[id] = %q, want %q", loaded.Fields["id"], "FEAT-001")
	}

	_ = path
}

func TestEntityStore_Load_EmptyFile(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := NewEntityStore(root)

	dir := filepath.Join(root, "epics")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "EPIC-TESTEPIC-empty.yaml"), []byte(""), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	got, err := store.Load("epic", "EPIC-TESTEPIC", "empty")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if len(got.Fields) != 0 {
		t.Fatalf("Load() fields = %v, want empty map for empty file", got.Fields)
	}
}

func TestEntityStore_Load_PopulatesFileHash(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := NewEntityStore(root)

	record := EntityRecord{
		Type: "feature",
		ID:   "FEAT-01HASH",
		Slug: "hash-test",
		Fields: map[string]any{
			"id":         "FEAT-01HASH",
			"slug":       "hash-test",
			"status":     "draft",
			"summary":    "Test file hash population",
			"created":    "2026-03-19T00:00:00Z",
			"created_by": "sam",
		},
	}

	if _, err := store.Write(record); err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	got, err := store.Load("feature", "FEAT-01HASH", "hash-test")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if got.FileHash == "" {
		t.Fatal("Load() FileHash is empty, expected SHA-256 hex digest")
	}
	if len(got.FileHash) != 64 {
		t.Fatalf("Load() FileHash length = %d, want 64 hex chars", len(got.FileHash))
	}
}

func TestEntityStore_Write_SucceedsWithCorrectFileHash(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := NewEntityStore(root)

	record := EntityRecord{
		Type: "feature",
		ID:   "FEAT-01LOCK",
		Slug: "lock-ok",
		Fields: map[string]any{
			"id":         "FEAT-01LOCK",
			"slug":       "lock-ok",
			"status":     "draft",
			"summary":    "Optimistic lock success",
			"created":    "2026-03-19T00:00:00Z",
			"created_by": "sam",
		},
	}

	if _, err := store.Write(record); err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	loaded, err := store.Load("feature", "FEAT-01LOCK", "lock-ok")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Modify a field and write back with the correct FileHash.
	loaded.Fields["summary"] = "Updated summary"
	if _, err := store.Write(loaded); err != nil {
		t.Fatalf("Write() with correct FileHash error = %v", err)
	}
}

func TestEntityStore_Write_ReturnsErrConflictOnStaleFileHash(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := NewEntityStore(root)

	record := EntityRecord{
		Type: "feature",
		ID:   "FEAT-01STALE",
		Slug: "lock-stale",
		Fields: map[string]any{
			"id":         "FEAT-01STALE",
			"slug":       "lock-stale",
			"status":     "draft",
			"summary":    "Will become stale",
			"created":    "2026-03-19T00:00:00Z",
			"created_by": "sam",
		},
	}

	if _, err := store.Write(record); err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	loaded, err := store.Load("feature", "FEAT-01STALE", "lock-stale")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Simulate a concurrent modification by writing directly.
	stale := loaded
	loaded.Fields["summary"] = "Concurrent update"
	loaded.FileHash = "" // bypass locking for the concurrent write
	if _, err := store.Write(loaded); err != nil {
		t.Fatalf("concurrent Write() error = %v", err)
	}

	// Now try to write with the stale hash.
	stale.Fields["summary"] = "Late update"
	_, err = store.Write(stale)
	if err == nil {
		t.Fatal("Write() with stale FileHash should return error")
	}
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("Write() error = %v, want ErrConflict", err)
	}
}

func TestEntityStore_Write_SucceedsWithEmptyFileHash(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := NewEntityStore(root)

	record := EntityRecord{
		Type: "feature",
		ID:   "FEAT-01EMPTY",
		Slug: "no-hash",
		Fields: map[string]any{
			"id":         "FEAT-01EMPTY",
			"slug":       "no-hash",
			"status":     "draft",
			"summary":    "No hash check",
			"created":    "2026-03-19T00:00:00Z",
			"created_by": "sam",
		},
	}

	// First write — no FileHash, file doesn't exist yet.
	if _, err := store.Write(record); err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	// Second write — no FileHash, file already exists. Should succeed without checking.
	record.Fields["summary"] = "Overwritten without hash"
	if _, err := store.Write(record); err != nil {
		t.Fatalf("Write() with empty FileHash on existing file error = %v", err)
	}
}

func TestEntityStore_Write_NewEntityWithFileHashSucceeds(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := NewEntityStore(root)

	record := EntityRecord{
		Type:     "feature",
		ID:       "FEAT-01NEW",
		Slug:     "brand-new",
		FileHash: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		Fields: map[string]any{
			"id":         "FEAT-01NEW",
			"slug":       "brand-new",
			"status":     "draft",
			"summary":    "New entity with a hash set",
			"created":    "2026-03-19T00:00:00Z",
			"created_by": "sam",
		},
	}

	// File doesn't exist, so the hash check should be skipped even though
	// FileHash is set.
	if _, err := store.Write(record); err != nil {
		t.Fatalf("Write() new entity with FileHash error = %v", err)
	}
}

func TestFeaturePhase2Fields_RoundTrip(t *testing.T) {
	t.Parallel()

	fields := map[string]any{
		"id":         "FEAT-A1B2C3D4E5F6",
		"slug":       "phase2-feature",
		"parent":     "P1",
		"status":     "proposed",
		"summary":    "Feature with Phase 2 fields",
		"design":     "P1/design-doc",
		"spec":       "P1/spec-doc",
		"dev_plan":   "P1/dev-plan-doc",
		"tags":       []string{"api", "backend"},
		"created":    "2025-01-15T10:00:00Z",
		"created_by": "alice",
		"updated":    "2025-01-16T12:30:00Z",
	}

	// Write
	yaml, err := MarshalCanonicalYAML("feature", fields)
	if err != nil {
		t.Fatalf("MarshalCanonicalYAML() error = %v", err)
	}

	// Read back
	got, err := UnmarshalCanonicalYAML(yaml)
	if err != nil {
		t.Fatalf("UnmarshalCanonicalYAML() error = %v", err)
	}

	// Convert expected for comparison (slices become []any in unmarshal)
	want := map[string]any{
		"id":         "FEAT-A1B2C3D4E5F6",
		"slug":       "phase2-feature",
		"parent":     "P1",
		"status":     "proposed",
		"summary":    "Feature with Phase 2 fields",
		"design":     "P1/design-doc",
		"spec":       "P1/spec-doc",
		"dev_plan":   "P1/dev-plan-doc",
		"tags":       []any{"api", "backend"},
		"created":    "2025-01-15T10:00:00Z",
		"created_by": "alice",
		"updated":    "2025-01-16T12:30:00Z",
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Phase 2 Feature round-trip mismatch\nwant: %#v\ngot:  %#v", want, got)
	}

	// Verify field order matches spec
	expectedOrder := []string{
		"id", "slug", "parent", "status", "summary",
		"design", "spec", "dev_plan", "tags",
		"created", "created_by", "updated",
	}

	lines := strings.Split(strings.TrimSpace(yaml), "\n")
	var actualOrder []string
	for _, line := range lines {
		if strings.Contains(line, ":") {
			key := strings.TrimSpace(strings.Split(line, ":")[0])
			actualOrder = append(actualOrder, key)
		}
	}

	if !reflect.DeepEqual(actualOrder, expectedOrder) {
		t.Fatalf("Phase 2 Feature field order mismatch\nwant: %v\ngot:  %v", expectedOrder, actualOrder)
	}
}
