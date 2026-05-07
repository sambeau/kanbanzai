---
name: verify-closeout
description:
  expert: "Execute a 10-item Definition of Done checklist with concrete
    verification actions per item, producing a structured pass/fail report"
  natural: "Check that a feature is truly done by running through the
    Definition of Done checklist and reporting what passed and what failed"
triggers:
  - verify close-out
  - check definition of done
  - audit feature completion
  - verify feature is done
  - run DoD checklist
roles: [verifier]
stage: verifying
constraint_level: high
---

## Vocabulary

- **DoD** — Definition of Done. The ten conditions every feature must satisfy before
  reaching `done`.
- **verification action** — a concrete, repeatable command or query that produces
  evidence for or against a DoD item (e.g., `git status --porcelain`, `go test ./...`).
- **evidence** — the output, exit code, or entity state that proves a DoD item's
  pass or fail status. Every verdict must have evidence.
- **pass** — the verification action produced the expected result. The DoD item is
  satisfied.
- **fail** — the verification action did not produce the expected result, or the
  check could not be performed. Include the reason.

## Anti-Patterns

### Trusting Orchestrator Assurances

- **Detect:** Marking a checklist item pass based on what the orchestrator claims
  rather than running the verification action independently.
- **BECAUSE:** The orchestrator's context is saturated by end of cycle. Close-out
  steps are exactly what it forgets. The verifier exists because the orchestrator
  cannot be trusted to self-verify.
- **Resolve:** Run every verification action yourself. If the orchestrator says
  something is done, verify it anyway.

### Partial Checklist Execution

- **Detect:** Producing output with fewer than 10 items, or marking items without
  evidence.
- **BECAUSE:** The DoD is a contract — all ten conditions must be verified. Skipping
  items creates blind spots that compound across features and cause incidents.
- **Resolve:** Check every item. Every item must have evidence. If a check cannot
  be performed (missing tool, inaccessible state), mark it `fail` with the reason.

## Procedure

You will receive a feature ID. Execute all ten verification actions below and
produce the structured output format at the end. Do not converse, ask, or clarify —
just check and report.

### Item 1: All Tasks Terminal

**Condition:** Every task under the feature is `done`, `not-planned`, or `duplicate`.
No task remains in `ready`, `active`, `needs-review`, or `needs-rework`.

**Verification action:** Call `entity(action: "list", type: "task", parent_feature: "<FEATURE-ID>")`. Confirm every returned task has status `done`, `not-planned`, or `duplicate`.

**Pass criterion:** Zero tasks in non-terminal statuses.
**Fail criterion:** Any task in `ready`, `active`, `needs-review`, or `needs-rework`.

### Item 2: All Changes Committed

**Condition:** `git status` is clean. No uncommitted source files, test files,
workflow state, or temporary artifacts.

**Verification action:** Run `git status --porcelain` in the repository root.
Confirm no output (empty string).

**Pass criterion:** `git status --porcelain` produces zero lines of output.
**Fail criterion:** Any line of output — uncommitted or untracked files exist.

### Item 3: Temporary Files Removed

**Condition:** Scratch scripts, repro files, debug output, or manual test fixtures
used during development are deleted.

**Verification action:** Run `git status --porcelain` and check for untracked files
(`??` prefix). Review each untracked file to determine if it is a temporary artifact.
Also check the worktree root for common temp file patterns (`*.tmp`, `debug_*`,
`scratch_*`, `repro_*`).

**Pass criterion:** No untracked temporary files. Committed files in appropriate
locations are acceptable.
**Fail criterion:** Untracked temporary files present in the worktree.

### Item 4: Tests Pass

**Condition:** `go test ./...` passes on the feature branch and on main after merge.
Suitable new tests exist for the change.

**Verification action:** Run `go test ./...` in the repository root. Record the
exit code and any failure output.

**Pass criterion:** `go test ./...` exits zero with no failures.
**Fail criterion:** Any test failure or non-zero exit code.

### Item 5: Code Reviewed

**Condition:** At minimum one review sub-agent with clean context has been dispatched
(via `orchestrate-review`), findings collated, and no blocking findings remain. A
review document is registered.

**Verification action:** Call `doc(action: "list", owner: "<FEATURE-ID>", type: "report")`. Confirm at least one report-type document exists. Call `entity(action: "get", id: "<FEATURE-ID>")` and confirm the feature reached `reviewing` stage with review completed.

