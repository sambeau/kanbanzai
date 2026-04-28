# Implementation Plan: `docint` Acceptance Criteria Pattern Recognition

| Field    | Value                                                                   |
|----------|-------------------------------------------------------------------------|
| Status   | Draft                                                                   |
| Created  | 2026-04-01                                                              |
| Feature  | FEAT-01KN4ZPCMJ1FP (docint-ac-pattern-recognition)                     |
| Spec     | `work/spec/docint-ac-pattern-recognition.md`                            |
| Design   | `work/design/kanbanzai-2.5-infrastructure-hardening.md` §3             |

---

## 1. Overview

This feature is a focused, self-contained fix to a single function:
`asmExtractCriteria` in `internal/mcp/assembly.go`. The function must be
extended to recognise the `**XX-NN.**` bold-identifier pattern used in
Kanbanzai specifications, in addition to the list-item and numbered-list
patterns it already handles.

The implementation has two tasks:

**Task 1 — Implementation + unit tests** (the core change, TDD).  
**Task 2 — Integration test** (confirms the fix reaches `decompose propose`
end-to-end).

These tasks are serial: Task 2 depends on Task 1 being complete and passing.

```
[Task 1: asmExtractCriteria fix + unit tests] → [Task 2: integration test]
```

Estimated total: 2–3 hours.

---

## 2. Context Reading

Before starting, read:

1. `work/spec/docint-ac-pattern-recognition.md` — requirements and ACs in full.
2. `internal/mcp/assembly.go` — locate `asmExtractCriteria`; understand the
   current extraction loop and how section role information is passed in.
3. `internal/mcp/assembly_test.go` — understand the existing test fixture
   patterns; do not break them.
4. `work/design/kanbanzai-2.5-infrastructure-hardening.md` §3 — design
   rationale and the extraction rule summary.

Commit policy: `work/design/git-commit-policy.md`.

---

## 3. Interface Contract

`asmExtractCriteria` has an existing signature. **Do not change it.** The fix
is internal to the function body only. Whatever parameters currently convey
the section's classifier role must be used to implement the context-sensitive
rules (REQ-04 and REQ-05). If the section role is not currently passed as a
parameter, read `assembly.go` to understand how section context is threaded
through the call and follow the same pattern.

The extracted criterion string format (REQ-07):

```
"XX-NN: <criterion text>"
```

Example: `**AC-01.** The system must do X.`  →  `"AC-01: The system must do X."`

This format is the contract between Task 1's output and Task 2's assertion.

---

## 4. Task Details

### Task 1 — Extend `asmExtractCriteria` + Unit Tests

**Objective:** Add recognition of the `**XX-NN.**` bold-identifier pattern to
`asmExtractCriteria`, with context-sensitive extraction rules, and cover all
new behaviour with unit tests.

**Spec references:** REQ-01 through REQ-12, AC-01 through AC-05.

**Input context:**
- `internal/mcp/assembly.go` — function to modify.
- `internal/mcp/assembly_test.go` — existing tests must continue to pass.
- Spec §4 (Requirements) — all nine REQ entries.

**Implementation steps:**

1. Add a compiled regex for the bold-identifier pattern. The pattern should
   match lines of the form `**XX-NN.** <text>` where XX is one or more
   uppercase ASCII letters and NN is one or more digits. Compile at package
   initialisation (not inside the function) to avoid repeated compilation.

2. In the extraction loop, after checking for list items and numbered lists,
   check for the bold-identifier pattern.

3. Apply the context-sensitive rule:
   - If the current section's role is `acceptance-criteria`, `requirements`,
     or `constraints`: extract all matching lines unconditionally.
   - Otherwise: extract only if the criterion text also contains an RFC 2119
     keyword (MUST, MUST NOT, SHALL, SHALL NOT, SHOULD, SHOULD NOT, MAY,
     REQUIRED, RECOMMENDED, OPTIONAL).

4. Format the extracted criterion as `"XX-NN: <text>"` — strip the `**` bold
   markers and the trailing period from the identifier; add `: ` separator;
   append the criterion text verbatim.

**Output artifacts:**
- `internal/mcp/assembly.go` — extended `asmExtractCriteria`.
- `internal/mcp/assembly_test.go` — new test cases:

