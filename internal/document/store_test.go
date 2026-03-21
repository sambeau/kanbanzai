package document

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func fixedTime(t *testing.T) time.Time {
	t.Helper()
	ts, err := time.Parse(time.RFC3339, "2025-06-15T10:30:00Z")
	if err != nil {
		t.Fatalf("parse fixed time: %v", err)
	}
	return ts
}

func sampleDoc(t *testing.T) Document {
	t.Helper()
	ts := fixedTime(t)
	return Document{
		Meta: DocMeta{
			ID:        "DOC-01J3KDOCTEST01",
			Type:      DocTypeProposal,
			Title:     "Widget Redesign",
			Status:    DocStatusDraft,
			Feature:   "FEAT-01J3K7MXP3RT5",
			CreatedBy: "alice",
			Created:   ts,
			Updated:   ts,
		},
		Body: "# Widget Redesign\n\nThis proposal covers the widget redesign.\n",
	}
}

func sampleDocWithApproval(t *testing.T) Document {
	t.Helper()
	doc := sampleDoc(t)
	doc.Meta.Status = DocStatusApproved
	doc.Meta.ApprovedBy = "bob"
	approvedAt := doc.Meta.Created.Add(48 * time.Hour)
	doc.Meta.ApprovedAt = &approvedAt
	return doc
}

func assertDocEqual(t *testing.T, want, got Document) {
	t.Helper()
	if got.Meta.ID != want.Meta.ID {
		t.Errorf("Meta.ID = %q, want %q", got.Meta.ID, want.Meta.ID)
	}
	if got.Meta.Type != want.Meta.Type {
		t.Errorf("Meta.Type = %q, want %q", got.Meta.Type, want.Meta.Type)
	}
	if got.Meta.Title != want.Meta.Title {
		t.Errorf("Meta.Title = %q, want %q", got.Meta.Title, want.Meta.Title)
	}
	if got.Meta.Status != want.Meta.Status {
		t.Errorf("Meta.Status = %q, want %q", got.Meta.Status, want.Meta.Status)
	}
	if got.Meta.Feature != want.Meta.Feature {
		t.Errorf("Meta.Feature = %q, want %q", got.Meta.Feature, want.Meta.Feature)
	}
	if got.Meta.CreatedBy != want.Meta.CreatedBy {
		t.Errorf("Meta.CreatedBy = %q, want %q", got.Meta.CreatedBy, want.Meta.CreatedBy)
	}
	if !got.Meta.Created.Equal(want.Meta.Created) {
		t.Errorf("Meta.Created = %v, want %v", got.Meta.Created, want.Meta.Created)
	}
	if !got.Meta.Updated.Equal(want.Meta.Updated) {
		t.Errorf("Meta.Updated = %v, want %v", got.Meta.Updated, want.Meta.Updated)
	}
	if got.Meta.ApprovedBy != want.Meta.ApprovedBy {
		t.Errorf("Meta.ApprovedBy = %q, want %q", got.Meta.ApprovedBy, want.Meta.ApprovedBy)
	}
	if (want.Meta.ApprovedAt == nil) != (got.Meta.ApprovedAt == nil) {
		t.Errorf("Meta.ApprovedAt nil mismatch: got nil=%v, want nil=%v",
			got.Meta.ApprovedAt == nil, want.Meta.ApprovedAt == nil)
	} else if want.Meta.ApprovedAt != nil && !got.Meta.ApprovedAt.Equal(*want.Meta.ApprovedAt) {
		t.Errorf("Meta.ApprovedAt = %v, want %v", *got.Meta.ApprovedAt, *want.Meta.ApprovedAt)
	}
	if got.Body != want.Body {
		t.Errorf("Body mismatch\ngot:\n%s\nwant:\n%s", got.Body, want.Body)
	}
}

// --- slugify tests ---

