# Specification: AC Pattern Recognition and Decompose Hardening

| Field   | Value                                          |
|---------|------------------------------------------------|
| Status  | Draft                                          |
| Feature | `FEAT-01KPPG1MF4DAT`                           |
| Design  | `work/design/p24-ac-pattern-and-decompose.md`  |
| Plan    | P24 — Retro Recommendations                    |

---

## Overview

This specification implements the design described in
`work/design/p24-ac-pattern-and-decompose.md`.

The `**AC-NN.**` bold-identifier format used in Kanbanzai specification documents is
unrecognised at two independent layers of the system, causing `decompose propose` to fail
on well-formed specifications and `doc_intel` queries to miss AC sections with
non-standard headings.

**Layer A — `internal/docint/extractor.go`.** `extractConventionalRoles` classifies
sections by heading keywords only. A section whose heading does not contain `"acceptance"`,
`"criteria"`, or `"requirements"` is never assigned the `requirement` conventional role,
even when its body contains bold-identifier lines such as `**AC-01.** text`.

**Layer B — `service/decompose.go` and `internal/mcp/assembly.go`.** Both
`parseSpecStructure` and `asmExtractCriteria` fail to match the list-item variant
`- **AC-NN.** text` because their regexes require the line to begin with `**`. The
leading `- ` prefix causes the match to fail. When `parseSpecStructure` returns zero
criteria, `DecomposeFeature` emits a generic error that does not identify the actual
cause.

This specification covers three fixes (one per defect site) and a diagnostic improvement,
each validated by a verify-then-fix test.

---

## Scope

**In scope:**

- Extending `extractConventionalRoles` in `internal/docint/extractor.go` to accept
  document content and perform content-based AC detection (Layer 2, deterministic regex).
- A new package-level regex `acBoldIdentLineRe` in `internal/docint/extractor.go` that
  matches both the bare form (`**AC-01.** text`) and the list-item form
  (`- **AC-01.** text`).
- Fixing `parseSpecStructure` in `service/decompose.go` to recognise the list-item
  bold-identifier form and extract it as a canonical `XX-NN: text` criterion.
- Fixing `asmExtractCriteria` in `internal/mcp/assembly.go` to normalise the list-item
  bold-identifier form to `XX-NN: text` rather than preserving raw bold markers.
- Replacing the generic zero-criteria error in `DecomposeFeature` with a structured
  diagnostic via a new unexported helper `buildZeroCriteriaDiagnostic`.
- Three verify-then-fix tests (failing before fix, passing after):
  `TestExtractConventionalRoles_ACContentInNonACSection`,
  `TestParseSpecStructure_ListItemBoldIdent`, and
  `TestAsmExtractCriteria_ListItemBoldIdent_Normalised`.

**Out of scope:**

- Non-standard bold-identifier variants (`*AC-01.*`, `__AC-01.__`, `AC-01:` without
  bold markers).
- Validation that identifier numbers are sequential or non-duplicate.
- Changes to the document record store, index files, or persistence layer.
- Changes to `decompose review` or `decompose apply`.
- Introduction of a new `"acceptance-criteria"` `FragmentRole` constant or taxonomy
  changes to `taxonomy.go`, `AllRoles()`, or `ValidRole()`.
- Layer 3 (LLM-based) reclassification of AC sections.

---

## Functional Requirements

**FR-001:** `extractConventionalRoles` in `internal/docint/extractor.go` MUST accept
`content []byte` as its first parameter. The call site in `ExtractPatterns` MUST pass
the document content as this argument.

**FR-002:** A package-level compiled regex `acBoldIdentLineRe` MUST be defined in
`internal/docint/extractor.go`. It MUST match any line that begins — optionally after a
list marker (`- `, `* `, or `+ `) — with a bold-identifier token of the form `**XX-NN.**`
(where `XX` is one or more uppercase ASCII letters and `NN` is one or more ASCII digits),
followed by at least one whitespace character. Non-standard variants using single
asterisks or underscores MUST NOT be matched.

**FR-003:** After the heading-based walk in `extractConventionalRoles`, every section
that was NOT assigned a role by the heading walk MUST be scanned for bold-identifier
lines. The scan MUST use `acBoldIdentLineRe` against the byte slice
`content[s.ByteOffset : s.ByteOffset+s.ByteCount]` for each such section.

**FR-004:** If at least one line in a section's byte range matches `acBoldIdentLineRe`,
that section MUST be added to the result as `ConventionalRole{SectionPath: s.Path,
Role: "requirement", Confidence: "medium"}`.

**FR-005:** Sections that were already assigned a role by the heading-based walk MUST NOT
be re-processed by the content scan. Each section path MUST appear at most once in the
returned slice.