**Pass criterion:** At least one review document registered. Feature passed through
`reviewing` stage with no blocking findings.
**Fail criterion:** No review document found, or feature skipped the `reviewing` stage,
or blocking findings remain unresolved.

### Item 6: Full Lifecycle Advanced

**Condition:** Feature advanced through `developing → reviewing → merging → verifying → done`.
Each transition is an explicit `entity(action: "transition")` call. No stage skipped.

**Verification action:** Call `entity(action: "get", id: "<FEATURE-ID>")`. Check
the status history if available, or the current status. The feature should currently
be in `verifying` status and have passed through `reviewing` and `merging`.

**Pass criterion:** Feature status is `verifying`. Previous transitions through
`reviewing` and `merging` are confirmed.
**Fail criterion:** Feature is not in `verifying`, or lifecycle stages were skipped.

### Item 7: Merge Ancestry Verified

**Condition:** `merge(action: "execute")` succeeded and
`git merge-base --is-ancestor <feature-branch> main` exits zero.

**Verification action:** Determine the feature branch name (from the entity or worktree
record). Run `git merge-base --is-ancestor <branch> main` and check the exit code.
Also run `git branch --merged main | grep <feature-id>` to confirm the branch appears
as merged.

**Pass criterion:** `git merge-base --is-ancestor` exits zero. Branch appears in
`git branch --merged main`.
**Fail criterion:** Non-zero exit from `git merge-base`, or branch not found in
merged list.

### Item 8: Branch Deleted and Verified Absent

**Condition:** `git branch | grep <feature-id>` returns nothing. The branch is gone.

**Verification action:** Run `git branch | grep <feature-id>`. Confirm no output.

**Pass criterion:** Zero lines of output — the feature branch does not exist.
**Fail criterion:** The branch name appears in `git branch` output.

### Item 9: Worktrees Removed

**Condition:** `worktree(action: "remove")` called. `git worktree list` confirms
the worktree directory is gone.

**Verification action:** Run `git worktree list`. Confirm no worktree path contains
the feature ID. Also call `worktree(action: "list", entity_id: "<FEATURE-ID>")`
and confirm no active worktree record exists.

**Pass criterion:** No worktree for this feature in `git worktree list` and no
active worktree record.
**Fail criterion:** Worktree directory still exists or worktree record is still active.

### Item 10: Knowledge Curated and Entities Closed

**Condition:** Tier-2 knowledge entries contributed during the feature are confirmed,
flagged, or retired. Related entities (bugs, decisions) are transitioned to terminal
states.

**Verification action:** Call `knowledge(action: "list", topic_filter: "<FEATURE-ID>")`.
Check that all returned entries are in `confirmed`, `retired`, or `disputed` status
(not `contributed`). Call `entity(action: "list", parent_feature: "<FEATURE-ID>", type: "bug")`
and `entity(action: "list", parent_feature: "<FEATURE-ID>", type: "decision")`.
Confirm all related entities are in terminal states.

**Pass criterion:** No knowledge entries in `contributed` status. All related bugs
and decisions in terminal states.
**Fail criterion:** Unresolved knowledge entries or non-terminal related entities.

## Output Format

Produce exactly this structured output — no preamble, no conversation, no questions:

```
## Close-Out Verification Report

**Feature:** <FEATURE-ID>
**Aggregate Verdict:** all-pass | failures-found (<N> failures)

| # | DoD Item | Status | Evidence |
|---|----------|--------|----------|
| 1 | All tasks terminal | pass / fail | <evidence> |
| 2 | All changes committed | pass / fail | <evidence> |
| 3 | Temporary files removed | pass / fail | <evidence> |
| 4 | Tests pass | pass / fail | <evidence> |
| 5 | Code reviewed | pass / fail | <evidence> |
| 6 | Full lifecycle advanced | pass / fail | <evidence> |
| 7 | Merge ancestry verified | pass / fail | <evidence> |
| 8 | Branch deleted | pass / fail | <evidence> |
| 9 | Worktrees removed | pass / fail | <evidence> |
| 10 | Knowledge curated and entities closed | pass / fail | <evidence> |
```

## Questions This Skill Answers

- Is this feature actually done per the Definition of Done?
- What specifically passed and what failed?
- What evidence exists for each DoD condition?
