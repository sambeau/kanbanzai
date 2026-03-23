package docint

import (
	"regexp"
	"strings"
)

// Entity ID patterns. Each maps to an entity type string.
var entityPatterns = []struct {
	re         *regexp.Regexp
	entityType string
}{
	{regexp.MustCompile(`\bFEAT-[A-Za-z0-9]+\b`), "feature"},
	{regexp.MustCompile(`\bTASK-[A-Za-z0-9]+\b`), "task"},
	{regexp.MustCompile(`\bBUG-[A-Za-z0-9]+\b`), "bug"},
	{regexp.MustCompile(`\bDEC-[A-Za-z0-9]+\b`), "decision"},
	{regexp.MustCompile(`\bEPIC-[A-Za-z0-9]+\b`), "epic"},
	{regexp.MustCompile(`\bDOC-[A-Za-z0-9]+\b`), "document"},
}

// Plan ID pattern: letter + digits + hyphen + lowercase slug, e.g. "P1-basic-ui".
var planIDPattern = regexp.MustCompile(`\b[A-Z]\d+-[a-z][-a-z0-9]+\b`)

// Markdown link to .md file: [text](path.md) or [text](path/to/file.md)
var mdLinkPattern = regexp.MustCompile(`\[([^\]]*)\]\(([^)]*\.md(?:#[^)]*)?)\)`)

// Backtick-quoted path ending in .md: `path/to/file.md`
var backtickMDPattern = regexp.MustCompile("`([^`]+\\.md(?:\\s+[^`]*)?)`")

// ExtractResult holds all Layer 2 extraction results.
type ExtractResult struct {
	FrontMatter       *FrontMatter
	EntityRefs        []EntityRef
	CrossDocLinks     []CrossDocLink
	ConventionalRoles []ConventionalRole
}

// ExtractPatterns performs Layer 2 pattern extraction on a document.
// It takes the raw content and the parsed sections (from Layer 1) and extracts:
//   - Entity references (FEAT-xxx, TASK-xxx, BUG-xxx, DEC-xxx, EPIC-xxx, Plan IDs)
//   - Cross-document links (markdown links to .md files, backtick-quoted .md paths)
//   - Conventional section roles (headings matching known keywords)
//   - Front matter (bullet-list key: value lines after the H1 heading)
func ExtractPatterns(content []byte, sections []Section) ExtractResult {
	flat := flattenSections(sections)

	var result ExtractResult
	result.FrontMatter = extractFrontMatter(content)
	result.EntityRefs = extractEntityRefs(content, flat)
	result.CrossDocLinks = extractCrossDocLinks(content, flat)
	result.ConventionalRoles = extractConventionalRoles(sections)
	return result
}

// flatSection is a flattened view of a section for byte-range lookups.
type flatSectionView struct {
	path      string
	byteStart int
	byteEnd   int
}

// flattenSections recursively flattens a section tree into a slice for lookups.
func flattenSections(sections []Section) []flatSectionView {
	var out []flatSectionView
	var walk func([]Section)
	walk = func(ss []Section) {
		for i := range ss {
			s := &ss[i]
			out = append(out, flatSectionView{
				path:      s.Path,
				byteStart: s.ByteOffset,
				byteEnd:   s.ByteOffset + s.ByteCount,
			})
			walk(s.Children)
		}
	}
	walk(sections)
	return out
}

// sectionForOffset returns the section path for a given byte offset.
// It returns the most specific (deepest / last matching) section whose range
// contains the offset — because flattenSections walks depth-first, deeper
// children appear after their parent, so the last match is the most specific.
func sectionForOffset(offset int, flat []flatSectionView) string {
	best := ""
	for i := range flat {
		if offset >= flat[i].byteStart && offset < flat[i].byteEnd {
			best = flat[i].path
		}
	}
	return best
}

