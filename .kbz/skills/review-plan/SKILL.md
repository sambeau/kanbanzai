---
name: review-plan
description:
  expert: "Plan-level conformance review verifying feature delivery status,
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
- **scope guard** — the section of project documentation that tracks which plans are
  complete and which are active
- **feature census** — the enumeration of all features within a batch, including their
  current status and any scope changes (cancellation, supersession)
- **spec audit trail** — the chain of document records showing specification registration,
  approval, and any supersession events
- **documentation drift** — the gap between what the plan delivered and what project
  documentation describes, typically caused by updates made during development that
  were not propagated to aggregate docs
- **conformance-only review** — a review that verifies completeness and approval status
  without evaluating code quality, security, or implementation approach
- **batch scope reduction** — features that were cancelled or superseded during execution,
  which must be acknowledged in the review rather than silently ignored
- **stale document** — a document whose content no longer reflects the current state of
  the codebase or workflow it describes

## Anti-Patterns

### Rubber-Stamp Batch Review

- **Detect:** Plan verdict is "pass" without evidence of checking each feature's status
  individually, or the review report lists features without per-feature verification
- **BECAUSE:** Plan-level rubber-stamping hides incomplete features behind an aggregate
  "looks done" assessment; a single unverified feature can leave the batch in an
  inconsistent state where the batch is marked done but work remains open
- **Resolve:** Check each feature individually using `status()` or `entity()` calls;
  record per-feature status in the review output; every feature must have an explicit
  terminal-state confirmation or a documented conformance gap

### Scope Confusion

- **Detect:** The review evaluates code quality, security posture, test adequacy, or
  implementation approach instead of (or in addition to) batch conformance
- **BECAUSE:** Batch review and code review are different activities with different
  evaluation criteria; batch review checks "did we ship everything we said we would?"
  while code review checks "is what we shipped correct?"; mixing them produces a
  review that does neither well
- **Resolve:** Restrict evaluation to delivery status, spec approval status, and
  documentation currency; if code quality concerns surface during batch review, note
  them as out-of-scope observations but do not let them influence the batch verdict

### Silent Scope Reduction

- **Detect:** Cancelled or superseded features are omitted from the review report
  entirely, or counted as "done" without noting the scope change
- **BECAUSE:** Scope reduction is a legitimate outcome but must be explicitly
  acknowledged; silently dropping features from the census makes the review report
  an inaccurate record and hides planning gaps from retrospective analysis
- **Resolve:** Include every feature from the original plan in the feature census;
  mark cancelled/superseded features with their actual status and note whether the
  scope change was intentional

## Prerequisites

Before starting a batch review, verify these inputs are available:

1. **Plan ID** — the plan to review (e.g., `P10-review-and-doc-currency`).
2. **Plan document** — the implementation plan that defines features, acceptance
   criteria, and sequencing.
3. **Specification document(s)** — the binding contracts for each feature. May be
   a single spec covering the whole plan or per-feature specs.
4. **All features in terminal state** — features must be in done, cancelled, or
   superseded state. If any feature is still active, the plan is not ready for
   delivery verification.

IF any of these are missing or unclear → STOP. Ask before proceeding. Do not
begin a batch review with incomplete inputs.

## Checklist

```
Copy this checklist and track your progress:
- [ ] Retrieved plan dashboard via status()
- [ ] Enumerated all features in the plan (feature census)
- [ ] Verified each feature is in a terminal state
- [ ] Verified all tasks under each feature are in terminal state
- [ ] Checked specification approval status for each feature
- [ ] Verified each acceptance criterion against the implementation (spec conformance)
- [ ] Checked documentation currency (AGENTS.md, scope guard, SKILL files)
- [ ] Ran cross-cutting checks (go test, health, git status)
- [ ] Contributed retrospective observations
- [ ] Recorded all conformance gaps
- [ ] Wrote review report and registered as document record
```

## Procedure

### Step 1: Enumerate plan scope

1. Call `status(id: "<plan-id>")` to retrieve the full plan dashboard.
2. Record the complete feature census — every feature in the plan, including
   its current lifecycle state.
3. IF the plan dashboard is unavailable or returns errors → STOP. Report the
   missing context. Do not proceed with partial information.

### Step 2: Verify feature delivery

For each feature in the census:

1. Confirm the feature is in a terminal state (done, cancelled, or superseded).
2. IF any feature is in a non-terminal state (active, developing, reviewing,
   needs-rework, blocked) → record a conformance gap. The plan is not ready
   for completion.
3. For each feature in terminal state, check that all tasks under the feature
   are also in terminal state. Use `entity(action: "list", type: "task",
   parent: "<feature-id>")` if the dashboard does not show task-level detail.
