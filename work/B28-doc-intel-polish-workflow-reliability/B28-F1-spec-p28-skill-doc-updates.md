| Field  | Value       |
|--------|-------------|
| Date   | 2026-04-22  |
| Status | approved |
| Author | spec-author |

# Specification: Skill Documentation Updates

## Overview

This specification defines the markdown-only updates to three `.agents/skills/` files required
by Sprint 0 of the P28 plan (doc-intel polish and workflow reliability). It covers a
concise-output instruction and atomicity guarantee for the `kanbanzai-documents` classification
skill, and new "Resuming an in-flight plan" guidance in the `kanbanzai-getting-started` and
`kanbanzai-workflow` skills.

## Problem Statement

This specification covers the three skill-file updates required by Sprint 0 of the P28 plan
(doc-intel polish and workflow reliability). It is scoped to markdown-only changes to three
`.agents/skills/` files: `kanbanzai-documents`, `kanbanzai-getting-started`, and
`kanbanzai-workflow`. No Go code, YAML state files, or `.kbz/` skill files are modified.

**Parent design document:**
`P28-doc-intel-polish-workflow-reliability/design-p28-doc-intel-polish-workflow-reliability`

## Scope

**In scope:**

- Two additions to the Classification (Layer 3) section of `kanbanzai-documents/SKILL.md`:
  a concise-output instruction and an atomicity guarantee statement.
- A new "Resuming an in-flight plan" section in `kanbanzai-getting-started/SKILL.md`.
- A new "Resuming an in-flight plan" section (or authoritative cross-reference plus full
  checklist) in `kanbanzai-workflow/SKILL.md`.

**Out of scope:**

- Changes to `.kbz/skills/` files (task-execution skills).
- Code changes of any kind.
- Changes to any skill files other than the three named above.
- Duplication or replacement of content added by FEAT-01KPTHB61WPT0 (corpus hygiene).

---

## Functional Requirements

**REQ-001 — Concise-output instruction in classification skill**

`kanbanzai-documents/SKILL.md` Classification (Layer 3) section MUST include a mandatory
instruction directing agents to suppress per-document commentary during bulk classification
and to report only final counts. The instruction text MUST be:

> "Be concise — no commentary between documents, just run the tools. Report only final counts."

It MUST appear as a bolded rule or inside a sub-agent prompt template within the Classification
section, and MUST be positioned so that an agent reads it before dispatching sub-agents for
bulk classification work.

**REQ-002 — Atomicity guarantee statement in classification skill**

`kanbanzai-documents/SKILL.md` Classification (Layer 3) section MUST include an explicit
statement asserting all three of the following:

1. `classify` calls commit to the persistent index as they succeed.
2. After any batch failure, `doc_intel(action: "pending")` is the authoritative ground truth
   for which documents have already been classified.
3. Agents MUST check `doc_intel(action: "pending")` before re-dispatching a failed batch, to
   avoid re-classifying documents that were successfully classified before the failure.

**REQ-003 — "Resuming an in-flight plan" section in getting-started skill**

`kanbanzai-getting-started/SKILL.md` MUST include a new section titled
**"Resuming an in-flight plan"** containing the four-step checklist defined in REQ-005, in
the specified order.

**REQ-004 — "Resuming an in-flight plan" coverage in workflow skill**

`kanbanzai-workflow/SKILL.md` MUST include either:

- (a) A self-contained "Resuming an in-flight plan" section containing the four-step checklist
  defined in REQ-005; or
- (b) A section that explicitly cross-references `kanbanzai-getting-started` and presents or
  references all four checklist items such that an agent reading only the workflow skill can
  locate and apply all four steps.

**REQ-005 — Checklist step order and content**

The "Resuming an in-flight plan" checklist MUST contain all four of the following steps, in
exactly this order:

1. Run `git status` and commit any orphaned `.kbz/` working-tree changes.
2. Check the plan's lifecycle state; if `proposed`, step through `designing` then advance to
   `active` — or apply a direct `proposed → active` override once Sprint 2 lands.
3. For each feature, check whether a dev-plan document is registered; if not, apply the gate
   override — or skip this step once Sprint 2 lands.
4. Confirm a worktree exists for each in-flight feature; create missing ones, falling back to
   `terminal` + `git worktree add` if `worktree(action: create)` times out.

**REQ-006 — Transitional labelling for steps 2 and 3**

Steps 2 and 3 of the resume-plan checklist (REQ-005) MUST be labelled as transitional with a
note that each step becomes unnecessary once the corresponding P28 Sprint 2 fixes are merged.

## Non-Functional Requirements

**REQ-NF-001 — File scope**

Exactly three files MUST be modified: `.agents/skills/kanbanzai-documents/SKILL.md`,
`.agents/skills/kanbanzai-getting-started/SKILL.md`, and
`.agents/skills/kanbanzai-workflow/SKILL.md`. No other files may be modified.

**REQ-NF-002 — No content duplication with FEAT-01KPTHB61WPT0**

No requirement, procedure, or anti-pattern already present in the Classification (Layer 3)
section as a result of FEAT-01KPTHB61WPT0 MAY be duplicated. The two features' additions MUST
be complementary and non-overlapping.

**REQ-NF-003 — Existing Classification section structure preserved**

The additions to `kanbanzai-documents/SKILL.md` MUST be inserted into the existing
Classification (Layer 3) section structure. The existing sub-sections (step-by-step procedure,
priority ordering, anti-patterns) MUST remain intact and unmodified.

