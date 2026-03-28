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

This SKILL covers two perspectives:

1. **Sub-agent perspective** (the Procedure, Per-Dimension Guidance, and Structured Output
   sections below) — how a single review sub-agent evaluates its assigned review unit and
   formats its output.
2. **Orchestrator perspective** (the Orchestration Procedure section at the end) — how an
   orchestrator agent coordinates the full review workflow across multiple sub-agents.

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

---

## Orchestration Procedure

This section is for **orchestrator agents** — agents coordinating the full review workflow for
a feature or set of features. If you are a review sub-agent, follow the Procedure section above
instead.

### Two-Phase Structure

Review is split into two distinct phases:

| Phase | Mode | Parallelism | Purpose |
|-------|------|-------------|---------|
| **Analysis** (steps 1–6) | Read-only | Parallel sub-agents | Discover and document findings |
| **Remediation** (steps 7–10) | Write | Sequential or parallel (conflict-checked) | Fix blocking findings |

The phases are strictly ordered. Analysis completes fully before any remediation begins. This
separation ensures that findings are comprehensive before code changes start, and that
remediation tasks are scoped precisely to the blocking issues found.

---

### Analysis Phase (Read-Only, Parallel)

#### Step 1: Feature transitions to `reviewing`

Transition the feature to `reviewing` status. This can happen:
- When all implementation tasks are terminal (the orchestrator triggers it)
- When a human explicitly requests review
- When the `finish` tool completes the last task (future enhancement)

Use `update_status(entity_type="feature", id=<feature-id>, status="reviewing")`.

#### Step 2: Query metadata

Gather the information needed to plan the review. **Do not read source code** — that is the
sub-agents' job.

Collect:
- **Feature entity** — the feature record and its spec document reference
- **Spec document structure** — via `doc_outline` to understand what the spec covers
- **Task list and modified files** — via `list_entities_filtered(entity_type="task", parent=<feature-id>)` to know what was implemented and which files were touched
- **Review profile** — which dimensions and thresholds to apply (default: Feature Implementation Review Profile from quality gates policy §5.1)

#### Step 3: Decompose into review units

Partition the feature's files and spec sections into logical review units. Choose a
decomposition strategy based on the feature's size and structure:

| Strategy | When to use | Example |
|----------|-------------|---------|
| **By package** | Feature spans multiple Go packages | `internal/service/`, `internal/storage/`, `internal/mcp/` as separate units |
| **By spec section** | Spec has clear independent sections | §3 (API), §4 (validation), §5 (persistence) as separate units |
| **By concern** | Cross-cutting concerns need focused attention | Tests as one unit, documentation as another, implementation as a third |
| **By layer** | Feature touches multiple architectural layers | Service layer, storage layer, MCP layer as separate units |

**Sizing guidance:**
- **Small features** (≤10 files): 1 review unit. No decomposition needed.
- **Medium features** (11–30 files): 2–4 review units.
- **Large features** (30+ files): 3–8 review units. Aim for 5–15 files per unit.

Each review unit is defined by:
- A set of source files to examine
- The relevant spec section(s) (retrieved via `doc_section`)
- The review dimensions to check
- The review profile to apply

#### Step 4: Dispatch sub-agents in parallel

For each review unit, spawn a sub-agent with:

1. **Context packet** — `context_assemble(role="reviewer")` provides the reviewer profile,
   relevant knowledge entries, and project conventions.
2. **This SKILL** — the sub-agent follows the Procedure section (steps 1–4 above in the
   sub-agent procedure) to produce structured findings.
3. **File and spec scope** — the specific files and spec sections for this unit.
4. **Structured output format** — the sub-agent must return findings in the format defined
   in the Structured Output Format section of this SKILL.

Sub-agents are **read-only**. They examine files and produce findings. They do not modify
any code, create tasks, or transition entity states.

#### Step 5: Collate findings

When all sub-agents return, merge their outputs:

1. **Deduplicate** — multiple sub-agents may flag the same issue from different angles.
   Keep the most specific finding and note the overlap.
2. **Categorise** — separate findings into blocking and non-blocking using the rules in the
   Finding Classification section of this SKILL.
3. **Per-dimension verdict** — determine the verdict for each review dimension across all
   units. A dimension fails if any unit produces a blocking finding in that dimension.
4. **Aggregate verdict** — compute the overall verdict:
   - `approved` — no blocking findings in any dimension
   - `approved_with_followups` — no blocking findings, but non-blocking findings exist
   - `changes_required` — one or more blocking findings

#### Step 6: Write review document

Write the collated findings to a review document associated with the feature. This provides:
- A human-readable record of what was reviewed and what was found
- A machine-readable structure for remediation planning
- An audit trail for the feature's review history

---

### Decision Point

After the analysis phase, the orchestrator makes a routing decision based on the aggregate
verdict:

| Verdict | Action |
|---------|--------|
| No blocking findings | Transition feature to `done` |
| Blocking findings | Transition feature to `needs-rework`, proceed to remediation phase |
| Ambiguous findings requiring human judgment | Call `human_checkpoint` and wait for response |

