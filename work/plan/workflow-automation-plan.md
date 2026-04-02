# Workflow State Automation — Implementation Plan

**Status:** Draft
**Feature:** FEAT-01KN73BFK4M4Z (auto-commit-and-doc-ops)
**Plan:** P18-workflow-automation
**Design:** [Workflow Automation Design](../design/workflow-automation-design.md)
**Specification basis:** [Automation Opportunities Analysis](../reports/automation-opportunities-analysis.md)
**Date:** 2026-04-02

---

## 1. Scope

This plan decomposes the workflow state automation design into implementable tasks. It
covers all four pillars from the design document:

- **Pillar A:** Parameterised state commit (`CommitStateWithMessage`, auto-commit at
  transaction boundaries)
- **Pillar B:** Atomic document file operations (`CommitStateAndPaths`, `doc move`,
  `doc delete`, auto-refresh on approve, `auto_approve` flag)
- **Pillar C:** Decouple dev-plan approval from implementation trigger
- **Pillar D:** Document section validation on register and approve

**In scope:**

- New git commit functions in `internal/git/commit.go`
- Auto-commit call sites in MCP tool handlers
- New `doc` actions: `move`, `delete`
- Modified `doc` actions: `register` (auto-commit, auto-approve, section validation),
  `approve` (auto-refresh, section validation, cascade removal)
- Skill document updates for the new dev-plan workflow
- Tests for all new and modified behaviour

**Out of scope:**

- Changes to the `handoff` tool (continues to use its existing commit call)
- Changes to entity lifecycle state machine (transition rules unchanged)
- Changes to stage-bindings.yaml structure (required sections already declared)
- UI or CLI changes

**Specification references:** Recommendations 1–8 from the Automation Opportunities
Analysis report.

---

## 2. Task Breakdown

| # | Task | Files | Depends on | Effort |
|---|------|-------|-----------|--------|
| 1 | `CommitStateWithMessage` + `CommitStateAndPaths` | `internal/git/commit.go`, `internal/git/commit_test.go` | — | S |
| 2 | Auto-commit in `finish` handler | `internal/mcp/finish_tool.go`, `internal/mcp/finish_tool_test.go` | 1 | S |
| 3 | Auto-commit in `decompose apply` handler | `internal/mcp/decompose_tool.go`, `internal/mcp/decompose_tool_test.go` | 1 | S |
| 4 | Auto-commit in `merge execute` handler | `internal/mcp/merge_tool.go`, `internal/mcp/merge_tool_test.go` | 1 | S |
| 5 | Auto-commit in `entity` and `doc` handlers | `internal/mcp/entity_tool.go`, `internal/mcp/doc_tool.go`, tests | 1 | M |
| 6 | Auto-refresh hash on `doc approve` | `internal/service/documents.go`, `internal/service/documents_test.go` | — | S |
| 7 | Remove dev-plan approval cascade | `internal/service/documents.go`, `internal/service/documents_test.go` | — | S |
| 8 | `doc move` action | `internal/mcp/doc_tool.go`, `internal/service/documents.go`, tests | 1 | M |
| 9 | `doc delete` action | `internal/mcp/doc_tool.go`, `internal/service/documents.go`, tests | 1 | M |
| 10 | `auto_approve` flag on `doc register` | `internal/mcp/doc_tool.go`, `internal/service/documents.go`, tests | 6, 7 | M |
| 11 | Document section validation | `internal/service/documents.go`, `internal/service/section_validate.go`, tests | — | M |
| 12 | Skill and documentation updates | `.kbz/skills/`, `.agents/skills/`, `AGENTS.md` | 7, 10 | S |

**Effort key:** S = small (≤ 2 hours), M = medium (2–4 hours)

---

## 3. Task Details

### Task 1: `CommitStateWithMessage` + `CommitStateAndPaths`

**Goal:** Create the foundational git commit functions that all other tasks depend on.

