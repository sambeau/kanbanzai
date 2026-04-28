# Review Report: Agent Instructions & Knowledge Quality

| Field | Value |
|-------|-------|
| Date | 2026-04-02 |
| Author | Review Agent |
| Status | Draft |
| Scope | All agent-facing instruction files: roles, skills, AGENTS.md, copilot-instructions.md, stage bindings |
| Research basis | `work/research/ai-agent-best-practices-research.md`, `work/research/agent-orchestration-research.md`, `work/research/agent-skills-research.md` |

---

## Executive Summary

This report evaluates the quality of all agent-facing instructions in the Kanbanzai project — roles (`.kbz/roles/`), task-execution skills (`.kbz/skills/`), system skills (`.agents/skills/`), `AGENTS.md`, `copilot-instructions.md`, and stage bindings — against the principles established in three prior research documents covering AI agent best practices, orchestration patterns, and skill architecture.

**The headline finding is that the implementation is strong.** The new roles and task-execution skills directly implement the majority of high-priority research recommendations: vocabulary routing, attention-optimised section ordering, named anti-patterns with BECAUSE clauses, domain-specific reviewer panels, copy-paste checklists, BAD/GOOD examples, and stage-binding architecture. The system is architecturally well-aligned with the research across all three documents.

**Four areas need attention:**

1. **Quality gap between skill layers.** The `.kbz/skills/` task-execution skills are excellent. The `.agents/skills/` system skills are functional but structurally weaker — they lack vocabulary sections, structured anti-patterns, evaluation criteria, and retrieval anchors. Since system skills are the most frequently loaded (every session starts with `kanbanzai-getting-started`), this gap matters.

2. **Three `.agents/skills/` files have heavy overlap with new `.kbz/skills/` counterparts** and contain significant unique content that hasn't been migrated. Until that content is ported, retiring them risks losing procedural depth. After migration, retiring them eliminates conflicting guidance and a 708-line file that exceeds the 500-line budget.

3. **`AGENTS.md` has duplicated checklist items, stale-risk content, and a missing pointer** to the `.kbz/skills/` system. The fixes are incremental — the file is lean and well-scoped, but the highest-attention content is the most duplicated.

4. **`.kbz/` store discipline is not enforced.** Agents do not treat `.kbz/state/` changes as code changes, leaving document records, entity transitions, and knowledge entries uncommitted. This causes store drift between parallel agents, corrupted state when new tasks begin, and silent data loss when agents stash or discard unfamiliar files. No skill or instruction file currently makes this obligation explicit.

The report also includes a cross-platform discovery guide for agent instruction files, documenting how each major AI coding platform finds and loads project-level instructions. This serves both Kanbanzai's own setup and as reference material for users adopting Kanbanzai on different platforms.

---

## Table of Contents

