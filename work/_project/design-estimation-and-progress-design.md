# Estimation and Progress Design

- Status: design
- Purpose: design for lightweight estimation and progress tracking across the entity hierarchy
- Date: 2026-07-24
- Phase: post-Phase 1 (flagged now to inform Phase 1 field design if needed)
- Basis:
  - `workflow-design-basis.md` §8 (object model, entity hierarchy)
  - `phase-1-specification.md` §8 (entity semantics), §9 (required fields)

---

## 1. Purpose

This document defines how estimation and progress tracking work in Kanbanzai. The goals are:

- give humans and agents a lightweight way to estimate effort for any entity
- use the entity hierarchy (Epic → Feature → Task) to compute rollup totals without storing them
- track progress as work completes
- keep the design simple enough that it does not create cascading file writes or merge conflicts in a Git-native system

---

## 2. Design Principles

### 2.1 Compute on read, never store rollups

Rolled-up totals (sum of children's estimates, sum of completed points) are computed when queried, not stored in parent entity files. This avoids cascading writes — editing a Task estimate does not touch its parent Feature or grandparent Epic file.

### 2.2 Preserve original estimates

When an entity is decomposed into children, the original top-down estimate is kept as a permanent record. The system surfaces the delta between the original estimate and the computed total from children, rather than replacing one with the other.

### 2.3 One scale, enforced by soft limits

All entities use the same Modified Fibonacci scale. Per-entity-type soft limits produce warnings (not errors) when an estimate seems too large for the entity type. This avoids the complexity of maintaining separate scale definitions while still guiding sensible estimation.

### 2.4 Bugs are estimation-independent

Bugs carry their own estimates for tracking purposes but do not roll up into the Feature/Task hierarchy. Bugs exist outside the parent-child hierarchy — they have lateral associations (`origin_feature`, `origin_task`), not parent relationships. Forcing them into a rollup model would create ambiguity about which entity receives their points.

---

## 3. The Estimation Scale

All entities use the Modified Fibonacci sequence for story point estimation.

### 3.1 Point values

| Points | Size    | Meaning |
|--------|---------|---------|
| 0      | —       | No effort required; no business value delivered |
| 0.5    | XXS     | Minimal effort; little business value |
| 1      | XS      | Simple, well-understood; likely completed in one day |
| 2      | S       | Requires some thought; developers have done this often |
| 3      | M       | Well-understood work with a few extra steps |
| 5      | L       | Complex work or infrequent work; may need collaboration |
| 8      | XL      | Requires research and likely multiple contributors |
| 13     | XXL     | Highly complex with many unknowns; too large for a single sprint |
| 20     | —       | Roughly one month of work |
| 40     | —       | Roughly two months of work |
| 100    | —       | Roughly five months of work |

The rough conversion is approximately 5 points per week.

### 3.2 Soft limits by entity type

| Entity | Recommended maximum | Rationale |
|--------|---------------------|-----------|
| Task   | 13                  | Tasks should be completable within roughly a week; above 13 suggests the task needs further decomposition |
| Bug    | 13                  | Bugs vary widely but most fixes should be task-sized or smaller |
| Feature| 100                 | Features are larger units of work but should not span many months |
| Epic   | 100                 | Epics are the largest planning unit |

Exceeding the soft limit produces a warning, not a validation error. Reality is sometimes messy, and an honest estimate of 20 on a Task is better than a dishonest 13.

---

## 4. Entity Fields

### 4.1 Task

- `estimate` — story points (Modified Fibonacci value). Optional.

### 4.2 Feature

- `estimate` — the original top-down estimate, set before decomposition into Tasks. Optional. Once set, not overwritten by rollup.

The following are computed on read, not stored:

- **task total** — sum of `estimate` across all child Tasks in active states.
- **delta** — difference between `estimate` and task total, when both are available.
- **progress** — sum of `estimate` across child Tasks in completed terminal states.

### 4.3 Epic

- `estimate` — the original top-down estimate, set before decomposition into Features. Optional. Once set, not overwritten by rollup.

The following are computed on read, not stored:

- **feature total** — sum of each Feature's effective estimate (task total if the Feature has Tasks, otherwise the Feature's own `estimate`).
- **delta** — difference between `estimate` and feature total, when both are available.
- **progress** — sum of completed points, rolled up from Features.

### 4.4 Bug

- `estimate` — story points (Modified Fibonacci value). Optional.

Bug estimates are self-contained. They do not contribute to any parent entity's totals.

---

## 5. Computed Rollup Rules

### 5.1 Feature effective estimate

A Feature's effective estimate for rollup purposes is:

1. If the Feature has child Tasks with estimates: sum of child Task estimates (in active and completed states; excluding `not-planned` and `duplicate`).
2. If the Feature has no child Tasks, or none have estimates: the Feature's own `estimate` field.
3. If neither exists: no estimate available.

