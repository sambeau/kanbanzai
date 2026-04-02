# Workflow State Automation — Specification

**Status:** Draft
**Feature:** FEAT-01KN73BFK4M4Z (auto-commit-and-doc-ops)
**Plan:** P18-workflow-automation
**Design:** [Workflow Automation Design](../design/workflow-automation-design.md)
**Date:** 2026-04-02

---

## Overview

This specification defines the requirements for automating workflow state persistence
and document lifecycle operations in the Kanbanzai MCP server. The work is motivated
by the [Automation Opportunities Analysis](../reports/automation-opportunities-analysis.md)
and structured around the four pillars described in the design document: parameterised
state commit, atomic document file operations, dev-plan approval decoupling, and
document section validation.

---

## Scope

### In scope

- New git commit helper functions for parameterised and multi-path commits
- Auto-commit behaviour in all state-mutating MCP tool handlers
- New `doc` tool actions: `move` and `delete`
- Modified `doc` tool behaviour: auto-refresh hash on `approve`, `auto_approve` flag
  on `register`, section validation on `register` and `approve`
- Removal of the dev-plan approval → `developing` cascade
- Skill and documentation updates reflecting the new workflow

### Out of scope

- Changes to the `handoff` tool's existing commit behaviour
- Changes to entity lifecycle state machine rules or transition validation
- Changes to the structure of `stage-bindings.yaml` (required sections are already
  declared there)
- UI, CLI, or non-MCP interfaces
- Retroactive validation of existing registered documents

---

## Functional Requirements

### Pillar A: Parameterised State Commit

**FR-A01:** The system MUST provide a function `CommitStateWithMessage(repoRoot, message)`
that stages all files under `.kbz/state/` and commits them with the caller-supplied
message.

**FR-A02:** The system MUST provide a function `CommitStateAndPaths(repoRoot, message,
extraPaths...)` that stages all files under `.kbz/state/` plus each path in `extraPaths`,
and commits them in a single git commit with the caller-supplied message.

**FR-A03:** `CommitStateAndPaths` MUST only stage paths explicitly listed in `extraPaths`.
It MUST NOT use globs, directory scans, or pattern matching to discover additional paths.

**FR-A04:** Both functions MUST return `(false, nil)` when there are no dirty files to
commit (no empty commits).

**FR-A05:** The existing `CommitStateIfDirty` function MUST continue to work with its
current signature and fixed message. It SHOULD delegate to `CommitStateWithMessage`
internally.

**FR-A06:** The `finish` tool MUST call `CommitStateWithMessage` after successful task
completion with a message in the format:
`workflow(TASK-{id}): complete – {summary truncated to 50 chars}`.

**FR-A07:** The `finish` tool in batch mode MUST produce a single commit after all tasks
are processed, not one commit per task.

**FR-A08:** The `decompose` tool's `apply` action MUST call `CommitStateWithMessage` after
both passes (task creation and dependency wiring) complete, with a message in the format:
`workflow(FEAT-{id}): decompose into {N} tasks`.

**FR-A09:** The `merge` tool's `execute` action MUST call `CommitStateWithMessage` after
the worktree record is updated to `merged` status, with a message in the format:
`workflow(FEAT-{id}): mark worktree merged`.

**FR-A10:** The `entity` tool's `create` action MUST call `CommitStateWithMessage` after
successful entity creation, with a message in the format:
`workflow({id}): create {type}`.

**FR-A11:** The `entity` tool's `transition` action MUST call `CommitStateWithMessage`
after a successful transition, with a message in the format:
`workflow({id}): transition {from} → {to}`.

**FR-A12:** The `doc` tool's `approve` action MUST call `CommitStateWithMessage` after
successful approval, with a message in the format:
`workflow({doc-id}): approve {type}`.

**FR-A13:** All auto-commits MUST be best-effort. If the commit fails, the tool handler
MUST log a warning and return the normal tool result. The commit failure MUST NOT
propagate as a tool error.

**FR-A14:** All auto-commits MUST scope staging to `.kbz/state/` (and explicit extra
paths where applicable). Files outside these paths MUST NOT be staged.

### Pillar B: Atomic Document File Operations

#### `doc register` enhancements

**FR-B01:** The `doc` tool's `register` action MUST call `CommitStateAndPaths` after
successful registration, passing the document file path as an extra path. The commit
message MUST follow the format: `workflow({doc-id}): register {type}`.

**FR-B02:** The `doc` tool's `register` action MUST accept an optional `auto_approve`
boolean parameter (default: `false`).

**FR-B03:** When `auto_approve` is `true`, the `register` action MUST:
  1. Write the document record with status `draft`
  2. Validate that the document type is in the auto-approve whitelist
  3. Set the record status to `approved`, record `approved_by` and `approved_at`
  4. Fire the entity lifecycle cascade (where applicable for the document type)
  5. Commit atomically via `CommitStateAndPaths`

**FR-B04:** The auto-approve whitelist MUST include exactly: `dev-plan`, `research`,
`report`. Attempting to auto-approve any other document type MUST return an error with
the message format: `auto_approve is not permitted for {type} documents`.

**FR-B05:** When `auto_approve` is `false` or omitted, the `register` action MUST behave
identically to its current behaviour (plus the auto-commit from FR-B01).

#### `doc approve` enhancements

**FR-B06:** The `doc` tool's `approve` action MUST recompute the content hash from the
file on disk before approving. If the hash differs from the stored hash, the stored hash
MUST be updated as part of the approval. The separate `doc refresh` step MUST NOT be
required before approval.

**FR-B07:** If the document file does not exist on disk at approval time, `doc approve`
MUST return an error indicating the file is missing.

**FR-B08:** The `doc refresh` action MUST remain available for use cases outside the
approval path (e.g., detecting drift on `doc get`).

#### `doc move` (new action)

**FR-B09:** The `doc` tool MUST support a `move` action with required parameters `id`
and `new_path`.

**FR-B10:** The `move` action MUST:
  1. Load the document record by ID
  2. Verify the file exists at the current stored path
  3. Move (rename) the file on disk to `new_path`
  4. Update the record's `path` field to `new_path`
  5. Recompute the content hash from the new file location
  6. Write the updated record

**FR-B11:** If the new path is in a directory associated with a different document type
(e.g., `work/research/` → `work/reports/`), the `move` action MUST update the record's
`type` field accordingly.

**FR-B12:** The `move` action MUST preserve the document's ID, approval status, owner,
`approved_by`, `approved_at`, and all cross-references.

**FR-B13:** The `move` action MUST call `CommitStateAndPaths` passing both the old path
and the new path as extra paths, producing a single atomic commit.

**FR-B14:** The `move` action MUST return an error if the source file does not exist on
disk.

**FR-B15:** The `move` action MUST return an error if no document record exists for the
given ID.

#### `doc delete` (new action)

**FR-B16:** The `doc` tool MUST support a `delete` action with required parameter `id`
and optional parameter `force` (default: `false`).

**FR-B17:** If the document status is `approved` and `force` is `false`, the `delete`
action MUST return an error explaining that approved documents require `force: true`
to delete.

**FR-B18:** The `delete` action MUST:
  1. Remove the document file from disk
  2. If the document has an `owner` entity, clear the entity's document reference
     field (e.g., `design`, `spec`, or `dev_plan`)
  3. Remove the state record file from `.kbz/state/documents/`
  4. Remove any corresponding index file from `.kbz/index/documents/`

**FR-B19:** The `delete` action MUST call `CommitStateAndPaths` passing the document
file path as an extra path, producing a single atomic commit.

**FR-B20:** The `delete` action MUST handle the case where the document file is already
missing on disk without error (the record and index cleanup should still proceed).

**FR-B21:** The `delete` action MUST return an error if no document record exists for
the given ID.

### Pillar C: Dev-Plan Approval Decoupling

**FR-C01:** Approving a document of type `dev-plan` MUST NOT trigger an automatic entity
lifecycle cascade (i.e., MUST NOT transition the owning feature from `dev-planning` to
`developing`).

**FR-C02:** Approving a document of type `design` MUST continue to cascade the owning
feature to `specifying` (or the owning plan to `active`), unchanged from current
behaviour.

**FR-C03:** Approving a document of type `specification` MUST continue to cascade the
owning feature to `dev-planning`, unchanged from current behaviour.

**FR-C04:** The transition from `dev-planning` to `developing` MUST require an explicit
`entity(action: transition, id: ..., status: "developing")` call. The existing stage gate
prerequisites (approved dev-plan and at least one child task) MUST remain enforced on this
transition.

**FR-C05:** When `auto_approve` is used with a `dev-plan` document (FR-B03), the entity
lifecycle cascade MUST NOT fire (consistent with FR-C01).

### Pillar D: Document Section Validation

**FR-D01:** The system MUST provide a section validation function that, given a markdown
file path and a document type, checks for the presence of required sections declared in
`stage-bindings.yaml` for that type.

