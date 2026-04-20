# Dev Plan: AC Pattern Recognition and Decompose Hardening

| Field   | Value                                                         |
|---------|---------------------------------------------------------------|
| Date    | 2026-04-20                                                    |
| Author  | Architect                                                     |
| Feature | FEAT-01KPPG1MF4DAT                                            |
| Spec    | work/spec/p24-ac-pattern-and-decompose.md                     |
| Status  | Draft                                                         |

---

## Scope

This plan implements the requirements defined in
`work/spec/p24-ac-pattern-and-decompose.md`
(FEAT-01KPPG1MF4DAT/specification-p24-ac-pattern-and-decompose).

It covers three independent units of work, each touching a distinct file set:

1. Extending `extractConventionalRoles` in `internal/docint/extractor.go` to
   perform content-based AC detection (Layer 2).
2. Fixing `parseSpecStructure` and adding `buildZeroCriteriaDiagnostic` in
   `internal/service/decompose.go`.
3. Fixing `asmExtractCriteria` in `internal/mcp/assembly.go`.

Each task follows the verify-then-fix discipline: the test is authored to fail
without the production fix, and the fix makes it green.

This plan does **not** cover:

- Changes to `taxonomy.go`, `AllRoles()`, `ValidRole()`, or any Layer 3
  classification logic.
- Changes to `decompose review` or `decompose apply`.
- Changes to the document record store or persistence layer.
- Introduction of a new `"acceptance-criteria"` `FragmentRole` constant.

---

## Task Breakdown

### Task 1: Extend `extractConventionalRoles` with content-based AC detection

- **Description:** Update `internal/docint/extractor.go` to teach
  `extractConventionalRoles` to recognise sections whose *body* contains
  `**AC-NN.**` bold-identifier lines, even when the section heading does not
  match any existing keyword.

  Specifically:
  1. Change the function signature from
     `extractConventionalRoles(sections []Section) []ConventionalRole` to
     `extractConventionalRoles(content []byte, sections []Section) []ConventionalRole`.
  2. Update the call site in `ExtractPatterns` to pass the document content.
  3. Define a new package-level compiled regex at the top of `extractor.go`:
     ```
     // acBoldIdentLineRe matches a line that begins (optionally after a list
     // marker) with a bold-identifier token **XX-NN.**
     var acBoldIdentLineRe = regexp.MustCompile(`(?m)^(?:[-*+]\s+)?\*\*[A-Z]+-\d+\.\*\*\s+`)
     ```
  4. After the existing heading-based walk completes, iterate over all sections.
     For each section that did NOT receive a role from the heading walk, slice
     `content[s.ByteOffset : s.ByteOffset+s.ByteCount]` and test it against
     `acBoldIdentLineRe`. If at least one line matches, append:
     ```
     ConventionalRole{SectionPath: s.Path, Role: "requirement", Confidence: "medium"}
     ```
  5. Sections already assigned a heading-based role are skipped (no duplicates).

  Write the verify-then-fix test **before** applying the production fix.
  Confirm the test fails on the unmodified code, then apply the fix and confirm
  it passes.

- **Deliverable:**
  - Modified `internal/docint/extractor.go`
  - New or extended test cases in `internal/docint/extractor_test.go`:
    - `TestExtractConventionalRoles_ACContentInNonACSection` — section titled
      "Notes" with `**AC-01.**` lines receives Role="requirement",
      Confidence="medium".
    - `TestExtractConventionalRoles_NoDuplicateRole` — a heading-matched
      "Acceptance Criteria" section appears exactly once in the result.
    - `TestAcBoldIdentLineRe_BothForms` — regex matches bare
      `**AC-01.** text` and list-item `- **AC-01.** text`; rejects
      `*AC-01.* text` (single asterisks).

- **Depends on:** None (independent)

- **Effort:** Medium

- **Spec requirements:** FR-001, FR-002, FR-003, FR-004, FR-005, NFR-001,
  NFR-002, NFR-003, NFR-004, FR-012 (first test)

- **AC coverage:** AC-001, AC-002, AC-003

**Interface contract (consumed by no other task in this feature; purely
internal):**

```
// extractConventionalRoles returns ConventionalRole entries for sections
// whose heading or body content indicates a requirement/AC role.
// Heading-based matches carry Confidence "high"; content-based matches
// carry Confidence "medium". Each section path appears at most once.
func extractConventionalRoles(content []byte, sections []Section) []ConventionalRole
```

**Input context:**
- `internal/docint/extractor.go` — current signature of `extractConventionalRoles`
  and `ExtractPatterns` call site.
- `internal/docint/types.go` (or equivalent) — `Section.ByteOffset`,
  `Section.ByteCount`, `ConventionalRole` struct, `RoleRequirement` constant.
