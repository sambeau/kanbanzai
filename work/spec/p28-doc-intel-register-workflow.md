| Field  | Value                                                                 |
|--------|-----------------------------------------------------------------------|
| Date   | 2025-07-14                                                            |
| Status | approved |
| Author | spec-author                                                           |

# Specification: Doc-Intel Register Workflow Friction

**Feature:** FEAT-01KPVDDYSEK8P  
**Design reference:** P28-doc-intel-polish-workflow-reliability/design-p28-doc-intel-polish-workflow-reliability

## Overview

This specification covers two friction points in the doc-intel registration and classification
workflow: enhancing the `classification_nudge` field returned by `doc(action: "register")` to
include `content_hash` and section outline (reducing a three-call workflow to two), and auditing
and fixing missing `json:` tags on MCP parameter structs to prevent silent field-mapping failures.

## Problem Statement

This specification covers two friction points in the doc-intel registration and classification
workflow, both identified in Sprint 1 of the P28 plan design (§5.4 and §5.5).

**§5.4 — Workflow round-trip reduction.**  
When an agent registers a document, it currently receives a `classification_nudge` string
instructing it to call `doc_intel(action: "guide")` before calling
`doc_intel(action: "classify")`. This forces three tool calls (register → guide → classify)
for a task that logically requires two. Agents that already have the document content in
context at registration time have all the information needed to classify immediately; they
only lack the `content_hash` and section `outline` that the classify call requires.
Returning those values inside the nudge eliminates the intermediate guide call.

**§5.5 — JSON tag gap in MCP parameter structs.**  
The `Classification` struct in `internal/docint/types.go` had `yaml:` tags but no `json:`
tags, causing silent field-mapping failures when the struct was decoded via
`req.RequireString` + `json.Unmarshal`. The same pattern may exist in other parameter
structs. A targeted audit and a companion regression test are needed.

## Scope

**In scope:**
- Enhancing the `classification_nudge` field returned by `doc(action: "register")` (single
  and batch) to a structured object.
- Auditing and fixing `json:` tags on structs decoded from JSON string parameters.
- A Go test asserting `json:` tag completeness on identified structs.

**Out of scope:**
- Changes to `doc_intel(action: "guide")` itself.
- Any change to the `document` object returned alongside the nudge.
- New MCP tool actions or parameters.
- Structural refactoring of unrelated types.

---

## Functional Requirements

**REQ-001 — Structured `classification_nudge` object.**  
`doc(action: "register")` MUST return `classification_nudge` as a structured object
containing exactly three fields: `message` (string), `content_hash` (string), and
`outline` (array of section objects).

**REQ-002 — `message` field content.**  
The `message` field inside `classification_nudge` MUST contain the same instructional
string that the field previously held as a plain string value, unchanged.

**REQ-003 — `content_hash` field value.**  
The `content_hash` field inside `classification_nudge` MUST contain the document's content
hash — the identical value that must be passed as `content_hash` to a subsequent
`doc_intel(action: "classify")` call for the same document.

**REQ-004 — `outline` field value.**  
The `outline` field inside `classification_nudge` MUST contain the section tree that
`doc_intel(action: "guide")` would return for the same document: an array of objects each
with fields `path` (string), `title` (string), `level` (integer), and `word_count`
(integer).

**REQ-005 — Batch registration nudge.**  
When `doc(action: "register")` is called with the `documents` array (batch mode), each
item in the response MUST include a `classification_nudge` structured object with the same
three fields (REQ-001 through REQ-004) for its respective document.

**REQ-006 — Reduced workflow call count.**  
An agent that has document content in context at registration time MUST be able to call
`doc(action: "register")` and then immediately call `doc_intel(action: "classify")` using
only values returned by the register response — without issuing a separate
`doc_intel(action: "guide")` call.

**REQ-007 — JSON tag audit scope.**  
All structs in the `internal/` tree that are decoded from a JSON string tool parameter via
the `req.RequireString` + `json.Unmarshal` pattern MUST be identified and listed.

**REQ-008 — JSON tag remediation.**  
Every exported field of each struct identified in REQ-007 that has a `yaml:` tag but is
missing a corresponding `json:` tag MUST have a `json:` tag added. The tag value MUST use
the snake_case name that matches the JSON key already expected by callers.

**REQ-009 — JSON tag regression test.**  
A Go test MUST assert, at runtime, that every exported field of each struct identified in
REQ-007 has a non-empty `json:` struct tag. The test MUST fail if a new exported field is
added to any of those structs without a `json:` tag.

## Non-Functional Requirements

**REQ-NF-001 — Registration response latency.**  
The p99 latency of `doc(action: "register")` for a document with up to 50 sections MUST
NOT increase by more than 50 ms compared to the baseline (nudge as plain string) measured
on the same hardware. The outline is computed from already-indexed data; no additional
file I/O is permitted on the hot path.

**REQ-NF-002 — Test compilation and runtime.**  
The JSON tag regression test (REQ-009) MUST compile without errors and complete execution
in under 2 seconds in the standard `go test ./...` run.

