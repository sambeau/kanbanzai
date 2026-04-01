package skill

import (
	"bytes"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// parsedSKILLMD holds the raw parse result before section-level validation.
type parsedSKILLMD struct {
	Frontmatter SkillFrontmatter
	BodyRaw     string // raw Markdown body after the closing ---
	LineCount   int    // total lines in the file
}

// maxLines is the maximum number of lines in a SKILL.md file (FR-014).
const maxLines = 500

// parseSKILLMD reads a SKILL.md file, splits frontmatter from body,
// decodes frontmatter with strict YAML parsing, and enforces the 500-line limit.
// Returns all parse errors accumulated in a single pass.
func parseSKILLMD(data []byte) (*parsedSKILLMD, []error) {
	var errs []error
	result := &parsedSKILLMD{}

	lines := strings.Split(string(data), "\n")
	result.LineCount = len(lines)
	// A trailing newline produces an empty final element; don't count it as a line.
	if result.LineCount > 0 && lines[result.LineCount-1] == "" {
		result.LineCount--
	}

	if result.LineCount > maxLines {
		errs = append(errs, fmt.Errorf("file exceeds %d line limit: %d lines", maxLines, result.LineCount))
	}

	// Find the opening --- delimiter (must be the first non-empty line).
	openIdx := -1
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if trimmed == "---" {
			openIdx = i
		}
		break
	}

	if openIdx == -1 {
		errs = append(errs, fmt.Errorf("missing opening frontmatter delimiter"))
		return result, errs
	}

	// Find the closing --- delimiter (scanning from line after the opening).
	closeIdx := -1
	for i := openIdx + 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			closeIdx = i
			break
		}
	}

	if closeIdx == -1 {
		errs = append(errs, fmt.Errorf("missing closing frontmatter delimiter"))
		return result, errs
	}

	// Extract YAML content between the delimiters.
	yamlLines := lines[openIdx+1 : closeIdx]
	yamlContent := strings.Join(yamlLines, "\n")

	// Decode with strict parsing (unknown fields rejected).
	decoder := yaml.NewDecoder(bytes.NewReader([]byte(yamlContent)))
	decoder.KnownFields(true)
	if err := decoder.Decode(&result.Frontmatter); err != nil {
		errs = append(errs, fmt.Errorf("yaml decode error: %w", err))
	}

	// Everything after the closing --- is the body.
	if closeIdx+1 < len(lines) {
		result.BodyRaw = strings.Join(lines[closeIdx+1:], "\n")
	}

	return result, errs
}
