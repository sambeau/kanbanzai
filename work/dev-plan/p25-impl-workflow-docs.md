# Dev Plan: Implementation Workflow Documentation Improvements

**Feature:** FEAT-01KPQ08YH16WZ
**Plan:** P25 — Agent Tooling and Pipeline Quality
**Spec:** work/spec/p25-impl-workflow-docs.md
**Status:** Draft

---

## Overview

Three targeted documentation edits to `.kbz/skills/` files. All tasks are
independent documentation changes with no code changes. Tasks 1 and 2 are
fully independent; Task 3 must be coordinated with FEAT-01KPQ08YKHNS9 which
also modifies `orchestrate-development/SKILL.md`.

---

## Task Breakdown

### Task 1: Update implement-task/SKILL.md for heredoc-first worktree writes

- **Description:** Edit `.kbz/skills/implement-task/SKILL.md` to swap the
  primary and secondary file-write patterns: heredoc (`cat > file << 'GOEOF'`)
  becomes primary for Go source files; `python3 -c` becomes secondary scoped
  to Markdown and YAML. Add the `GOEOF` delimiter collision warning note. Update
  any checklist items that currently recommend `python3 -c` as the default for
  Go files.
- **Deliverable:** Updated `.kbz/skills/implement-task/SKILL.md`
- **Spec requirements:** FR-001, FR-002, FR-003, FR-004
- **Depends on:** None (independent)
- **Effort:** Small

### Task 2: Update decompose-feature/SKILL.md with manual entity fallback

- **Description:** Edit `.kbz/skills/decompose-feature/SKILL.md` to add (a)
  a fallback note at the end of Phase 2 directing agents away from
  `decompose apply` when the proposal is structurally wrong, and (b) a "Manual
  Fallback" subsection in Phase 4 documenting `entity(action: "create",
  type: "task")` with required fields (`name`, `summary`, `parent_feature`),
  optional `depends_on` with a wiring example, dependency-order creation
  guidance, and a `status()` verification step.
- **Deliverable:** Updated `.kbz/skills/decompose-feature/SKILL.md`
- **Spec requirements:** FR-005, FR-006, FR-007
- **Depends on:** None (independent)
- **Effort:** Small

### Task 3: Update orchestrate-development/SKILL.md with per-task sizing guidance

- **Description:** Edit `.kbz/skills/orchestrate-development/SKILL.md` to add
  (a) a sizing rule in Phase 3 before the dispatch steps stating the >3 tasks +
  >~300-line files threshold requiring one-sub-agent-per-task dispatch, with
  explicit exemption for small-file and doc-only features, and (b) a
  corresponding anti-pattern entry in the `## Anti-Patterns` section.
- **Deliverable:** Updated `.kbz/skills/orchestrate-development/SKILL.md`
- **Spec requirements:** FR-008, FR-009, FR-010
- **Depends on:** None (independent; coordinate with FEAT-01KPQ08YKHNS9 which
  also edits this file — implement in the same commit or sequence carefully)
- **Effort:** Small

---

## Dependency Graph

```
Task 1 ──┐
Task 2 ──┼── (all independent, can run in parallel)
Task 3 ──┘
```

Tasks 1, 2, and 3 have no dependencies on each other and can be implemented
in parallel. Task 3 must be coordinated with FEAT-01KPQ08YKHNS9 to avoid
a merge conflict on `orchestrate-development/SKILL.md`.

---

## Interface Contracts

No interface contracts — all changes are documentation-only and affect
separate sections of separate files (except Task 3 and FEAT-01KPQ08YKHNS9,
which both touch `orchestrate-development/SKILL.md` and must be sequenced
or merged carefully).

---

## Notes for Implementer

- No MCP server code changes of any kind.
- No changes to `.agents/skills/` or `internal/kbzinit/skills/` — these
  are task-execution skills under `.kbz/skills/` and do not have a
  dual-write counterpart.
- Read the existing sections before editing to match the file's voice and
  formatting style.
- For Task 3: check whether FEAT-01KPQ08YKHNS9 has already been merged.
  If so, apply Task 3's changes on top. If not, coordinate to avoid conflict.

---

## Traceability Matrix

| Requirement | Task |
|-------------|------|
| FR-001 (heredoc primary for Go files) | Task 1 |
| FR-002 (delimiter collision warning) | Task 1 |
| FR-003 (GOEOF standard delimiter) | Task 1 |
| FR-004 (checklist updated) | Task 1 |
| FR-005 (Phase 2 fallback note) | Task 2 |
| FR-006 (Phase 4 manual fallback subsection) | Task 2 |
| FR-007 (verification step after manual creation) | Task 2 |
| FR-008 (per-task sizing rule in Phase 3) | Task 3 |
| FR-009 (batch dispatch exemption) | Task 3 |
| FR-010 (anti-pattern entry) | Task 3 |
| NFR-001 (structure preservation) | Tasks 1, 2, 3 |
| NFR-002 (no cross-file consolidation) | Tasks 1, 2, 3 |
| NFR-003 (no invented patterns) | Tasks 1, 2, 3 |