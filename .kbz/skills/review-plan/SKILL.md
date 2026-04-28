---
name: review-plan
description:
  expert: "Batch-level conformance review verifying feature delivery status,
    specification approval audit, and documentation currency across all
    features within a batch"
  natural: "Check whether all the work in a batch is actually done — features
    shipped, specs approved, docs up to date — and produce a structured
    report of what's complete and what's not"
triggers:
  - review batch completion
  - check batch delivery status
  - verify batch readiness
  - audit batch conformance
roles: [reviewer-conformance]
stage: batch-reviewing
constraint_level: low
---

## Vocabulary

- **feature delivery status** — the lifecycle state of a feature within the batch (done,
  cancelled, superseded, or still in progress); terminal states are the only acceptable
  outcomes for a completed batch
- **specification approval status** — whether the specification document backing a feature
  has been formally approved; draft or superseded specs indicate incomplete governance
- **documentation currency** — whether project-level documentation (AGENTS.md, bootstrap
  workflow, SKILL files) reflects the work delivered by the batch
- **batch completeness** — the aggregate assessment of whether all features, specs, and
  documentation meet their terminal-state requirements
- **conformance gap** — a specific discrepancy between expected terminal state and actual
  state for any batch artifact (feature, spec, or document)
- **delivery verification** — the act of checking each feature against its expected
  terminal state, distinct from evaluating implementation quality
- **terminal state** — a lifecycle state from which no further work is expected (done,
  cancelled, superseded for features; approved for specs)
- **batch verdict** — the aggregate pass/fail assessment produced by this review
- **scope guard** — the section of project documentation that tracks which batches are
  complete and which are active
- **feature census** — the enumeration of all features within a batch, including their
  lifecycle states and document statuses

## Anti-Patterns

### Rubber-Stamp Plan Review

- **Detect:** The review concludes "all features are done" without enumerating feature
  counts, verifying spec statuses, or checking documentation currency
- **BECAUSE:** A non-enumerated pass verdict is indistinguishable from a skipped review.
  It provides no evidence that the reviewer actually examined the batch's artifacts.
- **Resolve:** Produce a full feature census. Verify each feature's status and spec
  approval individually. Check documentation currency against the batch's scope.

### Scope Confusion

- **Detect:** The review includes features from a different batch, or evaluates features
  against criteria not established during the batch's specification
- **BECAUSE:** Each batch has a defined set of features with approved specifications.
  Reviewing against unapproved criteria or including out-of-scope features invalidates
  the conformance assessment and creates ungoverned scope.
- **Resolve:** Derive the feature list exclusively from the batch's entity records.
  Evaluate each feature against its own approved specification only.

### Silent Scope Reduction

- **Detect:** Features are marked as delivery-complete without verifying that their specs
  were actually implemented, or specs were silently cut during implementation
- **BECAUSE:** A feature can reach `done` status without fulfilling all its requirements
  if the spec was never checked during review. This creates a gap between what was
  specified and what was delivered that the review is supposed to catch.
- **Resolve:** For each feature, verify that the implementation covers the approved spec's
  requirements. If scope was cut, it should be visible as a cancelled or superseded
  feature, not hidden inside a done feature.

## Prerequisites

Before starting this review, confirm:

1. The batch entity exists and its feature list is stable — no new features are being
   added during review
2. All features in the batch have reachable entity records
3. If a report from a prior review cycle exists, read it to scope the re-review:
   only re-examine the findings from the prior review rather than re-reviewing every
   feature from scratch

## Checklist

Copy this checklist and track your progress:
- [ ] Enumerated all features in this batch
- [ ] Verified each feature's delivery status
- [ ] Verified each feature's spec approval status
- [ ] Checked project documentation currency
- [ ] Contributed retrospective observations
- [ ] Wrote and registered review report

## Procedure

### Step 1: Enumerate batch scope

1. Call `entity(action: "list", type: "feature", parent: "B<n>-<slug>")`. Confirm the
   feature list matches expectations — no extra features, none missing.
2. For each feature, record: ID, name, current lifecycle status, spec document ID (if
   applicable), and dev-plan document ID (if applicable).
3. Cross-reference against the batch's dev-plan or scope document. Flag any feature that
   appears in the document but not in the entity list, or vice versa, as a conformance
   gap.

### Step 2: Verify feature delivery

1. For each feature in the batch:
   - IF the feature is `done`: verify all tasks under it are terminal (`done`,
     `cancelled`, `duplicate`). If the dev-plan specified certain acceptance criteria,
     verify they are satisfied by the implementation.
   - IF the feature is `cancelled` or `superseded`: confirm the decision log exists and
     the rationale is documented. If not, flag as a conformance gap.
   - IF the feature is NOT in a terminal state: flag as incomplete. The batch cannot be
     conformance-passed if any feature is still in development.
2. If any feature violated its dependency constraints (e.g., a dependent feature was
   merged before its dependency), flag as a conformance gap.

### Step 3: Verify spec conformance

For each feature that is `done`, verify:

1. The specification document exists and is `approved` — not `draft` or `superseded`.
2. The acceptance criteria in the spec are satisfied by the implementation.
3. The design document referenced by the spec is `approved` (design-approval propagates
   through the spec gate; an unapproved design behind an approved spec is a governance
   hole).

