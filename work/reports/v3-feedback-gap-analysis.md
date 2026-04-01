---
Title: V3.0 Feedback and Gap Analysis Report
Status: Final
Date: 2025-07-18
Informed by:
  - work/design/kanbanzai-3.0-workflow-and-tooling-v2.md
  - work/design/skills-system-redesign-v2.md
  - work/research/agent-orchestration-research.md
  - work/research/agent-skills-research.md
Purpose: Synthesise session feedback and retrospective signals against the V3.0 design plans to identify gaps, blockers, and priorities for the next round of design and planning.
---

# V3.0 Feedback and Gap Analysis Report

## 1. Introduction

This report synthesises feedback from five working sessions and one project retrospective, comparing observed friction points against the planned V3.0 design (the workflow-and-tooling doc and the skills-system-redesign doc). The goal is to identify what V3.0 already addresses, what it partially covers, what it misses entirely, and — critically — what infrastructure prerequisites must be resolved before the V3.0 design can work as intended.

The feedback sources are:

- **Five session retrospectives** covering: cross-document alignment review, document store housekeeping, research-to-design integration, research-driven design updates, entity-names feature implementation + review, and entity-names remediation.
- **One project retrospective** synthesising knowledge entries and retro signals from recent phases (P6–P12).

---

## 2. Session Feedback Summary

### 2.1 Consistently Strong: The Document-Led Workflow

Every session praised the same core strength: **the document chain works**. Traceability from research → design → spec → implementation → review means agents can verify claims against sources, trace decisions to their origins, and avoid drift. Specific highlights:

- Front matter conventions (status, "Informed by", "Supersedes") made document relationships navigable without guessing.
- The design decisions table (e.g., DD-15 through DD-20) gave every change a named, numbered, rationale-backed home.
- The alignment report format — structured overlaps, severity ratings, canonical ownership — made "apply these changes" mechanical rather than interpretive.
- The document store (`doc list`, batch registration) made inventory management straightforward at scale (133+ documents).

### 2.2 Recurring Friction: Document Size

Mentioned in **four of five sessions**. The skills redesign at ~1,400 lines was the most cited example, but the pattern is general:

- Agents can't hold entire documents in one read, leading to constant re-reading to verify cross-references.
- Cross-referencing between two large documents simultaneously pushes context limits.
- By the third round of edits, navigation overhead dominates thinking time.
- Suggested fixes: progressive disclosure for design documents (not just skills), splitting monolithic docs into smaller ones with their own lifecycles, routing documents with reference files.

### 2.3 Recurring Friction: Cross-Reference Integrity

Mentioned in **three sessions**:

- No automated way to verify `§X.Y` and `(skills doc §Z)` references point to real headings.
- Coordinated edits across two documents simultaneously are error-prone.
- No persistent way to register section-to-section relationships between documents (the knowledge system tracks facts/decisions, not document-section mappings).
- Suggested fix: a lightweight cross-reference validation tool; a "design review" mode in `doc_intel` that surfaces overlapping concepts across documents.

### 2.4 Recurring Friction: Gap Between Disk and Store

Highlighted in **two sessions** focused on document management:

- No "diff" between files on disk and documents in the store — the comparison was manual across 133+ entries.
- No way to distinguish "intentionally unregistered" from "accidentally missed."
- `doc import` exists but wasn't trusted (unclear behaviour, no dry-run).
- 22 documents accumulated unregistered during normal work, suggesting auto-registration should happen at creation time.
- Suggested fixes: `doc audit`/`doc unregistered` command, `--dry-run` for import, `.kbz/docignore`, auto-registration on creation.

### 2.5 Recurring Friction: Planning Precision

Mentioned in **two sessions** about implementation work:

- Plans didn't enumerate cascading consequences — making `Name` required broke ~125 test call sites across 18 files, but the plan only mentioned 2.
- The service layer between model and MCP wasn't called out, leading to mid-task discovery and context exhaustion.
- Two `Config` structs in different packages weren't both named, causing the most significant defect.
- Remediation docs contained small factual errors (e.g., "already imported" when it wasn't) that would trip up a literal-minded agent.
- Scope guards excluded test files by default, requiring manual follow-up.
- Suggested fixes: enumerate all affected call sites, default scope to include test files, validate doc claims against code before writing.

### 2.6 Recurring Friction: Context Budget and Review Scope

Mentioned in **three sessions**:

- Sub-agents ran out of context mid-work (Task 5 timeout, partial review).
- Reviewing 38 ACs plus quality/coverage/documentation in one pass is too much for a single agent run.
- Creating a new document while referencing three others burns context.
- No way to track "I've verified this claim" persistently across a session.
- Suggested fixes: split reviews into conformance + quality passes, scope budgets for reviews, structured claim-tracking.

### 2.7 Recurring Friction: Missing Workflow Stages

Mentioned in **two sessions**:

- No formal "alignment review" stage for cross-document verification before specification.
- The research-to-design integration pipeline (read research → synthesise → apply → cross-check) was effective but improvised — it should be a documented procedure or skill.
- The review-then-fix cycle could be tighter (presenting gaps and proposed fixes together rather than analysis-first).

### 2.8 One-Off but Notable

- **Batch test false-positives**: MCP batch handler counts `(toolResultError, nil)` as success, so validation failures look like successes in batch summaries — a systemic issue.
- **AC misalignment with reality**: spec assumed gradual migration, implementation did big-bang — spec should acknowledge both approaches.
- **The three-document chain** (research → recommendations → design) created useful intermediate synthesis but also introduced a translation layer where findings could be lost.

---

## 3. Retrospective Signals Summary

### 3.1 Critical: `decompose` + `docint` AC Pattern Recognition

`docint` doesn't recognise the `**AC-NN.**` acceptance criteria pattern, so `decompose propose` finds nothing to work with, falls back to section headers, and produces dangerously plausible garbage like "Implement 1. Purpose." The warning is buried and easy to miss. Index staleness compounds the problem — specs registered in the same session may not be indexed yet.

### 3.2 Important: Sub-Agent State Isolation

A sub-agent ran `git stash`, which captured uncommitted `.kbz/state/` changes from MCP tool calls. Feature statuses reverted, specs went from approved to draft, tasks from ready to queued. State was permanently lost. The recommended mitigations are: (1) commit `.kbz/state/` before spawning sub-agents, (2) prevent sub-agents from running git ops that affect the working tree outside their scope, or (3) use worktrees for sub-agent isolation.

### 3.3 Important: Agents Bypassing MCP Tools

Agents default to `cat`, `ls`, `grep` on `.kbz/state/` YAML files instead of using `status`, `entity get`, `doc list`. They also write reports from in-session memory without consulting `knowledge list` or `retro synthesise`.

### 3.4 Important: `doc approve` Doesn't Sync File Headers

`doc approve` updates the store record but leaves the file's own `Status: Draft` header unchanged. Agents and humans reading the file see "Draft" when the system says "Approved."

### 3.5 Patterns Working Well (Preserve and Codify)

- **`advance: true` multi-stage transitions** with automatic gate checks validates the enforceable gates design.
- **File ownership tables in dev plans** produced zero merge conflicts during parallel work.
- **Cross-plan features as review targets** — using real work as test cases caught genuine issues (fabricated lifecycles, stale tool names).

---

## 4. Gap Analysis: Feedback vs V3.0 Plans

### 4.1 Well-Addressed by V3.0

#### Context Budget / Review Scope Exhaustion

The feedback repeatedly flagged agents running out of context during reviews and large cross-document work. The V3.0 plans attack this from multiple angles:

- **Token budget management** (skills doc §6.2) — a layered system capping context at 40% of the window, with load-shedding at 60%.
- **Effort budgets** (workflow doc §8) — calibrating agent effort per stage, positioned in the high-attention zone.
- **Review scope** — the feedback specifically said "the review needs a scope budget." The review-rework formalisation (workflow doc §4.4) directly addresses this with focused re-reviews for cycle ≥ 2, including only rework tasks and prior findings rather than the full implementation.
- **Filesystem-output convention** (workflow doc §13) — sub-agents write to documents/tasks, orchestrators read references not contents.

The one gap is the feedback's suggestion to split reviews into conformance + quality passes — which is actually modelled in the skills doc (§5.3–5.4, skill composition during review) but isn't explicitly framed as a context-budget strategy.

#### Agents Skipping Steps / Missing Workflow Stages

The V3.0 plans are aggressive here:

- **Mandatory stage gates on all transitions** (workflow doc §3.2) — the single biggest change.
- **Lifecycle state validation in `handoff`/`next`** (workflow doc §7.2) — structurally impossible to assemble wrong-stage context.
- **Orchestration pattern signalling** (workflow doc §7.4) — explicit "single-agent, do not delegate" for sequential stages.
- **Binding registry** adds `researching` and `documenting` stages (skills doc §3.3) that didn't previously exist.

However, the feedback's specific suggestion of an "alignment review" stage for cross-document verification before specification — the pattern of reading two designs, checking them against research, finding deviations — isn't modelled. The `reviewing` stage applies to code/implementation review, not to design-against-research verification.

#### Review-Rework Loop Structure

Directly addressed:

- **Review cycle counter** (workflow doc §4.2).
- **Iteration cap at 3** with human escalation (workflow doc §4.3).
- **Focused re-review** for cycle ≥ 2 (workflow doc §4.4).

#### Agents Bypassing MCP Tools

Directly in scope:

- **ACI tool description audit** (workflow doc §5) — rewriting descriptions with "when to use" and negative guidance.
- **Agent-driven testing** (workflow doc §5.6) — observing where agents pick wrong tools and rewriting descriptions.
- **Actionable error messages** (workflow doc §6) — making the right path clearer when things fail.
- **Role vocabulary** (skills doc §3.1) — domain vocabulary routes agents toward the right tools.

The retro's specific suggestion to add explicit negative guidance like "Do NOT read `.kbz/` files directly" maps exactly to the ACI principles (workflow doc §5.2, point 2). The additional suggestion to add an "In-Session Memory Only" anti-pattern to report-writing skills is a content addition that fits naturally into the skill catalog.

### 4.2 Partially Addressed by V3.0

#### Document Size / Progressive Disclosure

This was the most frequently raised friction point — agents can't hold 1,400-line documents in one read, leading to constant re-reading. The V3.0 plans touch this indirectly:

- **Token budget management** (skills doc §6.2) caps context assembly and uses layered loading.
- **Stage-specific assembly** (workflow doc §7.3) excludes irrelevant context per stage.
- **DP-6** ("n=5 beats n=19") applies the lean principle to skills and templates.

But the feedback wasn't about skills or assembled context — it was about *design documents themselves* being too large to work with during the design phase. Multiple sessions suggested splitting monolithic design docs into smaller files, progressive disclosure for design documents (routing doc + reference files), and a "changes log" section on living documents. None of these are in the V3.0 plans. The document template system (workflow doc §10) defines structure for *new* documents but doesn't address the size problem for existing large documents.

**Gap: Addressed for context assembly; not addressed for design-phase document authoring.**

#### Cross-Reference Integrity

The V3.0 plans partially address this:

- **Cross-reference requirements** in document templates (skills doc §5.1) — "a specification must reference its parent design document."
- **Automated structural checks at stage gates** (workflow doc §10.4) — can check that cross-references are valid.
- **Validation scripts** per authoring skill (skills doc DD-14).

But the feedback was about *intra-document* section references (`§3.2`, `(skills doc §5)`) and *cross-document section-to-section* mappings — not just document-to-document references. The structural checks verify "spec references design doc" but not "§3.2 of this document correctly points to a heading that exists in the other document." The feedback also asked for a `doc_intel` "design review" mode that surfaces overlapping concepts between two documents — not planned.

**Gap: Document-level cross-references are covered; section-level cross-reference validation is not.**

#### Planning Precision / Cascading Consequences

The V3.0 plans address the decomposition side:

- **Decomposition quality validation** (workflow doc §11) — checks for descriptions, dependencies, sizing, test coverage, orphan detection.
- **Document templates for dev-plans** (skills doc §5.1) — require Task Breakdown, Dependency Graph, and Risk Assessment.
- **Gate-checkable templates** (workflow doc §10.4) — structural checks at stage transitions.

The feedback's core complaint — that plans didn't enumerate *all affected code locations* and that remediation docs had factual errors against the actual code — isn't directly solved. That's about *content quality* rather than *structure*, which falls into LLM-as-judge territory (workflow doc §10.5, flagged as medium-term).

**Gap: Structure is covered; content accuracy is medium-term.**

#### Research Integrity in Reviews

Two sessions flagged that there's no way to verify whether a design document's claims are faithful to the research it cites. The V3.0 plans have:

- **LLM-as-judge quality evaluation** (workflow doc §10.5) with a "factual accuracy" dimension.
- **Research traceability table** (workflow doc §17) mapping recommendations to sources.

But the LLM-as-judge is medium-term, and the traceability table is a static artifact. The feedback wanted a "research integrity" dimension in review skills — checking document-against-canonical-sources, not just document-against-document.

**Gap: Partially addressed by LLM-as-judge (medium-term). No near-term mechanism for research integrity checking.**

#### File Ownership in Dev-Plan Templates

The retro noted that file ownership tables in dev plans produced zero merge conflicts during parallel work. The dev-plan template's required sections (skills doc §5.1) are Scope, Task Breakdown, Dependency Graph, Risk Assessment, and Verification Approach. File ownership could fit under Task Breakdown but isn't explicit.

**Gap: Working pattern not codified in the template.**

### 4.3 Not Addressed by V3.0

#### `docint` AC Pattern Recognition — Infrastructure Blocker

`docint` doesn't recognise the `**AC-NN.**` acceptance criteria format used in specs. The entire V3.0 template → gate check → decomposition pipeline (workflow doc §10.4, §11, skills doc DD-19) assumes `docint` can parse spec structure. If it can't, the pipeline is built on a broken foundation.

**This is a prerequisite blocker for V3.0.** It needs fixing *before* V3.0 implementation, not as part of it.

#### Sub-Agent State Isolation

A sub-agent running `git stash` captured uncommitted `.kbz/state/` changes and permanently destroyed workflow state. The V3.0 design makes this *worse* — the more the system relies on `orchestrator-workers` topology with mandatory stage gates, the more catastrophic accidental state reversion becomes. The workflow doc §13 (Filesystem-Output Convention) addresses output routing but not state isolation. Neither document addresses what happens when sub-agents run git operations that affect `.kbz/state/`.

**Not addressed, and the V3.0 design increases the blast radius.**

#### `doc approve` File Header Sync

`doc approve` updates the store record but leaves the file's `Status: Draft` header unchanged. The V3.0 plans heavily depend on document approval status for stage gates, binding prerequisites, and structural checks. Combined with the tool-bypass pattern (agents reading files directly), this creates a real source of confusion.

**Small tooling fix, not a design concern, but compounds other problems.**

#### Gap Between Disk and Store

22 documents accumulated unregistered during normal work. No `doc audit` command, no `--dry-run` for import, no `.kbz/docignore`, no auto-registration on creation. The V3.0 plans focus on how documents are *used* (templates, gates, assembly) not on the *registration boundary* between files on disk and documents in the store.

**Not in V3.0 scope. Could be a quick-win independent of the 3.0 work.**

#### Persistent Claim-Tracking During Verification

When cross-referencing ~30 claims between documents, agents keep verification state in their head. There's no lightweight mechanism to mark individual claims as checked. The knowledge system tracks facts and decisions, not verification state within a document.

**Not addressed. Could potentially be served by structured cross-reference tracking or `doc_intel` improvements, but nothing is designed.**

#### Batch Test False-Positive

The MCP batch handler counts `(toolResultError, nil)` as success, so validation failures look like batch successes. This is a bug, not a design concern.

**Independent fix needed.**

---

## 5. Consolidated Priority View

The recommendations fall into three categories: **infrastructure and tooling fixes** that belong to neither design document (they fix bugs or add capabilities in the MCP server, `docint`, or `doc` tool), **workflow doc changes** (enforcement mechanisms, implementation sequencing, observability), and **skills doc changes** (skill content, templates, binding registry, anti-patterns). This three-way split is significant — half the near-term items are infrastructure work that must be done independently of the V3.0 design round.

### 5.1 Infrastructure and Tooling (Neither Design Document)

These are fixes and enhancements to the MCP server, `docint`, and `doc` tool. They don't require design document changes — they require code changes. Items marked as **blockers** must ship before V3.0 implementation begins.

| # | Issue | Source | Research Backing | Priority |
|---|-------|--------|------------------|----------|
| 1 | **Fix `docint` AC-NN pattern recognition** | Retro | MetaGPT (structured artifacts must be parseable), Masters et al. (decomposition quality is the critical path) | **Blocker** — the template → gate → decomposition pipeline can't work without it |
| 2 | **Sub-agent state isolation** (commit `.kbz/state/` before sub-agent dispatch) | Retro | Microsoft (mutable shared state anti-pattern), Google (error amplification in independent agents) | **Blocker** — `orchestrator-workers` topology will cause catastrophic state loss. Fix at the tool level (`handoff`/dispatch path), not just as skill-level guidance |
| 3 | **Batch handler false-positive bug** | Sessions | SWE-agent (poka-yoke — make wrong usage hard, make errors visible) | **Blocker** — silently masks validation failures in batch operations |
| 4 | ~~`doc approve` file header sync~~ | Retro | ~~Poka-yoke — make wrong state hard~~ | **Withdrawn** — superseded by the consistent-front-matter design (`work/design/consistent-front-matter.md`), which removes `Status` from managed documents entirely rather than syncing it. The design, spec, and implementation plan exist but have not yet been executed. |
| 5 | `doc audit` / `doc unregistered` command | Sessions | ➖ Neutral | Quick win — compare store paths against disk paths |
| 6 | `doc import --dry-run` mode | Sessions | ➖ Neutral | Quick win — show what would be registered without committing |
| 7 | Focused cross-document section reads in `doc_intel` | Sessions | Skills research §6.4 ("look for repeated work") — agents repeatedly re-read large docs for cross-referencing; a small extension to retrieve specific sections from multiple documents in one call would reduce the overhead | Medium effort — `doc_intel` enhancement |

### 5.2 Workflow and Tooling Doc Changes

These are changes to `work/design/kanbanzai-3.0-workflow-and-tooling-v2.md` — the document that owns enforcement mechanisms, MCP tool behaviour, implementation sequencing, and observability.

| # | Issue | Source | Research Backing | Where It Fits |
|---|-------|--------|------------------|---------------|
| 8 | **Move evaluation suite from Phase E to Phase A** | Research | Skills research Theme 6 ("evaluation must precede documentation"), Anthropic ("start evaluating immediately with small samples") | Workflow doc §15 (Implementation Priority). Build 15–20 baseline scenarios *before* enabling mandatory gates. Measure each subsequent phase's impact against the baseline. This is a sequencing change, not a scope change. |
| 9 | LLM-as-judge for content accuracy | Sessions | ✅ Supported (Anthropic LLM-as-judge, Microsoft maker-checker) | Already flagged as medium-term in workflow doc §10.5. No change needed — just confirming this is correctly positioned. |
| 10 | Hard tool filtering (dynamic MCP tool scoping) | V3.0 design | ✅ Supported (Google tool-use bottleneck) | Already deferred to post-initial-release in workflow doc §9.3. No change needed. |

The workflow doc's existing content is otherwise well-aligned with the feedback. The mandatory gates (§3), ACI audit (§5), actionable errors (§6), stage-aware assembly (§7), effort budgets (§8), review-rework loops (§4), and decomposition validation (§11) all address the session feedback directly. The only action is the evaluation suite resequencing.

### 5.3 Skills Doc Changes

These are changes to `work/design/skills-system-redesign-v2.md` — the document that owns roles, skills, the binding registry, context assembly, document templates, and skill content guidelines.

| # | Issue | Source | Research Backing | Where It Fits |
|---|-------|--------|------------------|---------------|
| 11 | File ownership as explicit dev-plan template section | Retro | ✅ Supported — structured intermediate artifacts (MetaGPT). This working pattern produced zero merge conflicts during parallel work. | §5.1, dev-plan template required sections. Add "File Ownership" alongside Scope, Task Breakdown, Dependency Graph, Risk Assessment, Verification Approach. |
| 12 | Cross-document alignment **skill** (not a new lifecycle stage) | Sessions | ✅ As skill. ⚠️ Adding a new stage adds lifecycle complexity the research counsels against (Anthropic: "use the lowest level of complexity"). The existing `researching` stage with the right skill handles this. | §5.1 skill catalog, bound to the `researching` stage in the binding registry (§3.3). |
| 13 | "In-Session Memory Only" anti-pattern for report-writing skills | Retro | ✅ Supported — retrieval for contextual consistency (skills research §4). Agents write reports from session memory without consulting `knowledge list` or `retro synthesise`. | §5.1, add to `write-research` and report-writing skills. Detect: writing a report without calling knowledge/retro tools first. |
| 14 | "State Destruction via Git Operations" anti-pattern for orchestration skills | Retro | ✅ Supported — Microsoft mutable state anti-pattern. Note: the tool-level fix (#2 above) is the primary mitigation; this anti-pattern is defence-in-depth. | §5.2, add to `orchestrate-development` and `orchestrate-review` skills. Detect: sub-agent runs `git stash`, `git checkout`, or `git reset` affecting `.kbz/state/`. |
| 15 | Research integrity dimension in review skills | Sessions | ➖ Neutral — reasonable but not research-driven. Address as skill content, not a system feature. | §5.3, review skill catalog. Add "do my resolutions deviate from the canonical sources?" as a standard review criterion. |
| 16 | Design document progressive disclosure convention | Sessions | ✅ Supported — skills research §3.5 (progressive disclosure patterns), Theme 1 (context window is a shared resource). Apply the same routing-doc + reference-files pattern to design docs. | §8 (Skill Content Authoring Guidelines) or a new §8.6. The convention: routing document (~300–500 lines) with one-level-deep reference files for detailed sections. Applies to design documents, not just skills. |

### 5.4 Longer-Term / Deferred

These items are either not yet justified by the research, risk working against it, or should wait until simpler interventions have been tried first.

| # | Issue | Source | Research Alignment | Notes |
|---|-------|--------|-------------------|-------|
| 17 | Persistent claim-tracking during verification | Sessions | ⚠️ Caution — skills research supports session-scoped checklists (§3.6), not a new persistence mechanism. Try a copy-paste checklist in the alignment skill first. | Only build if the skill-based approach (#12) demonstrably fails |
| 18 | Section-level cross-reference validation tool | Sessions | ⚠️ Caution — fails the novelty test (skills research §3.1: "Does Claude really need this?"). An agent with clear instructions can verify cross-references manually. | Try skill-based approach first per §7.1 |
| 19 | `doc_intel` design review mode (overlapping concept detection) | Sessions | ⚠️ Caution — may encourage parallelizing sequential reasoning work. Google's sequential penalty (39–70% degradation) means design-phase cross-referencing should stay single-agent. | Evaluate whether focused section reads (#7) suffice |

---

## 6. Key Observations

### The V3.0 design is strongest where the feedback is about agent execution behaviour.

Skipping steps, context overload, review loops, tool selection — these are precisely what the V3.0 plans were designed to address, informed by the orchestration research. The mandatory gates, stage-aware context assembly, ACI tool descriptions, and review-rework formalisation form a coherent response to these problems.

### The V3.0 design is weakest where the feedback is about the design-phase experience.

Working with large documents, verifying cross-references between design docs, managing document registration, tracking claim verification state — these are problems that occur *before* agents are orchestrated through the V3.0 pipeline. The V3.0 plans assume well-structured documents already exist and focus on how to use them; the feedback says getting to well-structured documents is itself a significant challenge.

### The retro signals identify infrastructure prerequisites, not design gaps.

The session feedback identifies design-level gaps (missing stages, missing tools). The retro signals identify something different: infrastructure that must work correctly for the design to function at all. The `docint` parser, the state isolation problem, and the batch handler bug are not design concerns — they're implementation prerequisites. If unresolved, they'll cause the V3.0 design to fail in practice regardless of how well the design itself is constructed.

### The "working well" patterns should be explicitly codified.

File ownership tables, `advance: true` multi-stage transitions, and cross-plan features as test cases all emerged organically and proved valuable. The V3.0 design should capture these as required outputs or recommended practices rather than leaving them as tribal knowledge.

---

## 7. Research Alignment Analysis

The V3.0 design was informed by two research reports: the agent orchestration research (`work/research/agent-orchestration-research.md`) and the agent skills research (`work/research/agent-skills-research.md`). This section checks the feedback-driven suggestions against those research findings to ensure we don't introduce changes that contradict the evidence base.

### 7.1 Feedback Items That Risk Working Against the Research

#### Adding a new "alignment review" lifecycle stage

**Feedback source:** Sessions §2.7 — "No formal alignment review stage for cross-document verification before specification."

**Research tension:** Multiple sources counsel against adding lifecycle complexity without demonstrated need:

- Anthropic (Building Effective Agents): "The most successful implementations weren't using complex frameworks or specialized libraries. Instead, they were building with simple, composable patterns." (Orchestration research §1.1)
- Anthropic (Building Effective Agents): "Start with the right level of complexity... Use the lowest level of complexity that reliably meets your requirements." (Orchestration research §1.2, Microsoft)
- The binding registry already has a `researching` stage. The alignment review work described in the feedback — reading two designs, checking them against research, finding deviations — is sequential reasoning that fits within `researching` with the right skill attached.

**Recommendation:** Do not add a new lifecycle stage. Instead, add a `research-integration` or `cross-document-alignment` skill to the skill catalog, bound to the existing `researching` stage. This follows the V3.0 design principle WP-1: "The system prevents skipped steps; skills prevent bad steps." The *system* doesn't need a new stage; the *skill catalog* needs a new skill.

#### Persistent claim-tracking across sessions

**Feedback source:** Sessions §2.6 — "No way to track 'I've verified this claim' persistently across a session."

**Research tension:** The skills research (§3.6) supports *session-scoped* checklists — copy-paste checklists that agents track during a task:

> "For particularly complex workflows, provide a checklist that Claude can copy into its response and check off as it progresses."

But this is session-scoped, not persistent. Building a new persistence mechanism for individual claims within documents would add system complexity without research backing. The skills research Theme 1 warns: "The context window is a public good" — adding tracking overhead to every claim verification competes with the actual verification work.

**Recommendation:** Address this through skill design (a cross-document verification skill with a copy-paste checklist), not through new persistence infrastructure. If the need recurs frequently, consider whether knowledge entries (which already persist) could serve this purpose at a coarser granularity — one entry per "these sections are verified consistent" rather than per individual claim.

#### Section-level cross-reference validation tool

**Feedback source:** Sessions §2.3 — "No automated way to verify `§X.Y` references point to real headings."

**Research tension:** The skills research applies the "novelty test" (§3.1): "Does Claude really need this? Can I assume Claude knows this?" A capable agent can verify that `§3.2` points to a real heading by reading the target document. This is not a task that requires new tooling — it requires the agent to be instructed to do it.

The orchestration research's structural checks (§4.3, §4.5) focus on document-level cross-references at stage gates (spec references design doc, dev-plan references spec) — not intra-document section-reference validation. Building section-level reference validation is a significant `doc_intel` extension with no research precedent.

**Recommendation:** For V3.0, address this through the cross-document alignment skill's procedure ("verify all section cross-references resolve to real headings in the target document"). If agents consistently fail at this despite explicit instructions, *then* consider tooling — following the evaluation-driven principle (skills research Theme 6): identify actual failures first, then build the minimum tool to address them.

### 7.2 Feedback Items Strongly Supported by Research

#### Sub-agent state isolation (pre-V3.0 blocker)

**Research support:** Microsoft explicitly identifies "sharing mutable state between concurrent agents" as a common anti-pattern (Orchestration research §1.2). Google's finding that independent multi-agent systems amplify errors by 17.2× (vs 4.4× with centralised orchestration) is partly caused by uncoordinated state mutations (Orchestration research §2.1). The skills research's skill-creator guidance (§6.4) says: "If all test cases resulted in the subagent writing [the same script], that's a strong signal the skill should bundle that script." The repeated pattern of "commit `.kbz/state/` before spawning sub-agents" should be bundled into the `handoff` or `spawn_agent` workflow, not left as a prose instruction.

**Recommendation:** This blocker is research-justified. The fix should be at the tool level (automatic state commit before sub-agent dispatch), not just as an anti-pattern in skill text. This follows the core research principle: enforceable constraints beat advisory instructions (Orchestration research §2.2).

#### `docint` AC pattern recognition (pre-V3.0 blocker)

**Research support:** The orchestration research (§2.3) is unequivocal: "Performance gains correlate almost linearly with the quality of the induced task graph — underlining that structure learning, not raw language generation, is the critical path." MetaGPT's core finding (Orchestration research §1.1) is that structured intermediate artifacts must be *parseable and verifiable* for the assembly-line paradigm to work. If the parser can't parse the spec format, the entire MetaGPT-inspired pipeline (templates → structural checks → decomposition) is broken.

**Recommendation:** Fix this before V3.0 implementation. The research makes clear that decomposition quality depends on input parsing quality.

#### Agents bypassing MCP tools

**Research support:** This is the exact problem the ACI research addresses. SWE-agent (Orchestration research §1.1): "Purpose-built interfaces for agents dramatically outperform raw access to the same underlying functionality." Anthropic (Orchestration research §1.1): "Bad tool descriptions can send agents down completely wrong paths." The fix is the ACI redesign already in V3.0 Phase B, plus skill-level anti-patterns.

**Recommendation:** Already well-addressed by V3.0. Add the specific anti-patterns suggested by the retro ("In-Session Memory Only" for report-writing, "State Destruction via Git Operations" for orchestration) as skill content items during skill authoring.

### 7.3 Complementary Research Findings Not Yet Applied

#### Evaluation-driven rollout should come first, not last

The V3.0 implementation priority (workflow doc §15) puts observability and evaluation in Phase E — the last phase. Both research documents argue this should come earlier:

- Skills research Theme 6: "Evaluation must precede documentation. Write evaluations before writing extensive documentation. Identify what Claude actually gets wrong, then write the minimum instructions needed to address those specific failures. Don't document imagined problems."
- Skills research §3.9: "evaluation-driven development" — identify gaps first, create evaluations, establish a baseline, *then* write instructions.
- Orchestration research §4.7: The 15–20 scenario evaluation suite is described as "lower-impact" but Anthropic says "Start evaluating immediately with small samples — a set of about 20 test cases was enough to spot dramatic changes in early development."

The V3.0 plan adds mandatory gates (Phase A), then rewrites tool descriptions (Phase B), then adds stage-aware assembly (Phase C), then review loops (Phase D), and *then* builds the evaluation suite to measure whether any of it worked (Phase E). The research says: build the evaluation suite first so you can measure each change as it ships.

**Recommendation:** Move the small-sample evaluation suite (workflow doc §12.5) from Phase E to Phase A. Create 15–20 baseline scenarios *before* enabling mandatory gates. Then measure each subsequent phase's impact against the baseline. This is a sequencing change, not a scope change — the work is already planned.

#### "Start simple" applies to the feedback suggestions too

Anthropic's foundational guidance: "The most successful implementations weren't using complex frameworks or specialized libraries." Some feedback suggestions — section-level cross-reference validation, persistent claim-tracking, a `doc_intel` design review mode — add significant system complexity. The research counsel is to start with the simplest intervention (usually a skill or tool description change) and add tooling only when the simple intervention demonstrably fails.

**Recommendation:** For each feedback item in §5.3 (V3.0 Design Additions), ask: "Can this be addressed through a skill, an anti-pattern, or a tool description change before we build new tooling?" If yes, try that first.

#### The sequential penalty applies to design-phase work

Google's research (Orchestration research §2.4) found that multi-agent coordination degrades sequential reasoning tasks by 39–70%. The design-phase work described in the feedback — cross-referencing two large documents, verifying claims against research, writing alignment reports — is fundamentally sequential reasoning. Tools and conventions for this work should preserve single-agent, focused reasoning.

This has a specific implication: the feedback suggestion of "a `doc_intel` design review mode that takes two document IDs and surfaces sections with overlapping concepts" sounds like it would help, but it could encourage agents to parallelize what should be a single-agent, careful reasoning task. The value is in the agent *thinking through* the cross-references, not in a tool pre-computing them.

**Recommendation:** Design-phase tooling should support the single-agent reasoning pattern (providing focused context, reducing re-reading overhead) rather than attempting to automate the reasoning itself.

#### The "look for repeated work" principle identifies a tool opportunity

The skills research (§6.4) says: "If all test cases resulted in the subagent writing [the same script], that's a strong signal the skill should bundle that script." The feedback describes agents repeatedly re-reading large documents to verify cross-references — the same multi-step operation performed across multiple sessions. Rather than building new validation tooling, the repeated pattern suggests that `doc_intel` should better support *focused reads* of specific sections across documents — reducing the re-reading overhead that makes cross-reference verification expensive.

**Recommendation:** Evaluate whether `doc_intel(action: "section")` with cross-document section retrieval (read §3.2 from document A and §5.1 from document B in one call) would address the re-reading overhead. This is a small extension of existing tooling, not a new system.

#### Document size maps to the progressive disclosure research — but for documents, not skills

The skills research (§3.5) provides three concrete progressive disclosure patterns and recommends keeping SKILL.md under 500 lines with reference files one level deep. The feedback's most frequent friction point — 1,400-line design documents — is the same problem applied to a different artifact type. The research doesn't specifically address design document structure, but the underlying principle is the same: the context window is a shared resource, and large monolithic artifacts consume it inefficiently.

The research *does* warn against deep nesting: "Keep references one level deep from SKILL.md. Claude may partially read files that are referenced from other referenced files." This principle should extend to design documents: a routing document with one-level-deep reference files, not a monolithic document and not a deep hierarchy.

**Recommendation:** Add a design document structure convention to the V3.0 design additions: routing document (problem, decisions, section summaries, ~300–500 lines) with reference files for detailed sections. This applies the progressive disclosure pattern from the skills research to design documents, backed by the same attention-curve and context-window evidence.

### 7.4 Summary: Research Alignment Verdict

| Feedback Item | Research Alignment | Action |
|---|---|---|
| Fix `docint` AC parsing | ✅ Strongly supported (MetaGPT, Masters et al.) | Fix as pre-V3.0 blocker |
| Sub-agent state isolation | ✅ Strongly supported (Microsoft anti-pattern, Google error amplification) | Fix at tool level, not just skill text |
| Agents bypassing MCP tools | ✅ Strongly supported (SWE-agent ACI, Anthropic tool design) | Already in V3.0 Phase B |
| ~~`doc approve` header sync~~ | ~~Supported (poka-yoke)~~ | **Withdrawn** — superseded by consistent-front-matter design (strip `Status` from files, don't sync it) |
| `doc audit` / disk-to-store gap | ➖ Neutral — no research for or against | Quick win if low effort |
| Batch handler false-positive | ✅ Supported (poka-yoke — make errors visible) | Bug fix |
| File ownership in dev-plan template | ✅ Supported (structured intermediate artifacts, MetaGPT) | Add to template |
| "In-Session Memory Only" anti-pattern | ✅ Supported (retrieval for contextual consistency, skills research §4) | Add to skill catalog |
| "State Destruction via Git Ops" anti-pattern | ✅ Supported (Microsoft mutable state anti-pattern) | Add to skill catalog, also fix at tool level |
| Move evaluation suite to Phase A | ✅ Strongly supported (skills research Theme 6, Anthropic) | Resequence implementation priority |
| Alignment review skill (not stage) | ✅ Supported — as skill. ⚠️ Caution — as new lifecycle stage | Add skill, don't add stage |
| Design document progressive disclosure | ✅ Supported (skills research §3.5, Theme 1) | Add convention |
| Section-level cross-ref validation tool | ⚠️ Caution — fails novelty test, adds complexity | Try skill-based approach first |
| Persistent claim-tracking | ⚠️ Caution — no research backing for persistence mechanism | Use session-scoped checklists |
| `doc_intel` design review mode | ⚠️ Caution — may encourage parallelizing sequential work | Evaluate whether focused section reads suffice |
| Research integrity review dimension | ➖ Neutral — reasonable but not research-driven | Consider as skill content, not system feature |

---

## 8. Recommended Release Phasing: 2.5 / 3.0

The infrastructure and tooling items (§5.1) are independent of the V3.0 design — they fix bugs, add missing capabilities, and clear blockers. The skills doc and workflow doc changes (§5.2, §5.3) all assume the V3.0 skills system and can't ship until that design is implemented. This creates a natural release boundary.

### 8.1 Rationale

A **2.5 release** focused on infrastructure fixes would:

1. **Clear the V3.0 blockers.** The `docint` parser, state isolation, and batch handler issues will cause V3.0 to fail in practice if unfixed. Fixing them now means V3.0 development can focus on the design, not on firefighting infrastructure.
2. **Deliver immediate value.** The `doc approve` header sync, `doc audit`, and `doc import --dry-run` address friction that agents and humans experience today, independent of V3.0.
3. **Establish the measurement baseline.** The research is emphatic that evaluation must precede the changes it measures. Building the evaluation suite against the current (2.5) system gives us the "before" data that makes V3.0's impact measurable.
4. **Carry no design risk.** Every 2.5 item is additive (new commands, new capability) or corrective (bug fixes). Nothing changes existing behaviour in ways that could conflict with V3.0's design.

### 8.2 Version 2.5 Scope

**Bug fixes (blockers):**

| Item | Description | Risk if Deferred |
|------|-------------|------------------|
| `docint` AC-NN pattern recognition | Teach the document intelligence classifier to recognise the `**AC-NN.**` acceptance criteria format used in specs | V3.0's template → gate → decomposition pipeline is built on a parser that can't parse the actual spec format |
| Sub-agent state isolation | Commit `.kbz/state/` before sub-agent dispatch in the `handoff`/delegation path | V3.0's `orchestrator-workers` topology will cause catastrophic state loss the first time a sub-agent runs `git stash` |
| Batch handler false-positive | Fix `ExecuteBatch` to not count `(toolResultError, nil)` as success | Validation failures silently pass, masking real problems in batch operations |

**Tool enhancements (quick wins):**

| Item | Description | Value |
|------|-------------|-------|
| `doc approve` header sync | Patch the file's `Status:` front matter field when `doc approve` is called | Eliminates confusion when agents or humans read files directly instead of querying the store |
| `doc audit` / `doc unregistered` | New command: list files under known document directories that aren't registered in the store | Prevents the silent drift that led to 22 unregistered documents accumulating during normal work |
| `doc import --dry-run` | Show what `doc import` would register — inferred types, titles, owners — without committing | Makes `doc import` trustworthy enough to use, rather than falling back to manual batch registration |

**Evaluation baseline (measurement infrastructure):**

| Item | Description | Value |
|------|-------------|-------|
| Small-sample evaluation suite | Define 15–20 representative workflow scenarios against the current system. Capture baseline measurements: which tools agents call, in what order, where they get stuck, whether features reach the correct state. | Gives V3.0 a "before" measurement. Without this, we ship gates, templates, and tool descriptions with no way to know if they helped. The research (Anthropic, skills research Theme 6) says this must come before the changes, not after. |

### 8.3 Version 3.0 Scope (Unchanged)

V3.0 remains as currently designed in the two design documents, with these adjustments from this report:

**Workflow doc changes:**

- Move the evaluation suite from Phase E to Phase A (it's already built in 2.5; Phase A uses it to measure the impact of mandatory gates).

**Skills doc additions:**

- Add "File Ownership" to the dev-plan template required sections (§5.1).
- Add a `cross-document-alignment` skill to the skill catalog, bound to the `researching` stage (not a new lifecycle stage).
- Add "In-Session Memory Only" anti-pattern to `write-research` and report-writing skills (§5.1).
- Add "State Destruction via Git Operations" anti-pattern to `orchestrate-development` and `orchestrate-review` skills (§5.2).
- Add a research integrity criterion to review skills (§5.3).
- Add a design document progressive disclosure convention (§8 or new §8.6).

### 8.4 What's Not in Either Release

These items are deferred until the simpler interventions in 2.5 and 3.0 have been tried and evaluated:

| Item | Why Deferred |
|------|-------------|
| Persistent claim-tracking | Try the session-scoped checklist in the alignment skill (3.0) first. Build persistence only if that demonstrably fails. |
| Section-level cross-reference validation tool | Fails the novelty test. Try the skill-based approach first. |
| `doc_intel` design review mode | May encourage parallelizing sequential work. Evaluate whether focused section reads (2.5 #7, if included) suffice. |

### 8.5 Dependency Diagram

```
  2.5 (infrastructure)          3.0 (design implementation)
  ─────────────────────         ───────────────────────────
  Fix docint AC parsing ──────► Template → gate → decomposition pipeline works
  Fix state isolation ────────► orchestrator-workers topology is safe
  Fix batch handler ──────────► Batch operations report real results
  doc approve sync ───────────► Agents see correct status in files
  doc audit ──────────────────► Document store stays current
  doc import --dry-run ───────► Import is trustworthy
  Evaluation baseline ────────► Phase A measures gate impact against baseline
                                Phase B measures ACI audit impact
                                Phase C measures assembly changes
                                Phase D measures review loop changes
```

### 8.6 Scope Note on `doc_intel` Section Reads

The focused cross-document section reads enhancement (#7 in §5.1) is a judgment call. It's additive, doesn't conflict with V3.0, and addresses the most frequently raised friction point (document size / re-reading overhead). But it's more effort than the other 2.5 items and isn't a blocker for V3.0.

**Recommendation:** Include in 2.5 if the effort is small (extending `doc_intel(action: "section")` to accept multiple document+section pairs). Defer to 3.0 if it requires significant `doc_intel` refactoring. Make the call during 2.5 planning based on an implementation estimate.