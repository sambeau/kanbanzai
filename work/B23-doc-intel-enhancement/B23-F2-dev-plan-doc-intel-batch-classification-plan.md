# Dev Plan: Batch Classification Activation

> Feature: FEAT-01KPNNYYXQSYW — Batch Classification Activation
> Spec: work/spec/doc-intel-batch-classification.md

---

## Overview

This plan implements `work/spec/doc-intel-batch-classification.md`. The only code
change is adding a `classification_nudge` string field to the `doc(action: "register")`
response. Additionally, a batch classification workflow protocol document is produced.

The feature has no external dependencies and can be implemented in two independent
tasks.

---

## Task Breakdown

### Task 1: Add classification_nudge to doc register response

**Description:** Modify `docRegisterOne` in `internal/mcp/doc_tool.go` to include a
`classification_nudge` field in the response map. The nudge is a concise static
template with the registered document ID interpolated. Add tests in
`internal/mcp/doc_tool_test.go`.

**Files:** `internal/mcp/doc_tool.go`, `internal/mcp/doc_tool_test.go`

**Deliverable:**
- `doc(action: "register")` response includes `classification_nudge` string
- Nudge text is ≤ 3 lines, includes document ID, references `guide` and `classify`
- Works for both single and batch registration (each result has its own nudge)
- Existing response fields unchanged

**Traceability:** FR-001, FR-002, FR-003, FR-004, NFR-001, NFR-002

### Task 2: Document batch classification workflow protocol

**Description:** Add a short batch classification protocol section to the
`doc_intel` skill or a reference document. The protocol describes: start with
`doc_intel(action: "pending")`, classify each document via `guide` → `classify`,
repeat until done. Include priority ordering by document type.

**Files:** `.agents/skills/kanbanzai-documents/SKILL.md` (or `refs/document-map.md`
if that is more appropriate — check existing conventions)

**Deliverable:**
- Protocol documented in appropriate skill/reference file
- Priority ordering present: specifications → designs → dev-plans → research/reports

**Traceability:** FR-005, FR-006, FR-007

---

## Dependency Graph

```
T1 — no prerequisites
T2 — no prerequisites
T1 and T2 are fully independent and may run in parallel.
```

---

## Interface Contracts

`doc(action: "register")` response gains one new optional string field:

```
{
  "document": { ... },           // unchanged
  "warnings": [...],             // unchanged
  "classification_nudge": "Layer 3 classification pending for {doc_id}. ..."
}
```

For batch registration the `documents` array contains per-document results each
with their own `classification_nudge`.

No other public interfaces are changed.

---

## Traceability Matrix

| Task | Requirements |
|------|-------------|
| T1   | FR-001, FR-002, FR-003, FR-004, NFR-001, NFR-002 |
| T2   | FR-005, FR-006, FR-007 |
