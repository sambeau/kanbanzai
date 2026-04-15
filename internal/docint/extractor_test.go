package docint

import (
	"strings"
	"testing"
)

func TestExtractPatterns_EntityRefs(t *testing.T) {
	content := []byte(`# Design Document

This references FEAT-ABC123 and TASK-XYZ in the intro.

## Details

See BUG-001 and DEC-042 for context. Also DOC-ABC.
`)
	sections := ParseStructure(content)
	result := ExtractPatterns(content, sections)

	if len(result.EntityRefs) == 0 {
		t.Fatal("expected entity refs, got none")
	}

	expected := map[string]string{
		"FEAT-ABC123": "feature",
		"TASK-XYZ":    "task",
		"BUG-001":     "bug",
		"DEC-042":     "decision",
		"DOC-ABC":     "document",
	}

	found := map[string]string{}
	for _, ref := range result.EntityRefs {
		found[ref.EntityID] = ref.EntityType
	}

	for id, wantType := range expected {
		gotType, ok := found[id]
		if !ok {
			t.Errorf("missing entity ref %q", id)
			continue
		}
		if gotType != wantType {
			t.Errorf("entity %q: want type %q, got %q", id, wantType, gotType)
		}
	}
}

func TestExtractPatterns_EntityRefs_SectionAttribution(t *testing.T) {
	content := []byte(`# Top

FEAT-001 is here.

## Sub

TASK-002 is here.
`)
	sections := ParseStructure(content)
	result := ExtractPatterns(content, sections)

	refsByID := map[string]EntityRef{}
	for _, ref := range result.EntityRefs {
		refsByID[ref.EntityID] = ref
	}

	feat := refsByID["FEAT-001"]
	if feat.SectionPath != "1" {
		t.Errorf("FEAT-001 section: want %q, got %q", "1", feat.SectionPath)
	}

	task := refsByID["TASK-002"]
	if task.SectionPath != "1.1" {
		t.Errorf("TASK-002 section: want %q, got %q", "1.1", task.SectionPath)
	}
}

func TestExtractPatterns_PlanIDs(t *testing.T) {
	content := []byte(`# Plans

See P1-basic-ui and P2-auth-flow for details.
Also Q3-setup-wizard is relevant.
`)
	sections := ParseStructure(content)
	result := ExtractPatterns(content, sections)

	planIDs := map[string]bool{}
	for _, ref := range result.EntityRefs {
		if ref.EntityType == "plan" {
			planIDs[ref.EntityID] = true
		}
	}

	for _, want := range []string{"P1-basic-ui", "P2-auth-flow", "Q3-setup-wizard"} {
		if !planIDs[want] {
			t.Errorf("missing plan ID %q", want)
		}
	}
}

func TestExtractPatterns_PlanIDs_NotFalsePositive(t *testing.T) {
	// Single letter+digit without slug should not match.
	content := []byte(`# Doc

This has P1 alone and A1 alone which are not plan IDs.
`)
	sections := ParseStructure(content)
	result := ExtractPatterns(content, sections)

	for _, ref := range result.EntityRefs {
		if ref.EntityType == "plan" {
			t.Errorf("unexpected plan ID match: %q", ref.EntityID)
		}
	}
}

func TestExtractPatterns_CrossDocLinks_Markdown(t *testing.T) {
	content := []byte(`# Links

See [the design](work/design/foo.md) for details.
Also [spec](work/spec/phase-1-specification.md#section-3) is relevant.
`)
	sections := ParseStructure(content)
	result := ExtractPatterns(content, sections)

	if len(result.CrossDocLinks) != 2 {
		t.Fatalf("want 2 cross-doc links, got %d", len(result.CrossDocLinks))
	}

	links := map[string]CrossDocLink{}
	for _, l := range result.CrossDocLinks {
		links[l.TargetPath] = l
	}

	design, ok := links["work/design/foo.md"]
	if !ok {
		t.Fatal("missing link to work/design/foo.md")
	}
	if design.LinkText != "the design" {
		t.Errorf("link text: want %q, got %q", "the design", design.LinkText)
	}

	spec, ok := links["work/spec/phase-1-specification.md"]
	if !ok {
		t.Fatal("missing link to work/spec/phase-1-specification.md (anchor should be stripped)")
	}
	if spec.LinkText != "spec" {
		t.Errorf("link text: want %q, got %q", "spec", spec.LinkText)
	}
}

