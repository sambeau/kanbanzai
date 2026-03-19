# Review: Initial Workflow Analysis

- Status: review
- Reviewing: initial-workflow-analysis.md
- In light of: workflow-system-design.md
- Date: 2026-03-18

---

## Purpose

This document reviews `initial-workflow-analysis.md` against `workflow-system-design.md` to identify ideas worth preserving, gaps in the design document, and issues in both — with the goal of producing a single consolidated design document.

---

## 1. Overall Assessment

The two documents are substantially aligned. They reach the same conclusions on every major architectural point: structured YAML state files in git, SQLite as a derived cache, a Go binary with MCP server interface, markdown as a view layer, distributed ID allocation, worktrees for isolation, team-scoped knowledge stores, lifecycle state machines, and phased implementation starting with the kernel.

The design principles are essentially the same list, with the same priorities.

Neither document has fundamental problems. They are two views of the same design from different distances: the initial analysis is a research and options document; the design document commits to specific choices with concrete schemas.

They should not be merged verbatim. Instead, the design document should be updated to incorporate the ideas worth keeping from the initial analysis, and the initial analysis should be preserved as the research trail.

---

## 2. Ideas in the Initial Analysis Missing From the Design Document

The following ideas from the initial analysis are substantive and should be incorporated into the design document.

### 2.1 The Intake / Canonical / Projection Taxonomy

**Initial analysis lines 385–421.**

The initial analysis explicitly names three classes of material:

- **Intake artifacts** — human-provided material that has not yet been normalised (notes, drafts, brainstorms, rough bug reports, review commentary)
- **Canonical records** — validated structured workflow objects written through formal MCP operations
- **Projections** — generated human-facing views derived from canonical state (markdown summaries, dashboards, handoff packets)

The design document discusses normalisation and generated reports but never names these three classes explicitly. This taxonomy is genuinely useful — it prevents the ambiguity that caused Basil's documentation drift by making it clear whether a given piece of text is raw input, authoritative state, or a derived view.

**Recommendation:** Add this taxonomy to the design document, likely as a subsection of the architecture or principles section.

### 2.2 The Detailed Bug Workflow

**Initial analysis lines 846–994.**

The design document has bug schemas and lifecycle states, but the initial analysis is significantly more thorough:

- The **standard bugfix path** (report → triage → reproduce → plan → fix → verify → close) with detail on what each step involves
- **Bug-vs-spec-change distinction** with the three categories (implementation defect, specification defect, design problem) and the different workflow paths each implies
- **Bug metadata families** beyond the core schema (`severity`, `impact_area`, `bug_class`, `introduced_by`, `detected_in`, `customer_visible`, `reproducible`, `requires_hotfix`, `requires_backport`, `verification_class`)
- **Conversational bug reporting** as a worked example of the normalisation principle

The design document has the bug-vs-spec-change distinction (section 5.5) and a basic schema, but the workflow path detail and metadata families are absent.

**Recommendation:** Expand the bug section of the design document with the standard bugfix path, the metadata families, and the conversational intake example.

### 2.3 Incident and Root-Cause Analysis

**Initial analysis lines 891–908.**

The initial analysis distinguishes incidents (production-significant failures: outages, data corruption, security issues, severe degradations) from ordinary bugs, and proposes root-cause analysis as a separate artifact type capturing: what happened, why, why it wasn't caught earlier, what changed, and what will prevent recurrence.

The design document treats bugs as the main problem entity and doesn't mention incidents or root-cause analysis.

For a production social media app, this distinction will matter. Not every bug is an incident, but every incident needs a more rigorous workflow than a standard bugfix.

**Recommendation:** Add Incident and RootCauseAnalysis as deferred entity types — not in phase 1, but acknowledged in the object model as future additions. Add a brief note in the bug section about when a bug escalates to an incident.

### 2.4 Metadata Governance

**Initial analysis lines 608–670.**

The initial analysis proposes that metadata tags beyond the core schema must be formally defined, with each tag having: name, meaning, value type, allowed values or format, scope, examples, owner, and a process for introduction. Tags should be registered in a central glossary or schema registry.

It also draws a useful distinction between text search (good for narrative discovery, exploratory lookup, fuzzy finding) and structured metadata (good for filtering, queueing, routing, dashboards, validation, automation). The recommendation: use both.

The design document doesn't address metadata governance at all. This matters — ungoverned metadata is how fields like `priority` end up meaning different things to different teams.

**Recommendation:** Add a metadata governance section to the design document. It doesn't need to be elaborate — a rule that all metadata fields are defined in a schema registry, with the fields the initial analysis lists, would be sufficient.

### 2.5 The Four-Tier Agent Hierarchy

**Initial analysis lines 790–842.**

The initial analysis distinguishes four levels of agent:

1. **Humans** — goals, priorities, product direction, approvals, ship decisions
2. **PM / orchestration agents** — roadmap decomposition, dependency tracking, consistency checking, backlog hygiene
3. **Specialist team agents** — backend, frontend, infrastructure, documentation, QA — each with scoped memory and scoped artifacts
4. **Execution agents** — short-lived workers implementing one task

