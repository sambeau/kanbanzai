# Design: AC Pattern Recognition and Decompose Hardening

| Field | Value |
|-------|-------|
| Feature | FEAT-01KPPG1MF4DAT |
| Plan | P24-retro-recommendations |
| Status | Draft |

---

## Overview

Two related defects share a common root: the `**AC-NN.**` bold-identifier
format used in every Kanbanzai specification document is unrecognised at two
separate layers of the system.

**Layer A — `internal/docint` (document intelligence index).** The
`extractConventionalRoles` function in `extractor.go` classifies sections by
*heading keywords only*. A section whose heading is non-standard (e.g.
`"Background"`, `"Notes"`, or any heading that does not contain the words
`"acceptance"`, `"criteria"`, or `"requirements"`) is never assigned a
`requirement` conventional role, even when its body contains dozens of
`**AC-NN.**` lines. Downstream consumers — `doc_intel` queries, the
`asmExtractCriteria` path used by `next`/`handoff` — therefore never see those
sections as requirement-bearing.

**Layer B — `decompose propose` / `parseSpecStructure`.** The
`parseSpecStructure` function in `service/decompose.go` handles the bare
`**AC-NN.** text` form correctly *inside* sections whose title passes
`isAcceptanceCriteriaSection`. However the *list-item* variant `- **AC-NN.**
text` (a bullet-list line with the bold identifier but no checkbox) is never
matched: `reBoldIdent` requires the line to begin with `**`, so the `- ` prefix
causes the match to fail. The criterion then falls through to the generic
list-item path, which either adds the raw text with markdown bold markers
intact, or drops it entirely.

When `parseSpecStructure` returns zero criteria, `DecomposeFeature` returns an
error — but the error message names only the format variants it already
supports, giving the engineer no clue that a subtly different bullet form is the
cause.

This document describes both fixes and the verify-then-fix test strategy for
each.

---

## Goals and Non-Goals

**Goals**

1. Teach `internal/docint/extractor.go` to detect sections whose *content*
   contains `**AC-NN.**` lines and assign the `requirement` conventional role to
   those sections (Layer 2, pattern-based, deterministic).
2. Fix `parseSpecStructure` in `service/decompose.go` to recognise the list-item
   bold-identifier form `- **AC-NN.** text` in addition to the bare form.
3. Fix `asmExtractCriteria` in `internal/mcp/assembly.go` so that the same
   list-item form is normalised to `XX-NN: text` rather than kept with raw bold
   markers.
4. Replace the generic zero-criteria error in `DecomposeFeature` with a
   diagnostic that reports which sections were parsed, which formats were seen,
   and why criteria extraction still failed.

**Non-Goals**

- Support for non-standard bold-identifier formats (`*AC-01.*`,
  `__AC-01.__`, `AC-01:` without bold, etc.).
- Validation that identifier numbers are sequential or non-duplicate.
- Changes to the document record store, index files, or persistence layer.
- Changes to `decompose review` or `decompose apply`.
- LLM-based (Layer 3) reclassification of AC sections.

---

## Design

### 1. AC Pattern Recognition in docint (Layer 2)

#### Current behaviour

`extractConventionalRoles` in `extractor.go` has the signature:

```kanbanzai/internal/docint/extractor.go#L204-217
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
```

It only examines `s.Title`. Section body content is never consulted.
`ExtractPatterns` already receives `content []byte` but does not pass it to
this function.

#### Change: extend signature and add content scan

Change the call site in `ExtractPatterns`:

```kanbanzai/internal/docint/extractor.go#L50-52
// Before:
result.ConventionalRoles = extractConventionalRoles(sections)
// After:
result.ConventionalRoles = extractConventionalRoles(content, sections)
```

Update the function signature and body. The new logic runs *after* the
heading-based walk. For every section that received no heading-based role, scan
the section's content slice (`content[s.ByteOffset : s.ByteOffset+s.ByteCount]`)
with a new package-level regex:

