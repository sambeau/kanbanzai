# Policy and Documentation Updates Specification

| Document | Policy and Documentation Updates |
|----------|----------------------------------|
| Status   | Draft                            |
| Created  | 2026-03-28T00:27:32Z             |
| Updated  | 2026-03-28T00:27:32Z             |
| Related  | `work/design/code-review-workflow.md` §14.4, §14.5 |

---

## 1. Purpose

After Features D, E, and F of the P6 workflow quality plan are implemented, the project gains two new feature lifecycle states (`reviewing` and `needs-rework`), a reviewer context profile, a code review SKILL, and a validated orchestration pattern for parallel review. This specification defines the updates required to ensure that `AGENTS.md`, the quality gates policy, the bootstrap workflow, and the plan status document accurately reflect these additions. Without these updates, the documentation would be inconsistent with the running system and agents would receive conflicting instructions.

---

## 2. Goals

1. Make code review a visible, mandatory gate in `AGENTS.md` rather than an optional human step.
2. Eliminate duplicated inline review instructions from `AGENTS.md` by consolidating them into the reviewer SKILL.
3. Update `work/design/quality-gates-and-review-policy.md` to cross-reference the code review design and the reviewer profile, and to note that review dimensions are now operationalised through canonical YAML and SKILL files.
4. Update `work/bootstrap/bootstrap-workflow.md` to reflect the `reviewing` state in the feature completion path.
5. Record that P6 Phase 2 is now active in the plan status document.

---

## 3. Scope

### 3.1 In scope

- `AGENTS.md` — add a reference to the review SKILL (`.skills/code-review.md`), state that code review is a mandatory feature lifecycle gate, and remove any inline review instructions that duplicate the SKILL content.
- `work/design/quality-gates-and-review-policy.md` — add cross-references to `work/design/code-review-workflow.md` and to `.kbz/context/roles/reviewer.yaml`; add a note that review dimensions and profiles are operationalised through those files.
- `work/bootstrap/bootstrap-workflow.md` — audit the feature completion path; if it does not mention the `reviewing` state, add it.
- `work/plan/workflow-quality-and-review-plan.md` — update the status header from Phase 2 pending to Phase 2 active.
- Document record refresh — after each file is edited, confirm that the content hash in the document record store is current (re-submit or note drift if a record exists).

### 3.2 Deferred

- Adding new section headers or restructuring `AGENTS.md` beyond the minimal changes needed to remove duplicates and add the SKILL reference.
- Authoring a new version of the quality gates policy document (a targeted amendment is sufficient).
- Updating any document outside the four files listed in §3.1.

### 3.3 Explicitly excluded

- Changes to entity YAML files, lifecycle state machine code, or MCP tool behaviour — those are owned by Features D, E, and F.
- Creating or modifying the reviewer profile (`.kbz/context/roles/reviewer.yaml`) — owned by Feature E.
- Creating or modifying the code review SKILL (`.skills/code-review.md`) — owned by Feature D.
- Changes to any specification document other than this one.

---

## 4. Acceptance Criteria

**AC-G-01 — AGENTS.md: mandatory review gate**
`AGENTS.md` contains a statement, in the workflow stage gates section or the implementation stage description, that code review is a mandatory feature lifecycle gate (not an optional human step) and that features must pass through the `reviewing` state before they can be closed.

**AC-G-02 — AGENTS.md: review SKILL reference**
`AGENTS.md` contains an explicit reference to `.skills/code-review.md` as the canonical source for review instructions and expectations. The reference is present in a location a reader naturally encounters when looking for review guidance (e.g., the implementation stage gate or a dedicated review subsection).

**AC-G-03 — AGENTS.md: no duplicated inline review instructions**
Any inline review instructions in `AGENTS.md` that are substantively duplicated in `.skills/code-review.md` are removed. After the change, `AGENTS.md` directs readers to the SKILL rather than restating the same criteria. Factual statements not covered by the SKILL (e.g., lifecycle state names) may remain.

**AC-G-04 — quality-gates-and-review-policy.md: design cross-reference**
`work/design/quality-gates-and-review-policy.md` contains a cross-reference to `work/design/code-review-workflow.md`. The reference is in a visible location (e.g., a "See also" section, the introduction, or an inline note near the review dimensions section).

**AC-G-05 — quality-gates-and-review-policy.md: reviewer profile cross-reference**
`work/design/quality-gates-and-review-policy.md` contains a reference to `.kbz/context/roles/reviewer.yaml` noting that the reviewer context profile operationalises the review dimensions defined in the policy.

**AC-G-06 — quality-gates-and-review-policy.md: SKILL cross-reference**
`work/design/quality-gates-and-review-policy.md` contains a reference to `.skills/code-review.md` noting that the review SKILL operationalises the review procedure defined in the policy.

**AC-G-07 — quality-gates-and-review-policy.md: operationalisation note**
`work/design/quality-gates-and-review-policy.md` includes a note (inline or in a dedicated paragraph) stating that the review dimensions and reviewer profile described in the policy are now operationalised through the reviewer YAML and the review SKILL, and that those files are the canonical source for day-to-day execution. The policy document remains the normative statement of intent.

**AC-G-08 — bootstrap-workflow.md: reviewing state present**
`work/bootstrap/bootstrap-workflow.md` is audited. If its feature completion path (the sequence of steps from "implementation done" to "feature closed") does not mention the `reviewing` state, it is updated to include it as a required step before a feature transitions to done. If the `reviewing` state is already present and correctly positioned, no change is made (audit finding is documented in the verification record).

**AC-G-09 — plan status updated**
`work/plan/workflow-quality-and-review-plan.md` status header (or equivalent status field near the top of the document) reflects that P6 Phase 2 is now active (not pending or planned). The update must be consistent with the status language used elsewhere in the document.

**AC-G-10 — document records consistent**
For each file modified under AC-G-01 through AC-G-09, the kanbanzai document record (if one exists) is refreshed so its stored content hash matches the file on disk. No document record for any of these files reports drift after the changes are complete.

---

## 5. Verification

| Criterion | Verification method |
|-----------|---------------------|
| AC-G-01 | Read `AGENTS.md`; locate the workflow stage gates or implementation stage section; confirm the mandatory review gate statement is present. |
| AC-G-02 | `grep -n "code-review.md" AGENTS.md` returns at least one match in a section visible to agents at task time. |
| AC-G-03 | Read the diff of `AGENTS.md`; confirm removed lines are substantively covered by `.skills/code-review.md`; no review criteria appear only in `AGENTS.md`. |
| AC-G-04 | `grep -n "code-review-workflow" work/design/quality-gates-and-review-policy.md` returns at least one match. |
| AC-G-05 | `grep -n "reviewer.yaml" work/design/quality-gates-and-review-policy.md` returns at least one match. |
| AC-G-06 | `grep -n "code-review.md" work/design/quality-gates-and-review-policy.md` returns at least one match. |
| AC-G-07 | Read the updated policy document; locate the operationalisation note; confirm it directs readers to the YAML and SKILL as canonical execution sources while retaining normative text. |
| AC-G-08 | Read `work/bootstrap/bootstrap-workflow.md`; locate the feature completion path; confirm `reviewing` appears before the terminal state. If already present, note "no change required" in the commit message or PR description. |
| AC-G-09 | Read `work/plan/workflow-quality-and-review-plan.md`; confirm the status field near the top reads "active" (or equivalent) for Phase 2. |
| AC-G-10 | Run `doc_record_validate` (or equivalent drift check) for each modified document record; confirm no drift is reported. |