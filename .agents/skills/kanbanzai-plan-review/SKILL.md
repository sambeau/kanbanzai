---
name: kanbanzai-plan-review
description: "Use when reviewing a completed plan or phase, verifying feature delivery, checking specification conformance, assessing documentation currency, or writing a plan review report. Activates for plan review, phase review, delivery verification, retrospective assessment, or completion verification. Use even for small plans — the structured review catches gaps that informal checks miss."
metadata:
  kanbanzai-managed: "true"
  version: "0.2.0"
---

# SKILL: Plan Review

## Purpose

Produce a structured review of a completed plan — verifying that all features shipped,
specifications are approved, documentation is current, and the implementation meets its goals.
This SKILL tells a reviewer exactly what to check and in what order.

Plan review is a different activity from feature-level code review (`.skills/code-review.md`).
Code review evaluates implementation quality within a single feature. Plan review evaluates the
aggregate: did the plan deliver what it set out to deliver, and is everything tidy?

## Audience

| Role | How to use this SKILL |
|------|-----------------------|
| **Reviewers** (human or agent) | Follow the procedure step by step |
| **Orchestrators** | Use as the reference for what a plan review covers |
| **Humans** | Understand what will be checked before approving a plan |

## Scope

This SKILL covers plan-level review only. It does **not** cover:

- Feature-level code review (see `.skills/code-review.md`)
- Task-level implementation review
- Retrospective synthesis (see `retro(action: "synthesise")`)

Plan review and feature review are complementary. Feature reviews happen during each feature's
`developing → reviewing → done` lifecycle. Plan review happens after all features are done and
reviews the aggregate.

---

## Inputs

Before starting a plan review, you need:

1. **Plan ID** — the plan to review (e.g., `P10-review-and-doc-currency`).
2. **Plan document** — the implementation plan that defines features, acceptance criteria, and
   sequencing.
3. **Specification document(s)** — the binding contracts for each feature. May be a single spec
   covering the whole plan or per-feature specs.

If any of these are missing or unclear, ask the human before proceeding.

---

## Procedure

### Step 1: Discover plan scope

Call `status(id: "<plan-id>")` to get the full plan dashboard.

This gives you:
- Feature list with current status
- Task status summary
- Associated documents
- Attention items (blocked, stale, or inconsistent entities)

Record the feature list — you will check each one.

### Step 2: Verify feature completion

For each feature in the plan:

1. Confirm the feature is in a terminal state (`done`, `cancelled`, or `superseded`).
2. If any feature is in `needs-rework`, `blocked`, `active`, or `developing` — stop and
   report. The plan is not ready for review.
3. Use `entity(action: "list", type: "task", parent: "<feature-id>")` to check that all tasks
   under each feature are also in terminal state.

Record any features that were `cancelled` or `superseded` — note whether this was intentional
and whether the plan summary should reflect the reduced scope.

### Step 3: Spec conformance

For each feature that reached `done`:

1. Locate the specification document. Check `doc(action: "list", owner: "<plan-id>")` or the
   plan document's feature table for spec references.
2. Read the acceptance criteria table in the spec.
3. Verify each criterion against the implementation. For code changes, read the relevant source
   files. For documentation changes, check that the files exist and contain what the spec
   requires.
4. Record pass/fail per criterion.

If a feature has no spec (e.g., pure documentation work where the plan's acceptance criteria
served as the spec), verify against the plan document's acceptance criteria instead.

### Step 4: Documentation currency

Check that project-level documentation reflects the completed plan:

1. **AGENTS.md Project Status** — does it mention the plan and summarise what was delivered?
2. **AGENTS.md Scope Guard** — does it list the plan as complete?
3. **Spec document status** — are all spec documents in `Approved` status? Check with
   `doc(action: "list", owner: "<plan-id>", status: "draft")` — this should return no results.
4. **SKILL files** — if the plan added or modified SKILL files, check that tool references in
   those files are current (no references to removed or renamed tools).
5. **Bootstrap workflow** — if the plan changes conventions or processes, check that
   `work/bootstrap/bootstrap-workflow.md` is updated.

### Step 5: Cross-cutting checks