func TestExtractPatterns_CrossDocLinks_Backtick(t *testing.T) {
	content := []byte(`# References

See ` + "`work/design/foo.md`" + ` for the design.
Also ` + "`work/design/bar.md §7, §8`" + ` is relevant.
`)
	sections := ParseStructure(content)
	result := ExtractPatterns(content, sections)

	if len(result.CrossDocLinks) < 2 {
		t.Fatalf("want at least 2 cross-doc links, got %d", len(result.CrossDocLinks))
	}

	targets := map[string]bool{}
	for _, l := range result.CrossDocLinks {
		targets[l.TargetPath] = true
	}

	if !targets["work/design/foo.md"] {
		t.Error("missing backtick link to work/design/foo.md")
	}
	if !targets["work/design/bar.md"] {
		t.Error("missing backtick link to work/design/bar.md (should extract path only)")
	}
}

func TestExtractPatterns_CrossDocLinks_Dedup(t *testing.T) {
	content := []byte(`# Section

See [link1](path/doc.md) and [link2](path/doc.md) — same target twice.
`)
	sections := ParseStructure(content)
	result := ExtractPatterns(content, sections)

	count := 0
	for _, l := range result.CrossDocLinks {
		if l.TargetPath == "path/doc.md" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("want 1 deduped link to path/doc.md, got %d", count)
	}
}

func TestExtractPatterns_ConventionalRoles(t *testing.T) {
	content := []byte(`# Document

## Decisions

Some decisions here.

## Open Questions

What about this?

## Requirements

Must do X.

## Narrative Section

Just some text.
`)
	sections := ParseStructure(content)
	result := ExtractPatterns(content, sections)

	rolesBySection := map[string]string{}
	for _, r := range result.ConventionalRoles {
		rolesBySection[r.SectionPath] = r.Role
	}

	tests := []struct {
		sectionTitle string
		wantRole     string
	}{
		{"Decisions", "decision"},
		{"Open Questions", "question"},
		{"Requirements", "requirement"},
	}

	for _, tt := range tests {
		found := false
		for _, r := range result.ConventionalRoles {
			// Match by role since section paths are numeric.
			if r.Role == tt.wantRole {
				found = true
				if r.Confidence != "high" {
					t.Errorf("role %q: want confidence %q, got %q", tt.wantRole, "high", r.Confidence)
				}
				break
			}
		}
		if !found {
			t.Errorf("missing conventional role %q for heading %q", tt.wantRole, tt.sectionTitle)
		}
	}
}

func TestExtractPatterns_ConventionalRoles_Nested(t *testing.T) {
	content := []byte(`# Document

## Analysis

### Risks

Some risks here.

### Assumptions

Some assumptions.
`)
	sections := ParseStructure(content)
	result := ExtractPatterns(content, sections)

	roles := map[string]bool{}
	for _, r := range result.ConventionalRoles {
		roles[r.Role] = true
	}

	if !roles["risk"] {
		t.Error("missing role 'risk' for nested heading 'Risks'")
	}
	if !roles["assumption"] {
		t.Error("missing role 'assumption' for nested heading 'Assumptions'")
	}
}

