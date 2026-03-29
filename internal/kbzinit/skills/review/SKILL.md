---
name: kanbanzai-review
description: >
  Use this skill when conducting a code review for a kanbanzai feature. Provides
  structured review procedure, per-dimension evaluation guidance, output format,
  and orchestration steps.
# kanbanzai-managed: true
# kanbanzai-version: dev
---

## Purpose

This skill defines the procedure for conducting a code review for a kanbanzai feature.
It covers required inputs, per-dimension evaluation guidance, output format, finding
classification, edge cases, and the full orchestration sequence.

---

## Required Inputs

Before beginning a review, confirm all of the following are available:

- **Feature ID** — the FEAT-... identifier of the feature under review.
- **Spec document(s)** — all specification documents in `approved` status associated
  with the feature. Retrieve with `doc(action: "list", owner: "FEAT-...")`.
- **Worktree branch or PR diff** — the branch name from `worktree` or `branch`, or an
  open pull request diff accessible via `pr`.
- **Dev-plan document** — the dev-plan document associated with the feature, registered
  under the feature or its parent plan.

If any input is missing, consult **Edge Cases** before proceeding.

---

## Review Dimensions

Evaluate the implementation across five dimensions.

### 1. Spec Conformance

Verify that the implementation satisfies every acceptance criterion in the approved
spec document(s):

- For each AC: is it addressed? Is it addressed correctly and completely?
- Note any ACs that are partially satisfied or absent from the implementation.
- Check for scope creep — changes that go beyond what the spec requires are a finding.
- Where multiple spec documents exist, all must be checked.

### 2. Implementation Quality

Assess the code changes for correctness and maintainability:

- No logic errors or incorrect assumptions visible in the diff.
- Error handling is present and appropriate for the failure modes the code can encounter.
- No unnecessary complexity introduced beyond what the spec requires.
- Code follows the conventions visible in the surrounding codebase (naming, structure,
  error patterns, test style).
- No leftover debug output, commented-out code, placeholder values, or TODO comments
  that were not present before this feature.

### 3. Test Adequacy

Assess whether tests are sufficient to validate the implementation:

- New logic has corresponding tests.
- Tests assert meaningful outcomes — not merely that code runs without error.
- Edge cases identified in the spec are covered by tests.
- Tests are independent and do not rely on external mutable state.
- No test is trivially green (e.g. an assertion that always passes regardless of
  the code under test).

### 4. Documentation Currency

Verify that documentation is consistent with the implementation:

- Registered documents associated with the feature are in `approved` status, or a
  refresh has been recorded since the last edit.
- The dev-plan document reflects the final task breakdown; no task remains `active`
  without documented justification.
- AGENTS.md Scope Guard is consistent with the feature's scope if the parent plan
  is complete or nearing completion.

### 5. Workflow Integrity

Verify that the entity and document record state is clean:

- All tasks under the feature are in `done` or `cancelled` status; none remain in
  `ready` or `active` without documented justification.
- The feature is in `reviewing` status at the time this review is conducted.
- The feature's worktree is associated with the correct branch.
- No unrelated changes are present in the diff.

---

## Output Format

Produce a structured review report with the following sections.

### Per-Dimension Outcomes

For each of the five dimensions, record one of:

- **pass** — fully satisfied; no findings.
- **partial** — partially satisfied; list what is missing or incomplete.
- **fail** — not satisfied; one or more blocking findings are present.

### Overall Verdict

Choose exactly one:

- `approved` — all dimensions pass or partial with only non-blocking findings;
  the feature may be merged.
- `approved-with-notes` — all blocking criteria pass; one or more non-blocking
  findings are documented for the implementing team.
- `changes-required` — one or more blocking findings exist; the feature must not
  be merged until they are resolved.

### Findings

Separate all findings into two groups:

**Blocking findings** — must be resolved before the feature may be merged to main.

> **Finding classification rule:** A finding is blocking if it would prevent a safe,
> correct merge to main. This includes: failing acceptance criteria, logic errors,
> missing tests for required behaviour, unapproved required documents, and invalid
> entity state. All other findings are non-blocking.

For each blocking finding, record:
- Dimension
- Description of the problem
- Suggested resolution (when known)