```kanbanzai/internal/docint/extractor.go#L1-1
// acBoldIdentLineRe matches a line that begins (optionally after a list marker)
// with a bold-identifier token: **XX-NN.**
// It handles both bare form ("**AC-01.** text") and list-item form
// ("- **AC-01.** text") since either may appear in specification documents.
var acBoldIdentLineRe = regexp.MustCompile(`(?m)^(?:-\s+)?\*\*[A-Z]+-\d+\.\*\*\s+`)
```

If at least one line in the section body matches `acBoldIdentLineRe`, the
section is added to `roles` with:

```kanbanzai/internal/docint/extractor.go#L1-1
ConventionalRole{
    SectionPath: s.Path,
    Role:        string(RoleRequirement), // "requirement"
    Confidence:  "medium",               // content-based, not heading-based
}
```

Confidence is `"medium"` rather than `"high"` because the inference is
heuristic: a section heading match is a direct keyword hit, whereas a content
scan is circumstantial. Downstream consumers that filter on confidence can
treat the two differently.

Sections that already received a heading-based role are skipped by the content
scan to avoid duplicate entries.

#### What `ConventionalRole` value to assign

The role value is `"requirement"` (the existing `RoleRequirement` constant),
*not* a new `"acceptance-criteria"` string.

Rationale:

- The taxonomy already maps the heading `"Acceptance Criteria"` to
  `RoleRequirement`. Adding `"acceptance-criteria"` as a distinct constant
  would require updates to `taxonomy.go` (`AllRoles`, `ValidRole`), to the
  Layer 3 classification validator, and to every consumer that inspects roles —
  a cascading change with no functional benefit at this stage.
- AC items *are* requirements: they state testable conditions the implementation
  must satisfy. Using `RoleRequirement` is semantically correct.
- `asmExtractCriteria` already gates on title keywords (`"acceptance"`,
  `"criteria"`, `"requirement"`, `"constraint"`), not on the `ConventionalRole`
  field, so this change does not directly affect its extraction logic — but it
  does improve `doc_intel` queries and any future consumers that read
  `ConventionalRoles` from the index.

### 2. Hardened decompose propose behaviour

#### Where the current fallback lives and what it does

