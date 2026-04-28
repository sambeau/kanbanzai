# Dev Plan: Doc-Intel Register Workflow Friction

**Feature:** FEAT-01KPVDDYSEK8P
**Plan:** P28 — Doc-Intel Polish and Workflow Reliability
**Spec:** work/spec/p28-doc-intel-register-workflow.md
**Status:** Draft

---

## Overview

Two targeted improvements to reduce friction in the doc-intel registration and classification
workflow. First, promote `classification_nudge` from a plain string to a structured object
containing `message`, `content_hash`, and `outline`, shrinking the classify-on-register
workflow from three tool calls to two (§5.4). Second, audit all MCP parameter structs decoded
via `req.RequireString` + `json.Unmarshal`, add missing `json:` tags, and add a regression
test to prevent future drift (§5.5).

---

## Scope

This dev plan implements the requirements in specification
`work/spec/p28-doc-intel-register-workflow.md` (FEAT-01KPVDDYSEK8P), §5.4 and §5.5 of the
P28 design document. Changes are confined to:

- The `doc` tool handler and its registration paths (single and batch).
- The `internal/` structs identified by the `req.RequireString` + `json.Unmarshal` audit.
- Test files for the above.

No new MCP tool actions, parameters, or structural refactoring of unrelated types are in scope.

---

## Task Breakdown

### Task 1: Promote `classification_nudge` to structured object (§5.4)

- **Description:** Change the `classification_nudge` field in the `doc(action: "register")`
  response from a plain string to a structured object. Define a `ClassificationNudge` struct
  (or equivalent response type) with three fields: `message` (string — the existing
  instructional text, unchanged), `content_hash` (string — the document's content hash from
  the Layer 1 index), and `outline` (array of `{path, title, level, word_count}` objects
  already available from the Layer 1 index). Update both the single-registration path and the
  batch-registration path so each result includes the enhanced nudge. The `message` field MUST
  be serialised first to preserve backwards compatibility for callers that do string-prefix
  checks on the raw response.
- **Deliverable:** Updated `doc` tool handler (single + batch registration paths) returning the
  structured `classification_nudge` object; new or updated response type definition.
- **Depends on:** None
- **Effort:** Medium
- **Spec requirements:** REQ-001, REQ-002, REQ-003, REQ-004, REQ-005, REQ-006, REQ-NF-001, REQ-NF-003

### Task 2: Audit and fix `json:` tags on MCP parameter structs (§5.5)

- **Description:** Search the `internal/` tree for all structs that are decoded from a JSON
  string tool parameter via the `req.RequireString` + `json.Unmarshal` pattern. For each
  identified struct, inspect every exported field and add a `json:` tag where one is missing,
  using the snake_case name that matches the JSON key already expected by callers. Do not
  modify structs outside the identified decode pattern. Document the list of structs found
  during the audit (a comment in the test file added in Task 3 is sufficient).
- **Deliverable:** Updated struct definitions in `internal/` with `json:` tags on all exported
  fields of the identified structs.
- **Depends on:** None
- **Effort:** Small
- **Spec requirements:** REQ-007, REQ-008

### Task 3: Add `json:` tag regression test (§5.5)

- **Description:** Write a Go test that uses `reflect` to iterate over every exported field of
  each struct identified in Task 2 and asserts that a non-empty `json:` struct tag is present.
  The test must fail with a message identifying the struct name and field name if a tag is
  missing or empty. The test must compile without errors and complete in under 2 seconds in the
  standard `go test ./...` run.
- **Deliverable:** New or updated test file in `internal/` containing the `json:` tag
  regression test.
- **Depends on:** Task 2
- **Effort:** Small
- **Spec requirements:** REQ-009, REQ-NF-002

### Task 4: Unit tests for enhanced `classification_nudge` (§5.4)

- **Description:** Write unit tests covering the enhanced nudge:
  - AC-001: Register a fixture document; assert `classification_nudge` is a JSON object with
    keys `message`, `content_hash`, and `outline`.
  - AC-002: Assert `classification_nudge.message` equals the expected instructional string.
  - AC-003: Extract `content_hash` from the nudge; pass to `doc_intel(action: "classify")`
    and assert no hash-mismatch error.
  - AC-004: Assert `classification_nudge.outline` deep-equals the section array returned by
    `doc_intel(action: "guide")` for the same fixture document.
  - AC-005: Batch-register 3 fixture documents; assert each result contains a structurally
    valid `classification_nudge` with non-empty `content_hash` and non-empty `outline`.
  - AC-009: Benchmark the registration handler over 100 invocations with a 50-section fixture;
    assert p99 delta vs baseline does not exceed 50 ms.
- **Deliverable:** New or updated test file covering AC-001 through AC-006 and AC-009.
- **Depends on:** Task 1
- **Effort:** Medium
- **Spec requirements:** REQ-001, REQ-002, REQ-003, REQ-004, REQ-005, REQ-006, REQ-NF-001

---

## Dependency Graph

```
Task 1 (promote classification_nudge to struct)
    └─► Task 4 (unit tests for enhanced nudge)

Task 2 (audit + fix json: tags)
    └─► Task 3 (json: tag regression test)
```

Tasks 1 and 2 are independent and can proceed in parallel.
Task 3 depends on Task 2; Task 4 depends on Task 1.

---

## Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| Callers that treat `classification_nudge` as a plain string break after type change | Medium | High | Ensure `message` is the first serialised field (REQ-NF-003); review existing callers before merging. |
| Outline retrieval adds latency beyond the 50 ms budget (REQ-NF-001) | Low | Medium | Outline is sourced from already-indexed Layer 1 data — no additional file I/O. Benchmark in Task 4 catches regressions. |
| `json:` tag audit misses a struct (REQ-007) | Low | Low | Regression test in Task 3 acts as a compile-time + run-time safety net; code review confirms audit completeness. |
| Batch registration path diverges from single path (REQ-005) | Low | Medium | Shared helper for nudge construction; covered by AC-005 in Task 4 tests. |

---

## Verification Approach

| Acceptance Criterion | Verification Method | Producing Task |
|----------------------|---------------------|----------------|
| AC-001 — nudge is a JSON object with `message`, `content_hash`, `outline` | Unit test | Task 4 |
| AC-002 — `message` equals previous instructional string | Unit test | Task 4 |
| AC-003 — `content_hash` passes to `classify` without hash-mismatch error | Unit test | Task 4 |
| AC-004 — `outline` deep-equals `guide` response for same document | Unit test | Task 4 |
| AC-005 — batch registration: each result has structured nudge | Unit test | Task 4 |
| AC-006 — register + classify in 2 calls (no intermediate guide) | Demo / test | Task 4 |
| AC-007 — all exported fields of identified structs have `json:` and `yaml:` tags | Code inspection + test | Task 2 + Task 3 |
| AC-008 — tag regression test fails when `json:` tag removed | Test | Task 3 |
| AC-009 — p99 latency delta ≤ 50 ms over 100 invocations | Benchmark test | Task 4 |
| AC-010 — callers reading only `message` continue to work | Code inspection | Task 1 |