### Step 4: Check documentation currency

1. Check whether project-level documentation (`AGENTS.md`, workflow skills, reference
   files) references the features delivered by this batch. If the documentation describes
   behaviour that contradicts what was implemented, flag as a documentation gap.
2. Verify that any knowledge entries contributed during the batch's features are confirmed
   (not still `contributed`).
3. Check whether new capabilities delivered by this batch would change existing skills
   or procedures. If so, flag for documented future work.

### Step 5: Cross-cutting checks

1. Run `health()` and verify no errors or warnings related to the batch's entities.
2. Check for orphaned worktrees (`worktree(action: "list", status: "active")`) that
   belong to features in this batch — they should be `merged` or `abandoned`.
3. If the batch had an advance plan with cohorts, verify all cohort merge checkpoints
   were confirmed clean.

### Step 6: Retrospective contribution

Call `retro(action: "synthesise", scope: "B<n>-<slug>")` to surface the batch's
workflow signals — what worked, what didn't, what to improve. Include the synthesised
findings in the review report.

Do not skip this step. Retrospective synthesis is the primary mechanism for turning
per-task observations into actionable project-level improvements.

### Step 7: Write and register review report

1. Compile all findings into a review report.
2. Register the report: `doc(action: "register", path: "work/reviews/batch-review-<slug>-<date>.md", type: "report", owner: "B<n>-<slug>")`.
3. Await human approval. The conformance review has a human gate — the report must be
   approved before the batch can be closed.

## Output Format

```
# Batch Conformance Review: B<n>-<slug>

## Scope
- Batch: B<n>-<slug>
- Features: N total (N done, N cancelled/superseded, N incomplete)
- Review date: YYYY-MM-DD
- Reviewer: reviewer-conformance

## Feature Census
| Feature | Status | Spec Approved | Dev-Plan | Notes |
|---------|--------|---------------|----------|-------|
| FEAT-xxx | done | yes | yes | All 5 tasks done |
| FEAT-xxx | done | yes | yes | 2 known spec gaps (non-blocking) |
| FEAT-xxx | done | no | yes | **Spec is draft** — blocking |

## Conformance Gaps
| # | Feature | Type | Description | Severity |
|---|---------|------|-------------|----------|
| CG-1 | FEAT-xxx | spec-status | Spec FEAT-xxx is draft, not approved | blocking |
| CG-2 | FEAT-xxx | doc-currency | AGENTS.md references old terminology | non-blocking |

## Documentation Currency
- AGENTS.md: [current / needs update]
- Workflow skills: [current / needs update]
- Knowledge entries: N contributed, N confirmed, N flagged

## Retrospective Summary
<2-3 sentence synthesis from retro synthesise>

## Batch Verdict
<pass | pass-with-notes | fail>

## Evidence
- Batch entity: entity(action: "get", id: "B<n>-<slug>")
- Feature list: entity(action: "list", type: "feature", parent: "B<n>-<slug>")
- Health check: health()
- Retro synthesis: retro(action: "synthesise", scope: "B<n>-<slug>")
```

## Examples

### BAD: Rubber-stamp batch review

```
Feature: Webhooks
Status: done (appears done)

**Problem:** The reviewer accepted "done" at face value without checking
feature statuses, spec approvals, or documentation currency. This is indistinguishable
from skipping the review — no conformance assessment was actually performed.
```

### GOOD: Structured conformance review with gap

```
# Batch Conformance Review: B42-webhook-system

## Scope
- Batch: B42-webhook-system
- Features: 3 total (2 done, 0 cancelled, 1 incomplete)
- Review date: 2026-03-15
- Reviewer: reviewer-conformance

## Feature Census
| Feature | Status | Spec Approved | Notes |
|---------|--------|---------------|-------|
| FEAT-050 | done | yes | All 4 tasks done ✓ |
| FEAT-051 | done | yes | All 3 tasks done ✓ |
| FEAT-052 | developing | no | Feature not in terminal state — blocking |

## Batch Verdict
fail — FEAT-052 is still developing

## Evidence
- Batch entity: entity(action: "get", id: "B42-webhook-system") → 2/3 features done
- Feature list: entity(action: "list", type: "feature", parent: "B42-webhook-system")
```

## Evaluation Criteria

| # | Criterion | Weight |
|---|----------|--------|
| 1 | Every feature in the batch is enumerated and its status verified individually | required |
| 2 | Spec approval is checked for every done feature | required |
| 3 | Documentation currency is checked against the batch's scope | high |
| 4 | Retrospective synthesis is included in the review report | high |
| 5 | The review report is registered as a document | required |
| 6 | Conformance gaps are classified by severity (blocking vs non-blocking) | required |

## Questions This Skill Answers

- How do I know if a batch is complete?
- What counts as a conformance gap?
- How do I verify whether a feature's specification was actually implemented?
- What documentation should I check for currency?
- How do I write a batch conformance review report?

## Related

- `review-code` — checks individual features against their specifications
- `orchestrate-development` — generates the work this review examines
- `kanbanzai-workflow` — lifecycle transitions and stage gates