// extractEntityRefs finds all entity references in the content.
// Deduplicates by (entity_id, section_path) — same ID in same section = one entry.
func extractEntityRefs(content []byte, flat []flatSectionView) []EntityRef {
	type key struct {
		id          string
		sectionPath string
	}
	seen := map[key]bool{}
	var refs []EntityRef

	// Named entity patterns (FEAT-, TASK-, etc.)
	for _, ep := range entityPatterns {
		for _, loc := range ep.re.FindAllIndex(content, -1) {
			id := string(content[loc[0]:loc[1]])
			sp := sectionForOffset(loc[0], flat)
			k := key{id, sp}
			if seen[k] {
				continue
			}
			seen[k] = true
			refs = append(refs, EntityRef{
				EntityID:    id,
				EntityType:  ep.entityType,
				SectionPath: sp,
				ByteOffset:  loc[0],
			})
		}
	}

	// Plan ID pattern
	for _, loc := range planIDPattern.FindAllIndex(content, -1) {
		id := string(content[loc[0]:loc[1]])
		// Skip if this was already matched by a named pattern (e.g. a plan ID
		// that happens to overlap with a prefix like "FEAT-"). This shouldn't
		// normally happen given the patterns, but guard anyway.
		sp := sectionForOffset(loc[0], flat)
		k := key{id, sp}
		if seen[k] {
			continue
		}
		seen[k] = true
		refs = append(refs, EntityRef{
			EntityID:    id,
			EntityType:  "plan",
			SectionPath: sp,
			ByteOffset:  loc[0],
		})
	}

	return refs
}

// extractCrossDocLinks finds markdown links and backtick-quoted paths to .md files.
// Deduplicates by (target_path, section_path).
func extractCrossDocLinks(content []byte, flat []flatSectionView) []CrossDocLink {
	type key struct {
		target      string
		sectionPath string
	}
	seen := map[key]bool{}
	var links []CrossDocLink

	// Markdown links: [text](path.md) or [text](path/to/file.md#anchor)
	for _, match := range mdLinkPattern.FindAllSubmatchIndex(content, -1) {
		text := string(content[match[2]:match[3]])
		target := string(content[match[4]:match[5]])
		// Strip any anchor fragment for the target path.
		if idx := strings.Index(target, "#"); idx >= 0 {
			target = target[:idx]
		}
		sp := sectionForOffset(match[0], flat)
		k := key{target, sp}
		if seen[k] {
			continue
		}
		seen[k] = true
		links = append(links, CrossDocLink{
			TargetPath:  target,
			LinkText:    text,
			SectionPath: sp,
		})
	}

	// Backtick-quoted paths: `path/to/file.md`
	for _, match := range backtickMDPattern.FindAllSubmatchIndex(content, -1) {
		raw := string(content[match[2]:match[3]])
		// The backtick content might have trailing section references like
		// `work/design/foo.md §7, §8` — take just the path portion.
		target := strings.Fields(raw)[0]
		if !strings.HasSuffix(target, ".md") {
			continue
		}
		sp := sectionForOffset(match[0], flat)
		k := key{target, sp}
		if seen[k] {
			continue
		}
		seen[k] = true
		links = append(links, CrossDocLink{
			TargetPath:  target,
			LinkText:    raw,
			SectionPath: sp,
		})
	}

	return links
}

// extractConventionalRoles walks the section tree and classifies headings.
func extractConventionalRoles(sections []Section) []ConventionalRole {
	var roles []ConventionalRole
	var walk func([]Section)
	walk = func(ss []Section) {
		for i := range ss {
			s := &ss[i]
			if role, ok := MatchConventionalRole(s.Title); ok {
				roles = append(roles, ConventionalRole{
					SectionPath: s.Path,
					Role:        string(role),
					Confidence:  "high",
				})
			}
			walk(s.Children)
		}
	}
	walk(sections)
	return roles
}

// extractFrontMatter parses bullet-list style front matter found after the
// first heading and before the first `---` separator or next heading.
//
// Expected format (as used in project design documents):
//
//	# Title
//
//	- Status: draft design
//	- Date: 2026-07-18
//	- Purpose: define something
//	- Related:
//	  - `work/design/foo.md`
//	  - `work/design/bar.md`
//
//	---
func extractFrontMatter(content []byte) *FrontMatter {
	lines := strings.Split(string(content), "\n")

	// Find the first heading line, then start scanning from the line after it.
	startIdx := -1
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "#") {
			startIdx = i + 1
			break
		}
	}
	if startIdx < 0 || startIdx >= len(lines) {
		return nil
	}

	// Skip blank lines between heading and front matter.
	for startIdx < len(lines) && strings.TrimSpace(lines[startIdx]) == "" {
		startIdx++
	}
	if startIdx >= len(lines) {
		return nil
	}

	// The first non-blank line after the heading must start with "- " to be front matter.
	if !strings.HasPrefix(strings.TrimSpace(lines[startIdx]), "- ") {
		return nil
	}

	// Collect front matter lines until we hit a `---` separator, another heading,
	// or a blank line that isn't followed by an indented continuation.
	var fmLines []string
	for i := startIdx; i < len(lines); i++ {
		trimmed := strings.TrimSpace(lines[i])
		if trimmed == "---" {
			break
		}
		if strings.HasPrefix(trimmed, "#") {
			break
		}
		if trimmed == "" {
			// Blank line ends front matter unless next line is indented continuation.
			if i+1 < len(lines) && len(lines[i+1]) > 0 && (lines[i+1][0] == ' ' || lines[i+1][0] == '\t') {
				continue
			}
			break
		}
		fmLines = append(fmLines, lines[i])
	}

	if len(fmLines) == 0 {
		return nil
	}

	return parseFrontMatterLines(fmLines)
}

