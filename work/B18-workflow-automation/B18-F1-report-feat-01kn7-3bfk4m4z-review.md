# FEAT-01KN7-3BFK4M4Z Implementation Review
# Workflow State Automation

- **Date:** 2026-04-02
- **Feature:** FEAT-01KN7-3BFK4M4Z (auto-commit-and-doc-ops)
- **Plan:** P18-workflow-automation
- **Branch reviewed:** `feature/FEAT-01KN73BFK4M4Z-auto-commit-and-doc-ops` (commit `6373e1c`)
- **Reviewer:** AI agent
- **Status:** Review complete — issues found

---

## Overview

The implementation covers all four specification pillars in a single commit across 18 files
(1,252 insertions). The core logic is correct and the architecture is well-designed.
The primary gap is test coverage: service-layer tests for every new service method are
absent, and the auto-commit path in all tool handlers is untested despite the injection
infrastructure being in place. There are also two secondary issues: a tool description that
does not mention the new actions (agent discoverability), and commit message format
deviations from the spec.

---

## Scope

Files reviewed:

- `internal/git/commit.go` and `commit_test.go` (Pillar A foundation)
- `internal/service/section_validate.go` and `section_validate_test.go` (Pillar D)
- `internal/service/documents.go` and `documents_test.go` (Pillars B, C, D integration)
- `internal/mcp/finish_tool.go` (FR-A06, FR-A07)
- `internal/mcp/decompose_tool.go` (FR-A08)
- `internal/mcp/merge_tool.go` (FR-A09)
- `internal/mcp/entity_tool.go` (FR-A10, FR-A11)
- `internal/mcp/doc_tool.go` (FR-A12, FR-B01, FR-B09, FR-B16)
- `internal/mcp/server.go` (FR-D07 startup wiring)
- `.agents/skills/kanbanzai-documents/SKILL.md`
- `.agents/skills/kanbanzai-workflow/SKILL.md`
- `.agents/skills/kanbanzai-agents/SKILL.md`
- `.kbz/skills/write-dev-plan/SKILL.md`
- `.kbz/skills/decompose-feature/SKILL.md`
- `AGENTS.md`

---

## Specification Coverage

| Pillar | Requirements | Implemented | Tested |
|--------|-------------|-------------|--------|
| A — Parameterised commit | FR-A01–A14 | ✅ All | ⚠️ git layer tested; handler ACs untested |
| B — Doc file ops | FR-B01–B21 | ✅ All | ❌ Most ACs untested |
| C — Dev-plan decoupling | FR-C01–C05 | ✅ All | ❌ AC-C01 untested |
| D — Section validation | FR-D01–D07 | ✅ All | ⚠️ Unit tested; integration untested |

---

## Findings

### F01 — CRITICAL: No service-layer tests for `MoveDocument`, `DeleteDocument`, or `auto_approve`

**Requirement:** NFR-02 — *"All new functions and tool handler changes MUST be covered by
unit tests."*

`documents_test.go` was extended with `TestApproveDocument_AutoRefreshHashOnApproval` and
`TestApproveDocument_FileMissing` (both correct and useful), but has no tests at all for the
three new behaviours that represent the bulk of the new service-layer code:

- `MoveDocument` — acceptance criteria AC-B12 through AC-B18 are completely untested
- `DeleteDocument` — acceptance criteria AC-B19 through AC-B26 are completely untested
- `SubmitDocument` with `AutoApprove: true` — acceptance criteria AC-B02 through AC-B07
  are completely untested

The injection infrastructure (`SetEntityHook`, `SetSectionProvider`) is already available
in tests; the existing `TestApproveDocument_TransitionsFeatureOnSpecApproval` is a good
structural template for the new tests.

---

### F02 — CRITICAL: No test for dev-plan cascade removal (AC-C01)

The removal of the `DocumentTypeDevPlan` case from the approval cascade is the most
significant behavioural change in the feature. An accidental reintroduction of that case
would not be caught by any existing test.

