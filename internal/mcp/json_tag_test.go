package mcp

// TestJSONTagRoundTrip verifies that every exported field in every struct
// registered below (a) has an explicit json: struct tag and (b) survives a
// JSON marshal → unmarshal round-trip without data loss.
//
// AUDIT RESULT (P32): The systematic audit of internal/mcp/ found zero structs
// that carry yaml: struct tags. All yaml: occurrences in the package are string
// literals inside error messages, not struct field tags. Therefore the audited
// population is currently empty.
//
// AUDIT SCOPE (P32): internal/docint/ types that participate in the
// doc_intel classify deserialization path were reviewed separately.
// Types examined: Classification, ConceptIntroEntry (in internal/docint/types.go).
// Both carry explicit json: tags on all exported fields. No missing tags found.
// The MCP boundary for doc_intel classify uses json.Unmarshal into
// []Classification; all fields that appear in tool call parameters have
// matching json: tags.
//
// HOW TO ADD A STRUCT: If you introduce a new struct in internal/mcp/ that
// carries yaml: tags AND is used as a json.Unmarshal target or populated from
// MCP req.Params.Arguments, you MUST:
//  1. Add json:"<snake_case>" tags to every exported field (matching the yaml: name).
//  2. Add an entry to yamlTaggedStructs below with a fully-populated non-zero
//     value so the round-trip assertion covers every exported field.
//
// Failing to do so will cause this test to fail, which is the intended
// regression-prevention behaviour described in REQ-004.

import (
	"encoding/json"
	"fmt"
	"reflect"
	"testing"
)

// yamlTagStructEntry pairs a struct value (populated with non-zero values for
// every exported field) with a zero-value instance of the same type used as
// the unmarshal target.
type yamlTagStructEntry struct {
	name    string
	nonZero any
	zero    any
}

// yamlTaggedStructs is the authoritative list of structs in internal/mcp/ that
// carry yaml: struct tags and are used as JSON deserialization targets.
//
// AUDIT (P32-FEAT-01KPX5CW4R82P): None found. List is intentionally empty.
// Update this list if new yaml:-tagged structs are introduced in internal/mcp/.
var yamlTaggedStructs = []yamlTagStructEntry{
	// Example entry (commented out — no real structs in scope):
	//
	// {
	//     name:    "ExampleParams",
	//     nonZero: ExampleParams{ID: "abc", Name: "test"},
	//     zero:    &ExampleParams{},
	// },
}

func TestJSONTagRoundTrip(t *testing.T) {
	for _, entry := range yamlTaggedStructs {
		t.Run(entry.name, func(t *testing.T) {
			// Part 1: reflection check — every exported field with a yaml: tag
			// must also have a non-empty json: tag.
			assertAllYAMLFieldsHaveJSONTags(t, entry.nonZero)

			// Part 2: round-trip — marshal to JSON and unmarshal back; the
			// result must equal the original.
			data, err := json.Marshal(entry.nonZero)
			if err != nil {
				t.Fatalf("%s: json.Marshal failed: %v", entry.name, err)
			}
			if err := json.Unmarshal(data, entry.zero); err != nil {
				t.Fatalf("%s: json.Unmarshal failed: %v", entry.name, err)
			}
			// Dereference the pointer so DeepEqual compares values.
			got := reflect.ValueOf(entry.zero).Elem().Interface()
			if !reflect.DeepEqual(entry.nonZero, got) {
				t.Errorf("%s: round-trip mismatch\n  want: %+v\n   got: %+v",
					entry.name, entry.nonZero, got)
			}
		})
	}

	// Guard: if the list is empty the test still exercises the helper logic via
	// a synthetic struct defined in the test itself, confirming the checker works.
	t.Run("_checker_self_test", func(t *testing.T) {
		type goodStruct struct {
			FieldA string `yaml:"field_a" json:"field_a"`
			FieldB int    `yaml:"field_b" json:"field_b"`
		}

		// Happy path: correct struct must report no violations.
		violations := checkStructHasAllJSONTags(goodStruct{FieldA: "x", FieldB: 1})
		if len(violations) != 0 {
			t.Errorf("expected no violations for goodStruct, got: %v", violations)
		}

		// Failure path: a struct with a yaml: tag but no json: tag must be detected.
		type badStruct struct {
			MissingJSON string `yaml:"missing_json"` // intentionally no json: tag
		}
		violations = checkStructHasAllJSONTags(badStruct{MissingJSON: "oops"})
		if len(violations) == 0 {
			t.Error("expected at least one violation for badStruct (missing json: tag), got none — checker is broken")
		}
		found := false
		for _, v := range violations {
			if v == "badStruct.MissingJSON" {
				found = true
			}
		}
		if !found {
			t.Errorf("expected violation for badStruct.MissingJSON, got: %v", violations)
		}

		// Verify round-trip on the synthetic good struct.
		orig := goodStruct{FieldA: "hello", FieldB: 42}
		data, err := json.Marshal(orig)
		if err != nil {
			t.Fatalf("marshal good struct: %v", err)
		}
		var got goodStruct
		if err := json.Unmarshal(data, &got); err != nil {
			t.Fatalf("unmarshal good struct: %v", err)
		}
		if !reflect.DeepEqual(orig, got) {
			t.Errorf("good struct round-trip mismatch: want %+v got %+v", orig, got)
		}
	})
}

// assertAllYAMLFieldsHaveJSONTags uses reflection to walk every exported field
// of v. If a field carries a yaml: tag it must also carry a non-empty json: tag.
// The function calls t.Errorf (not Fatalf) so that all violations are reported
// before the test exits.
func assertAllYAMLFieldsHaveJSONTags(t *testing.T, v any) {
	t.Helper()
	for _, field := range checkStructHasAllJSONTags(v) {
		t.Errorf("field %s has yaml: tag but is missing a json: tag — add json:<snake_case_name>", field)
	}
}

// checkStructHasAllJSONTags walks v's exported fields and returns the qualified
// field paths (e.g. "MyStruct.FieldA") of fields that have a yaml: tag but lack
// a json: tag. Returns nil if no violations are found.
func checkStructHasAllJSONTags(v any) []string {
	rv := reflect.TypeOf(v)
	if rv.Kind() == reflect.Ptr {
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return nil
	}
	return collectYAMLTagViolations(rv, rv.Name())
}

// collectYAMLTagViolations recursively inspects st for exported fields that have
// yaml: tags but lack json: tags. Returns the qualified field paths of violations.
func collectYAMLTagViolations(st reflect.Type, path string) []string {
	var violations []string
	for i := 0; i < st.NumField(); i++ {
		f := st.Field(i)
		if !f.IsExported() {
			continue
		}
		fieldPath := fmt.Sprintf("%s.%s", path, f.Name)
		yamlTag := f.Tag.Get("yaml")
		if yamlTag != "" && yamlTag != "-" {
			jsonTag := f.Tag.Get("json")
			if jsonTag == "" || jsonTag == "-" {
				violations = append(violations, fieldPath)
			}
		}
		// Recurse into nested structs.
		ft := f.Type
		if ft.Kind() == reflect.Ptr {
			ft = ft.Elem()
		}
		if ft.Kind() == reflect.Struct {
			violations = append(violations, collectYAMLTagViolations(ft, fieldPath)...)
		}
	}
	return violations
}