**FR-006:** In `parseSpecStructure` (`service/decompose.go`), within the `inACSection`
branch, before applying `reBoldIdent` to a trimmed line, the implementation MUST strip a
leading list marker (`- `, `* `, or `+ `) from the line and re-attempt the regex match
on the stripped candidate.

**FR-007:** When the stripped candidate in FR-006 matches `reBoldIdent`, the
implementation MUST append a criterion to `spec.acceptanceCriteria` with text formatted
as `XX-NN: <criterion text>` (bold markers stripped, trailing period inside the bold span
stripped, colon and space inserted before the criterion text).

**FR-008:** In `asmExtractCriteria` (`internal/mcp/assembly.go`), after the bare
bold-identifier check against `trimmed` fails, the implementation MUST strip a list
marker (`- `, `* `, `+ `, or `•`) from `trimmed` and re-apply `boldIdentifierRe` to the
stripped text.

**FR-009:** When the re-applied `boldIdentifierRe` in FR-008 matches, the implementation
MUST emit the criterion in canonical `XX-NN: <criterion text>` format. The output MUST
NOT contain raw bold markers (`**`). The context-sensitive extraction rule from
`FEAT-01KN4ZPCMJ1FP` applies: in a recognised acceptance/requirement/constraint section
the criterion is always emitted; outside such a section it is emitted only if the
criterion text contains an RFC 2119 keyword.

**FR-010:** In `DecomposeFeature` (`service/decompose.go`), the zero-criteria error MUST
be replaced by a call to a new unexported helper with signature
`buildZeroCriteriaDiagnostic(specDocID string, content []byte, spec specStructure) error`.

**FR-011:** `buildZeroCriteriaDiagnostic` MUST produce an error message that contains
all four of the following elements:
- (a) The count of sections parsed from the spec and their titles.
- (b) The titles of sections that were recognised as acceptance-criteria or requirement
  sections.
- (c) The count of bold-identifier lines found inside recognised AC sections and the
  count found outside them.
- (d) A concrete remediation suggestion (e.g., rename the section heading to include
  "Acceptance Criteria", or switch to the `- [ ] text` checkbox form).

**FR-012:** Three verify-then-fix tests MUST be written such that each test fails before
its corresponding production fix is applied and passes after:
- `TestExtractConventionalRoles_ACContentInNonACSection` in
  `internal/docint/extractor_test.go`.
- `TestParseSpecStructure_ListItemBoldIdent` in
  `internal/service/decompose_test.go`.
- `TestAsmExtractCriteria_ListItemBoldIdent_Normalised` in
  `internal/mcp/assembly_test.go`.

---

## Non-Functional Requirements

**NFR-001:** The content-based AC detection introduced by FR-002 through FR-004 MUST be
deterministic. It MUST NOT invoke any LLM, external service, or non-deterministic
component. The fix resides entirely in Layer 2 (pattern-based).

**NFR-002:** `Section.ByteOffset` and `Section.ByteCount` MUST be used to bound the
content slice passed to `acBoldIdentLineRe` (FR-003). The scan MUST NOT pass the full
document content for each section.

**NFR-003:** All existing tests in `internal/docint/`, `internal/service/`, and
`internal/mcp/` MUST continue to pass without modification after each fix is applied.

**NFR-004:** The heading-based role assignment MUST retain `Confidence: "high"`. Only
the content-based detection path introduced by this specification uses
`Confidence: "medium"`.

---

## Acceptance Criteria

**AC-001.** Given a document with a section titled "Notes" whose body contains
`**AC-01.**` bold-identifier lines and whose heading does not match any existing heading
keyword, when `extractConventionalRoles(content, sections)` is called, then the "Notes"
section appears in the returned slice with `Role: "requirement"` and
`Confidence: "medium"`. (FR-003, FR-004)

**AC-002.** Given a document with an "Acceptance Criteria" section already assigned
`Confidence: "high"` by the heading walk, and whose body also contains bold-identifier
lines, when `extractConventionalRoles` is called, then the returned slice contains
exactly one `ConventionalRole` entry for that section — no duplicate from the content
scan. (FR-005)

**AC-003.** Given that `ExtractPatterns` is called with document content containing
`- **AC-01.** text` in a non-standard-headed section, then `acBoldIdentLineRe` matches
that line and the section is assigned a `ConventionalRole` entry with
`Role: "requirement"` and `Confidence: "medium"`. A line of the form `*AC-01.* text`
(single asterisks) MUST NOT produce a match. (FR-001, FR-002)

