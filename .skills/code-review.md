# SKILL: Code Review

## Purpose

Produce structured review findings for a single **review unit** — a bounded set of files and
associated specification sections. This SKILL tells a review sub-agent exactly what to check,
how to evaluate it, and how to format the output.

## Audience

| Role | How to use this SKILL |
|------|-----------------------|
| **Review sub-agents** | Follow the procedure step by step to produce findings |
| **Orchestrators** | Reference for what each sub-agent will produce and how to interpret results |
| **Humans** | Understand what agents will check and what the structured output means |

## Scope Exclusion

This SKILL covers the sub-agent perspective only — how a single review sub-agent evaluates
its assigned review unit and formats its output.

**This SKILL does not cover orchestration.** Orchestration — decomposing a feature into review
units, dispatching sub-agents in parallel, collating findings, creating remediation tasks, and
managing lifecycle transitions — is handled by the orchestrator agent using a separate SKILL.

---

## Inputs

A review sub-agent receives the following inputs before starting:

1. **Context packet** — assembled via `context_assemble(role="reviewer")`. Contains the reviewer
   context profile, relevant knowledge entries, and project conventions.

2. **File list** — the specific source files that form this review unit. Read these files to
   evaluate the implementation.

3. **Spec section(s)** — one or more sections from the feature's specification document,
   retrieved via `doc_section`. These are the binding requirements for this review unit.

4. **Review profile** — the named profile to apply (e.g., Feature Implementation, Bugfix,
   Lightweight). If not explicitly specified, default to Feature Implementation Review Profile.
   See `work/design/quality-gates-and-review-policy.md` §5 for profile definitions.

5. **Review unit descriptor** — a short label identifying what this unit covers (e.g.,
   `service-layer`, `storage-layer`, `tests`, `documentation`). Include this in your output.

---

## Procedure

### Step 1: Orient from inputs

Before evaluating anything:

1. Read the spec section(s) fully. Understand what was required.
2. Read all files in the file list. Understand what was implemented.
3. Note the review profile — this determines which dimensions are required.
4. If any input is missing or inaccessible, go to **Edge Case: Missing Context** before
   proceeding.

### Step 2: Evaluate each dimension independently

Evaluate each required dimension in order (see **Per-Dimension Guidance** below). For each
dimension:

- Work through the specific questions for that dimension.
- Record a per-dimension outcome: `pass`, `pass_with_notes`, `concern`, `fail`, or
  `not_applicable`.
- List individual findings, each classified as `blocking` or `non_blocking`.

Evaluate dimensions independently. Do not let a poor result in one dimension colour your
assessment of another.

### Step 3: Classify each finding

For every finding, determine whether it is **blocking** or **non-blocking** using the rules in
**Finding Classification** below.

Blocking findings drive the overall verdict. Non-blocking findings are recorded but do not
prevent approval.

### Step 4: Write structured output

Produce the complete structured output as defined in **Structured Output Format** below. Do not
summarise in prose — the structure is required.

---

## Per-Dimension Guidance

### Dimension 1: Specification Conformance

**Outcome** — Does the implementation match the approved specification for this review unit?

Questions to answer:
- Was everything required by the spec section(s) implemented?
- Was anything materially omitted or skipped?
- Was anything added that conflicts with the specification?
- Was any behavior changed without a corresponding decision or spec update?
- If the implementation deviates from the spec, is the deviation explicitly recorded and
  justified?

**Pass:** All requirements are satisfied. Any minor differences are clearly intentional and
justified.

**Pass with notes:** Requirements are satisfied but with small gaps or unexplained deviations
that should be noted.

**Concern:** A deviation or omission exists but may be intentional — insufficient information
to call it a failure. Raise it explicitly.

**Fail:** One or more requirements are clearly not met, or the implementation contradicts
the specification without justification.

**Not applicable:** This review unit has no linked specification. Go to **Edge Case: Missing
Spec**.

> This is the most important dimension for feature work. When uncertain, lean toward `concern`
> rather than `fail` — but do not use `concern` to avoid raising a real defect.

---

### Dimension 2: Implementation Quality

**Outcome** — Is the code correct, idiomatic, and maintainable?

Questions to answer:
- Is the solution appropriate to the task (not over-engineered, not under-engineered)?
- Is the code idiomatic for the language, framework, and codebase conventions?
- Is the structure understandable and maintainable?
- Are abstractions appropriately sized?
- Is naming clear and consistent with the rest of the codebase?
- Are comments present where logic is not self-evident, and absent where code is obvious?
- Does the implementation introduce unnecessary complexity?

**Pass:** Code is correct, clear, idiomatic, and maintainable.

**Pass with notes:** Code is functionally correct but has style or clarity improvements worth
making.

**Concern:** Code may be correct but has structural or complexity issues that could cause
problems.

**Fail:** Code is incorrect, clearly non-idiomatic, or unmaintainable in a way that requires
remediation.

> "High quality" means clear, correct, idiomatic, and maintainable — not maximally clever.

---

### Dimension 3: Test Adequacy

**Outcome** — Are tests appropriate, sufficient, and well-structured?

