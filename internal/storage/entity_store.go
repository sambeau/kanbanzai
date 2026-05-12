package storage

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/sambeau/kanbanzai/internal/fsutil"
	"github.com/sambeau/kanbanzai/internal/model"
	"gopkg.in/yaml.v3"
)

type EntityRecord struct {
	Type     string
	ID       string
	Slug     string
	Fields   map[string]any
	FileHash string
}

type EntityStore struct{ root string }

func NewEntityStore(root string) *EntityStore { return &EntityStore{root: root} }

func (s *EntityStore) Write(record EntityRecord) (string, error) {
	if err := validateRecord(record); err != nil {
		return "", err
	}
	dir := filepath.Join(s.root, entityDirectory(record.Type))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("create entity directory: %w", err)
	}
	path := filepath.Join(dir, entityFileName(record))
	if record.FileHash != "" {
		if current, err := os.ReadFile(path); err == nil {
			h := sha256.Sum256(current)
			if hex.EncodeToString(h[:]) != record.FileHash {
				return "", fmt.Errorf("write entity %s: %w", record.ID, ErrConflict)
			}
		}
	}
	content, err := MarshalCanonicalYAML(record.Type, record.Fields)
	if err != nil {
		return "", fmt.Errorf("marshal canonical yaml: %w", err)
	}
	if err := fsutil.WriteFileAtomic(path, []byte(content), 0o644); err != nil {
		return "", fmt.Errorf("write entity file: %w", err)
	}
	return path, nil
}

func (s *EntityStore) Load(entityType, id, slug string) (EntityRecord, error) {
	record := EntityRecord{Type: entityType, ID: id, Slug: slug}
	if strings.TrimSpace(entityType) == "" {
		return record, errors.New("entity type is required")
	}
	if strings.TrimSpace(id) == "" {
		return record, errors.New("entity id is required")
	}
	if strings.TrimSpace(slug) == "" {
		return record, errors.New("entity slug is required")
	}
	path := filepath.Join(s.root, entityDirectory(entityType), entityFileName(record))
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return record, fmt.Errorf("load entity %s: %w", id, err)
		}
		return record, fmt.Errorf("read entity file: %w", err)
	}
	fields, err := UnmarshalCanonicalYAML(string(data))
	if err != nil {
		return record, fmt.Errorf("unmarshal canonical yaml: %w", err)
	}
	// Backward compat: bugs with status "verified" are interpreted as "verifying"
	if entityType == "bug" {
		if status, ok := fields["status"]; ok {
			if s, isStr := status.(string); isStr && s == "verified" {
				fields["status"] = "verifying"
			}
		}
	}
	record.Fields = fields
	h := sha256.Sum256(data)
	record.FileHash = hex.EncodeToString(h[:])
	return record, nil
}

// SetField loads an entity, sets a single field, and writes it back.
// entityType is required (e.g. "bug", "feature"). slug is required for Load.
// Returns an error if the entity cannot be loaded or written.
func (s *EntityStore) SetField(entityType, id, slug, field, value string) error {
	record, err := s.Load(entityType, id, slug)
	if err != nil {
		return fmt.Errorf("set field %s on %s: %w", field, id, err)
	}
	record.Fields[field] = value
	_, err = s.Write(record)
	return err
}

func validateRecord(record EntityRecord) error {
	if strings.TrimSpace(record.Type) == "" {
		return errors.New("entity type is required")
	}
	if strings.TrimSpace(record.ID) == "" {
		return errors.New("entity id is required")
	}
	if strings.TrimSpace(record.Slug) == "" {
		return errors.New("entity slug is required")
	}
	if len(record.Fields) == 0 {
		return errors.New("entity fields are required")
	}
	id, ok := record.Fields["id"]
	if !ok {
		return errors.New("entity fields must include id")
	}
	if fmt.Sprint(id) != record.ID {
		return fmt.Errorf("entity id mismatch: record=%q fields=%q", record.ID, fmt.Sprint(id))
	}
	slug, ok := record.Fields["slug"]
	if !ok {
		return errors.New("entity fields must include slug")
	}
	if fmt.Sprint(slug) != record.Slug {
		return fmt.Errorf("entity slug mismatch: record=%q fields=%q", record.Slug, fmt.Sprint(slug))
	}
	return nil
}

func entityDirectory(entityType string) string {
	lower := strings.ToLower(strings.TrimSpace(entityType))
	if lower == string(model.EntityKindStrategicPlan) || lower == "plan" {
		return "plans"
	}
	if lower == string(model.EntityKindBatch) {
		return "batches"
	}
	return lower + "s"
}

func entityFileName(record EntityRecord) string {
	lowerType := strings.ToLower(strings.TrimSpace(record.Type))
	if lowerType == string(model.EntityKindBatch) || lowerType == "plan" || lowerType == string(model.EntityKindStrategicPlan) {
		return record.ID + ".yaml"
	}
	return fmt.Sprintf("%s-%s.yaml", record.ID, record.Slug)
}

