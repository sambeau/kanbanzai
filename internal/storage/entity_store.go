package storage

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/sambeau/kanbanzai/internal/fsutil"
	"github.com/sambeau/kanbanzai/internal/model"
)

type EntityRecord struct {
	Type string
	ID   string
	Slug string

	Fields   map[string]any
	FileHash string // SHA-256 hex digest of file contents at load time; used for optimistic locking
}

type EntityStore struct {
	root string
}

func NewEntityStore(root string) *EntityStore {
	return &EntityStore{root: root}
}

func (s *EntityStore) Write(record EntityRecord) (string, error) {
	if err := validateRecord(record); err != nil {
		return "", err
	}

	dir := filepath.Join(s.root, entityDirectory(record.Type))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("create entity directory: %w", err)
	}

	path := filepath.Join(dir, entityFileName(record))

	// Optimistic locking: if FileHash is set, verify the file hasn't changed.
	if record.FileHash != "" {
		current, err := os.ReadFile(path)
		if err == nil {
			h := sha256.Sum256(current)
			if hex.EncodeToString(h[:]) != record.FileHash {
				return "", fmt.Errorf("write entity %s: %w", record.ID, ErrConflict)
			}
		}
		// If the file doesn't exist (os.ErrNotExist), skip the check — new entity.
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
	record := EntityRecord{
		Type: entityType,
		ID:   id,
		Slug: slug,
	}

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

	record.Fields = fields

	h := sha256.Sum256(data)
	record.FileHash = hex.EncodeToString(h[:])

	return record, nil
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
	return strings.ToLower(strings.TrimSpace(entityType)) + "s"
}

func entityFileName(record EntityRecord) string {
	// Plan IDs already contain the slug (e.g., P1-basic-ui), so the
	// filename is just {id}.yaml per spec §15.1. All other entity types
	// use {id}-{slug}.yaml for human-readable filenames.
	if strings.ToLower(strings.TrimSpace(record.Type)) == string(model.EntityKindPlan) {
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
	lines := splitNonEmptyLines(content)
	result, next, err := parseMapping(lines, 0, 0)
	if err != nil {
		return nil, err
	}
	if next != len(lines) {
		return nil, fmt.Errorf("unexpected trailing content at line %d", next+1)
	}
	return result, nil
}

func writeOrderedMapping(
	b *strings.Builder,
	indent int,
	entityType string,
	fields map[string]any,
) error {
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
	sort.Strings(extras)

	return append(keys, extras...)
}

func fieldOrderForEntityType(entityType string) []string {
	switch strings.ToLower(strings.TrimSpace(entityType)) {
	case string(model.EntityKindPlan):
		return []string{
			"id",
			"slug",
			"title",
			"status",
			"summary",
			"design",
			"tags",
			"created",
			"created_by",
			"updated",
			"supersedes",
			"superseded_by",
		}
	case string(model.EntityKindEpic):
		return []string{
			"id",
			"slug",
			"title",
			"status",
			"estimate",
			"summary",
			"created",
			"created_by",
			"features",
		}
	case string(model.EntityKindFeature):
		return []string{
			"id",
			"slug",
			"label",
			"parent",
			"status",
			"estimate",
			"summary",
			"design",
			"spec",
			"dev_plan",
			"tasks",
			"decisions",
			"tags",
			"branch",
			"created",
			"created_by",
			"updated",
			"supersedes",
			"superseded_by",
		}
	case string(model.EntityKindTask):
		return []string{
			"id",
			"parent_feature",
			"slug",
			"label",
			"summary",
			"status",
			"estimate",
			"assignee",
			"depends_on",
			"files_planned",
			"started",
			"completed",
			"claimed_at",
			"dispatched_to",
			"dispatched_at",
			"dispatched_by",
			"completion_summary",
			"rework_reason",
			"verification",
			"tags",
		}
	case string(model.EntityKindBug):
		return []string{
			"id",
			"slug",
			"title",
			"status",
			"estimate",
			"severity",
			"priority",
			"type",
			"reported_by",
			"reported",
			"observed",
			"expected",
			"affects",
			"origin_feature",
			"origin_task",
			"environment",
			"reproduction",
			"duplicate_of",
			"fixed_by",
			"verified_by",
			"release_target",
			"tags",
		}
	case string(model.EntityKindDecision):
		return []string{
			"id",
			"slug",
			"summary",
			"rationale",
			"decided_by",
			"date",
			"status",
			"affects",
			"supersedes",
			"superseded_by",
			"tags",
		}
	case string(model.EntityKindDocument), "document_record":
		return []string{
			"id",
			"path",
			"type",
			"title",
			"status",
			"owner",
			"approved_by",
			"approved_at",
			"content_hash",
			"supersedes",
			"superseded_by",
			"created",
			"created_by",
			"updated",
			"quality_evaluation",
		}
	case string(model.EntityKindKnowledgeEntry):
		return []string{
			"id",
			"tier",
			"topic",
			"scope",
			"content",
			"learned_from",
			"status",
			"use_count",
			"miss_count",
			"confidence",
			"last_used",
			"ttl_days",
			"promoted_from",
			"merged_from",
			"deprecated_reason",
			"git_anchors",
			"tags",
			"created",
			"created_by",
			"updated",
		}
	case string(model.EntityKindIncident):
		return []string{
			"id",
			"slug",
			"title",
			"status",
			"severity",
			"reported_by",
			"detected_at",
			"triaged_at",
			"mitigated_at",
			"resolved_at",
			"affected_features",
			"linked_bugs",
			"linked_rca",
			"summary",
			"created",
			"created_by",
			"updated",
		}
	default:
		return nil
	}
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
		sort.Strings(keys)

		for _, nestedKey := range keys {
			if err := writeYAMLField(b, indent+1, nestedKey, typed[nestedKey]); err != nil {
				return err
			}
		}
		return nil
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
		return nil
	}
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
			sort.Strings(keys)

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

func parseMapping(lines []string, start, indent int) (map[string]any, int, error) {
	result := map[string]any{}
	i := start

	for i < len(lines) {
		line := lines[i]
		currentIndent := countIndent(line)
		if currentIndent < indent {
			break
		}
		if currentIndent > indent {
			return nil, i, fmt.Errorf("unexpected indentation at line %d", i+1)
		}

		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "- ") || trimmed == "-" {
			return nil, i, fmt.Errorf("unexpected list item at line %d", i+1)
		}

		parts := strings.SplitN(trimmed, ":", 2)
		if len(parts) != 2 {
			return nil, i, fmt.Errorf("invalid mapping entry at line %d", i+1)
		}

		key := strings.TrimSpace(parts[0])
		rest := strings.TrimSpace(parts[1])

		if key == "" {
			return nil, i, fmt.Errorf("empty key at line %d", i+1)
		}

		if rest != "" {
			result[key] = parseScalar(rest)
			i++
			continue
		}

		if i+1 >= len(lines) {
			result[key] = map[string]any{}
			i++
			continue
		}

		nextIndent := countIndent(lines[i+1])
		if nextIndent <= indent {
			result[key] = map[string]any{}
			i++
			continue
		}

		nextTrimmed := strings.TrimSpace(lines[i+1])
		if strings.HasPrefix(nextTrimmed, "- ") || nextTrimmed == "-" {
			list, next, err := parseList(lines, i+1, indent+1)
			if err != nil {
				return nil, i, err
			}
			result[key] = list
			i = next
			continue
		}

		nested, next, err := parseMapping(lines, i+1, indent+1)
		if err != nil {
			return nil, i, err
		}
		result[key] = nested
		i = next
	}

	return result, i, nil
}