**Non-blocking findings** — should be addressed but do not prevent merge. For each,
record the dimension and a description.

---

## Edge Cases

### Feature with No Spec

If no approved specification document exists for the feature:

1. Check whether the feature was intentionally scoped without a spec (for example,
   a small improvement delivered under a design-only flow, or a targeted bug fix).
2. If no spec is expected: note this explicitly in the review report and evaluate
   the feature against the approved design document instead, treating each design
   requirement as an acceptance criterion.
3. If a spec was expected but is missing or still in `draft`: record a **blocking
   finding** — "No approved specification document found for this feature."

### Partial Spec Satisfaction

If the implementation satisfies some but not all acceptance criteria:

1. List every unsatisfied or partially satisfied AC as a separate finding.
2. Classify each finding: blocking if the AC is part of the feature's core contract
   or is not explicitly marked as optional or deferred; non-blocking otherwise.
3. If the majority of ACs are unsatisfied, the overall verdict must be
   `changes-required` regardless of individual finding classifications.

---

## Review Procedure

The steps below form the complete orchestration sequence for conducting a feature review.

### Step 1: Assemble Context

```
status(id: "FEAT-...")                           # confirm feature is in reviewing state
entity(action: "get", id: "FEAT-...")            # get feature details and parent plan
doc(action: "list", owner: "FEAT-...")           # list all documents owned by the feature
worktree(action: "get", entity_id: "FEAT-...")   # get the associated worktree and branch
branch(action: "status", entity_id: "FEAT-...")  # check branch staleness and drift
```

Confirm all required inputs are present before proceeding. If a required input is
missing, apply the relevant edge case rule above, or pause and ask the human.

### Step 2: Dispatch Review Sub-Agents

For large or complex features, dispatch one sub-agent per dimension to parallelise
the review:

```
handoff(task_id: "TASK-...")
```

Pass each sub-agent the assembled context and the specific dimension it is responsible
for evaluating.

For small features, a single agent may evaluate all five dimensions sequentially.

Before dispatching parallel sub-agents, verify there is no file overlap:

```
conflict(action: "check", task_ids: ["TASK-...", "TASK-..."])
```

### Step 3: Evaluate Each Dimension

Work through all five dimensions using the guidance above.

Inspect the implementation using the branch diff or the open pull request:

```
pr(action: "status", entity_id: "FEAT-...")
```

Verify workflow integrity:

```
entity(action: "list", type: "task", parent: "FEAT-...")
health()
```

Record findings as you go. Do not defer — capture each finding immediately with its
dimension and classification.

### Step 4: Collect Results and Write Report

After all dimensions are evaluated, synthesise the findings into a structured review
report. Write the report to:

    work/review/review-{feature-id}-{slug}.md

where `{feature-id}` is the full FEAT-... identifier (e.g. `FEAT-01AB...`) and
`{slug}` is the feature's slug field.

Register and approve the report document immediately after writing it:

```
doc(action: "register",
    path: "work/review/review-{feature-id}-{slug}.md",
    type: "report",
    title: "Review: {feature title}",
    owner: "FEAT-...")
doc(action: "approve", id: "DOC-...")
```

### Step 5: Transition the Feature

Based on the overall verdict:

**`approved` or `approved-with-notes`** — transition the feature to `done` and merge:

```
entity(action: "transition", id: "FEAT-...", status: "done")
merge(action: "execute", entity_id: "FEAT-...")
```

**`changes-required`** — transition the feature to needs-rework:

```
entity(action: "transition", id: "FEAT-...", status: "needs-rework")
```

Contribute a knowledge entry summarising the blocking findings so the implementing
agent has clear context when rework begins:

```
knowledge(action: "contribute",
    topic: "rework-findings-{feature-id}",
    content: "Blocking findings from review: ...",
    scope: "project")
```

---

## Related

- `kanbanzai-workflow` — feature lifecycle states, stage gates, and the
  reviewing/needs-rework cycle
- `kanbanzai-plan-review` — plan-level review procedure, run after all features
  in a plan reach a terminal state
- `kanbanzai-documents` — document registration, approval, drift, and supersession