| Test name | What it covers |
|-----------|----------------|
| `TestAsmExtractCriteria_BoldAC_InACSection` | AC-01: `**AC-NN.**` in AC section → extracted |
| `TestAsmExtractCriteria_BoldREQ_InReqSection` | AC-02: `**REQ-NN.**` in requirements section → extracted |
| `TestAsmExtractCriteria_BoldC_InConstraintsSection` | AC-02: `**C-NN.**` in constraints section → extracted |
| `TestAsmExtractCriteria_BoldIdent_PreservesPrefix` | AC-03: extracted text is `"XX-NN: ..."` |
| `TestAsmExtractCriteria_BoldIdent_OutsideACSection_NoKeyword` | AC-04: bold-ident outside AC section, no RFC 2119 → not extracted |
| `TestAsmExtractCriteria_BoldIdent_OutsideACSection_WithKeyword` | AC-04: bold-ident outside AC section, with MUST → extracted |
| `TestAsmExtractCriteria_ListItems_Unaffected` | AC-05 regression: list-item extraction unchanged |
| `TestAsmExtractCriteria_NumberedList_Unaffected` | AC-05 regression: numbered-list extraction unchanged |
| `TestAsmExtractCriteria_NoBoldIdents_ZeroCriteria` | REQ-12: no bold-idents, no list items → zero results |

**Dependencies:** None (first task).

**Definition of done:** All new and existing tests pass under
`go test ./internal/mcp/... -race`.

---

### Task 2 — Integration Test: `decompose propose` End-to-End

**Objective:** Confirm that `decompose propose`, called on a specification
document whose acceptance criteria section uses exclusively the `**AC-NN.**`
format, produces task summaries derived from the criteria text rather than
section headings.

**Spec references:** REQ-13, REQ-14, AC-06.

**Input context:**
- `internal/mcp/assembly.go` — completed Task 1 output.
- Locate the existing `decompose propose` integration tests (search for test
  files that exercise `decompose` in `internal/mcp/`). Follow the same fixture
  pattern.
- Spec §4.6 for the pass/fail definition of REQ-13 and REQ-14.

**Output artifacts:**
- A fixture file `internal/mcp/testdata/spec-with-bold-ac.md` (or equivalent
  path following the existing testdata convention). This is a minimal markdown
  specification document containing:
  - A `## Acceptance Criteria` section heading.
  - Three or more lines in `**AC-NN.** text` format.
  - No list-item or numbered-list criteria.
- A new test function (or test case added to an existing integration test) that
  calls `decompose propose` on the fixture and asserts:
  1. The number of tasks generated equals the number of `**AC-NN.**` lines in
     the fixture's AC section (or at least one task per criterion).
  2. No task summary matches the pattern `"Implement <section heading>"` where
     `<section heading>` is a heading from the fixture document.
  3. At least one task summary contains text from a criterion in the fixture.

**Dependencies:** Task 1 must be complete and all Task 1 tests must pass.

**Definition of done:** The integration test passes under
`go test ./internal/mcp/... -race`. The `go vet ./...` check produces no new
warnings.

---

## 5. Acceptance Criteria Traceability

| AC   | Satisfied by |
|------|--------------|
| AC-01 | Task 1: `TestAsmExtractCriteria_BoldAC_InACSection` |
| AC-02 | Task 1: `TestAsmExtractCriteria_BoldREQ_InReqSection`, `..._BoldC_...` |
| AC-03 | Task 1: `TestAsmExtractCriteria_BoldIdent_PreservesPrefix` |
| AC-04 | Task 1: `TestAsmExtractCriteria_BoldIdent_OutsideACSection_*` |
| AC-05 | Task 1: `TestAsmExtractCriteria_ListItems_Unaffected`, `..._NumberedList_...` |
| AC-06 | Task 2: integration test assertion 2 + 3 |

---

## 6. Scope Boundaries (carried from spec)

**In scope:** `internal/mcp/assembly.go`, `internal/mcp/assembly_test.go`,
one test fixture file.

**Out of scope:** `internal/docint/taxonomy.go`; `decompose propose` task
construction logic; stored document records; any persistence layer.

Do not modify the function signature of `asmExtractCriteria` or any other
exported function.