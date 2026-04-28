# Workflow State Automation — Implementation Plan

**Status:** Draft
**Feature:** FEAT-01KN73BFK4M4Z (auto-commit-and-doc-ops)
**Plan:** P18-workflow-automation
**Design:** [Workflow Automation Design](../design/workflow-automation-design.md)
**Specification:** [Workflow Automation Spec](../spec/workflow-automation-spec.md)
**Date:** 2026-04-02

---

## Overview

This plan implements the requirements defined in
`work/spec/workflow-automation-spec.md` (FEAT-01KN73BFK4M4Z/specification-workflow-automation-spec).
It covers all four pillars of the specification:

- **Pillar A** — Parameterised state commit: new git commit helpers and auto-commit
  call sites in all state-mutating MCP tool handlers (FR-A01 through FR-A14)
- **Pillar B** — Atomic document file operations: `doc move`, `doc delete`, auto-refresh
  on approve, `auto_approve` flag on register, auto-commit with extra paths
  (FR-B01 through FR-B21)
- **Pillar C** — Dev-plan approval decoupling: remove the dev-plan approval cascade
  (FR-C01 through FR-C05)
- **Pillar D** — Document section validation: lint on register, gate on approve
  (FR-D01 through FR-D07)

**Out of scope:**

- Changes to the `handoff` tool (continues to use `CommitStateIfDirty`)
- Changes to entity lifecycle state machine rules or transition validation
- Changes to `stage-bindings.yaml` structure (required sections already declared)
- UI, CLI, or non-MCP interfaces
- Retroactive validation of existing registered documents

---

## Task Breakdown

### Task 1: `CommitStateWithMessage` and `CommitStateAndPaths`

**Objective:** Create the foundational parameterised commit functions that all
auto-commit call sites depend on.

**Specification references:** FR-A01, FR-A02, FR-A03, FR-A04, FR-A05

**Input context:**
- `internal/git/commit.go` — existing `CommitStateIfDirty`, `runGitCmd`, `stateDir`
  constant
- `internal/git/commit_test.go` — existing test patterns using real temp git repos
- Design §Pillar A — commit semantics and failure mode

**Output artifacts:**
- Modified `internal/git/commit.go` with two new exported functions
- Modified `internal/git/commit_test.go` with tests for both functions

**Changes required:**

1. Add `CommitStateWithMessage(repoRoot, message string) (bool, error)` — identical to
   `CommitStateIfDirty` but accepts caller-supplied message instead of the fixed
   `stateCommitMessage` constant.

2. Add `CommitStateAndPaths(repoRoot, message string, extraPaths ...string) (bool, error)`:
   - Check for dirty files under `.kbz/state/` OR any of `extraPaths` via
     `git status --porcelain -- .kbz/state/ <extraPaths...>`
   - Stage `.kbz/state/` via `git add -- .kbz/state/`
   - Stage each extra path via `git add -- <path>` (one `git add` per path)
   - Commit with the supplied message
   - Return `(false, nil)` if nothing was dirty

3. Refactor `CommitStateIfDirty` to delegate to `CommitStateWithMessage` with the
   existing `stateCommitMessage` constant. This preserves backward compatibility for
   the `handoff` call site.

**Dependencies:** None (foundation task).

**Effort:** Small

**Tests:**
- `CommitStateWithMessage` with custom message appears in `git log --oneline`
- `CommitStateAndPaths` stages both `.kbz/state/` and extra paths in one commit
- `CommitStateAndPaths` with extra path outside `.kbz/state/` includes it
- `CommitStateAndPaths` with no dirty files returns `(false, nil)`
- `CommitStateAndPaths` does not stage files not in `extraPaths`
- `CommitStateIfDirty` continues to work unchanged (regression)

---

### Task 2: Auto-commit in `finish` handler

**Objective:** The `finish` tool commits all state changes atomically after successful
task completion.

**Specification references:** FR-A06, FR-A07, FR-A13, FR-A14

**Input context:**
- `internal/mcp/finish_tool.go` — `finishOne` function, batch mode via `finishBatch`
- `internal/mcp/handoff_tool.go` — existing `commitStateFunc` pattern (package-level
  var for test injection)
- `internal/git/commit.go` — `CommitStateWithMessage` (from Task 1)

**Output artifacts:**
- Modified `internal/mcp/finish_tool.go` with commit call after task completion
- Modified or new `internal/mcp/finish_tool_test.go` with commit tests

**Changes required:**

1. Add a package-level `var finishCommitFunc` (same pattern as `handoff_tool.go`'s
   `commitStateFunc`) defaulting to `git.CommitStateWithMessage`.

2. In `finishOne`, after `dispatchSvc.CompleteTask()` returns successfully, call
   `finishCommitFunc(repoRoot, message)` with message format:
   `workflow(TASK-{id}): complete – {summary truncated to 50 chars}`

