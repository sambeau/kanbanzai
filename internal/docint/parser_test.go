package docint

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"
)

func TestParseStructure_EmptyDocument(t *testing.T) {
	result := ParseStructure([]byte{})
	if len(result) != 0 {
		t.Errorf("expected empty slice, got %d sections", len(result))
	}

	result = ParseStructure(nil)
	if len(result) != 0 {
		t.Errorf("expected empty slice for nil input, got %d sections", len(result))
	}
}

func TestParseStructure_NoHeadings(t *testing.T) {
	content := []byte("Just some text.\nNo headings here.\nAnother line.\n")
	result := ParseStructure(content)
	if len(result) != 0 {
		t.Errorf("expected empty slice, got %d sections", len(result))
	}
}

func TestParseStructure_SingleH1(t *testing.T) {
	content := []byte("# Hello World\n\nSome body text here.\n")
	result := ParseStructure(content)

	if len(result) != 1 {
		t.Fatalf("expected 1 section, got %d", len(result))
	}

	s := result[0]
	if s.Level != 1 {
		t.Errorf("expected level 1, got %d", s.Level)
	}
	if s.Title != "Hello World" {
		t.Errorf("expected title %q, got %q", "Hello World", s.Title)
	}
	if s.Path != "1" {
		t.Errorf("expected path %q, got %q", "1", s.Path)
	}
	if s.ByteOffset != 0 {
		t.Errorf("expected byte offset 0, got %d", s.ByteOffset)
	}
	if s.ByteCount != len(content) {
		t.Errorf("expected byte count %d, got %d", len(content), s.ByteCount)
	}
	// "# Hello World", "", "Some body text here." = 6 words
	if s.WordCount < 4 {
		t.Errorf("expected word count >= 4, got %d", s.WordCount)
	}
	if s.ContentHash == "" {
		t.Error("expected non-empty content hash")
	}
	if len(s.Children) != 0 {
		t.Errorf("expected no children, got %d", len(s.Children))
	}
}

func TestParseStructure_NestedHeadings(t *testing.T) {
	content := []byte("# Parent\n\nIntro text.\n\n## Child One\n\nChild one body.\n\n## Child Two\n\nChild two body.\n")
	result := ParseStructure(content)

	if len(result) != 1 {
		t.Fatalf("expected 1 root section, got %d", len(result))
	}

	root := result[0]
	if root.Level != 1 {
		t.Errorf("expected root level 1, got %d", root.Level)
	}
	if root.Title != "Parent" {
		t.Errorf("expected root title %q, got %q", "Parent", root.Title)
	}
	if root.Path != "1" {
		t.Errorf("expected root path %q, got %q", "1", root.Path)
	}
	if len(root.Children) != 2 {
		t.Fatalf("expected 2 children, got %d", len(root.Children))
	}

	child1 := root.Children[0]
	if child1.Level != 2 {
		t.Errorf("child1: expected level 2, got %d", child1.Level)
	}
	if child1.Title != "Child One" {
		t.Errorf("child1: expected title %q, got %q", "Child One", child1.Title)
	}
	if child1.Path != "1.1" {
		t.Errorf("child1: expected path %q, got %q", "1.1", child1.Path)
	}

	child2 := root.Children[1]
	if child2.Level != 2 {
		t.Errorf("child2: expected level 2, got %d", child2.Level)
	}
	if child2.Title != "Child Two" {
		t.Errorf("child2: expected title %q, got %q", "Child Two", child2.Title)
	}
	if child2.Path != "1.2" {
		t.Errorf("child2: expected path %q, got %q", "1.2", child2.Path)
	}
}

func TestParseStructure_MultipleTopLevel(t *testing.T) {
	content := []byte("# First\n\nBody one.\n\n# Second\n\nBody two.\n\n# Third\n\nBody three.\n")
	result := ParseStructure(content)

	if len(result) != 3 {
		t.Fatalf("expected 3 root sections, got %d", len(result))
	}

	expectations := []struct {
		path  string
		title string
		level int
	}{
		{"1", "First", 1},
		{"2", "Second", 1},
		{"3", "Third", 1},
	}

	for i, exp := range expectations {
		s := result[i]
		if s.Path != exp.path {
			t.Errorf("section %d: expected path %q, got %q", i, exp.path, s.Path)
		}
		if s.Title != exp.title {
			t.Errorf("section %d: expected title %q, got %q", i, exp.title, s.Title)
		}
		if s.Level != exp.level {
			t.Errorf("section %d: expected level %d, got %d", i, exp.level, s.Level)
		}
	}
}