**Files:**
- `internal/git/commit.go`
- `internal/git/commit_test.go`

**Depends on:** Nothing (foundation task).

**Changes required:**

1. Add `CommitStateWithMessage(repoRoot, message string) (bool, error)` that behaves
   identically to `CommitStateIfDirty` but accepts a caller-supplied commit message
   instead of the fixed `stateCommitMessage` constant.

2. Add `CommitStateAndPaths(repoRoot, message string, extraPaths ...string) (bool, error)`
   that:
   - Checks for dirty files under `.kbz/state/` OR any of the `extraPaths`
   - Stages `.kbz/state/` via `git add -- .kbz/state/`
   - Stages each extra path via `git add -- <path>`
   - Commits with the supplied message
   - Returns `(false, nil)` if nothing was dirty

3. Refactor `CommitStateIfDirty` to delegate to `CommitStateWithMessage` with the
   existing fixed message. This preserves backward compatibility for the `handoff`
   call site.

**Interface contract:** All downstream tasks (2–5, 8–10) call one of these two functions.
The signatures must be stable before parallel work begins.

**Tests:**
- `CommitStateWithMessage` with custom message appears in `git log`
- `CommitStateAndPaths` stages both `.kbz/state/` and extra paths
- `CommitStateAndPaths` with extra path outside `.kbz/state/` includes it in commit
- `CommitStateAndPaths` with no dirty files returns `(false, nil)`
- `CommitStateAndPaths` does not stage files not listed in `extraPaths`
- `CommitStateIfDirty` continues to work unchanged (regression)

**Spec ACs covered:** Rec 1 (foundation), Rec 2 (foundation), Rec 5 (foundation),
Rec 6 (foundation), Rec 7 (foundation).

---

### Task 2: Auto-commit in `finish` handler

**Goal:** `finish` commits all state changes atomically after successful task completion.

**Files:**
- `internal/mcp/finish_tool.go`
- `internal/mcp/finish_tool_test.go`

**Depends on:** Task 1.

**Changes required:**

1. After `CompleteTask` returns successfully, call `CommitStateWithMessage` with:
   `workflow(TASK-xxx): complete – {summary truncated to 50 chars}`

2. The commit is best-effort: if it fails, log a warning via `log.Printf` and return
   the normal `CompleteResult`. Do not propagate the commit error.

3. For batch mode (`tasks` array), commit once after all tasks are processed, not per
   task. Use a message like: `workflow: complete N tasks`.