3. For batch mode, commit once after all tasks are processed (not per task). Message:
   `workflow: complete {N} tasks`

4. Best-effort: if commit fails, log via `log.Printf` and return the normal result.

5. Derive `repoRoot` from `entitySvc.Root()` (which returns `.kbz/` root) by taking
   its parent directory, or pass it through from the tool constructor.

**Dependencies:** Task 1

**Effort:** Small

**Interface contract with Task 1:** Uses `CommitStateWithMessage(repoRoot, message)`.

**Tests:**
- Single task completion creates a git commit with expected message format
- Batch completion creates a single commit (not one per task)
- Commit failure does not prevent `finish` from returning the result
- All state files (task, knowledge, retro) are included in the commit

---

### Task 3: Auto-commit in `decompose apply` handler

**Objective:** `decompose apply` commits all created task files atomically after both
passes (task creation and dependency wiring).

**Specification references:** FR-A08, FR-A13, FR-A14

**Input context:**
- `internal/mcp/decompose_tool.go` — `decomposeApply` function, two-pass structure
- `internal/git/commit.go` — `CommitStateWithMessage` (from Task 1)

**Output artifacts:**
- Modified `internal/mcp/decompose_tool.go` with commit call after apply
- Modified or new `internal/mcp/decompose_tool_test.go` with commit tests

**Changes required:**

1. After Pass 2 (dependency wiring) completes in `decomposeApply`, call
   `CommitStateWithMessage` with message:
   `workflow(FEAT-{id}): decompose into {N} tasks`

2. Best-effort semantics (same pattern as Task 2).

3. Derive `repoRoot` from `entitySvc` — same approach as Task 2.

**Dependencies:** Task 1

**Effort:** Small

**Interface contract with Task 1:** Uses `CommitStateWithMessage(repoRoot, message)`.

**Tests:**
- `decompose apply` creates a commit containing all N task entity files
- Dependency wiring (Pass 2) is included in the same commit
- Commit failure does not block the decomposition result

---

### Task 4: Auto-commit in `merge execute` handler

**Objective:** `merge execute` commits the worktree state update after a successful
merge.

**Specification references:** FR-A09, FR-A13, FR-A14

**Input context:**
- `internal/mcp/merge_tool.go` — `executeMerge` function, already receives `repoPath`
  parameter; `postMergeInstall` hook runs after
- `internal/git/commit.go` — `CommitStateWithMessage` (from Task 1)
- Design §Pillar A — merge execute commit semantics

**Output artifacts:**
- Modified `internal/mcp/merge_tool.go` with commit call after worktree record update
- Modified or new `internal/mcp/merge_tool_test.go` with commit tests

**Changes required:**

1. After `worktreeStore.Update()` marks the worktree as `merged` (and after any branch
   deletion), call `CommitStateWithMessage(repoPath, message)` with message:
   `workflow(FEAT-{id}): mark worktree merged`

2. This must happen after the git merge and worktree record update — the merge itself
   is already a git commit; this is the follow-up state commit on main.

3. `repoPath` is already available as a parameter to `executeMerge`.

4. Best-effort semantics.

**Dependencies:** Task 1

**Effort:** Small

**Interface contract with Task 1:** Uses `CommitStateWithMessage(repoPath, message)`.

**Tests:**
- Successful merge creates a state commit after the merge commit
- Worktree record file is included in the state commit
- Commit failure does not prevent merge result from being returned

---

### Task 5: Auto-commit in `entity` and `doc` handlers

**Objective:** Add auto-commit to the remaining state-mutating tool handlers: entity
create, entity transition, doc register, and doc approve.

**Specification references:** FR-A10, FR-A11, FR-A12, FR-A13, FR-A14, FR-B01

**Input context:**
- `internal/mcp/entity_tool.go` — `entityCreateAction` (with `entityCreateOne`),
  `entityTransitionAction`, batch mode via `ExecuteBatch`
- `internal/mcp/doc_tool.go` — `docRegisterAction`, `docApproveAction`,
  `WithSideEffects` wrapper, `SignalMutation` pattern
- `internal/git/commit.go` — `CommitStateWithMessage`, `CommitStateAndPaths` (Task 1)

**Output artifacts:**
- Modified `internal/mcp/entity_tool.go` with commit calls
- Modified `internal/mcp/doc_tool.go` with commit calls
- Corresponding test file updates

**Changes required:**

For `entity_tool.go`:

1. `entityCreateAction`: after `entitySvc.Create*()` succeeds, commit with:
   `workflow({ID}): create {type}`
2. `entityTransitionAction`: after `entitySvc.UpdateStatus()` succeeds, commit with:
   `workflow({ID}): transition {from} → {to}`
3. For batch entity create, commit once after all items.

For `doc_tool.go`:

1. `docRegisterAction`: after `docSvc.SubmitDocument()` succeeds, commit with
   `CommitStateAndPaths(repoRoot, message, documentFilePath)`:
   `workflow({doc-id}): register {type}`
2. `docApproveAction`: after `docSvc.ApproveDocument()` succeeds, commit with:
   `workflow({doc-id}): approve {type}`

All commits are best-effort.

**Dependencies:** Task 1

**Effort:** Medium

**Interface contracts:**
- Entity create/transition use `CommitStateWithMessage(repoRoot, message)`
- Doc register uses `CommitStateAndPaths(repoRoot, message, docFilePath)` — this is the
  first call site that stages a document file alongside state
- Doc approve uses `CommitStateWithMessage(repoRoot, message)`

**Tests:**
- Entity create produces a commit with entity ID and type in message
- Entity transition produces a commit with from/to status in message
- Batch entity create produces a single commit
- Doc register produces a commit containing both state record and document file
- Doc approve produces a commit (including cascaded entity transition files)
- All commits are best-effort (inject failure, verify normal result returned)

---

### Task 6: Auto-refresh hash on `doc approve`

**Objective:** Eliminate the manual `doc refresh` step before `doc approve` by
recomputing the content hash at approval time.

**Specification references:** FR-B06, FR-B07, FR-B08

**Input context:**
- `internal/service/documents.go` — `ApproveDocument` method, specifically the content
  hash verification block that currently rejects on mismatch
- `internal/storage/document_store.go` — `ComputeContentHash` function
- `internal/service/documents_test.go` — existing approval tests

**Output artifacts:**
- Modified `internal/service/documents.go`
- Modified `internal/service/documents_test.go`

**Changes required:**

1. In `ApproveDocument`, replace the hash-mismatch rejection logic with: compute hash
   from file on disk → if different from stored hash, update stored hash → proceed
   with approval.

2. If the file does not exist on disk, return an error (this is a real problem, not
   hash drift).

3. `doc refresh` remains available and unchanged for non-approval use cases.

**Dependencies:** None (independent service-layer change).

**Effort:** Small

**Tests:**
- Approve succeeds when file has been edited since registration (no refresh needed)
- Stored hash is updated to match current file content after approval
- Approve fails when file does not exist on disk
- `doc refresh` still works independently (regression)

---

### Task 7: Remove dev-plan approval cascade

**Objective:** Approving a dev-plan no longer triggers an automatic feature transition
to `developing`.

**Specification references:** FR-C01, FR-C02, FR-C03, FR-C04, FR-C05

**Input context:**
- `internal/service/documents.go` — the approval cascade switch block (~L409–433):
  ```
  case entityType == "feature" && doc.Type == model.DocumentTypeDevPlan:
      targetStatus = "developing"
  ```
- `internal/service/documents_test.go` — existing cascade tests
- Design §Pillar C — what cascades are preserved vs removed

**Output artifacts:**
- Modified `internal/service/documents.go`
- Modified `internal/service/documents_test.go`

**Changes required:**

1. Remove (or skip) the `DocumentTypeDevPlan` case from the approval cascade switch.
   The approval still writes `approved` status. Only the entity transition is removed.

2. The other cascades remain unchanged:
   - plan + design → `active`
   - feature + design → `specifying`
   - feature + specification → `dev-planning`

3. The transition from `dev-planning` → `developing` now requires an explicit
   `entity(action: transition, status: "developing")` call. The existing stage gate
   (approved dev-plan + at least one child task) remains enforced.

**Dependencies:** None (independent service-layer change).

**Effort:** Small

**Tests:**
- Approving a dev-plan sets document status to `approved` but does NOT transition the
  owning feature
- Approving a design still cascades feature to `specifying` (regression)
- Approving a spec still cascades feature to `dev-planning` (regression)
- Approving a plan design still cascades plan to `active` (regression)

---

### Task 8: `doc move` action

**Objective:** Add an atomic document move action to the `doc` tool that relocates the
file, updates the record, and commits in a single operation.

**Specification references:** FR-B09, FR-B10, FR-B11, FR-B12, FR-B13, FR-B14, FR-B15

**Input context:**
- `internal/mcp/doc_tool.go` — `DispatchAction` map, `ActionHandler` pattern,
  `SignalMutation`, `docRecordToMap`
- `internal/service/documents.go` — `DocumentService`, `DocumentStore.Write()`,
  `ComputeContentHash`
- `internal/storage/document_store.go` — `Load`, `Write`, `Exists` methods
- `internal/git/commit.go` — `CommitStateAndPaths` (from Task 1)

**Output artifacts:**
- Modified `internal/service/documents.go` — new `MoveDocument` method
- Modified `internal/mcp/doc_tool.go` — new `"move"` action handler
- New or modified test files for both service and tool

**Changes required:**

1. Add `MoveDocument(input MoveDocumentInput) (DocumentResult, error)` to
   `DocumentService`. Input: `ID string`, `NewPath string`.

2. Implementation:
   - Load record by ID → error if not found (FR-B15)
   - Verify file exists at current `path` → error if missing (FR-B14)
   - `os.Rename(oldPath, newPath)` — move the file on disk
   - Update record's `path` field to `newPath`
   - If new path implies a different document type directory, update `type` (FR-B11)
   - Recompute content hash from new path
   - Write updated record via `DocumentStore.Write()`
   - Preserve ID, approval status, owner, `approved_by`, `approved_at`,
     cross-references (FR-B12)

3. Add `"move"` to the `DispatchAction` map in `doc_tool.go`. Parameters: `id`
   (required), `new_path` (required).

4. After the service call, commit via `CommitStateAndPaths(repoRoot, message,
   oldPath, newPath)` to produce a single atomic commit (FR-B13).

**Dependencies:** Task 1 (for `CommitStateAndPaths`)

**Effort:** Medium

**Interface contract with Task 1:** Uses `CommitStateAndPaths(repoRoot, message,
oldPath, newPath)` — two extra paths (old location for deletion, new location for
addition).

**Tests:**
- Move updates the record's `path` field
- Move preserves document ID, approval status, owner, and cross-references
- Move recomputes content hash from new location
- Move updates document type when new path is in a different type directory
- Move produces a single atomic commit containing both paths and state record
- Move returns error if source file does not exist
- Move returns error if document ID is not found

---

### Task 9: `doc delete` action

**Objective:** Add an atomic document deletion action to the `doc` tool that removes
the file, cleans up state and index records, clears entity references, and commits
atomically.

**Specification references:** FR-B16, FR-B17, FR-B18, FR-B19, FR-B20, FR-B21

**Input context:**
- `internal/mcp/doc_tool.go` — action dispatch pattern
- `internal/service/documents.go` — `DocumentService`, `EntityLifecycleHook` interface
  (specifically `SetDocumentRef` for clearing entity references)
- `internal/storage/document_store.go` — record file locations under `.kbz/state/documents/`
- `.kbz/index/documents/` — index files that must also be cleaned up
- `internal/git/commit.go` — `CommitStateAndPaths` (from Task 1)

**Output artifacts:**
- Modified `internal/service/documents.go` — new `DeleteDocument` method
- Modified `internal/mcp/doc_tool.go` — new `"delete"` action handler
- New or modified test files for both service and tool

**Changes required:**

1. Add `DeleteDocument(input DeleteDocumentInput) (DocumentResult, error)` to
   `DocumentService`. Input: `ID string`, `Force bool`.

2. Implementation:
   - Load record by ID → error if not found (FR-B21)
   - If status is `approved` and `Force` is false → return error (FR-B17)
   - `os.Remove(filePath)` — remove file, ignore "not exists" error (FR-B20)
   - If record has `owner`, clear the entity's document ref field via
     `entityHook.SetDocumentRef(owner, docField, "")` (FR-B18 item 2)
   - Remove state record file from `.kbz/state/documents/`
   - Remove index file from `.kbz/index/documents/` if it exists

3. Add `"delete"` to the `DispatchAction` map. Parameters: `id` (required), `force`
   (optional, default `false`).

4. Commit via `CommitStateAndPaths(repoRoot, message, filePath)` (FR-B19).

**Dependencies:** Task 1 (for `CommitStateAndPaths`)

**Effort:** Medium

**Interface contract with Task 1:** Uses `CommitStateAndPaths(repoRoot, message,
filePath)` — one extra path (document file, staged as a deletion).

**Interface contract with entity lifecycle:** Uses existing `EntityLifecycleHook.SetDocumentRef(entityID, docField, "")` to clear references. The `docField` is derived from the document type: `design` → `"design"`, `specification` → `"spec"`, `dev-plan` → `"dev_plan"`.

**Tests:**
- Delete removes both the file and the state record
- Delete of a draft document succeeds without `force`
- Delete of an approved document returns error without `force`
- Delete of an approved document succeeds with `force: true`
- Delete clears the owning entity's document reference field
- Delete produces a single atomic commit
- Delete succeeds when file is already missing on disk (record cleanup still proceeds)
- Delete returns error if document ID is not found

---

### Task 10: `auto_approve` flag on `doc register`

**Objective:** Allow agent-authored documents to be registered and approved in a single
call, restricted to safe document types.

**Specification references:** FR-B02, FR-B03, FR-B04, FR-B05

**Input context:**
- `internal/mcp/doc_tool.go` — `docRegisterAction`, parameter parsing
- `internal/service/documents.go` — `SubmitDocument` method, `ApproveDocument` method,
  approval cascade logic
- Task 6 result — auto-refresh on approve (hash is recomputed, no refresh needed)
- Task 7 result — dev-plan cascade removed (auto-approve of dev-plan is safe)

**Output artifacts:**
- Modified `internal/service/documents.go` — enhanced `SubmitDocument` or new method
- Modified `internal/mcp/doc_tool.go` — new `auto_approve` parameter
- Corresponding test updates

**Changes required:**

1. Add `AutoApprove bool` field to the register input.

2. In the registration flow, after writing the initial record, if `AutoApprove` is true:
   - Validate type is in whitelist: `dev-plan`, `research`, `report` (FR-B04)
   - If not in whitelist → return error: `auto_approve is not permitted for {type}
     documents`
   - Set status to `approved`, record `approved_by` and `approved_at`
   - Fire entity lifecycle cascade (which for dev-plans is now a no-op per Task 7)
   - Write updated record

3. Add `auto_approve` parameter parsing in `docRegisterAction`.

4. Commit via `CommitStateAndPaths` (from Task 5's register commit path — the document
   file is already included as an extra path).

**Dependencies:** Task 6 (auto-refresh on approve), Task 7 (cascade removal for dev-plans)

**Effort:** Medium

**Tests:**
- `auto_approve: true` with `dev-plan` registers and approves in one call
- `auto_approve: true` with `research` succeeds
- `auto_approve: true` with `report` succeeds
- `auto_approve: true` with `design` returns an error
- `auto_approve: true` with `specification` returns an error
- Auto-approved dev-plan does NOT cascade to `developing` (FR-C05)
- Auto-approved document has correct `approved_by` and `approved_at`
- Without `auto_approve`, register behaviour is unchanged (regression)

---

### Task 11: Document section validation

**Objective:** Validate that registered and approved documents contain the required
markdown sections declared in `stage-bindings.yaml`.

**Specification references:** FR-D01, FR-D02, FR-D03, FR-D04, FR-D05, FR-D06, FR-D07

**Input context:**
- `.kbz/stage-bindings.yaml` — `document_template.required_sections` per stage
- `internal/service/documents.go` — `SubmitDocument`, `ApproveDocument`
- `internal/config/` — configuration loading (stage-bindings is read here)

**Output artifacts:**
- New `internal/service/section_validate.go`
- New `internal/service/section_validate_test.go`
- Modified `internal/service/documents.go` — integration into register and approve
- Modified `internal/service/documents_test.go`

**Changes required:**

1. Create `section_validate.go` with:
   - `ValidateSections(filePath string, docType string, config) SectionValidationResult`
   - Loads required sections for the document type from stage-bindings config
   - Scans the markdown file for level-2 headings (`## ...`) only (FR-D02, FR-D03)
   - Case-insensitive comparison against required section names
   - Returns `SectionValidationResult{Missing []string, Found []string, Valid bool}`

2. In `SubmitDocument` (register path): after writing the record, call section
   validation. Include missing sections as a `warnings` field in the result.
   Registration proceeds regardless (FR-D04).

3. In `ApproveDocument`: call section validation before setting status to `approved`.
   If required sections are missing, return an error listing them (FR-D05).

4. If a document type has no required sections declared, validation passes
   unconditionally (FR-D06).

5. Required sections config is read from `stage-bindings.yaml` at startup and
   cached (FR-D07).

**Dependencies:** None (can be built independently, integrated last).

**Effort:** Medium

**Tests:**
- Document with all required sections passes validation
- Document missing one section returns it in `Missing`
- Heading matching is case-insensitive (`## overview` matches `Overview`)
- Level-1 (`#`) and level-3+ (`###`) headings do not match level-2 requirements
- `doc register` with missing sections succeeds but includes warnings
- `doc approve` with missing sections returns error listing them
- Document type with no declared required sections always passes

---

### Task 12: Skill and documentation updates

**Objective:** Update all skill documents and `AGENTS.md` to reflect the new workflow
behaviour: decoupled dev-plan approval, auto-commit, auto-approve, new doc actions.

**Specification references:** FR-C04 (workflow documentation of the explicit transition)

**Input context:**
- Task 7 result — dev-plan cascade removed
- Task 10 result — `auto_approve` available
- `.kbz/skills/write-dev-plan/SKILL.md`
- `.kbz/skills/decompose-feature/SKILL.md`
- `.agents/skills/kanbanzai-workflow/SKILL.md`
- `.agents/skills/kanbanzai-documents/SKILL.md`
- `.agents/skills/kanbanzai-agents/SKILL.md`
- `AGENTS.md`

**Output artifacts:**
- Modified files listed above

**Changes required:**

1. **`write-dev-plan/SKILL.md`:** Update procedure to reflect that approving a dev-plan
   no longer transitions the feature. Add guidance to use `auto_approve: true` on
   registration. Note that a human "go" signal is needed before transitioning to
   `developing`.

2. **`decompose-feature/SKILL.md`:** Clarify that decomposition occurs during
   `dev-planning` (the feature does not need to be in `developing`).

3. **`kanbanzai-workflow/SKILL.md`:** Update stage gate documentation to clarify that
   the dev-planning exit gate is on the entity transition, not document approval.

