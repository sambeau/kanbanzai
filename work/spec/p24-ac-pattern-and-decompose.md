# Specification: AC Pattern Recognition and Decompose Hardening

| Field   | Value                                                                 |
|---------|-----------------------------------------------------------------------|
| Status  | Draft                                                                 |
| Feature | `FEAT-01KPPG1MF4DAT`                                                  |
| Design  | `work/design/p24-ac-pattern-and-decompose.md`                         |
| Plan    | P24 — Retro Recommendations                                           |

---

## Problem Statement

This specification implements the design described in
`work/design/p24-ac-pattern-and-decompose.md` (FEAT-01KPPG1MF4DAT).

The `**AC-NN.**` bold-identifier format used in Kanbanzai specification
documents is unrecognised at two independent layers of the system, causing
`decompose propose` to fail on well-formed specifications and `doc_intel`
queries to miss AC sections with non-standard headings.

**Layer A — `internal/docint/extractor.go`.** `extractConventionalRoles`
classifies sections by heading keywords only. A section whose heading does not
contain `"acceptance"`, `"criteria"`, or `"requirements"` is never assigned the
`requirement` conventional role, even when its body contains bold-identifier
lines.

**Layer B — `service/decompose.go` and `internal/mcp/assembly.go`.** Both
`parseSpecStructure` and `asmExtractCriteria` fail to match the list-item
variant `- **AC-NN.** text` because their regexes require the line to begin
with `**`. When `parseSpecStructure` returns zero criteria, `DecomposeFeature`
emits a generic error that does not identify the actual cause.

**In scope:**
- Extending `extractConventionalRoles` to perform content-based AC detection
  (Layer 2, deterministic regex, no LLM involvement).
- Fixing `parseSpecStructure` to recognise the list-item bold-identifier form.
- Fixing `asmExtractCriteria` to normalise the list-item form correctly.
- Replacing the generic zero-criteria error in `DecomposeFeature` with a
  structured diagnostic.
- Three verify-then-fix tests (one per fix site).

**Out of scope:**
- Non-standard bold-identifier variants (`*AC-01.*`, `__AC-01.__`, `AC-01:`
  without bold markers).
- Validation that identifier numbers are sequential or non-duplicate.
- Changes to the document record store, index files, or persistence layer.
- Changes to `decompose review` or `decompose apply`.
- Introduction of a new `"acceptance-criteria"` `FragmentRole` constant or
  taxonomy changes.
- Layer 3 (LLM-based) reclassification of AC sections.

---

## Requirements

### Functional Requirements

**REQ-001:** `extractConventionalRoles` in `internal/docint/extractor.go` MUST
accept `content []byte` as its first parameter. The call site in
`ExtractPatterns` MUST pass the document content as this argument.

**REQ-002:** A package-level compiled regex `acBoldIdentLineRe` MUST be defined
in `internal/docint/extractor.go`. It MUST match any line that begins —
optionally after a list marker (`- `, `* `, `+ `) — with a bold-identifier
token of the form `**XX-NN.**` (where `XX` is one or more uppercase ASCII
letters and `NN` is one or more ASCII digits), followed by at least one
whitespace character.

**REQ-003:** After the heading-based walk in `extractConventionalRoles`, every
section that was NOT assigned a role by the heading walk MUST be scanned for
bold-identifier lines. The scan MUST use `acBoldIdentLineRe` against the
byte slice `content[s.ByteOffset : s.ByteOffset+s.ByteCount]`.

**REQ-004:** If at least one line in a section's byte range matches
`acBoldIdentLineRe`, that section MUST be added to the result as
`ConventionalRole{SectionPath: s.Path, Role: "requirement", Confidence:
"medium"}`.

**REQ-005:** Sections that were already assigned a role by the heading-based
walk MUST NOT be re-processed by the content scan. Each section path MUST
appear at most once in the returned slice.

**REQ-006:** In `parseSpecStructure` (`service/decompose.go`), within the
`inACSection` branch, before applying `reBoldIdent` to a line, the
implementation MUST strip a leading list marker (`- `, `* `, or `+ `) from the
trimmed line and re-attempt the regex match on the stripped candidate.