1. Run `go test -race ./...` — all tests must pass.
2. Run `health()` — check for new errors or warnings. Pay attention to:
   - Entity consistency warnings (features with non-terminal children, etc.)
   - Knowledge staleness or conflicts
   - Worktree cleanup items
3. Check `git status` — there should be no uncommitted changes from the plan's work.

### Step 6: Retrospective contribution

Before finishing the review, contribute observations about the plan:

- What worked well? (spec quality, test coverage, smooth tooling)
- What caused friction? (unclear specs, stale docs, tooling gaps)
- What would you change for next time?

Contribute signals using `knowledge(action: "contribute")` with tags `["retrospective"]`, or
via the `retrospective` parameter if completing a review task through `finish`.

Do not skip this step. The post-P9 feedback analysis that motivated this SKILL was itself a
demonstration of how valuable review-time observations are — and how easily they're lost when
not captured structurally.

### Step 7: Write the review report

Write findings to `work/reviews/review-<plan-slug>.md` using the format below.

Register the document:

```
doc(action: "register", path: "work/reviews/review-<plan-slug>.md", type: "report", title: "Review: <Plan Title>")
```

---

## Review Report Format

```
# Review: <Plan ID> — <Plan Title>

| Field    | Value                  |
|----------|------------------------|
| Plan     | <plan-id>              |
| Reviewer | <name>                 |
| Date     | <ISO 8601 UTC>         |
| Verdict  | Pass / Pass with findings / Fail |

---

## Summary

<One paragraph: what the plan delivered, overall assessment.>

---

## Feature Status

| Feature | Slug | Status | Spec Conformance |
|---------|------|--------|------------------|
| FEAT-... | ... | done | ✅ All criteria met |
| FEAT-... | ... | done | ⚠️ Minor deviation (see findings) |

---

## Spec Conformance Detail

### Feature: <slug>

| # | Criterion | Result | Notes |
|---|-----------|--------|-------|
| 1 | ...       | ✅     |       |
| 2 | ...       | ⚠️     | ...   |

<Repeat per feature>

---

## Documentation Currency

| Check | Result |
|-------|--------|
| AGENTS.md Project Status | ✅ / ❌ |
| AGENTS.md Scope Guard | ✅ / ❌ |
| Spec documents approved | ✅ / ❌ |
| SKILL files current | ✅ / ❌ / N/A |
| Bootstrap workflow current | ✅ / ❌ / N/A |

---

## Cross-Cutting Checks

| Check | Result |
|-------|--------|
| `go test -race ./...` | ✅ Pass / ❌ Failures |
| `health()` | ✅ Clean / ⚠️ Warnings |
| `git status` clean | ✅ / ❌ |

---

## Findings

| # | Severity | Location | Description |
|---|----------|----------|-------------|
| 1 | Minor/Significant/Critical | file or section | ... |

---

## Verdict

<Final assessment and any conditions for approval.>
```

---

## Verdicts

| Verdict | Meaning |
|---------|---------|
| **Pass** | All checks pass. Plan can transition to `done`. |
| **Pass with findings** | Minor issues found and either fixed during review or tracked for follow-up. No blocking problems. Plan can transition to `done`. |
| **Fail** | Blocking issues found. Plan remains in `reviewing` (or `active` if not yet transitioned). Issues must be resolved before re-review. |

---

## Verification Checklist

Run through this before submitting your review report:

- [ ] I called `status(id: "<plan-id>")` and reviewed the full dashboard.
- [ ] All features are in terminal state.
- [ ] All tasks under each feature are in terminal state.
- [ ] I checked every acceptance criterion in every spec against the implementation.
- [ ] I checked AGENTS.md Project Status and Scope Guard.
- [ ] I checked all spec documents are in Approved status.
- [ ] `go test -race ./...` passes.
- [ ] `health()` shows no new errors.
- [ ] I contributed retrospective observations.
- [ ] The review report is registered as a document record.

---

## Related

- `.skills/code-review.md` — feature-level code review SKILL (different scope, different procedure).
- `work/design/quality-gates-and-review-policy.md` — policy basis for review expectations.
- `work/research/post-p9-feedback-analysis.md` — the feedback analysis that motivated this SKILL.
- `AGENTS.md` — project conventions and reading order.