The old section-header fallback (which generated tasks named "Implement 1.
Purpose", "Implement 2. Goals", etc.) was removed as part of
FEAT-01KMT-58TV8V9C (`decompose-precondition-gates`). In the current codebase
`DecomposeFeature` in `service/decompose.go` already gates on zero criteria:

```kanbanzai/internal/service/decompose.go#L141-146
spec := parseSpecStructure(content)

// 6. Gate: spec must contain parseable acceptance criteria.
if len(spec.acceptanceCriteria) == 0 {
    return DecomposeResult{}, fmt.Errorf(
        "no acceptance criteria found in spec %q — ensure the spec "+
        "uses checkbox items (- [ ] ...), numbered items, or bold-identifier "+
        "lines (**AC-NN.** text) within an Acceptance Criteria section", specDocID)
}
```

The gate is correct but the error is generic. It does not tell the engineer
what was actually found or why the bold-identifier variant failed. More
critically, the *root cause* of zero criteria on well-formed specs is that
`parseSpecStructure` does not handle `- **AC-NN.** text`.

#### Fix A: handle `- **AC-NN.** text` in `parseSpecStructure`

In `parseSpecStructure`, the `reBoldIdent` match is applied to `trimmed` (the
raw line). For a line like `- **AC-01.** The system must do X.`, `trimmed`
begins with `- ` and the regex `^\*\*([A-Z]+)-(\d+)\.\*\*\s+(.+)$` does not
match. The existing numbered-list and table checks also do not match.

The fix: within the `inACSection` branch, before (or alongside) the existing
`reBoldIdent` check, strip a leading `- ` (or `* `/`+ `) and re-apply the
regex:

```kanbanzai/internal/service/decompose.go#L368-384
if inACSection {
    // Try bare bold-identifier form first: **XX-NN.** text
    candidate := trimmed
    // Also handle list-item bold-identifier: - **XX-NN.** text
    for _, marker := range []string{"- ", "* ", "+ "} {
        if strings.HasPrefix(candidate, marker) {
            candidate = strings.TrimSpace(candidate[len(marker):])
            break
        }
    }
    if m := reBoldIdent.FindStringSubmatch(candidate); m != nil {
        criterion := m[1] + "-" + m[2] + ": " + m[3]
        spec.acceptanceCriteria = append(spec.acceptanceCriteria, acceptanceCriterion{
            text:     criterion,
            section:  currentSection,
            parentL2: currentL2,
        })
        continue
    }
    // ... existing numbered-list and table handling unchanged ...
}
```

#### Fix B: same normalisation in `asmExtractCriteria`

`asmExtractCriteria` in `internal/mcp/assembly.go` currently checks
`boldIdentifierRe` against `trimmed` before stripping list markers. For
`- **AC-01.** text`, the bold-ident check fails; the line is then stripped to
`**AC-01.** text` by the generic list-marker loop, and if `isAcceptanceSection`
the raw bold-marked text is added as a criterion — wrong format.

The fix: after stripping the list marker, re-apply `boldIdentifierRe` to the
stripped text:

```kanbanzai/internal/mcp/assembly.go#L477-515
// Check bare bold-identifier form first.
if m := boldIdentifierRe.FindStringSubmatch(trimmed); m != nil {
    // ... existing logic ...
    continue
}

// Strip list marker and retry bold-identifier check.
bare := trimmed
for _, marker := range []string{"- ", "* ", "+ ", "• "} {
    if strings.HasPrefix(bare, marker) {
        bare = strings.TrimSpace(bare[len(marker):])
        break
    }
}
if bare != trimmed {
    if m := boldIdentifierRe.FindStringSubmatch(bare); m != nil {
        prefix, num, criterionText := m[1], m[2], m[3]
        criterion := prefix + "-" + num + ": " + criterionText
        if isAcceptanceSection || hasRFC2119Keyword(criterionText) {
            addCriterion(criterion)
        }
        continue
    }
}
// ... rest of existing list-item logic ...
```

#### Fix C: richer diagnostic in `DecomposeFeature`

Replace the generic zero-criteria error with a diagnostic that shows:

1. How many sections were parsed from the spec.
2. The titles of those sections and which ones were recognised as acceptance
   criteria / requirement sections.
3. Whether any `**XX-NN.**` bold-identifier lines were found *outside*
   recognised AC sections (indicating a section-naming mismatch).
4. Concrete remediation: rename the section heading, or switch to the
   `- [ ] text` checkbox form.

This requires adding a pass over `spec.sections` (already populated by
`parseSpecStructure`) in the zero-criteria branch to build the diagnostic
string. No new dependencies or service calls are needed — all data is already
present in the `specStructure` returned by `parseSpecStructure`.

Example improved error:

```kanbanzai/internal/service/decompose.go#L141-146
// After fix:
if len(spec.acceptanceCriteria) == 0 {
    return DecomposeResult{}, buildZeroCriteriaDiagnostic(specDocID, content, spec)
}
```

`buildZeroCriteriaDiagnostic` is a new unexported helper that scans `content`
for `**[A-Z]+-\d+\.\*\*` occurrences outside AC sections and formats a
message such as:

```kanbanzai/internal/service/decompose.go#L1-1
no acceptance criteria found in spec "DOC-...".
  Sections parsed (5): "1. Purpose", "2. Scope", "3. Requirements",
    "4. Implementation Notes", "5. Acceptance Criteria"
  Recognised AC sections: "5. Acceptance Criteria"
  Bold-identifier lines found outside AC sections: 0
  Bold-identifier lines found inside AC sections: 0
  Suggestion: the spec may use "- **AC-NN.** text" without a checkbox; upgrade
  to "- [ ] **AC-NN.** text" or ensure bold-identifier lines appear directly
  (without a leading "- ") in the Acceptance Criteria section.
```

After Fix A and Fix B land, the `- **AC-NN.** text` form will be correctly
parsed and this diagnostic will rarely be reached. It remains as a safety net
for genuinely malformed specs.

### 3. Verify-then-fix approach

Both fixes must be validated with tests that are written *before* the fix is
applied (red), then pass after (green).

#### Test 1 — `internal/docint`: content-based AC detection

File: `internal/docint/extractor_test.go`

```kanbanzai/internal/docint/extractor_test.go#L1-1
// TestExtractConventionalRoles_ACContentInNonACSection verifies that a section
// whose heading is not an AC keyword but whose body contains bold-identifier
// lines is assigned a "requirement" conventional role with confidence "medium".
//
// Before the fix: extractConventionalRoles receives only sections (no content);
// a section titled "Notes" with **AC-01.** lines in its body returns no roles.
// This test FAILS (zero roles) before the fix and PASSES (one role) after.
func TestExtractConventionalRoles_ACContentInNonACSection(t *testing.T) {
    content := []byte(
        "# Spec\n\n" +
        "## Notes\n\n" +
        "**AC-01.** The handler must reject empty inputs.\n\n" +
        "**AC-02.** The handler must return 400 on malformed JSON.\n",
    )
    sections := []Section{
        {Title: "Spec",  Path: "0", Level: 1, ByteOffset: 0,  ByteCount: len(content)},
        {Title: "Notes", Path: "1", Level: 2, ByteOffset: 9,  ByteCount: len(content) - 9},
    }
    roles := extractConventionalRoles(content, sections)
    var noteRole *ConventionalRole
    for i := range roles {
        if roles[i].SectionPath == "1" {
            noteRole = &roles[i]
        }
    }
    if noteRole == nil {
        t.Fatal("no ConventionalRole assigned to Notes section; expected requirement/medium")
    }
    if noteRole.Role != "requirement" {
        t.Errorf("Role = %q, want %q", noteRole.Role, "requirement")
    }
    if noteRole.Confidence != "medium" {
        t.Errorf("Confidence = %q, want %q", noteRole.Confidence, "medium")
    }
}
```

#### Test 2 — `service/decompose`: list-item bold-identifier extraction

File: `internal/service/decompose_test.go` (or wherever `parseSpecStructure`
is unit-tested)

```kanbanzai/internal/service/decompose_test.go#L1-1
// TestParseSpecStructure_ListItemBoldIdent verifies that "- **AC-NN.** text"
// (bold-identifier inside a bullet list item, no checkbox) is extracted as a
// criterion when inside an Acceptance Criteria section.
//
// Before the fix: reBoldIdent requires the line to start with "**"; a line
// starting with "- " never matches, so spec.acceptanceCriteria is empty.
// This test FAILS (0 criteria) before the fix and PASSES (2 criteria) after.
func TestParseSpecStructure_ListItemBoldIdent(t *testing.T) {
    content := `# Spec

## Acceptance Criteria

- **AC-01.** The system must reject requests without authentication tokens.
- **AC-02.** The system must return HTTP 401 in that case.
`
    spec := parseSpecStructure(content)
    if len(spec.acceptanceCriteria) != 2 {
        t.Fatalf("acceptanceCriteria len = %d, want 2; got: %v",
            len(spec.acceptanceCriteria), spec.acceptanceCriteria)
    }
    if spec.acceptanceCriteria[0].text != "AC-01: The system must reject requests without authentication tokens." {
        t.Errorf("criteria[0] = %q", spec.acceptanceCriteria[0].text)
    }
}
```

#### Test 3 — `assembly.go`: list-item form normalised without bold markers

File: `internal/mcp/assembly_test.go`

```kanbanzai/internal/mcp/assembly_test.go#L1-1
// TestAsmExtractCriteria_ListItemBoldIdent_Normalised verifies that
// "- **AC-NN.** text" inside an acceptance criteria section is extracted with
// the canonical "AC-NN: text" format, not with raw bold markers.
//
// Before the fix: the line is stripped to "**AC-01.** text" by the generic
// list-marker path and added as-is, producing "**AC-01.** text" in the output.
// After the fix: the bold-identifier re-check produces "AC-01: text".
func TestAsmExtractCriteria_ListItemBoldIdent_Normalised(t *testing.T) {
    t.Parallel()
    sections := []asmSpecSection{{
        document: "spec.md",
        section:  "Acceptance Criteria",
        content:  "- **AC-01.** The system must validate inputs.\n- **AC-02.** The system must log failures.",
    }}
    got := asmExtractCriteria(sections)
    if len(got) != 2 {
        t.Fatalf("len(criteria) = %d, want 2; got: %v", len(got), got)
    }
    want0 := "AC-01: The system must validate inputs."
    want1 := "AC-02: The system must log failures."
    if got[0] != want0 {
        t.Errorf("criteria[0] = %q, want %q", got[0], want0)
    }
    if got[1] != want1 {
        t.Errorf("criteria[1] = %q, want %q", got[1], want1)
    }
}
```

---

## Alternatives Considered

### Layer 3 (LLM classification) instead of Layer 2

Layer 3 classification is invoked explicitly via `doc(action: "classify")` and
stores results in the document index. It is the right tool for ambiguous or
context-dependent judgements (e.g. "is this prose section a rationale or a
narrative?").

`**AC-NN.**` recognition is a mechanical regex match on a machine-readable
token. Using LLM classification here would:

- Add latency on every reclassification cycle.
- Require the document to be explicitly re-classified after any content change.
- Introduce non-determinism (the model may occasionally miss or misclassify).
- Provide no improvement in accuracy over a simple regex.

**Decision: Layer 2.** The pattern is deterministic, syntactically distinct,
and already used in the same way by `boldIdentifierRe` in `assembly.go` and
`reBoldIdent` in `service/decompose.go`. Layer 2 is the correct home.

### Introduce `"acceptance-criteria"` as a new `FragmentRole`

Adding a dedicated `RoleAcceptanceCriteria = "acceptance-criteria"` constant
would allow consumers to distinguish between general requirements (prose `MUST`
statements) and structured acceptance criteria (bold-identifier lists). This
distinction could be useful in future tooling (e.g. generating test stubs
directly from ACs).

However, it requires:

- Adding the constant to `taxonomy.go`, `AllRoles()`, and `ValidRole()`.
- Updating the Layer 3 classification validator and any migration for existing
  index files.
- Updating `asmExtractCriteria` (which currently groups all three — acceptance,
  requirements, constraints — into one `isAcceptanceSection` flag) to handle
  the new role name.

The net value is low until a consumer actually needs the distinction. Using
`RoleRequirement` with `Confidence: "medium"` preserves the option to migrate
later without blocking this fix.

**Decision: use `"requirement"` (`RoleRequirement`) with `"medium"` confidence.**

### Fix only `asmExtractCriteria` (skip the docint layer)

The previous related feature (FEAT-01KN4ZPCMJ1FP, `docint-ac-pattern-recognition`)
scoped its fix to `asmExtractCriteria` and explicitly excluded `internal/docint`
taxonomy changes. That spec addressed the `**AC-NN.** text` bare form only in the
assembly path.

Limiting FEAT-01KPPG1MF4DAT to the same scope would leave `doc_intel` queries
unable to find AC sections by role, and would not help the `parseSpecStructure`
path used by `decompose propose` directly.

**Decision: fix all three sites** — `extractor.go` (docint Layer 2),
`parseSpecStructure` (decompose service), and `asmExtractCriteria` (assembly).
The three fixes are independent and can be implemented in a single PR.

---

## Dependencies

| ID | Dependency | Notes |
|----|-----------|-------|
| DEP-01 | `FEAT-01KN4ZPCMJ1FP` (docint-ac-pattern-recognition) must be merged | Established `boldIdentifierRe` in `assembly.go`; this feature builds on that |
| DEP-02 | `FEAT-01KMT-58TV8V9C` (decompose-precondition-gates) must be merged | Removed the section-header fallback and added the zero-criteria gate; this feature improves that gate's diagnostic |
| DEP-03 | `internal/docint` `Section.ByteOffset` and `Section.ByteCount` must be populated by Layer 1 parsing | Required for the content-scan approach in `extractConventionalRoles`; confirm with `TestSplitSections` tests that these fields are non-zero |
```

Now let me register the document and commit: