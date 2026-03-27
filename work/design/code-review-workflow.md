# Code Review Workflow Design

- Status: design proposal
- Purpose: define a feature-level code review gate and orchestrated review workflow for Kanbanzai
- Date: 2026-03-27T13:26:39Z
- Related:
  - `work/design/quality-gates-and-review-policy.md` (review dimensions, profiles, output format)
  - `work/design/document-centric-interface.md` (human interface model)
  - `work/design/workflow-design-basis.md` §4, §6 (lifecycle design)
  - `work/design/machine-context-design.md` (context assembly for agents)
  - `work/design/agent-interaction-protocol.md` (agent behavior)
  - `work/spec/phase-4b-specification.md` (task-level review, rework lifecycle)

---

## 1. Purpose

This document proposes a code review workflow for Kanbanzai that adds a feature-level review gate between implementation and completion.

The goal is to ensure that no feature is marked done until its implementation has been reviewed against the specification, quality standards, and project conventions — and to make that review process orchestratable by AI agents working in parallel, with minimal human intervention for the mechanical parts.

---

## 2. Problem Statement

### 2.1 The feature lifecycle has no review gate

The Phase 2 feature lifecycle goes:

    proposed → designing → specifying → dev-planning → developing → done

There is no review state. A feature transitions directly from `developing` to `done` when its tasks are complete. This means "done" only means "implemented" — not "reviewed."

The deprecated Phase 1 lifecycle had explicit review states (`in-progress → review → done / needs-rework`), but these were dropped in Phase 2's document-driven redesign without replacement.

### 2.2 Task-level review exists but is insufficient

Tasks have `needs-review` and `needs-rework` states, and the `finish` tool supports `to_status="needs-review"`. But task-level review checks individual units of work in isolation. It cannot answer feature-level questions:

- Does the feature as a whole satisfy its specification?
- Do the pieces fit together correctly?
- Is the documentation complete across the feature?
- Is the workflow state consistent?

### 2.3 The review tool was removed without replacement

The 1.0 `review_task_output` tool was removed in 2.0 Track K. The `ReviewService` code still exists in `internal/service/review.go`, but no MCP tool exposes it. There is currently no tool-assisted review capability in the 2.0 surface.

### 2.4 Review is currently a manual prompt

Today, requesting a review means typing something like:

> Please review the implementation of X: verify spec compliance, check code quality, ensure test coverage, check documentation, verify workflow state.

This is ad hoc, inconsistent, and doesn't integrate with the workflow lifecycle. The quality gates policy (`quality-gates-and-review-policy.md`) defines five review dimensions and structured output formats, but nothing operationalises them.

### 2.5 Large reviews exceed single-agent context

A single feature may touch 5–15 files. A phase-level review may cover dozens of features and hundreds of files. No single agent can hold all of this in context simultaneously. Review must be decomposable into parallel work units — but the current system has no mechanism for this.

---

## 3. Design Principles

### 3.1 Review is a first-class workflow state

Review is not an informal step. It is a lifecycle gate with the same standing as design, specification, and implementation. Features cannot reach `done` without passing through review.

### 3.2 Analysis and remediation are separate phases

Review decomposes into two distinct phases:

- **Analysis** — read-only, parallelisable. Sub-agents examine code against spec and report structured findings. No code is written.
- **Remediation** — write, potentially parallelisable. Remediation tasks address blocking findings. Conflict analysis determines safe parallelism.

Separating these phases eliminates merge conflicts during analysis and makes review inherently scalable.

### 3.3 The orchestrator works at the metadata level

The orchestrator never needs to hold the full codebase in context. It works with entity states, file lists, spec structure, and review findings. Sub-agents receive targeted context packets containing only the code and spec sections relevant to their review unit.

### 3.4 Structured findings, not prose judgments

Review output follows the structured format defined in the quality gates policy (§11): per-dimension outcomes (`pass` / `pass_with_notes` / `concern` / `fail` / `not_applicable`) with an overall verdict (`approved` / `approved_with_followups` / `changes_required` / `blocked`).

### 3.5 Build on what exists

The quality gates policy, context profiles, document intelligence, and the dispatch/complete workflow already provide the building blocks. This design composes them rather than replacing them.

---

## 4. Feature Lifecycle Change

### 4.1 New states

Two new feature statuses are added to the Phase 2 lifecycle:

- **`reviewing`** — implementation is complete, review is in progress.
- **`needs-rework`** — review found blocking issues, implementation must be corrected.

### 4.2 Updated transition map