Required test (absent): a test that approves a dev-plan on a feature in `dev-planning`
status and verifies the feature's status is **unchanged** after approval — in contrast to
the spec approval test (`TestApproveDocument_TransitionsFeatureOnSpecApproval`) which
verifies the cascade does fire for specs.

The existing regression tests for design and spec cascades (AC-C02, AC-C03, AC-C04)
are implicitly covered by existing tests, but AC-C01 is not.

---

### F03 — HIGH: Auto-commit path is untested in all tool handlers

The dev-plan specifies acceptance criteria AC-A06 through AC-A14: message format
correctness and best-effort failure semantics for every handler.

Package-level injection variables (`finishCommitFunc`, `entityCommitFunc`,
`decomposeCommitFunc`, `mergeCommitFunc`, `docCommitFunc`, `docCommitPathsFunc`) are
correctly wired in production but **no test uses them**. The following are untested:

- Correct message format for each handler (e.g. `workflow(TASK-xxx): complete – ...`)
- That a failing commit function does not prevent the tool result from being returned
  (best-effort semantics, AC-A13)
- That batch `finish` produces a single commit, not one per task (AC-A07)

The `handoff_tool_test.go` pattern for injecting `commitStateFunc` is the right model.

---

### F04 — HIGH: Section validation integration in service layer untested (AC-D05, AC-D06)

`section_validate_test.go` thoroughly tests `ValidateSections` in isolation, but the
integration path — where `SubmitDocument` with a `sectionProvider` returns `Warnings`
in the result (AC-D05), and `ApproveDocument` with a `sectionProvider` rejects when
sections are missing (AC-D06) — is not tested in `documents_test.go`.

The `sectionProvider` field defaults to `nil`, so neither path is exercised by any
existing test. A bug in the wiring between `sectionProvider` and `ValidateSections`
inside `SubmitDocument` or `ApproveDocument` would pass all current tests.

---

### F05 — HIGH: Tool description omits `move` and `delete` actions

In `doc_tool.go`, the `mcp.WithDescription` string reads:

```
"Actions: register, approve, get, content, list, gaps, validate, supersede, refresh,
chain, import, audit, evaluate, record_false_positive. ..."
```

The `action` parameter's description was correctly updated to include `move, delete`,
but agents deciding whether to use this tool read the **main description** first.
`move` and `delete` are therefore undiscoverable to agents browsing the tool list
unless they read every parameter description individually.

The main description must be updated to list the two new actions.

---

### F06 — MEDIUM: Commit message format deviates from spec (Contract 5)

The dev-plan's Contract 5 commit message table specifies:

| Action | Expected format |
|--------|----------------|
| `doc move` | `workflow({doc-id}): move to {new_path}` |
| `doc delete` | `workflow({doc-id}): delete {type}` |

The implementation uses:

- Move: `workflow(%s): move document to %s`  — adds the word "document" and changes
  "move to" to "move document to"
- Delete: `workflow(%s): delete %s document` — appends the word "document" after the type

This impairs `git log --oneline` readability (NFR-06) and creates inconsistency with the
documented message format that agents and tooling may rely on.

---

### F07 — MEDIUM: `auto_approve` bypasses the section hard gate

When `auto_approve: true` is passed to `doc register`, section validation in
`SubmitDocument` runs in warn-only mode (FR-D04). However, auto-approve results in an
`approved` document — effectively skipping the hard gate that FR-D05 requires at approval
time. A dev-plan auto-approved while missing required sections will have `status: approved`
but be structurally incomplete.

The spec is silent on this interaction: FR-D05 says "The `doc approve` action MUST call
section validation" but auto-approve does not call `ApproveDocument`. This is a design
gap in the spec that the implementation inherits. A future revision should decide whether
auto-approve should enforce FR-D05 semantics (recommended) or explicitly document that it
does not.

---

### F08 — MINOR: Double `entitySvc.Get` for feature Phase 2 transitions

