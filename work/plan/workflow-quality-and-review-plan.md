# Workflow Quality & Code Review Plan

- Status: Phase 1 complete, Phase 2 active
- Date: 2026-03-27T23:07:03Z
- Plan ID: P6-workflow-quality-and-review
- Motivation: `work/reports/kanbanzai-2.0-workflow-retrospective.md`
- Related design: `work/design/code-review-workflow.md` (status: design proposal)

---

## 1. Purpose

This plan coordinates two related bodies of work:

1. **Workflow quality improvements** driven by the Kanbanzai 2.0 retrospective — fixing friction points that affect every agent on every feature.
2. **Code review workflow** — adding a feature-level review gate and orchestrated review process, as designed in `work/design/code-review-workflow.md`.

These are sequenced deliberately. The retrospective's top friction points interact directly with the review workflow's lifecycle changes. Fixing the foundations first means the review workflow lands on solid ground rather than amplifying existing problems.

---

## 2. Sequencing Rationale

The code review design (§4) adds two new feature lifecycle states (`reviewing`, `needs-rework`) and removes the `developing → done` escape hatch. This extends the mandatory transition chain from 6 to 7 states.

The retrospective's #1 finding (4/6 agents) is that the current 6-state chain already has too much ceremony when upstream documents exist. Implementing the review workflow first would make this worse before fixing it.

Additionally, the review orchestrator pattern (§10.1) depends on:
- `list_entities_filtered(parent=...)` working correctly (currently broken per retro R7)
- Parent-child state consistency being detectable (currently undetected per retro R3)
- Agents being able to navigate lifecycle transitions without trial-and-error (retro R1, R9)

Therefore: prerequisites first, then review workflow, then independent quality-of-life improvements.

---

## 3. Work Phases

### Phase 1: Lifecycle & Entity Foundations

**Goal:** Fix the lifecycle ceremony problem and entity query issues that block the review workflow.

**Scope:** Retro items R1, R3, R7, R9.

#### Feature A: Smart Lifecycle Transitions

Retro items: R1 (smart lifecycle transitions), R9 (valid transitions in error messages)

Allow features to skip lifecycle stages when prerequisite documents already exist. When a feature has an approved specification, transitioning from `proposed` should not require walking through `designing → specifying` manually.

Work includes:
- Design the skip-gates logic: define what "prerequisite satisfied" means for each stage gate (approved design doc? approved spec? approved dev-plan?)
- Implement stage-skipping in the feature lifecycle state machine
- Surface valid transitions in all lifecycle error messages (the data is in `allowedTransitions` — expose it)
- Ensure the skip logic composes correctly with the new review states added in Phase 2

**Design needed:** Yes — the skip-gates logic needs a short design note covering the rules for when each stage can be skipped. This is a focused design question, not a full design document.

**Risk:** The skip logic must compose correctly with the review states added in Phase 2. Design both together, implement in sequence.

#### Feature B: Entity State Consistency

Retro item: R3

Add health check rules that detect parent-child state inconsistencies and surface them as warnings.

Work includes:
- Health check: feature is `done` but has non-terminal children
- Health check: all children are terminal but feature is in an early lifecycle state
- Health check: worktree is `active` but branch is already merged
- Optional: surface consistency warnings on `get_entity` responses

**Design needed:** No — these are health check rules with clear semantics. The existing health check infrastructure supports this directly.

#### Feature C: Entity Query & Update Fixes

Retro items: R6, R7

Fix `list_entities_filtered` parent filter and allow `update_entity` to set `depends_on` on tasks.

Work includes:
- Diagnose and fix `list_entities_filtered(parent=...)` returning empty results
- Add `depends_on` support to `update_entity` for task entities
- Add test coverage for both

**Design needed:** No — these are bug fixes and tool-gap closures with clear expected behaviour.

#### Phase 1 Dependencies

```
Feature C (entity query fixes) — no dependencies, can start immediately
Feature B (state consistency)  — no dependencies, can start immediately
Feature A (smart transitions)  — design note needed before implementation
```

Features B and C can be worked in parallel. Feature A needs its design note reviewed before implementation but can be designed in parallel with B and C's implementation.

---

### Phase 2: Code Review Workflow

**Goal:** Add a feature-level review gate with orchestrated parallel review, as designed in `work/design/code-review-workflow.md`.

**Depends on:** Phase 1 complete (specifically: smart transitions work, parent filter fixed, state consistency detectable).

**Scope:** The full code review design deliverables (§14):

#### Feature D: Review Lifecycle States

Design ref: `code-review-workflow.md` §4

Add `reviewing` and `needs-rework` to the feature lifecycle. Remove the `developing → done` direct path. Implement the transition map from §4.2.

Work includes:
- Add states and transitions to the feature state machine
- Ensure smart-skip logic from Phase 1 composes correctly (e.g., skipping to `developing` should still require passing through `reviewing` to reach `done`)
- Update health checks to understand the new states
- Update merge gate checks if they reference feature status

#### Feature E: Reviewer Context Profile & SKILL

Design ref: `code-review-workflow.md` §8, §9

