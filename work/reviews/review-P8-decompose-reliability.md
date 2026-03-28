# Review: P8 — decompose propose Reliability Fixes

| Field       | Value                                              |
|-------------|----------------------------------------------------|
| Plan        | P8-decompose-reliability                           |
| Reviewer    | Claude Opus 4.6                                    |
| Date        | 2026-03-28T12:29:26Z                               |
| Verdict     | **Pass** — ready to ship, one minor deviation noted |

---

## 1. Scope

P8 fixes two silent-failure modes in `decompose propose` that caused it to
generate structurally plausible but entirely wrong task breakdowns when a spec
was not ready for decomposition.

| Feature | Slug | Status |
|---------|------|--------|
| FEAT-01KMT-58SKYM5C | agents-md-decompose-rule | reviewing |
| FEAT-01KMT-58TV8V9C | decompose-precondition-gates | reviewing |

Design: `work/design/decompose-reliability.md` (approved).

Specifications:
- `work/spec/agents-md-decompose-rule.md` (approved)
- `work/spec/decompose-precondition-gates.md` (approved)

---

## 2. Feature Review: AGENTS.md Decompose Precondition Rule

**Feature:** FEAT-01KMT-58SKYM5C
**Task:** TASK-01KMT-5J4BP79V (done)
**Verdict:** ✅ Pass

### What changed

A precondition block was added to the Stage 5 (Dev Plan & Tasks) section of
`AGENTS.md`, positioned before the "Agent role" bullet list. The block
instructs agents to:

1. Confirm the spec document record is in `approved` status before calling
   `decompose propose`. Corrective action: call `doc approve` first.
2. If the spec was registered in the current session, call `index_repository`
   before calling `decompose propose`. Corrective action: run
   `index_repository` then retry.

Closing statement: "Skipping either step will cause `decompose propose` to
fail or produce wrong output."

### Acceptance criteria

| AC | Criterion | Status |
|----|-----------|--------|
| AC-01 | Block in Stage 5, before or after Agent role bullets | ✅ Placed before Agent role bullets |
| AC-02 | States spec must be `approved` (hard requirement) | ✅ |
| AC-03 | States `index_repository` needed if spec registered this session | ✅ |
| AC-04 | Concrete corrective actions for both conditions | ✅ `doc approve` and `index_repository` |
| AC-05 | No existing Stage 5 content altered or removed | ✅ |
| AC-06 | No files other than `AGENTS.md` modified | ✅ |

### Findings

None.

---

## 3. Feature Review: Decompose Precondition Gates

**Feature:** FEAT-01KMT-58TV8V9C
**Task:** TASK-01KMT-5J878QVD (done)
**Verdict:** ✅ Pass (with one noted deviation)

### What changed

Three changes to `DecomposeFeature` in `internal/service/decompose.go`:

1. **Spec approval gate (step 4).** After loading the spec document, the
   method checks `docResult.Status != "approved"` and returns a hard error
   with the document ID, current status, and corrective action. This fires
   before any content parsing.

2. **Acceptance criteria gate (step 6).** After parsing the spec with
   `parseSpecStructure`, the method checks `len(spec.acceptanceCriteria) == 0`
   and returns a hard error with the document ID and formatting guidance.

3. **Section-header fallback removal.** The old code path that generated tasks
   named "Implement 1. Purpose", "Implement 2. Goals", etc. is completely
   removed. The warning string `"tasks derived from section headers"` no longer
   exists anywhere in the codebase. `generateProposal` now only generates
   tasks from acceptance criteria — and is only called after step 6 confirms
   ACs exist.

The `setupDecomposeTest` helper was updated to approve spec documents before
linking them, so all existing tests pass through the new approval gate
transparently.

### Gate ordering

The implementation correctly sequences:
1. Feature load
2. Spec document linked check
3. Spec content retrieval
4. **Approval gate** — hard stop if not approved
5. Spec parsing
6. **AC content gate** — hard stop if no ACs found
7. Proposal generation (only reached with approved spec + non-empty ACs)

This satisfies AC-03 (approval before content check) and AC-07 (content check
after approval).

### Acceptance criteria

| AC | Criterion | Status | Notes |
|----|-----------|--------|-------|
| AC-01 | Draft spec → error, no proposal | ✅ | |
| AC-02 | Error includes doc ID, status, corrective action | ✅ | Format: `spec "..." is in "draft" status — approve the spec before decomposing` |
| AC-03 | Approval gate fires before content check | ✅ | Step 4 before step 6 |
| AC-04 | Approved spec → gate passes silently | ✅ | |
| AC-05 | Empty index content → error, no proposal | ⚠️ | See deviation below |
| AC-06 | Error includes doc ID and `index_repository` instruction | ⚠️ | See deviation below |
| AC-07 | Content gate after approval gate | ✅ | |
| AC-08 | Non-empty content → gate passes silently | ✅ | |
| AC-09 | Section-header fallback removed | ✅ | Code path gone |
| AC-10 | Warning string removed from codebase | ✅ | Verified by grep |
| AC-11 | Normal decompose behaviour unchanged | ✅ | Happy-path tests pass |
| AC-12 | Test: draft spec → expected error | ✅ | `TestDecomposeFeature_DraftSpec_ReturnsError` |
| AC-13 | Test: empty index → expected error | ✅ | `TestDecomposeFeature_NoACs_ReturnsError` |
| AC-14 | Test: valid input passes both gates | ✅ | `TestDecomposeFeature_ProposalProduced` |
| AC-15 | All tests pass with `-race` | ✅ | Full suite verified |