The design document collapses this to three (human, orchestrator, worker). The specialist team layer — with its scoped memory and scoped artifacts — is worth preserving as a distinct concept, because it's the natural owner of the team knowledge stores and the level at which expertise accumulates.

**Recommendation:** Expand the delegation model to four tiers, restoring the specialist team layer between orchestrator and worker.

### 2.6 Parallelise by Conflict Domain

**Initial analysis lines 1059–1068.**

The design document says "file-level conflict prevention" but the initial analysis makes the broader point: parallelism should be based on file overlap, dependency ordering, architectural boundaries, and verification boundaries — not just team structure.

This is a subtler and more useful framing. Two features that both touch the auth module should be sequenced regardless of which team owns them. Two features in unrelated modules can be parallelised even if the same team owns both.

**Recommendation:** Replace "file-level conflict prevention" with "conflict domain awareness" and use the initial analysis's fuller list of dimensions.

### 2.7 Prefer Vertical Slices

**Initial analysis lines 1070–1072.**

The initial analysis explicitly recommends vertical slices over horizontal layers for parallel work. Tasks that represent a coherent end-to-end capability tend to parallelise better than broad horizontal layers.

The design document doesn't mention this. It's a useful planning heuristic worth including.

**Recommendation:** Add as a principle or guideline in the concurrency section.

### 2.8 External Tools Worth Considering

**Initial analysis lines 1294–1337.**

The initial analysis has a section listing categories of external tools worth investigating:

1. Fast indexed search / semantic retrieval for workflow artifacts
2. Schema validation tools for YAML
3. Git worktree lifecycle tooling
4. Task graph / dependency graph tooling
5. Static documentation renderers
6. CI enforcement hooks
7. Logging / append-only event capture

The design document doesn't have a research or tooling wishlist. Some of these — especially CI enforcement hooks and append-only event logs — are worth keeping visible as future considerations.

**Recommendation:** Add a brief "future considerations" or "tooling to evaluate" section to the design document.

### 2.9 The Four-Layer Agent Instruction Model

**Initial analysis lines 1087–1156.**

The initial analysis has a four-layer model for agent instructions:

1. Platform-native agent instructions (AGENTS.md, skills, coding rules)
2. Workflow system rules (schemas, state transitions, naming rules)
3. Generated context packets (task handoffs, briefings)
4. Workflow MCP interface (the formal control surface)

The design document has three layers (platform instructions, workflow rules, generated context packets) and describes the MCP interface separately. The initial analysis's framing is clearer about where MCP sits in the stack — it's not a separate concern but the fourth layer of the instruction model.

**Recommendation:** Adopt the four-layer model in the design document.

### 2.10 Richer Deferred Object Model

**Initial analysis lines 427–465.**

The initial analysis proposes more entity types than the design document:

| Initial Analysis | Design Document | Status |
|-----------------|-----------------|--------|
| Project | — | Missing |
| RoadmapItem | — (generated view) | Design doc treats as generated |
| Milestone | Epic | Renamed |
| Feature | Feature | Same |
| ResearchNote | — | Missing |
| Design | — | Missing (docs only) |
| Specification | — (part of Feature) | Folded in |
| ImplementationPlan | — (part of Feature) | Folded in |
| Task | Task | Same |
| Bug | Bug | Same |
| Incident | — | Missing |
| RootCauseAnalysis | — | Missing |
| Decision | Decision | Same |
| Approval | — (field on Feature) | Folded in |
| Release | — | Missing |
| KnowledgeEntry | Knowledge entry | Same |
| TeamMemoryEntry | Knowledge entry | Merged |

The design document's simplification is appropriate for phase 1, but some of the deferred types will be needed: Release, Incident, and Approval as an explicit trackable object (not just a status field). ResearchNote and Design may or may not earn their place — they could remain as unstructured markdown.

**Recommendation:** Keep the design document's phase 1 scope but add an explicit "deferred entity types" subsection listing what will be added later and when it's expected to become necessary.

---

## 3. Ideas in the Design Document Missing From the Initial Analysis

The design document has several concrete elements the initial analysis lacks. These should be preserved in the consolidated document:

1. **Concrete YAML schemas with populated example data** — the initial analysis lists fields; the design document shows what they look like in use.

2. **The full MCP tool surface** — specific tool names, parameters, and groupings by category. The initial analysis has a vaguer "example MCP tool categories" list.

3. **Concrete directory structure** — `work/state/epics/`, `work/state/features/`, `work/state/tasks/`, etc. with the one-file-per-entity rationale.

4. **The distributed ID block allocation mechanism** — the initial analysis discusses options (sequential, time-based, hybrid); the design document commits to block allocation and explains the workflow.

5. **The six-step conversational boundary** — interpret → clarify → validate → normalise → execute → report. The initial analysis has a similar pipeline (intake → interpretation → clarification → normalisation → formal commit → projection) at a higher level.