```
proposed     → designing, specifying, superseded, cancelled
designing    → specifying, superseded, cancelled
specifying   → dev-planning, designing (backward), superseded, cancelled
dev-planning → developing, specifying (backward), superseded, cancelled
developing   → reviewing, dev-planning (backward), superseded, cancelled
reviewing    → done, needs-rework, superseded, cancelled
needs-rework → developing, reviewing (quick-fix), superseded, cancelled
```

Key transitions:

| From | To | Meaning |
|---|---|---|
| `developing` | `reviewing` | Implementation complete, review begins |
| `reviewing` | `done` | Review passed |
| `reviewing` | `needs-rework` | Review found blocking issues |
| `needs-rework` | `developing` | Rework begins (standard path) |
| `needs-rework` | `reviewing` | Quick-fix path for minor rework that doesn't need a full development cycle |

### 4.3 Relationship to task states

Feature review begins after all implementation tasks are in terminal states (`done`, `not-planned`, or `duplicate`). The feature-level review examines the aggregate result, not individual tasks.

Review may create new remediation tasks. These are children of the feature and follow the normal task lifecycle. When all remediation tasks reach terminal states, the feature can transition from `needs-rework` back to `reviewing` or directly to `done` if the orchestrator is satisfied.

### 4.4 The direct path is removed

The transition `developing → done` is removed. All features must pass through `reviewing`. This is the definition-of-done change: **no feature is complete until it has been reviewed.**

For features where review is trivially unnecessary (e.g., documentation-only changes using the lightweight review profile), the review phase may be brief, but it must still occur as an explicit lifecycle transition.

---

## 5. The Review Workflow

### 5.1 Overview

```
┌─────────────────────────────────────────────────┐
│                 ANALYSIS PHASE                   │
│              (read-only, parallel)               │
│                                                  │
│  1. Feature → reviewing                         │
│  2. Orchestrator queries metadata                │
│  3. Orchestrator decomposes into review units    │
│  4. Sub-agents review in parallel                │
│  5. Orchestrator collates findings               │
│  6. Review document created                      │
└────────────────────┬────────────────────────────┘
                     │
              ┌──────┴──────┐
              │  Decision   │
              └──────┬──────┘
                     │
         ┌───────────┼───────────┐
         │           │           │
    No blocking   Blocking   Human
    findings      findings   judgment
         │           │       needed
         ▼           ▼           ▼
    Feature →   Feature →    Human
      done      needs-       checkpoint
                rework
                  │
                  ▼
┌─────────────────────────────────────────────────┐
│              REMEDIATION PHASE                   │
│        (write, sequential or parallel)           │
│                                                  │
│  7. Orchestrator creates remediation tasks       │
│  8. Tasks dispatched (conflict-checked)          │
│  9. On completion, re-review affected sections   │
│ 10. Feature → reviewing (or → done)             │
└─────────────────────────────────────────────────┘
```

### 5.2 Analysis phase in detail

#### Step 1: Trigger

The feature transitions to `reviewing`. This can be triggered:
- Manually by a human
- By an orchestrator when all implementation tasks are terminal
- By the `finish` tool if the last task completes (future enhancement)

#### Step 2: Orchestrator queries metadata

The orchestrator gathers:
- The feature entity and its spec document reference
- The spec document structure via `doc_outline`
- The list of tasks and their modified files
- The review profile to apply (default: Feature Implementation Review Profile from §5.1 of the quality gates policy)

This is all metadata. The orchestrator does not read source code.

#### Step 3: Decompose into review units

The orchestrator partitions the work into logical review units. Decomposition strategies include:

- **By package** — each Go package reviewed as a unit
- **By spec section** — each specification section reviewed against its implementation
- **By concern** — cross-cutting concerns (tests, documentation, workflow) reviewed separately
- **By layer** — service layer, storage layer, MCP layer reviewed independently

The choice of strategy depends on the feature's size and structure. Small features (≤10 files) may be a single review unit. Larger features should use 3–8 parallel units.

Each review unit is defined by:
- A set of source files to examine
- The relevant spec section(s)
- The review dimensions to check
- The review profile to apply

#### Step 4: Sub-agents review in parallel

Each sub-agent receives:
- A context packet assembled with `context_assemble(role="reviewer")`
- The review SKILL (procedure to follow)
- The specific files and spec sections for their unit
- The structured output format to use

Sub-agents produce structured findings. They do not modify any files.

#### Step 5: Orchestrator collates findings

The orchestrator merges findings from all sub-agents:
- Deduplicates overlapping findings
- Categorises as blocking or non-blocking
- Determines overall verdict per dimension
- Computes aggregate verdict

#### Step 6: Review document created

