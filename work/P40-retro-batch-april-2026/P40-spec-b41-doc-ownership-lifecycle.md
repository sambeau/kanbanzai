# Specification: Fix Document Ownership and Lifecycle

| Field  | Value                         |
|--------|-------------------------------|
| Date   | 2026-04-30                    |
| Status | approved |
| Author | Spec Author                   |
| Batch  | B41-fix-doc-ownership-lifecycle |
| Design | P40-retro-batch-april-2026/design-p40-design-retro-batch-improvements |

---

## Problem Statement

This specification implements Workstream C of the design described in
`work/P40-retro-batch-april-2026/P40-design-retro-batch-improvements.md`
(P40-retro-batch-april-2026/design-p40-design-retro-batch-improvements, approved).

Three retrospective reports (P38 implementation, B36 review fixes, and P37
branch audit) document friction in the document registration and lifecycle
system. Documents registered inside a plan or batch folder default to `PROJECT/`
ownership, causing `decompose` and feature-level lookups to fail. The
`doc refresh` command resets approval status on every content change, forcing
re-approval for minor formatting edits. The merge gate system surfaces gate
failures but doesn't distinguish between override-able soft gates and hard
gates until execution time. Features cannot record post-hoc verification
outside the `finish` task flow, forcing gate overrides for rebase
re-verification.

This specification addresses four fixes from the design's Workstream C:

1. **C1 — Auto-infer document owner from path context:** When registering a
   document, infer the owner from the file path rather than defaulting to
   `PROJECT/`. Warn when a path is already registered under a different owner.
2. **C2 — Preserve approval status on minor doc edits:** `doc refresh` should
   not reset approval when only formatting changes are detected. At minimum,
   warn before resetting.
3. **C3 — Add `bypassable` field to merge gate results:** Each gate result in
   `merge(action: check)` output gains a `bypassable: bool` so callers know
   before execution which gates are hard stops.
4. **C4 — Add verification parameter to entity update:** `entity(action: update)`
   gains `verification` and `verification_status` parameters so post-hoc
   verification can be recorded outside the `finish` task flow.

**Scope inclusion:** The `doc` tool's register and refresh handlers, the `merge`
tool's check output format, and the `entity` tool's update handler.

**Scope exclusion:** The `decompose` tool's document lookup (that already works
when owner is correct — C1 fixes the owner so decompose finds the doc). The
`merge execute` flow (only the `check` output format changes). The `doc approve`
false-positive code path unification (documented as follow-up). Per-feature
spec requirements (batch-level specs are the established pattern for P40).

---

## Requirements

### Functional Requirements

- **REQ-001:** When `doc(action: register, path: "work/{plan-or-batch-slug}/...", ...)`
  is called without an explicit `owner` parameter, the tool MUST extract the
  plan or batch slug from the path and attempt to resolve it to an entity. If
  an entity is found, it MUST be used as the default owner instead of `PROJECT/`.

- **REQ-002:** When `doc(action: register, path: "...")` is called and the
  resolved owner differs from an existing registration at the same path, the
  tool MUST emit a warning: "This path is already registered under {existing_owner}.
  Did you mean owner: {resolved_owner}?"

- **REQ-003:** When no entity can be resolved from the path components in
  `doc(action: register)`, the tool MUST fall back to `PROJECT/` ownership
  (current behaviour). No error is raised for unresolvable paths.

- **REQ-004:** When `doc(action: refresh, id: "...")` detects a content hash
  change, the tool MUST evaluate the scope of the change. If the change is
  limited to formatting or whitespace only, the approval status MUST be
  preserved.

- **REQ-005:** When `doc(action: refresh, id: "...")` detects a substantive
  content change (beyond formatting), the tool MUST warn the caller before
  resetting approval status: "Refreshing will reset approval status from
  approved to draft. Continue?" If the caller confirms, the status resets;
  if not, the refresh is aborted with no change.

- **REQ-006:** Each gate result object in `merge(action: check)` output MUST
  include a `bypassable: bool` field. Hard gates that cannot be overridden
  with `override: true` (e.g., `review_report_exists`) MUST have
  `bypassable: false`. Soft gates that accept `override: true` MUST have
  `bypassable: true`.

- **REQ-007:** `entity(action: update)` MUST accept two new optional parameters:
  `verification` (string) and `verification_status` (string, values: "passed"
  or "failed"). When provided, these MUST be written directly to the entity
  record without triggering any lifecycle transition.

- **REQ-008:** When `verification` or `verification_status` is provided to
  `entity(action: update)` for an entity type that does not support
  verification fields (e.g., plans, batches), the tool MUST return a clear
  error indicating which entity types support verification.

### Non-Functional Requirements

- **REQ-NF-001:** Owner inference from path MUST NOT add measurable latency to
  doc registration — the entity lookup must be a single cache or store read,
  not a filesystem scan.

- **REQ-NF-002:** Adding `bypassable` to merge gate results MUST NOT change
  the behaviour of `merge(action: execute)` — the field is informational only
  in the check output.

