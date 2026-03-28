package kbzschema_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"kanbanzai/kbzschema"
)

// TestGeneratedSchemaMatchesCommitted verifies that the schema produced by
// GenerateSchema() matches the committed schema/kanbanzai.schema.json file.
//
// If this test fails, regenerate the schema with:
//
//	go run ./cmd/schemagen -o schema/kanbanzai.schema.json
//
// and commit the updated file.
func TestGeneratedSchemaMatchesCommitted(t *testing.T) {
	t.Parallel()

	// Find schema/kanbanzai.schema.json relative to this test file.
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot determine source file path")
	}
	// filename is .../kbzschema/schema_test.go; repo root is one level up.
	repoRoot := filepath.Dir(filepath.Dir(filename))
	committedPath := filepath.Join(repoRoot, "schema", "kanbanzai.schema.json")

	committedBytes, err := os.ReadFile(committedPath)
	if err != nil {
		t.Fatalf("read committed schema %s: %v\n"+
			"Generate it with: go run ./cmd/schemagen -o schema/kanbanzai.schema.json",
			committedPath, err)
	}

	generatedBytes, err := kbzschema.GenerateSchema()
	if err != nil {
		t.Fatalf("GenerateSchema() error = %v", err)
	}
	// schemagen appends a trailing newline; normalise both sides.
	generatedBytes = append(generatedBytes, '\n')

	// Normalise both sides through a JSON round-trip so that cosmetic
	// differences (trailing newline, key ordering) do not cause false
	// failures.
	normalise := func(t *testing.T, label string, src []byte) []byte {
		t.Helper()
		var v any
		if err := json.Unmarshal(src, &v); err != nil {
			t.Fatalf("parse %s JSON: %v", label, err)
		}
		out, err := json.MarshalIndent(v, "", "  ")
		if err != nil {
			t.Fatalf("re-marshal %s JSON: %v", label, err)
		}
		return append(out, '\n')
	}

	normCommitted := normalise(t, "committed", committedBytes)
	normGenerated := normalise(t, "generated", generatedBytes)

	if !bytes.Equal(normCommitted, normGenerated) {
		t.Errorf(
			"committed schema/kanbanzai.schema.json is out of sync with GenerateSchema().\n"+
				"Run: go run ./cmd/schemagen -o schema/kanbanzai.schema.json\n"+
				"and commit the updated file.\n\n"+
				"committed (%d bytes):\n%s\n\ngenerated (%d bytes):\n%s",
			len(normCommitted), normCommitted,
			len(normGenerated), normGenerated,
		)
	}
}