4. For cancelled or superseded features, note the scope reduction explicitly.
   Do not silently omit them from the census.

### Step 3: Verify spec conformance

For each feature that reached done:

1. Locate the specification document via the plan document or
   `doc(action: "list", owner: "<plan-id>")`.
2. Confirm the spec is in approved status. IF any spec is in draft status →
   record a conformance gap.
3. Read the acceptance criteria from the spec document. Use
   `doc(action: "content", id: "<spec-doc-id>")` or read the file directly.
4. Verify each criterion against the actual implementation. For code changes,
   read the relevant source files. For documentation changes, check that the
   files exist and contain what the spec requires. Do not treat approval
   status alone as proof of conformance — check the work itself.
5. Record pass/fail per criterion using the conformance table:

   | # | Criterion | Result | Notes |
   |---|-----------|--------|-------|
   | 1 | ...       | ✅     |       |
   | 2 | ...       | ⚠️     | ...   |

6. IF a feature has no associated spec (e.g., documentation-only work where
   the plan's acceptance criteria served as the spec), verify against the
   plan document's criteria instead and note this explicitly.

### Step 4: Check documentation currency

1. Check that project documentation reflects what the plan delivered:
   - AGENTS.md project status and scope guard sections
   - Specification documents in approved status (check with
     `doc(action: "list", owner: "<plan-id>", status: "draft")` — should
     return no results)
   - SKILL files (if the plan added or modified any)
   - Bootstrap workflow (if the plan changed conventions)
2. IF documentation does not reflect the delivered work → record a
   conformance gap with the specific document and section that needs updating.

### Step 5: Cross-cutting checks

Run three specific cross-cutting checks:

1. **Test suite** — run `go test -race ./...`. All tests must pass. Record
   any failures as conformance gaps.
2. **System health** — call `health()`. Check for new errors or warnings:
   entity consistency issues, knowledge staleness, worktree cleanup items.
   New warnings introduced by the plan's work are conformance gaps.
3. **Clean working tree** — run `git status`. There should be no uncommitted
   changes from the plan's work. Uncommitted changes indicate incomplete
   delivery.

### Step 6: Retrospective contribution

Before writing the review report, contribute observations about the plan:

- What worked well? (spec quality, test coverage, smooth tooling)
- What caused friction? (unclear specs, stale docs, tooling gaps,
  documentation drift)
- What would you change for next time?

Contribute signals using `finish(retrospective: [...])` when completing a
review task, or `knowledge(action: "contribute", tags: ["retrospective"])`
for standalone observations.

Do not skip this step. Retrospective signals captured at review time are
the primary input for process improvement.

### Step 7: Write and register review report

1. IF any conformance gaps were recorded → the batch verdict is fail or
   pass with findings, depending on severity.
2. IF the plan state is contradictory (e.g., features reference specs that
   don't exist, or the dashboard shows inconsistencies) → STOP. Report the
   contradiction. Do not produce a verdict on contradictory data.
3. Write findings to `work/reviews/review-<plan-slug>.md` using the output
   format below.
4. Register the document:
   `doc(action: "register", path: "work/reviews/review-<plan-slug>.md", type: "report", title: "Review: <Plan Title>")`

## Output Format

```
# Plan Review: <plan-id> — <plan-title>

| Field    | Value          |
|----------|----------------|
| Plan     | <plan-id>      |
| Reviewer | <name>         |
| Date     | <ISO 8601 UTC> |
| Verdict  | Pass / Pass with findings / Fail |

## Feature Census

| Feature   | Slug | Status     | Terminal | Notes              |
|-----------|------|------------|----------|--------------------|
| FEAT-...  | ...  | done       | ✅       |                    |
| FEAT-...  | ...  | cancelled  | ✅       | Scope reduction: <reason> |
| FEAT-...  | ...  | developing | ❌       | Conformance gap    |

## Specification Approval

| Feature   | Spec Document          | Status   |
|-----------|------------------------|----------|
| FEAT-...  | work/spec/...          | approved ✅ |
| FEAT-...  | work/spec/...          | draft ❌    |
| FEAT-...  | (plan criteria used)   | N/A      |

## Spec Conformance Detail

### Feature: <slug>

| # | Criterion | Result | Notes |
|---|-----------|--------|-------|
| 1 | ...       | ✅     |       |
| 2 | ...       | ⚠️     | ...   |

<Repeat per feature>

## Documentation Currency

| Check                      | Result    | Notes |
|----------------------------|-----------|-------|
| AGENTS.md project status   | ✅ / ❌  |       |
| AGENTS.md scope guard      | ✅ / ❌  |       |
| Spec documents approved    | ✅ / ❌  |       |
| SKILL files current        | ✅ / ❌ / N/A |  |
| Bootstrap workflow current | ✅ / ❌ / N/A |  |

## Cross-Cutting Checks

| Check | Result |
|-------|--------|
| `go test -race ./...` | ✅ Pass / ❌ Failures |
| `health()` | ✅ Clean / ⚠️ Warnings |
| `git status` clean | ✅ / ❌ |

## Conformance Gaps

| # | Category      | Location             | Description          |
|---|---------------|----------------------|----------------------|
| 1 | feature-status / spec-approval / documentation | ... | ... |

## Verdict

<Final assessment. Conditions for approval if verdict is not Pass.>
```

## Examples

### BAD: Rubber-stamp batch review

```
Plan P8 looks complete. All features appear to be done.
The code quality is good and tests are passing.
Verdict: Pass.
```

WHY BAD: No feature census — "all features appear to be done" without listing
them individually. No spec approval check. No documentation currency check.
Includes code quality commentary (scope confusion — that belongs in code review,
not batch review). No structured output. A reader cannot verify what was actually
checked.

### GOOD: Structured conformance review with gap

```
# Plan Review: P8-skills-system — Skills System Redesign

| Field    | Value                  |
|----------|------------------------|
| Plan     | P8-skills-system       |
| Reviewer | agent-reviewer         |
| Date     | 2025-07-14T10:30:00Z   |
| Verdict  | Pass with findings     |

## Feature Census

| Feature   | Slug              | Status | Terminal | Notes |
|-----------|-------------------|--------|----------|-------|
| FEAT-041  | skill-format      | done   | ✅       |       |
| FEAT-042  | role-format       | done   | ✅       |       |
| FEAT-043  | binding-registry  | done   | ✅       |       |
| FEAT-044  | context-assembly  | cancelled | ✅  | Scope reduction: deferred to P9 per decision DEC-012 |

## Specification Approval

| Feature   | Spec Document                       | Status      |
|-----------|-------------------------------------|-------------|
| FEAT-041  | work/spec/skills-system-spec-v2.md  | approved ✅ |
| FEAT-042  | work/spec/skills-system-spec-v2.md  | approved ✅ |
| FEAT-043  | work/spec/skills-system-spec-v2.md  | approved ✅ |
| FEAT-044  | (cancelled — no spec required)      | N/A         |

## Documentation Currency

| Check                      | Result | Notes |
|----------------------------|--------|-------|
| AGENTS.md project status   | ✅     | Updated in commit abc1234 |
| AGENTS.md scope guard      | ⚠️     | Lists P8 as active, not complete |
| Spec documents approved    | ✅     |       |
| SKILL files current        | ✅     | New SKILL files verified |
| Bootstrap workflow current | N/A    | No convention changes |

## Conformance Gaps

| # | Category      | Location                  | Description |
|---|---------------|---------------------------|-------------|
| 1 | documentation | AGENTS.md scope guard     | P8 still listed as active; should be marked complete |

## Verdict

Pass with findings. One documentation gap: AGENTS.md scope guard
needs updating to reflect P8 completion. All features verified
individually. FEAT-044 cancellation documented with decision reference.
```

WHY GOOD: Every feature checked individually with explicit terminal-state
confirmation. Cancelled feature acknowledged with decision reference instead
of silently omitted. Spec approval verified per feature. Documentation currency
checked with specific gap identified. Structured output that a reader can verify
claim by claim. Conformance-focused — no code quality commentary.

## Evaluation Criteria

1. Does the review include a complete feature census with per-feature status?
   Weight: required.
2. Is every feature individually verified against terminal state?
   Weight: required.
3. Does the review check specification approval status for each feature?
   Weight: required.
4. Does the review verify each acceptance criterion against the implementation
   (not just approval status)? Weight: required.
5. Does the review check documentation currency?
   Weight: high.
6. Does the review include cross-cutting checks (tests, health, git status)?
   Weight: high.
7. Are cancelled or superseded features explicitly acknowledged with reasons?
   Weight: high.
8. Does the review stay conformance-focused without scope creep into code quality?
   Weight: high.
9. Does the review include retrospective observations?
   Weight: medium.
10. Is the review report registered as a document record?
    Weight: medium.
11. Can a reader verify each claim in the review without re-running the checks?
    Weight: medium.

## Questions This Skill Answers

- How do I review a plan for completion?
- What should I check before closing a plan?
- How do I verify all specs in a plan are approved?
- How do I check spec conformance criterion by criterion?
- What is the difference between batch review and code review?
- How do I handle cancelled features in a batch review?
- What documentation should I check during batch review?
- What does a conformance gap look like in a batch review?
- When should I stop a batch review because of contradictory state?
- What cross-cutting checks should I run during batch review?
- How do I contribute retrospective observations during review?
- Where do I write and register the review report?