# Kanbanzai 2.0 Workflow Retrospective

**Date:** 2025-03-27T22:58:57Z
**Scope:** First complex feature implementation using Kanbanzai 2.0 workflow
**Sources:** 2 human observations, 6 AI agent retrospectives
**Purpose:** Inform the next version of Kanbanzai

---

## Table of Contents

1. [Executive Summary](#1-executive-summary)
2. [What Worked Well](#2-what-worked-well)
3. [Friction Points — Themed Analysis](#3-friction-points--themed-analysis)
   - 3.1 [Feature Lifecycle Transitions Are Too Verbose](#31-feature-lifecycle-transitions-are-too-verbose)
   - 3.2 [No First-Class Review Workflow](#32-no-first-class-review-workflow)
   - 3.3 [Entity State Inconsistencies Go Undetected](#33-entity-state-inconsistencies-go-undetected)
   - 3.4 [IDs Are Not Human-Friendly](#34-ids-are-not-human-friendly)
   - 3.5 [No Guided Plan-to-Feature Decomposition Step](#35-no-guided-plan-to-feature-decomposition-step)
   - 3.6 [Tool Gaps for Core Entity Operations](#36-tool-gaps-for-core-entity-operations)
   - 3.7 [Error Messages Don't Guide the User](#37-error-messages-dont-guide-the-user)
   - 3.8 [Bulk Operations and Batch Workflows Are Missing](#38-bulk-operations-and-batch-workflows-are-missing)
   - 3.9 [The `design` Field on Features Is Inert](#39-the-design-field-on-features-is-inert)
   - 3.10 [Retrospective Capture Has No Entry Point Outside Tasks](#310-retrospective-capture-has-no-entry-point-outside-tasks)
   - 3.11 [Documentation Drift Goes Undetected](#311-documentation-drift-goes-undetected)
   - 3.12 [Knowledge Graph Staleness Is Silent](#312-knowledge-graph-staleness-is-silent)
4. [Recommendations — Prioritised](#4-recommendations--prioritised)
5. [Appendix: Raw Signal Sources](#5-appendix-raw-signal-sources)

---

## 1. Executive Summary

The first complex feature built using Kanbanzai 2.0 produced strong validation of the core model — spec-linked entities, document intelligence, phased feature structures, and dependency-driven work queues all delivered real value. Agents consistently reported that context gathering dropped from minutes of grepping to a handful of tool calls.

The friction was concentrated in three areas:

1. **Lifecycle ceremony** — the document-driven state machine forces agents through 5–6 transitions even when upstream documents already exist. This was the single most-reported friction point (raised by 4 of 6 agents).
2. **No review workflow** — the system is optimised for the dispatch→implement→complete loop but has no structured support for the review→audit→verify loop. Three agents independently improvised the same manual process.
3. **State consistency gaps** — features marked "done" with non-terminal children, worktrees still "active" after merge, and no health check catches these. Three agents were confused by entity states that contradicted reality.

Two human-reported issues also surfaced: IDs lack the planned human-friendly split format, and there is no workflow guidance for the plan→features stage (before specification).

None of these are architectural problems. The foundations — entity model, document intelligence, knowledge store, dependency engine — are sound. The issues are in the workflow layer built on top of those foundations.

---

## 2. What Worked Well

These findings were consistent across multiple agents and represent strengths to preserve.

### 2.1 Spec-Linked Entity Traversal

**Reported by:** 5 of 6 agents

The `feature → design field → spec document → acceptance criteria` chain was universally praised. Agents went from a feature ID to the full binding specification in 2–3 tool calls. This replaced what would normally be 10+ minutes of directory exploration and manual correlation.

> "One `doc_record_get_content` call and I had the full spec with numbered acceptance criteria. No searching, no guessing what I was supposed to build. That single link eliminated what would normally be 10 minutes of context gathering."

### 2.2 Document Intelligence

**Reported by:** 4 of 6 agents

`get_entity`, `doc_record_get_content`, `list_entities_filtered`, and `doc_find_by_entity` gave agents the complete picture fast. Drift detection on approved specs meant agents could trust what they were reading without wondering if it was the current version.

### 2.3 Phased Feature Structure and Dependency Engine

**Reported by:** 3 of 6 agents

The phased feature model (Phase 1 done → Phase 2 done → Phase 3 is mine) made dependency reasoning trivial. The `work_queue` automatic promotion as dependencies reached terminal state was called "genuinely satisfying" — agents didn't have to manually track readiness.

### 2.4 Acceptance Criteria Traceability

**Reported by:** 4 of 6 agents

Numbered acceptance criteria (P5-1.x, P5-2.x) embedded in specs and referenced in task summaries made verification systematic rather than impressionistic. Agents could build coverage matrices and verify completeness.

### 2.5 Codebase Pattern Consistency

**Reported by:** 2 of 6 agents

The consistent code structure (`XxxTool()` returns `[]server.ServerTool`, services in `internal/service/`, tools in `internal/mcp/`) meant agents could look at existing tools and immediately know the shape of new ones. Good code patterns produce predictable extension points.

### 2.6 Test Infrastructure

**Reported by:** 3 of 6 agents

`t.TempDir()`, direct fixture creation via storage APIs, table-driven test patterns, and `go test -race ./...` as the single verification command provided fast, authoritative feedback. Agents didn't have to invent test scaffolding.

### 2.7 Knowledge Store Design

**Reported by:** 2 of 6 agents

The decision that retrospective signals *are* knowledge entries (not a new entity type) was validated — agents could test storage by writing knowledge records directly, and the tag-based organisation riding existing infrastructure simplified both implementation and review.

---

## 3. Friction Points — Themed Analysis

Friction points are grouped by theme, not by source. Each theme includes the number of independent reports, severity assessment, and concrete examples.

### 3.1 Feature Lifecycle Transitions Are Too Verbose

**Reports:** 4 of 6 agents
**Severity:** High — this was the most-reported friction point
**Category:** workflow-friction

The document-driven lifecycle requires features to progress through `proposed → designing → specifying → dev-planning → developing → done` — six states, five transitions. Each transition requires a separate `update_status` call. For features where the design and specification already exist (e.g., a feature extracted from a plan-level spec, or a phase in a multi-phase plan), this is pure ceremony.

Agents reported:

- Trial-and-error to discover the correct transition sequence
- Error messages that said a transition was *invalid* but never suggested *valid* alternatives
- Features stuck in `proposed` because the lifecycle gates don't recognise that upstream documents are already approved
- "Six transitions added no value" for features with existing specs

**Root cause:** The lifecycle assumes every feature goes through the full design cycle in order. It has no mechanism to recognise that prerequisite stages are already satisfied.

### 3.2 No First-Class Review Workflow

**Reports:** 3 of 6 agents
**Severity:** High — agents independently improvised the same manual process
**Category:** design-gap

The system is optimised for dispatch→implement→complete but has no structured support for review→audit→verify. `review_task_output` exists for individual tasks, but there is nothing for "review a feature against its spec." Agents had to manually assemble:

1. Get feature entity and spec
2. Find implementation files (via grep — no "what files implement this feature?" query)
3. Read code and run tests
4. Cross-reference acceptance criteria to test coverage
5. Compile findings as free text

This took ~15 tool calls and produced unstructured output. A structured review workflow could assemble the spec, linked tasks, implementation files, and test coverage into one context packet — making reviews repeatable and their findings actionable (e.g., auto-dispatched as follow-up tasks or bug entities).

Additionally, review findings arrive as free text in chat, entirely outside the workflow. There is no connection to a task, entity, or document record, and no structured findings to iterate over.

### 3.3 Entity State Inconsistencies Go Undetected

**Reports:** 3 of 6 agents
**Severity:** High — causes confusion and erodes trust in entity state
**Category:** design-gap

Multiple agents encountered entity states that contradicted reality:

- Feature marked "done" with tasks still in "ready" or "queued"
- Feature stuck in "proposed" with all child tasks "done"
- Worktree marked "active" when the branch was already merged into main

No health check or warning surfaced these inconsistencies. Agents had to triangulate from git history, task states, and downstream feature states to determine the actual situation.

**Root cause:** The system enforces lifecycle transitions on individual entities but does not enforce consistency *across* parent-child relationships or between entity state and git state.

### 3.4 IDs Are Not Human-Friendly

**Reports:** 1 human
**Severity:** Medium
**Category:** design-gap

The original design intended IDs to be split for human readability:

| Current | Intended |
|---|---|
| `FEAT-01KMRJ81DZ3X2` | `FEAT-01KMR-J81DZ3X2` |

The split format makes IDs easier to read, communicate verbally, and type partially. This was part of the original design but was not implemented.

### 3.5 No Guided Plan-to-Feature Decomposition Step

**Reports:** 1 human
**Severity:** Medium
**Category:** workflow-gap

There is no workflow guidance for the step between creating a plan and writing a specification. The human wanted to express: "Before you create a spec, maybe you should break the plan into features first?"

The current stage gates (§ Workflow Stage Gates in AGENTS.md) define the sequence planning → design → features → spec → dev-plan → tasks, but there is no tool or prompt that guides agents through the plan→features transition. Should the AI agent proactively suggest feature extraction from a design document? Should there be an explicit tool or workflow step?

### 3.6 Tool Gaps for Core Entity Operations

**Reports:** 2 of 6 agents
**Severity:** Medium
**Category:** tool-gap

Two specific tool gaps were identified:

1. **`update_entity` cannot set `depends_on` on tasks.** Agents had to edit YAML files directly to wire dependency chains. A task's dependencies are its most important structural property and should not require bypassing the tools.

2. **`list_entities_filtered` parent filter returns empty results.** Filtering tasks by `parent: FEAT-01KMRJ81DZ3X2` returned 0 results, forcing agents to list all tasks and scan manually. This is either a bug or an undocumented limitation — either way, the silent empty result is the wrong behaviour.

### 3.7 Error Messages Don't Guide the User

**Reports:** 2 of 6 agents
**Severity:** Medium
**Category:** usability

Lifecycle transition errors say a transition is *invalid* but never suggest what the *valid* next states are. The data exists in the `allowedTransitions` map. Instead of:

> `"invalid feature transition "proposed" → "done""`

The error should say:

> `"invalid transition; from "proposed" you can go to: designing, specifying, superseded, cancelled"`

This would eliminate the trial-and-error that agents reported when navigating lifecycle transitions.

### 3.8 Bulk Operations and Batch Workflows Are Missing

**Reports:** 2 of 6 agents
**Severity:** Medium
**Category:** workflow-friction

When work is completed in a single commit but spans multiple tasks, agents must individually transition each task through `queued → ready → active → done` — up to 11+ tool calls for 5 tasks. The `finish` tool handles ready→active shortcuts, but there is no equivalent for:

- Bulk-marking tasks that were implemented together
- Sequential task chains where all work is already done
- Backfilling status on tasks created after the work was committed

### 3.9 The `design` Field on Features Is Inert

**Reports:** 1 of 6 agents
**Severity:** Medium
**Category:** design-gap

The `design` field on feature entities exists in the schema and can be set, but no tool follows it. `decompose_feature` ignores it. The field creates false confidence — it looks meaningful but has no effect on tool behaviour.

> "Having a field that looks meaningful but isn't is worse than not having the field — it creates false confidence."

### 3.10 Retrospective Capture Has No Entry Point Outside Tasks

**Reports:** 2 of 6 agents
**Severity:** Low
**Category:** workflow-gap

The `finish` → retrospective pipeline only triggers on task completion. For work done without task entities (single-session implementations, reviews, ad-hoc fixes), there is no natural capture point for retrospective signals. The `knowledge_contribute` path exists but feels like an afterthought — a lightweight standalone `retro` command would make observation capture more natural.

The irony was noted: the agents who built the retrospective system couldn't use it to capture their own retrospective signals.

### 3.11 Documentation Drift Goes Undetected

**Reports:** 1 of 6 agents
**Severity:** Low
**Category:** tool-gap

The MCP tool reference document (`docs/mcp-tool-reference.md`) still referenced "97 tools" when 2.0 has 20, and newly added tools were absent. Despite the system having document intelligence and drift detection, no automated check caught this. The reference document is too large and too structural to maintain reliably by hand.

### 3.12 Knowledge Graph Staleness Is Silent

**Reports:** 1 of 6 agents
**Severity:** Low
**Category:** tool-gap

The codebase knowledge graph (`codebase-memory-mcp`) returned zero results for code that existed — likely because the graph wasn't re-indexed after new code was added. There is no warning when results might be incomplete due to staleness. Agents following the "prefer graph tools" guidance got empty results and had to fall back to grep without knowing why.

---

## 4. Recommendations — Prioritised

Recommendations are ordered by expected impact, accounting for both severity and frequency of the underlying friction.

### P0 — High Impact, Address First

#### R1: Smart Lifecycle Transitions

Allow features to skip lifecycle stages when prerequisite documents already exist. Options (not mutually exclusive):

- **Auto-skip:** When transitioning, if the required document for the next stage is already approved, automatically advance through intermediate states.
- **Force flag:** `update_status(id, status="developing", skip_gates=true)` with an audit trail.
- **Bulk transition:** `update_status(id, status="done")` that walks through all intermediate states in one call when preconditions are met.

At minimum, surface valid transitions in error messages so agents don't have to trial-and-error.

#### R2: Feature Review Workflow

Add a structured review primitive that assembles:

- The feature's spec and acceptance criteria
- The task decomposition and task statuses
- The implementation files (aggregated from task `files_modified`)
- Test coverage mapped to acceptance criteria
- Open findings from previous reviews

Output should be structured (not free text) so findings can be auto-dispatched as follow-up tasks or bug entities.

#### R3: Parent-Child State Consistency

Add health check rules for cross-entity consistency:

- Feature is `done` but has non-terminal children → warning
- Feature is `proposed` but all children are `done` → warning
- Worktree is `active` but branch is merged → warning
- Feature has non-terminal children but no `active` children → informational

Surface these in `health_check` output and optionally as warnings on `get_entity`.

### P1 — Medium Impact, Address Next

#### R4: Human-Friendly ID Display

Implement the planned ID split format: `FEAT-01KMR-J81DZ3X2` instead of `FEAT-01KMRJ81DZ3X2`. This was part of the original design. The split should be a display concern — storage can remain unsplit. Accept both formats as input.

#### R5: Plan-to-Feature Guidance

Add a workflow step or tool that helps decompose a plan (or approved design document) into features before specification begins. This could be:

- A `suggest_features` tool that analyses a design document and proposes feature extraction
- A prompt in the agent protocol: "Before writing a specification, check whether the plan should be decomposed into features first"
- A lifecycle gate: plans cannot transition to `active` without at least one child feature

#### R6: `update_entity` Should Support `depends_on`

Allow `update_entity` to set structural fields like `depends_on` on tasks. Agents should never need to edit YAML directly for core workflow operations.

#### R7: Fix `list_entities_filtered` Parent Filter

The parent filter either has a bug or undocumented constraints. Silent empty results for a valid-looking query are the wrong behaviour. Either fix the filter or return an error explaining the correct usage.

#### R8: Bulk Task Completion

Add a batch variant of `finish` that accepts multiple task IDs and transitions them through the required states in dependency order. For tasks that were implemented together in a single commit, this eliminates O(n × transitions) tool calls.

#### R9: Valid Transitions in Error Messages

When a lifecycle transition is rejected, include the list of valid transitions from the current state. The data is already in the `allowedTransitions` map.

### P2 — Lower Impact, Address When Convenient

#### R10: Make the `design` Field Functional

Either make the `design` field on features influence tool behaviour (e.g., `decompose_feature` follows it to find the spec) or remove it from the schema. An inert field that looks meaningful is worse than no field.

#### R11: Standalone Retrospective Capture

Add a lightweight retrospective entry point that doesn't require an active task. This could be a standalone `retro` command or a `knowledge_contribute` wrapper with retrospective-specific defaults (tags, scope, tier).

#### R12: Auto-Generate or Validate Tool Reference Docs

Add a CI check or generation step that compares the MCP tool reference document against the live server's tool registrations. A hand-maintained 2780-line reference document will always drift.

#### R13: Knowledge Graph Staleness Warning

When graph queries return zero results, surface a hint: "No results found. The graph may be stale — last indexed at {timestamp}. Run `index_repository` to refresh."

### Deferred — Monitor, Don't Act Yet

#### D1: Retrospective Clustering Quality at Scale

The greedy Jaccard clustering with a union-expanding centroid may produce fewer, larger clusters as signals accumulate. Monitor cluster quality as the signal corpus grows. No action needed now.

#### D2: Test Fixture Fragility with TSID IDs

Tight coupling between filename format and ID format means test fixtures need to know about TSID internals. A `testutil.CreateDecision(t, root, ...)` helper using the real allocator would prevent this class of bug. Low priority — affects tests, not production.

#### D3: Tool Count Assertions in Tests

Three test files hardcode `20` as the expected tool count. Deriving the count from the group map would make this zero-maintenance. Low priority — annoyance, not a bug.

---

## 5. Appendix: Raw Signal Sources

| # | Source | Role | Primary Topics |
|---|--------|------|----------------|
| 1 | Human | User | ID readability |
| 2 | Human | User | Plan-to-feature workflow gap |
| 3 | Agent | Implementer (Phase 3) | Lifecycle verbosity, dependency engine praise, retro capture gap |
| 4 | Agent | Implementer (Phase 2) | Lifecycle verbosity, spec-linked pattern praise, tool count tests |
| 5 | Agent | Implementer (Phase 1) | `depends_on` tool gap, `design` field inert, batch finish, dependency engine praise |
| 6 | Agent | Reviewer (Phase 1) | Stale worktree, state inconsistency, no review workflow, acceptance criteria praise |
| 7 | Agent | Reviewer (Phase 2) | `list_entities_filtered` bug, no review workflow, knowledge graph staleness |
| 8 | Agent | Fixer (post-review) | Review arrives as free text, doc drift undetected, lifecycle for bookkeeping only |

### Signal Frequency Matrix

| Theme | Agent 3 | Agent 4 | Agent 5 | Agent 6 | Agent 7 | Agent 8 | Total |
|-------|---------|---------|---------|---------|---------|---------|-------|
| Lifecycle verbosity | ✓ | ✓ | ✓ | | | ✓ | 4 |
| No review workflow | | | | ✓ | ✓ | ✓ | 3 |
| State inconsistency | | | ✓ | ✓ | ✓ | | 3 |
| Spec-linked praise | ✓ | ✓ | ✓ | ✓ | ✓ | | 5 |
| Acceptance criteria praise | ✓ | ✓ | | ✓ | ✓ | | 4 |
| Dependency engine praise | ✓ | | ✓ | | | | 2 |
| Bulk operations gap | ✓ | | ✓ | | | | 2 |
| Error message guidance | | ✓ | | | | | 1 |
| `depends_on` tool gap | | | ✓ | | | | 1 |
| `design` field inert | | | ✓ | | | | 1 |
| Doc drift undetected | | | | | | ✓ | 1 |
| Graph staleness silent | | | | | ✓ | | 1 |