func MarshalCanonicalYAML(entityType string, fields map[string]any) (string, error) {
	if len(fields) == 0 {
		return "", errors.New("fields are required")
	}
	var b strings.Builder
	if err := writeOrderedMapping(&b, 0, entityType, fields); err != nil {
		return "", err
	}
	return b.String(), nil
}

func UnmarshalCanonicalYAML(content string) (map[string]any, error) {
	if strings.TrimSpace(content) == "" {
		return map[string]any{}, nil
	}

	var doc yaml.Node
	if err := yaml.Unmarshal([]byte(content), &doc); err != nil {
		return nil, fmt.Errorf("unmarshal yaml: %w", err)
	}

	// doc is a Document node with a single child: the root mapping.
	if len(doc.Content) == 0 {
		return map[string]any{}, nil
	}

	result, err := nodeToValue(doc.Content[0])
	if err != nil {
		return nil, err
	}

	m, ok := result.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("expected top-level mapping, got %T", result)
	}
	return m, nil
}

// nodeToValue recursively converts a yaml.Node into a Go value
// (map[string]any, []any, string, int, float64, bool, or nil).
// It normalises time.Time values to RFC3339 strings.
func nodeToValue(node *yaml.Node) (any, error) {
	switch node.Kind {
	case yaml.MappingNode:
		m := make(map[string]any, len(node.Content)/2)
		for i := 0; i < len(node.Content); i += 2 {
			key := node.Content[i].Value
			val, err := nodeToValue(node.Content[i+1])
			if err != nil {
				return nil, err
			}
			m[key] = val
		}
		return m, nil

	case yaml.SequenceNode:
		result := make([]any, 0, len(node.Content))
		for _, child := range node.Content {
			val, err := nodeToValue(child)
			if err != nil {
				return nil, err
			}
			result = append(result, val)
		}
		return result, nil

	case yaml.ScalarNode:
		return scalarNodeValue(node), nil

	default:
		// Tag or Alias — not expected in canonical YAML
		return nil, fmt.Errorf("unexpected YAML node kind %d", node.Kind)
	}
}

// scalarNodeValue extracts the Go value from a yaml.ScalarNode.
// It returns time.Time values as RFC3339 strings for consistency.
func scalarNodeValue(node *yaml.Node) any {
	// If the tag is !!timestamp or the value looks like a timestamp that
	// yaml.v3 would parse, decode it and return as a string.
	if node.Tag == "!!timestamp" {
		if t, err := time.Parse(time.RFC3339, node.Value); err == nil {
			return t.Format(time.RFC3339)
		}
		return node.Value
	}

	switch node.Tag {
	case "!!int":
		// yaml.v3 always uses int64 for decimal ints
		if v, err := strconv.ParseInt(node.Value, 10, 64); err == nil {
			return int(v)
		}
		return node.Value
	case "!!float":
		if v, err := strconv.ParseFloat(node.Value, 64); err == nil {
			return v
		}
		return node.Value
	case "!!bool":
		return node.Value == "true"
	case "!!null":
		return nil
	case "!!str":
		return node.Value
	default:
		// Unrecognised tag — try inference or return raw
		return node.Value
	}
}

func writeOrderedMapping(b *strings.Builder, indent int, entityType string, fields map[string]any) error {
	for _, key := range orderedKeys(entityType, fields) {
		if err := writeYAMLField(b, indent, key, fields[key]); err != nil {
			return err
		}
	}
	return nil
}

func orderedKeys(entityType string, fields map[string]any) []string {
	schemaOrder := fieldOrderForEntityType(entityType)
	seen := make(map[string]struct{}, len(fields))
	keys := make([]string, 0, len(fields))
	for _, key := range schemaOrder {
		if _, ok := fields[key]; ok {
			keys = append(keys, key)
			seen[key] = struct{}{}
		}
	}
	var extras []string
	for key := range fields {
		if _, ok := seen[key]; ok {
			continue
		}
		extras = append(extras, key)
	}
	slices.Sort(extras)
	return append(keys, extras...)
}