- `internal/docint/extractor_test.go` — existing test patterns.
- FR-001 through FR-005, NFR-001 through NFR-004 in the spec.
- DEP-003: verify `TestSplitSections` (or equivalent) confirms that
  `Section.ByteOffset` and `Section.ByteCount` are non-zero for parsed
  documents before implementing FR-003.

---

### Task 2: Fix `parseSpecStructure` and add `buildZeroCriteriaDiagnostic`

- **Description:** Two targeted changes to `internal/service/decompose.go`,
  both in the same file and best implemented together to avoid conflicting edits.

  **Fix A — `parseSpecStructure` list-item bold-identifier extraction (FR-006,
  FR-007):**
  Within the `inACSection` branch, before applying `reBoldIdent` to `trimmed`,
  strip a leading list marker (`- `, `* `, or `+ `) and re-attempt the regex
  match on the stripped candidate. If it matches, append a criterion with text
  `XX-NN: <criterion text>` (bold markers and trailing period stripped). The
  existing numbered-list and table checks remain unchanged.

  Example: for a line `- **AC-01.** The system must reject empty inputs.`,
  `trimmed` starts with `- ` → stripped candidate is `**AC-01.** The system
  must reject empty inputs.` → `reBoldIdent` matches → criterion appended as
  `"AC-01: The system must reject empty inputs."`.

  **Fix B — `buildZeroCriteriaDiagnostic` (FR-010, FR-011):**
  Replace the generic zero-criteria `fmt.Errorf(...)` in `DecomposeFeature`
  with a call to a new unexported helper:
  ```
  func buildZeroCriteriaDiagnostic(specDocID string, content []byte, spec specStructure) error
  ```
  The helper scans `spec.sections` for titles and which were recognised as AC
  sections. It also scans `content` for `**[A-Z]+-\d+\.**` occurrences to
  count bold-identifier lines found inside and outside recognised AC sections.
  It returns a structured error message containing all four required elements
  from FR-011: section count + titles, recognised AC section titles,
  bold-identifier counts inside/outside, and a concrete remediation suggestion.

  Write the verify-then-fix test for Fix A **before** applying it.

- **Deliverable:**
  - Modified `internal/service/decompose.go`
  - New or extended test cases in `internal/service/decompose_test.go`:
    - `TestParseSpecStructure_ListItemBoldIdent` — spec with
      `- **AC-01.** text` and `- **AC-02.** text` in an Acceptance Criteria
      section produces exactly 2 criteria with canonical `XX-NN: text` format.
    - `TestDecomposeFeature_RichDiagnostic_BoldOutsideSection` — spec with
      bold-identifier lines outside recognised AC sections; error contains
      section list and non-zero outside count.
    - `TestDecomposeFeature_RichDiagnostic_NoBoldIdents` — spec with no
      bold-identifier lines anywhere; error contains section list and zero
      outside count.

- **Depends on:** None (independent; DEP-002 must be merged as a precondition)

- **Effort:** Medium

- **Spec requirements:** FR-006, FR-007, FR-010, FR-011, FR-012 (second test)

- **AC coverage:** AC-004, AC-006, AC-007

**Input context:**
- `internal/service/decompose.go` — `parseSpecStructure`, `reBoldIdent`,
  `specStructure`, `acceptanceCriterion`, `DecomposeFeature` zero-criteria gate.
- `internal/service/decompose_test.go` — existing test patterns.
- FR-006, FR-007, FR-010, FR-011 in the spec.
- DEP-002: confirm `FEAT-01KMT-58TV8V9C` is merged and the zero-criteria gate
  is present in `DecomposeFeature` before implementing Fix B.

---

### Task 3: Fix `asmExtractCriteria` list-item bold-identifier normalisation

- **Description:** Update `asmExtractCriteria` in `internal/mcp/assembly.go`
  so that `- **AC-NN.** text` is extracted as `AC-NN: text` rather than being
  kept with raw bold markers or dropped.

  After the existing bare bold-identifier check against `trimmed` fails (i.e.,
  `boldIdentifierRe.FindStringSubmatch(trimmed)` returns nil), strip a list
  marker (`- `, `* `, `+ `, or `•`) from `trimmed` to produce `bare`. If
  `bare != trimmed`, re-apply `boldIdentifierRe` to `bare`. On a match, build
  `criterion := prefix + "-" + num + ": " + criterionText` and apply the
  existing context-sensitive rule: emit unconditionally in a recognised
  acceptance/requirement/constraint section (`isAcceptanceSection == true`);
  emit only if `hasRFC2119Keyword(criterionText)` is true otherwise. Then
  `continue` to skip the generic list-item path.

  Write the verify-then-fix test **before** applying the fix.