func TestSlugify_BasicConversion(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "simple title", input: "Widget Redesign", want: "widget-redesign"},
		{name: "already lowercase", input: "hello-world", want: "hello-world"},
		{name: "underscores to dashes", input: "foo_bar_baz", want: "foo-bar-baz"},
		{name: "special characters removed", input: "hello! @world #2025", want: "hello-world-2025"},
		{name: "multiple spaces", input: "too   many   spaces", want: "too-many-spaces"},
		{name: "leading and trailing spaces", input: "  padded  ", want: "padded"},
		{name: "leading and trailing dashes", input: "--dashed--", want: "dashed"},
		{name: "mixed special chars", input: "My Doc: A (Great) Plan!", want: "my-doc-a-great-plan"},
		{name: "numbers preserved", input: "Phase 1 Plan 2025", want: "phase-1-plan-2025"},
		{name: "consecutive special chars collapse", input: "a!!!b", want: "a-b"},
		{name: "empty string", input: "", want: ""},
		{name: "only special characters", input: "!@#$%", want: ""},
		{name: "single word", input: "Hello", want: "hello"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := slugify(tt.input)
			if got != tt.want {
				t.Errorf("slugify(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// --- docFileName tests ---

func TestDocFileName_Format(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		meta DocMeta
		want string
	}{
		{
			name: "basic",
			meta: DocMeta{ID: "DOC-01J3KDOCTEST01", Title: "Widget Redesign"},
			want: "DOC-01J3KDOCTEST01-widget-redesign.md",
		},
		{
			name: "title with special chars",
			meta: DocMeta{ID: "DOC-01J3KDOCFM042", Title: "My Great Plan!"},
			want: "DOC-01J3KDOCFM042-my-great-plan.md",
		},
		{
			name: "title with spaces",
			meta: DocMeta{ID: "DOC-01J3KDOCFM100", Title: "Phase 1 Implementation Plan"},
			want: "DOC-01J3KDOCFM100-phase-1-implementation-plan.md",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := docFileName(tt.meta)
			if got != tt.want {
				t.Errorf("docFileName() = %q, want %q", got, tt.want)
			}
		})
	}
}

// --- marshalDocument / unmarshalDocument tests ---

func TestMarshalDocument_RoundTrip(t *testing.T) {
	t.Parallel()

	doc := sampleDoc(t)
	serialised := marshalDocument(doc)
	got, err := unmarshalDocument(serialised)
	if err != nil {
		t.Fatalf("unmarshalDocument() error = %v", err)
	}

	assertDocEqual(t, doc, got)
}

func TestMarshalDocument_RoundTripWithApproval(t *testing.T) {
	t.Parallel()

	doc := sampleDocWithApproval(t)
	serialised := marshalDocument(doc)
	got, err := unmarshalDocument(serialised)
	if err != nil {
		t.Fatalf("unmarshalDocument() error = %v", err)
	}

	assertDocEqual(t, doc, got)
}

func TestMarshalDocument_OmitsEmptyOptionalFields(t *testing.T) {
	t.Parallel()

	doc := sampleDoc(t)
	doc.Meta.Feature = ""
	doc.Meta.ApprovedBy = ""
	doc.Meta.ApprovedAt = nil

	serialised := marshalDocument(doc)

	if strings.Contains(serialised, "feature:") {
		t.Error("marshalled output should not contain feature when empty")
	}
	if strings.Contains(serialised, "approved_by:") {
		t.Error("marshalled output should not contain approved_by when empty")
	}
	if strings.Contains(serialised, "approved_at:") {
		t.Error("marshalled output should not contain approved_at when empty")
	}
}

func TestMarshalDocument_IncludesOptionalFieldsWhenSet(t *testing.T) {
	t.Parallel()

	doc := sampleDocWithApproval(t)
	serialised := marshalDocument(doc)

	if !strings.Contains(serialised, "feature: FEAT-01J3K7MXP3RT5") {
		t.Error("marshalled output should contain feature when set")
	}
	if !strings.Contains(serialised, "approved_by: bob") {
		t.Error("marshalled output should contain approved_by when set")
	}
	if !strings.Contains(serialised, "approved_at:") {
		t.Error("marshalled output should contain approved_at when set")
	}
}

func TestMarshalDocument_FrontmatterDelimiters(t *testing.T) {
	t.Parallel()

	doc := sampleDoc(t)
	serialised := marshalDocument(doc)

	if !strings.HasPrefix(serialised, "---\n") {
		t.Error("marshalled output should start with ---")
	}
	// The closing --- should appear after the opening one.
	rest := serialised[4:]
	if !strings.Contains(rest, "\n---\n") {
		t.Error("marshalled output should contain closing --- delimiter")
	}
}

func TestMarshalDocument_VerbatimBodyPreservation(t *testing.T) {
	t.Parallel()

	bodies := []struct {
		name string
		body string
	}{
		{
			name: "simple markdown",
			body: "# Title\n\nParagraph text.\n",
		},
		{
			name: "body with special yaml chars",
			body: "key: value\n- list item\n# not a yaml comment\n",
		},
		{
			name: "body with triple dashes",
			body: "Some text\n---\nMore text after horizontal rule\n",
		},
		{
			name: "body with leading whitespace",
			body: "  indented\n    more indented\n",
		},
		{
			name: "empty body",
			body: "",
		},
		{
			name: "body with unicode",
			body: "Héllo wörld 日本語\n",
		},
		{
			name: "multi paragraph",
			body: "First paragraph.\n\nSecond paragraph.\n\nThird paragraph.\n",
		},
	}

	for _, tt := range bodies {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			doc := sampleDoc(t)
			doc.Body = tt.body

			serialised := marshalDocument(doc)
			got, err := unmarshalDocument(serialised)
			if err != nil {
				t.Fatalf("unmarshalDocument() error = %v", err)
			}

			if got.Body != tt.body {
				t.Errorf("body not preserved verbatim\ngot:  %q\nwant: %q", got.Body, tt.body)
			}
		})
	}
}

