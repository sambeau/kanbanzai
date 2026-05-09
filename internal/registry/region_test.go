package registry

import (
	"strings"
	"testing"
)

// buildFile is a test helper that assembles a Markdown file with a single
// named region and configurable surrounding prose.
func buildFile(prose1, regionName, source, content, prose2 string) string {
	var sb strings.Builder
	sb.WriteString(prose1)
	sb.WriteString("<!-- registry-gen:begin:")
	sb.WriteString(regionName)
	if source != "" {
		sb.WriteString(" source=")
		sb.WriteString(source)
	}
	sb.WriteString(" -->\n")
	sb.WriteString(content)
	sb.WriteString("<!-- registry-gen:end:")
	sb.WriteString(regionName)
	sb.WriteString(" -->\n")
	sb.WriteString(prose2)
	return sb.String()
}

func TestParseRegions_Single(t *testing.T) {
	const filePath = "test.md"
	const inner = "generated line 1\ngenerated line 2\n"
	input := buildFile(
		"Before prose.\n",
		"MY-REGION",
		".kbz/stage-bindings.yaml",
		inner,
		"After prose.\n",
	)

	regions, err := ParseRegions(filePath, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(regions) != 1 {
		t.Fatalf("expected 1 region, got %d", len(regions))
	}
	r := regions[0]
	if r.Name != "MY-REGION" {
		t.Errorf("Name = %q, want %q", r.Name, "MY-REGION")
	}
	if r.Source != ".kbz/stage-bindings.yaml" {
		t.Errorf("Source = %q, want %q", r.Source, ".kbz/stage-bindings.yaml")
	}
	if r.Content != inner {
		t.Errorf("Content = %q, want %q", r.Content, inner)
	}
}

func TestParseRegions_Multiple(t *testing.T) {
	const filePath = "test.md"
	input := buildFile("", "REGION-A", "a.yaml", "content-a\n", "") +
		"middle prose\n" +
		buildFile("", "REGION-B", "b.yaml", "content-b\n", "")

	regions, err := ParseRegions(filePath, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(regions) != 2 {
		t.Fatalf("expected 2 regions, got %d", len(regions))
	}
	if regions[0].Name != "REGION-A" || regions[0].Content != "content-a\n" {
		t.Errorf("region 0: got Name=%q Content=%q", regions[0].Name, regions[0].Content)
	}
	if regions[1].Name != "REGION-B" || regions[1].Content != "content-b\n" {
		t.Errorf("region 1: got Name=%q Content=%q", regions[1].Name, regions[1].Content)
	}
}

func TestParseRegions_EmptyContent(t *testing.T) {
	const filePath = "test.md"
	input := buildFile("", "EMPTY-REGION", "", "", "")

	regions, err := ParseRegions(filePath, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(regions) != 1 {
		t.Fatalf("expected 1 region, got %d", len(regions))
	}
	if regions[0].Content != "" {
		t.Errorf("Content = %q, want empty string", regions[0].Content)
	}
}

func TestParseRegions_MissingEndMarker(t *testing.T) {
	const filePath = "docs.md"
	input := "prose\n<!-- registry-gen:begin:MY-REGION source=s.yaml -->\ncontent\n"

	_, err := ParseRegions(filePath, input)
	if err == nil {
		t.Fatal("expected error for missing end marker, got nil")
	}
	if !strings.Contains(err.Error(), filePath) {
		t.Errorf("error %q should contain file path %q", err.Error(), filePath)
	}
	if !strings.Contains(err.Error(), "MY-REGION") {
		t.Errorf("error %q should contain region name %q", err.Error(), "MY-REGION")
	}
}

func TestParseRegions_MissingBeginMarker(t *testing.T) {
	const filePath = "docs.md"
	input := "prose\n<!-- registry-gen:end:MY-REGION -->\n"

	_, err := ParseRegions(filePath, input)
	if err == nil {
		t.Fatal("expected error for end marker with no matching begin, got nil")
	}
	if !strings.Contains(err.Error(), filePath) {
		t.Errorf("error %q should contain file path %q", err.Error(), filePath)
	}
	if !strings.Contains(err.Error(), "MY-REGION") {
		t.Errorf("error %q should contain region name %q", err.Error(), "MY-REGION")
	}
}

func TestParseRegions_DuplicatedMarker(t *testing.T) {
	const filePath = "docs.md"
	// Two separate regions with the same name.
	block := buildFile("", "DUP-REGION", "", "first\n", "")
	input := block + buildFile("", "DUP-REGION", "", "second\n", "")

	_, err := ParseRegions(filePath, input)
	if err == nil {
		t.Fatal("expected error for duplicated marker, got nil")
	}
	if !strings.Contains(err.Error(), filePath) {
		t.Errorf("error %q should contain file path %q", err.Error(), filePath)
	}
	if !strings.Contains(err.Error(), "DUP-REGION") {
		t.Errorf("error %q should contain region name %q", err.Error(), "DUP-REGION")
	}
}

func TestParseRegions_NestedMarker(t *testing.T) {
	const filePath = "docs.md"
	input := "<!-- registry-gen:begin:OUTER source=a.yaml -->\n" +
		"some text\n" +
		"<!-- registry-gen:begin:INNER source=b.yaml -->\n" +
		"inner text\n" +
		"<!-- registry-gen:end:INNER -->\n" +
		"<!-- registry-gen:end:OUTER -->\n"

	_, err := ParseRegions(filePath, input)
	if err == nil {
		t.Fatal("expected error for nested marker, got nil")
	}
	if !strings.Contains(err.Error(), filePath) {
		t.Errorf("error %q should contain file path %q", err.Error(), filePath)
	}
	if !strings.Contains(err.Error(), "INNER") {
		t.Errorf("error %q should contain nested region name %q", err.Error(), "INNER")
	}
}

func TestSyncRegion_Replace(t *testing.T) {
	const filePath = "test.md"
	const candidate = "replaced content\n"
	input := buildFile("before\n", "R", "s.yaml", "original content\n", "after\n")

	result, err := SyncRegion(filePath, input, "R", candidate)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := buildFile("before\n", "R", "s.yaml", candidate, "after\n")
	if result != expected {
		t.Errorf("SyncRegion result:\ngot:  %q\nwant: %q", result, expected)
	}
}

func TestSyncRegion_PreservesSurroundingProse(t *testing.T) {
	const filePath = "test.md"
	// Use prose with characters that must not be altered in any way.
	const prefix = "# Heading\n\nThis text has **bold**, _italic_, and `code`.\n\n"
	const suffix = "\n---\n\nFooter text with [link](https://example.com).\n"
	const candidate = "new generated content\n"

	input := buildFile(prefix, "PROSE-TEST", "s.yaml", "old\n", suffix)
	result, err := SyncRegion(filePath, input, "PROSE-TEST", candidate)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.HasPrefix(result, prefix) {
		t.Errorf("prefix not preserved byte-for-byte: got %q", result[:len(prefix)])
	}
	if !strings.HasSuffix(result, suffix) {
		t.Errorf("suffix not preserved byte-for-byte: got last %d bytes = %q", len(suffix), result[len(result)-len(suffix):])
	}
	// Verify prefix bytes explicitly.
	for i := 0; i < len(prefix); i++ {
		if result[i] != prefix[i] {
			t.Errorf("byte %d differs: got 0x%02x want 0x%02x", i, result[i], prefix[i])
		}
	}
	// Verify suffix bytes explicitly.
	offset := len(result) - len(suffix)
	for i := 0; i < len(suffix); i++ {
		if result[offset+i] != suffix[i] {
			t.Errorf("suffix byte %d differs: got 0x%02x want 0x%02x", i, result[offset+i], suffix[i])
		}
	}
}

func TestSyncRegion_Idempotency(t *testing.T) {
	const filePath = "test.md"
	const candidate = "idempotent content line 1\nidempotent content line 2\n"
	input := buildFile("prose before\n", "IDEM-REGION", "s.yaml", "original\n", "prose after\n")

	result1, err := SyncRegion(filePath, input, "IDEM-REGION", candidate)
	if err != nil {
		t.Fatalf("first sync error: %v", err)
	}

	result2, err := SyncRegion(filePath, result1, "IDEM-REGION", candidate)
	if err != nil {
		t.Fatalf("second sync error: %v", err)
	}

	if result1 != result2 {
		t.Errorf("sync is not idempotent:\nfirst:  %q\nsecond: %q", result1, result2)
	}
}

func TestCheckRegion_Stale(t *testing.T) {
	const filePath = "check.md"
	const candidate = "expected content\n"
	input := buildFile("", "CHECK-R", "s.yaml", "current content\n", "")

	stale, region, file, err := CheckRegion(filePath, input, "CHECK-R", candidate)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !stale {
		t.Error("expected stale=true when content differs")
	}
	if region != "CHECK-R" {
		t.Errorf("region = %q, want %q", region, "CHECK-R")
	}
	if file != filePath {
		t.Errorf("file = %q, want %q", file, filePath)
	}
}

func TestCheckRegion_NotStale(t *testing.T) {
	const filePath = "check.md"
	const candidate = "current content\n"
	input := buildFile("", "CHECK-R", "s.yaml", candidate, "")

	stale, _, _, err := CheckRegion(filePath, input, "CHECK-R", candidate)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stale {
		t.Error("expected stale=false when content matches")
	}
}

func TestSyncRegion_RegionNotFound(t *testing.T) {
	const filePath = "test.md"
	input := buildFile("", "EXISTING", "s.yaml", "content\n", "")

	_, err := SyncRegion(filePath, input, "NONEXISTENT", "new\n")
	if err == nil {
		t.Fatal("expected error for nonexistent region, got nil")
	}
	if !strings.Contains(err.Error(), "NONEXISTENT") {
		t.Errorf("error %q should contain region name", err.Error())
	}
}

func TestCheckRegion_PropagatesParseError(t *testing.T) {
	const filePath = "broken.md"
	// A file with only a begin marker — will fail to parse.
	input := "<!-- registry-gen:begin:R source=s.yaml -->\ncontent\n"

	_, _, _, err := CheckRegion(filePath, input, "R", "anything\n")
	if err == nil {
		t.Fatal("expected parse error to be propagated, got nil")
	}
}

func TestParseRegions_NoTrailingNewlineAfterEndMarker(t *testing.T) {
	// End marker is the last line with no trailing newline.
	const filePath = "test.md"
	input := "<!-- registry-gen:begin:R source=s.yaml -->\ncontent\n<!-- registry-gen:end:R -->"

	regions, err := ParseRegions(filePath, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(regions) != 1 {
		t.Fatalf("expected 1 region, got %d", len(regions))
	}
	if regions[0].Content != "content\n" {
		t.Errorf("Content = %q, want %q", regions[0].Content, "content\n")
	}
}

func TestParseRegions_MarkerWithLeadingWhitespace(t *testing.T) {
	// Markers with leading spaces should still be detected via TrimSpace.
	const filePath = "test.md"
	input := "   <!-- registry-gen:begin:WS-REGION source=s.yaml -->\ncontent\n   <!-- registry-gen:end:WS-REGION -->\n"

	regions, err := ParseRegions(filePath, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(regions) != 1 {
		t.Fatalf("expected 1 region, got %d", len(regions))
	}
	if regions[0].Name != "WS-REGION" {
		t.Errorf("Name = %q, want WS-REGION", regions[0].Name)
	}
}
