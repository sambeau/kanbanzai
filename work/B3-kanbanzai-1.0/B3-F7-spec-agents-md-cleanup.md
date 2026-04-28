# AGENTS.md Cleanup Specification

| Document | AGENTS.md Cleanup |
|----------|-------------------|
| Status   | Draft             |
| Created  | 2026-03-28        |
| Updated  | 2026-03-28        |
| Feature  | FEAT-01KMTSPAV34HR (agents-md-cleanup) |
| Plan     | P3-kanbanzai-1.0  |
| Related  | `work/design/kanbanzai-1.0.md` §4.3, `.agents/skills/kanbanzai-*/SKILL.md` |

---

## 1. Purpose

The project's `AGENTS.md` file currently mixes two kinds of content:

1. **Project-specific** instructions for developing the kanbanzai server itself — Go conventions, repository structure, decision log references, build commands, test conventions, and codebase knowledge graph usage.
2. **Product-facing** workflow instructions that duplicate content already delivered through the six `.agents/skills/kanbanzai-*/SKILL.md` files.

A detailed analysis found approximately 60% project-specific content (correctly placed), 25% product-facing content redundant with skills (should be removed), and 15% mixed content (needs splitting).

The design document (`work/design/kanbanzai-1.0.md` §4.3) established the boundary: skills are the product interface; `AGENTS.md` stays project-specific. This specification defines the exact changes required to enforce that boundary.

---

## 2. Scope

### 2.1 In scope

- Removing sections of `AGENTS.md` that duplicate content in `.agents/skills/kanbanzai-*/SKILL.md` files.
- Moving product-facing principles from `AGENTS.md` into the appropriate skill files.
- Replacing outdated "Two Workflows" terminology.
- Verifying that no product-facing workflow guidance is lost in the process.

### 2.2 Out of scope

- Restructuring `AGENTS.md` beyond the removals, moves, and terminology updates defined here.
- Modifying the `.skills/` directory at the project root (used during kanbanzai development).
- Changes to entity YAML files, lifecycle state machine code, or MCP tool behaviour.
- Changes to any specification or design document other than this one.
- Rewriting or restructuring the six `.agents/skills/kanbanzai-*/SKILL.md` files beyond the targeted additions in AC-5 and AC-6.

---

## 3. Acceptance Criteria

### Sections to remove from AGENTS.md (redundant with skills)

**AC-1 — Workflow Stage Gates removed**
The "Workflow Stage Gates" section — including all six sub-stage sections (Planning, Design, Features, Specification, Dev Plan & Tasks, Implementation and Review) and the "Emergency Brake" subsection — must be removed from `AGENTS.md`. This content is covered by the `kanbanzai-workflow`, `kanbanzai-planning`, `kanbanzai-design`, and `kanbanzai-agents` skills.

**AC-2 — Document Creation Workflow removed**
The "Document Creation Workflow" section must be removed from `AGENTS.md`. This content is covered by the `kanbanzai-documents` skill and `.skills/document-creation.md`.

**AC-3 — Git commit format table and examples removed**
The Git Rules commit format table and worked examples must be removed from `AGENTS.md`. The commit message format specification is in the `kanbanzai-agents` skill. The "Git Rules" section may retain brief git branching rules and the commit type list if these are not fully duplicated in skills (see AC-11).

**AC-4 — Stage heading phrases absent**
After all removals, `AGENTS.md` must not contain any of the following as section headings (at any heading level): "Stage 1: Planning", "Stage 2: Design", "Stage 3: Features", "Stage 4: Specification", "Stage 5: Dev Plan", "Stage 6: Implementation", or "Emergency Brake".

### Content to move into skills (product-facing, not yet in skills)

**AC-5 — "Communicating With Humans" principle moved to skill**
The "Communicating With Humans" principle — reference documents by name, not decision IDs; use prose descriptions of decisions; save decision IDs for commit messages and agent-to-agent communication — must be added to the `kanbanzai-agents` or `kanbanzai-workflow` skill (whichever is the better fit for the content).

**AC-6 — "Documentation Accuracy" principle moved to skill**
The "Documentation Accuracy" principle — code is truth (if documentation conflicts with code, fix the documentation); spec is intent (if code conflicts with the specification, surface the conflict to the human) — must be added to the `kanbanzai-workflow` or `kanbanzai-agents` skill (whichever is the better fit for the content).

**AC-7 — Moved principles absent from AGENTS.md**
After the moves, neither the "Communicating With Humans" section nor the "Documentation Accuracy" section may appear in `AGENTS.md`. The content must exist only in the target skill files.

### Terminology update

**AC-8 — "Two Workflows" section replaced**
The "Two Workflows" section must be replaced. The terms "bootstrap-workflow" and "kbz-workflow" must not appear as primary framing concepts. A brief note explaining that kanbanzai manages its own development (i.e., it uses itself) is acceptable as a replacement.