func TestParseStructure_DeeplyNested(t *testing.T) {
	content := []byte("# Top\n\n## Mid\n\n### Deep\n\nDeep body.\n")
	result := ParseStructure(content)

	if len(result) != 1 {
		t.Fatalf("expected 1 root, got %d", len(result))
	}

	root := result[0]
	if root.Path != "1" {
		t.Errorf("root: expected path %q, got %q", "1", root.Path)
	}
	if root.Title != "Top" {
		t.Errorf("root: expected title %q, got %q", "Top", root.Title)
	}

	if len(root.Children) != 1 {
		t.Fatalf("root: expected 1 child, got %d", len(root.Children))
	}

	mid := root.Children[0]
	if mid.Path != "1.1" {
		t.Errorf("mid: expected path %q, got %q", "1.1", mid.Path)
	}
	if mid.Title != "Mid" {
		t.Errorf("mid: expected title %q, got %q", "Mid", mid.Title)
	}

	if len(mid.Children) != 1 {
		t.Fatalf("mid: expected 1 child, got %d", len(mid.Children))
	}

	deep := mid.Children[0]
	if deep.Path != "1.1.1" {
		t.Errorf("deep: expected path %q, got %q", "1.1.1", deep.Path)
	}
	if deep.Title != "Deep" {
		t.Errorf("deep: expected title %q, got %q", "Deep", deep.Title)
	}
	if len(deep.Children) != 0 {
		t.Errorf("deep: expected 0 children, got %d", len(deep.Children))
	}
}

func TestParseStructure_CodeBlockSkipping(t *testing.T) {
	content := []byte("# Real Heading\n\nSome text.\n\n```\n# Not A Heading\n## Also Not\n```\n\nMore text.\n")
	result := ParseStructure(content)

	if len(result) != 1 {
		t.Fatalf("expected 1 section (headings in code block ignored), got %d", len(result))
	}

	if result[0].Title != "Real Heading" {
		t.Errorf("expected title %q, got %q", "Real Heading", result[0].Title)
	}

	// The section should span the entire content since there's only one real heading.
	if result[0].ByteCount != len(content) {
		t.Errorf("expected byte count %d, got %d", len(content), result[0].ByteCount)
	}
}

func TestParseStructure_CodeBlockWithInfoString(t *testing.T) {
	content := []byte("# Title\n\n```go\n# comment in go code\nfunc main() {}\n```\n\n## Real Sub\n\nBody.\n")
	result := ParseStructure(content)

	if len(result) != 1 {
		t.Fatalf("expected 1 root, got %d", len(result))
	}

	if len(result[0].Children) != 1 {
		t.Fatalf("expected 1 child, got %d", len(result[0].Children))
	}

	if result[0].Children[0].Title != "Real Sub" {
		t.Errorf("expected child title %q, got %q", "Real Sub", result[0].Children[0].Title)
	}
}

func TestParseStructure_ContentHashDeterministic(t *testing.T) {
	content := []byte("# Deterministic\n\nSame content every time.\n")

	result1 := ParseStructure(content)
	result2 := ParseStructure(content)

	if len(result1) != 1 || len(result2) != 1 {
		t.Fatal("expected 1 section from each parse")
	}

	if result1[0].ContentHash != result2[0].ContentHash {
		t.Errorf("hashes differ: %q vs %q", result1[0].ContentHash, result2[0].ContentHash)
	}

	// Verify it's a valid SHA-256 hex string.
	if len(result1[0].ContentHash) != 64 {
		t.Errorf("expected 64-char hex hash, got %d chars", len(result1[0].ContentHash))
	}

	// Verify the hash matches manual computation.
	h := sha256.Sum256(content)
	expected := hex.EncodeToString(h[:])
	if result1[0].ContentHash != expected {
		t.Errorf("hash mismatch: got %q, want %q", result1[0].ContentHash, expected)
	}
}