4. **`kanbanzai-documents/SKILL.md`:** Add documentation for `doc move` and `doc delete`.
   Update "after editing a registered document" to note that `doc approve` auto-refreshes
   the hash. Remove guidance about calling `doc refresh` before `doc approve`.

5. **`kanbanzai-agents/SKILL.md`:** Soften guidance about manually committing
   `.kbz/state/` files — tools now auto-commit. Keep session-start orphaned-state check
   as a safety net.

6. **`AGENTS.md`:** Update pre-task checklist to note that orphaned state is less likely
   due to auto-commits. Keep the check but note it is diagnostic, not routine.

**Dependencies:** Task 7 (cascade removal), Task 10 (auto-approve)

**Effort:** Small

**Tests:** Manual review — skill documents are prose, not executable.

---

## Dependency Graph

```
        ┌─────────┐
        │ Task 1  │  CommitStateWithMessage + CommitStateAndPaths
        │ (found) │
        └────┬────┘
             │
     ┌───────┼───────┬───────┬───────┐
     │       │       │       │       │
     ▼       ▼       ▼       ▼       ▼
  Task 2  Task 3  Task 4  Task 5  Task 8 ──┐
  finish  decomp  merge   entity  doc      │
                          & doc   move     │
                                           │
                                        Task 9
                                        doc
                                        delete

  Task 6 ─────┐
  auto-refresh │
               ├──► Task 10 ──► Task 12
  Task 7 ─────┘    auto-       skill
  cascade           approve    updates
  removal

  Task 11 (independent — integrate last)
  section validation
```

**Parallel groups:**

- **Wave 1:** Tasks 1, 6, 7, 11 — no dependencies, fully independent
- **Wave 2:** Tasks 2, 3, 4, 5, 8, 9 — all depend on Task 1; independent of each other
- **Wave 3:** Task 10 — depends on Tasks 6 and 7
- **Wave 4:** Task 12 — depends on Tasks 7 and 10

**Critical path:** Task 6 → Task 10 → Task 12 (or Task 7 → Task 10 → Task 12)

---

## Interface Contracts

### Contract 1: Git commit function signatures

Defined by Task 1, consumed by Tasks 2–5, 8–10.

```
// internal/git/commit.go

// CommitStateWithMessage stages all files under .kbz/state/ and commits
// with the caller-supplied message. Returns (false, nil) when clean.
func CommitStateWithMessage(repoRoot, message string) (committed bool, err error)

// CommitStateAndPaths stages all files under .kbz/state/ plus each path
// in extraPaths, and commits in a single git commit. Returns (false, nil)
// when nothing is dirty. extraPaths are staged explicitly — no globs.
func CommitStateAndPaths(repoRoot, message string, extraPaths ...string) (committed bool, err error)

// CommitStateIfDirty remains unchanged — delegates to CommitStateWithMessage
// with the fixed stateCommitMessage constant.
func CommitStateIfDirty(repoRoot string) (committed bool, err error)
```

### Contract 2: Document service — MoveDocument

Defined by Task 8.

```
// internal/service/documents.go

type MoveDocumentInput struct {
    ID      string // document record ID (required)
    NewPath string // target file path relative to repo root (required)
}

// MoveDocument relocates a document file, updates its record (path, type,
// content hash), and preserves all other fields. Returns error if the ID
// is not found or the source file does not exist.
func (s *DocumentService) MoveDocument(input MoveDocumentInput) (DocumentResult, error)
```

### Contract 3: Document service — DeleteDocument

Defined by Task 9.

```
// internal/service/documents.go

type DeleteDocumentInput struct {
    ID    string // document record ID (required)
    Force bool   // required for approved documents (default false)
}

// DeleteDocument removes the document file, state record, index file,
// and clears the owning entity's document reference. Returns error if
// ID not found or if approved without Force.
func (s *DocumentService) DeleteDocument(input DeleteDocumentInput) (DocumentResult, error)
```

### Contract 4: Section validation function

Defined by Task 11, consumed by `SubmitDocument` and `ApproveDocument`.

```
// internal/service/section_validate.go

type SectionValidationResult struct {
    Missing []string // required sections not found in the document
    Found   []string // required sections found in the document
    Valid   bool     // true if Missing is empty
}

// ValidateSections checks a markdown file for level-2 headings matching
// the required sections declared in stage-bindings.yaml for the given
// document type. Returns Valid=true if the type has no required sections.
func ValidateSections(filePath string, docType string, requiredSections []string) (SectionValidationResult, error)
```

### Contract 5: Commit message format

All auto-commit messages follow the pattern: `workflow({entity-id}): {action description}`