### 5.2 Feature progress

Sum of `estimate` values from child Tasks in completed terminal states (e.g., `done`). Tasks in `not-planned` or `duplicate` states do not count as completed work.

### 5.3 Epic effective estimate

Sum of effective estimates across all child Features (using the Feature rule above).

### 5.4 Epic progress

Sum of progress values across all child Features.

### 5.5 Excluded states

The following terminal states are excluded from both totals and progress:

- `not-planned` — work was discarded, not completed.
- `duplicate` — work was never done.

This distinction matters: a Feature with five Tasks where two are `not-planned` should show a total based on the three remaining Tasks, not all five.

---

## 6. Delta Surfacing

When the system presents an entity with both an original estimate and a computed total from children, it should display both values and the delta between them.

Example output:

```
FEAT-xxx — original estimate: 8 · task total: 16 · delta: +8
EPIC-yyy — original estimate: 40 · feature total: 37 · delta: -3
```

A large positive delta suggests underestimation or scope growth. A large negative delta suggests overestimation or scope reduction. Either way, the information is valuable for planning and should not be hidden.

---

## 7. Bug Estimation

Bugs carry estimates but are tracked independently because:

1. Bugs have no formal parent — they use lateral associations (`origin_feature`, `origin_task`) that record where the bug was *found*, not where the fix *goes*.
2. A bug found in Feature A might be fixed as part of Feature B. Rolling up into the origin would misattribute effort.
3. Bug complexity varies enormously and bears no reliable relationship to the complexity of the associated entity.

Bug estimates are useful for:

- individual workload planning (how much effort is this fix?)
- aggregate reporting (how many points of bug work are in flight?)
- velocity analysis (what fraction of effort goes to bugs vs. features?)

They are not useful for hierarchical rollup, so they are excluded from it.

---

## 8. AI Agent Estimation Consistency

Story points are inherently relative — they work in human teams because the team calibrates together over time. AI agents lack this shared calibration. The following mechanisms address this:

### 8.1 Reference examples

Maintain a set of completed, estimated entities as calibration anchors. When estimating, agents compare new work against these references. Example:

> TASK-xxx was rated 3 and involved adding a single validation rule with tests.
> TASK-yyy was rated 8 and involved designing a new storage format with migration logic.

The reference set grows organically as work is completed and should be curated periodically to keep it representative.

### 8.2 Estimation as a dedicated operation

Rather than allowing agents to estimate as a side-effect of entity creation, estimation should be a specific operation with built-in context: the system presents the scale definitions and reference examples, and the agent provides its estimate. This standardises the information available at estimation time.

### 8.3 Human review

Treat estimates as provisional until a human confirms them, at least during early use. This builds the reference set and catches miscalibration before it compounds through rollups.

---

## 9. Phase 1 Implications

This design is targeted at post-Phase 1 implementation. However, the following Phase 1 considerations apply:

### 9.1 Field reservation

Phase 1 entity schemas should not use `estimate` as a field name for any other purpose. No action is required if the field is simply absent — adding optional fields to an existing schema is backwards-compatible.

### 9.2 Terminal state semantics

The rollup rules depend on distinguishing "completed" terminal states from "discarded" terminal states. Phase 1 lifecycle definitions should be clear about which terminal states represent completed work and which represent discarded work. This distinction is already implied by the current lifecycle designs (`done` vs. `not-planned`) but should be made explicit if it is not.

### 9.3 No structural changes needed

This design does not require changes to the entity hierarchy, file layout, or ID system. It adds optional fields and computed queries. Phase 1 does not need to prepare for it beyond the considerations above.

---

## 10. Open Questions

1. **Should the system warn on missing estimates?** If a Feature has five Tasks and only three have estimates, should the rollup note the gap or silently sum what's available?

2. **Should there be a `re-estimate` operation?** If an estimate turns out to be wrong mid-work, should the system track the history of estimate changes, or is the current value sufficient?

3. **How should reference examples be stored and presented?** As a curated document? As tagged entities? As part of the estimation operation's prompt context?

4. **Should progress percentages be shown?** Computing "8 of 21 points complete (38%)" is trivial but may imply false precision given the nature of story points.

---

## 11. Summary

- All entities use the Modified Fibonacci scale (0, 0.5, 1, 2, 3, 5, 8, 13, 20, 40, 100).
- Soft limits per entity type produce warnings, not errors.
- Original estimates are preserved; computed rollups are derived on read.
- The delta between original estimate and computed total is surfaced, not hidden.
- Bugs carry estimates independently and do not roll up into the hierarchy.
- Rollups exclude discarded work (`not-planned`, `duplicate`).
- AI consistency is addressed through reference examples, dedicated estimation operations, and human review.
- No Phase 1 structural changes are required.