- [1. Overall Assessment Against Research Principles](#1-overall-assessment-against-research-principles)
- [2. What Meets or Exceeds the Research](#2-what-meets-or-exceeds-the-research)
- [3. Gaps and Improvement Opportunities](#3-gaps-and-improvement-opportunities)
- [4. Overlap Analysis: `.agents/skills/` vs `.kbz/skills/`](#4-overlap-analysis-agentsskills-vs-kbzskills)
- [5. AGENTS.md Review](#5-agentsmd-review)
- [6. Cross-System Coherence and Discovery Path](#6-cross-system-coherence-and-discovery-path)
- [7. Cross-Platform Agent Discovery Guide](#7-cross-platform-agent-discovery-guide)
- [8. Consolidated Recommendations](#8-consolidated-recommendations)
- [Appendix A: File Inventory](#appendix-a-file-inventory)
- [Appendix B: Migration Checklists](#appendix-b-migration-checklists)

---

## 1. Overall Assessment Against Research Principles

### Best Practices Research (10 Principles)

| Research Principle | Status | Notes |
|---|---|---|
| **P1: Hardening** — deterministic tools for mechanical work | ✅ Strong | MCP tools handle all workflow mechanics |
| **P2: Context Hygiene** — attention-optimised, progressive | ✅ Good | Section ordering codified in CONVENTIONS.md; progressive disclosure partial |
| **P3: Living Documentation** — fresh, structured, example-rich | ✅ Good | BAD/GOOD examples present; freshness tracking not yet on skills |
| **P4: Disposable Blueprint** — versioned plan artifacts | ✅ Strong | Feature lifecycle, worktrees, document approval |
| **P5: Institutional Memory** — codified corrections | ✅ Good | Knowledge system exists; BECAUSE format in anti-patterns |
| **P6: Specialized Review** — vocabulary-routed specialists | ✅ Strong | 5 reviewer profiles with domain vocabulary |
| **P7: Observability** — structured logging, failure detection | ⚠️ Partial | Artifact trail good; sub-agent handoff logging absent |
| **P8: Strategic Human Gates** — low-friction, high-information | ✅ Strong | Checkpoint tool, stage gates, document approval |
| **P9: Token Economy** — cascade escalation, team caps | ✅ Good | Effort budgets in stage bindings; dispatch ceiling of 4 |
| **P10: Toolkit** — encode principles into tools | ✅ Strong | MCP server, stage bindings, context assembly |

### Orchestration Research

| Finding | Status | Notes |
|---|---|---|
| **Agent-Computer Interface** — tool descriptions matter | ✅ | MCP tool descriptions include "use INSTEAD OF" guidance |
| **Enforceable Constraints** — gates at tool level | ✅ | Lifecycle gates enforced by MCP tools |
| **Decomposition Quality** — structured task breakdowns | ✅ | `decompose-feature` skill with dependency graph |
| **Architecture Matches Task** — single-agent for sequential | ✅ | Stage bindings specify orchestration pattern per stage |
| **Proactive Orchestration** — decompose → refine → assign | ✅ | `orchestrate-development` skill codifies this pattern |

### Skills Research

| Finding | Status | Notes |
|---|---|---|
| **Conciseness** — only project-specific content | ✅ | Vocabulary terms are domain-specific; procedures are specific |
| **Freedom Levels** — matched to task risk | ✅ | `constraint_level` in frontmatter |
| **Progressive Disclosure** — reference files for detail | ⚠️ Partial | System skills have 2 ref files; task skills have 0 |
| **Copy-paste Checklists** | ✅ | Present in all workflow-critical skills |
| **Examples Beat Rules** | ✅ | BAD/GOOD pairs in task-execution skills |
| **Evaluation-Driven Development** | ❌ | Criteria exist but no evaluation process |
| **Consistent Terminology** | ✅ | Vocabulary sections enforce this per skill |
| **Skill System Consolidation** | ⚠️ | Two systems exist; split is justified but three files overlap heavily |
| **500-line Budget** | ✅ | All `.kbz/skills/` within budget; one `.agents/` skill over (708 lines) |

---

## 2. What Meets or Exceeds the Research

### 2.1 Vocabulary Routing

The research identified vocabulary routing as the **#1 quality lever** (Theme 1, R1, R5). The implementation delivers:

- **Every role** has a vocabulary section with domain-specific terms. `reviewer-security.yaml` has 15 terms including "OWASP Top 10 (2021 edition)", "STRIDE threat model", "CWE weakness classification". `implementer-go.yaml` has 12 terms like "goroutine leak (context cancellation, defer cleanup)" and "interface segregation (accept interfaces, return structs)."
- **Every task-execution skill** has a Vocabulary section with terms that pass the **15-year practitioner test** — terms a senior expert would use when speaking with a peer.
- `CONVENTIONS.md` codifies the vocabulary format and mandates its placement in the **highest-attention position** (first body section).

### 2.2 Attention-Optimised Section Ordering

The research (P2, Appendix) prescribes: vocabulary first → anti-patterns → procedure → examples → retrieval anchors last. `CONVENTIONS.md` codifies this ordering as mandatory:

1. Vocabulary — highest-attention position, activates knowledge clusters
2. Anti-Patterns — what NOT to do, before the procedure
3. Checklist — optional, for medium/low constraint skills
4. Procedure — numbered steps with IF/THEN conditions
5. Output Format — structured template
6. Examples — BAD/GOOD pairs
7. Evaluation Criteria — gradable questions with weights
8. Questions This Skill Answers — retrieval anchors, final section

Every `.kbz/skills/` skill follows this ordering.

### 2.3 Named Anti-Patterns with BECAUSE Clauses

The research (P5, Zamfirescu-Pereira "Why Johnny Can't Prompt") found "Always/Never X BECAUSE Y" is the optimal constraint format. Every role and task-execution skill implements this with a structured **Detect / BECAUSE / Resolve** format. The BECAUSE clauses explain *why*, not just *what*. Example from `review-code`:

> **Rubber-Stamp Review (MAST FM-3.1)**
> - **Detect:** Verdict is "approved" with zero findings AND no per-dimension evidence citations.
> - **BECAUSE:** LLM sycophancy makes approval the path of least resistance. FM-3.1 is the #1 quality failure mode in multi-agent systems (MAST, 2024).
> - **Resolve:** Require at least one finding OR substantive per-dimension evidence for every clearance.

`CONVENTIONS.md` enforces BECAUSE clause quality: "The BECAUSE clause must explain *why*, not restate *what*."

### 2.4 Domain-Specific Reviewer Profiles

The research (P6, R5) recommended replacing a single reviewer with specialised profiles. The implementation delivers 5:

| Profile | Identity | Vocabulary Terms | Unique Anti-Patterns |
|---|---|---|---|
| `reviewer` (base) | Senior code reviewer | 6 (finding classification, evidence-backed verdict, etc.) | 4 (Rubber-stamp, Dimension Bleed, Prose Commentary, Severity Inflation) |
| `reviewer-conformance` | Senior requirements verification engineer | 6 (acceptance criteria traceability, conformance matrix, etc.) | 3 (Assumed Conformance, Partial Verification, Phantom Requirement) |
| `reviewer-quality` | Senior software quality engineer | 9 (cyclomatic complexity, resource lifecycle, etc.) | 3 (Style-as-defect, Nitpick Escalation, Improvement Disguised as Defect) |
| `reviewer-security` | Senior application security engineer | 15 (OWASP, STRIDE, CWE, CVSS, etc.) | 5 (Checkbox Compliance, Scope Creep into Exploitation, Framework Trust, etc.) |
| `reviewer-testing` | Senior test engineer | 10 (boundary value analysis, mutation testing, test pyramid, etc.) | 5 (Coverage Theater, Mock Overuse, Happy-Path-Only, Test Coupling, Assertion-Free) |

The `orchestrate-review` skill handles adaptive dispatch — selecting specialists based on what actually changed (Captain Agent research: 15–25% improvement from adaptive composition).

### 2.5 Copy-Paste Checklists

The skills research (R1, Theme 5) identified checklists as "the most reliable way to prevent agents from skipping steps." Every task-execution skill includes a checklist. System skills include stage gate checklists, session start checklists, and task lifecycle checklists.

### 2.6 BAD/GOOD Examples with Explanations

The research (P3, Theme 4 "Examples Beat Rules") recommended 2–3 BAD vs GOOD pairs. Task-execution skills deliver this: `write-spec` has 3 (2 BAD, 1 GOOD), `review-code` has 3, `implement-task` has 3, `orchestrate-review` has 2, `orchestrate-development` has 3. `CONVENTIONS.md` codifies the recency bias principle: "Place the best GOOD example last."

### 2.7 Role/Skill Separation

Roles define *identity* (who you are — vocabulary, anti-patterns, tool constraints). Skills define *procedure* (what you're doing — steps, output format, checklist). This clean separation allows composition: the same `review-code` skill serves 5 different reviewer roles, each bringing different vocabulary lenses.

### 2.8 Tool Scoping per Role

Each role defines its allowed tools. `implementer-go` doesn't get `decompose` or `merge`. Reviewer roles don't get `worktree` or `pr`. This addresses the research on adaptive MCP tool filtering (R7) and the token economy principle (P9).

### 2.9 Stage Bindings as Single Source of Truth

`.kbz/stage-bindings.yaml` maps each workflow stage to its role, skill, prerequisites, orchestration pattern, effort budget, and document template. This is a direct implementation of the "mandatory orientation step" recommendation from the orchestration research (§3.5).

### 2.10 Evaluation Criteria

Every task-execution skill includes evaluation criteria phrased as gradable questions with weights (`required`, `high`, `medium`). These support automated LLM-as-judge evaluation — a prerequisite for measuring skill effectiveness.

### 2.11 Uncertainty Protocol

Every skill procedure includes explicit STOP instructions for ambiguous or missing inputs. This implements the hallucination reduction research finding about explicit uncertainty handling.

---

## 3. Gaps and Improvement Opportunities

### 3.1 System Skills Lack the New Architecture — ⚠️ Medium-High

The `.agents/skills/` system skills do **not** follow `CONVENTIONS.md`. They lack:

| Convention | `.kbz/skills/` (task) | `.agents/skills/` (system) |
|---|---|---|
| Dual-register description (`expert` + `natural`) | ✅ | ❌ Simple description only |
| Vocabulary section | ✅ (15–30 terms) | ❌ None |
| Anti-patterns (Detect/BECAUSE/Resolve) | ✅ | ❌ Prose paragraphs |
| Evaluation Criteria | ✅ | ❌ None |
| "Questions This Skill Answers" anchors | ✅ | ❌ None |
| Output Format section | ✅ | ❌ None |
| `constraint_level` in frontmatter | ✅ | ❌ Not present |

Since system skills are the most frequently loaded (every session starts with `kanbanzai-getting-started`), this quality gap affects every agent session.

**Recommendation:** Upgrade the 5 system skills being retained to match `CONVENTIONS.md` conventions. Add vocabulary sections, structured anti-patterns, evaluation criteria, and retrieval anchors.

### 3.2 No Progressive Disclosure in Task-Execution Skills — ⚠️ Medium

The `.agents/skills/` system skills have started using references (2 files: `lifecycle.md`, `design-quality.md`). The `.kbz/skills/` task-execution skills have **zero reference files**. Most are within the 500-line budget (largest is `orchestrate-review` at 375 lines), so this isn't urgent — but skills like `write-spec` (331 lines) with extensive examples could benefit from moving detailed examples to reference files.

**Recommendation:** Add `references/` directories to `.kbz/skills/` as skills grow. Not urgent but prepares for growth.

### 3.3 No Evaluation Framework — ⚠️ High

The research was emphatic: "Evaluation Must Precede Documentation." Skills include Evaluation Criteria sections (a prerequisite for evaluation), but there is no actual evaluation process — no test scenarios, no before/after measurement, no tracking of which skills improve output quality.

**Recommendation:** Build a minimal evaluation harness. Even 2–3 test scenarios per skill would validate effectiveness. Run the same task with and without the skill loaded and compare using the Evaluation Criteria to score.

### 3.4 `kanbanzai-code-review` at 708 Lines — ⚠️ Low-Medium

This legacy skill exceeds the 500-line budget and overlaps heavily with `review-code` + `orchestrate-review`. If it's still discovered by Claude Code's native scanner, agents may receive conflicting guidance. See §4 for the full migration analysis.

### 3.5 Effort Budget Enforcement — ⚠️ Low

`stage-bindings.yaml` includes effort budgets per stage (e.g., "5–15 tool calls" for specifying). These are advisory — there's no tool-level enforcement or evidence that they're included in assembled context packets.

**Recommendation:** Verify that `handoff` includes effort budgets in the context packet. If not, add them.

### 3.6 Position-Aware Context Assembly — ⚠️ Medium

Skills follow attention-curve ordering internally, but there's no guarantee that `handoff` or `next` orders the parts of a multi-part context packet optimally (identity/constraints first, supporting material middle, instructions/anchors last). The implementation of `internal/context/assemble.go` would need verification.

**Recommendation:** Audit `assemble.go` to confirm context packet ordering follows the attention curve. If not, reorder.

### 3.7 `.kbz/` Store Discipline Not Enforced — ⚠️ High

Since the new agent-facing instructions were added, agents have not been keeping the `.kbz/` store and git in sync. Three observed failure modes:

**1. Store changes not committed.** Agents do not treat `.kbz/state/` changes as code changes. When a document is registered, an entity transitions, or a knowledge entry is contributed, the corresponding YAML file in `.kbz/state/` changes on disk — but agents leave these uncommitted. This causes store drift between parallel agents and means the store is often dirty when a new task starts.

**2. Uncommitted store files discarded or stashed.** When agents encounter uncommitted `.kbz/` files at the start of a task (left by a previous agent or session), they sometimes stash or discard them rather than committing them. This silently loses workflow state — document records, entity transitions, knowledge entries — corrupting the store.

**3. No instruction makes this obligation explicit.** The `kanbanzai-agents` skill says "commit at logical checkpoints" and the `kanbanzai-getting-started` checklist says "commit coherent changes", but neither explicitly states that `.kbz/state/` changes are real changes that must be committed. The pre-task checklist in `AGENTS.md` says to commit "changes from previous work" but does not call out `.kbz/` files specifically. Agents appear to treat `.kbz/` as an ephemeral cache rather than versioned project state.

**Why this matters:** The `.kbz/` store is the source of truth for workflow state. It is tracked in git deliberately — so that parallel agents see the same state, so that `git bisect` works across workflow changes, and so that store corruption is recoverable. Uncommitted store changes are invisible to other agents, making parallel work unreliable and causing the exact coordination failures (MAST FM-2.2 race conditions, FM-2.5 resource contention) that the store was designed to prevent.

**Recommendation:** Add explicit store-commit discipline to three places:

1. **`kanbanzai-agents`** — Add to the commit discipline section: "Every `.kbz/state/` change is a code change. When you register a document, transition an entity, contribute knowledge, or perform any action that modifies `.kbz/`, commit the change immediately or include it in your next logical commit. Do not leave `.kbz/` files uncommitted at the end of a task."

2. **`kanbanzai-getting-started`** — Add to the pre-task checklist: "If uncommitted `.kbz/` files exist from a previous session, commit them now. Do not stash, discard, or ignore them — they are workflow state, not temporary files."

3. **`AGENTS.md`** — Add to the Git Discipline section: "`.kbz/state/` files are versioned project state, not ephemeral cache. Treat every store change as a code change. Never stash, discard, or `.gitignore` these files."

4. **`base.yaml` role** — Add an anti-pattern: "Store Neglect — Detect: `.kbz/state/` files left uncommitted after a task or discarded/stashed at the start of a new task. BECAUSE: the `.kbz/` store is the source of truth for parallel agents; uncommitted state causes drift, race conditions, and silent data loss. Resolve: commit `.kbz/` changes alongside code changes; at task start, commit any orphaned `.kbz/` files before proceeding."

---

## 4. Overlap Analysis: `.agents/skills/` vs `.kbz/skills/`

### 4.1 Summary

| `.agents/skills/` Skill | Lines | Overlaps with `.kbz/skills/`? | Verdict |
|---|---|---|---|
| `kanbanzai-getting-started` | 76 | **No overlap** | ✅ Keep — unique session bootstrap |
| `kanbanzai-workflow` | 221 | **Surface-level only** | ✅ Keep — defines stage *edges*, not stage *content* |
| `kanbanzai-agents` | 330 | **Moderate** — referenced by task skills | ✅ Keep — shared protocol (commits, finish, knowledge) |
| `kanbanzai-documents` | 224 | **Minor** (~5 overlapping points) | ✅ Keep — document system "theory of operations" |
| `kanbanzai-planning` | 222 | **Almost none** | ✅ Keep — completely different stage (pre-design scoping) |
| `kanbanzai-design` | 252 | **Moderate** with `write-design` | ⚠️ Migrate then retire |
| `kanbanzai-code-review` | 708 | **Heavy** with `review-code` + `orchestrate-review` | ⚠️ Migrate then retire |
| `kanbanzai-plan-review` | 258 | **Heavy** with `review-plan` | ⚠️ Migrate then retire |

### 4.2 The Five That Should Stay

These system skills operate at a fundamentally different layer than task-execution skills. They answer *how to use Kanbanzai* rather than *how to perform a specific task*. Task-execution skills depend on them.

**`kanbanzai-getting-started`** — Session bootstrap. No task-execution skill covers "I just opened a repo and don't know what to do." This is the entry point to the entire system.

**`kanbanzai-workflow`** — Stage gates, human/agent ownership, the emergency brake, lifecycle state machines. Task-execution skills operate *within* stages; this skill defines the *edges between* them and the rules for when to stop and ask.

**`kanbanzai-agents`** — The shared agent protocol: commit message format, `finish()` specification, knowledge contribution, retrospective format, sub-agent spawning, human communication conventions. Every task-execution skill delegates to this for protocol details. Inlining it would mean duplicating across 11 skills.

**`kanbanzai-documents`** — The document system "theory of operations": type taxonomy, approval lifecycle (`draft → approved → superseded`), content-hash drift, supersession protocol, batch import, commit discipline for registration records. Task-execution skills know *to call* `doc(action: register)` but this skill explains *how the system works*.

**`kanbanzai-planning`** — Pre-design scope conversation. Operates at a completely different stage than `write-dev-plan` (post-spec) or `decompose-feature` (post-spec). Covers the human-led planning conversation: "What are we building?", feature vs. plan sizing, design-with-ambition philosophy, conversational drift detection.

### 4.3 The Three That Should Be Retired After Migration

#### `kanbanzai-code-review` → `review-code` + `orchestrate-review`

The new skills have better structure (vocabulary, anti-patterns, examples, evaluation criteria) but the old skill has significant unique content:

| Unique Content | Priority | Migration Target |
|---|---|---|
| **Per-dimension evaluation questions** (5 dimensions × 5–7 questions) | High | Verify these exist in `reviewer-*.yaml` role files; migrate if not |
| **Edge case playbooks** (missing spec, partial implementation, ambiguous conformance) | High | Add to `review-code` — the new skill's terse STOP instructions are insufficient for nuanced cases |
| **Remediation phase** (task creation, conflict-check, re-review scoping, escalation) | High | Add to `orchestrate-review` — the new skill says "route to remediation" but doesn't say *how* |
| **Write review document** (naming convention, `doc(action: register)`) | Medium | Add to `orchestrate-review` — review results need a persistent artifact |
| **Human checkpoint integration** (3 trigger scenarios: ambiguous findings, high-stakes, disagreement) | Medium | Add to `orchestrate-review` |
| **Context budget strategy** (orchestrator ~6–14KB, sub-agent ~12–30KB) | Low | Reference file or note in `orchestrate-review` |
| **Tool chain reference** (step → MCP tool mapping table) | Low | Reference file |

#### `kanbanzai-plan-review` → `review-plan`

The new `review-plan` has better structure but is missing substantive procedural depth:

| Unique Content | Priority | Migration Target |
|---|---|---|
| **Criterion-by-criterion spec conformance** (read acceptance criteria, verify against code per criterion) | High | Add as new step — the new skill only checks approval status, not actual delivery conformance |
| **Cross-cutting checks** (`go test -race`, `health()`, `git status`) | High | Add as new step — concrete verification steps that catch real problems |
| **Retrospective contribution step** | Medium | Add to procedure — feeds the project learning loop |
| **Document registration step** (write to `work/reviews/`, register with `doc`) | Medium | Add to procedure — ensures review report enters doc governance |
| **Spec conformance detail table** in report format | Low | Add to output format |

#### `kanbanzai-design` → `write-design`

The new `write-design` is mechanically stronger but misses behavioral/philosophical content:

| Unique Content | Priority | Migration Target |
|---|---|---|
| **"Design with Ambition"** — always present the ambitious version first | High | Add as stance preamble to `write-design` |
| **Surfacing Risk** — 3-tier escalation (minor → significant → stop) | High | Add as procedural section |
| **Human/Agent role contract** — who decides vs. who proposes | High | Add to procedure or preamble |
| **Six-quality evaluation lens** (from `references/design-quality.md`) — Simplicity, Minimalism, Completeness, Composability + Honest, Durable | High | Port to `.kbz/skills/write-design/references/design-quality.md` |
| **Iterative process framing** — design is messy, that's normal | Medium | Add to procedure |
| **Gotchas** — registration, content hash drift, `doc refresh` | Medium | Add to `write-design` |
| **Design splitting guidance** — signs + supersession protocol | Medium | Expand Step 1.4 in `write-design` |

---

## 5. AGENTS.md Review

### 5.1 Section-by-Section Overlap Analysis

| Section | Lines | Overlaps With | Severity |
|---|---|---|---|
| **Overview** | 1–16 | `copilot-instructions.md` opening, `kanbanzai-getting-started` Purpose | Low — each version is brief, serves different entry point |
| **Naming Conventions** | 18–34 | Nothing | None — **unique** |
| **Self-Managed Development** | 36–38 | Nothing | None — unique framing |
| **Repository Structure** | 40–80 | Nothing | None — **unique, high-value** |
| **Before Every Task Checklist** | 82–96 | `kanbanzai-getting-started`, `kanbanzai-agents`, `implement-task` | **High** — git status + read AGENTS.md appear in 2+ places |
| **Document Reading Order** | 98–108 | `refs/document-map.md` | **Moderate** — parallel list that can go stale |
| **Decision-Making Rules** | 112–126 | Nothing | None — **unique, high-value** |
| **Git Discipline** | 128–154 | `kanbanzai-agents` Commit Format + Commit Discipline | **Moderate** — "commit at logical checkpoints" is word-for-word in 3 places |
| **Scope Guard** | 156–172 | Nothing | None — **unique** |
| **Build and Test Commands** | 174–184 | Nothing | None — **unique** |
| **Go Code Style and Testing** | 186–190 | Nothing (routing only) | None |
| **Codebase Knowledge Graph** | 192–194 | `copilot-instructions.md` Critical Rules | Low — both are routing pointers |
| **Delegating to Sub-Agents** | 196–198 | `kanbanzai-agents` Sub-Agent Spawning | **Moderate** — complementary but overlapping |

### 5.2 What AGENTS.md Does Well

**Clearly scoped purpose.** The "Self-Managed Development" section explicitly states the file "contains only project-specific instructions for developing the kanbanzai server itself." Good separation from system-level skills.

**Unique, high-value content.** Several sections have no equivalent elsewhere:
- **Repository Structure** — the only structural map of the codebase
- **Decision-Making Rules** — the 4 decision logs and "check before inventing" protocol
- **Scope Guard** — the explicit "do not build" list
- **Naming Conventions** — Kanbanzai / `kanbanzai` / `.kbz/` distinction
- **Build and Test Commands** — the Go toolchain commands

**Good delegation pattern.** Delegates to skills and refs/ files: "For commit message format, see the `kanbanzai-agents` skill", "See `refs/go-style.md` for full conventions." This is the progressive disclosure pattern.

**Reasonable length.** At 194 lines, well within attention-budget guidelines.

### 5.3 Quality Issues

**1. No vocabulary section.** Per the research (P6, Theme 1), vocabulary is the primary quality lever. AGENTS.md has naming conventions for the product name but no vocabulary priming for the development domain. Terms like "lifecycle gate", "stage binding", "context packet", "review unit" are used throughout the system but never introduced in the first file an agent reads.

*Counterpoint:* AGENTS.md says it's "project-specific instructions" — vocabulary belongs in roles/skills. Defensible, but the research says the 5–10 most critical terms should appear in the first file an agent reads because they route all subsequent comprehension.

**2. The pre-task checklist is duplicated at the wrong abstraction level.** The checklist mixes project-specific hygiene (git status, correct branch) with system-level workflow (read AGENTS.md, read design docs). The project-specific items belong here; the system-level items are already in `kanbanzai-getting-started`. The research (P2) says every token in the highest-attention position must be unique.

**3. Document Reading Order is stale-risk content.** The "Essential reads" list points to 5 design documents inline. If these are renamed, moved, or superseded, AGENTS.md becomes a stale pointer. `refs/document-map.md` is the canonical version and already exists.

**4. Decision-Making Rules use phase numbers.** The instruction to "check all four decision logs" lists them by phase number (Phase 1, 2, 3, 4). An agent doesn't know what "Phase 3" means. The intent is right but the mechanism is brittle.

**5. No anti-patterns section.** Per the research, named anti-patterns with BECAUSE clauses are the second most effective instruction format. AGENTS.md has none. The "Why this matters" paragraph under the checklist is the closest equivalent, but it's prose rather than structured.

**6. Repository Structure uses phase labels.** Every directory annotation includes "(Phase 2a)", "(Phase 3)", etc. This is time-sensitive content that will become meaningless to new agents. "(Phase 3)" adds no information that the functional description ("post-merge cleanup") doesn't already provide.

**7. Missing pointer to `.kbz/skills/` and stage bindings.** AGENTS.md references `.agents/skills/kanbanzai-*/SKILL.md` but not the task-execution skills or stage-bindings system. An agent entering via AGENTS.md won't discover `.kbz/skills/` at all.

**8. `refs/document-map.md` points to old skills.** The document map still references `kanbanzai-code-review` and `kanbanzai-plan-review` as canonical skills. If those are being retired, this pointer is stale.

---

## 6. Cross-System Coherence and Discovery Path

### 6.1 How an Agent Discovers Instructions Today

The full path from "agent opens repo" to "agent knows what to do":

1. **Platform auto-loads entry file** → `.github/copilot-instructions.md` (Copilot) or `CLAUDE.md` (Claude Code, if it existed) or `AGENTS.md` (Jules/Gemini)
2. **Entry file points to AGENTS.md** → project-specific conventions, structure, build commands
3. **Entry file points to stage bindings** → `.kbz/stage-bindings.yaml`
4. **Stage bindings point to role + skill** → agent reads the role YAML and skill SKILL.md for their current stage
5. **MCP tools deliver dynamic context** → `next()`, `handoff()` assemble context packets with spec sections, knowledge entries, file paths

This is a well-designed progressive disclosure chain. An agent with access to the copilot-instructions entry point can navigate to the right instructions for any stage.

### 6.2 Discovery Failure Points

1. **Agents that don't read copilot-instructions.md.** They'll only find `.agents/skills/` via native skill discovery and won't know about `.kbz/skills/`, roles, or stage bindings at all.

2. **Stage bindings require a two-hop lookup.** Read `stage-bindings.yaml`, then read the skill file it points to. Some agents may not follow the chain.

3. **AGENTS.md doesn't mention `.kbz/skills/`.** It references `.agents/skills/kanbanzai-*/SKILL.md` but delegates task-execution skills entirely to `copilot-instructions.md`. An agent that reads AGENTS.md but not the copilot-instructions misses the entire task-execution skill system.

4. **No `CLAUDE.md` exists.** Claude Code users depend on `.agents/skills/` native discovery and whatever AGENTS.md says. They have no Claude-Code-specific bootstrap file.

5. **`kanbanzai-getting-started` doesn't mention the role/skill system.** It says "If unsure about the current stage, check `kanbanzai-workflow`" but never mentions stage bindings, roles, or `.kbz/skills/`. An agent bootstrapping through this skill won't discover the task-execution layer.

---

## 7. Cross-Platform Agent Discovery Guide

This section documents how each major AI coding platform discovers and loads project-level instructions, and how Kanbanzai should be configured for each.

### 7.1 Platform Reference

#### Claude Code (Anthropic CLI)

| Aspect | Detail |
|---|---|
| **Auto-loaded** | `CLAUDE.md` at repo root — always loaded. Also walks parent directories and `~/.claude/CLAUDE.md` for user-global instructions. Subdirectory `CLAUDE.md` files loaded when agent works in that directory. |
| **Rules directory** | `.claude/commands/*.md` — custom slash commands (user-invoked, not auto-loaded). |
| **Skill format** | Plain markdown. No frontmatter or special schema. |
| **Loading order** | All levels additive (concatenated): user-global → parent dirs → repo root → subdirectories. |
| **Size limits** | Loaded every turn — large files consume context on every message. Keep concise. |
| **MCP support** | ✅ Full — configured via `.claude/mcp.json`. |

**Kanbanzai setup:** Create a `CLAUDE.md` that mirrors the bootstrap content in `.github/copilot-instructions.md` — point to `AGENTS.md`, stage bindings, roles, and skills. Keep it under 200 lines since it's loaded every turn.

#### GitHub Copilot (VS Code, JetBrains, GitHub.com)

| Aspect | Detail |
|---|---|
| **Auto-loaded** | `.github/copilot-instructions.md` — always injected as system prompt addition. |
| **Rules directory** | `.github/instructions/*.md` — additional files with optional YAML frontmatter for glob scoping (`applyTo: "**/*.go"`). Attached when matching files are in context. |
| **Prompt library** | `.github/prompts/*.md` — reusable prompts, user-invoked only. |
| **Loading order** | `copilot-instructions.md` always → `instructions/*.md` by glob match → `prompts/*.md` on explicit reference. Organization-level instructions layer below repo-level. |
| **Size limits** | Recommended under ~8KB. Larger files may be truncated. |
| **MCP support** | ✅ — configured in VS Code settings or `.vscode/mcp.json`. |

**Kanbanzai setup:** Already configured. `.github/copilot-instructions.md` serves as the bootstrap. Consider adding `.github/instructions/` files scoped by glob for Go-specific conventions that should only load when editing `.go` files.

#### Cursor

| Aspect | Detail |
|---|---|
| **Auto-loaded** | `.cursorrules` (legacy, single file) — always injected. `.cursor/rules/*.mdc` files with `alwaysApply: true` — always injected. |
| **Rules directory** | `.cursor/rules/*.mdc` — uses `.mdc` format (markdown with YAML frontmatter). |
| **Scoping** | Four rule types based on frontmatter: **Always** (`alwaysApply: true`), **Auto Attached** (`globs: [...]`), **Agent Requested** (`description: "..."` only — agent sees description, decides to load), **Manual** (no metadata — only on explicit `@` mention). |
| **Loading order** | `.cursorrules` + `.cursor/rules/` coexist (both active). Within rules, type determines loading: always → glob-matched → agent-requested → manual. User-level rules in `~/.cursor/rules/` layer below project rules. |
| **Size limits** | Rule files count against context window. Many "always" rules crowd out working context. |
| **MCP support** | ✅ — configured via `.cursor/mcp.json` or Cursor settings. |

**Kanbanzai setup:** Create a `.cursor/rules/kanbanzai-bootstrap.mdc` with `alwaysApply: true` containing the same bootstrap content as `copilot-instructions.md`. Alternatively, rely on MCP delivery since Cursor supports it. Consider glob-scoped rules for Go conventions: `.cursor/rules/go-style.mdc` with `globs: ["*.go"]`.

#### Windsurf (Codeium)

| Aspect | Detail |
|---|---|
| **Auto-loaded** | `.windsurfrules` (single file) — always injected. `.windsurf/rules/*.md` files with `trigger: always_on`. |
| **Rules directory** | `.windsurf/rules/*.md` — markdown with YAML frontmatter. |
| **Scoping** | Three trigger types: `always_on`, `glob_match` (with `globs` field), `model_decision` (agent reads description, decides to load). |
| **Loading order** | `.windsurfrules` always loaded. Rules directory follows trigger hierarchy. |
| **MCP support** | ✅ — added in 2025. |

**Kanbanzai setup:** Create `.windsurfrules` or `.windsurf/rules/kanbanzai.md` with `trigger: always_on` containing bootstrap content. MCP delivery handles the rest.

#### Cline (VS Code extension)

| Aspect | Detail |
|---|---|
| **Auto-loaded** | `.clinerules` — single file at repo root, always injected. |
| **Rules directory** | `.cline/rules/` — supported, but exact scoping mechanism is less well-documented than Cursor/Windsurf. |
| **Other** | `.clineignore` controls file visibility. "Custom Instructions" field in VS Code settings acts as user-level layer. Memory bank feature (`.cline/memory/`) is separate from static instructions. |
| **MCP support** | ✅ Full — one of the earliest MCP adopters. Configured in `cline_mcp_settings.json`. |

**Kanbanzai setup:** Create `.clinerules` with bootstrap content. Cline's strong MCP support means Kanbanzai's MCP delivery works well here.

#### Aider

| Aspect | Detail |
|---|---|
| **Auto-loaded** | `CONVENTIONS.md` at repo root — read into context if present. Note: this is loaded as file content, not injected as system prompt, which is a subtle but meaningful difference. |
| **Configuration** | `.aider.conf.yml` — structured YAML for aider settings. `--system-prompt-extra` flag injects additional system prompt text. `--read <file>` adds read-only context. |
| **MCP support** | ❌ No native MCP support. Aider is a traditional CLI tool using direct LLM API calls. |

**Kanbanzai setup:** This is the main gap. Without MCP, Kanbanzai must deliver instructions via `CONVENTIONS.md` or `--read` flags. Consider generating a `CONVENTIONS.md` that contains condensed Kanbanzai conventions for Aider users. Alternatively, Aider users interact with the Kanbanzai system through its CLI (`kanbanzai`) rather than MCP tools.

#### Amazon Q Developer

| Aspect | Detail |
|---|---|
| **Auto-loaded** | Not well-documented as of early 2026. There may be a `.amazonq/rules/` convention but specifics are uncertain. |
| **MCP support** | Uncertain — integrates with AWS services but MCP support status is unclear. |

**Kanbanzai setup:** Verify current documentation. If MCP is supported, the standard MCP configuration applies. Otherwise, rely on `AGENTS.md` if Q Developer reads it.

#### Google Gemini Code Assist / Jules

| Aspect | Detail |
|---|---|
| **Auto-loaded** | Jules reads `AGENTS.md` at repo root — this was promoted by Google as a cross-platform standard. Gemini Code Assist likely supports `.gemini/` directory and/or `GEMINI.md` but exact paths are evolving. |
| **MCP support** | Google has been adding MCP support but current project-level configuration is uncertain. |

**Kanbanzai setup:** `AGENTS.md` already exists and serves as the entry point for Jules. Ensure `AGENTS.md` contains or points to all critical bootstrap information.

#### JetBrains AI Assistant

| Aspect | Detail |
|---|---|
| **Auto-loaded** | No well-documented project-level instruction file convention. JetBrains' approach is more IDE-settings-driven than file-driven. |
| **MCP support** | ✅ — added in 2025. Configuration through IDE settings rather than dotfiles. |

**Kanbanzai setup:** MCP is the primary delivery channel. JetBrains users configure the Kanbanzai MCP server through IDE settings.

### 7.2 Cross-Platform Summary

| Platform | Bootstrap File | Rules Directory | Glob Scoping | Agent-Requested Rules | MCP |
|---|---|---|---|---|---|
| **Claude Code** | `CLAUDE.md` | `.claude/commands/` | ❌ (subdirs only) | ❌ | ✅ |
| **GitHub Copilot** | `.github/copilot-instructions.md` | `.github/instructions/` | ✅ (`applyTo`) | ❌ | ✅ |
| **Cursor** | `.cursorrules` / `.cursor/rules/` | `.cursor/rules/*.mdc` | ✅ (`globs`) | ✅ (`description`) | ✅ |
| **Windsurf** | `.windsurfrules` / `.windsurf/rules/` | `.windsurf/rules/` | ✅ (`globs`) | ✅ (`model_decision`) | ✅ |
| **Cline** | `.clinerules` | `.cline/rules/` | Uncertain | Uncertain | ✅ |
| **Aider** | `CONVENTIONS.md` | ❌ | ❌ | ❌ | ❌ |
| **Amazon Q** | Uncertain | Uncertain | Uncertain | Uncertain | Uncertain |
| **Gemini/Jules** | `AGENTS.md` | `.gemini/` (likely) | Uncertain | Uncertain | Likely |
| **JetBrains AI** | ❌ (settings-driven) | ❌ | ❌ | ❌ | ✅ |

### 7.3 Implications for Kanbanzai

**MCP is the universal delivery channel.** Claude Code, Copilot, Cursor, Windsurf, Cline, and JetBrains AI all support MCP. Kanbanzai's design — delivering instructions through MCP tool descriptions and the `handoff`/`next` tools — sidesteps platform fragmentation for the majority of the market. The MCP server works identically across all of them.

**The thin bootstrap file is still needed.** Each platform needs a small static file that tells the agent "Kanbanzai MCP tools exist, here's how to use them." This is the one piece that remains platform-specific. The content is identical; only the filename changes.

**Aider is the gap.** No MCP support means Kanbanzai would need a fundamentally different delivery mechanism for Aider users — static file instructions rather than dynamic tool-based delivery.

**The `AGENTS.md` convention is worth watching.** Promoted by Google/Jules, it could become a cross-platform standard. Kanbanzai already has one.

**Recommended bootstrap file strategy:**

| File | Purpose | Content |
|---|---|---|
| `AGENTS.md` | Cross-platform default (Jules, Gemini, fallback) | Project-specific: repo structure, build commands, decisions, scope guard. Points to stage bindings and skills. |
| `.github/copilot-instructions.md` | GitHub Copilot bootstrap | System-level: roles table, skills table, how to use the system, critical rules. Points to AGENTS.md. |
| `CLAUDE.md` | Claude Code bootstrap (not yet created) | Condensed: roles, skills, stage bindings, critical rules. Points to AGENTS.md. Keep short — loaded every turn. |

For Cursor, Windsurf, and Cline, users can either create platform-specific files (`cursorrules`, etc.) with the same bootstrap content, or rely entirely on MCP delivery if the platform's MCP support is configured. Kanbanzai documentation should provide a template for each.

---

## 8. Consolidated Recommendations

Ordered by priority. Each recommendation includes the specific action, the research principle it addresses, and the files affected.

### Priority 1: Migrate and Retire Overlapping Skills

| # | Action | Files | Research Basis |
|---|---|---|---|
| **R1** | Migrate per-dimension evaluation questions, edge case playbooks, remediation phase, review document creation, and checkpoint integration from `kanbanzai-code-review` into `review-code` and `orchestrate-review`. Then retire `kanbanzai-code-review`. | `.agents/skills/kanbanzai-code-review/`, `.kbz/skills/review-code/`, `.kbz/skills/orchestrate-review/` | Skills §8 "Dual-System Confusion"; removes 708-line file exceeding 500-line budget |
| **R2** | Migrate criterion-by-criterion conformance, cross-cutting checks, retrospective step, and document registration from `kanbanzai-plan-review` into `review-plan`. Then retire `kanbanzai-plan-review`. | `.agents/skills/kanbanzai-plan-review/`, `.kbz/skills/review-plan/` | Skills §8; fixes procedural gaps in new skill |
| **R3** | Migrate "Design with Ambition", risk escalation protocol, human/agent role contract, and six-quality lens from `kanbanzai-design` (and its `references/design-quality.md`) into `write-design`. Then retire `kanbanzai-design`. | `.agents/skills/kanbanzai-design/`, `.kbz/skills/write-design/` | Skills §8; preserves behavioral/philosophical content |

### Priority 2: Upgrade Retained System Skills

| # | Action | Files | Research Basis |
|---|---|---|---|
| **R4** | Add vocabulary sections (5–15 terms) to the 5 retained system skills. | `kanbanzai-getting-started`, `kanbanzai-workflow`, `kanbanzai-agents`, `kanbanzai-documents`, `kanbanzai-planning` | P6, Theme 1 "Vocabulary Routing Is the Primary Quality Lever" |
| **R5** | Convert prose anti-patterns to structured Detect/BECAUSE/Resolve format in the 5 retained system skills. | Same as R4 | P5, "Always/Never BECAUSE" format |
| **R6** | Add evaluation criteria and "Questions This Skill Answers" retrieval anchors to the 5 retained system skills. | Same as R4 | Skills §3.9, §3.11; consistent quality across both layers |

### Priority 3: Fix Store Discipline and AGENTS.md

| # | Action | Files | Research Basis |
|---|---|---|---|
| **R7** | Add explicit `.kbz/` store-commit discipline to `kanbanzai-agents` (commit discipline section), `kanbanzai-getting-started` (pre-task checklist), `AGENTS.md` (Git Discipline section), and `base.yaml` (new "Store Neglect" anti-pattern). See §3.7 for exact wording. | `kanbanzai-agents`, `kanbanzai-getting-started`, `AGENTS.md`, `.kbz/roles/base.yaml` | P7 "Observability"; MAST FM-2.2, FM-2.5; P1 "Hardening" |
| **R8** | Deduplicate the pre-task checklist. Keep project-specific items in AGENTS.md (git status, correct branch, check decision logs). Remove "read AGENTS.md" and "read design docs" — those are in `kanbanzai-getting-started`. | `AGENTS.md` | P2 "highest-attention position must be unique" |
| **R9** | Replace the inline Document Reading Order with a pointer to `refs/document-map.md`. | `AGENTS.md` | P3 "stale documentation is poisoned context" |
| **R10** | Remove phase labels from Repository Structure annotations. Replace "(Phase 3)" with nothing — the functional description already says what each package does. | `AGENTS.md` | Skills §3.7 "Avoid time-sensitive information" |
| **R11** | Add a 5-term mini-vocabulary at the top of AGENTS.md: *stage binding*, *role*, *skill*, *lifecycle gate*, *context packet*. One line each. | `AGENTS.md` | P6 "vocabulary routing"; primes comprehension before skills are loaded |
| **R12** | Add a pointer to `.kbz/skills/` and `.kbz/stage-bindings.yaml`. Currently AGENTS.md mentions `.agents/skills/` but not the task-execution skill system. | `AGENTS.md` | Discovery gap identified in §6.2 |
| **R13** | Restructure Decision-Making Rules to be less phase-dependent. Replace "check 4 specific log files by phase number" with guidance to use the knowledge tool or consult `refs/document-map.md`. | `AGENTS.md` | Skills §3.7 "no time-sensitive information" |

### Priority 4: Fix Discovery and Stale Pointers

| # | Action | Files | Research Basis |
|---|---|---|---|
| **R14** | Update `refs/document-map.md` to point to `.kbz/skills/review-code/` and `.kbz/skills/review-plan/` instead of the old `.agents/skills/` versions (after R1 and R2 are complete). | `refs/document-map.md` | P3 "stale documentation" |
| **R15** | Add explicit `.kbz/skills/` and stage bindings discovery to `kanbanzai-getting-started`. Currently it says "check `kanbanzai-workflow`" but never mentions the role+skill system. | `.agents/skills/kanbanzai-getting-started/SKILL.md` | Discovery gap identified in §6.2 |
| **R16** | Create a `CLAUDE.md` bootstrap file for Claude Code users. Content mirrors `copilot-instructions.md` but kept shorter (loaded every turn). | `CLAUDE.md` (new file) | Cross-platform discovery (§7) |

### Priority 5: Build Evaluation Infrastructure

| # | Action | Files | Research Basis |
|---|---|---|---|
| **R17** | Build a minimal evaluation harness: 2–3 test scenarios per skill, before/after comparison using each skill's Evaluation Criteria section. | New evaluation infrastructure | Skills §3.9 "Evaluation Must Precede Documentation" |
| **R18** | Verify that `internal/context/assemble.go` orders context packets following the attention curve (identity/constraints first, supporting material middle, instructions/anchors last). Fix if not. | `internal/context/assemble.go` | P2 "Position and Structure Matter" |
| **R19** | Verify that `handoff` includes effort budgets from stage bindings in assembled context packets. Add if not. | `internal/context/` | P9 "Token Economy" |

---

## Appendix A: File Inventory

### Roles (`.kbz/roles/`)

| File | Identity | Vocabulary Terms | Anti-Patterns | Inherits | Status |
|---|---|---|---|---|---|
| `base.yaml` | Software development agent | 5 conventions | 2 | — | ✅ Good |
| `architect.yaml` | Senior software architect | 12 | 4 | base | ✅ Good |
| `spec-author.yaml` | Senior requirements engineer | 9 | 5 | base | ✅ Good |
| `orchestrator.yaml` | Senior engineering manager | 15 + 12 constraints | 7 | base | ✅ Excellent |
| `implementer.yaml` | (base implementer) | — | — | base | Not reviewed |
| `implementer-go.yaml` | Senior Go engineer | 12 | 6 | implementer | ✅ Excellent |
| `reviewer.yaml` | Senior code reviewer | 6 | 4 | base | ✅ Good |
| `reviewer-conformance.yaml` | Senior requirements verification engineer | 6 | 3 | reviewer | ✅ Good |
| `reviewer-quality.yaml` | Senior software quality engineer | 9 | 3 | reviewer | ✅ Good |
| `reviewer-security.yaml` | Senior application security engineer | 15 | 5 | reviewer | ✅ Excellent |
| `reviewer-testing.yaml` | Senior test engineer | 10 | 5 | reviewer | ✅ Good |
| `researcher.yaml` | — | — | — | base | Not reviewed |
| `documenter.yaml` | — | — | — | base | Not reviewed |

### Task-Execution Skills (`.kbz/skills/`)

| Skill | Lines | Stage | Constraint Level | Has Examples | Has Eval Criteria | Status |
|---|---|---|---|---|---|---|
| `write-design` | 252 | designing | high | ✅ | ✅ | ✅ Good |
| `write-spec` | 331 | specifying | high | ✅ (3) | ✅ | ✅ Excellent |
| `write-dev-plan` | 331 | dev-planning | medium | ✅ | ✅ | ✅ Good |
| `decompose-feature` | 259 | dev-planning | medium | ✅ | ✅ | ✅ Good |
| `implement-task` | 202 | developing | medium | ✅ (3) | ✅ | ✅ Excellent |
| `orchestrate-development` | 256 | developing | medium | ✅ (3) | ✅ | ✅ Good |
| `review-code` | 335 | reviewing | medium | ✅ (3) | ✅ | ✅ Excellent |
| `orchestrate-review` | 375 | reviewing | medium | ✅ (2) | ✅ | ✅ Excellent |
| `review-plan` | 310 | plan-reviewing | medium | ✅ | ✅ | ✅ Good |
| `write-research` | 300 | researching | high | ✅ | ✅ | ✅ Good |
| `update-docs` | 217 | documenting | medium | ✅ | ✅ | ✅ Good |

### System Skills (`.agents/skills/`)

| Skill | Lines | Has Vocabulary | Has Structured Anti-Patterns | Has Eval Criteria | Has Retrieval Anchors | Status |
|---|---|---|---|---|---|---|
| `kanbanzai-getting-started` | 76 | ❌ | ❌ | ❌ | ❌ | ⚠️ Needs upgrade |
| `kanbanzai-workflow` | 221 | ❌ | ❌ Prose | ❌ | ❌ | ⚠️ Needs upgrade |
| `kanbanzai-agents` | 330 | ❌ | ❌ Prose | ❌ | ❌ | ⚠️ Needs upgrade |
| `kanbanzai-documents` | 224 | ❌ | ❌ Prose | ❌ | ❌ | ⚠️ Needs upgrade |
| `kanbanzai-planning` | 222 | ❌ | ❌ Prose | ❌ | ❌ | ⚠️ Needs upgrade |
| `kanbanzai-design` | 252 | ❌ | ❌ Prose | ❌ | ❌ | ⚠️ Retire after R3 |
| `kanbanzai-code-review` | 708 | ❌ | ❌ Prose | ❌ | ❌ | ⚠️ Retire after R1 |
| `kanbanzai-plan-review` | 258 | ❌ | ❌ Prose | ❌ | ❌ | ⚠️ Retire after R2 |

### Other Instruction Files

| File | Lines | Purpose | Status |
|---|---|---|---|
| `AGENTS.md` | 194 | Project-specific development instructions | ⚠️ Needs tightening (§5) |
| `.github/copilot-instructions.md` | 128 | GitHub Copilot bootstrap → roles, skills, critical rules | ✅ Good |
| `.kbz/stage-bindings.yaml` | ~110 | Stage → role + skill + prerequisites mapping | ✅ Excellent |
| `.kbz/skills/CONVENTIONS.md` | ~120 | Skill authoring standards | ✅ Excellent |
| `refs/document-map.md` | ~40 | Topic → document routing table | ⚠️ Has stale pointers |
| `refs/sub-agents.md` | ~40 | Sub-agent context propagation template | ✅ Good |
| `refs/go-style.md` | — | Go conventions reference | Not reviewed |
| `refs/testing.md` | — | Test conventions reference | Not reviewed |
| `refs/knowledge-graph.md` | — | Graph tool reference | Not reviewed |

---

## Appendix B: Migration Checklists

### B.1 `kanbanzai-code-review` → `review-code` + `orchestrate-review`

Migration checklist — complete every item before retiring the old skill:

- [ ] **Per-dimension evaluation questions** (old L100–239) — verify these exist in `reviewer-*.yaml` role files. If any dimension lacks evaluation questions, port them from the old skill.
- [ ] **Edge case: Missing Spec** (old L346–365) — add to `review-code` as a new subsection in the Procedure, expanding the current terse STOP instruction.
- [ ] **Edge case: Partial Implementation** (old L365–380) — add to `review-code` with the nuanced handling (set spec_conformance to `concern`, continue other dimensions).
- [ ] **Edge case: Ambiguous Conformance** (old L380–395) — add to `review-code` with the classification guidance (non-blocking if implementation appears intentionally better than spec).
- [ ] **Edge case: Missing Context** (old L395–410) — add to `review-code` with the per-dimension impact assessment.
- [ ] **Remediation phase** (old L583–620) — add to `orchestrate-review` as Steps 7–10 covering task creation, conflict-check, re-review scoping, and escalation cycle.
- [ ] **Write review document** (old L538–561) — add to `orchestrate-review` between current Step 5 and Step 6, including the naming convention (`review-{id}-{slug}.md`) and `doc(action: register)`.
- [ ] **Human checkpoint integration** (old L674–701) — add to `orchestrate-review` covering the 3 trigger scenarios (ambiguous findings, high-stakes features, dimension disagreement).
- [ ] **Context budget strategy** (old L620–657) — add as a reference file or note in `orchestrate-review`.
- [ ] **Tool chain reference** (old L657–674) — add as a reference file if useful.
- [ ] **Update `refs/document-map.md`** — change code review pointer from `kanbanzai-code-review` to `review-code` / `orchestrate-review`.
- [ ] **Delete `.agents/skills/kanbanzai-code-review/`**.

### B.2 `kanbanzai-plan-review` → `review-plan`

- [ ] **Criterion-by-criterion spec conformance** (old Step 3, L71–82) — add as a new step in `review-plan` between current Step 2 (feature terminal-state) and Step 3 (spec approval). Must read acceptance criteria and verify against implementation code, not just check approval status.
- [ ] **Cross-cutting checks** (old Step 5, L100–108) — add as a new step: `go test -race ./...`, `health()`, `git status`.
- [ ] **Retrospective contribution** (old Step 6, L110–122) — add as final procedure step.
- [ ] **Document registration** (old Step 7, L124–131) — add step to write findings to `work/reviews/` and register with `doc(action: register)`.
- [ ] **Spec Conformance Detail table** (old L152–173) — add per-feature, per-criterion table to output format.
- [ ] **Inputs section** (old L42–49) — add prerequisites list to `review-plan`.
- [ ] **Update `refs/document-map.md`** — change plan review pointer from `kanbanzai-plan-review` to `review-plan`.
- [ ] **Delete `.agents/skills/kanbanzai-plan-review/`**.

### B.3 `kanbanzai-design` → `write-design`

- [ ] **"Design with Ambition"** (old L36–43) — add as a stance preamble section in `write-design`, before the Procedure.
- [ ] **Surfacing Risk** (old L134–147) — add as a new section or procedure step with the 3-tier escalation model (minor: mention once → significant: raise clearly → security/data-integrity: stop).
- [ ] **Human/Agent role contract** (old L23–32) — add to `write-design` preamble: "Human = Design Manager, Agent = Senior Designer."
- [ ] **Six-quality evaluation lens** (old `references/design-quality.md`) — port to `.kbz/skills/write-design/references/design-quality.md`.
- [ ] **Iterative process framing** (old L47–58) — add note that design is iterative and messy, that's normal.
- [ ] **Draft lifecycle** (old L62–73) — add explanation of what draft status means and how to maintain drafts.
- [ ] **Design splitting guidance** (old L151–170) — expand `write-design` Step 1.4 with signs that a design needs splitting and the supersession protocol.
- [ ] **Gotchas** (old L185–200) — add to `write-design`: registration, content hash drift, `doc refresh`, editing approved docs.
- [ ] **Next Steps After Design** (old L204–208) — add handoff-to-specification guidance.
- [ ] **Delete `.agents/skills/kanbanzai-design/`** (and its `references/` directory).

### B.4 AGENTS.md Fixes

- [ ] **Add mini-vocabulary** (R10) — 5 terms at the top: stage binding, role, skill, lifecycle gate, context packet.
- [ ] **Add pointer to `.kbz/skills/` and stage bindings** (R11) — brief note in Self-Managed Development or a new section.
- [ ] **Deduplicate pre-task checklist** (R7) — keep git status/branch check; remove "read AGENTS.md" and "read design docs" (covered by `kanbanzai-getting-started`).
- [ ] **Replace Document Reading Order** (R8) — replace inline list with pointer to `refs/document-map.md`.
- [ ] **Remove phase labels** (R9) — strip "(Phase 2b)", "(Phase 3)", etc. from Repository Structure.
- [ ] **Restructure Decision-Making Rules** (R12) — replace phase-numbered log references with guidance to consult `refs/document-map.md` or the knowledge tool.
- [ ] **Update `refs/document-map.md`** (R13) — fix stale skill pointers after migrations complete.
- [ ] **Fix `kanbanzai-getting-started` discovery** (R15) — add mention of stage bindings and `.kbz/skills/`.
- [ ] **Create `CLAUDE.md`** (R16) — bootstrap file for Claude Code users.

### B.5 Store Discipline Fixes

- [ ] **`kanbanzai-agents` commit discipline** (R7) — add to the Commit Discipline section: "Every `.kbz/state/` change is a code change. When you register a document, transition an entity, contribute knowledge, or perform any action that modifies `.kbz/`, commit the change immediately or include it in your next logical commit. Do not leave `.kbz/` files uncommitted at the end of a task."
- [ ] **`kanbanzai-getting-started` pre-task checklist** (R7) — add item: "If uncommitted `.kbz/` files exist from a previous session, commit them now. Do not stash, discard, or ignore them — they are workflow state, not temporary files."
- [ ] **`AGENTS.md` Git Discipline section** (R7) — add: "`.kbz/state/` files are versioned project state, not ephemeral cache. Treat every store change as a code change. Never stash, discard, or `.gitignore` these files."
- [ ] **`base.yaml` role anti-pattern** (R7) — add "Store Neglect" anti-pattern with Detect/BECAUSE/Resolve: detect uncommitted or discarded `.kbz/state/` files; BECAUSE the store is the source of truth for parallel agents and uncommitted state causes drift, race conditions, and silent data loss; resolve by committing `.kbz/` changes alongside code changes and committing orphaned `.kbz/` files at task start.