In `entityTransitionAction`, when a feature undergoes a Phase 2 transition, `entitySvc.Get`
is called twice: once for gate evaluation (the `feature` struct loaded at the start of the
feature branch), and once to capture `fromStatusBeforeTransition` just before `UpdateStatus`.

The `currentStatus` variable from the first load already holds the information needed for
the commit message. The second Get is a redundant store read.

---

### F09 — MINOR: Orphaned comment between method declarations in `documents.go`

The comment:

```
// OldPath is not a field of DocumentResult — callers of MoveDocument use the
// input.ID's original path (load before calling) alongside the returned Path.
```

…appears as a free-standing comment between the closing brace of `MoveDocument` and the
opening of `DeleteDocument`. It will not be rendered as a doc comment for either method
and is invisible to IDE hover-over. This design note should either be incorporated into
`MoveDocument`'s godoc block or removed (the handler's `GetDocument`-before-move pattern
makes the intent self-evident).

---

## What Is Implemented Well

- **`CommitStateAndPaths`** correctly handles the case where only extra paths are dirty
  (state dir is clean) via a conditional `git add -- .kbz/state/` — this avoids a git
  error and is the correct behaviour. The tests cover this edge case explicitly.

- **`MoveDocument` rollback** — on hash failure after rename, the code calls
  `os.Rename(newFullPath, oldFullPath)` to restore the original location. This is a good
  defensive measure that prevents a half-completed move from leaving the file system and
  record in inconsistent states.

- **Best-effort commit placement** — all commit calls occur after the logical operation
  succeeds, and failures are logged without propagating as tool errors. The pattern is
  consistent across all six handler variables.

- **Section validation gate ordering in `ApproveDocument`** — validation runs before the
  status write. A document that fails validation is never written in an approved state.

- **`auto_approve` orphan prevention** — when the type whitelist check fails, the
  just-written draft record is cleaned up via `_ = s.store.Delete(doc.ID)` before
  returning the error. This prevents orphaned draft records from accumulating.

- **Skill document updates** — `kanbanzai-documents/SKILL.md` covers `move`, `delete`,
  `auto_approve`, and auto-refresh accurately. The `write-dev-plan/SKILL.md` note about
  the required explicit `entity(action: transition)` call is well-placed and clear.

---

## Recommended Actions (Priority Order)

1. **Add `MoveDocument` and `DeleteDocument` tests** in `documents_test.go`, covering
   all AC-B12 through AC-B26 scenarios. Use `TestApproveDocument_TransitionsFeatureOnSpecApproval`
   as a structural template.

2. **Add `auto_approve` tests** in `documents_test.go` — at minimum: dev-plan
   auto-approves successfully; design returns an error; auto-approved dev-plan does not
   cascade the feature (AC-C07).

3. **Add a dev-plan cascade removal test** — `TestApproveDocument_DevPlanApproval_NoCascade`
   that approves a dev-plan and asserts the owning feature stays in `dev-planning`.

4. **Add section validation integration tests** — one for `SubmitDocument` with a
   `sectionProvider` that returns missing sections (assert `result.Warnings` is populated),
   and one for `ApproveDocument` with missing sections (assert error is returned).

5. **Add auto-commit injection tests** per handler, following the `handoff_tool_test.go`
   pattern — verify message format and best-effort semantics (commit failure does not
   block result) for at least `finish`, `entity create`, `entity transition`, `doc register`,
   and `doc approve`.

6. **Fix the `doc` tool description** — add `move` and `delete` to the
   `mcp.WithDescription` action list.

7. **Fix commit message formats** — `"move document to"` → `"move to"`, and remove the
   trailing `" document"` suffix from the delete message, to match Contract 5.

8. **Resolve the `auto_approve` + section validation interaction** — either enforce the
   FR-D05 hard gate during auto-approve, or add an explicit note in the spec and skill
   documents that auto-approve intentionally skips section validation.

9. **Move or remove the orphaned comment** in `documents.go` between `MoveDocument` and
   `DeleteDocument`.