func parseList(lines []string, start, indent int) ([]any, int, error) {
	var result []any
	i := start

	for i < len(lines) {
		line := lines[i]
		currentIndent := countIndent(line)
		if currentIndent < indent {
			break
		}
		if currentIndent > indent {
			return nil, i, fmt.Errorf("unexpected indentation at line %d", i+1)
		}

		trimmed := strings.TrimSpace(line)
		if trimmed == "-" {
			if i+1 >= len(lines) {
				result = append(result, map[string]any{})
				i++
				continue
			}

			nextIndent := countIndent(lines[i+1])
			if nextIndent <= indent {
				result = append(result, map[string]any{})
				i++
				continue
			}

			nested, next, err := parseMapping(lines, i+1, indent+1)
			if err != nil {
				return nil, i, err
			}
			result = append(result, nested)
			i = next
			continue
		}

		if !strings.HasPrefix(trimmed, "- ") {
			return nil, i, fmt.Errorf("invalid list item at line %d", i+1)
		}

		value := strings.TrimSpace(strings.TrimPrefix(trimmed, "- "))
		result = append(result, parseScalar(value))
		i++
	}

	return result, i, nil
}

func parseScalar(value string) any {
	value = strings.TrimSpace(value)

	switch value {
	case "true":
		return true
	case "false":
		return false
	case "null":
		return nil
	}

	if intValue, err := strconv.Atoi(value); err == nil {
		return intValue
	}

	if floatValue, err := strconv.ParseFloat(value, 64); err == nil && strings.Contains(value, ".") {
		return floatValue
	}

	if len(value) >= 2 && strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"") {
		unquoted := value[1 : len(value)-1]
		unquoted = strings.ReplaceAll(unquoted, "\\\"", "\"")
		unquoted = strings.ReplaceAll(unquoted, "\\\\", "\\")
		unquoted = strings.ReplaceAll(unquoted, "\\n", "\n")
		return unquoted
	}

	return value
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
	replacer := strings.NewReplacer(
		`\`, `\\`,
		`"`, `\"`,
		"\n", `\n`,
	)
	return `"` + replacer.Replace(value) + `"`
}

func splitNonEmptyLines(content string) []string {
	raw := strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n")
	lines := make([]string, 0, len(raw))
	for _, line := range raw {
		if strings.TrimSpace(line) == "" {
			continue
		}
		lines = append(lines, line)
	}
	return lines
}

func countIndent(line string) int {
	count := 0
	for strings.HasPrefix(line, "  ") {
		count++
		line = line[2:]
	}
	return count
}