| Tool | Action | Message format |
|------|--------|----------------|
| `finish` | single complete | `workflow(TASK-xxx): complete – {summary≤50}` |
| `finish` | batch complete | `workflow: complete {N} tasks` |
| `decompose` | apply | `workflow(FEAT-xxx): decompose into {N} tasks` |
| `merge` | execute | `workflow(FEAT-xxx): mark worktree merged` |
| `entity` | create | `workflow({ID}): create {type}` |
| `entity` | transition | `workflow({ID}): transition {from} → {to}` |
| `doc` | register | `workflow({doc-id}): register {type}` |
| `doc` | approve | `workflow({doc-id}): approve {type}` |
| `doc` | move | `workflow({doc-id}): move to {new_path}` |
| `doc` | delete | `workflow({doc-id}): delete {type}` |

---

## Traceability Matrix

Every specification requirement maps to at least one task. Every task maps to at least
one requirement.

### Pillar A: Parameterised State Commit

| Requirement | Description | Task(s) |
|-------------|-------------|---------|
| FR-A01 | `CommitStateWithMessage` function | 1 |
| FR-A02 | `CommitStateAndPaths` function | 1 |
| FR-A03 | Explicit path list, no globs | 1 |
| FR-A04 | No-op when clean | 1 |
| FR-A05 | `CommitStateIfDirty` backward compat | 1 |
| FR-A06 | `finish` auto-commit (single) | 2 |
| FR-A07 | `finish` auto-commit (batch) | 2 |
| FR-A08 | `decompose apply` auto-commit | 3 |
| FR-A09 | `merge execute` auto-commit | 4 |
| FR-A10 | `entity create` auto-commit | 5 |
| FR-A11 | `entity transition` auto-commit | 5 |
| FR-A12 | `doc approve` auto-commit | 5 |
| FR-A13 | Best-effort semantics | 2, 3, 4, 5 |
| FR-A14 | Scoped staging | 1, 2, 3, 4, 5 |

### Pillar B: Atomic Document File Operations

| Requirement | Description | Task(s) |
|-------------|-------------|---------|
| FR-B01 | `doc register` auto-commit with file | 5 |
| FR-B02 | `auto_approve` parameter | 10 |
| FR-B03 | `auto_approve` behaviour | 10 |
| FR-B04 | Auto-approve whitelist | 10 |
| FR-B05 | Default register unchanged | 10 |
| FR-B06 | Auto-refresh hash on approve | 6 |
| FR-B07 | Approve error on missing file | 6 |
| FR-B08 | `doc refresh` remains available | 6 |
| FR-B09 | `doc move` action | 8 |
| FR-B10 | `doc move` behaviour | 8 |
| FR-B11 | `doc move` type update | 8 |
| FR-B12 | `doc move` preserves fields | 8 |
| FR-B13 | `doc move` atomic commit | 8 |
| FR-B14 | `doc move` error on missing file | 8 |
| FR-B15 | `doc move` error on missing record | 8 |
| FR-B16 | `doc delete` action | 9 |
| FR-B17 | `doc delete` force gate | 9 |
| FR-B18 | `doc delete` cleanup steps | 9 |
| FR-B19 | `doc delete` atomic commit | 9 |
| FR-B20 | `doc delete` handles missing file | 9 |
| FR-B21 | `doc delete` error on missing record | 9 |

### Pillar C: Dev-Plan Approval Decoupling

| Requirement | Description | Task(s) |
|-------------|-------------|---------|
| FR-C01 | Dev-plan approval no cascade | 7 |
| FR-C02 | Design approval cascade preserved | 7 |
| FR-C03 | Spec approval cascade preserved | 7 |
| FR-C04 | Explicit transition required | 7, 12 |
| FR-C05 | `auto_approve` + dev-plan no cascade | 10 |

### Pillar D: Document Section Validation

| Requirement | Description | Task(s) |
|-------------|-------------|---------|
| FR-D01 | Section validation function | 11 |
| FR-D02 | Level-2 heading match | 11 |
| FR-D03 | Other heading levels excluded | 11 |
| FR-D04 | Register: warnings only | 11 |
| FR-D05 | Approve: hard gate | 11 |
| FR-D06 | No required sections → pass | 11 |
| FR-D07 | Config cached at startup | 11 |

### Non-Functional Requirements

| Requirement | Description | Task(s) |
|-------------|-------------|---------|
| NFR-01 | Auto-commit latency | 1 |
| NFR-02 | Unit tests with real git repos | 1, 2, 3, 4, 5, 8, 9 |
| NFR-03 | Parallel-safe tests | All |
| NFR-04 | No accidental staging | 1 |
| NFR-05 | Backward compatibility | 1, 5, 6, 7, 10 |
| NFR-06 | Readable commit messages | 2, 3, 4, 5, 8, 9 |

---

## Risk Assessment

### Risk: Repo root path not uniformly available in tool handlers

- **Probability:** Medium
- **Impact:** Medium
- **Mitigation:** Most tool handlers receive `entitySvc` which has `Root()` returning
  the `.kbz/` directory. Derive repo root as its parent. `merge_tool.go` already receives
  `repoPath` explicitly. Establish a consistent pattern in Task 2 (first auto-commit
  call site) and replicate in Tasks 3–5.
- **Affected tasks:** 2, 3, 4, 5