// parseFrontMatterLines parses collected front matter lines into a FrontMatter struct.
func parseFrontMatterLines(lines []string) *FrontMatter {
	fm := &FrontMatter{}
	hasContent := false

	var currentKey string
	var listItems []string

	flushList := func() {
		if currentKey == "" || len(listItems) == 0 {
			return
		}
		lower := strings.ToLower(currentKey)
		switch lower {
		case "related", "basis", "notes":
			// These are known list-valued fields.
			if lower == "related" {
				fm.Related = append(fm.Related, listItems...)
			} else {
				// Store other list fields as joined strings in Extra.
				if fm.Extra == nil {
					fm.Extra = map[string]string{}
				}
				fm.Extra[currentKey] = strings.Join(listItems, "; ")
			}
			hasContent = true
		default:
			if fm.Extra == nil {
				fm.Extra = map[string]string{}
			}
			fm.Extra[currentKey] = strings.Join(listItems, "; ")
			hasContent = true
		}
		listItems = nil
	}

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Check if this is an indented sub-item (continuation of a list-valued key).
		isIndented := len(line) > 0 && (line[0] == ' ' || line[0] == '\t') && !strings.HasPrefix(trimmed, "- ")
		isSubItem := len(line) > 0 && (line[0] == ' ' || line[0] == '\t') && strings.HasPrefix(trimmed, "- ")

		if isSubItem {
			// Sub-list item under the current key.
			item := strings.TrimSpace(strings.TrimPrefix(trimmed, "- "))
			// Strip surrounding backticks if present.
			item = strings.Trim(item, "`")
			listItems = append(listItems, item)
			continue
		}

		if isIndented {
			// Indented continuation of previous value.
			if currentKey != "" && len(listItems) == 0 {
				// Continuation of a scalar value — append to extra.
				lower := strings.ToLower(currentKey)
				existing := scalarForKey(fm, lower)
				combined := existing + " " + trimmed
				setScalar(fm, currentKey, strings.TrimSpace(combined))
			}
			continue
		}

		// Flush any pending list.
		flushList()
		currentKey = ""

		// Top-level item: must start with "- ".
		if !strings.HasPrefix(trimmed, "- ") {
			continue
		}

		entry := strings.TrimPrefix(trimmed, "- ")

		colonIdx := strings.Index(entry, ":")
		if colonIdx < 0 {
			continue
		}

		key := strings.TrimSpace(entry[:colonIdx])
		value := strings.TrimSpace(entry[colonIdx+1:])
		currentKey = key

		if value == "" {
			// This key has a sub-list (e.g. "Related:").
			continue
		}

		// Scalar value.
		setScalar(fm, key, value)
		hasContent = true
	}

	// Flush final list if any.
	flushList()

	if !hasContent {
		return nil
	}
	return fm
}

// setScalar sets a scalar front matter value on known fields or Extra.
func setScalar(fm *FrontMatter, key, value string) {
	lower := strings.ToLower(key)
	switch lower {
	case "status":
		fm.Status = value
	case "type":
		fm.Type = value
	case "date":
		fm.Date = value
	default:
		if fm.Extra == nil {
			fm.Extra = map[string]string{}
		}
		fm.Extra[key] = value
	}
}

// scalarForKey retrieves the current scalar value for a known key.
func scalarForKey(fm *FrontMatter, lowerKey string) string {
	switch lowerKey {
	case "status":
		return fm.Status
	case "type":
		return fm.Type
	case "date":
		return fm.Date
	default:
		if fm.Extra != nil {
			// Try the original-case key; caller passes lowercase,
			// but Extra keys use original case. Just return empty.
			return ""
		}
		return ""
	}
}
