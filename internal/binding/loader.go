package binding

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"strconv"

	"gopkg.in/yaml.v3"
)

// BindingSupportedSchemaVersion is the schema version this binary understands.
const BindingSupportedSchemaVersion = 2

// ErrUnsupportedSchemaVersion is returned when the binding file's schema_version
// exceeds BindingSupportedSchemaVersion.
var ErrUnsupportedSchemaVersion = errors.New("unsupported schema_version")

// ErrBindingVersionMismatch is returned by DecodeBindingFileLegacy when a
// binding file has a schema_version that an older binary does not understand.
var ErrBindingVersionMismatch = errors.New("binding version mismatch")

// LoadBindingFile reads and decodes stage-bindings.yaml from the given path.
// It inspects schema_version first and dispatches to the appropriate decoder.
// Unsupported versions produce a structured ErrUnsupportedSchemaVersion error.
// Duplicate stage keys are detected via the yaml.Node API before structured
// decoding with KnownFields(true). Returns all parse/structural errors accumulated.
func LoadBindingFile(path string) (*BindingFile, []error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, []error{fmt.Errorf("reading binding file: %w", err)}
	}

	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, []error{fmt.Errorf("parsing YAML: %w", err)}
	}

	if doc.Kind != yaml.DocumentNode || len(doc.Content) == 0 {
		return nil, []error{fmt.Errorf("invalid YAML document")}
	}

	root := doc.Content[0]
	if root.Kind != yaml.MappingNode {
		return nil, []error{fmt.Errorf("root must be a mapping")}
	}

	var errs []error

	// --- Phase 1: inspect schema_version and dispatch ---
	// REQ-002: Extract schema_version before any structural validation.
	// Unsupported versions produce a clear, structured error.
	var schemaVersion int
	var stageBindingsValue *yaml.Node
	for i := 0; i < len(root.Content); i += 2 {
		keyNode := root.Content[i]
		valueNode := root.Content[i+1]
		switch keyNode.Value {
		case "schema_version":
			if valueNode.Kind == yaml.ScalarNode {
				v, parseErr := strconv.Atoi(valueNode.Value)
				if parseErr != nil {
					errs = append(errs, fmt.Errorf("schema_version must be an integer, got %q", valueNode.Value))
				} else {
					schemaVersion = v
				}
			} else {
				errs = append(errs, fmt.Errorf("schema_version must be a scalar integer"))
			}
		case "stage_bindings":
			stageBindingsValue = valueNode
		default:
			errs = append(errs, fmt.Errorf("unknown top-level key: %q", keyNode.Value))
		}
	}

	// REQ-002: version dispatch — if schema_version is present and exceeds the
	// supported version, return a structured error immediately.
	// Do this before checking other accumulated errors — unsupported version is a hard stop.
	if schemaVersion > BindingSupportedSchemaVersion {
		return nil, []error{
			fmt.Errorf("%w: schema_version %d is not supported by this binary (supports up to %d)",
				ErrUnsupportedSchemaVersion, schemaVersion, BindingSupportedSchemaVersion),
		}
	}

	if stageBindingsValue == nil {
		errs = append(errs, fmt.Errorf("missing required 'stage_bindings' key"))
	}

	if len(errs) > 0 {
		return nil, errs
	}

	if stageBindingsValue.Kind != yaml.MappingNode {
		errs = append(errs, fmt.Errorf("stage_bindings must be a mapping"))
		return nil, errs
	}

	// Check for duplicate stage keys within stage_bindings.
	seen := make(map[string]bool)
	for i := 0; i < len(stageBindingsValue.Content); i += 2 {
		key := stageBindingsValue.Content[i].Value
		if seen[key] {
			errs = append(errs, fmt.Errorf("duplicate stage key: %s", key))
		}
		seen[key] = true
	}

	if len(errs) > 0 {
		return nil, errs
	}

	// Structural checks passed — decode with strict field checking.
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)

	var bf BindingFile
	if err := decoder.Decode(&bf); err != nil {
		return nil, []error{fmt.Errorf("decoding binding file: %w", err)}
	}

	return &bf, nil
}

// DecodeBindingFileLegacy performs strict YAML decoding without schema_version
// dispatch, simulating an older binary that predates the version-dispatch loader.
// If the file contains a schema_version field, it returns ErrBindingVersionMismatch
// with a clear message explaining the version gap and remediation steps.
//
// REQ-005: Older binaries (without schema_version support) that encounter a v2
// file must refuse with a clear message rather than silently mis-decoding or
// producing a cryptic YAML decode failure.
func DecodeBindingFileLegacy(data []byte) (*BindingFile, error) {
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)

	var bf BindingFile
	if err := decoder.Decode(&bf); err != nil {
		return nil, fmt.Errorf("decoding binding file: %w", err)
	}

	if bf.SchemaVersion > 0 {
		return nil, fmt.Errorf(
			"%w: this binary only understands stage-bindings.yaml format version 1, "+
				"but the file uses schema_version %d — "+
				"please upgrade the kanbanzai binary to continue, "+
				"or downgrade the binding file to version 1",
			ErrBindingVersionMismatch, bf.SchemaVersion,
		)
	}

	return &bf, nil
}