Create the reviewer context profile and the code review SKILL document.

Work includes:
- Create `reviewer` context profile in `.kbz/context/roles/`
- Write the code review SKILL document in `.skills/`
- Update quality gates policy with any refinements from the review design

#### Feature F: Review Orchestration Pattern

Design ref: `code-review-workflow.md` §5, §6, §10

Implement the review orchestration pattern: analysis phase (parallel sub-agent review), findings collation, remediation task creation, and re-review cycle.

This is primarily a SKILL/procedure — the review design explicitly defers a dedicated tool (§10.2). The work is:
- Document the orchestrator procedure in the SKILL
- Verify the tool chain works end-to-end: `list_entities_filtered` → `doc_outline` → `context_assemble` → `spawn_agent` → `create_task` → `update_status`
- Test at single-feature and multi-feature scale (§11)

#### Feature G: Policy & Documentation Updates

Design ref: `code-review-workflow.md` §14.4, §14.5

Update AGENTS.md, quality gates policy, and any other documentation to reflect the new review workflow.

#### Phase 2 Dependencies

```
Feature D (lifecycle states) — depends on Phase 1 Feature A (smart transitions)
Feature E (profile & SKILL)  — no Phase 1 dependency, can start when design is approved
Feature F (orchestration)    — depends on D (states exist) and E (profile exists)
Feature G (documentation)    — depends on D, E, F (document what was built)
```

Feature E can start as soon as the code review design is approved, potentially in parallel with late Phase 1 work.

---

### Phase 3: Quality of Life (Independent)

**Goal:** Address remaining retrospective items that don't interact with the review workflow.

**Depends on:** Nothing — these can be worked in parallel with Phase 1 or Phase 2, or after.

#### Feature H: Human-Friendly ID Display

Retro item: R4

Implement the planned ID split format for display: `FEAT-01KMR-J81DZ3X2` instead of `FEAT-01KMRJ81DZ3X2`. Storage remains unsplit. Accept both formats as input.

#### Feature I: Bulk Task Operations

Retro item: R8

Add a batch variant of `finish` that accepts multiple task IDs and transitions them through required states in dependency order.

#### Feature J: Plan-to-Feature Guidance

Retro item: R5

Add workflow guidance for the plan → features transition. This could be a tool, a prompt convention, or a SKILL — the right form needs a brief design conversation.

#### Feature K: Minor Improvements

Retro items: R10 (make `design` field functional), R11 (standalone retro capture), R12 (auto-validate tool reference docs), R13 (knowledge graph staleness warning)

These are small, independent items that can be picked up opportunistically.

---

## 4. Work Summary

| Phase | Features | Depends on | Estimated Size |
|-------|----------|------------|----------------|
| **1: Foundations** | A (smart transitions), B (state consistency), C (query fixes) | — | Medium |
| **2: Code Review** | D (lifecycle), E (profile/SKILL), F (orchestration), G (docs) | Phase 1 | Large |
| **3: Quality of Life** | H (ID display), I (bulk ops), J (plan guidance), K (minor) | — | Small–Medium |

### Critical Path

```
Phase 1 Feature A design note
        ↓
Phase 1 Feature A implementation ──→ Phase 2 Feature D (review states)
                                              ↓
Phase 1 Feature C (query fixes) ────→ Phase 2 Feature F (orchestration)
                                              ↓
                                     Phase 2 Feature G (documentation)
```

Phase 1 Features B and C, Phase 2 Feature E, and all Phase 3 features are off the critical path and can be parallelised.

---

## 5. Design Document Status

| Document | Status | Action Needed |
|----------|--------|---------------|
| `work/design/code-review-workflow.md` | Design proposal | Needs human review and approval before Phase 2 features are created |
| Phase 1 Feature A design note | Not yet written | Needs to be written and approved before Feature A implementation |
| Phase 1 Features B, C | N/A | Bug fixes / tool gaps — no design needed |
| Phase 3 Feature J | TBD | May need a brief design conversation |

---

## 6. What This Plan Does Not Cover

- **Structured review records in `.kbz/state/`** — explicitly deferred by the code review design (§16.2). Can be added later if query needs emerge.
- **Orchestrator SKILL** — explicitly deferred by the code review design (§16.3) until the review pattern stabilises.
- **Dedicated review tool** — explicitly deferred by the code review design (§10.2). May be warranted after the orchestration pattern proves stable.
- **Retro items D1–D3** from the retrospective (clustering quality, test fixture fragility, tool count assertions) — deferred/monitor status, no action planned.

---

## 7. Next Steps

1. **Human reviews this plan** — confirm sequencing, scope, and phase boundaries.
2. **Human reviews `work/design/code-review-workflow.md`** — the design needs approval before Phase 2 features can be created.
3. **Write Phase 1 Feature A design note** — the skip-gates logic for smart lifecycle transitions.
4. **Create Plan entity and Feature entities** for Phase 1 once plan is approved.
5. **Phase 1 Feature A design note is reviewed** — then implementation begins.
6. **Phase 1 Features B and C can start immediately** after plan approval (no design needed).