**REQ-007:** When the stripped candidate in REQ-006 matches `reBoldIdent`, the
implementation MUST append a criterion to `spec.acceptanceCriteria` with text
formatted as `XX-NN: <criterion text>` (bold markers stripped, trailing period
inside the bold span stripped, colon and space inserted before the criterion
text).

**REQ-008:** In `asmExtractCriteria` (`internal/mcp/assembly.go`), after the
bare bold-identifier check against `trimmed` fails, the implementation MUST
strip a list marker (`- `, `* `, `+ `, or `• `) from `trimmed` and re-apply
`boldIdentifierRe` to the stripped text.

**REQ-009:** When the re-applied `boldIdentifierRe` in REQ-008 matches, the
implementation MUST emit the criterion in canonical `XX-NN: <criterion text>`
format. The output MUST NOT contain raw bold markers (`**`).

**REQ-010:** In `DecomposeFeature` (`service/decompose.go`), the zero-criteria
error MUST be replaced by a call to a new unexported helper
`buildZeroCriteriaDiagnostic(specDocID string, content []byte, spec
specStructure) error`.

**REQ-011:** `buildZeroCriteriaDiagnostic` MUST produce an error message that
contains all four of the following elements:
- (a) The count of sections parsed from the spec and their titles.
- (b) The titles of sections that were recognised as acceptance-criteria or
  requirement sections.
- (c) The count of bold-identifier lines found inside recognised AC sections
  and outside them.
- (d) A concrete remediation suggestion (e.g., rename the section heading, or
  switch to the `- [ ] text` checkbox form).

**REQ-012:** Three verify-then-fix tests MUST be written before the
corresponding production fix is applied (they MUST fail without the fix and
pass after):
- `TestExtractConventionalRoles_ACContentInNonACSection` in
  `internal/docint/extractor_test.go`.
- `TestParseSpecStructure_ListItemBoldIdent` in
  `internal/service/decompose_test.go`.
- `TestAsmExtractCriteria_ListItemBoldIdent_Normalised` in
  `internal/mcp/assembly_test.go`.

### Non-Functional Requirements

**REQ-NF-001:** The content-based AC detection introduced by REQ-002 through
REQ-004 MUST be deterministic. It MUST NOT invoke any LLM, external service,
or non-deterministic component.

**REQ-NF-002:** `Section.ByteOffset` and `Section.ByteCount` MUST be used to
bound the content slice passed to `acBoldIdentLineRe` (REQ-003). The scan MUST
NOT operate on the full document content for each section.

**REQ-NF-003:** All existing tests in `internal/docint/`, `internal/service/`,
and `internal/mcp/` MUST continue to pass without modification after each fix
is applied.

**REQ-NF-004:** The heading-based role assignment MUST retain
`Confidence: "high"`. Only the content-based path introduced by this
specification uses `Confidence: "medium"`.

---

## Constraints

- The role value assigned by the content scan MUST be `"requirement"`
  (`RoleRequirement`). No new `FragmentRole` constant is introduced.
- `asmExtractCriteria` strips four list marker variants: `- `, `* `, `+ `,
  `• `. `parseSpecStructure` strips three: `- `, `* `, `+ `. These sets are
  defined by the design; no additional markers are in scope.
- Non-standard bold formats (`*AC-01.*`, `__AC-01.__`) MUST NOT be matched.
  The regex MUST require double asterisks.
- This specification does NOT introduce changes to the document record store,
  the `Section` struct, `taxonomy.go`, `AllRoles()`, `ValidRole()`, Layer 3
  classification, or the `doc` / `doc_intel` tool APIs.
- `DEP-01` (`FEAT-01KN4ZPCMJ1FP`, docint-ac-pattern-recognition) must be
  merged before this feature is implemented; `boldIdentifierRe` in
  `assembly.go` must already exist.
- `DEP-02` (`FEAT-01KMT-58TV8V9C`, decompose-precondition-gates) must be
  merged; the zero-criteria gate in `DecomposeFeature` must already exist.
- `DEP-03`: `Section.ByteOffset` and `Section.ByteCount` must be populated by
  the Layer 1 parser. Confirm with existing `TestSplitSections` tests before
  implementing REQ-003.

---

## Acceptance Criteria