### Risk: Auto-commits create git lock contention under concurrent agents

- **Probability:** Medium
- **Impact:** Low
- **Mitigation:** All commits are best-effort (FR-A13). Lock contention surfaces as a
  failed commit, which is logged and retried naturally at the next tool call. The
  `handoff` safety net catches any uncommitted state before sub-agent dispatch.
- **Affected tasks:** 2, 3, 4, 5

### Risk: Removing dev-plan cascade breaks existing agent workflows

- **Probability:** High
- **Impact:** Medium
- **Mitigation:** Task 12 updates all skill documents before the feature ships. Agents
  that forget to transition explicitly will be caught by `developing` prerequisites on
  `handoff`. The failure mode is visible, not silent.
- **Affected tasks:** 7, 12

### Risk: `doc delete` accidentally removes a document other entities depend on

- **Probability:** Low
- **Impact:** High
- **Mitigation:** Default `force: false` prevents deletion of approved documents (FR-B17).
  Entity reference clearing (FR-B18) is part of the atomic operation, so no dangling
  references are left.
- **Affected tasks:** 9

### Risk: Section validation rejects previously approvable documents

- **Probability:** Medium
- **Impact:** Low
- **Mitigation:** Validation is only enforced on `approve`, not retroactively. Existing
  approved documents are unaffected. New documents get warnings on registration (FR-D04),
  giving the author time to fix before attempting approval.
- **Affected tasks:** 11

### Risk: Test suite execution time increases due to real git operations

- **Probability:** Medium
- **Impact:** Low
- **Mitigation:** Follow the existing `commit_test.go` pattern: temp directories with
  real git repos, isolated and `t.Parallel()`-safe. Consider the package-level var
  injection pattern (from `handoff_tool.go`) for tool-level tests that don't need real
  git.
- **Affected tasks:** 1, 2, 3, 4, 5, 8, 9

---

## Verification Approach

| Acceptance Criterion | Verification Method | Producing Task |
|---------------------|--------------------| ---------------|
| AC-A01: `CommitStateWithMessage` creates commit with supplied message | Unit test: `git log --oneline` after call | 1 |
| AC-A02: `CommitStateWithMessage` returns `(false, nil)` when clean | Unit test: clean repo, verify return value | 1 |
| AC-A03: `CommitStateAndPaths` creates single commit with state + extra paths | Unit test: `git show --stat` after call | 1 |
| AC-A04: `CommitStateAndPaths` does not stage outside explicit paths | Unit test: create unrelated dirty file, verify excluded | 1 |
| AC-A05: `CommitStateIfDirty` unchanged | Regression test: existing test still passes | 1 |
| AC-A06: `finish` (single) commits with task ID and summary | Unit test: verify `git log` message format | 2 |
| AC-A07: `finish` (batch) creates single commit | Unit test: verify commit count | 2 |
| AC-A08: `decompose apply` atomic commit | Unit test: verify single commit contains all tasks | 3 |
| AC-A09: `merge execute` state commit after merge | Unit test: verify commit after merge commit | 4 |
| AC-A10: `entity create` commits with ID and type | Unit test: verify `git log` message | 5 |
| AC-A11: `entity transition` commits with from/to | Unit test: verify `git log` message | 5 |
| AC-A12: `doc approve` commits with doc ID and type | Unit test: verify `git log` message | 5 |
| AC-A13: Commit failure is non-blocking | Unit test: inject failure, verify normal return | 2, 3, 4, 5 |
| AC-A14: Staging is scoped | Unit test: verify no files outside scope in commit | 1, 2, 3, 4, 5 |
| AC-B01: `doc register` commits state + document file | Unit test: verify commit contains both | 5 |
| AC-B02–B07: `auto_approve` whitelist and behaviour | Unit tests per type | 10 |
| AC-B08: Approve succeeds after edit without refresh | Unit test: edit file, approve, verify success | 6 |
| AC-B09: Approve updates stored hash | Unit test: verify hash changed | 6 |
| AC-B10: Approve fails on missing file | Unit test: delete file, verify error | 6 |
| AC-B11: `doc refresh` still works | Regression test | 6 |
| AC-B12–B18: `doc move` field updates and error cases | Unit tests per scenario | 8 |
| AC-B19–B26: `doc delete` cleanup and error cases | Unit tests per scenario | 9 |
| AC-C01: Dev-plan approval no cascade | Unit test: approve, verify feature unchanged | 7 |
| AC-C02–C04: Other cascades preserved | Regression tests | 7 |
| AC-C05: Explicit transition works with prerequisites | Unit test: transition with approved plan + tasks | 7 |
| AC-C06: Explicit transition fails without prerequisites | Unit test: transition without approved plan | 7 |
| AC-C07: `auto_approve` + dev-plan no cascade | Unit test | 10 |
| AC-D01–D07: Section validation scenarios | Unit tests per scenario | 11 |