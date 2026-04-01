package binding

import (
	"bytes"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// LoadBindingFile reads and decodes stage-bindings.yaml from the given path.
// It detects duplicate stage keys via the yaml.Node API before structured
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

	// Check for unknown top-level keys and locate stage_bindings.
	var stageBindingsValue *yaml.Node
	for i := 0; i < len(root.Content); i += 2 {
		keyNode := root.Content[i]
		if keyNode.Value != "stage_bindings" {
			errs = append(errs, fmt.Errorf("unknown top-level key: %q", keyNode.Value))
		} else {
			stageBindingsValue = root.Content[i+1]
		}
	}

	if stageBindingsValue == nil {
		errs = append(errs, fmt.Errorf("missing required 'stage_bindings' key"))
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
