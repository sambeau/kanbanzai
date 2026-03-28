# Dev Plan: AGENTS.md Cleanup

| Field        | Value                                          |
|--------------|------------------------------------------------|
| **Feature**  | FEAT-01KMTSPAV34HR (agents-md-cleanup)         |
| **Plan**     | P3-kanbanzai-1.0                               |
| **Spec**     | `work/spec/agents-md-cleanup.md`               |
| **Created**  | 2026-03-28                                     |

---

## 1. Overview

This is a documentation-only change. No Go code is affected. The work separates project-specific content from product-facing content in `AGENTS.md`, removing ~140 lines of workflow instructions that duplicate the kanbanzai skills, moving two uncovered product principles into the appropriate skill files, and replacing outdated terminology.

## 2. Task Execution Order

### Task 1: remove-redundant-sections (TASK-01KMTT3VY4R5C)

**Goal:** Remove sections from `AGENTS.md` that duplicate content already in `.agents/skills/kanbanzai-*/SKILL.md`.

**Steps:**
1. Read `AGENTS.md` to identify the exact line ranges for each section to remove.
2. Remove the "Workflow Stage Gates" section and all six sub-stage sections (Planning, Design, Features, Specification, Dev Plan & Tasks, Implementation and Review) including the "Emergency Brake" subsection.
3. Remove the "Document Creation Workflow" section.
4. Remove the Git commit format table and worked examples from the "Git Rules" section. Keep the branching rules and commit type list.
5. Verify AC-1, AC-2, AC-3, AC-4, AC-11 from the spec.

**Files modified:** `AGENTS.md`

### Task 2: update-terminology (TASK-01KMTT3VYM1AA)

**Goal:** Replace outdated "Two Workflows" framing in `AGENTS.md`.

**Steps:**
1. Replace the "Two Workflows" section with a brief note explaining that kanbanzai manages its own development (uses itself).
2. Remove all conceptual uses of "bootstrap" and "bootstrap-workflow" terminology, preserving only file path references (e.g., `work/bootstrap/bootstrap-workflow.md`).
3. Remove "kbz-workflow" terminology as a primary framing concept.
4. Verify AC-8, AC-9 from the spec.

**Files modified:** `AGENTS.md`

### Task 3: move-principles-to-skills (TASK-01KMTT3VZ2ZXQ)

**Goal:** Move two product-facing principles from `AGENTS.md` into the appropriate skill files.

**Steps:**
1. Read the "Communicating With Humans" section from `AGENTS.md`.
2. Add the principle (reference documents by name, not decision IDs; use prose descriptions; save IDs for commits and agent-to-agent communication) to the `kanbanzai-agents` skill at `.agents/skills/kanbanzai-agents/SKILL.md`.
3. Read the "Documentation Accuracy" section from `AGENTS.md`.
4. Add the principle (code is truth — if docs conflict with code, fix docs; spec is intent — if code conflicts with spec, surface conflict to human) to the `kanbanzai-workflow` skill at `.agents/skills/kanbanzai-workflow/SKILL.md`.
5. Remove both sections from `AGENTS.md`.
6. Verify AC-5, AC-6, AC-7 from the spec.

**Files modified:** `AGENTS.md`, `.agents/skills/kanbanzai-agents/SKILL.md`, `.agents/skills/kanbanzai-workflow/SKILL.md`

### Task 4: verify-coherence (TASK-01KMTT3VZFXT8)

**Depends on:** Tasks 1, 2, 3.

**Goal:** End-to-end verification that `AGENTS.md` is coherent after all changes and no product-facing content was lost.

**Steps:**
1. Read `AGENTS.md` end-to-end. Check for dangling cross-references, broken internal links, orphaned list items or headings.
2. Verify all sections listed in AC-10 are still present.
3. Verify AC-12 (structural coherence).
4. Verify AC-13 (`.skills/` at project root unaffected).
5. For each removed block, verify the corresponding guidance exists in at least one skill file (AC-14).
6. Document the content mapping in the commit message.

**Files modified:** `AGENTS.md` (minor coherence fixes only)

## 3. Risk Assessment

- **Low risk.** All changes are documentation-only. No code, no entity schemas, no MCP tool behaviour affected.
- **Primary risk:** Accidentally removing project-specific content. Mitigation: AC-10 explicitly lists every section that must be preserved.
- **Secondary risk:** Losing product-facing guidance. Mitigation: AC-14 requires verification that removed content exists in skills.

## 4. Verification

Run the full verification matrix from the spec §4 after all four tasks are complete.