Findings are written to a review document, associated with the feature. This provides:
- A human-readable record of what was reviewed
- A machine-readable structure for remediation planning
- An audit trail

### 5.3 Remediation phase in detail

If the review finds blocking issues:

#### Step 7: Create remediation tasks

The orchestrator creates tasks as children of the feature, one per blocking finding or logical group of related findings. Each task summary references the review finding it addresses.

#### Step 8: Dispatch tasks

Tasks are dispatched through the normal workflow. The orchestrator uses `conflict_domain_check` to determine which tasks can run in parallel safely.

#### Step 9: Re-review affected sections

After remediation tasks complete, the orchestrator re-reviews only the affected sections — not the entire feature. This is a targeted re-analysis using the same sub-agent pattern.

#### Step 10: Transition

If re-review passes, the feature transitions to `done`. If new blocking issues are found, the cycle repeats with the feature remaining in `reviewing` (or transitioning through `needs-rework` if a more substantial development cycle is needed).

---

## 6. The Orchestrator's Context Budget

A key concern is whether the orchestrator can manage a large review without exceeding its context window. The answer is yes, because the orchestrator operates at the metadata level:

### 6.1 What the orchestrator holds in context

| Data | Size | Source |
|---|---|---|
| Feature entity state | ~200 bytes | `get_entity` |
| Spec document outline | ~1–2 KB | `doc_outline` |
| Task list with file paths | ~1–3 KB | `query_plan_tasks` or `list_entities_filtered` |
| Review SKILL (procedure) | ~2–3 KB | `.skills/code-review.md` |
| Collated findings | ~2–5 KB | Sub-agent outputs |

**Total: ~6–14 KB** — well within any context window, regardless of codebase size.

### 6.2 What sub-agents hold in context

| Data | Size | Source |
|---|---|---|
| Reviewer profile | ~2 KB | `context_assemble(role="reviewer")` |
| Review SKILL | ~2–3 KB | `.skills/code-review.md` |
| Spec section(s) | ~2–5 KB | `doc_section` |
| Source files (their review unit) | ~5–20 KB | File reads |
| Structured output template | ~0.5 KB | From SKILL |

**Total: ~12–30 KB per sub-agent** — comfortably within context for a focused review unit.

### 6.3 Scaling to phase-level reviews

For reviewing an entire phase (many features):
- The orchestrator iterates over features, processing each through the review workflow
- Parallelism is within each feature (review units) and across features (independent features can be reviewed concurrently)
- The orchestrator's per-feature context cost is constant (~6–14 KB)
- The total review scales linearly with features, not with codebase size

---

## 7. Review Profiles and Dimensions

This design does not redefine review dimensions. It operationalises the dimensions and profiles already defined in `quality-gates-and-review-policy.md`.

### 7.1 Mapping to existing policy

| Quality gates policy concept | Operationalised as |
|---|---|
| Review dimensions (§4) | Fields in structured review output |
| Feature Implementation Review Profile (§5.1) | Default profile for feature review |
| Bugfix Review Profile (§5.2) | Default profile for bug review |
| Merge Readiness Review Profile (§5.3) | Applied at merge gate |
| Lightweight Review Profile (§5.4) | For trivial features |
| Review output format (§11) | Sub-agent output structure |
| Blocking vs non-blocking (§12) | Determines remediation vs follow-up |

### 7.2 Review profile selection

The review profile is selected based on:
1. Explicit human instruction (highest priority)
2. Entity type (feature → Feature Implementation, bug → Bugfix)
3. Feature size heuristic (≤3 files, no spec → Lightweight)

---

## 8. Reviewer Context Profile

A new context profile `.kbz/context/roles/reviewer.yaml` is created:

```
id: reviewer
inherits: base
description: >
  Context profile for code review agents. Provides review dimensions,
  structured output format, and quality gate criteria.

conventions:
  review_approach:
    - "Review is structured, not conversational. Produce findings, not commentary."
    - "Every finding has a dimension, severity, location, and description."
    - "Blocking findings must cite the specific requirement or convention violated."
    - "Non-blocking findings are suggestions, not demands."
    - "When uncertain whether something is a defect, classify as concern, not fail."

  output_format:
    - "Use the structured review output format from the review SKILL."
    - "Report per-dimension outcomes: pass, pass_with_notes, concern, fail, not_applicable."
    - "Report overall verdict: approved, approved_with_followups, changes_required, blocked."
    - "List blocking findings separately from non-blocking notes."

  dimensions:
    - "Specification conformance: does the implementation match the approved spec?"
    - "Implementation quality: is the code correct, idiomatic, and maintainable?"
    - "Test adequacy: are tests appropriate, sufficient, and well-structured?"
    - "Documentation currency: is documentation accurate and up to date?"
    - "Workflow integrity: is the workflow state consistent with the work?"
```