6. **Git worktree layout, merge strategy, and merge gates** — concrete and actionable where the initial analysis is principled but abstract.

7. **GitHub integration model** — what stays in `kanbanzai`, what goes to GitHub, and the principle that GitHub is a view, not the source of truth.

---

## 4. Problems in Both Documents

### 4.1 No Migration Path

Neither document addresses how an existing project (like Basil, with 149 features, 130 plans, and 26 bugs) migrates to this system. Importing existing workflow state into a new format is non-trivial and should at least be sketched. What gets imported? What gets archived? What's the cutover strategy?

**Recommendation:** Add a migration section, even if brief: "existing projects can be migrated by [approach], with [these] caveats."

### 4.2 No Rollback or Undo Story

Neither document addresses what happens when a workflow state change was wrong. A status update made in error, a decision recorded incorrectly, an entire feature's task decomposition that turns out to be misguided. Git history provides raw undo, but the workflow layer should have a cleaner story.

**Recommendation:** Add a brief section on error correction: how wrong state is fixed, whether there's a revert mechanism, and how the system handles the difference between "this was always wrong" and "this was right then but circumstances changed."

### 4.3 Normalisation Reliability

Both documents lean heavily on the AI agent to interpret, clarify, and normalise human input. But current AI agents sometimes normalise wrong — they silently change meaning while "cleaning up" text. The documents say the normalisation step should be reviewable, but neither is specific about how.

**Recommendation:** Be explicit: the agent must show a diff or summary of what it changed during normalisation, flag any places where it changed meaning (not just structure), and the human must confirm before the normalised version becomes canonical. This is especially important for specs and decisions, where a subtle meaning change can cascade.

### 4.4 Phase 1 Scope Creep Risk

Between the two documents, there are ~16 entity types, ~25 MCP tools, knowledge stores, worktree management, GitHub integration, and four tiers of agent delegation. Even with phased implementation, the phase 1 scope may be too large.

**Recommendation:** The consolidated document should be ruthlessly clear about what's in phase 1 and what's not. A good phase 1 might be just: Epic, Feature, Task, Bug, Decision as entity types; `create`, `status_update`, `search`, `health_check`, `doc_scaffold`, `doc_validate` as MCP tools; and the CLI. No knowledge stores, no worktree management, no GitHub sync, no orchestration until phase 1 is solid.

### 4.5 The ID Strategy Needs Testing

The design document commits to block allocation, which is reasonable. But neither document considers edge cases: what if a block runs out mid-feature? What if a branch is abandoned with unreturned IDs? What if two projects share a repository?

**Recommendation:** The consolidated document should at least acknowledge these edge cases and sketch answers, even if the full solution is deferred to implementation.

---

## 5. Structural Issues

### 5.1 The Initial Analysis Has Inconsistent Heading Levels

Several sections use `##` where they should use `###` (e.g. "Structured core, markdown surfaces" at line 241, "Standard metadata tags" at line 610, "Structured files" at line 676). This makes the document outline confusing. The consolidated document should have clean, consistent heading hierarchy.

### 5.2 The Design Document's Principles Are Stronger

The initial analysis ends with a numbered principles list (lines 1340–1357) that duplicates content from earlier sections. The design document integrates principles as named subsections (2.1 through 2.12) which is cleaner. The consolidated document should use the design document's approach.

### 5.3 The Initial Analysis Splits Related Content

The bug workflow, metadata governance, and agent hierarchy are scattered across the initial analysis rather than grouped with related architectural sections. The design document's numbered-section structure is better organised. The consolidated document should follow the design document's structure and fold the initial analysis's content into the appropriate sections.

---

## 6. Recommendations for the Consolidated Document

1. **Use the design document as the base.** Its structure, specificity, and commitment to concrete choices is the right foundation.

2. **Incorporate the following from the initial analysis:**
   - The intake / canonical / projection taxonomy (§2.1 above)
   - The detailed bug workflow and metadata families (§2.2)
   - Incident and root-cause analysis as deferred types (§2.3)
   - Metadata governance (§2.4)
   - The four-tier agent hierarchy (§2.5)
   - Conflict domain awareness (§2.6)
   - Vertical slice preference (§2.7)
   - External tools to evaluate (§2.8)
   - The four-layer agent instruction model (§2.9)
   - Deferred entity types (§2.10)

3. **Add new sections for:**
   - Migration path (§4.1)
   - Rollback and error correction (§4.2)
   - Normalisation review process (§4.3)
   - Explicit phase 1 scope boundary (§4.4)
   - ID strategy edge cases (§4.5)

4. **Preserve the initial analysis** as a companion research document. Mark it as superseded for architectural decisions but useful as a record of options considered and reasoning.

---

## 7. Summary

The two documents are convergent. The initial analysis is broader and more exploratory; the design document is more concrete and committed. The consolidated document should have the design document's structure and specificity, enriched with the initial analysis's deeper treatment of bugs, metadata, agent tiers, conflict domains, and the intake/canonical/projection taxonomy, plus new sections addressing migration, rollback, normalisation review, and scope discipline.