func TestParseStructure_ByteOffsets(t *testing.T) {
	content := []byte("Preamble text.\n\n# First\n\nBody.\n\n# Second\n\nMore body.\n")
	result := ParseStructure(content)

	if len(result) != 2 {
		t.Fatalf("expected 2 sections, got %d", len(result))
	}

	// First heading starts after "Preamble text.\n\n"
	expectedOffset1 := len("Preamble text.\n\n")
	if result[0].ByteOffset != expectedOffset1 {
		t.Errorf("first section: expected offset %d, got %d", expectedOffset1, result[0].ByteOffset)
	}

	// Second heading starts after "Preamble text.\n\n# First\n\nBody.\n\n"
	expectedOffset2 := len("Preamble text.\n\n# First\n\nBody.\n\n")
	if result[1].ByteOffset != expectedOffset2 {
		t.Errorf("second section: expected offset %d, got %d", expectedOffset2, result[1].ByteOffset)
	}

	// First section byte count runs from its heading to the start of the second heading.
	expectedByteCount1 := expectedOffset2 - expectedOffset1
	if result[0].ByteCount != expectedByteCount1 {
		t.Errorf("first section: expected byte count %d, got %d", expectedByteCount1, result[0].ByteCount)
	}

	// Second section byte count runs from its heading to EOF.
	expectedByteCount2 := len(content) - expectedOffset2
	if result[1].ByteCount != expectedByteCount2 {
		t.Errorf("second section: expected byte count %d, got %d", expectedByteCount2, result[1].ByteCount)
	}
}

func TestParseStructure_WordCount(t *testing.T) {
	content := []byte("# Title\n\nOne two three four five.\n")
	result := ParseStructure(content)

	if len(result) != 1 {
		t.Fatalf("expected 1 section, got %d", len(result))
	}

	// Words: "#", "Title", "One", "two", "three", "four", "five." = 7
	// The exact count depends on how we tokenize, but should be reasonable.
	s := result[0]
	if s.WordCount < 5 || s.WordCount > 10 {
		t.Errorf("expected word count between 5 and 10, got %d", s.WordCount)
	}
}

func TestParseStructure_PreambleIgnored(t *testing.T) {
	content := []byte("This is preamble.\nMore preamble.\n\n# Actual Heading\n\nBody.\n")
	result := ParseStructure(content)

	if len(result) != 1 {
		t.Fatalf("expected 1 section, got %d", len(result))
	}

	if result[0].Title != "Actual Heading" {
		t.Errorf("expected title %q, got %q", "Actual Heading", result[0].Title)
	}

	// The preamble bytes should not be included in the section.
	preambleLen := strings.Index(string(content), "# Actual Heading")
	if result[0].ByteOffset != preambleLen {
		t.Errorf("expected offset %d, got %d", preambleLen, result[0].ByteOffset)
	}
}

func TestParseStructure_CRLFLineEndings(t *testing.T) {
	content := []byte("# Title\r\n\r\nBody text.\r\n\r\n## Sub\r\n\r\nSub body.\r\n")
	result := ParseStructure(content)

	if len(result) != 1 {
		t.Fatalf("expected 1 root, got %d", len(result))
	}

	if result[0].Title != "Title" {
		t.Errorf("expected title %q, got %q", "Title", result[0].Title)
	}

	if len(result[0].Children) != 1 {
		t.Fatalf("expected 1 child, got %d", len(result[0].Children))
	}

	if result[0].Children[0].Title != "Sub" {
		t.Errorf("expected child title %q, got %q", "Sub", result[0].Children[0].Title)
	}
}

func TestParseStructure_TrailingHashes(t *testing.T) {
	content := []byte("# Title With Trailing ##\n\nBody.\n")
	result := ParseStructure(content)

	if len(result) != 1 {
		t.Fatalf("expected 1 section, got %d", len(result))
	}

	if result[0].Title != "Title With Trailing" {
		t.Errorf("expected title %q, got %q", "Title With Trailing", result[0].Title)
	}
}

func TestParseStructure_HeadingLevels(t *testing.T) {
	content := []byte("# L1\n## L2\n### L3\n#### L4\n##### L5\n###### L6\n")
	result := ParseStructure(content)

	if len(result) != 1 {
		t.Fatalf("expected 1 root, got %d", len(result))
	}

	// Walk down the tree checking levels.
	current := result[0]
	for expectedLevel := 1; expectedLevel <= 6; expectedLevel++ {
		if current.Level != expectedLevel {
			t.Errorf("expected level %d, got %d", expectedLevel, current.Level)
		}
		if expectedLevel < 6 {
			if len(current.Children) != 1 {
				t.Fatalf("level %d: expected 1 child, got %d", expectedLevel, len(current.Children))
			}
			current = current.Children[0]
		}
	}
}