**AC-004.** Given a spec document with an "Acceptance Criteria" section containing
`- **AC-01.** The system must reject requests without authentication tokens.` and
`- **AC-02.** The system must return HTTP 401 in that case.`, when `parseSpecStructure`
is called, then `spec.acceptanceCriteria` has length 2 with entries
`"AC-01: The system must reject requests without authentication tokens."` and
`"AC-02: The system must return HTTP 401 in that case."`. (FR-006, FR-007)

**AC-005.** Given a spec section recognised as an acceptance-criteria section containing
`- **AC-01.** The system must validate inputs.`, when `asmExtractCriteria` processes it,
then the output contains `"AC-01: The system must validate inputs."` and does NOT contain
`"**AC-01.** The system must validate inputs."`. (FR-008, FR-009)

**AC-006.** Given a spec file with zero parseable criteria where bold-identifier lines
exist outside recognised AC sections, when `DecomposeFeature` is called, then the
returned error string contains the count of sections parsed, their titles, the count of
bold-identifier lines found outside recognised AC sections (greater than zero), and a
remediation suggestion. (FR-010, FR-011)

**AC-007.** Given a spec file with zero parseable criteria and no bold-identifier lines
anywhere in the document, when `DecomposeFeature` is called, then the returned error
string contains the count of sections parsed, their titles, a bold-identifier-outside
count of zero, and a remediation suggestion. (FR-010, FR-011)

**AC-008.** All three named tests — `TestExtractConventionalRoles_ACContentInNonACSection`,
`TestParseSpecStructure_ListItemBoldIdent`, and
`TestAsmExtractCriteria_ListItemBoldIdent_Normalised` — exist in the codebase and pass
when `go test ./...` is run against the completed implementation. (FR-012)

---

## Verification Plan

| Criterion | Method      | Description                                                                                                              |
|-----------|-------------|--------------------------------------------------------------------------------------------------------------------------|
| AC-001    | Test        | `TestExtractConventionalRoles_ACContentInNonACSection` in `internal/docint/extractor_test.go` — asserts Role="requirement", Confidence="medium" for a non-AC-headed section with bold-identifier lines |
| AC-002    | Test        | `TestExtractConventionalRoles_NoDuplicateRole` in `internal/docint/extractor_test.go` — asserts heading-matched section appears exactly once |
| AC-003    | Test        | `TestAcBoldIdentLineRe_BothForms` in `internal/docint/extractor_test.go` — asserts regex matches bare and list-item forms; rejects single-asterisk variant |
| AC-004    | Test        | `TestParseSpecStructure_ListItemBoldIdent` in `internal/service/decompose_test.go` — asserts 2 criteria extracted with canonical `XX-NN: text` format |
| AC-005    | Test        | `TestAsmExtractCriteria_ListItemBoldIdent_Normalised` in `internal/mcp/assembly_test.go` — asserts output is `"AC-01: ..."` not `"**AC-01.** ..."` |
| AC-006    | Test        | `TestDecomposeFeature_RichDiagnostic_BoldOutsideSection` in `internal/service/decompose_test.go` — asserts error contains section list and non-zero outside count |
| AC-007    | Test        | `TestDecomposeFeature_RichDiagnostic_NoBoldIdents` in `internal/service/decompose_test.go` — asserts error contains section list and zero outside count |
| AC-008    | Test        | `go test ./...` passes; all three named verify-then-fix tests are present and green                                     |

---

## Dependencies and Assumptions

**DEP-001:** `FEAT-01KN4ZPCMJ1FP` (docint-ac-pattern-recognition) must be merged before
this feature is implemented. It established `boldIdentifierRe` in `assembly.go`; FR-008
and FR-009 build on that regex.

**DEP-002:** `FEAT-01KMT-58TV8V9C` (decompose-precondition-gates) must be merged. It
removed the section-header fallback and introduced the zero-criteria gate that FR-010
improves.

**DEP-003:** `Section.ByteOffset` and `Section.ByteCount` must be populated by the
Layer 1 parser before FR-003 can be implemented. Confirm with existing `TestSplitSections`
tests that these fields are non-zero for parsed documents.

**ASM-001:** The `**XX-NN.**` pattern does not appear as decorative formatting in running
prose; its presence in a section body is a reliable indicator of structured
acceptance-criteria or requirements content.

**ASM-002:** RFC 2119 keywords for the purposes of FR-009 are those already defined by
`FEAT-01KN4ZPCMJ1FP`: `MUST`, `MUST NOT`, `SHALL`, `SHALL NOT`, `SHOULD`, `SHOULD NOT`,
`MAY`, `REQUIRED`, `RECOMMENDED`, `OPTIONAL` (case-sensitive).