Questions to answer:
- Does the change have appropriate automated test coverage?
- Do the tests meaningfully verify behavior (not just existence)?
- Do the tests match the relevant acceptance criteria from the spec?
- Are important edge cases or regressions covered?
- Are tests placed at the correct level (unit, integration, end-to-end)?
- Are tests focused, readable, and maintainable?
- Does any test appear brittle, redundant, or misleading?

**Pass:** Tests are appropriate and sufficient for this review unit.

**Pass with notes:** Tests are present and mostly adequate; minor gaps noted.

**Concern:** Test coverage appears incomplete in a way that might mask defects.

**Fail:** Critical paths are untested, or tests do not verify the behavior they claim to.

**Not applicable:** The change is purely documentary or configurational and does not require
automated tests.

> The goal is appropriate verification, not maximum test count. It is valid to conclude that
> no additional tests are required for a particular unit.

---

### Dimension 4: Documentation Currency

**Outcome** — Is documentation accurate and up to date?

Questions to answer:
- Is user-facing documentation updated where behavior changed?
- Is developer-facing documentation (e.g., `AGENTS.md`, design docs) updated where affected?
- Do any references to behavior, commands, fields, or states still reflect the current
  implementation?
- Are there stale descriptions that now contradict the implementation?

**Pass:** All documentation affected by this review unit is accurate.

**Pass with notes:** Documentation is mostly accurate; minor gaps or stale references noted.

**Concern:** A documentation gap exists that could mislead a future agent or developer.

**Fail:** Documentation is materially incorrect or describes behavior that no longer exists.

**Not applicable:** This review unit made no changes that would require documentation updates.

> Only evaluate documentation that was in scope for this review unit. Do not penalise for
> unrelated documentation gaps elsewhere in the codebase.

---

### Dimension 5: Workflow Integrity

**Outcome** — Is the workflow state consistent with the work?

Questions to answer:
- Is the correct feature, bug, or task linked to this work?
- Is the current lifecycle state accurate for what has been done?
- Are relevant decisions recorded?
- Is supersession handled correctly where applicable?
- Is any follow-up work missing from the workflow record?

**Pass:** Workflow records accurately reflect the state of the work.

**Pass with notes:** Minor workflow state gaps noted that should be tidied.

**Concern:** Workflow state is unclear or inconsistent in a way that could cause confusion.

**Fail:** Workflow records are materially wrong or misleading (e.g., a feature is marked done
when blocking issues remain, or the wrong entity is linked).

---

## Structured Output Format

Produce your findings as a structured document with the following sections. All fields are
required unless marked optional.

```
## Review Unit: {unit_descriptor}

**Review profile:** {profile_name}
**Spec sections reviewed:** {section_ids or "none"}
**Files reviewed:** {file list}

---

### Dimension Results

#### Specification Conformance
- **Outcome:** {pass | pass_with_notes | concern | fail | not_applicable}
- **Findings:**
  - [{blocking|non_blocking}] `{file:line or "general"}` — {description}. spec_ref: {§ref}
  - (repeat for each finding; omit list if no findings)

#### Implementation Quality
- **Outcome:** {pass | pass_with_notes | concern | fail | not_applicable}
- **Findings:**
  - [{blocking|non_blocking}] `{file:line or "general"}` — {description}
  - (omit list if no findings)

#### Test Adequacy
- **Outcome:** {pass | pass_with_notes | concern | fail | not_applicable}
- **Findings:**
  - [{blocking|non_blocking}] `{file:line or "general"}` — {description}. spec_ref: {§ref if applicable}
  - (omit list if no findings)

#### Documentation Currency
- **Outcome:** {pass | pass_with_notes | concern | fail | not_applicable}
- **Findings:**
  - [{blocking|non_blocking}] `{file:line or "general"}` — {description}
  - (omit list if no findings)

#### Workflow Integrity
- **Outcome:** {pass | pass_with_notes | concern | fail | not_applicable}
- **Findings:**
  - [{blocking|non_blocking}] `{file:line or "general"}` — {description}
  - (omit list if no findings)

---

### Summary

- **Blocking findings:** {n}
- **Non-blocking findings:** {n}
- **Overall verdict:** {approved | approved_with_followups | changes_required | blocked}

### Verdict rationale
{One to three sentences explaining the verdict. Required if verdict is anything other than
`approved`. For `approved`, may be omitted or replaced with "No blocking findings."}
```

**Overall verdict rules:**

| Condition | Verdict |
|-----------|---------|
| No findings at all | `approved` |
| Only non-blocking findings | `approved_with_followups` |
| One or more blocking findings | `changes_required` |
| Cannot evaluate — missing files, missing access, missing spec with no fallback | `blocked` |

**`spec_ref` field:** Required on blocking findings under Specification Conformance and Test
Adequacy where the deficiency can be traced to a specific requirement. Omit it for findings
where no specific spec reference applies.

---

## Finding Classification

### Blocking findings

A finding is **blocking** if any of the following is true:

- The implementation violates a specific requirement in the specification.
- The code has incorrect behavior that would cause failures or data corruption.
- A critical path has no test coverage and the behavior is not otherwise verifiable.
- The change breaks an established API contract or interface without justification.
- Documentation is materially incorrect in a way that would actively mislead users or agents.
- Workflow records are in a state that would cause incorrect lifecycle transitions.

A blocking finding **must** cite its justification — the spec section violated, the behavior
that is incorrect, or the contract that is broken. A blocking finding without a justification
is not acceptable.

### Non-blocking findings

A finding is **non-blocking** if it is worth noting but does not prevent completion:

- Style or naming suggestions that do not affect correctness.
- Non-critical improvements to readability or structure.
- Test coverage for non-critical paths that would be useful but are not required.
- Minor documentation polish where the existing text is correct but could be clearer.
- Follow-up tasks that should be created but are not blockers for this work.
- Workflow improvements that are cosmetic rather than structurally wrong.

When uncertain whether a finding is blocking or non-blocking, prefer `non_blocking` — but
do not use this to avoid raising a real defect. If something is actually broken, say so.

---

## Edge Case Handling

### Edge Case 1: Missing Spec

**Symptom:** The review unit has no linked specification document, or no spec sections were
provided as input.

**What to do:**
1. Set Specification Conformance outcome to `not_applicable`.
2. Add a non-blocking finding under Specification Conformance: `[non_blocking] general —
   No linked specification found. Conformance cannot be assessed.`
3. Continue evaluating all other dimensions normally.
4. The overall verdict may still be `approved` or `approved_with_followups` if no other
   blocking issues are found. Do not block solely because the spec is missing.
5. In the verdict rationale, note that spec conformance was skipped.

### Edge Case 2: Partial Implementation

**Symptom:** Files are present in the review unit but the implementation is clearly incomplete
— functions are stubbed, key paths are missing, or TODO markers indicate work not yet done.

**What to do:**
1. Record this as a **blocking** finding under Implementation Quality: `[blocking] {file:line}
   — Implementation is incomplete: {describe what is missing}.`
2. Also record blocking findings under Test Adequacy if the incomplete code lacks tests.
3. Set overall verdict to `changes_required`.
4. In the verdict rationale, state clearly that the implementation is not yet complete.
5. Do not attempt to infer what the finished implementation would look like — evaluate what
   is actually present.

### Edge Case 3: Ambiguous Conformance

**Symptom:** The implementation differs from the specification, but the difference may be
intentional — the spec may be outdated, a decision may have been recorded that you were not
given, or the deviation may be a deliberate improvement.

**What to do:**
1. Record the deviation as a finding under Specification Conformance. Use outcome `concern`
   (not `fail`) when you cannot confirm whether the deviation is a defect.
2. Describe the deviation precisely: what the spec says, what the implementation does, and
   why it is ambiguous.
3. Classify the finding as `non_blocking` if the implementation appears intentionally better
   or equivalent, or `blocking` if the implementation appears to omit or contradict a
   requirement.
4. In the verdict rationale, name the ambiguity so the orchestrator or human can resolve it.
5. Do not guess at intent — surface the ambiguity explicitly.

### Edge Case 4: Missing Context

**Symptom:** The review unit cannot be evaluated because files are inaccessible, the context
packet is incomplete, or required inputs were not provided.

**What to do:**
1. Record what is missing specifically.
2. Set the overall verdict to `blocked`.
3. List the missing inputs as blocking findings under the affected dimensions, or under
   Specification Conformance if the gap is general.
4. Do not attempt to review dimensions for which you lack the required context — mark each
   affected dimension as `not_applicable` with a note.
5. Do not produce a partial review and present it as complete. If you cannot evaluate a
   required dimension, say so clearly.

---

## Verification Checklist

Run through this checklist before submitting your findings:

- [ ] I have read all files in the file list.
- [ ] I have read all provided spec sections.
- [ ] I have evaluated every dimension required by the review profile.
- [ ] Every blocking finding has a specific justification (spec reference, incorrect behavior,
      or broken contract). No blocking finding is vague.
- [ ] Every finding has a severity (`blocking` or `non_blocking`), a location, and a
      description.
- [ ] The overall verdict is consistent with the finding counts:
      no blocking → `approved` or `approved_with_followups`; any blocking → `changes_required`.
- [ ] If the verdict is anything other than `approved`, I have written a verdict rationale.
- [ ] I have not modified any files. Review sub-agents are read-only.
- [ ] I have not conflated uncertainty with absence of defect. Ambiguous cases use `concern`.
- [ ] The review unit descriptor and profile name are included in the output header.

---

## Related

- `work/design/quality-gates-and-review-policy.md` — policy basis for all review dimensions,
  profiles (§5), output format (§11), and blocking vs non-blocking rules (§12). Read this for
  the rationale behind each dimension.
- `work/design/code-review-workflow.md` — the full review workflow design, including the
  orchestrator pattern, context budgets, and lifecycle integration.
- `AGENTS.md` — project conventions, Go code style, and commit policy.
- `.kbz/context/roles/reviewer.yaml` — the context profile assembled for review sub-agents.