**AC-001 (REQ-003, REQ-004):** Given a document with a section titled "Notes"
whose body contains `**AC-01.**` bold-identifier lines and whose heading does
not match any existing heading keyword, when `extractConventionalRoles(content,
sections)` is called, then the "Notes" section appears in the returned slice
with `Role: "requirement"` and `Confidence: "medium"`.

**AC-002 (REQ-005):** Given a document with an "Acceptance Criteria" section
(assigned `Confidence: "high"` by the heading walk) that also contains
bold-identifier lines, when `extractConventionalRoles` is called, then the
returned slice contains exactly one entry for that section (no duplicate from
the content scan).

**AC-003 (REQ-001, REQ-002):** Given that `ExtractPatterns` is called with
document content containing `- **AC-01.** text` in a non-standard-headed
section, then `acBoldIdentLineRe` matches that line, and the section is
assigned a `ConventionalRole`.

**AC-004 (REQ-006, REQ-007):** Given a spec document with the following content
in an Acceptance Criteria section:
```
- **AC-01.** The system must reject requests without authentication tokens.
- **AC-02.** The system must return HTTP 401 in that case.
```
when `parseSpecStructure` is called, then `spec.acceptanceCriteria` has length
2, with entries `"AC-01: The system must reject requests without authentication
tokens."` and `"AC-02: The system must return HTTP 401 in that case."`.

**AC-005 (REQ-008, REQ-009):** Given a spec section with role
`acceptance-criteria` containing `- **AC-01.** The system must validate
inputs.`, when `asmExtractCriteria` processes it, then the output contains
`"AC-01: The system must validate inputs."` and does NOT contain
`"**AC-01.** The system must validate inputs."`.

**AC-006 (REQ-010, REQ-011):** Given a spec file with zero parseable criteria
where bold-identifier lines exist outside recognised AC sections, when
`DecomposeFeature` is called, then the returned error string contains the count
of sections parsed, their titles, the count of bold-identifier lines found
outside recognised AC sections (> 0), and a remediation suggestion.

**AC-007 (REQ-010, REQ-011):** Given a spec file with zero parseable criteria
and no bold-identifier lines anywhere in the document, when `DecomposeFeature`
is called, then the returned error string contains the count of sections parsed,
their titles, a count of bold-identifier lines outside AC sections of 0, and a
remediation suggestion.

**AC-008 (REQ-012):** All three tests —
`TestExtractConventionalRoles_ACContentInNonACSection`,
`TestParseSpecStructure_ListItemBoldIdent`, and
`TestAsmExtractCriteria_ListItemBoldIdent_Normalised` — exist in the codebase
and pass when `go test ./...` is run against the completed implementation.

---

## Verification Plan

| Criterion | Method | Description |
|-----------|--------|-------------|
| AC-001 | Test | `TestExtractConventionalRoles_ACContentInNonACSection` in `internal/docint/extractor_test.go` — asserts Role="requirement", Confidence="medium" for a non-AC-headed section with bold-identifier lines |
| AC-002 | Test | `TestExtractConventionalRoles_NoDuplicateRole` in `internal/docint/extractor_test.go` — asserts the heading-matched section appears exactly once in the result |
| AC-003 | Test | `TestAcBoldIdentLineRe_BothForms` in `internal/docint/extractor_test.go` — asserts regex matches `**AC-01.** text` and `- **AC-01.** text`, rejects `*AC-01.* text` |
| AC-004 | Test | `TestParseSpecStructure_ListItemBoldIdent` in `internal/service/decompose_test.go` — asserts 2 criteria extracted from list-item bold-ident form, text matches canonical format |
| AC-005 | Test | `TestAsmExtractCriteria_ListItemBoldIdent_Normalised` in `internal/mcp/assembly_test.go` — asserts output is `"AC-01: ..."` not `"**AC-01.** ..."` |
| AC-006 | Test | `TestDecomposeFeature_RichDiagnostic_BoldOutsideSection` in `internal/service/decompose_test.go` — asserts error contains section count, titles, non-zero outside-AC bold-ident count, remediation text |
| AC-007 | Test | `TestDecomposeFeature_RichDiagnostic_NoBoldIdents` in `internal/service/decompose_test.go` — asserts error contains section count, titles, zero outside-AC bold-ident count, remediation text |
| AC-008 | Test | `go test ./...` passes after implementation; all three named verify-then-fix tests are present and green |