func TestUnmarshalDocument_MissingOpeningDelimiter(t *testing.T) {
	t.Parallel()

	_, err := unmarshalDocument("no frontmatter here\n")
	if err == nil {
		t.Fatal("expected error for missing opening delimiter")
	}
	if !strings.Contains(err.Error(), "missing frontmatter delimiter") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestUnmarshalDocument_MissingClosingDelimiter(t *testing.T) {
	t.Parallel()

	_, err := unmarshalDocument("---\nid: DOC-01J3KDOCTEST01\ntype: proposal\n")
	if err == nil {
		t.Fatal("expected error for missing closing delimiter")
	}
	if !strings.Contains(err.Error(), "missing closing frontmatter delimiter") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestMarshalDocument_TitleWithColonIsQuoted(t *testing.T) {
	t.Parallel()

	doc := sampleDoc(t)
	doc.Meta.Title = "Design: A New Approach"

	serialised := marshalDocument(doc)
	got, err := unmarshalDocument(serialised)
	if err != nil {
		t.Fatalf("unmarshalDocument() error = %v", err)
	}

	if got.Meta.Title != doc.Meta.Title {
		t.Errorf("title not preserved through quoting round-trip\ngot:  %q\nwant: %q",
			got.Meta.Title, doc.Meta.Title)
	}
}

// --- DocStore.Write tests ---

func TestDocStoreWrite_CreatesTypeSubdirectory(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := NewDocStore(root)
	doc := sampleDoc(t)

	path, err := store.Write(doc)
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	typeDir := filepath.Join(root, string(doc.Meta.Type))
	info, err := os.Stat(typeDir)
	if err != nil {
		t.Fatalf("type directory not created: %v", err)
	}
	if !info.IsDir() {
		t.Fatal("type path is not a directory")
	}

	if _, err := os.Stat(path); err != nil {
		t.Fatalf("written file does not exist at returned path: %v", err)
	}
}

func TestDocStoreWrite_RejectsEmptyID(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := NewDocStore(root)
	doc := sampleDoc(t)
	doc.Meta.ID = ""

	_, err := store.Write(doc)
	if err == nil {
		t.Fatal("expected error for empty ID")
	}
}

func TestDocStoreWrite_RejectsEmptyTitle(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := NewDocStore(root)
	doc := sampleDoc(t)
	doc.Meta.Title = ""

	_, err := store.Write(doc)
	if err == nil {
		t.Fatal("expected error for empty title")
	}
}

func TestDocStoreWrite_RejectsInvalidType(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := NewDocStore(root)
	doc := sampleDoc(t)
	doc.Meta.Type = "invalid-type"

	_, err := store.Write(doc)
	if err == nil {
		t.Fatal("expected error for invalid document type")
	}
}

func TestDocStoreWrite_CorrectFileName(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := NewDocStore(root)
	doc := sampleDoc(t)

	path, err := store.Write(doc)
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	wantName := "DOC-01J3KDOCTEST01-widget-redesign.md"
	gotName := filepath.Base(path)
	if gotName != wantName {
		t.Errorf("file name = %q, want %q", gotName, wantName)
	}
}

// --- DocStore.Load tests ---

func TestDocStoreLoad_RoundTrip(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := NewDocStore(root)
	doc := sampleDoc(t)

	_, err := store.Write(doc)
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	got, err := store.Load(doc.Meta.Type, doc.Meta.ID, doc.Meta.Title)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	assertDocEqual(t, doc, got)
}

func TestDocStoreLoad_RoundTripWithApproval(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := NewDocStore(root)
	doc := sampleDocWithApproval(t)

	_, err := store.Write(doc)
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	got, err := store.Load(doc.Meta.Type, doc.Meta.ID, doc.Meta.Title)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	assertDocEqual(t, doc, got)
}

func TestDocStoreLoad_PreservesBodyVerbatim(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := NewDocStore(root)
	doc := sampleDoc(t)
	doc.Body = "Line one.\n\n  Indented line.\n\n```go\nfunc main() {}\n```\n\n---\n\nAfter rule.\n"

	_, err := store.Write(doc)
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	got, err := store.Load(doc.Meta.Type, doc.Meta.ID, doc.Meta.Title)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if got.Body != doc.Body {
		t.Errorf("body not preserved verbatim\ngot:  %q\nwant: %q", got.Body, doc.Body)
	}
}

func TestDocStoreLoad_NonExistentFile(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := NewDocStore(root)

	_, err := store.Load(DocTypeProposal, "DOC-01J3KDOCNF999", "nonexistent")
	if err == nil {
		t.Fatal("expected error for non-existent document")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' in error, got: %v", err)
	}
}

func TestDocStoreLoad_InvalidType(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := NewDocStore(root)

	_, err := store.Load("bogus", "DOC-01J3KDOCTEST01", "anything")
	if err == nil {
		t.Fatal("expected error for invalid document type")
	}
}

// --- DocStore.LoadByPath tests ---

func TestDocStoreLoadByPath_RoundTrip(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := NewDocStore(root)
	doc := sampleDoc(t)

	path, err := store.Write(doc)
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	got, err := store.LoadByPath(path)
	if err != nil {
		t.Fatalf("LoadByPath() error = %v", err)
	}

	assertDocEqual(t, doc, got)
}

func TestDocStoreLoadByPath_NonExistentFile(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := NewDocStore(root)

	_, err := store.LoadByPath(filepath.Join(root, "does-not-exist.md"))
	if err == nil {
		t.Fatal("expected error for non-existent path")
	}
}

// --- DocStore.List tests ---

func TestDocStoreList_EmptyDirectory(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := NewDocStore(root)

	paths, err := store.List(DocTypeProposal)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(paths) != 0 {
		t.Errorf("expected empty list, got %d items", len(paths))
	}
}

func TestDocStoreList_ReturnsMatchingDocs(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := NewDocStore(root)

	ts := fixedTime(t)
	docs := []Document{
		{
			Meta: DocMeta{
				ID: "DOC-01J3KDOCTEST01", Type: DocTypeProposal, Title: "First Proposal",
				Status: DocStatusDraft, CreatedBy: "alice", Created: ts, Updated: ts,
			},
			Body: "First body.\n",
		},
		{
			Meta: DocMeta{
				ID: "DOC-01J3KDOCTEST02", Type: DocTypeProposal, Title: "Second Proposal",
				Status: DocStatusDraft, CreatedBy: "bob", Created: ts, Updated: ts,
			},
			Body: "Second body.\n",
		},
		{
			Meta: DocMeta{
				ID: "DOC-01J3KDOCTEST03", Type: DocTypeDesign, Title: "A Design",
				Status: DocStatusDraft, CreatedBy: "carol", Created: ts, Updated: ts,
			},
			Body: "Design body.\n",
		},
	}

	for _, d := range docs {
		if _, err := store.Write(d); err != nil {
			t.Fatalf("Write() error = %v", err)
		}
	}

	proposals, err := store.List(DocTypeProposal)
	if err != nil {
		t.Fatalf("List(proposal) error = %v", err)
	}
	if len(proposals) != 2 {
		t.Errorf("expected 2 proposals, got %d", len(proposals))
	}

	designs, err := store.List(DocTypeDesign)
	if err != nil {
		t.Fatalf("List(design) error = %v", err)
	}
	if len(designs) != 1 {
		t.Errorf("expected 1 design, got %d", len(designs))
	}
}

func TestDocStoreList_InvalidType(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := NewDocStore(root)

	_, err := store.List("not-a-type")
	if err == nil {
		t.Fatal("expected error for invalid document type")
	}
}

// --- DocStore.ListAll tests ---

func TestDocStoreListAll_Empty(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := NewDocStore(root)

	paths, err := store.ListAll()
	if err != nil {
		t.Fatalf("ListAll() error = %v", err)
	}
	if len(paths) != 0 {
		t.Errorf("expected empty list, got %d items", len(paths))
	}
}

func TestDocStoreListAll_CombinesAcrossTypes(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := NewDocStore(root)

	ts := fixedTime(t)
	types := []DocType{DocTypeProposal, DocTypeDesign, DocTypeSpecification}
	docIDs := []string{"DOC-01J3KDOCLA001", "DOC-01J3KDOCLA002", "DOC-01J3KDOCLA003"}
	for i, dt := range types {
		doc := Document{
			Meta: DocMeta{
				ID:        docIDs[i],
				Type:      dt,
				Title:     "Doc " + string(dt),
				Status:    DocStatusDraft,
				CreatedBy: "alice",
				Created:   ts,
				Updated:   ts,
			},
			Body: "Body for " + string(dt) + ".\n",
		}
		if _, err := store.Write(doc); err != nil {
			t.Fatalf("Write(%s) error = %v", dt, err)
		}
	}

	paths, err := store.ListAll()
	if err != nil {
		t.Fatalf("ListAll() error = %v", err)
	}
	if len(paths) != 3 {
		t.Errorf("expected 3 documents across all types, got %d", len(paths))
	}
}

// --- Write then LoadByPath for each doc type ---

func TestDocStoreWrite_AllDocTypes(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := NewDocStore(root)
	ts := fixedTime(t)

	for _, dt := range AllDocTypes() {
		t.Run(string(dt), func(t *testing.T) {
			t.Parallel()
			doc := Document{
				Meta: DocMeta{
					ID:        "DOC-01J3KDOCSTN01",
					Type:      dt,
					Title:     "Test " + string(dt),
					Status:    DocStatusDraft,
					CreatedBy: "tester",
					Created:   ts,
					Updated:   ts,
				},
				Body: "Body for " + string(dt) + ".\n",
			}

			path, err := store.Write(doc)
			if err != nil {
				t.Fatalf("Write() error = %v", err)
			}

			// Verify the type subdirectory is correct.
			dir := filepath.Dir(path)
			if filepath.Base(dir) != string(dt) {
				t.Errorf("document written to wrong type dir: got %q, want %q",
					filepath.Base(dir), string(dt))
			}

			got, err := store.LoadByPath(path)
			if err != nil {
				t.Fatalf("LoadByPath() error = %v", err)
			}

			assertDocEqual(t, doc, got)
		})
	}
}

// --- marshalDocument idempotence ---

func TestMarshalDocument_Idempotent(t *testing.T) {
	t.Parallel()

	doc := sampleDocWithApproval(t)
	first := marshalDocument(doc)

	roundTripped, err := unmarshalDocument(first)
	if err != nil {
		t.Fatalf("unmarshalDocument() error = %v", err)
	}

	second := marshalDocument(roundTripped)

	if first != second {
		t.Errorf("marshal is not idempotent\nfirst:\n%s\nsecond:\n%s", first, second)
	}
}

// --- Write overwrites existing file ---

func TestDocStoreWrite_OverwritesExistingFile(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := NewDocStore(root)
	doc := sampleDoc(t)

	path1, err := store.Write(doc)
	if err != nil {
		t.Fatalf("first Write() error = %v", err)
	}

	doc.Body = "Updated body content.\n"
	doc.Meta.Status = DocStatusSubmitted

	path2, err := store.Write(doc)
	if err != nil {
		t.Fatalf("second Write() error = %v", err)
	}

	if path1 != path2 {
		t.Errorf("overwrite path changed: %q -> %q", path1, path2)
	}

	got, err := store.LoadByPath(path2)
	if err != nil {
		t.Fatalf("LoadByPath() error = %v", err)
	}

	if got.Body != "Updated body content.\n" {
		t.Errorf("body not updated: got %q", got.Body)
	}
	if got.Meta.Status != DocStatusSubmitted {
		t.Errorf("status not updated: got %q", got.Meta.Status)
	}
}

// --- WhitespaceOnly ID and Title ---

func TestDocStoreWrite_RejectsWhitespaceOnlyID(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := NewDocStore(root)
	doc := sampleDoc(t)
	doc.Meta.ID = "   "

	_, err := store.Write(doc)
	if err == nil {
		t.Fatal("expected error for whitespace-only ID")
	}
}

func TestDocStoreWrite_RejectsWhitespaceOnlyTitle(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := NewDocStore(root)
	doc := sampleDoc(t)
	doc.Meta.Title = "   "

	_, err := store.Write(doc)
	if err == nil {
		t.Fatal("expected error for whitespace-only title")
	}
}

// Ensure fmt is used (for TestDocStoreListAll_CombinesAcrossTypes).
var _ = fmt.Sprintf