func TestParseStructure_SevenHashesNotHeading(t *testing.T) {
	content := []byte("# Real\n\n####### Not a heading\n\nBody.\n")
	result := ParseStructure(content)

	if len(result) != 1 {
		t.Fatalf("expected 1 section, got %d", len(result))
	}

	if result[0].Title != "Real" {
		t.Errorf("expected title %q, got %q", "Real", result[0].Title)
	}
}

func TestParseStructure_HashWithoutSpace(t *testing.T) {
	content := []byte("#NotAHeading\n\n# Real Heading\n\nBody.\n")
	result := ParseStructure(content)

	if len(result) != 1 {
		t.Fatalf("expected 1 section, got %d", len(result))
	}

	if result[0].Title != "Real Heading" {
		t.Errorf("expected title %q, got %q", "Real Heading", result[0].Title)
	}
}

func TestParseStructure_SkippedLevels(t *testing.T) {
	// h1 followed by h3 (skipping h2) — h3 should still be a child of h1.
	content := []byte("# Top\n\n### Skipped To Three\n\nBody.\n")
	result := ParseStructure(content)

	if len(result) != 1 {
		t.Fatalf("expected 1 root, got %d", len(result))
	}

	if len(result[0].Children) != 1 {
		t.Fatalf("expected 1 child, got %d", len(result[0].Children))
	}

	child := result[0].Children[0]
	if child.Level != 3 {
		t.Errorf("expected level 3, got %d", child.Level)
	}
	if child.Path != "1.1" {
		t.Errorf("expected path %q, got %q", "1.1", child.Path)
	}
}

func TestParseStructure_MultipleCodeBlocks(t *testing.T) {
	content := []byte("# Start\n\n```\n# fake1\n```\n\nMiddle text.\n\n```\n## fake2\n```\n\n## Real Sub\n\nEnd.\n")
	result := ParseStructure(content)

	if len(result) != 1 {
		t.Fatalf("expected 1 root, got %d", len(result))
	}

	if len(result[0].Children) != 1 {
		t.Fatalf("expected 1 child, got %d", len(result[0].Children))
	}

	if result[0].Children[0].Title != "Real Sub" {
		t.Errorf("expected child title %q, got %q", "Real Sub", result[0].Children[0].Title)
	}
}

func TestParseStructure_IndentedCodeFence(t *testing.T) {
	// Code fence with leading spaces should still be recognised.
	content := []byte("# Title\n\n   ```\n   # Not heading\n   ```\n\nBody.\n")
	result := ParseStructure(content)

	if len(result) != 1 {
		t.Fatalf("expected 1 section, got %d", len(result))
	}

	if result[0].Title != "Title" {
		t.Errorf("expected title %q, got %q", "Title", result[0].Title)
	}
}

func TestParseStructure_SiblingsThenDeeper(t *testing.T) {
	content := []byte("# A\n\n## A1\n\n## A2\n\n### A2a\n\n# B\n\n## B1\n")
	result := ParseStructure(content)

	if len(result) != 2 {
		t.Fatalf("expected 2 roots, got %d", len(result))
	}

	a := result[0]
	if a.Path != "1" || a.Title != "A" {
		t.Errorf("root A: path=%q title=%q", a.Path, a.Title)
	}
	if len(a.Children) != 2 {
		t.Fatalf("A: expected 2 children, got %d", len(a.Children))
	}
	if a.Children[0].Path != "1.1" || a.Children[0].Title != "A1" {
		t.Errorf("A1: path=%q title=%q", a.Children[0].Path, a.Children[0].Title)
	}
	if a.Children[1].Path != "1.2" || a.Children[1].Title != "A2" {
		t.Errorf("A2: path=%q title=%q", a.Children[1].Path, a.Children[1].Title)
	}
	if len(a.Children[1].Children) != 1 {
		t.Fatalf("A2: expected 1 child, got %d", len(a.Children[1].Children))
	}
	if a.Children[1].Children[0].Path != "1.2.1" {
		t.Errorf("A2a: expected path %q, got %q", "1.2.1", a.Children[1].Children[0].Path)
	}

	b := result[1]
	if b.Path != "2" || b.Title != "B" {
		t.Errorf("root B: path=%q title=%q", b.Path, b.Title)
	}
	if len(b.Children) != 1 {
		t.Fatalf("B: expected 1 child, got %d", len(b.Children))
	}
	if b.Children[0].Path != "2.1" || b.Children[0].Title != "B1" {
		t.Errorf("B1: path=%q title=%q", b.Children[0].Path, b.Children[0].Title)
	}
}