**REQ-NF-004 — YAML front matter version increments**

All three skill files contain versioned YAML front matter. Version increments are permitted
but not required as part of this change.

---

## Constraints

- The `.kbz/skills/` directory (task-execution skills) MUST NOT be modified.
- No Go source files, YAML state files, or configuration files MAY be modified.
- Content added to `kanbanzai-documents/SKILL.md` by FEAT-01KPTHB61WPT0 MUST NOT be
  removed, altered, or duplicated.
- The resume-plan checklist MUST NOT include steps beyond those listed in REQ-005. No scope
  creep into unrelated agent-dispatch or review procedures.
- The Sprint 2 workarounds documented in steps 2 and 3 MUST NOT be presented as permanent
  guidance; they must be clearly marked transitional.

---

## Acceptance Criteria

**AC-001 (REQ-001):** Given the `kanbanzai-documents/SKILL.md` Classification (Layer 3)
section, when an agent reads the section prior to dispatching bulk-classification sub-agents,
then it encounters the instruction `"Be concise — no commentary between documents, just run
the tools. Report only final counts."` rendered in bold or within a sub-agent prompt template.

**AC-002 (REQ-002):** Given the `kanbanzai-documents/SKILL.md` Classification (Layer 3)
section, when an agent reads the section after a batch classification failure, then it finds
an explicit statement that `classify` calls are committed to the persistent index as they
succeed, and that `doc_intel(action: "pending")` is the authoritative source of truth for
completed work.

**AC-003 (REQ-002):** Given the `kanbanzai-documents/SKILL.md` Classification (Layer 3)
section, when an agent has experienced a partial batch failure and is deciding whether to
re-dispatch, then the section explicitly instructs it to call `doc_intel(action: "pending")`
before re-dispatching in order to avoid re-classifying already-classified documents.

**AC-004 (REQ-003):** Given `kanbanzai-getting-started/SKILL.md`, when an agent searches
it for guidance on resuming a plan that was started in a previous session, then it finds a
section with the heading "Resuming an in-flight plan" containing a numbered checklist.

**AC-005 (REQ-004):** Given `kanbanzai-workflow/SKILL.md`, when an agent searches it for
guidance on resuming an in-flight plan, then it can locate all four checklist steps either
directly in the file or via an explicit cross-reference to `kanbanzai-getting-started`.

**AC-006 (REQ-005):** Given either resume-plan section in either skill file, when the
checklist is read in order, then: step 1 addresses `git status` and committing orphaned `.kbz/`
changes, step 2 addresses the `proposed` lifecycle state and the workaround for advancing it,
step 3 addresses missing dev-plan documents and the gate override, and step 4 addresses
worktree existence and the `terminal` fallback — in exactly that sequence.

**AC-007 (REQ-006):** Given steps 2 and 3 of the resume-plan checklist in either skill file,
when an agent reads them, then each step carries an inline note indicating it is transitional
and will become unnecessary after P28 Sprint 2 is merged.

**AC-008 (REQ-NF-001):** Given a diff of the feature branch, when it is inspected, then
exactly three files are shown as modified, and all three paths begin with `.agents/skills/`.

**AC-009 (REQ-NF-002):** Given the updated `kanbanzai-documents/SKILL.md`, when the
Classification (Layer 3) section is compared against the version as it existed after
FEAT-01KPTHB61WPT0 merged, then no sentence or instruction appears more than once across
both versions' contributions.

**AC-010 (REQ-NF-003):** Given the updated `kanbanzai-documents/SKILL.md`, when the
existing step-by-step procedure, priority-ordering table, and anti-patterns sub-sections are
compared against their pre-edit baseline, then their content is unchanged.

---

## Verification Plan

| Criterion | Method     | Description |
|-----------|------------|-------------|
| AC-001    | Inspection | Read the Classification (Layer 3) section and confirm the concise-output instruction is present in bold or in a prompt template, positioned before any bulk-dispatch procedure. |
| AC-002    | Inspection | Read the Classification (Layer 3) section and confirm the atomicity statement explicitly names `classify` persistence and `doc_intel(action: "pending")` as authoritative after failure. |
| AC-003    | Inspection | Read the Classification (Layer 3) section and confirm the re-dispatch instruction explicitly references `doc_intel(action: "pending")` as the required check before re-dispatching. |
| AC-004    | Inspection | Search `kanbanzai-getting-started/SKILL.md` for the heading "Resuming an in-flight plan"; confirm the heading exists and is followed by a numbered checklist. |
| AC-005    | Inspection | Search `kanbanzai-workflow/SKILL.md` for the heading "Resuming an in-flight plan" or a cross-reference to it; confirm all four steps are reachable from within the file. |
| AC-006    | Inspection | Read both resume-plan sections and verify the four steps appear in the prescribed order: git status → proposed lifecycle → dev-plan doc → worktree creation. |
| AC-007    | Inspection | Read steps 2 and 3 in both resume-plan sections and confirm each carries a transitional note referencing P28 Sprint 2. |
| AC-008    | Inspection | Run `git diff --name-only` on the feature branch and verify exactly three filenames appear, all under `.agents/skills/`. |
| AC-009    | Inspection | Diff the Classification (Layer 3) section against the FEAT-01KPTHB61WPT0 baseline; confirm no instruction or sentence appears in both the old content and the new additions. |
| AC-010    | Inspection | Diff the step-by-step procedure, priority-ordering table, and anti-patterns sub-sections against their pre-edit baseline; confirm byte-for-byte equality in those sub-sections. |