func TestExtractPatterns_FrontMatter(t *testing.T) {
	content := []byte(`# Test Document

- Status: draft design
- Date: 2026-07-18
- Purpose: define something important
- Related:
  - ` + "`work/design/foo.md`" + `
  - ` + "`work/design/bar.md`" + `

---

## Body

Content here.
`)
	sections := ParseStructure(content)
	result := ExtractPatterns(content, sections)

	if result.FrontMatter == nil {
		t.Fatal("expected front matter, got nil")
	}

	fm := result.FrontMatter
	if fm.Status != "draft design" {
		t.Errorf("Status: want %q, got %q", "draft design", fm.Status)
	}
	if fm.Date != "2026-07-18" {
		t.Errorf("Date: want %q, got %q", "2026-07-18", fm.Date)
	}
	if fm.Extra == nil || fm.Extra["Purpose"] != "define something important" {
		t.Errorf("Purpose: want %q in Extra, got %v", "define something important", fm.Extra)
	}
	if len(fm.Related) != 2 {
		t.Fatalf("Related: want 2 entries, got %d", len(fm.Related))
	}
	if fm.Related[0] != "work/design/foo.md" {
		t.Errorf("Related[0]: want %q, got %q", "work/design/foo.md", fm.Related[0])
	}
	if fm.Related[1] != "work/design/bar.md" {
		t.Errorf("Related[1]: want %q, got %q", "work/design/bar.md", fm.Related[1])
	}
}

func TestExtractPatterns_FrontMatter_WithBasis(t *testing.T) {
	content := []byte(`# Design Doc

- Status: design basis
- Date: 2026-03-18
- Basis:
  - ` + "`workflow-system-design.md`" + `
  - ` + "`initial-analysis.md`" + `

---

## Content
`)
	sections := ParseStructure(content)
	result := ExtractPatterns(content, sections)

	if result.FrontMatter == nil {
		t.Fatal("expected front matter, got nil")
	}

	fm := result.FrontMatter
	if fm.Status != "design basis" {
		t.Errorf("Status: want %q, got %q", "design basis", fm.Status)
	}
	// Basis is a list field stored in Extra as joined string.
	if fm.Extra == nil {
		t.Fatal("expected Extra map for Basis field")
	}
	basis, ok := fm.Extra["Basis"]
	if !ok {
		t.Fatal("missing Basis in Extra")
	}
	if !strings.Contains(basis, "workflow-system-design.md") {
		t.Errorf("Basis should contain %q, got %q", "workflow-system-design.md", basis)
	}
}

func TestExtractPatterns_NoFrontMatter(t *testing.T) {
	content := []byte(`# Simple Document

This document has no front matter, just a heading and body text.

## Section

More text.
`)
	sections := ParseStructure(content)
	result := ExtractPatterns(content, sections)

	if result.FrontMatter != nil {
		t.Errorf("expected nil front matter, got %+v", result.FrontMatter)
	}
}

func TestExtractPatterns_NoFrontMatter_NoHeading(t *testing.T) {
	content := []byte(`Just some text without any heading at all.`)
	sections := ParseStructure(content)
	result := ExtractPatterns(content, sections)

	if result.FrontMatter != nil {
		t.Errorf("expected nil front matter for headingless doc, got %+v", result.FrontMatter)
	}
}

func TestExtractPatterns_EmptySections(t *testing.T) {
	content := []byte(`Some text with FEAT-001 and TASK-002 but no headings at all.`)
	sections := ParseStructure(content)

	if len(sections) != 0 {
		t.Fatal("expected no sections from headingless content")
	}

	result := ExtractPatterns(content, sections)

	// Should still find entity refs even without sections.
	if len(result.EntityRefs) == 0 {
		t.Error("expected entity refs even with empty sections")
	}

	ids := map[string]bool{}
	for _, ref := range result.EntityRefs {
		ids[ref.EntityID] = true
		// Section path should be empty since there are no sections.
		if ref.SectionPath != "" {
			t.Errorf("entity %q: want empty section path, got %q", ref.EntityID, ref.SectionPath)
		}
	}
	if !ids["FEAT-001"] {
		t.Error("missing FEAT-001")
	}
	if !ids["TASK-002"] {
		t.Error("missing TASK-002")
	}

	// No conventional roles from empty sections.
	if len(result.ConventionalRoles) != 0 {
		t.Errorf("expected no conventional roles, got %d", len(result.ConventionalRoles))
	}
}