- **Deliverable:**
  - Modified `internal/mcp/assembly.go`
  - New test in `internal/mcp/assembly_test.go`:
    - `TestAsmExtractCriteria_ListItemBoldIdent_Normalised` — section with
      `- **AC-01.** The system must validate inputs.` and
      `- **AC-02.** The system must log failures.` produces
      `["AC-01: The system must validate inputs.", "AC-02: The system must log failures."]`.

- **Depends on:** None (independent; DEP-001 must be merged as a precondition)

- **Effort:** Small

- **Spec requirements:** FR-008, FR-009, NFR-003, FR-012 (third test)

- **AC coverage:** AC-005, AC-008

**Input context:**
- `internal/mcp/assembly.go` — `asmExtractCriteria`, `boldIdentifierRe`,
  `isAcceptanceSection` logic, `hasRFC2119Keyword`, `addCriterion`.
- `internal/mcp/assembly_test.go` — existing test patterns for
  `asmExtractCriteria` introduced by FEAT-01KN4ZPCMJ1FP.
- FR-008, FR-009 in the spec.
- DEP-001: confirm `FEAT-01KN4ZPCMJ1FP` is merged and `boldIdentifierRe` is
  defined in `assembly.go` before starting.

---

## Dependency Graph

```
Task 1 — internal/docint/extractor.go        (no task dependencies)
Task 2 — internal/service/decompose.go       (no task dependencies)
Task 3 — internal/mcp/assembly.go            (no task dependencies)

Parallel groups: [Task 1, Task 2, Task 3]
Critical path:   any single task (all are leaf nodes)
```

All three tasks are fully independent within this feature. They touch disjoint
file sets and share no interfaces. They may be dispatched in parallel to three
separate agents.

External preconditions (not tasks in this plan):
- DEP-001 (`FEAT-01KN4ZPCMJ1FP`) must be merged before Task 3 starts.
- DEP-002 (`FEAT-01KMT-58TV8V9C`) must be merged before Task 2's Fix B starts.
- DEP-003: `Section.ByteOffset` / `Section.ByteCount` must be confirmed
  non-zero before Task 1 starts.

---

## Risk Assessment

### Risk: `Section.ByteOffset` / `Section.ByteCount` are zero

- **Probability:** Low
- **Impact:** High — the content scan in Task 1 silently produces no matches
  for every section, making `acBoldIdentLineRe` unreachable without runtime
  errors.
- **Mitigation:** Before implementing FR-003, run the existing `TestSplitSections`
  tests (or equivalent) and assert that parsed sections have non-zero
  `ByteCount`. If the fields are zero, file a separate bug; do not work around
  it within this feature.
- **Affected tasks:** Task 1

### Risk: `boldIdentifierRe` not present (DEP-001 not merged)

- **Probability:** Low (DEP-001 is a completed feature)
- **Impact:** Medium — Task 3 cannot compile without the symbol.
- **Mitigation:** Confirm `boldIdentifierRe` is exported or unexported and
  accessible in `assembly.go` before starting Task 3. If the symbol is absent,
  pause and resolve the dependency before proceeding.
- **Affected tasks:** Task 3

### Risk: zero-criteria gate absent (DEP-002 not merged)

- **Probability:** Low (DEP-002 is a completed feature)
- **Impact:** Medium — Fix B in Task 2 replaces an error path that does not
  exist yet, producing a compilation error.
- **Mitigation:** Confirm the zero-criteria `if len(spec.acceptanceCriteria) == 0`
  gate is present in `DecomposeFeature` before starting Fix B. If absent, pause
  and resolve the dependency.
- **Affected tasks:** Task 2

### Risk: list-marker stripping interacts with existing extraction paths

- **Probability:** Low
- **Impact:** Medium — an off-by-one in the strip loop could cause `trimmed`
  to be double-stripped or to consume characters from the criterion text.
- **Mitigation:** Unit tests cover the boundary: a line with exactly `- ` as
  its only prefix; a line with `- * ` double-prefixed; a bare `**AC-01.**`
  line with no list marker. Confirm all pass before merging.
- **Affected tasks:** Task 2, Task 3

---

## Verification Approach

| Acceptance Criterion | Verification Method   | Producing Task |
|---------------------|-----------------------|----------------|
| AC-001              | Unit test             | Task 1         |
| AC-002              | Unit test             | Task 1         |
| AC-003              | Unit test             | Task 1         |
| AC-004              | Unit test             | Task 2         |
| AC-005              | Unit test             | Task 3         |
| AC-006              | Unit test             | Task 2         |
| AC-007              | Unit test             | Task 2         |
| AC-008              | `go test ./...`       | All tasks      |
| NFR-003 (regression)| `go test ./...`       | All tasks      |
| NFR-004 (confidence)| Code inspection       | Task 1         |