This profile is assembled into context packets for review sub-agents via `context_assemble(role="reviewer")`.

---

## 9. Review SKILL

A new SKILL `.skills/code-review.md` is created. The SKILL packages the review procedure for agents.

### 9.1 SKILL scope

The SKILL covers:
- How to perform a review of a single review unit
- The structured output format
- How to evaluate each dimension
- How to classify findings (blocking vs non-blocking)
- How to handle ambiguity and edge cases

### 9.2 SKILL audience

The SKILL is used by:
- **Review sub-agents** — follow the procedure to produce findings
- **Orchestrators** — reference the decomposition guidance
- **Humans** — understand what agents will check

### 9.3 SKILL does not cover

The SKILL does not cover orchestration. Orchestration is a higher-level concern handled by the orchestrator agent, which uses the SKILL as one of its inputs but also manages lifecycle transitions, task creation, and workflow coordination.

---

## 10. Orchestrator Pattern

### 10.1 No dedicated review tool

The orchestrator uses existing Kanbanzai tools to manage the review workflow. No new MCP tool is required for the initial implementation.

The orchestrator's tool usage:

| Step | Tools used |
|---|---|
| Find features to review | `list_entities_filtered(entity_type="feature", status="reviewing")` |
| Get spec structure | `doc_outline`, `doc_section` |
| Get task/file lists | `list_entities_filtered(entity_type="task", parent=...)` |
| Build sub-agent context | `context_assemble(role="reviewer")` |
| Dispatch sub-agents | `spawn_agent` (delegated review units) |
| Create remediation tasks | `create_task` |
| Transition feature state | `update_status` |
| Check conflict risk | `conflict_domain_check` |
| Record decisions | `record_decision` |

### 10.2 Why not a dedicated tool?

The review workflow is an orchestration pattern, not a primitive operation. Building it from existing tools:
- Allows iteration on the workflow without changing the tool surface
- Keeps the 2.0 tool surface lean
- Makes the pattern transparent and debuggable
- Avoids over-engineering before the pattern stabilises

A dedicated `review` tool may be justified later if the pattern proves stable and the overhead of manual orchestration is significant. This is an explicit deferral, not an oversight.

### 10.3 Future: review as a feature group tool

If a dedicated tool is later warranted, it would likely be a 2.0 feature group tool:
- `review start <feature-id>` — transition to reviewing, return decomposition plan
- `review findings <feature-id>` — return collated findings
- `review complete <feature-id>` — transition based on findings (done or needs-rework)

This is noted for future consideration, not proposed for this design.

---

## 11. Review at Different Scales

### 11.1 Single task

Not the focus of this design. Task-level review uses `finish(to_status="needs-review")` and the existing task lifecycle. The `ReviewService` in `service/review.go` may be re-exposed as a 2.0 tool in future.

### 11.2 Single feature

The primary use case. The orchestrator manages one feature through the review workflow. Typically 2–5 parallel review sub-agents, completing in minutes.

### 11.3 Multiple features (phase review)

The orchestrator iterates over features in `reviewing` state. Features are independent — they can be reviewed in parallel at the feature level, with sub-agent parallelism within each feature.

For a phase with 10 features, this might mean 10 × 3 = 30 parallel sub-agent review units. The orchestrator dispatches in waves, collates findings per feature, and manages remediation per feature.

### 11.4 Full codebase audit

A special case of phase review, but potentially without features as the organising unit. The orchestrator would decompose by package or module rather than by feature. This is feasible but is outside the scope of this design — it would need a separate decomposition strategy.

---

## 12. Integration with the Merge Workflow

Review and merge are related but distinct gates:

- **Review** asks: does the implementation satisfy the spec and quality standards?
- **Merge** asks: is the branch technically ready to integrate?

The Merge Readiness Review Profile (quality gates policy §5.3) is applied at merge time, after review has passed. A feature's path to main is:

    developing → reviewing → done → merge readiness check → merge

The existing merge gate infrastructure (`merge_readiness_check`, `merge_execute`) is not changed. Review is an upstream gate that must pass before merge readiness is evaluated.

---

## 13. Human Role

### 13.1 What humans do

- Trigger review (explicitly or by approving transition to `reviewing`)
- Override review verdicts (approve despite concerns, reject despite pass)
- Make judgment calls on ambiguous findings
- Approve the final `reviewing → done` transition for high-stakes features

### 13.2 What humans don't need to do

