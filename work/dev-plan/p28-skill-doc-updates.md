# Dev Plan: Skill Documentation Updates

**Feature:** FEAT-01KPVDDYN855F
**Plan:** P28 — Doc-Intel Polish and Workflow Reliability
**Spec:** work/spec/p28-skill-doc-updates.md
**Status:** Draft

---

## Scope

This plan covers markdown-only edits to three agent skill files required by Sprint 0 of the
P28 plan. No Go code, YAML state files, or `.kbz/` skill files are modified.

Specification: `work/spec/p28-skill-doc-updates.md`
Design reference: `P28-doc-intel-polish-workflow-reliability/design-p28-doc-intel-polish-workflow-reliability`

**Deliverables:**
- `.agents/skills/kanbanzai-documents/SKILL.md` — concise-output instruction + atomicity guarantee added to Classification (Layer 3) section
- `.agents/skills/kanbanzai-getting-started/SKILL.md` — new "Resuming an in-flight plan" checklist section
- `.agents/skills/kanbanzai-workflow/SKILL.md` — new "Resuming an in-flight plan" coverage (full section or explicit cross-reference)

**Constraints:**
- Exactly three files modified; all under `.agents/skills/`.
- No content added by FEAT-01KPTHB61WPT0 may be duplicated or removed.
- Existing Classification section sub-sections must remain intact.

---

## Task Breakdown

### Task 1: Add concise-output instruction and atomicity guarantee to kanbanzai-documents SKILL.md

- **Description:** Edit `.agents/skills/kanbanzai-documents/SKILL.md`. In the Classification
  (Layer 3) section, insert (a) a bolded concise-output instruction — "Be concise — no
  commentary between documents, just run the tools. Report only final counts." — positioned
  before any bulk-dispatch procedure, and (b) an explicit atomicity guarantee statement
  asserting that `classify` calls commit to the persistent index as they succeed, that
  `doc_intel(action: "pending")` is the authoritative ground truth after any batch failure,
  and that agents must check `doc_intel(action: "pending")` before re-dispatching a failed
  batch. Additions must be complementary to and non-overlapping with content introduced by
  FEAT-01KPTHB61WPT0. Existing sub-sections (step-by-step procedure, priority ordering,
  anti-patterns) must be left byte-for-byte unchanged.
- **Deliverable:** Updated `.agents/skills/kanbanzai-documents/SKILL.md` with both additions
  present in the Classification (Layer 3) section.
- **Depends on:** None
- **Effort:** small
- **Spec requirement:** REQ-001, REQ-002, REQ-NF-002, REQ-NF-003

### Task 2: Add "Resuming an in-flight plan" section to kanbanzai-getting-started SKILL.md

- **Description:** Edit `.agents/skills/kanbanzai-getting-started/SKILL.md`. Add a new
  section titled **"Resuming an in-flight plan"** containing a four-step numbered checklist
  in exactly this order: (1) run `git status` and commit orphaned `.kbz/` working-tree
  changes; (2) check the plan's lifecycle state and handle `proposed` state with the Sprint 2
  workaround — labelled as transitional; (3) check whether a dev-plan document is registered
  for each feature and apply the gate override if not — labelled as transitional; (4) confirm
  a worktree exists for each in-flight feature, creating missing ones with a `terminal`
  fallback. Steps 2 and 3 must carry inline notes that they become unnecessary once P28
  Sprint 2 is merged.
- **Deliverable:** Updated `.agents/skills/kanbanzai-getting-started/SKILL.md` with the new
  "Resuming an in-flight plan" section and its four-step checklist.
- **Depends on:** None
- **Effort:** small
- **Spec requirement:** REQ-003, REQ-005, REQ-006

### Task 3: Add "Resuming an in-flight plan" coverage to kanbanzai-workflow SKILL.md

- **Description:** Edit `.agents/skills/kanbanzai-workflow/SKILL.md`. Add either a
  self-contained "Resuming an in-flight plan" section containing the same four-step checklist
  as Task 2, or a section that explicitly cross-references `kanbanzai-getting-started` and
  presents or references all four steps so that an agent reading only the workflow skill can
  locate and apply every step. Steps 2 and 3 must carry the same transitional labels as in
  Task 2.
- **Deliverable:** Updated `.agents/skills/kanbanzai-workflow/SKILL.md` with full resume-plan
  coverage reachable from within the file.
- **Depends on:** None
- **Effort:** small
- **Spec requirement:** REQ-004, REQ-005, REQ-006

### Task 4: Verify all three skill-file edits

- **Description:** Inspect the three modified files against the acceptance criteria. Confirm:
  the concise-output instruction is present in the Classification section in bold or a prompt
  template; the atomicity statement names `classify` persistence and `doc_intel(action:
  "pending")` as authoritative; the re-dispatch instruction references `doc_intel(action:
  "pending")`; both resume-plan sections contain the four steps in the correct order; steps 2
  and 3 carry transitional notes; `git diff --name-only` shows exactly three files, all under
  `.agents/skills/`; and no content from FEAT-01KPTHB61WPT0 has been duplicated or altered.
- **Deliverable:** Confirmed pass on AC-001 through AC-010; any failures resolved before
  marking the task done.
- **Depends on:** Task 1, Task 2, Task 3
- **Effort:** small
- **Spec requirement:** AC-001, AC-002, AC-003, AC-004, AC-005, AC-006, AC-007, AC-008, AC-009, AC-010

---

## Dependency Graph

```
Task 1 (kanbanzai-documents edits)   ─┐
Task 2 (kanbanzai-getting-started)   ─┤─► Task 4 (verification)
Task 3 (kanbanzai-workflow)          ─┘
```

Tasks 1, 2, and 3 are fully independent (different files) and may run in parallel.
Task 4 depends on all three and must run last.

---

## Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| New additions accidentally duplicate FEAT-01KPTHB61WPT0 content | Medium | Medium | Read Classification section baseline before editing; Task 4 explicitly checks for duplication (AC-009). |
| Existing Classification sub-sections inadvertently altered | Low | Medium | Make minimal targeted insertions; Task 4 diffs sub-sections against baseline (AC-010). |
| Resume-plan checklist steps out of order | Low | Low | Spec REQ-005 lists exact order; implementer copies verbatim from spec. |
| Transitional labels omitted from steps 2 and 3 | Low | Low | Checklist in Task 4 explicitly verifies AC-007. |

---

## Verification Approach

| Acceptance Criterion | Verification Method | Producing Task |
|----------------------|---------------------|----------------|
| AC-001 — Concise-output instruction present in bold or prompt template | Inspection | Task 4 |
| AC-002 — Atomicity statement names `classify` persistence and `pending` as authoritative | Inspection | Task 4 |
| AC-003 — Re-dispatch instruction references `doc_intel(action: "pending")` | Inspection | Task 4 |
| AC-004 — "Resuming an in-flight plan" heading + numbered checklist in getting-started | Inspection | Task 4 |
| AC-005 — All four steps reachable from kanbanzai-workflow | Inspection | Task 4 |
| AC-006 — Four steps appear in prescribed order in both skill files | Inspection | Task 4 |
| AC-007 — Steps 2 and 3 carry transitional notes referencing P28 Sprint 2 | Inspection | Task 4 |
| AC-008 — Exactly three files modified, all under `.agents/skills/` | Inspection (`git diff --name-only`) | Task 4 |
| AC-009 — No sentence from FEAT-01KPTHB61WPT0 additions duplicated | Inspection (diff against baseline) | Task 4 |
| AC-010 — Existing Classification sub-sections unchanged | Inspection (diff against baseline) | Task 4 |