---

### Remediation Phase (Write, Sequential or Parallel)

Enter this phase only when blocking findings exist.

#### Step 7: Create remediation tasks

Create tasks as children of the feature, one per blocking finding or logical group of related
findings. Each task summary should reference the review finding it addresses.

Use `create_task(parent_feature=<feature-id>, slug=<descriptive-slug>, summary=<summary>)`.

#### Step 8: Dispatch tasks with conflict checking

Before dispatching remediation tasks in parallel, check for file overlap:

Use `conflict_domain_check(task_ids=[...])` to determine which tasks can safely run
concurrently. Tasks modifying the same files should be serialised or checkpointed.

Dispatch tasks through the normal workflow (`dispatch_task`).

#### Step 9: Re-review affected sections only

After remediation tasks complete, re-review **only the affected sections** — not the entire
feature. This is a targeted re-analysis:

- Identify which review units are affected by the remediation
- Spawn sub-agents for those units only
- Collate the new findings with the unchanged findings from step 5

#### Step 10: Transition

- If re-review passes (no blocking findings): transition feature to `done`.
- If new blocking issues are found: repeat the remediation cycle (steps 7–10).
- If the cycle has repeated and issues persist: use `human_checkpoint` to escalate.

---

### Context Budget Strategy

The orchestrator and sub-agents have deliberately different context profiles. This is the
key scaling strategy that allows review to work on features of any size.

**The orchestrator works at the metadata level only.** It holds:

| Data | Approximate size | Source |
|------|-----------------|--------|
| Feature entity state | ~200 bytes | `get_entity` |
| Spec document outline | ~1–2 KB | `doc_outline` |
| Task list with file paths | ~1–3 KB | `list_entities_filtered` |
| Review SKILL (this document) | ~2–3 KB | `.skills/code-review.md` |
| Collated findings | ~2–5 KB | Sub-agent outputs |
| **Total** | **~6–14 KB** | |

The orchestrator **never reads source code**. It plans and coordinates.

**Sub-agents hold their review unit's context.** Each sub-agent holds:

| Data | Approximate size | Source |
|------|-----------------|--------|
| Reviewer profile | ~2 KB | `context_assemble(role="reviewer")` |
| Review SKILL (this document) | ~2–3 KB | `.skills/code-review.md` |
| Spec section(s) for their unit | ~2–5 KB | `doc_section` |
| Source files for their unit | ~5–20 KB | File reads |
| Structured output template | ~0.5 KB | From this SKILL |
| **Total** | **~12–30 KB per sub-agent** | |

This means:
- The orchestrator's context cost is **constant** regardless of codebase size.
- Sub-agent context scales with the review unit size, not the feature size.
- A 100-file feature with 5 review units uses the same orchestrator budget as a 10-file
  feature with 1 unit.

---

### Tool Chain Reference

| Step | Tools |
|------|-------|
| Find features to review | `list_entities_filtered(entity_type="feature", status="reviewing")` |
| Get spec structure | `doc_outline`, `doc_section` |
| Get task/file lists | `list_entities_filtered(entity_type="task", parent=<feature-id>)` |
| Build sub-agent context | `context_assemble(role="reviewer")` |
| Dispatch sub-agents | `spawn_agent` (each gets review SKILL + unit scope) |
| Create remediation tasks | `create_task` |
| Transition feature state | `update_status` |
| Check conflict risk | `conflict_domain_check` |
| Request human input | `human_checkpoint` |
| Record decisions | `record_decision` |

---

### Human Checkpoint Integration Points

Use `human_checkpoint` when:

1. **Ambiguous findings** — findings that are not clearly blocking or non-blocking. The
   orchestrator cannot make a confident routing decision and needs human judgment on whether
   to proceed to `done` or enter remediation.

2. **High-stakes features** — when the feature is critical and final approval should be
   explicit rather than automated. The orchestrator can pass the review document summary
   in the checkpoint context so the human can make an informed decision.

3. **Disagreement between dimensions** — when review dimensions produce conflicting signals.
   For example, spec conformance passes but implementation quality fails, or tests pass but
   documentation is missing. The human decides whether the passing dimensions are sufficient
   or the failing dimensions are blocking.

When creating a checkpoint, include:
- The aggregate verdict and per-dimension verdicts
- A summary of the contentious findings
- The recommended action and why the orchestrator is uncertain

Wait for the human response before proceeding. Do not dispatch remediation tasks or
transition feature state while a checkpoint is pending.

---

### Review at Different Scales

The orchestration procedure scales across different review scopes:

| Scale | Approach |
|-------|----------|
| **Single feature** | The primary use case. One pass through steps 1–10. Typically 2–5 parallel review sub-agents. |
| **Multiple features (phase review)** | Iterate over features in `reviewing` state. Features are independent — review them in parallel at the feature level, with sub-agent parallelism within each. |
| **Full codebase audit** | Decompose by package or module rather than by feature. Requires a separate decomposition strategy outside the feature-based workflow. |