- Read every file that was reviewed
- Manually decompose the review into units
- Collate findings from multiple reviewers
- Create remediation tasks
- Track remediation progress

### 13.3 Human checkpoints

The orchestrator should use `human_checkpoint` when:
- Review produces findings that require human judgment (not clearly blocking or non-blocking)
- The feature is high-stakes and final approval should be explicit
- There is disagreement between review dimensions (e.g., spec conformance passes but implementation quality fails)

---

## 14. Deliverables

This design proposes the following concrete deliverables:

### 14.1 Lifecycle change (code)
- Add `FeatureStatusReviewing` and `FeatureStatusNeedsRework` to `model/entities.go`
- Update Phase 2 feature transition map in `validate/lifecycle.go`
- Remove the `developing → done` direct transition
- Update health checks if they validate feature lifecycle completeness
- Update tests

### 14.2 Reviewer context profile (configuration)
- Create `.kbz/context/roles/reviewer.yaml`
- Inherits from `base`
- Contains review conventions, dimensions, and output format guidance

### 14.3 Code review SKILL (document)
- Create `.skills/code-review.md`
- Procedural document for review sub-agents
- Covers: single-unit review procedure, structured output format, dimension evaluation, finding classification

### 14.4 Quality gates policy update (document)
- Update `quality-gates-and-review-policy.md` to reference this design
- Note that the policy is now operationalised through the reviewer profile and review SKILL
- Add cross-references

### 14.5 AGENTS.md update (document)
- Reference the review SKILL for review expectations
- Remove any inline review instructions that duplicate the SKILL
- Note: minimal change — the quality gates policy reference already exists

---

## 15. What This Design Does Not Cover

- **Task-level review automation** — re-exposing `ReviewService` as a 2.0 tool is a separate concern
- **Automated review triggering** — automatic transition to `reviewing` when all tasks complete is a future enhancement
- **Review document format** — the exact format of the review findings document is defined by the SKILL, not this design
- **Cross-feature review** — reviewing interactions between features is a future concern
- **Full codebase audit** — decomposition by module rather than by feature needs a separate strategy
- **Review metrics** — tracking review pass rates, common findings, etc. is deferred

---

## 16. Open Questions

### 16.1 Should the `developing → done` path be preserved for exceptional cases?

The design removes it. This means every feature, no matter how trivial, must pass through `reviewing`. The lightweight review profile exists for trivial features, but the lifecycle transition is still required.

Alternative: keep `developing → done` but add a `skip_review_reason` field. This is more flexible but weakens the guarantee.

**Recommendation:** remove the direct path. The ceremony of transitioning through `reviewing` is minimal, and the guarantee that every feature was reviewed is valuable. If this proves too rigid in practice, we can add the exception path later.

### 16.2 Where do review findings live?

Options:
- As a document in `work/` (human-readable, version-controlled)
- As structured data in `.kbz/state/` (machine-readable, queryable)
- Both (document for humans, structured data for machines)

**Recommendation:** start with a document in `work/`. This aligns with the document-centric interface model. Structured review records can be added later if the query need emerges.

### 16.3 Should the orchestrator pattern be documented as a SKILL?

The review SKILL covers sub-agent behavior. Should there be a separate orchestrator SKILL that covers the decomposition and coordination pattern?

**Recommendation:** yes, eventually. But let the pattern stabilise through use first. Document it as a SKILL once we've run it a few times and understand what works.

### 16.4 How does this interact with the PR workflow?

Currently, features can have associated PRs. Should review happen before or after PR creation?

**Recommendation:** review happens before PR creation. The PR is a merge artifact, not a review artifact. The review workflow examines code in the worktree; the PR is created when the feature is ready to merge (after review passes). This keeps review independent of GitHub.

---

## 17. Summary

This design adds a feature-level code review gate to Kanbanzai by:

1. **Adding lifecycle states** — `reviewing` and `needs-rework` to the feature lifecycle, removing the direct `developing → done` path.

2. **Defining a two-phase workflow** — read-only parallel analysis followed by write-phase remediation, orchestrated by an agent working at the metadata level.

3. **Operationalising the quality gates policy** — through a reviewer context profile and a code review SKILL that give sub-agents the criteria, procedure, and output format they need.

4. **Composing existing tools** — the orchestrator uses `list_entities_filtered`, `doc_outline`, `context_assemble`, `spawn_agent`, `create_task`, and `update_status` rather than requiring a new dedicated review tool.

The design ensures that review scales independently of codebase size (the orchestrator holds metadata, not code), that large reviews are parallelisable (analysis phase is read-only), and that the human role is judgment and approval rather than mechanical coordination.