**FR-D02:** Section detection MUST match level-2 markdown headings (`## Section Name`)
using case-insensitive comparison against the required section names.

**FR-D03:** Headings at other levels (`#`, `###`, `####`, etc.) MUST NOT be treated as
matches for required sections.

**FR-D04:** The `doc register` action MUST call section validation after writing the
record. Missing sections MUST be returned as structured warnings in the result. The
registration MUST proceed regardless of validation outcome.

**FR-D05:** The `doc approve` action MUST call section validation before setting the
status to `approved`. If required sections are missing, approval MUST be rejected with
an error listing the missing sections.

**FR-D06:** If a document type has no required sections declared in `stage-bindings.yaml`,
section validation MUST pass unconditionally.

**FR-D07:** The required section configuration MUST be read from `stage-bindings.yaml`
at server startup and cached. Changes to `stage-bindings.yaml` MUST take effect on
server restart.

---

## Non-Functional Requirements

**NFR-01:** Auto-commit latency MUST NOT noticeably degrade MCP tool response times. The
git operations (`add`, `commit`) are expected to complete in under 100ms for typical
`.kbz/state/` directory sizes (< 1000 files).

**NFR-02:** All new functions and tool handler changes MUST be covered by unit tests
that run against real (temporary) git repositories, following the existing pattern in
`internal/git/commit_test.go`.

**NFR-03:** All new tests MUST be safe for parallel execution (`t.Parallel()`).

**NFR-04:** The `CommitStateAndPaths` function MUST not introduce a risk of staging
unrelated files. The explicit-path-list design (FR-A03) is a hard constraint.

**NFR-05:** Backward compatibility: existing MCP tool behaviour MUST be preserved for
callers that do not use new parameters (`auto_approve`, `force`, etc.). The only
observable difference should be that state changes are now auto-committed.

**NFR-06:** Git history readability: auto-commit messages MUST follow the format
`workflow({entity-id}): {action description}` so that `git log --oneline` produces a
human-readable workflow audit trail.

---

## Acceptance Criteria

### Pillar A: Parameterised State Commit

- [ ] **AC-A01:** `CommitStateWithMessage` creates a commit with the supplied message
      when `.kbz/state/` has dirty files
- [ ] **AC-A02:** `CommitStateWithMessage` returns `(false, nil)` when `.kbz/state/`
      is clean
- [ ] **AC-A03:** `CommitStateAndPaths` creates a single commit containing both
      `.kbz/state/` files and the specified extra paths
- [ ] **AC-A04:** `CommitStateAndPaths` does not stage files outside `.kbz/state/` and
      the explicit extra paths
- [ ] **AC-A05:** `CommitStateIfDirty` continues to work unchanged (regression)
- [ ] **AC-A06:** `finish` (single task) creates a commit with the task ID and summary
      in the message
- [ ] **AC-A07:** `finish` (batch) creates a single commit for all completed tasks
- [ ] **AC-A08:** `decompose apply` creates a single commit containing all created task
      files and dependency links
- [ ] **AC-A09:** `merge execute` creates a state commit after the merge commit, with
      the worktree record update
- [ ] **AC-A10:** `entity create` creates a commit with the entity ID and type
- [ ] **AC-A11:** `entity transition` creates a commit with the from/to status
- [ ] **AC-A12:** `doc approve` creates a commit with the document ID and type
- [ ] **AC-A13:** All auto-commits: if git commit fails, the tool returns its normal
      result and logs a warning
- [ ] **AC-A14:** All auto-commits: files outside `.kbz/state/` (and listed extra paths)
      are not staged

### Pillar B: Atomic Document File Operations

- [ ] **AC-B01:** `doc register` creates a commit containing both the document file and
      its state record
- [ ] **AC-B02:** `doc register` with `auto_approve: true` and type `dev-plan` registers
      and approves the document in one call
- [ ] **AC-B03:** `doc register` with `auto_approve: true` and type `research` succeeds
- [ ] **AC-B04:** `doc register` with `auto_approve: true` and type `report` succeeds
- [ ] **AC-B05:** `doc register` with `auto_approve: true` and type `design` returns an
      error
- [ ] **AC-B06:** `doc register` with `auto_approve: true` and type `specification`
      returns an error
- [ ] **AC-B07:** `doc register` without `auto_approve` behaves as before (plus
      auto-commit)
- [ ] **AC-B08:** `doc approve` succeeds on a file that has been edited since
      registration, without requiring a prior `doc refresh`
- [ ] **AC-B09:** `doc approve` updates the stored content hash to match the current
      file