func fieldOrderForEntityType(entityType string) []string {
	lower := strings.ToLower(strings.TrimSpace(entityType))
	switch {
	case lower == string(model.EntityKindBatch) || lower == "plan":
		return []string{"id", "slug", "name", "status", "summary", "design", "tags", "created", "created_by", "updated", "supersedes", "superseded_by"}
	case lower == string(model.EntityKindFeature):
		return []string{"id", "slug", "name", "parent", "status", "review_cycle", "blocked_reason", "estimate", "summary", "design", "spec", "dev_plan", "tags", "plan", "tasks", "decisions", "branch", "created", "created_by", "updated", "supersedes", "superseded_by", "overrides"}
	case lower == string(model.EntityKindTask):
		return []string{"id", "parent_feature", "slug", "name", "summary", "status", "estimate", "assignee", "depends_on", "files_planned", "started", "completed", "claimed_at", "dispatched_to", "dispatched_at", "dispatched_by", "completion_summary", "rework_reason", "verification", "tags"}
	case lower == string(model.EntityKindBug):
		return []string{"id", "slug", "name", "status", "estimate", "severity", "priority", "type", "reported_by", "reported", "observed", "expected", "affects", "origin_feature", "origin_task", "environment", "reproduction", "duplicate_of", "fixed_by", "verified_by", "release_target", "tags"}
	case lower == string(model.EntityKindDecision):
		return []string{"id", "slug", "name", "summary", "rationale", "decided_by", "date", "status", "affects", "supersedes", "superseded_by", "tags"}
	case lower == string(model.EntityKindStrategicPlan):
		return []string{"id", "slug", "name", "status", "summary", "parent", "design", "depends_on", "order", "tags", "created", "created_by", "updated", "supersedes", "superseded_by"}
	case lower == string(model.EntityKindIncident):
		return []string{"id", "slug", "name", "status", "severity", "reported_by", "detected_at", "triaged_at", "mitigated_at", "resolved_at", "affected_features", "linked_bugs", "linked_rca", "summary", "created", "created_by", "updated"}
	}
	return nil
}

func writeYAMLField(b *strings.Builder, indent int, key string, value any) error {
	prefix := strings.Repeat("  ", indent)
	switch typed := value.(type) {
	case map[string]any:
		b.WriteString(prefix)
		b.WriteString(key)
		b.WriteString(":\n")
		keys := make([]string, 0, len(typed))
		for nestedKey := range typed {
			keys = append(keys, nestedKey)
		}
		slices.Sort(keys)
		for _, nestedKey := range keys {
			if err := writeYAMLField(b, indent+1, nestedKey, typed[nestedKey]); err != nil {
				return err
			}
		}
	case []any:
		b.WriteString(prefix)
		b.WriteString(key)
		b.WriteString(":\n")
		return writeYAMLList(b, indent+1, typed)
	case []string:
		b.WriteString(prefix)
		b.WriteString(key)
		b.WriteString(":\n")
		items := make([]any, 0, len(typed))
		for _, item := range typed {
			items = append(items, item)
		}
		return writeYAMLList(b, indent+1, items)
	default:
		b.WriteString(prefix)
		b.WriteString(key)
		b.WriteString(": ")
		b.WriteString(formatScalar(value))
		b.WriteString("\n")
	}
	return nil
}

func writeYAMLList(b *strings.Builder, indent int, values []any) error {
	prefix := strings.Repeat("  ", indent)
	for _, value := range values {
		switch typed := value.(type) {
		case map[string]any:
			b.WriteString(prefix)
			b.WriteString("-\n")
			keys := make([]string, 0, len(typed))
			for key := range typed {
				keys = append(keys, key)
			}
			slices.Sort(keys)
			for _, key := range keys {
				if err := writeYAMLField(b, indent+1, key, typed[key]); err != nil {
					return err
				}
			}
		default:
			b.WriteString(prefix)
			b.WriteString("- ")
			b.WriteString(formatScalar(value))
			b.WriteString("\n")
		}
	}
	return nil
}

func formatScalar(value any) string {
	switch typed := value.(type) {
	case nil:
		return "null"
	case string:
		if needsQuotes(typed) {
			return quoteString(typed)
		}
		return typed
	case fmt.Stringer:
		text := typed.String()
		if needsQuotes(text) {
			return quoteString(text)
		}
		return text
	case bool:
		if typed {
			return "true"
		}
		return "false"
	case int, int8, int16, int32, int64:
		return fmt.Sprint(typed)
	case uint, uint8, uint16, uint32, uint64:
		return fmt.Sprint(typed)
	case float32, float64:
		text := fmt.Sprint(typed)
		if needsQuotes(text) {
			return quoteString(text)
		}
		return text
	default:
		text := fmt.Sprint(value)
		if needsQuotes(text) {
			return quoteString(text)
		}
		return text
	}
}

func needsQuotes(value string) bool {
	if value == "" {
		return true
	}
	lower := strings.ToLower(value)
	switch lower {
	case "true", "false", "null", "yes", "no", "on", "off", "~":
		return true
	}
	if strings.TrimSpace(value) != value {
		return true
	}
	if strings.ContainsAny(value, "\n\r") {
		return true
	}
	if _, err := strconv.Atoi(value); err == nil {
		return true
	}
	if _, err := strconv.ParseFloat(value, 64); err == nil {
		return true
	}
	for _, r := range value {
		switch r {
		case ':', '#', '{', '}', '[', ']', ',', '&', '*', '!', '|', '>', '@', '`', '"', '\'':
			return true
		}
	}
	if strings.HasPrefix(value, "-") || strings.HasPrefix(value, "?") {
		return true
	}
	return false
}

func quoteString(value string) string {
	replacer := strings.NewReplacer(`\`, `\\`, `"`, `\"`, "\n", `\n`)
	return `"` + replacer.Replace(value) + `"`
}