**REQ-NF-003 — Backwards compatibility.**  
Agents or integrations that read only the `message` field of `classification_nudge` MUST
continue to work without modification. The `message` field MUST be the first field in the
serialised object so that simple string-prefix checks on the raw response are unaffected.

---

## Constraints

- The `classification_nudge` field MUST remain present in every registration response
  (single and batch). It MUST NOT become conditional, optional, or omitted for any
  document type or registration path.
- The existing fields of the `document` response object (e.g. `id`, `path`, `title`,
  `type`, `status`) MUST NOT be renamed, removed, or have their types changed.
- The JSON tag audit MUST NOT modify structs that are not used in the
  `req.RequireString` + `json.Unmarshal` decode pattern. It is a targeted fix, not a
  blanket codebase refactor.
- No new MCP tool actions, tool names, or tool parameters are introduced by this feature.
- The `doc_intel(action: "guide")` action MUST remain available and fully functional; it
  is not deprecated by this feature.

---

## Acceptance Criteria

**AC-001 (REQ-001):** Given a document exists on disk, when `doc(action: "register")` is
called with a valid `path`, `type`, and `title`, then the response contains a
`classification_nudge` field that is a JSON object (not a string) with keys `message`,
`content_hash`, and `outline`.

**AC-002 (REQ-002):** Given a registration response is received, when the
`classification_nudge.message` field is read, then its value is identical to the
instructional string that was previously returned as the entire `classification_nudge`
value before this feature was implemented.

**AC-003 (REQ-003):** Given a registration response containing `classification_nudge`,
when `doc_intel(action: "classify")` is called with `content_hash` set to
`classification_nudge.content_hash`, then the classify call succeeds (does not return a
hash-mismatch error).

**AC-004 (REQ-004):** Given a registration response, when `classification_nudge.outline`
is inspected, then it contains the same array of section objects (same paths, titles,
levels, and word counts) that `doc_intel(action: "guide")` returns for the same document.

**AC-005 (REQ-005):** Given a batch registration call with N documents, when the response
is received, then each of the N result objects contains a `classification_nudge` with
`message`, `content_hash`, and `outline` for its respective document.

**AC-006 (REQ-006):** Given a document's content is in the agent's context, when the
agent calls `doc(action: "register")` and then immediately calls
`doc_intel(action: "classify")` using only `content_hash` and `outline` from the register
response (no intermediate `guide` call), then the classify call succeeds and returns a
valid classification result.

**AC-007 (REQ-007, REQ-008):** Given the audit of `internal/` structs decoded via
`req.RequireString` + `json.Unmarshal`, when the identified structs are inspected, then
every exported field has both a `yaml:` tag and a `json:` tag with a non-empty value.

**AC-008 (REQ-009):** Given the JSON tag regression test is run via `go test`, when a
struct identified in the audit has an exported field with a missing or empty `json:` tag,
then the test fails with a message identifying the struct and field name.

**AC-009 (REQ-NF-001):** Given a document with 50 sections, when `doc(action: "register")`
is called 100 times in isolation, then the p99 response time does not exceed the
pre-feature baseline by more than 50 ms.

**AC-010 (REQ-NF-003):** Given code that reads only `classification_nudge.message` (or
performs a string-prefix check on the field), when the feature is deployed, then that
code continues to function correctly without modification.

---

## Verification Plan

| Criterion | Method     | Description                                                                                          |
|-----------|------------|------------------------------------------------------------------------------------------------------|
| AC-001    | Test       | Unit test: call register handler with a fixture document; assert response shape via JSON unmarshal into typed struct with `message`, `content_hash`, `outline` fields. |
| AC-002    | Test       | Unit test: compare `classification_nudge.message` against the hardcoded expected instructional string. |
| AC-003    | Test       | Integration test: register a document, extract `content_hash` from nudge, pass to classify; assert no hash-mismatch error is returned. |
| AC-004    | Test       | Unit test: register a fixture document; independently call guide for the same document; assert `nudge.outline` deep-equals the guide response sections. |
| AC-005    | Test       | Unit test: batch register 3 fixture documents; assert each result contains a structurally valid `classification_nudge` with non-empty `content_hash` and non-empty `outline`. |
| AC-006    | Demo       | End-to-end walkthrough: agent registers a document and classifies it in 2 tool calls; verify no guide call appears in the tool call log. |
| AC-007    | Inspection | Code review: inspect each struct identified by the audit; confirm all exported fields carry both `yaml:` and `json:` tags. |
| AC-008    | Test       | Run `go test ./internal/...`; confirm the tag regression test passes on the fixed code and fails when a `json:` tag is artificially removed in a test scenario. |
| AC-009    | Test       | Benchmark test: measure `doc register` handler p99 over 100 invocations with a 50-section fixture; assert delta ≤ 50 ms vs baseline. |
| AC-010    | Inspection | Review any existing callers that reference `classification_nudge`; confirm they compile and behave correctly after the type change. |