- [ ] **AC-B10:** `doc approve` returns an error when the file does not exist on disk
- [ ] **AC-B11:** `doc refresh` continues to work for non-approval use cases
- [ ] **AC-B12:** `doc move` updates the record's path field to the new path
- [ ] **AC-B13:** `doc move` preserves the document ID, approval status, owner, and
      cross-references
- [ ] **AC-B14:** `doc move` recomputes the content hash from the new file location
- [ ] **AC-B15:** `doc move` updates the document type when the new path implies a
      different type
- [ ] **AC-B16:** `doc move` produces a single commit containing the old path removal,
      the new path addition, and the state record update
- [ ] **AC-B17:** `doc move` returns an error if the source file does not exist
- [ ] **AC-B18:** `doc move` returns an error if no record exists for the given ID
- [ ] **AC-B19:** `doc delete` removes both the file and the state record
- [ ] **AC-B20:** `doc delete` of a draft document succeeds without `force`
- [ ] **AC-B21:** `doc delete` of an approved document returns an error without `force`
- [ ] **AC-B22:** `doc delete` of an approved document succeeds with `force: true`
- [ ] **AC-B23:** `doc delete` clears the owning entity's document reference field
- [ ] **AC-B24:** `doc delete` produces a single atomic commit
- [ ] **AC-B25:** `doc delete` succeeds when the document file is already missing on disk
- [ ] **AC-B26:** `doc delete` returns an error if no record exists for the given ID

### Pillar C: Dev-Plan Approval Decoupling

- [ ] **AC-C01:** Approving a dev-plan sets document status to `approved` but does NOT
      transition the owning feature
- [ ] **AC-C02:** Approving a design document still cascades the owning feature to
      `specifying`
- [ ] **AC-C03:** Approving a specification still cascades the owning feature to
      `dev-planning`
- [ ] **AC-C04:** Approving a plan-level design still cascades the plan to `active`
- [ ] **AC-C05:** `entity(action: transition, status: "developing")` succeeds when the
      feature has an approved dev-plan and at least one child task
- [ ] **AC-C06:** `entity(action: transition, status: "developing")` fails when the
      feature has no approved dev-plan
- [ ] **AC-C07:** `auto_approve` with a dev-plan does NOT cascade the feature

### Pillar D: Document Section Validation

- [ ] **AC-D01:** Document with all required sections passes validation
- [ ] **AC-D02:** Document missing one or more required sections returns them in a
      `missing_sections` list
- [ ] **AC-D03:** Section heading matching is case-insensitive
- [ ] **AC-D04:** Level-1 (`#`) and level-3+ (`###`) headings do not match level-2
      requirements
- [ ] **AC-D05:** `doc register` with missing sections succeeds but includes warnings
      in the response
- [ ] **AC-D06:** `doc approve` with missing sections returns an error listing the
      missing sections
- [ ] **AC-D07:** Document type with no declared required sections always passes
      validation

---

## Dependencies and Assumptions

### Dependencies

1. **`stage-bindings.yaml` required sections** — The section validation (Pillar D) reads
   the `document_template.required_sections` field from `stage-bindings.yaml`. This field
   must exist for any document type that should be validated. Currently declared for:
   `designing`, `specifying`, `dev-planning`.

2. **Entity lifecycle hook** — The `doc delete` action (FR-B18) uses the existing
   `EntityLifecycleHook` interface to clear entity document reference fields. This
   interface must support field clearing (setting a field to empty string).

3. **Git availability** — Auto-commit functions require `git` to be available on the
   system PATH and the working directory to be inside a valid git repository. This is
   already a requirement for the existing `CommitStateIfDirty` function.

### Assumptions

1. **Single-process MCP server.** The auto-commit functions assume a single MCP server
   process interacts with the git repository at any given time. Concurrent access from
   multiple processes may cause git lock contention, which is handled by the best-effort
   semantics (FR-A13) but not prevented.

2. **Existing commit message convention.** The `workflow({id}): {description}` message
   format introduced by this specification is new. It does not conflict with the existing
   `chore(kbz): persist workflow state before sub-agent dispatch` message used by
   `handoff`, which remains unchanged.

3. **Document type directories.** The `doc move` type inference (FR-B11) assumes a
   stable mapping from directory paths to document types (e.g., `work/design/` → `design`,
   `work/reports/` → `report`). This mapping must be defined or derivable from the
   existing document registration conventions.

4. **Skill documents are prose.** Task 12 (skill and documentation updates) produces
   prose changes that are verified by manual review, not automated tests. All other
   tasks produce testable code changes.