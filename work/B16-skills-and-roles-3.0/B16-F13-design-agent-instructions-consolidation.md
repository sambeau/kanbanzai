# Design: Agent Instructions Consolidation

| Field | Value |
|-------|-------|
| Date | 2026-04-02 |
| Author | Design Agent |
| Status | Draft |
| Feature | Agent Instructions Consolidation |
| Plan | P16-skills-and-roles-3.0 |
| Review Report | `work/reports/agent-instructions-review.md` |
| Research Basis | `work/research/ai-agent-best-practices-research.md`, `work/research/agent-orchestration-research.md`, `work/research/agent-skills-research.md`, `work/research/skill-authoring-best-practices.md` |

---

## Overview

The review report (`work/reports/agent-instructions-review.md`) assessed all agent-facing
instruction files against three research reports and found the implementation is strong overall,
but identified four areas needing attention:

1. Three `.agents/skills/` files overlap heavily with new `.kbz/skills/` counterparts and
   contain significant unique content that hasn't been migrated.
2. The five retained `.agents/skills/` system skills lack the structural conventions
   (vocabulary, anti-patterns, evaluation criteria, retrieval anchors) that `.kbz/skills/`
   follows.
3. `AGENTS.md` has duplicated content, stale pointers, and a missing discovery path to
   `.kbz/skills/`.
4. `.kbz/` store discipline is not enforced — agents don't treat store changes as code changes.

This design addresses all 19 recommendations (R1–R19) from the review report, organised into
five work areas that can be executed largely in sequence.

---

## Goals and Non-Goals

### Goals

- Eliminate conflicting guidance by migrating unique content from three overlapping system
  skills into their `.kbz/skills/` counterparts, then retiring the old skills.
- Bring the five retained system skills up to the same structural quality as task-execution
  skills (vocabulary, anti-patterns, evaluation criteria, retrieval anchors).
- Fix store-commit discipline so agents treat `.kbz/state/` changes as code changes.
- Fix discovery gaps so agents entering via any bootstrap file can find the full
  role/skill/stage-binding system.
- Clean up `AGENTS.md` — deduplicate, remove stale content, add missing pointers.
- Create a `CLAUDE.md` bootstrap file for Claude Code users.
- Lay groundwork for skill evaluation by verifying context assembly behaviour.

### Non-Goals

- Building a full evaluation harness with automated test scenarios (R17). The review report
  marks this as Priority 5 and acknowledges it needs its own design. This feature will verify
  the prerequisites (evaluation criteria exist in all skills, context assembly ordering is
  correct) but the harness itself is out of scope.
- Cross-platform bootstrap files for Cursor, Windsurf, Cline, etc. The review report's
  §7 is a reference guide. Only `CLAUDE.md` is actionable now because Claude Code is the
  primary development platform.
- Changes to MCP tool descriptions. The review report found tool descriptions are already
  good (§1, Orchestration Research row 1). No changes needed.
- Changes to role YAML files beyond adding the "Store Neglect" anti-pattern to `base.yaml`.
  The review report rates all roles as Good or Excellent.

---

## Design

The work is organised into five areas matching the review report's priority structure.
Each area is self-contained — earlier areas do not depend on later ones, but Area 1 (migrations)
should complete before Area 4 (fix stale pointers) because the pointers need to point to the
migrated destinations.

### Area 1: Migrate and Retire Overlapping Skills (R1, R2, R3)

Three `.agents/skills/` files have heavy overlap with `.kbz/skills/` counterparts but contain
significant unique content. The migration strategy is: port the unique content into the new
skill, verify nothing is lost, then delete the old skill.

#### 1.1 `kanbanzai-code-review` → `review-code` + `orchestrate-review`

The old skill is 708 lines (exceeds the 500-line budget) and overlaps heavily with two new
skills. The review report (§4.3, Appendix B.1) identifies these unique content items to migrate:

**Into `review-code`:**
- Edge case playbooks (missing spec, partial implementation, ambiguous conformance, missing
  context). The new skill has terse STOP instructions; the old skill has nuanced multi-step
  handling for each case. Port the nuanced handling.

**Into `orchestrate-review`:**
- Remediation phase — task creation, conflict-check, re-review scoping, escalation cycle.
  The new skill says "route to remediation" but doesn't say *how*.
- Write review document — naming convention (`review-{id}-{slug}.md`), `doc(action: register)`.
  Review results need a persistent artifact.
- Human checkpoint integration — 3 trigger scenarios (ambiguous findings, high-stakes features,
  dimension disagreement).
- Context budget strategy — orchestrator ~6–14KB, sub-agent ~12–30KB. Add as a reference file
  or inline note.

**Verification before deletion:**
- Per-dimension evaluation questions — verify these exist in `reviewer-*.yaml` role files. The
  review report says they may already be covered. If any dimension lacks questions, port them.
- Tool chain reference (step → MCP tool mapping table) — evaluate whether this adds value as a
  reference file. If not, drop it.

**After migration:** Delete `.agents/skills/kanbanzai-code-review/`.

#### 1.2 `kanbanzai-plan-review` → `review-plan`

The review report (§4.3, Appendix B.2) identifies these gaps in the new skill:

**Into `review-plan`:**
- Criterion-by-criterion spec conformance — the old skill reads each acceptance criterion and
  verifies it against implementation code. The new skill only checks approval status.
- Cross-cutting checks — `go test -race ./...`, `health()`, `git status`. Concrete verification
  steps.
- Retrospective contribution step — feeds the project learning loop via `finish()`.
- Document registration step — write findings to `work/reviews/`, register with
  `doc(action: register)`.
- Spec Conformance Detail table in report format.
- Inputs/prerequisites list.

**After migration:** Delete `.agents/skills/kanbanzai-plan-review/`.

#### 1.3 `kanbanzai-design` → `write-design`

The review report (§4.3, Appendix B.3) identifies behavioural/philosophical content missing
from the new skill:

**Into `write-design`:**
- "Design with Ambition" stance — always present the ambitious version first. Add as a
  preamble section before the Procedure.
- Surfacing Risk — 3-tier escalation (minor → significant → stop). Add as a procedural
  section or step.
- Human/Agent role contract — "Human = Design Manager, Agent = Senior Designer."
- Six-quality evaluation lens — port `references/design-quality.md` to
  `.kbz/skills/write-design/references/design-quality.md`.
- Iterative process framing, draft lifecycle, design splitting guidance, gotchas
  (registration, content hash drift, `doc refresh`), next-steps-after-design.

**After migration:** Delete `.agents/skills/kanbanzai-design/` and its `references/` directory.

### Area 2: Upgrade Retained System Skills (R4, R5, R6)

The five retained `.agents/skills/` system skills are structurally weaker than `.kbz/skills/`.
They lack vocabulary sections, structured anti-patterns, evaluation criteria, and retrieval
anchors. Since these are the most frequently loaded skills (every session starts with
`kanbanzai-getting-started`), the quality gap matters.

**For each of the five skills** (`kanbanzai-getting-started`, `kanbanzai-workflow`,
`kanbanzai-agents`, `kanbanzai-documents`, `kanbanzai-planning`):

1. **Add a vocabulary section** (5–15 terms). These are system skills, not deep domain skills,
   so a shorter vocabulary is appropriate. Focus on terms that prime the agent's understanding
   of the Kanbanzai system: *stage binding*, *lifecycle gate*, *context packet*, *store
   discipline*, etc. Each skill should have vocabulary specific to its domain.

2. **Convert prose anti-patterns to Detect/BECAUSE/Resolve format.** The existing prose
   warnings are good content — they just need restructuring. This is a formatting change,
   not a content rewrite.

3. **Add evaluation criteria** (3–5 gradable questions per skill). System skills produce
   behaviour, not documents, so criteria should evaluate observable outcomes: "Did the agent
   commit `.kbz/` changes?" rather than "Is the document complete?"

4. **Add "Questions This Skill Answers" retrieval anchors** (5–8 per skill). These improve
   future discovery by exploiting the recency-bias position at the end of each skill.

**Constraint:** System skills must stay within their current approximate line count. The upgrade
adds ~40–60 lines of structured content per skill but this should be offset by tightening
existing prose. Do not exceed 350 lines for any system skill.

**Note on frontmatter:** System skills use Anthropic's native SKILL.md format (simple `name`
and `description` fields) rather than Kanbanzai's extended frontmatter (dual-register
description, triggers, roles, stage, constraint_level). This is deliberate — they are
discovered by Claude Code's native skill scanner, which expects the simpler format. Do not
add Kanbanzai-specific frontmatter fields.

### Area 3: Fix Store Discipline (R7)

The review report (§3.7) documents a high-severity gap: agents don't treat `.kbz/state/`
changes as code changes, causing store drift, silent data loss, and coordination failures.

**Four files need changes:**

1. **`kanbanzai-agents`** — Add to the commit discipline section:
   > Every `.kbz/state/` change is a code change. When you register a document, transition
   > an entity, contribute knowledge, or perform any action that modifies `.kbz/`, commit the
   > change immediately or include it in your next logical commit. Do not leave `.kbz/` files
   > uncommitted at the end of a task.

2. **`kanbanzai-getting-started`** — Add to the pre-task checklist:
   > If uncommitted `.kbz/` files exist from a previous session, commit them now. Do not
   > stash, discard, or ignore them — they are workflow state, not temporary files.

3. **`AGENTS.md`** — Add to the Git Discipline section:
   > `.kbz/state/` files are versioned project state, not ephemeral cache. Treat every store
   > change as a code change. Never stash, discard, or `.gitignore` these files.

4. **`.kbz/roles/base.yaml`** — Add a "Store Neglect" anti-pattern:
   - Detect: `.kbz/state/` files left uncommitted after a task or discarded/stashed at
     the start of a new task.
   - BECAUSE: the `.kbz/` store is the source of truth for parallel agents; uncommitted
     state causes drift, race conditions (MAST FM-2.2), and silent data loss (FM-2.5).
   - Resolve: commit `.kbz/` changes alongside code changes; at task start, commit any
     orphaned `.kbz/` files before proceeding.

### Area 4: Fix AGENTS.md and Discovery (R8–R16)

#### 4.1 AGENTS.md Cleanup (R8–R13)

These are incremental fixes to an already-lean file:

- **R8: Deduplicate pre-task checklist.** Keep project-specific items (git status, correct
  branch, check decision logs). Remove "read AGENTS.md" and "read design docs" — those are
  in `kanbanzai-getting-started`.
- **R9: Replace inline Document Reading Order** with a pointer to `refs/document-map.md`.
- **R10: Remove phase labels** from Repository Structure annotations. "(Phase 3)" adds no
  information the functional description doesn't already provide.
- **R11: Add a 5-term mini-vocabulary** at the top: *stage binding*, *role*, *skill*,
  *lifecycle gate*, *context packet*. One line each. Primes comprehension before skills load.
- **R12: Add a pointer to `.kbz/skills/` and `.kbz/stage-bindings.yaml`.** Currently AGENTS.md
  mentions `.agents/skills/` but not the task-execution skill system.
- **R13: Restructure Decision-Making Rules.** Replace phase-numbered log references with
  guidance to use the knowledge tool or consult `refs/document-map.md`.

#### 4.2 Fix Discovery Paths (R14, R15)

- **R14: Update `refs/document-map.md`** to point to `.kbz/skills/review-code/` and
  `.kbz/skills/review-plan/` instead of the old `.agents/skills/` versions. This must happen
  after Area 1 migrations complete.
- **R15: Fix `kanbanzai-getting-started` discovery.** Add mention of stage bindings and
  `.kbz/skills/` so agents bootstrapping through this skill discover the full system.

#### 4.3 Create CLAUDE.md (R16)

Create a `CLAUDE.md` bootstrap file for Claude Code users. Content mirrors
`.github/copilot-instructions.md` but kept shorter because it's loaded every turn.

Key content:
- Point to `AGENTS.md` for project-specific conventions
- Point to `.kbz/stage-bindings.yaml` for workflow stage → role + skill mapping
- List the role and skill tables (condensed)
- List critical rules (condensed)
- Keep under 150 lines

### Area 5: Verify Context Assembly (R18, R19)

These are verification tasks, not implementation tasks. The review report flags them as
"verify and fix if needed."

- **R18: Verify `internal/context/assemble.go` orders context packets following the
  attention curve.** Read the assembly pipeline and confirm that identity/constraints
  appear first, supporting material in the middle, and instructions/retrieval anchors
  last. If the ordering doesn't match, fix it.
- **R19: Verify that `handoff` includes effort budgets from stage bindings in assembled
  context packets.** Trace the code path from `handoff` through context assembly to confirm
  effort budget metadata is included. If not, add it.

---

## Alternatives Considered

### Alternative 1: Retire All System Skills, Absorb into Task-Execution Skills

Instead of upgrading the five retained system skills, absorb their content into the
`.kbz/skills/` task-execution skills and `AGENTS.md`.

**Rejected because:** The system skills operate at a fundamentally different layer. They
answer "how to use Kanbanzai" while task-execution skills answer "how to perform a specific
task." Task-execution skills depend on system skills for shared protocol (commit format,
finish specification, knowledge contribution). Inlining this into 11 task-execution skills
would duplicate ~330 lines of protocol across each skill. The two-layer architecture is
correct; the system layer just needs structural upgrading.

### Alternative 2: Build Evaluation Harness First

The review report says "Evaluation Must Precede Documentation." We could build the evaluation
harness (R17) before making any content changes, then measure the impact of each change.

**Rejected because:** The evaluation harness needs its own design and is Priority 5 in the
review report. The content migrations (R1–R3) are urgent because conflicting guidance
actively confuses agents today. The store discipline fix (R7) addresses a high-severity
operational problem. Waiting for evaluation infrastructure would delay fixing known issues
with measurable negative impact. The evaluation criteria in each skill provide a future hook
for automated evaluation when the harness exists.

### Alternative 3: Create Platform-Specific Bootstrap Files for All Platforms

The review report's §7 documents nine platforms. We could create bootstrap files for all
of them.

**Rejected because:** Only Claude Code (missing `CLAUDE.md`) has an actionable gap. All other
platforms either already work via MCP delivery or via the existing `AGENTS.md` /
`.github/copilot-instructions.md`. Creating files for Cursor (`.cursorrules`), Windsurf
(`.windsurfrules`), Cline (`.clinerules`), etc. adds maintenance burden for platforms we
don't actively test on. Users on those platforms can create their own bootstrap files from
the template in the review report's §7 if needed.

### Alternative 4: Full Rewrite of System Skills from Scratch

Instead of upgrading the five retained system skills incrementally (add vocabulary, restructure
anti-patterns, add evaluation criteria), rewrite them from scratch following `CONVENTIONS.md`.

**Rejected because:** The existing content in system skills is functionally correct and
well-tested through months of agent use. A from-scratch rewrite risks losing nuanced
procedural knowledge that was refined through operational experience. The incremental approach
preserves proven content while adding structural quality. If evaluation later shows a skill
isn't working, a targeted rewrite of that specific skill is the appropriate response.

---

## Dependencies

- **On P16 features (all done):** This feature builds on the role system, skill system, stage
  bindings, and context assembly pipeline delivered by P16. All are complete and stable.
- **On the review report:** The recommendations and migration checklists in
  `work/reports/agent-instructions-review.md` are the authoritative source for what to
  migrate. Implementers should read the relevant Appendix B checklist for each migration task.

---

## Risks

1. **Content loss during migration.** The migration checklists (Appendix B) are comprehensive
   but manual review is needed to confirm nothing is missed. Each migration task should include
   a verification step: compare the old skill's content against the new skill's content and
   confirm every unique item is either ported or explicitly dropped with justification.

2. **System skill line budget.** Adding vocabulary, anti-patterns, evaluation criteria, and
   retrieval anchors to system skills adds ~40–60 lines each. Some skills may need prose
   tightening to stay within the 350-line target. This is manageable but requires editorial
   judgment.

3. **Discovery regression.** Retiring three `.agents/skills/` files removes them from Claude
   Code's native skill discovery. If the discovery path through `copilot-instructions.md` →
   stage bindings → `.kbz/skills/` doesn't work reliably, agents may lose access to review
   and design guidance. Mitigation: verify discovery works end-to-end after retirement.

---

## Task Structure

The work decomposes into five sequential areas. Within each area, tasks can run in parallel
where they touch disjoint files.

| Area | Tasks | Dependencies | Files Affected |
|------|-------|-------------|----------------|
| **1. Migrate & Retire** | 3 (one per skill migration) | None | `.agents/skills/kanbanzai-code-review/`, `.agents/skills/kanbanzai-plan-review/`, `.agents/skills/kanbanzai-design/`, `.kbz/skills/review-code/`, `.kbz/skills/orchestrate-review/`, `.kbz/skills/review-plan/`, `.kbz/skills/write-design/` |
| **2. Upgrade System Skills** | 5 (one per skill) | None, but best after Area 1 | `.agents/skills/kanbanzai-getting-started/`, `.agents/skills/kanbanzai-workflow/`, `.agents/skills/kanbanzai-agents/`, `.agents/skills/kanbanzai-documents/`, `.agents/skills/kanbanzai-planning/` |
| **3. Store Discipline** | 1 (touches 4 files) | None | `.agents/skills/kanbanzai-agents/`, `.agents/skills/kanbanzai-getting-started/`, `AGENTS.md`, `.kbz/roles/base.yaml` |
| **4. AGENTS.md & Discovery** | 3 (AGENTS.md fixes, document-map + getting-started, CLAUDE.md) | Area 1 must complete before R14 | `AGENTS.md`, `refs/document-map.md`, `.agents/skills/kanbanzai-getting-started/`, `CLAUDE.md` (new) |
| **5. Verify Context Assembly** | 2 (assemble.go ordering, effort budgets) | None | `internal/context/assemble.go` (read + fix if needed) |

Total: ~14 tasks across 5 areas.

Parallelism opportunities:
- Area 1 tasks (3 migrations) can run in parallel — they touch disjoint file sets.
- Area 2 tasks (5 skill upgrades) can run in parallel — each touches a different skill.
- Area 3 touches files that overlap with Areas 2 and 4 (`kanbanzai-agents`,
  `kanbanzai-getting-started`, `AGENTS.md`), so it should be serialised with those areas.
- Area 5 tasks are independent and can run at any time.

Recommended execution order: Area 1 → Area 3 → Area 2 → Area 4 → Area 5.