package docint

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

// ParseStructure parses a Markdown document into a section tree.
// This is Layer 1: structural skeleton.
// It identifies headed sections with their level, title, byte offset,
// word count, byte count, and content hash.
func ParseStructure(content []byte) []Section {
	if len(content) == 0 {
		return nil
	}

	headings := scanHeadings(content)
	if len(headings) == 0 {
		return nil
	}

	flat := computeSections(content, headings)
	return buildTree(flat)
}

// rawHeading is a heading found during scanning.
type rawHeading struct {
	level      int
	title      string
	byteOffset int // byte offset of the heading line start
}

// scanHeadings scans content for ATX headings, skipping fenced code blocks.
func scanHeadings(content []byte) []rawHeading {
	var headings []rawHeading
	inCodeBlock := false
	i := 0

	for i < len(content) {
		lineStart := i
		lineEnd := findLineEnd(content, i)
		line := content[lineStart:lineEnd]

		// Check for fenced code block toggle (``` at start of line, possibly with info string).
		if isFenceLine(line) {
			inCodeBlock = !inCodeBlock
			i = advancePastNewline(content, lineEnd)
			continue
		}

		if !inCodeBlock {
			if level, title, ok := parseATXHeading(line); ok {
				headings = append(headings, rawHeading{
					level:      level,
					title:      title,
					byteOffset: lineStart,
				})
			}
		}

		i = advancePastNewline(content, lineEnd)
	}

	return headings
}

// findLineEnd returns the index of the first \r or \n at or after pos,
// or len(content) if no line ending is found.
func findLineEnd(content []byte, pos int) int {
	for i := pos; i < len(content); i++ {
		if content[i] == '\n' || content[i] == '\r' {
			return i
		}
	}
	return len(content)
}

// advancePastNewline returns the index just past the newline sequence at lineEnd.
// Handles \n, \r\n, and \r.
func advancePastNewline(content []byte, lineEnd int) int {
	if lineEnd >= len(content) {
		return len(content)
	}
	if content[lineEnd] == '\r' {
		if lineEnd+1 < len(content) && content[lineEnd+1] == '\n' {
			return lineEnd + 2
		}
		return lineEnd + 1
	}
	if content[lineEnd] == '\n' {
		return lineEnd + 1
	}
	return lineEnd
}

// isFenceLine returns true if the trimmed line starts with at least three backticks.
func isFenceLine(line []byte) bool {
	trimmed := bytes.TrimLeft(line, " \t")
	if len(trimmed) < 3 {
		return false
	}
	return trimmed[0] == '`' && trimmed[1] == '`' && trimmed[2] == '`'
}

// parseATXHeading parses a line as an ATX heading.
// Returns the level (1-6), the title text, and whether it matched.
func parseATXHeading(line []byte) (int, string, bool) {
	trimmed := bytes.TrimLeft(line, " \t")
	if len(trimmed) == 0 || trimmed[0] != '#' {
		return 0, "", false
	}

	level := 0
	for level < len(trimmed) && trimmed[level] == '#' {
		level++
	}

	if level > 6 {
		return 0, "", false
	}

	// After the #s there must be a space or end of line.
	rest := trimmed[level:]
	if len(rest) == 0 {
		// Bare heading like "##" with no title — valid ATX heading with empty title.
		return level, "", true
	}
	if rest[0] != ' ' && rest[0] != '\t' {
		return 0, "", false
	}

	title := string(bytes.TrimSpace(rest))
	// Strip optional trailing #s per ATX spec.
	title = stripTrailingHashes(title)
	return level, title, true
}

// stripTrailingHashes removes optional trailing # characters from a heading title.
func stripTrailingHashes(title string) string {
	if len(title) == 0 {
		return title
	}
	end := len(title) - 1
	for end >= 0 && title[end] == '#' {
		end--
	}
	// Only strip if the hashes are preceded by a space (or the entire title is hashes).
	if end < 0 {
		return ""
	}
	if title[end] == ' ' || title[end] == '\t' {
		return title[:end]
	}
	return title
}

// flatSection is an intermediate representation before tree building.
type flatSection struct {
	level      int
	title      string
	byteOffset int
	byteCount  int
	wordCount  int
	hash       string
}

// computeSections determines the byte range of each section and computes metrics.
// A section runs from its heading line start to just before the next heading
// of the same or higher (lower number) level, or EOF.
func computeSections(content []byte, headings []rawHeading) []flatSection {
	sections := make([]flatSection, len(headings))

	for i, h := range headings {
		start := h.byteOffset

		// Find end: next heading of same or higher level, or EOF.
		end := len(content)
		for j := i + 1; j < len(headings); j++ {
			if headings[j].level <= h.level {
				end = headings[j].byteOffset
				break
			}
		}

		sectionContent := content[start:end]

		hash := sha256.Sum256(sectionContent)

		sections[i] = flatSection{
			level:      h.level,
			title:      h.title,
			byteOffset: start,
			byteCount:  end - start,
			wordCount:  countWords(sectionContent),
			hash:       hex.EncodeToString(hash[:]),
		}
	}

	return sections
}

// countWords counts whitespace-separated tokens in content.
func countWords(content []byte) int {
	return len(bytes.Fields(content))
}

// buildTree converts a flat section list into a hierarchical tree with paths.
func buildTree(flat []flatSection) []Section {
	if len(flat) == 0 {
		return nil
	}

	// We use a stack-based approach to build the tree.
	// Each entry on the stack tracks the section and its child counter.
	type stackEntry struct {
		section  *Section
		level    int
		childSeq int // next child sequence number
	}

	var roots []Section
	var stack []stackEntry
	rootSeq := 0

	for i := range flat {
		f := &flat[i]

		// Pop stack entries that are at the same level or deeper than current.
		for len(stack) > 0 && stack[len(stack)-1].level >= f.level {
			stack = stack[:len(stack)-1]
		}

		s := Section{
			Level:       f.level,
			Title:       f.title,
			ByteOffset:  f.byteOffset,
			ByteCount:   f.byteCount,
			WordCount:   f.wordCount,
			ContentHash: f.hash,
		}

		if len(stack) == 0 {
			// Top-level section.
			rootSeq++
			s.Path = fmt.Sprintf("%d", rootSeq)
			roots = append(roots, s)
			stack = append(stack, stackEntry{
				section: &roots[len(roots)-1],
				level:   f.level,
			})
		} else {
			parent := stack[len(stack)-1].section
			stack[len(stack)-1].childSeq++
			seq := stack[len(stack)-1].childSeq
			s.Path = fmt.Sprintf("%s.%d", parent.Path, seq)
			parent.Children = append(parent.Children, s)
			stack = append(stack, stackEntry{
				section: &parent.Children[len(parent.Children)-1],
				level:   f.level,
			})
		}
	}

	return roots
}
