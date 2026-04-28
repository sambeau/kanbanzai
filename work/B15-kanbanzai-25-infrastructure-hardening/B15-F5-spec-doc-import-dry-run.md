# Specification: `doc import` Dry-Run Mode

| Field   | Value                                                              |
|---------|--------------------------------------------------------------------|
| Status  | Draft                                                              |
| Created | 2026-04-01                                                         |
| Updated | 2026-04-01                                                         |
| Feature | FEAT-01KN4ZPTQSZT5 (doc-import-dry-run)                           |
| Design  | `work/design/kanbanzai-2.5-infrastructure-hardening.md` §8        |

---

## 1. Purpose

This specification defines the requirements for a dry-run mode on the `doc import`
action. The dry-run mode allows agents to preview what would be registered —
including inferred types, titles, and owners — before committing any changes to
the document store. It addresses the observed pattern of agents avoiding `doc
import` in favour of explicit batch registration because the import's inference
behaviour is opaque.

---

## 2. Goals

1. The `doc import` action accepts a `dry_run` boolean parameter.
2. When `dry_run` is `true`, the full inference pipeline runs but no document
   records are written to the store.
3. The response faithfully represents what a non-dry-run import would produce,
   including all inferred metadata.
4. Files that would be skipped are reported with their skip reasons.
5. When `dry_run` is `false` or omitted, the existing import behaviour is
   unchanged.

---

## 3. Scope

### 3.1 In Scope

- The `dry_run` boolean parameter on the `doc import` action.
- Dry-run response format: `would_import`, `would_skip`, and `summary` fields.
- Inference of type, title, and owner for each file that would be imported.
- Reporting of files that would be skipped and their reasons.
- Preservation of existing non-dry-run import behaviour.

### 3.2 Out of Scope

- Changes to the type-inference or title-extraction logic itself (that logic is
  shared with the live import path and is not modified by this feature).
- Any new `doc import` parameters other than `dry_run`.
- Dry-run modes on other `doc` actions (`register`, `approve`, `audit`, etc.).
- Persisting or caching dry-run results.

---

## 4. Requirements

### 4.1 Parameter

**REQ-01.** The `doc import` action MUST accept a `dry_run` boolean parameter.

**REQ-02.** When `dry_run` is `false` or omitted, the action MUST behave
identically to the current implementation — no behaviour change.

**REQ-03.** The `dry_run` parameter MUST be optional and MUST default to `false`.

### 4.2 Dry-Run Execution

**REQ-04.** When `dry_run` is `true`, the import action MUST execute the full
inference pipeline: directory walking, type inference, title extraction, and
owner inference.

**REQ-05.** When `dry_run` is `true`, the action MUST NOT create, update, or
delete any document records in the store.

**REQ-06.** When `dry_run` is `true`, the action MUST NOT modify any files on
disk.

**REQ-07.** The dry-run execution MUST use the same directory-walking logic,
type-inference rules, title-extraction rules, and owner-inference rules as the
live import path.

### 4.3 Response Format

**REQ-08.** When `dry_run` is `true`, the response MUST include a `would_import`
array. Each entry MUST contain:

- `path` — the relative file path of the document that would be registered.
- `type` — the inferred document type.
- `title` — the inferred document title.
- `owner` — the inferred owner (empty string if none inferred).

**REQ-09.** When `dry_run` is `true`, the response MUST include a `would_skip`
array. Each entry MUST contain:

- `path` — the relative file path of the document that would be skipped.
- `reason` — a human-readable explanation of why the file would be skipped (e.g.,
  `"already registered"`).

**REQ-10.** When `dry_run` is `true`, the response MUST include a `summary`
object containing:

- `would_import` — integer count of files that would be registered.
- `would_skip` — integer count of files that would be skipped.

**REQ-11.** The `would_import` count in the summary MUST equal the length of the
`would_import` array. The `would_skip` count MUST equal the length of the
`would_skip` array.

### 4.4 Already-Registered Files

**REQ-12.** A file that already has a document record in the store MUST appear in
`would_skip` with reason `"already registered"`, consistent with the live import
path's skip behaviour.

### 4.5 Consistency with Live Import

**REQ-13.** For any given directory and store state, the set of files in
`would_import` (in a dry run) MUST be identical to the set of files that a
live import of the same directory would actually register, assuming no store
changes occur between the two calls.

---

## 5. Acceptance Criteria

**AC-21.** `doc(action: "import", path: "work/", dry_run: true)` returns the list
of files that would be imported with their inferred metadata.

**AC-22.** In dry-run mode, no document records are created in the store.

**AC-23.** Each entry in `would_import` includes the inferred type, title, and
owner.

**AC-24.** Files that would be skipped (already registered) are listed in
`would_skip` with the reason `"already registered"`.

**AC-25.** When `dry_run` is `false` or omitted, behaviour is unchanged from the
current implementation.

---

## 6. Dependencies and Assumptions

- The existing `doc import` directory-walking and inference logic is correct and
  reusable. This feature adds a commit-suppression mode on top of it; it does not
  rewrite the inference pipeline.
- The document store exposes a sufficient interface to check whether a given path
  is already registered, used by both the live import and the dry-run path.
- The `path` parameter scoping behaviour of `doc import` is unchanged.

---

## 7. Non-Requirements

The following are explicitly not required by this specification:

- The dry-run response need not be identical in field order or whitespace to the
  live import response — only the logical content must match.
- Dry-run mode need not be transactionally consistent with concurrent store
  writes; it is a best-effort preview.
- Dry-run mode need not support the `glob` parameter if that parameter does not
  exist on the live import path.