func TestExtractPatterns_Deduplication(t *testing.T) {
	content := []byte(`# Section

FEAT-001 is mentioned here. And FEAT-001 again in the same section.
Also FEAT-001 one more time.
`)
	sections := ParseStructure(content)
	result := ExtractPatterns(content, sections)

	count := 0
	for _, ref := range result.EntityRefs {
		if ref.EntityID == "FEAT-001" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("want 1 deduped FEAT-001 ref, got %d", count)
	}
}

func TestExtractPatterns_Deduplication_DifferentSections(t *testing.T) {
	content := []byte(`# Section One

FEAT-001 in section one.

## Section Two

FEAT-001 in section two.
`)
	sections := ParseStructure(content)
	result := ExtractPatterns(content, sections)

	count := 0
	for _, ref := range result.EntityRefs {
		if ref.EntityID == "FEAT-001" {
			count++
		}
	}
	// Same entity in different sections should appear twice.
	if count != 2 {
		t.Errorf("want 2 FEAT-001 refs (different sections), got %d", count)
	}
}

func TestExtractPatterns_EntityRefsInCodeBlock(t *testing.T) {
	// Entity refs inside fenced code blocks are still found by regex.
	// Layer 2 is pattern-based — it doesn't skip code blocks.
	// This test documents the current behavior.
	content := []byte(`# Doc

` + "```" + `
FEAT-999 in a code block
` + "```" + `
`)
	sections := ParseStructure(content)
	result := ExtractPatterns(content, sections)

	found := false
	for _, ref := range result.EntityRefs {
		if ref.EntityID == "FEAT-999" {
			found = true
		}
	}
	// Pattern extraction finds refs everywhere — this is expected Layer 2 behavior.
	if !found {
		t.Error("expected FEAT-999 to be found (Layer 2 does not skip code blocks)")
	}
}

func TestExtractPatterns_FullIntegration(t *testing.T) {
	content := []byte(`# Implementation Plan

- Status: active
- Date: 2026-08-01

---

## Overview

This plan covers FEAT-100 and FEAT-200.
See [design doc](work/design/impl.md) for background.

## Tasks

### Authentication

TASK-101 implements P1-auth-flow.
See also ` + "`work/spec/auth.md §3`" + `.

### Open Questions

What about TASK-102?

## Risks

DEC-050 might affect timeline.
`)
	sections := ParseStructure(content)
	result := ExtractPatterns(content, sections)

	// Front matter
	if result.FrontMatter == nil {
		t.Fatal("expected front matter")
	}
	if result.FrontMatter.Status != "active" {
		t.Errorf("Status: want %q, got %q", "active", result.FrontMatter.Status)
	}

	// Entity refs
	entityIDs := map[string]string{}
	for _, ref := range result.EntityRefs {
		entityIDs[ref.EntityID] = ref.EntityType
	}
	for _, want := range []struct {
		id  string
		typ string
	}{
		{"FEAT-100", "feature"},
		{"FEAT-200", "feature"},
		{"TASK-101", "task"},
		{"TASK-102", "task"},
		{"DEC-050", "decision"},
		{"P1-auth-flow", "plan"},
	} {
		got, ok := entityIDs[want.id]
		if !ok {
			t.Errorf("missing entity ref %q", want.id)
		} else if got != want.typ {
			t.Errorf("entity %q: want type %q, got %q", want.id, want.typ, got)
		}
	}

	// Cross-doc links
	targets := map[string]bool{}
	for _, l := range result.CrossDocLinks {
		targets[l.TargetPath] = true
	}
	if !targets["work/design/impl.md"] {
		t.Error("missing cross-doc link to work/design/impl.md")
	}
	if !targets["work/spec/auth.md"] {
		t.Error("missing cross-doc link to work/spec/auth.md")
	}

	// Conventional roles
	roles := map[string]bool{}
	for _, r := range result.ConventionalRoles {
		roles[r.Role] = true
	}
	if !roles["question"] {
		t.Error("missing conventional role 'question' for 'Open Questions'")
	}
	if !roles["risk"] {
		t.Error("missing conventional role 'risk' for 'Risks'")
	}
}