**AC-9 — "bootstrap" terminology removed**
The terms "bootstrap" and "bootstrap-workflow" must not appear in `AGENTS.md` except in references to historical document paths (e.g., `work/bootstrap/bootstrap-workflow.md` in reading lists or document tables).

### What must stay in AGENTS.md

**AC-10 — Project-specific sections preserved**
All of the following project-specific sections must be preserved in `AGENTS.md` (content may be lightly edited for coherence after removals, but must not be deleted):

- Overview
- Naming Conventions
- Repository Structure
- Before Any Task
- Document Reading Order
- Key Design Documents by Topic
- Decision-Making Rules
- Scope Guard
- YAML Serialisation Rules
- Build and Test Commands
- Go Code Style (including all subsections: Formatting, Naming, Error Handling, Comments, Interfaces, Concurrency, Package Design)
- File Organisation
- Dependencies
- Testing (including all subsections)
- Codebase Knowledge Graph
- Delegating to Sub-Agents

**AC-11 — Git Rules section trimmed, not gutted**
The "Git Rules" section may retain the brief git branching rules (AI commits to feature/bug branches, AI merges to main, etc.) and the commit type list (`feat`, `fix`, `docs`, etc.) if these are not fully duplicated in skills. The full commit message format table and worked examples must be removed per AC-3.

### Structural integrity

**AC-12 — Document coherence**
`AGENTS.md` must remain a valid, coherent Markdown document after all changes. There must be no dangling cross-references to removed sections, no broken internal links, and no orphaned list items or headings that reference content that no longer exists.

**AC-13 — Project-root `.skills/` unaffected**
The `.skills/` directory at the project root (used during kanbanzai development) is not modified by this change. Only `.agents/skills/kanbanzai-*/SKILL.md` files may be modified.

### Verification

**AC-14 — No product-facing instructions lost**
The six `.agents/skills/kanbanzai-*/SKILL.md` files must collectively contain all workflow guidance that was removed from `AGENTS.md`. No product-facing instructions may be lost in the transition. Content that existed in `AGENTS.md` and was removed must be verifiably present in at least one skill file (either pre-existing or newly added under AC-5/AC-6).

---

## 4. Verification Methods

| Criterion | Verification method |
|-----------|---------------------|
| AC-1 | `grep -n "Workflow Stage Gates" AGENTS.md` returns no matches. Read the diff to confirm the full section (all six sub-stages) is removed. |
| AC-2 | `grep -n "Document Creation Workflow" AGENTS.md` returns no matches. |
| AC-3 | Read the "Git Rules" section in `AGENTS.md`; confirm no format table or worked examples remain. Compare against the `kanbanzai-agents` skill to confirm the content exists there. |
| AC-4 | `grep -nE "^#{1,6} .*(Stage [1-6]:|Emergency Brake)" AGENTS.md` returns no matches. |
| AC-5 | `grep -l "Communicating With Humans\|reference documents by name" .agents/skills/kanbanzai-*/SKILL.md` returns at least one match. Read the matched file to confirm the principle is substantively present. |
| AC-6 | `grep -l "Documentation Accuracy\|Code is truth\|spec is intent" .agents/skills/kanbanzai-*/SKILL.md` returns at least one match. Read the matched file to confirm the principle is substantively present. |
| AC-7 | `grep -n "Communicating With Humans" AGENTS.md` and `grep -n "Documentation Accuracy" AGENTS.md` both return no matches as section headings. |
| AC-8 | `grep -n "Two Workflows" AGENTS.md` returns no matches as a section heading. `grep -n "bootstrap-workflow\|kbz-workflow" AGENTS.md` returns no matches outside of document path references. |
| AC-9 | `grep -n "bootstrap" AGENTS.md` returns only lines containing file paths (e.g., `work/bootstrap/`). No conceptual use of the term remains. |
| AC-10 | For each section listed in AC-10, `grep -n "<section name>" AGENTS.md` returns at least one heading-level match. |
| AC-11 | Read the "Git Rules" section; confirm branching rules and commit type list are present; confirm format table and examples are absent. |
| AC-12 | Read `AGENTS.md` end-to-end after changes. Confirm no dangling references, broken links, or orphaned content. Validate Markdown renders correctly. |
| AC-13 | `git diff --name-only` shows no files under `.skills/` (project root). Only files under `.agents/skills/kanbanzai-*/` are modified. |
| AC-14 | For each block of content removed from `AGENTS.md`, identify the corresponding skill file that covers the same guidance. Document the mapping in the PR description or review notes. |

---

## 5. Notes

- The six kanbanzai skills are: `kanbanzai-workflow`, `kanbanzai-planning`, `kanbanzai-design`, `kanbanzai-documents`, `kanbanzai-agents`, and `kanbanzai-review`. These live under `.agents/skills/`.
- The design basis for this separation is `work/design/kanbanzai-1.0.md` §4.3, which establishes that skills are the product interface and `AGENTS.md` is project-specific.
- This is a documentation-only change. No Go code, no entity schemas, and no MCP tool behaviour are affected.