**Interface contract with Task 1:** Uses `CommitStateWithMessage(repoRoot, message)`.
The `repoRoot` should be `"."` (consistent with `handoff`'s usage).

**Tests:**
- Single task completion creates a git commit with the expected message
- Batch completion creates a single commit
- Commit failure does not prevent `finish` from returning the result
- Task state files are included in the commit
- Knowledge and retro files written by `finish` are included in the commit

**Spec ACs covered:** Rec 5.

---

### Task 3: Auto-commit in `decompose apply` handler

**Goal:** `decompose apply` commits all created task files atomically after both passes.

**Files:**
- `internal/mcp/decompose_tool.go`
- `internal/mcp/decompose_tool_test.go`

**Depends on:** Task 1.

**Changes required:**

1. After `decompose apply` completes both passes (task creation and dependency wiring),
   call `CommitStateWithMessage` with:
   `workflow(FEAT-xxx): decompose into N tasks`

2. Best-effort semantics (same as Task 2).

**Tests:**
- `decompose apply` creates a commit containing all N task entity files
- Dependency wiring (pass 2) is included in the same commit
- Commit failure does not block the decomposition result

**Spec ACs covered:** Rec 6.

---

### Task 4: Auto-commit in `merge execute` handler

**Goal:** `merge execute` commits the worktree state update after a successful merge.

**Files:**
- `internal/mcp/merge_tool.go`
- `internal/mcp/merge_tool_test.go`

**Depends on:** Task 1.

**Changes required:**

1. After `executeMerge` returns successfully and the worktree record is updated to
   `merged`, call `CommitStateWithMessage` with:
   `workflow(FEAT-xxx): mark worktree merged`

2. This should run after the existing `postMergeInstall` hook (which is already
   best-effort). The commit captures the worktree record update.

3. Best-effort semantics.

**Tests:**
- Successful merge creates a state commit after the merge commit
- Worktree record file is included in the state commit
- Commit failure does not prevent merge result from being returned

**Spec ACs covered:** Rec 7.

---

### Task 5: Auto-commit in `entity` and `doc` handlers

**Goal:** Add auto-commit to the remaining state-mutating tool handlers.

**Files:**
- `internal/mcp/entity_tool.go`
- `internal/mcp/entity_tool_test.go`
- `internal/mcp/doc_tool.go`
- `internal/mcp/doc_tool_test.go`

**Depends on:** Task 1.

**Changes required:**

For `entity_tool.go`:
1. `entityCreateAction`: after successful creation, commit with
   `workflow(ID): create {type}`
2. `entityTransitionAction`: after successful transition, commit with
   `workflow(ID): transition {from} → {to}`

For `doc_tool.go`:
1. `docRegisterAction`: commit with `workflow(DOC-xxx): register {type}`
   (This becomes `CommitStateAndPaths` once Task 8/9 infrastructure is in place, but
   initially uses `CommitStateWithMessage` for state-only commit.)
2. `docApproveAction`: commit with `workflow(DOC-xxx): approve {type}`

All commits are best-effort.

**Tests:**
- Entity create produces a commit
- Entity transition produces a commit
- Doc register produces a commit
- Doc approve produces a commit (including cascaded entity transition files)
- Batch entity create produces a single commit

**Spec ACs covered:** Rec 1.

---

### Task 6: Auto-refresh hash on `doc approve`

**Goal:** Eliminate the manual `doc refresh` step before `doc approve`.

**Files:**
- `internal/service/documents.go`
- `internal/service/documents_test.go`

**Depends on:** Nothing.

**Changes required:**

1. In `ApproveDocument`, instead of loading the stored hash and comparing against the
   file, compute the hash from the file directly. If the computed hash differs from the
   stored hash, update the stored hash as part of the approval write.

2. Remove the error path that rejects approval due to hash mismatch. The hash is always
   recomputed from the current file content.

3. If the file does not exist on disk, return an error (this is a real problem, not a
   hash drift issue).

**Tests:**
- Approve succeeds when the file has been edited since registration (no refresh needed)
- The stored hash is updated to match the file's current content after approval
- Approve fails when the file does not exist on disk
- Existing tests that relied on hash-mismatch rejection are updated

**Spec ACs covered:** Rec 4.

---

### Task 7: Remove dev-plan approval cascade

**Goal:** Approving a dev-plan no longer transitions the feature to `developing`.

**Files:**
- `internal/service/documents.go`
- `internal/service/documents_test.go`

**Depends on:** Nothing.

**Changes required:**

1. In the approval cascade switch block (`documents.go` ~L418–435), remove or comment
   out the case:
   ```
   case entityType == "feature" && doc.Type == model.DocumentTypeDevPlan:
       targetStatus = "developing"
   ```

2. The approval still writes the document record with `approved` status. Only the
   entity transition cascade is removed.

3. Update or add tests to verify that dev-plan approval does NOT produce an
   `EntityTransition` side effect.

**Tests:**
- Approving a dev-plan sets document status to `approved`
- Approving a dev-plan does NOT transition the owning feature
- Approving a design still cascades (regression)
- Approving a spec still cascades (regression)

**Spec ACs covered:** Rec 3a.

---

### Task 8: `doc move` action

**Goal:** Add atomic document move to the `doc` tool.

**Files:**
- `internal/mcp/doc_tool.go`
- `internal/service/documents.go`
- `internal/service/documents_test.go`
- `internal/mcp/doc_tool_test.go`

**Depends on:** Task 1 (for `CommitStateAndPaths`).

**Changes required:**

1. Add `MoveDocument(input MoveDocumentInput) (DocumentResult, error)` to
   `DocumentService`. Input: `ID string`, `NewPath string`.

2. Implementation:
   - Load record by ID, verify file exists at current path
   - `os.Rename(oldPath, newPath)`
   - Update record's `path` field
   - If the new path implies a different document type directory, update `type`
   - Recompute content hash from new path
   - Write updated record via `DocumentStore.Write()`

3. Add `"move"` to the action dispatch in `doc_tool.go`. Parameters: `id` (required),
   `new_path` (required).

4. After the service call, commit via `CommitStateAndPaths(repoRoot, message, oldPath, newPath)`.

**Interface contract with Task 1:** Uses `CommitStateAndPaths` with two extra paths
(old and new locations).

**Tests:**
- Move updates the record's path field
- Move preserves document ID, approval status, and owner
- Move updates content hash
- Move changes document type when new path is in a different type directory
- Move produces a single atomic commit containing both file paths and state record
- Move fails if the source file does not exist
- Move fails if the document ID is not found

**Spec ACs covered:** Rec 2b.

---

### Task 9: `doc delete` action

**Goal:** Add atomic document deletion to the `doc` tool.

**Files:**
- `internal/mcp/doc_tool.go`
- `internal/service/documents.go`
- `internal/service/documents_test.go`
- `internal/mcp/doc_tool_test.go`

**Depends on:** Task 1 (for `CommitStateAndPaths`).

**Changes required:**

1. Add `DeleteDocument(input DeleteDocumentInput) (DocumentResult, error)` to
   `DocumentService`. Input: `ID string`, `Force bool`.

2. Implementation:
   - Load record by ID
   - If status is `approved` and `Force` is false, return error
   - `os.Remove(filePath)` — remove the document file
   - If record has an `owner`, clear the entity's document ref field via entity hook
   - Remove the state record file from `.kbz/state/documents/`
   - Remove any index file from `.kbz/index/documents/`

3. Add `"delete"` to the action dispatch in `doc_tool.go`. Parameters: `id` (required),
   `force` (optional, default false).

4. After the service call, commit via `CommitStateAndPaths(repoRoot, message, filePath)`.

**Tests:**
- Delete removes both the file and the state record
- Delete of a draft document succeeds without `force`
- Delete of an approved document fails without `force`
- Delete of an approved document succeeds with `force: true`
- Delete clears the owning entity's document reference field
- Delete produces a single atomic commit
- Delete fails gracefully if the file is already missing on disk

**Spec ACs covered:** Rec 2c.

---

### Task 10: `auto_approve` flag on `doc register`

**Goal:** Allow agent-authored documents to be registered and approved in one call.

**Files:**
- `internal/mcp/doc_tool.go`
- `internal/service/documents.go`
- `internal/service/documents_test.go`
- `internal/mcp/doc_tool_test.go`

**Depends on:** Task 6 (auto-refresh on approve), Task 7 (cascade removal for dev-plans).

**Changes required:**

1. Add `AutoApprove bool` to `RegisterDocumentInput`.

2. In `RegisterDocument`, after writing the initial record, if `AutoApprove` is true:
   - Validate document type is in the whitelist: `dev-plan`, `research`, `report`
   - If not in whitelist, return error: "auto_approve is not permitted for {type}
     documents"
   - Set status to `approved`, `approved_by` to the caller, `approved_at` to now
   - Fire the entity lifecycle cascade (which, for dev-plans, is now a no-op per Task 7)
   - Write the updated record

3. Add `auto_approve` parameter parsing in `doc_tool.go`'s register action.

4. Commit via `CommitStateAndPaths` (from Task 5's register commit, enhanced to include
   the document file path).

**Tests:**
- `auto_approve: true` with `dev-plan` type registers and approves in one call
- `auto_approve: true` with `research` type succeeds
- `auto_approve: true` with `report` type succeeds
- `auto_approve: true` with `design` type returns an error
- `auto_approve: true` with `specification` type returns an error
- Auto-approved dev-plan does NOT cascade to `developing` (Task 7 integration)
- Auto-approved document has correct `approved_by` and `approved_at` fields
- Without `auto_approve`, register behaviour is unchanged (regression)

**Spec ACs covered:** Rec 3c.

---

### Task 11: Document section validation

**Goal:** Validate required sections on register (warning) and approve (gate).

**Files:**
- `internal/service/section_validate.go` (new)
- `internal/service/section_validate_test.go` (new)
- `internal/service/documents.go`
- `internal/service/documents_test.go`

**Depends on:** Nothing (can be built independently, integrated into doc service last).

**Changes required:**

1. Create `section_validate.go` with:
   - A function to load required sections from stage-bindings config for a given
     document type
   - A function to scan a markdown file for level-2 headings (`## ...`)
   - A function to compare found headings against required sections
     (case-insensitive match)
   - Return type: `SectionValidationResult` with `Missing []string`,
     `Found []string`, `Valid bool`

2. In `RegisterDocument`, after writing the record, call section validation. Include
   any missing sections as a `warnings` field in the result. Registration proceeds
   regardless.

3. In `ApproveDocument`, call section validation before setting status to `approved`.
   If required sections are missing, return an error listing them. Approval is blocked.

**Tests:**
- Document with all required sections passes validation
- Document missing one section returns it in `Missing`
- Heading matching is case-insensitive
- Register with missing sections succeeds but includes warnings
- Approve with missing sections fails with descriptive error
- Document type with no required sections always passes
- Headings at level 1 (`#`) or level 3 (`###`) do not match level-2 requirements

**Spec ACs covered:** Rec 8.

---

### Task 12: Skill and documentation updates

**Goal:** Update skill documents and AGENTS.md to reflect the new workflow.

**Files:**
- `.kbz/skills/write-dev-plan/SKILL.md`
- `.kbz/skills/decompose-feature/SKILL.md`
- `.agents/skills/kanbanzai-workflow/SKILL.md`
- `.agents/skills/kanbanzai-documents/SKILL.md`
- `.agents/skills/kanbanzai-agents/SKILL.md`
- `AGENTS.md`

**Depends on:** Task 7 (cascade removal), Task 10 (auto-approve).

**Changes required:**

1. **`write-dev-plan/SKILL.md`:** Update the procedure to reflect that approving a
   dev-plan no longer transitions the feature. Add guidance to use `auto_approve: true`
   on registration. Note that a human "go" signal is needed before transitioning to
   `developing`.

2. **`decompose-feature/SKILL.md`:** Clarify that decomposition can occur during
   `dev-planning` (the feature does not need to be in `developing` state). Update the
   example workflow sequence.

3. **`kanbanzai-workflow/SKILL.md`:** Update the stage gate table to clarify that the
   dev-planning exit gate is on the entity transition, not the document approval. Update
   the anti-patterns section if needed.

4. **`kanbanzai-documents/SKILL.md`:** Add documentation for `doc move` and `doc delete`
   actions. Update the "after editing a registered document" section to note that
   `doc approve` now auto-refreshes the hash. Remove guidance about calling `doc refresh`
   before `doc approve`.

5. **`kanbanzai-agents/SKILL.md`:** Remove or soften the guidance about manually
   committing `.kbz/state/` files after tool calls. Note that tools now auto-commit.
   Keep the session-start check for orphaned state (as a safety net for edge cases).

6. **`AGENTS.md`:** Update the pre-task checklist to note that orphaned state is now
   less likely due to auto-commits. Keep the check but demote it from a required step
   to a diagnostic step.

**Tests:** Manual review — skill documents are prose, not executable.

**Spec ACs covered:** Rec 3 (workflow documentation), Rec 1 (commit documentation).

---

## 4. Dependency Graph

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

- **Wave 1:** Tasks 1, 6, 7, 11 (no dependencies, all independent)
- **Wave 2:** Tasks 2, 3, 4, 5, 8, 9 (all depend on Task 1; independent of each other)
- **Wave 3:** Task 10 (depends on Tasks 6 and 7)
- **Wave 4:** Task 12 (depends on Tasks 7 and 10)

**Critical path:** Task 1 → Task 10 (via Tasks 6 + 7) → Task 12

---

## 5. Risk Assessment

| Risk | Probability | Impact | Mitigation | Affected tasks |
|------|------------|--------|------------|----------------|
| Auto-commits create git lock contention when multiple agents run concurrently | Medium | Low | Commits are best-effort; failure is logged, not propagated. Retry naturally occurs at the next tool call. | 2, 3, 4, 5 |
| Removing dev-plan cascade breaks existing agent workflows | High | Medium | Task 12 updates all skill documents before the feature ships. The transition is explicit and well-documented. Agents that forget to transition will be caught by the `developing` prerequisites on `handoff`. | 7, 12 |
| `doc delete` accidentally removes an approved document that other entities depend on | Low | High | Default `force: false` prevents deletion of approved documents. Entity reference clearing is part of the atomic operation. | 9 |
| `CommitStateAndPaths` stages a path that doesn't exist (e.g., deleted file) | Low | Low | `git add` on a deleted file stages the deletion. This is correct behaviour for `doc delete` and `doc move` (old path). | 1 |
| Section validation rejects documents that were previously approvable | Medium | Low | Validation is only enforced on `approve`, not retroactively. Existing approved documents are unaffected. New documents get warnings on registration. | 11 |
| Test suite execution time increases due to real git operations in tests | Medium | Low | Use the existing test pattern from `commit_test.go`: temp directories with real git repos. Tests are isolated and parallel-safe. | 1, 2, 3, 4, 5, 8, 9 |

---

## 6. Verification Approach

| AC | Method | Producing task |
|----|--------|---------------|
| State-mutating tools auto-commit with descriptive messages | Unit test: verify `git log` after tool call | 2, 3, 4, 5 |
| `CommitStateAndPaths` stages both state and extra paths | Unit test: verify `git diff --cached` includes both trees | 1 |
| `finish` commit includes task, knowledge, retro, and unblocked task files | Unit test: verify commit contains all expected file paths | 2 |
| `decompose apply` commit is atomic (both passes in one commit) | Unit test: verify single commit after apply, all tasks present | 3 |
| `doc move` preserves ID, status, owner, and updates path | Unit test: verify record fields before and after move | 8 |
| `doc move` produces a single commit with old path removal and new path addition | Unit test: verify `git show` of commit | 8 |
| `doc delete` blocks on approved without `force` | Unit test: verify error returned | 9 |
| `doc delete` clears entity reference | Unit test: verify entity record after delete | 9 |
| `doc approve` no longer requires prior `refresh` | Unit test: edit file, approve directly, verify success | 6 |
| Dev-plan approval does not cascade to `developing` | Unit test: approve dev-plan, verify feature status unchanged | 7 |
| `auto_approve` with dev-plan succeeds | Unit test: register with `auto_approve: true`, verify `approved` status | 10 |
| `auto_approve` with design fails | Unit test: verify error returned | 10 |
| Section validation warns on register, blocks on approve | Unit test: document missing required section, verify warning vs. error | 11 |
| All commit failures are best-effort (non-blocking) | Unit test: inject commit failure, verify tool returns normal result | 2, 3, 4, 5 |
| Existing `handoff` commit behaviour unchanged | Regression test: verify `CommitStateIfDirty` still works | 1 |
| Skill documents reflect new dev-plan workflow | Manual review | 12 |