- **REQ-NF-003:** The `verification` and `verification_status` parameters on
  `entity update` MUST be fully backward-compatible — existing `entity update`
  calls without these parameters must continue to work identically.

---

## Constraints

- The `doc` tool's existing parameter surface is preserved. No parameters are
  removed or renamed. Owner inference is a default, not a requirement —
  explicit `owner` always takes precedence.
- The `merge` tool's check/execute lifecycle is unchanged. The `bypassable`
  field is additive to the check output only.
- The `entity` tool's update handler already supports optional field updates.
  `verification` and `verification_status` follow the same pattern as existing
  optional fields.
- This specification does NOT change the `doc approve` gate logic or the
  `record_false_positive` code path (documented as follow-up from Theme 11).
- This specification does NOT change how `decompose` searches for specs
  (C1 fixes the owner so existing search logic finds the doc).

---

## Acceptance Criteria

- **AC-001 (REQ-001):** Given a document at
  `work/B40-fix-tool-correctness/some-file.md`, when `doc(action: register,
  path: "work/B40-fix-tool-correctness/some-file.md", type: "specification")`
  is called without an `owner` parameter, then the document is registered with
  owner `B40-fix-tool-correctness`, not `PROJECT/`.

- **AC-002 (REQ-001):** Given a document at
  `work/B40-fix-tool-correctness/some-file.md`, when `doc(action: register,
  path: ..., owner: "PROJECT")` is called with an explicit owner, then the
  explicit owner is used — inference does not override explicit input.

- **AC-003 (REQ-002):** Given a document already registered at
  `work/B40-fix-tool-correctness/existing.md` under owner `PROJECT/`, when
  `doc(action: register, path: "work/B40-fix-tool-correctness/existing.md")`
  is called, then the response includes a warning that the path is already
  registered under `PROJECT/`.

- **AC-004 (REQ-003):** Given a document at `work/random-folder/doc.md` where
  no entity matches the path components, when `doc(action: register, path: ...)`
  is called, then the document is registered with owner `PROJECT/` — fallback
  behaviour is preserved.

- **AC-005 (REQ-004):** Given an approved document, when a whitespace-only
  change is made and `doc(action: refresh)` is called, then the document
  remains in `approved` status.

- **AC-006 (REQ-005):** Given an approved document, when a substantive content
  change is made and `doc(action: refresh)` is called, then the tool warns
  that approval will be reset and the status transitions to `draft`.

- **AC-007 (REQ-006):** Given a feature ready to merge, when
  `merge(action: check)` is called, then each gate result includes a
  `bypassable` field. The `review_report_exists` gate has `bypassable: false`;
  typical gates like `all_tasks_done` have `bypassable: true`.

- **AC-008 (REQ-007):** Given a feature in `developing` status, when
  `entity(action: update, id: "FEAT-...", verification: "Rebased and re-tested",
  verification_status: "passed")` is called, then the feature's verification
  fields are updated and the feature remains in `developing` — no lifecycle
  transition occurs.

- **AC-009 (REQ-007):** Given a feature with verification fields set via
  `entity update`, when `merge(action: check)` is subsequently called, then
  the `verification_exists` and `verification_passed` gates pass without
  requiring an override.

- **AC-010 (REQ-008):** Given a batch entity, when `entity(action: update,
  id: "B41-...", verification: "test", verification_status: "passed")` is
  called, then the tool returns an error indicating that verification fields
  are not supported for batches.

- **AC-011 (REQ-NF-001):** Given a doc registration with owner inference,
  when the latency is measured, then the entity lookup adds no more than one
  cache read to the registration path.

- **AC-012 (REQ-NF-003):** Given an existing `entity update` call with no
  verification parameters, when the call executes against the modified handler,
  then behaviour is identical to the pre-change implementation.

---

## Verification Plan

| Criterion | Method | Description |
|-----------|--------|-------------|
| AC-001 | Test | Automated test: register a document under a batch folder, assert owner is the batch ID not PROJECT/ |
| AC-002 | Test | Automated test: register with explicit owner, assert explicit owner is used |
| AC-003 | Test | Automated test: register a path already registered under different owner, assert warning in response |
| AC-004 | Test | Automated test: register under unresolvable path, assert fallback to PROJECT/ |
| AC-005 | Test | Automated test: approve doc, make whitespace change, refresh, assert status stays approved |
| AC-006 | Test | Automated test: approve doc, make content change, refresh, assert status transitions to draft |
| AC-007 | Test | Automated test: call merge check on a mergeable feature, assert bypassable field present on all gates |
| AC-008 | Test | Automated test: update feature verification fields, assert fields set and no lifecycle transition |
| AC-009 | Test | Integration test: set verification via update, call merge check, assert verification gates pass |
| AC-010 | Test | Automated test: update batch with verification fields, assert error |
| AC-011 | Inspection | Code review: confirm owner inference uses single cache/store lookup, no filesystem walk |
| AC-012 | Test | Run existing entity update tests; all pass without modification |