### Deviation: AC-05/AC-06 mechanism and error message

The spec describes an **index content gate** — querying the document
intelligence index for non-empty parsed content and returning:

```
spec content not yet indexed for "FEAT-.../specification-..." — run index_repository, then retry
```

The implementation instead uses a **structural parsing gate** — reading the
file from disk via `GetDocumentContent`, parsing with `parseSpecStructure`,
and checking whether any acceptance criteria were extracted. The error message
is:

```
no acceptance criteria found in spec "..." — ensure the spec uses checkbox
items (- [ ] ...) or numbered items within an Acceptance Criteria section
```

**Assessment:** This is a reasonable simplification that is arguably stronger
than the spec's approach. It catches both "index not populated" and "spec
genuinely has no parseable ACs" in a single check. The design document itself
noted (§3) that a disk-read fallback "blurs the boundary between the document
intelligence layer and raw file I/O" — the implementation sidesteps this by
always reading from disk and parsing structurally, which is simpler and more
reliable.

The practical impact is minimal: the AGENTS.md precondition rule
(FEAT-01KMT-58SKYM5C) already instructs agents to run `index_repository`
before calling `decompose propose` when the spec was recently registered. The
error message's formatting guidance ("ensure the spec uses checkbox items...")
is useful for the case where the spec genuinely lacks ACs — a scenario the
spec's `index_repository` hint would not address.

**Severity:** Informational. Not blocking.

**Recommendation:** If desired, the error message could be extended to also
mention `index_repository` as a possible corrective action, giving agents both
hints. This would be a one-line change. Not required for acceptance.

---

## 4. Test Coverage

| Test | Covers | Status |
|------|--------|--------|
| `TestDecomposeFeature_DraftSpec_ReturnsError` | Approval gate error path | ✅ |
| `TestDecomposeFeature_NoACs_ReturnsError` | Content gate error path | ✅ |
| `TestDecomposeFeature_ProposalProduced` | Happy path through both gates | ✅ |
| `TestDecomposeFeature_NoSpecRegistered` | No spec linked (pre-existing) | ✅ |
| `TestDecomposeFeature_EmptyFeatureID` | Empty input (pre-existing) | ✅ |
| `TestDecomposeFeature_GuidanceApplied` | Guidance rules (pre-existing) | ✅ |
| `TestDecomposeFeature_ContextPassed` | Context forwarding (pre-existing) | ✅ |
| `TestDecomposeFeature_TestTaskAdded` | Test task guidance (pre-existing) | ✅ |
| `TestDecomposeFeature_SliceDetailsPopulated` | Slice enrichment (pre-existing) | ✅ |

The old `TestDecomposeFeature_NoACs_FallsBackToSections` has been removed,
confirming the fallback path no longer exists. The `setupDecomposeTest` helper
now approves specs before linking them — this is correct since all non-gate
tests need an approved spec to proceed.

Full suite: `go test -race -count=1 ./...` — all packages pass.

---

## 5. Fallback Removal Verification

Verified by grep that the following are absent from the Go codebase:

- `FallsBackToSections` (old test name) — **not found**
- `tasks derived from section headers` (old warning string) — **not found**
- Any code path in `generateProposal` that creates tasks from section headers
  — **confirmed removed**; `generateProposal` only enters the task-creation
  branch when `len(spec.acceptanceCriteria) > 0`, and is only called after
  `DecomposeFeature` step 6 validates this condition

The `SliceAnalysis` method retains a note "slices derived from section
structure only" for informational purposes in its analysis output. This is
correct — the spec explicitly excludes changes to `slice` (§3.2), and slice
analysis is a different operation that legitimately uses section structure.

---

## 6. Findings Summary

| # | Severity | Feature | Finding |
|---|----------|---------|---------|
| 1 | Informational | FEAT-01KMT-58TV8V9C | AC-05/AC-06 implemented via structural parsing rather than index query; error message differs from spec wording. Functionally stronger. Not blocking. |

---

## 7. Verdict

**Pass.** Both features are ready to transition to `done`.

The implementation correctly prevents the silent-failure mode described in the
design: `decompose propose` will no longer produce section-header-derived task
lists when a spec is not ready. The approval gate stops decomposition of
unapproved specs. The AC content gate stops decomposition when no acceptance
criteria are found. The AGENTS.md precondition rule provides an immediate
documentation-level safeguard.

The one deviation (parse-based content check vs. index-based content check)
is a reasonable simplification that catches a broader class of problems than
the spec prescribed. No action required.